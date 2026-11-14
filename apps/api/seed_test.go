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

// seededClaim is one sample claim of F-04 §4, written out here so a wrong seed fails the test.
type seededClaim struct {
	requester string
	reviewer  string
	status    string
	total     string
	history   []string
	reason    string // the comment of the last history entry, or empty for none
}

var seededClaims = []seededClaim{
	{"req.anan", "rev.malee", "SUBMITTED", "1250.00", []string{"SUBMIT"}, ""},
	{"req.anan", "rev.malee", "APPROVED", "1250.00", []string{"SUBMIT", "APPROVE"}, ""},
	{"req.ploy", "rev.malee", "REJECTED", "1250.00", []string{"SUBMIT", "REJECT"}, "Receipt is missing."},
	{"req.niran", "rev.somchai", "SUBMITTED", "1250.00", []string{"SUBMIT"}, ""},
	{"rev.malee", "rev.prasert", "SUBMITTED", "1250.00", []string{"SUBMIT"}, ""},
	{"rev.prasert", "fin.wanida", "SUBMITTED", "40000.00", []string{"SUBMIT"}, ""},
}

// F04-TEST-22 (F04-AC-19): on an empty database, the seed adds the sample claims of F-04 §4 with their history. A second
// run leaves the same rows.
func TestF04TEST22SeedAddsSampleClaimsOnce(t *testing.T) {
	pool := pgtest.New(t)
	databaseURL := pool.Config().ConnString()

	counts := func() (claims, items, history int) {
		t.Helper()

		if err := pool.QueryRow(t.Context(),
			`SELECT (SELECT count(*) FROM claims), (SELECT count(*) FROM line_items), (SELECT count(*) FROM claim_history)`).
			Scan(&claims, &items, &history); err != nil {
			t.Fatalf("count rows: %v", err)
		}

		return claims, items, history
	}

	if err := seed(t.Context(), databaseURL, seedPassword, 10, io.Discard); err != nil {
		t.Fatalf("seed: %v", err)
	}

	claims, items, history := counts()
	if claims != len(seededClaims) || items < claims || history != 8 {
		t.Fatalf("after the first run: %d claims, %d items, %d history entries; want %d claims, at least as many items, 8 history entries",
			claims, items, history, len(seededClaims))
	}

	rows, err := pool.Query(t.Context(), `
		SELECT r.username, v.username, c.status::text, (c.total_satang / 100)::text || '.' || lpad((c.total_satang % 100)::text, 2, '0'),
		       (SELECT sum(amount_satang) FROM line_items i WHERE i.claim_id = c.id) = c.total_satang,
		       c.version,
		       array(SELECT h.action::text FROM claim_history h WHERE h.claim_id = c.id ORDER BY h.id),
		       coalesce((SELECT h.comment FROM claim_history h WHERE h.claim_id = c.id ORDER BY h.id DESC LIMIT 1), ''),
		       c.submitted_at IS NOT NULL, c.decided_at IS NOT NULL
		FROM claims c
		JOIN "user" r ON r.id = c.requester_id
		JOIN "user" v ON v.id = c.assigned_reviewer_id
		ORDER BY c.claim_no`)
	if err != nil {
		t.Fatalf("read claims: %v", err)
	}
	defer rows.Close()

	var got []seededClaim

	for rows.Next() {
		var (
			claim              seededClaim
			itemsMatch         bool
			version            int
			submitted, decided bool
		)
		if err := rows.Scan(&claim.requester, &claim.reviewer, &claim.status, &claim.total, &itemsMatch, &version, &claim.history,
			&claim.reason, &submitted, &decided); err != nil {
			t.Fatalf("scan claim: %v", err)
		}

		if !itemsMatch || !submitted || decided != (claim.status != "SUBMITTED") || (version == 1) != decided {
			t.Fatalf("claim %+v: items match %t, submitted %t, decided %t, version %d; want items summing to the total, a submission, and a decision with version 1 only when decided",
				claim, itemsMatch, submitted, decided, version)
		}

		got = append(got, claim)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("read claims: %v", err)
	}

	if len(got) != len(seededClaims) {
		t.Fatalf("claims = %+v, want %d", got, len(seededClaims))
	}

	for _, want := range seededClaims {
		if !slices.ContainsFunc(got, func(claim seededClaim) bool {
			return claim.requester == want.requester && claim.reviewer == want.reviewer && claim.status == want.status &&
				claim.total == want.total && slices.Equal(claim.history, want.history) && claim.reason == want.reason
		}) {
			t.Fatalf("claims = %+v, want one like %+v", got, want)
		}
	}

	if err := seed(t.Context(), databaseURL, seedPassword, 10, io.Discard); err != nil {
		t.Fatalf("second seed: %v", err)
	}

	if claims2, items2, history2 := counts(); claims2 != claims || items2 != items || history2 != history {
		t.Fatalf("after the second run: %d claims, %d items, %d history entries; want %d, %d, %d",
			claims2, items2, history2, claims, items, history)
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
	if got, want := out.String(), "seeded 7 users and 6 claims\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
