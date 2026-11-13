package claim

import (
	"context"
	"net/http"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/httperr"
)

var errUnauthenticated = &httperr.Error{
	Status: http.StatusUnauthorized,
	Code:   api.ProblemCodeUnauthenticated,
	Title:  "Authentication required.",
}

// Handler implements the claim operations of api.StrictServerInterface. It maps requests to Service calls and
// Service errors to problem responses.
type Handler struct {
	service Service
}

// NewHandler returns a Handler over service.
func NewHandler(service Service) Handler {
	return Handler{service: service}
}

// CreateClaim submits the claim of the request body for the session user.
func (h Handler) CreateClaim(ctx context.Context, request api.CreateClaimRequestObject) (api.CreateClaimResponseObject, error) {
	return nil, errUnauthenticated
}
