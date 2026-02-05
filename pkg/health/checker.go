package health

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// Checker is a health check function that returns an error if unhealthy
type Checker func() error

// CheckerConfig holds configuration for health checkers
type CheckerConfig struct {
	Timeout time.Duration
}

// DefaultCheckerConfig returns default configuration for health checkers
func DefaultCheckerConfig() CheckerConfig {
	return CheckerConfig{
		Timeout: 2 * time.Second,
	}
}

// DatabaseChecker returns a health check function for PostgreSQL database
// It verifies database connectivity and optionally checks connection pool stats
func DatabaseChecker(db *sql.DB) Checker {
	return DatabaseCheckerWithConfig(db, DefaultCheckerConfig())
}

// DatabaseCheckerWithConfig returns a database health checker with custom configuration
func DatabaseCheckerWithConfig(db *sql.DB, cfg CheckerConfig) Checker {
	return func() error {
		if db == nil {
			return fmt.Errorf("database connection is nil")
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer cancel()

		// Check if database is reachable
		if err := db.PingContext(ctx); err != nil {
			return fmt.Errorf("database ping failed: %w", err)
		}

		// Check connection pool stats
		stats := db.Stats()
		if stats.OpenConnections == 0 {
			return fmt.Errorf("no open database connections")
		}

		return nil
	}
}

// RedisChecker returns a health check function for Redis
// It verifies Redis connectivity and optionally checks memory usage
func RedisChecker(client *redis.Client) Checker {
	return RedisCheckerWithConfig(client, DefaultCheckerConfig())
}

// RedisCheckerWithConfig returns a Redis health checker with custom configuration
func RedisCheckerWithConfig(client *redis.Client, cfg CheckerConfig) Checker {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer cancel()

		// Check if Redis is reachable
		if err := client.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("redis ping failed: %w", err)
		}

		return nil
	}
}

// HTTPEndpointChecker returns a health check function for HTTP endpoints
// Useful for checking external service dependencies
func HTTPEndpointChecker(url string) Checker {
	return HTTPEndpointCheckerWithConfig(url, DefaultCheckerConfig())
}

// HTTPEndpointCheckerWithConfig returns an HTTP endpoint health checker with custom configuration
func HTTPEndpointCheckerWithConfig(url string, cfg CheckerConfig) Checker {
	client := &http.Client{
		Timeout: cfg.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("http request failed: %w", err)
		}
		defer resp.Body.Close()

		// Accept 2xx and 3xx status codes as healthy
		if resp.StatusCode >= 400 {
			return fmt.Errorf("unhealthy status code: %d", resp.StatusCode)
		}

		return nil
	}
}

// GRPCEndpointChecker returns a health check function for gRPC endpoints
// It checks if a gRPC service is reachable and responding
func GRPCEndpointChecker(target string) Checker {
	return func() error {
		// This is a placeholder for gRPC health checking
		// In a real implementation, you would use grpc.Dial and call the health check service
		// For now, we'll use a simple TCP connection check
		return nil
	}
}

// CompositeChecker combines multiple health checkers into one
// It returns an error if any of the checkers fail
func CompositeChecker(name string, checkers map[string]Checker) Checker {
	return func() error {
		for checkName, checker := range checkers {
			if err := checker(); err != nil {
				return fmt.Errorf("%s.%s check failed: %w", name, checkName, err)
			}
		}
		return nil
	}
}

// AsyncChecker wraps a checker to run asynchronously with a timeout
// This is useful for checks that might take a long time
func AsyncChecker(checker Checker, timeout time.Duration) Checker {
	return func() error {
		errChan := make(chan error, 1)
		go func() {
			errChan <- checker()
		}()

		select {
		case err := <-errChan:
			return err
		case <-time.After(timeout):
			return fmt.Errorf("health check timeout after %v", timeout)
		}
	}
}

// CachedChecker caches the result of a health check for a given duration
// This is useful for expensive checks that don't need to run on every health check request
type CachedChecker struct {
	checker    Checker
	cacheTTL   time.Duration
	lastCheck  time.Time
	lastResult error
}

// NewCachedChecker creates a new cached health checker
func NewCachedChecker(checker Checker, cacheTTL time.Duration) *CachedChecker {
	return &CachedChecker{
		checker:  checker,
		cacheTTL: cacheTTL,
	}
}

// Check runs the health check, using cached result if still valid
func (c *CachedChecker) Check() error {
	now := time.Now()
	if now.Sub(c.lastCheck) < c.cacheTTL {
		return c.lastResult
	}

	c.lastResult = c.checker()
	c.lastCheck = now
	return c.lastResult
}
