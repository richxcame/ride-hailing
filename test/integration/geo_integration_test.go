//go:build integration

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/richxcame/ride-hailing/internal/geo"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
	redisClient "github.com/richxcame/ride-hailing/pkg/redis"
)

const geoServiceKey = "geo"

var redisTestClient redisClient.ClientInterface

func startGeoService() *serviceInstance {
	// Initialize Redis client for testing
	if redisTestClient == nil {
		cfg := &config.Config{
			Redis: config.RedisConfig{
				Host:     "localhost",
				Port:     "6379",
				Password: "",
				DB:       1, // Use DB 1 for testing
			},
		}
		client, err := redisClient.NewRedisClient(&cfg.Redis)
		if err != nil {
			panic("Failed to connect to Redis: " + err.Error())
		}
		redisTestClient = client
	}

	service := geo.NewService(redisTestClient)
	handler := geo.NewHandler(service)

	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.CorrelationID())

	api := router.Group("/api/v1")
	api.Use(middleware.AuthMiddleware("integration-secret"))
	{
		geoGroup := api.Group("/geo")
		{
			geoGroup.POST("/location", middleware.RequireRole(models.RoleDriver), handler.UpdateLocation)
			geoGroup.GET("/drivers/:id/location", handler.GetDriverLocation)
			geoGroup.POST("/distance", handler.CalculateDistance)
		}
	}

	server := httptest.NewServer(router)
	return &serviceInstance{server: server, client: server.Client(), baseURL: server.URL}
}

func TestGeoIntegration_UpdateAndRetrieveDriverLocation(t *testing.T) {
	truncateTables(t)

	if _, ok := services[geoServiceKey]; !ok {
		services[geoServiceKey] = startGeoService()
	}

	// Clear Redis geo index before test
	ctx := context.Background()
	redisTestClient.GeoRemove(ctx, "drivers:geo:index", "*")

	driver := registerAndLogin(t, models.RoleDriver)

	// Update driver location
	updateReq := map[string]interface{}{
		"latitude":  37.7749,
		"longitude": -122.4194,
	}

	type updateResponse struct {
		Message string `json:"message"`
	}

	updateResp := doRequest[updateResponse](t, geoServiceKey, http.MethodPost, "/api/v1/geo/location", updateReq, authHeaders(driver.Token))
	require.True(t, updateResp.Success)
	require.Contains(t, updateResp.Data.Message, "successfully")

	// Retrieve driver location
	type locationResponse struct {
		DriverID  string    `json:"driver_id"`
		Latitude  float64   `json:"latitude"`
		Longitude float64   `json:"longitude"`
		Timestamp time.Time `json:"timestamp"`
	}

	locationPath := "/api/v1/geo/drivers/" + driver.User.ID.String() + "/location"
	locationResp := doRequest[locationResponse](t, geoServiceKey, http.MethodGet, locationPath, nil, authHeaders(driver.Token))
	require.True(t, locationResp.Success)
	require.Equal(t, driver.User.ID.String(), locationResp.Data.DriverID)
	require.InEpsilon(t, 37.7749, locationResp.Data.Latitude, 1e-6)
	require.InEpsilon(t, -122.4194, locationResp.Data.Longitude, 1e-6)
}

func TestGeoIntegration_UpdateLocationNonDriverForbidden(t *testing.T) {
	truncateTables(t)

	if _, ok := services[geoServiceKey]; !ok {
		services[geoServiceKey] = startGeoService()
	}

	rider := registerAndLogin(t, models.RoleRider)

	// Try to update location as rider (should fail)
	updateReq := map[string]interface{}{
		"latitude":  37.7749,
		"longitude": -122.4194,
	}

	updateResp := doRawRequest(t, geoServiceKey, http.MethodPost, "/api/v1/geo/location", updateReq, authHeaders(rider.Token))
	defer updateResp.Body.Close()
	require.Equal(t, http.StatusForbidden, updateResp.StatusCode)
}

func TestGeoIntegration_DistanceCalculation(t *testing.T) {
	truncateTables(t)

	if _, ok := services[geoServiceKey]; !ok {
		services[geoServiceKey] = startGeoService()
	}

	rider := registerAndLogin(t, models.RoleRider)

	// Calculate distance between two points
	// San Francisco to Oakland
	distanceReq := map[string]interface{}{
		"from_latitude":  37.7749,  // San Francisco
		"from_longitude": -122.4194,
		"to_latitude":    37.8044,  // Oakland
		"to_longitude":   -122.2712,
	}

	type distanceResponse struct {
		DistanceKm float64 `json:"distance_km"`
		EtaMinutes int     `json:"eta_minutes"`
	}

	distanceResp := doRequest[distanceResponse](t, geoServiceKey, http.MethodPost, "/api/v1/geo/distance", distanceReq, authHeaders(rider.Token))
	require.True(t, distanceResp.Success)
	require.Greater(t, distanceResp.Data.DistanceKm, 0.0)
	require.Greater(t, distanceResp.Data.EtaMinutes, 0)

	// Test distance calculation accuracy (SF to Oakland is roughly 13-15 km)
	require.InDelta(t, 13.0, distanceResp.Data.DistanceKm, 3.0)
}

