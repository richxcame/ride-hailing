package giftcards

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// Handler handles HTTP requests for gift cards
type Handler struct {
	service *Service
}

// NewHandler creates a new gift cards handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// USER ENDPOINTS
// ========================================

// Purchase creates and purchases a gift card
// POST /api/v1/gift-cards
func (h *Handler) Purchase(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req PurchaseGiftCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	card, err := h.service.PurchaseCard(c.Request.Context(), userID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to purchase gift card")
		return
	}

	common.CreatedResponse(c, card)
}

// Redeem applies a gift card to the user's account
// POST /api/v1/gift-cards/redeem
func (h *Handler) Redeem(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req RedeemGiftCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	card, err := h.service.RedeemCard(c.Request.Context(), userID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to redeem gift card")
		return
	}

	common.SuccessResponse(c, gin.H{
		"message": "Gift card redeemed successfully",
		"card":    card,
	})
}

// CheckBalance checks a gift card balance by code
// GET /api/v1/gift-cards/balance/:code
func (h *Handler) CheckBalance(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "gift card code required")
		return
	}

	balance, err := h.service.CheckBalance(c.Request.Context(), code)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "gift card not found")
		return
	}

	common.SuccessResponse(c, balance)
}

// GetMySummary returns the user's gift card summary
// GET /api/v1/gift-cards/me
func (h *Handler) GetMySummary(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	summary, err := h.service.GetMySummary(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get gift card summary")
		return
	}

	common.SuccessResponse(c, summary)
}

// GetPurchasedCards returns cards the user has bought
// GET /api/v1/gift-cards/purchased
func (h *Handler) GetPurchasedCards(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	cards, err := h.service.GetPurchasedCards(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get purchased cards")
		return
	}

	common.SuccessResponse(c, gin.H{
		"cards": cards,
		"count": len(cards),
	})
}

// ========================================
// ADMIN ENDPOINTS
// ========================================

// CreateBulk creates multiple gift cards
// POST /api/v1/admin/gift-cards/bulk
func (h *Handler) CreateBulk(c *gin.Context) {
	var req CreateBulkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.CreateBulk(c.Request.Context(), &req)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create gift cards")
		return
	}

	common.CreatedResponse(c, result)
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers gift card routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	// Public balance check (no auth needed)
	r.GET("/api/v1/gift-cards/balance/:code", h.CheckBalance)

	// Authenticated user routes
	cards := r.Group("/api/v1/gift-cards")
	cards.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		cards.POST("", h.Purchase)
		cards.POST("/redeem", h.Redeem)
		cards.GET("/me", h.GetMySummary)
		cards.GET("/purchased", h.GetPurchasedCards)
	}

	// Admin routes
	admin := r.Group("/api/v1/admin/gift-cards")
	admin.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	admin.Use(middleware.RequireRole(models.RoleAdmin))
	{
		admin.POST("/bulk", h.CreateBulk)
	}
}
