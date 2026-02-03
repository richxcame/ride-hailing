package earnings

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles earnings data access
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new earnings repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// EARNINGS
// ========================================

// CreateEarning records a new earning
func (r *Repository) CreateEarning(ctx context.Context, e *DriverEarning) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO driver_earnings (
			id, driver_id, ride_id, delivery_id, type,
			gross_amount, commission, net_amount, currency,
			description, is_paid_out, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		e.ID, e.DriverID, e.RideID, e.DeliveryID, e.Type,
		e.GrossAmount, e.Commission, e.NetAmount, e.Currency,
		e.Description, e.IsPaidOut, e.CreatedAt,
	)
	return err
}

// GetEarningsByDriver returns earnings for a driver in a time range
func (r *Repository) GetEarningsByDriver(ctx context.Context, driverID uuid.UUID, from, to time.Time, limit, offset int) ([]DriverEarning, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM driver_earnings
		WHERE driver_id = $1 AND created_at >= $2 AND created_at < $3`,
		driverID, from, to,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, driver_id, ride_id, delivery_id, type,
			gross_amount, commission, net_amount, currency,
			description, is_paid_out, payout_id, created_at
		FROM driver_earnings
		WHERE driver_id = $1 AND created_at >= $2 AND created_at < $3
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5`,
		driverID, from, to, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var earnings []DriverEarning
	for rows.Next() {
		e := DriverEarning{}
		if err := rows.Scan(
			&e.ID, &e.DriverID, &e.RideID, &e.DeliveryID, &e.Type,
			&e.GrossAmount, &e.Commission, &e.NetAmount, &e.Currency,
			&e.Description, &e.IsPaidOut, &e.PayoutID, &e.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		earnings = append(earnings, e)
	}
	return earnings, total, nil
}

// GetEarningsSummary returns aggregated earnings for a period
func (r *Repository) GetEarningsSummary(ctx context.Context, driverID uuid.UUID, from, to time.Time) (*EarningsSummary, error) {
	summary := &EarningsSummary{
		DriverID:    driverID,
		PeriodStart: from,
		PeriodEnd:   to,
		Currency:    "USD",
	}

	// Main aggregation
	err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(gross_amount), 0),
			COALESCE(SUM(commission), 0),
			COALESCE(SUM(net_amount), 0),
			COALESCE(SUM(CASE WHEN type = 'tip' THEN net_amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = 'bonus' THEN net_amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = 'surge' THEN net_amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = 'wait_time' THEN net_amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = 'delivery' THEN net_amount ELSE 0 END), 0),
			COUNT(CASE WHEN type = 'ride_fare' THEN 1 END),
			COUNT(CASE WHEN type = 'delivery' THEN 1 END)
		FROM driver_earnings
		WHERE driver_id = $1 AND created_at >= $2 AND created_at < $3`,
		driverID, from, to,
	).Scan(
		&summary.GrossEarnings,
		&summary.TotalCommission,
		&summary.NetEarnings,
		&summary.TipEarnings,
		&summary.BonusEarnings,
		&summary.SurgeEarnings,
		&summary.WaitTimeEarnings,
		&summary.DeliveryEarnings,
		&summary.RideCount,
		&summary.DeliveryCount,
	)
	if err != nil {
		return nil, err
	}

	return summary, nil
}

// GetEarningsBreakdown returns earnings grouped by type
func (r *Repository) GetEarningsBreakdown(ctx context.Context, driverID uuid.UUID, from, to time.Time) ([]EarningBreakdown, error) {
	rows, err := r.db.Query(ctx, `
		SELECT type, COALESCE(SUM(net_amount), 0), COUNT(*)
		FROM driver_earnings
		WHERE driver_id = $1 AND created_at >= $2 AND created_at < $3
		GROUP BY type
		ORDER BY SUM(net_amount) DESC`,
		driverID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var breakdown []EarningBreakdown
	for rows.Next() {
		b := EarningBreakdown{}
		if err := rows.Scan(&b.Type, &b.Amount, &b.Count); err != nil {
			return nil, err
		}
		breakdown = append(breakdown, b)
	}
	return breakdown, nil
}

// GetDailyEarnings returns day-by-day earnings
func (r *Repository) GetDailyEarnings(ctx context.Context, driverID uuid.UUID, from, to time.Time) ([]DailyEarning, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			to_char(created_at, 'YYYY-MM-DD') as day,
			COALESCE(SUM(gross_amount), 0),
			COALESCE(SUM(net_amount), 0),
			COALESCE(SUM(commission), 0),
			COALESCE(SUM(CASE WHEN type = 'tip' THEN net_amount ELSE 0 END), 0),
			COUNT(CASE WHEN type = 'ride_fare' THEN 1 END)
		FROM driver_earnings
		WHERE driver_id = $1 AND created_at >= $2 AND created_at < $3
		GROUP BY day
		ORDER BY day DESC`,
		driverID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var daily []DailyEarning
	for rows.Next() {
		d := DailyEarning{}
		if err := rows.Scan(
			&d.Date, &d.GrossAmount, &d.NetAmount,
			&d.Commission, &d.Tips, &d.RideCount,
		); err != nil {
			return nil, err
		}
		daily = append(daily, d)
	}
	return daily, nil
}

