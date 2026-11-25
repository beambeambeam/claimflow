package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// claimOfAmounts is a claim body whose TRAVEL items have the given amounts, such as "15000.00".
func claimOfAmounts(amounts ...string) string {
	items := make([]string, 0, len(amounts))
	for _, amount := range amounts {
		items = append(items, itemJSON("TRAVEL", "2026-11-03", amount))
	}

	return claimBodyWith(strings.Join(items, ","))
}

// submittedOf has username submit body and returns the claim id and its version.
func submittedOf(t *testing.T, router http.Handler, username, body string) (id string, version int) {
	t.Helper()

	recorder := postClaim(t, router, loginAs(t, router, username), body)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("%s submit status = %d, want 201; body = %s", username, recorder.Code, recorder.Body)
	}

	var created claimResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	return created.Data.ID, created.Data.Version
}

// ifMatch is the If-Match value of version.
func ifMatch(version int) string {
	return strconv.Quote(strconv.Itoa(version))
}

// pendingFinanceClaim has req.anan submit a 15,000.00 claim and rev.malee approve it, and returns the claim id and its
// version once it is PENDING_FINANCE.
func pendingFinanceClaim(t *testing.T, router http.Handler) (id string, version int) {
	t.Helper()

	id, submitted := submittedOf(t, router, "req.anan", claimOfAmounts("15000.00"))
	approved := decodeClaim(t, decide(t, router, loginAs(t, router, "rev.malee"), id, "approve", ifMatch(submitted), ""))

	if approved.Data.Status != "PENDING_FINANCE" {
		t.Fatalf("status after approve = %s, want PENDING_FINANCE", approved.Data.Status)
	}

	return id, approved.Data.Version
}

// F08-TEST-01 (F08-AC-01, BR-09): the assigned reviewer, who is not a Finance Officer, approves a 15,000.00 claim. It
// goes to PENDING_FINANCE with no decision time, one more version, and one APPROVE history row to PENDING_FINANCE.
func TestF08TEST01HighValueApprovalGoesToPendingFinance(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)

	id, version := submittedOf(t, router, "req.anan", claimOfAmounts("15000.00"))
	recorder := decide(t, router, loginAs(t, router, "rev.malee"), id, "approve", ifMatch(version), `{"comment": "Fine by me"}`)
	approved := decodeClaim(t, recorder)

	if approved.Data.Status != "PENDING_FINANCE" || approved.Data.DecidedAt != "" || approved.Data.Version != version+1 {
		t.Fatalf("claim = %s, decidedAt %q, version %d; want PENDING_FINANCE, no decidedAt, version %d",
			approved.Data.Status, approved.Data.DecidedAt, approved.Data.Version, version+1)
	}

	if got := recorder.Header().Get("ETag"); got != ifMatch(version+1) {
		t.Fatalf("ETag = %q, want %s", got, ifMatch(version+1))
	}

	history := approved.Data.History
	last := history[len(history)-1]

	if len(history) != 2 || last.Action != "APPROVE" || last.FromStatus == nil || *last.FromStatus != "SUBMITTED" ||
		last.ToStatus != "PENDING_FINANCE" || last.Comment == nil || *last.Comment != "Fine by me" {
		t.Fatalf("history = %+v, want SUBMIT, then APPROVE from SUBMITTED to PENDING_FINANCE with the comment", history)
	}

	if state := readState(t, pool, id); state.Status != "PENDING_FINANCE" || state.Decided {
		t.Fatalf("stored claim = %+v, want PENDING_FINANCE and not decided", state)
	}
}

