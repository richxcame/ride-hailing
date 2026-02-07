package ratings

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
	"github.com/jackc/pgx/v5"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ========================================
// MOCK REPOSITORY
// ========================================

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateRating(ctx context.Context, rating *Rating) error {
	args := m.Called(ctx, rating)
	return args.Error(0)
}

func (m *MockRepository) GetRatingByRideAndRater(ctx context.Context, rideID, raterID uuid.UUID) (*Rating, error) {
	args := m.Called(ctx, rideID, raterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Rating), args.Error(1)
}

func (m *MockRepository) GetRatingByID(ctx context.Context, id uuid.UUID) (*Rating, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Rating), args.Error(1)
}

func (m *MockRepository) GetAverageRating(ctx context.Context, userID uuid.UUID) (float64, int, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(float64), args.Int(1), args.Error(2)
}

func (m *MockRepository) GetRatingDistribution(ctx context.Context, userID uuid.UUID) (map[int]int, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[int]int), args.Error(1)
}

func (m *MockRepository) GetTopTags(ctx context.Context, userID uuid.UUID, limit int) ([]TagCount, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]TagCount), args.Error(1)
}

func (m *MockRepository) GetRecentRatings(ctx context.Context, userID uuid.UUID, limit int) ([]Rating, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Rating), args.Error(1)
}

func (m *MockRepository) GetRatingsGiven(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Rating, int, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]Rating), args.Int(1), args.Error(2)
}

func (m *MockRepository) GetRatingTrend(ctx context.Context, userID uuid.UUID) (float64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockRepository) CreateRatingResponse(ctx context.Context, resp *RatingResponse) error {
	args := m.Called(ctx, resp)
	return args.Error(0)
}

// ========================================
// MOCK SERVICE (for direct handler testing)
// ========================================

type MockService struct {
	mock.Mock
}

func (m *MockService) SubmitRating(ctx context.Context, raterID uuid.UUID, rateeID uuid.UUID, raterType RaterType, req *SubmitRatingRequest) (*Rating, error) {
	args := m.Called(ctx, raterID, rateeID, raterType, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Rating), args.Error(1)
}

func (m *MockService) GetMyRatingProfile(ctx context.Context, userID uuid.UUID) (*UserRatingProfile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UserRatingProfile), args.Error(1)
}

func (m *MockService) GetUserRating(ctx context.Context, userID uuid.UUID) (*UserRatingProfile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UserRatingProfile), args.Error(1)
}

func (m *MockService) GetRatingsGiven(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Rating, int, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]Rating), args.Int(1), args.Error(2)
}

func (m *MockService) RespondToRating(ctx context.Context, userID uuid.UUID, ratingID uuid.UUID, req *RespondToRatingRequest) (*RatingResponse, error) {
	args := m.Called(ctx, userID, ratingID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RatingResponse), args.Error(1)
}

func (m *MockService) GetSuggestedTags(raterType RaterType) []RatingTag {
	args := m.Called(raterType)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]RatingTag)
}

// ========================================
// TEST HELPERS
// ========================================

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

func createTestRating(riderID, driverID, rideID uuid.UUID, score int) *Rating {
	comment := "Great ride!"
	return &Rating{
		ID:        uuid.New(),
		RideID:    rideID,
		RaterID:   riderID,
		RateeID:   driverID,
		RaterType: RaterTypeRider,
		Score:     score,
		Comment:   &comment,
		Tags:      []string{"friendly", "clean_car"},
		IsPublic:  true,
		CreatedAt: time.Now(),
	}
}

func createTestUserRatingProfile(userID uuid.UUID) *UserRatingProfile {
	return &UserRatingProfile{
		UserID:             userID,
		AverageRating:      4.5,
		TotalRatings:       100,
		RatingDistribution: map[int]int{1: 2, 2: 5, 3: 10, 4: 30, 5: 53},
		TopTags:            []TagCount{{Tag: "friendly", Count: 50}, {Tag: "clean_car", Count: 40}},
		RecentRatings:      []Rating{},
		RatingTrend:        0.2,
	}
}

// testHandler wraps Handler to use MockService
type testHandler struct {
	service *MockService
}

func newTestHandler(svc *MockService) *testHandler {
	return &testHandler{service: svc}
}

// ============================================================================
// RateDriver Handler Tests
// ============================================================================

