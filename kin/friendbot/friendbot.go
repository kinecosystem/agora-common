package friendbot

import (
	"errors"
	"fmt"
	"net/http"
)

const (
	// friendbotURL is the base URL for requests to the friendbot service.
	friendbotURL = "http://friendbot-testnet.kininfrastructure.com"

	// quarksPerKin is the number of quarks in one kin.
	quarksPerKin = 100000

	// The minimum amount that can be requested from friendbot to fund an account, in quarks.
	minFundQuarks = 1

	// The maximum amount that can be requested from friendbot, in quarks. Equivalent to 10000 kin.
	maxQuarks = 1000000000
)

var (
	// ErrInvalidCreateAmount occurs when the amount for a friendbot create account request is out of bounds. The bounds
	// are defined in the friendbot service (see https://github.com/kinecosystem/friendbot/blob/master/src/routes.ts#L51).
	ErrInvalidCreateAmount = errors.New("friendbot create account request quark amount must be in the range [0, 1000000000]")

	// ErrInvalidFundAmount occurs when the amount for a friendbot fund account request is out of bounds. The bounds
	// defined in the friendbot service (https://github.com/kinecosystem/friendbot/blob/master/src/routes.ts#L51) allow
	// fund requests for 0 quarks, but the blockchain transaction will fail. Therefore, a min amount of 1 is required.
	ErrInvalidFundAmount = errors.New("friendbot fund account request quark amount must be in the range [1, 1000000000]")
)

// friendbotResult represents a successful result from a friendbot request.
type friendbotResult struct {
	// Hash is the hash of the successful transaction submitted by the friendbot service
	Hash string `json:"hash"`
}

// CreateAccount creates a new account on the test Kin network with the requested starting balance.
//
// friendbot accepts an amount in kin, but parses it as a float and throws an internal error if the amount has more
// than 5 decimal places, so quarks are used here to avoid input errors.
func CreateAccount(address string, quarkAmount uint) (hash string, err error) {
	if quarkAmount > maxQuarks {
		return "", ErrInvalidCreateAmount
	}

	url := fmt.Sprintf("%s?addr=%s&amount=%d.%d", friendbotURL, address, quarkAmount/quarksPerKin, quarkAmount%quarksPerKin)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	result := &friendbotResult{}
	err = decodeResponse(resp, result)
	if err != nil {
		return "", err
	}

	return result.Hash, nil
}

// FundAccount funds an existing account on the test Kin network with the requested amount.
func FundAccount(address string, quarkAmount uint) (hash string, err error) {
	if quarkAmount < minFundQuarks || quarkAmount > maxQuarks {
		return "", ErrInvalidFundAmount
	}

	url := fmt.Sprintf("%s/fund?addr=%s&amount=%d.%d", friendbotURL, address, quarkAmount/quarksPerKin, quarkAmount%quarksPerKin)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	result := &friendbotResult{}
	err = decodeResponse(resp, result)
	if err != nil {
		return "", err
	}

	return result.Hash, nil
}
