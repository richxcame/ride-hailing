package giftcards

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
// Mock Implementation
// ============================================================================

// MockGiftCardsRepository implements RepositoryInterface for testing
type MockGiftCardsRepository struct {
	mock.Mock
}

func (m *MockGiftCardsRepository) CreateCard(ctx context.Context, card *GiftCard) error {
	args := m.Called(ctx, card)
	return args.Error(0)
}

func (m *MockGiftCardsRepository) GetCardByCode(ctx context.Context, code string) (*GiftCard, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GiftCard), args.Error(1)
}

func (m *MockGiftCardsRepository) GetCardByID(ctx context.Context, id uuid.UUID) (*GiftCard, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GiftCard), args.Error(1)
}

func (m *MockGiftCardsRepository) RedeemCard(ctx context.Context, cardID, userID uuid.UUID) error {
	args := m.Called(ctx, cardID, userID)
	return args.Error(0)
}

func (m *MockGiftCardsRepository) DeductBalance(ctx context.Context, cardID uuid.UUID, amount float64) (bool, error) {
	args := m.Called(ctx, cardID, amount)
	return args.Bool(0), args.Error(1)
}

func (m *MockGiftCardsRepository) GetActiveCardsByUser(ctx context.Context, userID uuid.UUID) ([]GiftCard, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]GiftCard), args.Error(1)
}

func (m *MockGiftCardsRepository) GetPurchasedCardsByUser(ctx context.Context, userID uuid.UUID) ([]GiftCard, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]GiftCard), args.Error(1)
}

func (m *MockGiftCardsRepository) CreateTransaction(ctx context.Context, tx *GiftCardTransaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockGiftCardsRepository) GetTransactionsByUser(ctx context.Context, userID uuid.UUID, limit int) ([]GiftCardTransaction, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]GiftCardTransaction), args.Error(1)
}

