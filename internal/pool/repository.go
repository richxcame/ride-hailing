package pool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles pool ride database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new pool repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// POOL RIDE OPERATIONS
// ========================================

// CreatePoolRide creates a new pool ride
func (r *Repository) CreatePoolRide(ctx context.Context, pool *PoolRide) error {
	routeJSON, _ := json.Marshal(pool.OptimizedRoute)

	query := `
		INSERT INTO pool_rides (
			id, driver_id, vehicle_id, status, max_passengers, current_passengers,
			optimized_route, total_distance_km, total_duration_minutes,
			center_latitude, center_longitude, radius_km, h3_index,
			base_fare, per_km_rate, per_minute_rate,
			match_deadline, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`

	_, err := r.db.Exec(ctx, query,
		pool.ID, pool.DriverID, pool.VehicleID, pool.Status, pool.MaxPassengers, pool.CurrentPassengers,
		routeJSON, pool.TotalDistance, pool.TotalDuration,
		pool.CenterLatitude, pool.CenterLongitude, pool.RadiusKm, pool.H3Index,
		pool.BaseFare, pool.PerKmRate, pool.PerMinuteRate,
		pool.MatchDeadline, pool.CreatedAt, pool.UpdatedAt,
	)
	return err
}

// GetPoolRide gets a pool ride by ID
func (r *Repository) GetPoolRide(ctx context.Context, poolID uuid.UUID) (*PoolRide, error) {
	query := `
		SELECT id, driver_id, vehicle_id, status, max_passengers, current_passengers,
			optimized_route, total_distance_km, total_duration_minutes,
			center_latitude, center_longitude, radius_km, h3_index,
			base_fare, per_km_rate, per_minute_rate,
			match_deadline, started_at, completed_at, created_at, updated_at
		FROM pool_rides
		WHERE id = $1
	`

	var pool PoolRide
	var routeJSON []byte
	err := r.db.QueryRow(ctx, query, poolID).Scan(
		&pool.ID, &pool.DriverID, &pool.VehicleID, &pool.Status, &pool.MaxPassengers, &pool.CurrentPassengers,
		&routeJSON, &pool.TotalDistance, &pool.TotalDuration,
		&pool.CenterLatitude, &pool.CenterLongitude, &pool.RadiusKm, &pool.H3Index,
		&pool.BaseFare, &pool.PerKmRate, &pool.PerMinuteRate,
		&pool.MatchDeadline, &pool.StartedAt, &pool.CompletedAt, &pool.CreatedAt, &pool.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if routeJSON != nil {
		_ = json.Unmarshal(routeJSON, &pool.OptimizedRoute)
	}

	return &pool, nil
}

// GetActivePoolRides gets active pool rides for matching
func (r *Repository) GetActivePoolRides(ctx context.Context, h3Index string, status PoolStatus) ([]*PoolRide, error) {
	query := `
		SELECT id, driver_id, vehicle_id, status, max_passengers, current_passengers,
			optimized_route, total_distance_km, total_duration_minutes,
			center_latitude, center_longitude, radius_km, h3_index,
			base_fare, per_km_rate, per_minute_rate,
			match_deadline, started_at, completed_at, created_at, updated_at
		FROM pool_rides
		WHERE status = $1
			AND h3_index = $2
			AND match_deadline > NOW()
			AND current_passengers < max_passengers
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, status, h3Index)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pools []*PoolRide
	for rows.Next() {
		var pool PoolRide
		var routeJSON []byte
		err := rows.Scan(
			&pool.ID, &pool.DriverID, &pool.VehicleID, &pool.Status, &pool.MaxPassengers, &pool.CurrentPassengers,
			&routeJSON, &pool.TotalDistance, &pool.TotalDuration,
			&pool.CenterLatitude, &pool.CenterLongitude, &pool.RadiusKm, &pool.H3Index,
			&pool.BaseFare, &pool.PerKmRate, &pool.PerMinuteRate,
			&pool.MatchDeadline, &pool.StartedAt, &pool.CompletedAt, &pool.CreatedAt, &pool.UpdatedAt,
		)
		if err != nil {
			continue
		}
		if routeJSON != nil {
			_ = json.Unmarshal(routeJSON, &pool.OptimizedRoute)
		}
		pools = append(pools, &pool)
	}

	return pools, nil
}

// GetNearbyPoolRides gets pool rides within a radius using H3
func (r *Repository) GetNearbyPoolRides(ctx context.Context, h3Indexes []string, maxPassengers int) ([]*PoolRide, error) {
	query := `
		SELECT id, driver_id, vehicle_id, status, max_passengers, current_passengers,
			optimized_route, total_distance_km, total_duration_minutes,
			center_latitude, center_longitude, radius_km, h3_index,
			base_fare, per_km_rate, per_minute_rate,
			match_deadline, started_at, completed_at, created_at, updated_at
		FROM pool_rides
		WHERE status = 'matching'
			AND h3_index = ANY($1)
			AND match_deadline > NOW()
			AND current_passengers + $2 <= max_passengers
		ORDER BY created_at ASC
		LIMIT 10
	`

	rows, err := r.db.Query(ctx, query, h3Indexes, maxPassengers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pools []*PoolRide
	for rows.Next() {
		var pool PoolRide
		var routeJSON []byte
		err := rows.Scan(
			&pool.ID, &pool.DriverID, &pool.VehicleID, &pool.Status, &pool.MaxPassengers, &pool.CurrentPassengers,
			&routeJSON, &pool.TotalDistance, &pool.TotalDuration,
			&pool.CenterLatitude, &pool.CenterLongitude, &pool.RadiusKm, &pool.H3Index,
			&pool.BaseFare, &pool.PerKmRate, &pool.PerMinuteRate,
			&pool.MatchDeadline, &pool.StartedAt, &pool.CompletedAt, &pool.CreatedAt, &pool.UpdatedAt,
		)
		if err != nil {
			continue
		}
		if routeJSON != nil {
			_ = json.Unmarshal(routeJSON, &pool.OptimizedRoute)
		}
		pools = append(pools, &pool)
	}

	return pools, nil
}

// UpdatePoolStatus updates pool ride status
func (r *Repository) UpdatePoolStatus(ctx context.Context, poolID uuid.UUID, status PoolStatus) error {
	query := `UPDATE pool_rides SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, status, poolID)
	return err
}

