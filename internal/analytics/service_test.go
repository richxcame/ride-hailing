package analytics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockAnalyticsRepository is a mock implementation of the AnalyticsRepository interface.
type mockAnalyticsRepository struct {
	mock.Mock
}

func (m *mockAnalyticsRepository) GetRevenueMetrics(ctx context.Context, startDate, endDate time.Time) (*RevenueMetrics, error) {
	args := m.Called(ctx, startDate, endDate)
	metrics, _ := args.Get(0).(*RevenueMetrics)
	return metrics, args.Error(1)
}

func (m *mockAnalyticsRepository) GetPromoCodePerformance(ctx context.Context, startDate, endDate time.Time) ([]*PromoCodePerformance, error) {
	args := m.Called(ctx, startDate, endDate)
	perf, _ := args.Get(0).([]*PromoCodePerformance)
	return perf, args.Error(1)
}

func (m *mockAnalyticsRepository) GetRideTypeStats(ctx context.Context, startDate, endDate time.Time) ([]*RideTypeStats, error) {
	args := m.Called(ctx, startDate, endDate)
	stats, _ := args.Get(0).([]*RideTypeStats)
	return stats, args.Error(1)
}

func (m *mockAnalyticsRepository) GetReferralMetrics(ctx context.Context, startDate, endDate time.Time) (*ReferralMetrics, error) {
	args := m.Called(ctx, startDate, endDate)
	metrics, _ := args.Get(0).(*ReferralMetrics)
	return metrics, args.Error(1)
}

func (m *mockAnalyticsRepository) GetTopDrivers(ctx context.Context, startDate, endDate time.Time, limit int) ([]*DriverPerformance, error) {
	args := m.Called(ctx, startDate, endDate, limit)
	drivers, _ := args.Get(0).([]*DriverPerformance)
	return drivers, args.Error(1)
}

func (m *mockAnalyticsRepository) GetDashboardMetrics(ctx context.Context) (*DashboardMetrics, error) {
	args := m.Called(ctx)
	metrics, _ := args.Get(0).(*DashboardMetrics)
	return metrics, args.Error(1)
}

func (m *mockAnalyticsRepository) GetDemandHeatMap(ctx context.Context, startDate, endDate time.Time, gridSize float64) ([]*DemandHeatMap, error) {
	args := m.Called(ctx, startDate, endDate, gridSize)
	heatmap, _ := args.Get(0).([]*DemandHeatMap)
	return heatmap, args.Error(1)
}

func (m *mockAnalyticsRepository) GetFinancialReport(ctx context.Context, startDate, endDate time.Time) (*FinancialReport, error) {
	args := m.Called(ctx, startDate, endDate)
	report, _ := args.Get(0).(*FinancialReport)
	return report, args.Error(1)
}

func (m *mockAnalyticsRepository) GetDemandZones(ctx context.Context, startDate, endDate time.Time, minRides int) ([]*DemandZone, error) {
	args := m.Called(ctx, startDate, endDate, minRides)
	zones, _ := args.Get(0).([]*DemandZone)
	return zones, args.Error(1)
}

func (m *mockAnalyticsRepository) GetRevenueTimeSeries(ctx context.Context, startDate, endDate time.Time, granularity string) ([]*RevenueTimeSeries, error) {
	args := m.Called(ctx, startDate, endDate, granularity)
	data, _ := args.Get(0).([]*RevenueTimeSeries)
	return data, args.Error(1)
}

func (m *mockAnalyticsRepository) GetHourlyDistribution(ctx context.Context, startDate, endDate time.Time) ([]*HourlyDistribution, error) {
	args := m.Called(ctx, startDate, endDate)
	data, _ := args.Get(0).([]*HourlyDistribution)
	return data, args.Error(1)
}

func (m *mockAnalyticsRepository) GetDriverAnalytics(ctx context.Context, startDate, endDate time.Time) (*DriverAnalytics, error) {
	args := m.Called(ctx, startDate, endDate)
	data, _ := args.Get(0).(*DriverAnalytics)
	return data, args.Error(1)
}

func (m *mockAnalyticsRepository) GetRiderGrowth(ctx context.Context, startDate, endDate time.Time) (*RiderGrowth, error) {
	args := m.Called(ctx, startDate, endDate)
	data, _ := args.Get(0).(*RiderGrowth)
	return data, args.Error(1)
}

func (m *mockAnalyticsRepository) GetRideMetrics(ctx context.Context, startDate, endDate time.Time) (*RideMetrics, error) {
	args := m.Called(ctx, startDate, endDate)
	data, _ := args.Get(0).(*RideMetrics)
	return data, args.Error(1)
}

func (m *mockAnalyticsRepository) GetTopDriversDetailed(ctx context.Context, startDate, endDate time.Time, limit int) ([]*TopDriver, error) {
	args := m.Called(ctx, startDate, endDate, limit)
	data, _ := args.Get(0).([]*TopDriver)
	return data, args.Error(1)
}

func (m *mockAnalyticsRepository) GetPeriodComparison(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time) (*PeriodComparison, error) {
	args := m.Called(ctx, currentStart, currentEnd, previousStart, previousEnd)
	data, _ := args.Get(0).(*PeriodComparison)
	return data, args.Error(1)
}

// ============================================================================
// NewService Tests
// ============================================================================

func TestNewService(t *testing.T) {
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)

	assert.NotNil(t, service)
	assert.Equal(t, repo, service.repo)
}

func TestNewService_NilRepo(t *testing.T) {
	service := NewService(nil)

	assert.NotNil(t, service)
	assert.Nil(t, service.repo)
}

// ============================================================================
// GetRevenueMetrics Tests
// ============================================================================

func TestService_GetRevenueMetrics_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	expected := &RevenueMetrics{
		Period:           "2024-01-01 to 2024-01-31",
		TotalRides:       150,
		TotalRevenue:     12345.67,
		AvgFarePerRide:   82.3,
		TotalDiscounts:   500.00,
		PlatformEarnings: 2469.13,
		DriverEarnings:   9876.54,
	}

	repo.On("GetRevenueMetrics", ctx, start, end).Return(expected, nil).Once()

	actual, err := service.GetRevenueMetrics(ctx, start, end)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
	repo.AssertExpectations(t)
}

func TestService_GetRevenueMetrics_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-7 * 24 * time.Hour)
	end := time.Now()
	expectedErr := errors.New("database connection failed")

	repo.On("GetRevenueMetrics", ctx, start, end).Return((*RevenueMetrics)(nil), expectedErr).Once()

	actual, err := service.GetRevenueMetrics(ctx, start, end)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, actual)
	repo.AssertExpectations(t)
}

func TestService_GetRevenueMetrics_EmptyResult(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	expected := &RevenueMetrics{
		TotalRides:     0,
		TotalRevenue:   0,
		AvgFarePerRide: 0,
	}

	repo.On("GetRevenueMetrics", ctx, start, end).Return(expected, nil).Once()

	actual, err := service.GetRevenueMetrics(ctx, start, end)
	assert.NoError(t, err)
	assert.Equal(t, 0, actual.TotalRides)
	assert.Equal(t, float64(0), actual.TotalRevenue)
	repo.AssertExpectations(t)
}

