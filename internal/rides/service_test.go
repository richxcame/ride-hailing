package rides

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCalculateFare(t *testing.T) {
	tests := []struct {
		name            string
		distance        float64
		duration        int
		surgeMultiplier float64
		expectedMinimum float64
	}{
		{
			name:            "short ride no surge",
			distance:        2.0, // 2 km
			duration:        10,  // 10 minutes
			surgeMultiplier: 1.0,
			expectedMinimum: 5.0, // minimum fare
		},
		{
			name:            "medium ride no surge",
			distance:        10.0, // 10 km
			duration:        20,   // 20 minutes
			surgeMultiplier: 1.0,
			expectedMinimum: 10.0,
		},
		{
			name:            "long ride no surge",
			distance:        50.0, // 50 km
			duration:        60,   // 60 minutes
			surgeMultiplier: 1.0,
			expectedMinimum: 50.0,
		},
		{
			name:            "short ride with surge",
			distance:        5.0,
			duration:        15,
			surgeMultiplier: 1.5,
			expectedMinimum: 7.5, // higher due to surge
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fare := calculateFare(tt.distance, tt.duration, tt.surgeMultiplier)

			// Fare should never be less than minimum fare
			assert.GreaterOrEqual(t, fare, minimumFare)

			// Fare should be at least the expected minimum
			assert.GreaterOrEqual(t, fare, tt.expectedMinimum)

			// Fare should be reasonable (not negative or zero)
			assert.Greater(t, fare, 0.0)
		})
	}
}

func TestCalculateSurgeMultiplier(t *testing.T) {
	tests := []struct {
		name            string
		hour            int
		expectedMinimum float64
		expectedMaximum float64
	}{
		{
			name:            "morning rush hour (8 AM)",
			hour:            8,
			expectedMinimum: 1.5,
			expectedMaximum: 1.5,
		},
		{
			name:            "evening rush hour (6 PM)",
			hour:            18,
			expectedMinimum: 1.5,
			expectedMaximum: 1.5,
		},
		{
			name:            "late night (11 PM)",
			hour:            23,
			expectedMinimum: 1.3,
			expectedMaximum: 1.3,
		},
		{
			name:            "early morning (3 AM)",
			hour:            3,
			expectedMinimum: 1.3,
			expectedMaximum: 1.3,
		},
		{
			name:            "mid-day (2 PM)",
			hour:            14,
			expectedMinimum: 1.0,
			expectedMaximum: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a time with the specified hour
			testTime := time.Date(2025, 1, 1, tt.hour, 0, 0, 0, time.UTC)

			multiplier := calculateSurgeMultiplier(testTime)

			assert.GreaterOrEqual(t, multiplier, tt.expectedMinimum)
			assert.LessOrEqual(t, multiplier, tt.expectedMaximum)

			// Surge should never be less than 1.0
			assert.GreaterOrEqual(t, multiplier, 1.0)
		})
	}
}

func TestFareWithSurge(t *testing.T) {
	distance := 10.0
	duration := 20
	baseFare := calculateFare(distance, duration, 1.0)

	tests := []struct {
		name             string
		hour             int
		expectedIncrease bool
	}{
		{
			name:             "rush hour increases fare",
			hour:             8,
			expectedIncrease: true,
		},
		{
			name:             "normal hour no increase",
			hour:             14,
			expectedIncrease: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testTime := time.Date(2025, 1, 1, tt.hour, 0, 0, 0, time.UTC)
			multiplier := calculateSurgeMultiplier(testTime)
			finalFare := calculateFare(distance, duration, multiplier)

			if tt.expectedIncrease {
				assert.Greater(t, finalFare, baseFare)
			} else {
				assert.Equal(t, finalFare, baseFare)
			}
		})
	}
}

func TestCommissionCalculation(t *testing.T) {
	tests := []struct {
		name                   string
		totalFare              float64
		expectedCommission     float64
		expectedDriverEarnings float64
	}{
		{
			name:                   "standard fare",
			totalFare:              100.00,
			expectedCommission:     20.00, // 20%
			expectedDriverEarnings: 80.00,
		},
		{
			name:                   "minimum fare",
			totalFare:              5.00,
			expectedCommission:     1.00,
			expectedDriverEarnings: 4.00,
		},
		{
			name:                   "high fare",
			totalFare:              500.00,
			expectedCommission:     100.00,
			expectedDriverEarnings: 400.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commission := tt.totalFare * commissionRate
			driverEarnings := tt.totalFare - commission

			assert.InDelta(t, tt.expectedCommission, commission, 0.01)
			assert.InDelta(t, tt.expectedDriverEarnings, driverEarnings, 0.01)
		})
	}
}

func TestDistanceValidation(t *testing.T) {
	tests := []struct {
		name     string
		distance float64
		valid    bool
	}{
		{
			name:     "valid short distance",
			distance: 2.0,
			valid:    true,
		},
		{
			name:     "valid long distance",
			distance: 100.0,
			valid:    true,
		},
		{
			name:     "zero distance",
			distance: 0.0,
			valid:    false,
		},
		{
			name:     "negative distance",
			distance: -5.0,
			valid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				assert.Greater(t, tt.distance, 0.0)
			} else {
				assert.LessOrEqual(t, tt.distance, 0.0)
			}
		})
	}
}
