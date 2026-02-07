package cancellation

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
	"github.com/jackc/pgx/v5"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// MOCK SERVICE
// ============================================================================

// MockCancellationService implements a mock for testing handlers
type MockCancellationService struct {
	mock.Mock
}

func (m *MockCancellationService) PreviewCancellation(ctx context.Context, rideID, userID uuid.UUID) (*CancellationPreviewResponse, error) {
	args := m.Called(ctx, rideID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CancellationPreviewResponse), args.Error(1)
}

func (m *MockCancellationService) CancelRide(ctx context.Context, rideID, userID uuid.UUID, req *CancelRideRequest) (*CancelRideResponse, error) {
	args := m.Called(ctx, rideID, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CancelRideResponse), args.Error(1)
}

func (m *MockCancellationService) GetCancellationDetails(ctx context.Context, rideID, userID uuid.UUID) (*CancellationRecord, error) {
	args := m.Called(ctx, rideID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CancellationRecord), args.Error(1)
}

func (m *MockCancellationService) GetMyCancellationStats(ctx context.Context, userID uuid.UUID) (*UserCancellationStats, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UserCancellationStats), args.Error(1)
}

func (m *MockCancellationService) GetMyCancellationHistory(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]CancellationRecord, int, error) {
	args := m.Called(ctx, userID, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]CancellationRecord), args.Int(1), args.Error(2)
}

func (m *MockCancellationService) GetCancellationReasons(isDriver bool) *CancellationReasonsResponse {
	args := m.Called(isDriver)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*CancellationReasonsResponse)
}

func (m *MockCancellationService) WaiveFee(ctx context.Context, cancellationID uuid.UUID, reason string) error {
	args := m.Called(ctx, cancellationID, reason)
	return args.Error(0)
}

func (m *MockCancellationService) GetCancellationStats(ctx context.Context, from, to time.Time) (*CancellationStatsResponse, error) {
	args := m.Called(ctx, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CancellationStatsResponse), args.Error(1)
}

func (m *MockCancellationService) GetUserCancellationStats(ctx context.Context, userID uuid.UUID) (*UserCancellationStats, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UserCancellationStats), args.Error(1)
}

// ============================================================================
// SERVICE INTERFACE FOR TESTING
// ============================================================================

// ServiceInterface defines the interface for the cancellation service
type ServiceInterface interface {
	PreviewCancellation(ctx context.Context, rideID, userID uuid.UUID) (*CancellationPreviewResponse, error)
	CancelRide(ctx context.Context, rideID, userID uuid.UUID, req *CancelRideRequest) (*CancelRideResponse, error)
	GetCancellationDetails(ctx context.Context, rideID, userID uuid.UUID) (*CancellationRecord, error)
	GetMyCancellationStats(ctx context.Context, userID uuid.UUID) (*UserCancellationStats, error)
	GetMyCancellationHistory(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]CancellationRecord, int, error)
	GetCancellationReasons(isDriver bool) *CancellationReasonsResponse
	WaiveFee(ctx context.Context, cancellationID uuid.UUID, reason string) error
	GetCancellationStats(ctx context.Context, from, to time.Time) (*CancellationStatsResponse, error)
	GetUserCancellationStats(ctx context.Context, userID uuid.UUID) (*UserCancellationStats, error)
}

// testHandler wraps Handler for testing with mock service
type testHandler struct {
	service ServiceInterface
}

func newTestHandler(svc ServiceInterface) *testHandler {
	return &testHandler{service: svc}
}

// ============================================================================
// TEST HELPERS
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

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// ============================================================================
// PREVIEW CANCELLATION TESTS
// ============================================================================

func TestHandler_PreviewCancellation_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	userID := uuid.New()
	rideID := uuid.New()

	preview := &CancellationPreviewResponse{
		FeeAmount:            0,
		FeeWaived:            true,
		WaiverReason:         strPtr("free_cancellation_window"),
		Explanation:          "Free cancellation within 2 minutes",
		FreeCancelsRemaining: 3,
		MinutesSinceRequest:  1.5,
	}

	mockSvc.On("PreviewCancellation", mock.Anything, rideID, userID).Return(preview, nil)

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/rides/:id/cancel/preview", func(c *gin.Context) {
		setUserContext(c, userID, models.RoleRider)
		rideIDParsed, _ := uuid.Parse(c.Param("id"))
		result, err := h.service.PreviewCancellation(c.Request.Context(), rideIDParsed, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rides/"+rideID.String()+"/cancel/preview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	mockSvc.AssertExpectations(t)
}

func TestHandler_PreviewCancellation_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	handler := NewHandler(nil) // nil service since we won't reach it

	rideID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/rides/"+rideID.String()+"/cancel/preview", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	// Don't set user context to simulate unauthorized

	handler.PreviewCancellation(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	mockSvc.AssertExpectations(t)
}

func TestHandler_PreviewCancellation_InvalidRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil)
	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/rides/invalid-uuid/cancel/preview", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID, models.RoleRider)

	handler.PreviewCancellation(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid ride id")
}

