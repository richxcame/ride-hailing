package promos

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/pagination"
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
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Code       string  `json:"code" binding:"required"`
		RideAmount float64 `json:"ride_amount" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	validation, err := h.service.ValidatePromoCode(c.Request.Context(), req.Code, userID, req.RideAmount)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to validate promo code")
		return
	}

	common.SuccessResponse(c, validation)
}

// CreatePromoCode creates a new promo code (admin only)
func (h *Handler) CreatePromoCode(c *gin.Context) {
	var promo PromoCode
	if err := c.ShouldBindJSON(&promo); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Set created_by from authenticated user
	userID, _ := middleware.GetUserID(c)
	promo.CreatedBy = &userID

	err := h.service.CreatePromoCode(c.Request.Context(), &promo)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	common.CreatedResponse(c, promo)
}

// GetRideTypes returns all available ride types
func (h *Handler) GetRideTypes(c *gin.Context) {
	rideTypes, err := h.service.GetAllRideTypes(c.Request.Context())
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get ride types")
		return
	}

	if rideTypes == nil {
		rideTypes = []*RideType{}
	}

	common.SuccessResponse(c, rideTypes)
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
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	rideTypeID, err := uuid.Parse(req.RideTypeID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride_type_id")
		return
	}

	// Default surge multiplier to 1.0
	if req.SurgeMultiplier == 0 {
		req.SurgeMultiplier = 1.0
	}

	fare, err := h.service.CalculateFareForRideType(c.Request.Context(), rideTypeID, req.Distance, req.Duration, req.SurgeMultiplier)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to calculate fare")
		return
	}

	response := map[string]interface{}{
		"fare":             fare,
		"distance":         req.Distance,
		"duration":         req.Duration,
		"surge_multiplier": req.SurgeMultiplier,
	}

	common.SuccessResponse(c, response)
}

// GetMyReferralCode gets the authenticated user's referral code
func (h *Handler) GetMyReferralCode(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get user's first name for code generation (you might want to fetch this from users table)
	firstName := "USER" // Default, should be fetched from user profile

	referralCode, err := h.service.GenerateReferralCode(c.Request.Context(), userID, firstName)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to generate referral code")
		return
	}

	common.SuccessResponse(c, referralCode)
}

// ApplyReferralCode applies a referral code during signup
func (h *Handler) ApplyReferralCode(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		ReferralCode string `json:"referral_code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	err = h.service.ApplyReferralCode(c.Request.Context(), req.ReferralCode, userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	response := map[string]interface{}{
		"message": "Referral code applied successfully",
		"bonus":   ReferredBonusAmount,
	}

	common.SuccessResponse(c, response)
}

// HealthCheck returns service health
func (h *Handler) HealthCheck(c *gin.Context) {
	response := map[string]interface{}{
		"status":  "healthy",
		"service": "promos",
	}
	common.SuccessResponse(c, response)
}

// GetAllPromoCodes returns all promo codes with pagination (admin only)
func (h *Handler) GetAllPromoCodes(c *gin.Context) {
	params := pagination.ParseParams(c)

	promoCodes, total, err := h.service.GetAllPromoCodes(c.Request.Context(), params.Limit, params.Offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get promo codes")
		return
	}

	if promoCodes == nil {
		promoCodes = []*PromoCode{}
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, int64(total))
	common.SuccessResponseWithMeta(c, promoCodes, meta)
}

// GetAllReferralCodes returns all referral codes with pagination (admin only)
func (h *Handler) GetAllReferralCodes(c *gin.Context) {
	params := pagination.ParseParams(c)

	codes, total, err := h.service.GetAllReferralCodes(c.Request.Context(), params.Limit, params.Offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get referral codes")
		return
	}

	if codes == nil {
		codes = []*ReferralCode{}
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, int64(total))
	common.SuccessResponseWithMeta(c, codes, meta)
}

// GetPromoCode returns a specific promo code (admin only)
func (h *Handler) GetPromoCode(c *gin.Context) {
	promoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid promo code ID")
		return
	}

	promo, err := h.service.GetPromoCodeByID(c.Request.Context(), promoID)
	if err != nil {
		common.ErrorResponse(c, http.StatusNotFound, "promo code not found")
		return
	}

	common.SuccessResponse(c, promo)
}

// UpdatePromoCode updates an existing promo code (admin only)
func (h *Handler) UpdatePromoCode(c *gin.Context) {
	promoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid promo code ID")
		return
	}

	var promo PromoCode
	if err := c.ShouldBindJSON(&promo); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	promo.ID = promoID

	err = h.service.UpdatePromoCode(c.Request.Context(), &promo)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	common.SuccessResponse(c, promo)
}

// DeactivatePromoCode deactivates a promo code (admin only)
func (h *Handler) DeactivatePromoCode(c *gin.Context) {
	promoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid promo code ID")
		return
	}

	err = h.service.DeactivatePromoCode(c.Request.Context(), promoID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to deactivate promo code")
		return
	}

	response := map[string]interface{}{
		"message": "Promo code deactivated successfully",
	}

	common.SuccessResponse(c, response)
}

// GetPromoCodeUsageStats returns usage statistics for a promo code (admin only)
func (h *Handler) GetPromoCodeUsageStats(c *gin.Context) {
	promoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid promo code ID")
		return
	}

	stats, err := h.service.GetPromoCodeUsageStats(c.Request.Context(), promoID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get promo code stats")
		return
	}

	common.SuccessResponse(c, stats)
}

// GetReferralDetails returns detailed information about a referral (admin only)
func (h *Handler) GetReferralDetails(c *gin.Context) {
	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid referral ID")
		return
	}

	referral, err := h.service.GetReferralByID(c.Request.Context(), referralID)
	if err != nil {
		common.ErrorResponse(c, http.StatusNotFound, "referral not found")
		return
	}

	common.SuccessResponse(c, referral)
}

// GetMyReferralEarnings returns the authenticated user's referral earnings
func (h *Handler) GetMyReferralEarnings(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	earnings, err := h.service.GetReferralEarnings(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get referral earnings")
		return
	}

	common.SuccessResponse(c, earnings)
}
