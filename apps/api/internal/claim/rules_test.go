package claim

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/money"
	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// want is the field and rule of one expected violation. Tests compare messages where the contract fixes them.
type want struct{ field, rule string }

// item returns a valid line item that a case then changes.
func item(category string, amount money.Satang) LineItemInput {
	return LineItemInput{
		Category:    db.ExpenseCategory(category),
		Description: "Expense",
		ExpenseDate: time.Date(2026, time.November, 3, 0, 0, 0, 0, time.UTC),
		Amount:      amount,
		ReceiptRef:  "REF-1",
	}
}

// itemsOf returns n valid items of 1.00 THB.
func itemsOf(n int) []LineItemInput {
	items := make([]LineItemInput, n)
	for i := range items {
		items[i] = item("TRAVEL", 100)
	}

	return items
}

// violations returns the field and rule of every entry of a ValidationError, or nil when err is nil.
func violations(t *testing.T, err error) []want {
	t.Helper()

	if err == nil {
		return nil
	}

	var invalid ValidationError
	if !errors.As(err, &invalid) {
		t.Fatalf("error = %v, want a ValidationError", err)
	}

	got := make([]want, 0, len(invalid.Fields))
	for _, field := range invalid.Fields {
		got = append(got, want{field.Field, field.Rule})
	}

	return got
}

func sameViolations(got, wanted []want) bool {
	return slices.Equal(got, wanted)
}

// ruleCase is one row of the module tables: the items to create and the violations expected (none for a stored claim).
type ruleCase struct {
	name  string
	items []LineItemInput
	want  []want
}

// runRuleCases creates a claim for each case through claim.Service against PostgreSQL.
func runRuleCases(t *testing.T, cases []ruleCase) {
	t.Helper()

	pool := pgtest.New(t)
	service, actor := ruleService(t, pool)

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			input := Input{Title: "Title", Purpose: "Purpose", Items: test.items}

			_, err := service.Create(t.Context(), actor, input, true)
			if got := violations(t, err); !sameViolations(got, test.want) {
				t.Fatalf("violations = %v, want %v", got, test.want)
			}
		})
	}
}

func ruleService(t *testing.T, pool *pgxpool.Pool) (Service, Actor) {
	t.Helper()

	malee := member(t, pool, "rev.malee", "reviewer", uuid.Nil)
	anan := member(t, pool, "req.anan", "requester", malee)

	return newService(pool), Actor{ID: anan}
}

// F02-TEST-04 (F02-AC-03, BR-05): 1 to 20 line items.
func TestF02TEST04ItemCount(t *testing.T) {
	count := []want{{"items", "BR-05"}}

	runRuleCases(t, []ruleCase{
		{"0 items", itemsOf(0), count},
		{"1 item", itemsOf(1), nil},
		{"20 items", itemsOf(20), nil},
		{"21 items", itemsOf(21), count},
	})
}

// F02-TEST-06 (F02-AC-04, BR-05): each amount from 0.01 to 20,000.00.
func TestF02TEST06AmountRange(t *testing.T) {
	amount := []want{{"items[0].amount", "BR-05"}}

	runRuleCases(t, []ruleCase{
		{"0.00", []LineItemInput{item("TRAVEL", 0)}, amount},
		{"0.01", []LineItemInput{item("TRAVEL", 1)}, nil},
		{"20000.00", []LineItemInput{item("TRAVEL", 2000000)}, nil},
		{"20000.01", []LineItemInput{item("TRAVEL", 2000001)}, amount},
	})
}

// F02-TEST-08 (F02-AC-05, BR-06): a total of at most 50,000.00.
func TestF02TEST08Total(t *testing.T) {
	runRuleCases(t, []ruleCase{
		{"50000.00", []LineItemInput{item("TRAVEL", 2000000), item("TRAVEL", 2000000), item("TRAVEL", 1000000)}, nil},
		{"50000.01", []LineItemInput{item("TRAVEL", 2000000), item("TRAVEL", 2000000), item("TRAVEL", 1000001)}, []want{{"total", "BR-06"}}},
	})
}

