package loyalty

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/richxcame/ride-hailing/pkg/pagination"
)

// Handler handles HTTP requests for loyalty
type Handler struct {
	service *Service
}

// NewHandler creates a new loyalty handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// RIDER LOYALTY ENDPOINTS
// ========================================

// GetStatus gets the rider's loyalty status
// GET /api/v1/rider/loyalty/status
func (h *Handler) GetStatus(c *gin.Context) {
	riderID, err := h.getRiderID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	status, err := h.service.GetLoyaltyStatus(c.Request.Context(), riderID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get loyalty status")
		return
	}

	common.SuccessResponse(c, status)
}

// GetPointsHistory gets the rider's points history
// GET /api/v1/rider/loyalty/points/history
func (h *Handler) GetPointsHistory(c *gin.Context) {
	riderID, err := h.getRiderID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	params := pagination.ParseParams(c)

	history, err := h.service.GetPointsHistory(c.Request.Context(), riderID, params.Limit, params.Offset)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get points history")
		return
	}

	common.SuccessResponse(c, history)
}

// GetRewards gets available rewards
// GET /api/v1/rider/loyalty/rewards
func (h *Handler) GetRewards(c *gin.Context) {
	riderID, err := h.getRiderID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rewards, err := h.service.GetRewardsCatalog(c.Request.Context(), riderID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get rewards")
		return
	}

	common.SuccessResponse(c, gin.H{
		"rewards": rewards,
	})
}

// RedeemReward redeems points for a reward
// POST /api/v1/rider/loyalty/rewards/:id/redeem
func (h *Handler) RedeemReward(c *gin.Context) {
	riderID, err := h.getRiderID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rewardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid reward ID")
		return
	}

	result, err := h.service.RedeemPoints(c.Request.Context(), &RedeemPointsRequest{
		RiderID:  riderID,
		RewardID: rewardID,
	})
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to redeem reward")
		return
	}

	common.SuccessResponse(c, result)
}

// GetChallenges gets active challenges
// GET /api/v1/rider/loyalty/challenges
func (h *Handler) GetChallenges(c *gin.Context) {
	riderID, err := h.getRiderID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	challenges, err := h.service.GetActiveChallenges(c.Request.Context(), riderID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get challenges")
		return
	}

	common.SuccessResponse(c, challenges)
}

// GetTiers gets all loyalty tiers
// GET /api/v1/rider/loyalty/tiers
func (h *Handler) GetTiers(c *gin.Context) {
	tiers, err := h.service.GetAllTiers(c.Request.Context())
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get tiers")
		return
	}

	common.SuccessResponse(c, gin.H{
		"tiers": tiers,
	})
}

// ========================================
// ADMIN ENDPOINTS
// ========================================

// GetLoyaltyStats gets loyalty program statistics
// GET /api/v1/admin/loyalty/stats
func (h *Handler) GetLoyaltyStats(c *gin.Context) {
	stats, err := h.service.GetLoyaltyStats(c.Request.Context())
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get loyalty stats")
		return
	}

	common.SuccessResponse(c, stats)
}

// AwardPoints awards points to a rider (admin)
// POST /api/v1/admin/loyalty/award
func (h *Handler) AwardPoints(c *gin.Context) {
	var req struct {
		RiderID     uuid.UUID `json:"rider_id" binding:"required"`
		Points      int       `json:"points" binding:"required,gt=0"`
		Description string    `json:"description" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.service.EarnPoints(c.Request.Context(), &EarnPointsRequest{
		RiderID:     req.RiderID,
		Points:      req.Points,
		Source:      SourcePromotion,
		Description: req.Description,
	})
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to award points")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "points awarded successfully",
	})
}

// ========================================
// HELPER FUNCTIONS
// ========================================

// getRiderID gets the rider ID from the authenticated user
func (h *Handler) getRiderID(c *gin.Context) (uuid.UUID, error) {
	return middleware.GetUserID(c)
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers loyalty routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	// Rider loyalty routes
	loyalty := r.Group("/api/v1/rider/loyalty")
	loyalty.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		loyalty.GET("/status", h.GetStatus)
		loyalty.GET("/points/history", h.GetPointsHistory)
		loyalty.GET("/rewards", h.GetRewards)
		loyalty.POST("/rewards/:id/redeem", h.RedeemReward)
		loyalty.GET("/challenges", h.GetChallenges)
		loyalty.GET("/tiers", h.GetTiers)
	}

	// Admin routes
	adminLoyalty := r.Group("/api/v1/admin/loyalty")
	adminLoyalty.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	adminLoyalty.Use(middleware.RequireRole(models.RoleAdmin))
	{
		adminLoyalty.GET("/stats", h.GetLoyaltyStats)
		adminLoyalty.POST("/award", h.AwardPoints)
	}
}

// RegisterRoutesOnGroup registers loyalty routes on an existing router group
func (h *Handler) RegisterRoutesOnGroup(rg *gin.RouterGroup) {
	loyalty := rg.Group("/rider/loyalty")
	{
		loyalty.GET("/status", h.GetStatus)
		loyalty.GET("/points/history", h.GetPointsHistory)
		loyalty.GET("/rewards", h.GetRewards)
		loyalty.POST("/rewards/:id/redeem", h.RedeemReward)
		loyalty.GET("/challenges", h.GetChallenges)
		loyalty.GET("/tiers", h.GetTiers)
	}
}
