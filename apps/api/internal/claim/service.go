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
	// ErrReturnLimit is an invalid transition for a return of a claim that was already returned MaxReturns times (BR-14).
	ErrReturnLimit = fmt.Errorf("%w: the return limit is reached", ErrInvalidTransition)
	// ErrConflict means the claim changed since the caller read it: the version is not the current one (BR-13).
	ErrConflict = errors.New("claim: changed since it was read")
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

// Create stores a claim and its line items in one transaction. When submit is true it also writes a SUBMIT history entry
// (F02-AC-01, F02-AC-02). The assigned reviewer is copied from the actor's reviewer at this moment and never changes
// afterwards (BR-17). An actor with no reviewer gets ErrForbidden (BR-15). The total is the sum of the items; the
// input has no total (BR-08). A draft has no reviewer or submission times.
func (s Service) Create(ctx context.Context, actor Actor, in Input, submit bool) (Claim, error) {
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

		var submittedAt time.Time
		if submit {
			submittedAt = now
		}
		if err := checkRules(in, now, submittedAt); err != nil {
			return err
		}

		id, err := insertClaim(ctx, queries, actor, member.ReviewerID, in, now, submit)
		if err != nil { // coverage-ignore
			return err
		}

		created, err = load(ctx, queries, id)

		return err
	})
	if err != nil {
		return Claim{}, fmt.Errorf("create claim: %w", err)
	}

	_, created.AllowedActions = Allowed(actor, created.subject())

	return created, nil
}

// Get returns the claim with the id and the actions that the claim policy allows the actor. A missing claim is
// ErrNotFound. A claim that the actor may not see is ErrForbidden, and nothing of it is returned (F03-AC-06).
func (s Service) Get(ctx context.Context, actor Actor, id uuid.UUID) (Claim, error) {
	found, err := load(ctx, db.New(s.pool), id)
	if err != nil {
		return Claim{}, fmt.Errorf("get claim: %w", err)
	}

	visible, actions := Allowed(actor, found.subject())
	if !visible {
		return Claim{}, ErrForbidden
	}

	found.AllowedActions = actions

	return found, nil
}

// List returns one page of the claims in q's scope (SDS §3.1). ScopeMine is the actor's own claims in every status,
// newest first: created time descending, then id descending (F-03 §1.4). ScopeReview is the SUBMITTED claims assigned
// to the actor, oldest submission first: submitted time ascending, then id ascending (F-04 §3); decided claims leave
// it. A page past the last one has no claims and the correct total.
func (s Service) List(ctx context.Context, actor Actor, q Query) (Page, error) {
	switch q.Scope {
	case ScopeMine:
		return s.listMine(ctx, actor, q)
	case ScopeReview:
		return s.listReview(ctx, actor, q)
	default:
		return Page{}, fmt.Errorf("claim: unknown scope %q", q.Scope)
	}
}

