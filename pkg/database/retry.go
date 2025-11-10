package database

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/richxcame/ride-hailing/pkg/resilience"
)

// RetryableQuery executes a database query with retry logic for transient failures
func RetryableQuery[T any](ctx context.Context, pool interface {
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
}, query string, args []interface{}, scanner func(pgx.Rows) (T, error)) (T, error) {
	config := resilience.DefaultRetryConfig()
	config.MaxAttempts = 3
	config.InitialBackoff = 100 * time.Millisecond
	config.MaxBackoff = 2 * time.Second
	config.RetryableChecker = isPostgresRetryable

	result, err := resilience.RetryWithName(ctx, config, func(ctx context.Context) (interface{}, error) {
		rows, err := pool.Query(ctx, query, args...)
		if err != nil {
			return *new(T), err
		}
		defer rows.Close()

		return scanner(rows)
	}, "database.query")

	if err != nil {
		return *new(T), err
	}

	return result.(T), nil
}

// RetryableQueryRow executes a database query row with retry logic for transient failures
func RetryableQueryRow[T any](ctx context.Context, pool interface {
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}, query string, args []interface{}, scanner func(pgx.Row) (T, error)) (T, error) {
	config := resilience.DefaultRetryConfig()
	config.MaxAttempts = 3
	config.InitialBackoff = 100 * time.Millisecond
	config.MaxBackoff = 2 * time.Second
	config.RetryableChecker = isPostgresRetryable

	result, err := resilience.RetryWithName(ctx, config, func(ctx context.Context) (interface{}, error) {
		row := pool.QueryRow(ctx, query, args...)
		return scanner(row)
	}, "database.query_row")

	if err != nil {
		return *new(T), err
	}

	return result.(T), nil
}

// RetryableExec executes a database command with retry logic for transient failures
func RetryableExec(ctx context.Context, pool interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
}, query string, args ...interface{}) (pgconn.CommandTag, error) {
	config := resilience.DefaultRetryConfig()
	config.MaxAttempts = 3
	config.InitialBackoff = 100 * time.Millisecond
	config.MaxBackoff = 2 * time.Second
	config.RetryableChecker = isPostgresRetryable

	result, err := resilience.RetryWithName(ctx, config, func(ctx context.Context) (interface{}, error) {
		return pool.Exec(ctx, query, args...)
	}, "database.exec")

	if err != nil {
		return pgconn.CommandTag{}, err
	}

	return result.(pgconn.CommandTag), nil
}

// RetryableTransaction executes a transaction with retry logic for serialization failures
func RetryableTransaction(ctx context.Context, pool interface {
	Begin(context.Context) (pgx.Tx, error)
}, fn func(pgx.Tx) error) error {
	config := resilience.DefaultRetryConfig()
	config.MaxAttempts = 3
	config.InitialBackoff = 50 * time.Millisecond
	config.MaxBackoff = 1 * time.Second
	config.RetryableChecker = isPostgresRetryable

	_, err := resilience.RetryWithName(ctx, config, func(ctx context.Context) (interface{}, error) {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return nil, err
		}

		defer func() {
			if err != nil {
				// Rollback on error, but ignore rollback errors as the transaction is already failed
				_ = tx.Rollback(ctx)
			}
		}()

		err = fn(tx)
		if err != nil {
			return nil, err
		}

		err = tx.Commit(ctx)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}, "database.transaction")

	return err
}

// isPostgresRetryable determines if a PostgreSQL error should be retried
func isPostgresRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Don't retry context errors
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Check for PostgreSQL error codes
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// Retry on specific error codes
		switch pgErr.Code {
		case "40001": // serialization_failure
			return true
		case "40P01": // deadlock_detected
			return true
		case "55P03": // lock_not_available
			return true
		case "53000": // insufficient_resources
			return true
		case "53100": // disk_full
			return false // Don't retry on disk full
		case "53200": // out_of_memory
			return false // Don't retry on OOM
		case "53300": // too_many_connections
			return true
		case "53400": // configuration_limit_exceeded
			return true
		case "08000", "08003", "08006": // connection_exception
			return true
		case "57P01": // admin_shutdown
			return true
		case "57P02": // crash_shutdown
			return true
		case "57P03": // cannot_connect_now
			return true
		case "58000": // system_error
			return true
		case "XX000": // internal_error
			return true
		default:
			// Don't retry constraint violations, data exceptions, etc.
			if strings.HasPrefix(pgErr.Code, "23") { // Integrity constraint violation
				return false
			}
			if strings.HasPrefix(pgErr.Code, "22") { // Data exception
				return false
			}
			if strings.HasPrefix(pgErr.Code, "42") { // Syntax error or access rule violation
				return false
			}
		}
	}

	// Check for connection errors
	errMsg := err.Error()
	retryableMessages := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"no such host",
		"network is unreachable",
		"temporary failure",
		"timeout",
		"too many connections",
		"server closed",
		"unexpected EOF",
	}

	for _, msg := range retryableMessages {
		if strings.Contains(strings.ToLower(errMsg), msg) {
			return true
		}
	}

	// Check for specific pgx errors
	if errors.Is(err, pgx.ErrNoRows) {
		return false // No rows is not a retryable error
	}

	// Don't retry by default for unknown errors
	return false
}

// ConservativeRetryConfig returns a conservative retry configuration for critical operations
func ConservativeRetryConfig() resilience.RetryConfig {
	config := resilience.ConservativeRetryConfig()
	config.RetryableChecker = isPostgresRetryable
	return config
}

// AggressiveRetryConfig returns an aggressive retry configuration for non-critical reads
func AggressiveRetryConfig() resilience.RetryConfig {
	config := resilience.AggressiveRetryConfig()
	config.RetryableChecker = isPostgresRetryable
	return config
}
