package geo

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_UpdateDriverLocation_Success(t *testing.T) {
	// Arrange
	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	ctx := context.Background()
	driverID := uuid.New()
	latitude := 37.7749
	longitude := -122.4194

	// Mock expectations
	mockRedis.On("SetWithExpiration", ctx, mock.MatchedBy(func(key string) bool {
		return key == "driver:location:"+driverID.String()
	}), mock.AnythingOfType("[]uint8"), driverLocationTTL).Return(nil)

	mockRedis.On("GeoAdd", ctx, driverGeoIndexKey, longitude, latitude, driverID.String()).Return(nil)

	// Mock H3 cell update expectations (called by updateDriverH3Cell)
	mockRedis.On("GetString", ctx, "driver:h3cell:"+driverID.String()).Return("", errors.New("not found"))
	mockRedis.On("SetWithExpiration", ctx, "driver:h3cell:"+driverID.String(), mock.AnythingOfType("[]uint8"), driverLocationTTL).Return(nil)
	mockRedis.On("SetWithExpiration", ctx, mock.MatchedBy(func(key string) bool {
		return len(key) > len(h3CellDriversPrefix) && key[:len(h3CellDriversPrefix)] == h3CellDriversPrefix
	}), mock.AnythingOfType("[]uint8"), h3CellDriversTTL).Return(nil)

	// Act
	err := service.UpdateDriverLocation(ctx, driverID, latitude, longitude)

	// Assert
	assert.NoError(t, err)
	mockRedis.AssertExpectations(t)
}

func TestService_UpdateDriverLocation_SetError(t *testing.T) {
	// Arrange
	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	ctx := context.Background()
	driverID := uuid.New()
	latitude := 37.7749
	longitude := -122.4194

	// Mock expectations - Set fails
	mockRedis.On("SetWithExpiration", ctx, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("redis error"))

	// Act
	err := service.UpdateDriverLocation(ctx, driverID, latitude, longitude)

	// Assert
	assert.Error(t, err)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 500, appErr.Code)
	mockRedis.AssertExpectations(t)
}

func TestService_UpdateDriverLocation_GeoAddError(t *testing.T) {
	// Arrange
	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	ctx := context.Background()
	driverID := uuid.New()
	latitude := 37.7749
	longitude := -122.4194

	// Mock expectations - Set succeeds, GeoAdd fails
	mockRedis.On("SetWithExpiration", ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockRedis.On("GeoAdd", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("geo index error"))

	// Act
	err := service.UpdateDriverLocation(ctx, driverID, latitude, longitude)

	// Assert
	assert.Error(t, err)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 500, appErr.Code)
	mockRedis.AssertExpectations(t)
}

func TestService_GetDriverLocation_Success(t *testing.T) {
	// Arrange
	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	ctx := context.Background()
	driverID := uuid.New()

	expectedLocation := &DriverLocation{
		DriverID:  driverID,
		Latitude:  37.7749,
		Longitude: -122.4194,
		Timestamp: time.Now(),
	}

	locationJSON, _ := json.Marshal(expectedLocation)
	mockRedis.On("GetString", ctx, "driver:location:"+driverID.String()).
		Return(string(locationJSON), nil)

	// Act
	location, err := service.GetDriverLocation(ctx, driverID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, location)
	assert.Equal(t, driverID, location.DriverID)
	assert.Equal(t, 37.7749, location.Latitude)
	assert.Equal(t, -122.4194, location.Longitude)
	mockRedis.AssertExpectations(t)
}

func TestService_GetDriverLocation_NotFound(t *testing.T) {
	// Arrange
	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	ctx := context.Background()
	driverID := uuid.New()

	mockRedis.On("GetString", ctx, "driver:location:"+driverID.String()).
		Return("", errors.New("key not found"))

	// Act
	location, err := service.GetDriverLocation(ctx, driverID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, location)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 404, appErr.Code)
	mockRedis.AssertExpectations(t)
}

