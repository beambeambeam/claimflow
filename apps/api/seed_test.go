package main

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// seedPassword stands in for SEED_USER_PASSWORD. The test passes it to seed directly.
const seedPassword = "seed-test-password"

// seedUsernames are the seven SDS §2.5 accounts, sorted. They are written out here, not read from
// the seed, so a wrong seed fails this test.
var seedUsernames = []string{
	"fin.wanida", "req.anan", "req.niran", "req.ploy", "rev.malee", "rev.prasert", "rev.somchai",
}

// F01-TEST-10: the seed stores only bcrypt hashes of SEED_USER_PASSWORD, and the password never reaches
// the seed output, the logs, or a response body.
func TestF01TEST10SeedStoresBcryptHashesOnly(t *testing.T) {
	pool := pgtest.New(t)

	var seedOutput bytes.Buffer
	if err := seed(t.Context(), pool.Config().ConnString(), seedPassword, 10, &seedOutput); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var users int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM "user"`).Scan(&users); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if users != len(seedUsernames) {
		t.Fatalf("users = %d, want %d", users, len(seedUsernames))
	}

	rows, err := pool.Query(t.Context(),
		`SELECT u.username, a.password FROM "user" u JOIN account a ON a.user_id = u.id WHERE a.provider_id = 'credential'`)
	if err != nil {
		t.Fatalf("read accounts: %v", err)
	}
	defer rows.Close()

	var usernames []string
	for rows.Next() {
		var username, hash string
		if err := rows.Scan(&username, &hash); err != nil {
			t.Fatalf("scan account: %v", err)
		}
		usernames = append(usernames, username)

		if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
			t.Fatalf("%s: password does not start with $2a$ or $2b$", username)
		}

		cost, err := bcrypt.Cost([]byte(hash))
		if err != nil {
			t.Fatalf("%s: read bcrypt cost: %v", username, err)
		}
		if cost < 10 {
			t.Fatalf("%s: bcrypt cost = %d, want at least 10", username, cost)
		}

		if hash == seedPassword {
			t.Fatalf("%s: password is the plain SEED_USER_PASSWORD", username)
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(seedPassword)); err != nil {
			t.Fatalf("%s: hash does not match SEED_USER_PASSWORD: %v", username, err)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read accounts: %v", err)
	}

	slices.Sort(usernames)
	if !slices.Equal(usernames, seedUsernames) {
		t.Fatalf("credential accounts = %v, want %v", usernames, seedUsernames)
	}

	var logs bytes.Buffer
	router := newRouter(slog.New(slog.NewJSONHandler(&logs, nil)), testEnv(0), pool)

	wrong := postLogin(t, router, "req.anan", "wrong-password")
	decodeProblem(t, wrong, http.StatusUnauthorized, "UNAUTHENTICATED")

	right := postLogin(t, router, "req.anan", seedPassword)
	if right.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", right.Code, right.Body)
	}

	leaks := []struct{ where, text string }{
		{"seed output", seedOutput.String()},
		{"logs", logs.String()},
		{"wrong login body", wrong.Body.String()},
		{"right login body", right.Body.String()},
	}
	for _, leak := range leaks {
		if strings.Contains(leak.text, seedPassword) {
			t.Fatalf("%s contains SEED_USER_PASSWORD", leak.where)
		}
	}
}

// TestSeedIsIdempotent is outside the F01-TEST list. SDS §2.5 requires a second run to leave the same seven users.
func TestSeedIsIdempotent(t *testing.T) {
	pool := pgtest.New(t)
	databaseURL := pool.Config().ConnString()

	for range 2 {
		if err := seed(t.Context(), databaseURL, seedPassword, 10, io.Discard); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	var users, accounts int
	if err := pool.QueryRow(t.Context(),
		`SELECT (SELECT count(*) FROM "user"), (SELECT count(*) FROM account)`).Scan(&users, &accounts); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if users != len(seedUsernames) || accounts != len(seedUsernames) {
		t.Fatalf("users = %d, accounts = %d, want %d of each", users, accounts, len(seedUsernames))
	}
}

// TestSeedCommandRefusesEmptyPassword: `api seed` fails before it writes anything when SEED_USER_PASSWORD is unset.
func TestSeedCommandRefusesEmptyPassword(t *testing.T) {
	pool := pgtest.New(t)

	var out bytes.Buffer
	err := seedCommand(t.Context(), pool.Config().ConnString(), "", 10, &out)
	if err == nil {
		t.Fatal("seedCommand with empty password: err = nil, want error")
	}
	if !strings.Contains(err.Error(), "SEED_USER_PASSWORD") {
		t.Fatalf("error = %q, want it to name SEED_USER_PASSWORD", err)
	}
	if out.Len() != 0 {
		t.Fatalf("output = %q, want none", out.String())
	}

	var users int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM "user"`).Scan(&users); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if users != 0 {
		t.Fatalf("users = %d, want 0", users)
	}
}

// TestSeedCommandSeedsWithPassword: with SEED_USER_PASSWORD set, `api seed` runs the seed and reports the count.
func TestSeedCommandSeedsWithPassword(t *testing.T) {
	pool := pgtest.New(t)

	var out bytes.Buffer
	if err := seedCommand(t.Context(), pool.Config().ConnString(), seedPassword, 10, &out); err != nil {
		t.Fatalf("seedCommand: %v", err)
	}
	if got, want := out.String(), "seeded 7 users\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
