package pricing

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/internal/currency"
	"github.com/richxcame/ride-hailing/internal/geography"
)

// Service handles pricing business logic
type Service struct {
	repo        RepositoryInterface
	resolver    *Resolver
	calculator  *Calculator
	geoSvc      *geography.Service
	currencySvc *currency.Service
}

// NewService creates a new pricing service
func NewService(repo RepositoryInterface, geoSvc *geography.Service, currencySvc *currency.Service) *Service {
	resolver := NewResolver(repo)
	calculator := NewCalculator(repo, resolver, geoSvc)

	return &Service{
		repo:        repo,
		resolver:    resolver,
		calculator:  calculator,
		geoSvc:      geoSvc,
		currencySvc: currencySvc,
	}
}

// CalculateFare calculates the fare for a ride
func (s *Service) CalculateFare(ctx context.Context, input CalculateInput) (*FareCalculation, error) {
	return s.calculator.Calculate(ctx, input)
}

// GetEstimate returns a fare estimate
func (s *Service) GetEstimate(ctx context.Context, req EstimateRequest) (*EstimateResponse, error) {
	// Calculate distance
	distanceKm := haversineDistance(
		req.PickupLatitude, req.PickupLongitude,
		req.DropoffLatitude, req.DropoffLongitude,
	)

	// Estimate duration (simple calculation: 40 km/h average)
	durationMin := int(math.Ceil((distanceKm / 40.0) * 60))
	if durationMin < 1 {
		durationMin = 1
	}

	// Get currency from pickup location
	currencyCode := "USD"
	if s.geoSvc != nil {
		if curr, err := s.geoSvc.GetCurrencyForLocation(ctx, req.PickupLatitude, req.PickupLongitude); err == nil {
			currencyCode = curr
		}
	}

	// Calculate fare
	calculation, err := s.calculator.Calculate(ctx, CalculateInput{
		PickupLatitude:   req.PickupLatitude,
		PickupLongitude:  req.PickupLongitude,
		DropoffLatitude:  req.DropoffLatitude,
		DropoffLongitude: req.DropoffLongitude,
		DistanceKm:       distanceKm,
		DurationMin:      durationMin,
		RideTypeID:       req.RideTypeID,
		Currency:         currencyCode,
	})
	if err != nil {
		return nil, err
	}

	// Get pricing for minimum fare
	pricing, _ := s.calculator.GetPricingForLocation(ctx, req.PickupLatitude, req.PickupLongitude, req.RideTypeID)
	minFare := DefaultPricing.MinimumFare
	if pricing != nil {
		minFare = pricing.MinimumFare
	}

	// Format the fare
	formattedFare := fmt.Sprintf("%.2f %s", calculation.TotalFare, currencyCode)
	if s.currencySvc != nil {
		if formatted, err := s.currencySvc.FormatMoney(ctx, currency.Money{
			Amount:   calculation.TotalFare,
			Currency: currencyCode,
		}); err == nil {
			formattedFare = formatted
		}
	}

	return &EstimateResponse{
		Currency:         currencyCode,
		EstimatedFare:    calculation.TotalFare,
		MinimumFare:      minFare,
		SurgeMultiplier:  calculation.TotalMultiplier,
		DistanceKm:       distanceKm,
		EstimatedMinutes: durationMin,
		FareBreakdown:    calculation,
		FormattedFare:    formattedFare,
	}, nil
}

// GetPricing returns the resolved pricing for a location
func (s *Service) GetPricing(ctx context.Context, lat, lng float64, rideTypeID *uuid.UUID) (*ResolvedPricing, error) {
	return s.calculator.GetPricingForLocation(ctx, lat, lng, rideTypeID)
}

