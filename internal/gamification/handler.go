package gamification

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// Handler handles HTTP requests for gamification
type Handler struct {
	service *Service
}

// NewHandler creates a new gamification handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// DRIVER ENDPOINTS
// ========================================

// GetStatus gets the driver's gamification status
// GET /api/v1/driver/gamification/status
func (h *Handler) GetStatus(c *gin.Context) {
	driverID, err := h.getDriverID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	status, err := h.service.GetGamificationStatus(c.Request.Context(), driverID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get gamification status")
		return
	}

	common.SuccessResponse(c, status)
}

// GetQuests gets active quests for the driver
// GET /api/v1/driver/gamification/quests
func (h *Handler) GetQuests(c *gin.Context) {
	driverID, err := h.getDriverID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	quests, err := h.service.GetActiveQuestsWithProgress(c.Request.Context(), driverID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get quests")
		return
	}

	common.SuccessResponse(c, gin.H{
		"quests": quests,
	})
}

// JoinQuest joins a quest
// POST /api/v1/driver/gamification/quests/:id/join
func (h *Handler) JoinQuest(c *gin.Context) {
	driverID, err := h.getDriverID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	questID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid quest ID")
		return
	}

	if err := h.service.JoinQuest(c.Request.Context(), driverID, questID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to join quest")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Successfully joined quest",
	})
}

// ClaimQuestReward claims a quest reward
// POST /api/v1/driver/gamification/quests/:id/claim
func (h *Handler) ClaimQuestReward(c *gin.Context) {
	driverID, err := h.getDriverID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	questID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid quest ID")
		return
	}

	result, err := h.service.ClaimQuestReward(c.Request.Context(), &ClaimQuestRewardRequest{
		DriverID: driverID,
		QuestID:  questID,
	})
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to claim reward")
		return
	}

	common.SuccessResponse(c, result)
}

// GetAchievements gets driver's achievements
// GET /api/v1/driver/gamification/achievements
func (h *Handler) GetAchievements(c *gin.Context) {
	driverID, err := h.getDriverID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	achievements, err := h.service.GetDriverAchievements(c.Request.Context(), driverID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get achievements")
		return
	}

	common.SuccessResponse(c, gin.H{
		"achievements": achievements,
	})
}

// GetLeaderboard gets the leaderboard
// GET /api/v1/driver/gamification/leaderboard
func (h *Handler) GetLeaderboard(c *gin.Context) {
	driverID, err := h.getDriverID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	period := c.DefaultQuery("period", "weekly")
	category := c.DefaultQuery("category", "rides")

	leaderboard, err := h.service.GetLeaderboard(c.Request.Context(), driverID, period, category)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get leaderboard")
		return
	}

	common.SuccessResponse(c, leaderboard)
}

// GetTiers gets all driver tiers
// GET /api/v1/driver/gamification/tiers
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

// GetDriverStats gets gamification stats for a driver (admin)
// GET /api/v1/admin/gamification/driver/:id
func (h *Handler) GetDriverStats(c *gin.Context) {
	driverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid driver ID")
		return
	}

	status, err := h.service.GetGamificationStatus(c.Request.Context(), driverID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get driver stats")
		return
	}

	common.SuccessResponse(c, status)
}

// ResetWeeklyStats resets weekly stats (admin - scheduled job)
// POST /api/v1/admin/gamification/reset/weekly
func (h *Handler) ResetWeeklyStats(c *gin.Context) {
	if err := h.service.repo.ResetWeeklyStats(c.Request.Context()); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to reset weekly stats")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Weekly stats reset successfully",
	})
}

// ResetMonthlyStats resets monthly stats (admin - scheduled job)
// POST /api/v1/admin/gamification/reset/monthly
func (h *Handler) ResetMonthlyStats(c *gin.Context) {
	if err := h.service.repo.ResetMonthlyStats(c.Request.Context()); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to reset monthly stats")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Monthly stats reset successfully",
	})
}

// ========================================
// HELPER FUNCTIONS
// ========================================

// getDriverID gets the driver ID from context
func (h *Handler) getDriverID(c *gin.Context) (uuid.UUID, error) {
	return middleware.GetUserID(c)
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers gamification routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	// Driver gamification routes
	driver := r.Group("/api/v1/driver/gamification")
	driver.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		driver.GET("/status", h.GetStatus)
		driver.GET("/quests", h.GetQuests)
		driver.POST("/quests/:id/join", h.JoinQuest)
		driver.POST("/quests/:id/claim", h.ClaimQuestReward)
		driver.GET("/achievements", h.GetAchievements)
		driver.GET("/leaderboard", h.GetLeaderboard)
		driver.GET("/tiers", h.GetTiers)
	}

	// Admin routes
	admin := r.Group("/api/v1/admin/gamification")
	admin.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	admin.Use(middleware.RequireRole(models.RoleAdmin))
	{
		admin.GET("/driver/:id", h.GetDriverStats)
		admin.POST("/reset/weekly", h.ResetWeeklyStats)
		admin.POST("/reset/monthly", h.ResetMonthlyStats)
	}
}

// RegisterRoutesOnGroup registers gamification routes on an existing router group
func (h *Handler) RegisterRoutesOnGroup(rg *gin.RouterGroup) {
	gamification := rg.Group("/driver/gamification")
	{
		gamification.GET("/status", h.GetStatus)
		gamification.GET("/quests", h.GetQuests)
		gamification.POST("/quests/:id/join", h.JoinQuest)
		gamification.POST("/quests/:id/claim", h.ClaimQuestReward)
		gamification.GET("/achievements", h.GetAchievements)
		gamification.GET("/leaderboard", h.GetLeaderboard)
		gamification.GET("/tiers", h.GetTiers)
	}
}
