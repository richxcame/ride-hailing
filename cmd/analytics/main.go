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
	"github.com/richxcame/ride-hailing/internal/analytics"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/database"
	"github.com/richxcame/ride-hailing/pkg/errors"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/swagger"
	"github.com/richxcame/ride-hailing/pkg/tracing"
	"go.uber.org/zap"
)

const (
	serviceName = "analytics-service"
	version     = "1.0.0"
)

func main() {
	// Set default port for analytics service if not set
	if os.Getenv("PORT") == "" {
		os.Setenv("PORT", "8091")
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

	logger.Info("Starting analytics service",
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

	repo := analytics.NewRepository(db)
	service := analytics.NewService(repo)
	handler := analytics.NewHandler(service)

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
	router.GET("/healthz", handler.HealthCheck)
	router.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "alive", "service": serviceName, "version": version})
	})

	// Readiness probe with dependency checks
	healthChecks := make(map[string]func() error)
	healthChecks["database"] = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return db.Ping(ctx)
	}

	router.GET("/health/ready", func(c *gin.Context) {
		allHealthy := true
		for name, check := range healthChecks {
			if err := check(); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "service": serviceName, "failed_check": name, "error": err.Error()})
				allHealthy = false
				return
			}
		}
		if allHealthy {
			c.JSON(http.StatusOK, gin.H{"status": "ready", "service": serviceName, "version": version})
		}
	})

	router.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": serviceName,
			"version": version,
		})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	swagger.RegisterRoutes(router)

	// Admin-only analytics endpoints
	api := router.Group("/api/v1/analytics")
	api.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	api.Use(middleware.RequireAdmin())
	{
		// Dashboard & Overview
		api.GET("/dashboard", handler.GetDashboardMetrics)
		api.GET("/revenue", handler.GetRevenueMetrics)
		api.GET("/financial-report", handler.GetFinancialReport)

		// Time-Series & Charts
		api.GET("/revenue/timeseries", handler.GetRevenueTimeSeries)
		api.GET("/rides/hourly", handler.GetHourlyDistribution)

		// Driver Analytics
		api.GET("/drivers/performance", handler.GetDriverAnalytics)
		api.GET("/top-drivers", handler.GetTopDrivers)
		api.GET("/top-drivers/detailed", handler.GetTopDriversDetailed)

		// Rider Analytics
		api.GET("/riders/growth", handler.GetRiderGrowth)

		// Ride Metrics
		api.GET("/rides/metrics", handler.GetRideMetrics)
		api.GET("/ride-types", handler.GetRideTypeStats)

		// Geographic
		api.GET("/heat-map", handler.GetDemandHeatMap)
		api.GET("/demand-zones", handler.GetDemandZones)

		// Comparison
		api.GET("/comparison", handler.GetPeriodComparison)

		// Marketing
		api.GET("/promo-codes", handler.GetPromoCodePerformance)
		api.GET("/referrals", handler.GetReferralMetrics)
	}

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
