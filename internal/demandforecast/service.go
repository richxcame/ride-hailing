package demandforecast

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/internal/geo"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/uber/h3-go/v4"
	"go.uber.org/zap"
)

// WeatherService interface for fetching weather data
type WeatherService interface {
	GetCurrentWeather(ctx context.Context, latitude, longitude float64) (*WeatherData, error)
	GetForecast(ctx context.Context, latitude, longitude float64, hours int) ([]WeatherData, error)
}

// DriverLocationService interface for getting driver counts
type DriverLocationService interface {
	GetDriverCountInCell(ctx context.Context, h3Index string) (int, error)
	GetNearbyDriverCount(ctx context.Context, latitude, longitude float64, radiusKm float64) (int, error)
}

// Config holds service configuration
type Config struct {
	H3Resolution        int
	TrainingDataWeeks   int
	PredictionIntervals []PredictionTimeframe
	WeatherEnabled      bool
	EventsEnabled       bool
	MinConfidence       float64
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		H3Resolution:        geo.H3ResolutionDemand,
		TrainingDataWeeks:   8,
		PredictionIntervals: []PredictionTimeframe{Timeframe15Min, Timeframe30Min, Timeframe1Hour},
		WeatherEnabled:      true,
		EventsEnabled:       true,
		MinConfidence:       0.5,
	}
}

// Service handles demand forecasting
type Service struct {
	repo           RepositoryInterface
	weatherSvc     WeatherService
	driverSvc      DriverLocationService
	config         *Config
	modelWeights   *ModelWeights
	mu             sync.RWMutex
}

// ModelWeights holds feature weights for the prediction model
type ModelWeights struct {
	HistoricalPattern float64
	RecentTrend       float64
	TimeOfDay         float64
	DayOfWeek         float64
	Weather           float64
	Events            float64
	Seasonal          float64
}

// DefaultModelWeights returns default weights
func DefaultModelWeights() *ModelWeights {
	return &ModelWeights{
		HistoricalPattern: 0.35,
		RecentTrend:       0.25,
		TimeOfDay:         0.15,
		DayOfWeek:         0.10,
		Weather:           0.08,
		Events:            0.05,
		Seasonal:          0.02,
	}
}

// NewService creates a new demand forecast service
func NewService(repo RepositoryInterface, weatherSvc WeatherService, driverSvc DriverLocationService, config *Config) *Service {
	if config == nil {
		config = DefaultConfig()
	}

	return &Service{
		repo:         repo,
		weatherSvc:   weatherSvc,
		driverSvc:    driverSvc,
		config:       config,
		modelWeights: DefaultModelWeights(),
	}
}

// ========================================
// PREDICTION GENERATION
// ========================================

// GeneratePrediction generates a demand prediction for a location and timeframe
func (s *Service) GeneratePrediction(ctx context.Context, latitude, longitude float64, timeframe PredictionTimeframe) (*DemandPrediction, error) {
	h3Index := geo.LatLngToCell(latitude, longitude, s.config.H3Resolution).String()
	targetTime := s.getTargetTime(timeframe)

	// Build features
	features, err := s.buildFeatures(ctx, h3Index, targetTime)
	if err != nil {
		return nil, err
	}

	// Run prediction model
	prediction := s.predict(features)

	// Create prediction record
	pred := &DemandPrediction{
		ID:                  uuid.New(),
		H3Index:             h3Index,
		PredictionTime:      targetTime,
		GeneratedAt:         time.Now(),
		Timeframe:           timeframe,
		PredictedRides:      prediction.PredictedRides,
		DemandLevel:         s.classifyDemandLevel(prediction.PredictedRides, features.HistAvgRides),
		Confidence:          prediction.Confidence,
		RecommendedDrivers:  s.calculateRecommendedDrivers(prediction.PredictedRides),
		ExpectedSurge:       s.calculateExpectedSurge(prediction.PredictedRides, features.CurrentDrivers),
		HotspotScore:        s.calculateHotspotScore(prediction.PredictedRides, features),
		RepositionPriority:  s.calculateRepositionPriority(prediction.PredictedRides, features),
		FeatureContributions: s.calculateContributions(features, prediction),
	}

	// Save prediction
	if err := s.repo.SavePrediction(ctx, pred); err != nil {
		logger.Error("failed to save prediction", zap.Error(err))
	}

	return pred, nil
}

