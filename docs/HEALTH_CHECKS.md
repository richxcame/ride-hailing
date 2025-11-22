# Health Checks Guide

This document provides comprehensive information about health checks in the ride-hailing platform.

## Overview

Health checks are critical for monitoring service availability and ensuring proper orchestration in Kubernetes environments. Our platform implements a three-tier health check system:

1. **Liveness Probes** - Determines if a service is alive and should be restarted
2. **Readiness Probes** - Determines if a service can accept traffic
3. **Startup Probes** - Gives services time to start before liveness checks begin

## Health Check Endpoints

All services expose three health check endpoints:

### 1. `/healthz` - Basic Health Check

Simple health check that returns the service status.

**Response Example:**
```json
{
  "status": "healthy",
  "service": "rides-service",
  "version": "1.0.0",
  "timestamp": "2025-11-13T10:30:00Z",
  "uptime": "2h15m30s"
}
```

**HTTP Status Codes:**
- `200 OK` - Service is healthy

**Use Case:** General health monitoring, uptime checks

---

### 2. `/health/live` - Liveness Probe

Indicates whether the service process is alive. This endpoint should only fail if the service is completely broken and needs to be restarted.

**Response Example:**
```json
{
  "status": "alive",
  "service": "rides-service",
  "version": "1.0.0",
  "timestamp": "2025-11-13T10:30:00Z",
  "uptime": "2h15m30s"
}
```

**HTTP Status Codes:**
- `200 OK` - Service is alive

**Use Case:** Kubernetes liveness probe, startup probe

**When it fails:**
- Service process is deadlocked
- Critical panic/crash condition
- Memory exhausted

---

### 3. `/health/ready` - Readiness Probe

Indicates whether the service is ready to accept traffic by checking all critical dependencies.

**Response Example (Healthy):**
```json
{
  "status": "ready",
  "service": "rides-service",
  "version": "1.0.0",
  "timestamp": "2025-11-13T10:30:00Z",
  "uptime": "2h15m30s",
  "checks": {
    "database": {
      "status": "healthy",
      "duration": "15ms",
      "timestamp": "2025-11-13T10:30:00Z"
    },
    "redis": {
      "status": "healthy",
      "duration": "5ms",
      "timestamp": "2025-11-13T10:30:00Z"
    }
  }
}
```

**Response Example (Unhealthy):**
```json
{
  "status": "not ready",
  "service": "rides-service",
  "version": "1.0.0",
  "timestamp": "2025-11-13T10:30:00Z",
  "uptime": "2h15m30s",
  "checks": {
    "database": {
      "status": "unhealthy",
      "message": "database ping failed: connection refused",
      "duration": "2000ms",
      "timestamp": "2025-11-13T10:30:00Z"
    },
    "redis": {
      "status": "healthy",
      "duration": "5ms",
      "timestamp": "2025-11-13T10:30:00Z"
    }
  }
}
```

**HTTP Status Codes:**
- `200 OK` - All dependencies are healthy
- `503 Service Unavailable` - One or more dependencies are unhealthy

**Dependencies Checked per Service:**

| Service | Database | Redis | Notes |
|---------|----------|-------|-------|
| auth-service | ✅ | ❌ | PostgreSQL only |
| rides-service | ✅ | ✅ | PostgreSQL + Redis |
| payments-service | ✅ | ❌ | PostgreSQL only |
| geo-service | ❌ | ✅ | Redis only |
| notifications-service | ✅ | ❌ | PostgreSQL only |
| realtime-service | ✅ | ✅ | PostgreSQL + Redis |
| mobile-service | ✅ | ❌ | PostgreSQL only |
| admin-service | ✅ | ❌ | PostgreSQL only |
| promos-service | ✅ | ❌ | PostgreSQL only |
| scheduler-service | ✅ | ❌ | PostgreSQL only |
| analytics-service | ✅ | ❌ | PostgreSQL only |
| fraud-service | ✅ | ❌ | PostgreSQL only |
| ml-eta-service | ✅ | ✅ | PostgreSQL + Redis |

**Use Case:** Kubernetes readiness probe, load balancer health checks

**When it fails:**
- Database connection lost
- Redis connection lost
- Critical external service unavailable
- Connection pool exhausted

