package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func freePort(t *testing.T) string {
	t.Helper()

	listener, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}

	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	if err := listener.Close(); err != nil {
		t.Fatalf("release port: %v", err)
	}

	return port
}

func TestHealth(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/health", http.NoBody)

	newRouter(discardLogger()).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	if body := recorder.Body.String(); body != `{"status":"ok"}` {
		t.Fatalf("body = %s, want {\"status\":\"ok\"}", body)
	}
}

func TestRequestLogger(t *testing.T) {
	var logs bytes.Buffer

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/health", http.NoBody)

	newRouter(slog.New(slog.NewJSONHandler(&logs, nil))).ServeHTTP(recorder, request)

	for _, want := range []string{`"msg":"request"`, `"path":"/health"`, `"status":200`} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("logs = %s, want %s", logs.String(), want)
		}
	}
}

func TestRecoverer(t *testing.T) {
	var logs bytes.Buffer

	router := newRouter(slog.New(slog.NewJSONHandler(&logs, nil)))
	router.GET("/panic", func(*gin.Context) { panic("boom") })

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/panic", http.NoBody)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}

	if !strings.Contains(logs.String(), `"panic":"boom"`) {
		t.Fatalf("logs = %s, want panic entry", logs.String())
	}
}

func TestRunServesAndShutsDown(t *testing.T) {
	port := freePort(t)
	ctx, cancel := context.WithCancel(t.Context())

	done := make(chan error, 1)
	go func() {
		done <- run(ctx, map[string]string{gin.EnvGinMode: gin.TestMode, "PORT": port}, discardLogger())
	}()

	body := getWhenReady(t, "http://127.0.0.1:"+port+"/health")
	if body != `{"status":"ok"}` {
		t.Fatalf("body = %s, want {\"status\":\"ok\"}", body)
	}

	cancel()

	if err := <-done; err != nil {
		t.Fatalf("run() = %v, want nil", err)
	}
}

func TestRunConfigError(t *testing.T) {
	err := run(t.Context(), map[string]string{"PORT": "invalid"}, discardLogger())
	if err == nil || !strings.Contains(err.Error(), "parse env") {
		t.Fatalf("run() = %v, want config error", err)
	}
}

func TestRunListenError(t *testing.T) {
	listener, err := new(net.ListenConfig).Listen(t.Context(), "tcp", ":0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)

	err = run(t.Context(), map[string]string{gin.EnvGinMode: gin.TestMode, "PORT": port}, discardLogger())
	if err == nil || !strings.Contains(err.Error(), "listen on :"+port) {
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

		return string(body)
	}

	t.Fatal("server did not start")

	return ""
}
