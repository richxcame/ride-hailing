package payments

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
	"github.com/stripe/stripe-go/v83"
)

// MockRepository is a mock implementation of RepositoryInterface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreatePayment(ctx context.Context, payment *models.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockRepository) GetPaymentByID(ctx context.Context, id uuid.UUID) (*models.Payment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Payment), args.Error(1)
}

func (m *MockRepository) UpdatePaymentStatus(ctx context.Context, id uuid.UUID, status string, stripeChargeID *string) error {
	args := m.Called(ctx, id, status, stripeChargeID)
	return args.Error(0)
}

func (m *MockRepository) ProcessPaymentWithWallet(ctx context.Context, payment *models.Payment, transaction *models.WalletTransaction) error {
	args := m.Called(ctx, payment, transaction)
	return args.Error(0)
}

func (m *MockRepository) GetWalletByUserID(ctx context.Context, userID uuid.UUID) (*models.Wallet, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Wallet), args.Error(1)
}

func (m *MockRepository) CreateWallet(ctx context.Context, wallet *models.Wallet) error {
	args := m.Called(ctx, wallet)
	return args.Error(0)
}

func (m *MockRepository) UpdateWalletBalance(ctx context.Context, walletID uuid.UUID, amount float64) error {
	args := m.Called(ctx, walletID, amount)
	return args.Error(0)
}

func (m *MockRepository) CreateWalletTransaction(ctx context.Context, transaction *models.WalletTransaction) error {
	args := m.Called(ctx, transaction)
	return args.Error(0)
}

func (m *MockRepository) GetWalletTransactions(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*models.WalletTransaction, error) {
	args := m.Called(ctx, walletID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.WalletTransaction), args.Error(1)
}

func (m *MockRepository) GetWalletTransactionsWithTotal(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*models.WalletTransaction, int64, error) {
	args := m.Called(ctx, walletID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*models.WalletTransaction), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) GetRideDriverID(ctx context.Context, rideID uuid.UUID) (*uuid.UUID, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*uuid.UUID), args.Error(1)
}

func (m *MockRepository) GetPaymentsByRideID(ctx context.Context, rideID uuid.UUID) ([]*models.Payment, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Payment), args.Error(1)
}

func (m *MockRepository) GetAllPayments(ctx context.Context, limit, offset int, filter *AdminPaymentFilter) ([]*models.Payment, int64, error) {
	args := m.Called(ctx, limit, offset, filter)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*models.Payment), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) GetPaymentStats(ctx context.Context) (*PaymentStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PaymentStats), args.Error(1)
}

// MockStripeClient is a mock implementation of StripeClientInterface
type MockStripeClient struct {
	mock.Mock
}

func (m *MockStripeClient) CreateCustomer(email, name string, metadata map[string]string) (*stripe.Customer, error) {
	args := m.Called(email, name, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stripe.Customer), args.Error(1)
}

func (m *MockStripeClient) CreatePaymentIntent(amount int64, currency, customerID, description string, metadata map[string]string) (*stripe.PaymentIntent, error) {
	args := m.Called(amount, currency, customerID, description, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stripe.PaymentIntent), args.Error(1)
}

func (m *MockStripeClient) ConfirmPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	args := m.Called(paymentIntentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stripe.PaymentIntent), args.Error(1)
}

func (m *MockStripeClient) CapturePaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	args := m.Called(paymentIntentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stripe.PaymentIntent), args.Error(1)
}

func (m *MockStripeClient) CreateRefund(chargeID string, amount *int64, reason string) (*stripe.Refund, error) {
	args := m.Called(chargeID, amount, reason)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stripe.Refund), args.Error(1)
}

func (m *MockStripeClient) CreateTransfer(amount int64, currency, destination, description string, metadata map[string]string) (*stripe.Transfer, error) {
	args := m.Called(amount, currency, destination, description, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stripe.Transfer), args.Error(1)
}

func (m *MockStripeClient) GetPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	args := m.Called(paymentIntentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stripe.PaymentIntent), args.Error(1)
}

func (m *MockStripeClient) CancelPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	args := m.Called(paymentIntentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stripe.PaymentIntent), args.Error(1)
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

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

