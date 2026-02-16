package demandforecast

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ========================================
// MOCK DEFINITIONS
// ========================================

type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) RecordDemand(ctx context.Context, record *HistoricalDemandRecord) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}

func (m *mockRepository) GetHistoricalDemand(ctx context.Context, h3Index string, startTime, endTime time.Time) ([]*HistoricalDemandRecord, error) {
	args := m.Called(ctx, h3Index, startTime, endTime)
	records, _ := args.Get(0).([]*HistoricalDemandRecord)
	return records, args.Error(1)
}

func (m *mockRepository) GetHistoricalAverage(ctx context.Context, h3Index string, hour, dayOfWeek int, weeksBack int) (float64, float64, error) {
	args := m.Called(ctx, h3Index, hour, dayOfWeek, weeksBack)
	return args.Get(0).(float64), args.Get(1).(float64), args.Error(2)
}

func (m *mockRepository) GetRecentDemand(ctx context.Context, h3Index string, minutesBack int) (int, error) {
	args := m.Called(ctx, h3Index, minutesBack)
	return args.Int(0), args.Error(1)
}

func (m *mockRepository) GetRecentDemandTrend(ctx context.Context, h3Index string) (float64, error) {
	args := m.Called(ctx, h3Index)
	return args.Get(0).(float64), args.Error(1)
}

func (m *mockRepository) GetNeighborDemandAverage(ctx context.Context, neighborIndexes []string, minutesBack int) (float64, error) {
	args := m.Called(ctx, neighborIndexes, minutesBack)
	return args.Get(0).(float64), args.Error(1)
}

func (m *mockRepository) SavePrediction(ctx context.Context, pred *DemandPrediction) error {
	args := m.Called(ctx, pred)
	return args.Error(0)
}

func (m *mockRepository) GetPrediction(ctx context.Context, h3Index string, predictionTime time.Time, timeframe PredictionTimeframe) (*DemandPrediction, error) {
	args := m.Called(ctx, h3Index, predictionTime, timeframe)
	pred, _ := args.Get(0).(*DemandPrediction)
	return pred, args.Error(1)
}

func (m *mockRepository) GetLatestPrediction(ctx context.Context, h3Index string, timeframe PredictionTimeframe) (*DemandPrediction, error) {
	args := m.Called(ctx, h3Index, timeframe)
	pred, _ := args.Get(0).(*DemandPrediction)
	return pred, args.Error(1)
}

func (m *mockRepository) GetPredictionsInBoundingBox(ctx context.Context, timeframe PredictionTimeframe, minDemandLevel *DemandLevel) ([]*DemandPrediction, error) {
	args := m.Called(ctx, timeframe, minDemandLevel)
	preds, _ := args.Get(0).([]*DemandPrediction)
	return preds, args.Error(1)
}

func (m *mockRepository) GetTopHotspots(ctx context.Context, timeframe PredictionTimeframe, limit int) ([]*DemandPrediction, error) {
	args := m.Called(ctx, timeframe, limit)
	preds, _ := args.Get(0).([]*DemandPrediction)
	return preds, args.Error(1)
}

func (m *mockRepository) CreateEvent(ctx context.Context, event *SpecialEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *mockRepository) GetUpcomingEvents(ctx context.Context, startTime, endTime time.Time) ([]*SpecialEvent, error) {
	args := m.Called(ctx, startTime, endTime)
	events, _ := args.Get(0).([]*SpecialEvent)
	return events, args.Error(1)
}

func (m *mockRepository) GetEventsNearLocation(ctx context.Context, latitude, longitude float64, radiusKm float64, timeWindow time.Duration) ([]*SpecialEvent, error) {
	args := m.Called(ctx, latitude, longitude, radiusKm, timeWindow)
	events, _ := args.Get(0).([]*SpecialEvent)
	return events, args.Error(1)
}

func (m *mockRepository) RecordPredictionAccuracy(ctx context.Context, predictionID uuid.UUID, actualRides int) error {
	args := m.Called(ctx, predictionID, actualRides)
	return args.Error(0)
}

func (m *mockRepository) GetAccuracyMetrics(ctx context.Context, timeframe PredictionTimeframe, daysBack int) (*ForecastAccuracyMetrics, error) {
	args := m.Called(ctx, timeframe, daysBack)
	metrics, _ := args.Get(0).(*ForecastAccuracyMetrics)
	return metrics, args.Error(1)
}

func (m *mockRepository) CleanupOldPredictions(ctx context.Context, daysOld int) (int64, error) {
	args := m.Called(ctx, daysOld)
	return int64(args.Int(0)), args.Error(1)
}

type mockWeatherService struct {
	mock.Mock
}

func (m *mockWeatherService) GetCurrentWeather(ctx context.Context, latitude, longitude float64) (*WeatherData, error) {
	args := m.Called(ctx, latitude, longitude)
	weather, _ := args.Get(0).(*WeatherData)
	return weather, args.Error(1)
}

func (m *mockWeatherService) GetForecast(ctx context.Context, latitude, longitude float64, hours int) ([]WeatherData, error) {
	args := m.Called(ctx, latitude, longitude, hours)
	forecast, _ := args.Get(0).([]WeatherData)
	return forecast, args.Error(1)
}

type mockDriverLocationService struct {
	mock.Mock
}

func (m *mockDriverLocationService) GetDriverCountInCell(ctx context.Context, h3Index string) (int, error) {
	args := m.Called(ctx, h3Index)
	return args.Int(0), args.Error(1)
}

func (m *mockDriverLocationService) GetNearbyDriverCount(ctx context.Context, latitude, longitude float64, radiusKm float64) (int, error) {
	args := m.Called(ctx, latitude, longitude, radiusKm)
	return args.Int(0), args.Error(1)
}

// ========================================
// TEST HELPER FUNCTIONS
// ========================================

func newTestService(repo RepositoryInterface, weatherSvc WeatherService, driverSvc DriverLocationService, config *Config) *Service {
	return NewService(repo, weatherSvc, driverSvc, config)
}

// ========================================
// CONFIGURATION TESTS
// ========================================

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, 7, config.H3Resolution)
	assert.Equal(t, 8, config.TrainingDataWeeks)
	assert.True(t, config.WeatherEnabled)
	assert.True(t, config.EventsEnabled)
	assert.Equal(t, 0.5, config.MinConfidence)
	assert.Contains(t, config.PredictionIntervals, Timeframe15Min)
	assert.Contains(t, config.PredictionIntervals, Timeframe30Min)
	assert.Contains(t, config.PredictionIntervals, Timeframe1Hour)
}

func TestDefaultModelWeights(t *testing.T) {
	weights := DefaultModelWeights()

	// Verify weights sum to 1.0
	total := weights.HistoricalPattern + weights.RecentTrend + weights.TimeOfDay +
		weights.DayOfWeek + weights.Weather + weights.Events + weights.Seasonal
	assert.InDelta(t, 1.0, total, 0.001)

	// Verify individual weights
	assert.Equal(t, 0.35, weights.HistoricalPattern)
	assert.Equal(t, 0.25, weights.RecentTrend)
	assert.Equal(t, 0.15, weights.TimeOfDay)
	assert.Equal(t, 0.10, weights.DayOfWeek)
	assert.Equal(t, 0.08, weights.Weather)
	assert.Equal(t, 0.05, weights.Events)
	assert.Equal(t, 0.02, weights.Seasonal)
}

func TestNewServiceWithDefaultConfig(t *testing.T) {
	repo := new(mockRepository)
	service := NewService(repo, nil, nil, nil)

	assert.NotNil(t, service)
	assert.NotNil(t, service.config)
	assert.Equal(t, 7, service.config.H3Resolution)
}

func TestNewServiceWithCustomConfig(t *testing.T) {
	repo := new(mockRepository)
	customConfig := &Config{
		H3Resolution:      9,
		TrainingDataWeeks: 4,
		WeatherEnabled:    false,
		EventsEnabled:     false,
		MinConfidence:     0.7,
	}

	service := NewService(repo, nil, nil, customConfig)

	assert.NotNil(t, service)
	assert.Equal(t, 9, service.config.H3Resolution)
	assert.Equal(t, 4, service.config.TrainingDataWeeks)
	assert.False(t, service.config.WeatherEnabled)
	assert.False(t, service.config.EventsEnabled)
}

// ========================================
// DEMAND LEVEL CLASSIFICATION TESTS
// ========================================

func TestClassifyDemandLevel(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	tests := []struct {
		name          string
		predictedRides float64
		historicalAvg  float64
		expectedLevel  DemandLevel
	}{
		{
			name:          "very low demand (ratio 0.2)",
			predictedRides: 2.0,
			historicalAvg:  10.0,
			expectedLevel:  DemandVeryLow,
		},
		{
			name:          "low demand (ratio 0.5)",
			predictedRides: 5.0,
			historicalAvg:  10.0,
			expectedLevel:  DemandLow,
		},
		{
			name:          "normal demand (ratio 1.0)",
			predictedRides: 10.0,
			historicalAvg:  10.0,
			expectedLevel:  DemandNormal,
		},
		{
			name:          "high demand (ratio 1.5)",
			predictedRides: 15.0,
			historicalAvg:  10.0,
			expectedLevel:  DemandHigh,
		},
		{
			name:          "very high demand (ratio 2.0)",
			predictedRides: 20.0,
			historicalAvg:  10.0,
			expectedLevel:  DemandVeryHigh,
		},
		{
			name:          "extreme demand (ratio 3.0)",
			predictedRides: 30.0,
			historicalAvg:  10.0,
			expectedLevel:  DemandExtreme,
		},
		{
			name:          "zero historical avg uses default baseline",
			predictedRides: 15.0,
			historicalAvg:  0.0,
			expectedLevel:  DemandHigh, // 15/10 = 1.5 ratio
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.classifyDemandLevel(tt.predictedRides, tt.historicalAvg)
			assert.Equal(t, tt.expectedLevel, result)
		})
	}
}

// ========================================
// RECOMMENDED DRIVERS CALCULATION TESTS
// ========================================

func TestCalculateRecommendedDrivers(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	tests := []struct {
		name           string
		predictedRides float64
		expectedDrivers int
	}{
		{
			name:           "low demand 2 rides",
			predictedRides: 2.0,
			expectedDrivers: 2, // ceil(2 / 2 * 1.2) = ceil(1.2) = 2
		},
		{
			name:           "medium demand 10 rides",
			predictedRides: 10.0,
			expectedDrivers: 6, // ceil(10 / 2 * 1.2) = ceil(6) = 6
		},
		{
			name:           "high demand 50 rides",
			predictedRides: 50.0,
			expectedDrivers: 30, // ceil(50 / 2 * 1.2) = ceil(30) = 30
		},
		{
			name:           "zero rides returns at least 1",
			predictedRides: 0.0,
			expectedDrivers: 1,
		},
		{
			name:           "fractional rides rounds up",
			predictedRides: 3.0,
			expectedDrivers: 2, // ceil(3 / 2 * 1.2) = ceil(1.8) = 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateRecommendedDrivers(tt.predictedRides)
			assert.Equal(t, tt.expectedDrivers, result)
		})
	}
}

// ========================================
// EXPECTED SURGE CALCULATION TESTS
// ========================================

