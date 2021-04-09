package etcd

import (
	"context"
	"strconv"
	"time"

	"go.etcd.io/etcd/client/v3"
)

// SetStringConfig sets a string config value in etcd
func SetStringConfig(ctx context.Context, client *clientv3.Client, name string, value string) error {
	_, err := client.Put(ctx, name, value)
	return err
}

// SetBytesConfig sets a byte array config value in etcd
func SetBytesConfig(ctx context.Context, client *clientv3.Client, name string, value []byte) error {
	return SetStringConfig(ctx, client, name, string(value))
}

// SetInt64Config sets an int64 config value in etcd
func SetInt64Config(ctx context.Context, client *clientv3.Client, name string, value int64) error {
	return SetStringConfig(ctx, client, name, strconv.FormatInt(value, 10))
}

// SetUint64Config sets a uint64 config value in etcd
func SetUint64Config(ctx context.Context, client *clientv3.Client, name string, value uint64) error {
	return SetStringConfig(ctx, client, name, strconv.FormatUint(value, 10))
}

// SetFloat64Config sets a float64 config value in etcd
func SetFloat64Config(ctx context.Context, client *clientv3.Client, name string, value float64) error {
	return SetStringConfig(ctx, client, name, strconv.FormatFloat(value, 'f', -1, 64))
}

// SetDurationConfig sets a duration config value in etcd
func SetDurationConfig(ctx context.Context, client *clientv3.Client, name string, value time.Duration) error {
	return SetStringConfig(ctx, client, name, value.String())
}
