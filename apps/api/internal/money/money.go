// Package money holds the Satang type that every amount in the API uses (SDS §2.3).
package money

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

const satangPerBaht = 100

// ErrFormat means a string is not an amount with exactly 2 decimal places, such as "1250.00".
var ErrFormat = errors.New("money: want 1 to 6 digits, a dot, and 2 decimals, such as 1250.00")

// ErrNegative means a Satang below zero has no string form.
var ErrNegative = errors.New("money: a negative amount has no string form")

// amountPattern is the Money schema of the contract.
var amountPattern = regexp.MustCompile(`^\d{1,6}\.\d{2}$`)

// Satang is an amount of money in satang (1 THB = 100 satang). Floats are never used for money.
type Satang int64

// Parse reads an amount written as `^\d{1,6}\.\d{2}$`. Anything else, including "1.005", fails.
func Parse(text string) (Satang, error) {
	if !amountPattern.MatchString(text) {
		return 0, ErrFormat
	}

	// The pattern allows only digits and one dot, so removing the dot leaves an integer that fits an int64.
	digits := text[:len(text)-3] + text[len(text)-2:]
	satang, _ := strconv.ParseInt(digits, 10, 64)

	return Satang(satang), nil
}

// String returns the amount with exactly 2 decimal places, such as "1250.00". A negative amount has no string form.
func (s Satang) String() string {
	return fmt.Sprintf("%d.%02d", s/satangPerBaht, s%satangPerBaht)
}

// MarshalJSON encodes the amount as a JSON string of 2 decimals.
func (s Satang) MarshalJSON() ([]byte, error) {
	if s < 0 {
		return nil, ErrNegative
	}

	return []byte(`"` + s.String() + `"`), nil
}

// UnmarshalJSON decodes a JSON string that Parse accepts. It rejects every other JSON value, so no handler parses money.
func (s *Satang) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return ErrFormat
	}

	satang, err := Parse(text)
	if err != nil {
		return err
	}

	*s = satang

	return nil
}
