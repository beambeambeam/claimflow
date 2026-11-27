// Package report is the decision summary module (SDS §1, F-10). It reads claims and line items with its own queries
// and does not import internal/claim; depguard enforces that (F10-AC-07).
package report

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/money"
)

// roleFinance is the role whose summary covers every claim that is not a draft (F-10 §1.3).
const roleFinance = "finance"

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
func (s Service) Summary(ctx context.Context, actor Actor, r Range) (Summary, error) {
	allClaims := actor.Role == roleFinance
	actorID := uuid.NullUUID{UUID: actor.ID, Valid: true}

	rows, err := s.queries.CountClaimsByStatus(ctx, db.CountClaimsByStatusParams{AllClaims: allClaims, ActorID: actorID})
	if err != nil { // coverage-ignore
		return Summary{}, fmt.Errorf("count claims by status: %w", err)
	}

	counts := map[db.ClaimStatus]int{
		db.ClaimStatusSUBMITTED:      0,
		db.ClaimStatusRETURNED:       0,
		db.ClaimStatusPENDINGFINANCE: 0,
		db.ClaimStatusAPPROVED:       0,
		db.ClaimStatusREJECTED:       0,
		db.ClaimStatusWITHDRAWN:      0,
	}
	for _, row := range rows {
		counts[row.Status] = int(row.Total)
	}

	grouped, err := s.queries.ApprovedTotalsByMonthCategory(ctx, db.ApprovedTotalsByMonthCategoryParams{
		AllClaims: allClaims, ActorID: actorID, FromAt: timestamp(r.From), ToAt: timestamp(r.To),
	})
	if err != nil { // coverage-ignore
		return Summary{}, fmt.Errorf("approved totals by month and category: %w", err)
	}

	totals := make([]Total, 0, len(grouped))
	for _, row := range grouped {
		totals = append(totals, Total{Month: row.Month, Category: row.Category, Amount: money.Satang(row.Total)})
	}

	average, err := s.queries.AverageDecisionHours(ctx, db.AverageDecisionHoursParams{
		AllClaims: allClaims, ActorID: actorID, FromAt: timestamp(r.From), ToAt: timestamp(r.To),
	})
	if err != nil { // coverage-ignore
		return Summary{}, fmt.Errorf("average decision hours: %w", err)
	}

	summary := Summary{CountsByStatus: counts, ApprovedTotals: totals}
	if average.Decided > 0 {
		summary.AverageDecisionHours = &average.Hours
	}

	return summary, nil
}

// timestamp is a range bound as a query argument. A nil bound is NULL, which leaves that side of the range open.
func timestamp(bound *time.Time) pgtype.Timestamptz {
	if bound == nil {
		return pgtype.Timestamptz{}
	}

	return pgtype.Timestamptz{Time: *bound, Valid: true}
}
