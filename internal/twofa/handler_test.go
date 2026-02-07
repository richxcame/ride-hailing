package twofa

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// Mock Service Interface
// ============================================================================

// TwoFAServiceInterface defines the interface for the 2FA service used by the handler
type TwoFAServiceInterface interface {
	SendOTP(ctx context.Context, userID uuid.UUID, phoneNumber string, otpType OTPType, ipAddress, userAgent *string) (*SendOTPResponse, error)
	VerifyOTP(ctx context.Context, userID uuid.UUID, otp string, otpType OTPType, ipAddress, userAgent *string) error
	Enable2FA(ctx context.Context, userID uuid.UUID, method TwoFAMethod, phoneNumber string, ipAddress, userAgent *string) (*Enable2FAResponse, error)
	Disable2FA(ctx context.Context, userID uuid.UUID, otp, backupCode string, ipAddress, userAgent *string) error
	Get2FAStatus(ctx context.Context, userID uuid.UUID) (*TwoFAStatusResponse, error)
	RegenerateBackupCodes(ctx context.Context, userID uuid.UUID, otp string, ipAddress, userAgent *string) ([]string, error)
	GetTrustedDevices(ctx context.Context, userID uuid.UUID, currentToken string) ([]*TrustedDeviceResponse, error)
	RevokeTrustedDevice(ctx context.Context, userID, deviceID uuid.UUID, ipAddress, userAgent *string) error
	VerifyPhone(ctx context.Context, userID uuid.UUID, otp string, ipAddress, userAgent *string) error
	VerifyTOTP(ctx context.Context, userID uuid.UUID, code string) error
}

// MockTwoFAService is a mock implementation of TwoFAServiceInterface
type MockTwoFAService struct {
	mock.Mock
}

func (m *MockTwoFAService) SendOTP(ctx context.Context, userID uuid.UUID, phoneNumber string, otpType OTPType, ipAddress, userAgent *string) (*SendOTPResponse, error) {
	args := m.Called(ctx, userID, phoneNumber, otpType, ipAddress, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SendOTPResponse), args.Error(1)
}

func (m *MockTwoFAService) VerifyOTP(ctx context.Context, userID uuid.UUID, otp string, otpType OTPType, ipAddress, userAgent *string) error {
	args := m.Called(ctx, userID, otp, otpType, ipAddress, userAgent)
	return args.Error(0)
}

func (m *MockTwoFAService) Enable2FA(ctx context.Context, userID uuid.UUID, method TwoFAMethod, phoneNumber string, ipAddress, userAgent *string) (*Enable2FAResponse, error) {
	args := m.Called(ctx, userID, method, phoneNumber, ipAddress, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Enable2FAResponse), args.Error(1)
}

func (m *MockTwoFAService) Disable2FA(ctx context.Context, userID uuid.UUID, otp, backupCode string, ipAddress, userAgent *string) error {
	args := m.Called(ctx, userID, otp, backupCode, ipAddress, userAgent)
	return args.Error(0)
}

func (m *MockTwoFAService) Get2FAStatus(ctx context.Context, userID uuid.UUID) (*TwoFAStatusResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TwoFAStatusResponse), args.Error(1)
}

func (m *MockTwoFAService) RegenerateBackupCodes(ctx context.Context, userID uuid.UUID, otp string, ipAddress, userAgent *string) ([]string, error) {
	args := m.Called(ctx, userID, otp, ipAddress, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockTwoFAService) GetTrustedDevices(ctx context.Context, userID uuid.UUID, currentToken string) ([]*TrustedDeviceResponse, error) {
	args := m.Called(ctx, userID, currentToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*TrustedDeviceResponse), args.Error(1)
}

func (m *MockTwoFAService) RevokeTrustedDevice(ctx context.Context, userID, deviceID uuid.UUID, ipAddress, userAgent *string) error {
	args := m.Called(ctx, userID, deviceID, ipAddress, userAgent)
	return args.Error(0)
}

func (m *MockTwoFAService) VerifyPhone(ctx context.Context, userID uuid.UUID, otp string, ipAddress, userAgent *string) error {
	args := m.Called(ctx, userID, otp, ipAddress, userAgent)
	return args.Error(0)
}

func (m *MockTwoFAService) VerifyTOTP(ctx context.Context, userID uuid.UUID, code string) error {
	args := m.Called(ctx, userID, code)
	return args.Error(0)
}

// MockRepository for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) SetTOTPVerified(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// ============================================================================
// Testable Handler (wraps mock service)
// ============================================================================

// TestableHandler provides testable handler methods with mock service injection
type TestableHandler struct {
	service  *MockTwoFAService
	mockRepo *MockRepository
}

func NewTestableHandler(mockService *MockTwoFAService, mockRepo *MockRepository) *TestableHandler {
	return &TestableHandler{service: mockService, mockRepo: mockRepo}
}

// SendOTP handles sending OTP - testable version
func (h *TestableHandler) SendOTP(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		OTPType OTPType `json:"otp_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	phoneNumber, _ := c.Get("phone_number")
	phone, ok := phoneNumber.(string)
	if !ok || phone == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "phone number not available")
		return
	}

	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")

	response, err := h.service.SendOTP(c.Request.Context(), userID.(uuid.UUID), phone, req.OTPType, &ip, &ua)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send OTP")
		return
	}

	common.SuccessResponse(c, response)
}

// VerifyOTP handles verifying OTP - testable version
func (h *TestableHandler) VerifyOTP(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")

	if err := h.service.VerifyOTP(c.Request.Context(), userID.(uuid.UUID), req.OTP, req.OTPType, &ip, &ua); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusBadRequest, "OTP verification failed")
		return
	}

	common.SuccessResponse(c, gin.H{
		"success": true,
		"message": "OTP verified successfully",
	})
}

// Enable2FA handles enabling 2FA - testable version
func (h *TestableHandler) Enable2FA(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req Enable2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	phoneNumber, _ := c.Get("phone_number")
	phone, _ := phoneNumber.(string)

	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")

	response, err := h.service.Enable2FA(c.Request.Context(), userID.(uuid.UUID), req.Method, phone, &ip, &ua)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to enable 2FA")
		return
	}

	common.SuccessResponse(c, response)
}

// Disable2FA handles disabling 2FA - testable version
func (h *TestableHandler) Disable2FA(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req Disable2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")

	if err := h.service.Disable2FA(c.Request.Context(), userID.(uuid.UUID), req.OTP, req.BackupCode, &ip, &ua); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to disable 2FA")
		return
	}

	common.SuccessResponse(c, gin.H{
		"success": true,
		"message": "2FA disabled successfully",
	})
}

// Get2FAStatus handles getting 2FA status - testable version
func (h *TestableHandler) Get2FAStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	status, err := h.service.Get2FAStatus(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get 2FA status")
		return
	}

	common.SuccessResponse(c, status)
}

// RegenerateBackupCodes handles regenerating backup codes - testable version
func (h *TestableHandler) RegenerateBackupCodes(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		OTP string `json:"otp" binding:"required,len=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")

	codes, err := h.service.RegenerateBackupCodes(c.Request.Context(), userID.(uuid.UUID), req.OTP, &ip, &ua)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to regenerate backup codes")
		return
	}

	common.SuccessResponse(c, gin.H{
		"backup_codes": codes,
		"message":      "Please save these backup codes securely. They will not be shown again.",
	})
}

