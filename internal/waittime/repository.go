package waittime

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles wait time data access
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new wait time repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// CONFIGS
// ========================================

// GetActiveConfig retrieves the active wait time config
func (r *Repository) GetActiveConfig(ctx context.Context) (*WaitTimeConfig, error) {
	c := &WaitTimeConfig{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name, free_wait_minutes, charge_per_minute,
			max_wait_minutes, max_wait_charge, applies_to, is_active,
			created_at, updated_at
		FROM wait_time_configs
		WHERE is_active = true
		ORDER BY created_at DESC
		LIMIT 1`,
	).Scan(
		&c.ID, &c.Name, &c.FreeWaitMinutes, &c.ChargePerMinute,
		&c.MaxWaitMinutes, &c.MaxWaitCharge, &c.AppliesTo, &c.IsActive,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// GetAllConfigs retrieves all wait time configs
func (r *Repository) GetAllConfigs(ctx context.Context) ([]WaitTimeConfig, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, free_wait_minutes, charge_per_minute,
			max_wait_minutes, max_wait_charge, applies_to, is_active,
			created_at, updated_at
		FROM wait_time_configs
		ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []WaitTimeConfig
	for rows.Next() {
		c := WaitTimeConfig{}
		if err := rows.Scan(
			&c.ID, &c.Name, &c.FreeWaitMinutes, &c.ChargePerMinute,
			&c.MaxWaitMinutes, &c.MaxWaitCharge, &c.AppliesTo, &c.IsActive,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	return configs, nil
}

// CreateConfig inserts a new wait time config
func (r *Repository) CreateConfig(ctx context.Context, c *WaitTimeConfig) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO wait_time_configs (
			id, name, free_wait_minutes, charge_per_minute,
			max_wait_minutes, max_wait_charge, applies_to, is_active,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		c.ID, c.Name, c.FreeWaitMinutes, c.ChargePerMinute,
		c.MaxWaitMinutes, c.MaxWaitCharge, c.AppliesTo, c.IsActive,
		c.CreatedAt, c.UpdatedAt,
	)
	return err
}

// ========================================
// RECORDS
// ========================================

// CreateRecord inserts a new wait time record
func (r *Repository) CreateRecord(ctx context.Context, rec *WaitTimeRecord) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO wait_time_records (
			id, ride_id, driver_id, config_id, wait_type,
			arrived_at, free_minutes, total_wait_minutes, chargeable_minutes,
			charge_per_minute, total_charge, was_capped, status,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		rec.ID, rec.RideID, rec.DriverID, rec.ConfigID, rec.WaitType,
		rec.ArrivedAt, rec.FreeMinutes, rec.TotalWaitMinutes, rec.ChargeableMinutes,
		rec.ChargePerMinute, rec.TotalCharge, rec.WasCapped, rec.Status,
		rec.CreatedAt, rec.UpdatedAt,
	)
	return err
}

// GetActiveWaitByRide retrieves the currently active wait record for a ride
func (r *Repository) GetActiveWaitByRide(ctx context.Context, rideID uuid.UUID) (*WaitTimeRecord, error) {
	rec := &WaitTimeRecord{}
	err := r.db.QueryRow(ctx, `
		SELECT id, ride_id, driver_id, config_id, wait_type,
			arrived_at, started_at, free_minutes, total_wait_minutes, chargeable_minutes,
			charge_per_minute, total_charge, was_capped, status,
			created_at, updated_at
		FROM wait_time_records
		WHERE ride_id = $1 AND status = 'waiting'
		ORDER BY created_at DESC
		LIMIT 1`, rideID,
	).Scan(
		&rec.ID, &rec.RideID, &rec.DriverID, &rec.ConfigID, &rec.WaitType,
		&rec.ArrivedAt, &rec.StartedAt, &rec.FreeMinutes, &rec.TotalWaitMinutes, &rec.ChargeableMinutes,
		&rec.ChargePerMinute, &rec.TotalCharge, &rec.WasCapped, &rec.Status,
		&rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return rec, nil
}

// CompleteWait finalizes a wait time record with calculated charges
func (r *Repository) CompleteWait(ctx context.Context, recordID uuid.UUID, totalWaitMin, chargeableMin, totalCharge float64, wasCapped bool) error {
	_, err := r.db.Exec(ctx, `
		UPDATE wait_time_records
		SET status = 'completed',
			started_at = NOW(),
			total_wait_minutes = $2,
			chargeable_minutes = $3,
			total_charge = $4,
			was_capped = $5,
			updated_at = NOW()
		WHERE id = $1 AND status = 'waiting'`,
		recordID, totalWaitMin, chargeableMin, totalCharge, wasCapped,
	)
	return err
}

// WaiveCharge waives wait time charges
func (r *Repository) WaiveCharge(ctx context.Context, recordID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE wait_time_records
		SET status = 'waived', total_charge = 0, updated_at = NOW()
		WHERE id = $1`,
		recordID,
	)
	return err
}

// GetRecordsByRide retrieves all wait time records for a ride
func (r *Repository) GetRecordsByRide(ctx context.Context, rideID uuid.UUID) ([]WaitTimeRecord, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, ride_id, driver_id, config_id, wait_type,
			arrived_at, started_at, free_minutes, total_wait_minutes, chargeable_minutes,
			charge_per_minute, total_charge, was_capped, status,
			created_at, updated_at
		FROM wait_time_records
		WHERE ride_id = $1
		ORDER BY created_at ASC`, rideID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []WaitTimeRecord
	for rows.Next() {
		rec := WaitTimeRecord{}
		if err := rows.Scan(
			&rec.ID, &rec.RideID, &rec.DriverID, &rec.ConfigID, &rec.WaitType,
			&rec.ArrivedAt, &rec.StartedAt, &rec.FreeMinutes, &rec.TotalWaitMinutes, &rec.ChargeableMinutes,
			&rec.ChargePerMinute, &rec.TotalCharge, &rec.WasCapped, &rec.Status,
			&rec.CreatedAt, &rec.UpdatedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, nil
}

// SaveNotification records a wait time notification
func (r *Repository) SaveNotification(ctx context.Context, n *WaitTimeNotification) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO wait_time_notifications (id, record_id, ride_id, rider_id, minutes_mark, message, sent_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		n.ID, n.RecordID, n.RideID, n.RiderID, n.MinutesMark, n.Message, n.SentAt,
	)
	return err
}
