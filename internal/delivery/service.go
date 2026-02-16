package delivery

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/eventbus"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// PricingConfig holds pricing configuration for deliveries
type PricingConfig struct {
	BaseFarePerKm  float64            // Base fare per km
	MinimumFare    float64            // Minimum delivery fare
	ExpressPremium float64            // Multiplier for express (e.g., 1.5 = 50% premium)
	CommissionRate float64            // Platform commission rate
	SizeSurcharges map[PackageSize]float64 // Surcharges by package size
}

// DefaultPricingConfig returns default pricing configuration
func DefaultPricingConfig() *PricingConfig {
	return &PricingConfig{
		BaseFarePerKm:  1.2,
		MinimumFare:    3.0,
		ExpressPremium: 1.5,
		CommissionRate: 0.20,
		SizeSurcharges: map[PackageSize]float64{
			PackageSizeEnvelope: 0.0,
			PackageSizeSmall:    1.0,
			PackageSizeMedium:   3.0,
			PackageSizeLarge:    8.0,
			PackageSizeXLarge:   15.0,
		},
	}
}

// Service handles delivery business logic
type Service struct {
	repo          RepositoryInterface
	eventBus      *eventbus.Bus
	pricingConfig *PricingConfig
}

// NewService creates a new delivery service
func NewService(repo RepositoryInterface) *Service {
	return &Service{
		repo:          repo,
		pricingConfig: DefaultPricingConfig(),
	}
}

// SetPricingConfig sets custom pricing configuration
func (s *Service) SetPricingConfig(config *PricingConfig) {
	if config != nil {
		s.pricingConfig = config
	}
}

// SetEventBus sets the NATS event bus for publishing delivery events
func (s *Service) SetEventBus(bus *eventbus.Bus) {
	s.eventBus = bus
}

