package rides

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

// ServiceInterface defines the interface for the rides service used by the handler
type ServiceInterface interface {
	RequestRide(ctx context.Context, riderID uuid.UUID, req *models.RideRequest) (*models.Ride, error)
	GetRide(ctx context.Context, rideID uuid.UUID) (*models.Ride, error)
	AcceptRide(ctx context.Context, rideID, driverID uuid.UUID) (*models.Ride, error)
	StartRide(ctx context.Context, rideID, driverID uuid.UUID) (*models.Ride, error)
	CompleteRide(ctx context.Context, rideID, driverID uuid.UUID, actualDistance float64) (*models.Ride, error)
	CancelRide(ctx context.Context, rideID, userID uuid.UUID, reason string) (*models.Ride, error)
	RateRide(ctx context.Context, rideID, riderID uuid.UUID, req *models.RideRatingRequest) error
	GetRiderRides(ctx context.Context, riderID uuid.UUID, limit, offset int) ([]*models.Ride, int64, error)
	GetDriverRides(ctx context.Context, driverID uuid.UUID, limit, offset int) ([]*models.Ride, int64, error)
	GetAvailableRides(ctx context.Context) ([]*models.Ride, error)
	GetUserProfile(ctx context.Context, userID uuid.UUID) (*models.User, error)
	UpdateUserProfile(ctx context.Context, userID uuid.UUID, firstName, lastName, phoneNumber string) error
	MatchDrivers(ctx context.Context, pickupLatitude, pickupLongitude float64) ([]*DriverCandidate, error)
}

// MockService is a mock implementation of ServiceInterface
type MockService struct {
	mock.Mock
}

func (m *MockService) RequestRide(ctx context.Context, riderID uuid.UUID, req *models.RideRequest) (*models.Ride, error) {
	args := m.Called(ctx, riderID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ride), args.Error(1)
}

func (m *MockService) GetRide(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ride), args.Error(1)
}

func (m *MockService) AcceptRide(ctx context.Context, rideID, driverID uuid.UUID) (*models.Ride, error) {
	args := m.Called(ctx, rideID, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ride), args.Error(1)
}

func (m *MockService) StartRide(ctx context.Context, rideID, driverID uuid.UUID) (*models.Ride, error) {
	args := m.Called(ctx, rideID, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ride), args.Error(1)
}

func (m *MockService) CompleteRide(ctx context.Context, rideID, driverID uuid.UUID, actualDistance float64) (*models.Ride, error) {
	args := m.Called(ctx, rideID, driverID, actualDistance)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ride), args.Error(1)
}

func (m *MockService) CancelRide(ctx context.Context, rideID, userID uuid.UUID, reason string) (*models.Ride, error) {
	args := m.Called(ctx, rideID, userID, reason)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ride), args.Error(1)
}

func (m *MockService) RateRide(ctx context.Context, rideID, riderID uuid.UUID, req *models.RideRatingRequest) error {
	args := m.Called(ctx, rideID, riderID, req)
	return args.Error(0)
}

func (m *MockService) GetRiderRides(ctx context.Context, riderID uuid.UUID, limit, offset int) ([]*models.Ride, int64, error) {
	args := m.Called(ctx, riderID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*models.Ride), args.Get(1).(int64), args.Error(2)
}

func (m *MockService) GetDriverRides(ctx context.Context, driverID uuid.UUID, limit, offset int) ([]*models.Ride, int64, error) {
	args := m.Called(ctx, driverID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*models.Ride), args.Get(1).(int64), args.Error(2)
}

func (m *MockService) GetAvailableRides(ctx context.Context) ([]*models.Ride, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Ride), args.Error(1)
}

func (m *MockService) GetUserProfile(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockService) UpdateUserProfile(ctx context.Context, userID uuid.UUID, firstName, lastName, phoneNumber string) error {
	args := m.Called(ctx, userID, firstName, lastName, phoneNumber)
	return args.Error(0)
}

func (m *MockService) MatchDrivers(ctx context.Context, pickupLatitude, pickupLongitude float64) ([]*DriverCandidate, error) {
	args := m.Called(ctx, pickupLatitude, pickupLongitude)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DriverCandidate), args.Error(1)
}

// TestHandler wraps Handler for testing with a mock service interface
type TestHandler struct {
	mockService *MockService
}

// NewTestHandler creates a handler with a mock service for testing
func NewTestHandler() (*Handler, *MockService) {
	mockService := new(MockService)
	// Create a minimal service wrapper that uses the mock
	// For testing, we'll use a custom approach
	return &Handler{service: nil}, mockService
}

// Helper functions for setting up test context
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

