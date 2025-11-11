# Monitoring & Observability

This directory contains the monitoring and observability configuration for the ride-hailing platform.

## Overview

The monitoring stack includes:
- **Prometheus**: Metrics collection and alerting
- **Grafana**: Visualization dashboards
- **Tempo**: Distributed tracing backend
- **OpenTelemetry Collector**: Trace collection and processing

## Directory Structure

```
monitoring/
├── README.md                           # This file
├── prometheus.yml                      # Prometheus scrape configuration
├── prometheus/
│   └── alerts.yml                      # Prometheus alert rules (30+ rules)
└── grafana/
    ├── dashboards/
    │   ├── overview.json               # System overview dashboard
    │   ├── rides.json                  # Rides service dashboard
    │   └── payments.json               # Payments service dashboard
    └── provisioning/
        ├── datasources/
        │   └── datasources.yml         # Auto-configured data sources
        └── dashboards/
            └── dashboards.yml          # Dashboard provisioning config
```

## Quick Start

### 1. Start the Monitoring Stack

```bash
# Start all services including monitoring
docker-compose up -d

# Or start only monitoring services
docker-compose up -d prometheus grafana tempo otel-collector
```

### 2. Access Dashboards

- **Grafana**: http://localhost:3000
  - Username: `admin`
  - Password: `admin`
  - Dashboards are auto-loaded in the "RideHailing" folder

- **Prometheus**: http://localhost:9090
  - View metrics and alerts
  - Execute PromQL queries

- **Tempo**: http://localhost:3200
  - Access via Grafana (Explore → Tempo)

## Dashboards

### 1. System Overview (`ridehailing-overview`)

**Purpose**: High-level system health and performance monitoring

**Panels**:
- Request rate by service
- Request latency (P95, P99)
- Error rates (5xx)
- Traffic distribution
- Status code distribution
- CPU, memory, goroutines
- Global metrics (total RPS, P99 latency, error rate)

**Best for**: Operations team, incident response, health checks

### 2. Rides Service (`ridehailing-rides`)

**Purpose**: Business metrics for ride operations

**Panels**:
- Rides created/completed/cancelled (hourly)
- Cancellation rate
- Drivers online / active rides
- Ride duration percentiles
- Ride distance percentiles
- Drivers by region
- Rides by type
- Cancellation reasons
- Driver matching time

**Best for**: Product managers, business analysts

### 3. Payments Service (`ridehailing-payments`)

**Purpose**: Financial metrics and payment health

**Panels**:
- Revenue (hourly, trends)
- Payment success/failure rates
- Payment processing duration
- Payments by method
- Payment failure reasons
- Refunds (count and amount)
- Transaction amount percentiles

**Best for**: Finance team, operations

## Alerts

### Alert Categories

1. **System Alerts** (6 rules)
   - High error rate (>5%)
   - High latency (P99 >1s)
   - Service down
   - High CPU/memory usage
   - Too many goroutines

2. **Business Alerts** (5 rules)
   - Low driver availability (<10)
   - High cancellation rate (>30%)
   - High payment failure rate (>10%)
   - No rides created
   - Revenue drop (>50%)

3. **Database Alerts** (2 rules)
   - Connection pool exhaustion (>90%)
   - Slow queries (P95 >1s)

4. **Redis Alerts** (2 rules)
   - Low cache hit rate (<70%)
   - High memory usage (>90%)

5. **Circuit Breaker Alerts** (2 rules)
   - Circuit breaker open
   - High failure rate

6. **Rate Limit Alerts** (1 rule)
   - High rejection rate (>10%)

7. **Fraud Alerts** (2 rules)
   - High fraud detection rate
   - Blocked user spike

### Viewing Alerts

1. Open Prometheus: http://localhost:9090
2. Click "Alerts" tab
3. View alert states:
   - **Green**: Inactive (normal)
   - **Yellow**: Pending (condition met, waiting)
   - **Red**: Firing (alert active)

## Metrics Reference

### HTTP Metrics (All Services)

```promql
# Request rate per service
rate(http_requests_total[5m])

# P99 latency by service
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))

# Error rate by service
rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])
```