// publishEvent publishes an event asynchronously
func (s *Service) publishEvent(subject string, eventType, source string, data interface{}) {
	if s.eventBus == nil {
		return
	}
	go func() {
		evt, err := eventbus.NewEvent(eventType, source, data)
		if err != nil {
			logger.Warn("failed to create delivery event", zap.String("type", eventType), zap.Error(err))
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.eventBus.Publish(ctx, subject, evt); err != nil {
			logger.Warn("failed to publish delivery event", zap.String("type", eventType), zap.Error(err))
		}
	}()
}

// ========================================
// ESTIMATION
// ========================================

// GetEstimate calculates fare estimate for a delivery
func (s *Service) GetEstimate(ctx context.Context, req *DeliveryEstimateRequest) (*DeliveryEstimateResponse, error) {
	distance := haversineDistance(
		req.PickupLatitude, req.PickupLongitude,
		req.DropoffLatitude, req.DropoffLongitude,
	)

	// Add distance for intermediate stops
	if len(req.Stops) > 0 {
		prevLatitude, prevLongitude := req.PickupLatitude, req.PickupLongitude
		totalStopDistance := 0.0
		for _, stop := range req.Stops {
			totalStopDistance += haversineDistance(prevLatitude, prevLongitude, stop.Latitude, stop.Longitude)
			prevLatitude, prevLongitude = stop.Latitude, stop.Longitude
		}
		totalStopDistance += haversineDistance(prevLatitude, prevLongitude, req.DropoffLatitude, req.DropoffLongitude)
		distance = totalStopDistance
	}

	// Duration estimate: ~30 km/h average in urban areas
	durationMin := int(math.Ceil(distance / 30.0 * 60.0))
	// Add pickup/dropoff buffer (5 min each + 3 min per stop)
	durationMin += 10 + len(req.Stops)*3

	cfg := s.pricingConfig
	if cfg == nil {
		cfg = DefaultPricingConfig()
	}

	baseFare := math.Max(distance*cfg.BaseFarePerKm, cfg.MinimumFare)
	sizeSurcharge := cfg.SizeSurcharges[req.PackageSize]

	prioritySurcharge := 0.0
	if req.Priority == DeliveryPriorityExpress {
		prioritySurcharge = baseFare * (cfg.ExpressPremium - 1.0) // 50% of base
	}

	surgeMultiplier := 1.0 // Could integrate with surge calculator
	total := (baseFare + sizeSurcharge + prioritySurcharge) * surgeMultiplier

	return &DeliveryEstimateResponse{
		EstimatedDistance:  math.Round(distance*100) / 100,
		EstimatedDuration: durationMin,
		BaseFare:          math.Round(baseFare*100) / 100,
		SizeSurcharge:     sizeSurcharge,
		PrioritySurcharge: math.Round(prioritySurcharge*100) / 100,
		SurgeMultiplier:   surgeMultiplier,
		TotalEstimate:     math.Round(total*100) / 100,
		Currency:          "USD",
		Priority:          req.Priority,
		PackageSize:       req.PackageSize,
	}, nil
}

// ========================================
// DELIVERY LIFECYCLE
// ========================================

// CreateDelivery creates a new delivery order
func (s *Service) CreateDelivery(ctx context.Context, senderID uuid.UUID, req *CreateDeliveryRequest) (*DeliveryResponse, error) {
	// Validate package size
	if !isValidPackageSize(req.PackageSize) {
		return nil, common.NewBadRequestError("invalid package size", nil)
	}
	if !isValidPriority(req.Priority) {
		return nil, common.NewBadRequestError("invalid delivery priority", nil)
	}

	// Get fare estimate
	estimate, err := s.GetEstimate(ctx, &DeliveryEstimateRequest{
		PickupLatitude:   req.PickupLatitude,
		PickupLongitude:  req.PickupLongitude,
		DropoffLatitude:  req.DropoffLatitude,
		DropoffLongitude: req.DropoffLongitude,
		PackageSize:      req.PackageSize,
		Priority:         req.Priority,
		Stops:            req.Stops,
	})
	if err != nil {
		return nil, fmt.Errorf("estimate: %w", err)
	}

	now := time.Now()
	deliveryID := uuid.New()
	trackingCode := generateTrackingCode()

	// Generate a PIN for proof of delivery
	pin := generatePIN()

	delivery := &Delivery{
		ID:                 deliveryID,
		SenderID:           senderID,
		Status:             DeliveryStatusRequested,
		Priority:           req.Priority,
		TrackingCode:       trackingCode,
		PickupLatitude:     req.PickupLatitude,
		PickupLongitude:    req.PickupLongitude,
		PickupAddress:      req.PickupAddress,
		PickupContact:      req.PickupContact,
		PickupPhone:        req.PickupPhone,
		PickupNotes:        req.PickupNotes,
		DropoffLatitude:    req.DropoffLatitude,
		DropoffLongitude:   req.DropoffLongitude,
		DropoffAddress:     req.DropoffAddress,
		RecipientName:      req.RecipientName,
		RecipientPhone:     req.RecipientPhone,
		DropoffNotes:       req.DropoffNotes,
		PackageSize:        req.PackageSize,
		PackageDescription: req.PackageDescription,
		WeightKg:           req.WeightKg,
		IsFragile:          req.IsFragile,
		RequiresSignature:  req.RequiresSignature,
		DeclaredValue:      req.DeclaredValue,
		EstimatedDistance:   estimate.EstimatedDistance,
		EstimatedDuration:  estimate.EstimatedDuration,
		EstimatedFare:      estimate.TotalEstimate,
		SurgeMultiplier:    estimate.SurgeMultiplier,
		ProofPIN:           &pin,
		ScheduledPickupAt:  req.ScheduledPickupAt,
		RequestedAt:        now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Build stops
	var stops []DeliveryStop
	for i, stopInput := range req.Stops {
		stops = append(stops, DeliveryStop{
			ID:           uuid.New(),
			DeliveryID:   deliveryID,
			StopOrder:    i + 1,
			Latitude:     stopInput.Latitude,
			Longitude:    stopInput.Longitude,
			Address:      stopInput.Address,
			ContactName:  stopInput.ContactName,
			ContactPhone: stopInput.ContactPhone,
			Notes:        stopInput.Notes,
			Status:       "pending",
			CreatedAt:    now,
		})
	}

	if err := s.repo.CreateDelivery(ctx, delivery, stops); err != nil {
		return nil, fmt.Errorf("create delivery: %w", err)
	}

	s.publishEvent("deliveries.requested", "delivery.requested", "delivery-service", map[string]interface{}{
		"delivery_id":       deliveryID,
		"sender_id":         senderID,
		"pickup_latitude":   req.PickupLatitude,
		"pickup_longitude":  req.PickupLongitude,
		"dropoff_latitude":  req.DropoffLatitude,
		"dropoff_longitude": req.DropoffLongitude,
		"package_size":      string(req.PackageSize),
		"priority":          string(req.Priority),
		"estimated_fare":    estimate.TotalEstimate,
		"requested_at":      now,
	})

	return &DeliveryResponse{
		Delivery: delivery,
		Stops:    stops,
	}, nil
}

// GetDelivery retrieves a delivery with full details
func (s *Service) GetDelivery(ctx context.Context, userID, deliveryID uuid.UUID) (*DeliveryResponse, error) {
	delivery, err := s.repo.GetDeliveryByID(ctx, deliveryID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("delivery not found", err)
		}
		return nil, err
	}

	// Verify access: sender or assigned driver
	if delivery.SenderID != userID && (delivery.DriverID == nil || *delivery.DriverID != userID) {
		return nil, common.NewForbiddenError("not authorized to view this delivery")
	}

	stops, _ := s.repo.GetStopsByDeliveryID(ctx, deliveryID)
	tracking, _ := s.repo.GetTrackingByDeliveryID(ctx, deliveryID)

	// Hide PIN from response (it's in the json:"-" tag, but clear it anyway)
	delivery.ProofPIN = nil

	return &DeliveryResponse{
		Delivery: delivery,
		Stops:    stops,
		Tracking: tracking,
	}, nil
}

// TrackDelivery allows public tracking by tracking code (limited info)
func (s *Service) TrackDelivery(ctx context.Context, trackingCode string) (*DeliveryResponse, error) {
	delivery, err := s.repo.GetDeliveryByTrackingCode(ctx, trackingCode)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("delivery not found", err)
		}
		return nil, err
	}

	tracking, _ := s.repo.GetTrackingByDeliveryID(ctx, delivery.ID)

	// Strip sensitive fields for public tracking
	delivery.ProofPIN = nil
	delivery.PickupPhone = ""
	delivery.RecipientPhone = ""
	delivery.DeclaredValue = nil

	return &DeliveryResponse{
		Delivery: delivery,
		Tracking: tracking,
	}, nil
}

