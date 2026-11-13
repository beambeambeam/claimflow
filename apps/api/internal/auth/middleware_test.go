package auth

import (
	"errors"
	"net/http"
	"testing"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
)

// RequireSession answers 401 for an anonymous request to a protected route, without calling the handler. It
// lets a signed-in user through, and leaves routes outside protected, such as login, alone.
func TestRequireSessionGuardsProtectedRoutes(t *testing.T) {
	protected := map[string]bool{"GET /test": true}

	calls := 0
	status, code := serveTest(t, testRouter("", RequireSession(protected), &calls))
	if status != http.StatusUnauthorized || code != "UNAUTHENTICATED" || calls != 0 {
		t.Fatalf("anonymous: status = %d, code = %q, handler calls = %d; want 401 UNAUTHENTICATED and 0 calls", status, code, calls)
	}

	status, _ = serveTest(t, testRouter("requester", RequireSession(protected), &calls))
	if status != http.StatusNoContent || calls != 1 {
		t.Fatalf("signed in: status = %d, handler calls = %d; want 204 and 1 call", status, calls)
	}

	status, _ = serveTest(t, testRouter("", RequireSession(map[string]bool{}), &calls))
	if status != http.StatusNoContent || calls != 2 {
		t.Fatalf("public route: status = %d, handler calls = %d; want 204 and 2 calls", status, calls)
	}
}

// GetMe still answers 401 if it is ever reached without a user.
func TestGetMeWithoutUserIsUnauthenticated(t *testing.T) {
	_, err := Handler{}.GetMe(t.Context(), api.GetMeRequestObject{})
	if !errors.Is(err, errUnauthenticated) {
		t.Fatalf("err = %v, want errUnauthenticated", err)
	}
}

// Logout still answers 401 if it is ever reached without a session.
func TestLogoutWithoutSessionIsUnauthenticated(t *testing.T) {
	_, err := Handler{}.Logout(t.Context(), api.LogoutRequestObject{})
	if !errors.Is(err, errUnauthenticated) {
		t.Fatalf("err = %v, want errUnauthenticated", err)
	}
}
