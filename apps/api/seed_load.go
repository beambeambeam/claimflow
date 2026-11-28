package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/beambeambeam/claimflow/apps/api/internal/claim"
	"github.com/beambeambeam/claimflow/apps/api/internal/daterange"
	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/money"
)

// loadClaimCount is how many claims `api seed-load` adds (F-10 §1.8): CLM-LOAD-0001 to CLM-LOAD-0240.
const loadClaimCount = 240

// loadZone is the zone of the dataset's dates (AS-06).
var loadZone = daterange.Bangkok

// loadRequesters are the users who own load claims: every seed requester and reviewer that has a reviewer.
var loadRequesters = []string{"req.anan", "req.ploy", "req.niran", "rev.malee", "rev.somchai", "rev.prasert"}

// loadCategories are the expense categories in the order of the enum.
var loadCategories = []db.ExpenseCategory{
	db.ExpenseCategoryTRAVEL, db.ExpenseCategoryACCOMMODATION, db.ExpenseCategoryMEALS,
	db.ExpenseCategorySUPPLIES, db.ExpenseCategoryTRAINING, db.ExpenseCategoryOTHER,
}

var loadTitles = []string{
	"Client site visit", "Workshop supplies", "Training course fee", "Regional conference", "Office supplies",
	"Team lunch", "Vendor meeting travel", "Audit fieldwork",
}

var loadReasons = []string{
	"Receipt is missing or unreadable.", "Amount does not match the receipt.", "Please add the purpose of the trip.",
	"This expense is not covered by the policy.", "Please attach the original tax invoice.",
}

// The history paths of a load claim. Each path starts from a draft; the lifecycle decides every status.
var (
	loadApproved         = []claim.Action{claim.ActionSubmit, claim.ActionApprove}
	loadRejected         = []claim.Action{claim.ActionSubmit, claim.ActionReject}
	loadReturned         = []claim.Action{claim.ActionSubmit, claim.ActionReturn}
	loadWithdrawn        = []claim.Action{claim.ActionSubmit, claim.ActionWithdraw}
	loadWithdrawnReturn  = []claim.Action{claim.ActionSubmit, claim.ActionReturn, claim.ActionWithdraw}
	loadResubmitApproved = []claim.Action{claim.ActionSubmit, claim.ActionReturn, claim.ActionSubmit, claim.ActionApprove}
	loadFinalApproved    = []claim.Action{claim.ActionSubmit, claim.ActionApprove, claim.ActionFinalApprove}
	loadFinalRejected    = []claim.Action{claim.ActionSubmit, claim.ActionApprove, claim.ActionFinalReject}
	loadWaiting          = []claim.Action{claim.ActionSubmit}
	loadResubmitted      = []claim.Action{claim.ActionSubmit, claim.ActionReturn, claim.ActionSubmit}
)

// loadTotal says how high the total of a claim must be.
type loadTotal int

const (
	// totalFree leaves the total to the claim's number: one in four is more than 10,000.00.
	totalFree loadTotal = iota
	// totalLow keeps the total at 10,000.00 or less, so a reviewer's approval is final.
	totalLow
	// totalHigh makes the total more than 10,000.00, so a reviewer's approval goes to Finance.
	totalHigh
)

// loadPlan is the shape of one load claim before its details: who owns it, which path its history takes, how high its
// total is, and when its last event happens.
type loadPlan struct {
	requester string
	path      []claim.Action
	total     loadTotal
	last      time.Time
}

// loadEvent is one history row of a load claim.
type loadEvent struct {
	actor   string
	action  db.ClaimAction
	from    db.ClaimStatus
	to      db.ClaimStatus
	comment string
	at      time.Time
}

// loadClaim is one load claim with everything the database stores.
type loadClaim struct {
	claimNo   string
	requester string
	title     string
	purpose   string
	items     []claim.LineItemInput
	total     money.Satang
	status    db.ClaimStatus
	returns   int16
	version   int32
	events    []loadEvent
}

// first is the first submission and last the time of the final event.
func (c *loadClaim) first() time.Time { return c.events[0].at }

func (c *loadClaim) last() time.Time { return c.events[len(c.events)-1].at }

// submitted is the time of the latest submission or resubmission.
func (c *loadClaim) submitted() time.Time {
	for i := len(c.events) - 1; i >= 0; i-- {
		if c.events[i].action == db.ClaimActionSUBMIT || c.events[i].action == db.ClaimActionRESUBMIT {
			return c.events[i].at
		}
	}

	return c.first() // coverage-ignore: every path starts with SUBMIT
}

