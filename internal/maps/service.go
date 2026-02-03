package maps

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/richxcame/ride-hailing/pkg/logger"
	redisclient "github.com/richxcame/ride-hailing/pkg/redis"
	"github.com/richxcame/ride-hailing/pkg/resilience"
	"go.uber.org/zap"
)

// Service provides maps functionality with caching, fallbacks, and resilience
type Service struct {
	primary        MapsProvider
	fallbacks      []MapsProvider
	redis          redisclient.ClientInterface
	config         Config
	breakers       map[Provider]*resilience.CircuitBreaker
	mu             sync.RWMutex
	trafficCache   map[string]*TrafficFlowResponse
	trafficCacheTTL time.Duration
}

// NewService creates a new maps service
func NewService(config Config, redis redisclient.ClientInterface) (*Service, error) {
	s := &Service{
		redis:          redis,
		config:         config,
		breakers:       make(map[Provider]*resilience.CircuitBreaker),
		trafficCache:   make(map[string]*TrafficFlowResponse),
		trafficCacheTTL: time.Duration(config.TrafficRefreshSeconds) * time.Second,
	}

	// Initialize primary provider
	primary, err := s.createProvider(config.Primary)
	if err != nil {
		return nil, fmt.Errorf("failed to create primary provider: %w", err)
	}
	s.primary = primary

	// Initialize fallback providers
	for _, fc := range config.Fallbacks {
		fallback, err := s.createProvider(fc)
		if err != nil {
			logger.Warn("Failed to create fallback provider", zap.Error(err), zap.String("provider", string(fc.Provider)))
			continue
		}
		s.fallbacks = append(s.fallbacks, fallback)
	}

	// Initialize circuit breakers
	s.initCircuitBreakers()

	return s, nil
}

