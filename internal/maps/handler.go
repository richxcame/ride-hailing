package maps

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for maps functionality
type Handler struct {
	service *Service
}

// NewHandler creates a new maps handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all maps routes
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	maps := rg.Group("/maps")
	{
		// Routing endpoints
		maps.POST("/route", h.GetRoute)
		maps.POST("/eta", h.GetETA)
		maps.POST("/distance-matrix", h.GetDistanceMatrix)

		// Traffic endpoints
		maps.POST("/traffic/flow", h.GetTrafficFlow)
		maps.POST("/traffic/incidents", h.GetTrafficIncidents)

		// Geocoding endpoints
		maps.POST("/geocode", h.Geocode)
		maps.POST("/reverse-geocode", h.ReverseGeocode)

		// Places endpoints
		maps.POST("/places/search", h.SearchPlaces)

		// Road endpoints
		maps.POST("/roads/snap", h.SnapToRoad)
		maps.POST("/roads/speed-limits", h.GetSpeedLimits)

		// Utility endpoints
		maps.GET("/health", h.HealthCheck)
		maps.POST("/waypoints/optimize", h.OptimizeWaypoints)
		maps.POST("/eta/batch", h.BatchGetETA)
	}
}

// GetRoute handles route calculation requests
// @Summary Calculate route between two points
// @Description Returns detailed route information including distance, duration, and turn-by-turn directions
// @Tags Maps
// @Accept json
// @Produce json
// @Param request body RouteRequest true "Route request"
// @Success 200 {object} RouteResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/maps/route [post]
func (h *Handler) GetRoute(c *gin.Context) {
	var req RouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate coordinates
	if !isValidCoordinate(req.Origin) || !isValidCoordinate(req.Destination) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coordinates"})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.GetRoute(ctx, &req)
	if err != nil {
		logger.Error("Failed to get route", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate route"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetETA handles ETA calculation requests
// @Summary Calculate ETA between two points
// @Description Returns estimated time of arrival with traffic information
// @Tags Maps
// @Accept json
// @Produce json
// @Param request body ETARequest true "ETA request"
// @Success 200 {object} ETAResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/maps/eta [post]
func (h *Handler) GetETA(c *gin.Context) {
	var req ETARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if !isValidCoordinate(req.Origin) || !isValidCoordinate(req.Destination) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coordinates"})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.GetTrafficAwareETA(ctx, req.Origin, req.Destination)
	if err != nil {
		logger.Error("Failed to get ETA", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate ETA"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// BatchGetETA handles batch ETA calculation requests
// @Summary Calculate ETAs for multiple origin-destination pairs
// @Description Returns estimated times of arrival for multiple routes
// @Tags Maps
// @Accept json
// @Produce json
// @Param request body []ETARequest true "Batch ETA request"
// @Success 200 {array} ETAResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/maps/eta/batch [post]
func (h *Handler) BatchGetETA(c *gin.Context) {
	var requests []*ETARequest
	if err := c.ShouldBindJSON(&requests); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if len(requests) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "maximum 100 requests per batch"})
		return
	}

	// Validate all coordinates
	for i, req := range requests {
		if !isValidCoordinate(req.Origin) || !isValidCoordinate(req.Destination) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coordinates at index " + strconv.Itoa(i)})
			return
		}
	}

	ctx := c.Request.Context()
	responses, err := h.service.BatchGetETA(ctx, requests)
	if err != nil {
		logger.Error("Failed to batch get ETA", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate ETAs"})
		return
	}

	c.JSON(http.StatusOK, responses)
}

// GetDistanceMatrix handles distance matrix requests
// @Summary Calculate distances between multiple points
// @Description Returns a matrix of distances and durations between origins and destinations
// @Tags Maps
// @Accept json
// @Produce json
// @Param request body DistanceMatrixRequest true "Distance matrix request"
// @Success 200 {object} DistanceMatrixResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/maps/distance-matrix [post]
func (h *Handler) GetDistanceMatrix(c *gin.Context) {
	var req DistanceMatrixRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if len(req.Origins) == 0 || len(req.Destinations) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "origins and destinations are required"})
		return
	}

	if len(req.Origins)*len(req.Destinations) > 625 { // 25x25 max
		c.JSON(http.StatusBadRequest, gin.H{"error": "maximum 625 elements (25 origins x 25 destinations)"})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.GetDistanceMatrix(ctx, &req)
	if err != nil {
		logger.Error("Failed to get distance matrix", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate distance matrix"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetTrafficFlow handles traffic flow requests
// @Summary Get traffic flow information
// @Description Returns current traffic conditions in a specified area
// @Tags Maps
// @Accept json
// @Produce json
// @Param request body TrafficFlowRequest true "Traffic flow request"
// @Success 200 {object} TrafficFlowResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/maps/traffic/flow [post]
func (h *Handler) GetTrafficFlow(c *gin.Context) {
	var req TrafficFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.GetTrafficFlow(ctx, &req)
	if err != nil {
		logger.Error("Failed to get traffic flow", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get traffic data"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetTrafficIncidents handles traffic incidents requests
// @Summary Get traffic incidents
// @Description Returns traffic incidents in a specified area
// @Tags Maps
// @Accept json
// @Produce json
// @Param request body TrafficIncidentsRequest true "Traffic incidents request"
// @Success 200 {object} TrafficIncidentsResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/maps/traffic/incidents [post]
func (h *Handler) GetTrafficIncidents(c *gin.Context) {
	var req TrafficIncidentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.GetTrafficIncidents(ctx, &req)
	if err != nil {
		logger.Error("Failed to get traffic incidents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get traffic incidents"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Geocode handles geocoding requests
// @Summary Convert address to coordinates
// @Description Converts an address to geographic coordinates
// @Tags Maps
// @Accept json
// @Produce json
// @Param request body GeocodingRequest true "Geocoding request"
// @Success 200 {object} GeocodingResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/maps/geocode [post]
func (h *Handler) Geocode(c *gin.Context) {
	var req GeocodingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if req.Address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address is required"})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.Geocode(ctx, &req)
	if err != nil {
		logger.Error("Failed to geocode", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to geocode address"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ReverseGeocode handles reverse geocoding requests
// @Summary Convert coordinates to address
// @Description Converts geographic coordinates to an address
// @Tags Maps
// @Accept json
// @Produce json
// @Param request body GeocodingRequest true "Reverse geocoding request"
// @Success 200 {object} GeocodingResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/maps/reverse-geocode [post]
func (h *Handler) ReverseGeocode(c *gin.Context) {
	var req GeocodingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if req.Coordinate == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "coordinate is required"})
		return
	}

	if !isValidCoordinate(*req.Coordinate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coordinate"})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.ReverseGeocode(ctx, &req)
	if err != nil {
		logger.Error("Failed to reverse geocode", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reverse geocode"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// SearchPlaces handles place search requests
// @Summary Search for places
// @Description Search for places/POIs near a location
// @Tags Maps
// @Accept json
// @Produce json
// @Param request body PlaceSearchRequest true "Place search request"
// @Success 200 {object} PlaceSearchResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/maps/places/search [post]
func (h *Handler) SearchPlaces(c *gin.Context) {
	var req PlaceSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if req.Query == "" && req.Location == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query or location is required"})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.SearchPlaces(ctx, &req)
	if err != nil {
		logger.Error("Failed to search places", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search places"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// SnapToRoad handles snap-to-road requests
// @Summary Snap GPS points to roads
// @Description Snaps a path of GPS coordinates to the nearest roads
// @Tags Maps
// @Accept json
// @Produce json
// @Param request body SnapToRoadRequest true "Snap to road request"
// @Success 200 {object} SnapToRoadResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/maps/roads/snap [post]
func (h *Handler) SnapToRoad(c *gin.Context) {
	var req SnapToRoadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if len(req.Path) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	if len(req.Path) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "maximum 100 points per request"})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.SnapToRoad(ctx, &req)
	if err != nil {
		logger.Error("Failed to snap to road", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to snap to road"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetSpeedLimits handles speed limits requests
// @Summary Get speed limits
// @Description Returns speed limits along a path or for specific place IDs
// @Tags Maps
// @Accept json
// @Produce json
// @Param request body SpeedLimitsRequest true "Speed limits request"
// @Success 200 {object} SpeedLimitsResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/maps/roads/speed-limits [post]
func (h *Handler) GetSpeedLimits(c *gin.Context) {
	var req SpeedLimitsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if len(req.Path) == 0 && len(req.PlaceIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path or place_ids is required"})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.GetSpeedLimits(ctx, &req)
	if err != nil {
		logger.Error("Failed to get speed limits", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get speed limits"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// OptimizeWaypoints handles waypoint optimization requests
// @Summary Optimize waypoint order
// @Description Returns the optimal order to visit waypoints
// @Tags Maps
// @Accept json
// @Produce json
// @Param request body OptimizeWaypointsRequest true "Optimize waypoints request"
// @Success 200 {object} OptimizeWaypointsResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/maps/waypoints/optimize [post]
func (h *Handler) OptimizeWaypoints(c *gin.Context) {
	var req OptimizeWaypointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if len(req.Waypoints) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "waypoints are required"})
		return
	}

	if len(req.Waypoints) > 23 { // Max waypoints for Google Maps
		c.JSON(http.StatusBadRequest, gin.H{"error": "maximum 23 waypoints"})
		return
	}

	ctx := c.Request.Context()
	order, err := h.service.OptimizeWaypoints(ctx, req.Origin, req.Waypoints, req.Destination)
	if err != nil {
		logger.Error("Failed to optimize waypoints", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to optimize waypoints"})
		return
	}

	// Reorder waypoints
	optimizedWaypoints := make([]Coordinate, len(order))
	for i, idx := range order {
		optimizedWaypoints[i] = req.Waypoints[idx]
	}

	c.JSON(http.StatusOK, OptimizeWaypointsResponse{
		Order:              order,
		OptimizedWaypoints: optimizedWaypoints,
	})
}

// HealthCheck handles health check requests
// @Summary Check maps service health
// @Description Returns health status of all configured maps providers
// @Tags Maps
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/maps/health [get]
func (h *Handler) HealthCheck(c *gin.Context) {
	ctx := c.Request.Context()
	results := h.service.HealthCheck(ctx)

	status := "healthy"
	errors := make(map[string]string)

	for provider, err := range results {
		if err != nil {
			status = "degraded"
			errors[string(provider)] = err.Error()
		}
	}

	// If primary is down, status is unhealthy
	if err, ok := results[h.service.GetPrimaryProvider()]; ok && err != nil {
		status = "unhealthy"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   status,
		"primary":  h.service.GetPrimaryProvider(),
		"errors":   errors,
		"providers": results,
	})
}

// Request/Response types for handlers

// OptimizeWaypointsRequest represents a waypoint optimization request
type OptimizeWaypointsRequest struct {
	Origin      Coordinate   `json:"origin" binding:"required"`
	Waypoints   []Coordinate `json:"waypoints" binding:"required"`
	Destination Coordinate   `json:"destination" binding:"required"`
}

// OptimizeWaypointsResponse represents the waypoint optimization result
type OptimizeWaypointsResponse struct {
	Order              []int        `json:"order"`
	OptimizedWaypoints []Coordinate `json:"optimized_waypoints"`
}

// Validation helpers

func isValidCoordinate(c Coordinate) bool {
	return c.Latitude >= -90 && c.Latitude <= 90 &&
		c.Longitude >= -180 && c.Longitude <= 180
}

// RegisterInternalRoutes registers internal routes (no auth required)
func (h *Handler) RegisterInternalRoutes(rg *gin.RouterGroup) {
	internal := rg.Group("/internal/maps")
	{
		// Quick ETA for internal services
		internal.GET("/eta", func(c *gin.Context) {
			originLat, _ := strconv.ParseFloat(c.Query("origin_lat"), 64)
			originLng, _ := strconv.ParseFloat(c.Query("origin_lng"), 64)
			destLat, _ := strconv.ParseFloat(c.Query("dest_lat"), 64)
			destLng, _ := strconv.ParseFloat(c.Query("dest_lng"), 64)

			if originLat == 0 || originLng == 0 || destLat == 0 || destLng == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coordinates"})
				return
			}

			req := &ETARequest{
				Origin:      Coordinate{Latitude: originLat, Longitude: originLng},
				Destination: Coordinate{Latitude: destLat, Longitude: destLng},
			}

			ctx := c.Request.Context()
			resp, err := h.service.GetETA(ctx, req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, resp)
		})

		// Traffic level for a location
		internal.GET("/traffic", func(c *gin.Context) {
			lat, _ := strconv.ParseFloat(c.Query("lat"), 64)
			lng, _ := strconv.ParseFloat(c.Query("lng"), 64)

			if lat == 0 || lng == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coordinates"})
				return
			}

			req := &TrafficFlowRequest{
				Location:     Coordinate{Latitude: lat, Longitude: lng},
				RadiusMeters: 5000,
			}

			ctx := c.Request.Context()
			resp, err := h.service.GetTrafficFlow(ctx, req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"traffic_level": resp.OverallLevel,
				"updated_at":    resp.UpdatedAt,
			})
		})
	}
}

// RegisterAdminRoutes registers admin routes
func (h *Handler) RegisterAdminRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	admin := rg.Group("/admin/maps")
	admin.Use(authMiddleware)
	admin.Use(middleware.RequireRole("admin"))
	{
		admin.GET("/stats", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"primary_provider": h.service.GetPrimaryProvider(),
				"fallback_count":   len(h.service.fallbacks),
				"cache_enabled":    h.service.config.CacheEnabled,
				"traffic_enabled":  h.service.config.TrafficEnabled,
			})
		})
	}
}
