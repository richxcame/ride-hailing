package notifications

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers notification routes
func (h *Handler) RegisterRoutes(router *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	api := router.Group("/api/v1")

	// Protected routes
	protected := api.Group("")
	protected.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	{
		// Notification management
		protected.GET("/notifications", h.GetNotifications)
		protected.GET("/notifications/unread/count", h.GetUnreadCount)
		protected.POST("/notifications/:id/read", h.MarkAsRead)
		protected.POST("/notifications/send", h.SendNotification)
		protected.POST("/notifications/schedule", h.ScheduleNotification)

		// Ride-specific notifications (called by rides service)
		protected.POST("/notifications/ride/requested", h.NotifyRideRequested)
		protected.POST("/notifications/ride/accepted", h.NotifyRideAccepted)
		protected.POST("/notifications/ride/started", h.NotifyRideStarted)
		protected.POST("/notifications/ride/completed", h.NotifyRideCompleted)
		protected.POST("/notifications/ride/cancelled", h.NotifyRideCancelled)
	}

	// Admin routes
	admin := api.Group("/admin")
	admin.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	admin.Use(middleware.RequireRole("admin"))
	{
		admin.POST("/notifications/bulk", h.SendBulkNotification)
	}
}

// SendNotificationRequest represents a notification request
type SendNotificationRequest struct {
	UserID  string                 `json:"user_id" binding:"required"`
	Type    string                 `json:"type" binding:"required"`
	Channel string                 `json:"channel" binding:"required,oneof=push sms email"`
	Title   string                 `json:"title" binding:"required"`
	Body    string                 `json:"body" binding:"required"`
	Data    map[string]interface{} `json:"data"`
}

// ScheduleNotificationRequest represents a scheduled notification request
type ScheduleNotificationRequest struct {
	UserID      string                 `json:"user_id" binding:"required"`
	Type        string                 `json:"type" binding:"required"`
	Channel     string                 `json:"channel" binding:"required,oneof=push sms email"`
	Title       string                 `json:"title" binding:"required"`
	Body        string                 `json:"body" binding:"required"`
	Data        map[string]interface{} `json:"data"`
	ScheduledAt string                 `json:"scheduled_at" binding:"required"`
}

// RideNotificationRequest represents a ride-related notification
type RideNotificationRequest struct {
	UserID string                 `json:"user_id" binding:"required"`
	RideID string                 `json:"ride_id"`
	Data   map[string]interface{} `json:"data"`
}

