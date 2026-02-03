package negotiation

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/websocket"
	"go.uber.org/zap"
)

// toMap converts a struct to map[string]interface{} for WebSocket message
func toMap(v interface{}) map[string]interface{} {
	data, _ := json.Marshal(v)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result
}

// EventType represents negotiation event types
type EventType string

const (
	// Session events
	EventSessionCreated   EventType = "negotiation.session_created"
	EventSessionUpdated   EventType = "negotiation.session_updated"
	EventSessionExpired   EventType = "negotiation.session_expired"
	EventSessionCancelled EventType = "negotiation.session_cancelled"
	EventSessionCompleted EventType = "negotiation.session_completed"

	// Offer events
	EventOfferReceived     EventType = "negotiation.offer_received"
	EventOfferAccepted     EventType = "negotiation.offer_accepted"
	EventOfferRejected     EventType = "negotiation.offer_rejected"
	EventOfferWithdrawn    EventType = "negotiation.offer_withdrawn"
	EventCounterOffer      EventType = "negotiation.counter_offer"
	EventOfferExpired      EventType = "negotiation.offer_expired"

	// Driver events
	EventDriverInvited     EventType = "negotiation.driver_invited"
	EventDriverJoined      EventType = "negotiation.driver_joined"
	EventDriverLeft        EventType = "negotiation.driver_left"

	// System events
	EventFairPriceUpdated  EventType = "negotiation.fair_price_updated"
	EventTimeWarning       EventType = "negotiation.time_warning"
)

// EventEmitter handles real-time event emission for negotiations
type EventEmitter struct {
	hub *websocket.Hub
}

// NewEventEmitter creates a new event emitter
func NewEventEmitter(hub *websocket.Hub) *EventEmitter {
	return &EventEmitter{hub: hub}
}

// ========================================
// SESSION EVENTS
// ========================================

// SessionCreatedEvent represents a new negotiation session
type SessionCreatedEvent struct {
	SessionID        uuid.UUID     `json:"session_id"`
	RiderID          uuid.UUID     `json:"rider_id"`
	PickupLocation   Location      `json:"pickup_location"`
	DropoffLocation  Location      `json:"dropoff_location"`
	SystemEstimate   float64       `json:"system_estimate"`
	FairPriceMin     float64       `json:"fair_price_min"`
	FairPriceMax     float64       `json:"fair_price_max"`
	ProposedPrice    *float64      `json:"proposed_price,omitempty"`
	ExpiresAt        time.Time     `json:"expires_at"`
	VehicleType      string        `json:"vehicle_type,omitempty"`
	EstimatedDistance float64      `json:"estimated_distance_km"`
	EstimatedDuration int          `json:"estimated_duration_min"`
	CreatedAt        time.Time     `json:"created_at"`
}

// Location represents a geographic location
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address,omitempty"`
}

// EmitSessionCreated notifies nearby drivers about a new negotiation session
func (e *EventEmitter) EmitSessionCreated(ctx context.Context, event *SessionCreatedEvent, driverIDs []string) {
	msg := &websocket.Message{
		Type: string(EventSessionCreated),
		Data: toMap(event),
	}

	// Send to all invited drivers
	e.hub.SendToMultipleUsers(driverIDs, msg)

	// Add drivers to negotiation room for real-time updates
	for _, driverID := range driverIDs {
		e.hub.AddClientToNegotiation(driverID, event.SessionID.String())
	}

	// Also add the rider to the negotiation room
	e.hub.AddClientToNegotiation(event.RiderID.String(), event.SessionID.String())

	logger.Info("Session created event emitted",
		zap.String("session_id", event.SessionID.String()),
		zap.Int("driver_count", len(driverIDs)),
	)
}

// SessionCompletedEvent represents a completed negotiation
type SessionCompletedEvent struct {
	SessionID       uuid.UUID  `json:"session_id"`
	AcceptedOfferID uuid.UUID  `json:"accepted_offer_id"`
	DriverID        uuid.UUID  `json:"driver_id"`
	RiderID         uuid.UUID  `json:"rider_id"`
	FinalPrice      float64    `json:"final_price"`
	RideID          *uuid.UUID `json:"ride_id,omitempty"`
	CompletedAt     time.Time  `json:"completed_at"`
}

// EmitSessionCompleted notifies participants that negotiation is complete
func (e *EventEmitter) EmitSessionCompleted(ctx context.Context, event *SessionCompletedEvent) {
	msg := &websocket.Message{
		Type: string(EventSessionCompleted),
		Data: toMap(event),
	}

	// Send to negotiation room
	e.hub.SendToNegotiation(event.SessionID.String(), msg)

	// Close the negotiation room
	e.hub.CloseNegotiation(event.SessionID.String(), "session_completed")

	logger.Info("Session completed event emitted",
		zap.String("session_id", event.SessionID.String()),
		zap.String("driver_id", event.DriverID.String()),
		zap.Float64("final_price", event.FinalPrice),
	)
}

