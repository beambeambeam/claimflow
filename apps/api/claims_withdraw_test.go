package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// withdrawMessage is the refusal of a withdrawal from a status that does not allow it (F-07 §1.5).
const withdrawMessage = "Only a submitted or returned claim can be withdrawn."

// F07-TEST-03 (F07-AC-03, BR-16): an owner's DRAFT, APPROVED, REJECTED, or WITHDRAWN claim cannot be withdrawn, with the
// current version: 409 INVALID_TRANSITION for BR-16 with the message of F-07 §1.5, and the claim is unchanged.
func TestF07TEST03OnlyASubmittedOrReturnedClaimCanBeWithdrawnBR16(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "req.anan")

	for _, status := range []string{"DRAFT", "APPROVED", "REJECTED", "WITHDRAWN"} {
		t.Run(status, func(t *testing.T) {
			id := storedClaim(t, pool, "req.anan", status, claimsNow).String()
			before := readState(t, pool, id)

			expectTransition(t, decide(t, router, token, id, "withdraw", `"0"`, ""), "BR-16", withdrawMessage)

			if after := readState(t, pool, id); after != before {
				t.Fatalf("stored = %+v after the refused withdrawal, want %+v", after, before)
			}
		})
	}
}

// F07-TEST-01 (F07-AC-01, FR-11, FR-07, NFR-06): the owner withdraws a SUBMITTED claim with no body, and a RETURNED claim
// with a comment. Each is WITHDRAWN with version + 1, the new ETag, no decision time, and one WITHDRAW history entry that
// holds the owner and the trimmed comment, or no comment.
func TestF07TEST01OwnerWithdrawsASubmittedOrReturnedClaim(t *testing.T) {
	t.Run("a submitted claim, with no body", func(t *testing.T) {
		pool := pgtest.New(t)
		router := seededClaimRouter(t, pool)
		id := submittedBy(t, router, "req.anan")

		recorder := decide(t, router, loginAs(t, router, "req.anan"), id, "withdraw", `"0"`, "")
		claim := decodeClaim(t, recorder).Data

		if got := recorder.Header().Get("ETag"); got != `"1"` {
			t.Fatalf("ETag = %q, want \"1\"", got)
		}

		if claim.Status != "WITHDRAWN" || claim.Version != 1 || claim.DecidedAt != "" {
			t.Fatalf("claim = %+v, want WITHDRAWN, version 1, no decidedAt", claim)
		}

		if len(claim.History) != 2 {
			t.Fatalf("history = %+v, want 2 entries", claim.History)
		}

		last := claim.History[1]
		if last.Action != "WITHDRAW" || last.FromStatus == nil || *last.FromStatus != "SUBMITTED" || last.ToStatus != "WITHDRAWN" ||
			last.Actor.Name != "Anan K." || last.Comment != nil {
			t.Fatalf("last history = %+v, want WITHDRAW SUBMITTED -> WITHDRAWN by Anan K. with no comment", last)
		}

		if claim.AllowedActions == nil || len(claim.AllowedActions) != 0 {
			t.Fatalf("allowedActions = %v, want an empty list", claim.AllowedActions)
		}

		if state := readState(t, pool, id); state.Status != "WITHDRAWN" || state.Version != 1 || state.Decided || state.History != 2 {
			t.Fatalf("stored = %+v, want WITHDRAWN, version 1, not decided, 2 history rows", state)
		}
	})

	t.Run("a returned claim, with a comment", func(t *testing.T) {
		pool := pgtest.New(t)
		router := seededClaimRouter(t, pool)
		id := submittedBy(t, router, "req.anan")

		returned := decide(t, router, loginAs(t, router, "rev.malee"), id, "return", `"0"`, `{"comment": "Please attach the receipt."}`)
		if returned.Code != 200 || returned.Header().Get("ETag") != `"1"` {
			t.Fatalf("return = %d with ETag %q, want 200 with \"1\"", returned.Code, returned.Header().Get("ETag"))
		}

		recorder := decide(t, router, loginAs(t, router, "req.anan"), id, "withdraw", `"1"`, `{"comment": "  No longer needed  "}`)
		claim := decodeClaim(t, recorder).Data

		if got := recorder.Header().Get("ETag"); got != `"2"` {
			t.Fatalf("ETag = %q, want \"2\"", got)
		}

		if claim.Status != "WITHDRAWN" || claim.Version != 2 || claim.DecidedAt != "" || len(claim.History) != 3 {
			t.Fatalf("claim = %+v, want WITHDRAWN, version 2, no decidedAt, 3 history entries", claim)
		}

		last := claim.History[2]
		if last.Action != "WITHDRAW" || last.FromStatus == nil || *last.FromStatus != "RETURNED" || last.ToStatus != "WITHDRAWN" ||
			last.Actor.Name != "Anan K." || last.Comment == nil || *last.Comment != "No longer needed" {
			t.Fatalf("last history = %+v, want WITHDRAW RETURNED -> WITHDRAWN by Anan K. with the trimmed comment", last)
		}

		if state := readState(t, pool, id); state.Status != "WITHDRAWN" || state.Version != 2 || state.Decided || state.History != 3 {
			t.Fatalf("stored = %+v, want WITHDRAWN, version 2, not decided, 3 history rows", state)
		}
	})
}

