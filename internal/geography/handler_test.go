package geography

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// Mock Implementations
// ============================================================================

// MockRepository implements RepositoryInterface for testing
type MockRepoHandler struct {
	mock.Mock
}

func (m *MockRepoHandler) GetActiveCountries(ctx context.Context) ([]*Country, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Country), args.Error(1)
}

func (m *MockRepoHandler) GetCountryByCode(ctx context.Context, code string) (*Country, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Country), args.Error(1)
}

func (m *MockRepoHandler) GetCountryByID(ctx context.Context, id uuid.UUID) (*Country, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Country), args.Error(1)
}

func (m *MockRepoHandler) GetRegionsByCountry(ctx context.Context, countryID uuid.UUID) ([]*Region, error) {
	args := m.Called(ctx, countryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Region), args.Error(1)
}

func (m *MockRepoHandler) GetRegionByID(ctx context.Context, id uuid.UUID) (*Region, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Region), args.Error(1)
}

func (m *MockRepoHandler) GetCitiesByRegion(ctx context.Context, regionID uuid.UUID) ([]*City, error) {
	args := m.Called(ctx, regionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*City), args.Error(1)
}

func (m *MockRepoHandler) GetCityByID(ctx context.Context, id uuid.UUID) (*City, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*City), args.Error(1)
}

func (m *MockRepoHandler) ResolveLocation(ctx context.Context, lat, lng float64) (*ResolvedLocation, error) {
	args := m.Called(ctx, lat, lng)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ResolvedLocation), args.Error(1)
}

func (m *MockRepoHandler) FindNearestCity(ctx context.Context, lat, lng float64, maxDistanceKm float64) (*City, error) {
	args := m.Called(ctx, lat, lng, maxDistanceKm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*City), args.Error(1)
}

func (m *MockRepoHandler) GetPricingZonesByCity(ctx context.Context, cityID uuid.UUID) ([]*PricingZone, error) {
	args := m.Called(ctx, cityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*PricingZone), args.Error(1)
}

func (m *MockRepoHandler) GetDriverRegions(ctx context.Context, driverID uuid.UUID) ([]*DriverRegion, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DriverRegion), args.Error(1)
}

func (m *MockRepoHandler) AddDriverRegion(ctx context.Context, driverID, regionID uuid.UUID, isPrimary bool) error {
	args := m.Called(ctx, driverID, regionID, isPrimary)
	return args.Error(0)
}

func (m *MockRepoHandler) CreateCountry(ctx context.Context, country *Country) error {
	args := m.Called(ctx, country)
	return args.Error(0)
}

func (m *MockRepoHandler) CreateRegion(ctx context.Context, region *Region) error {
	args := m.Called(ctx, region)
	return args.Error(0)
}

func (m *MockRepoHandler) CreateCity(ctx context.Context, city *City) error {
	args := m.Called(ctx, city)
	return args.Error(0)
}

