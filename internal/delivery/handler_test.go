package delivery

import (
	"bytes"
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

// ========================================
// MOCK SERVICE INTERFACE
// ========================================

// ServiceInterface defines the interface for the delivery service used by the handler
type ServiceInterface interface {
	GetEstimate(ctx interface{}, req *DeliveryEstimateRequest) (*DeliveryEstimateResponse, error)
	CreateDelivery(ctx interface{}, senderID uuid.UUID, req *CreateDeliveryRequest) (*DeliveryResponse, error)
	GetDelivery(ctx interface{}, userID, deliveryID uuid.UUID) (*DeliveryResponse, error)
	GetMyDeliveries(ctx interface{}, senderID uuid.UUID, filters *DeliveryListFilters, limit, offset int) ([]*Delivery, int64, error)
	CancelDelivery(ctx interface{}, deliveryID, userID uuid.UUID, reason string) error
	RateDelivery(ctx interface{}, deliveryID, userID uuid.UUID, req *RateDeliveryRequest) error
	GetStats(ctx interface{}, userID uuid.UUID) (*DeliveryStats, error)
	TrackDelivery(ctx interface{}, code string) (*DeliveryResponse, error)
	GetAvailableDeliveries(ctx interface{}, latitude, longitude float64) ([]*Delivery, error)
	AcceptDelivery(ctx interface{}, deliveryID, driverID uuid.UUID) (*DeliveryResponse, error)
	ConfirmPickup(ctx interface{}, deliveryID, driverID uuid.UUID, req *ConfirmPickupRequest) error
	UpdateStatus(ctx interface{}, deliveryID, driverID uuid.UUID, status DeliveryStatus) error
	ConfirmDelivery(ctx interface{}, deliveryID, driverID uuid.UUID, req *ConfirmDeliveryRequest) error
	ReturnDelivery(ctx interface{}, deliveryID, driverID uuid.UUID, reason string) error
	GetActiveDelivery(ctx interface{}, driverID uuid.UUID) (*DeliveryResponse, error)
	GetDriverDeliveries(ctx interface{}, driverID uuid.UUID, limit, offset int) ([]*Delivery, int64, error)
	UpdateStopStatus(ctx interface{}, deliveryID, stopID, driverID uuid.UUID, status string, photoURL *string) error
}

// MockService is a mock implementation of ServiceInterface
type MockService struct {
	mock.Mock
}

func (m *MockService) GetEstimate(ctx interface{}, req *DeliveryEstimateRequest) (*DeliveryEstimateResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DeliveryEstimateResponse), args.Error(1)
}

func (m *MockService) CreateDelivery(ctx interface{}, senderID uuid.UUID, req *CreateDeliveryRequest) (*DeliveryResponse, error) {
	args := m.Called(ctx, senderID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DeliveryResponse), args.Error(1)
}

func (m *MockService) GetDelivery(ctx interface{}, userID, deliveryID uuid.UUID) (*DeliveryResponse, error) {
	args := m.Called(ctx, userID, deliveryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DeliveryResponse), args.Error(1)
}

func (m *MockService) GetMyDeliveries(ctx interface{}, senderID uuid.UUID, filters *DeliveryListFilters, limit, offset int) ([]*Delivery, int64, error) {
	args := m.Called(ctx, senderID, filters, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*Delivery), args.Get(1).(int64), args.Error(2)
}

func (m *MockService) CancelDelivery(ctx interface{}, deliveryID, userID uuid.UUID, reason string) error {
	args := m.Called(ctx, deliveryID, userID, reason)
	return args.Error(0)
}

func (m *MockService) RateDelivery(ctx interface{}, deliveryID, userID uuid.UUID, req *RateDeliveryRequest) error {
	args := m.Called(ctx, deliveryID, userID, req)
	return args.Error(0)
}

func (m *MockService) GetStats(ctx interface{}, userID uuid.UUID) (*DeliveryStats, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DeliveryStats), args.Error(1)
}

func (m *MockService) TrackDelivery(ctx interface{}, code string) (*DeliveryResponse, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DeliveryResponse), args.Error(1)
}

func (m *MockService) GetAvailableDeliveries(ctx interface{}, latitude, longitude float64) ([]*Delivery, error) {
	args := m.Called(ctx, latitude, longitude)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Delivery), args.Error(1)
}

func (m *MockService) AcceptDelivery(ctx interface{}, deliveryID, driverID uuid.UUID) (*DeliveryResponse, error) {
	args := m.Called(ctx, deliveryID, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DeliveryResponse), args.Error(1)
}

func (m *MockService) ConfirmPickup(ctx interface{}, deliveryID, driverID uuid.UUID, req *ConfirmPickupRequest) error {
	args := m.Called(ctx, deliveryID, driverID, req)
	return args.Error(0)
}

func (m *MockService) UpdateStatus(ctx interface{}, deliveryID, driverID uuid.UUID, status DeliveryStatus) error {
	args := m.Called(ctx, deliveryID, driverID, status)
	return args.Error(0)
}

func (m *MockService) ConfirmDelivery(ctx interface{}, deliveryID, driverID uuid.UUID, req *ConfirmDeliveryRequest) error {
	args := m.Called(ctx, deliveryID, driverID, req)
	return args.Error(0)
}

func (m *MockService) ReturnDelivery(ctx interface{}, deliveryID, driverID uuid.UUID, reason string) error {
	args := m.Called(ctx, deliveryID, driverID, reason)
	return args.Error(0)
}

func (m *MockService) GetActiveDelivery(ctx interface{}, driverID uuid.UUID) (*DeliveryResponse, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DeliveryResponse), args.Error(1)
}

