package kin

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonpb "github.com/kinecosystem/agora-api/genproto/common/v3"

	"github.com/kinecosystem/agora-common/solana"
	"github.com/kinecosystem/agora-common/solana/memo"
	"github.com/kinecosystem/agora-common/solana/system"
	"github.com/kinecosystem/agora-common/solana/token"
)

func TestParseTransaction_NoInvoices(t *testing.T) {
	keys := generateKeys(t, 5)

	//
	// Regular transfer
	//
	input := solana.NewTransaction(
		keys[0],
		token.Transfer(
			keys[1],
			keys[2],
			keys[3],
			10,
		),
		token.Transfer(
			keys[2],
			keys[3],
			keys[4],
			20,
		),
	)
	tx, err := ParseTransaction(input, nil)
	assert.NoError(t, err)
	assert.Equal(t, "", tx.AppID)
	assert.EqualValues(t, 0, tx.AppIndex)
	assert.Len(t, tx.Regions, 1)
	assert.Empty(t, tx.Regions[0].Creations)
	assert.Len(t, tx.Regions[0].Transfers, 2)
	assert.Empty(t, tx.Regions[0].Closures)

	for i := 0; i < 2; i++ {
		assert.EqualValues(t, tx.Regions[0].Transfers[i].Source, keys[1+i])
		assert.EqualValues(t, tx.Regions[0].Transfers[i].Destination, keys[2+i])
		assert.EqualValues(t, tx.Regions[0].Transfers[i].Owner, keys[3+i])
		assert.EqualValues(t, tx.Regions[0].Transfers[i].Amount, (1+i)*10)
	}

	//
	// Sender Create + Closure (with + without multi-region)
	//
	createInstructions := generateCreate(t, keys[0], keys[1], keys[2])

	inputs := make([]solana.Transaction, 0)
	for i := 0; i < 2; i++ {
		instructions := append(createInstructions, token.Transfer(keys[3], keys[4], keys[1], 10))

		if i == 1 {
			m, err := NewMemo(1, TransactionTypeP2P, 0, make([]byte, 29))
			require.NoError(t, err)
			instructions = append(instructions, memo.Instruction(base64.StdEncoding.EncodeToString(m[:])))
		}

		instructions = append(instructions, token.CloseAccount(
			keys[3],
			keys[0],
			keys[1],
		))

		inputs = append(inputs, solana.NewTransaction(keys[0], instructions...))
	}

	for i := range inputs {
		tx, err := ParseTransaction(inputs[i], nil)
		assert.NoError(t, err)
		assert.Equal(t, "", tx.AppID)
		assert.EqualValues(t, 0, tx.AppIndex)

		assert.Len(t, tx.Regions, 1+i)
		assert.Len(t, tx.Regions[0].Creations, 1) // todo: validate
		assert.Len(t, tx.Regions[0].Transfers, 1)

		regionOffset := i
		assert.Len(t, tx.Regions[regionOffset].Closures, 1)

		assoc, err := token.GetAssociatedAccount(keys[1], keys[2])
		require.NoError(t, err)

		assert.EqualValues(t, tx.Regions[0].Creations[0].Create.Funder, keys[0])
		assert.EqualValues(t, tx.Regions[0].Creations[0].Create.Address, assoc)
		assert.EqualValues(t, tx.Regions[0].Creations[0].Create.Owner, token.ProgramKey)

		assert.EqualValues(t, tx.Regions[0].Creations[0].Initialize.Account, assoc)
		assert.EqualValues(t, tx.Regions[0].Creations[0].Initialize.Mint, keys[2])
		assert.False(t, bytes.Equal(tx.Regions[0].Creations[0].Initialize.Owner, keys[1]))

		assert.EqualValues(t, tx.Regions[0].Creations[0].CloseAuthority.Account, assoc)
		assert.EqualValues(t, tx.Regions[0].Creations[0].CloseAuthority.NewAuthority, keys[0])
		assert.EqualValues(t, tx.Regions[0].Creations[0].CloseAuthority.Type, token.AuthorityTypeCloseAccount)

		assert.EqualValues(t, tx.Regions[0].Creations[0].AccountHolder.Account, assoc)
		assert.EqualValues(t, tx.Regions[0].Creations[0].AccountHolder.NewAuthority, keys[1])
		assert.EqualValues(t, tx.Regions[0].Creations[0].AccountHolder.Type, token.AuthorityTypeAccountHolder)

		assert.EqualValues(t, tx.Regions[0].Transfers[0].Source, keys[3])
		assert.EqualValues(t, tx.Regions[0].Transfers[0].Destination, keys[4])
		assert.EqualValues(t, tx.Regions[0].Transfers[0].Owner, keys[1])
		assert.EqualValues(t, tx.Regions[0].Transfers[0].Amount, 10)

		assert.EqualValues(t, tx.Regions[regionOffset].Closures[0].Account, keys[3])
		assert.EqualValues(t, tx.Regions[regionOffset].Closures[0].Destination, keys[0])
		assert.EqualValues(t, tx.Regions[regionOffset].Closures[0].Owner, keys[1])
	}
}

