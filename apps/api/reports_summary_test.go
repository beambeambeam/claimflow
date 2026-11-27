package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// summaryResponse is the 200 body of GET /reports/summary (SDS §4.4).
type summaryResponse struct {
	Data struct {
		CountsByStatus map[string]int `json:"countsByStatus"`
		ApprovedTotals []struct {
			Month    string `json:"month"`
			Category string `json:"category"`
			Total    string `json:"total"`
		} `json:"approvedTotals"`
		AverageDecisionHours *float64 `json:"averageDecisionHours"`
	} `json:"data"`
}

// getSummary sends GET /reports/summary with query as token.
func getSummary(t *testing.T, router http.Handler, token, query string) *httptest.ResponseRecorder {
	t.Helper()

	return getWithSession(t, router, "/api/v1/reports/summary"+query, token)
}

// decodeSummary decodes a 200 summary, and fails on any other status.
func decodeSummary(t *testing.T, recorder *httptest.ResponseRecorder) summaryResponse {
	t.Helper()

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /reports/summary status = %d, want 200; body = %s", recorder.Code, recorder.Body)
	}

	var summary summaryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}

	return summary
}

// decidedClaim stores req.anan's claim of status, submitted at created and decided at decided,: its one
// TRAVEL item of 1,250.00 is what the summary adds up.
func decidedClaim(t *testing.T, pool *pgxpool.Pool, status string, created, decided time.Time) {
	t.Helper()

	id := storedClaim(t, pool, "req.anan", status, created)
	if _, err := pool.Exec(t.Context(), `UPDATE claims SET decided_at = $1 WHERE id = $2`, decided, id); err != nil {
		t.Fatalf("decide claim: %v", err)
	}
}

var (
	summarySubmitted = time.Date(2026, time.October, 14, 9, 0, 0, 0, time.FixedZone("ICT", 7*60*60))
	summaryDecided   = summarySubmitted.Add(24 * time.Hour)
)

// F10-TEST-06 (F10-AC-05, F10-AC-06): rev.malee and fin.wanida each get the Summary shape for their scope, with the
// approved total and the average of the one claim that is decided. A claim that nobody decided, and a range that holds
// no decision, give an empty list and null.
func TestF10TEST06SummaryOfReviewerAndFinance(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	storedClaim(t, pool, "req.anan", "SUBMITTED", summarySubmitted)
	decidedClaim(t, pool, "APPROVED", summarySubmitted, summaryDecided)

	for _, username := range []string{"rev.malee", "fin.wanida"} {
		token := loginAs(t, router, username)

		t.Run(username, func(t *testing.T) {
			got := decodeSummary(t, getSummary(t, router, token, ""))

			wantCounts := map[string]int{
				"SUBMITTED": 1, "RETURNED": 0, "PENDING_FINANCE": 0, "APPROVED": 1, "REJECTED": 0, "WITHDRAWN": 0,
			}
			if len(got.Data.CountsByStatus) != len(wantCounts) {
				t.Fatalf("countsByStatus = %v, want %v", got.Data.CountsByStatus, wantCounts)
			}

			for status, want := range wantCounts {
				if got.Data.CountsByStatus[status] != want {
					t.Fatalf("countsByStatus = %v, want %v", got.Data.CountsByStatus, wantCounts)
				}
			}

			if len(got.Data.ApprovedTotals) != 1 || got.Data.ApprovedTotals[0].Month != "2026-10" ||
				got.Data.ApprovedTotals[0].Category != "TRAVEL" || got.Data.ApprovedTotals[0].Total != "1250.00" {
				t.Fatalf("approvedTotals = %+v, want 1250.00 of TRAVEL in 2026-10", got.Data.ApprovedTotals)
			}

			if got.Data.AverageDecisionHours == nil || *got.Data.AverageDecisionHours != 24 {
				t.Fatalf("averageDecisionHours = %v, want 24", got.Data.AverageDecisionHours)
			}

			inside := decodeSummary(t, getSummary(t, router, token, "?from=2026-10-15&to=2026-10-15"))
			if len(inside.Data.ApprovedTotals) != 1 {
				t.Fatalf("approvedTotals on 15 Oct = %+v, want the claim decided that day", inside.Data.ApprovedTotals)
			}

			recorder := getSummary(t, router, token, "?from=2026-10-16")
			if body := recorder.Body.String(); !strings.Contains(body, `"approvedTotals":[]`) || !strings.Contains(body, `"averageDecisionHours":null`) {
				t.Fatalf("body = %s, want an empty approvedTotals and a null average for a range with no decision", body)
			}

			if got := decodeSummary(t, recorder); got.Data.CountsByStatus["APPROVED"] != 1 {
				t.Fatalf("countsByStatus = %v, want the counts to ignore the range", got.Data.CountsByStatus)
			}
		})
	}
}

// F10-TEST-06: req.anan gets 403 and a request with no session gets 401, whatever the query. A bad date or a range that
// runs backwards is 400 with one errors entry per bad parameter and the F-09 §1.3 messages.
func TestF10TEST06SummaryRefusals(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)

	t.Run("requester", func(t *testing.T) {
		recorder := getSummary(t, router, loginAs(t, router, "req.anan"), "")

		var body struct {
			Code string `json:"code"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || recorder.Code != http.StatusForbidden || body.Code != "FORBIDDEN" {
			t.Fatalf("status = %d, body = %s, want 403 FORBIDDEN", recorder.Code, recorder.Body)
		}
	})

	t.Run("no session", func(t *testing.T) {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/reports/summary", http.NoBody)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusUnauthorized || !strings.Contains(recorder.Body.String(), `"UNAUTHENTICATED"`) {
			t.Fatalf("status = %d, body = %s, want 401 UNAUTHENTICATED", recorder.Code, recorder.Body)
		}
	})

	token := loginAs(t, router, "rev.malee")

	for _, test := range []struct {
		query string
		want  []string
	}{
		{"?from=2026-11-10&to=2026-11-01", []string{"to: The end date must be on or after the start date."}},
		{"?from=Oct", []string{"from: Use a date in the form YYYY-MM-DD."}},
		{"?to=2026-02-30", []string{"to: Use a date in the form YYYY-MM-DD."}},
		{"?from=&to=1%20Nov", []string{"from: Use a date in the form YYYY-MM-DD.", "to: Use a date in the form YYYY-MM-DD."}},
	} {
		t.Run(test.query, func(t *testing.T) {
			recorder := getSummary(t, router, token, test.query)

			var body struct {
				Code   string `json:"code"`
				Errors []struct {
					Field   string `json:"field"`
					Message string `json:"message"`
				} `json:"errors"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || recorder.Code != http.StatusBadRequest || body.Code != "BAD_REQUEST" {
				t.Fatalf("status = %d, body = %s, want 400 BAD_REQUEST", recorder.Code, recorder.Body)
			}

			got := make([]string, 0, len(body.Errors))
			for _, fieldError := range body.Errors {
				got = append(got, fieldError.Field+": "+fieldError.Message)
			}

			if !slices.Equal(got, test.want) {
				t.Fatalf("errors = %q, want %q", got, test.want)
			}
		})
	}
}