// AcceptDelivery lets a driver accept a delivery order
func (s *Service) AcceptDelivery(ctx context.Context, deliveryID, driverID uuid.UUID) (*DeliveryResponse, error) {
	// Check driver doesn't have an active delivery
	active, err := s.repo.GetActiveDeliveryForDriver(ctx, driverID)
	if err != nil {
		return nil, err
	}
	if active != nil {
		return nil, common.NewConflictError("you already have an active delivery")
	}

	accepted, err := s.repo.AtomicAcceptDelivery(ctx, deliveryID, driverID)
	if err != nil {
		return nil, err
	}
	if !accepted {
		return nil, common.NewConflictError("delivery already accepted by another driver")
	}

	delivery, err := s.repo.GetDeliveryByID(ctx, deliveryID)
	if err != nil {
		return nil, err
	}

	// Record tracking event
	s.repo.AddTrackingEvent(ctx, &DeliveryTracking{
		ID:         uuid.New(),
		DeliveryID: deliveryID,
		DriverID:   driverID,
		Status:     "Driver accepted delivery",
		Timestamp:  time.Now(),
	})

	s.publishEvent("deliveries.accepted", "delivery.accepted", "delivery-service", map[string]interface{}{
		"delivery_id": deliveryID,
		"driver_id":   driverID,
		"sender_id":   delivery.SenderID,
		"accepted_at": time.Now(),
	})

	stops, _ := s.repo.GetStopsByDeliveryID(ctx, deliveryID)
	return &DeliveryResponse{Delivery: delivery, Stops: stops}, nil
}

// UpdateStatus moves a delivery through its lifecycle
func (s *Service) UpdateStatus(ctx context.Context, deliveryID, driverID uuid.UUID, newStatus DeliveryStatus) error {
	delivery, err := s.repo.GetDeliveryByID(ctx, deliveryID)
	if err != nil {
		return common.NewNotFoundError("delivery not found", err)
	}

	if delivery.DriverID == nil || *delivery.DriverID != driverID {
		return common.NewForbiddenError("not your delivery")
	}

	// Validate state transitions
	if !isValidTransition(delivery.Status, newStatus) {
		return common.NewBadRequestError(
			fmt.Sprintf("cannot transition from %s to %s", delivery.Status, newStatus), nil)
	}

	if err := s.repo.UpdateDeliveryStatus(ctx, deliveryID, newStatus, driverID); err != nil {
		return err
	}

	// Record tracking event
	s.repo.AddTrackingEvent(ctx, &DeliveryTracking{
		ID:         uuid.New(),
		DeliveryID: deliveryID,
		DriverID:   driverID,
		Status:     fmt.Sprintf("Status changed to %s", newStatus),
		Timestamp:  time.Now(),
	})

	return nil
}

