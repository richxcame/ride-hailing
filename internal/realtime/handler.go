package realtime

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/richxcame/ride-hailing/pkg/common"
	ws "github.com/richxcame/ride-hailing/pkg/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Get allowed origins from environment
		allowedOrigins := os.Getenv("CORS_ORIGINS")
		if allowedOrigins == "" {
			// Development fallback
			allowedOrigins = "http://localhost:3000"
		}

		origin := r.Header.Get("Origin")
		if origin == "" {
			// Allow requests with no origin (e.g., mobile apps, Postman)
			return true
		}

		// Check if origin is in allowed list
		origins := strings.Split(allowedOrigins, ",")
		for _, allowedOrigin := range origins {
			if strings.TrimSpace(allowedOrigin) == origin {
				return true
			}
		}

		zap.L().Warn("WebSocket connection rejected from origin", zap.String("origin", origin))
		return false
	},
}

// Handler handles HTTP requests for real-time service
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new handler
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// HandleWebSocket handles WebSocket connection upgrades
func (h *Handler) HandleWebSocket(c *gin.Context) {
	// Extract user ID and role from JWT token (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	role, exists := c.Get("user_role")
	if !exists {
		role = "rider" // Default to rider
	}

	// Safely extract userID and role as strings.
	// Auth middleware stores user_id as uuid.UUID and user_role as models.UserRole.
	userIDStr := fmt.Sprintf("%v", userID)

	roleStr := fmt.Sprintf("%v", role)

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("failed to upgrade connection", zap.Error(err))
		return
	}

	// Create new WebSocket client
	client := ws.NewClient(userIDStr, conn, h.service.GetHub(), roleStr, h.logger)

	// Register client with hub
	h.service.GetHub().Register <- client

	// Start client goroutines
	go client.WritePump()
	go client.ReadPump()

	h.logger.Info("WebSocket connection established", zap.Any("user_id", userID), zap.Any("role", role))
}

// GetChatHistory retrieves chat history for a ride
func (h *Handler) GetChatHistory(c *gin.Context) {
	rideID := c.Param("ride_id")
	if rideID == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "ride_id is required")
		return
	}

	// Extract user ID from JWT token
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Verify user is part of this ride
	var count int
	query := `
		SELECT COUNT(*) FROM rides
		WHERE id = $1 AND (rider_id = $2 OR driver_id = $2)
	`
	err := h.service.db.QueryRow(query, rideID, userID).Scan(&count)
	if err != nil || count == 0 {
		common.ErrorResponse(c, http.StatusForbidden, "Not authorized for this ride")
		return
	}

	// Get chat history
	history, err := h.service.GetChatHistory(rideID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve chat history")
		return
	}

	common.SuccessResponse(c, gin.H{
		"ride_id":  rideID,
		"messages": history,
	})
}

// BroadcastRideUpdate broadcasts a ride update (called by other services)
func (h *Handler) BroadcastRideUpdate(c *gin.Context) {
	var req struct {
		RideID string                 `json:"ride_id" binding:"required"`
		Data   map[string]interface{} `json:"data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	h.service.BroadcastRideUpdate(req.RideID, req.Data)

	common.SuccessResponse(c, gin.H{"message": "Broadcast sent"})
}

// BroadcastToUser broadcasts a message to a specific user (called by other services)
func (h *Handler) BroadcastToUser(c *gin.Context) {
	var req struct {
		UserID string                 `json:"user_id" binding:"required"`
		Type   string                 `json:"type" binding:"required"`
		Data   map[string]interface{} `json:"data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	h.service.BroadcastToUser(req.UserID, req.Type, req.Data)

	common.SuccessResponse(c, gin.H{"message": "Broadcast sent"})
}

// GetStats returns connection statistics
func (h *Handler) GetStats(c *gin.Context) {
	stats := h.service.GetStats()
	common.SuccessResponse(c, stats)
}

// HealthCheck returns service health status
func (h *Handler) HealthCheck(c *gin.Context) {
	stats := h.service.GetStats()
	common.SuccessResponse(c, gin.H{
		"status":  "healthy",
		"service": "realtime",
		"stats":   stats,
	})
}
