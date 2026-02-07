package maps

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
// Mock Service
// ============================================================================

// MockMapsService is a mock implementation of the maps service for handler testing
type MockMapsService struct {
	mock.Mock
}

func (m *MockMapsService) GetRoute(ctx context.Context, req *RouteRequest) (*RouteResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RouteResponse), args.Error(1)
}

func (m *MockMapsService) GetETA(ctx context.Context, req *ETARequest) (*ETAResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ETAResponse), args.Error(1)
}

func (m *MockMapsService) GetTrafficAwareETA(ctx context.Context, origin, destination Coordinate) (*ETAResponse, error) {
	args := m.Called(ctx, origin, destination)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ETAResponse), args.Error(1)
}

func (m *MockMapsService) BatchGetETA(ctx context.Context, requests []*ETARequest) ([]*ETAResponse, error) {
	args := m.Called(ctx, requests)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ETAResponse), args.Error(1)
}

func (m *MockMapsService) GetDistanceMatrix(ctx context.Context, req *DistanceMatrixRequest) (*DistanceMatrixResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DistanceMatrixResponse), args.Error(1)
}

func (m *MockMapsService) GetTrafficFlow(ctx context.Context, req *TrafficFlowRequest) (*TrafficFlowResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TrafficFlowResponse), args.Error(1)
}

func (m *MockMapsService) GetTrafficIncidents(ctx context.Context, req *TrafficIncidentsRequest) (*TrafficIncidentsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TrafficIncidentsResponse), args.Error(1)
}

func (m *MockMapsService) Geocode(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GeocodingResponse), args.Error(1)
}

func (m *MockMapsService) ReverseGeocode(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GeocodingResponse), args.Error(1)
}

func (m *MockMapsService) SearchPlaces(ctx context.Context, req *PlaceSearchRequest) (*PlaceSearchResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PlaceSearchResponse), args.Error(1)
}

func (m *MockMapsService) SnapToRoad(ctx context.Context, req *SnapToRoadRequest) (*SnapToRoadResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SnapToRoadResponse), args.Error(1)
}

func (m *MockMapsService) GetSpeedLimits(ctx context.Context, req *SpeedLimitsRequest) (*SpeedLimitsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SpeedLimitsResponse), args.Error(1)
}

func (m *MockMapsService) OptimizeWaypoints(ctx context.Context, origin Coordinate, waypoints []Coordinate, destination Coordinate) ([]int, error) {
	args := m.Called(ctx, origin, waypoints, destination)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]int), args.Error(1)
}

func (m *MockMapsService) HealthCheck(ctx context.Context) map[Provider]error {
	args := m.Called(ctx)
	return args.Get(0).(map[Provider]error)
}

func (m *MockMapsService) GetPrimaryProvider() Provider {
	args := m.Called()
	return args.Get(0).(Provider)
}

// ============================================================================
// Testable Handler
// ============================================================================

// TestableHandler wraps the mock service for testing
type TestableHandler struct {
	service  *MockMapsService
	fallbacks []MapsProvider
}

func NewTestableHandler(mockService *MockMapsService) *TestableHandler {
	return &TestableHandler{service: mockService}
}

// Implement handler methods that delegate to the mock service

func (h *TestableHandler) GetRoute(c *gin.Context) {
	var req RouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid request: " + err.Error()}})
		return
	}

	if !isValidCoordinate(req.Origin) || !isValidCoordinate(req.Destination) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid coordinates"}})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.GetRoute(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to calculate route"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

func (h *TestableHandler) GetETA(c *gin.Context) {
	var req ETARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid request: " + err.Error()}})
		return
	}

	if !isValidCoordinate(req.Origin) || !isValidCoordinate(req.Destination) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid coordinates"}})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.GetTrafficAwareETA(ctx, req.Origin, req.Destination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to calculate ETA"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

func (h *TestableHandler) BatchGetETA(c *gin.Context) {
	var requests []*ETARequest
	if err := c.ShouldBindJSON(&requests); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid request: " + err.Error()}})
		return
	}

	if len(requests) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "maximum 100 requests per batch"}})
		return
	}

	for i, req := range requests {
		if !isValidCoordinate(req.Origin) || !isValidCoordinate(req.Destination) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid coordinates at index " + string(rune('0'+i))}})
			return
		}
	}

	ctx := c.Request.Context()
	responses, err := h.service.BatchGetETA(ctx, requests)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to calculate ETAs"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": responses})
}

func (h *TestableHandler) GetDistanceMatrix(c *gin.Context) {
	var req DistanceMatrixRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid request: " + err.Error()}})
		return
	}

	if len(req.Origins) == 0 || len(req.Destinations) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "origins and destinations are required"}})
		return
	}

	if len(req.Origins)*len(req.Destinations) > 625 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "maximum 625 elements (25 origins x 25 destinations)"}})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.GetDistanceMatrix(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to calculate distance matrix"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

func (h *TestableHandler) GetTrafficFlow(c *gin.Context) {
	var req TrafficFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid request: " + err.Error()}})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.GetTrafficFlow(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to get traffic data"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

func (h *TestableHandler) GetTrafficIncidents(c *gin.Context) {
	var req TrafficIncidentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid request: " + err.Error()}})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.GetTrafficIncidents(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to get traffic incidents"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

func (h *TestableHandler) Geocode(c *gin.Context) {
	var req GeocodingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid request: " + err.Error()}})
		return
	}

	if req.Address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "address is required"}})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.Geocode(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to geocode address"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

func (h *TestableHandler) ReverseGeocode(c *gin.Context) {
	var req GeocodingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid request: " + err.Error()}})
		return
	}

	if req.Coordinate == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "coordinate is required"}})
		return
	}

	if !isValidCoordinate(*req.Coordinate) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid coordinate"}})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.ReverseGeocode(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to reverse geocode"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

