package ridehistory

import (
	"time"

	"github.com/google/uuid"
)

// RideHistoryEntry represents a past ride with full details
type RideHistoryEntry struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	RiderID          uuid.UUID  `json:"rider_id" db:"rider_id"`
	DriverID         *uuid.UUID `json:"driver_id,omitempty" db:"driver_id"`
	Status           string     `json:"status" db:"status"`

	// Route
	PickupAddress    string     `json:"pickup_address" db:"pickup_address"`
	PickupLatitude   float64    `json:"pickup_latitude" db:"pickup_latitude"`
	PickupLongitude  float64    `json:"pickup_longitude" db:"pickup_longitude"`
	DropoffAddress   string     `json:"dropoff_address" db:"dropoff_address"`
	DropoffLatitude  float64    `json:"dropoff_latitude" db:"dropoff_latitude"`
	DropoffLongitude float64    `json:"dropoff_longitude" db:"dropoff_longitude"`
	Distance         float64    `json:"distance_km" db:"distance"`
	Duration         int        `json:"duration_minutes" db:"duration"`

	// Pricing
	BaseFare         float64    `json:"base_fare" db:"base_fare"`
	DistanceFare     float64    `json:"distance_fare" db:"distance_fare"`
	TimeFare         float64    `json:"time_fare" db:"time_fare"`
	SurgeMultiplier  float64    `json:"surge_multiplier" db:"surge_multiplier"`
	SurgeAmount      float64    `json:"surge_amount" db:"surge_amount"`
	TollFees         float64    `json:"toll_fees" db:"toll_fees"`
	WaitTimeCharge   float64    `json:"wait_time_charge" db:"wait_time_charge"`
	TipAmount        float64    `json:"tip_amount" db:"tip_amount"`
	DiscountAmount   float64    `json:"discount_amount" db:"discount_amount"`
	PromoCode        *string    `json:"promo_code,omitempty" db:"promo_code"`
	TotalFare        float64    `json:"total_fare" db:"total_fare"`
	Currency         string     `json:"currency" db:"currency"`

	// Payment
	PaymentMethod    string     `json:"payment_method" db:"payment_method"`
	PaymentStatus    string     `json:"payment_status" db:"payment_status"`

	// Vehicle & Driver info
	DriverName       *string    `json:"driver_name,omitempty" db:"driver_name"`
	DriverPhoto      *string    `json:"driver_photo,omitempty" db:"driver_photo"`
	VehicleMake      *string    `json:"vehicle_make,omitempty" db:"vehicle_make"`
	VehicleModel     *string    `json:"vehicle_model,omitempty" db:"vehicle_model"`
	VehicleColor     *string    `json:"vehicle_color,omitempty" db:"vehicle_color"`
	LicensePlate     *string    `json:"license_plate,omitempty" db:"license_plate"`
	VehicleCategory  *string    `json:"vehicle_category,omitempty" db:"vehicle_category"`

	// Rating
	RiderRating      *int       `json:"rider_rating,omitempty" db:"rider_rating"`
	DriverRating     *int       `json:"driver_rating,omitempty" db:"driver_rating"`

	// Route polyline
	RoutePolyline    *string    `json:"route_polyline,omitempty" db:"route_polyline"`

	// Cancellation
	CancelledBy      *string    `json:"cancelled_by,omitempty" db:"cancelled_by"`
	CancelReason     *string    `json:"cancel_reason,omitempty" db:"cancel_reason"`
	CancellationFee  float64    `json:"cancellation_fee" db:"cancellation_fee"`

	// Timestamps
	RequestedAt      time.Time  `json:"requested_at" db:"requested_at"`
	AcceptedAt       *time.Time `json:"accepted_at,omitempty" db:"accepted_at"`
	PickedUpAt       *time.Time `json:"picked_up_at,omitempty" db:"picked_up_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CancelledAt      *time.Time `json:"cancelled_at,omitempty" db:"cancelled_at"`
}

// Receipt represents a detailed ride receipt
type Receipt struct {
	ReceiptID       string           `json:"receipt_id"`
	RideID          uuid.UUID        `json:"ride_id"`
	IssuedAt        time.Time        `json:"issued_at"`
	RiderName       string           `json:"rider_name"`
	RiderEmail      *string          `json:"rider_email,omitempty"`

	// Trip summary
	PickupAddress   string           `json:"pickup_address"`
	DropoffAddress  string           `json:"dropoff_address"`
	Distance        float64          `json:"distance_km"`
	Duration        int              `json:"duration_minutes"`
	TripDate        string           `json:"trip_date"`
	TripStartTime   string           `json:"trip_start_time"`
	TripEndTime     string           `json:"trip_end_time"`

	// Fare breakdown
	FareBreakdown   []FareLineItem   `json:"fare_breakdown"`
	Subtotal        float64          `json:"subtotal"`
	Discounts       float64          `json:"discounts"`
	Fees            float64          `json:"fees"`
	Tip             float64          `json:"tip"`
	Total           float64          `json:"total"`
	Currency        string           `json:"currency"`

	// Payment
	PaymentMethod   string           `json:"payment_method"`
	PaymentLast4    *string          `json:"payment_last4,omitempty"`

	// Driver
	DriverName      *string          `json:"driver_name,omitempty"`
	VehicleInfo     *string          `json:"vehicle_info,omitempty"`
	LicensePlate    *string          `json:"license_plate,omitempty"`

	// Map
	RouteMapURL     *string          `json:"route_map_url,omitempty"`
}

// FareLineItem represents a line in the fare breakdown
type FareLineItem struct {
	Label  string  `json:"label"`
	Amount float64 `json:"amount"`
	Type   string  `json:"type"` // charge, discount, fee, tip
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// HistoryFilters filters ride history
type HistoryFilters struct {
	Status   *string    `json:"status,omitempty"`
	FromDate *time.Time `json:"from_date,omitempty"`
	ToDate   *time.Time `json:"to_date,omitempty"`
	MinFare  *float64   `json:"min_fare,omitempty"`
	MaxFare  *float64   `json:"max_fare,omitempty"`
}

// HistoryResponse returns paginated ride history
type HistoryResponse struct {
	Rides    []RideHistoryEntry `json:"rides"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

// RideStats summarizes a user's ride activity
type RideStats struct {
	TotalRides       int     `json:"total_rides"`
	CompletedRides   int     `json:"completed_rides"`
	CancelledRides   int     `json:"cancelled_rides"`
	TotalSpent       float64 `json:"total_spent"`
	TotalDistance     float64 `json:"total_distance_km"`
	TotalDuration    int     `json:"total_duration_minutes"`
	AverageFare      float64 `json:"average_fare"`
	AverageDistance   float64 `json:"average_distance_km"`
	AverageRating    float64 `json:"average_rating_given"`
	FavoritePickup   *string `json:"favorite_pickup,omitempty"`
	FavoriteDropoff  *string `json:"favorite_dropoff,omitempty"`
	Currency         string  `json:"currency"`
	Period           string  `json:"period"`
}

// FrequentRoute represents a commonly taken route
type FrequentRoute struct {
	PickupAddress  string  `json:"pickup_address"`
	DropoffAddress string  `json:"dropoff_address"`
	RideCount      int     `json:"ride_count"`
	AverageFare    float64 `json:"average_fare"`
	LastRideAt     string  `json:"last_ride_at"`
}
