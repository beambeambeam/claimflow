package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// storedClaim inserts a claim of requester with one line item and one history row, as the stored state of status,
// created at createdAt. The claim number is unique per call. It returns the claim id.
func storedClaim(t *testing.T, pool *pgxpool.Pool, requester, status string, createdAt time.Time) uuid.UUID {
	t.Helper()

	id := uuid.Must(uuid.NewV7())

	var number int
	if err := pool.QueryRow(t.Context(), `SELECT nextval('claim_no_seq')`).Scan(&number); err != nil {
		t.Fatalf("claim number: %v", err)
	}

	if _, err := pool.Exec(t.Context(), `
		WITH inserted AS (
			INSERT INTO claims (id, claim_no, requester_id, assigned_reviewer_id, title, purpose, status, total_satang,
			                    first_submitted_at, submitted_at, created_at, updated_at)
			SELECT $1, $2, u.id, u.reviewer_id, 'Bangkok workshop travel', 'Purpose', $3, 125000, $4, $4, $4, $4
			FROM "user" u WHERE u.username = $5
			RETURNING id, requester_id
		), item AS (
			INSERT INTO line_items (id, claim_id, line_no, category, description, expense_date, amount_satang, receipt_ref)
			SELECT gen_random_uuid(), id, 1, 'TRAVEL', 'Train', '2026-11-03', 125000, 'REF-1' FROM inserted
		)
		INSERT INTO claim_history (claim_id, actor_id, action, from_status, to_status, created_at)
		SELECT id, requester_id, 'SUBMIT', NULL, 'SUBMITTED', $4 FROM inserted`,
		id, "CLM-2026-"+strconv.Itoa(1000000 + number)[1:], status, createdAt, requester); err != nil {
		t.Fatalf("insert claim of %s: %v", requester, err)
	}

	return id
}

type claimPageResponse struct {
	Data []struct {
		ID        string `json:"id"`
		ClaimNo   string `json:"claimNo"`
		Status    string `json:"status"`
		Total     string `json:"total"`
		CreatedAt string `json:"createdAt"`
		Requester struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"requester"`
	} `json:"data"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

// listClaims lists as token and decodes the 200 page.
func listClaims(t *testing.T, router http.Handler, token, query string) claimPageResponse {
	t.Helper()

	recorder := getWithSession(t, router, "/api/v1/claims"+query, token)
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /claims%s status = %d, want 200; body = %s", query, recorder.Code, recorder.Body)
	}

	var page claimPageResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode page: %v", err)
	}

	return page
}

// F03-TEST-01 (F03-AC-01): req.anan lists three claims in three statuses, newest first by created_at. The ids run the
// other way round, so the order cannot come from the id.
func TestF03TEST01ListMineNewestFirst(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "req.anan")

	t1 := time.Date(2026, time.November, 1, 3, 0, 0, 0, time.UTC)
	t2, t3 := t1.Add(24*time.Hour), t1.Add(48*time.Hour)

	newest := storedClaim(t, pool, "req.anan", "REJECTED", t3)
	middle := storedClaim(t, pool, "req.anan", "APPROVED", t2)
	oldest := storedClaim(t, pool, "req.anan", "SUBMITTED", t1)
	storedClaim(t, pool, "req.ploy", "SUBMITTED", t2)

	page := listClaims(t, router, token, "?scope=mine")

	if page.Total != 3 || page.Page != 1 || page.PageSize != 20 || len(page.Data) != 3 {
		t.Fatalf("page = %+v, want total 3, page 1, pageSize 20, 3 claims", page)
	}

	want := []struct{ id, status, createdAt string }{
		{newest.String(), "REJECTED", "2026-11-03T03:00:00Z"},
		{middle.String(), "APPROVED", "2026-11-02T03:00:00Z"},
		{oldest.String(), "SUBMITTED", "2026-11-01T03:00:00Z"},
	}
	for i, claim := range page.Data {
		if claim.ID != want[i].id || claim.Status != want[i].status || claim.CreatedAt != want[i].createdAt ||
			claim.Total != "1250.00" || claim.Requester.Name != "Anan K." {
			t.Fatalf("data[%d] = %+v, want claim %s (%s, created %s) of Anan K.", i, claim, want[i].id, want[i].status, want[i].createdAt)
		}
	}
}

