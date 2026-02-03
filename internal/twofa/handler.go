package twofa

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
)

// Handler handles HTTP requests for 2FA
type Handler struct {
	service *Service
}

// NewHandler creates a new 2FA handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// getRequestContext extracts common request context
func (h *Handler) getRequestContext(c *gin.Context) (userID uuid.UUID, ipAddress, userAgent *string, err error) {
	userID, err = middleware.GetUserID(c)
	if err != nil {
		return uuid.Nil, nil, nil, err
	}

	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")
	ipAddress = &ip
	userAgent = &ua

	return userID, ipAddress, userAgent, nil
}

// ========================================
// OTP ENDPOINTS
// ========================================

// SendOTP sends an OTP to the user's phone
// POST /api/v1/2fa/otp/send
func (h *Handler) SendOTP(c *gin.Context) {
	userID, ipAddress, userAgent, err := h.getRequestContext(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		OTPType OTPType `json:"otp_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Get user's phone number from context or request
	phoneNumber, _ := c.Get("phone_number")
	phone, ok := phoneNumber.(string)
	if !ok || phone == "" {
		// Fallback: Get from user profile (would need to inject user service)
		common.ErrorResponse(c, http.StatusBadRequest, "phone number not available")
		return
	}

	response, err := h.service.SendOTP(c.Request.Context(), userID, phone, req.OTPType, ipAddress, userAgent)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send OTP")
		return
	}

	common.SuccessResponse(c, response)
}

// VerifyOTP verifies an OTP code
// POST /api/v1/2fa/otp/verify
func (h *Handler) VerifyOTP(c *gin.Context) {
	userID, ipAddress, userAgent, err := h.getRequestContext(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.VerifyOTP(c.Request.Context(), userID, req.OTP, req.OTPType, ipAddress, userAgent); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusBadRequest, "OTP verification failed")
		return
	}

	common.SuccessResponse(c, gin.H{
		"success": true,
		"message": "OTP verified successfully",
	})
}

// ========================================
// 2FA SETUP ENDPOINTS
// ========================================

// Enable2FA enables 2FA for the user
// POST /api/v1/2fa/enable
func (h *Handler) Enable2FA(c *gin.Context) {
	userID, ipAddress, userAgent, err := h.getRequestContext(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req Enable2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Get phone number from context
	phoneNumber, _ := c.Get("phone_number")
	phone, _ := phoneNumber.(string)

	response, err := h.service.Enable2FA(c.Request.Context(), userID, req.Method, phone, ipAddress, userAgent)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to enable 2FA")
		return
	}

	common.SuccessResponse(c, response)
}

// Disable2FA disables 2FA for the user
// POST /api/v1/2fa/disable
func (h *Handler) Disable2FA(c *gin.Context) {
	userID, ipAddress, userAgent, err := h.getRequestContext(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req Disable2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.Disable2FA(c.Request.Context(), userID, req.OTP, req.BackupCode, ipAddress, userAgent); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to disable 2FA")
		return
	}

	common.SuccessResponse(c, gin.H{
		"success": true,
		"message": "2FA disabled successfully",
	})
}

// Get2FAStatus gets the 2FA status for the user
// GET /api/v1/2fa/status
func (h *Handler) Get2FAStatus(c *gin.Context) {
	userID, _, _, err := h.getRequestContext(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	status, err := h.service.Get2FAStatus(c.Request.Context(), userID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get 2FA status")
		return
	}

	common.SuccessResponse(c, status)
}

// ========================================
// BACKUP CODES ENDPOINTS
// ========================================

// RegenerateBackupCodes regenerates backup codes
// POST /api/v1/2fa/backup-codes/regenerate
func (h *Handler) RegenerateBackupCodes(c *gin.Context) {
	userID, ipAddress, userAgent, err := h.getRequestContext(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		OTP string `json:"otp" binding:"required,len=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	codes, err := h.service.RegenerateBackupCodes(c.Request.Context(), userID, req.OTP, ipAddress, userAgent)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to regenerate backup codes")
		return
	}

	common.SuccessResponse(c, gin.H{
		"backup_codes": codes,
		"message":      "Please save these backup codes securely. They will not be shown again.",
	})
}

// ========================================
// TRUSTED DEVICES ENDPOINTS
// ========================================

// GetTrustedDevices gets all trusted devices
// GET /api/v1/2fa/devices
func (h *Handler) GetTrustedDevices(c *gin.Context) {
	userID, _, _, err := h.getRequestContext(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get current device token from cookie if present
	currentToken, _ := c.Cookie("trusted_device")

	devices, err := h.service.GetTrustedDevices(c.Request.Context(), userID, currentToken)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get devices")
		return
	}

	common.SuccessResponse(c, gin.H{
		"devices": devices,
	})
}

