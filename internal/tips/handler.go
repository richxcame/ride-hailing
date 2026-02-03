package tips

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// Handler handles HTTP requests for tips
type Handler struct {
	service *Service
}

// NewHandler creates a new tips handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// RIDER ENDPOINTS
// ========================================

// SendTip sends a tip to a driver
// POST /api/v1/tips
func (h *Handler) SendTip(c *gin.Context) {
	riderID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SendTipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get driver ID from ride (passed in request for simplicity; in production would fetch from ride service)
	driverIDStr := c.Query("driver_id")
	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "driver_id query parameter required")
		return
	}

	tip, err := h.service.SendTip(c.Request.Context(), riderID, driverID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send tip")
		return
	}

	common.CreatedResponse(c, tip)
}

// GetTipPresets returns suggested tip amounts
// GET /api/v1/tips/presets?fare=25.00
func (h *Handler) GetTipPresets(c *gin.Context) {
	fareStr := c.DefaultQuery("fare", "10.00")
	fare, err := strconv.ParseFloat(fareStr, 64)
	if err != nil || fare <= 0 {
		fare = 10.0
	}

	presets := h.service.GetTipPresets(fare)
	common.SuccessResponse(c, presets)
}

// GetMyTipHistory returns tips the rider has sent
// GET /api/v1/tips/history?limit=20&offset=0
func (h *Handler) GetMyTipHistory(c *gin.Context) {
	riderID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	history, err := h.service.GetRiderTipHistory(c.Request.Context(), riderID, limit, offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get tip history")
		return
	}

	common.SuccessResponse(c, history)
}

// ========================================
// DRIVER ENDPOINTS
// ========================================

// GetDriverTipSummary returns tip stats for the driver
// GET /api/v1/driver/tips/summary?period=this_week
func (h *Handler) GetDriverTipSummary(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	period := c.DefaultQuery("period", "this_month")

	summary, err := h.service.GetDriverTipSummary(c.Request.Context(), driverID, period)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get tip summary")
		return
	}

	common.SuccessResponse(c, summary)
}

// GetDriverTips returns tips received by the driver
// GET /api/v1/driver/tips?limit=20&offset=0
func (h *Handler) GetDriverTips(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	tips, total, err := h.service.GetDriverTips(c.Request.Context(), driverID, limit, offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get tips")
		return
	}

	common.SuccessResponse(c, gin.H{
		"tips":  tips,
		"total": total,
	})
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers tip routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	// Rider tip routes
	rider := r.Group("/api/v1/tips")
	rider.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		rider.POST("", h.SendTip)
		rider.GET("/presets", h.GetTipPresets)
		rider.GET("/history", h.GetMyTipHistory)
	}

	// Driver tip routes
	driver := r.Group("/api/v1/driver/tips")
	driver.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	driver.Use(middleware.RequireRole(models.RoleDriver))
	{
		driver.GET("/summary", h.GetDriverTipSummary)
		driver.GET("", h.GetDriverTips)
	}
}
