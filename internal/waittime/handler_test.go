package waittime

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
)

// ============================================================================
// Mock Implementations
// ============================================================================

// MockRepository implements RepositoryInterface for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetActiveConfig(ctx context.Context) (*WaitTimeConfig, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*WaitTimeConfig), args.Error(1)
}

func (m *MockRepository) GetAllConfigs(ctx context.Context) ([]WaitTimeConfig, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]WaitTimeConfig), args.Error(1)
}

func (m *MockRepository) CreateConfig(ctx context.Context, config *WaitTimeConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockRepository) CreateRecord(ctx context.Context, rec *WaitTimeRecord) error {
	args := m.Called(ctx, rec)
	return args.Error(0)
}

func (m *MockRepository) GetActiveWaitByRide(ctx context.Context, rideID uuid.UUID) (*WaitTimeRecord, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*WaitTimeRecord), args.Error(1)
}

func (m *MockRepository) CompleteWait(ctx context.Context, recordID uuid.UUID, totalWaitMin, chargeableMin, totalCharge float64, wasCapped bool) error {
	args := m.Called(ctx, recordID, totalWaitMin, chargeableMin, totalCharge, wasCapped)
	return args.Error(0)
}

func (m *MockRepository) WaiveCharge(ctx context.Context, recordID uuid.UUID) error {
	args := m.Called(ctx, recordID)
	return args.Error(0)
}

func (m *MockRepository) GetRecordsByRide(ctx context.Context, rideID uuid.UUID) ([]WaitTimeRecord, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]WaitTimeRecord), args.Error(1)
}

func (m *MockRepository) SaveNotification(ctx context.Context, n *WaitTimeNotification) error {
	args := m.Called(ctx, n)
	return args.Error(0)
}

// ============================================================================
// Helper Functions
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
	service := NewService(mockRepo)
	return NewHandler(service)
}

