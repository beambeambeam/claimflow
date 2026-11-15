package money_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/beambeambeam/claimflow/apps/api/internal/money"
)

// F02-TEST-05 (F02-AC-04): only strings of 2 decimals decode. 0.00, 0.01, 20000.00, and 20000.01 decode (the range is BR-05,
// checked by the claim rules); 1.005, 1.5, -1.00, and abc fail.
func TestF02TEST05SatangDecodesOnlyTwoDecimalStrings(t *testing.T) {
	tests := []struct {
		input string
		want  money.Satang
		ok    bool
	}{
		{`"0.00"`, 0, true},
		{`"0.01"`, 1, true},
		{`"20000.00"`, 2000000, true},
		{`"20000.01"`, 2000001, true},
		{`"999999.99"`, 99999999, true},
		{`"1.005"`, 0, false},
		{`"1.5"`, 0, false},
		{`"-1.00"`, 0, false},
		{`"abc"`, 0, false},
		{`""`, 0, false},
		{`"1234567.00"`, 0, false},
		{`"1,250.00"`, 0, false},
		{`1.00`, 0, false},
		{`null`, 0, false},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			var got money.Satang

			err := json.Unmarshal([]byte(test.input), &got)
			if test.ok != (err == nil) {
				t.Fatalf("decode %s: err = %v, want ok = %t", test.input, err, test.ok)
			}

			if got != test.want {
				t.Fatalf("decode %s = %d, want %d", test.input, got, test.want)
			}
		})
	}
}

func TestSatangEncodesTwoDecimalString(t *testing.T) {
	tests := map[money.Satang]string{0: `"0.00"`, 1: `"0.01"`, 125000: `"1250.00"`, 5000001: `"50000.01"`}

	for satang, want := range tests {
		got, err := json.Marshal(satang)
		if err != nil || string(got) != want {
			t.Fatalf("encode %d = %s, %v; want %s", satang, got, err, want)
		}

		if satang.String() != want[1:len(want)-1] {
			t.Fatalf("String(%d) = %q, want %q", satang, satang.String(), want[1:len(want)-1])
		}
	}
}

func TestSatangEncodeRejectsNegative(t *testing.T) {
	if _, err := json.Marshal(money.Satang(-1)); !errors.Is(err, money.ErrNegative) {
		t.Fatalf("encode -1: err = %v, want ErrNegative", err)
	}
}
