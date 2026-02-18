package admin

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/pagination"
)

// Handler handles HTTP requests for admin operations
type Handler struct {
	service *Service
}

// NewHandler creates a new admin handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetDashboard retrieves dashboard statistics
func (h *Handler) GetDashboard(c *gin.Context) {
	stats, err := h.service.GetDashboardStats(c.Request.Context())
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch dashboard stats")
		return
	}

	common.SuccessResponse(c, stats)
}

// GetAllUsers retrieves all users with pagination and filters
func (h *Handler) GetAllUsers(c *gin.Context) {
	params := pagination.ParseParams(c)

	// Parse filter query parameters
	filter := &UserFilter{}

	// Role filter: ?role=admin, ?role=driver, ?role=rider
	if role := c.Query("role"); role != "" {
		filter.Role = role
	}

	// Status filter: ?status=active or ?status=inactive
	if status := c.Query("status"); status != "" {
		switch status {
		case "active":
			isActive := true
			filter.IsActive = &isActive
		case "inactive":
			isActive := false
			filter.IsActive = &isActive
		}
	}

	// Search filter: ?search=john or ?search=john@example.com or ?search=+1234567890
	if search := c.Query("search"); search != "" {
		filter.Search = search
	}

	users, total, err := h.service.GetAllUsers(c.Request.Context(), params.Limit, params.Offset, filter)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch users")
		return
	}

	// Get user stats for the stats cards
	userStats, err := h.service.GetUserStats(c.Request.Context())
	if err != nil {
		// Log error but don't fail the request - stats are supplementary
		userStats = nil
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, int64(total))
	meta.Stats = userStats
	common.SuccessResponseWithMeta(c, users, meta)
}

// GetUser retrieves a specific user
func (h *Handler) GetUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	user, err := h.service.GetUser(c.Request.Context(), userID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "User not found")
		return
	}

	common.SuccessResponse(c, user)
}

// getAdminID extracts the authenticated admin user ID from the request context.
func getAdminID(c *gin.Context) uuid.UUID {
	if id, ok := c.Get("user_id"); ok {
		if uid, ok := id.(uuid.UUID); ok {
			return uid
		}
	}
	return uuid.Nil
}

// SuspendUserRequest holds the optional reason for a suspension.
type SuspendUserRequest struct {
	Reason string `json:"reason"`
}

// SuspendUser suspends a user account
func (h *Handler) SuspendUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req SuspendUserRequest
	_ = c.ShouldBindJSON(&req) // reason is optional

	if err := h.service.SuspendUser(c.Request.Context(), getAdminID(c), userID, req.Reason); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to suspend user")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "User suspended successfully")
}

// ActivateUser activates a user account
func (h *Handler) ActivateUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	if err := h.service.ActivateUser(c.Request.Context(), getAdminID(c), userID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to activate user")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "User activated successfully")
}

// GetAllDrivers retrieves all drivers with pagination, filters, and stats
func (h *Handler) GetAllDrivers(c *gin.Context) {
	params := pagination.ParseParams(c)

	// Parse filter query parameters
	filter := &DriverFilter{}

	// Status filter: ?status=online, ?status=offline, ?status=available, ?status=pending
	if status := c.Query("status"); status != "" {
		filter.Status = status
	}

	// Search filter: ?search=ABC123 (license, name, vehicle model, plate)
	if search := c.Query("search"); search != "" {
		filter.Search = search
	}

	drivers, total, err := h.service.GetAllDrivers(c.Request.Context(), params.Limit, params.Offset, filter)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch drivers")
		return
	}

	// Get driver stats for the stats cards
	driverStats, err := h.service.GetDriverStats(c.Request.Context())
	if err != nil {
		// Log error but don't fail the request - stats are supplementary
		driverStats = nil
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, total)
	meta.Stats = driverStats
	common.SuccessResponseWithMeta(c, drivers, meta)
}

// GetDriver retrieves a specific driver by ID
func (h *Handler) GetDriver(c *gin.Context) {
	driverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid driver ID")
		return
	}

	driver, err := h.service.GetDriver(c.Request.Context(), driverID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "Driver not found")
		return
	}

	common.SuccessResponse(c, driver)
}

// GetPendingDrivers retrieves drivers awaiting approval
func (h *Handler) GetPendingDrivers(c *gin.Context) {
	params := pagination.ParseParams(c)

	drivers, total, err := h.service.GetPendingDrivers(c.Request.Context(), params.Limit, params.Offset)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch pending drivers")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, total)
	common.SuccessResponseWithMeta(c, drivers, meta)
}

// ApproveDriver approves a driver application
func (h *Handler) ApproveDriver(c *gin.Context) {
	driverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid driver ID")
		return
	}

	if err := h.service.ApproveDriver(c.Request.Context(), getAdminID(c), driverID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to approve driver")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "Driver approved successfully")
}

