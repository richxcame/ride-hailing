package family

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
)

// Handler handles HTTP requests for family accounts
type Handler struct {
	service *Service
}

// NewHandler creates a new family handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// FAMILY MANAGEMENT
// ========================================

// CreateFamily creates a new family account
// POST /api/v1/family
func (h *Handler) CreateFamily(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateFamilyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.CreateFamily(c.Request.Context(), userID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create family")
		return
	}

	common.CreatedResponse(c, resp)
}

// GetMyFamily returns the user's family account
// GET /api/v1/family/me
func (h *Handler) GetMyFamily(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	resp, err := h.service.GetMyFamily(c.Request.Context(), userID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get family")
		return
	}

	common.SuccessResponse(c, resp)
}

// GetFamily returns a specific family account
// GET /api/v1/family/:id
func (h *Handler) GetFamily(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	familyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid family id")
		return
	}

	resp, err := h.service.GetFamily(c.Request.Context(), familyID, userID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get family")
		return
	}

	common.SuccessResponse(c, resp)
}

// UpdateFamily updates family settings
// PUT /api/v1/family/:id
func (h *Handler) UpdateFamily(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	familyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid family id")
		return
	}

	var req UpdateFamilyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	family, err := h.service.UpdateFamily(c.Request.Context(), familyID, userID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update family")
		return
	}

	common.SuccessResponse(c, family)
}

// ========================================
// INVITATIONS
// ========================================

// InviteMember sends a family invite
// POST /api/v1/family/:id/invites
func (h *Handler) InviteMember(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	familyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid family id")
		return
	}

	var req InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	invite, err := h.service.InviteMember(c.Request.Context(), familyID, userID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send invite")
		return
	}

	common.CreatedResponse(c, invite)
}

// GetPendingInvites returns the user's pending family invites
// GET /api/v1/family/invites
func (h *Handler) GetPendingInvites(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	invites, err := h.service.GetPendingInvites(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get invites")
		return
	}

	common.SuccessResponse(c, gin.H{
		"invites": invites,
		"count":   len(invites),
	})
}

// RespondToInvite accepts or declines a family invite
// POST /api/v1/family/invites/:inviteId/respond
func (h *Handler) RespondToInvite(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	inviteID, err := uuid.Parse(c.Param("inviteId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid invite id")
		return
	}

	var req RespondToInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.RespondToInvite(c.Request.Context(), inviteID, userID, &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to respond to invite")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "invite response recorded"})
}

// ========================================
// MEMBER MANAGEMENT
// ========================================

// UpdateMember updates a family member's settings
// PUT /api/v1/family/:id/members/:memberId
func (h *Handler) UpdateMember(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	familyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid family id")
		return
	}

	memberID, err := uuid.Parse(c.Param("memberId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid member id")
		return
	}

	var req UpdateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	member, err := h.service.UpdateMember(c.Request.Context(), familyID, memberID, userID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update member")
		return
	}

	common.SuccessResponse(c, member)
}

// RemoveMember removes a member from the family
// DELETE /api/v1/family/:id/members/:memberId
func (h *Handler) RemoveMember(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	familyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid family id")
		return
	}

	memberID, err := uuid.Parse(c.Param("memberId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid member id")
		return
	}

	if err := h.service.RemoveMember(c.Request.Context(), familyID, memberID, userID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to remove member")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "member removed"})
}

// LeaveFamily allows the current user to leave their family
// POST /api/v1/family/leave
func (h *Handler) LeaveFamily(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.service.LeaveFamily(c.Request.Context(), userID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to leave family")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "you have left the family"})
}

// ========================================
// RIDE AUTHORIZATION
// ========================================

// AuthorizeRide checks if a family member can take a ride
// POST /api/v1/family/authorize-ride
func (h *Handler) AuthorizeRide(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		FareAmount float64 `json:"fare_amount" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	auth, err := h.service.AuthorizeRide(c.Request.Context(), userID, req.FareAmount)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to authorize ride")
		return
	}

	common.SuccessResponse(c, auth)
}

// ========================================
// SPENDING REPORTS
// ========================================

// GetSpendingReport returns the family spending report
// GET /api/v1/family/:id/spending
func (h *Handler) GetSpendingReport(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	familyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid family id")
		return
	}

	report, err := h.service.GetSpendingReport(c.Request.Context(), familyID, userID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get spending report")
		return
	}

	common.SuccessResponse(c, report)
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers family account routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	family := r.Group("/api/v1/family")
	family.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		// Family management
		family.POST("", h.CreateFamily)
		family.GET("/me", h.GetMyFamily)
		family.GET("/:id", h.GetFamily)
		family.PUT("/:id", h.UpdateFamily)

		// Invitations
		family.GET("/invites", h.GetPendingInvites)
		family.POST("/:id/invites", h.InviteMember)
		family.POST("/invites/:inviteId/respond", h.RespondToInvite)

		// Member management
		family.PUT("/:id/members/:memberId", h.UpdateMember)
		family.DELETE("/:id/members/:memberId", h.RemoveMember)
		family.POST("/leave", h.LeaveFamily)

		// Ride authorization
		family.POST("/authorize-ride", h.AuthorizeRide)

		// Spending reports
		family.GET("/:id/spending", h.GetSpendingReport)
	}
}
