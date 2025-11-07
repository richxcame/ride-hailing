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

	ridesdocs "github.com/richxcame/ride-hailing/docs/rides"

	"github.com/richxcame/ride-hailing/internal/pricing"
	"github.com/richxcame/ride-hailing/internal/rides"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/database"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/ratelimit"
	redisclient "github.com/richxcame/ride-hailing/pkg/redis"
	"go.uber.org/zap"
)

// @title           Rides Service API
// @version         1.0
// @description     API for managing rider and driver interactions in the rides domain.
// @BasePath        /api/v1
// @schemes         http https
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     Provide a valid JWT token using the format `Bearer <token>`.

const (
	serviceName = "rides-service"
	version     = "1.0.0"
)

const swaggerPage = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <title>Rides Service API Docs</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            window.ui = SwaggerUIBundle({
                url: '%s',
                dom_id: '#swagger-ui',
                presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
                layout: 'BaseLayout',
            });
        };
    </script>
</body>
</html>`

func main() {
	cfg, err := config.Load(serviceName)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	if err := logger.Init(cfg.Server.Environment); err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	swaggerSpec, err := ridesdocs.Files.ReadFile("swagger.yaml")
	if err != nil {
		logger.Fatal("Failed to load Swagger spec", zap.Error(err))
	}

	logger.Info("Starting rides service",
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

	var (
		redisClient *redisclient.Client
		limiter     *ratelimit.Limiter
	)

	if cfg.RateLimit.Enabled {
		redisClient, err = redisclient.NewRedisClient(&cfg.Redis)
		if err != nil {
			logger.Fatal("Failed to initialize redis for rate limiting", zap.Error(err))
		}

		limiter = ratelimit.NewLimiter(redisClient.Client, cfg.RateLimit)
		logger.Info("Rate limiting enabled",
			zap.Int("default_limit", cfg.RateLimit.DefaultLimit),
			zap.Int("default_burst", cfg.RateLimit.DefaultBurst),
			zap.Duration("window", cfg.RateLimit.Window()),
		)

		defer func() {
			if err := redisClient.Close(); err != nil {
				logger.Warn("Failed to close redis client", zap.Error(err))
			}
		}()
	}

	// Get Promos service URL from environment
	promosServiceURL := os.Getenv("PROMOS_SERVICE_URL")
	if promosServiceURL == "" {
		promosServiceURL = "http://localhost:8089" // Default for development
	}
	logger.Info("Promos service URL configured", zap.String("url", promosServiceURL))

	repo := rides.NewRepository(db)
	service := rides.NewService(repo, promosServiceURL)

	// Initialize dynamic surge pricing calculator
	surgeCalculator := pricing.NewSurgeCalculator(db)
	service.SetSurgeCalculator(surgeCalculator)
	logger.Info("Dynamic surge pricing enabled")

	handler := rides.NewHandler(service)

	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.CorrelationID())
	router.Use(middleware.RequestLogger(serviceName))
	router.Use(middleware.CORS())
	router.Use(middleware.Metrics(serviceName))

	router.GET("/healthz", common.HealthCheck(serviceName, version))
	router.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": serviceName,
			"version": version,
		})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	router.GET("/swagger/doc.yaml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", swaggerSpec)
	})
	router.GET("/swagger", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fmt.Sprintf(swaggerPage, "/swagger/doc.yaml")))
	})
	router.GET("/swagger/*path", func(c *gin.Context) {
		if c.Param("path") == "/doc.yaml" {
			c.Data(http.StatusOK, "application/yaml; charset=utf-8", swaggerSpec)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fmt.Sprintf(swaggerPage, "/swagger/doc.yaml")))
	})

	handler.RegisterRoutes(router, cfg.JWT.Secret, limiter, cfg.RateLimit)

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
