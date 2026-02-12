package admin

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// RepositoryInterface defines the contract for admin repository operations
type RepositoryInterface interface {
	// User operations
	GetAllUsers(ctx context.Context, limit, offset int, filter *UserFilter) ([]*models.User, int, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	UpdateUserStatus(ctx context.Context, id uuid.UUID, isActive bool) error
	GetUserStats(ctx context.Context) (*UserStats, error)

	// Driver operations
	GetAllDriversWithTotal(ctx context.Context, limit, offset int, filter *DriverFilter) ([]*models.Driver, int64, error)
	GetPendingDrivers(ctx context.Context) ([]*models.Driver, error)
	GetPendingDriversWithTotal(ctx context.Context, limit, offset int) ([]*models.Driver, int64, error)
	GetDriverByID(ctx context.Context, driverID uuid.UUID) (*models.Driver, error)
	GetDriverStats(ctx context.Context) (*DriverStats, error)
	ApproveDriver(ctx context.Context, driverID uuid.UUID) error
	RejectDriver(ctx context.Context, driverID uuid.UUID) error

	// Ride operations
	GetRideByID(ctx context.Context, rideID uuid.UUID) (*AdminRideDetail, error)
	GetRecentRides(ctx context.Context, limit int) ([]*models.Ride, error)
	GetRecentRidesWithTotal(ctx context.Context, limit, offset int) ([]*models.Ride, int64, error)
	GetRecentRidesWithDetails(ctx context.Context, limit, offset int) ([]*AdminRideDetail, int64, error)
	GetRideStats(ctx context.Context, startDate, endDate *time.Time) (*RideStats, error)

	// Dashboard operations
	GetRealtimeMetrics(ctx context.Context) (*RealtimeMetrics, error)
	GetDashboardSummary(ctx context.Context, period string) (*DashboardSummary, error)
	GetRevenueTrend(ctx context.Context, period, groupBy string) (*RevenueTrend, error)
	GetActionItems(ctx context.Context) (*ActionItems, error)

	// Audit logging
	InsertAuditLog(ctx context.Context, adminID uuid.UUID, action, targetType string, targetID uuid.UUID, metadata map[string]interface{})
	GetAuditLogs(ctx context.Context, limit, offset int, filter *AuditLogFilter) ([]*AuditLog, int64, error)
}
