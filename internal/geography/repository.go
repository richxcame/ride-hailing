package geography

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for geography
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new geography repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// GetActiveCountries retrieves all active countries
func (r *Repository) GetActiveCountries(ctx context.Context) ([]*Country, error) {
	query := `
		SELECT id, code, code3, name, native_name, currency_code, default_language,
		       timezone, phone_prefix, is_active, launched_at, regulations,
		       payment_methods, required_driver_documents, created_at, updated_at
		FROM countries
		WHERE is_active = true
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get countries: %w", err)
	}
	defer rows.Close()

	countries := make([]*Country, 0)
	for rows.Next() {
		c := &Country{}
		err := rows.Scan(
			&c.ID, &c.Code, &c.Code3, &c.Name, &c.NativeName, &c.CurrencyCode,
			&c.DefaultLanguage, &c.Timezone, &c.PhonePrefix, &c.IsActive,
			&c.LaunchedAt, &c.Regulations, &c.PaymentMethods,
			&c.RequiredDriverDocuments, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan country: %w", err)
		}
		countries = append(countries, c)
	}

	return countries, nil
}

// GetCountryByCode retrieves a country by its ISO code
func (r *Repository) GetCountryByCode(ctx context.Context, code string) (*Country, error) {
	query := `
		SELECT id, code, code3, name, native_name, currency_code, default_language,
		       timezone, phone_prefix, is_active, launched_at, regulations,
		       payment_methods, required_driver_documents, created_at, updated_at
		FROM countries
		WHERE code = $1
	`

	c := &Country{}
	err := r.db.QueryRow(ctx, query, code).Scan(
		&c.ID, &c.Code, &c.Code3, &c.Name, &c.NativeName, &c.CurrencyCode,
		&c.DefaultLanguage, &c.Timezone, &c.PhonePrefix, &c.IsActive,
		&c.LaunchedAt, &c.Regulations, &c.PaymentMethods,
		&c.RequiredDriverDocuments, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get country: %w", err)
	}

	return c, nil
}

// GetCountryByID retrieves a country by its ID
func (r *Repository) GetCountryByID(ctx context.Context, id uuid.UUID) (*Country, error) {
	query := `
		SELECT id, code, code3, name, native_name, currency_code, default_language,
		       timezone, phone_prefix, is_active, launched_at, regulations,
		       payment_methods, required_driver_documents, created_at, updated_at
		FROM countries
		WHERE id = $1
	`

	c := &Country{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Code, &c.Code3, &c.Name, &c.NativeName, &c.CurrencyCode,
		&c.DefaultLanguage, &c.Timezone, &c.PhonePrefix, &c.IsActive,
		&c.LaunchedAt, &c.Regulations, &c.PaymentMethods,
		&c.RequiredDriverDocuments, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get country: %w", err)
	}

	return c, nil
}

// GetRegionsByCountry retrieves all active regions for a country
func (r *Repository) GetRegionsByCountry(ctx context.Context, countryID uuid.UUID) ([]*Region, error) {
	query := `
		SELECT id, country_id, code, name, native_name, timezone, is_active,
		       launched_at, created_at, updated_at
		FROM regions
		WHERE country_id = $1 AND is_active = true
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query, countryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get regions: %w", err)
	}
	defer rows.Close()

	regions := make([]*Region, 0)
	for rows.Next() {
		reg := &Region{}
		err := rows.Scan(
			&reg.ID, &reg.CountryID, &reg.Code, &reg.Name, &reg.NativeName,
			&reg.Timezone, &reg.IsActive, &reg.LaunchedAt, &reg.CreatedAt, &reg.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan region: %w", err)
		}
		regions = append(regions, reg)
	}

	return regions, nil
}

