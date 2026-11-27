package main

import (
	"bytes"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beambeambeam/claimflow/apps/api/internal/claim"
	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/money"
	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// loadActions maps a history action back to the claim action that causes it.
var loadActions = map[db.ClaimAction]claim.Action{
	db.ClaimActionSUBMIT:       claim.ActionSubmit,
	db.ClaimActionRESUBMIT:     claim.ActionSubmit,
	db.ClaimActionAPPROVE:      claim.ActionApprove,
	db.ClaimActionREJECT:       claim.ActionReject,
	db.ClaimActionRETURN:       claim.ActionReturn,
	db.ClaimActionWITHDRAW:     claim.ActionWithdraw,
	db.ClaimActionFINALAPPROVE: claim.ActionFinalApprove,
	db.ClaimActionFINALREJECT:  claim.ActionFinalReject,
}

// loadCount counts rows with query, failing the test on an error.
func loadCount(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int {
	t.Helper()

	var n int
	if err := pool.QueryRow(t.Context(), query, args...).Scan(&n); err != nil {
		t.Fatalf("count (%s): %v", query, err)
	}

	return n
}

// loadStrings returns the first column of every row of query as strings.
func loadStrings(t *testing.T, pool *pgxpool.Pool, query string) []string {
	t.Helper()

	rows, err := pool.Query(t.Context(), query)
	if err != nil {
		t.Fatalf("query (%s): %v", query, err)
	}
	defer rows.Close()

	var values []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			t.Fatalf("scan (%s): %v", query, err)
		}

		values = append(values, value)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("read (%s): %v", query, err)
	}

	return values
}

// F10-TEST-10 (F10-AC-10, NFR-07): seed-load adds at least 200 claims on top of the base seed, spread over every
// non-draft status, category, requester and reviewer, and five decision months. A second run adds nothing. Every history
// is a valid path of the lifecycle, and every claim's line items pass the BR rules.
func TestF10TEST10SeedLoadAddsValidSpreadOnce(t *testing.T) {
	pool := pgtest.New(t)
	connString := pool.Config().ConnString()

	var base bytes.Buffer
	if err := seed(t.Context(), connString, seedPassword, testBcryptCost, &base); err != nil {
		t.Fatalf("seed: %v", err)
	}

	const countClaims = `SELECT count(*) FROM claims WHERE claim_no LIKE 'CLM-LOAD-%'`

	var first bytes.Buffer
	if err := seedLoad(t.Context(), connString, &first); err != nil {
		t.Fatalf("seedLoad: %v", err)
	}

	if got := first.String(); got != "added 240 new load claims\n" {
		t.Fatalf("first output = %q", got)
	}

	claims := loadCount(t, pool, countClaims)
	items := loadCount(t, pool, `SELECT count(*) FROM line_items i JOIN claims c ON c.id = i.claim_id WHERE c.claim_no LIKE 'CLM-LOAD-%'`)
	history := loadCount(t, pool, `SELECT count(*) FROM claim_history h JOIN claims c ON c.id = h.claim_id WHERE c.claim_no LIKE 'CLM-LOAD-%'`)

	if claims < 200 {
		t.Fatalf("load claims = %d, want at least 200", claims)
	}

	var second bytes.Buffer
	if err := seedLoad(t.Context(), connString, &second); err != nil {
		t.Fatalf("second seedLoad: %v", err)
	}

	if got := second.String(); got != "added 0 new load claims\n" {
		t.Fatalf("second output = %q", got)
	}

	if got := loadCount(t, pool, countClaims); got != claims {
		t.Fatalf("load claims after the second run = %d, want %d", got, claims)
	}

	if got := loadCount(t, pool, `SELECT count(*) FROM line_items i JOIN claims c ON c.id = i.claim_id WHERE c.claim_no LIKE 'CLM-LOAD-%'`); got != items {
		t.Fatalf("line items after the second run = %d, want %d", got, items)
	}

	if got := loadCount(t, pool, `SELECT count(*) FROM claim_history h JOIN claims c ON c.id = h.claim_id WHERE c.claim_no LIKE 'CLM-LOAD-%'`); got != history {
		t.Fatalf("history after the second run = %d, want %d", got, history)
	}

	if got := loadCount(t, pool, `SELECT count(*) FROM claims`); got != claims+10 {
		t.Fatalf("all claims = %d, want the %d load claims and the 10 base ones", got, claims)
	}

	t.Run("spread", func(t *testing.T) {
		assertLoadSpread(t, pool)
	})
	t.Run("histories replay through the lifecycle", func(t *testing.T) {
		assertLoadHistories(t, pool)
	})
	t.Run("line items pass the BR rules", func(t *testing.T) {
		assertLoadItems(t, pool)
	})
}

