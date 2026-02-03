package payments

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/richxcame/ride-hailing/test/mocks"
	"github.com/stripe/stripe-go/v83"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_ProcessRidePayment_Wallet_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()
	amount := 25.50

	mockRepo.On("ProcessPaymentWithWallet", ctx, mock.AnythingOfType("*models.Payment"), mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	// Act
	payment, err := service.ProcessRidePayment(ctx, rideID, riderID, driverID, amount, "wallet")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, payment)
	assert.Equal(t, rideID, payment.RideID)
	assert.Equal(t, amount, payment.Amount)
	assert.Equal(t, "wallet", payment.PaymentMethod)
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessRidePayment_Stripe_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()
	amount := 25.50

	piID := "pi_test123"
	mockPI := &stripe.PaymentIntent{
		ID:     piID,
		Status: stripe.PaymentIntentStatusSucceeded,
	}

	mockStripe.On("CreatePaymentIntent", int64(2550), "usd", "", mock.Anything, mock.Anything).Return(mockPI, nil)
	mockRepo.On("CreatePayment", ctx, mock.AnythingOfType("*models.Payment")).Return(nil)

	// Act
	payment, err := service.ProcessRidePayment(ctx, rideID, riderID, driverID, amount, "stripe")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, payment)
	assert.Equal(t, piID, *payment.StripePaymentID)
	assert.Equal(t, string(stripe.PaymentIntentStatusSucceeded), payment.Status)
	mockRepo.AssertExpectations(t)
	mockStripe.AssertExpectations(t)
}

func TestService_ProcessRidePayment_InvalidMethod(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()
	amount := 25.50

	// Act
	payment, err := service.ProcessRidePayment(ctx, rideID, riderID, driverID, amount, "invalid_method")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, payment)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.Code)
}

func TestService_ProcessRidePayment_Stripe_CreatePaymentIntentFails(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()
	amount := 25.50

	mockStripe.On("CreatePaymentIntent", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("stripe error"))

	// Act
	payment, err := service.ProcessRidePayment(ctx, rideID, riderID, driverID, amount, "stripe")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, payment)
	mockStripe.AssertExpectations(t)
}

func TestService_TopUpWallet_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	amount := 100.0
	walletID := uuid.New()

	existingWallet := &models.Wallet{
		ID:       walletID,
		UserID:   userID,
		Balance:  50.0,
		Currency: "usd",
		IsActive: true,
	}

	piID := "pi_topup123"
	mockPI := &stripe.PaymentIntent{
		ID:     piID,
		Status: stripe.PaymentIntentStatusSucceeded,
	}

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(existingWallet, nil)
	mockStripe.On("CreatePaymentIntent", int64(10000), "usd", "", "Wallet top-up", mock.Anything).Return(mockPI, nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	// Act
	transaction, err := service.TopUpWallet(ctx, userID, amount, "pm_test123")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, transaction)
	assert.Equal(t, walletID, transaction.WalletID)
	assert.Equal(t, amount, transaction.Amount)
	assert.Equal(t, "credit", transaction.Type)
	mockRepo.AssertExpectations(t)
	mockStripe.AssertExpectations(t)
}

func TestService_TopUpWallet_CreateWalletIfNotExists(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	amount := 100.0

	piID := "pi_topup123"
	mockPI := &stripe.PaymentIntent{
		ID:     piID,
		Status: stripe.PaymentIntentStatusSucceeded,
	}

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(nil, errors.New("not found"))
	mockRepo.On("CreateWallet", ctx, mock.AnythingOfType("*models.Wallet")).Return(nil)
	mockStripe.On("CreatePaymentIntent", int64(10000), "usd", "", "Wallet top-up", mock.Anything).Return(mockPI, nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	// Act
	transaction, err := service.TopUpWallet(ctx, userID, amount, "pm_test123")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, transaction)
	mockRepo.AssertExpectations(t)
	mockStripe.AssertExpectations(t)
}