// RejectDriver rejects a driver application
func (h *Handler) RejectDriver(c *gin.Context) {
	driverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid driver ID")
		return
	}

	if err := h.service.RejectDriver(c.Request.Context(), getAdminID(c), driverID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to reject driver")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "Driver rejected successfully")
}

// GetRideStats retrieves ride statistics
func (h *Handler) GetRideStats(c *gin.Context) {
	var startDate, endDate *time.Time

	if start := c.Query("start_date"); start != "" {
		t, err := time.Parse("2006-01-02", start)
		if err == nil {
			startDate = &t
		}
	}

	if end := c.Query("end_date"); end != "" {
		t, err := time.Parse("2006-01-02", end)
		if err == nil {
			endDate = &t
		}
	}

	stats, err := h.service.GetRideStats(c.Request.Context(), startDate, endDate)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch ride stats")
		return
	}

	common.SuccessResponse(c, stats)
}

// GetRide retrieves a specific ride by ID with rider and driver details
func (h *Handler) GetRide(c *gin.Context) {
	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid ride ID")
		return
	}

	ride, err := h.service.GetRide(c.Request.Context(), rideID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "Ride not found")
		return
	}

	common.SuccessResponse(c, ride)
}

// GetRecentRides retrieves recent rides for monitoring
func (h *Handler) GetRecentRides(c *gin.Context) {
	params := pagination.ParseParams(c)
	// For recent rides, default limit is higher (50 instead of 20)
	if c.Query("limit") == "" {
		params.Limit = 50
	}

	rides, total, err := h.service.GetRecentRides(c.Request.Context(), params.Limit, params.Offset)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch recent rides")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, total)
	common.SuccessResponseWithMeta(c, rides, meta)
}

// HealthCheck returns service health status
func (h *Handler) HealthCheck(c *gin.Context) {
	healthData := gin.H{
		"status":  "healthy",
		"service": "admin",
	}
	common.SuccessResponse(c, healthData)
}

// GetRealtimeMetrics retrieves real-time dashboard metrics
func (h *Handler) GetRealtimeMetrics(c *gin.Context) {
	metrics, err := h.service.GetRealtimeMetrics(c.Request.Context())
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch realtime metrics")
		return
	}

	common.SuccessResponse(c, metrics)
}

// GetDashboardSummary retrieves comprehensive dashboard summary
func (h *Handler) GetDashboardSummary(c *gin.Context) {
	period := c.DefaultQuery("period", "today")

	summary, err := h.service.GetDashboardSummary(c.Request.Context(), period)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch dashboard summary")
		return
	}

	common.SuccessResponse(c, summary)
}

// GetRevenueTrend retrieves revenue trend data for charts
func (h *Handler) GetRevenueTrend(c *gin.Context) {
	period := c.DefaultQuery("period", "7days")
	groupBy := c.DefaultQuery("group_by", "day")

	trend, err := h.service.GetRevenueTrend(c.Request.Context(), period, groupBy)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch revenue trend")
		return
	}

	common.SuccessResponse(c, trend)
}

// GetActionItems retrieves items requiring admin attention
func (h *Handler) GetActionItems(c *gin.Context) {
	items, err := h.service.GetActionItems(c.Request.Context())
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch action items")
		return
	}

	common.SuccessResponse(c, items)
}

// GetActivityFeed retrieves recent activity events across the system
func (h *Handler) GetActivityFeed(c *gin.Context) {
	params := pagination.ParseParams(c)
	if c.Query("limit") == "" {
		params.Limit = 20
	}

	items, total, err := h.service.GetActivityFeed(c.Request.Context(), params.Limit, params.Offset)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch activity feed")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, total)
	common.SuccessResponseWithMeta(c, items, meta)
}

// GetAuditLogs retrieves audit logs with pagination and filters
func (h *Handler) GetAuditLogs(c *gin.Context) {
	params := pagination.ParseParams(c)
	if c.Query("limit") == "" {
		params.Limit = 50
	}

	filter := &AuditLogFilter{}

	if adminIDStr := c.Query("admin_id"); adminIDStr != "" {
		if id, err := uuid.Parse(adminIDStr); err == nil {
			filter.AdminID = &id
		}
	}
	if action := c.Query("action"); action != "" {
		filter.Action = action
	}
	if targetType := c.Query("target_type"); targetType != "" {
		filter.TargetType = targetType
	}
	if targetIDStr := c.Query("target_id"); targetIDStr != "" {
		if id, err := uuid.Parse(targetIDStr); err == nil {
			filter.TargetID = &id
		}
	}

	logs, total, err := h.service.GetAuditLogs(c.Request.Context(), params.Limit, params.Offset, filter)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch audit logs")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, total)
	common.SuccessResponseWithMeta(c, logs, meta)
}
