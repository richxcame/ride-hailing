package redis

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/richxcame/ride-hailing/pkg/resilience"
)

// RetryableOperation executes a Redis operation with retry logic for transient failures
func RetryableOperation[T any](ctx context.Context, operation func(context.Context) (T, error), operationName string) (T, error) {
	config := resilience.DefaultRetryConfig()
	config.MaxAttempts = 3
	config.InitialBackoff = 50 * time.Millisecond
	config.MaxBackoff = 1 * time.Second
	config.RetryableChecker = isRedisRetryable

	result, err := resilience.RetryWithName(ctx, config, func(ctx context.Context) (interface{}, error) {
		return operation(ctx)
	}, operationName)

	if err != nil {
		return *new(T), err
	}

	return result.(T), nil
}

// RetryableSet sets a key-value pair with retry logic
func (c *Client) RetryableSet(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	_, err := RetryableOperation(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, c.Set(ctx, key, value, expiration).Err()
	}, "redis.set")
	return err
}

// RetryableGet gets a value by key with retry logic
func (c *Client) RetryableGet(ctx context.Context, key string) (string, error) {
	return RetryableOperation(ctx, func(ctx context.Context) (string, error) {
		return c.Get(ctx, key).Result()
	}, "redis.get")
}

// RetryableGeoAdd adds a location with retry logic
func (c *Client) RetryableGeoAdd(ctx context.Context, key string, longitude, latitude float64, member string) error {
	_, err := RetryableOperation(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, c.GeoAdd(ctx, key, longitude, latitude, member)
	}, "redis.geoadd")
	return err
}

// RetryableGeoRadius searches for members within a radius with retry logic
func (c *Client) RetryableGeoRadius(ctx context.Context, key string, longitude, latitude, radiusKm float64, count int) ([]string, error) {
	return RetryableOperation(ctx, func(ctx context.Context) ([]string, error) {
		return c.GeoRadius(ctx, key, longitude, latitude, radiusKm, count)
	}, "redis.georadius")
}

// RetryableHSet sets a hash field with retry logic
func (c *Client) RetryableHSet(ctx context.Context, key, field string, value interface{}) error {
	_, err := RetryableOperation(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, c.HSet(ctx, key, field, value)
	}, "redis.hset")
	return err
}

// RetryableHGet gets a hash field with retry logic
func (c *Client) RetryableHGet(ctx context.Context, key, field string) (string, error) {
	return RetryableOperation(ctx, func(ctx context.Context) (string, error) {
		return c.HGet(ctx, key, field)
	}, "redis.hget")
}

// RetryableIncr increments a counter with retry logic
func (c *Client) RetryableIncr(ctx context.Context, key string) (int64, error) {
	return RetryableOperation(ctx, func(ctx context.Context) (int64, error) {
		return c.Incr(ctx, key)
	}, "redis.incr")
}

// RetryableSAdd adds members to a set with retry logic
func (c *Client) RetryableSAdd(ctx context.Context, key string, members ...interface{}) error {
	_, err := RetryableOperation(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, c.SAdd(ctx, key, members...)
	}, "redis.sadd")
	return err
}

// RetryableSMembers gets all members of a set with retry logic
func (c *Client) RetryableSMembers(ctx context.Context, key string) ([]string, error) {
	return RetryableOperation(ctx, func(ctx context.Context) ([]string, error) {
		return c.SMembers(ctx, key)
	}, "redis.smembers")
}

// RetryableZAdd adds members to a sorted set with retry logic
func (c *Client) RetryableZAdd(ctx context.Context, key string, score float64, member interface{}) error {
	_, err := RetryableOperation(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, c.ZAdd(ctx, key, score, member)
	}, "redis.zadd")
	return err
}

// RetryableZRange gets a range from sorted set with retry logic
func (c *Client) RetryableZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return RetryableOperation(ctx, func(ctx context.Context) ([]string, error) {
		return c.ZRange(ctx, key, start, stop)
	}, "redis.zrange")
}

// RetryableDelete deletes keys with retry logic
func (c *Client) RetryableDelete(ctx context.Context, keys ...string) error {
	_, err := RetryableOperation(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, c.Delete(ctx, keys...)
	}, "redis.delete")
	return err
}

// RetryableExpire sets an expiration with retry logic
func (c *Client) RetryableExpire(ctx context.Context, key string, expiration time.Duration) error {
	_, err := RetryableOperation(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, c.Expire(ctx, key, expiration)
	}, "redis.expire")
	return err
}

// isRedisRetryable determines if a Redis error should be retried
func isRedisRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Don't retry context errors
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Don't retry Nil (key not found) - this is expected behavior
	if errors.Is(err, redis.Nil) {
		return false
	}

	// Check for connection errors
	errMsg := strings.ToLower(err.Error())
	retryableMessages := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"no such host",
		"network is unreachable",
		"temporary failure",
		"timeout",
		"server closed",
		"unexpected eof",
		"pool timeout",
		"i/o timeout",
		"connection pool exhausted",
		"loading",     // Redis is loading dataset
		"busy",        // Redis is busy (script execution, etc.)
		"masterdown",  // Master is down
		"readonly",    // Replica is read-only (for write operations on replica)
		"noscript",    // Script not in cache (can retry after loading)
		"cluster",     // Cluster-related transient errors
		"moved",       // Key moved to another node (Redis Cluster)
		"ask",         // Redirection in Redis Cluster
		"tryagain",    // Redis asking to retry
		"clusterdown", // Cluster is down
	}

	for _, msg := range retryableMessages {
		if strings.Contains(errMsg, msg) {
			return true
		}
	}

	// Don't retry on validation errors
	nonRetryableMessages := []string{
		"wrongtype",   // Operation against a key holding the wrong kind of value
		"err syntax",  // Syntax error
		"err invalid", // Invalid argument
		"noauth",      // Authentication required
		"wrongpass",   // Invalid password
		"noperm",      // No permission
		"err unknown", // Unknown command
		"execabort",   // Transaction aborted
	}

	for _, msg := range nonRetryableMessages {
		if strings.Contains(errMsg, msg) {
			return false
		}
	}

	// Retry by default for unknown errors (conservative approach for cache)
	return true
}

// ConservativeRetryConfig returns a conservative retry configuration for Redis
func ConservativeRetryConfig() resilience.RetryConfig {
	config := resilience.ConservativeRetryConfig()
	config.InitialBackoff = 50 * time.Millisecond
	config.MaxBackoff = 1 * time.Second
	config.RetryableChecker = isRedisRetryable
	return config
}

// AggressiveRetryConfig returns an aggressive retry configuration for Redis
func AggressiveRetryConfig() resilience.RetryConfig {
	config := resilience.AggressiveRetryConfig()
	config.InitialBackoff = 20 * time.Millisecond
	config.MaxBackoff = 500 * time.Millisecond
	config.RetryableChecker = isRedisRetryable
	return config
}
