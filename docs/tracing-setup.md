# Distributed Tracing Setup Guide

This guide will help you get started with distributed tracing in the ride-hailing platform.

## Quick Start

### 1. Start the Stack

```bash
# Start all services including tracing infrastructure
docker-compose up -d

# Check that OTel Collector and Tempo are running
docker ps | grep -E "otel-collector|tempo"
```

### 2. Verify Tracing is Working

```bash
# Check OTel Collector health
curl http://localhost:13133

# Check Tempo is running
curl http://localhost:3200/ready
```

### 3. Access Grafana

1. Open http://localhost:3000
2. Login with `admin` / `admin`
3. Go to **Explore** → Select **Tempo** data source
4. Click **Search** to see available traces

### 4. Generate Test Traces

```bash
# Make a test request to the rides service
curl -X POST http://localhost:8082/api/rides \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "pickup_latitude": 40.7128,
    "pickup_longitude": -74.0060,
    "dropoff_latitude": 40.7580,
    "dropoff_longitude": -73.9855,
    "pickup_address": "New York, NY",
    "dropoff_address": "Times Square, NY"
  }'

# Copy the X-Trace-ID from the response headers
# Use it to search for the trace in Grafana
```

---

## Configuration

### Environment Variables (Per Service)

Each service is configured with the following environment variables in `docker-compose.yml`:

```yaml
environment:
  OTEL_ENABLED: "true"
  OTEL_SERVICE_NAME: "rides-service"
  OTEL_SERVICE_VERSION: "1.0.0"
  OTEL_EXPORTER_OTLP_ENDPOINT: "otel-collector:4317"
```

### Sampling Configuration

Edit sampling rates in your service's environment or in [pkg/tracing/tracer.go](../pkg/tracing/tracer.go:127):

```go
// Default sample rates by environment
switch cfg.Environment {
case "development", "dev", "local":
    sampleRate = 1.0 // 100% in development
case "staging", "stage":
    sampleRate = 0.5 // 50% in staging
case "production", "prod":
    sampleRate = 0.1 // 10% in production
}
```

Override with environment variable:
```bash
OTEL_TRACE_SAMPLE_RATE=0.05  # 5% sampling
```

### OpenTelemetry Collector Configuration

Edit [deploy/otel-collector.yml](../deploy/otel-collector.yml) to customize:

- **Receivers**: OTLP endpoints
- **Processors**: Batch size, memory limits, tail sampling
- **Exporters**: Tempo, logging, Prometheus

### Tempo Configuration

Edit [deploy/tempo.yml](../deploy/tempo.yml) to customize:

- **Storage**: Local filesystem, S3, or MinIO
- **Retention**: Trace retention period (default: 48h)
- **Metrics Generator**: Service graphs and span metrics

---

## Instrumenting Additional Services

### Step 1: Update main.go

Add tracing initialization in your service's `main.go`:

```go
import (
    "github.com/richxcame/ride-hailing/pkg/tracing"
    "github.com/richxcame/ride-hailing/pkg/middleware"
)

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

        tp, err := tracing.InitTracer(tracerCfg, logger.Get())
        if err != nil {
            logger.Warn("Failed to initialize tracer", zap.Error(err))
        } else {
            defer func() {
                ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
                defer cancel()
                if err := tp.Shutdown(ctx); err != nil {
                    logger.Warn("Failed to shutdown tracer", zap.Error(err))
                }
            }()
        }
    }

    // ... setup router ...

    // Add tracing middleware
    if tracerEnabled {
        router.Use(middleware.TracingMiddleware(serviceName))
    }
}
```

### Step 2: Add Business Logic Spans

Import the tracing package in your service files:

```go
import (
    "github.com/richxcame/ride-hailing/pkg/tracing"
    "go.opentelemetry.io/otel/attribute"
)
```

Wrap important functions with spans:

```go
func (s *Service) ProcessPayment(ctx context.Context, paymentID uuid.UUID, amount float64) error {
    // Start span
    ctx, span := tracing.StartSpan(ctx, "payments-service", "ProcessPayment")
    defer span.End()

    // Add attributes
    tracing.AddSpanAttributes(ctx,
        tracing.PaymentIDKey.String(paymentID.String()),
        tracing.FareAmountKey.Float64(amount),
    )

    // Your business logic...
    err := s.chargeCard(ctx, amount)
    if err != nil {
        tracing.RecordError(ctx, err)
        return err
    }

    // Add success event
    tracing.AddSpanEvent(ctx, "payment_successful",
        attribute.String("payment_id", paymentID.String()),
    )

    return nil
}
```

