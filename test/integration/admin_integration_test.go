//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/richxcame/ride-hailing/internal/admin"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/richxcame/ride-hailing/test/helpers"
)

const adminServiceKey = "admin"

func init() {
	if services == nil {
		services = make(map[string]*serviceInstance)
	}
}

func startAdminService() *serviceInstance {
	repo := admin.NewRepository(dbPool)
	service := admin.NewService(repo)
	handler := admin.NewHandler(service)

	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.CorrelationID())

	// Health check
	router.GET("/healthz", handler.GetDashboard)

	// Admin API routes - require authentication + admin role
	api := router.Group("/api/v1/admin")
	api.Use(middleware.AuthMiddleware("integration-secret"))
	api.Use(middleware.RequireAdmin())
	{
		api.GET("/dashboard", handler.GetDashboard)

		users := api.Group("/users")
		{
			users.GET("", handler.GetAllUsers)
			users.GET("/:id", handler.GetUser)
			users.POST("/:id/suspend", handler.SuspendUser)
			users.POST("/:id/activate", handler.ActivateUser)
		}

		drivers := api.Group("/drivers")
		{
			drivers.GET("/pending", handler.GetPendingDrivers)
			drivers.POST("/:id/approve", handler.ApproveDriver)
			drivers.POST("/:id/reject", handler.RejectDriver)
		}

		rides := api.Group("/rides")
		{
			rides.GET("/recent", handler.GetRecentRides)
			rides.GET("/stats", handler.GetRideStats)
		}
	}

	server := httptest.NewServer(router)
	return &serviceInstance{server: server, client: server.Client(), baseURL: server.URL}
}

func TestAdminIntegration_Dashboard(t *testing.T) {
	truncateTables(t)

	// Ensure admin service is started
	if _, ok := services[adminServiceKey]; !ok {
		services[adminServiceKey] = startAdminService()
	}

	// Create test data
	admin := registerAndLogin(t, models.RoleAdmin)
	rider1 := registerAndLogin(t, models.RoleRider)
	rider2 := registerAndLogin(t, models.RoleRider)
	_ = registerAndLogin(t, models.RoleDriver)

	// Create a ride
	rideReq := &models.RideRequest{
		PickupLatitude:   37.7749,
		PickupLongitude:  -122.4194,
		PickupAddress:    "123 Market St",
		DropoffLatitude:  37.8044,
		DropoffLongitude: -122.2712,
		DropoffAddress:   "456 Broadway",
	}
	rideResp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, "/api/v1/rides", rideReq, authHeaders(rider1.Token))
	require.True(t, rideResp.Success)

	// Get dashboard stats
	type dashboardResponse struct {
		Users struct {
			Total      int `json:"total"`
			Riders     int `json:"riders"`
			Drivers    int `json:"drivers"`
			Admins     int `json:"admins"`
			Active     int `json:"active"`
			Suspended  int `json:"suspended"`
		} `json:"users"`
		Rides struct {
			Total       int     `json:"total"`
			Completed   int     `json:"completed"`
			Cancelled   int     `json:"cancelled"`
			InProgress  int     `json:"in_progress"`
			TotalRevenue float64 `json:"total_revenue"`
		} `json:"rides"`
		TodayRides struct {
			Total int `json:"total"`
		} `json:"today_rides"`
	}

	dashboardResp := doRequest[dashboardResponse](t, adminServiceKey, http.MethodGet, "/api/v1/admin/dashboard", nil, authHeaders(admin.Token))
	require.True(t, dashboardResp.Success)
	require.Equal(t, 4, dashboardResp.Data.Users.Total) // admin + 2 riders + driver
	require.GreaterOrEqual(t, dashboardResp.Data.Rides.Total, 1)

	// Non-admin should be forbidden
	forbiddenResp := doRawRequest(t, adminServiceKey, http.MethodGet, "/api/v1/admin/dashboard", nil, authHeaders(rider2.Token))
	defer forbiddenResp.Body.Close()
	require.Equal(t, http.StatusForbidden, forbiddenResp.StatusCode)
}

