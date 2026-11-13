package main

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// TestGetMeRequiresSessionAndReturnsCurrentUser is outside the F01-TEST list. F-01 §3.2 and §1.6: /auth/me needs a
// session, returns the session's user with the role's permissions, and never echoes the cookie or a secret.
func TestGetMeRequiresSessionAndReturnsCurrentUser(t *testing.T) {
	pool := pgtest.New(t)
	userID := insertCredentialUser(t, pool, "req.anan", "correct-password")
	router := newRouter(discardLogger(), testEnv(0), pool)

	decodeProblem(t, get(t, router, "/api/v1/auth/me"), http.StatusUnauthorized, "UNAUTHENTICATED")

	login := postLogin(t, router, "req.anan", "correct-password")
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200; body = %s", login.Code, login.Body)
	}
	token := sessionToken(t, login, "sid", "HttpOnly; SameSite=Lax; Path=/; Max-Age=28800")

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/auth/me", http.NoBody)
	request.AddCookie(&http.Cookie{Name: "sid", Value: token})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body)
	}

	if strings.Contains(recorder.Body.String(), token) {
		t.Fatal("body echoes the session cookie")
	}

	var keys struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &keys); err != nil {
		t.Fatalf("decode keys: %v", err)
	}

	want := []string{"id", "name", "permissions", "role", "username"}
	if got := slices.Sorted(maps.Keys(keys.Data)); !slices.Equal(got, want) {
		t.Fatalf("data keys = %v, want %v", got, want)
	}

	var body struct {
		Data struct {
			ID          string              `json:"id"`
			Username    string              `json:"username"`
			Name        string              `json:"name"`
			Role        string              `json:"role"`
			Permissions map[string][]string `json:"permissions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if body.Data.ID != userID.String() || body.Data.Username != "req.anan" || body.Data.Name != "req.anan" || body.Data.Role != "requester" {
		t.Fatalf("data = %+v, want req.anan as requester with id %s", body.Data, userID)
	}

	wantPermissions := map[string][]string{"claim": {"create", "list"}}
	if !maps.EqualFunc(body.Data.Permissions, wantPermissions, slices.Equal) {
		t.Fatalf("permissions = %v, want %v", body.Data.Permissions, wantPermissions)
	}
}

// TestGetMeRejectsUnknownSession checks that a cookie matching no session gets the same 401 as no cookie.
func TestGetMeRejectsUnknownSession(t *testing.T) {
	router := newRouter(discardLogger(), testEnv(0), pgtest.New(t))

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/auth/me", http.NoBody)
	request.AddCookie(&http.Cookie{Name: "sid", Value: "not-a-session"})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	decodeProblem(t, recorder, http.StatusUnauthorized, "UNAUTHENTICATED")
}
