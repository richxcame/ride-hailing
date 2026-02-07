package documents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/richxcame/ride-hailing/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// Mock Implementations
// ============================================================================

// MockDriverService implements DriverServiceInterface for testing
type MockDriverService struct {
	mock.Mock
}

func (m *MockDriverService) GetDriverByUserID(ctx context.Context, userID uuid.UUID) (*models.Driver, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Driver), args.Error(1)
}

// MockStorageHandler implements storage.Storage for testing (renamed to avoid conflict with service_test.go)
type MockStorageHandler struct {
	mock.Mock
}

func (m *MockStorageHandler) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*storage.UploadResult, error) {
	args := m.Called(ctx, key, reader, size, contentType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.UploadResult), args.Error(1)
}

func (m *MockStorageHandler) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockStorageHandler) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockStorageHandler) GetURL(key string) string {
	args := m.Called(key)
	return args.String(0)
}

func (m *MockStorageHandler) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockStorageHandler) GetPresignedUploadURL(ctx context.Context, key, contentType string, expiry time.Duration) (*storage.PresignedURLResult, error) {
	args := m.Called(ctx, key, contentType, expiry)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.PresignedURLResult), args.Error(1)
}

func (m *MockStorageHandler) GetPresignedDownloadURL(ctx context.Context, key string, expiry time.Duration) (*storage.PresignedURLResult, error) {
	args := m.Called(ctx, key, expiry)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.PresignedURLResult), args.Error(1)
}

func (m *MockStorageHandler) Copy(ctx context.Context, sourceKey, destKey string) error {
	args := m.Called(ctx, sourceKey, destKey)
	return args.Error(0)
}

// MockRepositoryTestify is a testify/mock based repository mock
type MockRepositoryTestify struct {
	mock.Mock
}

func (m *MockRepositoryTestify) GetDocumentTypes(ctx context.Context) ([]*DocumentType, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DocumentType), args.Error(1)
}

func (m *MockRepositoryTestify) GetDocumentTypeByCode(ctx context.Context, code string) (*DocumentType, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DocumentType), args.Error(1)
}

func (m *MockRepositoryTestify) GetRequiredDocumentTypes(ctx context.Context) ([]*DocumentType, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DocumentType), args.Error(1)
}

func (m *MockRepositoryTestify) CreateDocument(ctx context.Context, doc *DriverDocument) error {
	args := m.Called(ctx, doc)
	return args.Error(0)
}

func (m *MockRepositoryTestify) GetDocument(ctx context.Context, documentID uuid.UUID) (*DriverDocument, error) {
	args := m.Called(ctx, documentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverDocument), args.Error(1)
}

func (m *MockRepositoryTestify) GetDriverDocuments(ctx context.Context, driverID uuid.UUID) ([]*DriverDocument, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DriverDocument), args.Error(1)
}

func (m *MockRepositoryTestify) GetLatestDocumentByType(ctx context.Context, driverID, documentTypeID uuid.UUID) (*DriverDocument, error) {
	args := m.Called(ctx, driverID, documentTypeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverDocument), args.Error(1)
}

func (m *MockRepositoryTestify) UpdateDocumentStatus(ctx context.Context, documentID uuid.UUID, status DocumentStatus, reviewedBy *uuid.UUID, reviewNotes, rejectionReason *string) error {
	args := m.Called(ctx, documentID, status, reviewedBy, reviewNotes, rejectionReason)
	return args.Error(0)
}

func (m *MockRepositoryTestify) UpdateDocumentOCRData(ctx context.Context, documentID uuid.UUID, ocrData map[string]interface{}, confidence float64) error {
	args := m.Called(ctx, documentID, ocrData, confidence)
	return args.Error(0)
}

func (m *MockRepositoryTestify) UpdateDocumentDetails(ctx context.Context, documentID uuid.UUID, documentNumber *string, issueDate, expiryDate *time.Time, issuingAuthority *string) error {
	args := m.Called(ctx, documentID, documentNumber, issueDate, expiryDate, issuingAuthority)
	return args.Error(0)
}

func (m *MockRepositoryTestify) SupersedeDocument(ctx context.Context, documentID uuid.UUID) error {
	args := m.Called(ctx, documentID)
	return args.Error(0)
}

func (m *MockRepositoryTestify) UpdateDocumentBackFile(ctx context.Context, documentID uuid.UUID, backFileURL, backFileKey string) error {
	args := m.Called(ctx, documentID, backFileURL, backFileKey)
	return args.Error(0)
}

func (m *MockRepositoryTestify) GetDriverVerificationStatus(ctx context.Context, driverID uuid.UUID) (*DriverVerificationStatus, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverVerificationStatus), args.Error(1)
}

