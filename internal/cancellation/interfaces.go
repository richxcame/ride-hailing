package cancellation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// Error definitions for the cancellation service
var (
	ErrNotAuthorized         = errors.New("not authorized for this ride")
	ErrNotAuthorizedToCancel = errors.New("not authorized to cancel this ride")
	ErrNotAuthorizedToView   = errors.New("not authorized to view this cancellation")
	ErrRideAlreadyFinished   = errors.New("ride already completed or cancelled")
	ErrRideNotFound          = errors.New("ride not found")
	ErrCancellationNotFound  = errors.New("cancellation record not found")
	ErrCreateRecordFailed    = errors.New("failed to create cancellation record")
	ErrUpdateRideFailed      = errors.New("failed to update ride status")
)

// RepositoryInterface defines the interface for the cancellation repository
// This enables mocking in tests
type RepositoryInterface interface {
	CreateCancellationRecord(ctx context.Context, rec *CancellationRecord) error
	GetCancellationByRideID(ctx context.Context, rideID uuid.UUID) (*CancellationRecord, error)
	GetCancellationByID(ctx context.Context, id uuid.UUID) (*CancellationRecord, error)
	CountUserCancellationsToday(ctx context.Context, userID uuid.UUID, cancelledBy CancelledBy) (int, error)
	CountUserCancellationsThisWeek(ctx context.Context, userID uuid.UUID, cancelledBy CancelledBy) (int, error)
	GetUserCancellationStats(ctx context.Context, userID uuid.UUID) (*UserCancellationStats, error)
	GetUserCancellationHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]CancellationRecord, int, error)
	WaiveCancellationFee(ctx context.Context, id uuid.UUID, reason string) error
	GetDefaultPolicy(ctx context.Context) (*CancellationPolicy, error)
	GetCancellationStats(ctx context.Context, from, to time.Time) (*CancellationStatsResponse, error)
}

// DBInterface defines the interface for database operations used by the service
// This enables mocking database calls in tests
type DBInterface interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (DBCommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) DBRow
}

// DBCommandTag represents the result of an Exec operation
type DBCommandTag interface {
	RowsAffected() int64
}

// DBRow represents a single row from a query
type DBRow interface {
	Scan(dest ...interface{}) error
}

// RideGetter defines an interface for retrieving ride information
type RideGetter interface {
	GetRide(ctx context.Context, rideID uuid.UUID) (*models.Ride, error)
}

// TestableService is a service wrapper that uses interfaces for testability
type TestableService struct {
	repo       RepositoryInterface
	db         DBInterface
	rideGetter RideGetter
}

// NewTestableService creates a new testable cancellation service
func NewTestableService(repo RepositoryInterface, db DBInterface, rideGetter RideGetter) *TestableService {
	return &TestableService{
		repo:       repo,
		db:         db,
		rideGetter: rideGetter,
	}
}

