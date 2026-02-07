package earnings

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
// Mock Service Interface
// ============================================================================

// EarningsServiceInterface defines the interface for the earnings service used by the handler
type EarningsServiceInterface interface {
	GetEarningsSummary(ctx context.Context, driverID uuid.UUID, period string) (*EarningsSummary, error)
	GetDailyEarnings(ctx context.Context, driverID uuid.UUID, period string) ([]DailyEarning, error)
	GetEarningsHistory(ctx context.Context, driverID uuid.UUID, period string, limit, offset int) (*EarningsHistoryResponse, error)
	GetUnpaidBalance(ctx context.Context, driverID uuid.UUID) (float64, error)
	RequestPayout(ctx context.Context, driverID uuid.UUID, req *RequestPayoutRequest) (*DriverPayout, error)
	GetPayoutHistory(ctx context.Context, driverID uuid.UUID, limit, offset int) (*PayoutHistoryResponse, error)
	AddBankAccount(ctx context.Context, driverID uuid.UUID, req *AddBankAccountRequest) (*DriverBankAccount, error)
	GetBankAccounts(ctx context.Context, driverID uuid.UUID) ([]DriverBankAccount, error)
	DeleteBankAccount(ctx context.Context, driverID uuid.UUID, accountID uuid.UUID) error
	SetEarningGoal(ctx context.Context, driverID uuid.UUID, req *SetEarningGoalRequest) (*EarningGoal, error)
	GetEarningGoals(ctx context.Context, driverID uuid.UUID) ([]EarningGoalStatus, error)
	RecordBonus(ctx context.Context, driverID uuid.UUID, amount float64, description string) (*DriverEarning, error)
}

// MockEarningsService is a mock implementation of EarningsServiceInterface
type MockEarningsService struct {
	mock.Mock
}

func (m *MockEarningsService) GetEarningsSummary(ctx context.Context, driverID uuid.UUID, period string) (*EarningsSummary, error) {
	args := m.Called(ctx, driverID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EarningsSummary), args.Error(1)
}

func (m *MockEarningsService) GetDailyEarnings(ctx context.Context, driverID uuid.UUID, period string) ([]DailyEarning, error) {
	args := m.Called(ctx, driverID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]DailyEarning), args.Error(1)
}

func (m *MockEarningsService) GetEarningsHistory(ctx context.Context, driverID uuid.UUID, period string, limit, offset int) (*EarningsHistoryResponse, error) {
	args := m.Called(ctx, driverID, period, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EarningsHistoryResponse), args.Error(1)
}

func (m *MockEarningsService) GetUnpaidBalance(ctx context.Context, driverID uuid.UUID) (float64, error) {
	args := m.Called(ctx, driverID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockEarningsService) RequestPayout(ctx context.Context, driverID uuid.UUID, req *RequestPayoutRequest) (*DriverPayout, error) {
	args := m.Called(ctx, driverID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverPayout), args.Error(1)
}

func (m *MockEarningsService) GetPayoutHistory(ctx context.Context, driverID uuid.UUID, limit, offset int) (*PayoutHistoryResponse, error) {
	args := m.Called(ctx, driverID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PayoutHistoryResponse), args.Error(1)
}

func (m *MockEarningsService) AddBankAccount(ctx context.Context, driverID uuid.UUID, req *AddBankAccountRequest) (*DriverBankAccount, error) {
	args := m.Called(ctx, driverID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverBankAccount), args.Error(1)
}

func (m *MockEarningsService) GetBankAccounts(ctx context.Context, driverID uuid.UUID) ([]DriverBankAccount, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]DriverBankAccount), args.Error(1)
}

func (m *MockEarningsService) DeleteBankAccount(ctx context.Context, driverID uuid.UUID, accountID uuid.UUID) error {
	args := m.Called(ctx, driverID, accountID)
	return args.Error(0)
}

func (m *MockEarningsService) SetEarningGoal(ctx context.Context, driverID uuid.UUID, req *SetEarningGoalRequest) (*EarningGoal, error) {
	args := m.Called(ctx, driverID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EarningGoal), args.Error(1)
}

func (m *MockEarningsService) GetEarningGoals(ctx context.Context, driverID uuid.UUID) ([]EarningGoalStatus, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]EarningGoalStatus), args.Error(1)
}

func (m *MockEarningsService) RecordBonus(ctx context.Context, driverID uuid.UUID, amount float64, description string) (*DriverEarning, error) {
	args := m.Called(ctx, driverID, amount, description)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverEarning), args.Error(1)
}

// ============================================================================
// Testable Handler (wraps mock service)
// ============================================================================

// TestableHandler provides testable handler methods with mock service injection
type TestableHandler struct {
	service *MockEarningsService
}

func NewTestableHandler(mockService *MockEarningsService) *TestableHandler {
	return &TestableHandler{service: mockService}
}

// GetSummary returns earnings summary for a period
func (h *TestableHandler) GetSummary(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	period := c.DefaultQuery("period", "today")

	summary, err := h.service.GetEarningsSummary(c.Request.Context(), driverID.(uuid.UUID), period)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get earnings summary")
		return
	}

	common.SuccessResponse(c, summary)
}

// GetDailyBreakdown returns day-by-day earnings
func (h *TestableHandler) GetDailyBreakdown(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	period := c.DefaultQuery("period", "this_week")

	daily, err := h.service.GetDailyEarnings(c.Request.Context(), driverID.(uuid.UUID), period)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get daily earnings")
		return
	}

	common.SuccessResponse(c, gin.H{
		"daily":  daily,
		"period": period,
	})
}

