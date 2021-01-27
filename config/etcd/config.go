package etcd

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.etcd.io/etcd/clientv3"

	"github.com/kinecosystem/agora-common/config"
	"github.com/kinecosystem/agora-common/config/wrapper"
	"github.com/kinecosystem/agora-common/retry"
	"github.com/kinecosystem/agora-common/retry/backoff"
)

type conf struct {
	client *clientv3.Client
	key    string

	cancelWatch  func()
	shutdownCh   chan struct{}
	shutdownOnce sync.Once

	mu    sync.RWMutex
	val   []byte
	empty bool
}

func NewConfig(c *clientv3.Client, key string) config.Config {
	ctx, cancel := context.WithCancel(context.Background())
	client := &conf{
		client:      c,
		key:         key,
		empty:       true,
		cancelWatch: cancel,
		shutdownCh:  make(chan struct{}),
	}

	go client.watch(ctx)

	return client
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
				c.empty = false
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

// NewBytesConfig creates a etcd-backed byte array config
func NewBytesConfig(c *clientv3.Client, key string, defaultValue []byte) config.Bytes {
	return wrapper.NewBytesConfig(NewConfig(c, key), defaultValue)
}

// NewInt64Config creates a etcd-backed int64 config
func NewInt64Config(c *clientv3.Client, key string, defaultValue int64) config.Int64 {
	return wrapper.NewInt64Config(NewConfig(c, key), defaultValue)
}

// NewUint64Config creates a etcd-backed uint64 config
func NewUint64Config(c *clientv3.Client, key string, defaultValue uint64) config.Uint64 {
	return wrapper.NewUint64Config(NewConfig(c, key), defaultValue)
}

// NewFloat64Config creates a etcd-backed float64 config
func NewFloat64Config(c *clientv3.Client, key string, defaultValue float64) config.Float64 {
	return wrapper.NewFloat64Config(NewConfig(c, key), defaultValue)
}

// NewDurationConfig creates a etcd-backed duration config
func NewDurationConfig(c *clientv3.Client, key string, defaultValue time.Duration) config.Duration {
	return wrapper.NewDurationConfig(NewConfig(c, key), defaultValue)
}

// NewStringConfig creates a etcd-backed string config
func NewStringConfig(c *clientv3.Client, key string, defaultValue string) config.String {
	return wrapper.NewStringConfig(NewConfig(c, key), defaultValue)
}

// NewBoolConfig creates a etcd-backed bool config
func NewBoolConfig(c *clientv3.Client, key string, defaultValue bool) config.Bool {
	return wrapper.NewBoolConfig(NewConfig(c, key), defaultValue)
}
