package main

import (
	"net/http"
	"slices"
	"testing"
	"time"

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

// F06-TEST-03 (F06-AC-02, BR-04): a return with a body that has no comment is a 422 on comment, not a 400.
func TestF06TEST03ReturnWithoutAReasonIs422(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedBy(t, router, "req.anan")
	before := readState(t, pool, id)

	rec := decide(t, router, loginAs(t, router, "rev.malee"), id, "return", `"0"`, `{}`)

	decodeProblem(t, rec, http.StatusUnprocessableEntity, "VALIDATION_FAILED")

	if got := problemErrors(t, rec); len(got) != 1 || got[0].Field != "comment" || got[0].Rule != "BR-04" {
		t.Fatalf("errors = %+v, want one on comment for BR-04", got)
	}

	if after := readState(t, pool, id); after != before {
		t.Fatalf("stored = %+v after the refused return, want %+v", after, before)
	}
}

// F06-TEST-04 (F06-AC-03, BR-01, BR-02, ADR-0001): a reviewer who is not assigned cannot return a claim, and neither can
// its owner. The Finance Officer who is assigned to a department head's claim can.
func TestF06TEST04OnlyTheAssignedReviewerCanReturn(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	reason := `{"comment": "Please attach the receipt."}`

	assigned := submittedBy(t, router, "req.anan")      // assigned to rev.malee
	own := submittedBy(t, router, "rev.malee")          // assigned to rev.prasert
	headsClaim := submittedBy(t, router, "rev.prasert") // assigned to fin.wanida

	for name, refused := range map[string]struct{ user, id string }{
		"a reviewer who is not assigned": {"rev.somchai", assigned},
		"a reviewer on their own claim":  {"rev.malee", own},
		"the owner of the claim":         {"req.anan", assigned},
	} {
		t.Run(name, func(t *testing.T) {
			before := readState(t, pool, refused.id)

			decodeProblem(t, decide(t, router, loginAs(t, router, refused.user), refused.id, "return", `"0"`, reason), http.StatusForbidden, "FORBIDDEN")

			if after := readState(t, pool, refused.id); after != before {
				t.Fatalf("stored = %+v after the refused return, want %+v", after, before)
			}
		})
	}

	claim := decodeClaim(t, decide(t, router, loginAs(t, router, "fin.wanida"), headsClaim, "return", `"0"`, reason)).Data
	if last := claim.History[len(claim.History)-1]; claim.Status != "RETURNED" || last.Action != "RETURN" || last.Actor.Name != "Wanida" {
		t.Fatalf("claim = %s, last history = %+v, want RETURNED by Wanida", claim.Status, last)
	}
}

// F06-TEST-08 (F06-AC-06, FR-09, FR-10): a returned claim leaves the reviewer's queue and stays in its owner's list as
// RETURNED; the reviewer can still open it and has no action on it. After a resubmission it is back in the queue, ordered
// by its new submittedAt.
func TestF06TEST08ReturnedClaimLeavesTheQueueUntilResubmitted(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	reviewer := loginAs(t, router, "rev.malee")
	owner := loginAs(t, router, "req.anan")
	id := submittedBy(t, router, "req.anan")

	// An earlier submission, so a resubmission at the clock moves the claim in the queue.
	earlier := claimsNow.Add(-72 * time.Hour)
	if _, err := pool.Exec(t.Context(), `UPDATE claims SET submitted_at = $2 WHERE id = $1`, id, earlier); err != nil {
		t.Fatalf("move the submission time: %v", err)
	}

	queued := func() (ids []string, submittedAt string) {
		t.Helper()

		page := listClaims(t, router, reviewer, "?scope=review")
		for _, claim := range page.Data {
			ids = append(ids, claim.ID)
			submittedAt = claim.SubmittedAt
		}

		return ids, submittedAt
	}

	if ids, at := queued(); !slices.Equal(ids, []string{id}) || at != "2026-11-02T03:00:00Z" {
		t.Fatalf("queue before the return = %v at %s, want the claim submitted 2026-11-02T03:00:00Z", ids, at)
	}

	decodeClaim(t, decide(t, router, reviewer, id, "return", `"0"`, `{"comment": "Please attach the receipt."}`))

	if ids, _ := queued(); len(ids) != 0 {
		t.Fatalf("queue after the return = %v, want empty", ids)
	}

	if detail := openClaim(t, router, reviewer, id).Data; detail.Status != "RETURNED" || len(detail.AllowedActions) != 0 {
		t.Fatalf("detail = %s with actions %v, want RETURNED with none", detail.Status, detail.AllowedActions)
	}

	mine := listClaims(t, router, owner, "?scope=mine")
	if len(mine.Data) != 1 || mine.Data[0].ID != id || mine.Data[0].Status != "RETURNED" {
		t.Fatalf("my claims = %+v, want the claim as RETURNED", mine.Data)
	}

	decodeClaim(t, decide(t, router, owner, id, "submit", `"1"`, ""))

	if ids, at := queued(); !slices.Equal(ids, []string{id}) || at != "2026-11-05T03:00:00Z" {
		t.Fatalf("queue after the resubmission = %v at %s, want the claim submitted 2026-11-05T03:00:00Z", ids, at)
	}
}
