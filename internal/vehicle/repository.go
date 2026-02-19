package vehicle

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles vehicle data access
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new vehicle repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// VEHICLES
// ========================================

// CreateVehicle registers a new vehicle
func (r *Repository) CreateVehicle(ctx context.Context, v *Vehicle) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO vehicles (
			id, driver_id, status, category,
			make, model, year, color, license_plate, vin,
			fuel_type, max_passengers, has_child_seat,
			has_wheelchair_access, has_wifi, has_charger,
			pet_friendly, luggage_capacity,
			is_active, is_primary, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22
		)`,
		v.ID, v.DriverID, v.Status, v.Category,
		v.Make, v.Model, v.Year, v.Color, v.LicensePlate, v.VIN,
		v.FuelType, v.MaxPassengers, v.HasChildSeat,
		v.HasWheelchairAccess, v.HasWifi, v.HasCharger,
		v.PetFriendly, v.LuggageCapacity,
		v.IsActive, v.IsPrimary, v.CreatedAt, v.UpdatedAt,
	)
	return err
}

// vehicleSelectCols is the standard column list for vehicle SELECT queries.
const vehicleSelectCols = `
	id, driver_id, status, category,
	make, model, year, color, license_plate, vin,
	fuel_type, max_passengers, has_child_seat,
	has_wheelchair_access, has_wifi, has_charger,
	pet_friendly, luggage_capacity,
	registration_photo_url, insurance_photo_url,
	front_photo_url, back_photo_url, side_photo_url, interior_photo_url,
	insurance_expiry, registration_expiry, inspection_expiry,
	rejection_reason, is_active, is_primary,
	created_at, updated_at`

// GetVehicleByID retrieves a vehicle by ID
func (r *Repository) GetVehicleByID(ctx context.Context, id uuid.UUID) (*Vehicle, error) {
	v := &Vehicle{}
	err := r.db.QueryRow(ctx, `SELECT `+vehicleSelectCols+` FROM vehicles WHERE id = $1`, id,
	).Scan(
		&v.ID, &v.DriverID, &v.Status, &v.Category,
		&v.Make, &v.Model, &v.Year, &v.Color, &v.LicensePlate, &v.VIN,
		&v.FuelType, &v.MaxPassengers, &v.HasChildSeat,
		&v.HasWheelchairAccess, &v.HasWifi, &v.HasCharger,
		&v.PetFriendly, &v.LuggageCapacity,
		&v.RegistrationPhotoURL, &v.InsurancePhotoURL,
		&v.FrontPhotoURL, &v.BackPhotoURL, &v.SidePhotoURL, &v.InteriorPhotoURL,
		&v.InsuranceExpiry, &v.RegistrationExpiry, &v.InspectionExpiry,
		&v.RejectionReason, &v.IsActive, &v.IsPrimary,
		&v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// GetVehiclesByDriver returns vehicles for a driver with total count
func (r *Repository) GetVehiclesByDriver(ctx context.Context, driverID uuid.UUID, limit, offset int) ([]Vehicle, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM vehicles WHERE driver_id = $1`, driverID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+vehicleSelectCols+` FROM vehicles WHERE driver_id = $1 ORDER BY is_primary DESC, created_at DESC LIMIT $2 OFFSET $3`,
		driverID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	vehicles := make([]Vehicle, 0)
	for rows.Next() {
		v := Vehicle{}
		if err := rows.Scan(
			&v.ID, &v.DriverID, &v.Status, &v.Category,
			&v.Make, &v.Model, &v.Year, &v.Color, &v.LicensePlate, &v.VIN,
			&v.FuelType, &v.MaxPassengers, &v.HasChildSeat,
			&v.HasWheelchairAccess, &v.HasWifi, &v.HasCharger,
			&v.PetFriendly, &v.LuggageCapacity,
			&v.RegistrationPhotoURL, &v.InsurancePhotoURL,
			&v.FrontPhotoURL, &v.BackPhotoURL, &v.SidePhotoURL, &v.InteriorPhotoURL,
			&v.InsuranceExpiry, &v.RegistrationExpiry, &v.InspectionExpiry,
			&v.RejectionReason, &v.IsActive, &v.IsPrimary,
			&v.CreatedAt, &v.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		vehicles = append(vehicles, v)
	}
	return vehicles, total, nil
}

