package demandforecast

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
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// Mock Implementations
// ============================================================================

// MockRepository implements RepositoryInterface for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) RecordDemand(ctx context.Context, record *HistoricalDemandRecord) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}

func (m *MockRepository) GetHistoricalDemand(ctx context.Context, h3Index string, startTime, endTime time.Time) ([]*HistoricalDemandRecord, error) {
	args := m.Called(ctx, h3Index, startTime, endTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*HistoricalDemandRecord), args.Error(1)
}

func (m *MockRepository) GetHistoricalAverage(ctx context.Context, h3Index string, hour, dayOfWeek, weeksBack int) (float64, float64, error) {
	args := m.Called(ctx, h3Index, hour, dayOfWeek, weeksBack)
	return args.Get(0).(float64), args.Get(1).(float64), args.Error(2)
}

func (m *MockRepository) GetRecentDemand(ctx context.Context, h3Index string, minutesBack int) (int, error) {
	args := m.Called(ctx, h3Index, minutesBack)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) GetRecentDemandTrend(ctx context.Context, h3Index string) (float64, error) {
	args := m.Called(ctx, h3Index)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockRepository) GetNeighborDemandAverage(ctx context.Context, neighborIndexes []string, minutesBack int) (float64, error) {
	args := m.Called(ctx, neighborIndexes, minutesBack)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockRepository) SavePrediction(ctx context.Context, pred *DemandPrediction) error {
	args := m.Called(ctx, pred)
	return args.Error(0)
}

func (m *MockRepository) GetPrediction(ctx context.Context, h3Index string, predictionTime time.Time, timeframe PredictionTimeframe) (*DemandPrediction, error) {
	args := m.Called(ctx, h3Index, predictionTime, timeframe)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DemandPrediction), args.Error(1)
}

func (m *MockRepository) GetLatestPrediction(ctx context.Context, h3Index string, timeframe PredictionTimeframe) (*DemandPrediction, error) {
	args := m.Called(ctx, h3Index, timeframe)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DemandPrediction), args.Error(1)
}

func (m *MockRepository) GetPredictionsInBoundingBox(ctx context.Context, timeframe PredictionTimeframe, minDemandLevel *DemandLevel) ([]*DemandPrediction, error) {
	args := m.Called(ctx, timeframe, minDemandLevel)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DemandPrediction), args.Error(1)
}

func (m *MockRepository) GetTopHotspots(ctx context.Context, timeframe PredictionTimeframe, limit int) ([]*DemandPrediction, error) {
	args := m.Called(ctx, timeframe, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DemandPrediction), args.Error(1)
}

func (m *MockRepository) CreateEvent(ctx context.Context, event *SpecialEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockRepository) GetUpcomingEvents(ctx context.Context, startTime, endTime time.Time) ([]*SpecialEvent, error) {
	args := m.Called(ctx, startTime, endTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*SpecialEvent), args.Error(1)
}

func (m *MockRepository) GetEventsNearLocation(ctx context.Context, lat, lng, radiusKm float64, timeWindow time.Duration) ([]*SpecialEvent, error) {
	args := m.Called(ctx, lat, lng, radiusKm, timeWindow)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*SpecialEvent), args.Error(1)
}

func (m *MockRepository) RecordPredictionAccuracy(ctx context.Context, predictionID uuid.UUID, actualRides int) error {
	args := m.Called(ctx, predictionID, actualRides)
	return args.Error(0)
}

func (m *MockRepository) GetAccuracyMetrics(ctx context.Context, timeframe PredictionTimeframe, daysBack int) (*ForecastAccuracyMetrics, error) {
	args := m.Called(ctx, timeframe, daysBack)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ForecastAccuracyMetrics), args.Error(1)
}

func (m *MockRepository) CleanupOldPredictions(ctx context.Context, daysOld int) (int64, error) {
	args := m.Called(ctx, daysOld)
	return args.Get(0).(int64), args.Error(1)
}

