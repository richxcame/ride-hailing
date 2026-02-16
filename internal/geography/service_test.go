package geography

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRepository is an in-package mock for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetActiveCountries(ctx context.Context) ([]*Country, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Country), args.Error(1)
}

func (m *MockRepository) GetCountryByCode(ctx context.Context, code string) (*Country, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Country), args.Error(1)
}

func (m *MockRepository) GetCountryByID(ctx context.Context, id uuid.UUID) (*Country, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Country), args.Error(1)
}

func (m *MockRepository) GetRegionsByCountry(ctx context.Context, countryID uuid.UUID) ([]*Region, error) {
	args := m.Called(ctx, countryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Region), args.Error(1)
}

func (m *MockRepository) GetRegionByID(ctx context.Context, id uuid.UUID) (*Region, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Region), args.Error(1)
}

func (m *MockRepository) GetCitiesByRegion(ctx context.Context, regionID uuid.UUID) ([]*City, error) {
	args := m.Called(ctx, regionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*City), args.Error(1)
}

func (m *MockRepository) GetCityByID(ctx context.Context, id uuid.UUID) (*City, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*City), args.Error(1)
}

func (m *MockRepository) ResolveLocation(ctx context.Context, latitude, longitude float64) (*ResolvedLocation, error) {
	args := m.Called(ctx, latitude, longitude)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ResolvedLocation), args.Error(1)
}

func (m *MockRepository) FindNearestCity(ctx context.Context, latitude, longitude float64, maxDistanceKm float64) (*City, error) {
	args := m.Called(ctx, latitude, longitude, maxDistanceKm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*City), args.Error(1)
}

func (m *MockRepository) GetPricingZonesByCity(ctx context.Context, cityID uuid.UUID) ([]*PricingZone, error) {
	args := m.Called(ctx, cityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*PricingZone), args.Error(1)
}

func (m *MockRepository) GetDriverRegions(ctx context.Context, driverID uuid.UUID) ([]*DriverRegion, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DriverRegion), args.Error(1)
}

func (m *MockRepository) AddDriverRegion(ctx context.Context, driverID, regionID uuid.UUID, isPrimary bool) error {
	args := m.Called(ctx, driverID, regionID, isPrimary)
	return args.Error(0)
}

func (m *MockRepository) CreateCountry(ctx context.Context, country *Country) error {
	args := m.Called(ctx, country)
	return args.Error(0)
}

func (m *MockRepository) CreateRegion(ctx context.Context, region *Region) error {
	args := m.Called(ctx, region)
	return args.Error(0)
}

func (m *MockRepository) CreateCity(ctx context.Context, city *City) error {
	args := m.Called(ctx, city)
	return args.Error(0)
}

func (m *MockRepository) CreatePricingZone(ctx context.Context, zone *PricingZone) error {
	args := m.Called(ctx, zone)
	return args.Error(0)
}

func (m *MockRepository) GetPricingZoneByID(ctx context.Context, id uuid.UUID) (*PricingZone, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PricingZone), args.Error(1)
}

func (m *MockRepository) GetGeographyStats(ctx context.Context) (*GeographyStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GeographyStats), args.Error(1)
}

func (m *MockRepository) GetAllCountries(ctx context.Context, limit, offset int, search string) ([]*Country, int64, error) {
	args := m.Called(ctx, limit, offset, search)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*Country), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) GetAllRegions(ctx context.Context, countryID *uuid.UUID, limit, offset int, search string) ([]*Region, int64, error) {
	args := m.Called(ctx, countryID, limit, offset, search)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*Region), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) GetAllCities(ctx context.Context, regionID *uuid.UUID, limit, offset int, search string) ([]*City, int64, error) {
	args := m.Called(ctx, regionID, limit, offset, search)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*City), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) GetAllPricingZones(ctx context.Context, cityID *uuid.UUID, limit, offset int, search string) ([]*PricingZone, int64, error) {
	args := m.Called(ctx, cityID, limit, offset, search)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*PricingZone), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) UpdateCountry(ctx context.Context, country *Country) error {
	args := m.Called(ctx, country)
	return args.Error(0)
}

func (m *MockRepository) UpdateRegion(ctx context.Context, region *Region) error {
	args := m.Called(ctx, region)
	return args.Error(0)
}

func (m *MockRepository) UpdateCity(ctx context.Context, city *City) error {
	args := m.Called(ctx, city)
	return args.Error(0)
}

func (m *MockRepository) UpdatePricingZone(ctx context.Context, zone *PricingZone) error {
	args := m.Called(ctx, zone)
	return args.Error(0)
}

