package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// Repository handles database operations for pricing
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new pricing repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// GetActiveVersionID returns the currently active pricing version ID
func (r *Repository) GetActiveVersionID(ctx context.Context) (uuid.UUID, error) {
	query := `
		SELECT id FROM pricing_config_versions
		WHERE status = 'active'
		  AND (effective_from IS NULL OR effective_from <= NOW())
		  AND (effective_until IS NULL OR effective_until > NOW())
		ORDER BY effective_from DESC NULLS LAST
		LIMIT 1
	`

	var versionID uuid.UUID
	err := r.db.QueryRow(ctx, query).Scan(&versionID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get active pricing version: %w", err)
	}

	return versionID, nil
}

// GetPricingConfigsForResolution retrieves all configs needed to resolve pricing for a location
func (r *Repository) GetPricingConfigsForResolution(ctx context.Context, versionID uuid.UUID, countryID, regionID, cityID, zoneID, rideTypeID *uuid.UUID) ([]*PricingConfig, error) {
	query := `
		SELECT id, version_id, country_id, region_id, city_id, zone_id, ride_type_id,
		       base_fare, per_km_rate, per_minute_rate, minimum_fare, booking_fee,
		       platform_commission_pct, driver_incentive_pct,
		       surge_min_multiplier, surge_max_multiplier,
		       tax_rate_pct, tax_inclusive, cancellation_fees, is_active, created_at, updated_at
		FROM pricing_configs
		WHERE version_id = $1
		  AND is_active = true
		  AND (
		    -- Global config (no location scope)
		    (country_id IS NULL AND region_id IS NULL AND city_id IS NULL AND zone_id IS NULL)
		    -- Country level
		    OR (country_id = $2 AND region_id IS NULL AND city_id IS NULL AND zone_id IS NULL)
		    -- Region level
		    OR (region_id = $3 AND city_id IS NULL AND zone_id IS NULL)
		    -- City level
		    OR (city_id = $4 AND zone_id IS NULL)
		    -- Zone level
		    OR zone_id = $5
		  )
		  AND (ride_type_id IS NULL OR ride_type_id = $6)
		ORDER BY
		  -- Hierarchy level (more specific = higher priority)
		  CASE
		    WHEN zone_id IS NOT NULL THEN 4
		    WHEN city_id IS NOT NULL THEN 3
		    WHEN region_id IS NOT NULL THEN 2
		    WHEN country_id IS NOT NULL THEN 1
		    ELSE 0
		  END DESC,
		  -- Ride type specificity (specific ride type > all ride types)
		  CASE WHEN ride_type_id IS NOT NULL THEN 1 ELSE 0 END DESC
	`

	rows, err := r.db.Query(ctx, query, versionID, countryID, regionID, cityID, zoneID, rideTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing configs: %w", err)
	}
	defer rows.Close()

	configs := make([]*PricingConfig, 0)
	for rows.Next() {
		config := &PricingConfig{}
		var cancellationFeesJSON []byte

		err := rows.Scan(
			&config.ID, &config.VersionID, &config.CountryID, &config.RegionID,
			&config.CityID, &config.ZoneID, &config.RideTypeID,
			&config.BaseFare, &config.PerKmRate, &config.PerMinuteRate,
			&config.MinimumFare, &config.BookingFee,
			&config.PlatformCommissionPct, &config.DriverIncentivePct,
			&config.SurgeMinMultiplier, &config.SurgeMaxMultiplier,
			&config.TaxRatePct, &config.TaxInclusive, &cancellationFeesJSON,
			&config.IsActive, &config.CreatedAt, &config.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pricing config: %w", err)
		}

		// Parse cancellation fees JSON
		if len(cancellationFeesJSON) > 0 {
			if err := json.Unmarshal(cancellationFeesJSON, &config.CancellationFees); err != nil {
				return nil, fmt.Errorf("failed to parse cancellation fees: %w", err)
			}
		}

		configs = append(configs, config)
	}

	return configs, nil
}

// GetZoneFees retrieves zone fees for pickup and dropoff zones
func (r *Repository) GetZoneFees(ctx context.Context, versionID uuid.UUID, pickupZoneID, dropoffZoneID *uuid.UUID, rideTypeID *uuid.UUID) ([]*ZoneFee, error) {
	if pickupZoneID == nil && dropoffZoneID == nil {
		return nil, nil
	}

	query := `
		SELECT id, zone_id, version_id, fee_type, ride_type_id, amount,
		       is_percentage, applies_pickup, applies_dropoff, schedule,
		       is_active, created_at, updated_at
		FROM zone_fees
		WHERE version_id = $1
		  AND is_active = true
		  AND (zone_id = $2 OR zone_id = $3)
		  AND (ride_type_id IS NULL OR ride_type_id = $4)
	`

	rows, err := r.db.Query(ctx, query, versionID, pickupZoneID, dropoffZoneID, rideTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone fees: %w", err)
	}
	defer rows.Close()

	fees := make([]*ZoneFee, 0)
	for rows.Next() {
		fee := &ZoneFee{}
		var scheduleJSON []byte

		err := rows.Scan(
			&fee.ID, &fee.ZoneID, &fee.VersionID, &fee.FeeType, &fee.RideTypeID,
			&fee.Amount, &fee.IsPercentage, &fee.AppliesPickup, &fee.AppliesDropoff,
			&scheduleJSON, &fee.IsActive, &fee.CreatedAt, &fee.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan zone fee: %w", err)
		}

		if len(scheduleJSON) > 0 {
			fee.Schedule = &FeeSchedule{}
			if err := json.Unmarshal(scheduleJSON, fee.Schedule); err != nil {
				return nil, fmt.Errorf("failed to parse fee schedule: %w", err)
			}
		}

		fees = append(fees, fee)
	}

	return fees, nil
}

