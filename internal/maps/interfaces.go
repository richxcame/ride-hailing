package maps

import (
	"context"
)

// ETAProvider defines the interface for ETA calculations
// This can be implemented by the full maps service or a simpler wrapper
type ETAProvider interface {
	// GetETA returns the estimated time and distance between two points
	GetETA(ctx context.Context, originLat, originLng, destLat, destLng float64) (*ETAResult, error)

	// GetTrafficLevel returns the current traffic level at a location
	GetTrafficLevel(ctx context.Context, lat, lng float64) (TrafficLevel, error)
}

// ETAResult represents the result of an ETA calculation
type ETAResult struct {
	DistanceKm        float64      `json:"distance_km"`
	DistanceMeters    int          `json:"distance_meters"`
	DurationMinutes   float64      `json:"duration_minutes"`
	DurationSeconds   int          `json:"duration_seconds"`
	TrafficLevel      TrafficLevel `json:"traffic_level"`
	Confidence        float64      `json:"confidence"`
	RoutePolyline     string       `json:"route_polyline,omitempty"`
}

// ServiceETAProvider wraps the full maps service to implement ETAProvider
type ServiceETAProvider struct {
	service *Service
}

// NewServiceETAProvider creates a new ETA provider from a maps service
func NewServiceETAProvider(service *Service) *ServiceETAProvider {
	return &ServiceETAProvider{service: service}
}

// GetETA implements ETAProvider
func (p *ServiceETAProvider) GetETA(ctx context.Context, originLat, originLng, destLat, destLng float64) (*ETAResult, error) {
	req := &ETARequest{
		Origin:      Coordinate{Latitude: originLat, Longitude: originLng},
		Destination: Coordinate{Latitude: destLat, Longitude: destLng},
	}

	resp, err := p.service.GetTrafficAwareETA(ctx, req.Origin, req.Destination)
	if err != nil {
		return nil, err
	}

	return &ETAResult{
		DistanceKm:      resp.DistanceKm,
		DistanceMeters:  resp.DistanceMeters,
		DurationMinutes: resp.DurationInTraffic,
		DurationSeconds: int(resp.DurationInTraffic * 60),
		TrafficLevel:    resp.TrafficLevel,
		Confidence:      resp.Confidence,
	}, nil
}

// GetTrafficLevel implements ETAProvider
func (p *ServiceETAProvider) GetTrafficLevel(ctx context.Context, lat, lng float64) (TrafficLevel, error) {
	req := &TrafficFlowRequest{
		Location:     Coordinate{Latitude: lat, Longitude: lng},
		RadiusMeters: 5000,
	}

	resp, err := p.service.GetTrafficFlow(ctx, req)
	if err != nil {
		return TrafficModerate, err
	}

	return resp.OverallLevel, nil
}

// HaversineETAProvider is a fallback provider using Haversine formula
type HaversineETAProvider struct {
	avgSpeedKmh float64
}

// NewHaversineETAProvider creates a new Haversine-based ETA provider
func NewHaversineETAProvider(avgSpeedKmh float64) *HaversineETAProvider {
	if avgSpeedKmh <= 0 {
		avgSpeedKmh = 35.0 // Default city speed
	}
	return &HaversineETAProvider{avgSpeedKmh: avgSpeedKmh}
}

// GetETA implements ETAProvider using Haversine formula
func (p *HaversineETAProvider) GetETA(ctx context.Context, originLat, originLng, destLat, destLng float64) (*ETAResult, error) {
	distance := haversineDistance(originLat, originLng, destLat, destLng)
	durationMinutes := (distance / p.avgSpeedKmh) * 60

	return &ETAResult{
		DistanceKm:      distance,
		DistanceMeters:  int(distance * 1000),
		DurationMinutes: durationMinutes,
		DurationSeconds: int(durationMinutes * 60),
		TrafficLevel:    TrafficModerate, // Assume moderate traffic
		Confidence:      0.5,             // Low confidence for Haversine
	}, nil
}

// GetTrafficLevel implements ETAProvider - always returns moderate for Haversine
func (p *HaversineETAProvider) GetTrafficLevel(ctx context.Context, lat, lng float64) (TrafficLevel, error) {
	return TrafficModerate, nil
}

// TrafficMultiplier returns a multiplier for fare calculation based on traffic level
func (t TrafficLevel) TrafficMultiplier() float64 {
	switch t {
	case TrafficFreeFlow:
		return 1.0
	case TrafficLight:
		return 1.05
	case TrafficModerate:
		return 1.1
	case TrafficHeavy:
		return 1.2
	case TrafficSevere:
		return 1.35
	case TrafficBlocked:
		return 1.5
	default:
		return 1.1
	}
}

// IsHighTraffic returns true if traffic is heavy or worse
func (t TrafficLevel) IsHighTraffic() bool {
	return t == TrafficHeavy || t == TrafficSevere || t == TrafficBlocked
}