func TestService_PayoutToDriver_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	rideID := uuid.New()
	driverID := uuid.New()
	walletID := uuid.New()
	amount := 100.0

	payment := &models.Payment{
		ID:       paymentID,
		RideID:   rideID,
		DriverID: driverID,
		Amount:   amount,
		Status:   "completed",
	}

	driverWallet := &models.Wallet{
		ID:       walletID,
		UserID:   driverID,
		Balance:  50.0,
		Currency: "usd",
		IsActive: true,
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, driverID).Return(driverWallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, 80.0).Return(nil) // 100 - 20% commission
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	// Act
	err := service.PayoutToDriver(ctx, paymentID)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_PayoutToDriver_PaymentNotCompleted(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	payment := &models.Payment{
		ID:     paymentID,
		Amount: 100.0,
		Status: "pending", // Not completed
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)

	// Act
	err := service.PayoutToDriver(ctx, paymentID)

	// Assert
	assert.Error(t, err)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessRefund_RiderCancelled_WithCancellationFee(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	rideID := uuid.New()
	riderID := uuid.New()
	walletID := uuid.New()
	chargeID := "ch_test123"
	amount := 100.0

	payment := &models.Payment{
		ID:              paymentID,
		RideID:          rideID,
		RiderID:         riderID,
		Amount:          amount,
		Status:          "completed",
		PaymentMethod:   "wallet",
		StripeChargeID:  &chargeID,
	}

	wallet := &models.Wallet{
		ID:       walletID,
		UserID:   riderID,
		Balance:  10.0,
		Currency: "usd",
		IsActive: true,
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, riderID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, 90.0).Return(nil) // 100 - 10% cancellation fee
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)
	mockRepo.On("UpdatePaymentStatus", ctx, paymentID, "refunded", mock.Anything).Return(nil)

	// Act
	err := service.ProcessRefund(ctx, paymentID, "rider_cancelled")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessRefund_Stripe_FullRefund(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	rideID := uuid.New()
	riderID := uuid.New()
	chargeID := "ch_test123"
	piID := "pi_test123"
	amount := 100.0

	payment := &models.Payment{
		ID:              paymentID,
		RideID:          rideID,
		RiderID:         riderID,
		Amount:          amount,
		Status:          "completed",
		PaymentMethod:   "stripe",
		StripePaymentID: &piID,
		StripeChargeID:  &chargeID,
	}

	mockRefund := &stripe.Refund{
		ID:     "re_test123",
		Amount: 10000,
		Status: "succeeded",
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockStripe.On("CreateRefund", chargeID, mock.AnythingOfType("*int64"), "driver_cancelled").Return(mockRefund, nil)
	mockRepo.On("UpdatePaymentStatus", ctx, paymentID, "refunded", mock.Anything).Return(nil)

	// Act
	err := service.ProcessRefund(ctx, paymentID, "driver_cancelled")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockStripe.AssertExpectations(t)
}

func TestService_ProcessRefund_AlreadyRefunded(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	payment := &models.Payment{
		ID:     paymentID,
		Amount: 100.0,
		Status: "refunded", // Already refunded
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)

	// Act
	err := service.ProcessRefund(ctx, paymentID, "any_reason")

	// Assert
	assert.Error(t, err)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestService_GetWallet_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	walletID := uuid.New()
	wallet := &models.Wallet{
		ID:       walletID,
		UserID:   userID,
		Balance:  150.0,
		Currency: "usd",
		IsActive: true,
	}

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(wallet, nil)

	// Act
	result, err := service.GetWallet(ctx, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, walletID, result.ID)
	assert.Equal(t, 150.0, result.Balance)
	mockRepo.AssertExpectations(t)
}

func TestService_GetWalletTransactions_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	walletID := uuid.New()
	wallet := &models.Wallet{
		ID:       walletID,
		UserID:   userID,
		Balance:  150.0,
		Currency: "usd",
		IsActive: true,
	}

	transactions := []*models.WalletTransaction{
		{ID: uuid.New(), WalletID: walletID, Type: "credit", Amount: 100.0},
		{ID: uuid.New(), WalletID: walletID, Type: "debit", Amount: 25.0},
	}

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(wallet, nil)
	mockRepo.On("GetWalletTransactionsWithTotal", ctx, walletID, 10, 0).Return(transactions, int64(2), nil)

	// Act
	result, total, err := service.GetWalletTransactions(ctx, userID, 10, 0)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, int64(2), total)
	mockRepo.AssertExpectations(t)
}