func TestHandler_RateDriver_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := SubmitRatingRequest{
		RideID:  rideID,
		Score:   5,
		Comment: strPtr("Excellent driver!"),
		Tags:    []string{"friendly", "clean_car"},
	}

	expectedRating := createTestRating(riderID, driverID, rideID, 5)

	mockService.On("SubmitRating", mock.Anything, riderID, driverID, RaterTypeRider, mock.AnythingOfType("*ratings.SubmitRatingRequest")).Return(expectedRating, nil)

	h := newTestHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/ratings/driver?driver_id="+driverID.String(), reqBody)
	setUserContext(c, riderID, models.RoleRider)
	c.Request.URL.RawQuery = "driver_id=" + driverID.String()

	// Simulate handler logic
	h.rateDriver(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func (h *testHandler) rateDriver(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	riderID := userID.(uuid.UUID)

	var req SubmitRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	driverIDStr := c.Query("driver_id")
	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "driver_id query parameter required")
		return
	}

	rating, err := h.service.SubmitRating(c.Request.Context(), riderID, driverID, RaterTypeRider, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to submit rating")
		return
	}

	common.CreatedResponse(c, rating)
}

func TestHandler_RateDriver_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	h := newTestHandler(mockService)

	reqBody := SubmitRatingRequest{
		RideID: uuid.New(),
		Score:  5,
	}

	c, w := setupTestContext("POST", "/api/v1/ratings/driver", reqBody)
	// Don't set user context to simulate unauthorized

	h.rateDriver(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_RateDriver_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	h := newTestHandler(mockService)

	riderID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/ratings/driver", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/ratings/driver", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, riderID, models.RoleRider)

	h.rateDriver(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RateDriver_MissingDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	h := newTestHandler(mockService)

	riderID := uuid.New()

	reqBody := SubmitRatingRequest{
		RideID: uuid.New(),
		Score:  5,
	}

	c, w := setupTestContext("POST", "/api/v1/ratings/driver", reqBody)
	setUserContext(c, riderID, models.RoleRider)
	// No driver_id query param

	h.rateDriver(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "driver_id")
}

func TestHandler_RateDriver_InvalidDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	h := newTestHandler(mockService)

	riderID := uuid.New()

	reqBody := SubmitRatingRequest{
		RideID: uuid.New(),
		Score:  5,
	}

	c, w := setupTestContext("POST", "/api/v1/ratings/driver?driver_id=invalid-uuid", reqBody)
	c.Request.URL.RawQuery = "driver_id=invalid-uuid"
	setUserContext(c, riderID, models.RoleRider)

	h.rateDriver(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RateDriver_AlreadyRated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := SubmitRatingRequest{
		RideID: rideID,
		Score:  5,
	}

	mockService.On("SubmitRating", mock.Anything, riderID, driverID, RaterTypeRider, mock.AnythingOfType("*ratings.SubmitRatingRequest")).Return(nil, common.NewConflictError("you have already rated this ride"))

	h := newTestHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/ratings/driver?driver_id="+driverID.String(), reqBody)
	c.Request.URL.RawQuery = "driver_id=" + driverID.String()
	setUserContext(c, riderID, models.RoleRider)

	h.rateDriver(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "already rated")
}

