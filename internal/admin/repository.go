package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/models"
	"go.uber.org/zap"
)

// Repository handles database operations for admin functions
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new admin repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// AuditLog represents an admin action audit record.
// Required table:
//
//	CREATE TABLE IF NOT EXISTS audit_logs (
//	    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
//	    admin_id UUID NOT NULL REFERENCES users(id),
//	    action TEXT NOT NULL,
//	    target_type TEXT NOT NULL,
//	    target_id UUID NOT NULL,
//	    metadata JSONB DEFAULT '{}',
//	    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
//	);
//	CREATE INDEX idx_audit_logs_admin ON audit_logs(admin_id);
//	CREATE INDEX idx_audit_logs_target ON audit_logs(target_type, target_id);
type AuditLog struct {
	ID         uuid.UUID              `json:"id"`
	AdminID    uuid.UUID              `json:"admin_id"`
	Action     string                 `json:"action"`
	TargetType string                 `json:"target_type"`
	TargetID   uuid.UUID              `json:"target_id"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time              `json:"created_at"`
}

// AuditLogFilter contains filter parameters for audit log queries
type AuditLogFilter struct {
	AdminID    *uuid.UUID
	Action     string
	TargetType string
	TargetID   *uuid.UUID
}

// GetAuditLogs retrieves audit logs with pagination and filters
func (r *Repository) GetAuditLogs(ctx context.Context, limit, offset int, filter *AuditLogFilter) ([]*AuditLog, int64, error) {
	whereClause := "WHERE 1=1"
	args := make([]interface{}, 0)
	argIndex := 1

	if filter != nil {
		if filter.AdminID != nil {
			whereClause += fmt.Sprintf(" AND al.admin_id = $%d", argIndex)
			args = append(args, *filter.AdminID)
			argIndex++
		}
		if filter.Action != "" {
			whereClause += fmt.Sprintf(" AND al.action = $%d", argIndex)
			args = append(args, filter.Action)
			argIndex++
		}
		if filter.TargetType != "" {
			whereClause += fmt.Sprintf(" AND al.target_type = $%d", argIndex)
			args = append(args, filter.TargetType)
			argIndex++
		}
		if filter.TargetID != nil {
			whereClause += fmt.Sprintf(" AND al.target_id = $%d", argIndex)
			args = append(args, *filter.TargetID)
			argIndex++
		}
	}

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs al %s", whereClause)
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT al.id, al.admin_id, al.action, al.target_type, al.target_id, al.metadata, al.created_at,
		       COALESCE(u.first_name || ' ' || u.last_name, '') as admin_name
		FROM audit_logs al
		LEFT JOIN users u ON al.admin_id = u.id
		%s
		ORDER BY al.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get audit logs: %w", err)
	}
	defer rows.Close()

	logs := make([]*AuditLog, 0)
	for rows.Next() {
		log := &AuditLog{}
		var adminName string
		err := rows.Scan(
			&log.ID, &log.AdminID, &log.Action, &log.TargetType,
			&log.TargetID, &log.Metadata, &log.CreatedAt, &adminName,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}
		if log.Metadata == nil {
			log.Metadata = map[string]interface{}{}
		}
		log.Metadata["admin_name"] = adminName
		logs = append(logs, log)
	}

	return logs, total, nil
}

// InsertAuditLog records an admin action in the audit log.
// Failures are logged but do not block the calling operation.
func (r *Repository) InsertAuditLog(ctx context.Context, adminID uuid.UUID, action, targetType string, targetID uuid.UUID, metadata map[string]interface{}) {
	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	query := `
		INSERT INTO audit_logs (id, admin_id, action, target_type, target_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(ctx, query, uuid.New(), adminID, action, targetType, targetID, metadata)
	if err != nil {
		// Audit log failure should not block the operation
		logger.Warn("failed to insert audit log", zap.Error(err))
	}
}

// UserFilter contains filter parameters for user queries
type UserFilter struct {
	Role     string // "admin", "driver", "rider", or empty for all
	IsActive *bool  // true, false, or nil for all
	Search   string // search by email, name, or phone
}

// GetAllUsers retrieves all users with pagination and filters
func (r *Repository) GetAllUsers(ctx context.Context, limit, offset int, filter *UserFilter) ([]*models.User, int, error) {
	// Build WHERE clause based on filters
	whereClause := "WHERE 1=1"
	args := make([]interface{}, 0)
	argIndex := 1

	if filter != nil {
		if filter.Role != "" {
			whereClause += fmt.Sprintf(" AND role = $%d", argIndex)
			args = append(args, filter.Role)
			argIndex++
		}
		if filter.IsActive != nil {
			whereClause += fmt.Sprintf(" AND is_active = $%d", argIndex)
			args = append(args, *filter.IsActive)
			argIndex++
		}
		if filter.Search != "" {
			// Search by email, first_name, last_name, or phone_number (case-insensitive)
			whereClause += fmt.Sprintf(" AND (email ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d OR phone_number ILIKE $%d OR CONCAT(first_name, ' ', last_name) ILIKE $%d)", argIndex, argIndex, argIndex, argIndex, argIndex)
			args = append(args, "%"+filter.Search+"%")
			argIndex++
		}
	}

	// Get total count with filters
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", whereClause)
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user count: %w", err)
	}

	// Get users with filters
	query := fmt.Sprintf(`
		SELECT id, email, phone_number, first_name, last_name, role, is_active, created_at, updated_at
		FROM users
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get users: %w", err)
	}
	defer rows.Close()

	users := make([]*models.User, 0)
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

// DriverFilter contains filter parameters for driver queries
type DriverFilter struct {
	Status string // "online", "offline", "available", "pending", or empty for all
	Search string // search by license, name, vehicle model, vehicle plate
}

// DriverStats represents driver statistics for stats cards
type DriverStats struct {
	TotalDrivers     int `json:"total_drivers"`
	OnlineDrivers    int `json:"online_drivers"`
	OfflineDrivers   int `json:"offline_drivers"`
	AvailableDrivers int `json:"available_drivers"`
	PendingApprovals int `json:"pending_approvals"`
}

// GetDriverStats retrieves driver statistics
func (r *Repository) GetDriverStats(ctx context.Context) (*DriverStats, error) {
	query := `
		SELECT
			COUNT(*) as total_drivers,
			COUNT(CASE WHEN is_online = true AND approval_status = 'approved' THEN 1 END) as online_drivers,
			COUNT(CASE WHEN is_online = false AND approval_status = 'approved' THEN 1 END) as offline_drivers,
			COUNT(CASE WHEN is_available = true AND is_online = true AND approval_status = 'approved' THEN 1 END) as available_drivers,
			COUNT(CASE WHEN approval_status = 'pending' THEN 1 END) as pending_approvals
		FROM drivers
	`

	stats := &DriverStats{}
	err := r.db.QueryRow(ctx, query).Scan(
		&stats.TotalDrivers,
		&stats.OnlineDrivers,
		&stats.OfflineDrivers,
		&stats.AvailableDrivers,
		&stats.PendingApprovals,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get driver stats: %w", err)
	}

	return stats, nil
}

// GetAllDriversWithTotal retrieves all drivers with pagination, filters, and total count
func (r *Repository) GetAllDriversWithTotal(ctx context.Context, limit, offset int, filter *DriverFilter) ([]*models.Driver, int64, error) {
	// Build WHERE clause based on filters
	whereClause := "WHERE 1=1"
	args := make([]interface{}, 0)
	argIndex := 1

	if filter != nil {
		// Status filter
		switch filter.Status {
		case "online":
			whereClause += " AND d.is_online = true AND d.approval_status = 'approved'"
		case "offline":
			whereClause += " AND d.is_online = false AND d.approval_status = 'approved'"
		case "available":
			whereClause += " AND d.is_available = true AND d.is_online = true AND d.approval_status = 'approved'"
		case "pending":
			whereClause += " AND d.approval_status = 'pending'"
		}

		// Search filter
		if filter.Search != "" {
			whereClause += fmt.Sprintf(` AND (
				d.license_number ILIKE $%d OR
				d.vehicle_model ILIKE $%d OR
				d.vehicle_plate ILIKE $%d OR
				d.vehicle_color ILIKE $%d OR
				u.first_name ILIKE $%d OR
				u.last_name ILIKE $%d OR
				u.phone_number ILIKE $%d OR
				CONCAT(u.first_name, ' ', u.last_name) ILIKE $%d
			)`, argIndex, argIndex, argIndex, argIndex, argIndex, argIndex, argIndex, argIndex)
			args = append(args, "%"+filter.Search+"%")
			argIndex++
		}
	}

	// Get total count with filters
	var total int64
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM drivers d
		JOIN users u ON d.user_id = u.id
		%s
	`, whereClause)
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count drivers: %w", err)
	}

	// Get paginated drivers with filters
	query := fmt.Sprintf(`
		SELECT d.id, d.user_id, d.license_number, d.vehicle_model, d.vehicle_plate,
		       d.vehicle_color, d.vehicle_year, d.is_available, d.is_online,
		       d.approval_status, d.approved_by, d.approved_at, d.rejection_reason, d.rejected_at,
		       d.rating, d.total_rides,
		       d.current_latitude, d.current_longitude, d.last_location_update,
		       d.created_at, d.updated_at
		FROM drivers d
		JOIN users u ON d.user_id = u.id
		%s
		ORDER BY d.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get drivers: %w", err)
	}
	defer rows.Close()

	drivers := make([]*models.Driver, 0)
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
			&driver.ApprovalStatus,
			&driver.ApprovedBy,
			&driver.ApprovedAt,
			&driver.RejectionReason,
			&driver.RejectedAt,
			&driver.Rating,
			&driver.TotalRides,
			&driver.CurrentLatitude,
			&driver.CurrentLongitude,
			&driver.LastLocationUpdate,
			&driver.CreatedAt,
			&driver.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan driver: %w", err)
		}

		drivers = append(drivers, driver)
	}

	return drivers, total, nil
}

// GetPendingDrivers retrieves pending (not yet approved) drivers
func (r *Repository) GetPendingDrivers(ctx context.Context) ([]*models.Driver, error) {
	query := `
		SELECT d.id, d.user_id, d.license_number, d.vehicle_model, d.vehicle_plate,
		       d.vehicle_color, d.vehicle_year, d.is_available, d.is_online,
		       d.approval_status, d.approved_by, d.approved_at, d.rejection_reason, d.rejected_at,
		       d.rating, d.total_rides, d.created_at, d.updated_at
		FROM drivers d
		WHERE d.approval_status = 'pending'
		ORDER BY d.created_at DESC
		LIMIT 100
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get drivers: %w", err)
	}
	defer rows.Close()

	drivers := make([]*models.Driver, 0)
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
			&driver.ApprovalStatus,
			&driver.ApprovedBy,
			&driver.ApprovedAt,
			&driver.RejectionReason,
			&driver.RejectedAt,
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

