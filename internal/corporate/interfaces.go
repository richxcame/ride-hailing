package corporate

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RepositoryInterface defines the contract for corporate repository operations
type RepositoryInterface interface {
	// Account operations
	CreateAccount(ctx context.Context, account *CorporateAccount) error
	GetAccount(ctx context.Context, accountID uuid.UUID) (*CorporateAccount, error)
	UpdateAccount(ctx context.Context, account *CorporateAccount) error
	UpdateAccountStatus(ctx context.Context, accountID uuid.UUID, status AccountStatus) error
	UpdateAccountBalance(ctx context.Context, accountID uuid.UUID, amount float64) error
	ListAccounts(ctx context.Context, status *AccountStatus, limit, offset int) ([]*CorporateAccount, error)

	// Department operations
	CreateDepartment(ctx context.Context, dept *Department) error
	GetDepartment(ctx context.Context, deptID uuid.UUID) (*Department, error)
	ListDepartments(ctx context.Context, accountID uuid.UUID) ([]*Department, error)
	UpdateDepartmentBudget(ctx context.Context, deptID uuid.UUID, amount float64) error
	ResetDepartmentBudgets(ctx context.Context, accountID uuid.UUID) error

	// Employee operations
	CreateEmployee(ctx context.Context, emp *CorporateEmployee) error
	GetEmployee(ctx context.Context, empID uuid.UUID) (*CorporateEmployee, error)
	GetEmployeeByUserID(ctx context.Context, userID uuid.UUID) (*CorporateEmployee, error)
	GetEmployeeByEmail(ctx context.Context, accountID uuid.UUID, email string) (*CorporateEmployee, error)
	ListEmployees(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*CorporateEmployee, error)
	UpdateEmployeeUsage(ctx context.Context, empID uuid.UUID, amount float64) error
	ResetEmployeeUsage(ctx context.Context, accountID uuid.UUID) error
	GetEmployeeCount(ctx context.Context, accountID uuid.UUID, activeOnly bool) (int, error)

	// Policy operations
	CreatePolicy(ctx context.Context, policy *RidePolicy) error
	GetPolicies(ctx context.Context, accountID uuid.UUID, departmentID *uuid.UUID) ([]*RidePolicy, error)

	// Corporate ride operations
	CreateCorporateRide(ctx context.Context, ride *CorporateRide) error
	GetCorporateRide(ctx context.Context, rideID uuid.UUID) (*CorporateRide, error)
	ListCorporateRides(ctx context.Context, accountID uuid.UUID, employeeID *uuid.UUID, startDate, endDate time.Time, limit, offset int) ([]*CorporateRide, error)
	GetPendingApprovals(ctx context.Context, accountID uuid.UUID, approverID *uuid.UUID) ([]*CorporateRide, error)
	ApproveRide(ctx context.Context, rideID, approverID uuid.UUID, approved bool) error

	// Invoice operations
	CreateInvoice(ctx context.Context, invoice *CorporateInvoice) error
	GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*CorporateInvoice, error)
	ListInvoices(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*CorporateInvoice, error)

	// Statistics
	GetPeriodStats(ctx context.Context, accountID uuid.UUID, startDate, endDate time.Time) (*PeriodStats, error)
	GetTopSpenders(ctx context.Context, accountID uuid.UUID, startDate, endDate time.Time, limit int) ([]EmployeeSpending, error)
	GetDepartmentUsage(ctx context.Context, accountID uuid.UUID, startDate, endDate time.Time) ([]DepartmentUsage, error)
}