func TestService_GetRevenueMetrics_SameDayRange(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	sameTime := time.Now()
	expected := &RevenueMetrics{TotalRides: 10}

	repo.On("GetRevenueMetrics", ctx, sameTime, sameTime).Return(expected, nil).Once()

	actual, err := service.GetRevenueMetrics(ctx, sameTime, sameTime)
	assert.NoError(t, err)
	assert.Equal(t, 10, actual.TotalRides)
	repo.AssertExpectations(t)
}

// ============================================================================
// GetPromoCodePerformance Tests
// ============================================================================

func TestService_GetPromoCodePerformance_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	expected := []*PromoCodePerformance{
		{
			PromoCodeID:    uuid.New(),
			Code:           "SAVE10",
			TotalUses:      100,
			TotalDiscount:  1000.00,
			AvgDiscount:    10.00,
			UniqueUsers:    80,
			TotalRideValue: 5000.00,
		},
		{
			PromoCodeID:    uuid.New(),
			Code:           "FIRST50",
			TotalUses:      50,
			TotalDiscount:  2500.00,
			AvgDiscount:    50.00,
			UniqueUsers:    50,
			TotalRideValue: 2500.00,
		},
	}

	repo.On("GetPromoCodePerformance", ctx, start, end).Return(expected, nil).Once()

	actual, err := service.GetPromoCodePerformance(ctx, start, end)
	assert.NoError(t, err)
	assert.Len(t, actual, 2)
	assert.Equal(t, "SAVE10", actual[0].Code)
	assert.Equal(t, 100, actual[0].TotalUses)
	repo.AssertExpectations(t)
}

func TestService_GetPromoCodePerformance_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	expectedErr := errors.New("query timeout")

	repo.On("GetPromoCodePerformance", ctx, start, end).Return(([]*PromoCodePerformance)(nil), expectedErr).Once()

	actual, err := service.GetPromoCodePerformance(ctx, start, end)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, actual)
	repo.AssertExpectations(t)
}

func TestService_GetPromoCodePerformance_EmptyList(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	expected := []*PromoCodePerformance{}

	repo.On("GetPromoCodePerformance", ctx, start, end).Return(expected, nil).Once()

	actual, err := service.GetPromoCodePerformance(ctx, start, end)
	assert.NoError(t, err)
	assert.Empty(t, actual)
	repo.AssertExpectations(t)
}

func TestService_GetPromoCodePerformance_SinglePromo(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-7 * 24 * time.Hour)
	end := time.Now()
	expected := []*PromoCodePerformance{
		{Code: "SINGLE", TotalUses: 1, UniqueUsers: 1},
	}

	repo.On("GetPromoCodePerformance", ctx, start, end).Return(expected, nil).Once()

	actual, err := service.GetPromoCodePerformance(ctx, start, end)
	assert.NoError(t, err)
	assert.Len(t, actual, 1)
	assert.Equal(t, "SINGLE", actual[0].Code)
	repo.AssertExpectations(t)
}

// ============================================================================
// GetRideTypeStats Tests
// ============================================================================

func TestService_GetRideTypeStats_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	expected := []*RideTypeStats{
		{
			RideTypeID:   uuid.New(),
			Name:         "Standard",
			TotalRides:   500,
			TotalRevenue: 10000.00,
			AvgFare:      20.00,
			Percentage:   50.0,
		},
		{
			RideTypeID:   uuid.New(),
			Name:         "Premium",
			TotalRides:   300,
			TotalRevenue: 15000.00,
			AvgFare:      50.00,
			Percentage:   30.0,
		},
		{
			RideTypeID:   uuid.New(),
			Name:         "Economy",
			TotalRides:   200,
			TotalRevenue: 3000.00,
			AvgFare:      15.00,
			Percentage:   20.0,
		},
	}

	repo.On("GetRideTypeStats", ctx, start, end).Return(expected, nil).Once()

	actual, err := service.GetRideTypeStats(ctx, start, end)
	assert.NoError(t, err)
	assert.Len(t, actual, 3)
	assert.Equal(t, "Standard", actual[0].Name)
	assert.Equal(t, 500, actual[0].TotalRides)
	repo.AssertExpectations(t)
}

func TestService_GetRideTypeStats_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	expectedErr := errors.New("database error")

	repo.On("GetRideTypeStats", ctx, start, end).Return(([]*RideTypeStats)(nil), expectedErr).Once()

	actual, err := service.GetRideTypeStats(ctx, start, end)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, actual)
	repo.AssertExpectations(t)
}

func TestService_GetRideTypeStats_EmptyList(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	expected := []*RideTypeStats{}

	repo.On("GetRideTypeStats", ctx, start, end).Return(expected, nil).Once()

	actual, err := service.GetRideTypeStats(ctx, start, end)
	assert.NoError(t, err)
	assert.Empty(t, actual)
	repo.AssertExpectations(t)
}

// ============================================================================
// GetReferralMetrics Tests
// ============================================================================

func TestService_GetReferralMetrics_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	expected := &ReferralMetrics{
		TotalReferrals:      100,
		CompletedReferrals:  75,
		TotalBonusesPaid:    750.00,
		AvgBonusPerReferral: 10.00,
		ConversionRate:      75.0,
	}

	repo.On("GetReferralMetrics", ctx, start, end).Return(expected, nil).Once()

	actual, err := service.GetReferralMetrics(ctx, start, end)
	assert.NoError(t, err)
	assert.Equal(t, 100, actual.TotalReferrals)
	assert.Equal(t, 75, actual.CompletedReferrals)
	assert.Equal(t, 75.0, actual.ConversionRate)
	repo.AssertExpectations(t)
}

func TestService_GetReferralMetrics_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	expectedErr := errors.New("referral table not found")

	repo.On("GetReferralMetrics", ctx, start, end).Return((*ReferralMetrics)(nil), expectedErr).Once()

	actual, err := service.GetReferralMetrics(ctx, start, end)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, actual)
	repo.AssertExpectations(t)
}

func TestService_GetReferralMetrics_NoReferrals(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	expected := &ReferralMetrics{
		TotalReferrals:     0,
		CompletedReferrals: 0,
		ConversionRate:     0,
	}

	repo.On("GetReferralMetrics", ctx, start, end).Return(expected, nil).Once()

	actual, err := service.GetReferralMetrics(ctx, start, end)
	assert.NoError(t, err)
	assert.Equal(t, 0, actual.TotalReferrals)
	assert.Equal(t, 0.0, actual.ConversionRate)
	repo.AssertExpectations(t)
}

func TestService_GetReferralMetrics_HighConversion(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-7 * 24 * time.Hour)
	end := time.Now()
	expected := &ReferralMetrics{
		TotalReferrals:     10,
		CompletedReferrals: 10,
		ConversionRate:     100.0,
	}

	repo.On("GetReferralMetrics", ctx, start, end).Return(expected, nil).Once()

	actual, err := service.GetReferralMetrics(ctx, start, end)
	assert.NoError(t, err)
	assert.Equal(t, 100.0, actual.ConversionRate)
	repo.AssertExpectations(t)
}

