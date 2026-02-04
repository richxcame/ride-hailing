package middleware

import (
	"crypto/subtle"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// InternalAPIKey returns a Gin middleware that validates requests using a shared
// secret passed in the X-Internal-API-Key header. It uses constant-time
// comparison to prevent timing attacks.
func InternalAPIKey() gin.HandlerFunc {
	expected := os.Getenv("INTERNAL_API_KEY")

	return func(c *gin.Context) {
		if expected == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "internal API key not configured",
			})
			return
		}

		provided := c.GetHeader("X-Internal-API-Key")
		if subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid internal API key",
			})
			return
		}

		c.Next()
	}
}
