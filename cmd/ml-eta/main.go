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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/richxcame/ride-hailing/internal/mleta"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/database"
	"github.com/richxcame/ride-hailing/pkg/errors"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	redisClient "github.com/richxcame/ride-hailing/pkg/redis"
	"github.com/richxcame/ride-hailing/pkg/swagger"
	"go.uber.org/zap"
)

const serviceName = "ml-eta"

func main() {
	// Set default port for ml-eta service if not set
	if os.Getenv("PORT") == "" {
		os.Setenv("PORT", "8093")
	}
	// Load configuration
	cfg, err := config.Load("ml-eta")
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

	rootCtx, cancelKeys := context.WithCancel(context.Background())
	defer cancelKeys()

	// Initialize database (pgxpool)
	dbPool, err := database.NewPostgresPool(&cfg.Database, cfg.Timeout.DatabaseQueryTimeout)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.Close(dbPool)

	// Initialize Redis
	redis, err := redisClient.NewRedisClient(&cfg.Redis)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redis.Close()

	// Initialize Sentry for error tracking
	sentryConfig := errors.DefaultSentryConfig()
	sentryConfig.ServerName = serviceName
	sentryConfig.Release = "1.0.0"
	if err := errors.InitSentry(sentryConfig); err != nil {
		logger.Warn("Failed to initialize Sentry, continuing without error tracking", zap.Error(err))
	} else {
		defer errors.Flush(2 * time.Second)
		logger.Info("Sentry error tracking initialized successfully")
	}

	// Initialize ML ETA service
	repo := mleta.NewRepository(dbPool, redis)
	service := mleta.NewService(repo, redis)
	handler := mleta.NewHandler(service)

	jwtProvider, err := jwtkeys.NewManagerFromConfig(rootCtx, cfg.JWT, true)
	if err != nil {
		logger.Fatal("Failed to initialize JWT key manager", zap.Error(err))
	}
	jwtProvider.StartAutoRefresh(rootCtx, time.Duration(cfg.JWT.RefreshMinutes)*time.Minute)

	// Start ML model training worker
	go service.StartModelTrainingWorker(rootCtx)

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
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
	router.Use(middleware.Metrics(cfg.Server.ServiceName))

	// Add Sentry error handler (should be near the end of middleware chain)
	router.Use(middleware.ErrorHandler())

	// Health check endpoints
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "ml-eta", "version": "1.0.0"})
	})
	router.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "alive", "service": "ml-eta", "version": "1.0.0"})
	})

	// Readiness probe with dependency checks
	healthChecks := make(map[string]func() error)
	healthChecks["database"] = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return dbPool.Ping(ctx)
	}
	healthChecks["redis"] = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return redis.Client.Ping(ctx).Err()
	}

	router.GET("/health/ready", func(c *gin.Context) {
		allHealthy := true
		for name, check := range healthChecks {
			if err := check(); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "service": "ml-eta", "failed_check": name, "error": err.Error()})
				allHealthy = false
				return
			}
		}
		if allHealthy {
			c.JSON(http.StatusOK, gin.H{"status": "ready", "service": "ml-eta", "version": "1.0.0"})
		}
	})

	// ML ETA API routes
	api := router.Group("/api/v1/eta")
	{
		// Public endpoints
		api.POST("/predict", handler.PredictETA)            // Predict ETA for a route
		api.POST("/predict/batch", handler.BatchPredictETA) // Batch ETA predictions

		// Admin endpoints (require JWT)
		admin := api.Group("")
		admin.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
		admin.Use(middleware.RequireAdmin())
		{
			admin.POST("/train", handler.TriggerModelTraining)     // Trigger model retraining
			admin.GET("/model/stats", handler.GetModelStats)       // Get model performance stats
			admin.GET("/model/accuracy", handler.GetModelAccuracy) // Get model accuracy metrics
			admin.POST("/model/tune", handler.TuneHyperparameters) // Tune ML model hyperparameters
		}

		// Analytics endpoints
		analytics := api.Group("/analytics")
		analytics.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
		{
			analytics.GET("/predictions", handler.GetPredictionHistory) // Historical predictions
			analytics.GET("/accuracy", handler.GetAccuracyTrends)       // Accuracy over time
			analytics.GET("/features", handler.GetFeatureImportance)    // Feature importance
		}
	}

	// Prometheus metrics
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	swagger.RegisterRoutes(router)

	// Start server
	port := cfg.Server.Port
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		logger.Info("ML ETA Service starting", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down ML ETA Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("ML ETA Service stopped")
}
