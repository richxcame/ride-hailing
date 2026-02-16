package websocket

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestNewClient tests client creation
func TestNewClient(t *testing.T) {
	hub := NewHub()
	conn := createTestWebSocketConn(t)

	client := NewClient("user-123", conn, hub, "rider", zap.NewNop())

	assert.NotNil(t, client)
	assert.Equal(t, "user-123", client.ID)
	assert.Equal(t, "rider", client.Role)
	assert.Equal(t, hub, client.Hub)
	assert.NotNil(t, client.Send)
	assert.Equal(t, "", client.RideID)
}

// TestClientSetRide tests setting ride ID
func TestClientSetRide(t *testing.T) {
	hub := NewHub()
	conn := createTestWebSocketConn(t)
	client := NewClient("user-123", conn, hub, "rider", zap.NewNop())

	assert.Equal(t, "", client.GetRide())

	rideID := "ride-789"
	client.SetRide(rideID)

	assert.Equal(t, rideID, client.GetRide())
}

// TestClientGetRide tests getting ride ID
func TestClientGetRide(t *testing.T) {
	hub := NewHub()
	conn := createTestWebSocketConn(t)
	client := NewClient("user-123", conn, hub, "rider", zap.NewNop())

	// Initially empty
	assert.Equal(t, "", client.GetRide())

	// After setting
	client.SetRide("ride-123")
	assert.Equal(t, "ride-123", client.GetRide())

	// Clear ride
	client.SetRide("")
	assert.Equal(t, "", client.GetRide())
}