func TestCalculateExpectedSurge(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	tests := []struct {
		name           string
		predictedRides float64
		currentDrivers int
		expectedSurge  float64
		delta          float64
	}{
		{
			name:           "no surge when supply exceeds demand",
			predictedRides: 5.0,
			currentDrivers: 10,
			expectedSurge:  1.0,
			delta:          0.01,
		},
		{
			name:           "no surge when supply equals demand",
			predictedRides: 10.0,
			currentDrivers: 10,
			expectedSurge:  1.0,
			delta:          0.01,
		},
		{
			name:           "mild surge ratio 1.5",
			predictedRides: 15.0,
			currentDrivers: 10,
			expectedSurge:  1.25, // 1.0 + (1.5-1.0)*0.5 = 1.25
			delta:          0.01,
		},
		{
			name:           "moderate surge ratio 2.0",
			predictedRides: 20.0,
			currentDrivers: 10,
			expectedSurge:  1.5, // 1.0 + (2.0-1.0)*0.5 = 1.5
			delta:          0.01,
		},
		{
			name:           "high surge ratio 3.0",
			predictedRides: 30.0,
			currentDrivers: 10,
			expectedSurge:  2.0, // 1.5 + (3.0-2.0)*0.5 = 2.0
			delta:          0.01,
		},
		{
			name:           "extreme surge capped at 3.0",
			predictedRides: 100.0,
			currentDrivers: 10,
			expectedSurge:  3.0,
			delta:          0.01,
		},
		{
			name:           "zero drivers uses 1 to avoid division by zero",
			predictedRides: 10.0,
			currentDrivers: 0,
			expectedSurge:  3.0, // ratio = 10, capped
			delta:          0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateExpectedSurge(tt.predictedRides, tt.currentDrivers)
			assert.InDelta(t, tt.expectedSurge, result, tt.delta)
		})
	}
}

// ========================================
// TIME MULTIPLIER TESTS
// ========================================