func (h *TestableHandler) SearchPlaces(c *gin.Context) {
	var req PlaceSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid request: " + err.Error()}})
		return
	}

	if req.Query == "" && req.Location == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "query or location is required"}})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.SearchPlaces(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to search places"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

func (h *TestableHandler) SnapToRoad(c *gin.Context) {
	var req SnapToRoadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid request: " + err.Error()}})
		return
	}

	if len(req.Path) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "path is required"}})
		return
	}

	if len(req.Path) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "maximum 100 points per request"}})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.SnapToRoad(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to snap to road"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

func (h *TestableHandler) GetSpeedLimits(c *gin.Context) {
	var req SpeedLimitsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid request: " + err.Error()}})
		return
	}

	if len(req.Path) == 0 && len(req.PlaceIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "path or place_ids is required"}})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.GetSpeedLimits(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to get speed limits"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

func (h *TestableHandler) OptimizeWaypoints(c *gin.Context) {
	var req OptimizeWaypointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "invalid request: " + err.Error()}})
		return
	}

	if len(req.Waypoints) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "waypoints are required"}})
		return
	}

	if len(req.Waypoints) > 23 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": 400, "message": "maximum 23 waypoints"}})
		return
	}

	ctx := c.Request.Context()
	order, err := h.service.OptimizeWaypoints(ctx, req.Origin, req.Waypoints, req.Destination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": 500, "message": "failed to optimize waypoints"}})
		return
	}

	optimizedWaypoints := make([]Coordinate, len(order))
	for i, idx := range order {
		optimizedWaypoints[i] = req.Waypoints[idx]
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": OptimizeWaypointsResponse{
		Order:              order,
		OptimizedWaypoints: optimizedWaypoints,
	}})
}

func (h *TestableHandler) HealthCheck(c *gin.Context) {
	ctx := c.Request.Context()
	results := h.service.HealthCheck(ctx)

	status := "healthy"
	errs := make(map[string]string)

	for provider, err := range results {
		if err != nil {
			status = "degraded"
			errs[string(provider)] = err.Error()
		}
	}

	primaryProvider := h.service.GetPrimaryProvider()
	if err, ok := results[primaryProvider]; ok && err != nil {
		status = "unhealthy"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"status":    status,
			"primary":   primaryProvider,
			"errors":    errs,
			"providers": results,
		},
	})
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

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

func createTestRouteResponse() *RouteResponse {
	return &RouteResponse{
		Routes: []Route{
			{
				DistanceMeters:  10000,
				DistanceKm:      10.0,
				DurationSeconds: 1200,
				DurationMinutes: 20.0,
				Summary:         "Via Main St",
			},
		},
		Provider:    ProviderGoogle,
		RequestedAt: time.Now(),
	}
}

func createTestETAResponse() *ETAResponse {
	return &ETAResponse{
		DistanceKm:          10.0,
		DistanceMeters:      10000,
		DurationMinutes:     20.0,
		DurationSeconds:     1200,
		DurationInTraffic:   25.0,
		TrafficDelayMinutes: 5.0,
		TrafficLevel:        TrafficModerate,
		EstimatedArrival:    time.Now().Add(25 * time.Minute),
		Provider:            ProviderGoogle,
		Confidence:          0.9,
		CacheHit:            false,
	}
}

func createTestDistanceMatrixResponse() *DistanceMatrixResponse {
	return &DistanceMatrixResponse{
		Rows: []DistanceMatrixRow{
			{
				Elements: []DistanceMatrixElement{
					{
						Status:          "OK",
						DistanceKm:      10.0,
						DistanceMeters:  10000,
						DurationMinutes: 20.0,
						DurationSeconds: 1200,
					},
					{
						Status:          "OK",
						DistanceKm:      15.0,
						DistanceMeters:  15000,
						DurationMinutes: 25.0,
						DurationSeconds: 1500,
					},
				},
			},
		},
		Provider:    ProviderGoogle,
		RequestedAt: time.Now(),
	}
}

func createTestTrafficFlowResponse() *TrafficFlowResponse {
	return &TrafficFlowResponse{
		Segments: []TrafficFlowSegment{
			{
				RoadName:         "Main St",
				CurrentSpeedKmh:  35.0,
				FreeFlowSpeedKmh: 50.0,
				TrafficLevel:     TrafficModerate,
				JamFactor:        4.0,
			},
		},
		OverallLevel: TrafficModerate,
		UpdatedAt:    time.Now(),
		Provider:     ProviderGoogle,
	}
}

func createTestTrafficIncidentsResponse() *TrafficIncidentsResponse {
	return &TrafficIncidentsResponse{
		Incidents: []TrafficIncident{
			{
				ID:          "inc-001",
				Type:        "accident",
				Severity:    "major",
				Location:    Coordinate{Latitude: 37.78, Longitude: -122.42},
				Description: "Multi-vehicle accident",
				RoadClosed:  false,
				UpdatedAt:   time.Now(),
			},
		},
		UpdatedAt: time.Now(),
		Provider:  ProviderGoogle,
	}
}

func createTestGeocodingResponse() *GeocodingResponse {
	return &GeocodingResponse{
		Results: []GeocodingResult{
			{
				FormattedAddress: "1600 Amphitheatre Parkway, Mountain View, CA",
				Coordinate:       Coordinate{Latitude: 37.4224, Longitude: -122.0842},
				PlaceID:          "place123",
				Types:            []string{"street_address"},
				Confidence:       0.95,
			},
		},
		Provider: ProviderGoogle,
	}
}

func createTestPlaceSearchResponse() *PlaceSearchResponse {
	rating := 4.5
	totalRatings := 1000
	return &PlaceSearchResponse{
		Results: []Place{
			{
				PlaceID:          "place789",
				Name:             "Coffee Shop",
				FormattedAddress: "123 Main St, San Francisco, CA",
				Coordinate:       Coordinate{Latitude: 37.7749, Longitude: -122.4194},
				Types:            []string{"cafe", "food"},
				Rating:           &rating,
				UserRatingsTotal: &totalRatings,
			},
		},
		Provider: ProviderGoogle,
	}
}

func createTestSnapToRoadResponse() *SnapToRoadResponse {
	return &SnapToRoadResponse{
		SnappedPoints: []SnappedPoint{
			{
				Location:      Coordinate{Latitude: 37.7750, Longitude: -122.4193},
				OriginalIndex: 0,
				RoadName:      "Market St",
			},
		},
		Provider: ProviderGoogle,
	}
}