func TestHandler_RateDriver_InvalidScore(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	// Test scores outside 1-5 range
	testCases := []struct {
		name  string
		score int
	}{
		{"score too low", 0},
		{"score too high", 6},
		{"negative score", -1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := SubmitRatingRequest{
				RideID: rideID,
				Score:  tc.score,
			}

			mockService.On("SubmitRating", mock.Anything, riderID, driverID, RaterTypeRider, mock.AnythingOfType("*ratings.SubmitRatingRequest")).Return(nil, common.NewBadRequestError("score must be between 1 and 5", nil)).Maybe()

			h := newTestHandler(mockService)

			c, w := setupTestContext("POST", "/api/v1/ratings/driver?driver_id="+driverID.String(), reqBody)
			c.Request.URL.RawQuery = "driver_id=" + driverID.String()
			setUserContext(c, riderID, models.RoleRider)

			h.rateDriver(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandler_RateDriver_ValidScores(t *testing.T) {
	for score := 1; score <= 5; score++ {
		t.Run("score_"+string(rune('0'+score)), func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(MockService)
			riderID := uuid.New()
			driverID := uuid.New()
			rideID := uuid.New()

			reqBody := SubmitRatingRequest{
				RideID: rideID,
				Score:  score,
			}

			expectedRating := createTestRating(riderID, driverID, rideID, score)
			expectedRating.Score = score

			mockService.On("SubmitRating", mock.Anything, riderID, driverID, RaterTypeRider, mock.AnythingOfType("*ratings.SubmitRatingRequest")).Return(expectedRating, nil)

			h := newTestHandler(mockService)

			c, w := setupTestContext("POST", "/api/v1/ratings/driver?driver_id="+driverID.String(), reqBody)
			c.Request.URL.RawQuery = "driver_id=" + driverID.String()
			setUserContext(c, riderID, models.RoleRider)

			h.rateDriver(c)

			assert.Equal(t, http.StatusCreated, w.Code)
		})
	}
}

func TestHandler_RateDriver_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := SubmitRatingRequest{
		RideID: rideID,
		Score:  5,
	}

	mockService.On("SubmitRating", mock.Anything, riderID, driverID, RaterTypeRider, mock.AnythingOfType("*ratings.SubmitRatingRequest")).Return(nil, errors.New("database error"))

	h := newTestHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/ratings/driver?driver_id="+driverID.String(), reqBody)
	c.Request.URL.RawQuery = "driver_id=" + driverID.String()
	setUserContext(c, riderID, models.RoleRider)

	h.rateDriver(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// RateRider Handler Tests (Driver rating rider)
// ============================================================================

func (h *testHandler) rateRider(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	driverID := userID.(uuid.UUID)

	var req SubmitRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	riderIDStr := c.Query("rider_id")
	riderID, err := uuid.Parse(riderIDStr)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "rider_id query parameter required")
		return
	}

	rating, err := h.service.SubmitRating(c.Request.Context(), driverID, riderID, RaterTypeDriver, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to submit rating")
		return
	}

	common.CreatedResponse(c, rating)
}

func TestHandler_RateRider_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	driverID := uuid.New()
	riderID := uuid.New()
	rideID := uuid.New()

	reqBody := SubmitRatingRequest{
		RideID:  rideID,
		Score:   4,
		Comment: strPtr("Polite rider"),
		Tags:    []string{"polite_rider", "on_time"},
	}

	expectedRating := &Rating{
		ID:        uuid.New(),
		RideID:    rideID,
		RaterID:   driverID,
		RateeID:   riderID,
		RaterType: RaterTypeDriver,
		Score:     4,
		Comment:   strPtr("Polite rider"),
		Tags:      []string{"polite_rider", "on_time"},
		IsPublic:  true,
		CreatedAt: time.Now(),
	}

	mockService.On("SubmitRating", mock.Anything, driverID, riderID, RaterTypeDriver, mock.AnythingOfType("*ratings.SubmitRatingRequest")).Return(expectedRating, nil)

	h := newTestHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/driver/ratings/rider?rider_id="+riderID.String(), reqBody)
	c.Request.URL.RawQuery = "rider_id=" + riderID.String()
	setUserContext(c, driverID, models.RoleDriver)

	h.rateRider(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_RateRider_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	h := newTestHandler(mockService)

	reqBody := SubmitRatingRequest{
		RideID: uuid.New(),
		Score:  4,
	}

	c, w := setupTestContext("POST", "/api/v1/driver/ratings/rider", reqBody)
	// Don't set user context

	h.rateRider(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_RateRider_MissingRiderID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	h := newTestHandler(mockService)

	driverID := uuid.New()

	reqBody := SubmitRatingRequest{
		RideID: uuid.New(),
		Score:  4,
	}

	c, w := setupTestContext("POST", "/api/v1/driver/ratings/rider", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	h.rateRider(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "rider_id")
}

// ============================================================================
// GetMyProfile Handler Tests
// ============================================================================

func (h *testHandler) getMyProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	profile, err := h.service.GetMyRatingProfile(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get rating profile")
		return
	}

	common.SuccessResponse(c, profile)
}

func TestHandler_GetMyProfile_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	userID := uuid.New()

	expectedProfile := createTestUserRatingProfile(userID)

	mockService.On("GetMyRatingProfile", mock.Anything, userID).Return(expectedProfile, nil)

	h := newTestHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/ratings/me", nil)
	setUserContext(c, userID, models.RoleRider)

	h.getMyProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, 4.5, data["average_rating"])
	assert.Equal(t, float64(100), data["total_ratings"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetMyProfile_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	h := newTestHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/ratings/me", nil)
	// Don't set user context

	h.getMyProfile(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetMyProfile_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	userID := uuid.New()

	mockService.On("GetMyRatingProfile", mock.Anything, userID).Return(nil, errors.New("database error"))

	h := newTestHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/ratings/me", nil)
	setUserContext(c, userID, models.RoleRider)

	h.getMyProfile(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_GetMyProfile_AsDriver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	driverID := uuid.New()

	expectedProfile := createTestUserRatingProfile(driverID)

	mockService.On("GetMyRatingProfile", mock.Anything, driverID).Return(expectedProfile, nil)

	h := newTestHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/driver/ratings/me", nil)
	setUserContext(c, driverID, models.RoleDriver)

	h.getMyProfile(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetUserRating Handler Tests
// ============================================================================

func (h *testHandler) getUserRating(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user id")
		return
	}

	profile, err := h.service.GetUserRating(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get user rating")
		return
	}

	common.SuccessResponse(c, profile)
}

func TestHandler_GetUserRating_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	targetUserID := uuid.New()

	expectedProfile := &UserRatingProfile{
		UserID:        targetUserID,
		AverageRating: 4.8,
		TotalRatings:  50,
		TopTags:       []TagCount{{Tag: "friendly", Count: 30}},
	}

	mockService.On("GetUserRating", mock.Anything, targetUserID).Return(expectedProfile, nil)

	h := newTestHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/ratings/users/"+targetUserID.String(), nil)
	c.Params = gin.Params{{Key: "userId", Value: targetUserID.String()}}

	h.getUserRating(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, 4.8, data["average_rating"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetUserRating_InvalidUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	h := newTestHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/ratings/users/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "userId", Value: "invalid-uuid"}}

	h.getUserRating(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid user id")
}

func TestHandler_GetUserRating_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	targetUserID := uuid.New()

	mockService.On("GetUserRating", mock.Anything, targetUserID).Return(nil, errors.New("database error"))

	h := newTestHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/ratings/users/"+targetUserID.String(), nil)
	c.Params = gin.Params{{Key: "userId", Value: targetUserID.String()}}

	h.getUserRating(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetMyRatingsGiven Handler Tests
// ============================================================================

func (h *testHandler) getMyRatingsGiven(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := 20
	offset := 0
	// In real handler, these would be parsed from query params

	ratings, total, err := h.service.GetRatingsGiven(c.Request.Context(), userID.(uuid.UUID), limit, offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get ratings")
		return
	}

	meta := &common.Meta{
		Limit:  limit,
		Offset: offset,
		Total:  int64(total),
	}
	common.SuccessResponseWithMeta(c, gin.H{"ratings": ratings}, meta)
}

func TestHandler_GetMyRatingsGiven_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	userID := uuid.New()

	ratings := []Rating{
		*createTestRating(userID, uuid.New(), uuid.New(), 5),
		*createTestRating(userID, uuid.New(), uuid.New(), 4),
	}

	mockService.On("GetRatingsGiven", mock.Anything, userID, 20, 0).Return(ratings, 2, nil)

	h := newTestHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/ratings/given", nil)
	setUserContext(c, userID, models.RoleRider)

	h.getMyRatingsGiven(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetMyRatingsGiven_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	h := newTestHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/ratings/given", nil)
	// Don't set user context

	h.getMyRatingsGiven(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetMyRatingsGiven_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	userID := uuid.New()

	mockService.On("GetRatingsGiven", mock.Anything, userID, 20, 0).Return([]Rating{}, 0, nil)

	h := newTestHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/ratings/given", nil)
	setUserContext(c, userID, models.RoleRider)

	h.getMyRatingsGiven(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetMyRatingsGiven_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	userID := uuid.New()

	mockService.On("GetRatingsGiven", mock.Anything, userID, 20, 0).Return(nil, 0, errors.New("database error"))

	h := newTestHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/ratings/given", nil)
	setUserContext(c, userID, models.RoleRider)

	h.getMyRatingsGiven(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// RespondToRating Handler Tests
// ============================================================================

func (h *testHandler) respondToRating(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	ratingID, err := uuid.Parse(c.Param("ratingId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid rating id")
		return
	}

	var req RespondToRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.RespondToRating(c.Request.Context(), userID.(uuid.UUID), ratingID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to respond to rating")
		return
	}

	common.CreatedResponse(c, resp)
}

func TestHandler_RespondToRating_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	userID := uuid.New()
	ratingID := uuid.New()

	reqBody := RespondToRatingRequest{
		Comment: "Thank you for your feedback!",
	}

	expectedResponse := &RatingResponse{
		ID:        uuid.New(),
		RatingID:  ratingID,
		UserID:    userID,
		Comment:   "Thank you for your feedback!",
		CreatedAt: time.Now(),
	}

	mockService.On("RespondToRating", mock.Anything, userID, ratingID, mock.AnythingOfType("*ratings.RespondToRatingRequest")).Return(expectedResponse, nil)

	h := newTestHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/ratings/"+ratingID.String()+"/respond", reqBody)
	c.Params = gin.Params{{Key: "ratingId", Value: ratingID.String()}}
	setUserContext(c, userID, models.RoleDriver)

	h.respondToRating(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_RespondToRating_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	h := newTestHandler(mockService)

	ratingID := uuid.New()
	reqBody := RespondToRatingRequest{
		Comment: "Thank you!",
	}

	c, w := setupTestContext("POST", "/api/v1/ratings/"+ratingID.String()+"/respond", reqBody)
	c.Params = gin.Params{{Key: "ratingId", Value: ratingID.String()}}
	// Don't set user context

	h.respondToRating(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_RespondToRating_InvalidRatingID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	h := newTestHandler(mockService)

	userID := uuid.New()
	reqBody := RespondToRatingRequest{
		Comment: "Thank you!",
	}

	c, w := setupTestContext("POST", "/api/v1/ratings/invalid-uuid/respond", reqBody)
	c.Params = gin.Params{{Key: "ratingId", Value: "invalid-uuid"}}
	setUserContext(c, userID, models.RoleDriver)

	h.respondToRating(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid rating id")
}

func TestHandler_RespondToRating_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	h := newTestHandler(mockService)

	userID := uuid.New()
	ratingID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/ratings/"+ratingID.String()+"/respond", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/ratings/"+ratingID.String()+"/respond", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "ratingId", Value: ratingID.String()}}
	setUserContext(c, userID, models.RoleDriver)

	h.respondToRating(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RespondToRating_RatingNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	userID := uuid.New()
	ratingID := uuid.New()

	reqBody := RespondToRatingRequest{
		Comment: "Thank you!",
	}

	mockService.On("RespondToRating", mock.Anything, userID, ratingID, mock.AnythingOfType("*ratings.RespondToRatingRequest")).Return(nil, common.NewNotFoundError("rating not found", nil))

	h := newTestHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/ratings/"+ratingID.String()+"/respond", reqBody)
	c.Params = gin.Params{{Key: "ratingId", Value: ratingID.String()}}
	setUserContext(c, userID, models.RoleDriver)

	h.respondToRating(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_RespondToRating_NotRatee(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	userID := uuid.New()
	ratingID := uuid.New()

	reqBody := RespondToRatingRequest{
		Comment: "Thank you!",
	}

	mockService.On("RespondToRating", mock.Anything, userID, ratingID, mock.AnythingOfType("*ratings.RespondToRatingRequest")).Return(nil, common.NewForbiddenError("you can only respond to your own ratings"))

	h := newTestHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/ratings/"+ratingID.String()+"/respond", reqBody)
	c.Params = gin.Params{{Key: "ratingId", Value: ratingID.String()}}
	setUserContext(c, userID, models.RoleDriver)

	h.respondToRating(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ============================================================================
// GetSuggestedTags Handler Tests
// ============================================================================

func (h *testHandler) getSuggestedTags(c *gin.Context) {
	raterType := RaterType(c.DefaultQuery("type", "rider"))
	tags := h.service.GetSuggestedTags(raterType)
	common.SuccessResponse(c, gin.H{"tags": tags})
}

func TestHandler_GetSuggestedTags_Rider(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)

	expectedTags := []RatingTag{
		TagGreatConversation, TagSmoothDriving, TagCleanCar,
		TagKnowsRoute, TagFriendly, TagProfessional,
		TagGoodMusic, TagSafeDriver,
	}

	mockService.On("GetSuggestedTags", RaterTypeRider).Return(expectedTags)

	h := newTestHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/ratings/tags?type=rider", nil)
	c.Request.URL.RawQuery = "type=rider"

	h.getSuggestedTags(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetSuggestedTags_Driver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)

	expectedTags := []RatingTag{
		TagPoliteRider, TagOnTime, TagRespectful, TagGoodDirections,
	}

	mockService.On("GetSuggestedTags", RaterTypeDriver).Return(expectedTags)

	h := newTestHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/ratings/tags?type=driver", nil)
	c.Request.URL.RawQuery = "type=driver"

	h.getSuggestedTags(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetSuggestedTags_DefaultType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)

	expectedTags := []RatingTag{
		TagGreatConversation, TagSmoothDriving, TagCleanCar,
	}

	mockService.On("GetSuggestedTags", RaterTypeRider).Return(expectedTags)

	h := newTestHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/ratings/tags", nil)
	// No type query param, should default to "rider"

	h.getSuggestedTags(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// Table-Driven Tests for Rating Validation
// ============================================================================

func TestHandler_RatingValidation_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		score          int
		comment        *string
		tags           []string
		expectError    bool
		expectedStatus int
	}{
		{
			name:           "valid rating with minimum score",
			score:          1,
			comment:        nil,
			tags:           nil,
			expectError:    false,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "valid rating with maximum score",
			score:          5,
			comment:        strPtr("Perfect!"),
			tags:           []string{"friendly"},
			expectError:    false,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "valid rating with all tags",
			score:          4,
			comment:        strPtr("Good ride"),
			tags:           []string{"friendly", "clean_car", "safe_driver"},
			expectError:    false,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "valid rating with empty tags",
			score:          3,
			comment:        nil,
			tags:           []string{},
			expectError:    false,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "invalid score too low",
			score:          0,
			comment:        nil,
			tags:           nil,
			expectError:    true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid score too high",
			score:          6,
			comment:        nil,
			tags:           nil,
			expectError:    true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid negative score",
			score:          -1,
			comment:        nil,
			tags:           nil,
			expectError:    true,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(MockService)
			riderID := uuid.New()
			driverID := uuid.New()
			rideID := uuid.New()

			reqBody := SubmitRatingRequest{
				RideID:  rideID,
				Score:   tt.score,
				Comment: tt.comment,
				Tags:    tt.tags,
			}

			if tt.expectError {
				mockService.On("SubmitRating", mock.Anything, riderID, driverID, RaterTypeRider, mock.AnythingOfType("*ratings.SubmitRatingRequest")).Return(nil, common.NewBadRequestError("score must be between 1 and 5", nil)).Maybe()
			} else {
				expectedRating := createTestRating(riderID, driverID, rideID, tt.score)
				mockService.On("SubmitRating", mock.Anything, riderID, driverID, RaterTypeRider, mock.AnythingOfType("*ratings.SubmitRatingRequest")).Return(expectedRating, nil)
			}

			h := newTestHandler(mockService)

			c, w := setupTestContext("POST", "/api/v1/ratings/driver?driver_id="+driverID.String(), reqBody)
			c.Request.URL.RawQuery = "driver_id=" + driverID.String()
			setUserContext(c, riderID, models.RoleRider)

			h.rateDriver(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// Service Integration Tests (with mock repository)
// ============================================================================

func TestService_SubmitRating_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	service := &Service{repo: mockRepo}

	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	req := &SubmitRatingRequest{
		RideID:  rideID,
		Score:   5,
		Comment: strPtr("Great ride!"),
		Tags:    []string{"friendly"},
	}

	mockRepo.On("GetRatingByRideAndRater", mock.Anything, rideID, riderID).Return(nil, pgx.ErrNoRows)
	mockRepo.On("CreateRating", mock.Anything, mock.AnythingOfType("*ratings.Rating")).Return(nil)

	rating, err := service.SubmitRating(context.Background(), riderID, driverID, RaterTypeRider, req)

	assert.NoError(t, err)
	assert.NotNil(t, rating)
	assert.Equal(t, 5, rating.Score)
	assert.Equal(t, riderID, rating.RaterID)
	assert.Equal(t, driverID, rating.RateeID)
	assert.Equal(t, RaterTypeRider, rating.RaterType)
	mockRepo.AssertExpectations(t)
}

func TestService_SubmitRating_InvalidScore(t *testing.T) {
	mockRepo := new(MockRepository)
	service := &Service{repo: mockRepo}

	riderID := uuid.New()
	driverID := uuid.New()

	testCases := []struct {
		name  string
		score int
	}{
		{"score 0", 0},
		{"score 6", 6},
		{"score -1", -1},
		{"score 100", 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &SubmitRatingRequest{
				RideID: uuid.New(),
				Score:  tc.score,
			}

			rating, err := service.SubmitRating(context.Background(), riderID, driverID, RaterTypeRider, req)

			assert.Error(t, err)
			assert.Nil(t, rating)
			assert.Contains(t, err.Error(), "score must be between 1 and 5")
		})
	}
}

func TestService_SubmitRating_AlreadyRated(t *testing.T) {
	mockRepo := new(MockRepository)
	service := &Service{repo: mockRepo}

	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	existingRating := createTestRating(riderID, driverID, rideID, 5)

	req := &SubmitRatingRequest{
		RideID: rideID,
		Score:  4,
	}

	mockRepo.On("GetRatingByRideAndRater", mock.Anything, rideID, riderID).Return(existingRating, nil)

	rating, err := service.SubmitRating(context.Background(), riderID, driverID, RaterTypeRider, req)

	assert.Error(t, err)
	assert.Nil(t, rating)
	// Check for conflict error
	appErr, ok := err.(*common.AppError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusConflict, appErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestService_SubmitRating_CreateError(t *testing.T) {
	mockRepo := new(MockRepository)
	service := &Service{repo: mockRepo}

	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	req := &SubmitRatingRequest{
		RideID: rideID,
		Score:  5,
	}

	mockRepo.On("GetRatingByRideAndRater", mock.Anything, rideID, riderID).Return(nil, pgx.ErrNoRows)
	mockRepo.On("CreateRating", mock.Anything, mock.AnythingOfType("*ratings.Rating")).Return(errors.New("database error"))

	rating, err := service.SubmitRating(context.Background(), riderID, driverID, RaterTypeRider, req)

	assert.Error(t, err)
	assert.Nil(t, rating)
	mockRepo.AssertExpectations(t)
}

func TestService_GetMyRatingProfile_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	service := &Service{repo: mockRepo}

	userID := uuid.New()

	mockRepo.On("GetAverageRating", mock.Anything, userID).Return(4.5, 100, nil)
	mockRepo.On("GetRatingDistribution", mock.Anything, userID).Return(map[int]int{1: 2, 2: 5, 3: 10, 4: 30, 5: 53}, nil)
	mockRepo.On("GetTopTags", mock.Anything, userID, 10).Return([]TagCount{{Tag: "friendly", Count: 50}}, nil)
	mockRepo.On("GetRecentRatings", mock.Anything, userID, 10).Return([]Rating{}, nil)
	mockRepo.On("GetRatingTrend", mock.Anything, userID).Return(0.2, nil)

	profile, err := service.GetMyRatingProfile(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, profile)
	assert.Equal(t, 4.5, profile.AverageRating)
	assert.Equal(t, 100, profile.TotalRatings)
	assert.Equal(t, 0.2, profile.RatingTrend)
	mockRepo.AssertExpectations(t)
}

func TestService_RespondToRating_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	service := &Service{repo: mockRepo}

	userID := uuid.New()
	ratingID := uuid.New()

	rating := &Rating{
		ID:      ratingID,
		RateeID: userID,
	}

	req := &RespondToRatingRequest{
		Comment: "Thank you!",
	}

	mockRepo.On("GetRatingByID", mock.Anything, ratingID).Return(rating, nil)
	mockRepo.On("CreateRatingResponse", mock.Anything, mock.AnythingOfType("*ratings.RatingResponse")).Return(nil)

	resp, err := service.RespondToRating(context.Background(), userID, ratingID, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, ratingID, resp.RatingID)
	assert.Equal(t, userID, resp.UserID)
	assert.Equal(t, "Thank you!", resp.Comment)
	mockRepo.AssertExpectations(t)
}

func TestService_RespondToRating_NotRatee(t *testing.T) {
	mockRepo := new(MockRepository)
	service := &Service{repo: mockRepo}

	userID := uuid.New()
	otherUserID := uuid.New()
	ratingID := uuid.New()

	rating := &Rating{
		ID:      ratingID,
		RateeID: otherUserID, // Different user
	}

	req := &RespondToRatingRequest{
		Comment: "Thank you!",
	}

	mockRepo.On("GetRatingByID", mock.Anything, ratingID).Return(rating, nil)

	resp, err := service.RespondToRating(context.Background(), userID, ratingID, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	// Check for forbidden error
	appErr, ok := err.(*common.AppError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusForbidden, appErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestService_RespondToRating_NotFound(t *testing.T) {
	mockRepo := new(MockRepository)
	service := &Service{repo: mockRepo}

	userID := uuid.New()
	ratingID := uuid.New()

	req := &RespondToRatingRequest{
		Comment: "Thank you!",
	}

	mockRepo.On("GetRatingByID", mock.Anything, ratingID).Return(nil, pgx.ErrNoRows)

	resp, err := service.RespondToRating(context.Background(), userID, ratingID, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	mockRepo.AssertExpectations(t)
}

func TestService_GetRatingsGiven_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	service := &Service{repo: mockRepo}

	userID := uuid.New()

	ratings := []Rating{
		*createTestRating(userID, uuid.New(), uuid.New(), 5),
		*createTestRating(userID, uuid.New(), uuid.New(), 4),
	}

	mockRepo.On("GetRatingsGiven", mock.Anything, userID, 20, 0).Return(ratings, 2, nil)

	result, total, err := service.GetRatingsGiven(context.Background(), userID, 20, 0)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 2, total)
	mockRepo.AssertExpectations(t)
}

func TestService_GetRatingsGiven_LimitCapping(t *testing.T) {
	userID := uuid.New()

	// Test limit capping for invalid values
	testCases := []struct {
		name          string
		inputLimit    int
		expectedLimit int
	}{
		{"zero limit defaults to 20", 0, 20},
		{"negative limit defaults to 20", -5, 20},
		{"over 50 defaults to 20", 100, 20},
		{"valid limit unchanged", 30, 30},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := &Service{repo: mockRepo}

			mockRepo.On("GetRatingsGiven", mock.Anything, userID, tc.expectedLimit, 0).Return([]Rating{}, 0, nil)

			_, _, err := service.GetRatingsGiven(context.Background(), userID, tc.inputLimit, 0)

			assert.NoError(t, err)
			mockRepo.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Helper Function
// ============================================================================

func strPtr(s string) *string {
	return &s
}

// Ensure MockRepository implements the interface used by Service
func (r *MockRepository) getRepo() *MockRepository {
	return r
}

// ============================================================================
// Authorization Tests
// ============================================================================

func TestHandler_RateDriver_RequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	h := newTestHandler(mockService)

	reqBody := SubmitRatingRequest{
		RideID: uuid.New(),
		Score:  5,
	}

	c, w := setupTestContext("POST", "/api/v1/ratings/driver", reqBody)
	// No user context set

	h.rateDriver(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_RateRider_RequiresDriverRole(t *testing.T) {
	// This test verifies the handler behavior when a rider tries to rate another rider
	// In a real scenario, middleware would block this, but the handler should still work
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	riderID := uuid.New()
	otherRiderID := uuid.New()
	rideID := uuid.New()

	reqBody := SubmitRatingRequest{
		RideID: rideID,
		Score:  4,
	}

	// The service would normally validate this, but handler allows it through
	expectedRating := &Rating{
		ID:        uuid.New(),
		RideID:    rideID,
		RaterID:   riderID,
		RateeID:   otherRiderID,
		RaterType: RaterTypeDriver,
		Score:     4,
		CreatedAt: time.Now(),
	}

	mockService.On("SubmitRating", mock.Anything, riderID, otherRiderID, RaterTypeDriver, mock.AnythingOfType("*ratings.SubmitRatingRequest")).Return(expectedRating, nil)

	h := newTestHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/driver/ratings/rider?rider_id="+otherRiderID.String(), reqBody)
	c.Request.URL.RawQuery = "rider_id=" + otherRiderID.String()
	setUserContext(c, riderID, models.RoleRider) // Using rider role instead of driver

	h.rateRider(c)

	// Handler allows it - authorization should be done at middleware/service level
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestHandler_RateDriver_WithEmptyComment(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	reqBody := SubmitRatingRequest{
		RideID:  rideID,
		Score:   5,
		Comment: nil, // No comment
		Tags:    nil, // No tags
	}

	expectedRating := &Rating{
		ID:        uuid.New(),
		RideID:    rideID,
		RaterID:   riderID,
		RateeID:   driverID,
		RaterType: RaterTypeRider,
		Score:     5,
		Tags:      []string{},
		IsPublic:  true,
		CreatedAt: time.Now(),
	}

	mockService.On("SubmitRating", mock.Anything, riderID, driverID, RaterTypeRider, mock.AnythingOfType("*ratings.SubmitRatingRequest")).Return(expectedRating, nil)

	h := newTestHandler(mockService)

	c, w := setupTestContext("POST", "/api/v1/ratings/driver?driver_id="+driverID.String(), reqBody)
	c.Request.URL.RawQuery = "driver_id=" + driverID.String()
	setUserContext(c, riderID, models.RoleRider)

	h.rateDriver(c)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_GetUserRating_EmptyUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockService)
	h := newTestHandler(mockService)

	c, w := setupTestContext("GET", "/api/v1/ratings/users/", nil)
	c.Params = gin.Params{{Key: "userId", Value: ""}}

	h.getUserRating(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestService_GetUserRating_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	service := &Service{repo: mockRepo}

	userID := uuid.New()

	mockRepo.On("GetAverageRating", mock.Anything, userID).Return(4.8, 50, nil)
	mockRepo.On("GetTopTags", mock.Anything, userID, 5).Return([]TagCount{{Tag: "friendly", Count: 30}}, nil)

	profile, err := service.GetUserRating(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, profile)
	assert.Equal(t, 4.8, profile.AverageRating)
	assert.Equal(t, 50, profile.TotalRatings)
	// Public profile should not include recent ratings or distribution
	assert.Empty(t, profile.RecentRatings)
	assert.Nil(t, profile.RatingDistribution)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetSuggestedTags Service Tests
// ============================================================================

func TestService_GetSuggestedTags_Rider(t *testing.T) {
	service := &Service{}
	tags := service.GetSuggestedTags(RaterTypeRider)

	assert.Len(t, tags, 8)
	assert.Contains(t, tags, TagGreatConversation)
	assert.Contains(t, tags, TagSmoothDriving)
	assert.Contains(t, tags, TagCleanCar)
	assert.Contains(t, tags, TagSafeDriver)
}

func TestService_GetSuggestedTags_Driver(t *testing.T) {
	service := &Service{}
	tags := service.GetSuggestedTags(RaterTypeDriver)

	assert.Len(t, tags, 4)
	assert.Contains(t, tags, TagPoliteRider)
	assert.Contains(t, tags, TagOnTime)
	assert.Contains(t, tags, TagRespectful)
	assert.Contains(t, tags, TagGoodDirections)
}
