package etcd

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/ory/dockertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kinecosystem/agora-common/config"
	"github.com/kinecosystem/agora-common/etcd/test"
)

var (
	pool *dockertest.Pool
)

func TestMain(m *testing.M) {
	var err error
	pool, err = dockertest.NewPool("")
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func TestConfigDoesntExist(t *testing.T) {
	name := "doesnt_exist"

	client, closeFunc, err := test.StartEtcd(context.Background(), pool)
	require.NoError(t, err)
	defer closeFunc()

	refreshTime := 500 * time.Millisecond
	c := NewConfig(context.Background(), client, name, refreshTime)
	defer c.Shutdown()

	// No data is set in etcd, so a no value error should be returned
	_, err = c.Get(context.Background())
	assert.Equal(t, config.ErrNoValue, err)
}

func TestConfigOverridden(t *testing.T) {
	name := "config"
	initialValue := []byte{byte(1)}
	nextValue := []byte{byte(2)}

	client, closeFunc, err := test.StartEtcd(context.Background(), pool)
	require.NoError(t, err)
	defer closeFunc()

	require.NoError(t, SetBytesConfig(context.Background(), client, name, initialValue))

	refreshTime := 500 * time.Millisecond
	c := NewConfig(context.Background(), client, name, refreshTime)
	require.NoError(t, err)

	time.Sleep(refreshTime)

	value, err := c.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, initialValue, value)

	require.NoError(t, SetBytesConfig(context.Background(), client, name, nextValue))

	time.Sleep(refreshTime)

	// Updates to the config value should be observed and returned
	value, err = c.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, nextValue, value)

	// Try fetching after shutdown
	c.Shutdown()
	value, err = c.Get(context.Background())
	assert.Equal(t, config.ErrShutdown, err)
	assert.Nil(t, value)
}