// GetUnpaidEarningsTotal returns total unpaid earnings
func (r *Repository) GetUnpaidEarningsTotal(ctx context.Context, driverID uuid.UUID) (float64, error) {
	var total float64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(net_amount), 0)
		FROM driver_earnings
		WHERE driver_id = $1 AND is_paid_out = false`,
		driverID,
	).Scan(&total)
	return total, err
}

// MarkEarningsAsPaidOut marks earnings as paid for a payout
func (r *Repository) MarkEarningsAsPaidOut(ctx context.Context, driverID uuid.UUID, payoutID uuid.UUID, before time.Time) (int64, error) {
	result, err := r.db.Exec(ctx, `
		UPDATE driver_earnings
		SET is_paid_out = true, payout_id = $2
		WHERE driver_id = $1 AND is_paid_out = false AND created_at < $3`,
		driverID, payoutID, before,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// ========================================
// PAYOUTS
// ========================================

// CreatePayout creates a new payout record
func (r *Repository) CreatePayout(ctx context.Context, p *DriverPayout) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO driver_payouts (
			id, driver_id, amount, currency, method, status,
			bank_account_id, reference, earning_count,
			period_start, period_end, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		p.ID, p.DriverID, p.Amount, p.Currency, p.Method, p.Status,
		p.BankAccountID, p.Reference, p.EarningCount,
		p.PeriodStart, p.PeriodEnd, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

// GetPayoutsByDriver returns payouts for a driver
func (r *Repository) GetPayoutsByDriver(ctx context.Context, driverID uuid.UUID, limit, offset int) ([]DriverPayout, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM driver_payouts WHERE driver_id = $1`,
		driverID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, driver_id, amount, currency, method, status,
			bank_account_id, reference, earning_count,
			period_start, period_end, processed_at,
			failure_reason, created_at, updated_at
		FROM driver_payouts
		WHERE driver_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		driverID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var payouts []DriverPayout
	for rows.Next() {
		p := DriverPayout{}
		if err := rows.Scan(
			&p.ID, &p.DriverID, &p.Amount, &p.Currency, &p.Method, &p.Status,
			&p.BankAccountID, &p.Reference, &p.EarningCount,
			&p.PeriodStart, &p.PeriodEnd, &p.ProcessedAt,
			&p.FailureReason, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		payouts = append(payouts, p)
	}
	return payouts, total, nil
}

// UpdatePayoutStatus updates a payout status
func (r *Repository) UpdatePayoutStatus(ctx context.Context, payoutID uuid.UUID, status PayoutStatus, failureReason *string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE driver_payouts
		SET status = $2, failure_reason = $3, processed_at = NOW(), updated_at = NOW()
		WHERE id = $1`,
		payoutID, status, failureReason,
	)
	return err
}

// ========================================
// BANK ACCOUNTS
// ========================================

// AddBankAccount adds a driver bank account
func (r *Repository) AddBankAccount(ctx context.Context, ba *DriverBankAccount) error {
	// If primary, unset other primary accounts
	if ba.IsPrimary {
		r.db.Exec(ctx, `
			UPDATE driver_bank_accounts SET is_primary = false
			WHERE driver_id = $1`, ba.DriverID)
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO driver_bank_accounts (
			id, driver_id, bank_name, account_holder,
			account_number, routing_number, iban, swift_code,
			currency, is_primary, is_verified, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		ba.ID, ba.DriverID, ba.BankName, ba.AccountHolder,
		ba.AccountNumber, ba.RoutingNumber, ba.IBAN, ba.SwiftCode,
		ba.Currency, ba.IsPrimary, ba.IsVerified, ba.CreatedAt, ba.UpdatedAt,
	)
	return err
}

// GetBankAccounts returns all bank accounts for a driver
func (r *Repository) GetBankAccounts(ctx context.Context, driverID uuid.UUID) ([]DriverBankAccount, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, driver_id, bank_name, account_holder,
			account_number, routing_number, iban, swift_code,
			currency, is_primary, is_verified, created_at, updated_at
		FROM driver_bank_accounts
		WHERE driver_id = $1
		ORDER BY is_primary DESC, created_at DESC`,
		driverID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []DriverBankAccount
	for rows.Next() {
		ba := DriverBankAccount{}
		if err := rows.Scan(
			&ba.ID, &ba.DriverID, &ba.BankName, &ba.AccountHolder,
			&ba.AccountNumber, &ba.RoutingNumber, &ba.IBAN, &ba.SwiftCode,
			&ba.Currency, &ba.IsPrimary, &ba.IsVerified, &ba.CreatedAt, &ba.UpdatedAt,
		); err != nil {
			return nil, err
		}
		// Mask account number
		if len(ba.AccountNumber) > 4 {
			ba.AccountNumber = "****" + ba.AccountNumber[len(ba.AccountNumber)-4:]
		}
		accounts = append(accounts, ba)
	}
	return accounts, nil
}

