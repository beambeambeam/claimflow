package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/beambeambeam/claimflow/apps/api/internal/pgtest"
)

const (
	authMigration   = "00001_auth.sql"
	claimsMigration = "00002_claims.sql"
)

func runMigrate(t *testing.T, databaseURL, command string) string {
	t.Helper()

	var out bytes.Buffer
	if err := migrate(t.Context(), databaseURL, []string{command}, &out); err != nil {
		t.Fatalf("migrate %s: %v", command, err)
	}

	return out.String()
}

// Every migration rolls back to an empty database and applies again (F-02 DoD: 00002_claims.sql runs down and up).
func TestMigrateDownStatusUp(t *testing.T) {
	databaseURL := pgtest.New(t).Config().ConnString()

	if out := runMigrate(t, databaseURL, "status"); !strings.Contains(out, "applied "+claimsMigration) {
		t.Fatalf("status = %q, want %s applied", out, claimsMigration)
	}

	for _, migration := range []string{claimsMigration, authMigration} {
		if out := runMigrate(t, databaseURL, "down"); out != "rolled back "+migration+"\n" {
			t.Fatalf("down = %q, want %s rolled back", out, migration)
		}
	}

	if out := runMigrate(t, databaseURL, "status"); !strings.Contains(out, "pending "+authMigration) ||
		!strings.Contains(out, "pending "+claimsMigration) {
		t.Fatalf("status = %q, want every migration pending", out)
	}

	if out := runMigrate(t, databaseURL, "up"); out != "applied "+authMigration+"\napplied "+claimsMigration+"\n" {
		t.Fatalf("up = %q, want %s and %s applied", out, authMigration, claimsMigration)
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
