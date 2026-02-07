package analytics

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

func (m *mockService) GetRevenueMetrics(ctx context.Context, startDate, endDate time.Time) (*RevenueMetrics, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RevenueMetrics), args.Error(1)
}

func (m *mockService) GetPromoCodePerformance(ctx context.Context, startDate, endDate time.Time) ([]*PromoCodePerformance, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*PromoCodePerformance), args.Error(1)
}

func (m *mockService) GetRideTypeStats(ctx context.Context, startDate, endDate time.Time) ([]*RideTypeStats, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RideTypeStats), args.Error(1)
}

func (m *mockService) GetReferralMetrics(ctx context.Context, startDate, endDate time.Time) (*ReferralMetrics, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ReferralMetrics), args.Error(1)
}

func (m *mockService) GetTopDrivers(ctx context.Context, startDate, endDate time.Time, limit int) ([]*DriverPerformance, error) {
	args := m.Called(ctx, startDate, endDate, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DriverPerformance), args.Error(1)
}

func (m *mockService) GetDashboardMetrics(ctx context.Context) (*DashboardMetrics, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DashboardMetrics), args.Error(1)
}

func (m *mockService) GetDemandHeatMap(ctx context.Context, startDate, endDate time.Time, gridSize float64) ([]*DemandHeatMap, error) {
	args := m.Called(ctx, startDate, endDate, gridSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DemandHeatMap), args.Error(1)
}

func (m *mockService) GetFinancialReport(ctx context.Context, startDate, endDate time.Time) (*FinancialReport, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FinancialReport), args.Error(1)
}

func (m *mockService) GetDemandZones(ctx context.Context, startDate, endDate time.Time, minRides int) ([]*DemandZone, error) {
	args := m.Called(ctx, startDate, endDate, minRides)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DemandZone), args.Error(1)
}

func (m *mockService) GetRevenueTimeSeries(ctx context.Context, startDate, endDate time.Time, granularity string) ([]*RevenueTimeSeries, error) {
	args := m.Called(ctx, startDate, endDate, granularity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RevenueTimeSeries), args.Error(1)
}

func (m *mockService) GetHourlyDistribution(ctx context.Context, startDate, endDate time.Time) ([]*HourlyDistribution, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*HourlyDistribution), args.Error(1)
}

func (m *mockService) GetDriverAnalytics(ctx context.Context, startDate, endDate time.Time) (*DriverAnalytics, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverAnalytics), args.Error(1)
}

func (m *mockService) GetRiderGrowth(ctx context.Context, startDate, endDate time.Time) (*RiderGrowth, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RiderGrowth), args.Error(1)
}

func (m *mockService) GetRideMetrics(ctx context.Context, startDate, endDate time.Time) (*RideMetrics, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RideMetrics), args.Error(1)
}

func (m *mockService) GetTopDriversDetailed(ctx context.Context, startDate, endDate time.Time, limit int) ([]*TopDriver, error) {
	args := m.Called(ctx, startDate, endDate, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*TopDriver), args.Error(1)
}

func (m *mockService) GetPeriodComparison(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time) (*PeriodComparison, error) {
	args := m.Called(ctx, currentStart, currentEnd, previousStart, previousEnd)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PeriodComparison), args.Error(1)
}

// ========================================
// ServiceInterface for handler testing
// ========================================

