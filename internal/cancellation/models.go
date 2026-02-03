package cancellation

import (
	"time"

	"github.com/google/uuid"
)

// CancelledBy represents who cancelled the ride
type CancelledBy string

const (
	CancelledByRider  CancelledBy = "rider"
	CancelledByDriver CancelledBy = "driver"
	CancelledBySystem CancelledBy = "system"
)

// CancellationReasonCode represents predefined cancellation reasons
type CancellationReasonCode string

// Rider cancellation reasons
const (
	ReasonRiderChangedMind   CancellationReasonCode = "changed_mind"
	ReasonRiderDriverTooFar  CancellationReasonCode = "driver_too_far"
	ReasonRiderWaitTooLong   CancellationReasonCode = "wait_too_long"
	ReasonRiderWrongLocation CancellationReasonCode = "wrong_location"
	ReasonRiderPriceChanged  CancellationReasonCode = "price_changed"
	ReasonRiderFoundOther    CancellationReasonCode = "found_other_ride"
	ReasonRiderEmergency     CancellationReasonCode = "emergency"
	ReasonRiderOther         CancellationReasonCode = "other"
)

// Driver cancellation reasons
const (
	ReasonDriverRiderNoShow     CancellationReasonCode = "rider_no_show"
	ReasonDriverRiderUnreachable CancellationReasonCode = "rider_unreachable"
	ReasonDriverVehicleIssue    CancellationReasonCode = "vehicle_issue"
	ReasonDriverUnsafePickup    CancellationReasonCode = "unsafe_pickup"
	ReasonDriverTooFar          CancellationReasonCode = "too_far"
	ReasonDriverEmergency       CancellationReasonCode = "driver_emergency"
	ReasonDriverOther           CancellationReasonCode = "driver_other"
)

// System cancellation reasons
const (
	ReasonSystemNoDriver    CancellationReasonCode = "no_driver_found"
	ReasonSystemTimeout     CancellationReasonCode = "request_timeout"
	ReasonSystemPaymentFail CancellationReasonCode = "payment_failed"
	ReasonSystemFraud       CancellationReasonCode = "fraud_detected"
)

// FeeWaiverReason represents why a fee was waived
type FeeWaiverReason string

const (
	WaiverFreeCancellationWindow FeeWaiverReason = "free_cancellation_window"
	WaiverDriverFault            FeeWaiverReason = "driver_fault"
	WaiverSystemIssue            FeeWaiverReason = "system_issue"
	WaiverFirstCancellation      FeeWaiverReason = "first_cancellation"
	WaiverAdminOverride          FeeWaiverReason = "admin_override"
	WaiverPromoExemption         FeeWaiverReason = "promo_exemption"
)

