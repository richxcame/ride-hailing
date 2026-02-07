package redis

import (
	"context"
	"time"
)

// ClientInterface defines the interface for Redis operations
type ClientInterface interface {
	SetWithExpiration(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	GetString(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)
	Close() error

	// Batch operations
	MGet(ctx context.Context, keys ...string) ([]interface{}, error)
	MGetStrings(ctx context.Context, keys ...string) ([]string, error)

	// Geospatial operations
	GeoAdd(ctx context.Context, key string, longitude, latitude float64, member string) error
	GeoRadius(ctx context.Context, key string, longitude, latitude, radiusKm float64, count int) ([]string, error)
	GeoRemove(ctx context.Context, key string, member string) error

	// Expiration
	Expire(ctx context.Context, key string, expiration time.Duration) error
}

// Ensure Client implements ClientInterface
var _ ClientInterface = (*Client)(nil)
