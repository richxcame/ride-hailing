package middleware

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS middleware handles Cross-Origin Resource Sharing.
// Allowed origins are read from the CORS_ORIGINS environment variable
// (comma-separated). Falls back to http://localhost:3000 for development.
func CORS() gin.HandlerFunc {
	originsStr := os.Getenv("CORS_ORIGINS")
	if originsStr == "" {
		originsStr = "http://localhost:3000"
	}
	allowedOrigins := make(map[string]bool)
	for _, o := range strings.Split(originsStr, ",") {
		allowedOrigins[strings.TrimSpace(o)] = true
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if allowedOrigins[origin] || allowedOrigins["*"] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Idempotency-Key, X-Request-ID, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
