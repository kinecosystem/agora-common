package kin

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// ToQuarks converts a string representation of kin
// the quark value.
//
// An error is returned if the value string is invalid, or
// it cannot be accurately represented as quarks. For example,
// a value smaller than quarks, or a value _far_ greater than
// the supply.
func ToQuarks(val string) (int64, error) {
	parts := strings.Split(val, ".")
	if len(parts) > 2 {
		return 0, errors.New("invalid kin value")
	}

	if len(parts[0]) > 14 {
		return 0, errors.New("value cannot be represented")
	}

	kin, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}

	var quarks uint64
	if len(parts) == 2 {
		if len(parts[1]) > 5 {
			return 0, errors.New("value cannot be represented")
		}

		padded := fmt.Sprintf("%s%s", parts[1], strings.Repeat("0", 5-len(parts[1])))
		quarks, err = strconv.ParseUint(padded, 10, 64)
		if err != nil {
			return 0, errors.Wrap(err, "invalid decimal component")
		}
	}

	return kin*1e5 + int64(quarks), nil
}

// MustToQuarks calls ToQuarks, panicking if there's an error.
//
// This should only be used if you know for sure this will not panic.
func MustToQuarks(val string) int64 {
	result, err := ToQuarks(val)
	if err != nil {
		panic(err)
	}

	return result
}

// FromQuarks converts an int64 amount of quarks to the
// string representation of kin.
func FromQuarks(amount int64) string {
	if amount < 1e5 {
		return fmt.Sprintf("0.%05d", amount)
	}

	return fmt.Sprintf("%d.%05d", amount/1e5, amount%1e5)
}