func (s *Service) createProvider(config ProviderConfig) (MapsProvider, error) {
	switch config.Provider {
	case ProviderGoogle:
		return NewGoogleMapsProvider(config), nil
	case ProviderHERE:
		return NewHEREMapsProvider(config), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
}

func (s *Service) initCircuitBreakers() {
	// Create breaker for primary
	s.breakers[s.primary.Name()] = resilience.NewCircuitBreaker(
		resilience.BuildSettings(
			fmt.Sprintf("maps-%s", s.primary.Name()),
			60, // interval seconds
			30, // timeout seconds
			5,  // failure threshold
			2,  // success threshold
		),
		nil,
	)

	// Create breakers for fallbacks
	for _, fb := range s.fallbacks {
		s.breakers[fb.Name()] = resilience.NewCircuitBreaker(
			resilience.BuildSettings(
				fmt.Sprintf("maps-%s", fb.Name()),
				60, 30, 5, 2,
			),
			nil,
		)
	}
}

// GetRoute calculates a route with caching and fallback support
func (s *Service) GetRoute(ctx context.Context, req *RouteRequest) (*RouteResponse, error) {
	// Try cache first
	if s.config.CacheEnabled {
		cacheKey := s.routeCacheKey(req)
		if cached, err := s.getFromCache(ctx, cacheKey); err == nil {
			var resp RouteResponse
			if err := json.Unmarshal(cached, &resp); err == nil {
				resp.CacheHit = true
				return &resp, nil
			}
		}
	}

	// Try primary provider
	resp, err := s.executeWithFallback(ctx, func(ctx context.Context, provider MapsProvider) (interface{}, error) {
		return provider.GetRoute(ctx, req)
	})

	if err != nil {
		return nil, err
	}

	routeResp := resp.(*RouteResponse)

	// Cache the result
	if s.config.CacheEnabled && len(routeResp.Routes) > 0 {
		cacheKey := s.routeCacheKey(req)
		if data, err := json.Marshal(routeResp); err == nil {
			_ = s.setCache(ctx, cacheKey, data, time.Duration(s.config.CacheTTLSeconds)*time.Second)
		}
	}

	return routeResp, nil
}

// GetETA calculates ETA with caching and fallback support
func (s *Service) GetETA(ctx context.Context, req *ETARequest) (*ETAResponse, error) {
	// Try cache first (shorter TTL for ETA due to traffic changes)
	if s.config.CacheEnabled {
		cacheKey := s.etaCacheKey(req)
		if cached, err := s.getFromCache(ctx, cacheKey); err == nil {
			var resp ETAResponse
			if err := json.Unmarshal(cached, &resp); err == nil {
				resp.CacheHit = true
				return &resp, nil
			}
		}
	}

	// Try primary provider
	resp, err := s.executeWithFallback(ctx, func(ctx context.Context, provider MapsProvider) (interface{}, error) {
		return provider.GetETA(ctx, req)
	})

	if err != nil {
		// Ultimate fallback: calculate using Haversine
		return s.fallbackETA(req), nil
	}

	etaResp := resp.(*ETAResponse)

	// Cache with shorter TTL for ETA
	if s.config.CacheEnabled {
		cacheKey := s.etaCacheKey(req)
		if data, err := json.Marshal(etaResp); err == nil {
			// ETA cache for 2 minutes
			_ = s.setCache(ctx, cacheKey, data, 2*time.Minute)
		}
	}

	return etaResp, nil
}

// GetDistanceMatrix calculates distances between multiple points
func (s *Service) GetDistanceMatrix(ctx context.Context, req *DistanceMatrixRequest) (*DistanceMatrixResponse, error) {
	resp, err := s.executeWithFallback(ctx, func(ctx context.Context, provider MapsProvider) (interface{}, error) {
		return provider.GetDistanceMatrix(ctx, req)
	})

	if err != nil {
		return nil, err
	}

	return resp.(*DistanceMatrixResponse), nil
}

// GetTrafficFlow returns traffic flow information
func (s *Service) GetTrafficFlow(ctx context.Context, req *TrafficFlowRequest) (*TrafficFlowResponse, error) {
	// Check in-memory cache first
	cacheKey := s.trafficCacheKey(req)
	s.mu.RLock()
	if cached, ok := s.trafficCache[cacheKey]; ok {
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	resp, err := s.executeWithFallback(ctx, func(ctx context.Context, provider MapsProvider) (interface{}, error) {
		return provider.GetTrafficFlow(ctx, req)
	})

	if err != nil {
		return &TrafficFlowResponse{
			OverallLevel: TrafficModerate,
			UpdatedAt:    time.Now(),
		}, nil
	}

	trafficResp := resp.(*TrafficFlowResponse)

	// Update in-memory cache
	s.mu.Lock()
	s.trafficCache[cacheKey] = trafficResp
	s.mu.Unlock()

	// Schedule cache cleanup
	go func() {
		time.Sleep(s.trafficCacheTTL)
		s.mu.Lock()
		delete(s.trafficCache, cacheKey)
		s.mu.Unlock()
	}()

	return trafficResp, nil
}

// GetTrafficIncidents returns traffic incidents in an area
func (s *Service) GetTrafficIncidents(ctx context.Context, req *TrafficIncidentsRequest) (*TrafficIncidentsResponse, error) {
	resp, err := s.executeWithFallback(ctx, func(ctx context.Context, provider MapsProvider) (interface{}, error) {
		return provider.GetTrafficIncidents(ctx, req)
	})

	if err != nil {
		return &TrafficIncidentsResponse{
			Incidents: []TrafficIncident{},
			UpdatedAt: time.Now(),
		}, nil
	}

	return resp.(*TrafficIncidentsResponse), nil
}

// Geocode converts an address to coordinates
func (s *Service) Geocode(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error) {
	// Try cache first
	if s.config.CacheEnabled {
		cacheKey := s.geocodeCacheKey(req)
		if cached, err := s.getFromCache(ctx, cacheKey); err == nil {
			var resp GeocodingResponse
			if err := json.Unmarshal(cached, &resp); err == nil {
				return &resp, nil
			}
		}
	}

	resp, err := s.executeWithFallback(ctx, func(ctx context.Context, provider MapsProvider) (interface{}, error) {
		return provider.Geocode(ctx, req)
	})

	if err != nil {
		return nil, err
	}

	geoResp := resp.(*GeocodingResponse)

	// Cache geocoding results for longer (24 hours)
	if s.config.CacheEnabled && len(geoResp.Results) > 0 {
		cacheKey := s.geocodeCacheKey(req)
		if data, err := json.Marshal(geoResp); err == nil {
			_ = s.setCache(ctx, cacheKey, data, 24*time.Hour)
		}
	}

	return geoResp, nil
}

// ReverseGeocode converts coordinates to an address
func (s *Service) ReverseGeocode(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error) {
	// Try cache first
	if s.config.CacheEnabled && req.Coordinate != nil {
		cacheKey := s.reverseGeocodeCacheKey(req)
		if cached, err := s.getFromCache(ctx, cacheKey); err == nil {
			var resp GeocodingResponse
			if err := json.Unmarshal(cached, &resp); err == nil {
				return &resp, nil
			}
		}
	}

	resp, err := s.executeWithFallback(ctx, func(ctx context.Context, provider MapsProvider) (interface{}, error) {
		return provider.ReverseGeocode(ctx, req)
	})

	if err != nil {
		return nil, err
	}

	geoResp := resp.(*GeocodingResponse)

	// Cache reverse geocoding results
	if s.config.CacheEnabled && len(geoResp.Results) > 0 && req.Coordinate != nil {
		cacheKey := s.reverseGeocodeCacheKey(req)
		if data, err := json.Marshal(geoResp); err == nil {
			_ = s.setCache(ctx, cacheKey, data, 24*time.Hour)
		}
	}

	return geoResp, nil
}

// SearchPlaces searches for nearby places
func (s *Service) SearchPlaces(ctx context.Context, req *PlaceSearchRequest) (*PlaceSearchResponse, error) {
	resp, err := s.executeWithFallback(ctx, func(ctx context.Context, provider MapsProvider) (interface{}, error) {
		return provider.SearchPlaces(ctx, req)
	})

	if err != nil {
		return nil, err
	}

	return resp.(*PlaceSearchResponse), nil
}

// SnapToRoad snaps GPS coordinates to the nearest road
func (s *Service) SnapToRoad(ctx context.Context, req *SnapToRoadRequest) (*SnapToRoadResponse, error) {
	resp, err := s.executeWithFallback(ctx, func(ctx context.Context, provider MapsProvider) (interface{}, error) {
		return provider.SnapToRoad(ctx, req)
	})

	if err != nil {
		return nil, err
	}

	return resp.(*SnapToRoadResponse), nil
}

// GetSpeedLimits returns speed limits along a path
func (s *Service) GetSpeedLimits(ctx context.Context, req *SpeedLimitsRequest) (*SpeedLimitsResponse, error) {
	resp, err := s.executeWithFallback(ctx, func(ctx context.Context, provider MapsProvider) (interface{}, error) {
		return provider.GetSpeedLimits(ctx, req)
	})

	if err != nil {
		return nil, err
	}

	return resp.(*SpeedLimitsResponse), nil
}

// HealthCheck checks the health of all providers
func (s *Service) HealthCheck(ctx context.Context) map[Provider]error {
	results := make(map[Provider]error)

	// Check primary
	results[s.primary.Name()] = s.primary.HealthCheck(ctx)

	// Check fallbacks
	for _, fb := range s.fallbacks {
		results[fb.Name()] = fb.HealthCheck(ctx)
	}

	return results
}

// GetPrimaryProvider returns the name of the primary provider
func (s *Service) GetPrimaryProvider() Provider {
	return s.primary.Name()
}

// executeWithFallback executes a function with the primary provider and falls back to others on failure
func (s *Service) executeWithFallback(ctx context.Context, fn func(context.Context, MapsProvider) (interface{}, error)) (interface{}, error) {
	providers := append([]MapsProvider{s.primary}, s.fallbacks...)

	var lastErr error
	for _, provider := range providers {
		breaker := s.breakers[provider.Name()]
		if breaker == nil {
			// No breaker, execute directly
			result, err := fn(ctx, provider)
			if err == nil {
				return result, nil
			}
			lastErr = err
			logger.Warn("Maps provider failed", zap.Error(err), zap.String("provider", string(provider.Name())))
			continue
		}

		// Execute with circuit breaker
		result, err := breaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
			return fn(ctx, provider)
		})

		if err == nil {
			return result, nil
		}

		lastErr = err
		logger.Warn("Maps provider failed", zap.Error(err), zap.String("provider", string(provider.Name())))
	}

	return nil, fmt.Errorf("all maps providers failed: %w", lastErr)
}

