package eventbus

import (
	"time"

	"github.com/google/uuid"
)

// RideRequestedData is emitted when a rider requests a ride.
// This contains all data needed by matching service to send ride offers to drivers.
type RideRequestedData struct {
	RideID            uuid.UUID `json:"ride_id"`
	RiderID           uuid.UUID `json:"rider_id"`
	RiderName         string    `json:"rider_name"`
	RiderRating       float64   `json:"rider_rating"`
	PickupLatitude    float64   `json:"pickup_latitude"`
	PickupLongitude   float64   `json:"pickup_longitude"`
	PickupAddress     string    `json:"pickup_address"`
	DropoffLatitude   float64   `json:"dropoff_latitude"`
	DropoffLongitude  float64   `json:"dropoff_longitude"`
	DropoffAddress    string    `json:"dropoff_address"`
	RideTypeID        uuid.UUID `json:"ride_type_id"`
	RideTypeName      string    `json:"ride_type_name"`
	EstimatedFare     float64   `json:"estimated_fare"`
	EstimatedDistance float64   `json:"estimated_distance_km"`
	EstimatedDuration int       `json:"estimated_duration_minutes"`
	Currency          string    `json:"currency"`
	RequestedAt       time.Time `json:"requested_at"`
}

// RideAcceptedData is emitted when a driver accepts a ride.
type RideAcceptedData struct {
	RideID            uuid.UUID `json:"ride_id"`
	RiderID           uuid.UUID `json:"rider_id"`
	DriverID          uuid.UUID `json:"driver_id"`
	PickupLatitude    float64   `json:"pickup_latitude"`
	PickupLongitude   float64   `json:"pickup_longitude"`
	DropoffLatitude   float64   `json:"dropoff_latitude"`
	DropoffLongitude  float64   `json:"dropoff_longitude"`
	AcceptedAt        time.Time `json:"accepted_at"`
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

// ========================================
// NEGOTIATION EVENTS
// ========================================

// NegotiationStartedData is emitted when a negotiation session starts.
type NegotiationStartedData struct {
	SessionID            uuid.UUID  `json:"session_id"`
	RiderID              uuid.UUID  `json:"rider_id"`
	PickupLatitude       float64    `json:"pickup_latitude"`
	PickupLongitude      float64    `json:"pickup_longitude"`
	DropoffLatitude      float64    `json:"dropoff_latitude"`
	DropoffLongitude     float64    `json:"dropoff_longitude"`
	PickupAddress        string     `json:"pickup_address"`
	DropoffAddress       string     `json:"dropoff_address"`
	CityID               *uuid.UUID `json:"city_id,omitempty"`
	EstimatedFare        float64    `json:"estimated_fare"`
	FairPriceMin         float64    `json:"fair_price_min"`
	FairPriceMax         float64    `json:"fair_price_max"`
	CurrencyCode         string     `json:"currency_code"`
	RiderInitialOffer    *float64   `json:"rider_initial_offer,omitempty"`
	ExpiresAt            time.Time  `json:"expires_at"`
	StartedAt            time.Time  `json:"started_at"`
}

// NegotiationOfferData is emitted when a driver makes an offer.
type NegotiationOfferData struct {
	OfferID             uuid.UUID `json:"offer_id"`
	SessionID           uuid.UUID `json:"session_id"`
	DriverID            uuid.UUID `json:"driver_id"`
	RiderID             uuid.UUID `json:"rider_id"`
	OfferedPrice        float64   `json:"offered_price"`
	CurrencyCode        string    `json:"currency_code"`
	EstimatedPickupTime *int      `json:"estimated_pickup_time,omitempty"`
	DriverRating        *float64  `json:"driver_rating,omitempty"`
	IsCounterOffer      bool      `json:"is_counter_offer"`
	CreatedAt           time.Time `json:"created_at"`
}

// NegotiationOfferAcceptedData is emitted when an offer is accepted.
type NegotiationOfferAcceptedData struct {
	SessionID    uuid.UUID `json:"session_id"`
	OfferID      uuid.UUID `json:"offer_id"`
	RiderID      uuid.UUID `json:"rider_id"`
	DriverID     uuid.UUID `json:"driver_id"`
	AcceptedPrice float64  `json:"accepted_price"`
	CurrencyCode string    `json:"currency_code"`
	AcceptedAt   time.Time `json:"accepted_at"`
}

// NegotiationExpiredData is emitted when a session expires.
type NegotiationExpiredData struct {
	SessionID    uuid.UUID `json:"session_id"`
	RiderID      uuid.UUID `json:"rider_id"`
	OffersCount  int       `json:"offers_count"`
	ExpiredAt    time.Time `json:"expired_at"`
}

// NegotiationCancelledData is emitted when a session is cancelled.
type NegotiationCancelledData struct {
	SessionID    uuid.UUID `json:"session_id"`
	RiderID      uuid.UUID `json:"rider_id"`
	Reason       string    `json:"reason"`
	CancelledAt  time.Time `json:"cancelled_at"`
}

// DriverInviteData is emitted to invite nearby drivers to a session.
type DriverInviteData struct {
	SessionID         uuid.UUID   `json:"session_id"`
	DriverIDs         []uuid.UUID `json:"driver_ids"`
	PickupLatitude    float64     `json:"pickup_latitude"`
	PickupLongitude   float64     `json:"pickup_longitude"`
	EstimatedFare     float64     `json:"estimated_fare"`
	FairPriceMin      float64     `json:"fair_price_min"`
	FairPriceMax      float64     `json:"fair_price_max"`
	ExpiresAt         time.Time   `json:"expires_at"`
}
