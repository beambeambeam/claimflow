package claim

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/money"
)

var (
	// ErrForbidden means the actor may not take the action. For Create it is BR-15: no assigned reviewer.
	ErrForbidden = errors.New("claim: forbidden")
	// ErrNotFound means no claim has the id.
	ErrNotFound = errors.New("claim: not found")
	// ErrDraftNotSupported means Create was asked to save a draft, which F-05 adds in Sprint 2.
	ErrDraftNotSupported = errors.New("claim: drafts are not supported yet")
)

// bangkok is the zone of every business date and of the claim number year (AS-06). Thailand has no daylight saving time.
var bangkok = time.FixedZone("Asia/Bangkok", 7*60*60)

// Pool is the database handle of a Service: it runs queries and starts the transaction of each call (NFR-06).
type Pool interface {
	db.DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Service is the claim module (SDS §3.1).
type Service struct {
	pool Pool
	now  func() time.Time
}

// NewService returns a Service over pool. now is the clock for submission times and the claim number year; tests inject a fixed one.
func NewService(pool Pool, now func() time.Time) Service {
	return Service{pool: pool, now: now}
}

// Create stores a claim and submits it in one transaction: the claim, its line items, and one SUBMIT history entry
// (F02-AC-01, F02-AC-02). The assigned reviewer is copied from the actor's reviewer at this moment and never changes
// afterwards (BR-17). An actor with no reviewer gets ErrForbidden (BR-15). The total is the sum of the items; the
// input has no total (BR-08). Saving a draft is F-05, so submit must be true for now.
func (s Service) Create(ctx context.Context, actor Actor, in Input, submit bool) (Claim, error) {
	if !submit {
		return Claim{}, ErrDraftNotSupported
	}

	now := s.now()

	var created Claim

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		queries := db.New(tx)

		member, err := queries.GetClaimMember(ctx, actor.ID)
		if err != nil { // coverage-ignore
			return fmt.Errorf("get member: %w", err)
		}

		if !member.ReviewerID.Valid {
			return ErrForbidden
		}

		if err := checkRules(in, now); err != nil {
			return err
		}

		id, err := insertSubmitted(ctx, queries, actor, member.ReviewerID, in, now)
		if err != nil { // coverage-ignore
			return err
		}

		created, err = load(ctx, queries, id)

		return err
	})
	if err != nil {
		return Claim{}, fmt.Errorf("create claim: %w", err)
	}

	return created, nil
}

