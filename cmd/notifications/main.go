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
	"github.com/richxcame/ride-hailing/internal/notifications"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/database"
	"github.com/richxcame/ride-hailing/pkg/errors"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/tracing"
	"github.com/richxcame/ride-hailing/pkg/resilience"
	"go.uber.org/zap"
)

const (
	serviceName = "notifications-service"
	version     = "1.0.0"
)

func main() {
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
	log.Info("Starting notifications service", zap.String("version", version))

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
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.Close(db)

	log.Info("Connected to database")

	// Initialize notification clients
	var firebaseBreaker, twilioBreaker, smtpBreaker *resilience.CircuitBreaker
	if cfg.Resilience.CircuitBreaker.Enabled {
		firebaseBreaker = buildBreaker(serviceName+"-firebase", cfg.Resilience.CircuitBreaker.SettingsFor("firebase-fcm"))
		twilioBreaker = buildBreaker(serviceName+"-twilio", cfg.Resilience.CircuitBreaker.SettingsFor("twilio-sms"))
		smtpBreaker = buildBreaker(serviceName+"-smtp", cfg.Resilience.CircuitBreaker.SettingsFor("smtp-email"))
	}

	// Firebase Client
	var firebaseClient *notifications.FirebaseClient
	switch {
	case cfg.Firebase.CredentialsJSON != "":
		firebaseClient, err = notifications.NewFirebaseClientFromJSON([]byte(cfg.Firebase.CredentialsJSON))
	case cfg.Firebase.CredentialsPath != "":
		firebaseClient, err = notifications.NewFirebaseClient(cfg.Firebase.CredentialsPath)
	}
	if err != nil {
		log.Warn("Failed to initialize Firebase client", zap.Error(err))
	} else if firebaseClient != nil {
		log.Info("Firebase client initialized")
	} else {
		log.Warn("Firebase credentials not provided, push notifications disabled")
	}

	// Twilio Client
	var twilioClient *notifications.TwilioClient
	twilioAccountSid := cfg.Notifications.TwilioAccountSID
	twilioAuthToken := cfg.Notifications.TwilioAuthToken
	twilioFromNumber := cfg.Notifications.TwilioFromNumber
	if twilioAccountSid != "" && twilioAuthToken != "" && twilioFromNumber != "" {
		twilioClient = notifications.NewTwilioClient(twilioAccountSid, twilioAuthToken, twilioFromNumber)
		log.Info("Twilio client initialized")
	} else {
		log.Warn("Twilio credentials not set, SMS notifications disabled")
	}

	// Email Client
	var emailClient *notifications.EmailClient
	smtpHost := cfg.Notifications.SMTPHost
	smtpPort := cfg.Notifications.SMTPPort
	smtpUsername := cfg.Notifications.SMTPUsername
	smtpPassword := cfg.Notifications.SMTPPassword
	smtpFromEmail := cfg.Notifications.SMTPFromEmail
	smtpFromName := cfg.Notifications.SMTPFromName

	if smtpHost != "" && smtpPort != "" {
		emailClient = notifications.NewEmailClient(
			smtpHost,
			smtpPort,
			smtpUsername,
			smtpPassword,
			smtpFromEmail,
			smtpFromName,
		)
		log.Info("Email client initialized")
	} else {
		log.Warn("SMTP credentials not set, email notifications disabled")
	}

	// Initialize notification service
	notificationRepo := notifications.NewRepository(db)
	notificationService := notifications.NewServiceWithClients(notificationRepo, firebaseClient, twilioClient, emailClient)
	notificationService.SetCircuitBreakers(firebaseBreaker, twilioBreaker, smtpBreaker)
	notificationHandler := notifications.NewHandler(notificationService)

	// Start background worker for processing scheduled notifications
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			err := notificationService.ProcessPendingNotifications(context.Background())
			if err != nil {
				log.Error("Failed to process pending notifications", zap.Error(err))
			}
		}
	}()

	// Setup Gin router
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(middleware.RecoveryWithSentry()) // Custom recovery with Sentry
	router.Use(middleware.SentryMiddleware())   // Sentry integration
	router.Use(middleware.CorrelationID())
	router.Use(middleware.RequestTimeout(&cfg.Timeout))
	router.Use(middleware.RequestLogger(serviceName))
	router.Use(middleware.CORS())
	router.Use(middleware.SecurityHeaders())
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
	healthChecks["database"] = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return db.Ping(ctx)
	}

	router.GET("/health/ready", common.ReadinessProbe(serviceName, version, healthChecks))

	// Metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Register notification routes
	jwtProvider, err := jwtkeys.NewManagerFromConfig(rootCtx, cfg.JWT, true)
	if err != nil {
		log.Fatal("Failed to initialize JWT key manager", zap.Error(err))
	}
	jwtProvider.StartAutoRefresh(rootCtx, time.Duration(cfg.JWT.RefreshMinutes)*time.Minute)

	notificationHandler.RegisterRoutes(router, jwtProvider)

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
			log.Fatal("Server failed to start", zap.Error(err))
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
		log.Fatal("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Server stopped")
}

func buildBreaker(name string, cbCfg config.CircuitBreakerSettings) *resilience.CircuitBreaker {
	return resilience.NewCircuitBreaker(
		resilience.BuildSettings(name, cbCfg.IntervalSeconds, cbCfg.TimeoutSeconds, cbCfg.FailureThreshold, cbCfg.SuccessThreshold),
		nil,
	)
}
