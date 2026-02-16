package mleta

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/richxcame/ride-hailing/test/mocks"
)

type fakeETARepository struct {
	getHistoricalFunc        func(ctx context.Context, pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude float64) (float64, error)
	storePredictionFunc      func(ctx context.Context, prediction *ETAPrediction) error
	getTrainingDataFunc      func(ctx context.Context, limit int) ([]*TrainingDataPoint, error)
	storeModelStatsFunc      func(ctx context.Context, model *ETAModel) error
	getPredictionHistoryFunc func(ctx context.Context, limit int, offset int) ([]*ETAPrediction, error)
	getAccuracyMetricsFunc   func(ctx context.Context, days int) (map[string]interface{}, error)
}

func (f *fakeETARepository) GetHistoricalETAForRoute(ctx context.Context, pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude float64) (float64, error) {
	if f.getHistoricalFunc != nil {
		return f.getHistoricalFunc(ctx, pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude)
	}
	return 0, nil
}

func (f *fakeETARepository) StorePrediction(ctx context.Context, prediction *ETAPrediction) error {
	if f.storePredictionFunc != nil {
		return f.storePredictionFunc(ctx, prediction)
	}
	return nil
}

func (f *fakeETARepository) GetTrainingData(ctx context.Context, limit int) ([]*TrainingDataPoint, error) {
	if f.getTrainingDataFunc != nil {
		return f.getTrainingDataFunc(ctx, limit)
	}
	return nil, nil
}

func (f *fakeETARepository) StoreModelStats(ctx context.Context, model *ETAModel) error {
	if f.storeModelStatsFunc != nil {
		return f.storeModelStatsFunc(ctx, model)
	}
	return nil
}

func (f *fakeETARepository) GetPredictionHistory(ctx context.Context, limit int, offset int) ([]*ETAPrediction, error) {
	if f.getPredictionHistoryFunc != nil {
		return f.getPredictionHistoryFunc(ctx, limit, offset)
	}
	return nil, nil
}

func (f *fakeETARepository) GetAccuracyMetrics(ctx context.Context, days int) (map[string]interface{}, error) {
	if f.getAccuracyMetricsFunc != nil {
		return f.getAccuracyMetricsFunc(ctx, days)
	}
	return map[string]interface{}{}, nil
}

func TestPredictETAUsesFeatureWeightsAndHistoricalData(t *testing.T) {
	ctx := context.Background()
	historicalETA := 18.0
	ch := make(chan struct{})

	repo := &fakeETARepository{
		getHistoricalFunc: func(ctx context.Context, pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude float64) (float64, error) {
			return historicalETA, nil
		},
		storePredictionFunc: func(ctx context.Context, prediction *ETAPrediction) error {
			select {
			case <-ch:
			default:
				close(ch)
			}
			return nil
		},
	}

	redisMock := new(mocks.MockRedisClient)
	redisMock.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return("", assert.AnError)
	redisMock.On("SetWithExpiration", mock.Anything, mock.AnythingOfType("string"), mock.Anything, 24*time.Hour).Return(nil)

	service := NewService(repo, redisMock)
	req := &ETAPredictionRequest{
		PickupLatitude:    37.7749,
		PickupLongitude:    -122.4194,
		DropoffLatitude:   37.8044,
		DropoffLongitude:   -122.2711,
		TrafficLevel: "high",
		Weather:      "rain",
	}

	resp, err := service.PredictETA(ctx, req)
	require.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("StorePrediction was not invoked")
	}

	trafficMultiplier := service.model.TrafficMultipliers["high"]
	hour := time.Now().Hour()
	timeOfDayFactor := service.model.TimeOfDayFactors[hour]
	if timeOfDayFactor == 0 {
		timeOfDayFactor = 1
	}
	dayOfWeekFactor := 1.0
	if time.Now().Weekday() == time.Saturday || time.Now().Weekday() == time.Sunday {
		dayOfWeekFactor = 0.9
	}
	weatherImpact := service.model.WeatherImpact["rain"]
	distance := calculateDistance(req.PickupLatitude, req.PickupLongitude, req.DropoffLatitude, req.DropoffLongitude)
	baseETA := (distance / service.model.BaseSpeed) * 60
	expectedUnrounded := baseETA
	expectedUnrounded *= (service.model.DistanceWeight * 1.0)
	expectedUnrounded *= (1 + (trafficMultiplier-1)*service.model.TrafficWeight)
	expectedUnrounded *= (1 + (timeOfDayFactor-1)*service.model.TimeOfDayWeight)
	expectedUnrounded *= (1 + (dayOfWeekFactor-1)*service.model.DayOfWeekWeight)
	expectedUnrounded *= (1 + (weatherImpact-1)*service.model.WeatherWeight)
	expectedUnrounded = expectedUnrounded*(1-service.model.HistoricalWeight) + historicalETA*service.model.HistoricalWeight
	expectedMinutes := math.Round(expectedUnrounded*10) / 10

	assert.InDelta(t, math.Round(distance*10)/10, resp.Distance, 0.0001)
	assert.InDelta(t, expectedMinutes, resp.EstimatedMinutes, 0.0001)
	assert.Equal(t, int(expectedUnrounded*60), resp.EstimatedSeconds)
	assert.Equal(t, trafficMultiplier, resp.Factors["traffic"])
	assert.Equal(t, weatherImpact, resp.Factors["weather"])
	assert.Equal(t, timeOfDayFactor, resp.Factors["time_of_day"])
	assert.Equal(t, dayOfWeekFactor, resp.Factors["day_of_week"])
	assert.Equal(t, historicalETA, resp.Factors["historical_eta"])
	assert.InDelta(t, baseETA, resp.Factors["base_eta"], 0.0001)
	expectedConfidence := math.Round(1.0*service.model.AccuracyRate*100) / 100
	assert.Equal(t, expectedConfidence, resp.Confidence)

	assert.WithinDuration(t, time.Now().Add(time.Duration(expectedUnrounded*float64(time.Minute))), resp.PredictedArrivalTime, 2*time.Second)

	redisMock.AssertExpectations(t)
}

