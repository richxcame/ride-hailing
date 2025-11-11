package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

func RequestTimeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		panicChan := make(chan interface{}, 1)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					panicChan <- r
				}
				close(done)
			}()
			c.Next()
		}()

		select {
		case <-done:
			// Request completed successfully
			return
		case p := <-panicChan:
			// Handler panicked
			if !c.Writer.Written() {
				c.Abort()
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
				})
			}
			logger.WithContext(ctx).Error("Handler panic",
				zap.Any("panic", p),
				zap.String("path", c.Request.URL.Path),
			)
		case <-ctx.Done():
			// Timeout expired
			if ctx.Err() == context.DeadlineExceeded {
				if !c.Writer.Written() {
					c.Abort()
					c.JSON(http.StatusGatewayTimeout, gin.H{
						"error":   "Request timeout",
						"message": "The request took too long to process",
					})

					logger.WithContext(ctx).Warn("Request timeout",
						zap.String("path", c.Request.URL.Path),
						zap.String("method", c.Request.Method),
						zap.Duration("timeout", timeout),
					)
				}
			}
		}
	}
}