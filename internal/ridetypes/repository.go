package ridetypes

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for ride types
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new ride types repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateRideType creates a new ride type
func (r *Repository) CreateRideType(ctx context.Context, rt *RideType) error {
	query := `
		INSERT INTO ride_types (id, name, description, icon, capacity, sort_order, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`
	rt.ID = uuid.New()
	err := r.db.QueryRow(ctx, query,
		rt.ID, rt.Name, rt.Description, rt.Icon, rt.Capacity, rt.SortOrder, rt.IsActive,
	).Scan(&rt.CreatedAt, &rt.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create ride type: %w", err)
	}
	return nil
}

// GetRideTypeByID retrieves a ride type by ID
func (r *Repository) GetRideTypeByID(ctx context.Context, id uuid.UUID) (*RideType, error) {
	query := `
		SELECT id, name, description, icon, capacity, sort_order, is_active, created_at, updated_at
		FROM ride_types WHERE id = $1
	`
	rt := &RideType{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&rt.ID, &rt.Name, &rt.Description, &rt.Icon, &rt.Capacity, &rt.SortOrder,
		&rt.IsActive, &rt.CreatedAt, &rt.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride type: %w", err)
	}
	return rt, nil
}

// ListRideTypes lists ride types with pagination
func (r *Repository) ListRideTypes(ctx context.Context, limit, offset int, includeInactive bool) ([]*RideType, int64, error) {
	whereClause := ""
	if !includeInactive {
		whereClause = "WHERE is_active = true"
	}

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM ride_types %s", whereClause)
	if err := r.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count ride types: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, name, description, icon, capacity, sort_order, is_active, created_at, updated_at
		FROM ride_types %s
		ORDER BY sort_order, name
		LIMIT $1 OFFSET $2
	`, whereClause)

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list ride types: %w", err)
	}
	defer rows.Close()

	items := make([]*RideType, 0)
	for rows.Next() {
		rt := &RideType{}
		err := rows.Scan(
			&rt.ID, &rt.Name, &rt.Description, &rt.Icon, &rt.Capacity, &rt.SortOrder,
			&rt.IsActive, &rt.CreatedAt, &rt.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan ride type: %w", err)
		}
		items = append(items, rt)
	}
	return items, total, nil
}

// UpdateRideType updates a ride type
func (r *Repository) UpdateRideType(ctx context.Context, rt *RideType) error {
	query := `
		UPDATE ride_types SET
			name = $2, description = $3, icon = $4, capacity = $5,
			sort_order = $6, is_active = $7, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query,
		rt.ID, rt.Name, rt.Description, rt.Icon, rt.Capacity, rt.SortOrder, rt.IsActive,
	).Scan(&rt.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update ride type: %w", err)
	}
	return nil
}

// DeleteRideType soft-deletes a ride type
func (r *Repository) DeleteRideType(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE ride_types SET is_active = false, updated_at = CURRENT_TIMESTAMP WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete ride type: %w", err)
	}
	return nil
}

// --- Country Ride Types ---

// AddRideTypeToCountry adds a ride type to a country
func (r *Repository) AddRideTypeToCountry(ctx context.Context, crt *CountryRideType) error {
	query := `
		INSERT INTO country_ride_types (country_id, ride_type_id, is_active, sort_order)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (country_id, ride_type_id) DO UPDATE SET
			is_active = EXCLUDED.is_active,
			sort_order = EXCLUDED.sort_order,
			updated_at = CURRENT_TIMESTAMP
		RETURNING created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query,
		crt.CountryID, crt.RideTypeID, crt.IsActive, crt.SortOrder,
	).Scan(&crt.CreatedAt, &crt.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to add ride type to country: %w", err)
	}
	return nil
}

// UpdateCountryRideType updates a country ride type mapping
func (r *Repository) UpdateCountryRideType(ctx context.Context, crt *CountryRideType) error {
	query := `
		UPDATE country_ride_types SET
			is_active = $3, sort_order = $4, updated_at = CURRENT_TIMESTAMP
		WHERE country_id = $1 AND ride_type_id = $2
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query,
		crt.CountryID, crt.RideTypeID, crt.IsActive, crt.SortOrder,
	).Scan(&crt.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update country ride type: %w", err)
	}
	return nil
}

