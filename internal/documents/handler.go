package documents

import (
	"context"
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

// Handler handles HTTP requests for documents
type Handler struct {
	service       *Service
	driverService DriverServiceInterface
}

// DriverServiceInterface defines methods needed from driver service
type DriverServiceInterface interface {
	GetDriverByUserID(ctx context.Context, userID uuid.UUID) (*models.Driver, error)
}

// NewHandler creates a new documents handler
func NewHandler(service *Service, driverService DriverServiceInterface) *Handler {
	return &Handler{
		service:       service,
		driverService: driverService,
	}
}

// getDriverID gets the driver ID from the authenticated user
func (h *Handler) getDriverID(c *gin.Context) (uuid.UUID, error) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return uuid.Nil, err
	}

	// Get driver by user ID
	driver, err := h.driverService.GetDriverByUserID(c.Request.Context(), userID)
	if err != nil {
		return uuid.Nil, err
	}

	return driver.ID, nil
}

// ========================================
// DOCUMENT TYPE ENDPOINTS
// ========================================

// GetDocumentTypes gets all available document types
// GET /api/v1/documents/types
func (h *Handler) GetDocumentTypes(c *gin.Context) {
	types, err := h.service.GetDocumentTypes(c.Request.Context())
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get document types")
		return
	}

	common.SuccessResponse(c, DocumentTypeListResponse{DocumentTypes: types})
}

// ========================================
// DRIVER DOCUMENT ENDPOINTS
// ========================================

// GetMyDocuments gets the authenticated driver's documents
// GET /api/v1/documents
func (h *Handler) GetMyDocuments(c *gin.Context) {
	driverID, err := h.getDriverID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "not a registered driver")
		return
	}

	docs, err := h.service.GetDriverDocuments(c.Request.Context(), driverID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get documents")
		return
	}

	common.SuccessResponse(c, gin.H{"documents": docs})
}

// GetMyVerificationStatus gets the driver's verification status
// GET /api/v1/documents/verification-status
func (h *Handler) GetMyVerificationStatus(c *gin.Context) {
	driverID, err := h.getDriverID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "not a registered driver")
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

// UploadDocument uploads a new document
// POST /api/v1/documents/upload
func (h *Handler) UploadDocument(c *gin.Context) {
	driverID, err := h.getDriverID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "not a registered driver")
		return
	}

	// Get file from form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	// Get document type from form
	documentTypeCode := c.PostForm("document_type_code")
	if documentTypeCode == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "document_type_code is required")
		return
	}

	// Build request
	req := &UploadDocumentRequest{
		DocumentTypeCode: documentTypeCode,
		DocumentNumber:   c.PostForm("document_number"),
		IssuingAuthority: c.PostForm("issuing_authority"),
	}

	// Parse dates if provided
	if issueDate := c.PostForm("issue_date"); issueDate != "" {
		// Parse as time.Time
		// For simplicity, dates are expected in YYYY-MM-DD format
	}
	if expiryDate := c.PostForm("expiry_date"); expiryDate != "" {
		// Parse as time.Time
	}

	response, err := h.service.UploadDocument(
		c.Request.Context(),
		driverID,
		req,
		file,
		header.Size,
		header.Filename,
		header.Header.Get("Content-Type"),
	)

	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to upload document")
		return
	}

	common.CreatedResponse(c, response)
}

// UploadDocumentBackSide uploads the back side of a document
// POST /api/v1/documents/:id/back
func (h *Handler) UploadDocumentBackSide(c *gin.Context) {
	documentIDStr := c.Param("id")
	documentID, err := uuid.Parse(documentIDStr)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid document ID")
		return
	}

	// Verify ownership
	driverID, err := h.getDriverID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "not a registered driver")
		return
	}

	doc, err := h.service.GetDocument(c.Request.Context(), documentID)
	if err != nil {
		common.ErrorResponse(c, http.StatusNotFound, "document not found")
		return
	}

	if doc.DriverID != driverID {
		common.ErrorResponse(c, http.StatusForbidden, "not your document")
		return
	}

	// Get file from form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	if err := h.service.UploadDocumentBackSide(
		c.Request.Context(),
		documentID,
		file,
		header.Size,
		header.Filename,
		header.Header.Get("Content-Type"),
	); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to upload back side")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Back side uploaded successfully"})
}

