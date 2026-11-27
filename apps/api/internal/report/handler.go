package report

import (
	"context"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
)

// Handler implements the report operations of api.StrictServerInterface.
type Handler struct {
	service Service
}

// NewHandler returns a Handler over service.
func NewHandler(service Service) Handler {
	return Handler{service: service}
}

// GetDecisionSummary returns the decision summary of the session user's scope.
func (h Handler) GetDecisionSummary(_ context.Context, _ api.GetDecisionSummaryRequestObject) (api.GetDecisionSummaryResponseObject, error) {
	return api.GetDecisionSummary200JSONResponse{}, nil
}
