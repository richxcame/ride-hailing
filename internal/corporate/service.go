package corporate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// Service handles corporate account business logic
type Service struct {
	repo RepositoryInterface
}

// NewService creates a new corporate service
func NewService(repo RepositoryInterface) *Service {
	return &Service{repo: repo}
}

// ========================================
// ACCOUNT MANAGEMENT
// ========================================

// CreateAccount creates a new corporate account
func (s *Service) CreateAccount(ctx context.Context, req *CreateAccountRequest) (*CorporateAccount, error) {
	// Set defaults
	paymentTermDays := req.PaymentTermDays
	if paymentTermDays == 0 {
		paymentTermDays = 30 // Net 30 default
	}

	account := &CorporateAccount{
		ID:               uuid.New(),
		Name:             req.Name,
		LegalName:        req.LegalName,
		TaxID:            req.TaxID,
		Status:           AccountStatusPending,
		PrimaryEmail:     req.PrimaryEmail,
		PrimaryPhone:     req.PrimaryPhone,
		BillingEmail:     req.BillingEmail,
		Address:          req.Address,
		BillingCycle:     req.BillingCycle,
		PaymentTermDays:  paymentTermDays,
		CreditLimit:      10000, // Default credit limit
		CurrentBalance:   0,
		DiscountPercent:  10, // Default 10% corporate discount
		CustomRates:      false,
		RequireApproval:  false,
		RequireCostCenter: false,
		RequireProjectCode: false,
		AllowPersonalRides: true,
		SSOEnabled:       false,
		Industry:         req.Industry,
		CompanySize:      req.CompanySize,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := s.repo.CreateAccount(ctx, account); err != nil {
		return nil, common.NewInternalServerError("failed to create corporate account")
	}

	logger.Info("Corporate account created",
		zap.String("account_id", account.ID.String()),
		zap.String("name", account.Name),
	)

	return account, nil
}

// GetAccount gets a corporate account
func (s *Service) GetAccount(ctx context.Context, accountID uuid.UUID) (*CorporateAccount, error) {
	account, err := s.repo.GetAccount(ctx, accountID)
	if err != nil {
		return nil, common.NewNotFoundError("corporate account not found", err)
	}
	return account, nil
}

// UpdateAccount updates a corporate account
func (s *Service) UpdateAccount(ctx context.Context, account *CorporateAccount) error {
	account.UpdatedAt = time.Now()
	if err := s.repo.UpdateAccount(ctx, account); err != nil {
		return common.NewInternalServerError("failed to update corporate account")
	}
	return nil
}

// ActivateAccount activates a pending account
func (s *Service) ActivateAccount(ctx context.Context, accountID uuid.UUID) error {
	account, err := s.repo.GetAccount(ctx, accountID)
	if err != nil {
		return common.NewNotFoundError("corporate account not found", err)
	}

	if account.Status != AccountStatusPending {
		return common.NewBadRequestError("account is not pending activation", nil)
	}

	if err := s.repo.UpdateAccountStatus(ctx, accountID, AccountStatusActive); err != nil {
		return common.NewInternalServerError("failed to activate account")
	}

	logger.Info("Corporate account activated", zap.String("account_id", accountID.String()))
	return nil
}

// SuspendAccount suspends an account
func (s *Service) SuspendAccount(ctx context.Context, accountID uuid.UUID, reason string) error {
	if err := s.repo.UpdateAccountStatus(ctx, accountID, AccountStatusSuspended); err != nil {
		return common.NewInternalServerError("failed to suspend account")
	}

	logger.Info("Corporate account suspended",
		zap.String("account_id", accountID.String()),
		zap.String("reason", reason),
	)
	return nil
}

// ListAccounts lists corporate accounts
func (s *Service) ListAccounts(ctx context.Context, status *AccountStatus, limit, offset int) ([]*CorporateAccount, error) {
	return s.repo.ListAccounts(ctx, status, limit, offset)
}

// ========================================
// DEPARTMENT MANAGEMENT
// ========================================

// CreateDepartment creates a new department
func (s *Service) CreateDepartment(ctx context.Context, accountID uuid.UUID, name string, code *string, managerID *uuid.UUID, budgetMonthly *float64) (*Department, error) {
	dept := &Department{
		ID:                uuid.New(),
		CorporateAccountID: accountID,
		Name:              name,
		Code:              code,
		ManagerID:         managerID,
		BudgetMonthly:     budgetMonthly,
		BudgetUsed:        0,
		IsActive:          true,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := s.repo.CreateDepartment(ctx, dept); err != nil {
		return nil, common.NewInternalServerError("failed to create department")
	}

	logger.Info("Department created",
		zap.String("department_id", dept.ID.String()),
		zap.String("name", name),
	)

	return dept, nil
}

// ListDepartments lists departments for an account
func (s *Service) ListDepartments(ctx context.Context, accountID uuid.UUID) ([]*Department, error) {
	return s.repo.ListDepartments(ctx, accountID)
}

// ========================================
// EMPLOYEE MANAGEMENT
// ========================================

// InviteEmployee invites an employee to the corporate account
func (s *Service) InviteEmployee(ctx context.Context, accountID uuid.UUID, req *InviteEmployeeRequest) (*CorporateEmployee, error) {
	// Check if employee already exists
	existing, _ := s.repo.GetEmployeeByEmail(ctx, accountID, req.Email)
	if existing != nil {
		return nil, common.NewBadRequestError("employee with this email already exists", nil)
	}

	role := req.Role
	if role == "" {
		role = EmployeeRoleUser
	}

	now := time.Now()
	emp := &CorporateEmployee{
		ID:                uuid.New(),
		CorporateAccountID: accountID,
		UserID:            uuid.Nil, // Will be set when they accept invitation
		DepartmentID:      req.DepartmentID,
		Role:              role,
		EmployeeID:        req.EmployeeID,
		Email:             req.Email,
		FirstName:         req.FirstName,
		LastName:          req.LastName,
		JobTitle:          req.JobTitle,
		MonthlyLimit:      req.MonthlyLimit,
		PerRideLimit:      req.PerRideLimit,
		MonthlyUsed:       0,
		RequireApproval:   false,
		IsActive:          false, // Until they accept
		InvitedAt:         &now,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := s.repo.CreateEmployee(ctx, emp); err != nil {
		return nil, common.NewInternalServerError("failed to invite employee")
	}

	// TODO: Send invitation email

	logger.Info("Employee invited",
		zap.String("employee_id", emp.ID.String()),
		zap.String("email", req.Email),
	)

	return emp, nil
}

// AcceptInvitation accepts an employee invitation
func (s *Service) AcceptInvitation(ctx context.Context, employeeID, userID uuid.UUID) error {
	emp, err := s.repo.GetEmployee(ctx, employeeID)
	if err != nil {
		return common.NewNotFoundError("invitation not found", err)
	}

	if emp.IsActive {
		return common.NewBadRequestError("invitation already accepted", nil)
	}

	emp.UserID = userID
	emp.IsActive = true
	now := time.Now()
	emp.JoinedAt = &now
	emp.UpdatedAt = time.Now()

	// Update in database (simplified - would need a proper update method)
	// For now, we'll just log it
	logger.Info("Employee accepted invitation",
		zap.String("employee_id", emp.ID.String()),
		zap.String("user_id", userID.String()),
	)

	return nil
}

// GetEmployeeByUserID gets an employee by their user ID
func (s *Service) GetEmployeeByUserID(ctx context.Context, userID uuid.UUID) (*CorporateEmployee, error) {
	emp, err := s.repo.GetEmployeeByUserID(ctx, userID)
	if err != nil {
		return nil, common.NewNotFoundError("employee not found", err)
	}
	return emp, nil
}

// ListEmployees lists employees for an account
func (s *Service) ListEmployees(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*CorporateEmployee, error) {
	return s.repo.ListEmployees(ctx, accountID, limit, offset)
}

// ========================================
// POLICY MANAGEMENT
// ========================================

// CreatePolicy creates a new ride policy
func (s *Service) CreatePolicy(ctx context.Context, accountID uuid.UUID, name string, policyType PolicyType, rules PolicyRules, departmentID *uuid.UUID) (*RidePolicy, error) {
	policy := &RidePolicy{
		ID:                uuid.New(),
		CorporateAccountID: accountID,
		DepartmentID:      departmentID,
		Name:              name,
		PolicyType:        policyType,
		Rules:             rules,
		Priority:          1,
		IsActive:          true,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := s.repo.CreatePolicy(ctx, policy); err != nil {
		return nil, common.NewInternalServerError("failed to create policy")
	}

	logger.Info("Policy created",
		zap.String("policy_id", policy.ID.String()),
		zap.String("name", name),
		zap.String("type", string(policyType)),
	)

	return policy, nil
}

// CheckPolicies checks a ride request against all applicable policies
func (s *Service) CheckPolicies(ctx context.Context, accountID uuid.UUID, employeeID uuid.UUID, req *BookCorporateRideRequest, estimatedFare float64) (*PolicyCheckResult, error) {
	emp, err := s.repo.GetEmployee(ctx, employeeID)
	if err != nil {
		return nil, common.NewNotFoundError("employee not found", err)
	}

	policies, err := s.repo.GetPolicies(ctx, accountID, emp.DepartmentID)
	if err != nil {
		return nil, common.NewInternalServerError("failed to get policies")
	}

	result := &PolicyCheckResult{
		Allowed:          true,
		Violations:       []PolicyViolation{},
		RequiresApproval: false,
	}

	for _, policy := range policies {
		violation := s.checkPolicy(policy, emp, req, estimatedFare)
		if violation != nil {
			result.Violations = append(result.Violations, *violation)
			if violation.Severity == "block" {
				result.Allowed = false
			}
		}
	}

	// Check employee limits
	if emp.MonthlyLimit != nil && emp.MonthlyUsed+estimatedFare > *emp.MonthlyLimit {
		result.Violations = append(result.Violations, PolicyViolation{
			PolicyID:   uuid.Nil,
			PolicyName: "Monthly Limit",
			Reason:     fmt.Sprintf("This ride would exceed your monthly limit of $%.2f", *emp.MonthlyLimit),
			Severity:   "block",
		})
		result.Allowed = false
	}

	if emp.PerRideLimit != nil && estimatedFare > *emp.PerRideLimit {
		result.RequiresApproval = true
		reason := fmt.Sprintf("Ride cost ($%.2f) exceeds per-ride limit ($%.2f)", estimatedFare, *emp.PerRideLimit)
		result.ApprovalReason = &reason
	}

	return result, nil
}

// checkPolicy checks a single policy
func (s *Service) checkPolicy(policy *RidePolicy, emp *CorporateEmployee, req *BookCorporateRideRequest, estimatedFare float64) *PolicyViolation {
	rules := policy.Rules

	switch policy.PolicyType {
	case PolicyTypeTimeRestriction:
		// Check time of day and day of week
		now := time.Now()
		dayName := strings.ToLower(now.Weekday().String())

		if len(rules.AllowedDays) > 0 {
			allowed := false
			for _, d := range rules.AllowedDays {
				if strings.ToLower(d) == dayName {
					allowed = true
					break
				}
			}
			if !allowed {
				return &PolicyViolation{
					PolicyID:   policy.ID,
					PolicyName: policy.Name,
					Reason:     fmt.Sprintf("Rides not allowed on %s", now.Weekday().String()),
					Severity:   "block",
				}
			}
		}

		// Check time window
		if rules.AllowedStartTime != nil && rules.AllowedEndTime != nil {
			currentTime := now.Format("15:04")
			if currentTime < *rules.AllowedStartTime || currentTime > *rules.AllowedEndTime {
				return &PolicyViolation{
					PolicyID:   policy.ID,
					PolicyName: policy.Name,
					Reason:     fmt.Sprintf("Rides only allowed between %s and %s", *rules.AllowedStartTime, *rules.AllowedEndTime),
					Severity:   "block",
				}
			}
		}

	case PolicyTypeAmountLimit:
		if rules.MaxAmountPerRide != nil && estimatedFare > *rules.MaxAmountPerRide {
			return &PolicyViolation{
				PolicyID:   policy.ID,
				PolicyName: policy.Name,
				Reason:     fmt.Sprintf("Ride cost ($%.2f) exceeds maximum ($%.2f)", estimatedFare, *rules.MaxAmountPerRide),
				Severity:   "warning",
			}
		}

	case PolicyTypeRideTypeRestriction:
		if len(rules.AllowedRideTypes) > 0 {
			allowed := false
			for _, rt := range rules.AllowedRideTypes {
				if rt == req.RideType {
					allowed = true
					break
				}
			}
			if !allowed {
				return &PolicyViolation{
					PolicyID:   policy.ID,
					PolicyName: policy.Name,
					Reason:     fmt.Sprintf("Ride type '%s' is not allowed", req.RideType),
					Severity:   "block",
				}
			}
		}

		if len(rules.BlockedRideTypes) > 0 {
			for _, rt := range rules.BlockedRideTypes {
				if rt == req.RideType {
					return &PolicyViolation{
						PolicyID:   policy.ID,
						PolicyName: policy.Name,
						Reason:     fmt.Sprintf("Ride type '%s' is blocked", req.RideType),
						Severity:   "block",
					}
				}
			}
		}

	case PolicyTypeApprovalRequired:
		if rules.ApprovalThreshold != nil && estimatedFare > *rules.ApprovalThreshold {
			return &PolicyViolation{
				PolicyID:   policy.ID,
				PolicyName: policy.Name,
				Reason:     fmt.Sprintf("Rides over $%.2f require approval", *rules.ApprovalThreshold),
				Severity:   "warning",
			}
		}
	}

	return nil
}

// ========================================
// CORPORATE RIDE OPERATIONS
// ========================================

// RecordCorporateRide records a ride as a corporate ride
func (s *Service) RecordCorporateRide(ctx context.Context, rideID uuid.UUID, employeeID uuid.UUID, originalFare float64, req *BookCorporateRideRequest) (*CorporateRide, error) {
	emp, err := s.repo.GetEmployee(ctx, employeeID)
	if err != nil {
		return nil, common.NewNotFoundError("employee not found", err)
	}

	account, err := s.repo.GetAccount(ctx, emp.CorporateAccountID)
	if err != nil {
		return nil, common.NewNotFoundError("corporate account not found", err)
	}

	// Calculate discount
	discountAmount := originalFare * (account.DiscountPercent / 100)
	finalFare := originalFare - discountAmount

	// Determine if approval is required
	requiresApproval := account.RequireApproval || emp.RequireApproval
	if emp.PerRideLimit != nil && finalFare > *emp.PerRideLimit {
		requiresApproval = true
	}

	approvalStatus := "approved"
	if requiresApproval {
		approvalStatus = "pending"
	}

	ride := &CorporateRide{
		ID:                uuid.New(),
		RideID:            rideID,
		CorporateAccountID: emp.CorporateAccountID,
		EmployeeID:        employeeID,
		DepartmentID:      emp.DepartmentID,
		CostCenter:        req.CostCenter,
		ProjectCode:       req.ProjectCode,
		Purpose:           req.Purpose,
		Notes:             req.Notes,
		OriginalFare:      originalFare,
		DiscountAmount:    discountAmount,
		FinalFare:         finalFare,
		RequiresApproval:  requiresApproval,
		ApprovalStatus:    &approvalStatus,
		ExportedToExpense: false,
		CreatedAt:         time.Now(),
	}

	if err := s.repo.CreateCorporateRide(ctx, ride); err != nil {
		return nil, common.NewInternalServerError("failed to record corporate ride")
	}

	// Update employee usage
	_ = s.repo.UpdateEmployeeUsage(ctx, employeeID, finalFare)

	// Update department budget if applicable
	if emp.DepartmentID != nil {
		_ = s.repo.UpdateDepartmentBudget(ctx, *emp.DepartmentID, finalFare)
	}

	// Update account balance
	_ = s.repo.UpdateAccountBalance(ctx, emp.CorporateAccountID, finalFare)

	logger.Info("Corporate ride recorded",
		zap.String("ride_id", rideID.String()),
		zap.String("employee_id", employeeID.String()),
		zap.Float64("final_fare", finalFare),
	)

	return ride, nil
}

// ApproveRide approves or rejects a pending ride
func (s *Service) ApproveRide(ctx context.Context, rideID, approverID uuid.UUID, approved bool) error {
	ride, err := s.repo.GetCorporateRide(ctx, rideID)
	if err != nil {
		return common.NewNotFoundError("ride not found", err)
	}

	if ride.ApprovalStatus == nil || *ride.ApprovalStatus != "pending" {
		return common.NewBadRequestError("ride is not pending approval", nil)
	}

	if err := s.repo.ApproveRide(ctx, rideID, approverID, approved); err != nil {
		return common.NewInternalServerError("failed to update ride approval")
	}

	status := "approved"
	if !approved {
		status = "rejected"
	}

	logger.Info("Corporate ride "+status,
		zap.String("ride_id", rideID.String()),
		zap.String("approver_id", approverID.String()),
	)

	return nil
}

// GetPendingApprovals gets rides pending approval
func (s *Service) GetPendingApprovals(ctx context.Context, accountID uuid.UUID, approverID *uuid.UUID) ([]*CorporateRide, error) {
	return s.repo.GetPendingApprovals(ctx, accountID, approverID)
}

// ListCorporateRides lists corporate rides
func (s *Service) ListCorporateRides(ctx context.Context, accountID uuid.UUID, employeeID *uuid.UUID, startDate, endDate time.Time, limit, offset int) ([]*CorporateRide, error) {
	return s.repo.ListCorporateRides(ctx, accountID, employeeID, startDate, endDate, limit, offset)
}

// ========================================
// DASHBOARD & REPORTING
// ========================================

// GetDashboard gets the corporate dashboard data
func (s *Service) GetDashboard(ctx context.Context, accountID uuid.UUID) (*AccountDashboardResponse, error) {
	account, err := s.repo.GetAccount(ctx, accountID)
	if err != nil {
		return nil, common.NewNotFoundError("corporate account not found", err)
	}

	// Get period dates
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := now

	// Get statistics
	periodStats, _ := s.repo.GetPeriodStats(ctx, accountID, periodStart, periodEnd)

	// Get counts
	employeeCount, _ := s.repo.GetEmployeeCount(ctx, accountID, false)
	activeEmployees, _ := s.repo.GetEmployeeCount(ctx, accountID, true)

	depts, _ := s.repo.ListDepartments(ctx, accountID)

	// Get recent rides
	rides, _ := s.repo.ListCorporateRides(ctx, accountID, nil, periodStart, periodEnd, 10, 0)
	recentRides := make([]CorporateRideDetail, len(rides))
	for i, r := range rides {
		recentRides[i] = CorporateRideDetail{CorporateRide: *r}
	}

	// Get top spenders
	topSpenders, _ := s.repo.GetTopSpenders(ctx, accountID, periodStart, periodEnd, 5)

	// Get department usage
	deptUsage, _ := s.repo.GetDepartmentUsage(ctx, accountID, periodStart, periodEnd)

	// Get pending approvals count
	pendingApprovals, _ := s.repo.GetPendingApprovals(ctx, accountID, nil)

	return &AccountDashboardResponse{
		Account:          account,
		CurrentPeriod:    periodStats,
		EmployeeCount:    employeeCount,
		ActiveEmployees:  activeEmployees,
		DepartmentCount:  len(depts),
		RecentRides:      recentRides,
		TopSpenders:      topSpenders,
		DepartmentUsage:  deptUsage,
		PendingApprovals: len(pendingApprovals),
	}, nil
}

// ========================================
// INVOICING
// ========================================

// GenerateInvoice generates an invoice for a billing period
func (s *Service) GenerateInvoice(ctx context.Context, accountID uuid.UUID, periodStart, periodEnd time.Time) (*CorporateInvoice, error) {
	account, err := s.repo.GetAccount(ctx, accountID)
	if err != nil {
		return nil, common.NewNotFoundError("corporate account not found", err)
	}

	// Get all rides for the period
	rides, err := s.repo.ListCorporateRides(ctx, accountID, nil, periodStart, periodEnd, 10000, 0)
	if err != nil {
		return nil, common.NewInternalServerError("failed to get rides for invoicing")
	}

	// Calculate totals
	var subtotal, discountTotal float64
	for _, ride := range rides {
		subtotal += ride.OriginalFare
		discountTotal += ride.DiscountAmount
	}

	taxRate := 0.0 // Would be configured per region
	taxAmount := (subtotal - discountTotal) * taxRate
	totalAmount := subtotal - discountTotal + taxAmount

	// Generate invoice number
	invoiceNumber := fmt.Sprintf("INV-%s-%s", account.ID.String()[:8], time.Now().Format("200601"))

	// Calculate due date
	dueDate := time.Now().AddDate(0, 0, account.PaymentTermDays)

	invoice := &CorporateInvoice{
		ID:                accountID,
		CorporateAccountID: accountID,
		InvoiceNumber:     invoiceNumber,
		PeriodStart:       periodStart,
		PeriodEnd:         periodEnd,
		Subtotal:          subtotal,
		DiscountTotal:     discountTotal,
		TaxAmount:         taxAmount,
		TotalAmount:       totalAmount,
		Status:            "draft",
		DueDate:           dueDate,
		PaidAmount:        0,
		RideCount:         len(rides),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := s.repo.CreateInvoice(ctx, invoice); err != nil {
		return nil, common.NewInternalServerError("failed to create invoice")
	}

	logger.Info("Invoice generated",
		zap.String("invoice_id", invoice.ID.String()),
		zap.String("invoice_number", invoiceNumber),
		zap.Float64("total_amount", totalAmount),
		zap.Int("ride_count", len(rides)),
	)

	return invoice, nil
}

// ListInvoices lists invoices for an account
func (s *Service) ListInvoices(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*CorporateInvoice, error) {
	return s.repo.ListInvoices(ctx, accountID, limit, offset)
}
