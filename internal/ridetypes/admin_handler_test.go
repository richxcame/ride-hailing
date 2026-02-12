package ridetypes

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
// Mock Repository for Admin Handler Tests
// ============================================================================

type MockRepoAdmin struct {
	mock.Mock
}

func (m *MockRepoAdmin) CreateRideType(ctx context.Context, rt *RideType) error {
	args := m.Called(ctx, rt)
	return args.Error(0)
}

func (m *MockRepoAdmin) GetRideTypeByID(ctx context.Context, id uuid.UUID) (*RideType, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RideType), args.Error(1)
}

func (m *MockRepoAdmin) ListRideTypes(ctx context.Context, limit, offset int, includeInactive bool) ([]*RideType, int64, error) {
	args := m.Called(ctx, limit, offset, includeInactive)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*RideType), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepoAdmin) UpdateRideType(ctx context.Context, rt *RideType) error {
	args := m.Called(ctx, rt)
	return args.Error(0)
}

func (m *MockRepoAdmin) DeleteRideType(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepoAdmin) AddRideTypeToCountry(ctx context.Context, crt *CountryRideType) error {
	args := m.Called(ctx, crt)
	return args.Error(0)
}

func (m *MockRepoAdmin) UpdateCountryRideType(ctx context.Context, crt *CountryRideType) error {
	args := m.Called(ctx, crt)
	return args.Error(0)
}

func (m *MockRepoAdmin) RemoveRideTypeFromCountry(ctx context.Context, countryID, rideTypeID uuid.UUID) error {
	args := m.Called(ctx, countryID, rideTypeID)
	return args.Error(0)
}

func (m *MockRepoAdmin) ListCountryRideTypes(ctx context.Context, countryID uuid.UUID, includeInactive bool) ([]*CountryRideTypeWithDetails, error) {
	args := m.Called(ctx, countryID, includeInactive)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*CountryRideTypeWithDetails), args.Error(1)
}

func (m *MockRepoAdmin) AddRideTypeToCity(ctx context.Context, crt *CityRideType) error {
	args := m.Called(ctx, crt)
	return args.Error(0)
}

func (m *MockRepoAdmin) UpdateCityRideType(ctx context.Context, crt *CityRideType) error {
	args := m.Called(ctx, crt)
	return args.Error(0)
}

func (m *MockRepoAdmin) RemoveRideTypeFromCity(ctx context.Context, cityID, rideTypeID uuid.UUID) error {
	args := m.Called(ctx, cityID, rideTypeID)
	return args.Error(0)
}

func (m *MockRepoAdmin) ListCityRideTypes(ctx context.Context, cityID uuid.UUID, includeInactive bool) ([]*CityRideTypeWithDetails, error) {
	args := m.Called(ctx, cityID, includeInactive)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*CityRideTypeWithDetails), args.Error(1)
}

func (m *MockRepoAdmin) GetAvailableRideTypes(ctx context.Context, countryID, cityID *uuid.UUID) ([]*RideType, error) {
	args := m.Called(ctx, countryID, cityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RideType), args.Error(1)
}

// ============================================================================
// Helpers
// ============================================================================

