package verification

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// Handler handles HTTP requests for verification
type Handler struct {
	service *Service
}

// NewHandler creates a new verification handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// DRIVER ENDPOINTS
// ========================================

// GetVerificationStatus gets the driver's verification status
// GET /api/v1/driver/verification/status
func (h *Handler) GetVerificationStatus(c *gin.Context) {
	driverID, err := h.getDriverID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	status, err := h.service.GetDriverVerificationStatus(c.Request.Context(), driverID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get verification status")
		return
	}

	common.SuccessResponse(c, status)
}

// SubmitSelfie submits a selfie for verification
// POST /api/v1/driver/verification/selfie
func (h *Handler) SubmitSelfie(c *gin.Context) {
	driverID, err := h.getDriverID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SubmitSelfieRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	req.DriverID = driverID

	result, err := h.service.VerifySelfie(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "selfie verification failed")
		return
	}

	common.SuccessResponse(c, result)
}

// GetSelfieStatus gets the status of a selfie verification
// GET /api/v1/driver/verification/selfie/:id
func (h *Handler) GetSelfieStatus(c *gin.Context) {
	verificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid verification ID")
		return
	}

	verification, err := h.service.GetSelfieVerificationStatus(c.Request.Context(), verificationID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "verification not found")
		return
	}

	common.SuccessResponse(c, verification)
}

// CheckSelfieRequired checks if driver needs to verify selfie
// GET /api/v1/driver/verification/selfie/required
func (h *Handler) CheckSelfieRequired(c *gin.Context) {
	driverID, err := h.getDriverID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	required, err := h.service.RequiresSelfieVerification(c.Request.Context(), driverID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to check verification requirement")
		return
	}

	common.SuccessResponse(c, gin.H{
		"selfie_required": required,
	})
}

// GetBackgroundStatus gets the driver's background check status
// GET /api/v1/driver/verification/background
func (h *Handler) GetBackgroundStatus(c *gin.Context) {
	driverID, err := h.getDriverID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	check, err := h.service.GetDriverBackgroundStatus(c.Request.Context(), driverID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "no background check found")
		return
	}

	// Sanitize response - don't expose full report URL to driver
	response := gin.H{
		"status":       check.Status,
		"check_type":   check.CheckType,
		"started_at":   check.StartedAt,
		"completed_at": check.CompletedAt,
		"expires_at":   check.ExpiresAt,
	}

	if check.Status == BGCheckStatusFailed && len(check.FailureReasons) > 0 {
		response["failure_reasons"] = check.FailureReasons
	}

	common.SuccessResponse(c, response)
}

// ========================================
// ADMIN ENDPOINTS
// ========================================

// InitiateBackgroundCheck starts a background check for a driver
// POST /api/v1/admin/verification/background
func (h *Handler) InitiateBackgroundCheck(c *gin.Context) {
	var req InitiateBackgroundCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.InitiateBackgroundCheck(c.Request.Context(), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to initiate background check")
		return
	}

	common.SuccessResponse(c, result)
}

// GetBackgroundCheck gets a specific background check (admin)
// GET /api/v1/admin/verification/background/:id
func (h *Handler) GetBackgroundCheck(c *gin.Context) {
	checkID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid check ID")
		return
	}

	check, err := h.service.GetBackgroundCheckStatus(c.Request.Context(), checkID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "background check not found")
		return
	}

	common.SuccessResponse(c, check)
}

// ReviewBackgroundCheck manually sets a background check status to passed or failed (admin)
// PATCH /api/v1/admin/verification/background/:id/review
func (h *Handler) ReviewBackgroundCheck(c *gin.Context) {
	checkID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid check ID")
		return
	}

	reviewerID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req ReviewBackgroundCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	check, err := h.service.ReviewBackgroundCheck(c.Request.Context(), checkID, reviewerID, req.Status, req.Notes)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to review background check")
		return
	}

	common.SuccessResponse(c, check)
}

// GetDriverBackgroundCheck gets background check for a specific driver (admin)
// GET /api/v1/admin/verification/driver/:id/background
func (h *Handler) GetDriverBackgroundCheck(c *gin.Context) {
	driverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid driver ID")
		return
	}

	check, err := h.service.GetDriverBackgroundStatus(c.Request.Context(), driverID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "no background check found for driver")
		return
	}

	common.SuccessResponse(c, check)
}

