package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/beambeambeam/claimflow/apps/api/env"
)

// setVarlockEnv sets the blob that `varlock run` injects, from a valid config with overrides.
// A nil override removes the key.
func setVarlockEnv(t *testing.T, overrides map[string]any) {
	t.Helper()

	values := map[string]any{
		"GIN_MODE":      "debug",
		"PORT":          9000,
		"DATABASE_URL":  "postgres://claimflow@localhost:5432/claimflow",
		"COOKIE_SECURE": false,
		"BCRYPT_COST":   12,
		"LOG_LEVEL":     "debug",
		"APP_VERSION":   "v1.0.0",
	}

	for key, value := range overrides {
		if value == nil {
			delete(values, key)
		} else {
			values[key] = value
		}
	}

	config := map[string]map[string]any{}
	for key, value := range values {
		config[key] = map[string]any{"value": value}
	}

	blob, err := json.Marshal(map[string]any{"config": config})
	if err != nil {
		t.Fatalf("marshal env: %v", err)
	}

	t.Setenv("__VARLOCK_ENV", string(blob))
}

func TestLoadConfig(t *testing.T) {
	setVarlockEnv(t, nil)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	want := env.Env{
		GinMode:      "debug",
		Port:         9000,
		DatabaseUrl:  "postgres://claimflow@localhost:5432/claimflow",
		CookieSecure: false,
		BcryptCost:   12,
		LogLevel:     "debug",
		AppVersion:   "v1.0.0",
	}
	if cfg != want {
		t.Fatalf("loadConfig() = %+v, want %+v", cfg, want)
	}
}

// F01-TEST-17: a missing DATABASE_URL or a BCRYPT_COST below 10 stops the API
// with an error that names the variable.
func TestLoadConfigNamesTheInvalidVariable(t *testing.T) {
	tests := []struct {
		name      string
		overrides map[string]any
		variable  string
	}{
		{"missing DATABASE_URL", map[string]any{"DATABASE_URL": nil}, "DATABASE_URL"},
		{"BCRYPT_COST below 10", map[string]any{"BCRYPT_COST": 9}, "BCRYPT_COST"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setVarlockEnv(t, test.overrides)

			if _, err := loadConfig(); err == nil || !strings.Contains(err.Error(), test.variable) {
				t.Fatalf("loadConfig() error = %v, want an error naming %s", err, test.variable)
			}
		})
	}
}

func TestLoadConfigWithoutVarlock(t *testing.T) {
	t.Setenv("__VARLOCK_ENV", "")

	if _, err := loadConfig(); err == nil || !strings.Contains(err.Error(), "load env") {
		t.Fatalf("loadConfig() error = %v, want load env error", err)
	}
}

func TestListenAddr(t *testing.T) {
	if got := listenAddr(8080); got != ":8080" {
		t.Fatalf("listenAddr() = %s, want :8080", got)
	}
}
