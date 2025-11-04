package payments

import (
	"go.uber.org/zap"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/models"
)

const (
	commissionRate      = 0.20 // 20% platform commission
	cancellationFeeRate = 0.10 // 10% cancellation fee
)

type Service struct {
	repo         *Repository
	stripeClient *StripeClient
}

func NewService(repo *Repository, stripeAPIKey string) *Service {
	return &Service{
		repo:         repo,
		stripeClient: NewStripeClient(stripeAPIKey),
	}
}

// ProcessRidePayment processes payment for a completed ride
func (s *Service) ProcessRidePayment(ctx context.Context, rideID, riderID, driverID uuid.UUID, amount float64, paymentMethod string) (*models.Payment, error) {
	payment := &models.Payment{
		ID:            uuid.New(),
		RideID:        rideID,
		RiderID:       riderID,
		DriverID:      driverID,
		Amount:        amount,
		Currency:      "usd",
		PaymentMethod: paymentMethod,
		Status:        "pending",
		Metadata:      map[string]interface{}{},
	}

	switch paymentMethod {
	case "wallet":
		return s.processWalletPayment(ctx, payment)
	case "stripe":
		return s.processStripePayment(ctx, payment)
	default:
		return nil, common.NewBadRequestError("invalid payment method", nil)
	}
}

// processWalletPayment processes payment using wallet balance
func (s *Service) processWalletPayment(ctx context.Context, payment *models.Payment) (*models.Payment, error) {
	walletTx := &models.WalletTransaction{
		ID:            uuid.New(),
		Type:          "debit",
		Amount:        payment.Amount,
		Description:   fmt.Sprintf("Payment for ride %s", payment.RideID),
		ReferenceType: "ride",
		ReferenceID:   &payment.RideID,
	}

	err := s.repo.ProcessPaymentWithWallet(ctx, payment, walletTx)
	if err != nil {
		logger.Get().Error("Failed to process wallet payment", zap.Error(err), "payment_id", payment.ID)
		return nil, err
	}

	logger.Get().Info("Wallet payment processed successfully", zap.String("payment_id", payment.ID.String()), "amount", payment.Amount)
	return payment, nil
}

// processStripePayment processes payment using Stripe
func (s *Service) processStripePayment(ctx context.Context, payment *models.Payment) (*models.Payment, error) {
	// Convert amount to cents for Stripe
	amountCents := int64(payment.Amount * 100)

	// Create payment intent
	metadata := map[string]string{
		"ride_id":   payment.RideID.String(),
		"rider_id":  payment.RiderID.String(),
		"driver_id": payment.DriverID.String(),
	}

	pi, err := s.stripeClient.CreatePaymentIntent(
		amountCents,
		payment.Currency,
		"", // Customer ID should be fetched from user profile
		fmt.Sprintf("Payment for ride %s", payment.RideID),
		metadata,
	)

	if err != nil {
		logger.Get().Error("Failed to create Stripe payment intent", zap.Error(err))
		return nil, common.NewInternalError("failed to create payment", err)
	}

	payment.StripePaymentID = &pi.ID
	payment.Status = string(pi.Status)

	// Save payment to database
	err = s.repo.CreatePayment(ctx, payment)
	if err != nil {
		logger.Get().Error("Failed to save payment", zap.Error(err))
		return nil, err
	}

	logger.Get().Info("Stripe payment created successfully", zap.String("payment_id", payment.ID.String()), "stripe_pi", pi.ID)
	return payment, nil
}

