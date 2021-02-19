package token

import (
	"testing"

	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAssociatedAccount(t *testing.T) {
	// Values generated from taken from spl code.
	wallet, err := base58.Decode("4uQeVj5tqViQh7yWWGStvkEG1Zmhx6uasJtWCJziofM")
	require.NoError(t, err)
	mint, err := base58.Decode("8opHzTAnfzRpPEx21XtnrVTX28YQuCpAjcn1PczScKh")
	require.NoError(t, err)
	addr, err := base58.Decode("H7MQwEzt97tUJryocn3qaEoy2ymWstwyEk1i9Yv3EmuZ")
	require.NoError(t, err)

	actual, err := GetAssociatedAccount(wallet, mint)
	require.NoError(t, err)
	assert.EqualValues(t, addr, actual)
}