// ============================================================================
// GetTopDrivers Tests
// ============================================================================

func TestService_GetTopDrivers_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	limit := 5
	drivers := []*DriverPerformance{
		{DriverID: uuid.New(), DriverName: "John Doe", TotalRides: 150, AvgRating: 4.92, TotalEarnings: 3000.00},
		{DriverID: uuid.New(), DriverName: "Jane Smith", TotalRides: 140, AvgRating: 4.88, TotalEarnings: 2800.00},
		{DriverID: uuid.New(), DriverName: "Bob Wilson", TotalRides: 130, AvgRating: 4.95, TotalEarnings: 2600.00},
	}

	repo.On("GetTopDrivers", ctx, start, end, limit).Return(drivers, nil).Once()

	result, err := service.GetTopDrivers(ctx, start, end, limit)
	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "John Doe", result[0].DriverName)
	assert.Equal(t, 150, result[0].TotalRides)
	repo.AssertExpectations(t)
}

func TestService_GetTopDrivers_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	expectedErr := errors.New("failed to query drivers")

	repo.On("GetTopDrivers", ctx, start, end, 10).Return(([]*DriverPerformance)(nil), expectedErr).Once()

	result, err := service.GetTopDrivers(ctx, start, end, 10)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetTopDrivers_EmptyList(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	drivers := []*DriverPerformance{}

	repo.On("GetTopDrivers", ctx, start, end, 5).Return(drivers, nil).Once()

	result, err := service.GetTopDrivers(ctx, start, end, 5)
	assert.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetTopDrivers_LimitOne(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	drivers := []*DriverPerformance{
		{DriverID: uuid.New(), DriverName: "Top Driver", TotalRides: 200, AvgRating: 5.0},
	}

	repo.On("GetTopDrivers", ctx, start, end, 1).Return(drivers, nil).Once()

	result, err := service.GetTopDrivers(ctx, start, end, 1)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Top Driver", result[0].DriverName)
	repo.AssertExpectations(t)
}

func TestService_GetTopDrivers_ZeroLimit(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	drivers := []*DriverPerformance{}

	repo.On("GetTopDrivers", ctx, start, end, 0).Return(drivers, nil).Once()

	result, err := service.GetTopDrivers(ctx, start, end, 0)
	assert.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

// ============================================================================
// GetDashboardMetrics Tests
// ============================================================================

func TestService_GetDashboardMetrics_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	expected := &DashboardMetrics{
		TotalRides:     9800,
		ActiveRides:    45,
		CompletedToday: 320,
		RevenueToday:   4500.00,
		ActiveDrivers:  450,
		ActiveRiders:   3200,
		AvgRating:      4.8,
		TopPromoCode:   &PromoCodePerformance{Code: "SAVE20", TotalUses: 50},
		TopRideType:    &RideTypeStats{Name: "Standard", TotalRides: 200},
	}

	repo.On("GetDashboardMetrics", ctx).Return(expected, nil).Once()

	result, err := service.GetDashboardMetrics(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 9800, result.TotalRides)
	assert.Equal(t, 450, result.ActiveDrivers)
	assert.Equal(t, 4.8, result.AvgRating)
	assert.NotNil(t, result.TopPromoCode)
	assert.NotNil(t, result.TopRideType)
	repo.AssertExpectations(t)
}

func TestService_GetDashboardMetrics_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	expectedErr := errors.New("dashboard query failed")

	repo.On("GetDashboardMetrics", ctx).Return((*DashboardMetrics)(nil), expectedErr).Once()

	result, err := service.GetDashboardMetrics(ctx)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetDashboardMetrics_NoTopPromo(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	expected := &DashboardMetrics{
		TotalRides:    100,
		ActiveDrivers: 10,
		TopPromoCode:  nil,
		TopRideType:   nil,
	}

	repo.On("GetDashboardMetrics", ctx).Return(expected, nil).Once()

	result, err := service.GetDashboardMetrics(ctx)
	assert.NoError(t, err)
	assert.Nil(t, result.TopPromoCode)
	assert.Nil(t, result.TopRideType)
	repo.AssertExpectations(t)
}

func TestService_GetDashboardMetrics_ZeroValues(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	expected := &DashboardMetrics{
		TotalRides:     0,
		ActiveRides:    0,
		CompletedToday: 0,
		RevenueToday:   0,
		ActiveDrivers:  0,
		ActiveRiders:   0,
		AvgRating:      0,
	}

	repo.On("GetDashboardMetrics", ctx).Return(expected, nil).Once()

	result, err := service.GetDashboardMetrics(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.TotalRides)
	assert.Equal(t, float64(0), result.RevenueToday)
	repo.AssertExpectations(t)
}

// ============================================================================
// GetDemandHeatMap Tests
// ============================================================================

func TestService_GetDemandHeatMap_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-time.Hour)
	end := time.Now()
	grid := 0.5
	heat := []*DemandHeatMap{
		{Latitude: 37.77, Longitude: -122.42, RideCount: 45, DemandLevel: "high", SurgeActive: true},
		{Latitude: 37.78, Longitude: -122.43, RideCount: 30, DemandLevel: "medium", SurgeActive: false},
		{Latitude: 37.76, Longitude: -122.41, RideCount: 60, DemandLevel: "very_high", SurgeActive: true},
	}

	repo.On("GetDemandHeatMap", ctx, start, end, grid).Return(heat, nil).Once()

	result, err := service.GetDemandHeatMap(ctx, start, end, grid)
	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, 45, result[0].RideCount)
	assert.Equal(t, "high", result[0].DemandLevel)
	repo.AssertExpectations(t)
}

func TestService_GetDemandHeatMap_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-time.Hour)
	end := time.Now()
	expectedErr := errors.New("geo query failed")

	repo.On("GetDemandHeatMap", ctx, start, end, 0.01).Return(([]*DemandHeatMap)(nil), expectedErr).Once()

	result, err := service.GetDemandHeatMap(ctx, start, end, 0.01)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetDemandHeatMap_EmptyResult(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-time.Hour)
	end := time.Now()
	heat := []*DemandHeatMap{}

	repo.On("GetDemandHeatMap", ctx, start, end, 0.1).Return(heat, nil).Once()

	result, err := service.GetDemandHeatMap(ctx, start, end, 0.1)
	assert.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetDemandHeatMap_DifferentGridSizes(t *testing.T) {
	tests := []struct {
		name     string
		gridSize float64
	}{
		{"small grid", 0.001},
		{"medium grid", 0.01},
		{"large grid", 0.1},
		{"zero grid", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(mockAnalyticsRepository)
			service := NewService(repo)
			start := time.Now().Add(-time.Hour)
			end := time.Now()
			heat := []*DemandHeatMap{{RideCount: 10}}

			repo.On("GetDemandHeatMap", ctx, start, end, tt.gridSize).Return(heat, nil).Once()

			result, err := service.GetDemandHeatMap(ctx, start, end, tt.gridSize)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			repo.AssertExpectations(t)
		})
	}
}

