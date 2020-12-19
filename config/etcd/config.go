package etcd

import (
	"context"
	"sync"
	"time"

	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/clientv3"

	"github.com/kinecosystem/agora-common/config"
)

type conf struct {
	log *logrus.Entry

	client *clientv3.Client
	key    string

	mux        sync.RWMutex
	kvs        []*mvccpb.KeyValue
	err        error
	cancelFunc context.CancelFunc
	shutdown   bool
}

func NewConfig(ctx context.Context, c *clientv3.Client, key string, refreshTime time.Duration) config.Config {
	ctx, cancelFunc := context.WithCancel(ctx)
	client := &conf{
		log:        logrus.StandardLogger().WithField("type", "config/etcd"),
		client:     c,
		key:        key,
		cancelFunc: cancelFunc,
	}
	client.watch(ctx, refreshTime)
	return client
}

func (c *conf) Get(ctx context.Context) (interface{}, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	if c.shutdown {
		c.log.Warn("attempted use of config after shutdown")
		return nil, config.ErrShutdown
	}

	if c.kvs == nil {
		if c.err != nil {
			return nil, errors.Wrap(c.err, "failed to fetch config")
		}
		return nil, config.ErrNoValue
	}
	if len(c.kvs) == 0 {
		return nil, config.ErrNoValue
	}

	return c.kvs[0].Value, nil
}

// Shutdown implements Config.Shutdown
func (c *conf) Shutdown() {
	c.mux.Lock()
	if !c.shutdown {
		c.log.Info("shutting down")
		c.cancelFunc()
		c.shutdown = true
	}
	c.mux.Unlock()
}

func (c *conf) watch(ctx context.Context, refreshTime time.Duration) {
	// TODO: use watcher instead of this
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			resp, err := c.client.Get(ctx, c.key)
			if err != nil {
				c.log.WithError(err).Warn("failed to get config")
				time.Sleep(refreshTime)
				continue
			}

			c.mux.Lock()
			c.kvs = resp.Kvs
			c.err = err
			c.mux.Unlock()

			time.Sleep(refreshTime)
		}
	}()
}
