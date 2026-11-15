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

// F01-TEST-14 (login): a state-changing request from another origin gets 403 and no cookie; the same origin succeeds.
func TestF01TEST14CrossOriginRequestsAreForbidden(t *testing.T) {
	pool := pgtest.New(t)
	insertCredentialUser(t, pool)
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

// postLogoutFrom sends a logout request with the session cookie and one extra request header.
func postLogoutFrom(t *testing.T, handler http.Handler, token, header, value string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/auth/logout", http.NoBody)
	request.AddCookie(&http.Cookie{Name: "sid", Value: token, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
	request.Header.Set(header, value)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return recorder
}

// F01-TEST-14 (logout): a cross-origin logout gets 403 and keeps the session; the same origin gets 204.
func TestF01TEST14CrossOriginLogoutIsForbidden(t *testing.T) {
	pool := pgtest.New(t)
	insertCredentialUser(t, pool)
	router := newRouter(discardLogger(), testEnv(0), pool)

	login := postLogin(t, router, "req.anan", "correct-password")
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200; body = %s", login.Code, login.Body)
	}
	token := sessionToken(t, login, "sid", sessionAttributesPlain)

	crossSite := postLogoutFrom(t, router, token, "Sec-Fetch-Site", "cross-site")
	decodeProblem(t, crossSite, http.StatusForbidden, "FORBIDDEN")

	foreignOrigin := postLogoutFrom(t, router, token, "Origin", "https://evil.test")
	decodeProblem(t, foreignOrigin, http.StatusForbidden, "FORBIDDEN")

	for _, recorder := range []*httptest.ResponseRecorder{crossSite, foreignOrigin} {
		if recorder.Header().Get("Set-Cookie") != "" {
			t.Fatalf("Set-Cookie = %q, want none on a cross-origin request", recorder.Header().Get("Set-Cookie"))
		}
	}

	if me := getMeWithCookie(t, router, token); me.Code != http.StatusOK {
		t.Fatalf("session after cross-origin logout: status = %d, want 200; body = %s", me.Code, me.Body)
	}

	sameOrigin := postLogoutFrom(t, router, token, "Sec-Fetch-Site", "same-origin")
	if sameOrigin.Code != http.StatusNoContent {
		t.Fatalf("same-origin status = %d, want 204; body = %s", sameOrigin.Code, sameOrigin.Body)
	}
}