### Step 3: Update docker-compose.yml

Ensure your service has the OTEL environment variables:

```yaml
your-service:
  environment:
    # ... existing vars ...
    OTEL_ENABLED: "true"
    OTEL_SERVICE_NAME: "your-service"
    OTEL_SERVICE_VERSION: "1.0.0"
    OTEL_EXPORTER_OTLP_ENDPOINT: "otel-collector:4317"
  depends_on:
    # ... existing deps ...
    otel-collector:
      condition: service_started
```

---

## Common Patterns

### Database Queries

```go
func (r *Repository) GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error) {
    query := "SELECT * FROM users WHERE id = $1"

    var user User
    err := tracing.TraceDBQuery(ctx, "auth-service", "SELECT", query, func() error {
        return r.db.QueryRow(ctx, query, userID).Scan(&user)
    })

    return &user, err
}
```

### Redis Operations

```go
func (s *Service) GetCachedData(ctx context.Context, key string) (string, error) {
    var result string

    err := tracing.TraceRedisCommand(ctx, "geo-service", "GET", key, func() error {
        var err error
        result, err = s.redis.Get(ctx, key).Result()
        return err
    })

    return result, err
}
```

### HTTP Client Calls

```go
func (s *Service) CallExternalService(ctx context.Context, url string) error {
    statusCode, err := tracing.TraceHTTPClient(ctx, "rides-service", "POST", url, func() (int, error) {
        resp, err := s.httpClient.Post(ctx, url, body)
        if err != nil {
            return 0, err
        }
        return resp.StatusCode, nil
    })

    if statusCode >= 400 {
        return fmt.Errorf("external service returned %d", statusCode)
    }

    return err
}
```

---

## Troubleshooting

### No traces appearing

1. Check service logs for tracing initialization:
```bash
docker logs ridehailing-rides 2>&1 | grep -i "tracing\|otel"
```

2. Verify OTel Collector is receiving traces:
```bash
docker logs ridehailing-otel-collector | tail -20
```

3. Check Tempo logs:
```bash
docker logs ridehailing-tempo | tail -20
```

### Traces missing attributes

Ensure you're passing `ctx` through all function calls:
```go
// ✅ Correct
result := s.someFunction(ctx, args)

// ❌ Wrong - creates new context without trace
result := s.someFunction(context.Background(), args)
```

### High memory usage

Reduce sampling rate:
```yaml
environment:
  OTEL_TRACE_SAMPLE_RATE: "0.01"  # 1% sampling
```

---

## Production Considerations

### 1. Sampling

Use tail sampling in production to keep all error traces:

Edit `deploy/otel-collector.yml` and uncomment the `tail_sampling` processor:

```yaml
processors:
  tail_sampling:
    decision_wait: 10s
    policies:
      - name: error-traces
        type: status_code
        status_code:
          status_codes: [ERROR]
      - name: probabilistic-policy
        type: probabilistic
        probabilistic:
          sampling_percentage: 10
```

### 2. Storage

For production, use S3 or MinIO for Tempo storage:

Edit `deploy/tempo.yml`:

```yaml
storage:
  trace:
    backend: s3
    s3:
      bucket: tempo-traces
      endpoint: s3.amazonaws.com
      access_key: YOUR_ACCESS_KEY
      secret_key: YOUR_SECRET_KEY
```

### 3. Retention

Configure trace retention based on your needs:

```yaml
compactor:
  compaction:
    block_retention: 168h  # 7 days
```

### 4. Resource Limits

Set resource limits for OTel Collector:

```yaml
processors:
  memory_limiter:
    limit_mib: 1024
    spike_limit_mib: 256
```

---

## Next Steps

1. ✅ Review [docs/observability.md](./observability.md) for comprehensive documentation
2. Add custom instrumentation to your services
3. Create Grafana dashboards for key traces
4. Set up alerting based on trace data
5. Configure production sampling strategy

For questions or issues, refer to:
- [OpenTelemetry Go Documentation](https://opentelemetry.io/docs/instrumentation/go/)
- [Grafana Tempo Documentation](https://grafana.com/docs/tempo/latest/)
