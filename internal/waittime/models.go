package waittime

import (
	"time"

	"github.com/google/uuid"
)

// WaitTimeConfig defines wait time charge rules per region/ride type
type WaitTimeConfig struct {
	ID               uuid.UUID `json:"id" db:"id"`
	Name             string    `json:"name" db:"name"`                           // e.g., "Standard", "Airport", "Premium"
	FreeWaitMinutes  int       `json:"free_wait_minutes" db:"free_wait_minutes"` // Grace period (typically 2-5 min)
	ChargePerMinute  float64   `json:"charge_per_minute" db:"charge_per_minute"` // Per-minute rate after grace
	MaxWaitMinutes   int       `json:"max_wait_minutes" db:"max_wait_minutes"`   // Auto-cancel threshold
	MaxWaitCharge    float64   `json:"max_wait_charge" db:"max_wait_charge"`     // Cap on wait charges
	AppliesTo        string    `json:"applies_to" db:"applies_to"`               // pickup, dropoff, both
	IsActive         bool      `json:"is_active" db:"is_active"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// WaitTimeRecord tracks actual wait time for a ride
type WaitTimeRecord struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	RideID          uuid.UUID  `json:"ride_id" db:"ride_id"`
	DriverID        uuid.UUID  `json:"driver_id" db:"driver_id"`
	ConfigID        uuid.UUID  `json:"config_id" db:"config_id"`
	WaitType        string     `json:"wait_type" db:"wait_type"` // pickup, dropoff
	ArrivedAt       time.Time  `json:"arrived_at" db:"arrived_at"`
	StartedAt       *time.Time `json:"started_at,omitempty" db:"started_at"` // When ride actually started
	FreeMinutes     int        `json:"free_minutes" db:"free_minutes"`
	TotalWaitMinutes float64   `json:"total_wait_minutes" db:"total_wait_minutes"`
	ChargeableMinutes float64  `json:"chargeable_minutes" db:"chargeable_minutes"`
	ChargePerMinute  float64   `json:"charge_per_minute" db:"charge_per_minute"`
	TotalCharge      float64   `json:"total_charge" db:"total_charge"`
	WasCapped        bool      `json:"was_capped" db:"was_capped"`
	Status           string    `json:"status" db:"status"` // waiting, completed, cancelled, waived
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// WaitTimeNotification tracks notifications sent about wait time
type WaitTimeNotification struct {
	ID        uuid.UUID `json:"id" db:"id"`
	RecordID  uuid.UUID `json:"record_id" db:"record_id"`
	RideID    uuid.UUID `json:"ride_id" db:"ride_id"`
	RiderID   uuid.UUID `json:"rider_id" db:"rider_id"`
	MinutesMark int     `json:"minutes_mark" db:"minutes_mark"` // At what minute was this sent
	Message   string    `json:"message" db:"message"`
	SentAt    time.Time `json:"sent_at" db:"sent_at"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// StartWaitRequest marks the driver as arrived and waiting
type StartWaitRequest struct {
	RideID   uuid.UUID `json:"ride_id" binding:"required"`
	WaitType string    `json:"wait_type" binding:"required"` // pickup
}

// StopWaitRequest marks the wait as ended (rider boarded)
type StopWaitRequest struct {
	RideID uuid.UUID `json:"ride_id" binding:"required"`
}

// WaiveChargeRequest waives wait time charges (admin/support)
type WaiveChargeRequest struct {
	RecordID uuid.UUID `json:"record_id" binding:"required"`
	Reason   string    `json:"reason" binding:"required"`
}

// WaitTimeSummary summarizes wait time charges for a ride
type WaitTimeSummary struct {
	RideID             uuid.UUID        `json:"ride_id"`
	Records            []WaitTimeRecord `json:"records"`
	TotalWaitMinutes   float64          `json:"total_wait_minutes"`
	TotalChargeableMin float64          `json:"total_chargeable_minutes"`
	TotalCharge        float64          `json:"total_charge"`
	HasActiveWait      bool             `json:"has_active_wait"`
}

// CreateConfigRequest creates a new wait time config (admin)
type CreateConfigRequest struct {
	Name            string  `json:"name" binding:"required"`
	FreeWaitMinutes int     `json:"free_wait_minutes" binding:"required"`
	ChargePerMinute float64 `json:"charge_per_minute" binding:"required"`
	MaxWaitMinutes  int     `json:"max_wait_minutes" binding:"required"`
	MaxWaitCharge   float64 `json:"max_wait_charge" binding:"required"`
	AppliesTo       string  `json:"applies_to" binding:"required"`
}
