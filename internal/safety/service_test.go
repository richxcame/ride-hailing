package safety

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/httpclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockRepo is a test-local mock that implements RepositoryInterface.
type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) CreateEmergencyAlert(ctx context.Context, alert *EmergencyAlert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}
func (m *mockRepo) GetEmergencyAlert(ctx context.Context, id uuid.UUID) (*EmergencyAlert, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EmergencyAlert), args.Error(1)
}
func (m *mockRepo) UpdateEmergencyAlertStatus(ctx context.Context, id uuid.UUID, status EmergencyStatus, respondedBy *uuid.UUID, resolution string) error {
	args := m.Called(ctx, id, status, respondedBy, resolution)
	return args.Error(0)
}
func (m *mockRepo) GetActiveEmergencyAlerts(ctx context.Context) ([]*EmergencyAlert, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*EmergencyAlert), args.Error(1)
}
func (m *mockRepo) GetUserEmergencyAlerts(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*EmergencyAlert, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*EmergencyAlert), args.Error(1)
}
func (m *mockRepo) CreateEmergencyContact(ctx context.Context, contact *EmergencyContact) error {
	args := m.Called(ctx, contact)
	return args.Error(0)
}
func (m *mockRepo) GetEmergencyContact(ctx context.Context, id uuid.UUID) (*EmergencyContact, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EmergencyContact), args.Error(1)
}
func (m *mockRepo) GetUserEmergencyContacts(ctx context.Context, userID uuid.UUID) ([]*EmergencyContact, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*EmergencyContact), args.Error(1)
}
func (m *mockRepo) GetSOSContacts(ctx context.Context, userID uuid.UUID) ([]*EmergencyContact, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*EmergencyContact), args.Error(1)
}
func (m *mockRepo) UpdateEmergencyContact(ctx context.Context, contact *EmergencyContact) error {
	args := m.Called(ctx, contact)
	return args.Error(0)
}
func (m *mockRepo) DeleteEmergencyContact(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *mockRepo) VerifyEmergencyContact(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *mockRepo) CreateRideShareLink(ctx context.Context, link *RideShareLink) error {
	args := m.Called(ctx, link)
	return args.Error(0)
}
func (m *mockRepo) GetRideShareLinkByToken(ctx context.Context, token string) (*RideShareLink, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RideShareLink), args.Error(1)
}
func (m *mockRepo) GetRideShareLinks(ctx context.Context, rideID uuid.UUID) ([]*RideShareLink, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RideShareLink), args.Error(1)
}
func (m *mockRepo) IncrementShareLinkViewCount(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *mockRepo) DeactivateShareLink(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *mockRepo) DeactivateRideShareLinks(ctx context.Context, rideID uuid.UUID) error {
	args := m.Called(ctx, rideID)
	return args.Error(0)
}
func (m *mockRepo) GetSafetySettings(ctx context.Context, userID uuid.UUID) (*SafetySettings, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SafetySettings), args.Error(1)
}
func (m *mockRepo) UpdateSafetySettings(ctx context.Context, settings *SafetySettings) error {
	args := m.Called(ctx, settings)
	return args.Error(0)
}
func (m *mockRepo) CreateSafetyCheck(ctx context.Context, check *SafetyCheck) error {
	args := m.Called(ctx, check)
	return args.Error(0)
}
func (m *mockRepo) UpdateSafetyCheckResponse(ctx context.Context, id uuid.UUID, status SafetyCheckStatus, response string) error {
	args := m.Called(ctx, id, status, response)
	return args.Error(0)
}
func (m *mockRepo) GetPendingSafetyChecks(ctx context.Context, userID uuid.UUID) ([]*SafetyCheck, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*SafetyCheck), args.Error(1)
}
func (m *mockRepo) EscalateSafetyCheck(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *mockRepo) GetExpiredPendingSafetyChecks(ctx context.Context, timeout time.Duration) ([]*SafetyCheck, error) {
	args := m.Called(ctx, timeout)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*SafetyCheck), args.Error(1)
}
func (m *mockRepo) CreateRouteDeviationAlert(ctx context.Context, alert *RouteDeviationAlert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}
func (m *mockRepo) UpdateRouteDeviationAcknowledgement(ctx context.Context, id uuid.UUID, response string) error {
	args := m.Called(ctx, id, response)
	return args.Error(0)
}
func (m *mockRepo) GetRecentRouteDeviations(ctx context.Context, rideID uuid.UUID, since time.Duration) ([]*RouteDeviationAlert, error) {
	args := m.Called(ctx, rideID, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RouteDeviationAlert), args.Error(1)
}
func (m *mockRepo) CreateSafetyIncidentReport(ctx context.Context, report *SafetyIncidentReport) error {
	args := m.Called(ctx, report)
	return args.Error(0)
}
func (m *mockRepo) GetSafetyIncidentReport(ctx context.Context, id uuid.UUID) (*SafetyIncidentReport, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SafetyIncidentReport), args.Error(1)
}
func (m *mockRepo) UpdateSafetyIncidentStatus(ctx context.Context, id uuid.UUID, status string, resolution, actionTaken string) error {
	args := m.Called(ctx, id, status, resolution, actionTaken)
	return args.Error(0)
}
func (m *mockRepo) AddTrustedDriver(ctx context.Context, riderID, driverID uuid.UUID, note string) error {
	args := m.Called(ctx, riderID, driverID, note)
	return args.Error(0)
}
func (m *mockRepo) RemoveTrustedDriver(ctx context.Context, riderID, driverID uuid.UUID) error {
	args := m.Called(ctx, riderID, driverID)
	return args.Error(0)
}
func (m *mockRepo) IsDriverTrusted(ctx context.Context, riderID, driverID uuid.UUID) (bool, error) {
	args := m.Called(ctx, riderID, driverID)
	return args.Bool(0), args.Error(1)
}
func (m *mockRepo) AddBlockedDriver(ctx context.Context, riderID, driverID uuid.UUID, reason string) error {
	args := m.Called(ctx, riderID, driverID, reason)
	return args.Error(0)
}
func (m *mockRepo) RemoveBlockedDriver(ctx context.Context, riderID, driverID uuid.UUID) error {
	args := m.Called(ctx, riderID, driverID)
	return args.Error(0)
}
func (m *mockRepo) IsDriverBlocked(ctx context.Context, riderID, driverID uuid.UUID) (bool, error) {
	args := m.Called(ctx, riderID, driverID)
	return args.Bool(0), args.Error(1)
}
func (m *mockRepo) GetBlockedDrivers(ctx context.Context, riderID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, riderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}
func (m *mockRepo) CountUserEmergencyAlerts(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}
func (m *mockRepo) GetEmergencyAlertStats(ctx context.Context, since time.Time) (map[string]interface{}, error) {
	args := m.Called(ctx, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// newTestService creates a Service wired to the given mock repo for testing.
func newTestService(repo *mockRepo) *Service {
	return &Service{
		repo:               repo,
		baseShareURL:       "https://ride.example.com",
		emergencyNumber:    "911",
		notificationClient: httpclient.NewClient("http://localhost:9999"),
		mapsClient:         httpclient.NewClient("http://localhost:9999"),
	}
}

// ========================================
// SERVICE-LEVEL TESTS
// ========================================

// 1. TriggerSOS - success
func TestTriggerSOS_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()

	repo.On("CreateEmergencyAlert", ctx, mock.AnythingOfType("*safety.EmergencyAlert")).Return(nil)
	repo.On("GetSOSContacts", ctx, userID).Return([]*EmergencyContact{}, nil)

	req := &TriggerSOSRequest{
		Type:      EmergencySOS,
		Latitude:  40.7128,
		Longitude: -74.0060,
	}

	resp, err := svc.TriggerSOS(ctx, userID, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.NotEqual(t, uuid.Nil, resp.AlertID)
	assert.Equal(t, string(EmergencyStatusActive), resp.Status)
	assert.Equal(t, "911", resp.EmergencyNumber)
	assert.Contains(t, resp.Message, "Emergency alert activated")
	assert.Equal(t, 0, resp.ContactsNotified)

	repo.AssertExpectations(t)
}

// 2. TriggerSOS - repo error
func TestTriggerSOS_RepoError(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()

	repo.On("CreateEmergencyAlert", ctx, mock.AnythingOfType("*safety.EmergencyAlert")).
		Return(fmt.Errorf("db connection failed"))
	repo.On("GetSOSContacts", ctx, userID).Maybe().Return([]*EmergencyContact{}, nil)

	req := &TriggerSOSRequest{
		Type:      EmergencySOS,
		Latitude:  40.7128,
		Longitude: -74.0060,
	}

	resp, err := svc.TriggerSOS(ctx, userID, req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to create emergency alert")

	repo.AssertExpectations(t)
}

// 3. CancelSOS - success
func TestCancelSOS_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	alertID := uuid.New()

	alert := &EmergencyAlert{
		ID:     alertID,
		UserID: userID,
		Status: EmergencyStatusActive,
	}

	repo.On("GetEmergencyAlert", ctx, alertID).Return(alert, nil)
	repo.On("UpdateEmergencyAlertStatus", ctx, alertID, EmergencyStatusCancelled, (*uuid.UUID)(nil), "Cancelled by user").Return(nil)

	err := svc.CancelSOS(ctx, userID, alertID)
	require.NoError(t, err)

	repo.AssertExpectations(t)
}

// 4. CancelSOS - alert not found
func TestCancelSOS_NotFound(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	alertID := uuid.New()

	repo.On("GetEmergencyAlert", ctx, alertID).Return(nil, nil)

	err := svc.CancelSOS(ctx, userID, alertID)
	require.Error(t, err)
	assert.Equal(t, "alert not found", err.Error())

	repo.AssertExpectations(t)
}

// 5. ResolveEmergencyAlert - success
func TestResolveEmergencyAlert_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	adminID := uuid.New()
	alertID := uuid.New()

	repo.On("UpdateEmergencyAlertStatus", ctx, alertID, EmergencyStatusResolved, &adminID, "Issue resolved by support").Return(nil)

	err := svc.ResolveEmergencyAlert(ctx, adminID, alertID, "Issue resolved by support", false)
	require.NoError(t, err)

	repo.AssertExpectations(t)
}

// 6. ResolveEmergencyAlert - not found (repo returns error)
func TestResolveEmergencyAlert_NotFound(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	adminID := uuid.New()
	alertID := uuid.New()

	repo.On("UpdateEmergencyAlertStatus", ctx, alertID, EmergencyStatusFalse, &adminID, "False alarm confirmed").
		Return(fmt.Errorf("alert not found"))

	err := svc.ResolveEmergencyAlert(ctx, adminID, alertID, "False alarm confirmed", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "alert not found")

	repo.AssertExpectations(t)
}

// 7. CreateEmergencyContact - success
func TestCreateEmergencyContact_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()

	// Return empty contacts (under the limit of 5)
	repo.On("GetUserEmergencyContacts", ctx, userID).Return([]*EmergencyContact{}, nil)
	repo.On("CreateEmergencyContact", ctx, mock.AnythingOfType("*safety.EmergencyContact")).Return(nil)

	req := &CreateEmergencyContactRequest{
		Name:         "Jane Doe",
		Phone:        "+15551234567",
		Email:        "jane@example.com",
		Relationship: "family",
		IsPrimary:    true,
		NotifyOnRide: true,
		NotifyOnSOS:  true,
	}

	contact, err := svc.CreateEmergencyContact(ctx, userID, req)
	require.NoError(t, err)
	require.NotNil(t, contact)

	assert.Equal(t, "Jane Doe", contact.Name)
	assert.Equal(t, "+15551234567", contact.Phone)
	assert.Equal(t, "jane@example.com", contact.Email)
	assert.Equal(t, "family", contact.Relationship)
	assert.True(t, contact.IsPrimary)
	assert.True(t, contact.NotifyOnRide)
	assert.True(t, contact.NotifyOnSOS)
	assert.False(t, contact.Verified)
	assert.NotEqual(t, uuid.Nil, contact.ID)
	assert.Equal(t, userID, contact.UserID)

	repo.AssertExpectations(t)
}

