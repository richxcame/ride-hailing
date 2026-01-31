package rides

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// Repository handles database operations for rides
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new rides repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateRide creates a new ride request
func (r *Repository) CreateRide(ctx context.Context, ride *models.Ride) error {
	query := `
		INSERT INTO rides (
			id, rider_id, status, pickup_latitude, pickup_longitude, pickup_address,
			dropoff_latitude, dropoff_longitude, dropoff_address, estimated_distance,
			estimated_duration, estimated_fare, surge_multiplier, requested_at,
			ride_type_id, promo_code_id, discount_amount, scheduled_at, is_scheduled,
			scheduled_notification_sent
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		ride.ID,
		ride.RiderID,
		ride.Status,
		ride.PickupLatitude,
		ride.PickupLongitude,
		ride.PickupAddress,
		ride.DropoffLatitude,
		ride.DropoffLongitude,
		ride.DropoffAddress,
		ride.EstimatedDistance,
		ride.EstimatedDuration,
		ride.EstimatedFare,
		ride.SurgeMultiplier,
		ride.RequestedAt,
		ride.RideTypeID,
		ride.PromoCodeID,
		ride.DiscountAmount,
		ride.ScheduledAt,
		ride.IsScheduled,
		ride.ScheduledNotificationSent,
	).Scan(&ride.CreatedAt, &ride.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create ride: %w", err)
	}

	return nil
}

// GetRideByID retrieves a ride by ID
func (r *Repository) GetRideByID(ctx context.Context, id uuid.UUID) (*models.Ride, error) {
	query := `
		SELECT id, rider_id, driver_id, status, pickup_latitude, pickup_longitude,
			   pickup_address, dropoff_latitude, dropoff_longitude, dropoff_address,
			   estimated_distance, estimated_duration, estimated_fare, actual_distance,
			   actual_duration, final_fare, surge_multiplier, requested_at, accepted_at,
			   started_at, completed_at, cancelled_at, cancellation_reason, rating,
			   feedback, created_at, updated_at, ride_type_id, promo_code_id,
			   discount_amount, scheduled_at, is_scheduled, scheduled_notification_sent
		FROM rides
		WHERE id = $1
	`

	ride := &models.Ride{}
	err := r.db.QueryRow(ctx, query, id).Scan(
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
		&ride.RideTypeID,
		&ride.PromoCodeID,
		&ride.DiscountAmount,
		&ride.ScheduledAt,
		&ride.IsScheduled,
		&ride.ScheduledNotificationSent,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get ride: %w", err)
	}

	return ride, nil
}

// UpdateRideStatus updates ride status and related fields.
func (r *Repository) UpdateRideStatus(ctx context.Context, id uuid.UUID, status models.RideStatus, driverID *uuid.UUID) error {
	now := time.Now()
	var query string
	var args []interface{}

	switch status {
	case models.RideStatusAccepted:
		query = `UPDATE rides SET status = $1, driver_id = $2, accepted_at = $3, updated_at = $4 WHERE id = $5`
		args = []interface{}{status, driverID, now, now, id}
	case models.RideStatusInProgress:
		query = `UPDATE rides SET status = $1, started_at = $2, updated_at = $3 WHERE id = $4`
		args = []interface{}{status, now, now, id}
	case models.RideStatusCompleted:
		query = `UPDATE rides SET status = $1, completed_at = $2, updated_at = $3 WHERE id = $4`
		args = []interface{}{status, now, now, id}
	case models.RideStatusCancelled:
		query = `UPDATE rides SET status = $1, cancelled_at = $2, updated_at = $3 WHERE id = $4`
		args = []interface{}{status, now, now, id}
	default:
		query = `UPDATE rides SET status = $1, updated_at = $2 WHERE id = $3`
		args = []interface{}{status, now, id}
	}

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update ride status: %w", err)
	}

	return nil
}

// AtomicAcceptRide atomically transitions a ride from "requested" to "accepted"
// in a single UPDATE with a WHERE status guard. Returns false if the ride was
// already accepted by another driver (prevents double-accept race condition).
func (r *Repository) AtomicAcceptRide(ctx context.Context, rideID, driverID uuid.UUID) (bool, error) {
	now := time.Now()
	query := `
		UPDATE rides
		SET status = $1, driver_id = $2, accepted_at = $3, updated_at = $3
		WHERE id = $4 AND status = $5
	`
	tag, err := r.db.Exec(ctx, query,
		models.RideStatusAccepted, driverID, now, rideID, models.RideStatusRequested,
	)
	if err != nil {
		return false, fmt.Errorf("failed to accept ride: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// UpdateRideCompletion updates ride with actual data upon completion
func (r *Repository) UpdateRideCompletion(ctx context.Context, id uuid.UUID, actualDistance float64, actualDuration int, finalFare float64) error {
	query := `
		UPDATE rides
		SET actual_distance = $1, actual_duration = $2, final_fare = $3, updated_at = $4
		WHERE id = $5
	`

	_, err := r.db.Exec(ctx, query, actualDistance, actualDuration, finalFare, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update ride completion: %w", err)
	}

	return nil
}

// UpdateRideRating updates ride rating and feedback
func (r *Repository) UpdateRideRating(ctx context.Context, id uuid.UUID, rating int, feedback *string) error {
	query := `
		UPDATE rides
		SET rating = $1, feedback = $2, updated_at = $3
		WHERE id = $4
	`

	_, err := r.db.Exec(ctx, query, rating, feedback, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update ride rating: %w", err)
	}

	return nil
}

// GetRidesByRider retrieves rides for a specific rider
func (r *Repository) GetRidesByRider(ctx context.Context, riderID uuid.UUID, limit, offset int) ([]*models.Ride, error) {
	query := `
		SELECT id, rider_id, driver_id, status, pickup_latitude, pickup_longitude,
			   pickup_address, dropoff_latitude, dropoff_longitude, dropoff_address,
			   estimated_distance, estimated_duration, estimated_fare, actual_distance,
			   actual_duration, final_fare, surge_multiplier, requested_at, accepted_at,
			   started_at, completed_at, cancelled_at, cancellation_reason, rating,
			   feedback, created_at, updated_at, ride_type_id, promo_code_id,
			   discount_amount, scheduled_at, is_scheduled, scheduled_notification_sent
		FROM rides
		WHERE rider_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, riderID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get rides: %w", err)
	}
	defer rows.Close()

	rides := make([]*models.Ride, 0)
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
			&ride.RideTypeID,
			&ride.PromoCodeID,
			&ride.DiscountAmount,
			&ride.ScheduledAt,
			&ride.IsScheduled,
			&ride.ScheduledNotificationSent,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ride: %w", err)
		}
		rides = append(rides, ride)
	}

	return rides, nil
}

