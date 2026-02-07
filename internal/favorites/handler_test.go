package favorites

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
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateFavorite(ctx context.Context, favorite *FavoriteLocation) error {
	args := m.Called(ctx, favorite)
	// Simulate ID generation
	if args.Error(0) == nil && favorite.ID == uuid.Nil {
		favorite.ID = uuid.New()
		favorite.CreatedAt = time.Now()
		favorite.UpdatedAt = time.Now()
	}
	return args.Error(0)
}

func (m *MockRepository) GetFavoriteByID(ctx context.Context, id uuid.UUID) (*FavoriteLocation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FavoriteLocation), args.Error(1)
}

func (m *MockRepository) GetFavoritesByUser(ctx context.Context, userID uuid.UUID) ([]*FavoriteLocation, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*FavoriteLocation), args.Error(1)
}

func (m *MockRepository) UpdateFavorite(ctx context.Context, favorite *FavoriteLocation) error {
	args := m.Called(ctx, favorite)
	return args.Error(0)
}

func (m *MockRepository) DeleteFavorite(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
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

func setUserContext(c *gin.Context, userID uuid.UUID) {
	c.Set("user_id", userID)
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

func createTestHandler(mockRepo *MockRepository) *Handler {
	service := NewService(mockRepo)
	return NewHandler(service)
}

func createTestFavoriteLocation(userID uuid.UUID) *FavoriteLocation {
	return &FavoriteLocation{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "Home",
		Address:   "123 Main St, City, Country",
		Latitude:  40.7128,
		Longitude: -74.0060,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ============================================================================
// CreateFavorite Handler Tests
// ============================================================================

func TestHandler_CreateFavorite_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"name":      "Home",
		"address":   "123 Main St, City, Country",
		"latitude":  40.7128,
		"longitude": -74.0060,
	}

	mockRepo.On("CreateFavorite", mock.Anything, mock.AnythingOfType("*favorites.FavoriteLocation")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/favorites", reqBody)
	setUserContext(c, userID)

	handler.CreateFavorite(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_CreateFavorite_Unauthorized_NoUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := map[string]interface{}{
		"name":      "Home",
		"address":   "123 Main St",
		"latitude":  40.7128,
		"longitude": -74.0060,
	}

	c, w := setupTestContext("POST", "/api/v1/favorites", reqBody)
	// Don't set user context

	handler.CreateFavorite(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_CreateFavorite_InvalidUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := map[string]interface{}{
		"name":      "Home",
		"address":   "123 Main St",
		"latitude":  40.7128,
		"longitude": -74.0060,
	}

	c, w := setupTestContext("POST", "/api/v1/favorites", reqBody)
	c.Set("user_id", "invalid-uuid-type") // Wrong type

	handler.CreateFavorite(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_CreateFavorite_MissingName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"address":   "123 Main St",
		"latitude":  40.7128,
		"longitude": -74.0060,
	}

	c, w := setupTestContext("POST", "/api/v1/favorites", reqBody)
	setUserContext(c, userID)

	handler.CreateFavorite(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateFavorite_MissingAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"name":      "Home",
		"latitude":  40.7128,
		"longitude": -74.0060,
	}

	c, w := setupTestContext("POST", "/api/v1/favorites", reqBody)
	setUserContext(c, userID)

	handler.CreateFavorite(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateFavorite_MissingLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"name":      "Home",
		"address":   "123 Main St",
		"longitude": -74.0060,
	}

	c, w := setupTestContext("POST", "/api/v1/favorites", reqBody)
	setUserContext(c, userID)

	handler.CreateFavorite(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateFavorite_MissingLongitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"name":     "Home",
		"address":  "123 Main St",
		"latitude": 40.7128,
	}

	c, w := setupTestContext("POST", "/api/v1/favorites", reqBody)
	setUserContext(c, userID)

	handler.CreateFavorite(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateFavorite_InvalidLatitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"name":      "Home",
		"address":   "123 Main St",
		"latitude":  100.0, // Invalid: > 90
		"longitude": -74.0060,
	}

	mockRepo.On("CreateFavorite", mock.Anything, mock.AnythingOfType("*favorites.FavoriteLocation")).Return(ErrInvalidCoordinates)

	c, w := setupTestContext("POST", "/api/v1/favorites", reqBody)
	setUserContext(c, userID)

	handler.CreateFavorite(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_CreateFavorite_InvalidLongitude(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"name":      "Home",
		"address":   "123 Main St",
		"latitude":  40.7128,
		"longitude": -200.0, // Invalid: < -180
	}

	mockRepo.On("CreateFavorite", mock.Anything, mock.AnythingOfType("*favorites.FavoriteLocation")).Return(ErrInvalidCoordinates)

	c, w := setupTestContext("POST", "/api/v1/favorites", reqBody)
	setUserContext(c, userID)

	handler.CreateFavorite(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_CreateFavorite_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/favorites", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/favorites", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID)

	handler.CreateFavorite(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateFavorite_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"name":      "Home",
		"address":   "123 Main St",
		"latitude":  40.7128,
		"longitude": -74.0060,
	}

	mockRepo.On("CreateFavorite", mock.Anything, mock.AnythingOfType("*favorites.FavoriteLocation")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/favorites", reqBody)
	setUserContext(c, userID)

	handler.CreateFavorite(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetFavorites Handler Tests
// ============================================================================

func TestHandler_GetFavorites_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	favorites := []*FavoriteLocation{
		createTestFavoriteLocation(userID),
		{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "Work",
			Address:   "456 Office St",
			Latitude:  40.7580,
			Longitude: -73.9855,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	mockRepo.On("GetFavoritesByUser", mock.Anything, userID).Return(favorites, nil)

	c, w := setupTestContext("GET", "/api/v1/favorites", nil)
	setUserContext(c, userID)

	handler.GetFavorites(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetFavorites_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/favorites", nil)
	// Don't set user context

	handler.GetFavorites(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetFavorites_InvalidUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/favorites", nil)
	c.Set("user_id", "invalid-uuid-type") // Wrong type

	handler.GetFavorites(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_GetFavorites_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetFavoritesByUser", mock.Anything, userID).Return([]*FavoriteLocation{}, nil)

	c, w := setupTestContext("GET", "/api/v1/favorites", nil)
	setUserContext(c, userID)

	handler.GetFavorites(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetFavorites_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetFavoritesByUser", mock.Anything, userID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/favorites", nil)
	setUserContext(c, userID)

	handler.GetFavorites(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetFavorite Handler Tests
// ============================================================================

func TestHandler_GetFavorite_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	favorite := createTestFavoriteLocation(userID)

	mockRepo.On("GetFavoriteByID", mock.Anything, favorite.ID).Return(favorite, nil)

	c, w := setupTestContext("GET", "/api/v1/favorites/"+favorite.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: favorite.ID.String()}}
	setUserContext(c, userID)

	handler.GetFavorite(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetFavorite_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	favoriteID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/favorites/"+favoriteID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: favoriteID.String()}}
	// Don't set user context

	handler.GetFavorite(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetFavorite_InvalidUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	favoriteID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/favorites/"+favoriteID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: favoriteID.String()}}
	c.Set("user_id", "invalid-uuid-type") // Wrong type

	handler.GetFavorite(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_GetFavorite_InvalidFavoriteID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/favorites/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID)

	handler.GetFavorite(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "Invalid favorite ID")
}

func TestHandler_GetFavorite_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	favoriteID := uuid.New()

	mockRepo.On("GetFavoriteByID", mock.Anything, favoriteID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/favorites/"+favoriteID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: favoriteID.String()}}
	setUserContext(c, userID)

	handler.GetFavorite(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetFavorite_UnauthorizedAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	otherUserID := uuid.New()
	favorite := createTestFavoriteLocation(otherUserID)

	mockRepo.On("GetFavoriteByID", mock.Anything, favorite.ID).Return(favorite, nil)

	c, w := setupTestContext("GET", "/api/v1/favorites/"+favorite.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: favorite.ID.String()}}
	setUserContext(c, userID) // Different user

	handler.GetFavorite(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "unauthorized")
}

// ============================================================================
// UpdateFavorite Handler Tests
// ============================================================================

func TestHandler_UpdateFavorite_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	favorite := createTestFavoriteLocation(userID)
	reqBody := map[string]interface{}{
		"name":      "Updated Home",
		"address":   "789 New St",
		"latitude":  40.7500,
		"longitude": -73.9900,
	}

	mockRepo.On("GetFavoriteByID", mock.Anything, favorite.ID).Return(favorite, nil)
	mockRepo.On("UpdateFavorite", mock.Anything, mock.AnythingOfType("*favorites.FavoriteLocation")).Return(nil)

	c, w := setupTestContext("PUT", "/api/v1/favorites/"+favorite.ID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: favorite.ID.String()}}
	setUserContext(c, userID)

	handler.UpdateFavorite(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_UpdateFavorite_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	favoriteID := uuid.New()
	reqBody := map[string]interface{}{
		"name":      "Updated",
		"address":   "New Address",
		"latitude":  40.7128,
		"longitude": -74.0060,
	}

	c, w := setupTestContext("PUT", "/api/v1/favorites/"+favoriteID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: favoriteID.String()}}
	// Don't set user context

	handler.UpdateFavorite(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_UpdateFavorite_InvalidUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	favoriteID := uuid.New()
	reqBody := map[string]interface{}{
		"name":      "Updated",
		"address":   "New Address",
		"latitude":  40.7128,
		"longitude": -74.0060,
	}

	c, w := setupTestContext("PUT", "/api/v1/favorites/"+favoriteID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: favoriteID.String()}}
	c.Set("user_id", "invalid-uuid-type") // Wrong type

	handler.UpdateFavorite(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_UpdateFavorite_InvalidFavoriteID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"name":      "Updated",
		"address":   "New Address",
		"latitude":  40.7128,
		"longitude": -74.0060,
	}

	c, w := setupTestContext("PUT", "/api/v1/favorites/invalid-uuid", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID)

	handler.UpdateFavorite(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateFavorite_MissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	favoriteID := uuid.New()
	reqBody := map[string]interface{}{
		"name": "Updated",
		// Missing other required fields
	}

	c, w := setupTestContext("PUT", "/api/v1/favorites/"+favoriteID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: favoriteID.String()}}
	setUserContext(c, userID)

	handler.UpdateFavorite(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateFavorite_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	favoriteID := uuid.New()
	reqBody := map[string]interface{}{
		"name":      "Updated",
		"address":   "New Address",
		"latitude":  40.7128,
		"longitude": -74.0060,
	}

	mockRepo.On("GetFavoriteByID", mock.Anything, favoriteID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("PUT", "/api/v1/favorites/"+favoriteID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: favoriteID.String()}}
	setUserContext(c, userID)

	handler.UpdateFavorite(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_UpdateFavorite_UnauthorizedAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	otherUserID := uuid.New()
	favorite := createTestFavoriteLocation(otherUserID)
	reqBody := map[string]interface{}{
		"name":      "Updated",
		"address":   "New Address",
		"latitude":  40.7128,
		"longitude": -74.0060,
	}

	mockRepo.On("GetFavoriteByID", mock.Anything, favorite.ID).Return(favorite, nil)

	c, w := setupTestContext("PUT", "/api/v1/favorites/"+favorite.ID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: favorite.ID.String()}}
	setUserContext(c, userID) // Different user

	handler.UpdateFavorite(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_UpdateFavorite_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	favorite := createTestFavoriteLocation(userID)
	reqBody := map[string]interface{}{
		"name":      "Updated",
		"address":   "New Address",
		"latitude":  40.7128,
		"longitude": -74.0060,
	}

	mockRepo.On("GetFavoriteByID", mock.Anything, favorite.ID).Return(favorite, nil)
	mockRepo.On("UpdateFavorite", mock.Anything, mock.AnythingOfType("*favorites.FavoriteLocation")).Return(errors.New("database error"))

	c, w := setupTestContext("PUT", "/api/v1/favorites/"+favorite.ID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: favorite.ID.String()}}
	setUserContext(c, userID)

	handler.UpdateFavorite(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// DeleteFavorite Handler Tests
// ============================================================================

func TestHandler_DeleteFavorite_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	favoriteID := uuid.New()

	mockRepo.On("DeleteFavorite", mock.Anything, favoriteID, userID).Return(nil)

	c, w := setupTestContext("DELETE", "/api/v1/favorites/"+favoriteID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: favoriteID.String()}}
	setUserContext(c, userID)

	handler.DeleteFavorite(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_DeleteFavorite_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	favoriteID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/favorites/"+favoriteID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: favoriteID.String()}}
	// Don't set user context

	handler.DeleteFavorite(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_DeleteFavorite_InvalidUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	favoriteID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/favorites/"+favoriteID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: favoriteID.String()}}
	c.Set("user_id", "invalid-uuid-type") // Wrong type

	handler.DeleteFavorite(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_DeleteFavorite_InvalidFavoriteID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/favorites/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID)

	handler.DeleteFavorite(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DeleteFavorite_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	favoriteID := uuid.New()

	mockRepo.On("DeleteFavorite", mock.Anything, favoriteID, userID).Return(errors.New("favorite location not found"))

	c, w := setupTestContext("DELETE", "/api/v1/favorites/"+favoriteID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: favoriteID.String()}}
	setUserContext(c, userID)

	handler.DeleteFavorite(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_DeleteFavorite_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	favoriteID := uuid.New()

	mockRepo.On("DeleteFavorite", mock.Anything, favoriteID, userID).Return(errors.New("database error"))

	c, w := setupTestContext("DELETE", "/api/v1/favorites/"+favoriteID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: favoriteID.String()}}
	setUserContext(c, userID)

	handler.DeleteFavorite(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_CreateFavorite_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		setupMock      func(*MockRepository)
		setUserID      bool
		expectedStatus int
	}{
		{
			name: "success",
			requestBody: map[string]interface{}{
				"name":      "Home",
				"address":   "123 Main St",
				"latitude":  40.7128,
				"longitude": -74.0060,
			},
			setupMock: func(m *MockRepository) {
				m.On("CreateFavorite", mock.Anything, mock.AnythingOfType("*favorites.FavoriteLocation")).Return(nil)
			},
			setUserID:      true,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "unauthorized",
			requestBody: map[string]interface{}{
				"name":      "Home",
				"address":   "123 Main St",
				"latitude":  40.7128,
				"longitude": -74.0060,
			},
			setupMock:      func(m *MockRepository) {},
			setUserID:      false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing name",
			requestBody: map[string]interface{}{
				"address":   "123 Main St",
				"latitude":  40.7128,
				"longitude": -74.0060,
			},
			setupMock:      func(m *MockRepository) {},
			setUserID:      true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service error",
			requestBody: map[string]interface{}{
				"name":      "Home",
				"address":   "123 Main St",
				"latitude":  40.7128,
				"longitude": -74.0060,
			},
			setupMock: func(m *MockRepository) {
				m.On("CreateFavorite", mock.Anything, mock.AnythingOfType("*favorites.FavoriteLocation")).Return(errors.New("database error"))
			},
			setUserID:      true,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			tt.setupMock(mockRepo)

			c, w := setupTestContext("POST", "/api/v1/favorites", tt.requestBody)
			if tt.setUserID {
				setUserContext(c, uuid.New())
			}

			handler.CreateFavorite(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_GetFavorite_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		favoriteID     string
		setupMock      func(*MockRepository, uuid.UUID, uuid.UUID)
		setUserID      bool
		isOwner        bool
		expectedStatus int
	}{
		{
			name:       "success - owner",
			favoriteID: uuid.New().String(),
			setupMock: func(m *MockRepository, userID uuid.UUID, favoriteID uuid.UUID) {
				favorite := &FavoriteLocation{
					ID:        favoriteID,
					UserID:    userID,
					Name:      "Home",
					Address:   "123 Main St",
					Latitude:  40.7128,
					Longitude: -74.0060,
				}
				m.On("GetFavoriteByID", mock.Anything, favoriteID).Return(favorite, nil)
			},
			setUserID:      true,
			isOwner:        true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid favorite ID",
			favoriteID:     "invalid-uuid",
			setupMock:      func(m *MockRepository, userID uuid.UUID, favoriteID uuid.UUID) {},
			setUserID:      true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unauthorized - no user ID",
			favoriteID:     uuid.New().String(),
			setupMock:      func(m *MockRepository, userID uuid.UUID, favoriteID uuid.UUID) {},
			setUserID:      false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:       "not found",
			favoriteID: uuid.New().String(),
			setupMock: func(m *MockRepository, userID uuid.UUID, favoriteID uuid.UUID) {
				m.On("GetFavoriteByID", mock.Anything, favoriteID).Return(nil, errors.New("not found"))
			},
			setUserID:      true,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:       "unauthorized access - not owner",
			favoriteID: uuid.New().String(),
			setupMock: func(m *MockRepository, userID uuid.UUID, favoriteID uuid.UUID) {
				favorite := &FavoriteLocation{
					ID:        favoriteID,
					UserID:    uuid.New(), // Different user
					Name:      "Home",
					Address:   "123 Main St",
					Latitude:  40.7128,
					Longitude: -74.0060,
				}
				m.On("GetFavoriteByID", mock.Anything, favoriteID).Return(favorite, nil)
			},
			setUserID:      true,
			isOwner:        false,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			userID := uuid.New()
			favoriteID, _ := uuid.Parse(tt.favoriteID)

			tt.setupMock(mockRepo, userID, favoriteID)

			c, w := setupTestContext("GET", "/api/v1/favorites/"+tt.favoriteID, nil)
			c.Params = gin.Params{{Key: "id", Value: tt.favoriteID}}
			if tt.setUserID {
				setUserContext(c, userID)
			}

			handler.GetFavorite(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_GetFavorites_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	favorites := []*FavoriteLocation{createTestFavoriteLocation(userID)}

	mockRepo.On("GetFavoritesByUser", mock.Anything, userID).Return(favorites, nil)

	c, w := setupTestContext("GET", "/api/v1/favorites", nil)
	setUserContext(c, userID)

	handler.GetFavorites(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	// Verify response structure
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["favorites"])
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetFavoritesByUser", mock.Anything, userID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/favorites", nil)
	setUserContext(c, userID)

	handler.GetFavorites(c)

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

func TestHandler_GetFavorite_EmptyFavoriteID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/favorites/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	setUserContext(c, userID)

	handler.GetFavorite(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateFavorite_EmptyFavoriteID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	reqBody := map[string]interface{}{
		"name":      "Updated",
		"address":   "New Address",
		"latitude":  40.7128,
		"longitude": -74.0060,
	}

	c, w := setupTestContext("PUT", "/api/v1/favorites/", reqBody)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	setUserContext(c, userID)

	handler.UpdateFavorite(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DeleteFavorite_EmptyFavoriteID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/favorites/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	setUserContext(c, userID)

	handler.DeleteFavorite(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateFavorite_BoundaryCoordinates(t *testing.T) {
	tests := []struct {
		name      string
		latitude  float64
		longitude float64
		shouldErr bool
	}{
		{
			name:      "valid max latitude",
			latitude:  90.0,
			longitude: 45.0, // Use non-zero to pass required validation
			shouldErr: false,
		},
		{
			name:      "valid min latitude",
			latitude:  -90.0,
			longitude: 45.0, // Use non-zero to pass required validation
			shouldErr: false,
		},
		{
			name:      "valid max longitude",
			latitude:  45.0, // Use non-zero to pass required validation
			longitude: 180.0,
			shouldErr: false,
		},
		{
			name:      "valid min longitude",
			latitude:  45.0, // Use non-zero to pass required validation
			longitude: -180.0,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			userID := uuid.New()
			reqBody := map[string]interface{}{
				"name":      "Test Location",
				"address":   "Test Address",
				"latitude":  tt.latitude,
				"longitude": tt.longitude,
			}

			mockRepo.On("CreateFavorite", mock.Anything, mock.AnythingOfType("*favorites.FavoriteLocation")).Return(nil)

			c, w := setupTestContext("POST", "/api/v1/favorites", reqBody)
			setUserContext(c, userID)

			handler.CreateFavorite(c)

			if tt.shouldErr {
				assert.NotEqual(t, http.StatusCreated, w.Code)
			} else {
				assert.Equal(t, http.StatusCreated, w.Code)
			}
		})
	}
}
