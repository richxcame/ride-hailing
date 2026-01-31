package rides

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/httpclient"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/resilience"
	"go.uber.org/zap"
)

// GeoMatchingProvider implements DriverDataProvider by calling the geo service
// for nearby drivers and the local DB for driver stats.
type GeoMatchingProvider struct {
	geoClient *httpclient.Client
	repo      *Repository
	breaker   *resilience.CircuitBreaker
}

// NewGeoMatchingProvider creates a provider backed by the geo service + rides DB.
func NewGeoMatchingProvider(geoClient *httpclient.Client, repo *Repository) *GeoMatchingProvider {
	return &GeoMatchingProvider{geoClient: geoClient, repo: repo}
}

// SetCircuitBreaker enables circuit breaker protection for geo service calls.
func (p *GeoMatchingProvider) SetCircuitBreaker(cb *resilience.CircuitBreaker) {
	p.breaker = cb
}

type nearbyDriverResponse struct {
	DriverID  uuid.UUID `json:"driver_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	H3Cell    string    `json:"h3_cell"`
}

type geoAPIResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Drivers []nearbyDriverResponse `json:"drivers"`
	} `json:"data"`
}

// GetNearbyDriverCandidates fetches nearby available drivers from geo service,
// then enriches them with DB stats (rating, acceptance rate, idle time).
func (p *GeoMatchingProvider) GetNearbyDriverCandidates(ctx context.Context, lat, lng float64, maxDistance float64, limit int) ([]*DriverCandidate, error) {
	// Call geo service for nearby drivers (with circuit breaker if configured)
	path := fmt.Sprintf("/api/v1/geo/drivers/nearby?latitude=%f&longitude=%f&limit=%d", lat, lng, limit)

	var body []byte
	var err error

	if p.breaker != nil {
		result, cbErr := p.breaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
			return p.geoClient.Get(ctx, path, nil)
		})
		if cbErr != nil {
			logger.WarnContext(ctx, "geo service call failed (circuit breaker)", zap.Error(cbErr))
			return []*DriverCandidate{}, nil
		}
		body = result.([]byte)
	} else {
		body, err = p.geoClient.Get(ctx, path, nil)
		if err != nil {
			logger.WarnContext(ctx, "failed to fetch nearby drivers from geo service", zap.Error(err))
			return []*DriverCandidate{}, nil
		}
	}

	var resp geoAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse geo response: %w", err)
	}

	if len(resp.Data.Drivers) == 0 {
		return []*DriverCandidate{}, nil
	}

	// Collect driver IDs
	driverIDs := make([]uuid.UUID, len(resp.Data.Drivers))
	for i, d := range resp.Data.Drivers {
		driverIDs[i] = d.DriverID
	}

	// Fetch stats from DB
	stats, err := p.repo.GetDriverMatchStats(ctx, driverIDs)
	if err != nil {
		logger.WarnContext(ctx, "failed to fetch driver stats, using defaults", zap.Error(err))
		stats = make(map[uuid.UUID]*DriverMatchStats)
	}

	// Build candidates
	candidates := make([]*DriverCandidate, 0, len(resp.Data.Drivers))
	for _, d := range resp.Data.Drivers {
		dist := haversine(lat, lng, d.Latitude, d.Longitude)
		if dist > maxDistance {
			continue
		}

		s := stats[d.DriverID]
		if s == nil {
			s = &DriverMatchStats{Rating: 4.0, AcceptanceRate: 0.8, IdleMinutes: 30.0}
		}

		candidates = append(candidates, &DriverCandidate{
			DriverID:       d.DriverID,
			DistanceKm:     dist,
			Rating:         s.Rating,
			AcceptanceRate: s.AcceptanceRate,
			IdleMinutes:    s.IdleMinutes,
		})
	}

	return candidates, nil
}

// haversine calculates distance between two coordinates in km.
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	return calculateDistance(lat1, lon1, lat2, lon2)
}
