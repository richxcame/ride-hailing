package middleware

import (
	"net/http"

	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

var (
	timeoutCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_timeouts_total",
			Help: "Total number of HTTP request timeouts by method and route",
		},
		[]string{"method", "route"},
	)
)

// RequestTimeout applies a timeout to incoming requests
// Uses route-specific timeouts from config when available, otherwise uses default timeout
func RequestTimeout(timeoutConfig *config.TimeoutConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add panic recovery specifically for timeout middleware
		defer func() {
			if r := recover(); r != nil {
				logger.WithContext(c.Request.Context()).Error("Panic in timeout middleware",
					zap.Any("panic", r),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)
				if !c.Writer.Written() {
					c.AbortWithStatus(http.StatusInternalServerError)
				}
			}
		}()

		routePath := c.FullPath()
		if routePath == "" {
			routePath = c.Request.URL.Path
		}
		timeoutDuration := timeoutConfig.TimeoutForRoute(c.Request.Method, routePath)

		timeoutResponse := func(c *gin.Context) {
			// Set timeout header for debugging
			c.Writer.Header().Set("X-Timeout", "true")

			// Preserve correlation ID from the original request context
			if correlationID := GetCorrelationID(c); correlationID != "" {
				c.Writer.Header().Set(CorrelationIDHeader, correlationID)
			}

			if !c.Writer.Written() {
				c.Abort()
				c.JSON(http.StatusGatewayTimeout, gin.H{
					"error":   "Request timeout",
					"message": "The request took too long to process",
				})

				// Log timeout event with correlation ID
				logger.WithContext(c.Request.Context()).Warn("Request timeout",
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.Duration("timeout", timeoutDuration),
				)

				// Increment Prometheus counter
				timeoutCounter.WithLabelValues(c.Request.Method, routePath).Inc()
			}
		}

		timeoutMiddleware := timeout.New(
			timeout.WithTimeout(timeoutDuration),
			timeout.WithResponse(timeoutResponse),
		)

		timeoutMiddleware(c)
	}
}
