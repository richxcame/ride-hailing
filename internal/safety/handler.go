package safety

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/pagination"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for safety features
type Handler struct {
	service *Service
}

// NewHandler creates a new safety handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all safety routes
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	safety := rg.Group("/safety")
	{
		// Emergency SOS
		safety.POST("/sos", h.TriggerSOS)
		safety.DELETE("/sos/:id", h.CancelSOS)
		safety.GET("/sos", h.GetUserEmergencies)
		safety.GET("/sos/:id", h.GetEmergency)

		// Emergency Contacts
		safety.POST("/contacts", h.CreateEmergencyContact)
		safety.GET("/contacts", h.GetEmergencyContacts)
		safety.PUT("/contacts/:id", h.UpdateEmergencyContact)
		safety.DELETE("/contacts/:id", h.DeleteEmergencyContact)
		safety.POST("/contacts/:id/verify", h.VerifyEmergencyContact)

		// Ride Sharing
		safety.POST("/share", h.CreateShareLink)
		safety.GET("/share/:id", h.GetShareLinks)
		safety.DELETE("/share/:id", h.DeactivateShareLink)

		// Safety Settings
		safety.GET("/settings", h.GetSafetySettings)
		safety.PUT("/settings", h.UpdateSafetySettings)

		// Safety Checks
		safety.POST("/checks/respond", h.RespondToSafetyCheck)
		safety.GET("/checks/pending", h.GetPendingSafetyChecks)

		// Trusted/Blocked Drivers
		safety.POST("/drivers/trust", h.AddTrustedDriver)
		safety.DELETE("/drivers/trust/:driver_id", h.RemoveTrustedDriver)
		safety.POST("/drivers/block", h.BlockDriver)
		safety.DELETE("/drivers/block/:driver_id", h.UnblockDriver)
		safety.GET("/drivers/blocked", h.GetBlockedDrivers)

		// Incident Reports
		safety.POST("/incidents", h.ReportIncident)
	}
}

// RegisterPublicRoutes registers routes that don't require authentication
func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	// Public share view (accessed via share link)
	rg.GET("/share/:token", h.ViewSharedRide)
}

// ========================================
// EMERGENCY SOS
// ========================================

// TriggerSOS triggers an emergency SOS alert
// @Summary Trigger emergency SOS
// @Description Activates an emergency alert and notifies contacts
// @Tags Safety
// @Accept json
// @Produce json
// @Param request body TriggerSOSRequest true "SOS request"
// @Success 200 {object} TriggerSOSResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/safety/sos [post]
func (h *Handler) TriggerSOS(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req TriggerSOSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	resp, err := h.service.TriggerSOS(c.Request.Context(), userID, &req)
	if err != nil {
		logger.Error("Failed to trigger SOS", zap.Error(err))
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to trigger emergency alert")
		return
	}

	common.SuccessResponse(c, resp)
}

// CancelSOS cancels an emergency alert
// @Summary Cancel SOS alert
// @Description Cancels an active emergency alert
// @Tags Safety
// @Param id path string true "Alert ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/safety/sos/{id} [delete]
func (h *Handler) CancelSOS(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	alertID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid alert ID")
		return
	}

	if err := h.service.CancelSOS(c.Request.Context(), userID, alertID); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	common.SuccessResponse(c, gin.H{"message": "emergency alert cancelled"})
}

// GetEmergency retrieves a specific emergency alert
func (h *Handler) GetEmergency(c *gin.Context) {
	alertID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid alert ID")
		return
	}

	alert, err := h.service.GetEmergencyAlert(c.Request.Context(), alertID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get alert")
		return
	}
	if alert == nil {
		common.ErrorResponse(c, http.StatusNotFound, "alert not found")
		return
	}

	common.SuccessResponse(c, alert)
}

