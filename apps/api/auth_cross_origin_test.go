package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// postLoginFrom sends a valid login request with one extra request header.
func postLoginFrom(t *testing.T, handler http.Handler, header, value string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(loginBody(t, "req.anan", "correct-password")))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(header, value)
	handler.ServeHTTP(recorder, request)

	return recorder
}

// F01-TEST-14: a state-changing request from another origin gets 403 and no cookie; the same origin succeeds.
// Logout is ticket 03, so the cases use login, the only state-changing operation in this slice.
func TestF01TEST14CrossOriginRequestsAreForbidden(t *testing.T) {
	pool := pgtest.New(t)
	insertCredentialUser(t, pool, "req.anan", "correct-password")
	router := newRouter(discardLogger(), testEnv(0), pool)

	crossSite := postLoginFrom(t, router, "Sec-Fetch-Site", "cross-site")
	decodeProblem(t, crossSite, http.StatusForbidden, "FORBIDDEN")

	foreignOrigin := postLoginFrom(t, router, "Origin", "https://evil.test")
	decodeProblem(t, foreignOrigin, http.StatusForbidden, "FORBIDDEN")

	for _, recorder := range []*httptest.ResponseRecorder{crossSite, foreignOrigin} {
		if recorder.Header().Get("Set-Cookie") != "" {
			t.Fatalf("Set-Cookie = %q, want none on a cross-origin request", recorder.Header().Get("Set-Cookie"))
		}
	}

	sameOrigin := postLoginFrom(t, router, "Sec-Fetch-Site", "same-origin")
	if sameOrigin.Code != http.StatusOK {
		t.Fatalf("same-origin status = %d, want 200; body = %s", sameOrigin.Code, sameOrigin.Body)
	}
}
