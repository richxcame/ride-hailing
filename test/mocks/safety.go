package mocks

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/internal/safety"
	"github.com/stretchr/testify/mock"
)

// MockSafetyRepository is a mock implementation of safety.RepositoryInterface
type MockSafetyRepository struct {
	mock.Mock
}

// Emergency Alerts

func (m *MockSafetyRepository) CreateEmergencyAlert(ctx context.Context, alert *safety.EmergencyAlert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockSafetyRepository) GetEmergencyAlert(ctx context.Context, id uuid.UUID) (*safety.EmergencyAlert, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*safety.EmergencyAlert), args.Error(1)
}

func (m *MockSafetyRepository) UpdateEmergencyAlertStatus(ctx context.Context, id uuid.UUID, status safety.EmergencyStatus, respondedBy *uuid.UUID, resolution string) error {
	args := m.Called(ctx, id, status, respondedBy, resolution)
	return args.Error(0)
}

func (m *MockSafetyRepository) GetActiveEmergencyAlerts(ctx context.Context) ([]*safety.EmergencyAlert, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*safety.EmergencyAlert), args.Error(1)
}

func (m *MockSafetyRepository) GetUserEmergencyAlerts(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*safety.EmergencyAlert, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*safety.EmergencyAlert), args.Error(1)
}

// Emergency Contacts

func (m *MockSafetyRepository) CreateEmergencyContact(ctx context.Context, contact *safety.EmergencyContact) error {
	args := m.Called(ctx, contact)
	return args.Error(0)
}

func (m *MockSafetyRepository) GetEmergencyContact(ctx context.Context, id uuid.UUID) (*safety.EmergencyContact, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*safety.EmergencyContact), args.Error(1)
}

func (m *MockSafetyRepository) GetUserEmergencyContacts(ctx context.Context, userID uuid.UUID) ([]*safety.EmergencyContact, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*safety.EmergencyContact), args.Error(1)
}

func (m *MockSafetyRepository) GetSOSContacts(ctx context.Context, userID uuid.UUID) ([]*safety.EmergencyContact, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*safety.EmergencyContact), args.Error(1)
}

func (m *MockSafetyRepository) UpdateEmergencyContact(ctx context.Context, contact *safety.EmergencyContact) error {
	args := m.Called(ctx, contact)
	return args.Error(0)
}

func (m *MockSafetyRepository) DeleteEmergencyContact(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSafetyRepository) VerifyEmergencyContact(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Ride Share Links

func (m *MockSafetyRepository) CreateRideShareLink(ctx context.Context, link *safety.RideShareLink) error {
	args := m.Called(ctx, link)
	return args.Error(0)
}

func (m *MockSafetyRepository) GetRideShareLinkByToken(ctx context.Context, token string) (*safety.RideShareLink, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*safety.RideShareLink), args.Error(1)
}

func (m *MockSafetyRepository) GetRideShareLinks(ctx context.Context, rideID uuid.UUID) ([]*safety.RideShareLink, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*safety.RideShareLink), args.Error(1)
}

func (m *MockSafetyRepository) IncrementShareLinkViewCount(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSafetyRepository) DeactivateShareLink(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSafetyRepository) DeactivateRideShareLinks(ctx context.Context, rideID uuid.UUID) error {
	args := m.Called(ctx, rideID)
	return args.Error(0)
}

// Safety Settings

func (m *MockSafetyRepository) GetSafetySettings(ctx context.Context, userID uuid.UUID) (*safety.SafetySettings, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*safety.SafetySettings), args.Error(1)
}

func (m *MockSafetyRepository) UpdateSafetySettings(ctx context.Context, settings *safety.SafetySettings) error {
	args := m.Called(ctx, settings)
	return args.Error(0)
}

// Safety Checks

func (m *MockSafetyRepository) CreateSafetyCheck(ctx context.Context, check *safety.SafetyCheck) error {
	args := m.Called(ctx, check)
	return args.Error(0)
}

