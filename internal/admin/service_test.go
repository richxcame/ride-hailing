package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ========================================
// INTERNAL MOCK (implements RepositoryInterface within this package)
// ========================================

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) GetAllUsers(ctx context.Context, limit, offset int, filter *UserFilter) ([]*models.User, int, error) {
	args := m.Called(ctx, limit, offset, filter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.User), args.Int(1), args.Error(2)
}

func (m *mockRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *mockRepo) UpdateUserStatus(ctx context.Context, id uuid.UUID, isActive bool) error {
	args := m.Called(ctx, id, isActive)
	return args.Error(0)
}

func (m *mockRepo) GetUserStats(ctx context.Context) (*UserStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UserStats), args.Error(1)
}

func (m *mockRepo) GetAllDriversWithTotal(ctx context.Context, limit, offset int, filter *DriverFilter) ([]*models.Driver, int64, error) {
	args := m.Called(ctx, limit, offset, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*models.Driver), args.Get(1).(int64), args.Error(2)
}

func (m *mockRepo) GetPendingDrivers(ctx context.Context) ([]*models.Driver, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Driver), args.Error(1)
}

func (m *mockRepo) GetPendingDriversWithTotal(ctx context.Context, limit, offset int) ([]*models.Driver, int64, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*models.Driver), args.Get(1).(int64), args.Error(2)
}

func (m *mockRepo) GetDriverByID(ctx context.Context, driverID uuid.UUID) (*models.Driver, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Driver), args.Error(1)
}

func (m *mockRepo) GetDriverStats(ctx context.Context) (*DriverStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverStats), args.Error(1)
}

func (m *mockRepo) ApproveDriver(ctx context.Context, driverID uuid.UUID) error {
	args := m.Called(ctx, driverID)
	return args.Error(0)
}

func (m *mockRepo) RejectDriver(ctx context.Context, driverID uuid.UUID) error {
	args := m.Called(ctx, driverID)
	return args.Error(0)
}

func (m *mockRepo) GetRideByID(ctx context.Context, rideID uuid.UUID) (*AdminRideDetail, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AdminRideDetail), args.Error(1)
}

func (m *mockRepo) GetRecentRides(ctx context.Context, limit int) ([]*models.Ride, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Ride), args.Error(1)
}

func (m *mockRepo) GetRecentRidesWithTotal(ctx context.Context, limit, offset int) ([]*models.Ride, int64, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*models.Ride), args.Get(1).(int64), args.Error(2)
}

func (m *mockRepo) GetRecentRidesWithDetails(ctx context.Context, limit, offset int) ([]*AdminRideDetail, int64, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*AdminRideDetail), args.Get(1).(int64), args.Error(2)
}

func (m *mockRepo) GetRideStats(ctx context.Context, startDate, endDate *time.Time) (*RideStats, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RideStats), args.Error(1)
}

func (m *mockRepo) GetRealtimeMetrics(ctx context.Context) (*RealtimeMetrics, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RealtimeMetrics), args.Error(1)
}

func (m *mockRepo) GetDashboardSummary(ctx context.Context, period string) (*DashboardSummary, error) {
	args := m.Called(ctx, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DashboardSummary), args.Error(1)
}

func (m *mockRepo) GetRevenueTrend(ctx context.Context, period, groupBy string) (*RevenueTrend, error) {
	args := m.Called(ctx, period, groupBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RevenueTrend), args.Error(1)
}

func (m *mockRepo) GetActionItems(ctx context.Context) (*ActionItems, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ActionItems), args.Error(1)
}

func (m *mockRepo) InsertAuditLog(ctx context.Context, adminID uuid.UUID, action, targetType string, targetID uuid.UUID, metadata map[string]interface{}) {
	m.Called(ctx, adminID, action, targetType, targetID, metadata)
}

func (m *mockRepo) GetAuditLogs(ctx context.Context, limit, offset int, filter *AuditLogFilter) ([]*AuditLog, int64, error) {
	args := m.Called(ctx, limit, offset, filter)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*AuditLog), args.Get(1).(int64), args.Error(2)
}

// ========================================
// TEST HELPERS
// ========================================

func newTestService(repo RepositoryInterface) *Service {
	return NewService(repo)
}

// ========================================
// TESTS: SuspendUser / ActivateUser
// ========================================

