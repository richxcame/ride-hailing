package cancellation

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles cancellation data access
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new cancellation repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateCancellationRecord records a cancellation
func (r *Repository) CreateCancellationRecord(ctx context.Context, rec *CancellationRecord) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO cancellation_records (
			id, ride_id, rider_id, driver_id, cancelled_by,
			reason_code, reason_text, fee_amount, fee_waived, waiver_reason,
			minutes_since_request, minutes_since_accept, ride_status_at_cancel,
			pickup_lat, pickup_lng, cancelled_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
		rec.ID, rec.RideID, rec.RiderID, rec.DriverID, rec.CancelledBy,
		rec.ReasonCode, rec.ReasonText, rec.FeeAmount, rec.FeeWaived, rec.WaiverReason,
		rec.MinutesSinceRequest, rec.MinutesSinceAccept, rec.RideStatus,
		rec.PickupLat, rec.PickupLng, rec.CancelledAt, rec.CreatedAt,
	)
	return err
}

// GetCancellationByRideID retrieves the cancellation record for a ride
func (r *Repository) GetCancellationByRideID(ctx context.Context, rideID uuid.UUID) (*CancellationRecord, error) {
	rec := &CancellationRecord{}
	err := r.db.QueryRow(ctx, `
		SELECT id, ride_id, rider_id, driver_id, cancelled_by,
			reason_code, reason_text, fee_amount, fee_waived, waiver_reason,
			minutes_since_request, minutes_since_accept, ride_status_at_cancel,
			pickup_lat, pickup_lng, cancelled_at, created_at
		FROM cancellation_records WHERE ride_id = $1`, rideID,
	).Scan(
		&rec.ID, &rec.RideID, &rec.RiderID, &rec.DriverID, &rec.CancelledBy,
		&rec.ReasonCode, &rec.ReasonText, &rec.FeeAmount, &rec.FeeWaived, &rec.WaiverReason,
		&rec.MinutesSinceRequest, &rec.MinutesSinceAccept, &rec.RideStatus,
		&rec.PickupLat, &rec.PickupLng, &rec.CancelledAt, &rec.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

// GetCancellationByID retrieves a cancellation record by its ID
func (r *Repository) GetCancellationByID(ctx context.Context, id uuid.UUID) (*CancellationRecord, error) {
	rec := &CancellationRecord{}
	err := r.db.QueryRow(ctx, `
		SELECT id, ride_id, rider_id, driver_id, cancelled_by,
			reason_code, reason_text, fee_amount, fee_waived, waiver_reason,
			minutes_since_request, minutes_since_accept, ride_status_at_cancel,
			pickup_lat, pickup_lng, cancelled_at, created_at
		FROM cancellation_records WHERE id = $1`, id,
	).Scan(
		&rec.ID, &rec.RideID, &rec.RiderID, &rec.DriverID, &rec.CancelledBy,
		&rec.ReasonCode, &rec.ReasonText, &rec.FeeAmount, &rec.FeeWaived, &rec.WaiverReason,
		&rec.MinutesSinceRequest, &rec.MinutesSinceAccept, &rec.RideStatus,
		&rec.PickupLat, &rec.PickupLng, &rec.CancelledAt, &rec.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

// CountUserCancellationsToday returns how many times a user cancelled today
func (r *Repository) CountUserCancellationsToday(ctx context.Context, userID uuid.UUID, cancelledBy CancelledBy) (int, error) {
	today := time.Now().Truncate(24 * time.Hour)
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM cancellation_records
		WHERE (rider_id = $1 OR driver_id = $1)
			AND cancelled_by = $2
			AND cancelled_at >= $3`,
		userID, cancelledBy, today,
	).Scan(&count)
	return count, err
}

// CountUserCancellationsThisWeek returns how many times a user cancelled this week
func (r *Repository) CountUserCancellationsThisWeek(ctx context.Context, userID uuid.UUID, cancelledBy CancelledBy) (int, error) {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	startOfWeek := now.AddDate(0, 0, -(weekday - 1)).Truncate(24 * time.Hour)
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM cancellation_records
		WHERE (rider_id = $1 OR driver_id = $1)
			AND cancelled_by = $2
			AND cancelled_at >= $3`,
		userID, cancelledBy, startOfWeek,
	).Scan(&count)
	return count, err
}

// GetUserCancellationStats returns aggregated stats for a user
func (r *Repository) GetUserCancellationStats(ctx context.Context, userID uuid.UUID) (*UserCancellationStats, error) {
	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	startOfWeek := now.AddDate(0, 0, -(weekday - 1)).Truncate(24 * time.Hour)
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	stats := &UserCancellationStats{UserID: userID}

	// Total cancellations
	r.db.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(CASE WHEN fee_waived = false THEN fee_amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN fee_waived = true THEN fee_amount ELSE 0 END), 0)
		FROM cancellation_records
		WHERE (rider_id = $1 OR driver_id = $1) AND cancelled_by != 'system'`,
		userID,
	).Scan(&stats.TotalCancellations, &stats.TotalFeesCharged, &stats.TotalFeesWaived)

	// Today
	r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM cancellation_records
		WHERE (rider_id = $1 OR driver_id = $1) AND cancelled_by != 'system' AND cancelled_at >= $2`,
		userID, today,
	).Scan(&stats.CancellationsToday)

	// This week
	r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM cancellation_records
		WHERE (rider_id = $1 OR driver_id = $1) AND cancelled_by != 'system' AND cancelled_at >= $2`,
		userID, startOfWeek,
	).Scan(&stats.CancellationsThisWeek)

	// This month
	r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM cancellation_records
		WHERE (rider_id = $1 OR driver_id = $1) AND cancelled_by != 'system' AND cancelled_at >= $2`,
		userID, startOfMonth,
	).Scan(&stats.CancellationsThisMonth)

	// Cancellation rate (last 30 days)
	var totalRides int
	r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM rides
		WHERE (rider_id = $1 OR driver_id = $1) AND created_at >= $2`,
		userID, now.AddDate(0, 0, -30),
	).Scan(&totalRides)

	var recentCancels int
	r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM cancellation_records
		WHERE (rider_id = $1 OR driver_id = $1) AND cancelled_by != 'system' AND cancelled_at >= $2`,
		userID, now.AddDate(0, 0, -30),
	).Scan(&recentCancels)

	if totalRides > 0 {
		stats.CancellationRate = float64(recentCancels) / float64(totalRides) * 100
	}

	// Last cancellation
	r.db.QueryRow(ctx, `
		SELECT cancelled_at FROM cancellation_records
		WHERE (rider_id = $1 OR driver_id = $1) AND cancelled_by != 'system'
		ORDER BY cancelled_at DESC LIMIT 1`,
		userID,
	).Scan(&stats.LastCancellationAt)

	return stats, nil
}