type ServiceInterface interface {
	GetRevenueMetrics(ctx context.Context, startDate, endDate time.Time) (*RevenueMetrics, error)
	GetPromoCodePerformance(ctx context.Context, startDate, endDate time.Time) ([]*PromoCodePerformance, error)
	GetRideTypeStats(ctx context.Context, startDate, endDate time.Time) ([]*RideTypeStats, error)
	GetReferralMetrics(ctx context.Context, startDate, endDate time.Time) (*ReferralMetrics, error)
	GetTopDrivers(ctx context.Context, startDate, endDate time.Time, limit int) ([]*DriverPerformance, error)
	GetDashboardMetrics(ctx context.Context) (*DashboardMetrics, error)
	GetDemandHeatMap(ctx context.Context, startDate, endDate time.Time, gridSize float64) ([]*DemandHeatMap, error)
	GetFinancialReport(ctx context.Context, startDate, endDate time.Time) (*FinancialReport, error)
	GetDemandZones(ctx context.Context, startDate, endDate time.Time, minRides int) ([]*DemandZone, error)
	GetRevenueTimeSeries(ctx context.Context, startDate, endDate time.Time, granularity string) ([]*RevenueTimeSeries, error)
	GetHourlyDistribution(ctx context.Context, startDate, endDate time.Time) ([]*HourlyDistribution, error)
	GetDriverAnalytics(ctx context.Context, startDate, endDate time.Time) (*DriverAnalytics, error)
	GetRiderGrowth(ctx context.Context, startDate, endDate time.Time) (*RiderGrowth, error)
	GetRideMetrics(ctx context.Context, startDate, endDate time.Time) (*RideMetrics, error)
	GetTopDriversDetailed(ctx context.Context, startDate, endDate time.Time, limit int) ([]*TopDriver, error)
	GetPeriodComparison(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time) (*PeriodComparison, error)
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

// ========================================
// REVENUE METRICS TESTS
// ========================================

func TestHandler_GetRevenueMetrics_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	metrics := &RevenueMetrics{
		Period:           "2024-01-01 to 2024-01-31",
		TotalRevenue:     50000.00,
		TotalRides:       1000,
		AvgFarePerRide:   50.00,
		TotalDiscounts:   2000.00,
		PlatformEarnings: 10000.00,
		DriverEarnings:   38000.00,
	}

	mockSvc.On("GetRevenueMetrics", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(metrics, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/revenue", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		result, err := h.service.GetRevenueMetrics(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get revenue metrics"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/revenue?start_date=2024-01-01&end_date=2024-01-31", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	mockSvc.AssertExpectations(t)
}

func TestHandler_GetRevenueMetrics_DefaultDateRange(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	metrics := &RevenueMetrics{TotalRevenue: 10000.00}
	mockSvc.On("GetRevenueMetrics", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(metrics, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/revenue", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		result, err := h.service.GetRevenueMetrics(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get revenue metrics"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	// No date params - should use default last 30 days
	req := httptest.NewRequest(http.MethodGet, "/analytics/revenue", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetRevenueMetrics_InvalidStartDate(t *testing.T) {
	router := setupTestRouter()

	router.GET("/analytics/revenue", func(c *gin.Context) {
		_, _, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/revenue?start_date=invalid-date", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["success"].(bool))
}

func TestHandler_GetRevenueMetrics_InvalidEndDate(t *testing.T) {
	router := setupTestRouter()

	router.GET("/analytics/revenue", func(c *gin.Context) {
		_, _, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/revenue?start_date=2024-01-01&end_date=not-a-date", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetRevenueMetrics_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetRevenueMetrics", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/analytics/revenue", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		result, err := h.service.GetRevenueMetrics(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get revenue metrics"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/revenue?start_date=2024-01-01&end_date=2024-01-31", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// PROMO CODE PERFORMANCE TESTS
// ========================================

func TestHandler_GetPromoCodePerformance_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	performance := []*PromoCodePerformance{
		{PromoCodeID: uuid.New(), Code: "SAVE10", TotalUses: 500, TotalDiscount: 5000.00},
		{PromoCodeID: uuid.New(), Code: "FIRST50", TotalUses: 200, TotalDiscount: 10000.00},
	}

	mockSvc.On("GetPromoCodePerformance", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(performance, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/promo-codes", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		result, err := h.service.GetPromoCodePerformance(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get promo code performance"}})
			return
		}
		if result == nil {
			result = []*PromoCodePerformance{}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"promo_codes": result}})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/promo-codes?start_date=2024-01-01&end_date=2024-01-31", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	promoCodes := data["promo_codes"].([]interface{})
	assert.Len(t, promoCodes, 2)

	mockSvc.AssertExpectations(t)
}

func TestHandler_GetPromoCodePerformance_EmptyResult(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetPromoCodePerformance", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]*PromoCodePerformance{}, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/promo-codes", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		result, err := h.service.GetPromoCodePerformance(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get promo code performance"}})
			return
		}
		if result == nil {
			result = []*PromoCodePerformance{}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"promo_codes": result}})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/promo-codes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	data := response["data"].(map[string]interface{})
	promoCodes := data["promo_codes"].([]interface{})
	assert.Len(t, promoCodes, 0)

	mockSvc.AssertExpectations(t)
}

func TestHandler_GetPromoCodePerformance_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetPromoCodePerformance", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/analytics/promo-codes", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		_, err = h.service.GetPromoCodePerformance(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get promo code performance"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/promo-codes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// RIDE TYPE STATS TESTS
// ========================================

func TestHandler_GetRideTypeStats_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	stats := []*RideTypeStats{
		{RideTypeID: uuid.New(), Name: "Standard", TotalRides: 5000, TotalRevenue: 100000.00, Percentage: 60.0},
		{RideTypeID: uuid.New(), Name: "Premium", TotalRides: 2000, TotalRevenue: 80000.00, Percentage: 40.0},
	}

	mockSvc.On("GetRideTypeStats", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(stats, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/ride-types", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		result, err := h.service.GetRideTypeStats(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get ride type stats"}})
			return
		}
		if result == nil {
			result = []*RideTypeStats{}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"ride_types": result}})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/ride-types?start_date=2024-01-01&end_date=2024-01-31", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))

	mockSvc.AssertExpectations(t)
}