// decidedAt is the decision time: set for an APPROVED or REJECTED claim, empty for any other (a return, a withdrawal,
// and a send to Finance are not decisions).
func (c *loadClaim) decidedAt() pgtype.Timestamptz {
	if c.status == db.ClaimStatusAPPROVED || c.status == db.ClaimStatusREJECTED {
		return pgtype.Timestamptz{Time: c.last(), Valid: true}
	}

	return pgtype.Timestamptz{}
}

// seedUserIndex finds the role and reviewer of a base seed user by username.
func seedUserIndex() map[string]seedUser {
	index := make(map[string]seedUser, len(seedUsers))
	for _, user := range seedUsers {
		index[user.username] = user
	}

	return index
}

// loadPlans lists the plan of every load claim, in claim number order. It is plain arithmetic on the claim number, so
// every run builds the same dataset. A third of the claims wait for rev.malee (80 of them, so three runs of 20 rounds
// have claims to approve). The others cycle through ten history paths and five decision months, and through the six
// requesters, so every combination appears.
func loadPlans() []loadPlan {
	users := seedUserIndex()
	plans := make([]loadPlan, 0, loadClaimCount)
	other := 0

	for i := 1; i <= loadClaimCount; i++ {
		if i%3 == 0 {
			// Waiting for rev.malee: submitted between 1 October and 25 November 2026, never more than 10,000.00.
			path := loadWaiting
			if (i/3)%4 == 0 {
				path = loadResubmitted
			}

			day := (i / 3 * 7) % 56
			plans = append(plans, loadPlan{
				requester: loadRequesters[(i/3)%2], path: path, total: totalLow,
				last: time.Date(2026, time.October, 1+day, 9+i%9, i*11%60, 0, 0, loadZone),
			})

			continue
		}

		round := other / 10
		requester := loadRequesters[(other+round)%len(loadRequesters)]
		financeApproves := users[users[requester].reviewer].role == claim.RoleFinance
		path, total := loadSlot(other%10, round, financeApproves)

		month := time.July + time.Month(round%5)
		plans = append(plans, loadPlan{
			requester: requester, path: path, total: total,
			last: time.Date(2026, month, 3+(other*7+5)%23, 9+other*5%9, other*11%60, 0, 0, loadZone),
		})
		other++
	}

	return plans
}

// loadSlot returns the history path and the total rule of slot (0 to 9) in a round. When Finance is the reviewer
// (the claim's owner is rev.prasert), an approval is final, so the two-stage paths become one-stage ones.
func loadSlot(slot, round int, financeApproves bool) ([]claim.Action, loadTotal) {
	odd := round%2 == 1

	switch slot {
	case 0, 9:
		if financeApproves {
			return loadApproved, totalFree
		}

		return loadApproved, totalLow
	case 1:
		return loadRejected, totalFree
	case 2:
		return loadReturned, totalFree
	case 3:
		// More than 10,000.00: a reviewer's approval waits for Finance, and Finance's own approval is final.
		return loadApproved, totalHigh
	case 4:
		if odd {
			return loadWithdrawnReturn, totalFree
		}

		return loadWithdrawn, totalFree
	case 5:
		if financeApproves {
			return loadResubmitApproved, totalFree
		}

		return loadResubmitApproved, totalLow
	case 6:
		if financeApproves {
			return loadApproved, totalHigh
		}

		return loadFinalApproved, totalHigh
	case 7:
		if financeApproves {
			return loadRejected, totalHigh
		}

		return loadFinalRejected, totalHigh
	default:
		if odd {
			return loadResubmitted, totalFree
		}

		return loadWaiting, totalFree
	}
}

// buildLoadClaims builds the whole dataset. Every history is derived with the lifecycle of internal/claim, and every
// set of line items is checked with its rules, so a plan that the lifecycle or the BR rules refuse is an error, not data.
func buildLoadClaims() ([]loadClaim, error) {
	users := seedUserIndex()
	plans := loadPlans()
	claims := make([]loadClaim, 0, len(plans))

	for i := range plans {
		built, err := buildLoadClaim(i+1, &plans[i], users)
		if err != nil { // coverage-ignore
			return nil, err
		}

		claims = append(claims, built)
	}

	return claims, nil
}

