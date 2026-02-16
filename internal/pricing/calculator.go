package pricing

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/internal/geography"
)

// Calculator handles fare calculation
type Calculator struct {
	repo     RepositoryInterface
	resolver *Resolver
	geoSvc   *geography.Service
}

// NewCalculator creates a new pricing calculator
func NewCalculator(repo RepositoryInterface, resolver *Resolver, geoSvc *geography.Service) *Calculator {
	return &Calculator{
		repo:     repo,
		resolver: resolver,
		geoSvc:   geoSvc,
	}
}

// CalculateInput contains all inputs for fare calculation
type CalculateInput struct {
	PickupLatitude   float64
	PickupLongitude  float64
	DropoffLatitude  float64
	DropoffLongitude float64
	DistanceKm       float64
	DurationMin      int
	RideTypeID       *uuid.UUID
	WeatherCondition string  // Optional: current weather
	DemandSupplyRatio float64 // Optional: demand/supply ratio for surge
	NegotiatedFare   *float64 // Optional: pre-negotiated fare
	Currency         string   // Target currency
}

// Calculate performs a complete fare calculation
func (c *Calculator) Calculate(ctx context.Context, input CalculateInput) (*FareCalculation, error) {
	// Resolve geographic hierarchy for pickup
	pickupResolved, err := c.geoSvc.ResolveLocation(ctx, input.PickupLatitude, input.PickupLongitude)
	if err != nil {
		pickupResolved = &geography.ResolvedLocation{}
	}

	// Resolve geographic hierarchy for dropoff
	dropoffResolved, err := c.geoSvc.ResolveLocation(ctx, input.DropoffLatitude, input.DropoffLongitude)
	if err != nil {
		dropoffResolved = &geography.ResolvedLocation{}
	}

	// Extract IDs from resolved locations
	var countryID, regionID, cityID, pickupZoneID, dropoffZoneID *uuid.UUID
	if pickupResolved.Country != nil {
		countryID = &pickupResolved.Country.ID
	}
	if pickupResolved.Region != nil {
		regionID = &pickupResolved.Region.ID
	}
	if pickupResolved.City != nil {
		cityID = &pickupResolved.City.ID
	}
	if pickupResolved.PricingZone != nil {
		pickupZoneID = &pickupResolved.PricingZone.ID
	}
	if dropoffResolved.PricingZone != nil {
		dropoffZoneID = &dropoffResolved.PricingZone.ID
	}

	// Resolve pricing configuration
	pricing, err := c.resolver.Resolve(ctx, ResolveOptions{
		CountryID:  countryID,
		RegionID:   regionID,
		CityID:     cityID,
		ZoneID:     pickupZoneID,
		RideTypeID: input.RideTypeID,
	})
	if err != nil {
		return nil, err
	}

	// Get version ID
	versionID, err := c.repo.GetActiveVersionID(ctx)
	if err != nil {
		versionID = uuid.Nil
	}

	// Calculate base components
	result := &FareCalculation{
		DistanceKm:       input.DistanceKm,
		DurationMin:      input.DurationMin,
		Currency:         input.Currency,
		BaseFare:         pricing.BaseFare,
		DistanceCharge:   input.DistanceKm * pricing.PerKmRate,
		TimeCharge:       float64(input.DurationMin) * pricing.PerMinuteRate,
		BookingFee:       pricing.BookingFee,
		PricingVersionID: versionID,
	}

	// Calculate zone fees
	result.ZoneFeesTotal, result.ZoneFeesBreakdown = c.calculateZoneFees(
		ctx, versionID, pickupZoneID, dropoffZoneID, input.RideTypeID,
	)

	// Calculate multipliers
	now := time.Now()
	result.TimeMultiplier = c.getTimeMultiplier(ctx, versionID, countryID, regionID, cityID, now)
	result.WeatherMultiplier = c.getWeatherMultiplier(ctx, versionID, countryID, regionID, cityID, input.WeatherCondition)
	result.EventMultiplier = c.getEventMultiplier(ctx, versionID, cityID, pickupZoneID, now)
	result.SurgeMultiplier = c.getSurgeMultiplier(ctx, versionID, countryID, regionID, cityID, input.DemandSupplyRatio, pricing)

	// Calculate total multiplier
	result.TotalMultiplier = result.TimeMultiplier * result.WeatherMultiplier * result.EventMultiplier * result.SurgeMultiplier

	// Calculate subtotal
	baseAmount := result.BaseFare + result.DistanceCharge + result.TimeCharge + result.BookingFee
	result.Subtotal = (baseAmount * result.TotalMultiplier) + result.ZoneFeesTotal

	// Apply minimum fare
	if result.Subtotal < pricing.MinimumFare {
		result.Subtotal = pricing.MinimumFare
	}

	// Calculate tax
	result.TaxRatePct = pricing.TaxRatePct
	if pricing.TaxInclusive {
		// Tax is included in the subtotal
		result.TaxAmount = result.Subtotal - (result.Subtotal / (1 + pricing.TaxRatePct/100))
		result.TotalFare = result.Subtotal
	} else {
		// Tax is added on top
		result.TaxAmount = result.Subtotal * (pricing.TaxRatePct / 100)
		result.TotalFare = result.Subtotal + result.TaxAmount
	}

	// Check for negotiated fare
	if input.NegotiatedFare != nil {
		result.WasNegotiated = true
		result.NegotiatedFare = input.NegotiatedFare
		result.TotalFare = *input.NegotiatedFare
	}

	// Calculate commission split
	result.PlatformCommissionPct = pricing.PlatformCommissionPct
	result.PlatformCommission = result.TotalFare * (pricing.PlatformCommissionPct / 100)
	result.DriverEarnings = result.TotalFare - result.PlatformCommission

	// Round all values to 2 decimal places
	result.roundValues()

	return result, nil
}