// ConfirmPickup marks the package as picked up from sender
func (s *Service) ConfirmPickup(ctx context.Context, deliveryID, driverID uuid.UUID, req *ConfirmPickupRequest) error {
	delivery, err := s.repo.GetDeliveryByID(ctx, deliveryID)
	if err != nil {
		return common.NewNotFoundError("delivery not found", err)
	}

	if delivery.DriverID == nil || *delivery.DriverID != driverID {
		return common.NewForbiddenError("not your delivery")
	}

	if delivery.Status != DeliveryStatusPickingUp && delivery.Status != DeliveryStatusAccepted {
		return common.NewBadRequestError("delivery must be in picking_up or accepted status", nil)
	}

	if err := s.repo.UpdateDeliveryStatus(ctx, deliveryID, DeliveryStatusPickedUp, driverID); err != nil {
		return err
	}

	statusMsg := "Package picked up from sender"
	if req.Notes != nil {
		statusMsg += " - " + *req.Notes
	}

	s.repo.AddTrackingEvent(ctx, &DeliveryTracking{
		ID:         uuid.New(),
		DeliveryID: deliveryID,
		DriverID:   driverID,
		Status:     statusMsg,
		Timestamp:  time.Now(),
	})

	s.publishEvent("deliveries.picked_up", "delivery.picked_up", "delivery-service", map[string]interface{}{
		"delivery_id":  deliveryID,
		"driver_id":    driverID,
		"sender_id":    delivery.SenderID,
		"picked_up_at": time.Now(),
	})

	return nil
}

// ConfirmDelivery marks the delivery as complete with proof
func (s *Service) ConfirmDelivery(ctx context.Context, deliveryID, driverID uuid.UUID, req *ConfirmDeliveryRequest) error {
	delivery, err := s.repo.GetDeliveryByID(ctx, deliveryID)
	if err != nil {
		return common.NewNotFoundError("delivery not found", err)
	}

	if delivery.DriverID == nil || *delivery.DriverID != driverID {
		return common.NewForbiddenError("not your delivery")
	}

	// Validate proof
	if delivery.RequiresSignature && req.ProofType != ProofTypeSignature {
		return common.NewBadRequestError("this delivery requires a signature", nil)
	}

	if req.ProofType == ProofTypePIN {
		if req.PIN == nil || delivery.ProofPIN == nil || *req.PIN != *delivery.ProofPIN {
			return common.NewBadRequestError("incorrect delivery PIN", nil)
		}
	}

	if req.ProofType == ProofTypePhoto && req.PhotoURL == nil {
		return common.NewBadRequestError("photo proof is required", nil)
	}

	if req.ProofType == ProofTypeSignature && req.SignatureURL == nil {
		return common.NewBadRequestError("signature is required", nil)
	}

	// Calculate final fare (same as estimate for now)
	finalFare := delivery.EstimatedFare

	completed, err := s.repo.AtomicCompleteDelivery(ctx, deliveryID, driverID, req.ProofType, req.PhotoURL, req.SignatureURL, finalFare)
	if err != nil {
		return err
	}
	if !completed {
		return common.NewConflictError("delivery already completed or not in transit")
	}

	s.repo.AddTrackingEvent(ctx, &DeliveryTracking{
		ID:         uuid.New(),
		DeliveryID: deliveryID,
		DriverID:   driverID,
		Status:     fmt.Sprintf("Delivered - proof: %s", req.ProofType),
		Timestamp:  time.Now(),
	})

	s.publishEvent("deliveries.completed", "delivery.completed", "delivery-service", map[string]interface{}{
		"delivery_id":  deliveryID,
		"driver_id":    driverID,
		"sender_id":    delivery.SenderID,
		"fare_amount":  finalFare,
		"distance_km":  delivery.EstimatedDistance,
		"completed_at": time.Now(),
	})

	return nil
}

