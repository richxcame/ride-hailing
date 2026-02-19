package vehicle

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/richxcame/ride-hailing/pkg/common"
)

const (
	maxVehiclesPerDriver = 3
	minVehicleYear       = 2010
)

// Service handles vehicle business logic
type Service struct {
	repo RepositoryInterface
}

// NewService creates a new vehicle service
func NewService(repo RepositoryInterface) *Service {
	return &Service{repo: repo}
}

// ========================================
// VEHICLE MANAGEMENT
// ========================================

// RegisterVehicle registers a new vehicle for a driver
func (s *Service) RegisterVehicle(ctx context.Context, driverID uuid.UUID, req *RegisterVehicleRequest) (*Vehicle, error) {
	// Validate year
	currentYear := time.Now().Year()
	if req.Year < minVehicleYear || req.Year > currentYear+1 {
		return nil, common.NewBadRequestError(
			fmt.Sprintf("vehicle year must be between %d and %d", minVehicleYear, currentYear+1), nil,
		)
	}

	// Check vehicle count (fetch all without pagination for the limit check)
	existing, _, err := s.repo.GetVehiclesByDriver(ctx, driverID, maxVehiclesPerDriver+1, 0)
	if err != nil {
		return nil, err
	}
	activeCount := 0
	for _, v := range existing {
		if v.IsActive {
			activeCount++
		}
	}
	if activeCount >= maxVehiclesPerDriver {
		return nil, common.NewBadRequestError(
			fmt.Sprintf("maximum %d active vehicles allowed", maxVehiclesPerDriver), nil,
		)
	}

	maxPassengers := req.MaxPassengers
	if maxPassengers <= 0 {
		maxPassengers = 4
	}

	now := time.Now()
	isFirst := activeCount == 0

	vehicle := &Vehicle{
		ID:                  uuid.New(),
		DriverID:            driverID,
		Status:              VehicleStatusPending,
		Category:            req.Category,
		Make:                req.Make,
		Model:               req.Model,
		Year:                req.Year,
		Color:               req.Color,
		LicensePlate:        req.LicensePlate,
		VIN:                 req.VIN,
		FuelType:            req.FuelType,
		MaxPassengers:       maxPassengers,
		HasChildSeat:        req.HasChildSeat,
		HasWheelchairAccess: req.HasWheelchairAccess,
		HasWifi:             req.HasWifi,
		HasCharger:          req.HasCharger,
		PetFriendly:         req.PetFriendly,
		LuggageCapacity:     req.LuggageCapacity,
		IsActive:            true,
		IsPrimary:           isFirst, // First vehicle is automatically primary
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := s.repo.CreateVehicle(ctx, vehicle); err != nil {
		return nil, fmt.Errorf("create vehicle: %w", err)
	}

	return vehicle, nil
}

// GetMyVehicles returns vehicles for the driver with total count for pagination
func (s *Service) GetMyVehicles(ctx context.Context, driverID uuid.UUID, limit, offset int) (*VehicleListResponse, int64, error) {
	vehicles, total, err := s.repo.GetVehiclesByDriver(ctx, driverID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return &VehicleListResponse{
		Vehicles: vehicles,
	}, total, nil
}

// GetVehicle returns a specific vehicle
func (s *Service) GetVehicle(ctx context.Context, vehicleID uuid.UUID, driverID uuid.UUID) (*Vehicle, error) {
	vehicle, err := s.repo.GetVehicleByID(ctx, vehicleID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("vehicle not found", nil)
		}
		return nil, err
	}

	if vehicle.DriverID != driverID {
		return nil, common.NewForbiddenError("not your vehicle")
	}

	return vehicle, nil
}

// GetPrimaryVehicle returns the driver's primary vehicle
func (s *Service) GetPrimaryVehicle(ctx context.Context, driverID uuid.UUID) (*Vehicle, error) {
	vehicle, err := s.repo.GetPrimaryVehicle(ctx, driverID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("no primary vehicle set", nil)
		}
		return nil, err
	}
	return vehicle, nil
}

