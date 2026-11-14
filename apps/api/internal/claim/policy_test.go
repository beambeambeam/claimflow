package claim

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
)

// F03-TEST-09 (F03-AC-09, NFR-02): the claim policy matrix. A claim is visible to its requester, and to its assigned
// reviewer once it is no longer a DRAFT. The rows cover every role, relation, and status. The policy never reads the
// role, so a finance officer follows the same rules as everyone else. Later features extend this table.
func TestF03TEST09PolicyMatrix(t *testing.T) {
	actors := map[string]uuid.UUID{
		"requester": uuid.Must(uuid.NewV7()),
		"reviewer":  uuid.Must(uuid.NewV7()),
		"finance":   uuid.Must(uuid.NewV7()),
	}
	other := uuid.Must(uuid.NewV7())

	statuses := []db.ClaimStatus{db.ClaimStatusDRAFT, db.ClaimStatusSUBMITTED, db.ClaimStatusAPPROVED, db.ClaimStatusREJECTED}

	// subject builds the claim as the actor relates to it. "own and assigned" is the corrupted assignment that points at the requester.
	relations := map[string]struct {
		subject func(actor uuid.UUID, status db.ClaimStatus) Subject
		visible func(status db.ClaimStatus) bool
	}{
		"own": {
			subject: func(actor uuid.UUID, status db.ClaimStatus) Subject {
				return Subject{RequesterID: actor, AssignedReviewerID: other, Status: status}
			},
			visible: func(db.ClaimStatus) bool { return true },
		},
		"assigned": {
			subject: func(actor uuid.UUID, status db.ClaimStatus) Subject {
				return Subject{RequesterID: other, AssignedReviewerID: actor, Status: status}
			},
			visible: func(status db.ClaimStatus) bool { return status != db.ClaimStatusDRAFT },
		},
		"unrelated": {
			subject: func(_ uuid.UUID, status db.ClaimStatus) Subject {
				return Subject{RequesterID: other, AssignedReviewerID: uuid.Must(uuid.NewV7()), Status: status}
			},
			visible: func(db.ClaimStatus) bool { return false },
		},
		"unassigned": {
			subject: func(_ uuid.UUID, status db.ClaimStatus) Subject {
				return Subject{RequesterID: other, Status: status}
			},
			visible: func(db.ClaimStatus) bool { return false },
		},
		"own and assigned": {
			subject: func(actor uuid.UUID, status db.ClaimStatus) Subject {
				return Subject{RequesterID: actor, AssignedReviewerID: actor, Status: status}
			},
			visible: func(db.ClaimStatus) bool { return true },
		},
	}

	for role, actor := range actors {
		for name, relation := range relations {
			for _, status := range statuses {
				t.Run(role+"/"+name+"/"+string(status), func(t *testing.T) {
					visible, actions := Allowed(actor, relation.subject(actor, status))

					if want := relation.visible(status); visible != want {
						t.Fatalf("visible = %t, want %t", visible, want)
					}

					// Sprint 1 grants no action to anyone: the owner has none until Sprint 2, approve and reject come in F-04.
					if actions == nil || !slices.Equal(actions, []string{}) {
						t.Fatalf("actions = %#v, want an empty, non-nil list", actions)
					}
				})
			}
		}
	}
}
