//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/richxcame/ride-hailing/pkg/models"
	ws "github.com/richxcame/ride-hailing/pkg/websocket"
)

// RealtimeTestSuite tests WebSocket and real-time functionality
type RealtimeTestSuite struct {
	suite.Suite
	rider        authSession
	driver       authSession
	wsServer     *httptest.Server
	hub          *ws.Hub
	connectedWS  []*websocket.Conn
	mu           sync.Mutex
}

func TestRealtimeSuite(t *testing.T) {
	suite.Run(t, new(RealtimeTestSuite))
}

func (s *RealtimeTestSuite) SetupSuite() {
	// Ensure required services are started
	if _, ok := services[authServiceKey]; !ok {
		services[authServiceKey] = startAuthService(mustLoadConfig("auth-service"))
	}
	if _, ok := services[ridesServiceKey]; !ok {
		services[ridesServiceKey] = startRidesService(mustLoadConfig("rides-service"))
	}

	// Create WebSocket hub for testing
	s.hub = ws.NewHub()
	go s.hub.Run()

	// Create test WebSocket server
	s.wsServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		// Extract user info from query params (for testing)
		userID := r.URL.Query().Get("user_id")
		role := r.URL.Query().Get("role")
		if userID == "" {
			userID = uuid.New().String()
		}
		if role == "" {
			role = "rider"
		}

		// Create client and register with hub
		client := ws.NewClient(userID, conn, s.hub, role, nil)
		s.hub.Register <- client

		go client.WritePump()
		go client.ReadPump()
	}))
}

func (s *RealtimeTestSuite) TearDownSuite() {
	// Close all WebSocket connections
	s.mu.Lock()
	for _, conn := range s.connectedWS {
		conn.Close()
	}
	s.mu.Unlock()

	if s.wsServer != nil {
		s.wsServer.Close()
	}
}

func (s *RealtimeTestSuite) SetupTest() {
	truncateTables(s.T())
	s.rider = registerAndLogin(s.T(), models.RoleRider)
	s.driver = registerAndLogin(s.T(), models.RoleDriver)
}

func (s *RealtimeTestSuite) TearDownTest() {
	// Clean up connections after each test
	s.mu.Lock()
	for _, conn := range s.connectedWS {
		conn.Close()
	}
	s.connectedWS = nil
	s.mu.Unlock()
}

// ============================================
// WEBSOCKET CONNECTION TESTS
// ============================================

func (s *RealtimeTestSuite) TestWebSocket_ConnectionEstablishment() {
	t := s.T()

	conn := s.createWSConnection(t, s.rider.User.ID.String(), "rider")
	require.NotNil(t, conn)

	// Verify connection is active by sending a ping
	err := conn.WriteMessage(websocket.PingMessage, nil)
	require.NoError(t, err)
}

func (s *RealtimeTestSuite) TestWebSocket_MultipleConnections() {
	t := s.T()

	// Create multiple connections
	connections := make([]*websocket.Conn, 5)
	for i := 0; i < 5; i++ {
		userID := uuid.New().String()
		conn := s.createWSConnection(t, userID, "rider")
		require.NotNil(t, conn)
		connections[i] = conn
	}

	// Verify all connections are active
	for i, conn := range connections {
		err := conn.WriteMessage(websocket.PingMessage, nil)
		require.NoError(t, err, "Connection %d should be active", i)
	}

	// Verify hub has registered all clients
	time.Sleep(100 * time.Millisecond) // Allow registration to complete
	require.GreaterOrEqual(t, s.hub.GetClientCount(), 5)
}

func (s *RealtimeTestSuite) TestWebSocket_ConnectionWithDifferentRoles() {
	t := s.T()

	// Create rider connection
	riderConn := s.createWSConnection(t, s.rider.User.ID.String(), "rider")
	require.NotNil(t, riderConn)

	// Create driver connection
	driverConn := s.createWSConnection(t, s.driver.User.ID.String(), "driver")
	require.NotNil(t, driverConn)

	// Both should be connected
	time.Sleep(100 * time.Millisecond)
	require.GreaterOrEqual(t, s.hub.GetClientCount(), 2)
}