func (m *MockService) GetDriverDeliveries(ctx interface{}, driverID uuid.UUID, limit, offset int) ([]*Delivery, int64, error) {
	args := m.Called(ctx, driverID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*Delivery), args.Get(1).(int64), args.Error(2)
}

func (m *MockService) UpdateStopStatus(ctx interface{}, deliveryID, stopID, driverID uuid.UUID, status string, photoURL *string) error {
	args := m.Called(ctx, deliveryID, stopID, driverID, status, photoURL)
	return args.Error(0)
}

// ========================================
// TESTABLE HANDLER (MockableHandler)
// ========================================

// MockableHandler provides testable handler methods with mock service injection
type MockableHandler struct {
	service *MockService
}

func NewMockableHandler(mockService *MockService) *MockableHandler {
	return &MockableHandler{service: mockService}
}

// GetEstimate handles fare estimation
func (h *MockableHandler) GetEstimate(c *gin.Context) {
	var req DeliveryEstimateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	estimate, err := h.service.GetEstimate(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get estimate")
		return
	}

	common.SuccessResponse(c, estimate)
}

// CreateDelivery handles delivery creation
func (h *MockableHandler) CreateDelivery(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateDeliveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	response, err := h.service.CreateDelivery(c.Request.Context(), userID.(uuid.UUID), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create delivery")
		return
	}

	common.CreatedResponse(c, response)
}

// GetDelivery retrieves a delivery
func (h *MockableHandler) GetDelivery(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	deliveryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid delivery ID")
		return
	}

	response, err := h.service.GetDelivery(c.Request.Context(), userID.(uuid.UUID), deliveryID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get delivery")
		return
	}

	common.SuccessResponse(c, response)
}

// GetMyDeliveries lists sender's deliveries
func (h *MockableHandler) GetMyDeliveries(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := 20
	offset := 0
	var filters DeliveryListFilters

	deliveries, total, err := h.service.GetMyDeliveries(c.Request.Context(), userID.(uuid.UUID), &filters, limit, offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list deliveries")
		return
	}

	if deliveries == nil {
		deliveries = []*Delivery{}
	}

	common.SuccessResponseWithMeta(c, gin.H{"deliveries": deliveries}, &common.Meta{Limit: limit, Offset: offset, Total: total})
}

// CancelDelivery cancels a delivery
func (h *MockableHandler) CancelDelivery(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	deliveryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid delivery ID")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	if err := h.service.CancelDelivery(c.Request.Context(), deliveryID, userID.(uuid.UUID), req.Reason); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to cancel delivery")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Delivery cancelled"})
}

// RateDelivery rates a completed delivery
func (h *MockableHandler) RateDelivery(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	deliveryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid delivery ID")
		return
	}

	var req RateDeliveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.RateDelivery(c.Request.Context(), deliveryID, userID.(uuid.UUID), &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to rate delivery")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Delivery rated"})
}

// GetStats returns delivery statistics
func (h *MockableHandler) GetStats(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	stats, err := h.service.GetStats(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get stats")
		return
	}

	common.SuccessResponse(c, stats)
}

// TrackDelivery handles public tracking
func (h *MockableHandler) TrackDelivery(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "tracking code required")
		return
	}

	response, err := h.service.TrackDelivery(c.Request.Context(), code)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "delivery not found")
		return
	}

	common.SuccessResponse(c, response)
}

// GetAvailableDeliveries lists deliveries for drivers
func (h *MockableHandler) GetAvailableDeliveries(c *gin.Context) {
	latStr := c.Query("latitude")
	lngStr := c.Query("longitude")
	if latStr == "" || lngStr == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "latitude and longitude are required")
		return
	}

	var latitude, longitude float64
	if _, err := parseFloat(latStr, &latitude); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid latitude")
		return
	}
	if _, err := parseFloat(lngStr, &longitude); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid longitude")
		return
	}

	deliveries, err := h.service.GetAvailableDeliveries(c.Request.Context(), latitude, longitude)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get available deliveries")
		return
	}

	if deliveries == nil {
		deliveries = []*Delivery{}
	}

	common.SuccessResponse(c, gin.H{"deliveries": deliveries, "count": len(deliveries)})
}

// AcceptDelivery lets a driver accept a delivery
func (h *MockableHandler) AcceptDelivery(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	deliveryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid delivery ID")
		return
	}

	response, err := h.service.AcceptDelivery(c.Request.Context(), deliveryID, driverID.(uuid.UUID))
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to accept delivery")
		return
	}

	common.SuccessResponse(c, response)
}

// ConfirmPickup marks package as picked up
func (h *MockableHandler) ConfirmPickup(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	deliveryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid delivery ID")
		return
	}

	var req ConfirmPickupRequest
	_ = c.ShouldBindJSON(&req)

	if err := h.service.ConfirmPickup(c.Request.Context(), deliveryID, driverID.(uuid.UUID), &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to confirm pickup")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Pickup confirmed"})
}

// UpdateStatus updates delivery status
func (h *MockableHandler) UpdateStatus(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	deliveryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid delivery ID")
		return
	}

	var req struct {
		Status DeliveryStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.UpdateStatus(c.Request.Context(), deliveryID, driverID.(uuid.UUID), req.Status); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update status")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Status updated"})
}

// ConfirmDelivery marks delivery as complete with proof
func (h *MockableHandler) ConfirmDelivery(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	deliveryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid delivery ID")
		return
	}

	var req ConfirmDeliveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.ConfirmDelivery(c.Request.Context(), deliveryID, driverID.(uuid.UUID), &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to confirm delivery")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Delivery confirmed"})
}

