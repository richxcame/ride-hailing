package maps

import (
	"context"
)

// MapsProvider defines the interface for maps service providers
type MapsProvider interface {
	// Core routing
	GetRoute(ctx context.Context, req *RouteRequest) (*RouteResponse, error)
	GetETA(ctx context.Context, req *ETARequest) (*ETAResponse, error)
	GetDistanceMatrix(ctx context.Context, req *DistanceMatrixRequest) (*DistanceMatrixResponse, error)

	// Traffic
	GetTrafficFlow(ctx context.Context, req *TrafficFlowRequest) (*TrafficFlowResponse, error)
	GetTrafficIncidents(ctx context.Context, req *TrafficIncidentsRequest) (*TrafficIncidentsResponse, error)

	// Geocoding
	Geocode(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error)
	ReverseGeocode(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error)

	// Places
	SearchPlaces(ctx context.Context, req *PlaceSearchRequest) (*PlaceSearchResponse, error)

	// Roads
	SnapToRoad(ctx context.Context, req *SnapToRoadRequest) (*SnapToRoadResponse, error)
	GetSpeedLimits(ctx context.Context, req *SpeedLimitsRequest) (*SpeedLimitsResponse, error)

	// Health
	HealthCheck(ctx context.Context) error
	Name() Provider
}

// ProviderConfig holds configuration for a maps provider
type ProviderConfig struct {
	Provider        Provider `json:"provider"`
	APIKey          string   `json:"api_key"`
	BaseURL         string   `json:"base_url,omitempty"`
	Timeout         int      `json:"timeout_seconds,omitempty"`
	MaxRetries      int      `json:"max_retries,omitempty"`
	RateLimitPerSec int      `json:"rate_limit_per_sec,omitempty"`
}

// Config holds the overall maps service configuration
type Config struct {
	// Primary provider for routing
	Primary         ProviderConfig `json:"primary"`

	// Fallback providers (in order of preference)
	Fallbacks       []ProviderConfig `json:"fallbacks,omitempty"`

	// Caching settings
	CacheEnabled    bool   `json:"cache_enabled"`
	CacheTTLSeconds int    `json:"cache_ttl_seconds"`
	CachePrefix     string `json:"cache_prefix"`

	// Traffic settings
	TrafficEnabled          bool `json:"traffic_enabled"`
	TrafficRefreshSeconds   int  `json:"traffic_refresh_seconds"`

	// Feature flags
	EnableRouteAlternatives bool `json:"enable_route_alternatives"`
	EnableSpeedLimits       bool `json:"enable_speed_limits"`
	EnableIncidentAlerts    bool `json:"enable_incident_alerts"`

	// Quality settings
	MinConfidenceThreshold  float64 `json:"min_confidence_threshold"` // Minimum acceptable confidence
	MaxETADeviationPercent  float64 `json:"max_eta_deviation_percent"` // Max deviation from cached ETA before refresh
}

// DefaultConfig returns sensible defaults for maps configuration
func DefaultConfig() Config {
	return Config{
		CacheEnabled:            true,
		CacheTTLSeconds:         300, // 5 minutes
		CachePrefix:             "maps:",
		TrafficEnabled:          true,
		TrafficRefreshSeconds:   60,
		EnableRouteAlternatives: true,
		EnableSpeedLimits:       false, // Requires additional API quota
		EnableIncidentAlerts:    true,
		MinConfidenceThreshold:  0.7,
		MaxETADeviationPercent:  25,
	}
}