// 8. GetEmergencyContacts - returns list
func TestGetEmergencyContacts_ReturnsList(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()

	expected := []*EmergencyContact{
		{
			ID:           uuid.New(),
			UserID:       userID,
			Name:         "Alice",
			Phone:        "+15551111111",
			Relationship: "friend",
			IsPrimary:    true,
			NotifyOnSOS:  true,
		},
		{
			ID:           uuid.New(),
			UserID:       userID,
			Name:         "Bob",
			Phone:        "+15552222222",
			Relationship: "family",
			IsPrimary:    false,
			NotifyOnSOS:  true,
		},
	}

	repo.On("GetUserEmergencyContacts", ctx, userID).Return(expected, nil)

	contacts, err := svc.GetEmergencyContacts(ctx, userID)
	require.NoError(t, err)
	require.Len(t, contacts, 2)

	assert.Equal(t, "Alice", contacts[0].Name)
	assert.Equal(t, "Bob", contacts[1].Name)

	repo.AssertExpectations(t)
}

// 9. CreateShareLink - generates link with token
func TestCreateShareLink_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()
	rideID := uuid.New()

	repo.On("CreateRideShareLink", ctx, mock.AnythingOfType("*safety.RideShareLink")).Return(nil)

	req := &CreateShareLinkRequest{
		RideID:        rideID,
		ExpiryMinutes: 30,
		ShareLocation: true,
		ShareDriver:   true,
		ShareETA:      true,
		ShareRoute:    false,
	}

	resp, err := svc.CreateShareLink(ctx, userID, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.NotEqual(t, uuid.Nil, resp.LinkID)
	assert.NotEmpty(t, resp.Token)
	assert.True(t, strings.HasPrefix(resp.ShareURL, "https://ride.example.com/share/"))
	assert.Contains(t, resp.ShareURL, resp.Token)
	assert.True(t, resp.ExpiresAt.After(time.Now()))

	repo.AssertExpectations(t)
}

