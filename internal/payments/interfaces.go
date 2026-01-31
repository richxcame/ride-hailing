package payments

import (
	"context"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/stripe/stripe-go/v83"
)

// RepositoryInterface defines the interface for payments repository operations
type RepositoryInterface interface {
	CreatePayment(ctx context.Context, payment *models.Payment) error
	GetPaymentByID(ctx context.Context, id uuid.UUID) (*models.Payment, error)
	UpdatePaymentStatus(ctx context.Context, id uuid.UUID, status string, stripeChargeID *string) error
	ProcessPaymentWithWallet(ctx context.Context, payment *models.Payment, transaction *models.WalletTransaction) error
	GetWalletByUserID(ctx context.Context, userID uuid.UUID) (*models.Wallet, error)
	CreateWallet(ctx context.Context, wallet *models.Wallet) error
	UpdateWalletBalance(ctx context.Context, walletID uuid.UUID, amount float64) error
	CreateWalletTransaction(ctx context.Context, transaction *models.WalletTransaction) error
	GetWalletTransactions(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*models.WalletTransaction, error)
	GetWalletTransactionsWithTotal(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*models.WalletTransaction, int64, error)
	GetRideDriverID(ctx context.Context, rideID uuid.UUID) (*uuid.UUID, error)
}

// StripeClientInterface defines the interface for Stripe operations
type StripeClientInterface interface {
	CreateCustomer(email, name string, metadata map[string]string) (*stripe.Customer, error)
	CreatePaymentIntent(amount int64, currency, customerID, description string, metadata map[string]string) (*stripe.PaymentIntent, error)
	ConfirmPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error)
	CapturePaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error)
	CreateRefund(chargeID string, amount *int64, reason string) (*stripe.Refund, error)
	CreateTransfer(amount int64, currency, destination, description string, metadata map[string]string) (*stripe.Transfer, error)
	GetPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error)
	CancelPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error)
}