// SessionCancelledEvent represents a cancelled negotiation
type SessionCancelledEvent struct {
	SessionID    uuid.UUID `json:"session_id"`
	CancelledBy  string    `json:"cancelled_by"` // "rider" or "system"
	Reason       string    `json:"reason"`
	CancelledAt  time.Time `json:"cancelled_at"`
}

// EmitSessionCancelled notifies participants that negotiation was cancelled
func (e *EventEmitter) EmitSessionCancelled(ctx context.Context, event *SessionCancelledEvent) {
	msg := &websocket.Message{
		Type: string(EventSessionCancelled),
		Data: toMap(event),
	}

	e.hub.SendToNegotiation(event.SessionID.String(), msg)
	e.hub.CloseNegotiation(event.SessionID.String(), event.Reason)

	logger.Info("Session cancelled event emitted",
		zap.String("session_id", event.SessionID.String()),
		zap.String("reason", event.Reason),
	)
}

// SessionExpiredEvent represents an expired negotiation
type SessionExpiredEvent struct {
	SessionID  uuid.UUID `json:"session_id"`
	ExpiredAt  time.Time `json:"expired_at"`
	TotalOffers int      `json:"total_offers"`
}

// EmitSessionExpired notifies participants that negotiation expired
func (e *EventEmitter) EmitSessionExpired(ctx context.Context, event *SessionExpiredEvent) {
	msg := &websocket.Message{
		Type: string(EventSessionExpired),
		Data: toMap(event),
	}

	e.hub.SendToNegotiation(event.SessionID.String(), msg)
	e.hub.CloseNegotiation(event.SessionID.String(), "session_expired")

	logger.Info("Session expired event emitted",
		zap.String("session_id", event.SessionID.String()),
	)
}

// ========================================
// OFFER EVENTS
// ========================================

// OfferReceivedEvent represents a new offer from a driver
type OfferReceivedEvent struct {
	SessionID     uuid.UUID  `json:"session_id"`
	OfferID       uuid.UUID  `json:"offer_id"`
	DriverID      uuid.UUID  `json:"driver_id"`
	DriverName    string     `json:"driver_name"`
	DriverRating  float64    `json:"driver_rating"`
	DriverPhoto   string     `json:"driver_photo,omitempty"`
	VehicleInfo   string     `json:"vehicle_info,omitempty"`
	OfferedPrice  float64    `json:"offered_price"`
	ETA           int        `json:"eta_minutes"`
	Message       string     `json:"message,omitempty"`
	OfferNumber   int        `json:"offer_number"` // 1st, 2nd, 3rd offer
	TotalOffers   int        `json:"total_offers"`
	CreatedAt     time.Time  `json:"created_at"`
	ExpiresAt     time.Time  `json:"expires_at"`
}

// EmitOfferReceived notifies the rider about a new offer
func (e *EventEmitter) EmitOfferReceived(ctx context.Context, event *OfferReceivedEvent) {
	msg := &websocket.Message{
		Type: string(EventOfferReceived),
		Data: toMap(event),
	}

	// Send to negotiation room (rider + all drivers can see offers)
	e.hub.SendToNegotiation(event.SessionID.String(), msg)

	logger.Info("Offer received event emitted",
		zap.String("session_id", event.SessionID.String()),
		zap.String("offer_id", event.OfferID.String()),
		zap.Float64("price", event.OfferedPrice),
	)
}

// OfferAcceptedEvent represents an accepted offer
type OfferAcceptedEvent struct {
	SessionID    uuid.UUID `json:"session_id"`
	OfferID      uuid.UUID `json:"offer_id"`
	DriverID     uuid.UUID `json:"driver_id"`
	RiderID      uuid.UUID `json:"rider_id"`
	AcceptedPrice float64  `json:"accepted_price"`
	AcceptedAt   time.Time `json:"accepted_at"`
}

// EmitOfferAccepted notifies participants that an offer was accepted
func (e *EventEmitter) EmitOfferAccepted(ctx context.Context, event *OfferAcceptedEvent) {
	msg := &websocket.Message{
		Type: string(EventOfferAccepted),
		Data: toMap(event),
	}

	e.hub.SendToNegotiation(event.SessionID.String(), msg)

	logger.Info("Offer accepted event emitted",
		zap.String("session_id", event.SessionID.String()),
		zap.String("offer_id", event.OfferID.String()),
	)
}