---

## Kubernetes Configuration

All services are configured with three types of probes in their Kubernetes manifests:

### Liveness Probe Configuration

```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 8080
    scheme: HTTP
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
  successThreshold: 1
```

**Settings Explained:**
- `initialDelaySeconds: 30` - Wait 30 seconds after container start before first check
- `periodSeconds: 10` - Check every 10 seconds
- `timeoutSeconds: 5` - Request timeout is 5 seconds
- `failureThreshold: 3` - Restart container after 3 consecutive failures
- `successThreshold: 1` - Mark as healthy after 1 success

**Behavior:** If the liveness probe fails 3 times (30 seconds), Kubernetes will restart the pod.

---

### Readiness Probe Configuration

```yaml
readinessProbe:
  httpGet:
    path: /health/ready
    port: 8080
    scheme: HTTP
  initialDelaySeconds: 15
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3
  successThreshold: 1
```

**Settings Explained:**
- `initialDelaySeconds: 15` - Wait 15 seconds after container start before first check
- `periodSeconds: 5` - Check every 5 seconds
- `timeoutSeconds: 3` - Request timeout is 3 seconds
- `failureThreshold: 3` - Mark as not ready after 3 consecutive failures
- `successThreshold: 1` - Mark as ready after 1 success

**Behavior:** If the readiness probe fails, the pod is removed from service endpoints and won't receive traffic until it passes again.

---

### Startup Probe Configuration

```yaml
startupProbe:
  httpGet:
    path: /health/live
    port: 8080
    scheme: HTTP
  initialDelaySeconds: 0
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 12
  successThreshold: 1
```

**Settings Explained:**
- `initialDelaySeconds: 0` - Start checking immediately
- `periodSeconds: 5` - Check every 5 seconds
- `timeoutSeconds: 3` - Request timeout is 3 seconds
- `failureThreshold: 12` - Allow up to 12 failures (60 seconds total)
- `successThreshold: 1` - Mark as started after 1 success

**Behavior:** Gives the service up to 60 seconds to start before the liveness probe takes over. This prevents slow-starting services from being prematurely killed.

---

## Implementation Details

### Health Check Package

Located in `pkg/health/checker.go`, this package provides reusable health check functions:

#### Database Checker

```go
import "github.com/richxcame/ride-hailing/pkg/health"

// Create a database health checker
dbChecker := health.DatabaseChecker(db)

// Or with custom configuration
dbChecker := health.DatabaseCheckerWithConfig(db, health.CheckerConfig{
    Timeout: 2 * time.Second,
})
```

**What it checks:**
- Database connectivity (ping)
- Connection pool has open connections

#### Redis Checker

```go
import "github.com/richxcame/ride-hailing/pkg/health"

// Create a Redis health checker
redisChecker := health.RedisChecker(redisClient)

// Or with custom configuration
redisChecker := health.RedisCheckerWithConfig(redisClient, health.CheckerConfig{
    Timeout: 2 * time.Second,
})
```

**What it checks:**
- Redis connectivity (ping)

#### HTTP Endpoint Checker

```go
import "github.com/richxcame/ride-hailing/pkg/health"

// Check if an external HTTP service is healthy
checker := health.HTTPEndpointChecker("https://api.stripe.com/v1/health")
```

**What it checks:**
- HTTP endpoint is reachable
- Returns 2xx or 3xx status code

#### Advanced Checkers

**Composite Checker** - Combine multiple checks:
```go
checker := health.CompositeChecker("external-services", map[string]health.Checker{
    "stripe": health.HTTPEndpointChecker("https://api.stripe.com"),
    "firebase": health.HTTPEndpointChecker("https://fcm.googleapis.com"),
})
```

**Async Checker** - Run checks with timeout:
```go
checker := health.AsyncChecker(expensiveCheck, 5*time.Second)
```

**Cached Checker** - Cache results to reduce load:
```go
cachedChecker := health.NewCachedChecker(dbChecker, 30*time.Second)
err := cachedChecker.Check()
```

---

### Common Health Handler

Located in `pkg/common/health.go`, this package provides standard health check HTTP handlers:

#### Basic Usage

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/richxcame/ride-hailing/pkg/common"
)

router := gin.Default()

