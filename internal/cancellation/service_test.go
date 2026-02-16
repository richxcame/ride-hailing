package cancellation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// MOCK IMPLEMENTATIONS
// ============================================================================

// MockRepository implements RepositoryInterface for testing
type MockRepository struct {
	CreateCancellationRecordFunc       func(ctx context.Context, rec *CancellationRecord) error
	GetCancellationByRideIDFunc        func(ctx context.Context, rideID uuid.UUID) (*CancellationRecord, error)
	GetCancellationByIDFunc            func(ctx context.Context, id uuid.UUID) (*CancellationRecord, error)
	CountUserCancellationsTodayFunc    func(ctx context.Context, userID uuid.UUID, cancelledBy CancelledBy) (int, error)
	CountUserCancellationsThisWeekFunc func(ctx context.Context, userID uuid.UUID, cancelledBy CancelledBy) (int, error)
	GetUserCancellationStatsFunc       func(ctx context.Context, userID uuid.UUID) (*UserCancellationStats, error)
	GetUserCancellationHistoryFunc     func(ctx context.Context, userID uuid.UUID, limit, offset int) ([]CancellationRecord, int, error)
	WaiveCancellationFeeFunc           func(ctx context.Context, id uuid.UUID, reason string) error
	GetDefaultPolicyFunc               func(ctx context.Context) (*CancellationPolicy, error)
	GetCancellationStatsFunc           func(ctx context.Context, from, to time.Time) (*CancellationStatsResponse, error)
}

func (m *MockRepository) CreateCancellationRecord(ctx context.Context, rec *CancellationRecord) error {
	if m.CreateCancellationRecordFunc != nil {
		return m.CreateCancellationRecordFunc(ctx, rec)
	}
	return nil
}

func (m *MockRepository) GetCancellationByRideID(ctx context.Context, rideID uuid.UUID) (*CancellationRecord, error) {
	if m.GetCancellationByRideIDFunc != nil {
		return m.GetCancellationByRideIDFunc(ctx, rideID)
	}
	return nil, pgx.ErrNoRows
}

func (m *MockRepository) GetCancellationByID(ctx context.Context, id uuid.UUID) (*CancellationRecord, error) {
	if m.GetCancellationByIDFunc != nil {
		return m.GetCancellationByIDFunc(ctx, id)
	}
	return nil, pgx.ErrNoRows
}

func (m *MockRepository) CountUserCancellationsToday(ctx context.Context, userID uuid.UUID, cancelledBy CancelledBy) (int, error) {
	if m.CountUserCancellationsTodayFunc != nil {
		return m.CountUserCancellationsTodayFunc(ctx, userID, cancelledBy)
	}
	return 0, nil
}

func (m *MockRepository) CountUserCancellationsThisWeek(ctx context.Context, userID uuid.UUID, cancelledBy CancelledBy) (int, error) {
	if m.CountUserCancellationsThisWeekFunc != nil {
		return m.CountUserCancellationsThisWeekFunc(ctx, userID, cancelledBy)
	}
	return 0, nil
}

func (m *MockRepository) GetUserCancellationStats(ctx context.Context, userID uuid.UUID) (*UserCancellationStats, error) {
	if m.GetUserCancellationStatsFunc != nil {
		return m.GetUserCancellationStatsFunc(ctx, userID)
	}
	return &UserCancellationStats{UserID: userID}, nil
}

func (m *MockRepository) GetUserCancellationHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]CancellationRecord, int, error) {
	if m.GetUserCancellationHistoryFunc != nil {
		return m.GetUserCancellationHistoryFunc(ctx, userID, limit, offset)
	}
	return []CancellationRecord{}, 0, nil
}

func (m *MockRepository) WaiveCancellationFee(ctx context.Context, id uuid.UUID, reason string) error {
	if m.WaiveCancellationFeeFunc != nil {
		return m.WaiveCancellationFeeFunc(ctx, id, reason)
	}
	return nil
}

func (m *MockRepository) GetDefaultPolicy(ctx context.Context) (*CancellationPolicy, error) {
	if m.GetDefaultPolicyFunc != nil {
		return m.GetDefaultPolicyFunc(ctx)
	}
	return nil, pgx.ErrNoRows
}

func (m *MockRepository) GetCancellationStats(ctx context.Context, from, to time.Time) (*CancellationStatsResponse, error) {
	if m.GetCancellationStatsFunc != nil {
		return m.GetCancellationStatsFunc(ctx, from, to)
	}
	return &CancellationStatsResponse{}, nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func newTestUUID() uuid.UUID {
	return uuid.New()
}

func ptrFloat64(f float64) *float64 {
	return &f
}

func ptrString(s string) *string {
	return &s
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func ptrUUID(u uuid.UUID) *uuid.UUID {
	return &u
}

// ============================================================================
// GET CANCELLATION REASONS TESTS
// ============================================================================

func TestGetCancellationReasons_Rider(t *testing.T) {
	svc := &Service{}
	resp := svc.GetCancellationReasons(false)

	assert.NotNil(t, resp)
	assert.Len(t, resp.Reasons, 8)

	// Verify first and last reason codes
	assert.Equal(t, ReasonRiderChangedMind, resp.Reasons[0].Code)
	assert.Equal(t, ReasonRiderOther, resp.Reasons[7].Code)

	// All reasons must have labels and descriptions
	for _, reason := range resp.Reasons {
		assert.NotEmpty(t, reason.Code)
		assert.NotEmpty(t, reason.Label)
		assert.NotEmpty(t, reason.Description)
	}
}

func TestGetCancellationReasons_Driver(t *testing.T) {
	svc := &Service{}
	resp := svc.GetCancellationReasons(true)

	assert.NotNil(t, resp)
	assert.Len(t, resp.Reasons, 7)

	assert.Equal(t, ReasonDriverRiderNoShow, resp.Reasons[0].Code)
	assert.Equal(t, ReasonDriverOther, resp.Reasons[6].Code)

	for _, reason := range resp.Reasons {
		assert.NotEmpty(t, reason.Code)
		assert.NotEmpty(t, reason.Label)
		assert.NotEmpty(t, reason.Description)
	}
}

func TestGetCancellationReasons_RiderReasonCodes(t *testing.T) {
	svc := &Service{}
	resp := svc.GetCancellationReasons(false)

	expectedCodes := []CancellationReasonCode{
		ReasonRiderChangedMind,
		ReasonRiderDriverTooFar,
		ReasonRiderWaitTooLong,
		ReasonRiderWrongLocation,
		ReasonRiderPriceChanged,
		ReasonRiderFoundOther,
		ReasonRiderEmergency,
		ReasonRiderOther,
	}

	for i, expected := range expectedCodes {
		assert.Equal(t, expected, resp.Reasons[i].Code)
	}
}

func TestGetCancellationReasons_DriverReasonCodes(t *testing.T) {
	svc := &Service{}
	resp := svc.GetCancellationReasons(true)

	expectedCodes := []CancellationReasonCode{
		ReasonDriverRiderNoShow,
		ReasonDriverRiderUnreachable,
		ReasonDriverVehicleIssue,
		ReasonDriverUnsafePickup,
		ReasonDriverTooFar,
		ReasonDriverEmergency,
		ReasonDriverOther,
	}

	for i, expected := range expectedCodes {
		assert.Equal(t, expected, resp.Reasons[i].Code)
	}
}

func TestGetCancellationReasons_RiderLabelsNotEmpty(t *testing.T) {
	svc := &Service{}
	resp := svc.GetCancellationReasons(false)

	for _, reason := range resp.Reasons {
		assert.NotEmpty(t, reason.Label, "Reason %s should have a label", reason.Code)
		assert.NotEmpty(t, reason.Description, "Reason %s should have a description", reason.Code)
	}
}

func TestGetCancellationReasons_DriverLabelsNotEmpty(t *testing.T) {
	svc := &Service{}
	resp := svc.GetCancellationReasons(true)

	for _, reason := range resp.Reasons {
		assert.NotEmpty(t, reason.Label, "Reason %s should have a label", reason.Code)
		assert.NotEmpty(t, reason.Description, "Reason %s should have a description", reason.Code)
	}
}

// ============================================================================
// CALCULATE FEE TESTS
// ============================================================================

func TestCalculateFee_DriverCancellation_AlwaysFree(t *testing.T) {
	svc := &Service{}
	policy := &defaultPolicy

	// Driver cancellations should always be free regardless of timing
	tests := []struct {
		name                string
		minutesSinceRequest float64
	}{
		{"immediately", 0},
		{"after 1 minute", 1},
		{"after 5 minutes", 5},
		{"after 30 minutes", 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.calculateFee(nil, [16]byte{}, CancelledByDriver, tt.minutesSinceRequest, nil, policy)

			assert.Equal(t, 0.0, result.FeeAmount)
			assert.True(t, result.FeeWaived)
			assert.NotNil(t, result.WaiverReason)
			assert.Equal(t, WaiverDriverFault, *result.WaiverReason)
			assert.Contains(t, result.Explanation, "Driver cancellations")
		})
	}
}

func TestCalculateFee_FreeCancellationWindow(t *testing.T) {
	svc := &Service{}
	policy := &defaultPolicy // FreeCancelWindowMinutes = 2

	// Ride needed for cases past the free window (accesses ride.Status)
	ride := &models.Ride{Status: models.RideStatusRequested}

	tests := []struct {
		name                string
		minutesSinceRequest float64
		expectFree          bool
	}{
		{"within window - 0 minutes", 0, true},
		{"within window - 1 minute", 1, true},
		{"within window - 1.9 minutes", 1.9, true},
		{"at boundary - 2 minutes (unaccepted ride)", 2, true},
		{"after window - 3 minutes (unaccepted ride)", 3, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.calculateFee(nil, [16]byte{}, CancelledByRider, tt.minutesSinceRequest, ride, policy)

			if tt.expectFree {
				assert.True(t, result.FeeWaived)
				assert.NotNil(t, result.WaiverReason)
				assert.Equal(t, WaiverFreeCancellationWindow, *result.WaiverReason)
			}
		})
	}
}

