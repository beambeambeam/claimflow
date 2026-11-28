package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// pendingFinanceMessage is the BR-03 message of a final decision on a claim that is not PENDING_FINANCE (F-08 §1.5).
const pendingFinanceMessage = "Only a claim pending finance can be finally approved or rejected."

// F08-TEST-06 (F08-AC-05): fin.wanida final-approves one PENDING_FINANCE claim with no body and final-rejects another
// with a reason. Each is decided, with one more version and a decision time, and its history reads SUBMIT, APPROVE,
// then the final row with the comment, or none, and the trimmed reason.
func TestF08TEST06FinalApproveAndFinalReject(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	wanida := loginAs(t, router, "fin.wanida")

	for _, test := range []struct {
		action, body, status, history string
		comment                       *string
	}{
		{"final-approve", "", "APPROVED", "FINAL_APPROVE", nil},
		{"final-reject", `{"comment": "  Budget code is wrong  "}`, "REJECTED", "FINAL_REJECT", new("Budget code is wrong")},
	} {
		t.Run(test.action, func(t *testing.T) {
			id, version := pendingFinanceClaim(t, router)

			recorder := decide(t, router, wanida, id, test.action, ifMatch(version), test.body)
			decided := decodeClaim(t, recorder)

			if decided.Data.Status != test.status || decided.Data.DecidedAt == "" || decided.Data.Version != version+1 {
				t.Fatalf("claim = %s, decidedAt %q, version %d; want %s, decided, version %d",
					decided.Data.Status, decided.Data.DecidedAt, decided.Data.Version, test.status, version+1)
			}

			if got := recorder.Header().Get("ETag"); got != ifMatch(version+1) {
				t.Fatalf("ETag = %q, want %s", got, ifMatch(version+1))
			}

			history := decided.Data.History
			if len(history) != 3 || history[0].Action != "SUBMIT" || history[1].Action != "APPROVE" || history[2].Action != test.history {
				t.Fatalf("history = %+v, want SUBMIT, APPROVE, %s", history, test.history)
			}

			last := history[2]
			if last.FromStatus == nil || *last.FromStatus != "PENDING_FINANCE" || last.ToStatus != test.status ||
				(last.Comment == nil) != (test.comment == nil) || (last.Comment != nil && *last.Comment != *test.comment) ||
				last.Actor.Name != "Wanida" {
				t.Fatalf("final history = %+v, want from PENDING_FINANCE to %s by Wanida with comment %v", last, test.status, test.comment)
			}
		})
	}
}

// F08-TEST-07 (F08-AC-06, BR-04): a final reject needs a reason of 10 to 500 characters after trimming; a final
// approve takes no comment or one of up to 500 characters.
func TestF08TEST07FinalDecisionCommentRule(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	wanida := loginAs(t, router, "fin.wanida")

	for _, test := range []struct {
		name, action, comment string
		status                int
	}{
		{"final reject, blank", "final-reject", "   ", http.StatusUnprocessableEntity},
		{"final reject, 9 characters", "final-reject", strings.Repeat("a", 9), http.StatusUnprocessableEntity},
		{"final reject, 10 characters", "final-reject", strings.Repeat("a", 10), http.StatusOK},
		{"final reject, 500 characters", "final-reject", strings.Repeat("a", 500), http.StatusOK},
		{"final reject, 501 characters", "final-reject", strings.Repeat("a", 501), http.StatusUnprocessableEntity},
		{"final approve, 500 characters", "final-approve", strings.Repeat("a", 500), http.StatusOK},
		{"final approve, 501 characters", "final-approve", strings.Repeat("a", 501), http.StatusUnprocessableEntity},
	} {
		t.Run(test.name, func(t *testing.T) {
			id, version := pendingFinanceClaim(t, router)
			recorder := decide(t, router, wanida, id, test.action, ifMatch(version), `{"comment": "`+test.comment+`"}`)

			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d; body = %s", recorder.Code, test.status, recorder.Body)
			}

			if test.status != http.StatusOK {
				decodeProblem(t, recorder, http.StatusUnprocessableEntity, "VALIDATION_FAILED")

				if got := problemErrors(t, recorder); len(got) != 1 || got[0].Field != "comment" || got[0].Rule != "BR-04" {
					t.Fatalf("errors = %+v, want one BR-04 error on comment", got)
				}

				if state := readState(t, pool, id); state.Status != "PENDING_FINANCE" || state.Version != version {
					t.Fatalf("claim = %+v, want it unchanged", state)
				}
			}
		})
	}
}

