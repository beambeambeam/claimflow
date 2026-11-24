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
