package token

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"

	"github.com/pkg/errors"

	"github.com/kinecosystem/agora-common/solana"
	"github.com/kinecosystem/agora-common/solana/system"
)

// ProgramKey is the address of the token program that should be used.
//
// Current key: TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA
//
// todo: lock this in, or make configurable.
var ProgramKey = ed25519.PublicKey{6, 221, 246, 225, 215, 101, 161, 147, 217, 203, 225, 70, 206, 235, 121, 172, 28, 180, 133, 237, 95, 91, 55, 145, 58, 140, 245, 133, 126, 255, 0, 169}

type command byte

const (
	// nolint:varcheck,deadcode,unused
	commandInitializeMint command = iota
	commandInitializeAccount
	// nolint:varcheck,deadcode,unused
	commandInitializeMultisig
	commandTransfer
	// nolint:varcheck,deadcode,unused
	commandApprove
	// nolint:varcheck,deadcode,unused
	commandRevoke
	commandSetAuthority
	// nolint:varcheck,deadcode,unused
	commandMintTo
	// nolint:varcheck,deadcode,unused
	commandBurn
	// nolint:varcheck,deadcode,unused
	commandCloseAccount
	// nolint:varcheck,deadcode,unused
	commandFreezeAccount
	// nolint:varcheck,deadcode,unused
	commandThawAccount
	// nolint:varcheck,deadcode,unused
	commandTransfer2
	// nolint:varcheck,deadcode,unused
	commandApprove2
	// nolint:varcheck,deadcode,unused
	commandMintTo2
	// nolint:varcheck,deadcode,unused
	commandBurn2
)

const (
	// nolint:varcheck,deadcode,unused
	ErrorNotRentExempt solana.CustomError = iota
	// nolint:varcheck,deadcode,unused
	ErrorInsufficientFunds
	// nolint:varcheck,deadcode,unused
	ErrorInvalidMint
	// nolint:varcheck,deadcode,unused
	ErrorMintMismatch
	// nolint:varcheck,deadcode,unused
	ErrorOwnerMismatch
	// nolint:varcheck,deadcode,unused
	ErrorFixedSupply
	// nolint:varcheck,deadcode,unused
	ErrorAlreadyInUse
	// nolint:varcheck,deadcode,unused
	ErrorInvalidNumberOfProvidedSigners
	// nolint:varcheck,deadcode,unused
	ErrorInvalidNumberOfRequiredSigners
	// nolint:varcheck,deadcode,unused
	ErrorUninitializedState
	// nolint:varcheck,deadcode,unused
	ErrorNativeNotSupported
	// nolint:varcheck,deadcode,unused
	ErrorNonNativeHasBalance
	// nolint:varcheck,deadcode,unused
	ErrorInvalidInstruction
	// nolint:varcheck,deadcode,unused
	ErrorInvalidState
	// nolint:varcheck,deadcode,unused
	ErrorOverflow
	// nolint:varcheck,deadcode,unused
	ErrorAuthorityTypeNotSupported
	// nolint:varcheck,deadcode,unused
	ErrorMintCannotFreeze
	// nolint:varcheck,deadcode,unused
	ErrorAccountFrozen
	// nolint:varcheck,deadcode,unused
	ErrorMintDecimalsMismatch
)

// Reference: https://github.com/solana-labs/solana-program-library/blob/b011698251981b5a12088acba18fad1d41c3719a/token/program/src/instruction.rs#L41-L55
func InitializeAccount(account, mint, owner ed25519.PublicKey) solana.Instruction {
	// Accounts expected by this instruction:
	//
	//   0. `[writable]`  The account to initialize.
	//   1. `[]` The mint this account will be associated with.
	//   2. `[]` The new account's owner/multisignature.
	//   3. `[]` Rent sysvar
	return solana.NewInstruction(
		ProgramKey,
		[]byte{byte(commandInitializeAccount)},
		solana.NewAccountMeta(account, true),
		solana.NewReadonlyAccountMeta(mint, false),
		solana.NewReadonlyAccountMeta(owner, false),
		solana.NewReadonlyAccountMeta(system.RentSysVar, false),
	)
}