// GetTimeMultipliers retrieves time-based multipliers
func (r *Repository) GetTimeMultipliers(ctx context.Context, versionID uuid.UUID, countryID, regionID, cityID *uuid.UUID, t time.Time) ([]*TimeMultiplier, error) {
	dayOfWeek := int(t.Weekday())
	timeOfDay := t.Format("15:04")

	query := `
		SELECT id, version_id, country_id, region_id, city_id, name,
		       days_of_week, start_time, end_time, multiplier, priority,
		       is_active, created_at, updated_at
		FROM time_multipliers
		WHERE version_id = $1
		  AND is_active = true
		  AND $2 = ANY(days_of_week)
		  AND (
		    (start_time <= end_time AND $3::time >= start_time AND $3::time <= end_time)
		    OR (start_time > end_time AND ($3::time >= start_time OR $3::time <= end_time))
		  )
		  AND (
		    (country_id IS NULL AND region_id IS NULL AND city_id IS NULL)
		    OR (country_id = $4 AND region_id IS NULL AND city_id IS NULL)
		    OR (region_id = $5 AND city_id IS NULL)
		    OR city_id = $6
		  )
		ORDER BY priority DESC
		LIMIT 1
	`

	rows, err := r.db.Query(ctx, query, versionID, dayOfWeek, timeOfDay, countryID, regionID, cityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get time multipliers: %w", err)
	}
	defer rows.Close()

	multipliers := make([]*TimeMultiplier, 0)
	for rows.Next() {
		m := &TimeMultiplier{}
		err := rows.Scan(
			&m.ID, &m.VersionID, &m.CountryID, &m.RegionID, &m.CityID,
			&m.Name, &m.DaysOfWeek, &m.StartTime, &m.EndTime, &m.Multiplier,
			&m.Priority, &m.IsActive, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan time multiplier: %w", err)
		}
		multipliers = append(multipliers, m)
	}

	return multipliers, nil
}

// GetWeatherMultiplier retrieves weather-based multiplier
func (r *Repository) GetWeatherMultiplier(ctx context.Context, versionID uuid.UUID, countryID, regionID, cityID *uuid.UUID, condition string) (*WeatherMultiplier, error) {
	query := `
		SELECT id, version_id, country_id, region_id, city_id,
		       weather_condition, multiplier, is_active, created_at
		FROM weather_multipliers
		WHERE version_id = $1
		  AND is_active = true
		  AND weather_condition = $2
		  AND (
		    (country_id IS NULL AND region_id IS NULL AND city_id IS NULL)
		    OR (country_id = $3 AND region_id IS NULL AND city_id IS NULL)
		    OR (region_id = $4 AND city_id IS NULL)
		    OR city_id = $5
		  )
		ORDER BY
		  CASE
		    WHEN city_id IS NOT NULL THEN 3
		    WHEN region_id IS NOT NULL THEN 2
		    WHEN country_id IS NOT NULL THEN 1
		    ELSE 0
		  END DESC
		LIMIT 1
	`

	m := &WeatherMultiplier{}
	err := r.db.QueryRow(ctx, query, versionID, condition, countryID, regionID, cityID).Scan(
		&m.ID, &m.VersionID, &m.CountryID, &m.RegionID, &m.CityID,
		&m.WeatherCondition, &m.Multiplier, &m.IsActive, &m.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get weather multiplier: %w", err)
	}

	return m, nil
}

// GetActiveEventMultipliers retrieves event-based multipliers
func (r *Repository) GetActiveEventMultipliers(ctx context.Context, versionID uuid.UUID, cityID, zoneID *uuid.UUID, t time.Time) ([]*EventMultiplier, error) {
	query := `
		SELECT id, version_id, zone_id, city_id, event_name, event_type,
		       starts_at, ends_at, pre_event_minutes, post_event_minutes,
		       multiplier, expected_demand_increase, is_active, created_at
		FROM event_multipliers
		WHERE version_id = $1
		  AND is_active = true
		  AND (city_id = $2 OR zone_id = $3)
		  AND $4 >= (starts_at - (pre_event_minutes || ' minutes')::interval)
		  AND $4 <= (ends_at + (post_event_minutes || ' minutes')::interval)
		ORDER BY multiplier DESC
		LIMIT 1
	`

	rows, err := r.db.Query(ctx, query, versionID, cityID, zoneID, t)
	if err != nil {
		return nil, fmt.Errorf("failed to get event multipliers: %w", err)
	}
	defer rows.Close()

	multipliers := make([]*EventMultiplier, 0)
	for rows.Next() {
		m := &EventMultiplier{}
		err := rows.Scan(
			&m.ID, &m.VersionID, &m.ZoneID, &m.CityID, &m.EventName,
			&m.EventType, &m.StartsAt, &m.EndsAt, &m.PreEventMinutes,
			&m.PostEventMinutes, &m.Multiplier, &m.ExpectedDemandIncrease,
			&m.IsActive, &m.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event multiplier: %w", err)
		}
		multipliers = append(multipliers, m)
	}

	return multipliers, nil
}

// GetSurgeThresholds retrieves surge thresholds for a location
func (r *Repository) GetSurgeThresholds(ctx context.Context, versionID uuid.UUID, countryID, regionID, cityID *uuid.UUID) ([]*SurgeThreshold, error) {
	query := `
		SELECT id, version_id, country_id, region_id, city_id,
		       demand_supply_ratio_min, demand_supply_ratio_max, multiplier,
		       is_active, created_at
		FROM surge_thresholds
		WHERE version_id = $1
		  AND is_active = true
		  AND (
		    (country_id IS NULL AND region_id IS NULL AND city_id IS NULL)
		    OR (country_id = $2 AND region_id IS NULL AND city_id IS NULL)
		    OR (region_id = $3 AND city_id IS NULL)
		    OR city_id = $4
		  )
		ORDER BY
		  CASE
		    WHEN city_id IS NOT NULL THEN 3
		    WHEN region_id IS NOT NULL THEN 2
		    WHEN country_id IS NOT NULL THEN 1
		    ELSE 0
		  END DESC,
		  demand_supply_ratio_min ASC
	`

	rows, err := r.db.Query(ctx, query, versionID, countryID, regionID, cityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get surge thresholds: %w", err)
	}
	defer rows.Close()

	thresholds := make([]*SurgeThreshold, 0)
	for rows.Next() {
		t := &SurgeThreshold{}
		err := rows.Scan(
			&t.ID, &t.VersionID, &t.CountryID, &t.RegionID, &t.CityID,
			&t.DemandSupplyRatioMin, &t.DemandSupplyRatioMax, &t.Multiplier,
			&t.IsActive, &t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan surge threshold: %w", err)
		}
		thresholds = append(thresholds, t)
	}

	return thresholds, nil
}

