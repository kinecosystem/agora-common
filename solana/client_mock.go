package solana

import (
	"crypto/ed25519"
	"sync"
	"time"

	"github.com/stretchr/testify/mock"
)

type MockClient struct {
	sync.Mutex
	mock.Mock
}

func NewMockClient() *MockClient {
	return &MockClient{}
}

func (m *MockClient) GetMinimumBalanceForRentExemption(size uint64) (lamports uint64, err error) {
	args := m.Called(size)
	switch t := args.Get(0).(type) {
	case int:
		return uint64(t), args.Error(1)
	case int64:
		return uint64(t), args.Error(1)
	case uint64:
		return uint64(t), args.Error(1)
	default:
		panic("invalid size parameter")
	}
}

func (m *MockClient) GetSlot(commitment Commitment) (uint64, error) {
	m.Lock()
	defer m.Unlock()

	args := m.Called(commitment)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockClient) GetRecentBlockhash() (Blockhash, error) {
	m.Lock()
	defer m.Unlock()

	args := m.Called()
	return args.Get(0).(Blockhash), args.Error(1)
}

func (m *MockClient) GetBlockTime(slot uint64) (time.Time, error) {
	m.Lock()
	defer m.Unlock()

	args := m.Called(slot)
	return args.Get(0).(time.Time), args.Error(1)
}

func (m *MockClient) GetConfirmedBlock(slot uint64) (*Block, error) {
	m.Lock()
	defer m.Unlock()

	args := m.Called(slot)
	return args.Get(0).(*Block), args.Error(1)
}

func (m *MockClient) GetConfirmedBlocksWithLimit(start, limit uint64) ([]uint64, error) {
	m.Lock()
	defer m.Unlock()

	args := m.Called(start, limit)
	return args.Get(0).([]uint64), args.Error(1)
}

func (m *MockClient) GetConfirmedTransaction(sig Signature) (ConfirmedTransaction, error) {
	m.Lock()
	defer m.Unlock()

	args := m.Called(sig)
	return args.Get(0).(ConfirmedTransaction), args.Error(1)
}

func (m *MockClient) GetBalance(account ed25519.PublicKey) (uint64, error) {
	m.Lock()
	defer m.Unlock()

	args := m.Called(account)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockClient) SimulateTransaction(txn Transaction) (*TransactionError, error) {
	m.Lock()
	defer m.Unlock()

	args := m.Called(txn)
	return args.Get(0).(*TransactionError), args.Error(1)
}

func (m *MockClient) SubmitTransaction(txn Transaction, commitment Commitment) (Signature, *SignatureStatus, error) {
	m.Lock()
	defer m.Unlock()

	args := m.Called(txn, commitment)
	return args.Get(0).(Signature), args.Get(1).(*SignatureStatus), args.Error(2)
}

func (m *MockClient) GetAccountInfo(account ed25519.PublicKey, commitment Commitment) (AccountInfo, error) {
	m.Lock()
	defer m.Unlock()

	args := m.Called(account, commitment)
	return args.Get(0).(AccountInfo), args.Error(1)
}

func (m *MockClient) RequestAirdrop(account ed25519.PublicKey, lamports uint64, commitment Commitment) (Signature, error) {
	m.Lock()
	defer m.Unlock()

	args := m.Called(account, lamports, commitment)
	return args.Get(0).(Signature), args.Error(1)
}

func (m *MockClient) GetConfirmationStatus(signature Signature, commitment Commitment) (bool, error) {
	m.Lock()
	defer m.Unlock()

	args := m.Called(signature, commitment)
	return args.Bool(0), args.Error(1)
}

func (m *MockClient) GetSignatureStatuses(signature []Signature) ([]*SignatureStatus, error) {
	m.Lock()
	defer m.Unlock()

	args := m.Called(signature)
	return args.Get(0).([]*SignatureStatus), args.Error(1)
}

func (m *MockClient) GetSignatureStatus(signature Signature, commitment Commitment) (*SignatureStatus, error) {
	m.Lock()
	defer m.Unlock()

	args := m.Called(signature)
	return args.Get(0).(*SignatureStatus), args.Error(1)
}

func (m *MockClient) GetTokenAccountsByOwner(owner, mint ed25519.PublicKey) ([]ed25519.PublicKey, error) {
	m.Lock()
	defer m.Unlock()

	args := m.Called(owner, mint)
	return args.Get(0).([]ed25519.PublicKey), args.Error(1)
}
