package favorites

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
)

// Handler handles HTTP requests for favorite locations
type Handler struct {
	service *Service
}

// NewHandler creates a new favorites handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// CreateFavorite creates a new favorite location
func (h *Handler) CreateFavorite(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		common.ErrorResponse(c, http.StatusInternalServerError, "invalid user context")
		return
	}

	var req struct {
		Name      string  `json:"name" binding:"required"`
		Address   string  `json:"address" binding:"required"`
		Latitude  float64 `json:"latitude" binding:"required"`
		Longitude float64 `json:"longitude" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	favorite, err := h.service.CreateFavoriteLocation(
		c.Request.Context(),
		userUUID,
		req.Name,
		req.Address,
		req.Latitude,
		req.Longitude,
	)

	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to create favorite location")
		return
	}

	common.CreatedResponse(c, favorite)
}

// GetFavorites retrieves all favorite locations for the user
func (h *Handler) GetFavorites(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		common.ErrorResponse(c, http.StatusInternalServerError, "invalid user context")
		return
	}

	favorites, err := h.service.GetFavoriteLocations(
		c.Request.Context(),
		userUUID,
	)

	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch favorite locations")
		return
	}

	common.SuccessResponse(c, gin.H{"favorites": favorites})
}

// GetFavorite retrieves a specific favorite location
func (h *Handler) GetFavorite(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		common.ErrorResponse(c, http.StatusInternalServerError, "invalid user context")
		return
	}

	favoriteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid favorite ID")
		return
	}

	favorite, err := h.service.GetFavoriteLocation(
		c.Request.Context(),
		favoriteID,
		userUUID,
	)

	if err != nil {
		common.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	common.SuccessResponse(c, favorite)
}

// UpdateFavorite updates a favorite location
func (h *Handler) UpdateFavorite(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		common.ErrorResponse(c, http.StatusInternalServerError, "invalid user context")
		return
	}

	favoriteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid favorite ID")
		return
	}

	var req struct {
		Name      string  `json:"name" binding:"required"`
		Address   string  `json:"address" binding:"required"`
		Latitude  float64 `json:"latitude" binding:"required"`
		Longitude float64 `json:"longitude" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	favorite, err := h.service.UpdateFavoriteLocation(
		c.Request.Context(),
		favoriteID,
		userUUID,
		req.Name,
		req.Address,
		req.Latitude,
		req.Longitude,
	)

	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to update favorite location")
		return
	}

	common.SuccessResponse(c, favorite)
}

// DeleteFavorite deletes a favorite location
func (h *Handler) DeleteFavorite(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		common.ErrorResponse(c, http.StatusInternalServerError, "invalid user context")
		return
	}

	favoriteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid favorite ID")
		return
	}

	err = h.service.DeleteFavoriteLocation(
		c.Request.Context(),
		favoriteID,
		userUUID,
	)

	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete favorite location")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Favorite location deleted"})
}