func TestAdminIntegration_UserManagement(t *testing.T) {
	truncateTables(t)

	if _, ok := services[adminServiceKey]; !ok {
		services[adminServiceKey] = startAdminService()
	}

	admin := registerAndLogin(t, models.RoleAdmin)
	rider := registerAndLogin(t, models.RoleRider)

	// Get all users
	type usersResponse struct {
		Users  []*models.User `json:"users"`
		Total  int            `json:"total"`
		Limit  int            `json:"limit"`
		Offset int            `json:"offset"`
	}

	usersResp := doRequest[usersResponse](t, adminServiceKey, http.MethodGet, "/api/v1/admin/users", nil, authHeaders(admin.Token))
	require.True(t, usersResp.Success)
	require.Equal(t, 2, usersResp.Data.Total)
	require.Len(t, usersResp.Data.Users, 2)

	// Get specific user
	userPath := fmt.Sprintf("/api/v1/admin/users/%s", rider.User.ID)
	userResp := doRequest[models.User](t, adminServiceKey, http.MethodGet, userPath, nil, authHeaders(admin.Token))
	require.True(t, userResp.Success)
	require.Equal(t, rider.User.ID, userResp.Data.ID)
	require.Equal(t, rider.User.Email, userResp.Data.Email)

	// Suspend user
	suspendPath := fmt.Sprintf("/api/v1/admin/users/%s/suspend", rider.User.ID)
	suspendResp := doRequest[map[string]string](t, adminServiceKey, http.MethodPost, suspendPath, nil, authHeaders(admin.Token))
	require.True(t, suspendResp.Success)

	// Verify user is suspended in database
	var isActive bool
	err := dbPool.QueryRow(context.Background(), "SELECT is_active FROM users WHERE id = $1", rider.User.ID).Scan(&isActive)
	require.NoError(t, err)
	require.False(t, isActive)

	// Activate user
	activatePath := fmt.Sprintf("/api/v1/admin/users/%s/activate", rider.User.ID)
	activateResp := doRequest[map[string]string](t, adminServiceKey, http.MethodPost, activatePath, nil, authHeaders(admin.Token))
	require.True(t, activateResp.Success)

	// Verify user is active in database
	err = dbPool.QueryRow(context.Background(), "SELECT is_active FROM users WHERE id = $1", rider.User.ID).Scan(&isActive)
	require.NoError(t, err)
	require.True(t, isActive)

	// Non-admin should be forbidden
	forbiddenResp := doRawRequest(t, adminServiceKey, http.MethodGet, "/api/v1/admin/users", nil, authHeaders(rider.Token))
	defer forbiddenResp.Body.Close()
	require.Equal(t, http.StatusForbidden, forbiddenResp.StatusCode)
}

func TestAdminIntegration_DriverApproval(t *testing.T) {
	truncateTables(t)

	if _, ok := services[adminServiceKey]; !ok {
		services[adminServiceKey] = startAdminService()
	}

	admin := registerAndLogin(t, models.RoleAdmin)
	driver := registerAndLogin(t, models.RoleDriver)

	// Create a driver record with pending status
	_, err := dbPool.Exec(context.Background(),
		`INSERT INTO drivers (user_id, vehicle_type, license_number, vehicle_make, vehicle_model,
		vehicle_year, vehicle_color, vehicle_plate, insurance_provider, insurance_policy_number,
		status) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		driver.User.ID, "sedan", "DL12345", "Toyota", "Camry", 2022, "Black", "ABC123",
		"State Farm", "POL123456", "pending")
	require.NoError(t, err)

	// Get driver ID
	var driverID string
	err = dbPool.QueryRow(context.Background(), "SELECT id FROM drivers WHERE user_id = $1", driver.User.ID).Scan(&driverID)
	require.NoError(t, err)

	// Get pending drivers
	type pendingDriversResponse struct {
		Drivers []*models.Driver `json:"drivers"`
		Count   int              `json:"count"`
	}

	pendingResp := doRequest[pendingDriversResponse](t, adminServiceKey, http.MethodGet, "/api/v1/admin/drivers/pending", nil, authHeaders(admin.Token))
	require.True(t, pendingResp.Success)
	require.GreaterOrEqual(t, pendingResp.Data.Count, 1)

	// Approve driver
	approvePath := fmt.Sprintf("/api/v1/admin/drivers/%s/approve", driverID)
	approveResp := doRequest[map[string]string](t, adminServiceKey, http.MethodPost, approvePath, nil, authHeaders(admin.Token))
	require.True(t, approveResp.Success)

	// Verify driver is approved in database
	var status string
	err = dbPool.QueryRow(context.Background(), "SELECT status FROM drivers WHERE id = $1", driverID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "approved", status)

	// Create another driver for rejection test
	driver2 := registerAndLogin(t, models.RoleDriver)
	_, err = dbPool.Exec(context.Background(),
		`INSERT INTO drivers (user_id, vehicle_type, license_number, vehicle_make, vehicle_model,
		vehicle_year, vehicle_color, vehicle_plate, insurance_provider, insurance_policy_number,
		status) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		driver2.User.ID, "suv", "DL54321", "Honda", "CR-V", 2021, "White", "XYZ789",
		"Allstate", "POL789012", "pending")
	require.NoError(t, err)

	var driver2ID string
	err = dbPool.QueryRow(context.Background(), "SELECT id FROM drivers WHERE user_id = $1", driver2.User.ID).Scan(&driver2ID)
	require.NoError(t, err)

	// Reject driver
	rejectPath := fmt.Sprintf("/api/v1/admin/drivers/%s/reject", driver2ID)
	rejectResp := doRequest[map[string]string](t, adminServiceKey, http.MethodPost, rejectPath, nil, authHeaders(admin.Token))
	require.True(t, rejectResp.Success)

	// Verify driver is rejected in database
	err = dbPool.QueryRow(context.Background(), "SELECT status FROM drivers WHERE id = $1", driver2ID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "rejected", status)
}

