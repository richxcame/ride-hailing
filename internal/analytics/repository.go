package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	commissionRate = 0.20 // 20% platform commission
)

// Repository handles database operations for analytics
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new analytics repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// GetRevenueMetrics retrieves revenue statistics for a time period
func (r *Repository) GetRevenueMetrics(ctx context.Context, startDate, endDate time.Time) (*RevenueMetrics, error) {
	query := `
		SELECT
			COUNT(*) as total_rides,
			COALESCE(SUM(final_fare), 0) as total_revenue,
			COALESCE(AVG(final_fare), 0) as avg_fare,
			COALESCE(SUM(discount_amount), 0) as total_discounts
		FROM rides
		WHERE status = 'completed'
		  AND completed_at >= $1
		  AND completed_at <= $2
	`

	metrics := &RevenueMetrics{}
	err := r.db.QueryRow(ctx, query, startDate, endDate).Scan(
		&metrics.TotalRides,
		&metrics.TotalRevenue,
		&metrics.AvgFarePerRide,
		&metrics.TotalDiscounts,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue metrics: %w", err)
	}

	metrics.PlatformEarnings = metrics.TotalRevenue * commissionRate
	metrics.DriverEarnings = metrics.TotalRevenue * (1 - commissionRate)
	metrics.Period = fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	return metrics, nil
}

// GetPromoCodePerformance retrieves promo code usage statistics
func (r *Repository) GetPromoCodePerformance(ctx context.Context, startDate, endDate time.Time) ([]*PromoCodePerformance, error) {
	query := `
		SELECT
			pc.id,
			pc.code,
			COUNT(pcu.id) as total_uses,
			COALESCE(SUM(pcu.discount_amount), 0) as total_discount,
			COALESCE(AVG(pcu.discount_amount), 0) as avg_discount,
			COUNT(DISTINCT pcu.user_id) as unique_users,
			COALESCE(SUM(pcu.ride_amount), 0) as total_ride_value
		FROM promo_codes pc
		LEFT JOIN promo_code_uses pcu ON pc.id = pcu.promo_code_id
			AND pcu.used_at >= $1
			AND pcu.used_at <= $2
		WHERE pc.created_at <= $2
		GROUP BY pc.id, pc.code
		HAVING COUNT(pcu.id) > 0
		ORDER BY total_uses DESC
		LIMIT 20
	`

	rows, err := r.db.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get promo code performance: %w", err)
	}
	defer rows.Close()

	var results []*PromoCodePerformance
	for rows.Next() {
		perf := &PromoCodePerformance{}
		err := rows.Scan(
			&perf.PromoCodeID,
			&perf.Code,
			&perf.TotalUses,
			&perf.TotalDiscount,
			&perf.AvgDiscount,
			&perf.UniqueUsers,
			&perf.TotalRideValue,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan promo performance: %w", err)
		}
		results = append(results, perf)
	}

	return results, nil
}

// GetRideTypeStats retrieves ride type usage statistics
func (r *Repository) GetRideTypeStats(ctx context.Context, startDate, endDate time.Time) ([]*RideTypeStats, error) {
	query := `
		WITH total_rides AS (
			SELECT COUNT(*) as count
			FROM rides
			WHERE status = 'completed'
			  AND completed_at >= $1
			  AND completed_at <= $2
		)
		SELECT
			rt.id,
			rt.name,
			COUNT(r.id) as total_rides,
			COALESCE(SUM(r.final_fare), 0) as total_revenue,
			COALESCE(AVG(r.final_fare), 0) as avg_fare,
			CASE
				WHEN tr.count > 0 THEN (COUNT(r.id)::float / tr.count * 100)
				ELSE 0
			END as percentage
		FROM ride_types rt
		LEFT JOIN rides r ON rt.id = r.ride_type_id
			AND r.status = 'completed'
			AND r.completed_at >= $1
			AND r.completed_at <= $2
		CROSS JOIN total_rides tr
		GROUP BY rt.id, rt.name, tr.count
		ORDER BY total_rides DESC
	`

	rows, err := r.db.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride type stats: %w", err)
	}
	defer rows.Close()

	var results []*RideTypeStats
	for rows.Next() {
		stats := &RideTypeStats{}
		err := rows.Scan(
			&stats.RideTypeID,
			&stats.Name,
			&stats.TotalRides,
			&stats.TotalRevenue,
			&stats.AvgFare,
			&stats.Percentage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ride type stats: %w", err)
		}
		results = append(results, stats)
	}

	return results, nil
}