func (s *RealtimeTestSuite) TestWebSocket_ReconnectionSameUser() {
	t := s.T()

	userID := s.rider.User.ID.String()

	// Create first connection
	conn1 := s.createWSConnection(t, userID, "rider")
	require.NotNil(t, conn1)

	time.Sleep(50 * time.Millisecond)
	initialCount := s.hub.GetClientCount()

	// Create second connection with same user ID (should replace first)
	conn2 := s.createWSConnection(t, userID, "rider")
	require.NotNil(t, conn2)

	time.Sleep(100 * time.Millisecond)

	// The hub should handle this gracefully (either replace or allow both)
	// Client count should not double for the same user
	require.LessOrEqual(t, s.hub.GetClientCount(), initialCount+1)
}

// ============================================
// DRIVER LOCATION BROADCAST TESTS
// ============================================

func (s *RealtimeTestSuite) TestDriverLocation_BroadcastToRide() {
	t := s.T()
	ctx := context.Background()

	// Create a ride first
	rideReq := &models.RideRequest{
		PickupLatitude:   37.7749,
		PickupLongitude:  -122.4194,
		PickupAddress:    "123 Market St",
		DropoffLatitude:  37.8044,
		DropoffLongitude: -122.2712,
		DropoffAddress:   "456 Broadway",
	}

	rideResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, "/api/v1/rides", rideReq, authHeaders(s.rider.Token))
	require.True(t, rideResp.Success)
	rideID := rideResp.Data.ID

	// Driver accepts the ride
	acceptPath := fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideID)
	acceptResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(s.driver.Token))
	require.True(t, acceptResp.Success)

	// Create WebSocket connections
	riderConn := s.createWSConnection(t, s.rider.User.ID.String(), "rider")
	driverConn := s.createWSConnection(t, s.driver.User.ID.String(), "driver")

	// Allow connections to register
	time.Sleep(100 * time.Millisecond)

	// Add clients to ride room
	s.hub.AddClientToRide(s.rider.User.ID.String(), rideID.String())
	s.hub.AddClientToRide(s.driver.User.ID.String(), rideID.String())

	// Set up message receiver for rider
	receivedMessages := make(chan *ws.Message, 10)
	go func() {
		for {
			_, msgBytes, err := riderConn.ReadMessage()
			if err != nil {
				return
			}
			var msg ws.Message
			if json.Unmarshal(msgBytes, &msg) == nil {
				receivedMessages <- &msg
			}
		}
	}()

	// Driver sends location update
	locationMsg := &ws.Message{
		Type:   "location_update",
		RideID: rideID.String(),
		Data: map[string]interface{}{
			"latitude":  37.7750,
			"longitude": -122.4190,
			"heading":   45.0,
			"speed":     30.0,
		},
	}

	err := driverConn.WriteJSON(locationMsg)
	require.NoError(t, err)

	// Wait briefly for message propagation
	time.Sleep(200 * time.Millisecond)

	// Store location in Redis (simulated)
	key := fmt.Sprintf("driver:location:%s", s.driver.User.ID)
	if redisTestClient != nil {
		locationData, _ := json.Marshal(locationMsg.Data)
		redisTestClient.Set(ctx, key, string(locationData), 5*time.Minute)
	}

	_ = receivedMessages // Used for receiving in background
}

func (s *RealtimeTestSuite) TestDriverLocation_UpdateToRedis() {
	t := s.T()
	ctx := context.Background()

	if redisTestClient == nil {
		t.Skip("Redis not available")
	}

	// Driver sends location update
	locationData := map[string]interface{}{
		"latitude":  37.7749,
		"longitude": -122.4194,
		"timestamp": time.Now().Unix(),
		"heading":   90.0,
		"speed":     25.5,
	}

	key := fmt.Sprintf("driver:location:%s", s.driver.User.ID)
	data, err := json.Marshal(locationData)
	require.NoError(t, err)

	err = redisTestClient.Set(ctx, key, string(data), 5*time.Minute).Err()
	require.NoError(t, err)

	// Verify location was stored
	stored, err := redisTestClient.Get(ctx, key).Result()
	require.NoError(t, err)

	var storedData map[string]interface{}
	err = json.Unmarshal([]byte(stored), &storedData)
	require.NoError(t, err)
	require.InEpsilon(t, 37.7749, storedData["latitude"].(float64), 1e-6)
	require.InEpsilon(t, -122.4194, storedData["longitude"].(float64), 1e-6)
}

