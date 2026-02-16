package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	redisclient "github.com/richxcame/ride-hailing/pkg/redis"
)

// Manager handles caching operations with JSON serialization
type Manager struct {
	redis *redisclient.Client
}

// NewManager creates a new cache manager
func NewManager(redis *redisclient.Client) *Manager {
	return &Manager{redis: redis}
}

// Get retrieves a cached value and unmarshals it into result
func (m *Manager) Get(ctx context.Context, key string, result interface{}) error {
	data, err := m.redis.GetString(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), result)
}

// Set marshals and caches a value with expiration
func (m *Manager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}

	return m.redis.SetWithExpiration(ctx, key, string(data), ttl)
}

// GetOrSet retrieves from cache or executes fn and caches the result
func (m *Manager) GetOrSet(ctx context.Context, key string, ttl time.Duration, result interface{}, fn func() (interface{}, error)) error {
	// Try cache first
	err := m.Get(ctx, key, result)
	if err == nil {
		return nil // Cache hit
	}

	// Cache miss - execute function
	data, err := fn()
	if err != nil {
		return err
	}

	// Cache the result (non-blocking)
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.Set(cacheCtx, key, data, ttl); err != nil {
			// Log but don't fail
			fmt.Printf("failed to cache key %s: %v\n", key, err)
		}
	}()

	// Marshal the result into the result pointer
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonData, result)
}

// Delete removes a key from cache
func (m *Manager) Delete(ctx context.Context, keys ...string) error {
	return m.redis.Delete(ctx, keys...)
}

// Invalidate removes keys matching a pattern
func (m *Manager) Invalidate(ctx context.Context, pattern string) error {
	// Note: This uses SCAN which is safe for production
	var cursor uint64
	var deletedKeys []string

	for {
		var keys []string
		var err error

		result := m.redis.Scan(ctx, cursor, pattern, 100)
		keys, cursor, err = result.Result()
		if err != nil {
			return fmt.Errorf("failed to scan keys: %w", err)
		}

		if len(keys) > 0 {
			deletedKeys = append(deletedKeys, keys...)
			if err := m.redis.Delete(ctx, keys...); err != nil {
				return fmt.Errorf("failed to delete keys: %w", err)
			}
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}

// CacheKeys defines common cache key patterns
type CacheKeys struct{}

var Keys = CacheKeys{}

// User returns cache key for user data
func (k CacheKeys) User(userID string) string {
	return fmt.Sprintf("user:%s", userID)
}

// Driver returns cache key for driver data
func (k CacheKeys) Driver(driverID string) string {
	return fmt.Sprintf("driver:%s", driverID)
}

// Ride returns cache key for ride data
func (k CacheKeys) Ride(rideID string) string {
	return fmt.Sprintf("ride:%s", rideID)
}

// RideHistory returns cache key for user's ride history
func (k CacheKeys) RideHistory(userID string, offset int) string {
	return fmt.Sprintf("ride_history:%s:offset:%d", userID, offset)
}

// DriverLocation returns cache key for driver location
func (k CacheKeys) DriverLocation(driverID string) string {
	return fmt.Sprintf("driver:location:%s", driverID)
}

// NearbyDrivers returns cache key for nearby drivers search
func (k CacheKeys) NearbyDrivers(latitude, longitude float64, radius float64) string {
	return fmt.Sprintf("nearby_drivers:%.6f:%.6f:%.1f", latitude, longitude, radius)
}

// Wallet returns cache key for wallet balance
func (k CacheKeys) Wallet(userID string) string {
	return fmt.Sprintf("wallet:%s", userID)
}

// PromoCode returns cache key for promo code
func (k CacheKeys) PromoCode(code string) string {
	return fmt.Sprintf("promo:%s", code)
}

// RideType returns cache key for ride type
func (k CacheKeys) RideType(typeID string) string {
	return fmt.Sprintf("ride_type:%s", typeID)
}

// DriverStats returns cache key for driver statistics
func (k CacheKeys) DriverStats(driverID string) string {
	return fmt.Sprintf("driver:stats:%s", driverID)
}

// Analytics returns cache key for analytics data
func (k CacheKeys) Analytics(metric string, params ...string) string {
	key := fmt.Sprintf("analytics:%s", metric)
	for _, p := range params {
		key += ":" + p
	}
	return key
}

// TTL defines common cache TTL durations
type CacheTTL struct{}

var TTL = CacheTTL{}

func (t CacheTTL) Short() time.Duration     { return 5 * time.Minute }
func (t CacheTTL) Medium() time.Duration    { return 15 * time.Minute }
func (t CacheTTL) Long() time.Duration      { return 1 * time.Hour }
func (t CacheTTL) VeryLong() time.Duration  { return 24 * time.Hour }
func (t CacheTTL) Permanent() time.Duration { return 7 * 24 * time.Hour }