type DecompiledInitializeAccount struct {
	Account ed25519.PublicKey
	Mint    ed25519.PublicKey
	Owner   ed25519.PublicKey
}

func DecompileInitializeAccount(m solana.Message, index int) (*DecompiledInitializeAccount, error) {
	if index >= len(m.Instructions) {
		return nil, errors.Errorf("instruction doesn't exist at %d", index)
	}

	i := m.Instructions[index]

	if !bytes.Equal(m.Accounts[i.ProgramIndex], ProgramKey) {
		return nil, solana.ErrIncorrectProgram
	}
	if !bytes.Equal([]byte{byte(commandInitializeAccount)}, i.Data) {
		return nil, solana.ErrIncorrectInstruction
	}
	if len(i.Accounts) != 4 {
		return nil, errors.Errorf("invalid number of accounts: %d", len(i.Accounts))
	}
	if !bytes.Equal(system.RentSysVar, m.Accounts[i.Accounts[3]]) {
		return nil, errors.Errorf("invalid rent program")
	}

	return &DecompiledInitializeAccount{
		Account: m.Accounts[i.Accounts[0]],
		Mint:    m.Accounts[i.Accounts[1]],
		Owner:   m.Accounts[i.Accounts[2]],
	}, nil
}

type AuthorityType byte

const (
	AuthorityTypeMintTokens AuthorityType = iota
	AuthorityTypeFreezeAccount
	AuthorityTypeAccountHolder
	AuthorityTypeCloseAccount
)

// Reference: https://github.com/solana-labs/solana-program-library/blob/b011698251981b5a12088acba18fad1d41c3719a/token/program/src/instruction.rs#L128-L139
func SetAuthority(account, currentAuthority, newAuthority ed25519.PublicKey, authorityType AuthorityType) solana.Instruction {
	// Sets a new authority of a mint or account.
	//
	// Accounts expected by this instruction:
	//
	//   * Single authority
	//   0. `[writable]` The mint or account to change the authority of.
	//   1. `[signer]` The current authority of the mint or account.
	//
	//   * Multisignature authority
	//   0. `[writable]` The mint or account to change the authority of.
	//   1. `[]` The mint's or account's multisignature authority.
	//   2. ..2+M `[signer]` M signer accounts
	data := []byte{byte(commandSetAuthority), byte(authorityType), 0}
	if len(newAuthority) > 0 {
		data[2] = 1
		data = append(data, newAuthority...)
	}

	return solana.NewInstruction(
		ProgramKey,
		data,
		solana.NewAccountMeta(account, false),
		solana.NewReadonlyAccountMeta(currentAuthority, true),
	)
}

type DecompiledSetAuthority struct {
	Account          ed25519.PublicKey
	CurrentAuthority ed25519.PublicKey
	NewAuthority     ed25519.PublicKey
	Type             AuthorityType
}

func DecompileSetAuthority(m solana.Message, index int) (*DecompiledSetAuthority, error) {
	if index >= len(m.Instructions) {
		return nil, errors.Errorf("instruction doesn't exist at %d", index)
	}

	i := m.Instructions[index]

	if !bytes.Equal(m.Accounts[i.ProgramIndex], ProgramKey) {
		return nil, solana.ErrIncorrectProgram
	}
	if len(i.Accounts) != 2 {
		return nil, errors.Errorf("invalid number of accounts: %d", len(i.Accounts))
	}
	if len(i.Data) < 3 {
		return nil, errors.Errorf("invalid data size: %d (expect at least 3)", len(i.Data))
	}
	if i.Data[0] != byte(commandSetAuthority) {
		return nil, solana.ErrIncorrectInstruction
	}
	if i.Data[2] == 0 && len(i.Data) != 3 {
		return nil, errors.Errorf("invalid data size: %d (expect 3)", len(i.Data))
	}
	if i.Data[2] == 1 && len(i.Data) != 3+ed25519.PublicKeySize {
		return nil, errors.Errorf("invalid data size: %d (expect %d)", len(i.Data), 3+ed25519.PublicKeySize)
	}

	decompiled := &DecompiledSetAuthority{
		Account:          m.Accounts[i.Accounts[0]],
		CurrentAuthority: m.Accounts[i.Accounts[1]],
		Type:             AuthorityType(i.Data[1]),
	}

	if i.Data[2] == 1 {
		decompiled.NewAuthority = i.Data[3 : 3+ed25519.PublicKeySize]
	}

	return decompiled, nil
}

