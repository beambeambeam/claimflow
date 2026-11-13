package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
)

// RequireSession answers 401 for an anonymous request to a route outside public, without calling the handler.
// It fails closed: a matched route missing from public, such as one added outside the contract or one whose
// key drifts from the gin route, needs a session. It lets a signed-in user through and leaves public routes alone.
func TestRequireSessionFailsClosedOutsidePublicRoutes(t *testing.T) {
	calls := 0
	status, code := serveTest(t, testRouter("", RequireSession(map[string]bool{}), &calls))
	if status != http.StatusUnauthorized || code != "UNAUTHENTICATED" || calls != 0 {
		t.Fatalf("anonymous: status = %d, code = %q, handler calls = %d; want 401 UNAUTHENTICATED and 0 calls", status, code, calls)
	}

	status, code = serveTest(t, testRouter("", RequireSession(map[string]bool{"GET /drifted": true}), &calls))
	if status != http.StatusUnauthorized || code != "UNAUTHENTICATED" || calls != 0 {
		t.Fatalf("drifted key: status = %d, code = %q, handler calls = %d; want 401 UNAUTHENTICATED and 0 calls", status, code, calls)
	}

	status, _ = serveTest(t, testRouter("requester", RequireSession(map[string]bool{}), &calls))
	if status != http.StatusNoContent || calls != 1 {
		t.Fatalf("signed in: status = %d, handler calls = %d; want 204 and 1 call", status, calls)
	}

	status, _ = serveTest(t, testRouter("", RequireSession(map[string]bool{"GET /test": true}), &calls))
	if status != http.StatusNoContent || calls != 2 {
		t.Fatalf("public route: status = %d, handler calls = %d; want 204 and 2 calls", status, calls)
	}
}

// RequireSession leaves a request that matches no route to NoRoute, so it gets 404, not 401.
func TestRequireSessionLeavesUnmatchedRequestsToNoRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequireSession(map[string]bool{}))
	router.NoRoute(func(c *gin.Context) { c.Status(http.StatusNotFound) })

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/missing", http.NoBody))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
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
