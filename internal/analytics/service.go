package analytics

import (
	"context"
	"time"
)

// AnalyticsRepository defines the persistence operations required by the service.
type AnalyticsRepository interface {
	GetRevenueMetrics(ctx context.Context, startDate, endDate time.Time) (*RevenueMetrics, error)
	GetPromoCodePerformance(ctx context.Context, startDate, endDate time.Time) ([]*PromoCodePerformance, error)
	GetRideTypeStats(ctx context.Context, startDate, endDate time.Time) ([]*RideTypeStats, error)
	GetReferralMetrics(ctx context.Context, startDate, endDate time.Time) (*ReferralMetrics, error)
	GetTopDrivers(ctx context.Context, startDate, endDate time.Time, limit int) ([]*DriverPerformance, error)
	GetDashboardMetrics(ctx context.Context) (*DashboardMetrics, error)
	GetDemandHeatMap(ctx context.Context, startDate, endDate time.Time, gridSize float64) ([]*DemandHeatMap, error)
	GetFinancialReport(ctx context.Context, startDate, endDate time.Time) (*FinancialReport, error)
	GetDemandZones(ctx context.Context, startDate, endDate time.Time, minRides int) ([]*DemandZone, error)
}

// Service handles analytics business logic
type Service struct {
	repo AnalyticsRepository
}

// NewService creates a new analytics service
func NewService(repo AnalyticsRepository) *Service {
	return &Service{repo: repo}
}

// GetRevenueMetrics retrieves revenue statistics
func (s *Service) GetRevenueMetrics(ctx context.Context, startDate, endDate time.Time) (*RevenueMetrics, error) {
	return s.repo.GetRevenueMetrics(ctx, startDate, endDate)
}

// GetPromoCodePerformance retrieves promo code statistics
func (s *Service) GetPromoCodePerformance(ctx context.Context, startDate, endDate time.Time) ([]*PromoCodePerformance, error) {
	return s.repo.GetPromoCodePerformance(ctx, startDate, endDate)
}

// GetRideTypeStats retrieves ride type statistics
func (s *Service) GetRideTypeStats(ctx context.Context, startDate, endDate time.Time) ([]*RideTypeStats, error) {
	return s.repo.GetRideTypeStats(ctx, startDate, endDate)
}

// GetReferralMetrics retrieves referral program statistics
func (s *Service) GetReferralMetrics(ctx context.Context, startDate, endDate time.Time) (*ReferralMetrics, error) {
	return s.repo.GetReferralMetrics(ctx, startDate, endDate)
}

// GetTopDrivers retrieves top performing drivers
func (s *Service) GetTopDrivers(ctx context.Context, startDate, endDate time.Time, limit int) ([]*DriverPerformance, error) {
	return s.repo.GetTopDrivers(ctx, startDate, endDate, limit)
}

// GetDashboardMetrics retrieves overall platform metrics
func (s *Service) GetDashboardMetrics(ctx context.Context) (*DashboardMetrics, error) {
	return s.repo.GetDashboardMetrics(ctx)
}

// GetDemandHeatMap retrieves geographic demand data for heat map visualization
func (s *Service) GetDemandHeatMap(ctx context.Context, startDate, endDate time.Time, gridSize float64) ([]*DemandHeatMap, error) {
	return s.repo.GetDemandHeatMap(ctx, startDate, endDate, gridSize)
}

// GetFinancialReport generates a comprehensive financial report for a period
func (s *Service) GetFinancialReport(ctx context.Context, startDate, endDate time.Time) (*FinancialReport, error) {
	return s.repo.GetFinancialReport(ctx, startDate, endDate)
}

// GetDemandZones identifies high-demand geographic zones
func (s *Service) GetDemandZones(ctx context.Context, startDate, endDate time.Time, minRides int) ([]*DemandZone, error) {
	return s.repo.GetDemandZones(ctx, startDate, endDate, minRides)
}
