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

func TestServiceGetRevenueMetrics(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	expected := &RevenueMetrics{
		TotalRides:     150,
		TotalRevenue:   12345.67,
		AvgFarePerRide: 82.3,
	}

	repo.On("GetRevenueMetrics", ctx, start, end).Return(expected, nil).Once()

	actual, err := service.GetRevenueMetrics(ctx, start, end)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
	repo.AssertExpectations(t)
}

func TestServiceGetRevenueMetricsError(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-7 * 24 * time.Hour)
	end := time.Now()
	expectedErr := errors.New("db down")

	repo.On("GetRevenueMetrics", ctx, start, end).Return((*RevenueMetrics)(nil), expectedErr).Once()

	actual, err := service.GetRevenueMetrics(ctx, start, end)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, actual)
	repo.AssertExpectations(t)
}

func TestServiceGetTopDrivers(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()
	drivers := []*DriverPerformance{
		{DriverID: uuid.New(), DriverName: "driver-1", TotalRides: 120, AvgRating: 4.92},
	}

	repo.On("GetTopDrivers", ctx, start, end, 5).Return(drivers, nil).Once()

	result, err := service.GetTopDrivers(ctx, start, end, 5)
	assert.NoError(t, err)
	assert.Equal(t, drivers, result)
	repo.AssertExpectations(t)
}

func TestServiceGetDemandHeatMap(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	start := time.Now().Add(-time.Hour)
	end := time.Now()
	grid := 0.5
	heat := []*DemandHeatMap{
		{Latitude: 37.77, Longitude: -122.42, RideCount: 45},
	}

	repo.On("GetDemandHeatMap", ctx, start, end, grid).Return(heat, nil).Once()

	result, err := service.GetDemandHeatMap(ctx, start, end, grid)
	assert.NoError(t, err)
	assert.Equal(t, heat, result)
	repo.AssertExpectations(t)
}

func TestServiceGetDashboardMetrics(t *testing.T) {
	ctx := context.Background()
	repo := new(mockAnalyticsRepository)
	service := NewService(repo)
	expected := &DashboardMetrics{
		TotalRides:    9800,
		ActiveDrivers: 450,
		ActiveRiders:  3200,
		AvgRating:     4.8,
	}

	repo.On("GetDashboardMetrics", ctx).Return(expected, nil).Once()

	result, err := service.GetDashboardMetrics(ctx)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}
