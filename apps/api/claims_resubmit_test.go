package main

import (
	"net/http"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// returnedByMalee has req.anan submit a claim, moves its first submission to 2026-11-20 09:00 Bangkok, has rev.malee return
// it, and returns its id. The router clock is 2026-11-25 10:00 Bangkok.
func returnedByMalee(t *testing.T, router http.Handler, pool *pgxpool.Pool) string {
	t.Helper()

	id := submittedBy(t, router, "req.anan")

	if _, err := pool.Exec(t.Context(), `UPDATE claims SET first_submitted_at = '2026-11-20T09:00:00+07:00', submitted_at = '2026-11-20T09:00:00+07:00' WHERE id = $1`, id); err != nil {
		t.Fatalf("move the first submission: %v", err)
	}

	decodeClaim(t, decide(t, router, loginAs(t, router, "rev.malee"), id, "return", `"0"`, `{"comment": "Please attach the receipt."}`))

	return id
}

// F06-TEST-09 (F06-AC-07, FR-10, BR-17, NFR-06): the owner resubmits a RETURNED claim after the owner's reviewer changed. The
// claim is SUBMITTED with submittedAt = the clock, firstSubmittedAt and the assigned reviewer unchanged, version + 1 with the
// new ETag, and one RESUBMIT entry from RETURNED.
func TestF06TEST09OwnerResubmitsToTheSameReviewer(t *testing.T) {
	pool := pgtest.New(t)
	router := draftRouter(t, pool)
	id := returnedByMalee(t, router, pool)

	if _, err := pool.Exec(t.Context(), `UPDATE "user" SET reviewer_id = (SELECT id FROM "user" WHERE username = 'rev.somchai') WHERE username = 'req.anan'`); err != nil {
		t.Fatalf("change the owner's reviewer: %v", err)
	}

	before := readState(t, pool, id)

	rec := decide(t, router, loginAs(t, router, "req.anan"), id, "submit", `"1"`, "")
	claim := decodeClaim(t, rec).Data

	if got := rec.Header().Get("ETag"); got != `"2"` {
		t.Fatalf("ETag = %q, want \"2\"", got)
	}

	if claim.Status != "SUBMITTED" || claim.Version != 2 || claim.SubmittedAt != "2026-11-25T03:00:00Z" ||
		claim.FirstSubmittedAt != "2026-11-20T02:00:00Z" || claim.AssignedReviewer.Name != "Malee S." {
		t.Fatalf("claim = %+v, want SUBMITTED, version 2, submitted 2026-11-25T03:00:00Z, first 2026-11-20T02:00:00Z, reviewer Malee S.", claim)
	}

	if len(claim.History) != before.History+1 {
		t.Fatalf("history has %d entries, want %d", len(claim.History), before.History+1)
	}

	last := claim.History[len(claim.History)-1]
	if last.Action != "RESUBMIT" || last.FromStatus == nil || *last.FromStatus != "RETURNED" || last.ToStatus != "SUBMITTED" || last.Comment != nil {
		t.Fatalf("last history = %+v, want RESUBMIT RETURNED -> SUBMITTED with no comment", last)
	}
}

// F06-TEST-11 (F06-AC-09, FR-10, FR-03, BR-10): the owner edits a RETURNED claim with the F-05 edit path and gets the new
// total. Another requester cannot edit it, and the reviewer cannot resubmit it. Both are 403.
func TestF06TEST11OnlyTheOwnerEditsAndResubmitsAReturnedClaim(t *testing.T) {
	pool := pgtest.New(t)
	router := draftRouter(t, pool)
	id := returnedByMalee(t, router, pool)
	path := "/api/v1/claims/" + id
	body := `{"title":"Edited title","purpose":"Edited purpose","items":[` +
		itemJSON("OTHER", "2026-11-05", "100.00") + `,` + itemJSON("TRAVEL", "2026-11-06", "200.00") + `]}`

	edited := expectDraftResponse(t, draftRequest(t, router, loginAs(t, router, "req.anan"), http.MethodPut, path, `"1"`, body), http.StatusOK)
	if edited.Status != api.ClaimStatusReturned || edited.Total != 30000 || len(edited.Items) != 2 || edited.Version != 2 ||
		!slices.Contains(edited.AllowedActions, "submit") {
		t.Fatalf("edited claim = %+v, want RETURNED with total 30000, 2 items, version 2, and submit allowed", edited)
	}

	before := readState(t, pool, id)

	decodeProblem(t, draftRequest(t, router, loginAs(t, router, "req.ploy"), http.MethodPut, path, `"2"`, body), http.StatusForbidden, "FORBIDDEN")
	decodeProblem(t, decide(t, router, loginAs(t, router, "rev.malee"), id, "submit", `"2"`, ""), http.StatusForbidden, "FORBIDDEN")

	if after := readState(t, pool, id); after != before {
		t.Fatalf("stored = %+v after the refused requests, want %+v", after, before)
	}
}
