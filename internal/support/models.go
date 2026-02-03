package support

import (
	"time"

	"github.com/google/uuid"
)

// TicketStatus represents the status of a support ticket
type TicketStatus string

const (
	TicketStatusOpen       TicketStatus = "open"
	TicketStatusInProgress TicketStatus = "in_progress"
	TicketStatusWaiting    TicketStatus = "waiting_on_user"
	TicketStatusResolved   TicketStatus = "resolved"
	TicketStatusClosed     TicketStatus = "closed"
	TicketStatusEscalated  TicketStatus = "escalated"
)

// TicketPriority represents the priority of a support ticket
type TicketPriority string

const (
	PriorityLow      TicketPriority = "low"
	PriorityMedium   TicketPriority = "medium"
	PriorityHigh     TicketPriority = "high"
	PriorityUrgent   TicketPriority = "urgent"
)

// TicketCategory represents the category of a support ticket
type TicketCategory string

const (
	CategoryPayment      TicketCategory = "payment"
	CategoryRide         TicketCategory = "ride"
	CategoryDriver       TicketCategory = "driver"
	CategoryAccount      TicketCategory = "account"
	CategorySafety       TicketCategory = "safety"
	CategoryPromo        TicketCategory = "promo"
	CategoryApp          TicketCategory = "app_issue"
	CategoryFeedback     TicketCategory = "feedback"
	CategoryLostItem     TicketCategory = "lost_item"
	CategoryFareDispute  TicketCategory = "fare_dispute"
	CategoryOther        TicketCategory = "other"
)

// MessageSender represents who sent the message
type MessageSender string

const (
	SenderUser  MessageSender = "user"
	SenderAgent MessageSender = "agent"
	SenderSystem MessageSender = "system"
)

// Ticket represents a support ticket
type Ticket struct {
	ID           uuid.UUID      `json:"id" db:"id"`
	UserID       uuid.UUID      `json:"user_id" db:"user_id"`
	RideID       *uuid.UUID     `json:"ride_id,omitempty" db:"ride_id"`
	AssignedTo   *uuid.UUID     `json:"assigned_to,omitempty" db:"assigned_to"`
	TicketNumber string         `json:"ticket_number" db:"ticket_number"`
	Category     TicketCategory `json:"category" db:"category"`
	Priority     TicketPriority `json:"priority" db:"priority"`
	Status       TicketStatus   `json:"status" db:"status"`
	Subject      string         `json:"subject" db:"subject"`
	Description  string         `json:"description" db:"description"`
	Tags         []string       `json:"tags,omitempty" db:"tags"`
	ResolvedAt   *time.Time     `json:"resolved_at,omitempty" db:"resolved_at"`
	ClosedAt     *time.Time     `json:"closed_at,omitempty" db:"closed_at"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
}

// TicketMessage represents a message in a support ticket thread
type TicketMessage struct {
	ID         uuid.UUID     `json:"id" db:"id"`
	TicketID   uuid.UUID     `json:"ticket_id" db:"ticket_id"`
	SenderID   uuid.UUID     `json:"sender_id" db:"sender_id"`
	SenderType MessageSender `json:"sender_type" db:"sender_type"`
	Message    string        `json:"message" db:"message"`
	Attachments []string     `json:"attachments,omitempty" db:"attachments"`
	IsInternal bool          `json:"is_internal" db:"is_internal"`
	CreatedAt  time.Time     `json:"created_at" db:"created_at"`
}

// FAQArticle represents a help center article
type FAQArticle struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Category  string    `json:"category" db:"category"`
	Title     string    `json:"title" db:"title"`
	Content   string    `json:"content" db:"content"`
	SortOrder int       `json:"sort_order" db:"sort_order"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	ViewCount int       `json:"view_count" db:"view_count"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TicketSummary is a lightweight view of a ticket for lists
type TicketSummary struct {
	ID           uuid.UUID      `json:"id"`
	TicketNumber string         `json:"ticket_number"`
	Category     TicketCategory `json:"category"`
	Priority     TicketPriority `json:"priority"`
	Status       TicketStatus   `json:"status"`
	Subject      string         `json:"subject"`
	LastMessage  *string        `json:"last_message,omitempty"`
	MessageCount int            `json:"message_count"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// TicketStats represents admin ticket analytics
type TicketStats struct {
	TotalOpen        int     `json:"total_open"`
	TotalInProgress  int     `json:"total_in_progress"`
	TotalWaiting     int     `json:"total_waiting"`
	TotalResolved    int     `json:"total_resolved"`
	TotalEscalated   int     `json:"total_escalated"`
	AvgResolutionMin float64 `json:"avg_resolution_minutes"`
	ByCategory       map[string]int `json:"by_category"`
	ByPriority       map[string]int `json:"by_priority"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// CreateTicketRequest represents a request to create a support ticket
type CreateTicketRequest struct {
	Category    TicketCategory `json:"category" binding:"required"`
	Subject     string         `json:"subject" binding:"required,min=5,max=200"`
	Description string         `json:"description" binding:"required,min=10,max=5000"`
	RideID      *uuid.UUID     `json:"ride_id,omitempty"`
	Priority    *TicketPriority `json:"priority,omitempty"`
}

// AddMessageRequest adds a message to a ticket
type AddMessageRequest struct {
	Message     string   `json:"message" binding:"required,min=1,max=5000"`
	Attachments []string `json:"attachments,omitempty"`
}

// UpdateTicketRequest updates ticket status/assignment (admin)
type UpdateTicketRequest struct {
	Status     *TicketStatus   `json:"status,omitempty"`
	Priority   *TicketPriority `json:"priority,omitempty"`
	AssignedTo *uuid.UUID      `json:"assigned_to,omitempty"`
	Tags       []string        `json:"tags,omitempty"`
}

// AdminReplyRequest is an admin reply with optional internal note
type AdminReplyRequest struct {
	Message    string `json:"message" binding:"required,min=1,max=5000"`
	IsInternal bool   `json:"is_internal"`
}
