package vehicle

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/richxcame/ride-hailing/pkg/pagination"
)

// Handler handles HTTP requests for vehicles
type Handler struct {
	service *Service
}

// NewHandler creates a new vehicle handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// DRIVER ENDPOINTS
// ========================================

// Register registers a new vehicle
// POST /api/v1/driver/vehicles
func (h *Handler) Register(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req RegisterVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	vehicle, err := h.service.RegisterVehicle(c.Request.Context(), driverID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to register vehicle")
		return
	}

	common.CreatedResponse(c, vehicle)
}

// GetMyVehicles returns all driver's vehicles
// GET /api/v1/driver/vehicles
func (h *Handler) GetMyVehicles(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	params := pagination.ParseParams(c)
	resp, total, err := h.service.GetMyVehicles(c.Request.Context(), driverID, params.Limit, params.Offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get vehicles")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, total)
	common.SuccessResponseWithMeta(c, resp, meta)
}

// GetVehicle returns a specific vehicle
// GET /api/v1/driver/vehicles/:id
func (h *Handler) GetVehicle(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	vehicleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid vehicle id")
		return
	}

	vehicle, err := h.service.GetVehicle(c.Request.Context(), vehicleID, driverID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get vehicle")
		return
	}

	common.SuccessResponse(c, vehicle)
}

// UpdateVehicle updates vehicle details
// PUT /api/v1/driver/vehicles/:id
func (h *Handler) UpdateVehicle(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	vehicleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid vehicle id")
		return
	}

	var req UpdateVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	vehicle, err := h.service.UpdateVehicle(c.Request.Context(), vehicleID, driverID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update vehicle")
		return
	}

	common.SuccessResponse(c, vehicle)
}

// UploadPhotos uploads vehicle photos
// POST /api/v1/driver/vehicles/:id/photos
func (h *Handler) UploadPhotos(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	vehicleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid vehicle id")
		return
	}

	var req UploadVehiclePhotosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.UploadPhotos(c.Request.Context(), vehicleID, driverID, &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to upload photos")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "photos uploaded"})
}

// SetPrimary sets a vehicle as primary
// POST /api/v1/driver/vehicles/:id/primary
func (h *Handler) SetPrimary(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	vehicleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid vehicle id")
		return
	}

	if err := h.service.SetPrimary(c.Request.Context(), vehicleID, driverID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to set primary vehicle")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "primary vehicle set"})
}

// Retire retires a vehicle
// POST /api/v1/driver/vehicles/:id/retire
func (h *Handler) Retire(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	vehicleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid vehicle id")
		return
	}

	if err := h.service.RetireVehicle(c.Request.Context(), vehicleID, driverID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to retire vehicle")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "vehicle retired"})
}

// GetMaintenanceReminders returns maintenance reminders
// GET /api/v1/driver/vehicles/:id/maintenance
func (h *Handler) GetMaintenanceReminders(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	vehicleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid vehicle id")
		return
	}

	reminders, err := h.service.GetMaintenanceReminders(c.Request.Context(), vehicleID, driverID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get reminders")
		return
	}

	common.SuccessResponse(c, gin.H{"reminders": reminders})
}

// CompleteMaintenance marks a reminder as done
// POST /api/v1/driver/vehicles/maintenance/:reminderId/complete
func (h *Handler) CompleteMaintenance(c *gin.Context) {
	reminderID, err := uuid.Parse(c.Param("reminderId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid reminder id")
		return
	}

	if err := h.service.CompleteMaintenance(c.Request.Context(), reminderID); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to complete maintenance")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "maintenance completed"})
}

// GetInspections returns inspection history
// GET /api/v1/driver/vehicles/:id/inspections
func (h *Handler) GetInspections(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	vehicleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid vehicle id")
		return
	}

	inspections, err := h.service.GetInspections(c.Request.Context(), vehicleID, driverID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get inspections")
		return
	}

	common.SuccessResponse(c, gin.H{"inspections": inspections})
}

// ========================================
// ADMIN ENDPOINTS
// ========================================