// F02-TEST-09 (F02-AC-06, BR-12): a Meals item of at most 1,500.00.
func TestF02TEST09MealsLimit(t *testing.T) {
	runRuleCases(t, []ruleCase{
		{"MEALS 1500.00", []LineItemInput{item("MEALS", 150000)}, nil},
		{"MEALS 1500.01", []LineItemInput{item("MEALS", 150001)}, []want{{"items[0].amount", "BR-12"}}},
		{"TRAVEL 1500.01", []LineItemInput{item("TRAVEL", 150001)}, nil},
	})
}

// F02-TEST-11 (F02-AC-07, BR-07): an expense date from 90 days before today up to today, in Bangkok dates.
func TestF02TEST11ExpenseDateWindow(t *testing.T) {
	bangkokClock := time.Date(2026, time.November, 5, 10, 0, 0, 0, time.FixedZone("ICT", 7*60*60))
	midnightClock := time.Date(2026, time.November, 5, 17, 30, 0, 0, time.UTC) // 6 Nov 00:30 in Bangkok

	tests := []struct {
		name  string
		now   time.Time
		date  string
		field string
		want  string
	}{
		{"tomorrow", bangkokClock, "2026-11-06", "items[0].expenseDate", "Expense date cannot be in the future."},
		{"today", bangkokClock, "2026-11-05", "", ""},
		{"90 days ago", bangkokClock, "2026-08-07", "", ""},
		{"91 days ago", bangkokClock, "2026-08-06", "items[0].expenseDate", "Expense date cannot be more than 90 days before the claim was first submitted."},
		{"just after Bangkok midnight", midnightClock, "2026-11-06", "", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			date, err := time.Parse(time.DateOnly, test.date)
			if err != nil {
				t.Fatalf("parse %s: %v", test.date, err)
			}

			line := item("TRAVEL", 100)
			line.ExpenseDate = date

			err = checkRules(Input{Items: []LineItemInput{line}}, test.now)
			if test.field == "" {
				if err != nil {
					t.Fatalf("checkRules = %v, want nil", err)
				}

				return
			}

			var invalid ValidationError
			if !errors.As(err, &invalid) || len(invalid.Fields) != 1 {
				t.Fatalf("checkRules = %v, want one ValidationError entry", err)
			}

			got := invalid.Fields[0]
			if got.Field != test.field || got.Rule != "BR-07" || got.Message != test.want {
				t.Fatalf("entry = %+v, want %s BR-07 %q", got, test.field, test.want)
			}
		})
	}
}

// F02-AC-11 (NFR-08): every broken rule is reported at once, one entry per field, in field order.
func TestCheckRulesReportsEveryViolation(t *testing.T) {
	future := item("TRAVEL", 100)
	future.ExpenseDate = time.Date(2026, time.November, 6, 0, 0, 0, 0, time.UTC)

	err := checkRules(Input{Items: []LineItemInput{future, item("MEALS", 160000)}}, fixedNow)
	if got, wanted := violations(t, err), []want{{"items[0].expenseDate", "BR-07"}, {"items[1].amount", "BR-12"}}; !sameViolations(got, wanted) {
		t.Fatalf("violations = %v, want %v", got, wanted)
	}

	// An amount over the BR-05 range is one entry for that field even when BR-12 would also apply, and the total
	// is reported with it.
	err = checkRules(Input{Items: []LineItemInput{item("MEALS", 5000001)}}, fixedNow)
	if got, wanted := violations(t, err), []want{{"items[0].amount", "BR-05"}, {"total", "BR-06"}}; !sameViolations(got, wanted) {
		t.Fatalf("violations = %v, want %v", got, wanted)
	}

	err = checkRules(Input{}, fixedNow)
	if got, wanted := violations(t, err), []want{{"items", "BR-05"}}; !sameViolations(got, wanted) {
		t.Fatalf("violations = %v, want %v", got, wanted)
	}
}
