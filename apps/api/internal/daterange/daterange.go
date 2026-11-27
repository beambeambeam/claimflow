// Package daterange is the Asia/Bangkok date range of a request: the optional `from` and `to` query dates of the All
// Claims filters (F-09) and of the decision summary (F-10). Both features read the range here, so their messages and
// bounds cannot drift apart (SDS §1.4, rule 2).
package daterange

import (
	"time"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
)

// Bangkok is the calendar of every date in the API (SDS §2.3). Asia/Bangkok has no daylight saving, so a fixed offset
// is the same as the zone and needs no tz database.
var Bangkok = time.FixedZone("Asia/Bangkok", 7*60*60)

// The 400 messages of a date range (F-09 §1.3).
const (
	BadDateMessage  = "Use a date in the form YYYY-MM-DD."
	BadRangeMessage = "The end date must be on or after the start date."
)

// Parse reads the optional `from` and `to` query dates, each YYYY-MM-DD and inclusive. A nil value is a side left open.
// It returns each date as midnight UTC of that calendar date, or one FieldError per bad parameter: `from` and `to` that
// are not calendar dates, then `to` when `from` is after it. The dates are nil when there is a problem with either.
func Parse(from, to *string) (first, last *time.Time, problems []api.FieldError) {
	for _, date := range []struct {
		field string
		value *string
		bound **time.Time
	}{{"from", from, &first}, {"to", to, &last}} {
		if date.value == nil {
			continue
		}

		day, err := time.Parse(time.DateOnly, *date.value)
		if err != nil {
			problems = append(problems, api.FieldError{Field: date.field, Rule: "format", Message: BadDateMessage})

			continue
		}

		*date.bound = &day
	}

	if first != nil && last != nil && first.After(*last) {
		problems = append(problems, api.FieldError{Field: "to", Rule: "range", Message: BadRangeMessage})
	}

	if len(problems) > 0 {
		return nil, nil, problems
	}

	return first, last, nil
}

// Midnight returns the start, in Asia/Bangkok, of the calendar date of date plus days.
func Midnight(date time.Time, days int) time.Time {
	year, month, day := date.Date()

	return time.Date(year, month, day+days, 0, 0, 0, 0, Bangkok)
}

// Bounds returns the half-open interval of instants that holds the dates from first to last whole (F-09 §1.8): the
// start of first, and the start of the day after last. A nil date leaves its side open.
func Bounds(first, last *time.Time) (start, end *time.Time) {
	if first != nil {
		start = new(Midnight(*first, 0))
	}

	if last != nil {
		end = new(Midnight(*last, 1))
	}

	return start, end
}