// GetZoneName retrieves a zone name by ID
func (r *Repository) GetZoneName(ctx context.Context, zoneID uuid.UUID) (string, error) {
	var name string
	err := r.db.QueryRow(ctx, `SELECT name FROM pricing_zones WHERE id = $1`, zoneID).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("failed to get zone name: %w", err)
	}
	return name, nil
}

// ============================================================
// Version CRUD
// ============================================================

// CreateVersion creates a new pricing config version
func (r *Repository) CreateVersion(ctx context.Context, version *PricingConfigVersion) error {
	query := `
		INSERT INTO pricing_config_versions (id, version_number, name, description, status,
		            effective_from, effective_until, created_by)
		VALUES ($1, (SELECT COALESCE(MAX(version_number), 0) + 1 FROM pricing_config_versions),
		        $2, $3, 'draft', $4, $5, $6)
		RETURNING version_number, status, created_at, updated_at
	`
	version.ID = uuid.New()
	err := r.db.QueryRow(ctx, query,
		version.ID, version.Name, version.Description,
		version.EffectiveFrom, version.EffectiveUntil, version.CreatedBy,
	).Scan(&version.VersionNumber, &version.Status, &version.CreatedAt, &version.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create version: %w", err)
	}
	return nil
}

// GetVersionByID retrieves a pricing config version by ID
func (r *Repository) GetVersionByID(ctx context.Context, id uuid.UUID) (*PricingConfigVersion, error) {
	query := `
		SELECT id, version_number, name, description, status, ab_test_percentage,
		       effective_from, effective_until, created_by, approved_by, approved_at,
		       created_at, updated_at
		FROM pricing_config_versions
		WHERE id = $1
	`
	v := &PricingConfigVersion{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&v.ID, &v.VersionNumber, &v.Name, &v.Description, &v.Status,
		&v.ABTestPercentage, &v.EffectiveFrom, &v.EffectiveUntil,
		&v.CreatedBy, &v.ApprovedBy, &v.ApprovedAt, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}
	return v, nil
}

// ListVersions lists pricing config versions with pagination and optional status filter
func (r *Repository) ListVersions(ctx context.Context, limit, offset int, status string) ([]*PricingConfigVersion, int64, error) {
	whereClause := "WHERE 1=1"
	args := make([]interface{}, 0)
	argIndex := 1

	if status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM pricing_config_versions %s", whereClause)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count versions: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, version_number, name, description, status, ab_test_percentage,
		       effective_from, effective_until, created_by, approved_by, approved_at,
		       created_at, updated_at
		FROM pricing_config_versions %s
		ORDER BY version_number DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list versions: %w", err)
	}
	defer rows.Close()

	versions := make([]*PricingConfigVersion, 0)
	for rows.Next() {
		v := &PricingConfigVersion{}
		err := rows.Scan(
			&v.ID, &v.VersionNumber, &v.Name, &v.Description, &v.Status,
			&v.ABTestPercentage, &v.EffectiveFrom, &v.EffectiveUntil,
			&v.CreatedBy, &v.ApprovedBy, &v.ApprovedAt, &v.CreatedAt, &v.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan version: %w", err)
		}
		versions = append(versions, v)
	}

	return versions, total, nil
}

