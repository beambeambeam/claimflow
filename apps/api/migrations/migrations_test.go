package migrations_test

import (
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

func TestMain(m *testing.M) {
	os.Exit(pgtest.Run(m))
}

func insertUser(t *testing.T, pool *pgxpool.Pool, username string) uuid.UUID {
	t.Helper()

	id := uuid.Must(uuid.NewV7())
	if _, err := pool.Exec(t.Context(), `
		INSERT INTO "user" (id, name, email, username, display_username, position)
		VALUES ($1, $2, $3, $2, $2, 'Staff')`, id, username, username+"@example.com"); err != nil {
		t.Fatalf("insert user %s: %v", username, err)
	}

	if _, err := pool.Exec(t.Context(), `
		INSERT INTO session (id, user_id, token, expires_at)
		VALUES ($1, $2, $3, now() + interval '8 hours')`, uuid.Must(uuid.NewV7()), id, []byte(username)); err != nil {
		t.Fatalf("insert session for %s: %v", username, err)
	}

	return id
}

func sessionCount(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) int {
	t.Helper()

	var count int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM session WHERE user_id = $1`, userID).Scan(&count); err != nil {
		t.Fatalf("count sessions: %v", err)
	}

	return count
}

// F01-AC-05: bans are set with a plain UPDATE (seed data or a migration), so the
// database itself deletes the user's sessions when a ban starts.
func TestBanDeletesSessions(t *testing.T) {
	tests := []struct {
		name         string
		ban          string
		wantSessions int
	}{
		{"ban with no expiry", `UPDATE "user" SET banned = true WHERE id = $1`, 0},
		{"ban until tomorrow", `UPDATE "user" SET banned = true, ban_expires = now() + interval '1 day' WHERE id = $1`, 0},
		{"ban that already expired", `UPDATE "user" SET banned = true, ban_expires = now() - interval '1 day' WHERE id = $1`, 1},
		{"other column change", `UPDATE "user" SET position = 'Lead' WHERE id = $1`, 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pool := pgtest.New(t)
			userID := insertUser(t, pool, "member")
			otherID := insertUser(t, pool, "other")

			if _, err := pool.Exec(t.Context(), test.ban, userID); err != nil {
				t.Fatalf("update user: %v", err)
			}

			if got := sessionCount(t, pool, userID); got != test.wantSessions {
				t.Fatalf("sessions after update = %d, want %d", got, test.wantSessions)
			}

			if got := sessionCount(t, pool, otherID); got != 1 {
				t.Fatalf("other user's sessions = %d, want 1", got)
			}
		})
	}
}
