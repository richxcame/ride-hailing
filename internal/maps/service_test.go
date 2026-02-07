package maps

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/richxcame/ride-hailing/pkg/resilience"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ========================================
// MOCK: MapsProvider
// ========================================

type mockMapsProvider struct {
	mock.Mock
	name Provider
}

func newMockMapsProvider(name Provider) *mockMapsProvider {
	return &mockMapsProvider{name: name}
}

func (m *mockMapsProvider) GetRoute(ctx context.Context, req *RouteRequest) (*RouteResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RouteResponse), args.Error(1)
}

func (m *mockMapsProvider) GetETA(ctx context.Context, req *ETARequest) (*ETAResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ETAResponse), args.Error(1)
}

func (m *mockMapsProvider) GetDistanceMatrix(ctx context.Context, req *DistanceMatrixRequest) (*DistanceMatrixResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DistanceMatrixResponse), args.Error(1)
}

func (m *mockMapsProvider) GetTrafficFlow(ctx context.Context, req *TrafficFlowRequest) (*TrafficFlowResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TrafficFlowResponse), args.Error(1)
}

func (m *mockMapsProvider) GetTrafficIncidents(ctx context.Context, req *TrafficIncidentsRequest) (*TrafficIncidentsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TrafficIncidentsResponse), args.Error(1)
}

func (m *mockMapsProvider) Geocode(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GeocodingResponse), args.Error(1)
}

func (m *mockMapsProvider) ReverseGeocode(ctx context.Context, req *GeocodingRequest) (*GeocodingResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GeocodingResponse), args.Error(1)
}

func (m *mockMapsProvider) SearchPlaces(ctx context.Context, req *PlaceSearchRequest) (*PlaceSearchResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PlaceSearchResponse), args.Error(1)
}

func (m *mockMapsProvider) SnapToRoad(ctx context.Context, req *SnapToRoadRequest) (*SnapToRoadResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SnapToRoadResponse), args.Error(1)
}

func (m *mockMapsProvider) GetSpeedLimits(ctx context.Context, req *SpeedLimitsRequest) (*SpeedLimitsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SpeedLimitsResponse), args.Error(1)
}

func (m *mockMapsProvider) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockMapsProvider) Name() Provider {
	return m.name
}

// ========================================
// MOCK: Redis ClientInterface
// ========================================

type mockRedisClient struct {
	mock.Mock
}

func (m *mockRedisClient) SetWithExpiration(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	args := m.Called(ctx, key, value, expiration)
	return args.Error(0)
}

func (m *mockRedisClient) GetString(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *mockRedisClient) Delete(ctx context.Context, keys ...string) error {
	args := m.Called(ctx, keys)
	return args.Error(0)
}

func (m *mockRedisClient) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *mockRedisClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockRedisClient) GeoAdd(ctx context.Context, key string, longitude, latitude float64, member string) error {
	args := m.Called(ctx, key, longitude, latitude, member)
	return args.Error(0)
}

func (m *mockRedisClient) GeoRadius(ctx context.Context, key string, longitude, latitude, radiusKm float64, count int) ([]string, error) {
	args := m.Called(ctx, key, longitude, latitude, radiusKm, count)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockRedisClient) GeoRemove(ctx context.Context, key string, member string) error {
	args := m.Called(ctx, key, member)
	return args.Error(0)
}

func (m *mockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	args := m.Called(ctx, key, expiration)
	return args.Error(0)
}

func (m *mockRedisClient) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	args := m.Called(ctx, keys)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *mockRedisClient) MGetStrings(ctx context.Context, keys ...string) ([]string, error) {
	args := m.Called(ctx, keys)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// ========================================
// TEST HELPERS
// ========================================

// createTestService creates a service with mock providers for testing
// This bypasses the normal NewService to inject mocks directly
func createTestService(primary MapsProvider, fallbacks []MapsProvider, redis *mockRedisClient, config Config) *Service {
	s := &Service{
		primary:         primary,
		fallbacks:       fallbacks,
		redis:           redis,
		config:          config,
		breakers:        make(map[Provider]*resilience.CircuitBreaker),
		trafficCache:    make(map[string]*TrafficFlowResponse),
		trafficCacheTTL: time.Duration(config.TrafficRefreshSeconds) * time.Second,
	}
	return s
}

// testConfig returns a test configuration
func testConfig(cacheEnabled bool) Config {
	return Config{
		CacheEnabled:          cacheEnabled,
		CacheTTLSeconds:       300,
		CachePrefix:           "test:maps:",
		TrafficRefreshSeconds: 60,
	}
}

// ========================================
// TESTS: GetRoute
// ========================================

func TestGetRoute_Success(t *testing.T) {
	tests := []struct {
		name         string
		cacheEnabled bool
		cacheHit     bool
		setupMocks   func(primary *mockMapsProvider, redis *mockRedisClient, req *RouteRequest)
		validate     func(t *testing.T, resp *RouteResponse)
	}{
		{
			name:         "success - cache disabled, provider returns route",
			cacheEnabled: false,
			cacheHit:     false,
			setupMocks: func(primary *mockMapsProvider, redis *mockRedisClient, req *RouteRequest) {
				routeResp := &RouteResponse{
					Routes: []Route{
						{
							DistanceMeters:  10000,
							DistanceKm:      10.0,
							DurationSeconds: 1200,
							DurationMinutes: 20.0,
						},
					},
					Provider: ProviderGoogle,
				}
				primary.On("GetRoute", mock.Anything, req).Return(routeResp, nil)
			},
			validate: func(t *testing.T, resp *RouteResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Routes, 1)
				assert.Equal(t, 10.0, resp.Routes[0].DistanceKm)
				assert.Equal(t, 20.0, resp.Routes[0].DurationMinutes)
				assert.False(t, resp.CacheHit)
			},
		},
		{
			name:         "success - cache enabled, cache miss, provider returns route",
			cacheEnabled: true,
			cacheHit:     false,
			setupMocks: func(primary *mockMapsProvider, redis *mockRedisClient, req *RouteRequest) {
				// Cache miss
				redis.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return("", errors.New("cache miss"))

				routeResp := &RouteResponse{
					Routes: []Route{
						{
							DistanceMeters:  15000,
							DistanceKm:      15.0,
							DurationSeconds: 1800,
							DurationMinutes: 30.0,
						},
					},
					Provider: ProviderGoogle,
				}
				primary.On("GetRoute", mock.Anything, req).Return(routeResp, nil)

				// Cache set
				redis.On("SetWithExpiration", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)
			},
			validate: func(t *testing.T, resp *RouteResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Routes, 1)
				assert.Equal(t, 15.0, resp.Routes[0].DistanceKm)
				assert.False(t, resp.CacheHit)
			},
		},
		{
			name:         "success - cache enabled, cache hit",
			cacheEnabled: true,
			cacheHit:     true,
			setupMocks: func(primary *mockMapsProvider, redis *mockRedisClient, req *RouteRequest) {
				cachedResp := &RouteResponse{
					Routes: []Route{
						{
							DistanceMeters:  8000,
							DistanceKm:      8.0,
							DurationSeconds: 960,
							DurationMinutes: 16.0,
						},
					},
					Provider: ProviderGoogle,
				}
				cachedData, _ := json.Marshal(cachedResp)
				redis.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return(string(cachedData), nil)
				// Provider should NOT be called on cache hit
			},
			validate: func(t *testing.T, resp *RouteResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Routes, 1)
				assert.Equal(t, 8.0, resp.Routes[0].DistanceKm)
				assert.True(t, resp.CacheHit)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			primary := newMockMapsProvider(ProviderGoogle)
			redis := new(mockRedisClient)

			req := &RouteRequest{
				Origin:      Coordinate{Latitude: 37.7749, Longitude: -122.4194},
				Destination: Coordinate{Latitude: 37.3382, Longitude: -121.8863},
			}

			tt.setupMocks(primary, redis, req)

			config := testConfig(tt.cacheEnabled)
			svc := createTestService(primary, nil, redis, config)

			resp, err := svc.GetRoute(context.Background(), req)

			require.NoError(t, err)
			tt.validate(t, resp)

			primary.AssertExpectations(t)
			if tt.cacheEnabled {
				redis.AssertExpectations(t)
			}
		})
	}
}

