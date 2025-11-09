package payments

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

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers payment routes
func (h *Handler) RegisterRoutes(router *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	api := router.Group("/api/v1")

	// Protected routes
	protected := api.Group("")
	protected.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		// Wallet routes
		protected.GET("/wallet", h.GetWallet)
		protected.POST("/wallet/topup", h.TopUpWallet)
		protected.GET("/wallet/transactions", h.GetWalletTransactions)

		// Payment routes
		protected.POST("/payments/process", h.ProcessPayment)
		protected.POST("/payments/:id/refund", h.RefundPayment)
		protected.GET("/payments/:id", h.GetPayment)
	}

	// Webhook routes (no auth)
	api.POST("/webhooks/stripe", h.HandleStripeWebhook)
}

// ProcessPaymentRequest represents a payment request
type ProcessPaymentRequest struct {
	RideID        string  `json:"ride_id" binding:"required"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
	PaymentMethod string  `json:"payment_method" binding:"required,oneof=wallet stripe"`
}

// TopUpWalletRequest represents a wallet top-up request
type TopUpWalletRequest struct {
	Amount              float64 `json:"amount" binding:"required,gt=0"`
	StripePaymentMethod string  `json:"stripe_payment_method"`
}

// RefundRequest represents a refund request
type RefundRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// ProcessPayment processes a payment for a ride
func (h *Handler) ProcessPayment(c *gin.Context) {
	userUUID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req ProcessPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	rideID, err := uuid.Parse(req.RideID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	// In a real implementation, you would fetch the driver ID from the ride
	// For now, we'll use a placeholder
	driverID := uuid.New()

	payment, err := h.service.ProcessRidePayment(
		c.Request.Context(),
		rideID,
		userUUID,
		driverID,
		req.Amount,
		req.PaymentMethod,
	)

	if err != nil {
		appErr, ok := err.(*common.AppError)
		if ok {
			common.ErrorResponse(c, appErr.Code, appErr.Message)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to process payment")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, payment, "Payment processed successfully")
}

// TopUpWallet adds funds to user's wallet
func (h *Handler) TopUpWallet(c *gin.Context) {
	userUUID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req TopUpWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	transaction, err := h.service.TopUpWallet(
		c.Request.Context(),
		userUUID,
		req.Amount,
		req.StripePaymentMethod,
	)

	if err != nil {
		appErr, ok := err.(*common.AppError)
		if ok {
			common.ErrorResponse(c, appErr.Code, appErr.Message)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to top up wallet")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, transaction, "Wallet top-up initiated")
}

// GetWallet retrieves user's wallet information
func (h *Handler) GetWallet(c *gin.Context) {
	userUUID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	wallet, err := h.service.GetWallet(c.Request.Context(), userUUID)
	if err != nil {
		appErr, ok := err.(*common.AppError)
		if ok {
			common.ErrorResponse(c, appErr.Code, appErr.Message)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get wallet")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, wallet, "")
}

// GetWalletTransactions retrieves wallet transaction history
func (h *Handler) GetWalletTransactions(c *gin.Context) {
	userUUID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	transactions, err := h.service.GetWalletTransactions(
		c.Request.Context(),
		userUUID,
		limit,
		offset,
	)

	if err != nil {
		appErr, ok := err.(*common.AppError)
		if ok {
			common.ErrorResponse(c, appErr.Code, appErr.Message)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get transactions")
		return
	}

	common.SuccessResponseWithMeta(c, transactions, &common.Meta{
		Limit:  limit,
		Offset: offset,
		Total:  int64(len(transactions)),
	})
}

// GetPayment retrieves payment details
func (h *Handler) GetPayment(c *gin.Context) {
	userUUID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	paymentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid payment ID")
		return
	}

	payment, err := h.service.repo.GetPaymentByID(c.Request.Context(), paymentID)
	if err != nil {
		appErr, ok := err.(*common.AppError)
		if ok {
			common.ErrorResponse(c, appErr.Code, appErr.Message)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get payment")
		return
	}

	// Verify user owns this payment
	if payment.RiderID != userUUID && payment.DriverID != userUUID {
		common.ErrorResponse(c, http.StatusForbidden, "access denied")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, payment, "")
}

// RefundPayment processes a refund
func (h *Handler) RefundPayment(c *gin.Context) {
	userUUID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	role, err := middleware.GetUserRole(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Only admins and riders can request refunds
	if role != models.RoleAdmin && role != models.RoleRider {
		common.ErrorResponse(c, http.StatusForbidden, "access denied")
		return
	}

	paymentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid payment ID")
		return
	}

	var req RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Verify user owns this payment (if not admin)
	if role != models.RoleAdmin {
		payment, err := h.service.repo.GetPaymentByID(c.Request.Context(), paymentID)
		if err != nil {
			common.ErrorResponse(c, http.StatusNotFound, "payment not found")
			return
		}
		if payment.RiderID != userUUID {
			common.ErrorResponse(c, http.StatusForbidden, "access denied")
			return
		}
	}

	err = h.service.ProcessRefund(c.Request.Context(), paymentID, req.Reason)
	if err != nil {
		appErr, ok := err.(*common.AppError)
		if ok {
			common.ErrorResponse(c, appErr.Code, appErr.Message)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to process refund")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "Refund processed successfully")
}

// HandleStripeWebhook handles Stripe webhook events
func (h *Handler) HandleStripeWebhook(c *gin.Context) {
	// In a real implementation, you would:
	// 1. Verify the webhook signature
	// 2. Parse the event
	// 3. Handle the event based on type

	var event struct {
		Type string                 `json:"type"`
		Data map[string]interface{} `json:"data"`
	}

	if err := c.ShouldBindJSON(&event); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid webhook payload")
		return
	}

	// Extract payment intent ID
	paymentIntentID := ""
	if obj, ok := event.Data["object"].(map[string]interface{}); ok {
		if id, ok := obj["id"].(string); ok {
			paymentIntentID = id
		}
	}

	err := h.service.HandleStripeWebhook(c.Request.Context(), event.Type, paymentIntentID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to handle webhook")
		return
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}
