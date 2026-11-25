// Package claim implements claims: submitting them today, and reading, reviewing, and finance decisions in later features (F-02 onward).
// It holds the claim business rules and knows nothing about HTTP (SDS §1.3).
package claim

import (
	"time"

	"github.com/google/uuid"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/money"
)

// RoleFinance is the role of a Finance Officer, the only role that the claim policy reads (F-08 §1.8).
const RoleFinance = "finance"

// Actor is the member who makes a call: the id and role of the session user. The module reads everything else, such as
// the assigned reviewer, from the database.
type Actor struct {
	ID   uuid.UUID
	Role string
}

// LineItemInput is one line item of a new claim. ExpenseDate is a calendar date.
type LineItemInput struct {
	Category    db.ExpenseCategory
	Description string
	ExpenseDate time.Time
	Amount      money.Satang
	ReceiptRef  string
}

// Input is the content of a new claim. It has no total: the module calculates it (BR-08).
type Input struct {
	Title   string
	Purpose string
	Items   []LineItemInput
}

// Member is a requester, assigned reviewer, or history actor, as the claim shows them.
type Member struct {
	ID       uuid.UUID
	Name     string
	Position string
}

// LineItem is a stored line item.
type LineItem struct {
	LineNo      int
	Category    db.ExpenseCategory
	Description string
	ExpenseDate time.Time
	Amount      money.Satang
	ReceiptRef  string
}

// HistoryEntry is one row of the append-only claim history (BR-11). FromStatus is nil for a claim submitted directly.
type HistoryEntry struct {
	ID         int64
	Action     db.ClaimAction
	FromStatus *db.ClaimStatus
	ToStatus   db.ClaimStatus
	Actor      Member
	Comment    *string
	CreatedAt  time.Time
}

// Summary is a claim as a list shows it.
type Summary struct {
	ID          uuid.UUID
	ClaimNo     string
	Status      db.ClaimStatus
	Title       string
	Total       money.Satang
	Requester   Member
	SubmittedAt *time.Time
	CreatedAt   time.Time
}

// Scope is which claims List reads.
type Scope string

// The scopes of List.
const (
	// ScopeMine is the actor's own claims in every status, newest first (F-03 §1.4).
	ScopeMine Scope = "mine"
	// ScopeReview is the SUBMITTED claims assigned to the actor, oldest submission first (F-04 §3).
	ScopeReview Scope = "review"
)

// Query selects one page of a list: the scope, the page number, 1-based, and the page size. F-09 adds filters here.
type Query struct {
	Scope    Scope
	Page     int
	PageSize int
}

// Page is one page of a list. Total counts the claims on every page.
type Page struct {
	Claims []Summary
	Total  int
}

// Claim is a stored claim with its line items and history. AllowedActions comes from the claim policy (Allowed).
type Claim struct {
	ID               uuid.UUID
	ClaimNo          string
	Status           db.ClaimStatus
	Title            string
	Purpose          string
	Total            money.Satang
	ReturnCount      int
	Version          int
	Requester        Member
	AssignedReviewer *Member
	FirstSubmittedAt *time.Time
	SubmittedAt      *time.Time
	DecidedAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Items            []LineItem
	History          []HistoryEntry
	AllowedActions   []Action
	ownerReviewerID  uuid.NullUUID
}

// subject returns what the claim policy reads about the claim.
func (c *Claim) subject() Subject {
	subject := Subject{
		RequesterID: c.Requester.ID, Status: c.Status, OwnerHasReviewer: c.ownerReviewerID.Valid, ReturnCount: c.ReturnCount,
		TotalSatang: c.Total,
	}
	if c.AssignedReviewer != nil {
		subject.AssignedReviewerID = c.AssignedReviewer.ID
	}

	return subject
}
