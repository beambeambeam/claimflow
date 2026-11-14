package claim

import (
	"github.com/google/uuid"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
)

// The actions of the claim policy. They are the values of a claim's allowedActions and the last segment of the
// operation path.
const (
	ActionApprove = "approve"
	ActionReject  = "reject"
)

// Subject is what the claim policy reads about a claim. A claim with no assigned reviewer has the nil UUID there.
type Subject struct {
	RequesterID        uuid.UUID
	AssignedReviewerID uuid.UUID
	Status             db.ClaimStatus
}

// transition is what an action does to a claim: the status it needs, the status it leads to, and the history action.
type transition struct {
	from, to db.ClaimStatus
	history  db.ClaimAction
}

// transitions is the lifecycle of Sprint 1 (F-04 §1.3). APPROVED and REJECTED are terminal. A later feature adds rows.
var transitions = map[string]transition{
	ActionApprove: {from: db.ClaimStatusSUBMITTED, to: db.ClaimStatusAPPROVED, history: db.ClaimActionAPPROVE},
	ActionReject:  {from: db.ClaimStatusSUBMITTED, to: db.ClaimStatusREJECTED, history: db.ClaimActionREJECT},
}

// allActions lists every action in the order that Allowed returns them.
var allActions = []string{ActionApprove, ActionReject}

// isVisible reports whether actor may see the claim: it is own, or it is assigned and not a DRAFT.
func isVisible(actor uuid.UUID, claim Subject) bool {
	return claim.RequesterID == actor || (claim.AssignedReviewerID == actor && claim.Status != db.ClaimStatusDRAFT)
}

// check decides whether actor may take action on the claim, in the order of F-04 §1.4: a claim that is not visible, a
// requester (BR-01), and a member who is not the assigned reviewer (BR-02) are ErrForbidden whatever the status; a
// claim whose status does not allow the action is ErrInvalidTransition (BR-03). It returns what the action does.
func check(actor uuid.UUID, claim Subject, action string) (transition, error) {
	if !isVisible(actor, claim) || claim.RequesterID == actor || claim.AssignedReviewerID != actor {
		return transition{}, ErrForbidden
	}

	found := transitions[action]
	if claim.Status != found.from {
		return transition{}, ErrInvalidTransition
	}

	return found, nil
}

// Allowed is the claim policy (F-03 §1.3, F-04 §1.3): whether actor may see the claim, and which actions actor may take
// on it. "Own" means the actor is the requester; "assigned" means the actor is the assigned reviewer. A claim is
// visible when it is own, or when it is assigned and not a DRAFT. The assigned reviewer, when not also the requester,
// may approve and reject a SUBMITTED claim. It is a pure function of its inputs. Actions is never nil.
//
// The owner's edit, submit, delete, and withdraw come in Sprint 2. Later features add rows here and do not change the
// visibility.
func Allowed(actor uuid.UUID, claim Subject) (visible bool, allowed []string) {
	allowed = []string{}

	for _, action := range allActions {
		if _, err := check(actor, claim, action); err == nil {
			allowed = append(allowed, action)
		}
	}

	return isVisible(actor, claim), allowed
}
