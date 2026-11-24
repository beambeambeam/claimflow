package claim

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

var draftNow = time.Date(2026, time.November, 25, 10, 0, 0, 0, bangkok)

func draftOwner(t *testing.T, pool *pgxpool.Pool) (Service, Actor) {
	t.Helper()

	reviewerID := member(t, pool, "rev.malee", "reviewer", uuid.Nil)
	owner := Actor{ID: member(t, pool, "req.anan", "requester", reviewerID)}

	return NewService(pool, func() time.Time { return draftNow }), owner
}

func saveDraft(t *testing.T, service Service, owner Actor, input Input) Claim {
	t.Helper()

	draft, err := service.Create(t.Context(), owner, input, false)
	if err != nil {
		t.Fatalf("save draft: %v", err)
	}

	return draft
}

func TestF05TEST12SubmissionChecksExpenseDateWindow(t *testing.T) {
	pool := pgtest.New(t)
	service, owner := draftOwner(t, pool)

	for _, test := range []struct {
		date    string
		message string
	}{
		{"2026-11-26", "Expense date cannot be in the future."},
		{"2026-08-27", ""},
		{"2026-08-26", "Expense date cannot be more than 90 days before the claim was first submitted."},
	} {
		t.Run(test.date, func(t *testing.T) {
			date, err := time.Parse(time.DateOnly, test.date)
			if err != nil {
				t.Fatal(err)
			}

			input := validInput()
			input.Items = input.Items[:1]
			input.Items[0].ExpenseDate = date
			draft := saveDraft(t, service, owner, input)
			result, err := service.Act(t.Context(), owner, draft.ID, draft.Version, ActionSubmit, "")
			if test.message == "" {
				if err != nil || result.Status != db.ClaimStatusSUBMITTED {
					t.Fatalf("submit at 90 days: status = %s, error = %v", result.Status, err)
				}

				return
			}

			var invalid ValidationError
			want := FieldError{Field: "items[0].expenseDate", Rule: "BR-07", Message: test.message}
			if !errors.As(err, &invalid) || len(invalid.Fields) != 1 || invalid.Fields[0] != want {
				t.Fatalf("submit error = %v (%+v), want %+v", err, invalid.Fields, want)
			}
			got, err := service.Get(t.Context(), owner, draft.ID)
			if err != nil || got.Status != db.ClaimStatusDRAFT || len(got.History) != 0 {
				t.Fatalf("after refusal: status = %s, history = %v, error = %v", got.Status, got.History, err)
			}
		})
	}
}