func TestSuspendUser(t *testing.T) {
	adminID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
	}{
		{
			name: "success - user suspended with audit log",
			setupMocks: func(m *mockRepo) {
				m.On("UpdateUserStatus", mock.Anything, userID, false).Return(nil)
				m.On("InsertAuditLog", mock.Anything, adminID, "suspend_user", "user", userID, (map[string]interface{})(nil)).Return()
			},
			wantErr: false,
		},
		{
			name: "error - user not found",
			setupMocks: func(m *mockRepo) {
				m.On("UpdateUserStatus", mock.Anything, userID, false).Return(errors.New("user not found"))
			},
			wantErr:    true,
			errContain: "user not found",
		},
		{
			name: "error - database error",
			setupMocks: func(m *mockRepo) {
				m.On("UpdateUserStatus", mock.Anything, userID, false).Return(errors.New("database connection failed"))
			},
			wantErr:    true,
			errContain: "database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.SuspendUser(context.Background(), adminID, userID)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

func TestActivateUser(t *testing.T) {
	adminID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
	}{
		{
			name: "success - user activated with audit log",
			setupMocks: func(m *mockRepo) {
				m.On("UpdateUserStatus", mock.Anything, userID, true).Return(nil)
				m.On("InsertAuditLog", mock.Anything, adminID, "activate_user", "user", userID, (map[string]interface{})(nil)).Return()
			},
			wantErr: false,
		},
		{
			name: "error - user not found",
			setupMocks: func(m *mockRepo) {
				m.On("UpdateUserStatus", mock.Anything, userID, true).Return(errors.New("user not found"))
			},
			wantErr:    true,
			errContain: "user not found",
		},
		{
			name: "error - database error",
			setupMocks: func(m *mockRepo) {
				m.On("UpdateUserStatus", mock.Anything, userID, true).Return(errors.New("database timeout"))
			},
			wantErr:    true,
			errContain: "database timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.ActivateUser(context.Background(), adminID, userID)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: ApproveDriver / RejectDriver
// ========================================

func TestApproveDriver(t *testing.T) {
	adminID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
	}{
		{
			name: "success - driver approved with audit log",
			setupMocks: func(m *mockRepo) {
				m.On("ApproveDriver", mock.Anything, driverID).Return(nil)
				m.On("InsertAuditLog", mock.Anything, adminID, "approve_driver", "driver", driverID, (map[string]interface{})(nil)).Return()
			},
			wantErr: false,
		},
		{
			name: "error - driver not found",
			setupMocks: func(m *mockRepo) {
				m.On("ApproveDriver", mock.Anything, driverID).Return(errors.New("driver not found"))
			},
			wantErr:    true,
			errContain: "driver not found",
		},
		{
			name: "error - database error during approval",
			setupMocks: func(m *mockRepo) {
				m.On("ApproveDriver", mock.Anything, driverID).Return(errors.New("failed to approve driver"))
			},
			wantErr:    true,
			errContain: "failed to approve driver",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.ApproveDriver(context.Background(), adminID, driverID)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

func TestRejectDriver(t *testing.T) {
	adminID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
	}{
		{
			name: "success - driver rejected with audit log",
			setupMocks: func(m *mockRepo) {
				m.On("RejectDriver", mock.Anything, driverID).Return(nil)
				m.On("InsertAuditLog", mock.Anything, adminID, "reject_driver", "driver", driverID, (map[string]interface{})(nil)).Return()
			},
			wantErr: false,
		},
		{
			name: "error - driver not found",
			setupMocks: func(m *mockRepo) {
				m.On("RejectDriver", mock.Anything, driverID).Return(errors.New("driver not found"))
			},
			wantErr:    true,
			errContain: "driver not found",
		},
		{
			name: "error - database error during rejection",
			setupMocks: func(m *mockRepo) {
				m.On("RejectDriver", mock.Anything, driverID).Return(errors.New("failed to reject driver"))
			},
			wantErr:    true,
			errContain: "failed to reject driver",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.RejectDriver(context.Background(), adminID, driverID)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetDashboardStats
// ========================================

func TestGetDashboardStats(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
		validate   func(t *testing.T, stats *DashboardStats)
	}{
		{
			name: "success - full dashboard stats aggregation",
			setupMocks: func(m *mockRepo) {
				userStats := &UserStats{
					TotalUsers:   1000,
					TotalRiders:  800,
					TotalDrivers: 150,
					TotalAdmins:  50,
					ActiveUsers:  900,
				}
				allTimeRideStats := &RideStats{
					TotalRides:     5000,
					CompletedRides: 4500,
					CancelledRides: 300,
					ActiveRides:    200,
					TotalRevenue:   250000.50,
					AvgFare:        55.56,
				}
				todayRideStats := &RideStats{
					TotalRides:     100,
					CompletedRides: 85,
					CancelledRides: 10,
					ActiveRides:    5,
					TotalRevenue:   5500.00,
					AvgFare:        64.71,
				}

				m.On("GetUserStats", mock.Anything).Return(userStats, nil)
				m.On("GetRideStats", mock.Anything, (*time.Time)(nil), (*time.Time)(nil)).Return(allTimeRideStats, nil)
				m.On("GetRideStats", mock.Anything, mock.AnythingOfType("*time.Time"), (*time.Time)(nil)).Return(todayRideStats, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, stats *DashboardStats) {
				require.NotNil(t, stats)
				require.NotNil(t, stats.Users)
				require.NotNil(t, stats.Rides)
				require.NotNil(t, stats.TodayRides)

				// Validate user stats
				assert.Equal(t, 1000, stats.Users.TotalUsers)
				assert.Equal(t, 800, stats.Users.TotalRiders)
				assert.Equal(t, 150, stats.Users.TotalDrivers)
				assert.Equal(t, 50, stats.Users.TotalAdmins)
				assert.Equal(t, 900, stats.Users.ActiveUsers)

				// Validate all-time ride stats
				assert.Equal(t, 5000, stats.Rides.TotalRides)
				assert.Equal(t, 4500, stats.Rides.CompletedRides)
				assert.Equal(t, 300, stats.Rides.CancelledRides)
				assert.Equal(t, 250000.50, stats.Rides.TotalRevenue)

				// Validate today's ride stats
				assert.Equal(t, 100, stats.TodayRides.TotalRides)
				assert.Equal(t, 85, stats.TodayRides.CompletedRides)
				assert.Equal(t, 5500.00, stats.TodayRides.TotalRevenue)
			},
		},
		{
			name: "error - failed to get user stats",
			setupMocks: func(m *mockRepo) {
				m.On("GetUserStats", mock.Anything).Return(nil, errors.New("failed to get user stats"))
			},
			wantErr:    true,
			errContain: "failed to get user stats",
		},
		{
			name: "error - failed to get all-time ride stats",
			setupMocks: func(m *mockRepo) {
				userStats := &UserStats{TotalUsers: 100}
				m.On("GetUserStats", mock.Anything).Return(userStats, nil)
				m.On("GetRideStats", mock.Anything, (*time.Time)(nil), (*time.Time)(nil)).Return(nil, errors.New("failed to get ride stats"))
			},
			wantErr:    true,
			errContain: "failed to get ride stats",
		},
		{
			name: "error - failed to get today's ride stats",
			setupMocks: func(m *mockRepo) {
				userStats := &UserStats{TotalUsers: 100}
				allTimeRideStats := &RideStats{TotalRides: 1000}
				m.On("GetUserStats", mock.Anything).Return(userStats, nil)
				m.On("GetRideStats", mock.Anything, (*time.Time)(nil), (*time.Time)(nil)).Return(allTimeRideStats, nil)
				m.On("GetRideStats", mock.Anything, mock.AnythingOfType("*time.Time"), (*time.Time)(nil)).Return(nil, errors.New("failed to get today stats"))
			},
			wantErr:    true,
			errContain: "failed to get today stats",
		},
		{
			name: "success - empty stats (no data)",
			setupMocks: func(m *mockRepo) {
				userStats := &UserStats{
					TotalUsers:   0,
					TotalRiders:  0,
					TotalDrivers: 0,
					TotalAdmins:  0,
					ActiveUsers:  0,
				}
				emptyRideStats := &RideStats{
					TotalRides:     0,
					CompletedRides: 0,
					CancelledRides: 0,
					ActiveRides:    0,
					TotalRevenue:   0,
					AvgFare:        0,
				}

				m.On("GetUserStats", mock.Anything).Return(userStats, nil)
				m.On("GetRideStats", mock.Anything, (*time.Time)(nil), (*time.Time)(nil)).Return(emptyRideStats, nil)
				m.On("GetRideStats", mock.Anything, mock.AnythingOfType("*time.Time"), (*time.Time)(nil)).Return(emptyRideStats, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, stats *DashboardStats) {
				require.NotNil(t, stats)
				assert.Equal(t, 0, stats.Users.TotalUsers)
				assert.Equal(t, 0, stats.Rides.TotalRides)
				assert.Equal(t, 0, stats.TodayRides.TotalRides)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			stats, err := svc.GetDashboardStats(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
				assert.Nil(t, stats)
			} else {
				require.NoError(t, err)
				require.NotNil(t, stats)
				if tt.validate != nil {
					tt.validate(t, stats)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetAllUsers (with pagination and filtering)
// ========================================

func TestGetAllUsers(t *testing.T) {
	userID1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name       string
		limit      int
		offset     int
		filter     *UserFilter
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, users []*models.User, total int)
	}{
		{
			name:   "success - returns users with pagination",
			limit:  10,
			offset: 0,
			filter: nil,
			setupMocks: func(m *mockRepo) {
				users := []*models.User{
					{ID: userID1, Email: "user1@test.com", Role: "rider", IsActive: true},
					{ID: userID2, Email: "user2@test.com", Role: "driver", IsActive: true},
				}
				m.On("GetAllUsers", mock.Anything, 10, 0, (*UserFilter)(nil)).Return(users, 2, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, users []*models.User, total int) {
				assert.Len(t, users, 2)
				assert.Equal(t, 2, total)
				assert.Equal(t, userID1, users[0].ID)
				assert.Equal(t, userID2, users[1].ID)
			},
		},
		{
			name:   "success - limits default to 20 when invalid",
			limit:  0,
			offset: 0,
			filter: nil,
			setupMocks: func(m *mockRepo) {
				users := []*models.User{}
				m.On("GetAllUsers", mock.Anything, 20, 0, (*UserFilter)(nil)).Return(users, 0, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, users []*models.User, total int) {
				assert.Len(t, users, 0)
				assert.Equal(t, 0, total)
			},
		},
		{
			name:   "success - limits capped at 100",
			limit:  500,
			offset: 0,
			filter: nil,
			setupMocks: func(m *mockRepo) {
				users := []*models.User{}
				m.On("GetAllUsers", mock.Anything, 20, 0, (*UserFilter)(nil)).Return(users, 0, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, users []*models.User, total int) {
				assert.Len(t, users, 0)
			},
		},
		{
			name:   "success - filters by role",
			limit:  20,
			offset: 0,
			filter: &UserFilter{Role: "driver"},
			setupMocks: func(m *mockRepo) {
				users := []*models.User{
					{ID: userID2, Email: "driver@test.com", Role: models.RoleDriver, IsActive: true},
				}
				m.On("GetAllUsers", mock.Anything, 20, 0, mock.AnythingOfType("*admin.UserFilter")).Return(users, 1, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, users []*models.User, total int) {
				assert.Len(t, users, 1)
				assert.Equal(t, 1, total)
				assert.Equal(t, models.RoleDriver, users[0].Role)
			},
		},
		{
			name:   "success - ignores invalid role filter",
			limit:  20,
			offset: 0,
			filter: &UserFilter{Role: "invalid_role"},
			setupMocks: func(m *mockRepo) {
				users := []*models.User{}
				// After validation, invalid role is cleared
				m.On("GetAllUsers", mock.Anything, 20, 0, mock.AnythingOfType("*admin.UserFilter")).Return(users, 0, nil)
			},
			wantErr: false,
		},
		{
			name:   "success - negative offset defaults to 0",
			limit:  20,
			offset: -5,
			filter: nil,
			setupMocks: func(m *mockRepo) {
				users := []*models.User{}
				m.On("GetAllUsers", mock.Anything, 20, 0, (*UserFilter)(nil)).Return(users, 0, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			users, total, err := svc.GetAllUsers(context.Background(), tt.limit, tt.offset, tt.filter)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, users, total)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetAllDrivers (with pagination and filtering)
// ========================================

func TestGetAllDrivers(t *testing.T) {
	driverID1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name       string
		limit      int
		offset     int
		filter     *DriverFilter
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, drivers []*models.Driver, total int64)
	}{
		{
			name:   "success - returns drivers with pagination",
			limit:  10,
			offset: 0,
			filter: nil,
			setupMocks: func(m *mockRepo) {
				drivers := []*models.Driver{
					{ID: driverID1, LicenseNumber: "LIC001", IsAvailable: true, IsOnline: true},
					{ID: driverID2, LicenseNumber: "LIC002", IsAvailable: false, IsOnline: false},
				}
				m.On("GetAllDriversWithTotal", mock.Anything, 10, 0, (*DriverFilter)(nil)).Return(drivers, int64(2), nil)
			},
			wantErr: false,
			validate: func(t *testing.T, drivers []*models.Driver, total int64) {
				assert.Len(t, drivers, 2)
				assert.Equal(t, int64(2), total)
			},
		},
		{
			name:   "success - filters by status online",
			limit:  20,
			offset: 0,
			filter: &DriverFilter{Status: "online"},
			setupMocks: func(m *mockRepo) {
				drivers := []*models.Driver{
					{ID: driverID1, LicenseNumber: "LIC001", IsAvailable: true, IsOnline: true},
				}
				m.On("GetAllDriversWithTotal", mock.Anything, 20, 0, mock.AnythingOfType("*admin.DriverFilter")).Return(drivers, int64(1), nil)
			},
			wantErr: false,
			validate: func(t *testing.T, drivers []*models.Driver, total int64) {
				assert.Len(t, drivers, 1)
				assert.True(t, drivers[0].IsOnline)
			},
		},
		{
			name:   "success - filters by status pending",
			limit:  20,
			offset: 0,
			filter: &DriverFilter{Status: "pending"},
			setupMocks: func(m *mockRepo) {
				drivers := []*models.Driver{
					{ID: driverID2, LicenseNumber: "LIC002", IsAvailable: false, IsOnline: false},
				}
				m.On("GetAllDriversWithTotal", mock.Anything, 20, 0, mock.AnythingOfType("*admin.DriverFilter")).Return(drivers, int64(1), nil)
			},
			wantErr: false,
			validate: func(t *testing.T, drivers []*models.Driver, total int64) {
				assert.Len(t, drivers, 1)
				assert.False(t, drivers[0].IsAvailable)
			},
		},
		{
			name:   "success - ignores invalid status filter",
			limit:  20,
			offset: 0,
			filter: &DriverFilter{Status: "invalid_status"},
			setupMocks: func(m *mockRepo) {
				drivers := []*models.Driver{}
				// After validation, invalid status is cleared
				m.On("GetAllDriversWithTotal", mock.Anything, 20, 0, mock.AnythingOfType("*admin.DriverFilter")).Return(drivers, int64(0), nil)
			},
			wantErr: false,
		},
		{
			name:   "success - default limit when invalid",
			limit:  -10,
			offset: 0,
			filter: nil,
			setupMocks: func(m *mockRepo) {
				drivers := []*models.Driver{}
				m.On("GetAllDriversWithTotal", mock.Anything, 20, 0, (*DriverFilter)(nil)).Return(drivers, int64(0), nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			drivers, total, err := svc.GetAllDrivers(context.Background(), tt.limit, tt.offset, tt.filter)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, drivers, total)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetPendingDrivers
// ========================================

func TestGetPendingDrivers(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name       string
		limit      int
		offset     int
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, drivers []*models.Driver, total int64)
	}{
		{
			name:   "success - returns pending drivers",
			limit:  10,
			offset: 0,
			setupMocks: func(m *mockRepo) {
				drivers := []*models.Driver{
					{ID: driverID, LicenseNumber: "LIC001", IsAvailable: false},
				}
				m.On("GetPendingDriversWithTotal", mock.Anything, 10, 0).Return(drivers, int64(1), nil)
			},
			wantErr: false,
			validate: func(t *testing.T, drivers []*models.Driver, total int64) {
				assert.Len(t, drivers, 1)
				assert.Equal(t, int64(1), total)
				assert.False(t, drivers[0].IsAvailable)
			},
		},
		{
			name:   "success - empty list when no pending drivers",
			limit:  20,
			offset: 0,
			setupMocks: func(m *mockRepo) {
				drivers := []*models.Driver{}
				m.On("GetPendingDriversWithTotal", mock.Anything, 20, 0).Return(drivers, int64(0), nil)
			},
			wantErr: false,
			validate: func(t *testing.T, drivers []*models.Driver, total int64) {
				assert.Len(t, drivers, 0)
				assert.Equal(t, int64(0), total)
			},
		},
		{
			name:   "success - default limit when invalid",
			limit:  0,
			offset: 0,
			setupMocks: func(m *mockRepo) {
				drivers := []*models.Driver{}
				m.On("GetPendingDriversWithTotal", mock.Anything, 20, 0).Return(drivers, int64(0), nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			drivers, total, err := svc.GetPendingDrivers(context.Background(), tt.limit, tt.offset)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, drivers, total)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetUser and GetDriver
// ========================================

func TestGetUser(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, user *models.User)
	}{
		{
			name: "success - returns user",
			setupMocks: func(m *mockRepo) {
				user := &models.User{
					ID:       userID,
					Email:    "test@example.com",
					Role:     models.RoleRider,
					IsActive: true,
				}
				m.On("GetUserByID", mock.Anything, userID).Return(user, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, user *models.User) {
				assert.Equal(t, userID, user.ID)
				assert.Equal(t, "test@example.com", user.Email)
				assert.Equal(t, models.RoleRider, user.Role)
			},
		},
		{
			name: "error - user not found",
			setupMocks: func(m *mockRepo) {
				m.On("GetUserByID", mock.Anything, userID).Return(nil, errors.New("user not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			user, err := svc.GetUser(context.Background(), userID)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				if tt.validate != nil {
					tt.validate(t, user)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

func TestGetDriver(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, driver *models.Driver)
	}{
		{
			name: "success - returns driver",
			setupMocks: func(m *mockRepo) {
				driver := &models.Driver{
					ID:            driverID,
					LicenseNumber: "LIC12345",
					VehicleModel:  "Toyota Camry",
					IsAvailable:   true,
					IsOnline:      true,
					Rating:        4.8,
				}
				m.On("GetDriverByID", mock.Anything, driverID).Return(driver, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, driver *models.Driver) {
				assert.Equal(t, driverID, driver.ID)
				assert.Equal(t, "LIC12345", driver.LicenseNumber)
				assert.Equal(t, "Toyota Camry", driver.VehicleModel)
				assert.Equal(t, 4.8, driver.Rating)
			},
		},
		{
			name: "error - driver not found",
			setupMocks: func(m *mockRepo) {
				m.On("GetDriverByID", mock.Anything, driverID).Return(nil, errors.New("driver not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			driver, err := svc.GetDriver(context.Background(), driverID)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, driver)
			} else {
				require.NoError(t, err)
				require.NotNil(t, driver)
				if tt.validate != nil {
					tt.validate(t, driver)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetUserStats and GetDriverStats
// ========================================

func TestGetUserStats(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, stats *UserStats)
	}{
		{
			name: "success - returns user stats",
			setupMocks: func(m *mockRepo) {
				stats := &UserStats{
					TotalUsers:   500,
					TotalRiders:  400,
					TotalDrivers: 80,
					TotalAdmins:  20,
					ActiveUsers:  450,
				}
				m.On("GetUserStats", mock.Anything).Return(stats, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, stats *UserStats) {
				assert.Equal(t, 500, stats.TotalUsers)
				assert.Equal(t, 400, stats.TotalRiders)
				assert.Equal(t, 80, stats.TotalDrivers)
				assert.Equal(t, 20, stats.TotalAdmins)
				assert.Equal(t, 450, stats.ActiveUsers)
			},
		},
		{
			name: "error - database failure",
			setupMocks: func(m *mockRepo) {
				m.On("GetUserStats", mock.Anything).Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			stats, err := svc.GetUserStats(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, stats)
			} else {
				require.NoError(t, err)
				require.NotNil(t, stats)
				if tt.validate != nil {
					tt.validate(t, stats)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

func TestGetDriverStats(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, stats *DriverStats)
	}{
		{
			name: "success - returns driver stats",
			setupMocks: func(m *mockRepo) {
				stats := &DriverStats{
					TotalDrivers:     100,
					OnlineDrivers:    40,
					OfflineDrivers:   60,
					AvailableDrivers: 25,
					PendingApprovals: 5,
				}
				m.On("GetDriverStats", mock.Anything).Return(stats, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, stats *DriverStats) {
				assert.Equal(t, 100, stats.TotalDrivers)
				assert.Equal(t, 40, stats.OnlineDrivers)
				assert.Equal(t, 60, stats.OfflineDrivers)
				assert.Equal(t, 25, stats.AvailableDrivers)
				assert.Equal(t, 5, stats.PendingApprovals)
			},
		},
		{
			name: "error - database failure",
			setupMocks: func(m *mockRepo) {
				m.On("GetDriverStats", mock.Anything).Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			stats, err := svc.GetDriverStats(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, stats)
			} else {
				require.NoError(t, err)
				require.NotNil(t, stats)
				if tt.validate != nil {
					tt.validate(t, stats)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetRideStats
// ========================================

func TestGetRideStats(t *testing.T) {
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC)

	tests := []struct {
		name       string
		startDate  *time.Time
		endDate    *time.Time
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, stats *RideStats)
	}{
		{
			name:      "success - returns ride stats for date range",
			startDate: &startDate,
			endDate:   &endDate,
			setupMocks: func(m *mockRepo) {
				stats := &RideStats{
					TotalRides:     1000,
					CompletedRides: 900,
					CancelledRides: 50,
					ActiveRides:    50,
					TotalRevenue:   45000.00,
					AvgFare:        50.00,
				}
				m.On("GetRideStats", mock.Anything, &startDate, &endDate).Return(stats, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, stats *RideStats) {
				assert.Equal(t, 1000, stats.TotalRides)
				assert.Equal(t, 900, stats.CompletedRides)
				assert.Equal(t, 50, stats.CancelledRides)
				assert.Equal(t, 45000.00, stats.TotalRevenue)
			},
		},
		{
			name:      "success - returns all-time stats when no date range",
			startDate: nil,
			endDate:   nil,
			setupMocks: func(m *mockRepo) {
				stats := &RideStats{
					TotalRides:     5000,
					CompletedRides: 4500,
					TotalRevenue:   225000.00,
				}
				m.On("GetRideStats", mock.Anything, (*time.Time)(nil), (*time.Time)(nil)).Return(stats, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, stats *RideStats) {
				assert.Equal(t, 5000, stats.TotalRides)
				assert.Equal(t, 4500, stats.CompletedRides)
				assert.Equal(t, 225000.00, stats.TotalRevenue)
			},
		},
		{
			name:      "error - database failure",
			startDate: &startDate,
			endDate:   &endDate,
			setupMocks: func(m *mockRepo) {
				m.On("GetRideStats", mock.Anything, &startDate, &endDate).Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			stats, err := svc.GetRideStats(context.Background(), tt.startDate, tt.endDate)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, stats)
			} else {
				require.NoError(t, err)
				require.NotNil(t, stats)
				if tt.validate != nil {
					tt.validate(t, stats)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetRecentRides
// ========================================

func TestGetRecentRides(t *testing.T) {
	rideID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	riderID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name       string
		limit      int
		offset     int
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, rides []*AdminRideDetail, total int64)
	}{
		{
			name:   "success - returns recent rides with details",
			limit:  10,
			offset: 0,
			setupMocks: func(m *mockRepo) {
				rides := []*AdminRideDetail{
					{
						Ride: &models.Ride{
							ID:       rideID,
							RiderID:  riderID,
							Status:   models.RideStatusCompleted,
							FinalFare: func() *float64 { f := 50.0; return &f }(),
						},
						Rider: &RideUserInfo{
							ID:        riderID,
							Email:     "rider@test.com",
							FirstName: "John",
							LastName:  "Doe",
						},
					},
				}
				m.On("GetRecentRidesWithDetails", mock.Anything, 10, 0).Return(rides, int64(1), nil)
			},
			wantErr: false,
			validate: func(t *testing.T, rides []*AdminRideDetail, total int64) {
				assert.Len(t, rides, 1)
				assert.Equal(t, int64(1), total)
				assert.Equal(t, models.RideStatusCompleted, rides[0].Ride.Status)
				assert.NotNil(t, rides[0].Rider)
				assert.Equal(t, "John", rides[0].Rider.FirstName)
			},
		},
		{
			name:   "success - default limit when invalid",
			limit:  0,
			offset: 0,
			setupMocks: func(m *mockRepo) {
				rides := []*AdminRideDetail{}
				m.On("GetRecentRidesWithDetails", mock.Anything, 50, 0).Return(rides, int64(0), nil)
			},
			wantErr: false,
		},
		{
			name:   "success - limit defaults to 50 when exceeds 100",
			limit:  200,
			offset: 0,
			setupMocks: func(m *mockRepo) {
				rides := []*AdminRideDetail{}
				m.On("GetRecentRidesWithDetails", mock.Anything, 50, 0).Return(rides, int64(0), nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			rides, total, err := svc.GetRecentRides(context.Background(), tt.limit, tt.offset)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, rides, total)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetRide
// ========================================

func TestGetRide(t *testing.T) {
	rideID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	riderID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	driverID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, ride *AdminRideDetail)
	}{
		{
			name: "success - returns ride with rider and driver details",
			setupMocks: func(m *mockRepo) {
				ride := &AdminRideDetail{
					Ride: &models.Ride{
						ID:       rideID,
						RiderID:  riderID,
						DriverID: &driverID,
						Status:   "completed",
					},
					Rider: &RideUserInfo{
						ID:        riderID,
						Email:     "rider@test.com",
						FirstName: "John",
						LastName:  "Doe",
					},
					Driver: &RideDriverInfo{
						ID:           driverID,
						FirstName:    "Jane",
						LastName:     "Smith",
						VehicleModel: "Toyota Camry",
						Rating:       4.9,
					},
				}
				m.On("GetRideByID", mock.Anything, rideID).Return(ride, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, ride *AdminRideDetail) {
				assert.Equal(t, rideID, ride.Ride.ID)
				assert.NotNil(t, ride.Rider)
				assert.NotNil(t, ride.Driver)
				assert.Equal(t, "John", ride.Rider.FirstName)
				assert.Equal(t, "Jane", ride.Driver.FirstName)
			},
		},
		{
			name: "error - ride not found",
			setupMocks: func(m *mockRepo) {
				m.On("GetRideByID", mock.Anything, rideID).Return(nil, errors.New("ride not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			ride, err := svc.GetRide(context.Background(), rideID)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, ride)
			} else {
				require.NoError(t, err)
				require.NotNil(t, ride)
				if tt.validate != nil {
					tt.validate(t, ride)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetRealtimeMetrics
// ========================================

func TestGetRealtimeMetrics(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, metrics *RealtimeMetrics)
	}{
		{
			name: "success - returns realtime metrics",
			setupMocks: func(m *mockRepo) {
				metrics := &RealtimeMetrics{
					ActiveRides:        15,
					AvailableDrivers:   30,
					PendingRequests:    5,
					TodayRevenue:       2500.00,
					TodayRevenueChange: 12.5,
					OnlineDrivers:      45,
					TotalRidersActive:  100,
					AvgWaitTime:        3.5,
					AvgETA:             8.2,
				}
				m.On("GetRealtimeMetrics", mock.Anything).Return(metrics, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, metrics *RealtimeMetrics) {
				assert.Equal(t, 15, metrics.ActiveRides)
				assert.Equal(t, 30, metrics.AvailableDrivers)
				assert.Equal(t, 5, metrics.PendingRequests)
				assert.Equal(t, 2500.00, metrics.TodayRevenue)
				assert.Equal(t, 12.5, metrics.TodayRevenueChange)
				assert.Equal(t, 45, metrics.OnlineDrivers)
			},
		},
		{
			name: "error - database failure",
			setupMocks: func(m *mockRepo) {
				m.On("GetRealtimeMetrics", mock.Anything).Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			metrics, err := svc.GetRealtimeMetrics(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, metrics)
			} else {
				require.NoError(t, err)
				require.NotNil(t, metrics)
				if tt.validate != nil {
					tt.validate(t, metrics)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetDashboardSummary
// ========================================

func TestGetDashboardSummary(t *testing.T) {
	tests := []struct {
		name       string
		period     string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, summary *DashboardSummary)
	}{
		{
			name:   "success - returns dashboard summary for today",
			period: "today",
			setupMocks: func(m *mockRepo) {
				summary := &DashboardSummary{
					Rides: &SummaryRides{
						Total:            50,
						Completed:        45,
						Cancelled:        3,
						InProgress:       2,
						CompletionRate:   90.0,
						CancellationRate: 6.0,
					},
					Drivers: &SummaryDrivers{
						TotalActive:      80,
						OnlineNow:        35,
						AvailableNow:     20,
						PendingApprovals: 3,
						AvgRating:        4.7,
					},
					Riders: &SummaryRiders{
						TotalActive: 500,
						NewSignups:  10,
						ActiveToday: 40,
					},
					Revenue: &SummaryRevenue{
						Total:      3500.00,
						Commission: 700.00,
						AvgFare:    77.78,
					},
					Alerts: &SummaryAlerts{
						FraudAlerts:    2,
						CriticalAlerts: 1,
					},
				}
				m.On("GetDashboardSummary", mock.Anything, "today").Return(summary, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, summary *DashboardSummary) {
				require.NotNil(t, summary.Rides)
				require.NotNil(t, summary.Drivers)
				require.NotNil(t, summary.Riders)
				require.NotNil(t, summary.Revenue)
				require.NotNil(t, summary.Alerts)

				assert.Equal(t, 50, summary.Rides.Total)
				assert.Equal(t, 90.0, summary.Rides.CompletionRate)
				assert.Equal(t, 35, summary.Drivers.OnlineNow)
				assert.Equal(t, 3500.00, summary.Revenue.Total)
			},
		},
		{
			name:   "success - returns dashboard summary for week",
			period: "week",
			setupMocks: func(m *mockRepo) {
				summary := &DashboardSummary{
					Rides:   &SummaryRides{Total: 300},
					Drivers: &SummaryDrivers{TotalActive: 80},
					Riders:  &SummaryRiders{TotalActive: 500},
					Revenue: &SummaryRevenue{Total: 20000.00},
					Alerts:  &SummaryAlerts{},
				}
				m.On("GetDashboardSummary", mock.Anything, "week").Return(summary, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, summary *DashboardSummary) {
				assert.Equal(t, 300, summary.Rides.Total)
				assert.Equal(t, 20000.00, summary.Revenue.Total)
			},
		},
		{
			name:   "error - database failure",
			period: "today",
			setupMocks: func(m *mockRepo) {
				m.On("GetDashboardSummary", mock.Anything, "today").Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			summary, err := svc.GetDashboardSummary(context.Background(), tt.period)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, summary)
			} else {
				require.NoError(t, err)
				require.NotNil(t, summary)
				if tt.validate != nil {
					tt.validate(t, summary)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetRevenueTrend
// ========================================

func TestGetRevenueTrend(t *testing.T) {
	tests := []struct {
		name       string
		period     string
		groupBy    string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, trend *RevenueTrend)
	}{
		{
			name:    "success - returns revenue trend by day",
			period:  "7days",
			groupBy: "day",
			setupMocks: func(m *mockRepo) {
				trend := &RevenueTrend{
					Period:          "7days",
					GroupBy:         "day",
					TotalRevenue:    15000.00,
					AvgDailyRevenue: 2142.86,
					Trend: []RevenueTrendData{
						{Date: "2025-01-01", Revenue: 2000.00, Rides: 40, AvgFare: 50.00},
						{Date: "2025-01-02", Revenue: 2200.00, Rides: 44, AvgFare: 50.00},
					},
				}
				m.On("GetRevenueTrend", mock.Anything, "7days", "day").Return(trend, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, trend *RevenueTrend) {
				assert.Equal(t, "7days", trend.Period)
				assert.Equal(t, "day", trend.GroupBy)
				assert.Equal(t, 15000.00, trend.TotalRevenue)
				assert.Len(t, trend.Trend, 2)
			},
		},
		{
			name:    "error - database failure",
			period:  "7days",
			groupBy: "day",
			setupMocks: func(m *mockRepo) {
				m.On("GetRevenueTrend", mock.Anything, "7days", "day").Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			trend, err := svc.GetRevenueTrend(context.Background(), tt.period, tt.groupBy)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, trend)
			} else {
				require.NoError(t, err)
				require.NotNil(t, trend)
				if tt.validate != nil {
					tt.validate(t, trend)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetActionItems
// ========================================

func TestGetActionItems(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, items *ActionItems)
	}{
		{
			name: "success - returns action items",
			setupMocks: func(m *mockRepo) {
				items := &ActionItems{
					PendingDriverApprovals: &ActionDriverApprovals{
						Count:       3,
						UrgentCount: 1,
						Items: []PendingDriverItem{
							{
								DriverID:    driverID,
								DriverName:  "John Doe",
								DaysWaiting: 2,
							},
						},
					},
					FraudAlerts: &ActionFraudAlerts{
						Count:         5,
						CriticalCount: 2,
						HighCount:     3,
					},
					NegativeFeedback: &ActionNegativeFeedback{
						Count:        10,
						OneStarCount: 3,
					},
					LowBalanceDrivers: &ActionLowBalance{Count: 0},
					ExpiredDocuments:  &ActionExpiredDocs{Count: 0},
				}
				m.On("GetActionItems", mock.Anything).Return(items, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, items *ActionItems) {
				require.NotNil(t, items.PendingDriverApprovals)
				assert.Equal(t, 3, items.PendingDriverApprovals.Count)
				assert.Equal(t, 1, items.PendingDriverApprovals.UrgentCount)
				assert.Len(t, items.PendingDriverApprovals.Items, 1)

				require.NotNil(t, items.FraudAlerts)
				assert.Equal(t, 5, items.FraudAlerts.Count)
				assert.Equal(t, 2, items.FraudAlerts.CriticalCount)

				require.NotNil(t, items.NegativeFeedback)
				assert.Equal(t, 10, items.NegativeFeedback.Count)
			},
		},
		{
			name: "success - empty action items",
			setupMocks: func(m *mockRepo) {
				items := &ActionItems{
					PendingDriverApprovals: &ActionDriverApprovals{Count: 0, Items: []PendingDriverItem{}},
					FraudAlerts:            &ActionFraudAlerts{Count: 0, Items: []FraudAlertItem{}},
					NegativeFeedback:       &ActionNegativeFeedback{Count: 0, Items: []NegativeFeedbackItem{}},
					LowBalanceDrivers:      &ActionLowBalance{Count: 0, Items: []LowBalanceItem{}},
					ExpiredDocuments:       &ActionExpiredDocs{Count: 0, Items: []ExpiredDocItem{}},
				}
				m.On("GetActionItems", mock.Anything).Return(items, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, items *ActionItems) {
				assert.Equal(t, 0, items.PendingDriverApprovals.Count)
				assert.Equal(t, 0, items.FraudAlerts.Count)
				assert.Equal(t, 0, items.NegativeFeedback.Count)
			},
		},
		{
			name: "error - database failure",
			setupMocks: func(m *mockRepo) {
				m.On("GetActionItems", mock.Anything).Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			items, err := svc.GetActionItems(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, items)
			} else {
				require.NoError(t, err)
				require.NotNil(t, items)
				if tt.validate != nil {
					tt.validate(t, items)
				}
			}

			m.AssertExpectations(t)
		})
	}
}
