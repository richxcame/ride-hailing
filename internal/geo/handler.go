package geo

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// Handler handles HTTP requests for geolocation
type Handler struct {
	service *Service
}

// NewHandler creates a new geo handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// UpdateLocation handles updating driver location
func (h *Handler) UpdateLocation(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Latitude  float64 `json:"latitude" binding:"required"`
		Longitude float64 `json:"longitude" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.UpdateDriverLocation(c.Request.Context(), driverID, req.Latitude, req.Longitude); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update location")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "location updated successfully"})
}

// GetDriverLocation handles getting a driver's location
func (h *Handler) GetDriverLocation(c *gin.Context) {
	driverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid driver ID")
		return
	}

	location, err := h.service.GetDriverLocation(c.Request.Context(), driverID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get driver location")
		return
	}

	common.SuccessResponse(c, location)
}

// CalculateDistance handles distance calculation
func (h *Handler) CalculateDistance(c *gin.Context) {
	var req struct {
		FromLat float64 `json:"from_latitude" binding:"required"`
		FromLon float64 `json:"from_longitude" binding:"required"`
		ToLat   float64 `json:"to_latitude" binding:"required"`
		ToLon   float64 `json:"to_longitude" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	distance := h.service.CalculateDistance(req.FromLat, req.FromLon, req.ToLat, req.ToLon)
	eta := h.service.CalculateETA(distance)

	common.SuccessResponse(c, gin.H{
		"distance_km": distance,
		"eta_minutes": eta,
	})
}

// RegisterRoutes registers geo routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))

	geo := api.Group("/geo")
	{
		// Driver location updates
		geo.POST("/location", middleware.RequireRole(models.RoleDriver), h.UpdateLocation)
		geo.GET("/drivers/:id/location", h.GetDriverLocation)

		// Utility endpoints
		geo.POST("/distance", h.CalculateDistance)
	}
}
