package main

import (
	"context"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/richxcame/ride-hailing/internal/promos"
	"github.com/richxcame/ride-hailing/pkg/cache"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/errors"
	redisclient "github.com/richxcame/ride-hailing/pkg/redis"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/ratelimit"
	"github.com/richxcame/ride-hailing/pkg/swagger"
	"github.com/richxcame/ride-hailing/pkg/tracing"
	"go.uber.org/zap"
)

func main() {
	// Set default port for promos service if not set
	if os.Getenv("PORT") == "" {
		os.Setenv("PORT", "8089")
	}
	// Load configuration
	cfg, err := config.Load("promos")
	if err != nil {
		stdlog.Fatalf("Failed to load config: %v", err)
	}
	defer cfg.Close()

	// Initialize logger
	environment := os.Getenv("ENVIRONMENT")
	if environment == "" {
		environment = "development"
	}
	if err := logger.Init(environment); err != nil {
		stdlog.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	port := cfg.Server.Port

	rootCtx, cancelKeys := context.WithCancel(context.Background())
	defer cancelKeys()

	jwtProvider, err := jwtkeys.NewManagerFromConfig(rootCtx, cfg.JWT, true)
	if err != nil {
		logger.Fatal("Failed to initialize JWT key manager", zap.Error(err))
	}
	jwtProvider.StartAutoRefresh(rootCtx, time.Duration(cfg.JWT.RefreshMinutes)*time.Minute)

	// Connect to PostgreSQL
	ctx := rootCtx
	dsn := cfg.Database.DSN()
	dbConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		logger.Fatal("Failed to parse database config", zap.Error(err))
	}

	db, err := pgxpool.NewWithConfig(ctx, dbConfig)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(ctx); err != nil {
		logger.Fatal("Failed to ping database", zap.Error(err))
	}
	logger.Info("Connected to PostgreSQL database")

	// Set up Gin router
	serviceName := "promos-service"
	version := "1.0.0"

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
			logger.Warn("Failed to initialize tracer", zap.Error(err))
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

	// Initialize Redis for caching
	redisClient, err := redisclient.NewRedisClient(&cfg.Redis)
	if err != nil {
		logger.Warn("Failed to initialize Redis - caching disabled", zap.Error(err))
	} else {
		defer func() {
			if err := redisClient.Close(); err != nil {
				logger.Warn("Failed to close redis", zap.Error(err))
			}
		}()
	}

	// Initialize rate limiter
	var limiter *ratelimit.Limiter
	if redisClient != nil && cfg.RateLimit.Enabled {
		limiter = ratelimit.NewLimiter(redisClient.Client, cfg.RateLimit)
		logger.Info("Rate limiting enabled",
			zap.Int("default_limit", cfg.RateLimit.DefaultLimit),
			zap.Int("default_burst", cfg.RateLimit.DefaultBurst),
			zap.Duration("window", cfg.RateLimit.Window()),
		)
	}

	// Create repository, service and handler
	repo := promos.NewRepository(db)
	service := promos.NewService(repo)
	if redisClient != nil {
		service.SetCache(cache.NewManager(redisClient))
		logger.Info("Cache layer enabled for promos service")
	}
	handler := promos.NewHandler(service)

	// Set up Gin router
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
	if limiter != nil {
		router.Use(middleware.RateLimit(limiter, cfg.RateLimit))
	}
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
		c.JSON(200, gin.H{"status": "alive", "service": serviceName, "version": version})
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
				c.JSON(503, gin.H{"status": "not ready", "service": serviceName, "failed_check": name, "error": err.Error()})
				allHealthy = false
				return
			}
		}
		if allHealthy {
			c.JSON(200, gin.H{"status": "ready", "service": serviceName, "version": version})
		}
	})

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	swagger.RegisterRoutes(router)

	// API routes
	api := router.Group("/api/v1")
	{
		// Public endpoints
		api.GET("/ride-types", handler.GetRideTypes)
		api.POST("/ride-types/calculate-fare", handler.CalculateFare)

		// Authenticated endpoints
		authenticated := api.Group("")
		authenticated.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
		{
			// Promo codes
			authenticated.POST("/promo-codes/validate", handler.ValidatePromoCode)

			// Referral codes
			authenticated.GET("/referrals/my-code", handler.GetMyReferralCode)
			authenticated.GET("/referrals/my-earnings", handler.GetMyReferralEarnings)
			authenticated.POST("/referrals/apply", handler.ApplyReferralCode)
		}

		// Admin endpoints
		admin := api.Group("/admin")
		admin.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
		admin.Use(middleware.RequireAdmin())
		{
			admin.POST("/promo-codes", handler.CreatePromoCode)
			admin.GET("/promo-codes", handler.GetAllPromoCodes)
			admin.GET("/promo-codes/:id", handler.GetPromoCode)
			admin.PATCH("/promo-codes/:id", handler.UpdatePromoCode)
			admin.DELETE("/promo-codes/:id", handler.DeactivatePromoCode)
			admin.GET("/promo-codes/:id/usage", handler.GetPromoCodeUsageStats)
			admin.GET("/referral-codes", handler.GetAllReferralCodes)
			admin.GET("/referrals/:id", handler.GetReferralDetails)
		}
	}

	// Create HTTP server with timeouts
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Promos service starting", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with 5 second timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server stopped")
}
