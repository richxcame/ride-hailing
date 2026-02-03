package ridehistory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles ride history data access
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new ride history repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// GetRiderHistory returns paginated ride history for a rider
func (r *Repository) GetRiderHistory(ctx context.Context, riderID uuid.UUID, filters *HistoryFilters, limit, offset int) ([]RideHistoryEntry, int, error) {
	where := []string{"r.rider_id = $1"}
	args := []interface{}{riderID}
	argIdx := 2

	if filters != nil {
		if filters.Status != nil {
			where = append(where, fmt.Sprintf("r.status = $%d", argIdx))
			args = append(args, *filters.Status)
			argIdx++
		}
		if filters.FromDate != nil {
			where = append(where, fmt.Sprintf("r.requested_at >= $%d", argIdx))
			args = append(args, *filters.FromDate)
			argIdx++
		}
		if filters.ToDate != nil {
			where = append(where, fmt.Sprintf("r.requested_at < $%d", argIdx))
			args = append(args, *filters.ToDate)
			argIdx++
		}
		if filters.MinFare != nil {
			where = append(where, fmt.Sprintf("r.total_fare >= $%d", argIdx))
			args = append(args, *filters.MinFare)
			argIdx++
		}
		if filters.MaxFare != nil {
			where = append(where, fmt.Sprintf("r.total_fare <= $%d", argIdx))
			args = append(args, *filters.MaxFare)
			argIdx++
		}
	}

	whereClause := strings.Join(where, " AND ")

	// Count
	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM rides r WHERE %s`, whereClause)
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Query
	query := fmt.Sprintf(`
		SELECT r.id, r.rider_id, r.driver_id, r.status,
			r.pickup_address, r.pickup_latitude, r.pickup_longitude,
			r.dropoff_address, r.dropoff_latitude, r.dropoff_longitude,
			COALESCE(r.distance, 0), COALESCE(r.duration, 0),
			COALESCE(r.base_fare, 0), COALESCE(r.distance_fare, 0),
			COALESCE(r.time_fare, 0), COALESCE(r.surge_multiplier, 1),
			COALESCE(r.surge_amount, 0), COALESCE(r.toll_fees, 0),
			COALESCE(r.wait_time_charge, 0), COALESCE(r.tip_amount, 0),
			COALESCE(r.discount_amount, 0), r.promo_code,
			COALESCE(r.total_fare, 0), COALESCE(r.currency, 'USD'),
			COALESCE(r.payment_method, 'card'), COALESCE(r.payment_status, 'completed'),
			r.rider_rating, r.driver_rating,
			r.route_polyline,
			r.cancelled_by, r.cancel_reason, COALESCE(r.cancellation_fee, 0),
			r.requested_at, r.accepted_at, r.picked_up_at, r.completed_at, r.cancelled_at
		FROM rides r
		WHERE %s
		ORDER BY r.requested_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var rides []RideHistoryEntry
	for rows.Next() {
		e := RideHistoryEntry{}
		if err := rows.Scan(
			&e.ID, &e.RiderID, &e.DriverID, &e.Status,
			&e.PickupAddress, &e.PickupLatitude, &e.PickupLongitude,
			&e.DropoffAddress, &e.DropoffLatitude, &e.DropoffLongitude,
			&e.Distance, &e.Duration,
			&e.BaseFare, &e.DistanceFare,
			&e.TimeFare, &e.SurgeMultiplier,
			&e.SurgeAmount, &e.TollFees,
			&e.WaitTimeCharge, &e.TipAmount,
			&e.DiscountAmount, &e.PromoCode,
			&e.TotalFare, &e.Currency,
			&e.PaymentMethod, &e.PaymentStatus,
			&e.RiderRating, &e.DriverRating,
			&e.RoutePolyline,
			&e.CancelledBy, &e.CancelReason, &e.CancellationFee,
			&e.RequestedAt, &e.AcceptedAt, &e.PickedUpAt, &e.CompletedAt, &e.CancelledAt,
		); err != nil {
			return nil, 0, err
		}
		rides = append(rides, e)
	}
	return rides, total, nil
}

