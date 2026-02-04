# Health Checks

All services expose three endpoints on their HTTP port.

## Endpoints

### `GET /healthz` -- Basic Health

Returns `200` always.

```json
{
  "status": "healthy",
  "service": "rides-service",
  "version": "1.0.0",
  "timestamp": "2025-11-13T10:30:00Z",
  "uptime": "2h15m30s"
}
```

### `GET /health/live` -- Liveness

Returns `200` always (unless the process is wedged).

```json
{
  "status": "alive",
  "service": "rides-service",
  "version": "1.0.0",
  "timestamp": "2025-11-13T10:30:00Z",
  "uptime": "2h15m30s"
}
```

### `GET /health/ready` -- Readiness

Pings every critical dependency. Returns `200` when all pass, `503` when any fail.

**Healthy:**

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

**Unhealthy** (note `status` + per-check `message`):

```json
{
  "status": "not ready",
  "service": "rides-service",
  "checks": {
    "database": {
      "status": "unhealthy",
      "message": "database ping failed: connection refused",
      "duration": "2000ms"
    },
    "redis": {
      "status": "healthy",
      "duration": "5ms"
    }
  }
}
```

## Service Dependency Matrix

| Service | PostgreSQL | Redis |
|---------|:----------:|:-----:|
| auth-service | x | |
| rides-service | x | x |
| payments-service | x | |
| geo-service | | x |
| notifications-service | x | |
| realtime-service | x | x |
| mobile-service | x | |
| admin-service | x | |
| promos-service | x | |
| scheduler-service | x | |
| analytics-service | x | |
| fraud-service | x | |
| ml-eta-service | x | x |

## Kubernetes Probe Config

All services use the same probe template. Adjust port per service.

```yaml
startupProbe:
  httpGet:
    path: /health/live
    port: 8080
  periodSeconds: 5
  failureThreshold: 12        # up to 60 s to start

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
```

## Prometheus Metrics

```
health_check_failures_total{service="<svc>", check="<dep>"}   # counter
health_check_duration_seconds{service="<svc>", check="<dep>"} # histogram
```

### Alert rules (`monitoring/prometheus/alerts.yml`)

```yaml
- alert: ServiceUnhealthy
  expr: up{job="ridehailing"} == 0
  for: 1m
  labels:
    severity: critical

- alert: HighHealthCheckFailureRate
  expr: rate(health_check_failures_total[5m]) > 0.1
  for: 5m
  labels:
    severity: warning
```

## Troubleshooting

| Symptom | Likely cause | First look |
|---------|-------------|------------|
| Pod never Ready | DB or Redis unreachable | `kubectl logs`, then `kubectl exec -- curl localhost:8080/health/ready` |
| CrashLoopBackOff | Liveness failing or app panic | `kubectl logs --previous`, `kubectl describe pod` |
| Slow / timing-out probes | Dependency latency or pool exhaustion | Check `health_check_duration_seconds` metric |
| Intermittent failures | Network jitter or aggressive timeouts | Raise `failureThreshold` or `timeoutSeconds` |