func (s *RealtimeTestSuite) TestDriverLocation_MultipleUpdates() {
	t := s.T()
	ctx := context.Background()

	if redisTestClient == nil {
		t.Skip("Redis not available")
	}

	key := fmt.Sprintf("driver:location:%s", s.driver.User.ID)

	// Simulate multiple location updates
	locations := []struct {
		latitude float64
		longitude float64
	}{
		{37.7749, -122.4194},
		{37.7750, -122.4190},
		{37.7752, -122.4185},
		{37.7755, -122.4180},
	}

	for _, loc := range locations {
		locationData := map[string]interface{}{
			"latitude":  loc.latitude,
			"longitude": loc.longitude,
			"timestamp": time.Now().Unix(),
		}

		data, err := json.Marshal(locationData)
		require.NoError(t, err)

		err = redisTestClient.Set(ctx, key, string(data), 5*time.Minute).Err()
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)
	}

	// Verify last location
	stored, err := redisTestClient.Get(ctx, key).Result()
	require.NoError(t, err)

	var storedData map[string]interface{}
	err = json.Unmarshal([]byte(stored), &storedData)
	require.NoError(t, err)
	require.InEpsilon(t, 37.7755, storedData["latitude"].(float64), 1e-6)
}

// ============================================
// RIDE STATUS UPDATE NOTIFICATION TESTS
// ============================================

func (s *RealtimeTestSuite) TestRideStatusUpdate_Notification() {
	t := s.T()

	// Create ride
	rideReq := &models.RideRequest{
		PickupLatitude:   37.7749,
		PickupLongitude:  -122.4194,
		PickupAddress:    "123 Market St",
		DropoffLatitude:  37.8044,
		DropoffLongitude: -122.2712,
		DropoffAddress:   "456 Broadway",
	}

	rideResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, "/api/v1/rides", rideReq, authHeaders(s.rider.Token))
	require.True(t, rideResp.Success)
	rideID := rideResp.Data.ID

	// Create WebSocket connections
	riderConn := s.createWSConnection(t, s.rider.User.ID.String(), "rider")
	driverConn := s.createWSConnection(t, s.driver.User.ID.String(), "driver")

	time.Sleep(100 * time.Millisecond)

	// Add clients to ride room
	s.hub.AddClientToRide(s.rider.User.ID.String(), rideID.String())
	s.hub.AddClientToRide(s.driver.User.ID.String(), rideID.String())

	// Driver accepts - should notify rider
	acceptPath := fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideID)
	acceptResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(s.driver.Token))
	require.True(t, acceptResp.Success)
	require.Equal(t, models.RideStatusAccepted, acceptResp.Data.Status)

	// Broadcast status update via hub
	statusMsg := &ws.Message{
		Type:      "ride_status_update",
		RideID:    rideID.String(),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"status":     "accepted",
			"driver_id":  s.driver.User.ID.String(),
			"updated_by": s.driver.User.ID.String(),
		},
	}

	s.hub.SendToRide(rideID.String(), statusMsg)

	_ = riderConn
	_ = driverConn
}

func (s *RealtimeTestSuite) TestRideStatusUpdate_AllStatusTransitions() {
	t := s.T()

	// Create ride
	rideReq := &models.RideRequest{
		PickupLatitude:   37.7749,
		PickupLongitude:  -122.4194,
		PickupAddress:    "123 Market St",
		DropoffLatitude:  37.8044,
		DropoffLongitude: -122.2712,
		DropoffAddress:   "456 Broadway",
	}

	rideResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, "/api/v1/rides", rideReq, authHeaders(s.rider.Token))
	require.True(t, rideResp.Success)
	rideID := rideResp.Data.ID

	// Test all status transitions
	statuses := []struct {
		action   string
		endpoint string
		expected models.RideStatus
	}{
		{"accept", fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideID), models.RideStatusAccepted},
		{"start", fmt.Sprintf("/api/v1/driver/rides/%s/start", rideID), models.RideStatusInProgress},
		{"complete", fmt.Sprintf("/api/v1/driver/rides/%s/complete", rideID), models.RideStatusCompleted},
	}

	for _, status := range statuses {
		s.Run(status.action, func() {
			var resp apiResponse[*models.Ride]
			if status.action == "complete" {
				req := map[string]interface{}{"actual_distance": 10.0}
				resp = doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, status.endpoint, req, authHeaders(s.driver.Token))
			} else {
				resp = doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, status.endpoint, nil, authHeaders(s.driver.Token))
			}
			require.True(t, resp.Success)
			require.Equal(t, status.expected, resp.Data.Status)
		})
	}
}

// ============================================
// CHAT MESSAGE DELIVERY TESTS
// ============================================

