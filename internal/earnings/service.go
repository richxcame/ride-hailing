package earnings

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/richxcame/ride-hailing/pkg/common"
)

const (
	defaultCommissionRate = 0.20 // 20% platform commission
	minPayoutAmount       = 5.0
	defaultCurrency       = "USD"
)

// Service handles earnings business logic
type Service struct {
	repo *Repository
}

// NewService creates a new earnings service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ========================================
// RECORDING EARNINGS
// ========================================

// RecordRideEarning records earnings from a completed ride
func (s *Service) RecordRideEarning(ctx context.Context, driverID uuid.UUID, rideID uuid.UUID, grossAmount float64, commissionRate *float64) (*DriverEarning, error) {
	rate := defaultCommissionRate
	if commissionRate != nil {
		rate = *commissionRate
	}

	commission := grossAmount * rate
	netAmount := grossAmount - commission

	earning := &DriverEarning{
		ID:          uuid.New(),
		DriverID:    driverID,
		RideID:      &rideID,
		Type:        EarningTypeRideFare,
		GrossAmount: grossAmount,
		Commission:  commission,
		NetAmount:   netAmount,
		Currency:    defaultCurrency,
		Description: "Ride fare",
		IsPaidOut:   false,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.CreateEarning(ctx, earning); err != nil {
		return nil, fmt.Errorf("record ride earning: %w", err)
	}

	// Update earning goals
	go s.updateGoals(context.Background(), driverID, netAmount)

	return earning, nil
}

// RecordTip records a tip for a driver
func (s *Service) RecordTip(ctx context.Context, driverID uuid.UUID, rideID uuid.UUID, amount float64) (*DriverEarning, error) {
	if amount <= 0 {
		return nil, common.NewBadRequestError("tip amount must be positive", nil)
	}

	earning := &DriverEarning{
		ID:          uuid.New(),
		DriverID:    driverID,
		RideID:      &rideID,
		Type:        EarningTypeTip,
		GrossAmount: amount,
		Commission:  0, // No commission on tips
		NetAmount:   amount,
		Currency:    defaultCurrency,
		Description: "Tip",
		IsPaidOut:   false,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.CreateEarning(ctx, earning); err != nil {
		return nil, fmt.Errorf("record tip: %w", err)
	}

	go s.updateGoals(context.Background(), driverID, amount)

	return earning, nil
}

// RecordBonus records a bonus earning
func (s *Service) RecordBonus(ctx context.Context, driverID uuid.UUID, amount float64, description string) (*DriverEarning, error) {
	earning := &DriverEarning{
		ID:          uuid.New(),
		DriverID:    driverID,
		Type:        EarningTypeBonus,
		GrossAmount: amount,
		Commission:  0,
		NetAmount:   amount,
		Currency:    defaultCurrency,
		Description: description,
		IsPaidOut:   false,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.CreateEarning(ctx, earning); err != nil {
		return nil, fmt.Errorf("record bonus: %w", err)
	}

	go s.updateGoals(context.Background(), driverID, amount)

	return earning, nil
}

// RecordSurge records surge earnings
func (s *Service) RecordSurge(ctx context.Context, driverID uuid.UUID, rideID uuid.UUID, surgeAmount float64) (*DriverEarning, error) {
	earning := &DriverEarning{
		ID:          uuid.New(),
		DriverID:    driverID,
		RideID:      &rideID,
		Type:        EarningTypeSurge,
		GrossAmount: surgeAmount,
		Commission:  0,
		NetAmount:   surgeAmount,
		Currency:    defaultCurrency,
		Description: "Surge bonus",
		IsPaidOut:   false,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.CreateEarning(ctx, earning); err != nil {
		return nil, fmt.Errorf("record surge: %w", err)
	}

	return earning, nil
}

// RecordDeliveryEarning records earnings from a delivery
func (s *Service) RecordDeliveryEarning(ctx context.Context, driverID uuid.UUID, deliveryID uuid.UUID, grossAmount float64) (*DriverEarning, error) {
	commission := grossAmount * defaultCommissionRate
	netAmount := grossAmount - commission

	earning := &DriverEarning{
		ID:          uuid.New(),
		DriverID:    driverID,
		DeliveryID:  &deliveryID,
		Type:        EarningTypeDelivery,
		GrossAmount: grossAmount,
		Commission:  commission,
		NetAmount:   netAmount,
		Currency:    defaultCurrency,
		Description: "Delivery fare",
		IsPaidOut:   false,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.CreateEarning(ctx, earning); err != nil {
		return nil, fmt.Errorf("record delivery earning: %w", err)
	}

	go s.updateGoals(context.Background(), driverID, netAmount)

	return earning, nil
}

// ========================================
// DASHBOARD
// ========================================

// GetEarningsSummary returns earnings summary for a period
func (s *Service) GetEarningsSummary(ctx context.Context, driverID uuid.UUID, period string) (*EarningsSummary, error) {
	from, to, err := s.periodToTimeRange(period)
	if err != nil {
		return nil, err
	}

	summary, err := s.repo.GetEarningsSummary(ctx, driverID, from, to)
	if err != nil {
		return nil, err
	}

	summary.Period = period

	// Get breakdown
	breakdown, err := s.repo.GetEarningsBreakdown(ctx, driverID, from, to)
	if err == nil {
		summary.Breakdown = breakdown
	}
	if summary.Breakdown == nil {
		summary.Breakdown = []EarningBreakdown{}
	}

	return summary, nil
}

// GetDailyEarnings returns day-by-day earnings
func (s *Service) GetDailyEarnings(ctx context.Context, driverID uuid.UUID, period string) ([]DailyEarning, error) {
	from, to, err := s.periodToTimeRange(period)
	if err != nil {
		return nil, err
	}

	daily, err := s.repo.GetDailyEarnings(ctx, driverID, from, to)
	if err != nil {
		return nil, err
	}
	if daily == nil {
		daily = []DailyEarning{}
	}

	return daily, nil
}

// GetEarningsHistory returns paginated earnings
func (s *Service) GetEarningsHistory(ctx context.Context, driverID uuid.UUID, period string, page, pageSize int) (*EarningsHistoryResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	from, to, err := s.periodToTimeRange(period)
	if err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	earnings, total, err := s.repo.GetEarningsByDriver(ctx, driverID, from, to, pageSize, offset)
	if err != nil {
		return nil, err
	}
	if earnings == nil {
		earnings = []DriverEarning{}
	}

	return &EarningsHistoryResponse{
		Earnings: earnings,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// ========================================
// PAYOUTS
// ========================================

// RequestPayout requests a payout for the driver
func (s *Service) RequestPayout(ctx context.Context, driverID uuid.UUID, req *RequestPayoutRequest) (*DriverPayout, error) {
	// Get unpaid total
	unpaidTotal, err := s.repo.GetUnpaidEarningsTotal(ctx, driverID)
	if err != nil {
		return nil, fmt.Errorf("get unpaid total: %w", err)
	}

	if unpaidTotal < minPayoutAmount {
		return nil, common.NewBadRequestError(
			fmt.Sprintf("minimum payout amount is %.2f, current balance: %.2f", minPayoutAmount, unpaidTotal),
			nil,
		)
	}

	// Get bank account
	if req.Method == PayoutMethodBankTransfer {
		var ba *DriverBankAccount
		if req.BankAccountID != nil {
			accounts, _ := s.repo.GetBankAccounts(ctx, driverID)
			for i := range accounts {
				if accounts[i].ID == *req.BankAccountID {
					ba = &accounts[i]
					break
				}
			}
		} else {
			ba, err = s.repo.GetPrimaryBankAccount(ctx, driverID)
			if err != nil && err != pgx.ErrNoRows {
				return nil, err
			}
		}
		if ba == nil {
			return nil, common.NewBadRequestError("no bank account found for payout", nil)
		}
	}

	now := time.Now()
	payout := &DriverPayout{
		ID:        uuid.New(),
		DriverID:  driverID,
		Amount:    unpaidTotal,
		Currency:  defaultCurrency,
		Method:    req.Method,
		Status:    PayoutStatusPending,
		Reference: generatePayoutReference(),
		PeriodEnd: now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Set period start to earliest unpaid earning
	payout.PeriodStart = now.AddDate(0, -1, 0) // Fallback to 1 month ago

	if err := s.repo.CreatePayout(ctx, payout); err != nil {
		return nil, fmt.Errorf("create payout: %w", err)
	}

	// Mark earnings as paid
	count, err := s.repo.MarkEarningsAsPaidOut(ctx, driverID, payout.ID, now)
	if err != nil {
		return nil, fmt.Errorf("mark earnings paid: %w", err)
	}
	payout.EarningCount = int(count)

	return payout, nil
}

// GetPayoutHistory returns payout history
func (s *Service) GetPayoutHistory(ctx context.Context, driverID uuid.UUID, page, pageSize int) (*PayoutHistoryResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	payouts, total, err := s.repo.GetPayoutsByDriver(ctx, driverID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	if payouts == nil {
		payouts = []DriverPayout{}
	}

	return &PayoutHistoryResponse{
		Payouts:  payouts,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetUnpaidBalance returns unpaid balance
func (s *Service) GetUnpaidBalance(ctx context.Context, driverID uuid.UUID) (float64, error) {
	return s.repo.GetUnpaidEarningsTotal(ctx, driverID)
}

// ========================================
// BANK ACCOUNTS
// ========================================

// AddBankAccount adds a bank account
func (s *Service) AddBankAccount(ctx context.Context, driverID uuid.UUID, req *AddBankAccountRequest) (*DriverBankAccount, error) {
	currency := defaultCurrency
	if req.Currency != "" {
		currency = req.Currency
	}

	now := time.Now()
	ba := &DriverBankAccount{
		ID:            uuid.New(),
		DriverID:      driverID,
		BankName:      req.BankName,
		AccountHolder: req.AccountHolder,
		AccountNumber: req.AccountNumber,
		RoutingNumber: req.RoutingNumber,
		IBAN:          req.IBAN,
		SwiftCode:     req.SwiftCode,
		Currency:      currency,
		IsPrimary:     req.IsPrimary,
		IsVerified:    false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.AddBankAccount(ctx, ba); err != nil {
		return nil, fmt.Errorf("add bank account: %w", err)
	}

	// Mask the account number in response
	if len(ba.AccountNumber) > 4 {
		ba.AccountNumber = "****" + ba.AccountNumber[len(ba.AccountNumber)-4:]
	}

	return ba, nil
}

// GetBankAccounts returns bank accounts
func (s *Service) GetBankAccounts(ctx context.Context, driverID uuid.UUID) ([]DriverBankAccount, error) {
	accounts, err := s.repo.GetBankAccounts(ctx, driverID)
	if err != nil {
		return nil, err
	}
	if accounts == nil {
		accounts = []DriverBankAccount{}
	}
	return accounts, nil
}

// DeleteBankAccount removes a bank account
func (s *Service) DeleteBankAccount(ctx context.Context, driverID uuid.UUID, accountID uuid.UUID) error {
	return s.repo.DeleteBankAccount(ctx, accountID, driverID)
}

// ========================================
// EARNING GOALS
// ========================================

// SetEarningGoal sets an earning goal
func (s *Service) SetEarningGoal(ctx context.Context, driverID uuid.UUID, req *SetEarningGoalRequest) (*EarningGoal, error) {
	if req.TargetAmount <= 0 {
		return nil, common.NewBadRequestError("target amount must be positive", nil)
	}
	if req.Period != "daily" && req.Period != "weekly" && req.Period != "monthly" {
		return nil, common.NewBadRequestError("period must be daily, weekly, or monthly", nil)
	}

	now := time.Now()
	goal := &EarningGoal{
		ID:           uuid.New(),
		DriverID:     driverID,
		TargetAmount: req.TargetAmount,
		Period:       req.Period,
		Currency:     defaultCurrency,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.SetEarningGoal(ctx, goal); err != nil {
		return nil, fmt.Errorf("set earning goal: %w", err)
	}

	return goal, nil
}

// GetEarningGoals returns active earning goals with progress
func (s *Service) GetEarningGoals(ctx context.Context, driverID uuid.UUID) ([]EarningGoalStatus, error) {
	goals, err := s.repo.GetActiveGoals(ctx, driverID)
	if err != nil {
		return nil, err
	}

	var statuses []EarningGoalStatus
	for i := range goals {
		goal := &goals[i]
		progressPct := 0.0
		if goal.TargetAmount > 0 {
			progressPct = (goal.CurrentAmount / goal.TargetAmount) * 100
			if progressPct > 100 {
				progressPct = 100
			}
		}

		remaining := goal.TargetAmount - goal.CurrentAmount
		if remaining < 0 {
			remaining = 0
		}

		statuses = append(statuses, EarningGoalStatus{
			Goal:          goal,
			CurrentAmount: goal.CurrentAmount,
			ProgressPct:   progressPct,
			Remaining:     remaining,
			OnTrack:       goal.CurrentAmount >= goal.TargetAmount*s.expectedProgressFraction(goal.Period),
		})
	}

	if statuses == nil {
		statuses = []EarningGoalStatus{}
	}

	return statuses, nil
}

// ========================================
// HELPERS
// ========================================

func (s *Service) periodToTimeRange(period string) (time.Time, time.Time, error) {
	now := time.Now()
	to := now

	switch period {
	case "today":
		from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return from, to, nil
	case "yesterday":
		yesterday := now.AddDate(0, 0, -1)
		from := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, now.Location())
		to = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return from, to, nil
	case "this_week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		from := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		return from, to, nil
	case "last_week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		to = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		from := to.AddDate(0, 0, -7)
		return from, to, nil
	case "this_month":
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return from, to, nil
	case "last_month":
		firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		to = firstOfMonth
		from := firstOfMonth.AddDate(0, -1, 0)
		return from, to, nil
	default:
		return time.Time{}, time.Time{}, common.NewBadRequestError(
			"period must be: today, yesterday, this_week, last_week, this_month, or last_month", nil,
		)
	}
}

func (s *Service) expectedProgressFraction(period string) float64 {
	now := time.Now()
	switch period {
	case "daily":
		return float64(now.Hour()) / 24.0
	case "weekly":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		return float64(weekday) / 7.0
	case "monthly":
		daysInMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day()
		return float64(now.Day()) / float64(daysInMonth)
	default:
		return 0.5
	}
}

func (s *Service) updateGoals(ctx context.Context, driverID uuid.UUID, amount float64) {
	goals, err := s.repo.GetActiveGoals(ctx, driverID)
	if err != nil {
		return
	}
	for _, goal := range goals {
		s.repo.UpdateGoalProgress(ctx, goal.ID, amount)
	}
}

// generatePayoutReference creates a unique payout reference
func generatePayoutReference() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	ref := make([]byte, 10)
	for i := range ref {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		ref[i] = chars[n.Int64()]
	}
	return fmt.Sprintf("PAY-%s", string(ref))
}
