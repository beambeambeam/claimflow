package claim

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// F06-TEST-02 (F06-AC-02, BR-04): a return reason follows the reject rule. Empty, blanks only, one character below the
// minimum, and one above the maximum are a ValidationError on comment; the minimum and the maximum after trimming are
// accepted. The limits are the constants, so a change of the minimum changes no test.
func TestF06TEST02ReturnReasonFollowsTheRejectRuleBR04(t *testing.T) {
	cases := []struct {
		name    string
		comment string
		valid   bool
	}{
		{"empty", "", false},
		{"blanks only", "   ", false},
		{"one below the minimum", strings.Repeat("a", MinReasonLength-1), false},
		{"one below the minimum with blanks around", "  " + strings.Repeat("a", MinReasonLength-1) + "  ", false},
		{"the minimum", strings.Repeat("a", MinReasonLength), true},
		{"the minimum with blanks around", "  " + strings.Repeat("a", MinReasonLength) + "  ", true},
		{"the maximum", strings.Repeat("a", MaxCommentLength), true},
		{"one above the maximum", strings.Repeat("a", MaxCommentLength+1), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pool := pgtest.New(t)
			created, reviewer := submittedClaim(t, pool)

			returned, err := newService(pool).Act(t.Context(), reviewer, created.ID, created.Version, ActionReturn, tc.comment)

			if tc.valid {
				if err != nil {
					t.Fatalf("Act error = %v, want the claim returned", err)
				}

				last := returned.History[len(returned.History)-1]
				if want := strings.TrimSpace(tc.comment); returned.Status != db.ClaimStatusRETURNED || last.Comment == nil || *last.Comment != want {
					t.Fatalf("claim = %s with comment %v, want RETURNED with the trimmed reason", returned.Status, last.Comment)
				}

				return
			}

			// The refusal is the one that reject gives for the same text (F-06 §1.6).
			var want ValidationError
			if !errors.As(checkComment(ActionReject, tc.comment), &want) {
				t.Fatalf("the reject rule accepts %q, so the case is not invalid", tc.comment)
			}

			var invalid ValidationError
			if !errors.As(err, &invalid) || !reflect.DeepEqual(invalid, want) || invalid.Fields[0].Field != "comment" || invalid.Fields[0].Rule != "BR-04" {
				t.Fatalf("Act error = %v, want the ValidationError %+v", err, want)
			}

			after, err := newService(pool).Get(t.Context(), reviewer, created.ID)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}

			if after.Status != db.ClaimStatusSUBMITTED || after.ReturnCount != 0 || after.Version != created.Version || len(after.History) != 1 {
				t.Fatalf("claim = %+v after the refused reason, want it unchanged", after)
			}
		})
	}
}
