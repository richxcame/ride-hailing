package database

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/resilience"
)

// DBPool holds both primary and replica connection pools
type DBPool struct {
	Primary  *pgxpool.Pool
	Replicas []*pgxpool.Pool
	mu       sync.RWMutex
	rrIndex  int // Round-robin index for replica selection
	metrics  *DBMetrics
}

// DBMetrics holds Prometheus metrics for database pools
type DBMetrics struct {
	primaryConns  prometheus.Gauge
	replicaConns  prometheus.Gauge
	queryDuration *prometheus.HistogramVec
	queryErrors   *prometheus.CounterVec
	poolWaitTime  *prometheus.HistogramVec
}

// NewDBMetrics creates Prometheus metrics for database monitoring
func NewDBMetrics(serviceName string) *DBMetrics {
	return &DBMetrics{
		primaryConns: promauto.NewGauge(prometheus.GaugeOpts{
			Name: fmt.Sprintf("%s_db_primary_connections", serviceName),
			Help: "Number of active connections to primary database",
		}),
		replicaConns: promauto.NewGauge(prometheus.GaugeOpts{
			Name: fmt.Sprintf("%s_db_replica_connections", serviceName),
			Help: "Number of active connections to replica databases",
		}),
		queryDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    fmt.Sprintf("%s_db_query_duration_seconds", serviceName),
			Help:    "Database query duration in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		}, []string{"query_type", "pool_type"}),
		queryErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_db_query_errors_total", serviceName),
			Help: "Total number of database query errors",
		}, []string{"query_type", "pool_type"}),
		poolWaitTime: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    fmt.Sprintf("%s_db_pool_wait_duration_seconds", serviceName),
			Help:    "Time spent waiting for a connection from the pool",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		}, []string{"pool_type"}),
	}
}

// NewPostgresPool creates a new PostgreSQL connection pool with optimized settings
// If queryTimeoutSeconds is 0 or negative, uses config.DefaultDatabaseQueryTimeout
func NewPostgresPool(cfg *config.DatabaseConfig, queryTimeoutSeconds ...int) (*pgxpool.Pool, error) {
	dsn := cfg.DSN()

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	// Connection pool settings
	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MinConns = int32(cfg.MinConns)
	poolConfig.MaxConnLifetime = time.Hour        // Recycle connections after 1 hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute // Close idle connections after 30 mins
	poolConfig.HealthCheckPeriod = time.Minute    // Check connection health every minute
	poolConfig.ConnConfig.ConnectTimeout = 10 * time.Second

	// Statement cache for better performance
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheStatement

	// Runtime parameters for performance
	poolConfig.ConnConfig.RuntimeParams["application_name"] = "ridehailing"
	poolConfig.ConnConfig.RuntimeParams["timezone"] = "UTC"

	// Enable prepared statement caching
	poolConfig.ConnConfig.RuntimeParams["plan_cache_mode"] = "auto"

	// Optimize work_mem for complex queries (only for this connection)
	// Note: This is per-operation, not per-connection
	poolConfig.ConnConfig.RuntimeParams["work_mem"] = "16MB"

	// Connection callback for additional setup
	timeoutSeconds := resolveQueryTimeout(queryTimeoutSeconds...)
	poolConfig.AfterConnect = createStatementTimeoutCallback(timeoutSeconds)

	createPool := func(ctx context.Context) (*pgxpool.Pool, error) {
		pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
		if err != nil {
			return nil, fmt.Errorf("unable to create connection pool: %w", err)
		}

		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			return nil, fmt.Errorf("unable to ping database: %w", err)
		}

		return pool, nil
	}

	if cfg.Breaker.Enabled {
		name := fmt.Sprintf("%s-db-primary", sanitizeBreakerName(cfg.ServiceName))
		if name == "-db-primary" {
			name = "database-primary"
		}

		interval := time.Duration(cfg.Breaker.IntervalSeconds) * time.Second
		if interval <= 0 {
			interval = time.Minute
		}

		timeout := time.Duration(cfg.Breaker.TimeoutSeconds) * time.Second
		if timeout <= 0 {
			timeout = 30 * time.Second
		}

		breaker := resilience.NewCircuitBreaker(resilience.Settings{
			Name:             name,
			Interval:         interval,
			Timeout:          timeout,
			FailureThreshold: uint32(max(cfg.Breaker.FailureThreshold, 1)),
			SuccessThreshold: uint32(max(cfg.Breaker.SuccessThreshold, 1)),
		}, nil)

		result, err := breaker.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
			return createPool(ctx)
		})
		if err != nil {
			return nil, err
		}
		return result.(*pgxpool.Pool), nil
	}

	return createPool(context.Background())
}