// UpdateVersion updates a pricing config version (only draft versions)
func (r *Repository) UpdateVersion(ctx context.Context, version *PricingConfigVersion) error {
	query := `
		UPDATE pricing_config_versions SET
			name = $2, description = $3, effective_from = $4, effective_until = $5,
			updated_at = NOW()
		WHERE id = $1 AND status = 'draft'
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query,
		version.ID, version.Name, version.Description,
		version.EffectiveFrom, version.EffectiveUntil,
	).Scan(&version.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update version: %w", err)
	}
	return nil
}

// ActivateVersion activates a version (archives the current active one)
func (r *Repository) ActivateVersion(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Archive currently active version(s)
	_, err = tx.Exec(ctx, `
		UPDATE pricing_config_versions SET status = 'archived', updated_at = NOW()
		WHERE status = 'active'
	`)
	if err != nil {
		return fmt.Errorf("failed to archive current version: %w", err)
	}

	// Activate the new version
	_, err = tx.Exec(ctx, `
		UPDATE pricing_config_versions SET
			status = 'active', approved_by = $2, approved_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'draft'
	`, id, adminID)
	if err != nil {
		return fmt.Errorf("failed to activate version: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// ArchiveVersion archives a version
func (r *Repository) ArchiveVersion(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE pricing_config_versions SET status = 'archived', updated_at = NOW()
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("failed to archive version: %w", err)
	}
	return nil
}

// CloneVersion deep-copies a version with all its configs, multipliers, thresholds, and zone fees
func (r *Repository) CloneVersion(ctx context.Context, sourceID uuid.UUID, name string, adminID uuid.UUID) (*PricingConfigVersion, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	newID := uuid.New()

	// Create new version
	var v PricingConfigVersion
	err = tx.QueryRow(ctx, `
		INSERT INTO pricing_config_versions (id, version_number, name, description, status, created_by)
		SELECT $1, (SELECT COALESCE(MAX(version_number), 0) + 1 FROM pricing_config_versions),
		       $2, description, 'draft', $3
		FROM pricing_config_versions WHERE id = $4
		RETURNING id, version_number, name, description, status, created_by, created_at, updated_at
	`, newID, name, adminID, sourceID).Scan(
		&v.ID, &v.VersionNumber, &v.Name, &v.Description, &v.Status,
		&v.CreatedBy, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to clone version: %w", err)
	}

	// Clone pricing configs
	_, err = tx.Exec(ctx, `
		INSERT INTO pricing_configs (id, version_id, country_id, region_id, city_id, zone_id,
		       ride_type_id, base_fare, per_km_rate, per_minute_rate, minimum_fare, booking_fee,
		       platform_commission_pct, driver_incentive_pct, surge_min_multiplier, surge_max_multiplier,
		       tax_rate_pct, tax_inclusive, cancellation_fees, is_active)
		SELECT gen_random_uuid(), $1, country_id, region_id, city_id, zone_id,
		       ride_type_id, base_fare, per_km_rate, per_minute_rate, minimum_fare, booking_fee,
		       platform_commission_pct, driver_incentive_pct, surge_min_multiplier, surge_max_multiplier,
		       tax_rate_pct, tax_inclusive, cancellation_fees, is_active
		FROM pricing_configs WHERE version_id = $2
	`, newID, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to clone configs: %w", err)
	}

	// Clone time multipliers
	_, err = tx.Exec(ctx, `
		INSERT INTO time_multipliers (id, version_id, country_id, region_id, city_id,
		       name, days_of_week, start_time, end_time, multiplier, priority, is_active)
		SELECT gen_random_uuid(), $1, country_id, region_id, city_id,
		       name, days_of_week, start_time, end_time, multiplier, priority, is_active
		FROM time_multipliers WHERE version_id = $2
	`, newID, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to clone time multipliers: %w", err)
	}

	// Clone weather multipliers
	_, err = tx.Exec(ctx, `
		INSERT INTO weather_multipliers (id, version_id, country_id, region_id, city_id,
		       weather_condition, multiplier, is_active)
		SELECT gen_random_uuid(), $1, country_id, region_id, city_id,
		       weather_condition, multiplier, is_active
		FROM weather_multipliers WHERE version_id = $2
	`, newID, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to clone weather multipliers: %w", err)
	}

	// Clone event multipliers
	_, err = tx.Exec(ctx, `
		INSERT INTO event_multipliers (id, version_id, zone_id, city_id, event_name, event_type,
		       starts_at, ends_at, pre_event_minutes, post_event_minutes, multiplier,
		       expected_demand_increase, is_active)
		SELECT gen_random_uuid(), $1, zone_id, city_id, event_name, event_type,
		       starts_at, ends_at, pre_event_minutes, post_event_minutes, multiplier,
		       expected_demand_increase, is_active
		FROM event_multipliers WHERE version_id = $2
	`, newID, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to clone event multipliers: %w", err)
	}

	// Clone zone fees
	_, err = tx.Exec(ctx, `
		INSERT INTO zone_fees (id, zone_id, version_id, fee_type, ride_type_id, amount,
		       is_percentage, applies_pickup, applies_dropoff, schedule, is_active)
		SELECT gen_random_uuid(), zone_id, $1, fee_type, ride_type_id, amount,
		       is_percentage, applies_pickup, applies_dropoff, schedule, is_active
		FROM zone_fees WHERE version_id = $2
	`, newID, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to clone zone fees: %w", err)
	}

	// Clone surge thresholds
	_, err = tx.Exec(ctx, `
		INSERT INTO surge_thresholds (id, version_id, country_id, region_id, city_id,
		       demand_supply_ratio_min, demand_supply_ratio_max, multiplier, is_active)
		SELECT gen_random_uuid(), $1, country_id, region_id, city_id,
		       demand_supply_ratio_min, demand_supply_ratio_max, multiplier, is_active
		FROM surge_thresholds WHERE version_id = $2
	`, newID, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to clone surge thresholds: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &v, nil
}

// ============================================================
// Config CRUD
// ============================================================

// CreateConfig creates a new pricing config
func (r *Repository) CreateConfig(ctx context.Context, config *PricingConfig) error {
	query := `
		INSERT INTO pricing_configs (id, version_id, country_id, region_id, city_id, zone_id,
		       ride_type_id, base_fare, per_km_rate, per_minute_rate, minimum_fare, booking_fee,
		       platform_commission_pct, driver_incentive_pct, surge_min_multiplier, surge_max_multiplier,
		       tax_rate_pct, tax_inclusive, cancellation_fees, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		RETURNING created_at, updated_at
	`
	config.ID = uuid.New()
	var cancellationFeesJSON []byte
	if len(config.CancellationFees) > 0 {
		var err error
		cancellationFeesJSON, err = json.Marshal(config.CancellationFees)
		if err != nil {
			return fmt.Errorf("failed to marshal cancellation fees: %w", err)
		}
	} else {
		cancellationFeesJSON = []byte("[]")
	}

	err := r.db.QueryRow(ctx, query,
		config.ID, config.VersionID, config.CountryID, config.RegionID,
		config.CityID, config.ZoneID, config.RideTypeID,
		config.BaseFare, config.PerKmRate, config.PerMinuteRate,
		config.MinimumFare, config.BookingFee,
		config.PlatformCommissionPct, config.DriverIncentivePct,
		config.SurgeMinMultiplier, config.SurgeMaxMultiplier,
		config.TaxRatePct, config.TaxInclusive, cancellationFeesJSON, config.IsActive,
	).Scan(&config.CreatedAt, &config.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}
	return nil
}

// GetConfigByID retrieves a pricing config by ID
func (r *Repository) GetConfigByID(ctx context.Context, id uuid.UUID) (*PricingConfig, error) {
	query := `
		SELECT id, version_id, country_id, region_id, city_id, zone_id, ride_type_id,
		       base_fare, per_km_rate, per_minute_rate, minimum_fare, booking_fee,
		       platform_commission_pct, driver_incentive_pct,
		       surge_min_multiplier, surge_max_multiplier,
		       tax_rate_pct, tax_inclusive, cancellation_fees, is_active, created_at, updated_at
		FROM pricing_configs
		WHERE id = $1
	`
	config := &PricingConfig{}
	var cancellationFeesJSON []byte
	err := r.db.QueryRow(ctx, query, id).Scan(
		&config.ID, &config.VersionID, &config.CountryID, &config.RegionID,
		&config.CityID, &config.ZoneID, &config.RideTypeID,
		&config.BaseFare, &config.PerKmRate, &config.PerMinuteRate,
		&config.MinimumFare, &config.BookingFee,
		&config.PlatformCommissionPct, &config.DriverIncentivePct,
		&config.SurgeMinMultiplier, &config.SurgeMaxMultiplier,
		&config.TaxRatePct, &config.TaxInclusive, &cancellationFeesJSON,
		&config.IsActive, &config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	if len(cancellationFeesJSON) > 0 {
		if err := json.Unmarshal(cancellationFeesJSON, &config.CancellationFees); err != nil {
			return nil, fmt.Errorf("failed to parse cancellation fees: %w", err)
		}
	}
	return config, nil
}

// ListConfigs lists pricing configs for a version with pagination
func (r *Repository) ListConfigs(ctx context.Context, versionID uuid.UUID, limit, offset int) ([]*PricingConfig, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM pricing_configs WHERE version_id = $1`, versionID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count configs: %w", err)
	}

	query := `
		SELECT id, version_id, country_id, region_id, city_id, zone_id, ride_type_id,
		       base_fare, per_km_rate, per_minute_rate, minimum_fare, booking_fee,
		       platform_commission_pct, driver_incentive_pct,
		       surge_min_multiplier, surge_max_multiplier,
		       tax_rate_pct, tax_inclusive, cancellation_fees, is_active, created_at, updated_at
		FROM pricing_configs
		WHERE version_id = $1
		ORDER BY
		  CASE
		    WHEN zone_id IS NOT NULL THEN 4
		    WHEN city_id IS NOT NULL THEN 3
		    WHEN region_id IS NOT NULL THEN 2
		    WHEN country_id IS NOT NULL THEN 1
		    ELSE 0
		  END, created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, versionID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list configs: %w", err)
	}
	defer rows.Close()

	configs := make([]*PricingConfig, 0)
	for rows.Next() {
		config := &PricingConfig{}
		var cancellationFeesJSON []byte
		err := rows.Scan(
			&config.ID, &config.VersionID, &config.CountryID, &config.RegionID,
			&config.CityID, &config.ZoneID, &config.RideTypeID,
			&config.BaseFare, &config.PerKmRate, &config.PerMinuteRate,
			&config.MinimumFare, &config.BookingFee,
			&config.PlatformCommissionPct, &config.DriverIncentivePct,
			&config.SurgeMinMultiplier, &config.SurgeMaxMultiplier,
			&config.TaxRatePct, &config.TaxInclusive, &cancellationFeesJSON,
			&config.IsActive, &config.CreatedAt, &config.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan config: %w", err)
		}
		if len(cancellationFeesJSON) > 0 {
			json.Unmarshal(cancellationFeesJSON, &config.CancellationFees)
		}
		configs = append(configs, config)
	}

	return configs, total, nil
}

// UpdateConfig updates a pricing config
func (r *Repository) UpdateConfig(ctx context.Context, config *PricingConfig) error {
	var cancellationFeesJSON []byte
	if len(config.CancellationFees) > 0 {
		var err error
		cancellationFeesJSON, err = json.Marshal(config.CancellationFees)
		if err != nil {
			return fmt.Errorf("failed to marshal cancellation fees: %w", err)
		}
	} else {
		cancellationFeesJSON = []byte("[]")
	}

	query := `
		UPDATE pricing_configs SET
			country_id = $2, region_id = $3, city_id = $4, zone_id = $5, ride_type_id = $6,
			base_fare = $7, per_km_rate = $8, per_minute_rate = $9, minimum_fare = $10, booking_fee = $11,
			platform_commission_pct = $12, driver_incentive_pct = $13,
			surge_min_multiplier = $14, surge_max_multiplier = $15,
			tax_rate_pct = $16, tax_inclusive = $17, cancellation_fees = $18,
			is_active = $19, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query,
		config.ID, config.CountryID, config.RegionID, config.CityID, config.ZoneID,
		config.RideTypeID, config.BaseFare, config.PerKmRate, config.PerMinuteRate,
		config.MinimumFare, config.BookingFee, config.PlatformCommissionPct,
		config.DriverIncentivePct, config.SurgeMinMultiplier, config.SurgeMaxMultiplier,
		config.TaxRatePct, config.TaxInclusive, cancellationFeesJSON, config.IsActive,
	).Scan(&config.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}
	return nil
}

