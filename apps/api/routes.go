package main

import (
	"github.com/beambeambeam/claimflow/apps/api/internal/auth"
)

// operationPermissions names the route permission (SDS §5.1) that an operation needs on top of a session.
// The key is the operation name that oapi-codegen passes to strict middleware: the operationId with its first
// letter in upper case. The check runs after the request is validated (F-01 §3.4, step 5). An operation not
// listed needs only a session. For example, F-02 adds "CreateClaim": {Resource: "claim", Action: "create"}.
var operationPermissions = map[string]auth.Permission{}
