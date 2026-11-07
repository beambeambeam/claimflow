package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

const authMigration = "00001_auth.sql"

func runMigrate(t *testing.T, databaseURL, command string) string {
	t.Helper()

	var out bytes.Buffer
	if err := migrate(t.Context(), databaseURL, []string{command}, &out); err != nil {
		t.Fatalf("migrate %s: %v", command, err)
	}

	return out.String()
}

func TestMigrateDownStatusUp(t *testing.T) {
	databaseURL := pgtest.New(t).Config().ConnString()

	if out := runMigrate(t, databaseURL, "status"); !strings.Contains(out, "applied "+authMigration) {
		t.Fatalf("status = %q, want %s applied", out, authMigration)
	}

	if out := runMigrate(t, databaseURL, "down"); out != "rolled back "+authMigration+"\n" {
		t.Fatalf("down = %q, want %s rolled back", out, authMigration)
	}

	if out := runMigrate(t, databaseURL, "status"); !strings.Contains(out, "pending "+authMigration) {
		t.Fatalf("status = %q, want %s pending", out, authMigration)
	}

	if out := runMigrate(t, databaseURL, "up"); out != "applied "+authMigration+"\n" {
		t.Fatalf("up = %q, want %s applied", out, authMigration)
	}
}

func TestMigrateUsage(t *testing.T) {
	for _, args := range [][]string{nil, {"sideways"}, {"up", "extra"}} {
		err := migrate(t.Context(), "postgres://claimflow@127.0.0.1:1/claimflow", args, &bytes.Buffer{})
		if !errors.Is(err, errMigrateUsage) {
			t.Fatalf("migrate(%q) = %v, want usage error", args, err)
		}
	}
}

func TestMigrateUnreachableDatabase(t *testing.T) {
	for _, command := range []string{"up", "down", "status"} {
		err := migrate(t.Context(), "postgres://claimflow@127.0.0.1:1/claimflow?connect_timeout=1", []string{command}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "migrate "+command) {
			t.Fatalf("migrate %s = %v, want an error", command, err)
		}
	}
}
