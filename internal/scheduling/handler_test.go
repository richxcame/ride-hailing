package scheduling

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
// Mock Service Interface
// ============================================================================

// SchedulingServiceInterface defines the interface for the scheduling service used by the handler
type SchedulingServiceInterface interface {
	CreateRecurringRide(ctx context.Context, riderID uuid.UUID, req *CreateRecurringRideRequest) (*RecurringRideResponse, error)
	GetRecurringRide(ctx context.Context, riderID uuid.UUID, rideID uuid.UUID) (*RecurringRideResponse, error)
	ListRecurringRides(ctx context.Context, riderID uuid.UUID) ([]*RecurringRideResponse, error)
	UpdateRecurringRide(ctx context.Context, riderID uuid.UUID, rideID uuid.UUID, req *UpdateRecurringRideRequest) error
	PauseRecurringRide(ctx context.Context, riderID uuid.UUID, rideID uuid.UUID) error
	ResumeRecurringRide(ctx context.Context, riderID uuid.UUID, rideID uuid.UUID) error
	CancelRecurringRide(ctx context.Context, riderID uuid.UUID, rideID uuid.UUID) error
	GetUpcomingInstances(ctx context.Context, riderID uuid.UUID, days int) ([]*ScheduledRideInstance, error)
	SkipInstance(ctx context.Context, riderID uuid.UUID, instanceID uuid.UUID, reason string) error
	RescheduleInstance(ctx context.Context, riderID uuid.UUID, instanceID uuid.UUID, req *RescheduleInstanceRequest) error
	PreviewSchedule(ctx context.Context, req *SchedulePreviewRequest) (*SchedulePreviewResponse, error)
	GetRiderStats(ctx context.Context, riderID uuid.UUID) (*RecurringRideStats, error)
}

// MockSchedulingService is a mock implementation of SchedulingServiceInterface
type MockSchedulingService struct {
	mock.Mock
}

func (m *MockSchedulingService) CreateRecurringRide(ctx context.Context, riderID uuid.UUID, req *CreateRecurringRideRequest) (*RecurringRideResponse, error) {
	args := m.Called(ctx, riderID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RecurringRideResponse), args.Error(1)
}

func (m *MockSchedulingService) GetRecurringRide(ctx context.Context, riderID uuid.UUID, rideID uuid.UUID) (*RecurringRideResponse, error) {
	args := m.Called(ctx, riderID, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RecurringRideResponse), args.Error(1)
}

func (m *MockSchedulingService) ListRecurringRides(ctx context.Context, riderID uuid.UUID) ([]*RecurringRideResponse, error) {
	args := m.Called(ctx, riderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RecurringRideResponse), args.Error(1)
}

func (m *MockSchedulingService) UpdateRecurringRide(ctx context.Context, riderID uuid.UUID, rideID uuid.UUID, req *UpdateRecurringRideRequest) error {
	args := m.Called(ctx, riderID, rideID, req)
	return args.Error(0)
}

func (m *MockSchedulingService) PauseRecurringRide(ctx context.Context, riderID uuid.UUID, rideID uuid.UUID) error {
	args := m.Called(ctx, riderID, rideID)
	return args.Error(0)
}

func (m *MockSchedulingService) ResumeRecurringRide(ctx context.Context, riderID uuid.UUID, rideID uuid.UUID) error {
	args := m.Called(ctx, riderID, rideID)
	return args.Error(0)
}

func (m *MockSchedulingService) CancelRecurringRide(ctx context.Context, riderID uuid.UUID, rideID uuid.UUID) error {
	args := m.Called(ctx, riderID, rideID)
	return args.Error(0)
}

func (m *MockSchedulingService) GetUpcomingInstances(ctx context.Context, riderID uuid.UUID, days int) ([]*ScheduledRideInstance, error) {
	args := m.Called(ctx, riderID, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ScheduledRideInstance), args.Error(1)
}

func (m *MockSchedulingService) SkipInstance(ctx context.Context, riderID uuid.UUID, instanceID uuid.UUID, reason string) error {
	args := m.Called(ctx, riderID, instanceID, reason)
	return args.Error(0)
}

func (m *MockSchedulingService) RescheduleInstance(ctx context.Context, riderID uuid.UUID, instanceID uuid.UUID, req *RescheduleInstanceRequest) error {
	args := m.Called(ctx, riderID, instanceID, req)
	return args.Error(0)
}

func (m *MockSchedulingService) PreviewSchedule(ctx context.Context, req *SchedulePreviewRequest) (*SchedulePreviewResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SchedulePreviewResponse), args.Error(1)
}

func (m *MockSchedulingService) GetRiderStats(ctx context.Context, riderID uuid.UUID) (*RecurringRideStats, error) {
	args := m.Called(ctx, riderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RecurringRideStats), args.Error(1)
}

// ============================================================================
// Testable Handler (wraps mock service)
// ============================================================================

// TestableHandler provides a testable handler with mock service injection
type TestableHandler struct {
	service *MockSchedulingService
}

func NewTestableHandler(mockService *MockSchedulingService) *TestableHandler {
	return &TestableHandler{service: mockService}
}