func TestService_GetDriverLocation_InvalidJSON(t *testing.T) {
	// Arrange
	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	ctx := context.Background()
	driverID := uuid.New()

	mockRedis.On("GetString", ctx, "driver:location:"+driverID.String()).
		Return("invalid json", nil)

	// Act
	location, err := service.GetDriverLocation(ctx, driverID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, location)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 500, appErr.Code)
	mockRedis.AssertExpectations(t)
}

func TestService_FindNearbyDrivers_Success(t *testing.T) {
	// Arrange
	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	ctx := context.Background()
	pickupLat := 37.7749
	pickupLon := -122.4194
	maxDrivers := 5

	driver1ID := uuid.New()
	driver2ID := uuid.New()

	// Mock GeoRadius to return driver IDs (service passes maxDrivers*2 for over-fetching)
	mockRedis.On("GeoRadius", ctx, driverGeoIndexKey, pickupLon, pickupLat, searchRadiusKm, maxDrivers*2).
		Return([]string{driver1ID.String(), driver2ID.String()}, nil)

	// Mock GetDriverLocation for each driver
	location1 := &DriverLocation{
		DriverID:  driver1ID,
		Latitude:  37.7750,
		Longitude: -122.4195,
		Timestamp: time.Now(),
	}
	location2 := &DriverLocation{
		DriverID:  driver2ID,
		Latitude:  37.7751,
		Longitude: -122.4196,
		Timestamp: time.Now(),
	}

	location1JSON, _ := json.Marshal(location1)
	location2JSON, _ := json.Marshal(location2)

	mockRedis.On("GetString", ctx, "driver:location:"+driver1ID.String()).
		Return(string(location1JSON), nil)
	mockRedis.On("GetString", ctx, "driver:location:"+driver2ID.String()).
		Return(string(location2JSON), nil)

	// Act
	locations, err := service.FindNearbyDrivers(ctx, pickupLat, pickupLon, maxDrivers)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, locations, 2)
	mockRedis.AssertExpectations(t)
}

func TestService_FindNearbyDrivers_NoDriversFound(t *testing.T) {
	// Arrange
	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	ctx := context.Background()
	pickupLat := 37.7749
	pickupLon := -122.4194
	maxDrivers := 5

	// Mock GeoRadius to return empty list (service passes maxDrivers*2)
	mockRedis.On("GeoRadius", ctx, driverGeoIndexKey, pickupLon, pickupLat, searchRadiusKm, maxDrivers*2).
		Return([]string{}, nil)

	// Act
	locations, err := service.FindNearbyDrivers(ctx, pickupLat, pickupLon, maxDrivers)

	// Assert
	assert.NoError(t, err)
	assert.Empty(t, locations)
	mockRedis.AssertExpectations(t)
}

func TestService_FindNearbyDrivers_GeoRadiusError(t *testing.T) {
	// Arrange
	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	ctx := context.Background()
	pickupLat := 37.7749
	pickupLon := -122.4194
	maxDrivers := 5

	mockRedis.On("GeoRadius", ctx, driverGeoIndexKey, pickupLon, pickupLat, searchRadiusKm, maxDrivers*2).
		Return(nil, errors.New("redis error"))

	// Act
	locations, err := service.FindNearbyDrivers(ctx, pickupLat, pickupLon, maxDrivers)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, locations)
	mockRedis.AssertExpectations(t)
}

func TestService_SetDriverStatus_Available(t *testing.T) {
	// Arrange
	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	ctx := context.Background()
	driverID := uuid.New()
	status := "available"

	mockRedis.On("SetWithExpiration", ctx, "driver:status:"+driverID.String(),
		mock.AnythingOfType("[]uint8"), driverLocationTTL).Return(nil)

	// Act
	err := service.SetDriverStatus(ctx, driverID, status)

	// Assert
	assert.NoError(t, err)
	mockRedis.AssertExpectations(t)
}

