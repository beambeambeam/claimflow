package auth

import (
	"context"
	"fmt"
	"slices"

	"github.com/gin-gonic/gin"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/httperr"
)

// Permissions maps a resource to the actions granted on it.
type Permissions map[string][]string

// Permission is one action on one resource, such as claim:create.
type Permission struct {
	Resource string
	Action   string
}

// statement lists every resource and action that a route can require (SDS §5.1). Each role grants a subset.
var statement = Permissions{
	"claim":   {"create", "list"}, // create claims; list own claims
	"review":  {"list"},           // list the review queue
	"finance": {"list"},           // list the finance queue (Sprint 3)
	"report":  {"read"},           // decision summary (Sprint 3)
}

// roles maps each role to its route permissions, per SDS §5.1 and F-01 §1.6. Code checks permissions, never
// role names.
var roles = map[api.CurrentUserRole]Permissions{
	api.RoleRequester: {"claim": {"create", "list"}},
	api.RoleReviewer:  {"claim": {"create", "list"}, "review": {"list"}, "report": {"read"}},
	api.RoleFinance:   {"review": {"list"}, "finance": {"list"}, "report": {"read"}},
}

// Allows reports whether p grants action on resource.
func (p Permissions) Allows(resource, action string) bool {
	return slices.Contains(p[resource], action)
}

// PermissionsOf returns the route permissions of role. An unknown role has none.
func PermissionsOf(role string) Permissions {
	return roles[api.CurrentUserRole(role)]
}

// RequirePermission returns gin middleware that lets the request through only when the session user's role
// grants action on resource. It answers 403 FORBIDDEN otherwise, and 401 when no user is signed in, without
// calling the handler. A permission outside the statement panics, so a typo fails when the router is built.
func RequirePermission(resource, action string) gin.HandlerFunc {
	mustBeInStatement(Permission{Resource: resource, Action: action})

	return func(c *gin.Context) {
		if err := authorize(c.Request.Context(), resource, action); err != nil {
			httperr.Report(c, err)
			c.Abort()

			return
		}

		c.Next()
	}
}

// OperationPermissions returns strict middleware that applies RequirePermission's check to each operation
// in byOperation, keyed by the operation name the generated strict server passes. It runs after the request
// is decoded and validated (F-01 §3.4, step 5). Operations not in byOperation pass through unchanged.
func OperationPermissions(byOperation map[string]Permission) api.StrictMiddlewareFunc {
	for _, permission := range byOperation {
		mustBeInStatement(permission)
	}

	return func(next api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		permission, ok := byOperation[operationID]
		if !ok {
			return next
		}

		return func(c *gin.Context, request any) (any, error) {
			if err := authorize(c.Request.Context(), permission.Resource, permission.Action); err != nil {
				return nil, err
			}

			return next(c, request)
		}
	}
}

// authorize returns nil when the user in ctx may take action on resource.
func authorize(ctx context.Context, resource, action string) error {
	user, ok := UserFrom(ctx)
	if !ok {
		return errUnauthenticated
	}

	if !PermissionsOf(user.Role).Allows(resource, action) {
		return httperr.ErrForbidden
	}

	return nil
}

// mustBeInStatement panics when permission is not in the statement.
func mustBeInStatement(permission Permission) {
	if !statement.Allows(permission.Resource, permission.Action) {
		panic(fmt.Sprintf("auth: %s:%s is not in the permission statement", permission.Resource, permission.Action))
	}
}