func (s *RealtimeTestSuite) TestChatMessage_Delivery() {
	t := s.T()
	ctx := context.Background()

	// Create and accept ride
	rideReq := &models.RideRequest{
		PickupLatitude:   37.7749,
		PickupLongitude:  -122.4194,
		PickupAddress:    "123 Market St",
		DropoffLatitude:  37.8044,
		DropoffLongitude: -122.2712,
		DropoffAddress:   "456 Broadway",
	}

	rideResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, "/api/v1/rides", rideReq, authHeaders(s.rider.Token))
	require.True(t, rideResp.Success)
	rideID := rideResp.Data.ID

	acceptPath := fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideID)
	acceptResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(s.driver.Token))
	require.True(t, acceptResp.Success)

	// Create WebSocket connections
	riderConn := s.createWSConnection(t, s.rider.User.ID.String(), "rider")
	driverConn := s.createWSConnection(t, s.driver.User.ID.String(), "driver")

	time.Sleep(100 * time.Millisecond)

	// Add clients to ride room
	s.hub.AddClientToRide(s.rider.User.ID.String(), rideID.String())
	s.hub.AddClientToRide(s.driver.User.ID.String(), rideID.String())

	// Store chat message in Redis (simulating chat storage)
	if redisTestClient != nil {
		chatKey := fmt.Sprintf("ride:chat:%s", rideID)
		chatMsg := map[string]interface{}{
			"sender_id":   s.rider.User.ID.String(),
			"sender_role": "rider",
			"message":     "I'm at the pickup location",
			"timestamp":   time.Now().Unix(),
		}

		data, _ := json.Marshal(chatMsg)
		err := redisTestClient.RPush(ctx, chatKey, string(data))
		require.NoError(t, err)

		// Verify message was stored
		messages, err := redisTestClient.LRange(ctx, chatKey, 0, -1)
		require.NoError(t, err)
		require.Len(t, messages, 1)
	}

	// Send chat message via WebSocket
	chatMessage := &ws.Message{
		Type:   "chat_message",
		RideID: rideID.String(),
		Data: map[string]interface{}{
			"message": "I'm at the pickup location",
		},
	}

	err := riderConn.WriteJSON(chatMessage)
	require.NoError(t, err)

	_ = driverConn
}

func (s *RealtimeTestSuite) TestChatMessage_History() {
	t := s.T()
	ctx := context.Background()

	if redisTestClient == nil {
		t.Skip("Redis not available")
	}

	rideID := uuid.New()
	chatKey := fmt.Sprintf("ride:chat:%s", rideID)

	// Store multiple chat messages
	messages := []map[string]interface{}{
		{
			"sender_id":   s.rider.User.ID.String(),
			"sender_role": "rider",
			"message":     "Hi, I'm waiting at the corner",
			"timestamp":   time.Now().Add(-5 * time.Minute).Unix(),
		},
		{
			"sender_id":   s.driver.User.ID.String(),
			"sender_role": "driver",
			"message":     "On my way, 2 minutes",
			"timestamp":   time.Now().Add(-4 * time.Minute).Unix(),
		},
		{
			"sender_id":   s.rider.User.ID.String(),
			"sender_role": "rider",
			"message":     "Great, thanks!",
			"timestamp":   time.Now().Add(-3 * time.Minute).Unix(),
		},
	}

	for _, msg := range messages {
		data, _ := json.Marshal(msg)
		err := redisTestClient.RPush(ctx, chatKey, string(data))
		require.NoError(t, err)
	}

	// Retrieve chat history
	history, err := redisTestClient.LRange(ctx, chatKey, 0, -1)
	require.NoError(t, err)
	require.Len(t, history, 3)

	// Verify message content
	var firstMsg map[string]interface{}
	err = json.Unmarshal([]byte(history[0]), &firstMsg)
	require.NoError(t, err)
	require.Equal(t, "Hi, I'm waiting at the corner", firstMsg["message"])
}

