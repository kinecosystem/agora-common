package solana

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/ybbus/jsonrpc"
)

// TransactionErrorKey is the string key returned in a transaction error.
//
// Source: https://github.com/solana-labs/solana/blob/master/sdk/src/transaction.rs#L22
type TransactionErrorKey string

const (
	TransactionErrorAccountInUse               TransactionErrorKey = "AccountInUse"
	TransactionErrorAccountLoadedTwice         TransactionErrorKey = "AccountLoadedTwice"
	TransactionErrorAccountNotFound            TransactionErrorKey = "AccountNotFound"
	TransactionErrorProgramAccountNotFound     TransactionErrorKey = "ProgramAccountNotFound"
	TransactionErrorInsufficientFundsForFee    TransactionErrorKey = "InsufficientFundsForFee"
	TransactionErrorInvalidAccountForFee       TransactionErrorKey = "InvalidAccountForFee"
	TransactionErrorDuplicateSignature         TransactionErrorKey = "DuplicateSignature"
	TransactionErrorBlockhashNotFound          TransactionErrorKey = "BlockhashNotFound"
	TransactionErrorInstructionError           TransactionErrorKey = "InstructionError"
	TransactionErrorCallChainTooDeep           TransactionErrorKey = "CallChainTooDeep"
	TransactionErrorMissingSignatureForFee     TransactionErrorKey = "MissingSignatureForFee"
	TransactionErrorInvalidAccountIndex        TransactionErrorKey = "InvalidAccountIndex"
	TransactionErrorSignatureFailure           TransactionErrorKey = "SignatureFailure"
	TransactionErrorInvalidProgramForExecution TransactionErrorKey = "InvalidProgramForExecution"
	TransactionErrorSanitizeFailure            TransactionErrorKey = "SanitizeFailure"
	TransactionErrorClusterMaintenance         TransactionErrorKey = "ClusterMaintenance"
)

// InstructionErrorKey is the string keys returned in an instruction error.
//
// Source: https://github.com/solana-labs/solana/blob/master/sdk/src/instruction.rs#L11
type InstructionErrorKey string

const (
	InstructionErrorGenericError                   InstructionErrorKey = "GenericError"
	InstructionErrorInvalidArgument                InstructionErrorKey = "InvalidArgument"
	InstructionErrorInvalidInstructionData         InstructionErrorKey = "InvalidInstructionData"
	InstructionErrorInvalidAccountData             InstructionErrorKey = "InvalidAccountData"
	InstructionErrorAccountDataTooSmall            InstructionErrorKey = "AccountDataTooSmall"
	InstructionErrorInsufficientFunds              InstructionErrorKey = "InsufficientFunds"
	InstructionErrorIncorrectProgramID             InstructionErrorKey = "IncorrectProgramId"
	InstructionErrorMissingRequiredSignature       InstructionErrorKey = "MissingRequiredSignature"
	InstructionErrorAccountAlreadyInitialized      InstructionErrorKey = "AccountAlreadyInitialized"
	InstructionErrorUninitializedAccount           InstructionErrorKey = "UninitializedAccount"
	InstructionErrorUnbalancedInstruction          InstructionErrorKey = "UnbalancedInstruction"
	InstructionErrorModifiedProgramID              InstructionErrorKey = "ModifiedProgramId"
	InstructionErrorExternalAccountLamportSpend    InstructionErrorKey = "ExternalAccountLamportSpend"
	InstructionErrorExternalAccountDataModified    InstructionErrorKey = "ExternalAccountDataModified"
	InstructionErrorReadonlyLamportChange          InstructionErrorKey = "ReadonlyLamportChange"
	InstructionErrorReadonlyDataModified           InstructionErrorKey = "ReadonlyDataModified"
	InstructionErrorDuplicateAccountIndex          InstructionErrorKey = "DuplicateAccountIndex"
	InstructionErrorExecutableModified             InstructionErrorKey = "ExecutableModified"
	InstructionErrorRentEpochModified              InstructionErrorKey = "RentEpochModified"
	InstructionErrorNotEnoughAccountKeys           InstructionErrorKey = "NotEnoughAccountKeys"
	InstructionErrorAccountDataSizeChanged         InstructionErrorKey = "AccountDataSizeChanged"
	InstructionErrorAccountNotExecutable           InstructionErrorKey = "AccountNotExecutable"
	InstructionErrorAccountBorrowFailed            InstructionErrorKey = "AccountBorrowFailed"
	InstructionErrorAccountBorrowOutstanding       InstructionErrorKey = "AccountBorrowOutstanding"
	InstructionErrorDuplicateAccountOutOfSync      InstructionErrorKey = "DuplicateAccountOutOfSync"
	InstructionErrorCustom                         InstructionErrorKey = "Custom"
	InstructionErrorInvalidError                   InstructionErrorKey = "InvalidError"
	InstructionErrorExecutableDataModified         InstructionErrorKey = "ExecutableDataModified"
	InstructionErrorExecutableLamportChange        InstructionErrorKey = "ExecutableLamportChange"
	InstructionErrorExecutableAccountNotRentExempt InstructionErrorKey = "ExecutableAccountNotRentExempt"
	InstructionErrorUnsupportedProgramID           InstructionErrorKey = "UnsupportedProgramId"
	InstructionErrorCallDepth                      InstructionErrorKey = "CallDepth"
	InstructionErrorMissingAccount                 InstructionErrorKey = "MissingAccount"
	InstructionErrorReentrancyNotAllowed           InstructionErrorKey = "ReentrancyNotAllowed"
	InstructionErrorMaxSeedLengthExceeded          InstructionErrorKey = "MaxSeedLengthExceeded"
	InstructionErrorInvalidSeeds                   InstructionErrorKey = "InvalidSeeds"
	InstructionErrorInvalidRealloc                 InstructionErrorKey = "InvalidRealloc"
)

