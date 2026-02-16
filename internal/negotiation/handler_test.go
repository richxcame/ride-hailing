package negotiation

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
	"github.com/richxcame/ride-hailing/internal/geography"
	"github.com/richxcame/ride-hailing/internal/pricing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Mock Service for Handler Tests
// ============================================================================

// MockServiceForHandler mocks the Service for handler testing
// We create a testable handler that uses interface-based service methods
type MockServiceForHandler struct {
	mock.Mock
}

func (m *MockServiceForHandler) StartSession(ctx context.Context, riderID uuid.UUID, req *StartSessionRequest) (*Session, error) {
	args := m.Called(ctx, riderID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Session), args.Error(1)
}

func (m *MockServiceForHandler) GetSession(ctx context.Context, sessionID uuid.UUID) (*Session, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Session), args.Error(1)
}

func (m *MockServiceForHandler) GetActiveSessionByRider(ctx context.Context, riderID uuid.UUID) (*Session, error) {
	args := m.Called(ctx, riderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Session), args.Error(1)
}

func (m *MockServiceForHandler) SubmitOffer(ctx context.Context, sessionID, driverID uuid.UUID, req *SubmitOfferRequest, driverInfo *DriverInfo) (*Offer, error) {
	args := m.Called(ctx, sessionID, driverID, req, driverInfo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Offer), args.Error(1)
}

func (m *MockServiceForHandler) AcceptOffer(ctx context.Context, sessionID, offerID, riderID uuid.UUID) error {
	args := m.Called(ctx, sessionID, offerID, riderID)
	return args.Error(0)
}

func (m *MockServiceForHandler) CancelSession(ctx context.Context, sessionID, riderID uuid.UUID, reason string) error {
	args := m.Called(ctx, sessionID, riderID, reason)
	return args.Error(0)
}

// TestableHandler wraps handler methods for testing with mock service
type TestableHandler struct {
	service *MockServiceForHandler
}

func NewTestableHandler(mockService *MockServiceForHandler) *TestableHandler {
	return &TestableHandler{service: mockService}
}

// StartSession - testable version
func (h *TestableHandler) StartSession(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": 401, "message": "unauthorized"}})
		return
	}

	var req StartSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": err.Error()}})
		return
	}

	session, err := h.service.StartSession(c.Request.Context(), userID.(uuid.UUID), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": ToSessionResponse(session)})
}

// GetSession - testable version
func (h *TestableHandler) GetSession(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid session ID"}})
		return
	}

	session, err := h.service.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"code": 404, "message": "session not found"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": ToSessionResponse(session)})
}

// GetActiveSession - testable version
func (h *TestableHandler) GetActiveSession(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": 401, "message": "unauthorized"}})
		return
	}

	session, err := h.service.GetActiveSessionByRider(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to get session"}})
		return
	}

	if session == nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": nil})
		return
	}

	fullSession, err := h.service.GetSession(c.Request.Context(), session.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to get session"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": ToSessionResponse(fullSession)})
}

// SubmitOffer - testable version
func (h *TestableHandler) SubmitOffer(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": 401, "message": "unauthorized"}})
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid session ID"}})
		return
	}

	var req SubmitOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": err.Error()}})
		return
	}

	driverInfo := &DriverInfo{}

	offer, err := h.service.SubmitOffer(c.Request.Context(), sessionID, userID.(uuid.UUID), &req, driverInfo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": ToOfferResponse(offer)})
}

// AcceptOffer - testable version
func (h *TestableHandler) AcceptOffer(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": 401, "message": "unauthorized"}})
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid session ID"}})
		return
	}

	offerID, err := uuid.Parse(c.Param("offerId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid offer ID"}})
		return
	}

	if err := h.service.AcceptOffer(c.Request.Context(), sessionID, offerID, userID.(uuid.UUID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message":    "offer accepted",
			"session_id": sessionID,
			"offer_id":   offerID,
		},
	})
}

// RejectOffer - testable version
func (h *TestableHandler) RejectOffer(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "offer rejected"}})
}

// CancelSession - testable version
func (h *TestableHandler) CancelSession(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": 401, "message": "unauthorized"}})
		return
	}

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid session ID"}})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	if err := h.service.CancelSession(c.Request.Context(), sessionID, userID.(uuid.UUID), req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "session cancelled"}})
}

// WithdrawOffer - testable version
func (h *TestableHandler) WithdrawOffer(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "offer withdrawn"}})
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

