package claim

import (
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/money"
	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// highValueClaim creates a SUBMITTED 15,000.00 claim of req.anan, assigned to rev.malee, and returns the claim and the
// reviewer.
func highValueClaim(t *testing.T, pool *pgxpool.Pool) (Claim, Actor) {
	t.Helper()

	malee := member(t, pool, "rev.malee", "reviewer", uuid.Nil)
	anan := member(t, pool, "req.anan", "requester", malee)

	input := validInput()
	input.Items = input.Items[:1]
	input.Items[0].Amount = money.Satang(1_500_000)

	created, err := newService(pool).Create(t.Context(), Actor{ID: anan, Role: "requester"}, input, true)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	return created, Actor{ID: malee, Role: "reviewer"}
}

// pendingFinanceClaim has rev.malee approve a high-value claim and returns it once it is PENDING_FINANCE.
func pendingFinanceClaim(t *testing.T, pool *pgxpool.Pool) Claim {
	t.Helper()

	created, reviewer := highValueClaim(t, pool)

	pending, err := newService(pool).Act(t.Context(), reviewer, created.ID, created.Version, ActionApprove, "")
	if err != nil {
		t.Fatalf("Act approve: %v", err)
	}

	if pending.Status != db.ClaimStatusPENDINGFINANCE {
		t.Fatalf("status = %s, want PENDING_FINANCE", pending.Status)
	}

	return pending
}

// F08-TEST-11 (F08-AC-08, BR-13): two Finance Officers start a final approval and a final rejection together with the
// same version. Exactly one wins, the other is ErrConflict, and one final decision is in the history.
func TestF08TEST11ConcurrentFinalDecisionsOneWins(t *testing.T) {
	pool := pgtest.New(t)
	pending := pendingFinanceClaim(t, pool)
	service := newService(pool)

	decisions := []struct {
		actor   Actor
		action  Action
		comment string
	}{
		{Actor{ID: member(t, pool, "fin.wanida", RoleFinance, uuid.Nil), Role: RoleFinance}, ActionFinalApprove, ""},
		{Actor{ID: member(t, pool, "fin.second", RoleFinance, uuid.Nil), Role: RoleFinance}, ActionFinalReject, "Budget code is wrong"},
	}
	results := make([]error, len(decisions))

	var (
		ready sync.WaitGroup
		done  sync.WaitGroup
		start = make(chan struct{})
	)

	for i, decision := range decisions {
		ready.Add(1)
		done.Add(1)

		go func() {
			defer done.Done()

			ready.Done()
			<-start

			_, results[i] = service.Act(t.Context(), decision.actor, pending.ID, pending.Version, decision.action, decision.comment)
		}()
	}

	ready.Wait()
	close(start)
	done.Wait()

	var wins []Action

	for i, err := range results {
		switch {
		case err == nil:
			wins = append(wins, decisions[i].action)
		case !errors.Is(err, ErrConflict):
			t.Fatalf("decision %d error = %v, want ErrConflict", i, err)
		}
	}

	if len(wins) != 1 {
		t.Fatalf("%d decisions succeeded, want exactly one", len(wins))
	}

	after, err := service.Get(t.Context(), decisions[0].actor, pending.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	want := map[Action]struct {
		status  db.ClaimStatus
		history db.ClaimAction
	}{
		ActionFinalApprove: {db.ClaimStatusAPPROVED, db.ClaimActionFINALAPPROVE},
		ActionFinalReject:  {db.ClaimStatusREJECTED, db.ClaimActionFINALREJECT},
	}[wins[0]]

	last := after.History[len(after.History)-1]
	if after.Status != want.status || after.Version != pending.Version+1 || len(after.History) != 3 || last.Action != want.history {
		t.Fatalf("claim = %s, version %d, %d history entries, last %s; want %s, version %d, 3 entries, last %s",
			after.Status, after.Version, len(after.History), last.Action, want.status, pending.Version+1, want.history)
	}
}

// F08-TEST-12 (F08-AC-09, NFR-06): when the history insert fails during an approval to PENDING_FINANCE, a final
// approval, or a final rejection, the change is rolled back: the status and the version are as before.
func TestF08TEST12HistoryFailureRollsBackEachStage(t *testing.T) {
	finance := func(t *testing.T, pool *pgxpool.Pool) Actor {
		t.Helper()

		return Actor{ID: member(t, pool, "fin.wanida", RoleFinance, uuid.Nil), Role: RoleFinance}
	}

	for _, test := range []struct {
		name    string
		action  Action
		comment string
		setup   func(t *testing.T, pool *pgxpool.Pool) (Claim, Actor)
	}{
		{"approve to pending finance", ActionApprove, "", highValueClaim},
		{"final approve", ActionFinalApprove, "", func(t *testing.T, pool *pgxpool.Pool) (Claim, Actor) {
			t.Helper()

			return pendingFinanceClaim(t, pool), finance(t, pool)
		}},
		{"final reject", ActionFinalReject, "Budget code is wrong", func(t *testing.T, pool *pgxpool.Pool) (Claim, Actor) {
			t.Helper()

			return pendingFinanceClaim(t, pool), finance(t, pool)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			pool := pgtest.New(t)
			before, actor := test.setup(t, pool)
			failHistoryInserts(t, pool)

			if _, err := newService(pool).Act(t.Context(), actor, before.ID, before.Version, test.action, test.comment); err == nil {
				t.Fatal("Act succeeded, want the history failure")
			}

			var (
				status  db.ClaimStatus
				version int
			)
			if err := pool.QueryRow(t.Context(), `SELECT status, version FROM claims WHERE id = $1`, before.ID).Scan(&status, &version); err != nil {
				t.Fatalf("read claim: %v", err)
			}

			if status != before.Status || version != before.Version {
				t.Fatalf("claim = %s, version %d; want %s, version %d", status, version, before.Status, before.Version)
			}
		})
	}
}