// CustomError is the numerical error returned by a non-system program.
type CustomError int

func (c CustomError) Error() string {
	return fmt.Sprintf("custom program error: %x", int(c))
}

// InstructionError indicates an instruction returned an error in a transaction.
type InstructionError struct {
	Index int
	Err   error
}

func parseInstructionError(v interface{}) (e InstructionError, err error) {
	values, ok := v.([]interface{})
	if !ok {
		return e, errors.New("unexpected instruction error format")
	}

	if len(values) != 2 {
		return e, errors.Errorf("too many entries in InstructionError tuple: %d", len(values))
	}

	e.Index, err = parseJSONNumber(values[0])
	if err != nil {
		return e, err
	}

	switch t := values[1].(type) {
	case string:
		e.Err = errors.New(t)
	case map[string]interface{}:
		if len(t) != 1 {
			e.Err = errors.New("unhandled InstructionError")
			return e, errors.Errorf("invalid instruction result size: %d", len(t))
		}

		var k string
		var v interface{}
		for k, v = range t {
		}

		if k != "Custom" {
			e.Err = errors.New(k)
			break
		}

		code, err := parseJSONNumber(v)
		if err != nil {
			e.Err = errors.New("unhandled CustomError")
			break
		}

		e.Err = CustomError(code)
	}

	return e, nil
}

func (i InstructionError) Error() string {
	return fmt.Sprintf("Error processing Instruction %d: %v", i.Index, i.Err)
}

func (i InstructionError) ErrorKey() InstructionErrorKey {
	if i.Err == nil {
		return ""
	}

	if i.CustomError() != nil {
		return InstructionErrorCustom
	}

	return InstructionErrorKey(i.Err.Error())
}

func (i InstructionError) JSONString() string {
	if e, ok := i.Err.(CustomError); ok {
		return fmt.Sprintf(`[%d, {"%s": %d}]`, i.Index, InstructionErrorCustom, e)
	}

	return fmt.Sprintf(`[%d, "%s"]`, i.Index, i.Err.Error())
}

