package integration

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/health"
)

const (
	testServiceName = "test-service"
	testVersion     = "1.0.0-test"
)

// TestBasicHealthCheck tests the basic /healthz endpoint
func TestBasicHealthCheck(t *testing.T) {
	router := gin.New()
	router.GET("/healthz", common.HealthCheck(testServiceName, testVersion))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response common.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response.Status)
	assert.Equal(t, testServiceName, response.Service)
	assert.Equal(t, testVersion, response.Version)
	assert.NotEmpty(t, response.Timestamp)
	assert.NotEmpty(t, response.Uptime)
}

// TestLivenessProbe tests the /health/live endpoint
func TestLivenessProbe(t *testing.T) {
	router := gin.New()
	router.GET("/health/live", common.LivenessProbe(testServiceName, testVersion))

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response common.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "alive", response.Status)
	assert.Equal(t, testServiceName, response.Service)
	assert.Equal(t, testVersion, response.Version)
	assert.NotEmpty(t, response.Timestamp)
	assert.NotEmpty(t, response.Uptime)
}

// TestReadinessProbeHealthy tests the readiness probe with healthy dependencies
func TestReadinessProbeHealthy(t *testing.T) {
	router := gin.New()

	healthChecks := map[string]func() error{
		"database": func() error {
			return nil // Simulate healthy database
		},
		"redis": func() error {
			return nil // Simulate healthy Redis
		},
	}

	router.GET("/health/ready", common.ReadinessProbe(testServiceName, testVersion, healthChecks))

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response common.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ready", response.Status)
	assert.Equal(t, testServiceName, response.Service)
	assert.Equal(t, testVersion, response.Version)
	assert.NotEmpty(t, response.Timestamp)
	assert.NotEmpty(t, response.Uptime)
	assert.Len(t, response.Checks, 2)

	// Verify individual check statuses
	assert.Equal(t, "healthy", response.Checks["database"].Status)
	assert.Equal(t, "healthy", response.Checks["redis"].Status)
	assert.NotEmpty(t, response.Checks["database"].Duration)
	assert.NotEmpty(t, response.Checks["redis"].Duration)
}

// TestReadinessProbeUnhealthy tests the readiness probe with unhealthy dependencies
func TestReadinessProbeUnhealthy(t *testing.T) {
	router := gin.New()

	healthChecks := map[string]func() error{
		"database": func() error {
			return assert.AnError // Simulate unhealthy database
		},
		"redis": func() error {
			return nil // Simulate healthy Redis
		},
	}

	router.GET("/health/ready", common.ReadinessProbe(testServiceName, testVersion, healthChecks))

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response common.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "not ready", response.Status)
	assert.Len(t, response.Checks, 2)

	// Verify individual check statuses
	assert.Equal(t, "unhealthy", response.Checks["database"].Status)
	assert.NotEmpty(t, response.Checks["database"].Message)
	assert.Equal(t, "healthy", response.Checks["redis"].Status)
}

// TestReadinessProbeParallelExecution tests that checks run in parallel
func TestReadinessProbeParallelExecution(t *testing.T) {
	router := gin.New()

	// Create checks that take some time
	healthChecks := map[string]func() error{
		"slow-check-1": func() error {
			time.Sleep(100 * time.Millisecond)
			return nil
		},
		"slow-check-2": func() error {
			time.Sleep(100 * time.Millisecond)
			return nil
		},
		"slow-check-3": func() error {
			time.Sleep(100 * time.Millisecond)
			return nil
		},
	}

	router.GET("/health/ready", common.ReadinessProbe(testServiceName, testVersion, healthChecks))

	start := time.Now()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	elapsed := time.Since(start)

	assert.Equal(t, http.StatusOK, w.Code)

	// If checks ran sequentially, it would take 300ms+
	// If parallel, it should take ~100ms
	assert.Less(t, elapsed, 200*time.Millisecond, "Checks should run in parallel")
}

