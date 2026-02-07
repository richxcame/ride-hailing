package subscriptions

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
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// Mock Implementations
// ============================================================================

// MockRepository implements RepositoryInterface for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreatePlan(ctx context.Context, plan *SubscriptionPlan) error {
	args := m.Called(ctx, plan)
	return args.Error(0)
}

func (m *MockRepository) GetPlanByID(ctx context.Context, id uuid.UUID) (*SubscriptionPlan, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SubscriptionPlan), args.Error(1)
}

func (m *MockRepository) GetPlanBySlug(ctx context.Context, slug string) (*SubscriptionPlan, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SubscriptionPlan), args.Error(1)
}

func (m *MockRepository) ListActivePlans(ctx context.Context) ([]*SubscriptionPlan, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*SubscriptionPlan), args.Error(1)
}

func (m *MockRepository) ListAllPlans(ctx context.Context) ([]*SubscriptionPlan, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*SubscriptionPlan), args.Error(1)
}

func (m *MockRepository) UpdatePlanStatus(ctx context.Context, id uuid.UUID, status PlanStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockRepository) CreateSubscription(ctx context.Context, sub *Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}

func (m *MockRepository) GetActiveSubscription(ctx context.Context, userID uuid.UUID) (*Subscription, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Subscription), args.Error(1)
}

func (m *MockRepository) GetSubscriptionByID(ctx context.Context, id uuid.UUID) (*Subscription, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Subscription), args.Error(1)
}

func (m *MockRepository) UpdateSubscription(ctx context.Context, sub *Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}

func (m *MockRepository) IncrementRideUsage(ctx context.Context, subID uuid.UUID, savings float64) error {
	args := m.Called(ctx, subID, savings)
	return args.Error(0)
}

func (m *MockRepository) IncrementUpgradeUsage(ctx context.Context, subID uuid.UUID) error {
	args := m.Called(ctx, subID)
	return args.Error(0)
}

func (m *MockRepository) IncrementCancellationUsage(ctx context.Context, subID uuid.UUID) error {
	args := m.Called(ctx, subID)
	return args.Error(0)
}

func (m *MockRepository) GetExpiredSubscriptions(ctx context.Context) ([]*Subscription, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Subscription), args.Error(1)
}

func (m *MockRepository) GetSubscriberCount(ctx context.Context, planID uuid.UUID) (int, error) {
	args := m.Called(ctx, planID)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) CreateUsageLog(ctx context.Context, log *SubscriptionUsageLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockRepository) GetUsageLogs(ctx context.Context, subID uuid.UUID, limit, offset int) ([]*SubscriptionUsageLog, error) {
	args := m.Called(ctx, subID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*SubscriptionUsageLog), args.Error(1)
}

func (m *MockRepository) GetUserAverageMonthlySpend(ctx context.Context, userID uuid.UUID) (float64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(float64), args.Error(1)
}

// MockPaymentProcessor implements PaymentProcessor for testing
type MockPaymentProcessor struct {
	mock.Mock
}