// ============================================================================
// GetFinancialReport Tests
// ============================================================================

func TestService_GetFinancialReport_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	expected := &FinancialReport{
		Period:             "2024-01-01 to 2024-01-31",
		GrossRevenue:       50000.00,
		NetRevenue:         48000.00,
		PlatformCommission: 10000.00,
		DriverPayouts:      40000.00,
		PromoDiscounts:     2000.00,
		ReferralBonuses:    500.00,
		Refunds:            200.00,
		TotalExpenses:      42700.00,
		Profit:             7300.00,
		ProfitMargin:       15.21,
		TotalRides:         2500,
		CompletedRides:     2400,
		CancelledRides:     100,
		AvgRevenuePerRide:  20.00,
		TopRevenueDay:      "2024-01-15",
		TopRevenueDayAmount: 2500.00,
	}

	repo.On("GetFinancialReport", ctx, start, end).Return(expected, nil).Once()

	result, err := service.GetFinancialReport(ctx, start, end)
	assert.NoError(t, err)
	assert.Equal(t, 50000.00, result.GrossRevenue)
	assert.Equal(t, 2500, result.TotalRides)
	assert.Equal(t, "2024-01-15", result.TopRevenueDay)
	repo.AssertExpectations(t)
}

func TestService_GetFinancialReport_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	expectedErr := errors.New("financial data unavailable")

	repo.On("GetFinancialReport", ctx, start, end).Return((*FinancialReport)(nil), expectedErr).Once()

	result, err := service.GetFinancialReport(ctx, start, end)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetFinancialReport_ZeroRevenue(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	expected := &FinancialReport{
		GrossRevenue:   0,
		NetRevenue:     0,
		TotalRides:     0,
		CompletedRides: 0,
	}

	repo.On("GetFinancialReport", ctx, start, end).Return(expected, nil).Once()

	result, err := service.GetFinancialReport(ctx, start, end)
	assert.NoError(t, err)
	assert.Equal(t, float64(0), result.GrossRevenue)
	repo.AssertExpectations(t)
}

func TestService_GetFinancialReport_NegativeProfit(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-7 * 24 * time.Hour)
	end := time.Now()
	expected := &FinancialReport{
		GrossRevenue:       1000.00,
		PlatformCommission: 200.00,
		ReferralBonuses:    300.00,
		Refunds:            100.00,
		Profit:             -200.00,
		ProfitMargin:       -20.0,
	}

	repo.On("GetFinancialReport", ctx, start, end).Return(expected, nil).Once()

	result, err := service.GetFinancialReport(ctx, start, end)
	assert.NoError(t, err)
	assert.Less(t, result.Profit, float64(0))
	assert.Less(t, result.ProfitMargin, float64(0))
	repo.AssertExpectations(t)
}

// ============================================================================
// GetDemandZones Tests
// ============================================================================

func TestService_GetDemandZones_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	minRides := 10
	expected := []*DemandZone{
		{ZoneName: "Zone 1", CenterLatitude: 37.77, CenterLongitude: -122.42, TotalRides: 150, AvgSurge: 1.5, PeakHours: "[08:00 17:00 18:00]"},
		{ZoneName: "Zone 2", CenterLatitude: 37.78, CenterLongitude: -122.43, TotalRides: 100, AvgSurge: 1.3, PeakHours: "[09:00 18:00 19:00]"},
	}

	repo.On("GetDemandZones", ctx, start, end, minRides).Return(expected, nil).Once()

	result, err := service.GetDemandZones(ctx, start, end, minRides)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "Zone 1", result[0].ZoneName)
	assert.Equal(t, 150, result[0].TotalRides)
	repo.AssertExpectations(t)
}

func TestService_GetDemandZones_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	expectedErr := errors.New("zone calculation failed")

	repo.On("GetDemandZones", ctx, start, end, 5).Return(([]*DemandZone)(nil), expectedErr).Once()

	result, err := service.GetDemandZones(ctx, start, end, 5)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetDemandZones_EmptyResult(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-time.Hour)
	end := time.Now()
	zones := []*DemandZone{}

	repo.On("GetDemandZones", ctx, start, end, 100).Return(zones, nil).Once()

	result, err := service.GetDemandZones(ctx, start, end, 100)
	assert.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetDemandZones_ZeroMinRides(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	zones := []*DemandZone{{ZoneName: "Zone 1", TotalRides: 1}}

	repo.On("GetDemandZones", ctx, start, end, 0).Return(zones, nil).Once()

	result, err := service.GetDemandZones(ctx, start, end, 0)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	repo.AssertExpectations(t)
}

// ============================================================================
// GetRevenueTimeSeries Tests
// ============================================================================

func TestService_GetRevenueTimeSeries_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-7 * 24 * time.Hour)
	end := time.Now()
	granularity := "day"
	expected := []*RevenueTimeSeries{
		{Date: "2024-01-01", Revenue: 1000.00, Rides: 50, AvgFare: 20.00},
		{Date: "2024-01-02", Revenue: 1200.00, Rides: 60, AvgFare: 20.00},
		{Date: "2024-01-03", Revenue: 1100.00, Rides: 55, AvgFare: 20.00},
	}

	repo.On("GetRevenueTimeSeries", ctx, start, end, granularity).Return(expected, nil).Once()

	result, err := service.GetRevenueTimeSeries(ctx, start, end, granularity)
	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "2024-01-01", result[0].Date)
	assert.Equal(t, 1000.00, result[0].Revenue)
	repo.AssertExpectations(t)
}

func TestService_GetRevenueTimeSeries_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	expectedErr := errors.New("timeseries query failed")

	repo.On("GetRevenueTimeSeries", ctx, start, end, "week").Return(([]*RevenueTimeSeries)(nil), expectedErr).Once()

	result, err := service.GetRevenueTimeSeries(ctx, start, end, "week")
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetRevenueTimeSeries_DifferentGranularities(t *testing.T) {
	granularities := []string{"day", "week", "month"}

	for _, gran := range granularities {
		t.Run("granularity_"+gran, func(t *testing.T) {
			ctx := context.Background()
			repo := new(mockAnalyticsRepository)
			service := NewService(repo)
			start := time.Now().Add(-90 * 24 * time.Hour)
			end := time.Now()
			expected := []*RevenueTimeSeries{{Date: "2024-01", Revenue: 10000.00}}

			repo.On("GetRevenueTimeSeries", ctx, start, end, gran).Return(expected, nil).Once()

			result, err := service.GetRevenueTimeSeries(ctx, start, end, gran)
			assert.NoError(t, err)
			assert.NotEmpty(t, result)
			repo.AssertExpectations(t)
		})
	}
}

