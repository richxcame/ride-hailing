package support

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

// Handler handles HTTP requests for support
type Handler struct {
	service *Service
}

// NewHandler creates a new support handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// USER ENDPOINTS
// ========================================

// CreateTicket creates a new support ticket
// POST /api/v1/support/tickets
func (h *Handler) CreateTicket(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	ticket, err := h.service.CreateTicket(c.Request.Context(), userID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create ticket")
		return
	}

	common.CreatedResponse(c, ticket)
}

// GetMyTickets returns the current user's tickets
// GET /api/v1/support/tickets?status=open&page=1&page_size=20
func (h *Handler) GetMyTickets(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var status *TicketStatus
	if s := c.Query("status"); s != "" {
		st := TicketStatus(s)
		status = &st
	}

	tickets, total, err := h.service.GetMyTickets(c.Request.Context(), userID, status, page, pageSize)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get tickets")
		return
	}

	common.SuccessResponse(c, gin.H{
		"tickets":   tickets,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetTicket returns a specific ticket
// GET /api/v1/support/tickets/:id
func (h *Handler) GetTicket(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ticket id")
		return
	}

	ticket, err := h.service.GetTicket(c.Request.Context(), ticketID, userID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get ticket")
		return
	}

	common.SuccessResponse(c, ticket)
}

// GetTicketMessages returns messages for a ticket
// GET /api/v1/support/tickets/:id/messages
func (h *Handler) GetTicketMessages(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ticket id")
		return
	}

	messages, err := h.service.GetTicketMessages(c.Request.Context(), ticketID, userID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get messages")
		return
	}

	common.SuccessResponse(c, gin.H{"messages": messages})
}

// AddMessage adds a message to a ticket
// POST /api/v1/support/tickets/:id/messages
func (h *Handler) AddMessage(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ticket id")
		return
	}

	var req AddMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	msg, err := h.service.AddMessage(c.Request.Context(), ticketID, userID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to add message")
		return
	}

	common.CreatedResponse(c, msg)
}

// CloseTicket closes a ticket
// POST /api/v1/support/tickets/:id/close
func (h *Handler) CloseTicket(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ticket id")
		return
	}

	if err := h.service.CloseTicket(c.Request.Context(), ticketID, userID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to close ticket")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "ticket closed"})
}

// ========================================
// FAQ ENDPOINTS
// ========================================

// GetFAQArticles returns help center articles
// GET /api/v1/support/faq?category=payment
func (h *Handler) GetFAQArticles(c *gin.Context) {
	var category *string
	if cat := c.Query("category"); cat != "" {
		category = &cat
	}

	articles, err := h.service.GetFAQArticles(c.Request.Context(), category)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get articles")
		return
	}

	common.SuccessResponse(c, gin.H{"articles": articles})
}

// GetFAQArticle returns a single article
// GET /api/v1/support/faq/:id
func (h *Handler) GetFAQArticle(c *gin.Context) {
	articleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid article id")
		return
	}

	article, err := h.service.GetFAQArticle(c.Request.Context(), articleID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get article")
		return
	}

	common.SuccessResponse(c, article)
}

// ========================================
// ADMIN ENDPOINTS
// ========================================

// AdminGetTickets returns all tickets with filters
// GET /api/v1/admin/support/tickets?status=open&priority=urgent&category=safety&page=1
func (h *Handler) AdminGetTickets(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var status *TicketStatus
	if s := c.Query("status"); s != "" {
		st := TicketStatus(s)
		status = &st
	}
	var priority *TicketPriority
	if p := c.Query("priority"); p != "" {
		pr := TicketPriority(p)
		priority = &pr
	}
	var category *TicketCategory
	if cat := c.Query("category"); cat != "" {
		tc := TicketCategory(cat)
		category = &tc
	}

	tickets, total, err := h.service.AdminGetTickets(c.Request.Context(), status, priority, category, page, pageSize)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get tickets")
		return
	}

	common.SuccessResponse(c, gin.H{
		"tickets":   tickets,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// AdminGetTicket returns a specific ticket (admin view)
// GET /api/v1/admin/support/tickets/:id
func (h *Handler) AdminGetTicket(c *gin.Context) {
	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ticket id")
		return
	}

	ticket, err := h.service.AdminGetTicket(c.Request.Context(), ticketID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get ticket")
		return
	}

	common.SuccessResponse(c, ticket)
}

// AdminGetMessages returns all messages including internal notes
// GET /api/v1/admin/support/tickets/:id/messages
func (h *Handler) AdminGetMessages(c *gin.Context) {
	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ticket id")
		return
	}

	messages, err := h.service.AdminGetMessages(c.Request.Context(), ticketID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get messages")
		return
	}

	common.SuccessResponse(c, gin.H{"messages": messages})
}

// AdminReply adds an admin reply to a ticket
// POST /api/v1/admin/support/tickets/:id/reply
func (h *Handler) AdminReply(c *gin.Context) {
	agentID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ticket id")
		return
	}

	var req AdminReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	msg, err := h.service.AdminReply(c.Request.Context(), ticketID, agentID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to reply")
		return
	}

	common.CreatedResponse(c, msg)
}

// AdminUpdateTicket updates ticket metadata
// PUT /api/v1/admin/support/tickets/:id
func (h *Handler) AdminUpdateTicket(c *gin.Context) {
	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ticket id")
		return
	}

	var req UpdateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.AdminUpdateTicket(c.Request.Context(), ticketID, &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update ticket")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "ticket updated"})
}

// AdminGetStats returns ticket analytics
// GET /api/v1/admin/support/stats
func (h *Handler) AdminGetStats(c *gin.Context) {
	stats, err := h.service.AdminGetStats(c.Request.Context())
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get stats")
		return
	}

	common.SuccessResponse(c, stats)
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers support routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	// User support routes
	support := r.Group("/api/v1/support")
	support.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		support.POST("/tickets", h.CreateTicket)
		support.GET("/tickets", h.GetMyTickets)
		support.GET("/tickets/:id", h.GetTicket)
		support.GET("/tickets/:id/messages", h.GetTicketMessages)
		support.POST("/tickets/:id/messages", h.AddMessage)
		support.POST("/tickets/:id/close", h.CloseTicket)
	}

	// FAQ routes (public)
	faq := r.Group("/api/v1/support/faq")
	{
		faq.GET("", h.GetFAQArticles)
		faq.GET("/:id", h.GetFAQArticle)
	}

	// Admin support routes
	admin := r.Group("/api/v1/admin/support")
	admin.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	admin.Use(middleware.RequireRole(models.RoleAdmin))
	{
		admin.GET("/tickets", h.AdminGetTickets)
		admin.GET("/tickets/:id", h.AdminGetTicket)
		admin.GET("/tickets/:id/messages", h.AdminGetMessages)
		admin.POST("/tickets/:id/reply", h.AdminReply)
		admin.PUT("/tickets/:id", h.AdminUpdateTicket)
		admin.GET("/stats", h.AdminGetStats)
	}
}
