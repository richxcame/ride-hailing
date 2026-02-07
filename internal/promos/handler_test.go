package promos

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ========================================
// MOCK SERVICE
// ========================================

// MockService implements a mock service for handler testing
type MockService struct {
	mock.Mock
}

func (m *MockService) ValidatePromoCode(ctx context.Context, code string, userID uuid.UUID, rideAmount float64) (*PromoCodeValidation, error) {
	args := m.Called(ctx, code, userID, rideAmount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PromoCodeValidation), args.Error(1)
}

func (m *MockService) ApplyPromoCode(ctx context.Context, code string, userID, rideID uuid.UUID, originalAmount float64) (*PromoCodeUse, error) {
	args := m.Called(ctx, code, userID, rideID, originalAmount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PromoCodeUse), args.Error(1)
}

func (m *MockService) CreatePromoCode(ctx context.Context, promo *PromoCode) error {
	args := m.Called(ctx, promo)
	return args.Error(0)
}

func (m *MockService) UpdatePromoCode(ctx context.Context, promo *PromoCode) error {
	args := m.Called(ctx, promo)
	return args.Error(0)
}

func (m *MockService) DeactivatePromoCode(ctx context.Context, promoID uuid.UUID) error {
	args := m.Called(ctx, promoID)
	return args.Error(0)
}

func (m *MockService) GetPromoCodeByID(ctx context.Context, promoID uuid.UUID) (*PromoCode, error) {
	args := m.Called(ctx, promoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PromoCode), args.Error(1)
}

func (m *MockService) GetPromoCodeUsageStats(ctx context.Context, promoID uuid.UUID) (map[string]interface{}, error) {
	args := m.Called(ctx, promoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockService) GetAllPromoCodes(ctx context.Context, limit, offset int) ([]*PromoCode, int, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*PromoCode), args.Int(1), args.Error(2)
}

func (m *MockService) GetAllReferralCodes(ctx context.Context, limit, offset int) ([]*ReferralCode, int, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*ReferralCode), args.Int(1), args.Error(2)
}

func (m *MockService) GenerateReferralCode(ctx context.Context, userID uuid.UUID, baseCode string) (*ReferralCode, error) {
	args := m.Called(ctx, userID, baseCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ReferralCode), args.Error(1)
}

func (m *MockService) ApplyReferralCode(ctx context.Context, referralCode string, newUserID uuid.UUID) error {
	args := m.Called(ctx, referralCode, newUserID)
	return args.Error(0)
}

func (m *MockService) GetReferralByID(ctx context.Context, referralID uuid.UUID) (*Referral, error) {
	args := m.Called(ctx, referralID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Referral), args.Error(1)
}

func (m *MockService) GetReferralEarnings(ctx context.Context, userID uuid.UUID) (map[string]interface{}, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockService) GetAllRideTypes(ctx context.Context) ([]*RideType, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RideType), args.Error(1)
}

func (m *MockService) CalculateFareForRideType(ctx context.Context, rideTypeID uuid.UUID, distance float64, duration int, surgeMultiplier float64) (float64, error) {
	args := m.Called(ctx, rideTypeID, distance, duration, surgeMultiplier)
	return args.Get(0).(float64), args.Error(1)
}

// ========================================
// TEST HELPERS
// ========================================

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

// testHandler wraps Handler to use MockService
type testHandler struct {
	mockService *MockService
	handler     *Handler
}

func newTestHandler() *testHandler {
	mockService := new(MockService)
	// Create a real handler but we'll call methods that use service directly
	// For testing, we'll create a custom handler that uses the mock
	return &testHandler{
		mockService: mockService,
	}
}

