package mleta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// Mock Implementations
// ============================================================================

// MockETARepository implements ETARepository for testing
type MockETARepository struct {
	mock.Mock
}

func (m *MockETARepository) GetHistoricalETAForRoute(ctx context.Context, pickupLat, pickupLng, dropoffLat, dropoffLng float64) (float64, error) {
	args := m.Called(ctx, pickupLat, pickupLng, dropoffLat, dropoffLng)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockETARepository) StorePrediction(ctx context.Context, prediction *ETAPrediction) error {
	args := m.Called(ctx, prediction)
	return args.Error(0)
}

func (m *MockETARepository) GetTrainingData(ctx context.Context, limit int) ([]*TrainingDataPoint, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*TrainingDataPoint), args.Error(1)
}

func (m *MockETARepository) StoreModelStats(ctx context.Context, model *ETAModel) error {
	args := m.Called(ctx, model)
	return args.Error(0)
}

func (m *MockETARepository) GetPredictionHistory(ctx context.Context, limit int, offset int) ([]*ETAPrediction, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ETAPrediction), args.Error(1)
}

func (m *MockETARepository) GetAccuracyMetrics(ctx context.Context, days int) (map[string]interface{}, error) {
	args := m.Called(ctx, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// MockRedisClient implements redis.ClientInterface for testing
type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) GetString(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockRedisClient) SetWithExpiration(ctx context.Context, key string, value interface{}, expiry time.Duration) error {
	args := m.Called(ctx, key, value, expiry)
	return args.Error(0)
}

func (m *MockRedisClient) Delete(ctx context.Context, keys ...string) error {
	args := m.Called(ctx, keys)
	return args.Error(0)
}

func (m *MockRedisClient) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockRedisClient) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	args := m.Called(ctx, key, value)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	args := m.Called(ctx, key, expiration)
	return args.Error(0)
}

func (m *MockRedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *MockRedisClient) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRedisClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockRedisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	args := m.Called(ctx, channel, message)
	return args.Error(0)
}

func (m *MockRedisClient) GeoAdd(ctx context.Context, key string, longitude, latitude float64, member string) error {
	args := m.Called(ctx, key, longitude, latitude, member)
	return args.Error(0)
}

func (m *MockRedisClient) GeoRadius(ctx context.Context, key string, longitude, latitude, radiusKm float64, count int) ([]string, error) {
	args := m.Called(ctx, key, longitude, latitude, radiusKm, count)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockRedisClient) GeoRemove(ctx context.Context, key string, member string) error {
	args := m.Called(ctx, key, member)
	return args.Error(0)
}

func (m *MockRedisClient) SetNX(ctx context.Context, key string, value string, expiry time.Duration) (bool, error) {
	args := m.Called(ctx, key, value, expiry)
	return args.Bool(0), args.Error(1)
}

// ============================================================================
// Helper Functions
// ============================================================================

func setupTestContext(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	c.Request = req

	return c, w
}

