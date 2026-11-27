package report

import (
	"maps"
	"testing"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// scopeDataset stores the fixed dataset of F10-TEST-01 and F10-TEST-02. Malee's assigned claims are two SUBMITTED, one
// RETURNED, one PENDING_FINANCE, two APPROVED, one REJECTED, and one WITHDRAWN. Somchai's are one SUBMITTED, one
// PENDING_FINANCE, and one APPROVED. Malee also has one SUBMITTED claim of her own, assigned to Wanida. Two drafts,
// one of them with an assigned reviewer, belong to nobody's summary.
func scopeDataset(t *testing.T, p *people, store func(*stored)) {
	t.Helper()

	submitted := at(10, 1, 9, 0, 0)
	decided := decidedAt(at(10, 5, 9, 0, 0))

	for _, c := range []*stored{
		{requester: p.anan, reviewer: p.malee, status: db.ClaimStatusSUBMITTED, firstSubmitted: submitted},
		{requester: p.anan, reviewer: p.malee, status: db.ClaimStatusSUBMITTED, firstSubmitted: submitted},
		{requester: p.anan, reviewer: p.malee, status: db.ClaimStatusRETURNED, firstSubmitted: submitted},
		{requester: p.anan, reviewer: p.malee, status: db.ClaimStatusPENDINGFINANCE, firstSubmitted: submitted},
		{requester: p.anan, reviewer: p.malee, status: db.ClaimStatusAPPROVED, firstSubmitted: submitted, decided: decided},
		{requester: p.anan, reviewer: p.malee, status: db.ClaimStatusAPPROVED, firstSubmitted: submitted, decided: decided},
		{requester: p.anan, reviewer: p.malee, status: db.ClaimStatusREJECTED, firstSubmitted: submitted, decided: decided},
		{requester: p.anan, reviewer: p.malee, status: db.ClaimStatusWITHDRAWN, firstSubmitted: submitted},
		{requester: p.bella, reviewer: p.somchai, status: db.ClaimStatusSUBMITTED, firstSubmitted: submitted},
		{requester: p.bella, reviewer: p.somchai, status: db.ClaimStatusPENDINGFINANCE, firstSubmitted: submitted},
		{requester: p.bella, reviewer: p.somchai, status: db.ClaimStatusAPPROVED, firstSubmitted: submitted, decided: decided},
		{requester: p.malee, reviewer: p.wanida, status: db.ClaimStatusSUBMITTED, firstSubmitted: submitted},
		{requester: p.anan, status: db.ClaimStatusDRAFT},
		{requester: p.bella, reviewer: p.malee, status: db.ClaimStatusDRAFT},
	} {
		store(c)
	}
}

// F10-TEST-01 (F10-AC-01): a reviewer's summary covers only the claims assigned to her, not another reviewer's, not her
// own claims, and no draft. A Finance Officer's covers every claim that is not a draft.
func TestF10TEST01ScopeOfReviewerAndFinance(t *testing.T) {
	pool := pgtest.New(t)
	p := seedPeople(t, pool)
	scopeDataset(t, p, func(c *stored) { insertClaim(t, pool, c) })
	service := NewService(pool)

	for _, test := range []struct {
		name  string
		actor Actor
		want  map[db.ClaimStatus]int
	}{
		{"reviewer", p.maleeActor(), map[db.ClaimStatus]int{
			db.ClaimStatusSUBMITTED: 2, db.ClaimStatusRETURNED: 1, db.ClaimStatusPENDINGFINANCE: 1,
			db.ClaimStatusAPPROVED: 2, db.ClaimStatusREJECTED: 1, db.ClaimStatusWITHDRAWN: 1,
		}},
		{"finance", p.wanidaActor(), map[db.ClaimStatus]int{
			db.ClaimStatusSUBMITTED: 4, db.ClaimStatusRETURNED: 1, db.ClaimStatusPENDINGFINANCE: 2,
			db.ClaimStatusAPPROVED: 3, db.ClaimStatusREJECTED: 1, db.ClaimStatusWITHDRAWN: 1,
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := service.Summary(t.Context(), test.actor, Range{})
			if err != nil {
				t.Fatalf("Summary: %v", err)
			}

			if !maps.Equal(got.CountsByStatus, test.want) {
				t.Fatalf("countsByStatus = %v, want %v (six statuses, no DRAFT)", got.CountsByStatus, test.want)
			}
		})
	}
}

// F10-TEST-02 (F10-AC-02): countsByStatus has the six non-draft keys with 0 for a status with no claim, and it counts
// current statuses without the date range, so a range that excludes every decision leaves the counts as they are.
func TestF10TEST02CountsHaveSixKeysAndIgnoreTheRange(t *testing.T) {
	pool := pgtest.New(t)
	p := seedPeople(t, pool)
	scopeDataset(t, p, func(c *stored) {
		if c.status != db.ClaimStatusRETURNED {
			insertClaim(t, pool, c)
		}
	})
	service := NewService(pool)

	want := map[db.ClaimStatus]int{
		db.ClaimStatusSUBMITTED: 2, db.ClaimStatusRETURNED: 0, db.ClaimStatusPENDINGFINANCE: 1,
		db.ClaimStatusAPPROVED: 2, db.ClaimStatusREJECTED: 1, db.ClaimStatusWITHDRAWN: 1,
	}

	for name, rng := range map[string]Range{
		"no range":                {},
		"range after every claim": {From: new(at(12, 1, 0, 0, 0))},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := service.Summary(t.Context(), p.maleeActor(), rng)
			if err != nil {
				t.Fatalf("Summary: %v", err)
			}

			if !maps.Equal(got.CountsByStatus, want) {
				t.Fatalf("countsByStatus = %v, want %v", got.CountsByStatus, want)
			}
		})
	}
}
