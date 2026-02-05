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
	"github.com/richxcame/ride-hailing/internal/cancellation"
	"github.com/richxcame/ride-hailing/internal/chat"
	"github.com/richxcame/ride-hailing/internal/corporate"
	"github.com/richxcame/ride-hailing/internal/currency"
	"github.com/richxcame/ride-hailing/internal/delivery"
	"github.com/richxcame/ride-hailing/internal/demandforecast"
	"github.com/richxcame/ride-hailing/internal/disputes"
	"github.com/richxcame/ride-hailing/internal/earnings"
	"github.com/richxcame/ride-hailing/internal/experiments"
	"github.com/richxcame/ride-hailing/internal/family"
	"github.com/richxcame/ride-hailing/internal/favorites"
	"github.com/richxcame/ride-hailing/internal/fraud"
	"github.com/richxcame/ride-hailing/internal/gamification"
	"github.com/richxcame/ride-hailing/internal/geography"
	"github.com/richxcame/ride-hailing/internal/giftcards"
	"github.com/richxcame/ride-hailing/internal/loyalty"
	"github.com/richxcame/ride-hailing/internal/negotiation"
	"github.com/richxcame/ride-hailing/internal/onboarding"
	"github.com/richxcame/ride-hailing/internal/paymentmethods"
	"github.com/richxcame/ride-hailing/internal/paymentsplit"
	"github.com/richxcame/ride-hailing/internal/pool"
	"github.com/richxcame/ride-hailing/internal/preferences"
	"github.com/richxcame/ride-hailing/internal/pricing"
	"github.com/richxcame/ride-hailing/internal/ratings"
	"github.com/richxcame/ride-hailing/internal/recording"
	"github.com/richxcame/ride-hailing/internal/ridehistory"
	"github.com/richxcame/ride-hailing/internal/rides"
	"github.com/richxcame/ride-hailing/internal/safety"
	"github.com/richxcame/ride-hailing/internal/subscriptions"
	"github.com/richxcame/ride-hailing/internal/support"
	"github.com/richxcame/ride-hailing/internal/tips"
	"github.com/richxcame/ride-hailing/internal/twofa"
	"github.com/richxcame/ride-hailing/internal/vehicle"
	"github.com/richxcame/ride-hailing/internal/waittime"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/errors"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/swagger"
	"github.com/richxcame/ride-hailing/pkg/tracing"
	ws "github.com/richxcame/ride-hailing/pkg/websocket"
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
	jwtSecret := getEnv("JWT_SECRET", "")
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

	// Initialize WebSocket hub
	wsHub := ws.NewHub()
	go wsHub.Run()

	// Initialize repositories
	ridesRepo := rides.NewRepository(db)
	favoritesRepo := favorites.NewRepository(db)
	cancellationRepo := cancellation.NewRepository(db)
	supportRepo := support.NewRepository(db)
	disputesRepo := disputes.NewRepository(db)
	tipsRepo := tips.NewRepository(db)
	ratingsRepo := ratings.NewRepository(db)
	earningsRepo := earnings.NewRepository(db)
	vehicleRepo := vehicle.NewRepository(db)
	paymentmethodsRepo := paymentmethods.NewRepository(db)
	ridehistoryRepo := ridehistory.NewRepository(db)
	familyRepo := family.NewRepository(db)
	giftcardsRepo := giftcards.NewRepository(db)
	subscriptionsRepo := subscriptions.NewRepository(db)
	preferencesRepo := preferences.NewRepository(db)
	waittimeRepo := waittime.NewRepository(db)
	chatRepo := chat.NewRepository(db)
	corporateRepo := corporate.NewRepository(db)
	twofaRepo := twofa.NewRepository(db)
	loyaltyRepo := loyalty.NewRepository(db)
	poolRepo := pool.NewRepository(db)
	deliveryRepo := delivery.NewRepository(db)
	recordingRepo := recording.NewRepository(db)
	onboardingRepo := onboarding.NewRepository(db)
	demandforecastRepo := demandforecast.NewRepository(db)
	experimentsRepo := experiments.NewRepository(db)
	fraudRepo := fraud.NewRepository(db)
	gamificationRepo := gamification.NewRepository(db)
	paymentsplitRepo := paymentsplit.NewRepository(db)
	geographyRepo := geography.NewRepository(db)
	currencyRepo := currency.NewRepository(db)
	pricingRepo := pricing.NewRepository(db)
	negotiationRepo := negotiation.NewRepository(db)
	safetyRepo := safety.NewRepository(db)

	// Initialize services
	ridesService := rides.NewService(ridesRepo, promosServiceURL, nil) // CircuitBreaker is nil-safe
	favoritesService := favorites.NewService(favoritesRepo)
	cancellationService := cancellation.NewService(cancellationRepo, db)
	supportService := support.NewService(supportRepo)
	disputesService := disputes.NewService(disputesRepo)
	tipsService := tips.NewService(tipsRepo)
	ratingsService := ratings.NewService(ratingsRepo)
	earningsService := earnings.NewService(earningsRepo)
	vehicleService := vehicle.NewService(vehicleRepo)
	paymentmethodsService := paymentmethods.NewService(paymentmethodsRepo)
	ridehistoryService := ridehistory.NewService(ridehistoryRepo)
	familyService := family.NewService(familyRepo)
	giftcardsService := giftcards.NewService(giftcardsRepo)
	subscriptionsService := subscriptions.NewService(subscriptionsRepo, &stubPaymentProcessor{})
	preferencesService := preferences.NewService(preferencesRepo)
	waittimeService := waittime.NewService(waittimeRepo)
	chatService := chat.NewService(chatRepo, wsHub)
	corporateService := corporate.NewService(corporateRepo)
	twofaService := twofa.NewService(twofaRepo, &stubSMSSender{}, nil, getEnv("APP_NAME", "RideHailing")) // Redis is nil-safe (OTP stored in DB)
	loyaltyService := loyalty.NewService(loyaltyRepo)
	poolService := pool.NewService(poolRepo, &stubMapsService{}, pool.DefaultServiceConfig())
	deliveryService := delivery.NewService(deliveryRepo)
	recordingService := recording.NewService(recordingRepo, &stubStorage{}, recording.Config{})
	onboardingService := onboarding.NewService(onboardingRepo, &stubDocumentService{}, nil) // NotificationService is nil-safe
	demandforecastService := demandforecast.NewService(demandforecastRepo, nil, nil, nil) // All deps are nil-safe
	experimentsService := experiments.NewService(experimentsRepo)
	fraudService := fraud.NewService(fraudRepo)
	gamificationService := gamification.NewService(gamificationRepo)
	paymentsplitService := paymentsplit.NewService(paymentsplitRepo, &stubPaymentService{}, &stubSplitNotificationService{})
	geographyService := geography.NewService(geographyRepo)
	currencyService := currency.NewService(currencyRepo, getEnv("BASE_CURRENCY", "USD"))
	pricingService := pricing.NewService(pricingRepo, geographyService, currencyService)
	negotiationService := negotiation.NewService(negotiationRepo, pricingService, geographyService)
	safetyService := safety.NewService(safetyRepo, safety.Config{
		EmergencyNumber: getEnv("EMERGENCY_NUMBER", "112"),
	})

	// Initialize handlers
	ridesHandler := rides.NewHandler(ridesService)
	favoritesHandler := favorites.NewHandler(favoritesService)
	cancellationHandler := cancellation.NewHandler(cancellationService)
	supportHandler := support.NewHandler(supportService)
	disputesHandler := disputes.NewHandler(disputesService)
	tipsHandler := tips.NewHandler(tipsService)
	ratingsHandler := ratings.NewHandler(ratingsService)
	earningsHandler := earnings.NewHandler(earningsService)
	vehicleHandler := vehicle.NewHandler(vehicleService)
	paymentmethodsHandler := paymentmethods.NewHandler(paymentmethodsService)
	ridehistoryHandler := ridehistory.NewHandler(ridehistoryService)
	familyHandler := family.NewHandler(familyService)
	giftcardsHandler := giftcards.NewHandler(giftcardsService)
	subscriptionsHandler := subscriptions.NewHandler(subscriptionsService)
	preferencesHandler := preferences.NewHandler(preferencesService)
	waittimeHandler := waittime.NewHandler(waittimeService)
	chatHandler := chat.NewHandler(chatService)
	corporateHandler := corporate.NewHandler(corporateService)
	twofaHandler := twofa.NewHandler(twofaService)
	loyaltyHandler := loyalty.NewHandler(loyaltyService)
	poolHandler := pool.NewHandler(poolService)
	deliveryHandler := delivery.NewHandler(deliveryService)
	recordingHandler := recording.NewHandler(recordingService)
	onboardingHandler := onboarding.NewHandler(onboardingService)
	demandforecastHandler := demandforecast.NewHandler(demandforecastService)
	experimentsHandler := experiments.NewHandler(experimentsService)
	fraudHandler := fraud.NewHandler(fraudService)
	gamificationHandler := gamification.NewHandler(gamificationService)
	paymentsplitHandler := paymentsplit.NewHandler(paymentsplitService)
	geographyHandler := geography.NewHandler(geographyService)
	currencyHandler := currency.NewHandler(currencyService)
	pricingHandler := pricing.NewHandler(pricingService)
	negotiationHandler := negotiation.NewHandler(negotiationService)
	safetyHandler := safety.NewHandler(safetyService)

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

	// Register feature routes
	cancellationHandler.RegisterRoutes(router, jwtProvider)
	supportHandler.RegisterRoutes(router, jwtProvider)
	disputesHandler.RegisterRoutes(router, jwtProvider)
	tipsHandler.RegisterRoutes(router, jwtProvider)
	ratingsHandler.RegisterRoutes(router, jwtProvider)
	earningsHandler.RegisterRoutes(router, jwtProvider)
	vehicleHandler.RegisterRoutes(router, jwtProvider)
	paymentmethodsHandler.RegisterRoutes(router, jwtProvider)
	ridehistoryHandler.RegisterRoutes(router, jwtProvider)
	familyHandler.RegisterRoutes(router, jwtProvider)
	giftcardsHandler.RegisterRoutes(router, jwtProvider)
	subscriptionsHandler.RegisterRoutes(router, jwtProvider)
	preferencesHandler.RegisterRoutes(router, jwtProvider)
	waittimeHandler.RegisterRoutes(router, jwtProvider)
	chatHandler.RegisterRoutes(router, jwtProvider)
	corporateHandler.RegisterRoutes(router, jwtProvider)
	twofaHandler.RegisterRoutes(router, jwtProvider)
	loyaltyHandler.RegisterRoutes(router, jwtProvider)
	poolHandler.RegisterRoutes(router, jwtProvider)
	deliveryHandler.RegisterRoutes(router, jwtProvider)
	recordingHandler.RegisterRoutes(router, jwtProvider)
	onboardingHandler.RegisterRoutes(router, jwtProvider)
	demandforecastHandler.RegisterRoutes(router, jwtProvider)
	experimentsHandler.RegisterRoutes(router, jwtProvider)
	fraudHandler.RegisterRoutes(router, jwtProvider)
	gamificationHandler.RegisterRoutes(router, jwtProvider)
	paymentsplitHandler.RegisterRoutes(router, jwtProvider)

	// Register RouterGroup-based routes
	apiGroup := router.Group("/api/v1")
	apiGroup.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	geographyHandler.RegisterRoutes(apiGroup)
	currencyHandler.RegisterRoutes(apiGroup)
	pricingHandler.RegisterRoutes(apiGroup)
	negotiationHandler.RegisterRoutes(apiGroup)
	safetyHandler.RegisterRoutes(apiGroup)

	// Create HTTP server with timeouts
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Mobile API service starting", zap.String("port", port))
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