// GeneratePredictionsForArea generates predictions for all cells in a bounding box
func (s *Service) GeneratePredictionsForArea(ctx context.Context, minLatitude, minLongitude, maxLatitude, maxLongitude float64, timeframe PredictionTimeframe) ([]*DemandPrediction, error) {
	// Get all H3 cells that cover the area
	cells := s.getCellsInBoundingBox(minLatitude, minLongitude, maxLatitude, maxLongitude)

	var predictions []*DemandPrediction
	for _, cell := range cells {
		latitude, longitude := geo.CellToLatLng(cell)
		pred, err := s.GeneratePrediction(ctx, latitude, longitude, timeframe)
		if err != nil {
			continue
		}
		predictions = append(predictions, pred)
	}

	return predictions, nil
}

// buildFeatures constructs the feature vector for prediction
func (s *Service) buildFeatures(ctx context.Context, h3Index string, targetTime time.Time) (*ModelFeatures, error) {
	hour := targetTime.Hour()
	dayOfWeek := int(targetTime.Weekday())
	isWeekend := dayOfWeek == 0 || dayOfWeek == 6

	// Historical average for this hour/day
	histAvg, histStd, _ := s.repo.GetHistoricalAverage(ctx, h3Index, hour, dayOfWeek, s.config.TrainingDataWeeks)

	// Recent demand
	recentRides15min, _ := s.repo.GetRecentDemand(ctx, h3Index, 15)
	recentRides1hr, _ := s.repo.GetRecentDemand(ctx, h3Index, 60)

	// Recent trend
	recentTrend, _ := s.repo.GetRecentDemandTrend(ctx, h3Index)

	// Current drivers
	var currentDrivers int
	if s.driverSvc != nil {
		currentDrivers, _ = s.driverSvc.GetDriverCountInCell(ctx, h3Index)
	}

	// Neighbor demand
	cell := geo.StringToCell(h3Index)
	neighbors := geo.GetNeighborCells(cell)
	neighborIndexes := make([]string, len(neighbors))
	for i, n := range neighbors {
		neighborIndexes[i] = n.String()
	}
	neighborDemandAvg, _ := s.repo.GetNeighborDemandAverage(ctx, neighborIndexes, 30)

	// Weather features
	var weatherCode int
	var temperature, precipProb float64
	if s.weatherSvc != nil && s.config.WeatherEnabled {
		latitude, longitude := geo.CellToLatLng(cell)
		weather, err := s.weatherSvc.GetCurrentWeather(ctx, latitude, longitude)
		if err == nil && weather != nil {
			weatherCode = weather.ConditionCode
			temperature = weather.Temperature
			precipProb = weather.PrecipitationProb
		}
	}

	// Event features
	var eventNearby bool
	var eventScale int
	if s.config.EventsEnabled {
		latitude, longitude := geo.CellToLatLng(cell)
		events, err := s.repo.GetEventsNearLocation(ctx, latitude, longitude, 5.0, 2*time.Hour)
		if err == nil && len(events) > 0 {
			eventNearby = true
			for _, e := range events {
				if e.ExpectedAttendees > eventScale {
					eventScale = e.ExpectedAttendees
				}
			}
		}
	}

	// Lagging demand (same time last week)
	laggingTime1hr := targetTime.AddDate(0, 0, -7).Add(-1 * time.Hour)
	laggingTime2hr := targetTime.AddDate(0, 0, -7).Add(-2 * time.Hour)
	laggingDemand1hr, _ := s.repo.GetRecentDemand(ctx, h3Index, int(time.Since(laggingTime1hr).Minutes()))
	laggingDemand2hr, _ := s.repo.GetRecentDemand(ctx, h3Index, int(time.Since(laggingTime2hr).Minutes()))

	return &ModelFeatures{
		H3Index:           h3Index,
		TargetTime:        targetTime,
		Hour:              hour,
		DayOfWeek:         dayOfWeek,
		IsWeekend:         isWeekend,
		IsHoliday:         s.isHoliday(targetTime),
		WeekOfYear:        getWeekOfYear(targetTime),
		MonthOfYear:       int(targetTime.Month()),
		HistAvgRides:      histAvg,
		HistStdRides:      histStd,
		RecentRides15min:  recentRides15min,
		RecentRides1hr:    recentRides1hr,
		RecentTrend:       recentTrend,
		CurrentDrivers:    currentDrivers,
		NeighborDemandAvg: neighborDemandAvg,
		WeatherCode:       weatherCode,
		Temperature:       temperature,
		PrecipitationProb: precipProb,
		EventNearby:       eventNearby,
		EventScale:        eventScale,
		LaggingDemand1hr:  float64(laggingDemand1hr),
		LaggingDemand2hr:  float64(laggingDemand2hr),
	}, nil
}

