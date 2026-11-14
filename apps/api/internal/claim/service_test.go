package claim

import (
	"errors"
	"os"
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

// fixedNow is the clock of the module tests: 5 Nov 2026 10:00 Bangkok (F-02 §6).
var fixedNow = time.Date(2026, time.November, 5, 10, 0, 0, 0, time.FixedZone("ICT", 7*60*60))

func newService(pool *pgxpool.Pool) Service {
	return NewService(pool, func() time.Time { return fixedNow })
}

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

// twoItems returns the items of F02-TEST-01: 250.00 and 1,000.00.
func twoItems() []LineItemInput {
	return []LineItemInput{
		{
			Category:    db.ExpenseCategory("TRAVEL"),
			Description: "Train BKK-Ayutthaya return",
			ExpenseDate: time.Date(2026, time.November, 3, 0, 0, 0, 0, time.UTC),
			Amount:      money.Satang(25000),
			ReceiptRef:  "SRT-88412",
		},
		{
			Category:    db.ExpenseCategory("ACCOMMODATION"),
			Description: "Hotel, one night",
			ExpenseDate: time.Date(2026, time.November, 4, 0, 0, 0, 0, time.UTC),
			Amount:      money.Satang(100000),
			ReceiptRef:  "HTL-1",
		},
	}
}

func validInput() Input {
	return Input{Title: "Bangkok workshop travel", Purpose: "Train fares and hotel for the 3 Nov workshop", Items: twoItems()}
}

// Row counts of every table that Create writes, keyed by table name.
var countQueries = map[string]string{
	"claims":        "SELECT count(*) FROM claims",
	"line_items":    "SELECT count(*) FROM line_items",
	"claim_history": "SELECT count(*) FROM claim_history",
}

func count(t *testing.T, pool *pgxpool.Pool, table string) int {
	t.Helper()

	var n int
	if err := pool.QueryRow(t.Context(), countQueries[table]).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}

	return n
}

// F02-TEST-02: the claim number uses the Bangkok year of the clock, not the UTC year.
func TestF02TEST02ClaimNumberUsesBangkokYear(t *testing.T) {
	now := time.Date(2026, time.December, 31, 17, 30, 0, 0, time.UTC) // 1 Jan 2027, 00:30 in Bangkok

	if got := claimNumber(now, 123); got != "CLM-2027-000123" {
		t.Fatalf("claimNumber = %q, want CLM-2027-000123", got)
	}
}

// F02-TEST-03 (F02-AC-02): when the history insert fails, the claim and its line items are rolled back.
func TestF02TEST03HistoryFailureRollsBackClaimAndLineItems(t *testing.T) {
	pool := pgtest.New(t)
	reviewerID := member(t, pool, "rev.malee", "reviewer", uuid.Nil)
	requesterID := member(t, pool, "req.anan", "requester", reviewerID)

	// A test-only trigger makes every history insert fail.
	if _, err := pool.Exec(t.Context(), `
		CREATE FUNCTION fail_history_insert() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'forced history failure'; END; $$;
		CREATE TRIGGER fail_history_insert BEFORE INSERT ON claim_history
		FOR EACH ROW EXECUTE FUNCTION fail_history_insert();`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	if _, err := newService(pool).Create(t.Context(), Actor{ID: requesterID}, validInput(), true); err == nil {
		t.Fatal("Create succeeded, want the history failure")
	}

	for _, table := range []string{"claims", "line_items", "claim_history"} {
		if got := count(t, pool, table); got != 0 {
			t.Fatalf("%s rows = %d after the rollback, want 0", table, got)
		}
	}
}

// F02-TEST-15 (F02-AC-09, BR-15): a member with no assigned reviewer cannot create a claim.
func TestF02TEST15CreateWithoutReviewerIsForbidden(t *testing.T) {
	pool := pgtest.New(t)
	requesterID := member(t, pool, "lonely", "requester", uuid.Nil)

	_, err := newService(pool).Create(t.Context(), Actor{ID: requesterID}, validInput(), true)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Create error = %v, want ErrForbidden", err)
	}

	for _, table := range []string{"claims", "line_items", "claim_history"} {
		if got := count(t, pool, table); got != 0 {
			t.Fatalf("%s rows = %d, want 0", table, got)
		}
	}
}