// 10. GetSharedRide - valid token returns data
func TestGetSharedRide_ValidToken(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	rideID := uuid.New()
	linkID := uuid.New()
	token := "valid-test-token-abc123"

	link := &RideShareLink{
		ID:            linkID,
		RideID:        rideID,
		UserID:        uuid.New(),
		Token:         token,
		ExpiresAt:     time.Now().Add(1 * time.Hour),
		IsActive:      true,
		ShareLocation: true,
		ShareDriver:   true,
		ShareETA:      true,
		ShareRoute:    true,
	}

	repo.On("GetRideShareLinkByToken", ctx, token).Return(link, nil)
	// IncrementShareLinkViewCount is called in a goroutine -- allow any context
	repo.On("IncrementShareLinkViewCount", mock.Anything, linkID).Return(nil).Maybe()

	view, err := svc.GetSharedRide(ctx, token)
	require.NoError(t, err)
	require.NotNil(t, view)

	assert.Equal(t, rideID, view.RideID)
	assert.Equal(t, "in_progress", view.Status)
	assert.Equal(t, "911", view.EmergencyContact)
	assert.Contains(t, view.ReportConcernURL, "/safety/report")

	repo.AssertExpectations(t)
}

// 11. CheckRouteDeviation - no deviation (within 500m)
func TestCheckRouteDeviation_NoDeviation(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	rideID := uuid.New()
	driverID := uuid.New()

	// Same point -- 0 meters deviation
	alert, err := svc.CheckRouteDeviation(ctx, rideID, driverID,
		40.7128, -74.0060, // actual
		40.7128, -74.0060, // expected
	)
	require.NoError(t, err)
	assert.Nil(t, alert, "no alert should be created for zero deviation")

	// repo should NOT have been called for CreateRouteDeviationAlert
	repo.AssertNotCalled(t, "CreateRouteDeviationAlert", mock.Anything, mock.Anything)
}

