package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch dashboard stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetAllUsers retrieves all users with pagination
func (h *Handler) GetAllUsers(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	users, total, err := h.service.GetAllUsers(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users":  users,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetUser retrieves a specific user
func (h *Handler) GetUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.service.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// SuspendUser suspends a user account
func (h *Handler) SuspendUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := h.service.SuspendUser(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to suspend user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User suspended successfully"})
}

// ActivateUser activates a user account
func (h *Handler) ActivateUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := h.service.ActivateUser(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to activate user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User activated successfully"})
}

// GetPendingDrivers retrieves drivers awaiting approval
func (h *Handler) GetPendingDrivers(c *gin.Context) {
	drivers, err := h.service.GetPendingDrivers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pending drivers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"drivers": drivers,
		"count":   len(drivers),
	})
}

// ApproveDriver approves a driver application
func (h *Handler) ApproveDriver(c *gin.Context) {
	driverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid driver ID"})
		return
	}

	if err := h.service.ApproveDriver(c.Request.Context(), driverID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve driver"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Driver approved successfully"})
}

// RejectDriver rejects a driver application
func (h *Handler) RejectDriver(c *gin.Context) {
	driverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid driver ID"})
		return
	}

	if err := h.service.RejectDriver(c.Request.Context(), driverID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject driver"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Driver rejected successfully"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch ride stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetRecentRides retrieves recent rides for monitoring
func (h *Handler) GetRecentRides(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	rides, err := h.service.GetRecentRides(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch recent rides"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rides": rides,
		"count": len(rides),
	})
}

// HealthCheck returns service health status
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "admin",
	})
}