func (m *MockPaymentProcessor) ChargeSubscription(ctx context.Context, userID uuid.UUID, amount float64, currency, paymentMethod string) error {
	args := m.Called(ctx, userID, amount, currency, paymentMethod)
	return args.Error(0)
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

func createTestHandler(mockRepo *MockRepository, mockPayments *MockPaymentProcessor) *Handler {
	service := NewService(mockRepo, mockPayments)
	return NewHandler(service)
}

func createTestPlan() *SubscriptionPlan {
	ridesIncluded := 20
	return &SubscriptionPlan{
		ID:                uuid.New(),
		Name:              "Premium Plan",
		Slug:              "premium-plan",
		Description:       "Premium subscription with 20 rides",
		PlanType:          PlanTypePackage,
		BillingPeriod:     BillingMonthly,
		Price:             29.99,
		Currency:          "USD",
		Status:            PlanStatusActive,
		RidesIncluded:     &ridesIncluded,
		DiscountPct:       15.0,
		PriorityMatching:  true,
		FreeCancellations: 3,
		FreeUpgrades:      2,
		SurgeProtection:   true,
		PopularBadge:      true,
		SavingsLabel:      "Save up to 20%",
		DisplayOrder:      1,
		TrialDays:         7,
		TrialRides:        3,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

func createTestSubscription(userID, planID uuid.UUID) *Subscription {
	now := time.Now()
	periodEnd := now.AddDate(0, 1, 0)
	return &Subscription{
		ID:                 uuid.New(),
		UserID:             userID,
		PlanID:             planID,
		Status:             SubStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   periodEnd,
		RidesUsed:          5,
		UpgradesUsed:       1,
		CancellationsUsed:  0,
		TotalSaved:         25.50,
		PaymentMethod:      "card",
		AutoRenew:          true,
		ActivatedAt:        &now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// ============================================================================
// ListPlans Handler Tests
// ============================================================================

func TestHandler_ListPlans_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	plans := []*SubscriptionPlan{createTestPlan(), createTestPlan()}
	plans[1].Name = "Basic Plan"
	plans[1].Slug = "basic-plan"

	mockRepo.On("ListActivePlans", mock.Anything).Return(plans, nil)

	c, w := setupTestContext("GET", "/api/v1/subscriptions/plans", nil)
	userID := uuid.New()
	setUserContext(c, userID, models.RoleRider)

	handler.ListPlans(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_ListPlans_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	mockRepo.On("ListActivePlans", mock.Anything).Return([]*SubscriptionPlan{}, nil)

	c, w := setupTestContext("GET", "/api/v1/subscriptions/plans", nil)
	userID := uuid.New()
	setUserContext(c, userID, models.RoleRider)

	handler.ListPlans(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_ListPlans_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	mockRepo.On("ListActivePlans", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/subscriptions/plans", nil)
	userID := uuid.New()
	setUserContext(c, userID, models.RoleRider)

	handler.ListPlans(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

// ============================================================================
// ComparePlans Handler Tests
// ============================================================================

func TestHandler_ComparePlans_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()
	plans := []*SubscriptionPlan{createTestPlan()}

	mockRepo.On("ListActivePlans", mock.Anything).Return(plans, nil)
	mockRepo.On("GetUserAverageMonthlySpend", mock.Anything, userID).Return(100.0, nil)

	c, w := setupTestContext("GET", "/api/v1/subscriptions/compare", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.ComparePlans(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_ComparePlans_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	c, w := setupTestContext("GET", "/api/v1/subscriptions/compare", nil)
	// Don't set user context

	handler.ComparePlans(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_ComparePlans_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()

	mockRepo.On("ListActivePlans", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/subscriptions/compare", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.ComparePlans(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// Subscribe Handler Tests
// ============================================================================

func TestHandler_Subscribe_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()
	plan := createTestPlan()

	reqBody := SubscribeRequest{
		PlanID:        plan.ID,
		PaymentMethod: "card",
		AutoRenew:     true,
	}

	mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(nil, nil)
	mockRepo.On("GetPlanByID", mock.Anything, plan.ID).Return(plan, nil)
	mockRepo.On("CreateSubscription", mock.Anything, mock.AnythingOfType("*subscriptions.Subscription")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/subscriptions", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.Subscribe(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_Subscribe_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	plan := createTestPlan()
	reqBody := SubscribeRequest{
		PlanID:        plan.ID,
		PaymentMethod: "card",
		AutoRenew:     true,
	}

	c, w := setupTestContext("POST", "/api/v1/subscriptions", reqBody)
	// Don't set user context

	handler.Subscribe(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Subscribe_InvalidRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/subscriptions", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/subscriptions", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleRider)

	handler.Subscribe(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Subscribe_AlreadySubscribed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()
	plan := createTestPlan()
	existingSub := createTestSubscription(userID, plan.ID)

	reqBody := SubscribeRequest{
		PlanID:        plan.ID,
		PaymentMethod: "card",
		AutoRenew:     true,
	}

	mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(existingSub, nil)

	c, w := setupTestContext("POST", "/api/v1/subscriptions", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.Subscribe(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Subscribe_PlanNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()
	planID := uuid.New()

	reqBody := SubscribeRequest{
		PlanID:        planID,
		PaymentMethod: "card",
		AutoRenew:     true,
	}

	mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(nil, nil)
	mockRepo.On("GetPlanByID", mock.Anything, planID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("POST", "/api/v1/subscriptions", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.Subscribe(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Subscribe_PlanInactive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()
	plan := createTestPlan()
	plan.Status = PlanStatusInactive

	reqBody := SubscribeRequest{
		PlanID:        plan.ID,
		PaymentMethod: "card",
		AutoRenew:     true,
	}

	mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(nil, nil)
	mockRepo.On("GetPlanByID", mock.Anything, plan.ID).Return(plan, nil)

	c, w := setupTestContext("POST", "/api/v1/subscriptions", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.Subscribe(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// GetSubscription Handler Tests
// ============================================================================

func TestHandler_GetSubscription_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()
	plan := createTestPlan()
	sub := createTestSubscription(userID, plan.ID)

	mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(sub, nil)
	mockRepo.On("GetPlanByID", mock.Anything, plan.ID).Return(plan, nil)

	c, w := setupTestContext("GET", "/api/v1/subscriptions/me", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetSubscription(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.True(t, data["has_subscription"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetSubscription_NoSubscription(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()

	// Mock returns an error, which the service wraps in common.NewNotFoundError
	// The handler detects this AppError and returns 404
	mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/subscriptions/me", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetSubscription(c)

	// Service wraps errors in AppError, so handler returns 404 NotFound
	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetSubscription_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	c, w := setupTestContext("GET", "/api/v1/subscriptions/me", nil)
	// Don't set user context

	handler.GetSubscription(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// PauseSubscription Handler Tests
// ============================================================================

func TestHandler_PauseSubscription_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()
	plan := createTestPlan()
	sub := createTestSubscription(userID, plan.ID)

	mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(sub, nil)
	mockRepo.On("UpdateSubscription", mock.Anything, mock.AnythingOfType("*subscriptions.Subscription")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/subscriptions/me/pause", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.PauseSubscription(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_PauseSubscription_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	c, w := setupTestContext("POST", "/api/v1/subscriptions/me/pause", nil)
	// Don't set user context

	handler.PauseSubscription(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_PauseSubscription_NoSubscription(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()

	mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("POST", "/api/v1/subscriptions/me/pause", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.PauseSubscription(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_PauseSubscription_AlreadyPaused(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()
	plan := createTestPlan()
	sub := createTestSubscription(userID, plan.ID)
	sub.Status = SubStatusPaused

	mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(sub, nil)

	c, w := setupTestContext("POST", "/api/v1/subscriptions/me/pause", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.PauseSubscription(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// ResumeSubscription Handler Tests
// ============================================================================

func TestHandler_ResumeSubscription_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()
	plan := createTestPlan()
	sub := createTestSubscription(userID, plan.ID)
	sub.Status = SubStatusPaused

	mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(sub, nil)
	mockRepo.On("UpdateSubscription", mock.Anything, mock.AnythingOfType("*subscriptions.Subscription")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/subscriptions/me/resume", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.ResumeSubscription(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_ResumeSubscription_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	c, w := setupTestContext("POST", "/api/v1/subscriptions/me/resume", nil)
	// Don't set user context

	handler.ResumeSubscription(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_ResumeSubscription_NoSubscription(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()

	mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("POST", "/api/v1/subscriptions/me/resume", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.ResumeSubscription(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_ResumeSubscription_NotPaused(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()
	plan := createTestPlan()
	sub := createTestSubscription(userID, plan.ID)
	sub.Status = SubStatusActive

	mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(sub, nil)

	c, w := setupTestContext("POST", "/api/v1/subscriptions/me/resume", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.ResumeSubscription(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// CancelSubscription Handler Tests
// ============================================================================

func TestHandler_CancelSubscription_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()
	plan := createTestPlan()
	sub := createTestSubscription(userID, plan.ID)

	reqBody := map[string]string{
		"reason": "Too expensive",
	}

	mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(sub, nil)
	mockRepo.On("UpdateSubscription", mock.Anything, mock.AnythingOfType("*subscriptions.Subscription")).Return(nil)

	c, w := setupTestContext("DELETE", "/api/v1/subscriptions/me", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.CancelSubscription(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_CancelSubscription_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	c, w := setupTestContext("DELETE", "/api/v1/subscriptions/me", nil)
	// Don't set user context

	handler.CancelSubscription(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CancelSubscription_NoSubscription(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()

	mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("DELETE", "/api/v1/subscriptions/me", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.CancelSubscription(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_CancelSubscription_WithoutReason(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()
	plan := createTestPlan()
	sub := createTestSubscription(userID, plan.ID)

	mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(sub, nil)
	mockRepo.On("UpdateSubscription", mock.Anything, mock.AnythingOfType("*subscriptions.Subscription")).Return(nil)

	c, w := setupTestContext("DELETE", "/api/v1/subscriptions/me", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.CancelSubscription(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================================
// CreatePlan Admin Handler Tests
// ============================================================================

func TestHandler_CreatePlan_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	adminID := uuid.New()
	reqBody := CreatePlanRequest{
		Name:              "New Plan",
		Slug:              "new-plan",
		Description:       "A new subscription plan",
		PlanType:          PlanTypeDiscount,
		BillingPeriod:     BillingMonthly,
		Price:             19.99,
		Currency:          "USD",
		DiscountPct:       10.0,
		PriorityMatching:  false,
		FreeCancellations: 2,
		FreeUpgrades:      1,
		SurgeProtection:   false,
	}

	mockRepo.On("GetPlanBySlug", mock.Anything, "new-plan").Return(nil, nil)
	mockRepo.On("CreatePlan", mock.Anything, mock.AnythingOfType("*subscriptions.SubscriptionPlan")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/subscriptions/plans", reqBody)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.CreatePlan(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_CreatePlan_InvalidRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	adminID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/admin/subscriptions/plans", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/subscriptions/plans", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, adminID, models.RoleAdmin)

	handler.CreatePlan(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreatePlan_DuplicateSlug(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	adminID := uuid.New()
	existingPlan := createTestPlan()

	reqBody := CreatePlanRequest{
		Name:          "Duplicate Plan",
		Slug:          existingPlan.Slug,
		Description:   "A duplicate plan",
		PlanType:      PlanTypeDiscount,
		BillingPeriod: BillingMonthly,
		Price:         19.99,
		Currency:      "USD",
	}

	mockRepo.On("GetPlanBySlug", mock.Anything, existingPlan.Slug).Return(existingPlan, nil)

	c, w := setupTestContext("POST", "/api/v1/admin/subscriptions/plans", reqBody)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.CreatePlan(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// ListAllPlans Admin Handler Tests
// ============================================================================

func TestHandler_ListAllPlans_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	adminID := uuid.New()
	plans := []*SubscriptionPlan{createTestPlan()}
	inactivePlan := createTestPlan()
	inactivePlan.Status = PlanStatusInactive
	plans = append(plans, inactivePlan)

	mockRepo.On("ListAllPlans", mock.Anything).Return(plans, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/subscriptions/plans", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ListAllPlans(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_ListAllPlans_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	adminID := uuid.New()

	mockRepo.On("ListAllPlans", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/subscriptions/plans", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ListAllPlans(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// DeactivatePlan Admin Handler Tests
// ============================================================================

func TestHandler_DeactivatePlan_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	adminID := uuid.New()
	planID := uuid.New()

	mockRepo.On("UpdatePlanStatus", mock.Anything, planID, PlanStatusInactive).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/subscriptions/plans/"+planID.String()+"/deactivate", nil)
	c.Params = gin.Params{{Key: "id", Value: planID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.DeactivatePlan(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_DeactivatePlan_InvalidPlanID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	adminID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/admin/subscriptions/plans/invalid-uuid/deactivate", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.DeactivatePlan(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DeactivatePlan_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	adminID := uuid.New()
	planID := uuid.New()

	mockRepo.On("UpdatePlanStatus", mock.Anything, planID, PlanStatusInactive).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/admin/subscriptions/plans/"+planID.String()+"/deactivate", nil)
	c.Params = gin.Params{{Key: "id", Value: planID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.DeactivatePlan(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// Table-Driven Tests - Authorization Cases
// ============================================================================

func TestHandler_AuthorizationCases(t *testing.T) {
	tests := []struct {
		name           string
		endpoint       string
		method         string
		setAuth        bool
		userRole       models.UserRole
		expectedStatus int
		handler        func(*Handler, *gin.Context)
	}{
		{
			name:           "ComparePlans - unauthorized",
			endpoint:       "/api/v1/subscriptions/compare",
			method:         "GET",
			setAuth:        false,
			expectedStatus: http.StatusUnauthorized,
			handler: func(h *Handler, c *gin.Context) {
				h.ComparePlans(c)
			},
		},
		{
			name:           "Subscribe - unauthorized",
			endpoint:       "/api/v1/subscriptions",
			method:         "POST",
			setAuth:        false,
			expectedStatus: http.StatusUnauthorized,
			handler: func(h *Handler, c *gin.Context) {
				h.Subscribe(c)
			},
		},
		{
			name:           "GetSubscription - unauthorized",
			endpoint:       "/api/v1/subscriptions/me",
			method:         "GET",
			setAuth:        false,
			expectedStatus: http.StatusUnauthorized,
			handler: func(h *Handler, c *gin.Context) {
				h.GetSubscription(c)
			},
		},
		{
			name:           "PauseSubscription - unauthorized",
			endpoint:       "/api/v1/subscriptions/me/pause",
			method:         "POST",
			setAuth:        false,
			expectedStatus: http.StatusUnauthorized,
			handler: func(h *Handler, c *gin.Context) {
				h.PauseSubscription(c)
			},
		},
		{
			name:           "ResumeSubscription - unauthorized",
			endpoint:       "/api/v1/subscriptions/me/resume",
			method:         "POST",
			setAuth:        false,
			expectedStatus: http.StatusUnauthorized,
			handler: func(h *Handler, c *gin.Context) {
				h.ResumeSubscription(c)
			},
		},
		{
			name:           "CancelSubscription - unauthorized",
			endpoint:       "/api/v1/subscriptions/me",
			method:         "DELETE",
			setAuth:        false,
			expectedStatus: http.StatusUnauthorized,
			handler: func(h *Handler, c *gin.Context) {
				h.CancelSubscription(c)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			mockPayments := new(MockPaymentProcessor)
			handler := createTestHandler(mockRepo, mockPayments)

			c, w := setupTestContext(tt.method, tt.endpoint, nil)

			if tt.setAuth {
				setUserContext(c, uuid.New(), tt.userRole)
			}

			tt.handler(handler, c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// Table-Driven Tests - Subscription Status Cases
// ============================================================================

func TestHandler_SubscriptionStatusCases(t *testing.T) {
	tests := []struct {
		name           string
		initialStatus  SubscriptionStatus
		action         string
		expectedStatus int
	}{
		{
			name:           "pause active subscription",
			initialStatus:  SubStatusActive,
			action:         "pause",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "pause already paused subscription",
			initialStatus:  SubStatusPaused,
			action:         "pause",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "resume paused subscription",
			initialStatus:  SubStatusPaused,
			action:         "resume",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "resume active subscription",
			initialStatus:  SubStatusActive,
			action:         "resume",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "cancel active subscription",
			initialStatus:  SubStatusActive,
			action:         "cancel",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "cancel paused subscription",
			initialStatus:  SubStatusPaused,
			action:         "cancel",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			mockPayments := new(MockPaymentProcessor)
			handler := createTestHandler(mockRepo, mockPayments)

			userID := uuid.New()
			plan := createTestPlan()
			sub := createTestSubscription(userID, plan.ID)
			sub.Status = tt.initialStatus

			mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(sub, nil)
			if tt.expectedStatus == http.StatusOK {
				mockRepo.On("UpdateSubscription", mock.Anything, mock.AnythingOfType("*subscriptions.Subscription")).Return(nil)
			}

			var c *gin.Context
			var w *httptest.ResponseRecorder

			switch tt.action {
			case "pause":
				c, w = setupTestContext("POST", "/api/v1/subscriptions/me/pause", nil)
				setUserContext(c, userID, models.RoleRider)
				handler.PauseSubscription(c)
			case "resume":
				c, w = setupTestContext("POST", "/api/v1/subscriptions/me/resume", nil)
				setUserContext(c, userID, models.RoleRider)
				handler.ResumeSubscription(c)
			case "cancel":
				c, w = setupTestContext("DELETE", "/api/v1/subscriptions/me", nil)
				setUserContext(c, userID, models.RoleRider)
				handler.CancelSubscription(c)
			}

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_SuccessResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	plans := []*SubscriptionPlan{createTestPlan()}

	mockRepo.On("ListActivePlans", mock.Anything).Return(plans, nil)

	c, w := setupTestContext("GET", "/api/v1/subscriptions/plans", nil)
	userID := uuid.New()
	setUserContext(c, userID, models.RoleRider)

	handler.ListPlans(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	// Verify response structure
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
	assert.NotNil(t, response["meta"])
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	mockRepo.On("ListActivePlans", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/subscriptions/plans", nil)
	userID := uuid.New()
	setUserContext(c, userID, models.RoleRider)

	handler.ListPlans(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)

	// Verify error response structure
	assert.False(t, response["success"].(bool))
	assert.NotNil(t, response["error"])

	errorInfo := response["error"].(map[string]interface{})
	assert.NotNil(t, errorInfo["code"])
	assert.NotNil(t, errorInfo["message"])
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestHandler_EmptyPlanID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	adminID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/admin/subscriptions/plans//deactivate", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.DeactivatePlan(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SubscribeWithTrialPlan(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()
	plan := createTestPlan()
	plan.TrialDays = 14

	reqBody := SubscribeRequest{
		PlanID:        plan.ID,
		PaymentMethod: "card",
		AutoRenew:     true,
	}

	mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(nil, nil)
	mockRepo.On("GetPlanByID", mock.Anything, plan.ID).Return(plan, nil)
	mockRepo.On("CreateSubscription", mock.Anything, mock.AnythingOfType("*subscriptions.Subscription")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/subscriptions", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.Subscribe(c)

	assert.Equal(t, http.StatusOK, w.Code)
	// Payment should not be called for trial plans
	mockPayments.AssertNotCalled(t, "ChargeSubscription")
}

func TestHandler_SubscribeWithPayment(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()
	plan := createTestPlan()
	plan.TrialDays = 0 // No trial

	reqBody := SubscribeRequest{
		PlanID:        plan.ID,
		PaymentMethod: "card",
		AutoRenew:     true,
	}

	mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(nil, nil)
	mockRepo.On("GetPlanByID", mock.Anything, plan.ID).Return(plan, nil)
	mockPayments.On("ChargeSubscription", mock.Anything, userID, plan.Price, plan.Currency, "card").Return(nil)
	mockRepo.On("CreateSubscription", mock.Anything, mock.AnythingOfType("*subscriptions.Subscription")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/subscriptions", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.Subscribe(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockPayments.AssertCalled(t, "ChargeSubscription", mock.Anything, userID, plan.Price, plan.Currency, "card")
}

func TestHandler_SubscribePaymentFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	userID := uuid.New()
	plan := createTestPlan()
	plan.TrialDays = 0 // No trial

	reqBody := SubscribeRequest{
		PlanID:        plan.ID,
		PaymentMethod: "card",
		AutoRenew:     true,
	}

	mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(nil, nil)
	mockRepo.On("GetPlanByID", mock.Anything, plan.ID).Return(plan, nil)
	mockPayments.On("ChargeSubscription", mock.Anything, userID, plan.Price, plan.Currency, "card").Return(errors.New("payment failed"))

	c, w := setupTestContext("POST", "/api/v1/subscriptions", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.Subscribe(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// AppError Response Tests
// ============================================================================

func TestHandler_AppErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupMock      func(*MockRepository, uuid.UUID, uuid.UUID)
		expectedStatus int
	}{
		{
			name: "bad request error - inactive plan",
			setupMock: func(mockRepo *MockRepository, userID, planID uuid.UUID) {
				mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(nil, nil)
				// Return an inactive plan to trigger BadRequest
				inactivePlan := createTestPlan()
				inactivePlan.ID = planID
				inactivePlan.Status = PlanStatusInactive
				mockRepo.On("GetPlanByID", mock.Anything, planID).Return(inactivePlan, nil)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "not found error - plan not found",
			setupMock: func(mockRepo *MockRepository, userID, planID uuid.UUID) {
				mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(nil, nil)
				mockRepo.On("GetPlanByID", mock.Anything, planID).Return(nil, errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			mockPayments := new(MockPaymentProcessor)
			handler := createTestHandler(mockRepo, mockPayments)

			userID := uuid.New()
			planID := uuid.New()

			reqBody := SubscribeRequest{
				PlanID:        planID,
				PaymentMethod: "card",
				AutoRenew:     true,
			}

			// Setup mock based on test case
			tt.setupMock(mockRepo, userID, planID)

			c, w := setupTestContext("POST", "/api/v1/subscriptions", reqBody)
			setUserContext(c, userID, models.RoleRider)

			handler.Subscribe(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// Mock Repository Full Coverage Tests
// ============================================================================

func TestMockRepository_AllMethods(t *testing.T) {
	mockRepo := new(MockRepository)

	ctx := context.Background()
	userID := uuid.New()
	planID := uuid.New()
	subID := uuid.New()
	plan := createTestPlan()
	sub := createTestSubscription(userID, planID)

	// Test CreatePlan
	mockRepo.On("CreatePlan", mock.Anything, plan).Return(nil)
	err := mockRepo.CreatePlan(ctx, plan)
	assert.NoError(t, err)

	// Test GetPlanByID
	mockRepo.On("GetPlanByID", mock.Anything, planID).Return(plan, nil)
	result, err := mockRepo.GetPlanByID(ctx, planID)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Test GetPlanBySlug
	mockRepo.On("GetPlanBySlug", mock.Anything, "test-slug").Return(plan, nil)
	result, err = mockRepo.GetPlanBySlug(ctx, "test-slug")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Test ListActivePlans
	mockRepo.On("ListActivePlans", mock.Anything).Return([]*SubscriptionPlan{plan}, nil)
	plans, err := mockRepo.ListActivePlans(ctx)
	assert.NoError(t, err)
	assert.Len(t, plans, 1)

	// Test ListAllPlans
	mockRepo.On("ListAllPlans", mock.Anything).Return([]*SubscriptionPlan{plan}, nil)
	plans, err = mockRepo.ListAllPlans(ctx)
	assert.NoError(t, err)
	assert.Len(t, plans, 1)

	// Test UpdatePlanStatus
	mockRepo.On("UpdatePlanStatus", mock.Anything, planID, PlanStatusInactive).Return(nil)
	err = mockRepo.UpdatePlanStatus(ctx, planID, PlanStatusInactive)
	assert.NoError(t, err)

	// Test CreateSubscription
	mockRepo.On("CreateSubscription", mock.Anything, sub).Return(nil)
	err = mockRepo.CreateSubscription(ctx, sub)
	assert.NoError(t, err)

	// Test GetActiveSubscription
	mockRepo.On("GetActiveSubscription", mock.Anything, userID).Return(sub, nil)
	subResult, err := mockRepo.GetActiveSubscription(ctx, userID)
	assert.NoError(t, err)
	assert.NotNil(t, subResult)

	// Test GetSubscriptionByID
	mockRepo.On("GetSubscriptionByID", mock.Anything, subID).Return(sub, nil)
	subResult, err = mockRepo.GetSubscriptionByID(ctx, subID)
	assert.NoError(t, err)
	assert.NotNil(t, subResult)

	// Test UpdateSubscription
	mockRepo.On("UpdateSubscription", mock.Anything, sub).Return(nil)
	err = mockRepo.UpdateSubscription(ctx, sub)
	assert.NoError(t, err)

	// Test IncrementRideUsage
	mockRepo.On("IncrementRideUsage", mock.Anything, subID, 10.0).Return(nil)
	err = mockRepo.IncrementRideUsage(ctx, subID, 10.0)
	assert.NoError(t, err)

	// Test IncrementUpgradeUsage
	mockRepo.On("IncrementUpgradeUsage", mock.Anything, subID).Return(nil)
	err = mockRepo.IncrementUpgradeUsage(ctx, subID)
	assert.NoError(t, err)

	// Test IncrementCancellationUsage
	mockRepo.On("IncrementCancellationUsage", mock.Anything, subID).Return(nil)
	err = mockRepo.IncrementCancellationUsage(ctx, subID)
	assert.NoError(t, err)

	// Test GetExpiredSubscriptions
	mockRepo.On("GetExpiredSubscriptions", mock.Anything).Return([]*Subscription{sub}, nil)
	subs, err := mockRepo.GetExpiredSubscriptions(ctx)
	assert.NoError(t, err)
	assert.Len(t, subs, 1)

	// Test GetSubscriberCount
	mockRepo.On("GetSubscriberCount", mock.Anything, planID).Return(10, nil)
	count, err := mockRepo.GetSubscriberCount(ctx, planID)
	assert.NoError(t, err)
	assert.Equal(t, 10, count)

	// Test CreateUsageLog
	log := &SubscriptionUsageLog{ID: uuid.New()}
	mockRepo.On("CreateUsageLog", mock.Anything, log).Return(nil)
	err = mockRepo.CreateUsageLog(ctx, log)
	assert.NoError(t, err)

	// Test GetUsageLogs
	mockRepo.On("GetUsageLogs", mock.Anything, subID, 10, 0).Return([]*SubscriptionUsageLog{log}, nil)
	logs, err := mockRepo.GetUsageLogs(ctx, subID, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, logs, 1)

	// Test GetUserAverageMonthlySpend
	mockRepo.On("GetUserAverageMonthlySpend", mock.Anything, userID).Return(150.0, nil)
	spend, err := mockRepo.GetUserAverageMonthlySpend(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, 150.0, spend)

	mockRepo.AssertExpectations(t)
}

// ============================================================================
// Service Error Propagation Tests
// ============================================================================

func TestHandler_ServiceErrorPropagation(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockRepository, *MockPaymentProcessor, uuid.UUID)
		request        func(uuid.UUID) interface{}
		handler        func(*Handler, *gin.Context)
		expectedStatus int
	}{
		{
			name: "create subscription fails",
			setupMock: func(mr *MockRepository, mp *MockPaymentProcessor, userID uuid.UUID) {
				plan := createTestPlan()
				mr.On("GetActiveSubscription", mock.Anything, userID).Return(nil, nil)
				mr.On("GetPlanByID", mock.Anything, mock.Anything).Return(plan, nil)
				mr.On("CreateSubscription", mock.Anything, mock.AnythingOfType("*subscriptions.Subscription")).Return(errors.New("db error"))
			},
			request: func(userID uuid.UUID) interface{} {
				return SubscribeRequest{
					PlanID:        uuid.New(),
					PaymentMethod: "card",
					AutoRenew:     true,
				}
			},
			handler: func(h *Handler, c *gin.Context) {
				h.Subscribe(c)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "update subscription fails on pause",
			setupMock: func(mr *MockRepository, mp *MockPaymentProcessor, userID uuid.UUID) {
				plan := createTestPlan()
				sub := createTestSubscription(userID, plan.ID)
				mr.On("GetActiveSubscription", mock.Anything, userID).Return(sub, nil)
				mr.On("UpdateSubscription", mock.Anything, mock.AnythingOfType("*subscriptions.Subscription")).Return(errors.New("db error"))
			},
			request: func(userID uuid.UUID) interface{} {
				return nil
			},
			handler: func(h *Handler, c *gin.Context) {
				h.PauseSubscription(c)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "create plan fails",
			setupMock: func(mr *MockRepository, mp *MockPaymentProcessor, userID uuid.UUID) {
				mr.On("GetPlanBySlug", mock.Anything, "test-plan").Return(nil, nil)
				mr.On("CreatePlan", mock.Anything, mock.AnythingOfType("*subscriptions.SubscriptionPlan")).Return(errors.New("db error"))
			},
			request: func(userID uuid.UUID) interface{} {
				return CreatePlanRequest{
					Name:          "Test Plan",
					Slug:          "test-plan",
					Description:   "Test description",
					PlanType:      PlanTypeDiscount,
					BillingPeriod: BillingMonthly,
					Price:         19.99,
					Currency:      "USD",
				}
			},
			handler: func(h *Handler, c *gin.Context) {
				h.CreatePlan(c)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			mockPayments := new(MockPaymentProcessor)
			handler := createTestHandler(mockRepo, mockPayments)

			userID := uuid.New()
			tt.setupMock(mockRepo, mockPayments, userID)

			reqBody := tt.request(userID)
			var c *gin.Context
			var w *httptest.ResponseRecorder

			if reqBody != nil {
				c, w = setupTestContext("POST", "/test", reqBody)
			} else {
				c, w = setupTestContext("POST", "/test", nil)
			}
			setUserContext(c, userID, models.RoleAdmin)

			tt.handler(handler, c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// Pagination Tests
// ============================================================================

func TestHandler_ListPlans_WithPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockPayments := new(MockPaymentProcessor)
	handler := createTestHandler(mockRepo, mockPayments)

	plans := make([]*SubscriptionPlan, 25)
	for i := 0; i < 25; i++ {
		plans[i] = createTestPlan()
	}

	mockRepo.On("ListActivePlans", mock.Anything).Return(plans, nil)

	c, w := setupTestContext("GET", "/api/v1/subscriptions/plans?limit=10&offset=0", nil)
	userID := uuid.New()
	setUserContext(c, userID, models.RoleRider)

	handler.ListPlans(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["meta"])
}

// ============================================================================
// Invalid JSON Tests
// ============================================================================

func TestHandler_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		endpoint string
		handler  func(*Handler, *gin.Context)
	}{
		{
			name:     "subscribe with invalid json",
			endpoint: "/api/v1/subscriptions",
			handler: func(h *Handler, c *gin.Context) {
				h.Subscribe(c)
			},
		},
		{
			name:     "create plan with invalid json",
			endpoint: "/api/v1/admin/subscriptions/plans",
			handler: func(h *Handler, c *gin.Context) {
				h.CreatePlan(c)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			mockPayments := new(MockPaymentProcessor)
			handler := createTestHandler(mockRepo, mockPayments)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", tt.endpoint, bytes.NewReader([]byte("invalid json {")))
			c.Request.Header.Set("Content-Type", "application/json")
			setUserContext(c, uuid.New(), models.RoleAdmin)

			tt.handler(handler, c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			response := parseResponse(w)
			assert.False(t, response["success"].(bool))
		})
	}
}
