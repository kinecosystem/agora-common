package token

import (
	"crypto/ed25519"
	"encoding/binary"
)

type AccountState byte

const (
	AccountStateUninitialized AccountState = iota
	AccountStateInitialized
	AccountStateFrozen
)

// Reference: https://github.com/solana-labs/solana-program-library/blob/11b1e3eefdd4e523768d63f7c70a7aa391ea0d02/token/program/src/state.rs#L125
const AccountSize = 165

// Reference: https://github.com/solana-labs/solana-program-library/blob/8944f428fe693c3a4226bf766a79be9c75e8e520/token/program/src/state.rs#L214
const MultisigAccountSize = 355

type Account struct {
	// The mint associated with this account
	Mint ed25519.PublicKey
	// The owner of this account.
	Owner ed25519.PublicKey
	// The amount of tokens this account holds.
	Amount uint64
	// If set, then the 'DelegatedAmount' represents the amount
	// authorized by the delegate.
	Delegate ed25519.PublicKey
	/// The account's state
	State AccountState
	// If set, this is a native token, and the value logs the rent-exempt reserve. An Account
	// is required to be rent-exempt, so the value is used by the Processor to ensure that wrapped
	// SOL accounts do not drop below this threshold.
	IsNative *uint64
	// The amount delegated
	DelegatedAmount uint64
	// Optional authority to close the account.
	CloseAuthority ed25519.PublicKey
}

func (a *Account) Marshal() []byte {
	b := make([]byte, AccountSize)

	var offset int
	writeKey(b, a.Mint, &offset)
	writeKey(b[offset:], a.Owner, &offset)
	writeUint64(b[offset:], a.Amount, &offset)
	writeOptionalKey(b[offset:], a.Delegate, &offset)
	b[offset] = byte(a.State)
	offset++
	writeOptionalUint64(b[offset:], a.IsNative, &offset)
	writeUint64(b[offset:], a.DelegatedAmount, &offset)
	writeOptionalKey(b[offset:], a.CloseAuthority, &offset)

	return b
}

func writeKey(dst []byte, src []byte, offset *int) {
	copy(dst, src)
	*offset += ed25519.PublicKeySize
}

func writeOptionalKey(dst []byte, src []byte, offset *int) {
	if len(src) > 0 {
		dst[0] = 1
		copy(dst[4:], src)
	}

	*offset += 4 + ed25519.PublicKeySize
}

func writeUint64(dst []byte, v uint64, offset *int) {
	binary.LittleEndian.PutUint64(dst, v)
	*offset += 8
}

func writeOptionalUint64(dst []byte, v *uint64, offset *int) {
	if v != nil {
		dst[0] = 1
		binary.LittleEndian.PutUint64(dst[4:], *v)
	}
	*offset += 4 + 8
}

func (a *Account) Unmarshal(b []byte) bool {
	if len(b) != AccountSize {
		return false
	}

	var offset int
	loadKey(b, &a.Mint, &offset)
	loadKey(b[offset:], &a.Owner, &offset)
	loadUint64(b[offset:], &a.Amount, &offset)
	loadOptionalKey(b[offset:], &a.Delegate, &offset)
	a.State = AccountState(b[offset])
	offset++
	loadOptionalUint64(b[offset:], &a.IsNative, &offset)
	loadUint64(b[offset:], &a.DelegatedAmount, &offset)
	loadOptionalKey(b[offset:], &a.CloseAuthority, &offset)

	return true
}

func loadKey(src []byte, dst *ed25519.PublicKey, offset *int) {
	*dst = make([]byte, ed25519.PublicKeySize)
	copy(*dst, src)
	*offset += ed25519.PublicKeySize
}

func loadOptionalKey(src []byte, dst *ed25519.PublicKey, offset *int) {
	if src[0] == 1 {
		*dst = make([]byte, ed25519.PublicKeySize)
		copy(*dst, src[4:])
	}
	*offset += 4 + ed25519.PublicKeySize
}

func loadUint64(src []byte, dst *uint64, offset *int) {
	*dst = binary.LittleEndian.Uint64(src)
	*offset += 8
}
func loadOptionalUint64(src []byte, dst **uint64, offset *int) {
	if src[0] == 1 {
		val := binary.LittleEndian.Uint64(src[4:])
		*dst = &val
	}
	*offset += 4 + 8
}
