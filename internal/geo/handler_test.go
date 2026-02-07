package geo

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/richxcame/ride-hailing/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)


// ============================================================================
// Test Helper Functions
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

func setupTestContextWithQuery(method, path string, query map[string]string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	fullPath := path
	if len(query) > 0 {
		params := url.Values{}
		for k, v := range query {
			params.Set(k, v)
		}
		fullPath += "?" + params.Encode()
	}

	req := httptest.NewRequest(method, fullPath, nil)
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

func createTestDriverLocation(driverID uuid.UUID, lat, lng float64) *DriverLocation {
	return &DriverLocation{
		DriverID:  driverID,
		Latitude:  lat,
		Longitude: lng,
		H3Cell:    GetMatchingCell(lat, lng),
		Heading:   45.0,
		Speed:     30.0,
		Timestamp: time.Now(),
	}
}

func createTestSurgeInfo(lat, lng float64) *SurgeInfo {
	return &SurgeInfo{
		H3Cell:          GetSurgeZone(lat, lng),
		SurgeMultiplier: 1.5,
		DemandCount:     10,
		SupplyCount:     5,
		UpdatedAt:       time.Now(),
	}
}

func createTestDemandInfo(lat, lng float64) *DemandInfo {
	return &DemandInfo{
		H3Cell:       GetDemandZone(lat, lng),
		RequestCount: 15,
		Latitude:     lat,
		Longitude:    lng,
	}
}

func createTestGeocodingResult(lat, lng float64) *GeocodingResult {
	return &GeocodingResult{
		PlaceID:          "test-place-id-123",
		FormattedAddress: "123 Test Street, Test City",
		Latitude:         lat,
		Longitude:        lng,
		Types:            []string{"street_address"},
		H3Cell:           GetMatchingCell(lat, lng),
	}
}

func createTestAutocompleteResult() *AutocompleteResult {
	return &AutocompleteResult{
		PlaceID:     "test-place-id-456",
		Description: "123 Test Street, Test City, Country",
		MainText:    "123 Test Street",
		SecondText:  "Test City, Country",
	}
}

// ============================================================================
// UpdateLocation Handler Tests
// ============================================================================

func TestHandler_UpdateLocation_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	driverID := uuid.New()

	// Setup mock expectations for UpdateDriverLocationFull
	mockRedis.On("SetWithExpiration", mock.Anything, mock.MatchedBy(func(key string) bool {
		return key == "driver:location:"+driverID.String()
	}), mock.AnythingOfType("[]uint8"), driverLocationTTL).Return(nil)

	mockRedis.On("GeoAdd", mock.Anything, driverGeoIndexKey, float64(-122.4194), float64(37.7749), driverID.String()).Return(nil)

	// Mock H3 cell update expectations
	mockRedis.On("GetString", mock.Anything, "driver:h3cell:"+driverID.String()).Return("", errors.New("not found"))
	mockRedis.On("SetWithExpiration", mock.Anything, "driver:h3cell:"+driverID.String(), mock.AnythingOfType("[]uint8"), driverLocationTTL).Return(nil)
	mockRedis.On("SetWithExpiration", mock.Anything, mock.MatchedBy(func(key string) bool {
		return len(key) > len(h3CellDriversPrefix) && key[:len(h3CellDriversPrefix)] == h3CellDriversPrefix
	}), mock.AnythingOfType("[]uint8"), h3CellDriversTTL).Return(nil)

	reqBody := map[string]interface{}{
		"latitude":  37.7749,
		"longitude": -122.4194,
		"heading":   45.0,
		"speed":     30.0,
	}

	c, w := setupTestContext("POST", "/api/v1/geo/location", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateLocation(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "location updated successfully", data["message"])
	assert.NotEmpty(t, data["h3_cell"])
}

func TestHandler_UpdateLocation_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	reqBody := map[string]interface{}{
		"latitude":  37.7749,
		"longitude": -122.4194,
	}

	c, w := setupTestContext("POST", "/api/v1/geo/location", reqBody)
	// Don't set user context

	handler.UpdateLocation(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_UpdateLocation_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)
	driverID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/geo/location", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/geo/location", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateLocation(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateLocation_MissingLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)
	driverID := uuid.New()

	reqBody := map[string]interface{}{
		"longitude": -122.4194,
	}

	c, w := setupTestContext("POST", "/api/v1/geo/location", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateLocation(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateLocation_MissingLongitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)
	driverID := uuid.New()

	reqBody := map[string]interface{}{
		"latitude": 37.7749,
	}

	c, w := setupTestContext("POST", "/api/v1/geo/location", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateLocation(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateLocation_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	driverID := uuid.New()

	mockRedis.On("SetWithExpiration", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("redis error"))

	reqBody := map[string]interface{}{
		"latitude":  37.7749,
		"longitude": -122.4194,
	}

	c, w := setupTestContext("POST", "/api/v1/geo/location", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateLocation(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_UpdateLocation_WithHeadingAndSpeed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	driverID := uuid.New()

	// Setup mock expectations
	mockRedis.On("SetWithExpiration", mock.Anything, mock.Anything, mock.AnythingOfType("[]uint8"), mock.Anything).Return(nil)
	mockRedis.On("GeoAdd", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockRedis.On("GetString", mock.Anything, mock.Anything).Return("", errors.New("not found"))

	reqBody := map[string]interface{}{
		"latitude":  37.7749,
		"longitude": -122.4194,
		"heading":   90.0,
		"speed":     50.0,
	}

	c, w := setupTestContext("POST", "/api/v1/geo/location", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateLocation(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_UpdateLocation_ZeroHeadingAndSpeed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	driverID := uuid.New()

	// Setup mock expectations
	mockRedis.On("SetWithExpiration", mock.Anything, mock.Anything, mock.AnythingOfType("[]uint8"), mock.Anything).Return(nil)
	mockRedis.On("GeoAdd", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockRedis.On("GetString", mock.Anything, mock.Anything).Return("", errors.New("not found"))

	reqBody := map[string]interface{}{
		"latitude":  37.7749,
		"longitude": -122.4194,
		"heading":   0,
		"speed":     0,
	}

	c, w := setupTestContext("POST", "/api/v1/geo/location", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateLocation(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================================
// GetDriverLocation Handler Tests
// ============================================================================

func TestHandler_GetDriverLocation_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	driverID := uuid.New()
	expectedLocation := createTestDriverLocation(driverID, 37.7749, -122.4194)

	locationJSON, _ := json.Marshal(expectedLocation)
	mockRedis.On("GetString", mock.Anything, "driver:location:"+driverID.String()).
		Return(string(locationJSON), nil)

	c, w := setupTestContext("GET", "/api/v1/geo/drivers/"+driverID.String()+"/location", nil)
	c.Params = gin.Params{{Key: "id", Value: driverID.String()}}

	handler.GetDriverLocation(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRedis.AssertExpectations(t)
}

func TestHandler_GetDriverLocation_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContext("GET", "/api/v1/geo/drivers/invalid-uuid/location", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.GetDriverLocation(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid driver ID")
}

func TestHandler_GetDriverLocation_EmptyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContext("GET", "/api/v1/geo/drivers//location", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.GetDriverLocation(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetDriverLocation_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	driverID := uuid.New()

	mockRedis.On("GetString", mock.Anything, "driver:location:"+driverID.String()).
		Return("", errors.New("key not found"))

	c, w := setupTestContext("GET", "/api/v1/geo/drivers/"+driverID.String()+"/location", nil)
	c.Params = gin.Params{{Key: "id", Value: driverID.String()}}

	handler.GetDriverLocation(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRedis.AssertExpectations(t)
}

func TestHandler_GetDriverLocation_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	driverID := uuid.New()

	// Return invalid JSON to trigger unmarshal error
	mockRedis.On("GetString", mock.Anything, "driver:location:"+driverID.String()).
		Return("invalid json", nil)

	c, w := setupTestContext("GET", "/api/v1/geo/drivers/"+driverID.String()+"/location", nil)
	c.Params = gin.Params{{Key: "id", Value: driverID.String()}}

	handler.GetDriverLocation(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// FindNearbyDrivers Handler Tests
// ============================================================================

func TestHandler_FindNearbyDrivers_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	driver1ID := uuid.New()
	driver2ID := uuid.New()

	location1 := createTestDriverLocation(driver1ID, 37.7750, -122.4195)
	location2 := createTestDriverLocation(driver2ID, 37.7751, -122.4196)

	location1JSON, _ := json.Marshal(location1)
	location2JSON, _ := json.Marshal(location2)

	// Setup mock expectations
	// Handler uses default limit 20, FindAvailableDrivers doubles it to 40, FindNearbyDrivers doubles again to 80
	mockRedis.On("GeoRadius", mock.Anything, driverGeoIndexKey, float64(-122.4194), float64(37.7749), searchRadiusKm, 80).
		Return([]string{driver1ID.String(), driver2ID.String()}, nil)

	// Batch fetch for driver locations (new optimized path)
	mockRedis.On("MGetStrings", mock.Anything, mock.Anything).
		Return([]string{string(location1JSON), string(location2JSON)}, nil)

	// Batch fetch for driver statuses (new optimized path)
	mockRedis.On("MGetStrings", mock.Anything, mock.Anything).
		Return([]string{`{"status":"available"}`, `{"status":"available"}`}, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/drivers/nearby", map[string]string{
		"latitude":  "37.7749",
		"longitude": "-122.4194",
	})

	handler.FindNearbyDrivers(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_FindNearbyDrivers_InvalidLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/drivers/nearby", map[string]string{
		"latitude":  "invalid",
		"longitude": "-122.4194",
	})

	handler.FindNearbyDrivers(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid latitude")
}

func TestHandler_FindNearbyDrivers_InvalidLongitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/drivers/nearby", map[string]string{
		"latitude":  "37.7749",
		"longitude": "invalid",
	})

	handler.FindNearbyDrivers(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid longitude")
}

func TestHandler_FindNearbyDrivers_MissingLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/drivers/nearby", map[string]string{
		"longitude": "-122.4194",
	})

	handler.FindNearbyDrivers(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_FindNearbyDrivers_MissingLongitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/drivers/nearby", map[string]string{
		"latitude": "37.7749",
	})

	handler.FindNearbyDrivers(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_FindNearbyDrivers_WithLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	// Setup mock expectations with custom limit=5
	// FindAvailableDrivers doubles it to 10, FindNearbyDrivers doubles again to 20
	mockRedis.On("GeoRadius", mock.Anything, driverGeoIndexKey, float64(-122.4194), float64(37.7749), searchRadiusKm, 20).
		Return([]string{}, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/drivers/nearby", map[string]string{
		"latitude":  "37.7749",
		"longitude": "-122.4194",
		"limit":     "5",
	})

	handler.FindNearbyDrivers(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_FindNearbyDrivers_InvalidLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	// Invalid limit should fallback to default (20)
	// FindAvailableDrivers doubles it to 40, FindNearbyDrivers doubles again to 80
	mockRedis.On("GeoRadius", mock.Anything, driverGeoIndexKey, float64(-122.4194), float64(37.7749), searchRadiusKm, 80).
		Return([]string{}, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/drivers/nearby", map[string]string{
		"latitude":  "37.7749",
		"longitude": "-122.4194",
		"limit":     "invalid",
	})

	handler.FindNearbyDrivers(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_FindNearbyDrivers_NoDriversFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	mockRedis.On("GeoRadius", mock.Anything, driverGeoIndexKey, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]string{}, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/drivers/nearby", map[string]string{
		"latitude":  "37.7749",
		"longitude": "-122.4194",
	})

	handler.FindNearbyDrivers(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_FindNearbyDrivers_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	mockRedis.On("GeoRadius", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("redis error"))

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/drivers/nearby", map[string]string{
		"latitude":  "37.7749",
		"longitude": "-122.4194",
	})

	handler.FindNearbyDrivers(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// CalculateDistance Handler Tests
// ============================================================================

func TestHandler_CalculateDistance_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	reqBody := map[string]interface{}{
		"from_latitude":  37.7749,
		"from_longitude": -122.4194,
		"to_latitude":    37.7850,
		"to_longitude":   -122.4094,
	}

	c, w := setupTestContext("POST", "/api/v1/geo/distance", reqBody)

	handler.CalculateDistance(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["distance_km"])
	assert.NotNil(t, data["eta_minutes"])
	assert.NotNil(t, data["from_h3_cell"])
	assert.NotNil(t, data["to_h3_cell"])
}

func TestHandler_CalculateDistance_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContext("POST", "/api/v1/geo/distance", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/geo/distance", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CalculateDistance(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CalculateDistance_MissingFromLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	reqBody := map[string]interface{}{
		"from_longitude": -122.4194,
		"to_latitude":    37.7850,
		"to_longitude":   -122.4094,
	}

	c, w := setupTestContext("POST", "/api/v1/geo/distance", reqBody)

	handler.CalculateDistance(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CalculateDistance_MissingFromLongitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	reqBody := map[string]interface{}{
		"from_latitude": 37.7749,
		"to_latitude":   37.7850,
		"to_longitude":  -122.4094,
	}

	c, w := setupTestContext("POST", "/api/v1/geo/distance", reqBody)

	handler.CalculateDistance(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CalculateDistance_MissingToLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	reqBody := map[string]interface{}{
		"from_latitude":  37.7749,
		"from_longitude": -122.4194,
		"to_longitude":   -122.4094,
	}

	c, w := setupTestContext("POST", "/api/v1/geo/distance", reqBody)

	handler.CalculateDistance(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CalculateDistance_MissingToLongitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	reqBody := map[string]interface{}{
		"from_latitude":  37.7749,
		"from_longitude": -122.4194,
		"to_latitude":    37.7850,
	}

	c, w := setupTestContext("POST", "/api/v1/geo/distance", reqBody)

	handler.CalculateDistance(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CalculateDistance_SameLocation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	reqBody := map[string]interface{}{
		"from_latitude":  37.7749,
		"from_longitude": -122.4194,
		"to_latitude":    37.7749,
		"to_longitude":   -122.4194,
	}

	c, w := setupTestContext("POST", "/api/v1/geo/distance", reqBody)

	handler.CalculateDistance(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["distance_km"])
	assert.Equal(t, float64(0), data["eta_minutes"])
}

// ============================================================================
// GetSurgeInfo Handler Tests
// ============================================================================

func TestHandler_GetSurgeInfo_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	surgeInfo := createTestSurgeInfo(37.7749, -122.4194)
	surgeJSON, _ := json.Marshal(surgeInfo)

	surgeZone := GetSurgeZone(37.7749, -122.4194)
	mockRedis.On("GetString", mock.Anything, h3SurgePrefix+surgeZone).
		Return(string(surgeJSON), nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/h3/surge", map[string]string{
		"latitude":  "37.7749",
		"longitude": "-122.4194",
	})

	handler.GetSurgeInfo(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRedis.AssertExpectations(t)
}

func TestHandler_GetSurgeInfo_InvalidLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/h3/surge", map[string]string{
		"latitude":  "invalid",
		"longitude": "-122.4194",
	})

	handler.GetSurgeInfo(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid latitude")
}

func TestHandler_GetSurgeInfo_InvalidLongitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/h3/surge", map[string]string{
		"latitude":  "37.7749",
		"longitude": "invalid",
	})

	handler.GetSurgeInfo(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid longitude")
}

func TestHandler_GetSurgeInfo_MissingLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/h3/surge", map[string]string{
		"longitude": "-122.4194",
	})

	handler.GetSurgeInfo(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetSurgeInfo_NoSurgeData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	// Return not found, service should return default surge info
	mockRedis.On("GetString", mock.Anything, mock.Anything).
		Return("", errors.New("not found"))

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/h3/surge", map[string]string{
		"latitude":  "37.7749",
		"longitude": "-122.4194",
	})

	handler.GetSurgeInfo(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(1.0), data["surge_multiplier"])
}

// ============================================================================
// GetDemandHeatmap Handler Tests
// ============================================================================

func TestHandler_GetDemandHeatmap_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	demandInfo := createTestDemandInfo(37.7749, -122.4194)
	demandJSON, _ := json.Marshal(demandInfo)

	// Mock k-ring cell lookups - return demand data for some cells
	mockRedis.On("GetString", mock.Anything, mock.MatchedBy(func(key string) bool {
		return len(key) > len(h3DemandPrefix) && key[:len(h3DemandPrefix)] == h3DemandPrefix
	})).Return(string(demandJSON), nil).Maybe()

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/h3/demand-heatmap", map[string]string{
		"latitude":  "37.7749",
		"longitude": "-122.4194",
	})

	handler.GetDemandHeatmap(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetDemandHeatmap_InvalidLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/h3/demand-heatmap", map[string]string{
		"latitude":  "invalid",
		"longitude": "-122.4194",
	})

	handler.GetDemandHeatmap(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetDemandHeatmap_InvalidLongitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/h3/demand-heatmap", map[string]string{
		"latitude":  "37.7749",
		"longitude": "invalid",
	})

	handler.GetDemandHeatmap(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetDemandHeatmap_NoDemandData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	// Return not found for all cells
	mockRedis.On("GetString", mock.Anything, mock.Anything).
		Return("", errors.New("not found")).Maybe()

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/h3/demand-heatmap", map[string]string{
		"latitude":  "37.7749",
		"longitude": "-122.4194",
	})

	handler.GetDemandHeatmap(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

// ============================================================================
// GetH3Cell Handler Tests
// ============================================================================

func TestHandler_GetH3Cell_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/h3/cell", map[string]string{
		"latitude":  "37.7749",
		"longitude": "-122.4194",
	})

	handler.GetH3Cell(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(37.7749), data["latitude"])
	assert.Equal(t, float64(-122.4194), data["longitude"])
	assert.NotEmpty(t, data["matching_cell"])
	assert.NotEmpty(t, data["surge_zone"])
	assert.NotEmpty(t, data["demand_zone"])
	resolutions := data["resolutions"].(map[string]interface{})
	assert.NotEmpty(t, resolutions["6_city"])
	assert.NotEmpty(t, resolutions["7_demand"])
	assert.NotEmpty(t, resolutions["8_surge"])
	assert.NotEmpty(t, resolutions["9_matching"])
}

