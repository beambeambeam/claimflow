package auth

import (
	"context"
	"encoding/json"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/httperr"
)

// asUser returns gin middleware that signs the request in as a user with role, as Session does for a valid
// cookie. An empty role leaves the request anonymous.
func asUser(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if role != "" {
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), userKey{}, User{Role: role}))
		}

		c.Next()
	}
}

// testRouter serves GET /test as role, behind guard. It counts the handler's calls in calls.
func testRouter(role string, guard gin.HandlerFunc, calls *int) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(httperr.RequestID(), httperr.Handle(slog.New(slog.DiscardHandler)), asUser(role))
	router.GET("/test", guard, func(c *gin.Context) {
		*calls++
		c.Status(http.StatusNoContent)
	})

	return router
}

// serveTest sends GET /test and returns the status and the problem code, if any.
func serveTest(t *testing.T, router http.Handler) (status int, code string) {
	t.Helper()

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody))

	var body struct {
		Code string `json:"code"`
	}
	if recorder.Body.Len() > 0 {
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
	}

	return recorder.Code, body.Code
}

// F01-TEST-13: a route behind RequirePermission("review", "list") refuses a requester with 403 FORBIDDEN
// without calling the handler, and runs the handler for a reviewer.
func TestF01TEST13RequirePermissionForbidsRolesWithoutIt(t *testing.T) {
	calls := 0

	status, code := serveTest(t, testRouter("requester", RequirePermission("review", "list"), &calls))
	if status != http.StatusForbidden || code != "FORBIDDEN" || calls != 0 {
		t.Fatalf("requester: status = %d, code = %q, handler calls = %d; want 403 FORBIDDEN and 0 calls", status, code, calls)
	}

	status, _ = serveTest(t, testRouter("reviewer", RequirePermission("review", "list"), &calls))
	if status != http.StatusNoContent || calls != 1 {
		t.Fatalf("reviewer: status = %d, handler calls = %d; want 204 and 1 call", status, calls)
	}
}

// RequirePermission answers 401, not 403, when no user is signed in.
func TestRequirePermissionNeedsAUser(t *testing.T) {
	calls := 0

	status, code := serveTest(t, testRouter("", RequirePermission("review", "list"), &calls))
	if status != http.StatusUnauthorized || code != "UNAUTHENTICATED" || calls != 0 {
		t.Fatalf("status = %d, code = %q, handler calls = %d; want 401 UNAUTHENTICATED and 0 calls", status, code, calls)
	}
}

// A permission outside the statement is a programming error, so it fails when the router is built.
func TestRequirePermissionPanicsOnUnknownPermission(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("RequirePermission(\"claim\", \"approve\") did not panic")
		}
	}()

	_ = RequirePermission("claim", "approve")
}

// Every role grants only permissions from the statement (SDS §5.1), and PermissionsOf returns the F-01 §1.6 map.
func TestRolesGrantSubsetsOfTheStatement(t *testing.T) {
	for role, permissions := range roles {
		for resource, actions := range permissions {
			for _, action := range actions {
				if !statement.Allows(resource, action) {
					t.Errorf("%s grants %s:%s, which the statement does not list", role, resource, action)
				}
			}
		}
	}

	want := Permissions{"review": {"list"}, "finance": {"list"}, "report": {"read"}}
	if got := PermissionsOf(string(api.RoleFinance)); !maps.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("PermissionsOf(finance) = %v, want %v", got, want)
	}
}

// operationRouter serves GET /test as role through a strict handler named "TestOperation", wrapped in
// OperationPermissions(byOperation).
func operationRouter(role string, byOperation map[string]Permission, calls *int) *gin.Engine {
	strict := OperationPermissions(byOperation)(func(*gin.Context, any) (any, error) {
		*calls++

		return nil, nil
	}, "TestOperation")

	return testRouter(role, func(c *gin.Context) {
		if _, err := strict(c, nil); err != nil {
			httperr.Report(c, err)
			c.Abort()
		}
	}, new(int))
}

// OperationPermissions checks the permission mapped to an operation, after the request decoded, and leaves
// an unmapped operation alone.
func TestOperationPermissionsChecksTheMappedPermission(t *testing.T) {
	needsReview := map[string]Permission{"TestOperation": {Resource: "review", Action: "list"}}

	calls := 0
	status, code := serveTest(t, operationRouter("requester", needsReview, &calls))
	if status != http.StatusForbidden || code != "FORBIDDEN" || calls != 0 {
		t.Fatalf("requester: status = %d, code = %q, operation calls = %d; want 403 FORBIDDEN and 0 calls", status, code, calls)
	}

	calls = 0
	_, _ = serveTest(t, operationRouter("reviewer", needsReview, &calls))
	if calls != 1 {
		t.Fatalf("reviewer: operation calls = %d, want 1", calls)
	}

	calls = 0
	_, _ = serveTest(t, operationRouter("requester", map[string]Permission{}, &calls))
	if calls != 1 {
		t.Fatalf("unmapped operation: operation calls = %d, want 1", calls)
	}
}

// OperationPermissions refuses a map entry outside the statement when the router is built.
func TestOperationPermissionsPanicsOnUnknownPermission(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("OperationPermissions with claim:approve did not panic")
		}
	}()

	_ = OperationPermissions(map[string]Permission{"ApproveClaim": {Resource: "claim", Action: "approve"}})
}