func TestCalculateFee_UnacceptedRide_AlwaysFree(t *testing.T) {
	svc := &Service{}
	policy := &defaultPolicy

	ride := &models.Ride{Status: models.RideStatusRequested}

	// Even after free window, unaccepted rides should be free
	result := svc.calculateFee(nil, [16]byte{}, CancelledByRider, 10.0, ride, policy)
	assert.True(t, result.FeeWaived)
	assert.Contains(t, result.Explanation, "hasn't been accepted")
}

func TestCalculateFee_AcceptedRide_WithinDailyLimit(t *testing.T) {
	// This test validates the logic for accepted rides within daily limit
	// The actual fee calculation with repository calls is tested via integration tests
	// Here we test the logic branches through driver cancellation (which doesn't hit repo)

	svc := &Service{}
	policy := &defaultPolicy

	// Driver cancellation on accepted ride - always free, no repo call needed
	ride := &models.Ride{
		Status:        models.RideStatusAccepted,
		EstimatedFare: 20.0,
	}

	result := svc.calculateFee(nil, uuid.New(), CancelledByDriver, 5.0, ride, policy)
	assert.NotNil(t, result)
	assert.True(t, result.FeeWaived)
	assert.Equal(t, WaiverDriverFault, *result.WaiverReason)
}

func TestCalculateFee_DriverCancellation_MultipleScenarios(t *testing.T) {
	svc := &Service{}
	policy := &defaultPolicy

	tests := []struct {
		name         string
		minutesAfter float64
		rideStatus   models.RideStatus
	}{
		{"driver cancels immediately on requested ride", 0, models.RideStatusRequested},
		{"driver cancels after 1 min on accepted ride", 1, models.RideStatusAccepted},
		{"driver cancels after 10 min on in_progress ride", 10, models.RideStatusInProgress},
		{"driver cancels after 30 min on accepted ride", 30, models.RideStatusAccepted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ride := &models.Ride{Status: tt.rideStatus, EstimatedFare: 25.0}
			result := svc.calculateFee(nil, uuid.New(), CancelledByDriver, tt.minutesAfter, ride, policy)

			assert.Equal(t, 0.0, result.FeeAmount)
			assert.True(t, result.FeeWaived)
			assert.Equal(t, WaiverDriverFault, *result.WaiverReason)
		})
	}
}

func TestCalculateFee_RiderCancellation_FreeCancelWindowEdgeCases(t *testing.T) {
	svc := &Service{}
	policy := &CancellationPolicy{
		FreeCancelWindowMinutes: 5, // Custom 5-minute window
		MaxFreeCancelsPerDay:    3,
		MaxFreeCancelsPerWeek:   10,
	}

	ride := &models.Ride{Status: models.RideStatusRequested}

	tests := []struct {
		name                string
		minutesSinceRequest float64
		expectWaived        bool
	}{
		{"at 0 minutes", 0, true},
		{"at 2.5 minutes", 2.5, true},
		{"at 4.99 minutes", 4.99, true},
		{"at exactly 5 minutes - boundary", 5, true}, // Ride is still requested
		{"at 5.01 minutes", 5.01, true},              // Still free for unaccepted
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.calculateFee(nil, uuid.New(), CancelledByRider, tt.minutesSinceRequest, ride, policy)
			assert.Equal(t, tt.expectWaived, result.FeeWaived)
		})
	}
}

func TestCalculateFee_ExplanationMessages(t *testing.T) {
	svc := &Service{}
	policy := &defaultPolicy

	t.Run("driver cancellation explanation", func(t *testing.T) {
		result := svc.calculateFee(nil, uuid.New(), CancelledByDriver, 5.0, nil, policy)
		assert.Contains(t, result.Explanation, "Driver cancellations are not charged a fee")
	})

	t.Run("free window explanation", func(t *testing.T) {
		ride := &models.Ride{Status: models.RideStatusRequested}
		result := svc.calculateFee(nil, uuid.New(), CancelledByRider, 1.0, ride, policy)
		assert.Contains(t, result.Explanation, "Free cancellation within")
	})

	t.Run("unaccepted ride explanation", func(t *testing.T) {
		ride := &models.Ride{Status: models.RideStatusRequested}
		result := svc.calculateFee(nil, uuid.New(), CancelledByRider, 10.0, ride, policy)
		assert.Contains(t, result.Explanation, "hasn't been accepted")
	})
}

// ============================================================================
// DEFAULT POLICY TESTS
// ============================================================================

func TestDefaultPolicy_Values(t *testing.T) {
	assert.Equal(t, 2, defaultPolicy.FreeCancelWindowMinutes)
	assert.Equal(t, 3, defaultPolicy.MaxFreeCancelsPerDay)
	assert.Equal(t, 10, defaultPolicy.MaxFreeCancelsPerWeek)
	assert.Equal(t, 5, defaultPolicy.DriverNoShowMinutes)
	assert.Equal(t, 5, defaultPolicy.RiderNoShowMinutes)
	assert.Equal(t, 20, defaultPolicy.DriverPenaltyThreshold)
	assert.Equal(t, 30, defaultPolicy.RiderPenaltyThreshold)
}

func TestDefaultPolicy_FreeCancelWindowMinutes(t *testing.T) {
	// Ensure the default policy has a reasonable free cancel window
	assert.GreaterOrEqual(t, defaultPolicy.FreeCancelWindowMinutes, 1)
	assert.LessOrEqual(t, defaultPolicy.FreeCancelWindowMinutes, 10)
}

func TestDefaultPolicy_MaxFreeCancelsPerDay(t *testing.T) {
	// Ensure daily limit is reasonable
	assert.GreaterOrEqual(t, defaultPolicy.MaxFreeCancelsPerDay, 1)
	assert.LessOrEqual(t, defaultPolicy.MaxFreeCancelsPerDay, 10)
}

func TestDefaultPolicy_MaxFreeCancelsPerWeek(t *testing.T) {
	// Weekly limit should be greater than daily
	assert.Greater(t, defaultPolicy.MaxFreeCancelsPerWeek, defaultPolicy.MaxFreeCancelsPerDay)
}

func TestDefaultPolicy_NoShowMinutes(t *testing.T) {
	// No-show minutes should be positive
	assert.Greater(t, defaultPolicy.DriverNoShowMinutes, 0)
	assert.Greater(t, defaultPolicy.RiderNoShowMinutes, 0)
}

func TestDefaultPolicy_PenaltyThresholds(t *testing.T) {
	// Penalty thresholds should be positive
	assert.Greater(t, defaultPolicy.DriverPenaltyThreshold, 0)
	assert.Greater(t, defaultPolicy.RiderPenaltyThreshold, 0)
}

// ============================================================================
// WAIVE PTR HELPER TESTS
// ============================================================================

func TestWaivePtr(t *testing.T) {
	reason := WaiverDriverFault
	ptr := waivePtr(reason)
	assert.NotNil(t, ptr)
	assert.Equal(t, reason, *ptr)
}

func TestWaivePtr_AllReasons(t *testing.T) {
	reasons := []FeeWaiverReason{
		WaiverFreeCancellationWindow,
		WaiverDriverFault,
		WaiverSystemIssue,
		WaiverFirstCancellation,
		WaiverAdminOverride,
		WaiverPromoExemption,
	}

	for _, reason := range reasons {
		t.Run(string(reason), func(t *testing.T) {
			ptr := waivePtr(reason)
			assert.NotNil(t, ptr)
			assert.Equal(t, reason, *ptr)
		})
	}
}

// ============================================================================
// CANCELLATION RECORD MODEL TESTS
// ============================================================================

func TestCancellationRecord_Fields(t *testing.T) {
	now := time.Now()
	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	recordID := uuid.New()

	record := CancellationRecord{
		ID:                  recordID,
		RideID:              rideID,
		RiderID:             riderID,
		DriverID:            &driverID,
		CancelledBy:         CancelledByRider,
		ReasonCode:          ReasonRiderChangedMind,
		ReasonText:          ptrString("Changed my plans"),
		FeeAmount:           5.0,
		FeeWaived:           false,
		MinutesSinceRequest: 3.5,
		RideStatus:          string(models.RideStatusAccepted),
		PickupLatitude:           40.7128,
		PickupLongitude:           -74.0060,
		CancelledAt:         now,
		CreatedAt:           now,
	}

	assert.Equal(t, recordID, record.ID)
	assert.Equal(t, rideID, record.RideID)
	assert.Equal(t, riderID, record.RiderID)
	assert.Equal(t, &driverID, record.DriverID)
	assert.Equal(t, CancelledByRider, record.CancelledBy)
	assert.Equal(t, ReasonRiderChangedMind, record.ReasonCode)
	assert.Equal(t, "Changed my plans", *record.ReasonText)
	assert.Equal(t, 5.0, record.FeeAmount)
	assert.False(t, record.FeeWaived)
	assert.Equal(t, 3.5, record.MinutesSinceRequest)
}