// GetPrimaryVehicle returns the driver's active primary vehicle
func (r *Repository) GetPrimaryVehicle(ctx context.Context, driverID uuid.UUID) (*Vehicle, error) {
	v := &Vehicle{}
	err := r.db.QueryRow(ctx,
		`SELECT `+vehicleSelectCols+` FROM vehicles WHERE driver_id = $1 AND is_primary = true AND is_active = true`,
		driverID,
	).Scan(
		&v.ID, &v.DriverID, &v.Status, &v.Category,
		&v.Make, &v.Model, &v.Year, &v.Color, &v.LicensePlate, &v.VIN,
		&v.FuelType, &v.MaxPassengers, &v.HasChildSeat,
		&v.HasWheelchairAccess, &v.HasWifi, &v.HasCharger,
		&v.PetFriendly, &v.LuggageCapacity,
		&v.RegistrationPhotoURL, &v.InsurancePhotoURL,
		&v.FrontPhotoURL, &v.BackPhotoURL, &v.SidePhotoURL, &v.InteriorPhotoURL,
		&v.InsuranceExpiry, &v.RegistrationExpiry, &v.InspectionExpiry,
		&v.RejectionReason, &v.IsActive, &v.IsPrimary,
		&v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// UpdateVehicle updates vehicle details
func (r *Repository) UpdateVehicle(ctx context.Context, v *Vehicle) error {
	_, err := r.db.Exec(ctx, `
		UPDATE vehicles
		SET color = $2, category = $3, max_passengers = $4,
			has_child_seat = $5, has_wheelchair_access = $6,
			has_wifi = $7, has_charger = $8,
			pet_friendly = $9, luggage_capacity = $10,
			updated_at = $11
		WHERE id = $1`,
		v.ID, v.Color, v.Category, v.MaxPassengers,
		v.HasChildSeat, v.HasWheelchairAccess,
		v.HasWifi, v.HasCharger,
		v.PetFriendly, v.LuggageCapacity,
		v.UpdatedAt,
	)
	return err
}

// UpdateVehiclePhotos updates vehicle photos
func (r *Repository) UpdateVehiclePhotos(ctx context.Context, vehicleID uuid.UUID, photos *UploadVehiclePhotosRequest) error {
	_, err := r.db.Exec(ctx, `
		UPDATE vehicles
		SET registration_photo_url = COALESCE($2, registration_photo_url),
			insurance_photo_url = COALESCE($3, insurance_photo_url),
			front_photo_url = COALESCE($4, front_photo_url),
			back_photo_url = COALESCE($5, back_photo_url),
			side_photo_url = COALESCE($6, side_photo_url),
			interior_photo_url = COALESCE($7, interior_photo_url),
			updated_at = NOW()
		WHERE id = $1`,
		vehicleID, photos.RegistrationPhotoURL, photos.InsurancePhotoURL,
		photos.FrontPhotoURL, photos.BackPhotoURL,
		photos.SidePhotoURL, photos.InteriorPhotoURL,
	)
	return err
}

// UpdateVehicleStatus updates the review status
func (r *Repository) UpdateVehicleStatus(ctx context.Context, vehicleID uuid.UUID, status VehicleStatus, reason *string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE vehicles
		SET status = $2, rejection_reason = $3, updated_at = NOW()
		WHERE id = $1`,
		vehicleID, status, reason,
	)
	return err
}

// SetPrimaryVehicle sets the driver's primary vehicle
func (r *Repository) SetPrimaryVehicle(ctx context.Context, driverID, vehicleID uuid.UUID) error {
	// Unset current primary
	r.db.Exec(ctx, `
		UPDATE vehicles SET is_primary = false, updated_at = NOW()
		WHERE driver_id = $1`, driverID)

	// Set new primary
	_, err := r.db.Exec(ctx, `
		UPDATE vehicles SET is_primary = true, updated_at = NOW()
		WHERE id = $1 AND driver_id = $2`,
		vehicleID, driverID,
	)
	return err
}

// RetireVehicle marks a vehicle as retired
func (r *Repository) RetireVehicle(ctx context.Context, vehicleID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE vehicles
		SET status = 'retired', is_active = false, is_primary = false, updated_at = NOW()
		WHERE id = $1`,
		vehicleID,
	)
	return err
}

// ========================================
// ADMIN QUERIES
// ========================================

// GetPendingReviewVehicles returns vehicles awaiting admin review
func (r *Repository) GetPendingReviewVehicles(ctx context.Context, limit, offset int) ([]Vehicle, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM vehicles WHERE status = 'pending'`,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+vehicleSelectCols+` FROM vehicles WHERE status = 'pending' ORDER BY created_at ASC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var vehicles []Vehicle
	for rows.Next() {
		v := Vehicle{}
		if err := rows.Scan(
			&v.ID, &v.DriverID, &v.Status, &v.Category,
			&v.Make, &v.Model, &v.Year, &v.Color, &v.LicensePlate, &v.VIN,
			&v.FuelType, &v.MaxPassengers, &v.HasChildSeat,
			&v.HasWheelchairAccess, &v.HasWifi, &v.HasCharger,
			&v.PetFriendly, &v.LuggageCapacity,
			&v.RegistrationPhotoURL, &v.InsurancePhotoURL,
			&v.FrontPhotoURL, &v.BackPhotoURL, &v.SidePhotoURL, &v.InteriorPhotoURL,
			&v.InsuranceExpiry, &v.RegistrationExpiry, &v.InspectionExpiry,
			&v.RejectionReason, &v.IsActive, &v.IsPrimary,
			&v.CreatedAt, &v.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		vehicles = append(vehicles, v)
	}
	return vehicles, total, nil
}

// GetAllVehicles returns vehicles for admin with filters and pagination
func (r *Repository) GetAllVehicles(ctx context.Context, filter *AdminVehicleFilter, limit, offset int) ([]Vehicle, int64, error) {
	args := []interface{}{}
	where := []string{}
	i := 1

	if filter.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", i))
		args = append(args, filter.Status)
		i++
	}
	if filter.Category != "" {
		where = append(where, fmt.Sprintf("category = $%d", i))
		args = append(args, filter.Category)
		i++
	}
	if filter.DriverID != nil {
		where = append(where, fmt.Sprintf("driver_id = $%d", i))
		args = append(args, *filter.DriverID)
		i++
	}
	if filter.Search != "" {
		where = append(where, fmt.Sprintf("(make ILIKE $%d OR model ILIKE $%d OR license_plate ILIKE $%d)", i, i, i))
		args = append(args, "%"+filter.Search+"%")
		i++
	}
	if filter.YearFrom != nil {
		where = append(where, fmt.Sprintf("year >= $%d", i))
		args = append(args, *filter.YearFrom)
		i++
	}
	if filter.YearTo != nil {
		where = append(where, fmt.Sprintf("year <= $%d", i))
		args = append(args, *filter.YearTo)
		i++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	sortCol := "created_at"
	if filter.SortBy == "updated_at" {
		sortCol = "updated_at"
	}
	sortDir := "DESC"
	if strings.ToUpper(filter.SortDir) == "ASC" {
		sortDir = "ASC"
	}

	var total int64
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if err := r.db.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM vehicles %s", whereClause),
		countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(
		"SELECT %s FROM vehicles %s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		vehicleSelectCols, whereClause, sortCol, sortDir, i, i+1,
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	vehicles := make([]Vehicle, 0)
	for rows.Next() {
		v := Vehicle{}
		if err := rows.Scan(
			&v.ID, &v.DriverID, &v.Status, &v.Category,
			&v.Make, &v.Model, &v.Year, &v.Color, &v.LicensePlate, &v.VIN,
			&v.FuelType, &v.MaxPassengers, &v.HasChildSeat,
			&v.HasWheelchairAccess, &v.HasWifi, &v.HasCharger,
			&v.PetFriendly, &v.LuggageCapacity,
			&v.RegistrationPhotoURL, &v.InsurancePhotoURL,
			&v.FrontPhotoURL, &v.BackPhotoURL, &v.SidePhotoURL, &v.InteriorPhotoURL,
			&v.InsuranceExpiry, &v.RegistrationExpiry, &v.InspectionExpiry,
			&v.RejectionReason, &v.IsActive, &v.IsPrimary,
			&v.CreatedAt, &v.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		vehicles = append(vehicles, v)
	}
	return vehicles, total, nil
}

// GetVehicleStats returns stats for admin dashboard
func (r *Repository) GetVehicleStats(ctx context.Context) (*VehicleStats, error) {
	stats := &VehicleStats{}
	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(CASE WHEN status = 'pending' THEN 1 END),
			COUNT(CASE WHEN status = 'approved' THEN 1 END),
			COUNT(CASE WHEN status = 'suspended' THEN 1 END),
			COUNT(CASE WHEN insurance_expiry IS NOT NULL
				AND insurance_expiry < NOW() + INTERVAL '30 days' THEN 1 END),
			COUNT(CASE WHEN registration_expiry IS NOT NULL
				AND registration_expiry < NOW() + INTERVAL '30 days' THEN 1 END)
		FROM vehicles
		WHERE is_active = true`,
	).Scan(
		&stats.TotalVehicles,
		&stats.PendingReview,
		&stats.ApprovedVehicles,
		&stats.SuspendedVehicles,
		&stats.ExpiringInsurance,
		&stats.ExpiringRegistration,
	)
	return stats, err
}

// GetExpiringVehicles returns vehicles with expiring documents
func (r *Repository) GetExpiringVehicles(ctx context.Context, daysAhead int) ([]Vehicle, error) {
	threshold := time.Now().AddDate(0, 0, daysAhead)
	rows, err := r.db.Query(ctx,
		`SELECT `+vehicleSelectCols+`
		FROM vehicles
		WHERE is_active = true AND status = 'approved'
			AND (
				(insurance_expiry IS NOT NULL AND insurance_expiry < $1)
				OR (registration_expiry IS NOT NULL AND registration_expiry < $1)
				OR (inspection_expiry IS NOT NULL AND inspection_expiry < $1)
			)
		ORDER BY LEAST(
			COALESCE(insurance_expiry, '9999-12-31'),
			COALESCE(registration_expiry, '9999-12-31'),
			COALESCE(inspection_expiry, '9999-12-31')
		)`, threshold,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vehicles []Vehicle
	for rows.Next() {
		v := Vehicle{}
		if err := rows.Scan(
			&v.ID, &v.DriverID, &v.Status, &v.Category,
			&v.Make, &v.Model, &v.Year, &v.Color, &v.LicensePlate, &v.VIN,
			&v.FuelType, &v.MaxPassengers, &v.HasChildSeat,
			&v.HasWheelchairAccess, &v.HasWifi, &v.HasCharger,
			&v.PetFriendly, &v.LuggageCapacity,
			&v.RegistrationPhotoURL, &v.InsurancePhotoURL,
			&v.FrontPhotoURL, &v.BackPhotoURL, &v.SidePhotoURL, &v.InteriorPhotoURL,
			&v.InsuranceExpiry, &v.RegistrationExpiry, &v.InspectionExpiry,
			&v.RejectionReason, &v.IsActive, &v.IsPrimary,
			&v.CreatedAt, &v.UpdatedAt,
		); err != nil {
			return nil, err
		}
		vehicles = append(vehicles, v)
	}
	return vehicles, nil
}

// ========================================
// INSPECTIONS
// ========================================

// CreateInspection creates a vehicle inspection record
func (r *Repository) CreateInspection(ctx context.Context, insp *VehicleInspection) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO vehicle_inspections (
			id, vehicle_id, inspector_id, status,
			scheduled_at, notes, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		insp.ID, insp.VehicleID, insp.InspectorID, insp.Status,
		insp.ScheduledAt, insp.Notes, insp.CreatedAt,
	)
	return err
}

// GetInspectionsByVehicle returns inspections for a vehicle
func (r *Repository) GetInspectionsByVehicle(ctx context.Context, vehicleID uuid.UUID) ([]VehicleInspection, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, vehicle_id, inspector_id, status,
			scheduled_at, completed_at, notes, created_at
		FROM vehicle_inspections
		WHERE vehicle_id = $1
		ORDER BY scheduled_at DESC`, vehicleID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inspections []VehicleInspection
	for rows.Next() {
		insp := VehicleInspection{}
		if err := rows.Scan(
			&insp.ID, &insp.VehicleID, &insp.InspectorID, &insp.Status,
			&insp.ScheduledAt, &insp.CompletedAt, &insp.Notes, &insp.CreatedAt,
		); err != nil {
			return nil, err
		}
		inspections = append(inspections, insp)
	}
	return inspections, nil
}

// ========================================
// MAINTENANCE REMINDERS
// ========================================

// CreateMaintenanceReminder creates a maintenance reminder
func (r *Repository) CreateMaintenanceReminder(ctx context.Context, rem *MaintenanceReminder) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO maintenance_reminders (
			id, vehicle_id, type, description,
			due_date, due_mileage, is_completed, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		rem.ID, rem.VehicleID, rem.Type, rem.Description,
		rem.DueDate, rem.DueMileage, rem.IsCompleted, rem.CreatedAt,
	)
	return err
}

// GetPendingReminders returns pending maintenance reminders
func (r *Repository) GetPendingReminders(ctx context.Context, vehicleID uuid.UUID) ([]MaintenanceReminder, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, vehicle_id, type, description,
			due_date, due_mileage, is_completed, completed_at, created_at
		FROM maintenance_reminders
		WHERE vehicle_id = $1 AND is_completed = false
		ORDER BY due_date ASC NULLS LAST`, vehicleID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reminders []MaintenanceReminder
	for rows.Next() {
		rem := MaintenanceReminder{}
		if err := rows.Scan(
			&rem.ID, &rem.VehicleID, &rem.Type, &rem.Description,
			&rem.DueDate, &rem.DueMileage, &rem.IsCompleted, &rem.CompletedAt, &rem.CreatedAt,
		); err != nil {
			return nil, err
		}
		reminders = append(reminders, rem)
	}
	return reminders, nil
}

// CompleteReminder marks a reminder as completed
func (r *Repository) CompleteReminder(ctx context.Context, reminderID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE maintenance_reminders
		SET is_completed = true, completed_at = NOW()
		WHERE id = $1`,
		reminderID,
	)
	return err
}