func TestGetRoute_ProviderFailure(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	req := &RouteRequest{
		Origin:      Coordinate{Latitude: 37.7749, Longitude: -122.4194},
		Destination: Coordinate{Latitude: 37.3382, Longitude: -121.8863},
	}

	// Cache miss
	redis.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return("", errors.New("cache miss"))

	// Provider fails
	primary.On("GetRoute", mock.Anything, req).Return(nil, errors.New("provider unavailable"))

	config := testConfig(true)
	svc := createTestService(primary, nil, redis, config)

	resp, err := svc.GetRoute(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "all maps providers failed")

	primary.AssertExpectations(t)
	redis.AssertExpectations(t)
}

// ========================================
// TESTS: GetETA
// ========================================

func TestGetETA_Success(t *testing.T) {
	tests := []struct {
		name         string
		cacheEnabled bool
		cacheHit     bool
		setupMocks   func(primary *mockMapsProvider, redis *mockRedisClient, req *ETARequest)
		validate     func(t *testing.T, resp *ETAResponse)
	}{
		{
			name:         "success - provider returns ETA",
			cacheEnabled: false,
			cacheHit:     false,
			setupMocks: func(primary *mockMapsProvider, redis *mockRedisClient, req *ETARequest) {
				etaResp := &ETAResponse{
					DistanceKm:        5.5,
					DistanceMeters:    5500,
					DurationMinutes:   12.0,
					DurationSeconds:   720,
					DurationInTraffic: 14.0,
					TrafficLevel:      TrafficModerate,
					EstimatedArrival:  time.Now().Add(14 * time.Minute),
					Provider:          ProviderGoogle,
					Confidence:        0.9,
				}
				primary.On("GetETA", mock.Anything, req).Return(etaResp, nil)
			},
			validate: func(t *testing.T, resp *ETAResponse) {
				require.NotNil(t, resp)
				assert.Equal(t, 5.5, resp.DistanceKm)
				assert.Equal(t, 12.0, resp.DurationMinutes)
				assert.Equal(t, 14.0, resp.DurationInTraffic)
				assert.Equal(t, TrafficModerate, resp.TrafficLevel)
				assert.Equal(t, 0.9, resp.Confidence)
				assert.False(t, resp.CacheHit)
			},
		},
		{
			name:         "success - cache hit returns cached ETA",
			cacheEnabled: true,
			cacheHit:     true,
			setupMocks: func(primary *mockMapsProvider, redis *mockRedisClient, req *ETARequest) {
				cachedResp := &ETAResponse{
					DistanceKm:        3.2,
					DistanceMeters:    3200,
					DurationMinutes:   8.0,
					DurationSeconds:   480,
					DurationInTraffic: 9.5,
					TrafficLevel:      TrafficLight,
					Provider:          ProviderGoogle,
					Confidence:        0.85,
				}
				cachedData, _ := json.Marshal(cachedResp)
				redis.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return(string(cachedData), nil)
			},
			validate: func(t *testing.T, resp *ETAResponse) {
				require.NotNil(t, resp)
				assert.Equal(t, 3.2, resp.DistanceKm)
				assert.Equal(t, 8.0, resp.DurationMinutes)
				assert.True(t, resp.CacheHit)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			primary := newMockMapsProvider(ProviderGoogle)
			redis := new(mockRedisClient)

			req := &ETARequest{
				Origin:      Coordinate{Latitude: 37.7749, Longitude: -122.4194},
				Destination: Coordinate{Latitude: 37.7849, Longitude: -122.4094},
			}

			tt.setupMocks(primary, redis, req)

			config := testConfig(tt.cacheEnabled)
			svc := createTestService(primary, nil, redis, config)

			resp, err := svc.GetETA(context.Background(), req)

			require.NoError(t, err)
			tt.validate(t, resp)

			primary.AssertExpectations(t)
			if tt.cacheEnabled {
				redis.AssertExpectations(t)
			}
		})
	}
}