// GetRegionByID retrieves a region by its ID with country
func (r *Repository) GetRegionByID(ctx context.Context, id uuid.UUID) (*Region, error) {
	query := `
		SELECT r.id, r.country_id, r.code, r.name, r.native_name, r.timezone,
		       r.is_active, r.launched_at, r.created_at, r.updated_at,
		       c.id, c.code, c.code3, c.name, c.native_name, c.currency_code,
		       c.default_language, c.timezone, c.phone_prefix, c.is_active,
		       c.launched_at, c.regulations, c.payment_methods,
		       c.required_driver_documents, c.created_at, c.updated_at
		FROM regions r
		JOIN countries c ON r.country_id = c.id
		WHERE r.id = $1
	`

	reg := &Region{Country: &Country{}}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&reg.ID, &reg.CountryID, &reg.Code, &reg.Name, &reg.NativeName,
		&reg.Timezone, &reg.IsActive, &reg.LaunchedAt, &reg.CreatedAt, &reg.UpdatedAt,
		&reg.Country.ID, &reg.Country.Code, &reg.Country.Code3, &reg.Country.Name,
		&reg.Country.NativeName, &reg.Country.CurrencyCode, &reg.Country.DefaultLanguage,
		&reg.Country.Timezone, &reg.Country.PhonePrefix, &reg.Country.IsActive,
		&reg.Country.LaunchedAt, &reg.Country.Regulations, &reg.Country.PaymentMethods,
		&reg.Country.RequiredDriverDocuments, &reg.Country.CreatedAt, &reg.Country.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get region: %w", err)
	}

	return reg, nil
}

