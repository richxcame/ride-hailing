package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/richxcame/ride-hailing/pkg/resilience"
)

// DependencyStatus represents the health status of a single dependency
type DependencyStatus struct {
	Name      string        `json:"name"`
	Status    string        `json:"status"` // "healthy", "unhealthy", "degraded"
	Latency   time.Duration `json:"latency_ms"`
	Message   string        `json:"message,omitempty"`
	CheckedAt time.Time     `json:"checked_at"`
}

// DeepHealthStatus represents the complete health status of the service
type DeepHealthStatus struct {
	Status       string                      `json:"status"` // "healthy", "unhealthy", "degraded"
	Version      string                      `json:"version,omitempty"`
	Uptime       time.Duration               `json:"uptime_seconds"`
	Dependencies map[string]DependencyStatus `json:"dependencies"`
	Breakers     map[string]BreakerStatus    `json:"circuit_breakers,omitempty"`
	CheckedAt    time.Time                   `json:"checked_at"`
}

// BreakerStatus represents the status of a circuit breaker
type BreakerStatus struct {
	Name   string `json:"name"`
	State  string `json:"state"` // "closed", "half-open", "open"
	Allows bool   `json:"allows_requests"`
}

// DeepChecker performs comprehensive health checks on all dependencies
type DeepChecker struct {
	db          *sql.DB
	redis       *redis.Client
	breakers    map[string]*resilience.CircuitBreaker
	endpoints   map[string]string // name -> URL
	version     string
	startTime   time.Time
	timeout     time.Duration
	mu          sync.RWMutex
	lastResult  *DeepHealthStatus
	cacheTTL    time.Duration
	lastChecked time.Time
}

// DeepCheckerConfig holds configuration for the deep checker
type DeepCheckerConfig struct {
	Version  string
	Timeout  time.Duration
	CacheTTL time.Duration
}

// DefaultDeepCheckerConfig returns sensible defaults
func DefaultDeepCheckerConfig() DeepCheckerConfig {
	return DeepCheckerConfig{
		Version:  "unknown",
		Timeout:  5 * time.Second,
		CacheTTL: 10 * time.Second,
	}
}

// NewDeepChecker creates a new deep health checker
func NewDeepChecker(config DeepCheckerConfig) *DeepChecker {
	return &DeepChecker{
		breakers:  make(map[string]*resilience.CircuitBreaker),
		endpoints: make(map[string]string),
		version:   config.Version,
		startTime: time.Now(),
		timeout:   config.Timeout,
		cacheTTL:  config.CacheTTL,
	}
}

// SetDatabase sets the database connection to check
func (d *DeepChecker) SetDatabase(db *sql.DB) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.db = db
}

// SetRedis sets the Redis client to check
func (d *DeepChecker) SetRedis(client *redis.Client) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.redis = client
}

// AddCircuitBreaker adds a circuit breaker to monitor
func (d *DeepChecker) AddCircuitBreaker(name string, breaker *resilience.CircuitBreaker) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.breakers[name] = breaker
}

// AddEndpoint adds an HTTP endpoint to check
func (d *DeepChecker) AddEndpoint(name, url string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.endpoints[name] = url
}

// Check performs a deep health check on all dependencies
func (d *DeepChecker) Check(ctx context.Context) *DeepHealthStatus {
	d.mu.RLock()
	// Return cached result if still valid
	if d.lastResult != nil && time.Since(d.lastChecked) < d.cacheTTL {
		result := d.lastResult
		d.mu.RUnlock()
		return result
	}
	d.mu.RUnlock()

	// Perform new check
	status := &DeepHealthStatus{
		Status:       "healthy",
		Version:      d.version,
		Uptime:       time.Since(d.startTime),
		Dependencies: make(map[string]DependencyStatus),
		Breakers:     make(map[string]BreakerStatus),
		CheckedAt:    time.Now(),
	}

	// Check all dependencies concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Check database
	if d.db != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			depStatus := d.checkDatabase(ctx)
			mu.Lock()
			status.Dependencies["postgres"] = depStatus
			if depStatus.Status != "healthy" {
				status.Status = "degraded"
			}
			mu.Unlock()
		}()
	}

	// Check Redis
	if d.redis != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			depStatus := d.checkRedis(ctx)
			mu.Lock()
			status.Dependencies["redis"] = depStatus
			if depStatus.Status != "healthy" {
				status.Status = "degraded"
			}
			mu.Unlock()
		}()
	}

	// Check HTTP endpoints
	for name, url := range d.endpoints {
		wg.Add(1)
		go func(name, url string) {
			defer wg.Done()
			depStatus := d.checkHTTPEndpoint(ctx, name, url)
			mu.Lock()
			status.Dependencies[name] = depStatus
			if depStatus.Status == "unhealthy" {
				status.Status = "degraded"
			}
			mu.Unlock()
		}(name, url)
	}

	wg.Wait()

	// Check circuit breakers (synchronous, fast)
	for name, breaker := range d.breakers {
		allows := breaker.Allow()
		state := "closed"
		if !allows {
			state = "open"
			status.Status = "degraded"
		}
		status.Breakers[name] = BreakerStatus{
			Name:   name,
			State:  state,
			Allows: allows,
		}
	}

	// Cache the result
	d.mu.Lock()
	d.lastResult = status
	d.lastChecked = time.Now()
	d.mu.Unlock()

	return status
}

