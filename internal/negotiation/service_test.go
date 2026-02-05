package negotiation

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/internal/geography"
	"github.com/richxcame/ride-hailing/internal/pricing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Internal mocks (same package, so they can satisfy unexported-friendly interfaces) ---

type mockRepo struct{ mock.Mock }

func (m *mockRepo) CreateSession(ctx context.Context, session *Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}
func (m *mockRepo) GetSessionByID(ctx context.Context, id uuid.UUID) (*Session, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Session), args.Error(1)
}
func (m *mockRepo) GetActiveSessionByRider(ctx context.Context, riderID uuid.UUID) (*Session, error) {
	args := m.Called(ctx, riderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Session), args.Error(1)
}
func (m *mockRepo) UpdateSessionStatus(ctx context.Context, id uuid.UUID, status SessionStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}
func (m *mockRepo) AcceptOffer(ctx context.Context, sessionID, offerID uuid.UUID) error {
	args := m.Called(ctx, sessionID, offerID)
	return args.Error(0)
}
func (m *mockRepo) CreateOffer(ctx context.Context, offer *Offer) error {
	args := m.Called(ctx, offer)
	return args.Error(0)
}
func (m *mockRepo) GetOfferByID(ctx context.Context, id uuid.UUID) (*Offer, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Offer), args.Error(1)
}
func (m *mockRepo) GetOffersBySession(ctx context.Context, sessionID uuid.UUID) ([]*Offer, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Offer), args.Error(1)
}
func (m *mockRepo) GetSettings(ctx context.Context, countryID, regionID, cityID *uuid.UUID) (*Settings, error) {
	args := m.Called(ctx, countryID, regionID, cityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Settings), args.Error(1)
}
func (m *mockRepo) CountDriverActiveOffers(ctx context.Context, driverID uuid.UUID) (int, error) {
	args := m.Called(ctx, driverID)
	return args.Int(0), args.Error(1)
}
func (m *mockRepo) ExpireSession(ctx context.Context, sessionID uuid.UUID) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}
func (m *mockRepo) GetExpiredSessions(ctx context.Context) ([]*Session, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Session), args.Error(1)
}

type mockPricing struct{ mock.Mock }

func (m *mockPricing) GetEstimate(ctx context.Context, req pricing.EstimateRequest) (*pricing.EstimateResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pricing.EstimateResponse), args.Error(1)
}

type mockGeo struct{ mock.Mock }

func (m *mockGeo) ResolveLocation(ctx context.Context, lat, lng float64) (*geography.ResolvedLocation, error) {
	args := m.Called(ctx, lat, lng)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*geography.ResolvedLocation), args.Error(1)
}

// --- helpers ---

func defaultSettings() *Settings {
	return &Settings{
		NegotiationEnabled:         true,
		SessionTimeoutSeconds:      300,
		MaxOffersPerSession:        20,
		MaxCounterOffers:           3,
		OfferTimeoutSeconds:        60,
		MinPriceMultiplier:         0.70,
		MaxPriceMultiplier:         1.50,
		MaxActiveSessionsPerDriver: 5,
	}
}

func defaultEstimate() *pricing.EstimateResponse {
	return &pricing.EstimateResponse{
		Currency:         "USD",
		EstimatedFare:    100.0,
		MinimumFare:      5.0,
		SurgeMultiplier:  1.0,
		DistanceKm:       10.0,
		EstimatedMinutes: 15,
	}
}

func defaultResolvedLocation() *geography.ResolvedLocation {
	return &geography.ResolvedLocation{}
}

func newService(repo *mockRepo, pSvc *mockPricing, gSvc *mockGeo) *Service {
	return NewService(repo, pSvc, gSvc)
}

// ================================
// StartSession Tests
// ================================