// CancelDelivery cancels a delivery order
func (s *Service) CancelDelivery(ctx context.Context, deliveryID, userID uuid.UUID, reason string) error {
	delivery, err := s.repo.GetDeliveryByID(ctx, deliveryID)
	if err != nil {
		return common.NewNotFoundError("delivery not found", err)
	}

	// Only sender can cancel, or driver before pickup
	isSender := delivery.SenderID == userID
	isDriver := delivery.DriverID != nil && *delivery.DriverID == userID

	if !isSender && !isDriver {
		return common.NewForbiddenError("not authorized to cancel")
	}

	// Can't cancel after pickup
	if isDriver && (delivery.Status == DeliveryStatusPickedUp ||
		delivery.Status == DeliveryStatusInTransit ||
		delivery.Status == DeliveryStatusArrived) {
		return common.NewBadRequestError("cannot cancel after package pickup - use return instead", nil)
	}

	if err := s.repo.CancelDelivery(ctx, deliveryID, reason); err != nil {
		return err
	}

	cancelledBy := "sender"
	if isDriver {
		cancelledBy = "driver"
	}

	s.publishEvent("deliveries.cancelled", "delivery.cancelled", "delivery-service", map[string]interface{}{
		"delivery_id":  deliveryID,
		"sender_id":    delivery.SenderID,
		"driver_id":    delivery.DriverID,
		"cancelled_by": cancelledBy,
		"reason":       reason,
		"cancelled_at": time.Now(),
	})

	return nil
}

// ReturnDelivery returns the package to sender (after pickup, recipient unavailable)
func (s *Service) ReturnDelivery(ctx context.Context, deliveryID, driverID uuid.UUID, reason string) error {
	delivery, err := s.repo.GetDeliveryByID(ctx, deliveryID)
	if err != nil {
		return common.NewNotFoundError("delivery not found", err)
	}

	if delivery.DriverID == nil || *delivery.DriverID != driverID {
		return common.NewForbiddenError("not your delivery")
	}

	// Must be in transit/arrived to return
	if delivery.Status != DeliveryStatusInTransit && delivery.Status != DeliveryStatusArrived {
		return common.NewBadRequestError("can only return packages in transit or arrived", nil)
	}

	if err := s.repo.ReturnDelivery(ctx, deliveryID, reason); err != nil {
		return err
	}

	s.repo.AddTrackingEvent(ctx, &DeliveryTracking{
		ID:         uuid.New(),
		DeliveryID: deliveryID,
		DriverID:   driverID,
		Status:     "Package being returned to sender: " + reason,
		Timestamp:  time.Now(),
	})

	return nil
}

// RateDelivery allows sender or driver to rate the delivery
func (s *Service) RateDelivery(ctx context.Context, deliveryID, userID uuid.UUID, req *RateDeliveryRequest) error {
	delivery, err := s.repo.GetDeliveryByID(ctx, deliveryID)
	if err != nil {
		return common.NewNotFoundError("delivery not found", err)
	}

	if delivery.Status != DeliveryStatusDelivered {
		return common.NewBadRequestError("can only rate completed deliveries", nil)
	}

	if delivery.SenderID == userID {
		return s.repo.RateDeliveryBySender(ctx, deliveryID, req.Rating, req.Feedback)
	}
	if delivery.DriverID != nil && *delivery.DriverID == userID {
		return s.repo.RateDeliveryByDriver(ctx, deliveryID, req.Rating, req.Feedback)
	}

	return common.NewForbiddenError("not part of this delivery")
}

// ========================================
// LISTING
// ========================================

// GetMyDeliveries lists deliveries for the authenticated sender
func (s *Service) GetMyDeliveries(ctx context.Context, senderID uuid.UUID, filters *DeliveryListFilters, limit, offset int) ([]*Delivery, int64, error) {
	return s.repo.GetDeliveriesBySender(ctx, senderID, filters, limit, offset)
}

// GetAvailableDeliveries lists deliveries for drivers to pick up
func (s *Service) GetAvailableDeliveries(ctx context.Context, latitude, longitude float64) ([]*Delivery, error) {
	return s.repo.GetAvailableDeliveries(ctx, latitude, longitude, 15.0) // 15km radius
}

// GetDriverDeliveries lists completed/active deliveries for a driver
func (s *Service) GetDriverDeliveries(ctx context.Context, driverID uuid.UUID, limit, offset int) ([]*Delivery, int64, error) {
	return s.repo.GetDriverDeliveries(ctx, driverID, limit, offset)
}

