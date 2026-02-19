package rides

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/internal/pricing"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/eventbus"
	pkggeo "github.com/richxcame/ride-hailing/pkg/geo"
	"github.com/richxcame/ride-hailing/pkg/httpclient"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/richxcame/ride-hailing/pkg/resilience"
	"github.com/richxcame/ride-hailing/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// PricingConfig holds pricing configuration for rides
type PricingConfig struct {
	BaseFarePerKm     float64 // Base fare per kilometer
	BaseFarePerMinute float64 // Base fare per minute
	MinimumFare       float64 // Minimum fare
	CommissionRate    float64 // Commission rate (e.g., 0.20 for 20%)
}

// DefaultPricingConfig returns default pricing configuration
func DefaultPricingConfig() *PricingConfig {
	return &PricingConfig{
		BaseFarePerKm:     1.5,
		BaseFarePerMinute: 0.25,
		MinimumFare:       5.0,
		CommissionRate:    0.20,
	}
}

// Service handles ride business logic
type Service struct {
	repo                *Repository
	promosClient        *httpclient.Client
	promosBreaker       *resilience.CircuitBreaker
	surgeCalculator     SurgeCalculator
	mlEtaClient         *httpclient.Client
	mlEtaBreaker        *resilience.CircuitBreaker
	matcher             *Matcher
	eventBus            *eventbus.Bus
	pricingConfig       *PricingConfig
	pricingService      *pricing.Service
	locationResolver    LocationResolver
	rideTypeNameFetcher func(ctx context.Context, id uuid.UUID) (string, error)
}

// SurgeCalculator defines the interface for surge pricing calculation
type SurgeCalculator interface {
	CalculateSurgeMultiplier(ctx context.Context, latitude, longitude float64) (float64, error)
	GetCurrentSurgeInfo(ctx context.Context, latitude, longitude float64) (map[string]interface{}, error)
}

// LocationContext carries the resolved geographic hierarchy for a coordinate.
// Only the IDs are kept here; the rides service does not need the full objects.
type LocationContext struct {
	CountryID     *uuid.UUID
	RegionID      *uuid.UUID
	CityID        *uuid.UUID
	PricingZoneID *uuid.UUID
	Timezone      string // IANA tz name, e.g. "Asia/Ashgabat"
}

// LocationResolver resolves a latitude/longitude to the geographic hierarchy.
// Implemented by geography.Service — but defined here to avoid import cycles.
type LocationResolver interface {
	ResolveLocationContext(ctx context.Context, latitude, longitude float64) (*LocationContext, error)
}

// LocationResolverFunc is a function adapter that implements LocationResolver.
// Use it in main.go to wire geography.Service without importing the geography package
// inside the rides package (which would create an import cycle).
type LocationResolverFunc func(ctx context.Context, latitude, longitude float64) (*LocationContext, error)

func (f LocationResolverFunc) ResolveLocationContext(ctx context.Context, latitude, longitude float64) (*LocationContext, error) {
	return f(ctx, latitude, longitude)
}

// NewService creates a new rides service
func NewService(repo *Repository, promosServiceURL string, breaker *resilience.CircuitBreaker, httpClientTimeout ...time.Duration) *Service {
	if repo == nil {
		panic("rides: repository cannot be nil")
	}
	return &Service{
		repo:            repo,
		promosClient:    httpclient.NewClient(promosServiceURL, httpClientTimeout...),
		promosBreaker:   breaker,
		surgeCalculator: nil, // Will be set via SetSurgeCalculator
		pricingConfig:   DefaultPricingConfig(),
	}
}

// SetPricingConfig sets custom pricing configuration
func (s *Service) SetPricingConfig(config *PricingConfig) {
	if config != nil {
		s.pricingConfig = config
	}
}

// SetSurgeCalculator sets the surge pricing calculator
func (s *Service) SetSurgeCalculator(calculator SurgeCalculator) {
	s.surgeCalculator = calculator
}

