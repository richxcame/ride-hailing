package chat

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
)

// Handler handles HTTP requests for chat
type Handler struct {
	service *Service
}

// NewHandler creates a new chat handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// MESSAGE ENDPOINTS
// ========================================

// SendMessage sends a chat message
// POST /api/v1/chat/messages
func (h *Handler) SendMessage(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	role, err := middleware.GetUserRole(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	msg, err := h.service.SendMessage(c.Request.Context(), userID, string(role), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send message")
		return
	}

	common.CreatedResponse(c, msg)
}

// GetConversation retrieves chat history for a ride
// GET /api/v1/chat/rides/:id/messages
func (h *Handler) GetConversation(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	conversation, err := h.service.GetConversation(c.Request.Context(), userID, rideID, limit, offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get conversation")
		return
	}

	common.SuccessResponse(c, conversation)
}

// MarkAsRead marks messages as read
// POST /api/v1/chat/read
func (h *Handler) MarkAsRead(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req MarkReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.MarkAsRead(c.Request.Context(), userID, &req); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to mark as read")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Messages marked as read"})
}

// GetActiveConversations lists rides with active chat
// GET /api/v1/chat/conversations
func (h *Handler) GetActiveConversations(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	rideIDs, err := h.service.GetActiveConversations(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get conversations")
		return
	}

	if rideIDs == nil {
		rideIDs = []uuid.UUID{}
	}

	common.SuccessResponse(c, gin.H{
		"ride_ids": rideIDs,
		"count":    len(rideIDs),
	})
}

// ========================================
// QUICK REPLIES
// ========================================

// GetQuickReplies returns quick reply options
// GET /api/v1/chat/quick-replies
func (h *Handler) GetQuickReplies(c *gin.Context) {
	role, err := middleware.GetUserRole(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	replies, err := h.service.GetQuickReplies(c.Request.Context(), string(role))
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get quick replies")
		return
	}

	common.SuccessResponse(c, gin.H{
		"quick_replies": replies,
		"count":         len(replies),
	})
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers chat routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	chat := r.Group("/api/v1/chat")
	chat.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		chat.POST("/messages", h.SendMessage)
		chat.GET("/rides/:id/messages", h.GetConversation)
		chat.POST("/read", h.MarkAsRead)
		chat.GET("/conversations", h.GetActiveConversations)
		chat.GET("/quick-replies", h.GetQuickReplies)
	}
}