// GetUserEmergencies retrieves emergency alerts for the current user
// @Summary Get user's emergency alerts
// @Description Retrieves emergency alert history for the authenticated user
// @Tags Safety
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} EmergencyAlert
// @Security BearerAuth
// @Router /api/v1/safety/sos [get]
func (h *Handler) GetUserEmergencies(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	params := pagination.ParseParams(c)

	alerts, err := h.service.GetUserEmergencies(c.Request.Context(), userID, params.Limit, params.Offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get emergencies")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, int64(len(alerts)))
	common.SuccessResponseWithMeta(c, alerts, meta)
}

// ========================================
// EMERGENCY CONTACTS
// ========================================

// CreateEmergencyContact creates a new emergency contact
// @Summary Create emergency contact
// @Description Adds a new emergency contact for the user
// @Tags Safety
// @Accept json
// @Produce json
// @Param request body CreateEmergencyContactRequest true "Contact details"
// @Success 201 {object} EmergencyContact
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/safety/contacts [post]
func (h *Handler) CreateEmergencyContact(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateEmergencyContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	contact, err := h.service.CreateEmergencyContact(c.Request.Context(), userID, &req)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	common.CreatedResponse(c, contact)
}

// GetEmergencyContacts retrieves all emergency contacts for a user
// @Summary Get emergency contacts
// @Description Retrieves all emergency contacts for the authenticated user
// @Tags Safety
// @Produce json
// @Success 200 {array} EmergencyContact
// @Security BearerAuth
// @Router /api/v1/safety/contacts [get]
func (h *Handler) GetEmergencyContacts(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	contacts, err := h.service.GetEmergencyContacts(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get contacts")
		return
	}

	common.SuccessResponse(c, contacts)
}

// UpdateEmergencyContact updates an emergency contact
// @Summary Update emergency contact
// @Description Updates an existing emergency contact
// @Tags Safety
// @Accept json
// @Produce json
// @Param id path string true "Contact ID"
// @Param request body CreateEmergencyContactRequest true "Updated contact details"
// @Success 200 {object} EmergencyContact
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/safety/contacts/{id} [put]
func (h *Handler) UpdateEmergencyContact(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	contactID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid contact ID")
		return
	}

	var req CreateEmergencyContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	contact, err := h.service.UpdateEmergencyContact(c.Request.Context(), userID, contactID, &req)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	common.SuccessResponse(c, contact)
}

// DeleteEmergencyContact deletes an emergency contact
// @Summary Delete emergency contact
// @Description Removes an emergency contact
// @Tags Safety
// @Param id path string true "Contact ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/safety/contacts/{id} [delete]
func (h *Handler) DeleteEmergencyContact(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	contactID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid contact ID")
		return
	}

	if err := h.service.DeleteEmergencyContact(c.Request.Context(), userID, contactID); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	common.SuccessResponse(c, gin.H{"message": "contact deleted"})
}

// VerifyEmergencyContact verifies an emergency contact's phone
// @Summary Verify emergency contact
// @Description Verifies the phone number of an emergency contact
// @Tags Safety
// @Accept json
// @Param id path string true "Contact ID"
// @Param code body object true "Verification code" example({"code": "123456"})
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/safety/contacts/{id}/verify [post]
func (h *Handler) VerifyEmergencyContact(c *gin.Context) {
	contactID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid contact ID")
		return
	}

	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if err := h.service.VerifyEmergencyContact(c.Request.Context(), contactID, req.Code); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	common.SuccessResponse(c, gin.H{"message": "contact verified"})
}

// ========================================
// RIDE SHARING
// ========================================

// CreateShareLink creates a shareable link for live ride tracking
// @Summary Create ride share link
// @Description Creates a shareable link for live ride tracking
// @Tags Safety
// @Accept json
// @Produce json
// @Param request body CreateShareLinkRequest true "Share link request"
// @Success 201 {object} ShareLinkResponse
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/safety/share [post]
func (h *Handler) CreateShareLink(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateShareLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	resp, err := h.service.CreateShareLink(c.Request.Context(), userID, &req)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	common.CreatedResponse(c, resp)
}

// GetShareLinks retrieves share links for a ride
func (h *Handler) GetShareLinks(c *gin.Context) {
	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	links, err := h.service.GetRideShareLinks(c.Request.Context(), rideID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get share links")
		return
	}

	common.SuccessResponse(c, links)
}

// DeactivateShareLink deactivates a share link
// @Summary Deactivate share link
// @Description Deactivates a ride sharing link
// @Tags Safety
// @Param id path string true "Link ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/safety/share/{id} [delete]
func (h *Handler) DeactivateShareLink(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	linkID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid link ID")
		return
	}

	if err := h.service.DeactivateShareLink(c.Request.Context(), userID, linkID); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	common.SuccessResponse(c, gin.H{"message": "share link deactivated"})
}

