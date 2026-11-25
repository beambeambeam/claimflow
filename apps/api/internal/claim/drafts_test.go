package claim

import (
	"errors"
	"reflect"
	"slices"
	"sync"
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

func TestF05TEST02DraftsCheckItemAndTotalLimitsButSkipDates(t *testing.T) {
	pool := pgtest.New(t)
	service, owner := draftOwner(t, pool)
	for _, test := range []struct {
		name  string
		items []LineItemInput
		want  want
	}{
		{"empty", nil, want{"items", "BR-05"}},
		{"21 items", itemsOf(21), want{"items", "BR-05"}},
		{"amount", []LineItemInput{item("TRAVEL", 2000001)}, want{"items[0].amount", "BR-05"}},
		{"total", []LineItemInput{item("TRAVEL", 2000000), item("OTHER", 2000000), item("TRAVEL", 1000001)}, want{"total", "BR-06"}},
		{"meal", []LineItemInput{item("MEALS", 150001)}, want{"items[0].amount", "BR-12"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := validInput()
			input.Items = test.items
			_, err := service.Create(t.Context(), owner, input, false)
			if got := violations(t, err); !slices.Equal(got, []want{test.want}) {
				t.Fatalf("violations = %v, want %v", got, test.want)
			}
		})
	}

	for _, date := range []time.Time{
		time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.November, 26, 0, 0, 0, 0, time.UTC),
	} {
		input := validInput()
		input.Items[0].ExpenseDate = date
		draft := saveDraft(t, service, owner, input)
		if draft.Status != db.ClaimStatusDRAFT || !draft.Items[0].ExpenseDate.Equal(date) {
			t.Fatalf("draft did not keep expense date %s: %+v", date, draft)
		}
	}
	if n := count(t, pool, "claims"); n != 2 {
		t.Fatalf("stored claims = %d, want 2", n)
	}
}

func TestF05TEST04CreateDraftRequiresCurrentReviewer(t *testing.T) {
	pool := pgtest.New(t)
	owner := Actor{ID: member(t, pool, "req.anan", "requester", uuid.Nil)}

	_, err := newService(pool).Create(t.Context(), owner, validInput(), false)
	if !errors.Is(err, ErrForbidden) || count(t, pool, "claims") != 0 {
		t.Fatalf("Create = %v, want ErrForbidden and no claim", err)
	}
}

func TestF05TEST08InvalidEditLeavesDraftUnchanged(t *testing.T) {
	pool := pgtest.New(t)
	service, owner := draftOwner(t, pool)
	draft := saveDraft(t, service, owner, validInput())
	for _, test := range []struct {
		name  string
		items []LineItemInput
		want  want
	}{
		{"21 items", itemsOf(21), want{"items", "BR-05"}},
		{"meal", []LineItemInput{item("MEALS", 150001)}, want{"items[0].amount", "BR-12"}},
		{"total", []LineItemInput{item("TRAVEL", 2000000), item("OTHER", 2000000), item("TRAVEL", 1000001)}, want{"total", "BR-06"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := Input{Title: "Changed title", Purpose: "Changed purpose", Items: test.items}
			_, err := service.Edit(t.Context(), owner, draft.ID, draft.Version, input)
			if got := violations(t, err); !slices.Equal(got, []want{test.want}) {
				t.Fatalf("violations = %v, want %v", got, test.want)
			}
			got, err := service.Get(t.Context(), owner, draft.ID)
			if err != nil || !reflect.DeepEqual(got, draft) {
				t.Fatalf("after failed edit: claim = %+v, error = %v", got, err)
			}
		})
	}
}

func setDraftReviewer(t *testing.T, pool *pgxpool.Pool, owner, reviewer uuid.UUID) {
	t.Helper()

	if _, err := pool.Exec(t.Context(), `UPDATE "user" SET reviewer_id = $1 WHERE id = $2`,
		uuid.NullUUID{UUID: reviewer, Valid: reviewer != uuid.Nil}, owner); err != nil {
		t.Fatalf("change review assignment: %v", err)
	}
}

