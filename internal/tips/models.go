package tips

import (
	"time"

	"github.com/google/uuid"
)

// TipStatus represents the status of a tip
type TipStatus string

const (
	TipStatusPending   TipStatus = "pending"
	TipStatusCompleted TipStatus = "completed"
	TipStatusRefunded  TipStatus = "refunded"
	TipStatusFailed    TipStatus = "failed"
)

// Tip represents a tip from a rider to a driver
type Tip struct {
	ID         uuid.UUID `json:"id" db:"id"`
	RideID     uuid.UUID `json:"ride_id" db:"ride_id"`
	RiderID    uuid.UUID `json:"rider_id" db:"rider_id"`
	DriverID   uuid.UUID `json:"driver_id" db:"driver_id"`
	Amount     float64   `json:"amount" db:"amount"`
	Currency   string    `json:"currency" db:"currency"`
	Status     TipStatus `json:"status" db:"status"`
	Message    *string   `json:"message,omitempty" db:"message"`
	IsAnonymous bool    `json:"is_anonymous" db:"is_anonymous"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// TipPreset represents a suggested tip amount
type TipPreset struct {
	Amount     float64 `json:"amount"`
	Label      string  `json:"label"`      // e.g., "$2", "15%"
	IsPercent  bool    `json:"is_percent"` // true if percentage-based
	IsDefault  bool    `json:"is_default"` // highlighted option
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// SendTipRequest creates a new tip
type SendTipRequest struct {
	RideID      uuid.UUID `json:"ride_id" binding:"required"`
	Amount      float64   `json:"amount" binding:"required"`
	Message     *string   `json:"message,omitempty"`
	IsAnonymous bool      `json:"is_anonymous"`
}

// TipPresetsResponse returns suggested tip amounts
type TipPresetsResponse struct {
	Presets  []TipPreset `json:"presets"`
	MaxTip   float64     `json:"max_tip"`
	Currency string      `json:"currency"`
}

// DriverTipSummary shows tip stats for a driver
type DriverTipSummary struct {
	TotalTips       float64 `json:"total_tips"`
	TipCount        int     `json:"tip_count"`
	AverageTip      float64 `json:"average_tip"`
	HighestTip      float64 `json:"highest_tip"`
	TipRate         float64 `json:"tip_rate"` // Percentage of rides that get tipped
	PeriodRideCount int     `json:"period_ride_count"`
	Currency        string  `json:"currency"`
	Period          string  `json:"period"`
}

// RiderTipHistory returns tips a rider has given
type RiderTipHistory struct {
	Tips  []Tip `json:"tips"`
	Total int   `json:"total"`
}
