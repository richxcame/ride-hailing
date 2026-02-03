package paymentmethods

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/richxcame/ride-hailing/pkg/common"
)

const (
	maxPaymentMethods = 10
	minTopUpAmount    = 5.0
	maxTopUpAmount    = 500.0
	defaultCurrency   = "USD"
)

// Service handles payment method business logic
type Service struct {
	repo *Repository
}

// NewService creates a new payment methods service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ========================================
// PAYMENT METHODS
// ========================================

// AddCard adds a card payment method
func (s *Service) AddCard(ctx context.Context, userID uuid.UUID, req *AddCardRequest) (*PaymentMethod, error) {
	// Check method count
	methods, err := s.repo.GetPaymentMethodsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(methods) >= maxPaymentMethods {
		return nil, common.NewBadRequestError(
			fmt.Sprintf("maximum %d payment methods allowed", maxPaymentMethods), nil,
		)
	}

	// In production: validate token with Stripe/payment provider and get card details
	// For now, simulate extracting card info from token
	brand := CardBrandVisa
	last4 := "4242" // Would come from provider
	expMonth := 12
	expYear := time.Now().Year() + 3
	holder := "Card Holder"

	isFirst := len(methods) == 0

	now := time.Now()
	pm := &PaymentMethod{
		ID:             uuid.New(),
		UserID:         userID,
		Type:           PaymentMethodCard,
		IsDefault:      isFirst || req.SetAsDefault,
		IsActive:       true,
		CardBrand:      &brand,
		CardLast4:      &last4,
		CardExpMonth:   &expMonth,
		CardExpYear:    &expYear,
		CardHolderName: &holder,
		ProviderID:     &req.Token,
		Nickname:       req.Nickname,
		Currency:       defaultCurrency,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	providerType := "stripe"
	pm.ProviderType = &providerType

	if err := s.repo.CreatePaymentMethod(ctx, pm); err != nil {
		return nil, fmt.Errorf("create payment method: %w", err)
	}

	return pm, nil
}

// AddDigitalWallet adds Apple Pay or Google Pay
func (s *Service) AddDigitalWallet(ctx context.Context, userID uuid.UUID, req *AddWalletPaymentRequest) (*PaymentMethod, error) {
	if req.Type != PaymentMethodApplePay && req.Type != PaymentMethodGooglePay {
		return nil, common.NewBadRequestError("type must be apple_pay or google_pay", nil)
	}

	now := time.Now()
	pm := &PaymentMethod{
		ID:         uuid.New(),
		UserID:     userID,
		Type:       req.Type,
		IsDefault:  false,
		IsActive:   true,
		ProviderID: &req.Token,
		Nickname:   req.Nickname,
		Currency:   defaultCurrency,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.repo.CreatePaymentMethod(ctx, pm); err != nil {
		return nil, fmt.Errorf("create digital wallet: %w", err)
	}

	return pm, nil
}

// EnableCash enables cash as a payment method
func (s *Service) EnableCash(ctx context.Context, userID uuid.UUID) (*PaymentMethod, error) {
	// Check if cash already enabled
	methods, err := s.repo.GetPaymentMethodsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, m := range methods {
		if m.Type == PaymentMethodCash {
			return &m, nil // Already enabled
		}
	}

	now := time.Now()
	pm := &PaymentMethod{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      PaymentMethodCash,
		IsDefault: false,
		IsActive:  true,
		Currency:  defaultCurrency,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.CreatePaymentMethod(ctx, pm); err != nil {
		return nil, err
	}

	return pm, nil
}

// GetPaymentMethods returns all payment methods for a user
func (s *Service) GetPaymentMethods(ctx context.Context, userID uuid.UUID) (*PaymentMethodsResponse, error) {
	methods, err := s.repo.GetPaymentMethodsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if methods == nil {
		methods = []PaymentMethod{}
	}

	walletBalance, _ := s.repo.GetWalletBalance(ctx, userID)

	var defaultID *uuid.UUID
	for _, m := range methods {
		if m.IsDefault {
			id := m.ID
			defaultID = &id
			break
		}
	}

	return &PaymentMethodsResponse{
		Methods:       methods,
		DefaultMethod: defaultID,
		WalletBalance: walletBalance,
		Currency:      defaultCurrency,
	}, nil
}

// SetDefault sets the default payment method
func (s *Service) SetDefault(ctx context.Context, userID uuid.UUID, methodID uuid.UUID) error {
	pm, err := s.repo.GetPaymentMethodByID(ctx, methodID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return common.NewNotFoundError("payment method not found", nil)
		}
		return err
	}

	if pm.UserID != userID {
		return common.NewForbiddenError("not your payment method")
	}

	return s.repo.SetDefault(ctx, userID, methodID)
}

// RemovePaymentMethod removes a payment method
func (s *Service) RemovePaymentMethod(ctx context.Context, userID uuid.UUID, methodID uuid.UUID) error {
	pm, err := s.repo.GetPaymentMethodByID(ctx, methodID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return common.NewNotFoundError("payment method not found", nil)
		}
		return err
	}

	if pm.UserID != userID {
		return common.NewForbiddenError("not your payment method")
	}

	if pm.Type == PaymentMethodWallet {
		return common.NewBadRequestError("wallet cannot be removed", nil)
	}

	return s.repo.DeactivatePaymentMethod(ctx, methodID, userID)
}

// ========================================
// WALLET
// ========================================

// TopUpWallet adds funds to the user's wallet
func (s *Service) TopUpWallet(ctx context.Context, userID uuid.UUID, req *TopUpWalletRequest) (*WalletSummary, error) {
	if req.Amount < minTopUpAmount || req.Amount > maxTopUpAmount {
		return nil, common.NewBadRequestError(
			fmt.Sprintf("top up amount must be between %.2f and %.2f", minTopUpAmount, maxTopUpAmount), nil,
		)
	}

	// Verify source payment method
	source, err := s.repo.GetPaymentMethodByID(ctx, req.SourceMethodID)
	if err != nil {
		return nil, common.NewNotFoundError("source payment method not found", nil)
	}
	if source.UserID != userID {
		return nil, common.NewForbiddenError("not your payment method")
	}
	if source.Type != PaymentMethodCard {
		return nil, common.NewBadRequestError("can only top up from a card", nil)
	}

	// Ensure wallet exists
	wallet, err := s.repo.EnsureWalletExists(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ensure wallet: %w", err)
	}

	balanceBefore := 0.0
	if wallet.WalletBalance != nil {
		balanceBefore = *wallet.WalletBalance
	}

	// Update balance
	newBalance, err := s.repo.UpdateWalletBalance(ctx, wallet.ID, req.Amount)
	if err != nil {
		return nil, fmt.Errorf("update wallet balance: %w", err)
	}

	// Record transaction
	tx := &WalletTransaction{
		ID:              uuid.New(),
		UserID:          userID,
		PaymentMethodID: wallet.ID,
		Type:            "topup",
		Amount:          req.Amount,
		BalanceBefore:   balanceBefore,
		BalanceAfter:    newBalance,
		Description:     "Wallet top up",
		CreatedAt:       time.Now(),
	}
	s.repo.CreateWalletTransaction(ctx, tx)

	return s.GetWalletSummary(ctx, userID)
}

// GetWalletSummary returns wallet info with recent transactions
func (s *Service) GetWalletSummary(ctx context.Context, userID uuid.UUID) (*WalletSummary, error) {
	balance, _ := s.repo.GetWalletBalance(ctx, userID)
	txns, _ := s.repo.GetWalletTransactions(ctx, userID, 20)
	if txns == nil {
		txns = []WalletTransaction{}
	}

	return &WalletSummary{
		Balance:            balance,
		Currency:           defaultCurrency,
		RecentTransactions: txns,
	}, nil
}

// DeductFromWallet deducts from wallet for a ride payment
func (s *Service) DeductFromWallet(ctx context.Context, userID uuid.UUID, rideID uuid.UUID, amount float64) (float64, error) {
	wallet, err := s.repo.GetWallet(ctx, userID)
	if err != nil {
		return 0, common.NewNotFoundError("wallet not found", nil)
	}

	currentBalance := 0.0
	if wallet.WalletBalance != nil {
		currentBalance = *wallet.WalletBalance
	}

	if currentBalance <= 0 {
		return 0, nil
	}

	deductAmount := amount
	if deductAmount > currentBalance {
		deductAmount = currentBalance
	}

	newBalance, err := s.repo.UpdateWalletBalance(ctx, wallet.ID, -deductAmount)
	if err != nil {
		return 0, err
	}

	tx := &WalletTransaction{
		ID:              uuid.New(),
		UserID:          userID,
		PaymentMethodID: wallet.ID,
		Type:            "debit",
		Amount:          deductAmount,
		BalanceBefore:   currentBalance,
		BalanceAfter:    newBalance,
		Description:     "Ride payment",
		RideID:          &rideID,
		CreatedAt:       time.Now(),
	}
	s.repo.CreateWalletTransaction(ctx, tx)

	return deductAmount, nil
}
