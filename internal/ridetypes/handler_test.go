package ridetypes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// Mock Repository for Handler Tests
// ============================================================================

type MockRepoHandler struct {
	mock.Mock
}

func (m *MockRepoHandler) CreateRideType(ctx context.Context, rt *RideType) error {
	args := m.Called(ctx, rt)
	return args.Error(0)
}

func (m *MockRepoHandler) GetRideTypeByID(ctx context.Context, id uuid.UUID) (*RideType, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RideType), args.Error(1)
}

func (m *MockRepoHandler) ListRideTypes(ctx context.Context, limit, offset int, includeInactive bool) ([]*RideType, int64, error) {
	args := m.Called(ctx, limit, offset, includeInactive)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*RideType), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepoHandler) UpdateRideType(ctx context.Context, rt *RideType) error {
	args := m.Called(ctx, rt)
	return args.Error(0)
}

func (m *MockRepoHandler) DeleteRideType(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepoHandler) AddRideTypeToCountry(ctx context.Context, crt *CountryRideType) error {
	args := m.Called(ctx, crt)
	return args.Error(0)
}

func (m *MockRepoHandler) UpdateCountryRideType(ctx context.Context, crt *CountryRideType) error {
	args := m.Called(ctx, crt)
	return args.Error(0)
}

func (m *MockRepoHandler) RemoveRideTypeFromCountry(ctx context.Context, countryID, rideTypeID uuid.UUID) error {
	args := m.Called(ctx, countryID, rideTypeID)
	return args.Error(0)
}

func (m *MockRepoHandler) ListCountryRideTypes(ctx context.Context, countryID uuid.UUID, includeInactive bool) ([]*CountryRideTypeWithDetails, error) {
	args := m.Called(ctx, countryID, includeInactive)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*CountryRideTypeWithDetails), args.Error(1)
}

func (m *MockRepoHandler) AddRideTypeToCity(ctx context.Context, crt *CityRideType) error {
	args := m.Called(ctx, crt)
	return args.Error(0)
}

func (m *MockRepoHandler) UpdateCityRideType(ctx context.Context, crt *CityRideType) error {
	args := m.Called(ctx, crt)
	return args.Error(0)
}

func (m *MockRepoHandler) RemoveRideTypeFromCity(ctx context.Context, cityID, rideTypeID uuid.UUID) error {
	args := m.Called(ctx, cityID, rideTypeID)
	return args.Error(0)
}

func (m *MockRepoHandler) ListCityRideTypes(ctx context.Context, cityID uuid.UUID, includeInactive bool) ([]*CityRideTypeWithDetails, error) {
	args := m.Called(ctx, cityID, includeInactive)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*CityRideTypeWithDetails), args.Error(1)
}

func (m *MockRepoHandler) GetAvailableRideTypes(ctx context.Context, countryID, cityID *uuid.UUID) ([]*RideType, error) {
	args := m.Called(ctx, countryID, cityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RideType), args.Error(1)
}

// ============================================================================
// Helpers
// ============================================================================

func setupHandlerTestContext(method, path string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, path, nil)
	c.Request = req
	return c, w
}

func parseHandlerResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

func createMobileTestHandler(mockRepo *MockRepoHandler) *Handler {
	service := NewService(mockRepo, nil)
	return NewHandler(service)
}

// ============================================================================
// GetAvailableRideTypes Handler Tests
// ============================================================================

func TestHandler_GetAvailableRideTypes_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createMobileTestHandler(mockRepo)

	rideTypes := []*RideType{createTestRideType(), createTestRideType()}

	mockRepo.On("GetAvailableRideTypes", mock.Anything, (*uuid.UUID)(nil), (*uuid.UUID)(nil)).
		Return(rideTypes, nil)

	c, w := setupHandlerTestContext("GET", "/api/v1/ride-types/available?latitude=41.2995&longitude=69.2401")
	c.Request.URL.RawQuery = "latitude =41.2995&longitude=69.2401"

	handler.GetAvailableRideTypes(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseHandlerResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetAvailableRideTypes_MissingLat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createMobileTestHandler(mockRepo)

	c, w := setupHandlerTestContext("GET", "/api/v1/ride-types/available?longitude=69.2401")
	c.Request.URL.RawQuery = "longitude =69.2401"

	handler.GetAvailableRideTypes(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseHandlerResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_GetAvailableRideTypes_MissingLng(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createMobileTestHandler(mockRepo)

	c, w := setupHandlerTestContext("GET", "/api/v1/ride-types/available?latitude=41.2995")
	c.Request.URL.RawQuery = "latitude =41.2995"

	handler.GetAvailableRideTypes(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetAvailableRideTypes_MissingBoth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createMobileTestHandler(mockRepo)

	c, w := setupHandlerTestContext("GET", "/api/v1/ride-types/available")

	handler.GetAvailableRideTypes(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetAvailableRideTypes_InvalidLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createMobileTestHandler(mockRepo)

	c, w := setupHandlerTestContext("GET", "/api/v1/ride-types/available?latitude=abc&longitude=69.2401")
	c.Request.URL.RawQuery = "latitude =abc&longitude=69.2401"

	handler.GetAvailableRideTypes(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetAvailableRideTypes_InvalidLongitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createMobileTestHandler(mockRepo)

	c, w := setupHandlerTestContext("GET", "/api/v1/ride-types/available?latitude=41.2995&longitude=xyz")
	c.Request.URL.RawQuery = "latitude =41.2995&longitude=xyz"

	handler.GetAvailableRideTypes(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetAvailableRideTypes_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createMobileTestHandler(mockRepo)

	mockRepo.On("GetAvailableRideTypes", mock.Anything, (*uuid.UUID)(nil), (*uuid.UUID)(nil)).
		Return(nil, errors.New("db error"))

	c, w := setupHandlerTestContext("GET", "/api/v1/ride-types/available?latitude=41.2995&longitude=69.2401")
	c.Request.URL.RawQuery = "latitude =41.2995&longitude=69.2401"

	handler.GetAvailableRideTypes(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseHandlerResponse(w)
	assert.False(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetAvailableRideTypes_EmptyResult(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoHandler)
	handler := createMobileTestHandler(mockRepo)

	mockRepo.On("GetAvailableRideTypes", mock.Anything, (*uuid.UUID)(nil), (*uuid.UUID)(nil)).
		Return([]*RideType{}, nil)

	c, w := setupHandlerTestContext("GET", "/api/v1/ride-types/available?latitude=41.2995&longitude=69.2401")
	c.Request.URL.RawQuery = "latitude =41.2995&longitude=69.2401"

	handler.GetAvailableRideTypes(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseHandlerResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}
