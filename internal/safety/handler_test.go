package safety

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// Mock Service
// ============================================================================

// MockSafetyService is a mock implementation of the safety service
type MockSafetyService struct {
	mock.Mock
	repo *MockRepository
}

// MockRepository is a mock implementation of RepositoryInterface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateEmergencyAlert(ctx context.Context, alert *EmergencyAlert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockRepository) GetEmergencyAlert(ctx context.Context, id uuid.UUID) (*EmergencyAlert, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EmergencyAlert), args.Error(1)
}

func (m *MockRepository) UpdateEmergencyAlertStatus(ctx context.Context, id uuid.UUID, status EmergencyStatus, respondedBy *uuid.UUID, resolution string) error {
	args := m.Called(ctx, id, status, respondedBy, resolution)
	return args.Error(0)
}

func (m *MockRepository) GetActiveEmergencyAlerts(ctx context.Context) ([]*EmergencyAlert, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*EmergencyAlert), args.Error(1)
}

func (m *MockRepository) GetUserEmergencyAlerts(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*EmergencyAlert, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*EmergencyAlert), args.Error(1)
}

func (m *MockRepository) CreateEmergencyContact(ctx context.Context, contact *EmergencyContact) error {
	args := m.Called(ctx, contact)
	return args.Error(0)
}

func (m *MockRepository) GetEmergencyContact(ctx context.Context, id uuid.UUID) (*EmergencyContact, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EmergencyContact), args.Error(1)
}

func (m *MockRepository) GetUserEmergencyContacts(ctx context.Context, userID uuid.UUID) ([]*EmergencyContact, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*EmergencyContact), args.Error(1)
}

func (m *MockRepository) GetSOSContacts(ctx context.Context, userID uuid.UUID) ([]*EmergencyContact, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*EmergencyContact), args.Error(1)
}

func (m *MockRepository) UpdateEmergencyContact(ctx context.Context, contact *EmergencyContact) error {
	args := m.Called(ctx, contact)
	return args.Error(0)
}

func (m *MockRepository) DeleteEmergencyContact(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) VerifyEmergencyContact(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) CreateRideShareLink(ctx context.Context, link *RideShareLink) error {
	args := m.Called(ctx, link)
	return args.Error(0)
}

func (m *MockRepository) GetRideShareLinkByToken(ctx context.Context, token string) (*RideShareLink, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RideShareLink), args.Error(1)
}

func (m *MockRepository) GetRideShareLinks(ctx context.Context, rideID uuid.UUID) ([]*RideShareLink, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RideShareLink), args.Error(1)
}

func (m *MockRepository) IncrementShareLinkViewCount(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) DeactivateShareLink(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) DeactivateRideShareLinks(ctx context.Context, rideID uuid.UUID) error {
	args := m.Called(ctx, rideID)
	return args.Error(0)
}

func (m *MockRepository) GetSafetySettings(ctx context.Context, userID uuid.UUID) (*SafetySettings, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SafetySettings), args.Error(1)
}

func (m *MockRepository) UpdateSafetySettings(ctx context.Context, settings *SafetySettings) error {
	args := m.Called(ctx, settings)
	return args.Error(0)
}

func (m *MockRepository) CreateSafetyCheck(ctx context.Context, check *SafetyCheck) error {
	args := m.Called(ctx, check)
	return args.Error(0)
}

func (m *MockRepository) UpdateSafetyCheckResponse(ctx context.Context, id uuid.UUID, status SafetyCheckStatus, response string) error {
	args := m.Called(ctx, id, status, response)
	return args.Error(0)
}

func (m *MockRepository) GetPendingSafetyChecks(ctx context.Context, userID uuid.UUID) ([]*SafetyCheck, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*SafetyCheck), args.Error(1)
}

func (m *MockRepository) EscalateSafetyCheck(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) GetExpiredPendingSafetyChecks(ctx context.Context, timeout time.Duration) ([]*SafetyCheck, error) {
	args := m.Called(ctx, timeout)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*SafetyCheck), args.Error(1)
}

func (m *MockRepository) CreateRouteDeviationAlert(ctx context.Context, alert *RouteDeviationAlert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *MockRepository) UpdateRouteDeviationAcknowledgement(ctx context.Context, id uuid.UUID, response string) error {
	args := m.Called(ctx, id, response)
	return args.Error(0)
}

func (m *MockRepository) GetRecentRouteDeviations(ctx context.Context, rideID uuid.UUID, since time.Duration) ([]*RouteDeviationAlert, error) {
	args := m.Called(ctx, rideID, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RouteDeviationAlert), args.Error(1)
}

func (m *MockRepository) CreateSafetyIncidentReport(ctx context.Context, report *SafetyIncidentReport) error {
	args := m.Called(ctx, report)
	return args.Error(0)
}

func (m *MockRepository) GetSafetyIncidentReport(ctx context.Context, id uuid.UUID) (*SafetyIncidentReport, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SafetyIncidentReport), args.Error(1)
}

func (m *MockRepository) UpdateSafetyIncidentStatus(ctx context.Context, id uuid.UUID, status string, resolution, actionTaken string) error {
	args := m.Called(ctx, id, status, resolution, actionTaken)
	return args.Error(0)
}

func (m *MockRepository) AddTrustedDriver(ctx context.Context, riderID, driverID uuid.UUID, note string) error {
	args := m.Called(ctx, riderID, driverID, note)
	return args.Error(0)
}

func (m *MockRepository) RemoveTrustedDriver(ctx context.Context, riderID, driverID uuid.UUID) error {
	args := m.Called(ctx, riderID, driverID)
	return args.Error(0)
}

