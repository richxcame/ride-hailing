package pricing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// Mock Implementations
// ============================================================================

// MockPricingService implements the pricing service interface for testing
type MockPricingService struct {
	mock.Mock
}

func (m *MockPricingService) GetEstimate(ctx context.Context, req EstimateRequest) (*EstimateResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EstimateResponse), args.Error(1)
}

func (m *MockPricingService) GetSurgeInfo(ctx context.Context, latitude, longitude float64) (map[string]interface{}, error) {
	args := m.Called(ctx, latitude, longitude)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockPricingService) ValidateNegotiatedPrice(ctx context.Context, req EstimateRequest, negotiatedPrice float64) error {
	args := m.Called(ctx, req, negotiatedPrice)
	return args.Error(0)
}

func (m *MockPricingService) GetPricing(ctx context.Context, latitude, longitude float64, rideTypeID *uuid.UUID) (*ResolvedPricing, error) {
	args := m.Called(ctx, latitude, longitude, rideTypeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ResolvedPricing), args.Error(1)
}

func (m *MockPricingService) GetCancellationFee(ctx context.Context, latitude, longitude float64, minutesSinceRequest float64, estimatedFare float64) (float64, error) {
	args := m.Called(ctx, latitude, longitude, minutesSinceRequest, estimatedFare)
	return args.Get(0).(float64), args.Error(1)
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

func setupTestContextWithQuery(method, path, query string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(method, path+"?"+query, nil)
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

func createTestHandler(mockService *MockPricingService) *Handler {
	return &Handler{service: &Service{}}
}

func createTestEstimateResponse() *EstimateResponse {
	return &EstimateResponse{
		Currency:         "USD",
		EstimatedFare:    25.50,
		MinimumFare:      5.00,
		SurgeMultiplier:  1.0,
		DistanceKm:       10.5,
		EstimatedMinutes: 20,
		FormattedFare:    "$25.50",
		FareBreakdown: &FareCalculation{
			DistanceKm:        10.5,
			DurationMin:       20,
			Currency:          "USD",
			BaseFare:          3.00,
			DistanceCharge:    15.75,
			TimeCharge:        5.00,
			BookingFee:        1.00,
			TotalFare:         25.50,
			TimeMultiplier:    1.0,
			WeatherMultiplier: 1.0,
			EventMultiplier:   1.0,
			SurgeMultiplier:   1.0,
			TotalMultiplier:   1.0,
		},
	}
}

func createTestResolvedPricing() *ResolvedPricing {
	return &ResolvedPricing{
		VersionID:             uuid.New(),
		BaseFare:              3.00,
		PerKmRate:             1.50,
		PerMinuteRate:         0.25,
		MinimumFare:           5.00,
		BookingFee:            1.00,
		PlatformCommissionPct: 20.00,
		DriverIncentivePct:    0.00,
		SurgeMinMultiplier:    1.00,
		SurgeMaxMultiplier:    5.00,
		TaxRatePct:            0.00,
		TaxInclusive:          false,
	}
}

// ============================================================================
// GetEstimate Handler Tests
// ============================================================================

// Note: TestHandler_GetEstimate_Success requires full integration test setup
// with database, geography service, and currency service. The validation
// error tests below cover handler binding behavior without external deps.

func TestHandler_GetEstimate_ValidationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		request        interface{}
		expectedStatus int
	}{
		{
			name:           "empty request body",
			request:        nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid json",
			request:        "invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing pickup latitude",
			request: map[string]interface{}{
				"pickup_longitude":  -122.4194,
				"dropoff_latitude":  37.7849,
				"dropoff_longitude": -122.4094,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing pickup longitude",
			request: map[string]interface{}{
				"pickup_latitude":   37.7749,
				"dropoff_latitude":  37.7849,
				"dropoff_longitude": -122.4094,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing dropoff latitude",
			request: map[string]interface{}{
				"pickup_latitude":   37.7749,
				"pickup_longitude":  -122.4194,
				"dropoff_longitude": -122.4094,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing dropoff longitude",
			request: map[string]interface{}{
				"pickup_latitude":  37.7749,
				"pickup_longitude": -122.4194,
				"dropoff_latitude": 37.7849,
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{service: &Service{}}

			c, w := setupTestContext("POST", "/api/v1/pricing/estimate", tt.request)

			handler.GetEstimate(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			response := parseResponse(w)
			assert.False(t, response["success"].(bool))
		})
	}
}

// ============================================================================
// GetSurge Handler Tests
// ============================================================================

// Note: TestHandler_GetSurge_Success requires full integration test setup
// with geography service. The validation error tests below cover handler
// parameter validation without external dependencies.

func TestHandler_GetSurge_ValidationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		query          string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "missing latitude and longitude",
			query:          "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "latitude and longitude are required",
		},
		{
			name:           "missing latitude",
			query:          "longitude=-122.4194",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "latitude and longitude are required",
		},
		{
			name:           "missing longitude",
			query:          "latitude=37.7749",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "latitude and longitude are required",
		},
		{
			name:           "invalid latitude format",
			query:          "latitude=invalid&longitude=-122.4194",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid latitude",
		},
		{
			name:           "invalid longitude format",
			query:          "latitude=37.7749&longitude=invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid longitude",
		},
		{
			name:           "latitude with letters",
			query:          "latitude=37.7abc&longitude=-122.4194",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid latitude",
		},
		{
			name:           "longitude with letters",
			query:          "latitude=37.7749&longitude=-122.xyz",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid longitude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{service: &Service{}}

			c, w := setupTestContextWithQuery("GET", "/api/v1/pricing/surge", tt.query)

			handler.GetSurge(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			response := parseResponse(w)
			assert.False(t, response["success"].(bool))
			errorInfo := response["error"].(map[string]interface{})
			assert.Contains(t, errorInfo["message"], tt.expectedError)
		})
	}
}

// ============================================================================
// ValidatePrice Handler Tests
// ============================================================================

func TestHandler_ValidatePrice_ValidationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		request        interface{}
		expectedStatus int
	}{
		{
			name:           "empty request body",
			request:        nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing negotiated price",
			request: map[string]interface{}{
				"pickup_latitude":   37.7749,
				"pickup_longitude":  -122.4194,
				"dropoff_latitude":  37.7849,
				"dropoff_longitude": -122.4094,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "zero negotiated price",
			request: map[string]interface{}{
				"pickup_latitude":   37.7749,
				"pickup_longitude":  -122.4194,
				"dropoff_latitude":  37.7849,
				"dropoff_longitude": -122.4094,
				"negotiated_price":  0,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "negative negotiated price",
			request: map[string]interface{}{
				"pickup_latitude":   37.7749,
				"pickup_longitude":  -122.4194,
				"dropoff_latitude":  37.7849,
				"dropoff_longitude": -122.4094,
				"negotiated_price":  -10.0,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing pickup coordinates",
			request: map[string]interface{}{
				"dropoff_latitude":  37.7849,
				"dropoff_longitude": -122.4094,
				"negotiated_price":  25.0,
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{service: &Service{}}

			c, w := setupTestContext("POST", "/api/v1/pricing/validate", tt.request)

			handler.ValidatePrice(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			response := parseResponse(w)
			assert.False(t, response["success"].(bool))
		})
	}
}

// ============================================================================
// GetPricing Handler Tests
// ============================================================================

func TestHandler_GetPricing_ValidationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		query          string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "missing latitude and longitude",
			query:          "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "latitude and longitude are required",
		},
		{
			name:           "missing latitude",
			query:          "longitude=-122.4194",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "latitude and longitude are required",
		},
		{
			name:           "missing longitude",
			query:          "latitude=37.7749",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "latitude and longitude are required",
		},
		{
			name:           "invalid latitude",
			query:          "latitude=invalid&longitude=-122.4194",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid latitude",
		},
		{
			name:           "invalid longitude",
			query:          "latitude=37.7749&longitude=invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid longitude",
		},
		{
			name:           "invalid ride_type_id",
			query:          "latitude=37.7749&longitude=-122.4194&ride_type_id=not-a-uuid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid ride_type_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{service: &Service{}}

			c, w := setupTestContextWithQuery("GET", "/api/v1/pricing/config", tt.query)

			handler.GetPricing(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			response := parseResponse(w)
			assert.False(t, response["success"].(bool))
			errorInfo := response["error"].(map[string]interface{})
			assert.Contains(t, errorInfo["message"], tt.expectedError)
		})
	}
}

