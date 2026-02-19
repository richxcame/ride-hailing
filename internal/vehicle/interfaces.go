package vehicle

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryInterface defines the contract for vehicle repository operations
type RepositoryInterface interface {
	// Vehicle CRUD operations
	CreateVehicle(ctx context.Context, v *Vehicle) error
	GetVehicleByID(ctx context.Context, id uuid.UUID) (*Vehicle, error)
	GetVehiclesByDriver(ctx context.Context, driverID uuid.UUID, limit, offset int) ([]Vehicle, int64, error)
	GetPrimaryVehicle(ctx context.Context, driverID uuid.UUID) (*Vehicle, error)
	UpdateVehicle(ctx context.Context, v *Vehicle) error
	UpdateVehiclePhotos(ctx context.Context, vehicleID uuid.UUID, photos *UploadVehiclePhotosRequest) error
	UpdateVehicleStatus(ctx context.Context, vehicleID uuid.UUID, status VehicleStatus, reason *string) error
	SetPrimaryVehicle(ctx context.Context, driverID, vehicleID uuid.UUID) error
	RetireVehicle(ctx context.Context, vehicleID uuid.UUID) error

	// Admin operations
	GetPendingReviewVehicles(ctx context.Context, limit, offset int) ([]Vehicle, int, error)
	GetVehicleStats(ctx context.Context) (*VehicleStats, error)
	GetExpiringVehicles(ctx context.Context, daysAhead int) ([]Vehicle, error)

	// Inspections
	CreateInspection(ctx context.Context, insp *VehicleInspection) error
	GetInspectionsByVehicle(ctx context.Context, vehicleID uuid.UUID) ([]VehicleInspection, error)

	// Maintenance reminders
	CreateMaintenanceReminder(ctx context.Context, rem *MaintenanceReminder) error
	GetPendingReminders(ctx context.Context, vehicleID uuid.UUID) ([]MaintenanceReminder, error)
	CompleteReminder(ctx context.Context, reminderID uuid.UUID) error
}
