package websocket

import (
	"log"
	"sync"
)

// MessageHandler is a function that handles incoming messages
type MessageHandler func(*Client, *Message)

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients by user ID
	clients map[string]*Client

	// Clients grouped by ride ID
	rides map[string]map[string]*Client

	// Register requests from clients
	Register chan *Client

	// Unregister requests from clients
	Unregister chan *Client

	// Broadcast messages to specific users
	Broadcast chan *BroadcastMessage

	// Message handlers by message type
	handlers map[string]MessageHandler

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// BroadcastMessage represents a message to be broadcast
type BroadcastMessage struct {
	Target   string   // "user", "ride", "all"
	TargetID string   // User ID or Ride ID
	Message  *Message // Message to send
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		rides:      make(map[string]map[string]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan *BroadcastMessage, 256),
		handlers:   make(map[string]MessageHandler),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	log.Println("WebSocket Hub started")
	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)

		case client := <-h.Unregister:
			h.unregisterClient(client)

		case broadcast := <-h.Broadcast:
			h.broadcastMessage(broadcast)
		}
	}
}

// registerClient adds a client to the hub
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Remove existing client with same ID
	if existingClient, ok := h.clients[client.ID]; ok {
		close(existingClient.Send)
	}

	h.clients[client.ID] = client
	log.Printf("Client registered: %s (role: %s)", client.ID, client.Role)
}

// unregisterClient removes a client from the hub
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.ID]; ok {
		// Remove from clients map
		delete(h.clients, client.ID)

		// Remove from ride room if in one
		rideID := client.GetRide()
		if rideID != "" {
			if ride, ok := h.rides[rideID]; ok {
				delete(ride, client.ID)
				if len(ride) == 0 {
					delete(h.rides, rideID)
				}
			}
		}

		close(client.Send)
		log.Printf("Client unregistered: %s", client.ID)
	}
}

// broadcastMessage sends a message to target clients
func (h *Hub) broadcastMessage(broadcast *BroadcastMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	switch broadcast.Target {
	case "user":
		// Send to specific user
		if client, ok := h.clients[broadcast.TargetID]; ok {
			client.SendMessage(broadcast.Message)
		}

	case "ride":
		// Send to all clients in a ride
		if ride, ok := h.rides[broadcast.TargetID]; ok {
			for _, client := range ride {
				client.SendMessage(broadcast.Message)
			}
		}

	case "all":
		// Send to all connected clients
		for _, client := range h.clients {
			client.SendMessage(broadcast.Message)
		}
	}
}

// HandleMessage routes incoming messages to appropriate handlers
func (h *Hub) HandleMessage(client *Client, msg *Message) {
	h.mu.RLock()
	handler, exists := h.handlers[msg.Type]
	h.mu.RUnlock()

	if exists {
		handler(client, msg)
	} else {
		log.Printf("No handler for message type: %s", msg.Type)
	}
}

// RegisterHandler registers a message handler for a specific type
func (h *Hub) RegisterHandler(msgType string, handler MessageHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handlers[msgType] = handler
	log.Printf("Registered handler for message type: %s", msgType)
}

// AddClientToRide adds a client to a ride room
func (h *Hub) AddClientToRide(clientID, rideID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, ok := h.clients[clientID]
	if !ok {
		return
	}

	// Create ride room if doesn't exist
	if _, ok := h.rides[rideID]; !ok {
		h.rides[rideID] = make(map[string]*Client)
	}

	// Add client to ride room
	h.rides[rideID][clientID] = client
	client.SetRide(rideID)

	log.Printf("Client %s joined ride %s", clientID, rideID)
}

// RemoveClientFromRide removes a client from a ride room
func (h *Hub) RemoveClientFromRide(clientID, rideID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if ride, ok := h.rides[rideID]; ok {
		delete(ride, clientID)
		if len(ride) == 0 {
			delete(h.rides, rideID)
		}
	}

	if client, ok := h.clients[clientID]; ok {
		client.SetRide("")
	}

	log.Printf("Client %s left ride %s", clientID, rideID)
}

// SendToUser sends a message to a specific user
func (h *Hub) SendToUser(userID string, msg *Message) {
	h.Broadcast <- &BroadcastMessage{
		Target:   "user",
		TargetID: userID,
		Message:  msg,
	}
}

// SendToRide sends a message to all clients in a ride
func (h *Hub) SendToRide(rideID string, msg *Message) {
	h.Broadcast <- &BroadcastMessage{
		Target:   "ride",
		TargetID: rideID,
		Message:  msg,
	}
}

// SendToAll broadcasts a message to all connected clients
func (h *Hub) SendToAll(msg *Message) {
	h.Broadcast <- &BroadcastMessage{
		Target:  "all",
		Message: msg,
	}
}

// GetClient returns a client by ID
func (h *Hub) GetClient(clientID string) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	client, ok := h.clients[clientID]
	return client, ok
}

// GetClientsInRide returns all clients in a ride
func (h *Hub) GetClientsInRide(rideID string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := make([]*Client, 0)
	if ride, ok := h.rides[rideID]; ok {
		for _, client := range ride {
			clients = append(clients, client)
		}
	}
	return clients
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetRideCount returns the number of active rides
func (h *Hub) GetRideCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rides)
}
