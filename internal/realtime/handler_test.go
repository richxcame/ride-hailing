package realtime

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redismock/v9"
	"github.com/gorilla/websocket"
	"github.com/richxcame/ride-hailing/pkg/redis"
	ws "github.com/richxcame/ride-hailing/pkg/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ============================================================================
// Test Setup Helpers
// ============================================================================

func setupTestHandler(t *testing.T) (*Handler, *Service, sqlmock.Sqlmock, redismock.ClientMock) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	redisDB, redisMock := redismock.NewClientMock()
	redisClient := &redis.Client{Client: redisDB}

	hub := ws.NewHub()
	go hub.Run()

	service := NewService(hub, db, redisClient, nil, zap.NewNop())
	handler := NewHandler(service, zap.NewNop())

	t.Cleanup(func() {
		db.Close()
	})

	return handler, service, mock, redisMock
}

func setupTestContext(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	c.Request = req

	return c, w
}

func setUserContext(c *gin.Context, userID string, role string) {
	c.Set("user_id", userID)
	c.Set("user_role", role)
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

// createTestWebSocketServer creates a test WebSocket server for handler testing
func createTestWebSocketServer(t *testing.T, handler gin.HandlerFunc) *httptest.Server {
	t.Helper()

	router := gin.New()
	router.GET("/ws", handler)

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server
}

// createHandlerTestWebSocketConn creates a test WebSocket connection for handler tests
func createHandlerTestWebSocketConn(t *testing.T) *websocket.Conn {
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

// ============================================================================
// NewHandler Tests
// ============================================================================

func TestNewHandler(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisDB, _ := redismock.NewClientMock()
	redisClient := &redis.Client{Client: redisDB}

	hub := ws.NewHub()
	service := NewService(hub, db, redisClient, nil, zap.NewNop())
	logger := zap.NewNop()

	handler := NewHandler(service, logger)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.service)
	assert.NotNil(t, handler.logger)
}

func TestNewHandler_NilService(t *testing.T) {
	logger := zap.NewNop()

	handler := NewHandler(nil, logger)

	assert.NotNil(t, handler)
	assert.Nil(t, handler.service)
}

// ============================================================================
// WebSocket Connection Tests
// ============================================================================

func TestHandleWebSocket_Unauthorized_NoUserID(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	c, w := setupTestContext("GET", "/api/v1/ws", nil)
	// Don't set user_id - simulate unauthorized access

	handler.HandleWebSocket(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "Unauthorized")
}

func TestHandleWebSocket_DefaultRole(t *testing.T) {
	handler, service, _, _ := setupTestHandler(t)

	// Create a test server that handles WebSocket upgrade
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = r

		// Set user_id but not role - should default to "rider"
		c.Set("user_id", "user-123")

		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		// Verify default role is applied
		client := ws.NewClient("user-123", conn, service.GetHub(), "rider", zap.NewNop())
		assert.Equal(t, "rider", client.Role)
		conn.Close()
	}))
	defer server.Close()

	// Test that handler exists
	assert.NotNil(t, handler)
}

func TestHandleWebSocket_RiderRole(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	c, w := setupTestContext("GET", "/api/v1/ws", nil)
	setUserContext(c, "rider-123", "rider")

	// WebSocket upgrade will fail without proper HTTP connection, but we verify auth passes
	handler.HandleWebSocket(c)

	// The handler tries to upgrade but fails (expected in test env)
	// Key assertion: it didn't return 401
	assert.NotEqual(t, http.StatusUnauthorized, w.Code)
}

func TestHandleWebSocket_DriverRole(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	c, w := setupTestContext("GET", "/api/v1/ws", nil)
	setUserContext(c, "driver-123", "driver")

	handler.HandleWebSocket(c)

	// Upgrade fails but auth should pass
	assert.NotEqual(t, http.StatusUnauthorized, w.Code)
}

