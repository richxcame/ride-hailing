package mocks

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/internal/corporate"
	"github.com/stretchr/testify/mock"
)

// MockCorporateRepository is a mock implementation of the corporate RepositoryInterface
type MockCorporateRepository struct {
	mock.Mock
}

// ========================================
// ACCOUNT OPERATIONS
// ========================================

// CreateAccount mocks creating a corporate account
func (m *MockCorporateRepository) CreateAccount(ctx context.Context, account *corporate.CorporateAccount) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

// GetAccount mocks getting a corporate account by ID
func (m *MockCorporateRepository) GetAccount(ctx context.Context, accountID uuid.UUID) (*corporate.CorporateAccount, error) {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corporate.CorporateAccount), args.Error(1)
}

// UpdateAccount mocks updating a corporate account
func (m *MockCorporateRepository) UpdateAccount(ctx context.Context, account *corporate.CorporateAccount) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

// UpdateAccountStatus mocks updating account status
func (m *MockCorporateRepository) UpdateAccountStatus(ctx context.Context, accountID uuid.UUID, status corporate.AccountStatus) error {
	args := m.Called(ctx, accountID, status)
	return args.Error(0)
}

// UpdateAccountBalance mocks updating account balance
func (m *MockCorporateRepository) UpdateAccountBalance(ctx context.Context, accountID uuid.UUID, amount float64) error {
	args := m.Called(ctx, accountID, amount)
	return args.Error(0)
}

// ListAccounts mocks listing corporate accounts
func (m *MockCorporateRepository) ListAccounts(ctx context.Context, status *corporate.AccountStatus, limit, offset int) ([]*corporate.CorporateAccount, error) {
	args := m.Called(ctx, status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*corporate.CorporateAccount), args.Error(1)
}

// ========================================
// DEPARTMENT OPERATIONS
// ========================================

// CreateDepartment mocks creating a department
func (m *MockCorporateRepository) CreateDepartment(ctx context.Context, dept *corporate.Department) error {
	args := m.Called(ctx, dept)
	return args.Error(0)
}

// GetDepartment mocks getting a department by ID
func (m *MockCorporateRepository) GetDepartment(ctx context.Context, deptID uuid.UUID) (*corporate.Department, error) {
	args := m.Called(ctx, deptID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corporate.Department), args.Error(1)
}

// ListDepartments mocks listing departments for an account
func (m *MockCorporateRepository) ListDepartments(ctx context.Context, accountID uuid.UUID) ([]*corporate.Department, error) {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*corporate.Department), args.Error(1)
}

// UpdateDepartmentBudget mocks updating department budget usage
func (m *MockCorporateRepository) UpdateDepartmentBudget(ctx context.Context, deptID uuid.UUID, amount float64) error {
	args := m.Called(ctx, deptID, amount)
	return args.Error(0)
}

// ResetDepartmentBudgets mocks resetting all department budgets
func (m *MockCorporateRepository) ResetDepartmentBudgets(ctx context.Context, accountID uuid.UUID) error {
	args := m.Called(ctx, accountID)
	return args.Error(0)
}

// ========================================
// EMPLOYEE OPERATIONS
// ========================================

// CreateEmployee mocks creating an employee
func (m *MockCorporateRepository) CreateEmployee(ctx context.Context, emp *corporate.CorporateEmployee) error {
	args := m.Called(ctx, emp)
	return args.Error(0)
}

// GetEmployee mocks getting an employee by ID
func (m *MockCorporateRepository) GetEmployee(ctx context.Context, empID uuid.UUID) (*corporate.CorporateEmployee, error) {
	args := m.Called(ctx, empID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corporate.CorporateEmployee), args.Error(1)
}

// GetEmployeeByUserID mocks getting an employee by user ID
func (m *MockCorporateRepository) GetEmployeeByUserID(ctx context.Context, userID uuid.UUID) (*corporate.CorporateEmployee, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corporate.CorporateEmployee), args.Error(1)
}

// GetEmployeeByEmail mocks getting an employee by email
func (m *MockCorporateRepository) GetEmployeeByEmail(ctx context.Context, accountID uuid.UUID, email string) (*corporate.CorporateEmployee, error) {
	args := m.Called(ctx, accountID, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corporate.CorporateEmployee), args.Error(1)
}

// ListEmployees mocks listing employees for an account
func (m *MockCorporateRepository) ListEmployees(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*corporate.CorporateEmployee, error) {
	args := m.Called(ctx, accountID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*corporate.CorporateEmployee), args.Error(1)
}

// UpdateEmployeeUsage mocks updating employee's monthly usage
func (m *MockCorporateRepository) UpdateEmployeeUsage(ctx context.Context, empID uuid.UUID, amount float64) error {
	args := m.Called(ctx, empID, amount)
	return args.Error(0)
}