// GetCitiesByRegion retrieves all active cities for a region
func (r *Repository) GetCitiesByRegion(ctx context.Context, regionID uuid.UUID) ([]*City, error) {
	query := `
		SELECT id, region_id, name, native_name, timezone, center_latitude,
		       center_longitude, ST_AsText(boundary), population, is_active,
		       launched_at, created_at, updated_at
		FROM cities
		WHERE region_id = $1 AND is_active = true
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query, regionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cities: %w", err)
	}
	defer rows.Close()

	cities := make([]*City, 0)
	for rows.Next() {
		city := &City{}
		err := rows.Scan(
			&city.ID, &city.RegionID, &city.Name, &city.NativeName, &city.Timezone,
			&city.CenterLatitude, &city.CenterLongitude, &city.Boundary,
			&city.Population, &city.IsActive, &city.LaunchedAt,
			&city.CreatedAt, &city.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan city: %w", err)
		}
		cities = append(cities, city)
	}

	return cities, nil
}

// GetCityByID retrieves a city by its ID with region and country
func (r *Repository) GetCityByID(ctx context.Context, id uuid.UUID) (*City, error) {
	query := `
		SELECT city.id, city.region_id, city.name, city.native_name, city.timezone,
		       city.center_latitude, city.center_longitude, ST_AsText(city.boundary),
		       city.population, city.is_active, city.launched_at, city.created_at, city.updated_at,
		       reg.id, reg.country_id, reg.code, reg.name, reg.native_name, reg.timezone,
		       reg.is_active, reg.launched_at, reg.created_at, reg.updated_at,
		       c.id, c.code, c.code3, c.name, c.native_name, c.currency_code,
		       c.default_language, c.timezone, c.phone_prefix, c.is_active,
		       c.launched_at, c.regulations, c.payment_methods,
		       c.required_driver_documents, c.created_at, c.updated_at
		FROM cities city
		JOIN regions reg ON city.region_id = reg.id
		JOIN countries c ON reg.country_id = c.id
		WHERE city.id = $1
	`

	city := &City{Region: &Region{Country: &Country{}}}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&city.ID, &city.RegionID, &city.Name, &city.NativeName, &city.Timezone,
		&city.CenterLatitude, &city.CenterLongitude, &city.Boundary,
		&city.Population, &city.IsActive, &city.LaunchedAt, &city.CreatedAt, &city.UpdatedAt,
		&city.Region.ID, &city.Region.CountryID, &city.Region.Code, &city.Region.Name,
		&city.Region.NativeName, &city.Region.Timezone, &city.Region.IsActive,
		&city.Region.LaunchedAt, &city.Region.CreatedAt, &city.Region.UpdatedAt,
		&city.Region.Country.ID, &city.Region.Country.Code, &city.Region.Country.Code3,
		&city.Region.Country.Name, &city.Region.Country.NativeName,
		&city.Region.Country.CurrencyCode, &city.Region.Country.DefaultLanguage,
		&city.Region.Country.Timezone, &city.Region.Country.PhonePrefix,
		&city.Region.Country.IsActive, &city.Region.Country.LaunchedAt,
		&city.Region.Country.Regulations, &city.Region.Country.PaymentMethods,
		&city.Region.Country.RequiredDriverDocuments, &city.Region.Country.CreatedAt,
		&city.Region.Country.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get city: %w", err)
	}

	return city, nil
}

// ResolveLocation finds the geographic hierarchy for a latitude/longitude point
func (r *Repository) ResolveLocation(ctx context.Context, latitude, longitude float64) (*ResolvedLocation, error) {
	result := &ResolvedLocation{
		Location: Location{Latitude: latitude, Longitude: longitude},
	}

	// Find city containing the point (uses PostGIS ST_Contains)
	cityQuery := `
		SELECT city.id, city.region_id, city.name, city.native_name, city.timezone,
		       city.center_latitude, city.center_longitude, ST_AsText(city.boundary),
		       city.population, city.is_active, city.launched_at, city.created_at, city.updated_at,
		       reg.id, reg.country_id, reg.code, reg.name, reg.native_name, reg.timezone,
		       reg.is_active, reg.launched_at, reg.created_at, reg.updated_at,
		       c.id, c.code, c.code3, c.name, c.native_name, c.currency_code,
		       c.default_language, c.timezone, c.phone_prefix, c.is_active,
		       c.launched_at, c.regulations, c.payment_methods,
		       c.required_driver_documents, c.created_at, c.updated_at
		FROM cities city
		JOIN regions reg ON city.region_id = reg.id
		JOIN countries c ON reg.country_id = c.id
		WHERE city.is_active = true
		  AND reg.is_active = true
		  AND c.is_active = true
		  AND ST_Contains(city.boundary, ST_SetSRID(ST_MakePoint($1, $2), 4326))
		ORDER BY city.population DESC NULLS LAST
		LIMIT 1
	`

	city := &City{Region: &Region{Country: &Country{}}}
	err := r.db.QueryRow(ctx, cityQuery, longitude, latitude).Scan(
		&city.ID, &city.RegionID, &city.Name, &city.NativeName, &city.Timezone,
		&city.CenterLatitude, &city.CenterLongitude, &city.Boundary,
		&city.Population, &city.IsActive, &city.LaunchedAt, &city.CreatedAt, &city.UpdatedAt,
		&city.Region.ID, &city.Region.CountryID, &city.Region.Code, &city.Region.Name,
		&city.Region.NativeName, &city.Region.Timezone, &city.Region.IsActive,
		&city.Region.LaunchedAt, &city.Region.CreatedAt, &city.Region.UpdatedAt,
		&city.Region.Country.ID, &city.Region.Country.Code, &city.Region.Country.Code3,
		&city.Region.Country.Name, &city.Region.Country.NativeName,
		&city.Region.Country.CurrencyCode, &city.Region.Country.DefaultLanguage,
		&city.Region.Country.Timezone, &city.Region.Country.PhonePrefix,
		&city.Region.Country.IsActive, &city.Region.Country.LaunchedAt,
		&city.Region.Country.Regulations, &city.Region.Country.PaymentMethods,
		&city.Region.Country.RequiredDriverDocuments, &city.Region.Country.CreatedAt,
		&city.Region.Country.UpdatedAt,
	)
	if err == nil {
		result.City = city
		result.Region = city.Region
		result.Country = city.Region.Country

		// Resolve timezone (city > region > country)
		if city.Timezone != nil && *city.Timezone != "" {
			result.Timezone = *city.Timezone
		} else if city.Region.Timezone != nil && *city.Region.Timezone != "" {
			result.Timezone = *city.Region.Timezone
		} else {
			result.Timezone = city.Region.Country.Timezone
		}
	}

	// Find pricing zone containing the point (highest priority)
	if result.City != nil {
		zoneQuery := `
			SELECT id, city_id, name, zone_type, ST_AsText(boundary),
			       center_latitude, center_longitude, priority, is_active,
			       metadata, created_at, updated_at
			FROM pricing_zones
			WHERE city_id = $1
			  AND is_active = true
			  AND ST_Contains(boundary, ST_SetSRID(ST_MakePoint($2, $3), 4326))
			ORDER BY priority DESC
			LIMIT 1
		`

		zone := &PricingZone{}
		err := r.db.QueryRow(ctx, zoneQuery, result.City.ID, longitude, latitude).Scan(
			&zone.ID, &zone.CityID, &zone.Name, &zone.ZoneType, &zone.Boundary,
			&zone.CenterLatitude, &zone.CenterLongitude, &zone.Priority,
			&zone.IsActive, &zone.Metadata, &zone.CreatedAt, &zone.UpdatedAt,
		)
		if err == nil {
			result.PricingZone = zone
		}
	}

	return result, nil
}

// FindNearestCity finds the nearest city to a point
func (r *Repository) FindNearestCity(ctx context.Context, latitude, longitude float64, maxDistanceKm float64) (*City, error) {
	query := `
		SELECT city.id, city.region_id, city.name, city.native_name, city.timezone,
		       city.center_latitude, city.center_longitude, ST_AsText(city.boundary),
		       city.population, city.is_active, city.launched_at, city.created_at, city.updated_at,
		       ST_Distance(
		           ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
		           ST_SetSRID(ST_MakePoint(city.center_longitude, city.center_latitude), 4326)::geography
		       ) / 1000 AS distance_km
		FROM cities city
		JOIN regions reg ON city.region_id = reg.id
		JOIN countries c ON reg.country_id = c.id
		WHERE city.is_active = true
		  AND reg.is_active = true
		  AND c.is_active = true
		ORDER BY distance_km
		LIMIT 1
	`

	city := &City{}
	var distanceKm float64
	err := r.db.QueryRow(ctx, query, longitude, latitude).Scan(
		&city.ID, &city.RegionID, &city.Name, &city.NativeName, &city.Timezone,
		&city.CenterLatitude, &city.CenterLongitude, &city.Boundary,
		&city.Population, &city.IsActive, &city.LaunchedAt, &city.CreatedAt, &city.UpdatedAt,
		&distanceKm,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearest city: %w", err)
	}

	if distanceKm > maxDistanceKm {
		return nil, fmt.Errorf("nearest city is %.2f km away, exceeds max %.2f km", distanceKm, maxDistanceKm)
	}

	return city, nil
}

// GetPricingZonesByCity retrieves all active pricing zones for a city
func (r *Repository) GetPricingZonesByCity(ctx context.Context, cityID uuid.UUID) ([]*PricingZone, error) {
	query := `
		SELECT id, city_id, name, zone_type, ST_AsText(boundary),
		       center_latitude, center_longitude, priority, is_active,
		       metadata, created_at, updated_at
		FROM pricing_zones
		WHERE city_id = $1 AND is_active = true
		ORDER BY priority DESC, name
	`

	rows, err := r.db.Query(ctx, query, cityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing zones: %w", err)
	}
	defer rows.Close()

	zones := make([]*PricingZone, 0)
	for rows.Next() {
		zone := &PricingZone{}
		err := rows.Scan(
			&zone.ID, &zone.CityID, &zone.Name, &zone.ZoneType, &zone.Boundary,
			&zone.CenterLatitude, &zone.CenterLongitude, &zone.Priority,
			&zone.IsActive, &zone.Metadata, &zone.CreatedAt, &zone.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pricing zone: %w", err)
		}
		zones = append(zones, zone)
	}

	return zones, nil
}

// GetDriverRegions retrieves all operating regions for a driver
func (r *Repository) GetDriverRegions(ctx context.Context, driverID uuid.UUID) ([]*DriverRegion, error) {
	query := `
		SELECT dr.id, dr.driver_id, dr.region_id, dr.is_primary, dr.is_verified,
		       dr.verified_at, dr.created_at,
		       reg.id, reg.country_id, reg.code, reg.name, reg.native_name,
		       reg.timezone, reg.is_active, reg.launched_at, reg.created_at, reg.updated_at
		FROM driver_regions dr
		JOIN regions reg ON dr.region_id = reg.id
		WHERE dr.driver_id = $1
		ORDER BY dr.is_primary DESC, reg.name
	`

	rows, err := r.db.Query(ctx, query, driverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver regions: %w", err)
	}
	defer rows.Close()

	driverRegions := make([]*DriverRegion, 0)
	for rows.Next() {
		dr := &DriverRegion{Region: &Region{}}
		err := rows.Scan(
			&dr.ID, &dr.DriverID, &dr.RegionID, &dr.IsPrimary, &dr.IsVerified,
			&dr.VerifiedAt, &dr.CreatedAt,
			&dr.Region.ID, &dr.Region.CountryID, &dr.Region.Code, &dr.Region.Name,
			&dr.Region.NativeName, &dr.Region.Timezone, &dr.Region.IsActive,
			&dr.Region.LaunchedAt, &dr.Region.CreatedAt, &dr.Region.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan driver region: %w", err)
		}
		driverRegions = append(driverRegions, dr)
	}

	return driverRegions, nil
}

// AddDriverRegion adds a new operating region for a driver
func (r *Repository) AddDriverRegion(ctx context.Context, driverID, regionID uuid.UUID, isPrimary bool) error {
	// If setting as primary, unset other primary regions first
	if isPrimary {
		_, err := r.db.Exec(ctx,
			`UPDATE driver_regions SET is_primary = false WHERE driver_id = $1`,
			driverID,
		)
		if err != nil {
			return fmt.Errorf("failed to unset primary region: %w", err)
		}
	}

	query := `
		INSERT INTO driver_regions (id, driver_id, region_id, is_primary, is_verified)
		VALUES ($1, $2, $3, $4, false)
		ON CONFLICT (driver_id, region_id) DO UPDATE SET is_primary = $4
	`

	_, err := r.db.Exec(ctx, query, uuid.New(), driverID, regionID, isPrimary)
	if err != nil {
		return fmt.Errorf("failed to add driver region: %w", err)
	}

	return nil
}

// marshalJSON marshals a JSON map to bytes, defaulting to {} if nil
func marshalJSON(j JSON) []byte {
	if j == nil {
		return []byte("{}")
	}
	b, err := json.Marshal(j)
	if err != nil {
		return []byte("{}")
	}
	return b
}

// CreateCountry creates a new country
func (r *Repository) CreateCountry(ctx context.Context, country *Country) error {
	regulationsJSON := marshalJSON(country.Regulations)
	paymentMethodsJSON := marshalJSON(country.PaymentMethods)
	requiredDocsJSON := marshalJSON(country.RequiredDriverDocuments)

	query := `
		INSERT INTO countries (id, code, code3, name, native_name, currency_code,
		                       default_language, timezone, phone_prefix, is_active,
		                       regulations, payment_methods, required_driver_documents)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING created_at, updated_at
	`

	country.ID = uuid.New()
	err := r.db.QueryRow(ctx, query,
		country.ID, country.Code, country.Code3, country.Name, country.NativeName,
		country.CurrencyCode, country.DefaultLanguage, country.Timezone,
		country.PhonePrefix, country.IsActive, regulationsJSON,
		paymentMethodsJSON, requiredDocsJSON,
	).Scan(&country.CreatedAt, &country.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create country: %w", err)
	}

	return nil
}

// CreateRegion creates a new region
func (r *Repository) CreateRegion(ctx context.Context, region *Region) error {
	query := `
		INSERT INTO regions (id, country_id, code, name, native_name, timezone, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`

	region.ID = uuid.New()
	err := r.db.QueryRow(ctx, query,
		region.ID, region.CountryID, region.Code, region.Name,
		region.NativeName, region.Timezone, region.IsActive,
	).Scan(&region.CreatedAt, &region.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create region: %w", err)
	}

	return nil
}

// CreateCity creates a new city
func (r *Repository) CreateCity(ctx context.Context, city *City) error {
	query := `
		INSERT INTO cities (id, region_id, name, native_name, timezone,
		                    center_latitude, center_longitude, boundary, population, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7,
		        CASE WHEN $8::text IS NOT NULL THEN ST_GeomFromText($8, 4326) ELSE NULL END,
		        $9, $10)
		RETURNING created_at, updated_at
	`

	city.ID = uuid.New()
	err := r.db.QueryRow(ctx, query,
		city.ID, city.RegionID, city.Name, city.NativeName, city.Timezone,
		city.CenterLatitude, city.CenterLongitude, city.Boundary,
		city.Population, city.IsActive,
	).Scan(&city.CreatedAt, &city.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create city: %w", err)
	}

	return nil
}

// GetAllCountries retrieves all countries with pagination and search
func (r *Repository) GetAllCountries(ctx context.Context, limit, offset int, search string) ([]*Country, int64, error) {
	whereClause := "WHERE 1=1"
	args := make([]interface{}, 0)
	argIndex := 1

	if search != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR code ILIKE $%d OR native_name ILIKE $%d)", argIndex, argIndex, argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM countries %s", whereClause)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count countries: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, code, code3, name, native_name, currency_code, default_language,
		       timezone, phone_prefix, is_active, launched_at, regulations,
		       payment_methods, required_driver_documents, created_at, updated_at
		FROM countries %s
		ORDER BY name
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get countries: %w", err)
	}
	defer rows.Close()

	countries := make([]*Country, 0)
	for rows.Next() {
		c := &Country{}
		err := rows.Scan(
			&c.ID, &c.Code, &c.Code3, &c.Name, &c.NativeName, &c.CurrencyCode,
			&c.DefaultLanguage, &c.Timezone, &c.PhonePrefix, &c.IsActive,
			&c.LaunchedAt, &c.Regulations, &c.PaymentMethods,
			&c.RequiredDriverDocuments, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan country: %w", err)
		}
		countries = append(countries, c)
	}

	return countries, total, nil
}

// GetAllRegions retrieves all regions for a country with pagination and search
func (r *Repository) GetAllRegions(ctx context.Context, countryID *uuid.UUID, limit, offset int, search string) ([]*Region, int64, error) {
	whereClause := "WHERE 1=1"
	args := make([]interface{}, 0)
	argIndex := 1

	if countryID != nil {
		whereClause += fmt.Sprintf(" AND country_id = $%d", argIndex)
		args = append(args, *countryID)
		argIndex++
	}
	if search != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR code ILIKE $%d OR native_name ILIKE $%d)", argIndex, argIndex, argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM regions %s", whereClause)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count regions: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, country_id, code, name, native_name, timezone, is_active,
		       launched_at, created_at, updated_at
		FROM regions %s
		ORDER BY name
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get regions: %w", err)
	}
	defer rows.Close()

	regions := make([]*Region, 0)
	for rows.Next() {
		reg := &Region{}
		err := rows.Scan(
			&reg.ID, &reg.CountryID, &reg.Code, &reg.Name, &reg.NativeName,
			&reg.Timezone, &reg.IsActive, &reg.LaunchedAt, &reg.CreatedAt, &reg.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan region: %w", err)
		}
		regions = append(regions, reg)
	}

	return regions, total, nil
}

// GetAllCities retrieves all cities for a region with pagination and search
func (r *Repository) GetAllCities(ctx context.Context, regionID *uuid.UUID, limit, offset int, search string) ([]*City, int64, error) {
	whereClause := "WHERE 1=1"
	args := make([]interface{}, 0)
	argIndex := 1

	if regionID != nil {
		whereClause += fmt.Sprintf(" AND region_id = $%d", argIndex)
		args = append(args, *regionID)
		argIndex++
	}
	if search != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR native_name ILIKE $%d)", argIndex, argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM cities %s", whereClause)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count cities: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, region_id, name, native_name, timezone, center_latitude,
		       center_longitude, ST_AsText(boundary), population, is_active,
		       launched_at, created_at, updated_at
		FROM cities %s
		ORDER BY name
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get cities: %w", err)
	}
	defer rows.Close()

	cities := make([]*City, 0)
	for rows.Next() {
		city := &City{}
		err := rows.Scan(
			&city.ID, &city.RegionID, &city.Name, &city.NativeName, &city.Timezone,
			&city.CenterLatitude, &city.CenterLongitude, &city.Boundary,
			&city.Population, &city.IsActive, &city.LaunchedAt,
			&city.CreatedAt, &city.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan city: %w", err)
		}
		cities = append(cities, city)
	}

	return cities, total, nil
}

// GetAllPricingZones retrieves all pricing zones for a city with pagination and search
func (r *Repository) GetAllPricingZones(ctx context.Context, cityID *uuid.UUID, limit, offset int, search string) ([]*PricingZone, int64, error) {
	whereClause := "WHERE 1=1"
	args := make([]interface{}, 0)
	argIndex := 1

	if cityID != nil {
		whereClause += fmt.Sprintf(" AND city_id = $%d", argIndex)
		args = append(args, *cityID)
		argIndex++
	}
	if search != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR zone_type ILIKE $%d)", argIndex, argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM pricing_zones %s", whereClause)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count pricing zones: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, city_id, name, zone_type, ST_AsText(boundary),
		       center_latitude, center_longitude, priority, is_active,
		       metadata, created_at, updated_at
		FROM pricing_zones %s
		ORDER BY priority DESC, name
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get pricing zones: %w", err)
	}
	defer rows.Close()

	zones := make([]*PricingZone, 0)
	for rows.Next() {
		zone := &PricingZone{}
		err := rows.Scan(
			&zone.ID, &zone.CityID, &zone.Name, &zone.ZoneType, &zone.Boundary,
			&zone.CenterLatitude, &zone.CenterLongitude, &zone.Priority,
			&zone.IsActive, &zone.Metadata, &zone.CreatedAt, &zone.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan pricing zone: %w", err)
		}
		zones = append(zones, zone)
	}

	return zones, total, nil
}

// UpdateCountry updates a country
func (r *Repository) UpdateCountry(ctx context.Context, country *Country) error {
	regulationsJSON := marshalJSON(country.Regulations)
	paymentMethodsJSON := marshalJSON(country.PaymentMethods)
	requiredDocsJSON := marshalJSON(country.RequiredDriverDocuments)

	query := `
		UPDATE countries SET
			code = $2, code3 = $3, name = $4, native_name = $5, currency_code = $6,
			default_language = $7, timezone = $8, phone_prefix = $9, is_active = $10,
			regulations = $11, payment_methods = $12, required_driver_documents = $13,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRow(ctx, query,
		country.ID, country.Code, country.Code3, country.Name, country.NativeName,
		country.CurrencyCode, country.DefaultLanguage, country.Timezone,
		country.PhonePrefix, country.IsActive, regulationsJSON,
		paymentMethodsJSON, requiredDocsJSON,
	).Scan(&country.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update country: %w", err)
	}

	return nil
}

// DeleteCountry soft-deletes a country by setting is_active to false
func (r *Repository) DeleteCountry(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE countries SET is_active = false, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete country: %w", err)
	}
	return nil
}

