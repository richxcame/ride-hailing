package geography

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
)

// Handler handles HTTP requests for geography
type Handler struct {
	service *Service
}

// NewHandler creates a new geography handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetCountries returns all active countries
func (h *Handler) GetCountries(c *gin.Context) {
	countries, err := h.service.GetActiveCountries(c.Request.Context())
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get countries")
		return
	}

	responses := make([]*CountryResponse, len(countries))
	for i, country := range countries {
		responses[i] = ToCountryResponse(country)
	}

	common.SuccessResponse(c, responses)
}

// GetCountry returns a country by code
func (h *Handler) GetCountry(c *gin.Context) {
	code := c.Param("code")
	if len(code) != 2 {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid country code")
		return
	}

	country, err := h.service.GetCountryByCode(c.Request.Context(), code)
	if err != nil {
		common.ErrorResponse(c, http.StatusNotFound, "country not found")
		return
	}

	common.SuccessResponse(c, ToCountryResponse(country))
}

// GetRegionsByCountry returns all regions for a country
func (h *Handler) GetRegionsByCountry(c *gin.Context) {
	code := c.Param("code")
	if len(code) != 2 {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid country code")
		return
	}

	regions, err := h.service.GetRegionsByCountryCode(c.Request.Context(), code)
	if err != nil {
		common.ErrorResponse(c, http.StatusNotFound, "country not found")
		return
	}

	responses := make([]*RegionResponse, len(regions))
	for i, region := range regions {
		responses[i] = ToRegionResponse(region)
	}

	common.SuccessResponse(c, responses)
}

// GetRegion returns a region by ID
func (h *Handler) GetRegion(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid region ID")
		return
	}

	region, err := h.service.GetRegionByID(c.Request.Context(), id)
	if err != nil {
		common.ErrorResponse(c, http.StatusNotFound, "region not found")
		return
	}

	common.SuccessResponse(c, ToRegionResponse(region))
}

// GetCitiesByRegion returns all cities for a region
func (h *Handler) GetCitiesByRegion(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid region ID")
		return
	}

	cities, err := h.service.GetCitiesByRegion(c.Request.Context(), id)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get cities")
		return
	}

	responses := make([]*CityResponse, len(cities))
	for i, city := range cities {
		responses[i] = ToCityResponse(city)
	}

	common.SuccessResponse(c, responses)
}

// GetCity returns a city by ID
func (h *Handler) GetCity(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid city ID")
		return
	}

	city, err := h.service.GetCityByID(c.Request.Context(), id)
	if err != nil {
		common.ErrorResponse(c, http.StatusNotFound, "city not found")
		return
	}

	common.SuccessResponse(c, ToCityResponse(city))
}

// GetPricingZones returns all pricing zones for a city
func (h *Handler) GetPricingZones(c *gin.Context) {
	cityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid city ID")
		return
	}

	zones, err := h.service.GetPricingZonesByCity(c.Request.Context(), cityID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get pricing zones")
		return
	}

	responses := make([]*PricingZoneResponse, len(zones))
	for i, zone := range zones {
		responses[i] = ToPricingZoneResponse(zone)
	}

	common.SuccessResponse(c, responses)
}

// ResolveLocation resolves a latitude/longitude to geographic hierarchy
func (h *Handler) ResolveLocation(c *gin.Context) {
	var req ResolveLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	resolved, err := h.service.ResolveLocation(c.Request.Context(), req.Latitude, req.Longitude)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to resolve location")
		return
	}

	common.SuccessResponse(c, ToResolveLocationResponse(resolved))
}

// CheckServiceability checks if a location is within an active service area
func (h *Handler) CheckServiceability(c *gin.Context) {
	var req ResolveLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	serviceable, resolved, err := h.service.IsLocationServiceable(c.Request.Context(), req.Latitude, req.Longitude)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to check serviceability")
		return
	}

	common.SuccessResponse(c, gin.H{
		"serviceable": serviceable,
		"location":    ToResolveLocationResponse(resolved),
	})
}

// RegisterRoutes registers geography routes
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	geo := rg.Group("/geography")
	{
		// Countries
		geo.GET("/countries", h.GetCountries)
		geo.GET("/countries/:code", h.GetCountry)
		geo.GET("/countries/:code/regions", h.GetRegionsByCountry)

		// Regions
		geo.GET("/regions/:id", h.GetRegion)
		geo.GET("/regions/:id/cities", h.GetCitiesByRegion)

		// Cities
		geo.GET("/cities/:id", h.GetCity)
		geo.GET("/cities/:id/zones", h.GetPricingZones)

		// Location
		geo.POST("/location/resolve", h.ResolveLocation)
		geo.POST("/location/serviceable", h.CheckServiceability)
	}
}