func TestHandler_GetH3Cell_InvalidLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/h3/cell", map[string]string{
		"latitude":  "invalid",
		"longitude": "-122.4194",
	})

	handler.GetH3Cell(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetH3Cell_InvalidLongitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/h3/cell", map[string]string{
		"latitude":  "37.7749",
		"longitude": "invalid",
	})

	handler.GetH3Cell(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetH3Cell_MissingLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/h3/cell", map[string]string{
		"longitude": "-122.4194",
	})

	handler.GetH3Cell(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetH3Cell_MissingLongitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/h3/cell", map[string]string{
		"latitude": "37.7749",
	})

	handler.GetH3Cell(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// ForwardGeocode Handler Tests
// ============================================================================

func TestHandler_ForwardGeocode_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	geocodingService := NewGeocodingService("test-api-key", mockRedis)
	handler := NewHandler(nil, geocodingService)

	expectedResult := createTestGeocodingResult(37.7749, -122.4194)
	resultJSON, _ := json.Marshal([]*GeocodingResult{expectedResult})

	// Mock cache hit
	mockRedis.On("GetString", mock.Anything, mock.MatchedBy(func(key string) bool {
		return len(key) > len(geocodeCachePrefix) && key[:len(geocodeCachePrefix)] == geocodeCachePrefix
	})).Return(string(resultJSON), nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/geocode", map[string]string{
		"address": "123 Test Street",
	})

	handler.ForwardGeocode(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_ForwardGeocode_MissingAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/geocode", map[string]string{})

	handler.ForwardGeocode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "address query parameter is required")
}

