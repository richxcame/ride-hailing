package common

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string                   `json:"status"`
	Service   string                   `json:"service"`
	Version   string                   `json:"version"`
	Timestamp string                   `json:"timestamp"`
	Uptime    string                   `json:"uptime,omitempty"`
	Checks    map[string]CheckStatus   `json:"checks,omitempty"`
	Metadata  map[string]interface{}   `json:"metadata,omitempty"`
}

// CheckStatus represents the status of a single health check
type CheckStatus struct {
	Status    string  `json:"status"`
	Message   string  `json:"message,omitempty"`
	Duration  string  `json:"duration,omitempty"`
	Timestamp string  `json:"timestamp"`
}

var (
	startTime = time.Now()
)

// HealthCheck returns a health check handler
func HealthCheck(serviceName, version string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, HealthResponse{
			Status:    "healthy",
			Service:   serviceName,
			Version:   version,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Uptime:    time.Since(startTime).String(),
		})
	}
}

// LivenessProbe returns a simple liveness check
// This endpoint indicates whether the service is running
// It should always return 200 OK unless the service is completely broken
func LivenessProbe(serviceName, version string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, HealthResponse{
			Status:    "alive",
			Service:   serviceName,
			Version:   version,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Uptime:    time.Since(startTime).String(),
		})
	}
}

// ReadinessProbe returns a readiness check with dependency validation
// This endpoint indicates whether the service is ready to accept traffic
// It checks critical dependencies (database, redis, etc.)
func ReadinessProbe(serviceName, version string, checks map[string]func() error) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := "ready"
		checkResults := make(map[string]CheckStatus)
		allHealthy := true
		now := time.Now().UTC()

		// Run checks in parallel for better performance
		type checkResult struct {
			name     string
			err      error
			duration time.Duration
		}

		resultChan := make(chan checkResult, len(checks))
		var wg sync.WaitGroup

		for name, checkFunc := range checks {
			wg.Add(1)
			go func(n string, cf func() error) {
				defer wg.Done()
				start := time.Now()
				err := cf()
				duration := time.Since(start)
				resultChan <- checkResult{name: n, err: err, duration: duration}
			}(name, checkFunc)
		}

		go func() {
			wg.Wait()
			close(resultChan)
		}()

		for result := range resultChan {
			if result.err != nil {
				checkResults[result.name] = CheckStatus{
					Status:    "unhealthy",
					Message:   result.err.Error(),
					Duration:  result.duration.String(),
					Timestamp: now.Format(time.RFC3339),
				}
				status = "not ready"
				allHealthy = false
			} else {
				checkResults[result.name] = CheckStatus{
					Status:    "healthy",
					Duration:  result.duration.String(),
					Timestamp: now.Format(time.RFC3339),
				}
			}
		}

		statusCode := http.StatusOK
		if !allHealthy {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, HealthResponse{
			Status:    status,
			Service:   serviceName,
			Version:   version,
			Timestamp: now.Format(time.RFC3339),
			Uptime:    time.Since(startTime).String(),
			Checks:    checkResults,
		})
	}
}

// HealthCheckWithDeps returns a health check handler with dependency checks
// This is similar to ReadinessProbe but with slightly different status semantics
func HealthCheckWithDeps(serviceName, version string, checks map[string]func() error) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := "healthy"
		checkResults := make(map[string]CheckStatus)
		now := time.Now().UTC()

		// Run checks in parallel
		type checkResult struct {
			name     string
			err      error
			duration time.Duration
		}

		resultChan := make(chan checkResult, len(checks))
		var wg sync.WaitGroup

		for name, checkFunc := range checks {
			wg.Add(1)
			go func(n string, cf func() error) {
				defer wg.Done()
				start := time.Now()
				err := cf()
				duration := time.Since(start)
				resultChan <- checkResult{name: n, err: err, duration: duration}
			}(name, checkFunc)
		}

		go func() {
			wg.Wait()
			close(resultChan)
		}()

		for result := range resultChan {
			if result.err != nil {
				checkResults[result.name] = CheckStatus{
					Status:    "unhealthy",
					Message:   result.err.Error(),
					Duration:  result.duration.String(),
					Timestamp: now.Format(time.RFC3339),
				}
				status = "unhealthy"
			} else {
				checkResults[result.name] = CheckStatus{
					Status:    "healthy",
					Duration:  result.duration.String(),
					Timestamp: now.Format(time.RFC3339),
				}
			}
		}

		statusCode := http.StatusOK
		if status == "unhealthy" {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, HealthResponse{
			Status:    status,
			Service:   serviceName,
			Version:   version,
			Timestamp: now.Format(time.RFC3339),
			Uptime:    time.Since(startTime).String(),
			Checks:    checkResults,
		})
	}
}

// DetailedHealthCheck returns a comprehensive health check with metadata
func DetailedHealthCheck(serviceName, version string, checks map[string]func() error, metadata map[string]interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := "healthy"
		checkResults := make(map[string]CheckStatus)
		now := time.Now().UTC()

		// Run checks in parallel
		type checkResult struct {
			name     string
			err      error
			duration time.Duration
		}

		resultChan := make(chan checkResult, len(checks))
		var wg sync.WaitGroup

		for name, checkFunc := range checks {
			wg.Add(1)
			go func(n string, cf func() error) {
				defer wg.Done()
				start := time.Now()
				err := cf()
				duration := time.Since(start)
				resultChan <- checkResult{name: n, err: err, duration: duration}
			}(name, checkFunc)
		}

		go func() {
			wg.Wait()
			close(resultChan)
		}()

		for result := range resultChan {
			if result.err != nil {
				checkResults[result.name] = CheckStatus{
					Status:    "unhealthy",
					Message:   result.err.Error(),
					Duration:  result.duration.String(),
					Timestamp: now.Format(time.RFC3339),
				}
				status = "degraded" // Use degraded instead of unhealthy for partial failures
			} else {
				checkResults[result.name] = CheckStatus{
					Status:    "healthy",
					Duration:  result.duration.String(),
					Timestamp: now.Format(time.RFC3339),
				}
			}
		}

		statusCode := http.StatusOK
		if status != "healthy" {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, HealthResponse{
			Status:    status,
			Service:   serviceName,
			Version:   version,
			Timestamp: now.Format(time.RFC3339),
			Uptime:    time.Since(startTime).String(),
			Checks:    checkResults,
			Metadata:  metadata,
		})
	}
}