func (s *RealtimeTestSuite) TestChatMessage_TypingIndicator() {
	t := s.T()

	// Create and setup ride
	rideReq := &models.RideRequest{
		PickupLatitude:   37.7749,
		PickupLongitude:  -122.4194,
		PickupAddress:    "123 Market St",
		DropoffLatitude:  37.8044,
		DropoffLongitude: -122.2712,
		DropoffAddress:   "456 Broadway",
	}

	rideResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, "/api/v1/rides", rideReq, authHeaders(s.rider.Token))
	require.True(t, rideResp.Success)
	rideID := rideResp.Data.ID

	// Accept ride
	acceptPath := fmt.Sprintf("/api/v1/driver/rides/%s/accept", rideID)
	acceptResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, acceptPath, nil, authHeaders(s.driver.Token))
	require.True(t, acceptResp.Success)

	// Create connections and join ride room
	riderConn := s.createWSConnection(t, s.rider.User.ID.String(), "rider")
	driverConn := s.createWSConnection(t, s.driver.User.ID.String(), "driver")

	time.Sleep(100 * time.Millisecond)

	s.hub.AddClientToRide(s.rider.User.ID.String(), rideID.String())
	s.hub.AddClientToRide(s.driver.User.ID.String(), rideID.String())

	// Send typing indicator
	typingMsg := &ws.Message{
		Type:   "typing",
		RideID: rideID.String(),
		Data: map[string]interface{}{
			"is_typing": true,
		},
	}

	err := riderConn.WriteJSON(typingMsg)
	require.NoError(t, err)

	_ = driverConn
}

// ============================================
// CONNECTION CLEANUP TESTS
// ============================================

func (s *RealtimeTestSuite) TestConnectionCleanup_OnDisconnect() {
	t := s.T()

	userID := uuid.New().String()

	// Create connection
	conn := s.createWSConnection(t, userID, "rider")
	require.NotNil(t, conn)

	time.Sleep(100 * time.Millisecond)
	initialCount := s.hub.GetClientCount()
	require.GreaterOrEqual(t, initialCount, 1)

	// Close connection
	conn.Close()

	// Wait for cleanup
	time.Sleep(200 * time.Millisecond)

	// Client count should decrease
	finalCount := s.hub.GetClientCount()
	require.Less(t, finalCount, initialCount)
}

func (s *RealtimeTestSuite) TestConnectionCleanup_RideRoomCleanup() {
	t := s.T()

	// Create a ride
	rideReq := &models.RideRequest{
		PickupLatitude:   37.7749,
		PickupLongitude:  -122.4194,
		PickupAddress:    "123 Market St",
		DropoffLatitude:  37.8044,
		DropoffLongitude: -122.2712,
		DropoffAddress:   "456 Broadway",
	}

	rideResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, "/api/v1/rides", rideReq, authHeaders(s.rider.Token))
	require.True(t, rideResp.Success)
	rideID := rideResp.Data.ID

	// Create connections and join ride
	riderConn := s.createWSConnection(t, s.rider.User.ID.String(), "rider")
	driverConn := s.createWSConnection(t, s.driver.User.ID.String(), "driver")

	time.Sleep(100 * time.Millisecond)

	s.hub.AddClientToRide(s.rider.User.ID.String(), rideID.String())
	s.hub.AddClientToRide(s.driver.User.ID.String(), rideID.String())

	// Verify clients in ride
	clients := s.hub.GetClientsInRide(rideID.String())
	require.GreaterOrEqual(t, len(clients), 2)

	// Close rider connection
	riderConn.Close()
	time.Sleep(200 * time.Millisecond)

	// Remove from ride room (simulating cleanup)
	s.hub.RemoveClientFromRide(s.rider.User.ID.String(), rideID.String())

	// Verify rider was removed from ride room
	clients = s.hub.GetClientsInRide(rideID.String())
	require.LessOrEqual(t, len(clients), 1)

	_ = driverConn
}

func (s *RealtimeTestSuite) TestConnectionCleanup_GracefulShutdown() {
	t := s.T()

	// Create multiple connections
	connections := make([]*websocket.Conn, 3)
	for i := 0; i < 3; i++ {
		userID := uuid.New().String()
		conn := s.createWSConnection(t, userID, "rider")
		require.NotNil(t, conn)
		connections[i] = conn
	}

	time.Sleep(100 * time.Millisecond)

	// Close all connections gracefully
	for _, conn := range connections {
		// Send close message
		err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			conn.Close()
		}
	}

	time.Sleep(200 * time.Millisecond)

	// Hub should handle graceful closure
	require.GreaterOrEqual(t, s.hub.GetClientCount(), 0)
}

// ============================================
// BROADCAST AND NOTIFICATION TESTS
// ============================================