func TestStartSession(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(repo *mockRepo, pSvc *mockPricing, gSvc *mockGeo)
		riderID   uuid.UUID
		req       *StartSessionRequest
		wantErr   bool
		errSubstr string
	}{
		{
			name: "success with valid price bounds",
			setup: func(repo *mockRepo, pSvc *mockPricing, gSvc *mockGeo) {
				riderID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

				// No active session
				repo.On("GetActiveSessionByRider", mock.Anything, riderID).
					Return(nil, nil)

				// Geo resolves
				gSvc.On("ResolveLocation", mock.Anything, 40.0, 29.0).
					Return(defaultResolvedLocation(), nil)
				gSvc.On("ResolveLocation", mock.Anything, 41.0, 30.0).
					Return(defaultResolvedLocation(), nil)

				// Pricing estimate
				pSvc.On("GetEstimate", mock.Anything, mock.AnythingOfType("EstimateRequest")).
					Return(defaultEstimate(), nil)

				// Settings (fall back to defaults on error)
				repo.On("GetSettings", mock.Anything, (*uuid.UUID)(nil), (*uuid.UUID)(nil), (*uuid.UUID)(nil)).
					Return(defaultSettings(), nil)

				// Create session
				repo.On("CreateSession", mock.Anything, mock.AnythingOfType("*negotiation.Session")).
					Return(nil)
			},
			riderID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			req: &StartSessionRequest{
				PickupLatitude:   40.0,
				PickupLongitude:  29.0,
				DropoffLatitude:  41.0,
				DropoffLongitude: 30.0,
				PickupAddress:    "123 Main St",
				DropoffAddress:   "456 Oak Ave",
			},
			wantErr: false,
		},
		{
			name: "rider already has active session",
			setup: func(repo *mockRepo, pSvc *mockPricing, gSvc *mockGeo) {
				riderID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
				repo.On("GetActiveSessionByRider", mock.Anything, riderID).
					Return(&Session{
						ID:      uuid.New(),
						RiderID: riderID,
						Status:  SessionStatusActive,
					}, nil)
			},
			riderID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			req: &StartSessionRequest{
				PickupLatitude:   40.0,
				PickupLongitude:  29.0,
				DropoffLatitude:  41.0,
				DropoffLongitude: 30.0,
				PickupAddress:    "123 Main St",
				DropoffAddress:   "456 Oak Ave",
			},
			wantErr:   true,
			errSubstr: "rider already has an active negotiation session",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mockRepo)
			pSvc := new(mockPricing)
			gSvc := new(mockGeo)
			tc.setup(repo, pSvc, gSvc)

			svc := newService(repo, pSvc, gSvc)
			session, err := svc.StartSession(context.Background(), tc.riderID, tc.req)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
				assert.Nil(t, session)
			} else {
				require.NoError(t, err)
				require.NotNil(t, session)
				assert.Equal(t, SessionStatusActive, session.Status)
				assert.Equal(t, tc.riderID, session.RiderID)
				// Validate fair price bounds: 100 * 0.7 = 70, 100 * 1.5 = 150
				assert.Equal(t, 70.0, session.FairPriceMin)
				assert.Equal(t, 150.0, session.FairPriceMax)
				assert.Equal(t, 100.0, session.SystemSuggestedPrice)
			}

			repo.AssertExpectations(t)
			pSvc.AssertExpectations(t)
			gSvc.AssertExpectations(t)
		})
	}
}

// ================================
// GetSession Tests
// ================================

func TestGetSession(t *testing.T) {
	tests := []struct {
		name      string
		sessionID uuid.UUID
		setup     func(repo *mockRepo)
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "success",
			sessionID: uuid.MustParse("00000000-0000-0000-0000-000000000010"),
			setup: func(repo *mockRepo) {
				sid := uuid.MustParse("00000000-0000-0000-0000-000000000010")
				repo.On("GetSessionByID", mock.Anything, sid).
					Return(&Session{
						ID:     sid,
						Status: SessionStatusActive,
					}, nil)
				repo.On("GetOffersBySession", mock.Anything, sid).
					Return([]*Offer{
						{ID: uuid.New(), SessionID: sid, OfferedPrice: 80.0},
					}, nil)
			},
			wantErr: false,
		},
		{
			name:      "not found",
			sessionID: uuid.MustParse("00000000-0000-0000-0000-000000000099"),
			setup: func(repo *mockRepo) {
				sid := uuid.MustParse("00000000-0000-0000-0000-000000000099")
				repo.On("GetSessionByID", mock.Anything, sid).
					Return(nil, fmt.Errorf("failed to get session: no rows"))
			},
			wantErr:   true,
			errSubstr: "failed to get session",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mockRepo)
			pSvc := new(mockPricing)
			gSvc := new(mockGeo)
			tc.setup(repo)

			svc := newService(repo, pSvc, gSvc)
			session, err := svc.GetSession(context.Background(), tc.sessionID)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
				assert.Nil(t, session)
			} else {
				require.NoError(t, err)
				require.NotNil(t, session)
				assert.Equal(t, tc.sessionID, session.ID)
				assert.Len(t, session.Offers, 1)
			}

			repo.AssertExpectations(t)
		})
	}
}