// GetBulkEstimate returns fare estimates for all available ride types at a location
func (s *Service) GetBulkEstimate(ctx context.Context, req BulkEstimateRequest, rideTypes []RideTypeInfo) (*BulkEstimateResponse, error) {
	// Calculate distance once (same for all ride types)
	distanceKm := haversineDistance(
		req.PickupLatitude, req.PickupLongitude,
		req.DropoffLatitude, req.DropoffLongitude,
	)

	// Estimate duration once (same for all ride types)
	durationMin := int(math.Ceil((distanceKm / 40.0) * 60))
	if durationMin < 1 {
		durationMin = 1
	}

	// Get currency from pickup location
	currencyCode := "USD"
	if s.geoSvc != nil {
		if curr, err := s.geoSvc.GetCurrencyForLocation(ctx, req.PickupLatitude, req.PickupLongitude); err == nil {
			currencyCode = curr
		}
	}

	// Build estimates for each ride type
	estimates := make([]RideTypeEstimate, 0, len(rideTypes))
	for _, rideType := range rideTypes {
		// Calculate fare for this ride type
		rideTypeID := rideType.ID
		calculation, err := s.calculator.Calculate(ctx, CalculateInput{
			PickupLatitude:   req.PickupLatitude,
			PickupLongitude:  req.PickupLongitude,
			DropoffLatitude:  req.DropoffLatitude,
			DropoffLongitude: req.DropoffLongitude,
			DistanceKm:       distanceKm,
			DurationMin:      durationMin,
			RideTypeID:       &rideTypeID,
			Currency:         currencyCode,
		})
		if err != nil {
			// Skip ride types that fail calculation
			continue
		}

		// Get pricing for minimum fare
		pricing, _ := s.calculator.GetPricingForLocation(ctx, req.PickupLatitude, req.PickupLongitude, &rideTypeID)
		minFare := DefaultPricing.MinimumFare
		if pricing != nil {
			minFare = pricing.MinimumFare
		}

		// Format the fare
		formattedFare := fmt.Sprintf("%.2f %s", calculation.TotalFare, currencyCode)
		if s.currencySvc != nil {
			if formatted, err := s.currencySvc.FormatMoney(ctx, currency.Money{
				Amount:   calculation.TotalFare,
				Currency: currencyCode,
			}); err == nil {
				formattedFare = formatted
			}
		}

		estimates = append(estimates, RideTypeEstimate{
			RideTypeID:      rideType.ID,
			RideTypeName:    rideType.Name,
			Description:     rideType.Description,
			Capacity:        rideType.Capacity,
			IconURL:         rideType.IconURL,
			Currency:        currencyCode,
			EstimatedFare:   calculation.TotalFare,
			MinimumFare:     minFare,
			SurgeMultiplier: calculation.TotalMultiplier,
			FareBreakdown:   calculation,
			FormattedFare:   formattedFare,
		})
	}

	return &BulkEstimateResponse{
		DistanceKm:       distanceKm,
		EstimatedMinutes: durationMin,
		RideOptions:      estimates,
	}, nil
}

// ValidateNegotiatedPrice checks if a negotiated price is within acceptable range
func (s *Service) ValidateNegotiatedPrice(ctx context.Context, req EstimateRequest, negotiatedPrice float64) error {
	estimate, err := s.GetEstimate(ctx, req)
	if err != nil {
		return err
	}

	// Allow negotiated price to be between 70% and 150% of estimated fare
	minAllowed := estimate.EstimatedFare * 0.70
	maxAllowed := estimate.EstimatedFare * 1.50

	if negotiatedPrice < minAllowed {
		return fmt.Errorf("negotiated price %.2f is below minimum allowed %.2f", negotiatedPrice, minAllowed)
	}

	if negotiatedPrice > maxAllowed {
		return fmt.Errorf("negotiated price %.2f exceeds maximum allowed %.2f", negotiatedPrice, maxAllowed)
	}

	return nil
}

// GetSurgeInfo returns current surge information for a location
func (s *Service) GetSurgeInfo(ctx context.Context, lat, lng float64) (map[string]interface{}, error) {
	// Resolve location
	resolved, _ := s.geoSvc.ResolveLocation(ctx, lat, lng)

	// Get pricing
	pricing, err := s.calculator.GetPricingForLocation(ctx, lat, lng, nil)
	if err != nil {
		pricing = &DefaultPricing
	}

	info := map[string]interface{}{
		"current_surge":     1.0, // Would be calculated from real-time data
		"surge_min":         pricing.SurgeMinMultiplier,
		"surge_max":         pricing.SurgeMaxMultiplier,
		"surge_active":      false,
		"surge_reason":      nil,
		"expected_duration": nil,
	}

	if resolved != nil {
		if resolved.City != nil {
			info["city"] = resolved.City.Name
		}
		if resolved.PricingZone != nil {
			info["zone"] = resolved.PricingZone.Name
		}
	}

	return info, nil
}

// GetCancellationFee returns the cancellation fee for a ride
func (s *Service) GetCancellationFee(ctx context.Context, lat, lng float64, minutesSinceRequest float64, estimatedFare float64) (float64, error) {
	pricing, err := s.calculator.GetPricingForLocation(ctx, lat, lng, nil)
	if err != nil {
		pricing = &DefaultPricing
	}

	return GetCancellationFee(pricing, minutesSinceRequest, estimatedFare), nil
}

// GetCommissionRate returns the commission rate for a location
func (s *Service) GetCommissionRate(ctx context.Context, lat, lng float64) (float64, error) {
	pricing, err := s.calculator.GetPricingForLocation(ctx, lat, lng, nil)
	if err != nil {
		return DefaultPricing.PlatformCommissionPct / 100, nil
	}
	return pricing.PlatformCommissionPct / 100, nil
}

// CalculateDriverEarnings calculates driver earnings from a fare
func (s *Service) CalculateDriverEarnings(ctx context.Context, lat, lng float64, fare float64) (float64, error) {
	pricing, err := s.calculator.GetPricingForLocation(ctx, lat, lng, nil)
	if err != nil {
		pricing = &DefaultPricing
	}

	commission := fare * (pricing.PlatformCommissionPct / 100)
	return fare - commission, nil
}

// haversineDistance calculates the distance between two points in km
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth's radius in km

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}