func TestService_ConfirmWalletTopUp_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	walletID := uuid.New()
	amount := 100.0
	piID := "pi_test123"

	wallet := &models.Wallet{
		ID:       walletID,
		UserID:   userID,
		Balance:  50.0,
		Currency: "usd",
		IsActive: true,
	}

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, amount).Return(nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	// Act
	err := service.ConfirmWalletTopUp(ctx, userID, amount, piID)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_HandleStripeWebhook_PaymentIntentSucceeded(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	// Act
	err := service.HandleStripeWebhook(ctx, "payment_intent.succeeded", "pi_test123")

	// Assert
	assert.NoError(t, err)
}

func TestService_HandleStripeWebhook_PaymentIntentFailed(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	// Act
	err := service.HandleStripeWebhook(ctx, "payment_intent.payment_failed", "pi_test123")

	// Assert
	assert.NoError(t, err)
}

func TestCommissionCalculation(t *testing.T) {
	tests := []struct {
		name               string
		totalAmount        float64
		expectedCommission float64
		expectedEarnings   float64
	}{
		{
			name:               "100 dollar ride",
			totalAmount:        100.0,
			expectedCommission: 20.0,
			expectedEarnings:   80.0,
		},
		{
			name:               "50 dollar ride",
			totalAmount:        50.0,
			expectedCommission: 10.0,
			expectedEarnings:   40.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commission := tt.totalAmount * defaultCommissionRate
			earnings := tt.totalAmount - commission

			assert.InDelta(t, tt.expectedCommission, commission, 0.01)
			assert.InDelta(t, tt.expectedEarnings, earnings, 0.01)
		})
	}
}

func TestCancellationFeeCalculation(t *testing.T) {
	tests := []struct {
		name           string
		amount         float64
		expectedFee    float64
		expectedRefund float64
	}{
		{
			name:           "100 dollar cancellation",
			amount:         100.0,
			expectedFee:    10.0,
			expectedRefund: 90.0,
		},
		{
			name:           "50 dollar cancellation",
			amount:         50.0,
			expectedFee:    5.0,
			expectedRefund: 45.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fee := tt.amount * defaultCancellationFeeRate
			refund := tt.amount - fee

			assert.InDelta(t, tt.expectedFee, fee, 0.01)
			assert.InDelta(t, tt.expectedRefund, refund, 0.01)
		})
	}
}

// Additional error path tests

func TestService_ProcessRidePayment_Wallet_RepositoryError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()

	mockRepo.On("ProcessPaymentWithWallet", ctx, mock.AnythingOfType("*models.Payment"), mock.AnythingOfType("*models.WalletTransaction")).
		Return(errors.New("insufficient balance"))

	payment, err := service.ProcessRidePayment(ctx, rideID, riderID, driverID, 50.0, "wallet")

	assert.Error(t, err)
	assert.Nil(t, payment)
	assert.Contains(t, err.Error(), "insufficient balance")
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessRidePayment_Stripe_CreatePaymentIntentError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()

	mockStripe.On("CreatePaymentIntent", mock.AnythingOfType("int64"), "usd", "", mock.Anything, mock.Anything).
		Return(nil, errors.New("stripe API error"))

	payment, err := service.ProcessRidePayment(ctx, rideID, riderID, driverID, 50.0, "stripe")

	assert.Error(t, err)
	assert.Nil(t, payment)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 500, appErr.Code)
	mockStripe.AssertExpectations(t)
}

func TestService_ProcessRidePayment_Stripe_CreatePaymentError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()
	piID := "pi_test123"

	pi := &stripe.PaymentIntent{
		ID:     piID,
		Status: stripe.PaymentIntentStatusRequiresPaymentMethod,
	}

	mockStripe.On("CreatePaymentIntent", mock.AnythingOfType("int64"), "usd", "", mock.Anything, mock.Anything).
		Return(pi, nil)
	mockRepo.On("CreatePayment", ctx, mock.AnythingOfType("*models.Payment")).
		Return(errors.New("database error"))

	payment, err := service.ProcessRidePayment(ctx, rideID, riderID, driverID, 50.0, "stripe")

	assert.Error(t, err)
	assert.Nil(t, payment)
	assert.Contains(t, err.Error(), "database error")
	mockRepo.AssertExpectations(t)
	mockStripe.AssertExpectations(t)
}

