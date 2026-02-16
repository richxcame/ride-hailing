package pool

import (
	"time"

	"github.com/google/uuid"
)

// PoolStatus represents the status of a pool ride
type PoolStatus string

const (
	PoolStatusMatching   PoolStatus = "matching"   // Looking for passengers
	PoolStatusConfirmed  PoolStatus = "confirmed"  // All passengers confirmed
	PoolStatusInProgress PoolStatus = "in_progress"
	PoolStatusCompleted  PoolStatus = "completed"
	PoolStatusCancelled  PoolStatus = "cancelled"
)

// PassengerStatus represents the status of a passenger in a pool
type PassengerStatus string

const (
	PassengerStatusPending   PassengerStatus = "pending"   // Waiting for confirmation
	PassengerStatusConfirmed PassengerStatus = "confirmed" // Confirmed in pool
	PassengerStatusPickedUp  PassengerStatus = "picked_up"
	PassengerStatusDroppedOff PassengerStatus = "dropped_off"
	PassengerStatusCancelled PassengerStatus = "cancelled"
	PassengerStatusNoShow    PassengerStatus = "no_show"
)

// RouteMatchScore represents how well a route matches an existing pool
type RouteMatchScore struct {
	Score           float64 `json:"score"`
	DetourMinutes   float64 `json:"detour_minutes"`
	DetourKm        float64 `json:"detour_km"`
	DetourPercent   float64 `json:"detour_percent"` // % increase in route
	CostSavings     float64 `json:"cost_savings"`
	CostSavingsPercent float64 `json:"cost_savings_percent"`
}

// PoolRide represents a shared ride with multiple passengers
type PoolRide struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	DriverID        *uuid.UUID `json:"driver_id,omitempty" db:"driver_id"`
	VehicleID       *uuid.UUID `json:"vehicle_id,omitempty" db:"vehicle_id"`
	Status          PoolStatus `json:"status" db:"status"`
	MaxPassengers   int        `json:"max_passengers" db:"max_passengers"` // Vehicle capacity
	CurrentPassengers int      `json:"current_passengers" db:"current_passengers"`

	// Route optimization
	OptimizedRoute  []RouteStop `json:"optimized_route"`
	TotalDistance   float64     `json:"total_distance_km" db:"total_distance_km"`
	TotalDuration   int         `json:"total_duration_minutes" db:"total_duration_minutes"`

	// Geographic bounds for matching
	CenterLatitude  float64    `json:"center_latitude" db:"center_latitude"`
	CenterLongitude float64    `json:"center_longitude" db:"center_longitude"`
	RadiusKm        float64    `json:"radius_km" db:"radius_km"`
	H3Index         string     `json:"h3_index" db:"h3_index"` // For spatial matching

	// Pricing
	BaseFare        float64    `json:"base_fare" db:"base_fare"`
	PerKmRate       float64    `json:"per_km_rate" db:"per_km_rate"`
	PerMinuteRate   float64    `json:"per_minute_rate" db:"per_minute_rate"`

	// Timing
	MatchDeadline   time.Time  `json:"match_deadline" db:"match_deadline"`
	StartedAt       *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// RouteStop represents a stop on the pool route
type RouteStop struct {
	ID              uuid.UUID      `json:"id"`
	PoolPassengerID uuid.UUID      `json:"pool_passenger_id"`
	Type            RouteStopType  `json:"type"` // pickup or dropoff
	Location        Location       `json:"location"`
	Address         string         `json:"address"`
	SequenceOrder   int            `json:"sequence_order"`
	EstimatedArrival time.Time     `json:"estimated_arrival"`
	ActualArrival   *time.Time     `json:"actual_arrival,omitempty"`
	WaitTimeSeconds int            `json:"wait_time_seconds"`
}

// RouteStopType represents the type of stop
type RouteStopType string

const (
	RouteStopPickup  RouteStopType = "pickup"
	RouteStopDropoff RouteStopType = "dropoff"
)