// UpdateVehicle updates vehicle details
func (s *Service) UpdateVehicle(ctx context.Context, vehicleID uuid.UUID, driverID uuid.UUID, req *UpdateVehicleRequest) (*Vehicle, error) {
	vehicle, err := s.repo.GetVehicleByID(ctx, vehicleID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("vehicle not found", nil)
		}
		return nil, err
	}

	if vehicle.DriverID != driverID {
		return nil, common.NewForbiddenError("not your vehicle")
	}

	if req.Color != nil {
		vehicle.Color = *req.Color
	}
	if req.Category != nil {
		vehicle.Category = *req.Category
	}
	if req.MaxPassengers != nil {
		vehicle.MaxPassengers = *req.MaxPassengers
	}
	if req.HasChildSeat != nil {
		vehicle.HasChildSeat = *req.HasChildSeat
	}
	if req.HasWheelchairAccess != nil {
		vehicle.HasWheelchairAccess = *req.HasWheelchairAccess
	}
	if req.HasWifi != nil {
		vehicle.HasWifi = *req.HasWifi
	}
	if req.HasCharger != nil {
		vehicle.HasCharger = *req.HasCharger
	}
	if req.PetFriendly != nil {
		vehicle.PetFriendly = *req.PetFriendly
	}
	if req.LuggageCapacity != nil {
		vehicle.LuggageCapacity = *req.LuggageCapacity
	}
	vehicle.UpdatedAt = time.Now()

	if err := s.repo.UpdateVehicle(ctx, vehicle); err != nil {
		return nil, fmt.Errorf("update vehicle: %w", err)
	}

	return vehicle, nil
}

// UploadPhotos uploads vehicle photos
func (s *Service) UploadPhotos(ctx context.Context, vehicleID uuid.UUID, driverID uuid.UUID, req *UploadVehiclePhotosRequest) error {
	vehicle, err := s.repo.GetVehicleByID(ctx, vehicleID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return common.NewNotFoundError("vehicle not found", nil)
		}
		return err
	}

	if vehicle.DriverID != driverID {
		return common.NewForbiddenError("not your vehicle")
	}

	return s.repo.UpdateVehiclePhotos(ctx, vehicleID, req)
}

// SetPrimary sets a vehicle as the driver's primary
func (s *Service) SetPrimary(ctx context.Context, vehicleID uuid.UUID, driverID uuid.UUID) error {
	vehicle, err := s.repo.GetVehicleByID(ctx, vehicleID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return common.NewNotFoundError("vehicle not found", nil)
		}
		return err
	}

	if vehicle.DriverID != driverID {
		return common.NewForbiddenError("not your vehicle")
	}

	if vehicle.Status != VehicleStatusApproved {
		return common.NewBadRequestError("only approved vehicles can be set as primary", nil)
	}

	return s.repo.SetPrimaryVehicle(ctx, driverID, vehicleID)
}

// RetireVehicle retires a vehicle
func (s *Service) RetireVehicle(ctx context.Context, vehicleID uuid.UUID, driverID uuid.UUID) error {
	vehicle, err := s.repo.GetVehicleByID(ctx, vehicleID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return common.NewNotFoundError("vehicle not found", nil)
		}
		return err
	}

	if vehicle.DriverID != driverID {
		return common.NewForbiddenError("not your vehicle")
	}

	return s.repo.RetireVehicle(ctx, vehicleID)
}

// ========================================
// ADMIN
// ========================================

// GetAllVehicles returns all vehicles for admin with filters and embedded driver details.
func (s *Service) GetAllVehicles(ctx context.Context, filter *AdminVehicleFilter, limit, offset int) ([]VehicleWithDriver, int64, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.GetAllVehicles(ctx, filter, limit, offset)
}

