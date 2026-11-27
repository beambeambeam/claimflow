package daterange

import (
	"slices"
	"testing"
	"time"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
)

func date(month time.Month, day int) *time.Time {
	return new(time.Date(2026, month, day, 0, 0, 0, 0, time.UTC))
}

// A valid date is its calendar date at midnight UTC, and a missing one stays nil.
func TestParseReadsDates(t *testing.T) {
	for _, test := range []struct {
		name     string
		from, to *string
		first    *time.Time
		last     *time.Time
	}{
		{"no dates", nil, nil, nil, nil},
		{"from only", new("2026-11-01"), nil, date(time.November, 1), nil},
		{"to only", nil, new("2026-11-30"), nil, date(time.November, 30)},
		{"both, the same day", new("2026-11-05"), new("2026-11-05"), date(time.November, 5), date(time.November, 5)},
	} {
		t.Run(test.name, func(t *testing.T) {
			first, last, problems := Parse(test.from, test.to)
			if len(problems) > 0 {
				t.Fatalf("problems = %v, want none", problems)
			}

			if !sameDate(first, test.first) || !sameDate(last, test.last) {
				t.Fatalf("dates = %v, %v; want %v, %v", first, last, test.first, test.last)
			}
		})
	}
}

func sameDate(a, b *time.Time) bool {
	return (a == nil) == (b == nil) && (a == nil || a.Equal(*b))
}

// Each bad parameter is one problem, in parameter order, and a range error is on `to`.
func TestParseProblems(t *testing.T) {
	for _, test := range []struct {
		name     string
		from, to *string
		want     []api.FieldError
	}{
		{"not a date", new("Oct"), nil, []api.FieldError{{Field: "from", Rule: "format", Message: BadDateMessage}}},
		{"empty", nil, new(""), []api.FieldError{{Field: "to", Rule: "format", Message: BadDateMessage}}},
		{"day that does not exist", new("2026-02-30"), nil, []api.FieldError{{Field: "from", Rule: "format", Message: BadDateMessage}}},
		{"both bad", new("05/11/2026"), new("2026-11-5"), []api.FieldError{
			{Field: "from", Rule: "format", Message: BadDateMessage}, {Field: "to", Rule: "format", Message: BadDateMessage},
		}},
		{"backwards", new("2026-11-10"), new("2026-11-01"), []api.FieldError{{Field: "to", Rule: "range", Message: BadRangeMessage}}},
		{"one bad date is not also a bad range", new("nope"), new("2026-11-01"), []api.FieldError{
			{Field: "from", Rule: "format", Message: BadDateMessage},
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			first, last, problems := Parse(test.from, test.to)

			if first != nil || last != nil {
				t.Fatalf("dates = %v, %v; want none with a problem", first, last)
			}

			if !slices.Equal(problems, test.want) {
				t.Fatalf("problems = %+v, want %+v", problems, test.want)
			}
		})
	}
}

// Midnight is the start of the Bangkok day of the date, which is 17:00 UTC the day before.
func TestMidnight(t *testing.T) {
	got := Midnight(*date(time.December, 31), 1)
	want := time.Date(2026, time.December, 31, 17, 0, 0, 0, time.UTC)

	if !got.Equal(want) || got.Location() != Bangkok {
		t.Fatalf("Midnight = %v, want %v in Asia/Bangkok", got, want)
	}
}

// Bounds are half-open: 00:00 on the first date up to, and not including, 00:00 on the day after the last.
func TestBounds(t *testing.T) {
	for _, test := range []struct {
		name        string
		first, last *time.Time
		start, end  *time.Time
	}{
		{"open", nil, nil, nil, nil},
		{"from only", date(time.November, 1), nil, new(time.Date(2026, time.October, 31, 17, 0, 0, 0, time.UTC)), nil},
		{"to only, the last day of a month", nil, date(time.October, 31), nil, new(time.Date(2026, time.October, 31, 17, 0, 0, 0, time.UTC))},
		{
			"one day", date(time.December, 31), date(time.December, 31),
			new(time.Date(2026, time.December, 30, 17, 0, 0, 0, time.UTC)), new(time.Date(2026, time.December, 31, 17, 0, 0, 0, time.UTC)),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			start, end := Bounds(test.first, test.last)

			if !sameDate(start, test.start) || !sameDate(end, test.end) {
				t.Fatalf("Bounds = %v, %v; want %v, %v", start, end, test.start, test.end)
			}
		})
	}
}
