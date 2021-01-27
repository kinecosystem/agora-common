package env

import (
	"context"
	"os"
	"testing"

	"github.com/kinecosystem/agora-common/config"
	"github.com/stretchr/testify/assert"
)

func TestConfigDoesntExist(t *testing.T) {
	const env = "AGORA_ENV_CONFIG_TEST_VAR"
	os.Setenv(env, "default")

	v, err := NewConfig(env).Get(context.Background())
	assert.Equal(t, []byte("default"), v)
	assert.Nil(t, err)

	os.Unsetenv(env)

	v, err = NewConfig(env).Get(context.Background())
	assert.Nil(t, v)
	assert.Equal(t, config.ErrNoValue, err)
}