// GetUserCancellationHistory returns recent cancellation records for a user
func (r *Repository) GetUserCancellationHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]CancellationRecord, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM cancellation_records
		WHERE rider_id = $1 OR driver_id = $1`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, ride_id, rider_id, driver_id, cancelled_by,
			reason_code, reason_text, fee_amount, fee_waived, waiver_reason,
			minutes_since_request, minutes_since_accept, ride_status_at_cancel,
			pickup_lat, pickup_lng, cancelled_at, created_at
		FROM cancellation_records
		WHERE rider_id = $1 OR driver_id = $1
		ORDER BY cancelled_at DESC
		LIMIT $2 OFFSET $3`, userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var records []CancellationRecord
	for rows.Next() {
		rec := CancellationRecord{}
		if err := rows.Scan(
			&rec.ID, &rec.RideID, &rec.RiderID, &rec.DriverID, &rec.CancelledBy,
			&rec.ReasonCode, &rec.ReasonText, &rec.FeeAmount, &rec.FeeWaived, &rec.WaiverReason,
			&rec.MinutesSinceRequest, &rec.MinutesSinceAccept, &rec.RideStatus,
			&rec.PickupLat, &rec.PickupLng, &rec.CancelledAt, &rec.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		records = append(records, rec)
	}
	if records == nil {
		records = []CancellationRecord{}
	}
	return records, total, nil
}

