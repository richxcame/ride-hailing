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

// NewRedisClient creates a new Redis client
func NewRedisClient(cfg *config.RedisConfig) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