// ViewSharedRide views a shared ride (public, no auth required)
// @Summary View shared ride
// @Description View a shared ride via token (no authentication required)
// @Tags Safety
// @Produce json
// @Param token path string true "Share token"
// @Success 200 {object} SharedRideView
// @Failure 404 {object} map[string]string
// @Router /share/{token} [get]
func (h *Handler) ViewSharedRide(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "token required")
		return
	}

	view, err := h.service.GetSharedRide(c.Request.Context(), token)
	if err != nil {
		common.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	common.SuccessResponse(c, view)
}

// ========================================
// SAFETY SETTINGS
// ========================================

// GetSafetySettings retrieves safety settings
// @Summary Get safety settings
// @Description Retrieves the user's safety settings
// @Tags Safety
// @Produce json
// @Success 200 {object} SafetySettings
// @Security BearerAuth
// @Router /api/v1/safety/settings [get]
func (h *Handler) GetSafetySettings(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	settings, err := h.service.GetSafetySettings(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get settings")
		return
	}

	common.SuccessResponse(c, settings)
}

// UpdateSafetySettings updates safety settings
// @Summary Update safety settings
// @Description Updates the user's safety settings
// @Tags Safety
// @Accept json
// @Produce json
// @Param request body UpdateSafetySettingsRequest true "Settings to update"
// @Success 200 {object} SafetySettings
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/safety/settings [put]
func (h *Handler) UpdateSafetySettings(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req UpdateSafetySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	settings, err := h.service.UpdateSafetySettings(c.Request.Context(), userID, &req)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	common.SuccessResponse(c, settings)
}

// ========================================
// SAFETY CHECKS
// ========================================

// RespondToSafetyCheck responds to a safety check
// @Summary Respond to safety check
// @Description Responds to a periodic safety check-in
// @Tags Safety
// @Accept json
// @Param request body SafetyCheckResponse true "Response"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/safety/checks/respond [post]
func (h *Handler) RespondToSafetyCheck(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SafetyCheckResponse
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if err := h.service.RespondToSafetyCheck(c.Request.Context(), userID, &req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	common.SuccessResponse(c, gin.H{"message": "response recorded"})
}

// GetPendingSafetyChecks gets pending safety checks for the user
func (h *Handler) GetPendingSafetyChecks(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	checks, err := h.service.GetPendingSafetyChecks(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get checks")
		return
	}

	common.SuccessResponse(c, checks)
}

// ========================================
// TRUSTED/BLOCKED DRIVERS
// ========================================

// AddTrustedDriver adds a driver to the trusted list
// @Summary Add trusted driver
// @Description Adds a driver to the user's trusted list
// @Tags Safety
// @Accept json
// @Param request body object true "Trusted driver" example({"driver_id": "uuid", "note": "Great driver"})
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/safety/drivers/trust [post]
func (h *Handler) AddTrustedDriver(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		DriverID string `json:"driver_id" binding:"required"`
		Note     string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	driverID, err := uuid.Parse(req.DriverID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid driver ID")
		return
	}

	if err := h.service.AddTrustedDriver(c.Request.Context(), userID, driverID, req.Note); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	common.SuccessResponse(c, gin.H{"message": "driver added to trusted list"})
}

// RemoveTrustedDriver removes a driver from the trusted list
func (h *Handler) RemoveTrustedDriver(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	driverID, err := uuid.Parse(c.Param("driver_id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid driver ID")
		return
	}

	if err := h.service.RemoveTrustedDriver(c.Request.Context(), userID, driverID); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	common.SuccessResponse(c, gin.H{"message": "driver removed from trusted list"})
}

// BlockDriver blocks a driver
// @Summary Block driver
// @Description Blocks a driver from matching with the user
// @Tags Safety
// @Accept json
// @Param request body object true "Block driver" example({"driver_id": "uuid", "reason": "Unsafe driving"})
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/safety/drivers/block [post]
func (h *Handler) BlockDriver(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		DriverID string `json:"driver_id" binding:"required"`
		Reason   string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	driverID, err := uuid.Parse(req.DriverID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid driver ID")
		return
	}

	if err := h.service.BlockDriver(c.Request.Context(), userID, driverID, req.Reason); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	common.SuccessResponse(c, gin.H{"message": "driver blocked"})
}

// UnblockDriver unblocks a driver
func (h *Handler) UnblockDriver(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	driverID, err := uuid.Parse(c.Param("driver_id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid driver ID")
		return
	}

	if err := h.service.UnblockDriver(c.Request.Context(), userID, driverID); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	common.SuccessResponse(c, gin.H{"message": "driver unblocked"})
}

// GetBlockedDrivers gets the list of blocked drivers
func (h *Handler) GetBlockedDrivers(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	driverIDs, err := h.service.GetBlockedDrivers(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get blocked drivers")
		return
	}

	common.SuccessResponse(c, gin.H{"blocked_drivers": driverIDs})
}

// ========================================
// INCIDENT REPORTS
// ========================================

// ReportIncident reports a safety incident
// @Summary Report safety incident
// @Description Reports a safety incident for review
// @Tags Safety
// @Accept json
// @Produce json
// @Param request body ReportIncidentRequest true "Incident report"
// @Success 201 {object} SafetyIncidentReport
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/safety/incidents [post]
func (h *Handler) ReportIncident(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req ReportIncidentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	report, err := h.service.ReportIncident(c.Request.Context(), userID, &req)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	common.CreatedResponse(c, report)
}

// ========================================
// ADMIN ENDPOINTS
// ========================================

// RegisterAdminRoutes registers admin routes for safety management
func (h *Handler) RegisterAdminRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	admin := rg.Group("/admin/safety")
	admin.Use(authMiddleware)
	admin.Use(middleware.RequireRole("admin"))
	{
		// Emergency alerts
		admin.GET("/emergencies", h.GetActiveEmergencies)
		admin.POST("/emergencies/:id/respond", h.RespondToEmergency)
		admin.POST("/emergencies/:id/resolve", h.ResolveEmergency)

		// Incident reports
		admin.GET("/incidents", h.AdminGetIncidents)
		admin.PUT("/incidents/:id", h.AdminUpdateIncident)

		// Statistics
		admin.GET("/stats", h.GetSafetyStats)
	}
}

