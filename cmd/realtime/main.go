package main

import (
	"context"
	"database/sql"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/richxcame/ride-hailing/internal/geo"
	"github.com/richxcame/ride-hailing/internal/matching"
	"github.com/richxcame/ride-hailing/internal/realtime"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/errors"
	"github.com/richxcame/ride-hailing/pkg/eventbus"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/redis"
	"github.com/richxcame/ride-hailing/pkg/swagger"
	"github.com/richxcame/ride-hailing/pkg/tracing"
	ws "github.com/richxcame/ride-hailing/pkg/websocket"
	"go.uber.org/zap"
)

// stubRidesRepository provides minimal implementation for matching service
type stubRidesRepository struct {
	db *sql.DB
}

func (r *stubRidesRepository) UpdateRideDriver(ctx context.Context, rideID, driverID uuid.UUID) error {
	query := `UPDATE rides SET driver_id = $1, status = 'accepted', updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, driverID, rideID)
	return err
}

func (r *stubRidesRepository) GetRideByID(ctx context.Context, rideID uuid.UUID) (interface{}, error) {
	var status string
	query := `SELECT status FROM rides WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, rideID).Scan(&status)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"status": status}, nil
}

// geoServiceAdapter adapts geo.Service to matching.GeoService interface
type geoServiceAdapter struct {
	*geo.Service
}

func (a *geoServiceAdapter) FindAvailableDrivers(ctx context.Context, latitude, longitude float64, maxDrivers int) ([]*matching.GeoDriverLocation, error) {
	drivers, err := a.Service.FindAvailableDrivers(ctx, latitude, longitude, maxDrivers)
	if err != nil {
		return nil, err
	}

	// Convert geo.DriverLocation to matching.GeoDriverLocation
	result := make([]*matching.GeoDriverLocation, len(drivers))
	for i, d := range drivers {
		result[i] = &matching.GeoDriverLocation{
			DriverID:  d.DriverID,
			Latitude:  d.Latitude,
			Longitude: d.Longitude,
			Status:    d.Status,
		}
	}
	return result, nil
}

func (a *geoServiceAdapter) CalculateDistance(latitude1, longitude1, latitude2, longitude2 float64) float64 {
	return a.Service.CalculateDistance(latitude1, longitude1, latitude2, longitude2)
}