// Basic health check
router.GET("/healthz", common.HealthCheck(serviceName, version))

// Liveness probe
router.GET("/health/live", common.LivenessProbe(serviceName, version))

// Readiness probe with dependency checks
healthChecks := map[string]func() error{
    "database": health.DatabaseChecker(db),
    "redis": health.RedisChecker(redisClient),
}
router.GET("/health/ready", common.ReadinessProbe(serviceName, version, healthChecks))
```

#### Advanced Usage

```go
// Detailed health check with metadata
metadata := map[string]interface{}{
    "region": "us-west-2",
    "environment": "production",
    "pod_name": os.Getenv("POD_NAME"),
}

router.GET("/health/detailed", common.DetailedHealthCheck(
    serviceName,
    version,
    healthChecks,
    metadata,
))
```

---

## Service-Specific Implementation

### Example: Rides Service

[rides-service/main.go:209-229](cmd/rides/main.go#L209-L229)

```go
// Health check endpoints
router.GET("/healthz", common.HealthCheck(serviceName, version))
router.GET("/health/live", common.LivenessProbe(serviceName, version))

// Readiness probe with dependency checks
healthChecks := make(map[string]func() error)
healthChecks["database"] = func() error {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    return db.PingContext(ctx)
}

if redisClient != nil {
    healthChecks["redis"] = func() error {
        ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
        defer cancel()
        return redisClient.Client.Ping(ctx).Err()
    }
}

router.GET("/health/ready", common.ReadinessProbe(serviceName, version, healthChecks))
```

---

## Testing Health Checks

### Manual Testing

```bash
# Test basic health check
curl http://localhost:8080/healthz

# Test liveness probe
curl http://localhost:8080/health/live

# Test readiness probe
curl http://localhost:8080/health/ready

# Get detailed response with jq
curl -s http://localhost:8080/health/ready | jq
```

### Load Testing

```bash
# Stress test health endpoints
hey -n 10000 -c 100 http://localhost:8080/health/ready
```

### Integration Tests

See `test/integration/health_test.go` for automated health check tests.

---

## Monitoring and Alerting

### Prometheus Metrics

Health check results are automatically exported as Prometheus metrics:

```promql
# Health check failures
health_check_failures_total{service="rides-service",check="database"}

# Health check duration
health_check_duration_seconds{service="rides-service",check="database"}
```

### Grafana Dashboards

Health check metrics are visualized in:
- **System Overview Dashboard** - Global service health
- **Service-Specific Dashboards** - Per-service health details

### Alert Rules

Configured in `monitoring/prometheus/alerts.yml`:

```yaml
- alert: ServiceUnhealthy
  expr: up{job="ridehailing"} == 0
  for: 1m
  labels:
    severity: critical
  annotations:
    summary: "Service {{ $labels.service }} is down"

- alert: HighHealthCheckFailureRate
  expr: rate(health_check_failures_total[5m]) > 0.1
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High health check failure rate for {{ $labels.service }}"
```

---

## Troubleshooting

### Common Issues

#### 1. Readiness Probe Keeps Failing

**Symptoms:** Pod never enters Ready state, no traffic is routed to it

**Possible Causes:**
- Database connection issue
- Redis connection issue
- Network policy blocking access
- Service dependencies not ready

**Debugging:**
```bash
# Check pod logs
kubectl logs -n ridehailing rides-service-xxx

# Exec into pod and test manually
kubectl exec -n ridehailing rides-service-xxx -- curl localhost:8080/health/ready

# Check service dependencies
kubectl get pods -n ridehailing | grep -E 'postgres|redis'
```

#### 2. Pod Keeps Restarting

**Symptoms:** Pod constantly restarts (CrashLoopBackOff)

**Possible Causes:**
- Liveness probe failing
- Application panic/crash
- Resource limits too low
- Startup taking longer than 60 seconds

**Debugging:**
```bash
# Check restart count
kubectl get pod -n ridehailing rides-service-xxx

# Check previous pod logs
kubectl logs -n ridehailing rides-service-xxx --previous