func createTestPayment(riderID, driverID, rideID uuid.UUID, status string) *models.Payment {
	now := time.Now()
	return &models.Payment{
		ID:            uuid.New(),
		RideID:        rideID,
		RiderID:       riderID,
		DriverID:      driverID,
		Amount:        25.00,
		Currency:      "usd",
		PaymentMethod: "wallet",
		Status:        status,
		Metadata:      map[string]interface{}{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func createTestWallet(userID uuid.UUID, balance float64) *models.Wallet {
	now := time.Now()
	return &models.Wallet{
		ID:        uuid.New(),
		UserID:    userID,
		Balance:   balance,
		Currency:  "usd",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func createTestWalletTransaction(walletID uuid.UUID, txType string, amount float64) *models.WalletTransaction {
	now := time.Now()
	return &models.WalletTransaction{
		ID:            uuid.New(),
		WalletID:      walletID,
		Type:          txType,
		Amount:        amount,
		Description:   "Test transaction",
		ReferenceType: "ride",
		BalanceBefore: 100.00,
		BalanceAfter:  100.00 - amount,
		CreatedAt:     now,
	}
}

func createTestHandler(mockRepo *MockRepository, mockStripe *MockStripeClient) *Handler {
	service := NewService(mockRepo, mockStripe, nil)
	return NewHandler(service)
}

// ============================================================================
// ProcessPayment Handler Tests
// ============================================================================

func TestHandler_ProcessPayment_Success_Wallet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := ProcessPaymentRequest{
		RideID:        rideID.String(),
		Amount:        25.00,
		PaymentMethod: "wallet",
	}

	mockRepo.On("GetRideDriverID", mock.Anything, rideID).Return(&driverID, nil)
	mockRepo.On("ProcessPaymentWithWallet", mock.Anything, mock.AnythingOfType("*models.Payment"), mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/payments/process", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.ProcessPayment(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_ProcessPayment_Success_Stripe(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := ProcessPaymentRequest{
		RideID:        rideID.String(),
		Amount:        25.00,
		PaymentMethod: "stripe",
	}

	mockRepo.On("GetRideDriverID", mock.Anything, rideID).Return(&driverID, nil)
	mockStripe.On("CreatePaymentIntent", int64(2500), "usd", "", mock.Anything, mock.Anything).Return(&stripe.PaymentIntent{
		ID:     "pi_test_123",
		Status: stripe.PaymentIntentStatusRequiresPaymentMethod,
	}, nil)
	mockRepo.On("CreatePayment", mock.Anything, mock.AnythingOfType("*models.Payment")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/payments/process", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.ProcessPayment(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
	mockStripe.AssertExpectations(t)
}

func TestHandler_ProcessPayment_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	reqBody := ProcessPaymentRequest{
		RideID:        uuid.New().String(),
		Amount:        25.00,
		PaymentMethod: "wallet",
	}

	c, w := setupTestContext("POST", "/api/v1/payments/process", reqBody)
	// Don't set user context to simulate unauthorized

	handler.ProcessPayment(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_ProcessPayment_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/payments/process", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/payments/process", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleRider)

	handler.ProcessPayment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ProcessPayment_MissingRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"amount":         25.00,
		"payment_method": "wallet",
	}

	c, w := setupTestContext("POST", "/api/v1/payments/process", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.ProcessPayment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ProcessPayment_MissingAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"ride_id":        uuid.New().String(),
		"payment_method": "wallet",
	}

	c, w := setupTestContext("POST", "/api/v1/payments/process", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.ProcessPayment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ProcessPayment_MissingPaymentMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"ride_id": uuid.New().String(),
		"amount":  25.00,
	}

	c, w := setupTestContext("POST", "/api/v1/payments/process", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.ProcessPayment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ProcessPayment_InvalidRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()

	reqBody := ProcessPaymentRequest{
		RideID:        "invalid-uuid",
		Amount:        25.00,
		PaymentMethod: "wallet",
	}

	c, w := setupTestContext("POST", "/api/v1/payments/process", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.ProcessPayment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid ride ID")
}

func TestHandler_ProcessPayment_InvalidPaymentMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"ride_id":        uuid.New().String(),
		"amount":         25.00,
		"payment_method": "bitcoin",
	}

	c, w := setupTestContext("POST", "/api/v1/payments/process", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.ProcessPayment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ProcessPayment_ZeroAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"ride_id":        uuid.New().String(),
		"amount":         0,
		"payment_method": "wallet",
	}

	c, w := setupTestContext("POST", "/api/v1/payments/process", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.ProcessPayment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ProcessPayment_NegativeAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"ride_id":        uuid.New().String(),
		"amount":         -25.00,
		"payment_method": "wallet",
	}

	c, w := setupTestContext("POST", "/api/v1/payments/process", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.ProcessPayment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ProcessPayment_RideNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	rideID := uuid.New()

	reqBody := ProcessPaymentRequest{
		RideID:        rideID.String(),
		Amount:        25.00,
		PaymentMethod: "wallet",
	}

	mockRepo.On("GetRideDriverID", mock.Anything, rideID).Return(nil, common.NewNotFoundError("ride not found", nil))

	c, w := setupTestContext("POST", "/api/v1/payments/process", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.ProcessPayment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "ride not found")
}

func TestHandler_ProcessPayment_RideNoDriverAssigned(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	rideID := uuid.New()

	reqBody := ProcessPaymentRequest{
		RideID:        rideID.String(),
		Amount:        25.00,
		PaymentMethod: "wallet",
	}

	// Return nil driver ID (no driver assigned)
	var nilDriverID *uuid.UUID = nil
	mockRepo.On("GetRideDriverID", mock.Anything, rideID).Return(nilDriverID, nil)

	c, w := setupTestContext("POST", "/api/v1/payments/process", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.ProcessPayment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "no driver assigned")
}

func TestHandler_ProcessPayment_InsufficientWalletBalance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := ProcessPaymentRequest{
		RideID:        rideID.String(),
		Amount:        25.00,
		PaymentMethod: "wallet",
	}

	mockRepo.On("GetRideDriverID", mock.Anything, rideID).Return(&driverID, nil)
	mockRepo.On("ProcessPaymentWithWallet", mock.Anything, mock.AnythingOfType("*models.Payment"), mock.AnythingOfType("*models.WalletTransaction")).Return(common.NewBadRequestError("insufficient wallet balance", nil))

	c, w := setupTestContext("POST", "/api/v1/payments/process", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.ProcessPayment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "insufficient")
}

func TestHandler_ProcessPayment_StripeError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := ProcessPaymentRequest{
		RideID:        rideID.String(),
		Amount:        25.00,
		PaymentMethod: "stripe",
	}

	mockRepo.On("GetRideDriverID", mock.Anything, rideID).Return(&driverID, nil)
	mockStripe.On("CreatePaymentIntent", int64(2500), "usd", "", mock.Anything, mock.Anything).Return(nil, errors.New("stripe API error"))

	c, w := setupTestContext("POST", "/api/v1/payments/process", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.ProcessPayment(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_ProcessPayment_DatabaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := ProcessPaymentRequest{
		RideID:        rideID.String(),
		Amount:        25.00,
		PaymentMethod: "stripe",
	}

	mockRepo.On("GetRideDriverID", mock.Anything, rideID).Return(&driverID, nil)
	mockStripe.On("CreatePaymentIntent", int64(2500), "usd", "", mock.Anything, mock.Anything).Return(&stripe.PaymentIntent{
		ID:     "pi_test_123",
		Status: stripe.PaymentIntentStatusRequiresPaymentMethod,
	}, nil)
	mockRepo.On("CreatePayment", mock.Anything, mock.AnythingOfType("*models.Payment")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/payments/process", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.ProcessPayment(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_ProcessPayment_LargeAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := ProcessPaymentRequest{
		RideID:        rideID.String(),
		Amount:        9999.99,
		PaymentMethod: "wallet",
	}

	mockRepo.On("GetRideDriverID", mock.Anything, rideID).Return(&driverID, nil)
	mockRepo.On("ProcessPaymentWithWallet", mock.Anything, mock.AnythingOfType("*models.Payment"), mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/payments/process", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.ProcessPayment(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ProcessPayment_SmallAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := ProcessPaymentRequest{
		RideID:        rideID.String(),
		Amount:        0.01,
		PaymentMethod: "wallet",
	}

	mockRepo.On("GetRideDriverID", mock.Anything, rideID).Return(&driverID, nil)
	mockRepo.On("ProcessPaymentWithWallet", mock.Anything, mock.AnythingOfType("*models.Payment"), mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/payments/process", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.ProcessPayment(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================================
// TopUpWallet Handler Tests
// ============================================================================

func TestHandler_TopUpWallet_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	wallet := createTestWallet(userID, 100.00)

	reqBody := TopUpWalletRequest{
		Amount: 50.00,
	}

	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(wallet, nil)
	mockStripe.On("CreatePaymentIntent", int64(5000), "usd", "", mock.Anything, mock.Anything).Return(&stripe.PaymentIntent{
		ID:     "pi_topup_123",
		Status: stripe.PaymentIntentStatusRequiresPaymentMethod,
	}, nil)
	mockRepo.On("CreateWalletTransaction", mock.Anything, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/wallet/topup", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
	mockStripe.AssertExpectations(t)
}

func TestHandler_TopUpWallet_CreateNewWallet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()

	reqBody := TopUpWalletRequest{
		Amount: 50.00,
	}

	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(nil, common.NewNotFoundError("wallet not found", nil))
	mockRepo.On("CreateWallet", mock.Anything, mock.AnythingOfType("*models.Wallet")).Return(nil)
	mockStripe.On("CreatePaymentIntent", int64(5000), "usd", "", mock.Anything, mock.Anything).Return(&stripe.PaymentIntent{
		ID:     "pi_topup_123",
		Status: stripe.PaymentIntentStatusRequiresPaymentMethod,
	}, nil)
	mockRepo.On("CreateWalletTransaction", mock.Anything, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/wallet/topup", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_TopUpWallet_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	reqBody := TopUpWalletRequest{
		Amount: 50.00,
	}

	c, w := setupTestContext("POST", "/api/v1/wallet/topup", reqBody)
	// Don't set user context

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_TopUpWallet_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/wallet/topup", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/wallet/topup", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TopUpWallet_MissingAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()

	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/wallet/topup", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TopUpWallet_ZeroAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"amount": 0,
	}

	c, w := setupTestContext("POST", "/api/v1/wallet/topup", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TopUpWallet_NegativeAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"amount": -50.00,
	}

	c, w := setupTestContext("POST", "/api/v1/wallet/topup", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TopUpWallet_StripeError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	wallet := createTestWallet(userID, 100.00)

	reqBody := TopUpWalletRequest{
		Amount: 50.00,
	}

	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(wallet, nil)
	mockStripe.On("CreatePaymentIntent", int64(5000), "usd", "", mock.Anything, mock.Anything).Return(nil, errors.New("stripe error"))

	c, w := setupTestContext("POST", "/api/v1/wallet/topup", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_TopUpWallet_WithPaymentMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	wallet := createTestWallet(userID, 100.00)

	reqBody := TopUpWalletRequest{
		Amount:              50.00,
		StripePaymentMethod: "pm_test_123",
	}

	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(wallet, nil)
	mockStripe.On("CreatePaymentIntent", int64(5000), "usd", "", mock.Anything, mock.Anything).Return(&stripe.PaymentIntent{
		ID:     "pi_topup_123",
		Status: stripe.PaymentIntentStatusRequiresPaymentMethod,
	}, nil)
	mockRepo.On("CreateWalletTransaction", mock.Anything, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/wallet/topup", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_TopUpWallet_LargeAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	wallet := createTestWallet(userID, 100.00)

	reqBody := TopUpWalletRequest{
		Amount: 10000.00,
	}

	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(wallet, nil)
	mockStripe.On("CreatePaymentIntent", int64(1000000), "usd", "", mock.Anything, mock.Anything).Return(&stripe.PaymentIntent{
		ID:     "pi_topup_large",
		Status: stripe.PaymentIntentStatusRequiresPaymentMethod,
	}, nil)
	mockRepo.On("CreateWalletTransaction", mock.Anything, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/wallet/topup", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================================
// GetWallet Handler Tests
// ============================================================================

func TestHandler_GetWallet_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	wallet := createTestWallet(userID, 150.00)

	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(wallet, nil)

	c, w := setupTestContext("GET", "/api/v1/wallet", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetWallet(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, 150.00, data["balance"])
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetWallet_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	c, w := setupTestContext("GET", "/api/v1/wallet", nil)
	// Don't set user context

	handler.GetWallet(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetWallet_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()

	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(nil, common.NewNotFoundError("wallet not found", nil))

	c, w := setupTestContext("GET", "/api/v1/wallet", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetWallet(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetWallet_DatabaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()

	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/wallet", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetWallet(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_GetWallet_AsDriver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	wallet := createTestWallet(userID, 500.00)

	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(wallet, nil)

	c, w := setupTestContext("GET", "/api/v1/wallet", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetWallet(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

// ============================================================================
// GetWalletTransactions Handler Tests
// ============================================================================

func TestHandler_GetWalletTransactions_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	wallet := createTestWallet(userID, 150.00)
	transactions := []*models.WalletTransaction{
		createTestWalletTransaction(wallet.ID, "credit", 50.00),
		createTestWalletTransaction(wallet.ID, "debit", 25.00),
	}

	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(wallet, nil)
	mockRepo.On("GetWalletTransactionsWithTotal", mock.Anything, wallet.ID, 20, 0).Return(transactions, int64(2), nil)

	c, w := setupTestContext("GET", "/api/v1/wallet/transactions", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetWalletTransactions(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].([]interface{})
	assert.Len(t, data, 2)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetWalletTransactions_WithPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	wallet := createTestWallet(userID, 150.00)
	transactions := []*models.WalletTransaction{
		createTestWalletTransaction(wallet.ID, "credit", 50.00),
	}

	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(wallet, nil)
	mockRepo.On("GetWalletTransactionsWithTotal", mock.Anything, wallet.ID, 10, 5).Return(transactions, int64(15), nil)

	c, w := setupTestContext("GET", "/api/v1/wallet/transactions?limit=10&offset=5", nil)
	c.Request.URL.RawQuery = "limit=10&offset=5"
	setUserContext(c, userID, models.RoleRider)

	handler.GetWalletTransactions(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	meta := response["meta"].(map[string]interface{})
	assert.Equal(t, float64(10), meta["limit"])
	assert.Equal(t, float64(5), meta["offset"])
	assert.Equal(t, float64(15), meta["total"])
}

func TestHandler_GetWalletTransactions_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	c, w := setupTestContext("GET", "/api/v1/wallet/transactions", nil)
	// Don't set user context

	handler.GetWalletTransactions(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetWalletTransactions_WalletNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()

	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(nil, common.NewNotFoundError("wallet not found", nil))

	c, w := setupTestContext("GET", "/api/v1/wallet/transactions", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetWalletTransactions(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetWalletTransactions_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	wallet := createTestWallet(userID, 150.00)

	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(wallet, nil)
	mockRepo.On("GetWalletTransactionsWithTotal", mock.Anything, wallet.ID, 20, 0).Return([]*models.WalletTransaction{}, int64(0), nil)

	c, w := setupTestContext("GET", "/api/v1/wallet/transactions", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetWalletTransactions(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].([]interface{})
	assert.Len(t, data, 0)
}

func TestHandler_GetWalletTransactions_DatabaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	wallet := createTestWallet(userID, 150.00)

	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(wallet, nil)
	mockRepo.On("GetWalletTransactionsWithTotal", mock.Anything, wallet.ID, 20, 0).Return(nil, int64(0), errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/wallet/transactions", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetWalletTransactions(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetPayment Handler Tests
// ============================================================================

func TestHandler_GetPayment_Success_AsRider(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	paymentID := uuid.New()
	payment := createTestPayment(userID, driverID, rideID, "completed")
	payment.ID = paymentID

	mockRepo.On("GetPaymentByID", mock.Anything, paymentID).Return(payment, nil)

	c, w := setupTestContext("GET", "/api/v1/payments/"+paymentID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: paymentID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetPayment(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetPayment_Success_AsDriver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	paymentID := uuid.New()
	payment := createTestPayment(riderID, driverID, rideID, "completed")
	payment.ID = paymentID

	mockRepo.On("GetPaymentByID", mock.Anything, paymentID).Return(payment, nil)

	c, w := setupTestContext("GET", "/api/v1/payments/"+paymentID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: paymentID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetPayment(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetPayment_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	paymentID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/payments/"+paymentID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: paymentID.String()}}
	// Don't set user context

	handler.GetPayment(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetPayment_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/payments/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetPayment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid payment ID")
}

func TestHandler_GetPayment_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	paymentID := uuid.New()

	mockRepo.On("GetPaymentByID", mock.Anything, paymentID).Return(nil, common.NewNotFoundError("payment not found", nil))

	c, w := setupTestContext("GET", "/api/v1/payments/"+paymentID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: paymentID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetPayment(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetPayment_AccessDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	otherUserID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	paymentID := uuid.New()
	payment := createTestPayment(otherUserID, driverID, rideID, "completed")
	payment.ID = paymentID

	mockRepo.On("GetPaymentByID", mock.Anything, paymentID).Return(payment, nil)

	c, w := setupTestContext("GET", "/api/v1/payments/"+paymentID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: paymentID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetPayment(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "access denied")
}

func TestHandler_GetPayment_DatabaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	paymentID := uuid.New()

	mockRepo.On("GetPaymentByID", mock.Anything, paymentID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/payments/"+paymentID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: paymentID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetPayment(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// RefundPayment Handler Tests
// ============================================================================

func TestHandler_RefundPayment_Success_AsRider(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	paymentID := uuid.New()
	payment := createTestPayment(userID, driverID, rideID, "completed")
	payment.ID = paymentID

	wallet := createTestWallet(userID, 100.00)

	reqBody := RefundRequest{
		Reason: "rider_cancelled",
	}

	mockRepo.On("GetPaymentByID", mock.Anything, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", mock.Anything, wallet.ID, mock.AnythingOfType("float64")).Return(nil)
	mockRepo.On("CreateWalletTransaction", mock.Anything, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)
	mockRepo.On("UpdatePaymentStatus", mock.Anything, paymentID, "refunded", mock.Anything).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/payments/"+paymentID.String()+"/refund", reqBody)
	c.Params = gin.Params{{Key: "id", Value: paymentID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.RefundPayment(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_RefundPayment_Success_AsAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	adminID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	paymentID := uuid.New()
	payment := createTestPayment(riderID, driverID, rideID, "completed")
	payment.ID = paymentID

	wallet := createTestWallet(riderID, 100.00)

	reqBody := RefundRequest{
		Reason: "admin_refund",
	}

	mockRepo.On("GetPaymentByID", mock.Anything, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", mock.Anything, riderID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", mock.Anything, wallet.ID, mock.AnythingOfType("float64")).Return(nil)
	mockRepo.On("CreateWalletTransaction", mock.Anything, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)
	mockRepo.On("UpdatePaymentStatus", mock.Anything, paymentID, "refunded", mock.Anything).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/payments/"+paymentID.String()+"/refund", reqBody)
	c.Params = gin.Params{{Key: "id", Value: paymentID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.RefundPayment(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_RefundPayment_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	paymentID := uuid.New()

	reqBody := RefundRequest{
		Reason: "rider_cancelled",
	}

	c, w := setupTestContext("POST", "/api/v1/payments/"+paymentID.String()+"/refund", reqBody)
	c.Params = gin.Params{{Key: "id", Value: paymentID.String()}}
	// Don't set user context

	handler.RefundPayment(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_RefundPayment_ForbiddenForDriver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	driverID := uuid.New()
	paymentID := uuid.New()

	reqBody := RefundRequest{
		Reason: "driver_request",
	}

	c, w := setupTestContext("POST", "/api/v1/payments/"+paymentID.String()+"/refund", reqBody)
	c.Params = gin.Params{{Key: "id", Value: paymentID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.RefundPayment(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_RefundPayment_InvalidPaymentID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()

	reqBody := RefundRequest{
		Reason: "rider_cancelled",
	}

	c, w := setupTestContext("POST", "/api/v1/payments/invalid-uuid/refund", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID, models.RoleRider)

	handler.RefundPayment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid payment ID")
}

func TestHandler_RefundPayment_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	paymentID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/payments/"+paymentID.String()+"/refund", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/payments/"+paymentID.String()+"/refund", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: paymentID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.RefundPayment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RefundPayment_MissingReason(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	paymentID := uuid.New()

	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/payments/"+paymentID.String()+"/refund", reqBody)
	c.Params = gin.Params{{Key: "id", Value: paymentID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.RefundPayment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RefundPayment_PaymentNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	paymentID := uuid.New()

	reqBody := RefundRequest{
		Reason: "rider_cancelled",
	}

	mockRepo.On("GetPaymentByID", mock.Anything, paymentID).Return(nil, common.NewNotFoundError("payment not found", nil))

	c, w := setupTestContext("POST", "/api/v1/payments/"+paymentID.String()+"/refund", reqBody)
	c.Params = gin.Params{{Key: "id", Value: paymentID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.RefundPayment(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_RefundPayment_AccessDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	otherUserID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	paymentID := uuid.New()
	payment := createTestPayment(otherUserID, driverID, rideID, "completed")
	payment.ID = paymentID

	reqBody := RefundRequest{
		Reason: "rider_cancelled",
	}

	mockRepo.On("GetPaymentByID", mock.Anything, paymentID).Return(payment, nil)

	c, w := setupTestContext("POST", "/api/v1/payments/"+paymentID.String()+"/refund", reqBody)
	c.Params = gin.Params{{Key: "id", Value: paymentID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.RefundPayment(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_RefundPayment_AlreadyRefunded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	paymentID := uuid.New()
	payment := createTestPayment(userID, driverID, rideID, "refunded")
	payment.ID = paymentID

	reqBody := RefundRequest{
		Reason: "rider_cancelled",
	}

	mockRepo.On("GetPaymentByID", mock.Anything, paymentID).Return(payment, nil).Once()
	mockRepo.On("GetPaymentByID", mock.Anything, paymentID).Return(payment, nil).Once()

	c, w := setupTestContext("POST", "/api/v1/payments/"+paymentID.String()+"/refund", reqBody)
	c.Params = gin.Params{{Key: "id", Value: paymentID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.RefundPayment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "already refunded")
}

func TestHandler_RefundPayment_StripePayment(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	paymentID := uuid.New()
	stripePaymentID := "pi_test_123"
	stripeChargeID := "ch_test_123"
	payment := createTestPayment(userID, driverID, rideID, "completed")
	payment.ID = paymentID
	payment.PaymentMethod = "stripe"
	payment.StripePaymentID = &stripePaymentID
	payment.StripeChargeID = &stripeChargeID

	reqBody := RefundRequest{
		Reason: "rider_cancelled",
	}

	mockRepo.On("GetPaymentByID", mock.Anything, paymentID).Return(payment, nil)
	mockStripe.On("CreateRefund", stripeChargeID, mock.AnythingOfType("*int64"), "rider_cancelled").Return(&stripe.Refund{
		ID: "re_test_123",
	}, nil)
	mockRepo.On("UpdatePaymentStatus", mock.Anything, paymentID, "refunded", mock.Anything).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/payments/"+paymentID.String()+"/refund", reqBody)
	c.Params = gin.Params{{Key: "id", Value: paymentID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.RefundPayment(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockStripe.AssertExpectations(t)
}

func TestHandler_RefundPayment_StripeError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	paymentID := uuid.New()
	stripePaymentID := "pi_test_123"
	stripeChargeID := "ch_test_123"
	payment := createTestPayment(userID, driverID, rideID, "completed")
	payment.ID = paymentID
	payment.PaymentMethod = "stripe"
	payment.StripePaymentID = &stripePaymentID
	payment.StripeChargeID = &stripeChargeID

	reqBody := RefundRequest{
		Reason: "rider_cancelled",
	}

	mockRepo.On("GetPaymentByID", mock.Anything, paymentID).Return(payment, nil)
	mockStripe.On("CreateRefund", stripeChargeID, mock.AnythingOfType("*int64"), "rider_cancelled").Return(nil, errors.New("stripe error"))

	c, w := setupTestContext("POST", "/api/v1/payments/"+paymentID.String()+"/refund", reqBody)
	c.Params = gin.Params{{Key: "id", Value: paymentID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.RefundPayment(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// HandleStripeWebhook Handler Tests
// ============================================================================

func TestHandler_HandleStripeWebhook_Success_PaymentSucceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	webhookBody := map[string]interface{}{
		"type": "payment_intent.succeeded",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id": "pi_test_123",
			},
		},
	}

	c, w := setupTestContext("POST", "/api/v1/webhooks/stripe", webhookBody)

	handler.HandleStripeWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_HandleStripeWebhook_Success_PaymentFailed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	webhookBody := map[string]interface{}{
		"type": "payment_intent.payment_failed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id": "pi_test_123",
			},
		},
	}

	c, w := setupTestContext("POST", "/api/v1/webhooks/stripe", webhookBody)

	handler.HandleStripeWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_HandleStripeWebhook_Success_ChargeRefunded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	webhookBody := map[string]interface{}{
		"type": "charge.refunded",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id": "ch_test_123",
			},
		},
	}

	c, w := setupTestContext("POST", "/api/v1/webhooks/stripe", webhookBody)

	handler.HandleStripeWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_HandleStripeWebhook_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	c, w := setupTestContext("POST", "/api/v1/webhooks/stripe", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/webhooks/stripe", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.HandleStripeWebhook(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid webhook payload")
}

func TestHandler_HandleStripeWebhook_UnknownEventType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	webhookBody := map[string]interface{}{
		"type": "unknown.event",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id": "test_123",
			},
		},
	}

	c, w := setupTestContext("POST", "/api/v1/webhooks/stripe", webhookBody)

	handler.HandleStripeWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_HandleStripeWebhook_EmptyData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	webhookBody := map[string]interface{}{
		"type": "payment_intent.succeeded",
		"data": map[string]interface{}{},
	}

	c, w := setupTestContext("POST", "/api/v1/webhooks/stripe", webhookBody)

	handler.HandleStripeWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_HandleStripeWebhook_MissingPaymentIntentID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	webhookBody := map[string]interface{}{
		"type": "payment_intent.succeeded",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"amount": 2500,
			},
		},
	}

	c, w := setupTestContext("POST", "/api/v1/webhooks/stripe", webhookBody)

	handler.HandleStripeWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================================
// Edge Cases and Additional Tests
// ============================================================================

func TestHandler_ProcessPayment_ConcurrentRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := ProcessPaymentRequest{
		RideID:        rideID.String(),
		Amount:        25.00,
		PaymentMethod: "wallet",
	}

	// Simulate concurrent request failure (payment already processed)
	mockRepo.On("GetRideDriverID", mock.Anything, rideID).Return(&driverID, nil)
	mockRepo.On("ProcessPaymentWithWallet", mock.Anything, mock.AnythingOfType("*models.Payment"), mock.AnythingOfType("*models.WalletTransaction")).Return(common.NewBadRequestError("payment already processed for this ride", nil))

	c, w := setupTestContext("POST", "/api/v1/payments/process", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.ProcessPayment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "already processed")
}

func TestHandler_TopUpWallet_CreateWalletError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()

	reqBody := TopUpWalletRequest{
		Amount: 50.00,
	}

	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(nil, common.NewNotFoundError("wallet not found", nil))
	mockRepo.On("CreateWallet", mock.Anything, mock.AnythingOfType("*models.Wallet")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/wallet/topup", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_GetWalletTransactions_LargeLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	wallet := createTestWallet(userID, 150.00)

	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(wallet, nil)
	// The handler should cap the limit at 100 (MaxLimit)
	mockRepo.On("GetWalletTransactionsWithTotal", mock.Anything, wallet.ID, 100, 0).Return([]*models.WalletTransaction{}, int64(0), nil)

	c, w := setupTestContext("GET", "/api/v1/wallet/transactions?limit=1000", nil)
	c.Request.URL.RawQuery = "limit=1000"
	setUserContext(c, userID, models.RoleRider)

	handler.GetWalletTransactions(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetWalletTransactions_NegativeOffset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	wallet := createTestWallet(userID, 150.00)

	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(wallet, nil)
	// The handler should default to 0 for negative offset
	mockRepo.On("GetWalletTransactionsWithTotal", mock.Anything, wallet.ID, 20, 0).Return([]*models.WalletTransaction{}, int64(0), nil)

	c, w := setupTestContext("GET", "/api/v1/wallet/transactions?offset=-10", nil)
	c.Request.URL.RawQuery = "offset=-10"
	setUserContext(c, userID, models.RoleRider)

	handler.GetWalletTransactions(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_RefundPayment_FullRefund(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	paymentID := uuid.New()
	payment := createTestPayment(userID, driverID, rideID, "completed")
	payment.ID = paymentID

	wallet := createTestWallet(userID, 100.00)

	reqBody := RefundRequest{
		Reason: "full_refund",
	}

	mockRepo.On("GetPaymentByID", mock.Anything, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(wallet, nil)
	// Full refund amount (no cancellation fee)
	mockRepo.On("UpdateWalletBalance", mock.Anything, wallet.ID, payment.Amount).Return(nil)
	mockRepo.On("CreateWalletTransaction", mock.Anything, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)
	mockRepo.On("UpdatePaymentStatus", mock.Anything, paymentID, "refunded", mock.Anything).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/payments/"+paymentID.String()+"/refund", reqBody)
	c.Params = gin.Params{{Key: "id", Value: paymentID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.RefundPayment(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ProcessPayment_DecimalPrecision(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := ProcessPaymentRequest{
		RideID:        rideID.String(),
		Amount:        25.99,
		PaymentMethod: "stripe",
	}

	mockRepo.On("GetRideDriverID", mock.Anything, rideID).Return(&driverID, nil)
	// Verify correct conversion to cents (25.99 * 100 = 2599)
	mockStripe.On("CreatePaymentIntent", int64(2599), "usd", "", mock.Anything, mock.Anything).Return(&stripe.PaymentIntent{
		ID:     "pi_test_123",
		Status: stripe.PaymentIntentStatusRequiresPaymentMethod,
	}, nil)
	mockRepo.On("CreatePayment", mock.Anything, mock.AnythingOfType("*models.Payment")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/payments/process", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.ProcessPayment(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockStripe.AssertExpectations(t)
}

func TestHandler_GetPayment_EmptyPaymentID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/payments/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetPayment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RefundPayment_WalletRefundError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	paymentID := uuid.New()
	payment := createTestPayment(userID, driverID, rideID, "completed")
	payment.ID = paymentID

	wallet := createTestWallet(userID, 100.00)

	reqBody := RefundRequest{
		Reason: "rider_cancelled",
	}

	mockRepo.On("GetPaymentByID", mock.Anything, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", mock.Anything, wallet.ID, mock.AnythingOfType("float64")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/payments/"+paymentID.String()+"/refund", reqBody)
	c.Params = gin.Params{{Key: "id", Value: paymentID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.RefundPayment(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_TopUpWallet_TransactionCreateError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStripe := new(MockStripeClient)
	handler := createTestHandler(mockRepo, mockStripe)

	userID := uuid.New()
	wallet := createTestWallet(userID, 100.00)

	reqBody := TopUpWalletRequest{
		Amount: 50.00,
	}

	mockRepo.On("GetWalletByUserID", mock.Anything, userID).Return(wallet, nil)
	mockStripe.On("CreatePaymentIntent", int64(5000), "usd", "", mock.Anything, mock.Anything).Return(&stripe.PaymentIntent{
		ID:     "pi_topup_123",
		Status: stripe.PaymentIntentStatusRequiresPaymentMethod,
	}, nil)
	mockRepo.On("CreateWalletTransaction", mock.Anything, mock.AnythingOfType("*models.WalletTransaction")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/wallet/topup", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TopUpWallet(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_ProcessPayment_AllPaymentMethods(t *testing.T) {
	tests := []struct {
		name          string
		paymentMethod string
		expectSuccess bool
	}{
		{"wallet method", "wallet", true},
		{"stripe method", "stripe", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			mockStripe := new(MockStripeClient)
			handler := createTestHandler(mockRepo, mockStripe)

			userID := uuid.New()
			driverID := uuid.New()
			rideID := uuid.New()

			reqBody := ProcessPaymentRequest{
				RideID:        rideID.String(),
				Amount:        25.00,
				PaymentMethod: tt.paymentMethod,
			}

			mockRepo.On("GetRideDriverID", mock.Anything, rideID).Return(&driverID, nil)

			if tt.paymentMethod == "wallet" {
				mockRepo.On("ProcessPaymentWithWallet", mock.Anything, mock.AnythingOfType("*models.Payment"), mock.AnythingOfType("*models.WalletTransaction")).Return(nil)
			} else {
				mockStripe.On("CreatePaymentIntent", int64(2500), "usd", "", mock.Anything, mock.Anything).Return(&stripe.PaymentIntent{
					ID:     "pi_test_123",
					Status: stripe.PaymentIntentStatusRequiresPaymentMethod,
				}, nil)
				mockRepo.On("CreatePayment", mock.Anything, mock.AnythingOfType("*models.Payment")).Return(nil)
			}

			c, w := setupTestContext("POST", "/api/v1/payments/process", reqBody)
			setUserContext(c, userID, models.RoleRider)

			handler.ProcessPayment(c)

			if tt.expectSuccess {
				assert.Equal(t, http.StatusOK, w.Code)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}