func TestParseTransaction_TextMemo(t *testing.T) {
	keys := generateKeys(t, 5)

	//
	// 'Single' Region Transfer
	//
	input := solana.NewTransaction(
		keys[0],
		memo.Instruction("1-test"),
		token.Transfer(
			keys[1],
			keys[2],
			keys[3],
			10,
		),
		token.Transfer(
			keys[2],
			keys[3],
			keys[4],
			20,
		),
	)
	tx, err := ParseTransaction(input, nil)
	assert.NoError(t, err)
	assert.Equal(t, "test", tx.AppID)
	assert.EqualValues(t, 0, tx.AppIndex)
	assert.Len(t, tx.Regions, 2)
	assert.Equal(t, Region{}, tx.Regions[0])
	assert.Empty(t, tx.Regions[1].Creations)
	assert.Len(t, tx.Regions[1].Transfers, 2)
	assert.Empty(t, tx.Regions[1].Closures)

	for i := 0; i < 2; i++ {
		assert.EqualValues(t, tx.Regions[1].Transfers[i].Source, keys[1+i])
		assert.EqualValues(t, tx.Regions[1].Transfers[i].Destination, keys[2+i])
		assert.EqualValues(t, tx.Regions[1].Transfers[i].Owner, keys[3+i])
		assert.EqualValues(t, tx.Regions[1].Transfers[i].Amount, (1+i)*10)
	}

	//
	// 'Multi' Region Transfer
	//
	input = solana.NewTransaction(
		keys[0],
		memo.Instruction("1-test-alpha"),
		token.Transfer(
			keys[1],
			keys[2],
			keys[3],
			10,
		),
		memo.Instruction("1-test-beta"),
		token.Transfer(
			keys[2],
			keys[3],
			keys[4],
			20,
		),
	)
	tx, err = ParseTransaction(input, nil)
	assert.NoError(t, err)
	assert.Equal(t, "test", tx.AppID)
	assert.EqualValues(t, 0, tx.AppIndex)
	assert.Len(t, tx.Regions, 3)
	assert.Equal(t, Region{}, tx.Regions[0])

	for i := 1; i < len(tx.Regions); i++ {
		assert.Empty(t, tx.Regions[i].Creations)
		assert.Len(t, tx.Regions[i].Transfers, 1)
		assert.Empty(t, tx.Regions[i].Closures)
	}

	for i := 1; i < len(tx.Regions); i++ {
		assert.EqualValues(t, tx.Regions[i].Transfers[0].Source, keys[1+i-1])
		assert.EqualValues(t, tx.Regions[i].Transfers[0].Destination, keys[2+i-1])
		assert.EqualValues(t, tx.Regions[i].Transfers[0].Owner, keys[3+i-1])
		assert.EqualValues(t, tx.Regions[i].Transfers[0].Amount, (i)*10)
	}

	//
	// Sender Create (with and without GC region)
	//
	createInstructions := generateCreate(t, keys[0], keys[1], keys[2])

	inputs := make([]solana.Transaction, 0)
	for i := 0; i < 2; i++ {
		instructions := make([]solana.Instruction, len(createInstructions))
		copy(instructions, createInstructions)

		instructions = append(instructions, memo.Instruction("1-test"))
		instructions = append(instructions, token.Transfer(
			keys[3],
			keys[4],
			keys[1],
			10,
		))

		if i == 1 {
			m, err := NewMemo(1, TransactionTypeP2P, 10, make([]byte, 29))
			require.NoError(t, err)
			instructions = append(instructions, memo.Instruction(base64.StdEncoding.EncodeToString(m[:])))
		}

		instructions = append(instructions, token.CloseAccount(
			keys[3],
			keys[0],
			keys[1],
		))

		inputs = append(inputs, solana.NewTransaction(keys[0], instructions...))
	}

	for i := range inputs {
		tx, err := ParseTransaction(inputs[i], nil)
		assert.NoError(t, err)
		assert.Equal(t, "test", tx.AppID)
		assert.EqualValues(t, 10*i, tx.AppIndex)

		assert.Len(t, tx.Regions, 2+i)
		assert.Len(t, tx.Regions[0].Creations, 1)
		assert.Len(t, tx.Regions[1].Transfers, 1)

		regionOffset := i + 1
		assert.Len(t, tx.Regions[regionOffset].Closures, 1)

		assert.EqualValues(t, tx.Regions[1].Transfers[0].Source, keys[3])
		assert.EqualValues(t, tx.Regions[1].Transfers[0].Destination, keys[4])
		assert.EqualValues(t, tx.Regions[1].Transfers[0].Owner, keys[1])
		assert.EqualValues(t, tx.Regions[1].Transfers[0].Amount, 10)

		assert.EqualValues(t, tx.Regions[regionOffset].Closures[0].Account, keys[3])
		assert.EqualValues(t, tx.Regions[regionOffset].Closures[0].Destination, keys[0])
		assert.EqualValues(t, tx.Regions[regionOffset].Closures[0].Owner, keys[1])
	}

	//
	// Conflicting AppIDs
	//
	input = solana.NewTransaction(
		keys[0],
		memo.Instruction("1-alph"),
		token.Transfer(
			keys[1],
			keys[2],
			keys[3],
			10,
		),
		memo.Instruction("1-beta"),
		token.Transfer(
			keys[2],
			keys[3],
			keys[4],
			20,
		),
	)
	tx, err = ParseTransaction(input, nil)
	assert.Error(t, err)
}