func TestGetETA_FallbackToHaversine(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	req := &ETARequest{
		Origin:      Coordinate{Latitude: 37.7749, Longitude: -122.4194},
		Destination: Coordinate{Latitude: 37.3382, Longitude: -121.8863},
	}

	// Cache miss
	redis.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return("", errors.New("cache miss"))

	// Provider fails
	primary.On("GetETA", mock.Anything, req).Return(nil, errors.New("provider unavailable"))

	config := testConfig(true)
	svc := createTestService(primary, nil, redis, config)

	resp, err := svc.GetETA(context.Background(), req)

	// GetETA uses fallbackETA when all providers fail, so it should NOT error
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify fallback ETA properties
	assert.Equal(t, "fallback", string(resp.Provider))
	assert.Equal(t, 0.5, resp.Confidence)
	assert.Equal(t, TrafficModerate, resp.TrafficLevel)
	assert.Greater(t, resp.DistanceKm, 0.0)
	assert.Greater(t, resp.DurationMinutes, 0.0)

	primary.AssertExpectations(t)
	redis.AssertExpectations(t)
}

// ========================================
// TESTS: Provider Fallback Chain
// ========================================

func TestProviderFallback_PrimaryFailsFallbackSucceeds(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	fallback := newMockMapsProvider(ProviderHERE)
	redis := new(mockRedisClient)

	req := &RouteRequest{
		Origin:      Coordinate{Latitude: 37.7749, Longitude: -122.4194},
		Destination: Coordinate{Latitude: 37.3382, Longitude: -121.8863},
	}

	// Cache miss
	redis.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return("", errors.New("cache miss"))

	// Primary fails
	primary.On("GetRoute", mock.Anything, req).Return(nil, errors.New("Google Maps unavailable"))

	// Fallback succeeds
	fallbackResp := &RouteResponse{
		Routes: []Route{
			{
				DistanceMeters:  12000,
				DistanceKm:      12.0,
				DurationSeconds: 1500,
				DurationMinutes: 25.0,
			},
		},
		Provider: ProviderHERE,
	}
	fallback.On("GetRoute", mock.Anything, req).Return(fallbackResp, nil)

	// Cache set
	redis.On("SetWithExpiration", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)

	config := testConfig(true)
	svc := createTestService(primary, []MapsProvider{fallback}, redis, config)

	resp, err := svc.GetRoute(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, ProviderHERE, resp.Provider)
	assert.Equal(t, 12.0, resp.Routes[0].DistanceKm)

	primary.AssertExpectations(t)
	fallback.AssertExpectations(t)
	redis.AssertExpectations(t)
}

func TestProviderFallback_AllProvidersFail(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	fallback := newMockMapsProvider(ProviderHERE)
	redis := new(mockRedisClient)

	req := &RouteRequest{
		Origin:      Coordinate{Latitude: 37.7749, Longitude: -122.4194},
		Destination: Coordinate{Latitude: 37.3382, Longitude: -121.8863},
	}

	// Cache miss
	redis.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return("", errors.New("cache miss"))

	// All providers fail
	primary.On("GetRoute", mock.Anything, req).Return(nil, errors.New("Google Maps unavailable"))
	fallback.On("GetRoute", mock.Anything, req).Return(nil, errors.New("HERE Maps unavailable"))

	config := testConfig(true)
	svc := createTestService(primary, []MapsProvider{fallback}, redis, config)

	resp, err := svc.GetRoute(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "all maps providers failed")

	primary.AssertExpectations(t)
	fallback.AssertExpectations(t)
	redis.AssertExpectations(t)
}

func TestProviderFallback_MultipleFallbacks(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	fallback1 := newMockMapsProvider(ProviderHERE)
	fallback2 := newMockMapsProvider(ProviderMapbox)
	redis := new(mockRedisClient)

	req := &DistanceMatrixRequest{
		Origins:      []Coordinate{{Latitude: 37.7749, Longitude: -122.4194}},
		Destinations: []Coordinate{{Latitude: 37.3382, Longitude: -121.8863}},
	}

	// Primary and first fallback fail
	primary.On("GetDistanceMatrix", mock.Anything, req).Return(nil, errors.New("Google unavailable"))
	fallback1.On("GetDistanceMatrix", mock.Anything, req).Return(nil, errors.New("HERE unavailable"))

	// Second fallback succeeds
	matrixResp := &DistanceMatrixResponse{
		Rows: []DistanceMatrixRow{
			{
				Elements: []DistanceMatrixElement{
					{
						Status:          "OK",
						DistanceKm:      45.5,
						DistanceMeters:  45500,
						DurationMinutes: 55.0,
						DurationSeconds: 3300,
					},
				},
			},
		},
		Provider: ProviderMapbox,
	}
	fallback2.On("GetDistanceMatrix", mock.Anything, req).Return(matrixResp, nil)

	config := testConfig(false)
	svc := createTestService(primary, []MapsProvider{fallback1, fallback2}, redis, config)

	resp, err := svc.GetDistanceMatrix(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, ProviderMapbox, resp.Provider)
	assert.Len(t, resp.Rows, 1)
	assert.Equal(t, 45.5, resp.Rows[0].Elements[0].DistanceKm)

	primary.AssertExpectations(t)
	fallback1.AssertExpectations(t)
	fallback2.AssertExpectations(t)
}

// ========================================
// TESTS: GetDistanceMatrix
// ========================================

func TestGetDistanceMatrix_Success(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	req := &DistanceMatrixRequest{
		Origins: []Coordinate{
			{Latitude: 37.7749, Longitude: -122.4194},
			{Latitude: 37.7849, Longitude: -122.4094},
		},
		Destinations: []Coordinate{
			{Latitude: 37.3382, Longitude: -121.8863},
		},
	}

	matrixResp := &DistanceMatrixResponse{
		Rows: []DistanceMatrixRow{
			{
				Elements: []DistanceMatrixElement{
					{Status: "OK", DistanceKm: 50.0, DurationMinutes: 45.0},
				},
			},
			{
				Elements: []DistanceMatrixElement{
					{Status: "OK", DistanceKm: 48.5, DurationMinutes: 42.0},
				},
			},
		},
		Provider: ProviderGoogle,
	}
	primary.On("GetDistanceMatrix", mock.Anything, req).Return(matrixResp, nil)

	config := testConfig(false)
	svc := createTestService(primary, nil, redis, config)

	resp, err := svc.GetDistanceMatrix(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Rows, 2)
	assert.Equal(t, 50.0, resp.Rows[0].Elements[0].DistanceKm)
	assert.Equal(t, 48.5, resp.Rows[1].Elements[0].DistanceKm)

	primary.AssertExpectations(t)
}

