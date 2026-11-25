package claim

import "github.com/beambeambeam/claimflow/apps/api/internal/db"

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
