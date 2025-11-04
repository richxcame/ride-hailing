package favorites

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	userID, _ := c.Get("user_id")

	var req struct {
		Name      string  `json:"name" binding:"required"`
		Address   string  `json:"address" binding:"required"`
		Latitude  float64 `json:"latitude" binding:"required"`
		Longitude float64 `json:"longitude" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	favorite, err := h.service.CreateFavoriteLocation(
		c.Request.Context(),
		userID.(uuid.UUID),
		req.Name,
		req.Address,
		req.Latitude,
		req.Longitude,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, favorite)
}

// GetFavorites retrieves all favorite locations for the user
func (h *Handler) GetFavorites(c *gin.Context) {
	userID, _ := c.Get("user_id")

	favorites, err := h.service.GetFavoriteLocations(
		c.Request.Context(),
		userID.(uuid.UUID),
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"favorites": favorites})
}

// GetFavorite retrieves a specific favorite location
func (h *Handler) GetFavorite(c *gin.Context) {
	userID, _ := c.Get("user_id")
	favoriteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid favorite ID"})
		return
	}

	favorite, err := h.service.GetFavoriteLocation(
		c.Request.Context(),
		favoriteID,
		userID.(uuid.UUID),
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, favorite)
}

// UpdateFavorite updates a favorite location
func (h *Handler) UpdateFavorite(c *gin.Context) {
	userID, _ := c.Get("user_id")
	favoriteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid favorite ID"})
		return
	}

	var req struct {
		Name      string  `json:"name" binding:"required"`
		Address   string  `json:"address" binding:"required"`
		Latitude  float64 `json:"latitude" binding:"required"`
		Longitude float64 `json:"longitude" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	favorite, err := h.service.UpdateFavoriteLocation(
		c.Request.Context(),
		favoriteID,
		userID.(uuid.UUID),
		req.Name,
		req.Address,
		req.Latitude,
		req.Longitude,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, favorite)
}

// DeleteFavorite deletes a favorite location
func (h *Handler) DeleteFavorite(c *gin.Context) {
	userID, _ := c.Get("user_id")
	favoriteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid favorite ID"})
		return
	}

	err = h.service.DeleteFavoriteLocation(
		c.Request.Context(),
		favoriteID,
		userID.(uuid.UUID),
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Favorite location deleted"})
}