// buildLoadClaim builds claim number n from its plan.
func buildLoadClaim(n int, plan *loadPlan, users map[string]seedUser) (loadClaim, error) {
	built := loadClaim{
		claimNo: fmt.Sprintf("CLM-LOAD-%04d", n), requester: plan.requester,
		title:   fmt.Sprintf("%s (load %04d)", loadTitles[n%len(loadTitles)], n),
		purpose: "Synthetic claim of the NFR-07 load dataset (F-10).",
		items:   loadItems(n, plan.total),
	}

	for _, item := range built.items {
		built.total += item.Amount
	}

	reviewer := users[plan.requester].reviewer
	times := loadTimes(n, plan)
	status := db.ClaimStatusDRAFT

	for k, action := range plan.path {
		actor := plan.requester

		switch action {
		case claim.ActionApprove, claim.ActionReject, claim.ActionReturn:
			actor = reviewer
		case claim.ActionFinalApprove, claim.ActionFinalReject:
			actor = "fin.wanida"
		case claim.ActionSubmit, claim.ActionWithdraw, claim.ActionEdit, claim.ActionDelete:
			// The requester acts.
		}

		to, history, ok := claim.NextStatus(action, status, built.total, users[actor].role)
		if !ok { // coverage-ignore
			return loadClaim{}, fmt.Errorf("%s: %s is not allowed from %s", built.claimNo, action, status)
		}

		built.events = append(built.events, loadEvent{
			actor: actor, action: history, from: status, to: to, comment: loadComment(n, history), at: times[k],
		})

		built.version++

		if history == db.ClaimActionRETURN {
			built.returns++
		}

		status = to
	}

	built.status = status

	// The expense dates fall one to 30 days before the first submission (BR-07).
	for k := range built.items {
		built.items[k].ExpenseDate = calendarDay(built.first().AddDate(0, 0, -(1 + (n+3*k)%30)))
	}

	if err := claim.CheckLineItems(built.items, built.last(), built.first()); err != nil { // coverage-ignore
		return loadClaim{}, fmt.Errorf("%s: %w", built.claimNo, err)
	}

	return built, nil
}

