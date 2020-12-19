package etcd

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/clientv3"

	"github.com/kinecosystem/agora-common/config"
	"github.com/kinecosystem/agora-common/retry"
	"github.com/kinecosystem/agora-common/retry/backoff"
)

type conf struct {
	log    *logrus.Entry
	client *clientv3.Client
	key    string

	cancelWatch  func()
	shutdownCh   chan struct{}
	shutdownOnce sync.Once

	mu    sync.RWMutex
	val   []byte
	empty bool
}

func NewConfig(c *clientv3.Client, key string) (config.Config, error) {
	ctx, cancel := context.WithCancel(context.Background())
	client := &conf{
		log:         logrus.StandardLogger().WithField("type", "config/etcd"),
		client:      c,
		key:         key,
		empty:       true,
		cancelWatch: cancel,
		shutdownCh:  make(chan struct{}),
	}

	kvs, err := c.Get(ctx, key)
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "failed to get initial value")
	}

	if len(kvs.Kvs) > 0 {
		client.val = kvs.Kvs[0].Value
		client.empty = false
	}

	go client.watch(ctx)

	return client, nil
}

func (c *conf) Get(ctx context.Context) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	select {
	case <-c.shutdownCh:
		return nil, config.ErrShutdown
	default:
	}

	if c.empty {
		return nil, config.ErrNoValue
	}

	return c.val, nil
}

// Shutdown implements Config.Shutdown
func (c *conf) Shutdown() {
	c.shutdownOnce.Do(func() {
		close(c.shutdownCh)
		c.cancelWatch()
	})
}

func (c *conf) watch(ctx context.Context) {
	_, _ = retry.Retry(
		func() error {
			watch := c.client.Watch(
				ctx,
				c.key,
				clientv3.WithCreatedNotify(),
				clientv3.WithFilterDelete(),
				clientv3.WithProgressNotify(),
			)

			get, err := c.client.Get(ctx, c.key)
			if err != nil {
				return err
			}
			if len(get.Kvs) > 0 {
				c.mu.Lock()
				c.val = get.Kvs[0].Value
				c.mu.Unlock()
			}

			var resp clientv3.WatchResponse
			for resp = range watch {
				if resp.Canceled {
					break
				}

				var newValue bool
				var lastValue []byte
				for _, e := range resp.Events {
					if e.Type != clientv3.EventTypePut {
						continue
					}

					lastValue = e.Kv.Value
					newValue = true
				}

				if newValue {
					c.mu.Lock()
					c.val = lastValue
					c.mu.Unlock()
				}

				// note: we should be checking for v3rpc.ErrCompacted, but
				//       the current etcd client release is all kind of messed up.
				//       the error is returned when resp.CompactRevision != 0
				if resp.Err() != nil && resp.CompactRevision == 0 {
					return resp.Err()
				}
			}

			return errors.New("watch ended, refreshing")
		},
		retry.NonRetriableErrors(context.Canceled),
		retry.BackoffWithJitter(backoff.BinaryExponential(500*time.Millisecond), 5*time.Second, 0.1),
	)
}
