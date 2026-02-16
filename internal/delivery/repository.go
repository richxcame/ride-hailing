package delivery

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles delivery data access
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new delivery repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// DELIVERY CRUD
// ========================================

// CreateDelivery inserts a new delivery and its stops in a transaction
func (r *Repository) CreateDelivery(ctx context.Context, d *Delivery, stops []DeliveryStop) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO deliveries (
			id, sender_id, status, priority, tracking_code,
			pickup_latitude, pickup_longitude, pickup_address, pickup_contact, pickup_phone, pickup_notes,
			dropoff_latitude, dropoff_longitude, dropoff_address, recipient_name, recipient_phone, dropoff_notes,
			package_size, package_description, weight_kg, is_fragile, requires_signature, declared_value,
			estimated_distance, estimated_duration, estimated_fare, surge_multiplier,
			scheduled_pickup_at, scheduled_dropoff_at, proof_pin,
			requested_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17,
			$18, $19, $20, $21, $22, $23,
			$24, $25, $26, $27,
			$28, $29, $30,
			$31, $32, $33
		)`,
		d.ID, d.SenderID, d.Status, d.Priority, d.TrackingCode,
		d.PickupLatitude, d.PickupLongitude, d.PickupAddress, d.PickupContact, d.PickupPhone, d.PickupNotes,
		d.DropoffLatitude, d.DropoffLongitude, d.DropoffAddress, d.RecipientName, d.RecipientPhone, d.DropoffNotes,
		d.PackageSize, d.PackageDescription, d.WeightKg, d.IsFragile, d.RequiresSignature, d.DeclaredValue,
		d.EstimatedDistance, d.EstimatedDuration, d.EstimatedFare, d.SurgeMultiplier,
		d.ScheduledPickupAt, d.ScheduledDropoffAt, d.ProofPIN,
		d.RequestedAt, d.CreatedAt, d.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert delivery: %w", err)
	}

	for _, stop := range stops {
		_, err = tx.Exec(ctx, `
			INSERT INTO delivery_stops (
				id, delivery_id, stop_order, latitude, longitude, address,
				contact_name, contact_phone, notes, status, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			stop.ID, stop.DeliveryID, stop.StopOrder,
			stop.Latitude, stop.Longitude, stop.Address,
			stop.ContactName, stop.ContactPhone, stop.Notes,
			stop.Status, stop.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert stop %d: %w", stop.StopOrder, err)
		}
	}

	return tx.Commit(ctx)
}