// MockWeatherService implements WeatherService for testing
type MockWeatherService struct {
	mock.Mock
}

func (m *MockWeatherService) GetCurrentWeather(ctx context.Context, lat, lng float64) (*WeatherData, error) {
	args := m.Called(ctx, lat, lng)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*WeatherData), args.Error(1)
}

func (m *MockWeatherService) GetForecast(ctx context.Context, lat, lng float64, hours int) ([]WeatherData, error) {
	args := m.Called(ctx, lat, lng, hours)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]WeatherData), args.Error(1)
}

// MockDriverLocationService implements DriverLocationService for testing
type MockDriverLocationService struct {
	mock.Mock
}

func (m *MockDriverLocationService) GetDriverCountInCell(ctx context.Context, h3Index string) (int, error) {
	args := m.Called(ctx, h3Index)
	return args.Int(0), args.Error(1)
}

func (m *MockDriverLocationService) GetNearbyDriverCount(ctx context.Context, lat, lng, radiusKm float64) (int, error) {
	args := m.Called(ctx, lat, lng, radiusKm)
	return args.Int(0), args.Error(1)
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

func setUserContext(c *gin.Context, userID uuid.UUID, role models.UserRole) {
	c.Set("user_id", userID)
	c.Set("user_role", role)
	c.Set("user_email", "test@example.com")
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

func createTestHandler(mockRepo *MockRepository) *Handler {
	config := DefaultConfig()
	config.WeatherEnabled = false // Disable weather for tests
	config.EventsEnabled = false  // Disable events for tests
	service := NewService(mockRepo, nil, nil, config)
	return NewHandler(service)
}

func createTestPrediction() *DemandPrediction {
	return &DemandPrediction{
		ID:                 uuid.New(),
		H3Index:            "872a10b1affffff",
		PredictionTime:     time.Now().Add(30 * time.Minute),
		GeneratedAt:        time.Now(),
		Timeframe:          Timeframe30Min,
		PredictedRides:     15.5,
		DemandLevel:        DemandHigh,
		Confidence:         0.85,
		RecommendedDrivers: 8,
		ExpectedSurge:      1.5,
		HotspotScore:       75.0,
		RepositionPriority: 2,
	}
}

func createTestEvent() *SpecialEvent {
	return &SpecialEvent{
		ID:                uuid.New(),
		Name:              "Concert",
		EventType:         "concert",
		Latitude:          37.7749,
		Longitude:         -122.4194,
		H3Index:           "872a10b1affffff",
		StartTime:         time.Now().Add(1 * time.Hour),
		EndTime:           time.Now().Add(4 * time.Hour),
		ExpectedAttendees: 5000,
		ImpactRadius:      5.0,
		DemandMultiplier:  1.5,
		IsRecurring:       false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

// ============================================================================
// GetPrediction Handler Tests
// ============================================================================

func TestHandler_GetPrediction_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := GetPredictionRequest{
		Latitude:  37.7749,
		Longitude: -122.4194,
		Timeframe: Timeframe30Min,
	}

	// Setup mock expectations for GeneratePrediction
	mockRepo.On("GetHistoricalAverage", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(10.0, 2.0, nil)
	mockRepo.On("GetRecentDemand", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(5, nil)
	mockRepo.On("GetRecentDemandTrend", mock.Anything, mock.AnythingOfType("string")).Return(0.5, nil)
	mockRepo.On("GetNeighborDemandAverage", mock.Anything, mock.Anything, mock.AnythingOfType("int")).Return(8.0, nil)
	mockRepo.On("SavePrediction", mock.Anything, mock.AnythingOfType("*demandforecast.DemandPrediction")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/demand/predict", reqBody)

	handler.GetPrediction(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetPrediction_InvalidRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("POST", "/api/v1/demand/predict", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/demand/predict", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.GetPrediction(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_GetPrediction_MissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	// Missing timeframe
	reqBody := map[string]interface{}{
		"latitude":  37.7749,
		"longitude": -122.4194,
	}

	c, w := setupTestContext("POST", "/api/v1/demand/predict", reqBody)

	handler.GetPrediction(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// GetHeatmap Handler Tests
// ============================================================================

func TestHandler_GetHeatmap_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := GetHeatmapRequest{
		MinLatitude:  37.7,
		MaxLatitude:  37.8,
		MinLongitude: -122.5,
		MaxLongitude: -122.4,
		Timeframe:    Timeframe30Min,
	}

	predictions := []*DemandPrediction{createTestPrediction()}
	mockRepo.On("GetPredictionsInBoundingBox", mock.Anything, Timeframe30Min, (*DemandLevel)(nil)).Return(predictions, nil)

	c, w := setupTestContext("POST", "/api/v1/demand/heatmap", reqBody)

	handler.GetHeatmap(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetHeatmap_InvalidRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("POST", "/api/v1/demand/heatmap", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/demand/heatmap", bytes.NewReader([]byte("invalid")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.GetHeatmap(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetHeatmap_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := GetHeatmapRequest{
		MinLatitude:  37.7,
		MaxLatitude:  37.8,
		MinLongitude: -122.5,
		MaxLongitude: -122.4,
		Timeframe:    Timeframe30Min,
	}

	mockRepo.On("GetPredictionsInBoundingBox", mock.Anything, Timeframe30Min, (*DemandLevel)(nil)).Return(nil, errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/demand/heatmap", reqBody)

	handler.GetHeatmap(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetHotspots Handler Tests
// ============================================================================

func TestHandler_GetHotspots_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	predictions := []*DemandPrediction{createTestPrediction()}
	mockRepo.On("GetTopHotspots", mock.Anything, PredictionTimeframe("30min"), 10).Return(predictions, nil)

	c, w := setupTestContext("GET", "/api/v1/demand/hotspots", nil)

	handler.GetHotspots(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetHotspots_WithQueryParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	predictions := []*DemandPrediction{createTestPrediction()}
	mockRepo.On("GetTopHotspots", mock.Anything, PredictionTimeframe("1hour"), 5).Return(predictions, nil)

	c, w := setupTestContext("GET", "/api/v1/demand/hotspots?timeframe=1hour&limit=5", nil)
	c.Request.URL.RawQuery = "timeframe=1hour&limit=5"

	handler.GetHotspots(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "1hour", data["timeframe"])
}

func TestHandler_GetHotspots_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	mockRepo.On("GetTopHotspots", mock.Anything, mock.Anything, mock.AnythingOfType("int")).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/demand/hotspots", nil)

	handler.GetHotspots(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_GetHotspots_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	mockRepo.On("GetTopHotspots", mock.Anything, mock.Anything, mock.AnythingOfType("int")).Return([]*DemandPrediction{}, nil)

	c, w := setupTestContext("GET", "/api/v1/demand/hotspots", nil)

	handler.GetHotspots(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["count"])
}

// ============================================================================
// GetRepositionRecommendations Handler Tests
// ============================================================================

func TestHandler_GetRepositionRecommendations_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"latitude":  37.7749,
		"longitude": -122.4194,
	}

	prediction := createTestPrediction()
	mockRepo.On("GetLatestPrediction", mock.Anything, mock.AnythingOfType("string"), Timeframe30Min).Return(prediction, nil)
	mockRepo.On("GetTopHotspots", mock.Anything, Timeframe30Min, 20).Return([]*DemandPrediction{prediction}, nil)

	c, w := setupTestContext("POST", "/api/v1/demand/reposition", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetRepositionRecommendations(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetRepositionRecommendations_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := map[string]interface{}{
		"latitude":  37.7749,
		"longitude": -122.4194,
	}

	c, w := setupTestContext("POST", "/api/v1/demand/reposition", reqBody)
	// Don't set user context

	handler.GetRepositionRecommendations(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetRepositionRecommendations_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/demand/reposition", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/demand/reposition", bytes.NewReader([]byte("invalid")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleDriver)

	handler.GetRepositionRecommendations(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetRepositionRecommendations_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"latitude":  37.7749,
		"longitude": -122.4194,
	}

	mockRepo.On("GetLatestPrediction", mock.Anything, mock.AnythingOfType("string"), Timeframe30Min).Return(nil, errors.New("not found"))
	mockRepo.On("GetTopHotspots", mock.Anything, Timeframe30Min, 20).Return(nil, errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/demand/reposition", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetRepositionRecommendations(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// CreateEvent Handler Tests
// ============================================================================

func TestHandler_CreateEvent_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	startTime := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	endTime := time.Now().Add(4 * time.Hour).Format(time.RFC3339)

	reqBody := CreateEventRequest{
		Name:              "Concert",
		EventType:         "concert",
		Latitude:          37.7749,
		Longitude:         -122.4194,
		StartTime:         startTime,
		EndTime:           endTime,
		ExpectedAttendees: 5000,
		ImpactRadius:      5.0,
		IsRecurring:       false,
	}

	mockRepo.On("CreateEvent", mock.Anything, mock.AnythingOfType("*demandforecast.SpecialEvent")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/demand/events", reqBody)

	handler.CreateEvent(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_CreateEvent_InvalidRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("POST", "/api/v1/admin/demand/events", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/demand/events", bytes.NewReader([]byte("invalid")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateEvent(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateEvent_InvalidStartTime(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := CreateEventRequest{
		Name:              "Concert",
		EventType:         "concert",
		Latitude:          37.7749,
		Longitude:         -122.4194,
		StartTime:         "invalid-time-format",
		EndTime:           time.Now().Add(4 * time.Hour).Format(time.RFC3339),
		ExpectedAttendees: 5000,
	}

	c, w := setupTestContext("POST", "/api/v1/admin/demand/events", reqBody)

	handler.CreateEvent(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateEvent_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	startTime := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	endTime := time.Now().Add(4 * time.Hour).Format(time.RFC3339)

	reqBody := CreateEventRequest{
		Name:              "Concert",
		EventType:         "concert",
		Latitude:          37.7749,
		Longitude:         -122.4194,
		StartTime:         startTime,
		EndTime:           endTime,
		ExpectedAttendees: 5000,
	}

	mockRepo.On("CreateEvent", mock.Anything, mock.AnythingOfType("*demandforecast.SpecialEvent")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/admin/demand/events", reqBody)

	handler.CreateEvent(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// ListUpcomingEvents Handler Tests
// ============================================================================

func TestHandler_ListUpcomingEvents_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	events := []*SpecialEvent{createTestEvent()}
	mockRepo.On("GetUpcomingEvents", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(events, nil)

	c, w := setupTestContext("GET", "/api/v1/demand/events", nil)

	handler.ListUpcomingEvents(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_ListUpcomingEvents_WithHoursParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	events := []*SpecialEvent{}
	mockRepo.On("GetUpcomingEvents", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(events, nil)

	c, w := setupTestContext("GET", "/api/v1/demand/events?hours=48", nil)
	c.Request.URL.RawQuery = "hours=48"

	handler.ListUpcomingEvents(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["count"])
}

func TestHandler_ListUpcomingEvents_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	mockRepo.On("GetUpcomingEvents", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/demand/events", nil)

	handler.ListUpcomingEvents(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetModelAccuracy Handler Tests
// ============================================================================

func TestHandler_GetModelAccuracy_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	metrics := &ForecastAccuracyMetrics{
		Timeframe:        Timeframe30Min,
		MAE:              2.5,
		RMSE:             3.0,
		MAPE:             0.15,
		R2Score:          0.85,
		DirectionAcc:     0.80,
		SamplesEvaluated: 1000,
		EvaluationPeriod: "last_7_days",
		UpdatedAt:        time.Now(),
	}
	mockRepo.On("GetAccuracyMetrics", mock.Anything, PredictionTimeframe("30min"), 7).Return(metrics, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/demand/accuracy", nil)

	handler.GetModelAccuracy(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetModelAccuracy_WithQueryParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	metrics := &ForecastAccuracyMetrics{
		Timeframe:        Timeframe1Hour,
		MAE:              3.5,
		RMSE:             4.0,
		MAPE:             0.20,
		R2Score:          0.80,
		DirectionAcc:     0.75,
		SamplesEvaluated: 500,
		EvaluationPeriod: "last_14_days",
		UpdatedAt:        time.Now(),
	}
	mockRepo.On("GetAccuracyMetrics", mock.Anything, PredictionTimeframe("1hour"), 14).Return(metrics, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/demand/accuracy?timeframe=1hour&days=14", nil)
	c.Request.URL.RawQuery = "timeframe=1hour&days=14"

	handler.GetModelAccuracy(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetModelAccuracy_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	mockRepo.On("GetAccuracyMetrics", mock.Anything, mock.Anything, mock.AnythingOfType("int")).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/demand/accuracy", nil)

	handler.GetModelAccuracy(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// RecordDemandSnapshot Handler Tests
// ============================================================================

func TestHandler_RecordDemandSnapshot_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := map[string]interface{}{
		"h3_index":          "872a10b1affffff",
		"ride_requests":     10,
		"completed_rides":   8,
		"available_drivers": 5,
		"avg_wait_time":     4.5,
		"surge_multiplier":  1.2,
	}

	mockRepo.On("RecordDemand", mock.Anything, mock.AnythingOfType("*demandforecast.HistoricalDemandRecord")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/internal/demand/record", reqBody)

	handler.RecordDemandSnapshot(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_RecordDemandSnapshot_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	// Missing required h3_index
	reqBody := map[string]interface{}{
		"ride_requests": 10,
	}

	c, w := setupTestContext("POST", "/api/v1/internal/demand/record", reqBody)

	handler.RecordDemandSnapshot(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RecordDemandSnapshot_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := map[string]interface{}{
		"h3_index":          "872a10b1affffff",
		"ride_requests":     10,
		"completed_rides":   8,
		"available_drivers": 5,
		"avg_wait_time":     4.5,
		"surge_multiplier":  1.2,
	}

	mockRepo.On("RecordDemand", mock.Anything, mock.AnythingOfType("*demandforecast.HistoricalDemandRecord")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/internal/demand/record", reqBody)

	handler.RecordDemandSnapshot(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_GetPrediction_ValidationCases(t *testing.T) {
	tests := []struct {
		name           string
		reqBody        interface{}
		expectedStatus int
	}{
		{
			name: "valid request",
			reqBody: GetPredictionRequest{
				Latitude:  37.7749,
				Longitude: -122.4194,
				Timeframe: Timeframe30Min,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "empty body",
			reqBody:        nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing latitude",
			reqBody: map[string]interface{}{
				"longitude": -122.4194,
				"timeframe": "30min",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing longitude",
			reqBody: map[string]interface{}{
				"latitude":  37.7749,
				"timeframe": "30min",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			if tt.expectedStatus == http.StatusOK {
				mockRepo.On("GetHistoricalAverage", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(10.0, 2.0, nil)
				mockRepo.On("GetRecentDemand", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("int")).Return(5, nil)
				mockRepo.On("GetRecentDemandTrend", mock.Anything, mock.AnythingOfType("string")).Return(0.5, nil)
				mockRepo.On("GetNeighborDemandAverage", mock.Anything, mock.Anything, mock.AnythingOfType("int")).Return(8.0, nil)
				mockRepo.On("SavePrediction", mock.Anything, mock.AnythingOfType("*demandforecast.DemandPrediction")).Return(nil)
			}

			c, w := setupTestContext("POST", "/api/v1/demand/predict", tt.reqBody)
			if tt.reqBody == nil {
				c.Request = httptest.NewRequest("POST", "/api/v1/demand/predict", nil)
				c.Request.Header.Set("Content-Type", "application/json")
			}

			handler.GetPrediction(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_TimeframeParsing(t *testing.T) {
	tests := []struct {
		name              string
		queryTimeframe    string
		expectedTimeframe PredictionTimeframe
	}{
		{
			name:              "default 30min",
			queryTimeframe:    "",
			expectedTimeframe: "30min",
		},
		{
			name:              "15min timeframe",
			queryTimeframe:    "15min",
			expectedTimeframe: "15min",
		},
		{
			name:              "1hour timeframe",
			queryTimeframe:    "1hour",
			expectedTimeframe: "1hour",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			mockRepo.On("GetTopHotspots", mock.Anything, tt.expectedTimeframe, 10).Return([]*DemandPrediction{}, nil)

			c, w := setupTestContext("GET", "/api/v1/demand/hotspots", nil)
			if tt.queryTimeframe != "" {
				c.Request.URL.RawQuery = "timeframe=" + tt.queryTimeframe
			}

			handler.GetHotspots(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_GetHotspots_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	predictions := []*DemandPrediction{createTestPrediction()}
	mockRepo.On("GetTopHotspots", mock.Anything, mock.Anything, mock.AnythingOfType("int")).Return(predictions, nil)

	c, w := setupTestContext("GET", "/api/v1/demand/hotspots", nil)

	handler.GetHotspots(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["hotspots"])
	assert.NotNil(t, data["count"])
	assert.NotNil(t, data["timeframe"])
}

func TestHandler_ListUpcomingEvents_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	events := []*SpecialEvent{createTestEvent()}
	mockRepo.On("GetUpcomingEvents", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(events, nil)

	c, w := setupTestContext("GET", "/api/v1/demand/events", nil)

	handler.ListUpcomingEvents(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["events"])
	assert.NotNil(t, data["count"])
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	mockRepo.On("GetTopHotspots", mock.Anything, mock.Anything, mock.AnythingOfType("int")).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/demand/hotspots", nil)

	handler.GetHotspots(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)

	assert.False(t, response["success"].(bool))
	assert.NotNil(t, response["error"])

	errorInfo := response["error"].(map[string]interface{})
	assert.NotNil(t, errorInfo["code"])
	assert.NotNil(t, errorInfo["message"])
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestHandler_GetHotspots_InvalidLimitParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	// Should default to 10 when limit is invalid
	mockRepo.On("GetTopHotspots", mock.Anything, mock.Anything, 0).Return([]*DemandPrediction{}, nil)

	c, w := setupTestContext("GET", "/api/v1/demand/hotspots?limit=invalid", nil)
	c.Request.URL.RawQuery = "limit=invalid"

	handler.GetHotspots(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ListUpcomingEvents_InvalidHoursParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	// Should default to 24 hours when invalid
	mockRepo.On("GetUpcomingEvents", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]*SpecialEvent{}, nil)

	c, w := setupTestContext("GET", "/api/v1/demand/events?hours=invalid", nil)
	c.Request.URL.RawQuery = "hours=invalid"

	handler.ListUpcomingEvents(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetModelAccuracy_InvalidDaysParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	// Should default to 7 days when invalid
	mockRepo.On("GetAccuracyMetrics", mock.Anything, mock.Anything, 0).Return(&ForecastAccuracyMetrics{}, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/demand/accuracy?days=invalid", nil)
	c.Request.URL.RawQuery = "days=invalid"

	handler.GetModelAccuracy(c)

	assert.Equal(t, http.StatusOK, w.Code)
}