// GetPendingDriversWithTotal retrieves pending (not yet approved) drivers with pagination and total count
func (r *Repository) GetPendingDriversWithTotal(ctx context.Context, limit, offset int) ([]*models.Driver, int64, error) {
	// Get total count of pending drivers
	var total int64
	countQuery := `SELECT COUNT(*) FROM drivers WHERE approval_status = 'pending'`
	err := r.db.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count pending drivers: %w", err)
	}

	// Get paginated pending drivers
	query := `
		SELECT d.id, d.user_id, d.license_number, d.vehicle_model, d.vehicle_plate,
		       d.vehicle_color, d.vehicle_year, d.is_available, d.is_online,
		       d.approval_status, d.approved_by, d.approved_at, d.rejection_reason, d.rejected_at,
		       d.rating, d.total_rides,
		       d.current_latitude, d.current_longitude, d.last_location_update,
		       d.created_at, d.updated_at
		FROM drivers d
		WHERE d.approval_status = 'pending'
		ORDER BY d.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get drivers: %w", err)
	}
	defer rows.Close()

	drivers := make([]*models.Driver, 0)
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
			&driver.ApprovalStatus,
			&driver.ApprovedBy,
			&driver.ApprovedAt,
			&driver.RejectionReason,
			&driver.RejectedAt,
			&driver.Rating,
			&driver.TotalRides,
			&driver.CurrentLatitude,
			&driver.CurrentLongitude,
			&driver.LastLocationUpdate,
			&driver.CreatedAt,
			&driver.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan driver: %w", err)
		}

		drivers = append(drivers, driver)
	}

	return drivers, total, nil
}

// GetDriverByID retrieves a driver by ID with user details
func (r *Repository) GetDriverByID(ctx context.Context, driverID uuid.UUID) (*models.Driver, error) {
	query := `
		SELECT d.id, d.user_id, d.license_number, d.vehicle_model, d.vehicle_plate,
		       d.vehicle_color, d.vehicle_year, d.is_available, d.is_online,
		       d.approval_status, d.approved_by, d.approved_at, d.rejection_reason, d.rejected_at,
		       d.rating, d.total_rides,
		       d.current_latitude, d.current_longitude, d.last_location_update,
		       d.created_at, d.updated_at
		FROM drivers d
		WHERE d.id = $1
	`

	driver := &models.Driver{}
	err := r.db.QueryRow(ctx, query, driverID).Scan(
		&driver.ID,
		&driver.UserID,
		&driver.LicenseNumber,
		&driver.VehicleModel,
		&driver.VehiclePlate,
		&driver.VehicleColor,
		&driver.VehicleYear,
		&driver.IsAvailable,
		&driver.IsOnline,
		&driver.ApprovalStatus,
		&driver.ApprovedBy,
		&driver.ApprovedAt,
		&driver.RejectionReason,
		&driver.RejectedAt,
		&driver.Rating,
		&driver.TotalRides,
		&driver.CurrentLatitude,
		&driver.CurrentLongitude,
		&driver.LastLocationUpdate,
		&driver.CreatedAt,
		&driver.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get driver: %w", err)
	}

	return driver, nil
}

// ApproveDriver sets a driver as available (simplified approval)
func (r *Repository) ApproveDriver(ctx context.Context, driverID uuid.UUID) error {
	query := `
		UPDATE drivers
		SET approval_status = 'approved',
		    approved_at = $1,
		    updated_at = $1
		WHERE id = $2
	`

	result, err := r.db.Exec(ctx, query, time.Now(), driverID)
	if err != nil {
		return fmt.Errorf("failed to approve driver: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("driver not found")
	}

	return nil
}

// RejectDriver sets a driver as rejected with optional reason
func (r *Repository) RejectDriver(ctx context.Context, driverID uuid.UUID) error {
	query := `
		UPDATE drivers
		SET approval_status = 'rejected',
		    rejected_at = $1,
		    updated_at = $1
		WHERE id = $2
	`

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
			COUNT(CASE WHEN role = 'admin' THEN 1 END) as total_admins,
			COUNT(CASE WHEN is_active = true THEN 1 END) as active_users
		FROM users
	`

	stats := &UserStats{}
	err := r.db.QueryRow(ctx, query).Scan(
		&stats.TotalUsers,
		&stats.TotalRiders,
		&stats.TotalDrivers,
		&stats.TotalAdmins,
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
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ride: %w", err)
		}
		rides = append(rides, ride)
	}

	return rides, nil
}

