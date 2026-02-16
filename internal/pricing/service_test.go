package pricing

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHaversineDistance(t *testing.T) {
	tests := []struct {
		name       string
		latitude1  float64
		longitude1 float64
		latitude2  float64
		longitude2 float64
		expected   float64
		delta      float64 // acceptable tolerance in km
	}{
		{
			name:       "same point returns zero",
			latitude1:  40.7128, longitude1: -74.0060,
			latitude2:  40.7128, longitude2: -74.0060,
			expected:   0.0,
			delta:      0.001,
		},
		{
			name:       "New York to Los Angeles approximately 3944 km",
			latitude1:  40.7128, longitude1: -74.0060,
			latitude2:  34.0522, longitude2: -118.2437,
			expected:   3944.0,
			delta:      50.0,
		},
		{
			name:       "London to Paris approximately 344 km",
			latitude1:  51.5074, longitude1: -0.1278,
			latitude2:  48.8566, longitude2: 2.3522,
			expected:   344.0,
			delta:      10.0,
		},
		{
			name:       "short distance within a city (about 1 km)",
			latitude1:  40.7128, longitude1: -74.0060,
			latitude2:  40.7218, longitude2: -74.0060,
			expected:   1.0,
			delta:      0.1,
		},
		{
			name:       "antipodal points approximately 20000 km",
			latitude1:  0, longitude1: 0,
			latitude2:  0, longitude2: 180,
			expected:   20015.0,
			delta:      100.0,
		},
		{
			name:       "equator 90 degrees apart",
			latitude1:  0, longitude1: 0,
			latitude2:  0, longitude2: 90,
			expected:   10008.0,
			delta:      50.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := haversineDistance(tt.latitude1, tt.longitude1, tt.latitude2, tt.longitude2)
			assert.InDelta(t, tt.expected, result, tt.delta)
		})
	}
}

func TestHaversineDistance_Symmetry(t *testing.T) {
	// Distance from A to B should equal distance from B to A
	d1 := haversineDistance(40.7128, -74.0060, 34.0522, -118.2437)
	d2 := haversineDistance(34.0522, -118.2437, 40.7128, -74.0060)
	assert.InDelta(t, d1, d2, 0.001)
}

func TestFareCalculation_RoundValues(t *testing.T) {
	tests := []struct {
		name     string
		input    FareCalculation
		checkFn  func(t *testing.T, f *FareCalculation)
	}{
		{
			name: "rounds to 2 decimal places",
			input: FareCalculation{
				BaseFare:           3.456,
				DistanceCharge:     12.3456,
				TimeCharge:         2.999,
				BookingFee:         1.005,
				ZoneFeesTotal:      0.0,
				Subtotal:           19.8066,
				TaxAmount:          1.9807,
				TotalFare:          21.7873,
				PlatformCommission: 4.3575,
				DriverEarnings:     17.4298,
			},
			checkFn: func(t *testing.T, f *FareCalculation) {
				assert.Equal(t, 3.46, f.BaseFare)
				assert.Equal(t, 12.35, f.DistanceCharge)
				assert.Equal(t, 3.0, f.TimeCharge)
				assert.Equal(t, 1.0, f.BookingFee) // 1.005 is 1.00499... in float64, rounds to 1.00
				assert.Equal(t, 19.81, f.Subtotal)
				assert.Equal(t, 1.98, f.TaxAmount)
				assert.Equal(t, 21.79, f.TotalFare)
				assert.Equal(t, 4.36, f.PlatformCommission)
				assert.Equal(t, 17.43, f.DriverEarnings)
			},
		},
		{
			name: "already rounded values stay the same",
			input: FareCalculation{
				BaseFare:           3.00,
				DistanceCharge:     10.00,
				TimeCharge:         2.50,
				BookingFee:         1.00,
				ZoneFeesTotal:      0.00,
				Subtotal:           16.50,
				TaxAmount:          0.00,
				TotalFare:          16.50,
				PlatformCommission: 3.30,
				DriverEarnings:     13.20,
			},
			checkFn: func(t *testing.T, f *FareCalculation) {
				assert.Equal(t, 3.00, f.BaseFare)
				assert.Equal(t, 10.00, f.DistanceCharge)
				assert.Equal(t, 2.50, f.TimeCharge)
				assert.Equal(t, 1.00, f.BookingFee)
				assert.Equal(t, 16.50, f.Subtotal)
				assert.Equal(t, 16.50, f.TotalFare)
			},
		},
		{
			name: "zone fees breakdown also rounded",
			input: FareCalculation{
				ZoneFeesBreakdown: []ZoneFeeBreakdown{
					{Amount: 2.555},
					{Amount: 1.234},
				},
			},
			checkFn: func(t *testing.T, f *FareCalculation) {
				assert.Equal(t, 2.56, f.ZoneFeesBreakdown[0].Amount)
				assert.Equal(t, 1.23, f.ZoneFeesBreakdown[1].Amount)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.input
			f.roundValues()
			tt.checkFn(t, &f)
		})
	}
}

