package ridetypes

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// Mock Repository
// ============================================================================

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateRideType(ctx context.Context, rt *RideType) error {
	args := m.Called(ctx, rt)
	return args.Error(0)
}

func (m *MockRepository) GetRideTypeByID(ctx context.Context, id uuid.UUID) (*RideType, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RideType), args.Error(1)
}

func (m *MockRepository) ListRideTypes(ctx context.Context, limit, offset int, includeInactive bool) ([]*RideType, int64, error) {
	args := m.Called(ctx, limit, offset, includeInactive)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*RideType), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) UpdateRideType(ctx context.Context, rt *RideType) error {
	args := m.Called(ctx, rt)
	return args.Error(0)
}

func (m *MockRepository) DeleteRideType(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) AddRideTypeToCountry(ctx context.Context, crt *CountryRideType) error {
	args := m.Called(ctx, crt)
	return args.Error(0)
}

func (m *MockRepository) UpdateCountryRideType(ctx context.Context, crt *CountryRideType) error {
	args := m.Called(ctx, crt)
	return args.Error(0)
}

func (m *MockRepository) RemoveRideTypeFromCountry(ctx context.Context, countryID, rideTypeID uuid.UUID) error {
	args := m.Called(ctx, countryID, rideTypeID)
	return args.Error(0)
}

func (m *MockRepository) ListCountryRideTypes(ctx context.Context, countryID uuid.UUID, includeInactive bool) ([]*CountryRideTypeWithDetails, error) {
	args := m.Called(ctx, countryID, includeInactive)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*CountryRideTypeWithDetails), args.Error(1)
}

func (m *MockRepository) AddRideTypeToCity(ctx context.Context, crt *CityRideType) error {
	args := m.Called(ctx, crt)
	return args.Error(0)
}

func (m *MockRepository) UpdateCityRideType(ctx context.Context, crt *CityRideType) error {
	args := m.Called(ctx, crt)
	return args.Error(0)
}

func (m *MockRepository) RemoveRideTypeFromCity(ctx context.Context, cityID, rideTypeID uuid.UUID) error {
	args := m.Called(ctx, cityID, rideTypeID)
	return args.Error(0)
}

func (m *MockRepository) ListCityRideTypes(ctx context.Context, cityID uuid.UUID, includeInactive bool) ([]*CityRideTypeWithDetails, error) {
	args := m.Called(ctx, cityID, includeInactive)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*CityRideTypeWithDetails), args.Error(1)
}

func (m *MockRepository) GetAvailableRideTypes(ctx context.Context, countryID, cityID *uuid.UUID) ([]*RideType, error) {
	args := m.Called(ctx, countryID, cityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RideType), args.Error(1)
}

// ============================================================================
// Helpers
// ============================================================================

