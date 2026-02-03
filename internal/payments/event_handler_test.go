package payments

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/eventbus"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/richxcame/ride-hailing/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestEventHandler_HandleRideCompleted(t *testing.T) {
	tests := []struct {
		name           string
		eventData      eventbus.RideCompletedData
		payments       []*models.Payment
		getPaymentsErr error
		setupMocks     func(mockRepo *mocks.MockPaymentsRepository, payments []*models.Payment)
		wantErr        bool
		errContains    string
	}{
		{
			name: "successful payout for completed payment",
			eventData: eventbus.RideCompletedData{
				RideID:      uuid.New(),
				RiderID:     uuid.New(),
				DriverID:    uuid.New(),
				FareAmount:  100.00,
				DistanceKm:  10.5,
				DurationMin: 20,
				CompletedAt: time.Now(),
			},
			payments: []*models.Payment{
				{
					ID:       uuid.New(),
					Status:   "completed",
					Amount:   100.00,
					DriverID: uuid.New(),
				},
			},
			setupMocks: func(mockRepo *mocks.MockPaymentsRepository, payments []*models.Payment) {
				payment := payments[0]
				wallet := &models.Wallet{
					ID:       uuid.New(),
					UserID:   payment.DriverID,
					Balance:  50.00,
					Currency: "usd",
					IsActive: true,
				}
				mockRepo.On("GetPaymentByID", mock.Anything, payment.ID).Return(payment, nil)
				mockRepo.On("GetWalletByUserID", mock.Anything, payment.DriverID).Return(wallet, nil)
				mockRepo.On("UpdateWalletBalance", mock.Anything, wallet.ID, mock.AnythingOfType("float64")).Return(nil)
				mockRepo.On("CreateWalletTransaction", mock.Anything, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "no completed payment found - only pending",
			eventData: eventbus.RideCompletedData{
				RideID:      uuid.New(),
				RiderID:     uuid.New(),
				DriverID:    uuid.New(),
				FareAmount:  50.00,
				DistanceKm:  5.0,
				DurationMin: 10,
				CompletedAt: time.Now(),
			},
			payments: []*models.Payment{
				{
					ID:       uuid.New(),
					Status:   "pending",
					Amount:   50.00,
					DriverID: uuid.New(),
				},
			},
			setupMocks: func(mockRepo *mocks.MockPaymentsRepository, payments []*models.Payment) {
				// No additional mocks needed - handler should not call PayoutToDriver
			},
			wantErr: false,
		},
		{
			name: "no payments found for ride",
			eventData: eventbus.RideCompletedData{
				RideID:      uuid.New(),
				RiderID:     uuid.New(),
				DriverID:    uuid.New(),
				FareAmount:  75.00,
				DistanceKm:  7.5,
				DurationMin: 15,
				CompletedAt: time.Now(),
			},
			payments:   []*models.Payment{},
			setupMocks: func(mockRepo *mocks.MockPaymentsRepository, payments []*models.Payment) {},
			wantErr:    false,
		},
		{
			name: "error fetching payments from repository",
			eventData: eventbus.RideCompletedData{
				RideID:      uuid.New(),
				RiderID:     uuid.New(),
				DriverID:    uuid.New(),
				FareAmount:  100.00,
				DistanceKm:  10.0,
				DurationMin: 20,
				CompletedAt: time.Now(),
			},
			getPaymentsErr: errors.New("database connection failed"),
			setupMocks:     func(mockRepo *mocks.MockPaymentsRepository, payments []*models.Payment) {},
			wantErr:        true,
			errContains:    "get payments for ride",
		},
		{
			name: "error during payout - wallet not found and create fails",
			eventData: eventbus.RideCompletedData{
				RideID:      uuid.New(),
				RiderID:     uuid.New(),
				DriverID:    uuid.New(),
				FareAmount:  100.00,
				DistanceKm:  10.0,
				DurationMin: 20,
				CompletedAt: time.Now(),
			},
			payments: []*models.Payment{
				{
					ID:       uuid.New(),
					Status:   "completed",
					Amount:   100.00,
					DriverID: uuid.New(),
				},
			},
			setupMocks: func(mockRepo *mocks.MockPaymentsRepository, payments []*models.Payment) {
				payment := payments[0]
				mockRepo.On("GetPaymentByID", mock.Anything, payment.ID).Return(payment, nil)
				mockRepo.On("GetWalletByUserID", mock.Anything, payment.DriverID).Return(nil, errors.New("not found"))
				mockRepo.On("CreateWallet", mock.Anything, mock.AnythingOfType("*models.Wallet")).Return(errors.New("create wallet failed"))
			},
			wantErr:     true,
			errContains: "payout to driver",
		},
		{
			name: "multiple payments - finds first completed one",
			eventData: eventbus.RideCompletedData{
				RideID:      uuid.New(),
				RiderID:     uuid.New(),
				DriverID:    uuid.New(),
				FareAmount:  200.00,
				DistanceKm:  20.0,
				DurationMin: 40,
				CompletedAt: time.Now(),
			},
			payments: []*models.Payment{
				{
					ID:       uuid.New(),
					Status:   "failed",
					Amount:   200.00,
					DriverID: uuid.New(),
				},
				{
					ID:       uuid.New(),
					Status:   "completed",
					Amount:   200.00,
					DriverID: uuid.New(),
				},
			},
			setupMocks: func(mockRepo *mocks.MockPaymentsRepository, payments []*models.Payment) {
				// The handler should process the second payment (completed)
				payment := payments[1]
				wallet := &models.Wallet{
					ID:       uuid.New(),
					UserID:   payment.DriverID,
					Balance:  100.00,
					Currency: "usd",
					IsActive: true,
				}
				mockRepo.On("GetPaymentByID", mock.Anything, payment.ID).Return(payment, nil)
				mockRepo.On("GetWalletByUserID", mock.Anything, payment.DriverID).Return(wallet, nil)
				mockRepo.On("UpdateWalletBalance", mock.Anything, wallet.ID, mock.AnythingOfType("float64")).Return(nil)
				mockRepo.On("CreateWalletTransaction", mock.Anything, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockRepo := new(mocks.MockPaymentsRepository)
			mockStripe := new(mocks.MockStripeClient)

			// Setup GetPaymentsByRideID mock
			mockRepo.On("GetPaymentsByRideID", mock.Anything, tt.eventData.RideID).
				Return(tt.payments, tt.getPaymentsErr).Once()

			// Setup additional mocks
			if tt.setupMocks != nil {
				tt.setupMocks(mockRepo, tt.payments)
			}

			// Create service and handler
			service := NewService(mockRepo, mockStripe, nil)
			handler := NewEventHandler(service)

			// Create event
			eventData, err := json.Marshal(tt.eventData)
			require.NoError(t, err)

			event := &eventbus.Event{
				ID:        uuid.New().String(),
				Type:      "ride.completed",
				Source:    "rides-service",
				Timestamp: time.Now(),
				Data:      eventData,
			}

			// Execute handler
			err = handler.handleRideCompleted(context.Background(), event)

			// Assert results
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestEventHandler_HandleRideCompleted_MalformedEvent(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	handler := NewEventHandler(service)

	event := &eventbus.Event{
		ID:        uuid.New().String(),
		Type:      "ride.completed",
		Source:    "test",
		Timestamp: time.Now(),
		Data:      []byte("invalid json{"),
	}

	err := handler.handleRideCompleted(context.Background(), event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal ride completed")
}

func TestNewEventHandler(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)

	handler := NewEventHandler(service)

	assert.NotNil(t, handler)
	assert.Equal(t, service, handler.service)
}

func TestEventHandler_HandleRideCompleted_PaymentNotCompleted(t *testing.T) {
	// Test that PayoutToDriver returns error when payment status is not "completed"
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)

	rideID := uuid.New()
	driverID := uuid.New()
	paymentID := uuid.New()

	eventData := eventbus.RideCompletedData{
		RideID:      rideID,
		RiderID:     uuid.New(),
		DriverID:    driverID,
		FareAmount:  100.00,
		DistanceKm:  10.0,
		DurationMin: 20,
		CompletedAt: time.Now(),
	}

	// Return a payment marked as "completed" in the list
	payment := &models.Payment{
		ID:       paymentID,
		RideID:   rideID,
		DriverID: driverID,
		Status:   "completed",
		Amount:   100.00,
	}

	mockRepo.On("GetPaymentsByRideID", mock.Anything, rideID).Return([]*models.Payment{payment}, nil)

	// But when we fetch it by ID, return it as pending (simulating race condition)
	paymentPending := &models.Payment{
		ID:       paymentID,
		RideID:   rideID,
		DriverID: driverID,
		Status:   "pending", // Changed status
		Amount:   100.00,
	}
	mockRepo.On("GetPaymentByID", mock.Anything, paymentID).Return(paymentPending, nil)

	service := NewService(mockRepo, mockStripe, nil)
	handler := NewEventHandler(service)

	eventDataBytes, err := json.Marshal(eventData)
	require.NoError(t, err)

	event := &eventbus.Event{
		ID:        uuid.New().String(),
		Type:      "ride.completed",
		Source:    "test",
		Timestamp: time.Now(),
		Data:      eventDataBytes,
	}

	err = handler.handleRideCompleted(context.Background(), event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payout to driver")
	mockRepo.AssertExpectations(t)
}