// GetDeliveryByID retrieves a delivery by ID
func (r *Repository) GetDeliveryByID(ctx context.Context, id uuid.UUID) (*Delivery, error) {
	d := &Delivery{}
	err := r.db.QueryRow(ctx, `
		SELECT id, sender_id, driver_id, status, priority, tracking_code,
			pickup_latitude, pickup_longitude, pickup_address, pickup_contact, pickup_phone, pickup_notes,
			dropoff_latitude, dropoff_longitude, dropoff_address, recipient_name, recipient_phone, dropoff_notes,
			package_size, package_description, weight_kg, is_fragile, requires_signature, declared_value,
			estimated_distance, estimated_duration, estimated_fare, final_fare, surge_multiplier,
			proof_type, proof_photo_url, proof_pin, signature_url,
			scheduled_pickup_at, scheduled_dropoff_at,
			requested_at, accepted_at, picked_up_at, delivered_at, cancelled_at, cancel_reason,
			sender_rating, sender_feedback, driver_rating, driver_feedback,
			created_at, updated_at
		FROM deliveries WHERE id = $1`, id,
	).Scan(
		&d.ID, &d.SenderID, &d.DriverID, &d.Status, &d.Priority, &d.TrackingCode,
		&d.PickupLatitude, &d.PickupLongitude, &d.PickupAddress, &d.PickupContact, &d.PickupPhone, &d.PickupNotes,
		&d.DropoffLatitude, &d.DropoffLongitude, &d.DropoffAddress, &d.RecipientName, &d.RecipientPhone, &d.DropoffNotes,
		&d.PackageSize, &d.PackageDescription, &d.WeightKg, &d.IsFragile, &d.RequiresSignature, &d.DeclaredValue,
		&d.EstimatedDistance, &d.EstimatedDuration, &d.EstimatedFare, &d.FinalFare, &d.SurgeMultiplier,
		&d.ProofType, &d.ProofPhotoURL, &d.ProofPIN, &d.SignatureURL,
		&d.ScheduledPickupAt, &d.ScheduledDropoffAt,
		&d.RequestedAt, &d.AcceptedAt, &d.PickedUpAt, &d.DeliveredAt, &d.CancelledAt, &d.CancelReason,
		&d.SenderRating, &d.SenderFeedback, &d.DriverRating, &d.DriverFeedback,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// GetDeliveryByTrackingCode retrieves a delivery by tracking code
func (r *Repository) GetDeliveryByTrackingCode(ctx context.Context, code string) (*Delivery, error) {
	d := &Delivery{}
	err := r.db.QueryRow(ctx, `
		SELECT id, sender_id, driver_id, status, priority, tracking_code,
			pickup_latitude, pickup_longitude, pickup_address, pickup_contact, pickup_phone, pickup_notes,
			dropoff_latitude, dropoff_longitude, dropoff_address, recipient_name, recipient_phone, dropoff_notes,
			package_size, package_description, weight_kg, is_fragile, requires_signature, declared_value,
			estimated_distance, estimated_duration, estimated_fare, final_fare, surge_multiplier,
			proof_type, proof_photo_url, proof_pin, signature_url,
			scheduled_pickup_at, scheduled_dropoff_at,
			requested_at, accepted_at, picked_up_at, delivered_at, cancelled_at, cancel_reason,
			sender_rating, sender_feedback, driver_rating, driver_feedback,
			created_at, updated_at
		FROM deliveries WHERE tracking_code = $1`, code,
	).Scan(
		&d.ID, &d.SenderID, &d.DriverID, &d.Status, &d.Priority, &d.TrackingCode,
		&d.PickupLatitude, &d.PickupLongitude, &d.PickupAddress, &d.PickupContact, &d.PickupPhone, &d.PickupNotes,
		&d.DropoffLatitude, &d.DropoffLongitude, &d.DropoffAddress, &d.RecipientName, &d.RecipientPhone, &d.DropoffNotes,
		&d.PackageSize, &d.PackageDescription, &d.WeightKg, &d.IsFragile, &d.RequiresSignature, &d.DeclaredValue,
		&d.EstimatedDistance, &d.EstimatedDuration, &d.EstimatedFare, &d.FinalFare, &d.SurgeMultiplier,
		&d.ProofType, &d.ProofPhotoURL, &d.ProofPIN, &d.SignatureURL,
		&d.ScheduledPickupAt, &d.ScheduledDropoffAt,
		&d.RequestedAt, &d.AcceptedAt, &d.PickedUpAt, &d.DeliveredAt, &d.CancelledAt, &d.CancelReason,
		&d.SenderRating, &d.SenderFeedback, &d.DriverRating, &d.DriverFeedback,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// AtomicAcceptDelivery atomically assigns a driver to a requested delivery
func (r *Repository) AtomicAcceptDelivery(ctx context.Context, deliveryID, driverID uuid.UUID) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE deliveries
		SET driver_id = $2, status = $3, accepted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = $4`,
		deliveryID, driverID, DeliveryStatusAccepted, DeliveryStatusRequested,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// UpdateDeliveryStatus updates the delivery status with a timestamp
func (r *Repository) UpdateDeliveryStatus(ctx context.Context, deliveryID uuid.UUID, status DeliveryStatus, driverID uuid.UUID) error {
	var query string
	switch status {
	case DeliveryStatusPickingUp:
		query = `UPDATE deliveries SET status = $2, updated_at = NOW() WHERE id = $1 AND driver_id = $3`
	case DeliveryStatusPickedUp:
		query = `UPDATE deliveries SET status = $2, picked_up_at = NOW(), updated_at = NOW() WHERE id = $1 AND driver_id = $3`
	case DeliveryStatusInTransit:
		query = `UPDATE deliveries SET status = $2, updated_at = NOW() WHERE id = $1 AND driver_id = $3`
	case DeliveryStatusArrived:
		query = `UPDATE deliveries SET status = $2, updated_at = NOW() WHERE id = $1 AND driver_id = $3`
	default:
		query = `UPDATE deliveries SET status = $2, updated_at = NOW() WHERE id = $1 AND driver_id = $3`
	}
	_, err := r.db.Exec(ctx, query, deliveryID, status, driverID)
	return err
}

// AtomicCompleteDelivery atomically marks a delivery as delivered with proof
func (r *Repository) AtomicCompleteDelivery(ctx context.Context, deliveryID, driverID uuid.UUID, proofType ProofType, proofPhotoURL, signatureURL *string, finalFare float64) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE deliveries
		SET status = $2, final_fare = $3, proof_type = $4, proof_photo_url = $5, signature_url = $6,
		    delivered_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND driver_id = $7 AND status IN ($8, $9)`,
		deliveryID, DeliveryStatusDelivered, finalFare, proofType, proofPhotoURL, signatureURL,
		driverID, DeliveryStatusInTransit, DeliveryStatusArrived,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// CancelDelivery cancels a delivery with a reason
func (r *Repository) CancelDelivery(ctx context.Context, deliveryID uuid.UUID, reason string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE deliveries
		SET status = $2, cancel_reason = $3, cancelled_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status NOT IN ($4, $5)`,
		deliveryID, DeliveryStatusCancelled, reason,
		DeliveryStatusDelivered, DeliveryStatusCancelled,
	)
	return err
}

// ReturnDelivery marks a delivery as returned to sender
func (r *Repository) ReturnDelivery(ctx context.Context, deliveryID uuid.UUID, reason string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE deliveries
		SET status = $2, cancel_reason = $3, updated_at = NOW()
		WHERE id = $1 AND driver_id IS NOT NULL`,
		deliveryID, DeliveryStatusReturned, reason,
	)
	return err
}

// RateDeliveryBySender updates sender's rating for the delivery
func (r *Repository) RateDeliveryBySender(ctx context.Context, deliveryID uuid.UUID, rating int, feedback *string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE deliveries SET sender_rating = $2, sender_feedback = $3, updated_at = NOW()
		WHERE id = $1`,
		deliveryID, rating, feedback,
	)
	return err
}

// RateDeliveryByDriver updates driver's rating for the delivery
func (r *Repository) RateDeliveryByDriver(ctx context.Context, deliveryID uuid.UUID, rating int, feedback *string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE deliveries SET driver_rating = $2, driver_feedback = $3, updated_at = NOW()
		WHERE id = $1`,
		deliveryID, rating, feedback,
	)
	return err
}

// ========================================
// LISTING & FILTERING
// ========================================

// GetDeliveriesBySender lists deliveries for a sender with pagination
func (r *Repository) GetDeliveriesBySender(ctx context.Context, senderID uuid.UUID, filters *DeliveryListFilters, limit, offset int) ([]*Delivery, int64, error) {
	where := []string{"sender_id = $1"}
	args := []interface{}{senderID}
	argIdx := 2

	if filters != nil {
		if filters.Status != nil {
			where = append(where, fmt.Sprintf("status = $%d", argIdx))
			args = append(args, *filters.Status)
			argIdx++
		}
		if filters.Priority != nil {
			where = append(where, fmt.Sprintf("priority = $%d", argIdx))
			args = append(args, *filters.Priority)
			argIdx++
		}
		if filters.FromDate != nil {
			where = append(where, fmt.Sprintf("created_at >= $%d", argIdx))
			args = append(args, *filters.FromDate)
			argIdx++
		}
		if filters.ToDate != nil {
			where = append(where, fmt.Sprintf("created_at <= $%d", argIdx))
			args = append(args, *filters.ToDate)
			argIdx++
		}
	}

	whereClause := strings.Join(where, " AND ")

	// Count
	var total int64
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.db.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM deliveries WHERE %s", whereClause),
		countArgs...,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Query
	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx,
		fmt.Sprintf(`
			SELECT id, sender_id, driver_id, status, priority, tracking_code,
				pickup_address, dropoff_address, recipient_name,
				package_size, package_description, estimated_fare, final_fare,
				requested_at, accepted_at, picked_up_at, delivered_at, cancelled_at,
				sender_rating, created_at, updated_at
			FROM deliveries
			WHERE %s
			ORDER BY created_at DESC
			LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1),
		args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var deliveries []*Delivery
	for rows.Next() {
		d := &Delivery{}
		if err := rows.Scan(
			&d.ID, &d.SenderID, &d.DriverID, &d.Status, &d.Priority, &d.TrackingCode,
			&d.PickupAddress, &d.DropoffAddress, &d.RecipientName,
			&d.PackageSize, &d.PackageDescription, &d.EstimatedFare, &d.FinalFare,
			&d.RequestedAt, &d.AcceptedAt, &d.PickedUpAt, &d.DeliveredAt, &d.CancelledAt,
			&d.SenderRating, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		deliveries = append(deliveries, d)
	}
	return deliveries, total, nil
}

// GetAvailableDeliveries lists deliveries awaiting driver acceptance
func (r *Repository) GetAvailableDeliveries(ctx context.Context, latitude, longitude float64, radiusKm float64) ([]*Delivery, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, sender_id, status, priority, tracking_code,
			pickup_latitude, pickup_longitude, pickup_address,
			dropoff_latitude, dropoff_longitude, dropoff_address,
			package_size, package_description, weight_kg, is_fragile,
			estimated_distance, estimated_duration, estimated_fare, surge_multiplier,
			scheduled_pickup_at, requested_at, created_at
		FROM deliveries
		WHERE status = $1
			AND (
				6371 * acos(
					cos(radians($2)) * cos(radians(pickup_latitude))
					* cos(radians(pickup_longitude) - radians($3))
					+ sin(radians($2)) * sin(radians(pickup_latitude))
				)
			) <= $4
		ORDER BY
			CASE priority
				WHEN 'express' THEN 1
				WHEN 'standard' THEN 2
				WHEN 'scheduled' THEN 3
			END,
			requested_at ASC
		LIMIT 50`,
		DeliveryStatusRequested, latitude, longitude, radiusKm,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []*Delivery
	for rows.Next() {
		d := &Delivery{}
		if err := rows.Scan(
			&d.ID, &d.SenderID, &d.Status, &d.Priority, &d.TrackingCode,
			&d.PickupLatitude, &d.PickupLongitude, &d.PickupAddress,
			&d.DropoffLatitude, &d.DropoffLongitude, &d.DropoffAddress,
			&d.PackageSize, &d.PackageDescription, &d.WeightKg, &d.IsFragile,
			&d.EstimatedDistance, &d.EstimatedDuration, &d.EstimatedFare, &d.SurgeMultiplier,
			&d.ScheduledPickupAt, &d.RequestedAt, &d.CreatedAt,
		); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, d)
	}
	return deliveries, nil
}

// GetDriverDeliveries lists deliveries assigned to a driver
func (r *Repository) GetDriverDeliveries(ctx context.Context, driverID uuid.UUID, limit, offset int) ([]*Delivery, int64, error) {
	var total int64
	err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM deliveries WHERE driver_id = $1", driverID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, sender_id, driver_id, status, priority, tracking_code,
			pickup_address, dropoff_address, recipient_name,
			package_size, package_description, estimated_fare, final_fare,
			requested_at, accepted_at, picked_up_at, delivered_at, cancelled_at,
			driver_rating, created_at, updated_at
		FROM deliveries
		WHERE driver_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		driverID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var deliveries []*Delivery
	for rows.Next() {
		d := &Delivery{}
		if err := rows.Scan(
			&d.ID, &d.SenderID, &d.DriverID, &d.Status, &d.Priority, &d.TrackingCode,
			&d.PickupAddress, &d.DropoffAddress, &d.RecipientName,
			&d.PackageSize, &d.PackageDescription, &d.EstimatedFare, &d.FinalFare,
			&d.RequestedAt, &d.AcceptedAt, &d.PickedUpAt, &d.DeliveredAt, &d.CancelledAt,
			&d.DriverRating, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		deliveries = append(deliveries, d)
	}
	return deliveries, total, nil
}

// ========================================
// STOPS
// ========================================

// GetStopsByDeliveryID gets all stops for a delivery
func (r *Repository) GetStopsByDeliveryID(ctx context.Context, deliveryID uuid.UUID) ([]DeliveryStop, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, delivery_id, stop_order, latitude, longitude, address,
			contact_name, contact_phone, notes, status,
			arrived_at, completed_at, proof_photo_url, created_at
		FROM delivery_stops
		WHERE delivery_id = $1
		ORDER BY stop_order ASC`,
		deliveryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stops []DeliveryStop
	for rows.Next() {
		s := DeliveryStop{}
		if err := rows.Scan(
			&s.ID, &s.DeliveryID, &s.StopOrder, &s.Latitude, &s.Longitude, &s.Address,
			&s.ContactName, &s.ContactPhone, &s.Notes, &s.Status,
			&s.ArrivedAt, &s.CompletedAt, &s.ProofPhotoURL, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		stops = append(stops, s)
	}
	return stops, nil
}

// UpdateStopStatus updates a stop's status
func (r *Repository) UpdateStopStatus(ctx context.Context, stopID uuid.UUID, status string, photoURL *string) error {
	var query string
	switch status {
	case "arrived":
		query = `UPDATE delivery_stops SET status = $2, arrived_at = NOW() WHERE id = $1`
	case "completed":
		query = `UPDATE delivery_stops SET status = $2, completed_at = NOW(), proof_photo_url = $3 WHERE id = $1`
	default:
		query = `UPDATE delivery_stops SET status = $2 WHERE id = $1`
	}
	if status == "completed" {
		_, err := r.db.Exec(ctx, query, stopID, status, photoURL)
		return err
	}
	_, err := r.db.Exec(ctx, query, stopID, status)
	return err
}

// ========================================
// TRACKING
// ========================================

// AddTrackingEvent records a tracking event for a delivery
func (r *Repository) AddTrackingEvent(ctx context.Context, t *DeliveryTracking) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO delivery_tracking (id, delivery_id, driver_id, latitude, longitude, status, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		t.ID, t.DeliveryID, t.DriverID, t.Latitude, t.Longitude, t.Status, t.Timestamp,
	)
	return err
}

// GetTrackingByDeliveryID gets tracking history for a delivery
func (r *Repository) GetTrackingByDeliveryID(ctx context.Context, deliveryID uuid.UUID) ([]DeliveryTracking, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, delivery_id, driver_id, latitude, longitude, status, timestamp
		FROM delivery_tracking
		WHERE delivery_id = $1
		ORDER BY timestamp ASC`,
		deliveryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracking []DeliveryTracking
	for rows.Next() {
		t := DeliveryTracking{}
		if err := rows.Scan(
			&t.ID, &t.DeliveryID, &t.DriverID, &t.Latitude, &t.Longitude, &t.Status, &t.Timestamp,
		); err != nil {
			return nil, err
		}
		tracking = append(tracking, t)
	}
	return tracking, nil
}

// ========================================
// STATS
// ========================================

// GetSenderStats gets delivery statistics for a sender
func (r *Repository) GetSenderStats(ctx context.Context, senderID uuid.UUID) (*DeliveryStats, error) {
	stats := &DeliveryStats{}

	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'delivered') as completed,
			COUNT(*) FILTER (WHERE status = 'cancelled') as cancelled,
			COUNT(*) FILTER (WHERE status IN ('accepted', 'picking_up', 'picked_up', 'in_transit', 'arrived')) as in_progress,
			COALESCE(AVG(sender_rating) FILTER (WHERE sender_rating IS NOT NULL), 0) as avg_rating,
			COALESCE(SUM(COALESCE(final_fare, estimated_fare)) FILTER (WHERE status = 'delivered'), 0) as total_spent,
			COALESCE(AVG(EXTRACT(EPOCH FROM (delivered_at - requested_at)) / 60) FILTER (WHERE status = 'delivered'), 0) as avg_delivery_min
		FROM deliveries
		WHERE sender_id = $1`, senderID,
	).Scan(
		&stats.TotalDeliveries, &stats.CompletedCount, &stats.CancelledCount,
		&stats.InProgressCount, &stats.AverageRating, &stats.TotalSpent,
		&stats.AverageDeliveryMin,
	)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// GetActiveDeliveryForDriver gets the driver's current active delivery
func (r *Repository) GetActiveDeliveryForDriver(ctx context.Context, driverID uuid.UUID) (*Delivery, error) {
	d := &Delivery{}
	err := r.db.QueryRow(ctx, `
		SELECT id, sender_id, driver_id, status, priority, tracking_code,
			pickup_latitude, pickup_longitude, pickup_address, pickup_contact, pickup_phone, pickup_notes,
			dropoff_latitude, dropoff_longitude, dropoff_address, recipient_name, recipient_phone, dropoff_notes,
			package_size, package_description, weight_kg, is_fragile, requires_signature, declared_value,
			estimated_distance, estimated_duration, estimated_fare, final_fare, surge_multiplier,
			proof_type, proof_photo_url, proof_pin, signature_url,
			scheduled_pickup_at, scheduled_dropoff_at,
			requested_at, accepted_at, picked_up_at, delivered_at, cancelled_at, cancel_reason,
			sender_rating, sender_feedback, driver_rating, driver_feedback,
			created_at, updated_at
		FROM deliveries
		WHERE driver_id = $1 AND status IN ($2, $3, $4, $5, $6)
		ORDER BY accepted_at DESC
		LIMIT 1`,
		driverID,
		DeliveryStatusAccepted, DeliveryStatusPickingUp, DeliveryStatusPickedUp,
		DeliveryStatusInTransit, DeliveryStatusArrived,
	).Scan(
		&d.ID, &d.SenderID, &d.DriverID, &d.Status, &d.Priority, &d.TrackingCode,
		&d.PickupLatitude, &d.PickupLongitude, &d.PickupAddress, &d.PickupContact, &d.PickupPhone, &d.PickupNotes,
		&d.DropoffLatitude, &d.DropoffLongitude, &d.DropoffAddress, &d.RecipientName, &d.RecipientPhone, &d.DropoffNotes,
		&d.PackageSize, &d.PackageDescription, &d.WeightKg, &d.IsFragile, &d.RequiresSignature, &d.DeclaredValue,
		&d.EstimatedDistance, &d.EstimatedDuration, &d.EstimatedFare, &d.FinalFare, &d.SurgeMultiplier,
		&d.ProofType, &d.ProofPhotoURL, &d.ProofPIN, &d.SignatureURL,
		&d.ScheduledPickupAt, &d.ScheduledDropoffAt,
		&d.RequestedAt, &d.AcceptedAt, &d.PickedUpAt, &d.DeliveredAt, &d.CancelledAt, &d.CancelReason,
		&d.SenderRating, &d.SenderFeedback, &d.DriverRating, &d.DriverFeedback,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return d, nil
}
