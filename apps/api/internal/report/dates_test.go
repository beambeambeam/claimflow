package report

import (
	"slices"
	"testing"
	"time"
)

// ParseRange turns inclusive Bangkok dates into the half-open range of F-09 §1.8: 00:00 on `from` up to, and not
// including, 00:00 on the day after `to`. A missing date leaves its side open.
func TestParseRangeBounds(t *testing.T) {
	for _, test := range []struct {
		name     string
		from, to *string
		want     Range
	}{
		{"no dates", nil, nil, Range{}},
		{"from only", new("2026-11-01"), nil, Range{From: new(time.Date(2026, time.November, 1, 0, 0, 0, 0, ict))}},
		{"to only, the last day of a month", nil, new("2026-10-31"), Range{To: new(time.Date(2026, time.November, 1, 0, 0, 0, 0, ict))}},
		{"one day", new("2026-12-31"), new("2026-12-31"), Range{
			From: new(time.Date(2026, time.December, 31, 0, 0, 0, 0, ict)),
			To:   new(time.Date(2027, time.January, 1, 0, 0, 0, 0, ict)),
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, problems := ParseRange(test.from, test.to)
			if len(problems) > 0 {
				t.Fatalf("problems = %v, want none", problems)
			}

			if !sameBound(got.From, test.want.From) || !sameBound(got.To, test.want.To) {
				t.Fatalf("range = %v to %v, want %v to %v", got.From, got.To, test.want.From, test.want.To)
			}
		})
	}
}

func sameBound(a, b *time.Time) bool {
	return (a == nil) == (b == nil) && (a == nil || a.Equal(*b))
}

// Each bad parameter is one problem with the F-09 §1.3 message, and both are reported together.
func TestParseRangeProblems(t *testing.T) {
	for _, test := range []struct {
		name     string
		from, to *string
		want     []string
	}{
		{"not a date", new("Oct"), nil, []string{"from: Use a date in the form YYYY-MM-DD."}},
		{"empty", nil, new(""), []string{"to: Use a date in the form YYYY-MM-DD."}},
		{"day that does not exist", new("2026-02-30"), nil, []string{"from: Use a date in the form YYYY-MM-DD."}},
		{"other format", new("05/11/2026"), new("2026-11-5"), []string{
			"from: Use a date in the form YYYY-MM-DD.", "to: Use a date in the form YYYY-MM-DD.",
		}},
		{"backwards", new("2026-11-10"), new("2026-11-01"), []string{"to: The end date must be on or after the start date."}},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, problems := ParseRange(test.from, test.to)

			got := make([]string, 0, len(problems))
			for _, problem := range problems {
				got = append(got, problem.Field+": "+problem.Message)
			}

			if !slices.Equal(got, test.want) {
				t.Fatalf("problems = %q, want %q", got, test.want)
			}
		})
	}
}
