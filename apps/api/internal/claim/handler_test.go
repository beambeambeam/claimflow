package claim

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/httperr"
	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// The route requires a session before the handler runs, so a missing user is only a guard.
func TestCreateClaimWithoutSessionUserIsUnauthenticated(t *testing.T) {
	handler := NewHandler(newService(pgtest.New(t)))

	_, err := handler.CreateClaim(t.Context(), api.CreateClaimRequestObject{Body: &api.CreateClaimJSONRequestBody{}})
	if !errors.Is(err, errUnauthenticated) {
		t.Fatalf("CreateClaim error = %v, want the 401 problem", err)
	}
}

// A service failure other than BR-15 is not a problem of its own: it passes up and becomes a 500.
func TestCreateClaimPassesUpUnexpectedErrors(t *testing.T) {
	pool := pgtest.New(t)
	pool.Close()

	_, err := newService(pool).Create(t.Context(), Actor{ID: uuid.Must(uuid.NewV7())}, validInput(), true)
	if err == nil {
		t.Fatal("Create on a closed pool succeeded")
	}

	var problem *httperr.Error
	if errors.As(err, &problem) {
		t.Fatalf("error = %v, want one with no problem response", err)
	}
}

// claimBody keeps what the first sprint never writes: a decision time, a from-status, and a comment.
func TestClaimBodyMapsOptionalFields(t *testing.T) {
	decided := time.Date(2026, time.November, 6, 3, 0, 0, 0, time.UTC)
	from := db.ClaimStatus("SUBMITTED")
	comment := "No receipt"

	body := claimBody(&Claim{
		DecidedAt: &decided,
		History: []HistoryEntry{{
			Action:     db.ClaimAction("REJECT"),
			FromStatus: &from,
			ToStatus:   db.ClaimStatus("REJECTED"),
			Comment:    &comment,
		}},
	})

	if body.AssignedReviewer != nil || body.SubmittedAt != nil || body.DecidedAt == nil || !body.DecidedAt.Equal(decided) {
		t.Fatalf("body = %+v, want no reviewer or submission time and the decision time", body)
	}

	if len(body.History) != 1 || body.History[0].FromStatus == nil || *body.History[0].FromStatus != api.ClaimStatusSubmitted ||
		body.History[0].Comment == nil || *body.History[0].Comment != comment {
		t.Fatalf("history = %+v, want the from-status and the comment", body.History)
	}
}
