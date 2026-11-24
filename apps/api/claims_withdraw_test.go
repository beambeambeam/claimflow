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
