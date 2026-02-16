package pool

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/internal/geo"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// MapsService interface for route calculations
type MapsService interface {
	GetRoute(ctx context.Context, origin, destination Location) (*RouteInfo, error)
	GetMultiStopRoute(ctx context.Context, stops []Location) (*MultiStopRouteInfo, error)
}

// RouteInfo represents basic route information
type RouteInfo struct {
	DistanceKm     float64
	DurationMinutes int
	Polyline       string
}

// MultiStopRouteInfo represents a route with multiple stops
type MultiStopRouteInfo struct {
	TotalDistanceKm     float64
	TotalDurationMinutes int
	Legs               []RouteLeg
	Polyline           string
}

// RouteLeg represents one leg of a multi-stop route
type RouteLeg struct {
	FromIndex      int
	ToIndex        int
	DistanceKm     float64
	DurationMinutes int
}

// Service handles pool ride business logic
type Service struct {
	repo        *Repository
	mapsService MapsService
	config      *ServiceConfig
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	DefaultMaxWaitMinutes   int
	DefaultMaxDetourPercent float64
	DefaultDiscountPercent  float64
	MaxPassengersPerRide    int
	H3Resolution            int
	MinMatchScore           float64
}

// DefaultServiceConfig returns default configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		DefaultMaxWaitMinutes:   5,
		DefaultMaxDetourPercent: 25,
		DefaultDiscountPercent:  25,
		MaxPassengersPerRide:    4,
		H3Resolution:            7, // ~1.2 km edge length
		MinMatchScore:           0.5,
	}
}

// NewService creates a new pool service
func NewService(repo *Repository, mapsService MapsService, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	return &Service{
		repo:        repo,
		mapsService: mapsService,
		config:      config,
	}
}

// ========================================
// POOL RIDE REQUESTS
// ========================================