// listMine is List for ScopeMine.
func (s Service) listMine(ctx context.Context, actor Actor, q Query) (Page, error) {
	queries := db.New(s.pool)

	total, err := queries.CountRequesterClaims(ctx, actor.ID)
	if err != nil { // coverage-ignore
		return Page{}, fmt.Errorf("count claims: %w", err)
	}

	return listPage(int(total), q, func(offset, limit int32) ([]Summary, error) {
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

// listReview is List for ScopeReview.
func (s Service) listReview(ctx context.Context, actor Actor, q Query) (Page, error) {
	queries := db.New(s.pool)
	reviewer := uuid.NullUUID{UUID: actor.ID, Valid: true}

	total, err := queries.CountReviewClaims(ctx, reviewer)
	if err != nil { // coverage-ignore
		return Page{}, fmt.Errorf("count review claims: %w", err)
	}

	return listPage(int(total), q, func(offset, limit int32) ([]Summary, error) {
		rows, err := queries.ListReviewClaims(ctx, db.ListReviewClaimsParams{ReviewerID: reviewer, PageOffset: offset, PageLimit: limit})
		if err != nil { // coverage-ignore
			return nil, fmt.Errorf("list review claims: %w", err)
		}

		summaries := make([]Summary, 0, len(rows))
		for i := range rows {
			// Both queries select the same columns. Go converts between struct types only when their fields match,
			// so a change to either query that breaks this stops the build.
			summaries = append(summaries, summarize((*db.ListRequesterClaimsRow)(&rows[i])))
		}

		return summaries, nil
	})
}

// listPage builds the page of q from total, the number of claims on every page, and fetch, which reads a window of them.
// A page past the last one is not fetched.
func listPage(total int, q Query, fetch func(offset, limit int32) ([]Summary, error)) (Page, error) {
	if total == 0 || q.Page-1 > (total-1)/q.PageSize {
		return Page{Claims: []Summary{}, Total: total}, nil
	}

	claims, err := fetch(int32((q.Page-1)*q.PageSize), int32(q.PageSize)) //nolint:gosec // below the total, which is an int32; the handler limits the size to 100
	if err != nil {                                                       // coverage-ignore
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

// Act changes the claim's status and writes one history entry in the same transaction.
// Actor, status, and field checks run before the conditional update, as in F-05 §1.4.
func (s Service) Act(ctx context.Context, actor Actor, id uuid.UUID, version int, action Action, comment string) (Claim, error) {
	now := s.now()
	var acted Claim

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		queries := db.New(tx)
		row, err := findClaim(ctx, queries, id)
		if err != nil {
			return err
		}

		if err := checkVersionedAction(actor, claimSubject(&row), action, version, int(row.Version)); err != nil {
			return err
		}

		step, ok := transitions[action][row.Status]
		if !ok {
			return ErrInvalidTransition
		}

		if err := record(ctx, queries, actor, &row, version, action, step, comment, now); err != nil {
			return err
		}

		acted, err = load(ctx, queries, id)
		return err
	})
	if err != nil {
		return Claim{}, fmt.Errorf("act on claim: %w", err)
	}

	_, acted.AllowedActions = Allowed(actor, acted.subject())

	return acted, nil
}

// record updates the claim only at the requested version, then appends the history entry.
func record(ctx context.Context, queries *db.Queries, actor Actor, row *db.GetClaimRow, version int, action Action, step transition, comment string, now time.Time) error {
	var changed int64
	var err error
	if action == ActionSubmit {
		changed, err = submitClaim(ctx, queries, row, version, now)
		comment = ""
	} else {
		if err := checkComment(action, comment); err != nil {
			return err
		}
		if int(row.Version) != version {
			return ErrConflict
		}
		if action == ActionApprove {
			step.to = approveTarget(row.TotalSatang, actor.Role)
		}
		changed, err = reviewStep(ctx, queries, row, action, step, now)
	}
	if err != nil {
		return fmt.Errorf("update claim status: %w", err)
	}
	if changed == 0 {
		return ErrConflict
	}

	err = queries.InsertClaimHistory(ctx, db.InsertClaimHistoryParams{
		ClaimID:    row.ID,
		ActorID:    actor.ID,
		Action:     step.history,
		FromStatus: db.NullClaimStatus{ClaimStatus: row.Status, Valid: true},
		ToStatus:   step.to,
		Comment:    optionalComment(comment),
		Now:        timestamptz(now),
	})
	if err != nil {
		return fmt.Errorf("insert history: %w", err)
	}

	return nil
}

// reviewStep applies the step at the version that was read: a return keeps the claim open for its requester, a
// withdrawal ends it without a decision, an approval that goes to Finance waits for the final decision, and a decision
// stamps the decision time.
func reviewStep(ctx context.Context, queries *db.Queries, row *db.GetClaimRow, action Action, step transition, now time.Time) (int64, error) {
	if action == ActionWithdraw {
		changed, err := queries.WithdrawClaim(ctx, db.WithdrawClaimParams{Now: timestamptz(now), ID: row.ID, Version: row.Version})
		if err != nil { // coverage-ignore
			return 0, fmt.Errorf("withdraw claim: %w", err)
		}

		return changed, nil
	}

	if action == ActionReturn {
		changed, err := queries.ReturnClaim(ctx, db.ReturnClaimParams{Now: timestamptz(now), ID: row.ID, Version: row.Version})
		if err != nil { // coverage-ignore
			return 0, fmt.Errorf("return claim: %w", err)
		}

		return changed, nil
	}

	if step.to == db.ClaimStatusPENDINGFINANCE {
		changed, err := queries.SendClaimToFinance(ctx, db.SendClaimToFinanceParams{Now: timestamptz(now), ID: row.ID, Version: row.Version})
		if err != nil { // coverage-ignore
			return 0, fmt.Errorf("send claim to finance: %w", err)
		}

		return changed, nil
	}

	changed, err := queries.DecideClaim(ctx, db.DecideClaimParams{Status: step.to, Now: timestamptz(now), ID: row.ID, Version: row.Version})
	if err != nil { // coverage-ignore
		return 0, fmt.Errorf("decide claim: %w", err)
	}

	return changed, nil
}

func submitClaim(ctx context.Context, queries *db.Queries, row *db.GetClaimRow, version int, now time.Time) (int64, error) {
	items, err := queries.ListClaimLineItems(ctx, row.ID)
	if err != nil { // coverage-ignore
		return 0, fmt.Errorf("list items to submit: %w", err)
	}

	input := Input{Items: make([]LineItemInput, 0, len(items))}
	for _, item := range items {
		input.Items = append(input.Items, LineItemInput{
			Category: item.Category, Amount: item.AmountSatang, ExpenseDate: item.ExpenseDate.Time,
		})
	}

	firstSubmittedAt := now
	if row.Status == db.ClaimStatusRETURNED {
		firstSubmittedAt = row.FirstSubmittedAt.Time
	}
	if err := checkRules(input, now, firstSubmittedAt); err != nil {
		return 0, err
	}

	if int(row.Version) != version {
		return 0, ErrConflict
	}

	requestedVersion := row.Version
	if row.Status == db.ClaimStatusDRAFT {
		changed, err := queries.SubmitDraftClaim(ctx, db.SubmitDraftClaimParams{
			ID: row.ID, Version: requestedVersion, ReviewerID: row.OwnerReviewerID, Now: timestamptz(now),
		})
		if err != nil { // coverage-ignore
			return 0, fmt.Errorf("submit draft: %w", err)
		}
		return changed, nil
	}

	changed, err := queries.ResubmitClaim(ctx, db.ResubmitClaimParams{ID: row.ID, Version: requestedVersion, Now: timestamptz(now)})
	if err != nil { // coverage-ignore
		return 0, fmt.Errorf("resubmit claim: %w", err)
	}
	return changed, nil
}

// optionalComment trims comment and stores an empty one as NULL.
func optionalComment(comment string) pgtype.Text {
	trimmed := strings.TrimSpace(comment)

	return pgtype.Text{String: trimmed, Valid: trimmed != ""}
}

// insertClaim writes the new claim and its items, with history only for a direct submission.
func insertClaim(ctx context.Context, queries *db.Queries, actor Actor, reviewerID uuid.NullUUID, in Input, now time.Time, submit bool) (uuid.UUID, error) {
	seq, err := queries.NextClaimSequence(ctx)
	if err != nil { // coverage-ignore
		return uuid.Nil, fmt.Errorf("next claim number: %w", err)
	}

	id := uuid.Must(uuid.NewV7())
	params := db.InsertDraftClaimParams{
		ID: id, ClaimNo: claimNumber(now, seq), RequesterID: actor.ID,
		Title: in.Title, Purpose: in.Purpose, TotalSatang: inputTotal(in.Items), Now: timestamptz(now),
	}
	if submit {
		err = queries.InsertClaim(ctx, db.InsertClaimParams{
			ID: id, ClaimNo: params.ClaimNo, RequesterID: actor.ID, AssignedReviewerID: reviewerID,
			Title: in.Title, Purpose: in.Purpose, Status: db.ClaimStatusSUBMITTED, TotalSatang: params.TotalSatang,
			FirstSubmittedAt: params.Now, SubmittedAt: params.Now, Now: params.Now,
		})
	} else {
		err = queries.InsertDraftClaim(ctx, params)
	}
	if err != nil { // coverage-ignore
		return uuid.Nil, fmt.Errorf("insert claim: %w", err)
	}

	if err := insertItems(ctx, queries, id, in.Items); err != nil { // coverage-ignore
		return uuid.Nil, err
	}

	if submit {
		err = queries.InsertClaimHistory(ctx, db.InsertClaimHistoryParams{
			ClaimID: id, ActorID: actor.ID, Action: db.ClaimActionSUBMIT, ToStatus: db.ClaimStatusSUBMITTED, Now: params.Now,
		})
		if err != nil {
			return uuid.Nil, fmt.Errorf("insert history: %w", err)
		}
	}

	return id, nil
}

func inputTotal(items []LineItemInput) money.Satang {
	var total money.Satang
	for _, item := range items {
		total += item.Amount
	}
	return total
}

// insertItems stores the line items in request order.
func insertItems(ctx context.Context, queries *db.Queries, id uuid.UUID, items []LineItemInput) error {
	for i, item := range items {
		err := queries.InsertLineItem(ctx, db.InsertLineItemParams{
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
			return fmt.Errorf("insert line item %d: %w", i+1, err)
		}
	}
	return nil
}

// checkVersionedAction reports a stale version rather than an invalid transition when the status changed.
func checkVersionedAction(actor Actor, subject Subject, action Action, version, current int) error {
	err := check(actor, subject, action)
	if errors.Is(err, ErrInvalidTransition) && version != current {
		return ErrConflict
	}

	return err
}

func findClaim(ctx context.Context, queries *db.Queries, id uuid.UUID) (db.GetClaimRow, error) {
	row, err := queries.GetClaim(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.GetClaimRow{}, ErrNotFound
	}
	if err != nil { // coverage-ignore
		return db.GetClaimRow{}, fmt.Errorf("get claim: %w", err)
	}

	return row, nil
}

func claimSubject(row *db.GetClaimRow) Subject {
	return Subject{
		RequesterID: row.RequesterID, AssignedReviewerID: row.ReviewerID.UUID, Status: row.Status,
		OwnerHasReviewer: row.OwnerReviewerID.Valid, ReturnCount: int(row.ReturnCount), TotalSatang: row.TotalSatang,
	}
}

// load reads a claim with its line items and history.
func load(ctx context.Context, queries *db.Queries, id uuid.UUID) (Claim, error) {
	row, err := findClaim(ctx, queries, id)
	if err != nil {
		return Claim{}, err
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
		Items:            lineItems(items),
		History:          historyEntries(history),
		AllowedActions:   []Action{},
		ownerReviewerID:  row.OwnerReviewerID,
	}

	if row.ReviewerID.Valid {
		claim.AssignedReviewer = &Member{ID: row.ReviewerID.UUID, Name: row.ReviewerName.String, Position: row.ReviewerPosition.String}
	}

	return claim, nil
}

// lineItems maps the line item rows of a claim, in line order.
func lineItems(rows []db.ListClaimLineItemsRow) []LineItem {
	items := make([]LineItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, LineItem{
			LineNo:      int(row.LineNo),
			Category:    row.Category,
			Description: row.Description,
			ExpenseDate: row.ExpenseDate.Time,
			Amount:      row.AmountSatang,
			ReceiptRef:  row.ReceiptRef,
		})
	}

	return items
}

// historyEntries maps the history rows of a claim, oldest first.
func historyEntries(rows []db.ListClaimHistoryRow) []HistoryEntry {
	entries := make([]HistoryEntry, 0, len(rows))
	for i := range rows {
		row := &rows[i]

		entries = append(entries, HistoryEntry{
			ID:         row.ID,
			Action:     row.Action,
			FromStatus: optionalStatus(row.FromStatus),
			ToStatus:   row.ToStatus,
			Actor:      Member{ID: row.ActorID, Name: row.ActorName, Position: row.ActorPosition},
			Comment:    optionalText(row.Comment),
			CreatedAt:  row.CreatedAt.Time,
		})
	}

	return entries
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