func TestHandler_ForwardGeocode_EmptyAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/geocode", map[string]string{
		"address": "",
	})

	handler.ForwardGeocode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// ReverseGeocode Handler Tests
// ============================================================================

func TestHandler_ReverseGeocode_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	geocodingService := NewGeocodingService("test-api-key", mockRedis)
	handler := NewHandler(nil, geocodingService)

	expectedResult := createTestGeocodingResult(37.7749, -122.4194)
	resultJSON, _ := json.Marshal([]*GeocodingResult{expectedResult})

	// Mock cache hit
	mockRedis.On("GetString", mock.Anything, mock.MatchedBy(func(key string) bool {
		return len(key) > len(geocodeCachePrefix) && key[:len(geocodeCachePrefix)] == geocodeCachePrefix
	})).Return(string(resultJSON), nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/geocode/reverse", map[string]string{
		"latitude":  "37.7749",
		"longitude": "-122.4194",
	})

	handler.ReverseGeocode(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_ReverseGeocode_InvalidLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/geocode/reverse", map[string]string{
		"latitude":  "invalid",
		"longitude": "-122.4194",
	})

	handler.ReverseGeocode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid latitude")
}

func TestHandler_ReverseGeocode_InvalidLongitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/geocode/reverse", map[string]string{
		"latitude":  "37.7749",
		"longitude": "invalid",
	})

	handler.ReverseGeocode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid longitude")
}