func createTestSpeedLimitsResponse() *SpeedLimitsResponse {
	return &SpeedLimitsResponse{
		SpeedLimits: []SpeedLimit{
			{
				PlaceID:       "road123",
				SpeedLimitKmh: 50,
				SpeedLimitMph: 31,
			},
		},
		Provider: ProviderGoogle,
	}
}

// ============================================================================
// GetRoute Handler Tests
// ============================================================================

func TestHandler_GetRoute_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestRouteResponse()

	mockService.On("GetRoute", mock.Anything, mock.AnythingOfType("*maps.RouteRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"origin": map[string]float64{
			"latitude":  37.7749,
			"longitude": -122.4194,
		},
		"destination": map[string]float64{
			"latitude":  37.3382,
			"longitude": -121.8863,
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/route", reqBody)

	handler.GetRoute(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetRoute_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/maps/route", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/maps/route", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.GetRoute(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_GetRoute_InvalidOriginCoordinates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"origin": map[string]float64{
			"latitude":  91.0, // Invalid: latitude out of range
			"longitude": -122.4194,
		},
		"destination": map[string]float64{
			"latitude":  37.3382,
			"longitude": -121.8863,
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/route", reqBody)

	handler.GetRoute(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid coordinates")
}

func TestHandler_GetRoute_InvalidDestinationCoordinates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"origin": map[string]float64{
			"latitude":  37.7749,
			"longitude": -122.4194,
		},
		"destination": map[string]float64{
			"latitude":  37.3382,
			"longitude": -181.0, // Invalid: longitude out of range
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/route", reqBody)

	handler.GetRoute(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetRoute_NullIslandOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	// Null Island (0,0) is valid coordinates but often indicates an error
	// The handler should still accept it as valid
	expectedResp := createTestRouteResponse()
	mockService.On("GetRoute", mock.Anything, mock.AnythingOfType("*maps.RouteRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"origin": map[string]float64{
			"latitude":  0.0,
			"longitude": 0.0,
		},
		"destination": map[string]float64{
			"latitude":  37.3382,
			"longitude": -121.8863,
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/route", reqBody)

	handler.GetRoute(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetRoute_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	mockService.On("GetRoute", mock.Anything, mock.AnythingOfType("*maps.RouteRequest")).Return(nil, errors.New("provider unavailable"))

	reqBody := map[string]interface{}{
		"origin": map[string]float64{
			"latitude":  37.7749,
			"longitude": -122.4194,
		},
		"destination": map[string]float64{
			"latitude":  37.3382,
			"longitude": -121.8863,
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/route", reqBody)

	handler.GetRoute(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetRoute_WithWaypoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestRouteResponse()
	mockService.On("GetRoute", mock.Anything, mock.AnythingOfType("*maps.RouteRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"origin": map[string]float64{
			"latitude":  37.7749,
			"longitude": -122.4194,
		},
		"destination": map[string]float64{
			"latitude":  37.3382,
			"longitude": -121.8863,
		},
		"waypoints": []map[string]float64{
			{"latitude": 37.5, "longitude": -122.0},
			{"latitude": 37.4, "longitude": -121.9},
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/route", reqBody)

	handler.GetRoute(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetRoute_WithAvoidOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestRouteResponse()
	mockService.On("GetRoute", mock.Anything, mock.AnythingOfType("*maps.RouteRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"origin": map[string]float64{
			"latitude":  37.7749,
			"longitude": -122.4194,
		},
		"destination": map[string]float64{
			"latitude":  37.3382,
			"longitude": -121.8863,
		},
		"avoid_tolls":    true,
		"avoid_highways": true,
		"avoid_ferries":  false,
	}

	c, w := setupTestContext("POST", "/api/v1/maps/route", reqBody)

	handler.GetRoute(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetETA Handler Tests
// ============================================================================

func TestHandler_GetETA_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestETAResponse()
	mockService.On("GetTrafficAwareETA", mock.Anything, mock.AnythingOfType("maps.Coordinate"), mock.AnythingOfType("maps.Coordinate")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"origin": map[string]float64{
			"latitude":  37.7749,
			"longitude": -122.4194,
		},
		"destination": map[string]float64{
			"latitude":  37.3382,
			"longitude": -121.8863,
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/eta", reqBody)

	handler.GetETA(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetETA_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/maps/eta", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/maps/eta", bytes.NewReader([]byte("invalid")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.GetETA(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetETA_InvalidOriginLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"origin": map[string]float64{
			"latitude":  -91.0, // Invalid
			"longitude": -122.4194,
		},
		"destination": map[string]float64{
			"latitude":  37.3382,
			"longitude": -121.8863,
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/eta", reqBody)

	handler.GetETA(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetETA_InvalidDestinationLongitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"origin": map[string]float64{
			"latitude":  37.7749,
			"longitude": -122.4194,
		},
		"destination": map[string]float64{
			"latitude":  37.3382,
			"longitude": 181.0, // Invalid
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/eta", reqBody)

	handler.GetETA(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetETA_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	mockService.On("GetTrafficAwareETA", mock.Anything, mock.AnythingOfType("maps.Coordinate"), mock.AnythingOfType("maps.Coordinate")).Return(nil, errors.New("service error"))

	reqBody := map[string]interface{}{
		"origin": map[string]float64{
			"latitude":  37.7749,
			"longitude": -122.4194,
		},
		"destination": map[string]float64{
			"latitude":  37.3382,
			"longitude": -121.8863,
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/eta", reqBody)

	handler.GetETA(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetETA_WithTrafficModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestETAResponse()
	mockService.On("GetTrafficAwareETA", mock.Anything, mock.AnythingOfType("maps.Coordinate"), mock.AnythingOfType("maps.Coordinate")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"origin": map[string]float64{
			"latitude":  37.7749,
			"longitude": -122.4194,
		},
		"destination": map[string]float64{
			"latitude":  37.3382,
			"longitude": -121.8863,
		},
		"traffic_model": "pessimistic",
	}

	c, w := setupTestContext("POST", "/api/v1/maps/eta", reqBody)

	handler.GetETA(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// BatchGetETA Handler Tests
// ============================================================================

func TestHandler_BatchGetETA_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := []*ETAResponse{createTestETAResponse(), createTestETAResponse()}
	mockService.On("BatchGetETA", mock.Anything, mock.AnythingOfType("[]*maps.ETARequest")).Return(expectedResp, nil)

	reqBody := []map[string]interface{}{
		{
			"origin":      map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
			"destination": map[string]float64{"latitude": 37.3382, "longitude": -121.8863},
		},
		{
			"origin":      map[string]float64{"latitude": 37.7849, "longitude": -122.4094},
			"destination": map[string]float64{"latitude": 37.4382, "longitude": -121.9863},
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/eta/batch", reqBody)

	handler.BatchGetETA(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_BatchGetETA_ExceedsLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	// Create 101 requests to exceed the limit
	reqBody := make([]map[string]interface{}, 101)
	for i := 0; i < 101; i++ {
		reqBody[i] = map[string]interface{}{
			"origin":      map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
			"destination": map[string]float64{"latitude": 37.3382, "longitude": -121.8863},
		}
	}

	c, w := setupTestContext("POST", "/api/v1/maps/eta/batch", reqBody)

	handler.BatchGetETA(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "maximum 100")
}

func TestHandler_BatchGetETA_InvalidCoordinatesInBatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	reqBody := []map[string]interface{}{
		{
			"origin":      map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
			"destination": map[string]float64{"latitude": 37.3382, "longitude": -121.8863},
		},
		{
			"origin":      map[string]float64{"latitude": 95.0, "longitude": -122.4094}, // Invalid
			"destination": map[string]float64{"latitude": 37.4382, "longitude": -121.9863},
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/eta/batch", reqBody)

	handler.BatchGetETA(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_BatchGetETA_EmptyBatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := []*ETAResponse{}
	mockService.On("BatchGetETA", mock.Anything, mock.AnythingOfType("[]*maps.ETARequest")).Return(expectedResp, nil)

	reqBody := []map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/maps/eta/batch", reqBody)

	handler.BatchGetETA(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_BatchGetETA_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	mockService.On("BatchGetETA", mock.Anything, mock.AnythingOfType("[]*maps.ETARequest")).Return(nil, errors.New("batch processing failed"))

	reqBody := []map[string]interface{}{
		{
			"origin":      map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
			"destination": map[string]float64{"latitude": 37.3382, "longitude": -121.8863},
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/eta/batch", reqBody)

	handler.BatchGetETA(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetDistanceMatrix Handler Tests
// ============================================================================

func TestHandler_GetDistanceMatrix_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestDistanceMatrixResponse()
	mockService.On("GetDistanceMatrix", mock.Anything, mock.AnythingOfType("*maps.DistanceMatrixRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"origins": []map[string]float64{
			{"latitude": 37.7749, "longitude": -122.4194},
		},
		"destinations": []map[string]float64{
			{"latitude": 37.3382, "longitude": -121.8863},
			{"latitude": 37.4382, "longitude": -121.9863},
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/distance-matrix", reqBody)

	handler.GetDistanceMatrix(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetDistanceMatrix_EmptyOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"origins": []map[string]float64{},
		"destinations": []map[string]float64{
			{"latitude": 37.3382, "longitude": -121.8863},
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/distance-matrix", reqBody)

	handler.GetDistanceMatrix(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "origins and destinations are required")
}

func TestHandler_GetDistanceMatrix_EmptyDestinations(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"origins": []map[string]float64{
			{"latitude": 37.7749, "longitude": -122.4194},
		},
		"destinations": []map[string]float64{},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/distance-matrix", reqBody)

	handler.GetDistanceMatrix(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetDistanceMatrix_ExceedsMaxElements(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	// Create 26x25 = 650 elements which exceeds the 625 limit
	origins := make([]map[string]float64, 26)
	for i := 0; i < 26; i++ {
		origins[i] = map[string]float64{"latitude": 37.7749 + float64(i)*0.01, "longitude": -122.4194}
	}
	destinations := make([]map[string]float64, 25)
	for i := 0; i < 25; i++ {
		destinations[i] = map[string]float64{"latitude": 37.3382 + float64(i)*0.01, "longitude": -121.8863}
	}

	reqBody := map[string]interface{}{
		"origins":      origins,
		"destinations": destinations,
	}

	c, w := setupTestContext("POST", "/api/v1/maps/distance-matrix", reqBody)

	handler.GetDistanceMatrix(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "maximum 625 elements")
}

func TestHandler_GetDistanceMatrix_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	mockService.On("GetDistanceMatrix", mock.Anything, mock.AnythingOfType("*maps.DistanceMatrixRequest")).Return(nil, errors.New("service error"))

	reqBody := map[string]interface{}{
		"origins": []map[string]float64{
			{"latitude": 37.7749, "longitude": -122.4194},
		},
		"destinations": []map[string]float64{
			{"latitude": 37.3382, "longitude": -121.8863},
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/distance-matrix", reqBody)

	handler.GetDistanceMatrix(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetTrafficFlow Handler Tests
// ============================================================================

func TestHandler_GetTrafficFlow_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestTrafficFlowResponse()
	mockService.On("GetTrafficFlow", mock.Anything, mock.AnythingOfType("*maps.TrafficFlowRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"location": map[string]float64{
			"latitude":  37.7749,
			"longitude": -122.4194,
		},
		"radius_meters": 5000,
	}

	c, w := setupTestContext("POST", "/api/v1/maps/traffic/flow", reqBody)

	handler.GetTrafficFlow(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetTrafficFlow_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/maps/traffic/flow", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/maps/traffic/flow", bytes.NewReader([]byte("invalid")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.GetTrafficFlow(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetTrafficFlow_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	mockService.On("GetTrafficFlow", mock.Anything, mock.AnythingOfType("*maps.TrafficFlowRequest")).Return(nil, errors.New("traffic data unavailable"))

	reqBody := map[string]interface{}{
		"location": map[string]float64{
			"latitude":  37.7749,
			"longitude": -122.4194,
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/traffic/flow", reqBody)

	handler.GetTrafficFlow(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetTrafficFlow_WithBoundingBox(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestTrafficFlowResponse()
	mockService.On("GetTrafficFlow", mock.Anything, mock.AnythingOfType("*maps.TrafficFlowRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"location": map[string]float64{
			"latitude":  37.7749,
			"longitude": -122.4194,
		},
		"bounding_box": map[string]interface{}{
			"northeast": map[string]float64{"latitude": 37.8, "longitude": -122.3},
			"southwest": map[string]float64{"latitude": 37.7, "longitude": -122.5},
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/traffic/flow", reqBody)

	handler.GetTrafficFlow(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetTrafficIncidents Handler Tests
// ============================================================================

func TestHandler_GetTrafficIncidents_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestTrafficIncidentsResponse()
	mockService.On("GetTrafficIncidents", mock.Anything, mock.AnythingOfType("*maps.TrafficIncidentsRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"location": map[string]float64{
			"latitude":  37.7749,
			"longitude": -122.4194,
		},
		"radius_meters": 10000,
	}

	c, w := setupTestContext("POST", "/api/v1/maps/traffic/incidents", reqBody)

	handler.GetTrafficIncidents(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetTrafficIncidents_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/maps/traffic/incidents", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/maps/traffic/incidents", bytes.NewReader([]byte("{")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.GetTrafficIncidents(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetTrafficIncidents_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	mockService.On("GetTrafficIncidents", mock.Anything, mock.AnythingOfType("*maps.TrafficIncidentsRequest")).Return(nil, errors.New("service unavailable"))

	reqBody := map[string]interface{}{
		"location": map[string]float64{
			"latitude":  37.7749,
			"longitude": -122.4194,
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/traffic/incidents", reqBody)

	handler.GetTrafficIncidents(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetTrafficIncidents_WithTypeFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestTrafficIncidentsResponse()
	mockService.On("GetTrafficIncidents", mock.Anything, mock.AnythingOfType("*maps.TrafficIncidentsRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"location": map[string]float64{
			"latitude":  37.7749,
			"longitude": -122.4194,
		},
		"types": []string{"accident", "construction"},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/traffic/incidents", reqBody)

	handler.GetTrafficIncidents(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// Geocode Handler Tests
// ============================================================================

func TestHandler_Geocode_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestGeocodingResponse()
	mockService.On("Geocode", mock.Anything, mock.AnythingOfType("*maps.GeocodingRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"address": "1600 Amphitheatre Parkway, Mountain View, CA",
	}

	c, w := setupTestContext("POST", "/api/v1/maps/geocode", reqBody)

	handler.Geocode(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_Geocode_MissingAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"language": "en",
	}

	c, w := setupTestContext("POST", "/api/v1/maps/geocode", reqBody)

	handler.Geocode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "address is required")
}

func TestHandler_Geocode_EmptyAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"address": "",
	}

	c, w := setupTestContext("POST", "/api/v1/maps/geocode", reqBody)

	handler.Geocode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Geocode_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	mockService.On("Geocode", mock.Anything, mock.AnythingOfType("*maps.GeocodingRequest")).Return(nil, errors.New("geocoding failed"))

	reqBody := map[string]interface{}{
		"address": "Some address",
	}

	c, w := setupTestContext("POST", "/api/v1/maps/geocode", reqBody)

	handler.Geocode(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_Geocode_WithLanguageAndRegion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestGeocodingResponse()
	mockService.On("Geocode", mock.Anything, mock.AnythingOfType("*maps.GeocodingRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"address":  "1600 Amphitheatre Parkway",
		"language": "en",
		"region":   "US",
	}

	c, w := setupTestContext("POST", "/api/v1/maps/geocode", reqBody)

	handler.Geocode(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// ReverseGeocode Handler Tests
// ============================================================================

func TestHandler_ReverseGeocode_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestGeocodingResponse()
	mockService.On("ReverseGeocode", mock.Anything, mock.AnythingOfType("*maps.GeocodingRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"coordinate": map[string]float64{
			"latitude":  37.4224,
			"longitude": -122.0842,
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/reverse-geocode", reqBody)

	handler.ReverseGeocode(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_ReverseGeocode_MissingCoordinate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"language": "en",
	}

	c, w := setupTestContext("POST", "/api/v1/maps/reverse-geocode", reqBody)

	handler.ReverseGeocode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "coordinate is required")
}

func TestHandler_ReverseGeocode_InvalidCoordinate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"coordinate": map[string]float64{
			"latitude":  91.0, // Invalid
			"longitude": -122.0842,
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/reverse-geocode", reqBody)

	handler.ReverseGeocode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid coordinate")
}

func TestHandler_ReverseGeocode_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	mockService.On("ReverseGeocode", mock.Anything, mock.AnythingOfType("*maps.GeocodingRequest")).Return(nil, errors.New("reverse geocoding failed"))

	reqBody := map[string]interface{}{
		"coordinate": map[string]float64{
			"latitude":  37.4224,
			"longitude": -122.0842,
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/reverse-geocode", reqBody)

	handler.ReverseGeocode(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// SearchPlaces Handler Tests
// ============================================================================

func TestHandler_SearchPlaces_SuccessWithQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestPlaceSearchResponse()
	mockService.On("SearchPlaces", mock.Anything, mock.AnythingOfType("*maps.PlaceSearchRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"query": "coffee shop",
		"location": map[string]float64{
			"latitude":  37.7749,
			"longitude": -122.4194,
		},
		"radius_meters": 1000,
	}

	c, w := setupTestContext("POST", "/api/v1/maps/places/search", reqBody)

	handler.SearchPlaces(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_SearchPlaces_SuccessWithLocationOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestPlaceSearchResponse()
	mockService.On("SearchPlaces", mock.Anything, mock.AnythingOfType("*maps.PlaceSearchRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"location": map[string]float64{
			"latitude":  37.7749,
			"longitude": -122.4194,
		},
		"types": []string{"restaurant"},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/places/search", reqBody)

	handler.SearchPlaces(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SearchPlaces_MissingQueryAndLocation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"radius_meters": 1000,
	}

	c, w := setupTestContext("POST", "/api/v1/maps/places/search", reqBody)

	handler.SearchPlaces(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "query or location is required")
}

func TestHandler_SearchPlaces_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	mockService.On("SearchPlaces", mock.Anything, mock.AnythingOfType("*maps.PlaceSearchRequest")).Return(nil, errors.New("search failed"))

	reqBody := map[string]interface{}{
		"query": "coffee",
	}

	c, w := setupTestContext("POST", "/api/v1/maps/places/search", reqBody)

	handler.SearchPlaces(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SearchPlaces_WithOpenNow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestPlaceSearchResponse()
	mockService.On("SearchPlaces", mock.Anything, mock.AnythingOfType("*maps.PlaceSearchRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"query":    "restaurant",
		"open_now": true,
	}

	c, w := setupTestContext("POST", "/api/v1/maps/places/search", reqBody)

	handler.SearchPlaces(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// SnapToRoad Handler Tests
// ============================================================================

func TestHandler_SnapToRoad_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestSnapToRoadResponse()
	mockService.On("SnapToRoad", mock.Anything, mock.AnythingOfType("*maps.SnapToRoadRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"path": []map[string]float64{
			{"latitude": 37.7749, "longitude": -122.4194},
			{"latitude": 37.7759, "longitude": -122.4184},
		},
		"interpolate": true,
	}

	c, w := setupTestContext("POST", "/api/v1/maps/roads/snap", reqBody)

	handler.SnapToRoad(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_SnapToRoad_EmptyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"path": []map[string]float64{},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/roads/snap", reqBody)

	handler.SnapToRoad(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "path is required")
}

func TestHandler_SnapToRoad_ExceedsMaxPoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	// Create 101 points to exceed the limit
	path := make([]map[string]float64, 101)
	for i := 0; i < 101; i++ {
		path[i] = map[string]float64{"latitude": 37.7749 + float64(i)*0.0001, "longitude": -122.4194}
	}

	reqBody := map[string]interface{}{
		"path": path,
	}

	c, w := setupTestContext("POST", "/api/v1/maps/roads/snap", reqBody)

	handler.SnapToRoad(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "maximum 100 points")
}

func TestHandler_SnapToRoad_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	mockService.On("SnapToRoad", mock.Anything, mock.AnythingOfType("*maps.SnapToRoadRequest")).Return(nil, errors.New("snap to road failed"))

	reqBody := map[string]interface{}{
		"path": []map[string]float64{
			{"latitude": 37.7749, "longitude": -122.4194},
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/roads/snap", reqBody)

	handler.SnapToRoad(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetSpeedLimits Handler Tests
// ============================================================================

func TestHandler_GetSpeedLimits_SuccessWithPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestSpeedLimitsResponse()
	mockService.On("GetSpeedLimits", mock.Anything, mock.AnythingOfType("*maps.SpeedLimitsRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"path": []map[string]float64{
			{"latitude": 37.7749, "longitude": -122.4194},
			{"latitude": 37.7769, "longitude": -122.4174},
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/roads/speed-limits", reqBody)

	handler.GetSpeedLimits(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetSpeedLimits_SuccessWithPlaceIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestSpeedLimitsResponse()
	mockService.On("GetSpeedLimits", mock.Anything, mock.AnythingOfType("*maps.SpeedLimitsRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"place_ids": []string{"road123", "road124"},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/roads/speed-limits", reqBody)

	handler.GetSpeedLimits(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetSpeedLimits_MissingPathAndPlaceIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/maps/roads/speed-limits", reqBody)

	handler.GetSpeedLimits(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "path or place_ids is required")
}

func TestHandler_GetSpeedLimits_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	mockService.On("GetSpeedLimits", mock.Anything, mock.AnythingOfType("*maps.SpeedLimitsRequest")).Return(nil, errors.New("speed limits unavailable"))

	reqBody := map[string]interface{}{
		"place_ids": []string{"road123"},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/roads/speed-limits", reqBody)

	handler.GetSpeedLimits(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// OptimizeWaypoints Handler Tests
// ============================================================================

func TestHandler_OptimizeWaypoints_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedOrder := []int{1, 0, 2}
	mockService.On("OptimizeWaypoints", mock.Anything, mock.AnythingOfType("maps.Coordinate"), mock.AnythingOfType("[]maps.Coordinate"), mock.AnythingOfType("maps.Coordinate")).Return(expectedOrder, nil)

	reqBody := map[string]interface{}{
		"origin": map[string]float64{
			"latitude":  37.0,
			"longitude": -122.0,
		},
		"waypoints": []map[string]float64{
			{"latitude": 37.5, "longitude": -122.0},
			{"latitude": 37.3, "longitude": -122.0},
			{"latitude": 37.7, "longitude": -122.0},
		},
		"destination": map[string]float64{
			"latitude":  38.0,
			"longitude": -122.0,
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/waypoints/optimize", reqBody)

	handler.OptimizeWaypoints(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_OptimizeWaypoints_EmptyWaypoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	reqBody := map[string]interface{}{
		"origin": map[string]float64{
			"latitude":  37.0,
			"longitude": -122.0,
		},
		"waypoints": []map[string]float64{},
		"destination": map[string]float64{
			"latitude":  38.0,
			"longitude": -122.0,
		},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/waypoints/optimize", reqBody)

	handler.OptimizeWaypoints(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "waypoints are required")
}

func TestHandler_OptimizeWaypoints_ExceedsMaxWaypoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	// Create 24 waypoints to exceed the 23 limit
	waypoints := make([]map[string]float64, 24)
	for i := 0; i < 24; i++ {
		waypoints[i] = map[string]float64{"latitude": 37.0 + float64(i)*0.1, "longitude": -122.0}
	}

	reqBody := map[string]interface{}{
		"origin":      map[string]float64{"latitude": 37.0, "longitude": -122.0},
		"waypoints":   waypoints,
		"destination": map[string]float64{"latitude": 38.0, "longitude": -122.0},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/waypoints/optimize", reqBody)

	handler.OptimizeWaypoints(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "maximum 23 waypoints")
}

func TestHandler_OptimizeWaypoints_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	mockService.On("OptimizeWaypoints", mock.Anything, mock.AnythingOfType("maps.Coordinate"), mock.AnythingOfType("[]maps.Coordinate"), mock.AnythingOfType("maps.Coordinate")).Return(nil, errors.New("optimization failed"))

	reqBody := map[string]interface{}{
		"origin":      map[string]float64{"latitude": 37.0, "longitude": -122.0},
		"waypoints":   []map[string]float64{{"latitude": 37.5, "longitude": -122.0}},
		"destination": map[string]float64{"latitude": 38.0, "longitude": -122.0},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/waypoints/optimize", reqBody)

	handler.OptimizeWaypoints(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// HealthCheck Handler Tests
// ============================================================================

func TestHandler_HealthCheck_AllHealthy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	healthResults := map[Provider]error{
		ProviderGoogle: nil,
		ProviderHERE:   nil,
	}
	mockService.On("HealthCheck", mock.Anything).Return(healthResults)
	mockService.On("GetPrimaryProvider").Return(ProviderGoogle)

	c, w := setupTestContext("GET", "/api/v1/maps/health", nil)

	handler.HealthCheck(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "healthy", data["status"])
	mockService.AssertExpectations(t)
}

func TestHandler_HealthCheck_Degraded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	healthResults := map[Provider]error{
		ProviderGoogle: nil,
		ProviderHERE:   errors.New("HERE unavailable"),
	}
	mockService.On("HealthCheck", mock.Anything).Return(healthResults)
	mockService.On("GetPrimaryProvider").Return(ProviderGoogle)

	c, w := setupTestContext("GET", "/api/v1/maps/health", nil)

	handler.HealthCheck(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "degraded", data["status"])
	mockService.AssertExpectations(t)
}

func TestHandler_HealthCheck_Unhealthy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	healthResults := map[Provider]error{
		ProviderGoogle: errors.New("Google unavailable"),
		ProviderHERE:   nil,
	}
	mockService.On("HealthCheck", mock.Anything).Return(healthResults)
	mockService.On("GetPrimaryProvider").Return(ProviderGoogle)

	c, w := setupTestContext("GET", "/api/v1/maps/health", nil)

	handler.HealthCheck(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "unhealthy", data["status"])
	mockService.AssertExpectations(t)
}

// ============================================================================
// Coordinate Validation Tests
// ============================================================================

func TestHandler_IsValidCoordinate_ValidCases(t *testing.T) {
	tests := []struct {
		name string
		coord Coordinate
		valid bool
	}{
		{"valid positive coordinates", Coordinate{Latitude: 37.7749, Longitude: -122.4194}, true},
		{"valid negative latitude", Coordinate{Latitude: -33.8688, Longitude: 151.2093}, true},
		{"valid extreme north", Coordinate{Latitude: 89.99, Longitude: 0}, true},
		{"valid extreme south", Coordinate{Latitude: -89.99, Longitude: 0}, true},
		{"valid extreme east", Coordinate{Latitude: 0, Longitude: 179.99}, true},
		{"valid extreme west", Coordinate{Latitude: 0, Longitude: -179.99}, true},
		{"null island valid", Coordinate{Latitude: 0, Longitude: 0}, true},
		{"boundary lat 90", Coordinate{Latitude: 90, Longitude: 0}, true},
		{"boundary lat -90", Coordinate{Latitude: -90, Longitude: 0}, true},
		{"boundary lng 180", Coordinate{Latitude: 0, Longitude: 180}, true},
		{"boundary lng -180", Coordinate{Latitude: 0, Longitude: -180}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidCoordinate(tt.coord)
			assert.Equal(t, tt.valid, result)
		})
	}
}

func TestHandler_IsValidCoordinate_InvalidCases(t *testing.T) {
	tests := []struct {
		name string
		coord Coordinate
	}{
		{"latitude too high", Coordinate{Latitude: 91, Longitude: 0}},
		{"latitude too low", Coordinate{Latitude: -91, Longitude: 0}},
		{"longitude too high", Coordinate{Latitude: 0, Longitude: 181}},
		{"longitude too low", Coordinate{Latitude: 0, Longitude: -181}},
		{"both out of range", Coordinate{Latitude: 100, Longitude: 200}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidCoordinate(tt.coord)
			assert.False(t, result)
		})
	}
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_GetRoute_ValidationCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           interface{}
		setupMock      func(*MockMapsService)
		expectedStatus int
	}{
		{
			name: "valid route request",
			body: map[string]interface{}{
				"origin":      map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
				"destination": map[string]float64{"latitude": 37.3382, "longitude": -121.8863},
			},
			setupMock: func(m *MockMapsService) {
				m.On("GetRoute", mock.Anything, mock.AnythingOfType("*maps.RouteRequest")).Return(createTestRouteResponse(), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "origin latitude out of range high",
			body: map[string]interface{}{
				"origin":      map[string]float64{"latitude": 91.0, "longitude": -122.4194},
				"destination": map[string]float64{"latitude": 37.3382, "longitude": -121.8863},
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "origin latitude out of range low",
			body: map[string]interface{}{
				"origin":      map[string]float64{"latitude": -91.0, "longitude": -122.4194},
				"destination": map[string]float64{"latitude": 37.3382, "longitude": -121.8863},
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "destination longitude out of range",
			body: map[string]interface{}{
				"origin":      map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
				"destination": map[string]float64{"latitude": 37.3382, "longitude": 181.0},
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service error",
			body: map[string]interface{}{
				"origin":      map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
				"destination": map[string]float64{"latitude": 37.3382, "longitude": -121.8863},
			},
			setupMock: func(m *MockMapsService) {
				m.On("GetRoute", mock.Anything, mock.AnythingOfType("*maps.RouteRequest")).Return(nil, errors.New("error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockMapsService)
			handler := NewTestableHandler(mockService)

			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			c, w := setupTestContext("POST", "/api/v1/maps/route", tt.body)

			handler.GetRoute(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_Geocode_ValidationCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           interface{}
		setupMock      func(*MockMapsService)
		expectedStatus int
	}{
		{
			name: "valid geocode request",
			body: map[string]interface{}{
				"address": "1600 Amphitheatre Parkway",
			},
			setupMock: func(m *MockMapsService) {
				m.On("Geocode", mock.Anything, mock.AnythingOfType("*maps.GeocodingRequest")).Return(createTestGeocodingResponse(), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing address",
			body: map[string]interface{}{
				"language": "en",
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty address",
			body: map[string]interface{}{
				"address": "",
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service error",
			body: map[string]interface{}{
				"address": "Some address",
			},
			setupMock: func(m *MockMapsService) {
				m.On("Geocode", mock.Anything, mock.AnythingOfType("*maps.GeocodingRequest")).Return(nil, errors.New("error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockMapsService)
			handler := NewTestableHandler(mockService)

			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			c, w := setupTestContext("POST", "/api/v1/maps/geocode", tt.body)

			handler.Geocode(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_DistanceMatrix_ValidationCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           interface{}
		setupMock      func(*MockMapsService)
		expectedStatus int
	}{
		{
			name: "valid request",
			body: map[string]interface{}{
				"origins":      []map[string]float64{{"latitude": 37.7749, "longitude": -122.4194}},
				"destinations": []map[string]float64{{"latitude": 37.3382, "longitude": -121.8863}},
			},
			setupMock: func(m *MockMapsService) {
				m.On("GetDistanceMatrix", mock.Anything, mock.AnythingOfType("*maps.DistanceMatrixRequest")).Return(createTestDistanceMatrixResponse(), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "empty origins",
			body: map[string]interface{}{
				"origins":      []map[string]float64{},
				"destinations": []map[string]float64{{"latitude": 37.3382, "longitude": -121.8863}},
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty destinations",
			body: map[string]interface{}{
				"origins":      []map[string]float64{{"latitude": 37.7749, "longitude": -122.4194}},
				"destinations": []map[string]float64{},
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockMapsService)
			handler := NewTestableHandler(mockService)

			if tt.setupMock != nil {
				tt.setupMock(mockService)
			}

			c, w := setupTestContext("POST", "/api/v1/maps/distance-matrix", tt.body)

			handler.GetDistanceMatrix(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// Edge Cases Tests
// ============================================================================

func TestHandler_GetRoute_ExtremeCoordinates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		lat  float64
		lng  float64
	}{
		{"North Pole", 90.0, 0.0},
		{"South Pole", -90.0, 0.0},
		{"Date Line East", 0.0, 180.0},
		{"Date Line West", 0.0, -180.0},
		{"Near North Pole", 89.999, 0.0},
		{"Near South Pole", -89.999, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockMapsService)
			handler := NewTestableHandler(mockService)

			expectedResp := createTestRouteResponse()
			mockService.On("GetRoute", mock.Anything, mock.AnythingOfType("*maps.RouteRequest")).Return(expectedResp, nil)

			reqBody := map[string]interface{}{
				"origin":      map[string]float64{"latitude": tt.lat, "longitude": tt.lng},
				"destination": map[string]float64{"latitude": 37.3382, "longitude": -121.8863},
			}

			c, w := setupTestContext("POST", "/api/v1/maps/route", reqBody)

			handler.GetRoute(c)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestHandler_GetETA_LongDistanceRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	// NYC to London - intercontinental
	expectedResp := createTestETAResponse()
	expectedResp.DistanceKm = 5570.0
	expectedResp.DurationMinutes = 480.0

	mockService.On("GetTrafficAwareETA", mock.Anything, mock.AnythingOfType("maps.Coordinate"), mock.AnythingOfType("maps.Coordinate")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"origin":      map[string]float64{"latitude": 40.7128, "longitude": -74.0060},
		"destination": map[string]float64{"latitude": 51.5074, "longitude": -0.1278},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/eta", reqBody)

	handler.GetETA(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SearchPlaces_SpecialCharactersInQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestPlaceSearchResponse()
	mockService.On("SearchPlaces", mock.Anything, mock.AnythingOfType("*maps.PlaceSearchRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"query": "McDonald's & Starbucks 'coffee' \"espresso\"",
	}

	c, w := setupTestContext("POST", "/api/v1/maps/places/search", reqBody)

	handler.SearchPlaces(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_Geocode_UnicodeAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestGeocodingResponse()
	mockService.On("Geocode", mock.Anything, mock.AnythingOfType("*maps.GeocodingRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"address": "東京都渋谷区神宮前",
	}

	c, w := setupTestContext("POST", "/api/v1/maps/geocode", reqBody)

	handler.Geocode(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_GetRoute_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	expectedResp := createTestRouteResponse()
	mockService.On("GetRoute", mock.Anything, mock.AnythingOfType("*maps.RouteRequest")).Return(expectedResp, nil)

	reqBody := map[string]interface{}{
		"origin":      map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
		"destination": map[string]float64{"latitude": 37.3382, "longitude": -121.8863},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/route", reqBody)

	handler.GetRoute(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	// Verify response structure
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["routes"])
	assert.NotNil(t, data["provider"])

	mockService.AssertExpectations(t)
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockMapsService)
	handler := NewTestableHandler(mockService)

	mockService.On("GetRoute", mock.Anything, mock.AnythingOfType("*maps.RouteRequest")).Return(nil, errors.New("error"))

	reqBody := map[string]interface{}{
		"origin":      map[string]float64{"latitude": 37.7749, "longitude": -122.4194},
		"destination": map[string]float64{"latitude": 37.3382, "longitude": -121.8863},
	}

	c, w := setupTestContext("POST", "/api/v1/maps/route", reqBody)

	handler.GetRoute(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)

	// Verify error response structure
	assert.False(t, response["success"].(bool))
	assert.NotNil(t, response["error"])

	errorInfo := response["error"].(map[string]interface{})
	assert.NotNil(t, errorInfo["code"])
	assert.NotNil(t, errorInfo["message"])

	mockService.AssertExpectations(t)
}