func (m *MockRepository) IsDriverTrusted(ctx context.Context, riderID, driverID uuid.UUID) (bool, error) {
	args := m.Called(ctx, riderID, driverID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) AddBlockedDriver(ctx context.Context, riderID, driverID uuid.UUID, reason string) error {
	args := m.Called(ctx, riderID, driverID, reason)
	return args.Error(0)
}

func (m *MockRepository) RemoveBlockedDriver(ctx context.Context, riderID, driverID uuid.UUID) error {
	args := m.Called(ctx, riderID, driverID)
	return args.Error(0)
}

func (m *MockRepository) IsDriverBlocked(ctx context.Context, riderID, driverID uuid.UUID) (bool, error) {
	args := m.Called(ctx, riderID, driverID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) GetBlockedDrivers(ctx context.Context, riderID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, riderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

func (m *MockRepository) CountUserEmergencyAlerts(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) GetEmergencyAlertStats(ctx context.Context, since time.Time) (map[string]interface{}, error) {
	args := m.Called(ctx, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// ============================================================================
// Test Helpers
// ============================================================================

func setupTestContext(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	c.Request = req

	return c, w
}

func setUserContext(c *gin.Context, userID uuid.UUID) {
	c.Set("user_id", userID)
}

func setAdminContext(c *gin.Context, adminID uuid.UUID) {
	c.Set("user_id", adminID)
	c.Set("user_role", "admin")
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

func createTestService(mockRepo *MockRepository) *Service {
	cfg := Config{
		BaseShareURL:    "https://example.com",
		EmergencyNumber: "911",
	}
	return NewService(mockRepo, cfg)
}

func createTestHandler(mockRepo *MockRepository) *Handler {
	service := createTestService(mockRepo)
	return NewHandler(service)
}

func createTestEmergencyAlert(userID uuid.UUID, status EmergencyStatus) *EmergencyAlert {
	return &EmergencyAlert{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      EmergencySOS,
		Status:    status,
		Latitude:  40.7128,
		Longitude: -74.0060,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func createTestEmergencyContact(userID uuid.UUID) *EmergencyContact {
	return &EmergencyContact{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         "John Doe",
		Phone:        "+1234567890",
		Email:        "john@example.com",
		Relationship: "family",
		IsPrimary:    true,
		NotifyOnSOS:  true,
		Verified:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func createTestRideShareLink(userID, rideID uuid.UUID) *RideShareLink {
	return &RideShareLink{
		ID:            uuid.New(),
		RideID:        rideID,
		UserID:        userID,
		Token:         "test-token-12345",
		ExpiresAt:     time.Now().Add(24 * time.Hour),
		IsActive:      true,
		ShareLocation: true,
		ShareDriver:   true,
		ShareETA:      true,
		ShareRoute:    true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func createTestSafetySettings(userID uuid.UUID) *SafetySettings {
	return &SafetySettings{
		UserID:              userID,
		SOSEnabled:          true,
		SOSGesture:          "shake",
		SOSAutoCall911:      false,
		AutoShareWithContacts: true,
		PeriodicCheckIn:     true,
		CheckInIntervalMins: 15,
		RouteDeviationAlert: true,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
}

func createTestSafetyCheck(userID, rideID uuid.UUID) *SafetyCheck {
	return &SafetyCheck{
		ID:        uuid.New(),
		RideID:    rideID,
		UserID:    userID,
		Type:      SafetyCheckPeriodic,
		Status:    SafetyCheckPending,
		SentAt:    time.Now(),
		CreatedAt: time.Now(),
	}
}

// ============================================================================
// TriggerSOS Handler Tests
// ============================================================================

func TestHandler_TriggerSOS_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := TriggerSOSRequest{
		Type:      EmergencySOS,
		Latitude:  40.7128,
		Longitude: -74.0060,
	}

	mockRepo.On("CreateEmergencyAlert", mock.Anything, mock.AnythingOfType("*safety.EmergencyAlert")).Return(nil)
	mockRepo.On("GetSOSContacts", mock.Anything, userID).Return([]*EmergencyContact{}, nil)

	c, w := setupTestContext("POST", "/api/v1/safety/sos", reqBody)
	setUserContext(c, userID)

	handler.TriggerSOS(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_TriggerSOS_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := TriggerSOSRequest{
		Type:      EmergencySOS,
		Latitude:  40.7128,
		Longitude: -74.0060,
	}

	c, w := setupTestContext("POST", "/api/v1/safety/sos", reqBody)
	// Don't set user context

	handler.TriggerSOS(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_TriggerSOS_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/safety/sos", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/safety/sos", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID)

	handler.TriggerSOS(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TriggerSOS_MissingType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"latitude":  40.7128,
		"longitude": -74.0060,
	}

	c, w := setupTestContext("POST", "/api/v1/safety/sos", reqBody)
	setUserContext(c, userID)

	handler.TriggerSOS(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TriggerSOS_MissingLocation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"type": "sos",
	}

	c, w := setupTestContext("POST", "/api/v1/safety/sos", reqBody)
	setUserContext(c, userID)

	handler.TriggerSOS(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TriggerSOS_WithRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	rideID := uuid.New()
	reqBody := TriggerSOSRequest{
		Type:      EmergencySOS,
		RideID:    &rideID,
		Latitude:  40.7128,
		Longitude: -74.0060,
	}

	mockRepo.On("CreateEmergencyAlert", mock.Anything, mock.AnythingOfType("*safety.EmergencyAlert")).Return(nil)
	mockRepo.On("GetSOSContacts", mock.Anything, userID).Return([]*EmergencyContact{}, nil)
	mockRepo.On("CreateRideShareLink", mock.Anything, mock.AnythingOfType("*safety.RideShareLink")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/safety/sos", reqBody)
	setUserContext(c, userID)

	handler.TriggerSOS(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_TriggerSOS_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := TriggerSOSRequest{
		Type:      EmergencySOS,
		Latitude:  40.7128,
		Longitude: -74.0060,
	}

	mockRepo.On("CreateEmergencyAlert", mock.Anything, mock.AnythingOfType("*safety.EmergencyAlert")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/safety/sos", reqBody)
	setUserContext(c, userID)

	handler.TriggerSOS(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_TriggerSOS_AllEmergencyTypes(t *testing.T) {
	emergencyTypes := []EmergencyType{
		EmergencySOS,
		EmergencyAccident,
		EmergencyMedical,
		EmergencyThreat,
		EmergencyOther,
	}

	for _, emergencyType := range emergencyTypes {
		t.Run(string(emergencyType), func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			userID := uuid.New()
			reqBody := TriggerSOSRequest{
				Type:      emergencyType,
				Latitude:  40.7128,
				Longitude: -74.0060,
			}

			mockRepo.On("CreateEmergencyAlert", mock.Anything, mock.AnythingOfType("*safety.EmergencyAlert")).Return(nil)
			mockRepo.On("GetSOSContacts", mock.Anything, userID).Return([]*EmergencyContact{}, nil)

			c, w := setupTestContext("POST", "/api/v1/safety/sos", reqBody)
			setUserContext(c, userID)

			handler.TriggerSOS(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

// ============================================================================
// CancelSOS Handler Tests
// ============================================================================

func TestHandler_CancelSOS_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	alertID := uuid.New()
	alert := createTestEmergencyAlert(userID, EmergencyStatusActive)
	alert.ID = alertID

	mockRepo.On("GetEmergencyAlert", mock.Anything, alertID).Return(alert, nil)
	mockRepo.On("UpdateEmergencyAlertStatus", mock.Anything, alertID, EmergencyStatusCancelled, (*uuid.UUID)(nil), "Cancelled by user").Return(nil)

	c, w := setupTestContext("DELETE", "/api/v1/safety/sos/"+alertID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}
	setUserContext(c, userID)

	handler.CancelSOS(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_CancelSOS_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	alertID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/safety/sos/"+alertID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}
	// Don't set user context

	handler.CancelSOS(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CancelSOS_InvalidAlertID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/safety/sos/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID)

	handler.CancelSOS(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid alert ID")
}

func TestHandler_CancelSOS_AlertNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	alertID := uuid.New()

	mockRepo.On("GetEmergencyAlert", mock.Anything, alertID).Return(nil, nil)

	c, w := setupTestContext("DELETE", "/api/v1/safety/sos/"+alertID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}
	setUserContext(c, userID)

	handler.CancelSOS(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_CancelSOS_NotOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	otherUserID := uuid.New()
	alertID := uuid.New()
	alert := createTestEmergencyAlert(otherUserID, EmergencyStatusActive)
	alert.ID = alertID

	mockRepo.On("GetEmergencyAlert", mock.Anything, alertID).Return(alert, nil)

	c, w := setupTestContext("DELETE", "/api/v1/safety/sos/"+alertID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}
	setUserContext(c, userID)

	handler.CancelSOS(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_CancelSOS_AlreadyResolved(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	alertID := uuid.New()
	alert := createTestEmergencyAlert(userID, EmergencyStatusResolved)
	alert.ID = alertID

	mockRepo.On("GetEmergencyAlert", mock.Anything, alertID).Return(alert, nil)

	c, w := setupTestContext("DELETE", "/api/v1/safety/sos/"+alertID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}
	setUserContext(c, userID)

	handler.CancelSOS(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetEmergency Handler Tests
// ============================================================================

func TestHandler_GetEmergency_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	alertID := uuid.New()
	alert := createTestEmergencyAlert(userID, EmergencyStatusActive)
	alert.ID = alertID

	mockRepo.On("GetEmergencyAlert", mock.Anything, alertID).Return(alert, nil)

	c, w := setupTestContext("GET", "/api/v1/safety/sos/"+alertID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}

	handler.GetEmergency(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetEmergency_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/safety/sos/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.GetEmergency(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetEmergency_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	alertID := uuid.New()

	mockRepo.On("GetEmergencyAlert", mock.Anything, alertID).Return(nil, nil)

	c, w := setupTestContext("GET", "/api/v1/safety/sos/"+alertID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}

	handler.GetEmergency(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetEmergency_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	alertID := uuid.New()

	mockRepo.On("GetEmergencyAlert", mock.Anything, alertID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/safety/sos/"+alertID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}

	handler.GetEmergency(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetUserEmergencies Handler Tests
// ============================================================================

func TestHandler_GetUserEmergencies_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	alerts := []*EmergencyAlert{
		createTestEmergencyAlert(userID, EmergencyStatusActive),
		createTestEmergencyAlert(userID, EmergencyStatusResolved),
	}

	mockRepo.On("GetUserEmergencyAlerts", mock.Anything, userID, 20, 0).Return(alerts, nil)

	c, w := setupTestContext("GET", "/api/v1/safety/sos", nil)
	setUserContext(c, userID)

	handler.GetUserEmergencies(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetUserEmergencies_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/safety/sos", nil)
	// Don't set user context

	handler.GetUserEmergencies(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetUserEmergencies_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetUserEmergencyAlerts", mock.Anything, userID, 20, 0).Return([]*EmergencyAlert{}, nil)

	c, w := setupTestContext("GET", "/api/v1/safety/sos", nil)
	setUserContext(c, userID)

	handler.GetUserEmergencies(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetUserEmergencies_WithPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	alerts := []*EmergencyAlert{
		createTestEmergencyAlert(userID, EmergencyStatusActive),
	}

	mockRepo.On("GetUserEmergencyAlerts", mock.Anything, userID, 10, 5).Return(alerts, nil)

	c, w := setupTestContext("GET", "/api/v1/safety/sos?limit=10&offset=5", nil)
	c.Request.URL.RawQuery = "limit=10&offset=5"
	setUserContext(c, userID)

	handler.GetUserEmergencies(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetUserEmergencies_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetUserEmergencyAlerts", mock.Anything, userID, 20, 0).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/safety/sos", nil)
	setUserContext(c, userID)

	handler.GetUserEmergencies(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// CreateEmergencyContact Handler Tests
// ============================================================================

func TestHandler_CreateEmergencyContact_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := CreateEmergencyContactRequest{
		Name:         "Jane Doe",
		Phone:        "+1234567890",
		Email:        "jane@example.com",
		Relationship: "family",
		IsPrimary:    true,
		NotifyOnSOS:  true,
	}

	mockRepo.On("GetUserEmergencyContacts", mock.Anything, userID).Return([]*EmergencyContact{}, nil)
	mockRepo.On("CreateEmergencyContact", mock.Anything, mock.AnythingOfType("*safety.EmergencyContact")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/safety/contacts", reqBody)
	setUserContext(c, userID)

	handler.CreateEmergencyContact(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_CreateEmergencyContact_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := CreateEmergencyContactRequest{
		Name:         "Jane Doe",
		Phone:        "+1234567890",
		Relationship: "family",
	}

	c, w := setupTestContext("POST", "/api/v1/safety/contacts", reqBody)
	// Don't set user context

	handler.CreateEmergencyContact(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CreateEmergencyContact_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/safety/contacts", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/safety/contacts", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID)

	handler.CreateEmergencyContact(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateEmergencyContact_MissingName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"phone":        "+1234567890",
		"relationship": "family",
	}

	c, w := setupTestContext("POST", "/api/v1/safety/contacts", reqBody)
	setUserContext(c, userID)

	handler.CreateEmergencyContact(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateEmergencyContact_MissingPhone(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"name":         "Jane Doe",
		"relationship": "family",
	}

	c, w := setupTestContext("POST", "/api/v1/safety/contacts", reqBody)
	setUserContext(c, userID)

	handler.CreateEmergencyContact(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateEmergencyContact_MissingRelationship(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"name":  "Jane Doe",
		"phone": "+1234567890",
	}

	c, w := setupTestContext("POST", "/api/v1/safety/contacts", reqBody)
	setUserContext(c, userID)

	handler.CreateEmergencyContact(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateEmergencyContact_MaxContactsReached(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := CreateEmergencyContactRequest{
		Name:         "Jane Doe",
		Phone:        "+1234567890",
		Relationship: "family",
	}

	existingContacts := make([]*EmergencyContact, 5)
	for i := 0; i < 5; i++ {
		existingContacts[i] = createTestEmergencyContact(userID)
	}

	mockRepo.On("GetUserEmergencyContacts", mock.Anything, userID).Return(existingContacts, nil)

	c, w := setupTestContext("POST", "/api/v1/safety/contacts", reqBody)
	setUserContext(c, userID)

	handler.CreateEmergencyContact(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_CreateEmergencyContact_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := CreateEmergencyContactRequest{
		Name:         "Jane Doe",
		Phone:        "+1234567890",
		Relationship: "family",
	}

	mockRepo.On("GetUserEmergencyContacts", mock.Anything, userID).Return([]*EmergencyContact{}, nil)
	mockRepo.On("CreateEmergencyContact", mock.Anything, mock.AnythingOfType("*safety.EmergencyContact")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/safety/contacts", reqBody)
	setUserContext(c, userID)

	handler.CreateEmergencyContact(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetEmergencyContacts Handler Tests
// ============================================================================

func TestHandler_GetEmergencyContacts_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	contacts := []*EmergencyContact{
		createTestEmergencyContact(userID),
		createTestEmergencyContact(userID),
	}

	mockRepo.On("GetUserEmergencyContacts", mock.Anything, userID).Return(contacts, nil)

	c, w := setupTestContext("GET", "/api/v1/safety/contacts", nil)
	setUserContext(c, userID)

	handler.GetEmergencyContacts(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetEmergencyContacts_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/safety/contacts", nil)
	// Don't set user context

	handler.GetEmergencyContacts(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetEmergencyContacts_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetUserEmergencyContacts", mock.Anything, userID).Return([]*EmergencyContact{}, nil)

	c, w := setupTestContext("GET", "/api/v1/safety/contacts", nil)
	setUserContext(c, userID)

	handler.GetEmergencyContacts(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetEmergencyContacts_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetUserEmergencyContacts", mock.Anything, userID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/safety/contacts", nil)
	setUserContext(c, userID)

	handler.GetEmergencyContacts(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// UpdateEmergencyContact Handler Tests
// ============================================================================

func TestHandler_UpdateEmergencyContact_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	contactID := uuid.New()
	contact := createTestEmergencyContact(userID)
	contact.ID = contactID

	reqBody := CreateEmergencyContactRequest{
		Name:         "Updated Name",
		Phone:        "+1234567890",
		Relationship: "friend",
	}

	mockRepo.On("GetEmergencyContact", mock.Anything, contactID).Return(contact, nil)
	mockRepo.On("UpdateEmergencyContact", mock.Anything, mock.AnythingOfType("*safety.EmergencyContact")).Return(nil)

	c, w := setupTestContext("PUT", "/api/v1/safety/contacts/"+contactID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: contactID.String()}}
	setUserContext(c, userID)

	handler.UpdateEmergencyContact(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_UpdateEmergencyContact_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	contactID := uuid.New()
	reqBody := CreateEmergencyContactRequest{
		Name:         "Updated Name",
		Phone:        "+1234567890",
		Relationship: "friend",
	}

	c, w := setupTestContext("PUT", "/api/v1/safety/contacts/"+contactID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: contactID.String()}}
	// Don't set user context

	handler.UpdateEmergencyContact(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_UpdateEmergencyContact_InvalidContactID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := CreateEmergencyContactRequest{
		Name:         "Updated Name",
		Phone:        "+1234567890",
		Relationship: "friend",
	}

	c, w := setupTestContext("PUT", "/api/v1/safety/contacts/invalid-uuid", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID)

	handler.UpdateEmergencyContact(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateEmergencyContact_ContactNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	contactID := uuid.New()
	reqBody := CreateEmergencyContactRequest{
		Name:         "Updated Name",
		Phone:        "+1234567890",
		Relationship: "friend",
	}

	mockRepo.On("GetEmergencyContact", mock.Anything, contactID).Return(nil, nil)

	c, w := setupTestContext("PUT", "/api/v1/safety/contacts/"+contactID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: contactID.String()}}
	setUserContext(c, userID)

	handler.UpdateEmergencyContact(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_UpdateEmergencyContact_NotOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	otherUserID := uuid.New()
	contactID := uuid.New()
	contact := createTestEmergencyContact(otherUserID)
	contact.ID = contactID

	reqBody := CreateEmergencyContactRequest{
		Name:         "Updated Name",
		Phone:        "+1234567890",
		Relationship: "friend",
	}

	mockRepo.On("GetEmergencyContact", mock.Anything, contactID).Return(contact, nil)

	c, w := setupTestContext("PUT", "/api/v1/safety/contacts/"+contactID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: contactID.String()}}
	setUserContext(c, userID)

	handler.UpdateEmergencyContact(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// DeleteEmergencyContact Handler Tests
// ============================================================================

func TestHandler_DeleteEmergencyContact_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	contactID := uuid.New()
	contact := createTestEmergencyContact(userID)
	contact.ID = contactID

	mockRepo.On("GetEmergencyContact", mock.Anything, contactID).Return(contact, nil)
	mockRepo.On("DeleteEmergencyContact", mock.Anything, contactID).Return(nil)

	c, w := setupTestContext("DELETE", "/api/v1/safety/contacts/"+contactID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: contactID.String()}}
	setUserContext(c, userID)

	handler.DeleteEmergencyContact(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_DeleteEmergencyContact_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	contactID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/safety/contacts/"+contactID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: contactID.String()}}
	// Don't set user context

	handler.DeleteEmergencyContact(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_DeleteEmergencyContact_InvalidContactID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/safety/contacts/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID)

	handler.DeleteEmergencyContact(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DeleteEmergencyContact_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	contactID := uuid.New()

	mockRepo.On("GetEmergencyContact", mock.Anything, contactID).Return(nil, nil)

	c, w := setupTestContext("DELETE", "/api/v1/safety/contacts/"+contactID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: contactID.String()}}
	setUserContext(c, userID)

	handler.DeleteEmergencyContact(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_DeleteEmergencyContact_NotOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	otherUserID := uuid.New()
	contactID := uuid.New()
	contact := createTestEmergencyContact(otherUserID)
	contact.ID = contactID

	mockRepo.On("GetEmergencyContact", mock.Anything, contactID).Return(contact, nil)

	c, w := setupTestContext("DELETE", "/api/v1/safety/contacts/"+contactID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: contactID.String()}}
	setUserContext(c, userID)

	handler.DeleteEmergencyContact(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// VerifyEmergencyContact Handler Tests
// ============================================================================

func TestHandler_VerifyEmergencyContact_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	contactID := uuid.New()
	reqBody := map[string]interface{}{
		"code": "123456",
	}

	mockRepo.On("VerifyEmergencyContact", mock.Anything, contactID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/safety/contacts/"+contactID.String()+"/verify", reqBody)
	c.Params = gin.Params{{Key: "id", Value: contactID.String()}}

	handler.VerifyEmergencyContact(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_VerifyEmergencyContact_InvalidContactID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := map[string]interface{}{
		"code": "123456",
	}

	c, w := setupTestContext("POST", "/api/v1/safety/contacts/invalid-uuid/verify", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.VerifyEmergencyContact(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_VerifyEmergencyContact_MissingCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	contactID := uuid.New()
	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/safety/contacts/"+contactID.String()+"/verify", reqBody)
	c.Params = gin.Params{{Key: "id", Value: contactID.String()}}

	handler.VerifyEmergencyContact(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_VerifyEmergencyContact_InvalidCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	contactID := uuid.New()
	reqBody := map[string]interface{}{
		"code": "123456",
	}

	mockRepo.On("VerifyEmergencyContact", mock.Anything, contactID).Return(errors.New("invalid or expired verification code"))

	c, w := setupTestContext("POST", "/api/v1/safety/contacts/"+contactID.String()+"/verify", reqBody)
	c.Params = gin.Params{{Key: "id", Value: contactID.String()}}

	handler.VerifyEmergencyContact(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// CreateShareLink Handler Tests
// ============================================================================

func TestHandler_CreateShareLink_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	rideID := uuid.New()
	reqBody := CreateShareLinkRequest{
		RideID:        rideID,
		ExpiryMinutes: 60,
		ShareLocation: true,
		ShareDriver:   true,
		ShareETA:      true,
		ShareRoute:    true,
	}

	mockRepo.On("CreateRideShareLink", mock.Anything, mock.AnythingOfType("*safety.RideShareLink")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/safety/share", reqBody)
	setUserContext(c, userID)

	handler.CreateShareLink(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_CreateShareLink_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	rideID := uuid.New()
	reqBody := CreateShareLinkRequest{
		RideID:        rideID,
		ShareLocation: true,
	}

	c, w := setupTestContext("POST", "/api/v1/safety/share", reqBody)
	// Don't set user context

	handler.CreateShareLink(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CreateShareLink_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/safety/share", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/safety/share", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID)

	handler.CreateShareLink(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateShareLink_MissingRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"share_location": true,
	}

	c, w := setupTestContext("POST", "/api/v1/safety/share", reqBody)
	setUserContext(c, userID)

	handler.CreateShareLink(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateShareLink_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	rideID := uuid.New()
	reqBody := CreateShareLinkRequest{
		RideID:        rideID,
		ShareLocation: true,
	}

	mockRepo.On("CreateRideShareLink", mock.Anything, mock.AnythingOfType("*safety.RideShareLink")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/safety/share", reqBody)
	setUserContext(c, userID)

	handler.CreateShareLink(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetShareLinks Handler Tests
// ============================================================================

func TestHandler_GetShareLinks_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	rideID := uuid.New()
	links := []*RideShareLink{
		createTestRideShareLink(userID, rideID),
	}

	mockRepo.On("GetRideShareLinks", mock.Anything, rideID).Return(links, nil)

	c, w := setupTestContext("GET", "/api/v1/safety/share/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}

	handler.GetShareLinks(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetShareLinks_InvalidRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/safety/share/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.GetShareLinks(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetShareLinks_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	rideID := uuid.New()

	mockRepo.On("GetRideShareLinks", mock.Anything, rideID).Return([]*RideShareLink{}, nil)

	c, w := setupTestContext("GET", "/api/v1/safety/share/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}

	handler.GetShareLinks(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetShareLinks_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	rideID := uuid.New()

	mockRepo.On("GetRideShareLinks", mock.Anything, rideID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/safety/share/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}

	handler.GetShareLinks(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// DeactivateShareLink Handler Tests
// ============================================================================

func TestHandler_DeactivateShareLink_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	linkID := uuid.New()

	mockRepo.On("DeactivateShareLink", mock.Anything, linkID).Return(nil)

	c, w := setupTestContext("DELETE", "/api/v1/safety/share/"+linkID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: linkID.String()}}
	setUserContext(c, userID)

	handler.DeactivateShareLink(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_DeactivateShareLink_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	linkID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/safety/share/"+linkID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: linkID.String()}}
	// Don't set user context

	handler.DeactivateShareLink(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_DeactivateShareLink_InvalidLinkID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/safety/share/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID)

	handler.DeactivateShareLink(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DeactivateShareLink_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	linkID := uuid.New()

	mockRepo.On("DeactivateShareLink", mock.Anything, linkID).Return(errors.New("database error"))

	c, w := setupTestContext("DELETE", "/api/v1/safety/share/"+linkID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: linkID.String()}}
	setUserContext(c, userID)

	handler.DeactivateShareLink(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// ViewSharedRide Handler Tests (Public)
// ============================================================================

func TestHandler_ViewSharedRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	rideID := uuid.New()
	token := "test-token-12345"
	link := createTestRideShareLink(userID, rideID)
	link.Token = token

	mockRepo.On("GetRideShareLinkByToken", mock.Anything, token).Return(link, nil)
	// IncrementShareLinkViewCount is called in a goroutine, so use Maybe()
	mockRepo.On("IncrementShareLinkViewCount", mock.Anything, link.ID).Return(nil).Maybe()

	c, w := setupTestContext("GET", "/share/"+token, nil)
	c.Params = gin.Params{{Key: "token", Value: token}}

	handler.ViewSharedRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	// Wait for goroutine to complete
	time.Sleep(10 * time.Millisecond)
}

func TestHandler_ViewSharedRide_EmptyToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/share/", nil)
	c.Params = gin.Params{{Key: "token", Value: ""}}

	handler.ViewSharedRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ViewSharedRide_TokenNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	token := "nonexistent-token"

	mockRepo.On("GetRideShareLinkByToken", mock.Anything, token).Return(nil, nil)

	c, w := setupTestContext("GET", "/share/"+token, nil)
	c.Params = gin.Params{{Key: "token", Value: token}}

	handler.ViewSharedRide(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_ViewSharedRide_ExpiredLink(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	token := "expired-token"

	mockRepo.On("GetRideShareLinkByToken", mock.Anything, token).Return(nil, errors.New("share link not found or expired"))

	c, w := setupTestContext("GET", "/share/"+token, nil)
	c.Params = gin.Params{{Key: "token", Value: token}}

	handler.ViewSharedRide(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetSafetySettings Handler Tests
// ============================================================================

func TestHandler_GetSafetySettings_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	settings := createTestSafetySettings(userID)

	mockRepo.On("GetSafetySettings", mock.Anything, userID).Return(settings, nil)

	c, w := setupTestContext("GET", "/api/v1/safety/settings", nil)
	setUserContext(c, userID)

	handler.GetSafetySettings(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetSafetySettings_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/safety/settings", nil)
	// Don't set user context

	handler.GetSafetySettings(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetSafetySettings_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetSafetySettings", mock.Anything, userID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/safety/settings", nil)
	setUserContext(c, userID)

	handler.GetSafetySettings(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// UpdateSafetySettings Handler Tests
// ============================================================================

func TestHandler_UpdateSafetySettings_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	settings := createTestSafetySettings(userID)
	sosEnabled := true
	reqBody := UpdateSafetySettingsRequest{
		SOSEnabled: &sosEnabled,
	}

	mockRepo.On("GetSafetySettings", mock.Anything, userID).Return(settings, nil)
	mockRepo.On("UpdateSafetySettings", mock.Anything, mock.AnythingOfType("*safety.SafetySettings")).Return(nil)

	c, w := setupTestContext("PUT", "/api/v1/safety/settings", reqBody)
	setUserContext(c, userID)

	handler.UpdateSafetySettings(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_UpdateSafetySettings_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	sosEnabled := true
	reqBody := UpdateSafetySettingsRequest{
		SOSEnabled: &sosEnabled,
	}

	c, w := setupTestContext("PUT", "/api/v1/safety/settings", reqBody)
	// Don't set user context

	handler.UpdateSafetySettings(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_UpdateSafetySettings_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("PUT", "/api/v1/safety/settings", nil)
	c.Request = httptest.NewRequest("PUT", "/api/v1/safety/settings", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID)

	handler.UpdateSafetySettings(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateSafetySettings_MultipleFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	settings := createTestSafetySettings(userID)
	sosEnabled := true
	sosGesture := "power_button"
	periodicCheckIn := true
	checkInInterval := 10

	reqBody := UpdateSafetySettingsRequest{
		SOSEnabled:          &sosEnabled,
		SOSGesture:          &sosGesture,
		PeriodicCheckIn:     &periodicCheckIn,
		CheckInIntervalMins: &checkInInterval,
	}

	mockRepo.On("GetSafetySettings", mock.Anything, userID).Return(settings, nil)
	mockRepo.On("UpdateSafetySettings", mock.Anything, mock.AnythingOfType("*safety.SafetySettings")).Return(nil)

	c, w := setupTestContext("PUT", "/api/v1/safety/settings", reqBody)
	setUserContext(c, userID)

	handler.UpdateSafetySettings(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_UpdateSafetySettings_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	settings := createTestSafetySettings(userID)
	sosEnabled := true
	reqBody := UpdateSafetySettingsRequest{
		SOSEnabled: &sosEnabled,
	}

	mockRepo.On("GetSafetySettings", mock.Anything, userID).Return(settings, nil)
	mockRepo.On("UpdateSafetySettings", mock.Anything, mock.AnythingOfType("*safety.SafetySettings")).Return(errors.New("database error"))

	c, w := setupTestContext("PUT", "/api/v1/safety/settings", reqBody)
	setUserContext(c, userID)

	handler.UpdateSafetySettings(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// RespondToSafetyCheck Handler Tests
// ============================================================================

func TestHandler_RespondToSafetyCheck_Success_Safe(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	checkID := uuid.New()
	reqBody := SafetyCheckResponse{
		CheckID:  checkID,
		Response: "safe",
	}

	mockRepo.On("UpdateSafetyCheckResponse", mock.Anything, checkID, SafetyCheckSafe, "").Return(nil)

	c, w := setupTestContext("POST", "/api/v1/safety/checks/respond", reqBody)
	setUserContext(c, userID)

	handler.RespondToSafetyCheck(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_RespondToSafetyCheck_Success_Help(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	checkID := uuid.New()
	reqBody := SafetyCheckResponse{
		CheckID:  checkID,
		Response: "help",
		Notes:    "I need assistance",
	}

	mockRepo.On("UpdateSafetyCheckResponse", mock.Anything, checkID, SafetyCheckHelp, "I need assistance").Return(nil)
	// EscalateSafetyCheck is called in a goroutine, so we allow it but don't assert
	mockRepo.On("EscalateSafetyCheck", mock.Anything, mock.Anything).Return(nil).Maybe()

	c, w := setupTestContext("POST", "/api/v1/safety/checks/respond", reqBody)
	setUserContext(c, userID)

	handler.RespondToSafetyCheck(c)

	assert.Equal(t, http.StatusOK, w.Code)
	// Only assert the synchronous call
	mockRepo.AssertCalled(t, "UpdateSafetyCheckResponse", mock.Anything, checkID, SafetyCheckHelp, "I need assistance")
}

func TestHandler_RespondToSafetyCheck_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	checkID := uuid.New()
	reqBody := SafetyCheckResponse{
		CheckID:  checkID,
		Response: "safe",
	}

	c, w := setupTestContext("POST", "/api/v1/safety/checks/respond", reqBody)
	// Don't set user context

	handler.RespondToSafetyCheck(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_RespondToSafetyCheck_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/safety/checks/respond", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/safety/checks/respond", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID)

	handler.RespondToSafetyCheck(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RespondToSafetyCheck_MissingCheckID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"response": "safe",
	}

	c, w := setupTestContext("POST", "/api/v1/safety/checks/respond", reqBody)
	setUserContext(c, userID)

	handler.RespondToSafetyCheck(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RespondToSafetyCheck_MissingResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	checkID := uuid.New()
	reqBody := map[string]interface{}{
		"check_id": checkID.String(),
	}

	c, w := setupTestContext("POST", "/api/v1/safety/checks/respond", reqBody)
	setUserContext(c, userID)

	handler.RespondToSafetyCheck(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RespondToSafetyCheck_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	checkID := uuid.New()
	reqBody := SafetyCheckResponse{
		CheckID:  checkID,
		Response: "safe",
	}

	mockRepo.On("UpdateSafetyCheckResponse", mock.Anything, checkID, SafetyCheckSafe, "").Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/safety/checks/respond", reqBody)
	setUserContext(c, userID)

	handler.RespondToSafetyCheck(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetPendingSafetyChecks Handler Tests
// ============================================================================

func TestHandler_GetPendingSafetyChecks_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	rideID := uuid.New()
	checks := []*SafetyCheck{
		createTestSafetyCheck(userID, rideID),
	}

	mockRepo.On("GetPendingSafetyChecks", mock.Anything, userID).Return(checks, nil)

	c, w := setupTestContext("GET", "/api/v1/safety/checks/pending", nil)
	setUserContext(c, userID)

	handler.GetPendingSafetyChecks(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetPendingSafetyChecks_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/safety/checks/pending", nil)
	// Don't set user context

	handler.GetPendingSafetyChecks(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetPendingSafetyChecks_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetPendingSafetyChecks", mock.Anything, userID).Return([]*SafetyCheck{}, nil)

	c, w := setupTestContext("GET", "/api/v1/safety/checks/pending", nil)
	setUserContext(c, userID)

	handler.GetPendingSafetyChecks(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetPendingSafetyChecks_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetPendingSafetyChecks", mock.Anything, userID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/safety/checks/pending", nil)
	setUserContext(c, userID)

	handler.GetPendingSafetyChecks(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// AddTrustedDriver Handler Tests
// ============================================================================

func TestHandler_AddTrustedDriver_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	reqBody := map[string]interface{}{
		"driver_id": driverID.String(),
		"note":      "Great driver",
	}

	mockRepo.On("AddTrustedDriver", mock.Anything, userID, driverID, "Great driver").Return(nil)

	c, w := setupTestContext("POST", "/api/v1/safety/drivers/trust", reqBody)
	setUserContext(c, userID)

	handler.AddTrustedDriver(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_AddTrustedDriver_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	reqBody := map[string]interface{}{
		"driver_id": driverID.String(),
	}

	c, w := setupTestContext("POST", "/api/v1/safety/drivers/trust", reqBody)
	// Don't set user context

	handler.AddTrustedDriver(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_AddTrustedDriver_MissingDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"note": "Great driver",
	}

	c, w := setupTestContext("POST", "/api/v1/safety/drivers/trust", reqBody)
	setUserContext(c, userID)

	handler.AddTrustedDriver(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AddTrustedDriver_InvalidDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"driver_id": "invalid-uuid",
	}

	c, w := setupTestContext("POST", "/api/v1/safety/drivers/trust", reqBody)
	setUserContext(c, userID)

	handler.AddTrustedDriver(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AddTrustedDriver_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	reqBody := map[string]interface{}{
		"driver_id": driverID.String(),
	}

	mockRepo.On("AddTrustedDriver", mock.Anything, userID, driverID, "").Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/safety/drivers/trust", reqBody)
	setUserContext(c, userID)

	handler.AddTrustedDriver(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// RemoveTrustedDriver Handler Tests
// ============================================================================

func TestHandler_RemoveTrustedDriver_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()

	mockRepo.On("RemoveTrustedDriver", mock.Anything, userID, driverID).Return(nil)

	c, w := setupTestContext("DELETE", "/api/v1/safety/drivers/trust/"+driverID.String(), nil)
	c.Params = gin.Params{{Key: "driver_id", Value: driverID.String()}}
	setUserContext(c, userID)

	handler.RemoveTrustedDriver(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_RemoveTrustedDriver_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/safety/drivers/trust/"+driverID.String(), nil)
	c.Params = gin.Params{{Key: "driver_id", Value: driverID.String()}}
	// Don't set user context

	handler.RemoveTrustedDriver(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_RemoveTrustedDriver_InvalidDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/safety/drivers/trust/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "driver_id", Value: "invalid-uuid"}}
	setUserContext(c, userID)

	handler.RemoveTrustedDriver(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// BlockDriver Handler Tests
// ============================================================================

func TestHandler_BlockDriver_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	reqBody := map[string]interface{}{
		"driver_id": driverID.String(),
		"reason":    "Unsafe driving",
	}

	mockRepo.On("RemoveTrustedDriver", mock.Anything, userID, driverID).Return(nil)
	mockRepo.On("AddBlockedDriver", mock.Anything, userID, driverID, "Unsafe driving").Return(nil)

	c, w := setupTestContext("POST", "/api/v1/safety/drivers/block", reqBody)
	setUserContext(c, userID)

	handler.BlockDriver(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_BlockDriver_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	reqBody := map[string]interface{}{
		"driver_id": driverID.String(),
	}

	c, w := setupTestContext("POST", "/api/v1/safety/drivers/block", reqBody)
	// Don't set user context

	handler.BlockDriver(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_BlockDriver_MissingDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"reason": "Unsafe driving",
	}

	c, w := setupTestContext("POST", "/api/v1/safety/drivers/block", reqBody)
	setUserContext(c, userID)

	handler.BlockDriver(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_BlockDriver_InvalidDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"driver_id": "invalid-uuid",
	}

	c, w := setupTestContext("POST", "/api/v1/safety/drivers/block", reqBody)
	setUserContext(c, userID)

	handler.BlockDriver(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// UnblockDriver Handler Tests
// ============================================================================

func TestHandler_UnblockDriver_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	driverID := uuid.New()

	mockRepo.On("RemoveBlockedDriver", mock.Anything, userID, driverID).Return(nil)

	c, w := setupTestContext("DELETE", "/api/v1/safety/drivers/block/"+driverID.String(), nil)
	c.Params = gin.Params{{Key: "driver_id", Value: driverID.String()}}
	setUserContext(c, userID)

	handler.UnblockDriver(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_UnblockDriver_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/safety/drivers/block/"+driverID.String(), nil)
	c.Params = gin.Params{{Key: "driver_id", Value: driverID.String()}}
	// Don't set user context

	handler.UnblockDriver(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_UnblockDriver_InvalidDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/safety/drivers/block/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "driver_id", Value: "invalid-uuid"}}
	setUserContext(c, userID)

	handler.UnblockDriver(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// GetBlockedDrivers Handler Tests
// ============================================================================

func TestHandler_GetBlockedDrivers_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	blockedDrivers := []uuid.UUID{uuid.New(), uuid.New()}

	mockRepo.On("GetBlockedDrivers", mock.Anything, userID).Return(blockedDrivers, nil)

	c, w := setupTestContext("GET", "/api/v1/safety/drivers/blocked", nil)
	setUserContext(c, userID)

	handler.GetBlockedDrivers(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetBlockedDrivers_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/safety/drivers/blocked", nil)
	// Don't set user context

	handler.GetBlockedDrivers(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetBlockedDrivers_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetBlockedDrivers", mock.Anything, userID).Return([]uuid.UUID{}, nil)

	c, w := setupTestContext("GET", "/api/v1/safety/drivers/blocked", nil)
	setUserContext(c, userID)

	handler.GetBlockedDrivers(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetBlockedDrivers_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetBlockedDrivers", mock.Anything, userID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/safety/drivers/blocked", nil)
	setUserContext(c, userID)

	handler.GetBlockedDrivers(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// ReportIncident Handler Tests
// ============================================================================

func TestHandler_ReportIncident_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	rideID := uuid.New()
	reportedUserID := uuid.New()
	reqBody := ReportIncidentRequest{
		RideID:         &rideID,
		ReportedUserID: &reportedUserID,
		Category:       "harassment",
		Severity:       "high",
		Description:    "Driver was verbally abusive",
	}

	mockRepo.On("CreateSafetyIncidentReport", mock.Anything, mock.AnythingOfType("*safety.SafetyIncidentReport")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/safety/incidents", reqBody)
	setUserContext(c, userID)

	handler.ReportIncident(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_ReportIncident_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := ReportIncidentRequest{
		Category:    "harassment",
		Severity:    "high",
		Description: "Driver was verbally abusive",
	}

	c, w := setupTestContext("POST", "/api/v1/safety/incidents", reqBody)
	// Don't set user context

	handler.ReportIncident(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_ReportIncident_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/safety/incidents", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/safety/incidents", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID)

	handler.ReportIncident(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ReportIncident_MissingCategory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"severity":    "high",
		"description": "Driver was verbally abusive",
	}

	c, w := setupTestContext("POST", "/api/v1/safety/incidents", reqBody)
	setUserContext(c, userID)

	handler.ReportIncident(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ReportIncident_MissingSeverity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"category":    "harassment",
		"description": "Driver was verbally abusive",
	}

	c, w := setupTestContext("POST", "/api/v1/safety/incidents", reqBody)
	setUserContext(c, userID)

	handler.ReportIncident(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ReportIncident_MissingDescription(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"category": "harassment",
		"severity": "high",
	}

	c, w := setupTestContext("POST", "/api/v1/safety/incidents", reqBody)
	setUserContext(c, userID)

	handler.ReportIncident(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ReportIncident_WithPhotos(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := ReportIncidentRequest{
		Category:    "harassment",
		Severity:    "high",
		Description: "Driver was verbally abusive",
		Photos:      []string{"https://example.com/photo1.jpg", "https://example.com/photo2.jpg"},
	}

	mockRepo.On("CreateSafetyIncidentReport", mock.Anything, mock.AnythingOfType("*safety.SafetyIncidentReport")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/safety/incidents", reqBody)
	setUserContext(c, userID)

	handler.ReportIncident(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_ReportIncident_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := ReportIncidentRequest{
		Category:    "harassment",
		Severity:    "high",
		Description: "Driver was verbally abusive",
	}

	mockRepo.On("CreateSafetyIncidentReport", mock.Anything, mock.AnythingOfType("*safety.SafetyIncidentReport")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/safety/incidents", reqBody)
	setUserContext(c, userID)

	handler.ReportIncident(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// Admin Endpoints Tests
// ============================================================================

func TestHandler_GetActiveEmergencies_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	alerts := []*EmergencyAlert{
		createTestEmergencyAlert(uuid.New(), EmergencyStatusActive),
		createTestEmergencyAlert(uuid.New(), EmergencyStatusActive),
	}

	mockRepo.On("GetActiveEmergencyAlerts", mock.Anything).Return(alerts, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/safety/emergencies", nil)

	handler.GetActiveEmergencies(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetActiveEmergencies_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	mockRepo.On("GetActiveEmergencyAlerts", mock.Anything).Return([]*EmergencyAlert{}, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/safety/emergencies", nil)

	handler.GetActiveEmergencies(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetActiveEmergencies_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	mockRepo.On("GetActiveEmergencyAlerts", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/safety/emergencies", nil)

	handler.GetActiveEmergencies(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_RespondToEmergency_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	alertID := uuid.New()

	mockRepo.On("UpdateEmergencyAlertStatus", mock.Anything, alertID, EmergencyStatusResponded, &adminID, "").Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/safety/emergencies/"+alertID.String()+"/respond", nil)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}
	setAdminContext(c, adminID)

	handler.RespondToEmergency(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_RespondToEmergency_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	alertID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/admin/safety/emergencies/"+alertID.String()+"/respond", nil)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}
	// Don't set admin context

	handler.RespondToEmergency(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_RespondToEmergency_InvalidAlertID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/admin/safety/emergencies/invalid-uuid/respond", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setAdminContext(c, adminID)

	handler.RespondToEmergency(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ResolveEmergency_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	alertID := uuid.New()
	reqBody := map[string]interface{}{
		"resolution":     "Police responded and situation resolved",
		"is_false_alarm": false,
	}

	mockRepo.On("UpdateEmergencyAlertStatus", mock.Anything, alertID, EmergencyStatusResolved, &adminID, "Police responded and situation resolved").Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/safety/emergencies/"+alertID.String()+"/resolve", reqBody)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}
	setAdminContext(c, adminID)

	handler.ResolveEmergency(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_ResolveEmergency_FalseAlarm(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	alertID := uuid.New()
	reqBody := map[string]interface{}{
		"resolution":     "Accidental trigger confirmed by user",
		"is_false_alarm": true,
	}

	mockRepo.On("UpdateEmergencyAlertStatus", mock.Anything, alertID, EmergencyStatusFalse, &adminID, "Accidental trigger confirmed by user").Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/safety/emergencies/"+alertID.String()+"/resolve", reqBody)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}
	setAdminContext(c, adminID)

	handler.ResolveEmergency(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_ResolveEmergency_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	alertID := uuid.New()
	reqBody := map[string]interface{}{
		"resolution": "Resolved",
	}

	c, w := setupTestContext("POST", "/api/v1/admin/safety/emergencies/"+alertID.String()+"/resolve", reqBody)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}
	// Don't set admin context

	handler.ResolveEmergency(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_ResolveEmergency_InvalidAlertID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	reqBody := map[string]interface{}{
		"resolution": "Resolved",
	}

	c, w := setupTestContext("POST", "/api/v1/admin/safety/emergencies/invalid-uuid/resolve", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setAdminContext(c, adminID)

	handler.ResolveEmergency(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ResolveEmergency_MissingResolution(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	alertID := uuid.New()
	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/admin/safety/emergencies/"+alertID.String()+"/resolve", reqBody)
	c.Params = gin.Params{{Key: "id", Value: alertID.String()}}
	setAdminContext(c, adminID)

	handler.ResolveEmergency(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminUpdateIncident_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	incidentID := uuid.New()
	reqBody := map[string]interface{}{
		"status":       "resolved",
		"resolution":   "Issue addressed with driver",
		"action_taken": "Warning issued to driver",
	}

	mockRepo.On("UpdateSafetyIncidentStatus", mock.Anything, incidentID, "resolved", "Issue addressed with driver", "Warning issued to driver").Return(nil)

	c, w := setupTestContext("PUT", "/api/v1/admin/safety/incidents/"+incidentID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: incidentID.String()}}

	handler.AdminUpdateIncident(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_AdminUpdateIncident_InvalidIncidentID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := map[string]interface{}{
		"status": "resolved",
	}

	c, w := setupTestContext("PUT", "/api/v1/admin/safety/incidents/invalid-uuid", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.AdminUpdateIncident(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminUpdateIncident_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	incidentID := uuid.New()
	reqBody := map[string]interface{}{
		"status": "resolved",
	}

	mockRepo.On("UpdateSafetyIncidentStatus", mock.Anything, incidentID, "resolved", "", "").Return(errors.New("database error"))

	c, w := setupTestContext("PUT", "/api/v1/admin/safety/incidents/"+incidentID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: incidentID.String()}}

	handler.AdminUpdateIncident(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// Edge Cases and Additional Tests
// ============================================================================

func TestHandler_TriggerSOS_WithDescription(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := TriggerSOSRequest{
		Type:        EmergencySOS,
		Latitude:    40.7128,
		Longitude:   -74.0060,
		Description: "I feel unsafe, driver is acting strange",
	}

	mockRepo.On("CreateEmergencyAlert", mock.Anything, mock.AnythingOfType("*safety.EmergencyAlert")).Return(nil)
	mockRepo.On("GetSOSContacts", mock.Anything, userID).Return([]*EmergencyContact{}, nil)

	c, w := setupTestContext("POST", "/api/v1/safety/sos", reqBody)
	setUserContext(c, userID)

	handler.TriggerSOS(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_CreateEmergencyContact_WithOptionalFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := CreateEmergencyContactRequest{
		Name:         "Jane Doe",
		Phone:        "+1234567890",
		Email:        "jane@example.com",
		Relationship: "family",
		IsPrimary:    true,
		NotifyOnRide: true,
		NotifyOnSOS:  true,
	}

	mockRepo.On("GetUserEmergencyContacts", mock.Anything, userID).Return([]*EmergencyContact{}, nil)
	mockRepo.On("CreateEmergencyContact", mock.Anything, mock.AnythingOfType("*safety.EmergencyContact")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/safety/contacts", reqBody)
	setUserContext(c, userID)

	handler.CreateEmergencyContact(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_CreateShareLink_WithRecipients(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	rideID := uuid.New()
	reqBody := CreateShareLinkRequest{
		RideID:        rideID,
		ExpiryMinutes: 60,
		ShareLocation: true,
		ShareDriver:   true,
		SendTo: []ShareRecipientInput{
			{Phone: "+1234567890"},
			{Email: "friend@example.com"},
		},
	}

	mockRepo.On("CreateRideShareLink", mock.Anything, mock.AnythingOfType("*safety.RideShareLink")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/safety/share", reqBody)
	setUserContext(c, userID)

	handler.CreateShareLink(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_EmptyUUID_Validation(t *testing.T) {
	tests := []struct {
		name     string
		handler  func(*Handler, *gin.Context)
		paramKey string
	}{
		{
			name: "GetEmergency with empty ID",
			handler: func(h *Handler, c *gin.Context) {
				h.GetEmergency(c)
			},
			paramKey: "id",
		},
		{
			name: "CancelSOS with empty ID",
			handler: func(h *Handler, c *gin.Context) {
				c.Set("user_id", uuid.New())
				h.CancelSOS(c)
			},
			paramKey: "id",
		},
		{
			name: "DeleteEmergencyContact with empty ID",
			handler: func(h *Handler, c *gin.Context) {
				c.Set("user_id", uuid.New())
				h.DeleteEmergencyContact(c)
			},
			paramKey: "id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			c, w := setupTestContext("GET", "/test", nil)
			c.Params = gin.Params{{Key: tt.paramKey, Value: ""}}

			tt.handler(handler, c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandler_InvalidUUIDFormats(t *testing.T) {
	tests := []struct {
		name string
		uuid string
	}{
		{"short uuid", "1234-5678"},
		{"wrong format", "1234-5678-9012-3456"},
		{"placeholder x", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"},
		{"partial uuid", "123e4567-e89b-12d3-a456"},
		{"simple string", "invalid"},
		{"empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			c, w := setupTestContext("GET", "/api/v1/safety/sos/test", nil)
			c.Params = gin.Params{{Key: "id", Value: tt.uuid}}

			handler.GetEmergency(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandler_AllIncidentCategories(t *testing.T) {
	categories := []string{
		"harassment",
		"unsafe_driving",
		"discrimination",
		"assault",
		"theft",
		"other",
	}

	for _, category := range categories {
		t.Run(category, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			userID := uuid.New()
			reqBody := ReportIncidentRequest{
				Category:    category,
				Severity:    "medium",
				Description: "Test incident for " + category,
			}

			mockRepo.On("CreateSafetyIncidentReport", mock.Anything, mock.AnythingOfType("*safety.SafetyIncidentReport")).Return(nil)

			c, w := setupTestContext("POST", "/api/v1/safety/incidents", reqBody)
			setUserContext(c, userID)

			handler.ReportIncident(c)

			assert.Equal(t, http.StatusCreated, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestHandler_AllIncidentSeverities(t *testing.T) {
	severities := []string{
		"low",
		"medium",
		"high",
		"critical",
	}

	for _, severity := range severities {
		t.Run(severity, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			userID := uuid.New()
			reqBody := ReportIncidentRequest{
				Category:    "harassment",
				Severity:    severity,
				Description: "Test incident with severity " + severity,
			}

			mockRepo.On("CreateSafetyIncidentReport", mock.Anything, mock.AnythingOfType("*safety.SafetyIncidentReport")).Return(nil)

			c, w := setupTestContext("POST", "/api/v1/safety/incidents", reqBody)
			setUserContext(c, userID)

			handler.ReportIncident(c)

			assert.Equal(t, http.StatusCreated, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}
