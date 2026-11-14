package claim

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// submittedClaim creates a SUBMITTED claim of req.anan, assigned to rev.malee, and returns the claim and the reviewer.
func submittedClaim(t *testing.T, pool *pgxpool.Pool) (Claim, Actor) {
	t.Helper()

	malee := member(t, pool, "rev.malee", "reviewer", uuid.Nil)
	anan := member(t, pool, "req.anan", "requester", malee)

	created, err := newService(pool).Create(t.Context(), Actor{ID: anan}, validInput(), true)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	return created, Actor{ID: malee}
}

// F04-TEST-06 (F04-AC-06, BR-04): the reject reason is required and has 5 to 500 characters after trimming, counted as
// characters, not bytes. An approve comment is optional with at most 500. A refused comment changes nothing.
func TestF04TEST06CommentRuleBR04(t *testing.T) {
	const (
		required = "Enter a reason (5 to 500 characters)."
		short    = "Reason must be at least 5 characters."
		long     = "Comment cannot be more than 500 characters."
	)

	cases := []struct {
		name    string
		action  string
		comment string
		message string // empty: the decision is taken
	}{
		{"reject without a reason", ActionReject, "", required},
		{"reject with blanks", ActionReject, "   ", required},
		{"reject with 4 characters", ActionReject, "abcd", short},
		{"reject with 4 characters and blanks around", ActionReject, "  abcd  ", short},
		{"reject with 4 Thai characters (12 bytes)", ActionReject, "กขคง", short},
		{"reject with 5 characters", ActionReject, "abcde", ""},
		{"reject with 5 characters and blanks around", ActionReject, "  abcde  ", ""},
		{"reject with 500 characters", ActionReject, strings.Repeat("a", 500), ""},
		{"reject with 500 Thai characters (1,500 bytes)", ActionReject, strings.Repeat("ก", 500), ""},
		{"reject with 501 characters", ActionReject, strings.Repeat("a", 501), long},
		{"reject with 501 Thai characters", ActionReject, strings.Repeat("ก", 501), long},
		{"approve without a comment", ActionApprove, "", ""},
		{"approve with blanks", ActionApprove, "   ", ""},
		{"approve with 500 characters", ActionApprove, strings.Repeat("a", 500), ""},
		{"approve with 501 characters", ActionApprove, strings.Repeat("a", 501), long},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pool := pgtest.New(t)
			created, reviewer := submittedClaim(t, pool)

			decided, err := newService(pool).Act(t.Context(), reviewer, created.ID, created.Version, tc.action, tc.comment)

			if tc.message == "" {
				if err != nil {
					t.Fatalf("Act error = %v, want the decision taken", err)
				}

				// The stored comment is the trimmed one, and an empty one is stored as no comment.
				last := decided.History[len(decided.History)-1]
				if want := strings.TrimSpace(tc.comment); (want == "") != (last.Comment == nil) || (last.Comment != nil && *last.Comment != want) {
					t.Fatalf("comment = %v, want %q", last.Comment, want)
				}

				return
			}

			var invalid ValidationError
			if !errors.As(err, &invalid) || len(invalid.Fields) != 1 {
				t.Fatalf("Act error = %v, want a ValidationError with one field", err)
			}

			if got := invalid.Fields[0]; got.Field != "comment" || got.Rule != "BR-04" || got.Message != tc.message {
				t.Fatalf("field error = %+v, want comment, BR-04, %q", got, tc.message)
			}

			after, err := newService(pool).Get(t.Context(), reviewer, created.ID)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}

			if after.Status != db.ClaimStatusSUBMITTED || after.Version != created.Version || after.DecidedAt != nil || len(after.History) != 1 {
				t.Fatalf("claim = %+v after the refused comment, want it unchanged", after)
			}
		})
	}
}

// The status rule comes before the comment rule (F-04 §1.4): a decided claim with a blank reason is a transition error.
func TestActChecksTheStatusBeforeTheComment(t *testing.T) {
	pool := pgtest.New(t)
	created, reviewer := submittedClaim(t, pool)
	service := newService(pool)

	approved, err := service.Act(t.Context(), reviewer, created.ID, created.Version, ActionApprove, "")
	if err != nil {
		t.Fatalf("Act approve: %v", err)
	}

	if _, err := service.Act(t.Context(), reviewer, created.ID, approved.Version, ActionReject, ""); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("Act error = %v, want ErrInvalidTransition", err)
	}
}