// RevokeTrustedDevice revokes a trusted device
// DELETE /api/v1/2fa/devices/:id
func (h *Handler) RevokeTrustedDevice(c *gin.Context) {
	userID, ipAddress, userAgent, err := h.getRequestContext(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	deviceIDStr := c.Param("id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid device ID")
		return
	}

	if err := h.service.RevokeTrustedDevice(c.Request.Context(), userID, deviceID, ipAddress, userAgent); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to revoke device")
		return
	}

	common.SuccessResponse(c, gin.H{
		"success": true,
		"message": "Device revoked successfully",
	})
}

// ========================================
// PHONE VERIFICATION
// ========================================

// SendPhoneVerification sends a phone verification OTP
// POST /api/v1/2fa/phone/send
func (h *Handler) SendPhoneVerification(c *gin.Context) {
	userID, ipAddress, userAgent, err := h.getRequestContext(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get phone number from request or context
	var req struct {
		PhoneNumber string `json:"phone_number"`
	}
	_ = c.ShouldBindJSON(&req)

	phone := req.PhoneNumber
	if phone == "" {
		phoneNumber, _ := c.Get("phone_number")
		phone, _ = phoneNumber.(string)
	}

	if phone == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "phone number required")
		return
	}

	response, err := h.service.SendOTP(c.Request.Context(), userID, phone, OTPTypePhoneVerification, ipAddress, userAgent)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send verification")
		return
	}

	common.SuccessResponse(c, response)
}

// VerifyPhone verifies the user's phone number
// POST /api/v1/2fa/phone/verify
func (h *Handler) VerifyPhone(c *gin.Context) {
	userID, ipAddress, userAgent, err := h.getRequestContext(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		OTP string `json:"otp" binding:"required,len=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.VerifyPhone(c.Request.Context(), userID, req.OTP, ipAddress, userAgent); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusBadRequest, "phone verification failed")
		return
	}

	common.SuccessResponse(c, gin.H{
		"success": true,
		"message": "Phone number verified successfully",
	})
}

// ========================================
// TOTP VERIFICATION
// ========================================

// VerifyTOTP verifies a TOTP code
// POST /api/v1/2fa/totp/verify
func (h *Handler) VerifyTOTP(c *gin.Context) {
	userID, _, _, err := h.getRequestContext(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Code string `json:"code" binding:"required,len=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.VerifyTOTP(c.Request.Context(), userID, req.Code); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusBadRequest, "TOTP verification failed")
		return
	}

	// Mark TOTP as verified
	_ = h.service.repo.SetTOTPVerified(c.Request.Context(), userID)

	common.SuccessResponse(c, gin.H{
		"success": true,
		"message": "TOTP verified successfully",
	})
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers 2FA routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	twofa := r.Group("/api/v1/2fa")
	twofa.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		// Status
		twofa.GET("/status", h.Get2FAStatus)

		// Enable/Disable
		twofa.POST("/enable", h.Enable2FA)
		twofa.POST("/disable", h.Disable2FA)

		// OTP
		twofa.POST("/otp/send", h.SendOTP)
		twofa.POST("/otp/verify", h.VerifyOTP)

		// Phone verification
		twofa.POST("/phone/send", h.SendPhoneVerification)
		twofa.POST("/phone/verify", h.VerifyPhone)

		// TOTP
		twofa.POST("/totp/verify", h.VerifyTOTP)

		// Backup codes
		twofa.POST("/backup-codes/regenerate", h.RegenerateBackupCodes)

		// Trusted devices
		twofa.GET("/devices", h.GetTrustedDevices)
		twofa.DELETE("/devices/:id", h.RevokeTrustedDevice)
	}
}

// RegisterRoutesOnGroup registers 2FA routes on an existing router group
func (h *Handler) RegisterRoutesOnGroup(rg *gin.RouterGroup) {
	twofa := rg.Group("/2fa")
	{
		// Status
		twofa.GET("/status", h.Get2FAStatus)

		// Enable/Disable
		twofa.POST("/enable", h.Enable2FA)
		twofa.POST("/disable", h.Disable2FA)

		// OTP
		twofa.POST("/otp/send", h.SendOTP)
		twofa.POST("/otp/verify", h.VerifyOTP)

		// Phone verification
		twofa.POST("/phone/send", h.SendPhoneVerification)
		twofa.POST("/phone/verify", h.VerifyPhone)

		// TOTP
		twofa.POST("/totp/verify", h.VerifyTOTP)

		// Backup codes
		twofa.POST("/backup-codes/regenerate", h.RegenerateBackupCodes)

		// Trusted devices
		twofa.GET("/devices", h.GetTrustedDevices)
		twofa.DELETE("/devices/:id", h.RevokeTrustedDevice)
	}
}