func TestF05TEST14DraftSubmissionUsesOwnersCurrentReviewer(t *testing.T) {
	pool := pgtest.New(t)
	service, owner := draftOwner(t, pool)
	draft := saveDraft(t, service, owner, validInput())
	reviewer := member(t, pool, "rev.somchai", "reviewer", uuid.Nil)
	setDraftReviewer(t, pool, owner.ID, reviewer)

	got, err := service.Act(t.Context(), owner, draft.ID, draft.Version, ActionSubmit, "")
	if err != nil || got.AssignedReviewer == nil || got.AssignedReviewer.ID != reviewer {
		t.Fatalf("submitted with reviewer %v, error %v", got.AssignedReviewer, err)
	}
}

func TestF05TEST15DraftSubmissionRequiresOwnersCurrentReviewer(t *testing.T) {
	pool := pgtest.New(t)
	service, owner := draftOwner(t, pool)
	draft := saveDraft(t, service, owner, validInput())
	setDraftReviewer(t, pool, owner.ID, uuid.Nil)

	got, err := service.Get(t.Context(), owner, draft.ID)
	if err != nil || !slices.Equal(got.AllowedActions, []Action{ActionEdit, ActionDelete}) {
		t.Fatalf("actions = %v, error = %v", got.AllowedActions, err)
	}
	if _, err := service.Act(t.Context(), owner, draft.ID, draft.Version, ActionSubmit, ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("submit error = %v, want ErrForbidden", err)
	}
	after, err := service.Get(t.Context(), owner, draft.ID)
	if err != nil || !reflect.DeepEqual(after, got) {
		t.Fatalf("after refused submit: claim = %+v, error = %v", after, err)
	}
}

func TestF05TEST17ConcurrentDraftSubmissionsOneWins(t *testing.T) {
	pool := pgtest.New(t)
	service, owner := draftOwner(t, pool)
	draft := saveDraft(t, service, owner, validInput())
	var ready, done sync.WaitGroup
	ready.Add(2)
	done.Add(2)
	start := make(chan struct{})
	results := make([]error, 2)

	for i := range results {
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			_, results[i] = service.Act(t.Context(), owner, draft.ID, draft.Version, ActionSubmit, "")
		}()
	}

	ready.Wait()
	close(start)
	done.Wait()

	wins := 0
	for _, err := range results {
		switch {
		case err == nil:
			wins++
		case !errors.Is(err, ErrConflict):
			t.Fatalf("submit = %v, want success or ErrConflict", err)
		}
	}
	got, err := service.Get(t.Context(), owner, draft.ID)
	if err != nil || wins != 1 || got.Status != db.ClaimStatusSUBMITTED || got.Version != draft.Version+1 || len(got.History) != 1 {
		t.Fatalf("wins = %d, claim = %+v, error = %v", wins, got, err)
	}
}

func TestF05TEST18HistoryFailureRollsBackDraftSubmission(t *testing.T) {
	pool := pgtest.New(t)
	service, owner := draftOwner(t, pool)
	draft := saveDraft(t, service, owner, validInput())
	if _, err := pool.Exec(t.Context(), `
		CREATE FUNCTION fail_draft_history() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'forced draft history failure'; END; $$;
		CREATE TRIGGER fail_draft_history BEFORE INSERT ON claim_history
		FOR EACH ROW EXECUTE FUNCTION fail_draft_history()`); err != nil {
		t.Fatalf("create history trigger: %v", err)
	}

	if _, err := service.Act(t.Context(), owner, draft.ID, draft.Version, ActionSubmit, ""); err == nil {
		t.Fatal("submission succeeded with a broken history trigger")
	}
	got, err := service.Get(t.Context(), owner, draft.ID)
	if err != nil || !reflect.DeepEqual(got, draft) {
		t.Fatalf("after rolled-back submission: claim = %+v, error = %v", got, err)
	}
}

