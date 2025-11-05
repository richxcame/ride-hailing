# Performance Optimization Guide

This document outlines the performance optimizations implemented in the Ride Hailing platform and provides guidance for maintaining optimal performance.

## Table of Contents

1. [Database Optimizations](#database-optimizations)
2. [Caching Strategy](#caching-strategy)
3. [Connection Pooling](#connection-pooling)
4. [Query Optimization](#query-optimization)
5. [Monitoring & Metrics](#monitoring--metrics)
6. [Scaling Strategies](#scaling-strategies)

## Database Optimizations

### Indexes

We've implemented strategic indexes to optimize common query patterns:

#### Users Table
- `idx_users_email_lower`: Case-insensitive email lookups
- `idx_users_role_active`: Filter active users by role
- `idx_users_created_at`: Sort/filter by registration date

#### Rides Table
- `idx_rides_status_requested_at`: Most common query pattern
- `idx_rides_pending`: Partial index for pending rides only
- `idx_rides_driver_status`: Driver's ride history
- `idx_rides_rider_requested_at`: Rider's ride history
- `idx_rides_completed_at`: Analytics on completed rides

#### Payments Table
- `idx_payments_user_created`: User payment history
- `idx_payments_status`: Filter by payment status
- `idx_payments_ride`: Lookup payments by ride

### Materialized Views

Pre-calculated aggregations for faster analytics:

#### mv_demand_zones
```sql
-- Refresh every 15 minutes via scheduler
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_demand_zones;
```

- Aggregates ride demand by geographic zones
- Used for heat maps and demand visualization
- Includes ride counts, average surge, and pricing data

#### mv_driver_performance
```sql
-- Refresh hourly
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_driver_performance;
```

- Driver statistics and ratings
- Completion rates and earnings
- Used for driver leaderboards and performance reports

#### mv_revenue_metrics
```sql
-- Refresh daily at midnight
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_revenue_metrics;
```

- Daily revenue aggregations
- Platform commission calculations
- Financial reporting data

### Refresh Schedule

Add to scheduler service (cron-like):
```go
// Refresh every 15 minutes
schedule.Every(15).Minutes().Do(func() {
    db.Exec(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY mv_demand_zones")
})

// Refresh hourly
schedule.Every(1).Hour().Do(func() {
    db.Exec(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY mv_driver_performance")
})

// Refresh daily
schedule.Every(1).Day().At("00:00").Do(func() {
    db.Exec(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY mv_revenue_metrics")
})
```

### Autovacuum Tuning

Optimized for high-traffic tables:

```sql
-- Rides table (high write volume)
ALTER TABLE rides SET (
    autovacuum_vacuum_scale_factor = 0.05,  -- Vacuum at 5% dead tuples
    autovacuum_analyze_scale_factor = 0.02   -- Analyze at 2% changes
);
```

## Caching Strategy

### Cache Layer

Use the provided `pkg/cache` package for consistent caching:

```go
import "github.com/richxcame/ride-hailing/pkg/cache"

// Get or set pattern
var user models.User
err := cache.GetOrSet(ctx, cache.UserKey(userID), 1*time.Hour, func() (interface{}, error) {
    return repo.GetUser(ctx, userID)
}, &user)
```

### Cache Keys

Pre-defined key generators:
- `cache.UserKey(userID)` - User profiles
- `cache.RideKey(rideID)` - Ride details
- `cache.DriverLocationKey(driverID)` - Real-time locations
- `cache.PromoCodeKey(code)` - Promo code validation
- `cache.SurgePricingKey(lat, lon)` - Surge calculations

### Cache TTLs

Recommended expiration times:

| Data Type | TTL | Reason |
|-----------|-----|--------|
| User Profile | 1 hour | Infrequent changes |
| Driver Location | 30 seconds | Real-time tracking |
| Promo Code | 5 minutes | Moderate changes |
| Surge Pricing | 2 minutes | Dynamic pricing |
| Analytics Dashboard | 5 minutes | Expensive queries |
| Ride Details | 10 minutes | Moderate updates |

### Cache Invalidation

Invalidate cache on updates:

```go
// After updating user
cache.Delete(ctx, cache.UserKey(userID))

// Pattern deletion
cache.DeletePattern(ctx, "user:*")
```

## Connection Pooling

### Optimized Pool Settings

```go
// Connection limits
MaxConns: 50              // Max concurrent connections
MinConns: 10              // Always keep 10 warm connections

// Connection lifecycle
MaxConnLifetime: 1 hour   // Recycle connections
MaxConnIdleTime: 30 mins  // Close idle connections
HealthCheckPeriod: 1 min  // Check health periodically
```

### Per-Environment Recommendations

**Development:**
```yaml
max_conns: 10
min_conns: 2
```

**Staging:**
```yaml
max_conns: 25
min_conns: 5
```

**Production:**
```yaml
max_conns: 50-100  # Based on load
min_conns: 10
```

### Formula for Max Connections

```
max_conns = (available_database_connections - buffer) / number_of_services
```

Example:
- Postgres max_connections: 200
- Services: 12
- Buffer: 20 (for admin/maintenance)
- Per service: (200 - 20) / 12 ≈ 15 connections

## Query Optimization

### Best Practices

1. **Use Prepared Statements**
```go
// Automatically cached by pgx
rows, err := db.Query(ctx, "SELECT * FROM users WHERE id = $1", userID)
```

2. **Avoid SELECT ***
```go
// Bad
SELECT * FROM rides

// Good
SELECT id, status, rider_id, estimated_fare FROM rides
```

3. **Use LIMIT for Large Datasets**
```go
SELECT * FROM rides
WHERE status = 'requested'
ORDER BY requested_at DESC
LIMIT 100
```

4. **Leverage Partial Indexes**
```sql
-- Index only active records
CREATE INDEX idx_active_drivers ON drivers(id)
WHERE is_active = true;
```

### Query Performance Monitoring

Use `EXPLAIN ANALYZE`:

```sql
EXPLAIN ANALYZE
SELECT * FROM rides
WHERE status = 'requested'
AND requested_at > NOW() - INTERVAL '1 hour';
```

Look for:
- Sequential Scans (bad) vs Index Scans (good)
- High execution time
- Large row counts

## Monitoring & Metrics

### Prometheus Metrics

All services expose `/metrics` endpoint:

**Database Metrics:**
- `db_connections_active`
- `db_connections_idle`
- `db_query_duration_seconds`

**HTTP Metrics:**
- `http_requests_total`
- `http_request_duration_seconds`
- `http_requests_in_progress`

**Business Metrics:**
- `rides_total{status="completed"}`
- `payments_total{status="succeeded"}`
- `fraud_alerts_total{level="critical"}`

### Grafana Dashboards

Access Grafana at `http://localhost:3000`

Pre-configured dashboards:
1. **Service Health**: Request rates, error rates, latency
2. **Database Performance**: Connection pools, query times
3. **Business Metrics**: Rides, revenue, user activity

### Alerts

Set up alerts for:
- High error rates (>1%)
- Slow response times (p95 >1s)
- Database connection pool exhaustion (>90%)
- High fraud alert rate

## Scaling Strategies

### Horizontal Scaling

All services are stateless and can be scaled horizontally:

```bash
docker-compose up --scale rides-service=3
```

### Database Read Replicas

For high read volume:

1. Set up PostgreSQL replication
2. Configure read-only connection pool
3. Route read queries to replicas

```go
// Read from replica
replicaPool := database.NewPostgresPool(replicaConfig)

// Writes go to primary
primaryPool := database.NewPostgresPool(primaryConfig)
```

### Redis Clustering

For high cache volume:

```yaml
redis:
  cluster:
    enabled: true
    nodes:
      - redis-1:6379
      - redis-2:6379
      - redis-3:6379
```

### CDN for Static Assets

Use CloudFront/CloudFlare for:
- API documentation
- Static resources
- Image uploads

### Database Sharding (Future)

For massive scale (>10M rides/month):

- Shard by geographic region
- Shard by user ID hash
- Use Vitess or Citus for transparent sharding

## Performance Targets

### Service Level Objectives (SLOs)

| Metric | Target | Acceptable |
|--------|--------|------------|
| API Response Time (p95) | <500ms | <1s |
| API Response Time (p99) | <1s | <2s |
| Availability | 99.9% | 99.5% |
| Database Query Time (p95) | <100ms | <500ms |
| Cache Hit Rate | >80% | >70% |

### Load Testing

Use k6 for load testing:

```javascript
import http from 'k6/http';

export let options = {
  stages: [
    { duration: '2m', target: 100 },  // Ramp up
    { duration: '5m', target: 100 },  // Stay at 100 users
    { duration: '2m', target: 0 },    // Ramp down
  ],
};

export default function() {
  http.get('http://localhost:8082/api/v1/rides/available');
}
```

## Troubleshooting

### Slow Queries

1. Check `pg_stat_statements`:
```sql
SELECT query, mean_exec_time, calls
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;
```

2. Enable slow query logging:
```sql
ALTER SYSTEM SET log_min_duration_statement = 1000; -- Log queries >1s
SELECT pg_reload_conf();
```

### Connection Pool Exhaustion

1. Check pool stats:
```sql
SELECT * FROM pg_stat_activity;
```

2. Identify long-running queries:
```sql
SELECT pid, now() - query_start as duration, query
FROM pg_stat_activity
WHERE state = 'active'
ORDER BY duration DESC;
```

3. Kill long-running query:
```sql
SELECT pg_terminate_backend(pid);
```

### Cache Issues

1. Check Redis memory:
```bash
redis-cli INFO memory
```

2. Monitor cache hit rate:
```bash
redis-cli INFO stats | grep hit
```

3. Flush cache if needed:
```bash
redis-cli FLUSHDB
```

## Best Practices Summary

1. ✅ Use indexes for common query patterns
2. ✅ Cache expensive queries (TTL based on update frequency)
3. ✅ Use connection pooling (never create new connections per request)
4. ✅ Monitor query performance (EXPLAIN ANALYZE)
5. ✅ Set up alerts for performance degradation
6. ✅ Refresh materialized views regularly
7. ✅ Use partial indexes for filtered queries
8. ✅ Avoid N+1 queries (use JOINs or batch loading)
9. ✅ Set statement timeouts to prevent runaway queries
10. ✅ Regularly vacuum and analyze tables

## Further Reading

- [PostgreSQL Performance Tuning](https://wiki.postgresql.org/wiki/Performance_Optimization)
- [Redis Best Practices](https://redis.io/docs/manual/patterns/)
- [Go Database Best Practices](https://github.com/golang/go/wiki/SQLInterface)
- [Prometheus Monitoring](https://prometheus.io/docs/practices/)
