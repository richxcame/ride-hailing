package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/logger"
)

const (
	// CorrelationIDHeader is the header name for correlation ID
	CorrelationIDHeader = "X-Request-ID"
	// CorrelationIDKey is the context key for correlation ID
	CorrelationIDKey = "correlation_id"
)

// CorrelationID middleware generates or extracts correlation ID for request tracing
func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get correlation ID from header
		correlationID := strings.TrimSpace(c.GetHeader(CorrelationIDHeader))

		// Validate provided correlation ID
		if correlationID != "" {
			if _, err := uuid.Parse(correlationID); err != nil {
				correlationID = ""
			}
		}

		// If not provided, generate a new UUID
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Store in context for use by handlers
		c.Set(CorrelationIDKey, correlationID)

		// Store in request context for downstream usage
		ctx := logger.ContextWithCorrelationID(c.Request.Context(), correlationID)
		c.Request = c.Request.WithContext(ctx)

		// Add to response headers
		c.Writer.Header().Set(CorrelationIDHeader, correlationID)

		c.Next()
	}
}

// GetCorrelationID extracts correlation ID from gin context
func GetCorrelationID(c *gin.Context) string {
	if id, exists := c.Get(CorrelationIDKey); exists {
		if correlationID, ok := id.(string); ok {
			return correlationID
		}
	}
	return logger.CorrelationIDFromContext(c.Request.Context())
}