// Note: TestHandler_GetPricing_ValidRideTypeID requires full integration test
// setup with geography service. Validation tests above cover parameter parsing.

// ============================================================================
// GetCancellationFee Handler Tests
// ============================================================================

func TestHandler_GetCancellationFee_ValidationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		request        interface{}
		expectedStatus int
	}{
		{
			name:           "empty request body",
			request:        nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing latitude",
			request: map[string]interface{}{
				"longitude":             -122.4194,
				"minutes_since_request": 5.0,
				"estimated_fare":        25.0,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing longitude",
			request: map[string]interface{}{
				"latitude":              37.7749,
				"minutes_since_request": 5.0,
				"estimated_fare":        25.0,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing minutes_since_request",
			request: map[string]interface{}{
				"latitude":       37.7749,
				"longitude":      -122.4194,
				"estimated_fare": 25.0,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing estimated_fare",
			request: map[string]interface{}{
				"latitude":              37.7749,
				"longitude":             -122.4194,
				"minutes_since_request": 5.0,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "negative minutes_since_request",
			request: map[string]interface{}{
				"latitude":              37.7749,
				"longitude":             -122.4194,
				"minutes_since_request": -5.0,
				"estimated_fare":        25.0,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "zero estimated_fare",
			request: map[string]interface{}{
				"latitude":              37.7749,
				"longitude":             -122.4194,
				"minutes_since_request": 5.0,
				"estimated_fare":        0,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "negative estimated_fare",
			request: map[string]interface{}{
				"latitude":              37.7749,
				"longitude":             -122.4194,
				"minutes_since_request": 5.0,
				"estimated_fare":        -25.0,
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{service: &Service{}}

			c, w := setupTestContext("POST", "/api/v1/pricing/cancellation-fee", tt.request)

			handler.GetCancellationFee(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			response := parseResponse(w)
			assert.False(t, response["success"].(bool))
		})
	}
}

// Note: TestHandler_GetCancellationFee_ValidRequest requires full integration
// test setup. Validation tests above cover parameter parsing.

// ============================================================================
// Table-Driven Tests - Coordinate Validation
// ============================================================================

func TestHandler_CoordinateValidation(t *testing.T) {
	// Tests for coordinate validation at handler level (validation errors only)
	// Tests expecting to reach service level require full integration setup
	tests := []struct {
		name           string
		endpoint       string
		method         string
		query          string
		body           interface{}
		expectedStatus int
	}{
		{
			name:           "surge - empty latitude",
			endpoint:       "/api/v1/pricing/surge",
			method:         "GET",
			query:          "latitude=&longitude=-122.4194",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "surge - empty longitude",
			endpoint:       "/api/v1/pricing/surge",
			method:         "GET",
			query:          "latitude=37.7749&longitude=",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			handler := &Handler{service: &Service{}}

			var c *gin.Context
			var w *httptest.ResponseRecorder

			if tt.method == "GET" {
				c, w = setupTestContextWithQuery(tt.method, tt.endpoint, tt.query)
			} else {
				c, w = setupTestContext(tt.method, tt.endpoint, tt.body)
			}

			switch tt.endpoint {
			case "/api/v1/pricing/surge":
				handler.GetSurge(c)
			case "/api/v1/pricing/config":
				handler.GetPricing(c)
			}

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &Handler{service: &Service{}}

	c, w := setupTestContextWithQuery("GET", "/api/v1/pricing/surge", "")

	handler.GetSurge(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
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
// Note: Edge case tests that expect to reach the service layer require full
// integration test setup with geography service. Validation tests above
// cover handler-level parameter parsing.
// ============================================================================

func TestHandler_GetSurge_EdgeCases(t *testing.T) {
	// Skipped: requires full service initialization with geography service
	t.Skip("Requires integration test setup with geography service")
}

func TestHandler_GetEstimate_EdgeCases(t *testing.T) {
	// Skipped: requires full service initialization with geography service
	t.Skip("Requires integration test setup with geography service")
}

// ============================================================================
// Route Registration Tests
// ============================================================================

func TestHandler_RegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handler := &Handler{service: &Service{}}

	rg := router.Group("/api/v1")
	handler.RegisterRoutes(rg)

	// Test that routes are registered
	routes := router.Routes()

	expectedRoutes := map[string]string{
		"POST/api/v1/pricing/estimate":         "GetEstimate",
		"GET/api/v1/pricing/surge":             "GetSurge",
		"POST/api/v1/pricing/validate":         "ValidatePrice",
		"GET/api/v1/pricing/config":            "GetPricing",
		"POST/api/v1/pricing/cancellation-fee": "GetCancellationFee",
	}

	registeredRoutes := make(map[string]bool)
	for _, route := range routes {
		key := route.Method + route.Path
		registeredRoutes[key] = true
	}

	for expectedRoute := range expectedRoutes {
		assert.True(t, registeredRoutes[expectedRoute], "Route %s should be registered", expectedRoute)
	}
}

// ============================================================================
// Invalid JSON Tests
// ============================================================================

func TestHandler_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		endpoint string
		handler  func(*Handler, *gin.Context)
	}{
		{
			name:     "estimate with invalid json",
			endpoint: "/api/v1/pricing/estimate",
			handler: func(h *Handler, c *gin.Context) {
				h.GetEstimate(c)
			},
		},
		{
			name:     "validate with invalid json",
			endpoint: "/api/v1/pricing/validate",
			handler: func(h *Handler, c *gin.Context) {
				h.ValidatePrice(c)
			},
		},
		{
			name:     "cancellation-fee with invalid json",
			endpoint: "/api/v1/pricing/cancellation-fee",
			handler: func(h *Handler, c *gin.Context) {
				h.GetCancellationFee(c)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{service: &Service{}}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", tt.endpoint, bytes.NewReader([]byte("invalid json {")))
			c.Request.Header.Set("Content-Type", "application/json")

			tt.handler(handler, c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			response := parseResponse(w)
			assert.False(t, response["success"].(bool))
		})
	}
}

// ============================================================================
// Content-Type Tests
// ============================================================================

func TestHandler_ContentType(t *testing.T) {
	// Skipped: requires full service initialization
	// Content-type handling is tested by the Gin framework itself
	t.Skip("Requires integration test setup with geography service")
}

// ============================================================================
// Service Error Handling Tests
// ============================================================================

func TestHandler_ServiceErrors(t *testing.T) {
	// Skipped: requires full service initialization with geography service
	// Service error handling is covered by integration tests
	t.Skip("Requires integration test setup with geography service")
}

// ============================================================================
// Haversine Distance Calculation Test (Helper Function)
// ============================================================================

func TestHaversineDistanceHandler(t *testing.T) {
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
			name:       "same location",
			latitude1:  37.7749,
			longitude1: -122.4194,
			latitude2:  37.7749,
			longitude2: -122.4194,
			expected:   0,
			delta:      0.001,
		},
		{
			name:       "short distance",
			latitude1:  37.7749,
			longitude1: -122.4194,
			latitude2:  37.7849,
			longitude2: -122.4094,
			expected:   1.4, // approximately 1.4 km
			delta:      0.2,
		},
		{
			name:       "SF to LA approximately",
			latitude1:  37.7749,
			longitude1: -122.4194,
			latitude2:  34.0522,
			longitude2: -118.2437,
			expected:   559, // approximately 559 km
			delta:      10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := haversineDistance(tt.latitude1, tt.longitude1, tt.latitude2, tt.longitude2)
			assert.InDelta(t, tt.expected, result, tt.delta)
		})
	}
}

// ============================================================================
// Nil Service Test
// ============================================================================

func TestHandler_NilServiceFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Handler with nil service should still handle request binding errors gracefully
	handler := &Handler{service: nil}

	c, w := setupTestContext("POST", "/api/v1/pricing/estimate", nil)

	// This should panic or fail gracefully - we're testing edge cases
	defer func() {
		if r := recover(); r != nil {
			// Expected behavior when service is nil
			t.Log("Handler panicked with nil service as expected")
		}
	}()

	handler.GetEstimate(c)

	// If we get here without panic, check response
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Mock Service Integration Tests
// ============================================================================

func TestMockPricingService(t *testing.T) {
	mockService := new(MockPricingService)

	// Test GetEstimate mock
	mockService.On("GetEstimate", mock.Anything, mock.AnythingOfType("EstimateRequest")).Return(createTestEstimateResponse(), nil)
	result, err := mockService.GetEstimate(context.Background(), EstimateRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Test GetSurgeInfo mock
	surgeInfo := map[string]interface{}{
		"current_surge": 1.5,
		"surge_active":  true,
	}
	mockService.On("GetSurgeInfo", mock.Anything, 37.7749, -122.4194).Return(surgeInfo, nil)
	surgeResult, err := mockService.GetSurgeInfo(context.Background(), 37.7749, -122.4194)
	assert.NoError(t, err)
	assert.Equal(t, 1.5, surgeResult["current_surge"])

	// Test ValidateNegotiatedPrice mock
	mockService.On("ValidateNegotiatedPrice", mock.Anything, mock.AnythingOfType("EstimateRequest"), 30.0).Return(nil)
	err = mockService.ValidateNegotiatedPrice(context.Background(), EstimateRequest{}, 30.0)
	assert.NoError(t, err)

	// Test GetPricing mock
	mockService.On("GetPricing", mock.Anything, 37.7749, -122.4194, (*uuid.UUID)(nil)).Return(createTestResolvedPricing(), nil)
	pricing, err := mockService.GetPricing(context.Background(), 37.7749, -122.4194, nil)
	assert.NoError(t, err)
	assert.Equal(t, 3.00, pricing.BaseFare)

	// Test GetCancellationFee mock
	mockService.On("GetCancellationFee", mock.Anything, 37.7749, -122.4194, 5.0, 25.0).Return(5.0, nil)
	fee, err := mockService.GetCancellationFee(context.Background(), 37.7749, -122.4194, 5.0, 25.0)
	assert.NoError(t, err)
	assert.Equal(t, 5.0, fee)

	mockService.AssertExpectations(t)
}

func TestMockPricingService_Errors(t *testing.T) {
	mockService := new(MockPricingService)

	// Test GetEstimate error
	mockService.On("GetEstimate", mock.Anything, mock.AnythingOfType("EstimateRequest")).Return(nil, errors.New("service error"))
	result, err := mockService.GetEstimate(context.Background(), EstimateRequest{})
	assert.Error(t, err)
	assert.Nil(t, result)

	// Test GetSurgeInfo error
	mockService.On("GetSurgeInfo", mock.Anything, 0.0, 0.0).Return(nil, errors.New("invalid coordinates"))
	surgeResult, err := mockService.GetSurgeInfo(context.Background(), 0.0, 0.0)
	assert.Error(t, err)
	assert.Nil(t, surgeResult)

	// Test GetPricing error
	mockService.On("GetPricing", mock.Anything, 0.0, 0.0, (*uuid.UUID)(nil)).Return(nil, errors.New("location not found"))
	pricing, err := mockService.GetPricing(context.Background(), 0.0, 0.0, nil)
	assert.Error(t, err)
	assert.Nil(t, pricing)

	mockService.AssertExpectations(t)
}
