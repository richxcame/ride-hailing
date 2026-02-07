package recording

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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

// MockRepository implements RepositoryInterface for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateRecording(ctx context.Context, recording *RideRecording) error {
	args := m.Called(ctx, recording)
	return args.Error(0)
}

func (m *MockRepository) GetRecording(ctx context.Context, recordingID uuid.UUID) (*RideRecording, error) {
	args := m.Called(ctx, recordingID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RideRecording), args.Error(1)
}

func (m *MockRepository) GetRecordingsByRide(ctx context.Context, rideID uuid.UUID) ([]*RideRecording, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RideRecording), args.Error(1)
}

func (m *MockRepository) GetActiveRecordingForRide(ctx context.Context, rideID, userID uuid.UUID) (*RideRecording, error) {
	args := m.Called(ctx, rideID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RideRecording), args.Error(1)
}

func (m *MockRepository) UpdateRecordingStatus(ctx context.Context, recordingID uuid.UUID, status RecordingStatus) error {
	args := m.Called(ctx, recordingID, status)
	return args.Error(0)
}

func (m *MockRepository) UpdateRecordingStopped(ctx context.Context, recordingID uuid.UUID, endedAt time.Time) error {
	args := m.Called(ctx, recordingID, endedAt)
	return args.Error(0)
}

func (m *MockRepository) UpdateRecordingUpload(ctx context.Context, recordingID uuid.UUID, uploadID string, chunksReceived, totalChunks int) error {
	args := m.Called(ctx, recordingID, uploadID, chunksReceived, totalChunks)
	return args.Error(0)
}

func (m *MockRepository) UpdateRecordingCompleted(ctx context.Context, recordingID uuid.UUID, fileURL string, fileSize int64, durationSeconds int) error {
	args := m.Called(ctx, recordingID, fileURL, fileSize, durationSeconds)
	return args.Error(0)
}

func (m *MockRepository) UpdateRecordingProcessed(ctx context.Context, recordingID uuid.UUID, thumbnailURL *string) error {
	args := m.Called(ctx, recordingID, thumbnailURL)
	return args.Error(0)
}

func (m *MockRepository) UpdateRetentionPolicy(ctx context.Context, recordingID uuid.UUID, policy RetentionPolicy, newExpiry time.Time) error {
	args := m.Called(ctx, recordingID, policy, newExpiry)
	return args.Error(0)
}

func (m *MockRepository) GetExpiredRecordings(ctx context.Context, limit int) ([]*RideRecording, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RideRecording), args.Error(1)
}

func (m *MockRepository) MarkRecordingDeleted(ctx context.Context, recordingID uuid.UUID) error {
	args := m.Called(ctx, recordingID)
	return args.Error(0)
}

func (m *MockRepository) CreateConsent(ctx context.Context, consent *RecordingConsent) error {
	args := m.Called(ctx, consent)
	return args.Error(0)
}

func (m *MockRepository) GetConsent(ctx context.Context, rideID, userID uuid.UUID) (*RecordingConsent, error) {
	args := m.Called(ctx, rideID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RecordingConsent), args.Error(1)
}

func (m *MockRepository) CheckAllConsented(ctx context.Context, rideID uuid.UUID) (bool, error) {
	args := m.Called(ctx, rideID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) LogAccess(ctx context.Context, log *RecordingAccessLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockRepository) GetAccessLogs(ctx context.Context, recordingID uuid.UUID) ([]*RecordingAccessLog, error) {
	args := m.Called(ctx, recordingID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*RecordingAccessLog), args.Error(1)
}

func (m *MockRepository) GetSettings(ctx context.Context, userID uuid.UUID) (*RecordingSettings, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RecordingSettings), args.Error(1)
}

func (m *MockRepository) UpsertSettings(ctx context.Context, settings *RecordingSettings) error {
	args := m.Called(ctx, settings)
	return args.Error(0)
}

func (m *MockRepository) GetRecordingStats(ctx context.Context) (*RecordingStatsResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RecordingStatsResponse), args.Error(1)
}

// MockStorageClient implements storage.Storage for testing
type MockStorageClient struct {
	mock.Mock
}