func TestGeoIntegration_MultipleDriverLocations(t *testing.T) {
	truncateTables(t)

	if _, ok := services[geoServiceKey]; !ok {
		services[geoServiceKey] = startGeoService()
	}

	// Clear Redis geo index before test
	ctx := context.Background()
	redisTestClient.GeoRemove(ctx, "drivers:geo:index", "*")

	// Create multiple drivers at different locations
	driver1 := registerAndLogin(t, models.RoleDriver)
	driver2 := registerAndLogin(t, models.RoleDriver)
	driver3 := registerAndLogin(t, models.RoleDriver)

	// Update locations
	locations := []struct {
		driver authSession
		latitude    float64
		longitude    float64
	}{
		{driver1, 37.7749, -122.4194},  // San Francisco
		{driver2, 37.7849, -122.4094},  // Nearby SF
		{driver3, 37.3382, -121.8863},  // San Jose (far away)
	}

	for _, loc := range locations {
		updateReq := map[string]interface{}{
			"latitude":  loc.latitude,
			"longitude": loc.longitude,
		}
		updateResp := doRequest[map[string]interface{}](t, geoServiceKey, http.MethodPost, "/api/v1/geo/location", updateReq, authHeaders(loc.driver.Token))
		require.True(t, updateResp.Success)
	}

	// Verify all drivers can retrieve their own locations
	for i, loc := range locations {
		locationPath := "/api/v1/geo/drivers/" + loc.driver.User.ID.String() + "/location"
		type locationResponse struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}
		locationResp := doRequest[locationResponse](t, geoServiceKey, http.MethodGet, locationPath, nil, authHeaders(loc.driver.Token))
		require.True(t, locationResp.Success, "Driver %d location retrieval failed", i+1)
		require.InEpsilon(t, loc.latitude, locationResp.Data.Latitude, 1e-6)
		require.InEpsilon(t, loc.longitude, locationResp.Data.Longitude, 1e-6)
	}
}

func TestGeoIntegration_LocationNotFound(t *testing.T) {
	truncateTables(t)

	if _, ok := services[geoServiceKey]; !ok {
		services[geoServiceKey] = startGeoService()
	}

	driver := registerAndLogin(t, models.RoleDriver)

	// Try to get location for a driver who hasn't updated their location
	locationPath := "/api/v1/geo/drivers/" + driver.User.ID.String() + "/location"
	locationResp := doRawRequest(t, geoServiceKey, http.MethodGet, locationPath, nil, authHeaders(driver.Token))
	defer locationResp.Body.Close()
	require.Equal(t, http.StatusNotFound, locationResp.StatusCode)
}

func TestGeoIntegration_InvalidCoordinates(t *testing.T) {
	truncateTables(t)

	if _, ok := services[geoServiceKey]; !ok {
		services[geoServiceKey] = startGeoService()
	}

	driver := registerAndLogin(t, models.RoleDriver)

	testCases := []struct {
		name      string
		latitude  interface{}
		longitude interface{}
	}{
		{"missing latitude", nil, -122.4194},
		{"missing longitude", 37.7749, nil},
		{"invalid latitude type", "invalid", -122.4194},
		{"invalid longitude type", 37.7749, "invalid"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			updateReq := map[string]interface{}{}
			if tc.latitude != nil {
				updateReq["latitude"] = tc.latitude
			}
			if tc.longitude != nil {
				updateReq["longitude"] = tc.longitude
			}

			updateResp := doRawRequest(t, geoServiceKey, http.MethodPost, "/api/v1/geo/location", updateReq, authHeaders(driver.Token))
			defer updateResp.Body.Close()
			require.Equal(t, http.StatusBadRequest, updateResp.StatusCode)
		})
	}
}

func TestGeoIntegration_DistanceCalculationEdgeCases(t *testing.T) {
	truncateTables(t)

	if _, ok := services[geoServiceKey]; !ok {
		services[geoServiceKey] = startGeoService()
	}

	rider := registerAndLogin(t, models.RoleRider)

	testCases := []struct {
		name           string
		fromLatitude   float64
		fromLongitude  float64
		toLatitude     float64
		toLongitude    float64
		expectError    bool
	}{
		{
			name:          "same location",
			fromLatitude:  37.7749,
			fromLongitude: -122.4194,
			toLatitude:    37.7749,
			toLongitude:   -122.4194,
			expectError:   false,
		},
		{
			name:          "cross country",
			fromLatitude:  37.7749,  // San Francisco
			fromLongitude: -122.4194,
			toLatitude:    40.7128,  // New York
			toLongitude:   -74.0060,
			expectError:   false,
		},
		{
			name:          "international",
			fromLatitude:  37.7749,  // San Francisco
			fromLongitude: -122.4194,
			toLatitude:    51.5074,  // London
			toLongitude:   -0.1278,
			expectError:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			distanceReq := map[string]interface{}{
				"from_latitude":  tc.fromLatitude,
				"from_longitude": tc.fromLongitude,
				"to_latitude":    tc.toLatitude,
				"to_longitude":   tc.toLongitude,
			}

			type distanceResponse struct {
				DistanceKm float64 `json:"distance_km"`
				EtaMinutes int     `json:"eta_minutes"`
			}

			distanceResp := doRequest[distanceResponse](t, geoServiceKey, http.MethodPost, "/api/v1/geo/distance", distanceReq, authHeaders(rider.Token))
			require.True(t, distanceResp.Success)
			require.GreaterOrEqual(t, distanceResp.Data.DistanceKm, 0.0)
		})
	}
}
