package payments

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/richxcame/ride-hailing/pkg/eventbus"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// EventHandler processes ride events and triggers payment operations.
type EventHandler struct {
	service *Service
}

// NewEventHandler creates an event handler backed by the payment service.
func NewEventHandler(service *Service) *EventHandler {
	return &EventHandler{service: service}
}

// RegisterSubscriptions subscribes to ride completion events on the bus.
func (h *EventHandler) RegisterSubscriptions(ctx context.Context, bus *eventbus.Bus) error {
	if err := bus.Subscribe(ctx, "rides.completed", "payments-ride-completed", h.handleRideCompleted); err != nil {
		return fmt.Errorf("subscribe to rides.completed: %w", err)
	}
	logger.Info("payments: subscribed to ride completion events for driver earnings")
	return nil
}

func (h *EventHandler) handleRideCompleted(ctx context.Context, event *eventbus.Event) error {
	var data eventbus.RideCompletedData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal ride completed: %w", err)
	}

	if data.FareAmount <= 0 {
		logger.Warn("payments: ride completed with zero fare, skipping earnings record",
			zap.String("ride_id", data.RideID.String()),
		)
		return nil
	}

	logger.Info("payments: recording driver earning for completed ride",
		zap.String("ride_id", data.RideID.String()),
		zap.String("driver_id", data.DriverID.String()),
		zap.Float64("fare_amount", data.FareAmount),
	)

	if err := h.service.RecordRideEarning(ctx, data.DriverID, data.RideID, data.FareAmount); err != nil {
		logger.Error("payments: failed to record driver earning",
			zap.String("ride_id", data.RideID.String()),
			zap.String("driver_id", data.DriverID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("record driver earning: %w", err)
	}

	logger.Info("payments: driver earning recorded successfully",
		zap.String("ride_id", data.RideID.String()),
		zap.String("driver_id", data.DriverID.String()),
		zap.Float64("gross", data.FareAmount),
	)
	return nil
}