// F08-TEST-03 (F08-AC-02, BR-09): a claim of exactly 10,000.00 needs one approval; one of 10,000.01 goes to Finance.
func TestF08TEST03ThresholdIsMoreThanTenThousand(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	malee := loginAs(t, router, "rev.malee")

	for _, test := range []struct {
		amount, status string
		decided        bool
	}{
		{"10000.00", "APPROVED", true},
		{"10000.01", "PENDING_FINANCE", false},
	} {
		id, version := submittedOf(t, router, "req.anan", claimOfAmounts(test.amount))
		approved := decodeClaim(t, decide(t, router, malee, id, "approve", ifMatch(version), ""))

		if approved.Data.Status != test.status || (approved.Data.DecidedAt != "") != test.decided {
			t.Fatalf("%s: status %s, decidedAt %q; want %s, decided %t",
				test.amount, approved.Data.Status, approved.Data.DecidedAt, test.status, test.decided)
		}
	}
}

// F08-TEST-04 (F08-AC-03, ADR-0001): fin.wanida is the assigned reviewer of rev.prasert's 40,000.00 claim. Her approval
// is final: no second stage.
func TestF08TEST04FinanceReviewerApprovalIsFinal(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)

	id, version := submittedOf(t, router, "rev.prasert", claimOfAmounts("20000.00", "20000.00"))
	approved := decodeClaim(t, decide(t, router, loginAs(t, router, "fin.wanida"), id, "approve", ifMatch(version), ""))

	if approved.Data.Status != "APPROVED" || approved.Data.DecidedAt == "" {
		t.Fatalf("status %s, decidedAt %q; want APPROVED with a decision time", approved.Data.Status, approved.Data.DecidedAt)
	}
}

// F08-TEST-05 (F08-AC-04, BR-16, D-11): once a claim is PENDING_FINANCE, its owner can no longer withdraw, submit,
// edit, or delete it, and its reviewer can no longer approve, reject, or return it. Each refusal is 409
// INVALID_TRANSITION with the rule of the action, and the claim and its history stay as they are.
func TestF08TEST05PendingFinanceAllowsNoOwnerOrReviewerAction(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id, version := pendingFinanceClaim(t, router)
	before := readState(t, pool, id)

	anan := loginAs(t, router, "req.anan")
	malee := loginAs(t, router, "rev.malee")
	path := "/api/v1/claims/" + id

	for _, test := range []struct {
		name          string
		token, method string
		path, body    string
		rule, message string
	}{
		{"withdraw", anan, http.MethodPost, path + "/withdraw", "", "BR-16", "Only a submitted or returned claim can be withdrawn."},
		{"submit", anan, http.MethodPost, path + "/submit", "", "BR-03", "Only a draft or a returned claim can be submitted."},
		{"edit", anan, http.MethodPut, path, threeItemsBody(), "BR-10", "Only a draft or a returned claim can be edited."},
		{"delete", anan, http.MethodDelete, path, "", "BR-10", "Only a draft can be deleted."},
		{"approve", malee, http.MethodPost, path + "/approve", "", "BR-03", "Only a submitted claim can be approved or rejected."},
		{"reject", malee, http.MethodPost, path + "/reject", `{"comment": "Receipt is missing."}`, "BR-03", "Only a submitted claim can be approved or rejected."},
		{"return", malee, http.MethodPost, path + "/return", `{"comment": "Receipt is missing."}`, "BR-03", "Only a submitted claim can be returned."},
	} {
		t.Run(test.name, func(t *testing.T) {
			expectTransition(t, draftRequest(t, router, test.token, test.method, test.path, ifMatch(version), test.body), test.rule, test.message)
		})
	}

	if after := readState(t, pool, id); after != before {
		t.Fatalf("claim = %+v after the refused actions, want %+v", after, before)
	}

	for _, token := range []string{anan, malee} {
		if actions := openClaimActions(t, router, token, id); len(actions) != 0 {
			t.Fatalf("allowedActions = %v, want none for the owner and the reviewer", actions)
		}
	}
}

// openClaimActions opens the claim as token and returns its allowedActions.
func openClaimActions(t *testing.T, router http.Handler, token, id string) []string {
	t.Helper()

	var body struct {
		Data struct {
			AllowedActions []string `json:"allowedActions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(getWithSession(t, router, "/api/v1/claims/"+id, token).Body.Bytes(), &body); err != nil {
		t.Fatalf("decode claim: %v", err)
	}

	return body.Data.AllowedActions
}
