package tips

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles tip data access
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new tips repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateTip creates a new tip record
func (r *Repository) CreateTip(ctx context.Context, t *Tip) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO tips (
			id, ride_id, rider_id, driver_id, amount,
			currency, status, message, is_anonymous,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		t.ID, t.RideID, t.RiderID, t.DriverID, t.Amount,
		t.Currency, t.Status, t.Message, t.IsAnonymous,
		t.CreatedAt, t.UpdatedAt,
	)
	return err
}

// GetTipByID retrieves a tip by ID
func (r *Repository) GetTipByID(ctx context.Context, id uuid.UUID) (*Tip, error) {
	t := &Tip{}
	err := r.db.QueryRow(ctx, `
		SELECT id, ride_id, rider_id, driver_id, amount,
			currency, status, message, is_anonymous,
			created_at, updated_at
		FROM tips WHERE id = $1`, id,
	).Scan(
		&t.ID, &t.RideID, &t.RiderID, &t.DriverID, &t.Amount,
		&t.Currency, &t.Status, &t.Message, &t.IsAnonymous,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// GetTipByRideAndRider checks if a rider already tipped for a ride
func (r *Repository) GetTipByRideAndRider(ctx context.Context, rideID, riderID uuid.UUID) (*Tip, error) {
	t := &Tip{}
	err := r.db.QueryRow(ctx, `
		SELECT id, ride_id, rider_id, driver_id, amount,
			currency, status, message, is_anonymous,
			created_at, updated_at
		FROM tips
		WHERE ride_id = $1 AND rider_id = $2 AND status != 'refunded'`, rideID, riderID,
	).Scan(
		&t.ID, &t.RideID, &t.RiderID, &t.DriverID, &t.Amount,
		&t.Currency, &t.Status, &t.Message, &t.IsAnonymous,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// UpdateTipStatus updates the status of a tip
func (r *Repository) UpdateTipStatus(ctx context.Context, tipID uuid.UUID, status TipStatus) error {
	_, err := r.db.Exec(ctx, `
		UPDATE tips SET status = $2, updated_at = NOW()
		WHERE id = $1`,
		tipID, status,
	)
	return err
}

// GetDriverTipSummary returns aggregated tip stats for a driver
func (r *Repository) GetDriverTipSummary(ctx context.Context, driverID uuid.UUID, from, to time.Time) (*DriverTipSummary, error) {
	s := &DriverTipSummary{Currency: "USD"}
	err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(amount), 0),
			COUNT(*),
			COALESCE(AVG(amount), 0),
			COALESCE(MAX(amount), 0)
		FROM tips
		WHERE driver_id = $1
			AND status = 'completed'
			AND created_at >= $2 AND created_at < $3`,
		driverID, from, to,
	).Scan(&s.TotalTips, &s.TipCount, &s.AverageTip, &s.HighestTip)
	return s, err
}

// GetTipsByRider returns tips given by a rider
func (r *Repository) GetTipsByRider(ctx context.Context, riderID uuid.UUID, limit, offset int) ([]Tip, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM tips WHERE rider_id = $1`, riderID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, ride_id, rider_id, driver_id, amount,
			currency, status, message, is_anonymous,
			created_at, updated_at
		FROM tips
		WHERE rider_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		riderID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tips []Tip
	for rows.Next() {
		t := Tip{}
		if err := rows.Scan(
			&t.ID, &t.RideID, &t.RiderID, &t.DriverID, &t.Amount,
			&t.Currency, &t.Status, &t.Message, &t.IsAnonymous,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		tips = append(tips, t)
	}
	return tips, total, nil
}

// GetTipsByDriver returns tips received by a driver
func (r *Repository) GetTipsByDriver(ctx context.Context, driverID uuid.UUID, limit, offset int) ([]Tip, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM tips WHERE driver_id = $1 AND status = 'completed'`,
		driverID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, ride_id, rider_id, driver_id, amount,
			currency, status, message, is_anonymous,
			created_at, updated_at
		FROM tips
		WHERE driver_id = $1 AND status = 'completed'
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		driverID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tips []Tip
	for rows.Next() {
		t := Tip{}
		if err := rows.Scan(
			&t.ID, &t.RideID, &t.RiderID, &t.DriverID, &t.Amount,
			&t.Currency, &t.Status, &t.Message, &t.IsAnonymous,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		// Hide rider info for anonymous tips
		if t.IsAnonymous {
			t.RiderID = uuid.Nil
		}
		tips = append(tips, t)
	}
	return tips, total, nil
}
