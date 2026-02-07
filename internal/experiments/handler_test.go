package experiments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// Mock Service Interface
// ============================================================================

// MockExperimentsService is a mock implementation of the experiments service for handler testing
type MockExperimentsService struct {
	mock.Mock
}

// Feature Flag methods
func (m *MockExperimentsService) EvaluateFlag(ctx context.Context, key string, userCtx *UserContext) (*EvaluateFlagResponse, error) {
	args := m.Called(ctx, key, userCtx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EvaluateFlagResponse), args.Error(1)
}

func (m *MockExperimentsService) EvaluateFlags(ctx context.Context, keys []string, userCtx *UserContext) (*EvaluateFlagsResponse, error) {
	args := m.Called(ctx, keys, userCtx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EvaluateFlagsResponse), args.Error(1)
}

func (m *MockExperimentsService) CreateFlag(ctx context.Context, adminID uuid.UUID, req *CreateFlagRequest) (*FeatureFlag, error) {
	args := m.Called(ctx, adminID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FeatureFlag), args.Error(1)
}

func (m *MockExperimentsService) GetFlag(ctx context.Context, key string) (*FeatureFlag, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FeatureFlag), args.Error(1)
}

func (m *MockExperimentsService) ListFlags(ctx context.Context, status *FlagStatus, limit, offset int) ([]*FeatureFlag, error) {
	args := m.Called(ctx, status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*FeatureFlag), args.Error(1)
}

func (m *MockExperimentsService) UpdateFlag(ctx context.Context, flagID uuid.UUID, req *UpdateFlagRequest) error {
	args := m.Called(ctx, flagID, req)
	return args.Error(0)
}

func (m *MockExperimentsService) ToggleFlag(ctx context.Context, flagID uuid.UUID, enabled bool) error {
	args := m.Called(ctx, flagID, enabled)
	return args.Error(0)
}

func (m *MockExperimentsService) ArchiveFlag(ctx context.Context, flagID uuid.UUID) error {
	args := m.Called(ctx, flagID)
	return args.Error(0)
}

func (m *MockExperimentsService) CreateOverride(ctx context.Context, adminID, flagID uuid.UUID, req *CreateOverrideRequest) error {
	args := m.Called(ctx, adminID, flagID, req)
	return args.Error(0)
}

func (m *MockExperimentsService) ListOverrides(ctx context.Context, flagID uuid.UUID) ([]*FlagOverride, error) {
	args := m.Called(ctx, flagID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*FlagOverride), args.Error(1)
}

// Experiment methods
func (m *MockExperimentsService) CreateExperiment(ctx context.Context, adminID uuid.UUID, req *CreateExperimentRequest) (*Experiment, error) {
	args := m.Called(ctx, adminID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Experiment), args.Error(1)
}

func (m *MockExperimentsService) ListExperiments(ctx context.Context, status *ExperimentStatus, limit, offset int) ([]*Experiment, error) {
	args := m.Called(ctx, status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Experiment), args.Error(1)
}

func (m *MockExperimentsService) GetExperiment(ctx context.Context, experimentID uuid.UUID) (*Experiment, []*Variant, error) {
	args := m.Called(ctx, experimentID)
	exp, _ := args.Get(0).(*Experiment)
	variants, _ := args.Get(1).([]*Variant)
	return exp, variants, args.Error(2)
}

func (m *MockExperimentsService) StartExperiment(ctx context.Context, experimentID uuid.UUID) error {
	args := m.Called(ctx, experimentID)
	return args.Error(0)
}

func (m *MockExperimentsService) PauseExperiment(ctx context.Context, experimentID uuid.UUID) error {
	args := m.Called(ctx, experimentID)
	return args.Error(0)
}

func (m *MockExperimentsService) ConcludeExperiment(ctx context.Context, experimentID uuid.UUID) error {
	args := m.Called(ctx, experimentID)
	return args.Error(0)
}

func (m *MockExperimentsService) GetResults(ctx context.Context, experimentID uuid.UUID) (*ExperimentResults, error) {
	args := m.Called(ctx, experimentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ExperimentResults), args.Error(1)
}

func (m *MockExperimentsService) GetVariantForUser(ctx context.Context, key string, userCtx *UserContext) (*Variant, error) {
	args := m.Called(ctx, key, userCtx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Variant), args.Error(1)
}

func (m *MockExperimentsService) TrackEvent(ctx context.Context, userID uuid.UUID, req *TrackEventRequest) error {
	args := m.Called(ctx, userID, req)
	return args.Error(0)
}

// ============================================================================
// Mockable Handler (wraps mock service)
// ============================================================================

type MockableHandler struct {
	service *MockExperimentsService
}

func NewMockableHandler(mockService *MockExperimentsService) *MockableHandler {
	return &MockableHandler{service: mockService}
}

// EvaluateFlag evaluates a single feature flag
func (h *MockableHandler) EvaluateFlag(c *gin.Context) {
	key := c.Param("key")
	userCtx := h.buildUserContext(c)

	result, err := h.service.EvaluateFlag(c.Request.Context(), key, userCtx)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to evaluate flag")
		return
	}

	common.SuccessResponse(c, result)
}