// GetRidesByDriver retrieves rides for a specific driver
func (r *Repository) GetRidesByDriver(ctx context.Context, driverID uuid.UUID, limit, offset int) ([]*models.Ride, error) {
	query := `
		SELECT id, rider_id, driver_id, status, pickup_latitude, pickup_longitude,
			   pickup_address, dropoff_latitude, dropoff_longitude, dropoff_address,
			   estimated_distance, estimated_duration, estimated_fare, actual_distance,
			   actual_duration, final_fare, surge_multiplier, requested_at, accepted_at,
			   started_at, completed_at, cancelled_at, cancellation_reason, rating,
			   feedback, created_at, updated_at, ride_type_id, promo_code_id,
			   discount_amount, scheduled_at, is_scheduled, scheduled_notification_sent
		FROM rides
		WHERE driver_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, driverID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get rides: %w", err)
	}
	defer rows.Close()

	rides := make([]*models.Ride, 0)
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
			&ride.RideTypeID,
			&ride.PromoCodeID,
			&ride.DiscountAmount,
			&ride.ScheduledAt,
			&ride.IsScheduled,
			&ride.ScheduledNotificationSent,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ride: %w", err)
		}
		rides = append(rides, ride)
	}

	return rides, nil
}

// GetRidesByRiderWithTotal retrieves rides for a specific rider with total count
func (r *Repository) GetRidesByRiderWithTotal(ctx context.Context, riderID uuid.UUID, limit, offset int) ([]*models.Ride, int64, error) {
	// Get total count
	var total int64
	countQuery := `SELECT COUNT(*) FROM rides WHERE rider_id = $1`
	err := r.db.QueryRow(ctx, countQuery, riderID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count rides: %w", err)
	}

	// Get paginated rides
	rides, err := r.GetRidesByRider(ctx, riderID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return rides, total, nil
}

// GetRidesByDriverWithTotal retrieves rides for a specific driver with total count
func (r *Repository) GetRidesByDriverWithTotal(ctx context.Context, driverID uuid.UUID, limit, offset int) ([]*models.Ride, int64, error) {
	// Get total count
	var total int64
	countQuery := `SELECT COUNT(*) FROM rides WHERE driver_id = $1`
	err := r.db.QueryRow(ctx, countQuery, driverID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count rides: %w", err)
	}

	// Get paginated rides
	rides, err := r.GetRidesByDriver(ctx, driverID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return rides, total, nil
}

// GetPendingRides retrieves all pending ride requests
func (r *Repository) GetPendingRides(ctx context.Context) ([]*models.Ride, error) {
	query := `
		SELECT id, rider_id, driver_id, status, pickup_latitude, pickup_longitude,
			   pickup_address, dropoff_latitude, dropoff_longitude, dropoff_address,
			   estimated_distance, estimated_duration, estimated_fare, actual_distance,
			   actual_duration, final_fare, surge_multiplier, requested_at, accepted_at,
			   started_at, completed_at, cancelled_at, cancellation_reason, rating,
			   feedback, created_at, updated_at, ride_type_id, promo_code_id,
			   discount_amount, scheduled_at, is_scheduled, scheduled_notification_sent
		FROM rides
		WHERE status = 'requested'
		ORDER BY requested_at ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending rides: %w", err)
	}
	defer rows.Close()

	rides := make([]*models.Ride, 0)
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
			&ride.RideTypeID,
			&ride.PromoCodeID,
			&ride.DiscountAmount,
			&ride.ScheduledAt,
			&ride.IsScheduled,
			&ride.ScheduledNotificationSent,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ride: %w", err)
		}
		rides = append(rides, ride)
	}

	return rides, nil
}