// ========================================
// TESTS: GetTrafficFlow
// ========================================

func TestGetTrafficFlow_Success(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	req := &TrafficFlowRequest{
		Location:     Coordinate{Latitude: 37.7749, Longitude: -122.4194},
		RadiusMeters: 5000,
	}

	trafficResp := &TrafficFlowResponse{
		OverallLevel: TrafficHeavy,
		UpdatedAt:    time.Now(),
		Provider:     ProviderGoogle,
		Segments: []TrafficFlowSegment{
			{
				RoadName:         "Market St",
				CurrentSpeedKmh:  25.0,
				FreeFlowSpeedKmh: 50.0,
				TrafficLevel:     TrafficHeavy,
				JamFactor:        6.5,
			},
		},
	}
	primary.On("GetTrafficFlow", mock.Anything, req).Return(trafficResp, nil)

	config := testConfig(false)
	svc := createTestService(primary, nil, redis, config)

	resp, err := svc.GetTrafficFlow(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, TrafficHeavy, resp.OverallLevel)
	assert.Len(t, resp.Segments, 1)
	assert.Equal(t, "Market St", resp.Segments[0].RoadName)

	primary.AssertExpectations(t)
}

func TestGetTrafficFlow_InMemoryCacheHit(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	req := &TrafficFlowRequest{
		Location:     Coordinate{Latitude: 37.775, Longitude: -122.419},
		RadiusMeters: 5000,
	}

	config := testConfig(false)
	svc := createTestService(primary, nil, redis, config)

	// Pre-populate in-memory cache
	cacheKey := svc.trafficCacheKey(req)
	cachedResp := &TrafficFlowResponse{
		OverallLevel: TrafficLight,
		UpdatedAt:    time.Now(),
		Provider:     ProviderGoogle,
	}
	svc.trafficCache[cacheKey] = cachedResp

	// Provider should NOT be called
	resp, err := svc.GetTrafficFlow(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, TrafficLight, resp.OverallLevel)

	// Primary should not have been called
	primary.AssertNotCalled(t, "GetTrafficFlow")
}

func TestGetTrafficFlow_ProviderFailsReturnsDefault(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	req := &TrafficFlowRequest{
		Location:     Coordinate{Latitude: 37.7749, Longitude: -122.4194},
		RadiusMeters: 5000,
	}

	primary.On("GetTrafficFlow", mock.Anything, req).Return(nil, errors.New("provider error"))

	config := testConfig(false)
	svc := createTestService(primary, nil, redis, config)

	resp, err := svc.GetTrafficFlow(context.Background(), req)

	// Should return default traffic response on error
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, TrafficModerate, resp.OverallLevel)

	primary.AssertExpectations(t)
}

// ========================================
// TESTS: GetTrafficIncidents
// ========================================

func TestGetTrafficIncidents_Success(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	req := &TrafficIncidentsRequest{
		Location:     &Coordinate{Latitude: 37.7749, Longitude: -122.4194},
		RadiusMeters: 10000,
	}

	incidentsResp := &TrafficIncidentsResponse{
		Incidents: []TrafficIncident{
			{
				ID:          "inc-001",
				Type:        "accident",
				Severity:    "major",
				Location:    Coordinate{Latitude: 37.78, Longitude: -122.42},
				Description: "Multi-vehicle accident",
				RoadClosed:  false,
			},
		},
		UpdatedAt: time.Now(),
		Provider:  ProviderGoogle,
	}
	primary.On("GetTrafficIncidents", mock.Anything, req).Return(incidentsResp, nil)

	config := testConfig(false)
	svc := createTestService(primary, nil, redis, config)

	resp, err := svc.GetTrafficIncidents(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Incidents, 1)
	assert.Equal(t, "accident", resp.Incidents[0].Type)
	assert.Equal(t, "major", resp.Incidents[0].Severity)

	primary.AssertExpectations(t)
}

func TestGetTrafficIncidents_ProviderFailsReturnsEmpty(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	req := &TrafficIncidentsRequest{
		Location:     &Coordinate{Latitude: 37.7749, Longitude: -122.4194},
		RadiusMeters: 10000,
	}

	primary.On("GetTrafficIncidents", mock.Anything, req).Return(nil, errors.New("provider error"))

	config := testConfig(false)
	svc := createTestService(primary, nil, redis, config)

	resp, err := svc.GetTrafficIncidents(context.Background(), req)

	// Should return empty incidents on error
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Incidents, 0)

	primary.AssertExpectations(t)
}

// ========================================
// TESTS: Geocode
// ========================================

func TestGeocode_Success(t *testing.T) {
	tests := []struct {
		name         string
		cacheEnabled bool
		cacheHit     bool
		setupMocks   func(primary *mockMapsProvider, redis *mockRedisClient, req *GeocodingRequest)
		validate     func(t *testing.T, resp *GeocodingResponse)
	}{
		{
			name:         "success - provider returns geocoding result",
			cacheEnabled: false,
			cacheHit:     false,
			setupMocks: func(primary *mockMapsProvider, redis *mockRedisClient, req *GeocodingRequest) {
				geoResp := &GeocodingResponse{
					Results: []GeocodingResult{
						{
							FormattedAddress: "1600 Amphitheatre Parkway, Mountain View, CA",
							Coordinate:       Coordinate{Latitude: 37.4224, Longitude: -122.0842},
							PlaceID:          "place123",
							Confidence:       0.95,
						},
					},
					Provider: ProviderGoogle,
				}
				primary.On("Geocode", mock.Anything, req).Return(geoResp, nil)
			},
			validate: func(t *testing.T, resp *GeocodingResponse) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Results, 1)
				assert.Equal(t, 37.4224, resp.Results[0].Coordinate.Latitude)
				assert.Equal(t, 0.95, resp.Results[0].Confidence)
			},
		},
		{
			name:         "success - cache hit",
			cacheEnabled: true,
			cacheHit:     true,
			setupMocks: func(primary *mockMapsProvider, redis *mockRedisClient, req *GeocodingRequest) {
				cachedResp := &GeocodingResponse{
					Results: []GeocodingResult{
						{
							FormattedAddress: "Cached Address",
							Coordinate:       Coordinate{Latitude: 40.7128, Longitude: -74.0060},
							Confidence:       0.88,
						},
					},
					Provider: ProviderGoogle,
				}
				cachedData, _ := json.Marshal(cachedResp)
				redis.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return(string(cachedData), nil)
			},
			validate: func(t *testing.T, resp *GeocodingResponse) {
				require.NotNil(t, resp)
				assert.Equal(t, "Cached Address", resp.Results[0].FormattedAddress)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			primary := newMockMapsProvider(ProviderGoogle)
			redis := new(mockRedisClient)

			req := &GeocodingRequest{
				Address:  "1600 Amphitheatre Parkway",
				Language: "en",
				Region:   "US",
			}

			tt.setupMocks(primary, redis, req)

			config := testConfig(tt.cacheEnabled)
			svc := createTestService(primary, nil, redis, config)

			resp, err := svc.Geocode(context.Background(), req)

			require.NoError(t, err)
			tt.validate(t, resp)

			primary.AssertExpectations(t)
			if tt.cacheEnabled {
				redis.AssertExpectations(t)
			}
		})
	}
}

