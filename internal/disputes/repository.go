package disputes

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles dispute data access
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new dispute repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// DISPUTES
// ========================================

// CreateDispute creates a new dispute
func (r *Repository) CreateDispute(ctx context.Context, d *Dispute) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO fare_disputes (
			id, ride_id, user_id, driver_id, dispute_number,
			reason, description, status, original_fare, disputed_amount,
			evidence, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		d.ID, d.RideID, d.UserID, d.DriverID, d.DisputeNumber,
		d.Reason, d.Description, d.Status, d.OriginalFare, d.DisputedAmount,
		d.Evidence, d.CreatedAt, d.UpdatedAt,
	)
	return err
}

// GetDisputeByID retrieves a dispute by ID
func (r *Repository) GetDisputeByID(ctx context.Context, id uuid.UUID) (*Dispute, error) {
	d := &Dispute{}
	err := r.db.QueryRow(ctx, `
		SELECT id, ride_id, user_id, driver_id, dispute_number,
			reason, description, status, original_fare, disputed_amount,
			refund_amount, resolution_type, resolution_note, resolved_by,
			evidence, resolved_at, created_at, updated_at
		FROM fare_disputes WHERE id = $1`, id,
	).Scan(
		&d.ID, &d.RideID, &d.UserID, &d.DriverID, &d.DisputeNumber,
		&d.Reason, &d.Description, &d.Status, &d.OriginalFare, &d.DisputedAmount,
		&d.RefundAmount, &d.ResolutionType, &d.ResolutionNote, &d.ResolvedBy,
		&d.Evidence, &d.ResolvedAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// GetDisputeByRideAndUser checks if a user already has an open dispute for a ride
func (r *Repository) GetDisputeByRideAndUser(ctx context.Context, rideID, userID uuid.UUID) (*Dispute, error) {
	d := &Dispute{}
	err := r.db.QueryRow(ctx, `
		SELECT id, ride_id, user_id, driver_id, dispute_number,
			reason, description, status, original_fare, disputed_amount,
			refund_amount, resolution_type, resolution_note, resolved_by,
			evidence, resolved_at, created_at, updated_at
		FROM fare_disputes
		WHERE ride_id = $1 AND user_id = $2 AND status NOT IN ('closed', 'rejected')
		LIMIT 1`, rideID, userID,
	).Scan(
		&d.ID, &d.RideID, &d.UserID, &d.DriverID, &d.DisputeNumber,
		&d.Reason, &d.Description, &d.Status, &d.OriginalFare, &d.DisputedAmount,
		&d.RefundAmount, &d.ResolutionType, &d.ResolutionNote, &d.ResolvedBy,
		&d.Evidence, &d.ResolvedAt, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// GetUserDisputes returns disputes for a user
func (r *Repository) GetUserDisputes(ctx context.Context, userID uuid.UUID, status *DisputeStatus, limit, offset int) ([]DisputeSummary, int, error) {
	countQuery := `SELECT COUNT(*) FROM fare_disputes WHERE user_id = $1`
	query := `
		SELECT id, dispute_number, ride_id, reason, status,
			original_fare, disputed_amount, refund_amount,
			created_at, updated_at
		FROM fare_disputes WHERE user_id = $1`

	args := []interface{}{userID}
	argIdx := 2

	if status != nil {
		filter := fmt.Sprintf(" AND status = $%d", argIdx)
		countQuery += filter
		query += filter
		args = append(args, *status)
		argIdx++
	}

	var total int
	r.db.QueryRow(ctx, countQuery, args...).Scan(&total)

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var disputes []DisputeSummary
	for rows.Next() {
		ds := DisputeSummary{}
		if err := rows.Scan(
			&ds.ID, &ds.DisputeNumber, &ds.RideID, &ds.Reason, &ds.Status,
			&ds.OriginalFare, &ds.DisputedAmount, &ds.RefundAmount,
			&ds.CreatedAt, &ds.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		disputes = append(disputes, ds)
	}
	if disputes == nil {
		disputes = []DisputeSummary{}
	}
	return disputes, total, nil
}

// ResolveDispute resolves a dispute
func (r *Repository) ResolveDispute(ctx context.Context, id uuid.UUID, status DisputeStatus, resType ResolutionType, refundAmount *float64, note string, resolvedBy uuid.UUID) error {
	now := time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE fare_disputes
		SET status = $2, resolution_type = $3, refund_amount = $4,
			resolution_note = $5, resolved_by = $6, resolved_at = $7, updated_at = $7
		WHERE id = $1`,
		id, status, resType, refundAmount, note, resolvedBy, now,
	)
	return err
}

// UpdateDisputeStatus updates the status of a dispute
func (r *Repository) UpdateDisputeStatus(ctx context.Context, id uuid.UUID, status DisputeStatus) error {
	_, err := r.db.Exec(ctx, `
		UPDATE fare_disputes SET status = $2, updated_at = $3 WHERE id = $1`,
		id, status, time.Now(),
	)
	return err
}

// ========================================
// COMMENTS
// ========================================

// CreateComment creates a dispute comment
func (r *Repository) CreateComment(ctx context.Context, c *DisputeComment) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO fare_dispute_comments (id, dispute_id, user_id, user_role, comment, is_internal, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		c.ID, c.DisputeID, c.UserID, c.UserRole, c.Comment, c.IsInternal, c.CreatedAt,
	)
	return err
}

// GetCommentsByDispute returns comments for a dispute
func (r *Repository) GetCommentsByDispute(ctx context.Context, disputeID uuid.UUID, includeInternal bool) ([]DisputeComment, error) {
	query := `
		SELECT id, dispute_id, user_id, user_role, comment, is_internal, created_at
		FROM fare_dispute_comments
		WHERE dispute_id = $1`

	if !includeInternal {
		query += " AND is_internal = false"
	}
	query += " ORDER BY created_at ASC"

	rows, err := r.db.Query(ctx, query, disputeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []DisputeComment
	for rows.Next() {
		c := DisputeComment{}
		if err := rows.Scan(
			&c.ID, &c.DisputeID, &c.UserID, &c.UserRole,
			&c.Comment, &c.IsInternal, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	if comments == nil {
		comments = []DisputeComment{}
	}
	return comments, nil
}

// ========================================
// RIDE CONTEXT
// ========================================

// GetRideContext returns ride details for a dispute
func (r *Repository) GetRideContext(ctx context.Context, rideID uuid.UUID) (*RideContext, error) {
	rc := &RideContext{}
	err := r.db.QueryRow(ctx, `
		SELECT id, estimated_fare, final_fare, estimated_distance, actual_distance,
			estimated_duration, actual_duration, surge_multiplier,
			pickup_address, dropoff_address, requested_at, completed_at
		FROM rides WHERE id = $1`, rideID,
	).Scan(
		&rc.RideID, &rc.EstimatedFare, &rc.FinalFare, &rc.EstimatedDistance,
		&rc.ActualDistance, &rc.EstimatedDuration, &rc.ActualDuration,
		&rc.SurgeMultiplier, &rc.PickupAddress, &rc.DropoffAddress,
		&rc.RequestedAt, &rc.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	return rc, nil
}

// ========================================
// ADMIN STATS
// ========================================

// GetAllDisputes returns disputes with filters (admin)
func (r *Repository) GetAllDisputes(ctx context.Context, status *DisputeStatus, reason *DisputeReason, limit, offset int) ([]DisputeSummary, int, error) {
	countQuery := `SELECT COUNT(*) FROM fare_disputes WHERE 1=1`
	query := `
		SELECT id, dispute_number, ride_id, reason, status,
			original_fare, disputed_amount, refund_amount,
			created_at, updated_at
		FROM fare_disputes WHERE 1=1`

	args := []interface{}{}
	argIdx := 1

	if status != nil {
		filter := fmt.Sprintf(" AND status = $%d", argIdx)
		countQuery += filter
		query += filter
		args = append(args, *status)
		argIdx++
	}
	if reason != nil {
		filter := fmt.Sprintf(" AND reason = $%d", argIdx)
		countQuery += filter
		query += filter
		args = append(args, *reason)
		argIdx++
	}

	var total int
	r.db.QueryRow(ctx, countQuery, args...).Scan(&total)

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var disputes []DisputeSummary
	for rows.Next() {
		ds := DisputeSummary{}
		if err := rows.Scan(
			&ds.ID, &ds.DisputeNumber, &ds.RideID, &ds.Reason, &ds.Status,
			&ds.OriginalFare, &ds.DisputedAmount, &ds.RefundAmount,
			&ds.CreatedAt, &ds.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		disputes = append(disputes, ds)
	}
	if disputes == nil {
		disputes = []DisputeSummary{}
	}
	return disputes, total, nil
}

// GetDisputeStats returns aggregated dispute analytics
func (r *Repository) GetDisputeStats(ctx context.Context, from, to time.Time) (*DisputeStats, error) {
	stats := &DisputeStats{}

	r.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'pending'),
			COUNT(*) FILTER (WHERE status = 'reviewing'),
			COUNT(*) FILTER (WHERE status IN ('approved', 'partial_refund')),
			COUNT(*) FILTER (WHERE status = 'rejected'),
			COALESCE(SUM(CASE WHEN refund_amount IS NOT NULL THEN refund_amount ELSE 0 END), 0),
			COALESCE(SUM(disputed_amount), 0),
			COALESCE(AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 3600) FILTER (WHERE resolved_at IS NOT NULL), 0)
		FROM fare_disputes
		WHERE created_at >= $1 AND created_at < $2`,
		from, to,
	).Scan(
		&stats.TotalDisputes, &stats.PendingDisputes, &stats.ReviewingDisputes,
		&stats.ApprovedDisputes, &stats.RejectedDisputes,
		&stats.TotalRefunded, &stats.TotalDisputed, &stats.AvgResolutionHours,
	)

	// Dispute rate
	var totalRides int
	r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM rides WHERE created_at >= $1 AND created_at < $2`,
		from, to,
	).Scan(&totalRides)
	if totalRides > 0 {
		stats.DisputeRate = float64(stats.TotalDisputes) / float64(totalRides) * 100
	}

	// By reason
	rows, err := r.db.Query(ctx, `
		SELECT reason, COUNT(*) as cnt
		FROM fare_disputes
		WHERE created_at >= $1 AND created_at < $2
		GROUP BY reason
		ORDER BY cnt DESC`, from, to,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			rc := ReasonCount{}
			if rows.Scan(&rc.Reason, &rc.Count) == nil {
				if stats.TotalDisputes > 0 {
					rc.Percentage = float64(rc.Count) / float64(stats.TotalDisputes) * 100
				}
				stats.ByReason = append(stats.ByReason, rc)
			}
		}
	}
	if stats.ByReason == nil {
		stats.ByReason = []ReasonCount{}
	}

	return stats, nil
}
