// Package migrations holds the goose SQL migrations, embedded in the API binary.
package migrations

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

//go:embed *.sql
var files embed.FS

// NewProvider returns a goose provider for the embedded migrations.
// A PostgreSQL advisory lock makes a second concurrent run wait for the first.
func NewProvider(db *sql.DB) (*goose.Provider, error) {
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil { // coverage-ignore
		return nil, fmt.Errorf("create migration lock: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, files,
		goose.WithSessionLocker(locker),
		goose.WithDisableGlobalRegistry(true),
	)
	if err != nil { // coverage-ignore
		return nil, fmt.Errorf("create migration provider: %w", err)
	}

	return provider, nil
}