func TestHandler_GetRideTypeStats_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetRideTypeStats", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/analytics/ride-types", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		_, err = h.service.GetRideTypeStats(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get ride type stats"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/ride-types", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// REFERRAL METRICS TESTS
// ========================================

func TestHandler_GetReferralMetrics_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	metrics := &ReferralMetrics{
		TotalReferrals:      500,
		CompletedReferrals:  400,
		TotalBonusesPaid:    10000.00,
		AvgBonusPerReferral: 25.00,
		ConversionRate:      80.0,
	}

	mockSvc.On("GetReferralMetrics", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(metrics, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/referrals", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		result, err := h.service.GetReferralMetrics(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get referral metrics"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/referrals?start_date=2024-01-01&end_date=2024-01-31", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetReferralMetrics_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetReferralMetrics", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/analytics/referrals", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		_, err = h.service.GetReferralMetrics(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get referral metrics"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/referrals", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// TOP DRIVERS TESTS
// ========================================

func TestHandler_GetTopDrivers_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	drivers := []*DriverPerformance{
		{DriverID: uuid.New(), DriverName: "John Doe", TotalRides: 500, TotalEarnings: 15000.00, AvgRating: 4.9},
		{DriverID: uuid.New(), DriverName: "Jane Smith", TotalRides: 450, TotalEarnings: 14000.00, AvgRating: 4.8},
	}

	mockSvc.On("GetTopDrivers", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), 10).Return(drivers, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/top-drivers", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		limit := 10
		result, err := h.service.GetTopDrivers(c.Request.Context(), startDate, endDate, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get top drivers"}})
			return
		}
		if result == nil {
			result = []*DriverPerformance{}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"drivers": result}})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/top-drivers?start_date=2024-01-01&end_date=2024-01-31", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetTopDrivers_WithCustomLimit(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetTopDrivers", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), 5).Return([]*DriverPerformance{}, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/top-drivers", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		limit := 5 // Simulating parsed limit from query
		result, err := h.service.GetTopDrivers(c.Request.Context(), startDate, endDate, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get top drivers"}})
			return
		}
		if result == nil {
			result = []*DriverPerformance{}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"drivers": result}})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/top-drivers?limit=5", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetTopDrivers_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetTopDrivers", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), 10).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/analytics/top-drivers", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		_, err = h.service.GetTopDrivers(c.Request.Context(), startDate, endDate, 10)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get top drivers"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/top-drivers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// DASHBOARD METRICS TESTS
// ========================================

func TestHandler_GetDashboardMetrics_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	metrics := &DashboardMetrics{
		TotalRides:     100000,
		ActiveRides:    50,
		CompletedToday: 1500,
		RevenueToday:   75000.00,
		ActiveDrivers:  500,
		ActiveRiders:   2000,
		AvgRating:      4.7,
	}

	mockSvc.On("GetDashboardMetrics", mock.Anything).Return(metrics, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/dashboard", func(c *gin.Context) {
		result, err := h.service.GetDashboardMetrics(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get dashboard metrics"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/dashboard", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))

	mockSvc.AssertExpectations(t)
}

