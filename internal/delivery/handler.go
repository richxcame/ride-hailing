package delivery

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/richxcame/ride-hailing/pkg/pagination"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for package delivery
type Handler struct {
	service *Service
}

// NewHandler creates a new delivery handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// ESTIMATION
// ========================================

// GetEstimate returns a fare estimate for a delivery
// POST /api/v1/deliveries/estimate
func (h *Handler) GetEstimate(c *gin.Context) {
	var req DeliveryEstimateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	estimate, err := h.service.GetEstimate(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get estimate")
		return
	}

	common.SuccessResponse(c, estimate)
}

// ========================================
// SENDER ENDPOINTS
// ========================================

// CreateDelivery creates a new delivery order
// POST /api/v1/deliveries
func (h *Handler) CreateDelivery(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateDeliveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	response, err := h.service.CreateDelivery(c.Request.Context(), userID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create delivery")
		return
	}

	common.CreatedResponse(c, response)
}

// GetDelivery retrieves a delivery with full details
// GET /api/v1/deliveries/:id
func (h *Handler) GetDelivery(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	deliveryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid delivery ID")
		return
	}

	response, err := h.service.GetDelivery(c.Request.Context(), userID, deliveryID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get delivery")
		return
	}

	common.SuccessResponse(c, response)
}

// GetMyDeliveries lists the sender's deliveries
// GET /api/v1/deliveries
func (h *Handler) GetMyDeliveries(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	params := pagination.ParseParams(c)

	var filters DeliveryListFilters
	if status := c.Query("status"); status != "" {
		s := DeliveryStatus(status)
		filters.Status = &s
	}
	if priority := c.Query("priority"); priority != "" {
		p := DeliveryPriority(priority)
		filters.Priority = &p
	}

	deliveries, total, err := h.service.GetMyDeliveries(c.Request.Context(), userID, &filters, params.Limit, params.Offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list deliveries")
		return
	}

	if deliveries == nil {
		deliveries = []*Delivery{}
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, int64(total))
	common.SuccessResponseWithMeta(c, gin.H{
		"deliveries": deliveries,
	}, meta)
}

// CancelDelivery cancels a delivery
// POST /api/v1/deliveries/:id/cancel
func (h *Handler) CancelDelivery(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	deliveryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid delivery ID")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	// Bind is optional - cancellation reason may not be provided
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		logger.Get().Debug("Failed to bind cancel delivery request body", zap.Error(err))
	}

	if err := h.service.CancelDelivery(c.Request.Context(), deliveryID, userID, req.Reason); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to cancel delivery")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Delivery cancelled"})
}

// RateDelivery rates a completed delivery
// POST /api/v1/deliveries/:id/rate
func (h *Handler) RateDelivery(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	deliveryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid delivery ID")
		return
	}

	var req RateDeliveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.RateDelivery(c.Request.Context(), deliveryID, userID, &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to rate delivery")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Delivery rated"})
}

// GetStats returns delivery statistics
// GET /api/v1/deliveries/stats
func (h *Handler) GetStats(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	stats, err := h.service.GetStats(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get stats")
		return
	}

	common.SuccessResponse(c, stats)
}

// ========================================
// PUBLIC TRACKING
// ========================================

// TrackDelivery allows public tracking by code
// GET /api/v1/deliveries/track/:code
func (h *Handler) TrackDelivery(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "tracking code required")
		return
	}

	response, err := h.service.TrackDelivery(c.Request.Context(), code)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "delivery not found")
		return
	}

	common.SuccessResponse(c, response)
}

// ========================================
// DRIVER ENDPOINTS
// ========================================

// GetAvailableDeliveries lists deliveries for drivers to accept
// GET /api/v1/driver/deliveries/available
func (h *Handler) GetAvailableDeliveries(c *gin.Context) {
	latStr := c.Query("latitude")
	lngStr := c.Query("longitude")
	if latStr == "" || lngStr == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "latitude and longitude are required")
		return
	}

	latitude, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid latitude")
		return
	}
	longitude, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid longitude")
		return
	}

	deliveries, err := h.service.GetAvailableDeliveries(c.Request.Context(), latitude, longitude)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get available deliveries")
		return
	}

	if deliveries == nil {
		deliveries = []*Delivery{}
	}

	common.SuccessResponse(c, gin.H{
		"deliveries": deliveries,
		"count":      len(deliveries),
	})
}

// AcceptDelivery lets a driver accept a delivery
// POST /api/v1/driver/deliveries/:id/accept
func (h *Handler) AcceptDelivery(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	deliveryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid delivery ID")
		return
	}

	response, err := h.service.AcceptDelivery(c.Request.Context(), deliveryID, driverID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to accept delivery")
		return
	}

	common.SuccessResponse(c, response)
}

