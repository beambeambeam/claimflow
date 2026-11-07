// Package system implements the liveness and readiness operations (SDS §4.3).
package system

import (
	"context"
	"fmt"
	"time"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/httperr"
)

// readyTimeout bounds the database check, so a probe gets an answer while the database hangs.
const readyTimeout = 2 * time.Second

// Handler implements the system operations of api.StrictServerInterface.
type Handler struct {
	version string
	queries *db.Queries
}

// NewHandler returns a Handler that reports version and checks the database through queries.
func NewHandler(version string, queries *db.Queries) Handler {
	return Handler{version: version, queries: queries}
}

// GetHealth reports that the process runs. It never uses the database.
func (h Handler) GetHealth(context.Context, api.GetHealthRequestObject) (api.GetHealthResponseObject, error) {
	return api.GetHealth200JSONResponse{Status: api.HealthStatusOk, Version: h.version}, nil
}

// GetReady reports whether the database answers a query.
func (h Handler) GetReady(ctx context.Context, _ api.GetReadyRequestObject) (api.GetReadyResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, readyTimeout)
	defer cancel()

	if _, err := h.queries.CheckReady(ctx); err != nil {
		return nil, fmt.Errorf("check database: %w: %w", httperr.ErrUnavailable, err)
	}

	return api.GetReady200JSONResponse{Status: api.ReadyStatusOk}, nil
}
