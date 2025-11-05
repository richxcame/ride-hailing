package promos

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/middleware"
)

// Handler handles HTTP requests for promos service
type Handler struct {
	service *Service
}

// NewHandler creates a new promos handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ValidatePromoCode validates a promo code
func (h *Handler) ValidatePromoCode(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Code       string  `json:"code" binding:"required"`
		RideAmount float64 `json:"ride_amount" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validation, err := h.service.ValidatePromoCode(c.Request.Context(), req.Code, userID, req.RideAmount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate promo code"})
		return
	}

	c.JSON(http.StatusOK, validation)
}

// CreatePromoCode creates a new promo code (admin only)
func (h *Handler) CreatePromoCode(c *gin.Context) {
	var promo PromoCode
	if err := c.ShouldBindJSON(&promo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set created_by from authenticated user
	userID, _ := middleware.GetUserID(c)
	promo.CreatedBy = &userID

	err := h.service.CreatePromoCode(c.Request.Context(), &promo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, promo)
}

// GetRideTypes returns all available ride types
func (h *Handler) GetRideTypes(c *gin.Context) {
	rideTypes, err := h.service.GetAllRideTypes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get ride types"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ride_types": rideTypes,
	})
}

// CalculateFare calculates fare for a specific ride type
func (h *Handler) CalculateFare(c *gin.Context) {
	var req struct {
		RideTypeID      string  `json:"ride_type_id" binding:"required"`
		Distance        float64 `json:"distance" binding:"required,gt=0"`
		Duration        int     `json:"duration" binding:"required,gt=0"`
		SurgeMultiplier float64 `json:"surge_multiplier"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rideTypeID, err := uuid.Parse(req.RideTypeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ride_type_id"})
		return
	}

	// Default surge multiplier to 1.0
	if req.SurgeMultiplier == 0 {
		req.SurgeMultiplier = 1.0
	}

	fare, err := h.service.CalculateFareForRideType(c.Request.Context(), rideTypeID, req.Distance, req.Duration, req.SurgeMultiplier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate fare"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"fare":             fare,
		"distance":         req.Distance,
		"duration":         req.Duration,
		"surge_multiplier": req.SurgeMultiplier,
	})
}

// GetMyReferralCode gets the authenticated user's referral code
func (h *Handler) GetMyReferralCode(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Get user's first name for code generation (you might want to fetch this from users table)
	firstName := "USER" // Default, should be fetched from user profile

	referralCode, err := h.service.GenerateReferralCode(c.Request.Context(), userID, firstName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate referral code"})
		return
	}

	c.JSON(http.StatusOK, referralCode)
}

// ApplyReferralCode applies a referral code during signup
func (h *Handler) ApplyReferralCode(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		ReferralCode string `json:"referral_code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.service.ApplyReferralCode(c.Request.Context(), req.ReferralCode, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Referral code applied successfully",
		"bonus":   ReferredBonusAmount,
	})
}

// HealthCheck returns service health
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "promos",
	})
}