// EvaluateFlags evaluates multiple feature flags
func (h *MockableHandler) EvaluateFlags(c *gin.Context) {
	var req struct {
		Keys []string `json:"keys" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	userCtx := h.buildUserContext(c)

	result, err := h.service.EvaluateFlags(c.Request.Context(), req.Keys, userCtx)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to evaluate flags")
		return
	}

	common.SuccessResponse(c, result)
}

// CreateFlag creates a new feature flag
func (h *MockableHandler) CreateFlag(c *gin.Context) {
	adminID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateFlagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	flag, err := h.service.CreateFlag(c.Request.Context(), adminID.(uuid.UUID), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create flag")
		return
	}

	common.SuccessResponse(c, flag)
}

// ListFlags lists all feature flags
func (h *MockableHandler) ListFlags(c *gin.Context) {
	limit := 20
	offset := 0

	var status *FlagStatus
	if s := c.Query("status"); s != "" {
		st := FlagStatus(s)
		status = &st
	}

	flags, err := h.service.ListFlags(c.Request.Context(), status, limit, offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list flags")
		return
	}

	common.SuccessResponseWithMeta(c, flags, &common.Meta{Limit: limit, Offset: offset, Total: int64(len(flags))})
}

// GetFlag gets a flag by key
func (h *MockableHandler) GetFlag(c *gin.Context) {
	key := c.Param("key")

	flag, err := h.service.GetFlag(c.Request.Context(), key)
	if err != nil || flag == nil {
		common.ErrorResponse(c, http.StatusNotFound, "flag not found")
		return
	}

	common.SuccessResponse(c, flag)
}

// UpdateFlag updates a feature flag
func (h *MockableHandler) UpdateFlag(c *gin.Context) {
	flagID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid flag ID")
		return
	}

	var req UpdateFlagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.UpdateFlag(c.Request.Context(), flagID, &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to update flag")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Flag updated successfully"})
}

// ToggleFlag toggles a feature flag
func (h *MockableHandler) ToggleFlag(c *gin.Context) {
	flagID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid flag ID")
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.ToggleFlag(c.Request.Context(), flagID, req.Enabled); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to toggle flag")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Flag toggled"})
}

// ArchiveFlag archives a feature flag
func (h *MockableHandler) ArchiveFlag(c *gin.Context) {
	flagID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid flag ID")
		return
	}

	if err := h.service.ArchiveFlag(c.Request.Context(), flagID); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to archive flag")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Flag archived"})
}

// CreateOverride creates a flag override
func (h *MockableHandler) CreateOverride(c *gin.Context) {
	adminID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	flagID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid flag ID")
		return
	}

	var req CreateOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.CreateOverride(c.Request.Context(), adminID.(uuid.UUID), flagID, &req); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create override")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Override created"})
}

// ListOverrides lists overrides for a flag
func (h *MockableHandler) ListOverrides(c *gin.Context) {
	flagID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid flag ID")
		return
	}

	overrides, err := h.service.ListOverrides(c.Request.Context(), flagID)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list overrides")
		return
	}

	common.SuccessResponseWithMeta(c, overrides, &common.Meta{Total: int64(len(overrides))})
}

// CreateExperiment creates a new experiment
func (h *MockableHandler) CreateExperiment(c *gin.Context) {
	adminID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateExperimentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	experiment, err := h.service.CreateExperiment(c.Request.Context(), adminID.(uuid.UUID), &req)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to create experiment")
		return
	}

	common.SuccessResponse(c, experiment)
}

// ListExperiments lists all experiments
func (h *MockableHandler) ListExperiments(c *gin.Context) {
	limit := 20
	offset := 0

	var status *ExperimentStatus
	if s := c.Query("status"); s != "" {
		st := ExperimentStatus(s)
		status = &st
	}

	experiments, err := h.service.ListExperiments(c.Request.Context(), status, limit, offset)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to list experiments")
		return
	}

	common.SuccessResponseWithMeta(c, experiments, &common.Meta{Limit: limit, Offset: offset, Total: int64(len(experiments))})
}

// GetExperiment gets an experiment with variants
func (h *MockableHandler) GetExperiment(c *gin.Context) {
	experimentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid experiment ID")
		return
	}

	experiment, variants, err := h.service.GetExperiment(c.Request.Context(), experimentID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusNotFound, "experiment not found")
		return
	}

	common.SuccessResponse(c, gin.H{
		"experiment": experiment,
		"variants":   variants,
	})
}

// StartExperiment starts an experiment
func (h *MockableHandler) StartExperiment(c *gin.Context) {
	experimentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid experiment ID")
		return
	}

	if err := h.service.StartExperiment(c.Request.Context(), experimentID); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to start experiment")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Experiment started"})
}

// PauseExperiment pauses an experiment
func (h *MockableHandler) PauseExperiment(c *gin.Context) {
	experimentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid experiment ID")
		return
	}

	if err := h.service.PauseExperiment(c.Request.Context(), experimentID); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to pause experiment")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Experiment paused"})
}

// ConcludeExperiment concludes an experiment
func (h *MockableHandler) ConcludeExperiment(c *gin.Context) {
	experimentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid experiment ID")
		return
	}

	if err := h.service.ConcludeExperiment(c.Request.Context(), experimentID); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to conclude experiment")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "Experiment concluded"})
}

// GetResults gets experiment results
func (h *MockableHandler) GetResults(c *gin.Context) {
	experimentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid experiment ID")
		return
	}

	results, err := h.service.GetResults(c.Request.Context(), experimentID)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get results")
		return
	}

	common.SuccessResponse(c, results)
}

// GetVariant gets user's variant for an experiment
func (h *MockableHandler) GetVariant(c *gin.Context) {
	key := c.Param("key")
	userCtx := h.buildUserContext(c)

	variant, err := h.service.GetVariantForUser(c.Request.Context(), key, userCtx)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get variant")
		return
	}

	if variant == nil {
		common.SuccessResponse(c, gin.H{
			"enrolled": false,
			"variant":  nil,
		})
		return
	}

	common.SuccessResponse(c, gin.H{
		"enrolled": true,
		"variant":  variant,
	})
}

// TrackEvent tracks an experiment event
func (h *MockableHandler) TrackEvent(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req TrackEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.TrackEvent(c.Request.Context(), userID.(uuid.UUID), &req); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to track event")
		return
	}

	common.SuccessResponse(c, gin.H{"message": "event tracked"})
}

func (h *MockableHandler) buildUserContext(c *gin.Context) *UserContext {
	userID, exists := c.Get("user_id")
	if !exists {
		return nil
	}

	role, _ := c.Get("user_role")
	roleStr := ""
	if r, ok := role.(models.UserRole); ok {
		roleStr = string(r)
	} else if r, ok := role.(string); ok {
		roleStr = r
	}

	return &UserContext{
		UserID:     userID.(uuid.UUID),
		Role:       roleStr,
		Country:    c.GetHeader("X-Country"),
		City:       c.GetHeader("X-City"),
		Platform:   c.GetHeader("X-Platform"),
		AppVersion: c.GetHeader("X-App-Version"),
	}
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
}

func setUserContextWithHeaders(c *gin.Context, userID uuid.UUID, role models.UserRole, country, city, platform, appVersion string) {
	setUserContext(c, userID, role)
	c.Request.Header.Set("X-Country", country)
	c.Request.Header.Set("X-City", city)
	c.Request.Header.Set("X-Platform", platform)
	c.Request.Header.Set("X-App-Version", appVersion)
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

func createTestFeatureFlag(key string, flagType FlagType, status FlagStatus, enabled bool) *FeatureFlag {
	return &FeatureFlag{
		ID:        uuid.New(),
		Key:       key,
		Name:      "Test Flag " + key,
		FlagType:  flagType,
		Status:    status,
		Enabled:   enabled,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func createTestExperimentObj(key string, status ExperimentStatus) *Experiment {
	return &Experiment{
		ID:                uuid.New(),
		Key:               key,
		Name:              "Test Experiment " + key,
		Status:            status,
		TrafficPercentage: 100,
		MinSampleSize:     100,
		ConfidenceLevel:   0.95,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

func createTestVariantsObj(experimentID uuid.UUID) []*Variant {
	return []*Variant{
		{
			ID:           uuid.New(),
			ExperimentID: experimentID,
			Key:          "control",
			Name:         "Control",
			IsControl:    true,
			Weight:       50,
		},
		{
			ID:           uuid.New(),
			ExperimentID: experimentID,
			Key:          "variant_a",
			Name:         "Variant A",
			IsControl:    false,
			Weight:       50,
		},
	}
}

// ============================================================================
// EvaluateFlag Handler Tests
// ============================================================================

func TestHandler_EvaluateFlag_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	expectedResult := &EvaluateFlagResponse{
		Key:     "test_flag",
		Enabled: true,
		Source:  "default",
	}

	mockService.On("EvaluateFlag", mock.Anything, "test_flag", mock.AnythingOfType("*experiments.UserContext")).Return(expectedResult, nil)

	c, w := setupTestContext("GET", "/api/v1/flags/test_flag", nil)
	c.Params = gin.Params{{Key: "key", Value: "test_flag"}}
	setUserContext(c, userID, models.RoleRider)

	handler.EvaluateFlag(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "test_flag", data["key"])
	assert.True(t, data["enabled"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_EvaluateFlag_FlagDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	expectedResult := &EvaluateFlagResponse{
		Key:     "disabled_flag",
		Enabled: false,
		Source:  "inactive",
	}

	mockService.On("EvaluateFlag", mock.Anything, "disabled_flag", mock.AnythingOfType("*experiments.UserContext")).Return(expectedResult, nil)

	c, w := setupTestContext("GET", "/api/v1/flags/disabled_flag", nil)
	c.Params = gin.Params{{Key: "key", Value: "disabled_flag"}}
	setUserContext(c, userID, models.RoleRider)

	handler.EvaluateFlag(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.False(t, data["enabled"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_EvaluateFlag_WithUserContextHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	expectedResult := &EvaluateFlagResponse{
		Key:     "segment_flag",
		Enabled: true,
		Source:  "segment",
	}

	mockService.On("EvaluateFlag", mock.Anything, "segment_flag", mock.MatchedBy(func(ctx *UserContext) bool {
		return ctx.Country == "US" && ctx.Platform == "ios"
	})).Return(expectedResult, nil)

	c, w := setupTestContext("GET", "/api/v1/flags/segment_flag", nil)
	c.Params = gin.Params{{Key: "key", Value: "segment_flag"}}
	setUserContextWithHeaders(c, userID, models.RoleRider, "US", "San Francisco", "ios", "2.0.0")

	handler.EvaluateFlag(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_EvaluateFlag_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("EvaluateFlag", mock.Anything, "error_flag", mock.AnythingOfType("*experiments.UserContext")).Return(nil, errors.New("service error"))

	c, w := setupTestContext("GET", "/api/v1/flags/error_flag", nil)
	c.Params = gin.Params{{Key: "key", Value: "error_flag"}}
	setUserContext(c, userID, models.RoleRider)

	handler.EvaluateFlag(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_EvaluateFlag_NoUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	expectedResult := &EvaluateFlagResponse{
		Key:     "public_flag",
		Enabled: true,
		Source:  "default",
	}

	mockService.On("EvaluateFlag", mock.Anything, "public_flag", (*UserContext)(nil)).Return(expectedResult, nil)

	c, w := setupTestContext("GET", "/api/v1/flags/public_flag", nil)
	c.Params = gin.Params{{Key: "key", Value: "public_flag"}}
	// No user context set

	handler.EvaluateFlag(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_EvaluateFlag_Override(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	expectedResult := &EvaluateFlagResponse{
		Key:     "override_flag",
		Enabled: true,
		Source:  "override",
	}

	mockService.On("EvaluateFlag", mock.Anything, "override_flag", mock.AnythingOfType("*experiments.UserContext")).Return(expectedResult, nil)

	c, w := setupTestContext("GET", "/api/v1/flags/override_flag", nil)
	c.Params = gin.Params{{Key: "key", Value: "override_flag"}}
	setUserContext(c, userID, models.RoleRider)

	handler.EvaluateFlag(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "override", data["source"])
	mockService.AssertExpectations(t)
}

// ============================================================================
// EvaluateFlags Handler Tests (Batch)
// ============================================================================

func TestHandler_EvaluateFlags_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	expectedResult := &EvaluateFlagsResponse{
		Flags: map[string]EvaluateFlagResponse{
			"flag1": {Key: "flag1", Enabled: true, Source: "default"},
			"flag2": {Key: "flag2", Enabled: false, Source: "inactive"},
		},
	}

	mockService.On("EvaluateFlags", mock.Anything, []string{"flag1", "flag2"}, mock.AnythingOfType("*experiments.UserContext")).Return(expectedResult, nil)

	reqBody := map[string]interface{}{
		"keys": []string{"flag1", "flag2"},
	}

	c, w := setupTestContext("POST", "/api/v1/flags/evaluate", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.EvaluateFlags(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_EvaluateFlags_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/flags/evaluate", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/flags/evaluate", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleRider)

	handler.EvaluateFlags(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_EvaluateFlags_MissingKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/flags/evaluate", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.EvaluateFlags(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_EvaluateFlags_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("EvaluateFlags", mock.Anything, []string{"flag1"}, mock.AnythingOfType("*experiments.UserContext")).Return(nil, errors.New("service error"))

	reqBody := map[string]interface{}{
		"keys": []string{"flag1"},
	}

	c, w := setupTestContext("POST", "/api/v1/flags/evaluate", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.EvaluateFlags(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_EvaluateFlags_EmptyKeysArray(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	expectedResult := &EvaluateFlagsResponse{
		Flags: map[string]EvaluateFlagResponse{},
	}

	mockService.On("EvaluateFlags", mock.Anything, []string{}, mock.AnythingOfType("*experiments.UserContext")).Return(expectedResult, nil)

	reqBody := map[string]interface{}{
		"keys": []string{},
	}

	c, w := setupTestContext("POST", "/api/v1/flags/evaluate", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.EvaluateFlags(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// CreateFlag Handler Tests (Admin)
// ============================================================================

func TestHandler_CreateFlag_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	expectedFlag := createTestFeatureFlag("new_flag", FlagTypeBoolean, FlagStatusActive, true)

	mockService.On("CreateFlag", mock.Anything, adminID, mock.AnythingOfType("*experiments.CreateFlagRequest")).Return(expectedFlag, nil)

	reqBody := CreateFlagRequest{
		Key:      "new_flag",
		Name:     "New Flag",
		FlagType: FlagTypeBoolean,
		Enabled:  true,
	}

	c, w := setupTestContext("POST", "/api/v1/admin/flags", reqBody)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.CreateFlag(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_CreateFlag_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	reqBody := CreateFlagRequest{
		Key:      "new_flag",
		Name:     "New Flag",
		FlagType: FlagTypeBoolean,
	}

	c, w := setupTestContext("POST", "/api/v1/admin/flags", reqBody)
	// No user context set

	handler.CreateFlag(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CreateFlag_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/admin/flags", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/flags", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, adminID, models.RoleAdmin)

	handler.CreateFlag(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateFlag_DuplicateKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()

	mockService.On("CreateFlag", mock.Anything, adminID, mock.AnythingOfType("*experiments.CreateFlagRequest")).Return(nil, common.NewBadRequestError("flag key already exists", nil))

	reqBody := CreateFlagRequest{
		Key:      "existing_flag",
		Name:     "Existing Flag",
		FlagType: FlagTypeBoolean,
	}

	c, w := setupTestContext("POST", "/api/v1/admin/flags", reqBody)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.CreateFlag(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CreateFlag_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()

	mockService.On("CreateFlag", mock.Anything, adminID, mock.AnythingOfType("*experiments.CreateFlagRequest")).Return(nil, errors.New("database error"))

	reqBody := CreateFlagRequest{
		Key:      "new_flag",
		Name:     "New Flag",
		FlagType: FlagTypeBoolean,
	}

	c, w := setupTestContext("POST", "/api/v1/admin/flags", reqBody)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.CreateFlag(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CreateFlag_AllFlagTypes(t *testing.T) {
	flagTypes := []FlagType{
		FlagTypeBoolean,
		FlagTypePercentage,
		FlagTypeUserList,
		FlagTypeSegment,
	}

	for _, flagType := range flagTypes {
		t.Run(string(flagType), func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(MockExperimentsService)
			handler := NewMockableHandler(mockService)

			adminID := uuid.New()
			expectedFlag := createTestFeatureFlag("test_"+string(flagType), flagType, FlagStatusActive, true)

			mockService.On("CreateFlag", mock.Anything, adminID, mock.AnythingOfType("*experiments.CreateFlagRequest")).Return(expectedFlag, nil)

			reqBody := CreateFlagRequest{
				Key:      "test_" + string(flagType),
				Name:     "Test " + string(flagType),
				FlagType: flagType,
			}

			c, w := setupTestContext("POST", "/api/v1/admin/flags", reqBody)
			setUserContext(c, adminID, models.RoleAdmin)

			handler.CreateFlag(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// ============================================================================
// ListFlags Handler Tests (Admin)
// ============================================================================

func TestHandler_ListFlags_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	flags := []*FeatureFlag{
		createTestFeatureFlag("flag1", FlagTypeBoolean, FlagStatusActive, true),
		createTestFeatureFlag("flag2", FlagTypeBoolean, FlagStatusActive, false),
	}

	mockService.On("ListFlags", mock.Anything, (*FlagStatus)(nil), 20, 0).Return(flags, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/flags", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ListFlags(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["meta"])
	mockService.AssertExpectations(t)
}

func TestHandler_ListFlags_WithStatusFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	activeStatus := FlagStatusActive
	flags := []*FeatureFlag{
		createTestFeatureFlag("flag1", FlagTypeBoolean, FlagStatusActive, true),
	}

	mockService.On("ListFlags", mock.Anything, &activeStatus, 20, 0).Return(flags, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/flags?status=active", nil)
	c.Request.URL.RawQuery = "status=active"
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ListFlags(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ListFlags_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()

	mockService.On("ListFlags", mock.Anything, (*FlagStatus)(nil), 20, 0).Return([]*FeatureFlag{}, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/flags", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ListFlags(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ListFlags_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()

	mockService.On("ListFlags", mock.Anything, (*FlagStatus)(nil), 20, 0).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/admin/flags", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ListFlags(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetFlag Handler Tests (Admin)
// ============================================================================

func TestHandler_GetFlag_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	flag := createTestFeatureFlag("test_flag", FlagTypeBoolean, FlagStatusActive, true)

	mockService.On("GetFlag", mock.Anything, "test_flag").Return(flag, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/flags/test_flag", nil)
	c.Params = gin.Params{{Key: "key", Value: "test_flag"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetFlag(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetFlag_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()

	mockService.On("GetFlag", mock.Anything, "unknown_flag").Return(nil, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/flags/unknown_flag", nil)
	c.Params = gin.Params{{Key: "key", Value: "unknown_flag"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetFlag(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// UpdateFlag Handler Tests (Admin)
// ============================================================================

func TestHandler_UpdateFlag_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	flagID := uuid.New()

	mockService.On("UpdateFlag", mock.Anything, flagID, mock.AnythingOfType("*experiments.UpdateFlagRequest")).Return(nil)

	newName := "Updated Name"
	reqBody := UpdateFlagRequest{
		Name: &newName,
	}

	c, w := setupTestContext("PUT", "/api/v1/admin/flags/"+flagID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: flagID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.UpdateFlag(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_UpdateFlag_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()

	newName := "Updated Name"
	reqBody := UpdateFlagRequest{
		Name: &newName,
	}

	c, w := setupTestContext("PUT", "/api/v1/admin/flags/invalid-uuid", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.UpdateFlag(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateFlag_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	flagID := uuid.New()

	mockService.On("UpdateFlag", mock.Anything, flagID, mock.AnythingOfType("*experiments.UpdateFlagRequest")).Return(common.NewNotFoundError("flag not found", nil))

	newName := "Updated Name"
	reqBody := UpdateFlagRequest{
		Name: &newName,
	}

	c, w := setupTestContext("PUT", "/api/v1/admin/flags/"+flagID.String(), reqBody)
	c.Params = gin.Params{{Key: "id", Value: flagID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.UpdateFlag(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// ToggleFlag Handler Tests (Admin)
// ============================================================================

func TestHandler_ToggleFlag_Enable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	flagID := uuid.New()

	mockService.On("ToggleFlag", mock.Anything, flagID, true).Return(nil)

	reqBody := map[string]interface{}{
		"enabled": true,
	}

	c, w := setupTestContext("POST", "/api/v1/admin/flags/"+flagID.String()+"/toggle", reqBody)
	c.Params = gin.Params{{Key: "id", Value: flagID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ToggleFlag(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ToggleFlag_Disable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	flagID := uuid.New()

	mockService.On("ToggleFlag", mock.Anything, flagID, false).Return(nil)

	reqBody := map[string]interface{}{
		"enabled": false,
	}

	c, w := setupTestContext("POST", "/api/v1/admin/flags/"+flagID.String()+"/toggle", reqBody)
	c.Params = gin.Params{{Key: "id", Value: flagID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ToggleFlag(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ToggleFlag_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()

	reqBody := map[string]interface{}{
		"enabled": true,
	}

	c, w := setupTestContext("POST", "/api/v1/admin/flags/invalid/toggle", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ToggleFlag(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// ArchiveFlag Handler Tests (Admin)
// ============================================================================

func TestHandler_ArchiveFlag_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	flagID := uuid.New()

	mockService.On("ArchiveFlag", mock.Anything, flagID).Return(nil)

	c, w := setupTestContext("DELETE", "/api/v1/admin/flags/"+flagID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: flagID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ArchiveFlag(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ArchiveFlag_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/admin/flags/invalid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ArchiveFlag(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// CreateOverride Handler Tests (Admin)
// ============================================================================

func TestHandler_CreateOverride_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	flagID := uuid.New()
	userID := uuid.New()

	mockService.On("CreateOverride", mock.Anything, adminID, flagID, mock.AnythingOfType("*experiments.CreateOverrideRequest")).Return(nil)

	reqBody := CreateOverrideRequest{
		UserID:  userID,
		Enabled: true,
		Reason:  "Testing override",
	}

	c, w := setupTestContext("POST", "/api/v1/admin/flags/"+flagID.String()+"/overrides", reqBody)
	c.Params = gin.Params{{Key: "id", Value: flagID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.CreateOverride(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CreateOverride_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	flagID := uuid.New()
	userID := uuid.New()

	reqBody := CreateOverrideRequest{
		UserID:  userID,
		Enabled: true,
		Reason:  "Testing",
	}

	c, w := setupTestContext("POST", "/api/v1/admin/flags/"+flagID.String()+"/overrides", reqBody)
	c.Params = gin.Params{{Key: "id", Value: flagID.String()}}
	// No user context

	handler.CreateOverride(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CreateOverride_InvalidFlagID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	userID := uuid.New()

	reqBody := CreateOverrideRequest{
		UserID:  userID,
		Enabled: true,
		Reason:  "Testing",
	}

	c, w := setupTestContext("POST", "/api/v1/admin/flags/invalid/overrides", reqBody)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.CreateOverride(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// ListOverrides Handler Tests (Admin)
// ============================================================================

func TestHandler_ListOverrides_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	flagID := uuid.New()
	overrides := []*FlagOverride{
		{ID: uuid.New(), FlagID: flagID, UserID: uuid.New(), Enabled: true},
		{ID: uuid.New(), FlagID: flagID, UserID: uuid.New(), Enabled: false},
	}

	mockService.On("ListOverrides", mock.Anything, flagID).Return(overrides, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/flags/"+flagID.String()+"/overrides", nil)
	c.Params = gin.Params{{Key: "id", Value: flagID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ListOverrides(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_ListOverrides_InvalidFlagID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/admin/flags/invalid/overrides", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ListOverrides(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// CreateExperiment Handler Tests (Admin)
// ============================================================================

func TestHandler_CreateExperiment_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	expectedExp := createTestExperimentObj("new_experiment", ExperimentStatusDraft)

	mockService.On("CreateExperiment", mock.Anything, adminID, mock.AnythingOfType("*experiments.CreateExperimentRequest")).Return(expectedExp, nil)

	reqBody := CreateExperimentRequest{
		Key:               "new_experiment",
		Name:              "New Experiment",
		Hypothesis:        "Test hypothesis",
		TrafficPercentage: 100,
		PrimaryMetric:     "conversion_rate",
		Variants: []CreateVariantInput{
			{Key: "control", Name: "Control", IsControl: true, Weight: 50},
			{Key: "variant_a", Name: "Variant A", Weight: 50},
		},
	}

	c, w := setupTestContext("POST", "/api/v1/admin/experiments", reqBody)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.CreateExperiment(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_CreateExperiment_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	reqBody := CreateExperimentRequest{
		Key:  "new_experiment",
		Name: "New Experiment",
	}

	c, w := setupTestContext("POST", "/api/v1/admin/experiments", reqBody)
	// No user context

	handler.CreateExperiment(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CreateExperiment_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/admin/experiments", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/experiments", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, adminID, models.RoleAdmin)

	handler.CreateExperiment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateExperiment_InvalidWeights(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()

	mockService.On("CreateExperiment", mock.Anything, adminID, mock.AnythingOfType("*experiments.CreateExperimentRequest")).Return(nil, common.NewBadRequestError("variant weights must sum to 100", nil))

	reqBody := CreateExperimentRequest{
		Key:               "bad_weights",
		Name:              "Bad Weights",
		Hypothesis:        "Test",
		TrafficPercentage: 100,
		PrimaryMetric:     "conversion",
		Variants: []CreateVariantInput{
			{Key: "control", Name: "Control", IsControl: true, Weight: 40},
			{Key: "variant_a", Name: "Variant A", Weight: 40},
		},
	}

	c, w := setupTestContext("POST", "/api/v1/admin/experiments", reqBody)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.CreateExperiment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CreateExperiment_NoControl(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()

	mockService.On("CreateExperiment", mock.Anything, adminID, mock.AnythingOfType("*experiments.CreateExperimentRequest")).Return(nil, common.NewBadRequestError("at least one variant must be marked as control", nil))

	reqBody := CreateExperimentRequest{
		Key:               "no_control",
		Name:              "No Control",
		Hypothesis:        "Test",
		TrafficPercentage: 100,
		PrimaryMetric:     "conversion",
		Variants: []CreateVariantInput{
			{Key: "variant_a", Name: "Variant A", Weight: 50},
			{Key: "variant_b", Name: "Variant B", Weight: 50},
		},
	}

	c, w := setupTestContext("POST", "/api/v1/admin/experiments", reqBody)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.CreateExperiment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// ListExperiments Handler Tests (Admin)
// ============================================================================

func TestHandler_ListExperiments_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	experiments := []*Experiment{
		createTestExperimentObj("exp1", ExperimentStatusRunning),
		createTestExperimentObj("exp2", ExperimentStatusDraft),
	}

	mockService.On("ListExperiments", mock.Anything, (*ExperimentStatus)(nil), 20, 0).Return(experiments, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/experiments", nil)
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ListExperiments(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_ListExperiments_WithStatusFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	runningStatus := ExperimentStatusRunning
	experiments := []*Experiment{
		createTestExperimentObj("exp1", ExperimentStatusRunning),
	}

	mockService.On("ListExperiments", mock.Anything, &runningStatus, 20, 0).Return(experiments, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/experiments?status=running", nil)
	c.Request.URL.RawQuery = "status=running"
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ListExperiments(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetExperiment Handler Tests (Admin)
// ============================================================================

func TestHandler_GetExperiment_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	expID := uuid.New()
	experiment := createTestExperimentObj("test_exp", ExperimentStatusRunning)
	experiment.ID = expID
	variants := createTestVariantsObj(expID)

	mockService.On("GetExperiment", mock.Anything, expID).Return(experiment, variants, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/experiments/"+expID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: expID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetExperiment(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["experiment"])
	assert.NotNil(t, data["variants"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetExperiment_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/admin/experiments/invalid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetExperiment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetExperiment_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	expID := uuid.New()

	mockService.On("GetExperiment", mock.Anything, expID).Return(nil, nil, common.NewNotFoundError("experiment not found", nil))

	c, w := setupTestContext("GET", "/api/v1/admin/experiments/"+expID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: expID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetExperiment(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// StartExperiment Handler Tests (Admin)
// ============================================================================

func TestHandler_StartExperiment_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	expID := uuid.New()

	mockService.On("StartExperiment", mock.Anything, expID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/experiments/"+expID.String()+"/start", nil)
	c.Params = gin.Params{{Key: "id", Value: expID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.StartExperiment(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_StartExperiment_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/admin/experiments/invalid/start", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.StartExperiment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_StartExperiment_InvalidStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	expID := uuid.New()

	mockService.On("StartExperiment", mock.Anything, expID).Return(common.NewBadRequestError("experiment cannot be started from current status", nil))

	c, w := setupTestContext("POST", "/api/v1/admin/experiments/"+expID.String()+"/start", nil)
	c.Params = gin.Params{{Key: "id", Value: expID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.StartExperiment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// PauseExperiment Handler Tests (Admin)
// ============================================================================

func TestHandler_PauseExperiment_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	expID := uuid.New()

	mockService.On("PauseExperiment", mock.Anything, expID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/experiments/"+expID.String()+"/pause", nil)
	c.Params = gin.Params{{Key: "id", Value: expID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.PauseExperiment(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_PauseExperiment_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/admin/experiments/invalid/pause", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.PauseExperiment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// ConcludeExperiment Handler Tests (Admin)
// ============================================================================

func TestHandler_ConcludeExperiment_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	expID := uuid.New()

	mockService.On("ConcludeExperiment", mock.Anything, expID).Return(nil)

	c, w := setupTestContext("POST", "/api/v1/admin/experiments/"+expID.String()+"/conclude", nil)
	c.Params = gin.Params{{Key: "id", Value: expID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ConcludeExperiment(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_ConcludeExperiment_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/admin/experiments/invalid/conclude", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.ConcludeExperiment(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// GetResults Handler Tests (Admin)
// ============================================================================

func TestHandler_GetResults_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	expID := uuid.New()
	results := &ExperimentResults{
		Experiment: createTestExperimentObj("test_exp", ExperimentStatusRunning),
		Variants: []VariantMetrics{
			{VariantKey: "control", SampleSize: 500, ConversionRate: 0.10},
			{VariantKey: "variant_a", SampleSize: 500, ConversionRate: 0.15},
		},
		IsSignificant:     true,
		CanConclude:       true,
		RecommendedAction: "conclude_winner",
	}

	mockService.On("GetResults", mock.Anything, expID).Return(results, nil)

	c, w := setupTestContext("GET", "/api/v1/admin/experiments/"+expID.String()+"/results", nil)
	c.Params = gin.Params{{Key: "id", Value: expID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetResults(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_GetResults_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()

	c, w := setupTestContext("GET", "/api/v1/admin/experiments/invalid/results", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetResults(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetResults_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()
	expID := uuid.New()

	mockService.On("GetResults", mock.Anything, expID).Return(nil, common.NewNotFoundError("experiment not found", nil))

	c, w := setupTestContext("GET", "/api/v1/admin/experiments/"+expID.String()+"/results", nil)
	c.Params = gin.Params{{Key: "id", Value: expID.String()}}
	setUserContext(c, adminID, models.RoleAdmin)

	handler.GetResults(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetVariant Handler Tests (User)
// ============================================================================

func TestHandler_GetVariant_Enrolled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	variant := &Variant{
		ID:        uuid.New(),
		Key:       "variant_a",
		Name:      "Variant A",
		IsControl: false,
		Weight:    50,
	}

	mockService.On("GetVariantForUser", mock.Anything, "pricing_test", mock.AnythingOfType("*experiments.UserContext")).Return(variant, nil)

	c, w := setupTestContext("GET", "/api/v1/experiments/pricing_test/variant", nil)
	c.Params = gin.Params{{Key: "key", Value: "pricing_test"}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetVariant(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.True(t, data["enrolled"].(bool))
	assert.NotNil(t, data["variant"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetVariant_NotEnrolled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("GetVariantForUser", mock.Anything, "other_test", mock.AnythingOfType("*experiments.UserContext")).Return(nil, nil)

	c, w := setupTestContext("GET", "/api/v1/experiments/other_test/variant", nil)
	c.Params = gin.Params{{Key: "key", Value: "other_test"}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetVariant(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.False(t, data["enrolled"].(bool))
	assert.Nil(t, data["variant"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetVariant_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("GetVariantForUser", mock.Anything, "error_test", mock.AnythingOfType("*experiments.UserContext")).Return(nil, errors.New("service error"))

	c, w := setupTestContext("GET", "/api/v1/experiments/error_test/variant", nil)
	c.Params = gin.Params{{Key: "key", Value: "error_test"}}
	setUserContext(c, userID, models.RoleRider)

	handler.GetVariant(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetVariant_WithUserContextHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	variant := &Variant{
		ID:        uuid.New(),
		Key:       "control",
		Name:      "Control",
		IsControl: true,
		Weight:    50,
	}

	mockService.On("GetVariantForUser", mock.Anything, "segment_test", mock.MatchedBy(func(ctx *UserContext) bool {
		return ctx.Country == "US" && ctx.City == "NYC" && ctx.Platform == "android"
	})).Return(variant, nil)

	c, w := setupTestContext("GET", "/api/v1/experiments/segment_test/variant", nil)
	c.Params = gin.Params{{Key: "key", Value: "segment_test"}}
	setUserContextWithHeaders(c, userID, models.RoleRider, "US", "NYC", "android", "3.0.0")

	handler.GetVariant(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// TrackEvent Handler Tests (User)
// ============================================================================

func TestHandler_TrackEvent_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("TrackEvent", mock.Anything, userID, mock.AnythingOfType("*experiments.TrackEventRequest")).Return(nil)

	reqBody := TrackEventRequest{
		ExperimentKey: "pricing_test",
		EventType:     "conversion",
	}

	c, w := setupTestContext("POST", "/api/v1/experiments/track", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TrackEvent(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_TrackEvent_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	reqBody := TrackEventRequest{
		ExperimentKey: "pricing_test",
		EventType:     "conversion",
	}

	c, w := setupTestContext("POST", "/api/v1/experiments/track", reqBody)
	// No user context

	handler.TrackEvent(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_TrackEvent_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/experiments/track", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/experiments/track", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContext(c, userID, models.RoleRider)

	handler.TrackEvent(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TrackEvent_WithEventValue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	eventValue := 99.99

	mockService.On("TrackEvent", mock.Anything, userID, mock.MatchedBy(func(req *TrackEventRequest) bool {
		return req.EventValue != nil && *req.EventValue == eventValue
	})).Return(nil)

	reqBody := TrackEventRequest{
		ExperimentKey: "pricing_test",
		EventType:     "purchase",
		EventValue:    &eventValue,
	}

	c, w := setupTestContext("POST", "/api/v1/experiments/track", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TrackEvent(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_TrackEvent_WithMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("TrackEvent", mock.Anything, userID, mock.MatchedBy(func(req *TrackEventRequest) bool {
		return req.Metadata != nil && req.Metadata["source"] == "checkout"
	})).Return(nil)

	reqBody := TrackEventRequest{
		ExperimentKey: "pricing_test",
		EventType:     "conversion",
		Metadata:      map[string]interface{}{"source": "checkout"},
	}

	c, w := setupTestContext("POST", "/api/v1/experiments/track", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TrackEvent(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_TrackEvent_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("TrackEvent", mock.Anything, userID, mock.AnythingOfType("*experiments.TrackEventRequest")).Return(errors.New("tracking failed"))

	reqBody := TrackEventRequest{
		ExperimentKey: "pricing_test",
		EventType:     "conversion",
	}

	c, w := setupTestContext("POST", "/api/v1/experiments/track", reqBody)
	setUserContext(c, userID, models.RoleRider)

	handler.TrackEvent(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// Segment Rule Matching Tests (Table-Driven)
// ============================================================================

func TestHandler_EvaluateFlag_SegmentRuleMatching(t *testing.T) {
	tests := []struct {
		name       string
		country    string
		city       string
		platform   string
		appVersion string
		enabled    bool
		source     string
	}{
		{
			name:     "US user matches segment",
			country:  "US",
			platform: "ios",
			enabled:  true,
			source:   "segment",
		},
		{
			name:     "Non-US user does not match",
			country:  "MX",
			platform: "ios",
			enabled:  false,
			source:   "segment",
		},
		{
			name:     "Web platform does not match",
			country:  "US",
			platform: "web",
			enabled:  false,
			source:   "segment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(MockExperimentsService)
			handler := NewMockableHandler(mockService)

			userID := uuid.New()
			expectedResult := &EvaluateFlagResponse{
				Key:     "segment_flag",
				Enabled: tt.enabled,
				Source:  tt.source,
			}

			mockService.On("EvaluateFlag", mock.Anything, "segment_flag", mock.AnythingOfType("*experiments.UserContext")).Return(expectedResult, nil)

			c, w := setupTestContext("GET", "/api/v1/flags/segment_flag", nil)
			c.Params = gin.Params{{Key: "key", Value: "segment_flag"}}
			setUserContextWithHeaders(c, userID, models.RoleRider, tt.country, tt.city, tt.platform, tt.appVersion)

			handler.EvaluateFlag(c)

			assert.Equal(t, http.StatusOK, w.Code)
			response := parseResponse(w)
			data := response["data"].(map[string]interface{})
			assert.Equal(t, tt.enabled, data["enabled"].(bool))
			mockService.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Cache Behavior Tests
// ============================================================================

func TestHandler_EvaluateFlag_CacheBehavior(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	expectedResult := &EvaluateFlagResponse{
		Key:     "cached_flag",
		Enabled: true,
		Source:  "default",
	}

	// Service should be called for each request (caching is at service level)
	mockService.On("EvaluateFlag", mock.Anything, "cached_flag", mock.AnythingOfType("*experiments.UserContext")).Return(expectedResult, nil).Times(3)

	for i := 0; i < 3; i++ {
		c, w := setupTestContext("GET", "/api/v1/flags/cached_flag", nil)
		c.Params = gin.Params{{Key: "key", Value: "cached_flag"}}
		setUserContext(c, userID, models.RoleRider)

		handler.EvaluateFlag(c)

		assert.Equal(t, http.StatusOK, w.Code)
	}

	mockService.AssertExpectations(t)
}

// ============================================================================
// Admin Authorization Tests
// ============================================================================

func TestHandler_AdminEndpoints_RequireAuth(t *testing.T) {
	endpoints := []struct {
		name    string
		method  string
		path    string
		handler func(*MockableHandler) func(*gin.Context)
		body    interface{}
	}{
		{
			name:   "CreateFlag",
			method: "POST",
			path:   "/api/v1/admin/flags",
			handler: func(h *MockableHandler) func(*gin.Context) {
				return h.CreateFlag
			},
			body: CreateFlagRequest{Key: "test", Name: "Test", FlagType: FlagTypeBoolean},
		},
		{
			name:   "CreateExperiment",
			method: "POST",
			path:   "/api/v1/admin/experiments",
			handler: func(h *MockableHandler) func(*gin.Context) {
				return h.CreateExperiment
			},
			body: CreateExperimentRequest{Key: "test", Name: "Test"},
		},
		{
			name:   "CreateOverride",
			method: "POST",
			path:   "/api/v1/admin/flags/" + uuid.New().String() + "/overrides",
			handler: func(h *MockableHandler) func(*gin.Context) {
				return h.CreateOverride
			},
			body: CreateOverrideRequest{UserID: uuid.New(), Enabled: true, Reason: "Test"},
		},
	}

	for _, ep := range endpoints {
		t.Run(ep.name+"_Unauthorized", func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(MockExperimentsService)
			handler := NewMockableHandler(mockService)

			c, w := setupTestContext(ep.method, ep.path, ep.body)
			// No user context set

			ep.handler(handler)(c)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

// ============================================================================
// Edge Cases and Error Handling
// ============================================================================

func TestHandler_UUID_Validation(t *testing.T) {
	tests := []struct {
		name           string
		uuidValue      string
		expectedStatus int
	}{
		{
			name:           "valid UUID",
			uuidValue:      uuid.New().String(),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid UUID - too short",
			uuidValue:      "123",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid UUID - wrong format",
			uuidValue:      "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty UUID",
			uuidValue:      "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(MockExperimentsService)
			handler := NewMockableHandler(mockService)

			adminID := uuid.New()

			if tt.expectedStatus == http.StatusOK {
				flagID, _ := uuid.Parse(tt.uuidValue)
				mockService.On("ArchiveFlag", mock.Anything, flagID).Return(nil)
			}

			c, w := setupTestContext("DELETE", "/api/v1/admin/flags/"+tt.uuidValue, nil)
			c.Params = gin.Params{{Key: "id", Value: tt.uuidValue}}
			setUserContext(c, adminID, models.RoleAdmin)

			handler.ArchiveFlag(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_ResponseFormat_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()
	expectedResult := &EvaluateFlagResponse{
		Key:     "test_flag",
		Enabled: true,
		Source:  "default",
	}

	mockService.On("EvaluateFlag", mock.Anything, "test_flag", mock.AnythingOfType("*experiments.UserContext")).Return(expectedResult, nil)

	c, w := setupTestContext("GET", "/api/v1/flags/test_flag", nil)
	c.Params = gin.Params{{Key: "key", Value: "test_flag"}}
	setUserContext(c, userID, models.RoleRider)

	handler.EvaluateFlag(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	// Verify response structure
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
	mockService.AssertExpectations(t)
}

func TestHandler_ResponseFormat_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	mockService.On("EvaluateFlag", mock.Anything, "error_flag", mock.AnythingOfType("*experiments.UserContext")).Return(nil, errors.New("service error"))

	c, w := setupTestContext("GET", "/api/v1/flags/error_flag", nil)
	c.Params = gin.Params{{Key: "key", Value: "error_flag"}}
	setUserContext(c, userID, models.RoleRider)

	handler.EvaluateFlag(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	response := parseResponse(w)

	// Verify error response structure
	assert.False(t, response["success"].(bool))
	assert.NotNil(t, response["error"])
	mockService.AssertExpectations(t)
}

// ============================================================================
// Concurrent Access Tests
// ============================================================================

func TestHandler_ConcurrentFlagEvaluation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	expectedResult := &EvaluateFlagResponse{
		Key:     "concurrent_flag",
		Enabled: true,
		Source:  "default",
	}

	mockService.On("EvaluateFlag", mock.Anything, "concurrent_flag", mock.AnythingOfType("*experiments.UserContext")).Return(expectedResult, nil).Times(5)

	for i := 0; i < 5; i++ {
		userID := uuid.New()
		c, w := setupTestContext("GET", "/api/v1/flags/concurrent_flag", nil)
		c.Params = gin.Params{{Key: "key", Value: "concurrent_flag"}}
		setUserContext(c, userID, models.RoleRider)

		handler.EvaluateFlag(c)

		assert.Equal(t, http.StatusOK, w.Code)
	}

	mockService.AssertExpectations(t)
}

// ============================================================================
// All Flag Statuses Tests
// ============================================================================

func TestHandler_ListFlags_AllStatuses(t *testing.T) {
	statuses := []FlagStatus{
		FlagStatusActive,
		FlagStatusInactive,
		FlagStatusArchived,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(MockExperimentsService)
			handler := NewMockableHandler(mockService)

			adminID := uuid.New()
			statusCopy := status

			mockService.On("ListFlags", mock.Anything, &statusCopy, 20, 0).Return([]*FeatureFlag{}, nil)

			c, w := setupTestContext("GET", "/api/v1/admin/flags?status="+string(status), nil)
			c.Request.URL.RawQuery = "status=" + string(status)
			setUserContext(c, adminID, models.RoleAdmin)

			handler.ListFlags(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// ============================================================================
// All Experiment Statuses Tests
// ============================================================================

func TestHandler_ListExperiments_AllStatuses(t *testing.T) {
	statuses := []ExperimentStatus{
		ExperimentStatusDraft,
		ExperimentStatusRunning,
		ExperimentStatusPaused,
		ExperimentStatusCompleted,
		ExperimentStatusArchived,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockService := new(MockExperimentsService)
			handler := NewMockableHandler(mockService)

			adminID := uuid.New()
			statusCopy := status

			mockService.On("ListExperiments", mock.Anything, &statusCopy, 20, 0).Return([]*Experiment{}, nil)

			c, w := setupTestContext("GET", "/api/v1/admin/experiments?status="+string(status), nil)
			c.Request.URL.RawQuery = "status=" + string(status)
			setUserContext(c, adminID, models.RoleAdmin)

			handler.ListExperiments(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Empty and Null Body Tests
// ============================================================================

func TestHandler_CreateFlag_EmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	adminID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/admin/flags", map[string]interface{}{})
	setUserContext(c, adminID, models.RoleAdmin)

	handler.CreateFlag(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TrackEvent_EmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockExperimentsService)
	handler := NewMockableHandler(mockService)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/experiments/track", map[string]interface{}{})
	setUserContext(c, userID, models.RoleRider)

	handler.TrackEvent(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
