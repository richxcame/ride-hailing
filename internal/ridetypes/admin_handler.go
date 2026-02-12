package ridetypes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/pagination"
)

// AdminHandler handles admin HTTP requests for ride type management
type AdminHandler struct {
	service *Service
}

// NewAdminHandler creates a new ride types admin handler
func NewAdminHandler(service *Service) *AdminHandler {
	return &AdminHandler{service: service}
}

// RegisterRoutes registers ride type admin routes
func (h *AdminHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rt := rg.Group("/ride-types")
	{
		rt.GET("", h.ListRideTypes)
		rt.POST("", h.CreateRideType)
		rt.GET("/:id", h.GetRideType)
		rt.PUT("/:id", h.UpdateRideType)
		rt.DELETE("/:id", h.DeleteRideType)
	}

	// Country ride type availability
	countryRT := rg.Group("/countries/:countryId/ride-types")
	{
		countryRT.GET("", h.ListCountryRideTypes)
		countryRT.POST("", h.AddRideTypeToCountry)
		countryRT.PUT("/:rideTypeId", h.UpdateCountryRideType)
		countryRT.DELETE("/:rideTypeId", h.RemoveRideTypeFromCountry)
	}

	// City ride type availability (overrides country-level)
	crt := rg.Group("/cities/:cityId/ride-types")
	{
		crt.GET("", h.ListCityRideTypes)
		crt.POST("", h.AddRideTypeToCity)
		crt.PUT("/:rideTypeId", h.UpdateCityRideType)
		crt.DELETE("/:rideTypeId", h.RemoveRideTypeFromCity)
	}
}

// ListRideTypes lists all ride types
func (h *AdminHandler) ListRideTypes(c *gin.Context) {
	params := pagination.ParseParams(c)
	includeInactive := c.Query("include_inactive") == "true"

	items, total, err := h.service.ListRideTypes(c.Request.Context(), params.Limit, params.Offset, includeInactive)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch ride types")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, total)
	common.SuccessResponseWithMeta(c, items, meta)
}

// CreateRideType creates a new ride type
func (h *AdminHandler) CreateRideType(c *gin.Context) {
	var req CreateRideTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	rt := &RideType{
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
		Capacity:    req.Capacity,
		SortOrder:   req.SortOrder,
		IsActive:    req.IsActive,
	}

	if err := h.service.CreateRideType(c.Request.Context(), rt); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to create ride type")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusCreated, rt, "Ride type created successfully")
}

// GetRideType retrieves a ride type by ID
func (h *AdminHandler) GetRideType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid ride type ID")
		return
	}

	rt, err := h.service.GetRideTypeByID(c.Request.Context(), id)
	if err != nil {
		common.ErrorResponse(c, http.StatusNotFound, "Ride type not found")
		return
	}

	common.SuccessResponse(c, rt)
}

// UpdateRideType updates a ride type
func (h *AdminHandler) UpdateRideType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid ride type ID")
		return
	}

	rt, err := h.service.GetRideTypeByID(c.Request.Context(), id)
	if err != nil {
		common.ErrorResponse(c, http.StatusNotFound, "Ride type not found")
		return
	}

	var req UpdateRideTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.Name != nil {
		rt.Name = *req.Name
	}
	if req.Description != nil {
		rt.Description = req.Description
	}
	if req.Icon != nil {
		rt.Icon = req.Icon
	}
	if req.Capacity != nil {
		rt.Capacity = *req.Capacity
	}
	if req.SortOrder != nil {
		rt.SortOrder = *req.SortOrder
	}
	if req.IsActive != nil {
		rt.IsActive = *req.IsActive
	}

	if err := h.service.UpdateRideType(c.Request.Context(), rt); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to update ride type")
		return
	}

	common.SuccessResponse(c, rt)
}

// DeleteRideType soft-deletes a ride type
func (h *AdminHandler) DeleteRideType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid ride type ID")
		return
	}

	if err := h.service.DeleteRideType(c.Request.Context(), id); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete ride type")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "Ride type deleted successfully")
}

// --- Country Ride Types ---

// ListCountryRideTypes lists ride types available in a country
func (h *AdminHandler) ListCountryRideTypes(c *gin.Context) {
	countryID, err := uuid.Parse(c.Param("countryId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid country ID")
		return
	}

	includeInactive := c.Query("include_inactive") == "true"
	items, err := h.service.ListCountryRideTypes(c.Request.Context(), countryID, includeInactive)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch country ride types")
		return
	}

	common.SuccessResponse(c, items)
}