// insertSubmitted writes the claim, its line items in request order, and the SUBMIT history entry, and returns the claim id.
func insertSubmitted(ctx context.Context, queries *db.Queries, actor Actor, reviewerID uuid.NullUUID, in Input, now time.Time) (uuid.UUID, error) {
	seq, err := queries.NextClaimSequence(ctx)
	if err != nil { // coverage-ignore
		return uuid.Nil, fmt.Errorf("next claim number: %w", err)
	}

	var total money.Satang
	for _, item := range in.Items {
		total += item.Amount
	}

	id := uuid.Must(uuid.NewV7())

	err = queries.InsertClaim(ctx, db.InsertClaimParams{
		ID:                 id,
		ClaimNo:            claimNumber(now, seq),
		RequesterID:        actor.ID,
		AssignedReviewerID: reviewerID,
		Title:              in.Title,
		Purpose:            in.Purpose,
		Status:             db.ClaimStatusSUBMITTED,
		TotalSatang:        total,
		FirstSubmittedAt:   timestamptz(now),
		SubmittedAt:        timestamptz(now),
		Now:                timestamptz(now),
	})
	if err != nil { // coverage-ignore
		return uuid.Nil, fmt.Errorf("insert claim: %w", err)
	}

	for i, item := range in.Items {
		err = queries.InsertLineItem(ctx, db.InsertLineItemParams{
			ID:           uuid.Must(uuid.NewV7()),
			ClaimID:      id,
			LineNo:       int16(i + 1), //nolint:gosec // BR-05 limits a claim to 20 items, and the table CHECK enforces it
			Category:     item.Category,
			Description:  item.Description,
			ExpenseDate:  pgtype.Date{Time: item.ExpenseDate, Valid: true},
			AmountSatang: item.Amount,
			ReceiptRef:   item.ReceiptRef,
		})
		if err != nil { // coverage-ignore
			return uuid.Nil, fmt.Errorf("insert line item %d: %w", i+1, err)
		}
	}

	err = queries.InsertClaimHistory(ctx, db.InsertClaimHistoryParams{
		ClaimID:    id,
		ActorID:    actor.ID,
		Action:     db.ClaimActionSUBMIT,
		FromStatus: db.NullClaimStatus{},
		ToStatus:   db.ClaimStatusSUBMITTED,
		Now:        timestamptz(now),
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert history: %w", err)
	}

	return id, nil
}

// load reads a claim with its line items and history.
func load(ctx context.Context, queries *db.Queries, id uuid.UUID) (Claim, error) {
	row, err := queries.GetClaim(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Claim{}, ErrNotFound
	}
	if err != nil { // coverage-ignore
		return Claim{}, fmt.Errorf("get claim: %w", err)
	}

	items, err := queries.ListClaimLineItems(ctx, id)
	if err != nil { // coverage-ignore
		return Claim{}, fmt.Errorf("list line items: %w", err)
	}

	history, err := queries.ListClaimHistory(ctx, id)
	if err != nil { // coverage-ignore
		return Claim{}, fmt.Errorf("list history: %w", err)
	}

	claim := Claim{
		ID:               row.ID,
		ClaimNo:          row.ClaimNo,
		Status:           row.Status,
		Title:            row.Title,
		Purpose:          row.Purpose,
		Total:            row.TotalSatang,
		ReturnCount:      int(row.ReturnCount),
		Version:          int(row.Version),
		Requester:        Member{ID: row.RequesterID, Name: row.RequesterName, Position: row.RequesterPosition},
		FirstSubmittedAt: optionalTime(row.FirstSubmittedAt),
		SubmittedAt:      optionalTime(row.SubmittedAt),
		DecidedAt:        optionalTime(row.DecidedAt),
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
		Items:            make([]LineItem, 0, len(items)),
		History:          make([]HistoryEntry, 0, len(history)),
		AllowedActions:   []string{},
	}

	if row.ReviewerID.Valid {
		claim.AssignedReviewer = &Member{ID: row.ReviewerID.UUID, Name: row.ReviewerName.String, Position: row.ReviewerPosition.String}
	}

	for _, item := range items {
		claim.Items = append(claim.Items, LineItem{
			LineNo:      int(item.LineNo),
			Category:    item.Category,
			Description: item.Description,
			ExpenseDate: item.ExpenseDate.Time,
			Amount:      item.AmountSatang,
			ReceiptRef:  item.ReceiptRef,
		})
	}

	for i := range history {
		entry := &history[i]

		claim.History = append(claim.History, HistoryEntry{
			ID:         entry.ID,
			Action:     entry.Action,
			FromStatus: optionalStatus(entry.FromStatus),
			ToStatus:   entry.ToStatus,
			Actor:      Member{ID: entry.ActorID, Name: entry.ActorName, Position: entry.ActorPosition},
			Comment:    optionalText(entry.Comment),
			CreatedAt:  entry.CreatedAt.Time,
		})
	}

	return claim, nil
}

// timestamptz wraps t as a non-null database timestamp.
func timestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// optionalTime returns nil for a NULL timestamp.
func optionalTime(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}

	return &t.Time
}

// optionalStatus returns nil for a NULL status.
func optionalStatus(status db.NullClaimStatus) *db.ClaimStatus {
	if !status.Valid {
		return nil
	}

	return &status.ClaimStatus
}

// optionalText returns nil for a NULL text.
func optionalText(text pgtype.Text) *string {
	if !text.Valid {
		return nil
	}

	return &text.String
}

// claimNumber builds CLM-<Bangkok year of now>-<seq, 6 digits> (SDS §2.2).
func claimNumber(now time.Time, seq int64) string {
	return fmt.Sprintf("CLM-%d-%06d", now.In(bangkok).Year(), seq)
}
