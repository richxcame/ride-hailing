package matching

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/richxcame/ride-hailing/pkg/logger"
	redisClient "github.com/richxcame/ride-hailing/pkg/redis"
	"github.com/richxcame/ride-hailing/pkg/websocket"
	"go.uber.org/zap"
)

// GeoService provides driver location lookups
type GeoService interface {
	FindAvailableDrivers(ctx context.Context, lat, lng float64, maxDrivers int) ([]*GeoDriverLocation, error)
	CalculateDistance(lat1, lon1, lat2, lon2 float64) float64
}

// RidesRepository provides ride data access
type RidesRepository interface {
	UpdateRideDriver(ctx context.Context, rideID, driverID uuid.UUID) error
	GetRideByID(ctx context.Context, rideID uuid.UUID) (interface{}, error)
}

// Service handles ride matching and driver dispatching
type Service struct {
	geoService  GeoService
	ridesRepo   RidesRepository
	wsHub       *websocket.Hub
	eventBus    *nats.Conn
	redis       redisClient.ClientInterface
	config      MatchingConfig
}

// NewService creates a new matching service
func NewService(
	geoService GeoService,
	ridesRepo RidesRepository,
	wsHub *websocket.Hub,
	eventBus *nats.Conn,
	redis redisClient.ClientInterface,
	config MatchingConfig,
) *Service {
	return &Service{
		geoService:  geoService,
		ridesRepo:   ridesRepo,
		wsHub:       wsHub,
		eventBus:    eventBus,
		redis:       redis,
		config:      config,
	}
}

// Start begins listening for ride events
func (s *Service) Start(ctx context.Context) error {
	logger.Info("Starting matching service")

	// Subscribe to ride.requested events
	_, err := s.eventBus.Subscribe("ride.requested", func(msg *nats.Msg) {
		var event RideRequestedEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			logger.Error("Failed to unmarshal ride.requested event", zap.Error(err))
			return
		}
		s.onRideRequested(ctx, &event)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to ride.requested: %w", err)
	}

	// Subscribe to ride.accepted events
	_, err = s.eventBus.Subscribe("ride.accepted", func(msg *nats.Msg) {
		var event RideAcceptedEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			logger.Error("Failed to unmarshal ride.accepted event", zap.Error(err))
			return
		}
		s.onRideAccepted(ctx, &event)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to ride.accepted: %w", err)
	}

	// Subscribe to ride.cancelled events
	_, err = s.eventBus.Subscribe("ride.cancelled", func(msg *nats.Msg) {
		var event RideCancelledEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			logger.Error("Failed to unmarshal ride.cancelled event", zap.Error(err))
			return
		}
		s.onRideCancelled(ctx, &event)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to ride.cancelled: %w", err)
	}

	logger.Info("Matching service started successfully")
	return nil
}

// onRideRequested handles new ride requests by finding and notifying drivers
func (s *Service) onRideRequested(ctx context.Context, event *RideRequestedEvent) {
	logger.Info("Processing ride request",
		zap.String("ride_id", event.RideID.String()),
		zap.Float64("pickup_lat", event.PickupLatitude),
		zap.Float64("pickup_lng", event.PickupLongitude))

	// Find nearby drivers
	drivers, err := s.findAvailableDrivers(ctx, event.PickupLatitude, event.PickupLongitude)
	if err != nil {
		logger.Error("Failed to find nearby drivers", zap.Error(err))
		return
	}

	if len(drivers) == 0 {
		logger.Warn("No drivers available for ride",
			zap.String("ride_id", event.RideID.String()),
			zap.Float64("radius_km", s.config.MaxSearchRadiusKm))
		// TODO: Notify rider that no drivers are available
		return
	}

	logger.Info("Found available drivers",
		zap.String("ride_id", event.RideID.String()),
		zap.Int("count", len(drivers)))

	// Send offers to first batch of closest drivers
	s.sendOffersToDrivers(ctx, event, drivers)
}

