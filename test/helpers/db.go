package helpers

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultTestDatabaseURL = "postgres://testuser:testpassword@localhost:5433/ride_hailing_test?sslmode=disable"
	migrationsPath         = "file://db/migrations"
)

// SetupTestDatabase creates a dedicated PostgreSQL connection pool for tests.
// It runs all migrations before returning the pool and registers a cleanup
// callback to close the pool when the test completes.
func SetupTestDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := getTestDatabaseURL()
	runMigrations(t, databaseURL)

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("failed to parse test database config: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create test database pool: %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Fatalf("failed to ping test database: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

// ResetTables truncates the supplied tables so every test can start from a
// known state without recreating the schema.
func ResetTables(t *testing.T, pool *pgxpool.Pool, tables ...string) {
	t.Helper()

	if len(tables) == 0 {
		return
	}

	stmt := fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", strings.Join(tables, ", "))
	if _, err := pool.Exec(context.Background(), stmt); err != nil {
		t.Fatalf("failed to truncate tables %v: %v", tables, err)
	}
}

func getTestDatabaseURL() string {
	if value := os.Getenv("TEST_DATABASE_URL"); value != "" {
		return value
	}
	if value := os.Getenv("DATABASE_URL"); value != "" {
		return value
	}
	return defaultTestDatabaseURL
}

func runMigrations(t *testing.T, databaseURL string) {
	t.Helper()

	m, err := migrate.New(migrationsPath, databaseURL)
	if err != nil {
		t.Fatalf("failed to initialize migrations: %v", err)
	}
	defer m.Close()

	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("failed to reset migrations: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("failed to apply migrations: %v", err)
	}
}