// F04-TEST-15 (F04-AC-12, BR-13): an approval and a rejection start together with the same version. Exactly one wins,
// the other is ErrConflict, and one decision is in the history.
func TestF04TEST15ConcurrentDecisionsOneWins(t *testing.T) {
	pool := pgtest.New(t)
	created, reviewer := submittedClaim(t, pool)
	service := newService(pool)

	actions := []struct{ action, comment string }{
		{ActionApprove, ""},
		{ActionReject, "Receipt is missing."},
	}
	results := make([]error, len(actions))

	var (
		ready sync.WaitGroup
		done  sync.WaitGroup
		start = make(chan struct{})
	)

	for i, action := range actions {
		ready.Add(1)
		done.Add(1)

		go func() {
			defer done.Done()

			ready.Done()
			<-start

			_, results[i] = service.Act(t.Context(), reviewer, created.ID, created.Version, action.action, action.comment)
		}()
	}

	ready.Wait()
	close(start)
	done.Wait()

	var wins []string

	for i, err := range results {
		switch {
		case err == nil:
			wins = append(wins, actions[i].action)
		case !errors.Is(err, ErrConflict):
			t.Fatalf("decision %d error = %v, want ErrConflict", i, err)
		}
	}

	if len(wins) != 1 {
		t.Fatalf("%d decisions succeeded, want exactly one", len(wins))
	}

	after, err := service.Get(t.Context(), reviewer, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	wantStatus := map[string]db.ClaimStatus{ActionApprove: db.ClaimStatusAPPROVED, ActionReject: db.ClaimStatusREJECTED}[wins[0]]
	if after.Status != wantStatus || after.Version != created.Version+1 || len(after.History) != 2 {
		t.Fatalf("claim = %+v, want %s, version %d, 2 history entries", after, wantStatus, created.Version+1)
	}
}

// A decision with a version that is not the current one is ErrConflict, also when another decision has since left the
// claim in a status that does not allow the action: the caller's screen is stale (BR-13). With the current version, the
// same action is ErrInvalidTransition (F04-TEST-14).
func TestActWithStaleVersionOnDecidedClaimIsConflict(t *testing.T) {
	pool := pgtest.New(t)
	created, reviewer := submittedClaim(t, pool)
	service := newService(pool)

	approved, err := service.Act(t.Context(), reviewer, created.ID, created.Version, ActionApprove, "")
	if err != nil {
		t.Fatalf("Act approve: %v", err)
	}

	if _, err := service.Act(t.Context(), reviewer, created.ID, created.Version, ActionReject, "Receipt is missing."); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale Act error = %v, want ErrConflict", err)
	}

	if _, err := service.Act(t.Context(), reviewer, created.ID, approved.Version, ActionReject, "Receipt is missing."); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("current Act error = %v, want ErrInvalidTransition", err)
	}
}

// F04-TEST-17 (F04-AC-14, NFR-06): when the history insert fails, the decision is rolled back: the status, the version,
// and the decision time are as before.
func TestF04TEST17HistoryFailureRollsBackTheDecision(t *testing.T) {
	pool := pgtest.New(t)
	created, reviewer := submittedClaim(t, pool)

	// A test-only trigger makes every history insert fail.
	if _, err := pool.Exec(t.Context(), `
		CREATE FUNCTION fail_history_insert() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'forced history failure'; END; $$;
		CREATE TRIGGER fail_history_insert BEFORE INSERT ON claim_history
		FOR EACH ROW EXECUTE FUNCTION fail_history_insert();`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	if _, err := newService(pool).Act(t.Context(), reviewer, created.ID, created.Version, ActionApprove, "OK"); err == nil {
		t.Fatal("Act succeeded, want the history failure")
	}

	var (
		status  string
		version int
		decided bool
	)
	if err := pool.QueryRow(t.Context(), `SELECT status::text, version, decided_at IS NOT NULL FROM claims WHERE id = $1`, created.ID).
		Scan(&status, &version, &decided); err != nil {
		t.Fatalf("read claim: %v", err)
	}

	if status != "SUBMITTED" || version != created.Version || decided {
		t.Fatalf("claim = %s, version %d, decided %t; want SUBMITTED, version %d, not decided", status, version, decided, created.Version)
	}
}
