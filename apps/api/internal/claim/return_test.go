package claim

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

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

// returnAndResubmit sends the claim back with a reason and has its requester submit it again, and returns the claim as
// the requester left it.
func returnAndResubmit(t *testing.T, pool *pgxpool.Pool, reviewer Actor, current *Claim) Claim {
	t.Helper()

	service := newService(pool)

	returned, err := service.Act(t.Context(), reviewer, current.ID, current.Version, ActionReturn, "Please attach the receipt.")
	if err != nil {
		t.Fatalf("Act return: %v", err)
	}

	resubmitted, err := service.Act(t.Context(), Actor{ID: current.Requester.ID}, current.ID, returned.Version, ActionSubmit, "")
	if err != nil {
		t.Fatalf("Act submit: %v", err)
	}

	return resubmitted
}

// F06-TEST-07 (F06-AC-05, BR-14): a claim that was returned MaxReturns times and resubmitted cannot be returned again:
// the refusal is an invalid transition for the return limit. The reviewer can still approve it.
func TestF06TEST07FourthReturnIsRefusedAndApproveStillWorks(t *testing.T) {
	pool := pgtest.New(t)
	created, reviewer := submittedClaim(t, pool)
	service := newService(pool)

	current := created
	for range MaxReturns {
		current = returnAndResubmit(t, pool, reviewer, &current)
	}

	if current.Status != db.ClaimStatusSUBMITTED || current.ReturnCount != MaxReturns {
		t.Fatalf("claim = %s with return count %d, want SUBMITTED with %d", current.Status, current.ReturnCount, MaxReturns)
	}

	_, err := service.Act(t.Context(), reviewer, current.ID, current.Version, ActionReturn, "One more thing.")
	if !errors.Is(err, ErrInvalidTransition) || !errors.Is(err, ErrReturnLimit) {
		t.Fatalf("Act return error = %v, want ErrInvalidTransition for the return limit", err)
	}

	after, err := service.Get(t.Context(), reviewer, current.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if after.Status != db.ClaimStatusSUBMITTED || after.ReturnCount != MaxReturns || after.Version != current.Version {
		t.Fatalf("claim = %+v after the refused return, want it unchanged", after)
	}

	approved, err := service.Act(t.Context(), reviewer, current.ID, current.Version, ActionApprove, "")
	if err != nil || approved.Status != db.ClaimStatusAPPROVED {
		t.Fatalf("Act approve = %s, %v; want APPROVED", approved.Status, err)
	}
}
