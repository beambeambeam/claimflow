package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// sessionAttributesPlain are the cookie attributes when COOKIE_SECURE is false (F-01 §3.3).
const sessionAttributesPlain = "HttpOnly; SameSite=Lax; Path=/; Max-Age=28800"

// insertUserWithBan writes a user and a credential account with the password "correct-password". A banned user
// gets ban_expires as given; nil means the ban has no end.
func insertUserWithBan(t *testing.T, pool *pgxpool.Pool, username string, banned bool, banExpires *time.Time) {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), 10)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	userID := uuid.Must(uuid.NewV7())
	if _, err := pool.Exec(t.Context(),
		`INSERT INTO "user" (id, name, email, username, display_username, position, banned, ban_expires)
		 VALUES ($1, $2, $3, $2, $2, 'Staff', $4, $5)`,
		userID, username, username+"@claimflow.test", banned, banExpires); err != nil {
		t.Fatalf("insert user %s: %v", username, err)
	}

	if _, err := pool.Exec(t.Context(),
		`INSERT INTO account (id, user_id, provider_id, account_id, password) VALUES ($1, $2, 'credential', $3, $4)`,
		uuid.Must(uuid.NewV7()), userID, userID.String(), string(hash)); err != nil {
		t.Fatalf("insert account %s: %v", username, err)
	}
}

// F01-TEST-05: a ban with no end or an end in the future blocks login with the generic 401. A ban that ended
// counts as no ban.
func TestF01TEST05BannedUsersCannotLogIn(t *testing.T) {
	pool := pgtest.New(t)
	tomorrow := time.Now().Add(24 * time.Hour)
	yesterday := time.Now().Add(-24 * time.Hour)
	insertUserWithBan(t, pool, "ban.forever", true, nil)
	insertUserWithBan(t, pool, "ban.tomorrow", true, &tomorrow)
	insertUserWithBan(t, pool, "ban.yesterday", true, &yesterday)
	router := newRouter(discardLogger(), testEnv(0), pool)

	forever := decodeProblem(t, postLogin(t, router, "ban.forever", "correct-password"), http.StatusUnauthorized, "UNAUTHENTICATED")
	until := decodeProblem(t, postLogin(t, router, "ban.tomorrow", "correct-password"), http.StatusUnauthorized, "UNAUTHENTICATED")

	if forever.Title != until.Title {
		t.Fatalf("titles differ: %q and %q", forever.Title, until.Title)
	}

	wrong := decodeProblem(t, postLogin(t, router, "ban.forever", "wrong-password"), http.StatusUnauthorized, "UNAUTHENTICATED")
	if wrong.Title != forever.Title {
		t.Fatalf("banned title = %q, want the wrong-password title %q", forever.Title, wrong.Title)
	}

	ended := postLogin(t, router, "ban.yesterday", "correct-password")
	if ended.Code != http.StatusOK {
		t.Fatalf("ended ban status = %d, want 200; body = %s", ended.Code, ended.Body)
	}
}

// getMeWithCookie calls GET /auth/me with the plain-named session cookie (COOKIE_SECURE false) set to token.
func getMeWithCookie(t *testing.T, handler http.Handler, token string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/auth/me", http.NoBody)
	request.AddCookie(&http.Cookie{Name: "sid", Value: token, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return recorder
}

// F01-TEST-06: a ban set with a plain UPDATE deletes the user's sessions, so the old cookie gets 401.
func TestF01TEST06BanDeletesSessions(t *testing.T) {
	pool := pgtest.New(t)
	userID := insertCredentialUser(t, pool)
	router := newRouter(discardLogger(), testEnv(0), pool)

	login := postLogin(t, router, "req.anan", "correct-password")
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200; body = %s", login.Code, login.Body)
	}
	token := sessionToken(t, login, "sid", sessionAttributesPlain)

	if got := getMeWithCookie(t, router, token); got.Code != http.StatusOK {
		t.Fatalf("status before ban = %d, want 200; body = %s", got.Code, got.Body)
	}

	if _, err := pool.Exec(t.Context(), `UPDATE "user" SET banned = true WHERE id = $1`, userID); err != nil {
		t.Fatalf("ban user: %v", err)
	}

	decodeProblem(t, getMeWithCookie(t, router, token), http.StatusUnauthorized, "UNAUTHENTICATED")

	var sessions int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM session WHERE user_id = $1`, userID).Scan(&sessions); err != nil {
		t.Fatalf("count sessions: %v", err)
	}

	if sessions != 0 {
		t.Fatalf("sessions after ban = %d, want 0", sessions)
	}
}
