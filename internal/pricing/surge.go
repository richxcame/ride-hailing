package pricing

import (
	"context"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SurgeCalculator handles dynamic surge pricing calculations
type SurgeCalculator struct {
	db *pgxpool.Pool
}

// NewSurgeCalculator creates a new surge pricing calculator
func NewSurgeCalculator(db *pgxpool.Pool) *SurgeCalculator {
	return &SurgeCalculator{db: db}
}

// SurgeFactors represents all factors that contribute to surge pricing
type SurgeFactors struct {
	DemandRatio    float64 // Rides requested / Available drivers
	TimeMultiplier float64 // Based on time of day
	DayMultiplier  float64 // Based on day of week
	ZoneMultiplier float64 // Based on geographic demand
	WeatherFactor  float64 // Weather conditions (future)
}

// CalculateSurgeMultiplier calculates the dynamic surge multiplier
func (sc *SurgeCalculator) CalculateSurgeMultiplier(ctx context.Context, latitude, longitude float64) (float64, error) {
	factors, err := sc.getSurgeFactors(ctx, latitude, longitude)
	if err != nil {
		// Fallback to time-based surge on error
		return calculateTimeBasedSurge(time.Now()), nil
	}

	// Calculate combined surge multiplier
	// Base multiplier starts at 1.0
	surgeMultiplier := 1.0

	// Add demand-based surge (most important factor - 60% weight)
	demandSurge := calculateDemandSurge(factors.DemandRatio)
	surgeMultiplier += (demandSurge - 1.0) * 0.6

	// Add time-based surge (20% weight)
	surgeMultiplier += (factors.TimeMultiplier - 1.0) * 0.2

	// Add day-based surge (10% weight)
	surgeMultiplier += (factors.DayMultiplier - 1.0) * 0.1

	// Add zone-based surge (10% weight)
	surgeMultiplier += (factors.ZoneMultiplier - 1.0) * 0.1

	// Ensure surge is within reasonable bounds (1.0 to 5.0)
	surgeMultiplier = math.Max(1.0, math.Min(5.0, surgeMultiplier))

	// Round to 1 decimal place
	surgeMultiplier = math.Round(surgeMultiplier*10) / 10

	return surgeMultiplier, nil
}

// getSurgeFactors retrieves all factors that contribute to surge pricing
func (sc *SurgeCalculator) getSurgeFactors(ctx context.Context, latitude, longitude float64) (*SurgeFactors, error) {
	now := time.Now()

	// Get demand ratio (active requests vs available drivers in the area)
	demandRatio, err := sc.getDemandRatio(ctx, latitude, longitude)
	if err != nil {
		demandRatio = 1.0 // Default neutral
	}

	// Get zone multiplier (high-demand geographic areas)
	zoneMultiplier, err := sc.getZoneMultiplier(ctx, latitude, longitude)
	if err != nil {
		zoneMultiplier = 1.0 // Default neutral
	}

	return &SurgeFactors{
		DemandRatio:    demandRatio,
		TimeMultiplier: calculateTimeBasedSurge(now),
		DayMultiplier:  calculateDayBasedSurge(now),
		ZoneMultiplier: zoneMultiplier,
		WeatherFactor:  1.0, // Future: integrate weather API
	}, nil
}

// getDemandRatio calculates the ratio of ride requests to available drivers
func (sc *SurgeCalculator) getDemandRatio(ctx context.Context, latitude, longitude float64) (float64, error) {
	// Count active ride requests in the past 15 minutes within 10km radius
	requestCountQuery := `
		SELECT COUNT(*) as request_count
		FROM rides
		WHERE status IN ('requested', 'accepted')
		  AND requested_at >= NOW() - INTERVAL '15 minutes'
		  AND ST_DWithin(
			  ST_MakePoint(pickup_longitude, pickup_latitude)::geography,
			  ST_MakePoint($1, $2)::geography,
			  10000
		  )
	`

	var requestCount int
	err := sc.db.QueryRow(ctx, requestCountQuery, longitude, latitude).Scan(&requestCount)
	if err != nil {
		return 1.0, err
	}

	// Count available drivers in the area (from users table with role='driver' and is_active=true)
	// In a real system, this would query Redis for real-time driver locations
	driverCountQuery := `
		SELECT COUNT(DISTINCT u.id) as driver_count
		FROM users u
		INNER JOIN rides r ON u.id = r.driver_id
		WHERE u.role = 'driver'
		  AND u.is_active = true
		  AND r.status = 'completed'
		  AND r.completed_at >= NOW() - INTERVAL '1 hour'
		  AND ST_DWithin(
			  ST_MakePoint(r.dropoff_longitude, r.dropoff_latitude)::geography,
			  ST_MakePoint($1, $2)::geography,
			  10000
		  )
	`

	var driverCount int
	err = sc.db.QueryRow(ctx, driverCountQuery, longitude, latitude).Scan(&driverCount)
	if err != nil || driverCount == 0 {
		// Assume average driver availability
		driverCount = 10
	}

	// Calculate demand ratio
	// If more requests than drivers, ratio > 1
	// If more drivers than requests, ratio < 1
	ratio := float64(requestCount) / float64(driverCount)

	return ratio, nil
}

// getZoneMultiplier calculates surge based on geographic demand patterns
func (sc *SurgeCalculator) getZoneMultiplier(ctx context.Context, latitude, longitude float64) (float64, error) {
	// Check if this location is in a high-demand zone
	// High-demand zones: airports, train stations, event venues, business districts
	// This would ideally be configured in a separate zones table
	// For now, we'll calculate based on recent ride density

	query := `
		SELECT COUNT(*) as ride_density
		FROM rides
		WHERE status = 'completed'
		  AND completed_at >= NOW() - INTERVAL '24 hours'
		  AND ST_DWithin(
			  ST_MakePoint(pickup_longitude, pickup_latitude)::geography,
			  ST_MakePoint($1, $2)::geography,
			  2000  -- 2km radius for zone detection
		  )
	`

	var rideDensity int
	err := sc.db.QueryRow(ctx, query, longitude, latitude).Scan(&rideDensity)
	if err != nil {
		return 1.0, err
	}

	// Calculate zone multiplier based on density
	// High density (>50 rides): 1.2x
	// Medium density (20-50 rides): 1.1x
	// Low density (<20 rides): 1.0x
	if rideDensity > 50 {
		return 1.2, nil
	} else if rideDensity > 20 {
		return 1.1, nil
	}

	return 1.0, nil
}

// calculateDemandSurge converts demand ratio to surge multiplier
func calculateDemandSurge(demandRatio float64) float64 {
	// Demand ratio ranges:
	// < 0.5: Low demand (1.0x)
	// 0.5-1.0: Normal demand (1.0x)
	// 1.0-2.0: High demand (1.0x - 2.0x)
	// 2.0-3.0: Very high demand (2.0x - 3.0x)
	// > 3.0: Extreme demand (3.0x - 4.0x)

	if demandRatio < 1.0 {
		return 1.0
	} else if demandRatio < 2.0 {
		// Linear increase from 1.0x to 2.0x
		return 1.0 + (demandRatio - 1.0)
	} else if demandRatio < 3.0 {
		// Linear increase from 2.0x to 3.0x
		return 2.0 + (demandRatio - 2.0)
	} else {
		// Cap at 4.0x with logarithmic scaling
		return math.Min(4.0, 3.0+math.Log10(demandRatio-2.0))
	}
}

// calculateTimeBasedSurge calculates surge based on time of day
func calculateTimeBasedSurge(t time.Time) float64 {
	hour := t.Hour()

	// Peak morning hours: 7-9 AM (1.5x)
	if hour >= 7 && hour < 9 {
		return 1.5
	}

	// Peak evening hours: 5-8 PM (1.8x - highest time-based surge)
	if hour >= 17 && hour < 20 {
		return 1.8
	}

	// Late night: 11 PM - 5 AM (1.4x)
	if hour >= 23 || hour < 5 {
		return 1.4
	}

	// Lunch hours: 12-2 PM (1.2x)
	if hour >= 12 && hour < 14 {
		return 1.2
	}

	// Regular hours
	return 1.0
}

// calculateDayBasedSurge calculates surge based on day of week
func calculateDayBasedSurge(t time.Time) float64 {
	weekday := t.Weekday()

	// Friday evening - Sunday (weekend surge)
	if weekday == time.Friday || weekday == time.Saturday || weekday == time.Sunday {
		hour := t.Hour()
		// Friday/Saturday night premium
		if (weekday == time.Friday || weekday == time.Saturday) && hour >= 20 {
			return 1.3
		}
		// General weekend premium
		return 1.2
	}

	// Monday morning surge (back to work)
	if weekday == time.Monday && t.Hour() >= 7 && t.Hour() < 10 {
		return 1.2
	}

	// Regular weekday
	return 1.0
}

// GetCurrentSurgeInfo returns detailed surge information for a location
func (sc *SurgeCalculator) GetCurrentSurgeInfo(ctx context.Context, latitude, longitude float64) (map[string]interface{}, error) {
	factors, err := sc.getSurgeFactors(ctx, latitude, longitude)
	if err != nil {
		return nil, err
	}

	surge, _ := sc.CalculateSurgeMultiplier(ctx, latitude, longitude)

	return map[string]interface{}{
		"surge_multiplier": surge,
		"factors": map[string]interface{}{
			"demand_ratio":    factors.DemandRatio,
			"demand_surge":    calculateDemandSurge(factors.DemandRatio),
			"time_multiplier": factors.TimeMultiplier,
			"day_multiplier":  factors.DayMultiplier,
			"zone_multiplier": factors.ZoneMultiplier,
			"weather_factor":  factors.WeatherFactor,
		},
		"is_surge_active": surge > 1.0,
		"message":         getSurgeMessage(surge),
	}, nil
}

// getSurgeMessage returns a user-friendly message about surge pricing
func getSurgeMessage(surge float64) string {
	if surge >= 3.0 {
		return "Very high demand - Fares are significantly higher than normal"
	} else if surge >= 2.0 {
		return "High demand - Fares are higher than normal"
	} else if surge >= 1.5 {
		return "Increased demand - Fares are slightly higher"
	} else if surge > 1.0 {
		return "Mild surge pricing active"
	}
	return "Normal pricing"
}
