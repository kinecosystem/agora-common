package token

import (
	"crypto/ed25519"

	"github.com/kinecosystem/agora-common/solana"
	"github.com/kinecosystem/agora-common/solana/system"
)

// AssociatedTokenAccountProgramKey  is the address of the associated token account program that should be used.
//
// Current key: ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL
var AssociatedTokenAccountProgramKey = ed25519.PublicKey{140, 151, 37, 143, 78, 36, 137, 241, 187, 61, 16, 41, 20, 142, 13, 131, 11, 90, 19, 153, 218, 255, 16, 132, 4, 142, 123, 216, 219, 233, 248, 89}

// GetAssociatedAccount returns the associated account address for an SPL token.
//
// Reference: https://spl.solana.com/associated-token-account#finding-the-associated-token-account-address
func GetAssociatedAccount(wallet, mint ed25519.PublicKey) (ed25519.PublicKey, error) {
	return solana.FindProgramAddress(
		AssociatedTokenAccountProgramKey,
		wallet,
		ProgramKey,
		mint,
	)
}

// Reference: https://github.com/solana-labs/solana-program-library/blob/0639953c7dd0f5228c3ceda3ba68fece3b46ff1d/associated-token-account/program/src/lib.rs#L54
func CreateAssociatedTokenAccount(subsidizer, wallet, mint ed25519.PublicKey) (solana.Instruction, error) {
	addr, err := GetAssociatedAccount(wallet, mint)
	if err != nil {
		return solana.Instruction{}, err
	}

	return solana.NewInstruction(
		AssociatedTokenAccountProgramKey,
		[]byte{},
		solana.NewAccountMeta(subsidizer, true),
		solana.NewAccountMeta(addr, false),
		solana.NewReadonlyAccountMeta(wallet, false),
		solana.NewReadonlyAccountMeta(mint, false),
		solana.NewReadonlyAccountMeta(system.ProgramKey[:], false),
		solana.NewReadonlyAccountMeta(ProgramKey, false),
		solana.NewReadonlyAccountMeta(system.RentSysVar, false),
	), nil
}
