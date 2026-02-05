package corporate

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ========================================
// INTERNAL MOCK (same package for unexported access)
// ========================================

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) CreateAccount(ctx context.Context, account *CorporateAccount) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *mockRepo) GetAccount(ctx context.Context, accountID uuid.UUID) (*CorporateAccount, error) {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CorporateAccount), args.Error(1)
}

func (m *mockRepo) UpdateAccount(ctx context.Context, account *CorporateAccount) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *mockRepo) UpdateAccountStatus(ctx context.Context, accountID uuid.UUID, status AccountStatus) error {
	args := m.Called(ctx, accountID, status)
	return args.Error(0)
}

func (m *mockRepo) UpdateAccountBalance(ctx context.Context, accountID uuid.UUID, amount float64) error {
	args := m.Called(ctx, accountID, amount)
	return args.Error(0)
}

func (m *mockRepo) ListAccounts(ctx context.Context, status *AccountStatus, limit, offset int) ([]*CorporateAccount, error) {
	args := m.Called(ctx, status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*CorporateAccount), args.Error(1)
}

func (m *mockRepo) CreateDepartment(ctx context.Context, dept *Department) error {
	args := m.Called(ctx, dept)
	return args.Error(0)
}

func (m *mockRepo) GetDepartment(ctx context.Context, deptID uuid.UUID) (*Department, error) {
	args := m.Called(ctx, deptID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Department), args.Error(1)
}

func (m *mockRepo) ListDepartments(ctx context.Context, accountID uuid.UUID) ([]*Department, error) {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Department), args.Error(1)
}

func (m *mockRepo) UpdateDepartmentBudget(ctx context.Context, deptID uuid.UUID, amount float64) error {
	args := m.Called(ctx, deptID, amount)
	return args.Error(0)
}

func (m *mockRepo) ResetDepartmentBudgets(ctx context.Context, accountID uuid.UUID) error {
	args := m.Called(ctx, accountID)
	return args.Error(0)
}

func (m *mockRepo) CreateEmployee(ctx context.Context, emp *CorporateEmployee) error {
	args := m.Called(ctx, emp)
	return args.Error(0)
}

func (m *mockRepo) GetEmployee(ctx context.Context, empID uuid.UUID) (*CorporateEmployee, error) {
	args := m.Called(ctx, empID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CorporateEmployee), args.Error(1)
}

func (m *mockRepo) GetEmployeeByUserID(ctx context.Context, userID uuid.UUID) (*CorporateEmployee, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CorporateEmployee), args.Error(1)
}

func (m *mockRepo) GetEmployeeByEmail(ctx context.Context, accountID uuid.UUID, email string) (*CorporateEmployee, error) {
	args := m.Called(ctx, accountID, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CorporateEmployee), args.Error(1)
}

func (m *mockRepo) ListEmployees(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*CorporateEmployee, error) {
	args := m.Called(ctx, accountID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*CorporateEmployee), args.Error(1)
}

func (m *mockRepo) UpdateEmployeeUsage(ctx context.Context, empID uuid.UUID, amount float64) error {
	args := m.Called(ctx, empID, amount)
	return args.Error(0)
}

func (m *mockRepo) ResetEmployeeUsage(ctx context.Context, accountID uuid.UUID) error {
	args := m.Called(ctx, accountID)
	return args.Error(0)
}

func (m *mockRepo) GetEmployeeCount(ctx context.Context, accountID uuid.UUID, activeOnly bool) (int, error) {
	args := m.Called(ctx, accountID, activeOnly)
	return args.Int(0), args.Error(1)
}

func (m *mockRepo) CreatePolicy(ctx context.Context, policy *RidePolicy) error {
	args := m.Called(ctx, policy)
	return args.Error(0)
}

func (m *mockRepo) GetPolicies(ctx context.Context, accountID uuid.UUID, departmentID *uuid.UUID) ([]*RidePolicy, error) {
	args := m.Called(ctx, accountID, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RidePolicy), args.Error(1)
}

