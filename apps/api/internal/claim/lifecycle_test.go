package claim

import (
	"testing"
	"time"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/money"
)

// TestNextStatus: NextStatus gives the status and history action of the lifecycle table, the approval target of BR-09,
// and refuses a step the table does not allow.
func TestNextStatus(t *testing.T) {
	cases := []struct {
		name    string
		action  Action
		from    db.ClaimStatus
		total   money.Satang
		role    string
		wantTo  db.ClaimStatus
		wantAct db.ClaimAction
		wantOK  bool
	}{
		{"submit a draft", ActionSubmit, db.ClaimStatusDRAFT, 100, "requester", db.ClaimStatusSUBMITTED, db.ClaimActionSUBMIT, true},
		{"resubmit a returned claim", ActionSubmit, db.ClaimStatusRETURNED, 100, "requester", db.ClaimStatusSUBMITTED, db.ClaimActionRESUBMIT, true},
		{"approve up to 10,000.00", ActionApprove, db.ClaimStatusSUBMITTED, HighValueThresholdSatang, "reviewer", db.ClaimStatusAPPROVED, db.ClaimActionAPPROVE, true},
		{"approve over 10,000.00", ActionApprove, db.ClaimStatusSUBMITTED, HighValueThresholdSatang + 1, "reviewer", db.ClaimStatusPENDINGFINANCE, db.ClaimActionAPPROVE, true},
		{"finance approves over 10,000.00", ActionApprove, db.ClaimStatusSUBMITTED, HighValueThresholdSatang + 1, RoleFinance, db.ClaimStatusAPPROVED, db.ClaimActionAPPROVE, true},
		{"final approve", ActionFinalApprove, db.ClaimStatusPENDINGFINANCE, HighValueThresholdSatang + 1, RoleFinance, db.ClaimStatusAPPROVED, db.ClaimActionFINALAPPROVE, true},
		{"approve a draft", ActionApprove, db.ClaimStatusDRAFT, 100, "reviewer", "", "", false},
		{"final approve a submitted claim", ActionFinalApprove, db.ClaimStatusSUBMITTED, 100, RoleFinance, "", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			to, act, ok := NextStatus(tc.action, tc.from, tc.total, tc.role)
			if to != tc.wantTo || act != tc.wantAct || ok != tc.wantOK {
				t.Fatalf("NextStatus = %q, %q, %v; want %q, %q, %v", to, act, ok, tc.wantTo, tc.wantAct, tc.wantOK)
			}
		})
	}
}

// TestCheckLineItems: CheckLineItems runs the rules of a claim's line items, including the expense date window when the
// first submission time is given.
func TestCheckLineItems(t *testing.T) {
	now := time.Date(2026, time.November, 20, 10, 0, 0, 0, bangkok)
	first := now.AddDate(0, 0, -10)
	good := []LineItemInput{{Category: db.ExpenseCategoryTRAVEL, Amount: 50_000, ExpenseDate: first.AddDate(0, 0, -1)}}

	if err := CheckLineItems(good, now, first); err != nil {
		t.Fatalf("valid items: %v", err)
	}

	late := []LineItemInput{{Category: db.ExpenseCategoryTRAVEL, Amount: 50_000, ExpenseDate: first.AddDate(0, 0, -91)}}
	if err := CheckLineItems(late, now, first); err == nil {
		t.Fatal("an expense 91 days before the first submission: err = nil, want BR-07")
	}

	if err := CheckLineItems(nil, now, first); err == nil {
		t.Fatal("no items: err = nil, want BR-05")
	}
}