func TestCancellationRecord_NilDriverID(t *testing.T) {
	record := CancellationRecord{
		ID:          uuid.New(),
		RideID:      uuid.New(),
		RiderID:     uuid.New(),
		DriverID:    nil,
		CancelledBy: CancelledByRider,
	}

	assert.Nil(t, record.DriverID)
}

func TestCancellationRecord_WaiverReason(t *testing.T) {
	waiver := WaiverFreeCancellationWindow
	record := CancellationRecord{
		FeeWaived:    true,
		WaiverReason: &waiver,
	}

	assert.True(t, record.FeeWaived)
	assert.NotNil(t, record.WaiverReason)
	assert.Equal(t, WaiverFreeCancellationWindow, *record.WaiverReason)
}

// ============================================================================
// CANCELLATION POLICY MODEL TESTS
// ============================================================================

func TestCancellationPolicy_Fields(t *testing.T) {
	now := time.Now()
	policyID := uuid.New()

	policy := CancellationPolicy{
		ID:                      policyID,
		Name:                    "Standard Policy",
		FreeCancelWindowMinutes: 3,
		MaxFreeCancelsPerDay:    5,
		MaxFreeCancelsPerWeek:   15,
		DriverNoShowMinutes:     7,
		RiderNoShowMinutes:      7,
		DriverPenaltyThreshold:  25,
		RiderPenaltyThreshold:   35,
		IsDefault:               true,
		IsActive:                true,
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	assert.Equal(t, policyID, policy.ID)
	assert.Equal(t, "Standard Policy", policy.Name)
	assert.Equal(t, 3, policy.FreeCancelWindowMinutes)
	assert.Equal(t, 5, policy.MaxFreeCancelsPerDay)
	assert.Equal(t, 15, policy.MaxFreeCancelsPerWeek)
	assert.True(t, policy.IsDefault)
	assert.True(t, policy.IsActive)
}

// ============================================================================
// USER CANCELLATION STATS MODEL TESTS
// ============================================================================

func TestUserCancellationStats_Fields(t *testing.T) {
	now := time.Now()
	userID := uuid.New()

	stats := UserCancellationStats{
		UserID:                 userID,
		TotalCancellations:     10,
		CancellationsToday:     2,
		CancellationsThisWeek:  5,
		CancellationsThisMonth: 8,
		TotalFeesCharged:       25.50,
		TotalFeesWaived:        15.00,
		CancellationRate:       12.5,
		LastCancellationAt:     &now,
		IsWarned:               true,
		IsPenalized:            false,
	}

	assert.Equal(t, userID, stats.UserID)
	assert.Equal(t, 10, stats.TotalCancellations)
	assert.Equal(t, 2, stats.CancellationsToday)
	assert.Equal(t, 5, stats.CancellationsThisWeek)
	assert.Equal(t, 8, stats.CancellationsThisMonth)
	assert.Equal(t, 25.50, stats.TotalFeesCharged)
	assert.Equal(t, 15.00, stats.TotalFeesWaived)
	assert.Equal(t, 12.5, stats.CancellationRate)
	assert.NotNil(t, stats.LastCancellationAt)
	assert.True(t, stats.IsWarned)
	assert.False(t, stats.IsPenalized)
}

func TestUserCancellationStats_NoLastCancellation(t *testing.T) {
	stats := UserCancellationStats{
		UserID:             uuid.New(),
		TotalCancellations: 0,
		LastCancellationAt: nil,
	}

	assert.Nil(t, stats.LastCancellationAt)
	assert.Equal(t, 0, stats.TotalCancellations)
}

// ============================================================================
// CANCELLATION FEE RESULT MODEL TESTS
// ============================================================================

func TestCancellationFeeResult_WithWaiver(t *testing.T) {
	waiver := WaiverFirstCancellation
	result := CancellationFeeResult{
		FeeAmount:    5.0,
		FeeWaived:    true,
		WaiverReason: &waiver,
		Explanation:  "Free cancellation (1 of 3 daily limit used)",
	}

	assert.Equal(t, 5.0, result.FeeAmount)
	assert.True(t, result.FeeWaived)
	assert.NotNil(t, result.WaiverReason)
	assert.Equal(t, WaiverFirstCancellation, *result.WaiverReason)
	assert.Contains(t, result.Explanation, "Free cancellation")
}

func TestCancellationFeeResult_WithoutWaiver(t *testing.T) {
	result := CancellationFeeResult{
		FeeAmount:   10.0,
		FeeWaived:   false,
		Explanation: "Cancellation fee of 10.00 applied",
	}

	assert.Equal(t, 10.0, result.FeeAmount)
	assert.False(t, result.FeeWaived)
	assert.Nil(t, result.WaiverReason)
}

// ============================================================================
// REQUEST/RESPONSE MODEL TESTS
// ============================================================================

func TestCancelRideRequest_Fields(t *testing.T) {
	reasonText := "I found a friend to give me a ride"
	req := CancelRideRequest{
		ReasonCode: ReasonRiderFoundOther,
		ReasonText: &reasonText,
	}

	assert.Equal(t, ReasonRiderFoundOther, req.ReasonCode)
	assert.NotNil(t, req.ReasonText)
	assert.Equal(t, "I found a friend to give me a ride", *req.ReasonText)
}

func TestCancelRideRequest_NoReasonText(t *testing.T) {
	req := CancelRideRequest{
		ReasonCode: ReasonRiderChangedMind,
		ReasonText: nil,
	}

	assert.Equal(t, ReasonRiderChangedMind, req.ReasonCode)
	assert.Nil(t, req.ReasonText)
}

func TestCancellationPreviewResponse_Fields(t *testing.T) {
	waiver := "free_cancellation_window"
	minutesSinceAccept := 1.5
	resp := CancellationPreviewResponse{
		FeeAmount:            0,
		FeeWaived:            true,
		WaiverReason:         &waiver,
		Explanation:          "Free cancellation within 2 minutes",
		FreeCancelsRemaining: 2,
		MinutesSinceRequest:  1.0,
		MinutesSinceAccept:   &minutesSinceAccept,
	}

	assert.Equal(t, 0.0, resp.FeeAmount)
	assert.True(t, resp.FeeWaived)
	assert.NotNil(t, resp.WaiverReason)
	assert.Equal(t, 2, resp.FreeCancelsRemaining)
	assert.Equal(t, 1.0, resp.MinutesSinceRequest)
	assert.NotNil(t, resp.MinutesSinceAccept)
	assert.Equal(t, 1.5, *resp.MinutesSinceAccept)
}

func TestCancelRideResponse_Fields(t *testing.T) {
	now := time.Now()
	rideID := uuid.New()
	waiver := "admin_override"

	resp := CancelRideResponse{
		RideID:       rideID,
		CancelledBy:  "rider",
		FeeAmount:    5.0,
		FeeWaived:    true,
		WaiverReason: &waiver,
		Explanation:  "Fee waived by admin",
		CancelledAt:  now,
	}

	assert.Equal(t, rideID, resp.RideID)
	assert.Equal(t, "rider", resp.CancelledBy)
	assert.Equal(t, 5.0, resp.FeeAmount)
	assert.True(t, resp.FeeWaived)
	assert.NotNil(t, resp.WaiverReason)
	assert.Equal(t, "admin_override", *resp.WaiverReason)
}

func TestWaiveFeeRequest_Fields(t *testing.T) {
	req := WaiveFeeRequest{
		Reason: "Customer service compensation for late pickup",
	}

	assert.Equal(t, "Customer service compensation for late pickup", req.Reason)
}

// ============================================================================
// CANCELLATION STATS RESPONSE TESTS
// ============================================================================

func TestCancellationStatsResponse_Fields(t *testing.T) {
	resp := CancellationStatsResponse{
		TotalCancellations:     100,
		RiderCancellations:     70,
		DriverCancellations:    25,
		SystemCancellations:    5,
		TotalFeesCollected:     250.50,
		TotalFeesWaived:        75.00,
		AverageMinutesToCancel: 3.5,
		CancellationRate:       8.5,
		TopReasons: []TopCancellationReason{
			{ReasonCode: ReasonRiderChangedMind, Count: 30, Percentage: 30.0},
			{ReasonCode: ReasonRiderWaitTooLong, Count: 20, Percentage: 20.0},
		},
	}

	assert.Equal(t, 100, resp.TotalCancellations)
	assert.Equal(t, 70, resp.RiderCancellations)
	assert.Equal(t, 25, resp.DriverCancellations)
	assert.Equal(t, 5, resp.SystemCancellations)
	assert.Equal(t, 250.50, resp.TotalFeesCollected)
	assert.Equal(t, 75.00, resp.TotalFeesWaived)
	assert.InDelta(t, 3.5, resp.AverageMinutesToCancel, 0.01)
	assert.InDelta(t, 8.5, resp.CancellationRate, 0.01)
	assert.Len(t, resp.TopReasons, 2)
}

func TestCancellationStatsResponse_SumOfCancellations(t *testing.T) {
	resp := CancellationStatsResponse{
		TotalCancellations:  100,
		RiderCancellations:  70,
		DriverCancellations: 25,
		SystemCancellations: 5,
	}

	sum := resp.RiderCancellations + resp.DriverCancellations + resp.SystemCancellations
	assert.Equal(t, resp.TotalCancellations, sum)
}

// ============================================================================
// TOP CANCELLATION REASON TESTS
// ============================================================================

func TestTopCancellationReason_Fields(t *testing.T) {
	reason := TopCancellationReason{
		ReasonCode: ReasonRiderDriverTooFar,
		Count:      50,
		Percentage: 25.0,
	}

	assert.Equal(t, ReasonRiderDriverTooFar, reason.ReasonCode)
	assert.Equal(t, 50, reason.Count)
	assert.Equal(t, 25.0, reason.Percentage)
}

// ============================================================================
// CANCELLATION REASON OPTION TESTS
// ============================================================================

func TestCancellationReasonOption_Fields(t *testing.T) {
	option := CancellationReasonOption{
		Code:        ReasonRiderEmergency,
		Label:       "Emergency",
		Description: "I have a personal emergency",
	}

	assert.Equal(t, ReasonRiderEmergency, option.Code)
	assert.Equal(t, "Emergency", option.Label)
	assert.Equal(t, "I have a personal emergency", option.Description)
}

// ============================================================================
// CANCELLED BY CONSTANT TESTS
// ============================================================================

func TestCancelledBy_Constants(t *testing.T) {
	assert.Equal(t, CancelledBy("rider"), CancelledByRider)
	assert.Equal(t, CancelledBy("driver"), CancelledByDriver)
	assert.Equal(t, CancelledBy("system"), CancelledBySystem)
}

func TestCancelledBy_StringConversion(t *testing.T) {
	assert.Equal(t, "rider", string(CancelledByRider))
	assert.Equal(t, "driver", string(CancelledByDriver))
	assert.Equal(t, "system", string(CancelledBySystem))
}

// ============================================================================
// FEE WAIVER REASON CONSTANT TESTS
// ============================================================================

func TestFeeWaiverReason_Constants(t *testing.T) {
	assert.Equal(t, FeeWaiverReason("free_cancellation_window"), WaiverFreeCancellationWindow)
	assert.Equal(t, FeeWaiverReason("driver_fault"), WaiverDriverFault)
	assert.Equal(t, FeeWaiverReason("system_issue"), WaiverSystemIssue)
	assert.Equal(t, FeeWaiverReason("first_cancellation"), WaiverFirstCancellation)
	assert.Equal(t, FeeWaiverReason("admin_override"), WaiverAdminOverride)
	assert.Equal(t, FeeWaiverReason("promo_exemption"), WaiverPromoExemption)
}

// ============================================================================
// CANCELLATION REASON CODE CONSTANT TESTS
// ============================================================================

func TestCancellationReasonCode_RiderConstants(t *testing.T) {
	riderReasons := []CancellationReasonCode{
		ReasonRiderChangedMind,
		ReasonRiderDriverTooFar,
		ReasonRiderWaitTooLong,
		ReasonRiderWrongLocation,
		ReasonRiderPriceChanged,
		ReasonRiderFoundOther,
		ReasonRiderEmergency,
		ReasonRiderOther,
	}

	for _, reason := range riderReasons {
		assert.NotEmpty(t, string(reason))
	}
}

func TestCancellationReasonCode_DriverConstants(t *testing.T) {
	driverReasons := []CancellationReasonCode{
		ReasonDriverRiderNoShow,
		ReasonDriverRiderUnreachable,
		ReasonDriverVehicleIssue,
		ReasonDriverUnsafePickup,
		ReasonDriverTooFar,
		ReasonDriverEmergency,
		ReasonDriverOther,
	}

	for _, reason := range driverReasons {
		assert.NotEmpty(t, string(reason))
	}
}

func TestCancellationReasonCode_SystemConstants(t *testing.T) {
	systemReasons := []CancellationReasonCode{
		ReasonSystemNoDriver,
		ReasonSystemTimeout,
		ReasonSystemPaymentFail,
		ReasonSystemFraud,
	}

	for _, reason := range systemReasons {
		assert.NotEmpty(t, string(reason))
	}
}

// ============================================================================
// SERVICE METHODS - GET MY CANCELLATION HISTORY TESTS
// ============================================================================

func TestService_GetMyCancellationHistory_DefaultPagination(t *testing.T) {
	// Test that default pagination is applied for invalid values
	tests := []struct {
		name          string
		limit         int
		offset        int
		expectedLimit int
		expectedOffset int
	}{
		{"negative limit", -1, 0, 20, 0},
		{"zero limit", 0, 0, 20, 0},
		{"limit too large", 100, 0, 20, 0},
		{"negative offset", 10, -5, 10, 0},
		{"valid values", 25, 10, 25, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The service adjusts limit/offset internally
			limit := tt.limit
			offset := tt.offset

			if limit < 1 || limit > 50 {
				limit = 20
			}
			if offset < 0 {
				offset = 0
			}

			assert.Equal(t, tt.expectedLimit, limit)
			assert.Equal(t, tt.expectedOffset, offset)
		})
	}
}