// ConfirmPickup marks package as picked up
// POST /api/v1/driver/deliveries/:id/pickup
func (h *Handler) ConfirmPickup(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	deliveryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid delivery ID")
		return
	}

	var req ConfirmPickupRequest
	// Bind is optional - pickup confirmation may not have additional data
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		logger.Get().Debug("Failed to bind confirm pickup request body", zap.Error(err))
	}

	if err := h.service.ConfirmPickup(c.Request.Context(), deliveryID, driverID, &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to confirm pickup")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Pickup confirmed"})
}

// UpdateStatus updates delivery status (en route, arrived, etc.)
// POST /api/v1/driver/deliveries/:id/status
func (h *Handler) UpdateStatus(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	deliveryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid delivery ID")
		return
	}

	var req struct {
		Status DeliveryStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.UpdateStatus(c.Request.Context(), deliveryID, driverID, req.Status); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update status")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Status updated"})
}

// ConfirmDelivery marks delivery as complete with proof
// POST /api/v1/driver/deliveries/:id/deliver
func (h *Handler) ConfirmDelivery(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	deliveryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid delivery ID")
		return
	}

	var req ConfirmDeliveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.ConfirmDelivery(c.Request.Context(), deliveryID, driverID, &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to confirm delivery")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Delivery confirmed"})
}

// ReturnDelivery returns package to sender
// POST /api/v1/driver/deliveries/:id/return
func (h *Handler) ReturnDelivery(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	deliveryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid delivery ID")
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "reason is required")
		return
	}

	if err := h.service.ReturnDelivery(c.Request.Context(), deliveryID, driverID, req.Reason); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to return delivery")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Return initiated"})
}

// GetActiveDelivery gets the driver's current active delivery
// GET /api/v1/driver/deliveries/active
func (h *Handler) GetActiveDelivery(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	response, err := h.service.GetActiveDelivery(c.Request.Context(), driverID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get active delivery")
		return
	}

	common.SuccessResponse(c, response)
}

// GetDriverDeliveries lists driver's delivery history
// GET /api/v1/driver/deliveries
func (h *Handler) GetDriverDeliveries(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	params := pagination.ParseParams(c)

	deliveries, total, err := h.service.GetDriverDeliveries(c.Request.Context(), driverID, params.Limit, params.Offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list deliveries")
		return
	}

	if deliveries == nil {
		deliveries = []*Delivery{}
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, int64(total))
	common.SuccessResponseWithMeta(c, gin.H{
		"deliveries": deliveries,
	}, meta)
}

// UpdateStopStatus updates an intermediate stop
// POST /api/v1/driver/deliveries/:id/stops/:stopId/status
func (h *Handler) UpdateStopStatus(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	deliveryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid delivery ID")
		return
	}

	stopID, err := uuid.Parse(c.Param("stopId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid stop ID")
		return
	}

	var req struct {
		Status   string  `json:"status" binding:"required"`
		PhotoURL *string `json:"photo_url,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.UpdateStopStatus(c.Request.Context(), deliveryID, stopID, driverID, req.Status, req.PhotoURL); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update stop status")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Stop status updated"})
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers delivery routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	// Public tracking (no auth required)
	r.GET("/api/v1/deliveries/track/:code", h.TrackDelivery)

	// Sender routes (riders can send packages)
	deliveries := r.Group("/api/v1/deliveries")
	deliveries.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		deliveries.POST("/estimate", h.GetEstimate)
		deliveries.POST("", h.CreateDelivery)
		deliveries.GET("", h.GetMyDeliveries)
		deliveries.GET("/stats", h.GetStats)
		deliveries.GET("/:id", h.GetDelivery)
		deliveries.POST("/:id/cancel", h.CancelDelivery)
		deliveries.POST("/:id/rate", h.RateDelivery)
	}

	// Driver routes
	driverDeliveries := r.Group("/api/v1/driver/deliveries")
	driverDeliveries.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	driverDeliveries.Use(middleware.RequireRole(models.RoleDriver))
	{
		driverDeliveries.GET("/available", h.GetAvailableDeliveries)
		driverDeliveries.GET("/active", h.GetActiveDelivery)
		driverDeliveries.GET("", h.GetDriverDeliveries)
		driverDeliveries.POST("/:id/accept", h.AcceptDelivery)
		driverDeliveries.POST("/:id/pickup", h.ConfirmPickup)
		driverDeliveries.POST("/:id/status", h.UpdateStatus)
		driverDeliveries.POST("/:id/deliver", h.ConfirmDelivery)
		driverDeliveries.POST("/:id/return", h.ReturnDelivery)
		driverDeliveries.POST("/:id/stops/:stopId/status", h.UpdateStopStatus)
	}
}