// GetPresignedUploadURL gets a presigned URL for direct upload
// POST /api/v1/documents/presigned-upload
func (h *Handler) GetPresignedUploadURL(c *gin.Context) {
	driverID, err := h.getDriverID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "not a registered driver")
		return
	}

	var req PresignedUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	response, err := h.service.GetPresignedUploadURL(c.Request.Context(), driverID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to generate upload URL")
		return
	}

	common.SuccessResponse(c, response)
}

// CompleteDirectUpload completes document creation after direct upload
// POST /api/v1/documents/upload-complete
func (h *Handler) CompleteDirectUpload(c *gin.Context) {
	driverID, err := h.getDriverID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "not a registered driver")
		return
	}

	var req UploadCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	response, err := h.service.CompleteDirectUpload(c.Request.Context(), driverID, &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to complete upload")
		return
	}

	common.SuccessResponse(c, response)
}

// GetDocument gets a specific document
// GET /api/v1/documents/:id
func (h *Handler) GetDocument(c *gin.Context) {
	documentIDStr := c.Param("id")
	documentID, err := uuid.Parse(documentIDStr)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid document ID")
		return
	}

	doc, err := h.service.GetDocument(c.Request.Context(), documentID)
	if err != nil {
		common.ErrorResponse(c, http.StatusNotFound, "document not found")
		return
	}

	// Verify ownership (unless admin)
	role, _ := middleware.GetUserRole(c)
	if role != models.RoleAdmin {
		driverID, err := h.getDriverID(c)
		if err != nil || doc.DriverID != driverID {
			common.ErrorResponse(c, http.StatusForbidden, "not your document")
			return
		}
	}

	common.SuccessResponse(c, doc)
}

// ========================================
// ADMIN ENDPOINTS
// ========================================

// GetPendingReviews gets documents pending review
// GET /api/v1/admin/documents/pending
func (h *Handler) GetPendingReviews(c *gin.Context) {
	params := pagination.ParseParams(c)

	reviews, total, err := h.service.GetPendingReviews(c.Request.Context(), params.Limit, params.Offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get pending reviews")
		return
	}

	meta := pagination.BuildMeta(params.Limit, params.Offset, int64(total))
	common.SuccessResponseWithMeta(c, gin.H{
		"documents": reviews,
	}, meta)
}

// GetExpiringDocuments gets documents expiring soon
// GET /api/v1/admin/documents/expiring
func (h *Handler) GetExpiringDocuments(c *gin.Context) {
	daysAhead, _ := strconv.Atoi(c.DefaultQuery("days", "30"))

	expiring, err := h.service.GetExpiringDocuments(c.Request.Context(), daysAhead)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get expiring documents")
		return
	}

	common.SuccessResponse(c, gin.H{"documents": expiring})
}

// StartDocumentReview marks a document as under review
// POST /api/v1/admin/documents/:id/start-review
func (h *Handler) StartDocumentReview(c *gin.Context) {
	documentIDStr := c.Param("id")
	documentID, err := uuid.Parse(documentIDStr)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid document ID")
		return
	}

	reviewerID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.service.StartReview(c.Request.Context(), documentID, reviewerID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to start review")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Review started"})
}

// ReviewDocument reviews a document (approve/reject)
// POST /api/v1/admin/documents/:id/review
func (h *Handler) ReviewDocument(c *gin.Context) {
	documentIDStr := c.Param("id")
	documentID, err := uuid.Parse(documentIDStr)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid document ID")
		return
	}

	reviewerID, err := middleware.GetUserID(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req ReviewDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.ReviewDocument(c.Request.Context(), documentID, reviewerID, &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to review document")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Document reviewed successfully"})
}

// GetDriverDocuments gets all documents for a specific driver (admin)
// GET /api/v1/admin/drivers/:driver_id/documents
func (h *Handler) GetDriverDocumentsAdmin(c *gin.Context) {
	driverIDStr := c.Param("driver_id")
	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid driver ID")
		return
	}

	docs, err := h.service.GetDriverDocuments(c.Request.Context(), driverID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get documents")
		return
	}

	common.SuccessResponse(c, gin.H{"documents": docs})
}

