package main

import (
	"testing"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// F06-TEST-01 (F06-AC-01, FR-09, FR-07, NFR-06): the assigned reviewer returns a SUBMITTED claim with a reason. The claim
// is RETURNED with return count 1 and version + 1, it has no decision time, the new ETag is sent, and one RETURN history
// entry holds the reviewer and the trimmed reason.
func TestF06TEST01ReviewerReturnsASubmittedClaim(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedBy(t, router, "req.anan")
	token := loginAs(t, router, "rev.malee")

	recorder := decide(t, router, token, id, "return", `"0"`, `{"comment": "  Please attach the receipt.  "}`)
	claim := decodeClaim(t, recorder).Data

	if got := recorder.Header().Get("ETag"); got != `"1"` {
		t.Fatalf("ETag = %q, want \"1\"", got)
	}

	if claim.Status != "RETURNED" || claim.ReturnCount != 1 || claim.Version != 1 || claim.DecidedAt != "" {
		t.Fatalf("claim = %+v, want RETURNED, returnCount 1, version 1, no decidedAt", claim)
	}

	if len(claim.History) != 2 {
		t.Fatalf("history = %+v, want 2 entries", claim.History)
	}

	last := claim.History[1]
	if last.Action != "RETURN" || last.FromStatus == nil || *last.FromStatus != "SUBMITTED" || last.ToStatus != "RETURNED" ||
		last.Actor.Name != "Malee S." || last.Comment == nil || *last.Comment != "Please attach the receipt." {
		t.Fatalf("last history = %+v, want RETURN SUBMITTED -> RETURNED by Malee S. with the trimmed reason", last)
	}

	if claim.AllowedActions == nil || len(claim.AllowedActions) != 0 {
		t.Fatalf("allowedActions = %v, want an empty list for the reviewer of a returned claim", claim.AllowedActions)
	}

	if state := readState(t, pool, id); state.Status != "RETURNED" || state.Version != 1 || state.Decided || state.History != 2 {
		t.Fatalf("stored = %+v, want RETURNED, version 1, not decided, 2 history rows", state)
	}
}