// Location represents a geographic point
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// PoolPassenger represents a passenger in a pool ride
type PoolPassenger struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	PoolRideID      uuid.UUID       `json:"pool_ride_id" db:"pool_ride_id"`
	RiderID         uuid.UUID       `json:"rider_id" db:"rider_id"`
	Status          PassengerStatus `json:"status" db:"status"`

	// Ride details
	PickupLocation  Location        `json:"pickup_location"`
	DropoffLocation Location        `json:"dropoff_location"`
	PickupAddress   string          `json:"pickup_address" db:"pickup_address"`
	DropoffAddress  string          `json:"dropoff_address" db:"dropoff_address"`

	// Personal route (if they rode alone)
	DirectDistance  float64         `json:"direct_distance_km" db:"direct_distance_km"`
	DirectDuration  int             `json:"direct_duration_minutes" db:"direct_duration_minutes"`

	// Pool pricing
	OriginalFare    float64         `json:"original_fare" db:"original_fare"`
	PoolFare        float64         `json:"pool_fare" db:"pool_fare"`
	SavingsPercent  float64         `json:"savings_percent" db:"savings_percent"`

	// Timing
	PickedUpAt      *time.Time      `json:"picked_up_at,omitempty" db:"picked_up_at"`
	DroppedOffAt    *time.Time      `json:"dropped_off_at,omitempty" db:"dropped_off_at"`
	EstimatedPickup time.Time       `json:"estimated_pickup" db:"estimated_pickup"`
	EstimatedDropoff time.Time      `json:"estimated_dropoff" db:"estimated_dropoff"`

	// Rating
	RiderRating     *float64        `json:"rider_rating,omitempty" db:"rider_rating"`
	DriverRating    *float64        `json:"driver_rating,omitempty" db:"driver_rating"`

	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
}