// SetMatcher sets the driver-rider matching engine.
func (s *Service) SetMatcher(matcher *Matcher) {
	s.matcher = matcher
}

// SetEventBus sets the NATS event bus for publishing ride lifecycle events.
func (s *Service) SetEventBus(bus *eventbus.Bus) {
	s.eventBus = bus
}

// SetPricingService sets the hierarchical pricing service for fare calculation.
func (s *Service) SetPricingService(svc *pricing.Service) {
	s.pricingService = svc
}

// SetLocationResolver wires the geography service so ride records are stamped
// with the correct country/region/city/zone IDs at creation time.
func (s *Service) SetLocationResolver(r LocationResolver) {
	s.locationResolver = r
}

// SetRideTypeNameFetcher provides a function to resolve a ride type name by ID.
// Call this during wiring (e.g. in cmd/rides/main.go) to enable real ride type names in events.
func (s *Service) SetRideTypeNameFetcher(fn func(ctx context.Context, id uuid.UUID) (string, error)) {
	s.rideTypeNameFetcher = fn
}

// publishEvent publishes an event asynchronously. Failures are logged but don't affect the caller.
func (s *Service) publishEvent(subject string, eventType, source string, data interface{}) {
	if s.eventBus == nil {
		return
	}
	go func() {
		evt, err := eventbus.NewEvent(eventType, source, data)
		if err != nil {
			logger.Warn("failed to create event", zap.String("type", eventType), zap.Error(err))
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.eventBus.Publish(ctx, subject, evt); err != nil {
			logger.Warn("failed to publish event", zap.String("type", eventType), zap.Error(err))
		}
	}()
}

// MatchDrivers finds and scores the best drivers for a pickup location.
func (s *Service) MatchDrivers(ctx context.Context, pickupLatitude, pickupLongitude float64) ([]*DriverCandidate, error) {
	if s.matcher == nil {
		return nil, common.NewInternalServerError("matching engine not configured")
	}
	return s.matcher.FindBestDrivers(ctx, pickupLatitude, pickupLongitude)
}

// EnableMLPredictions wires an optional ML ETA client with breaker protection.
func (s *Service) EnableMLPredictions(client *httpclient.Client, breaker *resilience.CircuitBreaker) {
	s.mlEtaClient = client
	s.mlEtaBreaker = breaker
}

// RequestRide creates a new ride request
func (s *Service) RequestRide(ctx context.Context, riderID uuid.UUID, req *models.RideRequest) (*models.Ride, error) {
	ctx, span := tracing.StartSpan(ctx, "rides-service", "RequestRide")
	defer span.End()

	tracing.AddSpanAttributes(ctx,
		tracing.UserIDKey.String(riderID.String()),
		tracing.LocationLatitudeKey.Float64(req.PickupLatitude),
		tracing.LocationLongitudeKey.Float64(req.PickupLongitude),
	)

	// Calculate estimated values
	distance := calculateDistance(
		req.PickupLatitude, req.PickupLongitude,
		req.DropoffLatitude, req.DropoffLongitude,
	)

	duration := estimateDuration(distance)
	if mlDuration, ok := s.predictETAFromML(ctx, req); ok && mlDuration > 0 {
		duration = int(math.Round(mlDuration))
	}

	var fare float64
	var surgeMultiplier float64
	var rideTypeID *uuid.UUID
	var currencyCode string
	var pricingVersionID *uuid.UUID
	var countryID, regionID, cityID, pickupZoneID, dropoffZoneID *uuid.UUID

	// Resolve geographic hierarchy for the pickup location (non-blocking, best-effort).
	// Industry pattern: stamp rides at creation time so analytics, pricing, and
	// compliance rules can be applied without re-querying geo boundaries later.
	if s.locationResolver != nil {
		resolveCtx, resolveCancel := context.WithTimeout(ctx, 500*time.Millisecond)
		if loc, err := s.locationResolver.ResolveLocationContext(resolveCtx, req.PickupLatitude, req.PickupLongitude); err == nil {
			countryID = loc.CountryID
			regionID = loc.RegionID
			cityID = loc.CityID
			pickupZoneID = loc.PricingZoneID
		} else {
			logger.WarnContext(ctx, "geo resolve failed for pickup, ride will have nil location IDs",
				zap.Float64("latitude", req.PickupLatitude),
				zap.Float64("longitude", req.PickupLongitude),
				zap.Error(err))
		}
		resolveCancel()

		// Also resolve dropoff zone (may differ from pickup zone)
		if pickupZoneID != nil {
			resolveCtx2, resolveCancel2 := context.WithTimeout(ctx, 500*time.Millisecond)
			if loc, err := s.locationResolver.ResolveLocationContext(resolveCtx2, req.DropoffLatitude, req.DropoffLongitude); err == nil {
				dropoffZoneID = loc.PricingZoneID
			}
			resolveCancel2()
		}
	}

	rideTypeID = req.RideTypeID

	// Use hierarchical pricing engine when available
	if s.pricingService != nil {
		calculation, err := s.pricingService.CalculateFare(ctx, pricing.CalculateInput{
			PickupLatitude:   req.PickupLatitude,
			PickupLongitude:  req.PickupLongitude,
			DropoffLatitude:  req.DropoffLatitude,
			DropoffLongitude: req.DropoffLongitude,
			DistanceKm:       distance,
			DurationMin:      duration,
			RideTypeID:       rideTypeID,
			Currency:         "USD", // Will be resolved by pricing engine
		})
		if err != nil {
			logger.Warn("hierarchical pricing failed, falling back to flat pricing", zap.Error(err))
			surgeMultiplier = calculateSurgeMultiplier(time.Now())
			fare = s.calculateFare(distance, duration, surgeMultiplier)
			currencyCode = "USD"
		} else {
			fare = calculation.TotalFare
			surgeMultiplier = calculation.TotalMultiplier
			currencyCode = calculation.Currency
			if calculation.PricingVersionID != uuid.Nil {
				pricingVersionID = &calculation.PricingVersionID
			}
		}
	} else {
		// Fallback: use dynamic surge pricing if available, otherwise time-based
		if s.surgeCalculator != nil {
			var err error
			surgeMultiplier, err = s.surgeCalculator.CalculateSurgeMultiplier(ctx, req.PickupLatitude, req.PickupLongitude)
			if err != nil {
				surgeMultiplier = calculateSurgeMultiplier(time.Now())
			}
		} else {
			surgeMultiplier = calculateSurgeMultiplier(time.Now())
		}

		fare = s.calculateFare(distance, duration, surgeMultiplier)
		currencyCode = "USD"
	}

	// Apply promo code if provided
	var promoCodeID *uuid.UUID
	var discountAmount float64
	if req.PromoCode != "" {
		validation, err := s.validatePromoCode(ctx, req.PromoCode, riderID, fare)
		if err == nil && validation.Valid {
			promoCodeID = &validation.PromoCodeID
			discountAmount = validation.DiscountAmount
			fare = validation.FinalAmount
		}
	}

	ride := &models.Ride{
		ID:                uuid.New(),
		RiderID:           riderID,
		Status:            models.RideStatusRequested,
		PickupLatitude:    req.PickupLatitude,
		PickupLongitude:   req.PickupLongitude,
		PickupAddress:     req.PickupAddress,
		DropoffLatitude:   req.DropoffLatitude,
		DropoffLongitude:  req.DropoffLongitude,
		DropoffAddress:    req.DropoffAddress,
		EstimatedDistance: distance,
		EstimatedDuration: duration,
		EstimatedFare:     fare,
		SurgeMultiplier:   surgeMultiplier,
		RequestedAt:       time.Now(),
		RideTypeID:        rideTypeID,
		PromoCodeID:       promoCodeID,
		DiscountAmount:    discountAmount,
		CountryID:         countryID,
		RegionID:          regionID,
		CityID:            cityID,
		PickupZoneID:      pickupZoneID,
		DropoffZoneID:     dropoffZoneID,
		CurrencyCode:      currencyCode,
		PricingVersionID:  pricingVersionID,
	}

	// Handle scheduled rides
	if req.IsScheduled && req.ScheduledAt != nil {
		ride.IsScheduled = true
		ride.ScheduledAt = req.ScheduledAt
		ride.ScheduledNotificationSent = false
	}

	if err := s.repo.CreateRide(ctx, ride); err != nil {
		tracing.RecordError(ctx, err)
		return nil, common.NewInternalServerError("failed to create ride request")
	}

	// Add final ride attributes to span
	tracing.AddSpanAttributes(ctx,
		tracing.RideIDKey.String(ride.ID.String()),
		tracing.FareAmountKey.Float64(fare),
		tracing.DistanceKey.Float64(distance),
		tracing.DurationKey.Int(duration),
		attribute.Float64("surge_multiplier", surgeMultiplier),
	)

	// Fetch rider info for event enrichment (non-blocking, with graceful fallbacks)
	riderName := "Rider" // Default fallback
	riderRating := 5.0   // Default fallback (new riders start at 5.0)
	if rider, err := s.repo.GetUserProfile(ctx, riderID); err == nil && rider != nil {
		if rider.FirstName != "" {
			riderName = rider.FirstName
			if rider.LastName != "" {
				riderName = rider.FirstName + " " + rider.LastName
			}
		}
		// Note: User model doesn't have rating field
		// Rating is tracked separately in ratings service
		// For MVP, use default 5.0 for all riders
		// TODO: Add rating fetch from ratings service for enriched data
	} else {
		logger.WarnContext(ctx, "failed to fetch rider profile for event, using defaults",
			zap.String("rider_id", riderID.String()),
			zap.Error(err))
	}

	// Get ride type name (non-blocking fallback)
	rideTypeName := "Standard" // Default
	resolvedRideTypeID := uuid.Nil
	if rideTypeID != nil {
		resolvedRideTypeID = *rideTypeID
		if s.rideTypeNameFetcher != nil {
			if name, err := s.rideTypeNameFetcher(ctx, *rideTypeID); err == nil {
				rideTypeName = name
			} else {
				logger.WarnContext(ctx, "failed to fetch ride type name, using default",
					zap.String("ride_type_id", rideTypeID.String()),
					zap.Error(err))
			}
		}
	}

	// Publish comprehensive event for matching service
	s.publishEvent(eventbus.SubjectRideRequested, "ride.requested", "rides-service", eventbus.RideRequestedData{
		RideID:            ride.ID,
		RiderID:           riderID,
		RiderName:         riderName,
		RiderRating:       riderRating,
		PickupLatitude:    req.PickupLatitude,
		PickupLongitude:   req.PickupLongitude,
		PickupAddress:     req.PickupAddress,
		DropoffLatitude:   req.DropoffLatitude,
		DropoffLongitude:  req.DropoffLongitude,
		DropoffAddress:    req.DropoffAddress,
		RideTypeID:        resolvedRideTypeID,
		RideTypeName:      rideTypeName,
		EstimatedFare:     fare,
		EstimatedDistance: distance,
		EstimatedDuration: duration,
		Currency:          currencyCode,
		RequestedAt:       ride.RequestedAt,
	})

	return ride, nil
}

// GetRide retrieves a ride by ID
func (s *Service) GetRide(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
	ride, err := s.repo.GetRideByID(ctx, rideID)
	if err != nil {
		return nil, common.NewNotFoundError("ride not found", nil)
	}

	return ride, nil
}

// AcceptRide allows a driver to accept a ride request
func (s *Service) AcceptRide(ctx context.Context, rideID, driverID uuid.UUID) (*models.Ride, error) {
	ctx, span := tracing.StartSpan(ctx, "rides-service", "AcceptRide")
	defer span.End()

	tracing.AddSpanAttributes(ctx,
		tracing.RideIDKey.String(rideID.String()),
		tracing.DriverIDKey.String(driverID.String()),
	)

	// Atomic accept: single UPDATE with status guard prevents double-accept race condition
	accepted, err := s.repo.AtomicAcceptRide(ctx, rideID, driverID)
	if err != nil {
		tracing.RecordError(ctx, err)
		return nil, common.NewInternalServerError("failed to accept ride")
	}
	if !accepted {
		return nil, common.NewErrorWithCode(409, common.ErrCodeRideNotAvailable,
			"ride is no longer available for acceptance", nil)
	}

	// Re-fetch ride with updated fields
	ride, err := s.repo.GetRideByID(ctx, rideID)
	if err != nil {
		tracing.RecordError(ctx, err)
		return nil, common.NewInternalServerError("failed to fetch accepted ride")
	}

	tracing.AddSpanEvent(ctx, "ride_accepted",
		attribute.String("ride_id", rideID.String()),
		attribute.String("driver_id", driverID.String()),
	)

	s.publishEvent(eventbus.SubjectRideAccepted, "ride.accepted", "rides-service", eventbus.RideAcceptedData{
		RideID:           rideID,
		RiderID:          ride.RiderID,
		DriverID:         driverID,
		PickupLatitude:   ride.PickupLatitude,
		PickupLongitude:  ride.PickupLongitude,
		DropoffLatitude:  ride.DropoffLatitude,
		DropoffLongitude: ride.DropoffLongitude,
		AcceptedAt:       *ride.AcceptedAt,
	})

	return ride, nil
}

// StartRide marks a ride as in progress
func (s *Service) StartRide(ctx context.Context, rideID, driverID uuid.UUID) (*models.Ride, error) {
	ride, err := s.repo.GetRideByID(ctx, rideID)
	if err != nil {
		return nil, common.NewNotFoundError("ride not found", nil)
	}

	if ride.Status != models.RideStatusAccepted {
		return nil, common.NewBadRequestError("ride has not been accepted", nil)
	}

	if ride.DriverID == nil || *ride.DriverID != driverID {
		return nil, common.NewBadRequestError("unauthorized driver", nil)
	}

	if err := s.repo.UpdateRideStatus(ctx, rideID, models.RideStatusInProgress, nil); err != nil {
		return nil, common.NewInternalServerError("failed to start ride")
	}

	ride.Status = models.RideStatusInProgress
	now := time.Now()
	ride.StartedAt = &now

	s.publishEvent(eventbus.SubjectRideStarted, "ride.started", "rides-service", eventbus.RideStartedData{
		RideID:    rideID,
		RiderID:   ride.RiderID,
		DriverID:  driverID,
		StartedAt: now,
	})

	return ride, nil
}

// CompleteRide marks a ride as completed
func (s *Service) CompleteRide(ctx context.Context, rideID, driverID uuid.UUID, actualDistance float64) (*models.Ride, error) {
	ctx, span := tracing.StartSpan(ctx, "rides-service", "CompleteRide")
	defer span.End()

	tracing.AddSpanAttributes(ctx,
		tracing.RideIDKey.String(rideID.String()),
		tracing.DriverIDKey.String(driverID.String()),
		tracing.DistanceKey.Float64(actualDistance),
	)

	ride, err := s.repo.GetRideByID(ctx, rideID)
	if err != nil {
		tracing.RecordError(ctx, err)
		return nil, common.NewNotFoundError("ride not found", nil)
	}

	if ride.Status != models.RideStatusInProgress {
		err := fmt.Errorf("ride status is %s, not in progress", ride.Status)
		tracing.RecordError(ctx, err)
		return nil, common.NewBadRequestError("ride is not in progress", nil)
	}

	if ride.DriverID == nil || *ride.DriverID != driverID {
		err := fmt.Errorf("unauthorized driver")
		tracing.RecordError(ctx, err)
		return nil, common.NewBadRequestError("unauthorized driver", nil)
	}

	// Calculate actual duration
	actualDuration := int(time.Since(*ride.StartedAt).Minutes())

	// Calculate final fare based on actual distance and duration
	var finalFare, driverEarnings float64
	if s.pricingService != nil {
		calculation, err := s.pricingService.CalculateFare(ctx, pricing.CalculateInput{
			PickupLatitude:   ride.PickupLatitude,
			PickupLongitude:  ride.PickupLongitude,
			DropoffLatitude:  ride.DropoffLatitude,
			DropoffLongitude: ride.DropoffLongitude,
			DistanceKm:       actualDistance,
			DurationMin:      actualDuration,
			RideTypeID:       ride.RideTypeID,
			Currency:         ride.CurrencyCode,
		})
		if err != nil {
			logger.Warn("hierarchical pricing failed on completion, falling back to flat pricing", zap.Error(err))
			finalFare = s.calculateFare(actualDistance, actualDuration, ride.SurgeMultiplier)
			driverEarnings = finalFare * 0.80 // flat 20% commission fallback
		} else {
			finalFare = calculation.TotalFare
			driverEarnings = calculation.DriverEarnings
		}
	} else {
		finalFare = s.calculateFare(actualDistance, actualDuration, ride.SurgeMultiplier)
		driverEarnings = finalFare * 0.80
	}

	tracing.AddSpanAttributes(ctx,
		tracing.DurationKey.Int(actualDuration),
		tracing.FareAmountKey.Float64(finalFare),
		attribute.Float64("estimated_fare", ride.EstimatedFare),
	)

	// Atomic completion: single UPDATE sets fare data + status with driver guard
	completed, err := s.repo.AtomicCompleteRide(ctx, rideID, driverID, actualDistance, actualDuration, finalFare)
	if err != nil {
		tracing.RecordError(ctx, err)
		return nil, common.NewInternalServerError("failed to complete ride")
	}
	if !completed {
		return nil, common.NewErrorWithCode(409, common.ErrCodeRideAlreadyDone,
			"ride is no longer in progress", nil)
	}

	ride.Status = models.RideStatusCompleted
	ride.ActualDistance = &actualDistance
	ride.ActualDuration = &actualDuration
	ride.FinalFare = &finalFare
	now := time.Now()
	ride.CompletedAt = &now

	currency := ride.CurrencyCode
	if currency == "" {
		currency = "USD"
	}
	s.publishEvent(eventbus.SubjectRideCompleted, "ride.completed", "rides-service", eventbus.RideCompletedData{
		RideID:         rideID,
		RiderID:        ride.RiderID,
		DriverID:       driverID,
		FareAmount:     finalFare,
		DriverEarnings: driverEarnings,
		Currency:       currency,
		DistanceKm:     actualDistance,
		DurationMin:    float64(actualDuration),
		CompletedAt:    now,
	})

	return ride, nil
}

// CancelRide cancels a ride
func (s *Service) CancelRide(ctx context.Context, rideID, userID uuid.UUID, reason string) (*models.Ride, error) {
	ride, err := s.repo.GetRideByID(ctx, rideID)
	if err != nil {
		return nil, common.NewNotFoundError("ride not found", nil)
	}

	// Verify user is either the rider or driver
	isRider := ride.RiderID == userID
	isDriver := ride.DriverID != nil && *ride.DriverID == userID

	if !isRider && !isDriver {
		return nil, common.NewBadRequestError("unauthorized to cancel this ride", nil)
	}

	if ride.Status == models.RideStatusCompleted || ride.Status == models.RideStatusCancelled {
		return nil, common.NewBadRequestError("cannot cancel completed or already cancelled ride", nil)
	}

	if err := s.repo.UpdateRideStatus(ctx, rideID, models.RideStatusCancelled, nil); err != nil {
		return nil, common.NewInternalServerError("failed to cancel ride")
	}

	ride.Status = models.RideStatusCancelled
	now := time.Now()
	ride.CancelledAt = &now
	ride.CancellationReason = &reason

	cancelledBy := "rider"
	if isDriver {
		cancelledBy = "driver"
	}
	driverID := uuid.Nil
	if ride.DriverID != nil {
		driverID = *ride.DriverID
	}
	s.publishEvent(eventbus.SubjectRideCancelled, "ride.cancelled", "rides-service", eventbus.RideCancelledData{
		RideID:      rideID,
		RiderID:     ride.RiderID,
		DriverID:    driverID,
		CancelledBy: cancelledBy,
		Reason:      reason,
		CancelledAt: now,
	})

	return ride, nil
}

// RateRide allows a rider to rate a completed ride
func (s *Service) RateRide(ctx context.Context, rideID, riderID uuid.UUID, req *models.RideRatingRequest) error {
	ride, err := s.repo.GetRideByID(ctx, rideID)
	if err != nil {
		return common.NewNotFoundError("ride not found", nil)
	}

	if ride.RiderID != riderID {
		return common.NewBadRequestError("unauthorized to rate this ride", nil)
	}

	if ride.Status != models.RideStatusCompleted {
		return common.NewBadRequestError("can only rate completed rides", nil)
	}

	if err := s.repo.UpdateRideRating(ctx, rideID, req.Rating, req.Feedback); err != nil {
		return common.NewInternalServerError("failed to rate ride")
	}

	return nil
}

// GetRiderRides retrieves rides for a rider with optional filters.
func (s *Service) GetRiderRides(ctx context.Context, riderID uuid.UUID, filters *RideFilters, limit, offset int) ([]*models.Ride, int64, error) {
	if filters == nil {
		filters = &RideFilters{}
	}
	rides, total, err := s.repo.GetRidesByRiderWithFilters(ctx, riderID, filters, limit, offset)
	if err != nil {
		return nil, 0, common.NewInternalServerError("failed to get rides")
	}
	return rides, int64(total), nil
}

// GetDriverRides retrieves rides for a driver with optional filters.
func (s *Service) GetDriverRides(ctx context.Context, driverID uuid.UUID, filters *RideFilters, limit, offset int) ([]*models.Ride, int64, error) {
	if filters == nil {
		filters = &RideFilters{}
	}
	rides, total, err := s.repo.GetRidesByDriverWithFilters(ctx, driverID, filters, limit, offset)
	if err != nil {
		return nil, 0, common.NewInternalServerError("failed to get rides")
	}
	return rides, int64(total), nil
}

// GetAvailableRides retrieves all available ride requests
func (s *Service) GetAvailableRides(ctx context.Context) ([]*models.Ride, error) {
	rides, err := s.repo.GetPendingRides(ctx)
	if err != nil {
		return nil, common.NewInternalServerError("failed to get available rides")
	}

	return rides, nil
}

// Helper functions

func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	return pkggeo.Haversine(lat1, lon1, lat2, lon2)
}