// calculateZoneFees calculates fees for pickup and dropoff zones
func (c *Calculator) calculateZoneFees(ctx context.Context, versionID uuid.UUID, pickupZoneID, dropoffZoneID *uuid.UUID, rideTypeID *uuid.UUID) (float64, []ZoneFeeBreakdown) {
	if versionID == uuid.Nil {
		return 0, nil
	}

	fees, err := c.repo.GetZoneFees(ctx, versionID, pickupZoneID, dropoffZoneID, rideTypeID)
	if err != nil {
		return 0, nil
	}

	var total float64
	var breakdown []ZoneFeeBreakdown

	for _, fee := range fees {
		// Check if fee applies to this location
		isPickup := pickupZoneID != nil && fee.ZoneID == *pickupZoneID && fee.AppliesPickup
		isDropoff := dropoffZoneID != nil && fee.ZoneID == *dropoffZoneID && fee.AppliesDropoff

		if !isPickup && !isDropoff {
			continue
		}

		// Check schedule if applicable
		if fee.Schedule != nil && !c.isWithinSchedule(fee.Schedule) {
			continue
		}

		amount := fee.Amount
		total += amount

		// Get zone name for breakdown
		zoneName := ""
		if isPickup && pickupZoneID != nil {
			zoneName, _ = c.repo.GetZoneName(ctx, *pickupZoneID)
		} else if isDropoff && dropoffZoneID != nil {
			zoneName, _ = c.repo.GetZoneName(ctx, *dropoffZoneID)
		}

		breakdown = append(breakdown, ZoneFeeBreakdown{
			ZoneID:   fee.ZoneID,
			ZoneName: zoneName,
			FeeType:  fee.FeeType,
			Amount:   amount,
		})
	}

	return total, breakdown
}

// getTimeMultiplier retrieves the applicable time multiplier
func (c *Calculator) getTimeMultiplier(ctx context.Context, versionID uuid.UUID, countryID, regionID, cityID *uuid.UUID, t time.Time) float64 {
	if versionID == uuid.Nil {
		return 1.0
	}

	multipliers, err := c.repo.GetTimeMultipliers(ctx, versionID, countryID, regionID, cityID, t)
	if err != nil || len(multipliers) == 0 {
		return 1.0
	}

	return multipliers[0].Multiplier
}

// getWeatherMultiplier retrieves the applicable weather multiplier
func (c *Calculator) getWeatherMultiplier(ctx context.Context, versionID uuid.UUID, countryID, regionID, cityID *uuid.UUID, condition string) float64 {
	if versionID == uuid.Nil || condition == "" || condition == WeatherClear {
		return 1.0
	}

	multiplier, err := c.repo.GetWeatherMultiplier(ctx, versionID, countryID, regionID, cityID, condition)
	if err != nil {
		return 1.0
	}

	return multiplier.Multiplier
}

// getEventMultiplier retrieves the applicable event multiplier
func (c *Calculator) getEventMultiplier(ctx context.Context, versionID uuid.UUID, cityID, zoneID *uuid.UUID, t time.Time) float64 {
	if versionID == uuid.Nil {
		return 1.0
	}

	multipliers, err := c.repo.GetActiveEventMultipliers(ctx, versionID, cityID, zoneID, t)
	if err != nil || len(multipliers) == 0 {
		return 1.0
	}

	// Return highest multiplier
	return multipliers[0].Multiplier
}