// UpdatePoolRoute updates the optimized route
func (r *Repository) UpdatePoolRoute(ctx context.Context, poolID uuid.UUID, route []RouteStop, totalDistance float64, totalDuration int) error {
	routeJSON, _ := json.Marshal(route)
	query := `
		UPDATE pool_rides
		SET optimized_route = $1, total_distance_km = $2, total_duration_minutes = $3, updated_at = NOW()
		WHERE id = $4
	`
	_, err := r.db.Exec(ctx, query, routeJSON, totalDistance, totalDuration, poolID)
	return err
}

// IncrementPassengerCount increments the passenger count
func (r *Repository) IncrementPassengerCount(ctx context.Context, poolID uuid.UUID, count int) error {
	query := `UPDATE pool_rides SET current_passengers = current_passengers + $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, count, poolID)
	return err
}

// AssignDriverToPool assigns a driver to a pool ride
func (r *Repository) AssignDriverToPool(ctx context.Context, poolID, driverID, vehicleID uuid.UUID) error {
	query := `
		UPDATE pool_rides
		SET driver_id = $1, vehicle_id = $2, status = 'confirmed', updated_at = NOW()
		WHERE id = $3
	`
	_, err := r.db.Exec(ctx, query, driverID, vehicleID, poolID)
	return err
}

// StartPoolRide marks the pool ride as started
func (r *Repository) StartPoolRide(ctx context.Context, poolID uuid.UUID) error {
	query := `UPDATE pool_rides SET status = 'in_progress', started_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, poolID)
	return err
}

