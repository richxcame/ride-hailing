package realtime

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	ws "github.com/richxcame/ride-hailing/pkg/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: In production, implement proper origin checking
		return true
	},
}

// Handler handles HTTP requests for real-time service
type Handler struct {
	service *Service
}

// NewHandler creates a new handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// HandleWebSocket handles WebSocket connection upgrades
func (h *Handler) HandleWebSocket(c *gin.Context) {
	// Extract user ID and role from JWT token
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	role, exists := c.Get("role")
	if !exists {
		role = "rider" // Default to rider
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Create new WebSocket client
	client := ws.NewClient(userID.(string), conn, h.service.GetHub(), role.(string))

	// Register client with hub
	h.service.GetHub().Register <- client

	// Start client goroutines
	go client.WritePump()
	go client.ReadPump()

	log.Printf("WebSocket connection established for user %s (role: %s)", userID, role)
}

// GetChatHistory retrieves chat history for a ride
func (h *Handler) GetChatHistory(c *gin.Context) {
	rideID := c.Param("ride_id")
	if rideID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ride_id is required"})
		return
	}

	// Extract user ID from JWT token
	userID, _ := c.Get("user_id")

	// Verify user is part of this ride
	var count int
	query := `
		SELECT COUNT(*) FROM rides
		WHERE id = $1 AND (rider_id = $2 OR driver_id = $2)
	`
	err := h.service.db.QueryRow(query, rideID, userID).Scan(&count)
	if err != nil || count == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized for this ride"})
		return
	}

	// Get chat history
	history, err := h.service.GetChatHistory(rideID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve chat history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.service.BroadcastRideUpdate(req.RideID, req.Data)

	c.JSON(http.StatusOK, gin.H{"message": "Broadcast sent"})
}

// BroadcastToUser broadcasts a message to a specific user (called by other services)
func (h *Handler) BroadcastToUser(c *gin.Context) {
	var req struct {
		UserID  string                 `json:"user_id" binding:"required"`
		Type    string                 `json:"type" binding:"required"`
		Data    map[string]interface{} `json:"data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.service.BroadcastToUser(req.UserID, req.Type, req.Data)

	c.JSON(http.StatusOK, gin.H{"message": "Broadcast sent"})
}

// GetStats returns connection statistics
func (h *Handler) GetStats(c *gin.Context) {
	stats := h.service.GetStats()
	c.JSON(http.StatusOK, stats)
}

// GetDriverLocation gets a driver's current location from Redis
func (h *Handler) GetDriverLocation(c *gin.Context) {
	driverID := c.Param("driver_id")
	if driverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "driver_id is required"})
		return
	}

	ctx := context.Background()
	key := "driver:location:" + driverID
	location, err := h.service.redis.Get(ctx, key).Result()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Driver location not found"})
		return
	}

	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, location)
}

// HealthCheck returns service health status
func (h *Handler) HealthCheck(c *gin.Context) {
	stats := h.service.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "realtime",
		"stats":   stats,
	})
}
