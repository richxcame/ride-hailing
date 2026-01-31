package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// Claims represents JWT claims
type Claims struct {
	UserID uuid.UUID       `json:"user_id"`
	Email  string          `json:"email"`
	Role   models.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// AuthMiddleware validates JWT tokens with a static secret (deprecated). Prefer
// AuthMiddlewareWithProvider to enable key rotation support.
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return AuthMiddlewareWithProvider(jwtkeys.NewStaticProvider(jwtSecret))
}

// AuthMiddlewareWithProvider validates JWT tokens using the supplied key provider.
func AuthMiddlewareWithProvider(provider jwtkeys.KeyProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// Extract token from "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				common.ErrorResponse(c, http.StatusUnauthorized, "invalid authorization header format")
				c.Abort()
				return
			}
			tokenString = parts[1]
		} else if t := c.Query("token"); t != "" {
			// Allow token via query param for WebSocket connections
			tokenString = t
		} else {
			common.ErrorResponse(c, http.StatusUnauthorized, "authorization required")
			c.Abort()
			return
		}

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return resolveSigningKey(provider, token)
		})

		if err != nil || !token.Valid {
			common.ErrorResponse(c, http.StatusUnauthorized, "invalid or expired token")
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			common.ErrorResponse(c, http.StatusUnauthorized, "invalid token claims")
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}

func resolveSigningKey(provider jwtkeys.KeyProvider, token *jwt.Token) ([]byte, error) {
	if provider == nil {
		return nil, errors.New("jwt provider is nil")
	}

	var kid string
	if headerKid, ok := token.Header["kid"]; ok {
		kid, _ = headerKid.(string)
	}

	if kid != "" {
		return provider.ResolveKey(kid)
	}

	legacy := provider.LegacyKey()
	if len(legacy) == 0 {
		return nil, jwtkeys.ErrKeyNotFound
	}
	return legacy, nil
}

// RequireRole middleware checks if user has required role
func RequireRole(roles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			common.ErrorResponse(c, http.StatusUnauthorized, "user role not found")
			c.Abort()
			return
		}

		role := userRole.(models.UserRole)

		// Check if user has any of the required roles
		hasRole := false
		for _, requiredRole := range roles {
			if role == requiredRole {
				hasRole = true
				break
			}
		}

		if !hasRole {
			common.ErrorResponse(c, http.StatusForbidden, "insufficient permissions")
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetUserID extracts user ID from context
func GetUserID(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, common.ErrUnauthorized
	}
	return userID.(uuid.UUID), nil
}

// GetUserRole extracts user role from context
func GetUserRole(c *gin.Context) (models.UserRole, error) {
	role, exists := c.Get("user_role")
	if !exists {
		return "", common.ErrUnauthorized
	}
	return role.(models.UserRole), nil
}
