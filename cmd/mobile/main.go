package main

import (
	"context"
	"fmt"
	stdlog "log"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/richxcame/ride-hailing/internal/favorites"
	"github.com/richxcame/ride-hailing/internal/rides"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/errors"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/swagger"
	"github.com/richxcame/ride-hailing/pkg/tracing"
	"go.uber.org/zap"
)

func main() {
	// Set default port for mobile service if not set
	if os.Getenv("PORT") == "" {
		os.Setenv("PORT", "8087")
	}
	// Load configuration
	cfg, err := config.Load("mobile")
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

	// Load environment variables
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "ride_hailing")
	jwtSecret := getEnv("JWT_SECRET", "your-secret-key")
	port := getEnv("PORT", "8087")
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
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

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
	sentryConfig.ServerName = "mobile-service"
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
			Environment:    getEnv("ENVIRONMENT", "development"),
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

	// Get Promos service URL from environment
	promosServiceURL := getEnv("PROMOS_SERVICE_URL", "http://localhost:8089")
	logger.Info("Promos service URL configured", zap.String("url", promosServiceURL))

	// Initialize repositories
	ridesRepo := rides.NewRepository(db)
	favoritesRepo := favorites.NewRepository(db)

	// Initialize services
	ridesService := rides.NewService(ridesRepo, promosServiceURL, nil)
	favoritesService := favorites.NewService(favoritesRepo)

	// Initialize handlers
	ridesHandler := rides.NewHandler(ridesService)
	favoritesHandler := favorites.NewHandler(favoritesService)

	// Set up Gin router
	router := gin.New()
	router.Use(middleware.RecoveryWithSentry()) // Custom recovery with Sentry
	router.Use(middleware.SentryMiddleware())   // Sentry integration
	router.Use(middleware.CorrelationID())
	router.Use(middleware.RequestTimeout(&cfg.Timeout))
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.MaxBodySize(10 << 20)) // 10MB request body limit
	router.Use(middleware.SanitizeRequest())

	// Add tracing middleware if enabled
	if tracerEnabled {
		router.Use(middleware.TracingMiddleware("mobile-service"))
	}

	// Add Sentry error handler (should be near the end of middleware chain)
	router.Use(middleware.ErrorHandler())

	// Health check endpoints
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "mobile-api", "version": "1.0.0"})
	})
	router.GET("/health/live", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "alive", "service": "mobile-api", "version": "1.0.0"})
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
				c.JSON(503, gin.H{"status": "not ready", "service": "mobile-api", "failed_check": name, "error": err.Error()})
				allHealthy = false
				return
			}
		}
		if allHealthy {
			c.JSON(200, gin.H{"status": "ready", "service": "mobile-api", "version": "1.0.0"})
		}
	})

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	swagger.RegisterRoutes(router)

	// API routes with authentication
	api := router.Group("/api/v1")
	api.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		// Ride history endpoints
		rides := api.Group("/rides")
		{
			rides.GET("/history", ridesHandler.GetRideHistory)
			rides.GET("/:id/receipt", ridesHandler.GetRideReceipt)
		}

		// Favorite locations endpoints
		favs := api.Group("/favorites")
		{
			favs.POST("", favoritesHandler.CreateFavorite)
			favs.GET("", favoritesHandler.GetFavorites)
			favs.GET("/:id", favoritesHandler.GetFavorite)
			favs.PUT("/:id", favoritesHandler.UpdateFavorite)
			favs.DELETE("/:id", favoritesHandler.DeleteFavorite)
		}

		// Ratings endpoints
		api.POST("/rides/:id/rate", ridesHandler.RateRide)

		// User profile endpoints
		api.GET("/profile", ridesHandler.GetUserProfile)
		api.PUT("/profile", ridesHandler.UpdateUserProfile)
	}

	// Start server
	addr := ":" + port
	logger.Info("Mobile API service starting", zap.String("port", port))
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
