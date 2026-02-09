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
	"github.com/richxcame/ride-hailing/internal/currency"
	"github.com/richxcame/ride-hailing/internal/geography"
	"github.com/richxcame/ride-hailing/internal/negotiation"
	"github.com/richxcame/ride-hailing/internal/pricing"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/database"
	"github.com/richxcame/ride-hailing/pkg/errors"
	"github.com/richxcame/ride-hailing/pkg/eventbus"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	redisclient "github.com/richxcame/ride-hailing/pkg/redis"
	"github.com/richxcame/ride-hailing/pkg/swagger"
	"github.com/richxcame/ride-hailing/pkg/tracing"
	"github.com/richxcame/ride-hailing/pkg/websocket"
	"go.uber.org/zap"
)

const (
	serviceName = "negotiation-service"
	version     = "1.0.0"
)

func main() {
	// Set default port for negotiation service if not set
	if os.Getenv("PORT") == "" {
		os.Setenv("PORT", "8095")
	}

	// Load configuration
	cfg, err := config.Load(serviceName)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	defer cfg.Close()

	rootCtx, cancelKeys := context.WithCancel(context.Background())
	defer cancelKeys()

	// Initialize logger
	if err := logger.Init(cfg.Server.Environment); err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	log := logger.Get()
	log.Info("Starting negotiation service", zap.String("version", version))

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

	// Initialize database
	db, err := database.NewPostgresPool(&cfg.Database, cfg.Timeout.DatabaseQueryTimeout)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.Close(db)

	log.Info("Connected to database")

	// Initialize Redis for real-time features
	redisClient, redisErr := redisclient.NewRedisClient(&cfg.Redis)
	if redisErr != nil {
		logger.Warn("Failed to initialize Redis - some features may be limited", zap.Error(redisErr))
	} else {
		defer redisClient.Close()
		logger.Info("Redis connected for negotiation state management")
	}

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run()
	log.Info("WebSocket hub started")

	// Initialize dependent services
	geoRepo := geography.NewRepository(db)
	geoService := geography.NewService(geoRepo)

	currencyRepo := currency.NewRepository(db)
	currencyService := currency.NewService(currencyRepo, "USD")

	pricingRepo := pricing.NewRepository(db)
	pricingService := pricing.NewService(pricingRepo, geoService, currencyService)

	// Initialize negotiation service
	negotiationRepo := negotiation.NewRepository(db)
	negotiationService := negotiation.NewService(negotiationRepo, pricingService, geoService)
	negotiationHandler := negotiation.NewHandler(negotiationService)

	// Initialize NATS event bus
	var eventBus *eventbus.Bus
	if cfg.NATS.Enabled && cfg.NATS.URL != "" {
		bus, err := eventbus.New(eventbus.Config{
			URL:        cfg.NATS.URL,
			Name:       serviceName,
			StreamName: cfg.NATS.StreamName,
		})
		if err != nil {
			logger.Warn("Failed to connect to NATS - event-driven features disabled", zap.Error(err))
		} else {
			eventBus = bus
			defer bus.Close()
			logger.Info("NATS event bus connected for negotiation events")

			// Set event bus on service
			negotiationService.SetEventBus(bus)
		}
	}

	// Start session expiry worker
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-rootCtx.Done():
				return
			case <-ticker.C:
				expired, err := negotiationService.ExpireStale(rootCtx)
				if err != nil {
					logger.Error("Failed to expire sessions", zap.Error(err))
				} else if expired > 0 {
					logger.Info("Expired negotiation sessions", zap.Int("count", expired))
				}
			}
		}
	}()

	jwtProvider, err := jwtkeys.NewManagerFromConfig(rootCtx, cfg.JWT, true)
	if err != nil {
		logger.Fatal("Failed to initialize JWT key manager", zap.Error(err))
	}
	jwtProvider.StartAutoRefresh(rootCtx, time.Duration(cfg.JWT.RefreshMinutes)*time.Minute)

	// Setup Gin router
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.NoRoute(common.NoRouteHandler())
	router.NoMethod(common.NoMethodHandler())

	// Global middleware
	router.Use(middleware.RecoveryWithSentry())
	router.Use(middleware.SentryMiddleware())
	router.Use(middleware.CorrelationID())
	router.Use(middleware.RequestTimeout(&cfg.Timeout))
	router.Use(middleware.RequestLogger(serviceName))
	router.Use(middleware.CORS())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.MaxBodySize(5 << 20)) // 5MB request body limit
	router.Use(middleware.SanitizeRequest())
	router.Use(middleware.Metrics(serviceName))

	// Add tracing middleware if enabled
	if tracerEnabled {
		router.Use(middleware.TracingMiddleware(serviceName))
	}

	// Add Sentry error handler
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
		c.JSON(http.StatusOK, gin.H{"service": serviceName, "version": version})
	})

	// Metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	swagger.RegisterRoutes(router)

	// API routes with authentication
	api := router.Group("/api/v1")
	api.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))

	// Register negotiation routes
	negotiationHandler.RegisterRoutes(api)

	// WebSocket endpoint for real-time updates
	router.GET("/ws/negotiation", func(c *gin.Context) {
		websocket.HandleWebSocket(c, wsHub, jwtProvider)
	})

	// Admin endpoints (for internal use)
	admin := router.Group("/admin")
	admin.GET("/stats", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":       serviceName,
			"ws_clients":    wsHub.GetClientCount(),
			"ws_negotiations": wsHub.GetNegotiationCount(),
			"event_bus":     eventBus != nil,
		})
	})

	// Setup HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Info("Server starting", zap.String("port", cfg.Server.Port), zap.String("environment", cfg.Server.Environment))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Graceful shutdown with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Server stopped")
}