// GetRecentRidesWithTotal retrieves recent rides for monitoring with total count
func (r *Repository) GetRecentRidesWithTotal(ctx context.Context, limit, offset int) ([]*models.Ride, int64, error) {
	// Get total count
	var total int64
	countQuery := `SELECT COUNT(*) FROM rides`
	err := r.db.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count rides: %w", err)
	}

	// Get paginated rides
	query := `
		SELECT id, rider_id, driver_id, status, pickup_latitude, pickup_longitude,
		       pickup_address, dropoff_latitude, dropoff_longitude, dropoff_address,
		       estimated_distance, estimated_duration, estimated_fare, actual_distance,
		       actual_duration, final_fare, surge_multiplier, requested_at, accepted_at,
		       started_at, completed_at, cancelled_at, cancellation_reason, rating,
		       feedback, created_at, updated_at
		FROM rides
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get recent rides: %w", err)
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
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan ride: %w", err)
		}
		rides = append(rides, ride)
	}

	return rides, total, nil
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
	TotalAdmins  int `json:"total_admins"`
	ActiveUsers  int `json:"active_users"`
}

// AdminRideDetail represents a ride with rider and driver details for admin view
type AdminRideDetail struct {
	*models.Ride
	Rider  *RideUserInfo   `json:"rider,omitempty"`
	Driver *RideDriverInfo `json:"driver,omitempty"`
}

// RideUserInfo represents basic user info for ride details
type RideUserInfo struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	PhoneNumber string    `json:"phone_number"`
}

// RideDriverInfo represents driver info for ride details
type RideDriverInfo struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	PhoneNumber  string    `json:"phone_number"`
	VehicleModel string    `json:"vehicle_model"`
	VehiclePlate string    `json:"vehicle_plate"`
	VehicleColor string    `json:"vehicle_color"`
	Rating       float64   `json:"rating"`
}

// GetRideByID retrieves a ride by ID with rider and driver details
func (r *Repository) GetRideByID(ctx context.Context, rideID uuid.UUID) (*AdminRideDetail, error) {
	query := `
		SELECT r.id, r.rider_id, r.driver_id, r.status, r.pickup_latitude, r.pickup_longitude,
		       r.pickup_address, r.dropoff_latitude, r.dropoff_longitude, r.dropoff_address,
		       r.estimated_distance, r.estimated_duration, r.estimated_fare, r.actual_distance,
		       r.actual_duration, r.final_fare, r.surge_multiplier, r.requested_at, r.accepted_at,
		       r.started_at, r.completed_at, r.cancelled_at, r.cancellation_reason, r.rating,
		       r.feedback, r.created_at, r.updated_at,
		       ru.id, ru.email, ru.first_name, ru.last_name, ru.phone_number,
		       d.id, d.user_id, du.first_name, du.last_name, du.phone_number,
		       d.vehicle_model, d.vehicle_plate, d.vehicle_color, d.rating
		FROM rides r
		JOIN users ru ON r.rider_id = ru.id
		LEFT JOIN drivers d ON r.driver_id = d.user_id
		LEFT JOIN users du ON r.driver_id = du.id
		WHERE r.id = $1
	`

	ride := &models.Ride{}
	riderInfo := &RideUserInfo{}
	var driverID, driverUserID *uuid.UUID
	var driverFirstName, driverLastName, driverPhone, vehicleModel, vehiclePlate, vehicleColor *string
	var driverRating *float64

	err := r.db.QueryRow(ctx, query, rideID).Scan(
		&ride.ID, &ride.RiderID, &ride.DriverID, &ride.Status,
		&ride.PickupLatitude, &ride.PickupLongitude, &ride.PickupAddress,
		&ride.DropoffLatitude, &ride.DropoffLongitude, &ride.DropoffAddress,
		&ride.EstimatedDistance, &ride.EstimatedDuration, &ride.EstimatedFare,
		&ride.ActualDistance, &ride.ActualDuration, &ride.FinalFare,
		&ride.SurgeMultiplier, &ride.RequestedAt, &ride.AcceptedAt,
		&ride.StartedAt, &ride.CompletedAt, &ride.CancelledAt, &ride.CancellationReason,
		&ride.Rating, &ride.Feedback, &ride.CreatedAt, &ride.UpdatedAt,
		&riderInfo.ID, &riderInfo.Email, &riderInfo.FirstName, &riderInfo.LastName, &riderInfo.PhoneNumber,
		&driverID, &driverUserID, &driverFirstName, &driverLastName, &driverPhone,
		&vehicleModel, &vehiclePlate, &vehicleColor, &driverRating,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride: %w", err)
	}

	result := &AdminRideDetail{
		Ride:  ride,
		Rider: riderInfo,
	}

	// Add driver info if present
	if driverID != nil {
		result.Driver = &RideDriverInfo{
			ID:           *driverID,
			UserID:       *driverUserID,
			FirstName:    stringValue(driverFirstName),
			LastName:     stringValue(driverLastName),
			PhoneNumber:  stringValue(driverPhone),
			VehicleModel: stringValue(vehicleModel),
			VehiclePlate: stringValue(vehiclePlate),
			VehicleColor: stringValue(vehicleColor),
			Rating:       floatValue(driverRating),
		}
	}

	return result, nil
}

// GetRecentRidesWithDetails retrieves recent rides with rider and driver info
func (r *Repository) GetRecentRidesWithDetails(ctx context.Context, limit, offset int) ([]*AdminRideDetail, int64, error) {
	// Get total count
	var total int64
	countQuery := `SELECT COUNT(*) FROM rides`
	err := r.db.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count rides: %w", err)
	}

	// Get paginated rides with user details
	query := `
		SELECT r.id, r.rider_id, r.driver_id, r.status, r.pickup_latitude, r.pickup_longitude,
		       r.pickup_address, r.dropoff_latitude, r.dropoff_longitude, r.dropoff_address,
		       r.estimated_distance, r.estimated_duration, r.estimated_fare, r.actual_distance,
		       r.actual_duration, r.final_fare, r.surge_multiplier, r.requested_at, r.accepted_at,
		       r.started_at, r.completed_at, r.cancelled_at, r.cancellation_reason, r.rating,
		       r.feedback, r.created_at, r.updated_at,
		       ru.id, ru.email, ru.first_name, ru.last_name, ru.phone_number,
		       d.id, d.user_id, du.first_name, du.last_name, du.phone_number,
		       d.vehicle_model, d.vehicle_plate, d.vehicle_color, d.rating
		FROM rides r
		JOIN users ru ON r.rider_id = ru.id
		LEFT JOIN drivers d ON r.driver_id = d.user_id
		LEFT JOIN users du ON r.driver_id = du.id
		ORDER BY r.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get recent rides: %w", err)
	}
	defer rows.Close()

	rides := make([]*AdminRideDetail, 0)
	for rows.Next() {
		ride := &models.Ride{}
		riderInfo := &RideUserInfo{}
		var driverID, driverUserID *uuid.UUID
		var driverFirstName, driverLastName, driverPhone, vehicleModel, vehiclePlate, vehicleColor *string
		var driverRating *float64

		err := rows.Scan(
			&ride.ID, &ride.RiderID, &ride.DriverID, &ride.Status,
			&ride.PickupLatitude, &ride.PickupLongitude, &ride.PickupAddress,
			&ride.DropoffLatitude, &ride.DropoffLongitude, &ride.DropoffAddress,
			&ride.EstimatedDistance, &ride.EstimatedDuration, &ride.EstimatedFare,
			&ride.ActualDistance, &ride.ActualDuration, &ride.FinalFare,
			&ride.SurgeMultiplier, &ride.RequestedAt, &ride.AcceptedAt,
			&ride.StartedAt, &ride.CompletedAt, &ride.CancelledAt, &ride.CancellationReason,
			&ride.Rating, &ride.Feedback, &ride.CreatedAt, &ride.UpdatedAt,
			&riderInfo.ID, &riderInfo.Email, &riderInfo.FirstName, &riderInfo.LastName, &riderInfo.PhoneNumber,
			&driverID, &driverUserID, &driverFirstName, &driverLastName, &driverPhone,
			&vehicleModel, &vehiclePlate, &vehicleColor, &driverRating,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan ride: %w", err)
		}

		detail := &AdminRideDetail{
			Ride:  ride,
			Rider: riderInfo,
		}

		// Add driver info if present
		if driverID != nil {
			detail.Driver = &RideDriverInfo{
				ID:           *driverID,
				UserID:       *driverUserID,
				FirstName:    stringValue(driverFirstName),
				LastName:     stringValue(driverLastName),
				PhoneNumber:  stringValue(driverPhone),
				VehicleModel: stringValue(vehicleModel),
				VehiclePlate: stringValue(vehiclePlate),
				VehicleColor: stringValue(vehicleColor),
				Rating:       floatValue(driverRating),
			}
		}

		rides = append(rides, detail)
	}

	return rides, total, nil
}