// F08-TEST-08 (F08-AC-07, BR-01, BR-02): only a Finance Officer who is not the requester makes a final decision. The
// assigned reviewer, another reviewer, and the owner get 403, and so does a Finance Officer on her own claim.
func TestF08TEST08OnlyFinanceMakesTheFinalDecision(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id, version := pendingFinanceClaim(t, router)
	before := readState(t, pool, id)

	for _, username := range []string{"rev.malee", "rev.somchai", "req.anan"} {
		token := loginAs(t, router, username)

		for _, action := range []string{"final-approve", "final-reject"} {
			recorder := decide(t, router, token, id, action, ifMatch(version), `{"comment": "Budget code is wrong"}`)
			decodeProblem(t, recorder, http.StatusForbidden, "FORBIDDEN")
		}
	}

	if after := readState(t, pool, id); after != before {
		t.Fatalf("claim = %+v, want %+v", after, before)
	}

	// Finance cannot create claims and fin.wanida has no reviewer, so the test stores rev.prasert's claim, which is
	// assigned to her, and swaps the two: she becomes the requester (BR-01).
	stored := storedClaim(t, pool, "rev.prasert", "PENDING_FINANCE", claimsNow.Add(-time.Hour))
	if _, err := pool.Exec(t.Context(),
		`UPDATE claims SET requester_id = assigned_reviewer_id, assigned_reviewer_id = requester_id WHERE id = $1`, stored); err != nil {
		t.Fatalf("make the claim fin.wanida's: %v", err)
	}

	own := stored.String()
	for _, action := range []string{"final-approve", "final-reject"} {
		recorder := decide(t, router, loginAs(t, router, "fin.wanida"), own, action, ifMatch(0), `{"comment": "Budget code is wrong"}`)
		decodeProblem(t, recorder, http.StatusForbidden, "FORBIDDEN")
	}
}

// F08-TEST-10 (F08-AC-08, BR-03, BR-13): a final decision on a claim that is not PENDING_FINANCE is 409
// INVALID_TRANSITION with the F-08 message; a stale version is 409 CONFLICT, no If-Match 428, and a malformed one 400.
func TestF08TEST10FinalDecisionNeedsPendingFinanceAndTheCurrentVersion(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	wanida := loginAs(t, router, "fin.wanida")

	for _, status := range []string{"SUBMITTED", "APPROVED", "REJECTED"} {
		id := storedClaim(t, pool, "req.anan", status, claimsNow.Add(-time.Hour)).String()

		for _, action := range []string{"final-approve", "final-reject"} {
			expectTransition(t, decide(t, router, wanida, id, action, ifMatch(0), `{"comment": "Budget code is wrong"}`), "BR-03", pendingFinanceMessage)
		}
	}

	id, version := pendingFinanceClaim(t, router)
	decodeProblem(t, decide(t, router, wanida, id, "final-approve", ifMatch(version+1), ""), http.StatusConflict, "CONFLICT")
	decodeProblem(t, decide(t, router, wanida, id, "final-approve", "", ""), http.StatusPreconditionRequired, "PRECONDITION_REQUIRED")
	decodeProblem(t, decide(t, router, wanida, id, "final-approve", "abc", ""), http.StatusBadRequest, "BAD_REQUEST")

	if state := readState(t, pool, id); state.Status != "PENDING_FINANCE" || state.Version != version {
		t.Fatalf("claim = %+v, want it unchanged", state)
	}
}

