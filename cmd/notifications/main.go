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
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
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

	rootCtx, cancelKeys := context.WithCancel(context.Background())
	defer cancelKeys()

	// Initialize logger
	if err := logger.Init(cfg.Server.Environment); err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	log := logger.Get()
	log.Info("Starting notifications service", zap.String("version", version))

	// Initialize database
	db, err := database.NewPostgresPool(&cfg.Database)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.Close(db)

	log.Info("Connected to database")

	// Initialize notification clients

	// Firebase Client
	var firebaseClient *notifications.FirebaseClient
	firebaseCredPath := os.Getenv("FIREBASE_CREDENTIALS_PATH")
	if firebaseCredPath != "" {
		firebaseClient, err = notifications.NewFirebaseClient(firebaseCredPath)
		if err != nil {
			log.Warn("Failed to initialize Firebase client", zap.Error(err))
		} else {
			log.Info("Firebase client initialized")
		}
	} else {
		log.Warn("FIREBASE_CREDENTIALS_PATH not set, push notifications disabled")
	}

	// Twilio Client
	var twilioClient *notifications.TwilioClient
	twilioAccountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	twilioAuthToken := os.Getenv("TWILIO_AUTH_TOKEN")
	twilioFromNumber := os.Getenv("TWILIO_FROM_NUMBER")
	if twilioAccountSid != "" && twilioAuthToken != "" && twilioFromNumber != "" {
		twilioClient = notifications.NewTwilioClient(twilioAccountSid, twilioAuthToken, twilioFromNumber)
		log.Info("Twilio client initialized")
	} else {
		log.Warn("Twilio credentials not set, SMS notifications disabled")
	}

	// Email Client
	var emailClient *notifications.EmailClient
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUsername := os.Getenv("SMTP_USERNAME")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	smtpFromEmail := os.Getenv("SMTP_FROM_EMAIL")
	smtpFromName := os.Getenv("SMTP_FROM_NAME")

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
	router.Use(middleware.Recovery())
	router.Use(middleware.CorrelationID())
	router.Use(middleware.RequestLogger(serviceName))
	router.Use(middleware.CORS())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.SanitizeRequest())
	router.Use(middleware.Metrics(serviceName))

	// Health check
	router.GET("/healthz", common.HealthCheck(serviceName, version))

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