// ========================================
// TESTS: ReverseGeocode
// ========================================

func TestReverseGeocode_Success(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	req := &GeocodingRequest{
		Coordinate: &Coordinate{Latitude: 37.4224, Longitude: -122.0842},
		Language:   "en",
	}

	// Cache miss
	redis.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return("", errors.New("cache miss"))

	geoResp := &GeocodingResponse{
		Results: []GeocodingResult{
			{
				FormattedAddress: "1600 Amphitheatre Parkway, Mountain View, CA",
				Coordinate:       Coordinate{Latitude: 37.4224, Longitude: -122.0842},
				PlaceID:          "place456",
				Confidence:       0.92,
			},
		},
		Provider: ProviderGoogle,
	}
	primary.On("ReverseGeocode", mock.Anything, req).Return(geoResp, nil)

	// Cache set
	redis.On("SetWithExpiration", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)

	config := testConfig(true)
	svc := createTestService(primary, nil, redis, config)

	resp, err := svc.ReverseGeocode(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Results, 1)
	assert.Contains(t, resp.Results[0].FormattedAddress, "Amphitheatre")

	primary.AssertExpectations(t)
	redis.AssertExpectations(t)
}

// ========================================
// TESTS: SearchPlaces
// ========================================

func TestSearchPlaces_Success(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	req := &PlaceSearchRequest{
		Query:        "coffee shop",
		Location:     &Coordinate{Latitude: 37.7749, Longitude: -122.4194},
		RadiusMeters: 1000,
		OpenNow:      true,
	}

	placesResp := &PlaceSearchResponse{
		Results: []Place{
			{
				PlaceID:          "place789",
				Name:             "Blue Bottle Coffee",
				FormattedAddress: "66 Mint St, San Francisco, CA",
				Coordinate:       Coordinate{Latitude: 37.7823, Longitude: -122.4056},
				Types:            []string{"cafe", "food"},
			},
			{
				PlaceID:          "place790",
				Name:             "Starbucks",
				FormattedAddress: "100 Market St, San Francisco, CA",
				Coordinate:       Coordinate{Latitude: 37.7941, Longitude: -122.3951},
				Types:            []string{"cafe", "food"},
			},
		},
		Provider: ProviderGoogle,
	}
	primary.On("SearchPlaces", mock.Anything, req).Return(placesResp, nil)

	config := testConfig(false)
	svc := createTestService(primary, nil, redis, config)

	resp, err := svc.SearchPlaces(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Results, 2)
	assert.Equal(t, "Blue Bottle Coffee", resp.Results[0].Name)

	primary.AssertExpectations(t)
}

// ========================================
// TESTS: SnapToRoad
// ========================================

func TestSnapToRoad_Success(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	req := &SnapToRoadRequest{
		Path: []Coordinate{
			{Latitude: 37.7749, Longitude: -122.4194},
			{Latitude: 37.7759, Longitude: -122.4184},
			{Latitude: 37.7769, Longitude: -122.4174},
		},
		Interpolate: true,
	}

	snapResp := &SnapToRoadResponse{
		SnappedPoints: []SnappedPoint{
			{
				Location:      Coordinate{Latitude: 37.7750, Longitude: -122.4193},
				OriginalIndex: 0,
				RoadName:      "Market St",
			},
			{
				Location:      Coordinate{Latitude: 37.7760, Longitude: -122.4183},
				OriginalIndex: 1,
				RoadName:      "Market St",
			},
			{
				Location:      Coordinate{Latitude: 37.7770, Longitude: -122.4173},
				OriginalIndex: 2,
				RoadName:      "Market St",
			},
		},
		Provider: ProviderGoogle,
	}
	primary.On("SnapToRoad", mock.Anything, req).Return(snapResp, nil)

	config := testConfig(false)
	svc := createTestService(primary, nil, redis, config)

	resp, err := svc.SnapToRoad(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.SnappedPoints, 3)
	assert.Equal(t, "Market St", resp.SnappedPoints[0].RoadName)

	primary.AssertExpectations(t)
}

// ========================================
// TESTS: GetSpeedLimits
// ========================================

func TestGetSpeedLimits_Success(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	req := &SpeedLimitsRequest{
		Path: []Coordinate{
			{Latitude: 37.7749, Longitude: -122.4194},
			{Latitude: 37.7769, Longitude: -122.4174},
		},
	}

	speedResp := &SpeedLimitsResponse{
		SpeedLimits: []SpeedLimit{
			{
				PlaceID:       "road123",
				SpeedLimitKmh: 50,
				SpeedLimitMph: 31,
			},
			{
				PlaceID:       "road124",
				SpeedLimitKmh: 65,
				SpeedLimitMph: 40,
			},
		},
		Provider: ProviderGoogle,
	}
	primary.On("GetSpeedLimits", mock.Anything, req).Return(speedResp, nil)

	config := testConfig(false)
	svc := createTestService(primary, nil, redis, config)

	resp, err := svc.GetSpeedLimits(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.SpeedLimits, 2)
	assert.Equal(t, 50, resp.SpeedLimits[0].SpeedLimitKmh)
	assert.Equal(t, 65, resp.SpeedLimits[1].SpeedLimitKmh)

	primary.AssertExpectations(t)
}

