package notifications

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/eventbus"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// EventHandler processes events from the NATS event bus and triggers notifications.
type EventHandler struct {
	service *Service
}

// NewEventHandler creates an event handler backed by the notification service.
func NewEventHandler(service *Service) *EventHandler {
	return &EventHandler{service: service}
}

// RegisterSubscriptions subscribes to ride lifecycle events on the bus.
func (h *EventHandler) RegisterSubscriptions(ctx context.Context, bus *eventbus.Bus) error {
	if err := bus.Subscribe(ctx, "rides.>", "notifications-rides", h.handleRideEvent); err != nil {
		return fmt.Errorf("subscribe to rides events: %w", err)
	}
	logger.Info("notifications: subscribed to ride lifecycle events")
	return nil
}

func (h *EventHandler) handleRideEvent(ctx context.Context, event *eventbus.Event) error {
	switch event.Type {
	case "ride.requested":
		return h.onRideRequested(ctx, event)
	case "ride.accepted":
		return h.onRideAccepted(ctx, event)
	case "ride.started":
		return h.onRideStarted(ctx, event)
	case "ride.completed":
		return h.onRideCompleted(ctx, event)
	case "ride.cancelled":
		return h.onRideCancelled(ctx, event)
	default:
		logger.Debug("notifications: ignoring unknown event type", zap.String("type", event.Type))
		return nil
	}
}

func (h *EventHandler) onRideRequested(ctx context.Context, event *eventbus.Event) error {
	var data eventbus.RideRequestedData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal ride requested: %w", err)
	}

	// Notify the rider that their ride request is being processed
	_, err := h.service.SendNotification(ctx, data.RiderID,
		"ride_requested", "push",
		"Ride Requested",
		"We're finding a driver for you. Hang tight!",
		map[string]interface{}{"ride_id": data.RideID.String()},
	)
	if err != nil {
		logger.Warn("failed to send ride_requested notification", zap.Error(err))
	}
	return nil
}

func (h *EventHandler) onRideAccepted(ctx context.Context, event *eventbus.Event) error {
	var data eventbus.RideAcceptedData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal ride accepted: %w", err)
	}

	_, err := h.service.SendNotification(ctx, data.RiderID,
		"ride_accepted", "push",
		"Driver Found!",
		"A driver has accepted your ride and is on the way.",
		map[string]interface{}{
			"ride_id":   data.RideID.String(),
			"driver_id": data.DriverID.String(),
		},
	)
	if err != nil {
		logger.Warn("failed to send ride_accepted notification", zap.Error(err))
	}
	return nil
}

func (h *EventHandler) onRideStarted(ctx context.Context, event *eventbus.Event) error {
	var data eventbus.RideStartedData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal ride started: %w", err)
	}

	_, err := h.service.SendNotification(ctx, data.RiderID,
		"ride_started", "push",
		"Ride Started",
		"Your ride has begun. Enjoy your trip!",
		map[string]interface{}{"ride_id": data.RideID.String()},
	)
	if err != nil {
		logger.Warn("failed to send ride_started notification", zap.Error(err))
	}
	return nil
}

func (h *EventHandler) onRideCompleted(ctx context.Context, event *eventbus.Event) error {
	var data eventbus.RideCompletedData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal ride completed: %w", err)
	}

	// Push notification to rider
	_, err := h.service.SendNotification(ctx, data.RiderID,
		"ride_completed", "push",
		"Ride Completed",
		fmt.Sprintf("Your ride is complete. Fare: $%.2f", data.FareAmount),
		map[string]interface{}{
			"ride_id":     data.RideID.String(),
			"fare_amount": data.FareAmount,
			"distance_km": data.DistanceKm,
		},
	)
	if err != nil {
		logger.Warn("failed to send ride_completed push notification", zap.Error(err))
	}

	// Also send receipt email to rider
	_, err = h.service.SendNotification(ctx, data.RiderID,
		"ride_receipt", "email",
		"Your Ride Receipt",
		fmt.Sprintf("Ride completed. Distance: %.1f km, Duration: %.0f min, Fare: $%.2f",
			data.DistanceKm, data.DurationMin, data.FareAmount),
		map[string]interface{}{
			"ride_id":      data.RideID.String(),
			"fare_amount":  data.FareAmount,
			"distance_km":  data.DistanceKm,
			"duration_min": data.DurationMin,
		},
	)
	if err != nil {
		logger.Warn("failed to send ride receipt email", zap.Error(err))
	}

	// Notify driver about payment
	_, err = h.service.SendNotification(ctx, data.DriverID,
		"payment_received", "push",
		"Payment Received",
		fmt.Sprintf("You earned $%.2f for this ride.", data.FareAmount*0.80),
		map[string]interface{}{
			"ride_id": data.RideID.String(),
			"amount":  data.FareAmount * 0.80,
		},
	)
	if err != nil {
		logger.Warn("failed to send payment notification to driver", zap.Error(err))
	}

	return nil
}

func (h *EventHandler) onRideCancelled(ctx context.Context, event *eventbus.Event) error {
	var data eventbus.RideCancelledData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal ride cancelled: %w", err)
	}

	// Notify the other party
	var recipientID uuid.UUID
	var title, body string

	if data.CancelledBy == "rider" && data.DriverID != uuid.Nil {
		recipientID = data.DriverID
		title = "Ride Cancelled"
		body = "The rider has cancelled the ride."
	} else if data.CancelledBy == "driver" {
		recipientID = data.RiderID
		title = "Ride Cancelled"
		body = "Your driver has cancelled. We're finding you a new driver."
	} else {
		// No one to notify (rider cancelled before driver assigned)
		return nil
	}

	_, err := h.service.SendNotification(ctx, recipientID,
		"ride_cancelled", "push",
		title, body,
		map[string]interface{}{
			"ride_id":      data.RideID.String(),
			"cancelled_by": data.CancelledBy,
			"reason":       data.Reason,
		},
	)
	if err != nil {
		logger.Warn("failed to send ride_cancelled notification", zap.Error(err))
	}
	return nil
}
