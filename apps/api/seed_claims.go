package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/money"
)

// seedItem is one line item of a sample claim.
type seedItem struct {
	category    db.ExpenseCategory
	description string
	amount      money.Satang
}

// seedClaim is one sample claim of F-04 §4. A draft has no history. For submitted claims, decision is empty for a claim that
// still waits, or APPROVE or REJECT for one its reviewer has decided. reason is the comment of the decision.
type seedClaim struct {
	claimNo   string
	requester string
	title     string
	purpose   string
	items     []seedItem
	draft     bool
	decision  db.ClaimAction
	reason    string
	// createdAgo orders the examples in My Claims and the review queue.
	createdAgo time.Duration
}

// seedClaims are the sample claims, with fixed claim numbers in a range the live sequence does not reach. A second run
// finds them by number and adds nothing.
var seedClaims = []seedClaim{
	{
		claimNo: "CLM-2026-900001", requester: "req.anan", title: "Chiang Mai site visit", purpose: "Train and taxi for the site visit",
		items:      []seedItem{{db.ExpenseCategoryTRAVEL, "Train, Bangkok to Chiang Mai", 80000}, {db.ExpenseCategoryTRAVEL, "Taxi", 45000}},
		createdAgo: 6 * time.Hour,
	},
	{
		claimNo: "CLM-2026-900002", requester: "req.anan", title: "Team workshop supplies", purpose: "Printing and stationery for the workshop",
		items:      []seedItem{{db.ExpenseCategorySUPPLIES, "Printing and stationery", 125000}},
		decision:   db.ClaimActionAPPROVE,
		createdAgo: 72 * time.Hour,
	},
	{
		claimNo: "CLM-2026-900003", requester: "req.ploy", title: "Client dinner", purpose: "Dinner with the client after the audit",
		items:      []seedItem{{db.ExpenseCategoryMEALS, "Dinner for four", 125000}},
		decision:   db.ClaimActionREJECT,
		reason:     "Receipt is missing.",
		createdAgo: 48 * time.Hour,
	},
	{
		claimNo: "CLM-2026-900004", requester: "req.niran", title: "Training course fee", purpose: "Data handling course",
		items:      []seedItem{{db.ExpenseCategoryTRAINING, "Course fee", 125000}},
		createdAgo: 12 * time.Hour,
	},
	{
		claimNo: "CLM-2026-900005", requester: "rev.malee", title: "Section meeting expenses", purpose: "Venue and supplies for the section meeting",
		items:      []seedItem{{db.ExpenseCategoryOTHER, "Venue hire", 125000}},
		createdAgo: 24 * time.Hour,
	},
	{
		claimNo: "CLM-2026-900006", requester: "rev.prasert", title: "Regional conference", purpose: "Flights and hotel for the regional conference",
		items: []seedItem{
			{db.ExpenseCategoryTRAVEL, "Return flights", 2_000_000},
			{db.ExpenseCategoryACCOMMODATION, "Hotel, 4 nights", 2_000_000},
		},
		createdAgo: 36 * time.Hour,
	},
	{
		claimNo: "CLM-2026-900007", requester: "req.anan", title: "Office supplies – draft", purpose: "Supplies for the office",
		items: []seedItem{{db.ExpenseCategorySUPPLIES, "Printer paper", 100000}, {db.ExpenseCategorySUPPLIES, "Folders", 25000}},
		draft: true,
	},
}

// decisionStatuses is the status a claim has after each decision.
var decisionStatuses = map[db.ClaimAction]db.ClaimStatus{
	db.ClaimActionAPPROVE: db.ClaimStatusAPPROVED,
	db.ClaimActionREJECT:  db.ClaimStatusREJECTED,
}

// seedClaimRows adds the sample claims that do not exist yet, each with its line items and history, as of now. ids holds
// the user ids by username, and reviewers the username of each requester's reviewer. It returns how many claims it added.
func seedClaimRows(ctx context.Context, queries *db.Queries, ids map[string]uuid.UUID, reviewers map[string]string, now time.Time) (int, error) {
	added := 0

	for i := range seedClaims {
		claim := &seedClaims[i]
		createdAt := now.Add(-claim.createdAgo)
		decidedAt := createdAt.Add(2 * time.Hour)
		status, version, decided := db.ClaimStatusSUBMITTED, int32(0), pgtype.Timestamptz{}
		if claim.draft {
			status = db.ClaimStatusDRAFT
		} else if claim.decision != "" {
			status, version, decided = decisionStatuses[claim.decision], 1, pgtype.Timestamptz{Time: decidedAt, Valid: true}
		}

		var total money.Satang
		for _, item := range claim.items {
			total += item.amount
		}

		id := uuid.Must(uuid.NewV7())

		inserted, err := queries.InsertSeedClaim(ctx, db.InsertSeedClaimParams{
			ID: id, ClaimNo: claim.claimNo, Title: claim.title, Purpose: claim.purpose, Status: status, TotalSatang: total,
			Version: version, CreatedAt: pgtype.Timestamptz{Time: createdAt, Valid: true}, DecidedAt: decided,
			Requester: claim.requester,
		})
		if err != nil { // coverage-ignore
			return 0, fmt.Errorf("insert claim %s: %w", claim.claimNo, err)
		}

		if inserted == 0 { // seeded before
			continue
		}

		added++

		for n, item := range claim.items {
			err := queries.InsertLineItem(ctx, db.InsertLineItemParams{
				ID: uuid.Must(uuid.NewV7()), ClaimID: id, LineNo: int16(n + 1), Category: item.category, Description: item.description,
				ExpenseDate: pgtype.Date{Time: createdAt.AddDate(0, 0, -1), Valid: true}, AmountSatang: item.amount, ReceiptRef: fmt.Sprintf("RCPT-%d", n+1),
			})
			if err != nil { // coverage-ignore
				return 0, fmt.Errorf("insert line item of %s: %w", claim.claimNo, err)
			}
		}

		if claim.draft {
			continue
		}

		err = queries.InsertClaimHistory(ctx, db.InsertClaimHistoryParams{
			ClaimID: id, ActorID: ids[claim.requester], Action: db.ClaimActionSUBMIT, ToStatus: db.ClaimStatusSUBMITTED, Now: pgtype.Timestamptz{Time: createdAt, Valid: true},
		})
		if err != nil { // coverage-ignore
			return 0, fmt.Errorf("insert history of %s: %w", claim.claimNo, err)
		}

		if claim.decision == "" {
			continue
		}

		err = queries.InsertClaimHistory(ctx, db.InsertClaimHistoryParams{
			ClaimID: id, ActorID: ids[reviewers[claim.requester]], Action: claim.decision,
			FromStatus: db.NullClaimStatus{ClaimStatus: db.ClaimStatusSUBMITTED, Valid: true}, ToStatus: status,
			Comment: pgtype.Text{String: claim.reason, Valid: claim.reason != ""}, Now: decided,
		})
		if err != nil { // coverage-ignore
			return 0, fmt.Errorf("insert decision of %s: %w", claim.claimNo, err)
		}
	}

	return added, nil
}
