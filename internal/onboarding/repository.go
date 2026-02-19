package onboarding

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for onboarding
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new onboarding repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// DriverInfo represents driver information needed for onboarding
type DriverInfo struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	FirstName      string     `json:"first_name"`
	LastName       string     `json:"last_name"`
	PhoneNumber    string     `json:"phone_number"`
	Email          string     `json:"email"`
	LicenseNumber  *string    `json:"license_number,omitempty"`
	ApprovalStatus string     `json:"approval_status"` // pending, approved, rejected
	IsApproved     bool       `json:"is_approved"`
	IsSuspended    bool       `json:"is_suspended"`
	ApprovedAt     *time.Time `json:"approved_at,omitempty"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

// BackgroundCheck represents a background check record
type BackgroundCheck struct {
	ID          uuid.UUID  `json:"id"`
	DriverID    uuid.UUID  `json:"driver_id"`
	Status      string     `json:"status"` // pending, in_progress, passed, failed
	Provider    string     `json:"provider,omitempty"`
	ReportURL   *string    `json:"report_url,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// GetDriver retrieves driver information
func (r *Repository) GetDriver(ctx context.Context, driverID uuid.UUID) (*DriverInfo, error) {
	query := `
		SELECT d.id, d.user_id, u.first_name, u.last_name, u.phone_number, u.email,
			   d.license_number, d.approval_status,
			   (u.is_active = false) as is_suspended,
			   d.approved_at, d.created_at, d.updated_at
		FROM drivers d
		JOIN users u ON d.user_id = u.id
		WHERE d.id = $1
	`

	driver := &DriverInfo{}
	err := r.db.QueryRow(ctx, query, driverID).Scan(
		&driver.ID, &driver.UserID, &driver.FirstName, &driver.LastName,
		&driver.PhoneNumber, &driver.Email, &driver.LicenseNumber,
		&driver.ApprovalStatus, &driver.IsSuspended,
		&driver.ApprovedAt, &driver.CreatedAt, &driver.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	driver.IsApproved = driver.ApprovalStatus == "approved"
	return driver, nil
}

// GetDriverByUserID retrieves driver by user ID
func (r *Repository) GetDriverByUserID(ctx context.Context, userID uuid.UUID) (*DriverInfo, error) {
	query := `
		SELECT d.id, d.user_id, u.first_name, u.last_name, u.phone_number, u.email,
			   d.license_number, d.approval_status,
			   (u.is_active = false) as is_suspended,
			   d.approved_at, d.created_at, d.updated_at
		FROM drivers d
		JOIN users u ON d.user_id = u.id
		WHERE d.user_id = $1
	`

	driver := &DriverInfo{}
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&driver.ID, &driver.UserID, &driver.FirstName, &driver.LastName,
		&driver.PhoneNumber, &driver.Email, &driver.LicenseNumber,
		&driver.ApprovalStatus, &driver.IsSuspended,
		&driver.ApprovedAt, &driver.CreatedAt, &driver.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	driver.IsApproved = driver.ApprovalStatus == "approved"
	return driver, nil
}

// CreateDriver creates a new driver record with placeholder values for required fields.
// Vehicle/license details are filled in during subsequent onboarding steps.
// Uses userID as placeholder for unique fields to avoid conflicts on re-registration.
func (r *Repository) CreateDriver(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	driverID := uuid.New()

	// Use userID as placeholder for license_number and vehicle_plate (both unique).
	// Truncate to 20 chars for vehicle_plate (VARCHAR(20)).
	// ON CONFLICT (user_id) returns the existing driver id without error on retry.
	query := `
		INSERT INTO drivers (id, user_id, license_number, vehicle_model, vehicle_plate, vehicle_color, vehicle_year,
		                     is_online, is_available, approval_status, created_at, updated_at)
		VALUES ($1, $2, $2::text, '', LEFT($2::text, 20), '', 0, false, false, 'pending', NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET updated_at = NOW()
		RETURNING id
	`

	var returnedID uuid.UUID
	err := r.db.QueryRow(ctx, query, driverID, userID).Scan(&returnedID)
	if err != nil {
		return uuid.Nil, err
	}

	return returnedID, nil
}

// HasApprovedVehicle returns true if the user has at least one approved, active vehicle.
// Note: vehicles.driver_id is a FK to users.id (not drivers.id), so we query by user_id.
func (r *Repository) HasApprovedVehicle(ctx context.Context, userID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM vehicles WHERE driver_id = $1 AND status = 'approved' AND is_active = true`,
		userID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetBackgroundCheck retrieves background check info for a driver
func (r *Repository) GetBackgroundCheck(ctx context.Context, driverID uuid.UUID) (*BackgroundCheck, error) {
	query := `
		SELECT id, driver_id, status, provider, report_url, notes, started_at, completed_at, created_at
		FROM driver_background_checks
		WHERE driver_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	check := &BackgroundCheck{}
	err := r.db.QueryRow(ctx, query, driverID).Scan(
		&check.ID, &check.DriverID, &check.Status, &check.Provider,
		&check.ReportURL, &check.Notes, &check.StartedAt, &check.CompletedAt, &check.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return check, nil
}

// CreateBackgroundCheck creates a new background check record
func (r *Repository) CreateBackgroundCheck(ctx context.Context, driverID uuid.UUID, provider string) (*BackgroundCheck, error) {
	check := &BackgroundCheck{
		ID:        uuid.New(),
		DriverID:  driverID,
		Status:    "pending",
		Provider:  provider,
		CreatedAt: time.Now(),
	}

	query := `
		INSERT INTO driver_background_checks (id, driver_id, status, provider, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.Exec(ctx, query, check.ID, check.DriverID, check.Status, check.Provider, check.CreatedAt)
	if err != nil {
		return nil, err
	}

	return check, nil
}

// UpdateBackgroundCheckStatus updates background check status
func (r *Repository) UpdateBackgroundCheckStatus(ctx context.Context, checkID uuid.UUID, status string, notes *string) error {
	query := `
		UPDATE driver_background_checks
		SET status = $1, notes = $2,
		    started_at = CASE WHEN $1 = 'in_progress' AND started_at IS NULL THEN NOW() ELSE started_at END,
		    completed_at = CASE WHEN $1 IN ('passed', 'failed') THEN NOW() ELSE completed_at END,
		    updated_at = NOW()
		WHERE id = $3
	`

	_, err := r.db.Exec(ctx, query, status, notes, checkID)
	return err
}

// ApproveDriver marks a driver as approved
func (r *Repository) ApproveDriver(ctx context.Context, driverID uuid.UUID, approvedBy uuid.UUID) error {
	query := `
		UPDATE drivers
		SET approval_status = 'approved', approved_at = NOW(), approved_by = $1, updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, approvedBy, driverID)
	return err
}

// RejectDriver marks a driver as rejected
func (r *Repository) RejectDriver(ctx context.Context, driverID uuid.UUID, rejectedBy uuid.UUID, reason string) error {
	query := `
		UPDATE drivers
		SET approval_status = 'rejected', rejection_reason = $1, rejected_at = NOW(), updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, reason, driverID)
	return err
}

// GetOnboardingStats gets onboarding statistics
func (r *Repository) GetOnboardingStats(ctx context.Context) (*OnboardingStats, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE approval_status = 'pending') as pending_count,
			COUNT(*) FILTER (WHERE approval_status = 'approved') as approved_count,
			COUNT(*) FILTER (WHERE approval_status = 'rejected') as rejected_count,
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '7 days') as new_this_week,
			AVG(EXTRACT(EPOCH FROM (approved_at - created_at))/3600) FILTER (WHERE approved_at IS NOT NULL) as avg_approval_hours
		FROM drivers
	`

	stats := &OnboardingStats{}
	err := r.db.QueryRow(ctx, query).Scan(
		&stats.PendingCount, &stats.ApprovedCount, &stats.RejectedCount,
		&stats.NewThisWeek, &stats.AvgApprovalHours,
	)

	if err != nil {
		return nil, err
	}

	return stats, nil
}

// OnboardingStats represents onboarding statistics
type OnboardingStats struct {
	PendingCount     int      `json:"pending_count"`
	ApprovedCount    int      `json:"approved_count"`
	RejectedCount    int      `json:"rejected_count"`
	NewThisWeek      int      `json:"new_this_week"`
	AvgApprovalHours *float64 `json:"avg_approval_hours,omitempty"`
}