// GetActivityFeed retrieves recent activity events from rides, drivers, and users
func (r *Repository) GetActivityFeed(ctx context.Context, limit, offset int) ([]*ActivityFeedItem, int64, error) {
	// Count total events
	countQuery := `
		SELECT (
			SELECT COUNT(*) FROM rides
		) + (
			SELECT COUNT(*) FROM drivers
		) + (
			SELECT COUNT(*) FROM users WHERE role IN ('rider', 'driver')
		)
	`
	var total int64
	if err := r.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count activity feed: %w", err)
	}

	query := `
		(
			SELECT r.id,
			       CASE r.status
			           WHEN 'requested' THEN 'ride_requested'
			           WHEN 'accepted' THEN 'ride_accepted'
			           WHEN 'in_progress' THEN 'ride_started'
			           WHEN 'completed' THEN 'ride_completed'
			           WHEN 'cancelled' THEN 'ride_cancelled'
			       END AS event_type,
			       CASE r.status
			           WHEN 'requested' THEN CONCAT(u.first_name, ' ', u.last_name, ' requested a ride from ', r.pickup_address)
			           WHEN 'accepted' THEN CONCAT(COALESCE(du.first_name, ''), ' ', COALESCE(du.last_name, ''), ' accepted a ride')
			           WHEN 'in_progress' THEN CONCAT('Ride to ', r.dropoff_address, ' started')
			           WHEN 'completed' THEN CONCAT('Ride completed — $', COALESCE(r.final_fare::text, r.estimated_fare::text))
			           WHEN 'cancelled' THEN CONCAT(u.first_name, ' ', u.last_name, ' cancelled a ride', CASE WHEN r.cancellation_reason IS NOT NULL THEN CONCAT(' — ', r.cancellation_reason) ELSE '' END)
			       END AS description,
			       CONCAT(u.first_name, ' ', u.last_name) AS actor_name,
			       u.role AS actor_role,
			       'ride' AS entity_type,
			       r.id AS entity_id,
			       '' AS metadata,
			       COALESCE(
			           CASE r.status
			               WHEN 'completed' THEN r.completed_at
			               WHEN 'cancelled' THEN r.cancelled_at
			               WHEN 'accepted' THEN r.accepted_at
			               WHEN 'in_progress' THEN r.started_at
			               ELSE r.requested_at
			           END, r.created_at
			       ) AS created_at
			FROM rides r
			JOIN users u ON r.rider_id = u.id
			LEFT JOIN users du ON r.driver_id = du.id
		)
		UNION ALL
		(
			SELECT d.id,
			       'driver_registered' AS event_type,
			       CONCAT(u.first_name, ' ', u.last_name, ' registered as a driver') AS description,
			       CONCAT(u.first_name, ' ', u.last_name) AS actor_name,
			       'driver' AS actor_role,
			       'driver' AS entity_type,
			       d.id AS entity_id,
			       '' AS metadata,
			       d.created_at
			FROM drivers d
			JOIN users u ON d.user_id = u.id
		)
		UNION ALL
		(
			SELECT u.id,
			       'user_registered' AS event_type,
			       CONCAT(u.first_name, ' ', u.last_name, ' signed up as a ', u.role) AS description,
			       CONCAT(u.first_name, ' ', u.last_name) AS actor_name,
			       u.role AS actor_role,
			       'user' AS entity_type,
			       u.id AS entity_id,
			       '' AS metadata,
			       u.created_at
			FROM users u
			WHERE u.role IN ('rider', 'driver')
		)
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get activity feed: %w", err)
	}
	defer rows.Close()

	items := make([]*ActivityFeedItem, 0)
	for rows.Next() {
		item := &ActivityFeedItem{}
		err := rows.Scan(
			&item.ID, &item.EventType, &item.Description, &item.ActorName,
			&item.ActorRole, &item.EntityType, &item.EntityID, &item.Metadata,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan activity feed item: %w", err)
		}
		items = append(items, item)
	}

	return items, total, nil
}

// Helper functions for nullable values
func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func floatValue(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

// GetRealtimeMetrics retrieves real-time dashboard metrics
func (r *Repository) GetRealtimeMetrics(ctx context.Context) (*RealtimeMetrics, error) {
	metrics := &RealtimeMetrics{}

	// Get active rides count
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM rides WHERE status = 'in_progress'`).Scan(&metrics.ActiveRides)
	if err != nil {
		return nil, fmt.Errorf("failed to get active rides: %w", err)
	}

	// Get available drivers count
	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM drivers WHERE is_available = true AND is_online = true AND approval_status = 'approved'`).Scan(&metrics.AvailableDrivers)
	if err != nil {
		return nil, fmt.Errorf("failed to get available drivers: %w", err)
	}

	// Get pending requests count
	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM rides WHERE status = 'pending'`).Scan(&metrics.PendingRequests)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending requests: %w", err)
	}

	// Get online drivers count
	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM drivers WHERE is_online = true`).Scan(&metrics.OnlineDrivers)
	if err != nil {
		return nil, fmt.Errorf("failed to get online drivers: %w", err)
	}

	// Get today's revenue
	today := time.Now().Truncate(24 * time.Hour)
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(final_fare), 0)
		FROM rides
		WHERE status = 'completed' AND created_at >= $1
	`, today).Scan(&metrics.TodayRevenue)
	if err != nil {
		return nil, fmt.Errorf("failed to get today revenue: %w", err)
	}

	// Get yesterday's revenue for comparison
	yesterday := today.Add(-24 * time.Hour)
	var yesterdayRevenue float64
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(final_fare), 0)
		FROM rides
		WHERE status = 'completed' AND created_at >= $1 AND created_at < $2
	`, yesterday, today).Scan(&yesterdayRevenue)
	if err != nil {
		return nil, fmt.Errorf("failed to get yesterday revenue: %w", err)
	}

	// Calculate revenue change percentage
	if yesterdayRevenue > 0 {
		metrics.TodayRevenueChange = ((metrics.TodayRevenue - yesterdayRevenue) / yesterdayRevenue) * 100
	}

	// Get active riders today (users who made a ride today)
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT rider_id)
		FROM rides
		WHERE created_at >= $1
	`, today).Scan(&metrics.TotalRidersActive)
	if err != nil {
		return nil, fmt.Errorf("failed to get active riders: %w", err)
	}

	// Get average wait time (time from requested_at to accepted_at) in minutes
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (accepted_at - requested_at))/60), 0)
		FROM rides
		WHERE accepted_at IS NOT NULL AND requested_at IS NOT NULL
		AND created_at >= $1
	`, today).Scan(&metrics.AvgWaitTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get avg wait time: %w", err)
	}

	// Get average ETA (estimated_duration) in minutes
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(estimated_duration), 0)
		FROM rides
		WHERE estimated_duration IS NOT NULL
		AND created_at >= $1
	`, today).Scan(&metrics.AvgETA)
	if err != nil {
		return nil, fmt.Errorf("failed to get avg eta: %w", err)
	}

	return metrics, nil
}