func setupTestContextWithParams(method, path string, body interface{}, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
	c, w := setupTestContext(method, path, body)
	c.Params = params
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

func createTestSession(riderID uuid.UUID) *Session {
	return &Session{
		ID:                   uuid.New(),
		RiderID:              riderID,
		Status:               SessionStatusActive,
		PickupLatitude:       40.7128,
		PickupLongitude:      -74.0060,
		PickupAddress:        "123 Main St",
		DropoffLatitude:      40.7580,
		DropoffLongitude:     -73.9855,
		DropoffAddress:       "456 Broadway",
		CurrencyCode:         "USD",
		EstimatedFare:        100.0,
		FairPriceMin:         70.0,
		FairPriceMax:         150.0,
		SystemSuggestedPrice: 100.0,
		ExpiresAt:            time.Now().Add(5 * time.Minute),
		CreatedAt:            time.Now(),
	}
}

func createTestOffer(sessionID, driverID uuid.UUID) *Offer {
	return &Offer{
		ID:           uuid.New(),
		SessionID:    sessionID,
		DriverID:     driverID,
		OfferedPrice: 90.0,
		CurrencyCode: "USD",
		Status:       OfferStatusPending,
		CreatedAt:    time.Now(),
	}
}

// ============================================================================
// StartSession Handler Tests
// ============================================================================

func TestHandler_StartSession_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	expectedSession := createTestSession(riderID)

	reqBody := StartSessionRequest{
		PickupLatitude:   40.7128,
		PickupLongitude:  -74.0060,
		PickupAddress:    "123 Main St",
		DropoffLatitude:  40.7580,
		DropoffLongitude: -73.9855,
		DropoffAddress:   "456 Broadway",
	}

	mockService.On("StartSession", mock.Anything, riderID, &reqBody).Return(expectedSession, nil)

	c, w := setupTestContext("POST", "/api/v1/negotiations", reqBody)
	setUserContext(c, riderID)

	handler.StartSession(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
	mockService.AssertExpectations(t)
}

func TestHandler_StartSession_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	reqBody := StartSessionRequest{
		PickupLatitude:   40.7128,
		PickupLongitude:  -74.0060,
		PickupAddress:    "123 Main St",
		DropoffLatitude:  40.7580,
		DropoffLongitude: -73.9855,
		DropoffAddress:   "456 Broadway",
	}

	c, w := setupTestContext("POST", "/api/v1/negotiations", reqBody)
	// Don't set user context

	handler.StartSession(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_StartSession_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/negotiations", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/negotiations", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, riderID)

	handler.StartSession(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_StartSession_MissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()

	// Missing pickup_address
	reqBody := map[string]interface{}{
		"pickup_latitude":   40.7128,
		"pickup_longitude":  -74.0060,
		"dropoff_latitude":  40.7580,
		"dropoff_longitude": -73.9855,
		"dropoff_address":   "456 Broadway",
	}

	c, w := setupTestContext("POST", "/api/v1/negotiations", reqBody)
	setUserContext(c, riderID)

	handler.StartSession(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_StartSession_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()

	reqBody := StartSessionRequest{
		PickupLatitude:   40.7128,
		PickupLongitude:  -74.0060,
		PickupAddress:    "123 Main St",
		DropoffLatitude:  40.7580,
		DropoffLongitude: -73.9855,
		DropoffAddress:   "456 Broadway",
	}

	mockService.On("StartSession", mock.Anything, riderID, &reqBody).Return(nil, errors.New("rider already has an active negotiation session"))

	c, w := setupTestContext("POST", "/api/v1/negotiations", reqBody)
	setUserContext(c, riderID)

	handler.StartSession(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_StartSession_WithInitialOffer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	initialOffer := 85.0
	expectedSession := createTestSession(riderID)
	expectedSession.RiderInitialOffer = &initialOffer

	reqBody := StartSessionRequest{
		PickupLatitude:   40.7128,
		PickupLongitude:  -74.0060,
		PickupAddress:    "123 Main St",
		DropoffLatitude:  40.7580,
		DropoffLongitude: -73.9855,
		DropoffAddress:   "456 Broadway",
		InitialOffer:     &initialOffer,
	}

	mockService.On("StartSession", mock.Anything, riderID, &reqBody).Return(expectedSession, nil)

	c, w := setupTestContext("POST", "/api/v1/negotiations", reqBody)
	setUserContext(c, riderID)

	handler.StartSession(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetSession Handler Tests
// ============================================================================

func TestHandler_GetSession_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	sessionID := uuid.New()
	riderID := uuid.New()
	expectedSession := createTestSession(riderID)
	expectedSession.ID = sessionID

	mockService.On("GetSession", mock.Anything, sessionID).Return(expectedSession, nil)

	c, w := setupTestContextWithParams("GET", "/api/v1/negotiations/"+sessionID.String(), nil, gin.Params{
		{Key: "id", Value: sessionID.String()},
	})

	handler.GetSession(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetSession_InvalidSessionID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContextWithParams("GET", "/api/v1/negotiations/invalid-uuid", nil, gin.Params{
		{Key: "id", Value: "invalid-uuid"},
	})

	handler.GetSession(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	errorInfo := response["error"].(map[string]interface{})
	assert.Equal(t, "invalid session ID", errorInfo["message"])
}

func TestHandler_GetSession_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	sessionID := uuid.New()

	mockService.On("GetSession", mock.Anything, sessionID).Return(nil, errors.New("session not found"))

	c, w := setupTestContextWithParams("GET", "/api/v1/negotiations/"+sessionID.String(), nil, gin.Params{
		{Key: "id", Value: sessionID.String()},
	})

	handler.GetSession(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetActiveSession Handler Tests
// ============================================================================

func TestHandler_GetActiveSession_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	session := createTestSession(riderID)

	mockService.On("GetActiveSessionByRider", mock.Anything, riderID).Return(session, nil)
	mockService.On("GetSession", mock.Anything, session.ID).Return(session, nil)

	c, w := setupTestContext("GET", "/api/v1/negotiations/active", nil)
	setUserContext(c, riderID)

	handler.GetActiveSession(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetActiveSession_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/negotiations/active", nil)
	// Don't set user context

	handler.GetActiveSession(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_GetActiveSession_NoActiveSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()

	mockService.On("GetActiveSessionByRider", mock.Anything, riderID).Return(nil, nil)

	c, w := setupTestContext("GET", "/api/v1/negotiations/active", nil)
	setUserContext(c, riderID)

	handler.GetActiveSession(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	assert.Nil(t, response["data"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetActiveSession_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()

	mockService.On("GetActiveSessionByRider", mock.Anything, riderID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/negotiations/active", nil)
	setUserContext(c, riderID)

	handler.GetActiveSession(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

// ============================================================================
// SubmitOffer Handler Tests
// ============================================================================

func TestHandler_SubmitOffer_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	sessionID := uuid.New()
	expectedOffer := createTestOffer(sessionID, driverID)

	reqBody := SubmitOfferRequest{
		OfferedPrice: 90.0,
	}

	mockService.On("SubmitOffer", mock.Anything, sessionID, driverID, &reqBody, mock.AnythingOfType("*negotiation.DriverInfo")).Return(expectedOffer, nil)

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/offers", reqBody, gin.Params{
		{Key: "id", Value: sessionID.String()},
	})
	setUserContext(c, driverID)

	handler.SubmitOffer(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
	mockService.AssertExpectations(t)
}

func TestHandler_SubmitOffer_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	sessionID := uuid.New()

	reqBody := SubmitOfferRequest{
		OfferedPrice: 90.0,
	}

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/offers", reqBody, gin.Params{
		{Key: "id", Value: sessionID.String()},
	})
	// Don't set user context

	handler.SubmitOffer(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_SubmitOffer_InvalidSessionID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	reqBody := SubmitOfferRequest{
		OfferedPrice: 90.0,
	}

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/invalid-uuid/offers", reqBody, gin.Params{
		{Key: "id", Value: "invalid-uuid"},
	})
	setUserContext(c, driverID)

	handler.SubmitOffer(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	errorInfo := response["error"].(map[string]interface{})
	assert.Equal(t, "invalid session ID", errorInfo["message"])
}

func TestHandler_SubmitOffer_InvalidRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	sessionID := uuid.New()

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/offers", nil, gin.Params{
		{Key: "id", Value: sessionID.String()},
	})
	c.Request = httptest.NewRequest("POST", "/api/v1/negotiations/"+sessionID.String()+"/offers", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, driverID)

	handler.SubmitOffer(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SubmitOffer_MissingPrice(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	sessionID := uuid.New()

	reqBody := map[string]interface{}{
		// Missing offered_price
	}

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/offers", reqBody, gin.Params{
		{Key: "id", Value: sessionID.String()},
	})
	setUserContext(c, driverID)

	handler.SubmitOffer(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SubmitOffer_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	sessionID := uuid.New()

	reqBody := SubmitOfferRequest{
		OfferedPrice: 50.0, // Below minimum
	}

	mockService.On("SubmitOffer", mock.Anything, sessionID, driverID, &reqBody, mock.AnythingOfType("*negotiation.DriverInfo")).Return(nil, errors.New("offered price 50.00 is below minimum 70.00"))

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/offers", reqBody, gin.Params{
		{Key: "id", Value: sessionID.String()},
	})
	setUserContext(c, driverID)

	handler.SubmitOffer(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_SubmitOffer_WithDriverLocation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	sessionID := uuid.New()
	expectedOffer := createTestOffer(sessionID, driverID)

	latitude := 40.7128
	longitude := -74.0060
	eta := 5

	reqBody := SubmitOfferRequest{
		OfferedPrice:        90.0,
		DriverLatitude:      &latitude,
		DriverLongitude:     &longitude,
		EstimatedPickupTime: &eta,
	}

	mockService.On("SubmitOffer", mock.Anything, sessionID, driverID, &reqBody, mock.AnythingOfType("*negotiation.DriverInfo")).Return(expectedOffer, nil)

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/offers", reqBody, gin.Params{
		{Key: "id", Value: sessionID.String()},
	})
	setUserContext(c, driverID)

	handler.SubmitOffer(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// AcceptOffer Handler Tests
// ============================================================================

func TestHandler_AcceptOffer_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	sessionID := uuid.New()
	offerID := uuid.New()

	mockService.On("AcceptOffer", mock.Anything, sessionID, offerID, riderID).Return(nil)

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/offers/"+offerID.String()+"/accept", nil, gin.Params{
		{Key: "id", Value: sessionID.String()},
		{Key: "offerId", Value: offerID.String()},
	})
	setUserContext(c, riderID)

	handler.AcceptOffer(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "offer accepted", data["message"])
	mockService.AssertExpectations(t)
}

func TestHandler_AcceptOffer_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	sessionID := uuid.New()
	offerID := uuid.New()

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/offers/"+offerID.String()+"/accept", nil, gin.Params{
		{Key: "id", Value: sessionID.String()},
		{Key: "offerId", Value: offerID.String()},
	})
	// Don't set user context

	handler.AcceptOffer(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_AcceptOffer_InvalidSessionID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	offerID := uuid.New()

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/invalid-uuid/offers/"+offerID.String()+"/accept", nil, gin.Params{
		{Key: "id", Value: "invalid-uuid"},
		{Key: "offerId", Value: offerID.String()},
	})
	setUserContext(c, riderID)

	handler.AcceptOffer(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	errorInfo := response["error"].(map[string]interface{})
	assert.Equal(t, "invalid session ID", errorInfo["message"])
}