func estimateDuration(distance float64) int {
	return pkggeo.EstimateDuration(distance)
}

// calculateFare calculates ride fare using service pricing config
func (s *Service) calculateFare(distance float64, duration int, surgeMultiplier float64) float64 {
	cfg := s.pricingConfig
	if cfg == nil {
		cfg = DefaultPricingConfig()
	}

	baseFare := (distance * cfg.BaseFarePerKm) + (float64(duration) * cfg.BaseFarePerMinute)
	fare := baseFare * surgeMultiplier

	if fare < cfg.MinimumFare {
		fare = cfg.MinimumFare
	}

	return math.Round(fare*100) / 100 // Round to 2 decimal places
}

// calculateSurgeMultiplier calculates surge pricing multiplier based on time
func calculateSurgeMultiplier(t time.Time) float64 {
	hour := t.Hour()

	// Peak hours: 7-9 AM and 5-8 PM
	if (hour >= 7 && hour < 9) || (hour >= 17 && hour < 20) {
		return 1.5
	}

	// Late night: 11 PM - 5 AM
	if hour >= 23 || hour < 5 {
		return 1.3
	}

	return 1.0
}

// GetUserProfile retrieves a user's profile
func (s *Service) GetUserProfile(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	return s.repo.GetUserProfile(ctx, userID)
}