// F02-TEST-16 (F02-AC-10, BR-17): the assigned reviewer is copied at submission and does not follow later changes.
func TestF02TEST16AssignedReviewerStaysAfterReviewerChange(t *testing.T) {
	pool := pgtest.New(t)
	malee := member(t, pool, "rev.malee", "reviewer", uuid.Nil)
	somchai := member(t, pool, "rev.somchai", "reviewer", uuid.Nil)
	anan := member(t, pool, "req.anan", "requester", malee)
	service := newService(pool)

	created, err := service.Create(t.Context(), Actor{ID: anan}, validInput(), true)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := pool.Exec(t.Context(), `UPDATE "user" SET reviewer_id = $1 WHERE id = $2`, somchai, anan); err != nil {
		t.Fatalf("change reviewer: %v", err)
	}

	got, err := load(t.Context(), db.New(pool), created.ID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if got.AssignedReviewer == nil || got.AssignedReviewer.ID != malee {
		t.Fatalf("assigned reviewer = %+v, want rev.malee (%s)", got.AssignedReviewer, malee)
	}
}

// The stored claim: status, number, total, times, version, items in request order, and the first history entry (F02-AC-01).
func TestCreateStoresSubmittedClaim(t *testing.T) {
	pool := pgtest.New(t)
	malee := member(t, pool, "rev.malee", "reviewer", uuid.Nil)
	anan := member(t, pool, "req.anan", "requester", malee)

	got, err := newService(pool).Create(t.Context(), Actor{ID: anan}, validInput(), true)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if got.Status != db.ClaimStatus("SUBMITTED") || got.Version != 0 || got.ReturnCount != 0 || got.Total != 125000 {
		t.Fatalf("claim = %+v, want SUBMITTED, version 0, return count 0, total 125000", got)
	}

	if got.ClaimNo[:9] != "CLM-2026-" || len(got.ClaimNo) != len("CLM-2026-000001") {
		t.Fatalf("claim number = %q, want CLM-2026-<6 digits>", got.ClaimNo)
	}

	if got.FirstSubmittedAt == nil || !got.FirstSubmittedAt.Equal(fixedNow) || got.SubmittedAt == nil || !got.SubmittedAt.Equal(fixedNow) {
		t.Fatalf("submission times = %v and %v, want %v", got.FirstSubmittedAt, got.SubmittedAt, fixedNow)
	}

	if got.Requester.ID != anan || got.AssignedReviewer == nil || got.AssignedReviewer.ID != malee {
		t.Fatalf("members = %+v and %+v, want req.anan and rev.malee", got.Requester, got.AssignedReviewer)
	}

	if len(got.Items) != 2 || got.Items[0].LineNo != 1 || got.Items[1].LineNo != 2 || got.Items[1].Amount != 100000 {
		t.Fatalf("items = %+v, want 2 items in request order", got.Items)
	}

	if len(got.History) != 1 || got.History[0].Action != db.ClaimAction("SUBMIT") || got.History[0].FromStatus != nil ||
		got.History[0].ToStatus != db.ClaimStatus("SUBMITTED") || got.History[0].Actor.ID != anan {
		t.Fatalf("history = %+v, want one SUBMIT entry from nil to SUBMITTED by req.anan", got.History)
	}

	if len(got.AllowedActions) != 0 {
		t.Fatalf("allowed actions = %v, want none", got.AllowedActions)
	}
}

// Sprint 1 always submits; saving a draft is F-05.
func TestCreateDraftIsNotSupportedYet(t *testing.T) {
	pool := pgtest.New(t)
	malee := member(t, pool, "rev.malee", "reviewer", uuid.Nil)
	anan := member(t, pool, "req.anan", "requester", malee)

	if _, err := newService(pool).Create(t.Context(), Actor{ID: anan}, validInput(), false); !errors.Is(err, ErrDraftNotSupported) {
		t.Fatalf("Create draft error = %v, want ErrDraftNotSupported", err)
	}

	if got := count(t, pool, "claims"); got != 0 {
		t.Fatalf("claims = %d, want 0", got)
	}
}

func TestGetUnknownClaimIsNotFound(t *testing.T) {
	pool := pgtest.New(t)

	if _, err := load(t.Context(), db.New(pool), uuid.Must(uuid.NewV7())); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get error = %v, want ErrNotFound", err)
	}
}

// Get also reads claims that later sprints write: a draft has no reviewer and no submission time, and a decided claim has
// a decision time and history entries with a from-status and a comment. The rows are inserted directly.
func TestGetReadsNullableClaimFields(t *testing.T) {
	pool := pgtest.New(t)
	anan := member(t, pool, "req.anan", "requester", uuid.Nil)
	id := uuid.Must(uuid.NewV7())

	for _, statement := range []string{
		`INSERT INTO claims (id, claim_no, requester_id, title, purpose, status, total_satang, decided_at)
		VALUES ($1, 'CLM-2026-000001', $2, 'Draft', 'Purpose', 'DRAFT', 100, '2026-11-05T03:00:00Z')`,
		`INSERT INTO claim_history (claim_id, actor_id, action, from_status, to_status, comment)
		VALUES ($1, $2, 'REJECT', 'SUBMITTED', 'REJECTED', 'No receipt')`,
	} {
		if _, err := pool.Exec(t.Context(), statement, id, anan); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	got, err := load(t.Context(), db.New(pool), id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if got.AssignedReviewer != nil || got.FirstSubmittedAt != nil || got.SubmittedAt != nil || got.DecidedAt == nil {
		t.Fatalf("claim = %+v, want no reviewer, no submission times, and a decision time", got)
	}

	if len(got.History) != 1 || got.History[0].FromStatus == nil || *got.History[0].FromStatus != db.ClaimStatus("SUBMITTED") ||
		got.History[0].Comment == nil || *got.History[0].Comment != "No receipt" {
		t.Fatalf("history = %+v, want one entry from SUBMITTED with the comment", got.History)
	}
}
