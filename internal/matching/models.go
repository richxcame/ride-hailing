package matching

import (
	"time"

	"github.com/google/uuid"
)

// RideRequestedEvent represents a ride request event from NATS
type RideRequestedEvent struct {
	RideID          uuid.UUID `json:"ride_id"`
	RiderID         uuid.UUID `json:"rider_id"`
	RiderName       string    `json:"rider_name"`
	RiderRating     float64   `json:"rider_rating"`
	PickupLatitude  float64   `json:"pickup_latitude"`
	PickupLongitude float64   `json:"pickup_longitude"`
	PickupAddress   string    `json:"pickup_address"`
	DropoffLatitude  *float64  `json:"dropoff_latitude,omitempty"`
	DropoffLongitude *float64  `json:"dropoff_longitude,omitempty"`
	DropoffAddress   *string   `json:"dropoff_address,omitempty"`
	RideTypeID      uuid.UUID `json:"ride_type_id"`
	RideTypeName    string    `json:"ride_type_name"`
	EstimatedFare   float64   `json:"estimated_fare"`
	EstimatedDistance float64 `json:"estimated_distance_km"`
	EstimatedDuration int     `json:"estimated_duration_minutes"`
	Currency        string    `json:"currency"`
	RequestedAt     time.Time `json:"requested_at"`
}

// RideAcceptedEvent represents a ride acceptance event
type RideAcceptedEvent struct {
	RideID     uuid.UUID `json:"ride_id"`
	DriverID   uuid.UUID `json:"driver_id"`
	DriverName string    `json:"driver_name"`
	AcceptedAt time.Time `json:"accepted_at"`
}

// RideCancelledEvent represents a ride cancellation event
type RideCancelledEvent struct {
	RideID      uuid.UUID `json:"ride_id"`
	CancelledBy uuid.UUID `json:"cancelled_by"`
	Reason      string    `json:"reason"`
	CancelledAt time.Time `json:"cancelled_at"`
}

// DriverLocation represents a driver's location from Redis geo
type DriverLocation struct {
	UserID    uuid.UUID
	DriverID  uuid.UUID
	Latitude  float64
	Longitude float64
	Distance  float64 // Distance to pickup in km
}

// RideOffer represents an offer sent to a driver
type RideOffer struct {
	RideID               uuid.UUID              `json:"ride_id"`
	RiderName            string                 `json:"rider_name"`
	RiderRating          float64                `json:"rider_rating"`
	PickupLocation       Location               `json:"pickup_location"`
	DropoffLocation      *Location              `json:"dropoff_location,omitempty"`
	RideType             string                 `json:"ride_type"`
	EstimatedFare        float64                `json:"estimated_fare"`
	EstimatedDistance    float64                `json:"estimated_distance_km"`
	EstimatedDuration    int                    `json:"estimated_duration_minutes"`
	DistanceToPickup     float64                `json:"distance_to_pickup_km"`
	ETAToPickup          int                    `json:"eta_to_pickup_minutes"`
	Currency             string                 `json:"currency"`
	ExpiresAt            time.Time              `json:"expires_at"`
	TimeoutSeconds       int                    `json:"timeout_seconds"`
}

// Location represents a geographic location
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address"`
}

// OfferTracking tracks pending offers in Redis
type OfferTracking struct {
	DriverID  uuid.UUID `json:"driver_id"`
	SentAt    time.Time `json:"sent_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// MatchingConfig holds configuration for the matching service
type MatchingConfig struct {
	SearchRadiusKm      float64       // Initial search radius in km
	MaxSearchRadiusKm   float64       // Maximum search radius if no drivers found
	RadiusIncrementKm   float64       // Increment radius by this amount
	MaxDriversToNotify  int           // Maximum number of drivers to notify
	OfferTimeoutSeconds int           // Timeout for driver to accept offer
	RetryDelaySeconds   int           // Delay before sending to next batch of drivers
	FirstBatchSize      int           // Number of drivers to notify first
}

// DefaultMatchingConfig returns default configuration
func DefaultMatchingConfig() MatchingConfig {
	return MatchingConfig{
		SearchRadiusKm:      5.0,  // Start with 5km radius
		MaxSearchRadiusKm:   20.0, // Max 20km radius
		RadiusIncrementKm:   5.0,  // Increase by 5km each time
		MaxDriversToNotify:  10,   // Notify up to 10 drivers
		OfferTimeoutSeconds: 30,   // 30 seconds to accept
		RetryDelaySeconds:   10,   // Wait 10 seconds before next batch
		FirstBatchSize:      3,    // Send to 3 closest drivers first
	}
}