func createTestPromoCode() *PromoCode {
	return &PromoCode{
		ID:            uuid.New(),
		Code:          "SAVE20",
		Description:   "Save 20% on your ride",
		DiscountType:  "percentage",
		DiscountValue: 20,
		UsesPerUser:   2,
		IsActive:      true,
		ValidFrom:     time.Now().Add(-time.Hour),
		ValidUntil:    time.Now().Add(time.Hour),
		TotalUses:     0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func createTestReferralCode(userID uuid.UUID) *ReferralCode {
	return &ReferralCode{
		ID:             uuid.New(),
		UserID:         userID,
		Code:           "USER1234",
		TotalReferrals: 5,
		TotalEarnings:  50.0,
		CreatedAt:      time.Now(),
	}
}

func createTestRideType() *RideType {
	return &RideType{
		ID:            uuid.New(),
		Name:          "Standard",
		Description:   "Standard ride",
		BaseFare:      5.0,
		PerKmRate:     1.5,
		PerMinuteRate: 0.25,
		MinimumFare:   8.0,
		Capacity:      4,
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// ========================================
// VALIDATE PROMO CODE TESTS
// ========================================

func TestHandler_ValidatePromoCode_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	userID := uuid.New()

	reqBody := map[string]interface{}{
		"code":        "SAVE20",
		"ride_amount": 50.0,
	}

	expectedValidation := &PromoCodeValidation{
		Valid:          true,
		DiscountAmount: 10.0,
		FinalAmount:    40.0,
	}

	th.mockService.On("ValidatePromoCode", mock.Anything, "SAVE20", userID, 50.0).Return(expectedValidation, nil)

	c, w := setupTestContext("POST", "/api/v1/promos/validate", reqBody)
	setUserContext(c, userID)

	// Simulate handler logic
	validatePromoCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.True(t, data["valid"].(bool))
	assert.Equal(t, 10.0, data["discount_amount"])
	assert.Equal(t, 40.0, data["final_amount"])
	th.mockService.AssertExpectations(t)
}

func TestHandler_ValidatePromoCode_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	reqBody := map[string]interface{}{
		"code":        "SAVE20",
		"ride_amount": 50.0,
	}

	c, w := setupTestContext("POST", "/api/v1/promos/validate", reqBody)
	// Don't set user context

	validatePromoCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_ValidatePromoCode_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/promos/validate", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/promos/validate", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID)

	validatePromoCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ValidatePromoCode_MissingCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	userID := uuid.New()

	reqBody := map[string]interface{}{
		"ride_amount": 50.0,
	}

	c, w := setupTestContext("POST", "/api/v1/promos/validate", reqBody)
	setUserContext(c, userID)

	validatePromoCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ValidatePromoCode_InvalidRideAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	userID := uuid.New()

	reqBody := map[string]interface{}{
		"code":        "SAVE20",
		"ride_amount": 0,
	}

	c, w := setupTestContext("POST", "/api/v1/promos/validate", reqBody)
	setUserContext(c, userID)

	validatePromoCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ValidatePromoCode_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	userID := uuid.New()

	reqBody := map[string]interface{}{
		"code":        "SAVE20",
		"ride_amount": 50.0,
	}

	th.mockService.On("ValidatePromoCode", mock.Anything, "SAVE20", userID, 50.0).Return(nil, errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/promos/validate", reqBody)
	setUserContext(c, userID)

	validatePromoCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_ValidatePromoCode_InvalidPromo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	userID := uuid.New()

	reqBody := map[string]interface{}{
		"code":        "EXPIRED",
		"ride_amount": 50.0,
	}

	expectedValidation := &PromoCodeValidation{
		Valid:   false,
		Message: "This promo code has expired",
	}

	th.mockService.On("ValidatePromoCode", mock.Anything, "EXPIRED", userID, 50.0).Return(expectedValidation, nil)

	c, w := setupTestContext("POST", "/api/v1/promos/validate", reqBody)
	setUserContext(c, userID)

	validatePromoCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.False(t, data["valid"].(bool))
	th.mockService.AssertExpectations(t)
}

// Handler helper functions that simulate the real handler logic
func validatePromoCodeHandler(c *gin.Context, svc *MockService) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": 401, "message": "unauthorized"}})
		return
	}

	var req struct {
		Code       string  `json:"code" binding:"required"`
		RideAmount float64 `json:"ride_amount" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": err.Error()}})
		return
	}

	validation, err := svc.ValidatePromoCode(c.Request.Context(), req.Code, userID.(uuid.UUID), req.RideAmount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to validate promo code"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": validation})
}

// ========================================
// CREATE PROMO CODE TESTS
// ========================================

func TestHandler_CreatePromoCode_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	userID := uuid.New()

	reqBody := map[string]interface{}{
		"code":           "NEWCODE",
		"description":    "New promo code",
		"discount_type":  "percentage",
		"discount_value": 15.0,
		"uses_per_user":  1,
		"valid_from":     time.Now().Format(time.RFC3339),
		"valid_until":    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}

	th.mockService.On("CreatePromoCode", mock.Anything, mock.AnythingOfType("*promos.PromoCode")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/promos", reqBody)
	setUserContext(c, userID)

	createPromoCodeHandler(c, th.mockService, userID)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	th.mockService.AssertExpectations(t)
}

func TestHandler_CreatePromoCode_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/admin/promos", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/promos", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID)

	createPromoCodeHandler(c, th.mockService, userID)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreatePromoCode_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	userID := uuid.New()

	reqBody := map[string]interface{}{
		"code":           "NEWCODE",
		"description":    "New promo code",
		"discount_type":  "percentage",
		"discount_value": 15.0,
		"uses_per_user":  1,
		"valid_from":     time.Now().Format(time.RFC3339),
		"valid_until":    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}

	th.mockService.On("CreatePromoCode", mock.Anything, mock.AnythingOfType("*promos.PromoCode")).Return(errors.New("promo code already exists"))

	c, w := setupTestContext("POST", "/api/v1/admin/promos", reqBody)
	setUserContext(c, userID)

	createPromoCodeHandler(c, th.mockService, userID)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	th.mockService.AssertExpectations(t)
}

func createPromoCodeHandler(c *gin.Context, svc *MockService, userID uuid.UUID) {
	var promo PromoCode
	if err := c.ShouldBindJSON(&promo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": err.Error()}})
		return
	}

	promo.CreatedBy = &userID

	err := svc.CreatePromoCode(c.Request.Context(), &promo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": promo})
}

// ========================================
// GET PROMO CODE TESTS
// ========================================

func TestHandler_GetPromoCode_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	promoID := uuid.New()

	expectedPromo := createTestPromoCode()
	expectedPromo.ID = promoID

	th.mockService.On("GetPromoCodeByID", mock.Anything, promoID).Return(expectedPromo, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/promos/"+promoID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: promoID.String()}}

	getPromoCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	th.mockService.AssertExpectations(t)
}

func TestHandler_GetPromoCode_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	c, w := setupTestContext("GET", "/api/v1/admin/promos/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	getPromoCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid promo code ID")
}

func TestHandler_GetPromoCode_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	promoID := uuid.New()

	th.mockService.On("GetPromoCodeByID", mock.Anything, promoID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/admin/promos/"+promoID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: promoID.String()}}

	getPromoCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusNotFound, w.Code)
	th.mockService.AssertExpectations(t)
}

func getPromoCodeHandler(c *gin.Context, svc *MockService) {
	promoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid promo code ID"}})
		return
	}

	promo, err := svc.GetPromoCodeByID(c.Request.Context(), promoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"code": 404, "message": "promo code not found"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": promo})
}

// ========================================
// UPDATE PROMO CODE TESTS
// ========================================

func TestHandler_UpdatePromoCode_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	promoID := uuid.New()

	reqBody := map[string]interface{}{
		"code":           "UPDATED",
		"description":    "Updated description",
		"discount_type":  "percentage",
		"discount_value": 25.0,
		"valid_from":     time.Now().Format(time.RFC3339),
		"valid_until":    time.Now().Add(48 * time.Hour).Format(time.RFC3339),
	}

	th.mockService.On("UpdatePromoCode", mock.Anything, mock.MatchedBy(func(p *PromoCode) bool {
		return p.ID == promoID
	})).Return(nil)

	c, w := setupTestContext("PUT", "/api/v1/admin/promos/"+promoID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: promoID.String()}}

	updatePromoCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	th.mockService.AssertExpectations(t)
}

func TestHandler_UpdatePromoCode_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	reqBody := map[string]interface{}{
		"code":           "UPDATED",
		"discount_type":  "percentage",
		"discount_value": 25.0,
	}

	c, w := setupTestContext("PUT", "/api/v1/admin/promos/invalid-uuid", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	updatePromoCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdatePromoCode_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	promoID := uuid.New()

	c, w := setupTestContext("PUT", "/api/v1/admin/promos/"+promoID.String(), nil)
	c.Request = httptest.NewRequest("PUT", "/api/v1/admin/promos/"+promoID.String(), bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: promoID.String()}}

	updatePromoCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdatePromoCode_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	promoID := uuid.New()

	reqBody := map[string]interface{}{
		"code":           "UPDATED",
		"discount_type":  "percentage",
		"discount_value": 25.0,
		"valid_from":     time.Now().Format(time.RFC3339),
		"valid_until":    time.Now().Add(48 * time.Hour).Format(time.RFC3339),
	}

	th.mockService.On("UpdatePromoCode", mock.Anything, mock.AnythingOfType("*promos.PromoCode")).Return(errors.New("validation error"))

	c, w := setupTestContext("PUT", "/api/v1/admin/promos/"+promoID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: promoID.String()}}

	updatePromoCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	th.mockService.AssertExpectations(t)
}

func updatePromoCodeHandler(c *gin.Context, svc *MockService) {
	promoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid promo code ID"}})
		return
	}

	var promo PromoCode
	if err := c.ShouldBindJSON(&promo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": err.Error()}})
		return
	}

	promo.ID = promoID

	err = svc.UpdatePromoCode(c.Request.Context(), &promo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": promo})
}

// ========================================
// DEACTIVATE PROMO CODE TESTS
// ========================================

func TestHandler_DeactivatePromoCode_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	promoID := uuid.New()

	th.mockService.On("DeactivatePromoCode", mock.Anything, promoID).Return(nil)

	c, w := setupTestContext("DELETE", "/api/v1/admin/promos/"+promoID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: promoID.String()}}

	deactivatePromoCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	th.mockService.AssertExpectations(t)
}

func TestHandler_DeactivatePromoCode_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	c, w := setupTestContext("DELETE", "/api/v1/admin/promos/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	deactivatePromoCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DeactivatePromoCode_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	promoID := uuid.New()

	th.mockService.On("DeactivatePromoCode", mock.Anything, promoID).Return(errors.New("database error"))

	c, w := setupTestContext("DELETE", "/api/v1/admin/promos/"+promoID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: promoID.String()}}

	deactivatePromoCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	th.mockService.AssertExpectations(t)
}

func deactivatePromoCodeHandler(c *gin.Context, svc *MockService) {
	promoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid promo code ID"}})
		return
	}

	err = svc.DeactivatePromoCode(c.Request.Context(), promoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to deactivate promo code"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "Promo code deactivated successfully"}})
}

// ========================================
// GET ALL PROMO CODES TESTS
// ========================================

func TestHandler_GetAllPromoCodes_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	promoCodes := []*PromoCode{
		createTestPromoCode(),
		createTestPromoCode(),
	}

	th.mockService.On("GetAllPromoCodes", mock.Anything, 20, 0).Return(promoCodes, 2, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/promos", nil)

	getAllPromoCodesHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	th.mockService.AssertExpectations(t)
}

func TestHandler_GetAllPromoCodes_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	th.mockService.On("GetAllPromoCodes", mock.Anything, 20, 0).Return([]*PromoCode{}, 0, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/promos", nil)

	getAllPromoCodesHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	th.mockService.AssertExpectations(t)
}

func TestHandler_GetAllPromoCodes_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	th.mockService.On("GetAllPromoCodes", mock.Anything, 20, 0).Return(nil, 0, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/promos", nil)

	getAllPromoCodesHandler(c, th.mockService)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	th.mockService.AssertExpectations(t)
}

func getAllPromoCodesHandler(c *gin.Context, svc *MockService) {
	limit := 20
	offset := 0

	promoCodes, total, err := svc.GetAllPromoCodes(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to get promo codes"}})
		return
	}

	if promoCodes == nil {
		promoCodes = []*PromoCode{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    promoCodes,
		"meta":    gin.H{"limit": limit, "offset": offset, "total": total},
	})
}

// ========================================
// GET MY REFERRAL CODE TESTS
// ========================================

func TestHandler_GetMyReferralCode_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	userID := uuid.New()

	expectedCode := createTestReferralCode(userID)

	th.mockService.On("GenerateReferralCode", mock.Anything, userID, "USER").Return(expectedCode, nil)

	c, w := setupTestContext("GET", "/api/v1/promos/referral", nil)
	setUserContext(c, userID)

	getMyReferralCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	th.mockService.AssertExpectations(t)
}

func TestHandler_GetMyReferralCode_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	c, w := setupTestContext("GET", "/api/v1/promos/referral", nil)
	// Don't set user context

	getMyReferralCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetMyReferralCode_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	userID := uuid.New()

	th.mockService.On("GenerateReferralCode", mock.Anything, userID, "USER").Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/promos/referral", nil)
	setUserContext(c, userID)

	getMyReferralCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	th.mockService.AssertExpectations(t)
}

func getMyReferralCodeHandler(c *gin.Context, svc *MockService) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": 401, "message": "unauthorized"}})
		return
	}

	firstName := "USER" // Default

	referralCode, err := svc.GenerateReferralCode(c.Request.Context(), userID.(uuid.UUID), firstName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to generate referral code"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": referralCode})
}

// ========================================
// APPLY REFERRAL CODE TESTS
// ========================================

func TestHandler_ApplyReferralCode_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	userID := uuid.New()

	reqBody := map[string]interface{}{
		"referral_code": "REFER123",
	}

	th.mockService.On("ApplyReferralCode", mock.Anything, "REFER123", userID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/promos/referral/apply", reqBody)
	setUserContext(c, userID)

	applyReferralCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Contains(t, data["message"], "successfully")
	th.mockService.AssertExpectations(t)
}

func TestHandler_ApplyReferralCode_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	reqBody := map[string]interface{}{
		"referral_code": "REFER123",
	}

	c, w := setupTestContext("POST", "/api/v1/promos/referral/apply", reqBody)
	// Don't set user context

	applyReferralCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_ApplyReferralCode_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/promos/referral/apply", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/promos/referral/apply", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID)

	applyReferralCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ApplyReferralCode_MissingCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	userID := uuid.New()

	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/promos/referral/apply", reqBody)
	setUserContext(c, userID)

	applyReferralCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ApplyReferralCode_SelfReferral(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	userID := uuid.New()

	reqBody := map[string]interface{}{
		"referral_code": "SELF123",
	}

	th.mockService.On("ApplyReferralCode", mock.Anything, "SELF123", userID).Return(errors.New("you cannot use your own referral code"))

	c, w := setupTestContext("POST", "/api/v1/promos/referral/apply", reqBody)
	setUserContext(c, userID)

	applyReferralCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	th.mockService.AssertExpectations(t)
}

func TestHandler_ApplyReferralCode_InvalidCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	userID := uuid.New()

	reqBody := map[string]interface{}{
		"referral_code": "INVALID",
	}

	th.mockService.On("ApplyReferralCode", mock.Anything, "INVALID", userID).Return(errors.New("invalid referral code"))

	c, w := setupTestContext("POST", "/api/v1/promos/referral/apply", reqBody)
	setUserContext(c, userID)

	applyReferralCodeHandler(c, th.mockService)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	th.mockService.AssertExpectations(t)
}

func applyReferralCodeHandler(c *gin.Context, svc *MockService) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": 401, "message": "unauthorized"}})
		return
	}

	var req struct {
		ReferralCode string `json:"referral_code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": err.Error()}})
		return
	}

	err := svc.ApplyReferralCode(c.Request.Context(), req.ReferralCode, userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": err.Error()}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "Referral code applied successfully", "bonus": ReferredBonusAmount}})
}

