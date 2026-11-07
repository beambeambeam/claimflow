package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/beambeambeam/claimflow/apps/api/env"
	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

const testVersion = "v1.2.3-test"

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(pgtest.Run(m))
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// problem is the RFC 9457 body every error response must have.
type problem struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Code      string `json:"code"`
	Instance  string `json:"instance"`
	RequestID string `json:"requestId"`
}

func get(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody)
	handler.ServeHTTP(recorder, request)

	return recorder
}

func decodeProblem(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) problem {
	t.Helper()

	if recorder.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, status, recorder.Body)
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want application/problem+json", got)
	}

	var body problem
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode problem: %v", err)
	}

	if body.Type != "about:blank" || body.Status != status || body.Code != code || body.Title == "" {
		t.Fatalf("problem = %+v, want type about:blank, status %d, code %s, a title", body, status, code)
	}

	if body.RequestID == "" || body.RequestID != recorder.Header().Get("X-Request-Id") {
		t.Fatalf("requestId = %q, X-Request-Id = %q, want the same non-empty value",
			body.RequestID, recorder.Header().Get("X-Request-Id"))
	}

	return body
}

// F01-TEST-15: unexpected errors and unknown paths return problems, and a 500
// hides the error text, which only the log line with the same requestId has.
func TestProblemResponses(t *testing.T) {
	var logs bytes.Buffer

	router := newRouter(slog.New(slog.NewJSONHandler(&logs, nil)), testVersion, pgtest.New(t))
	router.GET("/api/v1/test/internal-error", func(c *gin.Context) {
		_ = c.Error(errors.New(`pq: relation "secret_table" does not exist`))
	})
	router.GET("/api/v1/test/panic", func(*gin.Context) { panic("secret panic value") })

	for _, path := range []string{"/api/v1/test/internal-error", "/api/v1/test/panic"} {
		logs.Reset()

		recorder := get(t, router, path)
		body := decodeProblem(t, recorder, http.StatusInternalServerError, "INTERNAL")

		if strings.Contains(recorder.Body.String(), "secret") {
			t.Fatalf("%s: 500 body leaks the error: %s", path, recorder.Body)
		}

		if body.Instance != path {
			t.Fatalf("%s: instance = %q, want the path", path, body.Instance)
		}

		if !logLineHas(logs.String(), body.RequestID, "secret") {
			t.Fatalf("%s: no log line has requestId %s and the error text; logs = %s", path, body.RequestID, logs.String())
		}
	}

	notFound := decodeProblem(t, get(t, router, "/api/v1/nope"), http.StatusNotFound, "NOT_FOUND")
	if notFound.Instance != "/api/v1/nope" {
		t.Fatalf("instance = %q, want /api/v1/nope", notFound.Instance)
	}
}

// logLineHas reports whether one JSON log line has requestId and contains text.
func logLineHas(logs, requestID, text string) bool {
	for line := range strings.Lines(logs) {
		var entry map[string]any
		if json.Unmarshal([]byte(line), &entry) == nil && entry["requestId"] == requestID && strings.Contains(line, text) {
			return true
		}
	}

	return false
}

func TestEveryResponseHasRequestID(t *testing.T) {
	var logs bytes.Buffer

	recorder := get(t, newRouter(slog.New(slog.NewJSONHandler(&logs, nil)), testVersion, pgtest.New(t)), "/api/v1/health")

	requestID := recorder.Header().Get("X-Request-Id")
	if requestID == "" {
		t.Fatal("X-Request-Id is empty")
	}

	if !logLineHas(logs.String(), requestID, `"status":200`) {
		t.Fatalf("no request log line with requestId %s; logs = %s", requestID, logs.String())
	}
}