// RemoveRideTypeFromCountry removes a ride type from a country
func (r *Repository) RemoveRideTypeFromCountry(ctx context.Context, countryID, rideTypeID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM country_ride_types WHERE country_id = $1 AND ride_type_id = $2`,
		countryID, rideTypeID,
	)
	if err != nil {
		return fmt.Errorf("failed to remove ride type from country: %w", err)
	}
	return nil
}

// ListCountryRideTypes lists ride types for a country with details
func (r *Repository) ListCountryRideTypes(ctx context.Context, countryID uuid.UUID, includeInactive bool) ([]*CountryRideTypeWithDetails, error) {
	whereClause := "WHERE crt.country_id = $1"
	if !includeInactive {
		whereClause += " AND crt.is_active = true AND rt.is_active = true"
	}

	query := fmt.Sprintf(`
		SELECT crt.country_id, crt.ride_type_id, crt.is_active, crt.sort_order,
		       crt.created_at, crt.updated_at,
		       rt.name, rt.description, rt.icon, rt.capacity
		FROM country_ride_types crt
		JOIN ride_types rt ON rt.id = crt.ride_type_id
		%s
		ORDER BY crt.sort_order, rt.name
	`, whereClause)

	rows, err := r.db.Query(ctx, query, countryID)
	if err != nil {
		return nil, fmt.Errorf("failed to list country ride types: %w", err)
	}
	defer rows.Close()

	items := make([]*CountryRideTypeWithDetails, 0)
	for rows.Next() {
		crt := &CountryRideTypeWithDetails{}
		err := rows.Scan(
			&crt.CountryID, &crt.RideTypeID, &crt.IsActive, &crt.SortOrder,
			&crt.CreatedAt, &crt.UpdatedAt,
			&crt.RideTypeName, &crt.RideTypeDescription, &crt.RideTypeIcon, &crt.RideTypeCapacity,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan country ride type: %w", err)
		}
		items = append(items, crt)
	}
	return items, nil
}

// --- City Ride Types ---

// AddRideTypeToCity adds a ride type to a city
func (r *Repository) AddRideTypeToCity(ctx context.Context, crt *CityRideType) error {
	query := `
		INSERT INTO city_ride_types (city_id, ride_type_id, is_active, sort_order)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (city_id, ride_type_id) DO UPDATE SET
			is_active = EXCLUDED.is_active,
			sort_order = EXCLUDED.sort_order,
			updated_at = CURRENT_TIMESTAMP
		RETURNING created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query,
		crt.CityID, crt.RideTypeID, crt.IsActive, crt.SortOrder,
	).Scan(&crt.CreatedAt, &crt.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to add ride type to city: %w", err)
	}
	return nil
}

// UpdateCityRideType updates a city ride type mapping
func (r *Repository) UpdateCityRideType(ctx context.Context, crt *CityRideType) error {
	query := `
		UPDATE city_ride_types SET
			is_active = $3, sort_order = $4, updated_at = CURRENT_TIMESTAMP
		WHERE city_id = $1 AND ride_type_id = $2
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query,
		crt.CityID, crt.RideTypeID, crt.IsActive, crt.SortOrder,
	).Scan(&crt.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update city ride type: %w", err)
	}
	return nil
}

// RemoveRideTypeFromCity removes a ride type from a city
func (r *Repository) RemoveRideTypeFromCity(ctx context.Context, cityID, rideTypeID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM city_ride_types WHERE city_id = $1 AND ride_type_id = $2`,
		cityID, rideTypeID,
	)
	if err != nil {
		return fmt.Errorf("failed to remove ride type from city: %w", err)
	}
	return nil
}