// GetPrimaryBankAccount returns the primary bank account
func (r *Repository) GetPrimaryBankAccount(ctx context.Context, driverID uuid.UUID) (*DriverBankAccount, error) {
	ba := &DriverBankAccount{}
	err := r.db.QueryRow(ctx, `
		SELECT id, driver_id, bank_name, account_holder,
			account_number, routing_number, iban, swift_code,
			currency, is_primary, is_verified, created_at, updated_at
		FROM driver_bank_accounts
		WHERE driver_id = $1 AND is_primary = true`,
		driverID,
	).Scan(
		&ba.ID, &ba.DriverID, &ba.BankName, &ba.AccountHolder,
		&ba.AccountNumber, &ba.RoutingNumber, &ba.IBAN, &ba.SwiftCode,
		&ba.Currency, &ba.IsPrimary, &ba.IsVerified, &ba.CreatedAt, &ba.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return ba, nil
}

// DeleteBankAccount removes a bank account
func (r *Repository) DeleteBankAccount(ctx context.Context, accountID uuid.UUID, driverID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM driver_bank_accounts
		WHERE id = $1 AND driver_id = $2`,
		accountID, driverID,
	)
	return err
}

// ========================================
// EARNING GOALS
// ========================================

// SetEarningGoal creates or updates an earning goal
func (r *Repository) SetEarningGoal(ctx context.Context, goal *EarningGoal) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO driver_earning_goals (
			id, driver_id, target_amount, period,
			current_amount, currency, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (driver_id, period) WHERE is_active = true
		DO UPDATE SET target_amount = $3, updated_at = $9`,
		goal.ID, goal.DriverID, goal.TargetAmount, goal.Period,
		goal.CurrentAmount, goal.Currency, goal.IsActive, goal.CreatedAt, goal.UpdatedAt,
	)
	return err
}

// GetActiveGoals returns active earning goals for a driver
func (r *Repository) GetActiveGoals(ctx context.Context, driverID uuid.UUID) ([]EarningGoal, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, driver_id, target_amount, period,
			current_amount, currency, is_active, created_at, updated_at
		FROM driver_earning_goals
		WHERE driver_id = $1 AND is_active = true
		ORDER BY period`,
		driverID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []EarningGoal
	for rows.Next() {
		g := EarningGoal{}
		if err := rows.Scan(
			&g.ID, &g.DriverID, &g.TargetAmount, &g.Period,
			&g.CurrentAmount, &g.Currency, &g.IsActive, &g.CreatedAt, &g.UpdatedAt,
		); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, nil
}

// UpdateGoalProgress updates the current amount of a goal
func (r *Repository) UpdateGoalProgress(ctx context.Context, goalID uuid.UUID, amount float64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE driver_earning_goals
		SET current_amount = current_amount + $2, updated_at = NOW()
		WHERE id = $1`,
		goalID, amount,
	)
	return err
}

// ResetGoals resets goals for a period
func (r *Repository) ResetGoals(ctx context.Context, period string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE driver_earning_goals
		SET current_amount = 0, updated_at = NOW()
		WHERE period = $1 AND is_active = true`,
		period,
	)
	return err
}

// ========================================
// STATS
// ========================================

// GetLifetimeEarnings returns total lifetime earnings for a driver
func (r *Repository) GetLifetimeEarnings(ctx context.Context, driverID uuid.UUID) (float64, error) {
	var total float64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(net_amount), 0)
		FROM driver_earnings
		WHERE driver_id = $1`,
		driverID,
	).Scan(&total)
	return total, err
}

// GetTopEarningDay returns the highest earning day
func (r *Repository) GetTopEarningDay(ctx context.Context, driverID uuid.UUID) (*DailyEarning, error) {
	d := &DailyEarning{}
	err := r.db.QueryRow(ctx, `
		SELECT
			to_char(created_at, 'YYYY-MM-DD'),
			COALESCE(SUM(gross_amount), 0),
			COALESCE(SUM(net_amount), 0),
			COALESCE(SUM(commission), 0),
			COALESCE(SUM(CASE WHEN type = 'tip' THEN net_amount ELSE 0 END), 0),
			COUNT(CASE WHEN type = 'ride_fare' THEN 1 END)
		FROM driver_earnings
		WHERE driver_id = $1
		GROUP BY to_char(created_at, 'YYYY-MM-DD')
		ORDER BY SUM(net_amount) DESC
		LIMIT 1`,
		driverID,
	).Scan(&d.Date, &d.GrossAmount, &d.NetAmount, &d.Commission, &d.Tips, &d.RideCount)
	if err != nil {
		return nil, fmt.Errorf("get top earning day: %w", err)
	}
	return d, nil
}