func TestParseTransaction_OptionalAuthority(t *testing.T) {
	keys := generateKeys(t, 3)

	createAssoc, assoc, err := token.CreateAssociatedTokenAccount(keys[0], keys[1], keys[2])
	require.NoError(t, err)

	txs := []solana.Transaction{
		solana.NewTransaction(
			keys[0],
			generateCreate(t, keys[0], keys[1], keys[2])[:3]...,
		),
		solana.NewTransaction(
			keys[0],
			createAssoc,
			token.SetAuthority(
				assoc,
				assoc,
				keys[0],
				token.AuthorityTypeCloseAccount,
			),
		),
	}

	for _, tx := range txs {
		parsed, err := ParseTransaction(tx, nil)

		assert.NoError(t, err)
		assert.Equal(t, 1, len(parsed.Regions))
		assert.NotNil(t, parsed.Regions[0].Creations)
		assert.NotNil(t, parsed.Regions[0].Creations[0].CloseAuthority)
		assert.Nil(t, parsed.Regions[0].Creations[0].AccountHolder)
		assert.EqualValues(t, keys[0], parsed.Regions[0].Creations[0].CloseAuthority.NewAuthority)
	}
}

func TestParseTransaction_MissingAuthority(t *testing.T) {
	keys := generateKeys(t, 3)

	createAssoc, _, err := token.CreateAssociatedTokenAccount(keys[0], keys[1], keys[2])
	require.NoError(t, err)

	txs := []solana.Transaction{
		solana.NewTransaction(
			keys[0],
			generateCreate(t, keys[0], keys[1], keys[2])[:2]...,
		),
		solana.NewTransaction(
			keys[0],
			createAssoc,
		),
	}

	for _, tx := range txs {
		_, err := ParseTransaction(tx, nil)
		assert.Error(t, err)
	}
}