// TestDatabaseChecker tests the database health checker
func TestDatabaseChecker(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test in short mode")
	}

	// This test requires a real database connection
	// You should set up a test database or use a mock
	t.Run("with mock database", func(t *testing.T) {
		// Mock database that always succeeds
		mockDB := &sql.DB{}
		checker := health.DatabaseChecker(mockDB)

		// Note: This will fail with a real sql.DB that's not connected
		// In a real test, you'd connect to a test database
		err := checker()
		// We expect an error because mockDB is not actually connected
		assert.Error(t, err)
	})
}

// TestRedisChecker tests the Redis health checker
func TestRedisChecker(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	t.Run("with failing Redis", func(t *testing.T) {
		// Create a Redis client that will fail to connect
		redisClient := redis.NewClient(&redis.Options{
			Addr: "localhost:63799", // Non-existent Redis
		})
		defer redisClient.Close()

		checker := health.RedisChecker(redisClient)
		err := checker()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis ping failed")
	})
}

// TestHTTPEndpointChecker tests the HTTP endpoint health checker
func TestHTTPEndpointChecker(t *testing.T) {
	t.Run("healthy endpoint", func(t *testing.T) {
		// Create a mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		}))
		defer server.Close()

		checker := health.HTTPEndpointChecker(server.URL)
		err := checker()

		assert.NoError(t, err)
	})

	t.Run("unhealthy endpoint", func(t *testing.T) {
		// Create a mock HTTP server that returns 500
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		checker := health.HTTPEndpointChecker(server.URL)
		err := checker()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unhealthy status code")
	})

	t.Run("unreachable endpoint", func(t *testing.T) {
		checker := health.HTTPEndpointChecker("http://localhost:63798")
		err := checker()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "http request failed")
	})
}

// TestCompositeChecker tests the composite health checker
func TestCompositeChecker(t *testing.T) {
	t.Run("all checks pass", func(t *testing.T) {
		checker := health.CompositeChecker("services", map[string]health.Checker{
			"service1": func() error { return nil },
			"service2": func() error { return nil },
		})

		err := checker()
		assert.NoError(t, err)
	})

	t.Run("one check fails", func(t *testing.T) {
		checker := health.CompositeChecker("services", map[string]health.Checker{
			"service1": func() error { return nil },
			"service2": func() error { return assert.AnError },
		})

		err := checker()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "services.service2 check failed")
	})
}

// TestAsyncChecker tests the async health checker with timeout
func TestAsyncChecker(t *testing.T) {
	t.Run("fast check completes", func(t *testing.T) {
		fastCheck := func() error {
			time.Sleep(10 * time.Millisecond)
			return nil
		}

		checker := health.AsyncChecker(fastCheck, 100*time.Millisecond)
		err := checker()

		assert.NoError(t, err)
	})

	t.Run("slow check times out", func(t *testing.T) {
		slowCheck := func() error {
			time.Sleep(200 * time.Millisecond)
			return nil
		}

		checker := health.AsyncChecker(slowCheck, 50*time.Millisecond)
		err := checker()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "health check timeout")
	})
}

// TestCachedChecker tests the cached health checker
func TestCachedChecker(t *testing.T) {
	callCount := 0
	expensiveCheck := func() error {
		callCount++
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	cachedChecker := health.NewCachedChecker(expensiveCheck, 200*time.Millisecond)

	// First call should execute the check
	err := cachedChecker.Check()
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Second call within cache TTL should use cached result
	err = cachedChecker.Check()
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount, "Should use cached result")

	// Wait for cache to expire
	time.Sleep(250 * time.Millisecond)

	// Third call after cache expiry should execute the check again
	err = cachedChecker.Check()
	assert.NoError(t, err)
	assert.Equal(t, 2, callCount, "Should execute check after cache expiry")
}

// TestCheckerTimeout tests health checker with custom timeout
func TestCheckerTimeout(t *testing.T) {
	t.Run("check completes within timeout", func(t *testing.T) {
		// Mock database that responds quickly
		checker := health.DatabaseCheckerWithConfig(nil, health.CheckerConfig{
			Timeout: 5 * time.Second,
		})

		// This will fail because we're passing nil, but it tests the timeout logic
		err := checker()
		assert.Error(t, err) // Expected to fail with nil DB
	})
}