// GetDashboardSummary retrieves comprehensive dashboard summary
func (r *Repository) GetDashboardSummary(ctx context.Context, period string) (*DashboardSummary, error) {
	startDate, previousStart := r.getPeriodDates(period)

	summary := &DashboardSummary{
		Rides:   &SummaryRides{},
		Drivers: &SummaryDrivers{},
		Riders:  &SummaryRiders{},
		Revenue: &SummaryRevenue{},
		Alerts:  &SummaryAlerts{},
	}

	// Get ride statistics
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
			COUNT(CASE WHEN status = 'cancelled' THEN 1 END) as cancelled,
			COUNT(CASE WHEN status = 'in_progress' THEN 1 END) as in_progress,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
			COALESCE(AVG(CASE WHEN status = 'completed' AND actual_duration IS NOT NULL THEN actual_duration END), 0) as avg_duration,
			COALESCE(AVG(CASE WHEN status = 'completed' AND actual_distance IS NOT NULL THEN actual_distance END), 0) as avg_distance
		FROM rides
		WHERE created_at >= $1
	`

	err := r.db.QueryRow(ctx, query, startDate).Scan(
		&summary.Rides.Total,
		&summary.Rides.Completed,
		&summary.Rides.Cancelled,
		&summary.Rides.InProgress,
		&summary.Rides.Pending,
		&summary.Rides.AvgDuration,
		&summary.Rides.AvgDistance,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride stats: %w", err)
	}

	// Calculate rates
	if summary.Rides.Total > 0 {
		summary.Rides.CompletionRate = float64(summary.Rides.Completed) / float64(summary.Rides.Total) * 100
		summary.Rides.CancellationRate = float64(summary.Rides.Cancelled) / float64(summary.Rides.Total) * 100
	}

	// Get cancellation breakdown (simplified - using cancellation_reason field)
	r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM rides
		WHERE status = 'cancelled' AND cancellation_reason LIKE '%rider%'
		AND created_at >= $1
	`, startDate).Scan(&summary.Rides.CancelledByRider)

	r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM rides
		WHERE status = 'cancelled' AND cancellation_reason LIKE '%driver%'
		AND created_at >= $1
	`, startDate).Scan(&summary.Rides.CancelledByDriver)

	// Get previous period total for comparison
	var previousTotal int
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM rides WHERE created_at >= $1 AND created_at < $2`,
		previousStart, startDate).Scan(&previousTotal)

	if previousTotal > 0 {
		summary.Rides.ChangeVsPrevious = float64(summary.Rides.Total-previousTotal) / float64(previousTotal) * 100
	}

	// Get driver statistics
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM drivers WHERE approval_status = 'approved'`).Scan(&summary.Drivers.TotalActive)
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM drivers WHERE is_online = true AND approval_status = 'approved'`).Scan(&summary.Drivers.OnlineNow)
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM drivers WHERE is_online = true AND is_available = true AND approval_status = 'approved'`).Scan(&summary.Drivers.AvailableNow)

	summary.Drivers.BusyNow = summary.Drivers.OnlineNow - summary.Drivers.AvailableNow

	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM drivers WHERE approval_status = 'pending'`).Scan(&summary.Drivers.PendingApprovals)
	r.db.QueryRow(ctx, `SELECT COALESCE(AVG(rating), 0) FROM drivers WHERE rating > 0`).Scan(&summary.Drivers.AvgRating)

	// New driver signups in period
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM drivers WHERE created_at >= $1`, startDate).Scan(&summary.Drivers.NewSignups)

	// Utilization rate (simplified calculation)
	var totalOnlineTime, totalBusyTime float64
	r.db.QueryRow(ctx, `
		SELECT
			COUNT(DISTINCT driver_id) * EXTRACT(EPOCH FROM (NOW() - $1))/3600 as total_hours,
			COALESCE(SUM(actual_duration/60), 0) as busy_hours
		FROM rides
		WHERE created_at >= $1 AND status = 'completed'
	`, startDate).Scan(&totalOnlineTime, &totalBusyTime)

	if totalOnlineTime > 0 {
		summary.Drivers.UtilizationRate = (totalBusyTime / totalOnlineTime) * 100
	}

	// Get rider statistics
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'rider' AND is_active = true`).Scan(&summary.Riders.TotalActive)
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'rider' AND created_at >= $1`, startDate).Scan(&summary.Riders.NewSignups)
	r.db.QueryRow(ctx, `SELECT COUNT(DISTINCT rider_id) FROM rides WHERE created_at >= $1`, startDate).Scan(&summary.Riders.ActiveToday)

	// Retention rate (simplified - riders who made > 1 ride)
	var repeatRiders int
	r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM (
			SELECT rider_id FROM rides GROUP BY rider_id HAVING COUNT(*) > 1
		) as repeat_riders
	`).Scan(&repeatRiders)

	if summary.Riders.TotalActive > 0 {
		summary.Riders.RetentionRate = float64(repeatRiders) / float64(summary.Riders.TotalActive) * 100
	}

	// Get revenue statistics
	r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(final_fare), 0),
			COALESCE(AVG(final_fare), 0)
		FROM rides
		WHERE status = 'completed' AND created_at >= $1
	`, startDate).Scan(&summary.Revenue.Total, &summary.Revenue.AvgFare)

	// Calculate commission (20% platform fee)
	summary.Revenue.Commission = summary.Revenue.Total * 0.20
	summary.Revenue.DriverEarnings = summary.Revenue.Total - summary.Revenue.Commission

	// Get previous period revenue for comparison
	var previousRevenue float64
	r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(final_fare), 0)
		FROM rides
		WHERE status = 'completed' AND created_at >= $1 AND created_at < $2
	`, previousStart, startDate).Scan(&previousRevenue)

	if previousRevenue > 0 {
		summary.Revenue.ChangeVsPrevious = ((summary.Revenue.Total - previousRevenue) / previousRevenue) * 100
	}

	// Payment methods breakdown (simplified - using mock data as payment_method not in rides table)
	summary.Revenue.ByPaymentMethod = []PaymentMethodRevenue{
		{Method: "card", Amount: summary.Revenue.Total * 0.77, Percentage: 77.3},
		{Method: "wallet", Amount: summary.Revenue.Total * 0.18, Percentage: 17.7},
		{Method: "cash", Amount: summary.Revenue.Total * 0.05, Percentage: 5.0},
	}

	// Get alert statistics
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM fraud_alerts WHERE status = 'pending'`).Scan(&summary.Alerts.FraudAlerts)
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM fraud_alerts WHERE alert_level = 'critical' AND status = 'pending'`).Scan(&summary.Alerts.CriticalAlerts)
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM fraud_alerts WHERE status = 'investigating'`).Scan(&summary.Alerts.PendingInvestigations)

	return summary, nil
}

// getPeriodDates returns start date and previous period start date
func (r *Repository) getPeriodDates(period string) (time.Time, time.Time) {
	now := time.Now()
	var startDate, previousStart time.Time

	switch period {
	case "today":
		startDate = now.Truncate(24 * time.Hour)
		previousStart = startDate.Add(-24 * time.Hour)
	case "week":
		startDate = now.AddDate(0, 0, -7).Truncate(24 * time.Hour)
		previousStart = startDate.Add(-7 * 24 * time.Hour)
	case "month":
		startDate = now.AddDate(0, -1, 0).Truncate(24 * time.Hour)
		previousStart = startDate.AddDate(0, -1, 0)
	default: // "all"
		startDate = time.Time{}
		previousStart = time.Time{}
	}

	return startDate, previousStart
}

// GetRevenueTrend retrieves revenue trend data with flexible grouping
func (r *Repository) GetRevenueTrend(ctx context.Context, period, groupBy string) (*RevenueTrend, error) {
	startDate, days := r.getPeriodStartDate(period)

	trend := &RevenueTrend{
		Period:  period,
		GroupBy: groupBy,
		Trend:   make([]RevenueTrendData, 0),
	}

	// Get total revenue for the period
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(final_fare), 0)
		FROM rides
		WHERE status = 'completed' AND created_at >= $1
	`, startDate).Scan(&trend.TotalRevenue)
	if err != nil {
		return nil, fmt.Errorf("failed to get total revenue: %w", err)
	}

	if days > 0 {
		trend.AvgDailyRevenue = trend.TotalRevenue / float64(days)
	}

	// Build query based on groupBy parameter
	query := r.buildRevenueTrendQuery(groupBy)

	rows, err := r.db.Query(ctx, query, startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue trend: %w", err)
	}
	defer rows.Close()

	// Parse results based on groupBy format
	for rows.Next() {
		var data RevenueTrendData
		var dateValue interface{}

		err := rows.Scan(&dateValue, &data.Revenue, &data.Rides, &data.AvgFare, &data.Commission)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trend data: %w", err)
		}

		// Format date based on groupBy
		data.Date = r.formatTrendDate(dateValue, groupBy)
		trend.Trend = append(trend.Trend, data)
	}

	return trend, nil
}