// GetHistory returns paginated earnings history
func (h *TestableHandler) GetHistory(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	period := c.DefaultQuery("period", "this_month")

	// Default pagination
	limit := 20
	offset := 0

	resp, err := h.service.GetEarningsHistory(c.Request.Context(), driverID.(uuid.UUID), period, limit, offset)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get earnings history")
		return
	}

	common.SuccessResponse(c, resp)
}

// GetBalance returns unpaid balance
func (h *TestableHandler) GetBalance(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	balance, err := h.service.GetUnpaidBalance(c.Request.Context(), driverID.(uuid.UUID))
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get balance")
		return
	}

	common.SuccessResponse(c, gin.H{
		"unpaid_balance": balance,
		"currency":       "USD",
	})
}

// RequestPayout requests a payout
func (h *TestableHandler) RequestPayout(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req RequestPayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	payout, err := h.service.RequestPayout(c.Request.Context(), driverID.(uuid.UUID), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to request payout")
		return
	}

	common.CreatedResponse(c, payout)
}

// GetPayoutHistory returns payout history
func (h *TestableHandler) GetPayoutHistory(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Default pagination
	limit := 20
	offset := 0

	resp, err := h.service.GetPayoutHistory(c.Request.Context(), driverID.(uuid.UUID), limit, offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get payout history")
		return
	}

	common.SuccessResponse(c, resp)
}

// AddBankAccount adds a bank account
func (h *TestableHandler) AddBankAccount(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req AddBankAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	account, err := h.service.AddBankAccount(c.Request.Context(), driverID.(uuid.UUID), &req)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to add bank account")
		return
	}

	common.CreatedResponse(c, account)
}

// GetBankAccounts returns all bank accounts
func (h *TestableHandler) GetBankAccounts(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	accounts, err := h.service.GetBankAccounts(c.Request.Context(), driverID.(uuid.UUID))
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get bank accounts")
		return
	}

	common.SuccessResponse(c, gin.H{
		"accounts": accounts,
		"count":    len(accounts),
	})
}

// DeleteBankAccount removes a bank account
func (h *TestableHandler) DeleteBankAccount(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	accountID, err := uuid.Parse(c.Param("accountId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account id")
		return
	}

	if err := h.service.DeleteBankAccount(c.Request.Context(), driverID.(uuid.UUID), accountID); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to delete bank account")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "bank account removed"})
}

// SetGoal sets an earning goal
func (h *TestableHandler) SetGoal(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SetEarningGoalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	goal, err := h.service.SetEarningGoal(c.Request.Context(), driverID.(uuid.UUID), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to set earning goal")
		return
	}

	common.CreatedResponse(c, goal)
}

// GetGoals returns earning goals with progress
func (h *TestableHandler) GetGoals(c *gin.Context) {
	driverID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	goals, err := h.service.GetEarningGoals(c.Request.Context(), driverID.(uuid.UUID))
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get earning goals")
		return
	}

	common.SuccessResponse(c, gin.H{"goals": goals})
}

