package verification

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
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// MOCK REPOSITORY
// ============================================================================

// MockRepository is a mock implementation of RepositoryInterface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateBackgroundCheck(ctx context.Context, check *BackgroundCheck) error {
	args := m.Called(ctx, check)
	return args.Error(0)
}

func (m *MockRepository) GetBackgroundCheck(ctx context.Context, checkID uuid.UUID) (*BackgroundCheck, error) {
	args := m.Called(ctx, checkID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BackgroundCheck), args.Error(1)
}

func (m *MockRepository) GetBackgroundCheckByExternalID(ctx context.Context, provider BackgroundCheckProvider, externalID string) (*BackgroundCheck, error) {
	args := m.Called(ctx, provider, externalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BackgroundCheck), args.Error(1)
}

func (m *MockRepository) GetLatestBackgroundCheck(ctx context.Context, driverID uuid.UUID) (*BackgroundCheck, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BackgroundCheck), args.Error(1)
}

func (m *MockRepository) UpdateBackgroundCheckStatus(ctx context.Context, checkID uuid.UUID, status BackgroundCheckStatus, notes *string, failureReasons []string) error {
	args := m.Called(ctx, checkID, status, notes, failureReasons)
	return args.Error(0)
}

func (m *MockRepository) UpdateBackgroundCheckStarted(ctx context.Context, checkID uuid.UUID, externalID string) error {
	args := m.Called(ctx, checkID, externalID)
	return args.Error(0)
}

func (m *MockRepository) UpdateBackgroundCheckCompleted(ctx context.Context, checkID uuid.UUID, status BackgroundCheckStatus, reportURL *string, expiresAt *time.Time) error {
	args := m.Called(ctx, checkID, status, reportURL, expiresAt)
	return args.Error(0)
}

func (m *MockRepository) GetPendingBackgroundChecks(ctx context.Context, limit int) ([]*BackgroundCheck, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*BackgroundCheck), args.Error(1)
}

func (m *MockRepository) CreateSelfieVerification(ctx context.Context, verification *SelfieVerification) error {
	args := m.Called(ctx, verification)
	return args.Error(0)
}

func (m *MockRepository) GetSelfieVerification(ctx context.Context, verificationID uuid.UUID) (*SelfieVerification, error) {
	args := m.Called(ctx, verificationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SelfieVerification), args.Error(1)
}

func (m *MockRepository) GetLatestSelfieVerification(ctx context.Context, driverID uuid.UUID) (*SelfieVerification, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SelfieVerification), args.Error(1)
}

func (m *MockRepository) GetTodaysSelfieVerification(ctx context.Context, driverID uuid.UUID) (*SelfieVerification, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SelfieVerification), args.Error(1)
}

func (m *MockRepository) UpdateSelfieVerificationResult(ctx context.Context, verificationID uuid.UUID, status SelfieVerificationStatus, confidenceScore *float64, matchResult *bool, failureReason *string) error {
	args := m.Called(ctx, verificationID, status, confidenceScore, matchResult, failureReason)
	return args.Error(0)
}

func (m *MockRepository) GetDriverReferencePhoto(ctx context.Context, driverID uuid.UUID) (*string, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*string), args.Error(1)
}

func (m *MockRepository) GetDriverVerificationStatus(ctx context.Context, driverID uuid.UUID) (*DriverVerificationStatus, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverVerificationStatus), args.Error(1)
}

func (m *MockRepository) UpdateDriverApproval(ctx context.Context, driverID uuid.UUID, approved bool, approvedBy *uuid.UUID, reason *string) error {
	args := m.Called(ctx, driverID, approved, approvedBy, reason)
	return args.Error(0)
}

// ============================================================================
// HELPER FUNCTIONS
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