// assertLoadSpread checks that the load claims cover every status, category, owner, reviewer, and decision month, and
// that at least 60 wait for rev.malee.
func assertLoadSpread(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	statuses := loadStrings(t, pool, `SELECT DISTINCT status::text FROM claims WHERE claim_no LIKE 'CLM-LOAD-%' ORDER BY 1`)
	if want := []string{"APPROVED", "PENDING_FINANCE", "REJECTED", "RETURNED", "SUBMITTED", "WITHDRAWN"}; !slices.Equal(statuses, want) {
		t.Fatalf("statuses = %v, want %v", statuses, want)
	}

	categories := loadStrings(t, pool, `SELECT DISTINCT i.category::text FROM line_items i JOIN claims c ON c.id = i.claim_id WHERE c.claim_no LIKE 'CLM-LOAD-%' ORDER BY 1`)
	if want := []string{"ACCOMMODATION", "MEALS", "OTHER", "SUPPLIES", "TRAINING", "TRAVEL"}; !slices.Equal(categories, want) {
		t.Fatalf("categories = %v, want %v", categories, want)
	}

	months := loadStrings(t, pool, `SELECT DISTINCT to_char(decided_at AT TIME ZONE 'Asia/Bangkok', 'YYYY-MM') FROM claims WHERE claim_no LIKE 'CLM-LOAD-%' AND decided_at IS NOT NULL ORDER BY 1`)
	if want := []string{"2026-07", "2026-08", "2026-09", "2026-10", "2026-11"}; !slices.Equal(months, want) {
		t.Fatalf("decision months = %v, want %v", months, want)
	}

	requesters := loadStrings(t, pool, `SELECT DISTINCT u.username FROM claims c JOIN "user" u ON u.id = c.requester_id WHERE c.claim_no LIKE 'CLM-LOAD-%' ORDER BY 1`)
	if want := []string{"req.anan", "req.niran", "req.ploy", "rev.malee", "rev.prasert", "rev.somchai"}; !slices.Equal(requesters, want) {
		t.Fatalf("requesters = %v, want %v", requesters, want)
	}

	reviewers := loadStrings(t, pool, `SELECT DISTINCT u.username FROM claims c JOIN "user" u ON u.id = c.assigned_reviewer_id WHERE c.claim_no LIKE 'CLM-LOAD-%' ORDER BY 1`)
	if want := []string{"fin.wanida", "rev.malee", "rev.prasert", "rev.somchai"}; !slices.Equal(reviewers, want) {
		t.Fatalf("reviewers = %v, want %v", reviewers, want)
	}

	waiting := loadCount(t, pool, `SELECT count(*) FROM claims c JOIN "user" u ON u.id = c.assigned_reviewer_id
		WHERE c.claim_no LIKE 'CLM-LOAD-%' AND c.status = 'SUBMITTED' AND u.username = 'rev.malee'`)
	if waiting < 60 {
		t.Fatalf("SUBMITTED to rev.malee = %d, want at least 60", waiting)
	}

	// The reviewer is the requester's own, and the stored times match the status.
	if bad := loadCount(t, pool, `SELECT count(*) FROM claims c JOIN "user" r ON r.id = c.requester_id
		WHERE c.claim_no LIKE 'CLM-LOAD-%' AND (c.assigned_reviewer_id IS DISTINCT FROM r.reviewer_id
			OR c.first_submitted_at IS NULL OR c.submitted_at < c.first_submitted_at OR c.created_at > c.first_submitted_at
			OR (c.decided_at IS NOT NULL) <> (c.status IN ('APPROVED', 'REJECTED'))
			OR c.updated_at < c.submitted_at OR c.total_satang > 5000000 OR c.return_count > 3)`); bad != 0 {
		t.Fatalf("%d load claims have a reviewer or time that does not match", bad)
	}

	// A claim over 10,000.00 that one reviewer approves waits for Finance, and one under it is approved.
	if bad := loadCount(t, pool, `SELECT count(*) FROM claims c
		WHERE c.claim_no LIKE 'CLM-LOAD-%' AND c.status = 'PENDING_FINANCE' AND c.total_satang <= 1000000`); bad != 0 {
		t.Fatalf("%d PENDING_FINANCE load claims are not over 10,000.00", bad)
	}

	for _, action := range []string{"SUBMIT", "RESUBMIT", "APPROVE", "REJECT", "RETURN", "WITHDRAW", "FINAL_APPROVE", "FINAL_REJECT"} {
		if loadCount(t, pool, `SELECT count(*) FROM claim_history h JOIN claims c ON c.id = h.claim_id
			WHERE c.claim_no LIKE 'CLM-LOAD-%' AND h.action = $1::claim_action`, action) == 0 {
			t.Errorf("no load claim has a %s history entry", action)
		}
	}
}