// GetReferralMetrics retrieves referral program statistics
func (r *Repository) GetReferralMetrics(ctx context.Context, startDate, endDate time.Time) (*ReferralMetrics, error) {
	query := `
		SELECT
			COUNT(*) as total_referrals,
			COUNT(*) FILTER (WHERE referrer_bonus_applied = true) as completed_referrals,
			COALESCE(SUM(referrer_bonus + referred_bonus) FILTER (WHERE referrer_bonus_applied = true), 0) as total_bonuses
		FROM referrals
		WHERE created_at >= $1
		  AND created_at <= $2
	`

	metrics := &ReferralMetrics{}
	var totalReferrals, completedReferrals int
	var totalBonuses float64

	err := r.db.QueryRow(ctx, query, startDate, endDate).Scan(
		&totalReferrals,
		&completedReferrals,
		&totalBonuses,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get referral metrics: %w", err)
	}

	metrics.TotalReferrals = totalReferrals
	metrics.CompletedReferrals = completedReferrals
	metrics.TotalBonusesPaid = totalBonuses

	if completedReferrals > 0 {
		metrics.AvgBonusPerReferral = totalBonuses / float64(completedReferrals)
	}

	if totalReferrals > 0 {
		metrics.ConversionRate = float64(completedReferrals) / float64(totalReferrals) * 100
	}

	return metrics, nil
}

// GetTopDrivers retrieves top performing drivers
func (r *Repository) GetTopDrivers(ctx context.Context, startDate, endDate time.Time, limit int) ([]*DriverPerformance, error) {
	query := `
		SELECT
			u.id,
			COALESCE(u.first_name || ' ' || u.last_name, u.email) as driver_name,
			COUNT(r.id) as total_rides,
			COALESCE(SUM(r.final_fare * 0.8), 0) as total_earnings,
			COALESCE(AVG(r.rating), 0) as avg_rating,
			CASE
				WHEN COUNT(r.id) > 0 THEN
					(COUNT(*) FILTER (WHERE r.status = 'completed')::float / COUNT(*) * 100)
				ELSE 0
			END as completion_rate,
			CASE
				WHEN COUNT(r.id) > 0 THEN
					(COUNT(*) FILTER (WHERE r.status = 'cancelled')::float / COUNT(*) * 100)
				ELSE 0
			END as cancellation_rate
		FROM users u
		INNER JOIN rides r ON u.id = r.driver_id
		WHERE u.role = 'driver'
		  AND r.completed_at >= $1
		  AND r.completed_at <= $2
		GROUP BY u.id, driver_name
		ORDER BY total_earnings DESC
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, query, startDate, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top drivers: %w", err)
	}
	defer rows.Close()

	var results []*DriverPerformance
	for rows.Next() {
		perf := &DriverPerformance{}
		err := rows.Scan(
			&perf.DriverID,
			&perf.DriverName,
			&perf.TotalRides,
			&perf.TotalEarnings,
			&perf.AvgRating,
			&perf.CompletionRate,
			&perf.CancellationRate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan driver performance: %w", err)
		}
		results = append(results, perf)
	}

	return results, nil
}

// GetDashboardMetrics retrieves overall platform metrics
func (r *Repository) GetDashboardMetrics(ctx context.Context) (*DashboardMetrics, error) {
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	metrics := &DashboardMetrics{}

	// Get overall ride counts
	query1 := `
		SELECT
			COUNT(*) as total_rides,
			COUNT(*) FILTER (WHERE status IN ('requested', 'accepted', 'in_progress')) as active_rides,
			COUNT(*) FILTER (WHERE status = 'completed' AND completed_at >= $1 AND completed_at < $2) as completed_today,
			COALESCE(SUM(final_fare) FILTER (WHERE status = 'completed' AND completed_at >= $1 AND completed_at < $2), 0) as revenue_today,
			COALESCE(AVG(rating) FILTER (WHERE rating IS NOT NULL), 0) as avg_rating
		FROM rides
	`

	err := r.db.QueryRow(ctx, query1, today, tomorrow).Scan(
		&metrics.TotalRides,
		&metrics.ActiveRides,
		&metrics.CompletedToday,
		&metrics.RevenueToday,
		&metrics.AvgRating,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard ride metrics: %w", err)
	}

	// Get active users
	query2 := `
		SELECT
			COUNT(DISTINCT CASE WHEN role = 'driver' THEN id END) as active_drivers,
			COUNT(DISTINCT CASE WHEN role = 'rider' THEN id END) as active_riders
		FROM users
		WHERE is_active = true
	`

	err = r.db.QueryRow(ctx, query2).Scan(
		&metrics.ActiveDrivers,
		&metrics.ActiveRiders,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard user metrics: %w", err)
	}

	// Get top promo code for today
	promos, err := r.GetPromoCodePerformance(ctx, today, tomorrow)
	if err == nil && len(promos) > 0 {
		metrics.TopPromoCode = promos[0]
	}

	// Get top ride type for today
	rideTypes, err := r.GetRideTypeStats(ctx, today, tomorrow)
	if err == nil && len(rideTypes) > 0 {
		metrics.TopRideType = rideTypes[0]
	}

	return metrics, nil
}
