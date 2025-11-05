package analytics

import (
	"context"
	"time"
)

// Service handles analytics business logic
type Service struct {
	repo *Repository
}

// NewService creates a new analytics service
func NewService(repo *Repository) *Service {
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