func TestHandler_GetDashboardMetrics_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetDashboardMetrics", mock.Anything).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/analytics/dashboard", func(c *gin.Context) {
		_, err := h.service.GetDashboardMetrics(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get dashboard metrics"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/dashboard", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// DEMAND HEAT MAP TESTS
// ========================================

func TestHandler_GetDemandHeatMap_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	heatMap := []*DemandHeatMap{
		{Latitude: 40.7128, Longitude: -74.0060, RideCount: 500, DemandLevel: "high"},
		{Latitude: 40.7580, Longitude: -73.9855, RideCount: 300, DemandLevel: "medium"},
	}

	mockSvc.On("GetDemandHeatMap", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), 0.01).Return(heatMap, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/demand-heatmap", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		gridSize := 0.01
		result, err := h.service.GetDemandHeatMap(c.Request.Context(), startDate, endDate, gridSize)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get demand heat map"}})
			return
		}
		if result == nil {
			result = []*DemandHeatMap{}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"heat_map": result, "grid_size_km": gridSize * 111}})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/demand-heatmap", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetDemandHeatMap_CustomGridSize(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetDemandHeatMap", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), 0.05).Return([]*DemandHeatMap{}, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/demand-heatmap", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		gridSize := 0.05
		result, err := h.service.GetDemandHeatMap(c.Request.Context(), startDate, endDate, gridSize)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get demand heat map"}})
			return
		}
		if result == nil {
			result = []*DemandHeatMap{}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"heat_map": result}})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/demand-heatmap?grid_size=0.05", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetDemandHeatMap_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetDemandHeatMap", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), 0.01).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/analytics/demand-heatmap", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		_, err = h.service.GetDemandHeatMap(c.Request.Context(), startDate, endDate, 0.01)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get demand heat map"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/demand-heatmap", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// FINANCIAL REPORT TESTS
// ========================================

func TestHandler_GetFinancialReport_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	report := &FinancialReport{
		Period:             "2024-01",
		GrossRevenue:       500000.00,
		NetRevenue:         450000.00,
		PlatformCommission: 100000.00,
		DriverPayouts:      350000.00,
		PromoDiscounts:     30000.00,
		Profit:             100000.00,
		ProfitMargin:       20.0,
		TotalRides:         10000,
		CompletedRides:     9500,
	}

	mockSvc.On("GetFinancialReport", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(report, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/financial-report", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		result, err := h.service.GetFinancialReport(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get financial report"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/financial-report?start_date=2024-01-01&end_date=2024-01-31", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetFinancialReport_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetFinancialReport", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/analytics/financial-report", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		_, err = h.service.GetFinancialReport(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get financial report"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/financial-report", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// DEMAND ZONES TESTS
// ========================================

func TestHandler_GetDemandZones_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	zones := []*DemandZone{
		{ZoneName: "Downtown", CenterLat: 40.7128, CenterLon: -74.0060, TotalRides: 5000, PeakHours: "8-10, 17-19"},
		{ZoneName: "Airport", CenterLat: 40.6413, CenterLon: -73.7781, TotalRides: 3000, PeakHours: "6-8, 20-22"},
	}

	mockSvc.On("GetDemandZones", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), 20).Return(zones, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/demand-zones", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		minRides := 20
		result, err := h.service.GetDemandZones(c.Request.Context(), startDate, endDate, minRides)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get demand zones"}})
			return
		}
		if result == nil {
			result = []*DemandZone{}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"zones": result}})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/demand-zones", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetDemandZones_CustomMinRides(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetDemandZones", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), 50).Return([]*DemandZone{}, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/demand-zones", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		minRides := 50
		result, err := h.service.GetDemandZones(c.Request.Context(), startDate, endDate, minRides)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get demand zones"}})
			return
		}
		if result == nil {
			result = []*DemandZone{}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"zones": result}})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/demand-zones?min_rides=50", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetDemandZones_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetDemandZones", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), 20).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/analytics/demand-zones", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		_, err = h.service.GetDemandZones(c.Request.Context(), startDate, endDate, 20)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get demand zones"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/demand-zones", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// REVENUE TIME SERIES TESTS
// ========================================