// UpdateRegion updates a region
func (r *Repository) UpdateRegion(ctx context.Context, region *Region) error {
	query := `
		UPDATE regions SET
			country_id = $2, code = $3, name = $4, native_name = $5,
			timezone = $6, is_active = $7, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRow(ctx, query,
		region.ID, region.CountryID, region.Code, region.Name,
		region.NativeName, region.Timezone, region.IsActive,
	).Scan(&region.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update region: %w", err)
	}

	return nil
}

// DeleteRegion soft-deletes a region by setting is_active to false
func (r *Repository) DeleteRegion(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE regions SET is_active = false, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete region: %w", err)
	}
	return nil
}

// UpdateCity updates a city
func (r *Repository) UpdateCity(ctx context.Context, city *City) error {
	query := `
		UPDATE cities SET
			region_id = $2, name = $3, native_name = $4, timezone = $5,
			center_latitude = $6, center_longitude = $7,
			boundary = CASE WHEN $8::text IS NOT NULL THEN ST_GeomFromText($8, 4326) ELSE boundary END,
			population = $9, is_active = $10, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRow(ctx, query,
		city.ID, city.RegionID, city.Name, city.NativeName, city.Timezone,
		city.CenterLatitude, city.CenterLongitude, city.Boundary,
		city.Population, city.IsActive,
	).Scan(&city.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update city: %w", err)
	}

	return nil
}

