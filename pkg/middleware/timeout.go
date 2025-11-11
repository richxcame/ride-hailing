package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// RequestTimeout creates a middleware that sets a timeout on the request context
// If the timeout expires, it returns a 504 Gateway Timeout response
func RequestTimeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			// Request completed before timeout
		case <-ctx.Done():
			// Timeout expired
			if ctx.Err() == context.DeadlineExceeded {
				if !c.Writer.Written() {
					c.Abort()
					c.JSON(http.StatusGatewayTimeout, gin.H{
						"error":   "Request timeout",
						"message": "The request took too long to process",
					})

					logger.WithContext(c.Request.Context()).Warn("Request timeout",
						zap.String("path", c.Request.URL.Path),
						zap.String("method", c.Request.Method),
						zap.Duration("timeout", timeout),
					)
				}
			}
		}
	}
}

