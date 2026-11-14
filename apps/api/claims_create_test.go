package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
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

// F02-AC-09 (BR-15), second half: a requester-role member whose reviewer_id is NULL passes the route permission and is
// refused by the claim module, so the handler maps ErrForbidden to 403 and stores nothing.
func TestCreateClaimWithoutAssignedReviewerIsForbidden(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "req.anan")

	if _, err := pool.Exec(t.Context(), `UPDATE "user" SET reviewer_id = NULL WHERE username = 'req.anan'`); err != nil {
		t.Fatalf("clear reviewer: %v", err)
	}

	decodeProblem(t, postClaim(t, router, token, twoItemsBody("")), http.StatusForbidden, "FORBIDDEN")

	if got := rowCount(t, pool, "SELECT count(*) FROM claims"); got != 0 {
		t.Fatalf("claims = %d, want 0", got)
	}
}

// failingBegin is a pool whose transactions cannot start, as in a database outage that begins after the session check.
type failingBegin struct {
	*pgxpool.Pool
}

func (failingBegin) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("begin refused")
}

// An unexpected claim module failure is a 500 INTERNAL problem with no internal text.
func TestCreateClaimFailureIsInternalProblem(t *testing.T) {
	pool := pgtest.New(t)
	seededClaimRouter(t, pool)
	router := newRouterAt(discardLogger(), testEnv(0), failingBegin{pool}, func() time.Time { return claimsNow })
	token := loginAs(t, router, "req.anan")

	recorder := postClaim(t, router, token, twoItemsBody(""))
	decodeProblem(t, recorder, http.StatusInternalServerError, "INTERNAL")

	if strings.Contains(recorder.Body.String(), "begin refused") {
		t.Fatalf("body leaks the error text: %s", recorder.Body)
	}
}

// fieldError is one errors[] entry of a problem response.
type fieldError struct {
	Field   string `json:"field"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// problemErrors returns the errors[] of a problem response body.
func problemErrors(t *testing.T, recorder *httptest.ResponseRecorder) []fieldError {
	t.Helper()

	var body struct {
		Errors []fieldError `json:"errors"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode problem: %v", err)
	}

	return body.Errors
}

// claimBodyWith returns a request with the given items JSON and a valid title and purpose.
func claimBodyWith(items string) string {
	return `{"title": "Bangkok workshop travel", "purpose": "Train fares and hotel", "items": [` + items + `]}`
}

// itemJSON returns a line item JSON with a valid description and receipt reference.
func itemJSON(category, date, amount string) string {
	return `{"category": "` + category + `", "description": "Expense", "expenseDate": "` + date + `", "amount": "` + amount + `", "receiptRef": "REF-1"}`
}

// F02-TEST-12 (F02-AC-07, BR-07): a future expense date is a 422 on that field with the contract message, and nothing is stored.
func TestF02TEST12FutureExpenseDateIs422(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "req.anan")

	recorder := postClaim(t, router, token, claimBodyWith(itemJSON("TRAVEL", "2026-11-06", "250.00")))
	decodeProblem(t, recorder, http.StatusUnprocessableEntity, "VALIDATION_FAILED")

	want := []fieldError{{Field: "items[0].expenseDate", Rule: "BR-07", Message: "Expense date cannot be in the future."}}
	if got := problemErrors(t, recorder); !slices.Equal(got, want) {
		t.Fatalf("errors = %+v, want %+v", got, want)
	}

	if got := rowCount(t, pool, "SELECT count(*) FROM claims"); got != 0 {
		t.Fatalf("claims = %d, want 0", got)
	}
}

// F02-TEST-17 (F02-AC-11, NFR-08): two broken rules on two fields come back in one 422 with two entries.
func TestF02TEST17EveryViolationInOneResponse(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "req.anan")

	items := itemJSON("TRAVEL", "2026-11-06", "250.00") + "," + itemJSON("MEALS", "2026-11-05", "1600.00")
	recorder := postClaim(t, router, token, claimBodyWith(items))
	decodeProblem(t, recorder, http.StatusUnprocessableEntity, "VALIDATION_FAILED")

	want := []fieldError{
		{Field: "items[0].expenseDate", Rule: "BR-07", Message: "Expense date cannot be in the future."},
		{Field: "items[1].amount", Rule: "BR-12", Message: "A Meals item cannot be more than 1,500.00 THB."},
	}
	if got := problemErrors(t, recorder); !slices.Equal(got, want) {
		t.Fatalf("errors = %+v, want %+v", got, want)
	}
}

