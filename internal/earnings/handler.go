package earnings

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/jwtkeys"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
)

// Handler handles HTTP requests for driver earnings
type Handler struct {
	service *Service
}

// NewHandler creates a new earnings handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ========================================
// EARNINGS DASHBOARD
// ========================================

// GetSummary returns earnings summary for a period
// GET /api/v1/driver/earnings/summary?period=this_week
func (h *Handler) GetSummary(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	period := c.DefaultQuery("period", "today")

	summary, err := h.service.GetEarningsSummary(c.Request.Context(), driverID, period)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get earnings summary")
		return
	}

	common.SuccessResponse(c, summary)
}

// GetDailyBreakdown returns day-by-day earnings
// GET /api/v1/driver/earnings/daily?period=this_week
func (h *Handler) GetDailyBreakdown(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	period := c.DefaultQuery("period", "this_week")

	daily, err := h.service.GetDailyEarnings(c.Request.Context(), driverID, period)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get daily earnings")
		return
	}

	common.SuccessResponse(c, gin.H{
		"daily":  daily,
		"period": period,
	})
}

// GetHistory returns paginated earnings history
// GET /api/v1/driver/earnings/history?period=this_month&page=1&page_size=20
func (h *Handler) GetHistory(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	period := c.DefaultQuery("period", "this_month")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	resp, err := h.service.GetEarningsHistory(c.Request.Context(), driverID, period, page, pageSize)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get earnings history")
		return
	}

	common.SuccessResponse(c, resp)
}

// GetBalance returns unpaid balance
// GET /api/v1/driver/earnings/balance
func (h *Handler) GetBalance(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	balance, err := h.service.GetUnpaidBalance(c.Request.Context(), driverID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get balance")
		return
	}

	common.SuccessResponse(c, gin.H{
		"unpaid_balance": balance,
		"currency":       "USD",
	})
}

// ========================================
// PAYOUTS
// ========================================

// RequestPayout requests a payout
// POST /api/v1/driver/earnings/payouts
func (h *Handler) RequestPayout(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req RequestPayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	payout, err := h.service.RequestPayout(c.Request.Context(), driverID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to request payout")
		return
	}

	common.CreatedResponse(c, payout)
}

// GetPayoutHistory returns payout history
// GET /api/v1/driver/earnings/payouts?page=1&page_size=20
func (h *Handler) GetPayoutHistory(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	resp, err := h.service.GetPayoutHistory(c.Request.Context(), driverID, page, pageSize)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get payout history")
		return
	}

	common.SuccessResponse(c, resp)
}

// ========================================
// BANK ACCOUNTS
// ========================================

// AddBankAccount adds a bank account
// POST /api/v1/driver/earnings/bank-accounts
func (h *Handler) AddBankAccount(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req AddBankAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	account, err := h.service.AddBankAccount(c.Request.Context(), driverID, &req)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to add bank account")
		return
	}

	common.CreatedResponse(c, account)
}

// GetBankAccounts returns all bank accounts
// GET /api/v1/driver/earnings/bank-accounts
func (h *Handler) GetBankAccounts(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	accounts, err := h.service.GetBankAccounts(c.Request.Context(), driverID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get bank accounts")
		return
	}

	common.SuccessResponse(c, gin.H{
		"accounts": accounts,
		"count":    len(accounts),
	})
}

// DeleteBankAccount removes a bank account
// DELETE /api/v1/driver/earnings/bank-accounts/:accountId
func (h *Handler) DeleteBankAccount(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	accountID, err := uuid.Parse(c.Param("accountId"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid account id")
		return
	}

	if err := h.service.DeleteBankAccount(c.Request.Context(), driverID, accountID); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to delete bank account")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "bank account removed"})
}

// ========================================
// EARNING GOALS
// ========================================

// SetGoal sets an earning goal
// POST /api/v1/driver/earnings/goals
func (h *Handler) SetGoal(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SetEarningGoalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	goal, err := h.service.SetEarningGoal(c.Request.Context(), driverID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to set earning goal")
		return
	}

	common.CreatedResponse(c, goal)
}

// GetGoals returns earning goals with progress
// GET /api/v1/driver/earnings/goals
func (h *Handler) GetGoals(c *gin.Context) {
	driverID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	goals, err := h.service.GetEarningGoals(c.Request.Context(), driverID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get earning goals")
		return
	}

	common.SuccessResponse(c, gin.H{"goals": goals})
}

// ========================================
// ADMIN ENDPOINTS
// ========================================

// AdminRecordBonus records a bonus for a driver (admin only)
// POST /api/v1/admin/earnings/bonus
func (h *Handler) AdminRecordBonus(c *gin.Context) {
	var req struct {
		DriverID    uuid.UUID `json:"driver_id" binding:"required"`
		Amount      float64   `json:"amount" binding:"required"`
		Description string    `json:"description" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	earning, err := h.service.RecordBonus(c.Request.Context(), req.DriverID, req.Amount, req.Description)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to record bonus")
		return
	}

	common.CreatedResponse(c, earning)
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers earnings routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	// Driver earnings routes
	driver := r.Group("/api/v1/driver/earnings")
	driver.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	driver.Use(middleware.RequireRole(models.RoleDriver))
	{
		// Dashboard
		driver.GET("/summary", h.GetSummary)
		driver.GET("/daily", h.GetDailyBreakdown)
		driver.GET("/history", h.GetHistory)
		driver.GET("/balance", h.GetBalance)

		// Payouts
		driver.POST("/payouts", h.RequestPayout)
		driver.GET("/payouts", h.GetPayoutHistory)

		// Bank accounts
		driver.POST("/bank-accounts", h.AddBankAccount)
		driver.GET("/bank-accounts", h.GetBankAccounts)
		driver.DELETE("/bank-accounts/:accountId", h.DeleteBankAccount)

		// Goals
		driver.POST("/goals", h.SetGoal)
		driver.GET("/goals", h.GetGoals)
	}

	// Admin routes
	admin := r.Group("/api/v1/admin/earnings")
	admin.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	admin.Use(middleware.RequireRole(models.RoleAdmin))
	{
		admin.POST("/bonus", h.AdminRecordBonus)
	}
}
