package claim

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	// ErrInvalidTransition means the claim's status does not allow the action (BR-03).
	ErrInvalidTransition = errors.New("claim: invalid transition")
	// ErrConflict means the claim changed since the caller read it: the version is not the current one (BR-13).
	ErrConflict = errors.New("claim: changed since it was read")
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

	_, created.AllowedActions = Allowed(actor.ID, created.subject())

	return created, nil
}

// Get returns the claim with the id and the actions that the claim policy allows the actor. A missing claim is
// ErrNotFound. A claim that the actor may not see is ErrForbidden, and nothing of it is returned (F03-AC-06).
func (s Service) Get(ctx context.Context, actor Actor, id uuid.UUID) (Claim, error) {
	found, err := load(ctx, db.New(s.pool), id)
	if err != nil {
		return Claim{}, fmt.Errorf("get claim: %w", err)
	}

	visible, actions := Allowed(actor.ID, found.subject())
	if !visible {
		return Claim{}, ErrForbidden
	}

	found.AllowedActions = actions

	return found, nil
}

// ListMine returns one page, 1-based, of the actor's own claims in every status, newest first: created time
// descending, then id descending (F-03 §1.4). A page past the last one has no claims and the correct total.
func (s Service) ListMine(ctx context.Context, actor Actor, page, pageSize int) (Page, error) {
	queries := db.New(s.pool)

	total, err := queries.CountRequesterClaims(ctx, actor.ID)
	if err != nil { // coverage-ignore
		return Page{}, fmt.Errorf("count claims: %w", err)
	}

	return listPage(int(total), page, pageSize, func(offset, limit int32) ([]Summary, error) {
		rows, err := queries.ListRequesterClaims(ctx, db.ListRequesterClaimsParams{RequesterID: actor.ID, PageOffset: offset, PageLimit: limit})
		if err != nil { // coverage-ignore
			return nil, fmt.Errorf("list claims: %w", err)
		}

		summaries := make([]Summary, 0, len(rows))
		for i := range rows {
			summaries = append(summaries, summarize(&rows[i]))
		}

		return summaries, nil
	})
}

// ListReview returns one page, 1-based, of the SUBMITTED claims assigned to the actor, oldest submission first:
// submitted time ascending, then id ascending (F-04 §3). Decided claims leave the queue. A page past the last one has
// no claims and the correct total.
func (s Service) ListReview(ctx context.Context, actor Actor, page, pageSize int) (Page, error) {
	queries := db.New(s.pool)
	reviewer := uuid.NullUUID{UUID: actor.ID, Valid: true}

	total, err := queries.CountReviewClaims(ctx, reviewer)
	if err != nil { // coverage-ignore
		return Page{}, fmt.Errorf("count review claims: %w", err)
	}

	return listPage(int(total), page, pageSize, func(offset, limit int32) ([]Summary, error) {
		rows, err := queries.ListReviewClaims(ctx, db.ListReviewClaimsParams{ReviewerID: reviewer, PageOffset: offset, PageLimit: limit})
		if err != nil { // coverage-ignore
			return nil, fmt.Errorf("list review claims: %w", err)
		}

		summaries := make([]Summary, 0, len(rows))
		for i := range rows {
			summaries = append(summaries, summarize((*db.ListRequesterClaimsRow)(&rows[i])))
		}

		return summaries, nil
	})
}

// listPage builds one page from total, the number of claims on every page, and fetch, which reads a window of them.
// A page past the last one is not fetched.
func listPage(total, page, pageSize int, fetch func(offset, limit int32) ([]Summary, error)) (Page, error) {
	if total == 0 || page-1 > (total-1)/pageSize {
		return Page{Claims: []Summary{}, Total: total}, nil
	}

	claims, err := fetch(int32((page-1)*pageSize), int32(pageSize)) //nolint:gosec // below the total, which is an int32; the handler limits the size to 100
	if err != nil {                                                 // coverage-ignore
		return Page{}, err
	}

	return Page{Claims: claims, Total: total}, nil
}

// summarize maps a list row to a Summary.
func summarize(row *db.ListRequesterClaimsRow) Summary {
	return Summary{
		ID:          row.ID,
		ClaimNo:     row.ClaimNo,
		Status:      row.Status,
		Title:       row.Title,
		Total:       row.TotalSatang,
		Requester:   Member{ID: row.RequesterID, Name: row.RequesterName, Position: row.RequesterPosition},
		SubmittedAt: optionalTime(row.SubmittedAt),
		CreatedAt:   row.CreatedAt.Time,
	}
}

// Act takes a decision on a claim for the actor, in one transaction (F-04 §1.2): the claim policy allows the action, the
// comment is trimmed (empty is stored as no comment), the claim moves to the decided status with its version plus one
// and its decision time set, and one history entry is added. version is the one the caller read. It checks in the order of
// F-04 §1.4: a missing claim is ErrNotFound; a claim the actor may not decide is ErrForbidden; a status that does not
// allow the action is ErrInvalidTransition; a changed version is ErrConflict (BR-13). It returns the decided claim.
//
// A claim that is no longer SUBMITTED, read with a version other than the caller's, is ErrConflict: another decision
// has won since the caller read the claim (BR-13). The caller's screen is stale, so its action is not "invalid".
func (s Service) Act(ctx context.Context, actor Actor, id uuid.UUID, version int, action, comment string) (Claim, error) {
	now := s.now()

	var decided Claim

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		queries := db.New(tx)

		found, err := load(ctx, queries, id)
		if err != nil {
			return err
		}

		step, err := check(actor.ID, found.subject(), action)
		if errors.Is(err, ErrInvalidTransition) && version != found.Version {
			return ErrConflict
		}

		if err != nil {
			return err
		}

		changed, err := queries.DecideClaim(ctx, db.DecideClaimParams{
			Status:  step.to,
			Now:     timestamptz(now),
			ID:      id,
			Version: int32(version), //nolint:gosec // the handler reads a 31-bit version
		})
		if err != nil { // coverage-ignore
			return fmt.Errorf("decide claim: %w", err)
		}

		if changed == 0 {
			return ErrConflict
		}

		err = queries.InsertClaimHistory(ctx, db.InsertClaimHistoryParams{
			ClaimID:    id,
			ActorID:    actor.ID,
			Action:     step.history,
			FromStatus: db.NullClaimStatus{ClaimStatus: step.from, Valid: true},
			ToStatus:   step.to,
			Comment:    optionalComment(comment),
			Now:        timestamptz(now),
		})
		if err != nil {
			return fmt.Errorf("insert history: %w", err)
		}

		decided, err = load(ctx, queries, id)

		return err
	})
	if err != nil {
		return Claim{}, fmt.Errorf("decide claim: %w", err)
	}

	_, decided.AllowedActions = Allowed(actor.ID, decided.subject())

	return decided, nil
}

// optionalComment trims comment and stores an empty one as NULL.
func optionalComment(comment string) pgtype.Text {
	trimmed := strings.TrimSpace(comment)

	return pgtype.Text{String: trimmed, Valid: trimmed != ""}
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
