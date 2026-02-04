# Observability Guide

## Stack Overview

| Tool | Role |
|------|------|
| **OpenTelemetry (OTel)** | Trace instrumentation & collection (SDK + Collector) |
| **Grafana Tempo** | Trace storage & querying |
| **Prometheus** | Metrics collection & alerting |
| **Grafana** | Unified visualization (traces, metrics, logs) |
| **Zap** | Structured JSON logging with trace correlation |
| **Sentry** | Error tracking & crash reporting |

```
Service --> OTel SDK --> OTLP --> OTel Collector --> Tempo
                                                --> Prometheus
        --> Sentry SDK --> Sentry
        --> Zap Logger --> Structured Logs
                                  |
                    Grafana (visualization)
                    Sentry  (error aggregation)
```

---

## Environment Variables

### Per-Service Tracing

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `OTEL_ENABLED` | Enable/disable tracing | `false` | `true` |
| `OTEL_SERVICE_NAME` | Service name | - | `rides-service` |
| `OTEL_SERVICE_VERSION` | Service version | - | `1.0.0` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Collector endpoint | `localhost:4317` | `otel-collector:4317` |
| `OTEL_TRACE_SAMPLE_RATE` | Override sample rate | env-based | `0.1` |

### Sentry

| Variable | Description | Example |
|----------|-------------|---------|
| `SENTRY_DSN` | Project DSN (required) | `https://key@o0.ingest.sentry.io/id` |
| `SENTRY_ENVIRONMENT` | Environment tag | `production` |
| `SENTRY_RELEASE` | Release version | `1.0.0` |
| `SENTRY_SAMPLE_RATE` | Error sample rate (0.0-1.0) | `1.0` |
| `SENTRY_TRACES_SAMPLE_RATE` | Trace sample rate | `0.1` |
| `SENTRY_DEBUG` | Enable debug output | `false` |
| `SERVICE_NAME` | Identifies the service | `auth-service` |

Global OTel/Tempo config lives in `deploy/otel-collector.yml` and `deploy/tempo.yml`.

---

## Sampling Strategy

| Environment | OTel Traces | Sentry Errors | Sentry Traces | Notes |
|-------------|-------------|---------------|---------------|-------|
| **development** | 100% | 100% | 100% | Full visibility |
| **staging** | 50% | 100% | 50% | Error spans always sampled |
| **production** | 10% | 100% | 10% | Configurable via `OTEL_TRACE_SAMPLE_RATE`; error spans always sampled; parent-based with trace-ID ratio |

---

## Span Naming (summary)

- **HTTP**: `{METHOD} {ROUTE}` (e.g. `POST /api/rides`)
- **Business logic**: `{OperationName}` (e.g. `CalculateFare`)
- **Database**: `db.{operation}` (e.g. `db.query`)
- **External API**: `{service}.{operation}` (e.g. `ml-eta-service.predict`)

---

## Service-Specific Wiring

Every service follows the same pattern in its `main.go`:

```go
// 1. Initialize tracing
if os.Getenv("OTEL_ENABLED") == "true" {
    tp, err := tracing.InitTracer(tracing.Config{
        ServiceName:    os.Getenv("OTEL_SERVICE_NAME"),
        ServiceVersion: os.Getenv("OTEL_SERVICE_VERSION"),
        Environment:    cfg.Server.Environment,
        OTLPEndpoint:   os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
        Enabled:        true,
    }, logger.Logger())
    if err == nil { defer tp.Shutdown(context.Background()) }
}

// 2. Initialize Sentry
sentryConfig := errors.DefaultSentryConfig()
sentryConfig.ServerName = serviceName
sentryConfig.Release = version
if err := errors.InitSentry(sentryConfig); err == nil {
    defer errors.Flush(2 * time.Second)
}

// 3. Middleware order (matters)
router := gin.New()
router.Use(middleware.RecoveryWithSentry())   // first - catches panics
router.Use(middleware.SentryMiddleware())      // sets up Sentry scope
router.Use(middleware.CorrelationID())
router.Use(middleware.TracingMiddleware(serviceName))
// ... other middleware ...
router.Use(middleware.ErrorHandler())          // last - captures errors
```

### Sentry Error Filtering

Reported: 5xx, panics, 429, DB connection errors, external service failures.
Not reported: 400, 401, 403, 404, 409, other 4xx.

### Log Correlation

Every log line (via Zap) includes `request_id`, `trace_id`, and `span_id`, enabling direct log-to-trace jumps in Grafana.

---

## Metrics

All services expose `/metrics` for Prometheus scraping.

### Common HTTP Metrics

- `http_requests_total` (counter) -- labels: `method`, `endpoint`, `status`
- `http_request_duration_seconds` (histogram) -- labels: `method`, `endpoint`; buckets: 0.005 .. 10

### Rides-Service Metrics

`rides_created_total`, `rides_completed_total`, `rides_cancelled_total`, `ride_duration_seconds`, `ride_distance_meters`

---

## Grafana Dashboards

Located under the **RideHailing** folder; auto-provisioned on startup (auto-refresh: 10s, default range: 1h).

| Dashboard | UID | Key panels |
|-----------|-----|------------|
| **System Overview** | `ridehailing-overview` | req/s by service, P95/P99 latency, 5xx error rate, status code distribution, CPU/memory/goroutines |
| **Rides Service** | `ridehailing-rides` | rides created/completed/cancelled per hour, cancellation rate, drivers online, ride duration/distance percentiles, rides by type |
| **Payments Service** | `ridehailing-payments` | revenue/hour, payment success/failure rate, processing duration percentiles, payment method distribution, refund counts, failure reasons |

---

## Alert Rules

Defined in `monitoring/prometheus/alerts.yml`.

### System Alerts
| Alert | Condition |
|-------|-----------|
| HighErrorRate | 5xx > 5% for 5 min |
| HighLatency | P99 > 1s for 5 min |
| ServiceDown | Unavailable > 1 min |
| HighCPUUsage | CPU > 85% for 10 min |
| HighMemoryUsage | Memory > 85% for 10 min |
| TooManyGoroutines | > 5000 for 5 min |

### Business Alerts
| Alert | Condition |
|-------|-----------|
| LowDriverAvailability | < 10 drivers online for 5 min |
| HighRideCancellationRate | > 30% for 10 min |
| HighPaymentFailureRate | > 10% for 5 min |
| NoRidesCreated | 0 rides for 10 min |
| RevenueDrop | > 50% drop vs same time yesterday |

### Infrastructure Alerts
| Alert | Condition |
|-------|-----------|
| DatabaseConnectionPoolExhaustion | > 90% pool for 5 min |
| SlowDatabaseQueries | P95 > 1s for 5 min |
| LowRedisHitRate | < 70% for 10 min |
| HighRedisMemoryUsage | > 90% for 5 min |
| CircuitBreakerOpen | Open > 2 min |
| HighCircuitBreakerFailureRate | > 10 failures/s for 5 min |
| HighRateLimitRejectionRate | > 10% for 5 min |

### Fraud Alerts
| Alert | Condition |
|-------|-----------|
| HighFraudDetectionRate | > 5 alerts/s for 5 min |
| BlockedUserSpike | 3x increase vs 1h ago |

Severity levels: **critical** (service down, payment failures) and **warning** (resource pressure, low availability).

---

## Local Access

| Tool | URL |
|------|-----|
| Grafana | http://localhost:3000 (admin/admin) |
| Prometheus | http://localhost:9090 |
| OTel Collector health | http://localhost:13133 |
| Tempo HTTP API | http://localhost:3200 |