// TestClientSendMessage tests sending message to client
func TestClientSendMessage(t *testing.T) {
	hub := NewHub()
	conn := createTestWebSocketConn(t)
	client := NewClient("user-123", conn, hub, "rider", zap.NewNop())

	msg := &Message{
		Type: "test",
		Data: map[string]interface{}{
			"key": "value",
		},
		Timestamp: time.Now(),
	}

	client.SendMessage(msg)

	// Message should be in channel
	select {
	case receivedMsg := <-client.Send:
		assert.Equal(t, msg.Type, receivedMsg.Type)
		assert.Equal(t, "value", receivedMsg.Data["key"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Message not received in channel")
	}
}

// TestClientSendMessageChannelFull tests handling of full send channel
func TestClientSendMessageChannelFull(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	conn := createTestWebSocketConn(t)
	client := NewClient("user-123", conn, hub, "rider", zap.NewNop())

	// Use small channel
	client.Send = make(chan *Message, 2)

	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	// Fill channel
	for i := 0; i < 2; i++ {
		msg := &Message{
			Type: "test",
			Data: map[string]interface{}{
				"count": i,
			},
		}
		client.SendMessage(msg)
	}

	// This should trigger channel close due to overflow
	msg := &Message{
		Type: "overflow",
		Data: map[string]interface{}{},
	}
	client.SendMessage(msg)

	time.Sleep(10 * time.Millisecond)
}

// TestClientConcurrentRideAccess tests thread-safe ride ID access
func TestClientConcurrentRideAccess(t *testing.T) {
	hub := NewHub()
	conn := createTestWebSocketConn(t)
	client := NewClient("user-123", conn, hub, "rider", zap.NewNop())

	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(id int) {
			client.SetRide("ride-" + string(rune(id)))
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			_ = client.GetRide()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Should not panic or race
}

// TestMessageMarshalJSON tests custom JSON marshaling
func TestMessageMarshalJSON(t *testing.T) {
	msg := &Message{
		Type:   "test_type",
		RideID: "ride-123",
		UserID: "user-456",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: map[string]interface{}{
			"key": "value",
		},
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, "test_type", result["type"])
	assert.Equal(t, "ride-123", result["ride_id"])
	assert.Equal(t, "user-456", result["user_id"])
	assert.Equal(t, "2024-01-01T12:00:00Z", result["timestamp"])

	dataMap := result["data"].(map[string]interface{})
	assert.Equal(t, "value", dataMap["key"])
}

// TestMessageUnmarshalJSON tests custom JSON unmarshaling
func TestMessageUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"type": "test_type",
		"ride_id": "ride-123",
		"user_id": "user-456",
		"timestamp": "2024-01-01T12:00:00Z",
		"data": {
			"key": "value"
		}
	}`

	var msg Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	require.NoError(t, err)

	assert.Equal(t, "test_type", msg.Type)
	assert.Equal(t, "ride-123", msg.RideID)
	assert.Equal(t, "user-456", msg.UserID)
	assert.Equal(t, "value", msg.Data["key"])

	expectedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, expectedTime, msg.Timestamp)
}

// TestMessageUnmarshalJSONInvalidTimestamp tests handling invalid timestamp
func TestMessageUnmarshalJSONInvalidTimestamp(t *testing.T) {
	jsonData := `{
		"type": "test",
		"timestamp": "invalid-timestamp",
		"data": {}
	}`

	var msg Message
	err := json.Unmarshal([]byte(jsonData), &msg)

	assert.Error(t, err)
}

// TestMessageUnmarshalJSONEmptyTimestamp tests handling empty timestamp
func TestMessageUnmarshalJSONEmptyTimestamp(t *testing.T) {
	jsonData := `{
		"type": "test",
		"data": {}
	}`

	var msg Message
	err := json.Unmarshal([]byte(jsonData), &msg)

	require.NoError(t, err)
	assert.Equal(t, "test", msg.Type)
}

// TestMessageMarshalUnmarshalRoundTrip tests full round trip
func TestMessageMarshalUnmarshalRoundTrip(t *testing.T) {
	original := &Message{
		Type:      "location_update",
		RideID:    "ride-123",
		UserID:    "driver-456",
		Timestamp: time.Now().Round(time.Second), // Round to avoid nanosecond precision issues
		Data: map[string]interface{}{
			"latitude":  37.7749,
			"longitude": -122.4194,
			"speed":     50.5,
		},
	}

	// Marshal
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal
	var decoded Message
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.Type, decoded.Type)
	assert.Equal(t, original.RideID, decoded.RideID)
	assert.Equal(t, original.UserID, decoded.UserID)
	assert.Equal(t, original.Timestamp.Unix(), decoded.Timestamp.Unix())
	assert.Equal(t, original.Data["latitude"], decoded.Data["latitude"])
	assert.Equal(t, original.Data["longitude"], decoded.Data["longitude"])
	assert.Equal(t, original.Data["speed"], decoded.Data["speed"])
}

// TestMessageWithComplexData tests marshaling with complex nested data
func TestMessageWithComplexData(t *testing.T) {
	msg := &Message{
		Type:      "ride_status",
		RideID:    "ride-123",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"status": "in_progress",
			"driver": map[string]interface{}{
				"id":   "driver-456",
				"name": "John Doe",
				"rating": 4.8,
			},
			"location": map[string]interface{}{
				"latitude": 37.7749,
				"longitude": -122.4194,
			},
			"timestamps": []interface{}{
				"2024-01-01T12:00:00Z",
				"2024-01-01T12:05:00Z",
			},
		},
	}

	// Marshal
	data, err := json.Marshal(msg)
	require.NoError(t, err)

	// Unmarshal
	var decoded Message
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Verify complex data
	assert.Equal(t, "in_progress", decoded.Data["status"])

	driver := decoded.Data["driver"].(map[string]interface{})
	assert.Equal(t, "driver-456", driver["id"])
	assert.Equal(t, "John Doe", driver["name"])
	assert.Equal(t, 4.8, driver["rating"])

	location := decoded.Data["location"].(map[string]interface{})
	assert.Equal(t, 37.7749, location["latitude"])
	assert.Equal(t, -122.4194, location["longitude"])

	timestamps := decoded.Data["timestamps"].([]interface{})
	assert.Len(t, timestamps, 2)
}

// TestClientChannelBuffering tests send channel buffering
func TestClientChannelBuffering(t *testing.T) {
	hub := NewHub()
	conn := createTestWebSocketConn(t)
	client := NewClient("user-123", conn, hub, "rider", zap.NewNop())

	// Default channel should have 256 capacity
	assert.Equal(t, 256, cap(client.Send))

	// Should be able to send 256 messages without blocking
	for i := 0; i < 256; i++ {
		msg := &Message{
			Type: "test",
			Data: map[string]interface{}{
				"count": i,
			},
		}
		client.SendMessage(msg)
	}

	// Verify all messages are in channel
	assert.Equal(t, 256, len(client.Send))
}

// TestMultipleClients tests handling multiple concurrent clients
func TestMultipleClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	numClients := 20
	clients := make([]*Client, numClients)

	// Create multiple clients
	for i := 0; i < numClients; i++ {
		conn := createTestWebSocketConn(t)
		client := NewClient("user-"+string(rune(i)), conn, hub, "rider", zap.NewNop())
		clients[i] = client

		hub.Register <- client
	}

	time.Sleep(20 * time.Millisecond)

	// Send message to each client
	for i, client := range clients {
		msg := &Message{
			Type: "personal",
			Data: map[string]interface{}{
				"id": i,
			},
		}
		client.SendMessage(msg)
	}

	// Each client should receive their message
	for i, client := range clients {
		select {
		case msg := <-client.Send:
			assert.Equal(t, "personal", msg.Type)
			assert.Equal(t, i, msg.Data["id"])
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Client %d did not receive message", i)
		}
	}
}

// TestClientRoleTypes tests different client roles
func TestClientRoleTypes(t *testing.T) {
	hub := NewHub()
	conn := createTestWebSocketConn(t)

	roles := []string{"rider", "driver", "admin"}

	for _, role := range roles {
		client := NewClient("user-"+role, conn, hub, role, zap.NewNop())
		assert.Equal(t, role, client.Role)
	}
}

// TestMessageTimestampPrecision tests timestamp precision
func TestMessageTimestampPrecision(t *testing.T) {
	now := time.Now()

	msg := &Message{
		Type:      "test",
		Timestamp: now,
		Data:      map[string]interface{}{},
	}

	// Marshal and unmarshal
	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded Message
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Timestamps should be equal to the second (RFC3339 precision)
	assert.Equal(t, now.Unix(), decoded.Timestamp.Unix())
}

// TestMessageDataTypes tests different data types in message data
func TestMessageDataTypes(t *testing.T) {
	msg := &Message{
		Type:      "test",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"string":  "value",
			"int":     42,
			"float":   3.14,
			"bool":    true,
			"null":    nil,
			"array":   []interface{}{1, 2, 3},
			"object":  map[string]interface{}{"nested": "value"},
		},
	}

	// Marshal
	data, err := json.Marshal(msg)
	require.NoError(t, err)

	// Unmarshal
	var decoded Message
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Verify all types
	assert.Equal(t, "value", decoded.Data["string"])
	assert.Equal(t, float64(42), decoded.Data["int"]) // JSON numbers are float64
	assert.Equal(t, 3.14, decoded.Data["float"])
	assert.Equal(t, true, decoded.Data["bool"])
	assert.Nil(t, decoded.Data["null"])

	array := decoded.Data["array"].([]interface{})
	assert.Len(t, array, 3)

	object := decoded.Data["object"].(map[string]interface{})
	assert.Equal(t, "value", object["nested"])
}

// TestClientIDUniqueness tests that each client has unique ID
func TestClientIDUniqueness(t *testing.T) {
	hub := NewHub()

	ids := make(map[string]bool)
	numClients := 100

	for i := 0; i < numClients; i++ {
		conn := createTestWebSocketConn(t)
		client := NewClient("user-"+string(rune(i)), conn, hub, "rider", zap.NewNop())

		// Check ID is not duplicate
		assert.False(t, ids[client.ID], "Duplicate client ID: %s", client.ID)
		ids[client.ID] = true
	}

	assert.Len(t, ids, numClients)
}

// TestMessageOptionalFields tests handling of optional message fields
func TestMessageOptionalFields(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantType string
	}{
		{
			name: "All fields present",
			jsonData: `{
				"type": "test",
				"ride_id": "ride-123",
				"user_id": "user-456",
				"timestamp": "2024-01-01T12:00:00Z",
				"data": {"key": "value"}
			}`,
			wantType: "test",
		},
		{
			name: "Only required fields",
			jsonData: `{
				"type": "test",
				"data": {"key": "value"}
			}`,
			wantType: "test",
		},
		{
			name: "With ride_id only",
			jsonData: `{
				"type": "ride_update",
				"ride_id": "ride-123",
				"data": {}
			}`,
			wantType: "ride_update",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg Message
			err := json.Unmarshal([]byte(tt.jsonData), &msg)
			require.NoError(t, err)
			assert.Equal(t, tt.wantType, msg.Type)
		})
	}
}
