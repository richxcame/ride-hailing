package realtime

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/richxcame/ride-hailing/pkg/redis"
	ws "github.com/richxcame/ride-hailing/pkg/websocket"
	"go.uber.org/zap"
)

// Service handles real-time communication
type Service struct {
	hub    *ws.Hub
	db     *sql.DB
	redis  *redis.Client
	logger *zap.Logger
}

// NewService creates a new real-time service
func NewService(hub *ws.Hub, db *sql.DB, redisClient *redis.Client, logger *zap.Logger) *Service {
	s := &Service{
		hub:    hub,
		db:     db,
		redis:  redisClient,
		logger: logger,
	}

	// Register message handlers
	s.registerHandlers()

	return s
}

// registerHandlers registers all message type handlers
func (s *Service) registerHandlers() {
	s.hub.RegisterHandler("location_update", s.handleLocationUpdate)
	s.hub.RegisterHandler("ride_status", s.handleRideStatus)
	s.hub.RegisterHandler("chat_message", s.handleChatMessage)
	s.hub.RegisterHandler("typing", s.handleTyping)
	s.hub.RegisterHandler("join_ride", s.handleJoinRide)
	s.hub.RegisterHandler("leave_ride", s.handleLeaveRide)
}

// handleLocationUpdate handles driver location updates
func (s *Service) handleLocationUpdate(client *ws.Client, msg *ws.Message) {
	// Only drivers can update location
	if client.Role != "driver" {
		s.logger.Warn("non-driver attempted location update", zap.String("client_id", client.ID))
		return
	}

	latitude, latOk := msg.Data["latitude"].(float64)
	longitude, lngOk := msg.Data["longitude"].(float64)

	if !latOk || !lngOk {
		s.logger.Warn("invalid location data from driver", zap.String("client_id", client.ID))
		return
	}

	// Update location in Redis
	ctx := context.Background()
	key := "driver:location:" + client.ID
	locationData := map[string]interface{}{
		"latitude": latitude,
		"longitude": longitude,
		"timestamp": time.Now().Unix(),
		"heading":   msg.Data["heading"],
		"speed":     msg.Data["speed"],
	}

	data, _ := json.Marshal(locationData)
	if err := s.redis.Set(ctx, key, string(data), 5*time.Minute).Err(); err != nil {
		s.logger.Error("failed to cache location", zap.Error(err))
	}

	// If driver is in a ride, broadcast to rider
	rideID := client.GetRide()
	if rideID != "" {
		// Get rider in this ride
		clients := s.hub.GetClientsInRide(rideID)
		for _, c := range clients {
			if c.Role == "rider" {
				c.SendMessage(&ws.Message{
					Type:      "driver_location",
					RideID:    rideID,
					UserID:    client.ID,
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"latitude": latitude,
						"longitude": longitude,
						"heading":   msg.Data["heading"],
						"speed":     msg.Data["speed"],
					},
				})
			}
		}
	}
}

// handleRideStatus handles ride status updates
func (s *Service) handleRideStatus(client *ws.Client, msg *ws.Message) {
	rideID, ok := msg.Data["ride_id"].(string)
	if !ok {
		s.logger.Warn("invalid ride_id in status update")
		return
	}

	status, ok := msg.Data["status"].(string)
	if !ok {
		s.logger.Warn("invalid status in status update")
		return
	}

	// Broadcast status update to all clients in the ride
	s.hub.SendToRide(rideID, &ws.Message{
		Type:      "ride_status_update",
		RideID:    rideID,
		UserID:    client.ID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"status":     status,
			"updated_by": client.ID,
			"role":       client.Role,
		},
	})
}

// handleChatMessage handles chat messages between rider and driver
func (s *Service) handleChatMessage(client *ws.Client, msg *ws.Message) {
	rideID := client.GetRide()
	if rideID == "" {
		s.logger.Warn("client attempted to send chat without being in a ride", zap.String("client_id", client.ID))
		return
	}

	message, ok := msg.Data["message"].(string)
	if !ok || message == "" {
		s.logger.Warn("invalid chat message from client", zap.String("client_id", client.ID))
		return
	}

	// Store message in Redis for chat history
	ctx := context.Background()
	chatKey := "ride:chat:" + rideID
	chatMsg := map[string]interface{}{
		"sender_id":   client.ID,
		"sender_role": client.Role,
		"message":     message,
		"timestamp":   time.Now().Unix(),
	}

	data, _ := json.Marshal(chatMsg)
	if err := s.redis.RPush(ctx, chatKey, string(data)); err != nil {
		s.logger.Error("failed to store chat message", zap.Error(err))
	}

	// Set expiry on chat history (24 hours)
	s.redis.Expire(ctx, chatKey, 24*time.Hour)

	// Broadcast to other clients in the ride
	clients := s.hub.GetClientsInRide(rideID)
	for _, c := range clients {
		if c.ID != client.ID {
			c.SendMessage(&ws.Message{
				Type:      "chat_message",
				RideID:    rideID,
				UserID:    client.ID,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"message":     message,
					"sender_id":   client.ID,
					"sender_role": client.Role,
				},
			})
		}
	}
}