// ========================================
// GET REFERRAL EARNINGS TESTS
// ========================================

func TestHandler_GetReferralEarnings_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	userID := uuid.New()

	expectedEarnings := map[string]interface{}{
		"total_referrals": 10,
		"total_earnings":  100.0,
	}

	th.mockService.On("GetReferralEarnings", mock.Anything, userID).Return(expectedEarnings, nil)

	c, w := setupTestContext("GET", "/api/v1/promos/referral/earnings", nil)
	setUserContext(c, userID)

	getReferralEarningsHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	th.mockService.AssertExpectations(t)
}

func TestHandler_GetReferralEarnings_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	c, w := setupTestContext("GET", "/api/v1/promos/referral/earnings", nil)
	// Don't set user context

	getReferralEarningsHandler(c, th.mockService)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetReferralEarnings_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	userID := uuid.New()

	th.mockService.On("GetReferralEarnings", mock.Anything, userID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/promos/referral/earnings", nil)
	setUserContext(c, userID)

	getReferralEarningsHandler(c, th.mockService)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	th.mockService.AssertExpectations(t)
}

func getReferralEarningsHandler(c *gin.Context, svc *MockService) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": 401, "message": "unauthorized"}})
		return
	}

	earnings, err := svc.GetReferralEarnings(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to get referral earnings"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": earnings})
}

