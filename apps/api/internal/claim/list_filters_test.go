package claim

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

func ptr[T any](value T) *T {
	return &value
}

// F09-TEST-07 (F09-AC-06): the handler filter check gives one field error per bad parameter with the F-09 §1.3
// message, and no errors for valid filters.
func TestF09TEST07FilterCheck(t *testing.T) {
	unknownStatus := api.FieldError{Field: "status", Rule: "enum", Message: "Unknown status."}
	unknownCategory := api.FieldError{Field: "category", Rule: "enum", Message: "Unknown category."}

	invalid := []struct {
		name   string
		params api.ListClaimsParams
		want   []api.FieldError
	}{
		{"unknown status", api.ListClaimsParams{Scope: api.ListScopeAll, Status: ptr(api.ClaimStatus("PAID"))}, []api.FieldError{unknownStatus}},
		{"unknown category", api.ListClaimsParams{Scope: api.ListScopeAll, Category: ptr(api.ExpenseCategory("FOOD"))}, []api.FieldError{unknownCategory}},
		{
			"from not a date",
			api.ListClaimsParams{Scope: api.ListScopeAll, From: ptr("05/11/2026")},
			[]api.FieldError{{Field: "from", Rule: "format", Message: "Use a date in the form YYYY-MM-DD."}},
		},
		{
			"to not a date",
			api.ListClaimsParams{Scope: api.ListScopeAll, To: ptr("2026-02-30")},
			[]api.FieldError{{Field: "to", Rule: "format", Message: "Use a date in the form YYYY-MM-DD."}},
		},
		{
			"from later than to",
			api.ListClaimsParams{Scope: api.ListScopeAll, From: ptr("2026-11-10"), To: ptr("2026-11-01")},
			[]api.FieldError{{Field: "to", Rule: "range", Message: "The end date must be on or after the start date."}},
		},
		{
			"q of 121 characters",
			api.ListClaimsParams{Scope: api.ListScopeAll, Q: ptr(strings.Repeat("ก", 121))},
			[]api.FieldError{{Field: "q", Rule: "maxLength", Message: "Search text must be at most 120 characters."}},
		},
		{
			"status and category",
			api.ListClaimsParams{Scope: api.ListScopeAll, Status: ptr(api.ClaimStatus("PAID")), Category: ptr(api.ExpenseCategory("FOOD"))},
			[]api.FieldError{unknownStatus, unknownCategory},
		},
		{
			"a filter with scope=mine",
			api.ListClaimsParams{Scope: api.ListScopeMine, Status: ptr(api.ClaimStatusApproved)},
			[]api.FieldError{{Field: "status", Rule: "scope", Message: "Filters are only available for scope=all."}},
		},
		{
			"every filter with scope=review",
			api.ListClaimsParams{
				Scope: api.ListScopeReview, Status: ptr(api.ClaimStatusApproved), Category: ptr(api.ExpenseCategoryMeals),
				From: ptr("2026-11-01"), To: ptr("2026-11-02"), Q: ptr(""),
			},
			[]api.FieldError{
				{Field: "status", Rule: "scope", Message: "Filters are only available for scope=all."},
				{Field: "category", Rule: "scope", Message: "Filters are only available for scope=all."},
				{Field: "from", Rule: "scope", Message: "Filters are only available for scope=all."},
				{Field: "to", Rule: "scope", Message: "Filters are only available for scope=all."},
				{Field: "q", Rule: "scope", Message: "Filters are only available for scope=all."},
			},
		},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			filters, got := listFilters(&tc.params)
			if !reflect.DeepEqual(got, tc.want) || !reflect.DeepEqual(filters, Filters{}) {
				t.Fatalf("listFilters = %+v, %+v; want no filters and %+v", filters, got, tc.want)
			}
		})
	}

	from, to := time.Date(2026, time.November, 5, 0, 0, 0, 0, time.UTC), time.Date(2026, time.November, 5, 0, 0, 0, 0, time.UTC)
	status, category := db.ClaimStatusAPPROVED, db.ExpenseCategoryMEALS
	valid := []struct {
		name   string
		params api.ListClaimsParams
		want   Filters
	}{
		{"no filters", api.ListClaimsParams{Scope: api.ListScopeAll}, Filters{}},
		{"no filters with scope=mine", api.ListClaimsParams{Scope: api.ListScopeMine}, Filters{}},
		{
			"every filter",
			api.ListClaimsParams{
				Scope: api.ListScopeAll, Status: ptr(api.ClaimStatusApproved), Category: ptr(api.ExpenseCategoryMeals),
				From: ptr("2026-11-05"), To: ptr("2026-11-05"), Q: ptr("  " + strings.Repeat("ก", 120) + " "),
			},
			Filters{Status: &status, Category: &category, From: &from, To: &to, Q: strings.Repeat("ก", 120)},
		},
	}
	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			filters, invalid := listFilters(&tc.params)
			if len(invalid) != 0 || !reflect.DeepEqual(filters, tc.want) {
				t.Fatalf("listFilters = %+v, %+v; want %+v and no errors", filters, invalid, tc.want)
			}
		})
	}
}

// The route requires a session before the handler runs, so a missing user is only a guard.
func TestListClaimsWithoutSessionUserIsUnauthenticated(t *testing.T) {
	handler := NewHandler(newService(pgtest.New(t)))

	_, err := handler.ListClaims(t.Context(), api.ListClaimsRequestObject{Params: api.ListClaimsParams{Scope: api.ListScopeAll}})
	if !errors.Is(err, errUnauthenticated) {
		t.Fatalf("ListClaims error = %v, want the 401 problem", err)
	}
}