// buildRevenueTrendQuery builds SQL query based on groupBy parameter
func (r *Repository) buildRevenueTrendQuery(groupBy string) string {
	baseQuery := `
		SELECT
			%s as date,
			COALESCE(SUM(final_fare), 0) as revenue,
			COUNT(*) as rides,
			COALESCE(AVG(final_fare), 0) as avg_fare,
			COALESCE(SUM(final_fare), 0) * 0.20 as commission
		FROM rides
		WHERE status = 'completed' AND created_at >= $1
		GROUP BY %s
		ORDER BY %s ASC
	`

	switch groupBy {
	case "hour":
		groupExpr := "DATE_TRUNC('hour', created_at)"
		return fmt.Sprintf(baseQuery, groupExpr, groupExpr, groupExpr)
	case "day":
		groupExpr := "DATE(created_at)"
		return fmt.Sprintf(baseQuery, groupExpr, groupExpr, groupExpr)
	case "week":
		groupExpr := "DATE_TRUNC('week', created_at)"
		return fmt.Sprintf(baseQuery, groupExpr, groupExpr, groupExpr)
	case "month":
		groupExpr := "DATE_TRUNC('month', created_at)"
		return fmt.Sprintf(baseQuery, groupExpr, groupExpr, groupExpr)
	default: // default to day
		groupExpr := "DATE(created_at)"
		return fmt.Sprintf(baseQuery, groupExpr, groupExpr, groupExpr)
	}
}

