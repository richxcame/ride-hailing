package matching

import (
	"context"

	"github.com/google/uuid"
)

// GeoServiceInterface provides driver location lookups
type GeoServiceInterface interface {
	FindAvailableDrivers(ctx context.Context, latitude, longitude float64, maxDrivers int) ([]*GeoDriverLocation, error)
	CalculateDistance(latitude1, longitude1, latitude2, longitude2 float64) float64
}

// RidesRepositoryInterface provides ride data access
type RidesRepositoryInterface interface {
	UpdateRideDriver(ctx context.Context, rideID, driverID uuid.UUID) error
	GetRideByID(ctx context.Context, rideID uuid.UUID) (interface{}, error)
}

// GeoDriverLocation represents driver location from geo service
type GeoDriverLocation struct {
	DriverID  uuid.UUID
	Latitude  float64
	Longitude float64
	Status    string
}