// todo(feature): support multi-sig
//
// Reference: https://github.com/solana-labs/solana-program-library/blob/b011698251981b5a12088acba18fad1d41c3719a/token/program/src/instruction.rs#L76-L91
func Transfer(source, dest, owner ed25519.PublicKey, amount uint64) solana.Instruction {
	// Accounts expected by this instruction:
	//
	//   * Single owner/delegate
	//   0. `[writable]` The source account.
	//   1. `[writable]` The destination account.
	//   2. `[signer]` The source account's owner/delegate.
	//
	//   * Multisignature owner/delegate
	//   0. `[writable]` The source account.
	//   1. `[writable]` The destination account.
	//   2. `[]` The source account's multisignature owner/delegate.
	//   3. ..3+M `[signer]` M signer accounts.
	data := make([]byte, 1+8)
	data[0] = byte(commandTransfer)
	binary.LittleEndian.PutUint64(data[1:], amount)

	return solana.NewInstruction(
		ProgramKey,
		data,
		solana.NewAccountMeta(source, false),
		solana.NewAccountMeta(dest, false),
		solana.NewAccountMeta(owner, true),
	)
}

type DecompiledTransferAccount struct {
	Source      ed25519.PublicKey
	Destination ed25519.PublicKey
	Owner       ed25519.PublicKey
	Amount      uint64
}

func DecompileTransferAccount(m solana.Message, index int) (*DecompiledTransferAccount, error) {
	if index >= len(m.Instructions) {
		return nil, errors.Errorf("instruction doesn't exist at %d", index)
	}

	i := m.Instructions[index]

	if !bytes.Equal(m.Accounts[i.ProgramIndex], ProgramKey) {
		return nil, solana.ErrIncorrectProgram
	}
	if len(i.Data) == 0 || i.Data[0] != byte(commandTransfer) {
		return nil, solana.ErrIncorrectInstruction
	}
	if len(i.Accounts) != 3 {
		return nil, errors.Errorf("invalid number of accounts: %d", len(i.Accounts))
	}
	if len(i.Data) != 9 {
		return nil, errors.Errorf("invalid instruction data size: %d", len(i.Data))
	}

	v := &DecompiledTransferAccount{
		Source:      m.Accounts[i.Accounts[0]],
		Destination: m.Accounts[i.Accounts[1]],
		Owner:       m.Accounts[i.Accounts[2]],
	}
	v.Amount = binary.LittleEndian.Uint64(i.Data[1:])
	return v, nil
}

// Reference: https://github.com/solana-labs/solana-program-library/blob/b011698251981b5a12088acba18fad1d41c3719a/token/program/src/instruction.rs#L183-L197
func CloseAccount(account, dest, owner ed25519.PublicKey) solana.Instruction {
	// Close an account by transferring all its SOL to the destination account.
	// Non-native accounts may only be closed if its token amount is zero.
	//
	// Accounts expected by this instruction:
	//
	//   * Single owner
	//   0. `[writable]` The account to close.
	//   1. `[writable]` The destination account.
	//   2. `[signer]` The account's owner.
	//
	//   * Multisignature owner
	//   0. `[writable]` The account to close.
	//   1. `[writable]` The destination account.
	//   2. `[]` The account's multisignature owner.
	//   3. ..3+M `[signer]` M signer accounts.
	return solana.NewInstruction(
		ProgramKey,
		[]byte{byte(commandCloseAccount)},
		solana.NewAccountMeta(account, false),
		solana.NewAccountMeta(dest, false),
		solana.NewReadonlyAccountMeta(owner, true),
	)
}
