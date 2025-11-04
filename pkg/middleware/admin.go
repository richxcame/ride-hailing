package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// RequireAdmin middleware ensures only admin users can access the endpoint
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Check if user is admin
		if role != models.RoleAdmin && role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}