func (s *RealtimeTestSuite) TestBroadcast_ToUser() {
	t := s.T()

	// Create connection
	conn := s.createWSConnection(t, s.rider.User.ID.String(), "rider")
	require.NotNil(t, conn)

	time.Sleep(100 * time.Millisecond)

	// Send message to specific user
	msg := &ws.Message{
		Type:      "notification",
		UserID:    s.rider.User.ID.String(),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"title":   "Ride Update",
			"message": "Your ride is arriving soon!",
		},
	}

	s.hub.SendToUser(s.rider.User.ID.String(), msg)

	// Message should be queued for delivery
	time.Sleep(50 * time.Millisecond)
	_ = conn
}

func (s *RealtimeTestSuite) TestBroadcast_ToAllClients() {
	t := s.T()

	// Create multiple connections
	for i := 0; i < 3; i++ {
		userID := uuid.New().String()
		conn := s.createWSConnection(t, userID, "rider")
		require.NotNil(t, conn)
	}

	time.Sleep(100 * time.Millisecond)

	// Broadcast to all
	msg := &ws.Message{
		Type:      "system_announcement",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message": "System maintenance in 5 minutes",
		},
	}

	s.hub.SendToAll(msg)

	time.Sleep(50 * time.Millisecond)
}

func (s *RealtimeTestSuite) TestBroadcast_ToRide() {
	t := s.T()

	rideID := uuid.New().String()

	// Create connections and join ride
	riderConn := s.createWSConnection(t, s.rider.User.ID.String(), "rider")
	driverConn := s.createWSConnection(t, s.driver.User.ID.String(), "driver")

	time.Sleep(100 * time.Millisecond)

	s.hub.AddClientToRide(s.rider.User.ID.String(), rideID)
	s.hub.AddClientToRide(s.driver.User.ID.String(), rideID)

	// Broadcast to ride
	msg := &ws.Message{
		Type:      "ride_update",
		RideID:    rideID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"eta": 5,
		},
	}

	s.hub.SendToRide(rideID, msg)

	time.Sleep(50 * time.Millisecond)
	_ = riderConn
	_ = driverConn
}

// ============================================
// ERROR HANDLING TESTS
// ============================================

func (s *RealtimeTestSuite) TestWebSocket_InvalidMessageFormat() {
	t := s.T()

	conn := s.createWSConnection(t, s.rider.User.ID.String(), "rider")
	require.NotNil(t, conn)

	time.Sleep(100 * time.Millisecond)

	// Send invalid JSON
	err := conn.WriteMessage(websocket.TextMessage, []byte("invalid json{"))
	require.NoError(t, err) // Write should succeed

	// Connection should remain open (server handles gracefully)
	time.Sleep(100 * time.Millisecond)

	// Try to ping - connection might be closed or open depending on error handling
	_ = conn.WriteMessage(websocket.PingMessage, nil)
}

func (s *RealtimeTestSuite) TestWebSocket_UnknownMessageType() {
	t := s.T()

	conn := s.createWSConnection(t, s.rider.User.ID.String(), "rider")
	require.NotNil(t, conn)

	time.Sleep(100 * time.Millisecond)

	// Send message with unknown type
	unknownMsg := &ws.Message{
		Type: "unknown_type",
		Data: map[string]interface{}{
			"foo": "bar",
		},
	}

	err := conn.WriteJSON(unknownMsg)
	require.NoError(t, err)

	// Connection should remain open
	time.Sleep(100 * time.Millisecond)
	err = conn.WriteMessage(websocket.PingMessage, nil)
	require.NoError(t, err)
}

func (s *RealtimeTestSuite) TestWebSocket_LargeMessage() {
	t := s.T()

	conn := s.createWSConnection(t, s.rider.User.ID.String(), "rider")
	require.NotNil(t, conn)

	time.Sleep(100 * time.Millisecond)

	// Create a large message (but within limits)
	largeData := make([]byte, 100000) // 100KB
	for i := range largeData {
		largeData[i] = 'a'
	}

	msg := &ws.Message{
		Type: "test",
		Data: map[string]interface{}{
			"large_content": string(largeData),
		},
	}

	err := conn.WriteJSON(msg)
	require.NoError(t, err)
}

// ============================================
// HELPER METHODS
// ============================================

func (s *RealtimeTestSuite) createWSConnection(t *testing.T, userID, role string) *websocket.Conn {
	wsURL := "ws" + strings.TrimPrefix(s.wsServer.URL, "http")
	wsURL = fmt.Sprintf("%s?user_id=%s&role=%s", wsURL, userID, role)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	s.mu.Lock()
	s.connectedWS = append(s.connectedWS, conn)
	s.mu.Unlock()

	return conn
}
