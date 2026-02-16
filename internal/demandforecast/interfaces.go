package demandforecast

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RepositoryInterface defines the interface for demand forecast data access.
// This interface allows for mocking in tests.
type RepositoryInterface interface {
	// Historical Data
	RecordDemand(ctx context.Context, record *HistoricalDemandRecord) error
	GetHistoricalDemand(ctx context.Context, h3Index string, startTime, endTime time.Time) ([]*HistoricalDemandRecord, error)
	GetHistoricalAverage(ctx context.Context, h3Index string, hour, dayOfWeek int, weeksBack int) (float64, float64, error)
	GetRecentDemand(ctx context.Context, h3Index string, minutesBack int) (int, error)
	GetRecentDemandTrend(ctx context.Context, h3Index string) (float64, error)
	GetNeighborDemandAverage(ctx context.Context, neighborIndexes []string, minutesBack int) (float64, error)

	// Predictions
	SavePrediction(ctx context.Context, pred *DemandPrediction) error
	GetPrediction(ctx context.Context, h3Index string, predictionTime time.Time, timeframe PredictionTimeframe) (*DemandPrediction, error)
	GetLatestPrediction(ctx context.Context, h3Index string, timeframe PredictionTimeframe) (*DemandPrediction, error)
	GetPredictionsInBoundingBox(ctx context.Context, timeframe PredictionTimeframe, minDemandLevel *DemandLevel) ([]*DemandPrediction, error)
	GetTopHotspots(ctx context.Context, timeframe PredictionTimeframe, limit int) ([]*DemandPrediction, error)

	// Special Events
	CreateEvent(ctx context.Context, event *SpecialEvent) error
	GetUpcomingEvents(ctx context.Context, startTime, endTime time.Time) ([]*SpecialEvent, error)
	GetEventsNearLocation(ctx context.Context, latitude, longitude float64, radiusKm float64, timeWindow time.Duration) ([]*SpecialEvent, error)

	// Model Accuracy Tracking
	RecordPredictionAccuracy(ctx context.Context, predictionID uuid.UUID, actualRides int) error
	GetAccuracyMetrics(ctx context.Context, timeframe PredictionTimeframe, daysBack int) (*ForecastAccuracyMetrics, error)
	CleanupOldPredictions(ctx context.Context, daysOld int) (int64, error)
}

// Ensure Repository implements RepositoryInterface
var _ RepositoryInterface = (*Repository)(nil)
