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

	results := make([]*PromoCodePerformance, 0)
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

	results := make([]*RideTypeStats, 0)
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

	results := make([]*DriverPerformance, 0)
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
			ROUND(pickup_latitude / $3) * $3 as grid_latitude,
			ROUND(pickup_longitude / $3) * $3 as grid_longitude,
			COUNT(*) as ride_count,
			COALESCE(AVG(EXTRACT(EPOCH FROM (accepted_at - requested_at)) / 60), 0)::int as avg_wait_minutes,
			COALESCE(AVG(final_fare), 0) as avg_fare,
			COALESCE(AVG(surge_multiplier), 1.0) as avg_surge
		FROM rides
		WHERE created_at >= $1
		  AND created_at <= $2
		  AND pickup_latitude IS NOT NULL
		  AND pickup_longitude IS NOT NULL
		GROUP BY grid_latitude, grid_longitude
		HAVING COUNT(*) >= 1
		ORDER BY ride_count DESC
		LIMIT 100
	`

	rows, err := r.db.Query(ctx, query, startDate, endDate, gridSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get demand heat map: %w", err)
	}
	defer rows.Close()

	results := make([]*DemandHeatMap, 0)
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
		minRides = 1 // Default minimum rides to qualify as a zone
	}

	query := `
		WITH zone_data AS (
			SELECT
				ROUND(pickup_latitude / 0.05) * 0.05 as zone_latitude,
				ROUND(pickup_longitude / 0.05) * 0.05 as zone_longitude,
				COUNT(*) as ride_count,
				COALESCE(AVG(surge_multiplier), 1.0) as avg_surge,
				ARRAY_AGG(DISTINCT EXTRACT(HOUR FROM requested_at)::int ORDER BY EXTRACT(HOUR FROM requested_at)::int) as hours
			FROM rides
			WHERE created_at >= $1
			  AND created_at <= $2
			  AND pickup_latitude IS NOT NULL
			  AND pickup_longitude IS NOT NULL
			GROUP BY zone_latitude, zone_longitude
			HAVING COUNT(*) >= $3
		)
		SELECT
			zone_latitude,
			zone_longitude,
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

	results := make([]*DemandZone, 0)
	zoneNum := 1
	for rows.Next() {
		zone := &DemandZone{
			RadiusKm: 5.0, // ~5km radius for each zone
		}

		var hours []int
		err := rows.Scan(
			&zone.CenterLatitude,
			&zone.CenterLongitude,
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

// GetRevenueTimeSeries retrieves time-series revenue data for charts
func (r *Repository) GetRevenueTimeSeries(ctx context.Context, startDate, endDate time.Time, granularity string) ([]*RevenueTimeSeries, error) {
	var dateFormat, groupBy string
	switch granularity {
	case "week":
		dateFormat = "YYYY-WW"
		groupBy = "DATE_TRUNC('week', completed_at)"
	case "month":
		dateFormat = "YYYY-MM"
		groupBy = "DATE_TRUNC('month', completed_at)"
	default: // day
		dateFormat = "YYYY-MM-DD"
		groupBy = "DATE(completed_at)"
	}

	query := fmt.Sprintf(`
		SELECT
			TO_CHAR(%s, '%s') as date,
			COALESCE(SUM(final_fare), 0) as revenue,
			COUNT(*) as rides,
			COALESCE(AVG(final_fare), 0) as avg_fare
		FROM rides
		WHERE status = 'completed'
		  AND completed_at >= $1
		  AND completed_at <= $2
		GROUP BY %s
		ORDER BY %s ASC
	`, groupBy, dateFormat, groupBy, groupBy)

	rows, err := r.db.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue timeseries: %w", err)
	}
	defer rows.Close()

	results := make([]*RevenueTimeSeries, 0)
	for rows.Next() {
		ts := &RevenueTimeSeries{}
		err := rows.Scan(&ts.Date, &ts.Revenue, &ts.Rides, &ts.AvgFare)
		if err != nil {
			return nil, fmt.Errorf("failed to scan revenue timeseries: %w", err)
		}
		results = append(results, ts)
	}

	return results, nil
}

// GetHourlyDistribution retrieves ride distribution by hour
func (r *Repository) GetHourlyDistribution(ctx context.Context, startDate, endDate time.Time) ([]*HourlyDistribution, error) {
	query := `
		SELECT
			EXTRACT(HOUR FROM requested_at)::int as hour,
			COUNT(*) as rides,
			COALESCE(AVG(final_fare), 0) as avg_fare,
			COALESCE(AVG(EXTRACT(EPOCH FROM (accepted_at - requested_at)) / 60), 0) as avg_wait_time
		FROM rides
		WHERE created_at >= $1
		  AND created_at <= $2
		GROUP BY hour
		ORDER BY hour ASC
	`

	rows, err := r.db.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get hourly distribution: %w", err)
	}
	defer rows.Close()

	results := make([]*HourlyDistribution, 0)
	for rows.Next() {
		hd := &HourlyDistribution{}
		err := rows.Scan(&hd.Hour, &hd.Rides, &hd.AvgFare, &hd.AvgWaitTime)
		if err != nil {
			return nil, fmt.Errorf("failed to scan hourly distribution: %w", err)
		}
		results = append(results, hd)
	}

	return results, nil
}

