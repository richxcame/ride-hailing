package fraud

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/pagination"
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
		api.POST("/detect/account/:user_id", h.DetectAccountFraud)

		// Statistics and patterns
		api.GET("/statistics", h.GetFraudStatistics)
		api.GET("/patterns", h.GetFraudPatterns)
	}
}

// RegisterAdminRoutes registers fraud detection routes on an existing router group.
func (h *Handler) RegisterAdminRoutes(rg *gin.RouterGroup) {
	fraud := rg.Group("/fraud")
	{
		// Fraud alerts
		fraud.GET("/alerts", h.GetPendingAlerts)
		fraud.GET("/alerts/:id", h.GetAlert)
		fraud.POST("/alerts", h.CreateAlert)
		fraud.PUT("/alerts/:id/investigate", h.InvestigateAlert)
		fraud.PUT("/alerts/:id/resolve", h.ResolveAlert)

		// User risk management
		fraud.GET("/users/:id/alerts", h.GetUserAlerts)
		fraud.GET("/users/:id/risk-profile", h.GetUserRiskProfile)
		fraud.POST("/users/:id/analyze", h.AnalyzeUser)
		fraud.POST("/users/:id/suspend", h.SuspendUser)
		fraud.POST("/users/:id/reinstate", h.ReinstateUser)

		// Fraud detection
		fraud.POST("/detect/payment/:user_id", h.DetectPaymentFraud)
		fraud.POST("/detect/ride/:user_id", h.DetectRideFraud)
		fraud.POST("/detect/account/:user_id", h.DetectAccountFraud)

		// Statistics and patterns
		fraud.GET("/statistics", h.GetFraudStatistics)
		fraud.GET("/patterns", h.GetFraudPatterns)
	}
}

// GetPendingAlerts retrieves all pending fraud alerts
func (h *Handler) GetPendingAlerts(c *gin.Context) {
	params := pagination.ParseParams(c)

	alerts, total, err := h.service.GetPendingAlerts(c.Request.Context(), params.Limit, params.Offset)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get pending alerts")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, total)
	common.SuccessResponseWithMeta(c, alerts, meta)
}

// GetAlert retrieves a specific fraud alert
func (h *Handler) GetAlert(c *gin.Context) {
	alertID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid alert ID")
		return
	}

	alert, err := h.service.GetAlert(c.Request.Context(), alertID)
	if err != nil {
		common.ErrorResponse(c, http.StatusNotFound, "Alert not found")
		return
	}

	common.SuccessResponse(c, alert)
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
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
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
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to create alert")
		return
	}

	common.CreatedResponse(c, alert)
}

// InvestigateAlertRequest represents a request to investigate an alert
type InvestigateAlertRequest struct {
	Notes string `json:"notes"`
}

// InvestigateAlert marks an alert as under investigation
func (h *Handler) InvestigateAlert(c *gin.Context) {
	alertID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid alert ID")
		return
	}

	var req InvestigateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Get admin ID from context
	adminID := c.GetString("user_id")
	investigatorID, err := uuid.Parse(adminID)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "invalid user ID")
		return
	}

	if err := h.service.InvestigateAlert(c.Request.Context(), alertID, investigatorID, req.Notes); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to investigate alert")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "alert marked as investigating"})
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
		common.ErrorResponse(c, http.StatusBadRequest, "invalid alert ID")
		return
	}

	var req ResolveAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Get admin ID from context
	adminID := c.GetString("user_id")
	investigatorID, err := uuid.Parse(adminID)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "invalid user ID")
		return
	}

	if err := h.service.ResolveAlert(c.Request.Context(), alertID, investigatorID, req.Confirmed, req.Notes, req.ActionTaken); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to resolve alert")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "alert resolved successfully"})
}

// GetUserAlerts retrieves all fraud alerts for a user
func (h *Handler) GetUserAlerts(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	params := pagination.ParseParams(c)

	alerts, total, err := h.service.GetUserAlerts(c.Request.Context(), userID, params.Limit, params.Offset)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get user alerts")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, total)
	common.SuccessResponseWithMeta(c, alerts, meta)
}

// GetUserRiskProfile retrieves a user's risk profile
func (h *Handler) GetUserRiskProfile(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	profile, err := h.service.GetUserRiskProfile(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to get user risk profile")
		return
	}

	common.SuccessResponse(c, profile)
}

// AnalyzeUser performs comprehensive fraud analysis on a user
func (h *Handler) AnalyzeUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	profile, err := h.service.AnalyzeUser(c.Request.Context(), userID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to analyze user")
		return
	}

	common.SuccessResponse(c, profile)
}

// SuspendUserRequest represents a request to suspend a user
type SuspendUserRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// SuspendUser suspends a user account due to fraud
func (h *Handler) SuspendUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	var req SuspendUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Get admin ID from context
	adminID := c.GetString("user_id")
	adminUUID, err := uuid.Parse(adminID)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "invalid admin ID")
		return
	}

	if err := h.service.SuspendUser(c.Request.Context(), userID, adminUUID, req.Reason); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to suspend user")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "user account suspended"})
}

// ReinstateUserRequest represents a request to reinstate a user
type ReinstateUserRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// ReinstateUser reinstates a suspended user account
func (h *Handler) ReinstateUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	var req ReinstateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Get admin ID from context
	adminID := c.GetString("user_id")
	adminUUID, err := uuid.Parse(adminID)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "invalid admin ID")
		return
	}

	if err := h.service.ReinstateUser(c.Request.Context(), userID, adminUUID, req.Reason); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to reinstate user")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "user account reinstated"})
}

// DetectPaymentFraud analyzes payment patterns and creates alerts if needed
func (h *Handler) DetectPaymentFraud(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	if err := h.service.DetectPaymentFraud(c.Request.Context(), userID); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to detect payment fraud")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "payment fraud detection completed"})
}

// DetectRideFraud analyzes ride patterns and creates alerts if needed
func (h *Handler) DetectRideFraud(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	if err := h.service.DetectRideFraud(c.Request.Context(), userID); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to detect ride fraud")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "ride fraud detection completed"})
}

// DetectAccountFraud analyzes account patterns and creates alerts if needed
func (h *Handler) DetectAccountFraud(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	if err := h.service.DetectAccountFraud(c.Request.Context(), userID); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to detect account fraud")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "account fraud detection completed"})
}

// GetFraudStatistics retrieves fraud statistics for a time period
func (h *Handler) GetFraudStatistics(c *gin.Context) {
	startDate := c.DefaultQuery("start_date", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	endDate := c.DefaultQuery("end_date", time.Now().Format("2006-01-02"))

	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid start_date format (use YYYY-MM-DD)")
		return
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid end_date format (use YYYY-MM-DD)")
		return
	}

	// Set end date to end of day
	end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	stats, err := h.service.GetFraudStatistics(c.Request.Context(), start, end)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to get fraud statistics")
		return
	}

	common.SuccessResponse(c, stats)
}

// GetFraudPatterns retrieves detected fraud patterns
func (h *Handler) GetFraudPatterns(c *gin.Context) {
	limit := 50
	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := c.GetQuery("limit"); err {
			if parsed, parseErr := parseInt(l); parseErr == nil && parsed > 0 {
				limit = parsed
			}
		}
	}

	patterns, err := h.service.GetFraudPatterns(c.Request.Context(), limit)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to get fraud patterns")
		return
	}

	if patterns == nil {
		patterns = []*FraudPattern{}
	}

	common.SuccessResponse(c, patterns)
}

// Helper function to parse int
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
