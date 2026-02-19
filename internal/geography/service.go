package geography

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Service handles geography business logic
type Service struct {
	repo RepositoryInterface
}

// NewService creates a new geography service
func NewService(repo RepositoryInterface) *Service {
	return &Service{repo: repo}
}

// GetActiveCountries returns all active countries
func (s *Service) GetActiveCountries(ctx context.Context) ([]*Country, error) {
	return s.repo.GetActiveCountries(ctx)
}

// GetCountryByCode returns a country by its ISO code
func (s *Service) GetCountryByCode(ctx context.Context, code string) (*Country, error) {
	return s.repo.GetCountryByCode(ctx, code)
}

// GetCountryByID returns a country by its ID
func (s *Service) GetCountryByID(ctx context.Context, id uuid.UUID) (*Country, error) {
	return s.repo.GetCountryByID(ctx, id)
}

// GetRegionsByCountryCode returns all active regions for a country code
func (s *Service) GetRegionsByCountryCode(ctx context.Context, countryCode string) ([]*Region, error) {
	country, err := s.repo.GetCountryByCode(ctx, countryCode)
	if err != nil {
		return nil, fmt.Errorf("country not found: %w", err)
	}

	return s.repo.GetRegionsByCountry(ctx, country.ID)
}

// GetRegionsByCountry returns all active regions for a country ID
func (s *Service) GetRegionsByCountry(ctx context.Context, countryID uuid.UUID) ([]*Region, error) {
	return s.repo.GetRegionsByCountry(ctx, countryID)
}

// GetRegionByID returns a region by its ID
func (s *Service) GetRegionByID(ctx context.Context, id uuid.UUID) (*Region, error) {
	return s.repo.GetRegionByID(ctx, id)
}

// GetCitiesByRegion returns all active cities for a region ID
func (s *Service) GetCitiesByRegion(ctx context.Context, regionID uuid.UUID) ([]*City, error) {
	return s.repo.GetCitiesByRegion(ctx, regionID)
}

// GetCityByID returns a city by its ID
func (s *Service) GetCityByID(ctx context.Context, id uuid.UUID) (*City, error) {
	return s.repo.GetCityByID(ctx, id)
}

// ResolveLocation resolves latitude/longitude to geographic hierarchy
func (s *Service) ResolveLocation(ctx context.Context, latitude, longitude float64) (*ResolvedLocation, error) {
	resolved, err := s.repo.ResolveLocation(ctx, latitude, longitude)
	if err != nil {
		return nil, err
	}

	// If no city found via boundary, try nearest city within 50km
	if resolved.City == nil {
		city, err := s.repo.FindNearestCity(ctx, latitude, longitude, 50.0)
		if err == nil && city != nil {
			// Load full hierarchy for nearest city
			fullCity, err := s.repo.GetCityByID(ctx, city.ID)
			if err == nil {
				resolved.City = fullCity
				resolved.Region = fullCity.Region
				resolved.Country = fullCity.Region.Country

				// Set timezone
				if fullCity.Timezone != nil && *fullCity.Timezone != "" {
					resolved.Timezone = *fullCity.Timezone
				} else if fullCity.Region.Timezone != nil && *fullCity.Region.Timezone != "" {
					resolved.Timezone = *fullCity.Region.Timezone
				} else {
					resolved.Timezone = fullCity.Region.Country.Timezone
				}
			}
		}
	}

	return resolved, nil
}

// GetPricingZonesByCity returns all pricing zones for a city
func (s *Service) GetPricingZonesByCity(ctx context.Context, cityID uuid.UUID) ([]*PricingZone, error) {
	return s.repo.GetPricingZonesByCity(ctx, cityID)
}

// GetDriverRegions returns all operating regions for a driver
func (s *Service) GetDriverRegions(ctx context.Context, driverID uuid.UUID) ([]*DriverRegion, error) {
	return s.repo.GetDriverRegions(ctx, driverID)
}

// AddDriverRegion adds a new operating region for a driver
func (s *Service) AddDriverRegion(ctx context.Context, driverID, regionID uuid.UUID, isPrimary bool) error {
	// Verify region exists and is active
	region, err := s.repo.GetRegionByID(ctx, regionID)
	if err != nil {
		return fmt.Errorf("region not found: %w", err)
	}

	if !region.IsActive {
		return fmt.Errorf("region %s is not active", region.Name)
	}

	return s.repo.AddDriverRegion(ctx, driverID, regionID, isPrimary)
}

