package migrations_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// insertSubmittedClaim writes a SUBMITTED claim with one line item and a SUBMIT and an APPROVE history row,
// straight to the tables, so the test depends only on the schema.
func insertSubmittedClaim(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()

	requesterID := insertUser(t, pool, "requester")
	reviewerID := insertUser(t, pool, "reviewer")
	claimID := uuid.Must(uuid.NewV7())

	statements := []struct {
		sql  string
		args []any
	}{
		{
			`INSERT INTO claims (id, claim_no, requester_id, assigned_reviewer_id, title, purpose, status, total_satang,
			first_submitted_at, submitted_at)
			VALUES ($1, 'CLM-2026-000001', $2, $3, 'Travel', 'Workshop', 'SUBMITTED', 25000, now(), now())`,
			[]any{claimID, requesterID, reviewerID},
		},
		{
			`INSERT INTO line_items (id, claim_id, line_no, category, description, expense_date, amount_satang, receipt_ref)
			VALUES ($1, $2, 1, 'TRAVEL', 'Train', '2026-11-03', 25000, 'SRT-1')`,
			[]any{uuid.Must(uuid.NewV7()), claimID},
		},
		{`INSERT INTO claim_history (claim_id, actor_id, action, from_status, to_status)
			VALUES ($1, $2, 'SUBMIT', NULL, 'SUBMITTED')`, []any{claimID, requesterID}},
		{`INSERT INTO claim_history (claim_id, actor_id, action, from_status, to_status, comment)
			VALUES ($1, $2, 'APPROVE', 'SUBMITTED', 'APPROVED', 'ok')`, []any{claimID, reviewerID}},
	}

	for _, statement := range statements {
		if _, err := pool.Exec(t.Context(), statement.sql, statement.args...); err != nil {
			t.Fatalf("seed claim: %v", err)
		}
	}

	return claimID
}

// historySnapshot returns every history row as text, in id order.
func historySnapshot(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()

	rows, err := pool.Query(t.Context(),
		`SELECT concat_ws('|', id, claim_id, actor_id, action, from_status, to_status, comment, created_at)
		FROM claim_history ORDER BY id`)
	if err != nil {
		t.Fatalf("read history: %v", err)
	}
	defer rows.Close()

	var snapshot []string
	for rows.Next() {
		var row string
		if err := rows.Scan(&row); err != nil {
			t.Fatalf("scan history: %v", err)
		}
		snapshot = append(snapshot, row)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("read history: %v", err)
	}

	return snapshot
}

// F04-TEST-18 (F04-AC-15, BR-11): UPDATE and DELETE on claim_history fail with a database error, and a claim
// with history cannot be deleted. Every row stays as it was.
func TestF04TEST18HistoryIsAppendOnly(t *testing.T) {
	pool := pgtest.New(t)
	claimID := insertSubmittedClaim(t, pool)
	before := historySnapshot(t, pool)

	if len(before) != 2 {
		t.Fatalf("history rows = %d, want 2", len(before))
	}

	statements := map[string]string{
		"update history": `UPDATE claim_history SET comment = 'x'`,
		"delete history": `DELETE FROM claim_history`,
		"delete claim":   `DELETE FROM claims WHERE id = '` + claimID.String() + `'`,
	}

	for name, statement := range statements {
		t.Run(name, func(t *testing.T) {
			if _, err := pool.Exec(t.Context(), statement); err == nil {
				t.Fatalf("%s succeeded, want a database error", statement)
			}

			after := historySnapshot(t, pool)
			if len(after) != len(before) {
				t.Fatalf("history rows = %d after %q, want %d", len(after), statement, len(before))
			}

			for i := range before {
				if after[i] != before[i] {
					t.Fatalf("history row %d = %q after %q, want %q", i, after[i], statement, before[i])
				}
			}

			var claims int
			if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM claims WHERE id = $1`, claimID).Scan(&claims); err != nil {
				t.Fatalf("count claims: %v", err)
			}

			if claims != 1 {
				t.Fatalf("claims = %d after %q, want 1", claims, statement)
			}
		})
	}
}

// F04-TEST-10 (F04-AC-07, BR-01): a claim cannot be assigned to its own requester. The CHECK constraint refuses the
// row, and no claim is stored.
func TestF04TEST10ClaimCannotBeAssignedToItsRequester(t *testing.T) {
	pool := pgtest.New(t)
	requesterID := insertUser(t, pool, "requester")

	_, err := pool.Exec(t.Context(), `
		INSERT INTO claims (id, claim_no, requester_id, assigned_reviewer_id, title, purpose, status, total_satang,
		first_submitted_at, submitted_at)
		VALUES ($1, 'CLM-2026-000001', $2, $2, 'Travel', 'Workshop', 'SUBMITTED', 25000, now(), now())`,
		uuid.Must(uuid.NewV7()), requesterID)

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		t.Fatalf("insert error = %v, want a check violation (23514)", err)
	}

	var claims int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM claims`).Scan(&claims); err != nil {
		t.Fatalf("count claims: %v", err)
	}

	if claims != 0 {
		t.Fatalf("claims = %d, want 0", claims)
	}
}