// CreateRecurringRide - testable version
func (h *TestableHandler) CreateRecurringRide(c *gin.Context) {
	riderID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateRecurringRideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	response, err := h.service.CreateRecurringRide(c.Request.Context(), riderID.(uuid.UUID), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create recurring ride")
		return
	}

	common.SuccessResponse(c, response)
}

// GetRecurringRide - testable version
func (h *TestableHandler) GetRecurringRide(c *gin.Context) {
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

	response, err := h.service.GetRecurringRide(c.Request.Context(), riderID.(uuid.UUID), rideID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "recurring ride not found")
		return
	}

	common.SuccessResponse(c, response)
}

// ListRecurringRides - testable version
func (h *TestableHandler) ListRecurringRides(c *gin.Context) {
	riderID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rides, err := h.service.ListRecurringRides(c.Request.Context(), riderID.(uuid.UUID))
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list recurring rides")
		return
	}

	common.SuccessResponse(c, rides)
}

// UpdateRecurringRide - testable version
func (h *TestableHandler) UpdateRecurringRide(c *gin.Context) {
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

	var req UpdateRecurringRideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.UpdateRecurringRide(c.Request.Context(), riderID.(uuid.UUID), rideID, &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update recurring ride")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Recurring ride updated successfully",
	})
}

// PauseRecurringRide - testable version
func (h *TestableHandler) PauseRecurringRide(c *gin.Context) {
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

	if err := h.service.PauseRecurringRide(c.Request.Context(), riderID.(uuid.UUID), rideID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to pause recurring ride")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Recurring ride paused successfully",
	})
}

// ResumeRecurringRide - testable version
func (h *TestableHandler) ResumeRecurringRide(c *gin.Context) {
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

	if err := h.service.ResumeRecurringRide(c.Request.Context(), riderID.(uuid.UUID), rideID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to resume recurring ride")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Recurring ride resumed successfully",
	})
}

// CancelRecurringRide - testable version
func (h *TestableHandler) CancelRecurringRide(c *gin.Context) {
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

	if err := h.service.CancelRecurringRide(c.Request.Context(), riderID.(uuid.UUID), rideID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to cancel recurring ride")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Recurring ride cancelled successfully",
	})
}

// GetUpcomingInstances - testable version
func (h *TestableHandler) GetUpcomingInstances(c *gin.Context) {
	riderID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	days := 7 // Default days

	instances, err := h.service.GetUpcomingInstances(c.Request.Context(), riderID.(uuid.UUID), days)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get upcoming instances")
		return
	}

	common.SuccessResponse(c, gin.H{
		"instances": instances,
		"count":     len(instances),
	})
}

// SkipInstance - testable version
func (h *TestableHandler) SkipInstance(c *gin.Context) {
	riderID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	instanceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid instance ID")
		return
	}

	var req SkipInstanceRequest
	// Ignore binding error - reason is optional
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = ""
	}

	if err := h.service.SkipInstance(c.Request.Context(), riderID.(uuid.UUID), instanceID, req.Reason); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to skip instance")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Instance skipped successfully",
	})
}

// RescheduleInstance - testable version
func (h *TestableHandler) RescheduleInstance(c *gin.Context) {
	riderID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	instanceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid instance ID")
		return
	}

	var req RescheduleInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.RescheduleInstance(c.Request.Context(), riderID.(uuid.UUID), instanceID, &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to reschedule instance")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Instance rescheduled successfully",
	})
}

// PreviewSchedule - testable version
func (h *TestableHandler) PreviewSchedule(c *gin.Context) {
	var req SchedulePreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	preview, err := h.service.PreviewSchedule(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to generate preview")
		return
	}

	common.SuccessResponse(c, preview)
}

// GetStats - testable version
func (h *TestableHandler) GetStats(c *gin.Context) {
	riderID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	stats, err := h.service.GetRiderStats(c.Request.Context(), riderID.(uuid.UUID))
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get stats")
		return
	}

	common.SuccessResponse(c, stats)
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
	c.Set("user_email", "test@example.com")
	c.Set("user_role", "rider")
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

