package claim

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
)

// F03-TEST-09 (F03-AC-09, NFR-02, F04-AC-07): the claim policy matrix. A claim is visible to its requester, and to its
// assigned reviewer once it is no longer a DRAFT. The assigned reviewer, when not also the requester, may approve and
// reject a SUBMITTED claim (F-04); nobody else has an action. The rows cover every role, relation, and status. The
// policy never reads the role, so a finance officer follows the same rules as everyone else. Later features extend
// this table.
func TestF03TEST09PolicyMatrix(t *testing.T) {
	actors := map[string]uuid.UUID{
		"requester": uuid.Must(uuid.NewV7()),
		"reviewer":  uuid.Must(uuid.NewV7()),
		"finance":   uuid.Must(uuid.NewV7()),
	}
	other := uuid.Must(uuid.NewV7())

	statuses := []db.ClaimStatus{db.ClaimStatusDRAFT, db.ClaimStatusSUBMITTED, db.ClaimStatusAPPROVED, db.ClaimStatusREJECTED}

	none := func(db.ClaimStatus) []string { return []string{} }

	// subject builds the claim as the actor relates to it. "own and assigned" is the corrupted assignment that points at the requester.
	relations := map[string]struct {
		subject func(actor uuid.UUID, status db.ClaimStatus) Subject
		visible func(status db.ClaimStatus) bool
		actions func(status db.ClaimStatus) []string
	}{
		"own": {
			subject: func(actor uuid.UUID, status db.ClaimStatus) Subject {
				return Subject{RequesterID: actor, AssignedReviewerID: other, Status: status}
			},
			visible: func(db.ClaimStatus) bool { return true },
			actions: none,
		},
		"assigned": {
			subject: func(actor uuid.UUID, status db.ClaimStatus) Subject {
				return Subject{RequesterID: other, AssignedReviewerID: actor, Status: status}
			},
			visible: func(status db.ClaimStatus) bool { return status != db.ClaimStatusDRAFT },
			actions: func(status db.ClaimStatus) []string {
				if status == db.ClaimStatusSUBMITTED {
					return []string{"approve", "reject"}
				}

				return []string{}
			},
		},
		"unrelated": {
			subject: func(_ uuid.UUID, status db.ClaimStatus) Subject {
				return Subject{RequesterID: other, AssignedReviewerID: uuid.Must(uuid.NewV7()), Status: status}
			},
			visible: func(db.ClaimStatus) bool { return false },
			actions: none,
		},
		"unassigned": {
			subject: func(_ uuid.UUID, status db.ClaimStatus) Subject {
				return Subject{RequesterID: other, Status: status}
			},
			visible: func(db.ClaimStatus) bool { return false },
			actions: none,
		},
		"own and assigned": {
			subject: func(actor uuid.UUID, status db.ClaimStatus) Subject {
				return Subject{RequesterID: actor, AssignedReviewerID: actor, Status: status}
			},
			visible: func(db.ClaimStatus) bool { return true },
			actions: none,
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

					// The owner has no action until Sprint 2. A claim assigned to its own requester gives no action (BR-01).
					if want := relation.actions(status); actions == nil || !slices.Equal(actions, want) {
						t.Fatalf("actions = %#v, want %#v", actions, want)
					}
				})
			}
		}
	}
}

// F04-TEST-09 (F04-AC-07, BR-01): a claim whose assigned reviewer is its requester gives that member no approve or
// reject, though the claim is SUBMITTED and the member is both the requester and the assigned reviewer. The database
// CHECK refuses to store such a claim (F04-TEST-10); the policy does not rely on it.
func TestF04TEST09PolicyGivesNoDecisionOnOwnClaim(t *testing.T) {
	malee := uuid.Must(uuid.NewV7())

	visible, actions := Allowed(malee, Subject{RequesterID: malee, AssignedReviewerID: malee, Status: db.ClaimStatusSUBMITTED})

	if !visible {
		t.Fatal("visible = false, want true: the requester sees their own claim")
	}

	for _, action := range []string{"approve", "reject"} {
		if slices.Contains(actions, action) {
			t.Fatalf("actions = %v, want no %s (BR-01)", actions, action)
		}
	}
}