// RideFilters represents filters for ride history queries
type RideFilters struct {
	Status    *string
	StartDate *time.Time
	EndDate   *time.Time
}

// GetRidesByRiderWithFilters retrieves filtered rides for a rider
func (r *Repository) GetRidesByRiderWithFilters(ctx context.Context, riderID uuid.UUID, filters *RideFilters, limit, offset int) ([]*models.Ride, int, error) {
	// Build dynamic query
	baseQuery := `
		SELECT id, rider_id, driver_id, status, pickup_latitude, pickup_longitude,
			   pickup_address, dropoff_latitude, dropoff_longitude, dropoff_address,
			   estimated_distance, estimated_duration, estimated_fare, actual_distance,
			   actual_duration, final_fare, surge_multiplier, requested_at, accepted_at,
			   started_at, completed_at, cancelled_at, cancellation_reason, rating,
			   feedback, created_at, updated_at, ride_type_id, promo_code_id,
			   discount_amount, scheduled_at, is_scheduled, scheduled_notification_sent
		FROM rides
		WHERE rider_id = $1
	`

	countQuery := `SELECT COUNT(*) FROM rides WHERE rider_id = $1`

	args := []interface{}{riderID}
	argCount := 2

	// Apply filters
	if filters.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filters.Status)
		argCount++
	}

	if filters.StartDate != nil {
		baseQuery += fmt.Sprintf(" AND created_at >= $%d", argCount)
		countQuery += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, *filters.StartDate)
		argCount++
	}

	if filters.EndDate != nil {
		baseQuery += fmt.Sprintf(" AND created_at <= $%d", argCount)
		countQuery += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, *filters.EndDate)
		argCount++
	}

	// Add ordering and pagination
	baseQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	// Get total count
	var total int
	err := r.db.QueryRow(ctx, countQuery, args[:argCount-2]...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get rides
	rows, err := r.db.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get filtered rides: %w", err)
	}
	defer rows.Close()

	rides := make([]*models.Ride, 0)
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
			&ride.RideTypeID,
			&ride.PromoCodeID,
			&ride.DiscountAmount,
			&ride.ScheduledAt,
			&ride.IsScheduled,
			&ride.ScheduledNotificationSent,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan ride: %w", err)
		}
		rides = append(rides, ride)
	}

	return rides, total, nil
}