// fallbackETA calculates ETA using Haversine formula when all providers fail
func (s *Service) fallbackETA(req *ETARequest) *ETAResponse {
	distance := haversineDistance(
		req.Origin.Latitude, req.Origin.Longitude,
		req.Destination.Latitude, req.Destination.Longitude,
	)

	// Assume average speed of 30 km/h for city traffic (conservative)
	const avgSpeed = 30.0
	durationMinutes := (distance / avgSpeed) * 60

	// Add 20% buffer for traffic
	durationMinutes *= 1.2

	return &ETAResponse{
		DistanceKm:        distance,
		DistanceMeters:    int(distance * 1000),
		DurationMinutes:   durationMinutes,
		DurationSeconds:   int(durationMinutes * 60),
		DurationInTraffic: durationMinutes,
		TrafficLevel:      TrafficModerate,
		EstimatedArrival:  time.Now().Add(time.Duration(durationMinutes) * time.Minute),
		Provider:          "fallback",
		Confidence:        0.5, // Low confidence for fallback
	}
}

// haversineDistance calculates the great-circle distance between two points
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0 // km

	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180.0)*math.Cos(lat2*math.Pi/180.0)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// Cache key generation

func (s *Service) routeCacheKey(req *RouteRequest) string {
	data := fmt.Sprintf("route:%f,%f:%f,%f:%v:%v:%v",
		req.Origin.Latitude, req.Origin.Longitude,
		req.Destination.Latitude, req.Destination.Longitude,
		req.AvoidTolls, req.AvoidHighways, req.AvoidFerries,
	)
	return s.config.CachePrefix + s.hashKey(data)
}

