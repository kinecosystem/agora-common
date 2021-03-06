package solana

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/ybbus/jsonrpc"

	"github.com/kinecosystem/agora-common/metrics"
	"github.com/kinecosystem/agora-common/retry"
	"github.com/kinecosystem/agora-common/retry/backoff"
)

const (
	// todo: we can retrieve these from the Syscall account
	//       but they're unlikely to change.
	ticksPerSec  = 160
	ticksPerSlot = 64
	slotsPerSec  = ticksPerSec / ticksPerSlot

	// PollRate is the rate at which blocks should be polled at.
	PollRate = (time.Second / slotsPerSec) / 2

	// Poll rate is ~2x the slot rate, and we want to wait ~32 slots
	sigStatusPollLimit = 2 * 32

	// Reference: https://github.com/solana-labs/solana/blob/14d793b22c1571fb092d5822189d5b64f32605e6/client/src/rpc_custom_error.rs#L10
	blockNotAvailableCode = -32004

	// Reference: https://github.com/solana-labs/solana/blob/71e9958e061493d7545bd28d4ac7a85aaed6ffbb/client/src/rpc_custom_error.rs#L11
	rpcNodeUnhealthyCode = -32005
)

var (
	rpcCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "solana",
		Name:      "requests_total",
		Help:      "Number of Solana RPCs made",
	}, []string{"method", "response_code"})
	rpcTimings = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "solana",
		Name:      "request_duration_seconds",
	}, []string{"method"})
	retryCount = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "solana",
		Name:      "retry_count",
		Buckets:   prometheus.LinearBuckets(1.0, 1.0, 3),
	}, []string{"method"})
	getSigStatusTimings = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "solana",
		Name:      "get_signature_status_duration_seconds",
		Help:      "Timing information for the GetSignatureStatus library call, which polls the GetSignatureStatus RPC",
		Buckets:   metrics.MinuteDistributionBuckets,
	}, []string{"commitment"})
	getSigStatusRetryCount = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "solana",
		Name:      "get_signature_status_retry_count",
		Buckets:   prometheus.LinearBuckets(1.0, 1.0, sigStatusPollLimit),
	}, []string{"commitment"})
)

func init() {
	rpcCounterVec = metrics.Register(rpcCounterVec).(*prometheus.CounterVec)
	rpcTimings = metrics.Register(rpcTimings).(*prometheus.HistogramVec)
	retryCount = metrics.Register(retryCount).(*prometheus.HistogramVec)
	getSigStatusTimings = metrics.Register(getSigStatusTimings).(*prometheus.HistogramVec)
	getSigStatusRetryCount = metrics.Register(getSigStatusRetryCount).(*prometheus.HistogramVec)
}

type Commitment struct {
	Commitment string `json:"commitment"`
}

var (
	CommitmentRecent = Commitment{Commitment: "recent"}
	CommitmentSingle = Commitment{Commitment: "single"}
	CommitmentRoot   = Commitment{Commitment: "root"}
	CommitmentMax    = Commitment{Commitment: "max"}
)

var (
	ErrNoAccountInfo     = errors.New("no account info")
	ErrSignatureNotFound = errors.New("signature not found")
	ErrBlockNotAvailable = errors.New("block not available")
)

// AccountInfo contains the Solana account information (not to be confused with a TokenAccount)
type AccountInfo struct {
	Data       []byte
	Owner      ed25519.PublicKey
	Lamports   uint64
	Executable bool
}

const (
	confirmationStatusProcessed = "processed"
	confirmationStatusConfirmed = "confirmed"
	confirmationStatusFinalized = "finalized"
)

type SignatureStatus struct {
	Slot        uint64
	ErrorResult *TransactionError

	// Confirmations will be nil if the transaction has been rooted.
	Confirmations      *int
	ConfirmationStatus string
}

func (s SignatureStatus) Confirmed() bool {
	if s.Finalized() {
		return true
	}

	if s.ConfirmationStatus == confirmationStatusConfirmed {
		return true
	}

	return *s.Confirmations >= 1
}

func (s SignatureStatus) Finalized() bool {
	return s.Confirmations == nil || s.ConfirmationStatus == confirmationStatusFinalized
}

type Block struct {
	Hash       []byte
	PrevHash   []byte
	ParentSlot uint64
	Slot       uint64

	Transactions []BlockTransaction
}

type BlockTransaction struct {
	Transaction Transaction
	Err         *TransactionError
}

type ConfirmedTransaction struct {
	Slot        uint64
	Transaction Transaction
	Err         *TransactionError
}