func TestService_TopUpWallet_CreateWalletError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(nil, errors.New("wallet not found"))
	mockRepo.On("CreateWallet", ctx, mock.AnythingOfType("*models.Wallet")).
		Return(errors.New("failed to create wallet"))

	tx, err := service.TopUpWallet(ctx, userID, 100.0, "pm_test123")

	assert.Error(t, err)
	assert.Nil(t, tx)
	assert.Contains(t, err.Error(), "failed to create wallet")
	mockRepo.AssertExpectations(t)
}

func TestService_TopUpWallet_StripeError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	walletID := uuid.New()
	wallet := &models.Wallet{
		ID:       walletID,
		UserID:   userID,
		Balance:  50.0,
		Currency: "usd",
	}

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(wallet, nil)
	mockStripe.On("CreatePaymentIntent", mock.AnythingOfType("int64"), "usd", "", mock.Anything, mock.Anything).
		Return(nil, errors.New("stripe error"))

	tx, err := service.TopUpWallet(ctx, userID, 100.0, "pm_test123")

	assert.Error(t, err)
	assert.Nil(t, tx)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	mockRepo.AssertExpectations(t)
	mockStripe.AssertExpectations(t)
}

func TestService_TopUpWallet_CreateTransactionError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	walletID := uuid.New()
	wallet := &models.Wallet{
		ID:       walletID,
		UserID:   userID,
		Balance:  50.0,
		Currency: "usd",
	}

	pi := &stripe.PaymentIntent{
		ID:     "pi_test123",
		Status: stripe.PaymentIntentStatusRequiresPaymentMethod,
	}

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(wallet, nil)
	mockStripe.On("CreatePaymentIntent", mock.AnythingOfType("int64"), "usd", "", mock.Anything, mock.Anything).
		Return(pi, nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).
		Return(errors.New("transaction error"))

	tx, err := service.TopUpWallet(ctx, userID, 100.0, "pm_test123")

	assert.Error(t, err)
	assert.Nil(t, tx)
	mockRepo.AssertExpectations(t)
	mockStripe.AssertExpectations(t)
}

func TestService_ConfirmWalletTopUp_GetWalletError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(nil, errors.New("wallet not found"))

	err := service.ConfirmWalletTopUp(ctx, userID, 100.0, "pi_test123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wallet not found")
	mockRepo.AssertExpectations(t)
}

func TestService_ConfirmWalletTopUp_UpdateBalanceError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	walletID := uuid.New()
	wallet := &models.Wallet{
		ID:      walletID,
		UserID:  userID,
		Balance: 50.0,
	}

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, 100.0).Return(errors.New("update error"))

	err := service.ConfirmWalletTopUp(ctx, userID, 100.0, "pi_test123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update error")
	mockRepo.AssertExpectations(t)
}

func TestService_ConfirmWalletTopUp_CreateTransactionError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	walletID := uuid.New()
	wallet := &models.Wallet{
		ID:      walletID,
		UserID:  userID,
		Balance: 50.0,
	}

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, 100.0).Return(nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).
		Return(errors.New("transaction create error"))

	err := service.ConfirmWalletTopUp(ctx, userID, 100.0, "pi_test123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transaction create error")
	mockRepo.AssertExpectations(t)
}

func TestService_PayoutToDriver_GetPaymentError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(nil, errors.New("payment not found"))

	err := service.PayoutToDriver(ctx, paymentID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payment not found")
	mockRepo.AssertExpectations(t)
}

func TestService_PayoutToDriver_CreateWalletError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	payment := &models.Payment{
		ID:       paymentID,
		Status:   "completed",
		Amount:   100.0,
		DriverID: driverID,
		RideID:   rideID,
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, driverID).Return(nil, errors.New("wallet not found"))
	mockRepo.On("CreateWallet", ctx, mock.AnythingOfType("*models.Wallet")).
		Return(errors.New("create wallet error"))

	err := service.PayoutToDriver(ctx, paymentID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create wallet error")
	mockRepo.AssertExpectations(t)
}