// GetTrustedDevices handles getting trusted devices - testable version
func (h *TestableHandler) GetTrustedDevices(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	currentToken, _ := c.Cookie("trusted_device")

	devices, err := h.service.GetTrustedDevices(c.Request.Context(), userID.(uuid.UUID), currentToken)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get devices")
		return
	}

	common.SuccessResponse(c, gin.H{
		"devices": devices,
	})
}

// RevokeTrustedDevice handles revoking a trusted device - testable version
func (h *TestableHandler) RevokeTrustedDevice(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	deviceIDStr := c.Param("id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid device ID")
		return
	}

	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")

	if err := h.service.RevokeTrustedDevice(c.Request.Context(), userID.(uuid.UUID), deviceID, &ip, &ua); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to revoke device")
		return
	}

	common.SuccessResponse(c, gin.H{
		"success": true,
		"message": "Device revoked successfully",
	})
}

// SendPhoneVerification handles sending phone verification - testable version
func (h *TestableHandler) SendPhoneVerification(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		PhoneNumber string `json:"phone_number"`
	}
	_ = c.ShouldBindJSON(&req)

	phone := req.PhoneNumber
	if phone == "" {
		phoneNumber, _ := c.Get("phone_number")
		phone, _ = phoneNumber.(string)
	}

	if phone == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "phone number required")
		return
	}

	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")

	response, err := h.service.SendOTP(c.Request.Context(), userID.(uuid.UUID), phone, OTPTypePhoneVerification, &ip, &ua)
	if err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to send verification")
		return
	}

	common.SuccessResponse(c, response)
}

// VerifyPhone handles phone verification - testable version
func (h *TestableHandler) VerifyPhone(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		OTP string `json:"otp" binding:"required,len=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")

	if err := h.service.VerifyPhone(c.Request.Context(), userID.(uuid.UUID), req.OTP, &ip, &ua); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusBadRequest, "phone verification failed")
		return
	}

	common.SuccessResponse(c, gin.H{
		"success": true,
		"message": "Phone number verified successfully",
	})
}

// VerifyTOTP handles TOTP verification - testable version
func (h *TestableHandler) VerifyTOTP(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Code string `json:"code" binding:"required,len=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.VerifyTOTP(c.Request.Context(), userID.(uuid.UUID), req.Code); err != nil {
		if appErr, ok := err.(*common.AppError); ok {
			common.AppErrorResponse(c, appErr)
			return
		}
		common.ErrorResponse(c, http.StatusBadRequest, "TOTP verification failed")
		return
	}

	// Mark TOTP as verified
	if h.mockRepo != nil {
		_ = h.mockRepo.SetTOTPVerified(c.Request.Context(), userID.(uuid.UUID))
	}

	common.SuccessResponse(c, gin.H{
		"success": true,
		"message": "TOTP verified successfully",
	})
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

func setUserContext(c *gin.Context, userID uuid.UUID) {
	c.Set("user_id", userID)
	c.Set("user_role", "rider")
	c.Set("user_email", "test@example.com")
}

func setUserContextWithPhone(c *gin.Context, userID uuid.UUID, phoneNumber string) {
	setUserContext(c, userID)
	c.Set("phone_number", phoneNumber)
}

func createTestSendOTPResponse() *SendOTPResponse {
	return &SendOTPResponse{
		Message:     "OTP sent successfully",
		Destination: "+1234567****",
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}
}

func createTest2FAStatusResponse(enabled bool) *TwoFAStatusResponse {
	return &TwoFAStatusResponse{
		Enabled:             enabled,
		Method:              MethodSMS,
		PhoneVerified:       true,
		TOTPVerified:        false,
		BackupCodesCount:    10,
		TrustedDevicesCount: 2,
	}
}

func createTestEnable2FAResponse() *Enable2FAResponse {
	return &Enable2FAResponse{
		Message:        "2FA enabled successfully",
		Method:         string(MethodSMS),
		BackupCodes:    []string{"ABCD-1234", "EFGH-5678", "IJKL-9012"},
		RequiresOTP:    true,
		OTPDestination: "+1234567****",
	}
}

func createTestTrustedDevices() []*TrustedDeviceResponse {
	now := time.Now()
	return []*TrustedDeviceResponse{
		{
			ID:         uuid.New(),
			DeviceName: "iPhone 15",
			TrustedAt:  now.Add(-24 * time.Hour),
			LastUsedAt: &now,
			IPAddress:  "192.168.1.1",
			IsCurrent:  true,
		},
		{
			ID:         uuid.New(),
			DeviceName: "Chrome on Mac",
			TrustedAt:  now.Add(-48 * time.Hour),
			LastUsedAt: nil,
			IPAddress:  "192.168.1.2",
			IsCurrent:  false,
		},
	}
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	return response
}

