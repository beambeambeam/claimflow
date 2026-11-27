package report

import (
	"time"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
)

// bangkok is the calendar of every date in the API (SDS §2.3). Asia/Bangkok has no daylight saving, so a fixed offset
// is the same as the zone and needs no tz database.
var bangkok = time.UTC

// The 400 messages of a date range are the F-09 §1.3 messages, so the Summary and the All Claims filters read alike.
const (
	badDateMessage  = "Use a date in the form YYYY-MM-DD."
	badRangeMessage = "The end date must be on or after the start date."
)

// ParseRange reads the optional `from` and `to` query dates (YYYY-MM-DD, both inclusive, in Asia/Bangkok) as a half-open
// Range: from 00:00 on `from` to 00:00 on the day after `to`, the Bangkok-midnight bounds of F-09 §1.8. A nil date
// leaves its side open. A date that is not a calendar date, or `from` after `to`, returns one FieldError per bad
// parameter and no Range.
func ParseRange(from, to *string) (Range, []api.FieldError) {
	var (
		first, last *time.Time
		problems    []api.FieldError
	)

	for _, date := range []struct {
		field string
		value *string
		bound **time.Time
	}{{"from", from, &first}, {"to", to, &last}} {
		if date.value == nil {
			continue
		}

		day, err := time.ParseInLocation(time.DateOnly, *date.value, bangkok)
		if err != nil {
			problems = append(problems, api.FieldError{Field: date.field, Rule: "format", Message: badDateMessage})

			continue
		}

		*date.bound = &day
	}

	if first != nil && last != nil && first.After(*last) {
		problems = append(problems, api.FieldError{Field: "to", Rule: "range", Message: badRangeMessage})
	}

	if len(problems) > 0 {
		return Range{}, problems
	}

	if last != nil {
		last = new(last.AddDate(0, 0, 1))
	}

	return Range{From: first, To: last}, nil
}