// UpdateUserProfile updates a user's profile
func (s *Service) UpdateUserProfile(ctx context.Context, userID uuid.UUID, firstName, lastName, phoneNumber string) error {
	return s.repo.UpdateUserProfile(ctx, userID, firstName, lastName, phoneNumber)
}

// PromoCodeValidation represents the response from promo code validation
type PromoCodeValidation struct {
	Valid          bool      `json:"valid"`
	PromoCodeID    uuid.UUID `json:"promo_code_id"`
	DiscountAmount float64   `json:"discount_amount"`
	FinalAmount    float64   `json:"final_amount"`
	Message        string    `json:"message"`
}

// validatePromoCode validates a promo code with the Promos service
func (s *Service) validatePromoCode(ctx context.Context, code string, riderID uuid.UUID, rideAmount float64) (*PromoCodeValidation, error) {
	requestBody := map[string]interface{}{
		"code":        code,
		"ride_amount": rideAmount,
	}

	// Get JWT token from context (if available)
	headers := make(map[string]string)
	// In a real scenario, you'd extract the JWT from the context
	// For now, we'll make a direct service-to-service call

	respBody, err := s.postToPromos(ctx, "/api/v1/promo-codes/validate", requestBody, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to validate promo code: %w", err)
	}

	var validation PromoCodeValidation
	if err := json.Unmarshal(respBody, &validation); err != nil {
		return nil, fmt.Errorf("failed to parse validation response: %w", err)
	}

	return &validation, nil
}