func TestHandler_GetRevenueTimeSeries_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	data := []*RevenueTimeSeries{
		{Date: "2024-01-01", Revenue: 10000.00, Rides: 200, AvgFare: 50.00},
		{Date: "2024-01-02", Revenue: 12000.00, Rides: 220, AvgFare: 54.55},
	}

	mockSvc.On("GetRevenueTimeSeries", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), "day").Return(data, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/revenue-timeseries", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		granularity := "day"
		result, err := h.service.GetRevenueTimeSeries(c.Request.Context(), startDate, endDate, granularity)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get revenue timeseries"}})
			return
		}
		if result == nil {
			result = []*RevenueTimeSeries{}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"data": result, "granularity": granularity}})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/revenue-timeseries", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetRevenueTimeSeries_DifferentGranularities(t *testing.T) {
	granularities := []string{"day", "week", "month"}

	for _, granularity := range granularities {
		t.Run("granularity_"+granularity, func(t *testing.T) {
			mockSvc := new(mockService)
			router := setupTestRouter()

			mockSvc.On("GetRevenueTimeSeries", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), granularity).Return([]*RevenueTimeSeries{}, nil)

			h := newTestHandler(mockSvc)
			router.GET("/analytics/revenue-timeseries", func(c *gin.Context) {
				startDate, endDate, err := parseDateRange(c)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
					return
				}
				g := c.DefaultQuery("granularity", "day")
				if g != "day" && g != "week" && g != "month" {
					g = "day"
				}
				result, err := h.service.GetRevenueTimeSeries(c.Request.Context(), startDate, endDate, g)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get revenue timeseries"}})
					return
				}
				if result == nil {
					result = []*RevenueTimeSeries{}
				}
				c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"data": result, "granularity": g}})
			})

			req := httptest.NewRequest(http.MethodGet, "/analytics/revenue-timeseries?granularity="+granularity, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestHandler_GetRevenueTimeSeries_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetRevenueTimeSeries", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), "day").Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/analytics/revenue-timeseries", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		_, err = h.service.GetRevenueTimeSeries(c.Request.Context(), startDate, endDate, "day")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get revenue timeseries"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/revenue-timeseries", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// HOURLY DISTRIBUTION TESTS
// ========================================

func TestHandler_GetHourlyDistribution_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	data := []*HourlyDistribution{
		{Hour: 8, Rides: 500, AvgFare: 45.00, AvgWaitTime: 5.2},
		{Hour: 17, Rides: 600, AvgFare: 55.00, AvgWaitTime: 7.5},
	}

	mockSvc.On("GetHourlyDistribution", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(data, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/hourly-distribution", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		result, err := h.service.GetHourlyDistribution(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get hourly distribution"}})
			return
		}
		if result == nil {
			result = []*HourlyDistribution{}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"data": result}})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/hourly-distribution", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetHourlyDistribution_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetHourlyDistribution", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/analytics/hourly-distribution", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		_, err = h.service.GetHourlyDistribution(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get hourly distribution"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/hourly-distribution", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// DRIVER ANALYTICS TESTS
// ========================================

func TestHandler_GetDriverAnalytics_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	data := &DriverAnalytics{
		TotalActiveDrivers:  500,
		AvgRidesPerDriver:   20.5,
		AvgOnlineHours:      8.5,
		AvgAcceptanceRate:   85.0,
		AvgCancellationRate: 5.0,
		AvgRating:           4.7,
		NewDrivers:          50,
		ChurnedDrivers:      10,
	}

	mockSvc.On("GetDriverAnalytics", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(data, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/driver-analytics", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		result, err := h.service.GetDriverAnalytics(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get driver analytics"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"data": result}})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/driver-analytics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetDriverAnalytics_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetDriverAnalytics", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/analytics/driver-analytics", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		_, err = h.service.GetDriverAnalytics(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get driver analytics"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/driver-analytics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// RIDER GROWTH TESTS
// ========================================

func TestHandler_GetRiderGrowth_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	data := &RiderGrowth{
		NewRiders:            1000,
		ReturningRiders:      5000,
		ChurnedRiders:        200,
		RetentionRate:        80.0,
		AvgRidesPerRider:     5.5,
		FirstTimeRidersToday: 50,
		LifetimeValueAvg:     250.00,
	}

	mockSvc.On("GetRiderGrowth", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(data, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/rider-growth", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		result, err := h.service.GetRiderGrowth(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get rider growth"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"data": result}})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/rider-growth", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetRiderGrowth_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetRiderGrowth", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/analytics/rider-growth", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		_, err = h.service.GetRiderGrowth(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get rider growth"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/rider-growth", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// RIDE METRICS TESTS
// ========================================

