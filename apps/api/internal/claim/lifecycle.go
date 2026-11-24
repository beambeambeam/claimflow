package claim

import "github.com/beambeambeam/claimflow/apps/api/internal/db"

// transition is the destination and history action for an allowed status change.
type transition struct {
	to      db.ClaimStatus
	history db.ClaimAction
}

// transitions lists the status changes by action and source status (F-04 §1.3, F-05 §1.10).
var transitions = map[Action]map[db.ClaimStatus]transition{
	ActionApprove: {db.ClaimStatusSUBMITTED: {to: db.ClaimStatusAPPROVED, history: db.ClaimActionAPPROVE}},
	ActionReject:  {db.ClaimStatusSUBMITTED: {to: db.ClaimStatusREJECTED, history: db.ClaimActionREJECT}},
	ActionSubmit: {
		db.ClaimStatusDRAFT:    {to: db.ClaimStatusSUBMITTED, history: db.ClaimActionSUBMIT},
		db.ClaimStatusRETURNED: {to: db.ClaimStatusSUBMITTED, history: db.ClaimActionRESUBMIT},
	},
}
