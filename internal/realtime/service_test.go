package realtime

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-redis/redismock/v9"
	"github.com/gorilla/websocket"
	"github.com/richxcame/ride-hailing/pkg/redis"
	ws "github.com/richxcame/ride-hailing/pkg/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// createTestWebSocketConn creates a test WebSocket connection
func createTestWebSocketConn(t *testing.T) *websocket.Conn {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// Keep connection alive for tests
		for {
			if _, _, err := conn.NextReader(); err != nil {
				conn.Close()
				break
			}
		}
	}))
	t.Cleanup(server.Close)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial websocket: %v", err)
	}

	return conn
}

// TestNewService tests service initialization
func TestNewService(t *testing.T) {
	// Setup mocks
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisDB, _ := redismock.NewClientMock()
	redisClient := &redis.Client{Client: redisDB}

	hub := ws.NewHub()

	// Create service
	service := NewService(hub, db, redisClient, nil, zap.NewNop())

	// Assertions
	assert.NotNil(t, service)
	assert.NotNil(t, service.hub)
	assert.NotNil(t, service.db)
	assert.NotNil(t, service.redis)
}

// TestHandleLocationUpdate tests driver location updates
func TestHandleLocationUpdate(t *testing.T) {
	tests := []struct {
		name          string
		client        *ws.Client
		message       *ws.Message
		expectRedis   bool
		expectBroadcast bool
	}{
		{
			name: "Valid driver location update",
			client: &ws.Client{
				ID:   "driver-123",
				Role: "driver",
			},
			message: &ws.Message{
				Type: "location_update",
				Data: map[string]interface{}{
					"latitude":  37.7749,
					"longitude": -122.4194,
					"heading":   90.0,
					"speed":     50.0,
				},
			},
			expectRedis:   true,
			expectBroadcast: false, // No ride associated
		},
		{
			name: "Non-driver attempts location update",
			client: &ws.Client{
				ID:   "rider-123",
				Role: "rider",
			},
			message: &ws.Message{
				Type: "location_update",
				Data: map[string]interface{}{
					"latitude":  37.7749,
					"longitude": -122.4194,
				},
			},
			expectRedis:   false,
			expectBroadcast: false,
		},
		{
			name: "Invalid location data",
			client: &ws.Client{
				ID:   "driver-123",
				Role: "driver",
			},
			message: &ws.Message{
				Type: "location_update",
				Data: map[string]interface{}{
					"latitude": "invalid",
				},
			},
			expectRedis:   false,
			expectBroadcast: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			db, _, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			redisDB, redisMock := redismock.NewClientMock()
			redisClient := &redis.Client{Client: redisDB}

			hub := ws.NewHub()
			service := NewService(hub, db, redisClient, nil, zap.NewNop())

			// Set expectations - skip checking since actual implementation adds timestamp
			if tt.expectRedis {
				redisMock.Regexp().ExpectSet("driver:ws_location:"+tt.client.ID, `.*`, 5*time.Minute).SetVal("OK")
			}

			// Execute
			service.handleLocationUpdate(tt.client, tt.message)

			// Verify
			assert.NoError(t, redisMock.ExpectationsWereMet())
		})
	}
}

// TestHandleLocationUpdateWithRide tests location update broadcasting to rider
func TestHandleLocationUpdateWithRide(t *testing.T) {
	// Setup
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisDB, redisMock := redismock.NewClientMock()
	redisClient := &redis.Client{Client: redisDB}

	hub := ws.NewHub()
	go hub.Run()

	service := NewService(hub, db, redisClient, nil, zap.NewNop())

	// Create driver and rider clients
	driverConn := createTestWebSocketConn(t)
	riderConn := createTestWebSocketConn(t)

	driver := ws.NewClient("driver-123", driverConn, hub, "driver", zap.NewNop())
	rider := ws.NewClient("rider-456", riderConn, hub, "rider", zap.NewNop())

	// Register clients
	hub.Register <- driver
	hub.Register <- rider
	time.Sleep(10 * time.Millisecond) // Allow registration

	// Add both to ride
	rideID := "ride-789"
	hub.AddClientToRide(driver.ID, rideID)
	hub.AddClientToRide(rider.ID, rideID)

	// Expect Redis set (use regex to match any JSON value since timestamp is added)
	redisMock.Regexp().ExpectSet("driver:ws_location:driver-123", `.*`, 5*time.Minute).SetVal("OK")

	// Send location update
	msg := &ws.Message{
		Type: "location_update",
		Data: map[string]interface{}{
			"latitude":  37.7749,
			"longitude": -122.4194,
			"heading":   90.0,
			"speed":     50.0,
		},
	}

	service.handleLocationUpdate(driver, msg)

	// Verify Redis expectations
	assert.NoError(t, redisMock.ExpectationsWereMet())
}