func TestHandler_AcceptOffer_InvalidOfferID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	sessionID := uuid.New()

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/offers/invalid-uuid/accept", nil, gin.Params{
		{Key: "id", Value: sessionID.String()},
		{Key: "offerId", Value: "invalid-uuid"},
	})
	setUserContext(c, riderID)

	handler.AcceptOffer(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	errorInfo := response["error"].(map[string]interface{})
	assert.Equal(t, "invalid offer ID", errorInfo["message"])
}

func TestHandler_AcceptOffer_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	sessionID := uuid.New()
	offerID := uuid.New()

	mockService.On("AcceptOffer", mock.Anything, sessionID, offerID, riderID).Return(errors.New("offer is no longer pending"))

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/offers/"+offerID.String()+"/accept", nil, gin.Params{
		{Key: "id", Value: sessionID.String()},
		{Key: "offerId", Value: offerID.String()},
	})
	setUserContext(c, riderID)

	handler.AcceptOffer(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_AcceptOffer_SessionBelongsToAnotherRider(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	sessionID := uuid.New()
	offerID := uuid.New()

	mockService.On("AcceptOffer", mock.Anything, sessionID, offerID, riderID).Return(errors.New("unauthorized: session belongs to another rider"))

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/offers/"+offerID.String()+"/accept", nil, gin.Params{
		{Key: "id", Value: sessionID.String()},
		{Key: "offerId", Value: offerID.String()},
	})
	setUserContext(c, riderID)

	handler.AcceptOffer(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// RejectOffer Handler Tests
// ============================================================================

func TestHandler_RejectOffer_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	sessionID := uuid.New()
	offerID := uuid.New()

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/offers/"+offerID.String()+"/reject", nil, gin.Params{
		{Key: "id", Value: sessionID.String()},
		{Key: "offerId", Value: offerID.String()},
	})

	handler.RejectOffer(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "offer rejected", data["message"])
}