func TestParseTransaction_Invoices(t *testing.T) {
	keys := generateKeys(t, 5)

	//
	// Basic Transfer
	//
	input := solana.NewTransaction(
		keys[0],
		getInvoiceMemoInstruction(t, TransactionTypeSpend, 10, 2),
		token.Transfer(
			keys[1],
			keys[2],
			keys[3],
			10,
		),
		token.Transfer(
			keys[2],
			keys[3],
			keys[4],
			20,
		),
	)
	tx, err := ParseTransaction(input, nil)
	assert.NoError(t, err)
	assert.Equal(t, "", tx.AppID)
	assert.EqualValues(t, 10, tx.AppIndex)
	assert.Len(t, tx.Regions, 2)
	assert.Equal(t, Region{}, tx.Regions[0])
	assert.Empty(t, tx.Regions[1].Creations)
	assert.Len(t, tx.Regions[1].Transfers, 2)
	assert.Empty(t, tx.Regions[1].Closures)

	for i := 0; i < 2; i++ {
		assert.EqualValues(t, tx.Regions[1].Transfers[i].Source, keys[1+i])
		assert.EqualValues(t, tx.Regions[1].Transfers[i].Destination, keys[2+i])
		assert.EqualValues(t, tx.Regions[1].Transfers[i].Owner, keys[3+i])
	}

	//
	// Multi-Region transfer
	//
	for _, txType := range []TransactionType{TransactionTypeSpend, TransactionTypeP2P} {
		input = solana.NewTransaction(
			keys[0],
			getInvoiceMemoInstruction(t, TransactionTypeSpend, 10, 1),
			token.Transfer(
				keys[1],
				keys[2],
				keys[3],
				10,
			),
			getInvoiceMemoInstruction(t, txType, 10, 1),
			token.Transfer(
				keys[2],
				keys[3],
				keys[4],
				20,
			),
		)
		tx, err = ParseTransaction(input, nil)
		assert.NoError(t, err)
		assert.Equal(t, "", tx.AppID)
		assert.EqualValues(t, 10, tx.AppIndex)
		assert.Len(t, tx.Regions, 3)
		assert.Equal(t, Region{}, tx.Regions[0])

		for i := 1; i < len(tx.Regions); i++ {
			assert.Empty(t, tx.Regions[i].Creations)
			assert.Empty(t, tx.Regions[i].Closures)
			assert.Len(t, tx.Regions[i].Transfers, 1)

			assert.EqualValues(t, tx.Regions[i].Transfers[0].Source, keys[i])
			assert.EqualValues(t, tx.Regions[i].Transfers[0].Destination, keys[i+1])
			assert.EqualValues(t, tx.Regions[i].Transfers[0].Owner, keys[i+2])
			assert.EqualValues(t, tx.Regions[i].Transfers[0].Amount, i*10)
		}
	}

	//
	// Sender Create (with and without GC region)
	//
	createInstructions := generateCreate(t, keys[0], keys[1], keys[2])

	inputs := make([]solana.Transaction, 0)
	for i := 0; i < 2; i++ {
		instructions := make([]solana.Instruction, len(createInstructions))
		copy(instructions, createInstructions)

		instructions = append(instructions, getInvoiceMemoInstruction(t, TransactionTypeSpend, 10, 1))
		instructions = append(instructions, token.Transfer(
			keys[3],
			keys[4],
			keys[1],
			10,
		))

		if i == 1 {
			m, err := NewMemo(1, TransactionTypeP2P, 10, make([]byte, 29))
			require.NoError(t, err)
			instructions = append(instructions, memo.Instruction(base64.StdEncoding.EncodeToString(m[:])))
		}

		instructions = append(instructions, token.CloseAccount(
			keys[3],
			keys[0],
			keys[1],
		))

		inputs = append(inputs, solana.NewTransaction(keys[0], instructions...))
	}

	for i := range inputs {
		tx, err := ParseTransaction(inputs[i], nil)
		assert.NoError(t, err)
		assert.Equal(t, "", tx.AppID)
		assert.EqualValues(t, 10, tx.AppIndex)

		assert.Len(t, tx.Regions, 2+i)
		assert.Len(t, tx.Regions[0].Creations, 1)
		assert.Len(t, tx.Regions[1].Transfers, 1)

		regionOffset := i + 1
		assert.Len(t, tx.Regions[regionOffset].Closures, 1)

		assert.EqualValues(t, tx.Regions[1].Transfers[0].Source, keys[3])
		assert.EqualValues(t, tx.Regions[1].Transfers[0].Destination, keys[4])
		assert.EqualValues(t, tx.Regions[1].Transfers[0].Owner, keys[1])
		assert.EqualValues(t, tx.Regions[1].Transfers[0].Amount, 10)

		assert.EqualValues(t, tx.Regions[regionOffset].Closures[0].Account, keys[3])
		assert.EqualValues(t, tx.Regions[regionOffset].Closures[0].Destination, keys[0])
		assert.EqualValues(t, tx.Regions[regionOffset].Closures[0].Owner, keys[1])
	}

	//
	// Mixed types (invalid)
	//
	for _, txType := range []TransactionType{TransactionTypeSpend, TransactionTypeP2P} {
		input = solana.NewTransaction(
			keys[0],
			getInvoiceMemoInstruction(t, TransactionTypeEarn, 10, 1),
			token.Transfer(
				keys[1],
				keys[2],
				keys[3],
				10,
			),
			getInvoiceMemoInstruction(t, txType, 10, 1),
			token.Transfer(
				keys[2],
				keys[3],
				keys[4],
				20,
			),
		)
		tx, err = ParseTransaction(input, nil)
		assert.Error(t, err)
	}

	//
	// Mixed app IDs
	//
	input = solana.NewTransaction(
		keys[0],
		getInvoiceMemoInstruction(t, TransactionTypeEarn, 10, 1),
		token.Transfer(
			keys[1],
			keys[2],
			keys[3],
			10,
		),
		getInvoiceMemoInstruction(t, TransactionTypeEarn, 20, 1),
		token.Transfer(
			keys[2],
			keys[3],
			keys[4],
			20,
		),
	)
	tx, err = ParseTransaction(input, nil)
	assert.Error(t, err)

	//
	// No matching region
	//
	il := &commonpb.InvoiceList{
		Invoices: []*commonpb.Invoice{
			{
				Items: []*commonpb.Invoice_LineItem{
					{
						Title: "Item1",
					},
				},
			},
		},
	}
	raw, err := proto.Marshal(il)
	require.NoError(t, err)
	h := sha256.Sum224(raw)
	fk := make([]byte, 29)
	copy(fk, h[:])

	m, err := NewMemo(1, TransactionTypeEarn, 10, fk)
	require.NoError(t, err)

	input = solana.NewTransaction(
		keys[0],
		getInvoiceMemoInstruction(t, TransactionTypeEarn, 20, 1),
		token.Transfer(
			keys[1],
			keys[2],
			keys[3],
			10,
		),
		getInvoiceMemoInstruction(t, TransactionTypeEarn, 20, 1),
		token.Transfer(
			keys[2],
			keys[3],
			keys[4],
			20,
		),
	)
	tx, err = ParseTransaction(input, il)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "exactly one region"))

	//
	// Multi-matched region
	//
	input = solana.NewTransaction(
		keys[0],
		memo.Instruction(base64.StdEncoding.EncodeToString(m[:])),
		token.Transfer(
			keys[1],
			keys[2],
			keys[3],
			10,
		),
		memo.Instruction(base64.StdEncoding.EncodeToString(m[:])),
		token.Transfer(
			keys[2],
			keys[3],
			keys[4],
			20,
		),
	)
	tx, err = ParseTransaction(input, il)
	assert.True(t, strings.Contains(err.Error(), "exactly one region"))

	//
	// Invoice Mismatched count
	//
	input = solana.NewTransaction(
		keys[0],
		memo.Instruction(base64.StdEncoding.EncodeToString(m[:])),
		token.Transfer(
			keys[1],
			keys[2],
			keys[3],
			10,
		),
		token.Transfer(
			keys[2],
			keys[3],
			keys[4],
			20,
		),
	)
	tx, err = ParseTransaction(input, il)
	assert.Error(t, err)
}

