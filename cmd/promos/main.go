package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/richxcame/ride-hailing/internal/promos"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
)

func main() {
	// Load configuration
	cfg, err := config.Load("promos")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	port := cfg.Server.Port

	rootCtx, cancelKeys := context.WithCancel(context.Background())
	defer cancelKeys()

	jwtProvider, err := jwtkeys.NewManagerFromConfig(rootCtx, cfg.JWT, true)
	if err != nil {
		log.Fatalf("Failed to initialize JWT key manager: %v", err)
	}
	jwtProvider.StartAutoRefresh(rootCtx, time.Duration(cfg.JWT.RefreshMinutes)*time.Minute)

	// Connect to PostgreSQL
	ctx := rootCtx
	dsn := cfg.Database.DSN()
	dbConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("Failed to parse database config: %v", err)
	}

	db, err := pgxpool.NewWithConfig(ctx, dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to PostgreSQL database")

	// Create repository, service and handler
	repo := promos.NewRepository(db)
	service := promos.NewService(repo)
	handler := promos.NewHandler(service)

	// Set up Gin router
	router := gin.Default()
	router.Use(middleware.CorrelationID())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.SanitizeRequest())

	// Health check and metrics
	router.GET("/healthz", handler.HealthCheck)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API routes
	api := router.Group("/api/v1")
	{
		// Public endpoints
		api.GET("/ride-types", handler.GetRideTypes)
		api.POST("/ride-types/calculate-fare", handler.CalculateFare)

		// Authenticated endpoints
		authenticated := api.Group("")
		authenticated.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
		{
			// Promo codes
			authenticated.POST("/promo-codes/validate", handler.ValidatePromoCode)

			// Referral codes
			authenticated.GET("/referrals/my-code", handler.GetMyReferralCode)
			authenticated.POST("/referrals/apply", handler.ApplyReferralCode)
		}

		// Admin endpoints
		admin := api.Group("/admin")
		admin.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
		admin.Use(middleware.RequireAdmin())
		{
			admin.POST("/promo-codes", handler.CreatePromoCode)
		}
	}

	// Start server
	addr := ":" + port
	log.Printf("Promos service starting on port %s", port)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