// F08-TEST-13 (F08-AC-10): scope=finance lists every PENDING_FINANCE claim, oldest submission first, with the F-03
// paging rules. A reviewer and a requester get 403.
func TestF08TEST13FinanceQueueListsPendingFinanceOldestFirst(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	t1 := time.Date(2026, time.November, 1, 3, 0, 0, 0, time.UTC)

	newest := storedClaim(t, pool, "req.anan", "PENDING_FINANCE", t1.Add(48*time.Hour))
	oldest := storedClaim(t, pool, "req.niran", "PENDING_FINANCE", t1)
	middle := storedClaim(t, pool, "rev.malee", "PENDING_FINANCE", t1.Add(24*time.Hour))
	storedClaim(t, pool, "req.ploy", "SUBMITTED", t1)

	wanida := loginAs(t, router, "fin.wanida")
	page := listClaims(t, router, wanida, "?scope=finance")

	want := []uuid.UUID{oldest, middle, newest}
	if page.Total != 3 || len(page.Data) != 3 {
		t.Fatalf("page = %+v, want 3 claims", page)
	}

	for i, claim := range page.Data {
		if claim.ID != want[i].String() || claim.Status != "PENDING_FINANCE" {
			t.Fatalf("data[%d] = %+v, want claim %s", i, claim, want[i])
		}
	}

	second := listClaims(t, router, wanida, "?scope=finance&pageSize=2&page=2")
	if second.Total != 3 || len(second.Data) != 1 || second.Data[0].ID != newest.String() {
		t.Fatalf("page 2 = %+v, want the newest claim and total 3", second)
	}

	for _, username := range []string{"rev.malee", "req.anan"} {
		recorder := getWithSession(t, router, "/api/v1/claims?scope=finance", loginAs(t, router, username))
		decodeProblem(t, recorder, http.StatusForbidden, "FORBIDDEN")
	}
}

// fullySeededRouter runs the seed, sample claims included, and returns a router on the claim clock.
func fullySeededRouter(t *testing.T, pool *pgxpool.Pool) http.Handler {
	t.Helper()

	if err := seed(t.Context(), pool.Config().ConnString(), seedPassword, testBcryptCost, io.Discard); err != nil {
		t.Fatalf("seed: %v", err)
	}

	return newRouterAt(discardLogger(), testEnv(0), pool, func() time.Time { return claimsNow })
}

// seededClaimID returns the id of the seeded claim with claimNo.
func seededClaimID(t *testing.T, pool *pgxpool.Pool, claimNo string) string {
	t.Helper()

	var id uuid.UUID
	if err := pool.QueryRow(t.Context(), `SELECT id FROM claims WHERE claim_no = $1`, claimNo).Scan(&id); err != nil {
		t.Fatalf("find %s: %v", claimNo, err)
	}

	return id.String()
}

// F08-TEST-14 (F08-AC-11, NFR-02): fin.wanida sees a claim that is not a DRAFT but gets no action on it, gets 403 with
// no claim data on a DRAFT, and cannot approve a claim assigned to another reviewer.
func TestF08TEST14FinanceSeesNonDraftClaimsWithoutActions(t *testing.T) {
	pool := pgtest.New(t)
	router := fullySeededRouter(t, pool)
	wanida := loginAs(t, router, "fin.wanida")

	rejected := openClaim(t, router, wanida, seededClaimID(t, pool, "CLM-2026-900003"))
	if rejected.Data.Status != "REJECTED" || rejected.Data.Requester.Name != "Ploy" {
		t.Fatalf("claim = %+v, want req.ploy's REJECTED claim", rejected.Data)
	}

	if actions := openClaimActions(t, router, wanida, seededClaimID(t, pool, "CLM-2026-900003")); len(actions) != 0 {
		t.Fatalf("allowedActions = %v, want none", actions)
	}

	draft := getWithSession(t, router, "/api/v1/claims/"+seededClaimID(t, pool, "CLM-2026-900007"), wanida)
	decodeProblem(t, draft, http.StatusForbidden, "FORBIDDEN")

	if strings.Contains(draft.Body.String(), "Office supplies") {
		t.Fatalf("body = %s, want no claim data", draft.Body)
	}

	submitted := seededClaimID(t, pool, "CLM-2026-900001")
	before := readState(t, pool, submitted)

	for _, action := range []string{"approve", "reject", "return"} {
		recorder := decide(t, router, wanida, submitted, action, ifMatch(before.Version), `{"comment": "Receipt is missing."}`)
		decodeProblem(t, recorder, http.StatusForbidden, "FORBIDDEN")
	}

	if after := readState(t, pool, submitted); after != before {
		t.Fatalf("claim = %+v, want %+v", after, before)
	}
}