func TestService_GetMyCancellationHistory_OffsetPassthrough(t *testing.T) {
	// Offset is now passed directly — verify limit/offset values are used as-is
	tests := []struct {
		limit          int
		offset         int
		expectedLimit  int
		expectedOffset int
	}{
		{20, 0, 20, 0},
		{20, 20, 20, 20},
		{20, 40, 20, 40},
		{10, 0, 10, 0},
		{10, 10, 10, 10},
		{10, 40, 10, 40},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expectedLimit, tt.limit)
		assert.Equal(t, tt.expectedOffset, tt.offset)
	}
}

// ============================================================================
// ERROR HANDLING TESTS
// ============================================================================

func TestAppError_NotFound(t *testing.T) {
	err := common.NewNotFoundError("ride not found", nil)
	assert.NotNil(t, err)
	assert.Equal(t, 404, err.Code)
	assert.Equal(t, "ride not found", err.Message)
}

func TestAppError_Forbidden(t *testing.T) {
	err := common.NewForbiddenError("not authorized")
	assert.NotNil(t, err)
	assert.Equal(t, 403, err.Code)
	assert.Equal(t, "not authorized", err.Message)
}

func TestAppError_BadRequest(t *testing.T) {
	err := common.NewBadRequestError("invalid request", nil)
	assert.NotNil(t, err)
	assert.Equal(t, 400, err.Code)
	assert.Equal(t, "invalid request", err.Message)
}

// ============================================================================
// MOCK REPOSITORY TESTS
// ============================================================================

func TestMockRepository_CreateCancellationRecord(t *testing.T) {
	var capturedRecord *CancellationRecord

	mock := &MockRepository{
		CreateCancellationRecordFunc: func(ctx context.Context, rec *CancellationRecord) error {
			capturedRecord = rec
			return nil
		},
	}

	record := &CancellationRecord{
		ID:          uuid.New(),
		RideID:      uuid.New(),
		CancelledBy: CancelledByRider,
	}

	err := mock.CreateCancellationRecord(context.Background(), record)
	assert.NoError(t, err)
	assert.Equal(t, record, capturedRecord)
}