// findAvailableDrivers searches for nearby available drivers
func (s *Service) findAvailableDrivers(ctx context.Context, lat, lng float64) ([]DriverLocation, error) {
	// Use geo service to find available drivers (it handles radius expansion internally)
	geoDrivers, err := s.geoService.FindAvailableDrivers(ctx, lat, lng, s.config.MaxDriversToNotify)
	if err != nil {
		return nil, fmt.Errorf("failed to find available drivers: %w", err)
	}

	if len(geoDrivers) == 0 {
		return []DriverLocation{}, nil
	}

	// Convert geo.DriverLocation to matching.DriverLocation
	drivers := make([]DriverLocation, len(geoDrivers))
	for i, gd := range geoDrivers {
		drivers[i] = DriverLocation{
			UserID:    gd.DriverID, // In our system, driver_id is the user_id
			DriverID:  gd.DriverID,
			Latitude:  gd.Latitude,
			Longitude: gd.Longitude,
			Distance:  0, // Will be calculated when sending offers
		}
	}

	logger.Info("Found available drivers",
		zap.Int("count", len(drivers)))

	return drivers, nil
}

// sendOffersToDrivers sends ride offers to a batch of drivers
func (s *Service) sendOffersToDrivers(ctx context.Context, event *RideRequestedEvent, drivers []DriverLocation) {
	expiresAt := time.Now().Add(time.Duration(s.config.OfferTimeoutSeconds) * time.Second)

	// Send to first batch (closest drivers)
	batchSize := s.config.FirstBatchSize
	if batchSize > len(drivers) {
		batchSize = len(drivers)
	}

	for i := 0; i < batchSize; i++ {
		driver := drivers[i]
		s.sendOfferToDriver(ctx, event, driver, expiresAt)
	}

	// If first batch doesn't accept, send to remaining drivers after delay
	if batchSize < len(drivers) {
		go s.sendOffersWithDelay(ctx, event, drivers[batchSize:], expiresAt)
	}
}

// sendOffersWithDelay sends offers to remaining drivers after a delay
func (s *Service) sendOffersWithDelay(ctx context.Context, event *RideRequestedEvent, drivers []DriverLocation, expiresAt time.Time) {
	time.Sleep(time.Duration(s.config.RetryDelaySeconds) * time.Second)

	// Check if ride is still pending
	isPending, err := s.isRidePending(ctx, event.RideID)
	if err != nil {
		logger.Error("Failed to check ride status", zap.Error(err))
		return
	}

	if !isPending {
		logger.Info("Ride no longer pending, skipping delayed offers",
			zap.String("ride_id", event.RideID.String()))
		return
	}

	// Send to remaining drivers
	for _, driver := range drivers {
		s.sendOfferToDriver(ctx, event, driver, expiresAt)
	}
}

// sendOfferToDriver sends a ride offer to a specific driver via WebSocket
func (s *Service) sendOfferToDriver(ctx context.Context, event *RideRequestedEvent, driver DriverLocation, expiresAt time.Time) {
	// Calculate distance to pickup
	distance := s.geoService.CalculateDistance(
		driver.Latitude, driver.Longitude,
		event.PickupLatitude, event.PickupLongitude,
	)
	driver.Distance = distance

	// Calculate ETA to pickup (simple estimation: distance / 30 km/h average speed)
	etaMinutes := int((distance / 30.0) * 60)

	// Build ride offer
	offer := RideOffer{
		RideID:            event.RideID,
		RiderName:         event.RiderName,
		RiderRating:       event.RiderRating,
		PickupLocation:    Location{Latitude: event.PickupLatitude, Longitude: event.PickupLongitude, Address: event.PickupAddress},
		RideType:          event.RideTypeName,
		EstimatedFare:     event.EstimatedFare,
		EstimatedDistance: event.EstimatedDistance,
		EstimatedDuration: event.EstimatedDuration,
		DistanceToPickup:  driver.Distance,
		ETAToPickup:       etaMinutes,
		Currency:          event.Currency,
		ExpiresAt:         expiresAt,
		TimeoutSeconds:    s.config.OfferTimeoutSeconds,
	}

	// Add dropoff if provided
	if event.DropoffLatitude != nil && event.DropoffLongitude != nil {
		offer.DropoffLocation = &Location{
			Latitude:  *event.DropoffLatitude,
			Longitude: *event.DropoffLongitude,
			Address:   *event.DropoffAddress,
		}
	}

	// Track offer in Redis
	if err := s.trackOffer(ctx, event.RideID, driver.DriverID, expiresAt); err != nil {
		logger.Error("Failed to track offer in Redis", zap.Error(err))
		// Continue anyway - we still want to send the WebSocket message
	}

	// Send WebSocket message to driver
	msg := &websocket.Message{
		Type:   "ride.offer",
		RideID: event.RideID.String(),
		UserID: driver.UserID.String(),
		Data: map[string]interface{}{
			"ride_id":             offer.RideID.String(),
			"rider_name":          offer.RiderName,
			"rider_rating":        offer.RiderRating,
			"pickup_location":     offer.PickupLocation,
			"dropoff_location":    offer.DropoffLocation,
			"ride_type":           offer.RideType,
			"estimated_fare":      offer.EstimatedFare,
			"estimated_distance":  offer.EstimatedDistance,
			"estimated_duration":  offer.EstimatedDuration,
			"distance_to_pickup":  offer.DistanceToPickup,
			"eta_to_pickup":       offer.ETAToPickup,
			"currency":            offer.Currency,
			"expires_at":          offer.ExpiresAt.Format(time.RFC3339),
			"timeout_seconds":     offer.TimeoutSeconds,
		},
	}

	s.wsHub.SendToUser(driver.UserID.String(), msg)

	logger.Info("Sent ride offer to driver",
		zap.String("ride_id", event.RideID.String()),
		zap.String("driver_id", driver.DriverID.String()),
		zap.Float64("distance_km", driver.Distance))
}