// GetTimezone returns the effective timezone for a location
func (s *Service) GetTimezone(ctx context.Context, latitude, longitude float64) (string, error) {
	resolved, err := s.ResolveLocation(ctx, latitude, longitude)
	if err != nil {
		return "", err
	}

	if resolved.Timezone == "" {
		return "UTC", nil
	}

	return resolved.Timezone, nil
}

// IsLocationServiceable checks if a location is within an active service area
func (s *Service) IsLocationServiceable(ctx context.Context, latitude, longitude float64) (bool, *ResolvedLocation, error) {
	resolved, err := s.ResolveLocation(ctx, latitude, longitude)
	if err != nil {
		return false, nil, err
	}

	// Location is serviceable if we found an active city
	if resolved.City != nil && resolved.City.IsActive {
		return true, resolved, nil
	}

	return false, resolved, nil
}

// GetCurrencyForLocation returns the currency code for a location
func (s *Service) GetCurrencyForLocation(ctx context.Context, latitude, longitude float64) (string, error) {
	resolved, err := s.ResolveLocation(ctx, latitude, longitude)
	if err != nil {
		return "", err
	}

	if resolved.Country != nil {
		return resolved.Country.CurrencyCode, nil
	}

	return "USD", nil // Default fallback
}

// CreateCountry creates a new country
func (s *Service) CreateCountry(ctx context.Context, country *Country) error {
	return s.repo.CreateCountry(ctx, country)
}

// CreateRegion creates a new region
func (s *Service) CreateRegion(ctx context.Context, region *Region) error {
	// Verify country exists
	_, err := s.repo.GetCountryByID(ctx, region.CountryID)
	if err != nil {
		return fmt.Errorf("country not found: %w", err)
	}

	return s.repo.CreateRegion(ctx, region)
}

// CreateCity creates a new city
func (s *Service) CreateCity(ctx context.Context, city *City) error {
	// Verify region exists
	_, err := s.repo.GetRegionByID(ctx, city.RegionID)
	if err != nil {
		return fmt.Errorf("region not found: %w", err)
	}

	return s.repo.CreateCity(ctx, city)
}

// CreatePricingZone creates a new pricing zone
func (s *Service) CreatePricingZone(ctx context.Context, zone *PricingZone) error {
	// Verify city exists
	_, err := s.repo.GetCityByID(ctx, zone.CityID)
	if err != nil {
		return fmt.Errorf("city not found: %w", err)
	}

	return s.repo.CreatePricingZone(ctx, zone)
}

// GetGeographyStats returns aggregate stats for all geography entities
func (s *Service) GetGeographyStats(ctx context.Context) (*GeographyStats, error) {
	return s.repo.GetGeographyStats(ctx)
}

// GetAllCountries returns all countries with pagination
func (s *Service) GetAllCountries(ctx context.Context, limit, offset int, search string) ([]*Country, int64, error) {
	return s.repo.GetAllCountries(ctx, limit, offset, search)
}

// GetAllRegions returns all regions with pagination
func (s *Service) GetAllRegions(ctx context.Context, countryID *uuid.UUID, limit, offset int, search string) ([]*Region, int64, error) {
	return s.repo.GetAllRegions(ctx, countryID, limit, offset, search)
}

// GetAllCities returns all cities with pagination
func (s *Service) GetAllCities(ctx context.Context, regionID *uuid.UUID, limit, offset int, search string) ([]*City, int64, error) {
	return s.repo.GetAllCities(ctx, regionID, limit, offset, search)
}

// GetAllPricingZones returns all pricing zones with pagination
func (s *Service) GetAllPricingZones(ctx context.Context, cityID *uuid.UUID, limit, offset int, search string) ([]*PricingZone, int64, error) {
	return s.repo.GetAllPricingZones(ctx, cityID, limit, offset, search)
}

// GetPricingZoneByID returns a pricing zone by its ID
func (s *Service) GetPricingZoneByID(ctx context.Context, id uuid.UUID) (*PricingZone, error) {
	return s.repo.GetPricingZoneByID(ctx, id)
}

// UpdateCountry updates a country
func (s *Service) UpdateCountry(ctx context.Context, country *Country) error {
	return s.repo.UpdateCountry(ctx, country)
}

// DeleteCountry soft-deletes a country
func (s *Service) DeleteCountry(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteCountry(ctx, id)
}

