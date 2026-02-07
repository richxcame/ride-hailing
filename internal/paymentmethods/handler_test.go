package paymentmethods

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
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Mock Repository
// ============================================================================

// MockRepository is a mock implementation of RepositoryInterface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreatePaymentMethod(ctx context.Context, pm *PaymentMethod) error {
	args := m.Called(ctx, pm)
	return args.Error(0)
}

func (m *MockRepository) GetPaymentMethodsByUser(ctx context.Context, userID uuid.UUID) ([]PaymentMethod, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]PaymentMethod), args.Error(1)
}

func (m *MockRepository) GetPaymentMethodByID(ctx context.Context, id uuid.UUID) (*PaymentMethod, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PaymentMethod), args.Error(1)
}

func (m *MockRepository) GetDefaultPaymentMethod(ctx context.Context, userID uuid.UUID) (*PaymentMethod, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PaymentMethod), args.Error(1)
}

func (m *MockRepository) SetDefault(ctx context.Context, userID, methodID uuid.UUID) error {
	args := m.Called(ctx, userID, methodID)
	return args.Error(0)
}

func (m *MockRepository) DeactivatePaymentMethod(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockRepository) GetWallet(ctx context.Context, userID uuid.UUID) (*PaymentMethod, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PaymentMethod), args.Error(1)
}

func (m *MockRepository) EnsureWalletExists(ctx context.Context, userID uuid.UUID) (*PaymentMethod, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PaymentMethod), args.Error(1)
}

func (m *MockRepository) UpdateWalletBalance(ctx context.Context, walletID uuid.UUID, delta float64) (float64, error) {
	args := m.Called(ctx, walletID, delta)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockRepository) GetWalletBalance(ctx context.Context, userID uuid.UUID) (float64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockRepository) CreateWalletTransaction(ctx context.Context, tx *WalletTransaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockRepository) GetWalletTransactions(ctx context.Context, userID uuid.UUID, limit int) ([]WalletTransaction, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]WalletTransaction), args.Error(1)
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