// GetActiveEmergencies retrieves all active emergencies (admin)
func (h *Handler) GetActiveEmergencies(c *gin.Context) {
	alerts, err := h.service.GetActiveEmergencies(c.Request.Context())
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get emergencies")
		return
	}

	common.SuccessResponse(c, alerts)
}

// RespondToEmergency marks an emergency as responded (admin)
func (h *Handler) RespondToEmergency(c *gin.Context) {
	adminID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	alertID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid alert ID")
		return
	}

	if err := h.service.MarkEmergencyResponded(c.Request.Context(), alertID, adminID); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update alert")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "emergency marked as responded"})
}

// ResolveEmergency resolves an emergency (admin)
func (h *Handler) ResolveEmergency(c *gin.Context) {
	adminID, err := getUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	alertID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid alert ID")
		return
	}

	var req struct {
		Resolution   string `json:"resolution" binding:"required"`
		IsFalseAlarm bool   `json:"is_false_alarm"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if err := h.service.ResolveEmergencyAlert(c.Request.Context(), adminID, alertID, req.Resolution, req.IsFalseAlarm); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to resolve alert")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "emergency resolved"})
}

// AdminGetIncidents retrieves incident reports (admin)
func (h *Handler) AdminGetIncidents(c *gin.Context) {
	// Would implement pagination and filtering
	common.SuccessResponse(c, gin.H{"incidents": []SafetyIncidentReport{}})
}

// AdminUpdateIncident updates an incident report (admin)
func (h *Handler) AdminUpdateIncident(c *gin.Context) {
	incidentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid incident ID")
		return
	}

	var req struct {
		Status      string `json:"status"`
		Resolution  string `json:"resolution"`
		ActionTaken string `json:"action_taken"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if err := h.service.UpdateIncidentStatus(c.Request.Context(), incidentID, req.Status, req.Resolution, req.ActionTaken); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update incident")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "incident updated"})
}

// GetSafetyStats retrieves safety statistics (admin)
func (h *Handler) GetSafetyStats(c *gin.Context) {
	// Get stats for last 30 days
	// stats, err := h.service.repo.GetEmergencyAlertStats(c.Request.Context(), time.Now().AddDate(0, 0, -30))
	common.SuccessResponse(c, gin.H{
		"message": "stats endpoint",
	})
}

// ========================================
// HELPERS
// ========================================

func getUserID(c *gin.Context) (uuid.UUID, error) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, fmt.Errorf("user not found in context")
	}

	switch v := userIDStr.(type) {
	case string:
		return uuid.Parse(v)
	case uuid.UUID:
		return v, nil
	default:
		return uuid.Nil, fmt.Errorf("invalid user ID type")
	}
}

