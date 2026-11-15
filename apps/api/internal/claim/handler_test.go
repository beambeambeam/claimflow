package claim

import (
	"errors"
	"testing"
	"time"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/db"
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

// The route requires a session before the handler runs, so a missing user is only a guard.
func TestGetClaimWithoutSessionUserIsUnauthenticated(t *testing.T) {
	handler := NewHandler(newService(pgtest.New(t)))

	_, err := handler.GetClaim(t.Context(), api.GetClaimRequestObject{})
	if !errors.Is(err, errUnauthenticated) {
		t.Fatalf("GetClaim error = %v, want the 401 problem", err)
	}
}

// The route requires a session before the handler runs, so a missing user is only a guard.
func TestDecisionsWithoutSessionUserAreUnauthenticated(t *testing.T) {
	handler := NewHandler(newService(pgtest.New(t)))
	ifMatch := `"0"`

	_, err := handler.ApproveClaim(t.Context(), api.ApproveClaimRequestObject{Params: api.ApproveClaimParams{IfMatch: &ifMatch}})
	if !errors.Is(err, errUnauthenticated) {
		t.Fatalf("ApproveClaim error = %v, want the 401 problem", err)
	}

	_, err = handler.RejectClaim(t.Context(), api.RejectClaimRequestObject{Params: api.RejectClaimParams{IfMatch: &ifMatch}})
	if !errors.Is(err, errUnauthenticated) {
		t.Fatalf("RejectClaim error = %v, want the 401 problem", err)
	}
}
