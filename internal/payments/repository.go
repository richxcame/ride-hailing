package payments

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/models"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreatePayment creates a new payment record
func (r *Repository) CreatePayment(ctx context.Context, payment *models.Payment) error {
	query := `
		INSERT INTO payments (id, ride_id, rider_id, driver_id, amount, currency,
			payment_method, status, stripe_payment_id, stripe_charge_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at, updated_at`

	err := r.db.QueryRow(ctx, query,
		payment.ID,
		payment.RideID,
		payment.RiderID,
		payment.DriverID,
		payment.Amount,
		payment.Currency,
		payment.PaymentMethod,
		payment.Status,
		payment.StripePaymentID,
		payment.StripeChargeID,
	).Scan(&payment.CreatedAt, &payment.UpdatedAt)

	if err != nil {
		return common.NewInternalError("failed to create payment", err)
	}

	return nil
}

// GetPaymentByID retrieves a payment by ID
func (r *Repository) GetPaymentByID(ctx context.Context, id uuid.UUID) (*models.Payment, error) {
	payment := &models.Payment{}
	query := `
		SELECT id, ride_id, rider_id, driver_id, amount, currency, payment_method,
			status, stripe_payment_id, stripe_charge_id, metadata,
			created_at, updated_at
		FROM payments
		WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&payment.ID,
		&payment.RideID,
		&payment.RiderID,
		&payment.DriverID,
		&payment.Amount,
		&payment.Currency,
		&payment.PaymentMethod,
		&payment.Status,
		&payment.StripePaymentID,
		&payment.StripeChargeID,
		&payment.Metadata,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)

	if err != nil {
		return nil, common.NewNotFoundError("payment not found", err)
	}

	return payment, nil
}

// UpdatePaymentStatus updates the payment status
func (r *Repository) UpdatePaymentStatus(ctx context.Context, id uuid.UUID, status string, stripeChargeID *string) error {
	query := `
		UPDATE payments
		SET status = $1, stripe_charge_id = COALESCE($2, stripe_charge_id), updated_at = NOW()
		WHERE id = $3`

	_, err := r.db.Exec(ctx, query, status, stripeChargeID, id)
	if err != nil {
		return common.NewInternalError("failed to update payment status", err)
	}

	return nil
}

// GetRideDriverID retrieves the driver ID for a given ride
func (r *Repository) GetRideDriverID(ctx context.Context, rideID uuid.UUID) (*uuid.UUID, error) {
	var driverID *uuid.UUID
	err := r.db.QueryRow(ctx, "SELECT driver_id FROM rides WHERE id = $1", rideID).Scan(&driverID)
	if err != nil {
		return nil, common.NewNotFoundError("ride not found", err)
	}
	return driverID, nil
}

// GetPaymentsByRideID retrieves payments for a specific ride
func (r *Repository) GetPaymentsByRideID(ctx context.Context, rideID uuid.UUID) ([]*models.Payment, error) {
	query := `
		SELECT id, ride_id, rider_id, driver_id, amount, currency, payment_method,
			status, stripe_payment_id, stripe_charge_id, metadata,
			created_at, updated_at
		FROM payments
		WHERE ride_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, rideID)
	if err != nil {
		return nil, common.NewInternalError("failed to get payments", err)
	}
	defer rows.Close()

	payments := make([]*models.Payment, 0)
	for rows.Next() {
		payment := &models.Payment{}
		err := rows.Scan(
			&payment.ID,
			&payment.RideID,
			&payment.RiderID,
			&payment.DriverID,
			&payment.Amount,
			&payment.Currency,
			&payment.PaymentMethod,
			&payment.Status,
			&payment.StripePaymentID,
			&payment.StripeChargeID,
			&payment.Metadata,
			&payment.CreatedAt,
			&payment.UpdatedAt,
		)
		if err != nil {
			return nil, common.NewInternalError("failed to scan payment", err)
		}
		payments = append(payments, payment)
	}

	return payments, nil
}

// Wallet operations

// GetWalletByUserID retrieves a user's wallet
func (r *Repository) GetWalletByUserID(ctx context.Context, userID uuid.UUID) (*models.Wallet, error) {
	wallet := &models.Wallet{}
	query := `
		SELECT id, user_id, balance, currency, is_active, created_at, updated_at
		FROM wallets
		WHERE user_id = $1`

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.Balance,
		&wallet.Currency,
		&wallet.IsActive,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
	)

	if err != nil {
		return nil, common.NewNotFoundError("wallet not found", err)
	}

	return wallet, nil
}

// CreateWallet creates a new wallet for a user
func (r *Repository) CreateWallet(ctx context.Context, wallet *models.Wallet) error {
	query := `
		INSERT INTO wallets (id, user_id, balance, currency, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at`

	err := r.db.QueryRow(ctx, query,
		wallet.ID,
		wallet.UserID,
		wallet.Balance,
		wallet.Currency,
		wallet.IsActive,
	).Scan(&wallet.CreatedAt, &wallet.UpdatedAt)

	if err != nil {
		return common.NewInternalError("failed to create wallet", err)
	}

	return nil
}

