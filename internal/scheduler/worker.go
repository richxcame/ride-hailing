package scheduler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

const (
	// Check for scheduled rides every minute
	checkInterval = 1 * time.Minute
	// Look ahead window - process rides scheduled within next 30 minutes
	lookAheadMinutes = 30
)

// Worker handles scheduled ride processing
type Worker struct {
	db     *pgxpool.Pool
	logger *zap.Logger
	done   chan struct{}
}

// NewWorker creates a new scheduler worker
func NewWorker(db *pgxpool.Pool, logger *zap.Logger) *Worker {
	return &Worker{
		db:     db,
		logger: logger,
		done:   make(chan struct{}),
	}
}

// Start begins the scheduled ride processing loop
func (w *Worker) Start(ctx context.Context) {
	w.logger.Info("Starting scheduled rides worker")

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// Run immediately on start
	w.processScheduledRides(ctx)

	for {
		select {
		case <-ticker.C:
			w.processScheduledRides(ctx)
		case <-ctx.Done():
			w.logger.Info("Scheduled rides worker stopped")
			return
		case <-w.done:
			w.logger.Info("Scheduled rides worker shutdown requested")
			return
		}
	}
}

// Stop gracefully stops the worker
func (w *Worker) Stop() {
	close(w.done)
}

// processScheduledRides finds and processes scheduled rides that are due
func (w *Worker) processScheduledRides(ctx context.Context) {
	// Get upcoming scheduled rides using the database function
	rides, err := w.getUpcomingScheduledRides(ctx, lookAheadMinutes)
	if err != nil {
		w.logger.Error("Failed to get upcoming scheduled rides", zap.Error(err))
		return
	}

	if len(rides) == 0 {
		w.logger.Debug("No scheduled rides to process")
		return
	}

	w.logger.Info("Processing scheduled rides", zap.Int("count", len(rides)))

	for _, ride := range rides {
		// Check if it's time to activate this ride
		timeUntilRide := time.Until(ride.ScheduledAt)

		if timeUntilRide <= 5*time.Minute {
			// Activate the ride (make it available for drivers)
			err := w.activateScheduledRide(ctx, ride.ID)
			if err != nil {
				w.logger.Error("Failed to activate scheduled ride",
					zap.String("ride_id", ride.ID.String()),
					zap.Error(err))
				continue
			}

			w.logger.Info("Activated scheduled ride",
				zap.String("ride_id", ride.ID.String()),
				zap.Time("scheduled_at", ride.ScheduledAt))
		} else if timeUntilRide <= 30*time.Minute && !ride.NotificationSent {
			// Send notification to rider about upcoming ride
			err := w.sendUpcomingRideNotification(ctx, ride.ID, ride.RiderID)
			if err != nil {
				w.logger.Error("Failed to send notification",
					zap.String("ride_id", ride.ID.String()),
					zap.Error(err))
				continue
			}

			w.logger.Info("Sent upcoming ride notification",
				zap.String("ride_id", ride.ID.String()),
				zap.String("rider_id", ride.RiderID.String()))
		}
	}
}

// ScheduledRide represents a scheduled ride from the database
type ScheduledRide struct {
	ID               uuid.UUID
	RiderID          uuid.UUID
	ScheduledAt      time.Time
	PickupAddress    string
	DropoffAddress   string
	EstimatedFare    float64
	NotificationSent bool
}

// getUpcomingScheduledRides retrieves scheduled rides within the look-ahead window
func (w *Worker) getUpcomingScheduledRides(ctx context.Context, minutesAhead int) ([]*ScheduledRide, error) {
	query := `
		SELECT id, rider_id, scheduled_at, pickup_address, dropoff_address,
		       estimated_fare, scheduled_notification_sent
		FROM rides
		WHERE is_scheduled = true
		  AND status = 'requested'
		  AND scheduled_at <= NOW() + ($1 || ' minutes')::INTERVAL
		  AND scheduled_at > NOW()
		ORDER BY scheduled_at ASC
	`

	rows, err := w.db.Query(ctx, query, minutesAhead)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rides []*ScheduledRide
	for rows.Next() {
		ride := &ScheduledRide{}
		err := rows.Scan(
			&ride.ID,
			&ride.RiderID,
			&ride.ScheduledAt,
			&ride.PickupAddress,
			&ride.DropoffAddress,
			&ride.EstimatedFare,
			&ride.NotificationSent,
		)
		if err != nil {
			return nil, err
		}
		rides = append(rides, ride)
	}

	return rides, nil
}

// activateScheduledRide changes a scheduled ride to active status
func (w *Worker) activateScheduledRide(ctx context.Context, rideID uuid.UUID) error {
	query := `
		UPDATE rides
		SET is_scheduled = false,
		    requested_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1
		  AND is_scheduled = true
		  AND status = 'requested'
	`

	result, err := w.db.Exec(ctx, query, rideID)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		w.logger.Warn("No rows affected when activating scheduled ride",
			zap.String("ride_id", rideID.String()))
	}

	return nil
}

// sendUpcomingRideNotification marks that a notification was sent for an upcoming ride
func (w *Worker) sendUpcomingRideNotification(ctx context.Context, rideID, riderID uuid.UUID) error {
	// In a real implementation, this would call the notifications service
	// For now, we'll just mark it as sent in the database

	query := `
		UPDATE rides
		SET scheduled_notification_sent = true,
		    updated_at = NOW()
		WHERE id = $1
		  AND scheduled_notification_sent = false
	`

	_, err := w.db.Exec(ctx, query, rideID)
	if err != nil {
		return err
	}

	// TODO: Actually send notification via notifications service
	w.logger.Info("Notification marked as sent (actual notification sending not yet implemented)",
		zap.String("ride_id", rideID.String()),
		zap.String("rider_id", riderID.String()))

	return nil
}