func (s *Service) etaCacheKey(req *ETARequest) string {
	data := fmt.Sprintf("eta:%f,%f:%f,%f",
		req.Origin.Latitude, req.Origin.Longitude,
		req.Destination.Latitude, req.Destination.Longitude,
	)
	return s.config.CachePrefix + s.hashKey(data)
}

func (s *Service) geocodeCacheKey(req *GeocodingRequest) string {
	data := fmt.Sprintf("geo:%s:%s:%s", req.Address, req.Language, req.Region)
	return s.config.CachePrefix + s.hashKey(data)
}

func (s *Service) reverseGeocodeCacheKey(req *GeocodingRequest) string {
	// Round coordinates to 5 decimal places (~1m precision)
	lat := math.Round(req.Coordinate.Latitude*100000) / 100000
	lng := math.Round(req.Coordinate.Longitude*100000) / 100000
	data := fmt.Sprintf("rgeo:%f,%f:%s", lat, lng, req.Language)
	return s.config.CachePrefix + s.hashKey(data)
}

func (s *Service) trafficCacheKey(req *TrafficFlowRequest) string {
	// Round coordinates for cache key
	lat := math.Round(req.Location.Latitude*1000) / 1000
	lng := math.Round(req.Location.Longitude*1000) / 1000
	return fmt.Sprintf("traffic:%f,%f", lat, lng)
}

