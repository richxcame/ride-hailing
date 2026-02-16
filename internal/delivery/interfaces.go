package delivery

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryInterface defines the contract for delivery repository operations
type RepositoryInterface interface {
	// Delivery CRUD
	CreateDelivery(ctx context.Context, d *Delivery, stops []DeliveryStop) error
	GetDeliveryByID(ctx context.Context, id uuid.UUID) (*Delivery, error)
	GetDeliveryByTrackingCode(ctx context.Context, code string) (*Delivery, error)

	// Delivery state transitions
	AtomicAcceptDelivery(ctx context.Context, deliveryID, driverID uuid.UUID) (bool, error)
	UpdateDeliveryStatus(ctx context.Context, deliveryID uuid.UUID, status DeliveryStatus, driverID uuid.UUID) error
	AtomicCompleteDelivery(ctx context.Context, deliveryID, driverID uuid.UUID, proofType ProofType, proofPhotoURL, signatureURL *string, finalFare float64) (bool, error)
	CancelDelivery(ctx context.Context, deliveryID uuid.UUID, reason string) error
	ReturnDelivery(ctx context.Context, deliveryID uuid.UUID, reason string) error

	// Ratings
	RateDeliveryBySender(ctx context.Context, deliveryID uuid.UUID, rating int, feedback *string) error
	RateDeliveryByDriver(ctx context.Context, deliveryID uuid.UUID, rating int, feedback *string) error

	// Listing & filtering
	GetDeliveriesBySender(ctx context.Context, senderID uuid.UUID, filters *DeliveryListFilters, limit, offset int) ([]*Delivery, int64, error)
	GetAvailableDeliveries(ctx context.Context, latitude, longitude float64, radiusKm float64) ([]*Delivery, error)
	GetDriverDeliveries(ctx context.Context, driverID uuid.UUID, limit, offset int) ([]*Delivery, int64, error)
	GetActiveDeliveryForDriver(ctx context.Context, driverID uuid.UUID) (*Delivery, error)

	// Stops
	GetStopsByDeliveryID(ctx context.Context, deliveryID uuid.UUID) ([]DeliveryStop, error)
	UpdateStopStatus(ctx context.Context, stopID uuid.UUID, status string, photoURL *string) error

	// Tracking
	AddTrackingEvent(ctx context.Context, t *DeliveryTracking) error
	GetTrackingByDeliveryID(ctx context.Context, deliveryID uuid.UUID) ([]DeliveryTracking, error)

	// Stats
	GetSenderStats(ctx context.Context, senderID uuid.UUID) (*DeliveryStats, error)
}