// AdminRecordBonus records a bonus for a driver (admin only)
func (h *TestableHandler) AdminRecordBonus(c *gin.Context) {
	var req struct {
		DriverID    uuid.UUID `json:"driver_id" binding:"required"`
		Amount      float64   `json:"amount" binding:"required"`
		Description string    `json:"description" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	earning, err := h.service.RecordBonus(c.Request.Context(), req.DriverID, req.Amount, req.Description)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to record bonus")
		return
	}

	common.CreatedResponse(c, earning)
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

func setDriverContext(c *gin.Context, driverID uuid.UUID) {
	c.Set("user_id", driverID)
	c.Set("user_role", models.RoleDriver)
	c.Set("user_email", "driver@example.com")
}

func setAdminContext(c *gin.Context, adminID uuid.UUID) {
	c.Set("user_id", adminID)
	c.Set("user_role", models.RoleAdmin)
	c.Set("user_email", "admin@example.com")
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

// Helper functions for creating test data
func createTestEarningsSummary(driverID uuid.UUID) *EarningsSummary {
	return &EarningsSummary{
		DriverID:         driverID,
		Period:           "today",
		PeriodStart:      time.Now().Truncate(24 * time.Hour),
		PeriodEnd:        time.Now(),
		GrossEarnings:    150.00,
		TotalCommission:  30.00,
		NetEarnings:      120.00,
		TipEarnings:      15.00,
		BonusEarnings:    10.00,
		SurgeEarnings:    5.00,
		WaitTimeEarnings: 2.00,
		DeliveryEarnings: 20.00,
		RideCount:        10,
		DeliveryCount:    3,
		OnlineHours:      6.5,
		EarningsPerHour:  18.46,
		Currency:         "USD",
		Breakdown: []EarningBreakdown{
			{Type: EarningTypeRideFare, Amount: 100.00, Count: 10},
			{Type: EarningTypeTip, Amount: 15.00, Count: 5},
		},
	}
}

func createTestDailyEarnings() []DailyEarning {
	return []DailyEarning{
		{
			Date:        "2024-01-15",
			GrossAmount: 150.00,
			NetAmount:   120.00,
			Commission:  30.00,
			Tips:        15.00,
			RideCount:   10,
			OnlineHours: 6.5,
		},
		{
			Date:        "2024-01-14",
			GrossAmount: 130.00,
			NetAmount:   104.00,
			Commission:  26.00,
			Tips:        10.00,
			RideCount:   8,
			OnlineHours: 5.5,
		},
	}
}

func createTestEarningsHistoryResponse() *EarningsHistoryResponse {
	rideID := uuid.New()
	return &EarningsHistoryResponse{
		Earnings: []DriverEarning{
			{
				ID:          uuid.New(),
				DriverID:    uuid.New(),
				RideID:      &rideID,
				Type:        EarningTypeRideFare,
				GrossAmount: 25.00,
				Commission:  5.00,
				NetAmount:   20.00,
				Currency:    "USD",
				Description: "Ride fare",
				IsPaidOut:   false,
				CreatedAt:   time.Now(),
			},
		},
		Total:    1,
		Page:     1,
		PageSize: 20,
	}
}

func createTestPayoutHistoryResponse() *PayoutHistoryResponse {
	return &PayoutHistoryResponse{
		Payouts: []DriverPayout{
			{
				ID:           uuid.New(),
				DriverID:     uuid.New(),
				Amount:       100.00,
				Currency:     "USD",
				Method:       PayoutMethodBankTransfer,
				Status:       PayoutStatusCompleted,
				Reference:    "PAY-ABC123",
				EarningCount: 10,
				PeriodStart:  time.Now().AddDate(0, 0, -7),
				PeriodEnd:    time.Now(),
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
		},
		Total:    1,
		Page:     1,
		PageSize: 20,
	}
}

func createTestDriverPayout(driverID uuid.UUID) *DriverPayout {
	now := time.Now()
	return &DriverPayout{
		ID:           uuid.New(),
		DriverID:     driverID,
		Amount:       100.00,
		Currency:     "USD",
		Method:       PayoutMethodBankTransfer,
		Status:       PayoutStatusPending,
		Reference:    "PAY-ABC123",
		EarningCount: 5,
		PeriodStart:  now.AddDate(0, 0, -7),
		PeriodEnd:    now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func createTestBankAccount(driverID uuid.UUID) *DriverBankAccount {
	now := time.Now()
	return &DriverBankAccount{
		ID:            uuid.New(),
		DriverID:      driverID,
		BankName:      "Test Bank",
		AccountHolder: "John Doe",
		AccountNumber: "****1234",
		Currency:      "USD",
		IsPrimary:     true,
		IsVerified:    true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func createTestEarningGoal(driverID uuid.UUID) *EarningGoal {
	now := time.Now()
	return &EarningGoal{
		ID:            uuid.New(),
		DriverID:      driverID,
		TargetAmount:  500.00,
		Period:        "weekly",
		CurrentAmount: 200.00,
		Currency:      "USD",
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func createTestEarningGoalStatus(driverID uuid.UUID) []EarningGoalStatus {
	goal := createTestEarningGoal(driverID)
	return []EarningGoalStatus{
		{
			Goal:          goal,
			CurrentAmount: 200.00,
			ProgressPct:   40.00,
			Remaining:     300.00,
			OnTrack:       true,
		},
	}
}

func createTestDriverEarning(driverID uuid.UUID) *DriverEarning {
	return &DriverEarning{
		ID:          uuid.New(),
		DriverID:    driverID,
		Type:        EarningTypeBonus,
		GrossAmount: 50.00,
		Commission:  0,
		NetAmount:   50.00,
		Currency:    "USD",
		Description: "Performance bonus",
		IsPaidOut:   false,
		CreatedAt:   time.Now(),
	}
}

// ============================================================================
// GetSummary Handler Tests
// ============================================================================

func TestHandler_GetSummary_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	expectedSummary := createTestEarningsSummary(driverID)

	mockService.On("GetEarningsSummary", mock.Anything, driverID, "today").Return(expectedSummary, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/summary?period=today", nil)
	c.Request.URL.RawQuery = "period=today"
	setDriverContext(c, driverID)

	handler.GetSummary(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetSummary_DefaultPeriod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	expectedSummary := createTestEarningsSummary(driverID)

	mockService.On("GetEarningsSummary", mock.Anything, driverID, "today").Return(expectedSummary, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/summary", nil)
	setDriverContext(c, driverID)

	handler.GetSummary(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetSummary_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/summary", nil)
	// Don't set user context

	handler.GetSummary(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_GetSummary_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	mockService.On("GetEarningsSummary", mock.Anything, driverID, "today").Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/summary", nil)
	setDriverContext(c, driverID)

	handler.GetSummary(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetSummary_AppError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	mockService.On("GetEarningsSummary", mock.Anything, driverID, "invalid_period").Return(nil, common.NewBadRequestError("invalid period", nil))

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/summary?period=invalid_period", nil)
	c.Request.URL.RawQuery = "period=invalid_period"
	setDriverContext(c, driverID)

	handler.GetSummary(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetDailyBreakdown Handler Tests
// ============================================================================

func TestHandler_GetDailyBreakdown_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	expectedDaily := createTestDailyEarnings()

	mockService.On("GetDailyEarnings", mock.Anything, driverID, "this_week").Return(expectedDaily, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/daily?period=this_week", nil)
	c.Request.URL.RawQuery = "period=this_week"
	setDriverContext(c, driverID)

	handler.GetDailyBreakdown(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["daily"])
	assert.Equal(t, "this_week", data["period"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetDailyBreakdown_DefaultPeriod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	expectedDaily := createTestDailyEarnings()

	mockService.On("GetDailyEarnings", mock.Anything, driverID, "this_week").Return(expectedDaily, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/daily", nil)
	setDriverContext(c, driverID)

	handler.GetDailyBreakdown(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetDailyBreakdown_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/daily", nil)

	handler.GetDailyBreakdown(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetDailyBreakdown_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	mockService.On("GetDailyEarnings", mock.Anything, driverID, "this_week").Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/daily", nil)
	setDriverContext(c, driverID)

	handler.GetDailyBreakdown(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetDailyBreakdown_AppError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	mockService.On("GetDailyEarnings", mock.Anything, driverID, "invalid").Return(nil, common.NewBadRequestError("invalid period", nil))

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/daily?period=invalid", nil)
	c.Request.URL.RawQuery = "period=invalid"
	setDriverContext(c, driverID)

	handler.GetDailyBreakdown(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetHistory Handler Tests
// ============================================================================

func TestHandler_GetHistory_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	expectedResp := createTestEarningsHistoryResponse()

	mockService.On("GetEarningsHistory", mock.Anything, driverID, "this_month", 20, 0).Return(expectedResp, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/history", nil)
	setDriverContext(c, driverID)

	handler.GetHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetHistory_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/history", nil)

	handler.GetHistory(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetHistory_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	mockService.On("GetEarningsHistory", mock.Anything, driverID, "this_month", 20, 0).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/history", nil)
	setDriverContext(c, driverID)

	handler.GetHistory(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetHistory_AppError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	mockService.On("GetEarningsHistory", mock.Anything, driverID, "this_month", 20, 0).Return(nil, common.NewBadRequestError("invalid period", nil))

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/history", nil)
	setDriverContext(c, driverID)

	handler.GetHistory(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetBalance Handler Tests
// ============================================================================

func TestHandler_GetBalance_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	mockService.On("GetUnpaidBalance", mock.Anything, driverID).Return(125.50, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/balance", nil)
	setDriverContext(c, driverID)

	handler.GetBalance(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, 125.50, data["unpaid_balance"])
	assert.Equal(t, "USD", data["currency"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetBalance_ZeroBalance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	mockService.On("GetUnpaidBalance", mock.Anything, driverID).Return(0.0, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/balance", nil)
	setDriverContext(c, driverID)

	handler.GetBalance(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, 0.0, data["unpaid_balance"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetBalance_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/balance", nil)

	handler.GetBalance(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetBalance_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	mockService.On("GetUnpaidBalance", mock.Anything, driverID).Return(0.0, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/balance", nil)
	setDriverContext(c, driverID)

	handler.GetBalance(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// RequestPayout Handler Tests
// ============================================================================

func TestHandler_RequestPayout_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	expectedPayout := createTestDriverPayout(driverID)

	reqBody := RequestPayoutRequest{
		Method: PayoutMethodBankTransfer,
	}

	mockService.On("RequestPayout", mock.Anything, driverID, mock.AnythingOfType("*earnings.RequestPayoutRequest")).Return(expectedPayout, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/earnings/payouts", reqBody)
	setDriverContext(c, driverID)

	handler.RequestPayout(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_RequestPayout_WithBankAccountID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	bankAccountID := uuid.New()
	expectedPayout := createTestDriverPayout(driverID)

	reqBody := RequestPayoutRequest{
		Method:        PayoutMethodBankTransfer,
		BankAccountID: &bankAccountID,
	}

	mockService.On("RequestPayout", mock.Anything, driverID, mock.AnythingOfType("*earnings.RequestPayoutRequest")).Return(expectedPayout, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/earnings/payouts", reqBody)
	setDriverContext(c, driverID)

	handler.RequestPayout(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_RequestPayout_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	reqBody := RequestPayoutRequest{
		Method: PayoutMethodBankTransfer,
	}

	c, w := setupTestContext("POST", "/api/v1/driver/earnings/payouts", reqBody)

	handler.RequestPayout(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_RequestPayout_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/earnings/payouts", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/driver/earnings/payouts", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setDriverContext(c, driverID)

	handler.RequestPayout(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RequestPayout_MissingMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/driver/earnings/payouts", reqBody)
	setDriverContext(c, driverID)

	handler.RequestPayout(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RequestPayout_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	reqBody := RequestPayoutRequest{
		Method: PayoutMethodBankTransfer,
	}

	mockService.On("RequestPayout", mock.Anything, driverID, mock.AnythingOfType("*earnings.RequestPayoutRequest")).Return(nil, errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/driver/earnings/payouts", reqBody)
	setDriverContext(c, driverID)

	handler.RequestPayout(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_RequestPayout_AppError_InsufficientBalance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	reqBody := RequestPayoutRequest{
		Method: PayoutMethodBankTransfer,
	}

	mockService.On("RequestPayout", mock.Anything, driverID, mock.AnythingOfType("*earnings.RequestPayoutRequest")).Return(nil, common.NewBadRequestError("minimum payout amount is 5.00, current balance: 2.50", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/earnings/payouts", reqBody)
	setDriverContext(c, driverID)

	handler.RequestPayout(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetPayoutHistory Handler Tests
// ============================================================================

func TestHandler_GetPayoutHistory_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	expectedResp := createTestPayoutHistoryResponse()

	mockService.On("GetPayoutHistory", mock.Anything, driverID, 20, 0).Return(expectedResp, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/payouts", nil)
	setDriverContext(c, driverID)

	handler.GetPayoutHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetPayoutHistory_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	expectedResp := &PayoutHistoryResponse{
		Payouts:  []DriverPayout{},
		Total:    0,
		Page:     1,
		PageSize: 20,
	}

	mockService.On("GetPayoutHistory", mock.Anything, driverID, 20, 0).Return(expectedResp, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/payouts", nil)
	setDriverContext(c, driverID)

	handler.GetPayoutHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetPayoutHistory_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/payouts", nil)

	handler.GetPayoutHistory(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetPayoutHistory_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	mockService.On("GetPayoutHistory", mock.Anything, driverID, 20, 0).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/payouts", nil)
	setDriverContext(c, driverID)

	handler.GetPayoutHistory(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// AddBankAccount Handler Tests
// ============================================================================

func TestHandler_AddBankAccount_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	expectedAccount := createTestBankAccount(driverID)

	reqBody := AddBankAccountRequest{
		BankName:      "Test Bank",
		AccountHolder: "John Doe",
		AccountNumber: "1234567890",
		Currency:      "USD",
		IsPrimary:     true,
	}

	mockService.On("AddBankAccount", mock.Anything, driverID, mock.AnythingOfType("*earnings.AddBankAccountRequest")).Return(expectedAccount, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/earnings/bank-accounts", reqBody)
	setDriverContext(c, driverID)

	handler.AddBankAccount(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_AddBankAccount_WithIBAN(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	expectedAccount := createTestBankAccount(driverID)
	iban := "DE89370400440532013000"
	expectedAccount.IBAN = &iban

	reqBody := AddBankAccountRequest{
		BankName:      "German Bank",
		AccountHolder: "Hans Mueller",
		AccountNumber: "1234567890",
		IBAN:          &iban,
		Currency:      "EUR",
	}

	mockService.On("AddBankAccount", mock.Anything, driverID, mock.AnythingOfType("*earnings.AddBankAccountRequest")).Return(expectedAccount, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/earnings/bank-accounts", reqBody)
	setDriverContext(c, driverID)

	handler.AddBankAccount(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_AddBankAccount_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	reqBody := AddBankAccountRequest{
		BankName:      "Test Bank",
		AccountHolder: "John Doe",
		AccountNumber: "1234567890",
	}

	c, w := setupTestContext("POST", "/api/v1/driver/earnings/bank-accounts", reqBody)

	handler.AddBankAccount(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_AddBankAccount_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/earnings/bank-accounts", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/driver/earnings/bank-accounts", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setDriverContext(c, driverID)

	handler.AddBankAccount(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AddBankAccount_MissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "missing bank_name",
			body: map[string]interface{}{
				"account_holder": "John Doe",
				"account_number": "1234567890",
			},
		},
		{
			name: "missing account_holder",
			body: map[string]interface{}{
				"bank_name":      "Test Bank",
				"account_number": "1234567890",
			},
		},
		{
			name: "missing account_number",
			body: map[string]interface{}{
				"bank_name":      "Test Bank",
				"account_holder": "John Doe",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockEarningsService)
			handler := NewTestableHandler(mockService)

			driverID := uuid.New()

			c, w := setupTestContext("POST", "/api/v1/driver/earnings/bank-accounts", tt.body)
			setDriverContext(c, driverID)

			handler.AddBankAccount(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandler_AddBankAccount_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	reqBody := AddBankAccountRequest{
		BankName:      "Test Bank",
		AccountHolder: "John Doe",
		AccountNumber: "1234567890",
	}

	mockService.On("AddBankAccount", mock.Anything, driverID, mock.AnythingOfType("*earnings.AddBankAccountRequest")).Return(nil, errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/driver/earnings/bank-accounts", reqBody)
	setDriverContext(c, driverID)

	handler.AddBankAccount(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetBankAccounts Handler Tests
// ============================================================================

func TestHandler_GetBankAccounts_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	expectedAccounts := []DriverBankAccount{*createTestBankAccount(driverID)}

	mockService.On("GetBankAccounts", mock.Anything, driverID).Return(expectedAccounts, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/bank-accounts", nil)
	setDriverContext(c, driverID)

	handler.GetBankAccounts(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["accounts"])
	assert.Equal(t, float64(1), data["count"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetBankAccounts_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	mockService.On("GetBankAccounts", mock.Anything, driverID).Return([]DriverBankAccount{}, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/bank-accounts", nil)
	setDriverContext(c, driverID)

	handler.GetBankAccounts(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["count"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetBankAccounts_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/bank-accounts", nil)

	handler.GetBankAccounts(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetBankAccounts_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	mockService.On("GetBankAccounts", mock.Anything, driverID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/bank-accounts", nil)
	setDriverContext(c, driverID)

	handler.GetBankAccounts(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// DeleteBankAccount Handler Tests
// ============================================================================

func TestHandler_DeleteBankAccount_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	accountID := uuid.New()

	mockService.On("DeleteBankAccount", mock.Anything, driverID, accountID).Return(nil)

	c, w := setupTestContext("DELETE", "/api/v1/driver/earnings/bank-accounts/"+accountID.String(), nil)
	c.Params = gin.Params{{Key: "accountId", Value: accountID.String()}}
	setDriverContext(c, driverID)

	handler.DeleteBankAccount(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "bank account removed", data["message"])
	mockService.AssertExpectations(t)
}

func TestHandler_DeleteBankAccount_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/driver/earnings/bank-accounts/"+accountID.String(), nil)
	c.Params = gin.Params{{Key: "accountId", Value: accountID.String()}}

	handler.DeleteBankAccount(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_DeleteBankAccount_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/driver/earnings/bank-accounts/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "accountId", Value: "invalid-uuid"}}
	setDriverContext(c, driverID)

	handler.DeleteBankAccount(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid account id")
}

func TestHandler_DeleteBankAccount_EmptyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/driver/earnings/bank-accounts/", nil)
	c.Params = gin.Params{{Key: "accountId", Value: ""}}
	setDriverContext(c, driverID)

	handler.DeleteBankAccount(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DeleteBankAccount_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	accountID := uuid.New()

	mockService.On("DeleteBankAccount", mock.Anything, driverID, accountID).Return(errors.New("database error"))

	c, w := setupTestContext("DELETE", "/api/v1/driver/earnings/bank-accounts/"+accountID.String(), nil)
	c.Params = gin.Params{{Key: "accountId", Value: accountID.String()}}
	setDriverContext(c, driverID)

	handler.DeleteBankAccount(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// SetGoal Handler Tests
// ============================================================================

func TestHandler_SetGoal_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	expectedGoal := createTestEarningGoal(driverID)

	reqBody := SetEarningGoalRequest{
		TargetAmount: 500.00,
		Period:       "weekly",
	}

	mockService.On("SetEarningGoal", mock.Anything, driverID, mock.AnythingOfType("*earnings.SetEarningGoalRequest")).Return(expectedGoal, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/earnings/goals", reqBody)
	setDriverContext(c, driverID)

	handler.SetGoal(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_SetGoal_DailyPeriod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	expectedGoal := createTestEarningGoal(driverID)
	expectedGoal.Period = "daily"

	reqBody := SetEarningGoalRequest{
		TargetAmount: 100.00,
		Period:       "daily",
	}

	mockService.On("SetEarningGoal", mock.Anything, driverID, mock.AnythingOfType("*earnings.SetEarningGoalRequest")).Return(expectedGoal, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/earnings/goals", reqBody)
	setDriverContext(c, driverID)

	handler.SetGoal(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SetGoal_MonthlyPeriod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	expectedGoal := createTestEarningGoal(driverID)
	expectedGoal.Period = "monthly"
	expectedGoal.TargetAmount = 2000.00

	reqBody := SetEarningGoalRequest{
		TargetAmount: 2000.00,
		Period:       "monthly",
	}

	mockService.On("SetEarningGoal", mock.Anything, driverID, mock.AnythingOfType("*earnings.SetEarningGoalRequest")).Return(expectedGoal, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/earnings/goals", reqBody)
	setDriverContext(c, driverID)

	handler.SetGoal(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SetGoal_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	reqBody := SetEarningGoalRequest{
		TargetAmount: 500.00,
		Period:       "weekly",
	}

	c, w := setupTestContext("POST", "/api/v1/driver/earnings/goals", reqBody)

	handler.SetGoal(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_SetGoal_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/earnings/goals", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/driver/earnings/goals", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setDriverContext(c, driverID)

	handler.SetGoal(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SetGoal_MissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "missing target_amount",
			body: map[string]interface{}{
				"period": "weekly",
			},
		},
		{
			name: "missing period",
			body: map[string]interface{}{
				"target_amount": 500.00,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockEarningsService)
			handler := NewTestableHandler(mockService)

			driverID := uuid.New()

			c, w := setupTestContext("POST", "/api/v1/driver/earnings/goals", tt.body)
			setDriverContext(c, driverID)

			handler.SetGoal(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandler_SetGoal_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	reqBody := SetEarningGoalRequest{
		TargetAmount: 500.00,
		Period:       "weekly",
	}

	mockService.On("SetEarningGoal", mock.Anything, driverID, mock.AnythingOfType("*earnings.SetEarningGoalRequest")).Return(nil, errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/driver/earnings/goals", reqBody)
	setDriverContext(c, driverID)

	handler.SetGoal(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SetGoal_AppError_InvalidPeriod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	reqBody := SetEarningGoalRequest{
		TargetAmount: 500.00,
		Period:       "invalid",
	}

	mockService.On("SetEarningGoal", mock.Anything, driverID, mock.AnythingOfType("*earnings.SetEarningGoalRequest")).Return(nil, common.NewBadRequestError("period must be daily, weekly, or monthly", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/earnings/goals", reqBody)
	setDriverContext(c, driverID)

	handler.SetGoal(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SetGoal_AppError_NegativeAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	reqBody := SetEarningGoalRequest{
		TargetAmount: -100.00,
		Period:       "weekly",
	}

	mockService.On("SetEarningGoal", mock.Anything, driverID, mock.AnythingOfType("*earnings.SetEarningGoalRequest")).Return(nil, common.NewBadRequestError("target amount must be positive", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/earnings/goals", reqBody)
	setDriverContext(c, driverID)

	handler.SetGoal(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetGoals Handler Tests
// ============================================================================

func TestHandler_GetGoals_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	expectedGoals := createTestEarningGoalStatus(driverID)

	mockService.On("GetEarningGoals", mock.Anything, driverID).Return(expectedGoals, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/goals", nil)
	setDriverContext(c, driverID)

	handler.GetGoals(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["goals"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetGoals_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	mockService.On("GetEarningGoals", mock.Anything, driverID).Return([]EarningGoalStatus{}, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/goals", nil)
	setDriverContext(c, driverID)

	handler.GetGoals(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetGoals_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/goals", nil)

	handler.GetGoals(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetGoals_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	mockService.On("GetEarningGoals", mock.Anything, driverID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/goals", nil)
	setDriverContext(c, driverID)

	handler.GetGoals(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// AdminRecordBonus Handler Tests
// ============================================================================

func TestHandler_AdminRecordBonus_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	adminID := uuid.New()
	expectedEarning := createTestDriverEarning(driverID)

	reqBody := map[string]interface{}{
		"driver_id":   driverID.String(),
		"amount":      50.00,
		"description": "Performance bonus",
	}

	mockService.On("RecordBonus", mock.Anything, driverID, 50.00, "Performance bonus").Return(expectedEarning, nil)

	c, w := setupTestContext("POST", "/api/v1/admin/earnings/bonus", reqBody)
	setAdminContext(c, adminID)

	handler.AdminRecordBonus(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_AdminRecordBonus_LargeBonus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	adminID := uuid.New()
	expectedEarning := createTestDriverEarning(driverID)
	expectedEarning.GrossAmount = 1000.00
	expectedEarning.NetAmount = 1000.00

	reqBody := map[string]interface{}{
		"driver_id":   driverID.String(),
		"amount":      1000.00,
		"description": "Yearly bonus",
	}

	mockService.On("RecordBonus", mock.Anything, driverID, 1000.00, "Yearly bonus").Return(expectedEarning, nil)

	c, w := setupTestContext("POST", "/api/v1/admin/earnings/bonus", reqBody)
	setAdminContext(c, adminID)

	handler.AdminRecordBonus(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_AdminRecordBonus_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	adminID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/admin/earnings/bonus", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/earnings/bonus", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setAdminContext(c, adminID)

	handler.AdminRecordBonus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminRecordBonus_MissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "missing driver_id",
			body: map[string]interface{}{
				"amount":      50.00,
				"description": "Bonus",
			},
		},
		{
			name: "missing amount",
			body: map[string]interface{}{
				"driver_id":   uuid.New().String(),
				"description": "Bonus",
			},
		},
		{
			name: "missing description",
			body: map[string]interface{}{
				"driver_id": uuid.New().String(),
				"amount":    50.00,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockEarningsService)
			handler := NewTestableHandler(mockService)

			adminID := uuid.New()

			c, w := setupTestContext("POST", "/api/v1/admin/earnings/bonus", tt.body)
			setAdminContext(c, adminID)

			handler.AdminRecordBonus(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandler_AdminRecordBonus_InvalidDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	adminID := uuid.New()

	reqBody := map[string]interface{}{
		"driver_id":   "invalid-uuid",
		"amount":      50.00,
		"description": "Bonus",
	}

	c, w := setupTestContext("POST", "/api/v1/admin/earnings/bonus", reqBody)
	setAdminContext(c, adminID)

	handler.AdminRecordBonus(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminRecordBonus_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	adminID := uuid.New()

	reqBody := map[string]interface{}{
		"driver_id":   driverID.String(),
		"amount":      50.00,
		"description": "Bonus",
	}

	mockService.On("RecordBonus", mock.Anything, driverID, 50.00, "Bonus").Return(nil, errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/admin/earnings/bonus", reqBody)
	setAdminContext(c, adminID)

	handler.AdminRecordBonus(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// Edge Cases and UUID Validation Tests
// ============================================================================

func TestHandler_DeleteBankAccount_UUIDFormats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		uuid           string
		expectedStatus int
	}{
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
			name:           "empty uuid",
			uuid:           "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "partial valid format",
			uuid:           "550e8400-e29b-41d4",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "too long uuid",
			uuid:           "550e8400-e29b-41d4-a716-446655440000-extra",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockEarningsService)
			handler := NewTestableHandler(mockService)

			driverID := uuid.New()

			c, w := setupTestContext("DELETE", "/api/v1/driver/earnings/bank-accounts/test-id", nil)
			c.Params = gin.Params{{Key: "accountId", Value: tt.uuid}}
			setDriverContext(c, driverID)

			handler.DeleteBankAccount(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// Table-Driven Tests for Periods
// ============================================================================

func TestHandler_GetSummary_AllPeriods(t *testing.T) {
	gin.SetMode(gin.TestMode)

	periods := []string{"today", "yesterday", "this_week", "last_week", "this_month", "last_month"}

	for _, period := range periods {
		t.Run(period, func(t *testing.T) {
			mockService := new(MockEarningsService)
			handler := NewTestableHandler(mockService)

			driverID := uuid.New()
			expectedSummary := createTestEarningsSummary(driverID)
			expectedSummary.Period = period

			mockService.On("GetEarningsSummary", mock.Anything, driverID, period).Return(expectedSummary, nil)

			c, w := setupTestContext("GET", "/api/v1/driver/earnings/summary?period="+period, nil)
			c.Request.URL.RawQuery = "period=" + period
			setDriverContext(c, driverID)

			handler.GetSummary(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_SetGoal_AllPeriods(t *testing.T) {
	gin.SetMode(gin.TestMode)

	periods := []string{"daily", "weekly", "monthly"}

	for _, period := range periods {
		t.Run(period, func(t *testing.T) {
			mockService := new(MockEarningsService)
			handler := NewTestableHandler(mockService)

			driverID := uuid.New()
			expectedGoal := createTestEarningGoal(driverID)
			expectedGoal.Period = period

			reqBody := SetEarningGoalRequest{
				TargetAmount: 500.00,
				Period:       period,
			}

			mockService.On("SetEarningGoal", mock.Anything, driverID, mock.AnythingOfType("*earnings.SetEarningGoalRequest")).Return(expectedGoal, nil)

			c, w := setupTestContext("POST", "/api/v1/driver/earnings/goals", reqBody)
			setDriverContext(c, driverID)

			handler.SetGoal(c)

			assert.Equal(t, http.StatusCreated, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_GetSummary_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	expectedSummary := createTestEarningsSummary(driverID)

	mockService.On("GetEarningsSummary", mock.Anything, driverID, "today").Return(expectedSummary, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/summary", nil)
	setDriverContext(c, driverID)

	handler.GetSummary(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	// Verify response structure
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["period"])
	assert.NotNil(t, data["gross_earnings"])
	assert.NotNil(t, data["net_earnings"])
	assert.NotNil(t, data["currency"])

	mockService.AssertExpectations(t)
}

func TestHandler_GetBalance_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	mockService.On("GetUnpaidBalance", mock.Anything, driverID).Return(100.50, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/balance", nil)
	setDriverContext(c, driverID)

	handler.GetBalance(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, 100.50, data["unpaid_balance"])
	assert.Equal(t, "USD", data["currency"])

	mockService.AssertExpectations(t)
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	mockService.On("GetEarningsSummary", mock.Anything, driverID, "today").Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/summary", nil)
	setDriverContext(c, driverID)

	handler.GetSummary(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)

	// Verify error response structure
	assert.False(t, response["success"].(bool))
	assert.NotNil(t, response["error"])

	errorInfo := response["error"].(map[string]interface{})
	assert.NotNil(t, errorInfo["code"])
	assert.NotNil(t, errorInfo["message"])

	mockService.AssertExpectations(t)
}

// ============================================================================
// Payout Method Tests
// ============================================================================

func TestHandler_RequestPayout_AllMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)

	methods := []PayoutMethod{PayoutMethodBankTransfer, PayoutMethodInstantPay, PayoutMethodWallet}

	for _, method := range methods {
		t.Run(string(method), func(t *testing.T) {
			mockService := new(MockEarningsService)
			handler := NewTestableHandler(mockService)

			driverID := uuid.New()
			expectedPayout := createTestDriverPayout(driverID)
			expectedPayout.Method = method

			reqBody := RequestPayoutRequest{
				Method: method,
			}

			mockService.On("RequestPayout", mock.Anything, driverID, mock.AnythingOfType("*earnings.RequestPayoutRequest")).Return(expectedPayout, nil)

			c, w := setupTestContext("POST", "/api/v1/driver/earnings/payouts", reqBody)
			setDriverContext(c, driverID)

			handler.RequestPayout(c)

			assert.Equal(t, http.StatusCreated, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Concurrent Request Tests
// ============================================================================

func TestHandler_ConcurrentRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	expectedSummary := createTestEarningsSummary(driverID)

	mockService.On("GetEarningsSummary", mock.Anything, driverID, "today").Return(expectedSummary, nil).Times(5)

	for i := 0; i < 5; i++ {
		c, w := setupTestContext("GET", "/api/v1/driver/earnings/summary", nil)
		setDriverContext(c, driverID)

		handler.GetSummary(c)

		assert.Equal(t, http.StatusOK, w.Code)
	}

	mockService.AssertExpectations(t)
}

// ============================================================================
// Multiple Bank Accounts Tests
// ============================================================================

func TestHandler_GetBankAccounts_MultipleAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()
	account1 := createTestBankAccount(driverID)
	account2 := createTestBankAccount(driverID)
	account2.ID = uuid.New()
	account2.BankName = "Second Bank"
	account2.IsPrimary = false

	expectedAccounts := []DriverBankAccount{*account1, *account2}

	mockService.On("GetBankAccounts", mock.Anything, driverID).Return(expectedAccounts, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/bank-accounts", nil)
	setDriverContext(c, driverID)

	handler.GetBankAccounts(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["count"])
	mockService.AssertExpectations(t)
}

// ============================================================================
// Multiple Goals Tests
// ============================================================================

func TestHandler_GetGoals_MultipleGoals(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockEarningsService)
	handler := NewTestableHandler(mockService)

	driverID := uuid.New()

	dailyGoal := createTestEarningGoal(driverID)
	dailyGoal.Period = "daily"
	dailyGoal.TargetAmount = 100.00

	weeklyGoal := createTestEarningGoal(driverID)
	weeklyGoal.ID = uuid.New()
	weeklyGoal.Period = "weekly"
	weeklyGoal.TargetAmount = 500.00

	expectedGoals := []EarningGoalStatus{
		{
			Goal:          dailyGoal,
			CurrentAmount: 50.00,
			ProgressPct:   50.00,
			Remaining:     50.00,
			OnTrack:       true,
		},
		{
			Goal:          weeklyGoal,
			CurrentAmount: 200.00,
			ProgressPct:   40.00,
			Remaining:     300.00,
			OnTrack:       false,
		},
	}

	mockService.On("GetEarningGoals", mock.Anything, driverID).Return(expectedGoals, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/earnings/goals", nil)
	setDriverContext(c, driverID)

	handler.GetGoals(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	goals := data["goals"].([]interface{})
	assert.Equal(t, 2, len(goals))
	mockService.AssertExpectations(t)
}
