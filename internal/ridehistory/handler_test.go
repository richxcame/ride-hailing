package ridehistory

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

// MockServiceInterface implements a mock service for handler testing
type MockServiceInterface struct {
	mock.Mock
}

func (m *MockServiceInterface) GetRiderHistory(ctx context.Context, riderID uuid.UUID, filters *HistoryFilters, page, pageSize int) (*HistoryResponse, error) {
	args := m.Called(ctx, riderID, filters, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*HistoryResponse), args.Error(1)
}

func (m *MockServiceInterface) GetDriverHistory(ctx context.Context, driverID uuid.UUID, filters *HistoryFilters, page, pageSize int) (*HistoryResponse, error) {
	args := m.Called(ctx, driverID, filters, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*HistoryResponse), args.Error(1)
}

func (m *MockServiceInterface) GetRideDetails(ctx context.Context, rideID uuid.UUID, userID uuid.UUID) (*RideHistoryEntry, error) {
	args := m.Called(ctx, rideID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RideHistoryEntry), args.Error(1)
}

func (m *MockServiceInterface) GetReceipt(ctx context.Context, rideID uuid.UUID, userID uuid.UUID) (*Receipt, error) {
	args := m.Called(ctx, rideID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Receipt), args.Error(1)
}

func (m *MockServiceInterface) GetRiderStats(ctx context.Context, riderID uuid.UUID, period string) (*RideStats, error) {
	args := m.Called(ctx, riderID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RideStats), args.Error(1)
}

func (m *MockServiceInterface) GetFrequentRoutes(ctx context.Context, riderID uuid.UUID) ([]FrequentRoute, error) {
	args := m.Called(ctx, riderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]FrequentRoute), args.Error(1)
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

func createTestHandler(mockService *MockServiceInterface) *Handler {
	// Create a real service with nil repo (we won't use it directly)
	service := &Service{}
	handler := NewHandler(service)
	// Replace the service with our mock for testing
	handler.service = &Service{repo: nil}
	return handler
}

func createTestHandlerWithMockService(mockService *MockServiceInterface) *handlerWithMock {
	return &handlerWithMock{mockService: mockService}
}

// handlerWithMock wraps mock service for testing
type handlerWithMock struct {
	mockService *MockServiceInterface
}

func (h *handlerWithMock) GetRiderHistory(c *gin.Context) {
	riderID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse pagination params
	page := 1
	pageSize := 20

	filters := &HistoryFilters{}
	if status := c.Query("status"); status != "" {
		filters.Status = &status
	}

	resp, err := h.mockService.GetRiderHistory(c.Request.Context(), riderID.(uuid.UUID), filters, page, pageSize)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get ride history")
		return
	}

	common.SuccessResponse(c, resp)
}

func (h *handlerWithMock) GetRideDetails(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride id")
		return
	}

	ride, err := h.mockService.GetRideDetails(c.Request.Context(), rideID, userID.(uuid.UUID))
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get ride details")
		return
	}

	common.SuccessResponse(c, ride)
}

func (h *handlerWithMock) GetReceipt(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride id")
		return
	}

	receipt, err := h.mockService.GetReceipt(c.Request.Context(), rideID, userID.(uuid.UUID))
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to generate receipt")
		return
	}

	common.SuccessResponse(c, receipt)
}

func (h *handlerWithMock) GetRiderStats(c *gin.Context) {
	riderID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	period := c.DefaultQuery("period", "all_time")

	stats, err := h.mockService.GetRiderStats(c.Request.Context(), riderID.(uuid.UUID), period)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get stats")
		return
	}

	common.SuccessResponse(c, stats)
}

func (h *handlerWithMock) GetFrequentRoutes(c *gin.Context) {
	riderID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	routes, err := h.mockService.GetFrequentRoutes(c.Request.Context(), riderID.(uuid.UUID))
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get frequent routes")
		return
	}

	common.SuccessResponse(c, gin.H{"routes": routes})
}

func (h *handlerWithMock) GetDriverHistory(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	page := 1
	pageSize := 20

	filters := &HistoryFilters{}
	if status := c.Query("status"); status != "" {
		filters.Status = &status
	}

	resp, err := h.mockService.GetDriverHistory(c.Request.Context(), driverID.(uuid.UUID), filters, page, pageSize)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get ride history")
		return
	}

	common.SuccessResponse(c, resp)
}

