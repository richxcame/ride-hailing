package cancellation

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richxcame/ride-hailing/internal/pricing"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// Default policy values when no database policy exists
var defaultPolicy = CancellationPolicy{
	FreeCancelWindowMinutes: 2,
	MaxFreeCancelsPerDay:    3,
	MaxFreeCancelsPerWeek:   10,
	DriverNoShowMinutes:     5,
	RiderNoShowMinutes:      5,
	DriverPenaltyThreshold:  20,
	RiderPenaltyThreshold:   30,
}

// Service handles cancellation business logic
type Service struct {
	repo *Repository
	db   *pgxpool.Pool
}

// NewService creates a new cancellation service
func NewService(repo *Repository, db *pgxpool.Pool) *Service {
	return &Service{repo: repo, db: db}
}

// PreviewCancellation shows what would happen if the user cancels
func (s *Service) PreviewCancellation(ctx context.Context, rideID, userID uuid.UUID) (*CancellationPreviewResponse, error) {
	ride, err := s.getRide(ctx, rideID)
	if err != nil {
		return nil, common.NewNotFoundError("ride not found", nil)
	}

	isRider := ride.RiderID == userID
	isDriver := ride.DriverID != nil && *ride.DriverID == userID
	if !isRider && !isDriver {
		return nil, common.NewForbiddenError("not authorized for this ride")
	}

	if ride.Status == models.RideStatusCompleted || ride.Status == models.RideStatusCancelled {
		return nil, common.NewBadRequestError("ride already completed or cancelled", nil)
	}

	cancelledBy := CancelledByRider
	if isDriver {
		cancelledBy = CancelledByDriver
	}

	policy := s.getPolicy(ctx)
	minutesSinceRequest := time.Since(ride.CreatedAt).Minutes()

	var minutesSinceAccept *float64
	if ride.AcceptedAt != nil {
		m := time.Since(*ride.AcceptedAt).Minutes()
		minutesSinceAccept = &m
	}

	feeResult := s.calculateFee(ctx, userID, cancelledBy, minutesSinceRequest, ride, policy)

	freeCancelsToday, _ := s.repo.CountUserCancellationsToday(ctx, userID, cancelledBy)
	remaining := policy.MaxFreeCancelsPerDay - freeCancelsToday
	if remaining < 0 {
		remaining = 0
	}

	var waiverStr *string
	if feeResult.WaiverReason != nil {
		s := string(*feeResult.WaiverReason)
		waiverStr = &s
	}

	return &CancellationPreviewResponse{
		FeeAmount:            feeResult.FeeAmount,
		FeeWaived:            feeResult.FeeWaived,
		WaiverReason:         waiverStr,
		Explanation:          feeResult.Explanation,
		FreeCancelsRemaining: remaining,
		MinutesSinceRequest:  minutesSinceRequest,
		MinutesSinceAccept:   minutesSinceAccept,
	}, nil
}

// CancelRide processes a ride cancellation with fee calculation
func (s *Service) CancelRide(ctx context.Context, rideID, userID uuid.UUID, req *CancelRideRequest) (*CancelRideResponse, error) {
	ride, err := s.getRide(ctx, rideID)
	if err != nil {
		return nil, common.NewNotFoundError("ride not found", nil)
	}

	isRider := ride.RiderID == userID
	isDriver := ride.DriverID != nil && *ride.DriverID == userID
	if !isRider && !isDriver {
		return nil, common.NewForbiddenError("not authorized to cancel this ride")
	}

	if ride.Status == models.RideStatusCompleted || ride.Status == models.RideStatusCancelled {
		return nil, common.NewBadRequestError("ride already completed or cancelled", nil)
	}

	cancelledBy := CancelledByRider
	if isDriver {
		cancelledBy = CancelledByDriver
	}

	policy := s.getPolicy(ctx)
	now := time.Now()
	minutesSinceRequest := now.Sub(ride.CreatedAt).Minutes()

	var minutesSinceAccept *float64
	if ride.AcceptedAt != nil {
		m := now.Sub(*ride.AcceptedAt).Minutes()
		minutesSinceAccept = &m
	}

	feeResult := s.calculateFee(ctx, userID, cancelledBy, minutesSinceRequest, ride, policy)

	record := &CancellationRecord{
		ID:                  uuid.New(),
		RideID:              rideID,
		RiderID:             ride.RiderID,
		DriverID:            ride.DriverID,
		CancelledBy:         cancelledBy,
		ReasonCode:          req.ReasonCode,
		ReasonText:          req.ReasonText,
		FeeAmount:           feeResult.FeeAmount,
		FeeWaived:           feeResult.FeeWaived,
		WaiverReason:        feeResult.WaiverReason,
		MinutesSinceRequest: minutesSinceRequest,
		MinutesSinceAccept:  minutesSinceAccept,
		RideStatus:          string(ride.Status),
		PickupLat:           ride.PickupLatitude,
		PickupLng:           ride.PickupLongitude,
		CancelledAt:         now,
		CreatedAt:           now,
	}

	if err := s.repo.CreateCancellationRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("create cancellation record: %w", err)
	}

	// Update ride status
	_, err = s.db.Exec(ctx, `
		UPDATE rides SET status = $1, cancelled_at = $2, cancellation_reason = $3, updated_at = $4
		WHERE id = $5`,
		models.RideStatusCancelled, now, string(req.ReasonCode), now, rideID,
	)
	if err != nil {
		return nil, fmt.Errorf("update ride status: %w", err)
	}

	var waiverStr *string
	if feeResult.WaiverReason != nil {
		ws := string(*feeResult.WaiverReason)
		waiverStr = &ws
	}

	return &CancelRideResponse{
		RideID:       rideID,
		CancelledBy:  string(cancelledBy),
		FeeAmount:    feeResult.FeeAmount,
		FeeWaived:    feeResult.FeeWaived,
		WaiverReason: waiverStr,
		Explanation:  feeResult.Explanation,
		CancelledAt:  now,
	}, nil
}