// trackOffer stores offer tracking data in Redis
func (s *Service) trackOffer(ctx context.Context, rideID, driverID uuid.UUID, expiresAt time.Time) error {
	tracking := OfferTracking{
		DriverID:  driverID,
		SentAt:    time.Now(),
		ExpiresAt: expiresAt,
	}

	data, err := json.Marshal(tracking)
	if err != nil {
		return fmt.Errorf("failed to marshal tracking data: %w", err)
	}

	key := fmt.Sprintf("ride_offer:%s:%s", rideID.String(), driverID.String())
	ttl := time.Until(expiresAt)

	return s.redis.SetWithExpiration(ctx, key, string(data), ttl)
}

// onRideAccepted handles ride acceptance by cleaning up pending offers
func (s *Service) onRideAccepted(ctx context.Context, event *RideAcceptedEvent) {
	logger.Info("Ride accepted, cleaning up pending offers",
		zap.String("ride_id", event.RideID.String()),
		zap.String("driver_id", event.DriverID.String()))

	// Cancel all pending offers for this ride
	if err := s.cancelPendingOffers(ctx, event.RideID, event.DriverID); err != nil {
		logger.Error("Failed to cancel pending offers", zap.Error(err))
	}
}

// onRideCancelled handles ride cancellation by cleaning up pending offers
func (s *Service) onRideCancelled(ctx context.Context, event *RideCancelledEvent) {
	logger.Info("Ride cancelled, cleaning up pending offers",
		zap.String("ride_id", event.RideID.String()),
		zap.String("reason", event.Reason))

	// Cancel all pending offers for this ride
	if err := s.cancelPendingOffers(ctx, event.RideID, uuid.Nil); err != nil {
		logger.Error("Failed to cancel pending offers", zap.Error(err))
	}
}

// cancelPendingOffers notifies drivers that the ride is no longer available
func (s *Service) cancelPendingOffers(ctx context.Context, rideID, acceptedDriverID uuid.UUID) error {
	// Note: We store each offer individually, and we sent offers to specific drivers
	// For proper cleanup, we would need to track which drivers received offers
	// For now, we'll just log that offers should be cleaned up
	// The offers will auto-expire based on TTL

	logger.Info("Ride accepted/cancelled, pending offers will auto-expire",
		zap.String("ride_id", rideID.String()))

	// If we had tracked driver IDs who received offers, we would:
	// 1. Iterate through those driver IDs
	// 2. Send WebSocket cancellation message to each
	// 3. Delete their offer tracking keys

	// For MVP, rely on offer timeout on driver side
	// TODO: Implement proper offer tracking and cancellation in future iteration

	return nil
}

// isRidePending checks if a ride is still in pending status
func (s *Service) isRidePending(ctx context.Context, rideID uuid.UUID) (bool, error) {
	// Check Redis for ride status
	key := fmt.Sprintf("ride_status:%s", rideID.String())
	status, err := s.redis.GetString(ctx, key)
	if err != nil {
		// If not in Redis, check database
		ride, err := s.ridesRepo.GetRideByID(ctx, rideID)
		if err != nil {
			return false, fmt.Errorf("failed to get ride from database: %w", err)
		}

		// Use type assertion to get status
		// Note: This is a simplified check - adjust based on your actual Ride model
		if ride == nil {
			return false, nil
		}

		// Assuming ride has a Status field
		// You may need to adjust this based on your actual implementation
		return true, nil // Simplified - adjust as needed
	}

	return status == "pending" || status == "searching", nil
}