// OfferRejectedEvent represents a rejected offer
type OfferRejectedEvent struct {
	SessionID  uuid.UUID `json:"session_id"`
	OfferID    uuid.UUID `json:"offer_id"`
	DriverID   uuid.UUID `json:"driver_id"`
	Reason     string    `json:"reason,omitempty"`
	RejectedAt time.Time `json:"rejected_at"`
}

// EmitOfferRejected notifies the driver that their offer was rejected
func (e *EventEmitter) EmitOfferRejected(ctx context.Context, event *OfferRejectedEvent) {
	msg := &websocket.Message{
		Type: string(EventOfferRejected),
		Data: toMap(event),
	}

	// Send only to the driver who made the offer
	e.hub.SendToUser(event.DriverID.String(), msg)

	logger.Info("Offer rejected event emitted",
		zap.String("session_id", event.SessionID.String()),
		zap.String("offer_id", event.OfferID.String()),
	)
}

// CounterOfferEvent represents a counter-offer from the rider
type CounterOfferEvent struct {
	SessionID      uuid.UUID `json:"session_id"`
	OriginalOfferID uuid.UUID `json:"original_offer_id"`
	DriverID       uuid.UUID `json:"driver_id"`
	CounterPrice   float64   `json:"counter_price"`
	Message        string    `json:"message,omitempty"`
	Round          int       `json:"round"` // Counter-offer round number
	MaxRounds      int       `json:"max_rounds"`
	ExpiresAt      time.Time `json:"expires_at"`
	CreatedAt      time.Time `json:"created_at"`
}

// EmitCounterOffer notifies the driver about a counter-offer
func (e *EventEmitter) EmitCounterOffer(ctx context.Context, event *CounterOfferEvent) {
	msg := &websocket.Message{
		Type: string(EventCounterOffer),
		Data: toMap(event),
	}

	// Send to the specific driver
	e.hub.SendToUser(event.DriverID.String(), msg)

	logger.Info("Counter offer event emitted",
		zap.String("session_id", event.SessionID.String()),
		zap.String("driver_id", event.DriverID.String()),
		zap.Float64("counter_price", event.CounterPrice),
	)
}

// OfferWithdrawnEvent represents a withdrawn offer
type OfferWithdrawnEvent struct {
	SessionID   uuid.UUID `json:"session_id"`
	OfferID     uuid.UUID `json:"offer_id"`
	DriverID    uuid.UUID `json:"driver_id"`
	Reason      string    `json:"reason,omitempty"`
	WithdrawnAt time.Time `json:"withdrawn_at"`
}

// EmitOfferWithdrawn notifies participants that an offer was withdrawn
func (e *EventEmitter) EmitOfferWithdrawn(ctx context.Context, event *OfferWithdrawnEvent) {
	msg := &websocket.Message{
		Type: string(EventOfferWithdrawn),
		Data: toMap(event),
	}

	e.hub.SendToNegotiation(event.SessionID.String(), msg)

	logger.Info("Offer withdrawn event emitted",
		zap.String("session_id", event.SessionID.String()),
		zap.String("offer_id", event.OfferID.String()),
	)
}

// ========================================
// DRIVER EVENTS
// ========================================

// DriverInvitedEvent represents a driver invitation
type DriverInvitedEvent struct {
	SessionID       uuid.UUID `json:"session_id"`
	DriverID        uuid.UUID `json:"driver_id"`
	PickupLocation  Location  `json:"pickup_location"`
	DropoffLocation Location  `json:"dropoff_location"`
	EstimatedPrice  float64   `json:"estimated_price"`
	FairPriceMin    float64   `json:"fair_price_min"`
	FairPriceMax    float64   `json:"fair_price_max"`
	DistanceToPickup float64  `json:"distance_to_pickup_km"`
	ETAToPickup     int       `json:"eta_to_pickup_min"`
	ExpiresAt       time.Time `json:"expires_at"`
}

// EmitDriverInvited sends invitation to a specific driver
func (e *EventEmitter) EmitDriverInvited(ctx context.Context, event *DriverInvitedEvent) {
	msg := &websocket.Message{
		Type: string(EventDriverInvited),
		Data: toMap(event),
	}

	e.hub.SendToUser(event.DriverID.String(), msg)
	e.hub.AddClientToNegotiation(event.DriverID.String(), event.SessionID.String())

	logger.Debug("Driver invited event emitted",
		zap.String("session_id", event.SessionID.String()),
		zap.String("driver_id", event.DriverID.String()),
	)
}

// DriverJoinedEvent represents a driver joining the negotiation
type DriverJoinedEvent struct {
	SessionID      uuid.UUID `json:"session_id"`
	DriverID       uuid.UUID `json:"driver_id"`
	DriverName     string    `json:"driver_name"`
	DriverRating   float64   `json:"driver_rating"`
	TotalDrivers   int       `json:"total_drivers"`
	JoinedAt       time.Time `json:"joined_at"`
}

