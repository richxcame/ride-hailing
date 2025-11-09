package fraud

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
)

// Handler handles HTTP requests for fraud detection
type Handler struct {
	service *Service
}

// NewHandler creates a new fraud detection handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers fraud detection routes
func (h *Handler) RegisterRoutes(router *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	api := router.Group("/api/v1/fraud")
	api.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	api.Use(middleware.RequireAdmin())
	{
		// Fraud alerts
		api.GET("/alerts", h.GetPendingAlerts)
		api.GET("/alerts/:id", h.GetAlert)
		api.POST("/alerts", h.CreateAlert)
		api.PUT("/alerts/:id/investigate", h.InvestigateAlert)
		api.PUT("/alerts/:id/resolve", h.ResolveAlert)

		// User risk management
		api.GET("/users/:id/alerts", h.GetUserAlerts)
		api.GET("/users/:id/risk-profile", h.GetUserRiskProfile)
		api.POST("/users/:id/analyze", h.AnalyzeUser)
		api.POST("/users/:id/suspend", h.SuspendUser)
		api.POST("/users/:id/reinstate", h.ReinstateUser)

		// Fraud detection
		api.POST("/detect/payment/:user_id", h.DetectPaymentFraud)
		api.POST("/detect/ride/:user_id", h.DetectRideFraud)
	}
}

// GetPendingAlerts retrieves all pending fraud alerts
func (h *Handler) GetPendingAlerts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	alerts, err := h.service.GetPendingAlerts(c.Request.Context(), page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts":   alerts,
		"page":     page,
		"per_page": perPage,
	})
}

// GetAlert retrieves a specific fraud alert
func (h *Handler) GetAlert(c *gin.Context) {
	alertID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert ID"})
		return
	}

	alert, err := h.service.GetAlert(c.Request.Context(), alertID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, alert)
}

// CreateAlertRequest represents a request to create a fraud alert
type CreateAlertRequest struct {
	UserID      uuid.UUID              `json:"user_id" binding:"required"`
	AlertType   FraudAlertType         `json:"alert_type" binding:"required"`
	AlertLevel  FraudAlertLevel        `json:"alert_level" binding:"required"`
	Description string                 `json:"description" binding:"required"`
	Details     map[string]interface{} `json:"details"`
	RiskScore   float64                `json:"risk_score" binding:"required,min=0,max=100"`
}

// CreateAlert creates a new fraud alert
func (h *Handler) CreateAlert(c *gin.Context) {
	var req CreateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	alert := &FraudAlert{
		UserID:      req.UserID,
		AlertType:   req.AlertType,
		AlertLevel:  req.AlertLevel,
		Description: req.Description,
		Details:     req.Details,
		RiskScore:   req.RiskScore,
	}

	if err := h.service.CreateAlert(c.Request.Context(), alert); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, alert)
}

// InvestigateAlertRequest represents a request to investigate an alert
type InvestigateAlertRequest struct {
	Notes string `json:"notes"`
}

// InvestigateAlert marks an alert as under investigation
func (h *Handler) InvestigateAlert(c *gin.Context) {
	alertID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert ID"})
		return
	}

	var req InvestigateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get admin ID from context
	adminID := c.GetString("user_id")
	investigatorID, err := uuid.Parse(adminID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}

	if err := h.service.InvestigateAlert(c.Request.Context(), alertID, investigatorID, req.Notes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "alert marked as investigating"})
}

// ResolveAlertRequest represents a request to resolve an alert
type ResolveAlertRequest struct {
	Confirmed   bool   `json:"confirmed"`
	Notes       string `json:"notes"`
	ActionTaken string `json:"action_taken"`
}

// ResolveAlert resolves a fraud alert
func (h *Handler) ResolveAlert(c *gin.Context) {
	alertID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert ID"})
		return
	}

	var req ResolveAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get admin ID from context
	adminID := c.GetString("user_id")
	investigatorID, err := uuid.Parse(adminID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}

	if err := h.service.ResolveAlert(c.Request.Context(), alertID, investigatorID, req.Confirmed, req.Notes, req.ActionTaken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "alert resolved successfully"})
}

// GetUserAlerts retrieves all fraud alerts for a user
func (h *Handler) GetUserAlerts(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	alerts, err := h.service.GetUserAlerts(c.Request.Context(), userID, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts":   alerts,
		"page":     page,
		"per_page": perPage,
	})
}

// GetUserRiskProfile retrieves a user's risk profile
func (h *Handler) GetUserRiskProfile(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	profile, err := h.service.GetUserRiskProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// AnalyzeUser performs comprehensive fraud analysis on a user
func (h *Handler) AnalyzeUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	profile, err := h.service.AnalyzeUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// SuspendUserRequest represents a request to suspend a user
type SuspendUserRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// SuspendUser suspends a user account due to fraud
func (h *Handler) SuspendUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req SuspendUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get admin ID from context
	adminID := c.GetString("user_id")
	adminUUID, err := uuid.Parse(adminID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin ID"})
		return
	}

	if err := h.service.SuspendUser(c.Request.Context(), userID, adminUUID, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user account suspended"})
}

// ReinstateUserRequest represents a request to reinstate a user
type ReinstateUserRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// ReinstateUser reinstates a suspended user account
func (h *Handler) ReinstateUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req ReinstateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get admin ID from context
	adminID := c.GetString("user_id")
	adminUUID, err := uuid.Parse(adminID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin ID"})
		return
	}

	if err := h.service.ReinstateUser(c.Request.Context(), userID, adminUUID, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user account reinstated"})
}

// DetectPaymentFraud analyzes payment patterns and creates alerts if needed
func (h *Handler) DetectPaymentFraud(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	if err := h.service.DetectPaymentFraud(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "payment fraud detection completed"})
}

// DetectRideFraud analyzes ride patterns and creates alerts if needed
func (h *Handler) DetectRideFraud(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	if err := h.service.DetectRideFraud(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ride fraud detection completed"})
}
