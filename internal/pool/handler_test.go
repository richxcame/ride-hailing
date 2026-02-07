package pool

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// Mock Service Interface and Implementation
// ============================================================================

// PoolServiceInterface defines the interface for pool service used by handler tests
type PoolServiceInterface interface {
	RequestPoolRide(ctx context.Context, riderID uuid.UUID, req *RequestPoolRideRequest) (*RequestPoolRideResponse, error)
	ConfirmPoolRide(ctx context.Context, riderID uuid.UUID, req *ConfirmPoolRequest) error
	GetPoolStatus(ctx context.Context, riderID uuid.UUID, poolRideID uuid.UUID) (*GetPoolStatusResponse, error)
	CancelPoolRide(ctx context.Context, riderID uuid.UUID, poolPassengerID uuid.UUID) error
	GetDriverPoolRide(ctx context.Context, driverID uuid.UUID) (*DriverPoolRideInfo, error)
	StartPoolRide(ctx context.Context, driverID uuid.UUID, poolRideID uuid.UUID) error
	PickupPassenger(ctx context.Context, driverID uuid.UUID, passengerID uuid.UUID) error
	DropoffPassenger(ctx context.Context, driverID uuid.UUID, passengerID uuid.UUID) error
	GetPoolStats(ctx context.Context) (*PoolStatsResponse, error)
}

// MockPoolService is a mock implementation of PoolServiceInterface
type MockPoolService struct {
	mock.Mock
}

func (m *MockPoolService) RequestPoolRide(ctx context.Context, riderID uuid.UUID, req *RequestPoolRideRequest) (*RequestPoolRideResponse, error) {
	args := m.Called(ctx, riderID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RequestPoolRideResponse), args.Error(1)
}

func (m *MockPoolService) ConfirmPoolRide(ctx context.Context, riderID uuid.UUID, req *ConfirmPoolRequest) error {
	args := m.Called(ctx, riderID, req)
	return args.Error(0)
}

func (m *MockPoolService) GetPoolStatus(ctx context.Context, riderID uuid.UUID, poolRideID uuid.UUID) (*GetPoolStatusResponse, error) {
	args := m.Called(ctx, riderID, poolRideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GetPoolStatusResponse), args.Error(1)
}

func (m *MockPoolService) CancelPoolRide(ctx context.Context, riderID uuid.UUID, poolPassengerID uuid.UUID) error {
	args := m.Called(ctx, riderID, poolPassengerID)
	return args.Error(0)
}

func (m *MockPoolService) GetDriverPoolRide(ctx context.Context, driverID uuid.UUID) (*DriverPoolRideInfo, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverPoolRideInfo), args.Error(1)
}

func (m *MockPoolService) StartPoolRide(ctx context.Context, driverID uuid.UUID, poolRideID uuid.UUID) error {
	args := m.Called(ctx, driverID, poolRideID)
	return args.Error(0)
}

func (m *MockPoolService) PickupPassenger(ctx context.Context, driverID uuid.UUID, passengerID uuid.UUID) error {
	args := m.Called(ctx, driverID, passengerID)
	return args.Error(0)
}

func (m *MockPoolService) DropoffPassenger(ctx context.Context, driverID uuid.UUID, passengerID uuid.UUID) error {
	args := m.Called(ctx, driverID, passengerID)
	return args.Error(0)
}

func (m *MockPoolService) GetPoolStats(ctx context.Context) (*PoolStatsResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PoolStatsResponse), args.Error(1)
}

// MockRepository mocks the repository for GetActivePassengerForRider
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetActivePassengerForRider(ctx context.Context, riderID uuid.UUID) (*PoolPassenger, error) {
	args := m.Called(ctx, riderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PoolPassenger), args.Error(1)
}

// ============================================================================
// Testable Handler (wraps mock service)
// ============================================================================

// TestableHandler provides testable handler methods with mock service injection
type TestableHandler struct {
	service  *MockPoolService
	mockRepo *MockRepository
}

func NewTestableHandler(mockService *MockPoolService, mockRepo *MockRepository) *TestableHandler {
	return &TestableHandler{service: mockService, mockRepo: mockRepo}
}

// RequestPoolRide handles pool ride request - testable version
func (h *TestableHandler) RequestPoolRide(c *gin.Context) {
	riderID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req RequestPoolRideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.RequestPoolRide(c.Request.Context(), riderID.(uuid.UUID), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to request pool ride")
		return
	}

	common.SuccessResponse(c, result)
}

// ConfirmPoolRide handles pool ride confirmation - testable version
func (h *TestableHandler) ConfirmPoolRide(c *gin.Context) {
	riderID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req ConfirmPoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.ConfirmPoolRide(c.Request.Context(), riderID.(uuid.UUID), &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to confirm pool")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Pool ride " + map[bool]string{true: "confirmed", false: "declined"}[req.Accept],
	})
}

// GetPoolStatus handles getting pool status - testable version
func (h *TestableHandler) GetPoolStatus(c *gin.Context) {
	riderID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	poolRideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid pool ride ID")
		return
	}

	result, err := h.service.GetPoolStatus(c.Request.Context(), riderID.(uuid.UUID), poolRideID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "pool ride not found")
		return
	}

	common.SuccessResponse(c, result)
}

// CancelPoolRide handles pool ride cancellation - testable version
func (h *TestableHandler) CancelPoolRide(c *gin.Context) {
	riderID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	poolPassengerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid pool passenger ID")
		return
	}

	if err := h.service.CancelPoolRide(c.Request.Context(), riderID.(uuid.UUID), poolPassengerID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to cancel pool ride")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Pool ride cancelled successfully",
	})
}

// GetActivePool handles getting active pool - testable version
func (h *TestableHandler) GetActivePool(c *gin.Context) {
	riderID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	passenger, err := h.mockRepo.GetActivePassengerForRider(c.Request.Context(), riderID.(uuid.UUID))
	if err != nil {
		common.SuccessResponse(c, gin.H{
			"active_pool": nil,
			"message":     "No active pool ride",
		})
		return
	}

	poolStatus, err := h.service.GetPoolStatus(c.Request.Context(), riderID.(uuid.UUID), passenger.PoolRideID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get pool status")
		return
	}

	common.SuccessResponse(c, gin.H{
		"active_pool":  poolStatus,
		"passenger_id": passenger.ID,
		"my_fare":      passenger.PoolFare,
		"savings":      passenger.SavingsPercent,
	})
}