func setupTestContextWithQuery(method, path, query string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	c, w := setupTestContext(method, path, body)
	c.Request.URL.RawQuery = query
	return c, w
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

func createTestHandler(mockRepo *MockETARepository, mockRedis *MockRedisClient) *Handler {
	service := NewService(mockRepo, mockRedis)
	return NewHandler(service)
}

func createTestETAPredictionRequest() *ETAPredictionRequest {
	return &ETAPredictionRequest{
		PickupLat:    40.7128,
		PickupLng:    -74.0060,
		DropoffLat:   40.7580,
		DropoffLng:   -73.9855,
		TrafficLevel: "medium",
		Weather:      "clear",
		DriverID:     "driver-123",
		RideTypeID:   1,
	}
}

func createTestPrediction() *ETAPrediction {
	return &ETAPrediction{
		ID:               1,
		PickupLat:        40.7128,
		PickupLng:        -74.0060,
		DropoffLat:       40.7580,
		DropoffLng:       -73.9855,
		PredictedMinutes: 15.5,
		Distance:         8.2,
		TrafficLevel:     "medium",
		Weather:          "clear",
		TimeOfDay:        10,
		DayOfWeek:        1,
		Confidence:       0.85,
		CreatedAt:        time.Now(),
	}
}

// ============================================================================
// PredictETA Handler Tests
// ============================================================================

func TestHandler_PredictETA_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	req := createTestETAPredictionRequest()

	// Mock getting historical ETA (cache miss, then DB, then cache the result)
	mockRedis.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return("", errors.New("not found"))
	mockRedis.On("SetWithExpiration", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)
	mockRepo.On("GetHistoricalETAForRoute", mock.Anything, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64")).Return(float64(0), nil)
	mockRepo.On("StorePrediction", mock.Anything, mock.AnythingOfType("*mleta.ETAPrediction")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/eta/predict", req)

	handler.PredictETA(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["prediction"])
}

func TestHandler_PredictETA_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	c, w := setupTestContext("POST", "/api/v1/eta/predict", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/eta/predict", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.PredictETA(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "Invalid request body")
}

func TestHandler_PredictETA_InvalidCoordinates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	// Missing coordinates
	req := &ETAPredictionRequest{
		PickupLat:  0,
		PickupLng:  0,
		DropoffLat: 0,
		DropoffLng: 0,
	}

	c, w := setupTestContext("POST", "/api/v1/eta/predict", req)

	handler.PredictETA(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "Invalid coordinates")
}

func TestHandler_PredictETA_PartialCoordinates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	// Only pickup coordinates
	req := &ETAPredictionRequest{
		PickupLat:  40.7128,
		PickupLng:  -74.0060,
		DropoffLat: 0,
		DropoffLng: 0,
	}

	c, w := setupTestContext("POST", "/api/v1/eta/predict", req)

	handler.PredictETA(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "Invalid coordinates")
}

// ============================================================================
// BatchPredictETA Handler Tests
// ============================================================================

func TestHandler_BatchPredictETA_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	requests := []*ETAPredictionRequest{
		createTestETAPredictionRequest(),
		{
			PickupLat:    40.7580,
			PickupLng:    -73.9855,
			DropoffLat:   40.6892,
			DropoffLng:   -74.0445,
			TrafficLevel: "low",
			Weather:      "cloudy",
		},
	}

	mockRedis.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return("", errors.New("not found"))
	mockRedis.On("SetWithExpiration", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)
	mockRepo.On("GetHistoricalETAForRoute", mock.Anything, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64")).Return(float64(0), nil)
	mockRepo.On("StorePrediction", mock.Anything, mock.AnythingOfType("*mleta.ETAPrediction")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/eta/batch-predict", requests)

	handler.BatchPredictETA(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["predictions"])
	assert.Equal(t, float64(2), data["count"].(float64))
}

func TestHandler_BatchPredictETA_EmptyRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	requests := []*ETAPredictionRequest{}

	c, w := setupTestContext("POST", "/api/v1/eta/batch-predict", requests)

	handler.BatchPredictETA(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "No requests provided")
}

func TestHandler_BatchPredictETA_TooManyRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	// Create 101 requests (over the limit of 100)
	requests := make([]*ETAPredictionRequest, 101)
	for i := 0; i < 101; i++ {
		requests[i] = createTestETAPredictionRequest()
	}

	c, w := setupTestContext("POST", "/api/v1/eta/batch-predict", requests)

	handler.BatchPredictETA(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "Maximum 100 requests")
}

func TestHandler_BatchPredictETA_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	c, w := setupTestContext("POST", "/api/v1/eta/batch-predict", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/eta/batch-predict", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchPredictETA(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// TriggerModelTraining Handler Tests
// ============================================================================

func TestHandler_TriggerModelTraining_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	// Mock the background goroutine's call to TrainModel which calls GetTrainingData
	mockRepo.On("GetTrainingData", mock.Anything, 10000).Return([]*TrainingDataPoint{}, nil).Maybe()
	mockRepo.On("StoreModelStats", mock.Anything, mock.AnythingOfType("*mleta.ETAModel")).Return(nil).Maybe()

	c, w := setupTestContext("POST", "/api/v1/eta/admin/train", nil)

	handler.TriggerModelTraining(c)

	assert.Equal(t, http.StatusAccepted, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Contains(t, data["message"], "Model training started")

	// Brief delay to let the goroutine run
	time.Sleep(10 * time.Millisecond)
}

// ============================================================================
// GetModelStats Handler Tests
// ============================================================================

func TestHandler_GetModelStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	c, w := setupTestContext("GET", "/api/v1/eta/admin/stats", nil)

	handler.GetModelStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["stats"])
}

