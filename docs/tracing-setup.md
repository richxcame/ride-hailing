# Distributed Tracing Setup

## Enable Tracing

Set these environment variables per service (in `docker-compose.yml` or your shell):

```yaml
environment:
  OTEL_ENABLED: "true"
  OTEL_SERVICE_NAME: "rides-service"
  OTEL_SERVICE_VERSION: "1.0.0"
  OTEL_EXPORTER_OTLP_ENDPOINT: "otel-collector:4317"
```

## Verify It Works

```bash
# 1. Confirm the OTel Collector is healthy
curl http://localhost:13133

# 2. Send a test request
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

# 3. Copy the X-Trace-ID from the response headers
```

Then open Grafana at http://localhost:3000 (`admin` / `admin`), go to
**Explore** > **Tempo**, and search for that trace ID.

## Sampling Rates

Defaults are set in [pkg/tracing/tracer.go](../pkg/tracing/tracer.go:127):

| Environment              | Rate |
| ------------------------ | ---- |
| `development` / `local`  | 100% |
| `staging`                | 50%  |
| `production`             | 10%  |

Override per service with:

```bash
OTEL_TRACE_SAMPLE_RATE=0.05  # 5% sampling
```

## Config File Locations

| File | Purpose |
| ---- | ------- |
| `deploy/otel-collector.yml` | Collector receivers, processors (batch, memory limit, tail sampling), and exporters |
| `deploy/tempo.yml` | Tempo storage backend, retention, and metrics generator |
| `docker-compose.yml` | Per-service OTEL_* environment variables |
| `pkg/tracing/tracer.go` | Go tracer initialization and default sample rates |