func (m *MockGiftCardsRepository) GetTotalBalance(ctx context.Context, userID uuid.UUID) (float64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockGiftCardsRepository) ExpireCards(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

// ============================================================================
// Helper Functions
// ============================================================================

func setupGiftCardsTestContext(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

func setGiftCardsUserContext(c *gin.Context, userID uuid.UUID, role models.UserRole) {
	c.Set("user_id", userID)
	c.Set("user_role", role)
	c.Set("user_email", "test@example.com")
}

func parseGiftCardsResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

func createTestGiftCardsHandler(mockRepo *MockGiftCardsRepository) *Handler {
	service := NewService(mockRepo)
	return NewHandler(service)
}

func createTestGiftCard(purchaserID *uuid.UUID) *GiftCard {
	expiresAt := time.Now().AddDate(1, 0, 0)
	return &GiftCard{
		ID:              uuid.New(),
		Code:            "ABCD-1234-EFGH-5678",
		CardType:        CardTypePurchased,
		Status:          CardStatusActive,
		OriginalAmount:  50.0,
		RemainingAmount: 50.0,
		Currency:        "USD",
		PurchaserID:     purchaserID,
		ExpiresAt:       &expiresAt,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func createTestGiftCardTransaction(cardID, userID uuid.UUID) *GiftCardTransaction {
	return &GiftCardTransaction{
		ID:            uuid.New(),
		CardID:        cardID,
		UserID:        userID,
		Amount:        10.0,
		BalanceBefore: 50.0,
		BalanceAfter:  40.0,
		Description:   "Ride payment",
		CreatedAt:     time.Now(),
	}
}

// ============================================================================
// Purchase Handler Tests
// ============================================================================

func TestHandler_Purchase_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()
	reqBody := PurchaseGiftCardRequest{
		Amount:   50.0,
		Currency: "USD",
	}

	mockRepo.On("CreateCard", mock.Anything, mock.AnythingOfType("*giftcards.GiftCard")).Return(nil)

	c, w := setupGiftCardsTestContext("POST", "/api/v1/gift-cards", reqBody)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.Purchase(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseGiftCardsResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_Purchase_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	reqBody := PurchaseGiftCardRequest{
		Amount:   50.0,
		Currency: "USD",
	}

	c, w := setupGiftCardsTestContext("POST", "/api/v1/gift-cards", reqBody)
	// Don't set user context

	handler.Purchase(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Purchase_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()

	c, w := setupGiftCardsTestContext("POST", "/api/v1/gift-cards", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/gift-cards", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.Purchase(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Purchase_AmountTooLow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()
	reqBody := PurchaseGiftCardRequest{
		Amount:   2.0, // Below minimum of 5
		Currency: "USD",
	}

	c, w := setupGiftCardsTestContext("POST", "/api/v1/gift-cards", reqBody)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.Purchase(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Purchase_AmountTooHigh(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()
	reqBody := PurchaseGiftCardRequest{
		Amount:   1000.0, // Above maximum of 500
		Currency: "USD",
	}

	c, w := setupGiftCardsTestContext("POST", "/api/v1/gift-cards", reqBody)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.Purchase(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Purchase_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()
	reqBody := PurchaseGiftCardRequest{
		Amount:   50.0,
		Currency: "USD",
	}

	mockRepo.On("CreateCard", mock.Anything, mock.AnythingOfType("*giftcards.GiftCard")).Return(errors.New("database error"))

	c, w := setupGiftCardsTestContext("POST", "/api/v1/gift-cards", reqBody)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.Purchase(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_Purchase_WithRecipient(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()
	email := "recipient@example.com"
	name := "John Doe"
	message := "Happy Birthday!"
	reqBody := PurchaseGiftCardRequest{
		Amount:          100.0,
		Currency:        "USD",
		RecipientEmail:  &email,
		RecipientName:   &name,
		PersonalMessage: &message,
	}

	mockRepo.On("CreateCard", mock.Anything, mock.AnythingOfType("*giftcards.GiftCard")).Return(nil)

	c, w := setupGiftCardsTestContext("POST", "/api/v1/gift-cards", reqBody)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.Purchase(c)

	assert.Equal(t, http.StatusCreated, w.Code)
}

// ============================================================================
// Redeem Handler Tests
// ============================================================================

func TestHandler_Redeem_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()
	purchaserID := uuid.New()
	card := createTestGiftCard(&purchaserID)
	reqBody := RedeemGiftCardRequest{
		Code: card.Code,
	}

	mockRepo.On("GetCardByCode", mock.Anything, card.Code).Return(card, nil)
	mockRepo.On("RedeemCard", mock.Anything, card.ID, userID).Return(nil)

	c, w := setupGiftCardsTestContext("POST", "/api/v1/gift-cards/redeem", reqBody)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.Redeem(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseGiftCardsResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_Redeem_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	reqBody := RedeemGiftCardRequest{
		Code: "ABCD-1234-EFGH-5678",
	}

	c, w := setupGiftCardsTestContext("POST", "/api/v1/gift-cards/redeem", reqBody)
	// Don't set user context

	handler.Redeem(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Redeem_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()

	c, w := setupGiftCardsTestContext("POST", "/api/v1/gift-cards/redeem", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/gift-cards/redeem", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.Redeem(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Redeem_CardNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()
	reqBody := RedeemGiftCardRequest{
		Code: "INVALID-CODE",
	}

	mockRepo.On("GetCardByCode", mock.Anything, "INVALID-CODE").Return(nil, common.NewNotFoundError("gift card not found", nil))

	c, w := setupGiftCardsTestContext("POST", "/api/v1/gift-cards/redeem", reqBody)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.Redeem(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Redeem_CardInactive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()
	purchaserID := uuid.New()
	card := createTestGiftCard(&purchaserID)
	card.Status = CardStatusDisabled
	reqBody := RedeemGiftCardRequest{
		Code: card.Code,
	}

	mockRepo.On("GetCardByCode", mock.Anything, card.Code).Return(card, nil)

	c, w := setupGiftCardsTestContext("POST", "/api/v1/gift-cards/redeem", reqBody)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.Redeem(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Redeem_ZeroBalance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()
	purchaserID := uuid.New()
	card := createTestGiftCard(&purchaserID)
	card.RemainingAmount = 0
	reqBody := RedeemGiftCardRequest{
		Code: card.Code,
	}

	mockRepo.On("GetCardByCode", mock.Anything, card.Code).Return(card, nil)

	c, w := setupGiftCardsTestContext("POST", "/api/v1/gift-cards/redeem", reqBody)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.Redeem(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Redeem_Expired(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()
	purchaserID := uuid.New()
	card := createTestGiftCard(&purchaserID)
	pastDate := time.Now().AddDate(-1, 0, 0)
	card.ExpiresAt = &pastDate
	reqBody := RedeemGiftCardRequest{
		Code: card.Code,
	}

	mockRepo.On("GetCardByCode", mock.Anything, card.Code).Return(card, nil)

	c, w := setupGiftCardsTestContext("POST", "/api/v1/gift-cards/redeem", reqBody)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.Redeem(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Redeem_AlreadyRedeemedByOther(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()
	otherUserID := uuid.New()
	purchaserID := uuid.New()
	card := createTestGiftCard(&purchaserID)
	card.RecipientID = &otherUserID
	reqBody := RedeemGiftCardRequest{
		Code: card.Code,
	}

	mockRepo.On("GetCardByCode", mock.Anything, card.Code).Return(card, nil)

	c, w := setupGiftCardsTestContext("POST", "/api/v1/gift-cards/redeem", reqBody)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.Redeem(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// CheckBalance Handler Tests
// ============================================================================

func TestHandler_CheckBalance_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	purchaserID := uuid.New()
	card := createTestGiftCard(&purchaserID)

	mockRepo.On("GetCardByCode", mock.Anything, card.Code).Return(card, nil)

	c, w := setupGiftCardsTestContext("GET", "/api/v1/gift-cards/balance/"+card.Code, nil)
	c.Params = gin.Params{{Key: "code", Value: card.Code}}

	handler.CheckBalance(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseGiftCardsResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_CheckBalance_EmptyCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	c, w := setupGiftCardsTestContext("GET", "/api/v1/gift-cards/balance/", nil)
	c.Params = gin.Params{{Key: "code", Value: ""}}

	handler.CheckBalance(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CheckBalance_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	code := "INVALID-CODE"

	mockRepo.On("GetCardByCode", mock.Anything, code).Return(nil, common.NewNotFoundError("gift card not found", nil))

	c, w := setupGiftCardsTestContext("GET", "/api/v1/gift-cards/balance/"+code, nil)
	c.Params = gin.Params{{Key: "code", Value: code}}

	handler.CheckBalance(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_CheckBalance_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	code := "TEST-CODE"

	mockRepo.On("GetCardByCode", mock.Anything, code).Return(nil, errors.New("database error"))

	c, w := setupGiftCardsTestContext("GET", "/api/v1/gift-cards/balance/"+code, nil)
	c.Params = gin.Params{{Key: "code", Value: code}}

	handler.CheckBalance(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ============================================================================
// GetMySummary Handler Tests
// ============================================================================

func TestHandler_GetMySummary_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()
	purchaserID := uuid.New()
	card := createTestGiftCard(&purchaserID)
	card.RecipientID = &userID
	cards := []GiftCard{*card}
	transactions := []GiftCardTransaction{*createTestGiftCardTransaction(card.ID, userID)}

	mockRepo.On("GetActiveCardsByUser", mock.Anything, userID).Return(cards, nil)
	mockRepo.On("GetTotalBalance", mock.Anything, userID).Return(50.0, nil)
	mockRepo.On("GetTransactionsByUser", mock.Anything, userID, 20).Return(transactions, nil)

	c, w := setupGiftCardsTestContext("GET", "/api/v1/gift-cards/me", nil)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.GetMySummary(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseGiftCardsResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetMySummary_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	c, w := setupGiftCardsTestContext("GET", "/api/v1/gift-cards/me", nil)
	// Don't set user context

	handler.GetMySummary(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetMySummary_EmptyCards(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetActiveCardsByUser", mock.Anything, userID).Return([]GiftCard{}, nil)
	mockRepo.On("GetTotalBalance", mock.Anything, userID).Return(0.0, nil)
	mockRepo.On("GetTransactionsByUser", mock.Anything, userID, 20).Return([]GiftCardTransaction{}, nil)

	c, w := setupGiftCardsTestContext("GET", "/api/v1/gift-cards/me", nil)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.GetMySummary(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetMySummary_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetActiveCardsByUser", mock.Anything, userID).Return(nil, errors.New("database error"))
	mockRepo.On("GetTotalBalance", mock.Anything, userID).Return(0.0, errors.New("error"))
	mockRepo.On("GetTransactionsByUser", mock.Anything, userID, 20).Return(nil, errors.New("error"))

	c, w := setupGiftCardsTestContext("GET", "/api/v1/gift-cards/me", nil)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.GetMySummary(c)

	// The service handles errors gracefully
	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================================
// GetPurchasedCards Handler Tests
// ============================================================================

func TestHandler_GetPurchasedCards_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()
	card := createTestGiftCard(&userID)
	cards := []GiftCard{*card}

	mockRepo.On("GetPurchasedCardsByUser", mock.Anything, userID).Return(cards, nil)

	c, w := setupGiftCardsTestContext("GET", "/api/v1/gift-cards/purchased", nil)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.GetPurchasedCards(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseGiftCardsResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetPurchasedCards_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	c, w := setupGiftCardsTestContext("GET", "/api/v1/gift-cards/purchased", nil)
	// Don't set user context

	handler.GetPurchasedCards(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetPurchasedCards_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetPurchasedCardsByUser", mock.Anything, userID).Return([]GiftCard{}, nil)

	c, w := setupGiftCardsTestContext("GET", "/api/v1/gift-cards/purchased", nil)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.GetPurchasedCards(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetPurchasedCards_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetPurchasedCardsByUser", mock.Anything, userID).Return(nil, errors.New("database error"))

	c, w := setupGiftCardsTestContext("GET", "/api/v1/gift-cards/purchased", nil)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.GetPurchasedCards(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// CreateBulk Handler Tests (Admin)
// ============================================================================

func TestHandler_CreateBulk_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	adminID := uuid.New()
	reqBody := CreateBulkRequest{
		Count:    5,
		Amount:   25.0,
		Currency: "USD",
		CardType: CardTypePromotional,
	}

	mockRepo.On("CreateCard", mock.Anything, mock.AnythingOfType("*giftcards.GiftCard")).Return(nil).Times(5)

	c, w := setupGiftCardsTestContext("POST", "/api/v1/admin/gift-cards/bulk", reqBody)
	setGiftCardsUserContext(c, adminID, models.RoleAdmin)

	handler.CreateBulk(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseGiftCardsResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_CreateBulk_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	adminID := uuid.New()

	c, w := setupGiftCardsTestContext("POST", "/api/v1/admin/gift-cards/bulk", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/gift-cards/bulk", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setGiftCardsUserContext(c, adminID, models.RoleAdmin)

	handler.CreateBulk(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateBulk_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	adminID := uuid.New()
	reqBody := CreateBulkRequest{
		Count:    5,
		Amount:   25.0,
		Currency: "USD",
		CardType: CardTypePromotional,
	}

	mockRepo.On("CreateCard", mock.Anything, mock.AnythingOfType("*giftcards.GiftCard")).Return(errors.New("database error"))

	c, w := setupGiftCardsTestContext("POST", "/api/v1/admin/gift-cards/bulk", reqBody)
	setGiftCardsUserContext(c, adminID, models.RoleAdmin)

	handler.CreateBulk(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_CreateBulk_WithExpiry(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	adminID := uuid.New()
	expiresInDays := 30
	reqBody := CreateBulkRequest{
		Count:         10,
		Amount:        50.0,
		Currency:      "USD",
		CardType:      CardTypeCorporate,
		ExpiresInDays: &expiresInDays,
	}

	mockRepo.On("CreateCard", mock.Anything, mock.AnythingOfType("*giftcards.GiftCard")).Return(nil).Times(10)

	c, w := setupGiftCardsTestContext("POST", "/api/v1/admin/gift-cards/bulk", reqBody)
	setGiftCardsUserContext(c, adminID, models.RoleAdmin)

	handler.CreateBulk(c)

	assert.Equal(t, http.StatusCreated, w.Code)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_Purchase_AmountValidation(t *testing.T) {
	tests := []struct {
		name           string
		amount         float64
		expectedStatus int
	}{
		{
			name:           "minimum valid amount",
			amount:         5.0,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "maximum valid amount",
			amount:         500.0,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "below minimum",
			amount:         4.99,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "above maximum",
			amount:         500.01,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "middle valid amount",
			amount:         100.0,
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockGiftCardsRepository)
			handler := createTestGiftCardsHandler(mockRepo)
			userID := uuid.New()

			reqBody := PurchaseGiftCardRequest{
				Amount:   tt.amount,
				Currency: "USD",
			}

			if tt.expectedStatus == http.StatusCreated {
				mockRepo.On("CreateCard", mock.Anything, mock.AnythingOfType("*giftcards.GiftCard")).Return(nil)
			}

			c, w := setupGiftCardsTestContext("POST", "/api/v1/gift-cards", reqBody)
			setGiftCardsUserContext(c, userID, models.RoleRider)

			handler.Purchase(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_CheckBalance_CardStatusCases(t *testing.T) {
	tests := []struct {
		name            string
		status          CardStatus
		remainingAmount float64
		expired         bool
		expectedValid   bool
	}{
		{
			name:            "active card with balance",
			status:          CardStatusActive,
			remainingAmount: 50.0,
			expired:         false,
			expectedValid:   true,
		},
		{
			name:            "disabled card",
			status:          CardStatusDisabled,
			remainingAmount: 50.0,
			expired:         false,
			expectedValid:   false,
		},
		{
			name:            "expired card",
			status:          CardStatusActive,
			remainingAmount: 50.0,
			expired:         true,
			expectedValid:   false,
		},
		{
			name:            "zero balance",
			status:          CardStatusActive,
			remainingAmount: 0,
			expired:         false,
			expectedValid:   false,
		},
		{
			name:            "redeemed card",
			status:          CardStatusRedeemed,
			remainingAmount: 0,
			expired:         false,
			expectedValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockGiftCardsRepository)
			handler := createTestGiftCardsHandler(mockRepo)

			purchaserID := uuid.New()
			card := createTestGiftCard(&purchaserID)
			card.Status = tt.status
			card.RemainingAmount = tt.remainingAmount
			if tt.expired {
				pastDate := time.Now().AddDate(-1, 0, 0)
				card.ExpiresAt = &pastDate
			}

			mockRepo.On("GetCardByCode", mock.Anything, card.Code).Return(card, nil)

			c, w := setupGiftCardsTestContext("GET", "/api/v1/gift-cards/balance/"+card.Code, nil)
			c.Params = gin.Params{{Key: "code", Value: card.Code}}

			handler.CheckBalance(c)

			assert.Equal(t, http.StatusOK, w.Code)
			response := parseGiftCardsResponse(w)
			assert.True(t, response["success"].(bool))

			// Check the is_valid field in the response
			data := response["data"].(map[string]interface{})
			assert.Equal(t, tt.expectedValid, data["is_valid"].(bool))
		})
	}
}

func TestHandler_CreateBulk_CardTypeCases(t *testing.T) {
	tests := []struct {
		name           string
		cardType       CardType
		count          int
		amount         float64
		expectedStatus int
	}{
		{
			name:           "promotional cards",
			cardType:       CardTypePromotional,
			count:          10,
			amount:         25.0,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "corporate cards",
			cardType:       CardTypeCorporate,
			count:          50,
			amount:         100.0,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "refund cards",
			cardType:       CardTypeRefund,
			count:          1,
			amount:         15.0,
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockGiftCardsRepository)
			handler := createTestGiftCardsHandler(mockRepo)
			adminID := uuid.New()

			reqBody := CreateBulkRequest{
				Count:    tt.count,
				Amount:   tt.amount,
				Currency: "USD",
				CardType: tt.cardType,
			}

			mockRepo.On("CreateCard", mock.Anything, mock.AnythingOfType("*giftcards.GiftCard")).Return(nil).Times(tt.count)

			c, w := setupGiftCardsTestContext("POST", "/api/v1/admin/gift-cards/bulk", reqBody)
			setGiftCardsUserContext(c, adminID, models.RoleAdmin)

			handler.CreateBulk(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestHandler_Redeem_AlreadyRedeemedBySameUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()
	purchaserID := uuid.New()
	card := createTestGiftCard(&purchaserID)
	card.RecipientID = &userID // Already redeemed by this user
	redeemedAt := time.Now()
	card.RedeemedAt = &redeemedAt
	reqBody := RedeemGiftCardRequest{
		Code: card.Code,
	}

	mockRepo.On("GetCardByCode", mock.Anything, card.Code).Return(card, nil)

	c, w := setupGiftCardsTestContext("POST", "/api/v1/gift-cards/redeem", reqBody)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.Redeem(c)

	// Should return success since the user already redeemed it
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_CheckBalance_NoExpiry(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	purchaserID := uuid.New()
	card := createTestGiftCard(&purchaserID)
	card.ExpiresAt = nil // No expiry

	mockRepo.On("GetCardByCode", mock.Anything, card.Code).Return(card, nil)

	c, w := setupGiftCardsTestContext("GET", "/api/v1/gift-cards/balance/"+card.Code, nil)
	c.Params = gin.Params{{Key: "code", Value: card.Code}}

	handler.CheckBalance(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseGiftCardsResponse(w)
	data := response["data"].(map[string]interface{})
	assert.True(t, data["is_valid"].(bool))
}

func TestHandler_Purchase_DefaultCurrency(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()
	reqBody := PurchaseGiftCardRequest{
		Amount: 50.0,
		// No currency specified - should default to USD
	}

	mockRepo.On("CreateCard", mock.Anything, mock.MatchedBy(func(card *GiftCard) bool {
		return card.Currency == "USD"
	})).Return(nil)

	c, w := setupGiftCardsTestContext("POST", "/api/v1/gift-cards", reqBody)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.Purchase(c)

	assert.Equal(t, http.StatusCreated, w.Code)
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_GetMySummary_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetActiveCardsByUser", mock.Anything, userID).Return([]GiftCard{}, nil)
	mockRepo.On("GetTotalBalance", mock.Anything, userID).Return(0.0, nil)
	mockRepo.On("GetTransactionsByUser", mock.Anything, userID, 20).Return([]GiftCardTransaction{}, nil)

	c, w := setupGiftCardsTestContext("GET", "/api/v1/gift-cards/me", nil)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.GetMySummary(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseGiftCardsResponse(w)

	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["total_balance"])
	assert.NotNil(t, data["active_cards"])
	assert.NotNil(t, data["cards"])
	assert.NotNil(t, data["recent_transactions"])
}

func TestHandler_GetPurchasedCards_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockGiftCardsRepository)
	handler := createTestGiftCardsHandler(mockRepo)

	userID := uuid.New()
	card := createTestGiftCard(&userID)
	cards := []GiftCard{*card}

	mockRepo.On("GetPurchasedCardsByUser", mock.Anything, userID).Return(cards, nil)

	c, w := setupGiftCardsTestContext("GET", "/api/v1/gift-cards/purchased", nil)
	setGiftCardsUserContext(c, userID, models.RoleRider)

	handler.GetPurchasedCards(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseGiftCardsResponse(w)

	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["cards"])
	assert.NotNil(t, data["count"])
}
