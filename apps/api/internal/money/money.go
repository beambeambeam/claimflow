// Package money holds the Satang type that every amount in the API uses (SDS §2.3).
package money

import (
	"errors"
)

// Satang is an amount of money in satang (1 THB = 100 satang). Floats are never used for money.
type Satang int64

// ErrFormat means a string is not an amount with exactly 2 decimal places, such as "1250.00".
var ErrFormat = errors.New("money: want 1 to 6 digits, a dot, and 2 decimals, such as 1250.00")

// ErrNegative means a Satang below zero has no string form.
var ErrNegative = errors.New("money: a negative amount has no string form")

// Parse reads an amount written as `^\d{1,6}\.\d{2}$`. Anything else, including "1.005", fails.
func Parse(text string) (Satang, error) {
	return 0, ErrFormat
}

// String returns the amount with exactly 2 decimal places, such as "1250.00".
func (s Satang) String() string {
	return ""
}

// MarshalJSON encodes the amount as a JSON string of 2 decimals.
func (s Satang) MarshalJSON() ([]byte, error) {
	return nil, errors.New("not implemented")
}

// UnmarshalJSON decodes a JSON string that Parse accepts. It rejects every other JSON value, so no handler parses money.
func (s *Satang) UnmarshalJSON(data []byte) error {
	return ErrFormat
}
