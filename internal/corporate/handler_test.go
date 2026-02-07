package corporate

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
// SERVICE INTERFACE FOR TESTING
// ============================================================================

// ServiceInterface defines the interface for the corporate service used by the handler
type ServiceInterface interface {
	CreateAccount(ctx context.Context, req *CreateAccountRequest) (*CorporateAccount, error)
	GetAccount(ctx context.Context, accountID uuid.UUID) (*CorporateAccount, error)
	UpdateAccount(ctx context.Context, account *CorporateAccount) error
	ActivateAccount(ctx context.Context, accountID uuid.UUID) error
	SuspendAccount(ctx context.Context, accountID uuid.UUID, reason string) error
	ListAccounts(ctx context.Context, status *AccountStatus, limit, offset int) ([]*CorporateAccount, error)
	CreateDepartment(ctx context.Context, accountID uuid.UUID, name string, code *string, managerID *uuid.UUID, budgetMonthly *float64) (*Department, error)
	ListDepartments(ctx context.Context, accountID uuid.UUID) ([]*Department, error)
	InviteEmployee(ctx context.Context, accountID uuid.UUID, req *InviteEmployeeRequest) (*CorporateEmployee, error)
	GetEmployeeByUserID(ctx context.Context, userID uuid.UUID) (*CorporateEmployee, error)
	ListEmployees(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*CorporateEmployee, error)
	ListCorporateRides(ctx context.Context, accountID uuid.UUID, employeeID *uuid.UUID, startDate, endDate time.Time, limit, offset int) ([]*CorporateRide, error)
	GetPendingApprovals(ctx context.Context, accountID uuid.UUID, approverID *uuid.UUID) ([]*CorporateRide, error)
	ApproveRide(ctx context.Context, rideID, approverID uuid.UUID, approved bool) error
	ListInvoices(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*CorporateInvoice, error)
	GenerateInvoice(ctx context.Context, accountID uuid.UUID, periodStart, periodEnd time.Time) (*CorporateInvoice, error)
	CreatePolicy(ctx context.Context, accountID uuid.UUID, name string, policyType PolicyType, rules PolicyRules, departmentID *uuid.UUID) (*RidePolicy, error)
	CheckPolicies(ctx context.Context, accountID uuid.UUID, employeeID uuid.UUID, req *BookCorporateRideRequest, estimatedFare float64) (*PolicyCheckResult, error)
	GetDashboard(ctx context.Context, accountID uuid.UUID) (*AccountDashboardResponse, error)
}

// MockCorporateService is a mock implementation of ServiceInterface
type MockCorporateService struct {
	mock.Mock
}

func (m *MockCorporateService) CreateAccount(ctx context.Context, req *CreateAccountRequest) (*CorporateAccount, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CorporateAccount), args.Error(1)
}

func (m *MockCorporateService) GetAccount(ctx context.Context, accountID uuid.UUID) (*CorporateAccount, error) {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CorporateAccount), args.Error(1)
}

func (m *MockCorporateService) UpdateAccount(ctx context.Context, account *CorporateAccount) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockCorporateService) ActivateAccount(ctx context.Context, accountID uuid.UUID) error {
	args := m.Called(ctx, accountID)
	return args.Error(0)
}

func (m *MockCorporateService) SuspendAccount(ctx context.Context, accountID uuid.UUID, reason string) error {
	args := m.Called(ctx, accountID, reason)
	return args.Error(0)
}

func (m *MockCorporateService) ListAccounts(ctx context.Context, status *AccountStatus, limit, offset int) ([]*CorporateAccount, error) {
	args := m.Called(ctx, status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*CorporateAccount), args.Error(1)
}

func (m *MockCorporateService) CreateDepartment(ctx context.Context, accountID uuid.UUID, name string, code *string, managerID *uuid.UUID, budgetMonthly *float64) (*Department, error) {
	args := m.Called(ctx, accountID, name, code, managerID, budgetMonthly)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Department), args.Error(1)
}

func (m *MockCorporateService) ListDepartments(ctx context.Context, accountID uuid.UUID) ([]*Department, error) {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Department), args.Error(1)
}

func (m *MockCorporateService) InviteEmployee(ctx context.Context, accountID uuid.UUID, req *InviteEmployeeRequest) (*CorporateEmployee, error) {
	args := m.Called(ctx, accountID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CorporateEmployee), args.Error(1)
}

func (m *MockCorporateService) GetEmployeeByUserID(ctx context.Context, userID uuid.UUID) (*CorporateEmployee, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CorporateEmployee), args.Error(1)
}

func (m *MockCorporateService) ListEmployees(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*CorporateEmployee, error) {
	args := m.Called(ctx, accountID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*CorporateEmployee), args.Error(1)
}

func (m *MockCorporateService) ListCorporateRides(ctx context.Context, accountID uuid.UUID, employeeID *uuid.UUID, startDate, endDate time.Time, limit, offset int) ([]*CorporateRide, error) {
	args := m.Called(ctx, accountID, employeeID, startDate, endDate, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*CorporateRide), args.Error(1)
}

func (m *MockCorporateService) GetPendingApprovals(ctx context.Context, accountID uuid.UUID, approverID *uuid.UUID) ([]*CorporateRide, error) {
	args := m.Called(ctx, accountID, approverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*CorporateRide), args.Error(1)
}

func (m *MockCorporateService) ApproveRide(ctx context.Context, rideID, approverID uuid.UUID, approved bool) error {
	args := m.Called(ctx, rideID, approverID, approved)
	return args.Error(0)
}

func (m *MockCorporateService) ListInvoices(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*CorporateInvoice, error) {
	args := m.Called(ctx, accountID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*CorporateInvoice), args.Error(1)
}

func (m *MockCorporateService) GenerateInvoice(ctx context.Context, accountID uuid.UUID, periodStart, periodEnd time.Time) (*CorporateInvoice, error) {
	args := m.Called(ctx, accountID, periodStart, periodEnd)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CorporateInvoice), args.Error(1)
}

func (m *MockCorporateService) CreatePolicy(ctx context.Context, accountID uuid.UUID, name string, policyType PolicyType, rules PolicyRules, departmentID *uuid.UUID) (*RidePolicy, error) {
	args := m.Called(ctx, accountID, name, policyType, rules, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RidePolicy), args.Error(1)
}

func (m *MockCorporateService) CheckPolicies(ctx context.Context, accountID uuid.UUID, employeeID uuid.UUID, req *BookCorporateRideRequest, estimatedFare float64) (*PolicyCheckResult, error) {
	args := m.Called(ctx, accountID, employeeID, req, estimatedFare)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PolicyCheckResult), args.Error(1)
}

func (m *MockCorporateService) GetDashboard(ctx context.Context, accountID uuid.UUID) (*AccountDashboardResponse, error) {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AccountDashboardResponse), args.Error(1)
}

// ============================================================================
// TESTABLE HANDLER (uses mock service interface)
// ============================================================================

// TestableHandler wraps handler methods for testing with mock service
type TestableHandler struct {
	service ServiceInterface
}

func NewTestableHandler(svc ServiceInterface) *TestableHandler {
	return &TestableHandler{service: svc}
}