# Describe pod to see probe failures
kubectl describe pod -n ridehailing rides-service-xxx
```

#### 3. Slow Health Check Responses

**Symptoms:** Health checks timeout, high latency

**Possible Causes:**
- Database query too slow
- Redis timeout
- Too many checks running in parallel
- Resource contention

**Solutions:**
- Increase probe timeout
- Use cached checkers for expensive checks
- Optimize database connection pool
- Add proper indexing to database tables

#### 4. False Positive Failures

**Symptoms:** Health checks intermittently fail even though service is healthy

**Possible Causes:**
- Network jitter
- Timeouts too aggressive
- Transient database connection issues
- Resource spikes

**Solutions:**
- Increase `failureThreshold` (allow more failures)
- Increase timeout values
- Implement retry logic in health checks
- Use cached checkers to smooth out spikes

---

## Best Practices

### 1. Keep Health Checks Fast

Health checks should complete in milliseconds, not seconds:

✅ **Good:**
```go
healthChecks["database"] = func() error {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    return db.PingContext(ctx) // Simple ping, <50ms
}
```

❌ **Bad:**
```go
healthChecks["database"] = func() error {
    // Complex query taking 500ms+
    _, err := db.Query("SELECT COUNT(*) FROM rides WHERE status = 'active'")
    return err
}
```

### 2. Check Critical Dependencies Only

Only check dependencies that are required for the service to function:

✅ **Good:**
- Database (if service stores data)
- Redis (if service requires caching)
- Message queue (if service processes messages)

❌ **Bad:**
- External APIs that have fallbacks
- Non-critical cache layers
- Optional features

### 3. Use Appropriate Timeouts

Set realistic timeouts based on your environment:

- **Development:** Longer timeouts (5s) for slower machines
- **Production:** Shorter timeouts (2-3s) for faster detection
- **Critical services:** Very short timeouts (1s) for rapid response

### 4. Monitor Health Check Performance

Track health check duration and failure rates:

```go
// Add timing metrics to health checks
start := time.Now()
err := db.PingContext(ctx)
duration := time.Since(start)
metrics.HealthCheckDuration.WithLabelValues("database").Observe(duration.Seconds())
```

### 5. Document Service-Specific Requirements

Each service should document:
- Required dependencies
- Expected startup time
- Known failure scenarios
- Recovery procedures

---

## Performance Considerations

### Health Check Load

With default configuration, each pod generates:
- **Liveness:** 6 requests/minute
- **Readiness:** 12 requests/minute
- **Startup:** Up to 12 requests (only during startup)

For a 3-pod deployment:
- **Total:** ~54 health check requests/minute per service
- **All 13 services:** ~700 health checks/minute

### Optimization Strategies

1. **Use Cached Checkers** for expensive checks:
   ```go
   cachedDBCheck := health.NewCachedChecker(dbChecker, 10*time.Second)
   ```

2. **Parallel Execution** - Health checks run in parallel by default

3. **Connection Pooling** - Reuse database connections

4. **Index Database Tables** - Ensure fast ping queries

---

## Migration Guide

### From Old Health Checks

If you're migrating from the old `/healthz` endpoint:

**Before:**
```go
router.GET("/healthz", func(c *gin.Context) {
    c.JSON(200, gin.H{"status": "ok"})
})
```

**After:**
```go
router.GET("/healthz", common.HealthCheck(serviceName, version))
router.GET("/health/live", common.LivenessProbe(serviceName, version))

healthChecks := map[string]func() error{
    "database": health.DatabaseChecker(db),
}
router.GET("/health/ready", common.ReadinessProbe(serviceName, version, healthChecks))
```

**Kubernetes Manifest Changes:**
```yaml
# Before
livenessProbe:
  httpGet:
    path: /healthz

# After
livenessProbe:
  httpGet:
    path: /health/live
readinessProbe:
  httpGet:
    path: /health/ready
startupProbe:
  httpGet:
    path: /health/live
```

---

## Related Documentation

- [Error Handling Guide](ERROR_HANDLING.md) - Error handling best practices
- [Observability Guide](observability.md) - Monitoring and tracing
- [Kubernetes Deployment](../k8s/) - Deployment configurations

---

## References

- [Kubernetes Liveness, Readiness and Startup Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
- [Health Check Best Practices](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-probes)
- [Microservices Health Check Pattern](https://microservices.io/patterns/observability/health-check-api.html)