// PoolConfig holds pool ride configuration
type PoolConfig struct {
	ID                    uuid.UUID `json:"id" db:"id"`
	CityID                *uuid.UUID `json:"city_id,omitempty" db:"city_id"`

	// Matching parameters
	MaxDetourPercent      float64   `json:"max_detour_percent" db:"max_detour_percent"` // Max % route increase
	MaxDetourMinutes      int       `json:"max_detour_minutes" db:"max_detour_minutes"`
	MaxWaitMinutes        int       `json:"max_wait_minutes" db:"max_wait_minutes"` // Max wait for match
	MaxPassengersPerRide  int       `json:"max_passengers_per_ride" db:"max_passengers_per_ride"`
	MinMatchScore         float64   `json:"min_match_score" db:"min_match_score"` // 0-1

	// Geographic
	MatchRadiusKm         float64   `json:"match_radius_km" db:"match_radius_km"`
	H3Resolution          int       `json:"h3_resolution" db:"h3_resolution"`

	// Pricing
	DiscountPercent       float64   `json:"discount_percent" db:"discount_percent"` // Base discount for pooling
	MinSavingsPercent     float64   `json:"min_savings_percent" db:"min_savings_percent"`

	// Features
	AllowDifferentDropoffs bool     `json:"allow_different_dropoffs" db:"allow_different_dropoffs"`
	AllowMidRoutePickup   bool      `json:"allow_mid_route_pickup" db:"allow_mid_route_pickup"`

	IsActive              bool      `json:"is_active" db:"is_active"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time `json:"updated_at" db:"updated_at"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// RequestPoolRideRequest represents a request for a pool ride
type RequestPoolRideRequest struct {
	PickupLocation  Location `json:"pickup_location" binding:"required"`
	DropoffLocation Location `json:"dropoff_location" binding:"required"`
	PickupAddress   string   `json:"pickup_address"`
	DropoffAddress  string   `json:"dropoff_address"`
	PassengerCount  int      `json:"passenger_count"` // Number of seats needed
	MaxWaitMinutes  int      `json:"max_wait_minutes,omitempty"` // Override default
}

// RequestPoolRideResponse represents the response after requesting a pool ride
type RequestPoolRideResponse struct {
	PoolPassengerID   uuid.UUID       `json:"pool_passenger_id"`
	PoolRideID        *uuid.UUID      `json:"pool_ride_id,omitempty"`
	Status            PassengerStatus `json:"status"`
	MatchFound        bool            `json:"match_found"`

	// Pricing
	OriginalFare      float64         `json:"original_fare"`
	PoolFare          float64         `json:"pool_fare"`
	SavingsPercent    float64         `json:"savings_percent"`

	// Timing
	EstimatedPickup   time.Time       `json:"estimated_pickup"`
	EstimatedDropoff  time.Time       `json:"estimated_dropoff"`
	MatchDeadline     time.Time       `json:"match_deadline"`

	// Match details
	MatchScore        *RouteMatchScore `json:"match_score,omitempty"`
	CurrentPassengers int              `json:"current_passengers"`
	MaxPassengers     int              `json:"max_passengers"`

	Message           string          `json:"message"`
}

// ConfirmPoolRequest confirms joining a pool
type ConfirmPoolRequest struct {
	PoolPassengerID uuid.UUID `json:"pool_passenger_id" binding:"required"`
	Accept          bool      `json:"accept" binding:"required"`
}

// GetPoolStatusResponse returns the status of a pool ride
type GetPoolStatusResponse struct {
	PoolRide        *PoolRide       `json:"pool_ride"`
	MyStatus        PassengerStatus `json:"my_status"`
	OtherPassengers []PassengerInfo `json:"other_passengers"`
	RouteStops      []RouteStop     `json:"route_stops"`
	EstimatedArrival time.Time      `json:"estimated_arrival"`
}

// PassengerInfo represents minimal info about other passengers (privacy)
type PassengerInfo struct {
	FirstNameInitial string    `json:"first_name_initial"`
	Rating           *float64  `json:"rating,omitempty"`
	PickupStop       int       `json:"pickup_stop"`  // Which stop number
	DropoffStop      int       `json:"dropoff_stop"` // Which stop number
}

// PoolMatchCandidate represents a potential pool match
type PoolMatchCandidate struct {
	PoolRideID        uuid.UUID       `json:"pool_ride_id"`
	MatchScore        RouteMatchScore `json:"match_score"`
	CurrentPassengers int             `json:"current_passengers"`
	MaxPassengers     int             `json:"max_passengers"`
	EstimatedPickup   time.Time       `json:"estimated_pickup"`
	EstimatedDropoff  time.Time       `json:"estimated_dropoff"`
	PoolFare          float64         `json:"pool_fare"`
	SavingsPercent    float64         `json:"savings_percent"`
}

// DriverPoolRideInfo represents pool ride info for drivers
type DriverPoolRideInfo struct {
	PoolRide     *PoolRide       `json:"pool_ride"`
	Passengers   []PoolPassenger `json:"passengers"`
	RouteStops   []RouteStop     `json:"route_stops"`
	NextStop     *RouteStop      `json:"next_stop"`
	TotalFare    float64         `json:"total_fare"`
	DriverEarnings float64       `json:"driver_earnings"`
}

// PoolStatsResponse represents pool ride statistics
type PoolStatsResponse struct {
	TotalPoolRides       int64   `json:"total_pool_rides"`
	ActivePoolRides      int     `json:"active_pool_rides"`
	AvgPassengersPerPool float64 `json:"avg_passengers_per_pool"`
	AvgSavingsPercent    float64 `json:"avg_savings_percent"`
	AvgMatchTime         float64 `json:"avg_match_time_seconds"`
	MatchSuccessRate     float64 `json:"match_success_rate"`
	TotalCO2Saved        float64 `json:"total_co2_saved_kg"` // Environmental impact
}
