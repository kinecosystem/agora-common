package test

import (
	"context"
	"testing"
	"time"

	"github.com/ory/dockertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEtcd(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := dockertest.NewPool("")
	require.NoError(t, err)

	client, cleanupFunc, err := StartEtcd(ctx, pool)
	require.NoError(t, err)
	defer cleanupFunc()

	_, err = client.Put(context.Background(), "foo", "bar")
	require.NoError(t, err)

	resp, err := client.Get(context.Background(), "foo")
	require.NoError(t, err)

	assert.Len(t, resp.Kvs, 1)
	assert.Equal(t, "foo", string(resp.Kvs[0].Key))
	assert.Equal(t, "bar", string(resp.Kvs[0].Value))
}