func TestService_GetRevenueTimeSeries_EmptyResult(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	expected := []*RevenueTimeSeries{}

	repo.On("GetRevenueTimeSeries", ctx, start, end, "day").Return(expected, nil).Once()

	result, err := service.GetRevenueTimeSeries(ctx, start, end, "day")
	assert.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

// ============================================================================
// GetHourlyDistribution Tests
// ============================================================================

func TestService_GetHourlyDistribution_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	expected := []*HourlyDistribution{
		{Hour: 8, Rides: 100, AvgFare: 25.00, AvgWaitTime: 3.5},
		{Hour: 9, Rides: 150, AvgFare: 22.00, AvgWaitTime: 4.0},
		{Hour: 17, Rides: 200, AvgFare: 30.00, AvgWaitTime: 5.0},
		{Hour: 18, Rides: 180, AvgFare: 28.00, AvgWaitTime: 4.5},
	}

	repo.On("GetHourlyDistribution", ctx, start, end).Return(expected, nil).Once()

	result, err := service.GetHourlyDistribution(ctx, start, end)
	assert.NoError(t, err)
	assert.Len(t, result, 4)
	assert.Equal(t, 8, result[0].Hour)
	assert.Equal(t, 100, result[0].Rides)
	repo.AssertExpectations(t)
}

func TestService_GetHourlyDistribution_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	expectedErr := errors.New("hourly distribution query failed")

	repo.On("GetHourlyDistribution", ctx, start, end).Return(([]*HourlyDistribution)(nil), expectedErr).Once()

	result, err := service.GetHourlyDistribution(ctx, start, end)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetHourlyDistribution_AllHours(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-7 * 24 * time.Hour)
	end := time.Now()

	// Create 24 hours of data
	expected := make([]*HourlyDistribution, 24)
	for i := 0; i < 24; i++ {
		expected[i] = &HourlyDistribution{Hour: i, Rides: 10 + i*5, AvgFare: 20.00}
	}

	repo.On("GetHourlyDistribution", ctx, start, end).Return(expected, nil).Once()

	result, err := service.GetHourlyDistribution(ctx, start, end)
	assert.NoError(t, err)
	assert.Len(t, result, 24)
	assert.Equal(t, 0, result[0].Hour)
	assert.Equal(t, 23, result[23].Hour)
	repo.AssertExpectations(t)
}

func TestService_GetHourlyDistribution_EmptyResult(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-time.Hour)
	end := time.Now()
	expected := []*HourlyDistribution{}

	repo.On("GetHourlyDistribution", ctx, start, end).Return(expected, nil).Once()

	result, err := service.GetHourlyDistribution(ctx, start, end)
	assert.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

// ============================================================================
// GetDriverAnalytics Tests
// ============================================================================

func TestService_GetDriverAnalytics_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	expected := &DriverAnalytics{
		TotalActiveDrivers:  500,
		AvgRidesPerDriver:   25.5,
		AvgOnlineHours:      6.5,
		AvgAcceptanceRate:   92.0,
		AvgCancellationRate: 3.0,
		AvgRating:           4.75,
		NewDrivers:          50,
		ChurnedDrivers:      10,
	}

	repo.On("GetDriverAnalytics", ctx, start, end).Return(expected, nil).Once()

	result, err := service.GetDriverAnalytics(ctx, start, end)
	assert.NoError(t, err)
	assert.Equal(t, 500, result.TotalActiveDrivers)
	assert.Equal(t, 25.5, result.AvgRidesPerDriver)
	assert.Equal(t, 4.75, result.AvgRating)
	repo.AssertExpectations(t)
}

func TestService_GetDriverAnalytics_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	expectedErr := errors.New("driver analytics query failed")

	repo.On("GetDriverAnalytics", ctx, start, end).Return((*DriverAnalytics)(nil), expectedErr).Once()

	result, err := service.GetDriverAnalytics(ctx, start, end)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetDriverAnalytics_NoDrivers(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	expected := &DriverAnalytics{
		TotalActiveDrivers:  0,
		AvgRidesPerDriver:   0,
		AvgAcceptanceRate:   0,
		AvgCancellationRate: 0,
	}

	repo.On("GetDriverAnalytics", ctx, start, end).Return(expected, nil).Once()

	result, err := service.GetDriverAnalytics(ctx, start, end)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.TotalActiveDrivers)
	repo.AssertExpectations(t)
}

func TestService_GetDriverAnalytics_HighChurn(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	expected := &DriverAnalytics{
		TotalActiveDrivers: 100,
		NewDrivers:         10,
		ChurnedDrivers:     50,
	}

	repo.On("GetDriverAnalytics", ctx, start, end).Return(expected, nil).Once()

	result, err := service.GetDriverAnalytics(ctx, start, end)
	assert.NoError(t, err)
	assert.Greater(t, result.ChurnedDrivers, result.NewDrivers)
	repo.AssertExpectations(t)
}

// ============================================================================
// GetRiderGrowth Tests
// ============================================================================

func TestService_GetRiderGrowth_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	expected := &RiderGrowth{
		NewRiders:            500,
		ReturningRiders:      3000,
		ChurnedRiders:        100,
		RetentionRate:        85.0,
		AvgRidesPerRider:     3.5,
		FirstTimeRidersToday: 25,
		LifetimeValueAvg:     150.00,
	}

	repo.On("GetRiderGrowth", ctx, start, end).Return(expected, nil).Once()

	result, err := service.GetRiderGrowth(ctx, start, end)
	assert.NoError(t, err)
	assert.Equal(t, 500, result.NewRiders)
	assert.Equal(t, 3000, result.ReturningRiders)
	assert.Equal(t, 85.0, result.RetentionRate)
	repo.AssertExpectations(t)
}

func TestService_GetRiderGrowth_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	expectedErr := errors.New("rider growth query failed")

	repo.On("GetRiderGrowth", ctx, start, end).Return((*RiderGrowth)(nil), expectedErr).Once()

	result, err := service.GetRiderGrowth(ctx, start, end)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetRiderGrowth_NoRiders(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	expected := &RiderGrowth{
		NewRiders:         0,
		ReturningRiders:   0,
		RetentionRate:     0,
		AvgRidesPerRider:  0,
		LifetimeValueAvg:  0,
	}

	repo.On("GetRiderGrowth", ctx, start, end).Return(expected, nil).Once()

	result, err := service.GetRiderGrowth(ctx, start, end)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.NewRiders)
	assert.Equal(t, 0.0, result.RetentionRate)
	repo.AssertExpectations(t)
}

func TestService_GetRiderGrowth_HighRetention(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-7 * 24 * time.Hour)
	end := time.Now()
	expected := &RiderGrowth{
		NewRiders:       100,
		ReturningRiders: 900,
		ChurnedRiders:   5,
		RetentionRate:   99.5,
	}

	repo.On("GetRiderGrowth", ctx, start, end).Return(expected, nil).Once()

	result, err := service.GetRiderGrowth(ctx, start, end)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, result.RetentionRate, 99.0)
	repo.AssertExpectations(t)
}

// ============================================================================
// GetRideMetrics Tests
// ============================================================================

