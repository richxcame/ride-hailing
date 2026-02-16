package delivery

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ========================================
// INTERNAL MOCK (implements RepositoryInterface within this package)
// ========================================

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) CreateDelivery(ctx context.Context, d *Delivery, stops []DeliveryStop) error {
	args := m.Called(ctx, d, stops)
	return args.Error(0)
}

func (m *mockRepo) GetDeliveryByID(ctx context.Context, id uuid.UUID) (*Delivery, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Delivery), args.Error(1)
}

func (m *mockRepo) GetDeliveryByTrackingCode(ctx context.Context, code string) (*Delivery, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Delivery), args.Error(1)
}

func (m *mockRepo) AtomicAcceptDelivery(ctx context.Context, deliveryID, driverID uuid.UUID) (bool, error) {
	args := m.Called(ctx, deliveryID, driverID)
	return args.Bool(0), args.Error(1)
}

func (m *mockRepo) UpdateDeliveryStatus(ctx context.Context, deliveryID uuid.UUID, status DeliveryStatus, driverID uuid.UUID) error {
	args := m.Called(ctx, deliveryID, status, driverID)
	return args.Error(0)
}

func (m *mockRepo) AtomicCompleteDelivery(ctx context.Context, deliveryID, driverID uuid.UUID, proofType ProofType, proofPhotoURL, signatureURL *string, finalFare float64) (bool, error) {
	args := m.Called(ctx, deliveryID, driverID, proofType, proofPhotoURL, signatureURL, finalFare)
	return args.Bool(0), args.Error(1)
}

func (m *mockRepo) CancelDelivery(ctx context.Context, deliveryID uuid.UUID, reason string) error {
	args := m.Called(ctx, deliveryID, reason)
	return args.Error(0)
}

func (m *mockRepo) ReturnDelivery(ctx context.Context, deliveryID uuid.UUID, reason string) error {
	args := m.Called(ctx, deliveryID, reason)
	return args.Error(0)
}

func (m *mockRepo) RateDeliveryBySender(ctx context.Context, deliveryID uuid.UUID, rating int, feedback *string) error {
	args := m.Called(ctx, deliveryID, rating, feedback)
	return args.Error(0)
}

func (m *mockRepo) RateDeliveryByDriver(ctx context.Context, deliveryID uuid.UUID, rating int, feedback *string) error {
	args := m.Called(ctx, deliveryID, rating, feedback)
	return args.Error(0)
}

func (m *mockRepo) GetDeliveriesBySender(ctx context.Context, senderID uuid.UUID, filters *DeliveryListFilters, limit, offset int) ([]*Delivery, int64, error) {
	args := m.Called(ctx, senderID, filters, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*Delivery), args.Get(1).(int64), args.Error(2)
}

func (m *mockRepo) GetAvailableDeliveries(ctx context.Context, latitude, longitude float64, radiusKm float64) ([]*Delivery, error) {
	args := m.Called(ctx, latitude, longitude, radiusKm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Delivery), args.Error(1)
}

func (m *mockRepo) GetDriverDeliveries(ctx context.Context, driverID uuid.UUID, limit, offset int) ([]*Delivery, int64, error) {
	args := m.Called(ctx, driverID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*Delivery), args.Get(1).(int64), args.Error(2)
}

func (m *mockRepo) GetActiveDeliveryForDriver(ctx context.Context, driverID uuid.UUID) (*Delivery, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Delivery), args.Error(1)
}

func (m *mockRepo) GetStopsByDeliveryID(ctx context.Context, deliveryID uuid.UUID) ([]DeliveryStop, error) {
	args := m.Called(ctx, deliveryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]DeliveryStop), args.Error(1)
}

func (m *mockRepo) UpdateStopStatus(ctx context.Context, stopID uuid.UUID, status string, photoURL *string) error {
	args := m.Called(ctx, stopID, status, photoURL)
	return args.Error(0)
}

func (m *mockRepo) AddTrackingEvent(ctx context.Context, t *DeliveryTracking) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *mockRepo) GetTrackingByDeliveryID(ctx context.Context, deliveryID uuid.UUID) ([]DeliveryTracking, error) {
	args := m.Called(ctx, deliveryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]DeliveryTracking), args.Error(1)
}

func (m *mockRepo) GetSenderStats(ctx context.Context, senderID uuid.UUID) (*DeliveryStats, error) {
	args := m.Called(ctx, senderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DeliveryStats), args.Error(1)
}

// ========================================
// TEST HELPERS
// ========================================

func newTestService(repo RepositoryInterface) *Service {
	return NewService(repo)
}

func ptrString(s string) *string {
	return &s
}

func ptrFloat64(f float64) *float64 {
	return &f
}

// ========================================
// TESTS: GetEstimate (Fare Estimation)
// ========================================