func TestService_PayoutToDriver_UpdateBalanceError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	walletID := uuid.New()

	payment := &models.Payment{
		ID:       paymentID,
		Status:   "completed",
		Amount:   100.0,
		DriverID: driverID,
		RideID:   rideID,
	}

	wallet := &models.Wallet{
		ID:      walletID,
		UserID:  driverID,
		Balance: 50.0,
	}

	expectedEarnings := 80.0 // 100 - 20% commission

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, driverID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, expectedEarnings).
		Return(errors.New("update balance error"))

	err := service.PayoutToDriver(ctx, paymentID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update balance error")
	mockRepo.AssertExpectations(t)
}

func TestService_PayoutToDriver_CreateTransactionError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	walletID := uuid.New()

	payment := &models.Payment{
		ID:       paymentID,
		Status:   "completed",
		Amount:   100.0,
		DriverID: driverID,
		RideID:   rideID,
	}

	wallet := &models.Wallet{
		ID:      walletID,
		UserID:  driverID,
		Balance: 50.0,
	}

	expectedEarnings := 80.0

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, driverID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, expectedEarnings).Return(nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).
		Return(errors.New("create transaction error"))

	err := service.PayoutToDriver(ctx, paymentID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create transaction error")
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessRefund_GetPaymentError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(nil, errors.New("payment not found"))

	err := service.ProcessRefund(ctx, paymentID, "rider_cancelled")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payment not found")
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessRefund_Wallet_GetWalletError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	riderID := uuid.New()
	rideID := uuid.New()

	payment := &models.Payment{
		ID:            paymentID,
		Status:        "completed",
		Amount:        100.0,
		PaymentMethod: "wallet",
		RiderID:       riderID,
		RideID:        rideID,
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, riderID).Return(nil, errors.New("wallet not found"))

	err := service.ProcessRefund(ctx, paymentID, "driver_cancelled")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wallet not found")
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessRefund_Wallet_UpdateBalanceError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	riderID := uuid.New()
	rideID := uuid.New()
	walletID := uuid.New()

	payment := &models.Payment{
		ID:            paymentID,
		Status:        "completed",
		Amount:        100.0,
		PaymentMethod: "wallet",
		RiderID:       riderID,
		RideID:        rideID,
	}

	wallet := &models.Wallet{
		ID:      walletID,
		UserID:  riderID,
		Balance: 50.0,
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, riderID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, 100.0).
		Return(errors.New("update balance error"))

	err := service.ProcessRefund(ctx, paymentID, "driver_cancelled")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update balance error")
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessRefund_Wallet_CreateTransactionError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	riderID := uuid.New()
	rideID := uuid.New()
	walletID := uuid.New()

	payment := &models.Payment{
		ID:            paymentID,
		Status:        "completed",
		Amount:        100.0,
		PaymentMethod: "wallet",
		RiderID:       riderID,
		RideID:        rideID,
	}

	wallet := &models.Wallet{
		ID:      walletID,
		UserID:  riderID,
		Balance: 50.0,
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, riderID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, 100.0).Return(nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).
		Return(errors.New("create transaction error"))

	err := service.ProcessRefund(ctx, paymentID, "driver_cancelled")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create transaction error")
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessRefund_Wallet_UpdateStatusError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	riderID := uuid.New()
	rideID := uuid.New()
	walletID := uuid.New()

	payment := &models.Payment{
		ID:            paymentID,
		Status:        "completed",
		Amount:        100.0,
		PaymentMethod: "wallet",
		RiderID:       riderID,
		RideID:        rideID,
	}

	wallet := &models.Wallet{
		ID:      walletID,
		UserID:  riderID,
		Balance: 50.0,
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, riderID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, 100.0).Return(nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)
	mockRepo.On("UpdatePaymentStatus", ctx, paymentID, "refunded", mock.Anything).
		Return(errors.New("update status error"))

	err := service.ProcessRefund(ctx, paymentID, "driver_cancelled")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update status error")
	mockRepo.AssertExpectations(t)
}

func TestService_GetWalletTransactions_GetWalletError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(nil, errors.New("wallet not found"))

	txs, total, err := service.GetWalletTransactions(ctx, userID, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, txs)
	assert.Equal(t, int64(0), total)
	assert.Contains(t, err.Error(), "wallet not found")
	mockRepo.AssertExpectations(t)
}
