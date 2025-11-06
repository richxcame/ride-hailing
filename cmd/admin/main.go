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
	"github.com/richxcame/ride-hailing/internal/admin"
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
	port := getEnv("PORT", "8088")

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

	// Initialize repository, service, and handler
	repo := admin.NewRepository(db)
	service := admin.NewService(repo)
	handler := admin.NewHandler(service)

	// Set up Gin router
	router := gin.Default()
	router.Use(middleware.CorrelationID())

	// Health check and metrics (no auth required)
	router.GET("/healthz", handler.HealthCheck)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Admin API routes - all require authentication + admin role
	api := router.Group("/api/v1/admin")
	api.Use(middleware.AuthMiddleware(jwtSecret))
	api.Use(middleware.RequireAdmin())
	{
		// Dashboard
		api.GET("/dashboard", handler.GetDashboard)

		// User management
		users := api.Group("/users")
		{
			users.GET("", handler.GetAllUsers)
			users.GET("/:id", handler.GetUser)
			users.POST("/:id/suspend", handler.SuspendUser)
			users.POST("/:id/activate", handler.ActivateUser)
		}

		// Driver management
		drivers := api.Group("/drivers")
		{
			drivers.GET("/pending", handler.GetPendingDrivers)
			drivers.POST("/:id/approve", handler.ApproveDriver)
			drivers.POST("/:id/reject", handler.RejectDriver)
		}

		// Ride monitoring
		rides := api.Group("/rides")
		{
			rides.GET("/recent", handler.GetRecentRides)
			rides.GET("/stats", handler.GetRideStats)
		}
	}

	// Start server
	addr := ":" + port
	log.Printf("Admin service starting on port %s", port)
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