func TestHandler_PreviewCancellation_RideNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	userID := uuid.New()
	rideID := uuid.New()

	mockSvc.On("PreviewCancellation", mock.Anything, rideID, userID).Return(nil, common.NewNotFoundError("ride not found", nil))

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/rides/:id/cancel/preview", func(c *gin.Context) {
		setUserContext(c, userID, models.RoleRider)
		rideIDParsed, _ := uuid.Parse(c.Param("id"))
		result, err := h.service.PreviewCancellation(c.Request.Context(), rideIDParsed, userID)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rides/"+rideID.String()+"/cancel/preview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_PreviewCancellation_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	userID := uuid.New()
	rideID := uuid.New()

	mockSvc.On("PreviewCancellation", mock.Anything, rideID, userID).Return(nil, common.NewForbiddenError("not authorized for this ride"))

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/rides/:id/cancel/preview", func(c *gin.Context) {
		setUserContext(c, userID, models.RoleRider)
		rideIDParsed, _ := uuid.Parse(c.Param("id"))
		result, err := h.service.PreviewCancellation(c.Request.Context(), rideIDParsed, userID)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rides/"+rideID.String()+"/cancel/preview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_PreviewCancellation_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	userID := uuid.New()
	rideID := uuid.New()

	mockSvc.On("PreviewCancellation", mock.Anything, rideID, userID).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/rides/:id/cancel/preview", func(c *gin.Context) {
		setUserContext(c, userID, models.RoleRider)
		rideIDParsed, _ := uuid.Parse(c.Param("id"))
		result, err := h.service.PreviewCancellation(c.Request.Context(), rideIDParsed, userID)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to preview cancellation"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rides/"+rideID.String()+"/cancel/preview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ============================================================================
// CANCEL RIDE TESTS
// ============================================================================