// EmitDriverJoined notifies the rider that a driver joined
func (e *EventEmitter) EmitDriverJoined(ctx context.Context, event *DriverJoinedEvent) {
	msg := &websocket.Message{
		Type: string(EventDriverJoined),
		Data: toMap(event),
	}

	// Notify only the rider about new driver joining
	e.hub.SendToNegotiation(event.SessionID.String(), msg)

	logger.Debug("Driver joined event emitted",
		zap.String("session_id", event.SessionID.String()),
		zap.String("driver_id", event.DriverID.String()),
	)
}

// DriverLeftEvent represents a driver leaving the negotiation
type DriverLeftEvent struct {
	SessionID    uuid.UUID `json:"session_id"`
	DriverID     uuid.UUID `json:"driver_id"`
	Reason       string    `json:"reason,omitempty"`
	TotalDrivers int       `json:"total_drivers"`
	LeftAt       time.Time `json:"left_at"`
}

// EmitDriverLeft notifies participants that a driver left
func (e *EventEmitter) EmitDriverLeft(ctx context.Context, event *DriverLeftEvent) {
	msg := &websocket.Message{
		Type: string(EventDriverLeft),
		Data: toMap(event),
	}

	e.hub.SendToNegotiation(event.SessionID.String(), msg)
	e.hub.RemoveClientFromNegotiation(event.DriverID.String(), event.SessionID.String())

	logger.Debug("Driver left event emitted",
		zap.String("session_id", event.SessionID.String()),
		zap.String("driver_id", event.DriverID.String()),
	)
}

// ========================================
// SYSTEM EVENTS
// ========================================

// TimeWarningEvent represents a time warning for the negotiation
type TimeWarningEvent struct {
	SessionID       uuid.UUID `json:"session_id"`
	RemainingSeconds int      `json:"remaining_seconds"`
	WarningLevel    string    `json:"warning_level"` // "1min", "30sec", "10sec"
}

// EmitTimeWarning sends a countdown warning to all participants
func (e *EventEmitter) EmitTimeWarning(ctx context.Context, event *TimeWarningEvent) {
	msg := &websocket.Message{
		Type: string(EventTimeWarning),
		Data: toMap(event),
	}

	e.hub.SendToNegotiation(event.SessionID.String(), msg)

	logger.Debug("Time warning event emitted",
		zap.String("session_id", event.SessionID.String()),
		zap.Int("remaining_seconds", event.RemainingSeconds),
	)
}

// FairPriceUpdatedEvent represents an updated fair price calculation
type FairPriceUpdatedEvent struct {
	SessionID    uuid.UUID `json:"session_id"`
	FairPriceMin float64   `json:"fair_price_min"`
	FairPriceMax float64   `json:"fair_price_max"`
	Reason       string    `json:"reason"` // e.g., "surge_updated", "demand_changed"
	UpdatedAt    time.Time `json:"updated_at"`
}

// EmitFairPriceUpdated notifies participants about fair price changes
func (e *EventEmitter) EmitFairPriceUpdated(ctx context.Context, event *FairPriceUpdatedEvent) {
	msg := &websocket.Message{
		Type: string(EventFairPriceUpdated),
		Data: toMap(event),
	}

	e.hub.SendToNegotiation(event.SessionID.String(), msg)

	logger.Info("Fair price updated event emitted",
		zap.String("session_id", event.SessionID.String()),
		zap.Float64("min", event.FairPriceMin),
		zap.Float64("max", event.FairPriceMax),
	)
}

// ========================================
// HELPER METHODS
// ========================================

// MarshalJSON helper for consistent JSON output
func (e *EventEmitter) marshalEvent(event interface{}) ([]byte, error) {
	return json.Marshal(event)
}

// AddRiderToSession adds the rider to the negotiation room
func (e *EventEmitter) AddRiderToSession(riderID, sessionID string) {
	e.hub.AddClientToNegotiation(riderID, sessionID)
}

// RemoveParticipant removes a participant from the negotiation room
func (e *EventEmitter) RemoveParticipant(userID, sessionID string) {
	e.hub.RemoveClientFromNegotiation(userID, sessionID)
}

// GetParticipantCount returns the number of participants in a session
func (e *EventEmitter) GetParticipantCount(sessionID string) int {
	return len(e.hub.GetClientsInNegotiation(sessionID))
}

// IsUserInSession checks if a user is in a negotiation session
func (e *EventEmitter) IsUserInSession(userID, sessionID string) bool {
	clients := e.hub.GetClientsInNegotiation(sessionID)
	for _, client := range clients {
		if client.ID == userID {
			return true
		}
	}
	return false
}