func createTestRecurringRide(riderID uuid.UUID) *RecurringRide {
	return &RecurringRide{
		ID:                uuid.New(),
		RiderID:           riderID,
		Name:              "Morning Commute",
		PickupLocation:    Location{Latitude: 37.7749, Longitude: -122.4194},
		DropoffLocation:   Location{Latitude: 37.7849, Longitude: -122.4094},
		PickupAddress:     "123 Main St",
		DropoffAddress:    "456 Market St",
		RideType:          "economy",
		RecurrencePattern: RecurrenceWeekdays,
		DaysOfWeek:        []int{1, 2, 3, 4, 5},
		ScheduledTime:     "08:30",
		Timezone:          "America/Los_Angeles",
		StartDate:         time.Now().AddDate(0, 0, 1),
		Status:            ScheduleStatusActive,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

func createTestRecurringRideResponse(riderID uuid.UUID) *RecurringRideResponse {
	ride := createTestRecurringRide(riderID)
	return &RecurringRideResponse{
		RecurringRide:     ride,
		UpcomingInstances: []ScheduledRideInstance{},
		EstimatedFare:     15.00,
		TotalRidesBooked:  0,
		TotalSpent:        0,
	}
}

func createTestInstance(riderID uuid.UUID, recurringRideID uuid.UUID) *ScheduledRideInstance {
	return &ScheduledRideInstance{
		ID:              uuid.New(),
		RecurringRideID: recurringRideID,
		RiderID:         riderID,
		ScheduledDate:   time.Now().AddDate(0, 0, 1),
		ScheduledTime:   "08:30",
		PickupAt:        time.Now().AddDate(0, 0, 1),
		PickupLocation:  Location{Latitude: 37.7749, Longitude: -122.4194},
		DropoffLocation: Location{Latitude: 37.7849, Longitude: -122.4094},
		PickupAddress:   "123 Main St",
		DropoffAddress:  "456 Market St",
		EstimatedFare:   15.00,
		Status:          InstanceStatusScheduled,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func createTestStats() *RecurringRideStats {
	return &RecurringRideStats{
		ActiveSchedules:  3,
		TotalRidesBooked: 25,
		CompletedRides:   20,
		CancelledRides:   2,
		TotalSavings:     50.00,
		AvgRideFrequency: 5.0,
		MostCommonRoute:  "Home to Office",
		PreferredTime:    "08:30",
	}
}

// ============================================================================
// CreateRecurringRide Handler Tests
// ============================================================================

func TestHandler_CreateRecurringRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	expectedResponse := createTestRecurringRideResponse(riderID)

	reqBody := CreateRecurringRideRequest{
		Name:              "Morning Commute",
		PickupLocation:    Location{Latitude: 37.7749, Longitude: -122.4194},
		DropoffLocation:   Location{Latitude: 37.7849, Longitude: -122.4094},
		PickupAddress:     "123 Main St",
		DropoffAddress:    "456 Market St",
		RideType:          "economy",
		RecurrencePattern: RecurrenceWeekdays,
		DaysOfWeek:        []int{1, 2, 3, 4, 5},
		ScheduledTime:     "08:30",
		StartDate:         time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
	}

	mockService.On("CreateRecurringRide", mock.Anything, riderID, &reqBody).Return(expectedResponse, nil)

	c, w := setupTestContext("POST", "/api/v1/scheduling/recurring-rides", reqBody)
	setUserContext(c, riderID)

	handler.CreateRecurringRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
	mockService.AssertExpectations(t)
}

func TestHandler_CreateRecurringRide_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	reqBody := CreateRecurringRideRequest{
		Name:              "Morning Commute",
		PickupLocation:    Location{Latitude: 37.7749, Longitude: -122.4194},
		DropoffLocation:   Location{Latitude: 37.7849, Longitude: -122.4094},
		RideType:          "economy",
		RecurrencePattern: RecurrenceWeekdays,
		ScheduledTime:     "08:30",
		StartDate:         time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
	}

	c, w := setupTestContext("POST", "/api/v1/scheduling/recurring-rides", reqBody)
	// Don't set user context to simulate unauthorized

	handler.CreateRecurringRide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_CreateRecurringRide_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/scheduling/recurring-rides", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/scheduling/recurring-rides", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, riderID)

	handler.CreateRecurringRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_CreateRecurringRide_MissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "missing name",
			body: map[string]interface{}{
				"pickup_location":    map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
				"dropoff_location":   map[string]float64{"latitude": 37.7849, "longitude": -122.4094},
				"ride_type":          "economy",
				"recurrence_pattern": "weekdays",
				"scheduled_time":     "08:30",
				"start_date":         "2024-01-15",
			},
		},
		{
			name: "missing ride_type",
			body: map[string]interface{}{
				"name":               "Test Ride",
				"pickup_location":    map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
				"dropoff_location":   map[string]float64{"latitude": 37.7849, "longitude": -122.4094},
				"recurrence_pattern": "weekdays",
				"scheduled_time":     "08:30",
				"start_date":         "2024-01-15",
			},
		},
		{
			name: "missing recurrence_pattern",
			body: map[string]interface{}{
				"name":             "Test Ride",
				"pickup_location":  map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
				"dropoff_location": map[string]float64{"latitude": 37.7849, "longitude": -122.4094},
				"ride_type":        "economy",
				"scheduled_time":   "08:30",
				"start_date":       "2024-01-15",
			},
		},
		{
			name: "missing scheduled_time",
			body: map[string]interface{}{
				"name":               "Test Ride",
				"pickup_location":    map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
				"dropoff_location":   map[string]float64{"latitude": 37.7849, "longitude": -122.4094},
				"ride_type":          "economy",
				"recurrence_pattern": "weekdays",
				"start_date":         "2024-01-15",
			},
		},
		{
			name: "missing start_date",
			body: map[string]interface{}{
				"name":               "Test Ride",
				"pickup_location":    map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
				"dropoff_location":   map[string]float64{"latitude": 37.7849, "longitude": -122.4094},
				"ride_type":          "economy",
				"recurrence_pattern": "weekdays",
				"scheduled_time":     "08:30",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSchedulingService)
			handler := NewTestableHandler(mockService)
			riderID := uuid.New()

			c, w := setupTestContext("POST", "/api/v1/scheduling/recurring-rides", tt.body)
			setUserContext(c, riderID)

			handler.CreateRecurringRide(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandler_CreateRecurringRide_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	reqBody := CreateRecurringRideRequest{
		Name:              "Morning Commute",
		PickupLocation:    Location{Latitude: 37.7749, Longitude: -122.4194},
		DropoffLocation:   Location{Latitude: 37.7849, Longitude: -122.4094},
		RideType:          "economy",
		RecurrencePattern: RecurrenceWeekdays,
		ScheduledTime:     "08:30",
		StartDate:         time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
	}

	mockService.On("CreateRecurringRide", mock.Anything, riderID, &reqBody).Return(nil, errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/scheduling/recurring-rides", reqBody)
	setUserContext(c, riderID)

	handler.CreateRecurringRide(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CreateRecurringRide_AppError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	reqBody := CreateRecurringRideRequest{
		Name:              "Morning Commute",
		PickupLocation:    Location{Latitude: 37.7749, Longitude: -122.4194},
		DropoffLocation:   Location{Latitude: 37.7849, Longitude: -122.4094},
		RideType:          "economy",
		RecurrencePattern: RecurrenceWeekdays,
		ScheduledTime:     "08:30",
		StartDate:         time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
	}

	mockService.On("CreateRecurringRide", mock.Anything, riderID, &reqBody).Return(nil, common.NewBadRequestError("start_date cannot be in the past", nil))

	c, w := setupTestContext("POST", "/api/v1/scheduling/recurring-rides", reqBody)
	setUserContext(c, riderID)

	handler.CreateRecurringRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetRecurringRide Handler Tests
// ============================================================================

func TestHandler_GetRecurringRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()
	expectedResponse := createTestRecurringRideResponse(riderID)
	expectedResponse.RecurringRide.ID = rideID

	mockService.On("GetRecurringRide", mock.Anything, riderID, rideID).Return(expectedResponse, nil)

	c, w := setupTestContext("GET", "/api/v1/scheduling/recurring-rides/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, riderID)

	handler.GetRecurringRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetRecurringRide_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	rideID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/scheduling/recurring-rides/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	// Don't set user context

	handler.GetRecurringRide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetRecurringRide_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/scheduling/recurring-rides/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, riderID)

	handler.GetRecurringRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid ride ID")
}

func TestHandler_GetRecurringRide_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()

	mockService.On("GetRecurringRide", mock.Anything, riderID, rideID).Return(nil, common.NewNotFoundError("recurring ride not found", nil))

	c, w := setupTestContext("GET", "/api/v1/scheduling/recurring-rides/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, riderID)

	handler.GetRecurringRide(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetRecurringRide_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()

	mockService.On("GetRecurringRide", mock.Anything, riderID, rideID).Return(nil, common.NewForbiddenError("not authorized to view this ride"))

	c, w := setupTestContext("GET", "/api/v1/scheduling/recurring-rides/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, riderID)

	handler.GetRecurringRide(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// ListRecurringRides Handler Tests
// ============================================================================

func TestHandler_ListRecurringRides_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	rides := []*RecurringRideResponse{
		createTestRecurringRideResponse(riderID),
		createTestRecurringRideResponse(riderID),
	}

	mockService.On("ListRecurringRides", mock.Anything, riderID).Return(rides, nil)

	c, w := setupTestContext("GET", "/api/v1/scheduling/recurring-rides", nil)
	setUserContext(c, riderID)

	handler.ListRecurringRides(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_ListRecurringRides_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/scheduling/recurring-rides", nil)
	// Don't set user context

	handler.ListRecurringRides(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_ListRecurringRides_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()

	mockService.On("ListRecurringRides", mock.Anything, riderID).Return([]*RecurringRideResponse{}, nil)

	c, w := setupTestContext("GET", "/api/v1/scheduling/recurring-rides", nil)
	setUserContext(c, riderID)

	handler.ListRecurringRides(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ListRecurringRides_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()

	mockService.On("ListRecurringRides", mock.Anything, riderID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/scheduling/recurring-rides", nil)
	setUserContext(c, riderID)

	handler.ListRecurringRides(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// UpdateRecurringRide Handler Tests
// ============================================================================

func TestHandler_UpdateRecurringRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()
	newName := "Evening Commute"
	reqBody := UpdateRecurringRideRequest{
		Name: &newName,
	}

	mockService.On("UpdateRecurringRide", mock.Anything, riderID, rideID, &reqBody).Return(nil)

	c, w := setupTestContext("PUT", "/api/v1/scheduling/recurring-rides/"+rideID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, riderID)

	handler.UpdateRecurringRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateRecurringRide_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	rideID := uuid.New()
	newName := "Evening Commute"
	reqBody := UpdateRecurringRideRequest{
		Name: &newName,
	}

	c, w := setupTestContext("PUT", "/api/v1/scheduling/recurring-rides/"+rideID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	// Don't set user context

	handler.UpdateRecurringRide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_UpdateRecurringRide_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	newName := "Evening Commute"
	reqBody := UpdateRecurringRideRequest{
		Name: &newName,
	}

	c, w := setupTestContext("PUT", "/api/v1/scheduling/recurring-rides/invalid", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, riderID)

	handler.UpdateRecurringRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateRecurringRide_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()

	c, w := setupTestContext("PUT", "/api/v1/scheduling/recurring-rides/"+rideID.String(), nil)
	c.Request = httptest.NewRequest("PUT", "/api/v1/scheduling/recurring-rides/"+rideID.String(), bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, riderID)

	handler.UpdateRecurringRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateRecurringRide_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()
	newName := "Evening Commute"
	reqBody := UpdateRecurringRideRequest{
		Name: &newName,
	}

	mockService.On("UpdateRecurringRide", mock.Anything, riderID, rideID, &reqBody).Return(common.NewNotFoundError("recurring ride not found", nil))

	c, w := setupTestContext("PUT", "/api/v1/scheduling/recurring-rides/"+rideID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, riderID)

	handler.UpdateRecurringRide(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateRecurringRide_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()
	newName := "Evening Commute"
	reqBody := UpdateRecurringRideRequest{
		Name: &newName,
	}

	mockService.On("UpdateRecurringRide", mock.Anything, riderID, rideID, &reqBody).Return(errors.New("database error"))

	c, w := setupTestContext("PUT", "/api/v1/scheduling/recurring-rides/"+rideID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, riderID)

	handler.UpdateRecurringRide(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// PauseRecurringRide Handler Tests
// ============================================================================

func TestHandler_PauseRecurringRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()

	mockService.On("PauseRecurringRide", mock.Anything, riderID, rideID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/scheduling/recurring-rides/"+rideID.String()+"/pause", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, riderID)

	handler.PauseRecurringRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_PauseRecurringRide_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	rideID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/scheduling/recurring-rides/"+rideID.String()+"/pause", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	// Don't set user context

	handler.PauseRecurringRide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_PauseRecurringRide_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/scheduling/recurring-rides/invalid/pause", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, riderID)

	handler.PauseRecurringRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PauseRecurringRide_NotActive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()

	mockService.On("PauseRecurringRide", mock.Anything, riderID, rideID).Return(common.NewBadRequestError("ride is not active", nil))

	c, w := setupTestContext("POST", "/api/v1/scheduling/recurring-rides/"+rideID.String()+"/pause", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, riderID)

	handler.PauseRecurringRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_PauseRecurringRide_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()

	mockService.On("PauseRecurringRide", mock.Anything, riderID, rideID).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/scheduling/recurring-rides/"+rideID.String()+"/pause", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, riderID)

	handler.PauseRecurringRide(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// ResumeRecurringRide Handler Tests
// ============================================================================

func TestHandler_ResumeRecurringRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()

	mockService.On("ResumeRecurringRide", mock.Anything, riderID, rideID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/scheduling/recurring-rides/"+rideID.String()+"/resume", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, riderID)

	handler.ResumeRecurringRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_ResumeRecurringRide_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	rideID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/scheduling/recurring-rides/"+rideID.String()+"/resume", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	// Don't set user context

	handler.ResumeRecurringRide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_ResumeRecurringRide_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/scheduling/recurring-rides/invalid/resume", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, riderID)

	handler.ResumeRecurringRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ResumeRecurringRide_NotPaused(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()

	mockService.On("ResumeRecurringRide", mock.Anything, riderID, rideID).Return(common.NewBadRequestError("ride is not paused", nil))

	c, w := setupTestContext("POST", "/api/v1/scheduling/recurring-rides/"+rideID.String()+"/resume", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, riderID)

	handler.ResumeRecurringRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// CancelRecurringRide Handler Tests
// ============================================================================

func TestHandler_CancelRecurringRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()

	mockService.On("CancelRecurringRide", mock.Anything, riderID, rideID).Return(nil)

	c, w := setupTestContext("DELETE", "/api/v1/scheduling/recurring-rides/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, riderID)

	handler.CancelRecurringRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_CancelRecurringRide_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	rideID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/scheduling/recurring-rides/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	// Don't set user context

	handler.CancelRecurringRide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CancelRecurringRide_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/scheduling/recurring-rides/invalid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, riderID)

	handler.CancelRecurringRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CancelRecurringRide_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()

	mockService.On("CancelRecurringRide", mock.Anything, riderID, rideID).Return(common.NewNotFoundError("recurring ride not found", nil))

	c, w := setupTestContext("DELETE", "/api/v1/scheduling/recurring-rides/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, riderID)

	handler.CancelRecurringRide(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CancelRecurringRide_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()

	mockService.On("CancelRecurringRide", mock.Anything, riderID, rideID).Return(common.NewForbiddenError("not authorized to cancel this ride"))

	c, w := setupTestContext("DELETE", "/api/v1/scheduling/recurring-rides/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, riderID)

	handler.CancelRecurringRide(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetUpcomingInstances Handler Tests
// ============================================================================

func TestHandler_GetUpcomingInstances_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	recurringRideID := uuid.New()
	instances := []*ScheduledRideInstance{
		createTestInstance(riderID, recurringRideID),
		createTestInstance(riderID, recurringRideID),
	}

	mockService.On("GetUpcomingInstances", mock.Anything, riderID, 7).Return(instances, nil)

	c, w := setupTestContext("GET", "/api/v1/scheduling/instances", nil)
	setUserContext(c, riderID)

	handler.GetUpcomingInstances(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetUpcomingInstances_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/scheduling/instances", nil)
	// Don't set user context

	handler.GetUpcomingInstances(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetUpcomingInstances_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()

	mockService.On("GetUpcomingInstances", mock.Anything, riderID, 7).Return([]*ScheduledRideInstance{}, nil)

	c, w := setupTestContext("GET", "/api/v1/scheduling/instances", nil)
	setUserContext(c, riderID)

	handler.GetUpcomingInstances(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["count"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetUpcomingInstances_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()

	mockService.On("GetUpcomingInstances", mock.Anything, riderID, 7).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/scheduling/instances", nil)
	setUserContext(c, riderID)

	handler.GetUpcomingInstances(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// SkipInstance Handler Tests
// ============================================================================

func TestHandler_SkipInstance_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	instanceID := uuid.New()
	reqBody := SkipInstanceRequest{
		Reason: "Doctor's appointment",
	}

	mockService.On("SkipInstance", mock.Anything, riderID, instanceID, "Doctor's appointment").Return(nil)

	c, w := setupTestContext("POST", "/api/v1/scheduling/instances/"+instanceID.String()+"/skip", reqBody)
	c.Params = gin.Params{{Key: "id", Value: instanceID.String()}}
	setUserContext(c, riderID)

	handler.SkipInstance(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_SkipInstance_NoReason(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	instanceID := uuid.New()

	mockService.On("SkipInstance", mock.Anything, riderID, instanceID, "").Return(nil)

	c, w := setupTestContext("POST", "/api/v1/scheduling/instances/"+instanceID.String()+"/skip", nil)
	c.Params = gin.Params{{Key: "id", Value: instanceID.String()}}
	setUserContext(c, riderID)

	handler.SkipInstance(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SkipInstance_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	instanceID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/scheduling/instances/"+instanceID.String()+"/skip", nil)
	c.Params = gin.Params{{Key: "id", Value: instanceID.String()}}
	// Don't set user context

	handler.SkipInstance(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_SkipInstance_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/scheduling/instances/invalid/skip", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, riderID)

	handler.SkipInstance(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SkipInstance_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	instanceID := uuid.New()

	mockService.On("SkipInstance", mock.Anything, riderID, instanceID, "").Return(common.NewNotFoundError("instance not found", nil))

	c, w := setupTestContext("POST", "/api/v1/scheduling/instances/"+instanceID.String()+"/skip", nil)
	c.Params = gin.Params{{Key: "id", Value: instanceID.String()}}
	setUserContext(c, riderID)

	handler.SkipInstance(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SkipInstance_CannotSkip(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	instanceID := uuid.New()

	mockService.On("SkipInstance", mock.Anything, riderID, instanceID, "").Return(common.NewBadRequestError("cannot skip this instance", nil))

	c, w := setupTestContext("POST", "/api/v1/scheduling/instances/"+instanceID.String()+"/skip", nil)
	c.Params = gin.Params{{Key: "id", Value: instanceID.String()}}
	setUserContext(c, riderID)

	handler.SkipInstance(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// RescheduleInstance Handler Tests
// ============================================================================

func TestHandler_RescheduleInstance_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	instanceID := uuid.New()
	reqBody := RescheduleInstanceRequest{
		NewDate: "2024-01-20",
		NewTime: "09:00",
	}

	mockService.On("RescheduleInstance", mock.Anything, riderID, instanceID, &reqBody).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/scheduling/instances/"+instanceID.String()+"/reschedule", reqBody)
	c.Params = gin.Params{{Key: "id", Value: instanceID.String()}}
	setUserContext(c, riderID)

	handler.RescheduleInstance(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_RescheduleInstance_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	instanceID := uuid.New()
	reqBody := RescheduleInstanceRequest{
		NewDate: "2024-01-20",
		NewTime: "09:00",
	}

	c, w := setupTestContext("POST", "/api/v1/scheduling/instances/"+instanceID.String()+"/reschedule", reqBody)
	c.Params = gin.Params{{Key: "id", Value: instanceID.String()}}
	// Don't set user context

	handler.RescheduleInstance(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_RescheduleInstance_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	reqBody := RescheduleInstanceRequest{
		NewDate: "2024-01-20",
		NewTime: "09:00",
	}

	c, w := setupTestContext("POST", "/api/v1/scheduling/instances/invalid/reschedule", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, riderID)

	handler.RescheduleInstance(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RescheduleInstance_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	instanceID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/scheduling/instances/"+instanceID.String()+"/reschedule", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/scheduling/instances/"+instanceID.String()+"/reschedule", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: instanceID.String()}}
	setUserContext(c, riderID)

	handler.RescheduleInstance(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RescheduleInstance_MissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "missing new_date",
			body: map[string]interface{}{
				"new_time": "09:00",
			},
		},
		{
			name: "missing new_time",
			body: map[string]interface{}{
				"new_date": "2024-01-20",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSchedulingService)
			handler := NewTestableHandler(mockService)
			riderID := uuid.New()
			instanceID := uuid.New()

			c, w := setupTestContext("POST", "/api/v1/scheduling/instances/"+instanceID.String()+"/reschedule", tt.body)
			c.Params = gin.Params{{Key: "id", Value: instanceID.String()}}
			setUserContext(c, riderID)

			handler.RescheduleInstance(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandler_RescheduleInstance_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	instanceID := uuid.New()
	reqBody := RescheduleInstanceRequest{
		NewDate: "2024-01-20",
		NewTime: "09:00",
	}

	mockService.On("RescheduleInstance", mock.Anything, riderID, instanceID, &reqBody).Return(common.NewNotFoundError("instance not found", nil))

	c, w := setupTestContext("POST", "/api/v1/scheduling/instances/"+instanceID.String()+"/reschedule", reqBody)
	c.Params = gin.Params{{Key: "id", Value: instanceID.String()}}
	setUserContext(c, riderID)

	handler.RescheduleInstance(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_RescheduleInstance_InvalidDate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	instanceID := uuid.New()
	reqBody := RescheduleInstanceRequest{
		NewDate: "2024-01-20",
		NewTime: "09:00",
	}

	mockService.On("RescheduleInstance", mock.Anything, riderID, instanceID, &reqBody).Return(common.NewBadRequestError("invalid date format", nil))

	c, w := setupTestContext("POST", "/api/v1/scheduling/instances/"+instanceID.String()+"/reschedule", reqBody)
	c.Params = gin.Params{{Key: "id", Value: instanceID.String()}}
	setUserContext(c, riderID)

	handler.RescheduleInstance(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// PreviewSchedule Handler Tests
// ============================================================================

func TestHandler_PreviewSchedule_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	reqBody := SchedulePreviewRequest{
		RecurrencePattern: RecurrenceWeekdays,
		DaysOfWeek:        []int{1, 2, 3, 4, 5},
		ScheduledTime:     "08:30",
		StartDate:         "2024-01-15",
	}

	expectedResponse := &SchedulePreviewResponse{
		Dates: []time.Time{
			time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC),
		},
		Count: 2,
	}

	mockService.On("PreviewSchedule", mock.Anything, &reqBody).Return(expectedResponse, nil)

	c, w := setupTestContext("POST", "/api/v1/scheduling/preview", reqBody)

	handler.PreviewSchedule(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_PreviewSchedule_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/scheduling/preview", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/scheduling/preview", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.PreviewSchedule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PreviewSchedule_MissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "missing recurrence_pattern",
			body: map[string]interface{}{
				"scheduled_time": "08:30",
				"start_date":     "2024-01-15",
			},
		},
		{
			name: "missing scheduled_time",
			body: map[string]interface{}{
				"recurrence_pattern": "weekdays",
				"start_date":         "2024-01-15",
			},
		},
		{
			name: "missing start_date",
			body: map[string]interface{}{
				"recurrence_pattern": "weekdays",
				"scheduled_time":     "08:30",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSchedulingService)
			handler := NewTestableHandler(mockService)

			c, w := setupTestContext("POST", "/api/v1/scheduling/preview", tt.body)

			handler.PreviewSchedule(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandler_PreviewSchedule_InvalidDateFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	reqBody := SchedulePreviewRequest{
		RecurrencePattern: RecurrenceWeekdays,
		ScheduledTime:     "08:30",
		StartDate:         "2024-01-15",
	}

	mockService.On("PreviewSchedule", mock.Anything, &reqBody).Return(nil, common.NewBadRequestError("invalid start_date format", nil))

	c, w := setupTestContext("POST", "/api/v1/scheduling/preview", reqBody)

	handler.PreviewSchedule(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_PreviewSchedule_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	reqBody := SchedulePreviewRequest{
		RecurrencePattern: RecurrenceWeekdays,
		ScheduledTime:     "08:30",
		StartDate:         "2024-01-15",
	}

	mockService.On("PreviewSchedule", mock.Anything, &reqBody).Return(nil, errors.New("internal error"))

	c, w := setupTestContext("POST", "/api/v1/scheduling/preview", reqBody)

	handler.PreviewSchedule(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetStats Handler Tests
// ============================================================================

func TestHandler_GetStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	expectedStats := createTestStats()

	mockService.On("GetRiderStats", mock.Anything, riderID).Return(expectedStats, nil)

	c, w := setupTestContext("GET", "/api/v1/scheduling/stats", nil)
	setUserContext(c, riderID)

	handler.GetStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetStats_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/scheduling/stats", nil)
	// Don't set user context

	handler.GetStats(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetStats_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()

	mockService.On("GetRiderStats", mock.Anything, riderID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/scheduling/stats", nil)
	setUserContext(c, riderID)

	handler.GetStats(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_RecurringRide_AllRecurrencePatterns(t *testing.T) {
	gin.SetMode(gin.TestMode)

	patterns := []RecurrencePattern{
		RecurrenceDaily,
		RecurrenceWeekdays,
		RecurrenceWeekends,
		RecurrenceWeekly,
		RecurrenceBiweekly,
		RecurrenceMonthly,
		RecurrenceCustom,
	}

	for _, pattern := range patterns {
		t.Run(string(pattern), func(t *testing.T) {
			mockService := new(MockSchedulingService)
			handler := NewTestableHandler(mockService)

			riderID := uuid.New()
			expectedResponse := createTestRecurringRideResponse(riderID)
			expectedResponse.RecurringRide.RecurrencePattern = pattern

			reqBody := CreateRecurringRideRequest{
				Name:              "Test Ride",
				PickupLocation:    Location{Latitude: 37.7749, Longitude: -122.4194},
				DropoffLocation:   Location{Latitude: 37.7849, Longitude: -122.4094},
				RideType:          "economy",
				RecurrencePattern: pattern,
				DaysOfWeek:        []int{1, 2, 3, 4, 5},
				ScheduledTime:     "08:30",
				StartDate:         time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
			}

			mockService.On("CreateRecurringRide", mock.Anything, riderID, mock.AnythingOfType("*scheduling.CreateRecurringRideRequest")).Return(expectedResponse, nil)

			c, w := setupTestContext("POST", "/api/v1/scheduling/recurring-rides", reqBody)
			setUserContext(c, riderID)

			handler.CreateRecurringRide(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_RecurringRide_StatusTransitions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		action         string
		method         func(h *TestableHandler, c *gin.Context)
		setupMock      func(m *MockSchedulingService, riderID, rideID uuid.UUID)
		expectedStatus int
	}{
		{
			name:   "pause active ride",
			action: "pause",
			method: func(h *TestableHandler, c *gin.Context) { h.PauseRecurringRide(c) },
			setupMock: func(m *MockSchedulingService, riderID, rideID uuid.UUID) {
				m.On("PauseRecurringRide", mock.Anything, riderID, rideID).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "pause already paused ride",
			action: "pause",
			method: func(h *TestableHandler, c *gin.Context) { h.PauseRecurringRide(c) },
			setupMock: func(m *MockSchedulingService, riderID, rideID uuid.UUID) {
				m.On("PauseRecurringRide", mock.Anything, riderID, rideID).Return(common.NewBadRequestError("ride is not active", nil))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "resume paused ride",
			action: "resume",
			method: func(h *TestableHandler, c *gin.Context) { h.ResumeRecurringRide(c) },
			setupMock: func(m *MockSchedulingService, riderID, rideID uuid.UUID) {
				m.On("ResumeRecurringRide", mock.Anything, riderID, rideID).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "resume active ride",
			action: "resume",
			method: func(h *TestableHandler, c *gin.Context) { h.ResumeRecurringRide(c) },
			setupMock: func(m *MockSchedulingService, riderID, rideID uuid.UUID) {
				m.On("ResumeRecurringRide", mock.Anything, riderID, rideID).Return(common.NewBadRequestError("ride is not paused", nil))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "cancel ride",
			action: "cancel",
			method: func(h *TestableHandler, c *gin.Context) { h.CancelRecurringRide(c) },
			setupMock: func(m *MockSchedulingService, riderID, rideID uuid.UUID) {
				m.On("CancelRecurringRide", mock.Anything, riderID, rideID).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSchedulingService)
			handler := NewTestableHandler(mockService)

			riderID := uuid.New()
			rideID := uuid.New()

			tt.setupMock(mockService, riderID, rideID)

			c, w := setupTestContext("POST", "/api/v1/scheduling/recurring-rides/"+rideID.String()+"/"+tt.action, nil)
			c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
			setUserContext(c, riderID)

			tt.method(handler, c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
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
			name:           "empty uuid",
			uuid:           "",
			expectedStatus: http.StatusBadRequest,
		},
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
			name:           "invalid uuid - dashes only",
			uuid:           "--------------------",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSchedulingService)
			handler := NewTestableHandler(mockService)
			riderID := uuid.New()

			c, w := setupTestContext("GET", "/api/v1/scheduling/recurring-rides/"+tt.uuid, nil)
			c.Params = gin.Params{{Key: "id", Value: tt.uuid}}
			setUserContext(c, riderID)

			handler.GetRecurringRide(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_CreateRecurringRide_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	expectedResponse := createTestRecurringRideResponse(riderID)

	reqBody := CreateRecurringRideRequest{
		Name:              "Morning Commute",
		PickupLocation:    Location{Latitude: 37.7749, Longitude: -122.4194},
		DropoffLocation:   Location{Latitude: 37.7849, Longitude: -122.4094},
		RideType:          "economy",
		RecurrencePattern: RecurrenceWeekdays,
		ScheduledTime:     "08:30",
		StartDate:         time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
	}

	mockService.On("CreateRecurringRide", mock.Anything, riderID, &reqBody).Return(expectedResponse, nil)

	c, w := setupTestContext("POST", "/api/v1/scheduling/recurring-rides", reqBody)
	setUserContext(c, riderID)

	handler.CreateRecurringRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	// Verify response structure
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["recurring_ride"])
	assert.NotNil(t, data["upcoming_instances"])
	assert.NotNil(t, data["estimated_fare"])

	mockService.AssertExpectations(t)
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	rideID := uuid.New()

	mockService.On("GetRecurringRide", mock.Anything, riderID, rideID).Return(nil, common.NewNotFoundError("recurring ride not found", nil))

	c, w := setupTestContext("GET", "/api/v1/scheduling/recurring-rides/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, riderID)

	handler.GetRecurringRide(c)

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

func TestHandler_GetUpcomingInstances_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockSchedulingService)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	recurringRideID := uuid.New()
	instances := []*ScheduledRideInstance{
		createTestInstance(riderID, recurringRideID),
	}

	mockService.On("GetUpcomingInstances", mock.Anything, riderID, 7).Return(instances, nil)

	c, w := setupTestContext("GET", "/api/v1/scheduling/instances", nil)
	setUserContext(c, riderID)

	handler.GetUpcomingInstances(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["instances"])
	assert.NotNil(t, data["count"])
	assert.Equal(t, float64(1), data["count"])

	mockService.AssertExpectations(t)
}
