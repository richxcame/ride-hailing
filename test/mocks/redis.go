package mocks

import (
	"context"
	"time"

	redisClient "github.com/richxcame/ride-hailing/pkg/redis"
	"github.com/stretchr/testify/mock"
)

// MockRedisClient is a mock implementation of the Redis client
type MockRedisClient struct {
	mock.Mock
}

// Ensure MockRedisClient implements ClientInterface
var _ redisClient.ClientInterface = (*MockRedisClient)(nil)

// SetWithExpiration mocks setting a key with expiration
func (m *MockRedisClient) SetWithExpiration(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	args := m.Called(ctx, key, value, expiration)
	return args.Error(0)
}

// GetString mocks getting a string value
func (m *MockRedisClient) GetString(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

// GeoAdd mocks adding a location to a geospatial index
func (m *MockRedisClient) GeoAdd(ctx context.Context, key string, longitude, latitude float64, member string) error {
	args := m.Called(ctx, key, longitude, latitude, member)
	return args.Error(0)
}

// GeoRadius mocks finding members within a radius
func (m *MockRedisClient) GeoRadius(ctx context.Context, key string, longitude, latitude, radius float64, count int) ([]string, error) {
	args := m.Called(ctx, key, longitude, latitude, radius, count)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// GeoRemove mocks removing a member from a geospatial index
func (m *MockRedisClient) GeoRemove(ctx context.Context, key string, member string) error {
	args := m.Called(ctx, key, member)
	return args.Error(0)
}

// Delete mocks deleting a key
func (m *MockRedisClient) Delete(ctx context.Context, keys ...string) error {
	args := m.Called(ctx, keys)
	return args.Error(0)
}

// Exists mocks checking if a key exists
func (m *MockRedisClient) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

// Expire mocks setting expiration on a key
func (m *MockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	args := m.Called(ctx, key, expiration)
	return args.Error(0)
}

// Close mocks closing the client
func (m *MockRedisClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MGet mocks getting multiple keys at once
func (m *MockRedisClient) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	args := m.Called(ctx, keys)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]interface{}), args.Error(1)
}

// MGetStrings mocks getting multiple keys at once as strings
func (m *MockRedisClient) MGetStrings(ctx context.Context, keys ...string) ([]string, error) {
	args := m.Called(ctx, keys)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}
