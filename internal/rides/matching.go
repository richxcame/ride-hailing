package rides

import (
	"context"
	"math"
	"sort"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// MatchingConfig holds weights for the driver matching scoring algorithm.
type MatchingConfig struct {
	DistanceWeight    float64 // How much distance matters (default 0.40)
	RatingWeight      float64 // How much driver rating matters (default 0.25)
	AcceptanceWeight  float64 // How much acceptance rate matters (default 0.20)
	IdleTimeWeight    float64 // How much idle time matters (default 0.15)
	MaxDistanceKm     float64 // Max distance to consider (default 10km)
	MaxCandidates     int     // Max candidates to evaluate (default 50)
	MaxResults        int     // Max results to return (default 5)
}

// DefaultMatchingConfig returns production-tuned matching weights.
func DefaultMatchingConfig() MatchingConfig {
	return MatchingConfig{
		DistanceWeight:   0.40,
		RatingWeight:     0.25,
		AcceptanceWeight: 0.20,
		IdleTimeWeight:   0.15,
		MaxDistanceKm:    10.0,
		MaxCandidates:    50,
		MaxResults:       5,
	}
}

// DriverCandidate represents a driver being evaluated for matching.
type DriverCandidate struct {
	DriverID       uuid.UUID `json:"driver_id"`
	DistanceKm     float64   `json:"distance_km"`
	Rating         float64   `json:"rating"`
	AcceptanceRate float64   `json:"acceptance_rate"` // 0.0-1.0
	IdleMinutes    float64   `json:"idle_minutes"`    // Minutes since last ride completed
	Score          float64   `json:"score"`
}

// DriverDataProvider fetches driver metadata needed for scoring.
type DriverDataProvider interface {
	// GetNearbyDriverCandidates returns available drivers near the pickup location
	// with their distance, rating, and stats.
	GetNearbyDriverCandidates(ctx context.Context, lat, lng float64, maxDistance float64, limit int) ([]*DriverCandidate, error)
}

// Matcher scores and ranks driver candidates for a ride request.
type Matcher struct {
	cfg      MatchingConfig
	provider DriverDataProvider
}

// NewMatcher creates a new driver-rider matcher.
func NewMatcher(cfg MatchingConfig, provider DriverDataProvider) *Matcher {
	return &Matcher{cfg: cfg, provider: provider}
}

// FindBestDrivers returns the top-ranked drivers for a pickup location.
// Each candidate is scored using a weighted multi-factor algorithm:
//
//	score = w_dist * distScore + w_rating * ratingScore + w_accept * acceptScore + w_idle * idleScore
//
// Where each factor is normalized to [0, 1] with 1 being best.
func (m *Matcher) FindBestDrivers(ctx context.Context, pickupLat, pickupLng float64) ([]*DriverCandidate, error) {
	candidates, err := m.provider.GetNearbyDriverCandidates(ctx, pickupLat, pickupLng, m.cfg.MaxDistanceKm, m.cfg.MaxCandidates)
	if err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		return []*DriverCandidate{}, nil
	}

	// Find max values for normalization
	maxDist := 0.0
	maxIdle := 0.0
	for _, c := range candidates {
		if c.DistanceKm > maxDist {
			maxDist = c.DistanceKm
		}
		if c.IdleMinutes > maxIdle {
			maxIdle = c.IdleMinutes
		}
	}

	// Score each candidate
	for _, c := range candidates {
		c.Score = m.scoreCandidate(c, maxDist, maxIdle)
	}

	// Sort by score descending (highest = best match)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	// Limit results
	if len(candidates) > m.cfg.MaxResults {
		candidates = candidates[:m.cfg.MaxResults]
	}

	logger.DebugContext(ctx, "driver matching completed",
		zap.Int("candidates_evaluated", len(candidates)),
		zap.Float64("pickup_lat", pickupLat),
		zap.Float64("pickup_lng", pickupLng),
	)

	return candidates, nil
}

// scoreCandidate computes a normalized weighted score for a single driver.
func (m *Matcher) scoreCandidate(c *DriverCandidate, maxDist, maxIdle float64) float64 {
	// Distance score: closer = better. Use inverse linear scaling.
	distScore := 1.0
	if maxDist > 0 {
		distScore = 1.0 - (c.DistanceKm / maxDist)
	}

	// Rating score: higher = better. Normalize from [1, 5] to [0, 1].
	ratingScore := (c.Rating - 1.0) / 4.0
	if ratingScore < 0 {
		ratingScore = 0
	}

	// Acceptance rate score: direct [0, 1].
	acceptScore := c.AcceptanceRate

	// Idle time score: longer idle = higher priority (fairness).
	// Use logarithmic scaling to prevent extreme values from dominating.
	idleScore := 0.0
	if maxIdle > 0 && c.IdleMinutes > 0 {
		idleScore = math.Log1p(c.IdleMinutes) / math.Log1p(maxIdle)
	}

	score := m.cfg.DistanceWeight*distScore +
		m.cfg.RatingWeight*ratingScore +
		m.cfg.AcceptanceWeight*acceptScore +
		m.cfg.IdleTimeWeight*idleScore

	return math.Round(score*1000) / 1000
}