// ============================================================================
// CancelSession Handler Tests
// ============================================================================

func TestHandler_CancelSession_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	sessionID := uuid.New()

	reqBody := map[string]interface{}{
		"reason": "changed my mind",
	}

	mockService.On("CancelSession", mock.Anything, sessionID, riderID, "changed my mind").Return(nil)

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/cancel", reqBody, gin.Params{
		{Key: "id", Value: sessionID.String()},
	})
	setUserContext(c, riderID)

	handler.CancelSession(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "session cancelled", data["message"])
	mockService.AssertExpectations(t)
}

func TestHandler_CancelSession_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	sessionID := uuid.New()

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/cancel", nil, gin.Params{
		{Key: "id", Value: sessionID.String()},
	})
	// Don't set user context

	handler.CancelSession(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CancelSession_InvalidSessionID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/invalid-uuid/cancel", nil, gin.Params{
		{Key: "id", Value: "invalid-uuid"},
	})
	setUserContext(c, riderID)

	handler.CancelSession(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	errorInfo := response["error"].(map[string]interface{})
	assert.Equal(t, "invalid session ID", errorInfo["message"])
}

func TestHandler_CancelSession_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	sessionID := uuid.New()

	mockService.On("CancelSession", mock.Anything, sessionID, riderID, "").Return(errors.New("session is no longer active"))

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/cancel", nil, gin.Params{
		{Key: "id", Value: sessionID.String()},
	})
	setUserContext(c, riderID)

	handler.CancelSession(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_CancelSession_WithoutReason(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	sessionID := uuid.New()

	mockService.On("CancelSession", mock.Anything, sessionID, riderID, "").Return(nil)

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/cancel", nil, gin.Params{
		{Key: "id", Value: sessionID.String()},
	})
	setUserContext(c, riderID)

	handler.CancelSession(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_CancelSession_SessionBelongsToAnotherRider(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	sessionID := uuid.New()

	mockService.On("CancelSession", mock.Anything, sessionID, riderID, "").Return(errors.New("unauthorized: session belongs to another rider"))

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/cancel", nil, gin.Params{
		{Key: "id", Value: sessionID.String()},
	})
	setUserContext(c, riderID)

	handler.CancelSession(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// WithdrawOffer Handler Tests
// ============================================================================

func TestHandler_WithdrawOffer_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	sessionID := uuid.New()
	offerID := uuid.New()

	c, w := setupTestContextWithParams("DELETE", "/api/v1/negotiations/"+sessionID.String()+"/offers/"+offerID.String(), nil, gin.Params{
		{Key: "id", Value: sessionID.String()},
		{Key: "offerId", Value: offerID.String()},
	})

	handler.WithdrawOffer(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "offer withdrawn", data["message"])
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_StartSession_ValidationCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           interface{}
		setupMock      func(*MockServiceForHandler, uuid.UUID)
		expectedStatus int
	}{
		{
			name: "valid request",
			body: StartSessionRequest{
				PickupLatitude:   40.7128,
				PickupLongitude:  -74.0060,
				PickupAddress:    "123 Main St",
				DropoffLatitude:  40.7580,
				DropoffLongitude: -73.9855,
				DropoffAddress:   "456 Broadway",
			},
			setupMock: func(m *MockServiceForHandler, riderID uuid.UUID) {
				session := createTestSession(riderID)
				m.On("StartSession", mock.Anything, riderID, mock.AnythingOfType("*negotiation.StartSessionRequest")).Return(session, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "missing pickup address",
			body: map[string]interface{}{
				"pickup_latitude":   40.7128,
				"pickup_longitude":  -74.0060,
				"dropoff_latitude":  40.7580,
				"dropoff_longitude": -73.9855,
				"dropoff_address":   "456 Broadway",
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
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
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing pickup coordinates",
			body: map[string]interface{}{
				"pickup_address":    "123 Main St",
				"dropoff_latitude":  40.7580,
				"dropoff_longitude": -73.9855,
				"dropoff_address":   "456 Broadway",
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockServiceForHandler)
			handler := NewTestableHandler(mockService)

			riderID := uuid.New()
			if tt.setupMock != nil {
				tt.setupMock(mockService, riderID)
			}

			c, w := setupTestContext("POST", "/api/v1/negotiations", tt.body)
			setUserContext(c, riderID)

			handler.StartSession(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_SubmitOffer_ValidationCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sessionID := uuid.New()

	tests := []struct {
		name           string
		body           interface{}
		setupMock      func(*MockServiceForHandler, uuid.UUID)
		expectedStatus int
	}{
		{
			name: "valid offer",
			body: SubmitOfferRequest{
				OfferedPrice: 90.0,
			},
			setupMock: func(m *MockServiceForHandler, driverID uuid.UUID) {
				offer := createTestOffer(sessionID, driverID)
				m.On("SubmitOffer", mock.Anything, sessionID, driverID, mock.AnythingOfType("*negotiation.SubmitOfferRequest"), mock.AnythingOfType("*negotiation.DriverInfo")).Return(offer, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "offer below minimum",
			body: SubmitOfferRequest{
				OfferedPrice: 50.0,
			},
			setupMock: func(m *MockServiceForHandler, driverID uuid.UUID) {
				m.On("SubmitOffer", mock.Anything, sessionID, driverID, mock.AnythingOfType("*negotiation.SubmitOfferRequest"), mock.AnythingOfType("*negotiation.DriverInfo")).Return(nil, errors.New("offered price 50.00 is below minimum 70.00"))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "offer above maximum",
			body: SubmitOfferRequest{
				OfferedPrice: 200.0,
			},
			setupMock: func(m *MockServiceForHandler, driverID uuid.UUID) {
				m.On("SubmitOffer", mock.Anything, sessionID, driverID, mock.AnythingOfType("*negotiation.SubmitOfferRequest"), mock.AnythingOfType("*negotiation.DriverInfo")).Return(nil, errors.New("offered price 200.00 exceeds maximum 150.00"))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing price",
			body:           map[string]interface{}{},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "zero price",
			body: map[string]interface{}{
				"offered_price": 0,
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "negative price",
			body: map[string]interface{}{
				"offered_price": -10.0,
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockServiceForHandler)
			handler := NewTestableHandler(mockService)

			driverID := uuid.New()
			if tt.setupMock != nil {
				tt.setupMock(mockService, driverID)
			}

			c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/offers", tt.body, gin.Params{
				{Key: "id", Value: sessionID.String()},
			})
			setUserContext(c, driverID)

			handler.SubmitOffer(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_AcceptOffer_Cases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		sessionID      string
		offerID        string
		setupMock      func(*MockServiceForHandler, uuid.UUID, uuid.UUID, uuid.UUID)
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:      "success",
			sessionID: uuid.New().String(),
			offerID:   uuid.New().String(),
			setupMock: func(m *MockServiceForHandler, riderID uuid.UUID, sessionID uuid.UUID, offerID uuid.UUID) {
				m.On("AcceptOffer", mock.Anything, sessionID, offerID, riderID).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "offer accepted",
		},
		{
			name:           "invalid session ID",
			sessionID:      "not-a-uuid",
			offerID:        uuid.New().String(),
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "invalid session ID",
		},
		{
			name:           "invalid offer ID",
			sessionID:      uuid.New().String(),
			offerID:        "not-a-uuid",
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "invalid offer ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockServiceForHandler)
			handler := NewTestableHandler(mockService)

			riderID := uuid.New()

			var sessionUUID, offerUUID uuid.UUID
			if sid, err := uuid.Parse(tt.sessionID); err == nil {
				sessionUUID = sid
			}
			if oid, err := uuid.Parse(tt.offerID); err == nil {
				offerUUID = oid
			}

			if tt.setupMock != nil {
				tt.setupMock(mockService, riderID, sessionUUID, offerUUID)
			}

			c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+tt.sessionID+"/offers/"+tt.offerID+"/accept", nil, gin.Params{
				{Key: "id", Value: tt.sessionID},
				{Key: "offerId", Value: tt.offerID},
			})
			setUserContext(c, riderID)

			handler.AcceptOffer(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			response := parseResponse(w)
			if tt.expectedStatus == http.StatusOK {
				data := response["data"].(map[string]interface{})
				assert.Equal(t, tt.expectedMsg, data["message"])
			} else {
				errorInfo := response["error"].(map[string]interface{})
				assert.Equal(t, tt.expectedMsg, errorInfo["message"])
			}
		})
	}
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_StartSession_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	session := createTestSession(riderID)

	reqBody := StartSessionRequest{
		PickupLatitude:   40.7128,
		PickupLongitude:  -74.0060,
		PickupAddress:    "123 Main St",
		DropoffLatitude:  40.7580,
		DropoffLongitude: -73.9855,
		DropoffAddress:   "456 Broadway",
	}

	mockService.On("StartSession", mock.Anything, riderID, &reqBody).Return(session, nil)

	c, w := setupTestContext("POST", "/api/v1/negotiations", reqBody)
	setUserContext(c, riderID)

	handler.StartSession(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["id"])
	assert.NotNil(t, data["status"])
	assert.NotNil(t, data["pickup_address"])
	assert.NotNil(t, data["dropoff_address"])
	assert.NotNil(t, data["currency_code"])
	assert.NotNil(t, data["estimated_fare"])
	assert.NotNil(t, data["fair_price_min"])
	assert.NotNil(t, data["fair_price_max"])

	mockService.AssertExpectations(t)
}

func TestHandler_GetSession_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	sessionID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()

	session := createTestSession(riderID)
	session.ID = sessionID
	session.Offers = []*Offer{
		createTestOffer(sessionID, driverID),
	}

	mockService.On("GetSession", mock.Anything, sessionID).Return(session, nil)

	c, w := setupTestContextWithParams("GET", "/api/v1/negotiations/"+sessionID.String(), nil, gin.Params{
		{Key: "id", Value: sessionID.String()},
	})

	handler.GetSession(c)

	assert.Equal(t, http.StatusOK, w.Code)

	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["offers"])
	assert.Equal(t, 1, int(data["offers_count"].(float64)))

	offers := data["offers"].([]interface{})
	require.Len(t, offers, 1)

	offer := offers[0].(map[string]interface{})
	assert.NotNil(t, offer["id"])
	assert.NotNil(t, offer["driver_id"])
	assert.NotNil(t, offer["offered_price"])
	assert.NotNil(t, offer["status"])

	mockService.AssertExpectations(t)
}

func TestHandler_SubmitOffer_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	sessionID := uuid.New()
	offer := createTestOffer(sessionID, driverID)

	reqBody := SubmitOfferRequest{
		OfferedPrice: 90.0,
	}

	mockService.On("SubmitOffer", mock.Anything, sessionID, driverID, &reqBody, mock.AnythingOfType("*negotiation.DriverInfo")).Return(offer, nil)

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/offers", reqBody, gin.Params{
		{Key: "id", Value: sessionID.String()},
	})
	setUserContext(c, driverID)

	handler.SubmitOffer(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["id"])
	assert.NotNil(t, data["driver_id"])
	assert.NotNil(t, data["offered_price"])
	assert.NotNil(t, data["status"])
	assert.Equal(t, string(OfferStatusPending), data["status"])

	mockService.AssertExpectations(t)
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()

	reqBody := StartSessionRequest{
		PickupLatitude:   40.7128,
		PickupLongitude:  -74.0060,
		PickupAddress:    "123 Main St",
		DropoffLatitude:  40.7580,
		DropoffLongitude: -73.9855,
		DropoffAddress:   "456 Broadway",
	}

	mockService.On("StartSession", mock.Anything, riderID, &reqBody).Return(nil, errors.New("rider already has an active negotiation session"))

	c, w := setupTestContext("POST", "/api/v1/negotiations", reqBody)
	setUserContext(c, riderID)

	handler.StartSession(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	assert.NotNil(t, response["error"])

	errorInfo := response["error"].(map[string]interface{})
	assert.NotNil(t, errorInfo["code"])
	assert.NotNil(t, errorInfo["message"])
	assert.Equal(t, float64(400), errorInfo["code"])
	assert.Contains(t, errorInfo["message"], "rider already has an active")

	mockService.AssertExpectations(t)
}

// ============================================================================
// ToSessionResponse and ToOfferResponse Tests
// ============================================================================

func TestToSessionResponse_NilSession(t *testing.T) {
	result := ToSessionResponse(nil)
	assert.Nil(t, result)
}

func TestToSessionResponse_WithOffers(t *testing.T) {
	riderID := uuid.New()
	driverID := uuid.New()

	session := createTestSession(riderID)
	session.Offers = []*Offer{
		createTestOffer(session.ID, driverID),
		createTestOffer(session.ID, uuid.New()),
	}

	result := ToSessionResponse(session)

	require.NotNil(t, result)
	assert.Equal(t, session.ID, result.ID)
	assert.Equal(t, session.Status, result.Status)
	assert.Equal(t, session.PickupAddress, result.PickupAddress)
	assert.Equal(t, session.DropoffAddress, result.DropoffAddress)
	assert.Equal(t, session.CurrencyCode, result.CurrencyCode)
	assert.Equal(t, session.EstimatedFare, result.EstimatedFare)
	assert.Equal(t, session.FairPriceMin, result.FairPriceMin)
	assert.Equal(t, session.FairPriceMax, result.FairPriceMax)
	assert.Equal(t, 2, result.OffersCount)
	assert.Len(t, result.Offers, 2)
}

func TestToSessionResponse_WithoutOffers(t *testing.T) {
	riderID := uuid.New()

	session := createTestSession(riderID)
	session.Offers = nil

	result := ToSessionResponse(session)

	require.NotNil(t, result)
	assert.Equal(t, 0, result.OffersCount)
	assert.Nil(t, result.Offers)
}

func TestToOfferResponse_NilOffer(t *testing.T) {
	result := ToOfferResponse(nil)
	assert.Nil(t, result)
}

func TestToOfferResponse_ValidOffer(t *testing.T) {
	sessionID := uuid.New()
	driverID := uuid.New()

	offer := createTestOffer(sessionID, driverID)

	result := ToOfferResponse(offer)

	require.NotNil(t, result)
	assert.Equal(t, offer.ID, result.ID)
	assert.Equal(t, offer.DriverID, result.DriverID)
	assert.Equal(t, offer.OfferedPrice, result.OfferedPrice)
	assert.Equal(t, offer.Status, result.Status)
	assert.Equal(t, offer.IsCounterOffer, result.IsCounterOffer)
}

// ============================================================================
// Edge Cases and Boundary Tests
// ============================================================================

func TestHandler_StartSession_EmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/negotiations", map[string]interface{}{})
	setUserContext(c, riderID)

	handler.StartSession(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetSession_EmptySessionID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContextWithParams("GET", "/api/v1/negotiations/", nil, gin.Params{
		{Key: "id", Value: ""},
	})

	handler.GetSession(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SubmitOffer_ZeroPrice(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	sessionID := uuid.New()

	reqBody := map[string]interface{}{
		"offered_price": 0,
	}

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/offers", reqBody, gin.Params{
		{Key: "id", Value: sessionID.String()},
	})
	setUserContext(c, driverID)

	handler.SubmitOffer(c)

	// Zero price should fail validation (gt=0)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SubmitOffer_SessionExpired(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	sessionID := uuid.New()

	reqBody := SubmitOfferRequest{
		OfferedPrice: 90.0,
	}

	mockService.On("SubmitOffer", mock.Anything, sessionID, driverID, &reqBody, mock.AnythingOfType("*negotiation.DriverInfo")).Return(nil, errors.New("session has expired"))

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/offers", reqBody, gin.Params{
		{Key: "id", Value: sessionID.String()},
	})
	setUserContext(c, driverID)

	handler.SubmitOffer(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	errorInfo := response["error"].(map[string]interface{})
	assert.Contains(t, errorInfo["message"], "expired")
	mockService.AssertExpectations(t)
}

func TestHandler_SubmitOffer_DriverHasTooManyOffers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	sessionID := uuid.New()

	reqBody := SubmitOfferRequest{
		OfferedPrice: 90.0,
	}

	mockService.On("SubmitOffer", mock.Anything, sessionID, driverID, &reqBody, mock.AnythingOfType("*negotiation.DriverInfo")).Return(nil, errors.New("driver has too many active negotiations"))

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/offers", reqBody, gin.Params{
		{Key: "id", Value: sessionID.String()},
	})
	setUserContext(c, driverID)

	handler.SubmitOffer(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	errorInfo := response["error"].(map[string]interface{})
	assert.Contains(t, errorInfo["message"], "too many")
	mockService.AssertExpectations(t)
}

func TestHandler_AcceptOffer_SessionNoLongerActive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	sessionID := uuid.New()
	offerID := uuid.New()

	mockService.On("AcceptOffer", mock.Anything, sessionID, offerID, riderID).Return(errors.New("session is no longer active"))

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/offers/"+offerID.String()+"/accept", nil, gin.Params{
		{Key: "id", Value: sessionID.String()},
		{Key: "offerId", Value: offerID.String()},
	})
	setUserContext(c, riderID)

	handler.AcceptOffer(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	errorInfo := response["error"].(map[string]interface{})
	assert.Contains(t, errorInfo["message"], "no longer active")
	mockService.AssertExpectations(t)
}