func (m *mockRepo) CreateCorporateRide(ctx context.Context, ride *CorporateRide) error {
	args := m.Called(ctx, ride)
	return args.Error(0)
}

func (m *mockRepo) GetCorporateRide(ctx context.Context, rideID uuid.UUID) (*CorporateRide, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CorporateRide), args.Error(1)
}

func (m *mockRepo) ListCorporateRides(ctx context.Context, accountID uuid.UUID, employeeID *uuid.UUID, startDate, endDate time.Time, limit, offset int) ([]*CorporateRide, error) {
	args := m.Called(ctx, accountID, employeeID, startDate, endDate, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*CorporateRide), args.Error(1)
}

func (m *mockRepo) GetPendingApprovals(ctx context.Context, accountID uuid.UUID, approverID *uuid.UUID) ([]*CorporateRide, error) {
	args := m.Called(ctx, accountID, approverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*CorporateRide), args.Error(1)
}

func (m *mockRepo) ApproveRide(ctx context.Context, rideID, approverID uuid.UUID, approved bool) error {
	args := m.Called(ctx, rideID, approverID, approved)
	return args.Error(0)
}

func (m *mockRepo) CreateInvoice(ctx context.Context, invoice *CorporateInvoice) error {
	args := m.Called(ctx, invoice)
	return args.Error(0)
}

func (m *mockRepo) GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*CorporateInvoice, error) {
	args := m.Called(ctx, invoiceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CorporateInvoice), args.Error(1)
}

func (m *mockRepo) ListInvoices(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*CorporateInvoice, error) {
	args := m.Called(ctx, accountID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*CorporateInvoice), args.Error(1)
}

func (m *mockRepo) GetPeriodStats(ctx context.Context, accountID uuid.UUID, startDate, endDate time.Time) (*PeriodStats, error) {
	args := m.Called(ctx, accountID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PeriodStats), args.Error(1)
}

func (m *mockRepo) GetTopSpenders(ctx context.Context, accountID uuid.UUID, startDate, endDate time.Time, limit int) ([]EmployeeSpending, error) {
	args := m.Called(ctx, accountID, startDate, endDate, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]EmployeeSpending), args.Error(1)
}

func (m *mockRepo) GetDepartmentUsage(ctx context.Context, accountID uuid.UUID, startDate, endDate time.Time) ([]DepartmentUsage, error) {
	args := m.Called(ctx, accountID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]DepartmentUsage), args.Error(1)
}

// ========================================
// HELPER FUNCTIONS
// ========================================

func ptrFloat64(v float64) *float64 { return &v }
func ptrString(v string) *string    { return &v }

// ========================================
// TESTS
// ========================================

func TestCreateAccount_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	req := &CreateAccountRequest{
		Name:         "Acme Corp",
		LegalName:    "Acme Corporation LLC",
		PrimaryEmail: "admin@acme.com",
		BillingEmail: "billing@acme.com",
		BillingCycle: BillingCycleMonthly,
	}

	repo.On("CreateAccount", ctx, mock.AnythingOfType("*corporate.CorporateAccount")).Return(nil)

	account, err := svc.CreateAccount(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, account)
	assert.Equal(t, "Acme Corp", account.Name)
	assert.Equal(t, "Acme Corporation LLC", account.LegalName)
	assert.Equal(t, AccountStatusPending, account.Status)
	assert.Equal(t, 30, account.PaymentTermDays) // default
	assert.Equal(t, float64(10000), account.CreditLimit)
	assert.Equal(t, float64(10), account.DiscountPercent)
	assert.NotEqual(t, uuid.Nil, account.ID)
	repo.AssertExpectations(t)
}

func TestCreateAccount_RepoError(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	req := &CreateAccountRequest{
		Name:         "Acme Corp",
		LegalName:    "Acme Corporation LLC",
		PrimaryEmail: "admin@acme.com",
		BillingEmail: "billing@acme.com",
		BillingCycle: BillingCycleMonthly,
	}

	repo.On("CreateAccount", ctx, mock.AnythingOfType("*corporate.CorporateAccount")).Return(errors.New("db error"))

	account, err := svc.CreateAccount(ctx, req)

	require.Error(t, err)
	assert.Nil(t, account)
	assert.Contains(t, err.Error(), "internal server error")
	repo.AssertExpectations(t)
}