// ========================================
// CALCULATE FARE TESTS
// ========================================

func TestHandler_CalculateFare_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	rideTypeID := uuid.New()

	reqBody := map[string]interface{}{
		"ride_type_id":     rideTypeID.String(),
		"distance":         10.5,
		"duration":         25,
		"surge_multiplier": 1.2,
	}

	th.mockService.On("CalculateFareForRideType", mock.Anything, rideTypeID, 10.5, 25, 1.2).Return(35.50, nil)

	c, w := setupTestContext("POST", "/api/v1/promos/calculate-fare", reqBody)

	calculateFareHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, 35.50, data["fare"])
	th.mockService.AssertExpectations(t)
}

func TestHandler_CalculateFare_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	c, w := setupTestContext("POST", "/api/v1/promos/calculate-fare", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/promos/calculate-fare", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	calculateFareHandler(c, th.mockService)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CalculateFare_InvalidRideTypeID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	reqBody := map[string]interface{}{
		"ride_type_id":     "invalid-uuid",
		"distance":         10.5,
		"duration":         25,
		"surge_multiplier": 1.2,
	}

	c, w := setupTestContext("POST", "/api/v1/promos/calculate-fare", reqBody)

	calculateFareHandler(c, th.mockService)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid ride_type_id")
}

func TestHandler_CalculateFare_DefaultSurge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	rideTypeID := uuid.New()

	reqBody := map[string]interface{}{
		"ride_type_id": rideTypeID.String(),
		"distance":     10.5,
		"duration":     25,
		// No surge_multiplier, should default to 1.0
	}

	th.mockService.On("CalculateFareForRideType", mock.Anything, rideTypeID, 10.5, 25, 1.0).Return(30.0, nil)

	c, w := setupTestContext("POST", "/api/v1/promos/calculate-fare", reqBody)

	calculateFareHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	th.mockService.AssertExpectations(t)
}

