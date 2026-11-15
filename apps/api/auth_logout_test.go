package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// postLogout calls POST /auth/logout, with the session cookie when token is not empty.
func postLogout(t *testing.T, handler http.Handler, token string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/auth/logout", http.NoBody)
	if token != "" {
		request.AddCookie(&http.Cookie{Name: "sid", Value: token, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return recorder
}

// F01-TEST-08: logout answers 204, clears the cookie, and deletes the session row. The old cookie then gets 401.
func TestF01TEST08LogoutEndsSession(t *testing.T) {
	pool := pgtest.New(t)
	userID := insertCredentialUser(t, pool)
	router := newRouter(discardLogger(), testEnv(0), pool)

	login := postLogin(t, router, "req.anan", "correct-password")
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200; body = %s", login.Code, login.Body)
	}
	token := sessionToken(t, login, "sid", sessionAttributesPlain)

	logout := postLogout(t, router, token)
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want 204; body = %s", logout.Code, logout.Body)
	}

	if got, want := logout.Header().Get("Set-Cookie"), "sid=; HttpOnly; SameSite=Lax; Path=/; Max-Age=0"; got != want {
		t.Fatalf("Set-Cookie = %q, want %q", got, want)
	}

	decodeProblem(t, getMeWithCookie(t, router, token), http.StatusUnauthorized, "UNAUTHENTICATED")

	var sessions int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM session WHERE user_id = $1`, userID).Scan(&sessions); err != nil {
		t.Fatalf("count sessions: %v", err)
	}

	if sessions != 0 {
		t.Fatalf("sessions after logout = %d, want 0", sessions)
	}
}

// TestLogoutNeedsSession is outside the F01-TEST list. F-01 §3.2: logout needs a session, so a request with no
// cookie, or with a token of no session, gets 401.
func TestLogoutNeedsSession(t *testing.T) {
	pool := pgtest.New(t)
	router := newRouter(discardLogger(), testEnv(0), pool)

	decodeProblem(t, postLogout(t, router, ""), http.StatusUnauthorized, "UNAUTHENTICATED")
	decodeProblem(t, postLogout(t, router, "unknown-token"), http.StatusUnauthorized, "UNAUTHENTICATED")
}

// TestLogoutClearsHostCookieWhenSecure is outside the F01-TEST list. F-01 §3.3: with COOKIE_SECURE true, logout
// clears __Host-sid with the Secure attribute.
func TestLogoutClearsHostCookieWhenSecure(t *testing.T) {
	pool := pgtest.New(t)
	insertCredentialUser(t, pool)

	cfg := testEnv(0)
	cfg.CookieSecure = true
	router := newRouter(discardLogger(), cfg, pool)

	token := sessionToken(t, postLogin(t, router, "req.anan", "correct-password"), "__Host-sid", sessionAttributes)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/auth/logout", http.NoBody)
	request.AddCookie(&http.Cookie{Name: "__Host-sid", Value: token, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want 204; body = %s", recorder.Code, recorder.Body)
	}

	if got, want := recorder.Header().Get("Set-Cookie"), "__Host-sid=; HttpOnly; Secure; SameSite=Lax; Path=/; Max-Age=0"; got != want {
		t.Fatalf("Set-Cookie = %q, want %q", got, want)
	}
}
