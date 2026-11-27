package main

import (
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// claimNumbers returns the claim numbers of a page, in order.
func claimNumbers(page *claimPageResponse) []string {
	numbers := make([]string, 0, len(page.Data))
	for i := range page.Data {
		numbers = append(numbers, page.Data[i].ClaimNo)
	}

	return numbers
}

// F09-TEST-01 (F09-AC-01): scope=all lists the visible set of each seed user, newest first, with a matching total.
// Without a session it is 401.
func TestF09TEST01ListAllShowsTheVisibleSet(t *testing.T) {
	pool := pgtest.New(t)
	router := fullySeededRouter(t, pool)

	want := map[string][]string{
		// Her own claims, the draft included.
		"req.anan": {"CLM-2026-900007", "CLM-2026-900001", "CLM-2026-900008", "CLM-2026-900002"},
		// Her own claim and the claims of req.anan and req.ploy that are not drafts.
		"rev.malee": {"CLM-2026-900001", "CLM-2026-900009", "CLM-2026-900005", "CLM-2026-900008", "CLM-2026-900003", "CLM-2026-900002"},
		// Every claim that is not a draft.
		"fin.wanida": {
			"CLM-2026-900001", "CLM-2026-900004", "CLM-2026-900009", "CLM-2026-900005", "CLM-2026-900008",
			"CLM-2026-900006", "CLM-2026-900003", "CLM-2026-900010", "CLM-2026-900002",
		},
	}
	for username, numbers := range want {
		page := listClaims(t, router, loginAs(t, router, username), "?scope=all")
		if got := claimNumbers(&page); !slices.Equal(got, numbers) || page.Total != len(numbers) {
			t.Fatalf("%s: claims = %v (total %d), want %v", username, got, page.Total, numbers)
		}
	}

	decodeProblem(t, get(t, router, "/api/v1/claims?scope=all"), http.StatusUnauthorized, "UNAUTHENTICATED")
}

// F09-TEST-06 (F09-AC-06): bad filters are a 400 with one errors entry per bad parameter and the F-09 §1.3 message.
func TestF09TEST06InvalidFiltersAre400(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "req.anan")

	dateMessage := "Use a date in the form YYYY-MM-DD."
	cases := []struct {
		query string
		want  []fieldError
	}{
		{"?scope=all&status=PAID", []fieldError{{"status", "enum", "Unknown status."}}},
		{"?scope=all&category=FOOD", []fieldError{{"category", "enum", "Unknown category."}}},
		{"?scope=all&from=05/11/2026", []fieldError{{"from", "format", dateMessage}}},
		{"?scope=all&from=2026-11-10&to=2026-11-01", []fieldError{{"to", "range", "The end date must be on or after the start date."}}},
		{"?scope=all&q=" + strings.Repeat("a", 121), []fieldError{{"q", "maxLength", "Search text must be at most 120 characters."}}},
		{"?scope=all&status=PAID&category=FOOD", []fieldError{{"status", "enum", "Unknown status."}, {"category", "enum", "Unknown category."}}},
		{"?scope=mine&status=APPROVED", []fieldError{{"status", "scope", "Filters are only available for scope=all."}}},
	}
	for _, tc := range cases {
		recorder := getWithSession(t, router, "/api/v1/claims"+tc.query, token)
		decodeProblem(t, recorder, http.StatusBadRequest, "BAD_REQUEST")
		if got := problemErrors(t, recorder); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("GET /claims%s errors = %+v, want %+v", tc.query, got, tc.want)
		}
	}
}

// F09-TEST-09 (F09-AC-08, NFR-07): the migration adds both filter indexes, and q is a SQL parameter, not SQL text.
func TestF09TEST09FilterIndexesAndParameterisedSearch(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "req.anan")

	for _, index := range []string{"line_items_category_claim_id_idx", "claims_status_created_at_idx"} {
		var definition string
		if err := pool.QueryRow(t.Context(), `SELECT indexdef FROM pg_indexes WHERE indexname = $1`, index).Scan(&definition); err != nil {
			t.Fatalf("index %s: %v", index, err)
		}
	}

	at := time.Date(2026, time.November, 1, 3, 0, 0, 0, time.UTC)
	taxi := storedClaim(t, pool, "req.anan", "SUBMITTED", at)
	injection := storedClaim(t, pool, "req.anan", "SUBMITTED", at.Add(time.Hour))
	if _, err := pool.Exec(t.Context(), `UPDATE claims SET title = CASE WHEN id = $1 THEN 'Taxi' ELSE 'Refund '' OR 1=1 --' END
		WHERE id IN ($1, $2)`, taxi, injection); err != nil {
		t.Fatalf("set titles: %v", err)
	}

	page := listClaims(t, router, token, "?scope=all&q="+url.QueryEscape("' OR 1=1 --"))
	if page.Total != 1 || len(page.Data) != 1 || page.Data[0].ID != injection.String() {
		t.Fatalf("page = %+v, want only claim %s", page, injection)
	}
}