// AdminListAll returns all vehicles with filters and pagination
// GET /api/v1/admin/vehicles?status=pending&category=comfort&search=toyota&year_from=2018&sort_by=created_at&sort_dir=desc
func (h *Handler) AdminListAll(c *gin.Context) {
	var filter AdminVehicleFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid filter parameters")
		return
	}

	params := pagination.ParseParams(c)
	vehicles, total, err := h.service.GetAllVehicles(c.Request.Context(), &filter, params.Limit, params.Offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list vehicles")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, total)
	common.SuccessResponseWithMeta(c, gin.H{"vehicles": vehicles}, meta)
}

// AdminReview approves or rejects a vehicle
// POST /api/v1/admin/vehicles/:id/review
func (h *Handler) AdminReview(c *gin.Context) {
	vehicleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid vehicle id")
		return
	}

	var req AdminReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.ReviewVehicle(c.Request.Context(), vehicleID, &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to review vehicle")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "vehicle reviewed"})
}

// AdminGetPending returns vehicles pending review
// GET /api/v1/admin/vehicles/pending?limit=20&offset=0
func (h *Handler) AdminGetPending(c *gin.Context) {
	params := pagination.ParseParams(c)

	vehicles, total, err := h.service.GetPendingReviews(c.Request.Context(), params.Limit, params.Offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get pending vehicles")
		return
	}
	if vehicles == nil {
		vehicles = []Vehicle{}
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, int64(total))
	common.SuccessResponseWithMeta(c, gin.H{
		"vehicles": vehicles,
	}, meta)
}

// AdminGetStats returns vehicle statistics
// GET /api/v1/admin/vehicles/stats
func (h *Handler) AdminGetStats(c *gin.Context) {
	stats, err := h.service.GetVehicleStats(c.Request.Context())
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get vehicle stats")
		return
	}

	common.SuccessResponse(c, stats)
}

// AdminSuspend suspends a vehicle
// POST /api/v1/admin/vehicles/:id/suspend
func (h *Handler) AdminSuspend(c *gin.Context) {
	vehicleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid vehicle id")
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "reason is required")
		return
	}

	if err := h.service.SuspendVehicle(c.Request.Context(), vehicleID, req.Reason); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to suspend vehicle")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "vehicle suspended"})
}

// AdminGetExpiring returns vehicles with expiring documents
// GET /api/v1/admin/vehicles/expiring?days=30
func (h *Handler) AdminGetExpiring(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))

	vehicles, err := h.service.GetExpiringDocuments(c.Request.Context(), days)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get expiring vehicles")
		return
	}

	common.SuccessResponse(c, gin.H{
		"vehicles": vehicles,
		"count":    len(vehicles),
		"days":     days,
	})
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers driver-facing vehicle routes on the mobile server.
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	driver := r.Group("/api/v1/driver/vehicles")
	driver.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	driver.Use(middleware.RequireRole(models.RoleDriver))
	{
		driver.POST("", h.Register)
		driver.GET("", h.GetMyVehicles)
		driver.GET("/:id", h.GetVehicle)
		driver.PUT("/:id", h.UpdateVehicle)
		driver.POST("/:id/photos", h.UploadPhotos)
		driver.POST("/:id/primary", h.SetPrimary)
		driver.POST("/:id/retire", h.Retire)
		driver.GET("/:id/maintenance", h.GetMaintenanceReminders)
		driver.POST("/maintenance/:reminderId/complete", h.CompleteMaintenance)
		driver.GET("/:id/inspections", h.GetInspections)
	}
}

// RegisterAdminRoutes registers vehicle management routes on the admin server.
// api must already have admin auth middleware applied (AuthMiddlewareWithProvider + RequireAdmin).
func (h *Handler) RegisterAdminRoutes(api *gin.RouterGroup) {
	vehicles := api.Group("/vehicles")
	{
		vehicles.GET("", h.AdminListAll)
		vehicles.GET("/pending", h.AdminGetPending)
		vehicles.GET("/stats", h.AdminGetStats)
		vehicles.GET("/expiring", h.AdminGetExpiring)
		vehicles.POST("/:id/review", h.AdminReview)
		vehicles.POST("/:id/suspend", h.AdminSuspend)
	}
}