// ========================================
// TESTS: HealthCheck
// ========================================

func TestHealthCheck_Success(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	fallback := newMockMapsProvider(ProviderHERE)
	redis := new(mockRedisClient)

	primary.On("HealthCheck", mock.Anything).Return(nil)
	fallback.On("HealthCheck", mock.Anything).Return(nil)

	config := testConfig(false)
	svc := createTestService(primary, []MapsProvider{fallback}, redis, config)

	results := svc.HealthCheck(context.Background())

	assert.Len(t, results, 2)
	assert.NoError(t, results[ProviderGoogle])
	assert.NoError(t, results[ProviderHERE])

	primary.AssertExpectations(t)
	fallback.AssertExpectations(t)
}

func TestHealthCheck_PartialFailure(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	fallback := newMockMapsProvider(ProviderHERE)
	redis := new(mockRedisClient)

	primary.On("HealthCheck", mock.Anything).Return(nil)
	fallback.On("HealthCheck", mock.Anything).Return(errors.New("HERE service unavailable"))

	config := testConfig(false)
	svc := createTestService(primary, []MapsProvider{fallback}, redis, config)

	results := svc.HealthCheck(context.Background())

	assert.Len(t, results, 2)
	assert.NoError(t, results[ProviderGoogle])
	assert.Error(t, results[ProviderHERE])

	primary.AssertExpectations(t)
	fallback.AssertExpectations(t)
}

// ========================================
// TESTS: GetPrimaryProvider
// ========================================

func TestGetPrimaryProvider(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	config := testConfig(false)
	svc := createTestService(primary, nil, redis, config)

	provider := svc.GetPrimaryProvider()

	assert.Equal(t, ProviderGoogle, provider)
}

// ========================================
// TESTS: GetTrafficAwareETA
// ========================================

func TestGetTrafficAwareETA_Success(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	origin := Coordinate{Latitude: 37.7749, Longitude: -122.4194}
	destination := Coordinate{Latitude: 37.3382, Longitude: -121.8863}

	// Traffic flow response
	trafficResp := &TrafficFlowResponse{
		OverallLevel: TrafficHeavy,
		UpdatedAt:    time.Now(),
		Provider:     ProviderGoogle,
	}
	primary.On("GetTrafficFlow", mock.Anything, mock.AnythingOfType("*maps.TrafficFlowRequest")).Return(trafficResp, nil)

	// ETA response (cache miss)
	redis.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return("", errors.New("cache miss"))

	etaResp := &ETAResponse{
		DistanceKm:        50.0,
		DistanceMeters:    50000,
		DurationMinutes:   45.0,
		DurationSeconds:   2700,
		DurationInTraffic: 55.0,
		TrafficLevel:      TrafficModerate, // Will be overwritten by traffic flow
		Provider:          ProviderGoogle,
		Confidence:        0.85,
	}
	primary.On("GetETA", mock.Anything, mock.AnythingOfType("*maps.ETARequest")).Return(etaResp, nil)

	// Cache set
	redis.On("SetWithExpiration", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)

	config := testConfig(true)
	svc := createTestService(primary, nil, redis, config)

	resp, err := svc.GetTrafficAwareETA(context.Background(), origin, destination)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 50.0, resp.DistanceKm)
	assert.Equal(t, TrafficHeavy, resp.TrafficLevel) // Should be updated from traffic flow

	primary.AssertExpectations(t)
	redis.AssertExpectations(t)
}

// ========================================
// TESTS: BatchGetETA
// ========================================

func TestBatchGetETA_Success(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	requests := []*ETARequest{
		{
			Origin:      Coordinate{Latitude: 37.7749, Longitude: -122.4194},
			Destination: Coordinate{Latitude: 37.3382, Longitude: -121.8863},
		},
		{
			Origin:      Coordinate{Latitude: 37.7849, Longitude: -122.4094},
			Destination: Coordinate{Latitude: 37.4382, Longitude: -121.9863},
		},
	}

	// Cache miss for all requests
	redis.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return("", errors.New("cache miss"))

	// ETA responses
	for i := range requests {
		etaResp := &ETAResponse{
			DistanceKm:        float64(30 + i*10),
			DistanceMeters:    (30 + i*10) * 1000,
			DurationMinutes:   float64(25 + i*5),
			DurationSeconds:   (25 + i*5) * 60,
			DurationInTraffic: float64(30 + i*5),
			Provider:          ProviderGoogle,
			Confidence:        0.9,
		}
		primary.On("GetETA", mock.Anything, mock.AnythingOfType("*maps.ETARequest")).Return(etaResp, nil).Once()
	}

	// Cache sets
	redis.On("SetWithExpiration", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)

	config := testConfig(true)
	svc := createTestService(primary, nil, redis, config)

	responses, err := svc.BatchGetETA(context.Background(), requests)

	require.NoError(t, err)
	require.Len(t, responses, 2)

	// All responses should be valid
	for _, resp := range responses {
		assert.NotNil(t, resp)
		assert.Greater(t, resp.DistanceKm, 0.0)
	}

	primary.AssertExpectations(t)
}

func TestBatchGetETA_PartialFailure(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	requests := []*ETARequest{
		{
			Origin:      Coordinate{Latitude: 37.7749, Longitude: -122.4194},
			Destination: Coordinate{Latitude: 37.3382, Longitude: -121.8863},
		},
		{
			Origin:      Coordinate{Latitude: 37.7849, Longitude: -122.4094},
			Destination: Coordinate{Latitude: 37.4382, Longitude: -121.9863},
		},
	}

	// Cache miss for all requests
	redis.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return("", errors.New("cache miss"))

	// First request succeeds
	etaResp := &ETAResponse{
		DistanceKm:      30.0,
		DurationMinutes: 25.0,
		Provider:        ProviderGoogle,
		Confidence:      0.9,
	}
	primary.On("GetETA", mock.Anything, mock.AnythingOfType("*maps.ETARequest")).Return(etaResp, nil).Once()

	// Second request fails - provider error
	primary.On("GetETA", mock.Anything, mock.AnythingOfType("*maps.ETARequest")).Return(nil, errors.New("provider error")).Once()

	// Cache set for successful request
	redis.On("SetWithExpiration", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)

	config := testConfig(true)
	svc := createTestService(primary, nil, redis, config)

	responses, err := svc.BatchGetETA(context.Background(), requests)

	// Should not return an error, but use fallback for failed request
	require.NoError(t, err)
	require.Len(t, responses, 2)

	// All responses should be valid (second one uses fallback)
	for _, resp := range responses {
		assert.NotNil(t, resp)
	}

	// Second response should be from fallback (low confidence)
	// Note: Due to goroutine ordering, we can't guarantee which is which
	// So we just verify all responses are valid

	primary.AssertExpectations(t)
}