func (m *MockStorageClient) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*storage.UploadResult, error) {
	args := m.Called(ctx, key, reader, size, contentType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.UploadResult), args.Error(1)
}

func (m *MockStorageClient) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockStorageClient) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockStorageClient) GetURL(key string) string {
	args := m.Called(key)
	return args.String(0)
}

func (m *MockStorageClient) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockStorageClient) GetPresignedUploadURL(ctx context.Context, key, contentType string, expiry time.Duration) (*storage.PresignedURLResult, error) {
	args := m.Called(ctx, key, contentType, expiry)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.PresignedURLResult), args.Error(1)
}

func (m *MockStorageClient) GetPresignedDownloadURL(ctx context.Context, key string, expiry time.Duration) (*storage.PresignedURLResult, error) {
	args := m.Called(ctx, key, expiry)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.PresignedURLResult), args.Error(1)
}

func (m *MockStorageClient) Copy(ctx context.Context, sourceKey, destKey string) error {
	args := m.Called(ctx, sourceKey, destKey)
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
	c.Set("role", role)
	c.Set("user_email", "test@example.com")
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

func createTestHandler(mockRepo *MockRepository, mockStorage *MockStorageClient) *Handler {
	cfg := Config{
		BucketName:  "test-bucket",
		MaxDuration: 7200,
		MaxFileSize: 500 * 1024 * 1024,
	}
	service := NewService(mockRepo, mockStorage, cfg)
	return NewHandler(service)
}

func createTestRecording(userID, rideID uuid.UUID) *RideRecording {
	fileURL := "https://storage.example.com/recordings/test.m4a"
	return &RideRecording{
		ID:              uuid.New(),
		RideID:          rideID,
		UserID:          userID,
		UserType:        "driver",
		RecordingType:   RecordingTypeAudio,
		Status:          RecordingStatusRecording,
		RetentionPolicy: RetentionStandard,
		Format:          "m4a",
		Quality:         "medium",
		Encrypted:       true,
		StartedAt:       time.Now().Add(-10 * time.Minute),
		FileURL:         &fileURL,
		ExpiresAt:       time.Now().AddDate(0, 0, 7),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func createCompletedRecording(userID, rideID uuid.UUID) *RideRecording {
	rec := createTestRecording(userID, rideID)
	rec.Status = RecordingStatusCompleted
	fileSize := int64(1024 * 1024)
	duration := 600
	rec.FileSize = &fileSize
	rec.DurationSeconds = &duration
	return rec
}

func createTestSettings(userID uuid.UUID) *RecordingSettings {
	return &RecordingSettings{
		UserID:              userID,
		RecordingEnabled:    true,
		DefaultType:         RecordingTypeAudio,
		DefaultQuality:      "medium",
		AutoRecordNightRides: true,
		AutoRecordSOSRides:  true,
		NotifyOnRecording:   true,
		AllowDriverRecording: true,
		AllowRiderRecording:  true,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
}

// ============================================================================
// StartRecording Handler Tests
// ============================================================================

func TestHandler_StartRecording_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	rideID := uuid.New()

	reqBody := StartRecordingRequest{
		RideID:        rideID,
		RecordingType: RecordingTypeAudio,
		Quality:       "medium",
	}

	mockRepo.On("GetActiveRecordingForRide", mock.Anything, rideID, userID).Return(nil, nil)
	mockRepo.On("CreateRecording", mock.Anything, mock.AnythingOfType("*recording.RideRecording")).Return(nil)
	mockRepo.On("UpdateRecordingStatus", mock.Anything, mock.AnythingOfType("uuid.UUID"), RecordingStatusRecording).Return(nil)
	mockStorage.On("GetPresignedUploadURL", mock.Anything, mock.AnythingOfType("string"), "audio/m4a", time.Hour).Return(&storage.PresignedURLResult{
		URL:       "https://storage.example.com/upload?signature=abc",
		Method:    "PUT",
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil)

	c, w := setupTestContext("POST", "/api/v1/recordings/start", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.StartRecording(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestHandler_StartRecording_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	rideID := uuid.New()
	reqBody := StartRecordingRequest{
		RideID:        rideID,
		RecordingType: RecordingTypeAudio,
	}

	c, w := setupTestContext("POST", "/api/v1/recordings/start", reqBody)
	// Don't set user context

	handler.StartRecording(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_StartRecording_InvalidRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/recordings/start", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/recordings/start", bytes.NewReader([]byte("invalid")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleDriver)

	handler.StartRecording(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_StartRecording_AlreadyActive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	rideID := uuid.New()
	existingRecording := createTestRecording(userID, rideID)

	reqBody := StartRecordingRequest{
		RideID:        rideID,
		RecordingType: RecordingTypeAudio,
	}

	mockRepo.On("GetActiveRecordingForRide", mock.Anything, rideID, userID).Return(existingRecording, nil)

	c, w := setupTestContext("POST", "/api/v1/recordings/start", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.StartRecording(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_StartRecording_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	rideID := uuid.New()

	reqBody := StartRecordingRequest{
		RideID:        rideID,
		RecordingType: RecordingTypeAudio,
	}

	mockRepo.On("GetActiveRecordingForRide", mock.Anything, rideID, userID).Return(nil, nil)
	mockRepo.On("CreateRecording", mock.Anything, mock.AnythingOfType("*recording.RideRecording")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/recordings/start", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.StartRecording(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_StartRecording_VideoType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	rideID := uuid.New()

	reqBody := StartRecordingRequest{
		RideID:        rideID,
		RecordingType: RecordingTypeVideo,
		Quality:       "high",
	}

	mockRepo.On("GetActiveRecordingForRide", mock.Anything, rideID, userID).Return(nil, nil)
	mockRepo.On("CreateRecording", mock.Anything, mock.AnythingOfType("*recording.RideRecording")).Return(nil)
	mockRepo.On("UpdateRecordingStatus", mock.Anything, mock.AnythingOfType("uuid.UUID"), RecordingStatusRecording).Return(nil)
	mockStorage.On("GetPresignedUploadURL", mock.Anything, mock.AnythingOfType("string"), "video/mp4", time.Hour).Return(&storage.PresignedURLResult{
		URL:       "https://storage.example.com/upload?signature=abc",
		Method:    "PUT",
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil)

	c, w := setupTestContext("POST", "/api/v1/recordings/start", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.StartRecording(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================================
// StopRecording Handler Tests
// ============================================================================

func TestHandler_StopRecording_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	rideID := uuid.New()
	recording := createTestRecording(userID, rideID)

	reqBody := StopRecordingRequest{
		RecordingID: recording.ID,
	}

	mockRepo.On("GetRecording", mock.Anything, recording.ID).Return(recording, nil)
	mockRepo.On("UpdateRecordingStopped", mock.Anything, recording.ID, mock.AnythingOfType("time.Time")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/recordings/stop", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.StopRecording(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockRepo.AssertExpectations(t)
}

func TestHandler_StopRecording_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	reqBody := StopRecordingRequest{
		RecordingID: uuid.New(),
	}

	c, w := setupTestContext("POST", "/api/v1/recordings/stop", reqBody)
	// Don't set user context

	handler.StopRecording(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_StopRecording_InvalidRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/recordings/stop", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/recordings/stop", bytes.NewReader([]byte("invalid")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleDriver)

	handler.StopRecording(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_StopRecording_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	recordingID := uuid.New()

	reqBody := StopRecordingRequest{
		RecordingID: recordingID,
	}

	mockRepo.On("GetRecording", mock.Anything, recordingID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("POST", "/api/v1/recordings/stop", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.StopRecording(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_StopRecording_NotOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	otherUserID := uuid.New()
	rideID := uuid.New()
	recording := createTestRecording(otherUserID, rideID)

	reqBody := StopRecordingRequest{
		RecordingID: recording.ID,
	}

	mockRepo.On("GetRecording", mock.Anything, recording.ID).Return(recording, nil)

	c, w := setupTestContext("POST", "/api/v1/recordings/stop", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.StopRecording(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_StopRecording_NotActive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	rideID := uuid.New()
	recording := createCompletedRecording(userID, rideID)

	reqBody := StopRecordingRequest{
		RecordingID: recording.ID,
	}

	mockRepo.On("GetRecording", mock.Anything, recording.ID).Return(recording, nil)

	c, w := setupTestContext("POST", "/api/v1/recordings/stop", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.StopRecording(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// CompleteUpload Handler Tests
// ============================================================================

func TestHandler_CompleteUpload_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	rideID := uuid.New()
	recording := createTestRecording(userID, rideID)
	recording.Status = RecordingStatusStopped

	reqBody := CompleteUploadRequest{
		RecordingID:    recording.ID,
		TotalSize:      1024 * 1024,
		DurationSeconds: 600,
	}

	mockRepo.On("GetRecording", mock.Anything, recording.ID).Return(recording, nil)
	mockStorage.On("GetURL", mock.AnythingOfType("string")).Return("https://storage.example.com/recordings/test.m4a")
	mockRepo.On("UpdateRecordingCompleted", mock.Anything, recording.ID, mock.AnythingOfType("string"), int64(1024*1024), 600).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/recordings/complete", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.CompleteUpload(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_CompleteUpload_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	reqBody := CompleteUploadRequest{
		RecordingID:    uuid.New(),
		TotalSize:      1024 * 1024,
		DurationSeconds: 600,
	}

	c, w := setupTestContext("POST", "/api/v1/recordings/complete", reqBody)
	// Don't set user context

	handler.CompleteUpload(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CompleteUpload_InvalidRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/recordings/complete", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/recordings/complete", bytes.NewReader([]byte("invalid")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleDriver)

	handler.CompleteUpload(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CompleteUpload_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	recordingID := uuid.New()

	reqBody := CompleteUploadRequest{
		RecordingID:    recordingID,
		TotalSize:      1024 * 1024,
		DurationSeconds: 600,
	}

	mockRepo.On("GetRecording", mock.Anything, recordingID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("POST", "/api/v1/recordings/complete", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.CompleteUpload(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_CompleteUpload_FileTooLarge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	rideID := uuid.New()
	recording := createTestRecording(userID, rideID)
	recording.Status = RecordingStatusStopped

	reqBody := CompleteUploadRequest{
		RecordingID:    recording.ID,
		TotalSize:      1024 * 1024 * 1024, // 1GB - exceeds max
		DurationSeconds: 600,
	}

	mockRepo.On("GetRecording", mock.Anything, recording.ID).Return(recording, nil)

	c, w := setupTestContext("POST", "/api/v1/recordings/complete", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.CompleteUpload(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// GetRecording Handler Tests
// ============================================================================

func TestHandler_GetRecording_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	rideID := uuid.New()
	recording := createCompletedRecording(userID, rideID)

	mockRepo.On("GetRecording", mock.Anything, recording.ID).Return(recording, nil)
	mockStorage.On("GetPresignedDownloadURL", mock.Anything, mock.AnythingOfType("string"), time.Hour).Return(&storage.PresignedURLResult{
		URL:       "https://storage.example.com/download?signature=abc",
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil)
	mockRepo.On("LogAccess", mock.Anything, mock.AnythingOfType("*recording.RecordingAccessLog")).Return(nil)

	c, w := setupTestContext("GET", "/api/v1/recordings/"+recording.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: recording.ID.String()}}
	setUserContext(c, userID, models.RoleDriver)

	handler.GetRecording(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetRecording_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	recordingID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/recordings/"+recordingID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: recordingID.String()}}
	// Don't set user context

	handler.GetRecording(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetRecording_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/recordings/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID, models.RoleDriver)

	handler.GetRecording(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetRecording_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	recordingID := uuid.New()

	mockRepo.On("GetRecording", mock.Anything, recordingID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/recordings/"+recordingID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: recordingID.String()}}
	setUserContext(c, userID, models.RoleDriver)

	handler.GetRecording(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetRecording_NotOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	otherUserID := uuid.New()
	rideID := uuid.New()
	recording := createCompletedRecording(otherUserID, rideID)

	mockRepo.On("GetRecording", mock.Anything, recording.ID).Return(recording, nil)

	c, w := setupTestContext("GET", "/api/v1/recordings/"+recording.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: recording.ID.String()}}
	setUserContext(c, userID, models.RoleDriver)

	handler.GetRecording(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ============================================================================
// GetRecordingsForRide Handler Tests
// ============================================================================

func TestHandler_GetRecordingsForRide_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	rideID := uuid.New()
	recordings := []*RideRecording{createCompletedRecording(userID, rideID)}

	mockRepo.On("GetRecordingsByRide", mock.Anything, rideID).Return(recordings, nil)

	c, w := setupTestContext("GET", "/api/v1/recordings/ride/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "ride_id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleDriver)

	handler.GetRecordingsForRide(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetRecordingsForRide_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	rideID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/recordings/ride/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "ride_id", Value: rideID.String()}}
	// Don't set user context

	handler.GetRecordingsForRide(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetRecordingsForRide_InvalidRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/recordings/ride/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "ride_id", Value: "invalid-uuid"}}
	setUserContext(c, userID, models.RoleDriver)

	handler.GetRecordingsForRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetRecordingsForRide_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	rideID := uuid.New()

	mockRepo.On("GetRecordingsByRide", mock.Anything, rideID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/recordings/ride/"+rideID.String(), nil)
	c.Params = gin.Params{{Key: "ride_id", Value: rideID.String()}}
	setUserContext(c, userID, models.RoleDriver)

	handler.GetRecordingsForRide(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// DeleteRecording Handler Tests
// ============================================================================

func TestHandler_DeleteRecording_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	rideID := uuid.New()
	recording := createCompletedRecording(userID, rideID)

	mockRepo.On("GetRecording", mock.Anything, recording.ID).Return(recording, nil)
	mockStorage.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(nil)
	mockRepo.On("MarkRecordingDeleted", mock.Anything, recording.ID).Return(nil)

	c, w := setupTestContext("DELETE", "/api/v1/recordings/"+recording.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: recording.ID.String()}}
	setUserContext(c, userID, models.RoleDriver)

	handler.DeleteRecording(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_DeleteRecording_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	recordingID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/recordings/"+recordingID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: recordingID.String()}}
	// Don't set user context

	handler.DeleteRecording(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_DeleteRecording_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/recordings/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID, models.RoleDriver)

	handler.DeleteRecording(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DeleteRecording_NotOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	otherUserID := uuid.New()
	rideID := uuid.New()
	recording := createCompletedRecording(otherUserID, rideID)

	mockRepo.On("GetRecording", mock.Anything, recording.ID).Return(recording, nil)

	c, w := setupTestContext("DELETE", "/api/v1/recordings/"+recording.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: recording.ID.String()}}
	setUserContext(c, userID, models.RoleDriver)

	handler.DeleteRecording(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ============================================================================
// RecordConsent Handler Tests
// ============================================================================

func TestHandler_RecordConsent_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	rideID := uuid.New()

	reqBody := RecordingConsentRequest{
		RideID:    rideID,
		Consented: true,
	}

	mockRepo.On("CreateConsent", mock.Anything, mock.AnythingOfType("*recording.RecordingConsent")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/recordings/consent", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.RecordConsent(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_RecordConsent_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	reqBody := RecordingConsentRequest{
		RideID:    uuid.New(),
		Consented: true,
	}

	c, w := setupTestContext("POST", "/api/v1/recordings/consent", reqBody)
	// Don't set user context

	handler.RecordConsent(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_RecordConsent_InvalidRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/recordings/consent", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/recordings/consent", bytes.NewReader([]byte("invalid")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleRider)

	handler.RecordConsent(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RecordConsent_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	rideID := uuid.New()

	reqBody := RecordingConsentRequest{
		RideID:    rideID,
		Consented: true,
	}

	mockRepo.On("CreateConsent", mock.Anything, mock.AnythingOfType("*recording.RecordingConsent")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/recordings/consent", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.RecordConsent(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// GetSettings Handler Tests
// ============================================================================

func TestHandler_GetSettings_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	settings := createTestSettings(userID)

	mockRepo.On("GetSettings", mock.Anything, userID).Return(settings, nil)

	c, w := setupTestContext("GET", "/api/v1/recordings/settings", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetSettings(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetSettings_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	c, w := setupTestContext("GET", "/api/v1/recordings/settings", nil)
	// Don't set user context

	handler.GetSettings(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetSettings_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()

	mockRepo.On("GetSettings", mock.Anything, userID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/recordings/settings", nil)
	setUserContext(c, userID, models.RoleDriver)

	handler.GetSettings(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// UpdateSettings Handler Tests
// ============================================================================

func TestHandler_UpdateSettings_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()

	reqBody := RecordingSettings{
		RecordingEnabled:     true,
		DefaultType:          RecordingTypeAudio,
		DefaultQuality:       "high",
		AutoRecordNightRides: true,
		AutoRecordSOSRides:   true,
	}

	mockRepo.On("UpsertSettings", mock.Anything, mock.AnythingOfType("*recording.RecordingSettings")).Return(nil)

	c, w := setupTestContext("PUT", "/api/v1/recordings/settings", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.UpdateSettings(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_UpdateSettings_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	reqBody := RecordingSettings{
		RecordingEnabled: true,
	}

	c, w := setupTestContext("PUT", "/api/v1/recordings/settings", reqBody)
	// Don't set user context

	handler.UpdateSettings(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_UpdateSettings_InvalidRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()

	c, w := setupTestContext("PUT", "/api/v1/recordings/settings", nil)
	c.Request = httptest.NewRequest("PUT", "/api/v1/recordings/settings", bytes.NewReader([]byte("invalid")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleDriver)

	handler.UpdateSettings(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateSettings_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()

	reqBody := RecordingSettings{
		RecordingEnabled: true,
	}

	mockRepo.On("UpsertSettings", mock.Anything, mock.AnythingOfType("*recording.RecordingSettings")).Return(errors.New("database error"))

	c, w := setupTestContext("PUT", "/api/v1/recordings/settings", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.UpdateSettings(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// AdminGetRecording Handler Tests
// ============================================================================

func TestHandler_AdminGetRecording_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	adminID := uuid.New()
	userID := uuid.New()
	rideID := uuid.New()
	recording := createCompletedRecording(userID, rideID)

	mockRepo.On("GetRecording", mock.Anything, recording.ID).Return(recording, nil)
	mockStorage.On("GetPresignedDownloadURL", mock.Anything, mock.AnythingOfType("string"), time.Hour).Return(&storage.PresignedURLResult{
		URL:       "https://storage.example.com/download?signature=abc",
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil)
	mockRepo.On("LogAccess", mock.Anything, mock.AnythingOfType("*recording.RecordingAccessLog")).Return(nil)

	c, w := setupTestContext("GET", "/api/v1/admin/recordings/"+recording.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: recording.ID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetRecording(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_AdminGetRecording_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	recordingID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/admin/recordings/"+recordingID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: recordingID.String()}}
	// Don't set user context

	handler.AdminGetRecording(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_AdminGetRecording_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	adminID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/admin/recordings/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetRecording(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminGetRecording_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	adminID := uuid.New()
	recordingID := uuid.New()

	mockRepo.On("GetRecording", mock.Anything, recordingID).Return(nil, errors.New("not found"))

	c, w := setupTestContext("GET", "/api/v1/admin/recordings/"+recordingID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: recordingID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetRecording(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ============================================================================
// GetRecordingStats Handler Tests
// ============================================================================

func TestHandler_GetRecordingStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	stats := &RecordingStatsResponse{
		TotalRecordings:    1000,
		TotalDurationHours: 500.5,
		TotalStorageGB:     25.5,
		ActiveRecordings:   10,
		PendingUploads:     5,
		RecordingsByType: map[string]int64{
			"audio": 800,
			"video": 200,
		},
		RecordingsByStatus: map[string]int64{
			"completed": 900,
			"recording": 10,
			"stopped":   90,
		},
	}

	mockRepo.On("GetRecordingStats", mock.Anything).Return(stats, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/recordings/stats", nil)

	handler.GetRecordingStats(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_GetRecordingStats_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	mockRepo.On("GetRecordingStats", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/recordings/stats", nil)

	handler.GetRecordingStats(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// ExtendRetention Handler Tests
// ============================================================================

func TestHandler_ExtendRetention_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	recordingID := uuid.New()

	reqBody := map[string]interface{}{
		"policy": "extended",
	}

	mockRepo.On("UpdateRetentionPolicy", mock.Anything, recordingID, RetentionExtended, mock.AnythingOfType("time.Time")).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/recordings/"+recordingID.String()+"/extend", reqBody)
	c.Params = gin.Params{{Key: "id", Value: recordingID.String()}}

	handler.ExtendRetention(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
}

func TestHandler_ExtendRetention_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	reqBody := map[string]interface{}{
		"policy": "extended",
	}

	c, w := setupTestContext("POST", "/api/v1/admin/recordings/invalid-uuid/extend", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

	handler.ExtendRetention(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ExtendRetention_InvalidRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	recordingID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/admin/recordings/"+recordingID.String()+"/extend", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/recordings/"+recordingID.String()+"/extend", bytes.NewReader([]byte("invalid")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: recordingID.String()}}

	handler.ExtendRetention(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ExtendRetention_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	recordingID := uuid.New()

	reqBody := map[string]interface{}{
		"policy": "extended",
	}

	mockRepo.On("UpdateRetentionPolicy", mock.Anything, recordingID, RetentionExtended, mock.AnythingOfType("time.Time")).Return(errors.New("database error"))

	c, w := setupTestContext("POST", "/api/v1/admin/recordings/"+recordingID.String()+"/extend", reqBody)
	c.Params = gin.Params{{Key: "id", Value: recordingID.String()}}

	handler.ExtendRetention(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_StartRecording_ValidationCases(t *testing.T) {
	tests := []struct {
		name           string
		reqBody        interface{}
		setUserContext bool
		expectedStatus int
	}{
		{
			name: "valid audio recording",
			reqBody: StartRecordingRequest{
				RideID:        uuid.New(),
				RecordingType: RecordingTypeAudio,
			},
			setUserContext: true,
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid video recording",
			reqBody: StartRecordingRequest{
				RideID:        uuid.New(),
				RecordingType: RecordingTypeVideo,
			},
			setUserContext: true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "empty body",
			reqBody:        nil,
			setUserContext: true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing ride_id",
			reqBody: map[string]interface{}{
				"recording_type": "audio",
			},
			setUserContext: true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "unauthorized",
			reqBody: StartRecordingRequest{
				RideID:        uuid.New(),
				RecordingType: RecordingTypeAudio,
			},
			setUserContext: false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockRepo := new(MockRepository)
			mockStorage := new(MockStorageClient)
			handler := createTestHandler(mockRepo, mockStorage)

			if tt.expectedStatus == http.StatusOK {
				mockRepo.On("GetActiveRecordingForRide", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID")).Return(nil, nil)
				mockRepo.On("CreateRecording", mock.Anything, mock.AnythingOfType("*recording.RideRecording")).Return(nil)
				mockRepo.On("UpdateRecordingStatus", mock.Anything, mock.AnythingOfType("uuid.UUID"), RecordingStatusRecording).Return(nil)
				mockStorage.On("GetPresignedUploadURL", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), time.Hour).Return(&storage.PresignedURLResult{
					URL:       "https://storage.example.com/upload?signature=abc",
					Method:    "PUT",
					ExpiresAt: time.Now().Add(time.Hour),
				}, nil)
			}

			c, w := setupTestContext("POST", "/api/v1/recordings/start", tt.reqBody)
			if tt.reqBody == nil {
				c.Request = httptest.NewRequest("POST", "/api/v1/recordings/start", nil)
				c.Request.Header.Set("Content-Type", "application/json")
			}
			if tt.setUserContext {
				setUserContext(c, uuid.New(), models.RoleDriver)
			}

			handler.StartRecording(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_StartRecording_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	rideID := uuid.New()

	reqBody := StartRecordingRequest{
		RideID:        rideID,
		RecordingType: RecordingTypeAudio,
	}

	mockRepo.On("GetActiveRecordingForRide", mock.Anything, rideID, userID).Return(nil, nil)
	mockRepo.On("CreateRecording", mock.Anything, mock.AnythingOfType("*recording.RideRecording")).Return(nil)
	mockRepo.On("UpdateRecordingStatus", mock.Anything, mock.AnythingOfType("uuid.UUID"), RecordingStatusRecording).Return(nil)
	mockStorage.On("GetPresignedUploadURL", mock.Anything, mock.AnythingOfType("string"), "audio/m4a", time.Hour).Return(&storage.PresignedURLResult{
		URL:       "https://storage.example.com/upload?signature=abc",
		Method:    "PUT",
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil)

	c, w := setupTestContext("POST", "/api/v1/recordings/start", reqBody)
	setUserContext(c, userID, models.RoleDriver)

	handler.StartRecording(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["recording_id"])
	assert.NotNil(t, data["upload_url"])
	assert.NotNil(t, data["status"])
}

func TestHandler_GetRecording_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()
	rideID := uuid.New()
	recording := createCompletedRecording(userID, rideID)

	mockRepo.On("GetRecording", mock.Anything, recording.ID).Return(recording, nil)
	mockStorage.On("GetPresignedDownloadURL", mock.Anything, mock.AnythingOfType("string"), time.Hour).Return(&storage.PresignedURLResult{
		URL:       "https://storage.example.com/download?signature=abc",
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil)
	mockRepo.On("LogAccess", mock.Anything, mock.AnythingOfType("*recording.RecordingAccessLog")).Return(nil)

	c, w := setupTestContext("GET", "/api/v1/recordings/"+recording.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: recording.ID.String()}}
	setUserContext(c, userID, models.RoleDriver)

	handler.GetRecording(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["recording"])
	assert.NotNil(t, data["access_url"])
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	mockRepo.On("GetRecordingStats", mock.Anything).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/recordings/stats", nil)

	handler.GetRecordingStats(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)

	assert.False(t, response["success"].(bool))
	assert.NotNil(t, response["error"])

	errorInfo := response["error"].(map[string]interface{})
	assert.NotNil(t, errorInfo["code"])
	assert.NotNil(t, errorInfo["message"])
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestHandler_GetRecording_EmptyUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/recordings/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	setUserContext(c, userID, models.RoleDriver)

	handler.GetRecording(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetRecordingsForRide_EmptyRideID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/recordings/ride/", nil)
	c.Params = gin.Params{{Key: "ride_id", Value: ""}}
	setUserContext(c, userID, models.RoleDriver)

	handler.GetRecordingsForRide(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DeleteRecording_EmptyUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	userID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/recordings/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	setUserContext(c, userID, models.RoleDriver)

	handler.DeleteRecording(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminGetRecording_WithReasonParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockRepository)
	mockStorage := new(MockStorageClient)
	handler := createTestHandler(mockRepo, mockStorage)

	adminID := uuid.New()
	userID := uuid.New()
	rideID := uuid.New()
	recording := createCompletedRecording(userID, rideID)

	mockRepo.On("GetRecording", mock.Anything, recording.ID).Return(recording, nil)
	mockStorage.On("GetPresignedDownloadURL", mock.Anything, mock.AnythingOfType("string"), time.Hour).Return(&storage.PresignedURLResult{
		URL:       "https://storage.example.com/download?signature=abc",
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil)
	mockRepo.On("LogAccess", mock.Anything, mock.AnythingOfType("*recording.RecordingAccessLog")).Return(nil)

	c, w := setupTestContext("GET", "/api/v1/admin/recordings/"+recording.ID.String()+"?reason=dispute_investigation", nil)
	c.Params = gin.Params{{Key: "id", Value: recording.ID.String()}}
	c.Request.URL.RawQuery = "reason=dispute_investigation"
	setUserContext(c, adminID, models.RoleAdmin)

	handler.AdminGetRecording(c)

	assert.Equal(t, http.StatusOK, w.Code)
}