func TestAdminIntegration_RideMonitoring(t *testing.T) {
	truncateTables(t)

	if _, ok := services[adminServiceKey]; !ok {
		services[adminServiceKey] = startAdminService()
	}

	admin := registerAndLogin(t, models.RoleAdmin)
	rider := registerAndLogin(t, models.RoleRider)
	_ = registerAndLogin(t, models.RoleDriver)

	// Create rides
	rideReq1 := &models.RideRequest{
		PickupLatitude:   37.7749,
		PickupLongitude:  -122.4194,
		PickupAddress:    "123 Market St",
		DropoffLatitude:  37.8044,
		DropoffLongitude: -122.2712,
		DropoffAddress:   "456 Broadway",
	}
	ride1Resp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, "/api/v1/rides", rideReq1, authHeaders(rider.Token))
	require.True(t, ride1Resp.Success)

	rideReq2 := &models.RideRequest{
		PickupLatitude:   37.7849,
		PickupLongitude:  -122.4094,
		PickupAddress:    "789 Pine St",
		DropoffLatitude:  37.7944,
		DropoffLongitude: -122.2812,
		DropoffAddress:   "321 Oak Ave",
	}
	ride2Resp := doRequest[*models.Ride](t, ridesServiceKey, http.MethodPost, "/api/v1/rides", rideReq2, authHeaders(rider.Token))
	require.True(t, ride2Resp.Success)

	// Get recent rides
	type recentRidesResponse struct {
		Rides []*models.Ride `json:"rides"`
		Count int            `json:"count"`
	}

	recentResp := doRequest[recentRidesResponse](t, adminServiceKey, http.MethodGet, "/api/v1/admin/rides/recent?limit=10", nil, authHeaders(admin.Token))
	require.True(t, recentResp.Success)
	require.GreaterOrEqual(t, recentResp.Data.Count, 2)

	// Get ride stats
	type rideStatsResponse struct {
		Total        int     `json:"total"`
		Completed    int     `json:"completed"`
		Cancelled    int     `json:"cancelled"`
		InProgress   int     `json:"in_progress"`
		TotalRevenue float64 `json:"total_revenue"`
	}

	statsResp := doRequest[rideStatsResponse](t, adminServiceKey, http.MethodGet, "/api/v1/admin/rides/stats", nil, authHeaders(admin.Token))
	require.True(t, statsResp.Success)
	require.GreaterOrEqual(t, statsResp.Data.Total, 2)
}

func registerAndLoginWithRole(t *testing.T, role models.UserRole) authSession {
	registerReq := helpers.CreateTestRegisterRequest()
	registerReq.Role = role
	registerReq.Email = uniqueEmail(role)
	registerReq.PhoneNumber = uniquePhoneNumber()

	// Register with admin role if needed
	if role == models.RoleAdmin {
		// Insert admin user directly in the database for testing
		var userID uuid.UUID
		err := dbPool.QueryRow(context.Background(),
			`INSERT INTO users (email, first_name, last_name, phone_number, role, password_hash, is_active, is_verified)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
			registerReq.Email, "Admin", "User", registerReq.PhoneNumber, role,
			"$2a$10$hashedpassword", true, true).Scan(&userID)
		require.NoError(t, err)

		user := &models.User{
			ID:        userID,
			Email:     registerReq.Email,
			FirstName: "Admin",
			LastName:  "User",
			Role:      role,
		}

		// Create wallet for admin
		_, err = dbPool.Exec(context.Background(),
			`INSERT INTO wallets (user_id, balance) VALUES ($1, $2)`,
			userID, 0.0)
		require.NoError(t, err)

		// Generate token for admin
		loginReq := &models.LoginRequest{Email: registerReq.Email, Password: registerReq.Password}
		loginResp := doRequest[*models.LoginResponse](t, authServiceKey, http.MethodPost, "/api/v1/auth/login", loginReq, nil)

		// If login fails, use a test token generation approach
		if !loginResp.Success {
			// For testing, create a simple token
			token := "test-admin-token"
			return authSession{User: user, Token: token}
		}

		return authSession{User: loginResp.Data.User, Token: loginResp.Data.Token}
	}

	// Regular user registration
	return registerAndLogin(t, role)
}