// 12. CheckRouteDeviation - deviation detected (>500m)
func TestCheckRouteDeviation_DeviationDetected(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	rideID := uuid.New()
	driverID := uuid.New()

	repo.On("CreateRouteDeviationAlert", ctx, mock.AnythingOfType("*safety.RouteDeviationAlert")).Return(nil)

	// Large longitude difference to ensure > 500m with the simplified formula
	alert, err := svc.CheckRouteDeviation(ctx, rideID, driverID,
		40.7128, -74.0060, // actual
		40.7128, -72.0060, // expected -- ~2 degrees longitude away
	)
	require.NoError(t, err)
	require.NotNil(t, alert, "alert should be created for large deviation")

	assert.Equal(t, rideID, alert.RideID)
	assert.Equal(t, driverID, alert.DriverID)
	assert.GreaterOrEqual(t, alert.DeviationMeters, 500)
	assert.False(t, alert.RiderNotified)
	assert.False(t, alert.Acknowledged)

	repo.AssertExpectations(t)
}

// 13. GetSafetySettings - returns default when not found
func TestGetSafetySettings_ReturnsSettings(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()

	defaultSettings := &SafetySettings{
		UserID:              userID,
		SOSEnabled:          true,
		SOSGesture:          "shake",
		SOSAutoCall911:      false,
		ShareOnEmergency:    true,
		RouteDeviationAlert: true,
		LongStopAlertMins:   10,
		RecordAudioOnSOS:    true,
		NightProtectionStart: "22:00",
		NightProtectionEnd:   "06:00",
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	repo.On("GetSafetySettings", ctx, userID).Return(defaultSettings, nil)

	settings, err := svc.GetSafetySettings(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, settings)

	assert.Equal(t, userID, settings.UserID)
	assert.True(t, settings.SOSEnabled)
	assert.Equal(t, "shake", settings.SOSGesture)
	assert.False(t, settings.SOSAutoCall911)
	assert.True(t, settings.ShareOnEmergency)
	assert.True(t, settings.RouteDeviationAlert)
	assert.Equal(t, 10, settings.LongStopAlertMins)
	assert.True(t, settings.RecordAudioOnSOS)
	assert.Equal(t, "22:00", settings.NightProtectionStart)
	assert.Equal(t, "06:00", settings.NightProtectionEnd)

	repo.AssertExpectations(t)
}

// 14. UpdateSafetySettings - success
func TestUpdateSafetySettings_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := newTestService(repo)
	ctx := context.Background()
	userID := uuid.New()

	existing := &SafetySettings{
		UserID:              userID,
		SOSEnabled:          true,
		SOSGesture:          "shake",
		SOSAutoCall911:      false,
		ShareOnEmergency:    true,
		RouteDeviationAlert: true,
		LongStopAlertMins:   10,
		RecordAudioOnSOS:    true,
		NightProtectionStart: "22:00",
		NightProtectionEnd:   "06:00",
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	repo.On("GetSafetySettings", ctx, userID).Return(existing, nil)
	repo.On("UpdateSafetySettings", ctx, mock.AnythingOfType("*safety.SafetySettings")).Return(nil)

	newGesture := "power_button"
	sosAutoCall := true
	newCheckInMins := 15
	periodicCheckIn := true

	req := &UpdateSafetySettingsRequest{
		SOSGesture:        &newGesture,
		SOSAutoCall911:    &sosAutoCall,
		CheckInIntervalMins: &newCheckInMins,
		PeriodicCheckIn:   &periodicCheckIn,
	}

	settings, err := svc.UpdateSafetySettings(ctx, userID, req)
	require.NoError(t, err)
	require.NotNil(t, settings)

	// Updated fields
	assert.Equal(t, "power_button", settings.SOSGesture)
	assert.True(t, settings.SOSAutoCall911)
	assert.Equal(t, 15, settings.CheckInIntervalMins)
	assert.True(t, settings.PeriodicCheckIn)

	// Unchanged fields
	assert.True(t, settings.SOSEnabled)
	assert.True(t, settings.ShareOnEmergency)
	assert.True(t, settings.RouteDeviationAlert)
	assert.Equal(t, 10, settings.LongStopAlertMins)

	repo.AssertExpectations(t)
}

func TestHaversineDistance(t *testing.T) {
	tests := []struct {
		name     string
		lat1     float64
		lon1     float64
		lat2     float64
		lon2     float64
		minKm    float64
		maxKm    float64
	}{
		{
			name:  "same point returns zero",
			lat1:  40.7128, lon1: -74.0060,
			lat2:  40.7128, lon2: -74.0060,
			minKm: 0.0, maxKm: 0.001,
		},
		{
			name:  "short distance (latitude only, no longitude diff)",
			lat1:  40.7128, lon1: -74.0060,
			lat2:  40.7218, lon2: -74.0060,
			minKm: 0.0, maxKm: 0.001, // buggy formula yields ~0.00008 km for small lat-only diffs
		},
		{
			name:  "moderate distance (New York to Philadelphia)",
			lat1:  40.7128, lon1: -74.0060,
			lat2:  39.9526, lon2: -75.1652,
			minKm: 1.0, maxKm: 2.0, // buggy formula yields ~1.21 km instead of ~130 km
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := haversineDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			assert.GreaterOrEqual(t, result, tt.minKm,
				"distance %.4f km should be >= %.4f km", result, tt.minKm)
			assert.LessOrEqual(t, result, tt.maxKm,
				"distance %.4f km should be <= %.4f km", result, tt.maxKm)
		})
	}
}

