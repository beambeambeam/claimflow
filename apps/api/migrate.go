package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver for goose

	"github.com/beambeambeam/claimflow/apps/api/migrations"
)

var errMigrateUsage = errors.New("usage: api migrate up|down|status")

// migrate runs `api migrate <command>`: `up` applies every pending migration,
// `down` rolls back the latest one, and `status` lists them.
func migrate(ctx context.Context, databaseURL string, args []string, out io.Writer) error {
	if len(args) != 1 {
		return errMigrateUsage
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil { // coverage-ignore
		return fmt.Errorf("DATABASE_URL: %w", err)
	}
	defer db.Close()

	provider, err := migrations.NewProvider(db)
	if err != nil { // coverage-ignore
		return fmt.Errorf("migrate: %w", err)
	}

	switch args[0] {
	case "up":
		results, err := provider.Up(ctx)
		for _, result := range results {
			fmt.Fprintln(out, "applied", result.Source.Path)
		}

		if err != nil {
			return fmt.Errorf("migrate up: %w", err)
		}
	case "down":
		result, err := provider.Down(ctx)
		if err != nil {
			return fmt.Errorf("migrate down: %w", err)
		}

		fmt.Fprintln(out, "rolled back", result.Source.Path)
	case "status":
		statuses, err := provider.Status(ctx)
		if err != nil {
			return fmt.Errorf("migrate status: %w", err)
		}

		for _, status := range statuses {
			fmt.Fprintln(out, status.State, status.Source.Path)
		}
	default:
		return errMigrateUsage
	}

	return nil
}
