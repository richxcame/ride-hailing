package ridetypes

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryInterface defines the interface for ride type repository operations
type RepositoryInterface interface {
	// Ride Types (global product catalog)
	CreateRideType(ctx context.Context, rt *RideType) error
	GetRideTypeByID(ctx context.Context, id uuid.UUID) (*RideType, error)
	ListRideTypes(ctx context.Context, limit, offset int, includeInactive bool) ([]*RideType, int64, error)
	UpdateRideType(ctx context.Context, rt *RideType) error
	DeleteRideType(ctx context.Context, id uuid.UUID) error

	// Country Ride Types (country-level availability)
	AddRideTypeToCountry(ctx context.Context, crt *CountryRideType) error
	UpdateCountryRideType(ctx context.Context, crt *CountryRideType) error
	RemoveRideTypeFromCountry(ctx context.Context, countryID, rideTypeID uuid.UUID) error
	ListCountryRideTypes(ctx context.Context, countryID uuid.UUID, includeInactive bool) ([]*CountryRideTypeWithDetails, error)

	// City Ride Types (city-level override)
	AddRideTypeToCity(ctx context.Context, crt *CityRideType) error
	UpdateCityRideType(ctx context.Context, crt *CityRideType) error
	RemoveRideTypeFromCity(ctx context.Context, cityID, rideTypeID uuid.UUID) error
	ListCityRideTypes(ctx context.Context, cityID uuid.UUID, includeInactive bool) ([]*CityRideTypeWithDetails, error)

	// Cascading availability resolution
	GetAvailableRideTypes(ctx context.Context, countryID, cityID *uuid.UUID) ([]*RideType, error)
}
