package claim

import (
	"github.com/google/uuid"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
)

// Subject is what the claim policy reads about a claim. A claim with no assigned reviewer has the nil UUID there.
type Subject struct {
	RequesterID        uuid.UUID
	AssignedReviewerID uuid.UUID
	Status             db.ClaimStatus
}

// Allowed is the claim policy (F-03 §1.3): whether actor may see the claim, and which actions actor may take on it.
// "Own" means the actor is the requester; "assigned" means the actor is the assigned reviewer. A claim is visible when
// it is own, or when it is assigned and not a DRAFT. It is a pure function of its inputs. Actions is never nil.
//
// No actor has an action in Sprint 1: the owner's edit, submit, delete, and withdraw come in Sprint 2, and F-04 adds
// approve and reject (assigned, not own, SUBMITTED). Later features add rows here and do not change the visibility.
func Allowed(actor uuid.UUID, claim Subject) (visible bool, actions []string) {
	own := claim.RequesterID == actor
	assigned := claim.AssignedReviewerID == actor

	return own || (assigned && claim.Status != db.ClaimStatusDRAFT), []string{}
}
