package report

import (
	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/daterange"
)

// ParseRange reads the optional `from` and `to` query dates (YYYY-MM-DD, both inclusive, in Asia/Bangkok) as a half-open
// Range: from 00:00 on `from` to 00:00 on the day after `to`, the Bangkok-midnight bounds of F-09 §1.8. A nil date
// leaves its side open. A date that is not a calendar date, or `from` after `to`, returns one FieldError per bad
// parameter and no Range. The checks are those of the All Claims filters, in internal/daterange.
func ParseRange(from, to *string) (Range, []api.FieldError) {
	first, last, problems := daterange.Parse(from, to)
	if len(problems) > 0 {
		return Range{}, problems
	}

	start, end := daterange.Bounds(first, last)

	return Range{From: start, To: end}, nil
}