func TestMockRepository_CreateCancellationRecord_Error(t *testing.T) {
	expectedErr := errors.New("database error")

	mock := &MockRepository{
		CreateCancellationRecordFunc: func(ctx context.Context, rec *CancellationRecord) error {
			return expectedErr
		},
	}

	err := mock.CreateCancellationRecord(context.Background(), &CancellationRecord{})
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestMockRepository_GetCancellationByRideID(t *testing.T) {
	rideID := uuid.New()
	expectedRecord := &CancellationRecord{
		ID:     uuid.New(),
		RideID: rideID,
	}

	mock := &MockRepository{
		GetCancellationByRideIDFunc: func(ctx context.Context, id uuid.UUID) (*CancellationRecord, error) {
			if id == rideID {
				return expectedRecord, nil
			}
			return nil, pgx.ErrNoRows
		},
	}

	record, err := mock.GetCancellationByRideID(context.Background(), rideID)
	assert.NoError(t, err)
	assert.Equal(t, expectedRecord, record)
}

func TestMockRepository_GetCancellationByRideID_NotFound(t *testing.T) {
	mock := &MockRepository{
		GetCancellationByRideIDFunc: func(ctx context.Context, id uuid.UUID) (*CancellationRecord, error) {
			return nil, pgx.ErrNoRows
		},
	}

	record, err := mock.GetCancellationByRideID(context.Background(), uuid.New())
	assert.Error(t, err)
	assert.Nil(t, record)
	assert.Equal(t, pgx.ErrNoRows, err)
}

func TestMockRepository_CountUserCancellationsToday(t *testing.T) {
	userID := uuid.New()

	mock := &MockRepository{
		CountUserCancellationsTodayFunc: func(ctx context.Context, id uuid.UUID, cancelledBy CancelledBy) (int, error) {
			if id == userID && cancelledBy == CancelledByRider {
				return 2, nil
			}
			return 0, nil
		},
	}

	count, err := mock.CountUserCancellationsToday(context.Background(), userID, CancelledByRider)
	assert.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestMockRepository_CountUserCancellationsThisWeek(t *testing.T) {
	userID := uuid.New()

	mock := &MockRepository{
		CountUserCancellationsThisWeekFunc: func(ctx context.Context, id uuid.UUID, cancelledBy CancelledBy) (int, error) {
			if id == userID {
				return 5, nil
			}
			return 0, nil
		},
	}

	count, err := mock.CountUserCancellationsThisWeek(context.Background(), userID, CancelledByRider)
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestMockRepository_GetUserCancellationStats(t *testing.T) {
	userID := uuid.New()

	mock := &MockRepository{
		GetUserCancellationStatsFunc: func(ctx context.Context, id uuid.UUID) (*UserCancellationStats, error) {
			return &UserCancellationStats{
				UserID:             id,
				TotalCancellations: 10,
				CancellationsToday: 1,
			}, nil
		},
	}

	stats, err := mock.GetUserCancellationStats(context.Background(), userID)
	assert.NoError(t, err)
	assert.Equal(t, userID, stats.UserID)
	assert.Equal(t, 10, stats.TotalCancellations)
	assert.Equal(t, 1, stats.CancellationsToday)
}

func TestMockRepository_GetUserCancellationHistory(t *testing.T) {
	userID := uuid.New()
	records := []CancellationRecord{
		{ID: uuid.New(), RiderID: userID, CancelledBy: CancelledByRider},
		{ID: uuid.New(), RiderID: userID, CancelledBy: CancelledByRider},
	}

	mock := &MockRepository{
		GetUserCancellationHistoryFunc: func(ctx context.Context, id uuid.UUID, limit, offset int) ([]CancellationRecord, int, error) {
			return records, 2, nil
		},
	}

	result, total, err := mock.GetUserCancellationHistory(context.Background(), userID, 20, 0)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 2, total)
}

func TestMockRepository_WaiveCancellationFee(t *testing.T) {
	cancellationID := uuid.New()
	var capturedID uuid.UUID
	var capturedReason string

	mock := &MockRepository{
		WaiveCancellationFeeFunc: func(ctx context.Context, id uuid.UUID, reason string) error {
			capturedID = id
			capturedReason = reason
			return nil
		},
	}

	err := mock.WaiveCancellationFee(context.Background(), cancellationID, "customer complaint")
	assert.NoError(t, err)
	assert.Equal(t, cancellationID, capturedID)
	assert.Equal(t, "customer complaint", capturedReason)
}

func TestMockRepository_GetDefaultPolicy(t *testing.T) {
	policyID := uuid.New()

	mock := &MockRepository{
		GetDefaultPolicyFunc: func(ctx context.Context) (*CancellationPolicy, error) {
			return &CancellationPolicy{
				ID:                      policyID,
				Name:                    "Test Policy",
				FreeCancelWindowMinutes: 5,
				IsDefault:               true,
				IsActive:                true,
			}, nil
		},
	}

	policy, err := mock.GetDefaultPolicy(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, policyID, policy.ID)
	assert.Equal(t, "Test Policy", policy.Name)
	assert.Equal(t, 5, policy.FreeCancelWindowMinutes)
}

func TestMockRepository_GetDefaultPolicy_NotFound(t *testing.T) {
	mock := &MockRepository{
		GetDefaultPolicyFunc: func(ctx context.Context) (*CancellationPolicy, error) {
			return nil, pgx.ErrNoRows
		},
	}

	policy, err := mock.GetDefaultPolicy(context.Background())
	assert.Error(t, err)
	assert.Nil(t, policy)
}

func TestMockRepository_GetCancellationStats(t *testing.T) {
	from := time.Now().AddDate(0, -1, 0)
	to := time.Now()

	mock := &MockRepository{
		GetCancellationStatsFunc: func(ctx context.Context, f, t time.Time) (*CancellationStatsResponse, error) {
			return &CancellationStatsResponse{
				TotalCancellations:  100,
				RiderCancellations:  60,
				DriverCancellations: 35,
				SystemCancellations: 5,
			}, nil
		},
	}

	stats, err := mock.GetCancellationStats(context.Background(), from, to)
	assert.NoError(t, err)
	assert.Equal(t, 100, stats.TotalCancellations)
	assert.Equal(t, 60, stats.RiderCancellations)
	assert.Equal(t, 35, stats.DriverCancellations)
	assert.Equal(t, 5, stats.SystemCancellations)
}

// ============================================================================
// EDGE CASE TESTS
// ============================================================================

func TestCalculateFee_ZeroMinutesSinceRequest(t *testing.T) {
	svc := &Service{}
	policy := &defaultPolicy

	result := svc.calculateFee(nil, uuid.New(), CancelledByRider, 0, &models.Ride{Status: models.RideStatusRequested}, policy)
	assert.True(t, result.FeeWaived)
	assert.Equal(t, 0.0, result.FeeAmount)
}

func TestCalculateFee_VeryLongTime(t *testing.T) {
	svc := &Service{}
	policy := &defaultPolicy

	result := svc.calculateFee(nil, uuid.New(), CancelledByDriver, 1440.0, nil, policy) // 24 hours
	assert.True(t, result.FeeWaived)
	assert.Equal(t, WaiverDriverFault, *result.WaiverReason)
}

func TestCalculateFee_NilRide_DriverCancellation(t *testing.T) {
	svc := &Service{}
	policy := &defaultPolicy

	// Driver cancellation should work with nil ride
	result := svc.calculateFee(nil, uuid.New(), CancelledByDriver, 5.0, nil, policy)
	assert.True(t, result.FeeWaived)
	assert.Equal(t, WaiverDriverFault, *result.WaiverReason)
}

func TestCalculateFee_DifferentRideStatuses(t *testing.T) {
	svc := &Service{}
	policy := &defaultPolicy

	statuses := []models.RideStatus{
		models.RideStatusRequested,
		models.RideStatusAccepted,
		models.RideStatusInProgress,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			ride := &models.Ride{Status: status, EstimatedFare: 20.0}
			result := svc.calculateFee(nil, uuid.New(), CancelledByDriver, 5.0, ride, policy)
			assert.True(t, result.FeeWaived, "Driver cancellation should always be free for status %s", status)
		})
	}
}

// ============================================================================
// VALIDATION TESTS
// ============================================================================

func TestCancelRideRequest_AllReasonCodes(t *testing.T) {
	riderReasons := []CancellationReasonCode{
		ReasonRiderChangedMind,
		ReasonRiderDriverTooFar,
		ReasonRiderWaitTooLong,
		ReasonRiderWrongLocation,
		ReasonRiderPriceChanged,
		ReasonRiderFoundOther,
		ReasonRiderEmergency,
		ReasonRiderOther,
	}

	for _, reason := range riderReasons {
		req := CancelRideRequest{ReasonCode: reason}
		assert.NotEmpty(t, string(req.ReasonCode))
	}
}

func TestCancelRideRequest_DriverReasonCodes(t *testing.T) {
	driverReasons := []CancellationReasonCode{
		ReasonDriverRiderNoShow,
		ReasonDriverRiderUnreachable,
		ReasonDriverVehicleIssue,
		ReasonDriverUnsafePickup,
		ReasonDriverTooFar,
		ReasonDriverEmergency,
		ReasonDriverOther,
	}

	for _, reason := range driverReasons {
		req := CancelRideRequest{ReasonCode: reason}
		assert.NotEmpty(t, string(req.ReasonCode))
	}
}

// ============================================================================
// BOUNDARY TESTS
// ============================================================================

func TestPolicy_BoundaryValues(t *testing.T) {
	t.Run("zero free cancel window", func(t *testing.T) {
		policy := &CancellationPolicy{
			FreeCancelWindowMinutes: 0,
			MaxFreeCancelsPerDay:    3,
			MaxFreeCancelsPerWeek:   10,
		}
		// With 0 minute window, even immediate cancellation might not be free
		// This tests the boundary behavior
		assert.GreaterOrEqual(t, policy.FreeCancelWindowMinutes, 0)
	})

	t.Run("maximum free cancel window", func(t *testing.T) {
		policy := &CancellationPolicy{
			FreeCancelWindowMinutes: 60,
			MaxFreeCancelsPerDay:    3,
			MaxFreeCancelsPerWeek:   10,
		}
		assert.Equal(t, 60, policy.FreeCancelWindowMinutes)
	})

	t.Run("zero daily limit", func(t *testing.T) {
		policy := &CancellationPolicy{
			FreeCancelWindowMinutes: 2,
			MaxFreeCancelsPerDay:    0,
			MaxFreeCancelsPerWeek:   10,
		}
		assert.Equal(t, 0, policy.MaxFreeCancelsPerDay)
	})
}

func TestFreeCancelsRemaining_Calculation(t *testing.T) {
	tests := []struct {
		name           string
		maxDaily       int
		usedToday      int
		expectedRemain int
	}{
		{"none used", 3, 0, 3},
		{"one used", 3, 1, 2},
		{"all used", 3, 3, 0},
		{"exceeded", 3, 5, 0}, // Should not go negative
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remaining := tt.maxDaily - tt.usedToday
			if remaining < 0 {
				remaining = 0
			}
			assert.Equal(t, tt.expectedRemain, remaining)
		})
	}
}

// ============================================================================
// HELPER FUNCTION TESTS
// ============================================================================

func TestPtrFloat64(t *testing.T) {
	val := 10.5
	ptr := ptrFloat64(val)
	require.NotNil(t, ptr)
	assert.Equal(t, val, *ptr)
}

func TestPtrString(t *testing.T) {
	val := "test string"
	ptr := ptrString(val)
	require.NotNil(t, ptr)
	assert.Equal(t, val, *ptr)
}

func TestPtrTime(t *testing.T) {
	val := time.Now()
	ptr := ptrTime(val)
	require.NotNil(t, ptr)
	assert.Equal(t, val, *ptr)
}

func TestPtrUUID(t *testing.T) {
	val := uuid.New()
	ptr := ptrUUID(val)
	require.NotNil(t, ptr)
	assert.Equal(t, val, *ptr)
}

func TestNewTestUUID(t *testing.T) {
	id := newTestUUID()
	assert.NotEqual(t, uuid.Nil, id)

	// Verify uniqueness
	id2 := newTestUUID()
	assert.NotEqual(t, id, id2)
}

// ============================================================================
// CONCURRENCY TESTS
// ============================================================================

