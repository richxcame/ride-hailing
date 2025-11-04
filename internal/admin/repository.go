package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// Repository handles database operations for admin functions
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new admin repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// GetAllUsers retrieves all users with pagination
func (r *Repository) GetAllUsers(ctx context.Context, limit, offset int) ([]*models.User, int, error) {
	// Get total count
	var total int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user count: %w", err)
	}

	// Get users
	query := `
		SELECT id, email, phone_number, first_name, last_name, role, is_active, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PhoneNumber,
			&user.FirstName,
			&user.LastName,
			&user.Role,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, total, nil
}

// GetUserByID retrieves a user by ID
func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, email, phone_number, first_name, last_name, role, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &models.User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PhoneNumber,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// UpdateUserStatus updates a user's active status
func (r *Repository) UpdateUserStatus(ctx context.Context, id uuid.UUID, isActive bool) error {
	query := `UPDATE users SET is_active = $1, updated_at = $2 WHERE id = $3`

	result, err := r.db.Exec(ctx, query, isActive, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// GetPendingDrivers retrieves all drivers (simplified - no approval field in current model)
func (r *Repository) GetPendingDrivers(ctx context.Context) ([]*models.Driver, error) {
	query := `
		SELECT d.id, d.user_id, d.license_number, d.vehicle_model, d.vehicle_plate,
		       d.vehicle_color, d.vehicle_year, d.is_available, d.is_online,
		       d.rating, d.total_rides, d.created_at, d.updated_at
		FROM drivers d
		ORDER BY d.created_at DESC
		LIMIT 100
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get drivers: %w", err)
	}
	defer rows.Close()

	var drivers []*models.Driver
	for rows.Next() {
		driver := &models.Driver{}

		err := rows.Scan(
			&driver.ID,
			&driver.UserID,
			&driver.LicenseNumber,
			&driver.VehicleModel,
			&driver.VehiclePlate,
			&driver.VehicleColor,
			&driver.VehicleYear,
			&driver.IsAvailable,
			&driver.IsOnline,
			&driver.Rating,
			&driver.TotalRides,
			&driver.CreatedAt,
			&driver.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan driver: %w", err)
		}

		drivers = append(drivers, driver)
	}

	return drivers, nil
}

// ApproveDriver sets a driver as available (simplified approval)
func (r *Repository) ApproveDriver(ctx context.Context, driverID uuid.UUID) error {
	query := `UPDATE drivers SET is_available = true, updated_at = $1 WHERE id = $2`

	result, err := r.db.Exec(ctx, query, time.Now(), driverID)
	if err != nil {
		return fmt.Errorf("failed to approve driver: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("driver not found")
	}

	return nil
}

// RejectDriver sets a driver as unavailable
func (r *Repository) RejectDriver(ctx context.Context, driverID uuid.UUID) error {
	query := `UPDATE drivers SET is_available = false, updated_at = $1 WHERE id = $2`

	result, err := r.db.Exec(ctx, query, time.Now(), driverID)
	if err != nil {
		return fmt.Errorf("failed to reject driver: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("driver not found")
	}

	return nil
}

// GetRideStats retrieves ride statistics
func (r *Repository) GetRideStats(ctx context.Context, startDate, endDate *time.Time) (*RideStats, error) {
	query := `
		SELECT
			COUNT(*) as total_rides,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_rides,
			COUNT(CASE WHEN status = 'cancelled' THEN 1 END) as cancelled_rides,
			COUNT(CASE WHEN status = 'in_progress' THEN 1 END) as active_rides,
			COALESCE(SUM(CASE WHEN status = 'completed' THEN final_fare ELSE 0 END), 0) as total_revenue,
			COALESCE(AVG(CASE WHEN status = 'completed' THEN final_fare ELSE NULL END), 0) as avg_fare
		FROM rides
		WHERE ($1::timestamp IS NULL OR created_at >= $1)
		  AND ($2::timestamp IS NULL OR created_at <= $2)
	`

	stats := &RideStats{}
	err := r.db.QueryRow(ctx, query, startDate, endDate).Scan(
		&stats.TotalRides,
		&stats.CompletedRides,
		&stats.CancelledRides,
		&stats.ActiveRides,
		&stats.TotalRevenue,
		&stats.AvgFare,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get ride stats: %w", err)
	}

	return stats, nil
}

// GetUserStats retrieves user statistics
func (r *Repository) GetUserStats(ctx context.Context) (*UserStats, error) {
	query := `
		SELECT
			COUNT(*) as total_users,
			COUNT(CASE WHEN role = 'rider' THEN 1 END) as total_riders,
			COUNT(CASE WHEN role = 'driver' THEN 1 END) as total_drivers,
			COUNT(CASE WHEN is_active = true THEN 1 END) as active_users
		FROM users
	`

	stats := &UserStats{}
	err := r.db.QueryRow(ctx, query).Scan(
		&stats.TotalUsers,
		&stats.TotalRiders,
		&stats.TotalDrivers,
		&stats.ActiveUsers,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	return stats, nil
}

// GetRecentRides retrieves recent rides for monitoring
func (r *Repository) GetRecentRides(ctx context.Context, limit int) ([]*models.Ride, error) {
	query := `
		SELECT id, rider_id, driver_id, status, pickup_latitude, pickup_longitude,
		       pickup_address, dropoff_latitude, dropoff_longitude, dropoff_address,
		       estimated_distance, estimated_duration, estimated_fare, actual_distance,
		       actual_duration, final_fare, surge_multiplier, requested_at, accepted_at,
		       started_at, completed_at, cancelled_at, cancellation_reason, rating,
		       feedback, created_at, updated_at
		FROM rides
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent rides: %w", err)
	}
	defer rows.Close()

	var rides []*models.Ride
	for rows.Next() {
		ride := &models.Ride{}
		err := rows.Scan(
			&ride.ID,
			&ride.RiderID,
			&ride.DriverID,
			&ride.Status,
			&ride.PickupLatitude,
			&ride.PickupLongitude,
			&ride.PickupAddress,
			&ride.DropoffLatitude,
			&ride.DropoffLongitude,
			&ride.DropoffAddress,
			&ride.EstimatedDistance,
			&ride.EstimatedDuration,
			&ride.EstimatedFare,
			&ride.ActualDistance,
			&ride.ActualDuration,
			&ride.FinalFare,
			&ride.SurgeMultiplier,
			&ride.RequestedAt,
			&ride.AcceptedAt,
			&ride.StartedAt,
			&ride.CompletedAt,
			&ride.CancelledAt,
			&ride.CancellationReason,
			&ride.Rating,
			&ride.Feedback,
			&ride.CreatedAt,
			&ride.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ride: %w", err)
		}
		rides = append(rides, ride)
	}

	return rides, nil
}

// RideStats represents ride statistics
type RideStats struct {
	TotalRides     int     `json:"total_rides"`
	CompletedRides int     `json:"completed_rides"`
	CancelledRides int     `json:"cancelled_rides"`
	ActiveRides    int     `json:"active_rides"`
	TotalRevenue   float64 `json:"total_revenue"`
	AvgFare        float64 `json:"avg_fare"`
}

// UserStats represents user statistics
type UserStats struct {
	TotalUsers   int `json:"total_users"`
	TotalRiders  int `json:"total_riders"`
	TotalDrivers int `json:"total_drivers"`
	ActiveUsers  int `json:"active_users"`
}