// CancellationRecord represents a ride cancellation with full details
type CancellationRecord struct {
	ID               uuid.UUID              `json:"id" db:"id"`
	RideID           uuid.UUID              `json:"ride_id" db:"ride_id"`
	RiderID          uuid.UUID              `json:"rider_id" db:"rider_id"`
	DriverID         *uuid.UUID             `json:"driver_id,omitempty" db:"driver_id"`
	CancelledBy      CancelledBy            `json:"cancelled_by" db:"cancelled_by"`
	ReasonCode       CancellationReasonCode `json:"reason_code" db:"reason_code"`
	ReasonText       *string                `json:"reason_text,omitempty" db:"reason_text"`
	FeeAmount        float64                `json:"fee_amount" db:"fee_amount"`
	FeeWaived        bool                   `json:"fee_waived" db:"fee_waived"`
	WaiverReason     *FeeWaiverReason       `json:"waiver_reason,omitempty" db:"waiver_reason"`
	MinutesSinceRequest float64             `json:"minutes_since_request" db:"minutes_since_request"`
	MinutesSinceAccept  *float64            `json:"minutes_since_accept,omitempty" db:"minutes_since_accept"`
	RideStatus       string                 `json:"ride_status_at_cancel" db:"ride_status_at_cancel"`
	PickupLat        float64                `json:"pickup_lat" db:"pickup_lat"`
	PickupLng        float64                `json:"pickup_lng" db:"pickup_lng"`
	CancelledAt      time.Time              `json:"cancelled_at" db:"cancelled_at"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
}

// CancellationPolicy represents configurable cancellation rules
type CancellationPolicy struct {
	ID                     uuid.UUID `json:"id" db:"id"`
	Name                   string    `json:"name" db:"name"`
	FreeCancelWindowMinutes int      `json:"free_cancel_window_minutes" db:"free_cancel_window_minutes"`
	MaxFreeCancelsPerDay   int       `json:"max_free_cancels_per_day" db:"max_free_cancels_per_day"`
	MaxFreeCancelsPerWeek  int       `json:"max_free_cancels_per_week" db:"max_free_cancels_per_week"`
	DriverNoShowMinutes    int       `json:"driver_no_show_minutes" db:"driver_no_show_minutes"`
	RiderNoShowMinutes     int       `json:"rider_no_show_minutes" db:"rider_no_show_minutes"`
	DriverPenaltyThreshold int       `json:"driver_penalty_threshold" db:"driver_penalty_threshold"`
	RiderPenaltyThreshold  int       `json:"rider_penalty_threshold" db:"rider_penalty_threshold"`
	IsDefault              bool      `json:"is_default" db:"is_default"`
	IsActive               bool      `json:"is_active" db:"is_active"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`
}

// UserCancellationStats tracks a user's cancellation behavior
type UserCancellationStats struct {
	UserID                uuid.UUID `json:"user_id"`
	TotalCancellations    int       `json:"total_cancellations"`
	CancellationsToday    int       `json:"cancellations_today"`
	CancellationsThisWeek int       `json:"cancellations_this_week"`
	CancellationsThisMonth int      `json:"cancellations_this_month"`
	TotalFeesCharged      float64   `json:"total_fees_charged"`
	TotalFeesWaived       float64   `json:"total_fees_waived"`
	CancellationRate      float64   `json:"cancellation_rate"`
	LastCancellationAt    *time.Time `json:"last_cancellation_at,omitempty"`
	IsWarned              bool      `json:"is_warned"`
	IsPenalized           bool      `json:"is_penalized"`
}

// CancellationFeeResult represents the calculated fee for a cancellation
type CancellationFeeResult struct {
	FeeAmount    float64          `json:"fee_amount"`
	FeeWaived    bool             `json:"fee_waived"`
	WaiverReason *FeeWaiverReason `json:"waiver_reason,omitempty"`
	Explanation  string           `json:"explanation"`
}

// TopCancellationReason represents an aggregated reason count
type TopCancellationReason struct {
	ReasonCode CancellationReasonCode `json:"reason_code"`
	Count      int                    `json:"count"`
	Percentage float64                `json:"percentage"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// CancelRideRequest represents a cancellation request with structured reason
type CancelRideRequest struct {
	ReasonCode CancellationReasonCode `json:"reason_code" binding:"required"`
	ReasonText *string                `json:"reason_text,omitempty"`
}

// CancellationPreviewResponse shows what will happen if the user cancels
type CancellationPreviewResponse struct {
	FeeAmount           float64  `json:"fee_amount"`
	FeeWaived           bool     `json:"fee_waived"`
	WaiverReason        *string  `json:"waiver_reason,omitempty"`
	Explanation         string   `json:"explanation"`
	FreeCancelsRemaining int     `json:"free_cancels_remaining"`
	MinutesSinceRequest float64  `json:"minutes_since_request"`
	MinutesSinceAccept  *float64 `json:"minutes_since_accept,omitempty"`
}

// CancelRideResponse is the response after cancellation
type CancelRideResponse struct {
	RideID           uuid.UUID `json:"ride_id"`
	CancelledBy      string    `json:"cancelled_by"`
	FeeAmount        float64   `json:"fee_amount"`
	FeeWaived        bool      `json:"fee_waived"`
	WaiverReason     *string   `json:"waiver_reason,omitempty"`
	Explanation      string    `json:"explanation"`
	CancelledAt      time.Time `json:"cancelled_at"`
}

// WaiveFeeRequest is the admin request to waive a cancellation fee
type WaiveFeeRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// CancellationStatsResponse is the admin analytics response
type CancellationStatsResponse struct {
	TotalCancellations      int                     `json:"total_cancellations"`
	RiderCancellations      int                     `json:"rider_cancellations"`
	DriverCancellations     int                     `json:"driver_cancellations"`
	SystemCancellations     int                     `json:"system_cancellations"`
	TotalFeesCollected      float64                 `json:"total_fees_collected"`
	TotalFeesWaived         float64                 `json:"total_fees_waived"`
	AverageMinutesToCancel  float64                 `json:"average_minutes_to_cancel"`
	CancellationRate        float64                 `json:"cancellation_rate"`
	TopReasons              []TopCancellationReason `json:"top_reasons"`
}

// CancellationReasonsResponse lists available reasons for a user type
type CancellationReasonsResponse struct {
	Reasons []CancellationReasonOption `json:"reasons"`
}

// CancellationReasonOption represents a single selectable reason
type CancellationReasonOption struct {
	Code        CancellationReasonCode `json:"code"`
	Label       string                 `json:"label"`
	Description string                 `json:"description"`
}
