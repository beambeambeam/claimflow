package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// sessionAttributes are the cookie attributes after the value when COOKIE_SECURE is true (F-01 §3.3).
const sessionAttributes = "HttpOnly; Secure; SameSite=Lax; Path=/; Max-Age=28800"

// insertCredentialUser writes req.anan and its credential account directly, so login tests do not
// depend on the seed. The account_id is the user ID, as the seed writes it.
func insertCredentialUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), 10)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	userID := uuid.Must(uuid.NewV7())
	if _, err := pool.Exec(t.Context(),
		`INSERT INTO "user" (id, name, email, username, display_username, position) VALUES ($1, $2, $3, $2, $2, 'Staff')`,
		userID, "req.anan", "req.anan@claimflow.test"); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if _, err := pool.Exec(t.Context(),
		`INSERT INTO account (id, user_id, provider_id, account_id, password) VALUES ($1, $2, 'credential', $3, $4)`,
		uuid.Must(uuid.NewV7()), userID, userID.String(), string(hash)); err != nil {
		t.Fatalf("insert account: %v", err)
	}

	return userID
}

// loginBody encodes a login request body.
func loginBody(t *testing.T, username, password string) string {
	t.Helper()

	body, err := json.Marshal(map[string]string{"username": username, "password": password})
	if err != nil {
		t.Fatalf("encode login body: %v", err)
	}

	return string(body)
}

// postLoginBody sends a raw body to the login operation, so a test can send one the client would not encode.
func postLoginBody(t *testing.T, handler http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recorder, request)

	return recorder
}

func postLogin(t *testing.T, handler http.Handler, username, password string) *httptest.ResponseRecorder {
	t.Helper()

	return postLoginBody(t, handler, loginBody(t, username, password))
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
	insertCredentialUser(t, pool)

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

// F01-TEST-02: with COOKIE_SECURE false, the session cookie is named sid and has no Secure attribute.
func TestF01TEST02LoginSetsPlainSessionCookieWhenNotSecure(t *testing.T) {
	pool := pgtest.New(t)
	insertCredentialUser(t, pool)

	cfg := testEnv(0)
	cfg.CookieSecure = false

	recorder := postLogin(t, newRouter(discardLogger(), cfg, pool), "req.anan", "correct-password")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body)
	}

	sessionToken(t, recorder, "sid", "HttpOnly; SameSite=Lax; Path=/; Max-Age=28800")
}

// F01-TEST-03: the username is matched case-insensitively, and the response carries the user's ID.
func TestF01TEST03LoginMatchesUsernameIgnoringCase(t *testing.T) {
	pool := pgtest.New(t)
	userID := insertCredentialUser(t, pool)

	recorder := postLogin(t, newRouter(discardLogger(), testEnv(0), pool), "REQ.Anan", "correct-password")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body)
	}

	var body struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if body.Data.ID != userID.String() {
		t.Fatalf("data.id = %q, want %s", body.Data.ID, userID)
	}
}

// F01-TEST-04: a wrong password and an unknown username get the same 401 problem, and no cookie.
func TestF01TEST04WrongPasswordAndUnknownUserLookTheSame(t *testing.T) {
	pool := pgtest.New(t)
	insertCredentialUser(t, pool)
	router := newRouter(discardLogger(), testEnv(0), pool)

	wrongPassword := postLogin(t, router, "req.anan", "wrong-password")
	unknownUser := postLogin(t, router, "nobody", "correct-password")

	wrong := decodeProblem(t, wrongPassword, http.StatusUnauthorized, "UNAUTHENTICATED")
	unknown := decodeProblem(t, unknownUser, http.StatusUnauthorized, "UNAUTHENTICATED")

	for _, recorder := range []*httptest.ResponseRecorder{wrongPassword, unknownUser} {
		if recorder.Header().Get("Set-Cookie") != "" {
			t.Fatalf("Set-Cookie = %q, want none on failed login", recorder.Header().Get("Set-Cookie"))
		}
	}

	if wrong.Title != "Invalid username or password." {
		t.Fatalf("title = %q, want %q", wrong.Title, "Invalid username or password.")
	}

	wrong.RequestID, unknown.RequestID = "", ""
	if wrong != unknown {
		t.Fatalf("wrong password = %+v, unknown user = %+v, want equal except requestId", wrong, unknown)
	}
}

// F01-TEST-11: the session row stores the SHA-256 of the cookie value, never the cookie value itself.
func TestF01TEST11SessionStoresHashOfCookieValue(t *testing.T) {
	pool := pgtest.New(t)
	insertCredentialUser(t, pool)

	recorder := postLogin(t, newRouter(discardLogger(), testEnv(0), pool), "req.anan", "correct-password")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body)
	}

	cookie := sessionToken(t, recorder, "sid", "HttpOnly; SameSite=Lax; Path=/; Max-Age=28800")

	var stored []byte
	if err := pool.QueryRow(t.Context(), `SELECT token FROM session`).Scan(&stored); err != nil {
		t.Fatalf("read session token: %v", err)
	}

	if bytes.Equal(stored, []byte(cookie)) {
		t.Fatal("session.token = cookie value, want only its hash")
	}

	sum := sha256.Sum256([]byte(cookie))
	if !bytes.Equal(stored, sum[:]) {
		t.Fatalf("session.token = %x, want sha256(cookie) = %x", stored, sum)
	}
}

// TestLoginRejectsMalformedOrOutOfRangeRequests is outside the F01-TEST list. A body that does not decode,
// or breaks the LoginRequest bounds (username 1–50 characters, password 1–72 bytes), gets a 400 BAD_REQUEST problem.
// errors names each invalid field. A body that does not decode names no field, so it has no errors member.
func TestLoginRejectsMalformedOrOutOfRangeRequests(t *testing.T) {
	pool := pgtest.New(t)
	insertCredentialUser(t, pool)
	router := newRouter(discardLogger(), testEnv(0), pool)

	cases := []struct {
		name   string
		body   string
		fields []string
	}{
		{name: "malformed JSON", body: `{"username":`},
		{name: "empty username", body: loginBody(t, "", "correct-password"), fields: []string{"username"}},
		{name: "username over 50 characters", body: loginBody(t, strings.Repeat("a", 51), "correct-password"), fields: []string{"username"}},
		{name: "empty password", body: loginBody(t, "req.anan", ""), fields: []string{"password"}},
		{name: "password over 72 bytes", body: loginBody(t, "req.anan", strings.Repeat("p", 73)), fields: []string{"password"}},
		{name: "empty username and password", body: loginBody(t, "", ""), fields: []string{"username", "password"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := postLoginBody(t, router, tc.body)
			decodeProblem(t, recorder, http.StatusBadRequest, "BAD_REQUEST")

			var body struct {
				Errors *[]struct {
					Field string `json:"field"`
				} `json:"errors"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode problem: %v", err)
			}

			if tc.fields == nil {
				if body.Errors != nil {
					t.Fatalf("errors = %+v, want none when no field can be named", *body.Errors)
				}

				return
			}

			if body.Errors == nil {
				t.Fatalf("errors missing, want fields %v", tc.fields)
			}

			got := make([]string, 0, len(*body.Errors))
			for _, fieldError := range *body.Errors {
				got = append(got, fieldError.Field)
			}

			if !slices.Equal(got, tc.fields) {
				t.Fatalf("errors fields = %v, want %v", got, tc.fields)
			}
		})
	}
}
