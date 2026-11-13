package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// claimsNow is the clock of the claim tests: 5 Nov 2026 10:00 Bangkok (F-02 §6).
var claimsNow = time.Date(2026, time.November, 5, 10, 0, 0, 0, time.FixedZone("ICT", 7*60*60))

// twoItemsBody is the request of F02-TEST-01: two items of 250.00 and 1,000.00. extra is JSON members added to the body.
func twoItemsBody(extra string) string {
	return `{` + extra + `
		"title": "Bangkok workshop travel",
		"purpose": "Train fares and hotel for the 3 Nov workshop",
		"items": [
			{"category": "TRAVEL", "description": "Train BKK-Ayutthaya return", "expenseDate": "2026-11-03", "amount": "250.00", "receiptRef": "SRT-88412"},
			{"category": "ACCOMMODATION", "description": "Hotel, one night", "expenseDate": "2026-11-04", "amount": "1000.00", "receiptRef": "HTL-1"}
		]
	}`
}

// seededClaimRouter seeds the demo users and returns a router on the claim clock.
func seededClaimRouter(t *testing.T, pool *pgxpool.Pool) http.Handler {
	t.Helper()

	if err := seed(t.Context(), pool.Config().ConnString(), seedPassword, 10, io.Discard); err != nil {
		t.Fatalf("seed: %v", err)
	}

	return newRouterAt(discardLogger(), testEnv(0), pool, func() time.Time { return claimsNow })
}

// loginAs logs in a seeded user and returns the session token.
func loginAs(t *testing.T, router http.Handler, username string) string {
	t.Helper()

	login := postLogin(t, router, username, seedPassword)
	if login.Code != http.StatusOK {
		t.Fatalf("login %s status = %d, want 200; body = %s", username, login.Code, login.Body)
	}

	return sessionToken(t, login, "sid", sessionAttributesPlain)
}

// postClaim sends body to POST /claims with the session cookie.
func postClaim(t *testing.T, router http.Handler, token, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/claims", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "sid", Value: token, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return recorder
}

func rowCount(t *testing.T, pool *pgxpool.Pool, query string) int {
	t.Helper()

	var n int
	if err := pool.QueryRow(t.Context(), query).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}

	return n
}

type claimResponse struct {
	Data struct {
		ID               string `json:"id"`
		ClaimNo          string `json:"claimNo"`
		Status           string `json:"status"`
		Total            string `json:"total"`
		ReturnCount      int    `json:"returnCount"`
		Version          int    `json:"version"`
		FirstSubmittedAt string `json:"firstSubmittedAt"`
		SubmittedAt      string `json:"submittedAt"`
		Requester        struct {
			Name     string `json:"name"`
			Position string `json:"position"`
		} `json:"requester"`
		AssignedReviewer struct {
			Name     string `json:"name"`
			Position string `json:"position"`
		} `json:"assignedReviewer"`
		Items []struct {
			LineNo      int    `json:"lineNo"`
			Category    string `json:"category"`
			ExpenseDate string `json:"expenseDate"`
			Amount      string `json:"amount"`
		} `json:"items"`
		History []struct {
			Action     string  `json:"action"`
			FromStatus *string `json:"fromStatus"`
			ToStatus   string  `json:"toStatus"`
			Actor      struct {
				Name string `json:"name"`
			} `json:"actor"`
		} `json:"history"`
		AllowedActions []string `json:"allowedActions"`
	} `json:"data"`
}

// F02-TEST-01 (F02-AC-01): req.anan submits two valid items and gets the stored, submitted claim back.
func TestF02TEST01SubmitValidClaim(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "req.anan")

	recorder := postClaim(t, router, token, twoItemsBody(""))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", recorder.Code, recorder.Body)
	}

	var body claimResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	claim := body.Data

	if got := recorder.Header().Get("ETag"); got != `"0"` {
		t.Fatalf("ETag = %q, want \"0\"", got)
	}

	if got, want := recorder.Header().Get("Location"), "/api/v1/claims/"+claim.ID; got != want {
		t.Fatalf("Location = %q, want %q", got, want)
	}

	if !regexp.MustCompile(`^CLM-2026-\d{6}$`).MatchString(claim.ClaimNo) {
		t.Fatalf("claimNo = %q, want CLM-2026-<6 digits>", claim.ClaimNo)
	}

	const now = "2026-11-05T03:00:00Z"
	if claim.Status != "SUBMITTED" || claim.Total != "1250.00" || claim.Version != 0 || claim.ReturnCount != 0 ||
		claim.FirstSubmittedAt != now || claim.SubmittedAt != now {
		t.Fatalf("claim = %+v, want SUBMITTED, total 1250.00, version 0, submitted at %s", claim, now)
	}

	if claim.Requester.Name != "Anan K." || claim.Requester.Position != "Staff, Section A" ||
		claim.AssignedReviewer.Name != "Malee S." || claim.AssignedReviewer.Position != "Section Head A" {
		t.Fatalf("requester %+v and reviewer %+v, want req.anan and rev.malee", claim.Requester, claim.AssignedReviewer)
	}

	if len(claim.Items) != 2 || claim.Items[0].LineNo != 1 || claim.Items[0].Amount != "250.00" ||
		claim.Items[0].ExpenseDate != "2026-11-03" || claim.Items[1].LineNo != 2 || claim.Items[1].Category != "ACCOMMODATION" {
		t.Fatalf("items = %+v, want the two items in request order", claim.Items)
	}

	if len(claim.History) != 1 || claim.History[0].Action != "SUBMIT" || claim.History[0].FromStatus != nil ||
		claim.History[0].ToStatus != "SUBMITTED" || claim.History[0].Actor.Name != "Anan K." {
		t.Fatalf("history = %+v, want one SUBMIT entry from null to SUBMITTED by req.anan", claim.History)
	}

	if claim.AllowedActions == nil || len(claim.AllowedActions) != 0 {
		t.Fatalf("allowedActions = %v, want an empty list", claim.AllowedActions)
	}

	for query, want := range map[string]int{
		"SELECT count(*) FROM claims":        1,
		"SELECT count(*) FROM line_items":    2,
		"SELECT count(*) FROM claim_history": 1,
	} {
		if got := rowCount(t, pool, query); got != want {
			t.Fatalf("%s = %d, want %d", query, got, want)
		}
	}
}

// F02-TEST-13 (F02-AC-08, BR-08): the server ignores a client total and stores the sum of the items.
func TestF02TEST13ClientTotalIsIgnored(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "req.anan")

	recorder := postClaim(t, router, token, twoItemsBody(`"total": "1.00",`))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", recorder.Code, recorder.Body)
	}

	var body claimResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if body.Data.Total != "1250.00" {
		t.Fatalf("total = %q, want 1250.00", body.Data.Total)
	}

	var stored int
	if err := pool.QueryRow(t.Context(), `SELECT total_satang FROM claims`).Scan(&stored); err != nil {
		t.Fatalf("read total_satang: %v", err)
	}

	if stored != 125000 {
		t.Fatalf("total_satang = %d, want 125000", stored)
	}
}

// F02-TEST-14 (F02-AC-09, BR-15): a Finance Officer has no assigned reviewer, so POST /claims gets 403 and stores nothing.
func TestF02TEST14FinanceOfficerCannotCreateClaim(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "fin.wanida")

	decodeProblem(t, postClaim(t, router, token, twoItemsBody("")), http.StatusForbidden, "FORBIDDEN")

	if got := rowCount(t, pool, "SELECT count(*) FROM claims"); got != 0 {
		t.Fatalf("claims = %d, want 0", got)
	}
}