func createTestPaymentMethod(userID uuid.UUID, pmType PaymentMethodType, isDefault bool) *PaymentMethod {
	now := time.Now()
	return &PaymentMethod{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      pmType,
		IsDefault: isDefault,
		IsActive:  true,
		Currency:  "USD",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func createTestCard(userID uuid.UUID, isDefault bool) *PaymentMethod {
	pm := createTestPaymentMethod(userID, PaymentMethodCard, isDefault)
	brand := CardBrandVisa
	last4 := "4242"
	expMonth := 12
	expYear := time.Now().Year() + 3
	holder := "Card Holder"
	providerID := "tok_visa"
	providerType := "stripe"

	pm.CardBrand = &brand
	pm.CardLast4 = &last4
	pm.CardExpMonth = &expMonth
	pm.CardExpYear = &expYear
	pm.CardHolderName = &holder
	pm.ProviderID = &providerID
	pm.ProviderType = &providerType
	return pm
}

func createTestWallet(userID uuid.UUID, balance float64) *PaymentMethod {
	pm := createTestPaymentMethod(userID, PaymentMethodWallet, false)
	pm.WalletBalance = &balance
	return pm
}

func createTestWalletTransaction(userID uuid.UUID, txType string, amount float64) *WalletTransaction {
	return &WalletTransaction{
		ID:              uuid.New(),
		UserID:          userID,
		PaymentMethodID: uuid.New(),
		Type:            txType,
		Amount:          amount,
		BalanceBefore:   100.00,
		BalanceAfter:    100.00 + amount,
		Description:     "Test transaction",
		CreatedAt:       time.Now(),
	}
}

// ============================================================================
// GetPaymentMethods Handler Tests
// ============================================================================

func TestHandler_GetPaymentMethods_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	card := createTestCard(userID, true)
	methods := []PaymentMethod{*card}

	mockRepo.On("GetPaymentMethodsByUser", mock.Anything, userID).Return(methods, nil)
	mockRepo.On("GetWalletBalance", mock.Anything, userID).Return(50.0, nil)

	c, w := setupTestContext("GET", "/api/v1/payment-methods", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetPaymentMethods(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetPaymentMethods_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/payment-methods", nil)
	// Don't set user context to simulate unauthorized

	handler.GetPaymentMethods(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_GetPaymentMethods_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetPaymentMethodsByUser", mock.Anything, userID).Return([]PaymentMethod{}, nil)
	mockRepo.On("GetWalletBalance", mock.Anything, userID).Return(0.0, nil)

	c, w := setupTestContext("GET", "/api/v1/payment-methods", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetPaymentMethods(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	methods := data["methods"].([]interface{})
	assert.Len(t, methods, 0)
}

func TestHandler_GetPaymentMethods_MultiplePaymentMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	card1 := createTestCard(userID, true)
	card2 := createTestCard(userID, false)
	wallet := createTestWallet(userID, 100.00)
	methods := []PaymentMethod{*card1, *card2, *wallet}

	mockRepo.On("GetPaymentMethodsByUser", mock.Anything, userID).Return(methods, nil)
	mockRepo.On("GetWalletBalance", mock.Anything, userID).Return(100.0, nil)

	c, w := setupTestContext("GET", "/api/v1/payment-methods", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetPaymentMethods(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	methods_resp := data["methods"].([]interface{})
	assert.Len(t, methods_resp, 3)
}

func TestHandler_GetPaymentMethods_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetPaymentMethodsByUser", mock.Anything, userID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/payment-methods", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetPaymentMethods(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_GetPaymentMethods_WalletBalanceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	methods := []PaymentMethod{}

	mockRepo.On("GetPaymentMethodsByUser", mock.Anything, userID).Return(methods, nil)
	mockRepo.On("GetWalletBalance", mock.Anything, userID).Return(0.0, errors.New("wallet error"))

	c, w := setupTestContext("GET", "/api/v1/payment-methods", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetPaymentMethods(c)

	// Should still succeed - wallet balance error is not critical
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetPaymentMethods_AsDriver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	methods := []PaymentMethod{}

	mockRepo.On("GetPaymentMethodsByUser", mock.Anything, userID).Return(methods, nil)
	mockRepo.On("GetWalletBalance", mock.Anything, userID).Return(0.0, nil)

	c, w := setupTestContext("GET", "/api/v1/payment-methods", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetPaymentMethods(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================================
// AddCard Handler Tests
// ============================================================================

func TestHandler_AddCard_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	reqBody := AddCardRequest{
		Token:        "tok_visa",
		SetAsDefault: true,
	}

	mockRepo.On("GetPaymentMethodsByUser", mock.Anything, userID).Return([]PaymentMethod{}, nil)
	mockRepo.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/payment-methods/cards", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.AddCard(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_AddCard_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := AddCardRequest{
		Token: "tok_visa",
	}

	c, w := setupTestContext("POST", "/api/v1/payment-methods/cards", reqBody)
	// Don't set user context

	handler.AddCard(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_AddCard_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/payment-methods/cards", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/payment-methods/cards", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleRider)

	handler.AddCard(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AddCard_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"set_as_default": true,
	}

	c, w := setupTestContext("POST", "/api/v1/payment-methods/cards", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.AddCard(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AddCard_WithNickname(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	nickname := "My Business Card"

	reqBody := AddCardRequest{
		Token:    "tok_visa",
		Nickname: &nickname,
	}

	mockRepo.On("GetPaymentMethodsByUser", mock.Anything, userID).Return([]PaymentMethod{}, nil)
	mockRepo.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/payment-methods/cards", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.AddCard(c)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_AddCard_MaxPaymentMethodsExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	// Create 10 existing payment methods
	existingMethods := make([]PaymentMethod, 10)
	for i := 0; i < 10; i++ {
		existingMethods[i] = *createTestCard(userID, false)
	}

	reqBody := AddCardRequest{
		Token: "tok_visa",
	}

	mockRepo.On("GetPaymentMethodsByUser", mock.Anything, userID).Return(existingMethods, nil)

	c, w := setupTestContext("POST", "/api/v1/payment-methods/cards", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.AddCard(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "maximum")
}

func TestHandler_AddCard_FirstCardIsDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	reqBody := AddCardRequest{
		Token:        "tok_visa",
		SetAsDefault: false, // Even if false, first card should be default
	}

	mockRepo.On("GetPaymentMethodsByUser", mock.Anything, userID).Return([]PaymentMethod{}, nil)
	mockRepo.On("CreatePaymentMethod", mock.Anything, mock.MatchedBy(func(pm *PaymentMethod) bool {
		return pm.IsDefault == true // First card should be default
	})).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/payment-methods/cards", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.AddCard(c)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_AddCard_DatabaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	reqBody := AddCardRequest{
		Token: "tok_visa",
	}

	mockRepo.On("GetPaymentMethodsByUser", mock.Anything, userID).Return([]PaymentMethod{}, nil)
	mockRepo.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/payment-methods/cards", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.AddCard(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_AddCard_EmptyToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"token": "",
	}

	c, w := setupTestContext("POST", "/api/v1/payment-methods/cards", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.AddCard(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// AddDigitalWallet Handler Tests
// ============================================================================

func TestHandler_AddDigitalWallet_ApplePay_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	reqBody := AddWalletPaymentRequest{
		Type:  PaymentMethodApplePay,
		Token: "apple_pay_token",
	}

	mockRepo.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/payment-methods/digital-wallet", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.AddDigitalWallet(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_AddDigitalWallet_GooglePay_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	reqBody := AddWalletPaymentRequest{
		Type:  PaymentMethodGooglePay,
		Token: "google_pay_token",
	}

	mockRepo.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/payment-methods/digital-wallet", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.AddDigitalWallet(c)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_AddDigitalWallet_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := AddWalletPaymentRequest{
		Type:  PaymentMethodApplePay,
		Token: "apple_pay_token",
	}

	c, w := setupTestContext("POST", "/api/v1/payment-methods/digital-wallet", reqBody)
	// Don't set user context

	handler.AddDigitalWallet(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_AddDigitalWallet_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/payment-methods/digital-wallet", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/payment-methods/digital-wallet", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleRider)

	handler.AddDigitalWallet(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AddDigitalWallet_MissingType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"token": "some_token",
	}

	c, w := setupTestContext("POST", "/api/v1/payment-methods/digital-wallet", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.AddDigitalWallet(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AddDigitalWallet_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"type": "apple_pay",
	}

	c, w := setupTestContext("POST", "/api/v1/payment-methods/digital-wallet", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.AddDigitalWallet(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AddDigitalWallet_InvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	reqBody := AddWalletPaymentRequest{
		Type:  PaymentMethodCard, // Invalid for digital wallet
		Token: "some_token",
	}

	c, w := setupTestContext("POST", "/api/v1/payment-methods/digital-wallet", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.AddDigitalWallet(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "apple_pay or google_pay")
}

func TestHandler_AddDigitalWallet_WithNickname(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	nickname := "My Apple Pay"

	reqBody := AddWalletPaymentRequest{
		Type:     PaymentMethodApplePay,
		Token:    "apple_pay_token",
		Nickname: &nickname,
	}

	mockRepo.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/payment-methods/digital-wallet", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.AddDigitalWallet(c)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_AddDigitalWallet_DatabaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	reqBody := AddWalletPaymentRequest{
		Type:  PaymentMethodApplePay,
		Token: "apple_pay_token",
	}

	mockRepo.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/payment-methods/digital-wallet", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.AddDigitalWallet(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// EnableCash Handler Tests
// ============================================================================

func TestHandler_EnableCash_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetPaymentMethodsByUser", mock.Anything, userID).Return([]PaymentMethod{}, nil)
	mockRepo.On("CreatePaymentMethod", mock.Anything, mock.MatchedBy(func(pm *PaymentMethod) bool {
		return pm.Type == PaymentMethodCash
	})).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/payment-methods/cash", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.EnableCash(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_EnableCash_AlreadyEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	cashMethod := createTestPaymentMethod(userID, PaymentMethodCash, false)

	mockRepo.On("GetPaymentMethodsByUser", mock.Anything, userID).Return([]PaymentMethod{*cashMethod}, nil)

	c, w := setupTestContext("POST", "/api/v1/payment-methods/cash", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.EnableCash(c)

	// Should return existing cash method without error
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_EnableCash_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("POST", "/api/v1/payment-methods/cash", nil)
	// Don't set user context

	handler.EnableCash(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_EnableCash_DatabaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetPaymentMethodsByUser", mock.Anything, userID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/payment-methods/cash", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.EnableCash(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// SetDefault Handler Tests
// ============================================================================

func TestHandler_SetDefault_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	methodID := uuid.New()

	existingMethod := createTestCard(userID, false)
	existingMethod.ID = methodID

	reqBody := SetDefaultRequest{
		PaymentMethodID: methodID,
	}

	mockRepo.On("GetPaymentMethodByID", mock.Anything, methodID).Return(existingMethod, nil)
	mockRepo.On("SetDefault", mock.Anything, userID, methodID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/payment-methods/default", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.SetDefault(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_SetDefault_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := SetDefaultRequest{
		PaymentMethodID: uuid.New(),
	}

	c, w := setupTestContext("POST", "/api/v1/payment-methods/default", reqBody)
	// Don't set user context

	handler.SetDefault(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_SetDefault_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/payment-methods/default", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/payment-methods/default", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleRider)

	handler.SetDefault(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SetDefault_MissingPaymentMethodID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/payment-methods/default", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.SetDefault(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SetDefault_PaymentMethodNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	methodID := uuid.New()

	reqBody := SetDefaultRequest{
		PaymentMethodID: methodID,
	}

	mockRepo.On("GetPaymentMethodByID", mock.Anything, methodID).Return(nil, pgx.ErrNoRows)

	c, w := setupTestContext("POST", "/api/v1/payment-methods/default", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.SetDefault(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_SetDefault_NotYourPaymentMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	otherUserID := uuid.New()
	methodID := uuid.New()

	// Payment method belongs to another user
	existingMethod := createTestCard(otherUserID, false)
	existingMethod.ID = methodID

	reqBody := SetDefaultRequest{
		PaymentMethodID: methodID,
	}

	mockRepo.On("GetPaymentMethodByID", mock.Anything, methodID).Return(existingMethod, nil)

	c, w := setupTestContext("POST", "/api/v1/payment-methods/default", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.SetDefault(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_SetDefault_DatabaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	methodID := uuid.New()

	existingMethod := createTestCard(userID, false)
	existingMethod.ID = methodID

	reqBody := SetDefaultRequest{
		PaymentMethodID: methodID,
	}

	mockRepo.On("GetPaymentMethodByID", mock.Anything, methodID).Return(existingMethod, nil)
	mockRepo.On("SetDefault", mock.Anything, userID, methodID).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/payment-methods/default", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.SetDefault(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// RemovePaymentMethod Handler Tests
// ============================================================================

func TestHandler_RemovePaymentMethod_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	methodID := uuid.New()

	existingMethod := createTestCard(userID, false)
	existingMethod.ID = methodID

	mockRepo.On("GetPaymentMethodByID", mock.Anything, methodID).Return(existingMethod, nil)
	mockRepo.On("DeactivatePaymentMethod", mock.Anything, methodID, userID).Return(nil)

	c, w := setupTestContext("DELETE", "/api/v1/payment-methods/"+methodID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: methodID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.RemovePaymentMethod(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_RemovePaymentMethod_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	methodID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/payment-methods/"+methodID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: methodID.String()}}
	// Don't set user context

	handler.RemovePaymentMethod(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_RemovePaymentMethod_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/payment-methods/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID, models.RoleRider)

	handler.RemovePaymentMethod(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid payment method id")
}

func TestHandler_RemovePaymentMethod_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	methodID := uuid.New()

	mockRepo.On("GetPaymentMethodByID", mock.Anything, methodID).Return(nil, pgx.ErrNoRows)

	c, w := setupTestContext("DELETE", "/api/v1/payment-methods/"+methodID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: methodID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.RemovePaymentMethod(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_RemovePaymentMethod_NotYourPaymentMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	otherUserID := uuid.New()
	methodID := uuid.New()

	existingMethod := createTestCard(otherUserID, false)
	existingMethod.ID = methodID

	mockRepo.On("GetPaymentMethodByID", mock.Anything, methodID).Return(existingMethod, nil)

	c, w := setupTestContext("DELETE", "/api/v1/payment-methods/"+methodID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: methodID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.RemovePaymentMethod(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_RemovePaymentMethod_CannotRemoveWallet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	walletID := uuid.New()

	walletMethod := createTestWallet(userID, 100.0)
	walletMethod.ID = walletID

	mockRepo.On("GetPaymentMethodByID", mock.Anything, walletID).Return(walletMethod, nil)

	c, w := setupTestContext("DELETE", "/api/v1/payment-methods/"+walletID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: walletID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.RemovePaymentMethod(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "wallet cannot be removed")
}

func TestHandler_RemovePaymentMethod_DatabaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	methodID := uuid.New()

	existingMethod := createTestCard(userID, false)
	existingMethod.ID = methodID

	mockRepo.On("GetPaymentMethodByID", mock.Anything, methodID).Return(existingMethod, nil)
	mockRepo.On("DeactivatePaymentMethod", mock.Anything, methodID, userID).Return(errors.New("database error"))

	c, w := setupTestContext("DELETE", "/api/v1/payment-methods/"+methodID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: methodID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.RemovePaymentMethod(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_RemovePaymentMethod_EmptyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/payment-methods/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	setUserContext(c, userID, models.RoleRider)

	handler.RemovePaymentMethod(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// TopUpWallet Handler Tests
// ============================================================================

func TestHandler_TopUpWallet_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	cardID := uuid.New()

	card := createTestCard(userID, true)
	card.ID = cardID

	wallet := createTestWallet(userID, 50.00)

	reqBody := TopUpWalletRequest{
		Amount:         100.00,
		SourceMethodID: cardID,
	}

	mockRepo.On("GetPaymentMethodByID", mock.Anything, cardID).Return(card, nil)
	mockRepo.On("EnsureWalletExists", mock.Anything, userID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", mock.Anything, wallet.ID, 100.00).Return(150.00, nil)
	mockRepo.On("CreateWalletTransaction", mock.Anything, mock.AnythingOfType("*paymentmethods.WalletTransaction")).Return(nil)
	mockRepo.On("GetWalletBalance", mock.Anything, userID).Return(150.0, nil)
	mockRepo.On("GetWalletTransactions", mock.Anything, userID, 20).Return([]WalletTransaction{}, nil)

	c, w := setupTestContext("POST", "/api/v1/payment-methods/wallet/topup", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_TopUpWallet_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := TopUpWalletRequest{
		Amount:         100.00,
		SourceMethodID: uuid.New(),
	}

	c, w := setupTestContext("POST", "/api/v1/payment-methods/wallet/topup", reqBody)
	// Don't set user context

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_TopUpWallet_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/payment-methods/wallet/topup", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/payment-methods/wallet/topup", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TopUpWallet_MissingAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"source_method_id": uuid.New().String(),
	}

	c, w := setupTestContext("POST", "/api/v1/payment-methods/wallet/topup", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TopUpWallet_MissingSourceMethodID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"amount": 100.00,
	}

	c, w := setupTestContext("POST", "/api/v1/payment-methods/wallet/topup", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TopUpWallet_AmountTooLow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	cardID := uuid.New()

	card := createTestCard(userID, true)
	card.ID = cardID

	reqBody := TopUpWalletRequest{
		Amount:         1.00, // Below minimum (5.0)
		SourceMethodID: cardID,
	}

	mockRepo.On("GetPaymentMethodByID", mock.Anything, cardID).Return(card, nil)

	c, w := setupTestContext("POST", "/api/v1/payment-methods/wallet/topup", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "between")
}

func TestHandler_TopUpWallet_AmountTooHigh(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	cardID := uuid.New()

	card := createTestCard(userID, true)
	card.ID = cardID

	reqBody := TopUpWalletRequest{
		Amount:         1000.00, // Above maximum (500.0)
		SourceMethodID: cardID,
	}

	mockRepo.On("GetPaymentMethodByID", mock.Anything, cardID).Return(card, nil)

	c, w := setupTestContext("POST", "/api/v1/payment-methods/wallet/topup", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TopUpWallet_SourceMethodNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	cardID := uuid.New()

	reqBody := TopUpWalletRequest{
		Amount:         100.00,
		SourceMethodID: cardID,
	}

	mockRepo.On("GetPaymentMethodByID", mock.Anything, cardID).Return(nil, pgx.ErrNoRows)

	c, w := setupTestContext("POST", "/api/v1/payment-methods/wallet/topup", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_TopUpWallet_NotYourSourceMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	otherUserID := uuid.New()
	cardID := uuid.New()

	card := createTestCard(otherUserID, true)
	card.ID = cardID

	reqBody := TopUpWalletRequest{
		Amount:         100.00,
		SourceMethodID: cardID,
	}

	mockRepo.On("GetPaymentMethodByID", mock.Anything, cardID).Return(card, nil)

	c, w := setupTestContext("POST", "/api/v1/payment-methods/wallet/topup", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_TopUpWallet_SourceMustBeCard(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	methodID := uuid.New()

	cashMethod := createTestPaymentMethod(userID, PaymentMethodCash, false)
	cashMethod.ID = methodID

	reqBody := TopUpWalletRequest{
		Amount:         100.00,
		SourceMethodID: methodID,
	}

	mockRepo.On("GetPaymentMethodByID", mock.Anything, methodID).Return(cashMethod, nil)

	c, w := setupTestContext("POST", "/api/v1/payment-methods/wallet/topup", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "only top up from a card")
}

func TestHandler_TopUpWallet_EnsureWalletError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	cardID := uuid.New()

	card := createTestCard(userID, true)
	card.ID = cardID

	reqBody := TopUpWalletRequest{
		Amount:         100.00,
		SourceMethodID: cardID,
	}

	mockRepo.On("GetPaymentMethodByID", mock.Anything, cardID).Return(card, nil)
	mockRepo.On("EnsureWalletExists", mock.Anything, userID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/payment-methods/wallet/topup", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetWalletSummary Handler Tests
// ============================================================================

func TestHandler_GetWalletSummary_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	transactions := []WalletTransaction{
		*createTestWalletTransaction(userID, "topup", 50.00),
		*createTestWalletTransaction(userID, "debit", -25.00),
	}

	mockRepo.On("GetWalletBalance", mock.Anything, userID).Return(125.0, nil)
	mockRepo.On("GetWalletTransactions", mock.Anything, userID, 20).Return(transactions, nil)

	c, w := setupTestContext("GET", "/api/v1/payment-methods/wallet", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetWalletSummary(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, 125.0, data["balance"])
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetWalletSummary_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/payment-methods/wallet", nil)
	// Don't set user context

	handler.GetWalletSummary(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetWalletSummary_ZeroBalance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetWalletBalance", mock.Anything, userID).Return(0.0, nil)
	mockRepo.On("GetWalletTransactions", mock.Anything, userID, 20).Return([]WalletTransaction{}, nil)

	c, w := setupTestContext("GET", "/api/v1/payment-methods/wallet", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetWalletSummary(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, 0.0, data["balance"])
}

func TestHandler_GetWalletSummary_NoTransactions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetWalletBalance", mock.Anything, userID).Return(100.0, nil)
	mockRepo.On("GetWalletTransactions", mock.Anything, userID, 20).Return([]WalletTransaction{}, nil)

	c, w := setupTestContext("GET", "/api/v1/payment-methods/wallet", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetWalletSummary(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	transactions := data["recent_transactions"].([]interface{})
	assert.Len(t, transactions, 0)
}

func TestHandler_GetWalletSummary_BalanceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetWalletBalance", mock.Anything, userID).Return(0.0, errors.New("database error"))
	mockRepo.On("GetWalletTransactions", mock.Anything, userID, 20).Return([]WalletTransaction{}, nil)

	c, w := setupTestContext("GET", "/api/v1/payment-methods/wallet", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetWalletSummary(c)

	// Should still succeed - errors are logged but not returned
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetWalletSummary_TransactionsError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetWalletBalance", mock.Anything, userID).Return(100.0, nil)
	mockRepo.On("GetWalletTransactions", mock.Anything, userID, 20).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/payment-methods/wallet", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetWalletSummary(c)

	// Should still succeed - errors are logged but not returned
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetWalletSummary_AsDriver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetWalletBalance", mock.Anything, userID).Return(500.0, nil)
	mockRepo.On("GetWalletTransactions", mock.Anything, userID, 20).Return([]WalletTransaction{}, nil)

	c, w := setupTestContext("GET", "/api/v1/payment-methods/wallet", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetWalletSummary(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_AddCard_ValidationCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           interface{}
		setupMock      func(*MockRepository, uuid.UUID)
		expectedStatus int
	}{
		{
			name: "valid card with token",
			body: map[string]interface{}{
				"token": "tok_visa",
			},
			setupMock: func(m *MockRepository, userID uuid.UUID) {
				m.On("GetPaymentMethodsByUser", mock.Anything, userID).Return([]PaymentMethod{}, nil)
				m.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "valid card with all fields",
			body: map[string]interface{}{
				"token":          "tok_visa",
				"nickname":       "My Visa",
				"set_as_default": true,
			},
			setupMock: func(m *MockRepository, userID uuid.UUID) {
				m.On("GetPaymentMethodsByUser", mock.Anything, userID).Return([]PaymentMethod{}, nil)
				m.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "missing token",
			body:           map[string]interface{}{},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty token",
			body: map[string]interface{}{
				"token": "",
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			userID := uuid.New()
			if tt.setupMock != nil {
				tt.setupMock(mockRepo, userID)
			}

			c, w := setupTestContext("POST", "/api/v1/payment-methods/cards", tt.body)
			setUserContext(c, userID, models.RoleRider)

			handler.AddCard(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_SetDefault_ErrorCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupMock      func(*MockRepository, uuid.UUID, uuid.UUID)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "payment method not found",
			setupMock: func(m *MockRepository, userID, methodID uuid.UUID) {
				m.On("GetPaymentMethodByID", mock.Anything, methodID).Return(nil, pgx.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "not found",
		},
		{
			name: "payment method belongs to another user",
			setupMock: func(m *MockRepository, userID, methodID uuid.UUID) {
				otherUserID := uuid.New()
				pm := createTestCard(otherUserID, false)
				pm.ID = methodID
				m.On("GetPaymentMethodByID", mock.Anything, methodID).Return(pm, nil)
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "not your",
		},
		{
			name: "database error on set default",
			setupMock: func(m *MockRepository, userID, methodID uuid.UUID) {
				pm := createTestCard(userID, false)
				pm.ID = methodID
				m.On("GetPaymentMethodByID", mock.Anything, methodID).Return(pm, nil)
				m.On("SetDefault", mock.Anything, userID, methodID).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			userID := uuid.New()
			methodID := uuid.New()

			if tt.setupMock != nil {
				tt.setupMock(mockRepo, userID, methodID)
			}

			reqBody := SetDefaultRequest{
				PaymentMethodID: methodID,
			}

			c, w := setupTestContext("POST", "/api/v1/payment-methods/default", reqBody)
			setUserContext(c, userID, models.RoleRider)

			handler.SetDefault(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				response := parseResponse(w)
				assert.Contains(t, response["error"].(map[string]interface{})["message"], tt.expectedError)
			}
		})
	}
}

func TestHandler_RemovePaymentMethod_Cases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		methodID       string
		setupMock      func(*MockRepository, uuid.UUID, uuid.UUID)
		expectedStatus int
		expectedError  string
	}{
		{
			name:     "successful removal",
			methodID: uuid.New().String(),
			setupMock: func(m *MockRepository, userID uuid.UUID, methodID uuid.UUID) {
				pm := createTestCard(userID, false)
				pm.ID = methodID
				m.On("GetPaymentMethodByID", mock.Anything, methodID).Return(pm, nil)
				m.On("DeactivatePaymentMethod", mock.Anything, methodID, userID).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid UUID",
			methodID:       "not-a-uuid",
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid",
		},
		{
			name:     "cannot remove wallet",
			methodID: uuid.New().String(),
			setupMock: func(m *MockRepository, userID uuid.UUID, methodID uuid.UUID) {
				wallet := createTestWallet(userID, 100.0)
				wallet.ID = methodID
				m.On("GetPaymentMethodByID", mock.Anything, methodID).Return(wallet, nil)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "wallet cannot be removed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			userID := uuid.New()
			var methodID uuid.UUID
			if uid, err := uuid.Parse(tt.methodID); err == nil {
				methodID = uid
			}

			if tt.setupMock != nil {
				tt.setupMock(mockRepo, userID, methodID)
			}

			c, w := setupTestContext("DELETE", "/api/v1/payment-methods/"+tt.methodID, nil)
			c.Params = gin.Params{{Key: "id", Value: tt.methodID}}
			setUserContext(c, userID, models.RoleRider)

			handler.RemovePaymentMethod(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				response := parseResponse(w)
				assert.Contains(t, response["error"].(map[string]interface{})["message"], tt.expectedError)
			}
		})
	}
}

func TestHandler_TopUpWallet_ValidationCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           interface{}
		setupMock      func(*MockRepository, uuid.UUID, uuid.UUID)
		expectedStatus int
	}{
		{
			name: "amount too low",
			body: map[string]interface{}{
				"amount":           1.0,
				"source_method_id": uuid.New().String(),
			},
			setupMock: func(m *MockRepository, userID, cardID uuid.UUID) {
				card := createTestCard(userID, true)
				card.ID = cardID
				m.On("GetPaymentMethodByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(card, nil)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "amount too high",
			body: map[string]interface{}{
				"amount":           1000.0,
				"source_method_id": uuid.New().String(),
			},
			setupMock: func(m *MockRepository, userID, cardID uuid.UUID) {
				card := createTestCard(userID, true)
				card.ID = cardID
				m.On("GetPaymentMethodByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(card, nil)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "valid amount at minimum",
			body: map[string]interface{}{
				"amount":           5.0,
				"source_method_id": uuid.New().String(),
			},
			setupMock: func(m *MockRepository, userID, cardID uuid.UUID) {
				card := createTestCard(userID, true)
				card.ID = cardID
				wallet := createTestWallet(userID, 0)
				m.On("GetPaymentMethodByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(card, nil)
				m.On("EnsureWalletExists", mock.Anything, userID).Return(wallet, nil)
				m.On("UpdateWalletBalance", mock.Anything, wallet.ID, 5.0).Return(5.0, nil)
				m.On("CreateWalletTransaction", mock.Anything, mock.AnythingOfType("*paymentmethods.WalletTransaction")).Return(nil)
				m.On("GetWalletBalance", mock.Anything, userID).Return(5.0, nil)
				m.On("GetWalletTransactions", mock.Anything, userID, 20).Return([]WalletTransaction{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid amount at maximum",
			body: map[string]interface{}{
				"amount":           500.0,
				"source_method_id": uuid.New().String(),
			},
			setupMock: func(m *MockRepository, userID, cardID uuid.UUID) {
				card := createTestCard(userID, true)
				card.ID = cardID
				wallet := createTestWallet(userID, 0)
				m.On("GetPaymentMethodByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(card, nil)
				m.On("EnsureWalletExists", mock.Anything, userID).Return(wallet, nil)
				m.On("UpdateWalletBalance", mock.Anything, wallet.ID, 500.0).Return(500.0, nil)
				m.On("CreateWalletTransaction", mock.Anything, mock.AnythingOfType("*paymentmethods.WalletTransaction")).Return(nil)
				m.On("GetWalletBalance", mock.Anything, userID).Return(500.0, nil)
				m.On("GetWalletTransactions", mock.Anything, userID, 20).Return([]WalletTransaction{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			userID := uuid.New()
			cardID := uuid.New()

			if tt.setupMock != nil {
				tt.setupMock(mockRepo, userID, cardID)
			}

			c, w := setupTestContext("POST", "/api/v1/payment-methods/wallet/topup", tt.body)
			setUserContext(c, userID, models.RoleRider)

			handler.TopUpWallet(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// Edge Cases and Additional Tests
// ============================================================================

func TestHandler_AddDigitalWallet_AllTypes(t *testing.T) {
	tests := []struct {
		name        string
		walletType  PaymentMethodType
		expectError bool
	}{
		{"apple pay", PaymentMethodApplePay, false},
		{"google pay", PaymentMethodGooglePay, false},
		{"card type", PaymentMethodCard, true},
		{"wallet type", PaymentMethodWallet, true},
		{"cash type", PaymentMethodCash, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			userID := uuid.New()

			reqBody := AddWalletPaymentRequest{
				Type:  tt.walletType,
				Token: "some_token",
			}

			if !tt.expectError {
				mockRepo.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(nil)
			}

			c, w := setupTestContext("POST", "/api/v1/payment-methods/digital-wallet", reqBody)
			setUserContext(c, userID, models.RoleRider)

			handler.AddDigitalWallet(c)

			if tt.expectError {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			} else {
				assert.Equal(t, http.StatusCreated, w.Code)
			}
		})
	}
}

func TestHandler_ResponseFormats(t *testing.T) {
	t.Run("GetPaymentMethods response format", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		mockRepo := new(MockRepository)
		handler := createTestHandler(mockRepo)

		userID := uuid.New()
		card := createTestCard(userID, true)

		mockRepo.On("GetPaymentMethodsByUser", mock.Anything, userID).Return([]PaymentMethod{*card}, nil)
		mockRepo.On("GetWalletBalance", mock.Anything, userID).Return(50.0, nil)

		c, w := setupTestContext("GET", "/api/v1/payment-methods", nil)
		setUserContext(c, userID, models.RoleRider)

		handler.GetPaymentMethods(c)

		assert.Equal(t, http.StatusOK, w.Code)
		response := parseResponse(w)
		require.True(t, response["success"].(bool))
		require.NotNil(t, response["data"])

		data := response["data"].(map[string]interface{})
		require.NotNil(t, data["methods"])
		require.NotNil(t, data["currency"])
		assert.Equal(t, "USD", data["currency"])
	})

	t.Run("AddCard response format", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		mockRepo := new(MockRepository)
		handler := createTestHandler(mockRepo)

		userID := uuid.New()

		mockRepo.On("GetPaymentMethodsByUser", mock.Anything, userID).Return([]PaymentMethod{}, nil)
		mockRepo.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(nil)

		reqBody := AddCardRequest{Token: "tok_visa"}

		c, w := setupTestContext("POST", "/api/v1/payment-methods/cards", reqBody)
		setUserContext(c, userID, models.RoleRider)

		handler.AddCard(c)

		assert.Equal(t, http.StatusCreated, w.Code)
		response := parseResponse(w)
		require.True(t, response["success"].(bool))
		require.NotNil(t, response["data"])

		data := response["data"].(map[string]interface{})
		require.NotNil(t, data["id"])
		require.NotNil(t, data["type"])
		assert.Equal(t, string(PaymentMethodCard), data["type"])
	})

	t.Run("WalletSummary response format", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		mockRepo := new(MockRepository)
		handler := createTestHandler(mockRepo)

		userID := uuid.New()

		mockRepo.On("GetWalletBalance", mock.Anything, userID).Return(100.0, nil)
		mockRepo.On("GetWalletTransactions", mock.Anything, userID, 20).Return([]WalletTransaction{}, nil)

		c, w := setupTestContext("GET", "/api/v1/payment-methods/wallet", nil)
		setUserContext(c, userID, models.RoleRider)

		handler.GetWalletSummary(c)

		assert.Equal(t, http.StatusOK, w.Code)
		response := parseResponse(w)
		require.True(t, response["success"].(bool))
		require.NotNil(t, response["data"])

		data := response["data"].(map[string]interface{})
		require.NotNil(t, data["balance"])
		require.NotNil(t, data["currency"])
		require.NotNil(t, data["recent_transactions"])
	})
}

func TestHandler_ErrorResponseFormats(t *testing.T) {
	t.Run("Unauthorized error format", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		mockRepo := new(MockRepository)
		handler := createTestHandler(mockRepo)

		c, w := setupTestContext("GET", "/api/v1/payment-methods", nil)
		// Don't set user context

		handler.GetPaymentMethods(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		response := parseResponse(w)
		require.False(t, response["success"].(bool))
		require.NotNil(t, response["error"])

		errorInfo := response["error"].(map[string]interface{})
		require.NotNil(t, errorInfo["message"])
	})

	t.Run("BadRequest error format", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		mockRepo := new(MockRepository)
		handler := createTestHandler(mockRepo)

		userID := uuid.New()

		c, w := setupTestContext("DELETE", "/api/v1/payment-methods/invalid-uuid", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
		setUserContext(c, userID, models.RoleRider)

		handler.RemovePaymentMethod(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		response := parseResponse(w)
		require.False(t, response["success"].(bool))
		require.NotNil(t, response["error"])
	})
}

func TestHandler_ConcurrentAccessScenarios(t *testing.T) {
	t.Run("Concurrent set default", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		mockRepo := new(MockRepository)
		handler := createTestHandler(mockRepo)

		userID := uuid.New()
		methodID := uuid.New()

		pm := createTestCard(userID, false)
		pm.ID = methodID

		// Simulate concurrent modification error
		mockRepo.On("GetPaymentMethodByID", mock.Anything, methodID).Return(pm, nil)
		mockRepo.On("SetDefault", mock.Anything, userID, methodID).Return(common.NewConflictError("concurrent modification"))

		reqBody := SetDefaultRequest{PaymentMethodID: methodID}

		c, w := setupTestContext("POST", "/api/v1/payment-methods/default", reqBody)
		setUserContext(c, userID, models.RoleRider)

		handler.SetDefault(c)

		// Should return error, not infinite loop
		assert.True(t, w.Code >= 400)
	})
}

func TestHandler_NilAndEmptyValues(t *testing.T) {
	t.Run("Nil methods list handling", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		mockRepo := new(MockRepository)
		handler := createTestHandler(mockRepo)

		userID := uuid.New()

		// Return nil instead of empty slice
		var nilMethods []PaymentMethod = nil
		mockRepo.On("GetPaymentMethodsByUser", mock.Anything, userID).Return(nilMethods, nil)
		mockRepo.On("GetWalletBalance", mock.Anything, userID).Return(0.0, nil)

		c, w := setupTestContext("GET", "/api/v1/payment-methods", nil)
		setUserContext(c, userID, models.RoleRider)

		handler.GetPaymentMethods(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Nil wallet transactions handling", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		mockRepo := new(MockRepository)
		handler := createTestHandler(mockRepo)

		userID := uuid.New()

		mockRepo.On("GetWalletBalance", mock.Anything, userID).Return(100.0, nil)
		var nilTransactions []WalletTransaction = nil
		mockRepo.On("GetWalletTransactions", mock.Anything, userID, 20).Return(nilTransactions, nil)

		c, w := setupTestContext("GET", "/api/v1/payment-methods/wallet", nil)
		setUserContext(c, userID, models.RoleRider)

		handler.GetWalletSummary(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandler_DatabaseTimeoutScenarios(t *testing.T) {
	t.Run("GetPaymentMethods database timeout", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		mockRepo := new(MockRepository)
		handler := createTestHandler(mockRepo)

		userID := uuid.New()

		mockRepo.On("GetPaymentMethodsByUser", mock.Anything, userID).Return(nil, context.DeadlineExceeded)

		c, w := setupTestContext("GET", "/api/v1/payment-methods", nil)
		setUserContext(c, userID, models.RoleRider)

		handler.GetPaymentMethods(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ============================================================================
// Integration-like Tests (testing handler + service interaction)
// ============================================================================

func TestHandler_AddCardAndSetDefault_Flow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	// Step 1: Add first card (becomes default automatically)
	mockRepo.On("GetPaymentMethodsByUser", mock.Anything, userID).Return([]PaymentMethod{}, nil).Once()
	mockRepo.On("CreatePaymentMethod", mock.Anything, mock.MatchedBy(func(pm *PaymentMethod) bool {
		return pm.IsDefault == true && pm.Type == PaymentMethodCard
	})).Return(nil).Once()

	reqBody1 := AddCardRequest{Token: "tok_visa_1"}
	c1, w1 := setupTestContext("POST", "/api/v1/payment-methods/cards", reqBody1)
	setUserContext(c1, userID, models.RoleRider)

	handler.AddCard(c1)
	assert.Equal(t, http.StatusCreated, w1.Code)

	// Step 2: Add second card (not default)
	card1 := createTestCard(userID, true)
	mockRepo.On("GetPaymentMethodsByUser", mock.Anything, userID).Return([]PaymentMethod{*card1}, nil).Once()
	mockRepo.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(nil).Once()

	reqBody2 := AddCardRequest{Token: "tok_visa_2", SetAsDefault: false}
	c2, w2 := setupTestContext("POST", "/api/v1/payment-methods/cards", reqBody2)
	setUserContext(c2, userID, models.RoleRider)

	handler.AddCard(c2)
	assert.Equal(t, http.StatusCreated, w2.Code)

	mockRepo.AssertExpectations(t)
}

func TestHandler_FullWalletTopUpFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	cardID := uuid.New()
	walletID := uuid.New()

	card := createTestCard(userID, true)
	card.ID = cardID

	wallet := createTestWallet(userID, 0)
	wallet.ID = walletID

	// Set up full flow expectations
	mockRepo.On("GetPaymentMethodByID", mock.Anything, cardID).Return(card, nil)
	mockRepo.On("EnsureWalletExists", mock.Anything, userID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", mock.Anything, walletID, 100.0).Return(100.0, nil)
	mockRepo.On("CreateWalletTransaction", mock.Anything, mock.MatchedBy(func(tx *WalletTransaction) bool {
		return tx.Type == "topup" && tx.Amount == 100.0
	})).Return(nil)
	mockRepo.On("GetWalletBalance", mock.Anything, userID).Return(100.0, nil)
	mockRepo.On("GetWalletTransactions", mock.Anything, userID, 20).Return([]WalletTransaction{
		*createTestWalletTransaction(userID, "topup", 100.0),
	}, nil)

	reqBody := TopUpWalletRequest{
		Amount:         100.0,
		SourceMethodID: cardID,
	}

	c, w := setupTestContext("POST", "/api/v1/payment-methods/wallet/topup", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, 100.0, data["balance"])

	mockRepo.AssertExpectations(t)
}