// GetDriverAnalytics retrieves overall driver performance analytics
func (r *Repository) GetDriverAnalytics(ctx context.Context, startDate, endDate time.Time) (*DriverAnalytics, error) {
	analytics := &DriverAnalytics{}

	// Get total active drivers and avg rides per driver
	query1 := `
		SELECT
			COUNT(DISTINCT d.id) as active_drivers,
			COALESCE(COUNT(r.id)::float / NULLIF(COUNT(DISTINCT d.id), 0), 0) as avg_rides_per_driver,
			COALESCE(AVG(d.rating), 0) as avg_rating
		FROM drivers d
		LEFT JOIN rides r ON d.id = r.driver_id
			AND r.created_at >= $1
			AND r.created_at <= $2
		WHERE d.is_available = true
	`

	err := r.db.QueryRow(ctx, query1, startDate, endDate).Scan(
		&analytics.TotalActiveDrivers,
		&analytics.AvgRidesPerDriver,
		&analytics.AvgRating,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver analytics: %w", err)
	}

	// Get cancellation rates
	query2 := `
		SELECT
			COALESCE(COUNT(*) FILTER (WHERE status = 'cancelled')::float / NULLIF(COUNT(*), 0) * 100, 0) as cancellation_rate,
			COALESCE(COUNT(*) FILTER (WHERE status IN ('accepted', 'in_progress', 'completed'))::float / NULLIF(COUNT(*), 0) * 100, 0) as acceptance_rate
		FROM rides
		WHERE driver_id IS NOT NULL
		  AND created_at >= $1
		  AND created_at <= $2
	`

	err = r.db.QueryRow(ctx, query2, startDate, endDate).Scan(
		&analytics.AvgCancellationRate,
		&analytics.AvgAcceptanceRate,
	)
	if err != nil {
		analytics.AvgCancellationRate = 0
		analytics.AvgAcceptanceRate = 0
	}

	// Get new drivers in period
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM drivers WHERE created_at >= $1 AND created_at <= $2
	`, startDate, endDate).Scan(&analytics.NewDrivers)
	if err != nil {
		analytics.NewDrivers = 0
	}

	return analytics, nil
}

// GetRiderGrowth retrieves rider growth and retention metrics
func (r *Repository) GetRiderGrowth(ctx context.Context, startDate, endDate time.Time) (*RiderGrowth, error) {
	growth := &RiderGrowth{}
	today := time.Now().Truncate(24 * time.Hour)

	// New riders in period
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM users
		WHERE role = 'rider'
		  AND created_at >= $1
		  AND created_at <= $2
	`, startDate, endDate).Scan(&growth.NewRiders)
	if err != nil {
		growth.NewRiders = 0
	}

	// Returning riders (made more than 1 ride in period)
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT rider_id) FROM (
			SELECT rider_id FROM rides
			WHERE created_at >= $1 AND created_at <= $2
			GROUP BY rider_id
			HAVING COUNT(*) > 1
		) as returning
	`, startDate, endDate).Scan(&growth.ReturningRiders)
	if err != nil {
		growth.ReturningRiders = 0
	}

	// Avg rides per rider
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(COUNT(*)::float / NULLIF(COUNT(DISTINCT rider_id), 0), 0)
		FROM rides
		WHERE created_at >= $1 AND created_at <= $2
	`, startDate, endDate).Scan(&growth.AvgRidesPerRider)
	if err != nil {
		growth.AvgRidesPerRider = 0
	}

	// First time riders today
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM users
		WHERE role = 'rider'
		  AND created_at >= $1
	`, today).Scan(&growth.FirstTimeRidersToday)
	if err != nil {
		growth.FirstTimeRidersToday = 0
	}

	// Calculate retention rate (riders who made a ride this period vs last period)
	var totalRiders int
	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'rider' AND is_active = true`).Scan(&totalRiders)
	if err == nil && totalRiders > 0 {
		growth.RetentionRate = float64(growth.ReturningRiders) / float64(totalRiders) * 100
	}

	// Lifetime value avg (total revenue / total riders)
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(final_fare) / NULLIF(COUNT(DISTINCT rider_id), 0), 0)
		FROM rides
		WHERE status = 'completed'
	`).Scan(&growth.LifetimeValueAvg)
	if err != nil {
		growth.LifetimeValueAvg = 0
	}

	return growth, nil
}

// GetRideMetrics retrieves quality of service metrics
func (r *Repository) GetRideMetrics(ctx context.Context, startDate, endDate time.Time) (*RideMetrics, error) {
	metrics := &RideMetrics{}

	query := `
		SELECT
			COALESCE(AVG(EXTRACT(EPOCH FROM (accepted_at - requested_at)) / 60) FILTER (WHERE accepted_at IS NOT NULL), 0) as avg_wait_time,
			COALESCE(AVG(actual_duration) FILTER (WHERE actual_duration IS NOT NULL), 0) as avg_duration,
			COALESCE(AVG(actual_distance) FILTER (WHERE actual_distance IS NOT NULL), 0) as avg_distance,
			COALESCE(COUNT(*) FILTER (WHERE status = 'cancelled')::float / NULLIF(COUNT(*), 0) * 100, 0) as cancellation_rate,
			COALESCE(COUNT(*) FILTER (WHERE status = 'cancelled' AND cancellation_reason LIKE '%rider%')::float / NULLIF(COUNT(*), 0) * 100, 0) as rider_cancel_rate,
			COALESCE(COUNT(*) FILTER (WHERE status = 'cancelled' AND cancellation_reason LIKE '%driver%')::float / NULLIF(COUNT(*), 0) * 100, 0) as driver_cancel_rate,
			COALESCE(COUNT(*) FILTER (WHERE surge_multiplier > 1.0)::float / NULLIF(COUNT(*), 0) * 100, 0) as surge_percentage,
			COALESCE(AVG(surge_multiplier) FILTER (WHERE surge_multiplier > 1.0), 1.0) as avg_surge
		FROM rides
		WHERE created_at >= $1
		  AND created_at <= $2
	`

	err := r.db.QueryRow(ctx, query, startDate, endDate).Scan(
		&metrics.AvgWaitTimeMinutes,
		&metrics.AvgRideDurationMinutes,
		&metrics.AvgDistanceKm,
		&metrics.CancellationRate,
		&metrics.RiderCancellationRate,
		&metrics.DriverCancellationRate,
		&metrics.SurgeRidesPercentage,
		&metrics.AvgSurgeMultiplier,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride metrics: %w", err)
	}

	return metrics, nil
}

// GetTopDriversDetailed retrieves top performing drivers with detailed metrics
func (r *Repository) GetTopDriversDetailed(ctx context.Context, startDate, endDate time.Time, limit int) ([]*TopDriver, error) {
	query := `
		SELECT
			d.id,
			COALESCE(u.first_name || ' ' || u.last_name, u.email) as name,
			COUNT(r.id) as total_rides,
			COALESCE(SUM(r.final_fare), 0) as total_revenue,
			COALESCE(d.rating, 0) as avg_rating,
			COALESCE(COUNT(*) FILTER (WHERE r.status IN ('accepted', 'in_progress', 'completed'))::float / NULLIF(COUNT(*), 0) * 100, 0) as acceptance_rate,
			COALESCE(d.total_rides::float * 0.5, 0) as online_hours
		FROM drivers d
		JOIN users u ON d.user_id = u.id
		LEFT JOIN rides r ON d.id = r.driver_id
			AND r.created_at >= $1
			AND r.created_at <= $2
		GROUP BY d.id, u.first_name, u.last_name, u.email, d.rating, d.total_rides
		HAVING COUNT(r.id) > 0
		ORDER BY total_revenue DESC
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, query, startDate, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top drivers: %w", err)
	}
	defer rows.Close()

	results := make([]*TopDriver, 0)
	for rows.Next() {
		driver := &TopDriver{}
		err := rows.Scan(
			&driver.DriverID,
			&driver.Name,
			&driver.TotalRides,
			&driver.TotalRevenue,
			&driver.AvgRating,
			&driver.AcceptanceRate,
			&driver.OnlineHours,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan top driver: %w", err)
		}
		results = append(results, driver)
	}

	return results, nil
}