// TopUpWallet adds money to a user's wallet via Stripe
func (s *Service) TopUpWallet(ctx context.Context, userID uuid.UUID, amount float64, stripePaymentMethodID string) (*models.WalletTransaction, error) {
	// Get or create wallet
	wallet, err := s.repo.GetWalletByUserID(ctx, userID)
	if err != nil {
		// Create wallet if it doesn't exist
		wallet = &models.Wallet{
			ID:       uuid.New(),
			UserID:   userID,
			Balance:  0,
			Currency: "usd",
			IsActive: true,
		}
		err = s.repo.CreateWallet(ctx, wallet)
		if err != nil {
			return nil, err
		}
	}

	// Create Stripe payment intent for top-up
	amountCents := int64(amount * 100)
	metadata := map[string]string{
		"user_id":   userID.String(),
		"wallet_id": wallet.ID.String(),
		"type":      "wallet_topup",
	}

	pi, err := s.stripeClient.CreatePaymentIntent(
		amountCents,
		wallet.Currency,
		"", // Customer ID
		"Wallet top-up",
		metadata,
	)

	if err != nil {
		logger.Get().Error("Failed to create top-up payment intent", zap.Error(err))
		return nil, common.NewInternalError("failed to process top-up", err)
	}

	// Note: In a real implementation, you would wait for the webhook
	// to confirm payment before crediting the wallet
	// For now, we'll create a pending transaction

	walletTx := &models.WalletTransaction{
		ID:            uuid.New(),
		WalletID:      wallet.ID,
		Type:          "credit",
		Amount:        amount,
		Description:   "Wallet top-up via Stripe",
		ReferenceType: "stripe_payment",
		ReferenceID:   nil,
		BalanceBefore: wallet.Balance,
		BalanceAfter:  wallet.Balance, // Will be updated when payment confirms
		CreatedAt:     time.Now(),
	}

	err = s.repo.CreateWalletTransaction(ctx, walletTx)
	if err != nil {
		return nil, err
	}

	logger.Get().Info("Wallet top-up initiated", "user_id", userID, "amount", amount, "stripe_pi", pi.ID)
	return walletTx, nil
}

// ConfirmWalletTopUp confirms a wallet top-up after Stripe payment succeeds
func (s *Service) ConfirmWalletTopUp(ctx context.Context, userID uuid.UUID, amount float64, stripePaymentIntentID string) error {
	wallet, err := s.repo.GetWalletByUserID(ctx, userID)
	if err != nil {
		return err
	}

	// Update wallet balance
	err = s.repo.UpdateWalletBalance(ctx, wallet.ID, amount)
	if err != nil {
		return err
	}

	// Create confirmed transaction
	referenceID := uuid.New()
	walletTx := &models.WalletTransaction{
		ID:            uuid.New(),
		WalletID:      wallet.ID,
		Type:          "credit",
		Amount:        amount,
		Description:   "Wallet top-up confirmed",
		ReferenceType: "stripe_payment",
		ReferenceID:   &referenceID,
		BalanceBefore: wallet.Balance,
		BalanceAfter:  wallet.Balance + amount,
	}

	err = s.repo.CreateWalletTransaction(ctx, walletTx)
	if err != nil {
		return err
	}

	logger.Get().Info("Wallet top-up confirmed", "user_id", userID, "amount", amount)
	return nil
}

// PayoutToDriver processes payout to driver after ride completion
func (s *Service) PayoutToDriver(ctx context.Context, paymentID uuid.UUID) error {
	payment, err := s.repo.GetPaymentByID(ctx, paymentID)
	if err != nil {
		return err
	}

	if payment.Status != "completed" {
		return common.NewBadRequestError("payment not completed", nil)
	}

	// Calculate driver earnings (total - commission)
	commission := payment.Amount * commissionRate
	driverEarnings := payment.Amount - commission

	// Get driver's wallet
	driverWallet, err := s.repo.GetWalletByUserID(ctx, payment.DriverID)
	if err != nil {
		// Create wallet if doesn't exist
		driverWallet = &models.Wallet{
			ID:       uuid.New(),
			UserID:   payment.DriverID,
			Balance:  0,
			Currency: "usd",
			IsActive: true,
		}
		err = s.repo.CreateWallet(ctx, driverWallet)
		if err != nil {
			return err
		}
	}

	// Credit driver wallet
	err = s.repo.UpdateWalletBalance(ctx, driverWallet.ID, driverEarnings)
	if err != nil {
		return err
	}

	// Create wallet transaction
	walletTx := &models.WalletTransaction{
		ID:            uuid.New(),
		WalletID:      driverWallet.ID,
		Type:          "credit",
		Amount:        driverEarnings,
		Description:   fmt.Sprintf("Earnings from ride %s (%.2f%% commission)", payment.RideID, commissionRate*100),
		ReferenceType: "ride",
		ReferenceID:   &payment.RideID,
		BalanceBefore: driverWallet.Balance,
		BalanceAfter:  driverWallet.Balance + driverEarnings,
	}

	err = s.repo.CreateWalletTransaction(ctx, walletTx)
	if err != nil {
		return err
	}

	logger.Get().Info("Driver payout processed",
		"payment_id", paymentID,
		"driver_id", payment.DriverID,
		"amount", payment.Amount,
		"commission", commission,
		"driver_earnings", driverEarnings)

	return nil
}