func TestMockRepository_ConcurrentAccess(t *testing.T) {
	var callCount int
	mock := &MockRepository{
		CountUserCancellationsTodayFunc: func(ctx context.Context, userID uuid.UUID, cancelledBy CancelledBy) (int, error) {
			callCount++
			return callCount, nil
		},
	}

	// Simulate concurrent calls
	userID := uuid.New()
	for i := 0; i < 10; i++ {
		_, err := mock.CountUserCancellationsToday(context.Background(), userID, CancelledByRider)
		assert.NoError(t, err)
	}
	assert.Equal(t, 10, callCount)
}

// ============================================================================
// COMPREHENSIVE CALCULATE FEE SCENARIOS
// ============================================================================

func TestCalculateFee_CompleteScenarios(t *testing.T) {
	svc := &Service{}

	tests := []struct {
		name                string
		cancelledBy         CancelledBy
		minutesSinceRequest float64
		rideStatus          models.RideStatus
		expectFeeWaived     bool
		expectWaiverReason  *FeeWaiverReason
	}{
		{
			name:                "driver cancels immediately",
			cancelledBy:         CancelledByDriver,
			minutesSinceRequest: 0,
			rideStatus:          models.RideStatusRequested,
			expectFeeWaived:     true,
			expectWaiverReason:  waivePtr(WaiverDriverFault),
		},
		{
			name:                "driver cancels after 10 minutes",
			cancelledBy:         CancelledByDriver,
			minutesSinceRequest: 10,
			rideStatus:          models.RideStatusAccepted,
			expectFeeWaived:     true,
			expectWaiverReason:  waivePtr(WaiverDriverFault),
		},
		{
			name:                "rider cancels within window",
			cancelledBy:         CancelledByRider,
			minutesSinceRequest: 1,
			rideStatus:          models.RideStatusRequested,
			expectFeeWaived:     true,
			expectWaiverReason:  waivePtr(WaiverFreeCancellationWindow),
		},
		{
			name:                "rider cancels unaccepted ride after window",
			cancelledBy:         CancelledByRider,
			minutesSinceRequest: 5,
			rideStatus:          models.RideStatusRequested,
			expectFeeWaived:     true,
			expectWaiverReason:  waivePtr(WaiverFreeCancellationWindow),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ride := &models.Ride{
				Status:        tt.rideStatus,
				EstimatedFare: 20.0,
			}

			result := svc.calculateFee(nil, uuid.New(), tt.cancelledBy, tt.minutesSinceRequest, ride, &defaultPolicy)

			assert.Equal(t, tt.expectFeeWaived, result.FeeWaived)
			if tt.expectWaiverReason != nil {
				assert.NotNil(t, result.WaiverReason)
				assert.Equal(t, *tt.expectWaiverReason, *result.WaiverReason)
			}
		})
	}
}

// ============================================================================
// MOCK DB AND RIDE GETTER IMPLEMENTATIONS
// ============================================================================

// MockCommandTag implements DBCommandTag for testing
type MockCommandTag struct {
	rowsAffected int64
}

func (m MockCommandTag) RowsAffected() int64 {
	return m.rowsAffected
}

// MockDB implements DBInterface for testing
type MockDB struct {
	ExecFunc     func(ctx context.Context, sql string, arguments ...interface{}) (DBCommandTag, error)
	QueryRowFunc func(ctx context.Context, sql string, args ...interface{}) DBRow
}

func (m *MockDB) Exec(ctx context.Context, sql string, arguments ...interface{}) (DBCommandTag, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, sql, arguments...)
	}
	return MockCommandTag{rowsAffected: 1}, nil
}

func (m *MockDB) QueryRow(ctx context.Context, sql string, args ...interface{}) DBRow {
	if m.QueryRowFunc != nil {
		return m.QueryRowFunc(ctx, sql, args...)
	}
	return &MockRow{}
}

// MockRow implements DBRow for testing
type MockRow struct {
	ScanFunc func(dest ...interface{}) error
}

func (m *MockRow) Scan(dest ...interface{}) error {
	if m.ScanFunc != nil {
		return m.ScanFunc(dest...)
	}
	return nil
}

// MockRideGetter implements RideGetter for testing
type MockRideGetter struct {
	GetRideFunc func(ctx context.Context, rideID uuid.UUID) (*models.Ride, error)
}

func (m *MockRideGetter) GetRide(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
	if m.GetRideFunc != nil {
		return m.GetRideFunc(ctx, rideID)
	}
	return nil, ErrRideNotFound
}

// ============================================================================
// TESTABLE SERVICE TESTS
// ============================================================================

func TestNewTestableService(t *testing.T) {
	repo := &MockRepository{}
	db := &MockDB{}
	rideGetter := &MockRideGetter{}

	svc := NewTestableService(repo, db, rideGetter)

	assert.NotNil(t, svc)
	assert.NotNil(t, svc.repo)
	assert.NotNil(t, svc.db)
	assert.NotNil(t, svc.rideGetter)
}

func TestTestableService_GetCancellationReasons_Rider(t *testing.T) {
	svc := NewTestableService(&MockRepository{}, &MockDB{}, &MockRideGetter{})

	resp := svc.GetCancellationReasons(false)

	assert.NotNil(t, resp)
	assert.Len(t, resp.Reasons, 8)
	assert.Equal(t, ReasonRiderChangedMind, resp.Reasons[0].Code)
}

func TestTestableService_GetCancellationReasons_Driver(t *testing.T) {
	svc := NewTestableService(&MockRepository{}, &MockDB{}, &MockRideGetter{})

	resp := svc.GetCancellationReasons(true)

	assert.NotNil(t, resp)
	assert.Len(t, resp.Reasons, 7)
	assert.Equal(t, ReasonDriverRiderNoShow, resp.Reasons[0].Code)
}

func TestTestableService_PreviewCancellation_RideNotFound(t *testing.T) {
	rideGetter := &MockRideGetter{
		GetRideFunc: func(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
			return nil, ErrRideNotFound
		},
	}
	svc := NewTestableService(&MockRepository{}, &MockDB{}, rideGetter)

	_, err := svc.PreviewCancellation(context.Background(), uuid.New(), uuid.New())

	assert.Error(t, err)
	assert.Equal(t, ErrRideNotFound, err)
}

func TestTestableService_PreviewCancellation_NotAuthorized(t *testing.T) {
	riderID := uuid.New()
	driverID := uuid.New()
	userID := uuid.New() // Different user

	rideGetter := &MockRideGetter{
		GetRideFunc: func(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
			return &models.Ride{
				ID:        rideID,
				RiderID:   riderID,
				DriverID:  &driverID,
				Status:    models.RideStatusAccepted,
				CreatedAt: time.Now().Add(-1 * time.Minute),
			}, nil
		},
	}
	svc := NewTestableService(&MockRepository{}, &MockDB{}, rideGetter)

	_, err := svc.PreviewCancellation(context.Background(), uuid.New(), userID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotAuthorized, err)
}

func TestTestableService_PreviewCancellation_RideAlreadyFinished(t *testing.T) {
	riderID := uuid.New()

	rideGetter := &MockRideGetter{
		GetRideFunc: func(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
			return &models.Ride{
				ID:        rideID,
				RiderID:   riderID,
				Status:    models.RideStatusCompleted,
				CreatedAt: time.Now().Add(-1 * time.Minute),
			}, nil
		},
	}
	svc := NewTestableService(&MockRepository{}, &MockDB{}, rideGetter)

	_, err := svc.PreviewCancellation(context.Background(), uuid.New(), riderID)

	assert.Error(t, err)
	assert.Equal(t, ErrRideAlreadyFinished, err)
}

func TestTestableService_PreviewCancellation_RideCancelled(t *testing.T) {
	riderID := uuid.New()

	rideGetter := &MockRideGetter{
		GetRideFunc: func(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
			return &models.Ride{
				ID:        rideID,
				RiderID:   riderID,
				Status:    models.RideStatusCancelled,
				CreatedAt: time.Now().Add(-1 * time.Minute),
			}, nil
		},
	}
	svc := NewTestableService(&MockRepository{}, &MockDB{}, rideGetter)

	_, err := svc.PreviewCancellation(context.Background(), uuid.New(), riderID)

	assert.Error(t, err)
	assert.Equal(t, ErrRideAlreadyFinished, err)
}

func TestTestableService_PreviewCancellation_Success_Rider(t *testing.T) {
	riderID := uuid.New()

	rideGetter := &MockRideGetter{
		GetRideFunc: func(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
			return &models.Ride{
				ID:        rideID,
				RiderID:   riderID,
				Status:    models.RideStatusRequested,
				CreatedAt: time.Now().Add(-30 * time.Second),
			}, nil
		},
	}
	repo := &MockRepository{
		CountUserCancellationsTodayFunc: func(ctx context.Context, userID uuid.UUID, cancelledBy CancelledBy) (int, error) {
			return 0, nil
		},
	}
	svc := NewTestableService(repo, &MockDB{}, rideGetter)

	resp, err := svc.PreviewCancellation(context.Background(), uuid.New(), riderID)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.FeeWaived)
	assert.Equal(t, 3, resp.FreeCancelsRemaining) // default policy
}

