package ridehistory

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// Handler handles HTTP requests for ride history
type Handler struct {
	service *Service
}

// NewHandler creates a new ride history handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// RIDER ENDPOINTS
// ========================================

// GetRiderHistory returns the rider's ride history
// GET /api/v1/rides/history?page=1&page_size=20&status=completed&from=2025-01-01&to=2025-12-31
func (h *Handler) GetRiderHistory(c *gin.Context) {
	riderID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filters := &HistoryFilters{}
	if status := c.Query("status"); status != "" {
		filters.Status = &status
	}
	if fromStr := c.Query("from"); fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			filters.FromDate = &t
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			next := t.AddDate(0, 0, 1)
			filters.ToDate = &next
		}
	}

	resp, err := h.service.GetRiderHistory(c.Request.Context(), riderID, filters, page, pageSize)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get ride history")
		return
	}

	common.SuccessResponse(c, resp)
}

// GetRideDetails returns full details of a specific ride
// GET /api/v1/rides/history/:id
func (h *Handler) GetRideDetails(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride id")
		return
	}

	ride, err := h.service.GetRideDetails(c.Request.Context(), rideID, userID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get ride details")
		return
	}

	common.SuccessResponse(c, ride)
}

// GetReceipt generates a receipt for a ride
// GET /api/v1/rides/history/:id/receipt
func (h *Handler) GetReceipt(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride id")
		return
	}

	receipt, err := h.service.GetReceipt(c.Request.Context(), rideID, userID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to generate receipt")
		return
	}

	common.SuccessResponse(c, receipt)
}

// GetRiderStats returns ride statistics
// GET /api/v1/rides/stats?period=this_month
func (h *Handler) GetRiderStats(c *gin.Context) {
	riderID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	period := c.DefaultQuery("period", "all_time")

	stats, err := h.service.GetRiderStats(c.Request.Context(), riderID, period)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get stats")
		return
	}

	common.SuccessResponse(c, stats)
}

// GetFrequentRoutes returns commonly taken routes
// GET /api/v1/rides/frequent-routes
func (h *Handler) GetFrequentRoutes(c *gin.Context) {
	riderID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	routes, err := h.service.GetFrequentRoutes(c.Request.Context(), riderID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get frequent routes")
		return
	}

	common.SuccessResponse(c, gin.H{"routes": routes})
}

// ========================================
// DRIVER ENDPOINTS
// ========================================

// GetDriverHistory returns the driver's ride history
// GET /api/v1/driver/rides/history?page=1&page_size=20
func (h *Handler) GetDriverHistory(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filters := &HistoryFilters{}
	if status := c.Query("status"); status != "" {
		filters.Status = &status
	}
	if fromStr := c.Query("from"); fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			filters.FromDate = &t
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			next := t.AddDate(0, 0, 1)
			filters.ToDate = &next
		}
	}

	resp, err := h.service.GetDriverHistory(c.Request.Context(), driverID, filters, page, pageSize)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get ride history")
		return
	}

	common.SuccessResponse(c, resp)
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers ride history routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	// Rider routes
	rider := r.Group("/api/v1/rides")
	rider.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		rider.GET("/history", h.GetRiderHistory)
		rider.GET("/history/:id", h.GetRideDetails)
		rider.GET("/history/:id/receipt", h.GetReceipt)
		rider.GET("/stats", h.GetRiderStats)
		rider.GET("/frequent-routes", h.GetFrequentRoutes)
	}

	// Driver routes
	driver := r.Group("/api/v1/driver/rides")
	driver.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	driver.Use(middleware.RequireRole(models.RoleDriver))
	{
		driver.GET("/history", h.GetDriverHistory)
		driver.GET("/history/:id", h.GetRideDetails)
		driver.GET("/history/:id/receipt", h.GetReceipt)
	}
}
