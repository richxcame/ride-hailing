package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/richxcame/ride-hailing/internal/pricing"
	"github.com/richxcame/ride-hailing/internal/rides"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/database"
	"github.com/richxcame/ride-hailing/pkg/errors"
	"github.com/richxcame/ride-hailing/pkg/httpclient"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/ratelimit"
	redisclient "github.com/richxcame/ride-hailing/pkg/redis"
	"github.com/richxcame/ride-hailing/pkg/resilience"
	"github.com/richxcame/ride-hailing/pkg/tracing"
	"go.uber.org/zap"
)

const (
	serviceName = "rides-service"
	version     = "1.0.0"
)

func main() {
	cfg, err := config.Load(serviceName)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	defer cfg.Close()

	rootCtx, cancelKeys := context.WithCancel(context.Background())
	defer cancelKeys()

	if err := logger.Init(cfg.Server.Environment); err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	logger.Info("Starting rides service",
		zap.String("service", serviceName),
		zap.String("version", version),
		zap.String("environment", cfg.Server.Environment),
	)

	// Initialize Sentry for error tracking
	sentryConfig := errors.DefaultSentryConfig()
	sentryConfig.ServerName = serviceName
	sentryConfig.Release = version
	if err := errors.InitSentry(sentryConfig); err != nil {
		logger.Warn("Failed to initialize Sentry, continuing without error tracking", zap.Error(err))
	} else {
		defer errors.Flush(2 * time.Second)
		logger.Info("Sentry error tracking initialized successfully")
	}

	// Initialize OpenTelemetry tracer
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
			logger.Warn("Failed to initialize tracer, continuing without tracing", zap.Error(err))
		} else {
			defer func() {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := tp.Shutdown(shutdownCtx); err != nil {
					logger.Warn("Failed to shutdown tracer", zap.Error(err))
				}
			}()
			logger.Info("OpenTelemetry tracing initialized successfully")
		}
	}

	db, err := database.NewPostgresPool(&cfg.Database, cfg.Timeout.DatabaseQueryTimeout)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.Close(db)
	logger.Info("Connected to database")

	var (
		redisClient   *redisclient.Client
		limiter       *ratelimit.Limiter
		promosBreaker *resilience.CircuitBreaker
		mlEtaClient   *httpclient.Client
		mlEtaBreaker  *resilience.CircuitBreaker
	)

	if cfg.RateLimit.Enabled {
		redisClient, err = redisclient.NewRedisClient(&cfg.Redis)
		if err != nil {
			logger.Fatal("Failed to initialize redis for rate limiting", zap.Error(err))
		}

		limiter = ratelimit.NewLimiter(redisClient.Client, cfg.RateLimit)
		logger.Info("Rate limiting enabled",
			zap.Int("default_limit", cfg.RateLimit.DefaultLimit),
			zap.Int("default_burst", cfg.RateLimit.DefaultBurst),
			zap.Duration("window", cfg.RateLimit.Window()),
		)

		defer func() {
			if err := redisClient.Close(); err != nil {
				logger.Warn("Failed to close redis client", zap.Error(err))
			}
		}()
	}

	// Get Promos service URL from environment
	promosServiceURL := os.Getenv("PROMOS_SERVICE_URL")
	if promosServiceURL == "" {
		promosServiceURL = "http://localhost:8089" // Default for development
	}
	logger.Info("Promos service URL configured", zap.String("url", promosServiceURL))

	if cfg.Resilience.CircuitBreaker.Enabled {
		breakerCfg := cfg.Resilience.CircuitBreaker.SettingsFor("promos-service")
		promosBreaker = resilience.NewCircuitBreaker(resilience.Settings{
			Name:             "promos-service",
			Interval:         time.Duration(breakerCfg.IntervalSeconds) * time.Second,
			Timeout:          time.Duration(breakerCfg.TimeoutSeconds) * time.Second,
			FailureThreshold: uint32(breakerCfg.FailureThreshold),
			SuccessThreshold: uint32(breakerCfg.SuccessThreshold),
		}, nil)

		logger.Info("Circuit breaker configured for promos service",
			zap.Int("failure_threshold", breakerCfg.FailureThreshold),
			zap.Int("success_threshold", breakerCfg.SuccessThreshold),
			zap.Int("timeout_seconds", breakerCfg.TimeoutSeconds),
			zap.Int("interval_seconds", breakerCfg.IntervalSeconds),
		)
	}

	mlEtaURL := os.Getenv("ML_ETA_SERVICE_URL")
	if mlEtaURL != "" {
		mlEtaClient = httpclient.NewClient(mlEtaURL)
		if cfg.Resilience.CircuitBreaker.Enabled {
			cbCfg := cfg.Resilience.CircuitBreaker.SettingsFor("ml-eta-service")
			mlEtaBreaker = resilience.NewCircuitBreaker(
				resilience.BuildSettings("ml-eta-service", cbCfg.IntervalSeconds, cbCfg.TimeoutSeconds, cbCfg.FailureThreshold, cbCfg.SuccessThreshold),
				nil,
			)
		}
		logger.Info("ML ETA service URL configured", zap.String("url", mlEtaURL))
	}

	repo := rides.NewRepository(db)
	service := rides.NewService(repo, promosServiceURL, promosBreaker)
	if mlEtaClient != nil {
		service.EnableMLPredictions(mlEtaClient, mlEtaBreaker)
	}

	// Initialize dynamic surge pricing calculator
	surgeCalculator := pricing.NewSurgeCalculator(db)
	service.SetSurgeCalculator(surgeCalculator)
	logger.Info("Dynamic surge pricing enabled")

	handler := rides.NewHandler(service)

	jwtProvider, err := jwtkeys.NewManagerFromConfig(rootCtx, cfg.JWT, true)
	if err != nil {
		logger.Fatal("Failed to initialize JWT key manager", zap.Error(err))
	}
	jwtProvider.StartAutoRefresh(rootCtx, time.Duration(cfg.JWT.RefreshMinutes)*time.Minute)

	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(middleware.RecoveryWithSentry()) // Custom recovery with Sentry
	router.Use(middleware.SentryMiddleware())   // Sentry integration
	router.Use(middleware.CorrelationID())
	router.Use(middleware.RequestTimeout(&cfg.Timeout))
	router.Use(middleware.RequestLogger(serviceName))
	router.Use(middleware.CORS())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.SanitizeRequest())
	router.Use(middleware.Metrics(serviceName))

	// Add tracing middleware if enabled
	if tracerEnabled {
		router.Use(middleware.TracingMiddleware(serviceName))
	}

	// Add Sentry error handler (should be near the end of middleware chain)
	router.Use(middleware.ErrorHandler())

	// Health check endpoints
	router.GET("/healthz", common.HealthCheck(serviceName, version))
	router.GET("/health/live", common.LivenessProbe(serviceName, version))

	// Readiness probe with dependency checks
	healthChecks := make(map[string]func() error)
	healthChecks["database"] = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return db.Ping(ctx)
	}

	if redisClient != nil {
		healthChecks["redis"] = func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			return redisClient.Client.Ping(ctx).Err()
		}
	}

	router.GET("/health/ready", common.ReadinessProbe(serviceName, version, healthChecks))

	router.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": serviceName,
			"version": version,
		})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	handler.RegisterRoutes(router, jwtProvider, limiter, cfg.RateLimit)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	go func() {
		logger.Info("Server starting", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server stopped")
}