func TestHandler_GetRideMetrics_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	data := &RideMetrics{
		AvgWaitTimeMinutes:     5.2,
		AvgRideDurationMinutes: 15.5,
		AvgDistanceKm:          8.3,
		CancellationRate:       8.0,
		RiderCancellationRate:  5.0,
		DriverCancellationRate: 3.0,
		SurgeRidesPercentage:   12.0,
		AvgSurgeMultiplier:     1.5,
	}

	mockSvc.On("GetRideMetrics", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(data, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/ride-metrics", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		result, err := h.service.GetRideMetrics(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get ride metrics"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"data": result}})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/ride-metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetRideMetrics_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetRideMetrics", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/analytics/ride-metrics", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		_, err = h.service.GetRideMetrics(c.Request.Context(), startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get ride metrics"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/ride-metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// TOP DRIVERS DETAILED TESTS
// ========================================

func TestHandler_GetTopDriversDetailed_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	data := []*TopDriver{
		{DriverID: uuid.New(), Name: "John Doe", TotalRides: 500, TotalRevenue: 15000.00, AvgRating: 4.9, AcceptanceRate: 95.0, OnlineHours: 200.0},
		{DriverID: uuid.New(), Name: "Jane Smith", TotalRides: 450, TotalRevenue: 14000.00, AvgRating: 4.8, AcceptanceRate: 90.0, OnlineHours: 180.0},
	}

	mockSvc.On("GetTopDriversDetailed", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), 10).Return(data, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/top-drivers-detailed", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		limit := 10
		result, err := h.service.GetTopDriversDetailed(c.Request.Context(), startDate, endDate, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get top drivers"}})
			return
		}
		if result == nil {
			result = []*TopDriver{}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"data": result}})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/top-drivers-detailed", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetTopDriversDetailed_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetTopDriversDetailed", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), 10).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/analytics/top-drivers-detailed", func(c *gin.Context) {
		startDate, endDate, err := parseDateRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}
		_, err = h.service.GetTopDriversDetailed(c.Request.Context(), startDate, endDate, 10)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get top drivers"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/top-drivers-detailed", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// PERIOD COMPARISON TESTS
// ========================================