// ============================================================================
// GetModelAccuracy Handler Tests
// ============================================================================

func TestHandler_GetModelAccuracy_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	metrics := map[string]interface{}{
		"daily_metrics": []interface{}{},
		"period_days":   30,
	}

	mockRepo.On("GetAccuracyMetrics", mock.Anything, 30).Return(metrics, nil)

	c, w := setupTestContext("GET", "/api/v1/eta/admin/accuracy", nil)

	handler.GetModelAccuracy(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["metrics"])
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetModelAccuracy_WithDaysParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	metrics := map[string]interface{}{
		"daily_metrics": []interface{}{},
		"period_days":   60,
	}

	mockRepo.On("GetAccuracyMetrics", mock.Anything, 60).Return(metrics, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/eta/admin/accuracy", "days=60", nil)

	handler.GetModelAccuracy(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetModelAccuracy_InvalidDaysParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	metrics := map[string]interface{}{
		"daily_metrics": []interface{}{},
		"period_days":   30,
	}

	// Should default to 30 when days is invalid
	mockRepo.On("GetAccuracyMetrics", mock.Anything, 30).Return(metrics, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/eta/admin/accuracy", "days=invalid", nil)

	handler.GetModelAccuracy(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetModelAccuracy_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	mockRepo.On("GetAccuracyMetrics", mock.Anything, 30).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/eta/admin/accuracy", nil)

	handler.GetModelAccuracy(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// TuneHyperparameters Handler Tests
// ============================================================================

func TestHandler_TuneHyperparameters_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	distanceWeight := 0.65
	trafficWeight := 0.20
	params := map[string]interface{}{
		"distance_weight": distanceWeight,
		"traffic_weight":  trafficWeight,
	}

	c, w := setupTestContext("PUT", "/api/v1/eta/admin/hyperparameters", params)

	handler.TuneHyperparameters(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Contains(t, data["message"], "Hyperparameters updated")
}

func TestHandler_TuneHyperparameters_InvalidWeightSum(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	// Set all weights to very low values (sum < 0.8)
	distanceWeight := 0.1
	trafficWeight := 0.1
	timeOfDayWeight := 0.1
	dayOfWeekWeight := 0.1
	weatherWeight := 0.1
	historicalWeight := 0.1

	params := map[string]interface{}{
		"distance_weight":    distanceWeight,
		"traffic_weight":     trafficWeight,
		"time_of_day_weight": timeOfDayWeight,
		"day_of_week_weight": dayOfWeekWeight,
		"weather_weight":     weatherWeight,
		"historical_weight":  historicalWeight,
	}

	c, w := setupTestContext("PUT", "/api/v1/eta/admin/hyperparameters", params)

	handler.TuneHyperparameters(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "Weights must sum to approximately 1.0")
}

func TestHandler_TuneHyperparameters_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	c, w := setupTestContext("PUT", "/api/v1/eta/admin/hyperparameters", nil)
	c.Request = httptest.NewRequest("PUT", "/api/v1/eta/admin/hyperparameters", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.TuneHyperparameters(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// GetPredictionHistory Handler Tests
// ============================================================================

func TestHandler_GetPredictionHistory_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	predictions := []*ETAPrediction{
		createTestPrediction(),
		createTestPrediction(),
	}

	mockRepo.On("GetPredictionHistory", mock.Anything, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(predictions, nil)

	c, w := setupTestContext("GET", "/api/v1/eta/admin/history", nil)

	handler.GetPredictionHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["predictions"])
	assert.Equal(t, float64(2), data["count"].(float64))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetPredictionHistory_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	mockRepo.On("GetPredictionHistory", mock.Anything, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/eta/admin/history", nil)

	handler.GetPredictionHistory(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_GetPredictionHistory_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	mockRepo.On("GetPredictionHistory", mock.Anything, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return([]*ETAPrediction{}, nil)

	c, w := setupTestContext("GET", "/api/v1/eta/admin/history", nil)

	handler.GetPredictionHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["count"].(float64))
}

// ============================================================================
// GetAccuracyTrends Handler Tests
// ============================================================================

func TestHandler_GetAccuracyTrends_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	metrics := map[string]interface{}{
		"daily_metrics": []interface{}{},
		"period_days":   30,
	}

	mockRepo.On("GetAccuracyMetrics", mock.Anything, 30).Return(metrics, nil)

	c, w := setupTestContext("GET", "/api/v1/eta/admin/trends", nil)

	handler.GetAccuracyTrends(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["trends"])
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetAccuracyTrends_WithDaysParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	metrics := map[string]interface{}{
		"daily_metrics": []interface{}{},
		"period_days":   90,
	}

	mockRepo.On("GetAccuracyMetrics", mock.Anything, 90).Return(metrics, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/eta/admin/trends", "days=90", nil)

	handler.GetAccuracyTrends(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetAccuracyTrends_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	mockRepo.On("GetAccuracyMetrics", mock.Anything, 30).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/eta/admin/trends", nil)

	handler.GetAccuracyTrends(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetFeatureImportance Handler Tests
// ============================================================================

func TestHandler_GetFeatureImportance_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	c, w := setupTestContext("GET", "/api/v1/eta/admin/features", nil)

	handler.GetFeatureImportance(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["features"])
	assert.NotNil(t, data["model_version"])
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_PredictETA_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		request        interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "valid request with all fields",
			request: ETAPredictionRequest{
				PickupLat:    40.7128,
				PickupLng:    -74.0060,
				DropoffLat:   40.7580,
				DropoffLng:   -73.9855,
				TrafficLevel: "medium",
				Weather:      "clear",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid request without optional fields",
			request: ETAPredictionRequest{
				PickupLat:  40.7128,
				PickupLng:  -74.0060,
				DropoffLat: 40.7580,
				DropoffLng: -73.9855,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid - zero pickup lat",
			request: ETAPredictionRequest{
				PickupLat:  0,
				PickupLng:  -74.0060,
				DropoffLat: 40.7580,
				DropoffLng: -73.9855,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid coordinates",
		},
		{
			name: "invalid - zero dropoff coordinates",
			request: ETAPredictionRequest{
				PickupLat:  40.7128,
				PickupLng:  -74.0060,
				DropoffLat: 0,
				DropoffLng: 0,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid coordinates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockETARepository)
			mockRedis := new(MockRedisClient)
			handler := createTestHandler(mockRepo, mockRedis)

			if tt.expectedStatus == http.StatusOK {
				mockRedis.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return("", errors.New("not found"))
				mockRedis.On("SetWithExpiration", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)
				mockRepo.On("GetHistoricalETAForRoute", mock.Anything, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64")).Return(float64(0), nil)
				mockRepo.On("StorePrediction", mock.Anything, mock.AnythingOfType("*mleta.ETAPrediction")).Return(nil)
			}

			c, w := setupTestContext("POST", "/api/v1/eta/predict", tt.request)

			handler.PredictETA(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				response := parseResponse(w)
				assert.Contains(t, response["error"].(map[string]interface{})["message"], tt.expectedError)
			}
		})
	}
}

func TestHandler_GetModelAccuracy_DaysParam_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		daysParam    string
		expectedDays int
	}{
		{
			name:         "default 30 days",
			daysParam:    "",
			expectedDays: 30,
		},
		{
			name:         "custom 60 days",
			daysParam:    "days=60",
			expectedDays: 60,
		},
		{
			name:         "max 365 days",
			daysParam:    "days=365",
			expectedDays: 365,
		},
		{
			name:         "over max defaults to 30",
			daysParam:    "days=500",
			expectedDays: 30,
		},
		{
			name:         "negative defaults to 30",
			daysParam:    "days=-10",
			expectedDays: 30,
		},
		{
			name:         "zero defaults to 30",
			daysParam:    "days=0",
			expectedDays: 30,
		},
		{
			name:         "invalid string defaults to 30",
			daysParam:    "days=abc",
			expectedDays: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockETARepository)
			mockRedis := new(MockRedisClient)
			handler := createTestHandler(mockRepo, mockRedis)

			metrics := map[string]interface{}{
				"daily_metrics": []interface{}{},
				"period_days":   tt.expectedDays,
			}

			mockRepo.On("GetAccuracyMetrics", mock.Anything, tt.expectedDays).Return(metrics, nil)

			c, w := setupTestContextWithQuery("GET", "/api/v1/eta/admin/accuracy", tt.daysParam, nil)

			handler.GetModelAccuracy(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_PredictETA_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	mockRedis.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return("", errors.New("not found"))
	mockRedis.On("SetWithExpiration", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)
	mockRepo.On("GetHistoricalETAForRoute", mock.Anything, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64")).Return(float64(0), nil)
	mockRepo.On("StorePrediction", mock.Anything, mock.AnythingOfType("*mleta.ETAPrediction")).Return(nil)

	req := createTestETAPredictionRequest()

	c, w := setupTestContext("POST", "/api/v1/eta/predict", req)

	handler.PredictETA(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	// Verify response structure
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	prediction := data["prediction"].(map[string]interface{})
	assert.NotNil(t, prediction["estimated_minutes"])
	assert.NotNil(t, prediction["estimated_seconds"])
	assert.NotNil(t, prediction["distance_km"])
	assert.NotNil(t, prediction["confidence"])
	assert.NotNil(t, prediction["model_version"])
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	mockRepo.On("GetAccuracyMetrics", mock.Anything, 30).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/eta/admin/accuracy", nil)

	handler.GetModelAccuracy(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)

	// Verify error response structure
	assert.False(t, response["success"].(bool))
	assert.NotNil(t, response["error"])

	errorInfo := response["error"].(map[string]interface{})
	assert.NotNil(t, errorInfo["code"])
	assert.NotNil(t, errorInfo["message"])
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestHandler_BatchPredictETA_ExactlyMaxRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	// Create exactly 100 requests (at the limit)
	requests := make([]*ETAPredictionRequest, 100)
	for i := 0; i < 100; i++ {
		requests[i] = createTestETAPredictionRequest()
	}

	mockRedis.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return("", errors.New("not found"))
	mockRedis.On("SetWithExpiration", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("time.Duration")).Return(nil)
	mockRepo.On("GetHistoricalETAForRoute", mock.Anything, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64")).Return(float64(0), nil)
	mockRepo.On("StorePrediction", mock.Anything, mock.AnythingOfType("*mleta.ETAPrediction")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/eta/batch-predict", requests)

	handler.BatchPredictETA(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(100), data["count"].(float64))
}

func TestHandler_TuneHyperparameters_ValidWeightRange(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	// Set weights that sum to 1.0
	baseSpeed := 40.0
	params := map[string]interface{}{
		"base_speed": baseSpeed,
	}

	c, w := setupTestContext("PUT", "/api/v1/eta/admin/hyperparameters", params)

	handler.TuneHyperparameters(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_TuneHyperparameters_InvalidBaseSpeed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockETARepository)
	mockRedis := new(MockRedisClient)
	handler := createTestHandler(mockRepo, mockRedis)

	// Base speed of 0 or negative should not be applied
	baseSpeed := -10.0
	params := map[string]interface{}{
		"base_speed": baseSpeed,
	}

	c, w := setupTestContext("PUT", "/api/v1/eta/admin/hyperparameters", params)

	handler.TuneHyperparameters(c)

	// Should succeed but not apply the invalid value
	assert.Equal(t, http.StatusOK, w.Code)
}