func TestPredictETAConfidenceDropsWithoutSignals(t *testing.T) {
	ctx := context.Background()
	ch := make(chan struct{})
	repo := &fakeETARepository{
		storePredictionFunc: func(ctx context.Context, prediction *ETAPrediction) error {
			select {
			case <-ch:
			default:
				close(ch)
			}
			return nil
		},
	}

	redisMock := new(mocks.MockRedisClient)
	redisMock.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return("", assert.AnError)
	redisMock.On("SetWithExpiration", mock.Anything, mock.AnythingOfType("string"), mock.Anything, 24*time.Hour).Return(nil)

	service := NewService(repo, redisMock)
	req := &ETAPredictionRequest{
		PickupLatitude:  40.7128,
		PickupLongitude:  -74.0060,
		DropoffLatitude: 40.7306,
		DropoffLongitude: -73.9352,
	}

	resp, err := service.PredictETA(ctx, req)
	require.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("StorePrediction was not invoked")
	}

	assert.Equal(t, 1.0, resp.Factors["traffic"])
	assert.Equal(t, 1.0, resp.Factors["weather"])
	assert.Equal(t, 0.0, resp.Factors["historical_eta"])
	assert.Equal(t, 0.6, resp.Confidence)
	redisMock.AssertExpectations(t)
}

func TestTrainModelUpdatesAccuracyMetrics(t *testing.T) {
	ctx := context.Background()
	repo := &fakeETARepository{
		getHistoricalFunc: func(ctx context.Context, pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude float64) (float64, error) {
			return 0, nil
		},
		storePredictionFunc: func(ctx context.Context, prediction *ETAPrediction) error {
			return nil
		},
	}

	redisMock := new(mocks.MockRedisClient)
	redisMock.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return("", assert.AnError)
	redisMock.On("SetWithExpiration", mock.Anything, mock.AnythingOfType("string"), mock.Anything, 24*time.Hour).Return(nil)

	service := NewService(repo, redisMock)

	pickupLatitude, pickupLongitude := 34.0522, -118.2437
	dropoffLatitude, dropoffLongitude := 34.1400, -118.1500
	distance := calculateDistance(pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude)
	trafficMultiplier := service.model.TrafficMultipliers["medium"]
	hour := time.Now().Hour()
	timeOfDayFactor := service.model.TimeOfDayFactors[hour]
	if timeOfDayFactor == 0 {
		timeOfDayFactor = 1
	}
	dayOfWeekFactor := 1.0
	if time.Now().Weekday() == time.Saturday || time.Now().Weekday() == time.Sunday {
		dayOfWeekFactor = 0.9
	}
	weatherImpact := service.model.WeatherImpact["clear"]
	baseETA := (distance / service.model.BaseSpeed) * 60
	predicted := baseETA
	predicted *= (service.model.DistanceWeight * 1.0)
	predicted *= (1 + (trafficMultiplier-1)*service.model.TrafficWeight)
	predicted *= (1 + (timeOfDayFactor-1)*service.model.TimeOfDayWeight)
	predicted *= (1 + (dayOfWeekFactor-1)*service.model.DayOfWeekWeight)
	predicted *= (1 + (weatherImpact-1)*service.model.WeatherWeight)
	predictedRounded := math.Round(predicted*10) / 10
	actualMinutes := predictedRounded + 5

	dataPoints := make([]*TrainingDataPoint, 120)
	for i := range dataPoints {
		dataPoints[i] = &TrainingDataPoint{
			PickupLatitude:     pickupLatitude,
			PickupLongitude:     pickupLongitude,
			DropoffLatitude:    dropoffLatitude,
			DropoffLongitude:    dropoffLongitude,
			ActualMinutes: actualMinutes,
			Distance:      distance,
			TrafficLevel:  "medium",
			Weather:       "clear",
			TimeOfDay:     hour,
			DayOfWeek:     int(time.Now().Weekday()),
		}
	}

	var storedModel *ETAModel
	repo.getTrainingDataFunc = func(ctx context.Context, limit int) ([]*TrainingDataPoint, error) {
		require.Equal(t, 10000, limit)
		return dataPoints, nil
	}
	repo.storeModelStatsFunc = func(ctx context.Context, model *ETAModel) error {
		storedModel = model
		return nil
	}

	err := service.TrainModel(ctx)
	require.NoError(t, err)

	assert.InDelta(t, 5.0, service.model.MeanAbsoluteError, 0.001)
	expectedAccuracy := 1 - math.Min(5.0/15.0, 0.5)
	assert.InDelta(t, expectedAccuracy, service.model.AccuracyRate, 0.0001)
	require.NotNil(t, storedModel)
	assert.InDelta(t, service.model.MeanAbsoluteError, storedModel.MeanAbsoluteError, 0.0001)
	assert.InDelta(t, service.model.AccuracyRate, storedModel.AccuracyRate, 0.0001)

	redisMock.AssertExpectations(t)
}