func TestService_SetDriverStatus_Offline(t *testing.T) {
	// Arrange
	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	ctx := context.Background()
	driverID := uuid.New()
	status := "offline"

	mockRedis.On("SetWithExpiration", ctx, "driver:status:"+driverID.String(),
		mock.AnythingOfType("[]uint8"), driverLocationTTL).Return(nil)
	mockRedis.On("GeoRemove", ctx, driverGeoIndexKey, driverID.String()).Return(nil)
	mockRedis.On("Delete", ctx, []string{"driver:h3cell:" + driverID.String()}).Return(nil)

	// Act
	err := service.SetDriverStatus(ctx, driverID, status)

	// Assert
	assert.NoError(t, err)
	mockRedis.AssertExpectations(t)
}

func TestService_SetDriverStatus_Error(t *testing.T) {
	// Arrange
	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	ctx := context.Background()
	driverID := uuid.New()
	status := "available"

	mockRedis.On("SetWithExpiration", ctx, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("redis error"))

	// Act
	err := service.SetDriverStatus(ctx, driverID, status)

	// Assert
	assert.Error(t, err)
	mockRedis.AssertExpectations(t)
}

func TestService_GetDriverStatus_Success(t *testing.T) {
	// Arrange
	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	ctx := context.Background()
	driverID := uuid.New()

	statusData := map[string]interface{}{
		"status":    "available",
		"timestamp": time.Now(),
	}
	statusJSON, _ := json.Marshal(statusData)

	mockRedis.On("GetString", ctx, "driver:status:"+driverID.String()).
		Return(string(statusJSON), nil)

	// Act
	status, err := service.GetDriverStatus(ctx, driverID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "available", status)
	mockRedis.AssertExpectations(t)
}

func TestService_GetDriverStatus_NotFound(t *testing.T) {
	// Arrange
	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	ctx := context.Background()
	driverID := uuid.New()

	mockRedis.On("GetString", ctx, "driver:status:"+driverID.String()).
		Return("", errors.New("key not found"))

	// Act
	status, err := service.GetDriverStatus(ctx, driverID)

	// Assert
	assert.NoError(t, err) // Should not return error
	assert.Equal(t, "offline", status) // Defaults to offline
	mockRedis.AssertExpectations(t)
}

func TestService_CalculateDistance(t *testing.T) {
	tests := []struct {
		name      string
		lat1      float64
		lon1      float64
		lat2      float64
		lon2      float64
		expected  float64
		tolerance float64
	}{
		{
			name:      "San Francisco to Oakland",
			lat1:      37.7749,
			lon1:      -122.4194,
			lat2:      37.8044,
			lon2:      -122.2712,
			expected:  13.0, // approximately 13 km
			tolerance: 1.0,
		},
		{
			name:      "Same location",
			lat1:      37.7749,
			lon1:      -122.4194,
			lat2:      37.7749,
			lon2:      -122.4194,
			expected:  0.0,
			tolerance: 0.01,
		},
		{
			name:      "Short distance",
			lat1:      37.7749,
			lon1:      -122.4194,
			lat2:      37.7750,
			lon2:      -122.4195,
			expected:  0.01,
			tolerance: 0.05,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(nil)
			distance := service.CalculateDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			assert.InDelta(t, tt.expected, distance, tt.tolerance)
		})
	}
}

func TestService_CalculateETA(t *testing.T) {
	tests := []struct {
		name     string
		distance float64
		expected int
	}{
		{
			name:     "10 km distance",
			distance: 10.0,
			expected: 15, // (10 / 40) * 60 = 15 minutes
		},
		{
			name:     "20 km distance",
			distance: 20.0,
			expected: 30,
		},
		{
			name:     "5 km distance",
			distance: 5.0,
			expected: 8, // (5 / 40) * 60 = 7.5, rounded to 8
		},
		{
			name:     "zero distance",
			distance: 0.0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(nil)
			eta := service.CalculateETA(tt.distance)
			assert.Equal(t, tt.expected, eta)
		})
	}
}

