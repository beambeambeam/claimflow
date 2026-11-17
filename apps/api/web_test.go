package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beambeambeam/claimflow/apps/api/env"
)

const (
	testIndex = "<!doctype html><title>ClaimFlow</title>"
	testAsset = "console.log('app')"
)

// routerWithoutDB is the API router for cfg. Its pool never connects: the web routes need no database.
func routerWithoutDB(t *testing.T, cfg *env.Env) http.Handler {
	t.Helper()

	pool, err := pgxpool.New(t.Context(), cfg.DatabaseUrl)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	return newRouter(discardLogger(), cfg, pool)
}

// webRouter is the API router with WEB_DIR set to a web build of index.html, one hashed asset, and one
// file at the root. A file beside the build must never be served.
func webRouter(t *testing.T) http.Handler {
	t.Helper()

	root := t.TempDir()
	dist := filepath.Join(root, "dist")

	for name, body := range map[string]string{
		"dist/index.html":             testIndex,
		"dist/assets/index-Bx81kQ.js": testAsset,
		"dist/favicon.svg":            "<svg/>",
		"secret.txt":                  "outside the build",
	} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	cfg := testEnv(0)
	cfg.WebDir = &dist

	return routerWithoutDB(t, cfg)
}

// SD-15: the API serves the web build outside /api, with index.html for client routes.
func TestWebBuildIsServedOutsideAPI(t *testing.T) {
	router := webRouter(t)

	for _, tc := range []struct{ name, path, body, cache string }{
		{"root", "/", testIndex, "no-cache"},
		{"client route", "/claims/0193d0c2-7a3e-7000-8000-000000000001", testIndex, "no-cache"},
		{"client route with a query", "/login?redirect=%2Fclaims", testIndex, "no-cache"},
		{"hashed asset", "/assets/index-Bx81kQ.js", testAsset, "public, max-age=31536000, immutable"},
		{"file at the root", "/favicon.svg", "<svg/>", ""},
		{"path outside the build", "/../secret.txt", testIndex, "no-cache"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := get(t, router, tc.path)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body)
			}

			if recorder.Body.String() != tc.body {
				t.Fatalf("body = %q, want %q", recorder.Body, tc.body)
			}

			if got := recorder.Header().Get("Cache-Control"); got != tc.cache {
				t.Fatalf("Cache-Control = %q, want %q", got, tc.cache)
			}
		})
	}

	head := request(t, router, http.MethodHead, "/")
	if head.Code != http.StatusOK || head.Body.Len() != 0 {
		t.Fatalf("HEAD / = %d with %d body bytes, want 200 and none", head.Code, head.Body.Len())
	}

	if got := head.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Fatalf("HEAD / Content-Type = %q, want text/html", got)
	}
}

// SD-15: unknown API routes, missing assets, and other methods stay JSON 404 problems, never index.html.
func TestWebBuildLeavesNotFoundAsProblems(t *testing.T) {
	router := webRouter(t)

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api"},
		{http.MethodGet, "/api/v1/nope"},
		{http.MethodGet, "/assets/missing-Zz99.js"},
		{http.MethodPost, "/claims"},
	} {
		recorder := request(t, router, tc.method, tc.path)

		body := decodeProblem(t, recorder, http.StatusNotFound, "NOT_FOUND")
		if body.Instance != tc.path {
			t.Fatalf("%s %s: instance = %q", tc.method, tc.path, body.Instance)
		}
	}
}

// A build without index.html is a server error, not a page that never loads.
func TestWebBuildWithoutIndexIsAServerError(t *testing.T) {
	dir := t.TempDir()
	cfg := testEnv(0)
	cfg.WebDir = &dir

	decodeProblem(t, get(t, routerWithoutDB(t, cfg), "/claims"), http.StatusInternalServerError, "INTERNAL")
}

// Without WEB_DIR (development), the API serves no web build.
func TestNoWebBuildWithoutWebDir(t *testing.T) {
	decodeProblem(t, get(t, routerWithoutDB(t, testEnv(0)), "/"), http.StatusNotFound, "NOT_FOUND")
}
