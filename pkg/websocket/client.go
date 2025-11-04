package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512KB
)

// Message represents a WebSocket message
type Message struct {
	Type      string                 `json:"type"`       // Message type (location, status, chat, etc.)
	RideID    string                 `json:"ride_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// Client represents a WebSocket client connection
type Client struct {
	ID     string          // Unique client identifier (user ID)
	RideID string          // Current ride ID (if in a ride)
	Role   string          // "rider" or "driver"
	Conn   *websocket.Conn // WebSocket connection
	Send   chan *Message   // Buffered channel of outbound messages
	Hub    *Hub            // Reference to hub
	mu     sync.RWMutex    // Protects concurrent access
}

// NewClient creates a new WebSocket client
func NewClient(id string, conn *websocket.Conn, hub *Hub, role string) *Client {
	return &Client{
		ID:   id,
		Conn: conn,
		Send: make(chan *Message, 256),
		Hub:  hub,
		Role: role,
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var msg Message
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		msg.Timestamp = time.Now()
		msg.UserID = c.ID

		// Route message to appropriate handler
		c.Hub.HandleMessage(c, &msg)
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.Conn.WriteJSON(message)
			if err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendMessage sends a message to the client
func (c *Client) SendMessage(msg *Message) {
	select {
	case c.Send <- msg:
	default:
		log.Printf("Client %s channel full, closing connection", c.ID)
		close(c.Send)
		c.Hub.Unregister <- c
	}
}

// SetRide associates the client with a ride
func (c *Client) SetRide(rideID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.RideID = rideID
}

// GetRide returns the current ride ID
func (c *Client) GetRide() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.RideID
}

// MarshalJSON custom JSON marshaling
func (m *Message) MarshalJSON() ([]byte, error) {
	type Alias Message
	return json.Marshal(&struct {
		Timestamp string `json:"timestamp"`
		*Alias
	}{
		Timestamp: m.Timestamp.Format(time.RFC3339),
		Alias:     (*Alias)(m),
	})
}

// UnmarshalJSON custom JSON unmarshaling
func (m *Message) UnmarshalJSON(data []byte) error {
	type Alias Message
	aux := &struct {
		Timestamp string `json:"timestamp"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Timestamp != "" {
		t, err := time.Parse(time.RFC3339, aux.Timestamp)
		if err != nil {
			return err
		}
		m.Timestamp = t
	}

	return nil
}
