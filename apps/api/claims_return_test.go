package main

import (
	"net/http"
	"slices"
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

// F06-TEST-05 (F06-AC-04, BR-03, BR-13): an APPROVED, REJECTED, or RETURNED claim cannot be returned: 409
// INVALID_TRANSITION for BR-03 with the current version, and 409 CONFLICT with a stale one. A draft is not the
// reviewer's to see, so it is 403. Each refusal leaves the claim as it was.
func TestF06TEST05OnlyASubmittedClaimCanBeReturned(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "rev.malee")
	reason := `{"comment": "Please attach the receipt."}`

	for _, status := range []string{"APPROVED", "REJECTED", "RETURNED"} {
		t.Run(status, func(t *testing.T) {
			id := storedClaim(t, pool, "req.anan", status, claimsNow).String()
			before := readState(t, pool, id)

			expectTransition(t, decide(t, router, token, id, "return", `"0"`, reason), "BR-03", "Only a submitted claim can be returned.")

			if after := readState(t, pool, id); after != before {
				t.Fatalf("stored = %+v after the refused return, want %+v", after, before)
			}
		})
	}

	t.Run("a stale version is a conflict", func(t *testing.T) {
		id := storedClaim(t, pool, "req.anan", "APPROVED", claimsNow).String()

		decodeProblem(t, decide(t, router, token, id, "return", `"7"`, reason), http.StatusConflict, "CONFLICT")
	})

	t.Run("a draft is forbidden", func(t *testing.T) {
		id := storedClaim(t, pool, "req.anan", "DRAFT", claimsNow).String()

		decodeProblem(t, decide(t, router, token, id, "return", `"0"`, reason), http.StatusForbidden, "FORBIDDEN")
	})
}

// F06-AC-05 at the HTTP seam, with F06-TEST-07: a SUBMITTED claim with return count 3 cannot be returned, 409
// INVALID_TRANSITION for BR-14 on returnCount. The reviewer's allowedActions have no return, and approve still works.
func TestF06TEST07ReturnLimitIs409AndApproveStillWorks(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "rev.malee")
	id := submittedBy(t, router, "req.anan")

	if _, err := pool.Exec(t.Context(), `UPDATE claims SET return_count = 3 WHERE id = $1`, id); err != nil {
		t.Fatalf("set return count: %v", err)
	}

	if claim := openClaim(t, router, token, id).Data; claim.ReturnCount != 3 || !slices.Equal(claim.AllowedActions, []string{"approve", "reject"}) {
		t.Fatalf("claim = %+v, want return count 3 and approve, reject", claim)
	}

	before := readState(t, pool, id)
	rec := decide(t, router, token, id, "return", `"0"`, `{"comment": "One more thing."}`)

	decodeProblem(t, rec, http.StatusConflict, "INVALID_TRANSITION")

	want := []fieldError{{Field: "returnCount", Rule: "BR-14", Message: "This claim has already been returned 3 times."}}
	if got := problemErrors(t, rec); !slices.Equal(got, want) {
		t.Fatalf("errors = %+v, want %+v", got, want)
	}

	if after := readState(t, pool, id); after != before {
		t.Fatalf("stored = %+v after the refused return, want %+v", after, before)
	}

	if claim := decodeClaim(t, decide(t, router, token, id, "approve", `"0"`, "")).Data; claim.Status != "APPROVED" {
		t.Fatalf("status = %s, want APPROVED", claim.Status)
	}
}
