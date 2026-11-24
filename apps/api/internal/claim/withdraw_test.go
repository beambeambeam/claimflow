package claim

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// F07-TEST-08 (F07-AC-06, BR-13): the owner withdraws and the reviewer approves at the same moment with the same
// version. One of them succeeds, the other gets ErrConflict, and the claim has one new history entry that matches the
// final status.
func TestF07TEST08ConcurrentWithdrawAndApproveHaveOneWinner(t *testing.T) {
	pool := pgtest.New(t)
	created, reviewer := submittedClaim(t, pool)
	owner := Actor{ID: created.Requester.ID}
	service := newService(pool)

	steps := []struct {
		actor  Actor
		action Action
	}{
		{owner, ActionWithdraw},
		{reviewer, ActionApprove},
	}
	results := make([]error, len(steps))

	var (
		ready sync.WaitGroup
		done  sync.WaitGroup
		start = make(chan struct{})
	)

	for i, step := range steps {
		ready.Add(1)
		done.Add(1)

		go func() {
			defer done.Done()

			ready.Done()
			<-start

			_, results[i] = service.Act(t.Context(), step.actor, created.ID, created.Version, step.action, "")
		}()
	}

	ready.Wait()
	close(start)
	done.Wait()

	var wins []Action

	for i, err := range results {
		switch {
		case err == nil:
			wins = append(wins, steps[i].action)
		case !errors.Is(err, ErrConflict):
			t.Fatalf("%s error = %v, want ErrConflict", steps[i].action, err)
		}
	}

	if len(wins) != 1 {
		t.Fatalf("%d steps succeeded, want exactly one", len(wins))
	}

	after, err := service.Get(t.Context(), owner, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	want := map[Action]string{ActionWithdraw: "WITHDRAWN", ActionApprove: "APPROVED"}[wins[0]]
	if string(after.Status) != want || after.Version != created.Version+1 || len(after.History) != len(created.History)+1 ||
		string(after.History[len(after.History)-1].ToStatus) != want {
		t.Fatalf("claim = %s, version %d, %d history entries; want %s, version %d, %d entries ending in %s",
			after.Status, after.Version, len(after.History), want, created.Version+1, len(created.History)+1, want)
	}
}

// F07-TEST-09 (F07-AC-07, NFR-06): when the history insert fails, the withdrawal is rolled back: the status and the
// version are as before.
func TestF07TEST09HistoryFailureRollsBackTheWithdrawal(t *testing.T) {
	pool := pgtest.New(t)
	created, _ := submittedClaim(t, pool)

	// A test-only trigger makes every history insert fail.
	if _, err := pool.Exec(t.Context(), `
		CREATE FUNCTION fail_history_insert() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'forced history failure'; END; $$;
		CREATE TRIGGER fail_history_insert BEFORE INSERT ON claim_history
		FOR EACH ROW EXECUTE FUNCTION fail_history_insert();`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	if _, err := newService(pool).Act(t.Context(), Actor{ID: created.Requester.ID}, created.ID, created.Version, ActionWithdraw, "No longer needed."); err == nil {
		t.Fatal("Act succeeded, want the history failure")
	}

	var (
		status    string
		version   int
		updatedAt time.Time
	)
	if err := pool.QueryRow(t.Context(), `SELECT status::text, version, updated_at FROM claims WHERE id = $1`, created.ID).
		Scan(&status, &version, &updatedAt); err != nil {
		t.Fatalf("read claim: %v", err)
	}

	if status != "SUBMITTED" || version != created.Version || !updatedAt.Equal(created.UpdatedAt) {
		t.Fatalf("claim = %s, version %d, updated %s; want SUBMITTED, %d, %s", status, version, updatedAt, created.Version, created.UpdatedAt)
	}
}
