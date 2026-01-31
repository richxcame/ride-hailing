package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/stripe/stripe-go/v83"
	"github.com/stretchr/testify/mock"
)

// MockPaymentsRepository is a mock implementation of the payments repository
type MockPaymentsRepository struct {
	mock.Mock
}

func (m *MockPaymentsRepository) CreatePayment(ctx context.Context, payment *models.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockPaymentsRepository) GetPaymentByID(ctx context.Context, id uuid.UUID) (*models.Payment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Payment), args.Error(1)
}

func (m *MockPaymentsRepository) UpdatePaymentStatus(ctx context.Context, id uuid.UUID, status string, stripeChargeID *string) error {
	args := m.Called(ctx, id, status, stripeChargeID)
	return args.Error(0)
}

func (m *MockPaymentsRepository) ProcessPaymentWithWallet(ctx context.Context, payment *models.Payment, transaction *models.WalletTransaction) error {
	args := m.Called(ctx, payment, transaction)
	return args.Error(0)
}

func (m *MockPaymentsRepository) GetWalletByUserID(ctx context.Context, userID uuid.UUID) (*models.Wallet, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Wallet), args.Error(1)
}

func (m *MockPaymentsRepository) CreateWallet(ctx context.Context, wallet *models.Wallet) error {
	args := m.Called(ctx, wallet)
	return args.Error(0)
}

func (m *MockPaymentsRepository) UpdateWalletBalance(ctx context.Context, walletID uuid.UUID, amount float64) error {
	args := m.Called(ctx, walletID, amount)
	return args.Error(0)
}

func (m *MockPaymentsRepository) CreateWalletTransaction(ctx context.Context, transaction *models.WalletTransaction) error {
	args := m.Called(ctx, transaction)
	return args.Error(0)
}

func (m *MockPaymentsRepository) GetWalletTransactions(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*models.WalletTransaction, error) {
	args := m.Called(ctx, walletID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.WalletTransaction), args.Error(1)
}

func (m *MockPaymentsRepository) GetWalletTransactionsWithTotal(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*models.WalletTransaction, int64, error) {
	args := m.Called(ctx, walletID, limit, offset)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*models.WalletTransaction), int64(args.Int(1)), args.Error(2)
}

func (m *MockPaymentsRepository) GetRideDriverID(ctx context.Context, rideID uuid.UUID) (*uuid.UUID, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	id := args.Get(0).(*uuid.UUID)
	return id, args.Error(1)
}

// MockStripeClient is a mock implementation of the Stripe client
type MockStripeClient struct {
	mock.Mock
}

func (m *MockStripeClient) CreateCustomer(email, name string, metadata map[string]string) (*stripe.Customer, error) {
	args := m.Called(email, name, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stripe.Customer), args.Error(1)
}

func (m *MockStripeClient) CreatePaymentIntent(amount int64, currency, customerID, description string, metadata map[string]string) (*stripe.PaymentIntent, error) {
	args := m.Called(amount, currency, customerID, description, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stripe.PaymentIntent), args.Error(1)
}

func (m *MockStripeClient) ConfirmPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	args := m.Called(paymentIntentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stripe.PaymentIntent), args.Error(1)
}

func (m *MockStripeClient) CapturePaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	args := m.Called(paymentIntentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stripe.PaymentIntent), args.Error(1)
}

func (m *MockStripeClient) CreateRefund(chargeID string, amount *int64, reason string) (*stripe.Refund, error) {
	args := m.Called(chargeID, amount, reason)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stripe.Refund), args.Error(1)
}

func (m *MockStripeClient) CreateTransfer(amount int64, currency, destination, description string, metadata map[string]string) (*stripe.Transfer, error) {
	args := m.Called(amount, currency, destination, description, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stripe.Transfer), args.Error(1)
}

func (m *MockStripeClient) GetPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	args := m.Called(paymentIntentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stripe.PaymentIntent), args.Error(1)
}

func (m *MockStripeClient) CancelPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	args := m.Called(paymentIntentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stripe.PaymentIntent), args.Error(1)
}