// predict runs the ML model on features
func (s *Service) predict(features *ModelFeatures) *ModelPrediction {
	// Weighted ensemble model combining multiple factors
	// This is a simplified model; production would use a trained ML model

	w := s.modelWeights
	var predictedRides float64

	// Historical pattern contribution
	historicalContrib := features.HistAvgRides
	if features.IsHoliday {
		historicalContrib *= 0.7 // Lower demand on holidays
	}
	predictedRides += w.HistoricalPattern * historicalContrib

	// Recent trend contribution
	trendAdjustment := 1.0 + (features.RecentTrend * 0.1)
	if trendAdjustment < 0.5 {
		trendAdjustment = 0.5
	}
	if trendAdjustment > 2.0 {
		trendAdjustment = 2.0
	}
	recentContrib := float64(features.RecentRides1hr) * trendAdjustment / 4 // Extrapolate 15-min from 1-hour
	predictedRides += w.RecentTrend * recentContrib

	// Time of day contribution
	timeMultiplier := s.getTimeMultiplier(features.Hour, features.IsWeekend)
	timeContrib := features.HistAvgRides * timeMultiplier
	predictedRides += w.TimeOfDay * timeContrib

	// Day of week contribution
	dayMultiplier := s.getDayMultiplier(features.DayOfWeek)
	dayContrib := features.HistAvgRides * dayMultiplier
	predictedRides += w.DayOfWeek * dayContrib

	// Weather contribution
	weatherMultiplier := s.getWeatherMultiplier(features.WeatherCode, features.PrecipitationProb)
	weatherContrib := features.HistAvgRides * weatherMultiplier
	predictedRides += w.Weather * weatherContrib

	// Event contribution
	if features.EventNearby {
		eventMultiplier := s.getEventMultiplier(features.EventScale)
		eventContrib := features.HistAvgRides * eventMultiplier
		predictedRides += w.Events * eventContrib
	}

	// Seasonal contribution
	seasonalMultiplier := s.getSeasonalMultiplier(features.MonthOfYear)
	seasonalContrib := features.HistAvgRides * seasonalMultiplier
	predictedRides += w.Seasonal * seasonalContrib

	// Ensure non-negative
	if predictedRides < 0 {
		predictedRides = 0
	}

	// Calculate confidence based on data availability
	confidence := s.calculateConfidence(features)

	// Calculate bounds (95% CI)
	stdErr := features.HistStdRides * (1 - confidence)
	lowerBound := math.Max(0, predictedRides-1.96*stdErr)
	upperBound := predictedRides + 1.96*stdErr

	return &ModelPrediction{
		PredictedRides: predictedRides,
		LowerBound:     lowerBound,
		UpperBound:     upperBound,
		Confidence:     confidence,
	}
}

// ========================================
// HOTSPOT DETECTION
// ========================================