// WaiveCancellationFee marks a cancellation fee as waived
func (r *Repository) WaiveCancellationFee(ctx context.Context, id uuid.UUID, reason string) error {
	waiver := FeeWaiverReason(WaiverAdminOverride)
	_, err := r.db.Exec(ctx, `
		UPDATE cancellation_records
		SET fee_waived = true, waiver_reason = $2, reason_text = COALESCE(reason_text, '') || ' [Admin waived: ' || $3 || ']'
		WHERE id = $1`,
		id, waiver, reason,
	)
	return err
}

// GetDefaultPolicy retrieves the default cancellation policy
func (r *Repository) GetDefaultPolicy(ctx context.Context) (*CancellationPolicy, error) {
	policy := &CancellationPolicy{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name, free_cancel_window_minutes, max_free_cancels_per_day,
			max_free_cancels_per_week, driver_no_show_minutes, rider_no_show_minutes,
			driver_penalty_threshold, rider_penalty_threshold,
			is_default, is_active, created_at, updated_at
		FROM cancellation_policies
		WHERE is_default = true AND is_active = true
		LIMIT 1`,
	).Scan(
		&policy.ID, &policy.Name, &policy.FreeCancelWindowMinutes,
		&policy.MaxFreeCancelsPerDay, &policy.MaxFreeCancelsPerWeek,
		&policy.DriverNoShowMinutes, &policy.RiderNoShowMinutes,
		&policy.DriverPenaltyThreshold, &policy.RiderPenaltyThreshold,
		&policy.IsDefault, &policy.IsActive, &policy.CreatedAt, &policy.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return policy, nil
}

// GetCancellationStats returns aggregated cancellation analytics
func (r *Repository) GetCancellationStats(ctx context.Context, from, to time.Time) (*CancellationStatsResponse, error) {
	stats := &CancellationStatsResponse{}

	// Overall counts
	r.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE cancelled_by = 'rider'),
			COUNT(*) FILTER (WHERE cancelled_by = 'driver'),
			COUNT(*) FILTER (WHERE cancelled_by = 'system'),
			COALESCE(SUM(CASE WHEN fee_waived = false THEN fee_amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN fee_waived = true THEN fee_amount ELSE 0 END), 0),
			COALESCE(AVG(minutes_since_request), 0)
		FROM cancellation_records
		WHERE cancelled_at >= $1 AND cancelled_at < $2`,
		from, to,
	).Scan(
		&stats.TotalCancellations, &stats.RiderCancellations,
		&stats.DriverCancellations, &stats.SystemCancellations,
		&stats.TotalFeesCollected, &stats.TotalFeesWaived,
		&stats.AverageMinutesToCancel,
	)

	// Cancellation rate
	var totalRides int
	r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM rides WHERE created_at >= $1 AND created_at < $2`,
		from, to,
	).Scan(&totalRides)
	if totalRides > 0 {
		stats.CancellationRate = float64(stats.TotalCancellations) / float64(totalRides) * 100
	}

	// Top reasons
	rows, err := r.db.Query(ctx, `
		SELECT reason_code, COUNT(*) as cnt
		FROM cancellation_records
		WHERE cancelled_at >= $1 AND cancelled_at < $2
		GROUP BY reason_code
		ORDER BY cnt DESC
		LIMIT 10`, from, to,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			tr := TopCancellationReason{}
			if err := rows.Scan(&tr.ReasonCode, &tr.Count); err == nil {
				if stats.TotalCancellations > 0 {
					tr.Percentage = float64(tr.Count) / float64(stats.TotalCancellations) * 100
				}
				stats.TopReasons = append(stats.TopReasons, tr)
			}
		}
	}
	if stats.TopReasons == nil {
		stats.TopReasons = []TopCancellationReason{}
	}

	return stats, nil
}
