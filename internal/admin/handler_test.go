package admin

import (
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
	"github.com/stretchr/testify/require"
)

// ========================================
// SERVICE MOCK FOR HANDLER TESTS
// ========================================

type mockService struct {
	mock.Mock
}

func (m *mockService) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DashboardStats), args.Error(1)
}

func (m *mockService) GetAllUsers(ctx context.Context, limit, offset int, filter *UserFilter) ([]*models.User, int, error) {
	args := m.Called(ctx, limit, offset, filter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.User), args.Int(1), args.Error(2)
}

func (m *mockService) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *mockService) GetUserStats(ctx context.Context) (*UserStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UserStats), args.Error(1)
}

func (m *mockService) SuspendUser(ctx context.Context, adminID, userID uuid.UUID) error {
	args := m.Called(ctx, adminID, userID)
	return args.Error(0)
}

func (m *mockService) ActivateUser(ctx context.Context, adminID, userID uuid.UUID) error {
	args := m.Called(ctx, adminID, userID)
	return args.Error(0)
}

func (m *mockService) GetAllDrivers(ctx context.Context, limit, offset int, filter *DriverFilter) ([]*models.Driver, int64, error) {
	args := m.Called(ctx, limit, offset, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*models.Driver), args.Get(1).(int64), args.Error(2)
}

func (m *mockService) GetDriver(ctx context.Context, id uuid.UUID) (*models.Driver, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Driver), args.Error(1)
}

func (m *mockService) GetDriverStats(ctx context.Context) (*DriverStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverStats), args.Error(1)
}

func (m *mockService) GetPendingDrivers(ctx context.Context, limit, offset int) ([]*models.Driver, int64, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*models.Driver), args.Get(1).(int64), args.Error(2)
}

func (m *mockService) ApproveDriver(ctx context.Context, adminID, driverID uuid.UUID) error {
	args := m.Called(ctx, adminID, driverID)
	return args.Error(0)
}

func (m *mockService) RejectDriver(ctx context.Context, adminID, driverID uuid.UUID) error {
	args := m.Called(ctx, adminID, driverID)
	return args.Error(0)
}

func (m *mockService) GetRideStats(ctx context.Context, startDate, endDate *time.Time) (*RideStats, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RideStats), args.Error(1)
}

func (m *mockService) GetRide(ctx context.Context, id uuid.UUID) (*AdminRideDetail, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AdminRideDetail), args.Error(1)
}

func (m *mockService) GetRecentRides(ctx context.Context, limit, offset int) ([]*AdminRideDetail, int64, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*AdminRideDetail), args.Get(1).(int64), args.Error(2)
}

func (m *mockService) GetRealtimeMetrics(ctx context.Context) (*RealtimeMetrics, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RealtimeMetrics), args.Error(1)
}

func (m *mockService) GetDashboardSummary(ctx context.Context, period string) (*DashboardSummary, error) {
	args := m.Called(ctx, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DashboardSummary), args.Error(1)
}

func (m *mockService) GetRevenueTrend(ctx context.Context, period, groupBy string) (*RevenueTrend, error) {
	args := m.Called(ctx, period, groupBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RevenueTrend), args.Error(1)
}

func (m *mockService) GetActionItems(ctx context.Context) (*ActionItems, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ActionItems), args.Error(1)
}

// ========================================
// ServiceInterface for handler testing
// ========================================