// GetCancellationDetails returns the cancellation details for a ride
func (s *Service) GetCancellationDetails(ctx context.Context, rideID, userID uuid.UUID) (*CancellationRecord, error) {
	ride, err := s.getRide(ctx, rideID)
	if err != nil {
		return nil, common.NewNotFoundError("ride not found", nil)
	}

	isRider := ride.RiderID == userID
	isDriver := ride.DriverID != nil && *ride.DriverID == userID
	if !isRider && !isDriver {
		return nil, common.NewForbiddenError("not authorized to view this cancellation")
	}

	rec, err := s.repo.GetCancellationByRideID(ctx, rideID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("cancellation record not found", nil)
		}
		return nil, err
	}
	return rec, nil
}

// GetMyCancellationStats returns the current user's cancellation stats
func (s *Service) GetMyCancellationStats(ctx context.Context, userID uuid.UUID) (*UserCancellationStats, error) {
	stats, err := s.repo.GetUserCancellationStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user cancellation stats: %w", err)
	}

	policy := s.getPolicy(ctx)
	stats.IsWarned = stats.CancellationsThisWeek >= policy.RiderPenaltyThreshold/2
	stats.IsPenalized = stats.CancellationsThisWeek >= policy.RiderPenaltyThreshold

	return stats, nil
}

// GetMyCancellationHistory returns the current user's cancellation history
func (s *Service) GetMyCancellationHistory(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]CancellationRecord, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.repo.GetUserCancellationHistory(ctx, userID, pageSize, offset)
}

// GetCancellationReasons returns the available cancellation reasons for a user type
func (s *Service) GetCancellationReasons(isDriver bool) *CancellationReasonsResponse {
	if isDriver {
		return &CancellationReasonsResponse{
			Reasons: []CancellationReasonOption{
				{Code: ReasonDriverRiderNoShow, Label: "Rider didn't show up", Description: "The rider was not at the pickup location"},
				{Code: ReasonDriverRiderUnreachable, Label: "Can't reach rider", Description: "Unable to contact the rider by phone or chat"},
				{Code: ReasonDriverVehicleIssue, Label: "Vehicle issue", Description: "My vehicle has a mechanical problem"},
				{Code: ReasonDriverUnsafePickup, Label: "Unsafe pickup location", Description: "The pickup location is unsafe or inaccessible"},
				{Code: ReasonDriverTooFar, Label: "Too far away", Description: "The pickup location is too far from my current location"},
				{Code: ReasonDriverEmergency, Label: "Emergency", Description: "I have a personal emergency"},
				{Code: ReasonDriverOther, Label: "Other reason", Description: "Another reason not listed above"},
			},
		}
	}

	return &CancellationReasonsResponse{
		Reasons: []CancellationReasonOption{
			{Code: ReasonRiderChangedMind, Label: "Changed my mind", Description: "I no longer need a ride"},
			{Code: ReasonRiderDriverTooFar, Label: "Driver is too far", Description: "The driver is taking too long to arrive"},
			{Code: ReasonRiderWaitTooLong, Label: "Wait time too long", Description: "I've been waiting too long for a match"},
			{Code: ReasonRiderWrongLocation, Label: "Wrong pickup location", Description: "I set the wrong pickup or destination"},
			{Code: ReasonRiderPriceChanged, Label: "Price changed", Description: "The fare is higher than expected"},
			{Code: ReasonRiderFoundOther, Label: "Found another ride", Description: "I found another way to get there"},
			{Code: ReasonRiderEmergency, Label: "Emergency", Description: "I have a personal emergency"},
			{Code: ReasonRiderOther, Label: "Other reason", Description: "Another reason not listed above"},
		},
	}
}

