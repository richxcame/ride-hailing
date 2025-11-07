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

// Ensure Repository implements AnalyticsRepository.
var _ AnalyticsRepository = (*Repository)(nil)

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

// GetDemandHeatMap retrieves geographic demand data for heat map visualization
func (r *Repository) GetDemandHeatMap(ctx context.Context, startDate, endDate time.Time, gridSize float64) ([]*DemandHeatMap, error) {
	// Grid size is in degrees (approximately 0.01 = 1km)
	if gridSize == 0 {
		gridSize = 0.01 // Default ~1km grid
	}

	query := `
		SELECT
			ROUND(pickup_latitude / $3) * $3 as grid_lat,
			ROUND(pickup_longitude / $3) * $3 as grid_lon,
			COUNT(*) as ride_count,
			COALESCE(AVG(EXTRACT(EPOCH FROM (accepted_at - requested_at)) / 60), 0)::int as avg_wait_minutes,
			COALESCE(AVG(final_fare), 0) as avg_fare,
			COALESCE(AVG(surge_multiplier), 1.0) as avg_surge
		FROM rides
		WHERE status = 'completed'
		  AND completed_at >= $1
		  AND completed_at <= $2
		  AND pickup_latitude IS NOT NULL
		  AND pickup_longitude IS NOT NULL
		GROUP BY grid_lat, grid_lon
		HAVING COUNT(*) >= 3
		ORDER BY ride_count DESC
		LIMIT 100
	`

	rows, err := r.db.Query(ctx, query, startDate, endDate, gridSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get demand heat map: %w", err)
	}
	defer rows.Close()

	var results []*DemandHeatMap
	for rows.Next() {
		heatMap := &DemandHeatMap{}
		var avgSurge float64

		err := rows.Scan(
			&heatMap.Latitude,
			&heatMap.Longitude,
			&heatMap.RideCount,
			&heatMap.AvgWaitTime,
			&heatMap.AvgFare,
			&avgSurge,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan heat map data: %w", err)
		}

		// Determine demand level based on ride count
		switch {
		case heatMap.RideCount >= 50:
			heatMap.DemandLevel = "very_high"
		case heatMap.RideCount >= 20:
			heatMap.DemandLevel = "high"
		case heatMap.RideCount >= 10:
			heatMap.DemandLevel = "medium"
		default:
			heatMap.DemandLevel = "low"
		}

		heatMap.SurgeActive = avgSurge > 1.0

		results = append(results, heatMap)
	}

	return results, nil
}