// ListCityRideTypes lists ride types for a city with details
func (r *Repository) ListCityRideTypes(ctx context.Context, cityID uuid.UUID, includeInactive bool) ([]*CityRideTypeWithDetails, error) {
	whereClause := "WHERE crt.city_id = $1"
	if !includeInactive {
		whereClause += " AND crt.is_active = true AND rt.is_active = true"
	}

	query := fmt.Sprintf(`
		SELECT crt.city_id, crt.ride_type_id, crt.is_active, crt.sort_order,
		       crt.created_at, crt.updated_at,
		       rt.name, rt.description, rt.icon, rt.capacity
		FROM city_ride_types crt
		JOIN ride_types rt ON rt.id = crt.ride_type_id
		%s
		ORDER BY crt.sort_order, rt.name
	`, whereClause)

	rows, err := r.db.Query(ctx, query, cityID)
	if err != nil {
		return nil, fmt.Errorf("failed to list city ride types: %w", err)
	}
	defer rows.Close()

	items := make([]*CityRideTypeWithDetails, 0)
	for rows.Next() {
		crt := &CityRideTypeWithDetails{}
		err := rows.Scan(
			&crt.CityID, &crt.RideTypeID, &crt.IsActive, &crt.SortOrder,
			&crt.CreatedAt, &crt.UpdatedAt,
			&crt.RideTypeName, &crt.RideTypeDescription, &crt.RideTypeIcon, &crt.RideTypeCapacity,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan city ride type: %w", err)
		}
		items = append(items, crt)
	}
	return items, nil
}

// GetAvailableRideTypes returns active ride types for a resolved location using cascading logic:
//   1. If city_ride_types rows exist for this city → use those (city-level overrides)
//   2. Else if country_ride_types rows exist → use those (country-level, available in all cities)
//   3. Else → return all active ride types (global fallback)
func (r *Repository) GetAvailableRideTypes(ctx context.Context, countryID, cityID *uuid.UUID) ([]*RideType, error) {
	// Try city-level first
	if cityID != nil {
		query := `
			SELECT rt.id, rt.name, rt.description, rt.icon, rt.capacity,
			       COALESCE(crt.sort_order, rt.sort_order) AS sort_order,
			       rt.is_active, rt.created_at, rt.updated_at
			FROM ride_types rt
			JOIN city_ride_types crt ON crt.ride_type_id = rt.id
			WHERE crt.city_id = $1
			  AND crt.is_active = true
			  AND rt.is_active = true
			ORDER BY crt.sort_order, rt.name
		`
		items, err := r.scanRideTypes(ctx, query, *cityID)
		if err == nil && len(items) > 0 {
			return items, nil
		}
	}

	// Try country-level
	if countryID != nil {
		query := `
			SELECT rt.id, rt.name, rt.description, rt.icon, rt.capacity,
			       COALESCE(crt.sort_order, rt.sort_order) AS sort_order,
			       rt.is_active, rt.created_at, rt.updated_at
			FROM ride_types rt
			JOIN country_ride_types crt ON crt.ride_type_id = rt.id
			WHERE crt.country_id = $1
			  AND crt.is_active = true
			  AND rt.is_active = true
			ORDER BY crt.sort_order, rt.name
		`
		items, err := r.scanRideTypes(ctx, query, *countryID)
		if err == nil && len(items) > 0 {
			return items, nil
		}
	}

	// Global fallback: all active ride types
	query := `
		SELECT id, name, description, icon, capacity, sort_order,
		       is_active, created_at, updated_at
		FROM ride_types
		WHERE is_active = true
		ORDER BY sort_order, name
	`
	return r.scanRideTypes(ctx, query)
}

// scanRideTypes is a helper that executes a query and scans ride type rows
func (r *Repository) scanRideTypes(ctx context.Context, query string, args ...interface{}) ([]*RideType, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query ride types: %w", err)
	}
	defer rows.Close()

	items := make([]*RideType, 0)
	for rows.Next() {
		rt := &RideType{}
		err := rows.Scan(
			&rt.ID, &rt.Name, &rt.Description, &rt.Icon, &rt.Capacity, &rt.SortOrder,
			&rt.IsActive, &rt.CreatedAt, &rt.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ride type: %w", err)
		}
		items = append(items, rt)
	}
	return items, nil
}