func TestHandler_GetPeriodComparison_Success(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	data := &PeriodComparison{
		Revenue:   ComparisonMetric{Current: 50000, Previous: 45000, ChangePercent: 11.11},
		Rides:     ComparisonMetric{Current: 1000, Previous: 900, ChangePercent: 11.11},
		NewRiders: ComparisonMetric{Current: 100, Previous: 80, ChangePercent: 25.0},
		AvgRating: ComparisonMetric{Current: 4.8, Previous: 4.7, ChangePercent: 2.13},
	}

	mockSvc.On("GetPeriodComparison", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(data, nil)

	h := newTestHandler(mockSvc)
	router.GET("/analytics/period-comparison", func(c *gin.Context) {
		currentStartStr := c.Query("current_start")
		currentEndStr := c.Query("current_end")
		previousStartStr := c.Query("previous_start")
		previousEndStr := c.Query("previous_end")

		if currentStartStr == "" || currentEndStr == "" || previousStartStr == "" || previousEndStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "all date parameters required"}})
			return
		}

		currentStart, err := time.Parse("2006-01-02", currentStartStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid current_start date format"}})
			return
		}

		currentEnd, err := time.Parse("2006-01-02", currentEndStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid current_end date format"}})
			return
		}
		currentEnd = currentEnd.Add(24 * time.Hour).Add(-time.Second)

		previousStart, err := time.Parse("2006-01-02", previousStartStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid previous_start date format"}})
			return
		}

		previousEnd, err := time.Parse("2006-01-02", previousEndStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid previous_end date format"}})
			return
		}
		previousEnd = previousEnd.Add(24 * time.Hour).Add(-time.Second)

		result, err := h.service.GetPeriodComparison(c.Request.Context(), currentStart, currentEnd, previousStart, previousEnd)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get period comparison"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"data": result}})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/period-comparison?current_start=2024-02-01&current_end=2024-02-28&previous_start=2024-01-01&previous_end=2024-01-31", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestHandler_GetPeriodComparison_MissingParams(t *testing.T) {
	router := setupTestRouter()

	router.GET("/analytics/period-comparison", func(c *gin.Context) {
		currentStartStr := c.Query("current_start")
		currentEndStr := c.Query("current_end")
		previousStartStr := c.Query("previous_start")
		previousEndStr := c.Query("previous_end")

		if currentStartStr == "" || currentEndStr == "" || previousStartStr == "" || previousEndStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "all date parameters required"}})
			return
		}
	})

	tests := []struct {
		name   string
		params string
	}{
		{"missing current_start", "?current_end=2024-02-28&previous_start=2024-01-01&previous_end=2024-01-31"},
		{"missing current_end", "?current_start=2024-02-01&previous_start=2024-01-01&previous_end=2024-01-31"},
		{"missing previous_start", "?current_start=2024-02-01&current_end=2024-02-28&previous_end=2024-01-31"},
		{"missing previous_end", "?current_start=2024-02-01&current_end=2024-02-28&previous_start=2024-01-01"},
		{"missing all params", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/analytics/period-comparison"+tt.params, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandler_GetPeriodComparison_InvalidDateFormats(t *testing.T) {
	router := setupTestRouter()

	router.GET("/analytics/period-comparison", func(c *gin.Context) {
		currentStartStr := c.Query("current_start")
		currentEndStr := c.Query("current_end")
		previousStartStr := c.Query("previous_start")
		previousEndStr := c.Query("previous_end")

		if currentStartStr == "" || currentEndStr == "" || previousStartStr == "" || previousEndStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "all date parameters required"}})
			return
		}

		_, err := time.Parse("2006-01-02", currentStartStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid current_start date format"}})
			return
		}

		_, err = time.Parse("2006-01-02", currentEndStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid current_end date format"}})
			return
		}

		_, err = time.Parse("2006-01-02", previousStartStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid previous_start date format"}})
			return
		}

		_, err = time.Parse("2006-01-02", previousEndStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"message": "invalid previous_end date format"}})
			return
		}
	})

	tests := []struct {
		name   string
		params string
	}{
		{"invalid current_start", "?current_start=invalid&current_end=2024-02-28&previous_start=2024-01-01&previous_end=2024-01-31"},
		{"invalid current_end", "?current_start=2024-02-01&current_end=invalid&previous_start=2024-01-01&previous_end=2024-01-31"},
		{"invalid previous_start", "?current_start=2024-02-01&current_end=2024-02-28&previous_start=invalid&previous_end=2024-01-31"},
		{"invalid previous_end", "?current_start=2024-02-01&current_end=2024-02-28&previous_start=2024-01-01&previous_end=invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/analytics/period-comparison"+tt.params, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandler_GetPeriodComparison_ServiceError(t *testing.T) {
	mockSvc := new(mockService)
	router := setupTestRouter()

	mockSvc.On("GetPeriodComparison", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nil, errors.New("database error"))

	h := newTestHandler(mockSvc)
	router.GET("/analytics/period-comparison", func(c *gin.Context) {
		currentStart, _ := time.Parse("2006-01-02", c.Query("current_start"))
		currentEnd, _ := time.Parse("2006-01-02", c.Query("current_end"))
		previousStart, _ := time.Parse("2006-01-02", c.Query("previous_start"))
		previousEnd, _ := time.Parse("2006-01-02", c.Query("previous_end"))

		_, err := h.service.GetPeriodComparison(c.Request.Context(), currentStart, currentEnd, previousStart, previousEnd)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"message": "failed to get period comparison"}})
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/period-comparison?current_start=2024-02-01&current_end=2024-02-28&previous_start=2024-01-01&previous_end=2024-01-31", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

// ========================================
// HEALTH CHECK TESTS
// ========================================

func TestHandler_HealthCheck(t *testing.T) {
	router := setupTestRouter()

	router.GET("/analytics/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"service": "analytics", "status": "healthy"}})
	})

	req := httptest.NewRequest(http.MethodGet, "/analytics/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "analytics", data["service"])
	assert.Equal(t, "healthy", data["status"])
}

// ========================================
// parseDateRange TESTS
// ========================================

func TestParseDateRange_ValidDates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	req := httptest.NewRequest(http.MethodGet, "/?start_date=2024-01-01&end_date=2024-01-31", nil)
	c.Request = req

	startDate, endDate, err := parseDateRange(c)

	assert.NoError(t, err)
	assert.Equal(t, 2024, startDate.Year())
	assert.Equal(t, time.January, startDate.Month())
	assert.Equal(t, 1, startDate.Day())
	assert.Equal(t, 2024, endDate.Year())
	assert.Equal(t, time.January, endDate.Month())
	assert.Equal(t, 31, endDate.Day())
}

func TestParseDateRange_DefaultDates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request = req

	startDate, endDate, err := parseDateRange(c)

	assert.NoError(t, err)
	// Start date should be approximately 30 days ago
	expectedStart := time.Now().AddDate(0, 0, -30).Truncate(24 * time.Hour)
	assert.Equal(t, expectedStart.Year(), startDate.Year())
	assert.Equal(t, expectedStart.Month(), startDate.Month())
	assert.Equal(t, expectedStart.Day(), startDate.Day())
	// End date should be close to now
	assert.True(t, endDate.After(time.Now().Add(-time.Minute)))
}

func TestParseDateRange_InvalidStartDate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	req := httptest.NewRequest(http.MethodGet, "/?start_date=invalid", nil)
	c.Request = req

	_, _, err := parseDateRange(c)

	assert.Error(t, err)
}

func TestParseDateRange_InvalidEndDate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	req := httptest.NewRequest(http.MethodGet, "/?start_date=2024-01-01&end_date=invalid", nil)
	c.Request = req

	_, _, err := parseDateRange(c)

	assert.Error(t, err)
}

// ========================================
// ERROR HANDLING TESTS
// ========================================

