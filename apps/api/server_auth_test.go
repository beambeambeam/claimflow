package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

// TestAuthOperationsAreNotImplementedYet covers the stubs from T1. T2 (login) and T3 (current user)
// replace them, and this test goes with them.
func TestAuthOperationsAreNotImplementedYet(t *testing.T) {
	router := newRouter(discardLogger(), testEnv(0), pgtest.New(t))

	login := httptest.NewRecorder()
	loginRequest := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"username":"req.anan","password":"secret"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(login, loginRequest)
	decodeProblem(t, login, http.StatusNotImplemented, "INTERNAL")

	decodeProblem(t, get(t, router, "/api/v1/auth/me"), http.StatusNotImplemented, "INTERNAL")
}