func TestHandler_ReverseGeocode_MissingLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/geocode/reverse", map[string]string{
		"longitude": "-122.4194",
	})

	handler.ReverseGeocode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ReverseGeocode_MissingLongitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/geocode/reverse", map[string]string{
		"latitude": "37.7749",
	})

	handler.ReverseGeocode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// PlaceAutocomplete Handler Tests
// ============================================================================

func TestHandler_PlaceAutocomplete_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	geocodingService := NewGeocodingService("test-api-key", mockRedis)
	handler := NewHandler(nil, geocodingService)

	expectedResult := createTestAutocompleteResult()
	resultJSON, _ := json.Marshal([]*AutocompleteResult{expectedResult})

	// Mock cache hit
	mockRedis.On("GetString", mock.Anything, mock.MatchedBy(func(key string) bool {
		return len(key) > len(autocompletePrefix) && key[:len(autocompletePrefix)] == autocompletePrefix
	})).Return(string(resultJSON), nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/geocode/autocomplete", map[string]string{
		"input": "123 Test",
	})

	handler.PlaceAutocomplete(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_PlaceAutocomplete_MissingInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/geocode/autocomplete", map[string]string{})

	handler.PlaceAutocomplete(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "input query parameter is required")
}

func TestHandler_PlaceAutocomplete_EmptyInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/geocode/autocomplete", map[string]string{
		"input": "",
	})

	handler.PlaceAutocomplete(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PlaceAutocomplete_WithLocation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	geocodingService := NewGeocodingService("test-api-key", mockRedis)
	handler := NewHandler(nil, geocodingService)

	expectedResult := createTestAutocompleteResult()
	resultJSON, _ := json.Marshal([]*AutocompleteResult{expectedResult})

	// Mock cache hit
	mockRedis.On("GetString", mock.Anything, mock.Anything).Return(string(resultJSON), nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/geocode/autocomplete", map[string]string{
		"input":     "123 Test",
		"latitude":  "37.7749",
		"longitude": "-122.4194",
	})

	handler.PlaceAutocomplete(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_PlaceAutocomplete_InvalidLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/geocode/autocomplete", map[string]string{
		"input":    "123 Test",
		"latitude": "invalid",
	})

	handler.PlaceAutocomplete(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid latitude")
}