// GetPeriodComparison compares metrics between two periods
func (r *Repository) GetPeriodComparison(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time) (*PeriodComparison, error) {
	comparison := &PeriodComparison{}

	// Get current period metrics
	var currentRevenue, currentRating float64
	var currentRides, currentNewRiders int

	err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(final_fare), 0),
			COUNT(*),
			COALESCE(AVG(rating), 0)
		FROM rides
		WHERE status = 'completed'
		  AND completed_at >= $1
		  AND completed_at <= $2
	`, currentStart, currentEnd).Scan(&currentRevenue, &currentRides, &currentRating)
	if err != nil {
		return nil, fmt.Errorf("failed to get current period metrics: %w", err)
	}

	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM users
		WHERE role = 'rider' AND created_at >= $1 AND created_at <= $2
	`, currentStart, currentEnd).Scan(&currentNewRiders)
	if err != nil {
		currentNewRiders = 0
	}

	// Get previous period metrics
	var previousRevenue, previousRating float64
	var previousRides, previousNewRiders int

	err = r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(final_fare), 0),
			COUNT(*),
			COALESCE(AVG(rating), 0)
		FROM rides
		WHERE status = 'completed'
		  AND completed_at >= $1
		  AND completed_at <= $2
	`, previousStart, previousEnd).Scan(&previousRevenue, &previousRides, &previousRating)
	if err != nil {
		previousRevenue = 0
		previousRides = 0
		previousRating = 0
	}

	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM users
		WHERE role = 'rider' AND created_at >= $1 AND created_at <= $2
	`, previousStart, previousEnd).Scan(&previousNewRiders)
	if err != nil {
		previousNewRiders = 0
	}

	// Calculate comparisons
	comparison.Revenue = ComparisonMetric{
		Current:       currentRevenue,
		Previous:      previousRevenue,
		ChangePercent: calculateChangePercent(currentRevenue, previousRevenue),
	}

	comparison.Rides = ComparisonMetric{
		Current:       float64(currentRides),
		Previous:      float64(previousRides),
		ChangePercent: calculateChangePercent(float64(currentRides), float64(previousRides)),
	}

	comparison.NewRiders = ComparisonMetric{
		Current:       float64(currentNewRiders),
		Previous:      float64(previousNewRiders),
		ChangePercent: calculateChangePercent(float64(currentNewRiders), float64(previousNewRiders)),
	}

	comparison.AvgRating = ComparisonMetric{
		Current:       currentRating,
		Previous:      previousRating,
		ChangePercent: calculateChangePercent(currentRating, previousRating),
	}

	return comparison, nil
}

func calculateChangePercent(current, previous float64) float64 {
	if previous == 0 {
		if current > 0 {
			return 100
		}
		return 0
	}
	return ((current - previous) / previous) * 100
}