// DeleteConfig deletes a pricing config
func (r *Repository) DeleteConfig(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM pricing_configs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}
	return nil
}

// ============================================================
// Time Multiplier CRUD
// ============================================================

// CreateTimeMultiplier creates a new time multiplier
func (r *Repository) CreateTimeMultiplier(ctx context.Context, m *TimeMultiplier) error {
	query := `
		INSERT INTO time_multipliers (id, version_id, country_id, region_id, city_id,
		       name, days_of_week, start_time, end_time, multiplier, priority, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at
	`
	m.ID = uuid.New()
	err := r.db.QueryRow(ctx, query,
		m.ID, m.VersionID, m.CountryID, m.RegionID, m.CityID,
		m.Name, m.DaysOfWeek, m.StartTime, m.EndTime, m.Multiplier, m.Priority, m.IsActive,
	).Scan(&m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create time multiplier: %w", err)
	}
	return nil
}

// GetTimeMultiplierByID retrieves a time multiplier by ID
func (r *Repository) GetTimeMultiplierByID(ctx context.Context, id uuid.UUID) (*TimeMultiplier, error) {
	query := `
		SELECT id, version_id, country_id, region_id, city_id, name,
		       days_of_week, start_time, end_time, multiplier, priority,
		       is_active, created_at, updated_at
		FROM time_multipliers WHERE id = $1
	`
	m := &TimeMultiplier{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.VersionID, &m.CountryID, &m.RegionID, &m.CityID,
		&m.Name, &m.DaysOfWeek, &m.StartTime, &m.EndTime, &m.Multiplier,
		&m.Priority, &m.IsActive, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get time multiplier: %w", err)
	}
	return m, nil
}

// ListTimeMultipliersByVersion lists time multipliers for a version
func (r *Repository) ListTimeMultipliersByVersion(ctx context.Context, versionID uuid.UUID, limit, offset int) ([]*TimeMultiplier, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM time_multipliers WHERE version_id = $1`, versionID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count time multipliers: %w", err)
	}

	query := `
		SELECT id, version_id, country_id, region_id, city_id, name,
		       days_of_week, start_time, end_time, multiplier, priority,
		       is_active, created_at, updated_at
		FROM time_multipliers WHERE version_id = $1
		ORDER BY priority DESC, name
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, versionID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list time multipliers: %w", err)
	}
	defer rows.Close()

	items := make([]*TimeMultiplier, 0)
	for rows.Next() {
		m := &TimeMultiplier{}
		err := rows.Scan(
			&m.ID, &m.VersionID, &m.CountryID, &m.RegionID, &m.CityID,
			&m.Name, &m.DaysOfWeek, &m.StartTime, &m.EndTime, &m.Multiplier,
			&m.Priority, &m.IsActive, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan time multiplier: %w", err)
		}
		items = append(items, m)
	}
	return items, total, nil
}

