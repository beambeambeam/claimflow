package report

import (
	"errors"
	"testing"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// The route requires a session before the handler runs, so a missing user is only a guard.
func TestGetDecisionSummaryWithoutSessionUserIsUnauthenticated(t *testing.T) {
	handler := NewHandler(NewService(pgtest.New(t)))

	_, err := handler.GetDecisionSummary(t.Context(), api.GetDecisionSummaryRequestObject{})
	if !errors.Is(err, errUnauthenticated) {
		t.Fatalf("GetDecisionSummary error = %v, want the 401 problem", err)
	}
}