// calendarDay is the Bangkok calendar date of t as a date at midnight UTC, the way a DATE column is written.
func calendarDay(t time.Time) time.Time {
	year, month, day := t.In(loadZone).Date()

	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// loadTimes returns the time of every event of plan: the last one is plan.last, and each earlier one comes 2 to 21
// hours before the next.
func loadTimes(n int, plan *loadPlan) []time.Time {
	times := make([]time.Time, len(plan.path))
	times[len(times)-1] = plan.last

	for k := len(times) - 1; k > 0; k-- {
		times[k-1] = times[k].Add(-time.Duration(2+(n+k)%20) * time.Hour)
	}

	return times
}

// loadComment is the comment of a history entry: a reason of at least ten characters where BR-04 needs one.
func loadComment(n int, action db.ClaimAction) string {
	switch action {
	case db.ClaimActionREJECT, db.ClaimActionRETURN, db.ClaimActionFINALREJECT:
		return loadReasons[n%len(loadReasons)]
	case db.ClaimActionWITHDRAW:
		return "No longer needed for this trip."
	case db.ClaimActionSUBMIT, db.ClaimActionRESUBMIT:
		return ""
	case db.ClaimActionAPPROVE, db.ClaimActionFINALAPPROVE:
		if n%2 == 0 {
			return "Approved."
		}
	}

	return ""
}

// loadItems builds one to five line items for claim number n. Amounts are whole baht within BR-05 and BR-12, and the
// total follows rule: at most 10,000.00 for totalLow, more than that for totalHigh, and for totalFree more than that
// on every fourth claim.
func loadItems(n int, rule loadTotal) []claim.LineItemInput {
	const threshold = claim.HighValueThresholdSatang

	high := rule == totalHigh || (rule == totalFree && n%4 == 0)
	items := make([]claim.LineItemInput, 1+(n*7+n/5)%5)

	var sum money.Satang

	for k := range items {
		category := loadCategories[(n+2*k+n/6)%len(loadCategories)]
		amount := money.Satang(20_000+((n*131+k*57)%30)*10_000) / 100 * 100

		if k == 0 && high {
			// One large item, so that the total passes 10,000.00 whatever the others are.
			if category == db.ExpenseCategoryMEALS {
				category = db.ExpenseCategoryTRAVEL
			}

			amount = money.Satang(1_100_000 + (n%9)*100_000)
		}

		if category == db.ExpenseCategoryMEALS {
			amount = min(amount, claim.MaxMealsAmount)
		}

		items[k] = claim.LineItemInput{
			Category: category, Description: fmt.Sprintf("%s, item %d", loadTitles[(n+k)%len(loadTitles)], k+1),
			Amount: amount, ReceiptRef: fmt.Sprintf("LOAD-%04d-%d", n, k+1),
		}
		sum += amount
	}

	if !high && sum > threshold {
		// Scale a low total down to 10,000.00 at most, keeping whole baht.
		for k := range items {
			items[k].Amount = items[k].Amount * threshold / sum / 100 * 100
		}
	}

	return items
}

var errLoadNeedsSeed = errors.New("the base seed users are missing; run seed first")

// seedLoadRows adds the load claims that do not exist yet, each with its line items and history, and returns how many
// it added. ids holds the user ids by username.
func seedLoadRows(ctx context.Context, queries *db.Queries, ids map[string]uuid.UUID, claims []loadClaim) (int, error) {
	added := 0

	for i := range claims {
		built := &claims[i]
		id := uuid.Must(uuid.NewV7())

		inserted, err := queries.InsertLoadClaim(ctx, db.InsertLoadClaimParams{
			ID: id, ClaimNo: built.claimNo, Title: built.title, Purpose: built.purpose, Status: built.status,
			TotalSatang: built.total, Version: built.version, ReturnCount: built.returns,
			FirstSubmittedAt: pgtype.Timestamptz{Time: built.first(), Valid: true},
			SubmittedAt:      pgtype.Timestamptz{Time: built.submitted(), Valid: true},
			DecidedAt:        built.decidedAt(),
			CreatedAt:        pgtype.Timestamptz{Time: built.first().Add(-time.Duration(1+i%5) * time.Hour), Valid: true},
			UpdatedAt:        pgtype.Timestamptz{Time: built.last(), Valid: true},
			Requester:        built.requester,
		})
		if err != nil { // coverage-ignore
			return 0, fmt.Errorf("insert claim %s: %w", built.claimNo, err)
		}

		if inserted == 0 { // loaded before
			continue
		}

		added++

		for n, item := range built.items {
			err := queries.InsertLineItem(ctx, db.InsertLineItemParams{
				ID: uuid.Must(uuid.NewV7()), ClaimID: id, LineNo: int16(n + 1), Category: item.Category, Description: item.Description,
				ExpenseDate: pgtype.Date{Time: item.ExpenseDate, Valid: true}, AmountSatang: item.Amount, ReceiptRef: item.ReceiptRef,
			})
			if err != nil { // coverage-ignore
				return 0, fmt.Errorf("insert line item of %s: %w", built.claimNo, err)
			}
		}

		for _, event := range built.events {
			err := queries.InsertClaimHistory(ctx, db.InsertClaimHistoryParams{
				ClaimID: id, ActorID: ids[event.actor], Action: event.action,
				FromStatus: db.NullClaimStatus{ClaimStatus: event.from, Valid: true}, ToStatus: event.to,
				Comment: pgtype.Text{String: event.comment, Valid: event.comment != ""},
				Now:     pgtype.Timestamptz{Time: event.at, Valid: true},
			})
			if err != nil { // coverage-ignore
				return 0, fmt.Errorf("insert history of %s: %w", built.claimNo, err)
			}
		}
	}

	return added, nil
}

// seedLoad adds the 240 load claims of F-10 §1.8 that do not exist yet. It needs the base seed users and fails before it
// writes anything when they are missing. A second run adds no claim. It writes the number of claims added to out.
func seedLoad(ctx context.Context, databaseURL string, out io.Writer) error {
	claims, err := buildLoadClaims()
	if err != nil { // coverage-ignore
		return err
	}

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil { // coverage-ignore
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close(ctx)

	var added int

	err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		queries := db.New(tx)

		usernames := make([]string, 0, len(seedUsers))
		for _, user := range seedUsers {
			usernames = append(usernames, user.username)
		}

		rows, err := queries.ListUserIDsByUsername(ctx, usernames)
		if err != nil { // coverage-ignore
			return fmt.Errorf("read seed users: %w", err)
		}

		if len(rows) != len(usernames) {
			return errLoadNeedsSeed
		}

		ids := make(map[string]uuid.UUID, len(rows))
		for _, row := range rows {
			ids[row.Username] = row.ID
		}

		added, err = seedLoadRows(ctx, queries, ids, claims)

		return err
	})
	if err != nil {
		return fmt.Errorf("seed load: %w", err)
	}

	if _, err := fmt.Fprintf(out, "added %d new load claims\n", added); err != nil { // coverage-ignore
		return fmt.Errorf("write summary: %w", err)
	}

	return nil
}