func (i InstructionError) CustomError() *CustomError {
	ce, ok := i.Err.(CustomError)
	if ok {
		return &ce
	}

	return nil
}

// TransactionError contains the transaction error details.
type TransactionError struct {
	transactionError error
	instructionError *InstructionError
	raw              interface{}
}

// ParseRPCError parses the jsonrpc.RPCError returned from a method.
func ParseRPCError(err *jsonrpc.RPCError) (*TransactionError, error) {
	if err == nil {
		return nil, nil
	}

	i := err.Data
	data, ok := i.(map[string]interface{})
	if !ok {
		return nil, errors.New("expected map type")
	}

	if txErr, ok := data["err"]; ok && txErr != nil {
		return ParseTransactionError(txErr)
	}

	return nil, nil
}

// ParseTransactionError parses the JSON error returned from the "err" field in various
// RPC methods and fields.
func ParseTransactionError(raw interface{}) (*TransactionError, error) {
	if raw == nil {
		return nil, nil
	}

	switch t := raw.(type) {
	case string:
		return &TransactionError{
			transactionError: errors.New(t),
			raw:              raw,
		}, nil
	case map[string]interface{}:
		if len(t) != 1 {
			return &TransactionError{
				transactionError: errors.New("unhandled transaction error"),
				raw:              raw,
			}, errors.Errorf("invalid transaction result size: %d", len(t))
		}

		var k string
		var v interface{}
		for k, v = range t {
		}

		if k != "InstructionError" {
			return &TransactionError{
				transactionError: errors.New(k),
				raw:              raw,
			}, nil
		}

		instructionErr, err := parseInstructionError(v)
		if err != nil {
			return &TransactionError{
				transactionError: errors.New("unhandled transaction error"),
				raw:              raw,
			}, errors.Wrap(err, "failed to parse instruction error")
		}

		return &TransactionError{
			transactionError: errors.New(string(TransactionErrorInstructionError)),
			instructionError: &instructionErr,
			raw:              raw,
		}, nil
	default:
		return nil, errors.New("unhandled error type")
	}
}

func NewTransactionError(key TransactionErrorKey) *TransactionError {
	return &TransactionError{
		transactionError: errors.New(string(key)),
		raw:              string(key),
	}
}

func TransactionErrorFromInstructionError(err *InstructionError) (*TransactionError, error) {
	var raw interface{}
	if err := json.Unmarshal([]byte(err.JSONString()), &raw); err != nil {
		return nil, errors.Wrap(err, "failed to generate raw value")
	}

	return &TransactionError{
		transactionError: errors.New(string(TransactionErrorInstructionError)),
		instructionError: err,
		raw: map[string]interface{}{
			string(TransactionErrorInstructionError): raw,
		},
	}, nil
}

func (t TransactionError) Error() string {
	if t.instructionError != nil {
		return t.instructionError.Error()
	}

	if t.transactionError != nil {
		return t.transactionError.Error()
	}

	return ""
}

func (t TransactionError) ErrorKey() TransactionErrorKey {
	if t.transactionError == nil {
		return ""
	}

	return TransactionErrorKey(t.transactionError.Error())
}

func (t TransactionError) InstructionError() *InstructionError {
	return t.instructionError
}

func (t TransactionError) JSONString() (string, error) {
	b, err := json.Marshal(t.raw)
	return string(b), err
}

func parseJSONNumber(v interface{}) (int, error) {
	if num, ok := v.(json.Number); ok {
		index, err := num.Int64()
		if err != nil {
			return 0, errors.Errorf("non int64 value in InstructionError tuple: %v", v)
		}
		return int(index), nil
	} else if indexString, ok := v.(string); ok {
		index, err := strconv.ParseInt(indexString, 10, 64)
		if err != nil {
			return 0, errors.Errorf("non numeric value in InstructionError tuple: %v", v)
		}
		return int(index), nil
	} else if indexFloat, ok := v.(float64); ok {
		return int(indexFloat), nil
	}

	return 0, errors.Errorf("non numeric value in InstructionError tuple: %v", v)
}
