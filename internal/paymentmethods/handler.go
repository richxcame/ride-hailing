package paymentmethods

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
)

// Handler handles HTTP requests for payment methods
type Handler struct {
	service *Service
}

// NewHandler creates a new payment methods handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// PAYMENT METHODS
// ========================================

// GetPaymentMethods returns all payment methods
// GET /api/v1/payment-methods
func (h *Handler) GetPaymentMethods(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	resp, err := h.service.GetPaymentMethods(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get payment methods")
		return
	}

	common.SuccessResponse(c, resp)
}

// AddCard adds a card payment method
// POST /api/v1/payment-methods/cards
func (h *Handler) AddCard(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req AddCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	pm, err := h.service.AddCard(c.Request.Context(), userID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to add card")
		return
	}

	common.CreatedResponse(c, pm)
}

// AddDigitalWallet adds Apple Pay or Google Pay
// POST /api/v1/payment-methods/digital-wallet
func (h *Handler) AddDigitalWallet(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req AddWalletPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	pm, err := h.service.AddDigitalWallet(c.Request.Context(), userID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to add digital wallet")
		return
	}

	common.CreatedResponse(c, pm)
}

// EnableCash enables cash as a payment method
// POST /api/v1/payment-methods/cash
func (h *Handler) EnableCash(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	pm, err := h.service.EnableCash(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to enable cash")
		return
	}

	common.CreatedResponse(c, pm)
}

// SetDefault sets the default payment method
// POST /api/v1/payment-methods/default
func (h *Handler) SetDefault(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SetDefaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.SetDefault(c.Request.Context(), userID, req.PaymentMethodID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to set default")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "default payment method updated"})
}

// RemovePaymentMethod removes a payment method
// DELETE /api/v1/payment-methods/:id
func (h *Handler) RemovePaymentMethod(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	methodID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid payment method id")
		return
	}

	if err := h.service.RemovePaymentMethod(c.Request.Context(), userID, methodID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to remove payment method")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "payment method removed"})
}

// ========================================
// WALLET
// ========================================

// TopUpWallet adds funds to the wallet
// POST /api/v1/payment-methods/wallet/topup
func (h *Handler) TopUpWallet(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req TopUpWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	summary, err := h.service.TopUpWallet(c.Request.Context(), userID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to top up wallet")
		return
	}

	common.SuccessResponse(c, summary)
}

// GetWalletSummary returns wallet info
// GET /api/v1/payment-methods/wallet
func (h *Handler) GetWalletSummary(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	summary, err := h.service.GetWalletSummary(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get wallet")
		return
	}

	common.SuccessResponse(c, summary)
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers payment method routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	pm := r.Group("/api/v1/payment-methods")
	pm.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		pm.GET("", h.GetPaymentMethods)
		pm.POST("/cards", h.AddCard)
		pm.POST("/digital-wallet", h.AddDigitalWallet)
		pm.POST("/cash", h.EnableCash)
		pm.POST("/default", h.SetDefault)
		pm.DELETE("/:id", h.RemovePaymentMethod)

		// Wallet
		pm.GET("/wallet", h.GetWalletSummary)
		pm.POST("/wallet/topup", h.TopUpWallet)
	}
}