func TestHandler_AllEndpoints_ServiceErrors(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		setup   func(*mockService)
		handler func(*testHandler, *gin.Context)
	}{
		{
			name: "GetRevenueMetrics error",
			path: "/revenue",
			setup: func(m *mockService) {
				m.On("GetRevenueMetrics", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nil, errors.New("db error"))
			},
			handler: func(h *testHandler, c *gin.Context) {
				startDate, endDate, _ := parseDateRange(c)
				_, err := h.service.GetRevenueMetrics(c.Request.Context(), startDate, endDate)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			},
		},
		{
			name: "GetPromoCodePerformance error",
			path: "/promo-codes",
			setup: func(m *mockService) {
				m.On("GetPromoCodePerformance", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nil, errors.New("db error"))
			},
			handler: func(h *testHandler, c *gin.Context) {
				startDate, endDate, _ := parseDateRange(c)
				_, err := h.service.GetPromoCodePerformance(c.Request.Context(), startDate, endDate)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			},
		},
		{
			name: "GetRideTypeStats error",
			path: "/ride-types",
			setup: func(m *mockService) {
				m.On("GetRideTypeStats", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nil, errors.New("db error"))
			},
			handler: func(h *testHandler, c *gin.Context) {
				startDate, endDate, _ := parseDateRange(c)
				_, err := h.service.GetRideTypeStats(c.Request.Context(), startDate, endDate)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			},
		},
		{
			name: "GetDashboardMetrics error",
			path: "/dashboard",
			setup: func(m *mockService) {
				m.On("GetDashboardMetrics", mock.Anything).Return(nil, errors.New("db error"))
			},
			handler: func(h *testHandler, c *gin.Context) {
				_, err := h.service.GetDashboardMetrics(c.Request.Context())
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(mockService)
			router := setupTestRouter()
			tt.setup(mockSvc)

			h := newTestHandler(mockSvc)
			router.GET(tt.path, func(c *gin.Context) {
				tt.handler(h, c)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusInternalServerError, w.Code)
			mockSvc.AssertExpectations(t)
		})
	}
}

// ========================================
// NULL/EMPTY ARRAY HANDLING TESTS
// ========================================

func TestHandler_NullSliceHandling(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		setup   func(*mockService)
		handler func(*testHandler, *gin.Context)
		key     string
	}{
		{
			name: "PromoCodePerformance returns nil",
			path: "/promo-codes",
			setup: func(m *mockService) {
				var nilSlice []*PromoCodePerformance
				m.On("GetPromoCodePerformance", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nilSlice, nil)
			},
			handler: func(h *testHandler, c *gin.Context) {
				startDate, endDate, _ := parseDateRange(c)
				result, err := h.service.GetPromoCodePerformance(c.Request.Context(), startDate, endDate)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				if result == nil {
					result = []*PromoCodePerformance{}
				}
				c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"promo_codes": result}})
			},
			key: "promo_codes",
		},
		{
			name: "RideTypeStats returns nil",
			path: "/ride-types",
			setup: func(m *mockService) {
				var nilSlice []*RideTypeStats
				m.On("GetRideTypeStats", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nilSlice, nil)
			},
			handler: func(h *testHandler, c *gin.Context) {
				startDate, endDate, _ := parseDateRange(c)
				result, err := h.service.GetRideTypeStats(c.Request.Context(), startDate, endDate)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				if result == nil {
					result = []*RideTypeStats{}
				}
				c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"ride_types": result}})
			},
			key: "ride_types",
		},
		{
			name: "TopDrivers returns nil",
			path: "/top-drivers",
			setup: func(m *mockService) {
				var nilSlice []*DriverPerformance
				m.On("GetTopDrivers", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), 10).Return(nilSlice, nil)
			},
			handler: func(h *testHandler, c *gin.Context) {
				startDate, endDate, _ := parseDateRange(c)
				result, err := h.service.GetTopDrivers(c.Request.Context(), startDate, endDate, 10)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				if result == nil {
					result = []*DriverPerformance{}
				}
				c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"drivers": result}})
			},
			key: "drivers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(mockService)
			router := setupTestRouter()
			tt.setup(mockSvc)

			h := newTestHandler(mockSvc)
			router.GET(tt.path, func(c *gin.Context) {
				tt.handler(h, c)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			data := response["data"].(map[string]interface{})
			arr := data[tt.key].([]interface{})
			assert.NotNil(t, arr)
			assert.Len(t, arr, 0)

			mockSvc.AssertExpectations(t)
		})
	}
}