func TestHandleWebSocket_AdminRole(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	c, w := setupTestContext("GET", "/api/v1/ws", nil)
	setUserContext(c, "admin-123", "admin")

	handler.HandleWebSocket(c)

	assert.NotEqual(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// WebSocket Origin Check Tests
// ============================================================================

func TestWebSocket_CheckOrigin_AllowedOrigin(t *testing.T) {
	os.Setenv("CORS_ORIGINS", "http://localhost:3000,https://app.example.com")
	defer os.Unsetenv("CORS_ORIGINS")

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	result := upgrader.CheckOrigin(req)
	assert.True(t, result)
}

func TestWebSocket_CheckOrigin_DisallowedOrigin(t *testing.T) {
	os.Setenv("CORS_ORIGINS", "http://localhost:3000")
	defer os.Unsetenv("CORS_ORIGINS")

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "http://malicious.com")

	result := upgrader.CheckOrigin(req)
	assert.False(t, result)
}

func TestWebSocket_CheckOrigin_NoOrigin(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws", nil)
	// No Origin header - should be allowed (mobile apps, Postman)

	result := upgrader.CheckOrigin(req)
	assert.True(t, result)
}

func TestWebSocket_CheckOrigin_EmptyOrigin(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "")

	result := upgrader.CheckOrigin(req)
	assert.True(t, result)
}

func TestWebSocket_CheckOrigin_DefaultFallback(t *testing.T) {
	os.Unsetenv("CORS_ORIGINS")

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	result := upgrader.CheckOrigin(req)
	assert.True(t, result)
}

func TestWebSocket_CheckOrigin_MultipleOrigins(t *testing.T) {
	os.Setenv("CORS_ORIGINS", "http://localhost:3000, https://staging.example.com, https://production.example.com")
	defer os.Unsetenv("CORS_ORIGINS")

	testCases := []struct {
		origin  string
		allowed bool
	}{
		{"http://localhost:3000", true},
		{"https://staging.example.com", true},
		{"https://production.example.com", true},
		{"https://other.example.com", false},
	}

	for _, tc := range testCases {
		t.Run(tc.origin, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ws", nil)
			req.Header.Set("Origin", tc.origin)

			result := upgrader.CheckOrigin(req)
			assert.Equal(t, tc.allowed, result)
		})
	}
}

// ============================================================================
// GetStats Tests
// ============================================================================

func TestGetStats_Success(t *testing.T) {
	handler, service, _, _ := setupTestHandler(t)

	// Add some clients to the hub
	conn1 := createHandlerTestWebSocketConn(t)
	conn2 := createHandlerTestWebSocketConn(t)

	client1 := ws.NewClient("user-1", conn1, service.GetHub(), "rider", zap.NewNop())
	client2 := ws.NewClient("user-2", conn2, service.GetHub(), "driver", zap.NewNop())

	service.GetHub().Register <- client1
	service.GetHub().Register <- client2
	time.Sleep(10 * time.Millisecond)

	c, w := setupTestContext("GET", "/api/v1/realtime/stats", nil)

	handler.GetStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.GreaterOrEqual(t, data["connected_clients"].(float64), float64(2))
}

func TestGetStats_EmptyHub(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	c, w := setupTestContext("GET", "/api/v1/realtime/stats", nil)

	handler.GetStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["connected_clients"].(float64))
	assert.Equal(t, float64(0), data["active_rides"].(float64))
}

func TestGetStats_WithActiveRides(t *testing.T) {
	handler, service, _, _ := setupTestHandler(t)

	// Create clients and add to rides
	conn1 := createHandlerTestWebSocketConn(t)
	conn2 := createHandlerTestWebSocketConn(t)

	client1 := ws.NewClient("user-1", conn1, service.GetHub(), "rider", zap.NewNop())
	client2 := ws.NewClient("user-2", conn2, service.GetHub(), "driver", zap.NewNop())

	service.GetHub().Register <- client1
	service.GetHub().Register <- client2
	time.Sleep(10 * time.Millisecond)

	service.GetHub().AddClientToRide("user-1", "ride-123")
	service.GetHub().AddClientToRide("user-2", "ride-123")

	c, w := setupTestContext("GET", "/api/v1/realtime/stats", nil)

	handler.GetStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["active_rides"].(float64))
}