func TestHandler_PlaceAutocomplete_InvalidLongitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/geocode/autocomplete", map[string]string{
		"input":     "123 Test",
		"longitude": "invalid",
	})

	handler.PlaceAutocomplete(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid longitude")
}

// ============================================================================
// GetPlaceDetails Handler Tests
// ============================================================================

func TestHandler_GetPlaceDetails_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	geocodingService := NewGeocodingService("test-api-key", mockRedis)
	handler := NewHandler(nil, geocodingService)

	expectedResult := createTestGeocodingResult(37.7749, -122.4194)
	resultJSON, _ := json.Marshal([]*GeocodingResult{expectedResult})

	// Mock cache hit
	mockRedis.On("GetString", mock.Anything, mock.MatchedBy(func(key string) bool {
		return len(key) > len(geocodeCachePrefix) && key[:len(geocodeCachePrefix)] == geocodeCachePrefix
	})).Return(string(resultJSON), nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/geocode/place", map[string]string{
		"place_id": "test-place-id-123",
	})

	handler.GetPlaceDetails(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetPlaceDetails_MissingPlaceID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/geocode/place", map[string]string{})

	handler.GetPlaceDetails(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "place_id query parameter is required")
}

func TestHandler_GetPlaceDetails_EmptyPlaceID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/geocode/place", map[string]string{
		"place_id": "",
	})

	handler.GetPlaceDetails(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// RegisterActiveRide Handler Tests
// ============================================================================

func TestHandler_RegisterActiveRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)

	// Create and set ETA tracker
	etaTracker := &ETATracker{
		redis:      mockRedis,
		lastUpdate: make(map[string]time.Time),
	}
	service.SetETATracker(etaTracker)
	handler := NewHandler(service, nil)

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()

	mockRedis.On("SetWithExpiration", mock.Anything, activeRidePrefix+driverID.String(), mock.AnythingOfType("[]uint8"), 2*time.Hour).Return(nil)

	reqBody := map[string]interface{}{
		"ride_id":    rideID.String(),
		"rider_id":   riderID.String(),
		"driver_id":  driverID.String(),
		"dropoff_lat": 37.7850,
		"dropoff_lng": -122.4094,
		"status":     "accepted",
		"pickup_lat": 37.7749,
		"pickup_lng": -122.4194,
	}

	c, w := setupTestContext("POST", "/api/v1/internal/eta/register", reqBody)

	handler.RegisterActiveRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_RegisterActiveRide_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	etaTracker := &ETATracker{
		redis:      mockRedis,
		lastUpdate: make(map[string]time.Time),
	}
	service.SetETATracker(etaTracker)
	handler := NewHandler(service, nil)

	c, w := setupTestContext("POST", "/api/v1/internal/eta/register", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/internal/eta/register", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.RegisterActiveRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RegisterActiveRide_ETATrackerNotEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	// Don't set ETA tracker
	handler := NewHandler(service, nil)

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()

	reqBody := map[string]interface{}{
		"ride_id":    rideID.String(),
		"rider_id":   riderID.String(),
		"driver_id":  driverID.String(),
		"dropoff_lat": 37.7850,
		"dropoff_lng": -122.4094,
		"status":     "accepted",
	}

	c, w := setupTestContext("POST", "/api/v1/internal/eta/register", reqBody)

	handler.RegisterActiveRide(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "ETA tracking not enabled")
}

