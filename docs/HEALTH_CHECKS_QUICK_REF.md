# Health Checks Quick Reference

Quick reference guide for health check implementation and troubleshooting.

## Endpoints

| Endpoint | Purpose | Response Time | K8s Probe |
|----------|---------|---------------|-----------|
| `/healthz` | Basic health | <10ms | ❌ Legacy |
| `/health/live` | Liveness | <10ms | ✅ Liveness + Startup |
| `/health/ready` | Readiness | <100ms | ✅ Readiness |

## Quick Test Commands

```bash
# Test all health endpoints
curl http://localhost:8080/healthz
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready

# Pretty print with jq
curl -s http://localhost:8080/health/ready | jq

# Test specific service in Kubernetes
kubectl exec -n ridehailing rides-service-xxx -- \
  curl -s localhost:8080/health/ready | jq
```

## Implementation Template

```go
// In main.go after router initialization

// Health check endpoints
router.GET("/healthz", common.HealthCheck(serviceName, version))
router.GET("/health/live", common.LivenessProbe(serviceName, version))

// Readiness probe with dependency checks
healthChecks := make(map[string]func() error)

// Add database check (if using database)
healthChecks["database"] = func() error {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    return db.Ping(ctx) // For pgxpool.Pool
    // OR: return db.PingContext(ctx) // For database/sql.DB
}

// Add Redis check (if using Redis)
if redisClient != nil {
    healthChecks["redis"] = func() error {
        ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
        defer cancel()
        return redisClient.Client.Ping(ctx).Err()
    }
}

router.GET("/health/ready", common.ReadinessProbe(serviceName, version, healthChecks))
```

## Kubernetes Probe Configuration

```yaml
# Add to deployment spec under containers
livenessProbe:
  httpGet:
    path: /health/live
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /health/ready
    port: 8080
  initialDelaySeconds: 15
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3

startupProbe:
  httpGet:
    path: /health/live
    port: 8080
  initialDelaySeconds: 0
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 12
```

## Troubleshooting

### Pod Keeps Restarting
```bash
# Check logs
kubectl logs -n ridehailing <pod-name> --previous

# Describe pod
kubectl describe pod -n ridehailing <pod-name>

# Test liveness manually
kubectl exec -n ridehailing <pod-name> -- curl localhost:8080/health/live
```

### Pod Never Ready
```bash
# Check readiness probe
kubectl exec -n ridehailing <pod-name> -- curl localhost:8080/health/ready

# Check dependencies
kubectl get pods -n ridehailing | grep -E 'postgres|redis'

# View logs
kubectl logs -n ridehailing <pod-name>
```

### Slow Health Checks
```bash
# Time the request
time curl http://localhost:8080/health/ready

# Check with detailed output
curl -w "\nTotal: %{time_total}s\n" http://localhost:8080/health/ready

# In Kubernetes
kubectl exec -n ridehailing <pod-name> -- \
  time curl -s localhost:8080/health/ready
```

## Common Errors

### `db.PingContext undefined`
**Problem:** Using wrong method for pgxpool.Pool
**Solution:** Use `db.Ping(ctx)` instead of `db.PingContext(ctx)`

### `connection refused`
**Problem:** Database/Redis not reachable
**Solution:** Check service/port, verify network policies

### `timeout`
**Problem:** Check takes too long
**Solution:** Increase timeout or optimize check

## Service Dependencies

| Service | Database | Redis |
|---------|----------|-------|
| auth | ✅ | ❌ |
| rides | ✅ | ✅ |
| payments | ✅ | ❌ |
| geo | ❌ | ✅ |
| notifications | ✅ | ❌ |
| realtime | ✅ | ✅ |
| mobile | ✅ | ❌ |
| admin | ✅ | ❌ |
| promos | ✅ | ❌ |
| scheduler | ✅ | ❌ |
| analytics | ✅ | ❌ |
| fraud | ✅ | ❌ |
| ml-eta | ✅ | ✅ |

## Advanced Usage

### Cached Checker
```go
import "github.com/richxcame/ride-hailing/pkg/health"

expensiveCheck := func() error {
    // Expensive operation
    return complexValidation()
}

cachedChecker := health.NewCachedChecker(expensiveCheck, 30*time.Second)
healthChecks["expensive"] = cachedChecker.Check
```

### HTTP Endpoint Checker
```go
import "github.com/richxcame/ride-hailing/pkg/health"

healthChecks["stripe"] = health.HTTPEndpointChecker("https://api.stripe.com/v1/health")
```

### Composite Checker
```go
import "github.com/richxcame/ride-hailing/pkg/health"

checker := health.CompositeChecker("external-services", map[string]health.Checker{
    "stripe": health.HTTPEndpointChecker("https://api.stripe.com"),
    "firebase": health.HTTPEndpointChecker("https://fcm.googleapis.com"),
})

healthChecks["external"] = checker
```

## Response Codes

| Status | Code | Meaning |
|--------|------|---------|
| healthy | 200 | All checks passed |
| not ready | 503 | Dependency failed |
| error | 500 | Internal error |

## Monitoring Integration

### Prometheus Metrics
```promql
# Check failures
health_check_failures_total{service="rides-service",check="database"}

# Check duration
health_check_duration_seconds{service="rides-service",check="database"}
```

### Grafana Queries
```promql
# Service availability
up{job="ridehailing",service="rides-service"}

# Health check error rate
rate(health_check_failures_total[5m])
```

## Documentation Links

- [Full Documentation](HEALTH_CHECKS.md)
- [Implementation Summary](health-checks-implementation-summary.md)
- [Error Handling Guide](ERROR_HANDLING.md)
- [Observability Guide](observability.md)

## Contact

For questions or issues:
1. Check [HEALTH_CHECKS.md](HEALTH_CHECKS.md) for detailed docs
2. Review [Troubleshooting section](HEALTH_CHECKS.md#troubleshooting)
3. Check logs: `kubectl logs -n ridehailing <pod-name>`