// GetDriverPool handles getting driver pool - testable version
func (h *TestableHandler) GetDriverPool(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	result, err := h.service.GetDriverPoolRide(c.Request.Context(), driverID.(uuid.UUID))
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.SuccessResponse(c, gin.H{
			"active_pool": nil,
			"message":     "No active pool ride",
		})
		return
	}

	common.SuccessResponse(c, result)
}

// StartPoolRide handles starting pool ride - testable version
func (h *TestableHandler) StartPoolRide(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	poolRideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid pool ride ID")
		return
	}

	if err := h.service.StartPoolRide(c.Request.Context(), driverID.(uuid.UUID), poolRideID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to start pool ride")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Pool ride started",
	})
}

// PickupPassenger handles passenger pickup - testable version
func (h *TestableHandler) PickupPassenger(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	passengerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid passenger ID")
		return
	}

	if err := h.service.PickupPassenger(c.Request.Context(), driverID.(uuid.UUID), passengerID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to pickup passenger")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Passenger picked up",
	})
}

// DropoffPassenger handles passenger dropoff - testable version
func (h *TestableHandler) DropoffPassenger(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	passengerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid passenger ID")
		return
	}

	if err := h.service.DropoffPassenger(c.Request.Context(), driverID.(uuid.UUID), passengerID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to dropoff passenger")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Passenger dropped off",
	})
}

// GetPoolStats handles getting pool stats - testable version
func (h *TestableHandler) GetPoolStats(c *gin.Context) {
	stats, err := h.service.GetPoolStats(c.Request.Context())
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get pool statistics")
		return
	}

	common.SuccessResponse(c, stats)
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

