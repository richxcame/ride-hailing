package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/internal/geography"
	"github.com/richxcame/ride-hailing/internal/negotiation"
	"github.com/richxcame/ride-hailing/internal/pricing"
	"github.com/stretchr/testify/mock"
)

// MockNegotiationRepository is a mock implementation of negotiation.RepositoryInterface
type MockNegotiationRepository struct {
	mock.Mock
}

func (m *MockNegotiationRepository) CreateSession(ctx context.Context, session *negotiation.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockNegotiationRepository) GetSessionByID(ctx context.Context, id uuid.UUID) (*negotiation.Session, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*negotiation.Session), args.Error(1)
}

func (m *MockNegotiationRepository) GetActiveSessionByRider(ctx context.Context, riderID uuid.UUID) (*negotiation.Session, error) {
	args := m.Called(ctx, riderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*negotiation.Session), args.Error(1)
}

func (m *MockNegotiationRepository) UpdateSessionStatus(ctx context.Context, id uuid.UUID, status negotiation.SessionStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockNegotiationRepository) AcceptOffer(ctx context.Context, sessionID, offerID uuid.UUID) error {
	args := m.Called(ctx, sessionID, offerID)
	return args.Error(0)
}

func (m *MockNegotiationRepository) CreateOffer(ctx context.Context, offer *negotiation.Offer) error {
	args := m.Called(ctx, offer)
	return args.Error(0)
}

func (m *MockNegotiationRepository) GetOfferByID(ctx context.Context, id uuid.UUID) (*negotiation.Offer, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*negotiation.Offer), args.Error(1)
}

func (m *MockNegotiationRepository) GetOffersBySession(ctx context.Context, sessionID uuid.UUID) ([]*negotiation.Offer, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*negotiation.Offer), args.Error(1)
}

func (m *MockNegotiationRepository) GetSettings(ctx context.Context, countryID, regionID, cityID *uuid.UUID) (*negotiation.Settings, error) {
	args := m.Called(ctx, countryID, regionID, cityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*negotiation.Settings), args.Error(1)
}

func (m *MockNegotiationRepository) CountDriverActiveOffers(ctx context.Context, driverID uuid.UUID) (int, error) {
	args := m.Called(ctx, driverID)
	return args.Int(0), args.Error(1)
}

func (m *MockNegotiationRepository) ExpireSession(ctx context.Context, sessionID uuid.UUID) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockNegotiationRepository) GetExpiredSessions(ctx context.Context) ([]*negotiation.Session, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*negotiation.Session), args.Error(1)
}

// MockPricingService is a mock implementation of negotiation.PricingServiceInterface
type MockPricingService struct {
	mock.Mock
}

func (m *MockPricingService) GetEstimate(ctx context.Context, req pricing.EstimateRequest) (*pricing.EstimateResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pricing.EstimateResponse), args.Error(1)
}

// MockGeographyService is a mock implementation of negotiation.GeographyServiceInterface
type MockGeographyService struct {
	mock.Mock
}

func (m *MockGeographyService) ResolveLocation(ctx context.Context, lat, lng float64) (*geography.ResolvedLocation, error) {
	args := m.Called(ctx, lat, lng)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*geography.ResolvedLocation), args.Error(1)
}
