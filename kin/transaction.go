package kin

import (
	"bytes"
	"crypto/sha256"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"

	commonpb "github.com/kinecosystem/agora-api/genproto/common/v3"

	"github.com/kinecosystem/agora-common/solana"
	"github.com/kinecosystem/agora-common/solana/memo"
	"github.com/kinecosystem/agora-common/solana/system"
	"github.com/kinecosystem/agora-common/solana/token"
)

// Tx represents a parsed kin solana transaction
type Tx struct {
	AppIndex uint16
	AppID    string

	Regions []Region
}

// Region is an abstract 'region' within a transaction.
//
// See the documentation for SignTransaction in the transaction service API.
type Region struct {
	MemoData []byte
	Memo     *Memo

	Creations []Creation
	Transfers []*token.DecompiledTransfer
	Closures  []*token.DecompiledCloseAccount
}

type Creation struct {
	Create     *system.DecompiledCreateAccount
	Initialize *token.DecompiledInitializeAccount

	CreateAssoc *token.DecompiledCreateAssociatedAccount

	CloseAuthority *token.DecompiledSetAuthority
	AccountHolder  *token.DecompiledSetAuthority
}

// ParseTransaction parses a (solana transaction, invoice list) pair.
//
// The following invariants are checked while parsing:
//   1. Each instruction is one of the following:
//     - Memo::Memo
//     - System::CreateAccount
//     - SplToken::InitializeAccount
//     - SplAssociatedToken::CreateAssociatedAccount
//     - SplToken::SetAuthority
//     - SplToken::Transfer
//     - SplToken::CloseAccount
//   2. If an invoice is provided, it must match _exactly_ one region.
//   3. Transfer instructions cannot use the subsidizer as a source.
//   4. SetAuthority can only be related to a newly created account.
//   5. SetAuthority can only be of the types AccountHolder and CloseAccount.
//   6. SetAuthority must be related to a SplToken::Initialize/SplAssociatedToken::CreateAssociatedAccount instruction.
//   7. There cannot be multiple values (excluding none) of AppIndex or AppID.
//     - The link between AppIndex and AppID is not validated.
//   8. Earns cannot be mixed with P2P/Spend payments.
func ParseTransaction(
	tx solana.Transaction,
	il *commonpb.InvoiceList,
) (parsed Tx, err error) {
	if len(tx.Message.Instructions) == 0 {
		return parsed, errors.New("no instructions")
	}
	if len(tx.Signatures) == 0 {
		return parsed, errors.New("no allocated signatures")
	}

	parsed.Regions = make([]Region, 1)

	for i := 0; i < len(tx.Message.Instructions); i++ {
		if isMemo(&tx, i) {
			m, err := memo.DecompileMemo(tx.Message, i)
			if err != nil {
				return parsed, errors.Wrapf(err, "invalid Memo::Memo at %d", i)
			}

			parsed.Regions = append(parsed.Regions, Region{
				MemoData: m.Data,
			})
		} else if isSystem(&tx, i) {
			creation, err := system.DecompileCreateAccount(tx.Message, i)
			if err != nil {
				return parsed, errors.Wrapf(err, "invalid System::CreateAccount at %d", i)
			}

			if !bytes.Equal(creation.Owner, token.ProgramKey) {
				return parsed, errors.New("System::CreateAccount must assign owner to the SplToken program")
			}
			if creation.Size != token.AccountSize {
				return parsed, errors.New("invalid size in System::CreateAccount")
			}

			i++
			if i == len(tx.Message.Instructions) {
				return parsed, errors.New("missing SplToken::InitializeAccount instruction")
			}

			initialize, err := token.DecompileInitializeAccount(tx.Message, i)
			if err != nil {
				return parsed, errors.Wrapf(err, "invalid SplToken::InitializeAccount")
			}
			if !bytes.Equal(creation.Address, initialize.Account) {
				return parsed, errors.New("SplToken::InitializeAccount address does not match System::CreateAccount address")
			}

			i++
			if i == len(tx.Message.Instructions) {
				return parsed, errors.New("missing SplToken::SetAuthority(Close) instruction")
			}

			closeAuthority, err := token.DecompileSetAuthority(tx.Message, i)
			if err != nil {
				return parsed, errors.Wrapf(err, "invalid SplToken::SetAuthority")
			}
			if closeAuthority.Type != token.AuthorityTypeCloseAccount {
				return parsed, errors.New("SplToken::SetAuthority must be of type Close following an initialize")
			}
			if !bytes.Equal(closeAuthority.Account, creation.Address) {
				return parsed, errors.New("SplToken::SetAuthority(Close) authority must be for the created account")
			}

			parsed.Regions[len(parsed.Regions)-1].Creations = append(parsed.Regions[len(parsed.Regions)-1].Creations, Creation{
				Create:         creation,
				Initialize:     initialize,
				CloseAuthority: closeAuthority,
			})

			// Changing of the account holder is optional.
			i++
			if i == len(tx.Message.Instructions) {
				break
			}

			accountHolder, err := token.DecompileSetAuthority(tx.Message, i)
			if err != nil {
				i--
				continue
			}

			if accountHolder.Type != token.AuthorityTypeAccountHolder {
				return parsed, errors.New("SplToken::SetAuthority must be of type AccountHolder following a close authority")
			}
			if !bytes.Equal(accountHolder.Account, creation.Address) {
				return parsed, errors.New("SplToken::SetAuthority(AccountHolder) must be for the created account")
			}

			lastIdx := len(parsed.Regions[len(parsed.Regions)-1].Creations) - 1
			parsed.Regions[len(parsed.Regions)-1].Creations[lastIdx].AccountHolder = accountHolder
		} else if isSPLAssoc(&tx, i) {
			create, err := token.DecompileCreateAssociatedAccount(tx.Message, i)
			if err != nil {
				return parsed, errors.Wrap(err, "invalid SplAssociatedToken::CreateAssociatedAccount")
			}

			i++
			if i == len(tx.Message.Instructions) {
				return parsed, errors.New("missing SplToken::SetAuthority(Close) instruction")
			}

			closeAuthority, err := token.DecompileSetAuthority(tx.Message, i)
			if err != nil {
				return parsed, errors.Wrapf(err, "invalid SplToken::SetAuthority")
			}
			if closeAuthority.Type != token.AuthorityTypeCloseAccount {
				return parsed, errors.New("SplToken::SetAuthority must be of type Close following an initialize")
			}
			if !bytes.Equal(closeAuthority.Account, create.Address) {
				return parsed, errors.New("SplToken::SetAuthority(Close) authority must be for the created account")
			}

			parsed.Regions[len(parsed.Regions)-1].Creations = append(parsed.Regions[len(parsed.Regions)-1].Creations, Creation{
				CreateAssoc:    create,
				CloseAuthority: closeAuthority,
			})
		} else if isSPL(&tx, i) {
			cmd, _ := token.GetCommand(tx.Message, i)
			switch cmd {
			case token.CommandTransfer:
				transfer, err := token.DecompileTransfer(tx.Message, i)
				if err != nil {
					return parsed, errors.Wrapf(err, "invalid SplToken::Transfer at %d", i)
				}

				// Ensure that the transfer doesn't reference the subsidizer.
				if bytes.Equal(transfer.Owner, tx.Message.Accounts[0]) {
					return parsed, errors.New("cannot transfer from a subsidizer owned account")
				}

				parsed.Regions[len(parsed.Regions)-1].Transfers = append(parsed.Regions[len(parsed.Regions)-1].Transfers, transfer)
			case token.CommandCloseAccount:
				closure, err := token.DecompileCloseAccount(tx.Message, i)
				if err != nil {
					return parsed, errors.Wrapf(err, "invalid SplToken::CloseAccount at %d", i)
				}

				parsed.Regions[len(parsed.Regions)-1].Closures = append(parsed.Regions[len(parsed.Regions)-1].Closures, closure)
			default:
				return parsed, errors.Errorf("unsupported instruction at %d", i)
			}
		} else {
			return parsed, errors.Errorf("invalid instruction type at: %d", i)
		}
	}

	var refCount int
	var ilHash [sha256.Size224]byte
	if il != nil {
		raw, err := proto.Marshal(il)
		if err != nil {
			return parsed, errors.Wrap(err, "failed to marshal invoice list")
		}

		ilHash = sha256.Sum224(raw)
	}

	var hasEarn, hasSpend, hasP2P bool
	for r := range parsed.Regions {
		//
		// Validate CloseAuthority matches
		//
		for c := range parsed.Regions[r].Creations {
			closeAuth := parsed.Regions[r].Creations[c].CloseAuthority.NewAuthority

			if parsed.Regions[r].Creations[c].CreateAssoc != nil {
				if !bytes.Equal(parsed.Regions[r].Creations[c].CreateAssoc.Subsidizer, closeAuth) {
					return parsed, errors.New("SplToken::SetAuthority has incorrect new authority")
				}
			} else if parsed.Regions[r].Creations[c].Create != nil {
				if !bytes.Equal(parsed.Regions[r].Creations[c].Create.Funder, closeAuth) {
					return parsed, errors.New("SplToken::SetAuthority has incorrect new authority")
				}
			} else {
				// note: this shouldn't happen, but just in case.
				return parsed, errors.New("create without create instruction")
			}
		}

		//
		// Validate / extract memos
		//
		if len(parsed.Regions[r].MemoData) == 0 {
			continue
		}

		// Attempt to pull out an AppID or AppIndex from the memo data.
		//
		// If either are set, then we need to ensure that it's either the
		// "first" value for the transaction, or that it is the same as the
		// existing ones.
		//
		// Note: we don't care about whether or not the AppID/AppIndex match in
		// this case. We leave that up to the caller to verify/authorize.
		m, err := MemoFromBase64String(string(parsed.Regions[r].MemoData), false)
		if err != nil {
			if appID, ok := AppIDFromTextMemo(string(parsed.Regions[r].MemoData)); ok {
				if parsed.AppID == "" {
					parsed.AppID = appID
				} else if parsed.AppID != appID {
					return parsed, errors.Errorf("multiple app ids")
				}
			}

			continue
		}

		// From this point on, we assume we we have an invoice based memo.
		parsed.Regions[r].Memo = &m

		switch m.TransactionType() {
		case TransactionTypeEarn:
			hasEarn = true
		case TransactionTypeSpend:
			hasSpend = true
		case TransactionTypeP2P:
			hasP2P = true
		}

		if parsed.AppIndex > 0 && m.AppIndex() != parsed.AppIndex {
			return parsed, errors.Errorf("multiple app indexes")
		} else if parsed.AppIndex == 0 {
			parsed.AppIndex = m.AppIndex()
		}

		if il == nil {
			continue
		}

		fk := m.ForeignKey()
		if !bytes.Equal(fk[:28], ilHash[:]) || fk[28] != 0 {
			continue
		}

		refCount++
		if len(il.Invoices) != len(parsed.Regions[r].Transfers) {
			return parsed, errors.Errorf(
				"invoice count (%d) does not match transfer count (%d) in region %d",
				len(il.Invoices),
				len(parsed.Regions[r].Transfers),
				r,
			)
		}
	}

	if hasEarn && (hasSpend || hasP2P) {
		return parsed, errors.New("cannot mix earns with P2P/spends")
	}
	if il != nil && refCount != 1 {
		return parsed, errors.Errorf("invoice list does not match to exactly one region (matches %d regions)", refCount)
	}

	return parsed, nil
}

func isMemo(tx *solana.Transaction, index int) bool {
	return bytes.Equal(tx.Message.Accounts[tx.Message.Instructions[index].ProgramIndex], memo.ProgramKey)
}

func isSPL(tx *solana.Transaction, index int) bool {
	return bytes.Equal(tx.Message.Accounts[tx.Message.Instructions[index].ProgramIndex], token.ProgramKey)
}

func isSPLAssoc(tx *solana.Transaction, index int) bool {
	return bytes.Equal(tx.Message.Accounts[tx.Message.Instructions[index].ProgramIndex], token.AssociatedTokenAccountProgramKey)
}

func isSystem(tx *solana.Transaction, index int) bool {
	return bytes.Equal(tx.Message.Accounts[tx.Message.Instructions[index].ProgramIndex], system.ProgramKey[:])
}