// ReturnDelivery returns package to sender
func (h *MockableHandler) ReturnDelivery(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	deliveryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid delivery ID")
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "reason is required")
		return
	}

	if err := h.service.ReturnDelivery(c.Request.Context(), deliveryID, driverID.(uuid.UUID), req.Reason); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to return delivery")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Return initiated"})
}

// GetActiveDelivery gets driver's current active delivery
func (h *MockableHandler) GetActiveDelivery(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	response, err := h.service.GetActiveDelivery(c.Request.Context(), driverID.(uuid.UUID))
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get active delivery")
		return
	}

	common.SuccessResponse(c, response)
}

// GetDriverDeliveries lists driver's delivery history
func (h *MockableHandler) GetDriverDeliveries(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := 20
	offset := 0

	deliveries, total, err := h.service.GetDriverDeliveries(c.Request.Context(), driverID.(uuid.UUID), limit, offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list deliveries")
		return
	}

	if deliveries == nil {
		deliveries = []*Delivery{}
	}

	common.SuccessResponseWithMeta(c, gin.H{"deliveries": deliveries}, &common.Meta{Limit: limit, Offset: offset, Total: total})
}

// UpdateStopStatus updates an intermediate stop
func (h *MockableHandler) UpdateStopStatus(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	deliveryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid delivery ID")
		return
	}

	stopID, err := uuid.Parse(c.Param("stopId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid stop ID")
		return
	}

	var req struct {
		Status   string  `json:"status" binding:"required"`
		PhotoURL *string `json:"photo_url,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.UpdateStopStatus(c.Request.Context(), deliveryID, stopID, driverID.(uuid.UUID), req.Status, req.PhotoURL); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update stop status")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Stop status updated"})
}

// ========================================
// TEST HELPERS
// ========================================

func parseFloat(s string, f *float64) (bool, error) {
	_, err := json.Marshal(s)
	if err != nil {
		return false, err
	}
	return json.Unmarshal([]byte(s), f) == nil, nil
}

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

func createTestDelivery(senderID uuid.UUID, status DeliveryStatus) *Delivery {
	now := time.Now()
	return &Delivery{
		ID:                 uuid.New(),
		SenderID:           senderID,
		Status:             status,
		Priority:           DeliveryPriorityStandard,
		TrackingCode:       "DLV-ABCDE-12345",
		PickupLatitude:     40.7128,
		PickupLongitude:    -74.0060,
		PickupAddress:      "123 Main St, New York",
		PickupContact:      "John Sender",
		PickupPhone:        "+1234567890",
		DropoffLatitude:    40.7580,
		DropoffLongitude:   -73.9855,
		DropoffAddress:     "456 Broadway, New York",
		RecipientName:      "Jane Recipient",
		RecipientPhone:     "+0987654321",
		PackageSize:        PackageSizeSmall,
		PackageDescription: "Test Package",
		EstimatedDistance:  5.0,
		EstimatedDuration:  15,
		EstimatedFare:      12.50,
		SurgeMultiplier:    1.0,
		RequestedAt:        now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

func createTestDeliveryWithDriver(senderID, driverID uuid.UUID, status DeliveryStatus) *Delivery {
	delivery := createTestDelivery(senderID, status)
	delivery.DriverID = &driverID
	now := time.Now()
	delivery.AcceptedAt = &now
	return delivery
}

func createTestDeliveryResponse(delivery *Delivery) *DeliveryResponse {
	return &DeliveryResponse{
		Delivery: delivery,
		Stops:    []DeliveryStop{},
		Tracking: []DeliveryTracking{},
	}
}

func ptrStr(s string) *string {
	return &s
}

// ============================================================================
// DELIVERY CREATION TESTS
// ============================================================================

func TestHandler_CreateDelivery_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	expectedDelivery := createTestDelivery(userID, DeliveryStatusRequested)

	reqBody := CreateDeliveryRequest{
		PickupLatitude:     40.7128,
		PickupLongitude:    -74.0060,
		PickupAddress:      "123 Main St, New York",
		PickupContact:      "John Sender",
		PickupPhone:        "+1234567890",
		DropoffLatitude:    40.7580,
		DropoffLongitude:   -73.9855,
		DropoffAddress:     "456 Broadway, New York",
		RecipientName:      "Jane Recipient",
		RecipientPhone:     "+0987654321",
		PackageSize:        PackageSizeSmall,
		PackageDescription: "Test Package",
		Priority:           DeliveryPriorityStandard,
	}

	mockService.On("CreateDelivery", mock.Anything, userID, &reqBody).
		Return(createTestDeliveryResponse(expectedDelivery), nil)

	c, w := setupTestContext("POST", "/api/v1/deliveries", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.CreateDelivery(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_CreateDelivery_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := CreateDeliveryRequest{
		PickupLatitude:  40.7128,
		PickupLongitude: -74.0060,
		PickupAddress:   "123 Main St",
		PickupContact:   "John",
		PickupPhone:     "+1234567890",
	}

	c, w := setupTestContext("POST", "/api/v1/deliveries", reqBody)
	// Don't set user context

	handler.CreateDelivery(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_CreateDelivery_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/deliveries", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/deliveries", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleRider)

	handler.CreateDelivery(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateDelivery_InvalidPackageSize(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	reqBody := CreateDeliveryRequest{
		PickupLatitude:     40.7128,
		PickupLongitude:    -74.0060,
		PickupAddress:      "123 Main St",
		PickupContact:      "John",
		PickupPhone:        "+1234567890",
		DropoffLatitude:    40.7580,
		DropoffLongitude:   -73.9855,
		DropoffAddress:     "456 Broadway",
		RecipientName:      "Jane",
		RecipientPhone:     "+0987654321",
		PackageSize:        PackageSize("invalid_size"),
		PackageDescription: "Test",
		Priority:           DeliveryPriorityStandard,
	}

	mockService.On("CreateDelivery", mock.Anything, userID, mock.Anything).
		Return(nil, common.NewBadRequestError("invalid package size", nil))

	c, w := setupTestContext("POST", "/api/v1/deliveries", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.CreateDelivery(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CreateDelivery_ExpressDelivery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	expectedDelivery := createTestDelivery(userID, DeliveryStatusRequested)
	expectedDelivery.Priority = DeliveryPriorityExpress

	reqBody := CreateDeliveryRequest{
		PickupLatitude:     40.7128,
		PickupLongitude:    -74.0060,
		PickupAddress:      "123 Main St",
		PickupContact:      "John",
		PickupPhone:        "+1234567890",
		DropoffLatitude:    40.7580,
		DropoffLongitude:   -73.9855,
		DropoffAddress:     "456 Broadway",
		RecipientName:      "Jane",
		RecipientPhone:     "+0987654321",
		PackageSize:        PackageSizeSmall,
		PackageDescription: "Urgent Package",
		Priority:           DeliveryPriorityExpress,
	}

	mockService.On("CreateDelivery", mock.Anything, userID, &reqBody).
		Return(createTestDeliveryResponse(expectedDelivery), nil)

	c, w := setupTestContext("POST", "/api/v1/deliveries", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.CreateDelivery(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CreateDelivery_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	reqBody := CreateDeliveryRequest{
		PickupLatitude:     40.7128,
		PickupLongitude:    -74.0060,
		PickupAddress:      "123 Main St",
		PickupContact:      "John",
		PickupPhone:        "+1234567890",
		DropoffLatitude:    40.7580,
		DropoffLongitude:   -73.9855,
		DropoffAddress:     "456 Broadway",
		RecipientName:      "Jane",
		RecipientPhone:     "+0987654321",
		PackageSize:        PackageSizeSmall,
		PackageDescription: "Test",
		Priority:           DeliveryPriorityStandard,
	}

	mockService.On("CreateDelivery", mock.Anything, userID, &reqBody).
		Return(nil, errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/deliveries", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.CreateDelivery(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CreateDelivery_AllPackageSizes(t *testing.T) {
	tests := []struct {
		name        string
		packageSize PackageSize
	}{
		{"envelope", PackageSizeEnvelope},
		{"small", PackageSizeSmall},
		{"medium", PackageSizeMedium},
		{"large", PackageSizeLarge},
		{"xlarge", PackageSizeXLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(MockService)
			handler := NewMockableHandler(mockService)

			userID := uuid.New()
			expectedDelivery := createTestDelivery(userID, DeliveryStatusRequested)
			expectedDelivery.PackageSize = tt.packageSize

			reqBody := CreateDeliveryRequest{
				PickupLatitude:     40.7128,
				PickupLongitude:    -74.0060,
				PickupAddress:      "123 Main St",
				PickupContact:      "John",
				PickupPhone:        "+1234567890",
				DropoffLatitude:    40.7580,
				DropoffLongitude:   -73.9855,
				DropoffAddress:     "456 Broadway",
				RecipientName:      "Jane",
				RecipientPhone:     "+0987654321",
				PackageSize:        tt.packageSize,
				PackageDescription: "Test",
				Priority:           DeliveryPriorityStandard,
			}

			mockService.On("CreateDelivery", mock.Anything, userID, &reqBody).
				Return(createTestDeliveryResponse(expectedDelivery), nil)

			c, w := setupTestContext("POST", "/api/v1/deliveries", reqBody)
			setUserContext(c, userID, models.RoleRider)

			handler.CreateDelivery(c)

			assert.Equal(t, http.StatusCreated, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_CreateDelivery_WithMultipleStops(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	expectedDelivery := createTestDelivery(userID, DeliveryStatusRequested)

	reqBody := CreateDeliveryRequest{
		PickupLatitude:     40.7128,
		PickupLongitude:    -74.0060,
		PickupAddress:      "123 Main St",
		PickupContact:      "John",
		PickupPhone:        "+1234567890",
		DropoffLatitude:    40.7580,
		DropoffLongitude:   -73.9855,
		DropoffAddress:     "456 Broadway",
		RecipientName:      "Jane",
		RecipientPhone:     "+0987654321",
		PackageSize:        PackageSizeSmall,
		PackageDescription: "Multi-stop delivery",
		Priority:           DeliveryPriorityStandard,
		Stops: []StopInput{
			{
				Latitude:     40.7300,
				Longitude:    -73.9950,
				Address:      "First Stop",
				ContactName:  "Stop 1 Contact",
				ContactPhone: "+1111111111",
			},
			{
				Latitude:     40.7400,
				Longitude:    -73.9900,
				Address:      "Second Stop",
				ContactName:  "Stop 2 Contact",
				ContactPhone: "+2222222222",
			},
		},
	}

	mockService.On("CreateDelivery", mock.Anything, userID, &reqBody).
		Return(createTestDeliveryResponse(expectedDelivery), nil)

	c, w := setupTestContext("POST", "/api/v1/deliveries", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.CreateDelivery(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// STATUS MANAGEMENT TESTS
// ============================================================================

func TestHandler_UpdateStatus_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	mockService.On("UpdateStatus", mock.Anything, deliveryID, driverID, DeliveryStatusInTransit).
		Return(nil)

	reqBody := map[string]interface{}{
		"status": "in_transit",
	}

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/status", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateStatus_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	deliveryID := uuid.New()

	reqBody := map[string]interface{}{
		"status": "in_transit",
	}

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/status", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	// Don't set user context

	handler.UpdateStatus(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_UpdateStatus_InvalidDeliveryID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()

	reqBody := map[string]interface{}{
		"status": "in_transit",
	}

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/invalid-uuid/status", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid delivery ID")
}

func TestHandler_UpdateStatus_MissingStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/status", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateStatus_InvalidTransition(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	mockService.On("UpdateStatus", mock.Anything, deliveryID, driverID, DeliveryStatusDelivered).
		Return(common.NewBadRequestError("cannot transition from requested to delivered", nil))

	reqBody := map[string]interface{}{
		"status": "delivered",
	}

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/status", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateStatus_AllValidTransitions(t *testing.T) {
	tests := []struct {
		name       string
		fromStatus DeliveryStatus
		toStatus   DeliveryStatus
	}{
		{"requested_to_accepted", DeliveryStatusRequested, DeliveryStatusAccepted},
		{"accepted_to_picking_up", DeliveryStatusAccepted, DeliveryStatusPickingUp},
		{"picking_up_to_picked_up", DeliveryStatusPickingUp, DeliveryStatusPickedUp},
		{"picked_up_to_in_transit", DeliveryStatusPickedUp, DeliveryStatusInTransit},
		{"in_transit_to_arrived", DeliveryStatusInTransit, DeliveryStatusArrived},
		{"arrived_to_delivered", DeliveryStatusArrived, DeliveryStatusDelivered},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(MockService)
			handler := NewMockableHandler(mockService)

			driverID := uuid.New()
			deliveryID := uuid.New()

			mockService.On("UpdateStatus", mock.Anything, deliveryID, driverID, tt.toStatus).
				Return(nil)

			reqBody := map[string]interface{}{
				"status": string(tt.toStatus),
			}

			c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/status", reqBody)
			c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
			setUserContext(c, driverID, models.RoleDriver)

			handler.UpdateStatus(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_UpdateStatus_NotYourDelivery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	mockService.On("UpdateStatus", mock.Anything, deliveryID, driverID, DeliveryStatusInTransit).
		Return(common.NewForbiddenError("not your delivery"))

	reqBody := map[string]interface{}{
		"status": "in_transit",
	}

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/status", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateStatus(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateStatus_DeliveryNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	mockService.On("UpdateStatus", mock.Anything, deliveryID, driverID, DeliveryStatusInTransit).
		Return(common.NewNotFoundError("delivery not found", nil))

	reqBody := map[string]interface{}{
		"status": "in_transit",
	}

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/status", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateStatus(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// PROOF OF DELIVERY TESTS
// ============================================================================

func TestHandler_ConfirmDelivery_WithPhoto(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	photoURL := "https://storage.example.com/proof/photo123.jpg"
	reqBody := ConfirmDeliveryRequest{
		ProofType: ProofTypePhoto,
		PhotoURL:  &photoURL,
	}

	mockService.On("ConfirmDelivery", mock.Anything, deliveryID, driverID, &reqBody).
		Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/deliver", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.ConfirmDelivery(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_ConfirmDelivery_WithSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	signatureURL := "https://storage.example.com/signatures/sig123.png"
	reqBody := ConfirmDeliveryRequest{
		ProofType:    ProofTypeSignature,
		SignatureURL: &signatureURL,
	}

	mockService.On("ConfirmDelivery", mock.Anything, deliveryID, driverID, &reqBody).
		Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/deliver", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.ConfirmDelivery(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ConfirmDelivery_WithPIN(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	pin := "1234"
	reqBody := ConfirmDeliveryRequest{
		ProofType: ProofTypePIN,
		PIN:       &pin,
	}

	mockService.On("ConfirmDelivery", mock.Anything, deliveryID, driverID, &reqBody).
		Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/deliver", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.ConfirmDelivery(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ConfirmDelivery_InvalidPIN(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	pin := "9999"
	reqBody := ConfirmDeliveryRequest{
		ProofType: ProofTypePIN,
		PIN:       &pin,
	}

	mockService.On("ConfirmDelivery", mock.Anything, deliveryID, driverID, &reqBody).
		Return(common.NewBadRequestError("incorrect delivery PIN", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/deliver", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.ConfirmDelivery(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ConfirmDelivery_ContactlessDelivery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	photoURL := "https://storage.example.com/proof/contactless123.jpg"
	notes := "Left at front door"
	reqBody := ConfirmDeliveryRequest{
		ProofType: ProofTypeContactless,
		PhotoURL:  &photoURL,
		Notes:     &notes,
	}

	mockService.On("ConfirmDelivery", mock.Anything, deliveryID, driverID, &reqBody).
		Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/deliver", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.ConfirmDelivery(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ConfirmDelivery_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	deliveryID := uuid.New()

	pin := "1234"
	reqBody := ConfirmDeliveryRequest{
		ProofType: ProofTypePIN,
		PIN:       &pin,
	}

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/deliver", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	// Don't set user context

	handler.ConfirmDelivery(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_ConfirmDelivery_InvalidDeliveryID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()

	reqBody := ConfirmDeliveryRequest{
		ProofType: ProofTypePhoto,
	}

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/invalid-uuid/deliver", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.ConfirmDelivery(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ConfirmDelivery_MissingProofType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/deliver", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.ConfirmDelivery(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ConfirmDelivery_RequiresSignatureButPhotoProvided(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	photoURL := "https://storage.example.com/proof/photo.jpg"
	reqBody := ConfirmDeliveryRequest{
		ProofType: ProofTypePhoto,
		PhotoURL:  &photoURL,
	}

	mockService.On("ConfirmDelivery", mock.Anything, deliveryID, driverID, &reqBody).
		Return(common.NewBadRequestError("this delivery requires a signature", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/deliver", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.ConfirmDelivery(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ConfirmDelivery_PhotoRequiredButMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	reqBody := ConfirmDeliveryRequest{
		ProofType: ProofTypePhoto,
		// PhotoURL missing
	}

	mockService.On("ConfirmDelivery", mock.Anything, deliveryID, driverID, &reqBody).
		Return(common.NewBadRequestError("photo proof is required", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/deliver", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.ConfirmDelivery(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ConfirmDelivery_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	photoURL := "https://storage.example.com/proof/photo.jpg"
	reqBody := ConfirmDeliveryRequest{
		ProofType: ProofTypePhoto,
		PhotoURL:  &photoURL,
	}

	mockService.On("ConfirmDelivery", mock.Anything, deliveryID, driverID, &reqBody).
		Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/deliver", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.ConfirmDelivery(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// TRACKING TESTS
// ============================================================================

func TestHandler_GetDelivery_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	deliveryID := uuid.New()
	expectedDelivery := createTestDelivery(userID, DeliveryStatusInTransit)
	expectedDelivery.ID = deliveryID

	mockService.On("GetDelivery", mock.Anything, userID, deliveryID).
		Return(createTestDeliveryResponse(expectedDelivery), nil)

	c, w := setupTestContext("GET", "/api/v1/deliveries/"+deliveryID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetDelivery(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetDelivery_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	deliveryID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/deliveries/"+deliveryID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	// Don't set user context

	handler.GetDelivery(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetDelivery_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/deliveries/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetDelivery(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid delivery ID")
}

func TestHandler_GetDelivery_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	deliveryID := uuid.New()

	mockService.On("GetDelivery", mock.Anything, userID, deliveryID).
		Return(nil, common.NewNotFoundError("delivery not found", nil))

	c, w := setupTestContext("GET", "/api/v1/deliveries/"+deliveryID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetDelivery(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetDelivery_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	deliveryID := uuid.New()

	mockService.On("GetDelivery", mock.Anything, userID, deliveryID).
		Return(nil, common.NewForbiddenError("not authorized to view this delivery"))

	c, w := setupTestContext("GET", "/api/v1/deliveries/"+deliveryID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetDelivery(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_TrackDelivery_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	trackingCode := "DLV-ABCDE-12345"
	senderID := uuid.New()
	expectedDelivery := createTestDelivery(senderID, DeliveryStatusInTransit)

	mockService.On("TrackDelivery", mock.Anything, trackingCode).
		Return(createTestDeliveryResponse(expectedDelivery), nil)

	c, w := setupTestContext("GET", "/api/v1/deliveries/track/"+trackingCode, nil)
	c.Params = gin.Params{{Key: "code", Value: trackingCode}}

	handler.TrackDelivery(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_TrackDelivery_EmptyCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/deliveries/track/", nil)
	c.Params = gin.Params{{Key: "code", Value: ""}}

	handler.TrackDelivery(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TrackDelivery_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	trackingCode := "INVALID-CODE"

	mockService.On("TrackDelivery", mock.Anything, trackingCode).
		Return(nil, common.NewNotFoundError("delivery not found", nil))

	c, w := setupTestContext("GET", "/api/v1/deliveries/track/"+trackingCode, nil)
	c.Params = gin.Params{{Key: "code", Value: trackingCode}}

	handler.TrackDelivery(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetMyDeliveries_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	deliveries := []*Delivery{
		createTestDelivery(userID, DeliveryStatusDelivered),
		createTestDelivery(userID, DeliveryStatusInTransit),
	}

	mockService.On("GetMyDeliveries", mock.Anything, userID, mock.Anything, 20, 0).
		Return(deliveries, int64(2), nil)

	c, w := setupTestContext("GET", "/api/v1/deliveries", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetMyDeliveries(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetMyDeliveries_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/deliveries", nil)
	// Don't set user context

	handler.GetMyDeliveries(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetMyDeliveries_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("GetMyDeliveries", mock.Anything, userID, mock.Anything, 20, 0).
		Return(nil, int64(0), nil)

	c, w := setupTestContext("GET", "/api/v1/deliveries", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetMyDeliveries(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetMyDeliveries_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("GetMyDeliveries", mock.Anything, userID, mock.Anything, 20, 0).
		Return(nil, int64(0), errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/deliveries", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetMyDeliveries(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// DRIVER ENDPOINT TESTS
// ============================================================================

func TestHandler_AcceptDelivery_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	senderID := uuid.New()
	deliveryID := uuid.New()
	expectedDelivery := createTestDeliveryWithDriver(senderID, driverID, DeliveryStatusAccepted)
	expectedDelivery.ID = deliveryID

	mockService.On("AcceptDelivery", mock.Anything, deliveryID, driverID).
		Return(createTestDeliveryResponse(expectedDelivery), nil)

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/accept", nil)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.AcceptDelivery(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_AcceptDelivery_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	deliveryID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/accept", nil)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	// Don't set user context

	handler.AcceptDelivery(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_AcceptDelivery_AlreadyAccepted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	mockService.On("AcceptDelivery", mock.Anything, deliveryID, driverID).
		Return(nil, common.NewConflictError("delivery already accepted by another driver"))

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/accept", nil)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.AcceptDelivery(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_AcceptDelivery_DriverHasActiveDelivery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	mockService.On("AcceptDelivery", mock.Anything, deliveryID, driverID).
		Return(nil, common.NewConflictError("you already have an active delivery"))

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/accept", nil)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.AcceptDelivery(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ConfirmPickup_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	photoURL := "https://storage.example.com/pickup/photo.jpg"
	reqBody := ConfirmPickupRequest{
		PhotoURL: &photoURL,
	}

	mockService.On("ConfirmPickup", mock.Anything, deliveryID, driverID, &reqBody).
		Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/pickup", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.ConfirmPickup(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_ConfirmPickup_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	deliveryID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/pickup", nil)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	// Don't set user context

	handler.ConfirmPickup(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_ConfirmPickup_WrongStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	reqBody := ConfirmPickupRequest{}

	mockService.On("ConfirmPickup", mock.Anything, deliveryID, driverID, &reqBody).
		Return(common.NewBadRequestError("delivery must be in picking_up or accepted status", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/pickup", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.ConfirmPickup(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetAvailableDeliveries_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	senderID := uuid.New()
	deliveries := []*Delivery{
		createTestDelivery(senderID, DeliveryStatusRequested),
		createTestDelivery(senderID, DeliveryStatusRequested),
	}

	mockService.On("GetAvailableDeliveries", mock.Anything, 40.7128, -74.0060).
		Return(deliveries, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/deliveries/available?latitude=40.7128&longitude=-74.0060", nil)
	c.Request.URL.RawQuery = "latitude=40.7128&longitude=-74.0060"
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetAvailableDeliveries(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetAvailableDeliveries_MissingLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/driver/deliveries/available?longitude=-74.0060", nil)
	c.Request.URL.RawQuery = "longitude=-74.0060"
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetAvailableDeliveries(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetAvailableDeliveries_MissingLongitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/driver/deliveries/available?latitude=40.7128", nil)
	c.Request.URL.RawQuery = "latitude=40.7128"
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetAvailableDeliveries(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetActiveDelivery_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	senderID := uuid.New()
	expectedDelivery := createTestDeliveryWithDriver(senderID, driverID, DeliveryStatusInTransit)

	mockService.On("GetActiveDelivery", mock.Anything, driverID).
		Return(createTestDeliveryResponse(expectedDelivery), nil)

	c, w := setupTestContext("GET", "/api/v1/driver/deliveries/active", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetActiveDelivery(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetActiveDelivery_NoActiveDelivery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()

	mockService.On("GetActiveDelivery", mock.Anything, driverID).
		Return(nil, common.NewNotFoundError("no active delivery", nil))

	c, w := setupTestContext("GET", "/api/v1/driver/deliveries/active", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetActiveDelivery(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetActiveDelivery_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/driver/deliveries/active", nil)
	// Don't set user context

	handler.GetActiveDelivery(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetDriverDeliveries_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	senderID := uuid.New()
	deliveries := []*Delivery{
		createTestDeliveryWithDriver(senderID, driverID, DeliveryStatusDelivered),
		createTestDeliveryWithDriver(senderID, driverID, DeliveryStatusDelivered),
	}

	mockService.On("GetDriverDeliveries", mock.Anything, driverID, 20, 0).
		Return(deliveries, int64(2), nil)

	c, w := setupTestContext("GET", "/api/v1/driver/deliveries", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetDriverDeliveries(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_ReturnDelivery_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	reqBody := map[string]interface{}{
		"reason": "Recipient not available",
	}

	mockService.On("ReturnDelivery", mock.Anything, deliveryID, driverID, "Recipient not available").
		Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/return", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.ReturnDelivery(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_ReturnDelivery_MissingReason(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/return", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.ReturnDelivery(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ReturnDelivery_WrongStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	reqBody := map[string]interface{}{
		"reason": "Recipient not available",
	}

	mockService.On("ReturnDelivery", mock.Anything, deliveryID, driverID, "Recipient not available").
		Return(common.NewBadRequestError("can only return packages in transit or arrived", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/return", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.ReturnDelivery(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// CANCEL & RATE TESTS
// ============================================================================

func TestHandler_CancelDelivery_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	deliveryID := uuid.New()

	reqBody := map[string]interface{}{
		"reason": "Changed my mind",
	}

	mockService.On("CancelDelivery", mock.Anything, deliveryID, userID, "Changed my mind").
		Return(nil)

	c, w := setupTestContext("POST", "/api/v1/deliveries/"+deliveryID.String()+"/cancel", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.CancelDelivery(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_CancelDelivery_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	deliveryID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/deliveries/"+deliveryID.String()+"/cancel", nil)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	// Don't set user context

	handler.CancelDelivery(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CancelDelivery_CannotCancelAfterPickup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	deliveryID := uuid.New()

	reqBody := map[string]interface{}{
		"reason": "Changed my mind",
	}

	mockService.On("CancelDelivery", mock.Anything, deliveryID, userID, "Changed my mind").
		Return(common.NewBadRequestError("cannot cancel after package pickup - use return instead", nil))

	c, w := setupTestContext("POST", "/api/v1/deliveries/"+deliveryID.String()+"/cancel", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, userID, models.RoleDriver)

	handler.CancelDelivery(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_RateDelivery_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	deliveryID := uuid.New()

	feedback := "Great delivery service!"
	reqBody := RateDeliveryRequest{
		Rating:   5,
		Feedback: &feedback,
	}

	mockService.On("RateDelivery", mock.Anything, deliveryID, userID, &reqBody).
		Return(nil)

	c, w := setupTestContext("POST", "/api/v1/deliveries/"+deliveryID.String()+"/rate", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.RateDelivery(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_RateDelivery_InvalidRating(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	deliveryID := uuid.New()

	// Rating outside 1-5 range should fail validation
	reqBody := map[string]interface{}{
		"rating": 10,
	}

	c, w := setupTestContext("POST", "/api/v1/deliveries/"+deliveryID.String()+"/rate", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.RateDelivery(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RateDelivery_NotDelivered(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	deliveryID := uuid.New()

	reqBody := RateDeliveryRequest{
		Rating: 5,
	}

	mockService.On("RateDelivery", mock.Anything, deliveryID, userID, &reqBody).
		Return(common.NewBadRequestError("can only rate completed deliveries", nil))

	c, w := setupTestContext("POST", "/api/v1/deliveries/"+deliveryID.String()+"/rate", reqBody)
	c.Params = gin.Params{{Key: "id", Value: deliveryID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.RateDelivery(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// STATS AND ESTIMATE TESTS
// ============================================================================

func TestHandler_GetStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	expectedStats := &DeliveryStats{
		TotalDeliveries:    25,
		CompletedCount:     20,
		CancelledCount:     3,
		InProgressCount:    2,
		AverageRating:      4.5,
		TotalSpent:         250.0,
		AverageDeliveryMin: 30.0,
	}

	mockService.On("GetStats", mock.Anything, userID).
		Return(expectedStats, nil)

	c, w := setupTestContext("GET", "/api/v1/deliveries/stats", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetStats_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/deliveries/stats", nil)
	// Don't set user context

	handler.GetStats(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetEstimate_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := DeliveryEstimateRequest{
		PickupLatitude:   40.7128,
		PickupLongitude:  -74.0060,
		DropoffLatitude:  40.7580,
		DropoffLongitude: -73.9855,
		PackageSize:      PackageSizeSmall,
		Priority:         DeliveryPriorityStandard,
	}

	expectedEstimate := &DeliveryEstimateResponse{
		EstimatedDistance: 5.5,
		EstimatedDuration: 20,
		BaseFare:          6.60,
		SizeSurcharge:     1.0,
		PrioritySurcharge: 0.0,
		SurgeMultiplier:   1.0,
		TotalEstimate:     7.60,
		Currency:          "USD",
		Priority:          DeliveryPriorityStandard,
		PackageSize:       PackageSizeSmall,
	}

	mockService.On("GetEstimate", mock.Anything, &reqBody).
		Return(expectedEstimate, nil)

	c, w := setupTestContext("POST", "/api/v1/deliveries/estimate", reqBody)

	handler.GetEstimate(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetEstimate_ExpressPriority(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	reqBody := DeliveryEstimateRequest{
		PickupLatitude:   40.7128,
		PickupLongitude:  -74.0060,
		DropoffLatitude:  40.7580,
		DropoffLongitude: -73.9855,
		PackageSize:      PackageSizeMedium,
		Priority:         DeliveryPriorityExpress,
	}

	expectedEstimate := &DeliveryEstimateResponse{
		EstimatedDistance: 5.5,
		EstimatedDuration: 15,
		BaseFare:          6.60,
		SizeSurcharge:     3.0,
		PrioritySurcharge: 3.30,
		SurgeMultiplier:   1.0,
		TotalEstimate:     12.90,
		Currency:          "USD",
		Priority:          DeliveryPriorityExpress,
		PackageSize:       PackageSizeMedium,
	}

	mockService.On("GetEstimate", mock.Anything, &reqBody).
		Return(expectedEstimate, nil)

	c, w := setupTestContext("POST", "/api/v1/deliveries/estimate", reqBody)

	handler.GetEstimate(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetEstimate_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/deliveries/estimate", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/deliveries/estimate", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.GetEstimate(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// STOP STATUS TESTS
// ============================================================================

func TestHandler_UpdateStopStatus_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()
	stopID := uuid.New()

	photoURL := "https://storage.example.com/stops/photo.jpg"
	reqBody := map[string]interface{}{
		"status":    "completed",
		"photo_url": photoURL,
	}

	mockService.On("UpdateStopStatus", mock.Anything, deliveryID, stopID, driverID, "completed", &photoURL).
		Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/stops/"+stopID.String()+"/status", reqBody)
	c.Params = gin.Params{
		{Key: "id", Value: deliveryID.String()},
		{Key: "stopId", Value: stopID.String()},
	}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateStopStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateStopStatus_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	deliveryID := uuid.New()
	stopID := uuid.New()

	reqBody := map[string]interface{}{
		"status": "completed",
	}

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/stops/"+stopID.String()+"/status", reqBody)
	c.Params = gin.Params{
		{Key: "id", Value: deliveryID.String()},
		{Key: "stopId", Value: stopID.String()},
	}
	// Don't set user context

	handler.UpdateStopStatus(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_UpdateStopStatus_InvalidDeliveryID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	stopID := uuid.New()

	reqBody := map[string]interface{}{
		"status": "completed",
	}

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/invalid-uuid/stops/"+stopID.String()+"/status", reqBody)
	c.Params = gin.Params{
		{Key: "id", Value: "invalid-uuid"},
		{Key: "stopId", Value: stopID.String()},
	}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateStopStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateStopStatus_InvalidStopID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()

	reqBody := map[string]interface{}{
		"status": "completed",
	}

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/stops/invalid-uuid/status", reqBody)
	c.Params = gin.Params{
		{Key: "id", Value: deliveryID.String()},
		{Key: "stopId", Value: "invalid-uuid"},
	}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateStopStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateStopStatus_MissingStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	handler := NewMockableHandler(mockService)

	driverID := uuid.New()
	deliveryID := uuid.New()
	stopID := uuid.New()

	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/driver/deliveries/"+deliveryID.String()+"/stops/"+stopID.String()+"/status", reqBody)
	c.Params = gin.Params{
		{Key: "id", Value: deliveryID.String()},
		{Key: "stopId", Value: stopID.String()},
	}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateStopStatus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