func (m *MockRepoHandler) CreatePricingZone(ctx context.Context, zone *PricingZone) error {
	args := m.Called(ctx, zone)
	return args.Error(0)
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

func createTestCountryHandler() *Country {
	return &Country{
		ID:              uuid.New(),
		Code:            "US",
		Code3:           "USA",
		Name:            "United States",
		CurrencyCode:    "USD",
		DefaultLanguage: "en",
		Timezone:        "America/New_York",
		PhonePrefix:     "+1",
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func createTestRegionHandler(country *Country) *Region {
	return &Region{
		ID:        uuid.New(),
		CountryID: country.ID,
		Code:      "NY",
		Name:      "New York",
		IsActive:  true,
		Country:   country,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func createTestCityHandler(region *Region) *City {
	return &City{
		ID:              uuid.New(),
		RegionID:        region.ID,
		Name:            "New York City",
		CenterLatitude:  40.7128,
		CenterLongitude: -74.0060,
		IsActive:        true,
		Region:          region,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func createTestPricingZoneHandler(city *City) *PricingZone {
	return &PricingZone{
		ID:              uuid.New(),
		CityID:          city.ID,
		Name:            "JFK Airport",
		ZoneType:        ZoneTypeAirport,
		CenterLatitude:  40.6413,
		CenterLongitude: -73.7781,
		Priority:        100,
		IsActive:        true,
		City:            city,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func createTestHandler(mockRepo *MockRepoHandler) *Handler {
	service := NewService(mockRepo)
	return NewHandler(service)
}

// ============================================================================
// GetCountries Handler Tests
// ============================================================================

func TestHandler_GetCountries_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	countries := []*Country{
		createTestCountryHandler(),
		{
			ID:           uuid.New(),
			Code:         "CA",
			Name:         "Canada",
			CurrencyCode: "CAD",
			IsActive:     true,
		},
	}

	mockRepo.On("GetActiveCountries", mock.Anything).Return(countries, nil)

	c, w := setupTestContext("GET", "/api/v1/geography/countries", nil)

	handler.GetCountries(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetCountries_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	mockRepo.On("GetActiveCountries", mock.Anything).Return([]*Country{}, nil)

	c, w := setupTestContext("GET", "/api/v1/geography/countries", nil)

	handler.GetCountries(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetCountries_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	mockRepo.On("GetActiveCountries", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/geography/countries", nil)

	handler.GetCountries(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

// ============================================================================
// GetCountry Handler Tests
// ============================================================================

func TestHandler_GetCountry_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	country := createTestCountryHandler()

	mockRepo.On("GetCountryByCode", mock.Anything, "US").Return(country, nil)

	c, w := setupTestContext("GET", "/api/v1/geography/countries/US", nil)
	c.Params = gin.Params{{Key: "code", Value: "US"}}

	handler.GetCountry(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetCountry_InvalidCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/geography/countries/USA", nil)
	c.Params = gin.Params{{Key: "code", Value: "USA"}} // 3 letters instead of 2

	handler.GetCountry(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid country code")
}

func TestHandler_GetCountry_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	mockRepo.On("GetCountryByCode", mock.Anything, "XX").Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/geography/countries/XX", nil)
	c.Params = gin.Params{{Key: "code", Value: "XX"}}

	handler.GetCountry(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "country not found")
}

// ============================================================================
// GetRegionsByCountry Handler Tests
// ============================================================================

func TestHandler_GetRegionsByCountry_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	country := createTestCountryHandler()
	regions := []*Region{
		createTestRegionHandler(country),
		{
			ID:        uuid.New(),
			CountryID: country.ID,
			Code:      "CA",
			Name:      "California",
			IsActive:  true,
			Country:   country,
		},
	}

	mockRepo.On("GetCountryByCode", mock.Anything, "US").Return(country, nil)
	mockRepo.On("GetRegionsByCountry", mock.Anything, country.ID).Return(regions, nil)

	c, w := setupTestContext("GET", "/api/v1/geography/countries/US/regions", nil)
	c.Params = gin.Params{{Key: "code", Value: "US"}}

	handler.GetRegionsByCountry(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetRegionsByCountry_InvalidCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/geography/countries/X/regions", nil)
	c.Params = gin.Params{{Key: "code", Value: "X"}} // 1 letter instead of 2

	handler.GetRegionsByCountry(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetRegionsByCountry_CountryNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	mockRepo.On("GetCountryByCode", mock.Anything, "XX").Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/geography/countries/XX/regions", nil)
	c.Params = gin.Params{{Key: "code", Value: "XX"}}

	handler.GetRegionsByCountry(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ============================================================================
// GetRegion Handler Tests
// ============================================================================

func TestHandler_GetRegion_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	country := createTestCountryHandler()
	region := createTestRegionHandler(country)

	mockRepo.On("GetRegionByID", mock.Anything, region.ID).Return(region, nil)

	c, w := setupTestContext("GET", "/api/v1/geography/regions/"+region.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: region.ID.String()}}

	handler.GetRegion(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetRegion_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/geography/regions/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.GetRegion(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid region ID")
}

func TestHandler_GetRegion_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	regionID := uuid.New()

	mockRepo.On("GetRegionByID", mock.Anything, regionID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/geography/regions/"+regionID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: regionID.String()}}

	handler.GetRegion(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ============================================================================
// GetCitiesByRegion Handler Tests
// ============================================================================

func TestHandler_GetCitiesByRegion_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	country := createTestCountryHandler()
	region := createTestRegionHandler(country)
	cities := []*City{
		createTestCityHandler(region),
		{
			ID:       uuid.New(),
			RegionID: region.ID,
			Name:     "Buffalo",
			IsActive: true,
			Region:   region,
		},
	}

	mockRepo.On("GetCitiesByRegion", mock.Anything, region.ID).Return(cities, nil)

	c, w := setupTestContext("GET", "/api/v1/geography/regions/"+region.ID.String()+"/cities", nil)
	c.Params = gin.Params{{Key: "id", Value: region.ID.String()}}

	handler.GetCitiesByRegion(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetCitiesByRegion_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/geography/regions/invalid-uuid/cities", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.GetCitiesByRegion(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetCitiesByRegion_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	regionID := uuid.New()

	mockRepo.On("GetCitiesByRegion", mock.Anything, regionID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/geography/regions/"+regionID.String()+"/cities", nil)
	c.Params = gin.Params{{Key: "id", Value: regionID.String()}}

	handler.GetCitiesByRegion(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetCity Handler Tests
// ============================================================================

func TestHandler_GetCity_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	country := createTestCountryHandler()
	region := createTestRegionHandler(country)
	city := createTestCityHandler(region)

	mockRepo.On("GetCityByID", mock.Anything, city.ID).Return(city, nil)

	c, w := setupTestContext("GET", "/api/v1/geography/cities/"+city.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: city.ID.String()}}

	handler.GetCity(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetCity_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/geography/cities/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.GetCity(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid city ID")
}

func TestHandler_GetCity_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	cityID := uuid.New()

	mockRepo.On("GetCityByID", mock.Anything, cityID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/geography/cities/"+cityID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: cityID.String()}}

	handler.GetCity(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ============================================================================
// GetPricingZones Handler Tests
// ============================================================================

func TestHandler_GetPricingZones_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	country := createTestCountryHandler()
	region := createTestRegionHandler(country)
	city := createTestCityHandler(region)
	zones := []*PricingZone{
		createTestPricingZoneHandler(city),
		{
			ID:       uuid.New(),
			CityID:   city.ID,
			Name:     "Downtown Manhattan",
			ZoneType: ZoneTypeDowntown,
			Priority: 50,
			IsActive: true,
		},
	}

	mockRepo.On("GetPricingZonesByCity", mock.Anything, city.ID).Return(zones, nil)

	c, w := setupTestContext("GET", "/api/v1/geography/cities/"+city.ID.String()+"/zones", nil)
	c.Params = gin.Params{{Key: "cityId", Value: city.ID.String()}}

	handler.GetPricingZones(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetPricingZones_InvalidCityID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/geography/cities/invalid-uuid/zones", nil)
	c.Params = gin.Params{{Key: "cityId", Value: "invalid-uuid"}}

	handler.GetPricingZones(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid city ID")
}

func TestHandler_GetPricingZones_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	cityID := uuid.New()

	mockRepo.On("GetPricingZonesByCity", mock.Anything, cityID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/geography/cities/"+cityID.String()+"/zones", nil)
	c.Params = gin.Params{{Key: "cityId", Value: cityID.String()}}

	handler.GetPricingZones(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// ResolveLocation Handler Tests
// ============================================================================

func TestHandler_ResolveLocation_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	country := createTestCountryHandler()
	region := createTestRegionHandler(country)
	city := createTestCityHandler(region)

	resolved := &ResolvedLocation{
		Location: Location{Latitude: 40.7128, Longitude: -74.0060},
		Country:  country,
		Region:   region,
		City:     city,
		Timezone: "America/New_York",
	}

	mockRepo.On("ResolveLocation", mock.Anything, 40.7128, -74.0060).Return(resolved, nil)

	reqBody := ResolveLocationRequest{
		Latitude:  40.7128,
		Longitude: -74.0060,
	}

	c, w := setupTestContext("POST", "/api/v1/geography/location/resolve", reqBody)

	handler.ResolveLocation(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_ResolveLocation_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("POST", "/api/v1/geography/location/resolve", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/geography/location/resolve", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ResolveLocation(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ResolveLocation_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	mockRepo.On("ResolveLocation", mock.Anything, 40.7128, -74.0060).Return(nil, errors.New("service error"))

	reqBody := ResolveLocationRequest{
		Latitude:  40.7128,
		Longitude: -74.0060,
	}

	c, w := setupTestContext("POST", "/api/v1/geography/location/resolve", reqBody)

	handler.ResolveLocation(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// CheckServiceability Handler Tests
// ============================================================================

func TestHandler_CheckServiceability_Serviceable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	country := createTestCountryHandler()
	region := createTestRegionHandler(country)
	city := createTestCityHandler(region)

	resolved := &ResolvedLocation{
		Location: Location{Latitude: 40.7128, Longitude: -74.0060},
		Country:  country,
		Region:   region,
		City:     city,
		Timezone: "America/New_York",
	}

	mockRepo.On("ResolveLocation", mock.Anything, 40.7128, -74.0060).Return(resolved, nil)

	reqBody := ResolveLocationRequest{
		Latitude:  40.7128,
		Longitude: -74.0060,
	}

	c, w := setupTestContext("POST", "/api/v1/geography/location/serviceable", reqBody)

	handler.CheckServiceability(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.True(t, data["serviceable"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_CheckServiceability_NotServiceable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	// Use remote location (middle of Pacific ocean) - not serviceable
	lat, lng := 0.001, 0.001 // Non-zero to pass required validation

	resolved := &ResolvedLocation{
		Location: Location{Latitude: lat, Longitude: lng},
		City:     nil, // No city found
		Timezone: "",
	}

	// Service calls repo.ResolveLocation then repo.FindNearestCity if City is nil
	mockRepo.On("ResolveLocation", mock.Anything, lat, lng).Return(resolved, nil)
	mockRepo.On("FindNearestCity", mock.Anything, lat, lng, 50.0).Return(nil, nil) // No nearby city

	reqBody := ResolveLocationRequest{
		Latitude:  lat,
		Longitude: lng,
	}

	c, w := setupTestContext("POST", "/api/v1/geography/location/serviceable", reqBody)

	handler.CheckServiceability(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.False(t, data["serviceable"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_CheckServiceability_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("POST", "/api/v1/geography/location/serviceable", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/geography/location/serviceable", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CheckServiceability(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CheckServiceability_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	mockRepo.On("ResolveLocation", mock.Anything, 40.7128, -74.0060).Return(nil, errors.New("service error"))

	reqBody := ResolveLocationRequest{
		Latitude:  40.7128,
		Longitude: -74.0060,
	}

	c, w := setupTestContext("POST", "/api/v1/geography/location/serviceable", reqBody)

	handler.CheckServiceability(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_GetCountry_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		setupMock      func(*MockRepoHandler)
		expectedStatus int
	}{
		{
			name: "valid country code US",
			code: "US",
			setupMock: func(m *MockRepoHandler) {
				m.On("GetCountryByCode", mock.Anything, "US").Return(createTestCountryHandler(), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid country code - too long",
			code:           "USA",
			setupMock:      func(m *MockRepoHandler) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid country code - too short",
			code:           "U",
			setupMock:      func(m *MockRepoHandler) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "country not found",
			code: "XX",
			setupMock: func(m *MockRepoHandler) {
				m.On("GetCountryByCode", mock.Anything, "XX").Return(nil, errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepoHandler)
			handler := createTestHandler(mockRepo)

			tt.setupMock(mockRepo)

			c, w := setupTestContext("GET", "/api/v1/geography/countries/"+tt.code, nil)
			c.Params = gin.Params{{Key: "code", Value: tt.code}}

			handler.GetCountry(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestHandler_GetRegion_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		setupMock      func(*MockRepoHandler, uuid.UUID)
		expectedStatus int
	}{
		{
			name: "valid region ID",
			id:   uuid.New().String(),
			setupMock: func(m *MockRepoHandler, id uuid.UUID) {
				country := createTestCountryHandler()
				region := createTestRegionHandler(country)
				region.ID = id
				m.On("GetRegionByID", mock.Anything, id).Return(region, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid region ID format",
			id:             "not-a-uuid",
			setupMock:      func(m *MockRepoHandler, id uuid.UUID) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "region not found",
			id:   uuid.New().String(),
			setupMock: func(m *MockRepoHandler, id uuid.UUID) {
				m.On("GetRegionByID", mock.Anything, id).Return(nil, errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepoHandler)
			handler := createTestHandler(mockRepo)

			var id uuid.UUID
			if parsed, err := uuid.Parse(tt.id); err == nil {
				id = parsed
			}

			tt.setupMock(mockRepo, id)

			c, w := setupTestContext("GET", "/api/v1/geography/regions/"+tt.id, nil)
			c.Params = gin.Params{{Key: "id", Value: tt.id}}

			handler.GetRegion(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestHandler_ResolveLocation_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		request        interface{}
		setupMock      func(*MockRepoHandler)
		expectedStatus int
	}{
		{
			name: "valid location in service area",
			request: ResolveLocationRequest{
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			setupMock: func(m *MockRepoHandler) {
				country := createTestCountryHandler()
				region := createTestRegionHandler(country)
				city := createTestCityHandler(region)
				resolved := &ResolvedLocation{
					Location: Location{Latitude: 40.7128, Longitude: -74.0060},
					Country:  country,
					Region:   region,
					City:     city,
					Timezone: "America/New_York",
				}
				m.On("ResolveLocation", mock.Anything, 40.7128, -74.0060).Return(resolved, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing latitude",
			request:        map[string]interface{}{"longitude": -74.0060},
			setupMock:      func(m *MockRepoHandler) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing longitude",
			request:        map[string]interface{}{"latitude": 40.7128},
			setupMock:      func(m *MockRepoHandler) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service error",
			request: ResolveLocationRequest{
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			setupMock: func(m *MockRepoHandler) {
				m.On("ResolveLocation", mock.Anything, 40.7128, -74.0060).Return(nil, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepoHandler)
			handler := createTestHandler(mockRepo)

			tt.setupMock(mockRepo)

			c, w := setupTestContext("POST", "/api/v1/geography/location/resolve", tt.request)

			handler.ResolveLocation(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_GetCountries_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	countries := []*Country{createTestCountryHandler()}
	mockRepo.On("GetActiveCountries", mock.Anything).Return(countries, nil)

	c, w := setupTestContext("GET", "/api/v1/geography/countries", nil)

	handler.GetCountries(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	// Verify response structure
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	mockRepo.On("GetActiveCountries", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/geography/countries", nil)

	handler.GetCountries(c)

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

func TestHandler_GetCity_EmptyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/geography/cities/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.GetCity(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetPricingZones_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createTestHandler(mockRepo)

	cityID := uuid.New()

	mockRepo.On("GetPricingZonesByCity", mock.Anything, cityID).Return([]*PricingZone{}, nil)

	c, w := setupTestContext("GET", "/api/v1/geography/cities/"+cityID.String()+"/zones", nil)
	c.Params = gin.Params{{Key: "cityId", Value: cityID.String()}}

	handler.GetPricingZones(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}
