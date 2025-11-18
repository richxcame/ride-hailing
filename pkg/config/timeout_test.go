package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeoutConfigDefaults(t *testing.T) {
	// Clear environment variables
	os.Clearenv()

	cfg, err := Load("test-service")
	require.NoError(t, err)
	defer cfg.Close()

	assert.Equal(t, DefaultHTTPClientTimeout, cfg.Timeout.HTTPClientTimeout)
	assert.Equal(t, DefaultDatabaseQueryTimeout, cfg.Timeout.DatabaseQueryTimeout)
	assert.Equal(t, DefaultRedisOperationTimeout, cfg.Timeout.RedisOperationTimeout)
	assert.Equal(t, DefaultRedisReadTimeout, cfg.Timeout.RedisReadTimeout)
	assert.Equal(t, DefaultRedisWriteTimeout, cfg.Timeout.RedisWriteTimeout)
	assert.Equal(t, DefaultWebSocketConnectionTimeout, cfg.Timeout.WebSocketConnectionTimeout)
	assert.Equal(t, DefaultRequestTimeout, cfg.Timeout.DefaultRequestTimeout)
}

func TestTimeoutConfigCustomValues(t *testing.T) {
	// Set custom timeout values
	os.Clearenv()
	os.Setenv("HTTP_CLIENT_TIMEOUT", "60")
	os.Setenv("DB_QUERY_TIMEOUT", "20")
	os.Setenv("REDIS_OPERATION_TIMEOUT", "10")
	os.Setenv("REDIS_READ_TIMEOUT", "8")
	os.Setenv("REDIS_WRITE_TIMEOUT", "12")
	os.Setenv("WS_CONNECTION_TIMEOUT", "120")
	os.Setenv("DEFAULT_REQUEST_TIMEOUT", "45")

	cfg, err := Load("test-service")
	require.NoError(t, err)
	defer cfg.Close()

	assert.Equal(t, 60, cfg.Timeout.HTTPClientTimeout)
	assert.Equal(t, 20, cfg.Timeout.DatabaseQueryTimeout)
	assert.Equal(t, 10, cfg.Timeout.RedisOperationTimeout)
	assert.Equal(t, 8, cfg.Timeout.RedisReadTimeout)
	assert.Equal(t, 12, cfg.Timeout.RedisWriteTimeout)
	assert.Equal(t, 120, cfg.Timeout.WebSocketConnectionTimeout)
	assert.Equal(t, 45, cfg.Timeout.DefaultRequestTimeout)
}

func TestTimeoutConfigMaxValidation(t *testing.T) {
	t.Run("HTTP client timeout exceeds maximum", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("HTTP_CLIENT_TIMEOUT", "999")

		_, err := Load("test-service")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP_CLIENT_TIMEOUT")
		assert.Contains(t, err.Error(), "exceeds maximum")
	})

	t.Run("Database query timeout exceeds maximum", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("DB_QUERY_TIMEOUT", "999")

		_, err := Load("test-service")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DB_QUERY_TIMEOUT")
		assert.Contains(t, err.Error(), "exceeds maximum")
	})

	t.Run("Redis operation timeout exceeds maximum", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("REDIS_OPERATION_TIMEOUT", "999")

		_, err := Load("test-service")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "REDIS_OPERATION_TIMEOUT")
		assert.Contains(t, err.Error(), "exceeds maximum")
	})

	t.Run("Redis read timeout exceeds maximum", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("REDIS_READ_TIMEOUT", "999")

		_, err := Load("test-service")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "REDIS_READ_TIMEOUT")
		assert.Contains(t, err.Error(), "exceeds maximum")
	})

	t.Run("Redis write timeout exceeds maximum", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("REDIS_WRITE_TIMEOUT", "999")

		_, err := Load("test-service")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "REDIS_WRITE_TIMEOUT")
		assert.Contains(t, err.Error(), "exceeds maximum")
	})

	t.Run("WebSocket connection timeout exceeds maximum", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("WS_CONNECTION_TIMEOUT", "999")

		_, err := Load("test-service")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "WS_CONNECTION_TIMEOUT")
		assert.Contains(t, err.Error(), "exceeds maximum")
	})

	t.Run("Default request timeout exceeds maximum", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("DEFAULT_REQUEST_TIMEOUT", "999")

		_, err := Load("test-service")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "DEFAULT_REQUEST_TIMEOUT")
		assert.Contains(t, err.Error(), "exceeds maximum")
	})
}

func TestTimeoutConfigRouteOverrides(t *testing.T) {
	t.Run("should parse valid route overrides", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("ROUTE_TIMEOUT_OVERRIDES", `{"POST:/api/v1/rides": 60, "GET:/api/v1/analytics": 120}`)

		cfg, err := Load("test-service")
		require.NoError(t, err)
		defer cfg.Close()

		assert.Equal(t, 60, cfg.Timeout.RouteOverrides["POST:/api/v1/rides"])
		assert.Equal(t, 120, cfg.Timeout.RouteOverrides["GET:/api/v1/analytics"])
	})

	t.Run("should reject route override exceeding maximum", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("ROUTE_TIMEOUT_OVERRIDES", `{"POST:/api/v1/rides": 999}`)

		_, err := Load("test-service")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "route timeout")
		assert.Contains(t, err.Error(), "exceeds maximum")
	})

	t.Run("should filter out invalid timeout values", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("ROUTE_TIMEOUT_OVERRIDES", `{"POST:/api/v1/rides": 60, "GET:/api/v1/invalid": 0}`)

		cfg, err := Load("test-service")
		require.NoError(t, err)
		defer cfg.Close()

		assert.Equal(t, 60, cfg.Timeout.RouteOverrides["POST:/api/v1/rides"])
		assert.NotContains(t, cfg.Timeout.RouteOverrides, "GET:/api/v1/invalid")
	})

	t.Run("should return error for invalid JSON", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("ROUTE_TIMEOUT_OVERRIDES", `{invalid json}`)

		_, err := Load("test-service")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ROUTE_TIMEOUT_OVERRIDES")
	})
}