// F01-TEST-16: /health works without the database; /ready follows it.
func TestHealthAndReadyFollowTheDatabase(t *testing.T) {
	server, err := pgtest.Start(t.Context())
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	router := newRouter(discardLogger(), testVersion, server.New(t))

	assertHealthy := func() {
		t.Helper()

		recorder := get(t, router, "/api/v1/health")
		if recorder.Code != http.StatusOK || strings.TrimSpace(recorder.Body.String()) != `{"status":"ok","version":"`+testVersion+`"}` {
			t.Fatalf("health = %d %s, want 200 with status and version", recorder.Code, recorder.Body)
		}
	}

	assertHealthy()

	if recorder := get(t, router, "/api/v1/ready"); recorder.Code != http.StatusOK || strings.TrimSpace(recorder.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("ready with database = %d %s, want 200 {\"status\":\"ok\"}", recorder.Code, recorder.Body)
	}

	if err := server.Stop(t.Context()); err != nil {
		t.Fatalf("stop postgres: %v", err)
	}

	assertHealthy()
	decodeProblem(t, get(t, router, "/api/v1/ready"), http.StatusServiceUnavailable, "UNAVAILABLE")
}

func TestLoggerUsesCloudLoggingFields(t *testing.T) {
	var logs bytes.Buffer

	logger := newLogger(&logs, "warn")
	logger.Info("hidden")
	logger.Warn("shown")

	line := strings.TrimSpace(logs.String())
	if strings.Contains(line, "hidden") || !strings.Contains(line, `"severity":"WARN"`) || !strings.Contains(line, `"message":"shown"`) {
		t.Fatalf("logs = %s, want only the warn line with severity and message", line)
	}
}

func testEnv(port int) *env.Env {
	return &env.Env{
		GinMode:     gin.TestMode,
		Port:        int64(port),
		DatabaseUrl: "postgres://claimflow@127.0.0.1:1/claimflow",
		BcryptCost:  10,
		LogLevel:    "info",
		AppVersion:  testVersion,
	}
}

func freePort(t *testing.T) int {
	t.Helper()

	listener, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("release port: %v", err)
	}

	return port
}

func TestRunServesAndShutsDown(t *testing.T) {
	port := freePort(t)
	ctx, cancel := context.WithCancel(t.Context())

	done := make(chan error, 1)
	go func() {
		done <- run(ctx, testEnv(port), discardLogger())
	}()

	body := getWhenReady(t, "http://127.0.0.1:"+strconv.Itoa(port)+"/api/v1/health")
	if body != `{"status":"ok","version":"`+testVersion+`"}` {
		t.Fatalf("body = %s, want health with version", body)
	}

	cancel()

	if err := <-done; err != nil {
		t.Fatalf("run() = %v, want nil", err)
	}
}

func TestRunInvalidDatabaseURL(t *testing.T) {
	cfg := testEnv(freePort(t))
	cfg.DatabaseUrl = "postgres://%zz"

	if err := run(t.Context(), cfg, discardLogger()); err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("run() = %v, want DATABASE_URL error", err)
	}
}

func TestRunListenError(t *testing.T) {
	listener, err := new(net.ListenConfig).Listen(t.Context(), "tcp", ":0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	port := listener.Addr().(*net.TCPAddr).Port

	err = run(t.Context(), testEnv(port), discardLogger())
	if err == nil || !strings.Contains(err.Error(), "listen on :"+strconv.Itoa(port)) {
		t.Fatalf("run() = %v, want listen error", err)
	}
}

func TestServeError(t *testing.T) {
	listener, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	err = serve(t.Context(), listener, http.NotFoundHandler(), discardLogger())
	if err == nil || !strings.Contains(err.Error(), "serve:") {
		t.Fatalf("serve() = %v, want serve error", err)
	}
}

func getWhenReady(t *testing.T, url string) string {
	t.Helper()

	for range 50 {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
		if err != nil {
			t.Fatalf("build request: %v", err)
		}

		response, err := http.DefaultClient.Do(request)
		if err != nil {
			time.Sleep(20 * time.Millisecond)

			continue
		}

		body, err := io.ReadAll(response.Body)
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Fatalf("close body: %v", closeErr)
		}

		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		return strings.TrimSpace(string(body))
	}

	t.Fatal("server did not start")

	return ""
}