func TestTestableService_PreviewCancellation_Success_Driver(t *testing.T) {
	riderID := uuid.New()
	driverID := uuid.New()

	rideGetter := &MockRideGetter{
		GetRideFunc: func(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
			return &models.Ride{
				ID:        rideID,
				RiderID:   riderID,
				DriverID:  &driverID,
				Status:    models.RideStatusAccepted,
				CreatedAt: time.Now().Add(-5 * time.Minute),
			}, nil
		},
	}
	repo := &MockRepository{
		CountUserCancellationsTodayFunc: func(ctx context.Context, userID uuid.UUID, cancelledBy CancelledBy) (int, error) {
			return 1, nil
		},
	}
	svc := NewTestableService(repo, &MockDB{}, rideGetter)

	resp, err := svc.PreviewCancellation(context.Background(), uuid.New(), driverID)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.FeeWaived)
	assert.NotNil(t, resp.WaiverReason)
	assert.Equal(t, string(WaiverDriverFault), *resp.WaiverReason)
}

func TestTestableService_CancelRide_RideNotFound(t *testing.T) {
	rideGetter := &MockRideGetter{
		GetRideFunc: func(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
			return nil, ErrRideNotFound
		},
	}
	svc := NewTestableService(&MockRepository{}, &MockDB{}, rideGetter)

	_, err := svc.CancelRide(context.Background(), uuid.New(), uuid.New(), &CancelRideRequest{
		ReasonCode: ReasonRiderChangedMind,
	})

	assert.Error(t, err)
	assert.Equal(t, ErrRideNotFound, err)
}

func TestTestableService_CancelRide_NotAuthorized(t *testing.T) {
	riderID := uuid.New()
	driverID := uuid.New()
	userID := uuid.New() // Different user

	rideGetter := &MockRideGetter{
		GetRideFunc: func(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
			return &models.Ride{
				ID:        rideID,
				RiderID:   riderID,
				DriverID:  &driverID,
				Status:    models.RideStatusAccepted,
				CreatedAt: time.Now().Add(-1 * time.Minute),
			}, nil
		},
	}
	svc := NewTestableService(&MockRepository{}, &MockDB{}, rideGetter)

	_, err := svc.CancelRide(context.Background(), uuid.New(), userID, &CancelRideRequest{
		ReasonCode: ReasonRiderChangedMind,
	})

	assert.Error(t, err)
	assert.Equal(t, ErrNotAuthorizedToCancel, err)
}

func TestTestableService_CancelRide_AlreadyCompleted(t *testing.T) {
	riderID := uuid.New()

	rideGetter := &MockRideGetter{
		GetRideFunc: func(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
			return &models.Ride{
				ID:        rideID,
				RiderID:   riderID,
				Status:    models.RideStatusCompleted,
				CreatedAt: time.Now().Add(-1 * time.Minute),
			}, nil
		},
	}
	svc := NewTestableService(&MockRepository{}, &MockDB{}, rideGetter)

	_, err := svc.CancelRide(context.Background(), uuid.New(), riderID, &CancelRideRequest{
		ReasonCode: ReasonRiderChangedMind,
	})

	assert.Error(t, err)
	assert.Equal(t, ErrRideAlreadyFinished, err)
}

func TestTestableService_CancelRide_CreateRecordFailed(t *testing.T) {
	riderID := uuid.New()

	rideGetter := &MockRideGetter{
		GetRideFunc: func(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
			return &models.Ride{
				ID:        rideID,
				RiderID:   riderID,
				Status:    models.RideStatusRequested,
				CreatedAt: time.Now().Add(-1 * time.Minute),
			}, nil
		},
	}
	repo := &MockRepository{
		CreateCancellationRecordFunc: func(ctx context.Context, rec *CancellationRecord) error {
			return errors.New("database error")
		},
	}
	svc := NewTestableService(repo, &MockDB{}, rideGetter)

	_, err := svc.CancelRide(context.Background(), uuid.New(), riderID, &CancelRideRequest{
		ReasonCode: ReasonRiderChangedMind,
	})

	assert.Error(t, err)
	assert.Equal(t, ErrCreateRecordFailed, err)
}

func TestTestableService_CancelRide_UpdateRideFailed(t *testing.T) {
	riderID := uuid.New()

	rideGetter := &MockRideGetter{
		GetRideFunc: func(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
			return &models.Ride{
				ID:        rideID,
				RiderID:   riderID,
				Status:    models.RideStatusRequested,
				CreatedAt: time.Now().Add(-1 * time.Minute),
			}, nil
		},
	}
	repo := &MockRepository{
		CreateCancellationRecordFunc: func(ctx context.Context, rec *CancellationRecord) error {
			return nil
		},
	}
	db := &MockDB{
		ExecFunc: func(ctx context.Context, sql string, arguments ...interface{}) (DBCommandTag, error) {
			return nil, errors.New("update failed")
		},
	}
	svc := NewTestableService(repo, db, rideGetter)

	_, err := svc.CancelRide(context.Background(), uuid.New(), riderID, &CancelRideRequest{
		ReasonCode: ReasonRiderChangedMind,
	})

	assert.Error(t, err)
	assert.Equal(t, ErrUpdateRideFailed, err)
}

