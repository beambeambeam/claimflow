package claim

import (
	"errors"
	"slices"
	"strconv"
	"testing"

	"github.com/google/uuid"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/money"
)

// F03-TEST-09 (F03-AC-09, NFR-02, F04-AC-07): the claim policy matrix. A claim is visible to its requester, and to its
// assigned reviewer once it is no longer a DRAFT. The assigned reviewer, when not also the requester, may approve and
// reject a SUBMITTED claim (F-04); nobody else has an action. The rows cover every role, relation, and status. Since
// F-08 a Finance Officer also sees every claim that is not a DRAFT, with no action on these statuses (F08-TEST-09).
func TestF03TEST09PolicyMatrix(t *testing.T) {
	actors := map[string]uuid.UUID{
		"requester": uuid.Must(uuid.NewV7()),
		"reviewer":  uuid.Must(uuid.NewV7()),
		"finance":   uuid.Must(uuid.NewV7()),
	}
	other := uuid.Must(uuid.NewV7())

	statuses := []db.ClaimStatus{db.ClaimStatusDRAFT, db.ClaimStatusSUBMITTED, db.ClaimStatusAPPROVED, db.ClaimStatusREJECTED}

	none := func(db.ClaimStatus) []Action { return []Action{} }
	own := func(status db.ClaimStatus) []Action {
		if status == db.ClaimStatusDRAFT {
			return []Action{ActionEdit, ActionDelete}
		}
		if status == db.ClaimStatusSUBMITTED {
			return []Action{ActionWithdraw}
		}
		return []Action{}
	}

	// subject builds the claim as the actor relates to it. "own and assigned" is the corrupted assignment that points at the requester.
	relations := map[string]struct {
		subject func(actor uuid.UUID, status db.ClaimStatus) Subject
		visible func(status db.ClaimStatus) bool
		actions func(status db.ClaimStatus) []Action
	}{
		"own": {
			subject: func(actor uuid.UUID, status db.ClaimStatus) Subject {
				return Subject{RequesterID: actor, AssignedReviewerID: other, Status: status}
			},
			visible: func(db.ClaimStatus) bool { return true },
			actions: own,
		},
		"assigned": {
			subject: func(actor uuid.UUID, status db.ClaimStatus) Subject {
				return Subject{RequesterID: other, AssignedReviewerID: actor, Status: status}
			},
			visible: func(status db.ClaimStatus) bool { return status != db.ClaimStatusDRAFT },
			actions: func(status db.ClaimStatus) []Action {
				if status == db.ClaimStatusSUBMITTED {
					return []Action{ActionApprove, ActionReject, ActionReturn}
				}

				return []Action{}
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
			actions: own,
		},
	}

	for role, actor := range actors {
		for name, relation := range relations {
			for _, status := range statuses {
				t.Run(role+"/"+name+"/"+string(status), func(t *testing.T) {
					visible, actions := Allowed(Actor{ID: actor, Role: role}, relation.subject(actor, status))

					want := relation.visible(status) || (role == RoleFinance && status != db.ClaimStatusDRAFT)
					if visible != want {
						t.Fatalf("visible = %t, want %t", visible, want)
					}

					// A claim assigned to its own requester gives no reviewer action (BR-01).
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

	visible, actions := Allowed(Actor{ID: malee}, Subject{RequesterID: malee, AssignedReviewerID: malee, Status: db.ClaimStatusSUBMITTED})

	if !visible {
		t.Fatal("visible = false, want true: the requester sees their own claim")
	}

	for _, action := range []Action{ActionApprove, ActionReject, ActionReturn} {
		if slices.Contains(actions, action) {
			t.Fatalf("actions = %v, want no %s (BR-01)", actions, action)
		}
	}
}

func TestF05TEST06DraftAndReturnedPolicy(t *testing.T) {
	owner := uuid.New()
	other := uuid.New()

	for _, test := range []struct {
		name              string
		actor             uuid.UUID
		status            db.ClaimStatus
		reviewer, visible bool
		actions           []Action
	}{
		{"draft with reviewer", owner, db.ClaimStatusDRAFT, true, true, []Action{ActionEdit, ActionDelete, ActionSubmit}},
		{"draft without reviewer", owner, db.ClaimStatusDRAFT, false, true, []Action{ActionEdit, ActionDelete}},
		{"submitted", owner, db.ClaimStatusSUBMITTED, true, true, []Action{ActionWithdraw}},
		{"returned", owner, db.ClaimStatusRETURNED, false, true, []Action{ActionEdit, ActionSubmit, ActionWithdraw}},
		{"draft owned by someone else", other, db.ClaimStatusDRAFT, true, false, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			subject := Subject{RequesterID: owner, Status: test.status, OwnerHasReviewer: test.reviewer}
			visible, actions := Allowed(Actor{ID: test.actor}, subject)
			if visible != test.visible || actions == nil || !slices.Equal(actions, test.actions) {
				t.Fatalf("visible = %t, actions = %v; want %t, %v", visible, actions, test.visible, test.actions)
			}
		})
	}
}

// F06-TEST-06 (F06-AC-05, BR-14): the assigned reviewer of a SUBMITTED claim may return it until it has been returned
// MaxReturns times; after that approve and reject are what is left.
func TestF06TEST06ReturnIsAllowedBelowTheReturnLimit(t *testing.T) {
	reviewer := uuid.New()

	for _, test := range []struct {
		returnCount int
		actions     []Action
	}{
		{0, []Action{ActionApprove, ActionReject, ActionReturn}},
		{MaxReturns - 1, []Action{ActionApprove, ActionReject, ActionReturn}},
		{MaxReturns, []Action{ActionApprove, ActionReject}},
	} {
		subject := Subject{RequesterID: uuid.New(), AssignedReviewerID: reviewer, Status: db.ClaimStatusSUBMITTED, ReturnCount: test.returnCount}

		if _, actions := Allowed(Actor{ID: reviewer}, subject); !slices.Equal(actions, test.actions) {
			t.Fatalf("return count %d: actions = %v, want %v", test.returnCount, actions, test.actions)
		}
	}
}

// F07-TEST-05 (F07-AC-04, F07-AC-06): the owner may withdraw a SUBMITTED or RETURNED claim; a WITHDRAWN claim gives
// nobody an action; the assigned reviewer never gets withdraw.
func TestF07TEST05WithdrawIsTheOwnersOnSubmittedOrReturned(t *testing.T) {
	owner, reviewer := uuid.New(), uuid.New()

	for _, test := range []struct {
		name    string
		actor   uuid.UUID
		status  db.ClaimStatus
		actions []Action
	}{
		{"owner, submitted", owner, db.ClaimStatusSUBMITTED, []Action{ActionWithdraw}},
		{"owner, returned", owner, db.ClaimStatusRETURNED, []Action{ActionEdit, ActionSubmit, ActionWithdraw}},
		{"owner, withdrawn", owner, db.ClaimStatusWITHDRAWN, []Action{}},
		{"reviewer, withdrawn", reviewer, db.ClaimStatusWITHDRAWN, []Action{}},
		{"reviewer, returned", reviewer, db.ClaimStatusRETURNED, []Action{}},
		{"owner, draft", owner, db.ClaimStatusDRAFT, []Action{ActionEdit, ActionDelete, ActionSubmit}},
		{"owner, approved", owner, db.ClaimStatusAPPROVED, []Action{}},
		{"owner, rejected", owner, db.ClaimStatusREJECTED, []Action{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			subject := Subject{RequesterID: owner, AssignedReviewerID: reviewer, Status: test.status, OwnerHasReviewer: true}

			if _, actions := Allowed(Actor{ID: test.actor}, subject); !slices.Equal(actions, test.actions) {
				t.Fatalf("actions = %v, want %v", actions, test.actions)
			}
		})
	}

	// The reviewer of a SUBMITTED claim has the decisions and no withdraw.
	subject := Subject{RequesterID: owner, AssignedReviewerID: reviewer, Status: db.ClaimStatusSUBMITTED}
	if _, actions := Allowed(Actor{ID: reviewer}, subject); slices.Contains(actions, ActionWithdraw) {
		t.Fatalf("actions = %v, want no withdraw for the reviewer", actions)
	}

	// An unrelated member cannot see the claim, so the check refuses before the status.
	if err := check(Actor{ID: uuid.New()}, subject, ActionWithdraw); !errors.Is(err, ErrForbidden) {
		t.Fatalf("check error = %v, want ErrForbidden", err)
	}
}

// F08-TEST-09 (F08-AC-07, F08-AC-11, SDS §5.2): the Finance rows of the policy matrix, on every status and with totals
// on both sides of the threshold, which the policy does not read. Only a Finance Officer who is not the requester gets
// final-approve and final-reject, and only on PENDING_FINANCE. A Finance Officer sees every claim but a DRAFT, and gets
// approve, reject, and return only when assigned. The owner and the reviewer get nothing on PENDING_FINANCE.
func TestF08TEST09FinancePolicyMatrix(t *testing.T) {
	requester, reviewer, unrelated, finance := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	decisions := []Action{ActionApprove, ActionReject, ActionReturn}
	final := []Action{ActionFinalApprove, ActionFinalReject}
	ownActions := map[db.ClaimStatus][]Action{
		db.ClaimStatusDRAFT:     {ActionEdit, ActionDelete, ActionSubmit},
		db.ClaimStatusSUBMITTED: {ActionWithdraw},
		db.ClaimStatusRETURNED:  {ActionEdit, ActionSubmit, ActionWithdraw},
	}

	// subject is the claim's requester and assigned reviewer; visible and actions are by status, absent meaning
	// false and [].
	rows := []struct {
		name               string
		actor              Actor
		owner, assigned    uuid.UUID
		visibleOnDraft     bool
		visibleAfterSubmit bool
		actions            map[db.ClaimStatus][]Action
	}{
		{"owner", Actor{ID: requester, Role: "requester"}, requester, reviewer, true, true, ownActions},
		{
			"assigned reviewer",
			Actor{ID: reviewer, Role: "reviewer"},
			requester, reviewer, false, true,
			map[db.ClaimStatus][]Action{db.ClaimStatusSUBMITTED: decisions},
		},
		{"unrelated reviewer", Actor{ID: unrelated, Role: "reviewer"}, requester, reviewer, false, false, nil},
		{
			"finance, not own",
			Actor{ID: finance, Role: RoleFinance},
			requester, reviewer, false, true,
			map[db.ClaimStatus][]Action{db.ClaimStatusPENDINGFINANCE: final},
		},
		{"finance, own", Actor{ID: finance, Role: RoleFinance}, finance, reviewer, true, true, ownActions},
		{
			"finance, assigned",
			Actor{ID: finance, Role: RoleFinance},
			requester, finance, false, true,
			map[db.ClaimStatus][]Action{db.ClaimStatusSUBMITTED: decisions, db.ClaimStatusPENDINGFINANCE: final},
		},
	}

	statuses := []db.ClaimStatus{
		db.ClaimStatusDRAFT, db.ClaimStatusSUBMITTED, db.ClaimStatusRETURNED, db.ClaimStatusPENDINGFINANCE,
		db.ClaimStatusAPPROVED, db.ClaimStatusREJECTED, db.ClaimStatusWITHDRAWN,
	}

	for _, row := range rows {
		for _, status := range statuses {
			for _, total := range []int64{1_000_000, 1_500_000} {
				t.Run(row.name+"/"+string(status)+"/"+strconv.FormatInt(total, 10), func(t *testing.T) {
					subject := Subject{
						RequesterID: row.owner, AssignedReviewerID: row.assigned, Status: status,
						OwnerHasReviewer: true, TotalSatang: money.Satang(total),
					}
					visible, actions := Allowed(row.actor, subject)

					wantVisible := row.visibleAfterSubmit
					if status == db.ClaimStatusDRAFT {
						wantVisible = row.visibleOnDraft
					}

					wantActions := row.actions[status]
					if wantActions == nil {
						wantActions = []Action{}
					}

					if visible != wantVisible || actions == nil || !slices.Equal(actions, wantActions) {
						t.Fatalf("visible = %t, actions = %v; want %t, %v", visible, actions, wantVisible, wantActions)
					}
				})
			}
		}
	}
}