func TestService_GetRideMetrics_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	expected := &RideMetrics{
		AvgWaitTimeMinutes:     3.5,
		AvgRideDurationMinutes: 15.0,
		AvgDistanceKm:          8.5,
		CancellationRate:       5.0,
		RiderCancellationRate:  3.0,
		DriverCancellationRate: 2.0,
		SurgeRidesPercentage:   15.0,
		AvgSurgeMultiplier:     1.3,
	}

	repo.On("GetRideMetrics", ctx, start, end).Return(expected, nil).Once()

	result, err := service.GetRideMetrics(ctx, start, end)
	assert.NoError(t, err)
	assert.Equal(t, 3.5, result.AvgWaitTimeMinutes)
	assert.Equal(t, 15.0, result.AvgRideDurationMinutes)
	assert.Equal(t, 5.0, result.CancellationRate)
	repo.AssertExpectations(t)
}

func TestService_GetRideMetrics_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	expectedErr := errors.New("ride metrics query failed")

	repo.On("GetRideMetrics", ctx, start, end).Return((*RideMetrics)(nil), expectedErr).Once()

	result, err := service.GetRideMetrics(ctx, start, end)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetRideMetrics_NoSurge(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	expected := &RideMetrics{
		SurgeRidesPercentage: 0,
		AvgSurgeMultiplier:   1.0,
	}

	repo.On("GetRideMetrics", ctx, start, end).Return(expected, nil).Once()

	result, err := service.GetRideMetrics(ctx, start, end)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, result.SurgeRidesPercentage)
	assert.Equal(t, 1.0, result.AvgSurgeMultiplier)
	repo.AssertExpectations(t)
}

func TestService_GetRideMetrics_HighCancellation(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	expected := &RideMetrics{
		CancellationRate:       25.0,
		RiderCancellationRate:  15.0,
		DriverCancellationRate: 10.0,
	}

	repo.On("GetRideMetrics", ctx, start, end).Return(expected, nil).Once()

	result, err := service.GetRideMetrics(ctx, start, end)
	assert.NoError(t, err)
	assert.Equal(t, 25.0, result.CancellationRate)
	assert.Equal(t, result.RiderCancellationRate+result.DriverCancellationRate, result.CancellationRate)
	repo.AssertExpectations(t)
}

func TestService_GetRideMetrics_ZeroValues(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-time.Hour)
	end := time.Now()
	expected := &RideMetrics{
		AvgWaitTimeMinutes:     0,
		AvgRideDurationMinutes: 0,
		AvgDistanceKm:          0,
		CancellationRate:       0,
	}

	repo.On("GetRideMetrics", ctx, start, end).Return(expected, nil).Once()

	result, err := service.GetRideMetrics(ctx, start, end)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, result.AvgWaitTimeMinutes)
	repo.AssertExpectations(t)
}

// ============================================================================
// GetTopDriversDetailed Tests
// ============================================================================

func TestService_GetTopDriversDetailed_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	limit := 10
	expected := []*TopDriver{
		{DriverID: uuid.New(), Name: "Alice Brown", TotalRides: 200, TotalRevenue: 5000.00, AvgRating: 4.95, AcceptanceRate: 98.0, OnlineHours: 160.0},
		{DriverID: uuid.New(), Name: "Bob Smith", TotalRides: 180, TotalRevenue: 4500.00, AvgRating: 4.90, AcceptanceRate: 95.0, OnlineHours: 150.0},
		{DriverID: uuid.New(), Name: "Carol Davis", TotalRides: 170, TotalRevenue: 4200.00, AvgRating: 4.88, AcceptanceRate: 92.0, OnlineHours: 140.0},
	}

	repo.On("GetTopDriversDetailed", ctx, start, end, limit).Return(expected, nil).Once()

	result, err := service.GetTopDriversDetailed(ctx, start, end, limit)
	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "Alice Brown", result[0].Name)
	assert.Equal(t, 200, result[0].TotalRides)
	assert.Equal(t, 5000.00, result[0].TotalRevenue)
	repo.AssertExpectations(t)
}

func TestService_GetTopDriversDetailed_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	expectedErr := errors.New("top drivers query failed")

	repo.On("GetTopDriversDetailed", ctx, start, end, 5).Return(([]*TopDriver)(nil), expectedErr).Once()

	result, err := service.GetTopDriversDetailed(ctx, start, end, 5)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetTopDriversDetailed_EmptyList(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	expected := []*TopDriver{}

	repo.On("GetTopDriversDetailed", ctx, start, end, 10).Return(expected, nil).Once()

	result, err := service.GetTopDriversDetailed(ctx, start, end, 10)
	assert.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetTopDriversDetailed_SingleDriver(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	expected := []*TopDriver{
		{DriverID: uuid.New(), Name: "Solo Driver", TotalRides: 500, AvgRating: 5.0},
	}

	repo.On("GetTopDriversDetailed", ctx, start, end, 1).Return(expected, nil).Once()

	result, err := service.GetTopDriversDetailed(ctx, start, end, 1)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Solo Driver", result[0].Name)
	repo.AssertExpectations(t)
}

func TestService_GetTopDriversDetailed_LargeLimit(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-90 * 24 * time.Hour)
	end := time.Now()
	// Even with large limit, we only get available drivers
	expected := []*TopDriver{
		{Name: "Driver 1", TotalRides: 100},
		{Name: "Driver 2", TotalRides: 90},
	}

	repo.On("GetTopDriversDetailed", ctx, start, end, 1000).Return(expected, nil).Once()

	result, err := service.GetTopDriversDetailed(ctx, start, end, 1000)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	repo.AssertExpectations(t)
}

// ============================================================================
// GetPeriodComparison Tests
// ============================================================================

func TestService_GetPeriodComparison_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	currentStart := time.Now().Add(-7 * 24 * time.Hour)
	currentEnd := time.Now()
	previousStart := time.Now().Add(-14 * 24 * time.Hour)
	previousEnd := time.Now().Add(-7 * 24 * time.Hour)

	expected := &PeriodComparison{
		Revenue:   ComparisonMetric{Current: 10000.00, Previous: 8000.00, ChangePercent: 25.0},
		Rides:     ComparisonMetric{Current: 500, Previous: 400, ChangePercent: 25.0},
		NewRiders: ComparisonMetric{Current: 100, Previous: 80, ChangePercent: 25.0},
		AvgRating: ComparisonMetric{Current: 4.8, Previous: 4.7, ChangePercent: 2.13},
	}

	repo.On("GetPeriodComparison", ctx, currentStart, currentEnd, previousStart, previousEnd).Return(expected, nil).Once()

	result, err := service.GetPeriodComparison(ctx, currentStart, currentEnd, previousStart, previousEnd)
	assert.NoError(t, err)
	assert.Equal(t, 10000.00, result.Revenue.Current)
	assert.Equal(t, 8000.00, result.Revenue.Previous)
	assert.Equal(t, 25.0, result.Revenue.ChangePercent)
	repo.AssertExpectations(t)
}

