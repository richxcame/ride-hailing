package ridetypes

import (
	"context"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/internal/geography"
)

// Service handles ride type business logic
type Service struct {
	repo   RepositoryInterface
	geoSvc *geography.Service
}

// NewService creates a new ride types service
func NewService(repo RepositoryInterface, geoSvc *geography.Service) *Service {
	return &Service{repo: repo, geoSvc: geoSvc}
}

// --- Ride Types ---

// CreateRideType creates a new ride type
func (s *Service) CreateRideType(ctx context.Context, rt *RideType) error {
	return s.repo.CreateRideType(ctx, rt)
}

// GetRideTypeByID returns a ride type by ID
func (s *Service) GetRideTypeByID(ctx context.Context, id uuid.UUID) (*RideType, error) {
	return s.repo.GetRideTypeByID(ctx, id)
}

// ListRideTypes returns ride types with pagination
func (s *Service) ListRideTypes(ctx context.Context, limit, offset int, includeInactive bool) ([]*RideType, int64, error) {
	return s.repo.ListRideTypes(ctx, limit, offset, includeInactive)
}

// UpdateRideType updates a ride type
func (s *Service) UpdateRideType(ctx context.Context, rt *RideType) error {
	return s.repo.UpdateRideType(ctx, rt)
}

// DeleteRideType soft-deletes a ride type
func (s *Service) DeleteRideType(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteRideType(ctx, id)
}

// --- Country Ride Types ---

// AddRideTypeToCountry adds a ride type to a country
func (s *Service) AddRideTypeToCountry(ctx context.Context, crt *CountryRideType) error {
	return s.repo.AddRideTypeToCountry(ctx, crt)
}

// UpdateCountryRideType updates a country ride type mapping
func (s *Service) UpdateCountryRideType(ctx context.Context, crt *CountryRideType) error {
	return s.repo.UpdateCountryRideType(ctx, crt)
}

// RemoveRideTypeFromCountry removes a ride type from a country
func (s *Service) RemoveRideTypeFromCountry(ctx context.Context, countryID, rideTypeID uuid.UUID) error {
	return s.repo.RemoveRideTypeFromCountry(ctx, countryID, rideTypeID)
}

// ListCountryRideTypes lists ride types for a country
func (s *Service) ListCountryRideTypes(ctx context.Context, countryID uuid.UUID, includeInactive bool) ([]*CountryRideTypeWithDetails, error) {
	return s.repo.ListCountryRideTypes(ctx, countryID, includeInactive)
}

// --- City Ride Types ---

// AddRideTypeToCity adds a ride type to a city
func (s *Service) AddRideTypeToCity(ctx context.Context, crt *CityRideType) error {
	return s.repo.AddRideTypeToCity(ctx, crt)
}

// UpdateCityRideType updates a city ride type mapping
func (s *Service) UpdateCityRideType(ctx context.Context, crt *CityRideType) error {
	return s.repo.UpdateCityRideType(ctx, crt)
}

// RemoveRideTypeFromCity removes a ride type from a city
func (s *Service) RemoveRideTypeFromCity(ctx context.Context, cityID, rideTypeID uuid.UUID) error {
	return s.repo.RemoveRideTypeFromCity(ctx, cityID, rideTypeID)
}

// ListCityRideTypes lists ride types for a city
func (s *Service) ListCityRideTypes(ctx context.Context, cityID uuid.UUID, includeInactive bool) ([]*CityRideTypeWithDetails, error) {
	return s.repo.ListCityRideTypes(ctx, cityID, includeInactive)
}

// --- Mobile-facing ---

// GetAvailableRideTypes returns ride types available at a given location.
// Uses cascading resolution:
//   1. Resolve lat/lng → country + city via geography service
//   2. If city has explicit ride type mappings → use those
//   3. Else if country has ride type mappings → use those (available in all cities)
//   4. Else → return all active ride types globally
func (s *Service) GetAvailableRideTypes(ctx context.Context, lat, lng float64) ([]*RideType, error) {
	var countryID, cityID *uuid.UUID

	if s.geoSvc != nil {
		resolved, err := s.geoSvc.ResolveLocation(ctx, lat, lng)
		if err == nil && resolved != nil {
			if resolved.Country != nil {
				countryID = &resolved.Country.ID
			}
			if resolved.City != nil {
				cityID = &resolved.City.ID
			}
		}
	}

	return s.repo.GetAvailableRideTypes(ctx, countryID, cityID)
}