// ========================================
// ADMIN METHODS
// ========================================

// WaiveFee allows an admin to waive a cancellation fee
func (s *Service) WaiveFee(ctx context.Context, cancellationID uuid.UUID, reason string) error {
	_, err := s.repo.GetCancellationByID(ctx, cancellationID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return common.NewNotFoundError("cancellation not found", nil)
		}
		return err
	}
	return s.repo.WaiveCancellationFee(ctx, cancellationID, reason)
}

// GetCancellationStats returns platform-wide cancellation analytics
func (s *Service) GetCancellationStats(ctx context.Context, from, to time.Time) (*CancellationStatsResponse, error) {
	return s.repo.GetCancellationStats(ctx, from, to)
}

// GetUserCancellationStats returns cancellation stats for a specific user (admin view)
func (s *Service) GetUserCancellationStats(ctx context.Context, userID uuid.UUID) (*UserCancellationStats, error) {
	return s.repo.GetUserCancellationStats(ctx, userID)
}

// ========================================
// INTERNAL HELPERS
// ========================================

// calculateFee determines the cancellation fee
func (s *Service) calculateFee(ctx context.Context, userID uuid.UUID, cancelledBy CancelledBy, minutesSinceRequest float64, ride *models.Ride, policy *CancellationPolicy) *CancellationFeeResult {
	// Driver cancellations are never charged a fee to the driver
	if cancelledBy == CancelledByDriver {
		return &CancellationFeeResult{
			FeeAmount:   0,
			FeeWaived:   true,
			WaiverReason: waivePtr(WaiverDriverFault),
			Explanation: "Driver cancellations are not charged a fee",
		}
	}

	// Free cancellation window
	if minutesSinceRequest < float64(policy.FreeCancelWindowMinutes) {
		return &CancellationFeeResult{
			FeeAmount:    0,
			FeeWaived:    true,
			WaiverReason: waivePtr(WaiverFreeCancellationWindow),
			Explanation:  fmt.Sprintf("Free cancellation within %d minutes of requesting", policy.FreeCancelWindowMinutes),
		}
	}

	// Check if ride hasn't been accepted yet — lower fee
	if ride.Status == models.RideStatusRequested {
		return &CancellationFeeResult{
			FeeAmount:    0,
			FeeWaived:    true,
			WaiverReason: waivePtr(WaiverFreeCancellationWindow),
			Explanation:  "No fee when ride hasn't been accepted yet",
		}
	}

	// Calculate base fee from pricing engine
	fee := pricing.GetCancellationFee(&pricing.DefaultPricing, minutesSinceRequest, ride.EstimatedFare)

	// Check daily free cancel allowance
	todayCancels, _ := s.repo.CountUserCancellationsToday(ctx, userID, cancelledBy)
	if todayCancels < policy.MaxFreeCancelsPerDay {
		// Check weekly allowance too
		weekCancels, _ := s.repo.CountUserCancellationsThisWeek(ctx, userID, cancelledBy)
		if weekCancels < policy.MaxFreeCancelsPerWeek {
			return &CancellationFeeResult{
				FeeAmount:    fee,
				FeeWaived:    true,
				WaiverReason: waivePtr(WaiverFirstCancellation),
				Explanation:  fmt.Sprintf("Free cancellation (%d of %d daily limit used)", todayCancels+1, policy.MaxFreeCancelsPerDay),
			}
		}
	}

	return &CancellationFeeResult{
		FeeAmount:   fee,
		FeeWaived:   false,
		Explanation: fmt.Sprintf("Cancellation fee of %.2f applied (cancelled after %d minutes, daily free limit exceeded)", fee, int(minutesSinceRequest)),
	}
}

// getPolicy retrieves the active policy, falling back to defaults
func (s *Service) getPolicy(ctx context.Context) *CancellationPolicy {
	policy, err := s.repo.GetDefaultPolicy(ctx)
	if err != nil {
		return &defaultPolicy
	}
	return policy
}

// getRide fetches a ride by ID
func (s *Service) getRide(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
	ride := &models.Ride{}
	err := s.db.QueryRow(ctx, `
		SELECT id, rider_id, driver_id, status,
			pickup_latitude, pickup_longitude,
			estimated_fare, created_at, accepted_at
		FROM rides WHERE id = $1`, rideID,
	).Scan(
		&ride.ID, &ride.RiderID, &ride.DriverID, &ride.Status,
		&ride.PickupLatitude, &ride.PickupLongitude,
		&ride.EstimatedFare, &ride.CreatedAt, &ride.AcceptedAt,
	)
	if err != nil {
		return nil, err
	}
	return ride, nil
}

func waivePtr(r FeeWaiverReason) *FeeWaiverReason {
	return &r
}