// UpdateRegion updates a region
func (s *Service) UpdateRegion(ctx context.Context, region *Region) error {
	// Verify country exists
	_, err := s.repo.GetCountryByID(ctx, region.CountryID)
	if err != nil {
		return fmt.Errorf("country not found: %w", err)
	}
	return s.repo.UpdateRegion(ctx, region)
}

// DeleteRegion soft-deletes a region
func (s *Service) DeleteRegion(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteRegion(ctx, id)
}

// UpdateCity updates a city
func (s *Service) UpdateCity(ctx context.Context, city *City) error {
	// Verify region exists
	_, err := s.repo.GetRegionByID(ctx, city.RegionID)
	if err != nil {
		return fmt.Errorf("region not found: %w", err)
	}
	return s.repo.UpdateCity(ctx, city)
}

// DeleteCity soft-deletes a city
func (s *Service) DeleteCity(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteCity(ctx, id)
}

// UpdatePricingZone updates a pricing zone
func (s *Service) UpdatePricingZone(ctx context.Context, zone *PricingZone) error {
	// Verify city exists
	_, err := s.repo.GetCityByID(ctx, zone.CityID)
	if err != nil {
		return fmt.Errorf("city not found: %w", err)
	}
	return s.repo.UpdatePricingZone(ctx, zone)
}

// DeletePricingZone soft-deletes a pricing zone
func (s *Service) DeletePricingZone(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeletePricingZone(ctx, id)
}

// ToCountryResponse converts Country to API response
func ToCountryResponse(c *Country) *CountryResponse {
	if c == nil {
		return nil
	}

	var paymentMethods []string
	if c.PaymentMethods != nil {
		if methods, ok := c.PaymentMethods["methods"].([]interface{}); ok {
			for _, m := range methods {
				if str, ok := m.(string); ok {
					paymentMethods = append(paymentMethods, str)
				}
			}
		}
	}

	return &CountryResponse{
		ID:             c.ID,
		Code:           c.Code,
		Name:           c.Name,
		NativeName:     c.NativeName,
		CurrencyCode:   c.CurrencyCode,
		Timezone:       c.Timezone,
		PhonePrefix:    c.PhonePrefix,
		IsActive:       c.IsActive,
		PaymentMethods: paymentMethods,
	}
}

// ToRegionResponse converts Region to API response
func ToRegionResponse(r *Region) *RegionResponse {
	if r == nil {
		return nil
	}

	timezone := ""
	if r.Timezone != nil {
		timezone = *r.Timezone
	} else if r.Country != nil {
		timezone = r.Country.Timezone
	}

	return &RegionResponse{
		ID:         r.ID,
		Code:       r.Code,
		Name:       r.Name,
		NativeName: r.NativeName,
		Timezone:   timezone,
		IsActive:   r.IsActive,
		Country:    ToCountryResponse(r.Country),
	}
}

// ToCityResponse converts City to API response
func ToCityResponse(c *City) *CityResponse {
	if c == nil {
		return nil
	}

	timezone := ""
	if c.Timezone != nil {
		timezone = *c.Timezone
	} else if c.Region != nil && c.Region.Timezone != nil {
		timezone = *c.Region.Timezone
	} else if c.Region != nil && c.Region.Country != nil {
		timezone = c.Region.Country.Timezone
	}

	return &CityResponse{
		ID:              c.ID,
		Name:            c.Name,
		NativeName:      c.NativeName,
		Timezone:        timezone,
		CenterLatitude:  c.CenterLatitude,
		CenterLongitude: c.CenterLongitude,
		Population:      c.Population,
		IsActive:        c.IsActive,
		Region:          ToRegionResponse(c.Region),
	}
}

// ToPricingZoneResponse converts PricingZone to API response
func ToPricingZoneResponse(z *PricingZone) *PricingZoneResponse {
	if z == nil {
		return nil
	}

	return &PricingZoneResponse{
		ID:       z.ID,
		Name:     z.Name,
		ZoneType: z.ZoneType,
		Priority: z.Priority,
	}
}

// ToResolveLocationResponse converts ResolvedLocation to API response
func ToResolveLocationResponse(r *ResolvedLocation) *ResolveLocationResponse {
	if r == nil {
		return nil
	}

	return &ResolveLocationResponse{
		Country:     ToCountryResponse(r.Country),
		Region:      ToRegionResponse(r.Region),
		City:        ToCityResponse(r.City),
		PricingZone: ToPricingZoneResponse(r.PricingZone),
		Timezone:    r.Timezone,
	}
}