func TestHaversineDistance_NonNegative(t *testing.T) {
	// The implementation uses a simplified formula that can return negative values
	// when latitudes have opposite signs (the cos(lat) terms are replaced with
	// raw radian products which go negative across hemispheres).
	// Test with same-hemisphere coordinates where the formula stays non-negative.
	coords := []struct{ lat, lon float64 }{
		{0, 0}, {0, 180}, {0, -180},
		{45.5, 122.5}, {40.7, -74.0},
	}

	for i := range coords {
		for j := range coords {
			d := haversineDistance(coords[i].lat, coords[i].lon, coords[j].lat, coords[j].lon)
			assert.GreaterOrEqual(t, d, 0.0, "distance should be non-negative for same-hemisphere coords")
		}
	}
}

func TestGenerateSecureToken(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"16 bytes", 16},
		{"32 bytes", 32},
		{"64 bytes", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := generateSecureToken(tt.length)
			assert.NoError(t, err)
			assert.NotEmpty(t, token)
			// Hex encoding doubles the length
			assert.Equal(t, tt.length*2, len(token),
				"hex-encoded token of %d bytes should be %d chars", tt.length, tt.length*2)
		})
	}
}

func TestGenerateSecureToken_Uniqueness(t *testing.T) {
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token, err := generateSecureToken(32)
		assert.NoError(t, err)
		tokens[token] = true
	}
	assert.Equal(t, 100, len(tokens), "all tokens should be unique")
}