// ================================
// SubmitOffer Tests
// ================================

func TestSubmitOffer(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000020")
	driverID := uuid.MustParse("00000000-0000-0000-0000-000000000021")
	riderID := uuid.MustParse("00000000-0000-0000-0000-000000000022")

	activeSession := &Session{
		ID:            sessionID,
		RiderID:       riderID,
		Status:        SessionStatusActive,
		FairPriceMin:  70.0,
		FairPriceMax:  150.0,
		CurrencyCode:  "USD",
		ExpiresAt:     time.Now().Add(5 * time.Minute),
	}

	rating := 4.5
	rides := 100
	model := "Toyota Camry"
	color := "White"

	driverInfo := &DriverInfo{
		Rating:       &rating,
		TotalRides:   &rides,
		VehicleModel: &model,
		VehicleColor: &color,
	}

	tests := []struct {
		name      string
		setup     func(repo *mockRepo)
		req       *SubmitOfferRequest
		wantErr   bool
		errSubstr string
	}{
		{
			name: "valid offer within bounds",
			setup: func(repo *mockRepo) {
				repo.On("GetSessionByID", mock.Anything, sessionID).
					Return(activeSession, nil)
				repo.On("GetSettings", mock.Anything, (*uuid.UUID)(nil), (*uuid.UUID)(nil), (*uuid.UUID)(nil)).
					Return(defaultSettings(), nil)
				repo.On("CountDriverActiveOffers", mock.Anything, driverID).
					Return(0, nil)
				repo.On("CreateOffer", mock.Anything, mock.AnythingOfType("*negotiation.Offer")).
					Return(nil)
			},
			req:     &SubmitOfferRequest{OfferedPrice: 100.0},
			wantErr: false,
		},
		{
			name: "offer below minimum",
			setup: func(repo *mockRepo) {
				repo.On("GetSessionByID", mock.Anything, sessionID).
					Return(activeSession, nil)
				repo.On("GetSettings", mock.Anything, (*uuid.UUID)(nil), (*uuid.UUID)(nil), (*uuid.UUID)(nil)).
					Return(defaultSettings(), nil)
				repo.On("CountDriverActiveOffers", mock.Anything, driverID).
					Return(0, nil)
			},
			req:       &SubmitOfferRequest{OfferedPrice: 50.0},
			wantErr:   true,
			errSubstr: "below minimum",
		},
		{
			name: "offer above maximum",
			setup: func(repo *mockRepo) {
				repo.On("GetSessionByID", mock.Anything, sessionID).
					Return(activeSession, nil)
				repo.On("GetSettings", mock.Anything, (*uuid.UUID)(nil), (*uuid.UUID)(nil), (*uuid.UUID)(nil)).
					Return(defaultSettings(), nil)
				repo.On("CountDriverActiveOffers", mock.Anything, driverID).
					Return(0, nil)
			},
			req:       &SubmitOfferRequest{OfferedPrice: 200.0},
			wantErr:   true,
			errSubstr: "exceeds maximum",
		},
		{
			name: "driver has too many active offers",
			setup: func(repo *mockRepo) {
				repo.On("GetSessionByID", mock.Anything, sessionID).
					Return(activeSession, nil)
				repo.On("GetSettings", mock.Anything, (*uuid.UUID)(nil), (*uuid.UUID)(nil), (*uuid.UUID)(nil)).
					Return(defaultSettings(), nil)
				repo.On("CountDriverActiveOffers", mock.Anything, driverID).
					Return(5, nil) // max is 5
			},
			req:       &SubmitOfferRequest{OfferedPrice: 100.0},
			wantErr:   true,
			errSubstr: "too many active negotiations",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mockRepo)
			pSvc := new(mockPricing)
			gSvc := new(mockGeo)
			tc.setup(repo)

			svc := newService(repo, pSvc, gSvc)
			offer, err := svc.SubmitOffer(context.Background(), sessionID, driverID, tc.req, driverInfo)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
				assert.Nil(t, offer)
			} else {
				require.NoError(t, err)
				require.NotNil(t, offer)
				assert.Equal(t, tc.req.OfferedPrice, offer.OfferedPrice)
				assert.Equal(t, sessionID, offer.SessionID)
				assert.Equal(t, driverID, offer.DriverID)
				assert.Equal(t, OfferStatusPending, offer.Status)
			}

			repo.AssertExpectations(t)
		})
	}
}