func setUserContext(c *gin.Context, userID uuid.UUID, role models.UserRole) {
	c.Set("user_id", userID)
	c.Set("user_role", role)
	c.Set("user_email", "test@example.com")
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

func createTestHandler(mockRepo *MockRepository) *Handler {
	service := NewService(mockRepo, nil)
	return NewHandler(service)
}

func createTestBackgroundCheck(driverID uuid.UUID, status BackgroundCheckStatus) *BackgroundCheck {
	now := time.Now()
	externalID := "ext-123"
	return &BackgroundCheck{
		ID:         uuid.New(),
		DriverID:   driverID,
		Provider:   ProviderCheckr,
		ExternalID: &externalID,
		Status:     status,
		CheckType:  "driver_standard",
		StartedAt:  &now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func createTestSelfieVerification(driverID uuid.UUID, status SelfieVerificationStatus) *SelfieVerification {
	now := time.Now()
	confidence := 95.5
	match := true
	refURL := "https://example.com/reference.jpg"
	return &SelfieVerification{
		ID:                driverID,
		DriverID:          driverID,
		SelfieURL:         "https://example.com/selfie.jpg",
		ReferencePhotoURL: &refURL,
		Status:            status,
		ConfidenceScore:   &confidence,
		MatchResult:       &match,
		Provider:          "rekognition",
		CreatedAt:         now,
	}
}

func createTestDriverVerificationStatus(driverID uuid.UUID) *DriverVerificationStatus {
	return &DriverVerificationStatus{
		DriverID:                   driverID,
		BackgroundCheckStatus:      BGCheckStatusPassed,
		SelfieVerificationRequired: false,
		IsFullyVerified:            true,
		VerificationMessage:        "All verifications complete",
	}
}

// ============================================================================
// GetVerificationStatus Tests
// ============================================================================

func TestHandler_GetVerificationStatus_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	status := createTestDriverVerificationStatus(driverID)

	mockRepo.On("GetDriverVerificationStatus", mock.Anything, driverID).Return(status, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/verification/status", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetVerificationStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetVerificationStatus_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/driver/verification/status", nil)
	// Don't set user context to simulate unauthorized

	handler.GetVerificationStatus(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_GetVerificationStatus_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	mockRepo.On("GetDriverVerificationStatus", mock.Anything, driverID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/verification/status", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetVerificationStatus(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_GetVerificationStatus_AppError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	mockRepo.On("GetDriverVerificationStatus", mock.Anything, driverID).Return(nil, common.NewNotFoundError("driver not found", nil))

	c, w := setupTestContext("GET", "/api/v1/driver/verification/status", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetVerificationStatus(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ============================================================================
// SubmitSelfie Tests
// ============================================================================

func TestHandler_SubmitSelfie_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	refURL := "https://example.com/reference.jpg"

	// Note: DriverID in request is required by binding, but overwritten by handler from context
	reqBody := map[string]interface{}{
		"driver_id": driverID.String(),
		"selfie_url": "https://example.com/selfie.jpg",
	}

	mockRepo.On("GetDriverReferencePhoto", mock.Anything, driverID).Return(&refURL, nil)
	mockRepo.On("CreateSelfieVerification", mock.Anything, mock.AnythingOfType("*verification.SelfieVerification")).Return(nil)
	mockRepo.On("UpdateSelfieVerificationResult", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/verification/selfie", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.SubmitSelfie(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_SubmitSelfie_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := SubmitSelfieRequest{
		SelfieURL: "https://example.com/selfie.jpg",
	}

	c, w := setupTestContext("POST", "/api/v1/driver/verification/selfie", reqBody)
	// Don't set user context

	handler.SubmitSelfie(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_SubmitSelfie_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/verification/selfie", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/driver/verification/selfie", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, driverID, models.RoleDriver)

	handler.SubmitSelfie(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid request body")
}

func TestHandler_SubmitSelfie_MissingSelfieURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/driver/verification/selfie", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.SubmitSelfie(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SubmitSelfie_NoReferencePhoto(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	reqBody := map[string]interface{}{
		"driver_id": driverID.String(),
		"selfie_url": "https://example.com/selfie.jpg",
	}

	mockRepo.On("GetDriverReferencePhoto", mock.Anything, driverID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("POST", "/api/v1/driver/verification/selfie", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.SubmitSelfie(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "reference photo not found")
}

func TestHandler_SubmitSelfie_WithRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	rideID := uuid.New()
	refURL := "https://example.com/reference.jpg"

	reqBody := map[string]interface{}{
		"driver_id": driverID.String(),
		"selfie_url": "https://example.com/selfie.jpg",
		"ride_id":   rideID.String(),
	}

	mockRepo.On("GetDriverReferencePhoto", mock.Anything, driverID).Return(&refURL, nil)
	mockRepo.On("CreateSelfieVerification", mock.Anything, mock.AnythingOfType("*verification.SelfieVerification")).Return(nil)
	mockRepo.On("UpdateSelfieVerificationResult", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/verification/selfie", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.SubmitSelfie(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_SubmitSelfie_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	refURL := "https://example.com/reference.jpg"

	reqBody := map[string]interface{}{
		"driver_id": driverID.String(),
		"selfie_url": "https://example.com/selfie.jpg",
	}

	mockRepo.On("GetDriverReferencePhoto", mock.Anything, driverID).Return(&refURL, nil)
	mockRepo.On("CreateSelfieVerification", mock.Anything, mock.AnythingOfType("*verification.SelfieVerification")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/driver/verification/selfie", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.SubmitSelfie(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetSelfieStatus Tests
// ============================================================================

func TestHandler_GetSelfieStatus_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	verificationID := uuid.New()
	verification := createTestSelfieVerification(driverID, SelfieStatusVerified)
	verification.ID = verificationID

	mockRepo.On("GetSelfieVerification", mock.Anything, verificationID).Return(verification, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/verification/selfie/"+verificationID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: verificationID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetSelfieStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetSelfieStatus_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/driver/verification/selfie/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetSelfieStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid verification ID")
}

func TestHandler_GetSelfieStatus_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	verificationID := uuid.New()

	mockRepo.On("GetSelfieVerification", mock.Anything, verificationID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/driver/verification/selfie/"+verificationID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: verificationID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetSelfieStatus(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetSelfieStatus_AppError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	verificationID := uuid.New()

	mockRepo.On("GetSelfieVerification", mock.Anything, verificationID).Return(nil, common.NewNotFoundError("verification not found", nil))

	c, w := setupTestContext("GET", "/api/v1/driver/verification/selfie/"+verificationID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: verificationID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetSelfieStatus(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetSelfieStatus_EmptyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/driver/verification/selfie/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetSelfieStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// CheckSelfieRequired Tests
// ============================================================================

func TestHandler_CheckSelfieRequired_Required(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	mockRepo.On("GetTodaysSelfieVerification", mock.Anything, driverID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/driver/verification/selfie/required", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.CheckSelfieRequired(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.True(t, data["selfie_required"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_CheckSelfieRequired_NotRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	verification := createTestSelfieVerification(driverID, SelfieStatusVerified)

	mockRepo.On("GetTodaysSelfieVerification", mock.Anything, driverID).Return(verification, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/verification/selfie/required", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.CheckSelfieRequired(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.False(t, data["selfie_required"].(bool))
}

func TestHandler_CheckSelfieRequired_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/driver/verification/selfie/required", nil)
	// Don't set user context

	handler.CheckSelfieRequired(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// GetBackgroundStatus Tests
// ============================================================================

func TestHandler_GetBackgroundStatus_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	check := createTestBackgroundCheck(driverID, BGCheckStatusPassed)

	mockRepo.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(check, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/verification/background", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetBackgroundStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetBackgroundStatus_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/driver/verification/background", nil)
	// Don't set user context

	handler.GetBackgroundStatus(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetBackgroundStatus_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	mockRepo.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/driver/verification/background", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetBackgroundStatus(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetBackgroundStatus_FailedWithReasons(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	check := createTestBackgroundCheck(driverID, BGCheckStatusFailed)
	check.FailureReasons = []string{"Criminal record found", "MVR check failed"}

	mockRepo.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(check, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/verification/background", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetBackgroundStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "failed", data["status"])
	assert.NotNil(t, data["failure_reasons"])
}

func TestHandler_GetBackgroundStatus_InProgress(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	check := createTestBackgroundCheck(driverID, BGCheckStatusInProgress)

	mockRepo.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(check, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/verification/background", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetBackgroundStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "in_progress", data["status"])
}

// ============================================================================
// InitiateBackgroundCheck Tests (Admin)
// ============================================================================

func TestHandler_InitiateBackgroundCheck_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	driverID := uuid.New()

	reqBody := InitiateBackgroundCheckRequest{
		DriverID:      driverID,
		FirstName:     "John",
		LastName:      "Doe",
		DateOfBirth:   "1990-01-15",
		Email:         "john.doe@example.com",
		Phone:         "+15551234567",
		StreetAddress: "123 Main St",
		City:          "San Francisco",
		State:         "CA",
		ZipCode:       "94102",
		LicenseNumber: "D1234567",
		LicenseState:  "CA",
		Provider:      ProviderMock,
	}

	mockRepo.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(nil, errors.New("not found"))
	mockRepo.On("CreateBackgroundCheck", mock.Anything, mock.AnythingOfType("*verification.BackgroundCheck")).Return(nil)
	mockRepo.On("UpdateBackgroundCheckStarted", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.Anything).Return(nil)
	mockRepo.On("UpdateBackgroundCheckCompleted", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	mockRepo.On("UpdateDriverApproval", mock.Anything, driverID, true, mock.Anything, mock.Anything).Return(nil).Maybe()

	c, w := setupTestContext("POST", "/api/v1/admin/verification/background", reqBody)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.InitiateBackgroundCheck(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_InitiateBackgroundCheck_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/admin/verification/background", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/verification/background", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, adminID, models.RoleAdmin)

	handler.InitiateBackgroundCheck(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_InitiateBackgroundCheck_MissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()

	reqBody := map[string]interface{}{
		"first_name": "John",
		// Missing required fields
	}

	c, w := setupTestContext("POST", "/api/v1/admin/verification/background", reqBody)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.InitiateBackgroundCheck(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_InitiateBackgroundCheck_CheckAlreadyInProgress(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	driverID := uuid.New()
	existingCheck := createTestBackgroundCheck(driverID, BGCheckStatusInProgress)

	reqBody := InitiateBackgroundCheckRequest{
		DriverID:      driverID,
		FirstName:     "John",
		LastName:      "Doe",
		DateOfBirth:   "1990-01-15",
		Email:         "john.doe@example.com",
		Phone:         "+15551234567",
		StreetAddress: "123 Main St",
		City:          "San Francisco",
		State:         "CA",
		ZipCode:       "94102",
		LicenseNumber: "D1234567",
		LicenseState:  "CA",
	}

	mockRepo.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(existingCheck, nil)

	c, w := setupTestContext("POST", "/api/v1/admin/verification/background", reqBody)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.InitiateBackgroundCheck(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "already in progress")
}

func TestHandler_InitiateBackgroundCheck_CheckPending(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	driverID := uuid.New()
	existingCheck := createTestBackgroundCheck(driverID, BGCheckStatusPending)

	reqBody := InitiateBackgroundCheckRequest{
		DriverID:      driverID,
		FirstName:     "John",
		LastName:      "Doe",
		DateOfBirth:   "1990-01-15",
		Email:         "john.doe@example.com",
		Phone:         "+15551234567",
		StreetAddress: "123 Main St",
		City:          "San Francisco",
		State:         "CA",
		ZipCode:       "94102",
		LicenseNumber: "D1234567",
		LicenseState:  "CA",
	}

	mockRepo.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(existingCheck, nil)

	c, w := setupTestContext("POST", "/api/v1/admin/verification/background", reqBody)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.InitiateBackgroundCheck(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_InitiateBackgroundCheck_UnsupportedProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	driverID := uuid.New()

	reqBody := InitiateBackgroundCheckRequest{
		DriverID:      driverID,
		Provider:      "unsupported_provider",
		FirstName:     "John",
		LastName:      "Doe",
		DateOfBirth:   "1990-01-15",
		Email:         "john.doe@example.com",
		Phone:         "+15551234567",
		StreetAddress: "123 Main St",
		City:          "San Francisco",
		State:         "CA",
		ZipCode:       "94102",
		LicenseNumber: "D1234567",
		LicenseState:  "CA",
	}

	mockRepo.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(nil, errors.New("not found"))
	mockRepo.On("CreateBackgroundCheck", mock.Anything, mock.AnythingOfType("*verification.BackgroundCheck")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/verification/background", reqBody)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.InitiateBackgroundCheck(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_InitiateBackgroundCheck_CreateError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	driverID := uuid.New()

	reqBody := InitiateBackgroundCheckRequest{
		DriverID:      driverID,
		FirstName:     "John",
		LastName:      "Doe",
		DateOfBirth:   "1990-01-15",
		Email:         "john.doe@example.com",
		Phone:         "+15551234567",
		StreetAddress: "123 Main St",
		City:          "San Francisco",
		State:         "CA",
		ZipCode:       "94102",
		LicenseNumber: "D1234567",
		LicenseState:  "CA",
	}

	mockRepo.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(nil, errors.New("not found"))
	mockRepo.On("CreateBackgroundCheck", mock.Anything, mock.AnythingOfType("*verification.BackgroundCheck")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/admin/verification/background", reqBody)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.InitiateBackgroundCheck(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetBackgroundCheck Tests (Admin)
// ============================================================================

func TestHandler_GetBackgroundCheck_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	driverID := uuid.New()
	checkID := uuid.New()
	check := createTestBackgroundCheck(driverID, BGCheckStatusPassed)
	check.ID = checkID

	mockRepo.On("GetBackgroundCheck", mock.Anything, checkID).Return(check, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/verification/background/"+checkID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: checkID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetBackgroundCheck(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetBackgroundCheck_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/admin/verification/background/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetBackgroundCheck(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid check ID")
}

func TestHandler_GetBackgroundCheck_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	checkID := uuid.New()

	mockRepo.On("GetBackgroundCheck", mock.Anything, checkID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/admin/verification/background/"+checkID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: checkID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetBackgroundCheck(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetBackgroundCheck_EmptyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/admin/verification/background/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetBackgroundCheck(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// GetDriverBackgroundCheck Tests (Admin)
// ============================================================================

func TestHandler_GetDriverBackgroundCheck_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	driverID := uuid.New()
	check := createTestBackgroundCheck(driverID, BGCheckStatusPassed)

	mockRepo.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(check, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/verification/driver/"+driverID.String()+"/background", nil)
	c.Params = gin.Params{{Key: "id", Value: driverID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetDriverBackgroundCheck(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetDriverBackgroundCheck_InvalidDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/admin/verification/driver/invalid-uuid/background", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetDriverBackgroundCheck(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid driver ID")
}

func TestHandler_GetDriverBackgroundCheck_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	driverID := uuid.New()

	mockRepo.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/admin/verification/driver/"+driverID.String()+"/background", nil)
	c.Params = gin.Params{{Key: "id", Value: driverID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetDriverBackgroundCheck(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ============================================================================
// GetDriverVerificationStatusAdmin Tests
// ============================================================================

func TestHandler_GetDriverVerificationStatusAdmin_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	driverID := uuid.New()
	status := createTestDriverVerificationStatus(driverID)

	mockRepo.On("GetDriverVerificationStatus", mock.Anything, driverID).Return(status, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/verification/driver/"+driverID.String()+"/status", nil)
	c.Params = gin.Params{{Key: "id", Value: driverID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetDriverVerificationStatusAdmin(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetDriverVerificationStatusAdmin_InvalidDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/admin/verification/driver/invalid-uuid/status", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetDriverVerificationStatusAdmin(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetDriverVerificationStatusAdmin_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	driverID := uuid.New()

	mockRepo.On("GetDriverVerificationStatus", mock.Anything, driverID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/verification/driver/"+driverID.String()+"/status", nil)
	c.Params = gin.Params{{Key: "id", Value: driverID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetDriverVerificationStatusAdmin(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// HandleCheckrWebhook Tests
// ============================================================================

func TestHandler_HandleCheckrWebhook_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	check := createTestBackgroundCheck(driverID, BGCheckStatusInProgress)

	webhookPayload := WebhookPayload{
		EventType:  "report.completed",
		ExternalID: "ext-123",
		Status:     "clear",
		Data:       map[string]interface{}{},
		Timestamp:  time.Now(),
	}

	mockRepo.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderCheckr, "ext-123").Return(check, nil)
	mockRepo.On("UpdateBackgroundCheckCompleted", mock.Anything, check.ID, BGCheckStatusPassed, mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("UpdateDriverApproval", mock.Anything, driverID, true, mock.Anything, mock.Anything).Return(nil)

	c, w := setupTestContext("POST", "/webhooks/checkr", webhookPayload)

	handler.HandleCheckrWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "processed", data["status"])
}

func TestHandler_HandleCheckrWebhook_InvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("POST", "/webhooks/checkr", nil)
	c.Request = httptest.NewRequest("POST", "/webhooks/checkr", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.HandleCheckrWebhook(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid webhook payload")
}

func TestHandler_HandleCheckrWebhook_CheckNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	webhookPayload := WebhookPayload{
		EventType:  "report.completed",
		ExternalID: "unknown-ext-id",
		Status:     "clear",
		Data:       map[string]interface{}{},
		Timestamp:  time.Now(),
	}

	mockRepo.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderCheckr, "unknown-ext-id").Return(nil, errors.New("not found"))

	c, w := setupTestContext("POST", "/webhooks/checkr", webhookPayload)

	handler.HandleCheckrWebhook(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_HandleCheckrWebhook_ConsiderStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	check := createTestBackgroundCheck(driverID, BGCheckStatusInProgress)

	webhookPayload := WebhookPayload{
		EventType:  "report.completed",
		ExternalID: "ext-123",
		Status:     "consider",
		Data:       map[string]interface{}{},
		Timestamp:  time.Now(),
	}

	mockRepo.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderCheckr, "ext-123").Return(check, nil)
	mockRepo.On("UpdateBackgroundCheckCompleted", mock.Anything, check.ID, BGCheckStatusFailed, mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("UpdateBackgroundCheckStatus", mock.Anything, check.ID, BGCheckStatusFailed, mock.Anything, mock.Anything).Return(nil)

	c, w := setupTestContext("POST", "/webhooks/checkr", webhookPayload)

	handler.HandleCheckrWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_HandleCheckrWebhook_PendingStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	check := createTestBackgroundCheck(driverID, BGCheckStatusInProgress)

	webhookPayload := WebhookPayload{
		EventType:  "report.updated",
		ExternalID: "ext-123",
		Status:     "pending",
		Data:       map[string]interface{}{},
		Timestamp:  time.Now(),
	}

	mockRepo.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderCheckr, "ext-123").Return(check, nil)
	mockRepo.On("UpdateBackgroundCheckCompleted", mock.Anything, check.ID, BGCheckStatusInProgress, mock.Anything, mock.Anything).Return(nil)

	c, w := setupTestContext("POST", "/webhooks/checkr", webhookPayload)

	handler.HandleCheckrWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_HandleCheckrWebhook_WithReportURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	check := createTestBackgroundCheck(driverID, BGCheckStatusInProgress)

	webhookPayload := WebhookPayload{
		EventType:  "report.completed",
		ExternalID: "ext-123",
		Status:     "clear",
		Data: map[string]interface{}{
			"report_url": "https://checkr.com/report/123",
		},
		Timestamp: time.Now(),
	}

	mockRepo.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderCheckr, "ext-123").Return(check, nil)
	mockRepo.On("UpdateBackgroundCheckCompleted", mock.Anything, check.ID, BGCheckStatusPassed, mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("UpdateDriverApproval", mock.Anything, driverID, true, mock.Anything, mock.Anything).Return(nil)

	c, w := setupTestContext("POST", "/webhooks/checkr", webhookPayload)

	handler.HandleCheckrWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================================
// HandleSterlingWebhook Tests
// ============================================================================

func TestHandler_HandleSterlingWebhook_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	check := createTestBackgroundCheck(driverID, BGCheckStatusInProgress)
	check.Provider = ProviderSterling

	webhookPayload := WebhookPayload{
		EventType:  "screening.completed",
		ExternalID: "sterling-ext-123",
		Status:     "Completed",
		Data:       map[string]interface{}{},
		Timestamp:  time.Now(),
	}

	mockRepo.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderSterling, "sterling-ext-123").Return(check, nil)
	mockRepo.On("UpdateBackgroundCheckCompleted", mock.Anything, check.ID, BGCheckStatusPassed, mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("UpdateDriverApproval", mock.Anything, driverID, true, mock.Anything, mock.Anything).Return(nil)

	c, w := setupTestContext("POST", "/webhooks/sterling", webhookPayload)

	handler.HandleSterlingWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_HandleSterlingWebhook_InvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("POST", "/webhooks/sterling", nil)
	c.Request = httptest.NewRequest("POST", "/webhooks/sterling", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.HandleSterlingWebhook(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_HandleSterlingWebhook_FailStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	check := createTestBackgroundCheck(driverID, BGCheckStatusInProgress)
	check.Provider = ProviderSterling

	webhookPayload := WebhookPayload{
		EventType:  "screening.completed",
		ExternalID: "sterling-ext-123",
		Status:     "Fail",
		Data:       map[string]interface{}{},
		Timestamp:  time.Now(),
	}

	mockRepo.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderSterling, "sterling-ext-123").Return(check, nil)
	mockRepo.On("UpdateBackgroundCheckCompleted", mock.Anything, check.ID, BGCheckStatusFailed, mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("UpdateBackgroundCheckStatus", mock.Anything, check.ID, BGCheckStatusFailed, mock.Anything, mock.Anything).Return(nil)

	c, w := setupTestContext("POST", "/webhooks/sterling", webhookPayload)

	handler.HandleSterlingWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================================
// HandleOnfidoWebhook Tests
// ============================================================================

func TestHandler_HandleOnfidoWebhook_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	check := createTestBackgroundCheck(driverID, BGCheckStatusInProgress)
	check.Provider = ProviderOnfido

	webhookPayload := WebhookPayload{
		EventType:  "check.completed",
		ExternalID: "onfido-ext-123",
		Status:     "complete",
		Data: map[string]interface{}{
			"result": "clear",
		},
		Timestamp: time.Now(),
	}

	mockRepo.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderOnfido, "onfido-ext-123").Return(check, nil)
	mockRepo.On("UpdateBackgroundCheckCompleted", mock.Anything, check.ID, BGCheckStatusPassed, mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("UpdateDriverApproval", mock.Anything, driverID, true, mock.Anything, mock.Anything).Return(nil)

	c, w := setupTestContext("POST", "/webhooks/onfido", webhookPayload)

	handler.HandleOnfidoWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_HandleOnfidoWebhook_InvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("POST", "/webhooks/onfido", nil)
	c.Request = httptest.NewRequest("POST", "/webhooks/onfido", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.HandleOnfidoWebhook(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_HandleOnfidoWebhook_FailedResult(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	check := createTestBackgroundCheck(driverID, BGCheckStatusInProgress)
	check.Provider = ProviderOnfido

	webhookPayload := WebhookPayload{
		EventType:  "check.completed",
		ExternalID: "onfido-ext-123",
		Status:     "complete",
		Data: map[string]interface{}{
			"result": "consider",
		},
		Timestamp: time.Now(),
	}

	mockRepo.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderOnfido, "onfido-ext-123").Return(check, nil)
	mockRepo.On("UpdateBackgroundCheckCompleted", mock.Anything, check.ID, BGCheckStatusFailed, mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("UpdateBackgroundCheckStatus", mock.Anything, check.ID, BGCheckStatusFailed, mock.Anything, mock.Anything).Return(nil)

	c, w := setupTestContext("POST", "/webhooks/onfido", webhookPayload)

	handler.HandleOnfidoWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_HandleOnfidoWebhook_InProgressStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	check := createTestBackgroundCheck(driverID, BGCheckStatusInProgress)
	check.Provider = ProviderOnfido

	webhookPayload := WebhookPayload{
		EventType:  "check.started",
		ExternalID: "onfido-ext-123",
		Status:     "in_progress",
		Data:       map[string]interface{}{},
		Timestamp:  time.Now(),
	}

	mockRepo.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderOnfido, "onfido-ext-123").Return(check, nil)
	mockRepo.On("UpdateBackgroundCheckCompleted", mock.Anything, check.ID, BGCheckStatusInProgress, mock.Anything, mock.Anything).Return(nil)

	c, w := setupTestContext("POST", "/webhooks/onfido", webhookPayload)

	handler.HandleOnfidoWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================================
// Handler Creation Tests
// ============================================================================

func TestNewHandler(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, nil)
	handler := NewHandler(service)

	require.NotNil(t, handler)
	require.NotNil(t, handler.service)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_GetSelfieStatus_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		status         SelfieVerificationStatus
		expectedStatus int
	}{
		{"valid verified selfie", SelfieStatusVerified, http.StatusOK},
		{"valid failed selfie", SelfieStatusFailed, http.StatusOK},
		{"valid pending selfie", SelfieStatusPending, http.StatusOK},
		{"valid expired selfie", SelfieStatusExpired, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			driverID := uuid.New()
			verificationID := uuid.New()
			verification := createTestSelfieVerification(driverID, tt.status)
			verification.ID = verificationID

			mockRepo.On("GetSelfieVerification", mock.Anything, verificationID).Return(verification, nil)

			c, w := setupTestContext("GET", "/api/v1/driver/verification/selfie/"+verificationID.String(), nil)
			c.Params = gin.Params{{Key: "id", Value: verificationID.String()}}
			setUserContext(c, driverID, models.RoleDriver)

			handler.GetSelfieStatus(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			response := parseResponse(w)
			assert.True(t, response["success"].(bool))
		})
	}
}

func TestHandler_BackgroundCheckStatus_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		status         BackgroundCheckStatus
		expectedStatus int
	}{
		{"pending status", BGCheckStatusPending, http.StatusOK},
		{"in_progress status", BGCheckStatusInProgress, http.StatusOK},
		{"passed status", BGCheckStatusPassed, http.StatusOK},
		{"failed status", BGCheckStatusFailed, http.StatusOK},
		{"expired status", BGCheckStatusExpired, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			driverID := uuid.New()
			check := createTestBackgroundCheck(driverID, tt.status)

			mockRepo.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(check, nil)

			c, w := setupTestContext("GET", "/api/v1/driver/verification/background", nil)
			setUserContext(c, driverID, models.RoleDriver)

			handler.GetBackgroundStatus(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			response := parseResponse(w)
			data := response["data"].(map[string]interface{})
			assert.Equal(t, string(tt.status), data["status"])
		})
	}
}

func TestHandler_WebhookProviders_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		provider BackgroundCheckProvider
		endpoint string
		handler  func(h *Handler) func(*gin.Context)
	}{
		{"checkr webhook", ProviderCheckr, "/webhooks/checkr", func(h *Handler) func(*gin.Context) { return h.HandleCheckrWebhook }},
		{"sterling webhook", ProviderSterling, "/webhooks/sterling", func(h *Handler) func(*gin.Context) { return h.HandleSterlingWebhook }},
		{"onfido webhook", ProviderOnfido, "/webhooks/onfido", func(h *Handler) func(*gin.Context) { return h.HandleOnfidoWebhook }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			driverID := uuid.New()
			check := createTestBackgroundCheck(driverID, BGCheckStatusInProgress)
			check.Provider = tt.provider

			webhookPayload := WebhookPayload{
				EventType:  "check.completed",
				ExternalID: "ext-123",
				Status:     "clear",
				Data: map[string]interface{}{
					"result": "clear",
				},
				Timestamp: time.Now(),
			}

			mockRepo.On("GetBackgroundCheckByExternalID", mock.Anything, tt.provider, "ext-123").Return(check, nil)
			mockRepo.On("UpdateBackgroundCheckCompleted", mock.Anything, check.ID, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockRepo.On("UpdateBackgroundCheckStatus", mock.Anything, check.ID, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
			mockRepo.On("UpdateDriverApproval", mock.Anything, driverID, true, mock.Anything, mock.Anything).Return(nil).Maybe()

			c, w := setupTestContext("POST", tt.endpoint, webhookPayload)

			handlerFunc := tt.handler(handler)
			handlerFunc(c)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestHandler_SubmitSelfie_WithLocation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	refURL := "https://example.com/reference.jpg"

	reqBody := map[string]interface{}{
		"driver_id": driverID.String(),
		"selfie_url": "https://example.com/selfie.jpg",
		"location": map[string]interface{}{
			"latitude":  37.7749,
			"longitude": -122.4194,
			"accuracy":  10.0,
		},
	}

	mockRepo.On("GetDriverReferencePhoto", mock.Anything, driverID).Return(&refURL, nil)
	mockRepo.On("CreateSelfieVerification", mock.Anything, mock.AnythingOfType("*verification.SelfieVerification")).Return(nil)
	mockRepo.On("UpdateSelfieVerificationResult", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/verification/selfie", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.SubmitSelfie(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_InitiateBackgroundCheck_WithSSN(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	driverID := uuid.New()

	reqBody := InitiateBackgroundCheckRequest{
		DriverID:      driverID,
		FirstName:     "John",
		LastName:      "Doe",
		DateOfBirth:   "1990-01-15",
		SSN:           "123-45-6789",
		Email:         "john.doe@example.com",
		Phone:         "+15551234567",
		StreetAddress: "123 Main St",
		City:          "San Francisco",
		State:         "CA",
		ZipCode:       "94102",
		LicenseNumber: "D1234567",
		LicenseState:  "CA",
		Provider:      ProviderMock,
	}

	mockRepo.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(nil, errors.New("not found"))
	mockRepo.On("CreateBackgroundCheck", mock.Anything, mock.AnythingOfType("*verification.BackgroundCheck")).Return(nil)
	mockRepo.On("UpdateBackgroundCheckStarted", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.Anything).Return(nil)
	mockRepo.On("UpdateBackgroundCheckCompleted", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	mockRepo.On("UpdateDriverApproval", mock.Anything, driverID, true, mock.Anything, mock.Anything).Return(nil).Maybe()

	c, w := setupTestContext("POST", "/api/v1/admin/verification/background", reqBody)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.InitiateBackgroundCheck(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_WebhookProcessingError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	check := createTestBackgroundCheck(driverID, BGCheckStatusInProgress)

	webhookPayload := WebhookPayload{
		EventType:  "report.completed",
		ExternalID: "ext-123",
		Status:     "clear",
		Data:       map[string]interface{}{},
		Timestamp:  time.Now(),
	}

	mockRepo.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderCheckr, "ext-123").Return(check, nil)
	mockRepo.On("UpdateBackgroundCheckCompleted", mock.Anything, check.ID, BGCheckStatusPassed, mock.Anything, mock.Anything).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/webhooks/checkr", webhookPayload)

	handler.HandleCheckrWebhook(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_InitiateBackgroundCheck_DefaultCheckType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	driverID := uuid.New()

	// Use mock provider, no check_type specified - should default to driver_standard
	reqBody := InitiateBackgroundCheckRequest{
		DriverID:      driverID,
		Provider:      ProviderMock,
		FirstName:     "John",
		LastName:      "Doe",
		DateOfBirth:   "1990-01-15",
		Email:         "john.doe@example.com",
		Phone:         "+15551234567",
		StreetAddress: "123 Main St",
		City:          "San Francisco",
		State:         "CA",
		ZipCode:       "94102",
		LicenseNumber: "D1234567",
		LicenseState:  "CA",
	}

	mockRepo.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(nil, errors.New("not found"))
	mockRepo.On("CreateBackgroundCheck", mock.Anything, mock.AnythingOfType("*verification.BackgroundCheck")).Return(nil)
	mockRepo.On("UpdateBackgroundCheckStarted", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.Anything).Return(nil)
	mockRepo.On("UpdateBackgroundCheckCompleted", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	mockRepo.On("UpdateDriverApproval", mock.Anything, driverID, true, mock.Anything, mock.Anything).Return(nil).Maybe()

	c, w := setupTestContext("POST", "/api/v1/admin/verification/background", reqBody)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.InitiateBackgroundCheck(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetBackgroundStatus_AppError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	mockRepo.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(nil, common.NewNotFoundError("no background check", nil))

	c, w := setupTestContext("GET", "/api/v1/driver/verification/background", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetBackgroundStatus(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_InitiateBackgroundCheck_AppError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	driverID := uuid.New()

	reqBody := InitiateBackgroundCheckRequest{
		DriverID:      driverID,
		FirstName:     "John",
		LastName:      "Doe",
		DateOfBirth:   "1990-01-15",
		Email:         "john.doe@example.com",
		Phone:         "+15551234567",
		StreetAddress: "123 Main St",
		City:          "San Francisco",
		State:         "CA",
		ZipCode:       "94102",
		LicenseNumber: "D1234567",
		LicenseState:  "CA",
		Provider:      ProviderMock,
	}

	mockRepo.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(nil, errors.New("not found"))
	mockRepo.On("CreateBackgroundCheck", mock.Anything, mock.AnythingOfType("*verification.BackgroundCheck")).Return(common.NewInternalServerError("database connection failed"))

	c, w := setupTestContext("POST", "/api/v1/admin/verification/background", reqBody)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.InitiateBackgroundCheck(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
