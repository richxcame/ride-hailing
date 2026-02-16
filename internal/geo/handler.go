package geo

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

// Handler handles HTTP requests for geolocation
type Handler struct {
	service    *Service
	geocoding  *GeocodingService
}

// NewHandler creates a new geo handler
func NewHandler(service *Service, geocoding *GeocodingService) *Handler {
	return &Handler{service: service, geocoding: geocoding}
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
		Heading   float64 `json:"heading"`
		Speed     float64 `json:"speed"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.UpdateDriverLocationFull(c.Request.Context(), driverID, req.Latitude, req.Longitude, req.Heading, req.Speed); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update location")
		return
	}

	// Return the H3 cell for the driver's location
	h3Cell := GetMatchingCell(req.Latitude, req.Longitude)
	common.SuccessResponse(c, gin.H{
		"message": "location updated successfully",
		"h3_cell": h3Cell,
	})
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

// FindNearbyDrivers handles finding nearby available drivers
func (h *Handler) FindNearbyDrivers(c *gin.Context) {
	latitude, err := strconv.ParseFloat(c.Query("latitude"), 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid latitude")
		return
	}
	longitude, err := strconv.ParseFloat(c.Query("longitude"), 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid longitude")
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	drivers, err := h.service.FindAvailableDrivers(c.Request.Context(), latitude, longitude, limit)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to find nearby drivers")
		return
	}

	common.SuccessResponse(c, gin.H{
		"drivers":  drivers,
		"h3_cell":  GetMatchingCell(latitude, longitude),
		"surge":    GetSurgeZone(latitude, longitude),
	})
}

// CalculateDistance handles distance calculation
func (h *Handler) CalculateDistance(c *gin.Context) {
	var req struct {
		FromLatitude  float64 `json:"from_latitude" binding:"required"`
		FromLongitude float64 `json:"from_longitude" binding:"required"`
		ToLatitude    float64 `json:"to_latitude" binding:"required"`
		ToLongitude   float64 `json:"to_longitude" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	distance := h.service.CalculateDistance(req.FromLatitude, req.FromLongitude, req.ToLatitude, req.ToLongitude)
	eta := h.service.CalculateETA(distance)

	common.SuccessResponse(c, gin.H{
		"distance_km":    distance,
		"eta_minutes":    eta,
		"from_h3_cell":  GetMatchingCell(req.FromLatitude, req.FromLongitude),
		"to_h3_cell":    GetMatchingCell(req.ToLatitude, req.ToLongitude),
		"from_surge_zone": GetSurgeZone(req.FromLatitude, req.FromLongitude),
		"to_surge_zone":   GetSurgeZone(req.ToLatitude, req.ToLongitude),
	})
}

// GetSurgeInfo handles getting surge pricing info for a location
func (h *Handler) GetSurgeInfo(c *gin.Context) {
	latitude, err := strconv.ParseFloat(c.Query("latitude"), 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid latitude")
		return
	}
	longitude, err := strconv.ParseFloat(c.Query("longitude"), 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid longitude")
		return
	}

	surgeInfo, err := h.service.GetSurgeInfo(c.Request.Context(), latitude, longitude)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get surge info")
		return
	}

	common.SuccessResponse(c, surgeInfo)
}

// GetDemandHeatmap handles getting demand heatmap data
func (h *Handler) GetDemandHeatmap(c *gin.Context) {
	latitude, err := strconv.ParseFloat(c.Query("latitude"), 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid latitude")
		return
	}
	longitude, err := strconv.ParseFloat(c.Query("longitude"), 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid longitude")
		return
	}

	heatmap, err := h.service.GetDemandHeatmap(c.Request.Context(), latitude, longitude)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get demand heatmap")
		return
	}

	common.SuccessResponse(c, gin.H{
		"center_h3_cell": GetDemandZone(latitude, longitude),
		"heatmap":        heatmap,
	})
}

