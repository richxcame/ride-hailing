package pricing

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
)

// RideTypesProvider defines the interface for fetching ride types
type RideTypesProvider interface {
	GetAvailableRideTypes(ctx interface{}, lat, lng float64) ([]interface{}, error)
}

// Handler handles HTTP requests for pricing
type Handler struct {
	service           *Service
	rideTypesProvider RideTypesProvider
}

// NewHandler creates a new pricing handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// NewHandlerWithRideTypes creates a new pricing handler with ride types provider
func NewHandlerWithRideTypes(service *Service, rideTypesProvider RideTypesProvider) *Handler {
	return &Handler{
		service:           service,
		rideTypesProvider: rideTypesProvider,
	}
}

// GetEstimate returns a fare estimate
func (h *Handler) GetEstimate(c *gin.Context) {
	var req EstimateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	estimate, err := h.service.GetEstimate(c.Request.Context(), req)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to calculate estimate")
		return
	}

	common.SuccessResponse(c, estimate)
}

// GetBulkEstimate returns fare estimates for all available ride types
func (h *Handler) GetBulkEstimate(c *gin.Context) {
	var req BulkEstimateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Get available ride types for the pickup location
	var rideTypes []RideTypeInfo
	if h.rideTypesProvider != nil {
		rawRideTypes, err := h.rideTypesProvider.GetAvailableRideTypes(c.Request.Context(), req.PickupLatitude, req.PickupLongitude)
		if err != nil {
			common.ErrorResponse(c, http.StatusInternalServerError, "failed to get available ride types")
			return
		}

		// Convert to RideTypeInfo
		for _, rt := range rawRideTypes {
			if rtMap, ok := rt.(map[string]interface{}); ok {
				rideTypeInfo := RideTypeInfo{
					Name:        getString(rtMap, "name"),
					Description: getString(rtMap, "description"),
					Capacity:    getInt(rtMap, "capacity"),
				}
				if idStr, ok := rtMap["id"].(string); ok {
					if id, err := uuid.Parse(idStr); err == nil {
						rideTypeInfo.ID = id
					}
				}
				if iconURL, ok := rtMap["icon_url"].(string); ok && iconURL != "" {
					rideTypeInfo.IconURL = &iconURL
				}
				rideTypes = append(rideTypes, rideTypeInfo)
			}
		}
	}

	// If no ride types available, return empty result
	if len(rideTypes) == 0 {
		common.ErrorResponse(c, http.StatusNotFound, "no ride types available for this location")
		return
	}

	// Get bulk estimate
	bulkEstimate, err := h.service.GetBulkEstimate(c.Request.Context(), req, rideTypes)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to calculate estimates")
		return
	}

	common.SuccessResponse(c, bulkEstimate)
}

// Helper functions for type conversion
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	if v, ok := m[key].(int); ok {
		return v
	}
	return 0
}

// GetSurge returns current surge information
func (h *Handler) GetSurge(c *gin.Context) {
	latStr := c.Query("lat")
	lngStr := c.Query("lng")

	if latStr == "" || lngStr == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "lat and lng are required")
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid lat")
		return
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid lng")
		return
	}

	surgeInfo, err := h.service.GetSurgeInfo(c.Request.Context(), lat, lng)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get surge info")
		return
	}

	common.SuccessResponse(c, surgeInfo)
}

// ValidatePrice validates a negotiated price
func (h *Handler) ValidatePrice(c *gin.Context) {
	var req struct {
		EstimateRequest
		NegotiatedPrice float64 `json:"negotiated_price" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	err := h.service.ValidateNegotiatedPrice(c.Request.Context(), req.EstimateRequest, req.NegotiatedPrice)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Get the estimate for reference
	estimate, _ := h.service.GetEstimate(c.Request.Context(), req.EstimateRequest)

	common.SuccessResponse(c, gin.H{
		"valid":           true,
		"negotiated_price": req.NegotiatedPrice,
		"estimated_price": estimate.EstimatedFare,
		"variance_pct":    ((req.NegotiatedPrice - estimate.EstimatedFare) / estimate.EstimatedFare) * 100,
	})
}

// GetPricing returns resolved pricing for a location
func (h *Handler) GetPricing(c *gin.Context) {
	latStr := c.Query("lat")
	lngStr := c.Query("lng")
	rideTypeIDStr := c.Query("ride_type_id")

	if latStr == "" || lngStr == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "lat and lng are required")
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid lat")
		return
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid lng")
		return
	}

	var rideTypeID *uuid.UUID
	if rideTypeIDStr != "" {
		id, err := uuid.Parse(rideTypeIDStr)
		if err != nil {
			common.ErrorResponse(c, http.StatusBadRequest, "invalid ride_type_id")
			return
		}
		rideTypeID = &id
	}

	pricing, err := h.service.GetPricing(c.Request.Context(), lat, lng, rideTypeID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get pricing")
		return
	}

	common.SuccessResponse(c, pricing)
}

// GetCancellationFee returns the cancellation fee
func (h *Handler) GetCancellationFee(c *gin.Context) {
	var req struct {
		Latitude            float64 `json:"latitude" binding:"required"`
		Longitude           float64 `json:"longitude" binding:"required"`
		MinutesSinceRequest float64 `json:"minutes_since_request" binding:"required,gte=0"`
		EstimatedFare       float64 `json:"estimated_fare" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	fee, err := h.service.GetCancellationFee(
		c.Request.Context(),
		req.Latitude, req.Longitude,
		req.MinutesSinceRequest,
		req.EstimatedFare,
	)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to calculate cancellation fee")
		return
	}

	common.SuccessResponse(c, gin.H{
		"cancellation_fee":     fee,
		"minutes_since_request": req.MinutesSinceRequest,
	})
}

// RegisterRoutes registers pricing routes
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	pricing := rg.Group("/pricing")
	{
		pricing.POST("/estimate", h.GetEstimate)
		pricing.POST("/bulk-estimate", h.GetBulkEstimate)
		pricing.GET("/surge", h.GetSurge)
		pricing.POST("/validate", h.ValidatePrice)
		pricing.GET("/config", h.GetPricing)
		pricing.POST("/cancellation-fee", h.GetCancellationFee)
	}
}