// GetActiveDelivery gets the driver's current active delivery
func (s *Service) GetActiveDelivery(ctx context.Context, driverID uuid.UUID) (*DeliveryResponse, error) {
	delivery, err := s.repo.GetActiveDeliveryForDriver(ctx, driverID)
	if err != nil {
		return nil, err
	}
	if delivery == nil {
		return nil, common.NewNotFoundError("no active delivery", nil)
	}

	stops, _ := s.repo.GetStopsByDeliveryID(ctx, delivery.ID)
	tracking, _ := s.repo.GetTrackingByDeliveryID(ctx, delivery.ID)
	delivery.ProofPIN = nil

	return &DeliveryResponse{Delivery: delivery, Stops: stops, Tracking: tracking}, nil
}

// GetStats returns delivery statistics for a sender
func (s *Service) GetStats(ctx context.Context, senderID uuid.UUID) (*DeliveryStats, error) {
	return s.repo.GetSenderStats(ctx, senderID)
}

// ========================================
// STOPS
// ========================================

// UpdateStopStatus updates an intermediate stop status
func (s *Service) UpdateStopStatus(ctx context.Context, deliveryID, stopID, driverID uuid.UUID, status string, photoURL *string) error {
	delivery, err := s.repo.GetDeliveryByID(ctx, deliveryID)
	if err != nil {
		return common.NewNotFoundError("delivery not found", err)
	}

	if delivery.DriverID == nil || *delivery.DriverID != driverID {
		return common.NewForbiddenError("not your delivery")
	}

	return s.repo.UpdateStopStatus(ctx, stopID, status, photoURL)
}

// ========================================
// HELPERS
// ========================================

// haversineDistance calculates distance between two points in kilometers
func haversineDistance(latitude1, longitude1, latitude2, longitude2 float64) float64 {
	const earthRadiusKm = 6371.0
	deltaLatitude := (latitude2 - latitude1) * math.Pi / 180
	deltaLongitude := (longitude2 - longitude1) * math.Pi / 180
	a := math.Sin(deltaLatitude/2)*math.Sin(deltaLatitude/2) +
		math.Cos(latitude1*math.Pi/180)*math.Cos(latitude2*math.Pi/180)*
			math.Sin(deltaLongitude/2)*math.Sin(deltaLongitude/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

// generateTrackingCode creates a human-friendly tracking code
func generateTrackingCode() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 10)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		code[i] = chars[n.Int64()]
	}
	// Format: DLV-XXXXX-XXXXX
	return fmt.Sprintf("DLV-%s-%s", string(code[:5]), string(code[5:]))
}

// generatePIN creates a 4-digit PIN for delivery confirmation
func generatePIN() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(10000))
	return fmt.Sprintf("%04d", n.Int64())
}

// isValidPackageSize validates the package size enum
func isValidPackageSize(size PackageSize) bool {
	switch size {
	case PackageSizeEnvelope, PackageSizeSmall, PackageSizeMedium, PackageSizeLarge, PackageSizeXLarge:
		return true
	}
	return false
}

// isValidPriority validates the delivery priority enum
func isValidPriority(p DeliveryPriority) bool {
	switch p {
	case DeliveryPriorityStandard, DeliveryPriorityExpress, DeliveryPriorityScheduled:
		return true
	}
	return false
}

// isValidTransition checks if a status transition is allowed
func isValidTransition(from, to DeliveryStatus) bool {
	validTransitions := map[DeliveryStatus][]DeliveryStatus{
		DeliveryStatusRequested: {DeliveryStatusAccepted, DeliveryStatusCancelled},
		DeliveryStatusAccepted:  {DeliveryStatusPickingUp, DeliveryStatusPickedUp, DeliveryStatusCancelled},
		DeliveryStatusPickingUp: {DeliveryStatusPickedUp, DeliveryStatusCancelled},
		DeliveryStatusPickedUp:  {DeliveryStatusInTransit},
		DeliveryStatusInTransit: {DeliveryStatusArrived, DeliveryStatusDelivered, DeliveryStatusReturned},
		DeliveryStatusArrived:   {DeliveryStatusDelivered, DeliveryStatusReturned, DeliveryStatusFailed},
	}

	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}