// RequestPoolRide requests a pool ride for a rider
func (s *Service) RequestPoolRide(ctx context.Context, riderID uuid.UUID, req *RequestPoolRideRequest) (*RequestPoolRideResponse, error) {
	// Check if rider already has an active pool request
	existing, _ := s.repo.GetActivePassengerForRider(ctx, riderID)
	if existing != nil {
		return nil, common.NewBadRequestError("you already have an active pool ride", nil)
	}

	// Calculate direct route for pricing comparison
	directRoute, err := s.mapsService.GetRoute(ctx, req.PickupLocation, req.DropoffLocation)
	if err != nil {
		return nil, common.NewInternalServerError("failed to calculate route")
	}

	// Get pool config
	config, _ := s.repo.GetPoolConfig(ctx, nil)
	if config == nil {
		config = s.getDefaultConfig()
	}

	// Calculate original fare (if they rode alone)
	originalFare := s.calculateFare(directRoute.DistanceKm, directRoute.DurationMinutes, config)

	// Find matching pool rides
	matchCandidates, err := s.findMatchingPools(ctx, req, config)
	if err != nil {
		logger.Warn("failed to find matching pools", zap.Error(err))
	}

	var poolRideID *uuid.UUID
	var matchFound bool
	var bestMatch *PoolMatchCandidate
	poolFare := originalFare * (1 - config.DiscountPercent/100) // Default discount

	maxWaitMinutes := req.MaxWaitMinutes
	if maxWaitMinutes == 0 {
		maxWaitMinutes = config.MaxWaitMinutes
	}

	passengerCount := req.PassengerCount
	if passengerCount == 0 {
		passengerCount = 1
	}

	estimatedPickup := time.Now().Add(time.Duration(maxWaitMinutes) * time.Minute)
	estimatedDropoff := estimatedPickup.Add(time.Duration(directRoute.DurationMinutes) * time.Minute)

	if len(matchCandidates) > 0 {
		// Found matching pools - select the best one
		bestMatch = matchCandidates[0]
		for _, candidate := range matchCandidates {
			if candidate.MatchScore.Score > bestMatch.MatchScore.Score {
				bestMatch = candidate
			}
		}

		if bestMatch.MatchScore.Score >= config.MinMatchScore {
			matchFound = true
			poolRideID = &bestMatch.PoolRideID
			poolFare = bestMatch.PoolFare
			estimatedPickup = bestMatch.EstimatedPickup
			estimatedDropoff = bestMatch.EstimatedDropoff
		}
	}

	// Create new pool ride if no match found
	if !matchFound {
		newPool := s.createNewPoolRide(req, directRoute, config, maxWaitMinutes)
		if err := s.repo.CreatePoolRide(ctx, newPool); err != nil {
			return nil, common.NewInternalServerError("failed to create pool ride")
		}
		poolRideID = &newPool.ID
	}

	// Calculate savings
	savingsPercent := (1 - poolFare/originalFare) * 100
	if savingsPercent < 0 {
		savingsPercent = 0
	}

	// Create pool passenger record
	passenger := &PoolPassenger{
		ID:              uuid.New(),
		PoolRideID:      *poolRideID,
		RiderID:         riderID,
		Status:          PassengerStatusPending,
		PickupLocation:  req.PickupLocation,
		DropoffLocation: req.DropoffLocation,
		PickupAddress:   req.PickupAddress,
		DropoffAddress:  req.DropoffAddress,
		DirectDistance:  directRoute.DistanceKm,
		DirectDuration:  directRoute.DurationMinutes,
		OriginalFare:    originalFare,
		PoolFare:        poolFare,
		SavingsPercent:  savingsPercent,
		EstimatedPickup: estimatedPickup,
		EstimatedDropoff: estimatedDropoff,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.repo.CreatePoolPassenger(ctx, passenger); err != nil {
		return nil, common.NewInternalServerError("failed to create pool passenger")
	}

	// Update pool passenger count if joining existing pool
	if matchFound {
		_ = s.repo.IncrementPassengerCount(ctx, *poolRideID, passengerCount)
		// Re-optimize route
		go s.reoptimizeRoute(context.Background(), *poolRideID)
	}

	logger.Info("Pool ride requested",
		zap.String("rider_id", riderID.String()),
		zap.String("pool_ride_id", poolRideID.String()),
		zap.Bool("match_found", matchFound),
		zap.Float64("savings_percent", savingsPercent),
	)

	response := &RequestPoolRideResponse{
		PoolPassengerID:   passenger.ID,
		PoolRideID:        poolRideID,
		Status:            passenger.Status,
		MatchFound:        matchFound,
		OriginalFare:      originalFare,
		PoolFare:          poolFare,
		SavingsPercent:    savingsPercent,
		EstimatedPickup:   estimatedPickup,
		EstimatedDropoff:  estimatedDropoff,
		MatchDeadline:     time.Now().Add(time.Duration(maxWaitMinutes) * time.Minute),
		Message:           s.getResponseMessage(matchFound, savingsPercent),
	}

	if bestMatch != nil {
		response.MatchScore = &bestMatch.MatchScore
		response.CurrentPassengers = bestMatch.CurrentPassengers
		response.MaxPassengers = bestMatch.MaxPassengers
	}

	return response, nil
}

// ConfirmPoolRide confirms or declines joining a pool
func (s *Service) ConfirmPoolRide(ctx context.Context, riderID uuid.UUID, req *ConfirmPoolRequest) error {
	passenger, err := s.repo.GetPoolPassenger(ctx, req.PoolPassengerID)
	if err != nil {
		return common.NewNotFoundError("pool passenger not found", err)
	}

	if passenger.RiderID != riderID {
		return common.NewForbiddenError("not authorized to confirm this pool")
	}

	if passenger.Status != PassengerStatusPending {
		return common.NewBadRequestError("pool ride is not pending confirmation", nil)
	}

	if req.Accept {
		if err := s.repo.UpdatePassengerStatus(ctx, req.PoolPassengerID, PassengerStatusConfirmed); err != nil {
			return common.NewInternalServerError("failed to confirm pool")
		}

		logger.Info("Pool ride confirmed",
			zap.String("passenger_id", req.PoolPassengerID.String()),
			zap.String("rider_id", riderID.String()),
		)
	} else {
		if err := s.repo.UpdatePassengerStatus(ctx, req.PoolPassengerID, PassengerStatusCancelled); err != nil {
			return common.NewInternalServerError("failed to cancel pool")
		}

		// Decrement passenger count
		_ = s.repo.IncrementPassengerCount(ctx, passenger.PoolRideID, -1)

		logger.Info("Pool ride declined",
			zap.String("passenger_id", req.PoolPassengerID.String()),
			zap.String("rider_id", riderID.String()),
		)
	}

	return nil
}

// GetPoolStatus gets the status of a pool ride for a rider
func (s *Service) GetPoolStatus(ctx context.Context, riderID uuid.UUID, poolRideID uuid.UUID) (*GetPoolStatusResponse, error) {
	poolRide, err := s.repo.GetPoolRide(ctx, poolRideID)
	if err != nil {
		return nil, common.NewNotFoundError("pool ride not found", err)
	}

	passenger, err := s.repo.GetPoolPassengerByRider(ctx, riderID, poolRideID)
	if err != nil {
		return nil, common.NewForbiddenError("not authorized to view this pool")
	}

	passengers, _ := s.repo.GetPassengersForPool(ctx, poolRideID)

	// Build other passengers info (privacy-safe)
	var otherPassengers []PassengerInfo
	for _, p := range passengers {
		if p.RiderID != riderID {
			otherPassengers = append(otherPassengers, PassengerInfo{
				FirstNameInitial: "P", // In production, get from user service
				Rating:           nil,
				PickupStop:       s.getStopIndex(poolRide.OptimizedRoute, p.ID, RouteStopPickup),
				DropoffStop:      s.getStopIndex(poolRide.OptimizedRoute, p.ID, RouteStopDropoff),
			})
		}
	}

	return &GetPoolStatusResponse{
		PoolRide:        poolRide,
		MyStatus:        passenger.Status,
		OtherPassengers: otherPassengers,
		RouteStops:      poolRide.OptimizedRoute,
		EstimatedArrival: passenger.EstimatedDropoff,
	}, nil
}

// CancelPoolRide cancels a pool ride for a rider
func (s *Service) CancelPoolRide(ctx context.Context, riderID uuid.UUID, poolPassengerID uuid.UUID) error {
	passenger, err := s.repo.GetPoolPassenger(ctx, poolPassengerID)
	if err != nil {
		return common.NewNotFoundError("pool passenger not found", err)
	}

	if passenger.RiderID != riderID {
		return common.NewForbiddenError("not authorized to cancel this pool")
	}

	if passenger.Status == PassengerStatusPickedUp {
		return common.NewBadRequestError("cannot cancel after pickup", nil)
	}

	if passenger.Status == PassengerStatusDroppedOff || passenger.Status == PassengerStatusCancelled {
		return common.NewBadRequestError("pool ride is already completed or cancelled", nil)
	}

	if err := s.repo.UpdatePassengerStatus(ctx, poolPassengerID, PassengerStatusCancelled); err != nil {
		return common.NewInternalServerError("failed to cancel pool")
	}

	// Decrement passenger count
	_ = s.repo.IncrementPassengerCount(ctx, passenger.PoolRideID, -1)

	// Re-optimize route
	go s.reoptimizeRoute(context.Background(), passenger.PoolRideID)

	logger.Info("Pool ride cancelled",
		zap.String("passenger_id", poolPassengerID.String()),
		zap.String("rider_id", riderID.String()),
	)

	return nil
}

// ========================================
// DRIVER OPERATIONS
// ========================================

// GetDriverPoolRide gets the current pool ride for a driver
func (s *Service) GetDriverPoolRide(ctx context.Context, driverID uuid.UUID) (*DriverPoolRideInfo, error) {
	poolRide, err := s.repo.GetDriverActivePool(ctx, driverID)
	if err != nil {
		return nil, common.NewNotFoundError("no active pool ride", err)
	}

	passengersPtr, _ := s.repo.GetPassengersForPool(ctx, poolRide.ID)

	// Convert pointers to values
	passengers := make([]PoolPassenger, len(passengersPtr))
	for i, p := range passengersPtr {
		passengers[i] = *p
	}

	// Calculate total fare and driver earnings
	var totalFare float64
	for _, p := range passengers {
		totalFare += p.PoolFare
	}
	driverEarnings := totalFare * 0.75 // 75% to driver

	// Find next stop
	var nextStop *RouteStop
	for _, stop := range poolRide.OptimizedRoute {
		if stop.ActualArrival == nil {
			nextStop = &stop
			break
		}
	}

	return &DriverPoolRideInfo{
		PoolRide:       poolRide,
		Passengers:     passengers,
		RouteStops:     poolRide.OptimizedRoute,
		NextStop:       nextStop,
		TotalFare:      totalFare,
		DriverEarnings: driverEarnings,
	}, nil
}

// PickupPassenger marks a passenger as picked up
func (s *Service) PickupPassenger(ctx context.Context, driverID uuid.UUID, passengerID uuid.UUID) error {
	// Verify driver owns this pool
	poolRide, err := s.repo.GetDriverActivePool(ctx, driverID)
	if err != nil {
		return common.NewForbiddenError("no active pool ride")
	}

	passenger, err := s.repo.GetPoolPassenger(ctx, passengerID)
	if err != nil {
		return common.NewNotFoundError("passenger not found", err)
	}

	if passenger.PoolRideID != poolRide.ID {
		return common.NewForbiddenError("passenger not in your pool")
	}

	if passenger.Status != PassengerStatusConfirmed {
		return common.NewBadRequestError("passenger is not confirmed", nil)
	}

	if err := s.repo.UpdatePassengerPickedUp(ctx, passengerID); err != nil {
		return common.NewInternalServerError("failed to update passenger")
	}

	logger.Info("Passenger picked up",
		zap.String("passenger_id", passengerID.String()),
		zap.String("driver_id", driverID.String()),
	)

	return nil
}

// DropoffPassenger marks a passenger as dropped off
func (s *Service) DropoffPassenger(ctx context.Context, driverID uuid.UUID, passengerID uuid.UUID) error {
	// Verify driver owns this pool
	poolRide, err := s.repo.GetDriverActivePool(ctx, driverID)
	if err != nil {
		return common.NewForbiddenError("no active pool ride")
	}

	passenger, err := s.repo.GetPoolPassenger(ctx, passengerID)
	if err != nil {
		return common.NewNotFoundError("passenger not found", err)
	}

	if passenger.PoolRideID != poolRide.ID {
		return common.NewForbiddenError("passenger not in your pool")
	}

	if passenger.Status != PassengerStatusPickedUp {
		return common.NewBadRequestError("passenger is not picked up", nil)
	}

	if err := s.repo.UpdatePassengerDroppedOff(ctx, passengerID); err != nil {
		return common.NewInternalServerError("failed to update passenger")
	}

	// Check if all passengers are dropped off
	passengers, _ := s.repo.GetPassengersForPool(ctx, poolRide.ID)
	allDroppedOff := true
	for _, p := range passengers {
		if p.Status != PassengerStatusDroppedOff && p.Status != PassengerStatusCancelled && p.Status != PassengerStatusNoShow {
			allDroppedOff = false
			break
		}
	}

	if allDroppedOff {
		_ = s.repo.CompletePoolRide(ctx, poolRide.ID)
		logger.Info("Pool ride completed",
			zap.String("pool_ride_id", poolRide.ID.String()),
		)
	}

	logger.Info("Passenger dropped off",
		zap.String("passenger_id", passengerID.String()),
		zap.String("driver_id", driverID.String()),
	)

	return nil
}

// StartPoolRide starts the pool ride
func (s *Service) StartPoolRide(ctx context.Context, driverID uuid.UUID, poolRideID uuid.UUID) error {
	poolRide, err := s.repo.GetPoolRide(ctx, poolRideID)
	if err != nil {
		return common.NewNotFoundError("pool ride not found", err)
	}

	if poolRide.DriverID == nil || *poolRide.DriverID != driverID {
		return common.NewForbiddenError("not authorized to start this pool")
	}

	if poolRide.Status != PoolStatusConfirmed {
		return common.NewBadRequestError("pool ride is not confirmed", nil)
	}

	if err := s.repo.StartPoolRide(ctx, poolRideID); err != nil {
		return common.NewInternalServerError("failed to start pool ride")
	}

	logger.Info("Pool ride started",
		zap.String("pool_ride_id", poolRideID.String()),
		zap.String("driver_id", driverID.String()),
	)

	return nil
}

// ========================================
// MATCHING ALGORITHM
// ========================================

// findMatchingPools finds existing pools that match the new request
func (s *Service) findMatchingPools(ctx context.Context, req *RequestPoolRideRequest, config *PoolConfig) ([]*PoolMatchCandidate, error) {
	// Get neighboring H3 cells for wider search (2 rings = ~3.6 km radius)
	h3Indexes := geo.GetKRingCellStrings(req.PickupLocation.Latitude, req.PickupLocation.Longitude, s.config.H3Resolution, 2)

	passengerCount := req.PassengerCount
	if passengerCount == 0 {
		passengerCount = 1
	}

	// Get nearby active pools
	pools, err := s.repo.GetNearbyPoolRides(ctx, h3Indexes, passengerCount)
	if err != nil {
		return nil, err
	}

	var candidates []*PoolMatchCandidate
	for _, pool := range pools {
		// Calculate match score
		matchScore := s.calculateMatchScore(ctx, pool, req, config)
		if matchScore.Score < config.MinMatchScore {
			continue
		}

		// Calculate pool fare
		poolFare := s.calculatePoolFare(pool, req, matchScore, config)

		// Estimate times based on current route
		estimatedPickup := s.estimatePickupTime(pool, req.PickupLocation)
		estimatedDropoff := s.estimateDropoffTime(pool, req.DropoffLocation, estimatedPickup)

		candidates = append(candidates, &PoolMatchCandidate{
			PoolRideID:        pool.ID,
			MatchScore:        matchScore,
			CurrentPassengers: pool.CurrentPassengers,
			MaxPassengers:     pool.MaxPassengers,
			EstimatedPickup:   estimatedPickup,
			EstimatedDropoff:  estimatedDropoff,
			PoolFare:          poolFare,
			SavingsPercent:    matchScore.CostSavingsPercent,
		})
	}

	return candidates, nil
}

// calculateMatchScore calculates how well a new request matches an existing pool
func (s *Service) calculateMatchScore(ctx context.Context, pool *PoolRide, req *RequestPoolRideRequest, config *PoolConfig) RouteMatchScore {
	// Calculate current route distance
	currentDistance := pool.TotalDistance
	currentDuration := pool.TotalDuration

	// Estimate new route with added stops
	newRoute, err := s.mapsService.GetMultiStopRoute(ctx, s.buildRouteStops(pool, req))
	if err != nil {
		return RouteMatchScore{Score: 0}
	}

	detourKm := newRoute.TotalDistanceKm - currentDistance
	detourMinutes := float64(newRoute.TotalDurationMinutes - currentDuration)
	detourPercent := (detourKm / currentDistance) * 100

	// Check if detour is acceptable
	if detourPercent > config.MaxDetourPercent || detourMinutes > float64(config.MaxDetourMinutes) {
		return RouteMatchScore{Score: 0}
	}

	// Calculate match score (higher is better)
	// Based on: detour efficiency, route alignment, time added
	routeAlignmentScore := 1.0 - (detourPercent / config.MaxDetourPercent)
	timeScore := 1.0 - (detourMinutes / float64(config.MaxDetourMinutes))

	// Direction alignment (are they going the same way?)
	directionScore := s.calculateDirectionAlignment(pool, req)

	// Combined score (weighted)
	score := (routeAlignmentScore*0.4 + timeScore*0.3 + directionScore*0.3)

	// Calculate cost savings
	costSavingsPercent := config.DiscountPercent * score // Better match = more savings

	return RouteMatchScore{
		Score:              score,
		DetourMinutes:      detourMinutes,
		DetourKm:           detourKm,
		DetourPercent:      detourPercent,
		CostSavings:        0, // Will be calculated with actual fare
		CostSavingsPercent: costSavingsPercent,
	}
}

// calculateDirectionAlignment checks if routes are going in similar direction
func (s *Service) calculateDirectionAlignment(pool *PoolRide, req *RequestPoolRideRequest) float64 {
	// Calculate bearing from pickup to dropoff for both routes
	poolBearing := s.calculateBearing(pool.CenterLatitude, pool.CenterLongitude,
		req.DropoffLocation.Latitude, req.DropoffLocation.Longitude)
	reqBearing := s.calculateBearing(req.PickupLocation.Latitude, req.PickupLocation.Longitude,
		req.DropoffLocation.Latitude, req.DropoffLocation.Longitude)

	// Calculate angle difference
	diff := math.Abs(poolBearing - reqBearing)
	if diff > 180 {
		diff = 360 - diff
	}

	// Score: 0 for opposite direction (180°), 1 for same direction (0°)
	return 1.0 - (diff / 180.0)
}

// calculateBearing calculates bearing between two points
func (s *Service) calculateBearing(latitude1, longitude1, latitude2, longitude2 float64) float64 {
	latitude1Rad := latitude1 * math.Pi / 180
	latitude2Rad := latitude2 * math.Pi / 180
	deltaLongitude := (longitude2 - longitude1) * math.Pi / 180

	y := math.Sin(deltaLongitude) * math.Cos(latitude2Rad)
	x := math.Cos(latitude1Rad)*math.Sin(latitude2Rad) - math.Sin(latitude1Rad)*math.Cos(latitude2Rad)*math.Cos(deltaLongitude)

	bearing := math.Atan2(y, x) * 180 / math.Pi
	return math.Mod(bearing+360, 360)
}

// buildRouteStops builds the stop list for route calculation
func (s *Service) buildRouteStops(pool *PoolRide, req *RequestPoolRideRequest) []Location {
	// Start with existing stops and add new pickup/dropoff
	var stops []Location

	// Add existing stops
	for _, stop := range pool.OptimizedRoute {
		stops = append(stops, stop.Location)
	}

	// Add new pickup and dropoff (will be optimized later)
	stops = append(stops, req.PickupLocation, req.DropoffLocation)

	return stops
}

// ========================================
// HELPER FUNCTIONS
// ========================================

func (s *Service) calculateFare(distanceKm float64, durationMinutes int, config *PoolConfig) float64 {
	baseFare := 2.0 // Base fare
	perKmRate := 1.5
	perMinuteRate := 0.25

	fare := baseFare + (distanceKm * perKmRate) + (float64(durationMinutes) * perMinuteRate)
	return math.Round(fare*100) / 100
}

func (s *Service) calculatePoolFare(pool *PoolRide, req *RequestPoolRideRequest, matchScore RouteMatchScore, config *PoolConfig) float64 {
	// Get direct route fare
	directRoute, _ := s.mapsService.GetRoute(context.Background(), req.PickupLocation, req.DropoffLocation)
	if directRoute == nil {
		return 0
	}

	originalFare := s.calculateFare(directRoute.DistanceKm, directRoute.DurationMinutes, config)

	// Apply discount based on match score
	discount := config.DiscountPercent * matchScore.Score / 100
	poolFare := originalFare * (1 - discount)

	return math.Round(poolFare*100) / 100
}

func (s *Service) createNewPoolRide(req *RequestPoolRideRequest, route *RouteInfo, config *PoolConfig, maxWaitMinutes int) *PoolRide {
	h3Index := geo.LatLngToCell(req.PickupLocation.Latitude, req.PickupLocation.Longitude, s.config.H3Resolution).String()

	return &PoolRide{
		ID:                uuid.New(),
		Status:            PoolStatusMatching,
		MaxPassengers:     config.MaxPassengersPerRide,
		CurrentPassengers: 1,
		TotalDistance:     route.DistanceKm,
		TotalDuration:     route.DurationMinutes,
		CenterLatitude:         req.PickupLocation.Latitude,
		CenterLongitude:         req.PickupLocation.Longitude,
		RadiusKm:          config.MatchRadiusKm,
		H3Index:           h3Index,
		BaseFare:          2.0,
		PerKmRate:         1.5,
		PerMinuteRate:     0.25,
		MatchDeadline:     time.Now().Add(time.Duration(maxWaitMinutes) * time.Minute),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

func (s *Service) getDefaultConfig() *PoolConfig {
	return &PoolConfig{
		MaxDetourPercent:      s.config.DefaultMaxDetourPercent,
		MaxDetourMinutes:      15,
		MaxWaitMinutes:        s.config.DefaultMaxWaitMinutes,
		MaxPassengersPerRide:  s.config.MaxPassengersPerRide,
		MinMatchScore:         s.config.MinMatchScore,
		MatchRadiusKm:         3.0,
		H3Resolution:          s.config.H3Resolution,
		DiscountPercent:       s.config.DefaultDiscountPercent,
		MinSavingsPercent:     15,
		AllowDifferentDropoffs: true,
		AllowMidRoutePickup:   true,
		IsActive:              true,
	}
}

func (s *Service) getResponseMessage(matchFound bool, savingsPercent float64) string {
	if matchFound {
		return "Great! We found a pool match. You'll save " + formatPercent(savingsPercent) + " on this ride."
	}
	return "We're looking for other riders heading your way. You'll be notified when we find a match."
}

func formatPercent(p float64) string {
	return string(rune(int(p))) + "%"
}

func (s *Service) estimatePickupTime(pool *PoolRide, pickup Location) time.Time {
	// Simplified: estimate based on current route progress
	// In production, this would use actual ETA calculation
	return time.Now().Add(5 * time.Minute)
}

func (s *Service) estimateDropoffTime(pool *PoolRide, dropoff Location, afterPickup time.Time) time.Time {
	// Simplified estimate
	return afterPickup.Add(time.Duration(pool.TotalDuration) * time.Minute)
}

func (s *Service) getStopIndex(route []RouteStop, passengerID uuid.UUID, stopType RouteStopType) int {
	for i, stop := range route {
		if stop.PoolPassengerID == passengerID && stop.Type == stopType {
			return i + 1
		}
	}
	return 0
}

func (s *Service) reoptimizeRoute(ctx context.Context, poolRideID uuid.UUID) {
	// In production, this would:
	// 1. Get all active passengers
	// 2. Run TSP/VRP optimization
	// 3. Update the optimized route
	// 4. Notify affected passengers of new ETAs

	passengers, err := s.repo.GetPassengersForPool(ctx, poolRideID)
	if err != nil || len(passengers) == 0 {
		return
	}

	// Build stops from passengers
	var stops []Location
	for _, p := range passengers {
		if p.Status != PassengerStatusCancelled && p.Status != PassengerStatusDroppedOff {
			stops = append(stops, p.PickupLocation, p.DropoffLocation)
		}
	}

	if len(stops) < 2 {
		return
	}

	// Calculate optimized route
	routeInfo, err := s.mapsService.GetMultiStopRoute(ctx, stops)
	if err != nil {
		logger.Error("failed to reoptimize route", zap.Error(err))
		return
	}

	// Update route in database
	var routeStops []RouteStop
	for i, stop := range stops {
		stopType := RouteStopPickup
		if i%2 == 1 {
			stopType = RouteStopDropoff
		}
		passengerIdx := i / 2
		if passengerIdx < len(passengers) {
			routeStops = append(routeStops, RouteStop{
				ID:              uuid.New(),
				PoolPassengerID: passengers[passengerIdx].ID,
				Type:            stopType,
				Location:        stop,
				SequenceOrder:   i + 1,
			})
		}
	}

	_ = s.repo.UpdatePoolRoute(ctx, poolRideID, routeStops, routeInfo.TotalDistanceKm, routeInfo.TotalDurationMinutes)

	logger.Info("Pool route reoptimized",
		zap.String("pool_ride_id", poolRideID.String()),
		zap.Float64("total_distance_km", routeInfo.TotalDistanceKm),
		zap.Int("total_duration_min", routeInfo.TotalDurationMinutes),
	)
}

// GetPoolStats gets pool ride statistics
func (s *Service) GetPoolStats(ctx context.Context) (*PoolStatsResponse, error) {
	return s.repo.GetPoolStats(ctx)
}

// CleanupExpiredPools cleans up expired pool rides
func (s *Service) CleanupExpiredPools(ctx context.Context) (int64, error) {
	count, err := s.repo.CancelExpiredPoolRides(ctx)
	if err != nil {
		return 0, err
	}

	if count > 0 {
		logger.Info("Cleaned up expired pool rides", zap.Int64("count", count))
	}

	return count, nil
}