// handleTyping handles typing indicators
func (s *Service) handleTyping(client *ws.Client, msg *ws.Message) {
	rideID := client.GetRide()
	if rideID == "" {
		return
	}

	isTyping, ok := msg.Data["is_typing"].(bool)
	if !ok {
		return
	}

	// Broadcast typing indicator to other clients in the ride
	clients := s.hub.GetClientsInRide(rideID)
	for _, c := range clients {
		if c.ID != client.ID {
			c.SendMessage(&ws.Message{
				Type:      "typing_indicator",
				RideID:    rideID,
				UserID:    client.ID,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"is_typing":   isTyping,
					"sender_id":   client.ID,
					"sender_role": client.Role,
				},
			})
		}
	}
}

// handleJoinRide handles client joining a ride room
func (s *Service) handleJoinRide(client *ws.Client, msg *ws.Message) {
	rideID, ok := msg.Data["ride_id"].(string)
	if !ok || rideID == "" {
		s.logger.Warn("invalid ride_id in join_ride request")
		return
	}

	// Verify the client is part of this ride (check database)
	var count int
	query := `
		SELECT COUNT(*) FROM rides
		WHERE id = $1 AND (rider_id = $2 OR driver_id = $2)
	`
	err := s.db.QueryRow(query, rideID, client.ID).Scan(&count)
	if err != nil || count == 0 {
		s.logger.Warn("client not authorized for ride", zap.String("client_id", client.ID), zap.String("ride_id", rideID))
		client.SendMessage(&ws.Message{
			Type:      "error",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"message": "Not authorized for this ride",
			},
		})
		return
	}

	// Add client to ride room
	s.hub.AddClientToRide(client.ID, rideID)

	// Send confirmation
	client.SendMessage(&ws.Message{
		Type:      "joined_ride",
		RideID:    rideID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"ride_id": rideID,
		},
	})

	// Notify other clients in the ride
	clients := s.hub.GetClientsInRide(rideID)
	for _, c := range clients {
		if c.ID != client.ID {
			c.SendMessage(&ws.Message{
				Type:      "user_joined",
				RideID:    rideID,
				UserID:    client.ID,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"user_id": client.ID,
					"role":    client.Role,
				},
			})
		}
	}
}

// handleLeaveRide handles client leaving a ride room
func (s *Service) handleLeaveRide(client *ws.Client, msg *ws.Message) {
	rideID := client.GetRide()
	if rideID == "" {
		return
	}

	// Notify other clients
	clients := s.hub.GetClientsInRide(rideID)
	for _, c := range clients {
		if c.ID != client.ID {
			c.SendMessage(&ws.Message{
				Type:      "user_left",
				RideID:    rideID,
				UserID:    client.ID,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"user_id": client.ID,
					"role":    client.Role,
				},
			})
		}
	}

	// Remove client from ride room
	s.hub.RemoveClientFromRide(client.ID, rideID)

	// Send confirmation
	client.SendMessage(&ws.Message{
		Type:      "left_ride",
		RideID:    rideID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"ride_id": rideID,
		},
	})
}

// BroadcastRideUpdate broadcasts a ride update to all clients in the ride
func (s *Service) BroadcastRideUpdate(rideID string, data map[string]interface{}) {
	s.hub.SendToRide(rideID, &ws.Message{
		Type:      "ride_update",
		RideID:    rideID,
		Timestamp: time.Now(),
		Data:      data,
	})
}

// BroadcastToUser sends a message to a specific user
func (s *Service) BroadcastToUser(userID string, msgType string, data map[string]interface{}) {
	s.hub.SendToUser(userID, &ws.Message{
		Type:      msgType,
		UserID:    userID,
		Timestamp: time.Now(),
		Data:      data,
	})
}

// GetChatHistory retrieves chat history for a ride
func (s *Service) GetChatHistory(rideID string) ([]map[string]interface{}, error) {
	ctx := context.Background()
	chatKey := "ride:chat:" + rideID
	messages, err := s.redis.LRange(ctx, chatKey, 0, -1)
	if err != nil {
		return nil, err
	}

	history := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		var chatMsg map[string]interface{}
		if err := json.Unmarshal([]byte(msg), &chatMsg); err != nil {
			continue
		}
		history = append(history, chatMsg)
	}

	return history, nil
}

// GetHub returns the WebSocket hub
func (s *Service) GetHub() *ws.Hub {
	return s.hub
}

// GetStats returns connection statistics
func (s *Service) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"connected_clients": s.hub.GetClientCount(),
		"active_rides":      s.hub.GetRideCount(),
	}
}