func setupAdminTestContext(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

func parseAdminResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

func createAdminTestHandler(mockRepo *MockRepoAdmin) *AdminHandler {
	service := NewService(mockRepo, nil)
	return NewAdminHandler(service)
}

func createAdminTestRideType() *RideType {
	desc := "Affordable everyday rides"
	icon := "car-economy"
	return &RideType{
		ID:          uuid.New(),
		Name:        "Economy",
		Description: &desc,
		Icon:        &icon,
		Capacity:    4,
		SortOrder:   0,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// ============================================================================
// ListRideTypes Handler Tests
// ============================================================================

func TestAdminHandler_ListRideTypes_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	rideTypes := []*RideType{createAdminTestRideType(), createAdminTestRideType()}

	mockRepo.On("ListRideTypes", mock.Anything, 20, 0, false).
		Return(rideTypes, int64(2), nil)

	c, w := setupAdminTestContext("GET", "/api/v1/admin/ride-types", nil)

	handler.ListRideTypes(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseAdminResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestAdminHandler_ListRideTypes_WithInactive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	rideTypes := []*RideType{createAdminTestRideType()}

	mockRepo.On("ListRideTypes", mock.Anything, 20, 0, true).
		Return(rideTypes, int64(1), nil)

	c, w := setupAdminTestContext("GET", "/api/v1/admin/ride-types?include_inactive=true", nil)
	c.Request.URL.RawQuery = "include_inactive=true"

	handler.ListRideTypes(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestAdminHandler_ListRideTypes_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	mockRepo.On("ListRideTypes", mock.Anything, 20, 0, false).
		Return(nil, int64(0), errors.New("db error"))

	c, w := setupAdminTestContext("GET", "/api/v1/admin/ride-types", nil)

	handler.ListRideTypes(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseAdminResponse(w)
	assert.False(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// CreateRideType Handler Tests
// ============================================================================

func TestAdminHandler_CreateRideType_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	body := CreateRideTypeRequest{
		Name:     "Economy",
		Capacity: 4,
		IsActive: true,
	}

	mockRepo.On("CreateRideType", mock.Anything, mock.AnythingOfType("*ridetypes.RideType")).
		Return(nil)

	c, w := setupAdminTestContext("POST", "/api/v1/admin/ride-types", body)

	handler.CreateRideType(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseAdminResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestAdminHandler_CreateRideType_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	// Missing required fields
	body := map[string]interface{}{
		"description": "test",
	}

	c, w := setupAdminTestContext("POST", "/api/v1/admin/ride-types", body)

	handler.CreateRideType(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdminHandler_CreateRideType_RepoError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	body := CreateRideTypeRequest{
		Name:     "Economy",
		Capacity: 4,
		IsActive: true,
	}

	mockRepo.On("CreateRideType", mock.Anything, mock.AnythingOfType("*ridetypes.RideType")).
		Return(errors.New("db error"))

	c, w := setupAdminTestContext("POST", "/api/v1/admin/ride-types", body)

	handler.CreateRideType(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetRideType Handler Tests
// ============================================================================

func TestAdminHandler_GetRideType_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	rt := createAdminTestRideType()

	mockRepo.On("GetRideTypeByID", mock.Anything, rt.ID).Return(rt, nil)

	c, w := setupAdminTestContext("GET", "/api/v1/admin/ride-types/"+rt.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rt.ID.String()}}

	handler.GetRideType(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseAdminResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestAdminHandler_GetRideType_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	c, w := setupAdminTestContext("GET", "/api/v1/admin/ride-types/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.GetRideType(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdminHandler_GetRideType_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	id := uuid.New()

	mockRepo.On("GetRideTypeByID", mock.Anything, id).Return(nil, errors.New("not found"))

	c, w := setupAdminTestContext("GET", "/api/v1/admin/ride-types/"+id.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	handler.GetRideType(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// UpdateRideType Handler Tests
// ============================================================================

func TestAdminHandler_UpdateRideType_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	rt := createAdminTestRideType()
	newName := "Premium"
	body := UpdateRideTypeRequest{
		Name: &newName,
	}

	mockRepo.On("GetRideTypeByID", mock.Anything, rt.ID).Return(rt, nil)
	mockRepo.On("UpdateRideType", mock.Anything, rt).Return(nil)

	c, w := setupAdminTestContext("PUT", "/api/v1/admin/ride-types/"+rt.ID.String(), body)
	c.Params = gin.Params{{Key: "id", Value: rt.ID.String()}}

	handler.UpdateRideType(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseAdminResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestAdminHandler_UpdateRideType_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	id := uuid.New()
	newName := "Premium"
	body := UpdateRideTypeRequest{Name: &newName}

	mockRepo.On("GetRideTypeByID", mock.Anything, id).Return(nil, errors.New("not found"))

	c, w := setupAdminTestContext("PUT", "/api/v1/admin/ride-types/"+id.String(), body)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	handler.UpdateRideType(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// DeleteRideType Handler Tests
// ============================================================================

func TestAdminHandler_DeleteRideType_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	id := uuid.New()

	mockRepo.On("DeleteRideType", mock.Anything, id).Return(nil)

	c, w := setupAdminTestContext("DELETE", "/api/v1/admin/ride-types/"+id.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	handler.DeleteRideType(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseAdminResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestAdminHandler_DeleteRideType_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	c, w := setupAdminTestContext("DELETE", "/api/v1/admin/ride-types/bad-id", nil)
	c.Params = gin.Params{{Key: "id", Value: "bad-id"}}

	handler.DeleteRideType(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdminHandler_DeleteRideType_RepoError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	id := uuid.New()

	mockRepo.On("DeleteRideType", mock.Anything, id).Return(errors.New("db error"))

	c, w := setupAdminTestContext("DELETE", "/api/v1/admin/ride-types/"+id.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	handler.DeleteRideType(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// Country Ride Type Handler Tests
// ============================================================================

func TestAdminHandler_ListCountryRideTypes_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	countryID := uuid.New()
	items := []*CountryRideTypeWithDetails{
		{
			CountryRideType: CountryRideType{
				CountryID:  countryID,
				RideTypeID: uuid.New(),
				IsActive:   true,
			},
			RideTypeName:     "Economy",
			RideTypeCapacity: 4,
		},
	}

	mockRepo.On("ListCountryRideTypes", mock.Anything, countryID, false).Return(items, nil)

	c, w := setupAdminTestContext("GET", "/api/v1/admin/countries/"+countryID.String()+"/ride-types", nil)
	c.Params = gin.Params{{Key: "countryId", Value: countryID.String()}}

	handler.ListCountryRideTypes(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseAdminResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestAdminHandler_ListCountryRideTypes_InvalidCountryID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	c, w := setupAdminTestContext("GET", "/api/v1/admin/countries/bad-id/ride-types", nil)
	c.Params = gin.Params{{Key: "countryId", Value: "bad-id"}}

	handler.ListCountryRideTypes(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdminHandler_AddRideTypeToCountry_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	countryID := uuid.New()
	body := CountryRideTypeRequest{
		RideTypeID: uuid.New(),
		IsActive:   true,
		SortOrder:  0,
	}

	mockRepo.On("AddRideTypeToCountry", mock.Anything, mock.AnythingOfType("*ridetypes.CountryRideType")).
		Return(nil)

	c, w := setupAdminTestContext("POST", "/api/v1/admin/countries/"+countryID.String()+"/ride-types", body)
	c.Params = gin.Params{{Key: "countryId", Value: countryID.String()}}

	handler.AddRideTypeToCountry(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseAdminResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestAdminHandler_RemoveRideTypeFromCountry_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	countryID := uuid.New()
	rideTypeID := uuid.New()

	mockRepo.On("RemoveRideTypeFromCountry", mock.Anything, countryID, rideTypeID).Return(nil)

	c, w := setupAdminTestContext("DELETE", "/api/v1/admin/countries/"+countryID.String()+"/ride-types/"+rideTypeID.String(), nil)
	c.Params = gin.Params{
		{Key: "countryId", Value: countryID.String()},
		{Key: "rideTypeId", Value: rideTypeID.String()},
	}

	handler.RemoveRideTypeFromCountry(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// City Ride Type Handler Tests
// ============================================================================

func TestAdminHandler_ListCityRideTypes_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	cityID := uuid.New()
	items := []*CityRideTypeWithDetails{
		{
			CityRideType: CityRideType{
				CityID:     cityID,
				RideTypeID: uuid.New(),
				IsActive:   true,
			},
			RideTypeName:     "Premium",
			RideTypeCapacity: 4,
		},
	}

	mockRepo.On("ListCityRideTypes", mock.Anything, cityID, false).Return(items, nil)

	c, w := setupAdminTestContext("GET", "/api/v1/admin/cities/"+cityID.String()+"/ride-types", nil)
	c.Params = gin.Params{{Key: "cityId", Value: cityID.String()}}

	handler.ListCityRideTypes(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseAdminResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestAdminHandler_AddRideTypeToCity_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	cityID := uuid.New()
	body := CityRideTypeRequest{
		RideTypeID: uuid.New(),
		IsActive:   true,
		SortOrder:  0,
	}

	mockRepo.On("AddRideTypeToCity", mock.Anything, mock.AnythingOfType("*ridetypes.CityRideType")).
		Return(nil)

	c, w := setupAdminTestContext("POST", "/api/v1/admin/cities/"+cityID.String()+"/ride-types", body)
	c.Params = gin.Params{{Key: "cityId", Value: cityID.String()}}

	handler.AddRideTypeToCity(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestAdminHandler_RemoveRideTypeFromCity_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	cityID := uuid.New()
	rideTypeID := uuid.New()

	mockRepo.On("RemoveRideTypeFromCity", mock.Anything, cityID, rideTypeID).Return(nil)

	c, w := setupAdminTestContext("DELETE", "/api/v1/admin/cities/"+cityID.String()+"/ride-types/"+rideTypeID.String(), nil)
	c.Params = gin.Params{
		{Key: "cityId", Value: cityID.String()},
		{Key: "rideTypeId", Value: rideTypeID.String()},
	}

	handler.RemoveRideTypeFromCity(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestAdminHandler_ListCityRideTypes_InvalidCityID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepoAdmin)
	handler := createAdminTestHandler(mockRepo)

	c, w := setupAdminTestContext("GET", "/api/v1/admin/cities/invalid/ride-types", nil)
	c.Params = gin.Params{{Key: "cityId", Value: "invalid"}}

	handler.ListCityRideTypes(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
