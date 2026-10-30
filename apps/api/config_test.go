package main

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/beambeambeam/karupan/apps/api/env"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("__VARLOCK_ENV", `{"config":{"GIN_MODE":{"value":"debug"},"PORT":{"value":9000}}}`)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if cfg.GinMode != gin.DebugMode {
		t.Fatalf("GinMode = %s, want %s", cfg.GinMode, gin.DebugMode)
	}

	if cfg.Port != 9000 {
		t.Fatalf("Port = %d, want 9000", cfg.Port)
	}
}

func TestLoadConfigWithoutVarlock(t *testing.T) {
	t.Setenv("__VARLOCK_ENV", "")

	if _, err := loadConfig(); err == nil || !strings.Contains(err.Error(), "load env") {
		t.Fatalf("loadConfig() error = %v, want load env error", err)
	}
}

func TestListenAddr(t *testing.T) {
	if got := listenAddr(env.Env{Port: 8080}); got != ":8080" {
		t.Fatalf("listenAddr() = %s, want :8080", got)
	}
}