func (m *MockRepositoryTestify) GetPendingReviews(ctx context.Context, limit, offset int) ([]*PendingReviewDocument, int, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*PendingReviewDocument), args.Int(1), args.Error(2)
}

func (m *MockRepositoryTestify) GetExpiringDocuments(ctx context.Context, daysAhead int) ([]*ExpiringDocument, error) {
	args := m.Called(ctx, daysAhead)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ExpiringDocument), args.Error(1)
}

func (m *MockRepositoryTestify) CreateHistory(ctx context.Context, history *DocumentVerificationHistory) error {
	args := m.Called(ctx, history)
	return args.Error(0)
}

func (m *MockRepositoryTestify) GetDocumentHistory(ctx context.Context, documentID uuid.UUID) ([]*DocumentVerificationHistory, error) {
	args := m.Called(ctx, documentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DocumentVerificationHistory), args.Error(1)
}

func (m *MockRepositoryTestify) CreateOCRJob(ctx context.Context, job *OCRProcessingQueue) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockRepositoryTestify) GetPendingOCRJobs(ctx context.Context, limit int) ([]*OCRProcessingQueue, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*OCRProcessingQueue), args.Error(1)
}

func (m *MockRepositoryTestify) UpdateOCRJobStatus(ctx context.Context, jobID uuid.UUID, status string, result, errorMsg *string) error {
	args := m.Called(ctx, jobID, status, result, errorMsg)
	return args.Error(0)
}

func (m *MockRepositoryTestify) CompleteOCRJob(ctx context.Context, jobID uuid.UUID, extractedData map[string]interface{}, confidence float64, processingTimeMs int) error {
	args := m.Called(ctx, jobID, extractedData, confidence, processingTimeMs)
	return args.Error(0)
}

func (m *MockRepositoryTestify) FailOCRJob(ctx context.Context, jobID uuid.UUID, errorMessage string) error {
	args := m.Called(ctx, jobID, errorMessage)
	return args.Error(0)
}

func (m *MockRepositoryTestify) UpdateOCRJobRetry(ctx context.Context, jobID uuid.UUID, retryCount int, nextRetry time.Time) error {
	args := m.Called(ctx, jobID, retryCount, nextRetry)
	return args.Error(0)
}

// ============================================================================
// Helper Functions
// ============================================================================

func setupTestContext(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	c.Request = req

	return c, w
}

