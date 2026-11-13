package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// sessionAttributes are the cookie attributes after the value when COOKIE_SECURE is true (F-01 §3.3).
const sessionAttributes = "HttpOnly; Secure; SameSite=Lax; Path=/; Max-Age=28800"

// insertCredentialUser writes a user and its credential account directly, so login tests do not
// depend on the seed. The account_id is the user ID, as the seed writes it.
func insertCredentialUser(t *testing.T, pool *pgxpool.Pool, username, password string) uuid.UUID {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	userID := uuid.Must(uuid.NewV7())
	if _, err := pool.Exec(t.Context(),
		`INSERT INTO "user" (id, name, email, username, display_username, position) VALUES ($1, $2, $3, $2, $2, 'Staff')`,
		userID, username, username+"@claimflow.test"); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if _, err := pool.Exec(t.Context(),
		`INSERT INTO account (id, user_id, provider_id, account_id, password) VALUES ($1, $2, 'credential', $3, $4)`,
		uuid.Must(uuid.NewV7()), userID, userID.String(), string(hash)); err != nil {
		t.Fatalf("insert account: %v", err)
	}

	return userID
}

func postLogin(t *testing.T, handler http.Handler, username, password string) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(map[string]string{"username": username, "password": password})
	if err != nil {
		t.Fatalf("encode login body: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recorder, request)

	return recorder
}

// sessionToken checks that the Set-Cookie header is name=<token> with the given attributes, and returns the token.
func sessionToken(t *testing.T, recorder *httptest.ResponseRecorder, name, attributes string) string {
	t.Helper()

	header := recorder.Header().Get("Set-Cookie")

	pair, ok := strings.CutPrefix(header, name+"=")
	if !ok {
		t.Fatalf("Set-Cookie = %q, want it to start with %s=", header, name)
	}

	token, got, _ := strings.Cut(pair, "; ")
	if token == "" || got != attributes {
		t.Fatalf("Set-Cookie = %q, want %s=<token>; %s", header, name, attributes)
	}

	return token
}

// F01-TEST-01: a valid login returns the current user and sets __Host-sid when COOKIE_SECURE is true.
func TestF01TEST01LoginSetsHostSessionCookie(t *testing.T) {
	pool := pgtest.New(t)
	insertCredentialUser(t, pool, "req.anan", "correct-password")

	cfg := testEnv(0)
	cfg.CookieSecure = true

	recorder := postLogin(t, newRouter(discardLogger(), cfg, pool), "req.anan", "correct-password")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body)
	}

	var body struct {
		Data struct {
			Username string `json:"username"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if body.Data.Username != "req.anan" {
		t.Fatalf("data.username = %q, want req.anan", body.Data.Username)
	}

	sessionToken(t, recorder, "__Host-sid", sessionAttributes)
}
