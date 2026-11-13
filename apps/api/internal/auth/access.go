package auth

import "github.com/beambeambeam/claimflow/apps/api/internal/api"

// permissionsByRole maps each role to its route permissions, per SDS §5.1 and F-01 §1.6.
// Code checks permissions, never role names.
var permissionsByRole = map[api.CurrentUserRole]map[string][]string{
	api.RoleRequester: {"claim": {"create", "list"}},
	api.RoleReviewer:  {"claim": {"create", "list"}, "review": {"list"}, "report": {"read"}},
	api.RoleFinance:   {"review": {"list"}, "finance": {"list"}, "report": {"read"}},
}