// UpdateTimeMultiplier updates a time multiplier
func (r *Repository) UpdateTimeMultiplier(ctx context.Context, m *TimeMultiplier) error {
	query := `
		UPDATE time_multipliers SET
			country_id = $2, region_id = $3, city_id = $4, name = $5,
			days_of_week = $6, start_time = $7, end_time = $8,
			multiplier = $9, priority = $10, is_active = $11, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query,
		m.ID, m.CountryID, m.RegionID, m.CityID, m.Name,
		m.DaysOfWeek, m.StartTime, m.EndTime, m.Multiplier, m.Priority, m.IsActive,
	).Scan(&m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update time multiplier: %w", err)
	}
	return nil
}

// DeleteTimeMultiplier deletes a time multiplier
func (r *Repository) DeleteTimeMultiplier(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM time_multipliers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete time multiplier: %w", err)
	}
	return nil
}

// ============================================================
// Weather Multiplier CRUD
// ============================================================

// CreateWeatherMultiplier creates a new weather multiplier
func (r *Repository) CreateWeatherMultiplier(ctx context.Context, m *WeatherMultiplier) error {
	query := `
		INSERT INTO weather_multipliers (id, version_id, country_id, region_id, city_id,
		       weather_condition, multiplier, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at
	`
	m.ID = uuid.New()
	err := r.db.QueryRow(ctx, query,
		m.ID, m.VersionID, m.CountryID, m.RegionID, m.CityID,
		m.WeatherCondition, m.Multiplier, m.IsActive,
	).Scan(&m.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create weather multiplier: %w", err)
	}
	return nil
}

// GetWeatherMultiplierByID retrieves a weather multiplier by ID
func (r *Repository) GetWeatherMultiplierByID(ctx context.Context, id uuid.UUID) (*WeatherMultiplier, error) {
	query := `
		SELECT id, version_id, country_id, region_id, city_id,
		       weather_condition, multiplier, is_active, created_at
		FROM weather_multipliers WHERE id = $1
	`
	m := &WeatherMultiplier{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.VersionID, &m.CountryID, &m.RegionID, &m.CityID,
		&m.WeatherCondition, &m.Multiplier, &m.IsActive, &m.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get weather multiplier: %w", err)
	}
	return m, nil
}

// ListWeatherMultipliersByVersion lists weather multipliers for a version
func (r *Repository) ListWeatherMultipliersByVersion(ctx context.Context, versionID uuid.UUID, limit, offset int) ([]*WeatherMultiplier, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM weather_multipliers WHERE version_id = $1`, versionID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count weather multipliers: %w", err)
	}

	query := `
		SELECT id, version_id, country_id, region_id, city_id,
		       weather_condition, multiplier, is_active, created_at
		FROM weather_multipliers WHERE version_id = $1
		ORDER BY weather_condition
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, versionID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list weather multipliers: %w", err)
	}
	defer rows.Close()

	items := make([]*WeatherMultiplier, 0)
	for rows.Next() {
		m := &WeatherMultiplier{}
		err := rows.Scan(
			&m.ID, &m.VersionID, &m.CountryID, &m.RegionID, &m.CityID,
			&m.WeatherCondition, &m.Multiplier, &m.IsActive, &m.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan weather multiplier: %w", err)
		}
		items = append(items, m)
	}
	return items, total, nil
}

// UpdateWeatherMultiplier updates a weather multiplier
func (r *Repository) UpdateWeatherMultiplier(ctx context.Context, m *WeatherMultiplier) error {
	_, err := r.db.Exec(ctx, `
		UPDATE weather_multipliers SET
			country_id = $2, region_id = $3, city_id = $4,
			weather_condition = $5, multiplier = $6, is_active = $7
		WHERE id = $1
	`, m.ID, m.CountryID, m.RegionID, m.CityID, m.WeatherCondition, m.Multiplier, m.IsActive)
	if err != nil {
		return fmt.Errorf("failed to update weather multiplier: %w", err)
	}
	return nil
}

// DeleteWeatherMultiplier deletes a weather multiplier
func (r *Repository) DeleteWeatherMultiplier(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM weather_multipliers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete weather multiplier: %w", err)
	}
	return nil
}

// ============================================================
// Event Multiplier CRUD
// ============================================================

// CreateEventMultiplier creates a new event multiplier
func (r *Repository) CreateEventMultiplier(ctx context.Context, m *EventMultiplier) error {
	query := `
		INSERT INTO event_multipliers (id, version_id, zone_id, city_id, event_name, event_type,
		       starts_at, ends_at, pre_event_minutes, post_event_minutes, multiplier,
		       expected_demand_increase, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING created_at
	`
	m.ID = uuid.New()
	err := r.db.QueryRow(ctx, query,
		m.ID, m.VersionID, m.ZoneID, m.CityID, m.EventName, m.EventType,
		m.StartsAt, m.EndsAt, m.PreEventMinutes, m.PostEventMinutes,
		m.Multiplier, m.ExpectedDemandIncrease, m.IsActive,
	).Scan(&m.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create event multiplier: %w", err)
	}
	return nil
}

// GetEventMultiplierByID retrieves an event multiplier by ID
func (r *Repository) GetEventMultiplierByID(ctx context.Context, id uuid.UUID) (*EventMultiplier, error) {
	query := `
		SELECT id, version_id, zone_id, city_id, event_name, event_type,
		       starts_at, ends_at, pre_event_minutes, post_event_minutes,
		       multiplier, expected_demand_increase, is_active, created_at
		FROM event_multipliers WHERE id = $1
	`
	m := &EventMultiplier{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.VersionID, &m.ZoneID, &m.CityID, &m.EventName,
		&m.EventType, &m.StartsAt, &m.EndsAt, &m.PreEventMinutes,
		&m.PostEventMinutes, &m.Multiplier, &m.ExpectedDemandIncrease,
		&m.IsActive, &m.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get event multiplier: %w", err)
	}
	return m, nil
}

// ListEventMultipliersByVersion lists event multipliers for a version
func (r *Repository) ListEventMultipliersByVersion(ctx context.Context, versionID uuid.UUID, limit, offset int) ([]*EventMultiplier, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM event_multipliers WHERE version_id = $1`, versionID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count event multipliers: %w", err)
	}

	query := `
		SELECT id, version_id, zone_id, city_id, event_name, event_type,
		       starts_at, ends_at, pre_event_minutes, post_event_minutes,
		       multiplier, expected_demand_increase, is_active, created_at
		FROM event_multipliers WHERE version_id = $1
		ORDER BY starts_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, versionID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list event multipliers: %w", err)
	}
	defer rows.Close()

	items := make([]*EventMultiplier, 0)
	for rows.Next() {
		m := &EventMultiplier{}
		err := rows.Scan(
			&m.ID, &m.VersionID, &m.ZoneID, &m.CityID, &m.EventName,
			&m.EventType, &m.StartsAt, &m.EndsAt, &m.PreEventMinutes,
			&m.PostEventMinutes, &m.Multiplier, &m.ExpectedDemandIncrease,
			&m.IsActive, &m.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan event multiplier: %w", err)
		}
		items = append(items, m)
	}
	return items, total, nil
}

// UpdateEventMultiplier updates an event multiplier
func (r *Repository) UpdateEventMultiplier(ctx context.Context, m *EventMultiplier) error {
	_, err := r.db.Exec(ctx, `
		UPDATE event_multipliers SET
			zone_id = $2, city_id = $3, event_name = $4, event_type = $5,
			starts_at = $6, ends_at = $7, pre_event_minutes = $8, post_event_minutes = $9,
			multiplier = $10, expected_demand_increase = $11, is_active = $12
		WHERE id = $1
	`, m.ID, m.ZoneID, m.CityID, m.EventName, m.EventType,
		m.StartsAt, m.EndsAt, m.PreEventMinutes, m.PostEventMinutes,
		m.Multiplier, m.ExpectedDemandIncrease, m.IsActive)
	if err != nil {
		return fmt.Errorf("failed to update event multiplier: %w", err)
	}
	return nil
}

// DeleteEventMultiplier deletes an event multiplier
func (r *Repository) DeleteEventMultiplier(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM event_multipliers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete event multiplier: %w", err)
	}
	return nil
}

// ============================================================
// Zone Fee CRUD
// ============================================================

// CreateZoneFee creates a new zone fee
func (r *Repository) CreateZoneFee(ctx context.Context, fee *ZoneFee) error {
	var scheduleJSON []byte
	if fee.Schedule != nil {
		var err error
		scheduleJSON, err = json.Marshal(fee.Schedule)
		if err != nil {
			return fmt.Errorf("failed to marshal schedule: %w", err)
		}
	}

	query := `
		INSERT INTO zone_fees (id, zone_id, version_id, fee_type, ride_type_id, amount,
		       is_percentage, applies_pickup, applies_dropoff, schedule, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at
	`
	fee.ID = uuid.New()
	err := r.db.QueryRow(ctx, query,
		fee.ID, fee.ZoneID, fee.VersionID, fee.FeeType, fee.RideTypeID,
		fee.Amount, fee.IsPercentage, fee.AppliesPickup, fee.AppliesDropoff,
		scheduleJSON, fee.IsActive,
	).Scan(&fee.CreatedAt, &fee.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create zone fee: %w", err)
	}
	return nil
}

// GetZoneFeeByID retrieves a zone fee by ID
func (r *Repository) GetZoneFeeByID(ctx context.Context, id uuid.UUID) (*ZoneFee, error) {
	query := `
		SELECT id, zone_id, version_id, fee_type, ride_type_id, amount,
		       is_percentage, applies_pickup, applies_dropoff, schedule,
		       is_active, created_at, updated_at
		FROM zone_fees WHERE id = $1
	`
	fee := &ZoneFee{}
	var scheduleJSON []byte
	err := r.db.QueryRow(ctx, query, id).Scan(
		&fee.ID, &fee.ZoneID, &fee.VersionID, &fee.FeeType, &fee.RideTypeID,
		&fee.Amount, &fee.IsPercentage, &fee.AppliesPickup, &fee.AppliesDropoff,
		&scheduleJSON, &fee.IsActive, &fee.CreatedAt, &fee.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone fee: %w", err)
	}
	if len(scheduleJSON) > 0 {
		fee.Schedule = &FeeSchedule{}
		json.Unmarshal(scheduleJSON, fee.Schedule)
	}
	return fee, nil
}

// ListZoneFeesByVersion lists zone fees for a version with optional zone filter
func (r *Repository) ListZoneFeesByVersion(ctx context.Context, versionID uuid.UUID, zoneID *uuid.UUID, limit, offset int) ([]*ZoneFee, int64, error) {
	whereClause := "WHERE version_id = $1"
	args := []interface{}{versionID}
	argIndex := 2

	if zoneID != nil {
		whereClause += fmt.Sprintf(" AND zone_id = $%d", argIndex)
		args = append(args, *zoneID)
		argIndex++
	}

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM zone_fees %s", whereClause)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count zone fees: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, zone_id, version_id, fee_type, ride_type_id, amount,
		       is_percentage, applies_pickup, applies_dropoff, schedule,
		       is_active, created_at, updated_at
		FROM zone_fees %s
		ORDER BY fee_type
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list zone fees: %w", err)
	}
	defer rows.Close()

	items := make([]*ZoneFee, 0)
	for rows.Next() {
		fee := &ZoneFee{}
		var scheduleJSON []byte
		err := rows.Scan(
			&fee.ID, &fee.ZoneID, &fee.VersionID, &fee.FeeType, &fee.RideTypeID,
			&fee.Amount, &fee.IsPercentage, &fee.AppliesPickup, &fee.AppliesDropoff,
			&scheduleJSON, &fee.IsActive, &fee.CreatedAt, &fee.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan zone fee: %w", err)
		}
		if len(scheduleJSON) > 0 {
			fee.Schedule = &FeeSchedule{}
			json.Unmarshal(scheduleJSON, fee.Schedule)
		}
		items = append(items, fee)
	}
	return items, total, nil
}

// UpdateZoneFee updates a zone fee
func (r *Repository) UpdateZoneFee(ctx context.Context, fee *ZoneFee) error {
	var scheduleJSON []byte
	if fee.Schedule != nil {
		var err error
		scheduleJSON, err = json.Marshal(fee.Schedule)
		if err != nil {
			return fmt.Errorf("failed to marshal schedule: %w", err)
		}
	}

	query := `
		UPDATE zone_fees SET
			zone_id = $2, fee_type = $3, ride_type_id = $4, amount = $5,
			is_percentage = $6, applies_pickup = $7, applies_dropoff = $8,
			schedule = $9, is_active = $10, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query,
		fee.ID, fee.ZoneID, fee.FeeType, fee.RideTypeID, fee.Amount,
		fee.IsPercentage, fee.AppliesPickup, fee.AppliesDropoff,
		scheduleJSON, fee.IsActive,
	).Scan(&fee.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update zone fee: %w", err)
	}
	return nil
}