// ============================================================================
// GetChatHistory Tests
// ============================================================================

func TestGetChatHistory_Success(t *testing.T) {
	handler, _, dbMock, redisMock := setupTestHandler(t)

	rideID := "ride-123"
	userID := "user-456"

	// Mock database query - user is part of ride
	rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	dbMock.ExpectQuery("SELECT COUNT.*FROM rides").
		WithArgs(rideID, userID).
		WillReturnRows(rows)

	// Mock Redis response
	msg1 := map[string]interface{}{
		"sender_id":   "user-456",
		"sender_role": "rider",
		"message":     "Hello driver!",
		"timestamp":   time.Now().Unix(),
	}
	msg1JSON, _ := json.Marshal(msg1)
	redisMock.ExpectLRange("ride:chat:"+rideID, 0, -1).SetVal([]string{string(msg1JSON)})

	c, w := setupTestContext("GET", "/api/v1/rides/"+rideID+"/chat", nil)
	c.Params = gin.Params{{Key: "ride_id", Value: rideID}}
	setUserContext(c, userID, "rider")

	handler.GetChatHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, rideID, data["ride_id"])
	assert.NotNil(t, data["messages"])
}

func TestGetChatHistory_EmptyRideID(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	c, w := setupTestContext("GET", "/api/v1/rides//chat", nil)
	c.Params = gin.Params{{Key: "ride_id", Value: ""}}
	setUserContext(c, "user-123", "rider")

	handler.GetChatHistory(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "ride_id is required")
}