func setUserContext(c *gin.Context, userID uuid.UUID, role models.UserRole) {
	c.Set("user_id", userID)
	c.Set("user_role", role)
	c.Set("user_email", "test@example.com")
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

func createTestDriver(userID uuid.UUID) *models.Driver {
	return &models.Driver{
		ID:            uuid.New(),
		UserID:        userID,
		LicenseNumber: "DL12345",
		VehicleModel:  "Toyota Camry",
		VehiclePlate:  "ABC123",
		VehicleColor:  "Black",
		VehicleYear:   2020,
		IsAvailable:   true,
		IsOnline:      true,
		Rating:        4.5,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func createTestDocumentTypeHandler() *DocumentType {
	return &DocumentType{
		ID:                    uuid.New(),
		Code:                  "drivers_license",
		Name:                  "Driver's License",
		IsRequired:            true,
		RequiresExpiry:        true,
		RequiresFrontBack:     true,
		DefaultValidityMonths: 12,
		RenewalReminderDays:   30,
		RequiresManualReview:  true,
		AutoOCREnabled:        false,
		IsActive:              true,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
}

func createTestDriverDocument(driverID uuid.UUID, docType *DocumentType) *DriverDocument {
	return &DriverDocument{
		ID:             uuid.New(),
		DriverID:       driverID,
		DocumentTypeID: docType.ID,
		Status:         StatusPending,
		FileURL:        "https://storage.example.com/documents/test.jpg",
		FileKey:        "documents/test.jpg",
		FileName:       "test.jpg",
		Version:        1,
		SubmittedAt:    time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		DocumentType:   docType,
	}
}

func createTestHandler(mockRepo *MockRepositoryTestify, mockStorage *MockStorageHandler, mockDriverService *MockDriverService) *Handler {
	service := NewService(mockRepo, mockStorage, ServiceConfig{
		MaxFileSizeMB:    10,
		AllowedMimeTypes: []string{"image/jpeg", "image/png", "application/pdf"},
	})
	return NewHandler(service, mockDriverService)
}

func createMultipartFormRequest(method, path string, fields map[string]string, fileContent []byte, fileName, fieldName string) *http.Request {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields
	for key, value := range fields {
		writer.WriteField(key, value)
	}

	// Add file if provided
	if fileContent != nil {
		part, _ := writer.CreateFormFile(fieldName, fileName)
		part.Write(fileContent)
	}

	writer.Close()

	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

// ============================================================================
// GetDocumentTypes Handler Tests
// ============================================================================

func TestHandler_GetDocumentTypes_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	docTypes := []*DocumentType{
		createTestDocumentTypeHandler(),
		{
			ID:       uuid.New(),
			Code:     "vehicle_registration",
			Name:     "Vehicle Registration",
			IsActive: true,
		},
	}

	mockRepo.On("GetDocumentTypes", mock.Anything).Return(docTypes, nil)

	c, w := setupTestContext("GET", "/api/v1/documents/types", nil)

	handler.GetDocumentTypes(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetDocumentTypes_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	mockRepo.On("GetDocumentTypes", mock.Anything).Return([]*DocumentType{}, nil)

	c, w := setupTestContext("GET", "/api/v1/documents/types", nil)

	handler.GetDocumentTypes(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetDocumentTypes_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	mockRepo.On("GetDocumentTypes", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/documents/types", nil)

	handler.GetDocumentTypes(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

// ============================================================================
// GetMyDocuments Handler Tests
// ============================================================================

func TestHandler_GetMyDocuments_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	userID := uuid.New()
	driver := createTestDriver(userID)
	docType := createTestDocumentTypeHandler()
	docs := []*DriverDocument{
		createTestDriverDocument(driver.ID, docType),
	}

	mockDriverService.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockRepo.On("GetDriverDocuments", mock.Anything, driver.ID).Return(docs, nil)

	c, w := setupTestContext("GET", "/api/v1/documents", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetMyDocuments(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockDriverService.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetMyDocuments_Unauthorized_NoUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	c, w := setupTestContext("GET", "/api/v1/documents", nil)
	// Don't set user context

	handler.GetMyDocuments(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.False(t, response["success"].(bool))
}

func TestHandler_GetMyDocuments_NotADriver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	userID := uuid.New()

	mockDriverService.On("GetDriverByUserID", mock.Anything, userID).Return(nil, errors.New("driver not found"))

	c, w := setupTestContext("GET", "/api/v1/documents", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetMyDocuments(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "not a registered driver")
}

func TestHandler_GetMyDocuments_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	userID := uuid.New()
	driver := createTestDriver(userID)

	mockDriverService.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockRepo.On("GetDriverDocuments", mock.Anything, driver.ID).Return([]*DriverDocument{}, nil)

	c, w := setupTestContext("GET", "/api/v1/documents", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetMyDocuments(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetMyDocuments_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	userID := uuid.New()
	driver := createTestDriver(userID)

	mockDriverService.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockRepo.On("GetDriverDocuments", mock.Anything, driver.ID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/documents", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetMyDocuments(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetMyVerificationStatus Handler Tests
// ============================================================================

func TestHandler_GetMyVerificationStatus_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	userID := uuid.New()
	driver := createTestDriver(userID)
	docType := createTestDocumentTypeHandler()

	mockDriverService.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockRepo.On("GetRequiredDocumentTypes", mock.Anything).Return([]*DocumentType{docType}, nil)
	mockRepo.On("GetDriverDocuments", mock.Anything, driver.ID).Return([]*DriverDocument{}, nil)
	mockRepo.On("GetDriverVerificationStatus", mock.Anything, driver.ID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/documents/verification-status", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetMyVerificationStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetMyVerificationStatus_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	c, w := setupTestContext("GET", "/api/v1/documents/verification-status", nil)
	// Don't set user context

	handler.GetMyVerificationStatus(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetMyVerificationStatus_NotADriver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	userID := uuid.New()

	mockDriverService.On("GetDriverByUserID", mock.Anything, userID).Return(nil, errors.New("driver not found"))

	c, w := setupTestContext("GET", "/api/v1/documents/verification-status", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetMyVerificationStatus(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================================
// GetDocument Handler Tests
// ============================================================================

func TestHandler_GetDocument_Success_AsOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	userID := uuid.New()
	driver := createTestDriver(userID)
	docType := createTestDocumentTypeHandler()
	doc := createTestDriverDocument(driver.ID, docType)

	mockRepo.On("GetDocument", mock.Anything, doc.ID).Return(doc, nil)
	mockDriverService.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)

	c, w := setupTestContext("GET", "/api/v1/documents/"+doc.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: doc.ID.String()}}
	setUserContext(c, userID, models.RoleDriver)

	handler.GetDocument(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetDocument_Success_AsAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()
	driverID := uuid.New()
	docType := createTestDocumentTypeHandler()
	doc := createTestDriverDocument(driverID, docType)

	mockRepo.On("GetDocument", mock.Anything, doc.ID).Return(doc, nil)

	c, w := setupTestContext("GET", "/api/v1/documents/"+doc.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: doc.ID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetDocument(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetDocument_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/documents/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID, models.RoleDriver)

	handler.GetDocument(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid document ID")
}

func TestHandler_GetDocument_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	userID := uuid.New()
	documentID := uuid.New()

	mockRepo.On("GetDocument", mock.Anything, documentID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/documents/"+documentID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: documentID.String()}}
	setUserContext(c, userID, models.RoleDriver)

	handler.GetDocument(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetDocument_Forbidden_NotOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	userID := uuid.New()
	otherDriverID := uuid.New()
	driver := createTestDriver(userID)
	docType := createTestDocumentTypeHandler()
	doc := createTestDriverDocument(otherDriverID, docType)

	mockRepo.On("GetDocument", mock.Anything, doc.ID).Return(doc, nil)
	mockDriverService.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)

	c, w := setupTestContext("GET", "/api/v1/documents/"+doc.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: doc.ID.String()}}
	setUserContext(c, userID, models.RoleDriver)

	handler.GetDocument(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "not your document")
}

// ============================================================================
// GetPendingReviews Handler Tests
// ============================================================================

func TestHandler_GetPendingReviews_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()
	docType := createTestDocumentTypeHandler()
	driverID := uuid.New()
	doc := createTestDriverDocument(driverID, docType)
	pendingDocs := []*PendingReviewDocument{
		{
			Document:      doc,
			DriverName:    "John Doe",
			DriverPhone:   "+1234567890",
			DriverEmail:   "john@example.com",
			DocumentType:  docType.Name,
			HoursPending:  2.5,
		},
	}

	mockRepo.On("GetPendingReviews", mock.Anything, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(pendingDocs, 1, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/documents/pending", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetPendingReviews(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_GetPendingReviews_WithPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()

	mockRepo.On("GetPendingReviews", mock.Anything, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return([]*PendingReviewDocument{}, 50, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/documents/pending?limit=10&offset=20", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetPendingReviews(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	// Pagination params are parsed - verify response has meta
	_, hasMeta := response["meta"]
	assert.True(t, hasMeta)
}

func TestHandler_GetPendingReviews_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()

	mockRepo.On("GetPendingReviews", mock.Anything, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return([]*PendingReviewDocument{}, 0, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/documents/pending", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetPendingReviews(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetPendingReviews_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()

	mockRepo.On("GetPendingReviews", mock.Anything, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return(nil, 0, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/documents/pending", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetPendingReviews(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetExpiringDocuments Handler Tests
// ============================================================================

func TestHandler_GetExpiringDocuments_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()
	expiringDocs := []*ExpiringDocument{
		{
			DriverName:      "John Doe",
			DocumentType:    "Driver's License",
			DaysUntilExpiry: 15,
			Urgency:         "warning",
		},
	}

	mockRepo.On("GetExpiringDocuments", mock.Anything, 30).Return(expiringDocs, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/documents/expiring", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetExpiringDocuments(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetExpiringDocuments_WithCustomDays(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()

	mockRepo.On("GetExpiringDocuments", mock.Anything, 60).Return([]*ExpiringDocument{}, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/documents/expiring?days=60", nil)
	c.Request.URL.RawQuery = "days=60"
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetExpiringDocuments(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetExpiringDocuments_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()

	mockRepo.On("GetExpiringDocuments", mock.Anything, 30).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/documents/expiring", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetExpiringDocuments(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// StartDocumentReview Handler Tests
// ============================================================================

func TestHandler_StartDocumentReview_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()
	driverID := uuid.New()
	docType := createTestDocumentTypeHandler()
	doc := createTestDriverDocument(driverID, docType)
	doc.Status = StatusPending

	mockRepo.On("GetDocument", mock.Anything, doc.ID).Return(doc, nil)
	mockRepo.On("UpdateDocumentStatus", mock.Anything, doc.ID, StatusUnderReview, mock.AnythingOfType("*uuid.UUID"), (*string)(nil), (*string)(nil)).Return(nil)
	mockRepo.On("CreateHistory", mock.Anything, mock.AnythingOfType("*documents.DocumentVerificationHistory")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/documents/"+doc.ID.String()+"/start-review", nil)
	c.Params = gin.Params{{Key: "id", Value: doc.ID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.StartDocumentReview(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_StartDocumentReview_InvalidDocumentID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/admin/documents/invalid-uuid/start-review", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.StartDocumentReview(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_StartDocumentReview_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	documentID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/admin/documents/"+documentID.String()+"/start-review", nil)
	c.Params = gin.Params{{Key: "id", Value: documentID.String()}}
	// Don't set user context

	handler.StartDocumentReview(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_StartDocumentReview_DocumentNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()
	documentID := uuid.New()

	mockRepo.On("GetDocument", mock.Anything, documentID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("POST", "/api/v1/admin/documents/"+documentID.String()+"/start-review", nil)
	c.Params = gin.Params{{Key: "id", Value: documentID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.StartDocumentReview(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_StartDocumentReview_NotPending(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()
	driverID := uuid.New()
	docType := createTestDocumentTypeHandler()
	doc := createTestDriverDocument(driverID, docType)
	doc.Status = StatusApproved

	mockRepo.On("GetDocument", mock.Anything, doc.ID).Return(doc, nil)

	c, w := setupTestContext("POST", "/api/v1/admin/documents/"+doc.ID.String()+"/start-review", nil)
	c.Params = gin.Params{{Key: "id", Value: doc.ID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.StartDocumentReview(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// ReviewDocument Handler Tests
// ============================================================================

func TestHandler_ReviewDocument_Approve_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()
	driverID := uuid.New()
	docType := createTestDocumentTypeHandler()
	doc := createTestDriverDocument(driverID, docType)
	doc.Status = StatusUnderReview

	reqBody := ReviewDocumentRequest{
		Action: "approve",
		Notes:  "Document verified successfully",
	}

	mockRepo.On("GetDocument", mock.Anything, doc.ID).Return(doc, nil)
	mockRepo.On("UpdateDocumentStatus", mock.Anything, doc.ID, StatusApproved, mock.AnythingOfType("*uuid.UUID"), mock.AnythingOfType("*string"), (*string)(nil)).Return(nil)
	mockRepo.On("CreateHistory", mock.Anything, mock.AnythingOfType("*documents.DocumentVerificationHistory")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/documents/"+doc.ID.String()+"/review", reqBody)
	c.Params = gin.Params{{Key: "id", Value: doc.ID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ReviewDocument(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_ReviewDocument_Reject_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()
	driverID := uuid.New()
	docType := createTestDocumentTypeHandler()
	doc := createTestDriverDocument(driverID, docType)
	doc.Status = StatusPending

	reqBody := ReviewDocumentRequest{
		Action:          "reject",
		RejectionReason: "Document is blurry and unreadable",
	}

	mockRepo.On("GetDocument", mock.Anything, doc.ID).Return(doc, nil)
	mockRepo.On("UpdateDocumentStatus", mock.Anything, doc.ID, StatusRejected, mock.AnythingOfType("*uuid.UUID"), (*string)(nil), mock.AnythingOfType("*string")).Return(nil)
	mockRepo.On("CreateHistory", mock.Anything, mock.AnythingOfType("*documents.DocumentVerificationHistory")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/documents/"+doc.ID.String()+"/review", reqBody)
	c.Params = gin.Params{{Key: "id", Value: doc.ID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ReviewDocument(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ReviewDocument_Reject_MissingReason(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()
	driverID := uuid.New()
	docType := createTestDocumentTypeHandler()
	doc := createTestDriverDocument(driverID, docType)
	doc.Status = StatusPending

	reqBody := ReviewDocumentRequest{
		Action: "reject",
		// No rejection reason
	}

	mockRepo.On("GetDocument", mock.Anything, doc.ID).Return(doc, nil)

	c, w := setupTestContext("POST", "/api/v1/admin/documents/"+doc.ID.String()+"/review", reqBody)
	c.Params = gin.Params{{Key: "id", Value: doc.ID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ReviewDocument(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ReviewDocument_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()
	documentID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/admin/documents/"+documentID.String()+"/review", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/documents/"+documentID.String()+"/review", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: documentID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ReviewDocument(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ReviewDocument_InvalidDocumentID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()

	reqBody := ReviewDocumentRequest{
		Action: "approve",
	}

	c, w := setupTestContext("POST", "/api/v1/admin/documents/invalid-uuid/review", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ReviewDocument(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ReviewDocument_InvalidAction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()
	driverID := uuid.New()
	docType := createTestDocumentTypeHandler()
	doc := createTestDriverDocument(driverID, docType)
	doc.Status = StatusPending

	reqBody := map[string]interface{}{
		"action": "invalid_action",
	}

	c, w := setupTestContext("POST", "/api/v1/admin/documents/"+doc.ID.String()+"/review", reqBody)
	c.Params = gin.Params{{Key: "id", Value: doc.ID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ReviewDocument(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ReviewDocument_DocumentNotPending(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()
	driverID := uuid.New()
	docType := createTestDocumentTypeHandler()
	doc := createTestDriverDocument(driverID, docType)
	doc.Status = StatusApproved

	reqBody := ReviewDocumentRequest{
		Action: "approve",
	}

	mockRepo.On("GetDocument", mock.Anything, doc.ID).Return(doc, nil)

	c, w := setupTestContext("POST", "/api/v1/admin/documents/"+doc.ID.String()+"/review", reqBody)
	c.Params = gin.Params{{Key: "id", Value: doc.ID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ReviewDocument(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// GetDriverDocumentsAdmin Handler Tests
// ============================================================================

func TestHandler_GetDriverDocumentsAdmin_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()
	driverID := uuid.New()
	docType := createTestDocumentTypeHandler()
	docs := []*DriverDocument{
		createTestDriverDocument(driverID, docType),
	}

	mockRepo.On("GetDriverDocuments", mock.Anything, driverID).Return(docs, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/drivers/"+driverID.String()+"/documents", nil)
	c.Params = gin.Params{{Key: "driver_id", Value: driverID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetDriverDocumentsAdmin(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetDriverDocumentsAdmin_InvalidDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/admin/drivers/invalid-uuid/documents", nil)
	c.Params = gin.Params{{Key: "driver_id", Value: "invalid-uuid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetDriverDocumentsAdmin(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetDriverDocumentsAdmin_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()
	driverID := uuid.New()

	mockRepo.On("GetDriverDocuments", mock.Anything, driverID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/drivers/"+driverID.String()+"/documents", nil)
	c.Params = gin.Params{{Key: "driver_id", Value: driverID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetDriverDocumentsAdmin(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetDriverVerificationStatusAdmin Handler Tests
// ============================================================================

func TestHandler_GetDriverVerificationStatusAdmin_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()
	driverID := uuid.New()
	docType := createTestDocumentTypeHandler()

	mockRepo.On("GetRequiredDocumentTypes", mock.Anything).Return([]*DocumentType{docType}, nil)
	mockRepo.On("GetDriverDocuments", mock.Anything, driverID).Return([]*DriverDocument{}, nil)
	mockRepo.On("GetDriverVerificationStatus", mock.Anything, driverID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/admin/drivers/"+driverID.String()+"/verification-status", nil)
	c.Params = gin.Params{{Key: "driver_id", Value: driverID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetDriverVerificationStatusAdmin(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetDriverVerificationStatusAdmin_InvalidDriverID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/admin/drivers/invalid-uuid/verification-status", nil)
	c.Params = gin.Params{{Key: "driver_id", Value: "invalid-uuid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetDriverVerificationStatusAdmin(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// GetPresignedUploadURL Handler Tests
// ============================================================================

func TestHandler_GetPresignedUploadURL_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	userID := uuid.New()
	driver := createTestDriver(userID)
	docType := createTestDocumentTypeHandler()

	reqBody := PresignedUploadRequest{
		DocumentTypeCode: "drivers_license",
		FileName:         "license.jpg",
		ContentType:      "image/jpeg",
		IsFrontSide:      true,
	}

	mockDriverService.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockRepo.On("GetDocumentTypeByCode", mock.Anything, "drivers_license").Return(docType, nil)
	mockStorage.On("GetPresignedUploadURL", mock.Anything, mock.AnythingOfType("string"), "image/jpeg", 15*time.Minute).Return(&storage.PresignedURLResult{
		URL:       "https://storage.example.com/upload?signature=abc123",
		Method:    "PUT",
		Headers:   map[string]string{"Content-Type": "image/jpeg"},
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}, nil)

	c, w := setupTestContext("POST", "/api/v1/documents/presigned-upload", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetPresignedUploadURL(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetPresignedUploadURL_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	userID := uuid.New()
	driver := createTestDriver(userID)

	mockDriverService.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)

	c, w := setupTestContext("POST", "/api/v1/documents/presigned-upload", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/documents/presigned-upload", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleDriver)

	handler.GetPresignedUploadURL(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetPresignedUploadURL_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	reqBody := PresignedUploadRequest{
		DocumentTypeCode: "drivers_license",
		FileName:         "license.jpg",
		ContentType:      "image/jpeg",
	}

	c, w := setupTestContext("POST", "/api/v1/documents/presigned-upload", reqBody)
	// Don't set user context

	handler.GetPresignedUploadURL(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetPresignedUploadURL_InvalidDocumentType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	userID := uuid.New()
	driver := createTestDriver(userID)

	reqBody := PresignedUploadRequest{
		DocumentTypeCode: "invalid_type",
		FileName:         "doc.pdf",
		ContentType:      "application/pdf",
	}

	mockDriverService.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockRepo.On("GetDocumentTypeByCode", mock.Anything, "invalid_type").Return(nil, errors.New("not found"))

	c, w := setupTestContext("POST", "/api/v1/documents/presigned-upload", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetPresignedUploadURL(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// CompleteDirectUpload Handler Tests
// ============================================================================

func TestHandler_CompleteDirectUpload_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	userID := uuid.New()
	driver := createTestDriver(userID)
	docType := createTestDocumentTypeHandler()

	reqBody := UploadCompleteRequest{
		FileKey:          "documents/driver123/license.jpg",
		DocumentTypeCode: "drivers_license",
		IsFrontSide:      true,
	}

	mockDriverService.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockStorage.On("Exists", mock.Anything, "documents/driver123/license.jpg").Return(true, nil)
	mockRepo.On("GetDocumentTypeByCode", mock.Anything, "drivers_license").Return(docType, nil)
	mockRepo.On("GetLatestDocumentByType", mock.Anything, driver.ID, docType.ID).Return(nil, errors.New("not found"))
	mockStorage.On("GetURL", "documents/driver123/license.jpg").Return("https://storage.example.com/documents/driver123/license.jpg")
	mockRepo.On("CreateDocument", mock.Anything, mock.AnythingOfType("*documents.DriverDocument")).Return(nil)
	mockRepo.On("CreateHistory", mock.Anything, mock.AnythingOfType("*documents.DocumentVerificationHistory")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/documents/upload-complete", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.CompleteDirectUpload(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_CompleteDirectUpload_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	userID := uuid.New()
	driver := createTestDriver(userID)

	mockDriverService.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)

	c, w := setupTestContext("POST", "/api/v1/documents/upload-complete", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/documents/upload-complete", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleDriver)

	handler.CompleteDirectUpload(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CompleteDirectUpload_FileNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	userID := uuid.New()
	driver := createTestDriver(userID)

	reqBody := UploadCompleteRequest{
		FileKey:          "documents/nonexistent.jpg",
		DocumentTypeCode: "drivers_license",
		IsFrontSide:      true,
	}

	mockDriverService.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockStorage.On("Exists", mock.Anything, "documents/nonexistent.jpg").Return(false, nil)

	c, w := setupTestContext("POST", "/api/v1/documents/upload-complete", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.CompleteDirectUpload(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// UploadDocumentBackSide Handler Tests
// ============================================================================

func TestHandler_UploadDocumentBackSide_InvalidDocumentID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/documents/invalid-uuid/back", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID, models.RoleDriver)

	handler.UploadDocumentBackSide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UploadDocumentBackSide_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	documentID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/documents/"+documentID.String()+"/back", nil)
	c.Params = gin.Params{{Key: "id", Value: documentID.String()}}
	// Don't set user context

	handler.UploadDocumentBackSide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_UploadDocumentBackSide_DocumentNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	userID := uuid.New()
	driver := createTestDriver(userID)
	documentID := uuid.New()

	mockDriverService.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockRepo.On("GetDocument", mock.Anything, documentID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("POST", "/api/v1/documents/"+documentID.String()+"/back", nil)
	c.Params = gin.Params{{Key: "id", Value: documentID.String()}}
	setUserContext(c, userID, models.RoleDriver)

	handler.UploadDocumentBackSide(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_UploadDocumentBackSide_NotOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	userID := uuid.New()
	driver := createTestDriver(userID)
	otherDriverID := uuid.New()
	docType := createTestDocumentTypeHandler()
	doc := createTestDriverDocument(otherDriverID, docType)

	mockDriverService.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
	mockRepo.On("GetDocument", mock.Anything, doc.ID).Return(doc, nil)

	c, w := setupTestContext("POST", "/api/v1/documents/"+doc.ID.String()+"/back", nil)
	c.Params = gin.Params{{Key: "id", Value: doc.ID.String()}}
	setUserContext(c, userID, models.RoleDriver)

	handler.UploadDocumentBackSide(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_GetDocument_AuthorizationCases(t *testing.T) {
	tests := []struct {
		name           string
		role           models.UserRole
		isOwner        bool
		expectedStatus int
	}{
		{
			name:           "admin can access any document",
			role:           models.RoleAdmin,
			isOwner:        false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "driver can access own document",
			role:           models.RoleDriver,
			isOwner:        true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "driver cannot access other's document",
			role:           models.RoleDriver,
			isOwner:        false,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepositoryTestify)
			mockStorage := new(MockStorageHandler)
			mockDriverService := new(MockDriverService)
			handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

			userID := uuid.New()
			driver := createTestDriver(userID)
			docType := createTestDocumentTypeHandler()

			var doc *DriverDocument
			if tt.isOwner {
				doc = createTestDriverDocument(driver.ID, docType)
			} else {
				doc = createTestDriverDocument(uuid.New(), docType)
			}

			mockRepo.On("GetDocument", mock.Anything, doc.ID).Return(doc, nil)
			if tt.role != models.RoleAdmin {
				mockDriverService.On("GetDriverByUserID", mock.Anything, userID).Return(driver, nil)
			}

			c, w := setupTestContext("GET", "/api/v1/documents/"+doc.ID.String(), nil)
			c.Params = gin.Params{{Key: "id", Value: doc.ID.String()}}
			setUserContext(c, userID, tt.role)

			handler.GetDocument(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_ReviewDocument_ActionCases(t *testing.T) {
	tests := []struct {
		name            string
		action          string
		rejectionReason string
		expectedStatus  int
	}{
		{
			name:           "approve action",
			action:         "approve",
			expectedStatus: http.StatusOK,
		},
		{
			name:            "reject action with reason",
			action:          "reject",
			rejectionReason: "Document expired",
			expectedStatus:  http.StatusOK,
		},
		{
			name:            "request_resubmit action",
			action:          "request_resubmit",
			rejectionReason: "Better quality needed",
			expectedStatus:  http.StatusOK,
		},
		{
			name:           "reject without reason",
			action:         "reject",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepositoryTestify)
			mockStorage := new(MockStorageHandler)
			mockDriverService := new(MockDriverService)
			handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

			adminID := uuid.New()
			driverID := uuid.New()
			docType := createTestDocumentTypeHandler()
			doc := createTestDriverDocument(driverID, docType)
			doc.Status = StatusPending

			reqBody := ReviewDocumentRequest{
				Action:          tt.action,
				RejectionReason: tt.rejectionReason,
			}

			mockRepo.On("GetDocument", mock.Anything, doc.ID).Return(doc, nil)
			if tt.expectedStatus == http.StatusOK {
				mockRepo.On("UpdateDocumentStatus", mock.Anything, doc.ID, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockRepo.On("CreateHistory", mock.Anything, mock.AnythingOfType("*documents.DocumentVerificationHistory")).Return(nil)
			}

			c, w := setupTestContext("POST", "/api/v1/admin/documents/"+doc.ID.String()+"/review", reqBody)
			c.Params = gin.Params{{Key: "id", Value: doc.ID.String()}}
			setUserContext(c, adminID, models.RoleAdmin)

			handler.ReviewDocument(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestHandler_GetExpiringDocuments_InvalidDaysParameter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()

	// Should default to 30 when days is invalid
	mockRepo.On("GetExpiringDocuments", mock.Anything, 30).Return([]*ExpiringDocument{}, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/documents/expiring?days=invalid", nil)
	c.Request.URL.RawQuery = "days=invalid"
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetExpiringDocuments(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetPendingReviews_NegativeOffset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()

	// Should handle invalid offset parameter
	mockRepo.On("GetPendingReviews", mock.Anything, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return([]*PendingReviewDocument{}, 0, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/documents/pending?offset=-10", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetPendingReviews(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetPendingReviews_ExcessiveLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	adminID := uuid.New()

	// Should handle excessive limit parameter
	mockRepo.On("GetPendingReviews", mock.Anything, mock.AnythingOfType("int"), mock.AnythingOfType("int")).Return([]*PendingReviewDocument{}, 0, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/documents/pending?limit=1000", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetPendingReviews(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetDocument_EmptyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/documents/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	setUserContext(c, userID, models.RoleDriver)

	handler.GetDocument(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_GetDocumentTypes_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	docTypes := []*DocumentType{createTestDocumentTypeHandler()}

	mockRepo.On("GetDocumentTypes", mock.Anything).Return(docTypes, nil)

	c, w := setupTestContext("GET", "/api/v1/documents/types", nil)

	handler.GetDocumentTypes(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	// Verify response structure
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["document_types"])
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepositoryTestify)
	mockStorage := new(MockStorageHandler)
	mockDriverService := new(MockDriverService)
	handler := createTestHandler(mockRepo, mockStorage, mockDriverService)

	mockRepo.On("GetDocumentTypes", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/documents/types", nil)

	handler.GetDocumentTypes(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)

	// Verify error response structure
	assert.False(t, response["success"].(bool))
	assert.NotNil(t, response["error"])

	errorInfo := response["error"].(map[string]interface{})
	assert.NotNil(t, errorInfo["code"])
	assert.NotNil(t, errorInfo["message"])
}
