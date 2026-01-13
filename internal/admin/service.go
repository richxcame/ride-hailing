package admin

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// Service handles business logic for admin operations
type Service struct {
	repo *Repository
}

// NewService creates a new admin service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetAllUsers retrieves all users with pagination
func (s *Service) GetAllUsers(ctx context.Context, limit, offset int) ([]*models.User, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	return s.repo.GetAllUsers(ctx, limit, offset)
}

// GetUser retrieves a specific user
func (s *Service) GetUser(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	return s.repo.GetUserByID(ctx, userID)
}

// SuspendUser suspends a user account
func (s *Service) SuspendUser(ctx context.Context, userID uuid.UUID) error {
	return s.repo.UpdateUserStatus(ctx, userID, false)
}

// ActivateUser activates a user account
func (s *Service) ActivateUser(ctx context.Context, userID uuid.UUID) error {
	return s.repo.UpdateUserStatus(ctx, userID, true)
}

// GetAllDrivers retrieves all drivers with pagination
func (s *Service) GetAllDrivers(ctx context.Context, limit, offset int) ([]*models.Driver, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	return s.repo.GetAllDriversWithTotal(ctx, limit, offset)
}

// GetPendingDrivers retrieves drivers awaiting approval
func (s *Service) GetPendingDrivers(ctx context.Context, limit, offset int) ([]*models.Driver, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	return s.repo.GetPendingDriversWithTotal(ctx, limit, offset)
}

// ApproveDriver approves a driver application
func (s *Service) ApproveDriver(ctx context.Context, driverID uuid.UUID) error {
	return s.repo.ApproveDriver(ctx, driverID)
}

// RejectDriver rejects a driver application
func (s *Service) RejectDriver(ctx context.Context, driverID uuid.UUID) error {
	return s.repo.RejectDriver(ctx, driverID)
}

// GetDashboardStats retrieves overall dashboard statistics
func (s *Service) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	// Get user stats
	userStats, err := s.repo.GetUserStats(ctx)
	if err != nil {
		return nil, err
	}

	// Get ride stats (all time)
	rideStats, err := s.repo.GetRideStats(ctx, nil, nil)
	if err != nil {
		return nil, err
	}

	// Get today's stats
	today := time.Now().Truncate(24 * time.Hour)
	todayStats, err := s.repo.GetRideStats(ctx, &today, nil)
	if err != nil {
		return nil, err
	}

	return &DashboardStats{
		Users:      userStats,
		Rides:      rideStats,
		TodayRides: todayStats,
	}, nil
}

// GetRideStats retrieves ride statistics for a date range
func (s *Service) GetRideStats(ctx context.Context, startDate, endDate *time.Time) (*RideStats, error) {
	return s.repo.GetRideStats(ctx, startDate, endDate)
}

// GetRecentRides retrieves recent rides for monitoring
func (s *Service) GetRecentRides(ctx context.Context, limit, offset int) ([]*models.Ride, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	return s.repo.GetRecentRidesWithTotal(ctx, limit, offset)
}

// DashboardStats represents overall dashboard statistics
type DashboardStats struct {
	Users      *UserStats `json:"users"`
	Rides      *RideStats `json:"rides"`
	TodayRides *RideStats `json:"today_rides"`
}