func (m *MockRepository) DeleteCountry(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) DeleteRegion(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) DeleteCity(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) DeletePricingZone(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// =============================================================================
// Test Helper Functions
// =============================================================================

func createTestCountry() *Country {
	return &Country{
		ID:           uuid.New(),
		Code:         "TM",
		Code3:        "TKM",
		Name:         "Turkmenistan",
		CurrencyCode: "TMT",
		Timezone:     "Asia/Ashgabat",
		PhonePrefix:  "+993",
		IsActive:     true,
	}
}

func createTestRegion(countryID uuid.UUID) *Region {
	tz := "Asia/Ashgabat"
	return &Region{
		ID:        uuid.New(),
		CountryID: countryID,
		Code:      "ASH",
		Name:      "Ashgabat",
		Timezone:  &tz,
		IsActive:  true,
	}
}

func createTestCity(regionID uuid.UUID) *City {
	tz := "Asia/Ashgabat"
	pop := 1000000
	return &City{
		ID:              uuid.New(),
		RegionID:        regionID,
		Name:            "Ashgabat",
		Timezone:        &tz,
		CenterLatitude:  37.9601,
		CenterLongitude: 58.3261,
		Population:      &pop,
		IsActive:        true,
	}
}

func createTestPricingZone(cityID uuid.UUID) *PricingZone {
	return &PricingZone{
		ID:       uuid.New(),
		CityID:   cityID,
		Name:     "Airport Zone",
		ZoneType: ZoneTypeAirport,
		Priority: 10,
		IsActive: true,
	}
}

// =============================================================================
// Test NewService
// =============================================================================

func TestNewService(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
}

// =============================================================================
// Test GetActiveCountries
// =============================================================================

func TestGetActiveCountries(t *testing.T) {
	tests := []struct {
		name          string
		mockReturn    []*Country
		mockError     error
		expectedCount int
		expectError   bool
	}{
		{
			name: "returns active countries",
			mockReturn: []*Country{
				createTestCountry(),
				createTestCountry(),
			},
			mockError:     nil,
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:          "returns empty list when no countries",
			mockReturn:    []*Country{},
			mockError:     nil,
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "returns error on repository failure",
			mockReturn:    nil,
			mockError:     errors.New("database error"),
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo)
			ctx := context.Background()

			mockRepo.On("GetActiveCountries", ctx).Return(tt.mockReturn, tt.mockError)

			countries, err := service.GetActiveCountries(ctx)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, countries)
			} else {
				assert.NoError(t, err)
				assert.Len(t, countries, tt.expectedCount)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// Test GetCountryByCode
// =============================================================================

func TestGetCountryByCode(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		mockReturn  *Country
		mockError   error
		expectError bool
	}{
		{
			name:        "returns country by code",
			code:        "TM",
			mockReturn:  createTestCountry(),
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "returns error for non-existent country",
			code:        "XX",
			mockReturn:  nil,
			mockError:   errors.New("country not found"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo)
			ctx := context.Background()

			mockRepo.On("GetCountryByCode", ctx, tt.code).Return(tt.mockReturn, tt.mockError)

			country, err := service.GetCountryByCode(ctx, tt.code)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, country)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, country)
				assert.Equal(t, tt.code, country.Code)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// Test GetCountryByID
// =============================================================================

func TestGetCountryByID(t *testing.T) {
	countryID := uuid.New()

	tests := []struct {
		name        string
		id          uuid.UUID
		mockReturn  *Country
		mockError   error
		expectError bool
	}{
		{
			name: "returns country by ID",
			id:   countryID,
			mockReturn: func() *Country {
				c := createTestCountry()
				c.ID = countryID
				return c
			}(),
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "returns error for non-existent country",
			id:          uuid.New(),
			mockReturn:  nil,
			mockError:   errors.New("country not found"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo)
			ctx := context.Background()

			mockRepo.On("GetCountryByID", ctx, tt.id).Return(tt.mockReturn, tt.mockError)

			country, err := service.GetCountryByID(ctx, tt.id)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, country)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, country)
				assert.Equal(t, tt.id, country.ID)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// Test GetRegionsByCountryCode
// =============================================================================

func TestGetRegionsByCountryCode(t *testing.T) {
	country := createTestCountry()

	tests := []struct {
		name           string
		countryCode    string
		mockCountry    *Country
		mockCountryErr error
		mockRegions    []*Region
		mockRegionsErr error
		expectedCount  int
		expectError    bool
		errorContains  string
	}{
		{
			name:        "returns regions for valid country code",
			countryCode: "TM",
			mockCountry: country,
			mockRegions: []*Region{
				createTestRegion(country.ID),
				createTestRegion(country.ID),
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:           "returns error for non-existent country",
			countryCode:    "XX",
			mockCountry:    nil,
			mockCountryErr: errors.New("not found"),
			expectError:    true,
			errorContains:  "country not found",
		},
		{
			name:           "returns error when regions query fails",
			countryCode:    "TM",
			mockCountry:    country,
			mockRegions:    nil,
			mockRegionsErr: errors.New("database error"),
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo)
			ctx := context.Background()

			mockRepo.On("GetCountryByCode", ctx, tt.countryCode).Return(tt.mockCountry, tt.mockCountryErr)
			if tt.mockCountry != nil {
				mockRepo.On("GetRegionsByCountry", ctx, tt.mockCountry.ID).Return(tt.mockRegions, tt.mockRegionsErr)
			}

			regions, err := service.GetRegionsByCountryCode(ctx, tt.countryCode)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, regions, tt.expectedCount)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// Test GetRegionsByCountry
// =============================================================================

func TestGetRegionsByCountry(t *testing.T) {
	countryID := uuid.New()

	tests := []struct {
		name          string
		countryID     uuid.UUID
		mockReturn    []*Region
		mockError     error
		expectedCount int
		expectError   bool
	}{
		{
			name:      "returns regions for country",
			countryID: countryID,
			mockReturn: []*Region{
				createTestRegion(countryID),
			},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:          "returns empty list when no regions",
			countryID:     countryID,
			mockReturn:    []*Region{},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:        "returns error on repository failure",
			countryID:   countryID,
			mockReturn:  nil,
			mockError:   errors.New("database error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo)
			ctx := context.Background()

			mockRepo.On("GetRegionsByCountry", ctx, tt.countryID).Return(tt.mockReturn, tt.mockError)

			regions, err := service.GetRegionsByCountry(ctx, tt.countryID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, regions, tt.expectedCount)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// Test ResolveLocation - Core Logic
// =============================================================================

func TestResolveLocation_WithCity(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	latitude, longitude := 37.9601, 58.3261

	country := createTestCountry()
	region := createTestRegion(country.ID)
	region.Country = country
	city := createTestCity(region.ID)
	city.Region = region

	resolved := &ResolvedLocation{
		Location: Location{Latitude: latitude, Longitude: longitude},
		Country:  country,
		Region:   region,
		City:     city,
		Timezone: "Asia/Ashgabat",
	}

	mockRepo.On("ResolveLocation", ctx, latitude, longitude).Return(resolved, nil)

	result, err := service.ResolveLocation(ctx, latitude, longitude)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.City)
	assert.Equal(t, city.Name, result.City.Name)
	assert.Equal(t, "Asia/Ashgabat", result.Timezone)
	mockRepo.AssertExpectations(t)
}

func TestResolveLocation_WithNearestCityFallback(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	latitude, longitude := 37.9601, 58.3261

	// ResolveLocation returns no city (outside boundary)
	resolved := &ResolvedLocation{
		Location: Location{Latitude: latitude, Longitude: longitude},
		City:     nil,
	}

	// Setup the full city hierarchy for nearest city
	country := createTestCountry()
	region := createTestRegion(country.ID)
	region.Country = country
	city := createTestCity(region.ID)
	city.Region = region

	mockRepo.On("ResolveLocation", ctx, latitude, longitude).Return(resolved, nil)
	mockRepo.On("FindNearestCity", ctx, latitude, longitude, 50.0).Return(city, nil)
	mockRepo.On("GetCityByID", ctx, city.ID).Return(city, nil)

	result, err := service.ResolveLocation(ctx, latitude, longitude)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.City)
	assert.Equal(t, city.Name, result.City.Name)
	assert.Equal(t, country.CurrencyCode, result.Country.CurrencyCode)
	mockRepo.AssertExpectations(t)
}

func TestResolveLocation_NoNearestCity(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	latitude, longitude := 0.0, 0.0 // Middle of the ocean

	// ResolveLocation returns no city
	resolved := &ResolvedLocation{
		Location: Location{Latitude: latitude, Longitude: longitude},
		City:     nil,
	}

	mockRepo.On("ResolveLocation", ctx, latitude, longitude).Return(resolved, nil)
	mockRepo.On("FindNearestCity", ctx, latitude, longitude, 50.0).Return(nil, errors.New("no city found"))

	result, err := service.ResolveLocation(ctx, latitude, longitude)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, result.City)
	mockRepo.AssertExpectations(t)
}

func TestResolveLocation_TimezoneFromCity(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	latitude, longitude := 37.9601, 58.3261
	cityTz := "Asia/Ashgabat"

	country := createTestCountry()
	country.Timezone = "UTC"

	region := createTestRegion(country.ID)
	region.Timezone = nil // No region timezone
	region.Country = country

	city := createTestCity(region.ID)
	city.Timezone = &cityTz // City has specific timezone
	city.Region = region

	// No city found in boundary
	resolved := &ResolvedLocation{
		Location: Location{Latitude: latitude, Longitude: longitude},
		City:     nil,
	}

	mockRepo.On("ResolveLocation", ctx, latitude, longitude).Return(resolved, nil)
	mockRepo.On("FindNearestCity", ctx, latitude, longitude, 50.0).Return(city, nil)
	mockRepo.On("GetCityByID", ctx, city.ID).Return(city, nil)

	result, err := service.ResolveLocation(ctx, latitude, longitude)

	require.NoError(t, err)
	assert.Equal(t, "Asia/Ashgabat", result.Timezone)
	mockRepo.AssertExpectations(t)
}

func TestResolveLocation_TimezoneFromRegion(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	latitude, longitude := 37.9601, 58.3261
	regionTz := "Asia/Ashgabat"

	country := createTestCountry()
	country.Timezone = "UTC"

	region := createTestRegion(country.ID)
	region.Timezone = &regionTz
	region.Country = country

	city := createTestCity(region.ID)
	city.Timezone = nil // No city timezone - falls back to region
	city.Region = region

	// No city found in boundary
	resolved := &ResolvedLocation{
		Location: Location{Latitude: latitude, Longitude: longitude},
		City:     nil,
	}

	mockRepo.On("ResolveLocation", ctx, latitude, longitude).Return(resolved, nil)
	mockRepo.On("FindNearestCity", ctx, latitude, longitude, 50.0).Return(city, nil)
	mockRepo.On("GetCityByID", ctx, city.ID).Return(city, nil)

	result, err := service.ResolveLocation(ctx, latitude, longitude)

	require.NoError(t, err)
	assert.Equal(t, "Asia/Ashgabat", result.Timezone)
	mockRepo.AssertExpectations(t)
}

func TestResolveLocation_TimezoneFromCountry(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	latitude, longitude := 37.9601, 58.3261

	country := createTestCountry()
	country.Timezone = "Asia/Ashgabat"

	region := createTestRegion(country.ID)
	region.Timezone = nil // No region timezone
	region.Country = country

	city := createTestCity(region.ID)
	city.Timezone = nil // No city timezone - falls back to country
	city.Region = region

	// No city found in boundary
	resolved := &ResolvedLocation{
		Location: Location{Latitude: latitude, Longitude: longitude},
		City:     nil,
	}

	mockRepo.On("ResolveLocation", ctx, latitude, longitude).Return(resolved, nil)
	mockRepo.On("FindNearestCity", ctx, latitude, longitude, 50.0).Return(city, nil)
	mockRepo.On("GetCityByID", ctx, city.ID).Return(city, nil)

	result, err := service.ResolveLocation(ctx, latitude, longitude)

	require.NoError(t, err)
	assert.Equal(t, "Asia/Ashgabat", result.Timezone)
	mockRepo.AssertExpectations(t)
}

func TestResolveLocation_Error(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	ctx := context.Background()

	latitude, longitude := 37.9601, 58.3261

	mockRepo.On("ResolveLocation", ctx, latitude, longitude).Return(nil, errors.New("database error"))

	result, err := service.ResolveLocation(ctx, latitude, longitude)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

// =============================================================================
// Test GetTimezone
// =============================================================================

func TestGetTimezone(t *testing.T) {
	t.Run("returns timezone from resolved location with city", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)
		ctx := context.Background()

		latitude, longitude := 37.9601, 58.3261
		city := createTestCity(uuid.New())

		resolved := &ResolvedLocation{
			Location: Location{Latitude: latitude, Longitude: longitude},
			City:     city,
			Timezone: "Asia/Ashgabat",
		}
		mockRepo.On("ResolveLocation", ctx, latitude, longitude).Return(resolved, nil)

		timezone, err := service.GetTimezone(ctx, latitude, longitude)

		assert.NoError(t, err)
		assert.Equal(t, "Asia/Ashgabat", timezone)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns UTC when no timezone resolved and no city found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)
		ctx := context.Background()

		latitude, longitude := 0.0, 0.0

		resolved := &ResolvedLocation{
			Location: Location{Latitude: latitude, Longitude: longitude},
			City:     nil,
			Timezone: "",
		}
		mockRepo.On("ResolveLocation", ctx, latitude, longitude).Return(resolved, nil)
		mockRepo.On("FindNearestCity", ctx, latitude, longitude, 50.0).Return(nil, errors.New("no city found"))

		timezone, err := service.GetTimezone(ctx, latitude, longitude)

		assert.NoError(t, err)
		assert.Equal(t, "UTC", timezone)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns error when location resolution fails", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)
		ctx := context.Background()

		latitude, longitude := 37.9601, 58.3261

		mockRepo.On("ResolveLocation", ctx, latitude, longitude).Return(nil, errors.New("resolution failed"))

		timezone, err := service.GetTimezone(ctx, latitude, longitude)

		assert.Error(t, err)
		assert.Empty(t, timezone)
		mockRepo.AssertExpectations(t)
	})
}

// =============================================================================
// Test IsLocationServiceable
// =============================================================================

func TestIsLocationServiceable(t *testing.T) {
	t.Run("returns true for active city in boundary", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)
		ctx := context.Background()

		latitude, longitude := 37.9601, 58.3261
		city := createTestCity(uuid.New())
		city.IsActive = true

		resolved := &ResolvedLocation{
			Location: Location{Latitude: latitude, Longitude: longitude},
			City:     city,
		}
		mockRepo.On("ResolveLocation", ctx, latitude, longitude).Return(resolved, nil)

		serviceable, result, err := service.IsLocationServiceable(ctx, latitude, longitude)

		assert.NoError(t, err)
		assert.True(t, serviceable)
		assert.NotNil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns false for inactive city", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)
		ctx := context.Background()

		latitude, longitude := 37.9601, 58.3261
		city := createTestCity(uuid.New())
		city.IsActive = false

		resolved := &ResolvedLocation{
			Location: Location{Latitude: latitude, Longitude: longitude},
			City:     city,
		}
		mockRepo.On("ResolveLocation", ctx, latitude, longitude).Return(resolved, nil)

		serviceable, result, err := service.IsLocationServiceable(ctx, latitude, longitude)

		assert.NoError(t, err)
		assert.False(t, serviceable)
		assert.NotNil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns false when no city found even with fallback", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)
		ctx := context.Background()

		latitude, longitude := 0.0, 0.0

		resolved := &ResolvedLocation{
			Location: Location{Latitude: latitude, Longitude: longitude},
			City:     nil,
		}
		mockRepo.On("ResolveLocation", ctx, latitude, longitude).Return(resolved, nil)
		mockRepo.On("FindNearestCity", ctx, latitude, longitude, 50.0).Return(nil, errors.New("no city found"))

		serviceable, result, err := service.IsLocationServiceable(ctx, latitude, longitude)

		assert.NoError(t, err)
		assert.False(t, serviceable)
		assert.NotNil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns true when nearest city fallback finds active city", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)
		ctx := context.Background()

		latitude, longitude := 37.9601, 58.3261

		country := createTestCountry()
		region := createTestRegion(country.ID)
		region.Country = country
		city := createTestCity(region.ID)
		city.Region = region
		city.IsActive = true

		// No city in boundary
		resolved := &ResolvedLocation{
			Location: Location{Latitude: latitude, Longitude: longitude},
			City:     nil,
		}
		mockRepo.On("ResolveLocation", ctx, latitude, longitude).Return(resolved, nil)
		mockRepo.On("FindNearestCity", ctx, latitude, longitude, 50.0).Return(city, nil)
		mockRepo.On("GetCityByID", ctx, city.ID).Return(city, nil)

		serviceable, result, err := service.IsLocationServiceable(ctx, latitude, longitude)

		assert.NoError(t, err)
		assert.True(t, serviceable)
		assert.NotNil(t, result)
		assert.NotNil(t, result.City)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns error when resolution fails", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)
		ctx := context.Background()

		latitude, longitude := 37.9601, 58.3261

		mockRepo.On("ResolveLocation", ctx, latitude, longitude).Return(nil, errors.New("resolution failed"))

		serviceable, result, err := service.IsLocationServiceable(ctx, latitude, longitude)

		assert.Error(t, err)
		assert.False(t, serviceable)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})
}

// =============================================================================
// Test GetCurrencyForLocation
// =============================================================================

func TestGetCurrencyForLocation(t *testing.T) {
	t.Run("returns currency from country when city in boundary", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)
		ctx := context.Background()

		latitude, longitude := 37.9601, 58.3261
		country := createTestCountry()
		country.CurrencyCode = "TMT"
		city := createTestCity(uuid.New())

		resolved := &ResolvedLocation{
			Location: Location{Latitude: latitude, Longitude: longitude},
			Country:  country,
			City:     city,
		}
		mockRepo.On("ResolveLocation", ctx, latitude, longitude).Return(resolved, nil)

		currency, err := service.GetCurrencyForLocation(ctx, latitude, longitude)

		assert.NoError(t, err)
		assert.Equal(t, "TMT", currency)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns USD when no country and no city found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)
		ctx := context.Background()

		latitude, longitude := 0.0, 0.0

		resolved := &ResolvedLocation{
			Location: Location{Latitude: latitude, Longitude: longitude},
			Country:  nil,
			City:     nil,
		}
		mockRepo.On("ResolveLocation", ctx, latitude, longitude).Return(resolved, nil)
		mockRepo.On("FindNearestCity", ctx, latitude, longitude, 50.0).Return(nil, errors.New("no city found"))

		currency, err := service.GetCurrencyForLocation(ctx, latitude, longitude)

		assert.NoError(t, err)
		assert.Equal(t, "USD", currency)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns error when resolution fails", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo)
		ctx := context.Background()

		latitude, longitude := 37.9601, 58.3261

		mockRepo.On("ResolveLocation", ctx, latitude, longitude).Return(nil, errors.New("resolution failed"))

		currency, err := service.GetCurrencyForLocation(ctx, latitude, longitude)

		assert.Error(t, err)
		assert.Empty(t, currency)
		mockRepo.AssertExpectations(t)
	})
}