func TestParseTransaction_InvalidInstructions(t *testing.T) {
	keys := generateKeys(t, 4)

	invalidInstructions := []solana.Instruction{
		token.SetAuthority(
			keys[1],
			keys[2],
			keys[2],
			token.AuthorityTypeAccountHolder,
		),
		token.InitializeAccount(
			keys[1],
			keys[2],
			keys[3],
		),
		system.CreateAccount(
			keys[1],
			keys[2],
			keys[3],
			10,
			10,
		),
		system.AdvanceNonce(
			keys[1],
			keys[2],
		),
	}

	for i := range invalidInstructions {
		tx := solana.NewTransaction(
			keys[0],
			token.Transfer(
				keys[1],
				keys[2],
				keys[3],
				10,
			),
			invalidInstructions[i],
		)

		_, err := ParseTransaction(tx, nil)
		assert.Error(t, err)
	}
}

func TestParseTransaction_NoSignatures(t *testing.T) {
	keys := generateKeys(t, 4)

	tx := solana.NewTransaction(
		keys[0],
		token.Transfer(
			keys[1],
			keys[2],
			keys[3],
			10,
		),
	)
	tx.Signatures = nil

	_, err := ParseTransaction(tx, nil)
	assert.Error(t, err)
	assert.EqualError(t, err, "no allocated signatures")
}