// ResetEmployeeUsage mocks resetting all employee monthly usage
func (m *MockCorporateRepository) ResetEmployeeUsage(ctx context.Context, accountID uuid.UUID) error {
	args := m.Called(ctx, accountID)
	return args.Error(0)
}

// GetEmployeeCount mocks getting the count of employees for an account
func (m *MockCorporateRepository) GetEmployeeCount(ctx context.Context, accountID uuid.UUID, activeOnly bool) (int, error) {
	args := m.Called(ctx, accountID, activeOnly)
	return args.Int(0), args.Error(1)
}

// ========================================
// POLICY OPERATIONS
// ========================================

// CreatePolicy mocks creating a policy
func (m *MockCorporateRepository) CreatePolicy(ctx context.Context, policy *corporate.RidePolicy) error {
	args := m.Called(ctx, policy)
	return args.Error(0)
}

// GetPolicies mocks getting policies for an account/department
func (m *MockCorporateRepository) GetPolicies(ctx context.Context, accountID uuid.UUID, departmentID *uuid.UUID) ([]*corporate.RidePolicy, error) {
	args := m.Called(ctx, accountID, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*corporate.RidePolicy), args.Error(1)
}

// ========================================
// CORPORATE RIDE OPERATIONS
// ========================================

// CreateCorporateRide mocks creating a corporate ride record
func (m *MockCorporateRepository) CreateCorporateRide(ctx context.Context, ride *corporate.CorporateRide) error {
	args := m.Called(ctx, ride)
	return args.Error(0)
}

// GetCorporateRide mocks getting a corporate ride by ID
func (m *MockCorporateRepository) GetCorporateRide(ctx context.Context, rideID uuid.UUID) (*corporate.CorporateRide, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corporate.CorporateRide), args.Error(1)
}

// ListCorporateRides mocks listing corporate rides with filters
func (m *MockCorporateRepository) ListCorporateRides(ctx context.Context, accountID uuid.UUID, employeeID *uuid.UUID, startDate, endDate time.Time, limit, offset int) ([]*corporate.CorporateRide, error) {
	args := m.Called(ctx, accountID, employeeID, startDate, endDate, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*corporate.CorporateRide), args.Error(1)
}

// GetPendingApprovals mocks getting rides pending approval
func (m *MockCorporateRepository) GetPendingApprovals(ctx context.Context, accountID uuid.UUID, approverID *uuid.UUID) ([]*corporate.CorporateRide, error) {
	args := m.Called(ctx, accountID, approverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*corporate.CorporateRide), args.Error(1)
}

// ApproveRide mocks approving or rejecting a ride
func (m *MockCorporateRepository) ApproveRide(ctx context.Context, rideID, approverID uuid.UUID, approved bool) error {
	args := m.Called(ctx, rideID, approverID, approved)
	return args.Error(0)
}

// ========================================
// INVOICE OPERATIONS
// ========================================

// CreateInvoice mocks creating an invoice
func (m *MockCorporateRepository) CreateInvoice(ctx context.Context, invoice *corporate.CorporateInvoice) error {
	args := m.Called(ctx, invoice)
	return args.Error(0)
}

// GetInvoice mocks getting an invoice by ID
func (m *MockCorporateRepository) GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*corporate.CorporateInvoice, error) {
	args := m.Called(ctx, invoiceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corporate.CorporateInvoice), args.Error(1)
}

// ListInvoices mocks listing invoices for an account
func (m *MockCorporateRepository) ListInvoices(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*corporate.CorporateInvoice, error) {
	args := m.Called(ctx, accountID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*corporate.CorporateInvoice), args.Error(1)
}

// ========================================
// STATISTICS
// ========================================

// GetPeriodStats mocks getting statistics for a billing period
func (m *MockCorporateRepository) GetPeriodStats(ctx context.Context, accountID uuid.UUID, startDate, endDate time.Time) (*corporate.PeriodStats, error) {
	args := m.Called(ctx, accountID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corporate.PeriodStats), args.Error(1)
}

// GetTopSpenders mocks getting top spending employees
func (m *MockCorporateRepository) GetTopSpenders(ctx context.Context, accountID uuid.UUID, startDate, endDate time.Time, limit int) ([]corporate.EmployeeSpending, error) {
	args := m.Called(ctx, accountID, startDate, endDate, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]corporate.EmployeeSpending), args.Error(1)
}

// GetDepartmentUsage mocks getting usage by department
func (m *MockCorporateRepository) GetDepartmentUsage(ctx context.Context, accountID uuid.UUID, startDate, endDate time.Time) ([]corporate.DepartmentUsage, error) {
	args := m.Called(ctx, accountID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]corporate.DepartmentUsage), args.Error(1)
}
