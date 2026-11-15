package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// threeItemsBody is a valid claim of three items.
func threeItemsBody() string {
	return claimBodyWith(strings.Join([]string{
		itemJSON("TRAVEL", "2026-11-03", "250.00"),
		itemJSON("ACCOMMODATION", "2026-11-04", "1000.00"),
		itemJSON("OTHER", "2026-11-05", "50.00"),
	}, ","))
}

// submittedClaim has req.anan submit threeItemsBody and returns the claim id.
func submittedClaim(t *testing.T, router http.Handler) string {
	t.Helper()

	recorder := postClaim(t, router, loginAs(t, router, "req.anan"), threeItemsBody())
	if recorder.Code != http.StatusCreated {
		t.Fatalf("submit status = %d, want 201; body = %s", recorder.Code, recorder.Body)
	}

	var body claimResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	return body.Data.ID
}

// F03-TEST-05 (F03-AC-05): the requester opens their claim and gets the stored claim, the ETag, and no actions.
func TestF03TEST05RequesterOpensOwnClaim(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedClaim(t, router)

	recorder := getWithSession(t, router, "/api/v1/claims/"+id, loginAs(t, router, "req.anan"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body)
	}

	var body claimResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	claim := body.Data

	if got := recorder.Header().Get("ETag"); got != `"0"` {
		t.Fatalf("ETag = %q, want \"0\"", got)
	}

	if claim.ID != id || claim.Status != "SUBMITTED" || claim.Total != "1300.00" || claim.Version != 0 ||
		claim.AssignedReviewer.Name != "Malee S." || claim.Requester.Name != "Anan K." {
		t.Fatalf("claim = %+v, want SUBMITTED, total 1300.00, version 0, requester Anan K., reviewer Malee S.", claim)
	}

	if len(claim.Items) != 3 {
		t.Fatalf("items = %+v, want 3", claim.Items)
	}
	for i, item := range claim.Items {
		if item.LineNo != i+1 {
			t.Fatalf("items[%d].lineNo = %d, want %d", i, item.LineNo, i+1)
		}
	}

	if len(claim.History) != 1 || claim.History[0].Action != "SUBMIT" || claim.History[0].ToStatus != "SUBMITTED" {
		t.Fatalf("history = %+v, want one SUBMIT entry", claim.History)
	}

	if claim.AllowedActions == nil || len(claim.AllowedActions) != 0 {
		t.Fatalf("allowedActions = %v, want an empty list", claim.AllowedActions)
	}
}

// F03-TEST-06 (F03-AC-06, NFR-02): another requester and an unassigned reviewer get 403 and no claim data.
func TestF03TEST06UnrelatedMembersAreForbiddenWithNoClaimData(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedClaim(t, router)

	for _, username := range []string{"req.ploy", "rev.somchai"} {
		recorder := getWithSession(t, router, "/api/v1/claims/"+id, loginAs(t, router, username))
		decodeProblem(t, recorder, http.StatusForbidden, "FORBIDDEN")

		var fields map[string]json.RawMessage
		if err := json.Unmarshal(recorder.Body.Bytes(), &fields); err != nil {
			t.Fatalf("%s: decode body: %v", username, err)
		}

		// A problem has its own title ("Forbidden."), so the claim's title is checked by value below.
		for _, name := range []string{"data", "claimNo", "items"} {
			if _, found := fields[name]; found {
				t.Fatalf("%s: body has %q: %s", username, name, recorder.Body)
			}
		}

		if strings.Contains(recorder.Body.String(), "Bangkok workshop travel") {
			t.Fatalf("%s: body holds the claim title: %s", username, recorder.Body)
		}
	}
}

// F03-TEST-07 (F03-AC-07): the assigned reviewer opens a SUBMITTED claim.
func TestF03TEST07AssignedReviewerOpensSubmittedClaim(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	id := submittedClaim(t, router)

	recorder := getWithSession(t, router, "/api/v1/claims/"+id, loginAs(t, router, "rev.malee"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body)
	}
}

// F03-TEST-08 (F03-AC-08): an unknown UUID is 404, a malformed id is 400.
func TestF03TEST08UnknownAndMalformedIDs(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "req.anan")

	decodeProblem(t, getWithSession(t, router, "/api/v1/claims/"+uuid.Must(uuid.NewV7()).String(), token), http.StatusNotFound, "NOT_FOUND")
	decodeProblem(t, getWithSession(t, router, "/api/v1/claims/abc", token), http.StatusBadRequest, "BAD_REQUEST")
}