// Client provides an interaction with the Solana JSON RPC API.
//
// Reference: https://docs.solana.com/apps/jsonrpc-api
type Client interface {
	GetMinimumBalanceForRentExemption(size uint64) (lamports uint64, err error)
	GetSlot(Commitment) (uint64, error)
	GetRecentBlockhash() (Blockhash, error)
	GetBlockTime(block uint64) (time.Time, error)
	GetConfirmedBlock(slot uint64) (*Block, error)
	GetConfirmedBlocksWithLimit(start, limit uint64) ([]uint64, error)
	GetConfirmedTransaction(Signature) (ConfirmedTransaction, error)
	GetBalance(ed25519.PublicKey) (uint64, error)
	SimulateTransaction(Transaction) (*TransactionError, error)
	SubmitTransaction(Transaction, Commitment) (Signature, *SignatureStatus, error)
	GetAccountInfo(ed25519.PublicKey, Commitment) (AccountInfo, error)
	RequestAirdrop(ed25519.PublicKey, uint64, Commitment) (Signature, error)
	GetConfirmationStatus(Signature, Commitment) (bool, error)
	GetSignatureStatus(Signature, Commitment) (*SignatureStatus, error)
	GetSignatureStatuses([]Signature) ([]*SignatureStatus, error)
	GetTokenAccountsByOwner(owner, mint ed25519.PublicKey) ([]ed25519.PublicKey, error)
}

var (
	errRateLimited  = errors.New("rate limited")
	errServiceError = errors.New("service error")
)

type rpcResponse struct {
	Context struct {
		Slot int64 `json:"slot"`
	} `json:"context"`
	Value interface{} `json:"value"`
}

type client struct {
	log     *logrus.Entry
	client  jsonrpc.RPCClient
	retrier retry.Retrier

	blockMu   sync.RWMutex
	blockhash Blockhash
	lastWrite time.Time
}

// New returns a client using the specified endpoint.
func New(endpoint string) Client {
	return NewWithRPCOptions(endpoint, nil)
}

// NewWithRPCOptions returns a client configured with the specified RPC options.
func NewWithRPCOptions(endpoint string, opts *jsonrpc.RPCClientOpts) Client {
	return &client{
		log:    logrus.StandardLogger().WithField("type", "solana/client"),
		client: jsonrpc.NewClientWithOpts(endpoint, opts),
		retrier: retry.NewRetrier(
			retry.RetriableErrors(errRateLimited, errServiceError),
			retry.Limit(3),
			retry.BackoffWithJitter(backoff.BinaryExponential(time.Second), 10*time.Second, 0.1),
		),
	}
}

func (c *client) call(out interface{}, method string, params ...interface{}) error {
	start := time.Now()
	i, err := c.retrier.Retry(func() error {
		err := c.client.CallFor(out, method, params...)
		if err == nil {
			rpcCounterVec.WithLabelValues(method, "200").Inc()
			return nil
		}

		rpcErr, ok := err.(*jsonrpc.RPCError)
		if !ok {
			rpcCounterVec.WithLabelValues(method, "").Inc()
			return err
		}
		rpcCounterVec.WithLabelValues(method, strconv.Itoa(rpcErr.Code)).Inc()
		if rpcErr.Code == 429 {
			return errRateLimited
		}
		if rpcErr.Code >= 500 || rpcErr.Code == rpcNodeUnhealthyCode {
			return errServiceError
		}

		return err
	})
	rpcTimings.WithLabelValues(method).Observe(time.Since(start).Seconds())
	retryCount.WithLabelValues(method).Observe(float64(i))

	return err
}

func (c *client) GetMinimumBalanceForRentExemption(dataSize uint64) (lamports uint64, err error) {
	if err := c.call(&lamports, "getMinimumBalanceForRentExemption", dataSize); err != nil {
		return 0, errors.Wrapf(err, "failed to send request")
	}

	return lamports, nil
}

func (c *client) GetSlot(commitment Commitment) (slot uint64, err error) {
	// note: we have to wrap the commitment in an []interface{} otherwise the
	//       solana RPC node complains. Technically this is a violation of the
	//       JSON RPC v2.0 spec.
	if err := c.call(&slot, "getSlot", []interface{}{commitment}); err != nil {
		return 0, errors.Wrapf(err, "failed to send request")
	}

	return slot, nil
}

