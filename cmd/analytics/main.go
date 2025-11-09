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
	"github.com/richxcame/ride-hailing/internal/analytics"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/database"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"go.uber.org/zap"
)

const (
	serviceName = "analytics-service"
	version     = "1.0.0"
)

func main() {
	cfg, err := config.Load(serviceName)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	rootCtx, cancelKeys := context.WithCancel(context.Background())
	defer cancelKeys()

	if err := logger.Init(cfg.Server.Environment); err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	logger.Info("Starting analytics service",
		zap.String("service", serviceName),
		zap.String("version", version),
		zap.String("environment", cfg.Server.Environment),
	)

	db, err := database.NewPostgresPool(&cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.Close(db)
	logger.Info("Connected to database")

	repo := analytics.NewRepository(db)
	service := analytics.NewService(repo)
	handler := analytics.NewHandler(service)

	jwtProvider, err := jwtkeys.NewManagerFromConfig(rootCtx, cfg.JWT, true)
	if err != nil {
		logger.Fatal("Failed to initialize JWT key manager", zap.Error(err))
	}
	jwtProvider.StartAutoRefresh(rootCtx, time.Duration(cfg.JWT.RefreshMinutes)*time.Minute)

	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.CorrelationID())
	router.Use(middleware.RequestLogger(serviceName))
	router.Use(middleware.CORS())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.SanitizeRequest())
	router.Use(middleware.Metrics(serviceName))

	// Public endpoints
	router.GET("/healthz", handler.HealthCheck)
	router.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": serviceName,
			"version": version,
		})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Admin-only analytics endpoints
	api := router.Group("/api/v1/analytics")
	api.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	api.Use(middleware.RequireAdmin())
	{
		api.GET("/dashboard", handler.GetDashboardMetrics)
		api.GET("/revenue", handler.GetRevenueMetrics)
		api.GET("/promo-codes", handler.GetPromoCodePerformance)
		api.GET("/ride-types", handler.GetRideTypeStats)
		api.GET("/referrals", handler.GetReferralMetrics)
		api.GET("/top-drivers", handler.GetTopDrivers)
		api.GET("/heat-map", handler.GetDemandHeatMap)
		api.GET("/financial-report", handler.GetFinancialReport)
		api.GET("/demand-zones", handler.GetDemandZones)
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	go func() {
		logger.Info("Server starting", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server stopped")
}
