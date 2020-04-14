package metrics

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateClient(t *testing.T) {
	config := &ClientConfig{}
	client, err := CreateClient("test", config)
	require.Error(t, err)
	require.Nil(t, client)

	RegisterClientCtor("test", newClient)

	client, err = CreateClient("test", config)
	require.NoError(t, err)
	require.NotNil(t, client)
}

type testClient struct{}

func newClient(config *ClientConfig) (Client, error) {
	return &testClient{}, nil
}

func (t testClient) Count(name string, value int64, tags []string) error {
	return nil
}

func (t testClient) Close() error {
	return nil
}
