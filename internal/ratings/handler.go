package ratings

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// Handler handles HTTP requests for ratings
type Handler struct {
	service *Service
}

// NewHandler creates a new ratings handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// RIDER ENDPOINTS
// ========================================

// RateDriver rates a driver after a ride
// POST /api/v1/ratings/driver
func (h *Handler) RateDriver(c *gin.Context) {
	riderID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SubmitRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get driver ID from query param (in production, would look up from ride)
	driverIDStr := c.Query("driver_id")
	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "driver_id query parameter required")
		return
	}

	rating, err := h.service.SubmitRating(c.Request.Context(), riderID, driverID, RaterTypeRider, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to submit rating")
		return
	}

	common.CreatedResponse(c, rating)
}

// GetMyRiderProfile returns the rider's rating profile
// GET /api/v1/ratings/me
func (h *Handler) GetMyProfile(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	profile, err := h.service.GetMyRatingProfile(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get rating profile")
		return
	}

	common.SuccessResponse(c, profile)
}

// GetUserRating returns a user's public rating
// GET /api/v1/ratings/users/:userId
func (h *Handler) GetUserRating(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user id")
		return
	}

	profile, err := h.service.GetUserRating(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get user rating")
		return
	}

	common.SuccessResponse(c, profile)
}

// GetMyRatingsGiven returns ratings the user has given
// GET /api/v1/ratings/given?limit=20&offset=0
func (h *Handler) GetMyRatingsGiven(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	ratings, total, err := h.service.GetRatingsGiven(c.Request.Context(), userID, limit, offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get ratings")
		return
	}

	common.SuccessResponse(c, gin.H{
		"ratings": ratings,
		"total":   total,
	})
}

// RespondToRating responds to a received rating
// POST /api/v1/ratings/:ratingId/respond
func (h *Handler) RespondToRating(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	ratingID, err := uuid.Parse(c.Param("ratingId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid rating id")
		return
	}

	var req RespondToRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.RespondToRating(c.Request.Context(), userID, ratingID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to respond to rating")
		return
	}

	common.CreatedResponse(c, resp)
}

// GetSuggestedTags returns suggested rating tags
// GET /api/v1/ratings/tags?type=rider
func (h *Handler) GetSuggestedTags(c *gin.Context) {
	raterType := RaterType(c.DefaultQuery("type", "rider"))
	tags := h.service.GetSuggestedTags(raterType)
	common.SuccessResponse(c, gin.H{"tags": tags})
}

// ========================================
// DRIVER ENDPOINTS
// ========================================

// RateRider rates a rider after a ride
// POST /api/v1/driver/ratings/rider
func (h *Handler) RateRider(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SubmitRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	riderIDStr := c.Query("rider_id")
	riderID, err := uuid.Parse(riderIDStr)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "rider_id query parameter required")
		return
	}

	rating, err := h.service.SubmitRating(c.Request.Context(), driverID, riderID, RaterTypeDriver, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to submit rating")
		return
	}

	common.CreatedResponse(c, rating)
}

// GetDriverProfile returns the driver's rating profile
// GET /api/v1/driver/ratings/me
func (h *Handler) GetDriverProfile(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	profile, err := h.service.GetMyRatingProfile(c.Request.Context(), driverID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get rating profile")
		return
	}

	common.SuccessResponse(c, profile)
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers ratings routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	// General ratings (authenticated users)
	ratings := r.Group("/api/v1/ratings")
	ratings.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		ratings.POST("/driver", h.RateDriver)
		ratings.GET("/me", h.GetMyProfile)
		ratings.GET("/users/:userId", h.GetUserRating)
		ratings.GET("/given", h.GetMyRatingsGiven)
		ratings.POST("/:ratingId/respond", h.RespondToRating)
		ratings.GET("/tags", h.GetSuggestedTags)
	}

	// Driver-specific ratings
	driverRatings := r.Group("/api/v1/driver/ratings")
	driverRatings.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	driverRatings.Use(middleware.RequireRole(models.RoleDriver))
	{
		driverRatings.POST("/rider", h.RateRider)
		driverRatings.GET("/me", h.GetDriverProfile)
	}
}
