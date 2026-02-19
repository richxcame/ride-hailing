package onboarding

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryInterface defines all public methods of the Repository
type RepositoryInterface interface {
	// Driver operations
	GetDriver(ctx context.Context, driverID uuid.UUID) (*DriverInfo, error)
	GetDriverByUserID(ctx context.Context, userID uuid.UUID) (*DriverInfo, error)
	CreateDriver(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)

	// Background check operations
	GetBackgroundCheck(ctx context.Context, driverID uuid.UUID) (*BackgroundCheck, error)
	CreateBackgroundCheck(ctx context.Context, driverID uuid.UUID, provider string) (*BackgroundCheck, error)
	UpdateBackgroundCheckStatus(ctx context.Context, checkID uuid.UUID, status string, notes *string) error

	// Driver approval operations
	ApproveDriver(ctx context.Context, driverID uuid.UUID, approvedBy uuid.UUID) error
	RejectDriver(ctx context.Context, driverID uuid.UUID, rejectedBy uuid.UUID, reason string) error

	// Statistics
	GetOnboardingStats(ctx context.Context) (*OnboardingStats, error)

	// Vehicle check — vehicles.driver_id is a FK to users.id, so we query by user_id
	HasApprovedVehicle(ctx context.Context, userID uuid.UUID) (bool, error)
}
