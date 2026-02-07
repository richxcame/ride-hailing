package tips

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// MockServiceInterface implements a mock service for handler testing
type MockServiceInterface struct {
	mock.Mock
}

func (m *MockServiceInterface) SendTip(ctx context.Context, riderID uuid.UUID, driverID uuid.UUID, req *SendTipRequest) (*Tip, error) {
	args := m.Called(ctx, riderID, driverID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Tip), args.Error(1)
}

func (m *MockServiceInterface) GetTipPresets(fareAmount float64) *TipPresetsResponse {
	args := m.Called(fareAmount)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*TipPresetsResponse)
}

func (m *MockServiceInterface) GetRiderTipHistory(ctx context.Context, riderID uuid.UUID, limit, offset int) (*RiderTipHistory, error) {
	args := m.Called(ctx, riderID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RiderTipHistory), args.Error(1)
}

func (m *MockServiceInterface) GetDriverTipSummary(ctx context.Context, driverID uuid.UUID, period string) (*DriverTipSummary, error) {
	args := m.Called(ctx, driverID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverTipSummary), args.Error(1)
}

func (m *MockServiceInterface) GetDriverTips(ctx context.Context, driverID uuid.UUID, limit, offset int) ([]Tip, int, error) {
	args := m.Called(ctx, driverID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]Tip), args.Int(1), args.Error(2)
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

// handlerWithMock wraps mock service for testing
type handlerWithMock struct {
	mockService *MockServiceInterface
}

func createTestHandlerWithMockService(mockService *MockServiceInterface) *handlerWithMock {
	return &handlerWithMock{mockService: mockService}
}

func (h *handlerWithMock) SendTip(c *gin.Context) {
	riderID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SendTipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get driver ID from query parameter
	driverIDStr := c.Query("driver_id")
	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "driver_id query parameter required")
		return
	}

	tip, err := h.mockService.SendTip(c.Request.Context(), riderID.(uuid.UUID), driverID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send tip")
		return
	}

	common.CreatedResponse(c, tip)
}

func (h *handlerWithMock) GetTipPresets(c *gin.Context) {
	fareStr := c.DefaultQuery("fare", "10.00")
	fare := 10.0
	if f, err := parseFloat(fareStr); err == nil && f > 0 {
		fare = f
	}

	presets := h.mockService.GetTipPresets(fare)
	common.SuccessResponse(c, presets)
}

func parseFloat(s string) (float64, error) {
	var f float64
	err := json.Unmarshal([]byte(s), &f)
	return f, err
}

func (h *handlerWithMock) GetMyTipHistory(c *gin.Context) {
	riderID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Default pagination
	limit := 20
	offset := 0

	history, err := h.mockService.GetRiderTipHistory(c.Request.Context(), riderID.(uuid.UUID), limit, offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get tip history")
		return
	}

	common.SuccessResponse(c, history)
}

func (h *handlerWithMock) GetDriverTipSummary(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	period := c.DefaultQuery("period", "this_month")

	summary, err := h.mockService.GetDriverTipSummary(c.Request.Context(), driverID.(uuid.UUID), period)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get tip summary")
		return
	}

	common.SuccessResponse(c, summary)
}

func (h *handlerWithMock) GetDriverTips(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Default pagination
	limit := 20
	offset := 0

	tips, total, err := h.mockService.GetDriverTips(c.Request.Context(), driverID.(uuid.UUID), limit, offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get tips")
		return
	}

	common.SuccessResponse(c, gin.H{
		"tips":  tips,
		"total": total,
	})
}