// GetTopHotspots returns the top demand hotspots for driver positioning
func (s *Service) GetTopHotspots(ctx context.Context, timeframe PredictionTimeframe, limit int) ([]HotspotZone, error) {
	predictions, err := s.repo.GetTopHotspots(ctx, timeframe, limit)
	if err != nil {
		return nil, err
	}

	var hotspots []HotspotZone
	for _, pred := range predictions {
		cell := geo.StringToCell(pred.H3Index)
		latitude, longitude := geo.CellToLatLng(cell)

		var currentDrivers int
		if s.driverSvc != nil {
			currentDrivers, _ = s.driverSvc.GetDriverCountInCell(ctx, pred.H3Index)
		}

		hotspot := HotspotZone{
			H3Index:         pred.H3Index,
			CenterLatitude: latitude,
			CenterLongitude: longitude,
			DemandLevel:     pred.DemandLevel,
			PredictedRides:  pred.PredictedRides,
			CurrentDrivers:  currentDrivers,
			NeededDrivers:   pred.RecommendedDrivers,
			Gap:             pred.RecommendedDrivers - currentDrivers,
			ExpectedSurge:   pred.ExpectedSurge,
			HotspotScore:    pred.HotspotScore,
			ValidUntil:      pred.PredictionTime,
		}
		hotspots = append(hotspots, hotspot)
	}

	return hotspots, nil
}

// GetDemandHeatmap returns a heatmap of demand for a geographic area
func (s *Service) GetDemandHeatmap(ctx context.Context, req *GetHeatmapRequest) (*DemandHeatmap, error) {
	predictions, err := s.repo.GetPredictionsInBoundingBox(ctx, req.Timeframe, req.MinDemandLevel)
	if err != nil {
		return nil, err
	}

	// Filter by bounding box
	var zones []HotspotZone
	for _, pred := range predictions {
		cell := geo.StringToCell(pred.H3Index)
		latitude, longitude := geo.CellToLatLng(cell)

		if latitude < req.MinLatitude || latitude > req.MaxLatitude ||
			longitude < req.MinLongitude || longitude > req.MaxLongitude {
			continue
		}

		var currentDrivers int
		if s.driverSvc != nil {
			currentDrivers, _ = s.driverSvc.GetDriverCountInCell(ctx, pred.H3Index)
		}

		zone := HotspotZone{
			H3Index:         pred.H3Index,
			CenterLatitude: latitude,
			CenterLongitude: longitude,
			DemandLevel:     pred.DemandLevel,
			PredictedRides:  pred.PredictedRides,
			CurrentDrivers:  currentDrivers,
			NeededDrivers:   pred.RecommendedDrivers,
			Gap:             pred.RecommendedDrivers - currentDrivers,
			ExpectedSurge:   pred.ExpectedSurge,
			HotspotScore:    pred.HotspotScore,
			ValidUntil:      pred.PredictionTime,
		}
		zones = append(zones, zone)
	}

	return &DemandHeatmap{
		GeneratedAt: time.Now(),
		Timeframe:   req.Timeframe,
		Zones:       zones,
		BoundingBox: BoundingBox{
			MinLatitude:  req.MinLatitude,
			MaxLatitude:  req.MaxLatitude,
			MinLongitude: req.MinLongitude,
			MaxLongitude: req.MaxLongitude,
		},
	}, nil
}

// ========================================
// DRIVER REPOSITIONING
// ========================================