// ============================================================================
// SendOTP Handler Tests
// ============================================================================

func TestHandler_SendOTP_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	phoneNumber := "+12345678901"
	expectedResponse := createTestSendOTPResponse()

	mockService.On("SendOTP", mock.Anything, userID, phoneNumber, OTPTypeLogin, mock.Anything, mock.Anything).Return(expectedResponse, nil)

	reqBody := map[string]interface{}{
		"otp_type": "login",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/otp/send", reqBody)
	setUserContextWithPhone(c, userID, phoneNumber)

	handler.SendOTP(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_SendOTP_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	reqBody := map[string]interface{}{
		"otp_type": "login",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/otp/send", reqBody)
	// Don't set user context

	handler.SendOTP(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_SendOTP_MissingOTPType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	phoneNumber := "+12345678901"

	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/2fa/otp/send", reqBody)
	setUserContextWithPhone(c, userID, phoneNumber)

	handler.SendOTP(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SendOTP_MissingPhoneNumber(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"otp_type": "login",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/otp/send", reqBody)
	setUserContext(c, userID) // No phone number

	handler.SendOTP(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "phone number not available")
}

func TestHandler_SendOTP_RateLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	phoneNumber := "+12345678901"

	mockService.On("SendOTP", mock.Anything, userID, phoneNumber, OTPTypeLogin, mock.Anything, mock.Anything).
		Return(nil, common.NewTooManyRequestsError("too many OTP requests"))

	reqBody := map[string]interface{}{
		"otp_type": "login",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/otp/send", reqBody)
	setUserContextWithPhone(c, userID, phoneNumber)

	handler.SendOTP(c)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SendOTP_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	phoneNumber := "+12345678901"

	mockService.On("SendOTP", mock.Anything, userID, phoneNumber, OTPTypeLogin, mock.Anything, mock.Anything).
		Return(nil, errors.New("SMS service unavailable"))

	reqBody := map[string]interface{}{
		"otp_type": "login",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/otp/send", reqBody)
	setUserContextWithPhone(c, userID, phoneNumber)

	handler.SendOTP(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SendOTP_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	phoneNumber := "+12345678901"

	c, w := setupTestContext("POST", "/api/v1/2fa/otp/send", nil)
	c.Request = httptest.NewRequest("POST", "/api/v1/2fa/otp/send", bytes.NewReader([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	setUserContextWithPhone(c, userID, phoneNumber)

	handler.SendOTP(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SendOTP_AllOTPTypes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	otpTypes := []OTPType{
		OTPTypeLogin,
		OTPTypePhoneVerification,
		OTPTypeEnable2FA,
		OTPTypeDisable2FA,
		OTPTypePasswordReset,
	}

	for _, otpType := range otpTypes {
		t.Run(string(otpType), func(t *testing.T) {
			mockService := new(MockTwoFAService)
			handler := NewTestableHandler(mockService, nil)

			userID := uuid.New()
			phoneNumber := "+12345678901"
			expectedResponse := createTestSendOTPResponse()

			mockService.On("SendOTP", mock.Anything, userID, phoneNumber, otpType, mock.Anything, mock.Anything).Return(expectedResponse, nil)

			reqBody := map[string]interface{}{
				"otp_type": otpType,
			}

			c, w := setupTestContext("POST", "/api/v1/2fa/otp/send", reqBody)
			setUserContextWithPhone(c, userID, phoneNumber)

			handler.SendOTP(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// ============================================================================
// VerifyOTP Handler Tests
// ============================================================================

func TestHandler_VerifyOTP_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("VerifyOTP", mock.Anything, userID, "123456", OTPTypeLogin, mock.Anything, mock.Anything).Return(nil)

	reqBody := VerifyOTPRequest{
		OTP:     "123456",
		OTPType: OTPTypeLogin,
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/otp/verify", reqBody)
	setUserContext(c, userID)

	handler.VerifyOTP(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_VerifyOTP_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	reqBody := VerifyOTPRequest{
		OTP:     "123456",
		OTPType: OTPTypeLogin,
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/otp/verify", reqBody)

	handler.VerifyOTP(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_VerifyOTP_MissingOTP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"otp_type": "login",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/otp/verify", reqBody)
	setUserContext(c, userID)

	handler.VerifyOTP(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_VerifyOTP_InvalidOTPLength(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	tests := []struct {
		name string
		otp  string
	}{
		{"too short", "12345"},
		{"too long", "1234567"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := map[string]interface{}{
				"otp":      tt.otp,
				"otp_type": "login",
			}

			c, w := setupTestContext("POST", "/api/v1/2fa/otp/verify", reqBody)
			setUserContext(c, userID)

			handler.VerifyOTP(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandler_VerifyOTP_InvalidOTP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("VerifyOTP", mock.Anything, userID, "000000", OTPTypeLogin, mock.Anything, mock.Anything).
		Return(common.NewBadRequestError("invalid OTP", nil))

	reqBody := VerifyOTPRequest{
		OTP:     "000000",
		OTPType: OTPTypeLogin,
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/otp/verify", reqBody)
	setUserContext(c, userID)

	handler.VerifyOTP(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_VerifyOTP_ExpiredOTP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("VerifyOTP", mock.Anything, userID, "123456", OTPTypeLogin, mock.Anything, mock.Anything).
		Return(common.NewBadRequestError("no active OTP found or OTP expired", nil))

	reqBody := VerifyOTPRequest{
		OTP:     "123456",
		OTPType: OTPTypeLogin,
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/otp/verify", reqBody)
	setUserContext(c, userID)

	handler.VerifyOTP(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_VerifyOTP_MaxAttemptsExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("VerifyOTP", mock.Anything, userID, "123456", OTPTypeLogin, mock.Anything, mock.Anything).
		Return(common.NewTooManyRequestsError("maximum OTP attempts exceeded"))

	reqBody := VerifyOTPRequest{
		OTP:     "123456",
		OTPType: OTPTypeLogin,
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/otp/verify", reqBody)
	setUserContext(c, userID)

	handler.VerifyOTP(c)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_VerifyOTP_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("VerifyOTP", mock.Anything, userID, "123456", OTPTypeLogin, mock.Anything, mock.Anything).
		Return(errors.New("database error"))

	reqBody := VerifyOTPRequest{
		OTP:     "123456",
		OTPType: OTPTypeLogin,
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/otp/verify", reqBody)
	setUserContext(c, userID)

	handler.VerifyOTP(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// Enable2FA Handler Tests
// ============================================================================

func TestHandler_Enable2FA_Success_SMS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	phoneNumber := "+12345678901"
	expectedResponse := createTestEnable2FAResponse()

	mockService.On("Enable2FA", mock.Anything, userID, MethodSMS, phoneNumber, mock.Anything, mock.Anything).Return(expectedResponse, nil)

	reqBody := Enable2FARequest{
		Method: MethodSMS,
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/enable", reqBody)
	setUserContextWithPhone(c, userID, phoneNumber)

	handler.Enable2FA(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_Enable2FA_Success_TOTP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	phoneNumber := "+12345678901"
	expectedResponse := &Enable2FAResponse{
		Message:     "2FA enabled successfully",
		Method:      string(MethodTOTP),
		TOTPSecret:  "JBSWY3DPEHPK3PXP",
		TOTPQRCode:  "otpauth://totp/RideHailing:test@example.com?secret=JBSWY3DPEHPK3PXP",
		BackupCodes: []string{"ABCD-1234", "EFGH-5678"},
	}

	mockService.On("Enable2FA", mock.Anything, userID, MethodTOTP, phoneNumber, mock.Anything, mock.Anything).Return(expectedResponse, nil)

	reqBody := Enable2FARequest{
		Method: MethodTOTP,
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/enable", reqBody)
	setUserContextWithPhone(c, userID, phoneNumber)

	handler.Enable2FA(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.NotEmpty(t, data["totp_secret"])
	mockService.AssertExpectations(t)
}

func TestHandler_Enable2FA_Success_Both(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	phoneNumber := "+12345678901"
	expectedResponse := &Enable2FAResponse{
		Message:        "2FA enabled successfully",
		Method:         string(MethodBoth),
		TOTPSecret:     "JBSWY3DPEHPK3PXP",
		TOTPQRCode:     "otpauth://totp/RideHailing:test@example.com?secret=JBSWY3DPEHPK3PXP",
		BackupCodes:    []string{"ABCD-1234", "EFGH-5678"},
		RequiresOTP:    true,
		OTPDestination: "+1234567****",
	}

	mockService.On("Enable2FA", mock.Anything, userID, MethodBoth, phoneNumber, mock.Anything, mock.Anything).Return(expectedResponse, nil)

	reqBody := Enable2FARequest{
		Method: MethodBoth,
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/enable", reqBody)
	setUserContextWithPhone(c, userID, phoneNumber)

	handler.Enable2FA(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_Enable2FA_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	reqBody := Enable2FARequest{
		Method: MethodSMS,
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/enable", reqBody)

	handler.Enable2FA(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Enable2FA_MissingMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/2fa/enable", reqBody)
	setUserContext(c, userID)

	handler.Enable2FA(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Enable2FA_InvalidMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"method": "invalid",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/enable", reqBody)
	setUserContext(c, userID)

	handler.Enable2FA(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Enable2FA_AlreadyEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	phoneNumber := "+12345678901"

	mockService.On("Enable2FA", mock.Anything, userID, MethodSMS, phoneNumber, mock.Anything, mock.Anything).
		Return(nil, common.NewBadRequestError("2FA is already enabled", nil))

	reqBody := Enable2FARequest{
		Method: MethodSMS,
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/enable", reqBody)
	setUserContextWithPhone(c, userID, phoneNumber)

	handler.Enable2FA(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_Enable2FA_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	phoneNumber := "+12345678901"

	mockService.On("Enable2FA", mock.Anything, userID, MethodSMS, phoneNumber, mock.Anything, mock.Anything).
		Return(nil, errors.New("database error"))

	reqBody := Enable2FARequest{
		Method: MethodSMS,
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/enable", reqBody)
	setUserContextWithPhone(c, userID, phoneNumber)

	handler.Enable2FA(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// Disable2FA Handler Tests
// ============================================================================

func TestHandler_Disable2FA_Success_WithOTP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("Disable2FA", mock.Anything, userID, "123456", "", mock.Anything, mock.Anything).Return(nil)

	reqBody := Disable2FARequest{
		OTP: "123456",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/disable", reqBody)
	setUserContext(c, userID)

	handler.Disable2FA(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.True(t, data["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_Disable2FA_Success_WithBackupCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	// When both OTP and backup code are provided, the service handles the logic
	// The handler validates OTP is 6 chars, so we must provide a valid format
	mockService.On("Disable2FA", mock.Anything, userID, "123456", "ABCD-1234", mock.Anything, mock.Anything).Return(nil)

	reqBody := Disable2FARequest{
		OTP:        "123456",
		BackupCode: "ABCD-1234",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/disable", reqBody)
	setUserContext(c, userID)

	handler.Disable2FA(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_Disable2FA_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	reqBody := Disable2FARequest{
		OTP: "123456",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/disable", reqBody)

	handler.Disable2FA(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Disable2FA_MissingOTP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/2fa/disable", reqBody)
	setUserContext(c, userID)

	handler.Disable2FA(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Disable2FA_InvalidVerification(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("Disable2FA", mock.Anything, userID, "000000", "", mock.Anything, mock.Anything).
		Return(common.NewBadRequestError("invalid OTP or backup code", nil))

	reqBody := Disable2FARequest{
		OTP: "000000",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/disable", reqBody)
	setUserContext(c, userID)

	handler.Disable2FA(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_Disable2FA_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("Disable2FA", mock.Anything, userID, "123456", "", mock.Anything, mock.Anything).
		Return(errors.New("database error"))

	reqBody := Disable2FARequest{
		OTP: "123456",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/disable", reqBody)
	setUserContext(c, userID)

	handler.Disable2FA(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// Get2FAStatus Handler Tests
// ============================================================================

func TestHandler_Get2FAStatus_Success_Enabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	expectedStatus := createTest2FAStatusResponse(true)

	mockService.On("Get2FAStatus", mock.Anything, userID).Return(expectedStatus, nil)

	c, w := setupTestContext("GET", "/api/v1/2fa/status", nil)
	setUserContext(c, userID)

	handler.Get2FAStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.True(t, data["enabled"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_Get2FAStatus_Success_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	expectedStatus := createTest2FAStatusResponse(false)

	mockService.On("Get2FAStatus", mock.Anything, userID).Return(expectedStatus, nil)

	c, w := setupTestContext("GET", "/api/v1/2fa/status", nil)
	setUserContext(c, userID)

	handler.Get2FAStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.False(t, data["enabled"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_Get2FAStatus_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	c, w := setupTestContext("GET", "/api/v1/2fa/status", nil)

	handler.Get2FAStatus(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Get2FAStatus_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("Get2FAStatus", mock.Anything, userID).Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/2fa/status", nil)
	setUserContext(c, userID)

	handler.Get2FAStatus(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_Get2FAStatus_AppError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("Get2FAStatus", mock.Anything, userID).
		Return(nil, common.NewInternalServerError("failed to get 2FA status"))

	c, w := setupTestContext("GET", "/api/v1/2fa/status", nil)
	setUserContext(c, userID)

	handler.Get2FAStatus(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// RegenerateBackupCodes Handler Tests
// ============================================================================

func TestHandler_RegenerateBackupCodes_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	expectedCodes := []string{"ABCD-1234", "EFGH-5678", "IJKL-9012", "MNOP-3456"}

	mockService.On("RegenerateBackupCodes", mock.Anything, userID, "123456", mock.Anything, mock.Anything).Return(expectedCodes, nil)

	reqBody := map[string]interface{}{
		"otp": "123456",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/backup-codes/regenerate", reqBody)
	setUserContext(c, userID)

	handler.RegenerateBackupCodes(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["backup_codes"])
	mockService.AssertExpectations(t)
}

func TestHandler_RegenerateBackupCodes_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	reqBody := map[string]interface{}{
		"otp": "123456",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/backup-codes/regenerate", reqBody)

	handler.RegenerateBackupCodes(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_RegenerateBackupCodes_MissingOTP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	reqBody := map[string]interface{}{}

	c, w := setupTestContext("POST", "/api/v1/2fa/backup-codes/regenerate", reqBody)
	setUserContext(c, userID)

	handler.RegenerateBackupCodes(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RegenerateBackupCodes_InvalidOTPLength(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"otp": "12345", // Too short
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/backup-codes/regenerate", reqBody)
	setUserContext(c, userID)

	handler.RegenerateBackupCodes(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RegenerateBackupCodes_InvalidOTP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("RegenerateBackupCodes", mock.Anything, userID, "000000", mock.Anything, mock.Anything).
		Return(nil, common.NewBadRequestError("invalid OTP", nil))

	reqBody := map[string]interface{}{
		"otp": "000000",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/backup-codes/regenerate", reqBody)
	setUserContext(c, userID)

	handler.RegenerateBackupCodes(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_RegenerateBackupCodes_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("RegenerateBackupCodes", mock.Anything, userID, "123456", mock.Anything, mock.Anything).
		Return(nil, errors.New("database error"))

	reqBody := map[string]interface{}{
		"otp": "123456",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/backup-codes/regenerate", reqBody)
	setUserContext(c, userID)

	handler.RegenerateBackupCodes(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// GetTrustedDevices Handler Tests
// ============================================================================

func TestHandler_GetTrustedDevices_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	expectedDevices := createTestTrustedDevices()

	mockService.On("GetTrustedDevices", mock.Anything, userID, "").Return(expectedDevices, nil)

	c, w := setupTestContext("GET", "/api/v1/2fa/devices", nil)
	setUserContext(c, userID)

	handler.GetTrustedDevices(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["devices"])
	mockService.AssertExpectations(t)
}

func TestHandler_GetTrustedDevices_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("GetTrustedDevices", mock.Anything, userID, "").Return([]*TrustedDeviceResponse{}, nil)

	c, w := setupTestContext("GET", "/api/v1/2fa/devices", nil)
	setUserContext(c, userID)

	handler.GetTrustedDevices(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_GetTrustedDevices_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	c, w := setupTestContext("GET", "/api/v1/2fa/devices", nil)

	handler.GetTrustedDevices(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetTrustedDevices_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("GetTrustedDevices", mock.Anything, userID, "").Return(nil, errors.New("database error"))

	c, w := setupTestContext("GET", "/api/v1/2fa/devices", nil)
	setUserContext(c, userID)

	handler.GetTrustedDevices(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// RevokeTrustedDevice Handler Tests
// ============================================================================

func TestHandler_RevokeTrustedDevice_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	deviceID := uuid.New()

	mockService.On("RevokeTrustedDevice", mock.Anything, userID, deviceID, mock.Anything, mock.Anything).Return(nil)

	c, w := setupTestContext("DELETE", "/api/v1/2fa/devices/"+deviceID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: deviceID.String()}}
	setUserContext(c, userID)

	handler.RevokeTrustedDevice(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.True(t, data["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_RevokeTrustedDevice_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	deviceID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/2fa/devices/"+deviceID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: deviceID.String()}}

	handler.RevokeTrustedDevice(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_RevokeTrustedDevice_InvalidDeviceID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/2fa/devices/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	setUserContext(c, userID)

	handler.RevokeTrustedDevice(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "invalid device ID")
}

func TestHandler_RevokeTrustedDevice_EmptyDeviceID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	c, w := setupTestContext("DELETE", "/api/v1/2fa/devices/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	setUserContext(c, userID)

	handler.RevokeTrustedDevice(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RevokeTrustedDevice_DeviceNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	deviceID := uuid.New()

	mockService.On("RevokeTrustedDevice", mock.Anything, userID, deviceID, mock.Anything, mock.Anything).
		Return(common.NewNotFoundError("device not found", nil))

	c, w := setupTestContext("DELETE", "/api/v1/2fa/devices/"+deviceID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: deviceID.String()}}
	setUserContext(c, userID)

	handler.RevokeTrustedDevice(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_RevokeTrustedDevice_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	deviceID := uuid.New()

	mockService.On("RevokeTrustedDevice", mock.Anything, userID, deviceID, mock.Anything, mock.Anything).
		Return(errors.New("database error"))

	c, w := setupTestContext("DELETE", "/api/v1/2fa/devices/"+deviceID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: deviceID.String()}}
	setUserContext(c, userID)

	handler.RevokeTrustedDevice(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// SendPhoneVerification Handler Tests
// ============================================================================

func TestHandler_SendPhoneVerification_Success_FromRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	phoneNumber := "+12345678901"
	expectedResponse := createTestSendOTPResponse()

	mockService.On("SendOTP", mock.Anything, userID, phoneNumber, OTPTypePhoneVerification, mock.Anything, mock.Anything).Return(expectedResponse, nil)

	reqBody := map[string]interface{}{
		"phone_number": phoneNumber,
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/phone/send", reqBody)
	setUserContext(c, userID)

	handler.SendPhoneVerification(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SendPhoneVerification_Success_FromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	phoneNumber := "+12345678901"
	expectedResponse := createTestSendOTPResponse()

	mockService.On("SendOTP", mock.Anything, userID, phoneNumber, OTPTypePhoneVerification, mock.Anything, mock.Anything).Return(expectedResponse, nil)

	c, w := setupTestContext("POST", "/api/v1/2fa/phone/send", nil)
	setUserContextWithPhone(c, userID, phoneNumber)

	handler.SendPhoneVerification(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_SendPhoneVerification_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	c, w := setupTestContext("POST", "/api/v1/2fa/phone/send", nil)

	handler.SendPhoneVerification(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_SendPhoneVerification_MissingPhoneNumber(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/2fa/phone/send", nil)
	setUserContext(c, userID)

	handler.SendPhoneVerification(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)
	assert.Contains(t, response["error"].(map[string]interface{})["message"], "phone number required")
}

func TestHandler_SendPhoneVerification_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	phoneNumber := "+12345678901"

	mockService.On("SendOTP", mock.Anything, userID, phoneNumber, OTPTypePhoneVerification, mock.Anything, mock.Anything).
		Return(nil, errors.New("SMS service error"))

	reqBody := map[string]interface{}{
		"phone_number": phoneNumber,
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/phone/send", reqBody)
	setUserContext(c, userID)

	handler.SendPhoneVerification(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// VerifyPhone Handler Tests
// ============================================================================

func TestHandler_VerifyPhone_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("VerifyPhone", mock.Anything, userID, "123456", mock.Anything, mock.Anything).Return(nil)

	reqBody := map[string]interface{}{
		"otp": "123456",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/phone/verify", reqBody)
	setUserContext(c, userID)

	handler.VerifyPhone(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.True(t, data["success"].(bool))
	mockService.AssertExpectations(t)
}

func TestHandler_VerifyPhone_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	reqBody := map[string]interface{}{
		"otp": "123456",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/phone/verify", reqBody)

	handler.VerifyPhone(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_VerifyPhone_MissingOTP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/2fa/phone/verify", nil)
	setUserContext(c, userID)

	handler.VerifyPhone(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_VerifyPhone_InvalidOTP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("VerifyPhone", mock.Anything, userID, "000000", mock.Anything, mock.Anything).
		Return(common.NewBadRequestError("invalid OTP", nil))

	reqBody := map[string]interface{}{
		"otp": "000000",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/phone/verify", reqBody)
	setUserContext(c, userID)

	handler.VerifyPhone(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_VerifyPhone_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("VerifyPhone", mock.Anything, userID, "123456", mock.Anything, mock.Anything).
		Return(errors.New("database error"))

	reqBody := map[string]interface{}{
		"otp": "123456",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/phone/verify", reqBody)
	setUserContext(c, userID)

	handler.VerifyPhone(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// VerifyTOTP Handler Tests
// ============================================================================

func TestHandler_VerifyTOTP_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	mockRepo := new(MockRepository)
	handler := NewTestableHandler(mockService, mockRepo)

	userID := uuid.New()

	mockService.On("VerifyTOTP", mock.Anything, userID, "123456").Return(nil)
	mockRepo.On("SetTOTPVerified", mock.Anything, userID).Return(nil)

	reqBody := map[string]interface{}{
		"code": "123456",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/totp/verify", reqBody)
	setUserContext(c, userID)

	handler.VerifyTOTP(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)
	data := response["data"].(map[string]interface{})
	assert.True(t, data["success"].(bool))
	mockService.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestHandler_VerifyTOTP_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	reqBody := map[string]interface{}{
		"code": "123456",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/totp/verify", reqBody)

	handler.VerifyTOTP(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_VerifyTOTP_MissingCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	c, w := setupTestContext("POST", "/api/v1/2fa/totp/verify", nil)
	setUserContext(c, userID)

	handler.VerifyTOTP(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_VerifyTOTP_InvalidCodeLength(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	tests := []struct {
		name string
		code string
	}{
		{"too short", "12345"},
		{"too long", "1234567"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := map[string]interface{}{
				"code": tt.code,
			}

			c, w := setupTestContext("POST", "/api/v1/2fa/totp/verify", reqBody)
			setUserContext(c, userID)

			handler.VerifyTOTP(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestHandler_VerifyTOTP_InvalidCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("VerifyTOTP", mock.Anything, userID, "000000").
		Return(common.NewBadRequestError("invalid TOTP code", nil))

	reqBody := map[string]interface{}{
		"code": "000000",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/totp/verify", reqBody)
	setUserContext(c, userID)

	handler.VerifyTOTP(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_VerifyTOTP_TOTPNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("VerifyTOTP", mock.Anything, userID, "123456").
		Return(common.NewBadRequestError("TOTP not configured", nil))

	reqBody := map[string]interface{}{
		"code": "123456",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/totp/verify", reqBody)
	setUserContext(c, userID)

	handler.VerifyTOTP(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_VerifyTOTP_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("VerifyTOTP", mock.Anything, userID, "123456").
		Return(errors.New("database error"))

	reqBody := map[string]interface{}{
		"code": "123456",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/totp/verify", reqBody)
	setUserContext(c, userID)

	handler.VerifyTOTP(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertExpectations(t)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestHandler_Enable2FA_ValidationCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           map[string]interface{}
		setupMock      func(*MockTwoFAService, uuid.UUID, string)
		expectedStatus int
	}{
		{
			name: "valid SMS method",
			body: map[string]interface{}{
				"method": "sms",
			},
			setupMock: func(m *MockTwoFAService, userID uuid.UUID, phone string) {
				m.On("Enable2FA", mock.Anything, userID, MethodSMS, phone, mock.Anything, mock.Anything).
					Return(createTestEnable2FAResponse(), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid TOTP method",
			body: map[string]interface{}{
				"method": "totp",
			},
			setupMock: func(m *MockTwoFAService, userID uuid.UUID, phone string) {
				m.On("Enable2FA", mock.Anything, userID, MethodTOTP, phone, mock.Anything, mock.Anything).
					Return(createTestEnable2FAResponse(), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid both method",
			body: map[string]interface{}{
				"method": "both",
			},
			setupMock: func(m *MockTwoFAService, userID uuid.UUID, phone string) {
				m.On("Enable2FA", mock.Anything, userID, MethodBoth, phone, mock.Anything, mock.Anything).
					Return(createTestEnable2FAResponse(), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing method",
			body: map[string]interface{}{},
			setupMock: nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid method",
			body: map[string]interface{}{
				"method": "invalid",
			},
			setupMock: nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockTwoFAService)
			handler := NewTestableHandler(mockService, nil)

			userID := uuid.New()
			phoneNumber := "+12345678901"

			if tt.setupMock != nil {
				tt.setupMock(mockService, userID, phoneNumber)
			}

			c, w := setupTestContext("POST", "/api/v1/2fa/enable", tt.body)
			setUserContextWithPhone(c, userID, phoneNumber)

			handler.Enable2FA(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_AuthorizationCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handlers := []struct {
		name    string
		method  string
		path    string
		body    interface{}
		handler func(*TestableHandler, *gin.Context)
	}{
		{"SendOTP", "POST", "/api/v1/2fa/otp/send", map[string]interface{}{"otp_type": "login"}, (*TestableHandler).SendOTP},
		{"VerifyOTP", "POST", "/api/v1/2fa/otp/verify", VerifyOTPRequest{OTP: "123456", OTPType: OTPTypeLogin}, (*TestableHandler).VerifyOTP},
		{"Enable2FA", "POST", "/api/v1/2fa/enable", Enable2FARequest{Method: MethodSMS}, (*TestableHandler).Enable2FA},
		{"Disable2FA", "POST", "/api/v1/2fa/disable", Disable2FARequest{OTP: "123456"}, (*TestableHandler).Disable2FA},
		{"Get2FAStatus", "GET", "/api/v1/2fa/status", nil, (*TestableHandler).Get2FAStatus},
		{"RegenerateBackupCodes", "POST", "/api/v1/2fa/backup-codes/regenerate", map[string]interface{}{"otp": "123456"}, (*TestableHandler).RegenerateBackupCodes},
		{"GetTrustedDevices", "GET", "/api/v1/2fa/devices", nil, (*TestableHandler).GetTrustedDevices},
		{"SendPhoneVerification", "POST", "/api/v1/2fa/phone/send", map[string]interface{}{"phone_number": "+12345678901"}, (*TestableHandler).SendPhoneVerification},
		{"VerifyPhone", "POST", "/api/v1/2fa/phone/verify", map[string]interface{}{"otp": "123456"}, (*TestableHandler).VerifyPhone},
		{"VerifyTOTP", "POST", "/api/v1/2fa/totp/verify", map[string]interface{}{"code": "123456"}, (*TestableHandler).VerifyTOTP},
	}

	for _, h := range handlers {
		t.Run(h.name+"_Unauthorized", func(t *testing.T) {
			mockService := new(MockTwoFAService)
			handler := NewTestableHandler(mockService, nil)

			c, w := setupTestContext(h.method, h.path, h.body)
			// Don't set user context

			h.handler(handler, c)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestHandler_UUIDFormats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		uuid           string
		expectedStatus int
	}{
		{
			name:           "invalid format - too short",
			uuid:           "123",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid format - wrong characters",
			uuid:           "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty uuid",
			uuid:           "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid uuid - special characters",
			uuid:           "!@#$%^&*()_+-={}|[]",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockTwoFAService)
			handler := NewTestableHandler(mockService, nil)

			userID := uuid.New()

			c, w := setupTestContext("DELETE", "/api/v1/2fa/devices/test", nil)
			c.Params = gin.Params{{Key: "id", Value: tt.uuid}}
			setUserContext(c, userID)

			handler.RevokeTrustedDevice(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// ============================================================================
// Response Format Tests
// ============================================================================

func TestHandler_SendOTP_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	phoneNumber := "+12345678901"
	expectedResponse := createTestSendOTPResponse()

	mockService.On("SendOTP", mock.Anything, userID, phoneNumber, OTPTypeLogin, mock.Anything, mock.Anything).Return(expectedResponse, nil)

	reqBody := map[string]interface{}{
		"otp_type": "login",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/otp/send", reqBody)
	setUserContextWithPhone(c, userID, phoneNumber)

	handler.SendOTP(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["message"])
	assert.NotNil(t, data["destination"])
	assert.NotNil(t, data["expires_at"])
	mockService.AssertExpectations(t)
}

func TestHandler_Get2FAStatus_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	enabledAt := time.Now()
	expectedStatus := &TwoFAStatusResponse{
		Enabled:             true,
		Method:              MethodSMS,
		PhoneVerified:       true,
		TOTPVerified:        false,
		BackupCodesCount:    10,
		TrustedDevicesCount: 2,
		EnabledAt:           &enabledAt,
	}

	mockService.On("Get2FAStatus", mock.Anything, userID).Return(expectedStatus, nil)

	c, w := setupTestContext("GET", "/api/v1/2fa/status", nil)
	setUserContext(c, userID)

	handler.Get2FAStatus(c)

	assert.Equal(t, http.StatusOK, w.Code)
	response := parseResponse(w)

	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	data := response["data"].(map[string]interface{})
	assert.True(t, data["enabled"].(bool))
	assert.Equal(t, "sms", data["method"])
	assert.True(t, data["phone_verified"].(bool))
	assert.False(t, data["totp_verified"].(bool))
	assert.Equal(t, float64(10), data["backup_codes_count"])
	assert.Equal(t, float64(2), data["trusted_devices_count"])
	mockService.AssertExpectations(t)
}

func TestHandler_ErrorResponse_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	mockService.On("VerifyOTP", mock.Anything, userID, "000000", OTPTypeLogin, mock.Anything, mock.Anything).
		Return(common.NewBadRequestError("invalid OTP", nil))

	reqBody := VerifyOTPRequest{
		OTP:     "000000",
		OTPType: OTPTypeLogin,
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/otp/verify", reqBody)
	setUserContext(c, userID)

	handler.VerifyOTP(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	response := parseResponse(w)

	assert.False(t, response["success"].(bool))
	assert.NotNil(t, response["error"])

	errorInfo := response["error"].(map[string]interface{})
	assert.NotNil(t, errorInfo["code"])
	assert.NotNil(t, errorInfo["message"])
	mockService.AssertExpectations(t)
}

// ============================================================================
// Edge Cases and Concurrent Access Tests
// ============================================================================

func TestHandler_ConcurrentOTPRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()
	phoneNumber := "+12345678901"
	expectedResponse := createTestSendOTPResponse()

	mockService.On("SendOTP", mock.Anything, userID, phoneNumber, OTPTypeLogin, mock.Anything, mock.Anything).Return(expectedResponse, nil).Times(5)

	for i := 0; i < 5; i++ {
		reqBody := map[string]interface{}{
			"otp_type": "login",
		}

		c, w := setupTestContext("POST", "/api/v1/2fa/otp/send", reqBody)
		setUserContextWithPhone(c, userID, phoneNumber)

		handler.SendOTP(c)

		assert.Equal(t, http.StatusOK, w.Code)
	}

	mockService.AssertExpectations(t)
}

func TestHandler_AllMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)

	methods := []TwoFAMethod{MethodSMS, MethodTOTP, MethodBoth}

	for _, method := range methods {
		t.Run(string(method), func(t *testing.T) {
			mockService := new(MockTwoFAService)
			handler := NewTestableHandler(mockService, nil)

			userID := uuid.New()
			phoneNumber := "+12345678901"
			expectedResponse := &Enable2FAResponse{
				Message:     "2FA enabled successfully",
				Method:      string(method),
				BackupCodes: []string{"ABCD-1234"},
			}

			mockService.On("Enable2FA", mock.Anything, userID, method, phoneNumber, mock.Anything, mock.Anything).Return(expectedResponse, nil)

			reqBody := Enable2FARequest{Method: method}

			c, w := setupTestContext("POST", "/api/v1/2fa/enable", reqBody)
			setUserContextWithPhone(c, userID, phoneNumber)

			handler.Enable2FA(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_AllOTPTypesForVerification(t *testing.T) {
	gin.SetMode(gin.TestMode)

	otpTypes := []OTPType{
		OTPTypeLogin,
		OTPTypePhoneVerification,
		OTPTypeEnable2FA,
		OTPTypeDisable2FA,
		OTPTypePasswordReset,
	}

	for _, otpType := range otpTypes {
		t.Run(string(otpType), func(t *testing.T) {
			mockService := new(MockTwoFAService)
			handler := NewTestableHandler(mockService, nil)

			userID := uuid.New()

			mockService.On("VerifyOTP", mock.Anything, userID, "123456", otpType, mock.Anything, mock.Anything).Return(nil)

			reqBody := VerifyOTPRequest{
				OTP:     "123456",
				OTPType: otpType,
			}

			c, w := setupTestContext("POST", "/api/v1/2fa/otp/verify", reqBody)
			setUserContext(c, userID)

			handler.VerifyOTP(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_Disable2FA_WithBothOTPAndBackupCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockTwoFAService)
	handler := NewTestableHandler(mockService, nil)

	userID := uuid.New()

	// When both are provided, OTP takes precedence
	mockService.On("Disable2FA", mock.Anything, userID, "123456", "ABCD-1234", mock.Anything, mock.Anything).Return(nil)

	reqBody := Disable2FARequest{
		OTP:        "123456",
		BackupCode: "ABCD-1234",
	}

	c, w := setupTestContext("POST", "/api/v1/2fa/disable", reqBody)
	setUserContext(c, userID)

	handler.Disable2FA(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}