// =============================================================================
// Test AddDriverRegion
// =============================================================================

func TestAddDriverRegion(t *testing.T) {
	driverID := uuid.New()
	regionID := uuid.New()

	tests := []struct {
		name         string
		driverID     uuid.UUID
		regionID     uuid.UUID
		isPrimary    bool
		region       *Region
		regionErr    error
		addErr       error
		expectError  bool
		errorMessage string
	}{
		{
			name:      "adds driver region successfully",
			driverID:  driverID,
			regionID:  regionID,
			isPrimary: true,
			region: func() *Region {
				r := createTestRegion(uuid.New())
				r.ID = regionID
				r.IsActive = true
				return r
			}(),
			expectError: false,
		},
		{
			name:      "adds non-primary driver region",
			driverID:  driverID,
			regionID:  regionID,
			isPrimary: false,
			region: func() *Region {
				r := createTestRegion(uuid.New())
				r.ID = regionID
				r.IsActive = true
				return r
			}(),
			expectError: false,
		},
		{
			name:         "returns error for non-existent region",
			driverID:     driverID,
			regionID:     uuid.New(),
			isPrimary:    true,
			region:       nil,
			regionErr:    errors.New("not found"),
			expectError:  true,
			errorMessage: "region not found",
		},
		{
			name:      "returns error for inactive region",
			driverID:  driverID,
			regionID:  regionID,
			isPrimary: true,
			region: func() *Region {
				r := createTestRegion(uuid.New())
				r.ID = regionID
				r.Name = "Inactive Region"
				r.IsActive = false
				return r
			}(),
			expectError:  true,
			errorMessage: "not active",
		},
		{
			name:      "returns error when add fails",
			driverID:  driverID,
			regionID:  regionID,
			isPrimary: true,
			region: func() *Region {
				r := createTestRegion(uuid.New())
				r.ID = regionID
				r.IsActive = true
				return r
			}(),
			addErr:      errors.New("database error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo)
			ctx := context.Background()

			mockRepo.On("GetRegionByID", ctx, tt.regionID).Return(tt.region, tt.regionErr)
			if tt.region != nil && tt.region.IsActive {
				mockRepo.On("AddDriverRegion", ctx, tt.driverID, tt.regionID, tt.isPrimary).Return(tt.addErr)
			}

			err := service.AddDriverRegion(ctx, tt.driverID, tt.regionID, tt.isPrimary)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// Test GetDriverRegions
// =============================================================================

func TestGetDriverRegions(t *testing.T) {
	driverID := uuid.New()

	tests := []struct {
		name          string
		driverID      uuid.UUID
		mockReturn    []*DriverRegion
		mockError     error
		expectedCount int
		expectError   bool
	}{
		{
			name:     "returns driver regions",
			driverID: driverID,
			mockReturn: []*DriverRegion{
				{
					ID:        uuid.New(),
					DriverID:  driverID,
					RegionID:  uuid.New(),
					IsPrimary: true,
				},
			},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:          "returns empty list when driver has no regions",
			driverID:      driverID,
			mockReturn:    []*DriverRegion{},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:        "returns error on repository failure",
			driverID:    driverID,
			mockReturn:  nil,
			mockError:   errors.New("database error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo)
			ctx := context.Background()

			mockRepo.On("GetDriverRegions", ctx, tt.driverID).Return(tt.mockReturn, tt.mockError)

			regions, err := service.GetDriverRegions(ctx, tt.driverID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, regions, tt.expectedCount)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// Test GetPricingZonesByCity
// =============================================================================

func TestGetPricingZonesByCity(t *testing.T) {
	cityID := uuid.New()

	tests := []struct {
		name          string
		cityID        uuid.UUID
		mockReturn    []*PricingZone
		mockError     error
		expectedCount int
		expectError   bool
	}{
		{
			name:   "returns pricing zones for city",
			cityID: cityID,
			mockReturn: []*PricingZone{
				createTestPricingZone(cityID),
				createTestPricingZone(cityID),
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:          "returns empty list when no zones",
			cityID:        cityID,
			mockReturn:    []*PricingZone{},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:        "returns error on repository failure",
			cityID:      cityID,
			mockReturn:  nil,
			mockError:   errors.New("database error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo)
			ctx := context.Background()

			mockRepo.On("GetPricingZonesByCity", ctx, tt.cityID).Return(tt.mockReturn, tt.mockError)

			zones, err := service.GetPricingZonesByCity(ctx, tt.cityID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, zones, tt.expectedCount)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// Test CreateCountry
// =============================================================================

func TestCreateCountry(t *testing.T) {
	tests := []struct {
		name        string
		country     *Country
		mockError   error
		expectError bool
	}{
		{
			name:        "creates country successfully",
			country:     createTestCountry(),
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "returns error on repository failure",
			country:     createTestCountry(),
			mockError:   errors.New("duplicate key"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo)
			ctx := context.Background()

			mockRepo.On("CreateCountry", ctx, tt.country).Return(tt.mockError)

			err := service.CreateCountry(ctx, tt.country)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// Test CreateRegion
// =============================================================================

func TestCreateRegion(t *testing.T) {
	country := createTestCountry()

	tests := []struct {
		name         string
		region       *Region
		mockCountry  *Country
		countryErr   error
		createErr    error
		expectError  bool
		errorMessage string
	}{
		{
			name:        "creates region successfully",
			region:      createTestRegion(country.ID),
			mockCountry: country,
			expectError: false,
		},
		{
			name:         "returns error for non-existent country",
			region:       createTestRegion(uuid.New()),
			mockCountry:  nil,
			countryErr:   errors.New("not found"),
			expectError:  true,
			errorMessage: "country not found",
		},
		{
			name:        "returns error on create failure",
			region:      createTestRegion(country.ID),
			mockCountry: country,
			createErr:   errors.New("duplicate key"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo)
			ctx := context.Background()

			mockRepo.On("GetCountryByID", ctx, tt.region.CountryID).Return(tt.mockCountry, tt.countryErr)
			if tt.mockCountry != nil {
				mockRepo.On("CreateRegion", ctx, tt.region).Return(tt.createErr)
			}

			err := service.CreateRegion(ctx, tt.region)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// Test CreateCity
// =============================================================================

func TestCreateCity(t *testing.T) {
	region := createTestRegion(uuid.New())

	tests := []struct {
		name         string
		city         *City
		mockRegion   *Region
		regionErr    error
		createErr    error
		expectError  bool
		errorMessage string
	}{
		{
			name:        "creates city successfully",
			city:        createTestCity(region.ID),
			mockRegion:  region,
			expectError: false,
		},
		{
			name:         "returns error for non-existent region",
			city:         createTestCity(uuid.New()),
			mockRegion:   nil,
			regionErr:    errors.New("not found"),
			expectError:  true,
			errorMessage: "region not found",
		},
		{
			name:        "returns error on create failure",
			city:        createTestCity(region.ID),
			mockRegion:  region,
			createErr:   errors.New("duplicate key"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo)
			ctx := context.Background()

			mockRepo.On("GetRegionByID", ctx, tt.city.RegionID).Return(tt.mockRegion, tt.regionErr)
			if tt.mockRegion != nil {
				mockRepo.On("CreateCity", ctx, tt.city).Return(tt.createErr)
			}

			err := service.CreateCity(ctx, tt.city)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// Test CreatePricingZone
// =============================================================================

func TestCreatePricingZone(t *testing.T) {
	city := createTestCity(uuid.New())

	tests := []struct {
		name         string
		zone         *PricingZone
		mockCity     *City
		cityErr      error
		createErr    error
		expectError  bool
		errorMessage string
	}{
		{
			name:        "creates pricing zone successfully",
			zone:        createTestPricingZone(city.ID),
			mockCity:    city,
			expectError: false,
		},
		{
			name:         "returns error for non-existent city",
			zone:         createTestPricingZone(uuid.New()),
			mockCity:     nil,
			cityErr:      errors.New("not found"),
			expectError:  true,
			errorMessage: "city not found",
		},
		{
			name:        "returns error on create failure",
			zone:        createTestPricingZone(city.ID),
			mockCity:    city,
			createErr:   errors.New("duplicate key"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo)
			ctx := context.Background()

			mockRepo.On("GetCityByID", ctx, tt.zone.CityID).Return(tt.mockCity, tt.cityErr)
			if tt.mockCity != nil {
				mockRepo.On("CreatePricingZone", ctx, tt.zone).Return(tt.createErr)
			}

			err := service.CreatePricingZone(ctx, tt.zone)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// Test GetRegionByID
// =============================================================================

func TestGetRegionByID(t *testing.T) {
	regionID := uuid.New()

	tests := []struct {
		name        string
		id          uuid.UUID
		mockReturn  *Region
		mockError   error
		expectError bool
	}{
		{
			name: "returns region by ID",
			id:   regionID,
			mockReturn: func() *Region {
				r := createTestRegion(uuid.New())
				r.ID = regionID
				return r
			}(),
			expectError: false,
		},
		{
			name:        "returns error for non-existent region",
			id:          uuid.New(),
			mockReturn:  nil,
			mockError:   errors.New("not found"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo)
			ctx := context.Background()

			mockRepo.On("GetRegionByID", ctx, tt.id).Return(tt.mockReturn, tt.mockError)

			region, err := service.GetRegionByID(ctx, tt.id)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, region)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, region)
				assert.Equal(t, tt.id, region.ID)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// Test GetCitiesByRegion
// =============================================================================

func TestGetCitiesByRegion(t *testing.T) {
	regionID := uuid.New()

	tests := []struct {
		name          string
		regionID      uuid.UUID
		mockReturn    []*City
		mockError     error
		expectedCount int
		expectError   bool
	}{
		{
			name:     "returns cities for region",
			regionID: regionID,
			mockReturn: []*City{
				createTestCity(regionID),
				createTestCity(regionID),
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:          "returns empty list when no cities",
			regionID:      regionID,
			mockReturn:    []*City{},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:        "returns error on repository failure",
			regionID:    regionID,
			mockReturn:  nil,
			mockError:   errors.New("database error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo)
			ctx := context.Background()

			mockRepo.On("GetCitiesByRegion", ctx, tt.regionID).Return(tt.mockReturn, tt.mockError)

			cities, err := service.GetCitiesByRegion(ctx, tt.regionID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, cities, tt.expectedCount)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// Test GetCityByID
// =============================================================================

func TestGetCityByID(t *testing.T) {
	cityID := uuid.New()

	tests := []struct {
		name        string
		id          uuid.UUID
		mockReturn  *City
		mockError   error
		expectError bool
	}{
		{
			name: "returns city by ID",
			id:   cityID,
			mockReturn: func() *City {
				c := createTestCity(uuid.New())
				c.ID = cityID
				return c
			}(),
			expectError: false,
		},
		{
			name:        "returns error for non-existent city",
			id:          uuid.New(),
			mockReturn:  nil,
			mockError:   errors.New("not found"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo)
			ctx := context.Background()

			mockRepo.On("GetCityByID", ctx, tt.id).Return(tt.mockReturn, tt.mockError)

			city, err := service.GetCityByID(ctx, tt.id)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, city)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, city)
				assert.Equal(t, tt.id, city.ID)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// Test Response Converters
// =============================================================================

func TestToCountryResponse(t *testing.T) {
	t.Run("returns nil for nil country", func(t *testing.T) {
		result := ToCountryResponse(nil)
		assert.Nil(t, result)
	})

	t.Run("converts country to response", func(t *testing.T) {
		country := createTestCountry()
		result := ToCountryResponse(country)

		assert.NotNil(t, result)
		assert.Equal(t, country.ID, result.ID)
		assert.Equal(t, country.Code, result.Code)
		assert.Equal(t, country.Name, result.Name)
		assert.Equal(t, country.CurrencyCode, result.CurrencyCode)
		assert.Equal(t, country.Timezone, result.Timezone)
		assert.Equal(t, country.IsActive, result.IsActive)
	})

	t.Run("extracts payment methods from JSON", func(t *testing.T) {
		country := createTestCountry()
		country.PaymentMethods = JSON{
			"methods": []interface{}{"cash", "card", "wallet"},
		}

		result := ToCountryResponse(country)

		assert.NotNil(t, result)
		assert.Len(t, result.PaymentMethods, 3)
		assert.Contains(t, result.PaymentMethods, "cash")
		assert.Contains(t, result.PaymentMethods, "card")
		assert.Contains(t, result.PaymentMethods, "wallet")
	})
}

func TestToRegionResponse(t *testing.T) {
	t.Run("returns nil for nil region", func(t *testing.T) {
		result := ToRegionResponse(nil)
		assert.Nil(t, result)
	})

	t.Run("converts region to response with timezone", func(t *testing.T) {
		region := createTestRegion(uuid.New())
		result := ToRegionResponse(region)

		assert.NotNil(t, result)
		assert.Equal(t, region.ID, result.ID)
		assert.Equal(t, region.Code, result.Code)
		assert.Equal(t, region.Name, result.Name)
		assert.Equal(t, "Asia/Ashgabat", result.Timezone)
	})

	t.Run("uses country timezone when region timezone is nil", func(t *testing.T) {
		country := createTestCountry()
		country.Timezone = "Asia/Ashgabat"

		region := createTestRegion(country.ID)
		region.Timezone = nil
		region.Country = country

		result := ToRegionResponse(region)

		assert.NotNil(t, result)
		assert.Equal(t, "Asia/Ashgabat", result.Timezone)
	})
}

func TestToCityResponse(t *testing.T) {
	t.Run("returns nil for nil city", func(t *testing.T) {
		result := ToCityResponse(nil)
		assert.Nil(t, result)
	})

	t.Run("converts city to response with timezone", func(t *testing.T) {
		city := createTestCity(uuid.New())
		result := ToCityResponse(city)

		assert.NotNil(t, result)
		assert.Equal(t, city.ID, result.ID)
		assert.Equal(t, city.Name, result.Name)
		assert.Equal(t, "Asia/Ashgabat", result.Timezone)
		assert.Equal(t, city.CenterLatitude, result.CenterLatitude)
		assert.Equal(t, city.CenterLongitude, result.CenterLongitude)
	})

	t.Run("uses region timezone when city timezone is nil", func(t *testing.T) {
		regionTz := "Asia/Ashgabat"
		region := createTestRegion(uuid.New())
		region.Timezone = &regionTz

		city := createTestCity(region.ID)
		city.Timezone = nil
		city.Region = region

		result := ToCityResponse(city)

		assert.NotNil(t, result)
		assert.Equal(t, "Asia/Ashgabat", result.Timezone)
	})

	t.Run("uses country timezone when city and region timezone are nil", func(t *testing.T) {
		country := createTestCountry()
		country.Timezone = "Asia/Ashgabat"

		region := createTestRegion(country.ID)
		region.Timezone = nil
		region.Country = country

		city := createTestCity(region.ID)
		city.Timezone = nil
		city.Region = region

		result := ToCityResponse(city)

		assert.NotNil(t, result)
		assert.Equal(t, "Asia/Ashgabat", result.Timezone)
	})
}

func TestToPricingZoneResponse(t *testing.T) {
	t.Run("returns nil for nil zone", func(t *testing.T) {
		result := ToPricingZoneResponse(nil)
		assert.Nil(t, result)
	})

	t.Run("converts pricing zone to response", func(t *testing.T) {
		zone := createTestPricingZone(uuid.New())
		result := ToPricingZoneResponse(zone)

		assert.NotNil(t, result)
		assert.Equal(t, zone.ID, result.ID)
		assert.Equal(t, zone.Name, result.Name)
		assert.Equal(t, zone.ZoneType, result.ZoneType)
		assert.Equal(t, zone.Priority, result.Priority)
	})
}

func TestToResolveLocationResponse(t *testing.T) {
	t.Run("returns nil for nil resolved location", func(t *testing.T) {
		result := ToResolveLocationResponse(nil)
		assert.Nil(t, result)
	})

	t.Run("converts resolved location to response", func(t *testing.T) {
		country := createTestCountry()
		region := createTestRegion(country.ID)
		region.Country = country
		city := createTestCity(region.ID)
		city.Region = region
		zone := createTestPricingZone(city.ID)

		resolved := &ResolvedLocation{
			Location:    Location{Latitude: 37.9601, Longitude: 58.3261},
			Country:     country,
			Region:      region,
			City:        city,
			PricingZone: zone,
			Timezone:    "Asia/Ashgabat",
		}

		result := ToResolveLocationResponse(resolved)

		assert.NotNil(t, result)
		assert.NotNil(t, result.Country)
		assert.NotNil(t, result.Region)
		assert.NotNil(t, result.City)
		assert.NotNil(t, result.PricingZone)
		assert.Equal(t, "Asia/Ashgabat", result.Timezone)
	})

	t.Run("handles partial resolved location", func(t *testing.T) {
		country := createTestCountry()

		resolved := &ResolvedLocation{
			Location: Location{Latitude: 37.9601, Longitude: 58.3261},
			Country:  country,
			Timezone: "Asia/Ashgabat",
		}

		result := ToResolveLocationResponse(resolved)

		assert.NotNil(t, result)
		assert.NotNil(t, result.Country)
		assert.Nil(t, result.Region)
		assert.Nil(t, result.City)
		assert.Nil(t, result.PricingZone)
	})
}
