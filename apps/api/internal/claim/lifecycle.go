package claim

import (
	"time"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/money"
)

// transition is the destination and history action for an allowed status change.
type transition struct {
	to      db.ClaimStatus
	history db.ClaimAction
}

// transitions lists the status changes by action and source status (F-04 §1.3, F-05 §1.10, F-08 §1.3). An approval's
// destination here is the one without a second stage; Act picks the real one with approveTarget.
var transitions = map[Action]map[db.ClaimStatus]transition{
	ActionApprove: {db.ClaimStatusSUBMITTED: {to: db.ClaimStatusAPPROVED, history: db.ClaimActionAPPROVE}},
	ActionReject:  {db.ClaimStatusSUBMITTED: {to: db.ClaimStatusREJECTED, history: db.ClaimActionREJECT}},
	ActionReturn:  {db.ClaimStatusSUBMITTED: {to: db.ClaimStatusRETURNED, history: db.ClaimActionRETURN}},
	ActionFinalApprove: {
		db.ClaimStatusPENDINGFINANCE: {to: db.ClaimStatusAPPROVED, history: db.ClaimActionFINALAPPROVE},
	},
	ActionFinalReject: {
		db.ClaimStatusPENDINGFINANCE: {to: db.ClaimStatusREJECTED, history: db.ClaimActionFINALREJECT},
	},
	ActionWithdraw: {
		db.ClaimStatusSUBMITTED: {to: db.ClaimStatusWITHDRAWN, history: db.ClaimActionWITHDRAW},
		db.ClaimStatusRETURNED:  {to: db.ClaimStatusWITHDRAWN, history: db.ClaimActionWITHDRAW},
	},
	ActionSubmit: {
		db.ClaimStatusDRAFT:    {to: db.ClaimStatusSUBMITTED, history: db.ClaimActionSUBMIT},
		db.ClaimStatusRETURNED: {to: db.ClaimStatusSUBMITTED, history: db.ClaimActionRESUBMIT},
	},
}

// NextStatus returns the status and the history action that action leads to from the status from, for a claim with
// total and an actor with role, as Act applies it. ok is false when the lifecycle does not allow action from there.
// The load dataset builds and checks its histories with it (F-10 §1.8).
func NextStatus(action Action, from db.ClaimStatus, total money.Satang, role string) (to db.ClaimStatus, history db.ClaimAction, ok bool) {
	step, ok := transitions[action][from]
	if !ok {
		return "", "", false
	}

	if action == ActionApprove {
		step.to = approveTarget(total, role)
	}

	return step.to, step.history, true
}

// CheckLineItems runs the line item rules of a claim (BR-05, BR-06, BR-12) and, with firstSubmittedAt set, the expense
// date window of BR-07 as of now. It returns a ValidationError, or nil.
func CheckLineItems(items []LineItemInput, now, firstSubmittedAt time.Time) error {
	return checkRules(Input{Items: items}, now, firstSubmittedAt)
}