func TestHandler_RegisterActiveRide_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	etaTracker := &ETATracker{
		redis:      mockRedis,
		lastUpdate: make(map[string]time.Time),
	}
	service.SetETATracker(etaTracker)
	handler := NewHandler(service, nil)

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()

	mockRedis.On("SetWithExpiration", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("redis error"))

	reqBody := map[string]interface{}{
		"ride_id":    rideID.String(),
		"rider_id":   riderID.String(),
		"driver_id":  driverID.String(),
		"dropoff_lat": 37.7850,
		"dropoff_lng": -122.4094,
		"status":     "accepted",
	}

	c, w := setupTestContext("POST", "/api/v1/internal/eta/register", reqBody)

	handler.RegisterActiveRide(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// UnregisterActiveRide Handler Tests
// ============================================================================

func TestHandler_UnregisterActiveRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	etaTracker := &ETATracker{
		redis:      mockRedis,
		lastUpdate: make(map[string]time.Time),
	}
	service.SetETATracker(etaTracker)
	handler := NewHandler(service, nil)

	driverID := uuid.New()

	mockRedis.On("Delete", mock.Anything, []string{activeRidePrefix + driverID.String()}).Return(nil)

	reqBody := map[string]interface{}{
		"driver_id": driverID.String(),
	}

	c, w := setupTestContext("POST", "/api/v1/internal/eta/unregister", reqBody)

	handler.UnregisterActiveRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "ride unregistered from ETA tracking", data["message"])
}