func TestTimeoutConfigDurationMethods(t *testing.T) {
	cfg := TimeoutConfig{
		HTTPClientTimeout:          30,
		DatabaseQueryTimeout:       10,
		RedisOperationTimeout:      5,
		RedisReadTimeout:           7,
		RedisWriteTimeout:          8,
		WebSocketConnectionTimeout: 60,
		DefaultRequestTimeout:      45,
	}

	assert.Equal(t, 30*time.Second, cfg.HTTPClientTimeoutDuration())
	assert.Equal(t, 10*time.Second, cfg.DatabaseQueryTimeoutDuration())
	assert.Equal(t, 5*time.Second, cfg.RedisOperationTimeoutDuration())
	assert.Equal(t, 7*time.Second, cfg.RedisReadTimeoutDuration())
	assert.Equal(t, 8*time.Second, cfg.RedisWriteTimeoutDuration())
	assert.Equal(t, 60*time.Second, cfg.WebSocketConnectionTimeoutDuration())
	assert.Equal(t, 45*time.Second, cfg.DefaultRequestTimeoutDuration())
}

func TestTimeoutConfigRedisFallback(t *testing.T) {
	t.Run("RedisReadTimeout falls back to RedisOperationTimeout", func(t *testing.T) {
		cfg := TimeoutConfig{
			RedisOperationTimeout: 10,
			RedisReadTimeout:      0, // Not set
		}

		assert.Equal(t, 10*time.Second, cfg.RedisReadTimeoutDuration())
	})

	t.Run("RedisWriteTimeout falls back to RedisOperationTimeout", func(t *testing.T) {
		cfg := TimeoutConfig{
			RedisOperationTimeout: 10,
			RedisWriteTimeout:     0, // Not set
		}

		assert.Equal(t, 10*time.Second, cfg.RedisWriteTimeoutDuration())
	})
}

func TestTimeoutForRoute(t *testing.T) {
	cfg := TimeoutConfig{
		DefaultRequestTimeout: 30,
		RouteOverrides: map[string]int{
			"POST:/api/v1/rides":           60,
			"GET:/api/v1/analytics/reports": 120,
			"PUT:/api/v1/users/:id":        45,
		},
	}

	t.Run("returns default for non-overridden route", func(t *testing.T) {
		timeout := cfg.TimeoutForRoute("GET", "/api/v1/users")
		assert.Equal(t, 30*time.Second, timeout)
	})

	t.Run("returns override for exact match", func(t *testing.T) {
		timeout := cfg.TimeoutForRoute("POST", "/api/v1/rides")
		assert.Equal(t, 60*time.Second, timeout)
	})

	t.Run("returns default for different method", func(t *testing.T) {
		timeout := cfg.TimeoutForRoute("GET", "/api/v1/rides")
		assert.Equal(t, 30*time.Second, timeout)
	})

	t.Run("returns override for complex route", func(t *testing.T) {
		timeout := cfg.TimeoutForRoute("GET", "/api/v1/analytics/reports")
		assert.Equal(t, 120*time.Second, timeout)
	})

	t.Run("returns default when override is zero", func(t *testing.T) {
		cfg := TimeoutConfig{
			DefaultRequestTimeout: 30,
			RouteOverrides: map[string]int{
				"POST:/api/v1/rides": 0, // Invalid
			},
		}

		timeout := cfg.TimeoutForRoute("POST", "/api/v1/rides")
		assert.Equal(t, 30*time.Second, timeout)
	})
}

func TestDefaultDurationFunctions(t *testing.T) {
	assert.Equal(t, time.Duration(DefaultRedisReadTimeout)*time.Second, DefaultRedisReadTimeoutDuration())
	assert.Equal(t, time.Duration(DefaultRedisWriteTimeout)*time.Second, DefaultRedisWriteTimeoutDuration())
	assert.Equal(t, time.Duration(DefaultHTTPClientTimeout)*time.Second, DefaultHTTPClientTimeoutDuration())
}

func BenchmarkTimeoutForRoute(b *testing.B) {
	cfg := TimeoutConfig{
		DefaultRequestTimeout: 30,
		RouteOverrides: map[string]int{
			"POST:/api/v1/rides":            60,
			"GET:/api/v1/analytics/reports": 120,
			"PUT:/api/v1/users/:id":         45,
		},
	}

	b.Run("default route", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cfg.TimeoutForRoute("GET", "/api/v1/users")
		}
	})

	b.Run("overridden route", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cfg.TimeoutForRoute("POST", "/api/v1/rides")
		}
	})
}