func main() {
	// Set default port for realtime service if not set
	if os.Getenv("PORT") == "" {
		os.Setenv("PORT", "8086")
	}
	// Load configuration
	cfg, err := config.Load("realtime")
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
	dsn := cfg.Database.DSN()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		logger.Fatal("Failed to ping database", zap.Error(err))
	}
	logger.Info("Connected to PostgreSQL database")

	// Initialize Sentry for error tracking
	sentryConfig := errors.DefaultSentryConfig()
	sentryConfig.ServerName = "realtime-service"
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

	// Connect to Redis
	redisClient, err := redis.NewRedisClient(&cfg.Redis)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	logger.Info("Connected to Redis")

	// Create WebSocket hub
	hub := ws.NewHub()
	go hub.Run()
	logger.Info("WebSocket hub started")

	// Connect to NATS event bus
	var eventBus *eventbus.Bus
	if cfg.NATS.Enabled {
		eventBusCfg := eventbus.Config{
			URL:        cfg.NATS.URL,
			Name:       "realtime-service",
			StreamName: cfg.NATS.StreamName,
		}
		eventBus, err = eventbus.New(eventBusCfg)
		if err != nil {
			logger.Fatal("Failed to connect to NATS", zap.Error(err))
		}
		logger.Info("Connected to NATS event bus")
	}

	// Initialize services for matching
	geoService := geo.NewService(redisClient)
	geoAdapter := &geoServiceAdapter{Service: geoService}

	// Create stub rides repository (matching service doesn't use it heavily)
	stubRidesRepo := &stubRidesRepository{db: db}

	// Create matching service and start it
	if eventBus != nil {
		matchingConfig := matching.DefaultMatchingConfig()
		matchingSvc := matching.NewService(
			geoAdapter,
			stubRidesRepo,
			hub,
			eventBus.Conn(),
			redisClient,
			matchingConfig,
		)

		ctx := context.Background()
		if err := matchingSvc.Start(ctx); err != nil {
			logger.Fatal("Failed to start matching service", zap.Error(err))
		}
		logger.Info("Matching service started")
	} else {
		logger.Warn("NATS not enabled, matching service will not start")
	}

	// Create service and handler
	log := logger.Get()
	service := realtime.NewService(hub, db, redisClient, log)
	handler := realtime.NewHandler(service, log)

	// Set up Gin router with proper middleware stack
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.NoRoute(common.NoRouteHandler())
	router.NoMethod(common.NoMethodHandler())

	router.Use(middleware.Recovery())
	router.Use(middleware.RecoveryWithSentry()) // Custom recovery with Sentry
	router.Use(middleware.SentryMiddleware())   // Sentry integration
	router.Use(middleware.CorrelationID())
	router.Use(middleware.RequestTimeout(&cfg.Timeout))
	router.Use(middleware.RequestLogger("realtime-service"))
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.MaxBodySize(10 << 20)) // 10MB request body limit
	router.Use(middleware.SanitizeRequest())
	router.Use(middleware.Metrics("realtime-service"))

	// Add tracing middleware if enabled
	if tracerEnabled {
		router.Use(middleware.TracingMiddleware("realtime-service"))
	}

	// Add Sentry error handler (should be near the end of middleware chain)
	router.Use(middleware.ErrorHandler())

	// CORS configuration
	corsConfig := cors.DefaultConfig()
	// Parse CORS origins from config (comma-separated for production)
	if cfg.Server.CORSOrigins != "" {
		origins := strings.Split(cfg.Server.CORSOrigins, ",")
		// Trim whitespace from each origin
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
		corsConfig.AllowOrigins = origins
		logger.Info("CORS configured with origins", zap.Strings("origins", origins))
	} else {
		// Development fallback
		corsConfig.AllowOrigins = []string{"http://localhost:3000"}
		logger.Info("CORS configured for development (localhost:3000)")
	}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	corsConfig.AllowCredentials = true
	router.Use(cors.New(corsConfig))

	// Health check endpoints
	router.GET("/healthz", handler.HealthCheck)
	router.GET("/health/live", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "alive", "service": "realtime-service", "version": "1.0.0"})
	})

	// Readiness probe with dependency checks
	healthChecks := make(map[string]func() error)
	healthChecks["database"] = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return db.PingContext(ctx)
	}
	healthChecks["redis"] = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return redisClient.Client.Ping(ctx).Err()
	}

	router.GET("/health/ready", func(c *gin.Context) {
		allHealthy := true
		for name, check := range healthChecks {
			if err := check(); err != nil {
				c.JSON(503, gin.H{"status": "not ready", "service": "realtime-service", "failed_check": name, "error": err.Error()})
				allHealthy = false
				return
			}
		}
		if allHealthy {
			c.JSON(200, gin.H{"status": "ready", "service": "realtime-service", "version": "1.0.0"})
		}
	})

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	swagger.RegisterRoutes(router)

	// API routes
	api := router.Group("/api/v1")
	{
		// WebSocket connection (requires authentication)
		api.GET("/ws", middleware.AuthMiddlewareWithProvider(jwtProvider), handler.HandleWebSocket)

		// Chat history
		api.GET("/rides/:ride_id/chat", middleware.AuthMiddlewareWithProvider(jwtProvider), handler.GetChatHistory)

		// Driver location
		api.GET("/drivers/:driver_id/location", middleware.AuthMiddlewareWithProvider(jwtProvider), handler.GetDriverLocation)

		// Stats (admin only)
		api.GET("/stats", middleware.AuthMiddlewareWithProvider(jwtProvider), middleware.RequireAdmin(), handler.GetStats)

		// Internal endpoints (for other services to broadcast)
		internal := api.Group("/internal")
		internal.Use(middleware.InternalAPIKey())
		{
			internal.POST("/broadcast/ride", handler.BroadcastRideUpdate)
			internal.POST("/broadcast/user", handler.BroadcastToUser)
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
		logger.Info("Real-time service starting", zap.String("port", port))
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
