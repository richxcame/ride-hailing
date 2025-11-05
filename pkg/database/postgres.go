package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richxcame/ride-hailing/pkg/config"
)

// NewPostgresPool creates a new PostgreSQL connection pool with optimized settings
func NewPostgresPool(cfg *config.DatabaseConfig) (*pgxpool.Pool, error) {
	dsn := cfg.DSN()

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	// Connection pool settings
	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MinConns = int32(cfg.MinConns)
	poolConfig.MaxConnLifetime = time.Hour              // Recycle connections after 1 hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute       // Close idle connections after 30 mins
	poolConfig.HealthCheckPeriod = time.Minute          // Check connection health every minute
	poolConfig.ConnConfig.ConnectTimeout = 10 * time.Second

	// Statement cache for better performance
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheStatement

	// Runtime parameters for performance
	poolConfig.ConnConfig.RuntimeParams["application_name"] = "ridehailing"
	poolConfig.ConnConfig.RuntimeParams["timezone"] = "UTC"

	// Enable prepared statement caching
	poolConfig.ConnConfig.RuntimeParams["plan_cache_mode"] = "auto"

	// Optimize work_mem for complex queries (only for this connection)
	// Note: This is per-operation, not per-connection
	poolConfig.ConnConfig.RuntimeParams["work_mem"] = "16MB"

	// Connection callback for additional setup
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		// Set statement timeout to prevent long-running queries
		_, err := conn.Exec(ctx, "SET statement_timeout = '30s'")
		return err
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return pool, nil
}

// Close closes the database connection pool
func Close(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
	}
}
