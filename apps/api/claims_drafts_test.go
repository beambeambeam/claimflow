package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

const draftsPath = "/api/v1/claims?draft=true"

func draftRequest(t *testing.T, router http.Handler, token, method, path, version, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if version != "" {
		request.Header.Set("If-Match", version)
	}
	request.AddCookie(&http.Cookie{Name: "sid", Value: token, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return recorder
}

func expectDraftResponse(t *testing.T, rec *httptest.ResponseRecorder, status int) api.Claim {
	t.Helper()

	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, status, rec.Body)
	}

	var response api.ClaimEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode claim: %v", err)
	}

	return response.Data
}

func TestF05TEST01SaveDraftWithoutSubmission(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "req.anan")
	rec := draftRequest(t, router, token, http.MethodPost, draftsPath, "", twoItemsBody(`"total": "1.00",`))
	got := expectDraftResponse(t, rec, http.StatusCreated)

	if got.Status != api.ClaimStatusDraft || got.Total != 125000 || got.Version != 0 || got.ClaimNo == "" ||
		got.AssignedReviewer != nil || got.FirstSubmittedAt != nil || got.SubmittedAt != nil ||
		got.History == nil || len(got.History) != 0 || rec.Header().Get("ETag") != `"0"` ||
		rec.Header().Get("Location") != "/api/v1/claims/"+got.Id {
		t.Fatalf("unexpected draft: %s", rec.Body)
	}
	if n := rowCount(t, pool, "SELECT count(*) FROM claim_history"); n != 0 {
		t.Fatalf("history rows = %d, want 0", n)
	}
}

func TestF05TEST03FinanceOfficerCannotSaveDraft(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "fin.wanida")
	rec := draftRequest(t, router, token, http.MethodPost, draftsPath, "", twoItemsBody(""))
	if rec.Code != http.StatusForbidden || rowCount(t, pool, "SELECT count(*) FROM claims") != 0 {
		t.Fatalf("finance draft: status %d, body %s", rec.Code, rec.Body)
	}
}

func TestF05TEST09EditRequiresOwnerAndEditableStatus(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	owner := loginAs(t, router, "req.anan")
	other := loginAs(t, router, "req.ploy")
	draft := expectDraftResponse(t, draftRequest(t, router, owner, http.MethodPost, draftsPath, "", twoItemsBody("")), http.StatusCreated)
	submitted := submittedBy(t, router, "req.anan")

	rec := draftRequest(t, router, other, http.MethodPut, "/api/v1/claims/"+draft.Id, `"0"`, twoItemsBody(""))
	decodeProblem(t, rec, http.StatusForbidden, "FORBIDDEN")
	rec = draftRequest(t, router, owner, http.MethodPut, "/api/v1/claims/"+submitted, `"0"`, twoItemsBody(""))
	expectTransition(t, rec, "BR-10", "Only a draft or a returned claim can be edited.")
}

func TestF05TEST19DeleteRequiresOwnerAndDraftStatus(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	owner := loginAs(t, router, "req.anan")
	other := loginAs(t, router, "req.ploy")
	draft := expectDraftResponse(t, draftRequest(t, router, owner, http.MethodPost, draftsPath, "", twoItemsBody("")), http.StatusCreated)
	submitted := submittedBy(t, router, "req.anan")

	rec := draftRequest(t, router, other, http.MethodDelete, "/api/v1/claims/"+draft.Id, `"0"`, "")
	decodeProblem(t, rec, http.StatusForbidden, "FORBIDDEN")
	rec = draftRequest(t, router, owner, http.MethodDelete, "/api/v1/claims/"+submitted, `"0"`, "")
	expectTransition(t, rec, "BR-10", "Only a draft can be deleted.")

	rec = draftRequest(t, router, owner, http.MethodDelete, "/api/v1/claims/"+draft.Id, `"0"`, "")
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body)
	}
	decodeProblem(t, getWithSession(t, router, "/api/v1/claims/"+draft.Id, owner), http.StatusNotFound, "NOT_FOUND")
	if n := rowCount(t, pool, "SELECT count(*) FROM line_items"); n != 3 {
		t.Fatalf("line items = %d, want only the submitted claim's 3 items", n)
	}
}

func expectTransition(t *testing.T, rec *httptest.ResponseRecorder, rule, message string) {
	t.Helper()

	decodeProblem(t, rec, http.StatusConflict, "INVALID_TRANSITION")
	if got := problemErrors(t, rec); len(got) != 1 || got[0] != (fieldError{Field: "status", Rule: rule, Message: message}) {
		t.Fatalf("transition errors = %+v", got)
	}
}
