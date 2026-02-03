package chat

import (
	"time"

	"github.com/google/uuid"
)

// MessageType classifies chat messages
type MessageType string

const (
	MessageTypeText     MessageType = "text"
	MessageTypeImage    MessageType = "image"
	MessageTypeLocation MessageType = "location"
	MessageTypeSystem   MessageType = "system" // Auto-generated (driver arrived, etc.)
)

// MessageStatus tracks delivery state
type MessageStatus string

const (
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
)

// ChatMessage represents a single message in a ride conversation
type ChatMessage struct {
	ID          uuid.UUID     `json:"id" db:"id"`
	RideID      uuid.UUID     `json:"ride_id" db:"ride_id"`
	SenderID    uuid.UUID     `json:"sender_id" db:"sender_id"`
	SenderRole  string        `json:"sender_role" db:"sender_role"` // rider, driver, system
	MessageType MessageType   `json:"message_type" db:"message_type"`
	Content     string        `json:"content" db:"content"`
	ImageURL    *string       `json:"image_url,omitempty" db:"image_url"`
	Latitude    *float64      `json:"latitude,omitempty" db:"latitude"`
	Longitude   *float64      `json:"longitude,omitempty" db:"longitude"`
	Status      MessageStatus `json:"status" db:"status"`
	DeliveredAt *time.Time    `json:"delivered_at,omitempty" db:"delivered_at"`
	ReadAt      *time.Time    `json:"read_at,omitempty" db:"read_at"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
}

// QuickReply represents a predefined quick reply option
type QuickReply struct {
	ID       uuid.UUID `json:"id" db:"id"`
	Role     string    `json:"role" db:"role"` // rider, driver, both
	Category string    `json:"category" db:"category"`
	Text     string    `json:"text" db:"text"`
	SortOrder int      `json:"sort_order" db:"sort_order"`
	IsActive bool      `json:"is_active" db:"is_active"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// SendMessageRequest sends a new chat message
type SendMessageRequest struct {
	RideID      uuid.UUID   `json:"ride_id" binding:"required"`
	MessageType MessageType `json:"message_type" binding:"required"`
	Content     string      `json:"content" binding:"required"`
	ImageURL    *string     `json:"image_url,omitempty"`
	Latitude    *float64    `json:"latitude,omitempty"`
	Longitude   *float64    `json:"longitude,omitempty"`
}

// MarkReadRequest marks messages as read up to a given message
type MarkReadRequest struct {
	RideID      uuid.UUID `json:"ride_id" binding:"required"`
	LastReadID  uuid.UUID `json:"last_read_id" binding:"required"`
}

// ChatConversation represents a ride's chat with metadata
type ChatConversation struct {
	RideID       uuid.UUID      `json:"ride_id"`
	Messages     []ChatMessage  `json:"messages"`
	UnreadCount  int            `json:"unread_count"`
	LastMessage  *ChatMessage   `json:"last_message,omitempty"`
	Participants []Participant  `json:"participants"`
}

// Participant represents a chat participant
type Participant struct {
	UserID   uuid.UUID `json:"user_id"`
	Role     string    `json:"role"`
	Name     string    `json:"name"`
	IsOnline bool      `json:"is_online"`
}

// TranslationRequest requests auto-translation of a message
type TranslationRequest struct {
	MessageID    uuid.UUID `json:"message_id" binding:"required"`
	TargetLocale string    `json:"target_locale" binding:"required"`
}