func createTestConfig() *WaitTimeConfig {
	return &WaitTimeConfig{
		ID:              uuid.New(),
		Name:            "Standard",
		FreeWaitMinutes: 3,
		ChargePerMinute: 0.25,
		MaxWaitMinutes:  15,
		MaxWaitCharge:   10.0,
		AppliesTo:       "pickup",
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func createTestWaitRecord(driverID, rideID uuid.UUID) *WaitTimeRecord {
	config := createTestConfig()
	now := time.Now()
	return &WaitTimeRecord{
		ID:              uuid.New(),
		RideID:          rideID,
		DriverID:        driverID,
		ConfigID:        config.ID,
		WaitType:        "pickup",
		ArrivedAt:       now.Add(-5 * time.Minute),
		FreeMinutes:     config.FreeWaitMinutes,
		ChargePerMinute: config.ChargePerMinute,
		Status:          "waiting",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// ============================================================================
// StartWait Handler Tests
// ============================================================================

func TestHandler_StartWait_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	rideID := uuid.New()
	config := createTestConfig()

	reqBody := StartWaitRequest{
		RideID:   rideID,
		WaitType: "pickup",
	}

	mockRepo.On("GetActiveWaitByRide", mock.Anything, rideID).Return(nil, nil)
	mockRepo.On("GetActiveConfig", mock.Anything).Return(config, nil)
	mockRepo.On("CreateRecord", mock.Anything, mock.AnythingOfType("*waittime.WaitTimeRecord")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/wait/start", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.StartWait(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_StartWait_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	rideID := uuid.New()
	reqBody := StartWaitRequest{
		RideID:   rideID,
		WaitType: "pickup",
	}

	c, w := setupTestContext("POST", "/api/v1/driver/wait/start", reqBody)
	// Don't set user context

	handler.StartWait(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_StartWait_InvalidRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/wait/start", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/driver/wait/start", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, driverID, models.RoleDriver)

	handler.StartWait(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_StartWait_AlreadyActive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	rideID := uuid.New()
	existingRecord := createTestWaitRecord(driverID, rideID)

	reqBody := StartWaitRequest{
		RideID:   rideID,
		WaitType: "pickup",
	}

	mockRepo.On("GetActiveWaitByRide", mock.Anything, rideID).Return(existingRecord, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/wait/start", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.StartWait(c)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_StartWait_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	rideID := uuid.New()
	config := createTestConfig()

	reqBody := StartWaitRequest{
		RideID:   rideID,
		WaitType: "pickup",
	}

	mockRepo.On("GetActiveWaitByRide", mock.Anything, rideID).Return(nil, nil)
	mockRepo.On("GetActiveConfig", mock.Anything).Return(config, nil)
	mockRepo.On("CreateRecord", mock.Anything, mock.AnythingOfType("*waittime.WaitTimeRecord")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/driver/wait/start", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.StartWait(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_StartWait_NoActiveConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := StartWaitRequest{
		RideID:   rideID,
		WaitType: "pickup",
	}

	// No active config - should use defaults
	mockRepo.On("GetActiveWaitByRide", mock.Anything, rideID).Return(nil, nil)
	mockRepo.On("GetActiveConfig", mock.Anything).Return(nil, errors.New("no config"))
	mockRepo.On("CreateRecord", mock.Anything, mock.AnythingOfType("*waittime.WaitTimeRecord")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/wait/start", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.StartWait(c)

	assert.Equal(t, http.StatusCreated, w.Code)
}

// ============================================================================
// StopWait Handler Tests
// ============================================================================

func TestHandler_StopWait_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	rideID := uuid.New()
	record := createTestWaitRecord(driverID, rideID)

	reqBody := StopWaitRequest{
		RideID: rideID,
	}

	mockRepo.On("GetActiveWaitByRide", mock.Anything, rideID).Return(record, nil)
	mockRepo.On("GetActiveConfig", mock.Anything).Return(createTestConfig(), nil)
	mockRepo.On("CompleteWait", mock.Anything, record.ID, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("bool")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/wait/stop", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.StopWait(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_StopWait_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	rideID := uuid.New()
	reqBody := StopWaitRequest{
		RideID: rideID,
	}

	c, w := setupTestContext("POST", "/api/v1/driver/wait/stop", reqBody)
	// Don't set user context

	handler.StopWait(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_StopWait_InvalidRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/wait/stop", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/driver/wait/stop", bytes.NewReader([]byte("invalid")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, driverID, models.RoleDriver)

	handler.StopWait(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_StopWait_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := StopWaitRequest{
		RideID: rideID,
	}

	mockRepo.On("GetActiveWaitByRide", mock.Anything, rideID).Return(nil, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/wait/stop", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.StopWait(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_StopWait_NotYourRide(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	otherDriverID := uuid.New()
	rideID := uuid.New()
	record := createTestWaitRecord(otherDriverID, rideID)

	reqBody := StopWaitRequest{
		RideID: rideID,
	}

	mockRepo.On("GetActiveWaitByRide", mock.Anything, rideID).Return(record, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/wait/stop", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.StopWait(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_StopWait_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	rideID := uuid.New()
	record := createTestWaitRecord(driverID, rideID)

	reqBody := StopWaitRequest{
		RideID: rideID,
	}

	mockRepo.On("GetActiveWaitByRide", mock.Anything, rideID).Return(record, nil)
	mockRepo.On("GetActiveConfig", mock.Anything).Return(createTestConfig(), nil)
	mockRepo.On("CompleteWait", mock.Anything, record.ID, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("bool")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/driver/wait/stop", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.StopWait(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetWaitTimeSummary Handler Tests
// ============================================================================

func TestHandler_GetWaitTimeSummary_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	rideID := uuid.New()
	records := []WaitTimeRecord{
		{
			ID:               uuid.New(),
			RideID:           rideID,
			TotalWaitMinutes: 5.0,
			ChargeableMinutes: 2.0,
			TotalCharge:      0.50,
			Status:           "completed",
		},
	}

	mockRepo.On("GetRecordsByRide", mock.Anything, rideID).Return(records, nil)

	c, w := setupTestContext("GET", "/api/v1/rides/"+rideID.String()+"/wait-time", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}

	handler.GetWaitTimeSummary(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetWaitTimeSummary_InvalidRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/rides/invalid-uuid/wait-time", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.GetWaitTimeSummary(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetWaitTimeSummary_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	rideID := uuid.New()

	mockRepo.On("GetRecordsByRide", mock.Anything, rideID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/rides/"+rideID.String()+"/wait-time", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}

	handler.GetWaitTimeSummary(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_GetWaitTimeSummary_EmptyRecords(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	rideID := uuid.New()

	mockRepo.On("GetRecordsByRide", mock.Anything, rideID).Return(nil, nil)

	c, w := setupTestContext("GET", "/api/v1/rides/"+rideID.String()+"/wait-time", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}

	handler.GetWaitTimeSummary(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================================
// GetCurrentWaitStatus Handler Tests
// ============================================================================

func TestHandler_GetCurrentWaitStatus_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	rideID := uuid.New()
	record := createTestWaitRecord(driverID, rideID)

	mockRepo.On("GetActiveWaitByRide", mock.Anything, rideID).Return(record, nil)
	mockRepo.On("GetActiveConfig", mock.Anything).Return(createTestConfig(), nil)

	c, w := setupTestContext("GET", "/api/v1/rides/"+rideID.String()+"/wait-time/current", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}

	handler.GetCurrentWaitStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetCurrentWaitStatus_InvalidRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/rides/invalid-uuid/wait-time/current", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.GetCurrentWaitStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetCurrentWaitStatus_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	rideID := uuid.New()

	mockRepo.On("GetActiveWaitByRide", mock.Anything, rideID).Return(nil, nil)

	c, w := setupTestContext("GET", "/api/v1/rides/"+rideID.String()+"/wait-time/current", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}

	handler.GetCurrentWaitStatus(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetCurrentWaitStatus_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	rideID := uuid.New()

	mockRepo.On("GetActiveWaitByRide", mock.Anything, rideID).Return(nil, common.NewNotFoundError("not found", nil))

	c, w := setupTestContext("GET", "/api/v1/rides/"+rideID.String()+"/wait-time/current", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}

	handler.GetCurrentWaitStatus(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ============================================================================
// CreateConfig Handler Tests
// ============================================================================

func TestHandler_CreateConfig_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := CreateConfigRequest{
		Name:            "Premium",
		FreeWaitMinutes: 5,
		ChargePerMinute: 0.50,
		MaxWaitMinutes:  20,
		MaxWaitCharge:   15.0,
		AppliesTo:       "pickup",
	}

	mockRepo.On("CreateConfig", mock.Anything, mock.AnythingOfType("*waittime.WaitTimeConfig")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/wait-time/configs", reqBody)

	handler.CreateConfig(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_CreateConfig_InvalidRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("POST", "/api/v1/admin/wait-time/configs", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/wait-time/configs", bytes.NewReader([]byte("invalid")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateConfig(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateConfig_MissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	// Missing required fields
	reqBody := map[string]interface{}{
		"name": "Premium",
	}

	c, w := setupTestContext("POST", "/api/v1/admin/wait-time/configs", reqBody)

	handler.CreateConfig(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateConfig_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := CreateConfigRequest{
		Name:            "Premium",
		FreeWaitMinutes: 5,
		ChargePerMinute: 0.50,
		MaxWaitMinutes:  20,
		MaxWaitCharge:   15.0,
		AppliesTo:       "pickup",
	}

	mockRepo.On("CreateConfig", mock.Anything, mock.AnythingOfType("*waittime.WaitTimeConfig")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/admin/wait-time/configs", reqBody)

	handler.CreateConfig(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// ListConfigs Handler Tests
// ============================================================================

func TestHandler_ListConfigs_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	configs := []WaitTimeConfig{*createTestConfig()}
	mockRepo.On("GetAllConfigs", mock.Anything).Return(configs, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/wait-time/configs", nil)

	handler.ListConfigs(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_ListConfigs_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	mockRepo.On("GetAllConfigs", mock.Anything).Return(nil, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/wait-time/configs", nil)

	handler.ListConfigs(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ListConfigs_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	mockRepo.On("GetAllConfigs", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/wait-time/configs", nil)

	handler.ListConfigs(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// WaiveCharge Handler Tests
// ============================================================================

func TestHandler_WaiveCharge_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	recordID := uuid.New()
	reqBody := WaiveChargeRequest{
		RecordID: recordID,
		Reason:   "Customer complaint",
	}

	mockRepo.On("WaiveCharge", mock.Anything, recordID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/wait-time/waive", reqBody)

	handler.WaiveCharge(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_WaiveCharge_InvalidRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("POST", "/api/v1/admin/wait-time/waive", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/wait-time/waive", bytes.NewReader([]byte("invalid")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.WaiveCharge(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_WaiveCharge_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	recordID := uuid.New()
	reqBody := WaiveChargeRequest{
		RecordID: recordID,
		Reason:   "Customer complaint",
	}

	mockRepo.On("WaiveCharge", mock.Anything, recordID).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/admin/wait-time/waive", reqBody)

	handler.WaiveCharge(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_StartWait_ValidationCases(t *testing.T) {
	tests := []struct {
		name           string
		reqBody        interface{}
		setUserContext bool
		expectedStatus int
	}{
		{
			name: "valid request",
			reqBody: StartWaitRequest{
				RideID:   uuid.New(),
				WaitType: "pickup",
			},
			setUserContext: true,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "empty body",
			reqBody:        nil,
			setUserContext: true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing ride_id",
			reqBody: map[string]interface{}{
				"wait_type": "pickup",
			},
			setUserContext: true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "unauthorized",
			reqBody: StartWaitRequest{
				RideID:   uuid.New(),
				WaitType: "pickup",
			},
			setUserContext: false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			if tt.expectedStatus == http.StatusCreated {
				mockRepo.On("GetActiveWaitByRide", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil, nil)
				mockRepo.On("GetActiveConfig", mock.Anything).Return(createTestConfig(), nil)
				mockRepo.On("CreateRecord", mock.Anything, mock.AnythingOfType("*waittime.WaitTimeRecord")).Return(nil)
			}

			c, w := setupTestContext("POST", "/api/v1/driver/wait/start", tt.reqBody)
			if tt.reqBody == nil {
				c.Request = httptest.NewRequest("POST", "/api/v1/driver/wait/start", nil)
				c.Request.Header.Set("Content-Type", "application/json")
			}
			if tt.setUserContext {
				setUserContext(c, uuid.New(), models.RoleDriver)
			}

			handler.StartWait(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_CreateConfig_ValidationCases(t *testing.T) {
	tests := []struct {
		name           string
		reqBody        interface{}
		expectedStatus int
	}{
		{
			name: "valid config",
			reqBody: CreateConfigRequest{
				Name:            "Test",
				FreeWaitMinutes: 3,
				ChargePerMinute: 0.25,
				MaxWaitMinutes:  15,
				MaxWaitCharge:   10.0,
				AppliesTo:       "pickup",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "empty body",
			reqBody:        nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing name",
			reqBody: map[string]interface{}{
				"free_wait_minutes": 3,
				"charge_per_minute": 0.25,
				"max_wait_minutes":  15,
				"max_wait_charge":   10.0,
				"applies_to":        "pickup",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			if tt.expectedStatus == http.StatusCreated {
				mockRepo.On("CreateConfig", mock.Anything, mock.AnythingOfType("*waittime.WaitTimeConfig")).Return(nil)
			}

			c, w := setupTestContext("POST", "/api/v1/admin/wait-time/configs", tt.reqBody)
			if tt.reqBody == nil {
				c.Request = httptest.NewRequest("POST", "/api/v1/admin/wait-time/configs", nil)
				c.Request.Header.Set("Content-Type", "application/json")
			}

			handler.CreateConfig(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_StartWait_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	rideID := uuid.New()
	config := createTestConfig()

	reqBody := StartWaitRequest{
		RideID:   rideID,
		WaitType: "pickup",
	}

	mockRepo.On("GetActiveWaitByRide", mock.Anything, rideID).Return(nil, nil)
	mockRepo.On("GetActiveConfig", mock.Anything).Return(config, nil)
	mockRepo.On("CreateRecord", mock.Anything, mock.AnythingOfType("*waittime.WaitTimeRecord")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/wait/start", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.StartWait(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)

	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
}

func TestHandler_GetWaitTimeSummary_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	rideID := uuid.New()
	records := []WaitTimeRecord{
		{
			ID:               uuid.New(),
			RideID:           rideID,
			TotalWaitMinutes: 5.0,
			ChargeableMinutes: 2.0,
			TotalCharge:      0.50,
			Status:           "completed",
		},
	}

	mockRepo.On("GetRecordsByRide", mock.Anything, rideID).Return(records, nil)

	c, w := setupTestContext("GET", "/api/v1/rides/"+rideID.String()+"/wait-time", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}

	handler.GetWaitTimeSummary(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["ride_id"])
	assert.NotNil(t, data["records"])
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	mockRepo.On("GetAllConfigs", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/wait-time/configs", nil)

	handler.ListConfigs(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)

	assert.False(t, response["success"].(bool))
	assert.NotNil(t, response["error"])

	errorInfo := response["error"].(map[string]interface{})
	assert.NotNil(t, errorInfo["code"])
	assert.NotNil(t, errorInfo["message"])
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestHandler_GetWaitTimeSummary_EmptyUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/rides//wait-time", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.GetWaitTimeSummary(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetCurrentWaitStatus_EmptyUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/rides//wait-time/current", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.GetCurrentWaitStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_StopWait_RepoError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := StopWaitRequest{
		RideID: rideID,
	}

	mockRepo.On("GetActiveWaitByRide", mock.Anything, rideID).Return(nil, errors.New("db error"))

	c, w := setupTestContext("POST", "/api/v1/driver/wait/stop", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.StopWait(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_StartWait_RepoError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := StartWaitRequest{
		RideID:   rideID,
		WaitType: "pickup",
	}

	mockRepo.On("GetActiveWaitByRide", mock.Anything, rideID).Return(nil, errors.New("db error"))

	c, w := setupTestContext("POST", "/api/v1/driver/wait/start", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.StartWait(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
