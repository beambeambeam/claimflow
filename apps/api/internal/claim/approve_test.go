package claim

import (
	"testing"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/money"
)

// F08-TEST-02 (F08-AC-02, BR-09, ADR-0001): an approval of a claim over 10,000.00 THB by a member who is not a Finance
// Officer goes to PENDING_FINANCE. A claim of exactly 10,000.00, or any approval by a Finance Officer, is APPROVED.
func TestF08TEST02ApproveTargetUsesTheHighValueThreshold(t *testing.T) {
	for _, test := range []struct {
		total money.Satang
		role  string
		want  db.ClaimStatus
	}{
		{1_000_000, "reviewer", db.ClaimStatusAPPROVED},
		{1_000_000, "finance", db.ClaimStatusAPPROVED},
		{1_000_001, "reviewer", db.ClaimStatusPENDINGFINANCE},
		{1_000_001, "finance", db.ClaimStatusAPPROVED},
	} {
		if got := approveTarget(test.total, test.role); got != test.want {
			t.Errorf("approveTarget(%d, %s) = %s, want %s", test.total, test.role, got, test.want)
		}
	}
}
