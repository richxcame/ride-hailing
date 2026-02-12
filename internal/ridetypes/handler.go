package ridetypes

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/richxcame/ride-hailing/pkg/common"
)

// Handler handles mobile-facing HTTP requests for ride types
type Handler struct {
	service *Service
}

// NewHandler creates a new ride types handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers mobile-facing ride type routes
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rt := rg.Group("/ride-types")
	{
		rt.GET("/available", h.GetAvailableRideTypes)
	}
}

// GetAvailableRideTypes returns ride types available at a given location
func (h *Handler) GetAvailableRideTypes(c *gin.Context) {
	latStr := c.Query("lat")
	lngStr := c.Query("lng")

	if latStr == "" || lngStr == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "lat and lng are required")
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid lat value")
		return
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid lng value")
		return
	}

	rideTypes, err := h.service.GetAvailableRideTypes(c.Request.Context(), lat, lng)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to get available ride types")
		return
	}

	common.SuccessResponse(c, rideTypes)
}