// GetH3Cell handles resolving a location to H3 cell indexes at various resolutions
func (h *Handler) GetH3Cell(c *gin.Context) {
	latitude, err := strconv.ParseFloat(c.Query("latitude"), 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid latitude")
		return
	}
	longitude, err := strconv.ParseFloat(c.Query("longitude"), 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid longitude")
		return
	}

	common.SuccessResponse(c, gin.H{
		"latitude":      latitude,
		"longitude":     longitude,
		"matching_cell": GetMatchingCell(latitude, longitude),
		"surge_zone":    GetSurgeZone(latitude, longitude),
		"demand_zone":   GetDemandZone(latitude, longitude),
		"resolutions": gin.H{
			"6_city":     CellToString(LatLngToCell(latitude, longitude, H3ResolutionCity)),
			"7_demand":   CellToString(LatLngToCell(latitude, longitude, H3ResolutionDemand)),
			"8_surge":    CellToString(LatLngToCell(latitude, longitude, H3ResolutionSurge)),
			"9_matching": CellToString(LatLngToCell(latitude, longitude, H3ResolutionMatching)),
		},
	})
}

// ForwardGeocode handles converting an address to coordinates.
func (h *Handler) ForwardGeocode(c *gin.Context) {
	address := c.Query("address")
	if address == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "address query parameter is required")
		return
	}

	results, err := h.geocoding.ForwardGeocode(c.Request.Context(), address)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "geocoding failed")
		return
	}

	common.SuccessResponse(c, gin.H{"results": results})
}

// ReverseGeocode handles converting coordinates to an address.
func (h *Handler) ReverseGeocode(c *gin.Context) {
	latitude, err := strconv.ParseFloat(c.Query("latitude"), 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid latitude")
		return
	}
	longitude, err := strconv.ParseFloat(c.Query("longitude"), 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid longitude")
		return
	}

	results, err := h.geocoding.ReverseGeocode(c.Request.Context(), latitude, longitude)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "reverse geocoding failed")
		return
	}

	common.SuccessResponse(c, gin.H{"results": results})
}

// PlaceAutocomplete handles place search autocomplete.
func (h *Handler) PlaceAutocomplete(c *gin.Context) {
	input := c.Query("input")
	if input == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "input query parameter is required")
		return
	}

	var latitude, longitude float64
	if latitudeStr := c.Query("latitude"); latitudeStr != "" {
		var err error
		latitude, err = strconv.ParseFloat(latitudeStr, 64)
		if err != nil {
			common.ErrorResponse(c, http.StatusBadRequest, "invalid latitude value")
			return
		}
	}
	if longitudeStr := c.Query("longitude"); longitudeStr != "" {
		var err error
		longitude, err = strconv.ParseFloat(longitudeStr, 64)
		if err != nil {
			common.ErrorResponse(c, http.StatusBadRequest, "invalid longitude value")
			return
		}
	}

	results, err := h.geocoding.Autocomplete(c.Request.Context(), input, latitude, longitude)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "autocomplete failed")
		return
	}

	common.SuccessResponse(c, gin.H{"predictions": results})
}

// GetPlaceDetails handles getting details for a place by ID.
func (h *Handler) GetPlaceDetails(c *gin.Context) {
	placeID := c.Query("place_id")
	if placeID == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "place_id query parameter is required")
		return
	}

	result, err := h.geocoding.GetPlaceDetails(c.Request.Context(), placeID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get place details")
		return
	}

	common.SuccessResponse(c, result)
}

// RegisterActiveRide registers a ride for real-time ETA tracking (internal endpoint).
func (h *Handler) RegisterActiveRide(c *gin.Context) {
	var req ActiveRideInfo
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if h.service.etaTracker == nil {
		common.ErrorResponse(c, http.StatusServiceUnavailable, "ETA tracking not enabled")
		return
	}

	if err := h.service.etaTracker.RegisterActiveRide(c.Request.Context(), &req); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to register ride")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "ride registered for ETA tracking"})
}

