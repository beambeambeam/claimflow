package claim

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// resubmitNow is the clock of the resubmission tests (F-06 §1.8): 25 Nov 2026, 10:00 Bangkok.
var resubmitNow = time.Date(2026, time.November, 25, 10, 0, 0, 0, bangkok)

// firstSubmission is when the claims of these tests were first submitted: 20 Nov 2026, 09:00 Bangkok.
var firstSubmission = time.Date(2026, time.November, 20, 9, 0, 0, 0, bangkok)

// returnedClaim has req.anan submit a claim at firstSubmission and rev.malee return it, and returns the RETURNED claim, its
// owner, and its reviewer. The service of the resubmission has the clock at resubmitNow.
func returnedClaim(t *testing.T, pool *pgxpool.Pool) (claim Claim, owner, reviewer Actor) {
	t.Helper()

	malee := member(t, pool, "rev.malee", "reviewer", uuid.Nil)
	anan := member(t, pool, "req.anan", "requester", malee)
	owner, reviewer = Actor{ID: anan}, Actor{ID: malee}

	first := NewService(pool, func() time.Time { return firstSubmission })

	created, err := first.Create(t.Context(), owner, validInput(), true)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	claim, err = first.Act(t.Context(), reviewer, created.ID, created.Version, ActionReturn, "Please attach the receipt.")
	if err != nil {
		t.Fatalf("Act return: %v", err)
	}

	return claim, owner, reviewer
}

// resubmitService is the Service of a resubmission at resubmitNow.
func resubmitService(pool *pgxpool.Pool) Service {
	return NewService(pool, func() time.Time { return resubmitNow })
}

// F06-TEST-10 (F06-AC-08, BR-07): a resubmission counts the 90 days from the first submission, not from now. With the first
// submission on 2026-11-20, an item dated 2026-08-22 is accepted and one dated 2026-08-21 is a ValidationError on
// items[0].expenseDate for BR-07; the claim then stays RETURNED with no new history.
func TestF06TEST10ResubmissionCountsNinetyDaysFromTheFirstSubmission(t *testing.T) {
	cases := []struct {
		name        string
		expenseDate string
		valid       bool
	}{
		{"90 days before the first submission", "2026-08-22", true},
		{"91 days before the first submission", "2026-08-21", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pool := pgtest.New(t)
			returned, owner, _ := returnedClaim(t, pool)

			// Edit does not check dates (F-05 §1.4), so the date is set directly.
			if _, err := pool.Exec(t.Context(), `UPDATE line_items SET expense_date = $2 WHERE claim_id = $1 AND line_no = 1`, returned.ID, tc.expenseDate); err != nil {
				t.Fatalf("set the expense date: %v", err)
			}

			got, err := resubmitService(pool).Act(t.Context(), owner, returned.ID, returned.Version, ActionSubmit, "")

			if tc.valid {
				if err != nil || got.Status != db.ClaimStatusSUBMITTED {
					t.Fatalf("Act = %s, %v; want SUBMITTED", got.Status, err)
				}

				return
			}

			var invalid ValidationError
			if !errors.As(err, &invalid) || !slices.Equal(invalid.Fields, []FieldError{{
				Field: "items[0].expenseDate", Rule: "BR-07",
				Message: "Expense date cannot be more than 90 days before the claim was first submitted.",
			}}) {
				t.Fatalf("Act error = %v, want a BR-07 ValidationError on items[0].expenseDate", err)
			}

			after, err := resubmitService(pool).Get(t.Context(), owner, returned.ID)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}

			if after.Status != db.ClaimStatusRETURNED || after.Version != returned.Version || len(after.History) != len(returned.History) {
				t.Fatalf("claim = %s, version %d, %d history rows; want it RETURNED, version %d, %d rows",
					after.Status, after.Version, len(after.History), returned.Version, len(returned.History))
			}
		})
	}
}

// F06-TEST-17 (F06-AC-07, BR-17): the owner of a RETURNED claim can resubmit it after the owner's reviewer was removed:
// submit is in allowedActions, the claim goes back to the reviewer it had, and one RESUBMIT entry is written. BR-15 is
// not checked, because the claim already has an assigned reviewer.
func TestF06TEST17ResubmissionWorksWithoutAnOwnerReviewer(t *testing.T) {
	pool := pgtest.New(t)
	returned, owner, reviewer := returnedClaim(t, pool)
	service := resubmitService(pool)

	if _, err := pool.Exec(t.Context(), `UPDATE "user" SET reviewer_id = NULL WHERE id = $1`, owner.ID); err != nil {
		t.Fatalf("clear the owner's reviewer: %v", err)
	}

	seen, err := service.Get(t.Context(), owner, returned.ID)
	if err != nil || !slices.Contains(seen.AllowedActions, ActionSubmit) {
		t.Fatalf("Get allowed actions = %v, error = %v; want submit among them", seen.AllowedActions, err)
	}

	got, err := service.Act(t.Context(), owner, returned.ID, returned.Version, ActionSubmit, "")
	if err != nil {
		t.Fatalf("Act submit: %v", err)
	}

	if got.Status != db.ClaimStatusSUBMITTED || got.AssignedReviewer == nil || got.AssignedReviewer.ID != reviewer.ID {
		t.Fatalf("claim = %s with reviewer %+v, want SUBMITTED with the reviewer it had", got.Status, got.AssignedReviewer)
	}

	if len(got.History) != len(returned.History)+1 {
		t.Fatalf("history has %d entries, want %d", len(got.History), len(returned.History)+1)
	}

	last := got.History[len(got.History)-1]
	if last.Action != db.ClaimActionRESUBMIT || last.FromStatus == nil || *last.FromStatus != db.ClaimStatusRETURNED || last.Comment != nil {
		t.Fatalf("last history = %+v, want RESUBMIT from RETURNED with no comment", last)
	}
}

// F06-TEST-12, resubmission part (F06-AC-10, NFR-06): when the history insert fails, the resubmission is rolled back: the
// status, the return count, the submission time, and the version are as before.
func TestF06TEST12HistoryFailureRollsBackTheResubmission(t *testing.T) {
	pool := pgtest.New(t)
	returned, owner, _ := returnedClaim(t, pool)

	// A test-only trigger makes every history insert fail.
	if _, err := pool.Exec(t.Context(), `
		CREATE FUNCTION fail_history_insert() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'forced history failure'; END; $$;
		CREATE TRIGGER fail_history_insert BEFORE INSERT ON claim_history
		FOR EACH ROW EXECUTE FUNCTION fail_history_insert();`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	if _, err := resubmitService(pool).Act(t.Context(), owner, returned.ID, returned.Version, ActionSubmit, ""); err == nil {
		t.Fatal("Act succeeded, want the history failure")
	}

	var (
		status      string
		returnCount int
		submittedAt time.Time
		version     int
	)
	if err := pool.QueryRow(t.Context(), `SELECT status::text, return_count, submitted_at, version FROM claims WHERE id = $1`, returned.ID).
		Scan(&status, &returnCount, &submittedAt, &version); err != nil {
		t.Fatalf("read claim: %v", err)
	}

	if status != "RETURNED" || returnCount != returned.ReturnCount || !submittedAt.Equal(*returned.SubmittedAt) || version != returned.Version {
		t.Fatalf("claim = %s, return count %d, submitted %s, version %d; want RETURNED, %d, %s, %d",
			status, returnCount, submittedAt, version, returned.ReturnCount, returned.SubmittedAt, returned.Version)
	}
}
