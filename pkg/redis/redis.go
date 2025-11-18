package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/richxcame/ride-hailing/pkg/config"
)

// Client wraps the Redis client
type Client struct {
	*redis.Client
}

// NewRedisClient creates a new Redis client with comprehensive timeout and connection pool configuration
// If readTimeout or writeTimeout are not provided or zero, uses config.DefaultRedisReadTimeoutDuration and config.DefaultRedisWriteTimeoutDuration
func NewRedisClient(cfg *config.RedisConfig, timeouts ...time.Duration) (*Client, error) {
	var readTimeout, writeTimeout time.Duration

	if len(timeouts) >= 1 && timeouts[0] > 0 {
		readTimeout = timeouts[0]
	} else {
		readTimeout = config.DefaultRedisReadTimeoutDuration()
	}

	if len(timeouts) >= 2 && timeouts[1] > 0 {
		writeTimeout = timeouts[1]
	} else {
		writeTimeout = config.DefaultRedisWriteTimeoutDuration()
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.Password,
		DB:       cfg.DB,

		// Timeout configuration
		DialTimeout:  5 * time.Second, // Time to establish connection
		ReadTimeout:  readTimeout,      // Time to read response
		WriteTimeout: writeTimeout,     // Time to write request

		// Connection pool configuration
		PoolSize:     10,                   // Maximum number of socket connections
		MinIdleConns: 5,                    // Minimum idle connections to keep warm
		PoolTimeout:  readTimeout + 1*time.Second, // Time to wait for connection from pool

		// Connection lifecycle
		ConnMaxIdleTime: 5 * time.Minute, // Close idle connections after this duration
		ConnMaxLifetime: 1 * time.Hour,   // Force connection closure and recreation after this duration

		// Retry configuration
		MaxRetries:      3,               // Max number of retries for failed commands
		MinRetryBackoff: 8 * time.Millisecond,  // Minimum backoff between retries
		MaxRetryBackoff: 512 * time.Millisecond, // Maximum backoff between retries
	})

	pingTimeout := readTimeout
	if writeTimeout > readTimeout {
		pingTimeout = writeTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("unable to connect to redis: %w", err)
	}

	return &Client{Client: client}, nil
}

// SetWithExpiration sets a key-value pair with expiration
func (c *Client) SetWithExpiration(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.Set(ctx, key, value, expiration).Err()
}

// GetString gets a string value by key
func (c *Client) GetString(ctx context.Context, key string) (string, error) {
	return c.Get(ctx, key).Result()
}

// Delete deletes a key
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	return c.Del(ctx, keys...).Err()
}

// Exists checks if a key exists
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// Close closes the Redis client
func (c *Client) Close() error {
	return c.Client.Close()
}

// GeoAdd adds a location to a geospatial index
func (c *Client) GeoAdd(ctx context.Context, key string, longitude, latitude float64, member string) error {
	return c.Client.GeoAdd(ctx, key, &redis.GeoLocation{
		Longitude: longitude,
		Latitude:  latitude,
		Name:      member,
	}).Err()
}

// GeoRadius searches for members within a radius
func (c *Client) GeoRadius(ctx context.Context, key string, longitude, latitude, radiusKm float64, count int) ([]string, error) {
	result, err := c.Client.GeoRadius(ctx, key, longitude, latitude, &redis.GeoRadiusQuery{
		Radius:      radiusKm,
		Unit:        "km",
		WithCoord:   false,
		WithDist:    true,
		WithGeoHash: false,
		Count:       count,
		Sort:        "ASC", // Sort by distance ascending
	}).Result()

	if err != nil {
		return nil, err
	}

	var members []string
	for _, loc := range result {
		members = append(members, loc.Name)
	}

	return members, nil
}

// GeoRemove removes a member from geospatial index
func (c *Client) GeoRemove(ctx context.Context, key string, member string) error {
	return c.Client.ZRem(ctx, key, member).Err()
}

// GeoPos gets the position of a member
func (c *Client) GeoPos(ctx context.Context, key string, member string) (longitude, latitude float64, err error) {
	result, err := c.Client.GeoPos(ctx, key, member).Result()
	if err != nil {
		return 0, 0, err
	}

	if len(result) == 0 || result[0] == nil {
		return 0, 0, fmt.Errorf("member not found")
	}

	return result[0].Longitude, result[0].Latitude, nil
}