func setUserContext(c *gin.Context, userID uuid.UUID) {
	c.Set("user_id", userID)
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

func createTestPoolRide() *PoolRide {
	driverID := uuid.New()
	now := time.Now()
	return &PoolRide{
		ID:                uuid.New(),
		DriverID:          &driverID,
		Status:            PoolStatusConfirmed,
		MaxPassengers:     4,
		CurrentPassengers: 2,
		TotalDistance:     10.5,
		TotalDuration:     25,
		CenterLat:         37.7749,
		CenterLng:         -122.4194,
		RadiusKm:          3.0,
		H3Index:           "872830828ffffff",
		BaseFare:          2.0,
		PerKmRate:         1.5,
		PerMinuteRate:     0.25,
		MatchDeadline:     now.Add(5 * time.Minute),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func createTestPoolPassenger(poolRideID uuid.UUID, riderID uuid.UUID) *PoolPassenger {
	now := time.Now()
	return &PoolPassenger{
		ID:              uuid.New(),
		PoolRideID:      poolRideID,
		RiderID:         riderID,
		Status:          PassengerStatusConfirmed,
		PickupLocation:  Location{Latitude: 37.7749, Longitude: -122.4194},
		DropoffLocation: Location{Latitude: 37.7849, Longitude: -122.4094},
		PickupAddress:   "123 Market St",
		DropoffAddress:  "456 Mission St",
		DirectDistance:  5.0,
		DirectDuration:  15,
		OriginalFare:    15.0,
		PoolFare:        11.25,
		SavingsPercent:  25.0,
		EstimatedPickup: now.Add(5 * time.Minute),
		EstimatedDropoff: now.Add(20 * time.Minute),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func createTestRequestPoolRideResponse() *RequestPoolRideResponse {
	poolRideID := uuid.New()
	now := time.Now()
	return &RequestPoolRideResponse{
		PoolPassengerID:   uuid.New(),
		PoolRideID:        &poolRideID,
		Status:            PassengerStatusPending,
		MatchFound:        true,
		OriginalFare:      15.0,
		PoolFare:          11.25,
		SavingsPercent:    25.0,
		EstimatedPickup:   now.Add(5 * time.Minute),
		EstimatedDropoff:  now.Add(20 * time.Minute),
		MatchDeadline:     now.Add(5 * time.Minute),
		CurrentPassengers: 1,
		MaxPassengers:     4,
		Message:           "Great! We found a pool match.",
	}
}

func createTestGetPoolStatusResponse(poolRide *PoolRide) *GetPoolStatusResponse {
	return &GetPoolStatusResponse{
		PoolRide:        poolRide,
		MyStatus:        PassengerStatusConfirmed,
		OtherPassengers: []PassengerInfo{},
		RouteStops:      []RouteStop{},
		EstimatedArrival: time.Now().Add(20 * time.Minute),
	}
}

func createTestDriverPoolRideInfo(poolRide *PoolRide) *DriverPoolRideInfo {
	return &DriverPoolRideInfo{
		PoolRide:       poolRide,
		Passengers:     []PoolPassenger{},
		RouteStops:     []RouteStop{},
		NextStop:       nil,
		TotalFare:      22.50,
		DriverEarnings: 16.88,
	}
}

func createTestPoolStatsResponse() *PoolStatsResponse {
	return &PoolStatsResponse{
		TotalPoolRides:       1500,
		ActivePoolRides:      45,
		AvgPassengersPerPool: 2.3,
		AvgSavingsPercent:    22.5,
		AvgMatchTime:         45.0,
		MatchSuccessRate:     0.78,
		TotalCO2Saved:        1250.5,
	}
}

// ============================================================================
// RequestPoolRide Handler Tests
// ============================================================================

func TestHandler_RequestPoolRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	expectedResponse := createTestRequestPoolRideResponse()

	reqBody := RequestPoolRideRequest{
		PickupLocation:  Location{Latitude: 37.7749, Longitude: -122.4194},
		DropoffLocation: Location{Latitude: 37.7849, Longitude: -122.4094},
		PickupAddress:   "123 Market St",
		DropoffAddress:  "456 Mission St",
		PassengerCount:  1,
		MaxWaitMinutes:  5,
	}

	mockService.On("RequestPoolRide", mock.Anything, riderID, &reqBody).Return(expectedResponse, nil)

	c, w := setupTestContext("POST", "/api/v1/pool/request", reqBody)
	setUserContext(c, riderID)

	handler.RequestPoolRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
	mockService.AssertExpectations(t)
}

func TestHandler_RequestPoolRide_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	reqBody := RequestPoolRideRequest{
		PickupLocation:  Location{Latitude: 37.7749, Longitude: -122.4194},
		DropoffLocation: Location{Latitude: 37.7849, Longitude: -122.4094},
	}

	c, w := setupTestContext("POST", "/api/v1/pool/request", reqBody)
	// Don't set user context

	handler.RequestPoolRide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_RequestPoolRide_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/pool/request", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/pool/request", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, riderID)

	handler.RequestPoolRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_RequestPoolRide_MissingPickupLocation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()

	// Handler passes to service, service validates and returns error
	mockService.On("RequestPoolRide", mock.Anything, mock.Anything, mock.Anything).Return(nil, common.NewAppError(http.StatusBadRequest, "pickup location is required", nil))

	reqBody := map[string]interface{}{
		"dropoff_location": map[string]float64{"latitude": 37.7849, "longitude": -122.4094},
	}

	c, w := setupTestContext("POST", "/api/v1/pool/request", reqBody)
	setUserContext(c, riderID)

	handler.RequestPoolRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RequestPoolRide_MissingDropoffLocation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()

	// Handler passes to service, service validates and returns error
	mockService.On("RequestPoolRide", mock.Anything, mock.Anything, mock.Anything).Return(nil, common.NewAppError(http.StatusBadRequest, "dropoff location is required", nil))

	reqBody := map[string]interface{}{
		"pickup_location": map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
	}

	c, w := setupTestContext("POST", "/api/v1/pool/request", reqBody)
	setUserContext(c, riderID)

	handler.RequestPoolRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RequestPoolRide_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()

	reqBody := RequestPoolRideRequest{
		PickupLocation:  Location{Latitude: 37.7749, Longitude: -122.4194},
		DropoffLocation: Location{Latitude: 37.7849, Longitude: -122.4094},
	}

	mockService.On("RequestPoolRide", mock.Anything, riderID, &reqBody).Return(nil, errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/pool/request", reqBody)
	setUserContext(c, riderID)

	handler.RequestPoolRide(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_RequestPoolRide_AppError_AlreadyHasActiveRide(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()

	reqBody := RequestPoolRideRequest{
		PickupLocation:  Location{Latitude: 37.7749, Longitude: -122.4194},
		DropoffLocation: Location{Latitude: 37.7849, Longitude: -122.4094},
	}

	mockService.On("RequestPoolRide", mock.Anything, riderID, &reqBody).Return(nil, common.NewBadRequestError("you already have an active pool ride", nil))

	c, w := setupTestContext("POST", "/api/v1/pool/request", reqBody)
	setUserContext(c, riderID)

	handler.RequestPoolRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_RequestPoolRide_WithOptionalFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	expectedResponse := createTestRequestPoolRideResponse()

	reqBody := RequestPoolRideRequest{
		PickupLocation:  Location{Latitude: 37.7749, Longitude: -122.4194},
		DropoffLocation: Location{Latitude: 37.7849, Longitude: -122.4094},
		PickupAddress:   "123 Market St, San Francisco, CA",
		DropoffAddress:  "456 Mission St, San Francisco, CA",
		PassengerCount:  2,
		MaxWaitMinutes:  10,
	}

	mockService.On("RequestPoolRide", mock.Anything, riderID, &reqBody).Return(expectedResponse, nil)

	c, w := setupTestContext("POST", "/api/v1/pool/request", reqBody)
	setUserContext(c, riderID)

	handler.RequestPoolRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// ConfirmPoolRide Handler Tests
// ============================================================================

func TestHandler_ConfirmPoolRide_Success_Accept(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	passengerID := uuid.New()

	reqBody := ConfirmPoolRequest{
		PoolPassengerID: passengerID,
		Accept:          true,
	}

	mockService.On("ConfirmPoolRide", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("*pool.ConfirmPoolRequest")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/pool/confirm", reqBody)
	setUserContext(c, riderID)

	handler.ConfirmPoolRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Contains(t, data["message"].(string), "confirmed")
	mockService.AssertExpectations(t)
}

func TestHandler_ConfirmPoolRide_Success_Decline(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	passengerID := uuid.New()

	// Note: binding:"required" on Accept bool field in ConfirmPoolRequest
	// causes Gin validator to reject false values as "missing required field".
	// This test documents the current validation behavior.
	reqBody := ConfirmPoolRequest{
		PoolPassengerID: passengerID,
		Accept:          false,
	}

	// Service won't be called if validation fails
	mockService.On("ConfirmPoolRide", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("*pool.ConfirmPoolRequest")).Return(nil).Maybe()

	c, w := setupTestContext("POST", "/api/v1/pool/confirm", reqBody)
	setUserContext(c, riderID)

	handler.ConfirmPoolRide(c)

	// Due to binding:"required" on bool Accept field, false value fails validation
	// This documents the current behavior - consider removing binding from bool field
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ConfirmPoolRide_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	passengerID := uuid.New()

	reqBody := ConfirmPoolRequest{
		PoolPassengerID: passengerID,
		Accept:          true,
	}

	c, w := setupTestContext("POST", "/api/v1/pool/confirm", reqBody)
	// Don't set user context

	handler.ConfirmPoolRide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_ConfirmPoolRide_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/pool/confirm", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/pool/confirm", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, riderID)

	handler.ConfirmPoolRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ConfirmPoolRide_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	passengerID := uuid.New()

	reqBody := ConfirmPoolRequest{
		PoolPassengerID: passengerID,
		Accept:          true,
	}

	mockService.On("ConfirmPoolRide", mock.Anything, riderID, &reqBody).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/pool/confirm", reqBody)
	setUserContext(c, riderID)

	handler.ConfirmPoolRide(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ConfirmPoolRide_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	passengerID := uuid.New()

	reqBody := ConfirmPoolRequest{
		PoolPassengerID: passengerID,
		Accept:          true,
	}

	mockService.On("ConfirmPoolRide", mock.Anything, riderID, &reqBody).Return(common.NewNotFoundError("pool passenger not found", nil))

	c, w := setupTestContext("POST", "/api/v1/pool/confirm", reqBody)
	setUserContext(c, riderID)

	handler.ConfirmPoolRide(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ConfirmPoolRide_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	passengerID := uuid.New()

	reqBody := ConfirmPoolRequest{
		PoolPassengerID: passengerID,
		Accept:          true,
	}

	mockService.On("ConfirmPoolRide", mock.Anything, riderID, &reqBody).Return(common.NewForbiddenError("not authorized to confirm this pool"))

	c, w := setupTestContext("POST", "/api/v1/pool/confirm", reqBody)
	setUserContext(c, riderID)

	handler.ConfirmPoolRide(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetPoolStatus Handler Tests
// ============================================================================

func TestHandler_GetPoolStatus_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	poolRideID := uuid.New()
	poolRide := createTestPoolRide()
	poolRide.ID = poolRideID
	expectedResponse := createTestGetPoolStatusResponse(poolRide)

	mockService.On("GetPoolStatus", mock.Anything, riderID, poolRideID).Return(expectedResponse, nil)

	c, w := setupTestContext("GET", "/api/v1/pool/"+poolRideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: poolRideID.String()}}
	setUserContext(c, riderID)

	handler.GetPoolStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetPoolStatus_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	poolRideID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/pool/"+poolRideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: poolRideID.String()}}
	// Don't set user context

	handler.GetPoolStatus(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetPoolStatus_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/pool/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, riderID)

	handler.GetPoolStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid pool ride ID")
}

func TestHandler_GetPoolStatus_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	poolRideID := uuid.New()

	mockService.On("GetPoolStatus", mock.Anything, riderID, poolRideID).Return(nil, common.NewNotFoundError("pool ride not found", nil))

	c, w := setupTestContext("GET", "/api/v1/pool/"+poolRideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: poolRideID.String()}}
	setUserContext(c, riderID)

	handler.GetPoolStatus(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetPoolStatus_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	poolRideID := uuid.New()

	mockService.On("GetPoolStatus", mock.Anything, riderID, poolRideID).Return(nil, common.NewForbiddenError("not authorized to view this pool"))

	c, w := setupTestContext("GET", "/api/v1/pool/"+poolRideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: poolRideID.String()}}
	setUserContext(c, riderID)

	handler.GetPoolStatus(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetPoolStatus_EmptyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/pool/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	setUserContext(c, riderID)

	handler.GetPoolStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// CancelPoolRide Handler Tests
// ============================================================================

func TestHandler_CancelPoolRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	poolPassengerID := uuid.New()

	mockService.On("CancelPoolRide", mock.Anything, riderID, poolPassengerID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/pool/"+poolPassengerID.String()+"/cancel", nil)
	c.Params = gin.Params{{Key: "id", Value: poolPassengerID.String()}}
	setUserContext(c, riderID)

	handler.CancelPoolRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Contains(t, data["message"], "cancelled successfully")
	mockService.AssertExpectations(t)
}

func TestHandler_CancelPoolRide_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	poolPassengerID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/pool/"+poolPassengerID.String()+"/cancel", nil)
	c.Params = gin.Params{{Key: "id", Value: poolPassengerID.String()}}
	// Don't set user context

	handler.CancelPoolRide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CancelPoolRide_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/pool/invalid-uuid/cancel", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, riderID)

	handler.CancelPoolRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid pool passenger ID")
}

func TestHandler_CancelPoolRide_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	poolPassengerID := uuid.New()

	mockService.On("CancelPoolRide", mock.Anything, riderID, poolPassengerID).Return(common.NewNotFoundError("pool passenger not found", nil))

	c, w := setupTestContext("POST", "/api/v1/pool/"+poolPassengerID.String()+"/cancel", nil)
	c.Params = gin.Params{{Key: "id", Value: poolPassengerID.String()}}
	setUserContext(c, riderID)

	handler.CancelPoolRide(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CancelPoolRide_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	poolPassengerID := uuid.New()

	mockService.On("CancelPoolRide", mock.Anything, riderID, poolPassengerID).Return(common.NewForbiddenError("not authorized to cancel this pool"))

	c, w := setupTestContext("POST", "/api/v1/pool/"+poolPassengerID.String()+"/cancel", nil)
	c.Params = gin.Params{{Key: "id", Value: poolPassengerID.String()}}
	setUserContext(c, riderID)

	handler.CancelPoolRide(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CancelPoolRide_CannotCancelAfterPickup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	poolPassengerID := uuid.New()

	mockService.On("CancelPoolRide", mock.Anything, riderID, poolPassengerID).Return(common.NewBadRequestError("cannot cancel after pickup", nil))

	c, w := setupTestContext("POST", "/api/v1/pool/"+poolPassengerID.String()+"/cancel", nil)
	c.Params = gin.Params{{Key: "id", Value: poolPassengerID.String()}}
	setUserContext(c, riderID)

	handler.CancelPoolRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CancelPoolRide_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	poolPassengerID := uuid.New()

	mockService.On("CancelPoolRide", mock.Anything, riderID, poolPassengerID).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/pool/"+poolPassengerID.String()+"/cancel", nil)
	c.Params = gin.Params{{Key: "id", Value: poolPassengerID.String()}}
	setUserContext(c, riderID)

	handler.CancelPoolRide(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetActivePool Handler Tests
// ============================================================================

func TestHandler_GetActivePool_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	poolRideID := uuid.New()
	poolRide := createTestPoolRide()
	poolRide.ID = poolRideID
	passenger := createTestPoolPassenger(poolRideID, riderID)
	poolStatus := createTestGetPoolStatusResponse(poolRide)

	mockRepo.On("GetActivePassengerForRider", mock.Anything, riderID).Return(passenger, nil)
	mockService.On("GetPoolStatus", mock.Anything, riderID, poolRideID).Return(poolStatus, nil)

	c, w := setupTestContext("GET", "/api/v1/pool/active", nil)
	setUserContext(c, riderID)

	handler.GetActivePool(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["active_pool"])
	assert.NotNil(t, data["passenger_id"])
	mockRepo.AssertExpectations(t)
	mockService.AssertExpectations(t)
}

func TestHandler_GetActivePool_NoActiveRide(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()

	mockRepo.On("GetActivePassengerForRider", mock.Anything, riderID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/pool/active", nil)
	setUserContext(c, riderID)

	handler.GetActivePool(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Nil(t, data["active_pool"])
	assert.Equal(t, "No active pool ride", data["message"])
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetActivePool_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	c, w := setupTestContext("GET", "/api/v1/pool/active", nil)
	// Don't set user context

	handler.GetActivePool(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetActivePool_GetPoolStatusError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	poolRideID := uuid.New()
	passenger := createTestPoolPassenger(poolRideID, riderID)

	mockRepo.On("GetActivePassengerForRider", mock.Anything, riderID).Return(passenger, nil)
	mockService.On("GetPoolStatus", mock.Anything, riderID, poolRideID).Return(nil, errors.New("service error"))

	c, w := setupTestContext("GET", "/api/v1/pool/active", nil)
	setUserContext(c, riderID)

	handler.GetActivePool(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetDriverPool Handler Tests
// ============================================================================

func TestHandler_GetDriverPool_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()
	poolRide := createTestPoolRide()
	expectedResponse := createTestDriverPoolRideInfo(poolRide)

	mockService.On("GetDriverPoolRide", mock.Anything, driverID).Return(expectedResponse, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/pool", nil)
	setUserContext(c, driverID)

	handler.GetDriverPool(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetDriverPool_NoActiveRide(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()

	mockService.On("GetDriverPoolRide", mock.Anything, driverID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/driver/pool", nil)
	setUserContext(c, driverID)

	handler.GetDriverPool(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Nil(t, data["active_pool"])
	assert.Equal(t, "No active pool ride", data["message"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetDriverPool_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	c, w := setupTestContext("GET", "/api/v1/driver/pool", nil)
	// Don't set user context

	handler.GetDriverPool(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetDriverPool_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()

	mockService.On("GetDriverPoolRide", mock.Anything, driverID).Return(nil, common.NewNotFoundError("no active pool ride", nil))

	c, w := setupTestContext("GET", "/api/v1/driver/pool", nil)
	setUserContext(c, driverID)

	handler.GetDriverPool(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// StartPoolRide Handler Tests
// ============================================================================

func TestHandler_StartPoolRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()
	poolRideID := uuid.New()

	mockService.On("StartPoolRide", mock.Anything, driverID, poolRideID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/pool/"+poolRideID.String()+"/start", nil)
	c.Params = gin.Params{{Key: "id", Value: poolRideID.String()}}
	setUserContext(c, driverID)

	handler.StartPoolRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "Pool ride started", data["message"])
	mockService.AssertExpectations(t)
}

func TestHandler_StartPoolRide_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	poolRideID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/pool/"+poolRideID.String()+"/start", nil)
	c.Params = gin.Params{{Key: "id", Value: poolRideID.String()}}
	// Don't set user context

	handler.StartPoolRide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_StartPoolRide_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/pool/invalid-uuid/start", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, driverID)

	handler.StartPoolRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid pool ride ID")
}

func TestHandler_StartPoolRide_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()
	poolRideID := uuid.New()

	mockService.On("StartPoolRide", mock.Anything, driverID, poolRideID).Return(common.NewNotFoundError("pool ride not found", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/pool/"+poolRideID.String()+"/start", nil)
	c.Params = gin.Params{{Key: "id", Value: poolRideID.String()}}
	setUserContext(c, driverID)

	handler.StartPoolRide(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_StartPoolRide_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()
	poolRideID := uuid.New()

	mockService.On("StartPoolRide", mock.Anything, driverID, poolRideID).Return(common.NewForbiddenError("not authorized to start this pool"))

	c, w := setupTestContext("POST", "/api/v1/driver/pool/"+poolRideID.String()+"/start", nil)
	c.Params = gin.Params{{Key: "id", Value: poolRideID.String()}}
	setUserContext(c, driverID)

	handler.StartPoolRide(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_StartPoolRide_NotConfirmed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()
	poolRideID := uuid.New()

	mockService.On("StartPoolRide", mock.Anything, driverID, poolRideID).Return(common.NewBadRequestError("pool ride is not confirmed", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/pool/"+poolRideID.String()+"/start", nil)
	c.Params = gin.Params{{Key: "id", Value: poolRideID.String()}}
	setUserContext(c, driverID)

	handler.StartPoolRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_StartPoolRide_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()
	poolRideID := uuid.New()

	mockService.On("StartPoolRide", mock.Anything, driverID, poolRideID).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/driver/pool/"+poolRideID.String()+"/start", nil)
	c.Params = gin.Params{{Key: "id", Value: poolRideID.String()}}
	setUserContext(c, driverID)

	handler.StartPoolRide(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// PickupPassenger Handler Tests
// ============================================================================

func TestHandler_PickupPassenger_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()
	passengerID := uuid.New()

	mockService.On("PickupPassenger", mock.Anything, driverID, passengerID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/pool/passenger/"+passengerID.String()+"/pickup", nil)
	c.Params = gin.Params{{Key: "id", Value: passengerID.String()}}
	setUserContext(c, driverID)

	handler.PickupPassenger(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "Passenger picked up", data["message"])
	mockService.AssertExpectations(t)
}

func TestHandler_PickupPassenger_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	passengerID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/pool/passenger/"+passengerID.String()+"/pickup", nil)
	c.Params = gin.Params{{Key: "id", Value: passengerID.String()}}
	// Don't set user context

	handler.PickupPassenger(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_PickupPassenger_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/pool/passenger/invalid-uuid/pickup", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, driverID)

	handler.PickupPassenger(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid passenger ID")
}

func TestHandler_PickupPassenger_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()
	passengerID := uuid.New()

	mockService.On("PickupPassenger", mock.Anything, driverID, passengerID).Return(common.NewNotFoundError("passenger not found", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/pool/passenger/"+passengerID.String()+"/pickup", nil)
	c.Params = gin.Params{{Key: "id", Value: passengerID.String()}}
	setUserContext(c, driverID)

	handler.PickupPassenger(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_PickupPassenger_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()
	passengerID := uuid.New()

	mockService.On("PickupPassenger", mock.Anything, driverID, passengerID).Return(common.NewForbiddenError("passenger not in your pool"))

	c, w := setupTestContext("POST", "/api/v1/driver/pool/passenger/"+passengerID.String()+"/pickup", nil)
	c.Params = gin.Params{{Key: "id", Value: passengerID.String()}}
	setUserContext(c, driverID)

	handler.PickupPassenger(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_PickupPassenger_NotConfirmed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()
	passengerID := uuid.New()

	mockService.On("PickupPassenger", mock.Anything, driverID, passengerID).Return(common.NewBadRequestError("passenger is not confirmed", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/pool/passenger/"+passengerID.String()+"/pickup", nil)
	c.Params = gin.Params{{Key: "id", Value: passengerID.String()}}
	setUserContext(c, driverID)

	handler.PickupPassenger(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_PickupPassenger_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()
	passengerID := uuid.New()

	mockService.On("PickupPassenger", mock.Anything, driverID, passengerID).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/driver/pool/passenger/"+passengerID.String()+"/pickup", nil)
	c.Params = gin.Params{{Key: "id", Value: passengerID.String()}}
	setUserContext(c, driverID)

	handler.PickupPassenger(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// DropoffPassenger Handler Tests
// ============================================================================

func TestHandler_DropoffPassenger_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()
	passengerID := uuid.New()

	mockService.On("DropoffPassenger", mock.Anything, driverID, passengerID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/pool/passenger/"+passengerID.String()+"/dropoff", nil)
	c.Params = gin.Params{{Key: "id", Value: passengerID.String()}}
	setUserContext(c, driverID)

	handler.DropoffPassenger(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "Passenger dropped off", data["message"])
	mockService.AssertExpectations(t)
}

func TestHandler_DropoffPassenger_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	passengerID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/pool/passenger/"+passengerID.String()+"/dropoff", nil)
	c.Params = gin.Params{{Key: "id", Value: passengerID.String()}}
	// Don't set user context

	handler.DropoffPassenger(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_DropoffPassenger_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/pool/passenger/invalid-uuid/dropoff", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, driverID)

	handler.DropoffPassenger(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid passenger ID")
}

func TestHandler_DropoffPassenger_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()
	passengerID := uuid.New()

	mockService.On("DropoffPassenger", mock.Anything, driverID, passengerID).Return(common.NewNotFoundError("passenger not found", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/pool/passenger/"+passengerID.String()+"/dropoff", nil)
	c.Params = gin.Params{{Key: "id", Value: passengerID.String()}}
	setUserContext(c, driverID)

	handler.DropoffPassenger(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_DropoffPassenger_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()
	passengerID := uuid.New()

	mockService.On("DropoffPassenger", mock.Anything, driverID, passengerID).Return(common.NewForbiddenError("passenger not in your pool"))

	c, w := setupTestContext("POST", "/api/v1/driver/pool/passenger/"+passengerID.String()+"/dropoff", nil)
	c.Params = gin.Params{{Key: "id", Value: passengerID.String()}}
	setUserContext(c, driverID)

	handler.DropoffPassenger(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_DropoffPassenger_NotPickedUp(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()
	passengerID := uuid.New()

	mockService.On("DropoffPassenger", mock.Anything, driverID, passengerID).Return(common.NewBadRequestError("passenger is not picked up", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/pool/passenger/"+passengerID.String()+"/dropoff", nil)
	c.Params = gin.Params{{Key: "id", Value: passengerID.String()}}
	setUserContext(c, driverID)

	handler.DropoffPassenger(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_DropoffPassenger_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	driverID := uuid.New()
	passengerID := uuid.New()

	mockService.On("DropoffPassenger", mock.Anything, driverID, passengerID).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/driver/pool/passenger/"+passengerID.String()+"/dropoff", nil)
	c.Params = gin.Params{{Key: "id", Value: passengerID.String()}}
	setUserContext(c, driverID)

	handler.DropoffPassenger(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetPoolStats Handler Tests
// ============================================================================

func TestHandler_GetPoolStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	expectedStats := createTestPoolStatsResponse()

	mockService.On("GetPoolStats", mock.Anything).Return(expectedStats, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/pool/stats", nil)

	handler.GetPoolStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetPoolStats_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	mockService.On("GetPoolStats", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/pool/stats", nil)

	handler.GetPoolStats(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "failed to get pool statistics")
	mockService.AssertExpectations(t)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_RequestPoolRide_ValidationCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           interface{}
		setupMock      func(*MockPoolService, uuid.UUID)
		expectedStatus int
	}{
		{
			name: "valid request with all fields",
			body: map[string]interface{}{
				"pickup_location":  map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
				"dropoff_location": map[string]float64{"latitude": 37.7849, "longitude": -122.4094},
				"pickup_address":   "123 Market St",
				"dropoff_address":  "456 Mission St",
				"passenger_count":  1,
				"max_wait_minutes": 5,
			},
			setupMock: func(m *MockPoolService, riderID uuid.UUID) {
				m.On("RequestPoolRide", mock.Anything, riderID, mock.AnythingOfType("*pool.RequestPoolRideRequest")).Return(createTestRequestPoolRideResponse(), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing pickup_location",
			body: map[string]interface{}{
				"dropoff_location": map[string]float64{"latitude": 37.7849, "longitude": -122.4094},
			},
			setupMock: func(m *MockPoolService, riderID uuid.UUID) {
				// Handler passes to service, service validates and returns error
				m.On("RequestPoolRide", mock.Anything, mock.Anything, mock.Anything).Return(nil, common.NewAppError(http.StatusBadRequest, "pickup location is required", nil))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing dropoff_location",
			body: map[string]interface{}{
				"pickup_location": map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
			},
			setupMock: func(m *MockPoolService, riderID uuid.UUID) {
				// Handler passes to service, service validates and returns error
				m.On("RequestPoolRide", mock.Anything, mock.Anything, mock.Anything).Return(nil, common.NewAppError(http.StatusBadRequest, "dropoff location is required", nil))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "valid minimal request",
			body: map[string]interface{}{
				"pickup_location":  map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
				"dropoff_location": map[string]float64{"latitude": 37.7849, "longitude": -122.4094},
			},
			setupMock: func(m *MockPoolService, riderID uuid.UUID) {
				m.On("RequestPoolRide", mock.Anything, riderID, mock.AnythingOfType("*pool.RequestPoolRideRequest")).Return(createTestRequestPoolRideResponse(), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "multiple passengers",
			body: map[string]interface{}{
				"pickup_location":  map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
				"dropoff_location": map[string]float64{"latitude": 37.7849, "longitude": -122.4094},
				"passenger_count":  3,
			},
			setupMock: func(m *MockPoolService, riderID uuid.UUID) {
				m.On("RequestPoolRide", mock.Anything, riderID, mock.AnythingOfType("*pool.RequestPoolRideRequest")).Return(createTestRequestPoolRideResponse(), nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockPoolService)
			mockRepo := new(MockRepository)
			handler := NewTestableHandler(mockService, mockRepo)

			riderID := uuid.New()
			if tt.setupMock != nil {
				tt.setupMock(mockService, riderID)
			}

			c, w := setupTestContext("POST", "/api/v1/pool/request", tt.body)
			setUserContext(c, riderID)

			handler.RequestPoolRide(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_UUIDValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		uuid           string
		expectedStatus int
	}{
		{
			name:           "invalid format - too short",
			uuid:           "123",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid format - wrong characters",
			uuid:           "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty uuid",
			uuid:           "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid uuid - dashes only",
			uuid:           "--------------------",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "uuid with extra chars",
			uuid:           "123456789-abcd-efgh-ijkl-mnopqrstuvwx",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockPoolService)
			mockRepo := new(MockRepository)
			handler := NewTestableHandler(mockService, mockRepo)

			riderID := uuid.New()

			c, w := setupTestContext("GET", "/api/v1/pool/"+tt.uuid, nil)
			c.Params = gin.Params{{Key: "id", Value: tt.uuid}}
			setUserContext(c, riderID)

			handler.GetPoolStatus(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_DriverEndpoints_AuthorizationCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		endpoint       string
		method         string
		handler        func(*TestableHandler) gin.HandlerFunc
		setupContext   func(*gin.Context, *gin.Params) uuid.UUID
		setupMock      func(*MockPoolService, uuid.UUID)
		expectedStatus int
	}{
		{
			name:     "driver start pool ride - authorized",
			endpoint: "/api/v1/driver/pool/:id/start",
			method:   "POST",
			handler: func(h *TestableHandler) gin.HandlerFunc {
				return h.StartPoolRide
			},
			setupContext: func(c *gin.Context, params *gin.Params) uuid.UUID {
				driverID := uuid.New()
				poolRideID := uuid.New()
				setUserContext(c, driverID)
				*params = gin.Params{{Key: "id", Value: poolRideID.String()}}
				return driverID
			},
			setupMock: func(m *MockPoolService, driverID uuid.UUID) {
				m.On("StartPoolRide", mock.Anything, driverID, mock.AnythingOfType("uuid.UUID")).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "driver start pool ride - unauthorized",
			endpoint: "/api/v1/driver/pool/:id/start",
			method:   "POST",
			handler: func(h *TestableHandler) gin.HandlerFunc {
				return h.StartPoolRide
			},
			setupContext: func(c *gin.Context, params *gin.Params) uuid.UUID {
				poolRideID := uuid.New()
				// Don't set user context
				*params = gin.Params{{Key: "id", Value: poolRideID.String()}}
				return uuid.Nil
			},
			setupMock:      nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:     "driver pickup passenger - authorized",
			endpoint: "/api/v1/driver/pool/passenger/:id/pickup",
			method:   "POST",
			handler: func(h *TestableHandler) gin.HandlerFunc {
				return h.PickupPassenger
			},
			setupContext: func(c *gin.Context, params *gin.Params) uuid.UUID {
				driverID := uuid.New()
				passengerID := uuid.New()
				setUserContext(c, driverID)
				*params = gin.Params{{Key: "id", Value: passengerID.String()}}
				return driverID
			},
			setupMock: func(m *MockPoolService, driverID uuid.UUID) {
				m.On("PickupPassenger", mock.Anything, driverID, mock.AnythingOfType("uuid.UUID")).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "driver dropoff passenger - authorized",
			endpoint: "/api/v1/driver/pool/passenger/:id/dropoff",
			method:   "POST",
			handler: func(h *TestableHandler) gin.HandlerFunc {
				return h.DropoffPassenger
			},
			setupContext: func(c *gin.Context, params *gin.Params) uuid.UUID {
				driverID := uuid.New()
				passengerID := uuid.New()
				setUserContext(c, driverID)
				*params = gin.Params{{Key: "id", Value: passengerID.String()}}
				return driverID
			},
			setupMock: func(m *MockPoolService, driverID uuid.UUID) {
				m.On("DropoffPassenger", mock.Anything, driverID, mock.AnythingOfType("uuid.UUID")).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockPoolService)
			mockRepo := new(MockRepository)
			handler := NewTestableHandler(mockService, mockRepo)

			c, w := setupTestContext(tt.method, tt.endpoint, nil)
			var params gin.Params
			driverID := tt.setupContext(c, &params)
			c.Params = params

			if tt.setupMock != nil {
				tt.setupMock(mockService, driverID)
			}

			handlerFunc := tt.handler(handler)
			handlerFunc(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_RiderEndpoints_AuthorizationCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		endpoint       string
		method         string
		handler        func(*TestableHandler) gin.HandlerFunc
		setupContext   func(*gin.Context, *gin.Params) uuid.UUID
		setupMock      func(*MockPoolService, *MockRepository, uuid.UUID)
		expectedStatus int
	}{
		{
			name:     "rider get pool status - authorized",
			endpoint: "/api/v1/pool/:id",
			method:   "GET",
			handler: func(h *TestableHandler) gin.HandlerFunc {
				return h.GetPoolStatus
			},
			setupContext: func(c *gin.Context, params *gin.Params) uuid.UUID {
				riderID := uuid.New()
				poolRideID := uuid.New()
				setUserContext(c, riderID)
				*params = gin.Params{{Key: "id", Value: poolRideID.String()}}
				return riderID
			},
			setupMock: func(m *MockPoolService, r *MockRepository, riderID uuid.UUID) {
				poolRide := createTestPoolRide()
				m.On("GetPoolStatus", mock.Anything, riderID, mock.AnythingOfType("uuid.UUID")).Return(createTestGetPoolStatusResponse(poolRide), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "rider cancel pool ride - authorized",
			endpoint: "/api/v1/pool/:id/cancel",
			method:   "POST",
			handler: func(h *TestableHandler) gin.HandlerFunc {
				return h.CancelPoolRide
			},
			setupContext: func(c *gin.Context, params *gin.Params) uuid.UUID {
				riderID := uuid.New()
				poolPassengerID := uuid.New()
				setUserContext(c, riderID)
				*params = gin.Params{{Key: "id", Value: poolPassengerID.String()}}
				return riderID
			},
			setupMock: func(m *MockPoolService, r *MockRepository, riderID uuid.UUID) {
				m.On("CancelPoolRide", mock.Anything, riderID, mock.AnythingOfType("uuid.UUID")).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "rider get active pool - unauthorized",
			endpoint: "/api/v1/pool/active",
			method:   "GET",
			handler: func(h *TestableHandler) gin.HandlerFunc {
				return h.GetActivePool
			},
			setupContext: func(c *gin.Context, params *gin.Params) uuid.UUID {
				// Don't set user context
				return uuid.Nil
			},
			setupMock:      nil,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockPoolService)
			mockRepo := new(MockRepository)
			handler := NewTestableHandler(mockService, mockRepo)

			c, w := setupTestContext(tt.method, tt.endpoint, nil)
			var params gin.Params
			riderID := tt.setupContext(c, &params)
			c.Params = params

			if tt.setupMock != nil {
				tt.setupMock(mockService, mockRepo, riderID)
			}

			handlerFunc := tt.handler(handler)
			handlerFunc(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_RequestPoolRide_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	expectedResponse := createTestRequestPoolRideResponse()

	reqBody := RequestPoolRideRequest{
		PickupLocation:  Location{Latitude: 37.7749, Longitude: -122.4194},
		DropoffLocation: Location{Latitude: 37.7849, Longitude: -122.4094},
	}

	mockService.On("RequestPoolRide", mock.Anything, riderID, &reqBody).Return(expectedResponse, nil)

	c, w := setupTestContext("POST", "/api/v1/pool/request", reqBody)
	setUserContext(c, riderID)

	handler.RequestPoolRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	// Verify response structure
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["pool_passenger_id"])
	assert.NotNil(t, data["status"])
	assert.NotNil(t, data["match_found"])
	assert.NotNil(t, data["original_fare"])
	assert.NotNil(t, data["pool_fare"])
	assert.NotNil(t, data["savings_percent"])

	mockService.AssertExpectations(t)
}

func TestHandler_GetPoolStatus_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	poolRideID := uuid.New()
	poolRide := createTestPoolRide()
	poolRide.ID = poolRideID
	expectedResponse := createTestGetPoolStatusResponse(poolRide)

	mockService.On("GetPoolStatus", mock.Anything, riderID, poolRideID).Return(expectedResponse, nil)

	c, w := setupTestContext("GET", "/api/v1/pool/"+poolRideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: poolRideID.String()}}
	setUserContext(c, riderID)

	handler.GetPoolStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["pool_ride"])
	assert.NotNil(t, data["my_status"])

	mockService.AssertExpectations(t)
}

func TestHandler_GetPoolStats_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	expectedStats := createTestPoolStatsResponse()

	mockService.On("GetPoolStats", mock.Anything).Return(expectedStats, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/pool/stats", nil)

	handler.GetPoolStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["total_pool_rides"])
	assert.NotNil(t, data["active_pool_rides"])
	assert.NotNil(t, data["avg_passengers_per_pool"])
	assert.NotNil(t, data["avg_savings_percent"])
	assert.NotNil(t, data["match_success_rate"])
	assert.NotNil(t, data["total_co2_saved_kg"])

	mockService.AssertExpectations(t)
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	poolRideID := uuid.New()

	mockService.On("GetPoolStatus", mock.Anything, riderID, poolRideID).Return(nil, common.NewNotFoundError("pool ride not found", nil))

	c, w := setupTestContext("GET", "/api/v1/pool/"+poolRideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: poolRideID.String()}}
	setUserContext(c, riderID)

	handler.GetPoolStatus(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	response := parseResponse(w)

	// Verify error response structure
	assert.False(t, response["success"].(bool))
	assert.NotNil(t, response["error"])

	errorInfo := response["error"].(map[string]interface{})
	assert.NotNil(t, errorInfo["code"])
	assert.NotNil(t, errorInfo["message"])

	mockService.AssertExpectations(t)
}

// ============================================================================
// Edge Cases and Error Handling Tests
// ============================================================================

func TestHandler_RequestPoolRide_EmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()

	// Handler passes to service, service validates and returns error for empty body
	mockService.On("RequestPoolRide", mock.Anything, mock.Anything, mock.Anything).Return(nil, common.NewAppError(http.StatusBadRequest, "invalid request", nil))

	c, w := setupTestContext("POST", "/api/v1/pool/request", map[string]interface{}{})
	setUserContext(c, riderID)

	handler.RequestPoolRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ConfirmPoolRide_MissingPassengerID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()

	reqBody := map[string]interface{}{
		"accept": true,
	}

	c, w := setupTestContext("POST", "/api/v1/pool/confirm", reqBody)
	setUserContext(c, riderID)

	handler.ConfirmPoolRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ConcurrentOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockPoolService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	riderID := uuid.New()
	expectedResponse := createTestRequestPoolRideResponse()

	reqBody := RequestPoolRideRequest{
		PickupLocation:  Location{Latitude: 37.7749, Longitude: -122.4194},
		DropoffLocation: Location{Latitude: 37.7849, Longitude: -122.4094},
	}

	// Allow multiple calls
	mockService.On("RequestPoolRide", mock.Anything, riderID, &reqBody).Return(expectedResponse, nil).Times(5)

	for i := 0; i < 5; i++ {
		c, w := setupTestContext("POST", "/api/v1/pool/request", reqBody)
		setUserContext(c, riderID)

		handler.RequestPoolRide(c)

		assert.Equal(t, http.StatusOK, w.Code)
	}

	mockService.AssertExpectations(t)
}

func TestHandler_LocationCoordinates_Boundaries(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		pickup         map[string]float64
		dropoff        map[string]float64
		expectedStatus int
		setupMock      bool
	}{
		{
			name:           "valid coordinates - San Francisco",
			pickup:         map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
			dropoff:        map[string]float64{"latitude": 37.7849, "longitude": -122.4094},
			expectedStatus: http.StatusOK,
			setupMock:      true,
		},
		{
			name:           "valid coordinates - equator",
			pickup:         map[string]float64{"latitude": 0.0, "longitude": 0.0},
			dropoff:        map[string]float64{"latitude": 0.1, "longitude": 0.1},
			expectedStatus: http.StatusOK,
			setupMock:      true,
		},
		{
			name:           "valid coordinates - extreme latitude",
			pickup:         map[string]float64{"latitude": 89.9, "longitude": 0.0},
			dropoff:        map[string]float64{"latitude": 89.8, "longitude": 0.1},
			expectedStatus: http.StatusOK,
			setupMock:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockPoolService)
			mockRepo := new(MockRepository)
			handler := NewTestableHandler(mockService, mockRepo)

			riderID := uuid.New()

			if tt.setupMock {
				mockService.On("RequestPoolRide", mock.Anything, riderID, mock.AnythingOfType("*pool.RequestPoolRideRequest")).Return(createTestRequestPoolRideResponse(), nil)
			}

			reqBody := map[string]interface{}{
				"pickup_location":  tt.pickup,
				"dropoff_location": tt.dropoff,
			}

			c, w := setupTestContext("POST", "/api/v1/pool/request", reqBody)
			setUserContext(c, riderID)

			handler.RequestPoolRide(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_PassengerCount_Boundaries(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		passengerCount int
		expectedStatus int
	}{
		{
			name:           "single passenger",
			passengerCount: 1,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "two passengers",
			passengerCount: 2,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "three passengers",
			passengerCount: 3,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "four passengers",
			passengerCount: 4,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockPoolService)
			mockRepo := new(MockRepository)
			handler := NewTestableHandler(mockService, mockRepo)

			riderID := uuid.New()

			mockService.On("RequestPoolRide", mock.Anything, riderID, mock.AnythingOfType("*pool.RequestPoolRideRequest")).Return(createTestRequestPoolRideResponse(), nil)

			reqBody := map[string]interface{}{
				"pickup_location":  map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
				"dropoff_location": map[string]float64{"latitude": 37.7849, "longitude": -122.4094},
				"passenger_count":  tt.passengerCount,
			}

			c, w := setupTestContext("POST", "/api/v1/pool/request", reqBody)
			setUserContext(c, riderID)

			handler.RequestPoolRide(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