// NewDBPool creates a DBPool with primary and optional read replicas
// If queryTimeoutSeconds is 0 or negative, uses config.DefaultDatabaseQueryTimeout
func NewDBPool(cfg *config.DatabaseConfig, replicaDSNs []string, serviceName string, queryTimeoutSeconds ...int) (*DBPool, error) {
	// Create primary pool
	primary, err := NewPostgresPool(cfg, queryTimeoutSeconds...)
	if err != nil {
		return nil, fmt.Errorf("failed to create primary pool: %w", err)
	}

	dbPool := &DBPool{
		Primary:  primary,
		Replicas: make([]*pgxpool.Pool, 0, len(replicaDSNs)),
		metrics:  NewDBMetrics(serviceName),
	}

	// Create replica pools
	for i, dsn := range replicaDSNs {
		// Parse custom DSN if provided
		if dsn != "" {
			poolConfig, err := pgxpool.ParseConfig(dsn)
			if err != nil {
				return nil, fmt.Errorf("failed to parse replica DSN %d: %w", i, err)
			}

			// Apply same optimizations as primary
			poolConfig.MaxConns = int32(cfg.MaxConns)
			poolConfig.MinConns = int32(cfg.MinConns)
			poolConfig.MaxConnLifetime = time.Hour
			poolConfig.MaxConnIdleTime = 30 * time.Minute
			poolConfig.HealthCheckPeriod = time.Minute
			poolConfig.ConnConfig.ConnectTimeout = 10 * time.Second
			poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheStatement
			poolConfig.ConnConfig.RuntimeParams["application_name"] = "ridehailing-replica"
			poolConfig.ConnConfig.RuntimeParams["timezone"] = "UTC"
			poolConfig.ConnConfig.RuntimeParams["plan_cache_mode"] = "auto"
			poolConfig.ConnConfig.RuntimeParams["work_mem"] = "16MB"
			poolConfig.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"

			replicaTimeoutSeconds := resolveQueryTimeout(queryTimeoutSeconds...)
			poolConfig.AfterConnect = createStatementTimeoutCallback(replicaTimeoutSeconds)

			replica, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to create replica pool %d: %w", i, err)
			}

			if err := replica.Ping(context.Background()); err != nil {
				return nil, fmt.Errorf("failed to ping replica %d: %w", i, err)
			}

			dbPool.Replicas = append(dbPool.Replicas, replica)
		}
	}

	// Start metrics collection
	go dbPool.collectMetrics()

	return dbPool, nil
}

// GetReplica returns a replica pool using round-robin selection
// Falls back to primary if no replicas are available
func (p *DBPool) GetReplica() *pgxpool.Pool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.Replicas) == 0 {
		return p.Primary
	}

	// Round-robin selection
	replica := p.Replicas[p.rrIndex]
	p.rrIndex = (p.rrIndex + 1) % len(p.Replicas)

	return replica
}

// GetPrimary returns the primary pool (for writes)
func (p *DBPool) GetPrimary() *pgxpool.Pool {
	return p.Primary
}

// collectMetrics updates Prometheus metrics periodically
func (p *DBPool) collectMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Primary pool stats
		if p.Primary != nil {
			stats := p.Primary.Stat()
			p.metrics.primaryConns.Set(float64(stats.TotalConns()))
		}

		// Replica pool stats
		p.mu.RLock()
		var totalReplicaConns int32
		for _, replica := range p.Replicas {
			stats := replica.Stat()
			totalReplicaConns += stats.TotalConns()
		}
		p.mu.RUnlock()
		p.metrics.replicaConns.Set(float64(totalReplicaConns))
	}
}

// RecordQuery records query metrics
func (p *DBPool) RecordQuery(queryType, poolType string, duration time.Duration, err error) {
	p.metrics.queryDuration.WithLabelValues(queryType, poolType).Observe(duration.Seconds())
	if err != nil {
		p.metrics.queryErrors.WithLabelValues(queryType, poolType).Inc()
	}
}

// Close closes all database connection pools
func (p *DBPool) Close() {
	if p.Primary != nil {
		p.Primary.Close()
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, replica := range p.Replicas {
		if replica != nil {
			replica.Close()
		}
	}
}

// Close closes the database connection pool
func Close(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
	}
}

func sanitizeBreakerName(name string) string {
	trimmed := strings.TrimSpace(strings.ToLower(name))
	if trimmed == "" {
		return ""
	}
	return strings.ReplaceAll(trimmed, " ", "-")
}

func resolveQueryTimeout(queryTimeoutSeconds ...int) int {
	timeoutSeconds := config.DefaultDatabaseQueryTimeout
	if len(queryTimeoutSeconds) > 0 && queryTimeoutSeconds[0] > 0 {
		timeoutSeconds = queryTimeoutSeconds[0]
	}
	return timeoutSeconds
}

func createStatementTimeoutCallback(timeoutSeconds int) func(context.Context, *pgx.Conn) error {
	return func(ctx context.Context, conn *pgx.Conn) error {
		// Set statement timeout to prevent long-running queries
		// PostgreSQL expects statement_timeout in milliseconds as an integer
		timeoutMs := timeoutSeconds * 1000
		_, err := conn.Exec(ctx, fmt.Sprintf("SET statement_timeout = %d", timeoutMs))
		if err != nil {
			return fmt.Errorf("failed to set statement timeout: %w", err)
		}
		return nil
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