type mlEtaPredictRequest struct {
	PickupLatitude   float64 `json:"pickup_latitude"`
	PickupLongitude  float64 `json:"pickup_longitude"`
	DropoffLatitude  float64 `json:"dropoff_latitude"`
	DropoffLongitude float64 `json:"dropoff_longitude"`
	TrafficLevel     string  `json:"traffic_level"`
	Weather          string  `json:"weather"`
	DriverID         string  `json:"driver_id"`
	RideTypeID       int     `json:"ride_type_id"`
}

type mlEtaPredictResponse struct {
	EstimatedMinutes float64 `json:"estimated_minutes"`
	EstimatedSeconds int     `json:"estimated_seconds"`
}

func (s *Service) predictETAFromML(ctx context.Context, req *models.RideRequest) (float64, bool) {
	if s.mlEtaClient == nil {
		return 0, false
	}

	payload := mlEtaPredictRequest{
		PickupLatitude:   req.PickupLatitude,
		PickupLongitude:  req.PickupLongitude,
		DropoffLatitude:  req.DropoffLatitude,
		DropoffLongitude: req.DropoffLongitude,
		TrafficLevel:     "medium",
		Weather:          "clear",
		DriverID:         "",
	}
	if req.RideTypeID != nil {
		payload.RideTypeID = 1
	}

	call := func(ctx context.Context) (interface{}, error) {
		body, err := s.mlEtaClient.Post(ctx, "/api/v1/eta/predict", payload, nil)
		if err != nil {
			return nil, err
		}

		var resp mlEtaPredictResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, err
		}
		return &resp, nil
	}

	var (
		result interface{}
		err    error
	)

	if s.mlEtaBreaker != nil {
		result, err = s.mlEtaBreaker.Execute(ctx, call)
	} else {
		result, err = call(ctx)
	}

	if err != nil {
		return 0, false
	}

	resp, ok := result.(*mlEtaPredictResponse)
	if !ok || resp == nil || resp.EstimatedMinutes <= 0 {
		return 0, false
	}

	return resp.EstimatedMinutes, true
}

func (s *Service) postToPromos(ctx context.Context, path string, body interface{}, headers map[string]string) ([]byte, error) {
	if s.promosBreaker == nil {
		return s.promosClient.Post(ctx, path, body, headers)
	}

	result, err := s.promosBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return s.promosClient.Post(ctx, path, body, headers)
	})
	if err != nil {
		return nil, err
	}

	bytes, ok := result.([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected response type from promos service")
	}

	return bytes, nil
}