func TestHandler_UnregisterActiveRide_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContext("POST", "/api/v1/internal/eta/unregister", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/internal/eta/unregister", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UnregisterActiveRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UnregisterActiveRide_MissingDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/internal/eta/unregister", reqBody)

	handler.UnregisterActiveRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UnregisterActiveRide_InvalidDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	reqBody := map[string]interface{}{
		"driver_id": "invalid-uuid",
	}

	c, w := setupTestContext("POST", "/api/v1/internal/eta/unregister", reqBody)

	handler.UnregisterActiveRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid driver_id")
}

func TestHandler_UnregisterActiveRide_NoETATracker(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	// Don't set ETA tracker
	handler := NewHandler(service, nil)

	driverID := uuid.New()

	reqBody := map[string]interface{}{
		"driver_id": driverID.String(),
	}

	c, w := setupTestContext("POST", "/api/v1/internal/eta/unregister", reqBody)

	handler.UnregisterActiveRide(c)

	// Should still succeed even without ETA tracker
	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================================
// Edge Cases and Additional Tests
// ============================================================================

func TestHandler_UpdateLocation_ExtremeCoordinates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		latitude  float64
		longitude float64
	}{
		// Use coordinates that are within valid ranges but at different global locations
		{"Northern Europe", 59.3293, 18.0686},       // Stockholm
		{"Southern Hemisphere", -33.8688, 151.2093}, // Sydney
		{"Near Date Line", 35.6762, 139.6503},       // Tokyo
		{"Western Hemisphere", 40.7128, -74.0060},   // New York
		{"Equator", 0.1807, 32.4134},                // Kampala, Uganda (near equator)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRedis := new(mocks.MockRedisClient)
			service := NewService(mockRedis)
			handler := NewHandler(service, nil)

			driverID := uuid.New()

			// Setup mock expectations for each test case
			mockRedis.On("SetWithExpiration", mock.Anything, mock.Anything, mock.AnythingOfType("[]uint8"), mock.Anything).Return(nil)
			mockRedis.On("GeoAdd", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockRedis.On("GetString", mock.Anything, mock.Anything).Return("", errors.New("not found"))

			reqBody := map[string]interface{}{
				"latitude":  tt.latitude,
				"longitude": tt.longitude,
			}

			c, w := setupTestContext("POST", "/api/v1/geo/location", reqBody)
			setUserContext(c, driverID, models.RoleDriver)

			handler.UpdateLocation(c)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestHandler_CalculateDistance_LongDistance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	// New York to London
	reqBody := map[string]interface{}{
		"from_latitude":  40.7128,
		"from_longitude": -74.0060,
		"to_latitude":    51.5074,
		"to_longitude":   -0.1278,
	}

	c, w := setupTestContext("POST", "/api/v1/geo/distance", reqBody)

	handler.CalculateDistance(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	distance := data["distance_km"].(float64)
	// Distance from NYC to London is approximately 5500 km
	assert.Greater(t, distance, float64(5000))
	assert.Less(t, distance, float64(6000))
}

func TestHandler_FindNearbyDrivers_LimitEdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		limit         string
		expectedLimit int // This is the final limit passed to GeoRadius (limit * 2 * 2)
	}{
		{"zero limit uses default", "0", 80},      // 20 * 2 * 2 = 80
		{"negative limit uses default", "-5", 80}, // 20 * 2 * 2 = 80
		{"over max limit uses default", "100", 80}, // 20 * 2 * 2 = 80
		{"valid limit 1", "1", 4},                  // 1 * 2 * 2 = 4
		{"valid limit 50", "50", 200},              // 50 * 2 * 2 = 200
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRedis := new(mocks.MockRedisClient)
			service := NewService(mockRedis)
			handler := NewHandler(service, nil)

			mockRedis.On("GeoRadius", mock.Anything, driverGeoIndexKey, mock.Anything, mock.Anything, searchRadiusKm, tt.expectedLimit).
				Return([]string{}, nil)

			c, w := setupTestContextWithQuery("GET", "/api/v1/geo/drivers/nearby", map[string]string{
				"latitude":  "37.7749",
				"longitude": "-122.4194",
				"limit":     tt.limit,
			})

			handler.FindNearbyDrivers(c)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestHandler_GetH3Cell_AllResolutionsReturned(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil)

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/h3/cell", map[string]string{
		"latitude":  "37.7749",
		"longitude": "-122.4194",
	})

	handler.GetH3Cell(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})

	resolutions := data["resolutions"].(map[string]interface{})

	// Verify all resolution levels are present and non-empty
	assert.NotEmpty(t, resolutions["6_city"])
	assert.NotEmpty(t, resolutions["7_demand"])
	assert.NotEmpty(t, resolutions["8_surge"])
	assert.NotEmpty(t, resolutions["9_matching"])

	// Each should be a valid H3 cell string
	cityCell := resolutions["6_city"].(string)
	demandCell := resolutions["7_demand"].(string)
	surgeCell := resolutions["8_surge"].(string)
	matchingCell := resolutions["9_matching"].(string)

	// H3 cells are 15 characters (hex) or start with 8
	assert.Greater(t, len(cityCell), 10)
	assert.Greater(t, len(demandCell), 10)
	assert.Greater(t, len(surgeCell), 10)
	assert.Greater(t, len(matchingCell), 10)
}

