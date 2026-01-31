package geo

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/logger"
	redisClient "github.com/richxcame/ride-hailing/pkg/redis"
	"go.uber.org/zap"
)

const (
	driverLocationPrefix = "driver:location:"
	driverLocationTTL    = 5 * time.Minute
	driverGeoIndexKey    = "drivers:geo:index" // Redis GEO key for all active drivers
	driverStatusPrefix   = "driver:status:"
	searchRadiusKm       = 10.0 // Search radius in kilometers

	// H3-based Redis keys
	h3CellDriversPrefix = "h3:drivers:" // Set of driver IDs per H3 cell (resolution 9)
	h3CellDriversTTL    = 5 * time.Minute
	h3SurgePrefix       = "h3:surge:"  // Surge data per H3 cell (resolution 8)
	h3DemandPrefix      = "h3:demand:" // Demand counters per H3 cell (resolution 7)
)

// DriverLocation represents a driver's location
type DriverLocation struct {
	DriverID  uuid.UUID `json:"driver_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	H3Cell    string    `json:"h3_cell"`            // H3 cell at matching resolution
	Heading   float64   `json:"heading,omitempty"`   // Direction the driver is facing (0-360)
	Speed     float64   `json:"speed,omitempty"`     // Speed in km/h
	Timestamp time.Time `json:"timestamp"`
}

// SurgeInfo represents surge pricing data for a zone
type SurgeInfo struct {
	H3Cell          string    `json:"h3_cell"`
	SurgeMultiplier float64   `json:"surge_multiplier"`
	DemandCount     int       `json:"demand_count"`
	SupplyCount     int       `json:"supply_count"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// DemandInfo represents demand data for a zone
type DemandInfo struct {
	H3Cell       string  `json:"h3_cell"`
	RequestCount int     `json:"request_count"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
}

// Service handles geolocation business logic
type Service struct {
	redis          redisClient.ClientInterface
	locationBuffer *LocationBuffer
	etaTracker     *ETATracker
}

// NewService creates a new geo service
func NewService(redis redisClient.ClientInterface) *Service {
	return &Service{redis: redis}
}

// SetLocationBuffer enables batched location writes.
func (s *Service) SetLocationBuffer(lb *LocationBuffer) {
	s.locationBuffer = lb
}

// SetETATracker enables real-time ETA updates during active rides.
func (s *Service) SetETATracker(tracker *ETATracker) {
	s.etaTracker = tracker
}

// UpdateDriverLocation updates a driver's current location with H3 indexing
func (s *Service) UpdateDriverLocation(ctx context.Context, driverID uuid.UUID, latitude, longitude float64) error {
	return s.UpdateDriverLocationFull(ctx, driverID, latitude, longitude, 0, 0)
}

// UpdateDriverLocationFull updates a driver's location with heading and speed.
// When a LocationBuffer is configured, updates are batched for efficiency.
func (s *Service) UpdateDriverLocationFull(ctx context.Context, driverID uuid.UUID, latitude, longitude, heading, speed float64) error {
	h3Cell := GetMatchingCell(latitude, longitude)

	// Trigger real-time ETA update asynchronously
	if s.etaTracker != nil {
		go s.etaTracker.OnDriverLocationUpdate(ctx, driverID, latitude, longitude, heading, speed)
	}

	// Use buffered pipeline when available (high-throughput path)
	if s.locationBuffer != nil {
		s.locationBuffer.Enqueue(LocationUpdate{
			DriverID:  driverID,
			Latitude:  latitude,
			Longitude: longitude,
			Heading:   heading,
			Speed:     speed,
			H3Cell:    h3Cell,
			Timestamp: time.Now(),
		})
		return nil
	}

	// Fallback: synchronous write
	location := &DriverLocation{
		DriverID:  driverID,
		Latitude:  latitude,
		Longitude: longitude,
		H3Cell:    h3Cell,
		Heading:   heading,
		Speed:     speed,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(location)
	if err != nil {
		return common.NewInternalServerError("failed to marshal location data")
	}

	key := fmt.Sprintf("%s%s", driverLocationPrefix, driverID.String())
	if err := s.redis.SetWithExpiration(ctx, key, data, driverLocationTTL); err != nil {
		return common.NewInternalServerError("failed to update driver location")
	}

	if err := s.redis.GeoAdd(ctx, driverGeoIndexKey, longitude, latitude, driverID.String()); err != nil {
		return common.NewInternalServerError("failed to add driver to geo index")
	}

	s.updateDriverH3Cell(ctx, driverID, h3Cell)

	return nil
}

// updateDriverH3Cell manages the H3 cell-based driver index
func (s *Service) updateDriverH3Cell(ctx context.Context, driverID uuid.UUID, newCell string) {
	driverIDStr := driverID.String()
	prevCellKey := fmt.Sprintf("driver:h3cell:%s", driverIDStr)

	// Get previous cell
	prevCell, err := s.redis.GetString(ctx, prevCellKey)
	if err == nil && prevCell != "" && prevCell != newCell {
		// Remove from old cell set
		oldSetKey := fmt.Sprintf("%s%s:%s", h3CellDriversPrefix, prevCell, driverIDStr)
		s.redis.Delete(ctx, oldSetKey)
	}

	// Store current cell mapping
	s.redis.SetWithExpiration(ctx, prevCellKey, []byte(newCell), driverLocationTTL)

	// Add to new cell set
	newSetKey := fmt.Sprintf("%s%s:%s", h3CellDriversPrefix, newCell, driverIDStr)
	s.redis.SetWithExpiration(ctx, newSetKey, []byte(driverIDStr), h3CellDriversTTL)
}

// GetDriverLocation retrieves a driver's current location
func (s *Service) GetDriverLocation(ctx context.Context, driverID uuid.UUID) (*DriverLocation, error) {
	key := fmt.Sprintf("%s%s", driverLocationPrefix, driverID.String())
	data, err := s.redis.GetString(ctx, key)
	if err != nil {
		return nil, common.NewNotFoundError("driver location not found", nil)
	}

	var location DriverLocation
	if err := json.Unmarshal([]byte(data), &location); err != nil {
		return nil, common.NewInternalServerError("failed to unmarshal location data")
	}

	return &location, nil
}

// FindNearbyDrivers finds drivers near a given location using Redis GEO,
// sorted by distance (closest first).
func (s *Service) FindNearbyDrivers(ctx context.Context, latitude, longitude float64, maxDrivers int) ([]*DriverLocation, error) {
	// Use Redis GEORADIUS to find nearby drivers
	driverIDs, err := s.redis.GeoRadius(ctx, driverGeoIndexKey, longitude, latitude, searchRadiusKm, maxDrivers*2)
	if err != nil {
		return nil, common.NewInternalErrorWithError("failed to search nearby drivers", err)
	}

	if len(driverIDs) == 0 {
		return []*DriverLocation{}, nil
	}

	// Get detailed location data for each driver and calculate distance
	type driverWithDistance struct {
		location *DriverLocation
		distance float64
	}

	var drivers []driverWithDistance
	for _, driverIDStr := range driverIDs {
		driverID, err := uuid.Parse(driverIDStr)
		if err != nil {
			continue
		}

		location, err := s.GetDriverLocation(ctx, driverID)
		if err != nil {
			continue
		}

		distance := s.CalculateDistance(latitude, longitude, location.Latitude, location.Longitude)
		if distance <= searchRadiusKm {
			drivers = append(drivers, driverWithDistance{location: location, distance: distance})
		}
	}

	// Sort by distance (closest first)
	sort.Slice(drivers, func(i, j int) bool {
		return drivers[i].distance < drivers[j].distance
	})

	// Limit results
	locations := make([]*DriverLocation, 0, maxDrivers)
	for i, d := range drivers {
		if i >= maxDrivers {
			break
		}
		locations = append(locations, d.location)
	}

	return locations, nil
}

// SetDriverStatus sets driver's availability status
func (s *Service) SetDriverStatus(ctx context.Context, driverID uuid.UUID, status string) error {
	key := fmt.Sprintf("%s%s", driverStatusPrefix, driverID.String())
	data := map[string]interface{}{
		"status":    status,
		"timestamp": time.Now(),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return common.NewInternalErrorWithError("failed to marshal status", err)
	}

	if err := s.redis.SetWithExpiration(ctx, key, jsonData, driverLocationTTL); err != nil {
		return common.NewInternalErrorWithError("failed to set driver status", err)
	}

	// If driver goes offline, remove from all indexes
	if status == "offline" {
		s.redis.GeoRemove(ctx, driverGeoIndexKey, driverID.String())
		prevCellKey := fmt.Sprintf("driver:h3cell:%s", driverID.String())
		s.redis.Delete(ctx, prevCellKey)
	}

	return nil
}

// GetDriverStatus gets driver's availability status
func (s *Service) GetDriverStatus(ctx context.Context, driverID uuid.UUID) (string, error) {
	key := fmt.Sprintf("%s%s", driverStatusPrefix, driverID.String())
	data, err := s.redis.GetString(ctx, key)
	if err != nil {
		return "offline", nil // Default to offline if not found
	}

	var statusData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &statusData); err != nil {
		return "offline", nil
	}

	if status, ok := statusData["status"].(string); ok {
		return status, nil
	}

	return "offline", nil
}

// FindAvailableDrivers finds nearby available drivers (not busy with a ride)
func (s *Service) FindAvailableDrivers(ctx context.Context, latitude, longitude float64, maxDrivers int) ([]*DriverLocation, error) {
	nearbyDrivers, err := s.FindNearbyDrivers(ctx, latitude, longitude, maxDrivers*2)
	if err != nil {
		return nil, err
	}

	var availableDrivers []*DriverLocation
	for _, driver := range nearbyDrivers {
		status, err := s.GetDriverStatus(ctx, driver.DriverID)
		if err != nil || status != "available" {
			continue
		}

		availableDrivers = append(availableDrivers, driver)
		if len(availableDrivers) >= maxDrivers {
			break
		}
	}

	if availableDrivers == nil {
		availableDrivers = []*DriverLocation{}
	}

	return availableDrivers, nil
}

// GetSurgeInfo returns surge pricing information for a location using H3 zones
func (s *Service) GetSurgeInfo(ctx context.Context, latitude, longitude float64) (*SurgeInfo, error) {
	surgeZone := GetSurgeZone(latitude, longitude)
	key := fmt.Sprintf("%s%s", h3SurgePrefix, surgeZone)

	data, err := s.redis.GetString(ctx, key)
	if err != nil {
		return &SurgeInfo{
			H3Cell:          surgeZone,
			SurgeMultiplier: 1.0,
			DemandCount:     0,
			SupplyCount:     0,
			UpdatedAt:       time.Now(),
		}, nil
	}

	var info SurgeInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return nil, common.NewInternalServerError("failed to unmarshal surge data")
	}

	return &info, nil
}

// UpdateSurgeInfo updates surge pricing for a zone based on supply/demand
func (s *Service) UpdateSurgeInfo(ctx context.Context, latitude, longitude float64, demandCount, supplyCount int) error {
	surgeZone := GetSurgeZone(latitude, longitude)
	multiplier := calculateSurgeMultiplier(demandCount, supplyCount)

	info := &SurgeInfo{
		H3Cell:          surgeZone,
		SurgeMultiplier: multiplier,
		DemandCount:     demandCount,
		SupplyCount:     supplyCount,
		UpdatedAt:       time.Now(),
	}

	data, err := json.Marshal(info)
	if err != nil {
		return common.NewInternalServerError("failed to marshal surge data")
	}

	key := fmt.Sprintf("%s%s", h3SurgePrefix, surgeZone)
	return s.redis.SetWithExpiration(ctx, key, data, 5*time.Minute)
}

// IncrementDemand tracks a ride request in the demand heatmap
func (s *Service) IncrementDemand(ctx context.Context, latitude, longitude float64) {
	demandZone := GetDemandZone(latitude, longitude)
	key := fmt.Sprintf("%s%s", h3DemandPrefix, demandZone)

	data, err := s.redis.GetString(ctx, key)
	count := 0
	if err == nil {
		var info DemandInfo
		if json.Unmarshal([]byte(data), &info) == nil {
			count = info.RequestCount
		}
	}

	centerLat, centerLng := CellToLatLng(LatLngToCell(latitude, longitude, H3ResolutionDemand))
	info := &DemandInfo{
		H3Cell:       demandZone,
		RequestCount: count + 1,
		Latitude:     centerLat,
		Longitude:    centerLng,
	}

	jsonData, err := json.Marshal(info)
	if err != nil {
		logger.WarnContext(ctx, "failed to marshal demand data", zap.Error(err))
		return
	}

	s.redis.SetWithExpiration(ctx, key, jsonData, 15*time.Minute)
}

// GetDemandHeatmap returns demand data for areas surrounding a location
func (s *Service) GetDemandHeatmap(ctx context.Context, latitude, longitude float64) ([]*DemandInfo, error) {
	cells := GetKRingCellStrings(latitude, longitude, H3ResolutionDemand, 3)
	heatmap := make([]*DemandInfo, 0)

	for _, cellStr := range cells {
		key := fmt.Sprintf("%s%s", h3DemandPrefix, cellStr)
		data, err := s.redis.GetString(ctx, key)
		if err != nil {
			continue
		}

		var info DemandInfo
		if err := json.Unmarshal([]byte(data), &info); err != nil {
			continue
		}

		if info.RequestCount > 0 {
			heatmap = append(heatmap, &info)
		}
	}

	return heatmap, nil
}

// CalculateDistance calculates distance between two coordinates in kilometers
func (s *Service) CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0 // km

	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180.0)*math.Cos(lat2*math.Pi/180.0)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distance := earthRadius * c

	return math.Round(distance*100) / 100
}

// CalculateETA calculates estimated time of arrival in minutes
func (s *Service) CalculateETA(distance float64) int {
	const averageSpeed = 40.0 // km/h in city traffic
	eta := (distance / averageSpeed) * 60
	return int(math.Round(eta))
}

// calculateSurgeMultiplier computes surge based on demand/supply ratio.
func calculateSurgeMultiplier(demand, supply int) float64 {
	if supply == 0 {
		if demand == 0 {
			return 1.0
		}
		return 3.0 // Max surge when no supply
	}

	ratio := float64(demand) / float64(supply)

	switch {
	case ratio <= 1.0:
		return 1.0
	case ratio <= 1.5:
		return 1.0 + (ratio-1.0)*0.5
	case ratio <= 2.0:
		return 1.25 + (ratio-1.5)*0.75
	case ratio <= 3.0:
		return 1.625 + (ratio-2.0)*0.625
	default:
		surge := 2.25 + (ratio-3.0)*0.25
		if surge > 3.0 {
			surge = 3.0
		}
		return math.Round(surge*100) / 100
	}
}
