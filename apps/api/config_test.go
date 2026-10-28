package main

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLoadConfigDefaults(t *testing.T) {
	cfg, err := loadConfig(nil)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if cfg.GinMode != gin.ReleaseMode {
		t.Fatalf("GinMode = %s, want %s", cfg.GinMode, gin.ReleaseMode)
	}

	if got := cfg.listenAddr(); got != ":8080" {
		t.Fatalf("listenAddr() = %s, want :8080", got)
	}
}

func TestLoadConfigValues(t *testing.T) {
	cfg, err := loadConfig(map[string]string{"GIN_MODE": gin.DebugMode, "PORT": "9000"})
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if cfg.GinMode != gin.DebugMode {
		t.Fatalf("GinMode = %s, want %s", cfg.GinMode, gin.DebugMode)
	}

	if got := cfg.listenAddr(); got != ":9000" {
		t.Fatalf("listenAddr() = %s, want :9000", got)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	tests := map[string]map[string]string{
		"port not a number": {"PORT": "invalid"},
		"port out of range": {"PORT": "70000"},
	}

	for name, environ := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := loadConfig(environ); err == nil || !strings.Contains(err.Error(), "parse env") {
				t.Fatalf("loadConfig() error = %v, want parse env error", err)
			}
		})
	}
}
