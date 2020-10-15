package system

import (
	"crypto/ed25519"

	"github.com/mr-tron/base58/base58"
)

// RentSysVar points to the system variable "Rent"
//
// Source: https://github.com/solana-labs/solana/blob/f02a78d8fff2dd7297dc6ce6eb5a68a3002f5359/sdk/src/sysvar/rent.rs#L11
var RentSysVar ed25519.PublicKey

func init() {
	var err error

	RentSysVar, err = base58.Decode("SysvarRent111111111111111111111111111111111")
	if err != nil {
		panic(err)
	}
}