// GetRepositionRecommendations suggests where a driver should move for better earnings
func (s *Service) GetRepositionRecommendations(ctx context.Context, req *GetRepositionRecommendationsRequest) (*GetRepositionRecommendationsResponse, error) {
	maxDistance := req.MaxDistanceKm
	if maxDistance == 0 {
		maxDistance = 10.0
	}
	limit := req.Limit
	if limit == 0 {
		limit = 3
	}

	currentH3Index := geo.LatLngToCell(req.Latitude, req.Longitude, s.config.H3Resolution).String()

	// Get current zone prediction
	currentPred, _ := s.repo.GetLatestPrediction(ctx, currentH3Index, Timeframe30Min)
	var currentZone HotspotZone
	if currentPred != nil {
		var currentDrivers int
		if s.driverSvc != nil {
			currentDrivers, _ = s.driverSvc.GetDriverCountInCell(ctx, currentH3Index)
		}
		currentZone = HotspotZone{
			H3Index:         currentH3Index,
			CenterLatitude:  req.Latitude,
			CenterLongitude: req.Longitude,
			DemandLevel:     currentPred.DemandLevel,
			PredictedRides:  currentPred.PredictedRides,
			CurrentDrivers:  currentDrivers,
			NeededDrivers:   currentPred.RecommendedDrivers,
			Gap:             currentPred.RecommendedDrivers - currentDrivers,
			ExpectedSurge:   currentPred.ExpectedSurge,
			HotspotScore:    currentPred.HotspotScore,
			ValidUntil:      currentPred.PredictionTime,
		}
	}

	// Get top hotspots
	hotspots, err := s.GetTopHotspots(ctx, Timeframe30Min, 20)
	if err != nil {
		return nil, err
	}

	// Filter and rank recommendations
	var recommendations []DriverRepositionRecommendation
	for _, hotspot := range hotspots {
		if hotspot.H3Index == currentH3Index {
			continue
		}

		// Calculate distance
		distance := haversineDistance(req.Latitude, req.Longitude, hotspot.CenterLatitude, hotspot.CenterLongitude)
		if distance > maxDistance {
			continue
		}

		// Calculate expected earnings improvement
		expectedRides := hotspot.PredictedRides / float64(hotspot.NeededDrivers)
		baseEarnings := expectedRides * 15.0 // Assume $15 per ride base
		surgeEarnings := baseEarnings * hotspot.ExpectedSurge

		recommendation := DriverRepositionRecommendation{
			DriverID:           req.DriverID,
			CurrentH3Index:     currentH3Index,
			TargetH3Index:      hotspot.H3Index,
			TargetLatitude:     hotspot.CenterLatitude,
			TargetLongitude:    hotspot.CenterLongitude,
			DistanceKm:         distance,
			Priority:           s.calculatePriority(hotspot, distance),
			ExpectedRides:      expectedRides,
			ExpectedEarnings:   surgeEarnings,
			ExpectedSurge:      hotspot.ExpectedSurge,
			RecommendedArrival: time.Now().Add(time.Duration(distance/30*60) * time.Minute),
			Reason:             s.getRepositionReason(hotspot),
		}
		recommendations = append(recommendations, recommendation)

		if len(recommendations) >= limit {
			break
		}
	}

	// Calculate total estimated earnings if following top recommendation
	var estimatedEarnings float64
	if len(recommendations) > 0 {
		estimatedEarnings = recommendations[0].ExpectedEarnings
	}

	return &GetRepositionRecommendationsResponse{
		CurrentZone:       currentZone,
		Recommendations:   recommendations,
		EstimatedEarnings: estimatedEarnings,
	}, nil
}

// ========================================
// EVENT MANAGEMENT
// ========================================

// CreateEvent creates a special event that affects demand
func (s *Service) CreateEvent(ctx context.Context, req *CreateEventRequest) (*SpecialEvent, error) {
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return nil, common.NewBadRequestError("invalid start_time format", err)
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		return nil, common.NewBadRequestError("invalid end_time format", err)
	}

	h3Index := geo.LatLngToCell(req.Latitude, req.Longitude, s.config.H3Resolution).String()

	impactRadius := req.ImpactRadius
	if impactRadius == 0 {
		impactRadius = 5.0
	}

	// Calculate demand multiplier based on event size
	multiplier := s.calculateEventDemandMultiplier(req.ExpectedAttendees)

	event := &SpecialEvent{
		ID:                uuid.New(),
		Name:              req.Name,
		EventType:         req.EventType,
		Latitude:          req.Latitude,
		Longitude:         req.Longitude,
		H3Index:           h3Index,
		StartTime:         startTime,
		EndTime:           endTime,
		ExpectedAttendees: req.ExpectedAttendees,
		ImpactRadius:      impactRadius,
		DemandMultiplier:  multiplier,
		IsRecurring:       req.IsRecurring,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := s.repo.CreateEvent(ctx, event); err != nil {
		return nil, common.NewInternalServerError("failed to create event")
	}

	logger.Info("Special event created",
		zap.String("event_id", event.ID.String()),
		zap.String("name", event.Name),
		zap.Int("expected_attendees", event.ExpectedAttendees),
	)

	return event, nil
}