func TestReturnedClaimCanBeEditedAndUsesItsOriginalReviewAssignment(t *testing.T) {
	pool := pgtest.New(t)
	service, owner := draftOwner(t, pool)
	first := draftNow.AddDate(0, 0, -5)
	start := NewService(pool, func() time.Time { return first })
	original, err := start.Create(t.Context(), owner, validInput(), true)
	if err != nil {
		t.Fatalf("first submission: %v", err)
	}
	if _, err := pool.Exec(t.Context(), `UPDATE claims SET status = 'RETURNED' WHERE id = $1`, original.ID); err != nil {
		t.Fatalf("set returned status: %v", err)
	}
	setDraftReviewer(t, pool, owner.ID, uuid.Nil)

	input := validInput()
	input.Items[0].ExpenseDate = first.AddDate(0, 0, -90)
	edited, err := service.Edit(t.Context(), owner, original.ID, original.Version, input)
	if err != nil || edited.Status != db.ClaimStatusRETURNED || !slices.Equal(edited.AllowedActions, []Action{ActionEdit, ActionSubmit, ActionWithdraw}) {
		t.Fatalf("edit returned: claim = %+v, error = %v", edited, err)
	}
	if err := service.Delete(t.Context(), owner, edited.ID, edited.Version); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("delete returned = %v, want ErrInvalidTransition", err)
	}
	got, err := service.Act(t.Context(), owner, edited.ID, edited.Version, ActionSubmit, "")
	if err != nil || got.Status != db.ClaimStatusSUBMITTED || got.AssignedReviewer == nil ||
		got.AssignedReviewer.ID != original.AssignedReviewer.ID || got.FirstSubmittedAt == nil || !got.FirstSubmittedAt.Equal(first) ||
		got.SubmittedAt == nil || !got.SubmittedAt.Equal(draftNow) || len(got.History) != 2 || got.History[1].Action != db.ClaimActionRESUBMIT {
		t.Fatalf("submit returned: claim = %+v, error = %v", got, err)
	}
}

func TestActCannotReplaceOrDeleteDraftContent(t *testing.T) {
	pool := pgtest.New(t)
	service, owner := draftOwner(t, pool)
	draft := saveDraft(t, service, owner, validInput())
	for _, action := range []Action{ActionEdit, ActionDelete} {
		if _, err := service.Act(t.Context(), owner, draft.ID, draft.Version, action, ""); !errors.Is(err, ErrInvalidTransition) {
			t.Fatalf("Act(%s) = %v, want ErrInvalidTransition", action, err)
		}
	}

	got, err := service.Get(t.Context(), owner, draft.ID)
	if err != nil || !reflect.DeepEqual(got, draft) {
		t.Fatalf("content action changed draft: claim = %+v, error = %v", got, err)
	}
}

func TestDraftMutationsReportConflictWhenConditionalWriteChangesNoRows(t *testing.T) {
	pool := pgtest.New(t)
	service, owner := draftOwner(t, pool)
	draft := saveDraft(t, service, owner, validInput())

	// Make each conditional write affect zero rows, the result when a competing write wins first.
	if _, err := pool.Exec(t.Context(), `
		CREATE FUNCTION skip_claim_write() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RETURN NULL; END; $$;
		CREATE TRIGGER skip_claim_write BEFORE UPDATE OR DELETE ON claims
		FOR EACH ROW EXECUTE FUNCTION skip_claim_write()`); err != nil {
		t.Fatalf("create write trigger: %v", err)
	}

	input := validInput()
	input.Title = "Revised draft"
	if _, err := service.Edit(t.Context(), owner, draft.ID, draft.Version, input); !errors.Is(err, ErrConflict) {
		t.Fatalf("edit = %v, want ErrConflict", err)
	}
	if err := service.Delete(t.Context(), owner, draft.ID, draft.Version); !errors.Is(err, ErrConflict) {
		t.Fatalf("delete = %v, want ErrConflict", err)
	}
	if _, err := service.Act(t.Context(), owner, draft.ID, draft.Version, ActionSubmit, ""); !errors.Is(err, ErrConflict) {
		t.Fatalf("submit = %v, want ErrConflict", err)
	}

	got, err := service.Get(t.Context(), owner, draft.ID)
	if err != nil || !reflect.DeepEqual(got, draft) {
		t.Fatalf("after skipped write: claim = %+v, error = %v", got, err)
	}
}