func TestHandler_UpdateLocation_AppErrorHandling(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	driverID := uuid.New()

	// Return an error that triggers AppError handling
	mockRedis.On("SetWithExpiration", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("redis error"))

	reqBody := map[string]interface{}{
		"latitude":  37.7749,
		"longitude": -122.4194,
	}

	c, w := setupTestContext("POST", "/api/v1/geo/location", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateLocation(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_GetDriverLocation_AppErrorHandling(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	driverID := uuid.New()

	// Return not found error - this triggers the AppError path
	mockRedis.On("GetString", mock.Anything, "driver:location:"+driverID.String()).
		Return("", errors.New("key not found"))

	c, w := setupTestContext("GET", "/api/v1/geo/drivers/"+driverID.String()+"/location", nil)
	c.Params = gin.Params{{Key: "id", Value: driverID.String()}}

	handler.GetDriverLocation(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_MultipleDriversWithDifferentStatuses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	handler := NewHandler(service, nil)

	driver1ID := uuid.New()
	driver2ID := uuid.New()
	driver3ID := uuid.New()

	location1 := createTestDriverLocation(driver1ID, 37.7750, -122.4195)
	location2 := createTestDriverLocation(driver2ID, 37.7751, -122.4196)
	location3 := createTestDriverLocation(driver3ID, 37.7752, -122.4197)

	location1JSON, _ := json.Marshal(location1)
	location2JSON, _ := json.Marshal(location2)
	location3JSON, _ := json.Marshal(location3)

	// Mock GeoRadius - default limit 20, doubled twice = 80
	mockRedis.On("GeoRadius", mock.Anything, driverGeoIndexKey, mock.Anything, mock.Anything, searchRadiusKm, 80).
		Return([]string{driver1ID.String(), driver2ID.String(), driver3ID.String()}, nil)

	// Mock MGetStrings batch fetch for driver locations
	mockRedis.On("MGetStrings", mock.Anything, mock.Anything).
		Return([]string{string(location1JSON), string(location2JSON), string(location3JSON)}, nil).Once()

	// Mock MGetStrings batch fetch for driver statuses - only driver1 and driver3 are available
	mockRedis.On("MGetStrings", mock.Anything, mock.Anything).
		Return([]string{`{"status":"available"}`, `{"status":"busy"}`, `{"status":"available"}`}, nil).Once()

	c, w := setupTestContextWithQuery("GET", "/api/v1/geo/drivers/nearby", map[string]string{
		"latitude":  "37.7749",
		"longitude": "-122.4194",
	})

	handler.FindNearbyDrivers(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	drivers := data["drivers"].([]interface{})
	// Only available drivers should be returned
	assert.Equal(t, 2, len(drivers))
}