// getSurgeMultiplier calculates surge based on demand/supply ratio
func (c *Calculator) getSurgeMultiplier(ctx context.Context, versionID uuid.UUID, countryID, regionID, cityID *uuid.UUID, ratio float64, pricing *ResolvedPricing) float64 {
	if versionID == uuid.Nil || ratio <= 0 {
		return 1.0
	}

	thresholds, err := c.repo.GetSurgeThresholds(ctx, versionID, countryID, regionID, cityID)
	if err != nil || len(thresholds) == 0 {
		return 1.0
	}

	// Find applicable threshold
	for _, t := range thresholds {
		if ratio >= t.DemandSupplyRatioMin {
			if t.DemandSupplyRatioMax == nil || ratio < *t.DemandSupplyRatioMax {
				return ClampSurge(t.Multiplier, pricing)
			}
		}
	}

	return 1.0
}

// isWithinSchedule checks if the current time is within a fee schedule
func (c *Calculator) isWithinSchedule(schedule *FeeSchedule) bool {
	if schedule == nil {
		return true
	}

	now := time.Now()
	dayOfWeek := int(now.Weekday())

	// Check day
	dayMatch := false
	for _, d := range schedule.Days {
		if d == dayOfWeek {
			dayMatch = true
			break
		}
	}
	if !dayMatch {
		return false
	}

	// Check time
	currentTime := now.Format("15:04")
	startTime := schedule.StartTime
	endTime := schedule.EndTime

	// Handle overnight schedules
	if startTime > endTime {
		return currentTime >= startTime || currentTime <= endTime
	}

	return currentTime >= startTime && currentTime <= endTime
}

// roundValues rounds all monetary values to 2 decimal places
func (f *FareCalculation) roundValues() {
	f.BaseFare = math.Round(f.BaseFare*100) / 100
	f.DistanceCharge = math.Round(f.DistanceCharge*100) / 100
	f.TimeCharge = math.Round(f.TimeCharge*100) / 100
	f.BookingFee = math.Round(f.BookingFee*100) / 100
	f.ZoneFeesTotal = math.Round(f.ZoneFeesTotal*100) / 100
	f.Subtotal = math.Round(f.Subtotal*100) / 100
	f.TaxAmount = math.Round(f.TaxAmount*100) / 100
	f.TotalFare = math.Round(f.TotalFare*100) / 100
	f.PlatformCommission = math.Round(f.PlatformCommission*100) / 100
	f.DriverEarnings = math.Round(f.DriverEarnings*100) / 100

	for i := range f.ZoneFeesBreakdown {
		f.ZoneFeesBreakdown[i].Amount = math.Round(f.ZoneFeesBreakdown[i].Amount*100) / 100
	}
}

// QuickEstimate performs a quick fare estimate without zone fees or event multipliers
func (c *Calculator) QuickEstimate(ctx context.Context, distanceKm float64, durationMin int, pickupLatitude, pickupLongitude float64, rideTypeID *uuid.UUID) (float64, error) {
	// Resolve location
	resolved, _ := c.geoSvc.ResolveLocation(ctx, pickupLatitude, pickupLongitude)

	var countryID, regionID, cityID *uuid.UUID
	if resolved != nil {
		if resolved.Country != nil {
			countryID = &resolved.Country.ID
		}
		if resolved.Region != nil {
			regionID = &resolved.Region.ID
		}
		if resolved.City != nil {
			cityID = &resolved.City.ID
		}
	}

	// Get pricing
	pricing, err := c.resolver.Resolve(ctx, ResolveOptions{
		CountryID:  countryID,
		RegionID:   regionID,
		CityID:     cityID,
		RideTypeID: rideTypeID,
	})
	if err != nil {
		return 0, err
	}

	// Simple calculation
	fare := pricing.BaseFare + (distanceKm * pricing.PerKmRate) + (float64(durationMin) * pricing.PerMinuteRate)

	if fare < pricing.MinimumFare {
		fare = pricing.MinimumFare
	}

	return math.Round(fare*100) / 100, nil
}

// GetPricingForLocation returns resolved pricing for a location (for API)
func (c *Calculator) GetPricingForLocation(ctx context.Context, latitude, longitude float64, rideTypeID *uuid.UUID) (*ResolvedPricing, error) {
	resolved, _ := c.geoSvc.ResolveLocation(ctx, latitude, longitude)

	var countryID, regionID, cityID, zoneID *uuid.UUID
	if resolved != nil {
		if resolved.Country != nil {
			countryID = &resolved.Country.ID
		}
		if resolved.Region != nil {
			regionID = &resolved.Region.ID
		}
		if resolved.City != nil {
			cityID = &resolved.City.ID
		}
		if resolved.PricingZone != nil {
			zoneID = &resolved.PricingZone.ID
		}
	}

	return c.resolver.Resolve(ctx, ResolveOptions{
		CountryID:  countryID,
		RegionID:   regionID,
		CityID:     cityID,
		ZoneID:     zoneID,
		RideTypeID: rideTypeID,
	})
}