func TestGetCancellationFee(t *testing.T) {
	tests := []struct {
		name                string
		pricing             *ResolvedPricing
		minutesSinceRequest float64
		baseFare            float64
		expected            float64
	}{
		{
			name:                "no cancellation fees configured",
			pricing:             &ResolvedPricing{CancellationFees: nil},
			minutesSinceRequest: 10,
			baseFare:            20.0,
			expected:            0,
		},
		{
			name:                "empty cancellation fees",
			pricing:             &ResolvedPricing{CancellationFees: []CancellationFee{}},
			minutesSinceRequest: 10,
			baseFare:            20.0,
			expected:            0,
		},
		{
			name: "within free window (default pricing)",
			pricing: &ResolvedPricing{
				CancellationFees: []CancellationFee{
					{AfterMinutes: 0, Fee: 0, FeeType: "fixed"},
					{AfterMinutes: 2, Fee: 5, FeeType: "fixed"},
					{AfterMinutes: 5, Fee: 10, FeeType: "fixed"},
				},
			},
			minutesSinceRequest: 1,
			baseFare:            20.0,
			expected:            0,
		},
		{
			name: "after 2 minutes, $5 fee",
			pricing: &ResolvedPricing{
				CancellationFees: []CancellationFee{
					{AfterMinutes: 0, Fee: 0, FeeType: "fixed"},
					{AfterMinutes: 2, Fee: 5, FeeType: "fixed"},
					{AfterMinutes: 5, Fee: 10, FeeType: "fixed"},
				},
			},
			minutesSinceRequest: 3,
			baseFare:            20.0,
			expected:            5,
		},
		{
			name: "after 5 minutes, $10 fee",
			pricing: &ResolvedPricing{
				CancellationFees: []CancellationFee{
					{AfterMinutes: 0, Fee: 0, FeeType: "fixed"},
					{AfterMinutes: 2, Fee: 5, FeeType: "fixed"},
					{AfterMinutes: 5, Fee: 10, FeeType: "fixed"},
				},
			},
			minutesSinceRequest: 6,
			baseFare:            20.0,
			expected:            10,
		},
		{
			name: "percentage fee type",
			pricing: &ResolvedPricing{
				CancellationFees: []CancellationFee{
					{AfterMinutes: 0, Fee: 0, FeeType: "fixed"},
					{AfterMinutes: 5, Fee: 25, FeeType: "percentage"},
				},
			},
			minutesSinceRequest: 6,
			baseFare:            40.0,
			expected:            10.0, // 25% of $40 = $10
		},
		{
			name: "exactly at boundary",
			pricing: &ResolvedPricing{
				CancellationFees: []CancellationFee{
					{AfterMinutes: 0, Fee: 0, FeeType: "fixed"},
					{AfterMinutes: 2, Fee: 5, FeeType: "fixed"},
				},
			},
			minutesSinceRequest: 2,
			baseFare:            20.0,
			expected:            5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCancellationFee(tt.pricing, tt.minutesSinceRequest, tt.baseFare)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClampSurge(t *testing.T) {
	tests := []struct {
		name     string
		surge    float64
		pricing  *ResolvedPricing
		expected float64
	}{
		{
			name:  "within range returns same value",
			surge: 2.0,
			pricing: &ResolvedPricing{
				SurgeMinMultiplier: 1.0,
				SurgeMaxMultiplier: 5.0,
			},
			expected: 2.0,
		},
		{
			name:  "below minimum returns minimum",
			surge: 0.5,
			pricing: &ResolvedPricing{
				SurgeMinMultiplier: 1.0,
				SurgeMaxMultiplier: 5.0,
			},
			expected: 1.0,
		},
		{
			name:  "above maximum returns maximum",
			surge: 7.0,
			pricing: &ResolvedPricing{
				SurgeMinMultiplier: 1.0,
				SurgeMaxMultiplier: 5.0,
			},
			expected: 5.0,
		},
		{
			name:  "at minimum returns minimum",
			surge: 1.0,
			pricing: &ResolvedPricing{
				SurgeMinMultiplier: 1.0,
				SurgeMaxMultiplier: 5.0,
			},
			expected: 1.0,
		},
		{
			name:  "at maximum returns maximum",
			surge: 5.0,
			pricing: &ResolvedPricing{
				SurgeMinMultiplier: 1.0,
				SurgeMaxMultiplier: 5.0,
			},
			expected: 5.0,
		},
		{
			name:  "custom range",
			surge: 1.5,
			pricing: &ResolvedPricing{
				SurgeMinMultiplier: 1.2,
				SurgeMaxMultiplier: 3.0,
			},
			expected: 1.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClampSurge(tt.surge, tt.pricing)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateDemandSurge(t *testing.T) {
	tests := []struct {
		name        string
		demandRatio float64
		expected    float64
		delta       float64
	}{
		{
			name:        "low demand (ratio 0.5) returns 1.0",
			demandRatio: 0.5,
			expected:    1.0,
			delta:       0.01,
		},
		{
			name:        "normal demand (ratio 1.0) returns 1.0",
			demandRatio: 1.0,
			expected:    1.0,
			delta:       0.01,
		},
		{
			name:        "high demand ratio 1.5 returns 1.5",
			demandRatio: 1.5,
			expected:    1.5,
			delta:       0.01,
		},
		{
			name:        "high demand ratio 2.0 returns 2.0",
			demandRatio: 2.0,
			expected:    2.0,
			delta:       0.01,
		},
		{
			name:        "very high demand ratio 2.5 returns 2.5",
			demandRatio: 2.5,
			expected:    2.5,
			delta:       0.01,
		},
		{
			name:        "extreme demand ratio 3.0 returns 3.0",
			demandRatio: 3.0,
			expected:    3.0,
			delta:       0.01,
		},
		{
			name:        "extreme demand capped at 4.0",
			demandRatio: 100.0,
			expected:    4.0,
			delta:       0.01,
		},
		{
			name:        "zero demand returns 1.0",
			demandRatio: 0.0,
			expected:    1.0,
			delta:       0.01,
		},
		{
			name:        "negative demand returns 1.0",
			demandRatio: -1.0,
			expected:    1.0,
			delta:       0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateDemandSurge(tt.demandRatio)
			assert.InDelta(t, tt.expected, result, tt.delta)
		})
	}
}

func TestCalculateTimeBasedSurge(t *testing.T) {
	tests := []struct {
		name     string
		hour     int
		expected float64
	}{
		{"morning peak 7AM", 7, 1.5},
		{"morning peak 8AM", 8, 1.5},
		{"evening peak 5PM", 17, 1.8},
		{"evening peak 7PM", 19, 1.8},
		{"late night 11PM", 23, 1.4},
		{"late night 1AM", 1, 1.4},
		{"late night 4AM", 4, 1.4},
		{"lunch 12PM", 12, 1.2},
		{"lunch 1PM", 13, 1.2},
		{"regular 10AM", 10, 1.0},
		{"regular 3PM", 15, 1.0},
		{"early morning 5AM", 5, 1.0},
		{"late morning 9AM (after peak)", 9, 1.0},
		{"after evening peak 8PM", 20, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testTime := time.Date(2024, 6, 15, tt.hour, 30, 0, 0, time.UTC)
			result := calculateTimeBasedSurge(testTime)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateDayBasedSurge(t *testing.T) {
	tests := []struct {
		name     string
		day      time.Weekday
		hour     int
		expected float64
	}{
		{"Monday morning 8AM", time.Monday, 8, 1.2},
		{"Monday morning 9AM", time.Monday, 9, 1.2},
		{"Monday afternoon", time.Monday, 14, 1.0},
		{"Tuesday regular", time.Tuesday, 12, 1.0},
		{"Wednesday regular", time.Wednesday, 15, 1.0},
		{"Thursday regular", time.Thursday, 10, 1.0},
		{"Friday daytime", time.Friday, 14, 1.2},
		{"Friday night", time.Friday, 21, 1.3},
		{"Saturday daytime", time.Saturday, 12, 1.2},
		{"Saturday night", time.Saturday, 22, 1.3},
		{"Sunday daytime", time.Sunday, 10, 1.2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Find a date that matches the desired weekday
			base := time.Date(2024, 6, 10, tt.hour, 0, 0, 0, time.UTC) // June 10, 2024 is Monday
			daysToAdd := int(tt.day) - int(base.Weekday())
			if daysToAdd < 0 {
				daysToAdd += 7
			}
			testTime := base.AddDate(0, 0, daysToAdd)
			result := calculateDayBasedSurge(testTime)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetSurgeMessage(t *testing.T) {
	tests := []struct {
		name     string
		surge    float64
		expected string
	}{
		{"normal pricing", 1.0, "Normal pricing"},
		{"mild surge", 1.2, "Mild surge pricing active"},
		{"increased demand", 1.5, "Increased demand - Fares are slightly higher"},
		{"high demand at 2.0", 2.0, "High demand - Fares are higher than normal"},
		{"very high demand at 3.0", 3.0, "Very high demand - Fares are significantly higher than normal"},
		{"extreme demand at 4.5", 4.5, "Very high demand - Fares are significantly higher than normal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSurgeMessage(tt.surge)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultPricing_Values(t *testing.T) {
	assert.Equal(t, 3.00, DefaultPricing.BaseFare)
	assert.Equal(t, 1.50, DefaultPricing.PerKmRate)
	assert.Equal(t, 0.25, DefaultPricing.PerMinuteRate)
	assert.Equal(t, 5.00, DefaultPricing.MinimumFare)
	assert.Equal(t, 1.00, DefaultPricing.BookingFee)
	assert.Equal(t, 20.00, DefaultPricing.PlatformCommissionPct)
	assert.Equal(t, 1.00, DefaultPricing.SurgeMinMultiplier)
	assert.Equal(t, 5.00, DefaultPricing.SurgeMaxMultiplier)
	assert.Equal(t, 0.00, DefaultPricing.TaxRatePct)
	assert.False(t, DefaultPricing.TaxInclusive)
	assert.Len(t, DefaultPricing.CancellationFees, 3)
}

func TestCalculateDemandSurge_Monotonic(t *testing.T) {
	// Surge should be non-decreasing as demand ratio increases
	prev := calculateDemandSurge(0.0)
	for ratio := 0.1; ratio <= 5.0; ratio += 0.1 {
		curr := calculateDemandSurge(ratio)
		assert.GreaterOrEqual(t, curr, prev,
			"surge should be non-decreasing at ratio %.1f (prev=%.2f, curr=%.2f)", ratio, prev, curr)
		prev = curr
	}
}

func TestCalculateDemandSurge_MaxCap(t *testing.T) {
	// Even with extreme demand, surge should never exceed 4.0
	result := calculateDemandSurge(1000.0)
	assert.LessOrEqual(t, result, 4.0)

	result = calculateDemandSurge(math.MaxFloat64 / 2)
	assert.LessOrEqual(t, result, 4.0)
}