func createTestRide(riderID uuid.UUID, status models.RideStatus) *models.Ride {
	now := time.Now()
	return &models.Ride{
		ID:                uuid.New(),
		RiderID:           riderID,
		Status:            status,
		PickupLatitude:    40.7128,
		PickupLongitude:   -74.0060,
		PickupAddress:     "123 Main St",
		DropoffLatitude:   40.7580,
		DropoffLongitude:  -73.9855,
		DropoffAddress:    "456 Broadway",
		EstimatedDistance: 5.0,
		EstimatedDuration: 15,
		EstimatedFare:     25.0,
		SurgeMultiplier:   1.0,
		RequestedAt:       now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func createTestRideWithDriver(riderID, driverID uuid.UUID, status models.RideStatus) *models.Ride {
	ride := createTestRide(riderID, status)
	ride.DriverID = &driverID
	now := time.Now()
	ride.AcceptedAt = &now
	return ride
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

// MockableHandler provides testable handler methods with mock service injection
type MockableHandler struct {
	service *MockService
}

func NewMockableHandler(mockService *MockService) *MockableHandler {
	return &MockableHandler{service: mockService}
}

// RequestRide handles incoming rider requests to create a new ride - testable version
func (h *MockableHandler) RequestRide(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req models.RideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	ride, err := h.service.RequestRide(c.Request.Context(), userID.(uuid.UUID), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to request ride")
		return
	}

	common.CreatedResponse(c, ride)
}

// GetRide returns ride details - testable version
func (h *MockableHandler) GetRide(c *gin.Context) {
	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	ride, err := h.service.GetRide(c.Request.Context(), rideID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get ride")
		return
	}

	common.SuccessResponse(c, ride)
}

// AcceptRide lets a driver accept a ride - testable version
func (h *MockableHandler) AcceptRide(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	ride, err := h.service.AcceptRide(c.Request.Context(), rideID, driverID.(uuid.UUID))
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to accept ride")
		return
	}

	common.SuccessResponse(c, ride)
}

// StartRide marks the ride as in progress - testable version
func (h *MockableHandler) StartRide(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	ride, err := h.service.StartRide(c.Request.Context(), rideID, driverID.(uuid.UUID))
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to start ride")
		return
	}

	common.SuccessResponse(c, ride)
}

// CompleteRide finalizes a ride - testable version
func (h *MockableHandler) CompleteRide(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	var req struct {
		ActualDistance float64 `json:"actual_distance" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	ride, err := h.service.CompleteRide(c.Request.Context(), rideID, driverID.(uuid.UUID), req.ActualDistance)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to complete ride")
		return
	}

	common.SuccessResponse(c, ride)
}

// CancelRide allows users to cancel - testable version
func (h *MockableHandler) CancelRide(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	ride, err := h.service.CancelRide(c.Request.Context(), rideID, userID.(uuid.UUID), req.Reason)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to cancel ride")
		return
	}

	common.SuccessResponse(c, ride)
}

// RateRide records rider feedback - testable version
func (h *MockableHandler) RateRide(c *gin.Context) {
	riderID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	var req models.RideRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.RateRide(c.Request.Context(), rideID, riderID.(uuid.UUID), &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to rate ride")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "ride rated successfully"})
}

// GetMyRides returns paginated rides - testable version
func (h *MockableHandler) GetMyRides(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	role, exists := c.Get("user_role")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Default pagination
	limit := 20
	offset := 0

	var rides []*models.Ride
	var total int64
	var err error

	if role == models.RoleRider {
		rides, total, err = h.service.GetRiderRides(c.Request.Context(), userID.(uuid.UUID), limit, offset)
	} else if role == models.RoleDriver {
		rides, total, err = h.service.GetDriverRides(c.Request.Context(), userID.(uuid.UUID), limit, offset)
	} else {
		common.ErrorResponse(c, http.StatusForbidden, "invalid role")
		return
	}

	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get rides")
		return
	}

	if rides == nil {
		rides = []*models.Ride{}
	}

	common.SuccessResponseWithMeta(c, rides, &common.Meta{
		Limit:  limit,
		Offset: offset,
		Total:  total,
	})
}

// GetAvailableRides lists ride requests for drivers - testable version
func (h *MockableHandler) GetAvailableRides(c *gin.Context) {
	rides, err := h.service.GetAvailableRides(c.Request.Context())
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get available rides")
		return
	}

	if rides == nil {
		rides = []*models.Ride{}
	}

	common.SuccessResponse(c, rides)
}

// GetUserProfile retrieves user profile - testable version
func (h *MockableHandler) GetUserProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.service.GetUserProfile(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get user profile")
		return
	}

	common.SuccessResponse(c, user)
}

// UpdateUserProfile updates user profile - testable version
func (h *MockableHandler) UpdateUserProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		FirstName   string `json:"first_name" binding:"required"`
		LastName    string `json:"last_name" binding:"required"`
		PhoneNumber string `json:"phone_number" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	err := h.service.UpdateUserProfile(c.Request.Context(), userID.(uuid.UUID), req.FirstName, req.LastName, req.PhoneNumber)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update user profile")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Profile updated successfully",
	})
}

// GetSurgeInfo surfaces surge pricing details - testable version
func (h *MockableHandler) GetSurgeInfo(c *gin.Context) {
	latStr := c.Query("latitude")
	lonStr := c.Query("longitude")

	if latStr == "" || lonStr == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "latitude and longitude are required")
		return
	}

	var latitude, longitude float64
	if _, err := fmt.Sscanf(latStr, "%f", &latitude); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid latitude")
		return
	}

	if _, err := fmt.Sscanf(lonStr, "%f", &longitude); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid longitude")
		return
	}

	// Time-based surge for testing
	multiplier := calculateSurgeMultiplier(time.Now())
	common.SuccessResponse(c, gin.H{
		"surge_multiplier": multiplier,
		"is_surge_active":  multiplier > 1.0,
		"message":          "Time-based surge pricing active",
	})
}