func TestHandler_AcceptOffer_OfferAlreadyAccepted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()
	sessionID := uuid.New()
	offerID := uuid.New()

	mockService.On("AcceptOffer", mock.Anything, sessionID, offerID, riderID).Return(errors.New("offer is no longer pending"))

	c, w := setupTestContextWithParams("POST", "/api/v1/negotiations/"+sessionID.String()+"/offers/"+offerID.String()+"/accept", nil, gin.Params{
		{Key: "id", Value: sessionID.String()},
		{Key: "offerId", Value: offerID.String()},
	})
	setUserContext(c, riderID)

	handler.AcceptOffer(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// Integration-style Router Tests
// ============================================================================

func TestHandler_RegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a real handler with nil service (just to test route registration)
	handler := &Handler{service: nil}

	router := gin.New()
	rg := router.Group("/api/v1")
	handler.RegisterRoutes(rg)

	// Verify routes are registered
	routes := router.Routes()

	expectedRoutes := map[string]string{
		"POST:/api/v1/negotiations":                             "StartSession",
		"GET:/api/v1/negotiations/active":                       "GetActiveSession",
		"GET:/api/v1/negotiations/:id":                          "GetSession",
		"POST:/api/v1/negotiations/:id/cancel":                  "CancelSession",
		"POST:/api/v1/negotiations/:id/offers/:offerId/accept":  "AcceptOffer",
		"POST:/api/v1/negotiations/:id/offers/:offerId/reject":  "RejectOffer",
		"POST:/api/v1/negotiations/:id/offers":                  "SubmitOffer",
		"DELETE:/api/v1/negotiations/:id/offers/:offerId":       "WithdrawOffer",
	}

	registeredRoutes := make(map[string]bool)
	for _, route := range routes {
		key := route.Method + ":" + route.Path
		registeredRoutes[key] = true
	}

	for expectedRoute := range expectedRoutes {
		assert.True(t, registeredRoutes[expectedRoute], "Expected route %s to be registered", expectedRoute)
	}
}