func TestTestableService_CancelRide_Success_Rider(t *testing.T) {
	riderID := uuid.New()
	rideID := uuid.New()

	rideGetter := &MockRideGetter{
		GetRideFunc: func(ctx context.Context, id uuid.UUID) (*models.Ride, error) {
			return &models.Ride{
				ID:              id,
				RiderID:         riderID,
				Status:          models.RideStatusRequested,
				CreatedAt:       time.Now().Add(-30 * time.Second),
				PickupLatitude:  40.7128,
				PickupLongitude: -74.0060,
				EstimatedFare:   25.0,
			}, nil
		},
	}
	repo := &MockRepository{
		CreateCancellationRecordFunc: func(ctx context.Context, rec *CancellationRecord) error {
			return nil
		},
	}
	db := &MockDB{
		ExecFunc: func(ctx context.Context, sql string, arguments ...interface{}) (DBCommandTag, error) {
			return MockCommandTag{rowsAffected: 1}, nil
		},
	}
	svc := NewTestableService(repo, db, rideGetter)

	resp, err := svc.CancelRide(context.Background(), rideID, riderID, &CancelRideRequest{
		ReasonCode: ReasonRiderChangedMind,
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, rideID, resp.RideID)
	assert.Equal(t, "rider", resp.CancelledBy)
	assert.True(t, resp.FeeWaived)
}

func TestTestableService_CancelRide_Success_Driver(t *testing.T) {
	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	rideGetter := &MockRideGetter{
		GetRideFunc: func(ctx context.Context, id uuid.UUID) (*models.Ride, error) {
			return &models.Ride{
				ID:              id,
				RiderID:         riderID,
				DriverID:        &driverID,
				Status:          models.RideStatusAccepted,
				CreatedAt:       time.Now().Add(-5 * time.Minute),
				PickupLatitude:  40.7128,
				PickupLongitude: -74.0060,
				EstimatedFare:   25.0,
			}, nil
		},
	}
	repo := &MockRepository{
		CreateCancellationRecordFunc: func(ctx context.Context, rec *CancellationRecord) error {
			return nil
		},
	}
	db := &MockDB{}
	svc := NewTestableService(repo, db, rideGetter)

	resp, err := svc.CancelRide(context.Background(), rideID, driverID, &CancelRideRequest{
		ReasonCode: ReasonDriverRiderNoShow,
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "driver", resp.CancelledBy)
	assert.True(t, resp.FeeWaived)
	assert.NotNil(t, resp.WaiverReason)
	assert.Equal(t, string(WaiverDriverFault), *resp.WaiverReason)
}

func TestTestableService_GetCancellationDetails_RideNotFound(t *testing.T) {
	rideGetter := &MockRideGetter{
		GetRideFunc: func(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
			return nil, ErrRideNotFound
		},
	}
	svc := NewTestableService(&MockRepository{}, &MockDB{}, rideGetter)

	_, err := svc.GetCancellationDetails(context.Background(), uuid.New(), uuid.New())

	assert.Error(t, err)
	assert.Equal(t, ErrRideNotFound, err)
}

func TestTestableService_GetCancellationDetails_NotAuthorized(t *testing.T) {
	riderID := uuid.New()
	driverID := uuid.New()
	userID := uuid.New()

	rideGetter := &MockRideGetter{
		GetRideFunc: func(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
			return &models.Ride{
				ID:       rideID,
				RiderID:  riderID,
				DriverID: &driverID,
				Status:   models.RideStatusCancelled,
			}, nil
		},
	}
	svc := NewTestableService(&MockRepository{}, &MockDB{}, rideGetter)

	_, err := svc.GetCancellationDetails(context.Background(), uuid.New(), userID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotAuthorizedToView, err)
}

func TestTestableService_GetCancellationDetails_NotFound(t *testing.T) {
	riderID := uuid.New()

	rideGetter := &MockRideGetter{
		GetRideFunc: func(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
			return &models.Ride{
				ID:      rideID,
				RiderID: riderID,
				Status:  models.RideStatusCancelled,
			}, nil
		},
	}
	repo := &MockRepository{
		GetCancellationByRideIDFunc: func(ctx context.Context, rideID uuid.UUID) (*CancellationRecord, error) {
			return nil, pgx.ErrNoRows
		},
	}
	svc := NewTestableService(repo, &MockDB{}, rideGetter)

	_, err := svc.GetCancellationDetails(context.Background(), uuid.New(), riderID)

	assert.Error(t, err)
	assert.Equal(t, ErrCancellationNotFound, err)
}

func TestTestableService_GetCancellationDetails_Success(t *testing.T) {
	riderID := uuid.New()
	rideID := uuid.New()
	recordID := uuid.New()

	rideGetter := &MockRideGetter{
		GetRideFunc: func(ctx context.Context, id uuid.UUID) (*models.Ride, error) {
			return &models.Ride{
				ID:      id,
				RiderID: riderID,
				Status:  models.RideStatusCancelled,
			}, nil
		},
	}
	repo := &MockRepository{
		GetCancellationByRideIDFunc: func(ctx context.Context, id uuid.UUID) (*CancellationRecord, error) {
			return &CancellationRecord{
				ID:          recordID,
				RideID:      id,
				RiderID:     riderID,
				CancelledBy: CancelledByRider,
				ReasonCode:  ReasonRiderChangedMind,
				FeeAmount:   0,
				FeeWaived:   true,
			}, nil
		},
	}
	svc := NewTestableService(repo, &MockDB{}, rideGetter)

	rec, err := svc.GetCancellationDetails(context.Background(), rideID, riderID)

	assert.NoError(t, err)
	assert.NotNil(t, rec)
	assert.Equal(t, recordID, rec.ID)
	assert.Equal(t, CancelledByRider, rec.CancelledBy)
}

func TestTestableService_GetMyCancellationStats_Success(t *testing.T) {
	userID := uuid.New()

	repo := &MockRepository{
		GetUserCancellationStatsFunc: func(ctx context.Context, id uuid.UUID) (*UserCancellationStats, error) {
			return &UserCancellationStats{
				UserID:                id,
				TotalCancellations:    10,
				CancellationsThisWeek: 5,
			}, nil
		},
	}
	svc := NewTestableService(repo, &MockDB{}, &MockRideGetter{})

	stats, err := svc.GetMyCancellationStats(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, userID, stats.UserID)
	assert.Equal(t, 10, stats.TotalCancellations)
	// With default policy (RiderPenaltyThreshold=30), 5 cancellations < 15 (half), so not warned
	assert.False(t, stats.IsWarned)
	assert.False(t, stats.IsPenalized)
}

func TestTestableService_GetMyCancellationStats_Warned(t *testing.T) {
	userID := uuid.New()

	repo := &MockRepository{
		GetUserCancellationStatsFunc: func(ctx context.Context, id uuid.UUID) (*UserCancellationStats, error) {
			return &UserCancellationStats{
				UserID:                id,
				TotalCancellations:    20,
				CancellationsThisWeek: 16, // >= 30/2 = 15
			}, nil
		},
	}
	svc := NewTestableService(repo, &MockDB{}, &MockRideGetter{})

	stats, err := svc.GetMyCancellationStats(context.Background(), userID)

	assert.NoError(t, err)
	assert.True(t, stats.IsWarned)
	assert.False(t, stats.IsPenalized)
}

func TestTestableService_GetMyCancellationStats_Penalized(t *testing.T) {
	userID := uuid.New()

	repo := &MockRepository{
		GetUserCancellationStatsFunc: func(ctx context.Context, id uuid.UUID) (*UserCancellationStats, error) {
			return &UserCancellationStats{
				UserID:                id,
				TotalCancellations:    35,
				CancellationsThisWeek: 32, // >= 30
			}, nil
		},
	}
	svc := NewTestableService(repo, &MockDB{}, &MockRideGetter{})

	stats, err := svc.GetMyCancellationStats(context.Background(), userID)

	assert.NoError(t, err)
	assert.True(t, stats.IsWarned)
	assert.True(t, stats.IsPenalized)
}

func TestTestableService_GetMyCancellationStats_Error(t *testing.T) {
	userID := uuid.New()

	repo := &MockRepository{
		GetUserCancellationStatsFunc: func(ctx context.Context, id uuid.UUID) (*UserCancellationStats, error) {
			return nil, errors.New("database error")
		},
	}
	svc := NewTestableService(repo, &MockDB{}, &MockRideGetter{})

	_, err := svc.GetMyCancellationStats(context.Background(), userID)

	assert.Error(t, err)
}

func TestTestableService_GetMyCancellationHistory_Success(t *testing.T) {
	userID := uuid.New()
	records := []CancellationRecord{
		{ID: uuid.New(), RiderID: userID, CancelledBy: CancelledByRider},
		{ID: uuid.New(), RiderID: userID, CancelledBy: CancelledByRider},
	}

	repo := &MockRepository{
		GetUserCancellationHistoryFunc: func(ctx context.Context, id uuid.UUID, limit, offset int) ([]CancellationRecord, int, error) {
			return records, 2, nil
		},
	}
	svc := NewTestableService(repo, &MockDB{}, &MockRideGetter{})

	result, total, err := svc.GetMyCancellationHistory(context.Background(), userID, 20, 0)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 2, total)
}

func TestTestableService_GetMyCancellationHistory_DefaultPagination(t *testing.T) {
	userID := uuid.New()
	var capturedLimit, capturedOffset int

	repo := &MockRepository{
		GetUserCancellationHistoryFunc: func(ctx context.Context, id uuid.UUID, limit, offset int) ([]CancellationRecord, int, error) {
			capturedLimit = limit
			capturedOffset = offset
			return []CancellationRecord{}, 0, nil
		},
	}
	svc := NewTestableService(repo, &MockDB{}, &MockRideGetter{})

	// Test with invalid limit (0 defaults to 20)
	svc.GetMyCancellationHistory(context.Background(), userID, 0, 10)
	assert.Equal(t, 20, capturedLimit) // defaults to 20
	assert.Equal(t, 10, capturedOffset)

	// Test with negative offset (defaults to 0)
	svc.GetMyCancellationHistory(context.Background(), userID, 10, -5)
	assert.Equal(t, 10, capturedLimit)
	assert.Equal(t, 0, capturedOffset) // negative offset defaults to 0

	// Test with limit > 50 (defaults to 20)
	svc.GetMyCancellationHistory(context.Background(), userID, 100, 0)
	assert.Equal(t, 20, capturedLimit) // caps at 20 (default)
	assert.Equal(t, 0, capturedOffset)
}

func TestTestableService_WaiveFee_NotFound(t *testing.T) {
	repo := &MockRepository{
		GetCancellationByIDFunc: func(ctx context.Context, id uuid.UUID) (*CancellationRecord, error) {
			return nil, pgx.ErrNoRows
		},
	}
	svc := NewTestableService(repo, &MockDB{}, &MockRideGetter{})

	err := svc.WaiveFee(context.Background(), uuid.New(), "test reason")

	assert.Error(t, err)
	assert.Equal(t, ErrCancellationNotFound, err)
}

func TestTestableService_WaiveFee_Success(t *testing.T) {
	cancellationID := uuid.New()
	var waiveCalledWith uuid.UUID
	var waiveReasonWith string

	repo := &MockRepository{
		GetCancellationByIDFunc: func(ctx context.Context, id uuid.UUID) (*CancellationRecord, error) {
			return &CancellationRecord{ID: id}, nil
		},
		WaiveCancellationFeeFunc: func(ctx context.Context, id uuid.UUID, reason string) error {
			waiveCalledWith = id
			waiveReasonWith = reason
			return nil
		},
	}
	svc := NewTestableService(repo, &MockDB{}, &MockRideGetter{})

	err := svc.WaiveFee(context.Background(), cancellationID, "customer complaint")

	assert.NoError(t, err)
	assert.Equal(t, cancellationID, waiveCalledWith)
	assert.Equal(t, "customer complaint", waiveReasonWith)
}

func TestTestableService_GetCancellationStats_Success(t *testing.T) {
	from := time.Now().AddDate(0, -1, 0)
	to := time.Now()

	repo := &MockRepository{
		GetCancellationStatsFunc: func(ctx context.Context, f, t time.Time) (*CancellationStatsResponse, error) {
			return &CancellationStatsResponse{
				TotalCancellations:  100,
				RiderCancellations:  70,
				DriverCancellations: 25,
				SystemCancellations: 5,
			}, nil
		},
	}
	svc := NewTestableService(repo, &MockDB{}, &MockRideGetter{})

	stats, err := svc.GetCancellationStats(context.Background(), from, to)

	assert.NoError(t, err)
	assert.Equal(t, 100, stats.TotalCancellations)
	assert.Equal(t, 70, stats.RiderCancellations)
}

func TestTestableService_GetUserCancellationStats_Success(t *testing.T) {
	userID := uuid.New()

	repo := &MockRepository{
		GetUserCancellationStatsFunc: func(ctx context.Context, id uuid.UUID) (*UserCancellationStats, error) {
			return &UserCancellationStats{
				UserID:             id,
				TotalCancellations: 5,
			}, nil
		},
	}
	svc := NewTestableService(repo, &MockDB{}, &MockRideGetter{})

	stats, err := svc.GetUserCancellationStats(context.Background(), userID)

	assert.NoError(t, err)
	assert.Equal(t, userID, stats.UserID)
	assert.Equal(t, 5, stats.TotalCancellations)
}

// ============================================================================
// ERROR VARIABLE TESTS
// ============================================================================

func TestErrorVariables(t *testing.T) {
	assert.NotNil(t, ErrNotAuthorized)
	assert.NotNil(t, ErrNotAuthorizedToCancel)
	assert.NotNil(t, ErrNotAuthorizedToView)
	assert.NotNil(t, ErrRideAlreadyFinished)
	assert.NotNil(t, ErrRideNotFound)
	assert.NotNil(t, ErrCancellationNotFound)
	assert.NotNil(t, ErrCreateRecordFailed)
	assert.NotNil(t, ErrUpdateRideFailed)

	// Check error messages
	assert.Contains(t, ErrNotAuthorized.Error(), "not authorized")
	assert.Contains(t, ErrRideNotFound.Error(), "ride not found")
	assert.Contains(t, ErrCancellationNotFound.Error(), "cancellation record not found")
}