func TestGetAccount_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	accountID := uuid.New()
	expected := &CorporateAccount{
		ID:     accountID,
		Name:   "Acme Corp",
		Status: AccountStatusActive,
	}

	repo.On("GetAccount", ctx, accountID).Return(expected, nil)

	account, err := svc.GetAccount(ctx, accountID)

	require.NoError(t, err)
	require.NotNil(t, account)
	assert.Equal(t, accountID, account.ID)
	assert.Equal(t, "Acme Corp", account.Name)
	repo.AssertExpectations(t)
}

func TestGetAccount_NotFound(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	accountID := uuid.New()

	repo.On("GetAccount", ctx, accountID).Return(nil, errors.New("not found"))

	account, err := svc.GetAccount(ctx, accountID)

	require.Error(t, err)
	assert.Nil(t, account)
	assert.Contains(t, err.Error(), "not found")
	repo.AssertExpectations(t)
}

func TestActivateAccount_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	accountID := uuid.New()
	account := &CorporateAccount{
		ID:     accountID,
		Name:   "Acme Corp",
		Status: AccountStatusPending,
	}

	repo.On("GetAccount", ctx, accountID).Return(account, nil)
	repo.On("UpdateAccountStatus", ctx, accountID, AccountStatusActive).Return(nil)

	err := svc.ActivateAccount(ctx, accountID)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestActivateAccount_AlreadyActive(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	accountID := uuid.New()
	account := &CorporateAccount{
		ID:     accountID,
		Name:   "Acme Corp",
		Status: AccountStatusActive,
	}

	repo.On("GetAccount", ctx, accountID).Return(account, nil)

	err := svc.ActivateAccount(ctx, accountID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not pending activation")
	repo.AssertExpectations(t)
}

func TestSuspendAccount_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	accountID := uuid.New()

	repo.On("UpdateAccountStatus", ctx, accountID, AccountStatusSuspended).Return(nil)

	err := svc.SuspendAccount(ctx, accountID, "policy violation")

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestInviteEmployee_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	accountID := uuid.New()
	req := &InviteEmployeeRequest{
		Email:     "john@acme.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      EmployeeRoleUser,
	}

	// No existing employee found
	repo.On("GetEmployeeByEmail", ctx, accountID, "john@acme.com").Return(nil, errors.New("not found"))
	repo.On("CreateEmployee", ctx, mock.AnythingOfType("*corporate.CorporateEmployee")).Return(nil)

	emp, err := svc.InviteEmployee(ctx, accountID, req)

	require.NoError(t, err)
	require.NotNil(t, emp)
	assert.Equal(t, "john@acme.com", emp.Email)
	assert.Equal(t, "John", emp.FirstName)
	assert.Equal(t, "Doe", emp.LastName)
	assert.Equal(t, EmployeeRoleUser, emp.Role)
	assert.Equal(t, accountID, emp.CorporateAccountID)
	assert.False(t, emp.IsActive) // Not active until accepted
	assert.NotNil(t, emp.InvitedAt)
	repo.AssertExpectations(t)
}

func TestInviteEmployee_EmployeeAlreadyExists(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	accountID := uuid.New()
	req := &InviteEmployeeRequest{
		Email:     "john@acme.com",
		FirstName: "John",
		LastName:  "Doe",
	}

	existing := &CorporateEmployee{
		ID:    uuid.New(),
		Email: "john@acme.com",
	}

	repo.On("GetEmployeeByEmail", ctx, accountID, "john@acme.com").Return(existing, nil)

	emp, err := svc.InviteEmployee(ctx, accountID, req)

	require.Error(t, err)
	assert.Nil(t, emp)
	assert.Contains(t, err.Error(), "already exists")
	repo.AssertExpectations(t)
}