// DeleteZoneFee deletes a zone fee
func (r *Repository) DeleteZoneFee(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM zone_fees WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete zone fee: %w", err)
	}
	return nil
}

// ============================================================
// Surge Threshold CRUD
// ============================================================

// CreateSurgeThreshold creates a new surge threshold
func (r *Repository) CreateSurgeThreshold(ctx context.Context, t *SurgeThreshold) error {
	query := `
		INSERT INTO surge_thresholds (id, version_id, country_id, region_id, city_id,
		       demand_supply_ratio_min, demand_supply_ratio_max, multiplier, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at
	`
	t.ID = uuid.New()
	err := r.db.QueryRow(ctx, query,
		t.ID, t.VersionID, t.CountryID, t.RegionID, t.CityID,
		t.DemandSupplyRatioMin, t.DemandSupplyRatioMax, t.Multiplier, t.IsActive,
	).Scan(&t.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create surge threshold: %w", err)
	}
	return nil
}

// GetSurgeThresholdByID retrieves a surge threshold by ID
func (r *Repository) GetSurgeThresholdByID(ctx context.Context, id uuid.UUID) (*SurgeThreshold, error) {
	query := `
		SELECT id, version_id, country_id, region_id, city_id,
		       demand_supply_ratio_min, demand_supply_ratio_max, multiplier,
		       is_active, created_at
		FROM surge_thresholds WHERE id = $1
	`
	t := &SurgeThreshold{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.VersionID, &t.CountryID, &t.RegionID, &t.CityID,
		&t.DemandSupplyRatioMin, &t.DemandSupplyRatioMax, &t.Multiplier,
		&t.IsActive, &t.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get surge threshold: %w", err)
	}
	return t, nil
}

