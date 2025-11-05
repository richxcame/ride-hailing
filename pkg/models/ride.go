package models

import (
	"time"

	"github.com/google/uuid"
)

// RideStatus represents the status of a ride
type RideStatus string

const (
	RideStatusRequested  RideStatus = "requested"
	RideStatusAccepted   RideStatus = "accepted"
	RideStatusInProgress RideStatus = "in_progress"
	RideStatusCompleted  RideStatus = "completed"
	RideStatusCancelled  RideStatus = "cancelled"
)

// Ride represents a ride in the system
type Ride struct {
	ID                          uuid.UUID   `json:"id" db:"id"`
	RiderID                     uuid.UUID   `json:"rider_id" db:"rider_id"`
	DriverID                    *uuid.UUID  `json:"driver_id,omitempty" db:"driver_id"`
	Status                      RideStatus  `json:"status" db:"status"`
	PickupLatitude              float64     `json:"pickup_latitude" db:"pickup_latitude"`
	PickupLongitude             float64     `json:"pickup_longitude" db:"pickup_longitude"`
	PickupAddress               string      `json:"pickup_address" db:"pickup_address"`
	DropoffLatitude             float64     `json:"dropoff_latitude" db:"dropoff_latitude"`
	DropoffLongitude            float64     `json:"dropoff_longitude" db:"dropoff_longitude"`
	DropoffAddress              string      `json:"dropoff_address" db:"dropoff_address"`
	EstimatedDistance           float64     `json:"estimated_distance" db:"estimated_distance"` // in kilometers
	EstimatedDuration           int         `json:"estimated_duration" db:"estimated_duration"` // in minutes
	EstimatedFare               float64     `json:"estimated_fare" db:"estimated_fare"`
	ActualDistance              *float64    `json:"actual_distance,omitempty" db:"actual_distance"`
	ActualDuration              *int        `json:"actual_duration,omitempty" db:"actual_duration"`
	FinalFare                   *float64    `json:"final_fare,omitempty" db:"final_fare"`
	SurgeMultiplier             float64     `json:"surge_multiplier" db:"surge_multiplier"`
	RequestedAt                 time.Time   `json:"requested_at" db:"requested_at"`
	AcceptedAt                  *time.Time  `json:"accepted_at,omitempty" db:"accepted_at"`
	StartedAt                   *time.Time  `json:"started_at,omitempty" db:"started_at"`
	CompletedAt                 *time.Time  `json:"completed_at,omitempty" db:"completed_at"`
	CancelledAt                 *time.Time  `json:"cancelled_at,omitempty" db:"cancelled_at"`
	CancellationReason          *string     `json:"cancellation_reason,omitempty" db:"cancellation_reason"`
	Rating                      *int        `json:"rating,omitempty" db:"rating"`
	Feedback                    *string     `json:"feedback,omitempty" db:"feedback"`
	RideTypeID                  *uuid.UUID  `json:"ride_type_id,omitempty" db:"ride_type_id"`
	PromoCodeID                 *uuid.UUID  `json:"promo_code_id,omitempty" db:"promo_code_id"`
	DiscountAmount              float64     `json:"discount_amount" db:"discount_amount"`
	ScheduledAt                 *time.Time  `json:"scheduled_at,omitempty" db:"scheduled_at"`
	IsScheduled                 bool        `json:"is_scheduled" db:"is_scheduled"`
	ScheduledNotificationSent   bool        `json:"scheduled_notification_sent" db:"scheduled_notification_sent"`
	CreatedAt                   time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt                   time.Time   `json:"updated_at" db:"updated_at"`
}

// RideRequest represents a ride request from a rider
type RideRequest struct {
	PickupLatitude   float64    `json:"pickup_latitude" binding:"required"`
	PickupLongitude  float64    `json:"pickup_longitude" binding:"required"`
	PickupAddress    string     `json:"pickup_address" binding:"required"`
	DropoffLatitude  float64    `json:"dropoff_latitude" binding:"required"`
	DropoffLongitude float64    `json:"dropoff_longitude" binding:"required"`
	DropoffAddress   string     `json:"dropoff_address" binding:"required"`
	RideTypeID       *uuid.UUID `json:"ride_type_id,omitempty"`       // Optional: defaults to Economy
	PromoCode        string     `json:"promo_code,omitempty"`         // Optional promo code
	ScheduledAt      *time.Time `json:"scheduled_at,omitempty"`       // Optional: for scheduled rides
	IsScheduled      bool       `json:"is_scheduled,omitempty"`       // Is this a scheduled ride?
}

// RideResponse represents a ride response with additional details
type RideResponse struct {
	*Ride
	Rider  *User   `json:"rider,omitempty"`
	Driver *Driver `json:"driver,omitempty"`
}

// RideUpdateRequest represents a request to update ride status
type RideUpdateRequest struct {
	Status             RideStatus `json:"status" binding:"required,oneof=accepted in_progress completed cancelled"`
	CancellationReason *string    `json:"cancellation_reason,omitempty"`
}

// RideRatingRequest represents a request to rate a ride
type RideRatingRequest struct {
	Rating   int     `json:"rating" binding:"required,min=1,max=5"`
	Feedback *string `json:"feedback,omitempty"`
}
