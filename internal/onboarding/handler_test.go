package onboarding

import (
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
// Mock Service Interface
// ============================================================================

// OnboardingServiceInterface defines the service methods used by the handler
type OnboardingServiceInterface interface {
	GetOnboardingProgress(ctx context.Context, driverID uuid.UUID) (*OnboardingProgress, error)
	GetDocumentRequirements(ctx context.Context, driverID uuid.UUID) ([]*DocumentRequirement, error)
	StartOnboarding(ctx context.Context, userID uuid.UUID) (*OnboardingProgress, error)
}

// MockOnboardingService is a mock implementation of the onboarding service
type MockOnboardingService struct {
	mock.Mock
}

func (m *MockOnboardingService) GetOnboardingProgress(ctx context.Context, driverID uuid.UUID) (*OnboardingProgress, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*OnboardingProgress), args.Error(1)
}

func (m *MockOnboardingService) GetDocumentRequirements(ctx context.Context, driverID uuid.UUID) ([]*DocumentRequirement, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DocumentRequirement), args.Error(1)
}

func (m *MockOnboardingService) StartOnboarding(ctx context.Context, userID uuid.UUID) (*OnboardingProgress, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*OnboardingProgress), args.Error(1)
}

// ============================================================================
// Mock Repository
// ============================================================================

// MockRepository is a mock implementation of RepositoryInterface for handler tests
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetDriver(ctx context.Context, driverID uuid.UUID) (*DriverInfo, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverInfo), args.Error(1)
}

func (m *MockRepository) GetDriverByUserID(ctx context.Context, userID uuid.UUID) (*DriverInfo, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverInfo), args.Error(1)
}

func (m *MockRepository) CreateDriver(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockRepository) GetBackgroundCheck(ctx context.Context, driverID uuid.UUID) (*BackgroundCheck, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BackgroundCheck), args.Error(1)
}

func (m *MockRepository) CreateBackgroundCheck(ctx context.Context, driverID uuid.UUID, provider string) (*BackgroundCheck, error) {
	args := m.Called(ctx, driverID, provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BackgroundCheck), args.Error(1)
}

func (m *MockRepository) UpdateBackgroundCheckStatus(ctx context.Context, checkID uuid.UUID, status string, notes *string) error {
	args := m.Called(ctx, checkID, status, notes)
	return args.Error(0)
}

func (m *MockRepository) ApproveDriver(ctx context.Context, driverID uuid.UUID, approvedBy uuid.UUID) error {
	args := m.Called(ctx, driverID, approvedBy)
	return args.Error(0)
}

func (m *MockRepository) RejectDriver(ctx context.Context, driverID uuid.UUID, rejectedBy uuid.UUID, reason string) error {
	args := m.Called(ctx, driverID, rejectedBy, reason)
	return args.Error(0)
}

func (m *MockRepository) GetOnboardingStats(ctx context.Context) (*OnboardingStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*OnboardingStats), args.Error(1)
}

// ============================================================================
// Testable Handler
// ============================================================================

// TestableHandler provides handler methods with mock service and repository injection
type TestableHandler struct {
	service *MockOnboardingService
	repo    *MockRepository
}

func NewTestableHandler(mockService *MockOnboardingService, mockRepo *MockRepository) *TestableHandler {
	return &TestableHandler{
		service: mockService,
		repo:    mockRepo,
	}
}

// GetProgress handles getting onboarding progress - testable version
func (h *TestableHandler) GetProgress(c *gin.Context) {
	driverID, err := h.getDriverID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "not a registered driver")
		return
	}

	progress, err := h.service.GetOnboardingProgress(c.Request.Context(), driverID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get onboarding progress")
		return
	}

	common.SuccessResponse(c, progress)
}

// StartOnboarding handles starting the onboarding process - testable version
func (h *TestableHandler) StartOnboarding(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	progress, err := h.service.StartOnboarding(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to start onboarding")
		return
	}

	common.SuccessResponse(c, progress)
}