// AddRideTypeToCountry adds a ride type to a country
func (h *AdminHandler) AddRideTypeToCountry(c *gin.Context) {
	countryID, err := uuid.Parse(c.Param("countryId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid country ID")
		return
	}

	var req CountryRideTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	crt := &CountryRideType{
		CountryID:  countryID,
		RideTypeID: req.RideTypeID,
		IsActive:   req.IsActive,
		SortOrder:  req.SortOrder,
	}

	if err := h.service.AddRideTypeToCountry(c.Request.Context(), crt); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to add ride type to country")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusCreated, crt, "Ride type added to country successfully")
}

// UpdateCountryRideType updates a country ride type mapping
func (h *AdminHandler) UpdateCountryRideType(c *gin.Context) {
	countryID, err := uuid.Parse(c.Param("countryId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid country ID")
		return
	}

	rideTypeID, err := uuid.Parse(c.Param("rideTypeId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid ride type ID")
		return
	}

	var req UpdateCountryRideTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	crt := &CountryRideType{
		CountryID:  countryID,
		RideTypeID: rideTypeID,
	}

	if req.IsActive != nil {
		crt.IsActive = *req.IsActive
	} else {
		crt.IsActive = true
	}
	if req.SortOrder != nil {
		crt.SortOrder = *req.SortOrder
	}

	if err := h.service.UpdateCountryRideType(c.Request.Context(), crt); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to update country ride type")
		return
	}

	common.SuccessResponse(c, crt)
}

// RemoveRideTypeFromCountry removes a ride type from a country
func (h *AdminHandler) RemoveRideTypeFromCountry(c *gin.Context) {
	countryID, err := uuid.Parse(c.Param("countryId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid country ID")
		return
	}

	rideTypeID, err := uuid.Parse(c.Param("rideTypeId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid ride type ID")
		return
	}

	if err := h.service.RemoveRideTypeFromCountry(c.Request.Context(), countryID, rideTypeID); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to remove ride type from country")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "Ride type removed from country successfully")
}

// --- City Ride Types ---

// ListCityRideTypes lists ride types available in a city
func (h *AdminHandler) ListCityRideTypes(c *gin.Context) {
	cityID, err := uuid.Parse(c.Param("cityId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid city ID")
		return
	}

	includeInactive := c.Query("include_inactive") == "true"
	items, err := h.service.ListCityRideTypes(c.Request.Context(), cityID, includeInactive)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch city ride types")
		return
	}

	common.SuccessResponse(c, items)
}

// AddRideTypeToCity adds a ride type to a city
func (h *AdminHandler) AddRideTypeToCity(c *gin.Context) {
	cityID, err := uuid.Parse(c.Param("cityId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid city ID")
		return
	}

	var req CityRideTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	crt := &CityRideType{
		CityID:     cityID,
		RideTypeID: req.RideTypeID,
		IsActive:   req.IsActive,
		SortOrder:  req.SortOrder,
	}

	if err := h.service.AddRideTypeToCity(c.Request.Context(), crt); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to add ride type to city")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusCreated, crt, "Ride type added to city successfully")
}

// UpdateCityRideType updates a city ride type mapping
func (h *AdminHandler) UpdateCityRideType(c *gin.Context) {
	cityID, err := uuid.Parse(c.Param("cityId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid city ID")
		return
	}

	rideTypeID, err := uuid.Parse(c.Param("rideTypeId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid ride type ID")
		return
	}

	var req UpdateCityRideTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	crt := &CityRideType{
		CityID:     cityID,
		RideTypeID: rideTypeID,
	}

	// Get current values to merge partial update
	// For simplicity, default to true/0 if not provided
	if req.IsActive != nil {
		crt.IsActive = *req.IsActive
	} else {
		crt.IsActive = true
	}
	if req.SortOrder != nil {
		crt.SortOrder = *req.SortOrder
	}

	if err := h.service.UpdateCityRideType(c.Request.Context(), crt); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to update city ride type")
		return
	}

	common.SuccessResponse(c, crt)
}

// RemoveRideTypeFromCity removes a ride type from a city
func (h *AdminHandler) RemoveRideTypeFromCity(c *gin.Context) {
	cityID, err := uuid.Parse(c.Param("cityId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid city ID")
		return
	}

	rideTypeID, err := uuid.Parse(c.Param("rideTypeId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid ride type ID")
		return
	}

	if err := h.service.RemoveRideTypeFromCity(c.Request.Context(), cityID, rideTypeID); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to remove ride type from city")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "Ride type removed from city successfully")
}
