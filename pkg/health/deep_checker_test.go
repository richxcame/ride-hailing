package health_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/richxcame/ride-hailing/pkg/health"
	"github.com/richxcame/ride-hailing/pkg/resilience"
	"github.com/stretchr/testify/assert"
)

func TestNewDeepChecker(t *testing.T) {
	config := health.DefaultDeepCheckerConfig()
	config.Version = "1.0.0"

	checker := health.NewDeepChecker(config)
	assert.NotNil(t, checker)
}

func TestDeepChecker_CheckWithNoDepenencies(t *testing.T) {
	config := health.DefaultDeepCheckerConfig()
	checker := health.NewDeepChecker(config)

	status := checker.Check(context.Background())

	assert.Equal(t, "healthy", status.Status)
	assert.Empty(t, status.Dependencies)
	assert.Empty(t, status.Breakers)
	assert.False(t, status.CheckedAt.IsZero())
}

func TestDeepChecker_AddCircuitBreaker(t *testing.T) {
	config := health.DefaultDeepCheckerConfig()
	checker := health.NewDeepChecker(config)

	// Create a circuit breaker
	breaker := resilience.NewCircuitBreaker(resilience.Settings{
		Name:             "test-breaker",
		FailureThreshold: 5,
	}, nil)

	checker.AddCircuitBreaker("test-service", breaker)

	status := checker.Check(context.Background())

	assert.Len(t, status.Breakers, 1)
	breakerStatus := status.Breakers["test-service"]
	assert.Equal(t, "test-service", breakerStatus.Name)
	assert.Equal(t, "closed", breakerStatus.State)
	assert.True(t, breakerStatus.Allows)
}

func TestDeepChecker_AddEndpoint(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := health.DefaultDeepCheckerConfig()
	config.Timeout = 2 * time.Second
	checker := health.NewDeepChecker(config)

	checker.AddEndpoint("test-service", server.URL)

	status := checker.Check(context.Background())

	assert.Len(t, status.Dependencies, 1)
	depStatus := status.Dependencies["test-service"]
	assert.Equal(t, "test-service", depStatus.Name)
	assert.Equal(t, "healthy", depStatus.Status)
	assert.Contains(t, depStatus.Message, "200")
}

func TestDeepChecker_EndpointUnhealthy(t *testing.T) {
	// Create a test server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := health.DefaultDeepCheckerConfig()
	checker := health.NewDeepChecker(config)

	checker.AddEndpoint("failing-service", server.URL)

	status := checker.Check(context.Background())

	assert.Equal(t, "degraded", status.Status)
	depStatus := status.Dependencies["failing-service"]
	assert.Equal(t, "unhealthy", depStatus.Status)
}

func TestDeepChecker_EndpointTimeout(t *testing.T) {
	// Create a test server that times out
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := health.DefaultDeepCheckerConfig()
	config.Timeout = 50 * time.Millisecond
	checker := health.NewDeepChecker(config)

	checker.AddEndpoint("slow-service", server.URL)

	status := checker.Check(context.Background())

	assert.Equal(t, "degraded", status.Status)
	depStatus := status.Dependencies["slow-service"]
	assert.Equal(t, "unhealthy", depStatus.Status)
	assert.Contains(t, depStatus.Message, "request failed")
}

func TestDeepChecker_CachesResults(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := health.DefaultDeepCheckerConfig()
	config.CacheTTL = 100 * time.Millisecond
	checker := health.NewDeepChecker(config)

	checker.AddEndpoint("cached-service", server.URL)

	// First check
	checker.Check(context.Background())
	assert.Equal(t, 1, callCount)

	// Second check should use cache
	checker.Check(context.Background())
	assert.Equal(t, 1, callCount)

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third check should hit the server again
	checker.Check(context.Background())
	assert.Equal(t, 2, callCount)
}

func TestDeepChecker_IsHealthy(t *testing.T) {
	config := health.DefaultDeepCheckerConfig()
	checker := health.NewDeepChecker(config)

	assert.True(t, checker.IsHealthy())
}

func TestDeepChecker_IsReady(t *testing.T) {
	config := health.DefaultDeepCheckerConfig()
	checker := health.NewDeepChecker(config)

	// Without database/redis, should be ready
	assert.True(t, checker.IsReady())
}

func TestDeepChecker_Handler(t *testing.T) {
	config := health.DefaultDeepCheckerConfig()
	config.Version = "1.0.0"
	checker := health.NewDeepChecker(config)

	handler := checker.Handler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
	assert.Contains(t, w.Body.String(), "1.0.0")
}

func TestDeepChecker_UptimeIncreases(t *testing.T) {
	config := health.DefaultDeepCheckerConfig()
	config.CacheTTL = 1 * time.Millisecond // Very short cache
	checker := health.NewDeepChecker(config)

	status1 := checker.Check(context.Background())
	time.Sleep(50 * time.Millisecond)
	status2 := checker.Check(context.Background())

	assert.True(t, status2.Uptime > status1.Uptime)
}
