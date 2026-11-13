package main

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// publicOperations are the only operations that may answer without a session (F-01 §1.6, NFR-02).
var publicOperations = map[string]bool{
	"POST /auth/login": true,
	"GET /health":      true,
	"GET /ready":       true,
}

// F01-TEST-12: every operation in openapi.yaml except login, health, and ready answers 401 UNAUTHENTICATED
// with no cookie and with a random token. It reads the operations from the contract, so an operation that a
// later feature adds is covered without a new test.
func TestF01TEST12EveryOperationNeedsASession(t *testing.T) {
	spec, err := api.GetSpec()
	if err != nil {
		t.Fatalf("load contract: %v", err)
	}

	router := newRouter(discardLogger(), testEnv(0), pgtest.New(t))

	randomToken := make([]byte, 32)
	if _, err := rand.Read(randomToken); err != nil {
		t.Fatalf("random token: %v", err)
	}

	cookies := map[string]*http.Cookie{
		"no cookie": nil,
		"random token": {
			Name:     "sid",
			Value:    base64.RawURLEncoding.EncodeToString(randomToken),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		},
	}

	checked := 0
	for path, item := range spec.Paths.Map() {
		for method := range item.Operations() {
			if publicOperations[method+" "+path] {
				continue
			}
			checked++

			// Any value fills a path parameter: the session check runs before the request is validated.
			url := basePath + pathParam.ReplaceAllString(path, "019a5e2c-7b3d-7c41-9f2e-4b8d1a6c3e70")

			for name, cookie := range cookies {
				t.Run(method+" "+path+" with "+name, func(t *testing.T) {
					request := httptest.NewRequestWithContext(t.Context(), method, url, strings.NewReader("{}"))
					request.Header.Set("Content-Type", "application/json")
					if cookie != nil {
						request.AddCookie(cookie)
					}
					recorder := httptest.NewRecorder()
					router.ServeHTTP(recorder, request)

					decodeProblem(t, recorder, http.StatusUnauthorized, "UNAUTHENTICATED")
				})
			}
		}
	}

	if checked == 0 {
		t.Fatal("the contract has no operation that needs a session")
	}
}

// Every key of operationPermissions names an operation of the contract, so a typo cannot drop a check.
// oapi-codegen passes the operationId with its first letter in upper case.
func TestOperationPermissionsNameContractOperations(t *testing.T) {
	spec, err := api.GetSpec()
	if err != nil {
		t.Fatalf("load contract: %v", err)
	}

	names := map[string]bool{}
	for _, item := range spec.Paths.Map() {
		for _, operation := range item.Operations() {
			if operation.OperationID == "" {
				t.Fatalf("an operation in openapi.yaml has no operationId")
			}

			names[strings.ToUpper(operation.OperationID[:1])+operation.OperationID[1:]] = true
		}
	}

	for name := range operationPermissions {
		if !names[name] {
			t.Errorf("operationPermissions has %q, which is not an operation of openapi.yaml", name)
		}
	}
}