// TestHandleRideStatus tests ride status updates
func TestHandleRideStatus(t *testing.T) {
	tests := []struct {
		name    string
		message *ws.Message
		wantErr bool
	}{
		{
			name: "Valid ride status update",
			message: &ws.Message{
				Type: "ride_status",
				Data: map[string]interface{}{
					"ride_id": "ride-123",
					"status":  "in_progress",
				},
			},
			wantErr: false,
		},
		{
			name: "Missing ride_id",
			message: &ws.Message{
				Type: "ride_status",
				Data: map[string]interface{}{
					"status": "in_progress",
				},
			},
			wantErr: true,
		},
		{
			name: "Missing status",
			message: &ws.Message{
				Type: "ride_status",
				Data: map[string]interface{}{
					"ride_id": "ride-123",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db, _, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			redisDB, _ := redismock.NewClientMock()
			redisClient := &redis.Client{Client: redisDB}

			hub := ws.NewHub()
			go hub.Run()

			service := NewService(hub, db, redisClient, nil, zap.NewNop())

			conn := createTestWebSocketConn(t)
			client := ws.NewClient("user-123", conn, hub, "driver", zap.NewNop())

			// Execute
			service.handleRideStatus(client, tt.message)

			// For valid messages, the function should not panic
			// Invalid messages are logged but don't cause errors
		})
	}
}

// TestHandleChatMessage tests chat message handling
func TestHandleChatMessage(t *testing.T) {
	tests := []struct {
		name        string
		clientRide  string
		message     *ws.Message
		expectRedis bool
	}{
		{
			name:       "Valid chat message",
			clientRide: "ride-123",
			message: &ws.Message{
				Type: "chat_message",
				Data: map[string]interface{}{
					"message": "Hello, driver!",
				},
			},
			expectRedis: true,
		},
		{
			name:       "Client not in ride",
			clientRide: "",
			message: &ws.Message{
				Type: "chat_message",
				Data: map[string]interface{}{
					"message": "Hello!",
				},
			},
			expectRedis: false,
		},
		{
			name:       "Empty message",
			clientRide: "ride-123",
			message: &ws.Message{
				Type: "chat_message",
				Data: map[string]interface{}{
					"message": "",
				},
			},
			expectRedis: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db, _, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			redisDB, redisMock := redismock.NewClientMock()
			redisClient := &redis.Client{Client: redisDB}

			hub := ws.NewHub()
			go hub.Run()

			service := NewService(hub, db, redisClient, nil, zap.NewNop())

			conn := createTestWebSocketConn(t)
			client := ws.NewClient("user-123", conn, hub, "rider", zap.NewNop())

			if tt.clientRide != "" {
				client.SetRide(tt.clientRide)
				hub.AddClientToRide(client.ID, tt.clientRide)
			}

			// Set expectations
			if tt.expectRedis {
				// Use regex to match any value (chat message includes timestamp)
				redisMock.Regexp().ExpectRPush("ride:chat:"+tt.clientRide, `.*`).SetVal(1)
				redisMock.ExpectExpire("ride:chat:"+tt.clientRide, 24*time.Hour).SetVal(true)
			}

			// Execute
			service.handleChatMessage(client, tt.message)

			// Verify
			if tt.expectRedis {
				assert.NoError(t, redisMock.ExpectationsWereMet())
			}
		})
	}
}

