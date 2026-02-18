package main

import (
	"context"
	"fmt"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/richxcame/ride-hailing/internal/admin"
	"github.com/richxcame/ride-hailing/internal/analytics"
	"github.com/richxcame/ride-hailing/internal/cancellation"
	"github.com/richxcame/ride-hailing/internal/disputes"
	"github.com/richxcame/ride-hailing/internal/documents"
	"github.com/richxcame/ride-hailing/internal/earnings"
	"github.com/richxcame/ride-hailing/internal/fraud"
	"github.com/richxcame/ride-hailing/internal/geography"
	"github.com/richxcame/ride-hailing/internal/payments"
	"github.com/richxcame/ride-hailing/internal/pricing"
	"github.com/richxcame/ride-hailing/internal/promos"
	"github.com/richxcame/ride-hailing/internal/ridetypes"
	"github.com/richxcame/ride-hailing/internal/support"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/errors"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	redisclient "github.com/richxcame/ride-hailing/pkg/redis"
	"github.com/richxcame/ride-hailing/pkg/swagger"
	"github.com/richxcame/ride-hailing/pkg/tracing"
	"go.uber.org/zap"
)

func main() {
	// Set default port for admin service if not set
	if os.Getenv("PORT") == "" {
		os.Setenv("PORT", "8088")
	}

	// Load configuration
	cfg, err := config.Load("admin")
	if err != nil {
		stdlog.Fatalf("Failed to load config: %v", err)
	}
	defer cfg.Close()

	// Initialize logger
	environment := getEnv("ENVIRONMENT", "development")
	if err := logger.Init(environment); err != nil {
		stdlog.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting admin service",
		zap.String("service", "admin-service"),
		zap.String("version", "1.0.0"),
		zap.String("environment", environment),
	)

	// Load environment variables
	jwtSecret := getEnv("JWT_SECRET", "")
	rootCtx, cancelKeys := context.WithCancel(context.Background())
	defer cancelKeys()

	jwtProvider, err := jwtkeys.NewManager(rootCtx, jwtkeys.Config{
		KeyFilePath:      getEnv("JWT_KEYS_FILE", "config/jwt_keys.json"),
		RotationInterval: time.Duration(getEnvAsInt("JWT_ROTATION_HOURS", 24*30)) * time.Hour,
		GracePeriod:      time.Duration(getEnvAsInt("JWT_ROTATION_GRACE_HOURS", 24*30)) * time.Hour,
		LegacySecret:     jwtSecret,
		ReadOnly:         true,
	})
	if err != nil {
		logger.Fatal("Failed to initialize JWT key manager", zap.Error(err))
	}
	jwtProvider.StartAutoRefresh(rootCtx, time.Duration(getEnvAsInt("JWT_KEY_REFRESH_MINUTES", 5))*time.Minute)

	// Connect to PostgreSQL
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName, cfg.Database.SSLMode)

	ctx := rootCtx
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		logger.Fatal("Failed to parse database config", zap.Error(err))
	}

	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(ctx); err != nil {
		logger.Fatal("Failed to ping database", zap.Error(err))
	}
	logger.Info("Connected to PostgreSQL database")

	// Initialize Sentry for error tracking
	sentryConfig := errors.DefaultSentryConfig()
	sentryConfig.ServerName = "admin-service"
	sentryConfig.Release = "1.0.0"
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
			Environment:    environment,
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

	// Initialize Redis for real-time driver status
	redisClient, redisErr := redisclient.NewRedisClient(&cfg.Redis)
	if redisErr != nil {
		logger.Warn("Failed to initialize Redis - driver online status will use DB only", zap.Error(redisErr))
	} else {
		defer redisClient.Close()
		logger.Info("Connected to Redis")
	}

	// Initialize fraud early so it can be injected into admin service
	fraudRepo := fraud.NewRepository(db)
	fraudSvc := fraud.NewService(fraudRepo)
	fraudHandler := fraud.NewHandler(fraudSvc)

	// Initialize repository, service, and handler
	repo := admin.NewRepository(db)
	service := admin.NewService(repo, redisClient, fraudSvc)
	handler := admin.NewHandler(service)

	// Initialize geography admin
	geoRepo := geography.NewRepository(db)
	geoSvc := geography.NewService(geoRepo)
	geoAdminHandler := geography.NewAdminHandler(geoSvc)

	// Initialize pricing admin
	pricingRepo := pricing.NewRepository(db)
	pricingSvc := pricing.NewService(pricingRepo, geoSvc, nil)
	pricingAdminHandler := pricing.NewAdminHandler(pricingRepo, pricingSvc)

	// Initialize ride types admin
	rideTypeRepo := ridetypes.NewRepository(db)
	rideTypeSvc := ridetypes.NewService(rideTypeRepo, geoSvc)
	rideTypeHandler := ridetypes.NewAdminHandler(rideTypeSvc)

	// Initialize disputes
	disputesRepo := disputes.NewRepository(db)
	disputesSvc := disputes.NewService(disputesRepo)
	disputesHandler := disputes.NewHandler(disputesSvc)

	// Initialize support
	supportRepo := support.NewRepository(db)
	supportSvc := support.NewService(supportRepo)
	supportHandler := support.NewHandler(supportSvc)

	// Initialize promos
	promosRepo := promos.NewRepository(db)
	promosSvc := promos.NewService(promosRepo)
	promosHandler := promos.NewHandler(promosSvc)

	// Initialize cancellation
	cancellationRepo := cancellation.NewRepository(db)
	cancellationSvc := cancellation.NewService(cancellationRepo, db)
	cancellationHandler := cancellation.NewHandler(cancellationSvc)

	// Initialize analytics
	analyticsRepo := analytics.NewRepository(db)
	analyticsSvc := analytics.NewService(analyticsRepo)
	analyticsHandler := analytics.NewHandler(analyticsSvc)

	// Initialize earnings
	earningsRepo := earnings.NewRepository(db)
	earningsSvc := earnings.NewService(earningsRepo)
	earningsHandler := earnings.NewHandler(earningsSvc)

	// Initialize payments (nil stripe client — admin endpoints use repo directly)
	paymentsRepo := payments.NewRepository(db)
	paymentsSvc := payments.NewService(paymentsRepo, nil, nil)
	paymentsHandler := payments.NewHandler(paymentsSvc)

	// Initialize documents (stub storage + stub driver service — admin only reviews documents)
	documentsRepo := documents.NewRepository(db)
	documentsSvc := documents.NewService(documentsRepo, &stubStorage{}, documents.ServiceConfig{})
	documentsHandler := documents.NewHandler(documentsSvc, &stubDriverService{})

	// Set up Gin router
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.NoRoute(common.NoRouteHandler())
	router.NoMethod(common.NoMethodHandler())
	router.Use(middleware.RecoveryWithSentry()) // Custom recovery with Sentry
	router.Use(middleware.SentryMiddleware())   // Sentry integration
	router.Use(middleware.CorrelationID())
	router.Use(middleware.RequestTimeout(&cfg.Timeout))
	router.Use(middleware.CORS())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.MaxBodySize(10 << 20)) // 10MB request body limit
	router.Use(middleware.SanitizeRequest())

	// Add tracing middleware if enabled
	if tracerEnabled {
		router.Use(middleware.TracingMiddleware("admin-service"))
	}

	// Add Sentry error handler (should be near the end of middleware chain)
	router.Use(middleware.ErrorHandler())

	// Health check endpoints
	router.GET("/healthz", handler.HealthCheck)
	router.GET("/health/live", func(c *gin.Context) {
		healthData := gin.H{"status": "alive", "service": "admin-service", "version": "1.0.0"}
		common.SuccessResponse(c, healthData)
	})

	// Readiness probe with dependency checks
	healthChecks := make(map[string]func() error)
	healthChecks["database"] = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return db.Ping(ctx)
	}

	router.GET("/health/ready", func(c *gin.Context) {
		for name, check := range healthChecks {
			if err := check(); err != nil {
				errorMsg := fmt.Sprintf("Service not ready: %s check failed - %s", name, err.Error())
				common.ErrorResponse(c, 503, errorMsg)
				return
			}
		}
		healthData := gin.H{"status": "ready", "service": "admin-service", "version": "1.0.0"}
		common.SuccessResponse(c, healthData)
	})

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	swagger.RegisterRoutes(router)

	// Admin API routes - all require authentication + admin role
	api := router.Group("/api/v1/admin")
	api.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	api.Use(middleware.RequireAdmin())
	{
		// Dashboard
		api.GET("/dashboard", handler.GetDashboard)

		// Dashboard analytics endpoints
		dashboard := api.Group("/dashboard")
		{
			dashboard.GET("/realtime", handler.GetRealtimeMetrics)
			dashboard.GET("/summary", handler.GetDashboardSummary)
			dashboard.GET("/revenue-trend", handler.GetRevenueTrend)
			dashboard.GET("/action-items", handler.GetActionItems)
			dashboard.GET("/activity-feed", handler.GetActivityFeed)
		}

		// User management
		users := api.Group("/users")
		{
			users.GET("", handler.GetAllUsers)
			users.GET("/:id", handler.GetUser)
			users.POST("/:id/suspend", handler.SuspendUser)
			users.POST("/:id/activate", handler.ActivateUser)
		}

		// Driver management
		drivers := api.Group("/drivers")
		{
			drivers.GET("", handler.GetAllDrivers)
			drivers.GET("/pending", handler.GetPendingDrivers)
			drivers.GET("/:id", handler.GetDriver)
			drivers.POST("/:id/approve", handler.ApproveDriver)
			drivers.POST("/:id/reject", handler.RejectDriver)
		}

		// Ride monitoring
		rides := api.Group("/rides")
		{
			rides.GET("/recent", handler.GetRecentRides)
			rides.GET("/stats", handler.GetRideStats)
			rides.GET("/:id", handler.GetRide)
		}

		// Audit logs
		api.GET("/audit-logs", handler.GetAuditLogs)

		// Geography management (countries, regions, cities, pricing zones)
		geoAdminHandler.RegisterRoutes(api)

		// Pricing management (versions, configs, multipliers, zone fees, surge)
		pricingAdminHandler.RegisterRoutes(api)

		// Ride type management
		rideTypeHandler.RegisterRoutes(api)

		// Disputes management
		disputesHandler.RegisterAdminRoutes(api)

		// Support ticket management
		supportHandler.RegisterAdminRoutes(api)

		// Promos & referrals management
		promosHandler.RegisterAdminRoutes(api)

		// Cancellation management
		cancellationHandler.RegisterAdminRoutes(api)

		// Analytics
		analyticsHandler.RegisterAdminRoutes(api)

		// Fraud detection & management
		fraudHandler.RegisterAdminRoutes(api)

		// Earnings & payouts management
		earningsHandler.RegisterAdminRoutes(api)

		// Payment management
		paymentsHandler.RegisterAdminRoutes(api)

		// Document verification management
		documentsHandler.RegisterAdminRoutes(api)
	}

	// Create HTTP server with timeouts
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Admin service starting", zap.String("port", cfg.Server.Port))
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

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	if parsed, err := strconv.Atoi(value); err == nil {
		return parsed
	}
	return defaultValue
}