// GetUserProfile retrieves a user's profile information
func (r *Repository) GetUserProfile(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, email, phone_number, first_name, last_name, role,
			   is_active, is_verified, profile_image, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &models.User{}
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.PhoneNumber,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.IsActive,
		&user.IsVerified,
		&user.ProfileImage,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	return user, nil
}

// UpdateUserProfile updates a user's profile information
func (r *Repository) UpdateUserProfile(ctx context.Context, userID uuid.UUID, firstName, lastName, phoneNumber string) error {
	query := `
		UPDATE users
		SET first_name = $1, last_name = $2, phone_number = $3, updated_at = NOW()
		WHERE id = $4
	`

	result, err := r.db.Exec(ctx, query, firstName, lastName, phoneNumber, userID)
	if err != nil {
		return fmt.Errorf("failed to update user profile: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// DriverMatchStats holds aggregated stats used by the matching algorithm.
type DriverMatchStats struct {
	DriverID       uuid.UUID
	Rating         float64
	AcceptanceRate float64
	IdleMinutes    float64
}

// GetDriverMatchStats returns acceptance rate and idle time for a list of driver IDs.
func (r *Repository) GetDriverMatchStats(ctx context.Context, driverIDs []uuid.UUID) (map[uuid.UUID]*DriverMatchStats, error) {
	if len(driverIDs) == 0 {
		return make(map[uuid.UUID]*DriverMatchStats), nil
	}

	query := `
		SELECT
			u.id,
			COALESCE(u.rating, 4.0) AS rating,
			COALESCE(
				CAST(SUM(CASE WHEN r.status IN ('accepted','started','completed') THEN 1 ELSE 0 END) AS FLOAT) /
				NULLIF(COUNT(r.id), 0),
				0.8
			) AS acceptance_rate,
			COALESCE(
				EXTRACT(EPOCH FROM (NOW() - MAX(r.completed_at))) / 60.0,
				30.0
			) AS idle_minutes
		FROM users u
		LEFT JOIN rides r ON r.driver_id = u.id AND r.created_at > NOW() - INTERVAL '30 days'
		WHERE u.id = ANY($1)
		GROUP BY u.id, u.rating
	`

	rows, err := r.db.Query(ctx, query, driverIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver match stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[uuid.UUID]*DriverMatchStats, len(driverIDs))
	for rows.Next() {
		s := &DriverMatchStats{}
		if err := rows.Scan(&s.DriverID, &s.Rating, &s.AcceptanceRate, &s.IdleMinutes); err != nil {
			return nil, fmt.Errorf("failed to scan driver stats: %w", err)
		}
		stats[s.DriverID] = s
	}

	// Fill in defaults for drivers not in DB (new drivers)
	for _, id := range driverIDs {
		if _, ok := stats[id]; !ok {
			stats[id] = &DriverMatchStats{
				DriverID:       id,
				Rating:         4.0,
				AcceptanceRate: 0.8,
				IdleMinutes:    30.0,
			}
		}
	}

	return stats, nil
}

// GetPaymentByRideID retrieves payment information for a ride
func (r *Repository) GetPaymentByRideID(ctx context.Context, rideID uuid.UUID) (string, error) {
	query := `SELECT method FROM payments WHERE ride_id = $1 LIMIT 1`

	var method string
	err := r.db.QueryRow(ctx, query, rideID).Scan(&method)
	if err != nil {
		return "", fmt.Errorf("failed to get payment method: %w", err)
	}

	return method, nil
}