func TestCheckPolicies_RideAllowed(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	accountID := uuid.New()
	employeeID := uuid.New()
	deptID := uuid.New()

	emp := &CorporateEmployee{
		ID:                 employeeID,
		CorporateAccountID: accountID,
		DepartmentID:       &deptID,
		MonthlyUsed:        100,
		MonthlyLimit:       ptrFloat64(1000),
	}

	// No active policies
	policies := []*RidePolicy{}

	req := &BookCorporateRideRequest{
		RideType: "economy",
	}

	repo.On("GetEmployee", ctx, employeeID).Return(emp, nil)
	repo.On("GetPolicies", ctx, accountID, &deptID).Return(policies, nil)

	result, err := svc.CheckPolicies(ctx, accountID, employeeID, req, 50.0)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Allowed)
	assert.Empty(t, result.Violations)
	assert.False(t, result.RequiresApproval)
	repo.AssertExpectations(t)
}

func TestCheckPolicies_BudgetExceeded(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	accountID := uuid.New()
	employeeID := uuid.New()
	deptID := uuid.New()

	emp := &CorporateEmployee{
		ID:                 employeeID,
		CorporateAccountID: accountID,
		DepartmentID:       &deptID,
		MonthlyUsed:        950,
		MonthlyLimit:       ptrFloat64(1000),
	}

	// No active policies
	policies := []*RidePolicy{}

	req := &BookCorporateRideRequest{
		RideType: "economy",
	}

	repo.On("GetEmployee", ctx, employeeID).Return(emp, nil)
	repo.On("GetPolicies", ctx, accountID, &deptID).Return(policies, nil)

	result, err := svc.CheckPolicies(ctx, accountID, employeeID, req, 100.0)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Allowed)
	require.NotEmpty(t, result.Violations)

	found := false
	for _, v := range result.Violations {
		if v.PolicyName == "Monthly Limit" {
			found = true
			assert.Equal(t, "block", v.Severity)
			assert.Contains(t, v.Reason, "monthly limit")
		}
	}
	assert.True(t, found, "expected Monthly Limit violation")
	repo.AssertExpectations(t)
}

func TestCheckPolicies_TimeRestrictionViolated(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	accountID := uuid.New()
	employeeID := uuid.New()

	emp := &CorporateEmployee{
		ID:                 employeeID,
		CorporateAccountID: accountID,
		DepartmentID:       nil,
		MonthlyUsed:        0,
	}

	// Create a time restriction policy that only allows rides on a day that is NOT today
	now := time.Now()
	todayName := now.Weekday().String()

	// Pick a day that is different from today
	allowedDay := "saturday"
	if todayName == "Saturday" {
		allowedDay = "monday"
	}

	policyID := uuid.New()
	policies := []*RidePolicy{
		{
			ID:                 policyID,
			CorporateAccountID: accountID,
			Name:               "Weekday Only",
			PolicyType:         PolicyTypeTimeRestriction,
			Rules: PolicyRules{
				AllowedDays: []string{allowedDay},
			},
			IsActive: true,
		},
	}

	req := &BookCorporateRideRequest{
		RideType: "economy",
	}

	repo.On("GetEmployee", ctx, employeeID).Return(emp, nil)
	repo.On("GetPolicies", ctx, accountID, (*uuid.UUID)(nil)).Return(policies, nil)

	result, err := svc.CheckPolicies(ctx, accountID, employeeID, req, 30.0)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Allowed)
	require.NotEmpty(t, result.Violations)

	found := false
	for _, v := range result.Violations {
		if v.PolicyName == "Weekday Only" {
			found = true
			assert.Equal(t, "block", v.Severity)
			assert.Contains(t, v.Reason, "not allowed on")
		}
	}
	assert.True(t, found, "expected time restriction violation")
	repo.AssertExpectations(t)
}