// GetUpcomingEvents returns events in a time range
func (s *Service) GetUpcomingEvents(ctx context.Context, hours int) ([]*SpecialEvent, error) {
	startTime := time.Now()
	endTime := startTime.Add(time.Duration(hours) * time.Hour)
	return s.repo.GetUpcomingEvents(ctx, startTime, endTime)
}

// ========================================
// DATA COLLECTION
// ========================================

// RecordDemandSnapshot records current demand for ML training
func (s *Service) RecordDemandSnapshot(ctx context.Context, h3Index string, rideRequests, completedRides, availableDrivers int, avgWaitTime, surgeMultiplier float64) error {
	now := time.Now()

	record := &HistoricalDemandRecord{
		ID:               uuid.New(),
		H3Index:          h3Index,
		Timestamp:        now.Truncate(15 * time.Minute), // 15-min buckets
		Hour:             now.Hour(),
		DayOfWeek:        int(now.Weekday()),
		IsHoliday:        s.isHoliday(now),
		RideRequests:     rideRequests,
		CompletedRides:   completedRides,
		AvailableDrivers: availableDrivers,
		AvgWaitTimeMin:   avgWaitTime,
		SurgeMultiplier:  surgeMultiplier,
		WeatherCondition: "unknown",
		CreatedAt:        now,
	}

	// Add weather if available
	if s.weatherSvc != nil && s.config.WeatherEnabled {
		cell := geo.StringToCell(h3Index)
		latitude, longitude := geo.CellToLatLng(cell)
		weather, err := s.weatherSvc.GetCurrentWeather(ctx, latitude, longitude)
		if err == nil && weather != nil {
			record.WeatherCondition = weather.Condition
			record.Temperature = &weather.Temperature
			record.PrecipitationMM = &weather.PrecipitationMM
		}
	}

	return s.repo.RecordDemand(ctx, record)
}

// GetModelAccuracy returns accuracy metrics for the model
func (s *Service) GetModelAccuracy(ctx context.Context, timeframe PredictionTimeframe, daysBack int) (*ForecastAccuracyMetrics, error) {
	return s.repo.GetAccuracyMetrics(ctx, timeframe, daysBack)
}

// ========================================
// HELPER FUNCTIONS
// ========================================

func (s *Service) getTargetTime(timeframe PredictionTimeframe) time.Time {
	now := time.Now()
	switch timeframe {
	case Timeframe15Min:
		return now.Add(15 * time.Minute).Truncate(15 * time.Minute)
	case Timeframe30Min:
		return now.Add(30 * time.Minute).Truncate(15 * time.Minute)
	case Timeframe1Hour:
		return now.Add(1 * time.Hour).Truncate(15 * time.Minute)
	case Timeframe2Hour:
		return now.Add(2 * time.Hour).Truncate(15 * time.Minute)
	case Timeframe4Hour:
		return now.Add(4 * time.Hour).Truncate(15 * time.Minute)
	default:
		return now.Add(30 * time.Minute).Truncate(15 * time.Minute)
	}
}

func (s *Service) classifyDemandLevel(predictedRides, historicalAvg float64) DemandLevel {
	if historicalAvg == 0 {
		historicalAvg = 10 // Default baseline
	}

	ratio := predictedRides / historicalAvg
	switch {
	case ratio <= 0.3:
		return DemandVeryLow
	case ratio <= 0.7:
		return DemandLow
	case ratio <= 1.3:
		return DemandNormal
	case ratio <= 1.8:
		return DemandHigh
	case ratio <= 2.5:
		return DemandVeryHigh
	default:
		return DemandExtreme
	}
}

func (s *Service) calculateRecommendedDrivers(predictedRides float64) int {
	// Assume 2 rides per driver per hour, with 20% buffer
	drivers := int(math.Ceil(predictedRides / 2 * 1.2))
	if drivers < 1 {
		drivers = 1
	}
	return drivers
}

