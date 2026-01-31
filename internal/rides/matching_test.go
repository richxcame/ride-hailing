package rides

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// mockDataProvider implements DriverDataProvider for testing.
type mockDataProvider struct {
	candidates []*DriverCandidate
	err        error
}

func (m *mockDataProvider) GetNearbyDriverCandidates(_ context.Context, _, _ float64, _ float64, _ int) ([]*DriverCandidate, error) {
	return m.candidates, m.err
}

func TestMatcher_FindBestDrivers_Empty(t *testing.T) {
	provider := &mockDataProvider{candidates: []*DriverCandidate{}}
	matcher := NewMatcher(DefaultMatchingConfig(), provider)

	results, err := matcher.FindBestDrivers(context.Background(), 37.7749, -122.4194)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestMatcher_FindBestDrivers_SingleCandidate(t *testing.T) {
	provider := &mockDataProvider{
		candidates: []*DriverCandidate{
			{DriverID: uuid.New(), DistanceKm: 1.0, Rating: 4.5, AcceptanceRate: 0.9, IdleMinutes: 10},
		},
	}
	matcher := NewMatcher(DefaultMatchingConfig(), provider)

	results, err := matcher.FindBestDrivers(context.Background(), 37.7749, -122.4194)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Score <= 0 {
		t.Fatalf("expected positive score, got %f", results[0].Score)
	}
}

func TestMatcher_FindBestDrivers_RankedByScore(t *testing.T) {
	// Close driver with low rating vs far driver with high rating
	closeDriver := &DriverCandidate{
		DriverID: uuid.New(), DistanceKm: 0.5, Rating: 2.0, AcceptanceRate: 0.5, IdleMinutes: 5,
	}
	farGoodDriver := &DriverCandidate{
		DriverID: uuid.New(), DistanceKm: 5.0, Rating: 5.0, AcceptanceRate: 0.95, IdleMinutes: 30,
	}
	midDriver := &DriverCandidate{
		DriverID: uuid.New(), DistanceKm: 2.0, Rating: 4.5, AcceptanceRate: 0.85, IdleMinutes: 15,
	}

	provider := &mockDataProvider{
		candidates: []*DriverCandidate{closeDriver, farGoodDriver, midDriver},
	}
	matcher := NewMatcher(DefaultMatchingConfig(), provider)

	results, err := matcher.FindBestDrivers(context.Background(), 37.7749, -122.4194)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Verify descending score order
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Fatalf("results not sorted by score: index %d (%f) > index %d (%f)",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
}

func TestMatcher_FindBestDrivers_RespectsMaxResults(t *testing.T) {
	candidates := make([]*DriverCandidate, 10)
	for i := range candidates {
		candidates[i] = &DriverCandidate{
			DriverID: uuid.New(), DistanceKm: float64(i + 1), Rating: 4.0, AcceptanceRate: 0.8, IdleMinutes: float64(i * 5),
		}
	}

	cfg := DefaultMatchingConfig()
	cfg.MaxResults = 3
	provider := &mockDataProvider{candidates: candidates}
	matcher := NewMatcher(cfg, provider)

	results, err := matcher.FindBestDrivers(context.Background(), 37.7749, -122.4194)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results (MaxResults), got %d", len(results))
	}
}

func TestMatcher_ScoreCandidate_DistanceWeight(t *testing.T) {
	cfg := MatchingConfig{
		DistanceWeight:   1.0,
		RatingWeight:     0.0,
		AcceptanceWeight: 0.0,
		IdleTimeWeight:   0.0,
	}
	matcher := &Matcher{cfg: cfg}

	close := &DriverCandidate{DistanceKm: 1.0, Rating: 3.0, AcceptanceRate: 0.5, IdleMinutes: 0}
	far := &DriverCandidate{DistanceKm: 9.0, Rating: 3.0, AcceptanceRate: 0.5, IdleMinutes: 0}

	closeScore := matcher.scoreCandidate(close, 10.0, 1.0)
	farScore := matcher.scoreCandidate(far, 10.0, 1.0)

	if closeScore <= farScore {
		t.Fatalf("closer driver should score higher: close=%f, far=%f", closeScore, farScore)
	}
}

func TestMatcher_ScoreCandidate_RatingWeight(t *testing.T) {
	cfg := MatchingConfig{
		DistanceWeight:   0.0,
		RatingWeight:     1.0,
		AcceptanceWeight: 0.0,
		IdleTimeWeight:   0.0,
	}
	matcher := &Matcher{cfg: cfg}

	highRated := &DriverCandidate{DistanceKm: 5.0, Rating: 5.0, AcceptanceRate: 0.5, IdleMinutes: 10}
	lowRated := &DriverCandidate{DistanceKm: 5.0, Rating: 2.0, AcceptanceRate: 0.5, IdleMinutes: 10}

	highScore := matcher.scoreCandidate(highRated, 10.0, 30.0)
	lowScore := matcher.scoreCandidate(lowRated, 10.0, 30.0)

	if highScore <= lowScore {
		t.Fatalf("higher-rated driver should score higher: high=%f, low=%f", highScore, lowScore)
	}
}

func TestMatcher_ScoreCandidate_IdleTimeFairness(t *testing.T) {
	cfg := MatchingConfig{
		DistanceWeight:   0.0,
		RatingWeight:     0.0,
		AcceptanceWeight: 0.0,
		IdleTimeWeight:   1.0,
	}
	matcher := &Matcher{cfg: cfg}

	longIdle := &DriverCandidate{DistanceKm: 5.0, Rating: 4.0, AcceptanceRate: 0.8, IdleMinutes: 60}
	shortIdle := &DriverCandidate{DistanceKm: 5.0, Rating: 4.0, AcceptanceRate: 0.8, IdleMinutes: 5}

	longScore := matcher.scoreCandidate(longIdle, 10.0, 60.0)
	shortScore := matcher.scoreCandidate(shortIdle, 10.0, 60.0)

	if longScore <= shortScore {
		t.Fatalf("longer-idle driver should score higher (fairness): long=%f, short=%f", longScore, shortScore)
	}
}

func TestMatcher_ScoreCandidate_AllZeros(t *testing.T) {
	cfg := DefaultMatchingConfig()
	matcher := &Matcher{cfg: cfg}

	c := &DriverCandidate{DistanceKm: 0, Rating: 1.0, AcceptanceRate: 0, IdleMinutes: 0}
	score := matcher.scoreCandidate(c, 0, 0)

	// Should not panic, score should be non-negative
	if score < 0 {
		t.Fatalf("score should be non-negative, got %f", score)
	}
}