func TestService_FindAvailableDrivers_Success(t *testing.T) {
	// Arrange
	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	ctx := context.Background()
	pickupLat := 37.7749
	pickupLon := -122.4194
	maxDrivers := 2

	driver1ID := uuid.New()
	driver2ID := uuid.New()
	driver3ID := uuid.New()

	// Mock GeoRadius to return 3 drivers
	// FindAvailableDrivers passes maxDrivers*2 to FindNearbyDrivers, which internally doubles again
	mockRedis.On("GeoRadius", ctx, driverGeoIndexKey, pickupLon, pickupLat, searchRadiusKm, maxDrivers*2*2).
		Return([]string{driver1ID.String(), driver2ID.String(), driver3ID.String()}, nil)

	// Mock GetDriverLocation for each driver
	location1 := &DriverLocation{DriverID: driver1ID, Latitude: 37.7750, Longitude: -122.4195, Timestamp: time.Now()}
	location2 := &DriverLocation{DriverID: driver2ID, Latitude: 37.7751, Longitude: -122.4196, Timestamp: time.Now()}
	location3 := &DriverLocation{DriverID: driver3ID, Latitude: 37.7752, Longitude: -122.4197, Timestamp: time.Now()}

	location1JSON, _ := json.Marshal(location1)
	location2JSON, _ := json.Marshal(location2)
	location3JSON, _ := json.Marshal(location3)

	mockRedis.On("GetString", ctx, "driver:location:"+driver1ID.String()).Return(string(location1JSON), nil)
	mockRedis.On("GetString", ctx, "driver:location:"+driver2ID.String()).Return(string(location2JSON), nil)
	mockRedis.On("GetString", ctx, "driver:location:"+driver3ID.String()).Return(string(location3JSON), nil)

	// Mock GetDriverStatus - 2 available, 1 busy
	status1 := map[string]interface{}{"status": "available", "timestamp": time.Now()}
	status2 := map[string]interface{}{"status": "available", "timestamp": time.Now()}

	status1JSON, _ := json.Marshal(status1)
	status2JSON, _ := json.Marshal(status2)

	mockRedis.On("GetString", ctx, "driver:status:"+driver1ID.String()).Return(string(status1JSON), nil)
	mockRedis.On("GetString", ctx, "driver:status:"+driver2ID.String()).Return(string(status2JSON), nil)
	// Driver 3 won't be checked since we break after finding maxDrivers (2) available drivers

	// Act
	locations, err := service.FindAvailableDrivers(ctx, pickupLat, pickupLon, maxDrivers)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, locations, 2) // Only 2 available drivers
	mockRedis.AssertExpectations(t)
}

func TestService_FindAvailableDrivers_NoneAvailable(t *testing.T) {
	// Arrange
	mockRedis := new(mocks.MockRedisClient)
	service := NewService(mockRedis)
	ctx := context.Background()
	pickupLat := 37.7749
	pickupLon := -122.4194
	maxDrivers := 2

	driver1ID := uuid.New()

	// FindAvailableDrivers passes maxDrivers*2 to FindNearbyDrivers, which internally doubles again
	mockRedis.On("GeoRadius", ctx, driverGeoIndexKey, pickupLon, pickupLat, searchRadiusKm, maxDrivers*2*2).
		Return([]string{driver1ID.String()}, nil)

	location1 := &DriverLocation{DriverID: driver1ID, Latitude: 37.7750, Longitude: -122.4195, Timestamp: time.Now()}
	location1JSON, _ := json.Marshal(location1)
	mockRedis.On("GetString", ctx, "driver:location:"+driver1ID.String()).Return(string(location1JSON), nil)

	// Driver is busy
	status1 := map[string]interface{}{"status": "busy", "timestamp": time.Now()}
	status1JSON, _ := json.Marshal(status1)
	mockRedis.On("GetString", ctx, "driver:status:"+driver1ID.String()).Return(string(status1JSON), nil)

	// Act
	locations, err := service.FindAvailableDrivers(ctx, pickupLat, pickupLon, maxDrivers)

	// Assert
	assert.NoError(t, err)
	assert.Empty(t, locations)
	mockRedis.AssertExpectations(t)
}