// checkDatabase checks PostgreSQL connectivity
func (d *DeepChecker) checkDatabase(ctx context.Context) DependencyStatus {
	start := time.Now()
	status := DependencyStatus{
		Name:      "postgres",
		CheckedAt: start,
	}

	checkCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	if err := d.db.PingContext(checkCtx); err != nil {
		status.Status = "unhealthy"
		status.Message = fmt.Sprintf("ping failed: %v", err)
		status.Latency = time.Since(start)
		return status
	}

	// Check connection pool
	stats := d.db.Stats()
	if stats.OpenConnections == 0 {
		status.Status = "degraded"
		status.Message = "no open connections in pool"
	} else {
		status.Status = "healthy"
		status.Message = fmt.Sprintf("open=%d, idle=%d, in_use=%d",
			stats.OpenConnections, stats.Idle, stats.InUse)
	}

	status.Latency = time.Since(start)
	return status
}

// checkRedis checks Redis connectivity
func (d *DeepChecker) checkRedis(ctx context.Context) DependencyStatus {
	start := time.Now()
	status := DependencyStatus{
		Name:      "redis",
		CheckedAt: start,
	}

	checkCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	result, err := d.redis.Ping(checkCtx).Result()
	if err != nil {
		status.Status = "unhealthy"
		status.Message = fmt.Sprintf("ping failed: %v", err)
		status.Latency = time.Since(start)
		return status
	}

	status.Status = "healthy"
	status.Message = result
	status.Latency = time.Since(start)
	return status
}

// checkHTTPEndpoint checks an HTTP endpoint health
func (d *DeepChecker) checkHTTPEndpoint(ctx context.Context, name, url string) DependencyStatus {
	start := time.Now()
	status := DependencyStatus{
		Name:      name,
		CheckedAt: start,
	}

	client := &http.Client{
		Timeout: d.timeout,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		status.Status = "unhealthy"
		status.Message = fmt.Sprintf("request creation failed: %v", err)
		status.Latency = time.Since(start)
		return status
	}

	resp, err := client.Do(req)
	if err != nil {
		status.Status = "unhealthy"
		status.Message = fmt.Sprintf("request failed: %v", err)
		status.Latency = time.Since(start)
		return status
	}
	defer resp.Body.Close()

	status.Latency = time.Since(start)

	if resp.StatusCode >= 500 {
		status.Status = "unhealthy"
		status.Message = fmt.Sprintf("status code: %d", resp.StatusCode)
	} else if resp.StatusCode >= 400 {
		status.Status = "degraded"
		status.Message = fmt.Sprintf("status code: %d", resp.StatusCode)
	} else {
		status.Status = "healthy"
		status.Message = fmt.Sprintf("status code: %d", resp.StatusCode)
	}

	return status
}

// Handler returns an HTTP handler for the deep health check endpoint
func (d *DeepChecker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := d.Check(r.Context())

		w.Header().Set("Content-Type", "application/json")

		// Set appropriate HTTP status based on health
		switch status.Status {
		case "healthy":
			w.WriteHeader(http.StatusOK)
		case "degraded":
			w.WriteHeader(http.StatusOK) // Still return 200 for degraded
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		json.NewEncoder(w).Encode(status)
	}
}

// GinHandler returns a Gin handler for the deep health check endpoint
func (d *DeepChecker) GinHandler() func(c interface{ JSON(int, interface{}) }) {
	return func(c interface{ JSON(int, interface{}) }) {
		status := d.Check(context.Background())

		httpStatus := http.StatusOK
		if status.Status == "unhealthy" {
			httpStatus = http.StatusServiceUnavailable
		}

		c.JSON(httpStatus, status)
	}
}

// IsHealthy returns true if the service is healthy or degraded
func (d *DeepChecker) IsHealthy() bool {
	status := d.Check(context.Background())
	return status.Status != "unhealthy"
}

// IsReady returns true if all critical dependencies are healthy
func (d *DeepChecker) IsReady() bool {
	status := d.Check(context.Background())

	// Check database is healthy (critical)
	if dbStatus, ok := status.Dependencies["postgres"]; ok {
		if dbStatus.Status == "unhealthy" {
			return false
		}
	}

	// Check Redis is healthy (critical for real-time features)
	if redisStatus, ok := status.Dependencies["redis"]; ok {
		if redisStatus.Status == "unhealthy" {
			return false
		}
	}

	return true
}