func (c *client) GetRecentBlockhash() (hash Blockhash, err error) {
	// To avoid having thrashing around a similar periodic interval, we
	// randomize when we refresh our block hash. This is mostly only a
	// concern when running a batch migrator with a _ton_ of goroutines.
	window := time.Duration(float64(2*time.Second) * (0.8 + rand.Float64()))

	c.blockMu.RLock()
	if time.Since(c.lastWrite) < window {
		hash = c.blockhash
	}
	c.blockMu.RUnlock()

	if hash != (Blockhash{}) {
		return hash, nil
	}

	type response struct {
		Value struct {
			Blockhash string `json:"blockhash"`
		} `json:"value"`
	}

	var resp response
	if err := c.call(&resp, "getRecentBlockhash"); err != nil {
		return hash, errors.Wrapf(err, "failed to send request")
	}

	hashBytes, err := base58.Decode(resp.Value.Blockhash)
	if err != nil {
		return hash, errors.Wrap(err, "invalid base58 encoded hash in response")
	}

	copy(hash[:], hashBytes)

	c.blockMu.Lock()
	c.blockhash = hash
	c.lastWrite = time.Now()
	c.blockMu.Unlock()

	return hash, nil
}

func (c *client) GetBlockTime(slot uint64) (time.Time, error) {
	var unixTs int64
	if err := c.call(&unixTs, "getBlockTime", slot); err != nil {
		jsonRPCErr, ok := err.(*jsonrpc.RPCError)
		if !ok {
			return time.Time{}, errors.Wrapf(err, "failed to send request")
		}

		if jsonRPCErr.Code == blockNotAvailableCode {
			return time.Time{}, ErrBlockNotAvailable
		}
	}

	return time.Unix(unixTs, 0), nil
}

func (c *client) GetConfirmedBlock(slot uint64) (block *Block, err error) {
	type rawBlock struct {
		Hash       string `json:"blockhash"` // Since this value is in base58, we can't []byte
		PrevHash   string `json:"previousBlockhash"`
		ParentSlot uint64 `json:"parentSlot"`

		RawTransactions []struct {
			Transaction []string `json:"transaction"` // [string,encoding]
			Meta        *struct {
				Err interface{} `json:"err"`
			} `json:"meta"`
		} `json:"transactions"`
	}

	var rb *rawBlock
	if err := c.call(&rb, "getConfirmedBlock", slot, "base64"); err != nil {
		return nil, err
	}

	// Not all slots contain a block, which manifests itself as having a nil block
	if rb == nil {
		return nil, nil
	}

	block = &Block{
		ParentSlot: rb.ParentSlot,
		Slot:       slot,
	}

	if block.Hash, err = base58.Decode(rb.Hash); err != nil {
		return nil, errors.Wrap(err, "invalid base58 encoding for hash")
	}
	if block.PrevHash, err = base58.Decode(rb.PrevHash); err != nil {
		return nil, errors.Wrapf(err, "invalid base58 encoding for prevHash: %s", rb.PrevHash)
	}

	for i, txn := range rb.RawTransactions {
		txnBytes, err := base64.StdEncoding.DecodeString(txn.Transaction[0])
		if err != nil {
			return nil, errors.Wrapf(err, "invalid base58 encoding for transaction %d", i)
		}

		var t Transaction
		if err := t.Unmarshal(txnBytes); err != nil {
			return nil, errors.Wrapf(err, "invalid bytes for transaction %d", i)
		}

		var txErr *TransactionError
		if txn.Meta != nil {
			txErr, err = ParseTransactionError(txn.Meta.Err)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse transaction meta")
			}
		}

		block.Transactions = append(block.Transactions, BlockTransaction{
			Transaction: t,
			Err:         txErr,
		})
	}

	return block, nil
}

func (c *client) GetConfirmedBlocksWithLimit(start, limit uint64) (slots []uint64, err error) {
	return slots, c.call(&slots, "getConfirmedBlocksWithLimit", start, limit)
}

func (c *client) GetConfirmedTransaction(sig Signature) (ConfirmedTransaction, error) {
	type rpcResponse struct {
		Slot        uint64   `json:"slot"`
		Transaction []string `json:"transaction"` // [val, encoding]
		Meta        *struct {
			Err interface{} `json:"err"`
		} `json:"meta"`
	}

	var resp *rpcResponse
	if err := c.call(&resp, "getConfirmedTransaction", base58.Encode(sig[:]), "base64"); err != nil {
		return ConfirmedTransaction{}, err
	}

	if resp == nil {
		return ConfirmedTransaction{}, ErrSignatureNotFound
	}

	txn := ConfirmedTransaction{
		Slot: resp.Slot,
	}

	var err error
	rawTxn, err := base64.StdEncoding.DecodeString(resp.Transaction[0])
	if err != nil {
		return txn, errors.Wrap(err, "failed to decode transaction")
	}
	if err := txn.Transaction.Unmarshal(rawTxn); err != nil {
		return txn, errors.Wrap(err, "failed to unmarshal transaction")
	}

	if resp.Meta != nil {
		txn.Err, err = ParseTransactionError(resp.Meta.Err)
		if err != nil {
			return txn, errors.Wrap(err, "failed to parse transaction result")
		}
	}

	return txn, nil
}

