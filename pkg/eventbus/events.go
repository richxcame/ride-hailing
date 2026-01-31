package eventbus

import (
	"time"

	"github.com/google/uuid"
)

// RideRequestedData is emitted when a rider requests a ride.
type RideRequestedData struct {
	RideID      uuid.UUID `json:"ride_id"`
	RiderID     uuid.UUID `json:"rider_id"`
	PickupLat   float64   `json:"pickup_lat"`
	PickupLng   float64   `json:"pickup_lng"`
	DropoffLat  float64   `json:"dropoff_lat"`
	DropoffLng  float64   `json:"dropoff_lng"`
	RideType    string    `json:"ride_type"`
	RequestedAt time.Time `json:"requested_at"`
}

// RideAcceptedData is emitted when a driver accepts a ride.
type RideAcceptedData struct {
	RideID     uuid.UUID `json:"ride_id"`
	RiderID    uuid.UUID `json:"rider_id"`
	DriverID   uuid.UUID `json:"driver_id"`
	PickupLat  float64   `json:"pickup_lat"`
	PickupLng  float64   `json:"pickup_lng"`
	DropoffLat float64   `json:"dropoff_lat"`
	DropoffLng float64   `json:"dropoff_lng"`
	AcceptedAt time.Time `json:"accepted_at"`
}

// RideStartedData is emitted when a ride begins.
type RideStartedData struct {
	RideID    uuid.UUID `json:"ride_id"`
	RiderID   uuid.UUID `json:"rider_id"`
	DriverID  uuid.UUID `json:"driver_id"`
	StartedAt time.Time `json:"started_at"`
}

// RideCompletedData is emitted when a ride finishes.
type RideCompletedData struct {
	RideID      uuid.UUID `json:"ride_id"`
	RiderID     uuid.UUID `json:"rider_id"`
	DriverID    uuid.UUID `json:"driver_id"`
	FareAmount  float64   `json:"fare_amount"`
	DistanceKm  float64   `json:"distance_km"`
	DurationMin float64   `json:"duration_min"`
	CompletedAt time.Time `json:"completed_at"`
}

// RideCancelledData is emitted when a ride is cancelled.
type RideCancelledData struct {
	RideID      uuid.UUID `json:"ride_id"`
	RiderID     uuid.UUID `json:"rider_id"`
	DriverID    uuid.UUID `json:"driver_id"` // zero if not yet assigned
	CancelledBy string    `json:"cancelled_by"` // "rider" or "driver"
	Reason      string    `json:"reason"`
	CancelledAt time.Time `json:"cancelled_at"`
}

// PaymentProcessedData is emitted after successful payment.
type PaymentProcessedData struct {
	PaymentID uuid.UUID `json:"payment_id"`
	RideID    uuid.UUID `json:"ride_id"`
	RiderID   uuid.UUID `json:"rider_id"`
	DriverID  uuid.UUID `json:"driver_id"`
	Amount    float64   `json:"amount"`
	Currency  string    `json:"currency"`
	Method    string    `json:"method"`
	ProcessedAt time.Time `json:"processed_at"`
}

// PaymentFailedData is emitted when payment fails.
type PaymentFailedData struct {
	PaymentID uuid.UUID `json:"payment_id"`
	RideID    uuid.UUID `json:"ride_id"`
	RiderID   uuid.UUID `json:"rider_id"`
	Amount    float64   `json:"amount"`
	Error     string    `json:"error"`
	FailedAt  time.Time `json:"failed_at"`
}

// DriverLocationUpdatedData is emitted on significant location changes.
type DriverLocationUpdatedData struct {
	DriverID  uuid.UUID `json:"driver_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Heading   float64   `json:"heading"`
	Speed     float64   `json:"speed"`
	H3Cell    string    `json:"h3_cell"`
	Timestamp time.Time `json:"timestamp"`
}

// FraudDetectedData is emitted when suspicious activity is detected.
type FraudDetectedData struct {
	UserID     uuid.UUID `json:"user_id"`
	AlertType  string    `json:"alert_type"`
	Severity   string    `json:"severity"`
	Details    string    `json:"details"`
	DetectedAt time.Time `json:"detected_at"`
}
