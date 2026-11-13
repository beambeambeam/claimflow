package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// F01-TEST-09: a session is valid 7 h 59 min after login and gets 401 at 8 h. The next login deletes the
// expired row.
func TestF01TEST09SessionExpiresAfterEightHours(t *testing.T) {
	pool := pgtest.New(t)
	userID := insertCredentialUser(t, pool)

	t0 := time.Now().Truncate(time.Second)
	now := t0
	router := newRouterAt(discardLogger(), testEnv(0), pool, func() time.Time { return now })

	login := postLogin(t, router, "req.anan", "correct-password")
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200; body = %s", login.Code, login.Body)
	}
	oldToken := sessionToken(t, login, "sid", sessionAttributesPlain)

	now = t0.Add(7*time.Hour + 59*time.Minute)
	if got := getMeWithCookie(t, router, oldToken); got.Code != http.StatusOK {
		t.Fatalf("status at 7h59m = %d, want 200; body = %s", got.Code, got.Body)
	}

	now = t0.Add(8 * time.Hour)
	decodeProblem(t, getMeWithCookie(t, router, oldToken), http.StatusUnauthorized, "UNAUTHENTICATED")

	var before int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM session WHERE user_id = $1`, userID).Scan(&before); err != nil {
		t.Fatalf("count sessions: %v", err)
	}

	if before != 1 {
		t.Fatalf("sessions before the second login = %d, want 1 (the expired row)", before)
	}

	again := postLogin(t, router, "req.anan", "correct-password")
	if again.Code != http.StatusOK {
		t.Fatalf("second login status = %d, want 200; body = %s", again.Code, again.Body)
	}
	newToken := sessionToken(t, again, "sid", sessionAttributesPlain)

	var after int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM session WHERE user_id = $1`, userID).Scan(&after); err != nil {
		t.Fatalf("count sessions: %v", err)
	}

	if after != 1 {
		t.Fatalf("sessions after the second login = %d, want 1 (the new row only)", after)
	}

	if got := getMeWithCookie(t, router, newToken); got.Code != http.StatusOK {
		t.Fatalf("new session status = %d, want 200; body = %s", got.Code, got.Body)
	}
}
