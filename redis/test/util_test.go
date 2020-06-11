package test

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/ory/dockertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartRedis(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := dockertest.NewPool("")
	require.NoError(t, err)

	connString, cleanupFunc, err := StartRedis(ctx, pool)
	require.NoError(t, err)
	defer cleanupFunc()

	client := redis.NewClient(&redis.Options{
		Addr: connString,
	})

	require.NoError(t, client.Set("k", "v", 0).Err())

	resp := client.Get("k")
	require.NoError(t, resp.Err())
	assert.Equal(t, "v", resp.Val())
}
