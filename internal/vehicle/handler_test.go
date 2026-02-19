package vehicle

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
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Mock Repository
// ============================================================================

// MockRepository is a mock implementation of RepositoryInterface for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateVehicle(ctx context.Context, v *Vehicle) error {
	args := m.Called(ctx, v)
	return args.Error(0)
}

func (m *MockRepository) GetVehicleByID(ctx context.Context, id uuid.UUID) (*Vehicle, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Vehicle), args.Error(1)
}

func (m *MockRepository) GetVehiclesByDriver(ctx context.Context, driverID uuid.UUID, limit, offset int) ([]Vehicle, int64, error) {
	args := m.Called(ctx, driverID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]Vehicle), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) GetPrimaryVehicle(ctx context.Context, driverID uuid.UUID) (*Vehicle, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Vehicle), args.Error(1)
}

func (m *MockRepository) UpdateVehicle(ctx context.Context, v *Vehicle) error {
	args := m.Called(ctx, v)
	return args.Error(0)
}

func (m *MockRepository) UpdateVehiclePhotos(ctx context.Context, vehicleID uuid.UUID, photos *UploadVehiclePhotosRequest) error {
	args := m.Called(ctx, vehicleID, photos)
	return args.Error(0)
}

func (m *MockRepository) UpdateVehicleStatus(ctx context.Context, vehicleID uuid.UUID, status VehicleStatus, reason *string) error {
	args := m.Called(ctx, vehicleID, status, reason)
	return args.Error(0)
}

func (m *MockRepository) SetPrimaryVehicle(ctx context.Context, driverID, vehicleID uuid.UUID) error {
	args := m.Called(ctx, driverID, vehicleID)
	return args.Error(0)
}

func (m *MockRepository) RetireVehicle(ctx context.Context, vehicleID uuid.UUID) error {
	args := m.Called(ctx, vehicleID)
	return args.Error(0)
}

func (m *MockRepository) GetAllVehicles(ctx context.Context, filter *AdminVehicleFilter, limit, offset int) ([]Vehicle, int64, error) {
	args := m.Called(ctx, filter, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]Vehicle), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) GetPendingReviewVehicles(ctx context.Context, limit, offset int) ([]Vehicle, int, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]Vehicle), args.Int(1), args.Error(2)
}

func (m *MockRepository) GetVehicleStats(ctx context.Context) (*VehicleStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*VehicleStats), args.Error(1)
}

func (m *MockRepository) GetExpiringVehicles(ctx context.Context, daysAhead int) ([]Vehicle, error) {
	args := m.Called(ctx, daysAhead)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Vehicle), args.Error(1)
}

func (m *MockRepository) CreateInspection(ctx context.Context, insp *VehicleInspection) error {
	args := m.Called(ctx, insp)
	return args.Error(0)
}

func (m *MockRepository) GetInspectionsByVehicle(ctx context.Context, vehicleID uuid.UUID) ([]VehicleInspection, error) {
	args := m.Called(ctx, vehicleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]VehicleInspection), args.Error(1)
}

func (m *MockRepository) CreateMaintenanceReminder(ctx context.Context, rem *MaintenanceReminder) error {
	args := m.Called(ctx, rem)
	return args.Error(0)
}

func (m *MockRepository) GetPendingReminders(ctx context.Context, vehicleID uuid.UUID) ([]MaintenanceReminder, error) {
	args := m.Called(ctx, vehicleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]MaintenanceReminder), args.Error(1)
}

