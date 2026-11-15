package main

import (
	"fmt"
	"regexp"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/auth"
)

// operationPermissions names the route permission (SDS §5.1) that an operation needs on top of a session.
// The key is the operation name that oapi-codegen passes to strict middleware: the operationId with its first
// letter in upper case. The check runs after the request is validated (F-01 §3.4, step 5). An operation not
// listed needs only a session.
var operationPermissions = map[string]auth.Permission{
	"CreateClaim": {Resource: "claim", Action: "create"},
}

// pathParam matches a path parameter in a contract path, such as {id}.
var pathParam = regexp.MustCompile(`\{([^}]+)\}`)

// contractSpec returns the embedded openapi.yaml.
func contractSpec() *openapi3.T {
	spec, err := api.GetSpec()
	if err != nil { // coverage-ignore
		panic(fmt.Sprintf("load embedded openapi.yaml: %v", err))
	}

	return spec
}

// contractPublicRoutes returns publicRoutes of the embedded openapi.yaml.
func contractPublicRoutes() map[string]bool {
	return publicRoutes(contractSpec())
}

// ginRoute returns the gin route ("METHOD /api/v1/path/:param") of a contract operation.
func ginRoute(method, path string) string {
	return method + " " + basePath + pathParam.ReplaceAllString(path, ":$1")
}

// publicRoutes returns the gin route ("METHOD /api/v1/path") of every contract operation that opts out of the
// session (F-01 §1.6, NFR-02). auth.RequireSession asks for a session on every other route, so an operation
// added to openapi.yaml, or a gin route added outside it, needs a session unless it opts out.
func publicRoutes(spec *openapi3.T) map[string]bool {
	routes := map[string]bool{}

	for path, item := range spec.Paths.Map() {
		for method, operation := range item.Operations() {
			if isPublic(spec, operation) {
				routes[ginRoute(method, path)] = true
			}
		}
	}

	return routes
}

// isPublic reports whether operation opts out of the session with an empty security list (`security: []`),
// its own or, when it has none, the document's.
func isPublic(spec *openapi3.T, operation *openapi3.Operation) bool {
	if operation.Security != nil {
		return len(*operation.Security) == 0
	}

	return spec.Security != nil && len(spec.Security) == 0
}