// ProcessRefund processes a refund for a cancelled ride
func (s *Service) ProcessRefund(ctx context.Context, paymentID uuid.UUID, reason string) error {
	payment, err := s.repo.GetPaymentByID(ctx, paymentID)
	if err != nil {
		return err
	}

	if payment.Status == "refunded" {
		return common.NewBadRequestError("payment already refunded", nil)
	}

	var refundAmount = payment.Amount

	// Apply cancellation fee if applicable
	if reason == "rider_cancelled" {
		cancellationFee := payment.Amount * cancellationFeeRate
		refundAmount = payment.Amount - cancellationFee
		logger.Get().Info("Applying cancellation fee", "payment_id", paymentID, "fee", cancellationFee)
	}

	if payment.PaymentMethod == "stripe" && payment.StripePaymentID != nil {
		// Process Stripe refund
		refundAmountCents := int64(refundAmount * 100)
		_, err := s.stripeClient.CreateRefund(*payment.StripeChargeID, &refundAmountCents, reason)
		if err != nil {
			logger.Get().Error("Failed to create Stripe refund", zap.Error(err))
			return common.NewInternalError("failed to process refund", err)
		}
	} else if payment.PaymentMethod == "wallet" {
		// Refund to wallet
		wallet, err := s.repo.GetWalletByUserID(ctx, payment.RiderID)
		if err != nil {
			return err
		}

		err = s.repo.UpdateWalletBalance(ctx, wallet.ID, refundAmount)
		if err != nil {
			return err
		}

		// Create refund transaction
		walletTx := &models.WalletTransaction{
			ID:            uuid.New(),
			WalletID:      wallet.ID,
			Type:          "credit",
			Amount:        refundAmount,
			Description:   fmt.Sprintf("Refund for ride %s (%s)", payment.RideID, reason),
			ReferenceType: "ride",
			ReferenceID:   &payment.RideID,
			BalanceBefore: wallet.Balance,
			BalanceAfter:  wallet.Balance + refundAmount,
		}

		err = s.repo.CreateWalletTransaction(ctx, walletTx)
		if err != nil {
			return err
		}
	}

	// Update payment status
	status := "refunded"
	err = s.repo.UpdatePaymentStatus(ctx, paymentID, status, nil)
	if err != nil {
		return err
	}

	logger.Get().Info("Refund processed successfully",
		"payment_id", paymentID,
		"original_amount", payment.Amount,
		"refund_amount", refundAmount,
		"reason", reason)

	return nil
}

// GetWallet retrieves a user's wallet
func (s *Service) GetWallet(ctx context.Context, userID uuid.UUID) (*models.Wallet, error) {
	return s.repo.GetWalletByUserID(ctx, userID)
}

// GetWalletTransactions retrieves wallet transaction history
func (s *Service) GetWalletTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.WalletTransaction, error) {
	wallet, err := s.repo.GetWalletByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetWalletTransactions(ctx, wallet.ID, limit, offset)
}

// HandleStripeWebhook handles Stripe webhook events
func (s *Service) HandleStripeWebhook(ctx context.Context, eventType string, paymentIntentID string) error {
	logger.Get().Info("Handling Stripe webhook", "event_type", eventType, "payment_intent_id", paymentIntentID)

	switch eventType {
	case "payment_intent.succeeded":
		// Payment succeeded - update payment status
		// In a real implementation, you would:
		// 1. Retrieve payment by stripe_payment_id
		// 2. Update payment status to 'completed'
		// 3. If it's a wallet top-up, credit the wallet
		logger.Get().Info("Payment intent succeeded", "payment_intent_id", paymentIntentID)

	case "payment_intent.payment_failed":
		// Payment failed - update status
		logger.Get().Warn("Payment intent failed", "payment_intent_id", paymentIntentID)

	case "charge.refunded":
		// Refund processed
		logger.Get().Info("Charge refunded", "payment_intent_id", paymentIntentID)

	default:
		logger.Get().Debug("Unhandled webhook event", "event_type", eventType)
	}

	return nil
}
