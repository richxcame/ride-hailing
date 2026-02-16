package geography

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryInterface defines the interface for geography repository operations
type RepositoryInterface interface {
	// Read operations
	GetActiveCountries(ctx context.Context) ([]*Country, error)
	GetCountryByCode(ctx context.Context, code string) (*Country, error)
	GetCountryByID(ctx context.Context, id uuid.UUID) (*Country, error)
	GetRegionsByCountry(ctx context.Context, countryID uuid.UUID) ([]*Region, error)
	GetRegionByID(ctx context.Context, id uuid.UUID) (*Region, error)
	GetCitiesByRegion(ctx context.Context, regionID uuid.UUID) ([]*City, error)
	GetCityByID(ctx context.Context, id uuid.UUID) (*City, error)
	ResolveLocation(ctx context.Context, latitude, longitude float64) (*ResolvedLocation, error)
	FindNearestCity(ctx context.Context, latitude, longitude float64, maxDistanceKm float64) (*City, error)
	GetPricingZonesByCity(ctx context.Context, cityID uuid.UUID) ([]*PricingZone, error)
	GetPricingZoneByID(ctx context.Context, id uuid.UUID) (*PricingZone, error)
	GetDriverRegions(ctx context.Context, driverID uuid.UUID) ([]*DriverRegion, error)

	// Stats
	GetGeographyStats(ctx context.Context) (*GeographyStats, error)

	// Paginated list operations (admin)
	GetAllCountries(ctx context.Context, limit, offset int, search string) ([]*Country, int64, error)
	GetAllRegions(ctx context.Context, countryID *uuid.UUID, limit, offset int, search string) ([]*Region, int64, error)
	GetAllCities(ctx context.Context, regionID *uuid.UUID, limit, offset int, search string) ([]*City, int64, error)
	GetAllPricingZones(ctx context.Context, cityID *uuid.UUID, limit, offset int, search string) ([]*PricingZone, int64, error)

	// Write operations
	AddDriverRegion(ctx context.Context, driverID, regionID uuid.UUID, isPrimary bool) error
	CreateCountry(ctx context.Context, country *Country) error
	CreateRegion(ctx context.Context, region *Region) error
	CreateCity(ctx context.Context, city *City) error
	CreatePricingZone(ctx context.Context, zone *PricingZone) error

	// Update operations
	UpdateCountry(ctx context.Context, country *Country) error
	UpdateRegion(ctx context.Context, region *Region) error
	UpdateCity(ctx context.Context, city *City) error
	UpdatePricingZone(ctx context.Context, zone *PricingZone) error

	// Delete operations (soft-delete)
	DeleteCountry(ctx context.Context, id uuid.UUID) error
	DeleteRegion(ctx context.Context, id uuid.UUID) error
	DeleteCity(ctx context.Context, id uuid.UUID) error
	DeletePricingZone(ctx context.Context, id uuid.UUID) error
}
