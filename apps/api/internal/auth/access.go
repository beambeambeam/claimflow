package auth

import (
	"slices"

	"github.com/gin-gonic/gin"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
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

// RequirePermission is a stub.
func RequirePermission(_, _ string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

// OperationPermissions is a stub.
func OperationPermissions(_ map[string]Permission) api.StrictMiddlewareFunc {
	return func(next api.StrictHandlerFunc, _ string) api.StrictHandlerFunc {
		return next
	}
}