// ReviewVehicle approves or rejects a vehicle (admin)
func (s *Service) ReviewVehicle(ctx context.Context, vehicleID uuid.UUID, req *AdminReviewRequest) error {
	vehicle, err := s.repo.GetVehicleByID(ctx, vehicleID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return common.NewNotFoundError("vehicle not found", nil)
		}
		return err
	}

	if vehicle.Status != VehicleStatusPending {
		return common.NewBadRequestError("vehicle is not pending review", nil)
	}

	if req.Approved {
		return s.repo.UpdateVehicleStatus(ctx, vehicleID, VehicleStatusApproved, nil)
	}

	if req.RejectionReason == nil || *req.RejectionReason == "" {
		return common.NewBadRequestError("rejection reason is required", nil)
	}

	return s.repo.UpdateVehicleStatus(ctx, vehicleID, VehicleStatusRejected, req.RejectionReason)
}

// GetPendingReviews returns vehicles awaiting review with embedded driver details.
func (s *Service) GetPendingReviews(ctx context.Context, limit, offset int) ([]VehicleWithDriver, int, error) {
	if limit < 1 || limit > 50 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.GetPendingReviewVehicles(ctx, limit, offset)
}

// GetVehicleStats returns vehicle stats for admin
func (s *Service) GetVehicleStats(ctx context.Context) (*VehicleStats, error) {
	return s.repo.GetVehicleStats(ctx)
}

// SuspendVehicle suspends a vehicle (admin)
func (s *Service) SuspendVehicle(ctx context.Context, vehicleID uuid.UUID, reason string) error {
	return s.repo.UpdateVehicleStatus(ctx, vehicleID, VehicleStatusSuspended, &reason)
}

// GetExpiringDocuments returns vehicles with expiring documents
func (s *Service) GetExpiringDocuments(ctx context.Context, daysAhead int) ([]Vehicle, error) {
	if daysAhead <= 0 {
		daysAhead = 30
	}
	vehicles, err := s.repo.GetExpiringVehicles(ctx, daysAhead)
	if err != nil {
		return nil, err
	}
	if vehicles == nil {
		vehicles = []Vehicle{}
	}
	return vehicles, nil
}

// ========================================
// MAINTENANCE
// ========================================

// GetMaintenanceReminders returns pending reminders for a vehicle
func (s *Service) GetMaintenanceReminders(ctx context.Context, vehicleID uuid.UUID, driverID uuid.UUID) ([]MaintenanceReminder, error) {
	vehicle, err := s.repo.GetVehicleByID(ctx, vehicleID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("vehicle not found", nil)
		}
		return nil, err
	}
	if vehicle.DriverID != driverID {
		return nil, common.NewForbiddenError("not your vehicle")
	}

	reminders, err := s.repo.GetPendingReminders(ctx, vehicleID)
	if err != nil {
		return nil, err
	}
	if reminders == nil {
		reminders = []MaintenanceReminder{}
	}
	return reminders, nil
}

// CompleteMaintenance marks a maintenance reminder as done
func (s *Service) CompleteMaintenance(ctx context.Context, reminderID uuid.UUID) error {
	return s.repo.CompleteReminder(ctx, reminderID)
}

// GetInspections returns inspection history for a vehicle
func (s *Service) GetInspections(ctx context.Context, vehicleID uuid.UUID, driverID uuid.UUID) ([]VehicleInspection, error) {
	vehicle, err := s.repo.GetVehicleByID(ctx, vehicleID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, common.NewNotFoundError("vehicle not found", nil)
		}
		return nil, err
	}
	if vehicle.DriverID != driverID {
		return nil, common.NewForbiddenError("not your vehicle")
	}

	inspections, err := s.repo.GetInspectionsByVehicle(ctx, vehicleID)
	if err != nil {
		return nil, err
	}
	if inspections == nil {
		inspections = []VehicleInspection{}
	}
	return inspections, nil
}