func TestService_GetPeriodComparison_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	currentStart := time.Now().Add(-7 * 24 * time.Hour)
	currentEnd := time.Now()
	previousStart := time.Now().Add(-14 * 24 * time.Hour)
	previousEnd := time.Now().Add(-7 * 24 * time.Hour)
	expectedErr := errors.New("comparison query failed")

	repo.On("GetPeriodComparison", ctx, currentStart, currentEnd, previousStart, previousEnd).Return((*PeriodComparison)(nil), expectedErr).Once()

	result, err := service.GetPeriodComparison(ctx, currentStart, currentEnd, previousStart, previousEnd)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetPeriodComparison_NegativeGrowth(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	currentStart := time.Now().Add(-7 * 24 * time.Hour)
	currentEnd := time.Now()
	previousStart := time.Now().Add(-14 * 24 * time.Hour)
	previousEnd := time.Now().Add(-7 * 24 * time.Hour)

	expected := &PeriodComparison{
		Revenue:   ComparisonMetric{Current: 8000.00, Previous: 10000.00, ChangePercent: -20.0},
		Rides:     ComparisonMetric{Current: 400, Previous: 500, ChangePercent: -20.0},
		NewRiders: ComparisonMetric{Current: 50, Previous: 100, ChangePercent: -50.0},
		AvgRating: ComparisonMetric{Current: 4.5, Previous: 4.8, ChangePercent: -6.25},
	}

	repo.On("GetPeriodComparison", ctx, currentStart, currentEnd, previousStart, previousEnd).Return(expected, nil).Once()

	result, err := service.GetPeriodComparison(ctx, currentStart, currentEnd, previousStart, previousEnd)
	assert.NoError(t, err)
	assert.Less(t, result.Revenue.ChangePercent, float64(0))
	assert.Less(t, result.Rides.ChangePercent, float64(0))
	repo.AssertExpectations(t)
}

func TestService_GetPeriodComparison_ZeroPreviousPeriod(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	currentStart := time.Now().Add(-7 * 24 * time.Hour)
	currentEnd := time.Now()
	previousStart := time.Now().Add(-14 * 24 * time.Hour)
	previousEnd := time.Now().Add(-7 * 24 * time.Hour)

	expected := &PeriodComparison{
		Revenue:   ComparisonMetric{Current: 5000.00, Previous: 0, ChangePercent: 100.0},
		Rides:     ComparisonMetric{Current: 250, Previous: 0, ChangePercent: 100.0},
		NewRiders: ComparisonMetric{Current: 50, Previous: 0, ChangePercent: 100.0},
		AvgRating: ComparisonMetric{Current: 4.5, Previous: 0, ChangePercent: 100.0},
	}

	repo.On("GetPeriodComparison", ctx, currentStart, currentEnd, previousStart, previousEnd).Return(expected, nil).Once()

	result, err := service.GetPeriodComparison(ctx, currentStart, currentEnd, previousStart, previousEnd)
	assert.NoError(t, err)
	assert.Equal(t, 100.0, result.Revenue.ChangePercent)
	repo.AssertExpectations(t)
}

func TestService_GetPeriodComparison_BothPeriodsZero(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	currentStart := time.Now().Add(-7 * 24 * time.Hour)
	currentEnd := time.Now()
	previousStart := time.Now().Add(-14 * 24 * time.Hour)
	previousEnd := time.Now().Add(-7 * 24 * time.Hour)

	expected := &PeriodComparison{
		Revenue:   ComparisonMetric{Current: 0, Previous: 0, ChangePercent: 0},
		Rides:     ComparisonMetric{Current: 0, Previous: 0, ChangePercent: 0},
		NewRiders: ComparisonMetric{Current: 0, Previous: 0, ChangePercent: 0},
		AvgRating: ComparisonMetric{Current: 0, Previous: 0, ChangePercent: 0},
	}

	repo.On("GetPeriodComparison", ctx, currentStart, currentEnd, previousStart, previousEnd).Return(expected, nil).Once()

	result, err := service.GetPeriodComparison(ctx, currentStart, currentEnd, previousStart, previousEnd)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, result.Revenue.ChangePercent)
	repo.AssertExpectations(t)
}

func TestService_GetPeriodComparison_SamePeriod(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-7 * 24 * time.Hour)
	end := time.Now()

	expected := &PeriodComparison{
		Revenue:   ComparisonMetric{Current: 5000.00, Previous: 5000.00, ChangePercent: 0},
		Rides:     ComparisonMetric{Current: 250, Previous: 250, ChangePercent: 0},
		NewRiders: ComparisonMetric{Current: 50, Previous: 50, ChangePercent: 0},
		AvgRating: ComparisonMetric{Current: 4.5, Previous: 4.5, ChangePercent: 0},
	}

	repo.On("GetPeriodComparison", ctx, start, end, start, end).Return(expected, nil).Once()

	result, err := service.GetPeriodComparison(ctx, start, end, start, end)
	assert.NoError(t, err)
	assert.Equal(t, result.Revenue.Current, result.Revenue.Previous)
	repo.AssertExpectations(t)
}

// ============================================================================
// Context Cancellation Tests
// ============================================================================

func TestService_GetRevenueMetrics_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	expectedErr := context.Canceled

	repo.On("GetRevenueMetrics", ctx, start, end).Return((*RevenueMetrics)(nil), expectedErr).Once()

	_, err := service.GetRevenueMetrics(ctx, start, end)
	assert.ErrorIs(t, err, context.Canceled)
	repo.AssertExpectations(t)
}