// SendNotification sends a notification
func (h *Handler) SendNotification(c *gin.Context) {
	var req SendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	notification, err := h.service.SendNotification(
		c.Request.Context(),
		userID,
		req.Type,
		req.Channel,
		req.Title,
		req.Body,
		req.Data,
	)

	if err != nil {
		appErr, ok := err.(*common.AppError)
		if ok {
			common.ErrorResponse(c, appErr.Code, appErr.Message)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send notification")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, notification, "Notification sent successfully")
}

// ScheduleNotification schedules a notification
func (h *Handler) ScheduleNotification(c *gin.Context) {
	var req ScheduleNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid scheduled_at format, use RFC3339")
		return
	}

	notification, err := h.service.ScheduleNotification(
		c.Request.Context(),
		userID,
		req.Type,
		req.Channel,
		req.Title,
		req.Body,
		req.Data,
		scheduledAt,
	)

	if err != nil {
		appErr, ok := err.(*common.AppError)
		if ok {
			common.ErrorResponse(c, appErr.Code, appErr.Message)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to schedule notification")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, notification, "Notification scheduled successfully")
}

// GetNotifications retrieves user's notifications
func (h *Handler) GetNotifications(c *gin.Context) {
	userUUID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	notifications, err := h.service.GetUserNotifications(
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
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get notifications")
		return
	}

	common.SuccessResponseWithMeta(c, notifications, &common.Meta{
		Limit:  limit,
		Offset: offset,
		Total:  int64(len(notifications)),
	})
}

// GetUnreadCount gets count of unread notifications
func (h *Handler) GetUnreadCount(c *gin.Context) {
	userUUID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	count, err := h.service.GetUnreadCount(c.Request.Context(), userUUID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get unread count")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, gin.H{"count": count}, "Unread count retrieved")
}

// MarkAsRead marks notification as read
func (h *Handler) MarkAsRead(c *gin.Context) {
	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid notification ID")
		return
	}

	err = h.service.MarkAsRead(c.Request.Context(), notificationID)
	if err != nil {
		appErr, ok := err.(*common.AppError)
		if ok {
			common.ErrorResponse(c, appErr.Code, appErr.Message)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to mark as read")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "Notification marked as read")
}

// NotifyRideRequested handles ride requested notification
func (h *Handler) NotifyRideRequested(c *gin.Context) {
	var req RideNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	driverID, err := uuid.Parse(req.UserID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid driver ID")
		return
	}

	rideID, err := uuid.Parse(req.RideID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid ride ID")
		return
	}

	pickupLocation := ""
	if loc, ok := req.Data["pickup_location"].(string); ok {
		pickupLocation = loc
	}

	err = h.service.NotifyRideRequested(c.Request.Context(), driverID, rideID, pickupLocation)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send notification")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "Ride requested notification sent")
}

// NotifyRideAccepted handles ride accepted notification
func (h *Handler) NotifyRideAccepted(c *gin.Context) {
	var req RideNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	riderID, err := uuid.Parse(req.UserID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid rider ID")
		return
	}

	driverName := ""
	if name, ok := req.Data["driver_name"].(string); ok {
		driverName = name
	}

	eta := 5
	if e, ok := req.Data["eta"].(float64); ok {
		eta = int(e)
	}

	err = h.service.NotifyRideAccepted(c.Request.Context(), riderID, driverName, eta)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send notification")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "Ride accepted notification sent")
}

// NotifyRideStarted handles ride started notification
func (h *Handler) NotifyRideStarted(c *gin.Context) {
	var req RideNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	riderID, err := uuid.Parse(req.UserID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid rider ID")
		return
	}

	err = h.service.NotifyRideStarted(c.Request.Context(), riderID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send notification")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "Ride started notification sent")
}

// NotifyRideCompleted handles ride completed notification
func (h *Handler) NotifyRideCompleted(c *gin.Context) {
	var req struct {
		RiderID  string  `json:"rider_id" binding:"required"`
		DriverID string  `json:"driver_id" binding:"required"`
		Fare     float64 `json:"fare" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	riderID, err := uuid.Parse(req.RiderID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid rider ID")
		return
	}

	driverID, err := uuid.Parse(req.DriverID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid driver ID")
		return
	}

	err = h.service.NotifyRideCompleted(c.Request.Context(), riderID, driverID, req.Fare)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send notification")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "Ride completed notification sent")
}

// NotifyRideCancelled handles ride cancelled notification
func (h *Handler) NotifyRideCancelled(c *gin.Context) {
	var req struct {
		UserID      string `json:"user_id" binding:"required"`
		CancelledBy string `json:"cancelled_by" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	err = h.service.NotifyRideCancelled(c.Request.Context(), userID, req.CancelledBy)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send notification")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, nil, "Ride cancelled notification sent")
}

// SendBulkNotification sends notification to multiple users
func (h *Handler) SendBulkNotification(c *gin.Context) {
	var req struct {
		UserIDs []string               `json:"user_ids" binding:"required"`
		Type    string                 `json:"type" binding:"required"`
		Channel string                 `json:"channel" binding:"required,oneof=push sms email"`
		Title   string                 `json:"title" binding:"required"`
		Body    string                 `json:"body" binding:"required"`
		Data    map[string]interface{} `json:"data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	var userIDs []uuid.UUID
	for _, idStr := range req.UserIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			common.ErrorResponse(c, http.StatusBadRequest, "invalid user ID in list")
			return
		}
		userIDs = append(userIDs, id)
	}

	err := h.service.SendBulkNotification(
		c.Request.Context(),
		userIDs,
		req.Type,
		req.Channel,
		req.Title,
		req.Body,
		req.Data,
	)

	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send bulk notification")
		return
	}

	common.SuccessResponseWithStatus(c, http.StatusOK, gin.H{"sent": len(userIDs)}, "Bulk notifications sent")
}