### Business Metrics

```promql
# Rides per hour
increase(rides_created_total[1h])

# Revenue per minute
sum(rate(payment_amount_total[5m])) * 60

# Payment failure rate
rate(payments_failed_total[5m]) / (rate(payments_successful_total[5m]) + rate(payments_failed_total[5m]))

# Cancellation rate
rate(rides_cancelled_total[1h]) / (rate(rides_created_total[1h]) + rate(rides_cancelled_total[1h]))
```

### System Metrics

```promql
# CPU usage
100 - (avg by (job) (rate(process_cpu_seconds_total[5m])) * 100)

# Memory usage
(process_resident_memory_bytes / process_virtual_memory_max_bytes) * 100

# Goroutines
go_goroutines
```

## Configuration

### Adding New Metrics

1. Instrument your service with Prometheus client
2. Add scrape target to `prometheus.yml`:
   ```yaml
   - job_name: 'your-service'
     static_configs:
       - targets: ['your-service:8080']
   ```
3. Restart Prometheus

### Creating Custom Dashboards

**Option 1: Manual Creation**
1. Open Grafana → Create → Dashboard
2. Add panels with PromQL queries
3. Export JSON and save to `grafana/dashboards/`
4. Update `grafana/provisioning/dashboards/dashboards.yml`

**Option 2: Import**
1. Copy dashboard JSON to `grafana/dashboards/`
2. Restart Grafana (dashboards auto-load)

### Adding New Alert Rules

1. Edit `prometheus/alerts.yml`
2. Add new rule:
   ```yaml
   - alert: YourAlertName
     expr: your_metric > threshold
     for: 5m
     labels:
       severity: warning
     annotations:
       summary: "Alert summary"
       description: "Alert description"
   ```
3. Reload Prometheus:
   ```bash
   curl -X POST http://localhost:9090/-/reload
   ```

## Troubleshooting

### Dashboards Not Loading

```bash
# Check Grafana logs
docker logs ridehailing-grafana

# Verify provisioning directory
docker exec ridehailing-grafana ls -la /etc/grafana/provisioning/dashboards/

# Restart Grafana
docker-compose restart grafana
```

### Alerts Not Firing

```bash
# Check alert rules in Prometheus
curl http://localhost:9090/api/v1/rules

# Verify Prometheus can scrape targets
# Open http://localhost:9090/targets

# Check Prometheus logs
docker logs ridehailing-prometheus
```

### Missing Metrics

```bash
# Check if service is exposing metrics
curl http://localhost:8082/metrics

# Verify Prometheus scrape config
docker exec ridehailing-prometheus cat /etc/prometheus/prometheus.yml

# Check Prometheus targets
# Open http://localhost:9090/targets
```

## Best Practices

### Dashboard Design
- Keep dashboards focused (one purpose per dashboard)
- Use consistent time ranges
- Add descriptions to panels
- Use appropriate visualization types
- Set meaningful thresholds

### Alert Configuration
- Set appropriate thresholds based on SLOs
- Use "for" duration to avoid alert flapping
- Group related alerts
- Include actionable information in annotations
- Test alerts with synthetic data

### Metric Naming
- Use consistent naming conventions
- Add appropriate labels
- Document custom metrics
- Use appropriate metric types (counter, gauge, histogram)

## Production Considerations

### AlertManager Setup

For production, configure AlertManager for notifications:

```yaml
# docker-compose.yml
alertmanager:
  image: prom/alertmanager:latest
  ports:
    - "9093:9093"
  volumes:
    - ./monitoring/alertmanager.yml:/etc/alertmanager/alertmanager.yml
```

### Retention and Storage

Configure Prometheus retention:
```yaml
command:
  - "--storage.tsdb.retention.time=30d"
  - "--storage.tsdb.retention.size=50GB"
```

### High Availability

For HA setup:
- Run multiple Prometheus instances
- Use Thanos for long-term storage
- Configure Grafana with multiple data sources
- Use external alertmanager cluster

## Resources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [PromQL Cheat Sheet](https://promlabs.com/promql-cheat-sheet/)
- [Platform Observability Guide](../docs/observability.md)
