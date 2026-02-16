package pool

import (
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCalculateFare(t *testing.T) {
	svc := &Service{config: DefaultServiceConfig()}

	tests := []struct {
		name            string
		distanceKm      float64
		durationMinutes int
		expected        float64
	}{
		{
			name:            "short ride",
			distanceKm:      2.0,
			durationMinutes: 5,
			// 2.0 + (2.0 * 1.5) + (5 * 0.25) = 2.0 + 3.0 + 1.25 = 6.25
			expected: 6.25,
		},
		{
			name:            "medium ride",
			distanceKm:      10.0,
			durationMinutes: 20,
			// 2.0 + (10 * 1.5) + (20 * 0.25) = 2.0 + 15.0 + 5.0 = 22.0
			expected: 22.0,
		},
		{
			name:            "zero distance and duration",
			distanceKm:      0,
			durationMinutes: 0,
			// 2.0 + 0 + 0 = 2.0
			expected: 2.0,
		},
		{
			name:            "long ride",
			distanceKm:      50.0,
			durationMinutes: 60,
			// 2.0 + (50 * 1.5) + (60 * 0.25) = 2.0 + 75.0 + 15.0 = 92.0
			expected: 92.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &PoolConfig{}
			result := svc.calculateFare(tt.distanceKm, tt.durationMinutes, config)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateBearing(t *testing.T) {
	svc := &Service{config: DefaultServiceConfig()}

	tests := []struct {
		name       string
		latitude1  float64
		longitude1 float64
		latitude2  float64
		longitude2 float64
		minBear    float64 // minimum expected bearing
		maxBear    float64 // maximum expected bearing
	}{
		{
			name:       "due north",
			latitude1:  0, longitude1: 0,
			latitude2:  10, longitude2: 0,
			minBear:    355, maxBear: 5, // approximately 0 degrees
		},
		{
			name:       "due east",
			latitude1:  0, longitude1: 0,
			latitude2:  0, longitude2: 10,
			minBear:    85, maxBear: 95,
		},
		{
			name:       "due south",
			latitude1:  10, longitude1: 0,
			latitude2:  0, longitude2: 0,
			minBear:    175, maxBear: 185,
		},
		{
			name:       "due west",
			latitude1:  0, longitude1: 10,
			latitude2:  0, longitude2: 0,
			minBear:    265, maxBear: 275,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bearing := svc.calculateBearing(tt.latitude1, tt.longitude1, tt.latitude2, tt.longitude2)
			assert.GreaterOrEqual(t, bearing, 0.0, "bearing should be >= 0")
			assert.Less(t, bearing, 360.0, "bearing should be < 360")

			// Handle wraparound for north (near 0/360)
			if tt.minBear > tt.maxBear {
				assert.True(t, bearing >= tt.minBear || bearing <= tt.maxBear,
					"bearing %.1f should be near 0/360 degrees (between %.1f and %.1f)", bearing, tt.minBear, tt.maxBear)
			} else {
				assert.True(t, bearing >= tt.minBear && bearing <= tt.maxBear,
					"bearing %.1f should be between %.1f and %.1f", bearing, tt.minBear, tt.maxBear)
			}
		})
	}
}

func TestCalculateDirectionAlignment(t *testing.T) {
	svc := &Service{config: DefaultServiceConfig()}

	tests := []struct {
		name     string
		pool     *PoolRide
		req      *RequestPoolRideRequest
		minScore float64
		maxScore float64
	}{
		{
			name: "same direction (both heading north)",
			pool: &PoolRide{CenterLatitude: 0, CenterLongitude: 0},
			req: &RequestPoolRideRequest{
				PickupLocation:  Location{Latitude: 0, Longitude: 0},
				DropoffLocation: Location{Latitude: 10, Longitude: 0},
			},
			minScore: 0.8,
			maxScore: 1.0,
		},
		{
			name: "opposite direction",
			pool: &PoolRide{CenterLatitude: 0, CenterLongitude: -10}, // Pool center west of dropoff → bearing ~90° (east)
			req: &RequestPoolRideRequest{
				PickupLocation:  Location{Latitude: 0, Longitude: 10}, // Pickup east of dropoff → bearing ~270° (west)
				DropoffLocation: Location{Latitude: 0, Longitude: 0},
			},
			minScore: 0.0,
			maxScore: 0.05,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := svc.calculateDirectionAlignment(tt.pool, tt.req)
			assert.GreaterOrEqual(t, score, tt.minScore,
				"score %.2f should be >= %.2f", score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore,
				"score %.2f should be <= %.2f", score, tt.maxScore)
		})
	}
}

func TestCalculateDirectionAlignment_Range(t *testing.T) {
	svc := &Service{config: DefaultServiceConfig()}

	// Score should always be between 0 and 1
	for latitude := -90.0; latitude <= 90.0; latitude += 45.0 {
		pool := &PoolRide{CenterLatitude: latitude, CenterLongitude: 0}
		req := &RequestPoolRideRequest{
			PickupLocation:  Location{Latitude: latitude + 1, Longitude: 10},
			DropoffLocation: Location{Latitude: latitude - 1, Longitude: 20},
		}
		score := svc.calculateDirectionAlignment(pool, req)
		assert.GreaterOrEqual(t, score, 0.0)
		assert.LessOrEqual(t, score, 1.0)
	}
}

func TestGetStopIndex(t *testing.T) {
	svc := &Service{config: DefaultServiceConfig()}

	passengerID1 := uuid.New()
	passengerID2 := uuid.New()

	route := []RouteStop{
		{PoolPassengerID: passengerID1, Type: RouteStopPickup},
		{PoolPassengerID: passengerID2, Type: RouteStopPickup},
		{PoolPassengerID: passengerID1, Type: RouteStopDropoff},
		{PoolPassengerID: passengerID2, Type: RouteStopDropoff},
	}

	tests := []struct {
		name        string
		passengerID uuid.UUID
		stopType    RouteStopType
		expected    int
	}{
		{
			name:        "first passenger pickup",
			passengerID: passengerID1,
			stopType:    RouteStopPickup,
			expected:    1,
		},
		{
			name:        "second passenger pickup",
			passengerID: passengerID2,
			stopType:    RouteStopPickup,
			expected:    2,
		},
		{
			name:        "first passenger dropoff",
			passengerID: passengerID1,
			stopType:    RouteStopDropoff,
			expected:    3,
		},
		{
			name:        "second passenger dropoff",
			passengerID: passengerID2,
			stopType:    RouteStopDropoff,
			expected:    4,
		},
		{
			name:        "non-existent passenger returns 0",
			passengerID: uuid.New(),
			stopType:    RouteStopPickup,
			expected:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.getStopIndex(route, tt.passengerID, tt.stopType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStopIndex_EmptyRoute(t *testing.T) {
	svc := &Service{config: DefaultServiceConfig()}
	result := svc.getStopIndex(nil, uuid.New(), RouteStopPickup)
	assert.Equal(t, 0, result)
}

func TestGetResponseMessage(t *testing.T) {
	svc := &Service{config: DefaultServiceConfig()}

	tests := []struct {
		name           string
		matchFound     bool
		savingsPercent float64
		contains       string
	}{
		{
			name:           "match found",
			matchFound:     true,
			savingsPercent: 25.0,
			contains:       "We found a pool match",
		},
		{
			name:           "no match found",
			matchFound:     false,
			savingsPercent: 0,
			contains:       "looking for other riders",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.getResponseMessage(tt.matchFound, tt.savingsPercent)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestDefaultServiceConfig(t *testing.T) {
	config := DefaultServiceConfig()

	assert.Equal(t, 5, config.DefaultMaxWaitMinutes)
	assert.Equal(t, 25.0, config.DefaultMaxDetourPercent)
	assert.Equal(t, 25.0, config.DefaultDiscountPercent)
	assert.Equal(t, 4, config.MaxPassengersPerRide)
	assert.Equal(t, 7, config.H3Resolution)
	assert.Equal(t, 0.5, config.MinMatchScore)
}

func TestGetDefaultConfig(t *testing.T) {
	serviceConfig := DefaultServiceConfig()
	svc := &Service{config: serviceConfig}
	config := svc.getDefaultConfig()

	assert.Equal(t, serviceConfig.DefaultMaxDetourPercent, config.MaxDetourPercent)
	assert.Equal(t, 15, config.MaxDetourMinutes)
	assert.Equal(t, serviceConfig.DefaultMaxWaitMinutes, config.MaxWaitMinutes)
	assert.Equal(t, serviceConfig.MaxPassengersPerRide, config.MaxPassengersPerRide)
	assert.Equal(t, serviceConfig.MinMatchScore, config.MinMatchScore)
	assert.Equal(t, 3.0, config.MatchRadiusKm)
	assert.Equal(t, serviceConfig.H3Resolution, config.H3Resolution)
	assert.Equal(t, serviceConfig.DefaultDiscountPercent, config.DiscountPercent)
	assert.Equal(t, 15.0, config.MinSavingsPercent)
	assert.True(t, config.AllowDifferentDropoffs)
	assert.True(t, config.AllowMidRoutePickup)
	assert.True(t, config.IsActive)
}

func TestCalculateFare_Rounding(t *testing.T) {
	svc := &Service{config: DefaultServiceConfig()}
	config := &PoolConfig{}

	// A fare that should produce a non-round number
	// 2.0 + (3.33 * 1.5) + (7 * 0.25) = 2.0 + 4.995 + 1.75 = 8.745
	result := svc.calculateFare(3.33, 7, config)
	// Should be rounded to 2 decimal places
	assert.Equal(t, math.Round(result*100)/100, result,
		"fare should be rounded to 2 decimal places")
}

func TestBuildRouteStops(t *testing.T) {
	svc := &Service{config: DefaultServiceConfig()}

	pool := &PoolRide{
		OptimizedRoute: []RouteStop{
			{Location: Location{Latitude: 1.0, Longitude: 2.0}},
			{Location: Location{Latitude: 3.0, Longitude: 4.0}},
		},
	}

	req := &RequestPoolRideRequest{
		PickupLocation:  Location{Latitude: 5.0, Longitude: 6.0},
		DropoffLocation: Location{Latitude: 7.0, Longitude: 8.0},
	}

	stops := svc.buildRouteStops(pool, req)

	assert.Len(t, stops, 4)
	// First two are from existing route
	assert.Equal(t, 1.0, stops[0].Latitude)
	assert.Equal(t, 3.0, stops[1].Latitude)
	// Last two are the new pickup and dropoff
	assert.Equal(t, 5.0, stops[2].Latitude)
	assert.Equal(t, 7.0, stops[3].Latitude)
}

func TestBuildRouteStops_EmptyPool(t *testing.T) {
	svc := &Service{config: DefaultServiceConfig()}

	pool := &PoolRide{
		OptimizedRoute: nil,
	}

	req := &RequestPoolRideRequest{
		PickupLocation:  Location{Latitude: 5.0, Longitude: 6.0},
		DropoffLocation: Location{Latitude: 7.0, Longitude: 8.0},
	}

	stops := svc.buildRouteStops(pool, req)

	assert.Len(t, stops, 2)
	assert.Equal(t, 5.0, stops[0].Latitude)
	assert.Equal(t, 7.0, stops[1].Latitude)
}