// formatTrendDate formats the date value based on groupBy type
func (r *Repository) formatTrendDate(dateValue interface{}, groupBy string) string {
	switch v := dateValue.(type) {
	case time.Time:
		switch groupBy {
		case "hour":
			return v.Format("2006-01-02 15:00") // e.g., "2026-01-13 14:00"
		case "week":
			return v.Format("2006-01-02") // Monday of the week
		case "month":
			return v.Format("2006-01") // e.g., "2026-01"
		default: // day
			return v.Format("2006-01-02")
		}
	default:
		return fmt.Sprintf("%v", dateValue)
	}
}

// getPeriodDays returns number of days for the period
func (r *Repository) getPeriodDays(period string) int {
	switch period {
	case "today":
		return 1
	case "7days":
		return 7
	case "30days":
		return 30
	case "90days":
		return 90
	case "year":
		return 365
	default:
		return 7
	}
}

// getPeriodStartDate returns start date and number of days for the period
func (r *Repository) getPeriodStartDate(period string) (time.Time, int) {
	now := time.Now()
	var startDate time.Time
	var days int

	switch period {
	case "today":
		startDate = now.Truncate(24 * time.Hour)
		days = 1
	case "7days":
		startDate = now.AddDate(0, 0, -7).Truncate(24 * time.Hour)
		days = 7
	case "30days":
		startDate = now.AddDate(0, 0, -30).Truncate(24 * time.Hour)
		days = 30
	case "90days":
		startDate = now.AddDate(0, 0, -90).Truncate(24 * time.Hour)
		days = 90
	case "year":
		startDate = now.AddDate(-1, 0, 0).Truncate(24 * time.Hour)
		days = 365
	default: // default to 7 days
		startDate = now.AddDate(0, 0, -7).Truncate(24 * time.Hour)
		days = 7
	}

	return startDate, days
}