func (s *Service) hashKey(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes
}

// Redis cache operations

func (s *Service) getFromCache(ctx context.Context, key string) ([]byte, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("redis not available")
	}

	val, err := s.redis.GetString(ctx, key)
	if err != nil {
		return nil, err
	}

	return []byte(val), nil
}

func (s *Service) setCache(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	if s.redis == nil {
		return nil
	}

	return s.redis.SetWithExpiration(ctx, key, string(data), ttl)
}

// GetTrafficAwareETA calculates ETA with real-time traffic data
func (s *Service) GetTrafficAwareETA(ctx context.Context, origin, destination Coordinate) (*ETAResponse, error) {
	// Get traffic flow for the origin area
	trafficReq := &TrafficFlowRequest{
		Location:     origin,
		RadiusMeters: 5000,
	}

	traffic, _ := s.GetTrafficFlow(ctx, trafficReq)

	// Get base ETA
	etaReq := &ETARequest{
		Origin:      origin,
		Destination: destination,
	}

	eta, err := s.GetETA(ctx, etaReq)
	if err != nil {
		return nil, err
	}

	// Adjust based on traffic level if we have data
	if traffic != nil && traffic.OverallLevel != "" {
		eta.TrafficLevel = traffic.OverallLevel
	}

	return eta, nil
}

// BatchGetETA calculates ETAs for multiple origin-destination pairs efficiently
func (s *Service) BatchGetETA(ctx context.Context, requests []*ETARequest) ([]*ETAResponse, error) {
	responses := make([]*ETAResponse, len(requests))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var lastErr error

	for i, req := range requests {
		wg.Add(1)
		go func(idx int, r *ETARequest) {
			defer wg.Done()

			resp, err := s.GetETA(ctx, r)
			mu.Lock()
			if err != nil {
				lastErr = err
				responses[idx] = s.fallbackETA(r)
			} else {
				responses[idx] = resp
			}
			mu.Unlock()
		}(i, req)
	}

	wg.Wait()

	if lastErr != nil {
		logger.Warn("Some batch ETA requests failed", zap.Error(lastErr))
	}

	return responses, nil
}

// OptimizeWaypoints returns the optimal order for visiting waypoints
func (s *Service) OptimizeWaypoints(ctx context.Context, origin Coordinate, waypoints []Coordinate, destination Coordinate) ([]int, error) {
	if len(waypoints) == 0 {
		return []int{}, nil
	}

	if len(waypoints) == 1 {
		return []int{0}, nil
	}

	// Get distance matrix
	allPoints := append([]Coordinate{origin}, waypoints...)
	allPoints = append(allPoints, destination)

	matrixReq := &DistanceMatrixRequest{
		Origins:      allPoints,
		Destinations: allPoints,
	}

	matrix, err := s.GetDistanceMatrix(ctx, matrixReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get distance matrix: %w", err)
	}

	// Simple greedy nearest neighbor algorithm
	// For production, consider implementing 2-opt or using OR-Tools
	visited := make(map[int]bool)
	order := make([]int, 0, len(waypoints))
	current := 0 // Start from origin

	for len(order) < len(waypoints) {
		nearestIdx := -1
		nearestDist := math.MaxFloat64

		for j := 1; j <= len(waypoints); j++ {
			if visited[j-1] {
				continue
			}

			dist := matrix.Rows[current].Elements[j].DistanceKm
			if dist < nearestDist {
				nearestDist = dist
				nearestIdx = j - 1
			}
		}

		if nearestIdx >= 0 {
			visited[nearestIdx] = true
			order = append(order, nearestIdx)
			current = nearestIdx + 1
		}
	}

	return order, nil
}
