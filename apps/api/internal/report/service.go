// Package report is the decision summary module (SDS §1, F-10). It reads claims and line items with its own queries
// and does not import internal/claim; depguard enforces that (F10-AC-07).
package report

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/money"
)

// Actor is the session user a summary is read for.
type Actor struct {
	ID   uuid.UUID
	Role string
}

// Range is the decision-date range of a summary as a half-open interval of instants: From is the first instant in
// the range and To the first instant after it. A nil bound is open.
type Range struct {
	From, To *time.Time
}

// Total is the approved total of one month and category.
type Total struct {
	Month    string
	Category db.ExpenseCategory
	Amount   money.Satang
}

// Summary is the decision summary of F-10 §1.4.
type Summary struct {
	CountsByStatus       map[db.ClaimStatus]int
	ApprovedTotals       []Total
	AverageDecisionHours *float64
}

// Service is the report module.
type Service struct {
	queries *db.Queries
}

// NewService returns a Service that reads through pool.
func NewService(pool db.DBTX) Service {
	return Service{queries: db.New(pool)}
}

// Summary returns the decision summary of the actor's scope for the decisions in r.
func (s Service) Summary(_ context.Context, _ Actor, _ Range) (Summary, error) {
	return Summary{}, nil
}