func TestRecordCorporateRide_SuccessWithDiscount(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	accountID := uuid.New()
	employeeID := uuid.New()
	rideID := uuid.New()
	deptID := uuid.New()

	emp := &CorporateEmployee{
		ID:                 employeeID,
		CorporateAccountID: accountID,
		DepartmentID:       &deptID,
		RequireApproval:    false,
	}

	account := &CorporateAccount{
		ID:              accountID,
		DiscountPercent: 20,
		RequireApproval: false,
	}

	req := &BookCorporateRideRequest{
		RideType:    "economy",
		CostCenter:  ptrString("CC-001"),
		ProjectCode: ptrString("PROJ-100"),
		Purpose:     ptrString("Client meeting"),
	}

	repo.On("GetEmployee", ctx, employeeID).Return(emp, nil)
	repo.On("GetAccount", ctx, accountID).Return(account, nil)
	repo.On("CreateCorporateRide", ctx, mock.AnythingOfType("*corporate.CorporateRide")).Return(nil)
	repo.On("UpdateEmployeeUsage", ctx, employeeID, 80.0).Return(nil)
	repo.On("UpdateDepartmentBudget", ctx, deptID, 80.0).Return(nil)
	repo.On("UpdateAccountBalance", ctx, accountID, 80.0).Return(nil)

	ride, err := svc.RecordCorporateRide(ctx, rideID, employeeID, 100.0, req)

	require.NoError(t, err)
	require.NotNil(t, ride)
	assert.Equal(t, 100.0, ride.OriginalFare)
	assert.Equal(t, 20.0, ride.DiscountAmount)    // 20% of 100
	assert.Equal(t, 80.0, ride.FinalFare)          // 100 - 20
	assert.Equal(t, rideID, ride.RideID)
	assert.Equal(t, accountID, ride.CorporateAccountID)
	assert.Equal(t, employeeID, ride.EmployeeID)
	assert.Equal(t, "CC-001", *ride.CostCenter)
	assert.Equal(t, "PROJ-100", *ride.ProjectCode)
	assert.False(t, ride.RequiresApproval)
	require.NotNil(t, ride.ApprovalStatus)
	assert.Equal(t, "approved", *ride.ApprovalStatus)
	repo.AssertExpectations(t)
}

func TestGenerateInvoice_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	accountID := uuid.New()
	periodStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC)

	account := &CorporateAccount{
		ID:              accountID,
		PaymentTermDays: 30,
		DiscountPercent: 10,
	}

	rides := []*CorporateRide{
		{
			ID:             uuid.New(),
			OriginalFare:   100.0,
			DiscountAmount: 10.0,
			FinalFare:      90.0,
		},
		{
			ID:             uuid.New(),
			OriginalFare:   200.0,
			DiscountAmount: 20.0,
			FinalFare:      180.0,
		},
		{
			ID:             uuid.New(),
			OriginalFare:   50.0,
			DiscountAmount: 5.0,
			FinalFare:      45.0,
		},
	}

	repo.On("GetAccount", ctx, accountID).Return(account, nil)
	repo.On("ListCorporateRides", ctx, accountID, (*uuid.UUID)(nil), periodStart, periodEnd, 10000, 0).Return(rides, nil)
	repo.On("CreateInvoice", ctx, mock.AnythingOfType("*corporate.CorporateInvoice")).Return(nil)

	invoice, err := svc.GenerateInvoice(ctx, accountID, periodStart, periodEnd)

	require.NoError(t, err)
	require.NotNil(t, invoice)
	assert.Equal(t, 350.0, invoice.Subtotal)       // 100+200+50
	assert.Equal(t, 35.0, invoice.DiscountTotal)    // 10+20+5
	assert.Equal(t, 0.0, invoice.TaxAmount)         // taxRate is 0
	assert.Equal(t, 315.0, invoice.TotalAmount)     // 350-35+0
	assert.Equal(t, 3, invoice.RideCount)
	assert.Equal(t, "draft", invoice.Status)
	assert.Contains(t, invoice.InvoiceNumber, "INV-")
	repo.AssertExpectations(t)
}