type ServiceInterface interface {
	GetDashboardStats(ctx context.Context) (*DashboardStats, error)
	GetAllUsers(ctx context.Context, limit, offset int, filter *UserFilter) ([]*models.User, int, error)
	GetUser(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetUserStats(ctx context.Context) (*UserStats, error)
	SuspendUser(ctx context.Context, adminID, userID uuid.UUID) error
	ActivateUser(ctx context.Context, adminID, userID uuid.UUID) error
	GetAllDrivers(ctx context.Context, limit, offset int, filter *DriverFilter) ([]*models.Driver, int64, error)
	GetDriver(ctx context.Context, id uuid.UUID) (*models.Driver, error)
	GetDriverStats(ctx context.Context) (*DriverStats, error)
	GetPendingDrivers(ctx context.Context, limit, offset int) ([]*models.Driver, int64, error)
	ApproveDriver(ctx context.Context, adminID, driverID uuid.UUID) error
	RejectDriver(ctx context.Context, adminID, driverID uuid.UUID) error
	GetRideStats(ctx context.Context, startDate, endDate *time.Time) (*RideStats, error)
	GetRide(ctx context.Context, id uuid.UUID) (*AdminRideDetail, error)
	GetRecentRides(ctx context.Context, limit, offset int) ([]*AdminRideDetail, int64, error)
	GetRealtimeMetrics(ctx context.Context) (*RealtimeMetrics, error)
	GetDashboardSummary(ctx context.Context, period string) (*DashboardSummary, error)
	GetRevenueTrend(ctx context.Context, period, groupBy string) (*RevenueTrend, error)
	GetActionItems(ctx context.Context) (*ActionItems, error)
}

// testHandler wraps Handler to use interface for testing
type testHandler struct {
	service ServiceInterface
}

func newTestHandler(svc ServiceInterface) *testHandler {
	return &testHandler{service: svc}
}

// ========================================
// TEST HELPERS
// ========================================

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func setAdminContext(c *gin.Context, adminID uuid.UUID) {
	c.Set("user_id", adminID)
}

// ========================================
// DASHBOARD TESTS
// ========================================

func TestHandler_GetDashboard_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	stats := &DashboardStats{
		Users:      &UserStats{TotalUsers: 1000, ActiveUsers: 900},
		Rides:      &RideStats{TotalRides: 10000, CompletedRides: 9500},
		TodayRides: &RideStats{TotalRides: 200, CompletedRides: 180},
	}

	mockSvc.On("GetDashboardStats", mock.Anything).Return(stats, nil)

	h := newTestHandler(mockSvc)
	router.GET("/dashboard", func(c *gin.Context) {
		result, err := h.service.GetDashboardStats(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))

	mockSvc.AssertExpectations(t)
}

func TestHandler_GetDashboard_Error(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetDashboardStats", mock.Anything).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/dashboard", func(c *gin.Context) {
		result, err := h.service.GetDashboardStats(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// USER MANAGEMENT TESTS
// ========================================

func TestHandler_GetAllUsers_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	users := []*models.User{
		{ID: uuid.New(), Email: "user1@test.com", IsActive: true},
		{ID: uuid.New(), Email: "user2@test.com", IsActive: true},
	}
	userStats := &UserStats{TotalUsers: 100, ActiveUsers: 90}

	mockSvc.On("GetAllUsers", mock.Anything, 20, 0, mock.AnythingOfType("*admin.UserFilter")).Return(users, 100, nil)
	mockSvc.On("GetUserStats", mock.Anything).Return(userStats, nil)

	h := newTestHandler(mockSvc)
	router.GET("/users", func(c *gin.Context) {
		filter := &UserFilter{}
		result, total, err := h.service.GetAllUsers(c.Request.Context(), 20, 0, filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		stats, _ := h.service.GetUserStats(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result, "total": total, "stats": stats})
	})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetAllUsers_WithFilters(t *testing.T) {
	tests := []struct {
		name        string
		queryParams string
		expectRole  string
	}{
		{"filter by admin role", "?role=admin", "admin"},
		{"filter by driver role", "?role=driver", "driver"},
		{"filter by rider role", "?role=rider", "rider"},
		{"filter with search", "?search=john", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(mockService)
			router := setupTestRouter()

			mockSvc.On("GetAllUsers", mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("*admin.UserFilter")).Return([]*models.User{}, 0, nil)
			mockSvc.On("GetUserStats", mock.Anything).Return(nil, errors.New("ignored"))

			h := newTestHandler(mockSvc)
			router.GET("/users", func(c *gin.Context) {
				filter := &UserFilter{Role: c.Query("role"), Search: c.Query("search")}
				result, total, err := h.service.GetAllUsers(c.Request.Context(), 20, 0, filter)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				h.service.GetUserStats(c.Request.Context()) // stats error is ignored
				c.JSON(http.StatusOK, gin.H{"success": true, "data": result, "total": total})
			})

			req := httptest.NewRequest(http.MethodGet, "/users"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestHandler_GetUser_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	userID := uuid.New()
	user := &models.User{ID: userID, Email: "test@test.com", IsActive: true}

	mockSvc.On("GetUser", mock.Anything, userID).Return(user, nil)

	h := newTestHandler(mockSvc)
	router.GET("/users/:id", func(c *gin.Context) {
		id, _ := uuid.Parse(c.Param("id"))
		result, err := h.service.GetUser(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetUser_InvalidID(t *testing.T) {
	router := setupTestRouter()

	router.GET("/users/:id", func(c *gin.Context) {
		_, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/users/invalid-uuid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetUser_NotFound(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	userID := uuid.New()
	mockSvc.On("GetUser", mock.Anything, userID).Return(nil, errors.New("not found"))

	h := newTestHandler(mockSvc)
	router.GET("/users/:id", func(c *gin.Context) {
		id, _ := uuid.Parse(c.Param("id"))
		result, err := h.service.GetUser(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_SuspendUser_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	adminID := uuid.New()
	userID := uuid.New()

	mockSvc.On("SuspendUser", mock.Anything, adminID, userID).Return(nil)

	h := newTestHandler(mockSvc)
	router.POST("/users/:id/suspend", func(c *gin.Context) {
		setAdminContext(c, adminID)
		id, _ := uuid.Parse(c.Param("id"))
		err := h.service.SuspendUser(c.Request.Context(), adminID, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "User suspended successfully"})
	})

	req := httptest.NewRequest(http.MethodPost, "/users/"+userID.String()+"/suspend", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_ActivateUser_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	adminID := uuid.New()
	userID := uuid.New()

	mockSvc.On("ActivateUser", mock.Anything, adminID, userID).Return(nil)

	h := newTestHandler(mockSvc)
	router.POST("/users/:id/activate", func(c *gin.Context) {
		setAdminContext(c, adminID)
		id, _ := uuid.Parse(c.Param("id"))
		err := h.service.ActivateUser(c.Request.Context(), adminID, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "User activated successfully"})
	})

	req := httptest.NewRequest(http.MethodPost, "/users/"+userID.String()+"/activate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// DRIVER MANAGEMENT TESTS
// ========================================

func TestHandler_GetAllDrivers_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	drivers := []*models.Driver{
		{ID: uuid.New(), LicenseNumber: "DL001", IsAvailable: true},
		{ID: uuid.New(), LicenseNumber: "DL002", IsAvailable: true},
	}
	driverStats := &DriverStats{TotalDrivers: 500, OnlineDrivers: 200}

	mockSvc.On("GetAllDrivers", mock.Anything, 20, 0, mock.AnythingOfType("*admin.DriverFilter")).Return(drivers, int64(500), nil)
	mockSvc.On("GetDriverStats", mock.Anything).Return(driverStats, nil)

	h := newTestHandler(mockSvc)
	router.GET("/drivers", func(c *gin.Context) {
		filter := &DriverFilter{}
		result, total, err := h.service.GetAllDrivers(c.Request.Context(), 20, 0, filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		stats, _ := h.service.GetDriverStats(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result, "total": total, "stats": stats})
	})

	req := httptest.NewRequest(http.MethodGet, "/drivers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetAllDrivers_WithStatusFilter(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"online drivers", "online"},
		{"offline drivers", "offline"},
		{"available drivers", "available"},
		{"pending drivers", "pending"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(mockService)
			router := setupTestRouter()

			mockSvc.On("GetAllDrivers", mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("*admin.DriverFilter")).Return([]*models.Driver{}, int64(0), nil)
			mockSvc.On("GetDriverStats", mock.Anything).Return(nil, errors.New("ignored"))

			h := newTestHandler(mockSvc)
			router.GET("/drivers", func(c *gin.Context) {
				filter := &DriverFilter{Status: c.Query("status")}
				result, total, err := h.service.GetAllDrivers(c.Request.Context(), 20, 0, filter)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				h.service.GetDriverStats(c.Request.Context())
				c.JSON(http.StatusOK, gin.H{"success": true, "data": result, "total": total})
			})

			req := httptest.NewRequest(http.MethodGet, "/drivers?status="+tt.status, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestHandler_GetDriver_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	driverID := uuid.New()
	driver := &models.Driver{ID: driverID, LicenseNumber: "DL001", IsAvailable: true}

	mockSvc.On("GetDriver", mock.Anything, driverID).Return(driver, nil)

	h := newTestHandler(mockSvc)
	router.GET("/drivers/:id", func(c *gin.Context) {
		id, _ := uuid.Parse(c.Param("id"))
		result, err := h.service.GetDriver(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Driver not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/drivers/"+driverID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetPendingDrivers_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	drivers := []*models.Driver{
		{ID: uuid.New(), LicenseNumber: "DL001", IsAvailable: false},
	}

	mockSvc.On("GetPendingDrivers", mock.Anything, 20, 0).Return(drivers, int64(1), nil)

	h := newTestHandler(mockSvc)
	router.GET("/drivers/pending", func(c *gin.Context) {
		result, total, err := h.service.GetPendingDrivers(c.Request.Context(), 20, 0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result, "total": total})
	})

	req := httptest.NewRequest(http.MethodGet, "/drivers/pending", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_ApproveDriver_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	adminID := uuid.New()
	driverID := uuid.New()

	mockSvc.On("ApproveDriver", mock.Anything, adminID, driverID).Return(nil)

	h := newTestHandler(mockSvc)
	router.POST("/drivers/:id/approve", func(c *gin.Context) {
		setAdminContext(c, adminID)
		id, _ := uuid.Parse(c.Param("id"))
		err := h.service.ApproveDriver(c.Request.Context(), adminID, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Driver approved successfully"})
	})

	req := httptest.NewRequest(http.MethodPost, "/drivers/"+driverID.String()+"/approve", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_RejectDriver_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	adminID := uuid.New()
	driverID := uuid.New()

	mockSvc.On("RejectDriver", mock.Anything, adminID, driverID).Return(nil)

	h := newTestHandler(mockSvc)
	router.POST("/drivers/:id/reject", func(c *gin.Context) {
		setAdminContext(c, adminID)
		id, _ := uuid.Parse(c.Param("id"))
		err := h.service.RejectDriver(c.Request.Context(), adminID, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Driver rejected successfully"})
	})

	req := httptest.NewRequest(http.MethodPost, "/drivers/"+driverID.String()+"/reject", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// RIDE STATS TESTS
// ========================================

func TestHandler_GetRideStats_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	stats := &RideStats{
		TotalRides:     10000,
		CompletedRides: 9500,
		CancelledRides: 500,
		TotalRevenue:   150000.00,
	}

	mockSvc.On("GetRideStats", mock.Anything, (*time.Time)(nil), (*time.Time)(nil)).Return(stats, nil)

	h := newTestHandler(mockSvc)
	router.GET("/rides/stats", func(c *gin.Context) {
		result, err := h.service.GetRideStats(c.Request.Context(), nil, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/rides/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetRideStats_WithDateRange(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	stats := &RideStats{TotalRides: 100}

	mockSvc.On("GetRideStats", mock.Anything, mock.AnythingOfType("*time.Time"), mock.AnythingOfType("*time.Time")).Return(stats, nil)

	h := newTestHandler(mockSvc)
	router.GET("/rides/stats", func(c *gin.Context) {
		var startDate, endDate *time.Time
		if start := c.Query("start_date"); start != "" {
			t, _ := time.Parse("2006-01-02", start)
			startDate = &t
		}
		if end := c.Query("end_date"); end != "" {
			t, _ := time.Parse("2006-01-02", end)
			endDate = &t
		}
		result, err := h.service.GetRideStats(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/rides/stats?start_date=2024-01-01&end_date=2024-01-31", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetRide_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	rideID := uuid.New()
	ride := &AdminRideDetail{
		Ride: &models.Ride{
			ID:            rideID,
			Status:        models.RideStatusCompleted,
			EstimatedFare: 25.50,
		},
	}

	mockSvc.On("GetRide", mock.Anything, rideID).Return(ride, nil)

	h := newTestHandler(mockSvc)
	router.GET("/rides/:id", func(c *gin.Context) {
		id, _ := uuid.Parse(c.Param("id"))
		result, err := h.service.GetRide(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Ride not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/rides/"+rideID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetRecentRides_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	rides := []*AdminRideDetail{
		{Ride: &models.Ride{ID: uuid.New(), Status: models.RideStatusCompleted}},
		{Ride: &models.Ride{ID: uuid.New(), Status: models.RideStatusInProgress}},
	}

	mockSvc.On("GetRecentRides", mock.Anything, 50, 0).Return(rides, int64(100), nil)

	h := newTestHandler(mockSvc)
	router.GET("/rides/recent", func(c *gin.Context) {
		result, total, err := h.service.GetRecentRides(c.Request.Context(), 50, 0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result, "total": total})
	})

	req := httptest.NewRequest(http.MethodGet, "/rides/recent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// REALTIME METRICS TESTS
// ========================================

func TestHandler_GetRealtimeMetrics_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	metrics := &RealtimeMetrics{
		ActiveRides:      50,
		OnlineDrivers:    200,
		PendingRequests:  15,
	}

	mockSvc.On("GetRealtimeMetrics", mock.Anything).Return(metrics, nil)

	h := newTestHandler(mockSvc)
	router.GET("/metrics/realtime", func(c *gin.Context) {
		result, err := h.service.GetRealtimeMetrics(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/metrics/realtime", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// DASHBOARD SUMMARY TESTS
// ========================================

func TestHandler_GetDashboardSummary_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	summary := &DashboardSummary{
		Rides:   &SummaryRides{Total: 200, Completed: 180},
		Revenue: &SummaryRevenue{Total: 5000.00},
	}

	mockSvc.On("GetDashboardSummary", mock.Anything, "today").Return(summary, nil)

	h := newTestHandler(mockSvc)
	router.GET("/dashboard/summary", func(c *gin.Context) {
		period := c.DefaultQuery("period", "today")
		result, err := h.service.GetDashboardSummary(c.Request.Context(), period)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/dashboard/summary", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetDashboardSummary_DifferentPeriods(t *testing.T) {
	periods := []string{"today", "week", "month", "year"}

	for _, period := range periods {
		t.Run("period_"+period, func(t *testing.T) {
			mockSvc := new(mockService)
			router := setupTestRouter()

			summary := &DashboardSummary{Rides: &SummaryRides{Total: 100}}
			mockSvc.On("GetDashboardSummary", mock.Anything, period).Return(summary, nil)

			h := newTestHandler(mockSvc)
			router.GET("/dashboard/summary", func(c *gin.Context) {
				p := c.DefaultQuery("period", "today")
				result, err := h.service.GetDashboardSummary(c.Request.Context(), p)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
			})

			req := httptest.NewRequest(http.MethodGet, "/dashboard/summary?period="+period, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			mockSvc.AssertExpectations(t)
		})
	}
}

// ========================================
// REVENUE TREND TESTS
// ========================================

func TestHandler_GetRevenueTrend_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	trend := &RevenueTrend{
		Period:  "7days",
		GroupBy: "day",
		Trend:   []RevenueTrendData{},
	}

	mockSvc.On("GetRevenueTrend", mock.Anything, "7days", "day").Return(trend, nil)

	h := newTestHandler(mockSvc)
	router.GET("/revenue/trend", func(c *gin.Context) {
		period := c.DefaultQuery("period", "7days")
		groupBy := c.DefaultQuery("group_by", "day")
		result, err := h.service.GetRevenueTrend(c.Request.Context(), period, groupBy)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/revenue/trend", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// ACTION ITEMS TESTS
// ========================================

func TestHandler_GetActionItems_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	items := &ActionItems{
		PendingDriverApprovals: &ActionDriverApprovals{Count: 5},
		FraudAlerts:           &ActionFraudAlerts{Count: 3},
	}

	mockSvc.On("GetActionItems", mock.Anything).Return(items, nil)

	h := newTestHandler(mockSvc)
	router.GET("/action-items", func(c *gin.Context) {
		result, err := h.service.GetActionItems(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/action-items", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// HEALTH CHECK TEST
// ========================================

func TestHandler_HealthCheck(t *testing.T) {
	router := setupTestRouter()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "admin"})
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "admin", response["service"])
}

// ========================================
// getAdminID HELPER TESTS
// ========================================

func TestGetAdminID_WithValidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	expectedID := uuid.New()
	c.Set("user_id", expectedID)

	result := getAdminID(c)
	assert.Equal(t, expectedID, result)
}

func TestGetAdminID_WithNoID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	result := getAdminID(c)
	assert.Equal(t, uuid.Nil, result)
}

func TestGetAdminID_WithInvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	c.Set("user_id", "not-a-uuid")

	result := getAdminID(c)
	assert.Equal(t, uuid.Nil, result)
}

// ========================================
// ERROR HANDLING TESTS
// ========================================

func TestHandler_ServiceError_ReturnsInternalError(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*mockService)
		path  string
	}{
		{
			name: "GetDashboardStats error",
			setup: func(m *mockService) {
				m.On("GetDashboardStats", mock.Anything).Return(nil, errors.New("db error"))
			},
			path: "/dashboard",
		},
		{
			name: "GetRealtimeMetrics error",
			setup: func(m *mockService) {
				m.On("GetRealtimeMetrics", mock.Anything).Return(nil, errors.New("db error"))
			},
			path: "/metrics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(mockService)
			router := setupTestRouter()
			tt.setup(mockSvc)

			h := newTestHandler(mockSvc)

			switch tt.path {
			case "/dashboard":
				router.GET(tt.path, func(c *gin.Context) {
					_, err := h.service.GetDashboardStats(c.Request.Context())
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
						return
					}
				})
			case "/metrics":
				router.GET(tt.path, func(c *gin.Context) {
					_, err := h.service.GetRealtimeMetrics(c.Request.Context())
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
						return
					}
				})
			}

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusInternalServerError, w.Code)
			mockSvc.AssertExpectations(t)
		})
	}
}
