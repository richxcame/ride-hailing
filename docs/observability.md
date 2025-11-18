# Observability Guide

This guide explains the observability stack for the ride-hailing platform, including distributed tracing, metrics, logging, and error tracking.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Distributed Tracing](#distributed-tracing)
3. [Metrics](#metrics)
4. [Logging](#logging)
5. [Error Tracking](#error-tracking)
6. [Accessing Observability Tools](#accessing-observability-tools)
7. [Adding Custom Instrumentation](#adding-custom-instrumentation)
8. [Troubleshooting](#troubleshooting)

---

## Architecture Overview

The observability stack consists of:

- **OpenTelemetry Collector**: Receives traces from all services and exports to Tempo
- **Grafana Tempo**: Stores and queries distributed traces
- **Prometheus**: Collects and stores metrics from all services
- **Grafana**: Unified visualization dashboard for traces, metrics, and logs
- **Zap Logger**: Structured JSON logging with correlation ID support
- **Sentry**: Error tracking and crash reporting with context

### Data Flow

```
Service → OpenTelemetry SDK → OTLP → OTel Collector → Tempo
                                                     → Prometheus
        → Sentry SDK → Sentry (Error Tracking)
        → Zap Logger → Structured Logs

                                    ↓
                    Grafana (Traces, Metrics, Logs Visualization)
                    Sentry (Error Aggregation & Alerting)
```

---

## Distributed Tracing

### Overview

Distributed tracing with OpenTelemetry enables end-to-end request tracking across all microservices.

### Architecture

- **Instrumentation**: Services use OpenTelemetry SDK to create spans
- **Propagation**: W3C Trace Context headers propagate trace context between services
- **Collection**: OpenTelemetry Collector receives traces via OTLP (gRPC on port 4317, HTTP on port 4318)
- **Storage**: Grafana Tempo stores traces
- **Visualization**: Grafana UI displays traces

### Trace Context Propagation

All HTTP requests include trace context headers:

- `traceparent`: W3C trace parent header with trace ID, span ID, and sampling decision
- `tracestate`: Additional vendor-specific trace state
- `X-Trace-ID`: Response header containing the trace ID for debugging

### Sampling Strategy

**Development Environment** (`ENVIRONMENT=development`):
- **Sample Rate**: 100% (all traces captured)
- **Purpose**: Complete visibility for development and testing

**Staging Environment** (`ENVIRONMENT=staging`):
- **Sample Rate**: 50%
- **Always Sampled**: Error spans (status code 5xx)

**Production Environment** (`ENVIRONMENT=production`):
- **Sample Rate**: 10% (configurable via `OTEL_TRACE_SAMPLE_RATE`)
- **Always Sampled**: Error spans
- **Strategy**: Parent-based sampling with trace ID ratio

### Span Naming Conventions

Spans follow consistent naming conventions:

#### HTTP Spans
- Format: `{HTTP_METHOD} {ROUTE}`
- Examples:
  - `POST /api/rides`
  - `GET /api/rides/:id`
  - `PUT /api/rides/:id/accept`

#### Business Logic Spans
- Format: `{OperationName}`
- Examples:
  - `RequestRide`
  - `AcceptRide`
  - `CompleteRide`
  - `CalculateFare`
  - `ValidatePromoCode`

#### Database Spans
- Format: `db.{operation}`
- Examples:
  - `db.query`
  - `db.insert`
  - `db.update`
  - `db.delete`

#### External API Spans
- Format: `{service}.{operation}` or `HTTP {METHOD}`
- Examples:
  - `promos-service.validateCode`
  - `ml-eta-service.predict`
  - `HTTP POST` (for external services)

### Span Attributes

Standard attributes added to spans:

#### HTTP Attributes
- `http.method`: HTTP method (GET, POST, etc.)
- `http.url`: Full request URL
- `http.route`: Route pattern
- `http.status_code`: HTTP status code
- `http.client_ip`: Client IP address
- `http.user_agent`: User agent string
- `http.request_id`: Correlation ID

#### Business Attributes
- `user.id`: User UUID
- `ride.id`: Ride UUID
- `driver.id`: Driver UUID
- `payment.id`: Payment UUID
- `fare.amount`: Fare amount in currency units
- `distance.meters`: Distance in meters
- `duration.seconds`: Duration in seconds
- `location.lat`: Latitude
- `location.lon`: Longitude

#### Database Attributes
- `db.system`: Database system (e.g., "postgresql")
- `db.operation`: Operation type (SELECT, INSERT, etc.)
- `db.statement`: SQL statement
- `db.rows_affected`: Number of rows affected

### Viewing Traces in Grafana

1. **Access Grafana**: http://localhost:3000
2. **Navigate**: Explore → Select "Tempo" data source
3. **Search Options**:
   - **By Trace ID**: Enter trace ID from `X-Trace-ID` header
   - **By Service**: Filter by service name
   - **By Duration**: Find slow traces
   - **By Status**: Find error traces
   - **By Tags**: Search by span attributes (e.g., `ride.id="xxx"`)

### Example: Complete Ride Request Flow

A typical ride request trace shows:

```
HTTP POST /api/rides (rides-service)
├── RequestRide (rides-service)
│   ├── db.insert (rides-service → postgres)
│   ├── redis.get (rides-service → redis)
│   ├── promos-service.validateCode (rides-service → promos-service)
│   │   └── db.query (promos-service → postgres)
│   └── ml-eta-service.predict (rides-service → ml-eta-service)
│       └── db.query (ml-eta-service → postgres)
└── HTTP POST /api/notifications (rides-service → notifications-service)
    ├── SendNotification (notifications-service)
    │   ├── Firebase.send (notifications-service → Firebase FCM)
    │   └── db.insert (notifications-service → postgres)
```

---

## Metrics

### Overview

Prometheus collects metrics from all services exposed at `/metrics` endpoint.

### Available Metrics

#### HTTP Metrics (per service)
- `http_requests_total`: Total HTTP requests (counter)
  - Labels: `method`, `endpoint`, `status`
- `http_request_duration_seconds`: Request latency (histogram)
  - Labels: `method`, `endpoint`
  - Buckets: 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10

#### Application Metrics
Metrics are service-specific. Example for rides-service:
- `rides_created_total`: Total rides created
- `rides_completed_total`: Total rides completed
- `rides_cancelled_total`: Total rides cancelled
- `ride_duration_seconds`: Ride duration
- `ride_distance_meters`: Ride distance

### Accessing Metrics

- **Prometheus UI**: http://localhost:9090
- **Grafana Dashboards**: http://localhost:3000

### Grafana Dashboards

The platform includes three pre-configured dashboards that automatically load when Grafana starts:

#### 1. System Overview Dashboard
**UID**: `ridehailing-overview`

Provides a high-level view of the entire system:
- **Request Rate**: req/s by service
- **Request Latency**: P95 & P99 latency across all services
- **Error Rate**: 5xx error rate by service
- **Traffic Distribution**: Pie chart showing request distribution
- **Status Code Distribution**: Breakdown of HTTP status codes
- **System Resources**: CPU, memory, goroutines
- **Key Metrics**: Total request rate, global P99 latency, global error rate

**Best for**: Operations team, quick health checks, incident response

#### 2. Rides Service Dashboard
**UID**: `ridehailing-rides`

Business metrics for the rides service:
- **Key Stats**: Rides created/completed/cancelled per hour, cancellation rate
- **Driver Metrics**: Drivers online, active rides, drivers by region
- **Performance**: Ride duration percentiles, ride distance percentiles
- **Business Insights**: Rides by type, cancellation reasons, driver matching time
- **Trends**: Rides per minute over time

**Best for**: Product managers, business analysts, operations

#### 3. Payments Service Dashboard
**UID**: `ridehailing-payments`

Financial and payment metrics:
- **Revenue**: Revenue per hour, revenue trends
- **Success/Failure**: Payment success/failure rates
- **Performance**: Payment processing duration percentiles
- **Payment Methods**: Distribution of payment methods
- **Refunds**: Refund counts and amounts
- **Failure Analysis**: Payment failure reasons
- **Transaction Stats**: Median and P95 payment amounts

**Best for**: Finance team, operations, business analysts

### Accessing Dashboards

1. **Open Grafana**: http://localhost:3000
2. **Login**: admin / admin (change in production)
3. **Navigate**: Dashboards → Browse → RideHailing folder
4. **Select**: Choose from Overview, Rides, or Payments dashboard

Dashboards auto-refresh every 10 seconds and show data from the last hour (configurable).

### Example PromQL Queries

```promql
# Request rate (req/s) per service
rate(http_requests_total[5m])

# 99th percentile latency
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))

# Error rate
rate(http_requests_total{status=~"5.."}[5m])

# Rides per hour
increase(rides_created_total[1h])

# Payment failure rate
rate(payments_failed_total[5m]) / (rate(payments_successful_total[5m]) + rate(payments_failed_total[5m]))

# Revenue per minute
sum(rate(payment_amount_total[5m])) * 60
```

---

## Logging

### Overview

All services use structured logging with Uber's Zap library.

### Log Format

**Development**:
```
2025-11-11T10:30:45.123Z  INFO  rides-service  Starting rides service  {"service": "rides-service", "version": "1.0.0"}
```

**Production** (JSON):
```json
{
  "level": "info",
  "ts": "2025-11-11T10:30:45.123Z",
  "caller": "main.go:50",
  "msg": "Starting rides service",
  "service": "rides-service",
  "version": "1.0.0",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7"
}
```

### Log Levels

- `DEBUG`: Detailed information for debugging
- `INFO`: General informational messages
- `WARN`: Warning messages (recoverable issues)
- `ERROR`: Error messages (handled errors)
- `FATAL`: Critical errors (service will exit)

### Correlation

Logs include:
- `request_id`: Unique request identifier (from `X-Request-ID` header)
- `trace_id`: OpenTelemetry trace ID
- `span_id`: OpenTelemetry span ID

This enables correlation between logs and traces in Grafana.

---

## Error Tracking

### Overview

Sentry provides centralized error tracking and crash reporting with rich context. It automatically captures:
- Unhandled exceptions and panics
- Server errors (5xx status codes)
- Rate limiting errors (429)
- External service failures
- Database errors

### Features

- **Automatic Error Capture**: Middleware automatically captures errors and panics
- **Context Enrichment**: Includes request details, user info, tags, and breadcrumbs
- **Error Grouping**: Similar errors are grouped together
- **Release Tracking**: Track errors by release version
- **Performance Monitoring**: Optional performance tracing integration
- **Alerting**: Configure alerts for error rate thresholds

### Configuration

#### Environment Variables

```bash
# Required
SENTRY_DSN=https://your-key@o0.ingest.sentry.io/project-id

# Optional
SENTRY_ENVIRONMENT=production  # or: development, staging
SENTRY_RELEASE=1.0.0
SENTRY_SAMPLE_RATE=1.0  # 0.0 to 1.0 (100% = all errors)
SENTRY_TRACES_SAMPLE_RATE=0.1  # 10% of traces
SENTRY_DEBUG=false
SERVICE_NAME=auth-service
```

#### Sample Rates by Environment

- **Development**: 100% errors, 100% traces
- **Staging**: 100% errors, 50% traces
- **Production**: 100% errors, 10% traces

### Integration in Services

Sentry is integrated via middleware in each service's main.go:

```go
import (
    "github.com/richxcame/ride-hailing/pkg/errors"
    "github.com/richxcame/ride-hailing/pkg/middleware"
)

// Initialize Sentry
sentryConfig := errors.DefaultSentryConfig()
sentryConfig.ServerName = serviceName
sentryConfig.Release = version
if err := errors.InitSentry(sentryConfig); err != nil {
    logger.Warn("Failed to initialize Sentry", zap.Error(err))
} else {
    defer errors.Flush(2 * time.Second)
    logger.Info("Sentry initialized successfully")
}

// Add middleware
router := gin.New()
router.Use(middleware.RecoveryWithSentry())  // First - catches panics
router.Use(middleware.SentryMiddleware())    // Early - sets up context
// ... other middleware ...
router.Use(middleware.ErrorHandler())        // Last - captures errors
```

### Usage in Application Code

#### Capturing Errors

```go
import "github.com/richxcame/ride-hailing/pkg/errors"

// Simple error capture
if err != nil {
    errors.CaptureError(err)
    return err
}

// Error with context
if err != nil {
    errors.CaptureErrorWithContext(ctx, err, map[string]interface{}{
        "user_id": userID,
        "operation": "payment_processing",
        "amount": amount,
    })
    return err
}
```

#### Capturing Messages

```go
import "github.com/getsentry/sentry-go"

// Info message
errors.CaptureMessage("Payment processed successfully", sentry.LevelInfo)

// Warning
errors.CaptureMessage("Cache miss rate high", sentry.LevelWarning)

// Error
errors.CaptureMessage("External API degraded", sentry.LevelError)
```

#### Adding Breadcrumbs

Breadcrumbs provide context leading up to an error:

```go
// HTTP request breadcrumb
errors.AddBreadcrumbForRequest("POST", "/api/v1/payments", 200, 150*time.Millisecond)

// Custom breadcrumb
errors.AddBreadcrumb(&sentry.Breadcrumb{
    Category: "payment",
    Message:  "Stripe API called",
    Level:    sentry.LevelInfo,
    Data: map[string]interface{}{
        "amount": 99.99,
        "method": "stripe",
    },
})
```

#### Setting User Context

```go
errors.SetUser(
    userID,
    "user@example.com",
    "johndoe",
    "192.168.1.1",
)
```

#### Setting Tags

Tags enable filtering and grouping in Sentry:

```go
errors.SetTag("payment_method", "stripe")
errors.SetTag("feature_flag", "new_checkout")
errors.SetTag("environment", "production")
```

### Error Filtering

Not all errors should be reported to Sentry. The integration automatically filters:

#### Errors NOT Reported
- Validation failures (400 Bad Request)
- Authentication errors (401 Unauthorized)
- Authorization errors (403 Forbidden)
- Not found errors (404 Not Found)
- Conflict errors (409 Conflict)
- Other 4xx client errors (except 429)

#### Errors Reported
- Server errors (5xx status codes)
- Panics and crashes
- Rate limiting errors (429)
- Database connection errors
- External service failures
- Unexpected application errors

### What Gets Captured

When an error occurs, Sentry captures:

1. **Error Details**
   - Exception type and message
   - Full stack trace
   - Error severity level

2. **Request Context**
   - HTTP method, URL, headers
   - Query parameters
   - Request body (sanitized)
   - Client IP address
   - User agent

3. **User Information**
   - User ID, email, username
   - IP address
   - User role (as tag)

4. **Trace Context**
   - Correlation ID (X-Request-ID)
   - OpenTelemetry trace ID
   - OpenTelemetry span ID

5. **Application Context**
   - Service name
   - Release version
   - Environment (dev/staging/prod)
   - Server name

6. **Breadcrumbs**
   - Sequence of events leading to error
   - HTTP requests
   - Database queries
   - User actions
   - Custom events

### Accessing Sentry

1. **Sentry Dashboard**: https://sentry.io
2. **Navigate to Issues** to see all captured errors
3. **Click an issue** to view:
   - Error details and stack trace
   - Request context and headers
   - User information
   - Breadcrumb trail
   - Similar issues
   - Release information

### Best Practices

#### 1. Use Appropriate Log Levels

```go
// Info - informational (not typically errors)
errors.CaptureMessage("User logged in", sentry.LevelInfo)

// Warning - concerning but not critical
errors.CaptureMessage("Cache miss rate high", sentry.LevelWarning)

// Error - errors needing attention
errors.CaptureError(err)

// Fatal - critical errors
errors.CaptureMessage("Database unreachable", sentry.LevelFatal)
```

#### 2. Add Context to Errors

Always provide context:

```go
errors.CaptureErrorWithContext(ctx, err, map[string]interface{}{
    "operation": "process_payment",
    "user_id": userID,
    "amount": amount,
    "payment_method": "stripe",
})
```

#### 3. Use Consistent Tags

```go
errors.SetTag("service", serviceName)
errors.SetTag("payment_provider", "stripe")
errors.SetTag("feature", "new_checkout")
```

#### 4. Don't Report Business Errors

```go
if err != nil {
    // Check if error should be reported
    if errors.ShouldReportError(err, statusCode) {
        errors.CaptureError(err)
    }
    // Handle error normally
}
```

#### 5. Add Breadcrumbs Throughout Flow

```go
// At operation start
errors.AddBreadcrumb(&sentry.Breadcrumb{
    Category: "payment",
    Message: "Starting payment processing",
})

// During operation
errors.AddBreadcrumb(&sentry.Breadcrumb{
    Category: "payment",
    Message: "Calling Stripe API",
})

// On completion
errors.AddBreadcrumb(&sentry.Breadcrumb{
    Category: "payment",
    Message: "Payment completed",
})
```

### Troubleshooting Sentry

#### Errors Not Appearing

1. Check `SENTRY_DSN` is set correctly
2. Verify `SENTRY_SAMPLE_RATE` is not 0
3. Enable debug mode: `SENTRY_DEBUG=true`
4. Check network connectivity to sentry.io
5. Verify error is not being filtered

#### Too Many Errors

1. Lower `SENTRY_SAMPLE_RATE` (e.g., 0.5 for 50%)
2. Add more business error patterns to filter
3. Configure rate limits in Sentry dashboard
4. Review `BeforeSend` hook in `pkg/errors/sentry.go`

#### Missing Context

1. Ensure middleware order is correct
2. Call `SetUser()` after authentication
3. Add breadcrumbs throughout request flow
4. Use tags for important metadata

### Related Documentation

- [Sentry Integration Guide](./SENTRY_INTEGRATION_GUIDE.md) - Complete integration guide
- [Sentry Official Docs](https://docs.sentry.io/platforms/go/) - Sentry Go SDK
- [Error Handling Best Practices](./ERROR_HANDLING.md) - Error handling patterns

---

## Accessing Observability Tools

### Grafana
- **URL**: http://localhost:3000
- **Credentials**: admin / admin
- **Features**:
  - Explore traces (Tempo data source)
  - View metrics dashboards (Prometheus data source)
  - Correlate logs with traces

### Prometheus
- **URL**: http://localhost:9090
- **Features**:
  - Query metrics with PromQL
  - View targets and service discovery
  - Alert rules (configured separately)

### OpenTelemetry Collector
- **Health Check**: http://localhost:13133
- **Metrics**: http://localhost:8888/metrics
- **zPages**: http://localhost:55679/debug/tracez

### Tempo
- **HTTP API**: http://localhost:3200
- **Query Endpoint**: http://localhost:3200/api/search

---

## Adding Custom Instrumentation

### Step 1: Import Tracing Package

```go
import (
    "github.com/richxcame/ride-hailing/pkg/tracing"
    "go.opentelemetry.io/otel/attribute"
)
```

### Step 2: Add Spans to Functions

```go
func (s *Service) ProcessPayment(ctx context.Context, paymentID uuid.UUID, amount float64) error {
    // Start a span
    ctx, span := tracing.StartSpan(ctx, "payments-service", "ProcessPayment")
    defer span.End()

    // Add attributes
    tracing.AddSpanAttributes(ctx,
        tracing.PaymentIDKey.String(paymentID.String()),
        tracing.FareAmountKey.Float64(amount),
    )

    // Business logic here...
    err := s.chargeCard(ctx, amount)
    if err != nil {
        // Record error
        tracing.RecordError(ctx, err)
        return err
    }

    // Add event
    tracing.AddSpanEvent(ctx, "payment_successful",
        attribute.String("payment_id", paymentID.String()),
        attribute.Float64("amount", amount),
    )

    return nil
}
```

### Step 3: Trace Database Operations

```go
func (r *Repository) GetRideByID(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
    query := "SELECT * FROM rides WHERE id = $1"

    return tracing.TraceDBQuery(ctx, "rides-service", "SELECT", query, func() error {
        return r.db.QueryRow(ctx, query, rideID).Scan(...)
    })
}
```

### Step 4: Trace External API Calls

```go
func (s *Service) CallExternalAPI(ctx context.Context) error {
    return tracing.TraceExternalAPI(ctx, "rides-service", "stripe", "createCharge", func(ctx context.Context) error {
        // Make API call
        return s.stripeClient.CreateCharge(ctx, ...)
    })
}
```

### Step 5: Initialize Tracing in main.go

```go
func main() {
    // ... logger init ...

    // Initialize tracing
    tracerEnabled := os.Getenv("OTEL_ENABLED") == "true"
    if tracerEnabled {
        tracerCfg := tracing.Config{
            ServiceName:    os.Getenv("OTEL_SERVICE_NAME"),
            ServiceVersion: os.Getenv("OTEL_SERVICE_VERSION"),
            Environment:    cfg.Server.Environment,
            OTLPEndpoint:   os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
            Enabled:        true,
        }

        tp, err := tracing.InitTracer(tracerCfg, logger.Logger())
        if err != nil {
            logger.Warn("Failed to initialize tracer", zap.Error(err))
        } else {
            defer tp.Shutdown(context.Background())
        }
    }

    // ... rest of setup ...

    // Add tracing middleware
    if tracerEnabled {
        router.Use(middleware.TracingMiddleware(serviceName))
    }
}
```

---

## Troubleshooting

### No Traces Appearing in Grafana

1. **Check service configuration**:
   ```bash
   docker logs ridehailing-rides
   # Look for: "OpenTelemetry tracing initialized successfully"
   ```

2. **Verify OTel Collector is running**:
   ```bash
   curl http://localhost:13133
   # Should return: {"status":"Server available"}
   ```

3. **Check Tempo is receiving traces**:
   ```bash
   curl http://localhost:3200/api/search
   ```

4. **Verify environment variables**:
   ```bash
   docker exec ridehailing-rides env | grep OTEL
   ```

### Trace Context Not Propagating

1. **Check W3C headers in requests**:
   - Ensure `traceparent` header is present
   - Format: `00-{trace-id}-{parent-id}-{trace-flags}`

2. **Verify middleware order**:
   ```go
   // TracingMiddleware should be after CorrelationID
   router.Use(middleware.CorrelationID())
   router.Use(middleware.TracingMiddleware(serviceName))
   ```

### High Memory Usage from Tracing

1. **Reduce sample rate** in production:
   ```yaml
   environment:
     OTEL_TRACE_SAMPLE_RATE: "0.01"  # 1% sampling
   ```

2. **Enable tail sampling** in `deploy/otel-collector.yml`

### Missing Span Attributes

1. **Ensure context is passed through**:
   ```go
   // Correct
   result, err := s.someFunction(ctx, ...)

   // Wrong - creates new context
   result, err := s.someFunction(context.Background(), ...)
   ```

---

## Best Practices

### DO:
- Use consistent span naming conventions
- Add relevant business attributes to spans
- Record errors with `tracing.RecordError()`
- Pass context through all function calls
- Add meaningful span events for important state changes

### DON'T:
- Add PII (personally identifiable information) to span attributes
- Create too many spans (adds overhead)
- Block on span operations
- Log sensitive data in traces

---

## Environment Variables

### Per-Service Configuration

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `OTEL_ENABLED` | Enable/disable tracing | `false` | `true` |
| `OTEL_SERVICE_NAME` | Service name | - | `rides-service` |
| `OTEL_SERVICE_VERSION` | Service version | - | `1.0.0` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Collector endpoint | `localhost:4317` | `otel-collector:4317` |
| `OTEL_TRACE_SAMPLE_RATE` | Override sample rate | env-based | `0.1` |

### Global Configuration

Set in `deploy/otel-collector.yml` and `deploy/tempo.yml`

---

## Alerting

### Alert Rules

Prometheus is configured with comprehensive alerting rules organized into six categories:

#### System Alerts
Monitor system health and performance:
- **HighErrorRate**: 5xx error rate > 5% for 5 minutes
- **HighLatency**: P99 latency > 1s for 5 minutes
- **ServiceDown**: Service unavailable for > 1 minute
- **HighCPUUsage**: CPU usage > 85% for 10 minutes
- **HighMemoryUsage**: Memory usage > 85% for 10 minutes
- **TooManyGoroutines**: > 5000 goroutines for 5 minutes

#### Business Alerts
Monitor business metrics:
- **LowDriverAvailability**: < 10 drivers online for 5 minutes
- **HighRideCancellationRate**: > 30% cancellation rate for 10 minutes
- **HighPaymentFailureRate**: > 10% payment failure rate for 5 minutes
- **NoRidesCreated**: No rides created for 10 minutes
- **RevenueDrop**: Revenue dropped > 50% compared to same time yesterday

#### Database Alerts
Monitor database health:
- **DatabaseConnectionPoolExhaustion**: > 90% pool usage for 5 minutes
- **SlowDatabaseQueries**: P95 query time > 1s for 5 minutes

#### Redis Alerts
Monitor cache performance:
- **LowRedisHitRate**: < 70% cache hit rate for 10 minutes
- **HighRedisMemoryUsage**: > 90% memory usage for 5 minutes

#### Circuit Breaker Alerts
Monitor resilience patterns:
- **CircuitBreakerOpen**: Circuit breaker open for > 2 minutes
- **HighCircuitBreakerFailureRate**: > 10 failures/second for 5 minutes

#### Rate Limit Alerts
Monitor traffic control:
- **HighRateLimitRejectionRate**: > 10% rejection rate for 5 minutes

#### Fraud Alerts
Monitor security:
- **HighFraudDetectionRate**: > 5 fraud alerts/second for 5 minutes
- **BlockedUserSpike**: 3x increase in blocked users compared to 1 hour ago

### Alert Configuration

Alert rules are defined in [monitoring/prometheus/alerts.yml](../monitoring/prometheus/alerts.yml)

### Viewing Alerts in Prometheus

1. **Open Prometheus**: http://localhost:9090
2. **Navigate**: Alerts tab
3. **View**: See all configured alerts and their current state
   - **Green (Inactive)**: Alert condition not met
   - **Yellow (Pending)**: Alert condition met, waiting for duration
   - **Red (Firing)**: Alert actively firing

### Alert Severity Levels

- **Critical**: Immediate action required (service down, high error rate, payment failures)
- **Warning**: Investigation needed (high resource usage, low driver availability)

### Future: AlertManager Integration

To receive alert notifications (email, Slack, PagerDuty):

1. Deploy Alertmanager:
   ```yaml
   alertmanager:
     image: prom/alertmanager:latest
     ports:
       - "9093:9093"
     volumes:
       - ./monitoring/alertmanager.yml:/etc/alertmanager/alertmanager.yml
   ```

2. Update Prometheus config:
   ```yaml
   alerting:
     alertmanagers:
       - static_configs:
           - targets: ['alertmanager:9093']
   ```

3. Configure notification receivers in `alertmanager.yml`

---

## Further Reading

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [Grafana Tempo Documentation](https://grafana.com/docs/tempo/latest/)
- [W3C Trace Context Specification](https://www.w3.org/TR/trace-context/)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/)
- [Prometheus Alerting Rules](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/)
- [Grafana Dashboard Best Practices](https://grafana.com/docs/grafana/latest/dashboards/build-dashboards/best-practices/)
