package system

import (
	"crypto/ed25519"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kinecosystem/agora-common/solana"
)

func TestCreateAccount(t *testing.T) {
	keys := generateKeys(t, 3)

	instruction := CreateAccount(keys[0], keys[1], keys[2], 12345, 67890)

	command := make([]byte, 4)
	lamports := make([]byte, 8)
	binary.LittleEndian.PutUint64(lamports, 12345)
	size := make([]byte, 8)
	binary.LittleEndian.PutUint64(size, 67890)

	assert.Equal(t, command, instruction.Data[0:4])
	assert.Equal(t, lamports, instruction.Data[4:12])
	assert.Equal(t, size, instruction.Data[12:20])
	assert.Equal(t, []byte(keys[2]), instruction.Data[20:52])

	var tx solana.Transaction
	require.NoError(t, tx.Unmarshal(solana.NewTransaction(keys[0], instruction).Marshal()))

	decompiled, err := DecompileCreateAccount(tx.Message, 0)
	require.NoError(t, err)
	assert.Equal(t, decompiled.Funder, keys[0])
	assert.Equal(t, decompiled.Address, keys[1])
	assert.Equal(t, decompiled.Owner, keys[2])
	assert.EqualValues(t, decompiled.Lamports, 12345)
	assert.EqualValues(t, decompiled.Size, 67890)
}

func TestDecompileNonCreate(t *testing.T) {
	keys := generateKeys(t, 4)

	instruction := CreateAccount(keys[0], keys[1], keys[2], 12345, 67890)

	binary.BigEndian.PutUint32(instruction.Data, commandAllocate)
	_, err := DecompileCreateAccount(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.Equal(t, solana.ErrIncorrectInstruction, err)

	instruction.Data = make([]byte, 3)
	_, err = DecompileCreateAccount(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.NotNil(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "invalid instruction data size"))

	instruction.Accounts = instruction.Accounts[:1]
	_, err = DecompileCreateAccount(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.NotNil(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "invalid number of accounts"))

	instruction.Program = keys[3]
	_, err = DecompileCreateAccount(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.Equal(t, solana.ErrIncorrectProgram, err)

	_, err = DecompileCreateAccount(solana.NewTransaction(keys[0], instruction).Message, 1)
	assert.NotNil(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "instruction doesn't exist"))
}

func generateKeys(t *testing.T, amount int) []ed25519.PublicKey {
	keys := make([]ed25519.PublicKey, amount)

	for i := 0; i < amount; i++ {
		pub, _, err := ed25519.GenerateKey(nil)
		require.NoError(t, err)
		keys[i] = pub
	}

	return keys
}