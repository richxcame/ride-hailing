package negotiation

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/internal/geography"
	"github.com/richxcame/ride-hailing/internal/pricing"
	"github.com/richxcame/ride-hailing/pkg/eventbus"
)

// Service handles negotiation business logic
type Service struct {
	repo       RepositoryInterface
	pricingSvc PricingServiceInterface
	geoSvc     GeographyServiceInterface
	eventBus   *eventbus.Bus
}

// NewService creates a new negotiation service
func NewService(repo RepositoryInterface, pricingSvc PricingServiceInterface, geoSvc GeographyServiceInterface) *Service {
	return &Service{
		repo:       repo,
		pricingSvc: pricingSvc,
		geoSvc:     geoSvc,
	}
}

// SetEventBus sets the event bus for publishing negotiation events
func (s *Service) SetEventBus(bus *eventbus.Bus) {
	s.eventBus = bus
}

// StartSession starts a new negotiation session
func (s *Service) StartSession(ctx context.Context, riderID uuid.UUID, req *StartSessionRequest) (*Session, error) {
	// Check for existing active session
	existing, err := s.repo.GetActiveSessionByRider(ctx, riderID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("rider already has an active negotiation session")
	}

	// Resolve pickup location
	pickupResolved, err := s.geoSvc.ResolveLocation(ctx, req.PickupLatitude, req.PickupLongitude)
	if err != nil {
		pickupResolved = &geography.ResolvedLocation{}
	}

	// Resolve dropoff location
	dropoffResolved, err := s.geoSvc.ResolveLocation(ctx, req.DropoffLatitude, req.DropoffLongitude)
	if err != nil {
		dropoffResolved = &geography.ResolvedLocation{}
	}

	// Get pricing estimate
	estimate, err := s.pricingSvc.GetEstimate(ctx, pricing.EstimateRequest{
		PickupLatitude:   req.PickupLatitude,
		PickupLongitude:  req.PickupLongitude,
		DropoffLatitude:  req.DropoffLatitude,
		DropoffLongitude: req.DropoffLongitude,
		RideTypeID:       req.RideTypeID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get price estimate: %w", err)
	}

	// Get negotiation settings
	var countryID, regionID, cityID, pickupZoneID, dropoffZoneID *uuid.UUID
	if pickupResolved.Country != nil {
		countryID = &pickupResolved.Country.ID
	}
	if pickupResolved.Region != nil {
		regionID = &pickupResolved.Region.ID
	}
	if pickupResolved.City != nil {
		cityID = &pickupResolved.City.ID
	}
	if pickupResolved.PricingZone != nil {
		pickupZoneID = &pickupResolved.PricingZone.ID
	}
	if dropoffResolved.PricingZone != nil {
		dropoffZoneID = &dropoffResolved.PricingZone.ID
	}

	settings, err := s.repo.GetSettings(ctx, countryID, regionID, cityID)
	if err != nil {
		settings = &DefaultSettings
	}

	if !settings.NegotiationEnabled {
		return nil, fmt.Errorf("negotiation is not enabled in this region")
	}

	// Calculate fair price bounds
	fairPrice := s.calculateFairPrice(estimate.EstimatedFare, settings)

	// Validate rider's initial offer if provided
	if req.InitialOffer != nil {
		if *req.InitialOffer < fairPrice.MinPrice {
			return nil, fmt.Errorf("initial offer %.2f is below minimum %.2f", *req.InitialOffer, fairPrice.MinPrice)
		}
		if *req.InitialOffer > fairPrice.MaxPrice {
			return nil, fmt.Errorf("initial offer %.2f exceeds maximum %.2f", *req.InitialOffer, fairPrice.MaxPrice)
		}
	}

	// Create session
	session := &Session{
		RiderID:              riderID,
		PickupLatitude:       req.PickupLatitude,
		PickupLongitude:      req.PickupLongitude,
		PickupAddress:        req.PickupAddress,
		DropoffLatitude:      req.DropoffLatitude,
		DropoffLongitude:     req.DropoffLongitude,
		DropoffAddress:       req.DropoffAddress,
		CountryID:            countryID,
		RegionID:             regionID,
		CityID:               cityID,
		PickupZoneID:         pickupZoneID,
		DropoffZoneID:        dropoffZoneID,
		RideTypeID:           req.RideTypeID,
		CurrencyCode:         estimate.Currency,
		EstimatedDistance:    estimate.DistanceKm,
		EstimatedDuration:    estimate.EstimatedMinutes,
		EstimatedFare:        estimate.EstimatedFare,
		FairPriceMin:         fairPrice.MinPrice,
		FairPriceMax:         fairPrice.MaxPrice,
		SystemSuggestedPrice: fairPrice.SuggestedPrice,
		RiderInitialOffer:    req.InitialOffer,
		Status:               SessionStatusActive,
		ExpiresAt:            time.Now().Add(time.Duration(settings.SessionTimeoutSeconds) * time.Second),
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	// Publish event for driver notifications
	s.publishEvent("negotiation.started", session)

	return session, nil
}

// GetSession retrieves a session by ID
func (s *Service) GetSession(ctx context.Context, sessionID uuid.UUID) (*Session, error) {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Load offers
	offers, err := s.repo.GetOffersBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	session.Offers = offers

	return session, nil
}

// GetActiveSessionByRider retrieves a rider's active session
func (s *Service) GetActiveSessionByRider(ctx context.Context, riderID uuid.UUID) (*Session, error) {
	return s.repo.GetActiveSessionByRider(ctx, riderID)
}

// SubmitOffer allows a driver to submit an offer
func (s *Service) SubmitOffer(ctx context.Context, sessionID, driverID uuid.UUID, req *SubmitOfferRequest, driverInfo *DriverInfo) (*Offer, error) {
	// Get session
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Validate session is active
	if session.Status != SessionStatusActive {
		return nil, fmt.Errorf("session is no longer active")
	}

	// Check expiry
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session has expired")
	}

	// Get settings
	settings, err := s.repo.GetSettings(ctx, session.CountryID, session.RegionID, session.CityID)
	if err != nil {
		settings = &DefaultSettings
	}

	// Check driver eligibility
	if err := s.checkDriverEligibility(ctx, driverID, driverInfo, settings); err != nil {
		return nil, err
	}

	// Check driver's active offer count
	activeCount, err := s.repo.CountDriverActiveOffers(ctx, driverID)
	if err != nil {
		return nil, err
	}
	if activeCount >= settings.MaxActiveSessionsPerDriver {
		return nil, fmt.Errorf("driver has too many active negotiations")
	}

	// Validate offered price
	if req.OfferedPrice < session.FairPriceMin {
		return nil, fmt.Errorf("offered price %.2f is below minimum %.2f", req.OfferedPrice, session.FairPriceMin)
	}
	if req.OfferedPrice > session.FairPriceMax {
		return nil, fmt.Errorf("offered price %.2f exceeds maximum %.2f", req.OfferedPrice, session.FairPriceMax)
	}

	// Create offer
	offer := &Offer{
		SessionID:           sessionID,
		DriverID:            driverID,
		OfferedPrice:        req.OfferedPrice,
		CurrencyCode:        session.CurrencyCode,
		DriverLatitude:      req.DriverLatitude,
		DriverLongitude:     req.DriverLongitude,
		EstimatedPickupTime: req.EstimatedPickupTime,
		DriverRating:        driverInfo.Rating,
		DriverTotalRides:    driverInfo.TotalRides,
		VehicleModel:        driverInfo.VehicleModel,
		VehicleColor:        driverInfo.VehicleColor,
		Status:              OfferStatusPending,
		IsCounterOffer:      false,
	}

	if err := s.repo.CreateOffer(ctx, offer); err != nil {
		return nil, err
	}

	// Publish event
	s.publishEvent("negotiation.offer.new", map[string]interface{}{
		"session_id": sessionID,
		"offer":      offer,
	})

	return offer, nil
}

// AcceptOffer allows a rider to accept an offer
func (s *Service) AcceptOffer(ctx context.Context, sessionID, offerID, riderID uuid.UUID) error {
	// Get session
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Verify ownership
	if session.RiderID != riderID {
		return fmt.Errorf("unauthorized: session belongs to another rider")
	}

	// Validate session is active
	if session.Status != SessionStatusActive {
		return fmt.Errorf("session is no longer active")
	}

	// Get offer
	offer, err := s.repo.GetOfferByID(ctx, offerID)
	if err != nil {
		return fmt.Errorf("offer not found: %w", err)
	}

	// Validate offer belongs to session
	if offer.SessionID != sessionID {
		return fmt.Errorf("offer does not belong to this session")
	}

	// Validate offer is pending
	if offer.Status != OfferStatusPending {
		return fmt.Errorf("offer is no longer pending")
	}

	// Accept the offer atomically
	if err := s.repo.AcceptOffer(ctx, sessionID, offerID); err != nil {
		return err
	}

	// Publish event
	s.publishEvent("negotiation.offer.accepted", map[string]interface{}{
		"session_id": sessionID,
		"offer_id":   offerID,
		"driver_id":  offer.DriverID,
		"price":      offer.OfferedPrice,
	})

	return nil
}

// CancelSession cancels a negotiation session
func (s *Service) CancelSession(ctx context.Context, sessionID, riderID uuid.UUID, reason string) error {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	if session.RiderID != riderID {
		return fmt.Errorf("unauthorized: session belongs to another rider")
	}

	if session.Status != SessionStatusActive {
		return fmt.Errorf("session is no longer active")
	}

	if err := s.repo.UpdateSessionStatus(ctx, sessionID, SessionStatusCancelled); err != nil {
		return err
	}

	s.publishEvent("negotiation.cancelled", map[string]interface{}{
		"session_id": sessionID,
		"reason":     reason,
	})

	return nil
}

// DriverInfo contains driver information for offers
type DriverInfo struct {
	Rating       *float64
	TotalRides   *int
	VehicleModel *string
	VehicleColor *string
}

// calculateFairPrice calculates the fair price bounds
func (s *Service) calculateFairPrice(estimatedFare float64, settings *Settings) *FairPriceCalculation {
	return &FairPriceCalculation{
		MinPrice:       estimatedFare * settings.MinPriceMultiplier,
		MaxPrice:       estimatedFare * settings.MaxPriceMultiplier,
		SuggestedPrice: estimatedFare,
		EstimatedFare:  estimatedFare,
	}
}

// checkDriverEligibility checks if a driver is eligible to participate
func (s *Service) checkDriverEligibility(ctx context.Context, driverID uuid.UUID, info *DriverInfo, settings *Settings) error {
	if settings.MinDriverRatingToNegotiate != nil && info.Rating != nil {
		if *info.Rating < *settings.MinDriverRatingToNegotiate {
			return fmt.Errorf("driver rating %.2f is below minimum %.2f", *info.Rating, *settings.MinDriverRatingToNegotiate)
		}
	}

	if settings.MinDriverRidesToNegotiate != nil && info.TotalRides != nil {
		if *info.TotalRides < *settings.MinDriverRidesToNegotiate {
			return fmt.Errorf("driver has %d rides, minimum is %d", *info.TotalRides, *settings.MinDriverRidesToNegotiate)
		}
	}

	return nil
}

// publishEvent publishes a negotiation event
func (s *Service) publishEvent(eventType string, data interface{}) {
	if s.eventBus == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		evt, err := eventbus.NewEvent(eventType, "negotiation-service", data)
		if err != nil {
			return
		}

		subject := fmt.Sprintf("negotiation.%s", eventType)
		s.eventBus.Publish(ctx, subject, evt)
	}()
}