// GetDocumentRequirements handles getting document requirements - testable version
func (h *TestableHandler) GetDocumentRequirements(c *gin.Context) {
	driverID, err := h.getDriverID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "not a registered driver")
		return
	}

	requirements, err := h.service.GetDocumentRequirements(c.Request.Context(), driverID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get document requirements")
		return
	}

	common.SuccessResponse(c, gin.H{
		"requirements": requirements,
	})
}

// GetOnboardingStats handles getting onboarding statistics - testable version
func (h *TestableHandler) GetOnboardingStats(c *gin.Context) {
	stats, err := h.repo.GetOnboardingStats(c.Request.Context())
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get onboarding stats")
		return
	}

	common.SuccessResponse(c, stats)
}

// getDriverID gets the driver ID from the authenticated user
func (h *TestableHandler) getDriverID(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, errors.New("user not authenticated")
	}

	driver, err := h.repo.GetDriverByUserID(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		return uuid.Nil, err
	}

	return driver.ID, nil
}

// ============================================================================
// Test Helper Functions
// ============================================================================

func setupTestContext(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, nil)
		req.Header.Set("Content-Type", "application/json")
		_ = bodyBytes // for POST requests if needed
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

func createTestDriverInfo(driverID, userID uuid.UUID, approved bool) *DriverInfo {
	licenseNum := "DL12345678"
	now := time.Now()
	return &DriverInfo{
		ID:            driverID,
		UserID:        userID,
		FirstName:     "John",
		LastName:      "Doe",
		PhoneNumber:   "+15551234567",
		Email:         "john.doe@example.com",
		LicenseNumber: &licenseNum,
		IsApproved:    approved,
		IsSuspended:   false,
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
}

func createTestOnboardingProgress(driverID, userID uuid.UUID, status OnboardingStatus) *OnboardingProgress {
	return &OnboardingProgress{
		DriverID:        driverID,
		UserID:          userID,
		Status:          status,
		CurrentStep:     "profile",
		Steps:           []*OnboardingStep{},
		CompletedSteps:  0,
		TotalSteps:      6,
		ProgressPercent: 0,
		CanDrive:        status == StatusApproved,
		Message:         "Please complete your profile to continue",
	}
}

func createTestDocumentRequirements() []*DocumentRequirement {
	docTypeID := uuid.New()
	return []*DocumentRequirement{
		{
			DocumentTypeID:   docTypeID,
			DocumentTypeCode: "drivers_license",
			Name:             "Driver's License",
			Required:         true,
			Status:           "not_submitted",
		},
		{
			DocumentTypeID:   uuid.New(),
			DocumentTypeCode: "vehicle_registration",
			Name:             "Vehicle Registration",
			Required:         true,
			Status:           "not_submitted",
		},
	}
}

func createTestOnboardingStats() *OnboardingStats {
	avgHours := 24.5
	return &OnboardingStats{
		PendingCount:     15,
		ApprovedCount:    100,
		RejectedCount:    10,
		NewThisWeek:      25,
		AvgApprovalHours: &avgHours,
	}
}

// ============================================================================
// GetProgress Handler Tests
// ============================================================================

func TestHandler_GetProgress_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	driver := createTestDriverInfo(driverID, userID, false)
	progress := createTestOnboardingProgress(driverID, userID, StatusProfileIncomplete)

	mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockService.On("GetOnboardingProgress", mock.Anything, driverID).Return(progress, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/onboarding/progress", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetProgress(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
	mockService.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetProgress_ApprovedDriver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	driver := createTestDriverInfo(driverID, userID, true)
	progress := createTestOnboardingProgress(driverID, userID, StatusApproved)
	progress.CanDrive = true
	progress.CompletedSteps = 6
	progress.ProgressPercent = 100

	mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockService.On("GetOnboardingProgress", mock.Anything, driverID).Return(progress, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/onboarding/progress", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetProgress(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, true, data["can_drive"])
	mockService.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetProgress_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	c, w := setupTestContext("GET", "/api/v1/driver/onboarding/progress", nil)
	// Don't set user context to simulate unauthorized

	handler.GetProgress(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_GetProgress_NotADriver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()

	mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(nil, errors.New("driver not found"))

	c, w := setupTestContext("GET", "/api/v1/driver/onboarding/progress", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetProgress(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "not a registered driver")
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetProgress_DriverNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	driver := createTestDriverInfo(driverID, userID, false)

	mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockService.On("GetOnboardingProgress", mock.Anything, driverID).Return(nil, common.NewNotFoundError("driver not found", nil))

	c, w := setupTestContext("GET", "/api/v1/driver/onboarding/progress", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetProgress(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetProgress_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	driver := createTestDriverInfo(driverID, userID, false)

	mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockService.On("GetOnboardingProgress", mock.Anything, driverID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/onboarding/progress", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetProgress(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "failed to get onboarding progress")
	mockService.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetProgress_InternalServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	driver := createTestDriverInfo(driverID, userID, false)

	mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockService.On("GetOnboardingProgress", mock.Anything, driverID).Return(nil, common.NewInternalServerError("internal error"))

	c, w := setupTestContext("GET", "/api/v1/driver/onboarding/progress", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetProgress(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// StartOnboarding Handler Tests
// ============================================================================

func TestHandler_StartOnboarding_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	progress := createTestOnboardingProgress(driverID, userID, StatusProfileIncomplete)

	mockService.On("StartOnboarding", mock.Anything, userID).Return(progress, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/onboarding/start", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.StartOnboarding(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
	mockService.AssertExpectations(t)
}

func TestHandler_StartOnboarding_ExistingDriver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	progress := createTestOnboardingProgress(driverID, userID, StatusDocumentsPending)
	progress.CompletedSteps = 2
	progress.ProgressPercent = 33

	mockService.On("StartOnboarding", mock.Anything, userID).Return(progress, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/onboarding/start", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.StartOnboarding(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["completed_steps"])
	mockService.AssertExpectations(t)
}

func TestHandler_StartOnboarding_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	c, w := setupTestContext("POST", "/api/v1/driver/onboarding/start", nil)
	// Don't set user context

	handler.StartOnboarding(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_StartOnboarding_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()

	mockService.On("StartOnboarding", mock.Anything, userID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/driver/onboarding/start", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.StartOnboarding(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "failed to start onboarding")
	mockService.AssertExpectations(t)
}

func TestHandler_StartOnboarding_AppError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()

	mockService.On("StartOnboarding", mock.Anything, userID).Return(nil, common.NewInternalServerError("failed to create driver record"))

	c, w := setupTestContext("POST", "/api/v1/driver/onboarding/start", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.StartOnboarding(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_StartOnboarding_ConflictError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()

	mockService.On("StartOnboarding", mock.Anything, userID).Return(nil, common.NewConflictError("driver already exists"))

	c, w := setupTestContext("POST", "/api/v1/driver/onboarding/start", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.StartOnboarding(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetDocumentRequirements Handler Tests
// ============================================================================

func TestHandler_GetDocumentRequirements_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	driver := createTestDriverInfo(driverID, userID, false)
	requirements := createTestDocumentRequirements()

	mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockService.On("GetDocumentRequirements", mock.Anything, driverID).Return(requirements, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/onboarding/documents", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetDocumentRequirements(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["requirements"])
	mockService.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetDocumentRequirements_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	driver := createTestDriverInfo(driverID, userID, false)

	mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockService.On("GetDocumentRequirements", mock.Anything, driverID).Return([]*DocumentRequirement{}, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/onboarding/documents", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetDocumentRequirements(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetDocumentRequirements_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	c, w := setupTestContext("GET", "/api/v1/driver/onboarding/documents", nil)
	// Don't set user context

	handler.GetDocumentRequirements(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_GetDocumentRequirements_NotADriver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()

	mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(nil, errors.New("driver not found"))

	c, w := setupTestContext("GET", "/api/v1/driver/onboarding/documents", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetDocumentRequirements(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "not a registered driver")
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetDocumentRequirements_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	driver := createTestDriverInfo(driverID, userID, false)

	mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockService.On("GetDocumentRequirements", mock.Anything, driverID).Return(nil, errors.New("service unavailable"))

	c, w := setupTestContext("GET", "/api/v1/driver/onboarding/documents", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetDocumentRequirements(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "failed to get document requirements")
	mockService.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetDocumentRequirements_AppError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	driver := createTestDriverInfo(driverID, userID, false)

	mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockService.On("GetDocumentRequirements", mock.Anything, driverID).Return(nil, common.NewServiceUnavailableError("document service unavailable"))

	c, w := setupTestContext("GET", "/api/v1/driver/onboarding/documents", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetDocumentRequirements(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	mockService.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetOnboardingStats Handler Tests
// ============================================================================

func TestHandler_GetOnboardingStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	stats := createTestOnboardingStats()

	mockRepo.On("GetOnboardingStats", mock.Anything).Return(stats, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/onboarding/stats", nil)
	setUserContext(c, uuid.New(), models.RoleAdmin)

	handler.GetOnboardingStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(15), data["pending_count"])
	assert.Equal(t, float64(100), data["approved_count"])
	assert.Equal(t, float64(10), data["rejected_count"])
	assert.Equal(t, float64(25), data["new_this_week"])
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetOnboardingStats_EmptyStats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	stats := &OnboardingStats{
		PendingCount:  0,
		ApprovedCount: 0,
		RejectedCount: 0,
		NewThisWeek:   0,
	}

	mockRepo.On("GetOnboardingStats", mock.Anything).Return(stats, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/onboarding/stats", nil)
	setUserContext(c, uuid.New(), models.RoleAdmin)

	handler.GetOnboardingStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["pending_count"])
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetOnboardingStats_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	mockRepo.On("GetOnboardingStats", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/onboarding/stats", nil)
	setUserContext(c, uuid.New(), models.RoleAdmin)

	handler.GetOnboardingStats(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "failed to get onboarding stats")
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetOnboardingStats_DatabaseTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	mockRepo.On("GetOnboardingStats", mock.Anything).Return(nil, errors.New("database timeout"))

	c, w := setupTestContext("GET", "/api/v1/admin/onboarding/stats", nil)
	setUserContext(c, uuid.New(), models.RoleAdmin)

	handler.GetOnboardingStats(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_GetProgress_StatusCases(t *testing.T) {
	tests := []struct {
		name           string
		status         OnboardingStatus
		canDrive       bool
		completedSteps int
	}{
		{
			name:           "not started",
			status:         StatusNotStarted,
			canDrive:       false,
			completedSteps: 0,
		},
		{
			name:           "profile incomplete",
			status:         StatusProfileIncomplete,
			canDrive:       false,
			completedSteps: 0,
		},
		{
			name:           "documents pending",
			status:         StatusDocumentsPending,
			canDrive:       false,
			completedSteps: 2,
		},
		{
			name:           "documents under review",
			status:         StatusDocumentsReview,
			canDrive:       false,
			completedSteps: 3,
		},
		{
			name:           "background check",
			status:         StatusBackgroundCheck,
			canDrive:       false,
			completedSteps: 4,
		},
		{
			name:           "pending approval",
			status:         StatusPendingApproval,
			canDrive:       false,
			completedSteps: 5,
		},
		{
			name:           "approved",
			status:         StatusApproved,
			canDrive:       true,
			completedSteps: 6,
		},
		{
			name:           "rejected",
			status:         StatusRejected,
			canDrive:       false,
			completedSteps: 4,
		},
		{
			name:           "suspended",
			status:         StatusSuspended,
			canDrive:       false,
			completedSteps: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(MockOnboardingService)
			mockRepo := new(MockRepository)
			handler := NewTestableHandler(mockService, mockRepo)

			userID := uuid.New()
			driverID := uuid.New()
			driver := createTestDriverInfo(driverID, userID, tt.status == StatusApproved)
			progress := createTestOnboardingProgress(driverID, userID, tt.status)
			progress.CanDrive = tt.canDrive
			progress.CompletedSteps = tt.completedSteps

			mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
			mockService.On("GetOnboardingProgress", mock.Anything, driverID).Return(progress, nil)

			c, w := setupTestContext("GET", "/api/v1/driver/onboarding/progress", nil)
			setUserContext(c, userID, models.RoleDriver)

			handler.GetProgress(c)

			assert.Equal(t, http.StatusOK, w.Code)
			response := parseResponse(w)
			data := response["data"].(map[string]interface{})
			assert.Equal(t, tt.canDrive, data["can_drive"])
			assert.Equal(t, float64(tt.completedSteps), data["completed_steps"])
			mockService.AssertExpectations(t)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestHandler_AuthorizationCases(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		hasUserContext bool
		expectedStatus int
	}{
		{
			name: "authorized driver - not registered",
			setupContext: func(c *gin.Context) {
				setUserContext(c, uuid.New(), models.RoleDriver)
			},
			hasUserContext: true,
			expectedStatus: http.StatusUnauthorized, // Driver not found in repo returns 401
		},
		{
			name:           "no user context",
			setupContext:   func(c *gin.Context) {},
			hasUserContext: false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(MockOnboardingService)
			mockRepo := new(MockRepository)
			handler := NewTestableHandler(mockService, mockRepo)

			// When user context is set, the handler will try to get driver by user ID
			if tt.hasUserContext {
				mockRepo.On("GetDriverByUserID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil, errors.New("not found"))
			}

			c, w := setupTestContext("GET", "/api/v1/driver/onboarding/progress", nil)
			tt.setupContext(c)

			handler.GetProgress(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_DocumentRequirements_StatusVariations(t *testing.T) {
	tests := []struct {
		name         string
		requirements []*DocumentRequirement
	}{
		{
			name: "all documents not submitted",
			requirements: []*DocumentRequirement{
				{DocumentTypeCode: "drivers_license", Status: "not_submitted", Required: true},
				{DocumentTypeCode: "vehicle_registration", Status: "not_submitted", Required: true},
			},
		},
		{
			name: "some documents pending",
			requirements: []*DocumentRequirement{
				{DocumentTypeCode: "drivers_license", Status: "pending", Required: true},
				{DocumentTypeCode: "vehicle_registration", Status: "not_submitted", Required: true},
			},
		},
		{
			name: "all documents approved",
			requirements: []*DocumentRequirement{
				{DocumentTypeCode: "drivers_license", Status: "approved", Required: true},
				{DocumentTypeCode: "vehicle_registration", Status: "approved", Required: true},
			},
		},
		{
			name: "some documents rejected",
			requirements: []*DocumentRequirement{
				{DocumentTypeCode: "drivers_license", Status: "rejected", Required: true},
				{DocumentTypeCode: "vehicle_registration", Status: "approved", Required: true},
			},
		},
		{
			name: "documents expired",
			requirements: []*DocumentRequirement{
				{DocumentTypeCode: "drivers_license", Status: "expired", Required: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(MockOnboardingService)
			mockRepo := new(MockRepository)
			handler := NewTestableHandler(mockService, mockRepo)

			userID := uuid.New()
			driverID := uuid.New()
			driver := createTestDriverInfo(driverID, userID, false)

			mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
			mockService.On("GetDocumentRequirements", mock.Anything, driverID).Return(tt.requirements, nil)

			c, w := setupTestContext("GET", "/api/v1/driver/onboarding/documents", nil)
			setUserContext(c, userID, models.RoleDriver)

			handler.GetDocumentRequirements(c)

			assert.Equal(t, http.StatusOK, w.Code)
			response := parseResponse(w)
			assert.True(t, response["success"].(bool))
			mockService.AssertExpectations(t)
			mockRepo.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Edge Cases and Error Handling Tests
// ============================================================================

func TestHandler_GetProgress_ConcurrentCalls(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	driver := createTestDriverInfo(driverID, userID, false)
	progress := createTestOnboardingProgress(driverID, userID, StatusProfileIncomplete)

	mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil).Times(5)
	mockService.On("GetOnboardingProgress", mock.Anything, driverID).Return(progress, nil).Times(5)

	for i := 0; i < 5; i++ {
		c, w := setupTestContext("GET", "/api/v1/driver/onboarding/progress", nil)
		setUserContext(c, userID, models.RoleDriver)

		handler.GetProgress(c)

		assert.Equal(t, http.StatusOK, w.Code)
	}

	mockService.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetProgress_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	driver := createTestDriverInfo(driverID, userID, false)
	progress := &OnboardingProgress{
		DriverID:        driverID,
		UserID:          userID,
		Status:          StatusDocumentsPending,
		CurrentStep:     "documents",
		Steps:           []*OnboardingStep{},
		CompletedSteps:  2,
		TotalSteps:      6,
		ProgressPercent: 33,
		CanDrive:        false,
		Message:         "Upload your required documents",
		NextAction: &NextAction{
			Type:        "upload_documents",
			Title:       "Upload Documents",
			Description: "Upload 2 missing document(s)",
			ActionURL:   "/driver/documents",
			Priority:    "high",
		},
	}

	mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockService.On("GetOnboardingProgress", mock.Anything, driverID).Return(progress, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/onboarding/progress", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetProgress(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	// Verify response structure
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["driver_id"])
	assert.NotNil(t, data["user_id"])
	assert.NotNil(t, data["status"])
	assert.NotNil(t, data["current_step"])
	assert.NotNil(t, data["completed_steps"])
	assert.NotNil(t, data["total_steps"])
	assert.NotNil(t, data["progress_percent"])
	assert.NotNil(t, data["can_drive"])
	assert.NotNil(t, data["message"])
	assert.NotNil(t, data["next_action"])

	nextAction := data["next_action"].(map[string]interface{})
	assert.Equal(t, "upload_documents", nextAction["type"])
	assert.Equal(t, "high", nextAction["priority"])

	mockService.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	driver := createTestDriverInfo(driverID, userID, false)

	mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockService.On("GetOnboardingProgress", mock.Anything, driverID).Return(nil, common.NewNotFoundError("driver not found", nil))

	c, w := setupTestContext("GET", "/api/v1/driver/onboarding/progress", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetProgress(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	response := parseResponse(w)

	// Verify error response structure
	assert.False(t, response["success"].(bool))
	assert.NotNil(t, response["error"])

	errorInfo := response["error"].(map[string]interface{})
	assert.NotNil(t, errorInfo["code"])
	assert.NotNil(t, errorInfo["message"])

	mockService.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// Role-Based Access Tests
// ============================================================================

func TestHandler_RoleAccess_Driver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	driver := createTestDriverInfo(driverID, userID, false)
	progress := createTestOnboardingProgress(driverID, userID, StatusProfileIncomplete)

	mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockService.On("GetOnboardingProgress", mock.Anything, driverID).Return(progress, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/onboarding/progress", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetProgress(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestHandler_RoleAccess_RiderAsDriver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	// Rider who is also registering as a driver
	driver := createTestDriverInfo(driverID, userID, false)
	progress := createTestOnboardingProgress(driverID, userID, StatusProfileIncomplete)

	mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockService.On("GetOnboardingProgress", mock.Anything, driverID).Return(progress, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/onboarding/progress", nil)
	setUserContext(c, userID, models.RoleRider) // Rider role but has driver record

	handler.GetProgress(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// Integration-style Handler Tests
// ============================================================================

func TestHandler_FullOnboardingFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()
	driverID := uuid.New()

	// Step 1: Start onboarding
	startProgress := createTestOnboardingProgress(driverID, userID, StatusProfileIncomplete)
	mockService.On("StartOnboarding", mock.Anything, userID).Return(startProgress, nil).Once()

	c1, w1 := setupTestContext("POST", "/api/v1/driver/onboarding/start", nil)
	setUserContext(c1, userID, models.RoleDriver)
	handler.StartOnboarding(c1)

	require.Equal(t, http.StatusOK, w1.Code)

	// Step 2: Get progress
	driver := createTestDriverInfo(driverID, userID, false)
	getProgress := createTestOnboardingProgress(driverID, userID, StatusDocumentsPending)
	getProgress.CompletedSteps = 2

	mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockService.On("GetOnboardingProgress", mock.Anything, driverID).Return(getProgress, nil).Once()

	c2, w2 := setupTestContext("GET", "/api/v1/driver/onboarding/progress", nil)
	setUserContext(c2, userID, models.RoleDriver)
	handler.GetProgress(c2)

	require.Equal(t, http.StatusOK, w2.Code)

	// Step 3: Get document requirements
	requirements := createTestDocumentRequirements()
	mockService.On("GetDocumentRequirements", mock.Anything, driverID).Return(requirements, nil).Once()

	c3, w3 := setupTestContext("GET", "/api/v1/driver/onboarding/documents", nil)
	setUserContext(c3, userID, models.RoleDriver)
	handler.GetDocumentRequirements(c3)

	require.Equal(t, http.StatusOK, w3.Code)

	mockService.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestHandler_AdminStatsAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	stats := createTestOnboardingStats()
	mockRepo.On("GetOnboardingStats", mock.Anything).Return(stats, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/onboarding/stats", nil)
	setUserContext(c, uuid.New(), models.RoleAdmin)

	handler.GetOnboardingStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(15), data["pending_count"])
	assert.Equal(t, float64(100), data["approved_count"])

	mockRepo.AssertExpectations(t)
}

// ============================================================================
// Nil/Empty Response Handling Tests
// ============================================================================

func TestHandler_GetProgress_NilNextAction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	driver := createTestDriverInfo(driverID, userID, true)
	progress := &OnboardingProgress{
		DriverID:        driverID,
		UserID:          userID,
		Status:          StatusApproved,
		CurrentStep:     "completed",
		Steps:           []*OnboardingStep{},
		CompletedSteps:  6,
		TotalSteps:      6,
		ProgressPercent: 100,
		CanDrive:        true,
		Message:         "You're approved! Start accepting rides",
		NextAction:      nil, // No next action for approved drivers
	}

	mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockService.On("GetOnboardingProgress", mock.Anything, driverID).Return(progress, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/onboarding/progress", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetProgress(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Nil(t, data["next_action"])
	mockService.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetDocumentRequirements_WithAllStatuses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockOnboardingService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()
	driverID := uuid.New()
	driver := createTestDriverInfo(driverID, userID, false)

	now := time.Now()
	expiry := now.AddDate(1, 0, 0)
	rejectionReason := "Document is blurry"

	requirements := []*DocumentRequirement{
		{
			DocumentTypeID:   uuid.New(),
			DocumentTypeCode: "drivers_license",
			Name:             "Driver's License",
			Required:         true,
			Status:           "approved",
			SubmittedAt:      &now,
			ReviewedAt:       &now,
			ExpiresAt:        &expiry,
		},
		{
			DocumentTypeID:   uuid.New(),
			DocumentTypeCode: "vehicle_registration",
			Name:             "Vehicle Registration",
			Required:         true,
			Status:           "rejected",
			SubmittedAt:      &now,
			ReviewedAt:       &now,
			RejectionReason:  &rejectionReason,
		},
		{
			DocumentTypeID:   uuid.New(),
			DocumentTypeCode: "insurance",
			Name:             "Insurance",
			Required:         true,
			Status:           "pending",
			SubmittedAt:      &now,
		},
		{
			DocumentTypeID:   uuid.New(),
			DocumentTypeCode: "background_check",
			Name:             "Background Check",
			Required:         false,
			Status:           "not_submitted",
		},
	}

	mockRepo.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockService.On("GetDocumentRequirements", mock.Anything, driverID).Return(requirements, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/onboarding/documents", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetDocumentRequirements(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	reqs := data["requirements"].([]interface{})
	assert.Len(t, reqs, 4)

	mockService.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}
