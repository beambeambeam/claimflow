package main

import (
	"testing"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// withdrawMessage is the refusal of a withdrawal from a status that does not allow it (F-07 §1.5).
const withdrawMessage = "Only a submitted or returned claim can be withdrawn."

// F07-TEST-03 (F07-AC-03, BR-16): an owner's DRAFT, APPROVED, REJECTED, or WITHDRAWN claim cannot be withdrawn, with the
// current version: 409 INVALID_TRANSITION for BR-16 with the message of F-07 §1.5, and the claim is unchanged.
func TestF07TEST03OnlyASubmittedOrReturnedClaimCanBeWithdrawnBR16(t *testing.T) {
	pool := pgtest.New(t)
	router := seededClaimRouter(t, pool)
	token := loginAs(t, router, "req.anan")

	for _, status := range []string{"DRAFT", "APPROVED", "REJECTED", "WITHDRAWN"} {
		t.Run(status, func(t *testing.T) {
			id := storedClaim(t, pool, "req.anan", status, claimsNow).String()
			before := readState(t, pool, id)

			expectTransition(t, decide(t, router, token, id, "withdraw", `"0"`, ""), "BR-16", withdrawMessage)

			if after := readState(t, pool, id); after != before {
				t.Fatalf("stored = %+v after the refused withdrawal, want %+v", after, before)
			}
		})
	}
}
