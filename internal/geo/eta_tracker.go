package geo

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/httpclient"
	"github.com/richxcame/ride-hailing/pkg/logger"
	redisClient "github.com/richxcame/ride-hailing/pkg/redis"
	"go.uber.org/zap"
)

const (
	activeRidePrefix    = "ride:active:"    // Maps driver_id -> ride tracking info
	etaUpdateMinInterval = 5 * time.Second  // Don't recalculate ETA more often than this
)

// ActiveRideInfo tracks an in-progress ride for ETA updates.
type ActiveRideInfo struct {
	RideID         uuid.UUID `json:"ride_id"`
	RiderID        uuid.UUID `json:"rider_id"`
	DriverID       uuid.UUID `json:"driver_id"`
	DropoffLat     float64   `json:"dropoff_lat"`
	DropoffLng     float64   `json:"dropoff_lng"`
	Status         string    `json:"status"` // "accepted" (en route to pickup) or "started" (en route to dropoff)
	PickupLat      float64   `json:"pickup_lat"`
	PickupLng      float64   `json:"pickup_lng"`
	LastETAUpdate  time.Time `json:"last_eta_update"`
}

// ETAUpdate is the payload broadcast to riders via WebSocket.
type ETAUpdate struct {
	RideID          string  `json:"ride_id"`
	ETAMinutes      int     `json:"eta_minutes"`
	DistanceKm      float64 `json:"distance_km"`
	DriverLatitude  float64 `json:"driver_latitude"`
	DriverLongitude float64 `json:"driver_longitude"`
	DriverHeading   float64 `json:"driver_heading"`
	DriverSpeed     float64 `json:"driver_speed"`
	UpdatedAt       string  `json:"updated_at"`
}

// ETATracker recalculates and broadcasts ETA when drivers move during active rides.
type ETATracker struct {
	redis          redisClient.ClientInterface
	realtimeClient *httpclient.Client
	mu             sync.Mutex
	lastUpdate     map[string]time.Time // driverID -> last ETA broadcast time
}

// NewETATracker creates a new ETA tracker.
func NewETATracker(redis redisClient.ClientInterface, realtimeServiceURL string) *ETATracker {
	return &ETATracker{
		redis:          redis,
		realtimeClient: httpclient.NewClient(realtimeServiceURL),
		lastUpdate:     make(map[string]time.Time),
	}
}

// RegisterActiveRide registers a ride for real-time ETA tracking.
// Call this when a ride is accepted or started.
func (t *ETATracker) RegisterActiveRide(ctx context.Context, info *ActiveRideInfo) error {
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s%s", activeRidePrefix, info.DriverID.String())
	return t.redis.SetWithExpiration(ctx, key, data, 2*time.Hour)
}

// UnregisterActiveRide removes a ride from ETA tracking.
// Call this when a ride is completed or cancelled.
func (t *ETATracker) UnregisterActiveRide(ctx context.Context, driverID uuid.UUID) {
	key := fmt.Sprintf("%s%s", activeRidePrefix, driverID.String())
	t.redis.Delete(ctx, key)

	t.mu.Lock()
	delete(t.lastUpdate, driverID.String())
	t.mu.Unlock()
}

// OnDriverLocationUpdate is called when a driver's location changes.
// If the driver has an active ride, it recalculates and broadcasts the ETA.
func (t *ETATracker) OnDriverLocationUpdate(ctx context.Context, driverID uuid.UUID, lat, lng, heading, speed float64) {
	// Rate limit: skip if we recently sent an update for this driver
	driverIDStr := driverID.String()
	t.mu.Lock()
	if last, ok := t.lastUpdate[driverIDStr]; ok && time.Since(last) < etaUpdateMinInterval {
		t.mu.Unlock()
		return
	}
	t.mu.Unlock()

	// Check if driver has an active ride
	key := fmt.Sprintf("%s%s", activeRidePrefix, driverIDStr)
	data, err := t.redis.GetString(ctx, key)
	if err != nil {
		return // No active ride
	}

	var rideInfo ActiveRideInfo
	if err := json.Unmarshal([]byte(data), &rideInfo); err != nil {
		return
	}

	// Determine destination based on ride status
	destLat, destLng := rideInfo.DropoffLat, rideInfo.DropoffLng
	if rideInfo.Status == "accepted" {
		// Driver is heading to pickup
		destLat, destLng = rideInfo.PickupLat, rideInfo.PickupLng
	}

	// Calculate distance and ETA
	distance := haversineDistance(lat, lng, destLat, destLng)
	etaMinutes := estimateETAMinutes(distance, speed)

	// Record update time
	t.mu.Lock()
	t.lastUpdate[driverIDStr] = time.Now()
	t.mu.Unlock()

	// Broadcast ETA update to the rider
	update := &ETAUpdate{
		RideID:          rideInfo.RideID.String(),
		ETAMinutes:      etaMinutes,
		DistanceKm:      math.Round(distance*100) / 100,
		DriverLatitude:  lat,
		DriverLongitude: lng,
		DriverHeading:   heading,
		DriverSpeed:     speed,
		UpdatedAt:       time.Now().UTC().Format(time.RFC3339),
	}

	t.broadcastETAUpdate(ctx, rideInfo.RiderID.String(), rideInfo.RideID.String(), update)
}

func (t *ETATracker) broadcastETAUpdate(ctx context.Context, riderID, rideID string, update *ETAUpdate) {
	// Broadcast to rider via realtime service
	payload := map[string]interface{}{
		"user_id": riderID,
		"type":    "eta_update",
		"data":    update,
	}

	_, err := t.realtimeClient.Post(ctx, "/api/v1/internal/broadcast/user", payload, nil)
	if err != nil {
		logger.DebugContext(ctx, "failed to broadcast ETA update", zap.Error(err))
	}

	// Also broadcast to ride room
	ridePayload := map[string]interface{}{
		"ride_id": rideID,
		"data":    update,
	}
	_, err = t.realtimeClient.Post(ctx, "/api/v1/internal/broadcast/ride", ridePayload, nil)
	if err != nil {
		logger.DebugContext(ctx, "failed to broadcast ETA to ride room", zap.Error(err))
	}
}

// haversineDistance calculates great-circle distance in km.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180.0)*math.Cos(lat2*math.Pi/180.0)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}

// estimateETAMinutes estimates arrival time based on distance and current speed.
func estimateETAMinutes(distanceKm, speedKmh float64) int {
	if speedKmh > 5 {
		// Use actual speed when driver is moving
		eta := (distanceKm / speedKmh) * 60
		return int(math.Ceil(eta))
	}
	// Fallback to average city speed
	const avgSpeed = 30.0
	eta := (distanceKm / avgSpeed) * 60
	return int(math.Ceil(eta))
}