// assertLoadHistories replays every load claim's history through the lifecycle (internal/claim) and checks the path.
func assertLoadHistories(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	rows, err := pool.Query(t.Context(), `
		SELECT c.claim_no, c.status, c.total_satang, c.version, c.return_count, c.first_submitted_at, c.submitted_at, c.decided_at,
		       h.action, h.from_status, h.to_status, h.created_at, a.role, a.username, r.username, v.username
		FROM claims c
		JOIN "user" r ON r.id = c.requester_id
		JOIN "user" v ON v.id = c.assigned_reviewer_id
		JOIN claim_history h ON h.claim_id = c.id
		JOIN "user" a ON a.id = h.actor_id
		WHERE c.claim_no LIKE 'CLM-LOAD-%'
		ORDER BY c.claim_no, h.id`)
	if err != nil {
		t.Fatalf("read histories: %v", err)
	}
	defer rows.Close()

	type replay struct {
		status                   db.ClaimStatus
		steps, returns           int
		first, submitted, last   time.Time
		wantFirst, wantSubmitted time.Time
		wantDecided              *time.Time
		wantStatus               db.ClaimStatus
		wantVersion              int32
		wantReturns              int16
	}

	paths := map[string]*replay{}
	for rows.Next() {
		var (
			claimNo, role, actor, requester, reviewer string
			status, to                                db.ClaimStatus
			from                                      *db.ClaimStatus
			action                                    db.ClaimAction
			total                                     money.Satang
			version                                   int32
			returns                                   int16
			firstAt, submittedAt                      time.Time
			decidedAt                                 *time.Time
			at                                        time.Time
		)

		if err := rows.Scan(&claimNo, &status, &total, &version, &returns, &firstAt, &submittedAt, &decidedAt,
			&action, &from, &to, &at, &role, &actor, &requester, &reviewer); err != nil {
			t.Fatalf("scan history: %v", err)
		}

		path, ok := paths[claimNo]
		if !ok {
			path = &replay{
				status: db.ClaimStatusDRAFT, wantFirst: firstAt, wantSubmitted: submittedAt, wantDecided: decidedAt,
				wantStatus: status, wantVersion: version, wantReturns: returns,
			}
			paths[claimNo] = path
		}

		next, history, allowed := claim.NextStatus(loadActions[action], path.status, total, role)
		switch {
		case !allowed:
			t.Fatalf("%s: %s is not allowed from %s", claimNo, action, path.status)
		case history != action || next != to:
			t.Fatalf("%s: %s from %s leads to %s with %s, the history says %s with %s", claimNo, action, path.status, next, history, to, action)
		case from == nil || *from != path.status:
			t.Fatalf("%s: %s has from_status %v, want %s", claimNo, action, from, path.status)
		case path.steps > 0 && !at.After(path.last):
			t.Fatalf("%s: %s is not after the entry before it", claimNo, action)
		}

		// Who acts: the requester submits and withdraws, the assigned reviewer decides, Finance gives the final decision.
		wantActor := reviewer
		switch action {
		case db.ClaimActionSUBMIT, db.ClaimActionRESUBMIT, db.ClaimActionWITHDRAW:
			wantActor = requester
		case db.ClaimActionFINALAPPROVE, db.ClaimActionFINALREJECT:
			wantActor = "fin.wanida"
		case db.ClaimActionAPPROVE, db.ClaimActionREJECT, db.ClaimActionRETURN:
			// The assigned reviewer decides.
		}

		if actor != wantActor {
			t.Fatalf("%s: %s is by %s, want %s", claimNo, action, actor, wantActor)
		}

		if action == db.ClaimActionSUBMIT {
			if path.steps != 0 {
				t.Fatalf("%s: SUBMIT is not the first entry", claimNo)
			}

			path.first = at
		}

		if action == db.ClaimActionSUBMIT || action == db.ClaimActionRESUBMIT {
			path.submitted = at
		}

		if action == db.ClaimActionRETURN {
			path.returns++
		}

		path.status, path.last, path.steps = to, at, path.steps+1
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("read histories: %v", err)
	}

	if len(paths) != loadClaimCount {
		t.Fatalf("replayed %d histories, want %d", len(paths), loadClaimCount)
	}

	for claimNo, path := range paths {
		switch {
		case path.status != path.wantStatus:
			t.Errorf("%s: history ends in %s, the claim is %s", claimNo, path.status, path.wantStatus)
		case path.steps != int(path.wantVersion):
			t.Errorf("%s: %d history entries, version %d", claimNo, path.steps, path.wantVersion)
		case path.returns != int(path.wantReturns):
			t.Errorf("%s: %d returns in the history, return_count %d", claimNo, path.returns, path.wantReturns)
		case !path.first.Equal(path.wantFirst) || !path.submitted.Equal(path.wantSubmitted):
			t.Errorf("%s: first and last submission times do not match the history", claimNo)
		case (path.wantDecided == nil) == (path.status == db.ClaimStatusAPPROVED || path.status == db.ClaimStatusREJECTED):
			t.Errorf("%s: decided_at = %v for status %s", claimNo, path.wantDecided, path.status)
		case path.wantDecided != nil && !path.wantDecided.Equal(path.last):
			t.Errorf("%s: decided_at is not the time of the last history entry", claimNo)
		}
	}
}