// GetDriverVerificationStatusAdmin gets verification status for a driver (admin)
// GET /api/v1/admin/verification/driver/:id/status
func (h *Handler) GetDriverVerificationStatusAdmin(c *gin.Context) {
	driverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid driver ID")
		return
	}

	status, err := h.service.GetDriverVerificationStatus(c.Request.Context(), driverID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get verification status")
		return
	}

	common.SuccessResponse(c, status)
}

// ========================================
// WEBHOOK ENDPOINTS
// ========================================

// HandleCheckrWebhook handles Checkr webhooks
// POST /webhooks/checkr
func (h *Handler) HandleCheckrWebhook(c *gin.Context) {
	var payload WebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid webhook payload")
		return
	}

	payload.Provider = ProviderCheckr

	if err := h.service.ProcessWebhook(c.Request.Context(), &payload); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "webhook processing failed")
		return
	}

	common.SuccessResponse(c, gin.H{"status": "processed"})
}

// HandleSterlingWebhook handles Sterling webhooks
// POST /webhooks/sterling
func (h *Handler) HandleSterlingWebhook(c *gin.Context) {
	var payload WebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid webhook payload")
		return
	}

	payload.Provider = ProviderSterling

	if err := h.service.ProcessWebhook(c.Request.Context(), &payload); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "webhook processing failed")
		return
	}

	common.SuccessResponse(c, gin.H{"status": "processed"})
}

// HandleOnfidoWebhook handles Onfido webhooks
// POST /webhooks/onfido
func (h *Handler) HandleOnfidoWebhook(c *gin.Context) {
	var payload WebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid webhook payload")
		return
	}

	payload.Provider = ProviderOnfido

	if err := h.service.ProcessWebhook(c.Request.Context(), &payload); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "webhook processing failed")
		return
	}

	common.SuccessResponse(c, gin.H{"status": "processed"})
}

// ========================================
// HELPER FUNCTIONS
// ========================================

// getDriverID gets the driver ID from context
func (h *Handler) getDriverID(c *gin.Context) (uuid.UUID, error) {
	// For now, use user ID as driver ID
	// In production, you'd look up the driver record
	return middleware.GetUserID(c)
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers verification routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	// Driver verification routes
	driver := r.Group("/api/v1/driver/verification")
	driver.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		driver.GET("/status", h.GetVerificationStatus)
		driver.POST("/selfie", h.SubmitSelfie)
		driver.GET("/selfie/:id", h.GetSelfieStatus)
		driver.GET("/selfie/required", h.CheckSelfieRequired)
		driver.GET("/background", h.GetBackgroundStatus)
	}

	// Admin verification routes
	admin := r.Group("/api/v1/admin/verification")
	admin.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	admin.Use(middleware.RequireRole(models.RoleAdmin))
	{
		admin.POST("/background", h.InitiateBackgroundCheck)
		admin.GET("/background/:id", h.GetBackgroundCheck)
		admin.PATCH("/background/:id/review", h.ReviewBackgroundCheck)
		admin.GET("/driver/:id/background", h.GetDriverBackgroundCheck)
		admin.GET("/driver/:id/status", h.GetDriverVerificationStatusAdmin)
	}

	// Webhook routes (no auth - verified by provider signatures)
	webhooks := r.Group("/webhooks")
	{
		webhooks.POST("/checkr", h.HandleCheckrWebhook)
		webhooks.POST("/sterling", h.HandleSterlingWebhook)
		webhooks.POST("/onfido", h.HandleOnfidoWebhook)
	}
}

// RegisterRoutesOnGroup registers verification routes on an existing router group
func (h *Handler) RegisterRoutesOnGroup(rg *gin.RouterGroup) {
	verification := rg.Group("/driver/verification")
	{
		verification.GET("/status", h.GetVerificationStatus)
		verification.POST("/selfie", h.SubmitSelfie)
		verification.GET("/selfie/:id", h.GetSelfieStatus)
		verification.GET("/selfie/required", h.CheckSelfieRequired)
		verification.GET("/background", h.GetBackgroundStatus)
	}
}

// RegisterAdminRoutes registers admin verification routes on an already-authenticated admin group.
// The group is expected to already have auth + admin-role middleware applied.
func (h *Handler) RegisterAdminRoutes(api *gin.RouterGroup) {
	bg := api.Group("/verification")
	{
		bg.POST("/background", h.InitiateBackgroundCheck)
		bg.GET("/background/:id", h.GetBackgroundCheck)
		bg.PATCH("/background/:id/review", h.ReviewBackgroundCheck)
		bg.GET("/driver/:id/background", h.GetDriverBackgroundCheck)
		bg.GET("/driver/:id/status", h.GetDriverVerificationStatusAdmin)
	}
}
