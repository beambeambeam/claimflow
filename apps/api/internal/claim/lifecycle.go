package claim

import "github.com/beambeambeam/claimflow/apps/api/internal/db"

// transition is what an action does to a claim: the status it needs, the status it leads to, and the history action.
type transition struct {
	from, to db.ClaimStatus
	history  db.ClaimAction
}

// transitions is the lifecycle of Sprint 1 (F-04 §1.3). APPROVED and REJECTED are terminal. A later feature adds rows.
var transitions = map[Action]transition{
	ActionApprove: {from: db.ClaimStatusSUBMITTED, to: db.ClaimStatusAPPROVED, history: db.ClaimActionAPPROVE},
	ActionReject:  {from: db.ClaimStatusSUBMITTED, to: db.ClaimStatusREJECTED, history: db.ClaimActionREJECT},
}
