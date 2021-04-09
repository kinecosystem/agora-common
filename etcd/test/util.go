package test

import (
	"context"
	"time"

	"github.com/ory/dockertest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/client/v3"
)

const (
	containerName    = "quay.io/coreos/etcd"
	containerVersion = "v3.3.12"
)

var (
	log = logrus.StandardLogger().WithField("type", "etcd/test")
)

// StartEtcd starts a dockerized etcd node for testing.
func StartEtcd(ctx context.Context, pool *dockertest.Pool) (client *clientv3.Client, closeFunc func(), err error) {
	closeFunc = func() {}

	log.Debug("Starting etcd container")

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: containerName,
		Tag:        containerVersion,
		Cmd: []string{
			"etcd",
			"-listen-client-urls",
			"http://0.0.0.0:2379",
			"-advertise-client-urls",
			"http://0.0.0.0:2379",
		},
	})
	if err != nil {
		return nil, closeFunc, errors.Wrapf(err, "failed to create etcd resource")
	}

	closeFunc = func() {
		if err := pool.Purge(resource); err != nil {
			log.WithError(err).Warn("Failed to cleanup etcd resource")
		}
	}

	client, err = clientv3.New(clientv3.Config{
		Endpoints: []string{"localhost:%s", resource.GetHostPort("2379/tcp")},
	})
	if err != nil {
		closeFunc()
		return nil, func() {}, errors.Wrapf(err, "failed to create client")
	}

	for !isHealthy(client) {
		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			closeFunc()
			return nil, func() {}, errors.New("etcd didn't come up in time")
		}
	}

	return client, closeFunc, nil
}

func isHealthy(client *clientv3.Client) bool {
	_, err := client.Get(context.Background(), "/")
	return err == nil
}