func createTestTip(riderID, driverID, rideID uuid.UUID) *Tip {
	msg := "Great ride!"
	return &Tip{
		ID:          uuid.New(),
		RideID:      rideID,
		RiderID:     riderID,
		DriverID:    driverID,
		Amount:      5.00,
		Currency:    "USD",
		Status:      TipStatusCompleted,
		Message:     &msg,
		IsAnonymous: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func createTestTipPresets(fareAmount float64) *TipPresetsResponse {
	return &TipPresetsResponse{
		Presets: []TipPreset{
			{Amount: fareAmount * 0.10, Label: "10%", IsPercent: true, IsDefault: false},
			{Amount: fareAmount * 0.15, Label: "15%", IsPercent: true, IsDefault: true},
			{Amount: fareAmount * 0.20, Label: "20%", IsPercent: true, IsDefault: false},
			{Amount: fareAmount * 0.25, Label: "25%", IsPercent: true, IsDefault: false},
		},
		MaxTip:   200.0,
		Currency: "USD",
	}
}

func createTestDriverTipSummary(period string) *DriverTipSummary {
	return &DriverTipSummary{
		TotalTips:       150.00,
		TipCount:        25,
		AverageTip:      6.00,
		HighestTip:      20.00,
		TipRate:         0.35,
		PeriodRideCount: 70,
		Currency:        "USD",
		Period:          period,
	}
}

// ============================================================================
// SendTip Handler Tests
// ============================================================================

func TestHandler_SendTip_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := SendTipRequest{
		RideID:      rideID,
		Amount:      5.00,
		Message:     strPtr("Great ride!"),
		IsAnonymous: false,
	}

	tip := createTestTip(riderID, driverID, rideID)

	mockService.On("SendTip", mock.Anything, riderID, driverID, mock.AnythingOfType("*tips.SendTipRequest")).
		Return(tip, nil)

	c, w := setupTestContext("POST", "/api/v1/tips?driver_id="+driverID.String(), reqBody)
	c.Request.URL.RawQuery = "driver_id=" + driverID.String()
	setUserContext(c, riderID, models.RoleRider)

	handler.SendTip(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_SendTip_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := SendTipRequest{
		RideID:  rideID,
		Amount:  5.00,
	}

	c, w := setupTestContext("POST", "/api/v1/tips?driver_id="+driverID.String(), reqBody)
	c.Request.URL.RawQuery = "driver_id=" + driverID.String()
	// Don't set user context

	handler.SendTip(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_SendTip_InvalidRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	driverID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/tips?driver_id="+driverID.String(), nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/tips?driver_id="+driverID.String(), bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.URL.RawQuery = "driver_id=" + driverID.String()
	setUserContext(c, riderID, models.RoleRider)

	handler.SendTip(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid request body")
}

func TestHandler_SendTip_MissingDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	rideID := uuid.New()

	reqBody := SendTipRequest{
		RideID:  rideID,
		Amount:  5.00,
	}

	c, w := setupTestContext("POST", "/api/v1/tips", reqBody)
	setUserContext(c, riderID, models.RoleRider)

	handler.SendTip(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "driver_id")
}

func TestHandler_SendTip_InvalidDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	rideID := uuid.New()

	reqBody := SendTipRequest{
		RideID:  rideID,
		Amount:  5.00,
	}

	c, w := setupTestContext("POST", "/api/v1/tips?driver_id=invalid-uuid", reqBody)
	c.Request.URL.RawQuery = "driver_id=invalid-uuid"
	setUserContext(c, riderID, models.RoleRider)

	handler.SendTip(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SendTip_AlreadyTipped(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := SendTipRequest{
		RideID:  rideID,
		Amount:  5.00,
	}

	mockService.On("SendTip", mock.Anything, riderID, driverID, mock.AnythingOfType("*tips.SendTipRequest")).
		Return(nil, common.NewConflictError("you have already tipped for this ride"))

	c, w := setupTestContext("POST", "/api/v1/tips?driver_id="+driverID.String(), reqBody)
	c.Request.URL.RawQuery = "driver_id=" + driverID.String()
	setUserContext(c, riderID, models.RoleRider)

	handler.SendTip(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SendTip_AmountTooLow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := SendTipRequest{
		RideID:  rideID,
		Amount:  0.50, // Below minimum
	}

	mockService.On("SendTip", mock.Anything, riderID, driverID, mock.AnythingOfType("*tips.SendTipRequest")).
		Return(nil, common.NewBadRequestError("minimum tip is 1.00", nil))

	c, w := setupTestContext("POST", "/api/v1/tips?driver_id="+driverID.String(), reqBody)
	c.Request.URL.RawQuery = "driver_id=" + driverID.String()
	setUserContext(c, riderID, models.RoleRider)

	handler.SendTip(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SendTip_AmountTooHigh(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := SendTipRequest{
		RideID:  rideID,
		Amount:  500.00, // Above maximum
	}

	mockService.On("SendTip", mock.Anything, riderID, driverID, mock.AnythingOfType("*tips.SendTipRequest")).
		Return(nil, common.NewBadRequestError("maximum tip is 200.00", nil))

	c, w := setupTestContext("POST", "/api/v1/tips?driver_id="+driverID.String(), reqBody)
	c.Request.URL.RawQuery = "driver_id=" + driverID.String()
	setUserContext(c, riderID, models.RoleRider)

	handler.SendTip(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SendTip_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := SendTipRequest{
		RideID:  rideID,
		Amount:  5.00,
	}

	mockService.On("SendTip", mock.Anything, riderID, driverID, mock.AnythingOfType("*tips.SendTipRequest")).
		Return(nil, errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/tips?driver_id="+driverID.String(), reqBody)
	c.Request.URL.RawQuery = "driver_id=" + driverID.String()
	setUserContext(c, riderID, models.RoleRider)

	handler.SendTip(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SendTip_WithMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	msg := "Excellent service, thank you!"

	reqBody := SendTipRequest{
		RideID:  rideID,
		Amount:  10.00,
		Message: &msg,
	}

	tip := createTestTip(riderID, driverID, rideID)
	tip.Message = &msg
	tip.Amount = 10.00

	mockService.On("SendTip", mock.Anything, riderID, driverID, mock.AnythingOfType("*tips.SendTipRequest")).
		Return(tip, nil)

	c, w := setupTestContext("POST", "/api/v1/tips?driver_id="+driverID.String(), reqBody)
	c.Request.URL.RawQuery = "driver_id=" + driverID.String()
	setUserContext(c, riderID, models.RoleRider)

	handler.SendTip(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SendTip_Anonymous(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := SendTipRequest{
		RideID:      rideID,
		Amount:      5.00,
		IsAnonymous: true,
	}

	tip := createTestTip(riderID, driverID, rideID)
	tip.IsAnonymous = true

	mockService.On("SendTip", mock.Anything, riderID, driverID, mock.AnythingOfType("*tips.SendTipRequest")).
		Return(tip, nil)

	c, w := setupTestContext("POST", "/api/v1/tips?driver_id="+driverID.String(), reqBody)
	c.Request.URL.RawQuery = "driver_id=" + driverID.String()
	setUserContext(c, riderID, models.RoleRider)

	handler.SendTip(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetTipPresets Handler Tests
// ============================================================================

func TestHandler_GetTipPresets_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	presets := createTestTipPresets(25.0)

	mockService.On("GetTipPresets", 25.0).Return(presets)

	c, w := setupTestContext("GET", "/api/v1/tips/presets?fare=25.0", nil)
	c.Request.URL.RawQuery = "fare=25.0"

	handler.GetTipPresets(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetTipPresets_DefaultFare(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	presets := createTestTipPresets(10.0)

	mockService.On("GetTipPresets", 10.0).Return(presets)

	c, w := setupTestContext("GET", "/api/v1/tips/presets", nil)

	handler.GetTipPresets(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetTipPresets_InvalidFare(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	presets := createTestTipPresets(10.0)

	// Invalid fare should default to 10.0
	mockService.On("GetTipPresets", 10.0).Return(presets)

	c, w := setupTestContext("GET", "/api/v1/tips/presets?fare=invalid", nil)
	c.Request.URL.RawQuery = "fare=invalid"

	handler.GetTipPresets(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetTipPresets_ZeroFare(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	presets := createTestTipPresets(10.0)

	// Zero fare should default to 10.0
	mockService.On("GetTipPresets", 10.0).Return(presets)

	c, w := setupTestContext("GET", "/api/v1/tips/presets?fare=0", nil)
	c.Request.URL.RawQuery = "fare=0"

	handler.GetTipPresets(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetTipPresets_NegativeFare(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	presets := createTestTipPresets(10.0)

	// Negative fare should default to 10.0
	mockService.On("GetTipPresets", 10.0).Return(presets)

	c, w := setupTestContext("GET", "/api/v1/tips/presets?fare=-5", nil)
	c.Request.URL.RawQuery = "fare=-5"

	handler.GetTipPresets(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetTipPresets_HighFare(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	presets := createTestTipPresets(100.0)

	mockService.On("GetTipPresets", 100.0).Return(presets)

	c, w := setupTestContext("GET", "/api/v1/tips/presets?fare=100", nil)
	c.Request.URL.RawQuery = "fare=100"

	handler.GetTipPresets(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetMyTipHistory Handler Tests
// ============================================================================

func TestHandler_GetMyTipHistory_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	tips := []Tip{*createTestTip(riderID, driverID, rideID)}

	mockService.On("GetRiderTipHistory", mock.Anything, riderID, 20, 0).
		Return(&RiderTipHistory{
			Tips:  tips,
			Total: 1,
		}, nil)

	c, w := setupTestContext("GET", "/api/v1/tips/history", nil)
	setUserContext(c, riderID, models.RoleRider)

	handler.GetMyTipHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetMyTipHistory_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	c, w := setupTestContext("GET", "/api/v1/tips/history", nil)
	// Don't set user context

	handler.GetMyTipHistory(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetMyTipHistory_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()

	mockService.On("GetRiderTipHistory", mock.Anything, riderID, 20, 0).
		Return(&RiderTipHistory{
			Tips:  []Tip{},
			Total: 0,
		}, nil)

	c, w := setupTestContext("GET", "/api/v1/tips/history", nil)
	setUserContext(c, riderID, models.RoleRider)

	handler.GetMyTipHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetMyTipHistory_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()

	mockService.On("GetRiderTipHistory", mock.Anything, riderID, 20, 0).
		Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/tips/history", nil)
	setUserContext(c, riderID, models.RoleRider)

	handler.GetMyTipHistory(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetMyTipHistory_MultipleTips(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	driverID1 := uuid.New()
	driverID2 := uuid.New()
	rideID1 := uuid.New()
	rideID2 := uuid.New()

	tips := []Tip{
		*createTestTip(riderID, driverID1, rideID1),
		*createTestTip(riderID, driverID2, rideID2),
	}

	mockService.On("GetRiderTipHistory", mock.Anything, riderID, 20, 0).
		Return(&RiderTipHistory{
			Tips:  tips,
			Total: 2,
		}, nil)

	c, w := setupTestContext("GET", "/api/v1/tips/history", nil)
	setUserContext(c, riderID, models.RoleRider)

	handler.GetMyTipHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetDriverTipSummary Handler Tests
// ============================================================================

func TestHandler_GetDriverTipSummary_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	driverID := uuid.New()
	summary := createTestDriverTipSummary("this_month")

	mockService.On("GetDriverTipSummary", mock.Anything, driverID, "this_month").
		Return(summary, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/tips/summary", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetDriverTipSummary(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetDriverTipSummary_WithPeriod(t *testing.T) {
	periods := []string{"today", "this_week", "this_month", "all_time"}

	for _, period := range periods {
		t.Run("period_"+period, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(MockServiceInterface)
			handler := createTestHandlerWithMockService(mockService)

			driverID := uuid.New()
			summary := createTestDriverTipSummary(period)

			mockService.On("GetDriverTipSummary", mock.Anything, driverID, period).
				Return(summary, nil)

			c, w := setupTestContext("GET", "/api/v1/driver/tips/summary?period="+period, nil)
			c.Request.URL.RawQuery = "period=" + period
			setUserContext(c, driverID, models.RoleDriver)

			handler.GetDriverTipSummary(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_GetDriverTipSummary_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	c, w := setupTestContext("GET", "/api/v1/driver/tips/summary", nil)
	// Don't set user context

	handler.GetDriverTipSummary(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetDriverTipSummary_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	driverID := uuid.New()

	mockService.On("GetDriverTipSummary", mock.Anything, driverID, "this_month").
		Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/tips/summary", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetDriverTipSummary(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetDriverTipSummary_EmptyStats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	driverID := uuid.New()
	summary := &DriverTipSummary{
		TotalTips:       0,
		TipCount:        0,
		AverageTip:      0,
		HighestTip:      0,
		TipRate:         0,
		PeriodRideCount: 0,
		Currency:        "USD",
		Period:          "this_month",
	}

	mockService.On("GetDriverTipSummary", mock.Anything, driverID, "this_month").
		Return(summary, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/tips/summary", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetDriverTipSummary(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetDriverTips Handler Tests
// ============================================================================

func TestHandler_GetDriverTips_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	driverID := uuid.New()
	riderID := uuid.New()
	rideID := uuid.New()

	tips := []Tip{*createTestTip(riderID, driverID, rideID)}

	mockService.On("GetDriverTips", mock.Anything, driverID, 20, 0).
		Return(tips, 1, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/tips", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetDriverTips(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetDriverTips_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	c, w := setupTestContext("GET", "/api/v1/driver/tips", nil)
	// Don't set user context

	handler.GetDriverTips(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetDriverTips_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	driverID := uuid.New()

	mockService.On("GetDriverTips", mock.Anything, driverID, 20, 0).
		Return([]Tip{}, 0, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/tips", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetDriverTips(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetDriverTips_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	driverID := uuid.New()

	mockService.On("GetDriverTips", mock.Anything, driverID, 20, 0).
		Return(nil, 0, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/tips", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetDriverTips(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetDriverTips_MultipleTips(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	driverID := uuid.New()
	riderID1 := uuid.New()
	riderID2 := uuid.New()
	rideID1 := uuid.New()
	rideID2 := uuid.New()

	tips := []Tip{
		*createTestTip(riderID1, driverID, rideID1),
		*createTestTip(riderID2, driverID, rideID2),
	}

	mockService.On("GetDriverTips", mock.Anything, driverID, 20, 0).
		Return(tips, 2, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/tips", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetDriverTips(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_SendTip_AmountValidation(t *testing.T) {
	tests := []struct {
		name           string
		amount         float64
		setupMock      func(mockService *MockServiceInterface, riderID, driverID uuid.UUID, req *SendTipRequest)
		expectedStatus int
	}{
		{
			name:   "valid tip amount",
			amount: 5.00,
			setupMock: func(mockService *MockServiceInterface, riderID, driverID uuid.UUID, req *SendTipRequest) {
				tip := &Tip{ID: uuid.New(), Amount: 5.00, Status: TipStatusCompleted}
				mockService.On("SendTip", mock.Anything, riderID, driverID, mock.AnythingOfType("*tips.SendTipRequest")).
					Return(tip, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:   "minimum tip amount",
			amount: 1.00,
			setupMock: func(mockService *MockServiceInterface, riderID, driverID uuid.UUID, req *SendTipRequest) {
				tip := &Tip{ID: uuid.New(), Amount: 1.00, Status: TipStatusCompleted}
				mockService.On("SendTip", mock.Anything, riderID, driverID, mock.AnythingOfType("*tips.SendTipRequest")).
					Return(tip, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:   "maximum tip amount",
			amount: 200.00,
			setupMock: func(mockService *MockServiceInterface, riderID, driverID uuid.UUID, req *SendTipRequest) {
				tip := &Tip{ID: uuid.New(), Amount: 200.00, Status: TipStatusCompleted}
				mockService.On("SendTip", mock.Anything, riderID, driverID, mock.AnythingOfType("*tips.SendTipRequest")).
					Return(tip, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:   "below minimum tip",
			amount: 0.50,
			setupMock: func(mockService *MockServiceInterface, riderID, driverID uuid.UUID, req *SendTipRequest) {
				mockService.On("SendTip", mock.Anything, riderID, driverID, mock.AnythingOfType("*tips.SendTipRequest")).
					Return(nil, common.NewBadRequestError("minimum tip is 1.00", nil))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "above maximum tip",
			amount: 250.00,
			setupMock: func(mockService *MockServiceInterface, riderID, driverID uuid.UUID, req *SendTipRequest) {
				mockService.On("SendTip", mock.Anything, riderID, driverID, mock.AnythingOfType("*tips.SendTipRequest")).
					Return(nil, common.NewBadRequestError("maximum tip is 200.00", nil))
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(MockServiceInterface)
			handler := createTestHandlerWithMockService(mockService)

			riderID := uuid.New()
			driverID := uuid.New()
			rideID := uuid.New()

			reqBody := SendTipRequest{
				RideID:  rideID,
				Amount:  tt.amount,
			}

			tt.setupMock(mockService, riderID, driverID, &reqBody)

			c, w := setupTestContext("POST", "/api/v1/tips?driver_id="+driverID.String(), reqBody)
			c.Request.URL.RawQuery = "driver_id=" + driverID.String()
			setUserContext(c, riderID, models.RoleRider)

			handler.SendTip(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_TipPresets_FareRanges(t *testing.T) {
	tests := []struct {
		name         string
		fare         float64
		expectedFare float64
	}{
		{name: "low fare", fare: 5.0, expectedFare: 5.0},
		{name: "medium fare", fare: 25.0, expectedFare: 25.0},
		{name: "high fare", fare: 100.0, expectedFare: 100.0},
		{name: "very high fare", fare: 500.0, expectedFare: 500.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(MockServiceInterface)
			handler := createTestHandlerWithMockService(mockService)

			presets := createTestTipPresets(tt.expectedFare)
			mockService.On("GetTipPresets", tt.expectedFare).Return(presets)

			c, w := setupTestContext("GET", "/api/v1/tips/presets", nil)
			c.Request.URL.RawQuery = "fare=" + formatFloat(tt.fare)

			handler.GetTipPresets(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.1f", f)
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_SendTip_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := SendTipRequest{
		RideID:  rideID,
		Amount:  5.00,
	}

	tip := createTestTip(riderID, driverID, rideID)

	mockService.On("SendTip", mock.Anything, riderID, driverID, mock.AnythingOfType("*tips.SendTipRequest")).
		Return(tip, nil)

	c, w := setupTestContext("POST", "/api/v1/tips?driver_id="+driverID.String(), reqBody)
	c.Request.URL.RawQuery = "driver_id=" + driverID.String()
	setUserContext(c, riderID, models.RoleRider)

	handler.SendTip(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)

	// Verify response structure
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetDriverTips_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	driverID := uuid.New()
	riderID := uuid.New()
	rideID := uuid.New()

	tips := []Tip{*createTestTip(riderID, driverID, rideID)}

	mockService.On("GetDriverTips", mock.Anything, driverID, 20, 0).
		Return(tips, 1, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/tips", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetDriverTips(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	// Verify response structure
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["tips"])
	assert.NotNil(t, data["total"])
	mockService.AssertExpectations(t)
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()

	mockService.On("GetRiderTipHistory", mock.Anything, riderID, 20, 0).
		Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/tips/history", nil)
	setUserContext(c, riderID, models.RoleRider)

	handler.GetMyTipHistory(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)

	// Verify error response structure
	assert.False(t, response["success"].(bool))
	assert.NotNil(t, response["error"])

	errorInfo := response["error"].(map[string]interface{})
	assert.NotNil(t, errorInfo["code"])
	assert.NotNil(t, errorInfo["message"])
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestHandler_SendTip_ZeroAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	// Zero amount will fail binding validation (amount is required) before service is called
	reqBody := SendTipRequest{
		RideID: rideID,
		Amount: 0,
	}

	c, w := setupTestContext("POST", "/api/v1/tips?driver_id="+driverID.String(), reqBody)
	c.Request.URL.RawQuery = "driver_id=" + driverID.String()
	setUserContext(c, riderID, models.RoleRider)

	handler.SendTip(c)

	// Returns 400 due to binding validation failing (Amount is required and 0 is considered empty)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SendTip_NegativeAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := SendTipRequest{
		RideID:  rideID,
		Amount:  -5.00,
	}

	mockService.On("SendTip", mock.Anything, riderID, driverID, mock.AnythingOfType("*tips.SendTipRequest")).
		Return(nil, common.NewBadRequestError("minimum tip is 1.00", nil))

	c, w := setupTestContext("POST", "/api/v1/tips?driver_id="+driverID.String(), reqBody)
	c.Request.URL.RawQuery = "driver_id=" + driverID.String()
	setUserContext(c, riderID, models.RoleRider)

	handler.SendTip(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetTipPresets_ResponseContainsAllFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	presets := createTestTipPresets(25.0)

	mockService.On("GetTipPresets", 25.0).Return(presets)

	c, w := setupTestContext("GET", "/api/v1/tips/presets?fare=25.0", nil)
	c.Request.URL.RawQuery = "fare=25.0"

	handler.GetTipPresets(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["presets"])
	assert.NotNil(t, data["max_tip"])
	assert.NotNil(t, data["currency"])
	mockService.AssertExpectations(t)
}

// ============================================================================
// Helper Functions
// ============================================================================

func strPtr(s string) *string {
	return &s
}