// F07-TEST-02 (F07-AC-02, FR-11): the comment is optional. An empty object stores no comment, 500 characters are accepted,
// and 501 are a 400 from the contract, not a 422, with the claim unchanged.
func TestF07TEST02WithdrawCommentIsOptionalWithAtMost500Characters(t *testing.T) {
	for _, test := range []struct {
		name    string
		body    string
		comment string // empty: no comment is stored
	}{
		{"an empty object", `{}`, ""},
		{"500 characters", `{"comment": "` + strings.Repeat("a", 500) + `"}`, strings.Repeat("a", 500)},
	} {
		t.Run(test.name, func(t *testing.T) {
			pool := pgtest.New(t)
			router := seededClaimRouter(t, pool)
			id := submittedBy(t, router, "req.anan")

			claim := decodeClaim(t, decide(t, router, loginAs(t, router, "req.anan"), id, "withdraw", `"0"`, test.body)).Data

			last := claim.History[len(claim.History)-1]
			if claim.Status != "WITHDRAWN" || (test.comment == "") != (last.Comment == nil) || (last.Comment != nil && *last.Comment != test.comment) {
				t.Fatalf("claim = %s with comment %v, want WITHDRAWN with comment %q", claim.Status, last.Comment, test.comment)
			}
		})
	}

	t.Run("501 characters", func(t *testing.T) {
		pool := pgtest.New(t)
		router := seededClaimRouter(t, pool)
		id := submittedBy(t, router, "req.anan")
		before := readState(t, pool, id)

		decodeProblem(t, decide(t, router, loginAs(t, router, "req.anan"), id, "withdraw", `"0"`, `{"comment": "`+strings.Repeat("a", 501)+`"}`),
			http.StatusBadRequest, "BAD_REQUEST")

		if after := readState(t, pool, id); after != before {
			t.Fatalf("stored = %+v after the refused withdrawal, want %+v", after, before)
		}
	})
}

// F07-TEST-04 (F07-AC-04, FR-11, NFR-02): only the owner can withdraw. The assigned reviewer gets 403, and so does another
// requester, with no claim data. The claim is unchanged.
func TestF07TEST04OnlyTheOwnerCanWithdraw(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedBy(t, router, "req.anan")
	before := readState(t, pool, id)

	for _, username := range []string{"rev.malee", "req.ploy"} {
		recorder := decide(t, router, loginAs(t, router, username), id, "withdraw", `"0"`, "")
		decodeProblem(t, recorder, http.StatusForbidden, "FORBIDDEN")

		for _, leaked := range []string{`"data"`, "claimNo", "Bangkok workshop travel"} {
			if strings.Contains(recorder.Body.String(), leaked) {
				t.Fatalf("%s: body holds %q: %s", username, leaked, recorder.Body)
			}
		}
	}

	if after := readState(t, pool, id); after != before {
		t.Fatalf("stored = %+v after the refused withdrawals, want %+v", after, before)
	}
}