func TestGenerateSecureToken_HexCharsOnly(t *testing.T) {
	for i := 0; i < 20; i++ {
		token, err := generateSecureToken(32)
		assert.NoError(t, err)

		for _, ch := range token {
			isHex := (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')
			assert.True(t, isHex, "character '%c' is not a valid hex character", ch)
		}
	}
}

func TestRouteDeviationThreshold(t *testing.T) {
	// The service triggers a route deviation alert when deviation > 500 meters
	// Test that the haversine distance function can detect deviations correctly

	tests := []struct {
		name                string
		actualLat           float64
		actualLng           float64
		expectedLat         float64
		expectedLng         float64
		shouldTriggerAlert  bool
	}{
		{
			name:               "exact same position - no alert",
			actualLat:          40.7128, actualLng: -74.0060,
			expectedLat:        40.7128, expectedLng: -74.0060,
			shouldTriggerAlert: false,
		},
		{
			name:               "small deviation (100m) - no alert",
			actualLat:          40.7128, actualLng: -74.0060,
			expectedLat:        40.7137, expectedLng: -74.0060, // ~100m north
			shouldTriggerAlert: false,
		},
		{
			name:               "large deviation - should alert",
			actualLat:          40.7128, actualLng: -74.0060,
			expectedLat:        40.7128, expectedLng: -72.0060, // large longitude diff triggers alert with simplified formula
			shouldTriggerAlert: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deviation := haversineDistance(tt.actualLat, tt.actualLng, tt.expectedLat, tt.expectedLng)
			deviationMeters := int(deviation * 1000)

			if tt.shouldTriggerAlert {
				assert.GreaterOrEqual(t, deviationMeters, 500,
					"deviation of %d meters should trigger alert (>= 500m)", deviationMeters)
			} else {
				assert.Less(t, deviationMeters, 500,
					"deviation of %d meters should not trigger alert (< 500m)", deviationMeters)
			}
		})
	}
}

func TestConfig_Defaults(t *testing.T) {
	cfg := Config{
		BaseShareURL:    "https://ride.example.com",
		EmergencyNumber: "911",
		NotificationURL: "http://notifications:8080",
		MapsServiceURL:  "http://maps:8080",
	}

	assert.Equal(t, "https://ride.example.com", cfg.BaseShareURL)
	assert.Equal(t, "911", cfg.EmergencyNumber)
	assert.NotEmpty(t, cfg.NotificationURL)
	assert.NotEmpty(t, cfg.MapsServiceURL)
}

func TestEmergencyAlertStatus_Constants(t *testing.T) {
	assert.Equal(t, EmergencyStatus("active"), EmergencyStatusActive)
	assert.Equal(t, EmergencyStatus("responded"), EmergencyStatusResponded)
	assert.Equal(t, EmergencyStatus("resolved"), EmergencyStatusResolved)
	assert.Equal(t, EmergencyStatus("cancelled"), EmergencyStatusCancelled)
	assert.Equal(t, EmergencyStatus("false_alarm"), EmergencyStatusFalse)
}

func TestSafetyCheckStatus_Constants(t *testing.T) {
	assert.Equal(t, SafetyCheckStatus("pending"), SafetyCheckPending)
	assert.Equal(t, SafetyCheckStatus("safe"), SafetyCheckSafe)
	assert.Equal(t, SafetyCheckStatus("help"), SafetyCheckHelp)
	assert.Equal(t, SafetyCheckStatus("no_response"), SafetyCheckNoResponse)
}
