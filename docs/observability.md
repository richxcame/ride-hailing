# Observability Guide

This guide explains the observability stack for the ride-hailing platform, including distributed tracing, metrics, and logging.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Distributed Tracing](#distributed-tracing)
3. [Metrics](#metrics)
4. [Logging](#logging)
5. [Accessing Observability Tools](#accessing-observability-tools)
6. [Adding Custom Instrumentation](#adding-custom-instrumentation)
7. [Troubleshooting](#troubleshooting)

---

## Architecture Overview

The observability stack consists of:

- **OpenTelemetry Collector**: Receives traces from all services and exports to Tempo
- **Grafana Tempo**: Stores and queries distributed traces
- **Prometheus**: Collects and stores metrics from all services
- **Grafana**: Unified visualization dashboard for traces, metrics, and logs
- **Zap Logger**: Structured JSON logging with correlation ID support

### Data Flow

```
Service → OpenTelemetry SDK → OTLP → OTel Collector → Tempo
                                                     → Prometheus

                                    ↓
                                 Grafana (Visualization)
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

## Further Reading

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [Grafana Tempo Documentation](https://grafana.com/docs/tempo/latest/)
- [W3C Trace Context Specification](https://www.w3.org/TR/trace-context/)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/)