// CompletePoolRide marks the pool ride as completed
func (r *Repository) CompletePoolRide(ctx context.Context, poolID uuid.UUID) error {
	query := `UPDATE pool_rides SET status = 'completed', completed_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, poolID)
	return err
}

// ========================================
// POOL PASSENGER OPERATIONS
// ========================================

// CreatePoolPassenger creates a new pool passenger
func (r *Repository) CreatePoolPassenger(ctx context.Context, passenger *PoolPassenger) error {
	query := `
		INSERT INTO pool_passengers (
			id, pool_ride_id, rider_id, status,
			pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude,
			pickup_address, dropoff_address,
			direct_distance_km, direct_duration_minutes,
			original_fare, pool_fare, savings_percent,
			estimated_pickup, estimated_dropoff,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`

	_, err := r.db.Exec(ctx, query,
		passenger.ID, passenger.PoolRideID, passenger.RiderID, passenger.Status,
		passenger.PickupLocation.Latitude, passenger.PickupLocation.Longitude,
		passenger.DropoffLocation.Latitude, passenger.DropoffLocation.Longitude,
		passenger.PickupAddress, passenger.DropoffAddress,
		passenger.DirectDistance, passenger.DirectDuration,
		passenger.OriginalFare, passenger.PoolFare, passenger.SavingsPercent,
		passenger.EstimatedPickup, passenger.EstimatedDropoff,
		passenger.CreatedAt, passenger.UpdatedAt,
	)
	return err
}

// GetPoolPassenger gets a pool passenger by ID
func (r *Repository) GetPoolPassenger(ctx context.Context, passengerID uuid.UUID) (*PoolPassenger, error) {
	query := `
		SELECT id, pool_ride_id, rider_id, status,
			pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude,
			pickup_address, dropoff_address,
			direct_distance_km, direct_duration_minutes,
			original_fare, pool_fare, savings_percent,
			picked_up_at, dropped_off_at, estimated_pickup, estimated_dropoff,
			rider_rating, driver_rating, created_at, updated_at
		FROM pool_passengers
		WHERE id = $1
	`

	var p PoolPassenger
	var pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude float64
	err := r.db.QueryRow(ctx, query, passengerID).Scan(
		&p.ID, &p.PoolRideID, &p.RiderID, &p.Status,
		&pickupLatitude, &pickupLongitude, &dropoffLatitude, &dropoffLongitude,
		&p.PickupAddress, &p.DropoffAddress,
		&p.DirectDistance, &p.DirectDuration,
		&p.OriginalFare, &p.PoolFare, &p.SavingsPercent,
		&p.PickedUpAt, &p.DroppedOffAt, &p.EstimatedPickup, &p.EstimatedDropoff,
		&p.RiderRating, &p.DriverRating, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	p.PickupLocation = Location{Latitude: pickupLatitude, Longitude: pickupLongitude}
	p.DropoffLocation = Location{Latitude: dropoffLatitude, Longitude: dropoffLongitude}

	return &p, nil
}

// GetPoolPassengerByRider gets a pool passenger by rider ID and pool ride ID
func (r *Repository) GetPoolPassengerByRider(ctx context.Context, riderID, poolRideID uuid.UUID) (*PoolPassenger, error) {
	query := `
		SELECT id, pool_ride_id, rider_id, status,
			pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude,
			pickup_address, dropoff_address,
			direct_distance_km, direct_duration_minutes,
			original_fare, pool_fare, savings_percent,
			picked_up_at, dropped_off_at, estimated_pickup, estimated_dropoff,
			rider_rating, driver_rating, created_at, updated_at
		FROM pool_passengers
		WHERE rider_id = $1 AND pool_ride_id = $2
	`

	var p PoolPassenger
	var pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude float64
	err := r.db.QueryRow(ctx, query, riderID, poolRideID).Scan(
		&p.ID, &p.PoolRideID, &p.RiderID, &p.Status,
		&pickupLatitude, &pickupLongitude, &dropoffLatitude, &dropoffLongitude,
		&p.PickupAddress, &p.DropoffAddress,
		&p.DirectDistance, &p.DirectDuration,
		&p.OriginalFare, &p.PoolFare, &p.SavingsPercent,
		&p.PickedUpAt, &p.DroppedOffAt, &p.EstimatedPickup, &p.EstimatedDropoff,
		&p.RiderRating, &p.DriverRating, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	p.PickupLocation = Location{Latitude: pickupLatitude, Longitude: pickupLongitude}
	p.DropoffLocation = Location{Latitude: dropoffLatitude, Longitude: dropoffLongitude}

	return &p, nil
}

// GetPassengersForPool gets all passengers for a pool ride
func (r *Repository) GetPassengersForPool(ctx context.Context, poolRideID uuid.UUID) ([]*PoolPassenger, error) {
	query := `
		SELECT id, pool_ride_id, rider_id, status,
			pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude,
			pickup_address, dropoff_address,
			direct_distance_km, direct_duration_minutes,
			original_fare, pool_fare, savings_percent,
			picked_up_at, dropped_off_at, estimated_pickup, estimated_dropoff,
			rider_rating, driver_rating, created_at, updated_at
		FROM pool_passengers
		WHERE pool_ride_id = $1
		ORDER BY estimated_pickup ASC
	`

	rows, err := r.db.Query(ctx, query, poolRideID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var passengers []*PoolPassenger
	for rows.Next() {
		var p PoolPassenger
		var pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude float64
		err := rows.Scan(
			&p.ID, &p.PoolRideID, &p.RiderID, &p.Status,
			&pickupLatitude, &pickupLongitude, &dropoffLatitude, &dropoffLongitude,
			&p.PickupAddress, &p.DropoffAddress,
			&p.DirectDistance, &p.DirectDuration,
			&p.OriginalFare, &p.PoolFare, &p.SavingsPercent,
			&p.PickedUpAt, &p.DroppedOffAt, &p.EstimatedPickup, &p.EstimatedDropoff,
			&p.RiderRating, &p.DriverRating, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			continue
		}
		p.PickupLocation = Location{Latitude: pickupLatitude, Longitude: pickupLongitude}
		p.DropoffLocation = Location{Latitude: dropoffLatitude, Longitude: dropoffLongitude}
		passengers = append(passengers, &p)
	}

	return passengers, nil
}

// UpdatePassengerStatus updates passenger status
func (r *Repository) UpdatePassengerStatus(ctx context.Context, passengerID uuid.UUID, status PassengerStatus) error {
	query := `UPDATE pool_passengers SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, status, passengerID)
	return err
}