// GetActionItems retrieves items requiring admin attention
func (r *Repository) GetActionItems(ctx context.Context) (*ActionItems, error) {
	items := &ActionItems{
		PendingDriverApprovals: &ActionDriverApprovals{Items: make([]PendingDriverItem, 0)},
		FraudAlerts:            &ActionFraudAlerts{Items: make([]FraudAlertItem, 0)},
		NegativeFeedback:       &ActionNegativeFeedback{Items: make([]NegativeFeedbackItem, 0)},
		LowBalanceDrivers:      &ActionLowBalance{Items: make([]LowBalanceItem, 0)},
		ExpiredDocuments:       &ActionExpiredDocs{Items: make([]ExpiredDocItem, 0)},
	}

	// Get pending driver approvals
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM drivers WHERE approval_status = 'pending'`).Scan(&items.PendingDriverApprovals.Count)
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM drivers WHERE approval_status = 'pending' AND created_at < NOW() - INTERVAL '24 hours'`).Scan(&items.PendingDriverApprovals.UrgentCount)

	// Get first 5 pending drivers
	rows, err := r.db.Query(ctx, `
		SELECT d.id, CONCAT(u.first_name, ' ', u.last_name) as name, d.created_at,
		       EXTRACT(DAY FROM (NOW() - d.created_at)) as days_waiting
		FROM drivers d
		JOIN users u ON d.user_id = u.id
		WHERE d.approval_status = 'pending'
		ORDER BY d.created_at ASC
		LIMIT 5
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var item PendingDriverItem
			rows.Scan(&item.DriverID, &item.DriverName, &item.SubmittedAt, &item.DaysWaiting)
			items.PendingDriverApprovals.Items = append(items.PendingDriverApprovals.Items, item)
		}
	}

	// Get fraud alerts
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM fraud_alerts WHERE status = 'pending'`).Scan(&items.FraudAlerts.Count)
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM fraud_alerts WHERE alert_level = 'critical' AND status = 'pending'`).Scan(&items.FraudAlerts.CriticalCount)
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM fraud_alerts WHERE alert_level = 'high' AND status = 'pending'`).Scan(&items.FraudAlerts.HighCount)

	// Get first 5 fraud alerts
	rows, err = r.db.Query(ctx, `
		SELECT id, alert_type, alert_level, user_id, created_at
		FROM fraud_alerts
		WHERE status = 'pending'
		ORDER BY alert_level DESC, created_at DESC
		LIMIT 5
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var item FraudAlertItem
			rows.Scan(&item.AlertID, &item.AlertType, &item.AlertLevel, &item.UserID, &item.CreatedAt)
			items.FraudAlerts.Items = append(items.FraudAlerts.Items, item)
		}
	}

	// Get negative feedback (1-2 star ratings)
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM rides WHERE rating IS NOT NULL AND rating <= 2`).Scan(&items.NegativeFeedback.Count)
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM rides WHERE rating = 1`).Scan(&items.NegativeFeedback.OneStarCount)

	// Get first 5 negative feedback items
	rows, err = r.db.Query(ctx, `
		SELECT r.id, r.driver_id, CONCAT(u.first_name, ' ', u.last_name) as driver_name,
		       r.rating, r.feedback, r.created_at
		FROM rides r
		LEFT JOIN drivers d ON r.driver_id = d.user_id
		LEFT JOIN users u ON r.driver_id = u.id
		WHERE r.rating IS NOT NULL AND r.rating <= 2
		ORDER BY r.created_at DESC
		LIMIT 5
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var item NegativeFeedbackItem
			rows.Scan(&item.RideID, &item.DriverID, &item.DriverName, &item.Rating, &item.Feedback, &item.CreatedAt)
			items.NegativeFeedback.Items = append(items.NegativeFeedback.Items, item)
		}
	}

	// Low balance drivers (assuming wallet integration - using mock for now)
	items.LowBalanceDrivers.Count = 0 // Would query wallet service

	// Expired documents (not implemented in current schema - using mock)
	items.ExpiredDocuments.Count = 0

	return items, nil
}
