// Package pgtest starts a real PostgreSQL for integration tests.
//
// A test package calls Run from TestMain. Run starts one container and migrates a
// template database. New then gives each test its own database, cloned from the
// template, so tests are isolated and can run in parallel.
package pgtest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver for goose
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/beambeambeam/claimflow/apps/api/migrations"
)

const (
	image        = "postgres:17-alpine"
	templateName = "claimflow_template"
	user         = "claimflow"
	password     = "claimflow"
)

// Server is one PostgreSQL container with a migrated template database.
type Server struct {
	container *postgres.PostgresContainer
	mu        sync.RWMutex
	stopped   bool
	next      atomic.Int64
}

// Start starts a container and applies every migration to the template database.
func Start(ctx context.Context) (*Server, error) {
	container, err := postgres.Run(ctx, image,
		postgres.WithDatabase(templateName),
		postgres.WithUsername(user),
		postgres.WithPassword(password),
		postgres.BasicWaitStrategies(),
	)
	if err != nil { // coverage-ignore
		return nil, fmt.Errorf("start postgres: %w", err)
	}

	server := &Server{container: container}

	if err := server.migrateTemplate(ctx); err != nil { // coverage-ignore
		_ = testcontainers.TerminateContainer(container)

		return nil, err
	}

	return server, nil
}

func (s *Server) migrateTemplate(ctx context.Context) error {
	url, err := s.container.ConnectionString(ctx, "sslmode=disable")
	if err != nil { // coverage-ignore
		return fmt.Errorf("template connection string: %w", err)
	}

	db, err := sql.Open("pgx", url)
	if err != nil { // coverage-ignore
		return fmt.Errorf("open template: %w", err)
	}
	defer db.Close()

	provider, err := migrations.NewProvider(db)
	if err != nil { // coverage-ignore
		return fmt.Errorf("migration provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil { // coverage-ignore
		return fmt.Errorf("migrate template: %w", err)
	}

	return nil
}

// New returns a pool for a fresh database cloned from the migrated template.
// The database is dropped when the test ends.
func (s *Server) New(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := t.Context()
	name := fmt.Sprintf("test_%d", s.next.Add(1))

	s.exec(t, "CREATE DATABASE "+name+" TEMPLATE "+templateName)

	url, err := s.container.ConnectionString(ctx, "sslmode=disable")
	if err != nil { // coverage-ignore
		t.Fatalf("connection string: %v", err)
	}

	config, err := pgxpool.ParseConfig(url)
	if err != nil { // coverage-ignore
		t.Fatalf("parse connection string: %v", err)
	}

	config.ConnConfig.Database = name

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil { // coverage-ignore
		t.Fatalf("open %s: %v", name, err)
	}

	t.Cleanup(func() {
		pool.Close()

		s.mu.RLock()
		defer s.mu.RUnlock()

		if !s.stopped {
			s.exec(t, "DROP DATABASE "+name+" WITH (FORCE)")
		}
	})

	return pool
}

// exec runs one statement on the maintenance database.
func (s *Server) exec(t *testing.T, statement string) {
	t.Helper()

	ctx := context.WithoutCancel(t.Context())

	url, err := s.container.ConnectionString(ctx, "sslmode=disable")
	if err != nil { // coverage-ignore
		t.Fatalf("connection string: %v", err)
	}

	config, err := pgx.ParseConfig(url)
	if err != nil { // coverage-ignore
		t.Fatalf("parse connection string: %v", err)
	}

	config.Database = "postgres"

	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil { // coverage-ignore
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, statement); err != nil { // coverage-ignore
		t.Fatalf("%s: %v", statement, err)
	}
}

// Stop stops the container. Pools from New then fail, as they would in an outage.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopped = true

	if err := s.container.Terminate(ctx); err != nil { // coverage-ignore
		return fmt.Errorf("stop postgres: %w", err)
	}

	return nil
}

var shared *Server

// Run starts the shared server, runs the package's tests, and stops the server.
// Call it from TestMain: `func TestMain(m *testing.M) { os.Exit(pgtest.Run(m)) }`.
func Run(m *testing.M) int {
	ctx := context.Background()

	server, err := Start(ctx)
	if err != nil { // coverage-ignore
		fmt.Fprintln(os.Stderr, "pgtest:", err)

		return 1
	}

	shared = server

	// Coverage is written inside m.Run, so code after it never counts.
	defer func() { // coverage-ignore
		if err := server.Stop(ctx); err != nil { // coverage-ignore
			fmt.Fprintln(os.Stderr, "pgtest:", err)
		}
	}()

	return m.Run()
}

// New returns a fresh database on the shared server started by Run.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()

	return shared.New(t)
}