// CreateAccount creates a new corporate account
func (h *TestableHandler) CreateAccount(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	account, err := h.service.CreateAccount(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create account")
		return
	}

	common.SuccessResponse(c, account)
}

// GetAccount gets a corporate account
func (h *TestableHandler) GetAccount(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	account, err := h.service.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "account not found")
		return
	}

	common.SuccessResponse(c, account)
}

// GetDashboard gets the corporate dashboard
func (h *TestableHandler) GetDashboard(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	dashboard, err := h.service.GetDashboard(c.Request.Context(), accountID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get dashboard")
		return
	}

	common.SuccessResponse(c, dashboard)
}

// ListAccounts lists corporate accounts (admin)
func (h *TestableHandler) ListAccounts(c *gin.Context) {
	limit := 20
	offset := 0

	var status *AccountStatus
	if s := c.Query("status"); s != "" {
		st := AccountStatus(s)
		status = &st
	}

	accounts, err := h.service.ListAccounts(c.Request.Context(), status, limit, offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list accounts")
		return
	}

	common.SuccessResponse(c, accounts)
}

// ActivateAccount activates a pending account
func (h *TestableHandler) ActivateAccount(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	if err := h.service.ActivateAccount(c.Request.Context(), accountID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to activate account")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Account activated successfully",
	})
}

// SuspendAccount suspends an account
func (h *TestableHandler) SuspendAccount(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.SuspendAccount(c.Request.Context(), accountID, req.Reason); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to suspend account")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Account suspended successfully",
	})
}

// CreateDepartment creates a new department
func (h *TestableHandler) CreateDepartment(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	var req struct {
		Name          string     `json:"name" binding:"required"`
		Code          *string    `json:"code,omitempty"`
		ManagerID     *uuid.UUID `json:"manager_id,omitempty"`
		BudgetMonthly *float64   `json:"budget_monthly,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	dept, err := h.service.CreateDepartment(c.Request.Context(), accountID, req.Name, req.Code, req.ManagerID, req.BudgetMonthly)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create department")
		return
	}

	common.SuccessResponse(c, dept)
}

// ListDepartments lists departments for an account
func (h *TestableHandler) ListDepartments(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	depts, err := h.service.ListDepartments(c.Request.Context(), accountID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list departments")
		return
	}

	common.SuccessResponse(c, depts)
}

// InviteEmployee invites an employee
func (h *TestableHandler) InviteEmployee(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	var req InviteEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	emp, err := h.service.InviteEmployee(c.Request.Context(), accountID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to invite employee")
		return
	}

	common.SuccessResponse(c, gin.H{
		"employee": emp,
		"message":  "Invitation sent successfully",
	})
}

// ListEmployees lists employees for an account
func (h *TestableHandler) ListEmployees(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	limit := 20
	offset := 0

	employees, err := h.service.ListEmployees(c.Request.Context(), accountID, limit, offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list employees")
		return
	}

	common.SuccessResponse(c, employees)
}

// GetMyProfile gets the current user's corporate profile
func (h *TestableHandler) GetMyProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	emp, err := h.service.GetEmployeeByUserID(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		common.SuccessResponse(c, gin.H{
			"is_corporate_user": false,
			"message":           "Not a corporate user",
		})
		return
	}

	account, _ := h.service.GetAccount(c.Request.Context(), emp.CorporateAccountID)

	common.SuccessResponse(c, gin.H{
		"is_corporate_user": true,
		"employee":          emp,
		"account":           account,
	})
}

// ListRides lists corporate rides
func (h *TestableHandler) ListRides(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	limit := 50
	offset := 0
	startDate := time.Now().AddDate(0, -1, 0)
	endDate := time.Now()

	var employeeID *uuid.UUID
	if s := c.Query("employee_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			employeeID = &id
		}
	}

	rides, err := h.service.ListCorporateRides(c.Request.Context(), accountID, employeeID, startDate, endDate, limit, offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list rides")
		return
	}

	common.SuccessResponse(c, gin.H{
		"rides": rides,
		"count": len(rides),
	})
}

// GetPendingApprovals gets rides pending approval
func (h *TestableHandler) GetPendingApprovals(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	rides, err := h.service.GetPendingApprovals(c.Request.Context(), accountID, nil)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get pending approvals")
		return
	}

	common.SuccessResponse(c, gin.H{
		"pending_approvals": rides,
		"count":             len(rides),
	})
}

// ApproveRide approves or rejects a ride
func (h *TestableHandler) ApproveRide(c *gin.Context) {
	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Approved bool `json:"approved"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.ApproveRide(c.Request.Context(), rideID, userID.(uuid.UUID), req.Approved); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to process approval")
		return
	}

	status := "approved"
	if !req.Approved {
		status = "rejected"
	}

	common.SuccessResponse(c, gin.H{
		"message": "Ride " + status + " successfully",
	})
}

// ListInvoices lists invoices for an account
func (h *TestableHandler) ListInvoices(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	limit := 20
	offset := 0

	invoices, err := h.service.ListInvoices(c.Request.Context(), accountID, limit, offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list invoices")
		return
	}

	common.SuccessResponse(c, invoices)
}

// GenerateInvoice generates an invoice for a period
func (h *TestableHandler) GenerateInvoice(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	var req struct {
		PeriodStart string `json:"period_start" binding:"required"`
		PeriodEnd   string `json:"period_end" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	periodStart, err := time.Parse("2006-01-02", req.PeriodStart)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid period_start format")
		return
	}

	periodEnd, err := time.Parse("2006-01-02", req.PeriodEnd)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid period_end format")
		return
	}

	invoice, err := h.service.GenerateInvoice(c.Request.Context(), accountID, periodStart, periodEnd)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to generate invoice")
		return
	}

	common.SuccessResponse(c, invoice)
}

// CreatePolicy creates a new policy
func (h *TestableHandler) CreatePolicy(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	var req struct {
		Name         string      `json:"name" binding:"required"`
		PolicyType   PolicyType  `json:"policy_type" binding:"required"`
		Rules        PolicyRules `json:"rules" binding:"required"`
		DepartmentID *uuid.UUID  `json:"department_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	policy, err := h.service.CreatePolicy(c.Request.Context(), accountID, req.Name, req.PolicyType, req.Rules, req.DepartmentID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create policy")
		return
	}

	common.SuccessResponse(c, policy)
}

// CheckPolicy checks a ride against policies
func (h *TestableHandler) CheckPolicy(c *gin.Context) {
	var req struct {
		AccountID     uuid.UUID               `json:"account_id" binding:"required"`
		EmployeeID    uuid.UUID               `json:"employee_id" binding:"required"`
		RideRequest   BookCorporateRideRequest `json:"ride_request" binding:"required"`
		EstimatedFare float64                 `json:"estimated_fare" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.CheckPolicies(c.Request.Context(), req.AccountID, req.EmployeeID, &req.RideRequest, req.EstimatedFare)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to check policies")
		return
	}

	common.SuccessResponse(c, result)
}

// UpdateBudget updates an account's budget settings
func (h *TestableHandler) UpdateBudget(c *gin.Context) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account ID")
		return
	}

	var req struct {
		CreditLimit float64 `json:"credit_limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	account, err := h.service.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "account not found")
		return
	}

	account.CreditLimit = req.CreditLimit
	if err := h.service.UpdateAccount(c.Request.Context(), account); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update budget")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Budget updated successfully",
	})
}

