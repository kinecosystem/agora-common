package test

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/ory/dockertest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	containerName    = "redis"
	containerVersion = "5"
)

var (
	log = logrus.StandardLogger().WithField("type", "redis/test")
)

func StartRedis(ctx context.Context, pool *dockertest.Pool) (connString string, closeFunc func(), err error) {
	resource, err := pool.Run(containerName, containerVersion, nil)
	if err != nil {
		return "", func() {}, errors.Wrap(err, "failed to start resource")
	}

	closeFunc = func() {
		if err := pool.Purge(resource); err != nil {
			log.WithError(err).Warn("Failed to clean up Redis resource")
		}
	}

	port := resource.GetPort("6379/tcp")
	connString = fmt.Sprintf("localhost:%s", port)

	for {
		select {
		case <-time.After(1 * time.Second):
			if isHealthy(connString) {
				return connString, closeFunc, nil
			}
			closeFunc()
			log.Trace("Redis health check failed")
		case <-ctx.Done():
			closeFunc()
			return "", func() {}, errors.New("redis didn't come up in time")
		}
	}
}

func isHealthy(connString string) bool {
	client := redis.NewClient(&redis.Options{
		Addr: connString,
	})
	defer client.Close()

	return client.Ping().Err() == nil
}
