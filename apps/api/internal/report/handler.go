package report

import (
	"context"
	"fmt"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/auth"
	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/httperr"
)

// Handler implements the report operations of api.StrictServerInterface.
type Handler struct {
	service Service
}

// NewHandler returns a Handler over service.
func NewHandler(service Service) Handler {
	return Handler{service: service}
}

// GetDecisionSummary returns the decision summary of the session user's scope. The route permission report:read has
// already been checked. A date that is not YYYY-MM-DD, or from after to, is a 400 that names each bad parameter.
func (h Handler) GetDecisionSummary(ctx context.Context, request api.GetDecisionSummaryRequestObject) (api.GetDecisionSummaryResponseObject, error) {
	user, ok := auth.UserFrom(ctx)
	if !ok { // The route requires a session before the handler runs, so a missing user is only a guard.
		return nil, httperr.ErrUnauthenticated
	}

	rng, problems := ParseRange(request.Params.From, request.Params.To)
	if len(problems) > 0 {
		return nil, fmt.Errorf("summary range: %w", httperr.InvalidFields(problems))
	}

	summary, err := h.service.Summary(ctx, Actor{ID: user.ID, Role: user.Role}, rng)
	if err != nil { // coverage-ignore
		return nil, err
	}

	return api.GetDecisionSummary200JSONResponse{Data: body(&summary)}, nil
}

// body maps a Summary to the contract's Summary.
func body(summary *Summary) api.Summary {
	counts := summary.CountsByStatus

	totals := make([]api.ApprovedTotal, 0, len(summary.ApprovedTotals))
	for _, total := range summary.ApprovedTotals {
		totals = append(totals, api.ApprovedTotal{Month: total.Month, Category: api.ExpenseCategory(total.Category), Total: total.Amount})
	}

	return api.Summary{
		CountsByStatus: api.StatusCounts{
			SUBMITTED:      counts[db.ClaimStatusSUBMITTED],
			RETURNED:       counts[db.ClaimStatusRETURNED],
			PENDINGFINANCE: counts[db.ClaimStatusPENDINGFINANCE],
			APPROVED:       counts[db.ClaimStatusAPPROVED],
			REJECTED:       counts[db.ClaimStatusREJECTED],
			WITHDRAWN:      counts[db.ClaimStatusWITHDRAWN],
		},
		ApprovedTotals:       totals,
		AverageDecisionHours: summary.AverageDecisionHours,
	}
}