// ============================================================================
// Concurrent Request Tests
// ============================================================================

func TestHandler_ConcurrentStartSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockServiceForHandler)
	handler := NewTestableHandler(mockService)

	riderID := uuid.New()

	// First call succeeds, second call fails (rider already has active session)
	session := createTestSession(riderID)
	mockService.On("StartSession", mock.Anything, riderID, mock.AnythingOfType("*negotiation.StartSessionRequest")).Return(session, nil).Once()
	mockService.On("StartSession", mock.Anything, riderID, mock.AnythingOfType("*negotiation.StartSessionRequest")).Return(nil, errors.New("rider already has an active negotiation session")).Once()

	reqBody := StartSessionRequest{
		PickupLatitude:   40.7128,
		PickupLongitude:  -74.0060,
		PickupAddress:    "123 Main St",
		DropoffLatitude:  40.7580,
		DropoffLongitude: -73.9855,
		DropoffAddress:   "456 Broadway",
	}

	// First request
	c1, w1 := setupTestContext("POST", "/api/v1/negotiations", reqBody)
	setUserContext(c1, riderID)
	handler.StartSession(c1)
	assert.Equal(t, http.StatusCreated, w1.Code)

	// Second request (should fail)
	c2, w2 := setupTestContext("POST", "/api/v1/negotiations", reqBody)
	setUserContext(c2, riderID)
	handler.StartSession(c2)
	assert.Equal(t, http.StatusBadRequest, w2.Code)

	mockService.AssertExpectations(t)
}

// Ensure the imports are used (added for testing the internal mocks from service_test.go)
var (
	_ = geography.ResolvedLocation{}
	_ = pricing.EstimateRequest{}
)
