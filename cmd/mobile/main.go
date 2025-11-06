package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/richxcame/ride-hailing/internal/favorites"
	"github.com/richxcame/ride-hailing/internal/rides"
	"github.com/richxcame/ride-hailing/pkg/middleware"
)

func main() {
	// Load environment variables
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "ride_hailing")
	jwtSecret := getEnv("JWT_SECRET", "your-secret-key")
	port := getEnv("PORT", "8087")

	// Connect to PostgreSQL
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	ctx := context.Background()
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
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to PostgreSQL database")

	// Get Promos service URL from environment
	promosServiceURL := getEnv("PROMOS_SERVICE_URL", "http://localhost:8089")
	log.Printf("Promos service URL configured: %s", promosServiceURL)

	// Initialize repositories
	ridesRepo := rides.NewRepository(db)
	favoritesRepo := favorites.NewRepository(db)

	// Initialize services
	ridesService := rides.NewService(ridesRepo, promosServiceURL)
	favoritesService := favorites.NewService(favoritesRepo)

	// Initialize handlers
	ridesHandler := rides.NewHandler(ridesService)
	favoritesHandler := favorites.NewHandler(favoritesService)

	// Set up Gin router
	router := gin.Default()
	router.Use(middleware.CorrelationID())

	// Health check and metrics (no auth required)
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "mobile-api"})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API routes with authentication
	api := router.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(jwtSecret))
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
	log.Printf("Mobile API service starting on port %s", port)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
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