// Claims created at the same instant keep a stable order: id descending (F-03 §1.4).
func TestListMineBreaksCreatedAtTiesByIDDescending(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "req.anan")

	at := time.Date(2026, time.November, 1, 3, 0, 0, 0, time.UTC)
	first := storedClaim(t, pool, "req.anan", "SUBMITTED", at)
	second := storedClaim(t, pool, "req.anan", "SUBMITTED", at)

	page := listClaims(t, router, token, "?scope=mine")
	if len(page.Data) != 2 || page.Data[0].ID != second.String() || page.Data[1].ID != first.String() {
		t.Fatalf("data = %+v, want %s then %s", page.Data, second, first)
	}
}

// F03-TEST-02 (F03-AC-02, NFR-02): req.ploy's list holds only req.ploy's claim and no id of req.anan.
func TestF03TEST02ListMineNeverShowsOthersClaims(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "req.ploy")

	at := time.Date(2026, time.November, 1, 3, 0, 0, 0, time.UTC)
	for i, status := range []string{"SUBMITTED", "APPROVED", "REJECTED"} {
		storedClaim(t, pool, "req.anan", status, at.Add(time.Duration(i)*time.Hour))
	}
	own := storedClaim(t, pool, "req.ploy", "SUBMITTED", at)

	recorder := getWithSession(t, router, "/api/v1/claims?scope=mine", token)

	page := listClaims(t, router, token, "?scope=mine")
	if page.Total != 1 || len(page.Data) != 1 || page.Data[0].ID != own.String() || page.Data[0].Requester.Name != "Ploy" {
		t.Fatalf("page = %+v, want only claim %s of Ploy", page, own)
	}

	var anan uuid.UUID
	if err := pool.QueryRow(t.Context(), `SELECT id FROM "user" WHERE username = 'req.anan'`).Scan(&anan); err != nil {
		t.Fatalf("read req.anan: %v", err)
	}

	if got := recorder.Body.String(); strings.Contains(got, anan.String()) {
		t.Fatalf("body holds the id of req.anan: %s", got)
	}
}

// F03-TEST-03 (F03-AC-03): defaults, bounds, a page past the last one, and a missing or unknown scope.
func TestF03TEST03ListPaginationAndBounds(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "req.anan")

	at := time.Date(2026, time.November, 1, 3, 0, 0, 0, time.UTC)
	for i := range 25 {
		storedClaim(t, pool, "req.anan", "SUBMITTED", at.Add(time.Duration(i)*time.Minute))
	}

	pages := []struct {
		query                string
		page, pageSize, size int
	}{
		{"?scope=mine", 1, 20, 20},
		{"?scope=mine&page=2", 2, 20, 5},
		{"?scope=mine&pageSize=100", 1, 100, 25},
		{"?scope=mine&page=9", 9, 20, 0},
	}
	for _, tc := range pages {
		page := listClaims(t, router, token, tc.query)
		if page.Page != tc.page || page.PageSize != tc.pageSize || len(page.Data) != tc.size || page.Total != 25 {
			t.Fatalf("GET /claims%s = page %d, pageSize %d, %d claims, total %d; want page %d, pageSize %d, %d claims, total 25",
				tc.query, page.Page, page.PageSize, len(page.Data), page.Total, tc.page, tc.pageSize, tc.size)
		}
	}

	// A page past the last one has data: [], not null.
	if body := getWithSession(t, router, "/api/v1/claims?scope=mine&page=9", token).Body.String(); !strings.Contains(body, `"data":[]`) {
		t.Fatalf("body = %s, want data: []", body)
	}

	for _, query := range []string{
		"?scope=mine&pageSize=101", "?scope=mine&pageSize=0", "?scope=mine&page=0", "?scope=mine&page=-1",
		"?scope=mine&page=abc", "?scope=mine&pageSize=abc", "", "?page=1", "?scope=bogus", "?scope=review",
	} {
		decodeProblem(t, getWithSession(t, router, "/api/v1/claims"+query, token), http.StatusBadRequest, "BAD_REQUEST")
	}
}

// F03-TEST-04 (F03-AC-04, NFR-02): a Finance Officer has no claim:list, so scope=mine is 403.
func TestF03TEST04FinanceOfficerCannotListMine(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "fin.wanida")

	decodeProblem(t, getWithSession(t, router, "/api/v1/claims?scope=mine", token), http.StatusForbidden, "FORBIDDEN")
}