// DeleteCity soft-deletes a city by setting is_active to false
func (r *Repository) DeleteCity(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE cities SET is_active = false, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete city: %w", err)
	}
	return nil
}

// UpdatePricingZone updates a pricing zone
func (r *Repository) UpdatePricingZone(ctx context.Context, zone *PricingZone) error {
	metadataJSON := marshalJSON(zone.Metadata)

	query := `
		UPDATE pricing_zones SET
			city_id = $2, name = $3, zone_type = $4,
			boundary = ST_GeomFromText($5, 4326),
			center_latitude = $6, center_longitude = $7,
			priority = $8, is_active = $9, metadata = $10,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRow(ctx, query,
		zone.ID, zone.CityID, zone.Name, zone.ZoneType, zone.Boundary,
		zone.CenterLatitude, zone.CenterLongitude, zone.Priority,
		zone.IsActive, metadataJSON,
	).Scan(&zone.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update pricing zone: %w", err)
	}

	return nil
}

// DeletePricingZone soft-deletes a pricing zone by setting is_active to false
func (r *Repository) DeletePricingZone(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE pricing_zones SET is_active = false, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete pricing zone: %w", err)
	}
	return nil
}

// GetPricingZoneByID retrieves a pricing zone by its ID
func (r *Repository) GetPricingZoneByID(ctx context.Context, id uuid.UUID) (*PricingZone, error) {
	query := `
		SELECT id, city_id, name, zone_type, ST_AsText(boundary),
		       center_latitude, center_longitude, priority, is_active,
		       metadata, created_at, updated_at
		FROM pricing_zones
		WHERE id = $1
	`

	zone := &PricingZone{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&zone.ID, &zone.CityID, &zone.Name, &zone.ZoneType, &zone.Boundary,
		&zone.CenterLatitude, &zone.CenterLongitude, &zone.Priority,
		&zone.IsActive, &zone.Metadata, &zone.CreatedAt, &zone.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing zone: %w", err)
	}

	return zone, nil
}

// GetGeographyStats returns aggregate counts for all geography entities
func (r *Repository) GetGeographyStats(ctx context.Context) (*GeographyStats, error) {
	query := `
		SELECT
			(SELECT COUNT(*) FROM countries) AS countries_total,
			(SELECT COUNT(*) FROM countries WHERE is_active = true) AS countries_active,
			(SELECT COUNT(*) FROM regions) AS regions_total,
			(SELECT COUNT(*) FROM regions WHERE is_active = true) AS regions_active,
			(SELECT COUNT(*) FROM cities) AS cities_total,
			(SELECT COUNT(*) FROM cities WHERE is_active = true) AS cities_active,
			(SELECT COUNT(*) FROM pricing_zones) AS zones_total,
			(SELECT COUNT(*) FROM pricing_zones WHERE is_active = true) AS zones_active
	`

	stats := &GeographyStats{}
	err := r.db.QueryRow(ctx, query).Scan(
		&stats.Countries.Total, &stats.Countries.Active,
		&stats.Regions.Total, &stats.Regions.Active,
		&stats.Cities.Total, &stats.Cities.Active,
		&stats.PricingZones.Total, &stats.PricingZones.Active,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get geography stats: %w", err)
	}

	return stats, nil
}

// CreatePricingZone creates a new pricing zone
func (r *Repository) CreatePricingZone(ctx context.Context, zone *PricingZone) error {
	metadataJSON := marshalJSON(zone.Metadata)

	query := `
		INSERT INTO pricing_zones (id, city_id, name, zone_type, boundary,
		                           center_latitude, center_longitude, priority,
		                           is_active, metadata)
		VALUES ($1, $2, $3, $4, ST_GeomFromText($5, 4326), $6, $7, $8, $9, $10)
		RETURNING created_at, updated_at
	`

	zone.ID = uuid.New()
	err := r.db.QueryRow(ctx, query,
		zone.ID, zone.CityID, zone.Name, zone.ZoneType, zone.Boundary,
		zone.CenterLatitude, zone.CenterLongitude, zone.Priority,
		zone.IsActive, metadataJSON,
	).Scan(&zone.CreatedAt, &zone.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create pricing zone: %w", err)
	}

	return nil
}
