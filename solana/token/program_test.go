package token

import (
	"crypto/ed25519"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/kinecosystem/agora-common/solana"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeAccount(t *testing.T) {
	keys := generateKeys(t, 4)

	instruction := InitializeAccount(keys[0], keys[1], keys[2])

	assert.Equal(t, []byte{1}, instruction.Data)
	assert.True(t, instruction.Accounts[0].IsSigner)
	assert.True(t, instruction.Accounts[0].IsWritable)
	for i := 1; i < 4; i++ {
		assert.False(t, instruction.Accounts[i].IsSigner)
		assert.False(t, instruction.Accounts[i].IsWritable)
	}

	decompiled, err := DecompileInitializeAccount(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.NoError(t, err)
	assert.Equal(t, keys[0], decompiled.Account)
	assert.Equal(t, keys[1], decompiled.Mint)
	assert.Equal(t, keys[2], decompiled.Owner)

	instruction.Accounts[3].PublicKey = keys[3]
	_, err = DecompileInitializeAccount(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid rent program"))

	instruction.Accounts = instruction.Accounts[:2]
	_, err = DecompileInitializeAccount(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid number of accounts"))

	instruction.Data[0] = byte(commandTransfer)
	_, err = DecompileInitializeAccount(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.Equal(t, solana.ErrIncorrectInstruction, err)

	instruction.Program = keys[3]
	_, err = DecompileInitializeAccount(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.Equal(t, solana.ErrIncorrectProgram, err)
}

func TestTestAuthority(t *testing.T) {
	keys := generateKeys(t, 3)

	instruction := SetAuthority(keys[0], keys[1], keys[2], AuthorityCloseAccount)

	assert.EqualValues(t, 6, instruction.Data[0])
	assert.EqualValues(t, AuthorityCloseAccount, instruction.Data[1])

	assert.False(t, instruction.Accounts[0].IsSigner)
	assert.True(t, instruction.Accounts[0].IsWritable)

	assert.True(t, instruction.Accounts[1].IsSigner)
	assert.False(t, instruction.Accounts[1].IsWritable)

	decompiled, err := DecompileSetAuthority(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.NoError(t, err)
	assert.Equal(t, keys[0], decompiled.Account)
	assert.Equal(t, keys[1], decompiled.CurrentAuthority)
	assert.Equal(t, keys[2], decompiled.NewAuthority)
	assert.Equal(t, AuthorityCloseAccount, decompiled.Type)

	// Mess with the instruction for validation
	instruction.Data = instruction.Data[:len(instruction.Data)-2]
	_, err = DecompileSetAuthority(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid data size"))

	instruction.Data[0] = byte(commandApprove)
	_, err = DecompileSetAuthority(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.Equal(t, solana.ErrIncorrectInstruction, err)

	instruction.Data = instruction.Data[:2]
	_, err = DecompileSetAuthority(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid data size"))

	instruction.Accounts = instruction.Accounts[:1]
	_, err = DecompileSetAuthority(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid number of accounts"))

	instruction.Program = keys[0]
	_, err = DecompileSetAuthority(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.Equal(t, solana.ErrIncorrectProgram, err)
}

func TestTestAuthority_NoNewAuthority(t *testing.T) {
	keys := generateKeys(t, 3)

	instruction := SetAuthority(keys[0], keys[1], nil, AuthorityCloseAccount)

	assert.EqualValues(t, []byte{6, byte(AuthorityCloseAccount), 0}, instruction.Data)

	assert.False(t, instruction.Accounts[0].IsSigner)
	assert.True(t, instruction.Accounts[0].IsWritable)

	assert.True(t, instruction.Accounts[1].IsSigner)
	assert.False(t, instruction.Accounts[1].IsWritable)

	decompiled, err := DecompileSetAuthority(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.NoError(t, err)
	assert.Equal(t, keys[0], decompiled.Account)
	assert.Equal(t, keys[1], decompiled.CurrentAuthority)
	assert.Nil(t, decompiled.NewAuthority)
	assert.Equal(t, AuthorityCloseAccount, decompiled.Type)

	// Mess with the instruction for validation
	instruction.Data = append(instruction.Data, 0)
	_, err = DecompileSetAuthority(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid data size"))

	instruction.Data[0] = byte(commandApprove)
	_, err = DecompileSetAuthority(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.Equal(t, solana.ErrIncorrectInstruction, err)

	instruction.Data = instruction.Data[:2]
	_, err = DecompileSetAuthority(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid data size"))

	instruction.Accounts = instruction.Accounts[:1]
	_, err = DecompileSetAuthority(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid number of accounts"))

	instruction.Program = keys[0]
	_, err = DecompileSetAuthority(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.Equal(t, solana.ErrIncorrectProgram, err)
}

func TestTransfer(t *testing.T) {
	keys := generateKeys(t, 4)

	instruction := Transfer(keys[0], keys[1], keys[2], 123456789)

	expectedAmount := make([]byte, 8)
	binary.LittleEndian.PutUint64(expectedAmount, 123456789)

	assert.EqualValues(t, 3, instruction.Data[0])
	assert.EqualValues(t, expectedAmount, instruction.Data[1:])

	assert.False(t, instruction.Accounts[0].IsSigner)
	assert.True(t, instruction.Accounts[0].IsWritable)
	assert.False(t, instruction.Accounts[1].IsSigner)
	assert.True(t, instruction.Accounts[1].IsWritable)

	assert.True(t, instruction.Accounts[2].IsSigner)
	assert.True(t, instruction.Accounts[2].IsWritable)

	decompiled, err := DecompileTransferAccount(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.NoError(t, err)
	assert.EqualValues(t, 123456789, decompiled.Amount)
	assert.Equal(t, keys[0], decompiled.Source)
	assert.Equal(t, keys[1], decompiled.Destination)
	assert.Equal(t, keys[2], decompiled.Owner)

	instruction.Data = instruction.Data[:1]
	_, err = DecompileTransferAccount(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid instruction data size"))

	instruction.Accounts = instruction.Accounts[:2]
	_, err = DecompileTransferAccount(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid number of accounts"))

	instruction.Data[0] = byte(commandApprove)
	_, err = DecompileTransferAccount(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.Equal(t, solana.ErrIncorrectInstruction, err)

	instruction.Program = keys[3]
	_, err = DecompileTransferAccount(solana.NewTransaction(keys[0], instruction).Message, 0)
	assert.Equal(t, solana.ErrIncorrectProgram, err)
}

func TestCloseAccount(t *testing.T) {
	keys := generateKeys(t, 3)

	instruction := CloseAccount(keys[0], keys[1], keys[2])
	assert.Equal(t, []byte{9}, instruction.Data)

	assert.False(t, instruction.Accounts[0].IsSigner)
	assert.True(t, instruction.Accounts[0].IsWritable)
	assert.False(t, instruction.Accounts[1].IsSigner)
	assert.True(t, instruction.Accounts[1].IsWritable)

	assert.True(t, instruction.Accounts[2].IsSigner)
	assert.False(t, instruction.Accounts[2].IsWritable)
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
