package preferences

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// Handler handles HTTP requests for ride preferences
type Handler struct {
	service *Service
}

// NewHandler creates a new preferences handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// RIDER PREFERENCE ENDPOINTS
// ========================================

// GetPreferences returns the rider's default preferences
// GET /api/v1/preferences
func (h *Handler) GetPreferences(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	prefs, err := h.service.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get preferences")
		return
	}

	common.SuccessResponse(c, prefs)
}

// UpdatePreferences updates the rider's default preferences
// PUT /api/v1/preferences
func (h *Handler) UpdatePreferences(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	prefs, err := h.service.UpdatePreferences(c.Request.Context(), userID, &req)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update preferences")
		return
	}

	common.SuccessResponse(c, prefs)
}

// SetRidePreferences sets per-ride preference overrides
// POST /api/v1/preferences/rides
func (h *Handler) SetRidePreferences(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SetRidePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	override, err := h.service.SetRidePreferences(c.Request.Context(), userID, &req)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to set ride preferences")
		return
	}

	common.SuccessResponse(c, override)
}

// GetRidePreferences retrieves effective preferences for a ride
// GET /api/v1/preferences/rides/:id
func (h *Handler) GetRidePreferences(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	summary, err := h.service.GetRidePreferences(c.Request.Context(), rideID, userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get ride preferences")
		return
	}

	common.SuccessResponse(c, summary)
}

// ========================================
// DRIVER CAPABILITY ENDPOINTS
// ========================================

// GetDriverCapabilities returns driver's vehicle capabilities
// GET /api/v1/driver/capabilities
func (h *Handler) GetDriverCapabilities(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	caps, err := h.service.GetDriverCapabilities(c.Request.Context(), driverID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get capabilities")
		return
	}

	common.SuccessResponse(c, caps)
}

// UpdateDriverCapabilities updates driver's vehicle capabilities
// PUT /api/v1/driver/capabilities
func (h *Handler) UpdateDriverCapabilities(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req UpdateDriverCapabilitiesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	caps, err := h.service.UpdateDriverCapabilities(c.Request.Context(), driverID, &req)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update capabilities")
		return
	}

	common.SuccessResponse(c, caps)
}

// GetRidePreferencesForDriver returns rider preferences visible to the driver
// GET /api/v1/driver/rides/:id/preferences
func (h *Handler) GetRidePreferencesForDriver(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	// The riderID would normally come from the ride record.
	// For now, pass uuid.Nil and let the service handle the lookup.
	riderIDStr := c.Query("rider_id")
	riderID, _ := uuid.Parse(riderIDStr)

	summary, err := h.service.GetPreferencesForDriver(c.Request.Context(), rideID, driverID, riderID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get preferences")
		return
	}

	common.SuccessResponse(c, summary)
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers preference routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	// Rider preferences
	prefs := r.Group("/api/v1/preferences")
	prefs.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		prefs.GET("", h.GetPreferences)
		prefs.PUT("", h.UpdatePreferences)
		prefs.POST("/rides", h.SetRidePreferences)
		prefs.GET("/rides/:id", h.GetRidePreferences)
	}

	// Driver capabilities
	driver := r.Group("/api/v1/driver")
	driver.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	driver.Use(middleware.RequireRole(models.RoleDriver))
	{
		driver.GET("/capabilities", h.GetDriverCapabilities)
		driver.PUT("/capabilities", h.UpdateDriverCapabilities)
		driver.GET("/rides/:id/preferences", h.GetRidePreferencesForDriver)
	}
}