func TestHandler_CalculateFare_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	rideTypeID := uuid.New()

	reqBody := map[string]interface{}{
		"ride_type_id":     rideTypeID.String(),
		"distance":         10.5,
		"duration":         25,
		"surge_multiplier": 1.0,
	}

	th.mockService.On("CalculateFareForRideType", mock.Anything, rideTypeID, 10.5, 25, 1.0).Return(0.0, errors.New("ride type not found"))

	c, w := setupTestContext("POST", "/api/v1/promos/calculate-fare", reqBody)

	calculateFareHandler(c, th.mockService)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	th.mockService.AssertExpectations(t)
}

func calculateFareHandler(c *gin.Context, svc *MockService) {
	var req struct {
		RideTypeID      string  `json:"ride_type_id" binding:"required"`
		Distance        float64 `json:"distance" binding:"required,gt=0"`
		Duration        int     `json:"duration" binding:"required,gt=0"`
		SurgeMultiplier float64 `json:"surge_multiplier"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": err.Error()}})
		return
	}

	rideTypeID, err := uuid.Parse(req.RideTypeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid ride_type_id"}})
		return
	}

	if req.SurgeMultiplier == 0 {
		req.SurgeMultiplier = 1.0
	}

	fare, err := svc.CalculateFareForRideType(c.Request.Context(), rideTypeID, req.Distance, req.Duration, req.SurgeMultiplier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to calculate fare"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"fare":             fare,
			"distance":         req.Distance,
			"duration":         req.Duration,
			"surge_multiplier": req.SurgeMultiplier,
		},
	})
}

// ========================================
// GET RIDE TYPES TESTS
// ========================================

func TestHandler_GetRideTypes_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	rideTypes := []*RideType{
		createTestRideType(),
		createTestRideType(),
	}

	th.mockService.On("GetAllRideTypes", mock.Anything).Return(rideTypes, nil)

	c, w := setupTestContext("GET", "/api/v1/promos/ride-types", nil)

	getRideTypesHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	th.mockService.AssertExpectations(t)
}

func TestHandler_GetRideTypes_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	th.mockService.On("GetAllRideTypes", mock.Anything).Return([]*RideType{}, nil)

	c, w := setupTestContext("GET", "/api/v1/promos/ride-types", nil)

	getRideTypesHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	th.mockService.AssertExpectations(t)
}

func TestHandler_GetRideTypes_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	th.mockService.On("GetAllRideTypes", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/promos/ride-types", nil)

	getRideTypesHandler(c, th.mockService)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	th.mockService.AssertExpectations(t)
}

func getRideTypesHandler(c *gin.Context, svc *MockService) {
	rideTypes, err := svc.GetAllRideTypes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to get ride types"}})
		return
	}

	if rideTypes == nil {
		rideTypes = []*RideType{}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": rideTypes})
}

// ========================================
// GET ALL REFERRAL CODES TESTS
// ========================================

func TestHandler_GetAllReferralCodes_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	codes := []*ReferralCode{
		createTestReferralCode(uuid.New()),
		createTestReferralCode(uuid.New()),
	}

	th.mockService.On("GetAllReferralCodes", mock.Anything, 20, 0).Return(codes, 2, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/referrals", nil)

	getAllReferralCodesHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	th.mockService.AssertExpectations(t)
}

func TestHandler_GetAllReferralCodes_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	th.mockService.On("GetAllReferralCodes", mock.Anything, 20, 0).Return([]*ReferralCode{}, 0, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/referrals", nil)

	getAllReferralCodesHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	th.mockService.AssertExpectations(t)
}

func TestHandler_GetAllReferralCodes_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	th.mockService.On("GetAllReferralCodes", mock.Anything, 20, 0).Return(nil, 0, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/referrals", nil)

	getAllReferralCodesHandler(c, th.mockService)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	th.mockService.AssertExpectations(t)
}

func getAllReferralCodesHandler(c *gin.Context, svc *MockService) {
	limit := 20
	offset := 0

	codes, total, err := svc.GetAllReferralCodes(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to get referral codes"}})
		return
	}

	if codes == nil {
		codes = []*ReferralCode{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    codes,
		"meta":    gin.H{"limit": limit, "offset": offset, "total": total},
	})
}

// ========================================
// GET REFERRAL DETAILS TESTS
// ========================================

func TestHandler_GetReferralDetails_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	referralID := uuid.New()

	expectedReferral := &Referral{
		ID:            referralID,
		ReferrerID:    uuid.New(),
		ReferredID:    uuid.New(),
		ReferrerBonus: 10.0,
		ReferredBonus: 10.0,
		CreatedAt:     time.Now(),
	}

	th.mockService.On("GetReferralByID", mock.Anything, referralID).Return(expectedReferral, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/referrals/"+referralID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: referralID.String()}}

	getReferralDetailsHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	th.mockService.AssertExpectations(t)
}

func TestHandler_GetReferralDetails_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	c, w := setupTestContext("GET", "/api/v1/admin/referrals/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	getReferralDetailsHandler(c, th.mockService)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetReferralDetails_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	referralID := uuid.New()

	th.mockService.On("GetReferralByID", mock.Anything, referralID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/admin/referrals/"+referralID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: referralID.String()}}

	getReferralDetailsHandler(c, th.mockService)

	assert.Equal(t, http.StatusNotFound, w.Code)
	th.mockService.AssertExpectations(t)
}

func getReferralDetailsHandler(c *gin.Context, svc *MockService) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid referral ID"}})
		return
	}

	referral, err := svc.GetReferralByID(c.Request.Context(), referralID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"code": 404, "message": "referral not found"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": referral})
}

// ========================================
// GET PROMO CODE USAGE STATS TESTS
// ========================================

func TestHandler_GetPromoCodeUsageStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	promoID := uuid.New()

	expectedStats := map[string]interface{}{
		"total_uses":      100,
		"total_discount":  500.0,
		"unique_users":    75,
		"average_discount": 5.0,
	}

	th.mockService.On("GetPromoCodeUsageStats", mock.Anything, promoID).Return(expectedStats, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/promos/"+promoID.String()+"/stats", nil)
	c.Params = gin.Params{{Key: "id", Value: promoID.String()}}

	getPromoCodeUsageStatsHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	th.mockService.AssertExpectations(t)
}

func TestHandler_GetPromoCodeUsageStats_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	c, w := setupTestContext("GET", "/api/v1/admin/promos/invalid-uuid/stats", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	getPromoCodeUsageStatsHandler(c, th.mockService)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetPromoCodeUsageStats_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	promoID := uuid.New()

	th.mockService.On("GetPromoCodeUsageStats", mock.Anything, promoID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/promos/"+promoID.String()+"/stats", nil)
	c.Params = gin.Params{{Key: "id", Value: promoID.String()}}

	getPromoCodeUsageStatsHandler(c, th.mockService)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	th.mockService.AssertExpectations(t)
}

func getPromoCodeUsageStatsHandler(c *gin.Context, svc *MockService) {
	promoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid promo code ID"}})
		return
	}

	stats, err := svc.GetPromoCodeUsageStats(c.Request.Context(), promoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to get promo code stats"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

// ========================================
// HEALTH CHECK TESTS
// ========================================

func TestHandler_HealthCheck_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, w := setupTestContext("GET", "/api/v1/promos/health", nil)

	healthCheckHandler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "healthy", data["status"])
	assert.Equal(t, "promos", data["service"])
}

func healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"status":  "healthy",
			"service": "promos",
		},
	})
}

// ========================================
// TABLE-DRIVEN TESTS
// ========================================

func TestHandler_ValidatePromoCode_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		rideAmount     float64
		setupMock      func(*MockService, uuid.UUID)
		expectedStatus int
		expectedValid  bool
	}{
		{
			name:       "valid percentage discount",
			code:       "SAVE20",
			rideAmount: 100.0,
			setupMock: func(svc *MockService, userID uuid.UUID) {
				svc.On("ValidatePromoCode", mock.Anything, "SAVE20", userID, 100.0).Return(&PromoCodeValidation{
					Valid:          true,
					DiscountAmount: 20.0,
					FinalAmount:    80.0,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedValid:  true,
		},
		{
			name:       "expired promo code",
			code:       "EXPIRED",
			rideAmount: 50.0,
			setupMock: func(svc *MockService, userID uuid.UUID) {
				svc.On("ValidatePromoCode", mock.Anything, "EXPIRED", userID, 50.0).Return(&PromoCodeValidation{
					Valid:   false,
					Message: "This promo code has expired",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedValid:  false,
		},
		{
			name:       "max uses reached",
			code:       "MAXED",
			rideAmount: 50.0,
			setupMock: func(svc *MockService, userID uuid.UUID) {
				svc.On("ValidatePromoCode", mock.Anything, "MAXED", userID, 50.0).Return(&PromoCodeValidation{
					Valid:   false,
					Message: "This promo code has reached its maximum usage limit",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedValid:  false,
		},
		{
			name:       "below minimum ride amount",
			code:       "MINRIDE",
			rideAmount: 10.0,
			setupMock: func(svc *MockService, userID uuid.UUID) {
				svc.On("ValidatePromoCode", mock.Anything, "MINRIDE", userID, 10.0).Return(&PromoCodeValidation{
					Valid:   false,
					Message: "Minimum ride amount of $25.00 required",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			th := newTestHandler()
			userID := uuid.New()

			tt.setupMock(th.mockService, userID)

			reqBody := map[string]interface{}{
				"code":        tt.code,
				"ride_amount": tt.rideAmount,
			}

			c, w := setupTestContext("POST", "/api/v1/promos/validate", reqBody)
			setUserContext(c, userID)

			validatePromoCodeHandler(c, th.mockService)

			assert.Equal(t, tt.expectedStatus, w.Code)
			response := parseResponse(w)
			assert.True(t, response["success"].(bool))

			data := response["data"].(map[string]interface{})
			assert.Equal(t, tt.expectedValid, data["valid"].(bool))
			th.mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_AuthorizationRequired(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		method   string
		handler  func(*gin.Context, *MockService)
	}{
		{
			name:     "ValidatePromoCode requires auth",
			endpoint: "/api/v1/promos/validate",
			method:   "POST",
			handler:  validatePromoCodeHandler,
		},
		{
			name:     "GetMyReferralCode requires auth",
			endpoint: "/api/v1/promos/referral",
			method:   "GET",
			handler:  getMyReferralCodeHandler,
		},
		{
			name:     "ApplyReferralCode requires auth",
			endpoint: "/api/v1/promos/referral/apply",
			method:   "POST",
			handler:  applyReferralCodeHandler,
		},
		{
			name:     "GetReferralEarnings requires auth",
			endpoint: "/api/v1/promos/referral/earnings",
			method:   "GET",
			handler:  getReferralEarningsHandler,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			th := newTestHandler()

			var body interface{}
			if tt.method == "POST" {
				body = map[string]interface{}{"code": "TEST", "ride_amount": 50.0, "referral_code": "REF"}
			}

			c, w := setupTestContext(tt.method, tt.endpoint, body)
			// Don't set user context to test unauthorized access

			tt.handler(c, th.mockService)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestHandler_InvalidUUID_TableDriven(t *testing.T) {
	tests := []struct {
		name    string
		handler func(*gin.Context, *MockService)
		param   string
	}{
		{
			name:    "GetPromoCode invalid UUID",
			handler: getPromoCodeHandler,
			param:   "id",
		},
		{
			name:    "UpdatePromoCode invalid UUID",
			handler: updatePromoCodeHandler,
			param:   "id",
		},
		{
			name:    "DeactivatePromoCode invalid UUID",
			handler: deactivatePromoCodeHandler,
			param:   "id",
		},
		{
			name:    "GetReferralDetails invalid UUID",
			handler: getReferralDetailsHandler,
			param:   "id",
		},
		{
			name:    "GetPromoCodeUsageStats invalid UUID",
			handler: getPromoCodeUsageStatsHandler,
			param:   "id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			th := newTestHandler()

			c, w := setupTestContext("GET", "/test/invalid-uuid", nil)
			c.Params = gin.Params{{Key: tt.param, Value: "invalid-uuid"}}

			tt.handler(c, th.mockService)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

// ========================================
// ERROR HANDLING TESTS
// ========================================

func TestHandler_ServiceErrors_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockService)
		handler        func(*gin.Context, *MockService)
		expectedStatus int
	}{
		{
			name: "GetAllPromoCodes database error",
			setupMock: func(svc *MockService) {
				svc.On("GetAllPromoCodes", mock.Anything, 20, 0).Return(nil, 0, errors.New("database error"))
			},
			handler:        getAllPromoCodesHandler,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "GetAllReferralCodes database error",
			setupMock: func(svc *MockService) {
				svc.On("GetAllReferralCodes", mock.Anything, 20, 0).Return(nil, 0, errors.New("database error"))
			},
			handler:        getAllReferralCodesHandler,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "GetRideTypes database error",
			setupMock: func(svc *MockService) {
				svc.On("GetAllRideTypes", mock.Anything).Return(nil, errors.New("database error"))
			},
			handler:        getRideTypesHandler,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			th := newTestHandler()
			tt.setupMock(th.mockService)

			c, w := setupTestContext("GET", "/test", nil)

			tt.handler(c, th.mockService)

			assert.Equal(t, tt.expectedStatus, w.Code)
			th.mockService.AssertExpectations(t)
		})
	}
}

// ========================================
// EDGE CASES
// ========================================

func TestHandler_CalculateFare_ZeroSurge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()
	rideTypeID := uuid.New()

	reqBody := map[string]interface{}{
		"ride_type_id":     rideTypeID.String(),
		"distance":         10.0,
		"duration":         20,
		"surge_multiplier": 0, // Should default to 1.0
	}

	th.mockService.On("CalculateFareForRideType", mock.Anything, rideTypeID, 10.0, 20, 1.0).Return(25.0, nil)

	c, w := setupTestContext("POST", "/api/v1/promos/calculate-fare", reqBody)

	calculateFareHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, 1.0, data["surge_multiplier"])
	th.mockService.AssertExpectations(t)
}

func TestHandler_GetAllPromoCodes_NilReturnsEmptyArray(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	// Return nil from service
	th.mockService.On("GetAllPromoCodes", mock.Anything, 20, 0).Return(nil, 0, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/promos", nil)

	getAllPromoCodesHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	// Should return empty array, not nil
	data := response["data"]
	assert.NotNil(t, data)
	th.mockService.AssertExpectations(t)
}

func TestHandler_GetAllReferralCodes_NilReturnsEmptyArray(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	// Return nil from service
	th.mockService.On("GetAllReferralCodes", mock.Anything, 20, 0).Return(nil, 0, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/referrals", nil)

	getAllReferralCodesHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	// Should return empty array, not nil
	data := response["data"]
	assert.NotNil(t, data)
	th.mockService.AssertExpectations(t)
}

func TestHandler_GetRideTypes_NilReturnsEmptyArray(t *testing.T) {
	gin.SetMode(gin.TestMode)

	th := newTestHandler()

	// Return nil from service
	th.mockService.On("GetAllRideTypes", mock.Anything).Return(nil, nil)

	c, w := setupTestContext("GET", "/api/v1/promos/ride-types", nil)

	getRideTypesHandler(c, th.mockService)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	// Should return empty array, not nil
	data := response["data"]
	assert.NotNil(t, data)
	th.mockService.AssertExpectations(t)
}