// UpdatePassengerPickedUp marks passenger as picked up
func (r *Repository) UpdatePassengerPickedUp(ctx context.Context, passengerID uuid.UUID) error {
	query := `UPDATE pool_passengers SET status = 'picked_up', picked_up_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, passengerID)
	return err
}

// UpdatePassengerDroppedOff marks passenger as dropped off
func (r *Repository) UpdatePassengerDroppedOff(ctx context.Context, passengerID uuid.UUID) error {
	query := `UPDATE pool_passengers SET status = 'dropped_off', dropped_off_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, passengerID)
	return err
}

// GetActivePassengerForRider gets active pool passenger for a rider
func (r *Repository) GetActivePassengerForRider(ctx context.Context, riderID uuid.UUID) (*PoolPassenger, error) {
	query := `
		SELECT id, pool_ride_id, rider_id, status,
			pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude,
			pickup_address, dropoff_address,
			direct_distance_km, direct_duration_minutes,
			original_fare, pool_fare, savings_percent,
			picked_up_at, dropped_off_at, estimated_pickup, estimated_dropoff,
			rider_rating, driver_rating, created_at, updated_at
		FROM pool_passengers
		WHERE rider_id = $1 AND status NOT IN ('dropped_off', 'cancelled', 'no_show')
		ORDER BY created_at DESC
		LIMIT 1
	`

	var p PoolPassenger
	var pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude float64
	err := r.db.QueryRow(ctx, query, riderID).Scan(
		&p.ID, &p.PoolRideID, &p.RiderID, &p.Status,
		&pickupLatitude, &pickupLongitude, &dropoffLatitude, &dropoffLongitude,
		&p.PickupAddress, &p.DropoffAddress,
		&p.DirectDistance, &p.DirectDuration,
		&p.OriginalFare, &p.PoolFare, &p.SavingsPercent,
		&p.PickedUpAt, &p.DroppedOffAt, &p.EstimatedPickup, &p.EstimatedDropoff,
		&p.RiderRating, &p.DriverRating, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	p.PickupLocation = Location{Latitude: pickupLatitude, Longitude: pickupLongitude}
	p.DropoffLocation = Location{Latitude: dropoffLatitude, Longitude: dropoffLongitude}

	return &p, nil
}

// ========================================
// CONFIG OPERATIONS
// ========================================

// GetPoolConfig gets pool configuration for a city (or global if city is nil)
func (r *Repository) GetPoolConfig(ctx context.Context, cityID *uuid.UUID) (*PoolConfig, error) {
	var query string
	var config PoolConfig
	var err error

	if cityID != nil {
		query = `
			SELECT id, city_id, max_detour_percent, max_detour_minutes, max_wait_minutes,
				max_passengers_per_ride, min_match_score, match_radius_km, h3_resolution,
				discount_percent, min_savings_percent,
				allow_different_dropoffs, allow_mid_route_pickup,
				is_active, created_at, updated_at
			FROM pool_configs
			WHERE (city_id = $1 OR city_id IS NULL) AND is_active = true
			ORDER BY city_id NULLS LAST
			LIMIT 1
		`
		err = r.db.QueryRow(ctx, query, cityID).Scan(
			&config.ID, &config.CityID, &config.MaxDetourPercent, &config.MaxDetourMinutes, &config.MaxWaitMinutes,
			&config.MaxPassengersPerRide, &config.MinMatchScore, &config.MatchRadiusKm, &config.H3Resolution,
			&config.DiscountPercent, &config.MinSavingsPercent,
			&config.AllowDifferentDropoffs, &config.AllowMidRoutePickup,
			&config.IsActive, &config.CreatedAt, &config.UpdatedAt,
		)
	} else {
		query = `
			SELECT id, city_id, max_detour_percent, max_detour_minutes, max_wait_minutes,
				max_passengers_per_ride, min_match_score, match_radius_km, h3_resolution,
				discount_percent, min_savings_percent,
				allow_different_dropoffs, allow_mid_route_pickup,
				is_active, created_at, updated_at
			FROM pool_configs
			WHERE city_id IS NULL AND is_active = true
			LIMIT 1
		`
		err = r.db.QueryRow(ctx, query).Scan(
			&config.ID, &config.CityID, &config.MaxDetourPercent, &config.MaxDetourMinutes, &config.MaxWaitMinutes,
			&config.MaxPassengersPerRide, &config.MinMatchScore, &config.MatchRadiusKm, &config.H3Resolution,
			&config.DiscountPercent, &config.MinSavingsPercent,
			&config.AllowDifferentDropoffs, &config.AllowMidRoutePickup,
			&config.IsActive, &config.CreatedAt, &config.UpdatedAt,
		)
	}

	if err != nil {
		return nil, err
	}

	return &config, nil
}

// ========================================
// STATISTICS
// ========================================

// GetPoolStats gets pool ride statistics
func (r *Repository) GetPoolStats(ctx context.Context) (*PoolStatsResponse, error) {
	stats := &PoolStatsResponse{}

	// Total pool rides
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM pool_rides`).Scan(&stats.TotalPoolRides)
	if err != nil {
		return nil, err
	}

	// Active pool rides
	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM pool_rides WHERE status IN ('matching', 'confirmed', 'in_progress')`).Scan(&stats.ActivePoolRides)
	if err != nil {
		return nil, err
	}

	// Average passengers per pool
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(current_passengers), 0) FROM pool_rides WHERE status = 'completed'
	`).Scan(&stats.AvgPassengersPerPool)
	if err != nil {
		return nil, err
	}

	// Average savings percent
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(savings_percent), 0) FROM pool_passengers WHERE status = 'dropped_off'
	`).Scan(&stats.AvgSavingsPercent)
	if err != nil {
		return nil, err
	}

	// Calculate match success rate
	var totalRequests, matchedRequests int64
	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM pool_passengers`).Scan(&totalRequests)
	if err == nil {
		_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM pool_passengers WHERE status != 'cancelled'`).Scan(&matchedRequests)
		if totalRequests > 0 {
			stats.MatchSuccessRate = float64(matchedRequests) / float64(totalRequests) * 100
		}
	}

	// Estimate CO2 saved (roughly 0.21 kg CO2 per km saved)
	var totalKmSaved float64
	_ = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(direct_distance_km * savings_percent / 100), 0)
		FROM pool_passengers
		WHERE status = 'dropped_off'
	`).Scan(&totalKmSaved)
	stats.TotalCO2Saved = totalKmSaved * 0.21

	return stats, nil
}

// GetDriverActivePool gets the active pool ride for a driver
func (r *Repository) GetDriverActivePool(ctx context.Context, driverID uuid.UUID) (*PoolRide, error) {
	query := `
		SELECT id, driver_id, vehicle_id, status, max_passengers, current_passengers,
			optimized_route, total_distance_km, total_duration_minutes,
			center_latitude, center_longitude, radius_km, h3_index,
			base_fare, per_km_rate, per_minute_rate,
			match_deadline, started_at, completed_at, created_at, updated_at
		FROM pool_rides
		WHERE driver_id = $1 AND status IN ('confirmed', 'in_progress')
		ORDER BY created_at DESC
		LIMIT 1
	`

	var pool PoolRide
	var routeJSON []byte
	err := r.db.QueryRow(ctx, query, driverID).Scan(
		&pool.ID, &pool.DriverID, &pool.VehicleID, &pool.Status, &pool.MaxPassengers, &pool.CurrentPassengers,
		&routeJSON, &pool.TotalDistance, &pool.TotalDuration,
		&pool.CenterLatitude, &pool.CenterLongitude, &pool.RadiusKm, &pool.H3Index,
		&pool.BaseFare, &pool.PerKmRate, &pool.PerMinuteRate,
		&pool.MatchDeadline, &pool.StartedAt, &pool.CompletedAt, &pool.CreatedAt, &pool.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if routeJSON != nil {
		_ = json.Unmarshal(routeJSON, &pool.OptimizedRoute)
	}

	return &pool, nil
}

// CancelExpiredPoolRides cancels pool rides that passed their match deadline
func (r *Repository) CancelExpiredPoolRides(ctx context.Context) (int64, error) {
	query := `
		UPDATE pool_rides
		SET status = 'cancelled', updated_at = NOW()
		WHERE status = 'matching' AND match_deadline < NOW()
	`
	result, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// GetUnmatchedPassengers gets passengers that need refund due to failed matching
func (r *Repository) GetUnmatchedPassengers(ctx context.Context, poolRideID uuid.UUID) ([]*PoolPassenger, error) {
	query := `
		SELECT id, pool_ride_id, rider_id, status,
			pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude,
			pickup_address, dropoff_address,
			direct_distance_km, direct_duration_minutes,
			original_fare, pool_fare, savings_percent,
			picked_up_at, dropped_off_at, estimated_pickup, estimated_dropoff,
			rider_rating, driver_rating, created_at, updated_at
		FROM pool_passengers
		WHERE pool_ride_id = $1 AND status = 'pending'
	`

	rows, err := r.db.Query(ctx, query, poolRideID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var passengers []*PoolPassenger
	for rows.Next() {
		var p PoolPassenger
		var pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude float64
		err := rows.Scan(
			&p.ID, &p.PoolRideID, &p.RiderID, &p.Status,
			&pickupLatitude, &pickupLongitude, &dropoffLatitude, &dropoffLongitude,
			&p.PickupAddress, &p.DropoffAddress,
			&p.DirectDistance, &p.DirectDuration,
			&p.OriginalFare, &p.PoolFare, &p.SavingsPercent,
			&p.PickedUpAt, &p.DroppedOffAt, &p.EstimatedPickup, &p.EstimatedDropoff,
			&p.RiderRating, &p.DriverRating, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			continue
		}
		p.PickupLocation = Location{Latitude: pickupLatitude, Longitude: pickupLongitude}
		p.DropoffLocation = Location{Latitude: dropoffLatitude, Longitude: dropoffLongitude}
		passengers = append(passengers, &p)
	}

	return passengers, nil
}

// Helper for building query with dynamic conditions
func buildPoolSearchQuery(baseQuery string, conditions []string, args []interface{}) (string, []interface{}) {
	if len(conditions) == 0 {
		return baseQuery, args
	}

	query := baseQuery + " WHERE "
	for i, cond := range conditions {
		if i > 0 {
			query += " AND "
		}
		query += fmt.Sprintf("(%s)", cond)
	}
	return query, args
}
