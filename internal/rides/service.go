package rides

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/eventbus"
	"github.com/richxcame/ride-hailing/pkg/httpclient"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/richxcame/ride-hailing/pkg/resilience"
	"github.com/richxcame/ride-hailing/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

const (
	baseFarePerKm     = 1.5  // Base fare per kilometer
	baseFarePerMinute = 0.25 // Base fare per minute
	minimumFare       = 5.0  // Minimum fare
	commissionRate    = 0.20 // 20% commission
)

// Service handles ride business logic
type Service struct {
	repo            *Repository
	promosClient    *httpclient.Client
	promosBreaker   *resilience.CircuitBreaker
	surgeCalculator SurgeCalculator
	mlEtaClient     *httpclient.Client
	mlEtaBreaker    *resilience.CircuitBreaker
	matcher         *Matcher
	eventBus        *eventbus.Bus
}

// SurgeCalculator defines the interface for surge pricing calculation
type SurgeCalculator interface {
	CalculateSurgeMultiplier(ctx context.Context, lat, lon float64) (float64, error)
	GetCurrentSurgeInfo(ctx context.Context, lat, lon float64) (map[string]interface{}, error)
}

// NewService creates a new rides service
func NewService(repo *Repository, promosServiceURL string, breaker *resilience.CircuitBreaker, httpClientTimeout ...time.Duration) *Service {
	return &Service{
		repo:            repo,
		promosClient:    httpclient.NewClient(promosServiceURL, httpClientTimeout...),
		promosBreaker:   breaker,
		surgeCalculator: nil, // Will be set via SetSurgeCalculator
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
func (s *Service) MatchDrivers(ctx context.Context, pickupLat, pickupLng float64) ([]*DriverCandidate, error) {
	if s.matcher == nil {
		return nil, common.NewInternalServerError("matching engine not configured")
	}
	return s.matcher.FindBestDrivers(ctx, pickupLat, pickupLng)
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
		tracing.LocationLatKey.Float64(req.PickupLatitude),
		tracing.LocationLonKey.Float64(req.PickupLongitude),
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

	// Use dynamic surge pricing if available, otherwise fall back to time-based
	var surgeMultiplier float64
	if s.surgeCalculator != nil {
		var err error
		surgeMultiplier, err = s.surgeCalculator.CalculateSurgeMultiplier(ctx, req.PickupLatitude, req.PickupLongitude)
		if err != nil {
			// Fallback to time-based surge on error
			surgeMultiplier = calculateSurgeMultiplier(time.Now())
		}
	} else {
		surgeMultiplier = calculateSurgeMultiplier(time.Now())
	}

	var fare float64
	var rideTypeID *uuid.UUID

	// If ride type is specified, calculate fare using ride type pricing
	if req.RideTypeID != nil {
		rideTypeID = req.RideTypeID
		calculatedFare, err := s.calculateFareWithRideType(ctx, *req.RideTypeID, distance, duration, surgeMultiplier)
		if err != nil {
			// Fall back to default pricing if ride type calculation fails
			fare = calculateFare(distance, duration, surgeMultiplier)
		} else {
			fare = calculatedFare
		}
	} else {
		// Use default fare calculation
		fare = calculateFare(distance, duration, surgeMultiplier)
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

	s.publishEvent(eventbus.SubjectRideRequested, "ride.requested", "rides-service", eventbus.RideRequestedData{
		RideID:      ride.ID,
		RiderID:     riderID,
		PickupLat:   req.PickupLatitude,
		PickupLng:   req.PickupLongitude,
		DropoffLat:  req.DropoffLatitude,
		DropoffLng:  req.DropoffLongitude,
		RideType:    func() string { if rideTypeID != nil { return rideTypeID.String() }; return "" }(),
		RequestedAt: ride.RequestedAt,
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
		RideID:     rideID,
		RiderID:    ride.RiderID,
		DriverID:   driverID,
		PickupLat:  ride.PickupLatitude,
		PickupLng:  ride.PickupLongitude,
		DropoffLat: ride.DropoffLatitude,
		DropoffLng: ride.DropoffLongitude,
		AcceptedAt: *ride.AcceptedAt,
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
	finalFare := calculateFare(actualDistance, actualDuration, ride.SurgeMultiplier)

	tracing.AddSpanAttributes(ctx,
		tracing.DurationKey.Int(actualDuration),
		tracing.FareAmountKey.Float64(finalFare),
		attribute.Float64("estimated_fare", ride.EstimatedFare),
	)

	if err := s.repo.UpdateRideCompletion(ctx, rideID, actualDistance, actualDuration, finalFare); err != nil {
		tracing.RecordError(ctx, err)
		return nil, common.NewInternalServerError("failed to update ride completion")
	}

	if err := s.repo.UpdateRideStatus(ctx, rideID, models.RideStatusCompleted, nil); err != nil {
		tracing.RecordError(ctx, err)
		return nil, common.NewInternalServerError("failed to complete ride")
	}

	ride.Status = models.RideStatusCompleted
	ride.ActualDistance = &actualDistance
	ride.ActualDuration = &actualDuration
	ride.FinalFare = &finalFare
	now := time.Now()
	ride.CompletedAt = &now

	s.publishEvent(eventbus.SubjectRideCompleted, "ride.completed", "rides-service", eventbus.RideCompletedData{
		RideID:      rideID,
		RiderID:     ride.RiderID,
		DriverID:    driverID,
		FareAmount:  finalFare,
		DistanceKm:  actualDistance,
		DurationMin: float64(actualDuration),
		CompletedAt: now,
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

// GetRiderRides retrieves rides for a rider
func (s *Service) GetRiderRides(ctx context.Context, riderID uuid.UUID, limit, offset int) ([]*models.Ride, int64, error) {
	rides, total, err := s.repo.GetRidesByRiderWithTotal(ctx, riderID, limit, offset)
	if err != nil {
		return nil, 0, common.NewInternalServerError("failed to get rides")
	}

	return rides, total, nil
}

// GetDriverRides retrieves rides for a driver
func (s *Service) GetDriverRides(ctx context.Context, driverID uuid.UUID, limit, offset int) ([]*models.Ride, int64, error) {
	rides, total, err := s.repo.GetRidesByDriverWithTotal(ctx, driverID, limit, offset)
	if err != nil {
		return nil, 0, common.NewInternalServerError("failed to get rides")
	}

	return rides, total, nil
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

// calculateDistance calculates distance between two coordinates in kilometers
// Using Haversine formula
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0 // km

	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180.0)*math.Cos(lat2*math.Pi/180.0)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distance := earthRadius * c

	return math.Round(distance*100) / 100 // Round to 2 decimal places
}

// estimateDuration estimates trip duration in minutes based on distance
// Assumes average speed of 40 km/h in city traffic
func estimateDuration(distance float64) int {
	const averageSpeed = 40.0 // km/h
	duration := (distance / averageSpeed) * 60
	return int(math.Round(duration))
}

// calculateFare calculates ride fare
func calculateFare(distance float64, duration int, surgeMultiplier float64) float64 {
	baseFare := (distance * baseFarePerKm) + (float64(duration) * baseFarePerMinute)
	fare := baseFare * surgeMultiplier

	if fare < minimumFare {
		fare = minimumFare
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
	PickupLat    float64 `json:"pickup_lat"`
	PickupLng    float64 `json:"pickup_lng"`
	DropoffLat   float64 `json:"dropoff_lat"`
	DropoffLng   float64 `json:"dropoff_lng"`
	TrafficLevel string  `json:"traffic_level"`
	Weather      string  `json:"weather"`
	DriverID     string  `json:"driver_id"`
	RideTypeID   int     `json:"ride_type_id"`
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
		PickupLat:    req.PickupLatitude,
		PickupLng:    req.PickupLongitude,
		DropoffLat:   req.DropoffLatitude,
		DropoffLng:   req.DropoffLongitude,
		TrafficLevel: "medium",
		Weather:      "clear",
		DriverID:     "",
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

// RideTypeFareResponse represents the response from ride type fare calculation
type RideTypeFareResponse struct {
	Fare            float64 `json:"fare"`
	Distance        float64 `json:"distance"`
	Duration        int     `json:"duration"`
	SurgeMultiplier float64 `json:"surge_multiplier"`
}

// calculateFareWithRideType calculates fare using the Promos service ride type pricing
func (s *Service) calculateFareWithRideType(ctx context.Context, rideTypeID uuid.UUID, distance float64, duration int, surgeMultiplier float64) (float64, error) {
	requestBody := map[string]interface{}{
		"ride_type_id":     rideTypeID.String(),
		"distance":         distance,
		"duration":         duration,
		"surge_multiplier": surgeMultiplier,
	}

	respBody, err := s.postToPromos(ctx, "/api/v1/ride-types/calculate-fare", requestBody, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate fare with ride type: %w", err)
	}

	var fareResp RideTypeFareResponse
	if err := json.Unmarshal(respBody, &fareResp); err != nil {
		return 0, fmt.Errorf("failed to parse fare response: %w", err)
	}

	return fareResp.Fare, nil
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