// F07-TEST-06 (F07-AC-05, BR-13): a stale version is a 409 CONFLICT, a missing If-Match is a 428, and a malformed one is a
// 400. Each leaves the claim as it was.
func TestF07TEST06WithdrawNeedsTheCurrentIfMatch(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedBy(t, router, "req.anan")
	token := loginAs(t, router, "req.anan")
	before := readState(t, pool, id)

	decodeProblem(t, decide(t, router, token, id, "withdraw", `"1"`, ""), http.StatusConflict, "CONFLICT")
	decodeProblem(t, decide(t, router, token, id, "withdraw", "", ""), http.StatusPreconditionRequired, "PRECONDITION_REQUIRED")
	decodeProblem(t, decide(t, router, token, id, "withdraw", "abc", ""), http.StatusBadRequest, "BAD_REQUEST")

	if after := readState(t, pool, id); after != before {
		t.Fatalf("stored = %+v after the refused withdrawals, want %+v", after, before)
	}
}

// F07-TEST-07 (F07-AC-06, BR-03, BR-13, BR-10): WITHDRAWN is final. The reviewer's approve, reject, and return get 409
// CONFLICT with the version read before the withdrawal and 409 INVALID_TRANSITION for BR-03 with the current one; the
// owner's submit, edit, and delete get 409 INVALID_TRANSITION for BR-03, BR-10, and BR-10.
func TestF07TEST07AWithdrawnClaimCannotBeActedOnAgain(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedBy(t, router, "req.anan")
	owner := loginAs(t, router, "req.anan")
	reviewer := loginAs(t, router, "rev.malee")

	if claim := decodeClaim(t, decide(t, router, owner, id, "withdraw", `"0"`, "")).Data; claim.Version != 1 {
		t.Fatalf("version = %d after the withdrawal, want 1", claim.Version)
	}

	before := readState(t, pool, id)
	reason := `{"comment": "Please attach the receipt."}`

	for _, step := range []struct{ action, body, message string }{
		{"approve", "", "Only a submitted claim can be approved or rejected."},
		{"reject", reason, "Only a submitted claim can be approved or rejected."},
		{"return", reason, "Only a submitted claim can be returned."},
	} {
		decodeProblem(t, decide(t, router, reviewer, id, step.action, `"0"`, step.body), http.StatusConflict, "CONFLICT")
		expectTransition(t, decide(t, router, reviewer, id, step.action, `"1"`, step.body), "BR-03", step.message)
	}

	path := "/api/v1/claims/" + id
	expectTransition(t, decide(t, router, owner, id, "submit", `"1"`, ""), "BR-03", "Only a draft or a returned claim can be submitted.")
	expectTransition(t, draftRequest(t, router, owner, http.MethodPut, path, `"1"`, threeItemsBody()), "BR-10", "Only a draft or a returned claim can be edited.")
	expectTransition(t, draftRequest(t, router, owner, http.MethodDelete, path, `"1"`, ""), "BR-10", "Only a draft can be deleted.")

	if after := readState(t, pool, id); after != before {
		t.Fatalf("stored = %+v after the refused steps, want %+v", after, before)
	}
}

// F07-TEST-10 (F07-AC-08, FR-11): a withdrawn claim leaves the reviewer's review list. The owner and the assigned reviewer
// can still open it and see the WITHDRAW entry, and neither has an action on it.
func TestF07TEST10AWithdrawnClaimLeavesTheReviewQueueButStaysReadable(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedBy(t, router, "req.anan")
	owner := loginAs(t, router, "req.anan")
	reviewer := loginAs(t, router, "rev.malee")

	if queue := listClaims(t, router, reviewer, "?scope=review"); queue.Total != 1 {
		t.Fatalf("review queue total = %d before the withdrawal, want 1", queue.Total)
	}

	decodeClaim(t, decide(t, router, owner, id, "withdraw", `"0"`, `{"comment": "No longer needed"}`))

	if queue := listClaims(t, router, reviewer, "?scope=review"); queue.Total != 0 || len(queue.Data) != 0 {
		t.Fatalf("review queue = %+v after the withdrawal, want it empty", queue)
	}

	for name, token := range map[string]string{"owner": owner, "reviewer": reviewer} {
		claim := openClaim(t, router, token, id).Data
		last := claim.History[len(claim.History)-1]

		if claim.Status != "WITHDRAWN" || last.Action != "WITHDRAW" || last.Comment == nil || *last.Comment != "No longer needed" ||
			claim.AllowedActions == nil || len(claim.AllowedActions) != 0 {
			t.Fatalf("%s sees %s with last history %+v and actions %v, want WITHDRAWN, the WITHDRAW entry, and no actions", name, claim.Status, last, claim.AllowedActions)
		}
	}
}