func (s *Service) calculateExpectedSurge(predictedRides float64, currentDrivers int) float64 {
	if currentDrivers == 0 {
		currentDrivers = 1
	}

	ratio := predictedRides / float64(currentDrivers)
	if ratio <= 1.0 {
		return 1.0
	}
	if ratio <= 2.0 {
		return 1.0 + (ratio-1.0)*0.5
	}
	if ratio <= 3.0 {
		return 1.5 + (ratio-2.0)*0.5
	}

	// Cap at 3.0x
	return math.Min(3.0, 2.0+(ratio-3.0)*0.25)
}

func (s *Service) calculateHotspotScore(predictedRides float64, features *ModelFeatures) float64 {
	// Score from 0-100 based on:
	// - Predicted ride volume (40%)
	// - Driver shortage (30%)
	// - Surge potential (20%)
	// - Trend direction (10%)

	// Normalize predicted rides (assume 50 is high)
	volumeScore := math.Min(100, predictedRides*2)

	// Driver shortage score
	shortage := float64(s.calculateRecommendedDrivers(predictedRides) - features.CurrentDrivers)
	shortageScore := math.Min(100, math.Max(0, shortage*20))

	// Surge potential
	surge := s.calculateExpectedSurge(predictedRides, features.CurrentDrivers)
	surgeScore := (surge - 1.0) * 50

	// Trend score
	trendScore := 50 + features.RecentTrend*10
	if trendScore < 0 {
		trendScore = 0
	}
	if trendScore > 100 {
		trendScore = 100
	}

	score := volumeScore*0.4 + shortageScore*0.3 + surgeScore*0.2 + trendScore*0.1
	return math.Min(100, math.Max(0, score))
}

func (s *Service) calculateRepositionPriority(predictedRides float64, features *ModelFeatures) int {
	score := s.calculateHotspotScore(predictedRides, features)
	priority := int(10 - score/10)
	if priority < 1 {
		priority = 1
	}
	if priority > 10 {
		priority = 10
	}
	return priority
}

func (s *Service) calculateContributions(features *ModelFeatures, prediction *ModelPrediction) FeatureContributions {
	w := s.modelWeights
	return FeatureContributions{
		HistoricalPattern: w.HistoricalPattern,
		RecentTrend:       w.RecentTrend,
		TimeOfDay:         w.TimeOfDay,
		DayOfWeek:         w.DayOfWeek,
		Weather:           w.Weather,
		SpecialEvents:     w.Events,
		SeasonalTrend:     w.Seasonal,
	}
}

func (s *Service) calculateConfidence(features *ModelFeatures) float64 {
	confidence := 0.5 // Base confidence

	// More historical data = higher confidence
	if features.HistAvgRides > 0 {
		confidence += 0.2
	}

	// Recent data available
	if features.RecentRides1hr > 0 {
		confidence += 0.1
	}

	// Weather data available
	if features.WeatherCode > 0 {
		confidence += 0.1
	}

	// Low variance in historical data
	if features.HistStdRides < features.HistAvgRides*0.5 {
		confidence += 0.1
	}

	return math.Min(0.95, confidence)
}

func (s *Service) getTimeMultiplier(hour int, isWeekend bool) float64 {
	// Peak hours adjustment
	if hour >= 7 && hour <= 9 && !isWeekend {
		return 1.5 // Morning commute
	}
	if hour >= 17 && hour <= 19 && !isWeekend {
		return 1.6 // Evening commute
	}
	if hour >= 22 || hour <= 2 {
		if isWeekend {
			return 1.4 // Weekend night
		}
		return 1.2 // Weeknight
	}
	if hour >= 11 && hour <= 14 {
		return 1.1 // Lunch
	}
	return 1.0
}

func (s *Service) getDayMultiplier(dayOfWeek int) float64 {
	switch dayOfWeek {
	case 0: // Sunday
		return 0.8
	case 5: // Friday
		return 1.2
	case 6: // Saturday
		return 1.1
	default:
		return 1.0
	}
}

func (s *Service) getWeatherMultiplier(weatherCode int, precipProb float64) float64 {
	// Weather codes (simplified):
	// 0-3: Clear/cloudy
	// 4-5: Rain
	// 6-7: Snow

	if weatherCode >= 4 && weatherCode <= 5 {
		return 1.3 // Rain increases demand
	}
	if weatherCode >= 6 {
		return 1.5 // Snow significantly increases demand
	}
	if precipProb > 0.7 {
		return 1.2 // High chance of rain
	}
	return 1.0
}