// ========================================
// TESTS: OptimizeWaypoints
// ========================================

func TestOptimizeWaypoints_EmptyWaypoints(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	origin := Coordinate{Latitude: 37.7749, Longitude: -122.4194}
	destination := Coordinate{Latitude: 37.3382, Longitude: -121.8863}
	waypoints := []Coordinate{}

	config := testConfig(false)
	svc := createTestService(primary, nil, redis, config)

	order, err := svc.OptimizeWaypoints(context.Background(), origin, waypoints, destination)

	require.NoError(t, err)
	assert.Len(t, order, 0)
}

func TestOptimizeWaypoints_SingleWaypoint(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	origin := Coordinate{Latitude: 37.7749, Longitude: -122.4194}
	destination := Coordinate{Latitude: 37.3382, Longitude: -121.8863}
	waypoints := []Coordinate{
		{Latitude: 37.5, Longitude: -122.0},
	}

	config := testConfig(false)
	svc := createTestService(primary, nil, redis, config)

	order, err := svc.OptimizeWaypoints(context.Background(), origin, waypoints, destination)

	require.NoError(t, err)
	assert.Equal(t, []int{0}, order)
}

func TestOptimizeWaypoints_MultipleWaypoints(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	origin := Coordinate{Latitude: 37.0, Longitude: -122.0}
	destination := Coordinate{Latitude: 38.0, Longitude: -122.0}
	waypoints := []Coordinate{
		{Latitude: 37.8, Longitude: -122.0}, // Furthest from origin
		{Latitude: 37.3, Longitude: -122.0}, // Closest to origin
		{Latitude: 37.5, Longitude: -122.0}, // Middle
	}

	// Distance matrix response - create a realistic matrix
	// Origin(0), WP0(1), WP1(2), WP2(3), Dest(4)
	matrixResp := &DistanceMatrixResponse{
		Rows: []DistanceMatrixRow{
			// From origin
			{Elements: []DistanceMatrixElement{
				{Status: "OK", DistanceKm: 0},
				{Status: "OK", DistanceKm: 80},  // Origin to WP0
				{Status: "OK", DistanceKm: 30},  // Origin to WP1 (closest)
				{Status: "OK", DistanceKm: 50},  // Origin to WP2
				{Status: "OK", DistanceKm: 100}, // Origin to Dest
			}},
			// From WP0
			{Elements: []DistanceMatrixElement{
				{Status: "OK", DistanceKm: 80},
				{Status: "OK", DistanceKm: 0},
				{Status: "OK", DistanceKm: 50},
				{Status: "OK", DistanceKm: 30},
				{Status: "OK", DistanceKm: 20},
			}},
			// From WP1
			{Elements: []DistanceMatrixElement{
				{Status: "OK", DistanceKm: 30},
				{Status: "OK", DistanceKm: 50},
				{Status: "OK", DistanceKm: 0},
				{Status: "OK", DistanceKm: 20},
				{Status: "OK", DistanceKm: 70},
			}},
			// From WP2
			{Elements: []DistanceMatrixElement{
				{Status: "OK", DistanceKm: 50},
				{Status: "OK", DistanceKm: 30},
				{Status: "OK", DistanceKm: 20},
				{Status: "OK", DistanceKm: 0},
				{Status: "OK", DistanceKm: 50},
			}},
			// From Dest
			{Elements: []DistanceMatrixElement{
				{Status: "OK", DistanceKm: 100},
				{Status: "OK", DistanceKm: 20},
				{Status: "OK", DistanceKm: 70},
				{Status: "OK", DistanceKm: 50},
				{Status: "OK", DistanceKm: 0},
			}},
		},
		Provider: ProviderGoogle,
	}
	primary.On("GetDistanceMatrix", mock.Anything, mock.AnythingOfType("*maps.DistanceMatrixRequest")).Return(matrixResp, nil)

	config := testConfig(false)
	svc := createTestService(primary, nil, redis, config)

	order, err := svc.OptimizeWaypoints(context.Background(), origin, waypoints, destination)

	require.NoError(t, err)
	assert.Len(t, order, 3)
	// Greedy nearest neighbor should pick WP1 (index 1) first since it's closest to origin
	assert.Equal(t, 1, order[0]) // WP1 is closest to origin

	primary.AssertExpectations(t)
}

// ========================================
// TESTS: fallbackETA (Haversine calculation)
// ========================================

func TestFallbackETA_Calculation(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	config := testConfig(false)
	svc := createTestService(primary, nil, redis, config)

	req := &ETARequest{
		Origin:      Coordinate{Latitude: 37.7749, Longitude: -122.4194}, // San Francisco
		Destination: Coordinate{Latitude: 37.3382, Longitude: -121.8863}, // San Jose
	}

	resp := svc.fallbackETA(req)

	require.NotNil(t, resp)

	// Verify properties
	assert.Equal(t, "fallback", string(resp.Provider))
	assert.Equal(t, 0.5, resp.Confidence)
	assert.Equal(t, TrafficModerate, resp.TrafficLevel)

	// SF to San Jose is roughly 60-70km straight line
	assert.Greater(t, resp.DistanceKm, 50.0)
	assert.Less(t, resp.DistanceKm, 80.0)

	// Duration should be reasonable (distance / 30 km/h * 1.2 buffer)
	// For ~60km at 30km/h with 1.2x buffer = ~144 minutes
	assert.Greater(t, resp.DurationMinutes, 100.0)
	assert.Less(t, resp.DurationMinutes, 200.0)

	// DistanceMeters should match DistanceKm
	assert.Equal(t, int(resp.DistanceKm*1000), resp.DistanceMeters)

	// DurationSeconds should match DurationMinutes
	assert.Equal(t, int(resp.DurationMinutes*60), resp.DurationSeconds)

	// EstimatedArrival should be in the future
	assert.True(t, resp.EstimatedArrival.After(time.Now()))
}