// PreviewCancellation shows what would happen if the user cancels
func (s *TestableService) PreviewCancellation(ctx context.Context, rideID, userID uuid.UUID) (*CancellationPreviewResponse, error) {
	ride, err := s.rideGetter.GetRide(ctx, rideID)
	if err != nil {
		return nil, err
	}

	isRider := ride.RiderID == userID
	isDriver := ride.DriverID != nil && *ride.DriverID == userID
	if !isRider && !isDriver {
		return nil, ErrNotAuthorized
	}

	if ride.Status == models.RideStatusCompleted || ride.Status == models.RideStatusCancelled {
		return nil, ErrRideAlreadyFinished
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
		str := string(*feeResult.WaiverReason)
		waiverStr = &str
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
func (s *TestableService) CancelRide(ctx context.Context, rideID, userID uuid.UUID, req *CancelRideRequest) (*CancelRideResponse, error) {
	ride, err := s.rideGetter.GetRide(ctx, rideID)
	if err != nil {
		return nil, err
	}

	isRider := ride.RiderID == userID
	isDriver := ride.DriverID != nil && *ride.DriverID == userID
	if !isRider && !isDriver {
		return nil, ErrNotAuthorizedToCancel
	}

	if ride.Status == models.RideStatusCompleted || ride.Status == models.RideStatusCancelled {
		return nil, ErrRideAlreadyFinished
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
		PickupLatitude:      ride.PickupLatitude,
		PickupLongitude:     ride.PickupLongitude,
		CancelledAt:         now,
		CreatedAt:           now,
	}

	if err := s.repo.CreateCancellationRecord(ctx, record); err != nil {
		return nil, ErrCreateRecordFailed
	}

	// Update ride status
	_, err = s.db.Exec(ctx, `
		UPDATE rides SET status = $1, cancelled_at = $2, cancellation_reason = $3, updated_at = $4
		WHERE id = $5`,
		models.RideStatusCancelled, now, string(req.ReasonCode), now, rideID,
	)
	if err != nil {
		return nil, ErrUpdateRideFailed
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
func (s *TestableService) GetCancellationDetails(ctx context.Context, rideID, userID uuid.UUID) (*CancellationRecord, error) {
	ride, err := s.rideGetter.GetRide(ctx, rideID)
	if err != nil {
		return nil, err
	}

	isRider := ride.RiderID == userID
	isDriver := ride.DriverID != nil && *ride.DriverID == userID
	if !isRider && !isDriver {
		return nil, ErrNotAuthorizedToView
	}

	rec, err := s.repo.GetCancellationByRideID(ctx, rideID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrCancellationNotFound
		}
		return nil, err
	}
	return rec, nil
}

// GetMyCancellationStats returns the current user's cancellation stats
func (s *TestableService) GetMyCancellationStats(ctx context.Context, userID uuid.UUID) (*UserCancellationStats, error) {
	stats, err := s.repo.GetUserCancellationStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	policy := s.getPolicy(ctx)
	stats.IsWarned = stats.CancellationsThisWeek >= policy.RiderPenaltyThreshold/2
	stats.IsPenalized = stats.CancellationsThisWeek >= policy.RiderPenaltyThreshold

	return stats, nil
}

// GetMyCancellationHistory returns the current user's cancellation history
func (s *TestableService) GetMyCancellationHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]CancellationRecord, int, error) {
	if limit < 1 || limit > 50 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.GetUserCancellationHistory(ctx, userID, limit, offset)
}

// GetCancellationReasons returns the available cancellation reasons for a user type
func (s *TestableService) GetCancellationReasons(isDriver bool) *CancellationReasonsResponse {
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

// WaiveFee allows an admin to waive a cancellation fee
func (s *TestableService) WaiveFee(ctx context.Context, cancellationID uuid.UUID, reason string) error {
	_, err := s.repo.GetCancellationByID(ctx, cancellationID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrCancellationNotFound
		}
		return err
	}
	return s.repo.WaiveCancellationFee(ctx, cancellationID, reason)
}

// GetCancellationStats returns platform-wide cancellation analytics
func (s *TestableService) GetCancellationStats(ctx context.Context, from, to time.Time) (*CancellationStatsResponse, error) {
	return s.repo.GetCancellationStats(ctx, from, to)
}

// GetUserCancellationStats returns cancellation stats for a specific user (admin view)
func (s *TestableService) GetUserCancellationStats(ctx context.Context, userID uuid.UUID) (*UserCancellationStats, error) {
	return s.repo.GetUserCancellationStats(ctx, userID)
}

// calculateFee determines the cancellation fee
func (s *TestableService) calculateFee(ctx context.Context, userID uuid.UUID, cancelledBy CancelledBy, minutesSinceRequest float64, ride *models.Ride, policy *CancellationPolicy) *CancellationFeeResult {
	// Driver cancellations are never charged a fee to the driver
	if cancelledBy == CancelledByDriver {
		return &CancellationFeeResult{
			FeeAmount:    0,
			FeeWaived:    true,
			WaiverReason: waivePtr(WaiverDriverFault),
			Explanation:  "Driver cancellations are not charged a fee",
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

	// Calculate base fee (simplified for testing)
	fee := 5.0 // Default fixed fee

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
func (s *TestableService) getPolicy(ctx context.Context) *CancellationPolicy {
	policy, err := s.repo.GetDefaultPolicy(ctx)
	if err != nil {
		return &defaultPolicy
	}
	return policy
}