// ================================
// AcceptOffer Tests
// ================================

func TestAcceptOffer(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000030")
	offerID := uuid.MustParse("00000000-0000-0000-0000-000000000031")
	riderID := uuid.MustParse("00000000-0000-0000-0000-000000000032")
	driverID := uuid.MustParse("00000000-0000-0000-0000-000000000033")

	tests := []struct {
		name      string
		setup     func(repo *mockRepo)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "success",
			setup: func(repo *mockRepo) {
				repo.On("GetSessionByID", mock.Anything, sessionID).
					Return(&Session{
						ID:      sessionID,
						RiderID: riderID,
						Status:  SessionStatusActive,
					}, nil)
				repo.On("GetOfferByID", mock.Anything, offerID).
					Return(&Offer{
						ID:        offerID,
						SessionID: sessionID,
						DriverID:  driverID,
						Status:    OfferStatusPending,
						OfferedPrice: 90.0,
					}, nil)
				repo.On("AcceptOffer", mock.Anything, sessionID, offerID).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "session not found",
			setup: func(repo *mockRepo) {
				repo.On("GetSessionByID", mock.Anything, sessionID).
					Return(nil, fmt.Errorf("failed to get session: no rows"))
			},
			wantErr:   true,
			errSubstr: "session not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mockRepo)
			pSvc := new(mockPricing)
			gSvc := new(mockGeo)
			tc.setup(repo)

			svc := newService(repo, pSvc, gSvc)
			err := svc.AcceptOffer(context.Background(), sessionID, offerID, riderID)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
			} else {
				require.NoError(t, err)
			}

			repo.AssertExpectations(t)
		})
	}
}

// ================================
// CancelSession Tests
// ================================

func TestCancelSession(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000040")
	riderID := uuid.MustParse("00000000-0000-0000-0000-000000000041")

	tests := []struct {
		name      string
		setup     func(repo *mockRepo)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "success",
			setup: func(repo *mockRepo) {
				repo.On("GetSessionByID", mock.Anything, sessionID).
					Return(&Session{
						ID:      sessionID,
						RiderID: riderID,
						Status:  SessionStatusActive,
					}, nil)
				repo.On("UpdateSessionStatus", mock.Anything, sessionID, SessionStatusCancelled).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "not found",
			setup: func(repo *mockRepo) {
				repo.On("GetSessionByID", mock.Anything, sessionID).
					Return(nil, fmt.Errorf("failed to get session: no rows"))
			},
			wantErr:   true,
			errSubstr: "session not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mockRepo)
			pSvc := new(mockPricing)
			gSvc := new(mockGeo)
			tc.setup(repo)

			svc := newService(repo, pSvc, gSvc)
			err := svc.CancelSession(context.Background(), sessionID, riderID, "changed my mind")

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
			} else {
				require.NoError(t, err)
			}

			repo.AssertExpectations(t)
		})
	}
}

// ================================
// ExpireStale Tests
// ================================

func TestExpireStale(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(repo *mockRepo)
		wantCount int
		wantErr   bool
	}{
		{
			name: "expires old sessions",
			setup: func(repo *mockRepo) {
				s1 := &Session{
					ID:      uuid.MustParse("00000000-0000-0000-0000-000000000050"),
					RiderID: uuid.MustParse("00000000-0000-0000-0000-000000000051"),
					Status:  SessionStatusActive,
				}
				s2 := &Session{
					ID:      uuid.MustParse("00000000-0000-0000-0000-000000000052"),
					RiderID: uuid.MustParse("00000000-0000-0000-0000-000000000053"),
					Status:  SessionStatusActive,
				}
				repo.On("GetExpiredSessions", mock.Anything).
					Return([]*Session{s1, s2}, nil)
				repo.On("ExpireSession", mock.Anything, s1.ID).Return(nil)
				repo.On("ExpireSession", mock.Anything, s2.ID).Return(nil)
			},
			wantCount: 2,
			wantErr:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mockRepo)
			pSvc := new(mockPricing)
			gSvc := new(mockGeo)
			tc.setup(repo)

			svc := newService(repo, pSvc, gSvc)
			count, err := svc.ExpireStale(context.Background())

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantCount, count)
			}

			repo.AssertExpectations(t)
		})
	}
}