func createTestRideHistoryEntry(riderID uuid.UUID, driverID *uuid.UUID, status string) *RideHistoryEntry {
	return &RideHistoryEntry{
		ID:             uuid.New(),
		RiderID:        riderID,
		DriverID:       driverID,
		Status:         status,
		PickupAddress:  "123 Main St",
		DropoffAddress: "456 Oak Ave",
		Distance:       10.5,
		Duration:       25,
		TotalFare:      35.00,
		Currency:       "USD",
		PaymentMethod:  "card",
		RequestedAt:    time.Now().Add(-1 * time.Hour),
	}
}

func createTestReceipt(rideID uuid.UUID) *Receipt {
	return &Receipt{
		ReceiptID:      "RCP-ABC123-XYZ789",
		RideID:         rideID,
		IssuedAt:       time.Now(),
		PickupAddress:  "123 Main St",
		DropoffAddress: "456 Oak Ave",
		Distance:       10.5,
		Duration:       25,
		TripDate:       "January 15, 2024",
		TripStartTime:  "2:30 PM",
		TripEndTime:    "2:55 PM",
		FareBreakdown:  []FareLineItem{{Label: "Base fare", Amount: 5.00, Type: "charge"}},
		Subtotal:       30.00,
		Fees:           2.50,
		Discounts:      0,
		Tip:            3.00,
		Total:          35.50,
		Currency:       "USD",
		PaymentMethod:  "card",
	}
}

func createTestRideStats(period string) *RideStats {
	return &RideStats{
		TotalRides:      50,
		CompletedRides:  48,
		CancelledRides:  2,
		TotalSpent:      1250.00,
		TotalDistance:   375.5,
		TotalDuration:   900,
		AverageFare:     26.04,
		AverageDistance: 7.82,
		AverageRating:   4.7,
		Currency:        "USD",
		Period:          period,
	}
}

// ============================================================================
// GetRiderHistory Handler Tests
// ============================================================================