func (c *client) GetBalance(account ed25519.PublicKey) (uint64, error) {
	var resp rpcResponse
	if err := c.call(&resp, "getBalance", base58.Encode(account[:]), CommitmentRecent); err != nil {
		return 0, errors.Wrapf(err, "failed to send request")
	}

	if balance, ok := resp.Value.(float64); ok {
		return uint64(balance), nil
	}

	return 0, errors.Errorf("invalid value in response")
}

func (c *client) SimulateTransaction(txn Transaction) (*TransactionError, error) {
	type rpcResponse struct {
		Value struct {
			Error interface{} `json:"err"`
			Logs  []string    `json:"logs"`
		} `json:"value"`
	}

	var resp rpcResponse
	if err := c.call(&resp, "simulateTransaction", base58.Encode(txn.Marshal()), CommitmentSingle); err != nil {
		return nil, err
	}

	txErr, err := ParseTransactionError(resp.Value.Error)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse simulation error")
	}

	return txErr, nil
}

func (c *client) SubmitTransaction(txn Transaction, commitment Commitment) (Signature, *SignatureStatus, error) {
	sig := txn.Signatures[0]
	txnBytes := txn.Marshal()

	config := struct {
		SkipPreflight       bool   `json:"skipPreflight"`
		PreflightCommitment string `json:"preflightCommitment"`
	}{
		SkipPreflight:       false,
		PreflightCommitment: commitment.Commitment,
	}

	var sigStr string
	err := c.call(&sigStr, "sendTransaction", base58.Encode(txnBytes), config)
	if err != nil {
		jsonRPCErr, ok := err.(*jsonrpc.RPCError)
		if !ok {
			return sig, nil, errors.Wrapf(err, "failed to send request")
		}

		txResult, parseErr := ParseRPCError(jsonRPCErr)
		if parseErr != nil {
			return sig, nil, err
		}

		return sig, &SignatureStatus{ErrorResult: txResult}, nil
	}

	// todo(config): set this as a tunable option.
	//
	// Currently, max and root commitments take ~32 slots before they
	// register. To avoid spamming GetSignatureStatus(), we simply sleep
	// before attempting to poll. This saves a lot of HTTP requests under
	// the hood in this situation.
	//
	// Note: if we overshoot, it's latency performance hit, but still an
	//       overall performance gain. Most of these types will be batch
	//       or low volume tools.
	if commitment == CommitmentMax || commitment == CommitmentRoot {
		time.Sleep((32 / slotsPerSec) * time.Second)
	}

	status, err := c.GetSignatureStatus(txn.Signatures[0], commitment)
	return sig, status, err
}

func (c *client) GetAccountInfo(account ed25519.PublicKey, commitment Commitment) (accountInfo AccountInfo, err error) {
	type rpcResponse struct {
		Value *struct {
			Lamports   uint64   `json:"lamports"`
			Owner      string   `json:"owner"`
			Data       []string `json:"data"`
			Executable bool     `json:"executable"`
		} `json:"value"`
	}

	rpcConfig := struct {
		Commitment Commitment `json:"commitment"`
		Encoding   string     `json:"encoding"`
	}{
		Commitment: commitment,
		Encoding:   "base64",
	}

	var resp rpcResponse
	if err := c.call(&resp, "getAccountInfo", base58.Encode(account[:]), rpcConfig); err != nil {
		return accountInfo, errors.Wrap(err, "failed to send request")
	}

	if resp.Value == nil {
		return accountInfo, ErrNoAccountInfo
	}

	accountInfo.Owner, err = base58.Decode(resp.Value.Owner)
	if err != nil {
		return accountInfo, errors.Wrap(err, "invalid base58 encoded owner")
	}

	accountInfo.Data, err = base64.StdEncoding.DecodeString(resp.Value.Data[0])
	if err != nil {
		return accountInfo, errors.Wrap(err, "invalid base58 encoded data")
	}

	accountInfo.Lamports = resp.Value.Lamports
	accountInfo.Executable = resp.Value.Executable

	return accountInfo, nil
}