func (m *MockRepository) CompleteReminder(ctx context.Context, reminderID uuid.UUID) error {
	args := m.Called(ctx, reminderID)
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

func createTestVehicle(driverID uuid.UUID) *Vehicle {
	now := time.Now()
	return &Vehicle{
		ID:            uuid.New(),
		DriverID:      driverID,
		Status:        VehicleStatusPending,
		Category:      VehicleCategoryComfort,
		Make:          "Toyota",
		Model:         "Camry",
		Year:          2022,
		Color:         "Black",
		LicensePlate:  "ABC123",
		FuelType:      FuelTypeGasoline,
		MaxPassengers: 4,
		IsActive:      true,
		IsPrimary:     false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func createApprovedVehicle(driverID uuid.UUID) *Vehicle {
	vehicle := createTestVehicle(driverID)
	vehicle.Status = VehicleStatusApproved
	return vehicle
}

func createTestHandler(mockRepo *MockRepository) *Handler {
	service := NewService(mockRepo)
	return NewHandler(service)
}

// ============================================================================
// Register Vehicle Handler Tests
// ============================================================================

func TestHandler_Register_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	reqBody := RegisterVehicleRequest{
		Make:         "Toyota",
		Model:        "Camry",
		Year:         2022,
		Color:        "Black",
		LicensePlate: "ABC123",
		Category:     VehicleCategoryComfort,
		FuelType:     FuelTypeGasoline,
	}

	mockRepo.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return([]Vehicle{}, int64(0), nil)
	mockRepo.On("CreateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.Vehicle")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.Register(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_Register_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	reqBody := RegisterVehicleRequest{
		Make:         "Toyota",
		Model:        "Camry",
		Year:         2022,
		Color:        "Black",
		LicensePlate: "ABC123",
		Category:     VehicleCategoryComfort,
		FuelType:     FuelTypeGasoline,
	}

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles", reqBody)
	// Don't set user context

	handler.Register(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Register_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/driver/vehicles", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, driverID, models.RoleDriver)

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Register_MissingRequiredFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	reqBody := map[string]interface{}{
		"make": "Toyota",
		// Missing required fields
	}

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Register_InvalidYear_TooOld(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	reqBody := RegisterVehicleRequest{
		Make:         "Toyota",
		Model:        "Camry",
		Year:         2005, // Too old (min is 2010)
		Color:        "Black",
		LicensePlate: "ABC123",
		Category:     VehicleCategoryComfort,
		FuelType:     FuelTypeGasoline,
	}

	// Year validation happens before any DB call in the service

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_Register_InvalidYear_Future(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	futureYear := time.Now().Year() + 5 // More than 1 year in the future
	reqBody := RegisterVehicleRequest{
		Make:         "Toyota",
		Model:        "Camry",
		Year:         futureYear,
		Color:        "Black",
		LicensePlate: "ABC123",
		Category:     VehicleCategoryComfort,
		FuelType:     FuelTypeGasoline,
	}

	// Year validation happens before any DB call in the service

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_Register_MaxVehiclesReached(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	reqBody := RegisterVehicleRequest{
		Make:         "Toyota",
		Model:        "Camry",
		Year:         2022,
		Color:        "Black",
		LicensePlate: "ABC123",
		Category:     VehicleCategoryComfort,
		FuelType:     FuelTypeGasoline,
	}

	// Already has 3 active vehicles
	existingVehicles := []Vehicle{
		{ID: uuid.New(), DriverID: driverID, IsActive: true},
		{ID: uuid.New(), DriverID: driverID, IsActive: true},
		{ID: uuid.New(), DriverID: driverID, IsActive: true},
	}
	mockRepo.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return(existingVehicles, int64(len(existingVehicles)), nil)

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_Register_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	reqBody := RegisterVehicleRequest{
		Make:         "Toyota",
		Model:        "Camry",
		Year:         2022,
		Color:        "Black",
		LicensePlate: "ABC123",
		Category:     VehicleCategoryComfort,
		FuelType:     FuelTypeGasoline,
	}

	mockRepo.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return(nil, int64(0), errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.Register(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_Register_FirstVehicleIsPrimary(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	reqBody := RegisterVehicleRequest{
		Make:         "Toyota",
		Model:        "Camry",
		Year:         2022,
		Color:        "Black",
		LicensePlate: "ABC123",
		Category:     VehicleCategoryComfort,
		FuelType:     FuelTypeGasoline,
	}

	mockRepo.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return([]Vehicle{}, int64(0), nil)
	mockRepo.On("CreateVehicle", mock.Anything, mock.MatchedBy(func(v *Vehicle) bool {
		return v.IsPrimary == true // First vehicle should be primary
	})).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.Register(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_Register_WithAllCategories(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categories := []VehicleCategory{
		VehicleCategoryEconomy,
		VehicleCategoryComfort,
		VehicleCategoryPremium,
		VehicleCategoryLux,
		VehicleCategoryXL,
		VehicleCategoryWAV,
		VehicleCategoryElectric,
	}

	for _, category := range categories {
		t.Run(string(category), func(t *testing.T) {
			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			driverID := uuid.New()
			reqBody := RegisterVehicleRequest{
				Make:         "Toyota",
				Model:        "Camry",
				Year:         2022,
				Color:        "Black",
				LicensePlate: "ABC123",
				Category:     category,
				FuelType:     FuelTypeGasoline,
			}

			mockRepo.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return([]Vehicle{}, int64(0), nil)
			mockRepo.On("CreateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.Vehicle")).Return(nil)

			c, w := setupTestContext("POST", "/api/v1/driver/vehicles", reqBody)
			setUserContext(c, driverID, models.RoleDriver)

			handler.Register(c)

			assert.Equal(t, http.StatusCreated, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

// ============================================================================
// GetMyVehicles Handler Tests
// ============================================================================

func TestHandler_GetMyVehicles_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vehicles := []Vehicle{
		*createTestVehicle(driverID),
		*createTestVehicle(driverID),
	}

	mockRepo.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return(vehicles, int64(len(vehicles)), nil)

	c, w := setupTestContext("GET", "/api/v1/driver/vehicles", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetMyVehicles(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	meta := response["meta"].(map[string]interface{})
	assert.Equal(t, float64(2), meta["total"])
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetMyVehicles_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	mockRepo.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return([]Vehicle{}, int64(0), nil)

	c, w := setupTestContext("GET", "/api/v1/driver/vehicles", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetMyVehicles(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["meta"])
	// total omitted when 0 (omitempty); vehicles key present and empty is the key check
	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["vehicles"])
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetMyVehicles_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	c, w := setupTestContext("GET", "/api/v1/driver/vehicles", nil)

	handler.GetMyVehicles(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetMyVehicles_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	mockRepo.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return(nil, int64(0), errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/driver/vehicles", nil)
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetMyVehicles(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetVehicle Handler Tests
// ============================================================================

func TestHandler_GetVehicle_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createTestVehicle(driverID)
	vehicle.ID = vehicleID

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/vehicles/"+vehicleID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetVehicle(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetVehicle_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/driver/vehicles/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetVehicle(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetVehicle_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vehicleID := uuid.New()

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(nil, common.NewNotFoundError("vehicle not found", nil))

	c, w := setupTestContext("GET", "/api/v1/driver/vehicles/"+vehicleID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetVehicle(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetVehicle_NotOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	otherDriverID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createTestVehicle(otherDriverID) // Owned by another driver
	vehicle.ID = vehicleID

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/vehicles/"+vehicleID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetVehicle(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetVehicle_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	vehicleID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/driver/vehicles/"+vehicleID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}

	handler.GetVehicle(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// UpdateVehicle Handler Tests
// ============================================================================

func TestHandler_UpdateVehicle_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createTestVehicle(driverID)
	vehicle.ID = vehicleID

	newColor := "Red"
	reqBody := UpdateVehicleRequest{
		Color: &newColor,
	}

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
	mockRepo.On("UpdateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.Vehicle")).Return(nil)

	c, w := setupTestContext("PUT", "/api/v1/driver/vehicles/"+vehicleID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateVehicle(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_UpdateVehicle_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	newColor := "Red"
	reqBody := UpdateVehicleRequest{
		Color: &newColor,
	}

	c, w := setupTestContext("PUT", "/api/v1/driver/vehicles/invalid-uuid", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateVehicle(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateVehicle_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vehicleID := uuid.New()

	c, w := setupTestContext("PUT", "/api/v1/driver/vehicles/"+vehicleID.String(), nil)
	c.Request = httptest.NewRequest("PUT", "/api/v1/driver/vehicles/"+vehicleID.String(), bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateVehicle(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateVehicle_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vehicleID := uuid.New()
	newColor := "Red"
	reqBody := UpdateVehicleRequest{
		Color: &newColor,
	}

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(nil, common.NewNotFoundError("vehicle not found", nil))

	c, w := setupTestContext("PUT", "/api/v1/driver/vehicles/"+vehicleID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateVehicle(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_UpdateVehicle_NotOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	otherDriverID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createTestVehicle(otherDriverID)
	vehicle.ID = vehicleID

	newColor := "Red"
	reqBody := UpdateVehicleRequest{
		Color: &newColor,
	}

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)

	c, w := setupTestContext("PUT", "/api/v1/driver/vehicles/"+vehicleID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateVehicle(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_UpdateVehicle_AllFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createTestVehicle(driverID)
	vehicle.ID = vehicleID

	newColor := "Red"
	newCategory := VehicleCategoryPremium
	newMaxPassengers := 6
	hasChildSeat := true
	hasWheelchairAccess := true
	hasWifi := true
	hasCharger := true
	petFriendly := true
	luggageCapacity := 4

	reqBody := UpdateVehicleRequest{
		Color:               &newColor,
		Category:            &newCategory,
		MaxPassengers:       &newMaxPassengers,
		HasChildSeat:        &hasChildSeat,
		HasWheelchairAccess: &hasWheelchairAccess,
		HasWifi:             &hasWifi,
		HasCharger:          &hasCharger,
		PetFriendly:         &petFriendly,
		LuggageCapacity:     &luggageCapacity,
	}

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
	mockRepo.On("UpdateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.Vehicle")).Return(nil)

	c, w := setupTestContext("PUT", "/api/v1/driver/vehicles/"+vehicleID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateVehicle(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// UploadPhotos Handler Tests
// ============================================================================

func TestHandler_UploadPhotos_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createTestVehicle(driverID)
	vehicle.ID = vehicleID

	frontURL := "https://example.com/front.jpg"
	reqBody := UploadVehiclePhotosRequest{
		FrontPhotoURL: &frontURL,
	}

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
	mockRepo.On("UpdateVehiclePhotos", mock.Anything, vehicleID, &reqBody).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles/"+vehicleID.String()+"/photos", reqBody)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UploadPhotos(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_UploadPhotos_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	frontURL := "https://example.com/front.jpg"
	reqBody := UploadVehiclePhotosRequest{
		FrontPhotoURL: &frontURL,
	}

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles/invalid-uuid/photos", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UploadPhotos(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UploadPhotos_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vehicleID := uuid.New()
	frontURL := "https://example.com/front.jpg"
	reqBody := UploadVehiclePhotosRequest{
		FrontPhotoURL: &frontURL,
	}

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(nil, common.NewNotFoundError("vehicle not found", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles/"+vehicleID.String()+"/photos", reqBody)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UploadPhotos(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_UploadPhotos_NotOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	otherDriverID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createTestVehicle(otherDriverID)
	vehicle.ID = vehicleID

	frontURL := "https://example.com/front.jpg"
	reqBody := UploadVehiclePhotosRequest{
		FrontPhotoURL: &frontURL,
	}

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles/"+vehicleID.String()+"/photos", reqBody)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UploadPhotos(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_UploadPhotos_AllPhotoTypes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createTestVehicle(driverID)
	vehicle.ID = vehicleID

	regURL := "https://example.com/registration.jpg"
	insURL := "https://example.com/insurance.jpg"
	frontURL := "https://example.com/front.jpg"
	backURL := "https://example.com/back.jpg"
	sideURL := "https://example.com/side.jpg"
	interiorURL := "https://example.com/interior.jpg"

	reqBody := UploadVehiclePhotosRequest{
		RegistrationPhotoURL: &regURL,
		InsurancePhotoURL:    &insURL,
		FrontPhotoURL:        &frontURL,
		BackPhotoURL:         &backURL,
		SidePhotoURL:         &sideURL,
		InteriorPhotoURL:     &interiorURL,
	}

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
	mockRepo.On("UpdateVehiclePhotos", mock.Anything, vehicleID, &reqBody).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles/"+vehicleID.String()+"/photos", reqBody)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UploadPhotos(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// SetPrimary Handler Tests
// ============================================================================

func TestHandler_SetPrimary_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createApprovedVehicle(driverID)
	vehicle.ID = vehicleID

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
	mockRepo.On("SetPrimaryVehicle", mock.Anything, driverID, vehicleID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles/"+vehicleID.String()+"/primary", nil)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.SetPrimary(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_SetPrimary_VehicleNotApproved(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createTestVehicle(driverID) // Status is pending
	vehicle.ID = vehicleID

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles/"+vehicleID.String()+"/primary", nil)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.SetPrimary(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_SetPrimary_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles/invalid-uuid/primary", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.SetPrimary(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SetPrimary_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vehicleID := uuid.New()

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(nil, common.NewNotFoundError("vehicle not found", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles/"+vehicleID.String()+"/primary", nil)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.SetPrimary(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_SetPrimary_NotOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	otherDriverID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createApprovedVehicle(otherDriverID)
	vehicle.ID = vehicleID

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles/"+vehicleID.String()+"/primary", nil)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.SetPrimary(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// Retire Handler Tests
// ============================================================================

func TestHandler_Retire_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createTestVehicle(driverID)
	vehicle.ID = vehicleID

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
	mockRepo.On("RetireVehicle", mock.Anything, vehicleID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles/"+vehicleID.String()+"/retire", nil)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.Retire(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_Retire_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles/invalid-uuid/retire", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.Retire(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Retire_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vehicleID := uuid.New()

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(nil, common.NewNotFoundError("vehicle not found", nil))

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles/"+vehicleID.String()+"/retire", nil)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.Retire(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_Retire_NotOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	otherDriverID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createTestVehicle(otherDriverID)
	vehicle.ID = vehicleID

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles/"+vehicleID.String()+"/retire", nil)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.Retire(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetMaintenanceReminders Handler Tests
// ============================================================================

func TestHandler_GetMaintenanceReminders_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createTestVehicle(driverID)
	vehicle.ID = vehicleID

	reminders := []MaintenanceReminder{
		{ID: uuid.New(), VehicleID: vehicleID, Type: "oil_change", Description: "Oil change due"},
		{ID: uuid.New(), VehicleID: vehicleID, Type: "tire_rotation", Description: "Tire rotation due"},
	}

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
	mockRepo.On("GetPendingReminders", mock.Anything, vehicleID).Return(reminders, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/vehicles/"+vehicleID.String()+"/maintenance", nil)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetMaintenanceReminders(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetMaintenanceReminders_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createTestVehicle(driverID)
	vehicle.ID = vehicleID

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
	mockRepo.On("GetPendingReminders", mock.Anything, vehicleID).Return([]MaintenanceReminder{}, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/vehicles/"+vehicleID.String()+"/maintenance", nil)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetMaintenanceReminders(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetMaintenanceReminders_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/driver/vehicles/invalid-uuid/maintenance", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetMaintenanceReminders(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetMaintenanceReminders_NotOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	otherDriverID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createTestVehicle(otherDriverID)
	vehicle.ID = vehicleID

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/vehicles/"+vehicleID.String()+"/maintenance", nil)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetMaintenanceReminders(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// CompleteMaintenance Handler Tests
// ============================================================================

func TestHandler_CompleteMaintenance_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	reminderID := uuid.New()

	mockRepo.On("CompleteReminder", mock.Anything, reminderID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles/maintenance/"+reminderID.String()+"/complete", nil)
	c.Params = gin.Params{{Key: "reminderId", Value: reminderID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.CompleteMaintenance(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_CompleteMaintenance_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles/maintenance/invalid-uuid/complete", nil)
	c.Params = gin.Params{{Key: "reminderId", Value: "invalid-uuid"}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.CompleteMaintenance(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CompleteMaintenance_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	reminderID := uuid.New()

	mockRepo.On("CompleteReminder", mock.Anything, reminderID).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles/maintenance/"+reminderID.String()+"/complete", nil)
	c.Params = gin.Params{{Key: "reminderId", Value: reminderID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.CompleteMaintenance(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetInspections Handler Tests
// ============================================================================

func TestHandler_GetInspections_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createTestVehicle(driverID)
	vehicle.ID = vehicleID

	inspections := []VehicleInspection{
		{ID: uuid.New(), VehicleID: vehicleID, Status: "passed", ScheduledAt: time.Now()},
		{ID: uuid.New(), VehicleID: vehicleID, Status: "scheduled", ScheduledAt: time.Now().AddDate(0, 1, 0)},
	}

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
	mockRepo.On("GetInspectionsByVehicle", mock.Anything, vehicleID).Return(inspections, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/vehicles/"+vehicleID.String()+"/inspections", nil)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetInspections(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetInspections_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/driver/vehicles/invalid-uuid/inspections", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetInspections(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetInspections_NotOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	otherDriverID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createTestVehicle(otherDriverID)
	vehicle.ID = vehicleID

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)

	c, w := setupTestContext("GET", "/api/v1/driver/vehicles/"+vehicleID.String()+"/inspections", nil)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetInspections(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// Admin - AdminReview Handler Tests
// ============================================================================

func TestHandler_AdminReview_ApproveSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createTestVehicle(uuid.New())
	vehicle.ID = vehicleID
	vehicle.Status = VehicleStatusPending

	reqBody := AdminReviewRequest{
		Approved: true,
	}

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
	mockRepo.On("UpdateVehicleStatus", mock.Anything, vehicleID, VehicleStatusApproved, (*string)(nil)).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/vehicles/"+vehicleID.String()+"/review", reqBody)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminReview(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_AdminReview_RejectSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createTestVehicle(uuid.New())
	vehicle.ID = vehicleID
	vehicle.Status = VehicleStatusPending

	reason := "Vehicle does not meet safety requirements"
	reqBody := AdminReviewRequest{
		Approved:        false,
		RejectionReason: &reason,
	}

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
	mockRepo.On("UpdateVehicleStatus", mock.Anything, vehicleID, VehicleStatusRejected, &reason).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/vehicles/"+vehicleID.String()+"/review", reqBody)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminReview(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_AdminReview_RejectWithoutReason(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createTestVehicle(uuid.New())
	vehicle.ID = vehicleID
	vehicle.Status = VehicleStatusPending

	reqBody := AdminReviewRequest{
		Approved: false,
		// No rejection reason
	}

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)

	c, w := setupTestContext("POST", "/api/v1/admin/vehicles/"+vehicleID.String()+"/review", reqBody)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminReview(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_AdminReview_VehicleNotPending(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createApprovedVehicle(uuid.New()) // Already approved
	vehicle.ID = vehicleID

	reqBody := AdminReviewRequest{
		Approved: true,
	}

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)

	c, w := setupTestContext("POST", "/api/v1/admin/vehicles/"+vehicleID.String()+"/review", reqBody)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminReview(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_AdminReview_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	reqBody := AdminReviewRequest{
		Approved: true,
	}

	c, w := setupTestContext("POST", "/api/v1/admin/vehicles/invalid-uuid/review", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminReview(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminReview_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	vehicleID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/admin/vehicles/"+vehicleID.String()+"/review", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/vehicles/"+vehicleID.String()+"/review", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminReview(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminReview_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	vehicleID := uuid.New()

	reqBody := AdminReviewRequest{
		Approved: true,
	}

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(nil, common.NewNotFoundError("vehicle not found", nil))

	c, w := setupTestContext("POST", "/api/v1/admin/vehicles/"+vehicleID.String()+"/review", reqBody)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminReview(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// Admin - AdminGetPending Handler Tests
// ============================================================================

func TestHandler_AdminGetPending_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	vehicles := []Vehicle{
		*createTestVehicle(uuid.New()),
		*createTestVehicle(uuid.New()),
	}

	// Handler passes params.Limit and params.Offset directly to service.GetPendingReviews.
	// With defaults limit=20, offset=0, the service passes (20, 0) to the repo.
	mockRepo.On("GetPendingReviewVehicles", mock.Anything, 20, 0).Return(vehicles, 2, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/vehicles/pending", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetPending(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_AdminGetPending_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()

	mockRepo.On("GetPendingReviewVehicles", mock.Anything, 20, 0).Return([]Vehicle{}, 0, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/vehicles/pending", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetPending(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_AdminGetPending_WithPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	vehicles := []Vehicle{
		*createTestVehicle(uuid.New()),
	}

	mockRepo.On("GetPendingReviewVehicles", mock.Anything, 5, 10).Return(vehicles, 10, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/vehicles/pending?limit=5&offset=10", nil)
	c.Request.URL.RawQuery = "limit=5&offset=10"
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetPending(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_AdminGetPending_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()

	mockRepo.On("GetPendingReviewVehicles", mock.Anything, 20, 0).Return(nil, 0, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/vehicles/pending", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetPending(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// Admin - AdminGetStats Handler Tests
// ============================================================================

func TestHandler_AdminGetStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	stats := &VehicleStats{
		TotalVehicles:        100,
		PendingReview:        10,
		ApprovedVehicles:     80,
		SuspendedVehicles:    5,
		ExpiringInsurance:    3,
		ExpiringRegistration: 2,
	}

	mockRepo.On("GetVehicleStats", mock.Anything).Return(stats, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/vehicles/stats", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_AdminGetStats_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()

	mockRepo.On("GetVehicleStats", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/vehicles/stats", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetStats(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// Admin - AdminSuspend Handler Tests
// ============================================================================

func TestHandler_AdminSuspend_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	vehicleID := uuid.New()

	reqBody := map[string]interface{}{
		"reason": "Safety violation reported",
	}

	mockRepo.On("UpdateVehicleStatus", mock.Anything, vehicleID, VehicleStatusSuspended, mock.AnythingOfType("*string")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/vehicles/"+vehicleID.String()+"/suspend", reqBody)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminSuspend(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_AdminSuspend_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	reqBody := map[string]interface{}{
		"reason": "Safety violation",
	}

	c, w := setupTestContext("POST", "/api/v1/admin/vehicles/invalid-uuid/suspend", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminSuspend(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminSuspend_MissingReason(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	vehicleID := uuid.New()

	reqBody := map[string]interface{}{
		// No reason provided
	}

	c, w := setupTestContext("POST", "/api/v1/admin/vehicles/"+vehicleID.String()+"/suspend", reqBody)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminSuspend(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminSuspend_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	vehicleID := uuid.New()

	reqBody := map[string]interface{}{
		"reason": "Safety violation",
	}

	mockRepo.On("UpdateVehicleStatus", mock.Anything, vehicleID, VehicleStatusSuspended, mock.AnythingOfType("*string")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/admin/vehicles/"+vehicleID.String()+"/suspend", reqBody)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminSuspend(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// Admin - AdminGetExpiring Handler Tests
// ============================================================================

func TestHandler_AdminGetExpiring_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	vehicles := []Vehicle{
		*createTestVehicle(uuid.New()),
	}

	mockRepo.On("GetExpiringVehicles", mock.Anything, 30).Return(vehicles, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/vehicles/expiring", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetExpiring(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(30), data["days"])
	mockRepo.AssertExpectations(t)
}

func TestHandler_AdminGetExpiring_WithCustomDays(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()
	vehicles := []Vehicle{}

	mockRepo.On("GetExpiringVehicles", mock.Anything, 60).Return(vehicles, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/vehicles/expiring?days=60", nil)
	c.Request.URL.RawQuery = "days=60"
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetExpiring(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(60), data["days"])
	mockRepo.AssertExpectations(t)
}

func TestHandler_AdminGetExpiring_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()

	mockRepo.On("GetExpiringVehicles", mock.Anything, 30).Return([]Vehicle{}, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/vehicles/expiring", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetExpiring(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["count"])
	mockRepo.AssertExpectations(t)
}

func TestHandler_AdminGetExpiring_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()

	mockRepo.On("GetExpiringVehicles", mock.Anything, 30).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/vehicles/expiring", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetExpiring(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_Register_ValidationCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           interface{}
		setupMock      func(*MockRepository, uuid.UUID)
		expectedStatus int
	}{
		{
			name: "valid registration",
			body: RegisterVehicleRequest{
				Make:         "Toyota",
				Model:        "Camry",
				Year:         2022,
				Color:        "Black",
				LicensePlate: "ABC123",
				Category:     VehicleCategoryComfort,
				FuelType:     FuelTypeGasoline,
			},
			setupMock: func(m *MockRepository, driverID uuid.UUID) {
				m.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return([]Vehicle{}, int64(0), nil)
				m.On("CreateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.Vehicle")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "missing make",
			body: map[string]interface{}{
				"model":         "Camry",
				"year":          2022,
				"color":         "Black",
				"license_plate": "ABC123",
				"category":      "comfort",
				"fuel_type":     "gasoline",
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing model",
			body: map[string]interface{}{
				"make":          "Toyota",
				"year":          2022,
				"color":         "Black",
				"license_plate": "ABC123",
				"category":      "comfort",
				"fuel_type":     "gasoline",
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing year",
			body: map[string]interface{}{
				"make":          "Toyota",
				"model":         "Camry",
				"color":         "Black",
				"license_plate": "ABC123",
				"category":      "comfort",
				"fuel_type":     "gasoline",
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing color",
			body: map[string]interface{}{
				"make":          "Toyota",
				"model":         "Camry",
				"year":          2022,
				"license_plate": "ABC123",
				"category":      "comfort",
				"fuel_type":     "gasoline",
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing license_plate",
			body: map[string]interface{}{
				"make":      "Toyota",
				"model":     "Camry",
				"year":      2022,
				"color":     "Black",
				"category":  "comfort",
				"fuel_type": "gasoline",
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing category",
			body: map[string]interface{}{
				"make":          "Toyota",
				"model":         "Camry",
				"year":          2022,
				"color":         "Black",
				"license_plate": "ABC123",
				"fuel_type":     "gasoline",
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing fuel_type",
			body: map[string]interface{}{
				"make":          "Toyota",
				"model":         "Camry",
				"year":          2022,
				"color":         "Black",
				"license_plate": "ABC123",
				"category":      "comfort",
			},
			setupMock:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			driverID := uuid.New()
			if tt.setupMock != nil {
				tt.setupMock(mockRepo, driverID)
			}

			c, w := setupTestContext("POST", "/api/v1/driver/vehicles", tt.body)
			setUserContext(c, driverID, models.RoleDriver)

			handler.Register(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_FuelTypes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fuelTypes := []FuelType{
		FuelTypeGasoline,
		FuelTypeDiesel,
		FuelTypeElectric,
		FuelTypeHybrid,
		FuelTypeCNG,
		FuelTypeLPG,
	}

	for _, fuelType := range fuelTypes {
		t.Run(string(fuelType), func(t *testing.T) {
			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			driverID := uuid.New()
			reqBody := RegisterVehicleRequest{
				Make:         "Toyota",
				Model:        "Camry",
				Year:         2022,
				Color:        "Black",
				LicensePlate: "ABC123",
				Category:     VehicleCategoryComfort,
				FuelType:     fuelType,
			}

			mockRepo.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return([]Vehicle{}, int64(0), nil)
			mockRepo.On("CreateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.Vehicle")).Return(nil)

			c, w := setupTestContext("POST", "/api/v1/driver/vehicles", reqBody)
			setUserContext(c, driverID, models.RoleDriver)

			handler.Register(c)

			assert.Equal(t, http.StatusCreated, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestHandler_VehicleStatuses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		status         VehicleStatus
		canSetPrimary  bool
		expectedStatus int
	}{
		{VehicleStatusPending, false, http.StatusBadRequest},
		{VehicleStatusApproved, true, http.StatusOK},
		{VehicleStatusRejected, false, http.StatusBadRequest},
		{VehicleStatusSuspended, false, http.StatusBadRequest},
		{VehicleStatusRetired, false, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			mockRepo := new(MockRepository)
			handler := createTestHandler(mockRepo)

			driverID := uuid.New()
			vehicleID := uuid.New()
			vehicle := createTestVehicle(driverID)
			vehicle.ID = vehicleID
			vehicle.Status = tt.status

			mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
			if tt.canSetPrimary {
				mockRepo.On("SetPrimaryVehicle", mock.Anything, driverID, vehicleID).Return(nil)
			}

			c, w := setupTestContext("POST", "/api/v1/driver/vehicles/"+vehicleID.String()+"/primary", nil)
			c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
			setUserContext(c, driverID, models.RoleDriver)

			handler.SetPrimary(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_Register_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	reqBody := RegisterVehicleRequest{
		Make:          "Toyota",
		Model:         "Camry",
		Year:          2022,
		Color:         "Black",
		LicensePlate:  "ABC123",
		Category:      VehicleCategoryComfort,
		FuelType:      FuelTypeGasoline,
		MaxPassengers: 4,
	}

	mockRepo.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return([]Vehicle{}, int64(0), nil)
	mockRepo.On("CreateVehicle", mock.Anything, mock.AnythingOfType("*vehicle.Vehicle")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.Register(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	response := parseResponse(w)

	require.True(t, response["success"].(bool))
	require.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["id"])
	assert.Equal(t, "Toyota", data["make"])
	assert.Equal(t, "Camry", data["model"])
	assert.Equal(t, float64(2022), data["year"])
	assert.Equal(t, "Black", data["color"])
	assert.Equal(t, "ABC123", data["license_plate"])
	assert.Equal(t, "comfort", data["category"])
	assert.Equal(t, "gasoline", data["fuel_type"])
	assert.Equal(t, "pending", data["status"])

	mockRepo.AssertExpectations(t)
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vehicleID := uuid.New()

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(nil, common.NewNotFoundError("vehicle not found", nil))

	c, w := setupTestContext("GET", "/api/v1/driver/vehicles/"+vehicleID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.GetVehicle(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	response := parseResponse(w)

	require.False(t, response["success"].(bool))
	require.NotNil(t, response["error"])

	errorInfo := response["error"].(map[string]interface{})
	assert.NotNil(t, errorInfo["code"])
	assert.NotNil(t, errorInfo["message"])

	mockRepo.AssertExpectations(t)
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestHandler_Register_WithOptionalFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vin := "1HGBH41JXMN109186"
	reqBody := RegisterVehicleRequest{
		Make:                "Toyota",
		Model:               "Camry",
		Year:                2022,
		Color:               "Black",
		LicensePlate:        "ABC123",
		VIN:                 &vin,
		Category:            VehicleCategoryComfort,
		FuelType:            FuelTypeGasoline,
		MaxPassengers:       4,
		HasChildSeat:        true,
		HasWheelchairAccess: true,
		HasWifi:             true,
		HasCharger:          true,
		PetFriendly:         true,
		LuggageCapacity:     3,
	}

	mockRepo.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return([]Vehicle{}, int64(0), nil)
	mockRepo.On("CreateVehicle", mock.Anything, mock.MatchedBy(func(v *Vehicle) bool {
		return v.VIN != nil && *v.VIN == vin &&
			v.HasChildSeat == true &&
			v.HasWheelchairAccess == true &&
			v.HasWifi == true &&
			v.HasCharger == true &&
			v.PetFriendly == true &&
			v.LuggageCapacity == 3
	})).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.Register(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_Register_DefaultMaxPassengers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	reqBody := RegisterVehicleRequest{
		Make:         "Toyota",
		Model:        "Camry",
		Year:         2022,
		Color:        "Black",
		LicensePlate: "ABC123",
		Category:     VehicleCategoryComfort,
		FuelType:     FuelTypeGasoline,
		// MaxPassengers not set, should default to 4
	}

	mockRepo.On("GetVehiclesByDriver", mock.Anything, driverID, mock.Anything, mock.Anything).Return([]Vehicle{}, int64(0), nil)
	mockRepo.On("CreateVehicle", mock.Anything, mock.MatchedBy(func(v *Vehicle) bool {
		return v.MaxPassengers == 4 // Default value
	})).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/driver/vehicles", reqBody)
	setUserContext(c, driverID, models.RoleDriver)

	handler.Register(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_UpdateVehicle_PartialUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	driverID := uuid.New()
	vehicleID := uuid.New()
	vehicle := createTestVehicle(driverID)
	vehicle.ID = vehicleID
	vehicle.Color = "Black"
	vehicle.MaxPassengers = 4

	// Only update color, other fields should remain unchanged
	newColor := "Red"
	reqBody := UpdateVehicleRequest{
		Color: &newColor,
	}

	mockRepo.On("GetVehicleByID", mock.Anything, vehicleID).Return(vehicle, nil)
	mockRepo.On("UpdateVehicle", mock.Anything, mock.MatchedBy(func(v *Vehicle) bool {
		return v.Color == "Red" && v.MaxPassengers == 4
	})).Return(nil)

	c, w := setupTestContext("PUT", "/api/v1/driver/vehicles/"+vehicleID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}
	setUserContext(c, driverID, models.RoleDriver)

	handler.UpdateVehicle(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHandler_AdminGetPending_NullVehicles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	handler := createTestHandler(mockRepo)

	adminID := uuid.New()

	// Return nil instead of empty slice - handler should handle this gracefully
	mockRepo.On("GetPendingReviewVehicles", mock.Anything, 20, 0).Return(([]Vehicle)(nil), 0, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/vehicles/pending", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetPending(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}
