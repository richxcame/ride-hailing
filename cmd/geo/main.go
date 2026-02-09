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
	"github.com/richxcame/ride-hailing/internal/geo"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/errors"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	redisClient "github.com/richxcame/ride-hailing/pkg/redis"
	"github.com/richxcame/ride-hailing/pkg/resilience"
	"github.com/richxcame/ride-hailing/pkg/swagger"
	"github.com/richxcame/ride-hailing/pkg/tracing"
	"go.uber.org/zap"
)

const (
	serviceName = "geo-service"
	version     = "1.0.0"
)

func main() {
	// Set default port for geo service if not set
	if os.Getenv("PORT") == "" {
		os.Setenv("PORT", "8083")
	}
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

	logger.Info("Starting geo service",
		zap.String("service", serviceName),
		zap.String("version", version),
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

	redis, err := redisClient.NewRedisClient(&cfg.Redis)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redis.Close()
	logger.Info("Connected to Redis")

	service := geo.NewService(redis)

	// Initialize location batching pipeline
	locationBuffer := geo.NewLocationBuffer(redis, geo.DefaultLocationBufferConfig())
	service.SetLocationBuffer(locationBuffer)
	defer locationBuffer.Stop()
	logger.Info("Location batching pipeline enabled")

	// Initialize geocoding service
	googleAPIKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	if googleAPIKey == "" {
		logger.Warn("GOOGLE_MAPS_API_KEY not set, geocoding will not work")
	}
	geocodingSvc := geo.NewGeocodingService(googleAPIKey, redis)
	if cfg.Resilience.CircuitBreaker.Enabled {
		cbCfg := cfg.Resilience.CircuitBreaker.SettingsFor("google-maps-api")
		geocodingBreaker := resilience.NewCircuitBreaker(
			resilience.BuildSettings(fmt.Sprintf("%s-geocoding", serviceName), cbCfg.IntervalSeconds, cbCfg.TimeoutSeconds, cbCfg.FailureThreshold, cbCfg.SuccessThreshold),
			resilience.GracefulDegradation("google-maps-api"),
		)
		geocodingSvc.SetCircuitBreaker(geocodingBreaker)
		logger.Info("Circuit breaker enabled for geocoding API")
	}
	if region := os.Getenv("GEOCODING_REGION_BIAS"); region != "" {
		geocodingSvc.RegionBias = region
	}
	if lang := os.Getenv("GEOCODING_LANGUAGE"); lang != "" {
		geocodingSvc.LanguageBias = lang
	}

	// Initialize real-time ETA tracker
	realtimeServiceURL := os.Getenv("REALTIME_SERVICE_URL")
	if realtimeServiceURL == "" {
		realtimeServiceURL = "http://localhost:8086"
	}
	etaTracker := geo.NewETATracker(redis, realtimeServiceURL)
	service.SetETATracker(etaTracker)
	logger.Info("Real-time ETA tracking enabled")

	handler := geo.NewHandler(service, geocodingSvc)

	jwtProvider, err := jwtkeys.NewManagerFromConfig(rootCtx, cfg.JWT, true)
	if err != nil {
		logger.Fatal("Failed to initialize JWT key manager", zap.Error(err))
	}
	jwtProvider.StartAutoRefresh(rootCtx, time.Duration(cfg.JWT.RefreshMinutes)*time.Minute)

	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.NoRoute(common.NoRouteHandler())
	router.NoMethod(common.NoMethodHandler())
	router.Use(middleware.RecoveryWithSentry()) // Custom recovery with Sentry
	router.Use(middleware.SentryMiddleware())   // Sentry integration
	router.Use(middleware.CorrelationID())
	router.Use(middleware.RequestTimeout(&cfg.Timeout))
	router.Use(middleware.RequestLogger(serviceName))
	router.Use(middleware.CORS())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.MaxBodySize(10 << 20)) // 10MB request body limit
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
	healthChecks["redis"] = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return redis.Client.Ping(ctx).Err()
	}

	router.GET("/health/ready", common.ReadinessProbe(serviceName, version, healthChecks))

	router.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"service": serviceName, "version": version})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	swagger.RegisterRoutes(router)

	handler.RegisterRoutes(router, jwtProvider)

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
