package main

import (
	"encoding/json"
	"io"
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
	userID := insertCredentialUser(t, pool)
	router := newRouter(discardLogger(), testEnv(0), pool)

	decodeProblem(t, get(t, router, "/api/v1/auth/me"), http.StatusUnauthorized, "UNAUTHENTICATED")

	login := postLogin(t, router, "req.anan", "correct-password")
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200; body = %s", login.Code, login.Body)
	}
	token := sessionToken(t, login, "sid", "HttpOnly; SameSite=Lax; Path=/; Max-Age=28800")

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/auth/me", http.NoBody)
	request.AddCookie(&http.Cookie{
		Name:     "sid",
		Value:    token,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
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
	request.AddCookie(&http.Cookie{
		Name:     "sid",
		Value:    "not-a-session",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	decodeProblem(t, recorder, http.StatusUnauthorized, "UNAUTHENTICATED")
}

// getWithSession sends a GET to path with the `sid` session cookie that COOKIE_SECURE=false uses.
func getWithSession(t *testing.T, handler http.Handler, path, token string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody)
	// gosec (G124) requires these attributes on every http.Cookie; a request cookie ignores them.
	request.AddCookie(&http.Cookie{Name: "sid", Value: token, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return recorder
}

// jsonKeys returns every object key in a JSON document, at any depth.
func jsonKeys(t *testing.T, document []byte) []string {
	t.Helper()

	var value any
	if err := json.Unmarshal(document, &value); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	var keys []string
	var walk func(any)
	walk = func(value any) {
		switch value := value.(type) {
		case map[string]any:
			for key, child := range value {
				keys = append(keys, key)
				walk(child)
			}
		case []any:
			for _, child := range value {
				walk(child)
			}
		}
	}
	walk(value)

	return keys
}

// F01-TEST-07: /auth/me returns each seeded role's exact permission map from F-01 §1.6, and no secret key.
func TestF01TEST07GetMeReturnsThePermissionsOfEachRole(t *testing.T) {
	pool := pgtest.New(t)
	if err := seed(t.Context(), pool.Config().ConnString(), seedPassword, 10, io.Discard); err != nil {
		t.Fatalf("seed: %v", err)
	}
	router := newRouter(discardLogger(), testEnv(0), pool)

	tests := []struct {
		username    string
		role        string
		permissions map[string][]string
	}{
		{"req.anan", "requester", map[string][]string{"claim": {"create", "list"}}},
		{"rev.malee", "reviewer", map[string][]string{"claim": {"create", "list"}, "review": {"list"}, "report": {"read"}}},
		{"fin.wanida", "finance", map[string][]string{"review": {"list"}, "finance": {"list"}, "report": {"read"}}},
	}

	for _, test := range tests {
		t.Run(test.username, func(t *testing.T) {
			login := postLogin(t, router, test.username, seedPassword)
			if login.Code != http.StatusOK {
				t.Fatalf("login status = %d, want 200; body = %s", login.Code, login.Body)
			}
			token := sessionToken(t, login, "sid", "HttpOnly; SameSite=Lax; Path=/; Max-Age=28800")

			recorder := getWithSession(t, router, "/api/v1/auth/me", token)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body)
			}

			for _, key := range jsonKeys(t, recorder.Body.Bytes()) {
				lower := strings.ToLower(key)
				for _, secret := range []string{"password", "hash", "token"} {
					if strings.Contains(lower, secret) {
						t.Fatalf("body has key %q; body = %s", key, recorder.Body)
					}
				}
			}

			for _, secret := range []string{token, seedPassword} {
				if strings.Contains(recorder.Body.String(), secret) {
					t.Fatalf("body leaks a secret value; body = %s", recorder.Body)
				}
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

			if body.Data.ID == "" || body.Data.Name == "" || body.Data.Username != test.username || body.Data.Role != test.role {
				t.Fatalf("data = %+v, want %s as %s with an id and a name", body.Data, test.username, test.role)
			}

			if !maps.EqualFunc(body.Data.Permissions, test.permissions, slices.Equal) {
				t.Fatalf("permissions = %v, want %v", body.Data.Permissions, test.permissions)
			}
		})
	}
}