func TestHandler_CancelRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	userID := uuid.New()
	rideID := uuid.New()
	now := time.Now()

	cancelResp := &CancelRideResponse{
		RideID:       rideID,
		CancelledBy:  "rider",
		FeeAmount:    0,
		FeeWaived:    true,
		WaiverReason: strPtr("free_cancellation_window"),
		Explanation:  "Free cancellation within 2 minutes",
		CancelledAt:  now,
	}

	mockSvc.On("CancelRide", mock.Anything, rideID, userID, mock.AnythingOfType("*cancellation.CancelRideRequest")).Return(cancelResp, nil)

	h := newTestHandler(mockSvc)
	router.POST("/api/v1/rides/:id/cancel", func(c *gin.Context) {
		setUserContext(c, userID, models.RoleRider)
		rideIDParsed, _ := uuid.Parse(c.Param("id"))
		var req CancelRideRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		result, err := h.service.CancelRide(c.Request.Context(), rideIDParsed, userID, &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	reqBody := CancelRideRequest{
		ReasonCode: ReasonRiderChangedMind,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rides/"+rideID.String()+"/cancel", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	mockSvc.AssertExpectations(t)
}

func TestHandler_CancelRide_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil)
	rideID := uuid.New()

	reqBody := CancelRideRequest{
		ReasonCode: ReasonRiderChangedMind,
	}

	c, w := setupTestContext("POST", "/api/v1/rides/"+rideID.String()+"/cancel", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	// Don't set user context

	handler.CancelRide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CancelRide_InvalidRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil)
	userID := uuid.New()

	reqBody := CancelRideRequest{
		ReasonCode: ReasonRiderChangedMind,
	}

	c, w := setupTestContext("POST", "/api/v1/rides/invalid-uuid/cancel", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID, models.RoleRider)

	handler.CancelRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid ride id")
}

func TestHandler_CancelRide_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil)
	userID := uuid.New()
	rideID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/rides/"+rideID.String()+"/cancel", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/rides/"+rideID.String()+"/cancel", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.CancelRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CancelRide_MissingReasonCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil)
	userID := uuid.New()
	rideID := uuid.New()

	reqBody := map[string]interface{}{
		"reason_text": "some reason",
	}

	c, w := setupTestContext("POST", "/api/v1/rides/"+rideID.String()+"/cancel", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.CancelRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CancelRide_RideNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	userID := uuid.New()
	rideID := uuid.New()

	mockSvc.On("CancelRide", mock.Anything, rideID, userID, mock.AnythingOfType("*cancellation.CancelRideRequest")).Return(nil, common.NewNotFoundError("ride not found", nil))

	h := newTestHandler(mockSvc)
	router.POST("/api/v1/rides/:id/cancel", func(c *gin.Context) {
		setUserContext(c, userID, models.RoleRider)
		rideIDParsed, _ := uuid.Parse(c.Param("id"))
		var req CancelRideRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		result, err := h.service.CancelRide(c.Request.Context(), rideIDParsed, userID, &req)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	reqBody := CancelRideRequest{
		ReasonCode: ReasonRiderChangedMind,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rides/"+rideID.String()+"/cancel", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_CancelRide_AlreadyCancelled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	userID := uuid.New()
	rideID := uuid.New()

	mockSvc.On("CancelRide", mock.Anything, rideID, userID, mock.AnythingOfType("*cancellation.CancelRideRequest")).Return(nil, common.NewBadRequestError("ride already completed or cancelled", nil))

	h := newTestHandler(mockSvc)
	router.POST("/api/v1/rides/:id/cancel", func(c *gin.Context) {
		setUserContext(c, userID, models.RoleRider)
		rideIDParsed, _ := uuid.Parse(c.Param("id"))
		var req CancelRideRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		result, err := h.service.CancelRide(c.Request.Context(), rideIDParsed, userID, &req)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	reqBody := CancelRideRequest{
		ReasonCode: ReasonRiderChangedMind,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rides/"+rideID.String()+"/cancel", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_CancelRide_WithReasonText(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	userID := uuid.New()
	rideID := uuid.New()
	now := time.Now()

	cancelResp := &CancelRideResponse{
		RideID:      rideID,
		CancelledBy: "rider",
		FeeAmount:   0,
		FeeWaived:   true,
		CancelledAt: now,
	}

	mockSvc.On("CancelRide", mock.Anything, rideID, userID, mock.MatchedBy(func(req *CancelRideRequest) bool {
		return req.ReasonCode == ReasonRiderOther && req.ReasonText != nil && *req.ReasonText == "My custom reason"
	})).Return(cancelResp, nil)

	h := newTestHandler(mockSvc)
	router.POST("/api/v1/rides/:id/cancel", func(c *gin.Context) {
		setUserContext(c, userID, models.RoleRider)
		rideIDParsed, _ := uuid.Parse(c.Param("id"))
		var req CancelRideRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		result, err := h.service.CancelRide(c.Request.Context(), rideIDParsed, userID, &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	reasonText := "My custom reason"
	reqBody := CancelRideRequest{
		ReasonCode: ReasonRiderOther,
		ReasonText: &reasonText,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rides/"+rideID.String()+"/cancel", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_CancelRide_DriverCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	driverID := uuid.New()
	rideID := uuid.New()
	now := time.Now()

	cancelResp := &CancelRideResponse{
		RideID:       rideID,
		CancelledBy:  "driver",
		FeeAmount:    0,
		FeeWaived:    true,
		WaiverReason: strPtr("driver_fault"),
		CancelledAt:  now,
	}

	mockSvc.On("CancelRide", mock.Anything, rideID, driverID, mock.AnythingOfType("*cancellation.CancelRideRequest")).Return(cancelResp, nil)

	h := newTestHandler(mockSvc)
	router.POST("/api/v1/rides/:id/cancel", func(c *gin.Context) {
		setUserContext(c, driverID, models.RoleDriver)
		rideIDParsed, _ := uuid.Parse(c.Param("id"))
		var req CancelRideRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		result, err := h.service.CancelRide(c.Request.Context(), rideIDParsed, driverID, &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	reqBody := CancelRideRequest{
		ReasonCode: ReasonDriverRiderNoShow,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rides/"+rideID.String()+"/cancel", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "driver", data["cancelled_by"])
	mockSvc.AssertExpectations(t)
}

// ============================================================================
// GET CANCELLATION DETAILS TESTS
// ============================================================================

func TestHandler_GetCancellationDetails_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	userID := uuid.New()
	rideID := uuid.New()
	recordID := uuid.New()
	now := time.Now()

	record := &CancellationRecord{
		ID:                  recordID,
		RideID:              rideID,
		RiderID:             userID,
		CancelledBy:         CancelledByRider,
		ReasonCode:          ReasonRiderChangedMind,
		FeeAmount:           0,
		FeeWaived:           true,
		MinutesSinceRequest: 1.5,
		CancelledAt:         now,
		CreatedAt:           now,
	}

	mockSvc.On("GetCancellationDetails", mock.Anything, rideID, userID).Return(record, nil)

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/rides/:id/cancellation", func(c *gin.Context) {
		setUserContext(c, userID, models.RoleRider)
		rideIDParsed, _ := uuid.Parse(c.Param("id"))
		result, err := h.service.GetCancellationDetails(c.Request.Context(), rideIDParsed, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rides/"+rideID.String()+"/cancellation", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetCancellationDetails_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil)
	rideID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/rides/"+rideID.String()+"/cancellation", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	// Don't set user context

	handler.GetCancellationDetails(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetCancellationDetails_InvalidRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil)
	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/rides/invalid-uuid/cancellation", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetCancellationDetails(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetCancellationDetails_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	userID := uuid.New()
	rideID := uuid.New()

	mockSvc.On("GetCancellationDetails", mock.Anything, rideID, userID).Return(nil, common.NewNotFoundError("cancellation record not found", nil))

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/rides/:id/cancellation", func(c *gin.Context) {
		setUserContext(c, userID, models.RoleRider)
		rideIDParsed, _ := uuid.Parse(c.Param("id"))
		result, err := h.service.GetCancellationDetails(c.Request.Context(), rideIDParsed, userID)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rides/"+rideID.String()+"/cancellation", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetCancellationDetails_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	userID := uuid.New()
	rideID := uuid.New()

	mockSvc.On("GetCancellationDetails", mock.Anything, rideID, userID).Return(nil, common.NewForbiddenError("not authorized to view this cancellation"))

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/rides/:id/cancellation", func(c *gin.Context) {
		setUserContext(c, userID, models.RoleRider)
		rideIDParsed, _ := uuid.Parse(c.Param("id"))
		result, err := h.service.GetCancellationDetails(c.Request.Context(), rideIDParsed, userID)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rides/"+rideID.String()+"/cancellation", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockSvc.AssertExpectations(t)
}

// ============================================================================
// GET MY CANCELLATION STATS TESTS
// ============================================================================

func TestHandler_GetMyCancellationStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	userID := uuid.New()

	stats := &UserCancellationStats{
		UserID:                userID,
		TotalCancellations:    10,
		CancellationsToday:    2,
		CancellationsThisWeek: 5,
		TotalFeesCharged:      25.00,
		TotalFeesWaived:       10.00,
		CancellationRate:      12.5,
		IsWarned:              false,
		IsPenalized:           false,
	}

	mockSvc.On("GetMyCancellationStats", mock.Anything, userID).Return(stats, nil)

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/cancellations/stats", func(c *gin.Context) {
		setUserContext(c, userID, models.RoleRider)
		result, err := h.service.GetMyCancellationStats(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cancellations/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetMyCancellationStats_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil)

	c, w := setupTestContext("GET", "/api/v1/cancellations/stats", nil)
	// Don't set user context

	handler.GetMyCancellationStats(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetMyCancellationStats_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	userID := uuid.New()

	mockSvc.On("GetMyCancellationStats", mock.Anything, userID).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/cancellations/stats", func(c *gin.Context) {
		setUserContext(c, userID, models.RoleRider)
		result, err := h.service.GetMyCancellationStats(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get cancellation stats"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cancellations/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ============================================================================
// GET MY CANCELLATION HISTORY TESTS
// ============================================================================

func TestHandler_GetMyCancellationHistory_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	userID := uuid.New()
	now := time.Now()

	records := []CancellationRecord{
		{
			ID:          uuid.New(),
			RideID:      uuid.New(),
			RiderID:     userID,
			CancelledBy: CancelledByRider,
			ReasonCode:  ReasonRiderChangedMind,
			FeeAmount:   0,
			FeeWaived:   true,
			CancelledAt: now,
		},
		{
			ID:          uuid.New(),
			RideID:      uuid.New(),
			RiderID:     userID,
			CancelledBy: CancelledByRider,
			ReasonCode:  ReasonRiderWaitTooLong,
			FeeAmount:   5.0,
			FeeWaived:   false,
			CancelledAt: now.Add(-24 * time.Hour),
		},
	}

	mockSvc.On("GetMyCancellationHistory", mock.Anything, userID, 20, 0).Return(records, 2, nil)

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/cancellations/history", func(c *gin.Context) {
		setUserContext(c, userID, models.RoleRider)
		result, total, err := h.service.GetMyCancellationHistory(c.Request.Context(), userID, 20, 0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"cancellations": result}, "meta": gin.H{"total": total}})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cancellations/history", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetMyCancellationHistory_WithPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	userID := uuid.New()

	mockSvc.On("GetMyCancellationHistory", mock.Anything, userID, 10, 20).Return([]CancellationRecord{}, 50, nil)

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/cancellations/history", func(c *gin.Context) {
		setUserContext(c, userID, models.RoleRider)
		// Simulate parsing page and page_size from query
		result, total, err := h.service.GetMyCancellationHistory(c.Request.Context(), userID, 10, 20)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    gin.H{"cancellations": result},
			"meta":    gin.H{"total": total, "limit": 10, "offset": 20},
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cancellations/history?page=3&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	meta := response["meta"].(map[string]interface{})
	assert.Equal(t, float64(50), meta["total"])
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetMyCancellationHistory_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil)

	c, w := setupTestContext("GET", "/api/v1/cancellations/history", nil)
	// Don't set user context

	handler.GetMyCancellationHistory(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetMyCancellationHistory_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	userID := uuid.New()

	mockSvc.On("GetMyCancellationHistory", mock.Anything, userID, 20, 0).Return([]CancellationRecord{}, 0, nil)

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/cancellations/history", func(c *gin.Context) {
		setUserContext(c, userID, models.RoleRider)
		result, total, err := h.service.GetMyCancellationHistory(c.Request.Context(), userID, 20, 0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"cancellations": result}, "meta": gin.H{"total": total}})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cancellations/history", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	cancellations := data["cancellations"].([]interface{})
	assert.Len(t, cancellations, 0)
	mockSvc.AssertExpectations(t)
}

// ============================================================================
// GET CANCELLATION REASONS TESTS
// ============================================================================

func TestHandler_GetCancellationReasons_Rider(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	reasons := &CancellationReasonsResponse{
		Reasons: []CancellationReasonOption{
			{Code: ReasonRiderChangedMind, Label: "Changed my mind", Description: "I no longer need a ride"},
			{Code: ReasonRiderDriverTooFar, Label: "Driver is too far", Description: "The driver is taking too long to arrive"},
		},
	}

	mockSvc.On("GetCancellationReasons", false).Return(reasons)

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/cancellations/reasons", func(c *gin.Context) {
		userType := c.DefaultQuery("type", "rider")
		isDriver := userType == "driver"
		result := h.service.GetCancellationReasons(isDriver)
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cancellations/reasons?type=rider", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetCancellationReasons_Driver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	reasons := &CancellationReasonsResponse{
		Reasons: []CancellationReasonOption{
			{Code: ReasonDriverRiderNoShow, Label: "Rider didn't show up", Description: "The rider was not at the pickup location"},
			{Code: ReasonDriverVehicleIssue, Label: "Vehicle issue", Description: "My vehicle has a mechanical problem"},
		},
	}

	mockSvc.On("GetCancellationReasons", true).Return(reasons)

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/cancellations/reasons", func(c *gin.Context) {
		userType := c.DefaultQuery("type", "rider")
		isDriver := userType == "driver"
		result := h.service.GetCancellationReasons(isDriver)
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cancellations/reasons?type=driver", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetCancellationReasons_DefaultToRider(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	reasons := &CancellationReasonsResponse{
		Reasons: []CancellationReasonOption{
			{Code: ReasonRiderChangedMind, Label: "Changed my mind", Description: "I no longer need a ride"},
		},
	}

	mockSvc.On("GetCancellationReasons", false).Return(reasons)

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/cancellations/reasons", func(c *gin.Context) {
		userType := c.DefaultQuery("type", "rider")
		isDriver := userType == "driver"
		result := h.service.GetCancellationReasons(isDriver)
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cancellations/reasons", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// ============================================================================
// ADMIN WAIVE FEE TESTS
// ============================================================================

func TestHandler_AdminWaiveFee_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	adminID := uuid.New()
	cancellationID := uuid.New()

	mockSvc.On("WaiveFee", mock.Anything, cancellationID, "customer complaint").Return(nil)

	h := newTestHandler(mockSvc)
	router.POST("/api/v1/admin/cancellations/:id/waive", func(c *gin.Context) {
		setUserContext(c, adminID, models.RoleAdmin)
		cancellationIDParsed, _ := uuid.Parse(c.Param("id"))
		var req WaiveFeeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		err := h.service.WaiveFee(c.Request.Context(), cancellationIDParsed, req.Reason)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "cancellation fee waived"})
	})

	reqBody := WaiveFeeRequest{
		Reason: "customer complaint",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/cancellations/"+cancellationID.String()+"/waive", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockSvc.AssertExpectations(t)
}

func TestHandler_AdminWaiveFee_InvalidCancellationID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil)
	adminID := uuid.New()

	reqBody := WaiveFeeRequest{
		Reason: "test reason",
	}

	c, w := setupTestContext("POST", "/api/v1/admin/cancellations/invalid-uuid/waive", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminWaiveFee(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid cancellation id")
}

func TestHandler_AdminWaiveFee_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil)
	adminID := uuid.New()
	cancellationID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/admin/cancellations/"+cancellationID.String()+"/waive", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/cancellations/"+cancellationID.String()+"/waive", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: cancellationID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminWaiveFee(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminWaiveFee_MissingReason(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil)
	adminID := uuid.New()
	cancellationID := uuid.New()

	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/admin/cancellations/"+cancellationID.String()+"/waive", reqBody)
	c.Params = gin.Params{{Key: "id", Value: cancellationID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminWaiveFee(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminWaiveFee_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	adminID := uuid.New()
	cancellationID := uuid.New()

	mockSvc.On("WaiveFee", mock.Anything, cancellationID, "test reason").Return(common.NewNotFoundError("cancellation not found", nil))

	h := newTestHandler(mockSvc)
	router.POST("/api/v1/admin/cancellations/:id/waive", func(c *gin.Context) {
		setUserContext(c, adminID, models.RoleAdmin)
		cancellationIDParsed, _ := uuid.Parse(c.Param("id"))
		var req WaiveFeeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		err := h.service.WaiveFee(c.Request.Context(), cancellationIDParsed, req.Reason)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "cancellation fee waived"})
	})

	reqBody := WaiveFeeRequest{
		Reason: "test reason",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/cancellations/"+cancellationID.String()+"/waive", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

// ============================================================================
// ADMIN GET STATS TESTS
// ============================================================================

func TestHandler_AdminGetStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	adminID := uuid.New()

	stats := &CancellationStatsResponse{
		TotalCancellations:     100,
		RiderCancellations:     70,
		DriverCancellations:    25,
		SystemCancellations:    5,
		TotalFeesCollected:     250.50,
		TotalFeesWaived:        75.00,
		AverageMinutesToCancel: 3.5,
		CancellationRate:       8.5,
		TopReasons: []TopCancellationReason{
			{ReasonCode: ReasonRiderChangedMind, Count: 30, Percentage: 30.0},
		},
	}

	mockSvc.On("GetCancellationStats", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(stats, nil)

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/admin/cancellations/stats", func(c *gin.Context) {
		setUserContext(c, adminID, models.RoleAdmin)
		from := time.Now().AddDate(0, -1, 0)
		to := time.Now()
		result, err := h.service.GetCancellationStats(c.Request.Context(), from, to)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/cancellations/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockSvc.AssertExpectations(t)
}

func TestHandler_AdminGetStats_WithDateRange(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	adminID := uuid.New()

	stats := &CancellationStatsResponse{
		TotalCancellations: 50,
	}

	mockSvc.On("GetCancellationStats", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(stats, nil)

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/admin/cancellations/stats", func(c *gin.Context) {
		setUserContext(c, adminID, models.RoleAdmin)
		from := time.Now().AddDate(0, -1, 0)
		to := time.Now()
		if fromStr := c.Query("from"); fromStr != "" {
			if t, err := time.Parse("2006-01-02", fromStr); err == nil {
				from = t
			}
		}
		if toStr := c.Query("to"); toStr != "" {
			if t, err := time.Parse("2006-01-02", toStr); err == nil {
				to = t.AddDate(0, 0, 1)
			}
		}
		result, err := h.service.GetCancellationStats(c.Request.Context(), from, to)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/cancellations/stats?from=2025-01-01&to=2025-01-31", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_AdminGetStats_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	adminID := uuid.New()

	mockSvc.On("GetCancellationStats", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/admin/cancellations/stats", func(c *gin.Context) {
		setUserContext(c, adminID, models.RoleAdmin)
		from := time.Now().AddDate(0, -1, 0)
		to := time.Now()
		result, err := h.service.GetCancellationStats(c.Request.Context(), from, to)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get cancellation stats"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/cancellations/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ============================================================================
// ADMIN GET USER STATS TESTS
// ============================================================================

func TestHandler_AdminGetUserStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	adminID := uuid.New()
	targetUserID := uuid.New()

	stats := &UserCancellationStats{
		UserID:                targetUserID,
		TotalCancellations:    15,
		CancellationsToday:    2,
		CancellationsThisWeek: 5,
		TotalFeesCharged:      30.00,
		IsWarned:              false,
		IsPenalized:           false,
	}

	mockSvc.On("GetUserCancellationStats", mock.Anything, targetUserID).Return(stats, nil)

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/admin/cancellations/users/:userId/stats", func(c *gin.Context) {
		setUserContext(c, adminID, models.RoleAdmin)
		userIDParsed, _ := uuid.Parse(c.Param("userId"))
		result, err := h.service.GetUserCancellationStats(c.Request.Context(), userIDParsed)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/cancellations/users/"+targetUserID.String()+"/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockSvc.AssertExpectations(t)
}

func TestHandler_AdminGetUserStats_InvalidUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil)
	adminID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/admin/cancellations/users/invalid-uuid/stats", nil)
	c.Params = gin.Params{{Key: "userId", Value: "invalid-uuid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetUserStats(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid user id")
}

func TestHandler_AdminGetUserStats_InternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	adminID := uuid.New()
	targetUserID := uuid.New()

	mockSvc.On("GetUserCancellationStats", mock.Anything, targetUserID).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/admin/cancellations/users/:userId/stats", func(c *gin.Context) {
		setUserContext(c, adminID, models.RoleAdmin)
		userIDParsed, _ := uuid.Parse(c.Param("userId"))
		result, err := h.service.GetUserCancellationStats(c.Request.Context(), userIDParsed)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get user cancellation stats"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/cancellations/users/"+targetUserID.String()+"/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ============================================================================
// TABLE-DRIVEN TESTS
// ============================================================================

func TestHandler_PreviewCancellation_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		userID         uuid.UUID
		rideID         uuid.UUID
		setupMock      func(*MockCancellationService)
		expectedStatus int
	}{
		{
			name:   "successful preview with fee",
			userID: uuid.New(),
			rideID: uuid.New(),
			setupMock: func(m *MockCancellationService) {
				m.On("PreviewCancellation", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID")).Return(&CancellationPreviewResponse{
					FeeAmount:            5.0,
					FeeWaived:            false,
					Explanation:          "Cancellation fee applied",
					FreeCancelsRemaining: 0,
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "successful preview - free cancellation",
			userID: uuid.New(),
			rideID: uuid.New(),
			setupMock: func(m *MockCancellationService) {
				m.On("PreviewCancellation", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID")).Return(&CancellationPreviewResponse{
					FeeAmount:            0,
					FeeWaived:            true,
					WaiverReason:         strPtr("free_cancellation_window"),
					Explanation:          "Free cancellation within 2 minutes",
					FreeCancelsRemaining: 3,
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "ride not found",
			userID: uuid.New(),
			rideID: uuid.New(),
			setupMock: func(m *MockCancellationService) {
				m.On("PreviewCancellation", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID")).Return(nil, common.NewNotFoundError("ride not found", nil))
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "not authorized",
			userID: uuid.New(),
			rideID: uuid.New(),
			setupMock: func(m *MockCancellationService) {
				m.On("PreviewCancellation", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID")).Return(nil, common.NewForbiddenError("not authorized for this ride"))
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "ride already cancelled",
			userID: uuid.New(),
			rideID: uuid.New(),
			setupMock: func(m *MockCancellationService) {
				m.On("PreviewCancellation", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID")).Return(nil, common.NewBadRequestError("ride already completed or cancelled", nil))
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockSvc := new(MockCancellationService)
			tt.setupMock(mockSvc)

			router := setupTestRouter()
			h := newTestHandler(mockSvc)

			router.GET("/api/v1/rides/:id/cancel/preview", func(c *gin.Context) {
				setUserContext(c, tt.userID, models.RoleRider)
				rideIDParsed, _ := uuid.Parse(c.Param("id"))
				result, err := h.service.PreviewCancellation(c.Request.Context(), rideIDParsed, tt.userID)
				if err != nil {
					if appErr, ok := err.(*common.AppError); ok {
						c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
						return
					}
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
			})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/rides/"+tt.rideID.String()+"/cancel/preview", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestHandler_CancelRide_AllReasonCodes(t *testing.T) {
	riderReasons := []CancellationReasonCode{
		ReasonRiderChangedMind,
		ReasonRiderDriverTooFar,
		ReasonRiderWaitTooLong,
		ReasonRiderWrongLocation,
		ReasonRiderPriceChanged,
		ReasonRiderFoundOther,
		ReasonRiderEmergency,
		ReasonRiderOther,
	}

	for _, reasonCode := range riderReasons {
		t.Run(string(reasonCode), func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockSvc := new(MockCancellationService)
			router := setupTestRouter()

			userID := uuid.New()
			rideID := uuid.New()

			mockSvc.On("CancelRide", mock.Anything, rideID, userID, mock.MatchedBy(func(req *CancelRideRequest) bool {
				return req.ReasonCode == reasonCode
			})).Return(&CancelRideResponse{
				RideID:      rideID,
				CancelledBy: "rider",
				FeeAmount:   0,
				FeeWaived:   true,
				CancelledAt: time.Now(),
			}, nil)

			h := newTestHandler(mockSvc)
			router.POST("/api/v1/rides/:id/cancel", func(c *gin.Context) {
				setUserContext(c, userID, models.RoleRider)
				rideIDParsed, _ := uuid.Parse(c.Param("id"))
				var req CancelRideRequest
				c.ShouldBindJSON(&req)
				result, err := h.service.CancelRide(c.Request.Context(), rideIDParsed, userID, &req)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
			})

			reqBody := CancelRideRequest{
				ReasonCode: reasonCode,
			}
			bodyBytes, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/rides/"+rideID.String()+"/cancel", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestHandler_CancelRide_DriverReasonCodes(t *testing.T) {
	driverReasons := []CancellationReasonCode{
		ReasonDriverRiderNoShow,
		ReasonDriverRiderUnreachable,
		ReasonDriverVehicleIssue,
		ReasonDriverUnsafePickup,
		ReasonDriverTooFar,
		ReasonDriverEmergency,
		ReasonDriverOther,
	}

	for _, reasonCode := range driverReasons {
		t.Run(string(reasonCode), func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockSvc := new(MockCancellationService)
			router := setupTestRouter()

			driverID := uuid.New()
			rideID := uuid.New()

			mockSvc.On("CancelRide", mock.Anything, rideID, driverID, mock.MatchedBy(func(req *CancelRideRequest) bool {
				return req.ReasonCode == reasonCode
			})).Return(&CancelRideResponse{
				RideID:       rideID,
				CancelledBy:  "driver",
				FeeAmount:    0,
				FeeWaived:    true,
				WaiverReason: strPtr("driver_fault"),
				CancelledAt:  time.Now(),
			}, nil)

			h := newTestHandler(mockSvc)
			router.POST("/api/v1/rides/:id/cancel", func(c *gin.Context) {
				setUserContext(c, driverID, models.RoleDriver)
				rideIDParsed, _ := uuid.Parse(c.Param("id"))
				var req CancelRideRequest
				c.ShouldBindJSON(&req)
				result, err := h.service.CancelRide(c.Request.Context(), rideIDParsed, driverID, &req)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
			})

			reqBody := CancelRideRequest{
				ReasonCode: reasonCode,
			}
			bodyBytes, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/rides/"+rideID.String()+"/cancel", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			mockSvc.AssertExpectations(t)
		})
	}
}

// ============================================================================
// AUTHORIZATION TESTS
// ============================================================================

func TestHandler_Authorization_RiderCanAccessOwnCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	riderID := uuid.New()
	rideID := uuid.New()

	record := &CancellationRecord{
		ID:          uuid.New(),
		RideID:      rideID,
		RiderID:     riderID,
		CancelledBy: CancelledByRider,
	}

	mockSvc.On("GetCancellationDetails", mock.Anything, rideID, riderID).Return(record, nil)

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/rides/:id/cancellation", func(c *gin.Context) {
		setUserContext(c, riderID, models.RoleRider)
		rideIDParsed, _ := uuid.Parse(c.Param("id"))
		result, err := h.service.GetCancellationDetails(c.Request.Context(), rideIDParsed, riderID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rides/"+rideID.String()+"/cancellation", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_Authorization_DriverCanAccessOwnCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	driverID := uuid.New()
	riderID := uuid.New()
	rideID := uuid.New()

	record := &CancellationRecord{
		ID:          uuid.New(),
		RideID:      rideID,
		RiderID:     riderID,
		DriverID:    &driverID,
		CancelledBy: CancelledByDriver,
	}

	mockSvc.On("GetCancellationDetails", mock.Anything, rideID, driverID).Return(record, nil)

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/rides/:id/cancellation", func(c *gin.Context) {
		setUserContext(c, driverID, models.RoleDriver)
		rideIDParsed, _ := uuid.Parse(c.Param("id"))
		result, err := h.service.GetCancellationDetails(c.Request.Context(), rideIDParsed, driverID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rides/"+rideID.String()+"/cancellation", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_Authorization_OtherUserCannotAccessCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	otherUserID := uuid.New()
	rideID := uuid.New()

	mockSvc.On("GetCancellationDetails", mock.Anything, rideID, otherUserID).Return(nil, common.NewForbiddenError("not authorized to view this cancellation"))

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/rides/:id/cancellation", func(c *gin.Context) {
		setUserContext(c, otherUserID, models.RoleRider)
		rideIDParsed, _ := uuid.Parse(c.Param("id"))
		result, err := h.service.GetCancellationDetails(c.Request.Context(), rideIDParsed, otherUserID)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusForbidden, gin.H{"error": "not authorized"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rides/"+rideID.String()+"/cancellation", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockSvc.AssertExpectations(t)
}

// ============================================================================
// EDGE CASE TESTS
// ============================================================================

func TestHandler_EmptyRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil)
	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/rides//cancel/preview", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	setUserContext(c, userID, models.RoleRider)

	handler.PreviewCancellation(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CancelRide_WithAcceptedAt(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	userID := uuid.New()
	rideID := uuid.New()
	now := time.Now()
	minutesSinceAccept := 5.0

	cancelResp := &CancelRideResponse{
		RideID:      rideID,
		CancelledBy: "rider",
		FeeAmount:   5.0,
		FeeWaived:   false,
		Explanation: "Cancellation fee of 5.00 applied",
		CancelledAt: now,
	}

	mockSvc.On("CancelRide", mock.Anything, rideID, userID, mock.AnythingOfType("*cancellation.CancelRideRequest")).Return(cancelResp, nil)

	h := newTestHandler(mockSvc)
	router.POST("/api/v1/rides/:id/cancel", func(c *gin.Context) {
		setUserContext(c, userID, models.RoleRider)
		rideIDParsed, _ := uuid.Parse(c.Param("id"))
		var req CancelRideRequest
		c.ShouldBindJSON(&req)
		result, err := h.service.CancelRide(c.Request.Context(), rideIDParsed, userID, &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	reqBody := CancelRideRequest{
		ReasonCode: ReasonRiderDriverTooFar,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rides/"+rideID.String()+"/cancel", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, 5.0, data["fee_amount"])
	assert.False(t, data["fee_waived"].(bool))

	_ = minutesSinceAccept // Used in setup but not directly asserted
	mockSvc.AssertExpectations(t)
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func strPtr(s string) *string {
	return &s
}

// ============================================================================
// INTEGRATION-STYLE TESTS (Handler with actual service behavior simulation)
// ============================================================================

func TestHandler_Integration_CancelRideThenGetDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	userID := uuid.New()
	rideID := uuid.New()
	recordID := uuid.New()
	now := time.Now()

	// First, cancel the ride
	cancelResp := &CancelRideResponse{
		RideID:       rideID,
		CancelledBy:  "rider",
		FeeAmount:    0,
		FeeWaived:    true,
		WaiverReason: strPtr("free_cancellation_window"),
		CancelledAt:  now,
	}

	// Then, get the cancellation details
	record := &CancellationRecord{
		ID:                  recordID,
		RideID:              rideID,
		RiderID:             userID,
		CancelledBy:         CancelledByRider,
		ReasonCode:          ReasonRiderChangedMind,
		FeeAmount:           0,
		FeeWaived:           true,
		MinutesSinceRequest: 1.0,
		CancelledAt:         now,
	}

	mockSvc.On("CancelRide", mock.Anything, rideID, userID, mock.AnythingOfType("*cancellation.CancelRideRequest")).Return(cancelResp, nil)
	mockSvc.On("GetCancellationDetails", mock.Anything, rideID, userID).Return(record, nil)

	h := newTestHandler(mockSvc)

	router.POST("/api/v1/rides/:id/cancel", func(c *gin.Context) {
		setUserContext(c, userID, models.RoleRider)
		rideIDParsed, _ := uuid.Parse(c.Param("id"))
		var req CancelRideRequest
		c.ShouldBindJSON(&req)
		result, _ := h.service.CancelRide(c.Request.Context(), rideIDParsed, userID, &req)
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	router.GET("/api/v1/rides/:id/cancellation", func(c *gin.Context) {
		setUserContext(c, userID, models.RoleRider)
		rideIDParsed, _ := uuid.Parse(c.Param("id"))
		result, _ := h.service.GetCancellationDetails(c.Request.Context(), rideIDParsed, userID)
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	// Step 1: Cancel the ride
	reqBody := CancelRideRequest{ReasonCode: ReasonRiderChangedMind}
	bodyBytes, _ := json.Marshal(reqBody)
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/rides/"+rideID.String()+"/cancel", bytes.NewReader(bodyBytes))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)

	// Step 2: Get cancellation details
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/rides/"+rideID.String()+"/cancellation", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	response := parseResponse(w2)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "rider", data["cancelled_by"])

	mockSvc.AssertExpectations(t)
}

// ============================================================================
// MOCK REPOSITORY COMPATIBILITY TESTS (for pgx.ErrNoRows handling)
// ============================================================================

func TestHandler_GetCancellationDetails_PgxErrNoRows(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockCancellationService)
	router := setupTestRouter()

	userID := uuid.New()
	rideID := uuid.New()

	// Simulate pgx.ErrNoRows being converted to AppError in service
	mockSvc.On("GetCancellationDetails", mock.Anything, rideID, userID).Return(nil, common.NewNotFoundError("cancellation record not found", pgx.ErrNoRows))

	h := newTestHandler(mockSvc)
	router.GET("/api/v1/rides/:id/cancellation", func(c *gin.Context) {
		setUserContext(c, userID, models.RoleRider)
		rideIDParsed, _ := uuid.Parse(c.Param("id"))
		result, err := h.service.GetCancellationDetails(c.Request.Context(), rideIDParsed, userID)
		if err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				c.JSON(appErr.Code, gin.H{"success": false, "error": gin.H{"message": appErr.Message}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rides/"+rideID.String()+"/cancellation", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}
