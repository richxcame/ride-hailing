package analytics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/richxcame/ride-hailing/pkg/common"
)

// Handler handles HTTP requests for analytics
type Handler struct {
	service *Service
}

// NewHandler creates a new analytics handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetRevenueMetrics handles revenue metrics requests
func (h *Handler) GetRevenueMetrics(c *gin.Context) {
	startDate, endDate, err := parseDateRange(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	metrics, err := h.service.GetRevenueMetrics(c.Request.Context(), startDate, endDate)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get revenue metrics")
		return
	}

	common.SuccessResponse(c, metrics)
}

// GetPromoCodePerformance handles promo code performance requests
func (h *Handler) GetPromoCodePerformance(c *gin.Context) {
	startDate, endDate, err := parseDateRange(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	performance, err := h.service.GetPromoCodePerformance(c.Request.Context(), startDate, endDate)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get promo code performance")
		return
	}

	common.SuccessResponse(c, gin.H{
		"promo_codes": performance,
	})
}

// GetRideTypeStats handles ride type statistics requests
func (h *Handler) GetRideTypeStats(c *gin.Context) {
	startDate, endDate, err := parseDateRange(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	stats, err := h.service.GetRideTypeStats(c.Request.Context(), startDate, endDate)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get ride type stats")
		return
	}

	common.SuccessResponse(c, gin.H{
		"ride_types": stats,
	})
}

// GetReferralMetrics handles referral metrics requests
func (h *Handler) GetReferralMetrics(c *gin.Context) {
	startDate, endDate, err := parseDateRange(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	metrics, err := h.service.GetReferralMetrics(c.Request.Context(), startDate, endDate)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get referral metrics")
		return
	}

	common.SuccessResponse(c, metrics)
}

// GetTopDrivers handles top drivers requests
func (h *Handler) GetTopDrivers(c *gin.Context) {
	startDate, endDate, err := parseDateRange(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	drivers, err := h.service.GetTopDrivers(c.Request.Context(), startDate, endDate, limit)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get top drivers")
		return
	}

	common.SuccessResponse(c, gin.H{
		"drivers": drivers,
	})
}

// GetDashboardMetrics handles dashboard metrics requests
func (h *Handler) GetDashboardMetrics(c *gin.Context) {
	metrics, err := h.service.GetDashboardMetrics(c.Request.Context())
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get dashboard metrics")
		return
	}

	common.SuccessResponse(c, metrics)
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service": "analytics",
		"status":  "healthy",
	})
}

// parseDateRange parses start_date and end_date query parameters
func parseDateRange(c *gin.Context) (time.Time, time.Time, error) {
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	var startDate, endDate time.Time
	var err error

	// Default to last 30 days if not specified
	if startDateStr == "" {
		startDate = time.Now().AddDate(0, 0, -30).Truncate(24 * time.Hour)
	} else {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	if endDateStr == "" {
		endDate = time.Now()
	} else {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		// Set to end of day
		endDate = endDate.Add(24 * time.Hour).Add(-time.Second)
	}

	return startDate, endDate, nil
}
