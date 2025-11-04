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
	"github.com/richxcame/ride-hailing/internal/payments"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/database"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
)

const (
	serviceName = "payments-service"
	version     = "1.0.0"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger.Init(cfg.Server.Environment)
	log := logger.Get()

	log.Info("Starting payments service", "version", version)

	// Initialize database
	db, err := database.NewPostgresPool(&cfg.Database)
	if err != nil {
		log.Fatal("Failed to connect to database", "error", err)
	}
	defer db.Close()

	log.Info("Connected to database")

	// Get Stripe API key from environment
	stripeAPIKey := os.Getenv("STRIPE_API_KEY")
	if stripeAPIKey == "" {
		log.Warn("STRIPE_API_KEY not set, payment processing will be limited")
		stripeAPIKey = "sk_test_dummy" // Dummy key for development
	}

	// Initialize payment service
	paymentRepo := payments.NewRepository(db)
	paymentService := payments.NewService(paymentRepo, stripeAPIKey)
	paymentHandler := payments.NewHandler(paymentService)

	// Setup Gin router
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(middleware.LoggingMiddleware())
	router.Use(middleware.RecoveryMiddleware())
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.MetricsMiddleware())

	// Health check
	router.GET("/healthz", func(c *gin.Context) {
		common.HealthCheckResponse(c, serviceName, version)
	})

	// Metrics endpoint
	router.GET("/metrics", middleware.PrometheusHandler())

	// Register payment routes
	paymentHandler.RegisterRoutes(router, cfg.JWT.Secret)

	// Setup HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Info("Server starting", "port", cfg.Server.Port, "environment", cfg.Server.Environment)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start", "error", err)
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
		log.Fatal("Server forced to shutdown", "error", err)
	}

	log.Info("Server stopped")
}