// GetDriverVerificationStatusAdmin gets verification status for a driver (admin)
// GET /api/v1/admin/drivers/:driver_id/verification-status
func (h *Handler) GetDriverVerificationStatusAdmin(c *gin.Context) {
	driverIDStr := c.Param("driver_id")
	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid driver ID")
		return
	}

	status, err := h.service.GetDriverVerificationStatus(c.Request.Context(), driverID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get verification status")
		return
	}

	common.SuccessResponse(c, status)
}

// ========================================
// ROUTE REGISTRATION
// ========================================

// RegisterRoutes registers document routes
func (h *Handler) RegisterRoutes(r *gin.Engine, jwtProvider jwtkeys.KeyProvider) {
	// Public routes (document types)
	docs := r.Group("/api/v1/documents")
	{
		docs.GET("/types", h.GetDocumentTypes)
	}

	// Driver routes (authenticated)
	driverDocs := r.Group("/api/v1/documents")
	driverDocs.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	driverDocs.Use(middleware.RequireRole(models.RoleDriver))
	{
		driverDocs.GET("", h.GetMyDocuments)
		driverDocs.GET("/verification-status", h.GetMyVerificationStatus)
		driverDocs.POST("/upload", h.UploadDocument)
		driverDocs.POST("/presigned-upload", h.GetPresignedUploadURL)
		driverDocs.POST("/upload-complete", h.CompleteDirectUpload)
		driverDocs.GET("/:id", h.GetDocument)
		driverDocs.POST("/:id/back", h.UploadDocumentBackSide)
	}

	// Admin routes
	adminDocs := r.Group("/api/v1/admin/documents")
	adminDocs.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	adminDocs.Use(middleware.RequireRole(models.RoleAdmin))
	{
		adminDocs.GET("/pending", h.GetPendingReviews)
		adminDocs.GET("/expiring", h.GetExpiringDocuments)
		adminDocs.POST("/:id/start-review", h.StartDocumentReview)
		adminDocs.POST("/:id/review", h.ReviewDocument)
	}

	// Admin driver documents
	adminDriverDocs := r.Group("/api/v1/admin/drivers")
	adminDriverDocs.Use(middleware.AuthMiddlewareWithProvider(jwtProvider))
	adminDriverDocs.Use(middleware.RequireRole(models.RoleAdmin))
	{
		adminDriverDocs.GET("/:driver_id/documents", h.GetDriverDocumentsAdmin)
		adminDriverDocs.GET("/:driver_id/verification-status", h.GetDriverVerificationStatusAdmin)
	}
}

// RegisterAdminRoutes registers only admin document routes on an existing router group.
func (h *Handler) RegisterAdminRoutes(rg *gin.RouterGroup) {
	documents := rg.Group("/documents")
	{
		documents.GET("/pending", h.GetPendingReviews)
		documents.GET("/expiring", h.GetExpiringDocuments)
		documents.POST("/:id/start-review", h.StartDocumentReview)
		documents.POST("/:id/review", h.ReviewDocument)
		documents.GET("/drivers/:driver_id", h.GetDriverDocumentsAdmin)
		documents.GET("/drivers/:driver_id/verification-status", h.GetDriverVerificationStatusAdmin)
	}
}

// RegisterRoutesOnGroup registers document routes on an existing router group
func (h *Handler) RegisterRoutesOnGroup(rg *gin.RouterGroup) {
	// Document types (public within API)
	rg.GET("/documents/types", h.GetDocumentTypes)

	// Driver document routes
	docs := rg.Group("/documents")
	docs.Use(middleware.RequireRole(models.RoleDriver))
	{
		docs.GET("", h.GetMyDocuments)
		docs.GET("/verification-status", h.GetMyVerificationStatus)
		docs.POST("/upload", h.UploadDocument)
		docs.POST("/presigned-upload", h.GetPresignedUploadURL)
		docs.POST("/upload-complete", h.CompleteDirectUpload)
		docs.GET("/:id", h.GetDocument)
		docs.POST("/:id/back", h.UploadDocumentBackSide)
	}
}
