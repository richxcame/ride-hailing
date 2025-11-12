package middleware

import (
	"net/http"

	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// RequestTimeout applies a timeout to incoming requests
// Uses route-specific timeouts from config when available, otherwise uses default timeout
func RequestTimeout(timeoutConfig *config.TimeoutConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		routePath := c.FullPath()
		if routePath == "" {
			routePath = c.Request.URL.Path
		}
		timeoutDuration := timeoutConfig.TimeoutForRoute(c.Request.Method, routePath)

		timeoutResponse := func(c *gin.Context) {
			if !c.Writer.Written() {
				c.Abort()
				c.JSON(http.StatusGatewayTimeout, gin.H{
					"error":   "Request timeout",
					"message": "The request took too long to process",
				})

				logger.WithContext(c.Request.Context()).Warn("Request timeout",
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.Duration("timeout", timeoutDuration),
				)
			}
		}

		timeoutMiddleware := timeout.New(
			timeout.WithTimeout(timeoutDuration),
			timeout.WithResponse(timeoutResponse),
		)

		timeoutMiddleware(c)
	}
}