func (s *Service) getEventMultiplier(attendees int) float64 {
	// Scale based on expected attendees
	if attendees > 50000 {
		return 3.0 // Major stadium event
	}
	if attendees > 20000 {
		return 2.5
	}
	if attendees > 10000 {
		return 2.0
	}
	if attendees > 5000 {
		return 1.5
	}
	if attendees > 1000 {
		return 1.3
	}
	return 1.1
}

func (s *Service) getSeasonalMultiplier(month int) float64 {
	// Simple seasonal adjustment (northern hemisphere)
	switch month {
	case 12, 1, 2: // Winter
		return 1.1
	case 6, 7, 8: // Summer
		return 0.95
	default:
		return 1.0
	}
}

func (s *Service) isHoliday(t time.Time) bool {
	// Simplified holiday detection
	// In production, would check against a holiday calendar
	month := t.Month()
	day := t.Day()

	// US major holidays (simplified)
	if month == 12 && day == 25 { // Christmas
		return true
	}
	if month == 1 && day == 1 { // New Year
		return true
	}
	if month == 7 && day == 4 { // July 4th
		return true
	}
	if month == 11 && day >= 22 && day <= 28 && t.Weekday() == time.Thursday { // Thanksgiving
		return true
	}

	return false
}

func (s *Service) calculateEventDemandMultiplier(attendees int) float64 {
	return s.getEventMultiplier(attendees)
}

func (s *Service) calculatePriority(hotspot HotspotZone, distance float64) int {
	// Lower number = higher priority
	// Based on: hotspot score, distance, driver gap

	score := hotspot.HotspotScore
	distancePenalty := distance * 5 // 5 points per km

	adjustedScore := score - distancePenalty
	if adjustedScore < 0 {
		adjustedScore = 0
	}

	priority := int(10 - adjustedScore/10)
	if priority < 1 {
		priority = 1
	}
	if priority > 10 {
		priority = 10
	}

	return priority
}

func (s *Service) getRepositionReason(hotspot HotspotZone) string {
	if hotspot.ExpectedSurge >= 2.0 {
		return "High surge expected"
	}
	if hotspot.Gap > 5 {
		return "Significant driver shortage"
	}
	if hotspot.DemandLevel == DemandExtreme || hotspot.DemandLevel == DemandVeryHigh {
		return "Very high demand predicted"
	}
	return "Better earnings opportunity"
}

func (s *Service) getCellsInBoundingBox(minLatitude, minLongitude, maxLatitude, maxLongitude float64) []h3.Cell {
	// Sample points across the bounding box and collect unique cells
	cellMap := make(map[h3.Cell]bool)

	// Calculate step size based on resolution
	// H3 resolution 7 has ~1.2km edge length
	step := 0.01 // ~1km in latitude

	for latitude := minLatitude; latitude <= maxLatitude; latitude += step {
		for longitude := minLongitude; longitude <= maxLongitude; longitude += step {
			cell := geo.LatLngToCell(latitude, longitude, s.config.H3Resolution)
			cellMap[cell] = true
		}
	}

	cells := make([]h3.Cell, 0, len(cellMap))
	for cell := range cellMap {
		cells = append(cells, cell)
	}

	return cells
}

func getWeekOfYear(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}

func haversineDistance(latitude1, longitude1, latitude2, longitude2 float64) float64 {
	const R = 6371 // Earth's radius in km

	latitude1Rad := latitude1 * math.Pi / 180
	latitude2Rad := latitude2 * math.Pi / 180
	deltaLatitude := (latitude2 - latitude1) * math.Pi / 180
	deltaLongitude := (longitude2 - longitude1) * math.Pi / 180

	a := math.Sin(deltaLatitude/2)*math.Sin(deltaLatitude/2) +
		math.Cos(latitude1Rad)*math.Cos(latitude2Rad)*
			math.Sin(deltaLongitude/2)*math.Sin(deltaLongitude/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}