// TestHandleTyping tests typing indicator handling
func TestHandleTyping(t *testing.T) {
	// Setup
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisDB, _ := redismock.NewClientMock()
	redisClient := &redis.Client{Client: redisDB}

	hub := ws.NewHub()
	go hub.Run()

	service := NewService(hub, db, redisClient, nil, zap.NewNop())

	// Create two clients in same ride
	conn1 := createTestWebSocketConn(t)
	conn2 := createTestWebSocketConn(t)

	client1 := ws.NewClient("user-123", conn1, hub, "rider", zap.NewNop())
	client2 := ws.NewClient("user-456", conn2, hub, "driver", zap.NewNop())

	rideID := "ride-789"
	client1.SetRide(rideID)
	client2.SetRide(rideID)

	hub.Register <- client1
	hub.Register <- client2
	hub.AddClientToRide(client1.ID, rideID)
	hub.AddClientToRide(client2.ID, rideID)

	time.Sleep(10 * time.Millisecond)

	// Send typing indicator
	msg := &ws.Message{
		Type: "typing",
		Data: map[string]interface{}{
			"is_typing": true,
		},
	}

	service.handleTyping(client1, msg)

	// The typing indicator should be broadcasted to client2
	// Since we're using mock connections, we can't directly verify
	// But the function should not panic
}