func TestGetEstimate(t *testing.T) {
	tests := []struct {
		name     string
		req      *DeliveryEstimateRequest
		validate func(t *testing.T, resp *DeliveryEstimateResponse)
	}{
		{
			name: "success - standard priority envelope",
			req: &DeliveryEstimateRequest{
				PickupLatitude:   40.7128,
				PickupLongitude:  -74.0060,
				DropoffLatitude:  40.7580,
				DropoffLongitude: -73.9855,
				PackageSize:      PackageSizeEnvelope,
				Priority:         DeliveryPriorityStandard,
			},
			validate: func(t *testing.T, resp *DeliveryEstimateResponse) {
				assert.Equal(t, PackageSizeEnvelope, resp.PackageSize)
				assert.Equal(t, DeliveryPriorityStandard, resp.Priority)
				assert.Equal(t, 0.0, resp.SizeSurcharge) // Envelope has no surcharge
				assert.Equal(t, 0.0, resp.PrioritySurcharge) // Standard has no premium
				assert.Equal(t, 1.0, resp.SurgeMultiplier)
				assert.Equal(t, "USD", resp.Currency)
				assert.Greater(t, resp.EstimatedDistance, 0.0)
				assert.Greater(t, resp.EstimatedDuration, 0)
				assert.Greater(t, resp.BaseFare, 0.0)
				assert.Greater(t, resp.TotalEstimate, 0.0)
			},
		},
		{
			name: "success - express priority with express premium",
			req: &DeliveryEstimateRequest{
				PickupLatitude:   40.7128,
				PickupLongitude:  -74.0060,
				DropoffLatitude:  40.7580,
				DropoffLongitude: -73.9855,
				PackageSize:      PackageSizeSmall,
				Priority:         DeliveryPriorityExpress,
			},
			validate: func(t *testing.T, resp *DeliveryEstimateResponse) {
				assert.Equal(t, DeliveryPriorityExpress, resp.Priority)
				assert.Equal(t, 1.0, resp.SizeSurcharge) // Small package surcharge
				assert.Greater(t, resp.PrioritySurcharge, 0.0) // Express premium should be > 0
				// Express premium should be 50% of base fare
				expectedPrioritySurcharge := resp.BaseFare * 0.5
				assert.InDelta(t, expectedPrioritySurcharge, resp.PrioritySurcharge, 0.01)
			},
		},
		{
			name: "success - medium package size surcharge",
			req: &DeliveryEstimateRequest{
				PickupLatitude:   40.7128,
				PickupLongitude:  -74.0060,
				DropoffLatitude:  40.7580,
				DropoffLongitude: -73.9855,
				PackageSize:      PackageSizeMedium,
				Priority:         DeliveryPriorityStandard,
			},
			validate: func(t *testing.T, resp *DeliveryEstimateResponse) {
				assert.Equal(t, 3.0, resp.SizeSurcharge) // Medium surcharge is $3
			},
		},
		{
			name: "success - large package size surcharge",
			req: &DeliveryEstimateRequest{
				PickupLatitude:   40.7128,
				PickupLongitude:  -74.0060,
				DropoffLatitude:  40.7580,
				DropoffLongitude: -73.9855,
				PackageSize:      PackageSizeLarge,
				Priority:         DeliveryPriorityStandard,
			},
			validate: func(t *testing.T, resp *DeliveryEstimateResponse) {
				assert.Equal(t, 8.0, resp.SizeSurcharge) // Large surcharge is $8
			},
		},
		{
			name: "success - xlarge package size surcharge",
			req: &DeliveryEstimateRequest{
				PickupLatitude:   40.7128,
				PickupLongitude:  -74.0060,
				DropoffLatitude:  40.7580,
				DropoffLongitude: -73.9855,
				PackageSize:      PackageSizeXLarge,
				Priority:         DeliveryPriorityStandard,
			},
			validate: func(t *testing.T, resp *DeliveryEstimateResponse) {
				assert.Equal(t, 15.0, resp.SizeSurcharge) // XLarge surcharge is $15
			},
		},
		{
			name: "success - with intermediate stops increases distance",
			req: &DeliveryEstimateRequest{
				PickupLatitude:   40.7128,
				PickupLongitude:  -74.0060,
				DropoffLatitude:  40.7580,
				DropoffLongitude: -73.9855,
				PackageSize:      PackageSizeSmall,
				Priority:         DeliveryPriorityStandard,
				Stops: []StopInput{
					{Latitude: 40.7300, Longitude: -74.0000},
					{Latitude: 40.7400, Longitude: -73.9950},
				},
			},
			validate: func(t *testing.T, resp *DeliveryEstimateResponse) {
				// With stops, duration should include additional buffer (3 min per stop)
				assert.Greater(t, resp.EstimatedDuration, 10) // At least base 10 + stop buffers
			},
		},
		{
			name: "success - minimum fare applies for short distance",
			req: &DeliveryEstimateRequest{
				PickupLatitude:   40.7128,
				PickupLongitude:  -74.0060,
				DropoffLatitude:  40.7130, // Very close
				DropoffLongitude: -74.0062,
				PackageSize:      PackageSizeEnvelope,
				Priority:         DeliveryPriorityStandard,
			},
			validate: func(t *testing.T, resp *DeliveryEstimateResponse) {
				// Base fare should be at least the minimum fare of $3.0
				assert.GreaterOrEqual(t, resp.BaseFare, 3.0)
			},
		},
		{
			name: "success - scheduled priority no premium",
			req: &DeliveryEstimateRequest{
				PickupLatitude:   40.7128,
				PickupLongitude:  -74.0060,
				DropoffLatitude:  40.7580,
				DropoffLongitude: -73.9855,
				PackageSize:      PackageSizeSmall,
				Priority:         DeliveryPriorityScheduled,
			},
			validate: func(t *testing.T, resp *DeliveryEstimateResponse) {
				assert.Equal(t, DeliveryPriorityScheduled, resp.Priority)
				assert.Equal(t, 0.0, resp.PrioritySurcharge) // Scheduled has no premium
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			svc := newTestService(m)

			resp, err := svc.GetEstimate(context.Background(), tt.req)

			require.NoError(t, err)
			require.NotNil(t, resp)
			tt.validate(t, resp)

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: CreateDelivery
// ========================================

func TestCreateDelivery(t *testing.T) {
	senderID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name       string
		req        *CreateDeliveryRequest
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
		validate   func(t *testing.T, resp *DeliveryResponse)
	}{
		{
			name: "success - standard delivery without stops",
			req: &CreateDeliveryRequest{
				PickupLatitude:     40.7128,
				PickupLongitude:    -74.0060,
				PickupAddress:      "123 Main St, New York",
				PickupContact:      "John Doe",
				PickupPhone:        "+1234567890",
				DropoffLatitude:    40.7580,
				DropoffLongitude:   -73.9855,
				DropoffAddress:     "456 Broadway, New York",
				RecipientName:      "Jane Smith",
				RecipientPhone:     "+0987654321",
				PackageSize:        PackageSizeSmall,
				PackageDescription: "Documents",
				Priority:           DeliveryPriorityStandard,
			},
			setupMocks: func(m *mockRepo) {
				m.On("CreateDelivery", mock.Anything, mock.AnythingOfType("*delivery.Delivery"), mock.AnythingOfType("[]delivery.DeliveryStop")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *DeliveryResponse) {
				require.NotNil(t, resp.Delivery)
				assert.Equal(t, senderID, resp.Delivery.SenderID)
				assert.Equal(t, DeliveryStatusRequested, resp.Delivery.Status)
				assert.Equal(t, DeliveryPriorityStandard, resp.Delivery.Priority)
				assert.Equal(t, PackageSizeSmall, resp.Delivery.PackageSize)
				assert.NotEmpty(t, resp.Delivery.TrackingCode)
				assert.Contains(t, resp.Delivery.TrackingCode, "DLV-")
				assert.NotNil(t, resp.Delivery.ProofPIN)
				assert.Len(t, *resp.Delivery.ProofPIN, 4)
				assert.Greater(t, resp.Delivery.EstimatedFare, 0.0)
				assert.Empty(t, resp.Stops)
			},
		},
		{
			name: "success - express delivery with stops",
			req: &CreateDeliveryRequest{
				PickupLatitude:     40.7128,
				PickupLongitude:    -74.0060,
				PickupAddress:      "123 Main St, New York",
				PickupContact:      "John Doe",
				PickupPhone:        "+1234567890",
				DropoffLatitude:    40.7580,
				DropoffLongitude:   -73.9855,
				DropoffAddress:     "456 Broadway, New York",
				RecipientName:      "Jane Smith",
				RecipientPhone:     "+0987654321",
				PackageSize:        PackageSizeMedium,
				PackageDescription: "Electronics",
				IsFragile:          true,
				RequiresSignature:  true,
				Priority:           DeliveryPriorityExpress,
				Stops: []StopInput{
					{
						Latitude:     40.7300,
						Longitude:    -74.0000,
						Address:      "789 Stop St",
						ContactName:  "Stop Contact",
						ContactPhone: "+1111111111",
					},
				},
			},
			setupMocks: func(m *mockRepo) {
				m.On("CreateDelivery", mock.Anything, mock.AnythingOfType("*delivery.Delivery"), mock.AnythingOfType("[]delivery.DeliveryStop")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *DeliveryResponse) {
				assert.Equal(t, DeliveryPriorityExpress, resp.Delivery.Priority)
				assert.True(t, resp.Delivery.IsFragile)
				assert.True(t, resp.Delivery.RequiresSignature)
				assert.Len(t, resp.Stops, 1)
				assert.Equal(t, 1, resp.Stops[0].StopOrder)
				assert.Equal(t, "pending", resp.Stops[0].Status)
			},
		},
		{
			name: "success - scheduled delivery with declared value",
			req: &CreateDeliveryRequest{
				PickupLatitude:     40.7128,
				PickupLongitude:    -74.0060,
				PickupAddress:      "123 Main St, New York",
				PickupContact:      "John Doe",
				PickupPhone:        "+1234567890",
				DropoffLatitude:    40.7580,
				DropoffLongitude:   -73.9855,
				DropoffAddress:     "456 Broadway, New York",
				RecipientName:      "Jane Smith",
				RecipientPhone:     "+0987654321",
				PackageSize:        PackageSizeLarge,
				PackageDescription: "Expensive Equipment",
				DeclaredValue:      ptrFloat64(5000.0),
				Priority:           DeliveryPriorityScheduled,
				ScheduledPickupAt:  func() *time.Time { t := time.Now().Add(24 * time.Hour); return &t }(),
			},
			setupMocks: func(m *mockRepo) {
				m.On("CreateDelivery", mock.Anything, mock.AnythingOfType("*delivery.Delivery"), mock.AnythingOfType("[]delivery.DeliveryStop")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *DeliveryResponse) {
				assert.Equal(t, DeliveryPriorityScheduled, resp.Delivery.Priority)
				assert.NotNil(t, resp.Delivery.DeclaredValue)
				assert.Equal(t, 5000.0, *resp.Delivery.DeclaredValue)
				assert.NotNil(t, resp.Delivery.ScheduledPickupAt)
			},
		},
		{
			name: "error - invalid package size",
			req: &CreateDeliveryRequest{
				PickupLatitude:     40.7128,
				PickupLongitude:    -74.0060,
				PickupAddress:      "123 Main St",
				PickupContact:      "John",
				PickupPhone:        "+1234567890",
				DropoffLatitude:    40.7580,
				DropoffLongitude:   -73.9855,
				DropoffAddress:     "456 Broadway",
				RecipientName:      "Jane",
				RecipientPhone:     "+0987654321",
				PackageSize:        PackageSize("invalid"),
				PackageDescription: "Test",
				Priority:           DeliveryPriorityStandard,
			},
			setupMocks: func(m *mockRepo) {},
			wantErr:    true,
			errContain: "invalid package size",
		},
		{
			name: "error - invalid priority",
			req: &CreateDeliveryRequest{
				PickupLatitude:     40.7128,
				PickupLongitude:    -74.0060,
				PickupAddress:      "123 Main St",
				PickupContact:      "John",
				PickupPhone:        "+1234567890",
				DropoffLatitude:    40.7580,
				DropoffLongitude:   -73.9855,
				DropoffAddress:     "456 Broadway",
				RecipientName:      "Jane",
				RecipientPhone:     "+0987654321",
				PackageSize:        PackageSizeSmall,
				PackageDescription: "Test",
				Priority:           DeliveryPriority("invalid"),
			},
			setupMocks: func(m *mockRepo) {},
			wantErr:    true,
			errContain: "invalid delivery priority",
		},
		{
			name: "error - repository create fails",
			req: &CreateDeliveryRequest{
				PickupLatitude:     40.7128,
				PickupLongitude:    -74.0060,
				PickupAddress:      "123 Main St",
				PickupContact:      "John",
				PickupPhone:        "+1234567890",
				DropoffLatitude:    40.7580,
				DropoffLongitude:   -73.9855,
				DropoffAddress:     "456 Broadway",
				RecipientName:      "Jane",
				RecipientPhone:     "+0987654321",
				PackageSize:        PackageSizeSmall,
				PackageDescription: "Test",
				Priority:           DeliveryPriorityStandard,
			},
			setupMocks: func(m *mockRepo) {
				m.On("CreateDelivery", mock.Anything, mock.AnythingOfType("*delivery.Delivery"), mock.AnythingOfType("[]delivery.DeliveryStop")).Return(errors.New("database error"))
			},
			wantErr:    true,
			errContain: "create delivery",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			resp, err := svc.CreateDelivery(context.Background(), senderID, tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				tt.validate(t, resp)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetDelivery
// ========================================

func TestGetDelivery(t *testing.T) {
	senderID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	deliveryID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	otherUserID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	tests := []struct {
		name       string
		userID     uuid.UUID
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
		validate   func(t *testing.T, resp *DeliveryResponse)
	}{
		{
			name:   "success - sender can view their delivery",
			userID: senderID,
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					Status:   DeliveryStatusRequested,
					ProofPIN: ptrString("1234"),
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("GetStopsByDeliveryID", mock.Anything, deliveryID).Return([]DeliveryStop{}, nil)
				m.On("GetTrackingByDeliveryID", mock.Anything, deliveryID).Return([]DeliveryTracking{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *DeliveryResponse) {
				assert.Equal(t, deliveryID, resp.Delivery.ID)
				assert.Nil(t, resp.Delivery.ProofPIN) // PIN should be hidden
			},
		},
		{
			name:   "success - driver can view assigned delivery",
			userID: driverID,
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusAccepted,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("GetStopsByDeliveryID", mock.Anything, deliveryID).Return([]DeliveryStop{}, nil)
				m.On("GetTrackingByDeliveryID", mock.Anything, deliveryID).Return([]DeliveryTracking{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *DeliveryResponse) {
				assert.Equal(t, deliveryID, resp.Delivery.ID)
			},
		},
		{
			name:   "error - delivery not found",
			userID: senderID,
			setupMocks: func(m *mockRepo) {
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(nil, pgx.ErrNoRows)
			},
			wantErr:    true,
			errContain: "no rows",
		},
		{
			name:   "error - user not authorized (not sender or driver)",
			userID: otherUserID,
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusAccepted,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "forbidden",
		},
		{
			name:   "error - user not authorized (no driver assigned)",
			userID: otherUserID,
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: nil,
					Status:   DeliveryStatusRequested,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			resp, err := svc.GetDelivery(context.Background(), tt.userID, deliveryID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				tt.validate(t, resp)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: TrackDelivery (Public Tracking)
// ========================================

func TestTrackDelivery(t *testing.T) {
	deliveryID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name         string
		trackingCode string
		setupMocks   func(m *mockRepo)
		wantErr      bool
		errContain   string
		validate     func(t *testing.T, resp *DeliveryResponse)
	}{
		{
			name:         "success - track delivery by code",
			trackingCode: "DLV-ABCDE-12345",
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:             deliveryID,
					TrackingCode:   "DLV-ABCDE-12345",
					Status:         DeliveryStatusInTransit,
					PickupPhone:    "+1234567890",
					RecipientPhone: "+0987654321",
					DeclaredValue:  ptrFloat64(1000.0),
					ProofPIN:       ptrString("1234"),
				}
				m.On("GetDeliveryByTrackingCode", mock.Anything, "DLV-ABCDE-12345").Return(delivery, nil)
				m.On("GetTrackingByDeliveryID", mock.Anything, deliveryID).Return([]DeliveryTracking{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *DeliveryResponse) {
				// Sensitive data should be stripped for public tracking
				assert.Nil(t, resp.Delivery.ProofPIN)
				assert.Empty(t, resp.Delivery.PickupPhone)
				assert.Empty(t, resp.Delivery.RecipientPhone)
				assert.Nil(t, resp.Delivery.DeclaredValue)
			},
		},
		{
			name:         "error - tracking code not found",
			trackingCode: "DLV-XXXXX-XXXXX",
			setupMocks: func(m *mockRepo) {
				m.On("GetDeliveryByTrackingCode", mock.Anything, "DLV-XXXXX-XXXXX").Return(nil, pgx.ErrNoRows)
			},
			wantErr:    true,
			errContain: "no rows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			resp, err := svc.TrackDelivery(context.Background(), tt.trackingCode)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				tt.validate(t, resp)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: AcceptDelivery (Driver Assignment)
// ========================================

func TestAcceptDelivery(t *testing.T) {
	senderID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	deliveryID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
		validate   func(t *testing.T, resp *DeliveryResponse)
	}{
		{
			name: "success - driver accepts delivery",
			setupMocks: func(m *mockRepo) {
				m.On("GetActiveDeliveryForDriver", mock.Anything, driverID).Return(nil, nil)
				m.On("AtomicAcceptDelivery", mock.Anything, deliveryID, driverID).Return(true, nil)
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusAccepted,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("AddTrackingEvent", mock.Anything, mock.AnythingOfType("*delivery.DeliveryTracking")).Return(nil)
				m.On("GetStopsByDeliveryID", mock.Anything, deliveryID).Return([]DeliveryStop{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *DeliveryResponse) {
				assert.Equal(t, DeliveryStatusAccepted, resp.Delivery.Status)
				assert.Equal(t, driverID, *resp.Delivery.DriverID)
			},
		},
		{
			name: "error - driver already has active delivery",
			setupMocks: func(m *mockRepo) {
				activeDelivery := &Delivery{
					ID:     uuid.MustParse("55555555-5555-5555-5555-555555555555"),
					Status: DeliveryStatusInTransit,
				}
				m.On("GetActiveDeliveryForDriver", mock.Anything, driverID).Return(activeDelivery, nil)
			},
			wantErr:    true,
			errContain: "resource conflict",
		},
		{
			name: "error - delivery already accepted by another driver",
			setupMocks: func(m *mockRepo) {
				m.On("GetActiveDeliveryForDriver", mock.Anything, driverID).Return(nil, nil)
				m.On("AtomicAcceptDelivery", mock.Anything, deliveryID, driverID).Return(false, nil)
			},
			wantErr:    true,
			errContain: "resource conflict",
		},
		{
			name: "error - atomic accept fails",
			setupMocks: func(m *mockRepo) {
				m.On("GetActiveDeliveryForDriver", mock.Anything, driverID).Return(nil, nil)
				m.On("AtomicAcceptDelivery", mock.Anything, deliveryID, driverID).Return(false, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			resp, err := svc.AcceptDelivery(context.Background(), deliveryID, driverID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				tt.validate(t, resp)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: UpdateStatus (State Transitions)
// ========================================

func TestUpdateStatus(t *testing.T) {
	senderID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	otherDriverID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	deliveryID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	// Test valid transitions
	validTransitions := []struct {
		from DeliveryStatus
		to   DeliveryStatus
	}{
		// From Requested
		{DeliveryStatusRequested, DeliveryStatusAccepted},
		{DeliveryStatusRequested, DeliveryStatusCancelled},
		// From Accepted
		{DeliveryStatusAccepted, DeliveryStatusPickingUp},
		{DeliveryStatusAccepted, DeliveryStatusPickedUp},
		{DeliveryStatusAccepted, DeliveryStatusCancelled},
		// From PickingUp
		{DeliveryStatusPickingUp, DeliveryStatusPickedUp},
		{DeliveryStatusPickingUp, DeliveryStatusCancelled},
		// From PickedUp
		{DeliveryStatusPickedUp, DeliveryStatusInTransit},
		// From InTransit
		{DeliveryStatusInTransit, DeliveryStatusArrived},
		{DeliveryStatusInTransit, DeliveryStatusDelivered},
		{DeliveryStatusInTransit, DeliveryStatusReturned},
		// From Arrived
		{DeliveryStatusArrived, DeliveryStatusDelivered},
		{DeliveryStatusArrived, DeliveryStatusReturned},
		{DeliveryStatusArrived, DeliveryStatusFailed},
	}

	for _, tr := range validTransitions {
		t.Run("valid transition: "+string(tr.from)+" -> "+string(tr.to), func(t *testing.T) {
			m := new(mockRepo)
			delivery := &Delivery{
				ID:       deliveryID,
				SenderID: senderID,
				DriverID: &driverID,
				Status:   tr.from,
			}
			m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			m.On("UpdateDeliveryStatus", mock.Anything, deliveryID, tr.to, driverID).Return(nil)
			m.On("AddTrackingEvent", mock.Anything, mock.AnythingOfType("*delivery.DeliveryTracking")).Return(nil)

			svc := newTestService(m)
			err := svc.UpdateStatus(context.Background(), deliveryID, driverID, tr.to)

			require.NoError(t, err)
			m.AssertExpectations(t)
		})
	}

	// Test invalid transitions
	invalidTransitions := []struct {
		from DeliveryStatus
		to   DeliveryStatus
	}{
		// Cannot go backwards
		{DeliveryStatusPickedUp, DeliveryStatusAccepted},
		{DeliveryStatusInTransit, DeliveryStatusPickedUp},
		{DeliveryStatusDelivered, DeliveryStatusInTransit},
		// Cannot skip states
		{DeliveryStatusRequested, DeliveryStatusInTransit},
		{DeliveryStatusAccepted, DeliveryStatusDelivered},
		// Terminal states cannot transition
		{DeliveryStatusDelivered, DeliveryStatusCancelled},
		{DeliveryStatusCancelled, DeliveryStatusAccepted},
		{DeliveryStatusReturned, DeliveryStatusDelivered},
		{DeliveryStatusFailed, DeliveryStatusDelivered},
	}

	for _, tr := range invalidTransitions {
		t.Run("invalid transition: "+string(tr.from)+" -> "+string(tr.to), func(t *testing.T) {
			m := new(mockRepo)
			delivery := &Delivery{
				ID:       deliveryID,
				SenderID: senderID,
				DriverID: &driverID,
				Status:   tr.from,
			}
			m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)

			svc := newTestService(m)
			err := svc.UpdateStatus(context.Background(), deliveryID, driverID, tr.to)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "cannot transition")
			m.AssertExpectations(t)
		})
	}

	// Other error cases
	t.Run("error - delivery not found", func(t *testing.T) {
		m := new(mockRepo)
		m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(nil, pgx.ErrNoRows)

		svc := newTestService(m)
		err := svc.UpdateStatus(context.Background(), deliveryID, driverID, DeliveryStatusPickingUp)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no rows")
		m.AssertExpectations(t)
	})

	t.Run("error - not your delivery (different driver)", func(t *testing.T) {
		m := new(mockRepo)
		delivery := &Delivery{
			ID:       deliveryID,
			SenderID: senderID,
			DriverID: &driverID,
			Status:   DeliveryStatusAccepted,
		}
		m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)

		svc := newTestService(m)
		err := svc.UpdateStatus(context.Background(), deliveryID, otherDriverID, DeliveryStatusPickingUp)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden")
		m.AssertExpectations(t)
	})

	t.Run("error - no driver assigned", func(t *testing.T) {
		m := new(mockRepo)
		delivery := &Delivery{
			ID:       deliveryID,
			SenderID: senderID,
			DriverID: nil,
			Status:   DeliveryStatusRequested,
		}
		m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)

		svc := newTestService(m)
		err := svc.UpdateStatus(context.Background(), deliveryID, driverID, DeliveryStatusAccepted)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden")
		m.AssertExpectations(t)
	})
}

// ========================================
// TESTS: ConfirmPickup
// ========================================

func TestConfirmPickup(t *testing.T) {
	senderID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	otherDriverID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	deliveryID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name       string
		req        *ConfirmPickupRequest
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
	}{
		{
			name: "success - confirm pickup from picking_up status",
			req:  &ConfirmPickupRequest{},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusPickingUp,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("UpdateDeliveryStatus", mock.Anything, deliveryID, DeliveryStatusPickedUp, driverID).Return(nil)
				m.On("AddTrackingEvent", mock.Anything, mock.AnythingOfType("*delivery.DeliveryTracking")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - confirm pickup from accepted status",
			req:  &ConfirmPickupRequest{},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusAccepted,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("UpdateDeliveryStatus", mock.Anything, deliveryID, DeliveryStatusPickedUp, driverID).Return(nil)
				m.On("AddTrackingEvent", mock.Anything, mock.AnythingOfType("*delivery.DeliveryTracking")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - confirm pickup with notes",
			req:  &ConfirmPickupRequest{Notes: ptrString("Package in good condition")},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusPickingUp,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("UpdateDeliveryStatus", mock.Anything, deliveryID, DeliveryStatusPickedUp, driverID).Return(nil)
				m.On("AddTrackingEvent", mock.Anything, mock.AnythingOfType("*delivery.DeliveryTracking")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - delivery not found",
			req:  &ConfirmPickupRequest{},
			setupMocks: func(m *mockRepo) {
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(nil, pgx.ErrNoRows)
			},
			wantErr:    true,
			errContain: "no rows",
		},
		{
			name: "error - not your delivery",
			req:  &ConfirmPickupRequest{},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &otherDriverID,
					Status:   DeliveryStatusPickingUp,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "forbidden",
		},
		{
			name: "error - wrong status (in_transit)",
			req:  &ConfirmPickupRequest{},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusInTransit,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "picking_up or accepted",
		},
		{
			name: "error - wrong status (requested)",
			req:  &ConfirmPickupRequest{},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusRequested,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "picking_up or accepted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.ConfirmPickup(context.Background(), deliveryID, driverID, tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: ConfirmDelivery (PIN/Signature/Photo validation)
// ========================================

func TestConfirmDelivery(t *testing.T) {
	senderID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	otherDriverID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	deliveryID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name       string
		req        *ConfirmDeliveryRequest
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
	}{
		{
			name: "success - confirm with valid PIN",
			req: &ConfirmDeliveryRequest{
				ProofType: ProofTypePIN,
				PIN:       ptrString("1234"),
			},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:            deliveryID,
					SenderID:      senderID,
					DriverID:      &driverID,
					Status:        DeliveryStatusArrived,
					ProofPIN:      ptrString("1234"),
					EstimatedFare: 25.50,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("AtomicCompleteDelivery", mock.Anything, deliveryID, driverID, ProofTypePIN, (*string)(nil), (*string)(nil), 25.50).Return(true, nil)
				m.On("AddTrackingEvent", mock.Anything, mock.AnythingOfType("*delivery.DeliveryTracking")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - confirm with photo",
			req: &ConfirmDeliveryRequest{
				ProofType: ProofTypePhoto,
				PhotoURL:  ptrString("https://example.com/photo.jpg"),
			},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:            deliveryID,
					SenderID:      senderID,
					DriverID:      &driverID,
					Status:        DeliveryStatusInTransit,
					EstimatedFare: 30.00,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("AtomicCompleteDelivery", mock.Anything, deliveryID, driverID, ProofTypePhoto, ptrString("https://example.com/photo.jpg"), (*string)(nil), 30.00).Return(true, nil)
				m.On("AddTrackingEvent", mock.Anything, mock.AnythingOfType("*delivery.DeliveryTracking")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - confirm with signature",
			req: &ConfirmDeliveryRequest{
				ProofType:    ProofTypeSignature,
				SignatureURL: ptrString("https://example.com/signature.png"),
			},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:                deliveryID,
					SenderID:          senderID,
					DriverID:          &driverID,
					Status:            DeliveryStatusArrived,
					RequiresSignature: true,
					EstimatedFare:     50.00,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("AtomicCompleteDelivery", mock.Anything, deliveryID, driverID, ProofTypeSignature, (*string)(nil), ptrString("https://example.com/signature.png"), 50.00).Return(true, nil)
				m.On("AddTrackingEvent", mock.Anything, mock.AnythingOfType("*delivery.DeliveryTracking")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - confirm with contactless",
			req: &ConfirmDeliveryRequest{
				ProofType: ProofTypeContactless,
			},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:            deliveryID,
					SenderID:      senderID,
					DriverID:      &driverID,
					Status:        DeliveryStatusArrived,
					EstimatedFare: 20.00,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("AtomicCompleteDelivery", mock.Anything, deliveryID, driverID, ProofTypeContactless, (*string)(nil), (*string)(nil), 20.00).Return(true, nil)
				m.On("AddTrackingEvent", mock.Anything, mock.AnythingOfType("*delivery.DeliveryTracking")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - delivery not found",
			req: &ConfirmDeliveryRequest{
				ProofType: ProofTypePIN,
				PIN:       ptrString("1234"),
			},
			setupMocks: func(m *mockRepo) {
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(nil, pgx.ErrNoRows)
			},
			wantErr:    true,
			errContain: "no rows",
		},
		{
			name: "error - not your delivery",
			req: &ConfirmDeliveryRequest{
				ProofType: ProofTypePIN,
				PIN:       ptrString("1234"),
			},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &otherDriverID,
					Status:   DeliveryStatusArrived,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "forbidden",
		},
		{
			name: "error - signature required but not provided",
			req: &ConfirmDeliveryRequest{
				ProofType: ProofTypePIN,
				PIN:       ptrString("1234"),
			},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:                deliveryID,
					SenderID:          senderID,
					DriverID:          &driverID,
					Status:            DeliveryStatusArrived,
					RequiresSignature: true,
					ProofPIN:          ptrString("1234"),
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "requires a signature",
		},
		{
			name: "error - incorrect PIN",
			req: &ConfirmDeliveryRequest{
				ProofType: ProofTypePIN,
				PIN:       ptrString("0000"),
			},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusArrived,
					ProofPIN: ptrString("1234"),
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "incorrect delivery PIN",
		},
		{
			name: "error - PIN proof type but no PIN provided",
			req: &ConfirmDeliveryRequest{
				ProofType: ProofTypePIN,
				PIN:       nil,
			},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusArrived,
					ProofPIN: ptrString("1234"),
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "incorrect delivery PIN",
		},
		{
			name: "error - photo proof type but no photo URL",
			req: &ConfirmDeliveryRequest{
				ProofType: ProofTypePhoto,
				PhotoURL:  nil,
			},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusArrived,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "photo proof is required",
		},
		{
			name: "error - signature proof type but no signature URL",
			req: &ConfirmDeliveryRequest{
				ProofType:    ProofTypeSignature,
				SignatureURL: nil,
			},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:                deliveryID,
					SenderID:          senderID,
					DriverID:          &driverID,
					Status:            DeliveryStatusArrived,
					RequiresSignature: true,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "signature is required",
		},
		{
			name: "error - delivery already completed or not in transit",
			req: &ConfirmDeliveryRequest{
				ProofType: ProofTypePIN,
				PIN:       ptrString("1234"),
			},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:            deliveryID,
					SenderID:      senderID,
					DriverID:      &driverID,
					Status:        DeliveryStatusArrived,
					ProofPIN:      ptrString("1234"),
					EstimatedFare: 25.00,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("AtomicCompleteDelivery", mock.Anything, deliveryID, driverID, ProofTypePIN, (*string)(nil), (*string)(nil), 25.00).Return(false, nil)
			},
			wantErr:    true,
			errContain: "resource conflict",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.ConfirmDelivery(context.Background(), deliveryID, driverID, tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: CancelDelivery
// ========================================

func TestCancelDelivery(t *testing.T) {
	senderID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	deliveryID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	otherUserID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	tests := []struct {
		name       string
		userID     uuid.UUID
		reason     string
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
	}{
		{
			name:   "success - sender cancels requested delivery",
			userID: senderID,
			reason: "Changed my mind",
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: nil,
					Status:   DeliveryStatusRequested,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("CancelDelivery", mock.Anything, deliveryID, "Changed my mind").Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "success - sender cancels accepted delivery",
			userID: senderID,
			reason: "Wrong address",
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusAccepted,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("CancelDelivery", mock.Anything, deliveryID, "Wrong address").Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "success - driver cancels before pickup",
			userID: driverID,
			reason: "Cannot reach location",
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusPickingUp,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("CancelDelivery", mock.Anything, deliveryID, "Cannot reach location").Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "error - delivery not found",
			userID: senderID,
			reason: "Cancel",
			setupMocks: func(m *mockRepo) {
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(nil, pgx.ErrNoRows)
			},
			wantErr:    true,
			errContain: "no rows",
		},
		{
			name:   "error - not authorized",
			userID: otherUserID,
			reason: "Cancel",
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusAccepted,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "forbidden",
		},
		{
			name:   "error - driver cannot cancel after pickup (picked_up)",
			userID: driverID,
			reason: "Cancel",
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusPickedUp,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "use return instead",
		},
		{
			name:   "error - driver cannot cancel after pickup (in_transit)",
			userID: driverID,
			reason: "Cancel",
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusInTransit,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "use return instead",
		},
		{
			name:   "error - driver cannot cancel after pickup (arrived)",
			userID: driverID,
			reason: "Cancel",
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusArrived,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "use return instead",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.CancelDelivery(context.Background(), deliveryID, tt.userID, tt.reason)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: ReturnDelivery
// ========================================

func TestReturnDelivery(t *testing.T) {
	senderID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	otherDriverID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	deliveryID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name       string
		reason     string
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
	}{
		{
			name:   "success - return from in_transit",
			reason: "Recipient not available",
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusInTransit,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("ReturnDelivery", mock.Anything, deliveryID, "Recipient not available").Return(nil)
				m.On("AddTrackingEvent", mock.Anything, mock.AnythingOfType("*delivery.DeliveryTracking")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "success - return from arrived",
			reason: "Wrong address",
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusArrived,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("ReturnDelivery", mock.Anything, deliveryID, "Wrong address").Return(nil)
				m.On("AddTrackingEvent", mock.Anything, mock.AnythingOfType("*delivery.DeliveryTracking")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "error - delivery not found",
			reason: "Return",
			setupMocks: func(m *mockRepo) {
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(nil, pgx.ErrNoRows)
			},
			wantErr:    true,
			errContain: "no rows",
		},
		{
			name:   "error - not your delivery",
			reason: "Return",
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &otherDriverID,
					Status:   DeliveryStatusInTransit,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "forbidden",
		},
		{
			name:   "error - cannot return from accepted status",
			reason: "Return",
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusAccepted,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "in transit or arrived",
		},
		{
			name:   "error - cannot return from picked_up status",
			reason: "Return",
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusPickedUp,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "in transit or arrived",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.ReturnDelivery(context.Background(), deliveryID, driverID, tt.reason)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: RateDelivery
// ========================================

func TestRateDelivery(t *testing.T) {
	senderID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	otherUserID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	deliveryID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name       string
		userID     uuid.UUID
		req        *RateDeliveryRequest
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
	}{
		{
			name:   "success - sender rates delivery",
			userID: senderID,
			req: &RateDeliveryRequest{
				Rating:   5,
				Feedback: ptrString("Great service!"),
			},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusDelivered,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("RateDeliveryBySender", mock.Anything, deliveryID, 5, ptrString("Great service!")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "success - driver rates delivery",
			userID: driverID,
			req: &RateDeliveryRequest{
				Rating:   4,
				Feedback: ptrString("Easy pickup"),
			},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusDelivered,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("RateDeliveryByDriver", mock.Anything, deliveryID, 4, ptrString("Easy pickup")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "success - sender rates without feedback",
			userID: senderID,
			req: &RateDeliveryRequest{
				Rating:   3,
				Feedback: nil,
			},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusDelivered,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("RateDeliveryBySender", mock.Anything, deliveryID, 3, (*string)(nil)).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "error - delivery not found",
			userID: senderID,
			req:    &RateDeliveryRequest{Rating: 5},
			setupMocks: func(m *mockRepo) {
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(nil, pgx.ErrNoRows)
			},
			wantErr:    true,
			errContain: "no rows",
		},
		{
			name:   "error - delivery not completed",
			userID: senderID,
			req:    &RateDeliveryRequest{Rating: 5},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusInTransit,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "only rate completed deliveries",
		},
		{
			name:   "error - not part of delivery",
			userID: otherUserID,
			req:    &RateDeliveryRequest{Rating: 5},
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusDelivered,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.RateDelivery(context.Background(), deliveryID, tt.userID, tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetMyDeliveries
// ========================================

func TestGetMyDeliveries(t *testing.T) {
	senderID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name       string
		filters    *DeliveryListFilters
		limit      int
		offset     int
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, deliveries []*Delivery, total int64)
	}{
		{
			name:    "success - returns deliveries with pagination",
			filters: nil,
			limit:   10,
			offset:  0,
			setupMocks: func(m *mockRepo) {
				deliveries := []*Delivery{
					{ID: uuid.New(), SenderID: senderID, Status: DeliveryStatusDelivered},
					{ID: uuid.New(), SenderID: senderID, Status: DeliveryStatusRequested},
				}
				m.On("GetDeliveriesBySender", mock.Anything, senderID, (*DeliveryListFilters)(nil), 10, 0).Return(deliveries, int64(2), nil)
			},
			wantErr: false,
			validate: func(t *testing.T, deliveries []*Delivery, total int64) {
				assert.Len(t, deliveries, 2)
				assert.Equal(t, int64(2), total)
			},
		},
		{
			name: "success - filter by status",
			filters: &DeliveryListFilters{
				Status: func() *DeliveryStatus { s := DeliveryStatusDelivered; return &s }(),
			},
			limit:  20,
			offset: 0,
			setupMocks: func(m *mockRepo) {
				deliveries := []*Delivery{
					{ID: uuid.New(), SenderID: senderID, Status: DeliveryStatusDelivered},
				}
				m.On("GetDeliveriesBySender", mock.Anything, senderID, mock.AnythingOfType("*delivery.DeliveryListFilters"), 20, 0).Return(deliveries, int64(1), nil)
			},
			wantErr: false,
			validate: func(t *testing.T, deliveries []*Delivery, total int64) {
				assert.Len(t, deliveries, 1)
			},
		},
		{
			name:    "success - empty result",
			filters: nil,
			limit:   10,
			offset:  0,
			setupMocks: func(m *mockRepo) {
				m.On("GetDeliveriesBySender", mock.Anything, senderID, (*DeliveryListFilters)(nil), 10, 0).Return([]*Delivery{}, int64(0), nil)
			},
			wantErr: false,
			validate: func(t *testing.T, deliveries []*Delivery, total int64) {
				assert.Empty(t, deliveries)
				assert.Equal(t, int64(0), total)
			},
		},
		{
			name:    "error - repository error",
			filters: nil,
			limit:   10,
			offset:  0,
			setupMocks: func(m *mockRepo) {
				m.On("GetDeliveriesBySender", mock.Anything, senderID, (*DeliveryListFilters)(nil), 10, 0).Return(nil, int64(0), errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			deliveries, total, err := svc.GetMyDeliveries(context.Background(), senderID, tt.filters, tt.limit, tt.offset)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.validate(t, deliveries, total)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetAvailableDeliveries
// ========================================

func TestGetAvailableDeliveries(t *testing.T) {
	tests := []struct {
		name       string
		latitude        float64
		longitude        float64
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, deliveries []*Delivery)
	}{
		{
			name: "success - returns available deliveries",
			latitude:  40.7128,
			longitude:  -74.0060,
			setupMocks: func(m *mockRepo) {
				deliveries := []*Delivery{
					{ID: uuid.New(), Status: DeliveryStatusRequested, Priority: DeliveryPriorityExpress},
					{ID: uuid.New(), Status: DeliveryStatusRequested, Priority: DeliveryPriorityStandard},
				}
				m.On("GetAvailableDeliveries", mock.Anything, 40.7128, -74.0060, 15.0).Return(deliveries, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, deliveries []*Delivery) {
				assert.Len(t, deliveries, 2)
			},
		},
		{
			name: "success - no available deliveries",
			latitude:  40.7128,
			longitude:  -74.0060,
			setupMocks: func(m *mockRepo) {
				m.On("GetAvailableDeliveries", mock.Anything, 40.7128, -74.0060, 15.0).Return([]*Delivery{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, deliveries []*Delivery) {
				assert.Empty(t, deliveries)
			},
		},
		{
			name: "error - repository error",
			latitude:  40.7128,
			longitude:  -74.0060,
			setupMocks: func(m *mockRepo) {
				m.On("GetAvailableDeliveries", mock.Anything, 40.7128, -74.0060, 15.0).Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			deliveries, err := svc.GetAvailableDeliveries(context.Background(), tt.latitude, tt.longitude)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.validate(t, deliveries)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetActiveDelivery
// ========================================

func TestGetActiveDelivery(t *testing.T) {
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	deliveryID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
		validate   func(t *testing.T, resp *DeliveryResponse)
	}{
		{
			name: "success - returns active delivery",
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					DriverID: &driverID,
					Status:   DeliveryStatusInTransit,
					ProofPIN: ptrString("1234"),
				}
				m.On("GetActiveDeliveryForDriver", mock.Anything, driverID).Return(delivery, nil)
				m.On("GetStopsByDeliveryID", mock.Anything, deliveryID).Return([]DeliveryStop{}, nil)
				m.On("GetTrackingByDeliveryID", mock.Anything, deliveryID).Return([]DeliveryTracking{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *DeliveryResponse) {
				assert.Equal(t, deliveryID, resp.Delivery.ID)
				assert.Nil(t, resp.Delivery.ProofPIN) // PIN should be hidden
			},
		},
		{
			name: "error - no active delivery",
			setupMocks: func(m *mockRepo) {
				m.On("GetActiveDeliveryForDriver", mock.Anything, driverID).Return(nil, nil)
			},
			wantErr:    true,
			errContain: "no active delivery",
		},
		{
			name: "error - repository error",
			setupMocks: func(m *mockRepo) {
				m.On("GetActiveDeliveryForDriver", mock.Anything, driverID).Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			resp, err := svc.GetActiveDelivery(context.Background(), driverID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				tt.validate(t, resp)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetStats
// ========================================

func TestGetStats(t *testing.T) {
	senderID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, stats *DeliveryStats)
	}{
		{
			name: "success - returns stats",
			setupMocks: func(m *mockRepo) {
				stats := &DeliveryStats{
					TotalDeliveries:   50,
					CompletedCount:    45,
					CancelledCount:    3,
					InProgressCount:   2,
					AverageRating:     4.7,
					TotalSpent:        1250.00,
					AverageDeliveryMin: 35.5,
				}
				m.On("GetSenderStats", mock.Anything, senderID).Return(stats, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, stats *DeliveryStats) {
				assert.Equal(t, 50, stats.TotalDeliveries)
				assert.Equal(t, 45, stats.CompletedCount)
				assert.Equal(t, 3, stats.CancelledCount)
				assert.Equal(t, 2, stats.InProgressCount)
				assert.Equal(t, 4.7, stats.AverageRating)
				assert.Equal(t, 1250.00, stats.TotalSpent)
			},
		},
		{
			name: "success - no deliveries",
			setupMocks: func(m *mockRepo) {
				stats := &DeliveryStats{
					TotalDeliveries: 0,
					CompletedCount:  0,
					CancelledCount:  0,
					InProgressCount: 0,
					AverageRating:   0,
					TotalSpent:      0,
				}
				m.On("GetSenderStats", mock.Anything, senderID).Return(stats, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, stats *DeliveryStats) {
				assert.Equal(t, 0, stats.TotalDeliveries)
			},
		},
		{
			name: "error - repository error",
			setupMocks: func(m *mockRepo) {
				m.On("GetSenderStats", mock.Anything, senderID).Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			stats, err := svc.GetStats(context.Background(), senderID)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, stats)
			} else {
				require.NoError(t, err)
				require.NotNil(t, stats)
				tt.validate(t, stats)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: UpdateStopStatus
// ========================================

func TestUpdateStopStatus(t *testing.T) {
	senderID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	otherDriverID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	deliveryID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	stopID := uuid.MustParse("55555555-5555-5555-5555-555555555555")

	tests := []struct {
		name       string
		status     string
		photoURL   *string
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
	}{
		{
			name:     "success - mark stop as arrived",
			status:   "arrived",
			photoURL: nil,
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusInTransit,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("UpdateStopStatus", mock.Anything, stopID, "arrived", (*string)(nil)).Return(nil)
			},
			wantErr: false,
		},
		{
			name:     "success - mark stop as completed with photo",
			status:   "completed",
			photoURL: ptrString("https://example.com/photo.jpg"),
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &driverID,
					Status:   DeliveryStatusInTransit,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
				m.On("UpdateStopStatus", mock.Anything, stopID, "completed", ptrString("https://example.com/photo.jpg")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:     "error - delivery not found",
			status:   "arrived",
			photoURL: nil,
			setupMocks: func(m *mockRepo) {
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(nil, pgx.ErrNoRows)
			},
			wantErr:    true,
			errContain: "no rows",
		},
		{
			name:     "error - not your delivery",
			status:   "arrived",
			photoURL: nil,
			setupMocks: func(m *mockRepo) {
				delivery := &Delivery{
					ID:       deliveryID,
					SenderID: senderID,
					DriverID: &otherDriverID,
					Status:   DeliveryStatusInTransit,
				}
				m.On("GetDeliveryByID", mock.Anything, deliveryID).Return(delivery, nil)
			},
			wantErr:    true,
			errContain: "forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.UpdateStopStatus(context.Background(), deliveryID, stopID, driverID, tt.status, tt.photoURL)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: isValidTransition (Unit Tests for State Machine)
// ========================================

func TestIsValidTransition(t *testing.T) {
	tests := []struct {
		from     DeliveryStatus
		to       DeliveryStatus
		expected bool
	}{
		// From Requested
		{DeliveryStatusRequested, DeliveryStatusAccepted, true},
		{DeliveryStatusRequested, DeliveryStatusCancelled, true},
		{DeliveryStatusRequested, DeliveryStatusPickingUp, false},
		{DeliveryStatusRequested, DeliveryStatusInTransit, false},
		{DeliveryStatusRequested, DeliveryStatusDelivered, false},

		// From Accepted
		{DeliveryStatusAccepted, DeliveryStatusPickingUp, true},
		{DeliveryStatusAccepted, DeliveryStatusPickedUp, true},
		{DeliveryStatusAccepted, DeliveryStatusCancelled, true},
		{DeliveryStatusAccepted, DeliveryStatusRequested, false},
		{DeliveryStatusAccepted, DeliveryStatusDelivered, false},

		// From PickingUp
		{DeliveryStatusPickingUp, DeliveryStatusPickedUp, true},
		{DeliveryStatusPickingUp, DeliveryStatusCancelled, true},
		{DeliveryStatusPickingUp, DeliveryStatusDelivered, false},
		{DeliveryStatusPickingUp, DeliveryStatusAccepted, false},

		// From PickedUp
		{DeliveryStatusPickedUp, DeliveryStatusInTransit, true},
		{DeliveryStatusPickedUp, DeliveryStatusCancelled, false},
		{DeliveryStatusPickedUp, DeliveryStatusAccepted, false},
		{DeliveryStatusPickedUp, DeliveryStatusDelivered, false},

		// From InTransit
		{DeliveryStatusInTransit, DeliveryStatusArrived, true},
		{DeliveryStatusInTransit, DeliveryStatusDelivered, true},
		{DeliveryStatusInTransit, DeliveryStatusReturned, true},
		{DeliveryStatusInTransit, DeliveryStatusCancelled, false},
		{DeliveryStatusInTransit, DeliveryStatusPickedUp, false},

		// From Arrived
		{DeliveryStatusArrived, DeliveryStatusDelivered, true},
		{DeliveryStatusArrived, DeliveryStatusReturned, true},
		{DeliveryStatusArrived, DeliveryStatusFailed, true},
		{DeliveryStatusArrived, DeliveryStatusCancelled, false},
		{DeliveryStatusArrived, DeliveryStatusInTransit, false},

		// Terminal states cannot transition
		{DeliveryStatusDelivered, DeliveryStatusCancelled, false},
		{DeliveryStatusDelivered, DeliveryStatusRequested, false},
		{DeliveryStatusCancelled, DeliveryStatusRequested, false},
		{DeliveryStatusReturned, DeliveryStatusDelivered, false},
		{DeliveryStatusFailed, DeliveryStatusDelivered, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+" -> "+string(tt.to), func(t *testing.T) {
			result := isValidTransition(tt.from, tt.to)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ========================================
// TESTS: Package Size and Priority Validation
// ========================================

func TestIsValidPackageSize(t *testing.T) {
	tests := []struct {
		size     PackageSize
		expected bool
	}{
		{PackageSizeEnvelope, true},
		{PackageSizeSmall, true},
		{PackageSizeMedium, true},
		{PackageSizeLarge, true},
		{PackageSizeXLarge, true},
		{PackageSize("invalid"), false},
		{PackageSize(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.size), func(t *testing.T) {
			result := isValidPackageSize(tt.size)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidPriority(t *testing.T) {
	tests := []struct {
		priority DeliveryPriority
		expected bool
	}{
		{DeliveryPriorityStandard, true},
		{DeliveryPriorityExpress, true},
		{DeliveryPriorityScheduled, true},
		{DeliveryPriority("invalid"), false},
		{DeliveryPriority(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.priority), func(t *testing.T) {
			result := isValidPriority(tt.priority)
			assert.Equal(t, tt.expected, result)
		})
	}
}