// GetFinancialReport generates a comprehensive financial report for a period
func (r *Repository) GetFinancialReport(ctx context.Context, startDate, endDate time.Time) (*FinancialReport, error) {
	report := &FinancialReport{
		Period: fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
	}

	// Get ride and revenue statistics
	rideQuery := `
		SELECT
			COUNT(*) FILTER (WHERE status = 'completed') as completed_rides,
			COUNT(*) FILTER (WHERE status = 'cancelled') as cancelled_rides,
			COUNT(*) as total_rides,
			COALESCE(SUM(final_fare) FILTER (WHERE status = 'completed'), 0) as gross_revenue,
			COALESCE(SUM(discount_amount) FILTER (WHERE status = 'completed'), 0) as promo_discounts,
			COALESCE(AVG(final_fare) FILTER (WHERE status = 'completed'), 0) as avg_revenue_per_ride
		FROM rides
		WHERE requested_at >= $1
		  AND requested_at <= $2
	`

	err := r.db.QueryRow(ctx, rideQuery, startDate, endDate).Scan(
		&report.CompletedRides,
		&report.CancelledRides,
		&report.TotalRides,
		&report.GrossRevenue,
		&report.PromoDiscounts,
		&report.AvgRevenuePerRide,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride statistics: %w", err)
	}

	// Calculate platform commission (20% of gross revenue)
	report.PlatformCommission = report.GrossRevenue * commissionRate
	report.DriverPayouts = report.GrossRevenue * (1 - commissionRate)

	// Get referral bonuses paid
	referralQuery := `
		SELECT COALESCE(SUM(referrer_bonus + referred_bonus), 0)
		FROM referrals
		WHERE referrer_bonus_applied = true
		  AND created_at >= $1
		  AND created_at <= $2
	`

	err = r.db.QueryRow(ctx, referralQuery, startDate, endDate).Scan(&report.ReferralBonuses)
	if err != nil {
		report.ReferralBonuses = 0
	}

	// Get actual refunds from payments table
	refundQuery := `
		SELECT COALESCE(SUM(amount), 0)
		FROM payments
		WHERE status = 'refunded'
		  AND created_at >= $1
		  AND created_at <= $2
	`

	err = r.db.QueryRow(ctx, refundQuery, startDate, endDate).Scan(&report.Refunds)
	if err != nil {
		// Fallback to estimation if query fails
		report.Refunds = float64(report.CancelledRides) * 5.0
	}

	// Calculate financial metrics
	report.TotalExpenses = report.DriverPayouts + report.PromoDiscounts + report.ReferralBonuses + report.Refunds
	report.NetRevenue = report.GrossRevenue - report.PromoDiscounts
	report.Profit = report.PlatformCommission - report.ReferralBonuses - report.Refunds

	if report.NetRevenue > 0 {
		report.ProfitMargin = (report.Profit / report.NetRevenue) * 100
	}

	// Get top revenue day
	dayQuery := `
		SELECT
			DATE(completed_at) as revenue_day,
			SUM(final_fare) as day_revenue
		FROM rides
		WHERE status = 'completed'
		  AND completed_at >= $1
		  AND completed_at <= $2
		GROUP BY revenue_day
		ORDER BY day_revenue DESC
		LIMIT 1
	`

	var topDay *time.Time
	err = r.db.QueryRow(ctx, dayQuery, startDate, endDate).Scan(&topDay, &report.TopRevenueDayAmount)
	if err == nil && topDay != nil {
		report.TopRevenueDay = topDay.Format("2006-01-02")
	}

	return report, nil
}

// GetDemandZones identifies high-demand geographic zones
func (r *Repository) GetDemandZones(ctx context.Context, startDate, endDate time.Time, minRides int) ([]*DemandZone, error) {
	if minRides == 0 {
		minRides = 20 // Default minimum rides to qualify as a zone
	}

	query := `
		WITH zone_data AS (
			SELECT
				ROUND(pickup_latitude / 0.05) * 0.05 as zone_lat,
				ROUND(pickup_longitude / 0.05) * 0.05 as zone_lon,
				COUNT(*) as ride_count,
				COALESCE(AVG(surge_multiplier), 1.0) as avg_surge,
				ARRAY_AGG(DISTINCT EXTRACT(HOUR FROM requested_at)::int ORDER BY EXTRACT(HOUR FROM requested_at)::int) as hours
			FROM rides
			WHERE status = 'completed'
			  AND completed_at >= $1
			  AND completed_at <= $2
			GROUP BY zone_lat, zone_lon
			HAVING COUNT(*) >= $3
		)
		SELECT
			zone_lat,
			zone_lon,
			ride_count,
			avg_surge,
			hours
		FROM zone_data
		ORDER BY ride_count DESC
		LIMIT 20
	`

	rows, err := r.db.Query(ctx, query, startDate, endDate, minRides)
	if err != nil {
		return nil, fmt.Errorf("failed to get demand zones: %w", err)
	}
	defer rows.Close()

	var results []*DemandZone
	zoneNum := 1
	for rows.Next() {
		zone := &DemandZone{
			RadiusKm: 5.0, // ~5km radius for each zone
		}

		var hours []int
		err := rows.Scan(
			&zone.CenterLat,
			&zone.CenterLon,
			&zone.TotalRides,
			&zone.AvgSurge,
			&hours,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan demand zone: %w", err)
		}

		zone.ZoneName = fmt.Sprintf("Zone %d", zoneNum)
		zoneNum++

		// Format peak hours
		if len(hours) > 0 {
			peakHours := []string{}
			for _, h := range hours {
				if len(peakHours) < 3 { // Show top 3 peak hours
					peakHours = append(peakHours, fmt.Sprintf("%02d:00", h))
				}
			}
			zone.PeakHours = fmt.Sprintf("%v", peakHours)
		}

		results = append(results, zone)
	}

	return results, nil
}