// UnregisterActiveRide removes a ride from ETA tracking (internal endpoint).
func (h *Handler) UnregisterActiveRide(c *gin.Context) {
	var req struct {
		DriverID string `json:"driver_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	driverID, err := uuid.Parse(req.DriverID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid driver_id")
		return
	}

	if h.service.etaTracker != nil {
		h.service.etaTracker.UnregisterActiveRide(c.Request.Context(), driverID)
	}

	common.SuccessResponse(c, gin.H{"message": "ride unregistered from ETA tracking"})
}

// UpdateDriverStatus handles updating driver availability status
func (h *Handler) UpdateDriverStatus(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required,oneof=available busy offline"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "status must be one of: available, busy, offline")
		return
	}

	ctx := c.Request.Context()

	// Validate eligibility when going online
	if req.Status == "available" {
		if err := h.service.ValidateDriverEligibility(ctx, driverID); err != nil {
			if appErr, ok := err.(*common.AppError); ok {
				common.AppErrorResponse(c, appErr)
				return
			}
			common.ErrorResponse(c, http.StatusForbidden, err.Error())
			return
		}

		// Track session start time
		if err := h.service.TrackDriverSessionStart(ctx, driverID); err != nil {
			// Log error but don't block - session tracking is non-critical
			common.ErrorResponse(c, http.StatusInternalServerError, "failed to track session start")
			return
		}
	}

	// Track session end and get summary when going offline
	var sessionSummary *DriverSessionSummary
	if req.Status == "offline" {
		summary, err := h.service.EndDriverSession(ctx, driverID)
		if err != nil {
			// Log but don't fail - session summary is nice-to-have
			// Continue with status update
		} else {
			sessionSummary = summary
		}
	}

	// Update status in Redis
	if err := h.service.SetDriverStatus(ctx, driverID, req.Status); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update status")
		return
	}

	// Build response
	response := gin.H{
		"status":     req.Status,
		"updated_at": time.Now(),
	}

	// Include session summary if going offline
	if sessionSummary != nil {
		response["session_summary"] = sessionSummary
	}

	common.SuccessResponse(c, response)
}

// GetDriverStatus handles getting driver's current status
func (h *Handler) GetDriverStatus(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	status, err := h.service.GetDriverStatus(c.Request.Context(), driverID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get status")
		return
	}

	common.SuccessResponse(c, gin.H{
		"status": status,
	})
}

// RegisterRoutes registers geo routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))

	geo := api.Group("/geo")
	{
		// Driver location updates (drivers only)
		geo.POST("/location", middleware.RequireRole(models.RoleDriver), h.UpdateLocation)
		geo.GET("/drivers/:id/location", h.GetDriverLocation)

		// Nearby drivers search
		geo.GET("/drivers/nearby", h.FindNearbyDrivers)

		// Utility endpoints
		geo.POST("/distance", h.CalculateDistance)

		// H3 spatial indexing endpoints
		geo.GET("/h3/cell", h.GetH3Cell)
		geo.GET("/h3/surge", h.GetSurgeInfo)
		geo.GET("/h3/demand-heatmap", h.GetDemandHeatmap)

		// Geocoding endpoints
		geo.GET("/geocode", h.ForwardGeocode)
		geo.GET("/geocode/reverse", h.ReverseGeocode)
		geo.GET("/geocode/autocomplete", h.PlaceAutocomplete)
		geo.GET("/geocode/place", h.GetPlaceDetails)
	}

	// Driver status management
	driver := api.Group("/driver")
	driver.Use(middleware.RequireRole(models.RoleDriver))
	{
		driver.POST("/status", h.UpdateDriverStatus)
		driver.GET("/status", h.GetDriverStatus)
	}

	// Internal service-to-service endpoints (no JWT required)
	internal := r.Group("/api/v1/internal/eta")
	{
		internal.POST("/register", h.RegisterActiveRide)
		internal.POST("/unregister", h.UnregisterActiveRide)
	}
}