// ListSurgeThresholdsByVersion lists surge thresholds for a version
func (r *Repository) ListSurgeThresholdsByVersion(ctx context.Context, versionID uuid.UUID, limit, offset int) ([]*SurgeThreshold, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM surge_thresholds WHERE version_id = $1`, versionID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count surge thresholds: %w", err)
	}

	query := `
		SELECT id, version_id, country_id, region_id, city_id,
		       demand_supply_ratio_min, demand_supply_ratio_max, multiplier,
		       is_active, created_at
		FROM surge_thresholds WHERE version_id = $1
		ORDER BY demand_supply_ratio_min
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, versionID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list surge thresholds: %w", err)
	}
	defer rows.Close()

	items := make([]*SurgeThreshold, 0)
	for rows.Next() {
		t := &SurgeThreshold{}
		err := rows.Scan(
			&t.ID, &t.VersionID, &t.CountryID, &t.RegionID, &t.CityID,
			&t.DemandSupplyRatioMin, &t.DemandSupplyRatioMax, &t.Multiplier,
			&t.IsActive, &t.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan surge threshold: %w", err)
		}
		items = append(items, t)
	}
	return items, total, nil
}

// UpdateSurgeThreshold updates a surge threshold
func (r *Repository) UpdateSurgeThreshold(ctx context.Context, t *SurgeThreshold) error {
	_, err := r.db.Exec(ctx, `
		UPDATE surge_thresholds SET
			country_id = $2, region_id = $3, city_id = $4,
			demand_supply_ratio_min = $5, demand_supply_ratio_max = $6,
			multiplier = $7, is_active = $8
		WHERE id = $1
	`, t.ID, t.CountryID, t.RegionID, t.CityID,
		t.DemandSupplyRatioMin, t.DemandSupplyRatioMax, t.Multiplier, t.IsActive)
	if err != nil {
		return fmt.Errorf("failed to update surge threshold: %w", err)
	}
	return nil
}

// DeleteSurgeThreshold deletes a surge threshold
func (r *Repository) DeleteSurgeThreshold(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM surge_thresholds WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete surge threshold: %w", err)
	}
	return nil
}

// ============================================================
// Pricing Audit Log
// ============================================================

// InsertPricingAuditLog records a pricing change in the audit log
func (r *Repository) InsertPricingAuditLog(ctx context.Context, adminID uuid.UUID, action, entityType string, entityID uuid.UUID, oldValues, newValues map[string]interface{}, reason string) {
	query := `
		INSERT INTO pricing_audit_logs (id, admin_id, action, entity_type, entity_id, old_values, new_values, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	var reasonPtr *string
	if reason != "" {
		reasonPtr = &reason
	}
	_, err := r.db.Exec(ctx, query, uuid.New(), adminID, action, entityType, entityID, oldValues, newValues, reasonPtr)
	if err != nil {
		logger.Warn("failed to insert pricing audit log", zap.Error(err))
	}
}

// GetPricingAuditLogs retrieves pricing audit logs with filtering
func (r *Repository) GetPricingAuditLogs(ctx context.Context, entityType string, entityID *uuid.UUID, limit, offset int) ([]*PricingAuditLog, int64, error) {
	whereClause := "WHERE 1=1"
	args := make([]interface{}, 0)
	argIndex := 1

	if entityType != "" {
		whereClause += fmt.Sprintf(" AND entity_type = $%d", argIndex)
		args = append(args, entityType)
		argIndex++
	}
	if entityID != nil {
		whereClause += fmt.Sprintf(" AND entity_id = $%d", argIndex)
		args = append(args, *entityID)
		argIndex++
	}

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM pricing_audit_logs %s", whereClause)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, admin_id, action, entity_type, entity_id, old_values, new_values, reason, created_at
		FROM pricing_audit_logs %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get audit logs: %w", err)
	}
	defer rows.Close()

	logs := make([]*PricingAuditLog, 0)
	for rows.Next() {
		l := &PricingAuditLog{}
		err := rows.Scan(
			&l.ID, &l.AdminID, &l.Action, &l.EntityType, &l.EntityID,
			&l.OldValues, &l.NewValues, &l.Reason, &l.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, l)
	}

	return logs, total, nil
}