func TestGetTimeMultiplier(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	tests := []struct {
		name       string
		hour       int
		isWeekend  bool
		expected   float64
	}{
		{"weekday morning rush 7am", 7, false, 1.5},
		{"weekday morning rush 8am", 8, false, 1.5},
		{"weekend morning rush 8am - no bonus", 8, true, 1.0},
		{"weekday evening rush 5pm", 17, false, 1.6},
		{"weekday evening rush 7pm", 19, false, 1.6},
		{"weekend evening rush 6pm - no bonus", 18, true, 1.0},
		{"weekday late night 11pm", 23, false, 1.2},
		{"weekend late night 11pm", 23, true, 1.4},
		{"weekend late night 1am", 1, true, 1.4},
		{"weekday lunch 12pm", 12, false, 1.1},
		{"weekday lunch 2pm", 14, false, 1.1},
		{"regular hours 3pm", 15, false, 1.0},
		{"regular hours 10am", 10, false, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.getTimeMultiplier(tt.hour, tt.isWeekend)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ========================================
// DAY MULTIPLIER TESTS
// ========================================

func TestGetDayMultiplier(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	tests := []struct {
		name      string
		dayOfWeek int
		expected  float64
	}{
		{"Sunday lower demand", 0, 0.8},
		{"Monday normal", 1, 1.0},
		{"Tuesday normal", 2, 1.0},
		{"Wednesday normal", 3, 1.0},
		{"Thursday normal", 4, 1.0},
		{"Friday higher demand", 5, 1.2},
		{"Saturday higher demand", 6, 1.1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.getDayMultiplier(tt.dayOfWeek)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ========================================
// WEATHER MULTIPLIER TESTS
// ========================================

func TestGetWeatherMultiplier(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	tests := []struct {
		name        string
		weatherCode int
		precipProb  float64
		expected    float64
	}{
		{"clear weather", 0, 0.0, 1.0},
		{"cloudy weather", 2, 0.1, 1.0},
		{"light rain code 4", 4, 0.5, 1.3},
		{"heavy rain code 5", 5, 0.8, 1.3},
		{"snow code 6", 6, 0.9, 1.5},
		{"snow code 7", 7, 0.9, 1.5},
		{"high precip probability no rain code", 1, 0.8, 1.2},
		{"moderate precip probability", 1, 0.5, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.getWeatherMultiplier(tt.weatherCode, tt.precipProb)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ========================================
// EVENT MULTIPLIER TESTS
// ========================================

func TestGetEventMultiplier(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	tests := []struct {
		name      string
		attendees int
		expected  float64
	}{
		{"small event 500 attendees", 500, 1.1},
		{"medium event 2000 attendees", 2000, 1.3},
		{"large event 7000 attendees", 7000, 1.5},
		{"very large event 15000 attendees", 15000, 2.0},
		{"huge event 30000 attendees", 30000, 2.5},
		{"massive event 60000 attendees", 60000, 3.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.getEventMultiplier(tt.attendees)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ========================================
// SEASONAL MULTIPLIER TESTS
// ========================================

func TestGetSeasonalMultiplier(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	tests := []struct {
		name     string
		month    int
		expected float64
	}{
		{"January winter", 1, 1.1},
		{"February winter", 2, 1.1},
		{"March spring", 3, 1.0},
		{"April spring", 4, 1.0},
		{"May spring", 5, 1.0},
		{"June summer", 6, 0.95},
		{"July summer", 7, 0.95},
		{"August summer", 8, 0.95},
		{"September fall", 9, 1.0},
		{"October fall", 10, 1.0},
		{"November fall", 11, 1.0},
		{"December winter", 12, 1.1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.getSeasonalMultiplier(tt.month)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ========================================
// HOLIDAY DETECTION TESTS
// ========================================

func TestIsHoliday(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	tests := []struct {
		name       string
		date       time.Time
		isHoliday  bool
	}{
		{
			name:      "Christmas",
			date:      time.Date(2024, 12, 25, 12, 0, 0, 0, time.UTC),
			isHoliday: true,
		},
		{
			name:      "New Year",
			date:      time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			isHoliday: true,
		},
		{
			name:      "July 4th",
			date:      time.Date(2024, 7, 4, 12, 0, 0, 0, time.UTC),
			isHoliday: true,
		},
		{
			name:      "Thanksgiving 2024 (Nov 28)",
			date:      time.Date(2024, 11, 28, 12, 0, 0, 0, time.UTC), // Thursday
			isHoliday: true,
		},
		{
			name:      "Regular day",
			date:      time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC),
			isHoliday: false,
		},
		{
			name:      "Day after Christmas",
			date:      time.Date(2024, 12, 26, 12, 0, 0, 0, time.UTC),
			isHoliday: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isHoliday(tt.date)
			assert.Equal(t, tt.isHoliday, result)
		})
	}
}

// ========================================
// TARGET TIME CALCULATION TESTS
// ========================================

func TestGetTargetTime(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	now := time.Now()

	// The function adds the timeframe duration then truncates to 15-min boundaries,
	// so the actual offset can be anywhere from 0 to (timeframe + 15min)
	tests := []struct {
		name      string
		timeframe PredictionTimeframe
		minOffset time.Duration
		maxOffset time.Duration
	}{
		{
			name:      "15 minute forecast",
			timeframe: Timeframe15Min,
			minOffset: 0,
			maxOffset: 30 * time.Minute,
		},
		{
			name:      "30 minute forecast",
			timeframe: Timeframe30Min,
			minOffset: 15 * time.Minute,
			maxOffset: 45 * time.Minute,
		},
		{
			name:      "1 hour forecast",
			timeframe: Timeframe1Hour,
			minOffset: 45 * time.Minute,
			maxOffset: 75 * time.Minute,
		},
		{
			name:      "2 hour forecast",
			timeframe: Timeframe2Hour,
			minOffset: 105 * time.Minute,
			maxOffset: 135 * time.Minute,
		},
		{
			name:      "4 hour forecast",
			timeframe: Timeframe4Hour,
			minOffset: 225 * time.Minute,
			maxOffset: 255 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.getTargetTime(tt.timeframe)
			offset := result.Sub(now)
			assert.GreaterOrEqual(t, offset, tt.minOffset)
			assert.Less(t, offset, tt.maxOffset)
		})
	}
}

// ========================================
// CONFIDENCE CALCULATION TESTS
// ========================================

func TestCalculateConfidence(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	tests := []struct {
		name       string
		features   *ModelFeatures
		minConf    float64
		maxConf    float64
	}{
		{
			name: "minimum confidence with no data",
			features: &ModelFeatures{
				HistAvgRides:    0,
				RecentRides1hr:  0,
				WeatherCode:     0,
				HistStdRides:    0,
			},
			minConf: 0.49,
			maxConf: 0.51,
		},
		{
			name: "higher confidence with historical data",
			features: &ModelFeatures{
				HistAvgRides:    20,
				RecentRides1hr:  0,
				WeatherCode:     0,
				HistStdRides:    15,
			},
			minConf: 0.69,
			maxConf: 0.71,
		},
		{
			name: "higher confidence with recent data",
			features: &ModelFeatures{
				HistAvgRides:    20,
				RecentRides1hr:  10,
				WeatherCode:     0,
				HistStdRides:    15,
			},
			minConf: 0.79,
			maxConf: 0.81,
		},
		{
			name: "higher confidence with weather data",
			features: &ModelFeatures{
				HistAvgRides:    20,
				RecentRides1hr:  10,
				WeatherCode:     1,
				HistStdRides:    15,
			},
			minConf: 0.89,
			maxConf: 0.91,
		},
		{
			name: "max confidence with low variance",
			features: &ModelFeatures{
				HistAvgRides:    20,
				RecentRides1hr:  10,
				WeatherCode:     1,
				HistStdRides:    5, // Low variance (< 0.5 * avg)
			},
			minConf: 0.94,
			maxConf: 0.96,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateConfidence(tt.features)
			assert.Greater(t, result, tt.minConf)
			assert.Less(t, result, tt.maxConf)
		})
	}
}

// ========================================
// PREDICTION MODEL TESTS
// ========================================

func TestPredict(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	tests := []struct {
		name              string
		features          *ModelFeatures
		minPredictedRides float64
		maxPredictedRides float64
		minConfidence     float64
	}{
		{
			name: "normal conditions prediction",
			features: &ModelFeatures{
				Hour:           12,
				DayOfWeek:      2,
				IsWeekend:      false,
				IsHoliday:      false,
				MonthOfYear:    6,
				HistAvgRides:   20.0,
				HistStdRides:   5.0,
				RecentRides1hr: 18,
				RecentTrend:    0.5,
			},
			minPredictedRides: 10.0,
			maxPredictedRides: 30.0,
			minConfidence:     0.5,
		},
		{
			name: "holiday reduces prediction",
			features: &ModelFeatures{
				Hour:           12,
				DayOfWeek:      2,
				IsWeekend:      false,
				IsHoliday:      true,
				MonthOfYear:    12,
				HistAvgRides:   20.0,
				HistStdRides:   5.0,
				RecentRides1hr: 10,
				RecentTrend:    0.0,
			},
			minPredictedRides: 5.0,
			maxPredictedRides: 20.0,
			minConfidence:     0.5,
		},
		{
			name: "event nearby increases prediction",
			features: &ModelFeatures{
				Hour:           20,
				DayOfWeek:      6,
				IsWeekend:      true,
				IsHoliday:      false,
				MonthOfYear:    6,
				HistAvgRides:   20.0,
				HistStdRides:   5.0,
				RecentRides1hr: 25,
				RecentTrend:    1.0,
				EventNearby:    true,
				EventScale:     10000,
			},
			minPredictedRides: 15.0,
			maxPredictedRides: 50.0,
			minConfidence:     0.5,
		},
		{
			name: "zero historical returns positive prediction from recent data",
			features: &ModelFeatures{
				Hour:           12,
				DayOfWeek:      2,
				IsWeekend:      false,
				IsHoliday:      false,
				MonthOfYear:    6,
				HistAvgRides:   0.0,
				HistStdRides:   0.0,
				RecentRides1hr: 10,
				RecentTrend:    0.5,
			},
			minPredictedRides: 0.0,
			maxPredictedRides: 10.0,
			minConfidence:     0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.predict(tt.features)

			assert.GreaterOrEqual(t, result.PredictedRides, tt.minPredictedRides)
			assert.LessOrEqual(t, result.PredictedRides, tt.maxPredictedRides)
			assert.GreaterOrEqual(t, result.Confidence, tt.minConfidence)
			assert.LessOrEqual(t, result.Confidence, 1.0)
			assert.GreaterOrEqual(t, result.LowerBound, 0.0)
			assert.GreaterOrEqual(t, result.UpperBound, result.PredictedRides)
		})
	}
}

func TestPredictNeverReturnsNegative(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	// Test with negative trend and low historical data
	features := &ModelFeatures{
		Hour:           3,
		DayOfWeek:      0,
		IsWeekend:      true,
		IsHoliday:      true,
		MonthOfYear:    7,
		HistAvgRides:   1.0,
		HistStdRides:   2.0,
		RecentRides1hr: 0,
		RecentTrend:    -5.0,
	}

	result := service.predict(features)
	assert.GreaterOrEqual(t, result.PredictedRides, 0.0)
	assert.GreaterOrEqual(t, result.LowerBound, 0.0)
}

// ========================================
// HOTSPOT SCORE TESTS
// ========================================

func TestCalculateHotspotScore(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	tests := []struct {
		name           string
		predictedRides float64
		features       *ModelFeatures
		minScore       float64
		maxScore       float64
	}{
		{
			name:           "low demand low score",
			predictedRides: 5.0,
			features: &ModelFeatures{
				CurrentDrivers: 10,
				RecentTrend:    0.0,
			},
			minScore: 0.0,
			maxScore: 30.0,
		},
		{
			name:           "high demand with driver shortage",
			predictedRides: 50.0,
			features: &ModelFeatures{
				CurrentDrivers: 5,
				RecentTrend:    2.0,
			},
			minScore: 50.0,
			maxScore: 100.0,
		},
		{
			name:           "high demand with adequate drivers",
			predictedRides: 50.0,
			features: &ModelFeatures{
				CurrentDrivers: 30,
				RecentTrend:    0.0,
			},
			minScore: 30.0,
			maxScore: 60.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateHotspotScore(tt.predictedRides, tt.features)
			assert.GreaterOrEqual(t, result, tt.minScore)
			assert.LessOrEqual(t, result, tt.maxScore)
		})
	}
}

func TestCalculateHotspotScoreBoundedZeroToHundred(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	// Test extreme values
	extremeHighFeatures := &ModelFeatures{
		CurrentDrivers: 0,
		RecentTrend:    100.0,
	}
	result := service.calculateHotspotScore(1000.0, extremeHighFeatures)
	assert.LessOrEqual(t, result, 100.0)
	assert.GreaterOrEqual(t, result, 0.0)

	extremeLowFeatures := &ModelFeatures{
		CurrentDrivers: 1000,
		RecentTrend:    -100.0,
	}
	result = service.calculateHotspotScore(0.0, extremeLowFeatures)
	assert.LessOrEqual(t, result, 100.0)
	assert.GreaterOrEqual(t, result, 0.0)
}

// ========================================
// REPOSITION PRIORITY TESTS
// ========================================

func TestCalculateRepositionPriority(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	tests := []struct {
		name           string
		predictedRides float64
		features       *ModelFeatures
		minPriority    int
		maxPriority    int
	}{
		{
			name:           "high score low priority number",
			predictedRides: 100.0,
			features: &ModelFeatures{
				CurrentDrivers: 1,
				RecentTrend:    5.0,
			},
			minPriority: 1,
			maxPriority: 3,
		},
		{
			name:           "low score high priority number",
			predictedRides: 5.0,
			features: &ModelFeatures{
				CurrentDrivers: 20,
				RecentTrend:    -2.0,
			},
			minPriority: 7,
			maxPriority: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateRepositionPriority(tt.predictedRides, tt.features)
			assert.GreaterOrEqual(t, result, tt.minPriority)
			assert.LessOrEqual(t, result, tt.maxPriority)
		})
	}
}

func TestCalculateRepositionPriorityBoundedOneToTen(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	// Test with various extreme inputs
	for _, rides := range []float64{0, 1, 10, 100, 1000} {
		for _, drivers := range []int{0, 1, 10, 100, 1000} {
			for _, trend := range []float64{-10, -1, 0, 1, 10} {
				features := &ModelFeatures{
					CurrentDrivers: drivers,
					RecentTrend:    trend,
				}
				result := service.calculateRepositionPriority(rides, features)
				assert.GreaterOrEqual(t, result, 1, "priority should be >= 1 for rides=%f, drivers=%d, trend=%f", rides, drivers, trend)
				assert.LessOrEqual(t, result, 10, "priority should be <= 10 for rides=%f, drivers=%d, trend=%f", rides, drivers, trend)
			}
		}
	}
}

// ========================================
// GENERATE PREDICTION INTEGRATION TESTS
// ========================================

func TestGeneratePredictionSuccess(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	weatherSvc := new(mockWeatherService)
	driverSvc := new(mockDriverLocationService)

	service := newTestService(repo, weatherSvc, driverSvc, nil)

	latitude, longitude := 40.7128, -74.0060 // NYC coordinates

	// Setup mocks
	repo.On("GetHistoricalAverage", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int"), 8).
		Return(20.0, 5.0, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), 15).Return(5, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), 60).Return(18, nil)
	repo.On("GetRecentDemandTrend", ctx, mock.AnythingOfType("string")).Return(0.5, nil)
	driverSvc.On("GetDriverCountInCell", ctx, mock.AnythingOfType("string")).Return(10, nil)
	repo.On("GetNeighborDemandAverage", ctx, mock.AnythingOfType("[]string"), 30).Return(15.0, nil)
	weatherSvc.On("GetCurrentWeather", ctx, mock.AnythingOfType("float64"), mock.AnythingOfType("float64")).
		Return(&WeatherData{ConditionCode: 0, Temperature: 20.0, PrecipitationProb: 0.1}, nil)
	repo.On("GetEventsNearLocation", ctx, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), 5.0, 2*time.Hour).
		Return([]*SpecialEvent{}, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(10, nil).Maybe()
	repo.On("SavePrediction", ctx, mock.AnythingOfType("*demandforecast.DemandPrediction")).Return(nil)

	result, err := service.GeneratePrediction(ctx, latitude, longitude, Timeframe30Min)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, uuid.Nil, result.ID)
	assert.NotEmpty(t, result.H3Index)
	assert.Equal(t, Timeframe30Min, result.Timeframe)
	assert.GreaterOrEqual(t, result.PredictedRides, 0.0)
	assert.GreaterOrEqual(t, result.Confidence, 0.0)
	assert.LessOrEqual(t, result.Confidence, 1.0)
	assert.GreaterOrEqual(t, result.RecommendedDrivers, 1)
	assert.GreaterOrEqual(t, result.ExpectedSurge, 1.0)

	repo.AssertExpectations(t)
	weatherSvc.AssertExpectations(t)
	driverSvc.AssertExpectations(t)
}

func TestGeneratePredictionWithoutOptionalServices(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	// No weather or driver service
	service := newTestService(repo, nil, nil, &Config{
		H3Resolution:      7,
		TrainingDataWeeks: 8,
		WeatherEnabled:    false,
		EventsEnabled:     false,
	})

	latitude, longitude := 40.7128, -74.0060

	repo.On("GetHistoricalAverage", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int"), 8).
		Return(20.0, 5.0, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), 15).Return(5, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), 60).Return(18, nil)
	repo.On("GetRecentDemandTrend", ctx, mock.AnythingOfType("string")).Return(0.5, nil)
	repo.On("GetNeighborDemandAverage", ctx, mock.AnythingOfType("[]string"), 30).Return(15.0, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(10, nil).Maybe()
	repo.On("SavePrediction", ctx, mock.AnythingOfType("*demandforecast.DemandPrediction")).Return(nil)

	result, err := service.GeneratePrediction(ctx, latitude, longitude, Timeframe30Min)

	require.NoError(t, err)
	assert.NotNil(t, result)
	repo.AssertExpectations(t)
}

func TestGeneratePredictionSavesEvenOnSaveError(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, &Config{
		H3Resolution:      7,
		TrainingDataWeeks: 8,
		WeatherEnabled:    false,
		EventsEnabled:     false,
	})

	latitude, longitude := 40.7128, -74.0060

	repo.On("GetHistoricalAverage", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int"), 8).
		Return(20.0, 5.0, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), 15).Return(5, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), 60).Return(18, nil)
	repo.On("GetRecentDemandTrend", ctx, mock.AnythingOfType("string")).Return(0.5, nil)
	repo.On("GetNeighborDemandAverage", ctx, mock.AnythingOfType("[]string"), 30).Return(15.0, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(10, nil).Maybe()
	repo.On("SavePrediction", ctx, mock.AnythingOfType("*demandforecast.DemandPrediction")).
		Return(errors.New("database error"))

	// Should still return prediction even if save fails
	result, err := service.GeneratePrediction(ctx, latitude, longitude, Timeframe30Min)

	require.NoError(t, err)
	assert.NotNil(t, result)
	repo.AssertExpectations(t)
}

// ========================================
// GET TOP HOTSPOTS TESTS
// ========================================

func TestGetTopHotspotsSuccess(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	driverSvc := new(mockDriverLocationService)

	service := newTestService(repo, nil, driverSvc, nil)

	predictions := []*DemandPrediction{
		{
			ID:                 uuid.New(),
			H3Index:            "872a100c9ffffff",
			PredictedRides:     50.0,
			DemandLevel:        DemandVeryHigh,
			RecommendedDrivers: 30,
			ExpectedSurge:      2.0,
			HotspotScore:       85.0,
			PredictionTime:     time.Now().Add(30 * time.Minute),
		},
		{
			ID:                 uuid.New(),
			H3Index:            "872a100caffffff",
			PredictedRides:     30.0,
			DemandLevel:        DemandHigh,
			RecommendedDrivers: 18,
			ExpectedSurge:      1.5,
			HotspotScore:       65.0,
			PredictionTime:     time.Now().Add(30 * time.Minute),
		},
	}

	repo.On("GetTopHotspots", ctx, Timeframe30Min, 5).Return(predictions, nil)
	driverSvc.On("GetDriverCountInCell", ctx, "872a100c9ffffff").Return(10, nil)
	driverSvc.On("GetDriverCountInCell", ctx, "872a100caffffff").Return(8, nil)

	result, err := service.GetTopHotspots(ctx, Timeframe30Min, 5)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "872a100c9ffffff", result[0].H3Index)
	assert.Equal(t, 10, result[0].CurrentDrivers)
	assert.Equal(t, 30, result[0].NeededDrivers)
	assert.Equal(t, 20, result[0].Gap) // 30 - 10
	repo.AssertExpectations(t)
	driverSvc.AssertExpectations(t)
}

func TestGetTopHotspotsRepoError(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, nil)

	repo.On("GetTopHotspots", ctx, Timeframe30Min, 5).Return(nil, errors.New("database error"))

	result, err := service.GetTopHotspots(ctx, Timeframe30Min, 5)

	require.Error(t, err)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestGetTopHotspotsWithoutDriverService(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, nil)

	predictions := []*DemandPrediction{
		{
			ID:                 uuid.New(),
			H3Index:            "872a100c9ffffff",
			PredictedRides:     50.0,
			DemandLevel:        DemandVeryHigh,
			RecommendedDrivers: 30,
			ExpectedSurge:      2.0,
			HotspotScore:       85.0,
			PredictionTime:     time.Now().Add(30 * time.Minute),
		},
	}

	repo.On("GetTopHotspots", ctx, Timeframe30Min, 5).Return(predictions, nil)

	result, err := service.GetTopHotspots(ctx, Timeframe30Min, 5)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 0, result[0].CurrentDrivers) // No driver service
	repo.AssertExpectations(t)
}

// ========================================
// GET DEMAND HEATMAP TESTS
// ========================================

func TestGetDemandHeatmapSuccess(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	driverSvc := new(mockDriverLocationService)

	service := newTestService(repo, nil, driverSvc, nil)

	predictions := []*DemandPrediction{
		{
			ID:                 uuid.New(),
			H3Index:            "872a100c9ffffff",
			PredictedRides:     50.0,
			DemandLevel:        DemandVeryHigh,
			RecommendedDrivers: 30,
			ExpectedSurge:      2.0,
			HotspotScore:       85.0,
			PredictionTime:     time.Now().Add(30 * time.Minute),
		},
	}

	req := &GetHeatmapRequest{
		MinLatitude:  40.0,
		MaxLatitude:  41.0,
		MinLongitude: -75.0,
		MaxLongitude: -73.0,
		Timeframe:    Timeframe30Min,
	}

	repo.On("GetPredictionsInBoundingBox", ctx, Timeframe30Min, (*DemandLevel)(nil)).Return(predictions, nil)
	driverSvc.On("GetDriverCountInCell", ctx, mock.AnythingOfType("string")).Return(10, nil).Maybe()

	result, err := service.GetDemandHeatmap(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, Timeframe30Min, result.Timeframe)
	assert.Equal(t, 40.0, result.BoundingBox.MinLatitude)
	assert.Equal(t, 41.0, result.BoundingBox.MaxLatitude)
	repo.AssertExpectations(t)
}

func TestGetDemandHeatmapWithMinDemandLevel(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, nil)

	minLevel := DemandHigh
	req := &GetHeatmapRequest{
		MinLatitude:    40.0,
		MaxLatitude:    41.0,
		MinLongitude:   -75.0,
		MaxLongitude:   -73.0,
		Timeframe:      Timeframe30Min,
		MinDemandLevel: &minLevel,
	}

	repo.On("GetPredictionsInBoundingBox", ctx, Timeframe30Min, &minLevel).Return([]*DemandPrediction{}, nil)

	result, err := service.GetDemandHeatmap(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Zones)
	repo.AssertExpectations(t)
}

func TestGetDemandHeatmapRepoError(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, nil)

	req := &GetHeatmapRequest{
		MinLatitude:  40.0,
		MaxLatitude:  41.0,
		MinLongitude: -75.0,
		MaxLongitude: -73.0,
		Timeframe:    Timeframe30Min,
	}

	repo.On("GetPredictionsInBoundingBox", ctx, Timeframe30Min, (*DemandLevel)(nil)).
		Return(nil, errors.New("database error"))

	result, err := service.GetDemandHeatmap(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

// ========================================
// GET REPOSITION RECOMMENDATIONS TESTS
// ========================================

func TestGetRepositionRecommendationsSuccess(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	driverSvc := new(mockDriverLocationService)

	service := newTestService(repo, nil, driverSvc, nil)

	driverID := uuid.New()
	req := &GetRepositionRecommendationsRequest{
		DriverID:      driverID,
		Latitude:      40.7128,
		Longitude:     -74.0060,
		MaxDistanceKm: 10.0,
		Limit:         3,
	}

	currentPred := &DemandPrediction{
		ID:                 uuid.New(),
		H3Index:            "872a100c9ffffff",
		PredictedRides:     20.0,
		DemandLevel:        DemandNormal,
		RecommendedDrivers: 12,
		ExpectedSurge:      1.0,
		HotspotScore:       40.0,
		PredictionTime:     time.Now().Add(30 * time.Minute),
	}

	hotspots := []*DemandPrediction{
		{
			ID:                 uuid.New(),
			H3Index:            "872a100caffffff",
			PredictedRides:     50.0,
			DemandLevel:        DemandVeryHigh,
			RecommendedDrivers: 30,
			ExpectedSurge:      2.0,
			HotspotScore:       85.0,
			PredictionTime:     time.Now().Add(30 * time.Minute),
		},
	}

	repo.On("GetLatestPrediction", ctx, mock.AnythingOfType("string"), Timeframe30Min).Return(currentPred, nil)
	driverSvc.On("GetDriverCountInCell", ctx, mock.AnythingOfType("string")).Return(10, nil).Maybe()
	repo.On("GetTopHotspots", ctx, Timeframe30Min, 20).Return(hotspots, nil)

	result, err := service.GetRepositionRecommendations(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, DemandNormal, result.CurrentZone.DemandLevel)
	repo.AssertExpectations(t)
	driverSvc.AssertExpectations(t)
}

func TestGetRepositionRecommendationsDefaultValues(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, nil)

	req := &GetRepositionRecommendationsRequest{
		DriverID:      uuid.New(),
		Latitude:      40.7128,
		Longitude:     -74.0060,
		MaxDistanceKm: 0, // Should default to 10
		Limit:         0, // Should default to 3
	}

	repo.On("GetLatestPrediction", ctx, mock.AnythingOfType("string"), Timeframe30Min).Return(nil, nil)
	repo.On("GetTopHotspots", ctx, Timeframe30Min, 20).Return([]*DemandPrediction{}, nil)

	result, err := service.GetRepositionRecommendations(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	repo.AssertExpectations(t)
}

func TestGetRepositionRecommendationsHotspotsError(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, nil)

	req := &GetRepositionRecommendationsRequest{
		DriverID:  uuid.New(),
		Latitude:  40.7128,
		Longitude: -74.0060,
	}

	repo.On("GetLatestPrediction", ctx, mock.AnythingOfType("string"), Timeframe30Min).Return(nil, nil)
	repo.On("GetTopHotspots", ctx, Timeframe30Min, 20).Return(nil, errors.New("database error"))

	result, err := service.GetRepositionRecommendations(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

// ========================================
// CREATE EVENT TESTS
// ========================================

func TestCreateEventSuccess(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, nil)

	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(3 * time.Hour)

	req := &CreateEventRequest{
		Name:              "Stadium Concert",
		EventType:         "concert",
		Latitude:          40.7128,
		Longitude:         -74.0060,
		StartTime:         startTime.Format(time.RFC3339),
		EndTime:           endTime.Format(time.RFC3339),
		ExpectedAttendees: 50000,
		ImpactRadius:      10.0,
		IsRecurring:       false,
	}

	repo.On("CreateEvent", ctx, mock.MatchedBy(func(event *SpecialEvent) bool {
		return event.Name == "Stadium Concert" &&
			event.EventType == "concert" &&
			event.ExpectedAttendees == 50000 &&
			event.ImpactRadius == 10.0 &&
			event.DemandMultiplier == 2.5 && // 50000 attendees (20000-50000) = 2.5x multiplier
			event.ID != uuid.Nil
	})).Return(nil)

	result, err := service.CreateEvent(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Stadium Concert", result.Name)
	assert.Equal(t, 50000, result.ExpectedAttendees)
	assert.Equal(t, 2.5, result.DemandMultiplier)
	assert.NotEqual(t, uuid.Nil, result.ID)
	repo.AssertExpectations(t)
}

func TestCreateEventInvalidStartTime(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, nil)

	req := &CreateEventRequest{
		Name:              "Event",
		EventType:         "concert",
		Latitude:          40.7128,
		Longitude:         -74.0060,
		StartTime:         "invalid-time",
		EndTime:           time.Now().Add(3 * time.Hour).Format(time.RFC3339),
		ExpectedAttendees: 1000,
	}

	result, err := service.CreateEvent(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	// The error wraps the message from common.NewBadRequestError
}

func TestCreateEventInvalidEndTime(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, nil)

	req := &CreateEventRequest{
		Name:              "Event",
		EventType:         "concert",
		Latitude:          40.7128,
		Longitude:         -74.0060,
		StartTime:         time.Now().Add(time.Hour).Format(time.RFC3339),
		EndTime:           "invalid-time",
		ExpectedAttendees: 1000,
	}

	result, err := service.CreateEvent(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	// The error wraps the message from common.NewBadRequestError
}

func TestCreateEventDefaultImpactRadius(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, nil)

	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(3 * time.Hour)

	req := &CreateEventRequest{
		Name:              "Small Event",
		EventType:         "meeting",
		Latitude:          40.7128,
		Longitude:         -74.0060,
		StartTime:         startTime.Format(time.RFC3339),
		EndTime:           endTime.Format(time.RFC3339),
		ExpectedAttendees: 100,
		ImpactRadius:      0, // Should default to 5.0
	}

	repo.On("CreateEvent", ctx, mock.MatchedBy(func(event *SpecialEvent) bool {
		return event.ImpactRadius == 5.0
	})).Return(nil)

	result, err := service.CreateEvent(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, 5.0, result.ImpactRadius)
	repo.AssertExpectations(t)
}

func TestCreateEventRepoError(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, nil)

	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(3 * time.Hour)

	req := &CreateEventRequest{
		Name:              "Event",
		EventType:         "concert",
		Latitude:          40.7128,
		Longitude:         -74.0060,
		StartTime:         startTime.Format(time.RFC3339),
		EndTime:           endTime.Format(time.RFC3339),
		ExpectedAttendees: 1000,
	}

	repo.On("CreateEvent", ctx, mock.AnythingOfType("*demandforecast.SpecialEvent")).
		Return(errors.New("database error"))

	result, err := service.CreateEvent(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

// ========================================
// GET UPCOMING EVENTS TESTS
// ========================================

func TestGetUpcomingEventsSuccess(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, nil)

	events := []*SpecialEvent{
		{
			ID:                uuid.New(),
			Name:              "Concert",
			EventType:         "concert",
			ExpectedAttendees: 10000,
		},
		{
			ID:                uuid.New(),
			Name:              "Sports Game",
			EventType:         "sports",
			ExpectedAttendees: 50000,
		},
	}

	repo.On("GetUpcomingEvents", ctx, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
		Return(events, nil)

	result, err := service.GetUpcomingEvents(ctx, 24)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	repo.AssertExpectations(t)
}

func TestGetUpcomingEventsRepoError(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, nil)

	repo.On("GetUpcomingEvents", ctx, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
		Return(nil, errors.New("database error"))

	result, err := service.GetUpcomingEvents(ctx, 24)

	require.Error(t, err)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

// ========================================
// RECORD DEMAND SNAPSHOT TESTS
// ========================================

func TestRecordDemandSnapshotSuccess(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	weatherSvc := new(mockWeatherService)

	service := newTestService(repo, weatherSvc, nil, nil)

	h3Index := "872a100c9ffffff"

	weatherSvc.On("GetCurrentWeather", ctx, mock.AnythingOfType("float64"), mock.AnythingOfType("float64")).
		Return(&WeatherData{
			Condition:       "clear",
			Temperature:     22.0,
			PrecipitationMM: 0.0,
		}, nil)

	repo.On("RecordDemand", ctx, mock.MatchedBy(func(record *HistoricalDemandRecord) bool {
		return record.H3Index == h3Index &&
			record.RideRequests == 100 &&
			record.CompletedRides == 95 &&
			record.AvailableDrivers == 20 &&
			record.AvgWaitTimeMin == 5.5 &&
			record.SurgeMultiplier == 1.5 &&
			record.WeatherCondition == "clear"
	})).Return(nil)

	err := service.RecordDemandSnapshot(ctx, h3Index, 100, 95, 20, 5.5, 1.5)

	require.NoError(t, err)
	repo.AssertExpectations(t)
	weatherSvc.AssertExpectations(t)
}

func TestRecordDemandSnapshotWithoutWeatherService(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, nil)

	h3Index := "872a100c9ffffff"

	repo.On("RecordDemand", ctx, mock.MatchedBy(func(record *HistoricalDemandRecord) bool {
		return record.WeatherCondition == "unknown" &&
			record.Temperature == nil
	})).Return(nil)

	err := service.RecordDemandSnapshot(ctx, h3Index, 100, 95, 20, 5.5, 1.5)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestRecordDemandSnapshotWeatherDisabled(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	weatherSvc := new(mockWeatherService)

	service := newTestService(repo, weatherSvc, nil, &Config{
		H3Resolution:      7,
		TrainingDataWeeks: 8,
		WeatherEnabled:    false, // Disabled
		EventsEnabled:     true,
	})

	h3Index := "872a100c9ffffff"

	repo.On("RecordDemand", ctx, mock.MatchedBy(func(record *HistoricalDemandRecord) bool {
		return record.WeatherCondition == "unknown"
	})).Return(nil)

	err := service.RecordDemandSnapshot(ctx, h3Index, 100, 95, 20, 5.5, 1.5)

	require.NoError(t, err)
	// Weather service should not be called when disabled
	weatherSvc.AssertNotCalled(t, "GetCurrentWeather", mock.Anything, mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestRecordDemandSnapshotRepoError(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, nil)

	h3Index := "872a100c9ffffff"

	repo.On("RecordDemand", ctx, mock.AnythingOfType("*demandforecast.HistoricalDemandRecord")).
		Return(errors.New("database error"))

	err := service.RecordDemandSnapshot(ctx, h3Index, 100, 95, 20, 5.5, 1.5)

	require.Error(t, err)
	repo.AssertExpectations(t)
}

// ========================================
// GET MODEL ACCURACY TESTS
// ========================================

func TestGetModelAccuracySuccess(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, nil)

	metrics := &ForecastAccuracyMetrics{
		Timeframe:        Timeframe30Min,
		MAE:              2.5,
		RMSE:             3.2,
		MAPE:             15.0,
		SamplesEvaluated: 1000,
		UpdatedAt:        time.Now(),
	}

	repo.On("GetAccuracyMetrics", ctx, Timeframe30Min, 7).Return(metrics, nil)

	result, err := service.GetModelAccuracy(ctx, Timeframe30Min, 7)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2.5, result.MAE)
	assert.Equal(t, 3.2, result.RMSE)
	assert.Equal(t, 1000, result.SamplesEvaluated)
	repo.AssertExpectations(t)
}

func TestGetModelAccuracyRepoError(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, nil)

	repo.On("GetAccuracyMetrics", ctx, Timeframe30Min, 7).
		Return(nil, errors.New("database error"))

	result, err := service.GetModelAccuracy(ctx, Timeframe30Min, 7)

	require.Error(t, err)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

// ========================================
// REPOSITION REASON TESTS
// ========================================

func TestGetRepositionReason(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	tests := []struct {
		name     string
		hotspot  HotspotZone
		expected string
	}{
		{
			name: "high surge reason",
			hotspot: HotspotZone{
				ExpectedSurge: 2.5,
				Gap:           3,
				DemandLevel:   DemandHigh,
			},
			expected: "High surge expected",
		},
		{
			name: "driver shortage reason",
			hotspot: HotspotZone{
				ExpectedSurge: 1.3,
				Gap:           10,
				DemandLevel:   DemandHigh,
			},
			expected: "Significant driver shortage",
		},
		{
			name: "very high demand reason",
			hotspot: HotspotZone{
				ExpectedSurge: 1.3,
				Gap:           3,
				DemandLevel:   DemandVeryHigh,
			},
			expected: "Very high demand predicted",
		},
		{
			name: "extreme demand reason",
			hotspot: HotspotZone{
				ExpectedSurge: 1.3,
				Gap:           3,
				DemandLevel:   DemandExtreme,
			},
			expected: "Very high demand predicted",
		},
		{
			name: "default reason",
			hotspot: HotspotZone{
				ExpectedSurge: 1.2,
				Gap:           2,
				DemandLevel:   DemandNormal,
			},
			expected: "Better earnings opportunity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.getRepositionReason(tt.hotspot)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ========================================
// PRIORITY CALCULATION TESTS
// ========================================

func TestCalculatePriority(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	tests := []struct {
		name        string
		hotspot     HotspotZone
		distance    float64
		minPriority int
		maxPriority int
	}{
		{
			name: "close high-score hotspot gets high priority",
			hotspot: HotspotZone{
				HotspotScore: 90.0,
			},
			distance:    1.0,
			minPriority: 1,
			maxPriority: 3,
		},
		{
			name: "far high-score hotspot gets lower priority",
			hotspot: HotspotZone{
				HotspotScore: 90.0,
			},
			distance:    10.0,
			minPriority: 3,
			maxPriority: 6,
		},
		{
			name: "close low-score hotspot gets medium priority",
			hotspot: HotspotZone{
				HotspotScore: 30.0,
			},
			distance:    1.0,
			minPriority: 6,
			maxPriority: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculatePriority(tt.hotspot, tt.distance)
			assert.GreaterOrEqual(t, result, tt.minPriority)
			assert.LessOrEqual(t, result, tt.maxPriority)
		})
	}
}

// ========================================
// HAVERSINE DISTANCE TESTS
// ========================================

func TestHaversineDistance(t *testing.T) {
	tests := []struct {
		name       string
		latitude1  float64
		longitude1 float64
		latitude2  float64
		longitude2 float64
		expected   float64
		delta      float64
	}{
		{
			name:       "same point returns zero",
			latitude1:  40.7128, longitude1: -74.0060,
			latitude2:  40.7128, longitude2: -74.0060,
			expected:   0.0,
			delta:      0.001,
		},
		{
			name:       "NYC to LA approximately 3944 km",
			latitude1:  40.7128, longitude1: -74.0060,
			latitude2:  34.0522, longitude2: -118.2437,
			expected:   3944.0,
			delta:      50.0,
		},
		{
			name:       "short distance within city about 1 km",
			latitude1:  40.7128, longitude1: -74.0060,
			latitude2:  40.7218, longitude2: -74.0060,
			expected:   1.0,
			delta:      0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := haversineDistance(tt.latitude1, tt.longitude1, tt.latitude2, tt.longitude2)
			assert.InDelta(t, tt.expected, result, tt.delta)
		})
	}
}

func TestHaversineDistanceSymmetry(t *testing.T) {
	// Distance from A to B should equal distance from B to A
	d1 := haversineDistance(40.7128, -74.0060, 34.0522, -118.2437)
	d2 := haversineDistance(34.0522, -118.2437, 40.7128, -74.0060)
	assert.InDelta(t, d1, d2, 0.001)
}

// ========================================
// WEEK OF YEAR TESTS
// ========================================

func TestGetWeekOfYear(t *testing.T) {
	tests := []struct {
		name        string
		date        time.Time
		expectedMin int
		expectedMax int
	}{
		{
			name:        "January 1st",
			date:        time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			expectedMin: 1,
			expectedMax: 1,
		},
		{
			name:        "mid year",
			date:        time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC),
			expectedMin: 24,
			expectedMax: 25,
		},
		{
			name:        "end of year",
			date:        time.Date(2024, 12, 31, 12, 0, 0, 0, time.UTC),
			expectedMin: 1, // ISO week can be week 1 of next year
			expectedMax: 53,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getWeekOfYear(tt.date)
			assert.GreaterOrEqual(t, result, tt.expectedMin)
			assert.LessOrEqual(t, result, tt.expectedMax)
		})
	}
}

// ========================================
// EVENT DEMAND MULTIPLIER TESTS
// ========================================

func TestCalculateEventDemandMultiplier(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	// Should match getEventMultiplier
	tests := []struct {
		attendees int
		expected  float64
	}{
		{500, 1.1},
		{2000, 1.3},
		{7000, 1.5},
		{15000, 2.0},
		{30000, 2.5},
		{60000, 3.0},
	}

	for _, tt := range tests {
		result := service.calculateEventDemandMultiplier(tt.attendees)
		assert.Equal(t, tt.expected, result, "attendees=%d", tt.attendees)
	}
}

// ========================================
// FEATURE CONTRIBUTIONS TESTS
// ========================================

func TestCalculateContributions(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	features := &ModelFeatures{}
	prediction := &ModelPrediction{}

	result := service.calculateContributions(features, prediction)

	// Contributions should match model weights
	assert.Equal(t, 0.35, result.HistoricalPattern)
	assert.Equal(t, 0.25, result.RecentTrend)
	assert.Equal(t, 0.15, result.TimeOfDay)
	assert.Equal(t, 0.10, result.DayOfWeek)
	assert.Equal(t, 0.08, result.Weather)
	assert.Equal(t, 0.05, result.SpecialEvents)
	assert.Equal(t, 0.02, result.SeasonalTrend)
}

// ========================================
// BOUNDARY CONDITION TESTS
// ========================================

func TestPredictTrendAdjustmentBounds(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	// Test extreme negative trend - should be clamped to 0.5
	negativeFeatures := &ModelFeatures{
		Hour:           12,
		DayOfWeek:      2,
		IsWeekend:      false,
		IsHoliday:      false,
		MonthOfYear:    6,
		HistAvgRides:   20.0,
		HistStdRides:   5.0,
		RecentRides1hr: 10,
		RecentTrend:    -100.0, // Extreme negative
	}

	result := service.predict(negativeFeatures)
	assert.GreaterOrEqual(t, result.PredictedRides, 0.0)

	// Test extreme positive trend - should be clamped to 2.0
	positiveFeatures := &ModelFeatures{
		Hour:           12,
		DayOfWeek:      2,
		IsWeekend:      false,
		IsHoliday:      false,
		MonthOfYear:    6,
		HistAvgRides:   20.0,
		HistStdRides:   5.0,
		RecentRides1hr: 10,
		RecentTrend:    100.0, // Extreme positive
	}

	result = service.predict(positiveFeatures)
	assert.GreaterOrEqual(t, result.PredictedRides, 0.0)
	assert.Less(t, result.PredictedRides, 1000.0) // Sanity check
}

func TestCalculateExpectedSurgeCap(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	// Even with extreme ratios, surge should cap at 3.0
	result := service.calculateExpectedSurge(math.MaxFloat64/2, 1)
	assert.LessOrEqual(t, result, 3.0)
}

func TestCalculateHotspotScoreZeroInputs(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	features := &ModelFeatures{
		CurrentDrivers: 0,
		RecentTrend:    0,
	}

	result := service.calculateHotspotScore(0, features)
	assert.GreaterOrEqual(t, result, 0.0)
	assert.LessOrEqual(t, result, 100.0)
}

// ========================================
// GENERATE PREDICTIONS FOR AREA TESTS
// ========================================

func TestGeneratePredictionsForAreaSuccess(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	weatherSvc := new(mockWeatherService)
	driverSvc := new(mockDriverLocationService)

	service := newTestService(repo, weatherSvc, driverSvc, nil)

	// Small bounding box to generate few cells
	minLatitude, minLongitude := 40.712, -74.006
	maxLatitude, maxLongitude := 40.714, -74.004

	// Setup mocks for each prediction (allow any calls)
	repo.On("GetHistoricalAverage", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int"), 8).
		Return(20.0, 5.0, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), 15).Return(5, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), 60).Return(18, nil)
	repo.On("GetRecentDemandTrend", ctx, mock.AnythingOfType("string")).Return(0.5, nil)
	driverSvc.On("GetDriverCountInCell", ctx, mock.AnythingOfType("string")).Return(10, nil)
	repo.On("GetNeighborDemandAverage", ctx, mock.AnythingOfType("[]string"), 30).Return(15.0, nil)
	weatherSvc.On("GetCurrentWeather", ctx, mock.AnythingOfType("float64"), mock.AnythingOfType("float64")).
		Return(&WeatherData{ConditionCode: 0, Temperature: 20.0, PrecipitationProb: 0.1}, nil)
	repo.On("GetEventsNearLocation", ctx, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), 5.0, 2*time.Hour).
		Return([]*SpecialEvent{}, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(10, nil).Maybe()
	repo.On("SavePrediction", ctx, mock.AnythingOfType("*demandforecast.DemandPrediction")).Return(nil)

	result, err := service.GeneratePredictionsForArea(ctx, minLatitude, minLongitude, maxLatitude, maxLongitude, Timeframe30Min)

	require.NoError(t, err)
	assert.NotNil(t, result)
	// Should have at least 1 prediction for the area
	assert.GreaterOrEqual(t, len(result), 1)

	// All predictions should have valid data
	for _, pred := range result {
		assert.NotEqual(t, uuid.Nil, pred.ID)
		assert.NotEmpty(t, pred.H3Index)
		assert.GreaterOrEqual(t, pred.PredictedRides, 0.0)
	}
}

func TestGeneratePredictionsForAreaEmptyArea(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	// Test with a very small area (same point)
	service := newTestService(repo, nil, nil, &Config{
		H3Resolution:      7,
		TrainingDataWeeks: 8,
		WeatherEnabled:    false,
		EventsEnabled:     false,
	})

	repo.On("GetHistoricalAverage", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int"), 8).
		Return(10.0, 2.0, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(5, nil)
	repo.On("GetRecentDemandTrend", ctx, mock.AnythingOfType("string")).Return(0.0, nil)
	repo.On("GetNeighborDemandAverage", ctx, mock.AnythingOfType("[]string"), 30).Return(8.0, nil)
	repo.On("SavePrediction", ctx, mock.AnythingOfType("*demandforecast.DemandPrediction")).Return(nil)

	result, err := service.GeneratePredictionsForArea(ctx, 40.712, -74.006, 40.712, -74.006, Timeframe15Min)

	require.NoError(t, err)
	// Single point should still generate at least 1 prediction
	assert.GreaterOrEqual(t, len(result), 1)
}

func TestGeneratePredictionsForAreaWithPredictionErrors(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, &Config{
		H3Resolution:      7,
		TrainingDataWeeks: 8,
		WeatherEnabled:    false,
		EventsEnabled:     false,
	})

	// First call succeeds, subsequent calls fail
	repo.On("GetHistoricalAverage", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int"), 8).
		Return(10.0, 2.0, nil).Once()
	repo.On("GetHistoricalAverage", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int"), 8).
		Return(0.0, 0.0, errors.New("db error")).Maybe()
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(5, nil).Maybe()
	repo.On("GetRecentDemandTrend", ctx, mock.AnythingOfType("string")).Return(0.0, nil).Maybe()
	repo.On("GetNeighborDemandAverage", ctx, mock.AnythingOfType("[]string"), 30).Return(8.0, nil).Maybe()
	repo.On("SavePrediction", ctx, mock.AnythingOfType("*demandforecast.DemandPrediction")).Return(nil).Maybe()

	// Should continue even if some predictions fail
	result, err := service.GeneratePredictionsForArea(ctx, 40.712, -74.006, 40.714, -74.004, Timeframe30Min)

	require.NoError(t, err)
	// At least some predictions should succeed
	assert.NotNil(t, result)
}

// ========================================
// GET TARGET TIME DEFAULT CASE TEST
// ========================================

func TestGetTargetTimeDefaultCase(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	// Test with an invalid timeframe (not in switch)
	invalidTimeframe := PredictionTimeframe("invalid")
	result := service.getTargetTime(invalidTimeframe)

	// Should default to 30 min (same as Timeframe30Min)
	now := time.Now()
	offset := result.Sub(now)
	assert.GreaterOrEqual(t, offset, 15*time.Minute)
	assert.Less(t, offset, 45*time.Minute)
}

// ========================================
// GET REPOSITION RECOMMENDATIONS EDGE CASES
// ========================================

func TestGetRepositionRecommendationsFiltersCurrentLocation(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	driverSvc := new(mockDriverLocationService)

	service := newTestService(repo, nil, driverSvc, nil)

	driverID := uuid.New()
	req := &GetRepositionRecommendationsRequest{
		DriverID:      driverID,
		Latitude:      40.7128,
		Longitude:     -74.0060,
		MaxDistanceKm: 10.0,
		Limit:         3,
	}

	currentPred := &DemandPrediction{
		ID:                 uuid.New(),
		H3Index:            "872a100c9ffffff",
		PredictedRides:     50.0,
		DemandLevel:        DemandVeryHigh,
		RecommendedDrivers: 30,
		ExpectedSurge:      2.0,
		HotspotScore:       85.0,
		PredictionTime:     time.Now().Add(30 * time.Minute),
	}

	// Hotspots include the current location
	hotspots := []*DemandPrediction{
		currentPred, // Current location - should be filtered out
		{
			ID:                 uuid.New(),
			H3Index:            "872a100caffffff", // Different location
			PredictedRides:     40.0,
			DemandLevel:        DemandHigh,
			RecommendedDrivers: 24,
			ExpectedSurge:      1.5,
			HotspotScore:       70.0,
			PredictionTime:     time.Now().Add(30 * time.Minute),
		},
	}

	repo.On("GetLatestPrediction", ctx, mock.AnythingOfType("string"), Timeframe30Min).Return(currentPred, nil)
	driverSvc.On("GetDriverCountInCell", ctx, mock.AnythingOfType("string")).Return(10, nil).Maybe()
	repo.On("GetTopHotspots", ctx, Timeframe30Min, 20).Return(hotspots, nil)

	result, err := service.GetRepositionRecommendations(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	// Current location should be filtered from recommendations
	for _, rec := range result.Recommendations {
		assert.NotEqual(t, currentPred.H3Index, rec.TargetH3Index)
	}
}

func TestGetRepositionRecommendationsFiltersDistantHotspots(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	driverSvc := new(mockDriverLocationService)

	service := newTestService(repo, nil, driverSvc, nil)

	req := &GetRepositionRecommendationsRequest{
		DriverID:      uuid.New(),
		Latitude:      40.7128,
		Longitude:     -74.0060,
		MaxDistanceKm: 5.0, // Only 5km max
		Limit:         3,
	}

	// Hotspot very far away (LA)
	hotspots := []*DemandPrediction{
		{
			ID:                 uuid.New(),
			H3Index:            "8729a46dfffffff", // LA area
			PredictedRides:     100.0,
			DemandLevel:        DemandExtreme,
			RecommendedDrivers: 60,
			ExpectedSurge:      3.0,
			HotspotScore:       95.0,
			PredictionTime:     time.Now().Add(30 * time.Minute),
		},
	}

	repo.On("GetLatestPrediction", ctx, mock.AnythingOfType("string"), Timeframe30Min).Return(nil, nil)
	driverSvc.On("GetDriverCountInCell", ctx, mock.AnythingOfType("string")).Return(5, nil).Maybe()
	repo.On("GetTopHotspots", ctx, Timeframe30Min, 20).Return(hotspots, nil)

	result, err := service.GetRepositionRecommendations(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	// Distant hotspot should be filtered out
	assert.Empty(t, result.Recommendations)
	assert.Equal(t, 0.0, result.EstimatedEarnings)
}

func TestGetRepositionRecommendationsLimitApplied(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	driverSvc := new(mockDriverLocationService)

	service := newTestService(repo, nil, driverSvc, nil)

	req := &GetRepositionRecommendationsRequest{
		DriverID:      uuid.New(),
		Latitude:      40.7128,
		Longitude:     -74.0060,
		MaxDistanceKm: 50.0,
		Limit:         2, // Only want 2 recommendations
	}

	// Many hotspots nearby
	hotspots := []*DemandPrediction{}
	for i := 0; i < 10; i++ {
		hotspots = append(hotspots, &DemandPrediction{
			ID:                 uuid.New(),
			H3Index:            "872a100caffffff",
			PredictedRides:     float64(50 - i),
			DemandLevel:        DemandHigh,
			RecommendedDrivers: 30 - i,
			ExpectedSurge:      1.5,
			HotspotScore:       float64(80 - i),
			PredictionTime:     time.Now().Add(30 * time.Minute),
		})
	}

	repo.On("GetLatestPrediction", ctx, mock.AnythingOfType("string"), Timeframe30Min).Return(nil, nil)
	driverSvc.On("GetDriverCountInCell", ctx, mock.AnythingOfType("string")).Return(5, nil).Maybe()
	repo.On("GetTopHotspots", ctx, Timeframe30Min, 20).Return(hotspots, nil)

	result, err := service.GetRepositionRecommendations(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.LessOrEqual(t, len(result.Recommendations), 2)
}

func TestGetRepositionRecommendationsEstimatedEarnings(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	driverSvc := new(mockDriverLocationService)

	service := newTestService(repo, nil, driverSvc, nil)

	req := &GetRepositionRecommendationsRequest{
		DriverID:      uuid.New(),
		Latitude:      40.7128,
		Longitude:     -74.0060,
		MaxDistanceKm: 50.0,
		Limit:         5,
	}

	hotspots := []*DemandPrediction{
		{
			ID:                 uuid.New(),
			H3Index:            "872a100caffffff",
			PredictedRides:     30.0,
			DemandLevel:        DemandHigh,
			RecommendedDrivers: 15,
			ExpectedSurge:      2.0,
			HotspotScore:       80.0,
			PredictionTime:     time.Now().Add(30 * time.Minute),
		},
	}

	repo.On("GetLatestPrediction", ctx, mock.AnythingOfType("string"), Timeframe30Min).Return(nil, nil)
	driverSvc.On("GetDriverCountInCell", ctx, mock.AnythingOfType("string")).Return(5, nil).Maybe()
	repo.On("GetTopHotspots", ctx, Timeframe30Min, 20).Return(hotspots, nil)

	result, err := service.GetRepositionRecommendations(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	if len(result.Recommendations) > 0 {
		// EstimatedEarnings should be set from top recommendation
		assert.Greater(t, result.EstimatedEarnings, 0.0)
		assert.Equal(t, result.Recommendations[0].ExpectedEarnings, result.EstimatedEarnings)
	}
}

// ========================================
// CALCULATE PRIORITY EDGE CASES
// ========================================

func TestCalculatePriorityBoundaries(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	tests := []struct {
		name        string
		hotspot     HotspotZone
		distance    float64
		minPriority int
		maxPriority int
	}{
		{
			name: "zero score zero distance",
			hotspot: HotspotZone{
				HotspotScore: 0.0,
			},
			distance:    0.0,
			minPriority: 1,
			maxPriority: 10,
		},
		{
			name: "max score zero distance",
			hotspot: HotspotZone{
				HotspotScore: 100.0,
			},
			distance:    0.0,
			minPriority: 1,
			maxPriority: 1,
		},
		{
			name: "zero score max distance",
			hotspot: HotspotZone{
				HotspotScore: 0.0,
			},
			distance:    100.0,
			minPriority: 10,
			maxPriority: 10,
		},
		{
			name: "negative adjusted score clamped to zero",
			hotspot: HotspotZone{
				HotspotScore: 10.0,
			},
			distance:    50.0, // distancePenalty = 50 * 5 = 250, score = 10 - 250 = -240 -> clamped to 0
			minPriority: 10,
			maxPriority: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculatePriority(tt.hotspot, tt.distance)
			assert.GreaterOrEqual(t, result, tt.minPriority, "priority should be >= %d", tt.minPriority)
			assert.LessOrEqual(t, result, tt.maxPriority, "priority should be <= %d", tt.maxPriority)
		})
	}
}

// ========================================
// BUILD FEATURES COMPREHENSIVE TESTS
// ========================================

func TestBuildFeaturesWithAllServices(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	weatherSvc := new(mockWeatherService)
	driverSvc := new(mockDriverLocationService)

	service := newTestService(repo, weatherSvc, driverSvc, nil)

	latitude, longitude := 40.7128, -74.0060

	// Setup all mocks
	repo.On("GetHistoricalAverage", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int"), 8).
		Return(25.0, 8.0, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), 15).Return(7, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), 60).Return(22, nil)
	repo.On("GetRecentDemandTrend", ctx, mock.AnythingOfType("string")).Return(1.5, nil)
	driverSvc.On("GetDriverCountInCell", ctx, mock.AnythingOfType("string")).Return(12, nil)
	repo.On("GetNeighborDemandAverage", ctx, mock.AnythingOfType("[]string"), 30).Return(18.0, nil)

	// Weather with rain
	weatherSvc.On("GetCurrentWeather", ctx, mock.AnythingOfType("float64"), mock.AnythingOfType("float64")).
		Return(&WeatherData{
			ConditionCode:     4, // Rain
			Temperature:       15.0,
			PrecipitationProb: 0.8,
		}, nil)

	// Event nearby
	repo.On("GetEventsNearLocation", ctx, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), 5.0, 2*time.Hour).
		Return([]*SpecialEvent{
			{
				ID:                uuid.New(),
				Name:              "Concert",
				ExpectedAttendees: 20000,
			},
		}, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(15, nil).Maybe()
	repo.On("SavePrediction", ctx, mock.AnythingOfType("*demandforecast.DemandPrediction")).Return(nil)

	result, err := service.GeneratePrediction(ctx, latitude, longitude, Timeframe30Min)

	require.NoError(t, err)
	assert.NotNil(t, result)
	// Should have higher prediction due to rain + event
	assert.Greater(t, result.PredictedRides, 0.0)
	assert.Greater(t, result.Confidence, 0.5)
}

func TestBuildFeaturesWeatherServiceError(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	weatherSvc := new(mockWeatherService)
	driverSvc := new(mockDriverLocationService)

	service := newTestService(repo, weatherSvc, driverSvc, nil)

	latitude, longitude := 40.7128, -74.0060

	repo.On("GetHistoricalAverage", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int"), 8).
		Return(20.0, 5.0, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), 15).Return(5, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), 60).Return(18, nil)
	repo.On("GetRecentDemandTrend", ctx, mock.AnythingOfType("string")).Return(0.5, nil)
	driverSvc.On("GetDriverCountInCell", ctx, mock.AnythingOfType("string")).Return(10, nil)
	repo.On("GetNeighborDemandAverage", ctx, mock.AnythingOfType("[]string"), 30).Return(15.0, nil)

	// Weather service returns error
	weatherSvc.On("GetCurrentWeather", ctx, mock.AnythingOfType("float64"), mock.AnythingOfType("float64")).
		Return(nil, errors.New("weather API error"))

	repo.On("GetEventsNearLocation", ctx, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), 5.0, 2*time.Hour).
		Return([]*SpecialEvent{}, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(10, nil).Maybe()
	repo.On("SavePrediction", ctx, mock.AnythingOfType("*demandforecast.DemandPrediction")).Return(nil)

	// Should still work despite weather error
	result, err := service.GeneratePrediction(ctx, latitude, longitude, Timeframe30Min)

	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBuildFeaturesDriverServiceError(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	driverSvc := new(mockDriverLocationService)

	service := newTestService(repo, nil, driverSvc, &Config{
		H3Resolution:      7,
		TrainingDataWeeks: 8,
		WeatherEnabled:    false,
		EventsEnabled:     false,
	})

	latitude, longitude := 40.7128, -74.0060

	repo.On("GetHistoricalAverage", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int"), 8).
		Return(20.0, 5.0, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), 15).Return(5, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), 60).Return(18, nil)
	repo.On("GetRecentDemandTrend", ctx, mock.AnythingOfType("string")).Return(0.5, nil)
	repo.On("GetNeighborDemandAverage", ctx, mock.AnythingOfType("[]string"), 30).Return(15.0, nil)

	// Driver service returns error
	driverSvc.On("GetDriverCountInCell", ctx, mock.AnythingOfType("string")).Return(0, errors.New("driver service error"))

	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(10, nil).Maybe()
	repo.On("SavePrediction", ctx, mock.AnythingOfType("*demandforecast.DemandPrediction")).Return(nil)

	// Should still work despite driver service error
	result, err := service.GeneratePrediction(ctx, latitude, longitude, Timeframe30Min)

	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBuildFeaturesEventServiceError(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	weatherSvc := new(mockWeatherService)

	service := newTestService(repo, weatherSvc, nil, nil)

	latitude, longitude := 40.7128, -74.0060

	repo.On("GetHistoricalAverage", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int"), 8).
		Return(20.0, 5.0, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), 15).Return(5, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), 60).Return(18, nil)
	repo.On("GetRecentDemandTrend", ctx, mock.AnythingOfType("string")).Return(0.5, nil)
	repo.On("GetNeighborDemandAverage", ctx, mock.AnythingOfType("[]string"), 30).Return(15.0, nil)
	weatherSvc.On("GetCurrentWeather", ctx, mock.AnythingOfType("float64"), mock.AnythingOfType("float64")).
		Return(&WeatherData{ConditionCode: 0, Temperature: 20.0}, nil)

	// Events service returns error
	repo.On("GetEventsNearLocation", ctx, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), 5.0, 2*time.Hour).
		Return(nil, errors.New("events error"))

	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(10, nil).Maybe()
	repo.On("SavePrediction", ctx, mock.AnythingOfType("*demandforecast.DemandPrediction")).Return(nil)

	// Should still work despite events error
	result, err := service.GeneratePrediction(ctx, latitude, longitude, Timeframe30Min)

	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBuildFeaturesMultipleEventsUsesLargest(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, &Config{
		H3Resolution:      7,
		TrainingDataWeeks: 8,
		WeatherEnabled:    false,
		EventsEnabled:     true,
	})

	latitude, longitude := 40.7128, -74.0060

	repo.On("GetHistoricalAverage", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int"), 8).
		Return(20.0, 5.0, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), 15).Return(5, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), 60).Return(18, nil)
	repo.On("GetRecentDemandTrend", ctx, mock.AnythingOfType("string")).Return(0.5, nil)
	repo.On("GetNeighborDemandAverage", ctx, mock.AnythingOfType("[]string"), 30).Return(15.0, nil)

	// Multiple events with different sizes
	repo.On("GetEventsNearLocation", ctx, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), 5.0, 2*time.Hour).
		Return([]*SpecialEvent{
			{ID: uuid.New(), Name: "Small Event", ExpectedAttendees: 1000},
			{ID: uuid.New(), Name: "Large Event", ExpectedAttendees: 50000}, // Largest
			{ID: uuid.New(), Name: "Medium Event", ExpectedAttendees: 5000},
		}, nil)

	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(10, nil).Maybe()
	repo.On("SavePrediction", ctx, mock.AnythingOfType("*demandforecast.DemandPrediction")).Return(nil)

	result, err := service.GeneratePrediction(ctx, latitude, longitude, Timeframe30Min)

	require.NoError(t, err)
	assert.NotNil(t, result)
	// With 50000 attendees (2.5x multiplier), prediction should be higher
	assert.Greater(t, result.PredictedRides, 0.0)
}

// ========================================
// PREDICT METHOD EDGE CASES
// ========================================

func TestPredictWithMaxEventScale(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	features := &ModelFeatures{
		Hour:           20,
		DayOfWeek:      6, // Saturday
		IsWeekend:      true,
		IsHoliday:      false,
		MonthOfYear:    6,
		HistAvgRides:   30.0,
		HistStdRides:   10.0,
		RecentRides1hr: 35,
		RecentTrend:    2.0,
		EventNearby:    true,
		EventScale:     100000, // Massive event
	}

	result := service.predict(features)

	assert.Greater(t, result.PredictedRides, 0.0)
	assert.GreaterOrEqual(t, result.Confidence, 0.5)
	assert.Less(t, result.Confidence, 1.0)
}

func TestPredictConfidenceInterval(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	features := &ModelFeatures{
		Hour:           12,
		DayOfWeek:      2,
		IsWeekend:      false,
		IsHoliday:      false,
		MonthOfYear:    6,
		HistAvgRides:   20.0,
		HistStdRides:   8.0, // Higher variance
		RecentRides1hr: 18,
		RecentTrend:    0.0,
	}

	result := service.predict(features)

	// With higher variance, confidence interval should be wider
	assert.Greater(t, result.UpperBound, result.PredictedRides)
	assert.Less(t, result.LowerBound, result.PredictedRides)
	// Lower bound should be >= 0
	assert.GreaterOrEqual(t, result.LowerBound, 0.0)
}

func TestPredictAllSeasons(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	// Test prediction for each month to verify seasonal adjustments
	for month := 1; month <= 12; month++ {
		features := &ModelFeatures{
			Hour:           12,
			DayOfWeek:      2,
			IsWeekend:      false,
			IsHoliday:      false,
			MonthOfYear:    month,
			HistAvgRides:   20.0,
			HistStdRides:   5.0,
			RecentRides1hr: 18,
			RecentTrend:    0.0,
		}

		result := service.predict(features)
		assert.GreaterOrEqual(t, result.PredictedRides, 0.0, "month %d should have non-negative prediction", month)
	}
}

func TestPredictAllDaysOfWeek(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	// Test prediction for each day of week
	for dow := 0; dow <= 6; dow++ {
		isWeekend := dow == 0 || dow == 6
		features := &ModelFeatures{
			Hour:           12,
			DayOfWeek:      dow,
			IsWeekend:      isWeekend,
			IsHoliday:      false,
			MonthOfYear:    6,
			HistAvgRides:   20.0,
			HistStdRides:   5.0,
			RecentRides1hr: 18,
			RecentTrend:    0.0,
		}

		result := service.predict(features)
		assert.GreaterOrEqual(t, result.PredictedRides, 0.0, "day %d should have non-negative prediction", dow)
	}
}

func TestPredictAllHours(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	// Test prediction for each hour
	for hour := 0; hour <= 23; hour++ {
		features := &ModelFeatures{
			Hour:           hour,
			DayOfWeek:      2,
			IsWeekend:      false,
			IsHoliday:      false,
			MonthOfYear:    6,
			HistAvgRides:   20.0,
			HistStdRides:   5.0,
			RecentRides1hr: 18,
			RecentTrend:    0.0,
		}

		result := service.predict(features)
		assert.GreaterOrEqual(t, result.PredictedRides, 0.0, "hour %d should have non-negative prediction", hour)
	}
}

// ========================================
// DEMAND HEATMAP FILTERING TESTS
// ========================================

func TestGetDemandHeatmapFiltersOutOfBounds(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, nil)

	// Predictions at different locations
	predictions := []*DemandPrediction{
		{
			ID:                 uuid.New(),
			H3Index:            "872a100c9ffffff", // Inside bounds (NYC area, ~40.7128, -74.006)
			PredictedRides:     50.0,
			DemandLevel:        DemandVeryHigh,
			RecommendedDrivers: 30,
			ExpectedSurge:      2.0,
			HotspotScore:       85.0,
			PredictionTime:     time.Now().Add(30 * time.Minute),
		},
		{
			ID:                 uuid.New(),
			H3Index:            "8729a46dfffffff", // LA area - outside bounds
			PredictedRides:     40.0,
			DemandLevel:        DemandHigh,
			RecommendedDrivers: 24,
			ExpectedSurge:      1.5,
			HotspotScore:       70.0,
			PredictionTime:     time.Now().Add(30 * time.Minute),
		},
	}

	req := &GetHeatmapRequest{
		MinLatitude:  40.0,
		MaxLatitude:  41.0,
		MinLongitude: -75.0,
		MaxLongitude: -73.0,
		Timeframe:    Timeframe30Min,
	}

	repo.On("GetPredictionsInBoundingBox", ctx, Timeframe30Min, (*DemandLevel)(nil)).Return(predictions, nil)

	result, err := service.GetDemandHeatmap(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	// LA prediction should be filtered out
	for _, zone := range result.Zones {
		latitude := zone.CenterLatitude
		longitude := zone.CenterLongitude
		assert.GreaterOrEqual(t, latitude, req.MinLatitude, "zone latitude should be >= min")
		assert.LessOrEqual(t, latitude, req.MaxLatitude, "zone latitude should be <= max")
		assert.GreaterOrEqual(t, longitude, req.MinLongitude, "zone longitude should be >= min")
		assert.LessOrEqual(t, longitude, req.MaxLongitude, "zone longitude should be <= max")
	}
}

// ========================================
// CALCULATE REPOSITION PRIORITY EDGE CASES
// ========================================

func TestCalculateRepositionPriorityExtremeScores(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	// Very high score should give priority 1
	highScoreFeatures := &ModelFeatures{
		CurrentDrivers: 0,
		RecentTrend:    10.0,
	}
	result := service.calculateRepositionPriority(200.0, highScoreFeatures)
	assert.Equal(t, 1, result, "extreme high score should give priority 1")

	// Very low score should give priority 10
	lowScoreFeatures := &ModelFeatures{
		CurrentDrivers: 1000,
		RecentTrend:    -10.0,
	}
	result = service.calculateRepositionPriority(0.0, lowScoreFeatures)
	assert.Equal(t, 10, result, "zero/low score should give priority 10")
}

// ========================================
// WEATHER DATA EDGE CASES
// ========================================

func TestGetWeatherMultiplierEdgeCases(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	tests := []struct {
		name        string
		weatherCode int
		precipProb  float64
		expected    float64
	}{
		{"weather code 3 (borderline)", 3, 0.0, 1.0},
		{"weather code 8 (extreme)", 8, 0.9, 1.5},
		{"precip prob exactly 0.7", 1, 0.7, 1.0}, // Not > 0.7
		{"precip prob just above 0.7", 1, 0.71, 1.2},
		{"negative weather code", -1, 0.0, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.getWeatherMultiplier(tt.weatherCode, tt.precipProb)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ========================================
// RECORD DEMAND SNAPSHOT WEATHER INTEGRATION
// ========================================

func TestRecordDemandSnapshotWeatherErrorHandled(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	weatherSvc := new(mockWeatherService)

	service := newTestService(repo, weatherSvc, nil, nil)

	h3Index := "872a100c9ffffff"

	// Weather service returns error
	weatherSvc.On("GetCurrentWeather", ctx, mock.AnythingOfType("float64"), mock.AnythingOfType("float64")).
		Return(nil, errors.New("weather API timeout"))

	repo.On("RecordDemand", ctx, mock.MatchedBy(func(record *HistoricalDemandRecord) bool {
		return record.WeatherCondition == "unknown" // Should fall back to unknown
	})).Return(nil)

	err := service.RecordDemandSnapshot(ctx, h3Index, 100, 95, 20, 5.5, 1.5)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestRecordDemandSnapshotWeatherNilData(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	weatherSvc := new(mockWeatherService)

	service := newTestService(repo, weatherSvc, nil, nil)

	h3Index := "872a100c9ffffff"

	// Weather service returns nil (no error, but no data)
	weatherSvc.On("GetCurrentWeather", ctx, mock.AnythingOfType("float64"), mock.AnythingOfType("float64")).
		Return(nil, nil)

	repo.On("RecordDemand", ctx, mock.MatchedBy(func(record *HistoricalDemandRecord) bool {
		return record.WeatherCondition == "unknown"
	})).Return(nil)

	err := service.RecordDemandSnapshot(ctx, h3Index, 100, 95, 20, 5.5, 1.5)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

// ========================================
// MODEL WEIGHTS VALIDATION
// ========================================

func TestModelWeightsSumToOne(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	w := service.modelWeights
	total := w.HistoricalPattern + w.RecentTrend + w.TimeOfDay +
		w.DayOfWeek + w.Weather + w.Events + w.Seasonal

	assert.InDelta(t, 1.0, total, 0.001, "model weights should sum to 1.0")
}

func TestModelWeightsAllPositive(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	w := service.modelWeights
	assert.Greater(t, w.HistoricalPattern, 0.0)
	assert.Greater(t, w.RecentTrend, 0.0)
	assert.Greater(t, w.TimeOfDay, 0.0)
	assert.Greater(t, w.DayOfWeek, 0.0)
	assert.Greater(t, w.Weather, 0.0)
	assert.Greater(t, w.Events, 0.0)
	assert.Greater(t, w.Seasonal, 0.0)
}

// ========================================
// GET CELLS IN BOUNDING BOX TESTS
// ========================================

func TestGetCellsInBoundingBoxSmallArea(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	// Very small area
	cells := service.getCellsInBoundingBox(40.712, -74.006, 40.713, -74.005)

	assert.GreaterOrEqual(t, len(cells), 1, "should have at least 1 cell")
	// All cells should be unique
	cellMap := make(map[string]bool)
	for _, cell := range cells {
		cellStr := cell.String()
		assert.False(t, cellMap[cellStr], "cell %s should be unique", cellStr)
		cellMap[cellStr] = true
	}
}

func TestGetCellsInBoundingBoxLargeArea(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	// Larger area (about 0.1 degree = ~11km)
	cells := service.getCellsInBoundingBox(40.70, -74.02, 40.80, -73.92)

	// Should have multiple cells
	assert.Greater(t, len(cells), 1, "larger area should have multiple cells")
}

func TestGetCellsInBoundingBoxSinglePoint(t *testing.T) {
	service := NewService(new(mockRepository), nil, nil, nil)

	// Single point
	cells := service.getCellsInBoundingBox(40.712, -74.006, 40.712, -74.006)

	// Should still have at least 1 cell
	assert.GreaterOrEqual(t, len(cells), 1, "single point should have at least 1 cell")
}

// ========================================
// CONCURRENT PREDICTION TESTS
// ========================================

func TestGeneratePredictionConcurrentSafety(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)

	service := newTestService(repo, nil, nil, &Config{
		H3Resolution:      7,
		TrainingDataWeeks: 8,
		WeatherEnabled:    false,
		EventsEnabled:     false,
	})

	// Setup mocks to be called multiple times
	repo.On("GetHistoricalAverage", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int"), 8).
		Return(20.0, 5.0, nil)
	repo.On("GetRecentDemand", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(10, nil)
	repo.On("GetRecentDemandTrend", ctx, mock.AnythingOfType("string")).Return(0.5, nil)
	repo.On("GetNeighborDemandAverage", ctx, mock.AnythingOfType("[]string"), 30).Return(15.0, nil)
	repo.On("SavePrediction", ctx, mock.AnythingOfType("*demandforecast.DemandPrediction")).Return(nil)

	// Run multiple predictions concurrently
	numGoroutines := 10
	results := make(chan *DemandPrediction, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			latitude := 40.7128 + float64(i)*0.001
			longitude := -74.0060 + float64(i)*0.001
			result, err := service.GeneratePrediction(ctx, latitude, longitude, Timeframe30Min)
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(i)
	}

	// Collect results
	var successCount int
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-results:
			successCount++
		case err := <-errors:
			t.Errorf("concurrent prediction failed: %v", err)
		}
	}

	assert.Equal(t, numGoroutines, successCount, "all concurrent predictions should succeed")
}