func createTestRideType() *RideType {
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

func createTestCountryRideType() *CountryRideType {
	return &CountryRideType{
		CountryID:  uuid.New(),
		RideTypeID: uuid.New(),
		IsActive:   true,
		SortOrder:  0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func createTestCityRideType() *CityRideType {
	return &CityRideType{
		CityID:     uuid.New(),
		RideTypeID: uuid.New(),
		IsActive:   true,
		SortOrder:  0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// ============================================================================
// Ride Type CRUD Tests
// ============================================================================

func TestCreateRideType(t *testing.T) {
	tests := []struct {
		name        string
		mockError   error
		expectError bool
	}{
		{
			name:        "success",
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "repository error",
			mockError:   errors.New("db error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo, nil)
			ctx := context.Background()
			rt := createTestRideType()

			mockRepo.On("CreateRideType", ctx, rt).Return(tt.mockError)

			err := service.CreateRideType(ctx, rt)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetRideTypeByID(t *testing.T) {
	tests := []struct {
		name        string
		mockReturn  *RideType
		mockError   error
		expectError bool
	}{
		{
			name:        "success",
			mockReturn:  createTestRideType(),
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "not found",
			mockReturn:  nil,
			mockError:   errors.New("not found"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo, nil)
			ctx := context.Background()
			id := uuid.New()

			mockRepo.On("GetRideTypeByID", ctx, id).Return(tt.mockReturn, tt.mockError)

			rt, err := service.GetRideTypeByID(ctx, id)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, rt)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, rt)
				assert.Equal(t, tt.mockReturn.Name, rt.Name)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestListRideTypes(t *testing.T) {
	tests := []struct {
		name            string
		limit           int
		offset          int
		includeInactive bool
		mockReturn      []*RideType
		mockTotal       int64
		mockError       error
		expectedCount   int
		expectError     bool
	}{
		{
			name:            "returns active ride types",
			limit:           10,
			offset:          0,
			includeInactive: false,
			mockReturn: []*RideType{
				createTestRideType(),
				createTestRideType(),
			},
			mockTotal:     2,
			mockError:     nil,
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:            "empty list",
			limit:           10,
			offset:          0,
			includeInactive: false,
			mockReturn:      []*RideType{},
			mockTotal:       0,
			mockError:       nil,
			expectedCount:   0,
			expectError:     false,
		},
		{
			name:            "repository error",
			limit:           10,
			offset:          0,
			includeInactive: false,
			mockReturn:      nil,
			mockTotal:       0,
			mockError:       errors.New("db error"),
			expectedCount:   0,
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo, nil)
			ctx := context.Background()

			mockRepo.On("ListRideTypes", ctx, tt.limit, tt.offset, tt.includeInactive).
				Return(tt.mockReturn, tt.mockTotal, tt.mockError)

			items, total, err := service.ListRideTypes(ctx, tt.limit, tt.offset, tt.includeInactive)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, items)
			} else {
				assert.NoError(t, err)
				assert.Len(t, items, tt.expectedCount)
				assert.Equal(t, tt.mockTotal, total)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUpdateRideType(t *testing.T) {
	tests := []struct {
		name        string
		mockError   error
		expectError bool
	}{
		{
			name:        "success",
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "repository error",
			mockError:   errors.New("db error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo, nil)
			ctx := context.Background()
			rt := createTestRideType()

			mockRepo.On("UpdateRideType", ctx, rt).Return(tt.mockError)

			err := service.UpdateRideType(ctx, rt)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestDeleteRideType(t *testing.T) {
	tests := []struct {
		name        string
		mockError   error
		expectError bool
	}{
		{
			name:        "success",
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "repository error",
			mockError:   errors.New("db error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo, nil)
			ctx := context.Background()
			id := uuid.New()

			mockRepo.On("DeleteRideType", ctx, id).Return(tt.mockError)

			err := service.DeleteRideType(ctx, id)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Country Ride Type Tests
// ============================================================================

func TestAddRideTypeToCountry(t *testing.T) {
	tests := []struct {
		name        string
		mockError   error
		expectError bool
	}{
		{
			name:        "success",
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "repository error",
			mockError:   errors.New("db error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo, nil)
			ctx := context.Background()
			crt := createTestCountryRideType()

			mockRepo.On("AddRideTypeToCountry", ctx, crt).Return(tt.mockError)

			err := service.AddRideTypeToCountry(ctx, crt)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestListCountryRideTypes(t *testing.T) {
	countryID := uuid.New()
	rideTypeID := uuid.New()

	tests := []struct {
		name            string
		includeInactive bool
		mockReturn      []*CountryRideTypeWithDetails
		mockError       error
		expectedCount   int
		expectError     bool
	}{
		{
			name:            "returns country ride types",
			includeInactive: false,
			mockReturn: []*CountryRideTypeWithDetails{
				{
					CountryRideType: CountryRideType{
						CountryID:  countryID,
						RideTypeID: rideTypeID,
						IsActive:   true,
					},
					RideTypeName:     "Economy",
					RideTypeCapacity: 4,
				},
			},
			mockError:     nil,
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:            "repository error",
			includeInactive: false,
			mockReturn:      nil,
			mockError:       errors.New("db error"),
			expectedCount:   0,
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo, nil)
			ctx := context.Background()

			mockRepo.On("ListCountryRideTypes", ctx, countryID, tt.includeInactive).
				Return(tt.mockReturn, tt.mockError)

			items, err := service.ListCountryRideTypes(ctx, countryID, tt.includeInactive)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, items)
			} else {
				assert.NoError(t, err)
				assert.Len(t, items, tt.expectedCount)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestRemoveRideTypeFromCountry(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, nil)
	ctx := context.Background()
	countryID := uuid.New()
	rideTypeID := uuid.New()

	mockRepo.On("RemoveRideTypeFromCountry", ctx, countryID, rideTypeID).Return(nil)

	err := service.RemoveRideTypeFromCountry(ctx, countryID, rideTypeID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// City Ride Type Tests
// ============================================================================

func TestAddRideTypeToCity(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, nil)
	ctx := context.Background()
	crt := createTestCityRideType()

	mockRepo.On("AddRideTypeToCity", ctx, crt).Return(nil)

	err := service.AddRideTypeToCity(ctx, crt)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestListCityRideTypes(t *testing.T) {
	cityID := uuid.New()
	rideTypeID := uuid.New()

	tests := []struct {
		name            string
		includeInactive bool
		mockReturn      []*CityRideTypeWithDetails
		mockError       error
		expectedCount   int
		expectError     bool
	}{
		{
			name:            "returns city ride types",
			includeInactive: false,
			mockReturn: []*CityRideTypeWithDetails{
				{
					CityRideType: CityRideType{
						CityID:     cityID,
						RideTypeID: rideTypeID,
						IsActive:   true,
					},
					RideTypeName:     "Premium",
					RideTypeCapacity: 4,
				},
			},
			mockError:     nil,
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:            "repository error",
			includeInactive: false,
			mockReturn:      nil,
			mockError:       errors.New("db error"),
			expectedCount:   0,
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo, nil)
			ctx := context.Background()

			mockRepo.On("ListCityRideTypes", ctx, cityID, tt.includeInactive).
				Return(tt.mockReturn, tt.mockError)

			items, err := service.ListCityRideTypes(ctx, cityID, tt.includeInactive)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, items)
			} else {
				assert.NoError(t, err)
				assert.Len(t, items, tt.expectedCount)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestRemoveRideTypeFromCity(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, nil)
	ctx := context.Background()
	cityID := uuid.New()
	rideTypeID := uuid.New()

	mockRepo.On("RemoveRideTypeFromCity", ctx, cityID, rideTypeID).Return(nil)

	err := service.RemoveRideTypeFromCity(ctx, cityID, rideTypeID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetAvailableRideTypes Tests
// ============================================================================

func TestGetAvailableRideTypes_NoGeoService(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, nil) // nil geoSvc

	ctx := context.Background()
	rideTypes := []*RideType{createTestRideType(), createTestRideType()}

	// With nil geoSvc, countryID and cityID are nil — global fallback
	mockRepo.On("GetAvailableRideTypes", ctx, (*uuid.UUID)(nil), (*uuid.UUID)(nil)).
		Return(rideTypes, nil)

	result, err := service.GetAvailableRideTypes(ctx, 41.2995, 69.2401)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	mockRepo.AssertExpectations(t)
}

func TestGetAvailableRideTypes_RepoError(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, nil)
	ctx := context.Background()

	mockRepo.On("GetAvailableRideTypes", ctx, (*uuid.UUID)(nil), (*uuid.UUID)(nil)).
		Return(nil, errors.New("db error"))

	result, err := service.GetAvailableRideTypes(ctx, 41.2995, 69.2401)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}