// assertLoadItems checks that every load claim has one to five line items that pass the BR rules, and that they add up
// to the claim's total.
func assertLoadItems(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	rows, err := pool.Query(t.Context(), `
		SELECT c.claim_no, c.total_satang, c.first_submitted_at, c.updated_at, i.category, i.description, i.expense_date, i.amount_satang, i.receipt_ref
		FROM claims c JOIN line_items i ON i.claim_id = c.id
		WHERE c.claim_no LIKE 'CLM-LOAD-%'
		ORDER BY c.claim_no, i.line_no`)
	if err != nil {
		t.Fatalf("read line items: %v", err)
	}
	defer rows.Close()

	type stored struct {
		total    money.Satang
		first    time.Time
		updated  time.Time
		items    []claim.LineItemInput
		itemsSum money.Satang
	}

	claims := map[string]*stored{}
	for rows.Next() {
		var (
			claimNo string
			item    claim.LineItemInput
			total   money.Satang
			first   time.Time
			updated time.Time
		)
		if err := rows.Scan(&claimNo, &total, &first, &updated, &item.Category, &item.Description, &item.ExpenseDate, &item.Amount, &item.ReceiptRef); err != nil {
			t.Fatalf("scan line item: %v", err)
		}

		if claims[claimNo] == nil {
			claims[claimNo] = &stored{total: total, first: first, updated: updated}
		}

		claims[claimNo].items = append(claims[claimNo].items, item)
		claims[claimNo].itemsSum += item.Amount
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("read line items: %v", err)
	}

	if len(claims) != loadClaimCount {
		t.Fatalf("line items of %d claims, want %d", len(claims), loadClaimCount)
	}

	for claimNo, got := range claims {
		if n := len(got.items); n < 1 || n > 5 {
			t.Errorf("%s: %d line items, want 1 to 5", claimNo, n)
		}

		if got.itemsSum != got.total {
			t.Errorf("%s: line items add up to %d, the total is %d", claimNo, got.itemsSum, got.total)
		}

		if err := claim.CheckLineItems(got.items, got.updated, got.first); err != nil {
			t.Errorf("%s: line items break a rule: %v", claimNo, err)
		}
	}
}

// F10-TEST-10 (F10-AC-10): the dataset is the same on every build, so every run and every database gets the same claims.
func TestF10TEST10LoadDatasetIsDeterministic(t *testing.T) {
	first, err := buildLoadClaims()
	if err != nil {
		t.Fatalf("buildLoadClaims: %v", err)
	}

	second, err := buildLoadClaims()
	if err != nil {
		t.Fatalf("second buildLoadClaims: %v", err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatal("two builds of the load dataset differ")
	}

	if len(first) != loadClaimCount || first[0].claimNo != "CLM-LOAD-0001" || first[len(first)-1].claimNo != "CLM-LOAD-0240" {
		t.Fatalf("claims = %d from %s to %s, want %d from CLM-LOAD-0001 to CLM-LOAD-0240", len(first), first[0].claimNo, first[len(first)-1].claimNo, loadClaimCount)
	}
}

// F10-TEST-10 (F10-AC-10): seed-load needs the base seed users. Without them it fails and writes nothing.
func TestSeedLoadNeedsBaseSeedUsers(t *testing.T) {
	pool := pgtest.New(t)

	var out bytes.Buffer
	err := seedLoad(t.Context(), pool.Config().ConnString(), &out)
	if !errors.Is(err, errLoadNeedsSeed) {
		t.Fatalf("seedLoad without users: err = %v, want %v", err, errLoadNeedsSeed)
	}

	if !strings.Contains(err.Error(), "run seed first") || out.Len() != 0 {
		t.Fatalf("error = %q, output = %q", err, out.String())
	}

	if got := loadCount(t, pool, `SELECT count(*) FROM claims`); got != 0 {
		t.Fatalf("claims = %d, want 0", got)
	}
}
