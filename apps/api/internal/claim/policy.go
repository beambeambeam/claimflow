package claim

import (
	"github.com/google/uuid"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/money"
)

// Action is a step that the claim policy allows and Act takes. Its value is the one in a claim's allowedActions and
// the last segment of the operation path.
type Action string

// The actions of the claim policy.
const (
	ActionApprove  Action = "approve"
	ActionReject   Action = "reject"
	ActionReturn   Action = "return"
	ActionEdit     Action = "edit"
	ActionDelete   Action = "delete"
	ActionSubmit   Action = "submit"
	ActionWithdraw Action = "withdraw"
	// ActionFinalApprove and ActionFinalReject are the final decision of a Finance Officer on a PENDING_FINANCE claim (F-08).
	ActionFinalApprove Action = "final-approve"
	ActionFinalReject  Action = "final-reject"
)

// MaxReturns is how many times a claim can be returned to its requester (BR-14).
const MaxReturns = 3

// Subject is what the claim policy reads about a claim. A claim with no assigned reviewer has the nil UUID there.
type Subject struct {
	RequesterID        uuid.UUID
	AssignedReviewerID uuid.UUID
	Status             db.ClaimStatus
	OwnerHasReviewer   bool
	ReturnCount        int
	TotalSatang        money.Satang
}

// allActions lists every action in the order that Allowed returns them.
var allActions = []Action{
	ActionApprove, ActionReject, ActionReturn, ActionEdit, ActionDelete, ActionSubmit, ActionWithdraw, ActionFinalApprove, ActionFinalReject,
}

// isVisible reports whether actor may see the claim: it is own, or it is not a DRAFT and either assigned or the actor
// is a Finance Officer (SDS §5.2).
func isVisible(actor Actor, claim Subject) bool {
	if claim.RequesterID == actor.ID {
		return true
	}

	return claim.Status != db.ClaimStatusDRAFT && (claim.AssignedReviewerID == actor.ID || actor.Role == RoleFinance)
}

// check applies visibility, the actor relation, and the status rules in that order (F-04 §1.4, F-05 §1.4).
func check(actor Actor, claim Subject, action Action) error {
	if !isVisible(actor, claim) {
		return ErrForbidden
	}

	switch action {
	case ActionEdit, ActionDelete:
		if claim.RequesterID != actor.ID {
			return ErrForbidden
		}
		if claim.Status == db.ClaimStatusDRAFT || (action == ActionEdit && claim.Status == db.ClaimStatusRETURNED) {
			return nil
		}
		return ErrInvalidTransition
	case ActionSubmit:
		if claim.RequesterID != actor.ID || (claim.Status == db.ClaimStatusDRAFT && !claim.OwnerHasReviewer) {
			return ErrForbidden
		}
	case ActionWithdraw:
		if claim.RequesterID != actor.ID {
			return ErrForbidden
		}
	case ActionApprove, ActionReject, ActionReturn:
		if claim.RequesterID == actor.ID || claim.AssignedReviewerID != actor.ID {
			return ErrForbidden
		}
	case ActionFinalApprove, ActionFinalReject:
		if actor.Role != RoleFinance || claim.RequesterID == actor.ID {
			return ErrForbidden
		}
	}

	if _, ok := transitions[action][claim.Status]; ok {
		if action == ActionReturn && claim.ReturnCount >= MaxReturns {
			return ErrReturnLimit
		}

		return nil
	}
	return ErrInvalidTransition
}

// Allowed is the claim policy (F-03 §1.3, F-04 §1.3, F-08 §1.8): whether actor may see the claim, and which actions
// actor may take on it. "Own" means the actor is the requester; "assigned" means the actor is the assigned reviewer. A
// claim is visible when it is own, or when it is not a DRAFT and assigned or the actor is a Finance Officer. The
// assigned reviewer, when not also the requester, may approve and reject a SUBMITTED claim. A Finance Officer who is
// not the requester may make the final decision on a PENDING_FINANCE claim; seeing a claim grants no other action. It
// is a pure function of its inputs. Actions is never nil.
//
// The requester may edit or submit a draft or returned claim and may delete a draft (F-05 §1.9).
func Allowed(actor Actor, claim Subject) (visible bool, allowed []Action) {
	allowed = []Action{}

	for _, action := range allActions {
		if check(actor, claim, action) == nil {
			allowed = append(allowed, action)
		}
	}

	return isVisible(actor, claim), allowed
}