// GeoDist calculates distance between two members
func (c *Client) GeoDist(ctx context.Context, key, member1, member2 string) (float64, error) {
	result, err := c.Client.GeoDist(ctx, key, member1, member2, "km").Result()
	if err != nil {
		return 0, err
	}

	return result, nil
}

// RPush appends one or more values to a list
func (c *Client) RPush(ctx context.Context, key string, values ...interface{}) error {
	return c.Client.RPush(ctx, key, values...).Err()
}

// LRange retrieves a range of elements from a list
func (c *Client) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.Client.LRange(ctx, key, start, stop).Result()
}

// Expire sets an expiration on a key
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.Client.Expire(ctx, key, expiration).Err()
}

// Cache wraps caching operations with JSON serialization
func (c *Client) Cache(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	// Try to get from cache first
	result, err := c.GetString(ctx, key)
	if err == nil {
		return result, nil
	}

	// If not in cache, execute function
	data, err := fn()
	if err != nil {
		return nil, err
	}

	// Store in cache
	if err := c.SetWithExpiration(ctx, key, data, ttl); err != nil {
		// Log error but don't fail - cache is not critical
		fmt.Printf("failed to cache key %s: %v\n", key, err)
	}

	return data, nil
}

// MGet retrieves multiple keys at once
func (c *Client) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	return c.Client.MGet(ctx, keys...).Result()
}

// MSet sets multiple key-value pairs at once
func (c *Client) MSet(ctx context.Context, values map[string]interface{}) error {
	pairs := make([]interface{}, 0, len(values)*2)
	for k, v := range values {
		pairs = append(pairs, k, v)
	}
	return c.Client.MSet(ctx, pairs...).Err()
}

// Incr increments a counter
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	return c.Client.Incr(ctx, key).Result()
}

// Decr decrements a counter
func (c *Client) Decr(ctx context.Context, key string) (int64, error) {
	return c.Client.Decr(ctx, key).Result()
}

// HSet sets a field in a hash
func (c *Client) HSet(ctx context.Context, key, field string, value interface{}) error {
	return c.Client.HSet(ctx, key, field, value).Err()
}

// HGet gets a field from a hash
func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	return c.Client.HGet(ctx, key, field).Result()
}

// HGetAll gets all fields from a hash
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.Client.HGetAll(ctx, key).Result()
}

// HMSet sets multiple fields in a hash
func (c *Client) HMSet(ctx context.Context, key string, values map[string]interface{}) error {
	return c.Client.HMSet(ctx, key, values).Err()
}

// HDel deletes fields from a hash
func (c *Client) HDel(ctx context.Context, key string, fields ...string) error {
	return c.Client.HDel(ctx, key, fields...).Err()
}

// SAdd adds members to a set
func (c *Client) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return c.Client.SAdd(ctx, key, members...).Err()
}

// SMembers gets all members of a set
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.Client.SMembers(ctx, key).Result()
}

// SRem removes members from a set
func (c *Client) SRem(ctx context.Context, key string, members ...interface{}) error {
	return c.Client.SRem(ctx, key, members...).Err()
}

// SIsMember checks if member is in set
func (c *Client) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return c.Client.SIsMember(ctx, key, member).Result()
}

// ZAdd adds members to a sorted set with scores
func (c *Client) ZAdd(ctx context.Context, key string, score float64, member interface{}) error {
	return c.Client.ZAdd(ctx, key, redis.Z{Score: score, Member: member}).Err()
}

// ZRange gets a range from sorted set (by rank)
func (c *Client) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.Client.ZRange(ctx, key, start, stop).Result()
}

// ZRangeByScore gets members by score range
func (c *Client) ZRangeByScore(ctx context.Context, key string, min, max float64) ([]string, error) {
	return c.Client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%f", min),
		Max: fmt.Sprintf("%f", max),
	}).Result()
}

// ZRem removes members from sorted set
func (c *Client) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return c.Client.ZRem(ctx, key, members...).Err()
}

// Pipeline creates a pipeline for batching commands
func (c *Client) Pipeline() redis.Pipeliner {
	return c.Client.Pipeline()
}

// TxPipeline creates a transactional pipeline
func (c *Client) TxPipeline() redis.Pipeliner {
	return c.Client.TxPipeline()
}