func (m *MockSafetyRepository) UpdateSafetyCheckResponse(ctx context.Context, id uuid.UUID, status safety.SafetyCheckStatus, response string) error {
	args := m.Called(ctx, id, status, response)
	return args.Error(0)
}

func (m *MockSafetyRepository) GetPendingSafetyChecks(ctx context.Context, userID uuid.UUID) ([]*safety.SafetyCheck, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*safety.SafetyCheck), args.Error(1)
}

func (m *MockSafetyRepository) EscalateSafetyCheck(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSafetyRepository) GetExpiredPendingSafetyChecks(ctx context.Context, timeout time.Duration) ([]*safety.SafetyCheck, error) {
	args := m.Called(ctx, timeout)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*safety.SafetyCheck), args.Error(1)
}

// Route Deviation Alerts

func (m *MockSafetyRepository) CreateRouteDeviationAlert(ctx context.Context, alert *safety.RouteDeviationAlert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockSafetyRepository) UpdateRouteDeviationAcknowledgement(ctx context.Context, id uuid.UUID, response string) error {
	args := m.Called(ctx, id, response)
	return args.Error(0)
}

func (m *MockSafetyRepository) GetRecentRouteDeviations(ctx context.Context, rideID uuid.UUID, since time.Duration) ([]*safety.RouteDeviationAlert, error) {
	args := m.Called(ctx, rideID, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*safety.RouteDeviationAlert), args.Error(1)
}

// Safety Incident Reports

func (m *MockSafetyRepository) CreateSafetyIncidentReport(ctx context.Context, report *safety.SafetyIncidentReport) error {
	args := m.Called(ctx, report)
	return args.Error(0)
}

func (m *MockSafetyRepository) GetSafetyIncidentReport(ctx context.Context, id uuid.UUID) (*safety.SafetyIncidentReport, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*safety.SafetyIncidentReport), args.Error(1)
}

func (m *MockSafetyRepository) UpdateSafetyIncidentStatus(ctx context.Context, id uuid.UUID, status string, resolution, actionTaken string) error {
	args := m.Called(ctx, id, status, resolution, actionTaken)
	return args.Error(0)
}

// Trusted/Blocked Drivers

func (m *MockSafetyRepository) AddTrustedDriver(ctx context.Context, riderID, driverID uuid.UUID, note string) error {
	args := m.Called(ctx, riderID, driverID, note)
	return args.Error(0)
}

func (m *MockSafetyRepository) RemoveTrustedDriver(ctx context.Context, riderID, driverID uuid.UUID) error {
	args := m.Called(ctx, riderID, driverID)
	return args.Error(0)
}

func (m *MockSafetyRepository) IsDriverTrusted(ctx context.Context, riderID, driverID uuid.UUID) (bool, error) {
	args := m.Called(ctx, riderID, driverID)
	return args.Bool(0), args.Error(1)
}

func (m *MockSafetyRepository) AddBlockedDriver(ctx context.Context, riderID, driverID uuid.UUID, reason string) error {
	args := m.Called(ctx, riderID, driverID, reason)
	return args.Error(0)
}

func (m *MockSafetyRepository) RemoveBlockedDriver(ctx context.Context, riderID, driverID uuid.UUID) error {
	args := m.Called(ctx, riderID, driverID)
	return args.Error(0)
}

func (m *MockSafetyRepository) IsDriverBlocked(ctx context.Context, riderID, driverID uuid.UUID) (bool, error) {
	args := m.Called(ctx, riderID, driverID)
	return args.Bool(0), args.Error(1)
}

func (m *MockSafetyRepository) GetBlockedDrivers(ctx context.Context, riderID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, riderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

// Statistics

func (m *MockSafetyRepository) CountUserEmergencyAlerts(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockSafetyRepository) GetEmergencyAlertStats(ctx context.Context, since time.Time) (map[string]interface{}, error) {
	args := m.Called(ctx, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}