// ExpireStale marks expired sessions
func (s *Service) ExpireStale(ctx context.Context) (int, error) {
	sessions, err := s.repo.GetExpiredSessions(ctx)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, session := range sessions {
		if err := s.repo.ExpireSession(ctx, session.ID); err != nil {
			continue
		}
		count++

		s.publishEvent("negotiation.expired", map[string]interface{}{
			"session_id": session.ID,
			"rider_id":   session.RiderID,
		})
	}

	return count, nil
}

// ToSessionResponse converts Session to API response
func ToSessionResponse(s *Session) *SessionResponse {
	if s == nil {
		return nil
	}

	resp := &SessionResponse{
		ID:                   s.ID,
		Status:               s.Status,
		PickupAddress:        s.PickupAddress,
		DropoffAddress:       s.DropoffAddress,
		CurrencyCode:         s.CurrencyCode,
		EstimatedFare:        s.EstimatedFare,
		FairPriceMin:         s.FairPriceMin,
		FairPriceMax:         s.FairPriceMax,
		SystemSuggestedPrice: s.SystemSuggestedPrice,
		RiderInitialOffer:    s.RiderInitialOffer,
		AcceptedPrice:        s.AcceptedPrice,
		ExpiresAt:            s.ExpiresAt,
		OffersCount:          len(s.Offers),
	}

	if len(s.Offers) > 0 {
		resp.Offers = make([]*OfferResponse, len(s.Offers))
		for i, offer := range s.Offers {
			resp.Offers[i] = ToOfferResponse(offer)
		}
	}

	return resp
}

// ToOfferResponse converts Offer to API response
func ToOfferResponse(o *Offer) *OfferResponse {
	if o == nil {
		return nil
	}

	return &OfferResponse{
		ID:                  o.ID,
		DriverID:            o.DriverID,
		OfferedPrice:        o.OfferedPrice,
		Status:              o.Status,
		EstimatedPickupTime: o.EstimatedPickupTime,
		DriverRating:        o.DriverRating,
		VehicleModel:        o.VehicleModel,
		VehicleColor:        o.VehicleColor,
		IsCounterOffer:      o.IsCounterOffer,
		CreatedAt:           o.CreatedAt,
	}
}