// UpdateWalletBalance updates wallet balance atomically
func (r *Repository) UpdateWalletBalance(ctx context.Context, walletID uuid.UUID, amount float64) error {
	query := `
		UPDATE wallets
		SET balance = balance + $1, updated_at = NOW()
		WHERE id = $2 AND is_active = true
		RETURNING balance`

	var newBalance float64
	err := r.db.QueryRow(ctx, query, amount, walletID).Scan(&newBalance)
	if err != nil {
		return common.NewInternalError("failed to update wallet balance", err)
	}

	// Ensure balance doesn't go negative
	if newBalance < 0 {
		return common.NewBadRequestError("insufficient wallet balance", nil)
	}

	return nil
}

// CreateWalletTransaction creates a wallet transaction record
func (r *Repository) CreateWalletTransaction(ctx context.Context, tx *models.WalletTransaction) error {
	query := `
		INSERT INTO wallet_transactions (id, wallet_id, type, amount, description,
			reference_type, reference_id, balance_before, balance_after)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at`

	err := r.db.QueryRow(ctx, query,
		tx.ID,
		tx.WalletID,
		tx.Type,
		tx.Amount,
		tx.Description,
		tx.ReferenceType,
		tx.ReferenceID,
		tx.BalanceBefore,
		tx.BalanceAfter,
	).Scan(&tx.CreatedAt)

	if err != nil {
		return common.NewInternalError("failed to create wallet transaction", err)
	}

	return nil
}

// GetWalletTransactions retrieves wallet transaction history
func (r *Repository) GetWalletTransactions(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*models.WalletTransaction, error) {
	query := `
		SELECT id, wallet_id, type, amount, description, reference_type,
			reference_id, balance_before, balance_after, created_at
		FROM wallet_transactions
		WHERE wallet_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, walletID, limit, offset)
	if err != nil {
		return nil, common.NewInternalError("failed to get wallet transactions", err)
	}
	defer rows.Close()

	transactions := make([]*models.WalletTransaction, 0)
	for rows.Next() {
		tx := &models.WalletTransaction{}
		err := rows.Scan(
			&tx.ID,
			&tx.WalletID,
			&tx.Type,
			&tx.Amount,
			&tx.Description,
			&tx.ReferenceType,
			&tx.ReferenceID,
			&tx.BalanceBefore,
			&tx.BalanceAfter,
			&tx.CreatedAt,
		)
		if err != nil {
			return nil, common.NewInternalError("failed to scan wallet transaction", err)
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

// GetWalletTransactionsWithTotal retrieves wallet transaction history with total count
func (r *Repository) GetWalletTransactionsWithTotal(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*models.WalletTransaction, int64, error) {
	// Get total count
	var total int64
	countQuery := `SELECT COUNT(*) FROM wallet_transactions WHERE wallet_id = $1`
	err := r.db.QueryRow(ctx, countQuery, walletID).Scan(&total)
	if err != nil {
		return nil, 0, common.NewInternalError("failed to count wallet transactions", err)
	}

	// Get paginated transactions
	transactions, err := r.GetWalletTransactions(ctx, walletID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}

// ProcessPaymentWithWallet handles payment using wallet balance in a transaction
func (r *Repository) ProcessPaymentWithWallet(ctx context.Context, payment *models.Payment, walletTx *models.WalletTransaction) error {
	// Start database transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return common.NewInternalError("failed to start transaction", err)
	}
	defer tx.Rollback(ctx)

	// Get current wallet balance
	var currentBalance float64
	var walletID uuid.UUID
	query := `SELECT id, balance FROM wallets WHERE user_id = $1 AND is_active = true FOR UPDATE`
	err = tx.QueryRow(ctx, query, payment.RiderID).Scan(&walletID, &currentBalance)
	if err != nil {
		return common.NewNotFoundError("wallet not found", err)
	}

	// Check sufficient balance
	if currentBalance < payment.Amount {
		return common.NewBadRequestError("insufficient wallet balance", nil)
	}

	// Update wallet balance
	newBalance := currentBalance - payment.Amount
	_, err = tx.Exec(ctx, `UPDATE wallets SET balance = $1, updated_at = NOW() WHERE id = $2`, newBalance, walletID)
	if err != nil {
		return common.NewInternalError("failed to update wallet balance", err)
	}

	// Create payment record
	_, err = tx.Exec(ctx, `
		INSERT INTO payments (id, ride_id, rider_id, driver_id, amount, currency,
			payment_method, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())`,
		payment.ID, payment.RideID, payment.RiderID, payment.DriverID,
		payment.Amount, payment.Currency, payment.PaymentMethod, "completed")
	if err != nil {
		return common.NewInternalError("failed to create payment", err)
	}

	// Create wallet transaction record
	walletTx.WalletID = walletID
	walletTx.BalanceBefore = currentBalance
	walletTx.BalanceAfter = newBalance
	_, err = tx.Exec(ctx, `
		INSERT INTO wallet_transactions (id, wallet_id, type, amount, description,
			reference_type, reference_id, balance_before, balance_after, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())`,
		walletTx.ID, walletTx.WalletID, walletTx.Type, walletTx.Amount,
		walletTx.Description, walletTx.ReferenceType, walletTx.ReferenceID,
		walletTx.BalanceBefore, walletTx.BalanceAfter)
	if err != nil {
		return common.NewInternalError("failed to create wallet transaction", err)
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		return common.NewInternalError("failed to commit transaction", err)
	}

	payment.Status = "completed"
	payment.CreatedAt = time.Now()
	payment.UpdatedAt = time.Now()

	return nil
}