func (c *client) RequestAirdrop(account ed25519.PublicKey, lamports uint64, commitment Commitment) (Signature, error) {
	var sigStr string
	if err := c.call(&sigStr, "requestAirdrop", base58.Encode(account[:]), lamports, commitment); err != nil {
		return Signature{}, errors.Wrapf(err, "failed to send request")
	}

	sigBytes, err := base58.Decode(sigStr)
	if err != nil {
		return Signature{}, errors.Wrap(err, "invalid signature in response")
	}

	var sig Signature
	copy(sig[:], sigBytes)

	if sig == (Signature{}) {
		return Signature{}, errors.New("empty signature returned")
	}

	return sig, nil
}

func (c *client) GetConfirmationStatus(sig Signature, commitment Commitment) (bool, error) {
	type response struct {
		Value bool `json:"value"`
	}

	var resp response
	if err := c.call(&resp, "confirmTransaction", base58.Encode(sig[:]), commitment); err != nil {
		return false, err
	}

	return resp.Value, nil
}

func (c *client) GetSignatureStatus(sig Signature, commitment Commitment) (*SignatureStatus, error) {
	var s *SignatureStatus
	errConfirmationsNotReached := errors.New("confirmations not reached")
	start := time.Now()
	i, err := retry.Retry(
		func() error {
			statuses, err := c.GetSignatureStatuses([]Signature{sig})
			if err != nil {
				return err
			}

			s = statuses[0]
			if s == nil {
				return ErrSignatureNotFound
			}

			if s.ErrorResult != nil {
				return err
			}

			switch commitment {
			case CommitmentRecent:
				return nil
			case CommitmentSingle:
				if s.Confirmed() {
					return nil
				}
			case CommitmentMax, CommitmentRoot:
				if s.Finalized() {
					return nil
				}
			}

			return errConfirmationsNotReached
		},
		retry.RetriableErrors(ErrSignatureNotFound, errConfirmationsNotReached),
		retry.Limit(sigStatusPollLimit),
		retry.Backoff(backoff.Constant(PollRate), PollRate),
	)
	getSigStatusTimings.WithLabelValues(commitment.Commitment).Observe(time.Since(start).Seconds())
	getSigStatusRetryCount.WithLabelValues(commitment.Commitment).Observe(float64(i))

	return s, err
}

func (c *client) GetSignatureStatuses(sigs []Signature) ([]*SignatureStatus, error) {
	b58Sigs := make([]string, len(sigs))
	for i := range sigs {
		b58Sigs[i] = base58.Encode(sigs[i][:])
	}

	req := struct {
		SearchTransactionHistory bool `json:"searchTransactionHistory"`
	}{
		SearchTransactionHistory: false,
	}

	type signatureStatus struct {
		Slot               uint64          `json:"slot"`
		Confirmations      *int            `json:"confirmations"`
		ConfirmationStatus string          `json:"confirmationStatus"`
		Err                json.RawMessage `json:"err"`
	}

	type rpcResp struct {
		Context struct {
			Slot int `json:"slot"`
		} `json:"context"`
		Value []*signatureStatus `json:"value"`
	}

	var resp rpcResp
	if err := c.call(&resp, "getSignatureStatuses", b58Sigs, req); err != nil {
		return nil, err
	}

	statuses := make([]*SignatureStatus, len(sigs))
	for i, v := range resp.Value {
		if v == nil {
			continue
		}

		statuses[i] = &SignatureStatus{}
		statuses[i].Confirmations = v.Confirmations
		statuses[i].ConfirmationStatus = v.ConfirmationStatus
		statuses[i].Slot = v.Slot

		if len(v.Err) > 0 {
			var txError interface{}
			err := json.NewDecoder(bytes.NewBuffer(v.Err)).Decode(&txError)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse transaction result")
			}

			statuses[i].ErrorResult, err = ParseTransactionError(txError)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse transaction result")
			}
		}
	}

	return statuses, nil
}

func (c *client) GetTokenAccountsByOwner(owner, mint ed25519.PublicKey) ([]ed25519.PublicKey, error) {
	mintObject := struct {
		Mint string `json:"mint"`
	}{
		Mint: base58.Encode(mint),
	}
	config := struct {
		Encoding   string `json:"encoding"`
		Commitment Commitment
	}{
		Encoding:   "base64",
		Commitment: CommitmentSingle,
	}

	var resp struct {
		Value []struct {
			PubKey string `json:"pubkey"`
		} `json:"value"`
	}
	if err := c.call(&resp, "getTokenAccountsByOwner", base58.Encode(owner), mintObject, config); err != nil {
		return nil, err
	}

	keys := make([]ed25519.PublicKey, len(resp.Value))
	for i := range resp.Value {
		var err error
		keys[i], err = base58.Decode(resp.Value[i].PubKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode token account public key")
		}
	}

	return keys, nil
}
