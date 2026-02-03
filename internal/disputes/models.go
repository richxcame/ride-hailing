package disputes

import (
	"time"

	"github.com/google/uuid"
)

// DisputeStatus represents the status of a fare dispute
type DisputeStatus string

const (
	DisputeStatusPending    DisputeStatus = "pending"
	DisputeStatusReviewing  DisputeStatus = "reviewing"
	DisputeStatusApproved   DisputeStatus = "approved"
	DisputeStatusRejected   DisputeStatus = "rejected"
	DisputeStatusPartial    DisputeStatus = "partial_refund"
	DisputeStatusClosed     DisputeStatus = "closed"
)

// DisputeReason represents why the user is disputing the fare
type DisputeReason string

const (
	ReasonWrongRoute     DisputeReason = "wrong_route"
	ReasonOvercharged    DisputeReason = "overcharged"
	ReasonTripNotTaken   DisputeReason = "trip_not_taken"
	ReasonDriverDetour   DisputeReason = "driver_detour"
	ReasonWrongFare      DisputeReason = "wrong_fare"
	ReasonSurgeUnfair    DisputeReason = "unfair_surge"
	ReasonWaitTimeWrong  DisputeReason = "wrong_wait_time"
	ReasonCancelFeeWrong DisputeReason = "wrong_cancel_fee"
	ReasonDuplicateCharge DisputeReason = "duplicate_charge"
	ReasonOther          DisputeReason = "other"
)

// ResolutionType represents how the dispute was resolved
type ResolutionType string

const (
	ResolutionFullRefund    ResolutionType = "full_refund"
	ResolutionPartialRefund ResolutionType = "partial_refund"
	ResolutionCredits       ResolutionType = "credits"
	ResolutionNoAction      ResolutionType = "no_action"
	ResolutionFareAdjust    ResolutionType = "fare_adjustment"
)

// Dispute represents a fare dispute
type Dispute struct {
	ID              uuid.UUID      `json:"id" db:"id"`
	RideID          uuid.UUID      `json:"ride_id" db:"ride_id"`
	UserID          uuid.UUID      `json:"user_id" db:"user_id"`
	DriverID        *uuid.UUID     `json:"driver_id,omitempty" db:"driver_id"`
	DisputeNumber   string         `json:"dispute_number" db:"dispute_number"`
	Reason          DisputeReason  `json:"reason" db:"reason"`
	Description     string         `json:"description" db:"description"`
	Status          DisputeStatus  `json:"status" db:"status"`
	OriginalFare    float64        `json:"original_fare" db:"original_fare"`
	DisputedAmount  float64        `json:"disputed_amount" db:"disputed_amount"`
	RefundAmount    *float64       `json:"refund_amount,omitempty" db:"refund_amount"`
	ResolutionType  *ResolutionType `json:"resolution_type,omitempty" db:"resolution_type"`
	ResolutionNote  *string        `json:"resolution_note,omitempty" db:"resolution_note"`
	ResolvedBy      *uuid.UUID     `json:"resolved_by,omitempty" db:"resolved_by"`
	Evidence        []string       `json:"evidence,omitempty" db:"evidence"`
	ResolvedAt      *time.Time     `json:"resolved_at,omitempty" db:"resolved_at"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
}

// DisputeComment represents a comment in a dispute thread
type DisputeComment struct {
	ID         uuid.UUID `json:"id" db:"id"`
	DisputeID  uuid.UUID `json:"dispute_id" db:"dispute_id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	UserRole   string    `json:"user_role" db:"user_role"`
	Comment    string    `json:"comment" db:"comment"`
	IsInternal bool      `json:"is_internal" db:"is_internal"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// RideContext provides ride details relevant to a dispute
type RideContext struct {
	RideID           uuid.UUID  `json:"ride_id"`
	EstimatedFare    float64    `json:"estimated_fare"`
	FinalFare        *float64   `json:"final_fare,omitempty"`
	EstimatedDistance float64   `json:"estimated_distance_km"`
	ActualDistance    *float64   `json:"actual_distance_km,omitempty"`
	EstimatedDuration int       `json:"estimated_duration_min"`
	ActualDuration    *int       `json:"actual_duration_min,omitempty"`
	SurgeMultiplier  float64    `json:"surge_multiplier"`
	PickupAddress    string     `json:"pickup_address"`
	DropoffAddress   string     `json:"dropoff_address"`
	RequestedAt      time.Time  `json:"requested_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

// DisputeSummary is a lightweight dispute for lists
type DisputeSummary struct {
	ID             uuid.UUID     `json:"id"`
	DisputeNumber  string        `json:"dispute_number"`
	RideID         uuid.UUID     `json:"ride_id"`
	Reason         DisputeReason `json:"reason"`
	Status         DisputeStatus `json:"status"`
	OriginalFare   float64       `json:"original_fare"`
	DisputedAmount float64       `json:"disputed_amount"`
	RefundAmount   *float64      `json:"refund_amount,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// DisputeStats represents admin dispute analytics
type DisputeStats struct {
	TotalDisputes      int                  `json:"total_disputes"`
	PendingDisputes    int                  `json:"pending_disputes"`
	ReviewingDisputes  int                  `json:"reviewing_disputes"`
	ApprovedDisputes   int                  `json:"approved_disputes"`
	RejectedDisputes   int                  `json:"rejected_disputes"`
	TotalRefunded      float64              `json:"total_refunded"`
	TotalDisputed      float64              `json:"total_disputed"`
	AvgResolutionHours float64              `json:"avg_resolution_hours"`
	DisputeRate        float64              `json:"dispute_rate"`
	ByReason           []ReasonCount        `json:"by_reason"`
}

// ReasonCount represents a dispute reason with its count
type ReasonCount struct {
	Reason     DisputeReason `json:"reason"`
	Count      int           `json:"count"`
	Percentage float64       `json:"percentage"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// CreateDisputeRequest creates a new fare dispute
type CreateDisputeRequest struct {
	RideID         uuid.UUID     `json:"ride_id" binding:"required"`
	Reason         DisputeReason `json:"reason" binding:"required"`
	Description    string        `json:"description" binding:"required,min=10,max=2000"`
	DisputedAmount float64       `json:"disputed_amount" binding:"required,gt=0"`
	Evidence       []string      `json:"evidence,omitempty"`
}

// AddCommentRequest adds a comment to a dispute
type AddCommentRequest struct {
	Comment string `json:"comment" binding:"required,min=1,max=2000"`
}

// ResolveDisputeRequest resolves a dispute (admin)
type ResolveDisputeRequest struct {
	ResolutionType ResolutionType `json:"resolution_type" binding:"required"`
	RefundAmount   *float64       `json:"refund_amount,omitempty"`
	Note           string         `json:"note" binding:"required,min=5"`
}

// DisputeDetailResponse is the full dispute view
type DisputeDetailResponse struct {
	Dispute     *Dispute         `json:"dispute"`
	RideContext *RideContext     `json:"ride_context"`
	Comments    []DisputeComment `json:"comments"`
}

// DisputeReasonsResponse lists available dispute reasons
type DisputeReasonsResponse struct {
	Reasons []DisputeReasonOption `json:"reasons"`
}

// DisputeReasonOption is a selectable dispute reason
type DisputeReasonOption struct {
	Code        DisputeReason `json:"code"`
	Label       string        `json:"label"`
	Description string        `json:"description"`
}
