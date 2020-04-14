package metrics

import (
	"unicode"

	"github.com/pkg/errors"
)

func isValidMetricName(name string) error {
	if len(name) == 0 {
		return errors.New("name cannot be empty")
	}

	if !unicode.IsLetter(rune(name[0])) {
		return errors.New("first character must be a letter")
	}

	return nil
}
