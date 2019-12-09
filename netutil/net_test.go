package netutil

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetAvailablePortForAddress(t *testing.T) {
	port, err := GetAvailablePortForAddress("localhost")
	assert.NoError(t, err)

	// We expect the port to be non-privileged, which generally means
	// 1024 and above.
	assert.True(t, port >= 1024)
}
