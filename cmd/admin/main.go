package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/richxcame/ride-hailing/internal/admin"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/errors"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
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
		log.Fatalf("Failed to load config: %v", err)
	}
	defer cfg.Close()

	// Initialize logger
	environment := getEnv("ENVIRONMENT", "development")
	if err := logger.Init(environment); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting admin service",
		zap.String("service", "admin-service"),
		zap.String("version", "1.0.0"),
		zap.String("environment", environment),
	)

	// Load environment variables
	jwtSecret := getEnv("JWT_SECRET", "your-secret-key")
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
		log.Fatalf("Failed to initialize JWT key manager: %v", err)
	}
	jwtProvider.StartAutoRefresh(rootCtx, time.Duration(getEnvAsInt("JWT_KEY_REFRESH_MINUTES", 5))*time.Minute)

	// Connect to PostgreSQL
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName, cfg.Database.SSLMode)

	ctx := rootCtx
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("Failed to parse database config: %v", err)
	}

	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
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

	// Initialize repository, service, and handler
	repo := admin.NewRepository(db)
	service := admin.NewService(repo)
	handler := admin.NewHandler(service)

	// Set up Gin router
	router := gin.New()
	router.Use(middleware.RecoveryWithSentry()) // Custom recovery with Sentry
	router.Use(middleware.SentryMiddleware())   // Sentry integration
	router.Use(middleware.CorrelationID())
	router.Use(middleware.RequestTimeout(&cfg.Timeout))
	router.Use(middleware.CORS())
	router.Use(middleware.SecurityHeaders())
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

	// Admin API routes - all require authentication + admin role
	api := router.Group("/api/v1/admin")
	api.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	api.Use(middleware.RequireAdmin())
	{
		// Dashboard
		api.GET("/dashboard", handler.GetDashboard)

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
			drivers.GET("/pending", handler.GetPendingDrivers)
			drivers.POST("/:id/approve", handler.ApproveDriver)
			drivers.POST("/:id/reject", handler.RejectDriver)
		}

		// Ride monitoring
		rides := api.Group("/rides")
		{
			rides.GET("/recent", handler.GetRecentRides)
			rides.GET("/stats", handler.GetRideStats)
		}
	}

	// Start server
	addr := ":" + cfg.Server.Port
	logger.Info("Admin service starting", zap.String("port", cfg.Server.Port))
	if err := router.Run(addr); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
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