// ============================================================================
// HELPER FUNCTIONS
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

func ptrFloat(v float64) *float64   { return &v }
func ptrStr(v string) *string       { return &v }
func ptrUUID(v uuid.UUID) *uuid.UUID { return &v }

func createTestAccount(status AccountStatus) *CorporateAccount {
	now := time.Now()
	return &CorporateAccount{
		ID:              uuid.New(),
		Name:            "Acme Corp",
		LegalName:       "Acme Corporation LLC",
		Status:          status,
		PrimaryEmail:    "admin@acme.com",
		BillingEmail:    "billing@acme.com",
		BillingCycle:    BillingCycleMonthly,
		PaymentTermDays: 30,
		CreditLimit:     10000,
		CurrentBalance:  0,
		DiscountPercent: 10,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func createTestDepartment(accountID uuid.UUID) *Department {
	now := time.Now()
	return &Department{
		ID:                 uuid.New(),
		CorporateAccountID: accountID,
		Name:               "Engineering",
		Code:               ptrStr("ENG"),
		BudgetMonthly:      ptrFloat(5000),
		BudgetUsed:         1000,
		IsActive:           true,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

func createTestEmployee(accountID uuid.UUID, role EmployeeRole) *CorporateEmployee {
	now := time.Now()
	return &CorporateEmployee{
		ID:                 uuid.New(),
		CorporateAccountID: accountID,
		UserID:             uuid.New(),
		Role:               role,
		Email:              "john@acme.com",
		FirstName:          "John",
		LastName:           "Doe",
		MonthlyLimit:       ptrFloat(500),
		MonthlyUsed:        100,
		IsActive:           true,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

func createTestCorporateRide(accountID, employeeID uuid.UUID, approvalStatus string) *CorporateRide {
	now := time.Now()
	return &CorporateRide{
		ID:                 uuid.New(),
		RideID:             uuid.New(),
		CorporateAccountID: accountID,
		EmployeeID:         employeeID,
		OriginalFare:       50.0,
		DiscountAmount:     5.0,
		FinalFare:          45.0,
		RequiresApproval:   true,
		ApprovalStatus:     &approvalStatus,
		CreatedAt:          now,
	}
}

func createTestInvoice(accountID uuid.UUID) *CorporateInvoice {
	now := time.Now()
	return &CorporateInvoice{
		ID:                 uuid.New(),
		CorporateAccountID: accountID,
		InvoiceNumber:      "INV-12345",
		PeriodStart:        now.AddDate(0, -1, 0),
		PeriodEnd:          now,
		Subtotal:           1000.0,
		DiscountTotal:      100.0,
		TaxAmount:          0,
		TotalAmount:        900.0,
		Status:             "draft",
		DueDate:            now.AddDate(0, 0, 30),
		RideCount:          25,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

func createTestPolicy(accountID uuid.UUID, policyType PolicyType) *RidePolicy {
	now := time.Now()
	return &RidePolicy{
		ID:                 uuid.New(),
		CorporateAccountID: accountID,
		Name:               "Test Policy",
		PolicyType:         policyType,
		Rules:              PolicyRules{},
		Priority:           1,
		IsActive:           true,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// ============================================================================
// ACCOUNT MANAGEMENT TESTS
// ============================================================================

func TestHandler_CreateAccount_Success(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	reqBody := CreateAccountRequest{
		Name:         "Acme Corp",
		LegalName:    "Acme Corporation LLC",
		PrimaryEmail: "admin@acme.com",
		BillingEmail: "billing@acme.com",
		BillingCycle: BillingCycleMonthly,
	}

	expectedAccount := createTestAccount(AccountStatusPending)
	mockService.On("CreateAccount", mock.Anything, mock.AnythingOfType("*corporate.CreateAccountRequest")).Return(expectedAccount, nil)

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts", reqBody)
	handler.CreateAccount(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_CreateAccount_InvalidRequestBody(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts", "invalid json")
	handler.CreateAccount(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_CreateAccount_MissingRequiredFields(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]string{
		"name": "Acme Corp",
		// Missing required fields
	}

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts", reqBody)
	handler.CreateAccount(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateAccount_ServiceError(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	reqBody := CreateAccountRequest{
		Name:         "Acme Corp",
		LegalName:    "Acme Corporation LLC",
		PrimaryEmail: "admin@acme.com",
		BillingEmail: "billing@acme.com",
		BillingCycle: BillingCycleMonthly,
	}

	mockService.On("CreateAccount", mock.Anything, mock.AnythingOfType("*corporate.CreateAccountRequest")).Return(nil, errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts", reqBody)
	handler.CreateAccount(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CreateAccount_AppError(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	reqBody := CreateAccountRequest{
		Name:         "Acme Corp",
		LegalName:    "Acme Corporation LLC",
		PrimaryEmail: "admin@acme.com",
		BillingEmail: "billing@acme.com",
		BillingCycle: BillingCycleMonthly,
	}

	appErr := common.NewBadRequestError("duplicate email", nil)
	mockService.On("CreateAccount", mock.Anything, mock.AnythingOfType("*corporate.CreateAccountRequest")).Return(nil, appErr)

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts", reqBody)
	handler.CreateAccount(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetAccount_Success(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	account := createTestAccount(AccountStatusActive)
	mockService.On("GetAccount", mock.Anything, account.ID).Return(account, nil)

	c, w := setupTestContext("GET", "/api/v1/corporate/accounts/"+account.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: account.ID.String()}}
	setUserContext(c, uuid.New(), models.RoleAdmin)

	handler.GetAccount(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetAccount_InvalidID(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/corporate/accounts/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.GetAccount(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid account ID")
}

func TestHandler_GetAccount_NotFound(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	mockService.On("GetAccount", mock.Anything, accountID).Return(nil, common.NewNotFoundError("account not found", nil))

	c, w := setupTestContext("GET", "/api/v1/corporate/accounts/"+accountID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.GetAccount(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ActivateAccount_Success(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	mockService.On("ActivateAccount", mock.Anything, accountID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/corporate/accounts/"+accountID.String()+"/activate", nil)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}
	setUserContext(c, uuid.New(), models.RoleAdmin)

	handler.ActivateAccount(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_ActivateAccount_InvalidID(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/admin/corporate/accounts/invalid/activate", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, uuid.New(), models.RoleAdmin)

	handler.ActivateAccount(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ActivateAccount_AlreadyActive(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	appErr := common.NewBadRequestError("account is not pending activation", nil)
	mockService.On("ActivateAccount", mock.Anything, accountID).Return(appErr)

	c, w := setupTestContext("POST", "/api/v1/admin/corporate/accounts/"+accountID.String()+"/activate", nil)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}
	setUserContext(c, uuid.New(), models.RoleAdmin)

	handler.ActivateAccount(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ActivateAccount_ServiceError(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	mockService.On("ActivateAccount", mock.Anything, accountID).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/admin/corporate/accounts/"+accountID.String()+"/activate", nil)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}
	setUserContext(c, uuid.New(), models.RoleAdmin)

	handler.ActivateAccount(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SuspendAccount_Success(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	reqBody := map[string]string{"reason": "Policy violation"}
	mockService.On("SuspendAccount", mock.Anything, accountID, "Policy violation").Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/corporate/accounts/"+accountID.String()+"/suspend", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}
	setUserContext(c, uuid.New(), models.RoleAdmin)

	handler.SuspendAccount(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_SuspendAccount_MissingReason(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	reqBody := map[string]string{}

	c, w := setupTestContext("POST", "/api/v1/admin/corporate/accounts/"+accountID.String()+"/suspend", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}
	setUserContext(c, uuid.New(), models.RoleAdmin)

	handler.SuspendAccount(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListAccounts_Success(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accounts := []*CorporateAccount{
		createTestAccount(AccountStatusActive),
		createTestAccount(AccountStatusPending),
	}
	mockService.On("ListAccounts", mock.Anything, (*AccountStatus)(nil), 20, 0).Return(accounts, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/corporate/accounts", nil)
	setUserContext(c, uuid.New(), models.RoleAdmin)

	handler.ListAccounts(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_ListAccounts_WithStatusFilter(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accounts := []*CorporateAccount{createTestAccount(AccountStatusActive)}
	activeStatus := AccountStatusActive
	mockService.On("ListAccounts", mock.Anything, &activeStatus, 20, 0).Return(accounts, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/corporate/accounts?status=active", nil)
	c.Request.URL.RawQuery = "status=active"
	setUserContext(c, uuid.New(), models.RoleAdmin)

	handler.ListAccounts(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ListAccounts_ServiceError(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	mockService.On("ListAccounts", mock.Anything, (*AccountStatus)(nil), 20, 0).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/corporate/accounts", nil)
	setUserContext(c, uuid.New(), models.RoleAdmin)

	handler.ListAccounts(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// EMPLOYEE MANAGEMENT TESTS
// ============================================================================

func TestHandler_InviteEmployee_Success(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	reqBody := InviteEmployeeRequest{
		Email:     "john@acme.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      EmployeeRoleUser,
	}

	expectedEmp := createTestEmployee(accountID, EmployeeRoleUser)
	mockService.On("InviteEmployee", mock.Anything, accountID, mock.AnythingOfType("*corporate.InviteEmployeeRequest")).Return(expectedEmp, nil)

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/employees", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}
	setUserContext(c, uuid.New(), models.RoleAdmin)

	handler.InviteEmployee(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_InviteEmployee_InvalidAccountID(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	reqBody := InviteEmployeeRequest{
		Email:     "john@acme.com",
		FirstName: "John",
		LastName:  "Doe",
	}

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/invalid/employees", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	handler.InviteEmployee(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_InviteEmployee_InvalidRequestBody(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/employees", "invalid")
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.InviteEmployee(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_InviteEmployee_DuplicateEmail(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	reqBody := InviteEmployeeRequest{
		Email:     "john@acme.com",
		FirstName: "John",
		LastName:  "Doe",
	}

	appErr := common.NewBadRequestError("employee with this email already exists", nil)
	mockService.On("InviteEmployee", mock.Anything, accountID, mock.AnythingOfType("*corporate.InviteEmployeeRequest")).Return(nil, appErr)

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/employees", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.InviteEmployee(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_InviteEmployee_WithDepartment(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	deptID := uuid.New()
	reqBody := InviteEmployeeRequest{
		Email:        "john@acme.com",
		FirstName:    "John",
		LastName:     "Doe",
		Role:         EmployeeRoleUser,
		DepartmentID: &deptID,
	}

	expectedEmp := createTestEmployee(accountID, EmployeeRoleUser)
	expectedEmp.DepartmentID = &deptID
	mockService.On("InviteEmployee", mock.Anything, accountID, mock.AnythingOfType("*corporate.InviteEmployeeRequest")).Return(expectedEmp, nil)

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/employees", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.InviteEmployee(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_InviteEmployee_WithLimits(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	monthlyLimit := 500.0
	perRideLimit := 50.0
	reqBody := InviteEmployeeRequest{
		Email:        "john@acme.com",
		FirstName:    "John",
		LastName:     "Doe",
		Role:         EmployeeRoleUser,
		MonthlyLimit: &monthlyLimit,
		PerRideLimit: &perRideLimit,
	}

	expectedEmp := createTestEmployee(accountID, EmployeeRoleUser)
	expectedEmp.MonthlyLimit = &monthlyLimit
	expectedEmp.PerRideLimit = &perRideLimit
	mockService.On("InviteEmployee", mock.Anything, accountID, mock.AnythingOfType("*corporate.InviteEmployeeRequest")).Return(expectedEmp, nil)

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/employees", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.InviteEmployee(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ListEmployees_Success(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	employees := []*CorporateEmployee{
		createTestEmployee(accountID, EmployeeRoleAdmin),
		createTestEmployee(accountID, EmployeeRoleUser),
	}
	mockService.On("ListEmployees", mock.Anything, accountID, 20, 0).Return(employees, nil)

	c, w := setupTestContext("GET", "/api/v1/corporate/accounts/"+accountID.String()+"/employees", nil)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.ListEmployees(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_ListEmployees_InvalidAccountID(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/corporate/accounts/invalid/employees", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	handler.ListEmployees(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListEmployees_ServiceError(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	mockService.On("ListEmployees", mock.Anything, accountID, 20, 0).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/corporate/accounts/"+accountID.String()+"/employees", nil)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.ListEmployees(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetMyProfile_CorporateUser(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	accountID := uuid.New()
	employee := createTestEmployee(accountID, EmployeeRoleUser)
	employee.UserID = userID
	account := createTestAccount(AccountStatusActive)
	account.ID = accountID

	mockService.On("GetEmployeeByUserID", mock.Anything, userID).Return(employee, nil)
	mockService.On("GetAccount", mock.Anything, accountID).Return(account, nil)

	c, w := setupTestContext("GET", "/api/v1/corporate/me", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetMyProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.True(t, data["is_corporate_user"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetMyProfile_NonCorporateUser(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	mockService.On("GetEmployeeByUserID", mock.Anything, userID).Return(nil, common.NewNotFoundError("not found", nil))

	c, w := setupTestContext("GET", "/api/v1/corporate/me", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetMyProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.False(t, data["is_corporate_user"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetMyProfile_Unauthorized(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/corporate/me", nil)
	// No user context set

	handler.GetMyProfile(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// DEPARTMENT & BUDGET TESTS
// ============================================================================

func TestHandler_CreateDepartment_Success(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	reqBody := map[string]interface{}{
		"name":           "Engineering",
		"code":           "ENG",
		"budget_monthly": 5000.0,
	}

	expectedDept := createTestDepartment(accountID)
	mockService.On("CreateDepartment", mock.Anything, accountID, "Engineering", ptrStr("ENG"), (*uuid.UUID)(nil), ptrFloat(5000.0)).Return(expectedDept, nil)

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/departments", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.CreateDepartment(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_CreateDepartment_MissingName(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	reqBody := map[string]interface{}{
		"code": "ENG",
	}

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/departments", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.CreateDepartment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateDepartment_WithManager(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	managerID := uuid.New()
	reqBody := map[string]interface{}{
		"name":       "Engineering",
		"manager_id": managerID.String(),
	}

	expectedDept := createTestDepartment(accountID)
	expectedDept.ManagerID = &managerID
	mockService.On("CreateDepartment", mock.Anything, accountID, "Engineering", (*string)(nil), &managerID, (*float64)(nil)).Return(expectedDept, nil)

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/departments", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.CreateDepartment(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ListDepartments_Success(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	departments := []*Department{
		createTestDepartment(accountID),
		createTestDepartment(accountID),
	}
	departments[1].Name = "Marketing"
	mockService.On("ListDepartments", mock.Anything, accountID).Return(departments, nil)

	c, w := setupTestContext("GET", "/api/v1/corporate/accounts/"+accountID.String()+"/departments", nil)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.ListDepartments(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_ListDepartments_InvalidAccountID(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/corporate/accounts/invalid/departments", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	handler.ListDepartments(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateBudget_Success(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	account := createTestAccount(AccountStatusActive)
	account.ID = accountID

	reqBody := map[string]interface{}{
		"credit_limit": 25000.0,
	}

	mockService.On("GetAccount", mock.Anything, accountID).Return(account, nil)
	mockService.On("UpdateAccount", mock.Anything, mock.AnythingOfType("*corporate.CorporateAccount")).Return(nil)

	c, w := setupTestContext("PUT", "/api/v1/corporate/accounts/"+accountID.String()+"/budget", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.UpdateBudget(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateBudget_AccountNotFound(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	reqBody := map[string]interface{}{
		"credit_limit": 25000.0,
	}

	mockService.On("GetAccount", mock.Anything, accountID).Return(nil, common.NewNotFoundError("not found", nil))

	c, w := setupTestContext("PUT", "/api/v1/corporate/accounts/"+accountID.String()+"/budget", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.UpdateBudget(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetDashboard_Success(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	account := createTestAccount(AccountStatusActive)
	account.ID = accountID

	dashboard := &AccountDashboardResponse{
		Account:          account,
		EmployeeCount:    50,
		ActiveEmployees:  45,
		DepartmentCount:  5,
		PendingApprovals: 3,
	}

	mockService.On("GetDashboard", mock.Anything, accountID).Return(dashboard, nil)

	c, w := setupTestContext("GET", "/api/v1/corporate/accounts/"+accountID.String()+"/dashboard", nil)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.GetDashboard(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetDashboard_InvalidAccountID(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/corporate/accounts/invalid/dashboard", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	handler.GetDashboard(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// BILLING & INVOICE TESTS
// ============================================================================

func TestHandler_GenerateInvoice_Success(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	reqBody := map[string]string{
		"period_start": "2025-01-01",
		"period_end":   "2025-01-31",
	}

	expectedInvoice := createTestInvoice(accountID)
	periodStart, _ := time.Parse("2006-01-02", "2025-01-01")
	periodEnd, _ := time.Parse("2006-01-02", "2025-01-31")
	mockService.On("GenerateInvoice", mock.Anything, accountID, periodStart, periodEnd).Return(expectedInvoice, nil)

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/invoices", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.GenerateInvoice(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GenerateInvoice_InvalidAccountID(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]string{
		"period_start": "2025-01-01",
		"period_end":   "2025-01-31",
	}

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/invalid/invoices", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	handler.GenerateInvoice(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GenerateInvoice_InvalidDateFormat(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	reqBody := map[string]string{
		"period_start": "invalid-date",
		"period_end":   "2025-01-31",
	}

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/invoices", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.GenerateInvoice(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "period_start")
}

func TestHandler_GenerateInvoice_InvalidEndDateFormat(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	reqBody := map[string]string{
		"period_start": "2025-01-01",
		"period_end":   "invalid-date",
	}

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/invoices", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.GenerateInvoice(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "period_end")
}

func TestHandler_GenerateInvoice_MissingDates(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	reqBody := map[string]string{}

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/invoices", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.GenerateInvoice(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GenerateInvoice_ServiceError(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	reqBody := map[string]string{
		"period_start": "2025-01-01",
		"period_end":   "2025-01-31",
	}

	periodStart, _ := time.Parse("2006-01-02", "2025-01-01")
	periodEnd, _ := time.Parse("2006-01-02", "2025-01-31")
	mockService.On("GenerateInvoice", mock.Anything, accountID, periodStart, periodEnd).Return(nil, errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/invoices", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.GenerateInvoice(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ListInvoices_Success(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	invoices := []*CorporateInvoice{
		createTestInvoice(accountID),
		createTestInvoice(accountID),
	}
	mockService.On("ListInvoices", mock.Anything, accountID, 20, 0).Return(invoices, nil)

	c, w := setupTestContext("GET", "/api/v1/corporate/accounts/"+accountID.String()+"/invoices", nil)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.ListInvoices(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_ListInvoices_InvalidAccountID(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/corporate/accounts/invalid/invoices", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	handler.ListInvoices(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListInvoices_ServiceError(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	mockService.On("ListInvoices", mock.Anything, accountID, 20, 0).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/corporate/accounts/"+accountID.String()+"/invoices", nil)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.ListInvoices(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// POLICY ENFORCEMENT TESTS
// ============================================================================

func TestHandler_CreatePolicy_Success(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	reqBody := map[string]interface{}{
		"name":        "Business Hours Only",
		"policy_type": PolicyTypeTimeRestriction,
		"rules": PolicyRules{
			AllowedDays:      []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
			AllowedStartTime: ptrStr("08:00"),
			AllowedEndTime:   ptrStr("18:00"),
		},
	}

	expectedPolicy := createTestPolicy(accountID, PolicyTypeTimeRestriction)
	mockService.On("CreatePolicy", mock.Anything, accountID, "Business Hours Only", PolicyTypeTimeRestriction, mock.AnythingOfType("corporate.PolicyRules"), (*uuid.UUID)(nil)).Return(expectedPolicy, nil)

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/policies", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.CreatePolicy(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_CreatePolicy_InvalidAccountID(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"name":        "Test Policy",
		"policy_type": PolicyTypeTimeRestriction,
		"rules":       PolicyRules{},
	}

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/invalid/policies", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	handler.CreatePolicy(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreatePolicy_MissingName(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	reqBody := map[string]interface{}{
		"policy_type": PolicyTypeTimeRestriction,
		"rules":       PolicyRules{},
	}

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/policies", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.CreatePolicy(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreatePolicy_AmountLimitPolicy(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	reqBody := map[string]interface{}{
		"name":        "Ride Amount Limit",
		"policy_type": PolicyTypeAmountLimit,
		"rules": PolicyRules{
			MaxAmountPerRide: ptrFloat(100.0),
		},
	}

	expectedPolicy := createTestPolicy(accountID, PolicyTypeAmountLimit)
	mockService.On("CreatePolicy", mock.Anything, accountID, "Ride Amount Limit", PolicyTypeAmountLimit, mock.AnythingOfType("corporate.PolicyRules"), (*uuid.UUID)(nil)).Return(expectedPolicy, nil)

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/policies", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.CreatePolicy(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CreatePolicy_RideTypeRestrictionPolicy(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	reqBody := map[string]interface{}{
		"name":        "Economy Only",
		"policy_type": PolicyTypeRideTypeRestriction,
		"rules": PolicyRules{
			AllowedRideTypes: []string{"economy", "pool"},
		},
	}

	expectedPolicy := createTestPolicy(accountID, PolicyTypeRideTypeRestriction)
	mockService.On("CreatePolicy", mock.Anything, accountID, "Economy Only", PolicyTypeRideTypeRestriction, mock.AnythingOfType("corporate.PolicyRules"), (*uuid.UUID)(nil)).Return(expectedPolicy, nil)

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/policies", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.CreatePolicy(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CreatePolicy_WithDepartment(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	deptID := uuid.New()
	reqBody := map[string]interface{}{
		"name":          "Engineering Policy",
		"policy_type":   PolicyTypeTimeRestriction,
		"rules":         PolicyRules{},
		"department_id": deptID.String(),
	}

	expectedPolicy := createTestPolicy(accountID, PolicyTypeTimeRestriction)
	expectedPolicy.DepartmentID = &deptID
	mockService.On("CreatePolicy", mock.Anything, accountID, "Engineering Policy", PolicyTypeTimeRestriction, mock.AnythingOfType("corporate.PolicyRules"), &deptID).Return(expectedPolicy, nil)

	c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/policies", reqBody)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.CreatePolicy(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CheckPolicy_Allowed(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	employeeID := uuid.New()

	reqBody := map[string]interface{}{
		"account_id":  accountID.String(),
		"employee_id": employeeID.String(),
		"ride_request": BookCorporateRideRequest{
			PickupLocation:  Location{Latitude: 40.7128, Longitude: -74.0060},
			DropoffLocation: Location{Latitude: 40.7580, Longitude: -73.9855},
			RideType:        "economy",
		},
		"estimated_fare": 25.0,
	}

	result := &PolicyCheckResult{
		Allowed:          true,
		Violations:       []PolicyViolation{},
		RequiresApproval: false,
	}
	mockService.On("CheckPolicies", mock.Anything, accountID, employeeID, mock.AnythingOfType("*corporate.BookCorporateRideRequest"), 25.0).Return(result, nil)

	c, w := setupTestContext("POST", "/api/v1/corporate/policies/check", reqBody)

	handler.CheckPolicy(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_CheckPolicy_BudgetExceeded(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	employeeID := uuid.New()

	reqBody := map[string]interface{}{
		"account_id":  accountID.String(),
		"employee_id": employeeID.String(),
		"ride_request": BookCorporateRideRequest{
			RideType: "economy",
		},
		"estimated_fare": 500.0,
	}

	result := &PolicyCheckResult{
		Allowed: false,
		Violations: []PolicyViolation{
			{
				PolicyName: "Monthly Limit",
				Reason:     "This ride would exceed your monthly limit",
				Severity:   "block",
			},
		},
		RequiresApproval: false,
	}
	mockService.On("CheckPolicies", mock.Anything, accountID, employeeID, mock.AnythingOfType("*corporate.BookCorporateRideRequest"), 500.0).Return(result, nil)

	c, w := setupTestContext("POST", "/api/v1/corporate/policies/check", reqBody)

	handler.CheckPolicy(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.False(t, data["allowed"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_CheckPolicy_TimeRestrictionViolated(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	employeeID := uuid.New()

	reqBody := map[string]interface{}{
		"account_id":  accountID.String(),
		"employee_id": employeeID.String(),
		"ride_request": BookCorporateRideRequest{
			RideType: "economy",
		},
		"estimated_fare": 25.0,
	}

	result := &PolicyCheckResult{
		Allowed: false,
		Violations: []PolicyViolation{
			{
				PolicyName: "Business Hours Only",
				Reason:     "Rides only allowed between 08:00 and 18:00",
				Severity:   "block",
			},
		},
		RequiresApproval: false,
	}
	mockService.On("CheckPolicies", mock.Anything, accountID, employeeID, mock.AnythingOfType("*corporate.BookCorporateRideRequest"), 25.0).Return(result, nil)

	c, w := setupTestContext("POST", "/api/v1/corporate/policies/check", reqBody)

	handler.CheckPolicy(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.False(t, data["allowed"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_CheckPolicy_RequiresApproval(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	employeeID := uuid.New()

	reqBody := map[string]interface{}{
		"account_id":  accountID.String(),
		"employee_id": employeeID.String(),
		"ride_request": BookCorporateRideRequest{
			RideType: "premium",
		},
		"estimated_fare": 75.0,
	}

	approvalReason := "Ride cost exceeds per-ride limit"
	result := &PolicyCheckResult{
		Allowed:          true,
		Violations:       []PolicyViolation{},
		RequiresApproval: true,
		ApprovalReason:   &approvalReason,
	}
	mockService.On("CheckPolicies", mock.Anything, accountID, employeeID, mock.AnythingOfType("*corporate.BookCorporateRideRequest"), 75.0).Return(result, nil)

	c, w := setupTestContext("POST", "/api/v1/corporate/policies/check", reqBody)

	handler.CheckPolicy(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.True(t, data["allowed"].(bool))
	assert.True(t, data["requires_approval"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_CheckPolicy_InvalidRequestBody(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/corporate/policies/check", "invalid")

	handler.CheckPolicy(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CheckPolicy_EmployeeNotFound(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	employeeID := uuid.New()

	reqBody := map[string]interface{}{
		"account_id":  accountID.String(),
		"employee_id": employeeID.String(),
		"ride_request": BookCorporateRideRequest{
			RideType: "economy",
		},
		"estimated_fare": 25.0,
	}

	mockService.On("CheckPolicies", mock.Anything, accountID, employeeID, mock.AnythingOfType("*corporate.BookCorporateRideRequest"), 25.0).Return(nil, common.NewNotFoundError("employee not found", nil))

	c, w := setupTestContext("POST", "/api/v1/corporate/policies/check", reqBody)

	handler.CheckPolicy(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// RIDE APPROVAL TESTS
// ============================================================================

func TestHandler_ApproveRide_Approve_Success(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	rideID := uuid.New()
	reqBody := map[string]bool{"approved": true}

	mockService.On("ApproveRide", mock.Anything, rideID, userID, true).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/corporate/rides/"+rideID.String()+"/approve", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.ApproveRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Contains(t, data["message"], "approved")
	mockService.AssertExpectations(t)
}

func TestHandler_ApproveRide_Reject_Success(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	rideID := uuid.New()
	reqBody := map[string]bool{"approved": false}

	mockService.On("ApproveRide", mock.Anything, rideID, userID, false).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/corporate/rides/"+rideID.String()+"/approve", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.ApproveRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Contains(t, data["message"], "rejected")
	mockService.AssertExpectations(t)
}

func TestHandler_ApproveRide_InvalidRideID(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	reqBody := map[string]bool{"approved": true}

	c, w := setupTestContext("POST", "/api/v1/corporate/rides/invalid/approve", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, userID, models.RoleRider)

	handler.ApproveRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ApproveRide_Unauthorized(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	rideID := uuid.New()
	reqBody := map[string]bool{"approved": true}

	c, w := setupTestContext("POST", "/api/v1/corporate/rides/"+rideID.String()+"/approve", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	// No user context

	handler.ApproveRide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_ApproveRide_RideNotPending(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	rideID := uuid.New()
	reqBody := map[string]bool{"approved": true}

	appErr := common.NewBadRequestError("ride is not pending approval", nil)
	mockService.On("ApproveRide", mock.Anything, rideID, userID, true).Return(appErr)

	c, w := setupTestContext("POST", "/api/v1/corporate/rides/"+rideID.String()+"/approve", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.ApproveRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ApproveRide_RideNotFound(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	userID := uuid.New()
	rideID := uuid.New()
	reqBody := map[string]bool{"approved": true}

	mockService.On("ApproveRide", mock.Anything, rideID, userID, true).Return(common.NewNotFoundError("ride not found", nil))

	c, w := setupTestContext("POST", "/api/v1/corporate/rides/"+rideID.String()+"/approve", reqBody)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.ApproveRide(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetPendingApprovals_Success(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	employeeID := uuid.New()

	pendingRides := []*CorporateRide{
		createTestCorporateRide(accountID, employeeID, "pending"),
		createTestCorporateRide(accountID, employeeID, "pending"),
	}

	mockService.On("GetPendingApprovals", mock.Anything, accountID, (*uuid.UUID)(nil)).Return(pendingRides, nil)

	c, w := setupTestContext("GET", "/api/v1/corporate/accounts/"+accountID.String()+"/approvals", nil)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.GetPendingApprovals(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["count"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetPendingApprovals_Empty(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	mockService.On("GetPendingApprovals", mock.Anything, accountID, (*uuid.UUID)(nil)).Return([]*CorporateRide{}, nil)

	c, w := setupTestContext("GET", "/api/v1/corporate/accounts/"+accountID.String()+"/approvals", nil)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.GetPendingApprovals(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["count"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetPendingApprovals_InvalidAccountID(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/corporate/accounts/invalid/approvals", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	handler.GetPendingApprovals(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// RIDES LIST TESTS
// ============================================================================

func TestHandler_ListRides_Success(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	employeeID := uuid.New()

	rides := []*CorporateRide{
		createTestCorporateRide(accountID, employeeID, "approved"),
		createTestCorporateRide(accountID, employeeID, "approved"),
	}

	mockService.On("ListCorporateRides", mock.Anything, accountID, (*uuid.UUID)(nil), mock.Anything, mock.Anything, 50, 0).Return(rides, nil)

	c, w := setupTestContext("GET", "/api/v1/corporate/accounts/"+accountID.String()+"/rides", nil)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.ListRides(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["count"])
	mockService.AssertExpectations(t)
}

func TestHandler_ListRides_WithEmployeeFilter(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	employeeID := uuid.New()

	rides := []*CorporateRide{
		createTestCorporateRide(accountID, employeeID, "approved"),
	}

	mockService.On("ListCorporateRides", mock.Anything, accountID, &employeeID, mock.Anything, mock.Anything, 50, 0).Return(rides, nil)

	c, w := setupTestContext("GET", "/api/v1/corporate/accounts/"+accountID.String()+"/rides?employee_id="+employeeID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}
	c.Request.URL.RawQuery = "employee_id=" + employeeID.String()

	handler.ListRides(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ListRides_InvalidAccountID(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/corporate/accounts/invalid/rides", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	handler.ListRides(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListRides_ServiceError(t *testing.T) {
	mockService := new(MockCorporateService)
	handler := NewTestableHandler(mockService)

	accountID := uuid.New()
	mockService.On("ListCorporateRides", mock.Anything, accountID, (*uuid.UUID)(nil), mock.Anything, mock.Anything, 50, 0).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/corporate/accounts/"+accountID.String()+"/rides", nil)
	c.Params = gin.Params{{Key: "id", Value: accountID.String()}}

	handler.ListRides(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// TABLE-DRIVEN TESTS
// ============================================================================

func TestHandler_CreateAccount_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*MockCorporateService)
		expectedStatus int
		expectedBody   func(map[string]interface{}) bool
	}{
		{
			name: "success with all fields",
			requestBody: CreateAccountRequest{
				Name:            "Acme Corp",
				LegalName:       "Acme Corporation LLC",
				TaxID:           ptrStr("12-3456789"),
				PrimaryEmail:    "admin@acme.com",
				PrimaryPhone:    ptrStr("+1234567890"),
				BillingEmail:    "billing@acme.com",
				BillingCycle:    BillingCycleMonthly,
				PaymentTermDays: 30,
				Industry:        ptrStr("Technology"),
				CompanySize:     ptrStr("medium"),
			},
			setupMock: func(m *MockCorporateService) {
				m.On("CreateAccount", mock.Anything, mock.AnythingOfType("*corporate.CreateAccountRequest")).Return(createTestAccount(AccountStatusPending), nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(resp map[string]interface{}) bool {
				return resp["success"].(bool)
			},
		},
		{
			name: "success with weekly billing",
			requestBody: CreateAccountRequest{
				Name:         "Small Co",
				LegalName:    "Small Company Inc",
				PrimaryEmail: "admin@small.com",
				BillingEmail: "billing@small.com",
				BillingCycle: BillingCycleWeekly,
			},
			setupMock: func(m *MockCorporateService) {
				account := createTestAccount(AccountStatusPending)
				account.BillingCycle = BillingCycleWeekly
				m.On("CreateAccount", mock.Anything, mock.AnythingOfType("*corporate.CreateAccountRequest")).Return(account, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(resp map[string]interface{}) bool {
				return resp["success"].(bool)
			},
		},
		{
			name: "success with quarterly billing",
			requestBody: CreateAccountRequest{
				Name:         "Enterprise Co",
				LegalName:    "Enterprise Company LLC",
				PrimaryEmail: "admin@enterprise.com",
				BillingEmail: "billing@enterprise.com",
				BillingCycle: BillingCycleQuarterly,
			},
			setupMock: func(m *MockCorporateService) {
				account := createTestAccount(AccountStatusPending)
				account.BillingCycle = BillingCycleQuarterly
				m.On("CreateAccount", mock.Anything, mock.AnythingOfType("*corporate.CreateAccountRequest")).Return(account, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(resp map[string]interface{}) bool {
				return resp["success"].(bool)
			},
		},
		{
			name:           "invalid json body",
			requestBody:    "not valid json",
			setupMock:      func(m *MockCorporateService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(resp map[string]interface{}) bool {
				return !resp["success"].(bool)
			},
		},
		{
			name:           "empty request body",
			requestBody:    map[string]string{},
			setupMock:      func(m *MockCorporateService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(resp map[string]interface{}) bool {
				return !resp["success"].(bool)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockCorporateService)
			handler := NewTestableHandler(mockService)
			tt.setupMock(mockService)

			c, w := setupTestContext("POST", "/api/v1/corporate/accounts", tt.requestBody)
			handler.CreateAccount(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			response := parseResponse(w)
			assert.True(t, tt.expectedBody(response))
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_InviteEmployee_TableDriven(t *testing.T) {
	accountID := uuid.New()

	tests := []struct {
		name           string
		accountID      string
		requestBody    interface{}
		setupMock      func(*MockCorporateService)
		expectedStatus int
	}{
		{
			name:      "admin role",
			accountID: accountID.String(),
			requestBody: InviteEmployeeRequest{
				Email:     "admin@acme.com",
				FirstName: "Admin",
				LastName:  "User",
				Role:      EmployeeRoleAdmin,
			},
			setupMock: func(m *MockCorporateService) {
				emp := createTestEmployee(accountID, EmployeeRoleAdmin)
				m.On("InviteEmployee", mock.Anything, accountID, mock.AnythingOfType("*corporate.InviteEmployeeRequest")).Return(emp, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "manager role",
			accountID: accountID.String(),
			requestBody: InviteEmployeeRequest{
				Email:     "manager@acme.com",
				FirstName: "Manager",
				LastName:  "User",
				Role:      EmployeeRoleManager,
			},
			setupMock: func(m *MockCorporateService) {
				emp := createTestEmployee(accountID, EmployeeRoleManager)
				m.On("InviteEmployee", mock.Anything, accountID, mock.AnythingOfType("*corporate.InviteEmployeeRequest")).Return(emp, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "user role (default)",
			accountID: accountID.String(),
			requestBody: InviteEmployeeRequest{
				Email:     "user@acme.com",
				FirstName: "Regular",
				LastName:  "User",
			},
			setupMock: func(m *MockCorporateService) {
				emp := createTestEmployee(accountID, EmployeeRoleUser)
				m.On("InviteEmployee", mock.Anything, accountID, mock.AnythingOfType("*corporate.InviteEmployeeRequest")).Return(emp, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "invalid account id",
			accountID: "invalid-uuid",
			requestBody: InviteEmployeeRequest{
				Email:     "user@acme.com",
				FirstName: "Test",
				LastName:  "User",
			},
			setupMock:      func(m *MockCorporateService) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockCorporateService)
			handler := NewTestableHandler(mockService)
			tt.setupMock(mockService)

			c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+tt.accountID+"/employees", tt.requestBody)
			c.Params = gin.Params{{Key: "id", Value: tt.accountID}}
			handler.InviteEmployee(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_PolicyTypes_TableDriven(t *testing.T) {
	accountID := uuid.New()

	tests := []struct {
		name       string
		policyType PolicyType
		rules      PolicyRules
	}{
		{
			name:       "time restriction",
			policyType: PolicyTypeTimeRestriction,
			rules: PolicyRules{
				AllowedDays:      []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
				AllowedStartTime: ptrStr("08:00"),
				AllowedEndTime:   ptrStr("18:00"),
			},
		},
		{
			name:       "amount limit",
			policyType: PolicyTypeAmountLimit,
			rules: PolicyRules{
				MaxAmountPerRide:  ptrFloat(100.0),
				MaxAmountPerDay:   ptrFloat(200.0),
				MaxAmountPerMonth: ptrFloat(2000.0),
			},
		},
		{
			name:       "ride type restriction - allowed",
			policyType: PolicyTypeRideTypeRestriction,
			rules: PolicyRules{
				AllowedRideTypes: []string{"economy", "pool"},
			},
		},
		{
			name:       "ride type restriction - blocked",
			policyType: PolicyTypeRideTypeRestriction,
			rules: PolicyRules{
				BlockedRideTypes: []string{"premium", "luxury"},
			},
		},
		{
			name:       "approval required",
			policyType: PolicyTypeApprovalRequired,
			rules: PolicyRules{
				ApprovalThreshold: ptrFloat(50.0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockCorporateService)
			handler := NewTestableHandler(mockService)

			expectedPolicy := createTestPolicy(accountID, tt.policyType)
			expectedPolicy.Rules = tt.rules
			mockService.On("CreatePolicy", mock.Anything, accountID, "Test Policy", tt.policyType, mock.AnythingOfType("corporate.PolicyRules"), (*uuid.UUID)(nil)).Return(expectedPolicy, nil)

			reqBody := map[string]interface{}{
				"name":        "Test Policy",
				"policy_type": tt.policyType,
				"rules":       tt.rules,
			}

			c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/policies", reqBody)
			c.Params = gin.Params{{Key: "id", Value: accountID.String()}}
			handler.CreatePolicy(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_AccountStatuses_TableDriven(t *testing.T) {
	tests := []struct {
		name   string
		status AccountStatus
	}{
		{name: "pending status", status: AccountStatusPending},
		{name: "active status", status: AccountStatusActive},
		{name: "suspended status", status: AccountStatusSuspended},
		{name: "closed status", status: AccountStatusClosed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockCorporateService)
			handler := NewTestableHandler(mockService)

			accounts := []*CorporateAccount{createTestAccount(tt.status)}
			mockService.On("ListAccounts", mock.Anything, &tt.status, 20, 0).Return(accounts, nil)

			c, w := setupTestContext("GET", "/api/v1/admin/corporate/accounts?status="+string(tt.status), nil)
			c.Request.URL.RawQuery = "status=" + string(tt.status)
			setUserContext(c, uuid.New(), models.RoleAdmin)
			handler.ListAccounts(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// ============================================================================
// ERROR SCENARIO TESTS
// ============================================================================

func TestHandler_ErrorScenarios(t *testing.T) {
	t.Run("CreateAccount_InternalError", func(t *testing.T) {
		mockService := new(MockCorporateService)
		handler := NewTestableHandler(mockService)

		mockService.On("CreateAccount", mock.Anything, mock.AnythingOfType("*corporate.CreateAccountRequest")).Return(nil, errors.New("internal error"))

		reqBody := CreateAccountRequest{
			Name:         "Test",
			LegalName:    "Test LLC",
			PrimaryEmail: "test@test.com",
			BillingEmail: "billing@test.com",
			BillingCycle: BillingCycleMonthly,
		}

		c, w := setupTestContext("POST", "/api/v1/corporate/accounts", reqBody)
		handler.CreateAccount(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("GetAccount_InternalError", func(t *testing.T) {
		mockService := new(MockCorporateService)
		handler := NewTestableHandler(mockService)

		accountID := uuid.New()
		mockService.On("GetAccount", mock.Anything, accountID).Return(nil, errors.New("database error"))

		c, w := setupTestContext("GET", "/api/v1/corporate/accounts/"+accountID.String(), nil)
		c.Params = gin.Params{{Key: "id", Value: accountID.String()}}
		handler.GetAccount(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("CreateDepartment_InternalError", func(t *testing.T) {
		mockService := new(MockCorporateService)
		handler := NewTestableHandler(mockService)

		accountID := uuid.New()
		mockService.On("CreateDepartment", mock.Anything, accountID, "Test", (*string)(nil), (*uuid.UUID)(nil), (*float64)(nil)).Return(nil, errors.New("internal error"))

		reqBody := map[string]interface{}{"name": "Test"}
		c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/departments", reqBody)
		c.Params = gin.Params{{Key: "id", Value: accountID.String()}}
		handler.CreateDepartment(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("InviteEmployee_InternalError", func(t *testing.T) {
		mockService := new(MockCorporateService)
		handler := NewTestableHandler(mockService)

		accountID := uuid.New()
		mockService.On("InviteEmployee", mock.Anything, accountID, mock.AnythingOfType("*corporate.InviteEmployeeRequest")).Return(nil, errors.New("internal error"))

		reqBody := InviteEmployeeRequest{
			Email:     "test@test.com",
			FirstName: "Test",
			LastName:  "User",
		}
		c, w := setupTestContext("POST", "/api/v1/corporate/accounts/"+accountID.String()+"/employees", reqBody)
		c.Params = gin.Params{{Key: "id", Value: accountID.String()}}
		handler.InviteEmployee(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