// ========================================
// TESTS: haversineDistance
// ========================================

func TestHaversineDistance(t *testing.T) {
	tests := []struct {
		name     string
		lat1     float64
		lon1     float64
		lat2     float64
		lon2     float64
		minKm    float64
		maxKm    float64
	}{
		{
			name:  "same point returns zero",
			lat1:  37.7749,
			lon1:  -122.4194,
			lat2:  37.7749,
			lon2:  -122.4194,
			minKm: 0.0,
			maxKm: 0.001,
		},
		{
			name:  "SF to San Jose ~60km",
			lat1:  37.7749,
			lon1:  -122.4194,
			lat2:  37.3382,
			lon2:  -121.8863,
			minKm: 55.0,
			maxKm: 70.0,
		},
		{
			name:  "SF to LA ~550km",
			lat1:  37.7749,
			lon1:  -122.4194,
			lat2:  34.0522,
			lon2:  -118.2437,
			minKm: 530.0,
			maxKm: 580.0,
		},
		{
			name:  "NYC to London ~5500km",
			lat1:  40.7128,
			lon1:  -74.0060,
			lat2:  51.5074,
			lon2:  -0.1278,
			minKm: 5500.0,
			maxKm: 5700.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := haversineDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)

			assert.GreaterOrEqual(t, distance, tt.minKm)
			assert.LessOrEqual(t, distance, tt.maxKm)
		})
	}
}

// ========================================
// TESTS: Cache Key Generation
// ========================================

func TestRouteCacheKey_Deterministic(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	config := testConfig(true)
	svc := createTestService(primary, nil, redis, config)

	req := &RouteRequest{
		Origin:       Coordinate{Latitude: 37.7749, Longitude: -122.4194},
		Destination:  Coordinate{Latitude: 37.3382, Longitude: -121.8863},
		AvoidTolls:   true,
		AvoidHighways: false,
		AvoidFerries: false,
	}

	key1 := svc.routeCacheKey(req)
	key2 := svc.routeCacheKey(req)

	assert.Equal(t, key1, key2)
	assert.True(t, len(key1) > 0)
}

func TestETACacheKey_DifferentCoordinates(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	config := testConfig(true)
	svc := createTestService(primary, nil, redis, config)

	req1 := &ETARequest{
		Origin:      Coordinate{Latitude: 37.7749, Longitude: -122.4194},
		Destination: Coordinate{Latitude: 37.3382, Longitude: -121.8863},
	}
	req2 := &ETARequest{
		Origin:      Coordinate{Latitude: 37.7849, Longitude: -122.4294},
		Destination: Coordinate{Latitude: 37.3482, Longitude: -121.8963},
	}

	key1 := svc.etaCacheKey(req1)
	key2 := svc.etaCacheKey(req2)

	assert.NotEqual(t, key1, key2)
}

// ========================================
// TESTS: TrafficLevel Methods
// ========================================

func TestTrafficLevel_TrafficMultiplier(t *testing.T) {
	tests := []struct {
		level      TrafficLevel
		multiplier float64
	}{
		{TrafficFreeFlow, 1.0},
		{TrafficLight, 1.05},
		{TrafficModerate, 1.1},
		{TrafficHeavy, 1.2},
		{TrafficSevere, 1.35},
		{TrafficBlocked, 1.5},
		{TrafficLevel("unknown"), 1.1}, // Default
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			assert.Equal(t, tt.multiplier, tt.level.TrafficMultiplier())
		})
	}
}

func TestTrafficLevel_IsHighTraffic(t *testing.T) {
	tests := []struct {
		level    TrafficLevel
		isHigh   bool
	}{
		{TrafficFreeFlow, false},
		{TrafficLight, false},
		{TrafficModerate, false},
		{TrafficHeavy, true},
		{TrafficSevere, true},
		{TrafficBlocked, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			assert.Equal(t, tt.isHigh, tt.level.IsHighTraffic())
		})
	}
}

// ========================================
// TESTS: Redis unavailable scenarios
// ========================================

func TestGetRoute_NilRedis(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	req := &RouteRequest{
		Origin:      Coordinate{Latitude: 37.7749, Longitude: -122.4194},
		Destination: Coordinate{Latitude: 37.3382, Longitude: -121.8863},
	}

	routeResp := &RouteResponse{
		Routes: []Route{
			{DistanceKm: 10.0, DurationMinutes: 20.0},
		},
		Provider: ProviderGoogle,
	}
	primary.On("GetRoute", mock.Anything, req).Return(routeResp, nil)

	// Test with cache disabled - simulates when redis is not available
	config := testConfig(false) // Cache disabled
	svc := createTestService(primary, nil, redis, config)

	resp, err := svc.GetRoute(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 10.0, resp.Routes[0].DistanceKm)

	primary.AssertExpectations(t)
}

func TestGetRoute_CacheSetFails(t *testing.T) {
	primary := newMockMapsProvider(ProviderGoogle)
	redis := new(mockRedisClient)

	req := &RouteRequest{
		Origin:      Coordinate{Latitude: 37.7749, Longitude: -122.4194},
		Destination: Coordinate{Latitude: 37.3382, Longitude: -121.8863},
	}

	// Cache miss
	redis.On("GetString", mock.Anything, mock.AnythingOfType("string")).Return("", errors.New("cache miss"))

	routeResp := &RouteResponse{
		Routes: []Route{
			{DistanceKm: 10.0, DurationMinutes: 20.0},
		},
		Provider: ProviderGoogle,
	}
	primary.On("GetRoute", mock.Anything, req).Return(routeResp, nil)

	// Cache set fails - should not affect the response
	redis.On("SetWithExpiration", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(errors.New("redis connection failed"))

	config := testConfig(true)
	svc := createTestService(primary, nil, redis, config)

	resp, err := svc.GetRoute(context.Background(), req)

	// Should still succeed even if cache set fails
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 10.0, resp.Routes[0].DistanceKm)

	primary.AssertExpectations(t)
	redis.AssertExpectations(t)
}
