package cancellation

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

// Handler handles HTTP requests for cancellations
type Handler struct {
	service *Service
}

// NewHandler creates a new cancellation handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// RIDER / DRIVER ENDPOINTS
// ========================================

// PreviewCancellation shows the fee before cancelling
// GET /api/v1/rides/:id/cancel/preview
func (h *Handler) PreviewCancellation(c *gin.Context) {
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

	preview, err := h.service.PreviewCancellation(c.Request.Context(), rideID, userID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to preview cancellation")
		return
	}

	common.SuccessResponse(c, preview)
}

// CancelRide cancels a ride with structured reason and fee
// POST /api/v1/rides/:id/cancel
func (h *Handler) CancelRide(c *gin.Context) {
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

	var req CancelRideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.CancelRide(c.Request.Context(), rideID, userID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to cancel ride")
		return
	}

	common.SuccessResponse(c, resp)
}

// GetCancellationDetails returns cancellation details for a ride
// GET /api/v1/rides/:id/cancellation
func (h *Handler) GetCancellationDetails(c *gin.Context) {
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

	rec, err := h.service.GetCancellationDetails(c.Request.Context(), rideID, userID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get cancellation details")
		return
	}

	common.SuccessResponse(c, rec)
}

// GetMyCancellationStats returns the user's cancellation stats
// GET /api/v1/cancellations/stats
func (h *Handler) GetMyCancellationStats(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	stats, err := h.service.GetMyCancellationStats(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get cancellation stats")
		return
	}

	common.SuccessResponse(c, stats)
}

// GetMyCancellationHistory returns the user's cancellation history
// GET /api/v1/cancellations/history?page=1&page_size=20
func (h *Handler) GetMyCancellationHistory(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	records, total, err := h.service.GetMyCancellationHistory(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get cancellation history")
		return
	}

	common.SuccessResponse(c, gin.H{
		"cancellations": records,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
	})
}

// GetCancellationReasons returns available cancellation reasons
// GET /api/v1/cancellations/reasons?type=rider
func (h *Handler) GetCancellationReasons(c *gin.Context) {
	userType := c.DefaultQuery("type", "rider")
	isDriver := userType == "driver"
	reasons := h.service.GetCancellationReasons(isDriver)
	common.SuccessResponse(c, reasons)
}

// ========================================
// ADMIN ENDPOINTS
// ========================================

// AdminWaiveFee waives a cancellation fee
// POST /api/v1/admin/cancellations/:id/waive
func (h *Handler) AdminWaiveFee(c *gin.Context) {
	cancellationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid cancellation id")
		return
	}

	var req WaiveFeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.WaiveFee(c.Request.Context(), cancellationID, req.Reason); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to waive fee")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "cancellation fee waived"})
}

// AdminGetStats returns platform-wide cancellation analytics
// GET /api/v1/admin/cancellations/stats?from=2025-01-01&to=2025-12-31
func (h *Handler) AdminGetStats(c *gin.Context) {
	from := time.Now().AddDate(0, -1, 0)
	to := time.Now()

	if fromStr := c.Query("from"); fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = t
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			to = t.AddDate(0, 0, 1)
		}
	}

	stats, err := h.service.GetCancellationStats(c.Request.Context(), from, to)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get cancellation stats")
		return
	}

	common.SuccessResponse(c, stats)
}

// AdminGetUserStats returns cancellation stats for a specific user
// GET /api/v1/admin/cancellations/users/:userId/stats
func (h *Handler) AdminGetUserStats(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user id")
		return
	}

	stats, err := h.service.GetUserCancellationStats(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get user cancellation stats")
		return
	}

	common.SuccessResponse(c, stats)
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers cancellation routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	// Ride cancellation (authenticated users)
	rides := r.Group("/api/v1/rides")
	rides.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		rides.GET("/:id/cancel/preview", h.PreviewCancellation)
		rides.POST("/:id/cancel", h.CancelRide)
		rides.GET("/:id/cancellation", h.GetCancellationDetails)
	}

	// Cancellation history & stats (authenticated users)
	cancellations := r.Group("/api/v1/cancellations")
	cancellations.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		cancellations.GET("/stats", h.GetMyCancellationStats)
		cancellations.GET("/history", h.GetMyCancellationHistory)
		cancellations.GET("/reasons", h.GetCancellationReasons)
	}

	// Admin routes
	admin := r.Group("/api/v1/admin/cancellations")
	admin.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	admin.Use(middleware.RequireRole(models.RoleAdmin))
	{
		admin.POST("/:id/waive", h.AdminWaiveFee)
		admin.GET("/stats", h.AdminGetStats)
		admin.GET("/users/:userId/stats", h.AdminGetUserStats)
	}
}