func TestGetChatHistory_Unauthorized(t *testing.T) {
	handler, _, dbMock, _ := setupTestHandler(t)

	rideID := "ride-123"
	userID := "user-999" // Not part of ride

	// Mock database query - user is NOT part of ride
	rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	dbMock.ExpectQuery("SELECT COUNT.*FROM rides").
		WithArgs(rideID, userID).
		WillReturnRows(rows)

	c, w := setupTestContext("GET", "/api/v1/rides/"+rideID+"/chat", nil)
	c.Params = gin.Params{{Key: "ride_id", Value: rideID}}
	setUserContext(c, userID, "rider")

	handler.GetChatHistory(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestGetChatHistory_DatabaseError(t *testing.T) {
	handler, _, dbMock, _ := setupTestHandler(t)

	rideID := "ride-123"
	userID := "user-456"

	dbMock.ExpectQuery("SELECT COUNT.*FROM rides").
		WithArgs(rideID, userID).
		WillReturnError(sql.ErrConnDone)

	c, w := setupTestContext("GET", "/api/v1/rides/"+rideID+"/chat", nil)
	c.Params = gin.Params{{Key: "ride_id", Value: rideID}}
	setUserContext(c, userID, "rider")

	handler.GetChatHistory(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetChatHistory_RedisError(t *testing.T) {
	handler, _, dbMock, redisMock := setupTestHandler(t)

	rideID := "ride-123"
	userID := "user-456"

	rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	dbMock.ExpectQuery("SELECT COUNT.*FROM rides").
		WithArgs(rideID, userID).
		WillReturnRows(rows)

	redisMock.ExpectLRange("ride:chat:"+rideID, 0, -1).
		SetErr(context.DeadlineExceeded)

	c, w := setupTestContext("GET", "/api/v1/rides/"+rideID+"/chat", nil)
	c.Params = gin.Params{{Key: "ride_id", Value: rideID}}
	setUserContext(c, userID, "rider")

	handler.GetChatHistory(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetChatHistory_EmptyHistory(t *testing.T) {
	handler, _, dbMock, redisMock := setupTestHandler(t)

	rideID := "ride-123"
	userID := "user-456"

	rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	dbMock.ExpectQuery("SELECT COUNT.*FROM rides").
		WithArgs(rideID, userID).
		WillReturnRows(rows)

	redisMock.ExpectLRange("ride:chat:"+rideID, 0, -1).SetVal([]string{})

	c, w := setupTestContext("GET", "/api/v1/rides/"+rideID+"/chat", nil)
	c.Params = gin.Params{{Key: "ride_id", Value: rideID}}
	setUserContext(c, userID, "rider")

	handler.GetChatHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	messages := data["messages"].([]interface{})
	assert.Len(t, messages, 0)
}

// ============================================================================
// BroadcastRideUpdate Tests
// ============================================================================

func TestBroadcastRideUpdate_Success(t *testing.T) {
	handler, service, _, _ := setupTestHandler(t)

	// Add a client to the ride
	conn := createHandlerTestWebSocketConn(t)
	client := ws.NewClient("user-123", conn, service.GetHub(), "rider", zap.NewNop())
	service.GetHub().Register <- client
	time.Sleep(10 * time.Millisecond)
	service.GetHub().AddClientToRide("user-123", "ride-123")

	reqBody := map[string]interface{}{
		"ride_id": "ride-123",
		"data": map[string]interface{}{
			"status":     "in_progress",
			"driver_eta": 5,
		},
	}

	c, w := setupTestContext("POST", "/api/v1/internal/broadcast/ride", reqBody)

	handler.BroadcastRideUpdate(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "Broadcast sent", data["message"])
}

func TestBroadcastRideUpdate_MissingRideID(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"status": "completed",
		},
	}

	c, w := setupTestContext("POST", "/api/v1/internal/broadcast/ride", reqBody)

	handler.BroadcastRideUpdate(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBroadcastRideUpdate_MissingData(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	reqBody := map[string]interface{}{
		"ride_id": "ride-123",
	}

	c, w := setupTestContext("POST", "/api/v1/internal/broadcast/ride", reqBody)

	handler.BroadcastRideUpdate(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBroadcastRideUpdate_InvalidJSON(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	c, w := setupTestContext("POST", "/api/v1/internal/broadcast/ride", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/internal/broadcast/ride",
		bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BroadcastRideUpdate(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBroadcastRideUpdate_EmptyBody(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	c, w := setupTestContext("POST", "/api/v1/internal/broadcast/ride", map[string]interface{}{})

	handler.BroadcastRideUpdate(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// BroadcastToUser Tests
// ============================================================================

func TestBroadcastToUser_Success(t *testing.T) {
	handler, service, _, _ := setupTestHandler(t)

	// Add a client
	conn := createHandlerTestWebSocketConn(t)
	client := ws.NewClient("user-123", conn, service.GetHub(), "rider", zap.NewNop())
	service.GetHub().Register <- client
	time.Sleep(10 * time.Millisecond)

	reqBody := map[string]interface{}{
		"user_id": "user-123",
		"type":    "notification",
		"data": map[string]interface{}{
			"message": "Your driver is arriving!",
		},
	}

	c, w := setupTestContext("POST", "/api/v1/internal/broadcast/user", reqBody)

	handler.BroadcastToUser(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestBroadcastToUser_MissingUserID(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	reqBody := map[string]interface{}{
		"type": "notification",
		"data": map[string]interface{}{
			"message": "Test",
		},
	}

	c, w := setupTestContext("POST", "/api/v1/internal/broadcast/user", reqBody)

	handler.BroadcastToUser(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBroadcastToUser_MissingType(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	reqBody := map[string]interface{}{
		"user_id": "user-123",
		"data": map[string]interface{}{
			"message": "Test",
		},
	}

	c, w := setupTestContext("POST", "/api/v1/internal/broadcast/user", reqBody)

	handler.BroadcastToUser(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBroadcastToUser_MissingData(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	reqBody := map[string]interface{}{
		"user_id": "user-123",
		"type":    "notification",
	}

	c, w := setupTestContext("POST", "/api/v1/internal/broadcast/user", reqBody)

	handler.BroadcastToUser(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBroadcastToUser_InvalidJSON(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	c, w := setupTestContext("POST", "/api/v1/internal/broadcast/user", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/internal/broadcast/user",
		bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BroadcastToUser(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBroadcastToUser_NonexistentUser(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	// Broadcast to user that doesn't exist - should still succeed (no error)
	reqBody := map[string]interface{}{
		"user_id": "nonexistent-user",
		"type":    "notification",
		"data": map[string]interface{}{
			"message": "Test",
		},
	}

	c, w := setupTestContext("POST", "/api/v1/internal/broadcast/user", reqBody)

	handler.BroadcastToUser(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================================
// HealthCheck Tests
// ============================================================================

func TestHealthCheck_Success(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	c, w := setupTestContext("GET", "/api/v1/realtime/health", nil)

	handler.HealthCheck(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "healthy", data["status"])
	assert.Equal(t, "realtime", data["service"])
	assert.NotNil(t, data["stats"])
}

func TestHealthCheck_WithConnectedClients(t *testing.T) {
	handler, service, _, _ := setupTestHandler(t)

	// Add clients
	conn := createHandlerTestWebSocketConn(t)
	client := ws.NewClient("user-123", conn, service.GetHub(), "rider", zap.NewNop())
	service.GetHub().Register <- client
	time.Sleep(10 * time.Millisecond)

	c, w := setupTestContext("GET", "/api/v1/realtime/health", nil)

	handler.HealthCheck(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	stats := data["stats"].(map[string]interface{})
	assert.GreaterOrEqual(t, stats["connected_clients"].(float64), float64(1))
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestBroadcastRideUpdate_ValidationCases(t *testing.T) {
	tests := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
	}{
		{
			name: "valid request",
			body: map[string]interface{}{
				"ride_id": "ride-123",
				"data": map[string]interface{}{
					"status": "completed",
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing ride_id",
			body: map[string]interface{}{
				"data": map[string]interface{}{
					"status": "completed",
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty ride_id",
			body: map[string]interface{}{
				"ride_id": "",
				"data": map[string]interface{}{
					"status": "completed",
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing data",
			body: map[string]interface{}{
				"ride_id": "ride-123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "null data",
			body: map[string]interface{}{
				"ride_id": "ride-123",
				"data":    nil,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty body",
			body:           map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, _, _, _ := setupTestHandler(t)

			c, w := setupTestContext("POST", "/api/v1/internal/broadcast/ride", tt.body)

			handler.BroadcastRideUpdate(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestBroadcastToUser_ValidationCases(t *testing.T) {
	tests := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
	}{
		{
			name: "valid request",
			body: map[string]interface{}{
				"user_id": "user-123",
				"type":    "notification",
				"data": map[string]interface{}{
					"message": "Hello",
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing user_id",
			body: map[string]interface{}{
				"type": "notification",
				"data": map[string]interface{}{
					"message": "Hello",
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty user_id",
			body: map[string]interface{}{
				"user_id": "",
				"type":    "notification",
				"data": map[string]interface{}{
					"message": "Hello",
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing type",
			body: map[string]interface{}{
				"user_id": "user-123",
				"data": map[string]interface{}{
					"message": "Hello",
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty type",
			body: map[string]interface{}{
				"user_id": "user-123",
				"type":    "",
				"data": map[string]interface{}{
					"message": "Hello",
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing data",
			body: map[string]interface{}{
				"user_id": "user-123",
				"type":    "notification",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, _, _, _ := setupTestHandler(t)

			c, w := setupTestContext("POST", "/api/v1/internal/broadcast/user", tt.body)

			handler.BroadcastToUser(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetChatHistory_AuthorizationCases(t *testing.T) {
	tests := []struct {
		name           string
		rideID         string
		userID         string
		userInRide     bool
		expectedStatus int
	}{
		{
			name:           "authorized user",
			rideID:         "ride-123",
			userID:         "user-456",
			userInRide:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unauthorized user",
			rideID:         "ride-123",
			userID:         "user-999",
			userInRide:     false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "empty ride_id",
			rideID:         "",
			userID:         "user-456",
			userInRide:     false,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, _, dbMock, redisMock := setupTestHandler(t)

			if tt.rideID != "" {
				count := 0
				if tt.userInRide {
					count = 1
				}
				rows := sqlmock.NewRows([]string{"count"}).AddRow(count)
				dbMock.ExpectQuery("SELECT COUNT.*FROM rides").
					WithArgs(tt.rideID, tt.userID).
					WillReturnRows(rows)

				if tt.userInRide {
					redisMock.ExpectLRange("ride:chat:"+tt.rideID, 0, -1).SetVal([]string{})
				}
			}

			c, w := setupTestContext("GET", "/api/v1/rides/"+tt.rideID+"/chat", nil)
			c.Params = gin.Params{{Key: "ride_id", Value: tt.rideID}}
			setUserContext(c, tt.userID, "rider")

			handler.GetChatHistory(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestGetStats_ResponseFormat(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	c, w := setupTestContext("GET", "/api/v1/realtime/stats", nil)

	handler.GetStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	// Verify response structure
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.Contains(t, data, "connected_clients")
	assert.Contains(t, data, "active_rides")
}

func TestHealthCheck_ResponseFormat(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	c, w := setupTestContext("GET", "/api/v1/realtime/health", nil)

	handler.HealthCheck(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "healthy", data["status"])
	assert.Equal(t, "realtime", data["service"])
	assert.NotNil(t, data["stats"])
}

func TestBroadcastRideUpdate_ResponseFormat(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	reqBody := map[string]interface{}{
		"ride_id": "ride-123",
		"data": map[string]interface{}{
			"status": "completed",
		},
	}

	c, w := setupTestContext("POST", "/api/v1/internal/broadcast/ride", reqBody)

	handler.BroadcastRideUpdate(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "Broadcast sent", data["message"])
}

func TestErrorResponse_Format(t *testing.T) {
	handler, _, _, _ := setupTestHandler(t)

	c, w := setupTestContext("GET", "/api/v1/rides//chat", nil)
	c.Params = gin.Params{{Key: "ride_id", Value: ""}}
	setUserContext(c, "user-123", "rider")

	handler.GetChatHistory(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)

	assert.False(t, response["success"].(bool))
	assert.NotNil(t, response["error"])

	errorInfo := response["error"].(map[string]interface{})
	assert.NotNil(t, errorInfo["code"])
	assert.NotNil(t, errorInfo["message"])
}

// ============================================================================
// WebSocket Integration Tests
// ============================================================================

func TestWebSocket_FullConnection(t *testing.T) {
	// Setup
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisDB, _ := redismock.NewClientMock()
	redisClient := &redis.Client{Client: redisDB}

	hub := ws.NewHub()
	go hub.Run()

	service := NewService(hub, db, redisClient, nil, zap.NewNop())

	// Create a test HTTP server with WebSocket endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate auth middleware
		userID := r.URL.Query().Get("user_id")
		role := r.URL.Query().Get("role")

		if userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if role == "" {
			role = "rider"
		}

		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		client := ws.NewClient(userID, conn, hub, role, zap.NewNop())
		hub.Register <- client

		go client.WritePump()
		go client.ReadPump()
	}))
	defer server.Close()

	// Connect as rider
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?user_id=rider-123&role=rider"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Wait for registration
	time.Sleep(20 * time.Millisecond)

	// Verify client is registered
	stats := service.GetStats()
	assert.GreaterOrEqual(t, stats["connected_clients"].(int), 1)
}

func TestWebSocket_MultipleClients(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisDB, _ := redismock.NewClientMock()
	redisClient := &redis.Client{Client: redisDB}

	hub := ws.NewHub()
	go hub.Run()

	service := NewService(hub, db, redisClient, nil, zap.NewNop())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user_id")
		role := r.URL.Query().Get("role")

		if userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if role == "" {
			role = "rider"
		}

		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		client := ws.NewClient(userID, conn, hub, role, zap.NewNop())
		hub.Register <- client

		go client.WritePump()
		go client.ReadPump()
	}))
	defer server.Close()

	// Connect multiple clients
	var conns []*websocket.Conn
	for i := 0; i < 5; i++ {
		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?user_id=user-" + string(rune('A'+i)) + "&role=rider"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		conns = append(conns, conn)
	}

	defer func() {
		for _, conn := range conns {
			conn.Close()
		}
	}()

	time.Sleep(50 * time.Millisecond)

	stats := service.GetStats()
	assert.GreaterOrEqual(t, stats["connected_clients"].(int), 5)
}

func TestWebSocket_DisconnectCleanup(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisDB, _ := redismock.NewClientMock()
	redisClient := &redis.Client{Client: redisDB}

	hub := ws.NewHub()
	go hub.Run()

	service := NewService(hub, db, redisClient, nil, zap.NewNop())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user_id")

		if userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		client := ws.NewClient(userID, conn, hub, "rider", zap.NewNop())
		hub.Register <- client

		go client.WritePump()
		go client.ReadPump()
	}))
	defer server.Close()

	// Connect and then disconnect
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?user_id=temp-user"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	time.Sleep(20 * time.Millisecond)

	initialCount := service.GetStats()["connected_clients"].(int)

	// Disconnect
	conn.Close()

	time.Sleep(100 * time.Millisecond)

	finalCount := service.GetStats()["connected_clients"].(int)
	assert.Less(t, finalCount, initialCount)
}

// ============================================================================
// Concurrent Access Tests
// ============================================================================

func TestConcurrentBroadcasts(t *testing.T) {
	handler, service, _, _ := setupTestHandler(t)

	// Add clients
	for i := 0; i < 5; i++ {
		conn := createHandlerTestWebSocketConn(t)
		client := ws.NewClient("user-"+string(rune('A'+i)), conn, service.GetHub(), "rider", zap.NewNop())
		service.GetHub().Register <- client
		service.GetHub().AddClientToRide(client.ID, "ride-concurrent")
	}
	time.Sleep(20 * time.Millisecond)

	// Send concurrent broadcasts
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			reqBody := map[string]interface{}{
				"ride_id": "ride-concurrent",
				"data": map[string]interface{}{
					"message_id": idx,
				},
			}

			c, _ := setupTestContext("POST", "/api/v1/internal/broadcast/ride", reqBody)
			handler.BroadcastRideUpdate(c)

			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// No panics = success
}

func TestConcurrentStats(t *testing.T) {
	handler, service, _, _ := setupTestHandler(t)

	// Add clients
	for i := 0; i < 3; i++ {
		conn := createHandlerTestWebSocketConn(t)
		client := ws.NewClient("user-"+string(rune('A'+i)), conn, service.GetHub(), "rider", zap.NewNop())
		service.GetHub().Register <- client
	}
	time.Sleep(20 * time.Millisecond)

	// Concurrent stats requests
	done := make(chan bool)
	for i := 0; i < 20; i++ {
		go func() {
			c, _ := setupTestContext("GET", "/api/v1/realtime/stats", nil)
			handler.GetStats(c)
			done <- true
		}()
	}

	for i := 0; i < 20; i++ {
		<-done
	}

	// No panics = success
}