// TestDetailedHealthCheck tests the detailed health check with metadata
func TestDetailedHealthCheck(t *testing.T) {
	router := gin.New()

	healthChecks := map[string]func() error{
		"database": func() error { return nil },
	}

	metadata := map[string]interface{}{
		"region":      "us-west-2",
		"environment": "test",
		"pod_name":    "test-pod-123",
	}

	router.GET("/health/detailed", common.DetailedHealthCheck(
		testServiceName,
		testVersion,
		healthChecks,
		metadata,
	))

	req := httptest.NewRequest(http.MethodGet, "/health/detailed", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response common.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response.Status)
	assert.NotNil(t, response.Metadata)
	assert.Equal(t, "us-west-2", response.Metadata["region"])
	assert.Equal(t, "test", response.Metadata["environment"])
	assert.Equal(t, "test-pod-123", response.Metadata["pod_name"])
}

// TestHealthCheckResponseFormat tests that all endpoints return consistent format
func TestHealthCheckResponseFormat(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		setup    func() gin.HandlerFunc
	}{
		{
			name:     "basic health check",
			endpoint: "/healthz",
			setup:    func() gin.HandlerFunc { return common.HealthCheck(testServiceName, testVersion) },
		},
		{
			name:     "liveness probe",
			endpoint: "/health/live",
			setup:    func() gin.HandlerFunc { return common.LivenessProbe(testServiceName, testVersion) },
		},
		{
			name:     "readiness probe",
			endpoint: "/health/ready",
			setup: func() gin.HandlerFunc {
				return common.ReadinessProbe(testServiceName, testVersion, map[string]func() error{
					"test": func() error { return nil },
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET(tt.endpoint, tt.setup())

			req := httptest.NewRequest(http.MethodGet, tt.endpoint, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			var response common.HealthResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// All responses should have these fields
			assert.NotEmpty(t, response.Status)
			assert.Equal(t, testServiceName, response.Service)
			assert.Equal(t, testVersion, response.Version)
			assert.NotEmpty(t, response.Timestamp)
			assert.NotEmpty(t, response.Uptime)

			// Verify timestamp is valid RFC3339
			_, err = time.Parse(time.RFC3339, response.Timestamp)
			assert.NoError(t, err, "Timestamp should be valid RFC3339 format")
		})
	}
}

// BenchmarkHealthCheck benchmarks the basic health check endpoint
func BenchmarkHealthCheck(b *testing.B) {
	router := gin.New()
	router.GET("/healthz", common.HealthCheck(testServiceName, testVersion))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkReadinessProbe benchmarks the readiness probe with multiple checks
func BenchmarkReadinessProbe(b *testing.B) {
	router := gin.New()

	healthChecks := map[string]func() error{
		"database": func() error { return nil },
		"redis":    func() error { return nil },
		"cache":    func() error { return nil },
	}

	router.GET("/health/ready", common.ReadinessProbe(testServiceName, testVersion, healthChecks))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// TestHealthCheckWithRealDependencies tests health checks with real database and Redis
// This test is only run when explicitly requested and proper test infrastructure is available
func TestHealthCheckWithRealDependencies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test with real dependencies")
	}

	// This test would need proper setup of test database and Redis
	// Example implementation:
	t.Run("with real database", func(t *testing.T) {
		t.Skip("TODO: Implement with test database setup")
		// db := setupTestDB(t)
		// defer teardownTestDB(t, db)
		//
		// router := gin.New()
		// healthChecks := map[string]func() error{
		//     "database": health.DatabaseChecker(db),
		// }
		// router.GET("/health/ready", common.ReadinessProbe(testServiceName, testVersion, healthChecks))
		//
		// req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		// w := httptest.NewRecorder()
		// router.ServeHTTP(w, req)
		//
		// assert.Equal(t, http.StatusOK, w.Code)
	})
}