func TestService_GetDashboardMetrics_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond) // Let it timeout
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	expectedErr := context.DeadlineExceeded

	repo.On("GetDashboardMetrics", ctx).Return((*DashboardMetrics)(nil), expectedErr).Once()

	_, err := service.GetDashboardMetrics(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	repo.AssertExpectations(t)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestService_GetRevenueMetrics_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		startDate time.Time
		endDate   time.Time
		expected  *RevenueMetrics
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "one day range",
			startDate: time.Now().Add(-24 * time.Hour),
			endDate:   time.Now(),
			expected:  &RevenueMetrics{TotalRides: 100, TotalRevenue: 2000.00},
			wantErr:   false,
		},
		{
			name:      "one week range",
			startDate: time.Now().Add(-7 * 24 * time.Hour),
			endDate:   time.Now(),
			expected:  &RevenueMetrics{TotalRides: 700, TotalRevenue: 14000.00},
			wantErr:   false,
		},
		{
			name:      "one month range",
			startDate: time.Now().Add(-30 * 24 * time.Hour),
			endDate:   time.Now(),
			expected:  &RevenueMetrics{TotalRides: 3000, TotalRevenue: 60000.00},
			wantErr:   false,
		},
		{
			name:      "database error",
			startDate: time.Now().Add(-24 * time.Hour),
			endDate:   time.Now(),
			expected:  nil,
			wantErr:   true,
			errMsg:    "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(mockAnalyticsRepository)
			service := NewService(repo)

			if tt.wantErr {
				repo.On("GetRevenueMetrics", ctx, tt.startDate, tt.endDate).
					Return((*RevenueMetrics)(nil), errors.New(tt.errMsg)).Once()
			} else {
				repo.On("GetRevenueMetrics", ctx, tt.startDate, tt.endDate).
					Return(tt.expected, nil).Once()
			}

			result, err := service.GetRevenueMetrics(ctx, tt.startDate, tt.endDate)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestService_GetDashboardMetrics_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		expected *DashboardMetrics
		wantErr  bool
		errMsg   string
	}{
		{
			name: "active platform",
			expected: &DashboardMetrics{
				TotalRides:     10000,
				ActiveRides:    50,
				CompletedToday: 500,
				RevenueToday:   7500.00,
				ActiveDrivers:  500,
				ActiveRiders:   5000,
				AvgRating:      4.85,
			},
			wantErr: false,
		},
		{
			name: "quiet platform",
			expected: &DashboardMetrics{
				TotalRides:     100,
				ActiveRides:    0,
				CompletedToday: 5,
				RevenueToday:   75.00,
				ActiveDrivers:  10,
				ActiveRiders:   50,
				AvgRating:      4.5,
			},
			wantErr: false,
		},
		{
			name:     "database down",
			expected: nil,
			wantErr:  true,
			errMsg:   "database connection lost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(mockAnalyticsRepository)
			service := NewService(repo)

			if tt.wantErr {
				repo.On("GetDashboardMetrics", ctx).
					Return((*DashboardMetrics)(nil), errors.New(tt.errMsg)).Once()
			} else {
				repo.On("GetDashboardMetrics", ctx).
					Return(tt.expected, nil).Once()
			}

			result, err := service.GetDashboardMetrics(ctx)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestService_GetTopDrivers_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		expected []*DriverPerformance
		wantErr  bool
	}{
		{
			name:  "top 5 drivers",
			limit: 5,
			expected: []*DriverPerformance{
				{DriverName: "Driver 1", TotalRides: 200},
				{DriverName: "Driver 2", TotalRides: 180},
				{DriverName: "Driver 3", TotalRides: 160},
				{DriverName: "Driver 4", TotalRides: 140},
				{DriverName: "Driver 5", TotalRides: 120},
			},
			wantErr: false,
		},
		{
			name:     "top 10 with only 3 available",
			limit:    10,
			expected: []*DriverPerformance{{DriverName: "D1"}, {DriverName: "D2"}, {DriverName: "D3"}},
			wantErr:  false,
		},
		{
			name:     "no drivers",
			limit:    5,
			expected: []*DriverPerformance{},
			wantErr:  false,
		},
		{
			name:     "query error",
			limit:    5,
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(mockAnalyticsRepository)
			service := NewService(repo)
			start := time.Now().Add(-30 * 24 * time.Hour)
			end := time.Now()

			if tt.wantErr {
				repo.On("GetTopDrivers", ctx, start, end, tt.limit).
					Return(([]*DriverPerformance)(nil), errors.New("error")).Once()
			} else {
				repo.On("GetTopDrivers", ctx, start, end, tt.limit).
					Return(tt.expected, nil).Once()
			}

			result, err := service.GetTopDrivers(ctx, start, end, tt.limit)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
			repo.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Edge Case Tests
// ============================================================================

func TestService_GetRevenueMetrics_FutureDates(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(24 * time.Hour)
	end := time.Now().Add(48 * time.Hour)
	expected := &RevenueMetrics{TotalRides: 0, TotalRevenue: 0}

	repo.On("GetRevenueMetrics", ctx, start, end).Return(expected, nil).Once()

	result, err := service.GetRevenueMetrics(ctx, start, end)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.TotalRides)
	repo.AssertExpectations(t)
}

func TestService_GetRevenueMetrics_VeryOldDates(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-365 * 24 * time.Hour)
	end := time.Now().Add(-364 * 24 * time.Hour)
	expected := &RevenueMetrics{TotalRides: 50, TotalRevenue: 1000}

	repo.On("GetRevenueMetrics", ctx, start, end).Return(expected, nil).Once()

	result, err := service.GetRevenueMetrics(ctx, start, end)
	assert.NoError(t, err)
	assert.Equal(t, 50, result.TotalRides)
	repo.AssertExpectations(t)
}

func TestService_GetRevenueMetrics_StartAfterEnd(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now()
	end := time.Now().Add(-24 * time.Hour)
	expected := &RevenueMetrics{TotalRides: 0}

	repo.On("GetRevenueMetrics", ctx, start, end).Return(expected, nil).Once()

	result, err := service.GetRevenueMetrics(ctx, start, end)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.TotalRides)
	repo.AssertExpectations(t)
}

func TestService_GetDemandHeatMap_NegativeGridSize(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	gridSize := -0.01
	expected := []*DemandHeatMap{{RideCount: 5}}

	repo.On("GetDemandHeatMap", ctx, start, end, gridSize).Return(expected, nil).Once()

	result, err := service.GetDemandHeatMap(ctx, start, end, gridSize)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetTopDrivers_NegativeLimit(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	expected := []*DriverPerformance{}

	repo.On("GetTopDrivers", ctx, start, end, -5).Return(expected, nil).Once()

	result, err := service.GetTopDrivers(ctx, start, end, -5)
	assert.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetDemandZones_NegativeMinRides(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	expected := []*DemandZone{{ZoneName: "Zone 1"}}

	repo.On("GetDemandZones", ctx, start, end, -10).Return(expected, nil).Once()

	result, err := service.GetDemandZones(ctx, start, end, -10)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	repo.AssertExpectations(t)
}

// ============================================================================
// Large Data Tests
// ============================================================================

func TestService_GetPromoCodePerformance_LargeList(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-90 * 24 * time.Hour)
	end := time.Now()

	// Create a large list of promo codes
	expected := make([]*PromoCodePerformance, 100)
	for i := 0; i < 100; i++ {
		expected[i] = &PromoCodePerformance{
			Code:      "PROMO" + string(rune('A'+i%26)),
			TotalUses: 100 - i,
		}
	}

	repo.On("GetPromoCodePerformance", ctx, start, end).Return(expected, nil).Once()

	result, err := service.GetPromoCodePerformance(ctx, start, end)
	assert.NoError(t, err)
	assert.Len(t, result, 100)
	repo.AssertExpectations(t)
}

func TestService_GetDemandHeatMap_LargeResult(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()

	// Create large heat map data
	expected := make([]*DemandHeatMap, 1000)
	for i := 0; i < 1000; i++ {
		expected[i] = &DemandHeatMap{
			Latitude:  37.0 + float64(i)/1000,
			Longitude: -122.0 + float64(i)/1000,
			RideCount: i + 1,
		}
	}

	repo.On("GetDemandHeatMap", ctx, start, end, 0.001).Return(expected, nil).Once()

	result, err := service.GetDemandHeatMap(ctx, start, end, 0.001)
	assert.NoError(t, err)
	assert.Len(t, result, 1000)
	repo.AssertExpectations(t)
}