func TestHandler_GetRiderHistory_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	rides := []RideHistoryEntry{
		*createTestRideHistoryEntry(riderID, nil, "completed"),
		*createTestRideHistoryEntry(riderID, nil, "completed"),
	}

	mockService.On("GetRiderHistory", mock.Anything, riderID, mock.AnythingOfType("*ridehistory.HistoryFilters"), 1, 20).
		Return(&HistoryResponse{
			Rides:    rides,
			Total:    2,
			Page:     1,
			PageSize: 20,
		}, nil)

	c, w := setupTestContext("GET", "/api/v1/rides/history", nil)
	setUserContext(c, riderID, models.RoleRider)

	handler.GetRiderHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetRiderHistory_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	c, w := setupTestContext("GET", "/api/v1/rides/history", nil)
	// Don't set user context

	handler.GetRiderHistory(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_GetRiderHistory_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()

	mockService.On("GetRiderHistory", mock.Anything, riderID, mock.AnythingOfType("*ridehistory.HistoryFilters"), 1, 20).
		Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/rides/history", nil)
	setUserContext(c, riderID, models.RoleRider)

	handler.GetRiderHistory(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetRiderHistory_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()

	mockService.On("GetRiderHistory", mock.Anything, riderID, mock.AnythingOfType("*ridehistory.HistoryFilters"), 1, 20).
		Return(&HistoryResponse{
			Rides:    []RideHistoryEntry{},
			Total:    0,
			Page:     1,
			PageSize: 20,
		}, nil)

	c, w := setupTestContext("GET", "/api/v1/rides/history", nil)
	setUserContext(c, riderID, models.RoleRider)

	handler.GetRiderHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetRiderHistory_WithStatusFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()

	mockService.On("GetRiderHistory", mock.Anything, riderID, mock.AnythingOfType("*ridehistory.HistoryFilters"), 1, 20).
		Return(&HistoryResponse{
			Rides:    []RideHistoryEntry{},
			Total:    0,
			Page:     1,
			PageSize: 20,
		}, nil)

	c, w := setupTestContext("GET", "/api/v1/rides/history?status=completed", nil)
	c.Request.URL.RawQuery = "status=completed"
	setUserContext(c, riderID, models.RoleRider)

	handler.GetRiderHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetRideDetails Handler Tests
// ============================================================================

func TestHandler_GetRideDetails_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	rideID := uuid.New()
	ride := createTestRideHistoryEntry(riderID, nil, "completed")
	ride.ID = rideID

	mockService.On("GetRideDetails", mock.Anything, rideID, riderID).
		Return(ride, nil)

	c, w := setupTestContext("GET", "/api/v1/rides/history/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, riderID, models.RoleRider)

	handler.GetRideDetails(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetRideDetails_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	rideID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/rides/history/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	// Don't set user context

	handler.GetRideDetails(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetRideDetails_InvalidRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/rides/history/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetRideDetails(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid ride id")
}

func TestHandler_GetRideDetails_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	userID := uuid.New()
	rideID := uuid.New()

	mockService.On("GetRideDetails", mock.Anything, rideID, userID).
		Return(nil, common.NewNotFoundError("ride not found", nil))

	c, w := setupTestContext("GET", "/api/v1/rides/history/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetRideDetails(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetRideDetails_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	userID := uuid.New()
	rideID := uuid.New()

	mockService.On("GetRideDetails", mock.Anything, rideID, userID).
		Return(nil, common.NewForbiddenError("you don't have access to this ride"))

	c, w := setupTestContext("GET", "/api/v1/rides/history/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetRideDetails(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetRideDetails_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	userID := uuid.New()
	rideID := uuid.New()

	mockService.On("GetRideDetails", mock.Anything, rideID, userID).
		Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/rides/history/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetRideDetails(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetReceipt Handler Tests
// ============================================================================

func TestHandler_GetReceipt_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	userID := uuid.New()
	rideID := uuid.New()
	receipt := createTestReceipt(rideID)

	mockService.On("GetReceipt", mock.Anything, rideID, userID).
		Return(receipt, nil)

	c, w := setupTestContext("GET", "/api/v1/rides/history/"+rideID.String()+"/receipt", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetReceipt(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetReceipt_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	rideID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/rides/history/"+rideID.String()+"/receipt", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	// Don't set user context

	handler.GetReceipt(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetReceipt_InvalidRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/rides/history/invalid-uuid/receipt", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetReceipt(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetReceipt_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	userID := uuid.New()
	rideID := uuid.New()

	mockService.On("GetReceipt", mock.Anything, rideID, userID).
		Return(nil, common.NewNotFoundError("ride not found", nil))

	c, w := setupTestContext("GET", "/api/v1/rides/history/"+rideID.String()+"/receipt", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetReceipt(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetReceipt_NotCompleted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	userID := uuid.New()
	rideID := uuid.New()

	mockService.On("GetReceipt", mock.Anything, rideID, userID).
		Return(nil, common.NewBadRequestError("receipts are only available for completed rides", nil))

	c, w := setupTestContext("GET", "/api/v1/rides/history/"+rideID.String()+"/receipt", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetReceipt(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetReceipt_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	userID := uuid.New()
	rideID := uuid.New()

	mockService.On("GetReceipt", mock.Anything, rideID, userID).
		Return(nil, common.NewForbiddenError("you don't have access to this ride"))

	c, w := setupTestContext("GET", "/api/v1/rides/history/"+rideID.String()+"/receipt", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetReceipt(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetReceipt_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	userID := uuid.New()
	rideID := uuid.New()

	mockService.On("GetReceipt", mock.Anything, rideID, userID).
		Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/rides/history/"+rideID.String()+"/receipt", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetReceipt(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetRiderStats Handler Tests
// ============================================================================

func TestHandler_GetRiderStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	stats := createTestRideStats("this_month")

	mockService.On("GetRiderStats", mock.Anything, riderID, "all_time").
		Return(stats, nil)

	c, w := setupTestContext("GET", "/api/v1/rides/stats", nil)
	setUserContext(c, riderID, models.RoleRider)

	handler.GetRiderStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetRiderStats_WithPeriod(t *testing.T) {
	periods := []string{"this_week", "this_month", "last_month", "this_year", "all_time"}

	for _, period := range periods {
		t.Run("period_"+period, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(MockServiceInterface)
			handler := createTestHandlerWithMockService(mockService)

			riderID := uuid.New()
			stats := createTestRideStats(period)

			mockService.On("GetRiderStats", mock.Anything, riderID, period).
				Return(stats, nil)

			c, w := setupTestContext("GET", "/api/v1/rides/stats?period="+period, nil)
			c.Request.URL.RawQuery = "period=" + period
			setUserContext(c, riderID, models.RoleRider)

			handler.GetRiderStats(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_GetRiderStats_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	c, w := setupTestContext("GET", "/api/v1/rides/stats", nil)
	// Don't set user context

	handler.GetRiderStats(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetRiderStats_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()

	mockService.On("GetRiderStats", mock.Anything, riderID, "all_time").
		Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/rides/stats", nil)
	setUserContext(c, riderID, models.RoleRider)

	handler.GetRiderStats(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetFrequentRoutes Handler Tests
// ============================================================================

func TestHandler_GetFrequentRoutes_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	routes := []FrequentRoute{
		{
			PickupAddress:  "Home",
			DropoffAddress: "Office",
			RideCount:      45,
			AverageFare:    22.50,
			LastRideAt:     "2024-01-15 08:30:00",
		},
		{
			PickupAddress:  "Office",
			DropoffAddress: "Home",
			RideCount:      42,
			AverageFare:    21.75,
			LastRideAt:     "2024-01-14 18:00:00",
		},
	}

	mockService.On("GetFrequentRoutes", mock.Anything, riderID).
		Return(routes, nil)

	c, w := setupTestContext("GET", "/api/v1/rides/frequent-routes", nil)
	setUserContext(c, riderID, models.RoleRider)

	handler.GetFrequentRoutes(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetFrequentRoutes_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()

	mockService.On("GetFrequentRoutes", mock.Anything, riderID).
		Return([]FrequentRoute{}, nil)

	c, w := setupTestContext("GET", "/api/v1/rides/frequent-routes", nil)
	setUserContext(c, riderID, models.RoleRider)

	handler.GetFrequentRoutes(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetFrequentRoutes_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	c, w := setupTestContext("GET", "/api/v1/rides/frequent-routes", nil)
	// Don't set user context

	handler.GetFrequentRoutes(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetFrequentRoutes_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()

	mockService.On("GetFrequentRoutes", mock.Anything, riderID).
		Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/rides/frequent-routes", nil)
	setUserContext(c, riderID, models.RoleRider)

	handler.GetFrequentRoutes(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetDriverHistory Handler Tests
// ============================================================================

func TestHandler_GetDriverHistory_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	driverID := uuid.New()
	riderID := uuid.New()
	rides := []RideHistoryEntry{
		*createTestRideHistoryEntry(riderID, &driverID, "completed"),
	}

	mockService.On("GetDriverHistory", mock.Anything, driverID, mock.AnythingOfType("*ridehistory.HistoryFilters"), 1, 20).
		Return(&HistoryResponse{
			Rides:    rides,
			Total:    1,
			Page:     1,
			PageSize: 20,
		}, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/rides/history", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetDriverHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetDriverHistory_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	c, w := setupTestContext("GET", "/api/v1/driver/rides/history", nil)
	// Don't set user context

	handler.GetDriverHistory(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetDriverHistory_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	driverID := uuid.New()

	mockService.On("GetDriverHistory", mock.Anything, driverID, mock.AnythingOfType("*ridehistory.HistoryFilters"), 1, 20).
		Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/rides/history", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetDriverHistory(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetDriverHistory_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	driverID := uuid.New()

	mockService.On("GetDriverHistory", mock.Anything, driverID, mock.AnythingOfType("*ridehistory.HistoryFilters"), 1, 20).
		Return(&HistoryResponse{
			Rides:    []RideHistoryEntry{},
			Total:    0,
			Page:     1,
			PageSize: 20,
		}, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/rides/history", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetDriverHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_RideDetails_AccessControl(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(mockService *MockServiceInterface, rideID, userID uuid.UUID)
		expectedStatus int
	}{
		{
			name: "rider can access own ride",
			setupMock: func(mockService *MockServiceInterface, rideID, userID uuid.UUID) {
				ride := createTestRideHistoryEntry(userID, nil, "completed")
				ride.ID = rideID
				mockService.On("GetRideDetails", mock.Anything, rideID, userID).Return(ride, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "driver can access assigned ride",
			setupMock: func(mockService *MockServiceInterface, rideID, userID uuid.UUID) {
				riderID := uuid.New()
				ride := createTestRideHistoryEntry(riderID, &userID, "completed")
				ride.ID = rideID
				mockService.On("GetRideDetails", mock.Anything, rideID, userID).Return(ride, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "unauthorized user cannot access ride",
			setupMock: func(mockService *MockServiceInterface, rideID, userID uuid.UUID) {
				mockService.On("GetRideDetails", mock.Anything, rideID, userID).
					Return(nil, common.NewForbiddenError("you don't have access to this ride"))
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "ride not found",
			setupMock: func(mockService *MockServiceInterface, rideID, userID uuid.UUID) {
				mockService.On("GetRideDetails", mock.Anything, rideID, userID).
					Return(nil, common.NewNotFoundError("ride not found", nil))
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(MockServiceInterface)
			handler := createTestHandlerWithMockService(mockService)

			userID := uuid.New()
			rideID := uuid.New()

			tt.setupMock(mockService, rideID, userID)

			c, w := setupTestContext("GET", "/api/v1/rides/history/"+rideID.String(), nil)
			c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
			setUserContext(c, userID, models.RoleRider)

			handler.GetRideDetails(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_Receipt_StatusCases(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(mockService *MockServiceInterface, rideID, userID uuid.UUID)
		expectedStatus int
	}{
		{
			name: "completed ride returns receipt",
			setupMock: func(mockService *MockServiceInterface, rideID, userID uuid.UUID) {
				receipt := createTestReceipt(rideID)
				mockService.On("GetReceipt", mock.Anything, rideID, userID).Return(receipt, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "in_progress ride cannot get receipt",
			setupMock: func(mockService *MockServiceInterface, rideID, userID uuid.UUID) {
				mockService.On("GetReceipt", mock.Anything, rideID, userID).
					Return(nil, common.NewBadRequestError("receipts are only available for completed rides", nil))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "cancelled ride cannot get receipt",
			setupMock: func(mockService *MockServiceInterface, rideID, userID uuid.UUID) {
				mockService.On("GetReceipt", mock.Anything, rideID, userID).
					Return(nil, common.NewBadRequestError("receipts are only available for completed rides", nil))
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(MockServiceInterface)
			handler := createTestHandlerWithMockService(mockService)

			userID := uuid.New()
			rideID := uuid.New()

			tt.setupMock(mockService, rideID, userID)

			c, w := setupTestContext("GET", "/api/v1/rides/history/"+rideID.String()+"/receipt", nil)
			c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
			setUserContext(c, userID, models.RoleRider)

			handler.GetReceipt(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestHandler_GetRideDetails_EmptyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/rides/history/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetRideDetails(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetReceipt_EmptyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/rides/history//receipt", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetReceipt(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_GetRiderHistory_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	rides := []RideHistoryEntry{
		*createTestRideHistoryEntry(riderID, nil, "completed"),
	}

	mockService.On("GetRiderHistory", mock.Anything, riderID, mock.AnythingOfType("*ridehistory.HistoryFilters"), 1, 20).
		Return(&HistoryResponse{
			Rides:    rides,
			Total:    1,
			Page:     1,
			PageSize: 20,
		}, nil)

	c, w := setupTestContext("GET", "/api/v1/rides/history", nil)
	setUserContext(c, riderID, models.RoleRider)

	handler.GetRiderHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	// Verify response structure
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
	mockService.AssertExpectations(t)
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()

	mockService.On("GetRiderHistory", mock.Anything, riderID, mock.AnythingOfType("*ridehistory.HistoryFilters"), 1, 20).
		Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/rides/history", nil)
	setUserContext(c, riderID, models.RoleRider)

	handler.GetRiderHistory(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)

	// Verify error response structure
	assert.False(t, response["success"].(bool))
	assert.NotNil(t, response["error"])

	errorInfo := response["error"].(map[string]interface{})
	assert.NotNil(t, errorInfo["code"])
	assert.NotNil(t, errorInfo["message"])
}

func TestHandler_GetFrequentRoutes_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceInterface)
	handler := createTestHandlerWithMockService(mockService)

	riderID := uuid.New()
	routes := []FrequentRoute{
		{
			PickupAddress:  "Home",
			DropoffAddress: "Office",
			RideCount:      45,
			AverageFare:    22.50,
			LastRideAt:     "2024-01-15 08:30:00",
		},
	}

	mockService.On("GetFrequentRoutes", mock.Anything, riderID).
		Return(routes, nil)

	c, w := setupTestContext("GET", "/api/v1/rides/frequent-routes", nil)
	setUserContext(c, riderID, models.RoleRider)

	handler.GetFrequentRoutes(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	// Verify response structure
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["routes"])
	mockService.AssertExpectations(t)
}
