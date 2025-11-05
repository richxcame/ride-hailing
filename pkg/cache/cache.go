package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache provides a high-level caching interface
type Cache struct {
	client *redis.Client
}

// NewCache creates a new cache instance
func NewCache(client *redis.Client) *Cache {
	return &Cache{client: client}
}

// Get retrieves a value from cache and unmarshals it into the target
func (c *Cache) Get(ctx context.Context, key string, target interface{}) error {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), target)
}

// Set stores a value in cache with expiration
func (c *Cache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, expiration).Err()
}

// Delete removes a key from cache
func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// DeletePattern deletes all keys matching a pattern
func (c *Cache) DeletePattern(ctx context.Context, pattern string) error {
	var cursor uint64
	var err error

	for {
		var keys []string
		keys, cursor, err = c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			err = c.client.Del(ctx, keys...).Err()
			if err != nil {
				return err
			}
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}

// Exists checks if a key exists
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Expire sets an expiration on a key
func (c *Cache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.client.Expire(ctx, key, expiration).Err()
}

// Increment increments a counter
func (c *Cache) Increment(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

// Decrement decrements a counter
func (c *Cache) Decrement(ctx context.Context, key string) (int64, error) {
	return c.client.Decr(ctx, key).Result()
}

// GetOrSet retrieves a value from cache or sets it using the provided function
func (c *Cache) GetOrSet(ctx context.Context, key string, expiration time.Duration, fn func() (interface{}, error), target interface{}) error {
	// Try to get from cache first
	err := c.Get(ctx, key, target)
	if err == nil {
		return nil // Cache hit
	}

	// Cache miss or error - execute function
	value, err := fn()
	if err != nil {
		return err
	}

	// Store in cache
	if err := c.Set(ctx, key, value, expiration); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to cache value: %v\n", err)
	}

	// Copy value to target
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, target)
}

// Remember caches the result of a function call
func (c *Cache) Remember(ctx context.Context, key string, expiration time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	// Try to get from cache first
	val, err := c.client.Get(ctx, key).Result()
	if err == nil {
		var result interface{}
		if err := json.Unmarshal([]byte(val), &result); err == nil {
			return result, nil
		}
	}

	// Cache miss - execute function
	result, err := fn()
	if err != nil {
		return nil, err
	}

	// Store in cache
	if err := c.Set(ctx, key, result, expiration); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to cache value: %v\n", err)
	}

	return result, nil
}

// SetNX sets a value only if it doesn't exist (useful for locks)
func (c *Cache) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, err
	}

	return c.client.SetNX(ctx, key, data, expiration).Result()
}

// GetTTL returns the remaining time to live for a key
func (c *Cache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

// Keys retrieves all keys matching a pattern
func (c *Cache) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.client.Keys(ctx, pattern).Result()
}

// FlushDB removes all keys from the current database
func (c *Cache) FlushDB(ctx context.Context) error {
	return c.client.FlushDB(ctx).Err()
}

// Common cache key generators
func UserKey(userID string) string {
	return fmt.Sprintf("user:%s", userID)
}

func RideKey(rideID string) string {
	return fmt.Sprintf("ride:%s", rideID)
}

func DriverKey(driverID string) string {
	return fmt.Sprintf("driver:%s", driverID)
}

func DriverLocationKey(driverID string) string {
	return fmt.Sprintf("driver:location:%s", driverID)
}

func PromoCodeKey(code string) string {
	return fmt.Sprintf("promo:%s", code)
}

func ReferralKey(code string) string {
	return fmt.Sprintf("referral:%s", code)
}

func RateLimitKey(identifier, action string) string {
	return fmt.Sprintf("ratelimit:%s:%s", action, identifier)
}

func SessionKey(sessionID string) string {
	return fmt.Sprintf("session:%s", sessionID)
}

func AnalyticsKey(metric string) string {
	return fmt.Sprintf("analytics:%s", metric)
}

func SurgePricingKey(lat, lon float64) string {
	return fmt.Sprintf("surge:%.2f:%.2f", lat, lon)
}
