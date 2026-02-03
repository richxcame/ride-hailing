package websocket

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// Claims represents JWT claims for WebSocket authentication
type Claims struct {
	UserID uuid.UUID       `json:"user_id"`
	Email  string          `json:"email"`
	Role   models.UserRole `json:"role"`
	jwt.RegisteredClaims
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
}

// HandleWebSocket handles WebSocket upgrade and authentication
func HandleWebSocket(c *gin.Context, hub *Hub, jwtProvider jwtkeys.KeyProvider) {
	// Get token from query parameter or header
	tokenString := c.Query("token")
	if tokenString == "" {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}
	}

	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
		return
	}

	// Parse and validate token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return resolveSigningKey(jwtProvider, token)
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket: %v", err)
		return
	}

	// Determine role
	role := "rider"
	if claims.Role == models.RoleDriver {
		role = "driver"
	}

	// Create client
	client := NewClient(claims.UserID.String(), conn, hub, role)

	// Register client with hub
	hub.Register <- client

	// Start read/write pumps
	go client.WritePump()
	go client.ReadPump()
}

// resolveSigningKey resolves the JWT signing key
func resolveSigningKey(provider jwtkeys.KeyProvider, token *jwt.Token) ([]byte, error) {
	if provider == nil {
		return nil, jwt.ErrInvalidKey
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
