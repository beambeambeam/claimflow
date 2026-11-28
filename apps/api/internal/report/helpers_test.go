package report

import (
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/money"
	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

func TestMain(m *testing.M) {
	os.Exit(pgtest.Run(m))
}

// ict is Asia/Bangkok, which has no daylight saving.
var ict = time.FixedZone("ICT", 7*60*60)

// people are the members of the fixed dataset. Anan reports to Malee and Bella to Somchai. Malee and Somchai are
// reviewers whose own claims go to Wanida, the Finance Officer.
type people struct {
	anan, bella, malee, somchai, wanida uuid.UUID
}

func (p *people) maleeActor() Actor  { return Actor{ID: p.malee, Role: "reviewer"} }
func (p *people) wanidaActor() Actor { return Actor{ID: p.wanida, Role: "finance"} }

// member inserts a member of role with the given reviewer (uuid.Nil for none) and returns the id.
func member(t *testing.T, pool *pgxpool.Pool, name, role string, reviewerID uuid.UUID) uuid.UUID {
	t.Helper()

	reviewer := uuid.NullUUID{UUID: reviewerID, Valid: reviewerID != uuid.Nil}
	id := uuid.Must(uuid.NewV7())

	if _, err := pool.Exec(t.Context(), `
		INSERT INTO "user" (id, name, email, username, display_username, role, position, reviewer_id)
		VALUES ($1, $2, $3, $4, $4, $5, $6, $7)`,
		id, name, name+"@claimflow.test", name, role, "Position of "+name, reviewer); err != nil {
		t.Fatalf("insert member %s: %v", name, err)
	}

	return id
}

func seedPeople(t *testing.T, pool *pgxpool.Pool) *people {
	t.Helper()

	p := &people{}

	p.wanida = member(t, pool, "wanida", "finance", uuid.Nil)
	p.malee = member(t, pool, "malee", "reviewer", p.wanida)
	p.somchai = member(t, pool, "somchai", "reviewer", p.wanida)
	p.anan = member(t, pool, "anan", "requester", p.malee)
	p.bella = member(t, pool, "bella", "requester", p.somchai)

	return p
}

// item is one line item of a stored claim.
type item struct {
	category db.ExpenseCategory
	amount   money.Satang
}

// stored is the stored state of one claim. A zero firstSubmitted and a nil decided are stored as NULL.
type stored struct {
	requester, reviewer uuid.UUID
	status              db.ClaimStatus
	firstSubmitted      time.Time
	decided             *time.Time
	items               []item
}

var claimSeq atomic.Int64

// insertClaim stores c with its line items. A claim without items gets one TRAVEL item of 100.00.
func insertClaim(t *testing.T, pool *pgxpool.Pool, c *stored) {
	t.Helper()

	if len(c.items) == 0 {
		c.items = []item{{db.ExpenseCategoryTRAVEL, 10000}}
	}

	var total money.Satang
	for _, it := range c.items {
		total += it.amount
	}

	id := uuid.Must(uuid.NewV7())
	reviewer := uuid.NullUUID{UUID: c.reviewer, Valid: c.reviewer != uuid.Nil}

	var first *time.Time
	if !c.firstSubmitted.IsZero() {
		first = &c.firstSubmitted
	}

	if _, err := pool.Exec(t.Context(), `
		INSERT INTO claims (id, claim_no, requester_id, assigned_reviewer_id, title, purpose, status, total_satang,
		                    first_submitted_at, submitted_at, decided_at)
		VALUES ($1, $2, $3, $4, 'Workshop travel', 'Purpose', $5, $6, $7, $7, $8)`,
		id, fmt.Sprintf("CLM-T-%06d", claimSeq.Add(1)), c.requester, reviewer, c.status, total, first, c.decided); err != nil {
		t.Fatalf("insert claim: %v", err)
	}

	for i, it := range c.items {
		if _, err := pool.Exec(t.Context(), `
			INSERT INTO line_items (id, claim_id, line_no, category, description, expense_date, amount_satang, receipt_ref)
			VALUES (gen_random_uuid(), $1, $2, $3, 'Item', '2026-10-05', $4, 'REF')`,
			id, i+1, it.category, it.amount); err != nil {
			t.Fatalf("insert line item: %v", err)
		}
	}
}

// at returns a Bangkok time on 2026-month-day.
func at(month time.Month, day, hour, minute, second int) time.Time {
	return time.Date(2026, month, day, hour, minute, second, 0, ict)
}

// decidedAt returns the pointer that stored.decided takes.
func decidedAt(t time.Time) *time.Time { return &t }