// badRequestErrors posts body as req.anan, expects a 400 BAD_REQUEST problem, and returns its errors[]. Nothing may be stored.
func badRequestErrors(t *testing.T, body string) []fieldError {
	t.Helper()

	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "req.anan")

	recorder := postClaim(t, router, token, body)
	decodeProblem(t, recorder, http.StatusBadRequest, "BAD_REQUEST")

	if got := rowCount(t, pool, "SELECT count(*) FROM claims"); got != 0 {
		t.Fatalf("claims = %d, want 0", got)
	}

	return problemErrors(t, recorder)
}

// F02-TEST-07 (F02-AC-04, BR-05): an amount with 3 decimals is a 400 naming items[0].amount.
func TestF02TEST07ThreeDecimalAmountIs400(t *testing.T) {
	got := badRequestErrors(t, claimBodyWith(itemJSON("TRAVEL", "2026-11-03", "1.005")))

	want := []fieldError{{Field: "items[0].amount", Rule: "pattern", Message: "Enter an amount with 2 decimals, e.g. 1250.00."}}
	if !slices.Equal(got, want) {
		t.Fatalf("errors = %+v, want %+v", got, want)
	}
}

// F02-TEST-10 (F02-AC-06, BR-12): the category FOOD is a 400 naming items[0].category.
func TestF02TEST10UnknownCategoryIs400(t *testing.T) {
	got := badRequestErrors(t, claimBodyWith(itemJSON("FOOD", "2026-11-03", "250.00")))

	want := []fieldError{{Field: "items[0].category", Rule: "enum", Message: "Choose a category."}}
	if !slices.Equal(got, want) {
		t.Fatalf("errors = %+v, want %+v", got, want)
	}
}

// F02-TEST-18 (F02-AC-12): each shape violation is a 400 whose errors[] names the field, with the message of §1.4.
func TestF02TEST18ShapeViolationsAre400(t *testing.T) {
	const validItem = `{"category": "TRAVEL", "description": "Expense", "expenseDate": "2026-11-03", "amount": "250.00", "receiptRef": "REF-1"}`

	body := func(title, purpose, item string) string {
		return `{"title": ` + title + `, "purpose": ` + purpose + `, "items": [` + item + `]}`
	}
	quoted := func(n int) string { return `"` + strings.Repeat("x", n) + `"` }
	itemWith := func(member, value string) string {
		return `{"category": "TRAVEL", "description": "Expense", "expenseDate": "2026-11-03", "amount": "250.00", "receiptRef": "REF-1", "` + member + `": ` + value + `}`
	}

	cases := []struct {
		name string
		body string
		want fieldError
	}{
		{"missing title", `{"purpose": "Purpose", "items": [` + validItem + `]}`, fieldError{"title", "required", "Enter a title (up to 120 characters)."}},
		{"title over 120", body(quoted(121), `"Purpose"`, validItem), fieldError{"title", "maxLength", "Enter a title (up to 120 characters)."}},
		{"purpose over 1000", body(`"Title"`, quoted(1001), validItem), fieldError{"purpose", "maxLength", "Enter a purpose (up to 1,000 characters)."}},
		{"description over 200", body(`"Title"`, `"Purpose"`, itemWith("description", quoted(201))), fieldError{"items[0].description", "maxLength", "Enter a description (up to 200 characters)."}},
		{"receiptRef over 50", body(`"Title"`, `"Purpose"`, itemWith("receiptRef", quoted(51))), fieldError{"items[0].receiptRef", "maxLength", "Enter the receipt reference (up to 50 characters)."}},
		{"impossible date", body(`"Title"`, `"Purpose"`, itemWith("expenseDate", `"2026-02-31"`)), fieldError{"items[0].expenseDate", "format", "Enter a date as YYYY-MM-DD."}},
		{"bad date format", body(`"Title"`, `"Purpose"`, itemWith("expenseDate", `"05/11/2026"`)), fieldError{"items[0].expenseDate", "format", "Enter a date as YYYY-MM-DD."}},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := badRequestErrors(t, test.body); !slices.Equal(got, []fieldError{test.want}) {
				t.Fatalf("errors = %+v, want %+v", got, []fieldError{test.want})
			}
		})
	}
}
