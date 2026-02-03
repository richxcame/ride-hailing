package paymentmethods

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles payment method data access
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new payment methods repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// PAYMENT METHODS
// ========================================

// CreatePaymentMethod creates a new payment method
func (r *Repository) CreatePaymentMethod(ctx context.Context, pm *PaymentMethod) error {
	// If setting as default, unset other defaults
	if pm.IsDefault {
		r.db.Exec(ctx, `
			UPDATE payment_methods SET is_default = false, updated_at = NOW()
			WHERE user_id = $1`, pm.UserID)
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO payment_methods (
			id, user_id, type, is_default, is_active,
			card_brand, card_last4, card_exp_month, card_exp_year, card_holder_name,
			wallet_balance, provider_id, provider_type,
			nickname, currency, country, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13,
			$14, $15, $16, $17, $18
		)`,
		pm.ID, pm.UserID, pm.Type, pm.IsDefault, pm.IsActive,
		pm.CardBrand, pm.CardLast4, pm.CardExpMonth, pm.CardExpYear, pm.CardHolderName,
		pm.WalletBalance, pm.ProviderID, pm.ProviderType,
		pm.Nickname, pm.Currency, pm.Country, pm.CreatedAt, pm.UpdatedAt,
	)
	return err
}

// GetPaymentMethodsByUser returns all active payment methods for a user
func (r *Repository) GetPaymentMethodsByUser(ctx context.Context, userID uuid.UUID) ([]PaymentMethod, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, type, is_default, is_active,
			card_brand, card_last4, card_exp_month, card_exp_year, card_holder_name,
			wallet_balance, provider_id, provider_type,
			nickname, currency, country, created_at, updated_at
		FROM payment_methods
		WHERE user_id = $1 AND is_active = true
		ORDER BY is_default DESC, created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var methods []PaymentMethod
	for rows.Next() {
		pm := PaymentMethod{}
		if err := rows.Scan(
			&pm.ID, &pm.UserID, &pm.Type, &pm.IsDefault, &pm.IsActive,
			&pm.CardBrand, &pm.CardLast4, &pm.CardExpMonth, &pm.CardExpYear, &pm.CardHolderName,
			&pm.WalletBalance, &pm.ProviderID, &pm.ProviderType,
			&pm.Nickname, &pm.Currency, &pm.Country, &pm.CreatedAt, &pm.UpdatedAt,
		); err != nil {
			return nil, err
		}
		methods = append(methods, pm)
	}
	return methods, nil
}

// GetPaymentMethodByID retrieves a payment method by ID
func (r *Repository) GetPaymentMethodByID(ctx context.Context, id uuid.UUID) (*PaymentMethod, error) {
	pm := &PaymentMethod{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, type, is_default, is_active,
			card_brand, card_last4, card_exp_month, card_exp_year, card_holder_name,
			wallet_balance, provider_id, provider_type,
			nickname, currency, country, created_at, updated_at
		FROM payment_methods WHERE id = $1`, id,
	).Scan(
		&pm.ID, &pm.UserID, &pm.Type, &pm.IsDefault, &pm.IsActive,
		&pm.CardBrand, &pm.CardLast4, &pm.CardExpMonth, &pm.CardExpYear, &pm.CardHolderName,
		&pm.WalletBalance, &pm.ProviderID, &pm.ProviderType,
		&pm.Nickname, &pm.Currency, &pm.Country, &pm.CreatedAt, &pm.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return pm, nil
}

// GetDefaultPaymentMethod returns the user's default payment method
func (r *Repository) GetDefaultPaymentMethod(ctx context.Context, userID uuid.UUID) (*PaymentMethod, error) {
	pm := &PaymentMethod{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, type, is_default, is_active,
			card_brand, card_last4, card_exp_month, card_exp_year, card_holder_name,
			wallet_balance, provider_id, provider_type,
			nickname, currency, country, created_at, updated_at
		FROM payment_methods
		WHERE user_id = $1 AND is_default = true AND is_active = true`, userID,
	).Scan(
		&pm.ID, &pm.UserID, &pm.Type, &pm.IsDefault, &pm.IsActive,
		&pm.CardBrand, &pm.CardLast4, &pm.CardExpMonth, &pm.CardExpYear, &pm.CardHolderName,
		&pm.WalletBalance, &pm.ProviderID, &pm.ProviderType,
		&pm.Nickname, &pm.Currency, &pm.Country, &pm.CreatedAt, &pm.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return pm, nil
}

// GetWallet returns the user's wallet payment method
func (r *Repository) GetWallet(ctx context.Context, userID uuid.UUID) (*PaymentMethod, error) {
	pm := &PaymentMethod{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, type, is_default, is_active,
			card_brand, card_last4, card_exp_month, card_exp_year, card_holder_name,
			wallet_balance, provider_id, provider_type,
			nickname, currency, country, created_at, updated_at
		FROM payment_methods
		WHERE user_id = $1 AND type = 'wallet' AND is_active = true`, userID,
	).Scan(
		&pm.ID, &pm.UserID, &pm.Type, &pm.IsDefault, &pm.IsActive,
		&pm.CardBrand, &pm.CardLast4, &pm.CardExpMonth, &pm.CardExpYear, &pm.CardHolderName,
		&pm.WalletBalance, &pm.ProviderID, &pm.ProviderType,
		&pm.Nickname, &pm.Currency, &pm.Country, &pm.CreatedAt, &pm.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return pm, nil
}

// SetDefault sets a payment method as default
func (r *Repository) SetDefault(ctx context.Context, userID, methodID uuid.UUID) error {
	// Unset all defaults
	r.db.Exec(ctx, `
		UPDATE payment_methods SET is_default = false, updated_at = NOW()
		WHERE user_id = $1`, userID)

	// Set new default
	_, err := r.db.Exec(ctx, `
		UPDATE payment_methods SET is_default = true, updated_at = NOW()
		WHERE id = $1 AND user_id = $2`,
		methodID, userID,
	)
	return err
}

// DeactivatePaymentMethod soft-deletes a payment method
func (r *Repository) DeactivatePaymentMethod(ctx context.Context, id, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE payment_methods
		SET is_active = false, is_default = false, updated_at = NOW()
		WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	return err
}

// ========================================
// WALLET
// ========================================

// UpdateWalletBalance updates the wallet balance atomically
func (r *Repository) UpdateWalletBalance(ctx context.Context, walletID uuid.UUID, delta float64) (float64, error) {
	var newBalance float64
	err := r.db.QueryRow(ctx, `
		UPDATE payment_methods
		SET wallet_balance = wallet_balance + $2, updated_at = NOW()
		WHERE id = $1 AND type = 'wallet'
		RETURNING wallet_balance`,
		walletID, delta,
	).Scan(&newBalance)
	return newBalance, err
}

// CreateWalletTransaction records a wallet transaction
func (r *Repository) CreateWalletTransaction(ctx context.Context, tx *WalletTransaction) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO wallet_transactions (
			id, user_id, payment_method_id, type, amount,
			balance_before, balance_after, description,
			ride_id, reference_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		tx.ID, tx.UserID, tx.PaymentMethodID, tx.Type, tx.Amount,
		tx.BalanceBefore, tx.BalanceAfter, tx.Description,
		tx.RideID, tx.ReferenceID, tx.CreatedAt,
	)
	return err
}

// GetWalletTransactions returns recent wallet transactions
func (r *Repository) GetWalletTransactions(ctx context.Context, userID uuid.UUID, limit int) ([]WalletTransaction, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, payment_method_id, type, amount,
			balance_before, balance_after, description,
			ride_id, reference_id, created_at
		FROM wallet_transactions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []WalletTransaction
	for rows.Next() {
		tx := WalletTransaction{}
		if err := rows.Scan(
			&tx.ID, &tx.UserID, &tx.PaymentMethodID, &tx.Type, &tx.Amount,
			&tx.BalanceBefore, &tx.BalanceAfter, &tx.Description,
			&tx.RideID, &tx.ReferenceID, &tx.CreatedAt,
		); err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}
	return transactions, nil
}

// GetWalletBalance returns the current wallet balance
func (r *Repository) GetWalletBalance(ctx context.Context, userID uuid.UUID) (float64, error) {
	var balance float64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(wallet_balance, 0)
		FROM payment_methods
		WHERE user_id = $1 AND type = 'wallet' AND is_active = true`,
		userID,
	).Scan(&balance)
	if err != nil {
		return 0, nil // No wallet = 0 balance
	}
	return balance, nil
}

// EnsureWalletExists creates a wallet if the user doesn't have one
func (r *Repository) EnsureWalletExists(ctx context.Context, userID uuid.UUID) (*PaymentMethod, error) {
	// Try to get existing wallet
	wallet, err := r.GetWallet(ctx, userID)
	if err == nil {
		return wallet, nil
	}

	// Create wallet
	now := time.Now()
	balance := 0.0
	wallet = &PaymentMethod{
		ID:            uuid.New(),
		UserID:        userID,
		Type:          PaymentMethodWallet,
		IsDefault:     false,
		IsActive:      true,
		WalletBalance: &balance,
		Currency:      "USD",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := r.CreatePaymentMethod(ctx, wallet); err != nil {
		return nil, err
	}
	return wallet, nil
}