// GetDriverHistory returns paginated ride history for a driver
func (r *Repository) GetDriverHistory(ctx context.Context, driverID uuid.UUID, filters *HistoryFilters, limit, offset int) ([]RideHistoryEntry, int, error) {
	where := []string{"r.driver_id = $1"}
	args := []interface{}{driverID}
	argIdx := 2

	if filters != nil {
		if filters.Status != nil {
			where = append(where, fmt.Sprintf("r.status = $%d", argIdx))
			args = append(args, *filters.Status)
			argIdx++
		}
		if filters.FromDate != nil {
			where = append(where, fmt.Sprintf("r.requested_at >= $%d", argIdx))
			args = append(args, *filters.FromDate)
			argIdx++
		}
		if filters.ToDate != nil {
			where = append(where, fmt.Sprintf("r.requested_at < $%d", argIdx))
			args = append(args, *filters.ToDate)
			argIdx++
		}
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM rides r WHERE %s`, whereClause)
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT r.id, r.rider_id, r.driver_id, r.status,
			r.pickup_address, r.pickup_latitude, r.pickup_longitude,
			r.dropoff_address, r.dropoff_latitude, r.dropoff_longitude,
			COALESCE(r.distance, 0), COALESCE(r.duration, 0),
			COALESCE(r.base_fare, 0), COALESCE(r.distance_fare, 0),
			COALESCE(r.time_fare, 0), COALESCE(r.surge_multiplier, 1),
			COALESCE(r.surge_amount, 0), COALESCE(r.toll_fees, 0),
			COALESCE(r.wait_time_charge, 0), COALESCE(r.tip_amount, 0),
			COALESCE(r.discount_amount, 0), r.promo_code,
			COALESCE(r.total_fare, 0), COALESCE(r.currency, 'USD'),
			COALESCE(r.payment_method, 'card'), COALESCE(r.payment_status, 'completed'),
			r.rider_rating, r.driver_rating,
			r.route_polyline,
			r.cancelled_by, r.cancel_reason, COALESCE(r.cancellation_fee, 0),
			r.requested_at, r.accepted_at, r.picked_up_at, r.completed_at, r.cancelled_at
		FROM rides r
		WHERE %s
		ORDER BY r.requested_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var rides []RideHistoryEntry
	for rows.Next() {
		e := RideHistoryEntry{}
		if err := rows.Scan(
			&e.ID, &e.RiderID, &e.DriverID, &e.Status,
			&e.PickupAddress, &e.PickupLatitude, &e.PickupLongitude,
			&e.DropoffAddress, &e.DropoffLatitude, &e.DropoffLongitude,
			&e.Distance, &e.Duration,
			&e.BaseFare, &e.DistanceFare,
			&e.TimeFare, &e.SurgeMultiplier,
			&e.SurgeAmount, &e.TollFees,
			&e.WaitTimeCharge, &e.TipAmount,
			&e.DiscountAmount, &e.PromoCode,
			&e.TotalFare, &e.Currency,
			&e.PaymentMethod, &e.PaymentStatus,
			&e.RiderRating, &e.DriverRating,
			&e.RoutePolyline,
			&e.CancelledBy, &e.CancelReason, &e.CancellationFee,
			&e.RequestedAt, &e.AcceptedAt, &e.PickedUpAt, &e.CompletedAt, &e.CancelledAt,
		); err != nil {
			return nil, 0, err
		}
		rides = append(rides, e)
	}
	return rides, total, nil
}

// GetRideByID returns a single ride by ID
func (r *Repository) GetRideByID(ctx context.Context, rideID uuid.UUID) (*RideHistoryEntry, error) {
	e := &RideHistoryEntry{}
	err := r.db.QueryRow(ctx, `
		SELECT r.id, r.rider_id, r.driver_id, r.status,
			r.pickup_address, r.pickup_latitude, r.pickup_longitude,
			r.dropoff_address, r.dropoff_latitude, r.dropoff_longitude,
			COALESCE(r.distance, 0), COALESCE(r.duration, 0),
			COALESCE(r.base_fare, 0), COALESCE(r.distance_fare, 0),
			COALESCE(r.time_fare, 0), COALESCE(r.surge_multiplier, 1),
			COALESCE(r.surge_amount, 0), COALESCE(r.toll_fees, 0),
			COALESCE(r.wait_time_charge, 0), COALESCE(r.tip_amount, 0),
			COALESCE(r.discount_amount, 0), r.promo_code,
			COALESCE(r.total_fare, 0), COALESCE(r.currency, 'USD'),
			COALESCE(r.payment_method, 'card'), COALESCE(r.payment_status, 'completed'),
			r.rider_rating, r.driver_rating,
			r.route_polyline,
			r.cancelled_by, r.cancel_reason, COALESCE(r.cancellation_fee, 0),
			r.requested_at, r.accepted_at, r.picked_up_at, r.completed_at, r.cancelled_at
		FROM rides r
		WHERE r.id = $1`, rideID,
	).Scan(
		&e.ID, &e.RiderID, &e.DriverID, &e.Status,
		&e.PickupAddress, &e.PickupLatitude, &e.PickupLongitude,
		&e.DropoffAddress, &e.DropoffLatitude, &e.DropoffLongitude,
		&e.Distance, &e.Duration,
		&e.BaseFare, &e.DistanceFare,
		&e.TimeFare, &e.SurgeMultiplier,
		&e.SurgeAmount, &e.TollFees,
		&e.WaitTimeCharge, &e.TipAmount,
		&e.DiscountAmount, &e.PromoCode,
		&e.TotalFare, &e.Currency,
		&e.PaymentMethod, &e.PaymentStatus,
		&e.RiderRating, &e.DriverRating,
		&e.RoutePolyline,
		&e.CancelledBy, &e.CancelReason, &e.CancellationFee,
		&e.RequestedAt, &e.AcceptedAt, &e.PickedUpAt, &e.CompletedAt, &e.CancelledAt,
	)
	if err != nil {
		return nil, err
	}
	return e, nil
}

// GetRiderStats returns aggregated stats for a rider
func (r *Repository) GetRiderStats(ctx context.Context, riderID uuid.UUID, from, to time.Time) (*RideStats, error) {
	stats := &RideStats{Currency: "USD"}
	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(CASE WHEN status = 'completed' THEN 1 END),
			COUNT(CASE WHEN status = 'cancelled' THEN 1 END),
			COALESCE(SUM(CASE WHEN status = 'completed' THEN total_fare ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'completed' THEN distance ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'completed' THEN duration ELSE 0 END), 0),
			COALESCE(AVG(CASE WHEN status = 'completed' THEN total_fare END), 0),
			COALESCE(AVG(CASE WHEN status = 'completed' THEN distance END), 0),
			COALESCE(AVG(rider_rating), 0)
		FROM rides
		WHERE rider_id = $1 AND requested_at >= $2 AND requested_at < $3`,
		riderID, from, to,
	).Scan(
		&stats.TotalRides, &stats.CompletedRides, &stats.CancelledRides,
		&stats.TotalSpent, &stats.TotalDistance, &stats.TotalDuration,
		&stats.AverageFare, &stats.AverageDistance, &stats.AverageRating,
	)
	return stats, err
}

// GetFrequentRoutes returns commonly taken routes
func (r *Repository) GetFrequentRoutes(ctx context.Context, riderID uuid.UUID, limit int) ([]FrequentRoute, error) {
	rows, err := r.db.Query(ctx, `
		SELECT pickup_address, dropoff_address,
			COUNT(*) as ride_count,
			AVG(total_fare) as avg_fare,
			MAX(requested_at)::text as last_ride
		FROM rides
		WHERE rider_id = $1 AND status = 'completed'
		GROUP BY pickup_address, dropoff_address
		HAVING COUNT(*) >= 2
		ORDER BY ride_count DESC
		LIMIT $2`, riderID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []FrequentRoute
	for rows.Next() {
		fr := FrequentRoute{}
		if err := rows.Scan(
			&fr.PickupAddress, &fr.DropoffAddress,
			&fr.RideCount, &fr.AverageFare, &fr.LastRideAt,
		); err != nil {
			return nil, err
		}
		routes = append(routes, fr)
	}
	return routes, nil
}