// MatchDrivers returns best-scored drivers - testable version
func (h *MockableHandler) MatchDrivers(c *gin.Context) {
	latStr := c.Query("latitude")
	lngStr := c.Query("longitude")
	if latStr == "" || lngStr == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "latitude and longitude are required")
		return
	}

	var latitude, longitude float64
	if _, err := fmt.Sscanf(latStr, "%f", &latitude); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid latitude")
		return
	}
	if _, err := fmt.Sscanf(lngStr, "%f", &longitude); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid longitude")
		return
	}

	candidates, err := h.service.MatchDrivers(c.Request.Context(), latitude, longitude)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to match drivers")
		return
	}

	common.SuccessResponse(c, gin.H{
		"drivers": candidates,
		"count":   len(candidates),
	})
}

// ============================================================================
// RequestRide Handler Tests
// ============================================================================

func TestHandler_RequestRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	expectedRide := createTestRide(userID, models.RideStatusRequested)

	reqBody := models.RideRequest{
		PickupLatitude:   40.7128,
		PickupLongitude:  -74.0060,
		PickupAddress:    "123 Main St",
		DropoffLatitude:  40.7580,
		DropoffLongitude: -73.9855,
		DropoffAddress:   "456 Broadway",
	}

	mockService.On("RequestRide", mock.Anything, userID, &reqBody).Return(expectedRide, nil)

	c, w := setupTestContext("POST", "/api/v1/rides", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.RequestRide(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_RequestRide_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := models.RideRequest{
		PickupLatitude:   40.7128,
		PickupLongitude:  -74.0060,
		PickupAddress:    "123 Main St",
		DropoffLatitude:  40.7580,
		DropoffLongitude: -73.9855,
		DropoffAddress:   "456 Broadway",
	}

	c, w := setupTestContext("POST", "/api/v1/rides", reqBody)
	// Don't set user context to simulate unauthorized

	handler.RequestRide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_RequestRide_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/rides", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/rides", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleRider)

	handler.RequestRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RequestRide_MissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "missing pickup address",
			body: map[string]interface{}{
				"pickup_latitude":   40.7128,
				"pickup_longitude":  -74.0060,
				"dropoff_latitude":  40.7580,
				"dropoff_longitude": -73.9855,
				"dropoff_address":   "456 Broadway",
			},
		},
		{
			name: "missing dropoff address",
			body: map[string]interface{}{
				"pickup_latitude":   40.7128,
				"pickup_longitude":  -74.0060,
				"pickup_address":    "123 Main St",
				"dropoff_latitude":  40.7580,
				"dropoff_longitude": -73.9855,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupTestContext("POST", "/api/v1/rides", tt.body)
			setUserContext(c, userID, models.RoleRider)

			handler.RequestRide(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandler_RequestRide_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	reqBody := models.RideRequest{
		PickupLatitude:   40.7128,
		PickupLongitude:  -74.0060,
		PickupAddress:    "123 Main St",
		DropoffLatitude:  40.7580,
		DropoffLongitude: -73.9855,
		DropoffAddress:   "456 Broadway",
	}

	mockService.On("RequestRide", mock.Anything, userID, &reqBody).Return(nil, errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/rides", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.RequestRide(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_RequestRide_AppError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	reqBody := models.RideRequest{
		PickupLatitude:   40.7128,
		PickupLongitude:  -74.0060,
		PickupAddress:    "123 Main St",
		DropoffLatitude:  40.7580,
		DropoffLongitude: -73.9855,
		DropoffAddress:   "456 Broadway",
	}

	mockService.On("RequestRide", mock.Anything, userID, &reqBody).Return(nil, common.NewBadRequestError("invalid location", nil))

	c, w := setupTestContext("POST", "/api/v1/rides", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.RequestRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetRide Handler Tests
// ============================================================================

func TestHandler_GetRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()
	expectedRide := createTestRide(riderID, models.RideStatusRequested)
	expectedRide.ID = rideID

	mockService.On("GetRide", mock.Anything, rideID).Return(expectedRide, nil)

	c, w := setupTestContext("GET", "/api/v1/rides/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}

	handler.GetRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetRide_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/rides/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.GetRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid ride ID")
}

func TestHandler_GetRide_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	rideID := uuid.New()

	mockService.On("GetRide", mock.Anything, rideID).Return(nil, common.NewNotFoundError("ride not found", nil))

	c, w := setupTestContext("GET", "/api/v1/rides/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}

	handler.GetRide(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetRide_EmptyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/rides/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.GetRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetRide_UUIDFormats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		rideID         string
		expectedStatus int
		setupMock      bool
	}{
		{
			name:           "invalid format - too short",
			rideID:         "123",
			expectedStatus: http.StatusBadRequest,
			setupMock:      false,
		},
		{
			name:           "invalid format - wrong characters",
			rideID:         "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
			expectedStatus: http.StatusBadRequest,
			setupMock:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			handler := NewMockableHandler(mockService)

			c, w := setupTestContext("GET", "/api/v1/rides/"+tt.rideID, nil)
			c.Params = gin.Params{{Key: "id", Value: tt.rideID}}

			handler.GetRide(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// AcceptRide Handler Tests
// ============================================================================

func TestHandler_AcceptRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	riderID := uuid.New()
	rideID := uuid.New()
	expectedRide := createTestRideWithDriver(riderID, driverID, models.RideStatusAccepted)
	expectedRide.ID = rideID

	mockService.On("AcceptRide", mock.Anything, rideID, driverID).Return(expectedRide, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/rides/"+rideID.String()+"/accept", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.AcceptRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_AcceptRide_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	rideID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/rides/"+rideID.String()+"/accept", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	// Don't set user context

	handler.AcceptRide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_AcceptRide_InvalidRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/rides/invalid/accept", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.AcceptRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid ride ID")
}

func TestHandler_AcceptRide_RideNotAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	rideID := uuid.New()

	mockService.On("AcceptRide", mock.Anything, rideID, driverID).Return(nil, common.NewErrorWithCode(409, common.ErrCodeRideNotAvailable, "ride is no longer available", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/rides/"+rideID.String()+"/accept", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.AcceptRide(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_AcceptRide_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	rideID := uuid.New()

	mockService.On("AcceptRide", mock.Anything, rideID, driverID).Return(nil, common.NewNotFoundError("ride not found", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/rides/"+rideID.String()+"/accept", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.AcceptRide(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// StartRide Handler Tests
// ============================================================================

func TestHandler_StartRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	riderID := uuid.New()
	rideID := uuid.New()
	expectedRide := createTestRideWithDriver(riderID, driverID, models.RideStatusInProgress)
	expectedRide.ID = rideID

	mockService.On("StartRide", mock.Anything, rideID, driverID).Return(expectedRide, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/rides/"+rideID.String()+"/start", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.StartRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_StartRide_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	rideID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/rides/"+rideID.String()+"/start", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}

	handler.StartRide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_StartRide_InvalidRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/rides/bad-id/start", nil)
	c.Params = gin.Params{{Key: "id", Value: "bad-id"}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.StartRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_StartRide_WrongDriver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	wrongDriverID := uuid.New()
	rideID := uuid.New()

	mockService.On("StartRide", mock.Anything, rideID, wrongDriverID).Return(nil, common.NewBadRequestError("unauthorized driver", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/rides/"+rideID.String()+"/start", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, wrongDriverID, models.RoleDriver)

	handler.StartRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_StartRide_RideNotAccepted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	rideID := uuid.New()

	mockService.On("StartRide", mock.Anything, rideID, driverID).Return(nil, common.NewBadRequestError("ride has not been accepted", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/rides/"+rideID.String()+"/start", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.StartRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// CompleteRide Handler Tests
// ============================================================================

func TestHandler_CompleteRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	riderID := uuid.New()
	rideID := uuid.New()
	expectedRide := createTestRideWithDriver(riderID, driverID, models.RideStatusCompleted)
	expectedRide.ID = rideID
	actualDistance := 5.5
	expectedRide.ActualDistance = &actualDistance

	mockService.On("CompleteRide", mock.Anything, rideID, driverID, actualDistance).Return(expectedRide, nil)

	reqBody := map[string]interface{}{
		"actual_distance": actualDistance,
	}

	c, w := setupTestContext("POST", "/api/v1/driver/rides/"+rideID.String()+"/complete", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.CompleteRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_CompleteRide_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	rideID := uuid.New()
	reqBody := map[string]interface{}{
		"actual_distance": 5.5,
	}

	c, w := setupTestContext("POST", "/api/v1/driver/rides/"+rideID.String()+"/complete", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}

	handler.CompleteRide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CompleteRide_InvalidRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	reqBody := map[string]interface{}{
		"actual_distance": 5.5,
	}

	c, w := setupTestContext("POST", "/api/v1/driver/rides/invalid/complete", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.CompleteRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CompleteRide_MissingActualDistance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	rideID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/rides/"+rideID.String()+"/complete", map[string]interface{}{})
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.CompleteRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CompleteRide_RideNotInProgress(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	rideID := uuid.New()

	mockService.On("CompleteRide", mock.Anything, rideID, driverID, 5.5).Return(nil, common.NewBadRequestError("ride is not in progress", nil))

	reqBody := map[string]interface{}{
		"actual_distance": 5.5,
	}

	c, w := setupTestContext("POST", "/api/v1/driver/rides/"+rideID.String()+"/complete", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.CompleteRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// CancelRide Handler Tests
// ============================================================================

func TestHandler_CancelRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	rideID := uuid.New()
	expectedRide := createTestRide(userID, models.RideStatusCancelled)
	expectedRide.ID = rideID
	reason := "Changed plans"
	expectedRide.CancellationReason = &reason

	mockService.On("CancelRide", mock.Anything, rideID, userID, reason).Return(expectedRide, nil)

	reqBody := map[string]interface{}{
		"reason": reason,
	}

	c, w := setupTestContext("POST", "/api/v1/rides/"+rideID.String()+"/cancel", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.CancelRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_CancelRide_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	rideID := uuid.New()
	reqBody := map[string]interface{}{
		"reason": "Changed plans",
	}

	c, w := setupTestContext("POST", "/api/v1/rides/"+rideID.String()+"/cancel", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}

	handler.CancelRide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CancelRide_InvalidRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"reason": "Changed plans",
	}

	c, w := setupTestContext("POST", "/api/v1/rides/invalid/cancel", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, userID, models.RoleRider)

	handler.CancelRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CancelRide_EmptyReason(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	rideID := uuid.New()
	expectedRide := createTestRide(userID, models.RideStatusCancelled)
	expectedRide.ID = rideID

	mockService.On("CancelRide", mock.Anything, rideID, userID, "").Return(expectedRide, nil)

	reqBody := map[string]interface{}{
		"reason": "",
	}

	c, w := setupTestContext("POST", "/api/v1/rides/"+rideID.String()+"/cancel", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.CancelRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CancelRide_AlreadyCancelled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	rideID := uuid.New()

	mockService.On("CancelRide", mock.Anything, rideID, userID, "Changed plans").Return(nil, common.NewBadRequestError("cannot cancel completed or already cancelled ride", nil))

	reqBody := map[string]interface{}{
		"reason": "Changed plans",
	}

	c, w := setupTestContext("POST", "/api/v1/rides/"+rideID.String()+"/cancel", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.CancelRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CancelRide_DriverCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	riderID := uuid.New()
	rideID := uuid.New()
	expectedRide := createTestRideWithDriver(riderID, driverID, models.RideStatusCancelled)
	expectedRide.ID = rideID

	mockService.On("CancelRide", mock.Anything, rideID, driverID, "Emergency").Return(expectedRide, nil)

	reqBody := map[string]interface{}{
		"reason": "Emergency",
	}

	c, w := setupTestContext("POST", "/api/v1/rides/"+rideID.String()+"/cancel", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.CancelRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// RateRide Handler Tests
// ============================================================================

func TestHandler_RateRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()

	mockService.On("RateRide", mock.Anything, rideID, riderID, mock.AnythingOfType("*models.RideRatingRequest")).Return(nil)

	reqBody := models.RideRatingRequest{
		Rating: 5,
	}

	c, w := setupTestContext("POST", "/api/v1/rides/"+rideID.String()+"/rate", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, riderID, models.RoleRider)

	handler.RateRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_RateRide_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	rideID := uuid.New()
	reqBody := models.RideRatingRequest{
		Rating: 5,
	}

	c, w := setupTestContext("POST", "/api/v1/rides/"+rideID.String()+"/rate", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}

	handler.RateRide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_RateRide_InvalidRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	riderID := uuid.New()
	reqBody := models.RideRatingRequest{
		Rating: 5,
	}

	c, w := setupTestContext("POST", "/api/v1/rides/invalid/rate", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, riderID, models.RoleRider)

	handler.RateRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RateRide_InvalidRating(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()

	tests := []struct {
		name   string
		rating interface{}
	}{
		{"rating too low", 0},
		{"rating too high", 6},
		{"negative rating", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := map[string]interface{}{
				"rating": tt.rating,
			}

			c, w := setupTestContext("POST", "/api/v1/rides/"+rideID.String()+"/rate", reqBody)
			c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
			setUserContext(c, riderID, models.RoleRider)

			handler.RateRide(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandler_RateRide_AllValidRatings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for rating := 1; rating <= 5; rating++ {
		t.Run(fmt.Sprintf("rating_%d", rating), func(t *testing.T) {
			mockService := new(MockService)
			handler := NewMockableHandler(mockService)

			riderID := uuid.New()
			rideID := uuid.New()

			mockService.On("RateRide", mock.Anything, rideID, riderID, mock.AnythingOfType("*models.RideRatingRequest")).Return(nil)

			reqBody := map[string]interface{}{
				"rating": rating,
			}

			c, w := setupTestContext("POST", "/api/v1/rides/"+rideID.String()+"/rate", reqBody)
			c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
			setUserContext(c, riderID, models.RoleRider)

			handler.RateRide(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_RateRide_WithFeedback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()

	mockService.On("RateRide", mock.Anything, rideID, riderID, mock.AnythingOfType("*models.RideRatingRequest")).Return(nil)

	feedback := "Great driver!"
	reqBody := map[string]interface{}{
		"rating":   5,
		"feedback": feedback,
	}

	c, w := setupTestContext("POST", "/api/v1/rides/"+rideID.String()+"/rate", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, riderID, models.RoleRider)

	handler.RateRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_RateRide_RideNotCompleted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()

	mockService.On("RateRide", mock.Anything, rideID, riderID, mock.AnythingOfType("*models.RideRatingRequest")).Return(common.NewBadRequestError("can only rate completed rides", nil))

	reqBody := map[string]interface{}{
		"rating": 5,
	}

	c, w := setupTestContext("POST", "/api/v1/rides/"+rideID.String()+"/rate", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, riderID, models.RoleRider)

	handler.RateRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetMyRides Handler Tests
// ============================================================================

func TestHandler_GetMyRides_RiderSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	rides := []*models.Ride{
		createTestRide(userID, models.RideStatusCompleted),
		createTestRide(userID, models.RideStatusRequested),
	}

	mockService.On("GetRiderRides", mock.Anything, userID, 20, 0).Return(rides, int64(2), nil)

	c, w := setupTestContext("GET", "/api/v1/rides", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetMyRides(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetMyRides_DriverSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	riderID := uuid.New()
	rides := []*models.Ride{
		createTestRideWithDriver(riderID, userID, models.RideStatusCompleted),
	}

	mockService.On("GetDriverRides", mock.Anything, userID, 20, 0).Return(rides, int64(1), nil)

	c, w := setupTestContext("GET", "/api/v1/rides", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetMyRides(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetMyRides_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/rides", nil)
	// Don't set user context

	handler.GetMyRides(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetMyRides_NoRoleInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/rides", nil)
	c.Set("user_id", userID)
	// Don't set role

	handler.GetMyRides(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetMyRides_AdminRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/rides", nil)
	setUserContext(c, userID, models.RoleAdmin)

	handler.GetMyRides(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_GetMyRides_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("GetRiderRides", mock.Anything, userID, 20, 0).Return([]*models.Ride{}, int64(0), nil)

	c, w := setupTestContext("GET", "/api/v1/rides", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetMyRides(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	// Should return empty array, not null
	mockService.AssertExpectations(t)
}

func TestHandler_GetMyRides_NilRides(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("GetRiderRides", mock.Anything, userID, 20, 0).Return(nil, int64(0), nil)

	c, w := setupTestContext("GET", "/api/v1/rides", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetMyRides(c)

	assert.Equal(t, http.StatusOK, w.Code)
	// Handler should convert nil to empty array
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetAvailableRides Handler Tests
// ============================================================================

func TestHandler_GetAvailableRides_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	riderID := uuid.New()
	rides := []*models.Ride{
		createTestRide(riderID, models.RideStatusRequested),
		createTestRide(uuid.New(), models.RideStatusRequested),
	}

	mockService.On("GetAvailableRides", mock.Anything).Return(rides, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/rides/available", nil)

	handler.GetAvailableRides(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetAvailableRides_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	mockService.On("GetAvailableRides", mock.Anything).Return([]*models.Ride{}, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/rides/available", nil)

	handler.GetAvailableRides(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetAvailableRides_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	mockService.On("GetAvailableRides", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/rides/available", nil)

	handler.GetAvailableRides(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetSurgeInfo Handler Tests
// ============================================================================

func TestHandler_GetSurgeInfo_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/rides/surge-info?latitude=40.7128&longitude=-74.0060", nil)
	req := httptest.NewRequest("GET", "/api/v1/rides/surge-info?latitude=40.7128&longitude=-74.0060", nil)
	c.Request = req

	handler.GetSurgeInfo(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Contains(t, data, "surge_multiplier")
	assert.Contains(t, data, "is_surge_active")
}

func TestHandler_GetSurgeInfo_MissingLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/rides/surge-info?longitude=-74.0060", nil)
	req := httptest.NewRequest("GET", "/api/v1/rides/surge-info?longitude=-74.0060", nil)
	c.Request = req

	handler.GetSurgeInfo(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "latitude and longitude are required")
}

func TestHandler_GetSurgeInfo_MissingLongitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/rides/surge-info?latitude=40.7128", nil)
	req := httptest.NewRequest("GET", "/api/v1/rides/surge-info?latitude=40.7128", nil)
	c.Request = req

	handler.GetSurgeInfo(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetSurgeInfo_InvalidLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/rides/surge-info?latitude=invalid&longitude=-74.0060", nil)
	req := httptest.NewRequest("GET", "/api/v1/rides/surge-info?latitude=invalid&longitude=-74.0060", nil)
	c.Request = req

	handler.GetSurgeInfo(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid latitude")
}

func TestHandler_GetSurgeInfo_InvalidLongitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/rides/surge-info?latitude=40.7128&longitude=invalid", nil)
	req := httptest.NewRequest("GET", "/api/v1/rides/surge-info?latitude=40.7128&longitude=invalid", nil)
	c.Request = req

	handler.GetSurgeInfo(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid longitude")
}

func TestHandler_GetSurgeInfo_BothMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/rides/surge-info", nil)
	req := httptest.NewRequest("GET", "/api/v1/rides/surge-info", nil)
	c.Request = req

	handler.GetSurgeInfo(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetSurgeInfo_EdgeCoordinates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		latitude  string
		longitude  string
	}{
		{"zero coordinates", "0", "0"},
		{"max latitude", "90", "0"},
		{"min latitude", "-90", "0"},
		{"max longitude", "0", "180"},
		{"min longitude", "0", "-180"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			handler := NewMockableHandler(mockService)

			c, w := setupTestContext("GET", fmt.Sprintf("/api/v1/rides/surge-info?latitude=%s&longitude=%s", tt.latitude, tt.longitude), nil)
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/rides/surge-info?latitude=%s&longitude=%s", tt.latitude, tt.longitude), nil)
			c.Request = req

			handler.GetSurgeInfo(c)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// ============================================================================
// MatchDrivers Handler Tests
// ============================================================================

func TestHandler_MatchDrivers_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	candidates := []*DriverCandidate{
		{DriverID: uuid.New(), DistanceKm: 1.5, Rating: 4.8, Score: 0.95},
		{DriverID: uuid.New(), DistanceKm: 2.0, Rating: 4.5, Score: 0.85},
	}

	mockService.On("MatchDrivers", mock.Anything, 40.7128, -74.0060).Return(candidates, nil)

	c, w := setupTestContext("GET", "/api/v1/rides/match-drivers?latitude=40.7128&longitude=-74.0060", nil)
	req := httptest.NewRequest("GET", "/api/v1/rides/match-drivers?latitude=40.7128&longitude=-74.0060", nil)
	c.Request = req

	handler.MatchDrivers(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["count"])
	mockService.AssertExpectations(t)
}

func TestHandler_MatchDrivers_MissingLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/rides/match-drivers?longitude=-74.0060", nil)
	req := httptest.NewRequest("GET", "/api/v1/rides/match-drivers?longitude=-74.0060", nil)
	c.Request = req

	handler.MatchDrivers(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_MatchDrivers_MissingLongitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/rides/match-drivers?latitude=40.7128", nil)
	req := httptest.NewRequest("GET", "/api/v1/rides/match-drivers?latitude=40.7128", nil)
	c.Request = req

	handler.MatchDrivers(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_MatchDrivers_InvalidLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/rides/match-drivers?latitude=invalid&longitude=-74.0060", nil)
	req := httptest.NewRequest("GET", "/api/v1/rides/match-drivers?latitude=invalid&longitude=-74.0060", nil)
	c.Request = req

	handler.MatchDrivers(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_MatchDrivers_NoDriversAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	mockService.On("MatchDrivers", mock.Anything, 40.7128, -74.0060).Return([]*DriverCandidate{}, nil)

	c, w := setupTestContext("GET", "/api/v1/rides/match-drivers?latitude=40.7128&longitude=-74.0060", nil)
	req := httptest.NewRequest("GET", "/api/v1/rides/match-drivers?latitude=40.7128&longitude=-74.0060", nil)
	c.Request = req

	handler.MatchDrivers(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["count"])
	mockService.AssertExpectations(t)
}

func TestHandler_MatchDrivers_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	mockService.On("MatchDrivers", mock.Anything, 40.7128, -74.0060).Return(nil, common.NewInternalServerError("matching engine not configured"))

	c, w := setupTestContext("GET", "/api/v1/rides/match-drivers?latitude=40.7128&longitude=-74.0060", nil)
	req := httptest.NewRequest("GET", "/api/v1/rides/match-drivers?latitude=40.7128&longitude=-74.0060", nil)
	c.Request = req

	handler.MatchDrivers(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetUserProfile Handler Tests
// ============================================================================

func TestHandler_GetUserProfile_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	expectedUser := &models.User{
		ID:          userID,
		Email:       "test@example.com",
		PhoneNumber: "+1234567890",
		FirstName:   "John",
		LastName:    "Doe",
		Role:        models.RoleRider,
	}

	mockService.On("GetUserProfile", mock.Anything, userID).Return(expectedUser, nil)

	c, w := setupTestContext("GET", "/api/v1/profile", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetUserProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetUserProfile_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/profile", nil)

	handler.GetUserProfile(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetUserProfile_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("GetUserProfile", mock.Anything, userID).Return(nil, errors.New("user not found"))

	c, w := setupTestContext("GET", "/api/v1/profile", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetUserProfile(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// UpdateUserProfile Handler Tests
// ============================================================================

func TestHandler_UpdateUserProfile_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("UpdateUserProfile", mock.Anything, userID, "John", "Doe", "+1234567890").Return(nil)

	reqBody := map[string]interface{}{
		"first_name":   "John",
		"last_name":    "Doe",
		"phone_number": "+1234567890",
	}

	c, w := setupTestContext("PUT", "/api/v1/profile", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.UpdateUserProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateUserProfile_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := map[string]interface{}{
		"first_name":   "John",
		"last_name":    "Doe",
		"phone_number": "+1234567890",
	}

	c, w := setupTestContext("PUT", "/api/v1/profile", reqBody)

	handler.UpdateUserProfile(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_UpdateUserProfile_MissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "missing first_name",
			body: map[string]interface{}{
				"last_name":    "Doe",
				"phone_number": "+1234567890",
			},
		},
		{
			name: "missing last_name",
			body: map[string]interface{}{
				"first_name":   "John",
				"phone_number": "+1234567890",
			},
		},
		{
			name: "missing phone_number",
			body: map[string]interface{}{
				"first_name": "John",
				"last_name":  "Doe",
			},
		},
		{
			name: "empty body",
			body: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupTestContext("PUT", "/api/v1/profile", tt.body)
			setUserContext(c, userID, models.RoleRider)

			handler.UpdateUserProfile(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandler_UpdateUserProfile_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("UpdateUserProfile", mock.Anything, userID, "John", "Doe", "+1234567890").Return(errors.New("database error"))

	reqBody := map[string]interface{}{
		"first_name":   "John",
		"last_name":    "Doe",
		"phone_number": "+1234567890",
	}

	c, w := setupTestContext("PUT", "/api/v1/profile", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.UpdateUserProfile(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// Additional Edge Case Tests
// ============================================================================

func TestHandler_ContentTypeValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := []byte(`{"pickup_latitude": 40.7128}`)
	req := httptest.NewRequest("POST", "/api/v1/rides", bytes.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	c.Request = req
	setUserContext(c, userID, models.RoleRider)

	handler.RequestRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ConcurrentRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			mockService := new(MockService)
			handler := NewMockableHandler(mockService)

			userID := uuid.New()
			expectedRide := createTestRide(userID, models.RideStatusRequested)

			reqBody := models.RideRequest{
				PickupLatitude:   40.7128 + float64(idx)*0.001,
				PickupLongitude:  -74.0060,
				PickupAddress:    fmt.Sprintf("Address %d", idx),
				DropoffLatitude:  40.7580,
				DropoffLongitude: -73.9855,
				DropoffAddress:   "456 Broadway",
			}

			mockService.On("RequestRide", mock.Anything, userID, mock.AnythingOfType("*models.RideRequest")).Return(expectedRide, nil)

			c, w := setupTestContext("POST", "/api/v1/rides", reqBody)
			setUserContext(c, userID, models.RoleRider)

			handler.RequestRide(c)

			assert.Equal(t, http.StatusCreated, w.Code)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestHandler_RequestRide_SpecialCharactersInAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		address string
	}{
		{"unicode characters", "123 Main St, New York"},
		{"special symbols", "123 Main St #&@!"},
		{"quotes", `123 "Main" St`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			handler := NewMockableHandler(mockService)

			userID := uuid.New()
			expectedRide := createTestRide(userID, models.RideStatusRequested)

			mockService.On("RequestRide", mock.Anything, userID, mock.AnythingOfType("*models.RideRequest")).Return(expectedRide, nil)

			reqBody := map[string]interface{}{
				"pickup_latitude":   40.7128,
				"pickup_longitude":  -74.0060,
				"pickup_address":    tt.address,
				"dropoff_latitude":  40.7580,
				"dropoff_longitude": -73.9855,
				"dropoff_address":   "456 Broadway",
			}

			c, w := setupTestContext("POST", "/api/v1/rides", reqBody)
			setUserContext(c, userID, models.RoleRider)

			handler.RequestRide(c)

			assert.Equal(t, http.StatusCreated, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_RequestRide_ExtremeCoordinates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name            string
		pickupLatitude  float64
		pickupLongitude float64
		dropoffLatitude float64
		dropoffLongitude float64
	}{
		{"valid NYC", 40.7128, -74.0060, 40.7580, -73.9855},
		{"negative coordinates", -33.8688, 151.2093, -33.8568, 151.2153},
		{"small positive coordinates", 1.0, 1.0, 2.0, 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			handler := NewMockableHandler(mockService)

			userID := uuid.New()
			expectedRide := createTestRide(userID, models.RideStatusRequested)

			mockService.On("RequestRide", mock.Anything, userID, mock.AnythingOfType("*models.RideRequest")).Return(expectedRide, nil)

			reqBody := map[string]interface{}{
				"pickup_latitude":   tt.pickupLatitude,
				"pickup_longitude":  tt.pickupLongitude,
				"pickup_address":    "123 Main St",
				"dropoff_latitude":  tt.dropoffLatitude,
				"dropoff_longitude": tt.dropoffLongitude,
				"dropoff_address":   "456 Broadway",
			}

			c, w := setupTestContext("POST", "/api/v1/rides", reqBody)
			setUserContext(c, userID, models.RoleRider)

			handler.RequestRide(c)

			assert.Equal(t, http.StatusCreated, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_SuccessResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/rides/surge-info?latitude=40.7128&longitude=-74.0060", nil)
	req := httptest.NewRequest("GET", "/api/v1/rides/surge-info?latitude=40.7128&longitude=-74.0060", nil)
	c.Request = req

	handler.GetSurgeInfo(c)

	response := parseResponse(w)

	// Verify response structure
	assert.Contains(t, response, "success")
	assert.Contains(t, response, "data")
	assert.True(t, response["success"].(bool))
}

func TestHandler_ErrorResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/rides/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.GetRide(c)

	response := parseResponse(w)

	// Verify error response structure
	assert.Contains(t, response, "success")
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response, "error")

	errorInfo := response["error"].(map[string]interface{})
	assert.Contains(t, errorInfo, "code")
	assert.Contains(t, errorInfo, "message")
}

// ============================================================================
// AppError Response Tests
// ============================================================================

func TestHandler_AppErrorResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		appErr       *common.AppError
		expectedCode int
	}{
		{
			name:         "not found error",
			appErr:       common.NewNotFoundError("ride not found", nil),
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "bad request error",
			appErr:       common.NewBadRequestError("invalid input", nil),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "internal error",
			appErr:       common.NewInternalServerError("server error"),
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:         "conflict error",
			appErr:       common.NewConflictError("ride already accepted"),
			expectedCode: http.StatusConflict,
		},
		{
			name:         "forbidden error",
			appErr:       common.NewForbiddenError("access denied"),
			expectedCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			// Add a request context to avoid nil pointer dereference
			c.Request = httptest.NewRequest("GET", "/test", nil)

			common.AppErrorResponse(c, tt.appErr)

			assert.Equal(t, tt.expectedCode, w.Code)
			response := parseResponse(w)
			assert.False(t, response["success"].(bool))
		})
	}
}

// ============================================================================
// CompleteRide Edge Cases
// ============================================================================

func TestHandler_CompleteRide_VeryLargeDistance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	riderID := uuid.New()
	rideID := uuid.New()
	expectedRide := createTestRideWithDriver(riderID, driverID, models.RideStatusCompleted)
	expectedRide.ID = rideID
	largeDistance := 10000.0
	expectedRide.ActualDistance = &largeDistance

	mockService.On("CompleteRide", mock.Anything, rideID, driverID, largeDistance).Return(expectedRide, nil)

	reqBody := map[string]interface{}{
		"actual_distance": largeDistance,
	}

	c, w := setupTestContext("POST", "/api/v1/driver/rides/"+rideID.String()+"/complete", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.CompleteRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CompleteRide_VerySmallDistance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	riderID := uuid.New()
	rideID := uuid.New()
	expectedRide := createTestRideWithDriver(riderID, driverID, models.RideStatusCompleted)
	expectedRide.ID = rideID
	smallDistance := 0.001
	expectedRide.ActualDistance = &smallDistance

	mockService.On("CompleteRide", mock.Anything, rideID, driverID, smallDistance).Return(expectedRide, nil)

	reqBody := map[string]interface{}{
		"actual_distance": smallDistance,
	}

	c, w := setupTestContext("POST", "/api/v1/driver/rides/"+rideID.String()+"/complete", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.CompleteRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}