func getInvoiceMemoInstruction(t *testing.T, txType TransactionType, appIndex, transferCount int) solana.Instruction {
	il := &commonpb.InvoiceList{}
	for i := 0; i < transferCount; i++ {
		il.Invoices = append(il.Invoices, &commonpb.Invoice{
			Items: []*commonpb.Invoice_LineItem{
				{
					Title: "Item1",
				},
			},
		})
	}

	raw, err := proto.Marshal(il)
	require.NoError(t, err)

	h := sha256.Sum224(raw)
	fk := make([]byte, 29)
	copy(fk, h[:])

	m, err := NewMemo(1, txType, uint16(appIndex), fk)
	require.NoError(t, err)

	return memo.Instruction(base64.StdEncoding.EncodeToString(m[:]))
}

func generateKeys(t *testing.T, n int) []ed25519.PublicKey {
	keys := make([]ed25519.PublicKey, n)

	for i := 0; i < n; i++ {
		var err error
		keys[i], _, err = ed25519.GenerateKey(nil)
		require.NoError(t, err)
	}

	return keys
}

func generateCreate(t *testing.T, subsidizer, wallet, mint ed25519.PublicKey) []solana.Instruction {
	addr, err := token.GetAssociatedAccount(wallet, mint)
	require.NoError(t, err)

	pub, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	return []solana.Instruction{
		system.CreateAccount(
			subsidizer,
			addr,
			token.ProgramKey,
			token.AccountSize, // validated later
			token.AccountSize,
		),
		token.InitializeAccount(
			addr,
			mint,
			pub,
		),
		token.SetAuthority(
			addr,
			pub,
			subsidizer,
			token.AuthorityTypeCloseAccount,
		),
		token.SetAuthority(
			addr,
			pub,
			wallet,
			token.AuthorityTypeAccountHolder,
		),
	}
}