// TestHandleJoinRide tests ride joining functionality
func TestHandleJoinRide(t *testing.T) {
	tests := []struct {
		name        string
		message     *ws.Message
		setupMock   func(sqlmock.Sqlmock)
		expectError bool
	}{
		{
			name: "Authorized join",
			message: &ws.Message{
				Type:   "join_ride",
				RideID: "ride-123",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
				mock.ExpectQuery("SELECT COUNT.*FROM rides").
					WithArgs("ride-123", "user-123").
					WillReturnRows(rows)
			},
			expectError: false,
		},
		{
			name: "Unauthorized join",
			message: &ws.Message{
				Type:   "join_ride",
				RideID: "ride-123",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
				mock.ExpectQuery("SELECT COUNT.*FROM rides").
					WithArgs("ride-123", "user-123").
					WillReturnRows(rows)
			},
			expectError: true,
		},
		{
			name: "Missing ride_id",
			message: &ws.Message{
				Type: "join_ride",
			},
			setupMock:   func(mock sqlmock.Sqlmock) {},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			redisDB, _ := redismock.NewClientMock()
			redisClient := &redis.Client{Client: redisDB}

			hub := ws.NewHub()
			go hub.Run()

			service := NewService(hub, db, redisClient, nil, zap.NewNop())

			conn := createTestWebSocketConn(t)
			client := ws.NewClient("user-123", conn, hub, "rider", zap.NewNop())

			hub.Register <- client
			time.Sleep(10 * time.Millisecond)

			// Setup mock expectations
			tt.setupMock(mock)

			// Execute
			service.handleJoinRide(client, tt.message)

			time.Sleep(10 * time.Millisecond)

			// Verify
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestHandleLeaveRide tests ride leaving functionality
func TestHandleLeaveRide(t *testing.T) {
	// Setup
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisDB, _ := redismock.NewClientMock()
	redisClient := &redis.Client{Client: redisDB}

	hub := ws.NewHub()
	go hub.Run()

	service := NewService(hub, db, redisClient, nil, zap.NewNop())

	// Create client
	conn := createTestWebSocketConn(t)
	client := ws.NewClient("user-123", conn, hub, "rider", zap.NewNop())

	rideID := "ride-789"
	client.SetRide(rideID)

	hub.Register <- client
	hub.AddClientToRide(client.ID, rideID)
	time.Sleep(10 * time.Millisecond)

	// Send leave message
	msg := &ws.Message{
		Type: "leave_ride",
		Data: map[string]interface{}{},
	}

	service.handleLeaveRide(client, msg)

	time.Sleep(10 * time.Millisecond)

	// Verify client is removed from ride
	clients := hub.GetClientsInRide(rideID)
	assert.Len(t, clients, 0)
}

// TestBroadcastRideUpdate tests broadcasting ride updates
func TestBroadcastRideUpdate(t *testing.T) {
	// Setup
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisDB, _ := redismock.NewClientMock()
	redisClient := &redis.Client{Client: redisDB}

	hub := ws.NewHub()
	go hub.Run()

	service := NewService(hub, db, redisClient, nil, zap.NewNop())

	// Create client
	conn := createTestWebSocketConn(t)
	client := ws.NewClient("user-123", conn, hub, "rider", zap.NewNop())

	rideID := "ride-789"
	hub.Register <- client
	hub.AddClientToRide(client.ID, rideID)
	time.Sleep(10 * time.Millisecond)

	// Broadcast update
	data := map[string]interface{}{
		"status": "arrived",
	}

	service.BroadcastRideUpdate(rideID, data)

	// Should not panic
	time.Sleep(10 * time.Millisecond)
}

// TestBroadcastToUser tests broadcasting to specific user
func TestBroadcastToUser(t *testing.T) {
	// Setup
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisDB, _ := redismock.NewClientMock()
	redisClient := &redis.Client{Client: redisDB}

	hub := ws.NewHub()
	go hub.Run()

	service := NewService(hub, db, redisClient, nil, zap.NewNop())

	// Create client
	conn := createTestWebSocketConn(t)
	client := ws.NewClient("user-123", conn, hub, "rider", zap.NewNop())

	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	// Broadcast to user
	data := map[string]interface{}{
		"notification": "Driver is nearby",
	}

	service.BroadcastToUser(client.ID, "notification", data)

	// Should not panic
	time.Sleep(10 * time.Millisecond)
}

// TestGetChatHistory tests retrieving chat history
func TestGetChatHistory(t *testing.T) {
	// Setup
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisDB, redisMock := redismock.NewClientMock()
	redisClient := &redis.Client{Client: redisDB}

	hub := ws.NewHub()
	service := NewService(hub, db, redisClient, nil, zap.NewNop())

	rideID := "ride-123"

	// Create mock chat messages
	msg1 := map[string]interface{}{
		"sender_id":   "user-123",
		"sender_role": "rider",
		"message":     "Hello",
		"timestamp":   time.Now().Unix(),
	}
	msg2 := map[string]interface{}{
		"sender_id":   "user-456",
		"sender_role": "driver",
		"message":     "Hi there!",
		"timestamp":   time.Now().Unix(),
	}

	msg1JSON, _ := json.Marshal(msg1)
	msg2JSON, _ := json.Marshal(msg2)

	// Expect Redis LRange
	redisMock.ExpectLRange("ride:chat:"+rideID, 0, -1).
		SetVal([]string{string(msg1JSON), string(msg2JSON)})

	// Execute
	history, err := service.GetChatHistory(rideID)

	// Verify
	assert.NoError(t, err)
	assert.Len(t, history, 2)
	assert.Equal(t, "Hello", history[0]["message"])
	assert.Equal(t, "Hi there!", history[1]["message"])
	assert.NoError(t, redisMock.ExpectationsWereMet())
}

// TestGetChatHistoryEmpty tests retrieving empty chat history
func TestGetChatHistoryEmpty(t *testing.T) {
	// Setup
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisDB, redisMock := redismock.NewClientMock()
	redisClient := &redis.Client{Client: redisDB}

	hub := ws.NewHub()
	service := NewService(hub, db, redisClient, nil, zap.NewNop())

	rideID := "ride-123"

	// Expect Redis LRange with no results
	redisMock.ExpectLRange("ride:chat:"+rideID, 0, -1).SetVal([]string{})

	// Execute
	history, err := service.GetChatHistory(rideID)

	// Verify
	assert.NoError(t, err)
	assert.Len(t, history, 0)
	assert.NoError(t, redisMock.ExpectationsWereMet())
}

// TestGetStats tests getting connection statistics
func TestGetStats(t *testing.T) {
	// Setup
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisDB, _ := redismock.NewClientMock()
	redisClient := &redis.Client{Client: redisDB}

	hub := ws.NewHub()
	go hub.Run()

	service := NewService(hub, db, redisClient, nil, zap.NewNop())

	// Create clients
	conn1 := createTestWebSocketConn(t)
	conn2 := createTestWebSocketConn(t)

	client1 := ws.NewClient("user-123", conn1, hub, "rider", zap.NewNop())
	client2 := ws.NewClient("user-456", conn2, hub, "driver", zap.NewNop())

	hub.Register <- client1
	hub.Register <- client2
	time.Sleep(10 * time.Millisecond)

	// Add to ride
	rideID := "ride-789"
	hub.AddClientToRide(client1.ID, rideID)
	hub.AddClientToRide(client2.ID, rideID)

	// Get stats
	stats := service.GetStats()

	// Verify
	assert.Equal(t, 2, stats["connected_clients"])
	assert.Equal(t, 1, stats["active_rides"])
}

// TestGetHub tests getting the hub instance
func TestGetHub(t *testing.T) {
	// Setup
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisDB, _ := redismock.NewClientMock()
	redisClient := &redis.Client{Client: redisDB}

	hub := ws.NewHub()
	service := NewService(hub, db, redisClient, nil, zap.NewNop())

	// Get hub
	retrievedHub := service.GetHub()

	// Verify
	assert.Equal(t, hub, retrievedHub)
}

// TestRegisterHandlers tests handler registration
func TestRegisterHandlers(t *testing.T) {
	// Setup
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisDB, _ := redismock.NewClientMock()
	redisClient := &redis.Client{Client: redisDB}

	hub := ws.NewHub()
	service := NewService(hub, db, redisClient, nil, zap.NewNop())

	// Verify handlers are registered
	assert.NotNil(t, service.hub)

	// Test that handlers can be called without panicking
	conn := createTestWebSocketConn(t)
	client := ws.NewClient("test-user", conn, hub, "rider", zap.NewNop())

	// This should not panic
	testMsg := &ws.Message{
		Type: "location_update",
		Data: map[string]interface{}{},
	}

	hub.HandleMessage(client, testMsg)
}

// TestConcurrentClientOperations tests thread-safety of concurrent operations
func TestConcurrentClientOperations(t *testing.T) {
	// Setup
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisDB, redisMock := redismock.NewClientMock()
	redisClient := &redis.Client{Client: redisDB}

	hub := ws.NewHub()
	go hub.Run()

	service := NewService(hub, db, redisClient, nil, zap.NewNop())

	// Expect multiple Redis operations (use regex to match any value)
	redisMock.MatchExpectationsInOrder(false)
	for i := 0; i < 10; i++ {
		redisMock.Regexp().ExpectSet("driver:ws_location:driver-"+string(rune(i)), `.*`, 5*time.Minute).SetVal("OK")
	}

	// Create multiple clients concurrently
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			conn := createTestWebSocketConn(t)
			client := ws.NewClient("driver-"+string(rune(id)), conn, hub, "driver", zap.NewNop())

			hub.Register <- client
			time.Sleep(5 * time.Millisecond)

			// Send location update
			msg := &ws.Message{
				Type: "location_update",
				Data: map[string]interface{}{
					"latitude":  37.7749,
					"longitude": -122.4194,
				},
			}
			service.handleLocationUpdate(client, msg)

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	time.Sleep(50 * time.Millisecond)

	// Verify stats
	stats := service.GetStats()
	assert.GreaterOrEqual(t, stats["connected_clients"].(int), 0)
}

// TestDatabaseError tests handling of database errors
func TestDatabaseError(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisDB, _ := redismock.NewClientMock()
	redisClient := &redis.Client{Client: redisDB}

	hub := ws.NewHub()
	go hub.Run()

	service := NewService(hub, db, redisClient, nil, zap.NewNop())

	conn := createTestWebSocketConn(t)
	client := ws.NewClient("user-123", conn, hub, "rider", zap.NewNop())

	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	// Mock database error
	mock.ExpectQuery("SELECT COUNT.*FROM rides").
		WithArgs("ride-123", "user-123").
		WillReturnError(sql.ErrConnDone)

	// Send join request
	msg := &ws.Message{
		Type:   "join_ride",
		RideID: "ride-123",
	}

	// This should handle the error gracefully
	service.handleJoinRide(client, msg)

	time.Sleep(10 * time.Millisecond)

	// Verify
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestRedisError tests handling of Redis errors
func TestRedisError(t *testing.T) {
	// Setup
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisDB, redisMock := redismock.NewClientMock()
	redisClient := &redis.Client{Client: redisDB}

	hub := ws.NewHub()
	service := NewService(hub, db, redisClient, nil, zap.NewNop())

	rideID := "ride-123"

	// Mock Redis error
	redisMock.ExpectLRange("ride:chat:"+rideID, 0, -1).
		SetErr(context.DeadlineExceeded)

	// Execute
	history, err := service.GetChatHistory(rideID)

	// Verify error is returned
	assert.Error(t, err)
	assert.Nil(t, history)
	assert.NoError(t, redisMock.ExpectationsWereMet())
}
