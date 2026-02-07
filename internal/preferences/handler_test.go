package preferences

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

func (m *MockRepository) GetRiderPreferences(ctx context.Context, userID uuid.UUID) (*RiderPreferences, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RiderPreferences), args.Error(1)
}

func (m *MockRepository) UpsertRiderPreferences(ctx context.Context, p *RiderPreferences) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *MockRepository) SetRideOverride(ctx context.Context, o *RidePreferenceOverride) error {
	args := m.Called(ctx, o)
	return args.Error(0)
}

func (m *MockRepository) GetRideOverride(ctx context.Context, rideID uuid.UUID) (*RidePreferenceOverride, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RidePreferenceOverride), args.Error(1)
}

func (m *MockRepository) GetDriverCapabilities(ctx context.Context, driverID uuid.UUID) (*DriverCapabilities, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverCapabilities), args.Error(1)
}

func (m *MockRepository) UpsertDriverCapabilities(ctx context.Context, dc *DriverCapabilities) error {
	args := m.Called(ctx, dc)
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
	service := NewService(mockRepo)
	return NewHandler(service)
}

func createTestRiderPreferences(userID uuid.UUID) *RiderPreferences {
	return &RiderPreferences{
		ID:                 uuid.New(),
		UserID:             userID,
		Temperature:        TempNormal,
		Music:              MusicDriverChoice,
		Conversation:       ConversationFriendly,
		Route:              RouteFastest,
		ChildSeat:          false,
		PetFriendly:        false,
		WheelchairAccess:   false,
		PreferFemaleDriver: false,
		LuggageAssistance:  true,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
}

func createTestDriverCapabilities(driverID uuid.UUID) *DriverCapabilities {
	return &DriverCapabilities{
		ID:               uuid.New(),
		DriverID:         driverID,
		HasChildSeat:     true,
		PetFriendly:      true,
		WheelchairAccess: false,
		LuggageCapacity:  3,
		MaxPassengers:    4,
		Languages:        []string{"en", "es"},
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

func createTestRideOverride(rideID, userID uuid.UUID) *RidePreferenceOverride {
	temp := TempCool
	music := MusicQuiet
	return &RidePreferenceOverride{
		ID:          uuid.New(),
		RideID:      rideID,
		UserID:      userID,
		Temperature: &temp,
		Music:       &music,
		CreatedAt:   time.Now(),
	}
}

// ============================================================================
// GetPreferences Handler Tests
// ============================================================================

func TestHandler_GetPreferences_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	prefs := createTestRiderPreferences(userID)

	mockRepo.On("GetRiderPreferences", mock.Anything, userID).Return(prefs, nil)

	c, w := setupTestContext("GET", "/api/v1/preferences", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetPreferences(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetPreferences_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/preferences", nil)
	// Don't set user context

	handler.GetPreferences(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "unauthorized")
}

func TestHandler_GetPreferences_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetRiderPreferences", mock.Anything, userID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/preferences", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetPreferences(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// UpdatePreferences Handler Tests
// ============================================================================

func TestHandler_UpdatePreferences_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	existingPrefs := createTestRiderPreferences(userID)

	temp := TempCool
	req := UpdatePreferencesRequest{
		Temperature: &temp,
	}

	mockRepo.On("GetRiderPreferences", mock.Anything, userID).Return(existingPrefs, nil)
	mockRepo.On("UpsertRiderPreferences", mock.Anything, mock.AnythingOfType("*preferences.RiderPreferences")).Return(nil)

	c, w := setupTestContext("PUT", "/api/v1/preferences", req)
	setUserContext(c, userID, models.RoleRider)

	handler.UpdatePreferences(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_UpdatePreferences_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	temp := TempCool
	req := UpdatePreferencesRequest{
		Temperature: &temp,
	}

	c, w := setupTestContext("PUT", "/api/v1/preferences", req)
	// Don't set user context

	handler.UpdatePreferences(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_UpdatePreferences_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("PUT", "/api/v1/preferences", nil)
	c.Request = httptest.NewRequest("PUT", "/api/v1/preferences", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleRider)

	handler.UpdatePreferences(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid request body")
}

func TestHandler_UpdatePreferences_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	temp := TempCool
	req := UpdatePreferencesRequest{
		Temperature: &temp,
	}

	mockRepo.On("GetRiderPreferences", mock.Anything, userID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("PUT", "/api/v1/preferences", req)
	setUserContext(c, userID, models.RoleRider)

	handler.UpdatePreferences(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// SetRidePreferences Handler Tests
// ============================================================================

func TestHandler_SetRidePreferences_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	rideID := uuid.New()

	temp := TempCool
	req := SetRidePreferencesRequest{
		RideID:      rideID,
		Temperature: &temp,
	}

	mockRepo.On("SetRideOverride", mock.Anything, mock.AnythingOfType("*preferences.RidePreferenceOverride")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/preferences/rides", req)
	setUserContext(c, userID, models.RoleRider)

	handler.SetRidePreferences(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_SetRidePreferences_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	rideID := uuid.New()
	temp := TempCool
	req := SetRidePreferencesRequest{
		RideID:      rideID,
		Temperature: &temp,
	}

	c, w := setupTestContext("POST", "/api/v1/preferences/rides", req)
	// Don't set user context

	handler.SetRidePreferences(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_SetRidePreferences_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/preferences/rides", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/preferences/rides", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleRider)

	handler.SetRidePreferences(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SetRidePreferences_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	rideID := uuid.New()

	temp := TempCool
	req := SetRidePreferencesRequest{
		RideID:      rideID,
		Temperature: &temp,
	}

	mockRepo.On("SetRideOverride", mock.Anything, mock.AnythingOfType("*preferences.RidePreferenceOverride")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/preferences/rides", req)
	setUserContext(c, userID, models.RoleRider)

	handler.SetRidePreferences(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetRidePreferences Handler Tests
// ============================================================================

func TestHandler_GetRidePreferences_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	rideID := uuid.New()
	prefs := createTestRiderPreferences(userID)
	override := createTestRideOverride(rideID, userID)

	mockRepo.On("GetRiderPreferences", mock.Anything, userID).Return(prefs, nil)
	mockRepo.On("GetRideOverride", mock.Anything, rideID).Return(override, nil)

	c, w := setupTestContext("GET", "/api/v1/preferences/rides/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetRidePreferences(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetRidePreferences_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	rideID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/preferences/rides/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	// Don't set user context

	handler.GetRidePreferences(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetRidePreferences_InvalidRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/preferences/rides/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetRidePreferences(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid ride ID")
}

// ============================================================================
// GetDriverCapabilities Handler Tests
// ============================================================================

func TestHandler_GetDriverCapabilities_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	caps := createTestDriverCapabilities(driverID)

	mockRepo.On("GetDriverCapabilities", mock.Anything, driverID).Return(caps, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/capabilities", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetDriverCapabilities(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetDriverCapabilities_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/driver/capabilities", nil)
	// Don't set user context

	handler.GetDriverCapabilities(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetDriverCapabilities_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	mockRepo.On("GetDriverCapabilities", mock.Anything, driverID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/capabilities", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetDriverCapabilities(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// UpdateDriverCapabilities Handler Tests
// ============================================================================

func TestHandler_UpdateDriverCapabilities_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	existingCaps := createTestDriverCapabilities(driverID)

	hasChildSeat := true
	req := UpdateDriverCapabilitiesRequest{
		HasChildSeat: &hasChildSeat,
	}

	mockRepo.On("GetDriverCapabilities", mock.Anything, driverID).Return(existingCaps, nil)
	mockRepo.On("UpsertDriverCapabilities", mock.Anything, mock.AnythingOfType("*preferences.DriverCapabilities")).Return(nil)

	c, w := setupTestContext("PUT", "/api/v1/driver/capabilities", req)
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateDriverCapabilities(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_UpdateDriverCapabilities_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	hasChildSeat := true
	req := UpdateDriverCapabilitiesRequest{
		HasChildSeat: &hasChildSeat,
	}

	c, w := setupTestContext("PUT", "/api/v1/driver/capabilities", req)
	// Don't set user context

	handler.UpdateDriverCapabilities(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_UpdateDriverCapabilities_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	c, w := setupTestContext("PUT", "/api/v1/driver/capabilities", nil)
	c.Request = httptest.NewRequest("PUT", "/api/v1/driver/capabilities", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateDriverCapabilities(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateDriverCapabilities_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	hasChildSeat := true
	req := UpdateDriverCapabilitiesRequest{
		HasChildSeat: &hasChildSeat,
	}

	mockRepo.On("GetDriverCapabilities", mock.Anything, driverID).Return(nil, errors.New("database error"))
	mockRepo.On("UpsertDriverCapabilities", mock.Anything, mock.AnythingOfType("*preferences.DriverCapabilities")).Return(errors.New("database error"))

	c, w := setupTestContext("PUT", "/api/v1/driver/capabilities", req)
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateDriverCapabilities(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetRidePreferencesForDriver Handler Tests
// ============================================================================

func TestHandler_GetRidePreferencesForDriver_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	riderID := uuid.New()
	rideID := uuid.New()

	prefs := createTestRiderPreferences(riderID)
	override := createTestRideOverride(rideID, riderID)
	caps := createTestDriverCapabilities(driverID)

	mockRepo.On("GetRiderPreferences", mock.Anything, riderID).Return(prefs, nil)
	mockRepo.On("GetRideOverride", mock.Anything, rideID).Return(override, nil)
	mockRepo.On("GetDriverCapabilities", mock.Anything, driverID).Return(caps, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/rides/"+rideID.String()+"/preferences?rider_id="+riderID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	c.Request.URL.RawQuery = "rider_id=" + riderID.String()
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetRidePreferencesForDriver(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetRidePreferencesForDriver_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	rideID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/driver/rides/"+rideID.String()+"/preferences", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	// Don't set user context

	handler.GetRidePreferencesForDriver(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetRidePreferencesForDriver_InvalidRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/driver/rides/invalid-uuid/preferences", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetRidePreferencesForDriver(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid ride ID")
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_UpdatePreferences_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockRepository, uuid.UUID)
		request        UpdatePreferencesRequest
		expectedStatus int
	}{
		{
			name: "update temperature only",
			setupMock: func(m *MockRepository, userID uuid.UUID) {
				prefs := createTestRiderPreferences(userID)
				m.On("GetRiderPreferences", mock.Anything, userID).Return(prefs, nil)
				m.On("UpsertRiderPreferences", mock.Anything, mock.AnythingOfType("*preferences.RiderPreferences")).Return(nil)
			},
			request: func() UpdatePreferencesRequest {
				temp := TempCool
				return UpdatePreferencesRequest{Temperature: &temp}
			}(),
			expectedStatus: http.StatusOK,
		},
		{
			name: "update music only",
			setupMock: func(m *MockRepository, userID uuid.UUID) {
				prefs := createTestRiderPreferences(userID)
				m.On("GetRiderPreferences", mock.Anything, userID).Return(prefs, nil)
				m.On("UpsertRiderPreferences", mock.Anything, mock.AnythingOfType("*preferences.RiderPreferences")).Return(nil)
			},
			request: func() UpdatePreferencesRequest {
				music := MusicQuiet
				return UpdatePreferencesRequest{Music: &music}
			}(),
			expectedStatus: http.StatusOK,
		},
		{
			name: "update multiple fields",
			setupMock: func(m *MockRepository, userID uuid.UUID) {
				prefs := createTestRiderPreferences(userID)
				m.On("GetRiderPreferences", mock.Anything, userID).Return(prefs, nil)
				m.On("UpsertRiderPreferences", mock.Anything, mock.AnythingOfType("*preferences.RiderPreferences")).Return(nil)
			},
			request: func() UpdatePreferencesRequest {
				temp := TempWarm
				music := MusicLow
				conv := ConversationQuiet
				childSeat := true
				return UpdatePreferencesRequest{
					Temperature:  &temp,
					Music:        &music,
					Conversation: &conv,
					ChildSeat:    &childSeat,
				}
			}(),
			expectedStatus: http.StatusOK,
		},
		{
			name: "get preferences error",
			setupMock: func(m *MockRepository, userID uuid.UUID) {
				m.On("GetRiderPreferences", mock.Anything, userID).Return(nil, errors.New("db error"))
			},
			request: func() UpdatePreferencesRequest {
				temp := TempCool
				return UpdatePreferencesRequest{Temperature: &temp}
			}(),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "upsert error",
			setupMock: func(m *MockRepository, userID uuid.UUID) {
				prefs := createTestRiderPreferences(userID)
				m.On("GetRiderPreferences", mock.Anything, userID).Return(prefs, nil)
				m.On("UpsertRiderPreferences", mock.Anything, mock.AnythingOfType("*preferences.RiderPreferences")).Return(errors.New("db error"))
			},
			request: func() UpdatePreferencesRequest {
				temp := TempCool
				return UpdatePreferencesRequest{Temperature: &temp}
			}(),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			userID := uuid.New()
			tt.setupMock(mockRepo, userID)

			c, w := setupTestContext("PUT", "/api/v1/preferences", tt.request)
			setUserContext(c, userID, models.RoleRider)

			handler.UpdatePreferences(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestHandler_UpdateDriverCapabilities_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockRepository, uuid.UUID)
		request        UpdateDriverCapabilitiesRequest
		expectedStatus int
	}{
		{
			name: "update child seat",
			setupMock: func(m *MockRepository, driverID uuid.UUID) {
				caps := createTestDriverCapabilities(driverID)
				m.On("GetDriverCapabilities", mock.Anything, driverID).Return(caps, nil)
				m.On("UpsertDriverCapabilities", mock.Anything, mock.AnythingOfType("*preferences.DriverCapabilities")).Return(nil)
			},
			request: func() UpdateDriverCapabilitiesRequest {
				hasChildSeat := true
				return UpdateDriverCapabilitiesRequest{HasChildSeat: &hasChildSeat}
			}(),
			expectedStatus: http.StatusOK,
		},
		{
			name: "update multiple fields",
			setupMock: func(m *MockRepository, driverID uuid.UUID) {
				caps := createTestDriverCapabilities(driverID)
				m.On("GetDriverCapabilities", mock.Anything, driverID).Return(caps, nil)
				m.On("UpsertDriverCapabilities", mock.Anything, mock.AnythingOfType("*preferences.DriverCapabilities")).Return(nil)
			},
			request: func() UpdateDriverCapabilitiesRequest {
				hasChildSeat := true
				petFriendly := false
				maxPassengers := 6
				return UpdateDriverCapabilitiesRequest{
					HasChildSeat:  &hasChildSeat,
					PetFriendly:   &petFriendly,
					MaxPassengers: &maxPassengers,
				}
			}(),
			expectedStatus: http.StatusOK,
		},
		{
			name: "update languages",
			setupMock: func(m *MockRepository, driverID uuid.UUID) {
				caps := createTestDriverCapabilities(driverID)
				m.On("GetDriverCapabilities", mock.Anything, driverID).Return(caps, nil)
				m.On("UpsertDriverCapabilities", mock.Anything, mock.AnythingOfType("*preferences.DriverCapabilities")).Return(nil)
			},
			request: UpdateDriverCapabilitiesRequest{
				Languages: []string{"en", "fr", "de"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "create new capabilities (no existing)",
			setupMock: func(m *MockRepository, driverID uuid.UUID) {
				m.On("GetDriverCapabilities", mock.Anything, driverID).Return(nil, errors.New("not found"))
				m.On("UpsertDriverCapabilities", mock.Anything, mock.AnythingOfType("*preferences.DriverCapabilities")).Return(nil)
			},
			request: func() UpdateDriverCapabilitiesRequest {
				hasChildSeat := true
				return UpdateDriverCapabilitiesRequest{HasChildSeat: &hasChildSeat}
			}(),
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			driverID := uuid.New()
			tt.setupMock(mockRepo, driverID)

			c, w := setupTestContext("PUT", "/api/v1/driver/capabilities", tt.request)
			setUserContext(c, driverID, models.RoleDriver)

			handler.UpdateDriverCapabilities(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_GetPreferences_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	prefs := createTestRiderPreferences(userID)

	mockRepo.On("GetRiderPreferences", mock.Anything, userID).Return(prefs, nil)

	c, w := setupTestContext("GET", "/api/v1/preferences", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetPreferences(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	// Verify response structure
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()

	mockRepo.On("GetRiderPreferences", mock.Anything, userID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/preferences", nil)
	setUserContext(c, userID, models.RoleRider)

	handler.GetPreferences(c)

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

func TestHandler_SetRidePreferences_AllFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	rideID := uuid.New()

	temp := TempCool
	music := MusicQuiet
	conv := ConversationQuiet
	route := RouteScenic
	childSeat := true
	petFriendly := false
	notes := "Please be on time"

	req := SetRidePreferencesRequest{
		RideID:       rideID,
		Temperature:  &temp,
		Music:        &music,
		Conversation: &conv,
		Route:        &route,
		ChildSeat:    &childSeat,
		PetFriendly:  &petFriendly,
		Notes:        &notes,
	}

	mockRepo.On("SetRideOverride", mock.Anything, mock.AnythingOfType("*preferences.RidePreferenceOverride")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/preferences/rides", req)
	setUserContext(c, userID, models.RoleRider)

	handler.SetRidePreferences(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetRidePreferences_NoOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	rideID := uuid.New()
	prefs := createTestRiderPreferences(userID)

	mockRepo.On("GetRiderPreferences", mock.Anything, userID).Return(prefs, nil)
	mockRepo.On("GetRideOverride", mock.Anything, rideID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/preferences/rides/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetRidePreferences(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetRidePreferencesForDriver_NoRiderID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	rideID := uuid.New()

	// When rider_id is not provided, service should handle uuid.Nil
	mockRepo.On("GetRiderPreferences", mock.Anything, uuid.Nil).Return(nil, errors.New("not found"))
	mockRepo.On("GetRideOverride", mock.Anything, rideID).Return(nil, errors.New("not found"))
	mockRepo.On("GetDriverCapabilities", mock.Anything, driverID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/driver/rides/"+rideID.String()+"/preferences", nil)
	c.Params = gin.Params{{Key: "id", Value: rideID.String()}}
	// No rider_id query param
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetRidePreferencesForDriver(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_UpdateDriverCapabilities_EmptyRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	caps := createTestDriverCapabilities(driverID)

	req := UpdateDriverCapabilitiesRequest{}

	mockRepo.On("GetDriverCapabilities", mock.Anything, driverID).Return(caps, nil)
	mockRepo.On("UpsertDriverCapabilities", mock.Anything, mock.AnythingOfType("*preferences.DriverCapabilities")).Return(nil)

	c, w := setupTestContext("PUT", "/api/v1/driver/capabilities", req)
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateDriverCapabilities(c)

	// Should succeed even with empty request (no changes)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_UpdatePreferences_EmptyRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	userID := uuid.New()
	prefs := createTestRiderPreferences(userID)

	req := UpdatePreferencesRequest{}

	mockRepo.On("GetRiderPreferences", mock.Anything, userID).Return(prefs, nil)
	mockRepo.On("UpsertRiderPreferences", mock.Anything, mock.AnythingOfType("*preferences.RiderPreferences")).Return(nil)

	c, w := setupTestContext("PUT", "/api/v1/preferences", req)
	setUserContext(c, userID, models.RoleRider)

	handler.UpdatePreferences(c)

	// Should succeed even with empty request (no changes)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================================
// Authorization Tests
// ============================================================================

func TestHandler_AuthorizationCases(t *testing.T) {
	tests := []struct {
		name           string
		endpoint       func(*Handler, *gin.Context)
		method         string
		path           string
		body           interface{}
		hasUserContext bool
		expectedStatus int
	}{
		{
			name:           "GetPreferences - unauthorized",
			endpoint:       func(h *Handler, c *gin.Context) { h.GetPreferences(c) },
			method:         "GET",
			path:           "/api/v1/preferences",
			hasUserContext: false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "UpdatePreferences - unauthorized",
			endpoint:       func(h *Handler, c *gin.Context) { h.UpdatePreferences(c) },
			method:         "PUT",
			path:           "/api/v1/preferences",
			body:           UpdatePreferencesRequest{},
			hasUserContext: false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "SetRidePreferences - unauthorized",
			endpoint:       func(h *Handler, c *gin.Context) { h.SetRidePreferences(c) },
			method:         "POST",
			path:           "/api/v1/preferences/rides",
			body:           SetRidePreferencesRequest{RideID: uuid.New()},
			hasUserContext: false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "GetRidePreferences - unauthorized",
			endpoint: func(h *Handler, c *gin.Context) {
				c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
				h.GetRidePreferences(c)
			},
			method:         "GET",
			path:           "/api/v1/preferences/rides/:id",
			hasUserContext: false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "GetDriverCapabilities - unauthorized",
			endpoint:       func(h *Handler, c *gin.Context) { h.GetDriverCapabilities(c) },
			method:         "GET",
			path:           "/api/v1/driver/capabilities",
			hasUserContext: false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "UpdateDriverCapabilities - unauthorized",
			endpoint:       func(h *Handler, c *gin.Context) { h.UpdateDriverCapabilities(c) },
			method:         "PUT",
			path:           "/api/v1/driver/capabilities",
			body:           UpdateDriverCapabilitiesRequest{},
			hasUserContext: false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "GetRidePreferencesForDriver - unauthorized",
			endpoint: func(h *Handler, c *gin.Context) {
				c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}
				h.GetRidePreferencesForDriver(c)
			},
			method:         "GET",
			path:           "/api/v1/driver/rides/:id/preferences",
			hasUserContext: false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			c, w := setupTestContext(tt.method, tt.path, tt.body)

			if tt.hasUserContext {
				setUserContext(c, uuid.New(), models.RoleRider)
			}

			tt.endpoint(handler, c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
