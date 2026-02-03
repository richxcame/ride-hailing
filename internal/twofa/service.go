package twofa

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/logger"
	redisclient "github.com/richxcame/ride-hailing/pkg/redis"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// SMSSender interface for sending SMS
type SMSSender interface {
	SendOTP(to, otp string) (string, error)
}

// Service handles 2FA business logic
type Service struct {
	repo       *Repository
	smsSender  SMSSender
	redis      redisclient.ClientInterface
	issuerName string // For TOTP QR codes
}

// NewService creates a new 2FA service
func NewService(repo *Repository, smsSender SMSSender, redis redisclient.ClientInterface, issuerName string) *Service {
	if issuerName == "" {
		issuerName = "RideHailing"
	}
	return &Service{
		repo:       repo,
		smsSender:  smsSender,
		redis:      redis,
		issuerName: issuerName,
	}
}

// ========================================
// OTP OPERATIONS
// ========================================

// SendOTP generates and sends an OTP to the user
func (s *Service) SendOTP(ctx context.Context, userID uuid.UUID, phoneNumber string, otpType OTPType, ipAddress, userAgent *string) (*SendOTPResponse, error) {
	// Check rate limit
	allowed, err := s.repo.CheckRateLimit(ctx, phoneNumber, "phone", 5, 3600)
	if err != nil {
		logger.Warn("Rate limit check failed", zap.Error(err))
	} else if !allowed {
		return nil, common.NewTooManyRequestsError("too many OTP requests, please try again later")
	}

	// Invalidate any existing OTPs of this type
	_ = s.repo.InvalidateUserOTPs(ctx, userID, otpType)

	// Generate OTP
	otp := s.generateOTP()

	// Hash OTP for storage
	otpHash, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		return nil, common.NewInternalServerError("failed to process OTP")
	}

	// Create OTP record
	otpRecord := &OTPVerification{
		ID:             uuid.New(),
		UserID:         userID,
		OTPHash:        string(otpHash),
		OTPType:        otpType,
		DeliveryMethod: DeliverySMS,
		Destination:    phoneNumber,
		Attempts:       0,
		MaxAttempts:    OTPMaxAttempts,
		ExpiresAt:      time.Now().Add(time.Duration(OTPExpiryMinutes) * time.Minute),
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
	}

	if err := s.repo.CreateOTPVerification(ctx, otpRecord); err != nil {
		return nil, common.NewInternalServerError("failed to create OTP")
	}

	// Send OTP via SMS
	if s.smsSender != nil {
		if _, err := s.smsSender.SendOTP(phoneNumber, otp); err != nil {
			logger.Error("Failed to send OTP SMS", zap.Error(err), zap.String("phone", maskPhone(phoneNumber)))
			return nil, common.NewInternalServerError("failed to send OTP")
		}
	}

	// Log the event
	s.logAuditEvent(ctx, userID, "otp_sent", "success", map[string]interface{}{
		"otp_type":    otpType,
		"destination": maskPhone(phoneNumber),
	}, ipAddress, userAgent)

	return &SendOTPResponse{
		Message:     "OTP sent successfully",
		Destination: maskPhone(phoneNumber),
		ExpiresAt:   otpRecord.ExpiresAt,
	}, nil
}

// VerifyOTP verifies an OTP code
func (s *Service) VerifyOTP(ctx context.Context, userID uuid.UUID, otp string, otpType OTPType, ipAddress, userAgent *string) error {
	// Get active OTP
	otpRecord, err := s.repo.GetActiveOTP(ctx, userID, otpType)
	if err != nil {
		s.logAuditEvent(ctx, userID, "otp_verified", "failure", map[string]interface{}{
			"reason": "no active otp",
		}, ipAddress, userAgent)
		return common.NewBadRequestError("no active OTP found or OTP expired", nil)
	}

	// Check attempts
	if otpRecord.Attempts >= otpRecord.MaxAttempts {
		s.logAuditEvent(ctx, userID, "otp_verified", "failure", map[string]interface{}{
			"reason": "max attempts exceeded",
		}, ipAddress, userAgent)
		return common.NewTooManyRequestsError("maximum OTP attempts exceeded")
	}

	// Increment attempts
	_ = s.repo.IncrementOTPAttempts(ctx, otpRecord.ID)

	// Verify OTP
	if err := bcrypt.CompareHashAndPassword([]byte(otpRecord.OTPHash), []byte(otp)); err != nil {
		s.logAuditEvent(ctx, userID, "otp_verified", "failure", map[string]interface{}{
			"reason":   "invalid otp",
			"attempts": otpRecord.Attempts + 1,
		}, ipAddress, userAgent)
		return common.NewBadRequestError("invalid OTP", nil)
	}

	// Mark as verified
	if err := s.repo.MarkOTPVerified(ctx, otpRecord.ID); err != nil {
		return common.NewInternalServerError("failed to verify OTP")
	}

	s.logAuditEvent(ctx, userID, "otp_verified", "success", map[string]interface{}{
		"otp_type": otpType,
	}, ipAddress, userAgent)

	return nil
}

// ========================================
// 2FA ENABLE/DISABLE
// ========================================

// Enable2FA enables 2FA for a user
func (s *Service) Enable2FA(ctx context.Context, userID uuid.UUID, method TwoFAMethod, phoneNumber string, ipAddress, userAgent *string) (*Enable2FAResponse, error) {
	// Check if already enabled
	enabled, _, _, _, _, err := s.repo.Get2FASettings(ctx, userID)
	if err != nil {
		return nil, common.NewInternalServerError("failed to get 2FA settings")
	}
	if enabled {
		return nil, common.NewBadRequestError("2FA is already enabled", nil)
	}

	response := &Enable2FAResponse{
		Method: string(method),
	}

	var totpSecret string

	// Generate TOTP secret if needed
	if method == MethodTOTP || method == MethodBoth {
		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      s.issuerName,
			AccountName: phoneNumber, // Or email
			SecretSize:  TOTPSecretLength,
		})
		if err != nil {
			return nil, common.NewInternalServerError("failed to generate TOTP secret")
		}
		totpSecret = key.Secret()
		response.TOTPSecret = totpSecret
		response.TOTPQRCode = key.URL()
	}

	// Generate backup codes
	backupCodes, backupCodeHashes, err := s.generateBackupCodes()
	if err != nil {
		return nil, common.NewInternalServerError("failed to generate backup codes")
	}

	// Enable 2FA in database
	if err := s.repo.Enable2FA(ctx, userID, method, totpSecret); err != nil {
		return nil, common.NewInternalServerError("failed to enable 2FA")
	}

	// Store backup codes
	if err := s.repo.CreateBackupCodes(ctx, userID, backupCodeHashes); err != nil {
		return nil, common.NewInternalServerError("failed to store backup codes")
	}

	response.BackupCodes = backupCodes
	response.Message = "2FA enabled successfully"

	// If SMS method, send verification OTP
	if method == MethodSMS || method == MethodBoth {
		response.RequiresOTP = true
		response.OTPDestination = maskPhone(phoneNumber)
		// Send OTP for phone verification
		_, _ = s.SendOTP(ctx, userID, phoneNumber, OTPTypeEnable2FA, ipAddress, userAgent)
	}

	s.logAuditEvent(ctx, userID, "2fa_enabled", "success", map[string]interface{}{
		"method": method,
	}, ipAddress, userAgent)

	return response, nil
}

// Disable2FA disables 2FA for a user
func (s *Service) Disable2FA(ctx context.Context, userID uuid.UUID, otp, backupCode string, ipAddress, userAgent *string) error {
	// Verify either OTP or backup code
	verified := false

	if otp != "" {
		if err := s.VerifyOTP(ctx, userID, otp, OTPTypeDisable2FA, ipAddress, userAgent); err == nil {
			verified = true
		}
	}

	if !verified && backupCode != "" {
		if err := s.VerifyBackupCode(ctx, userID, backupCode); err == nil {
			verified = true
		}
	}

	if !verified {
		s.logAuditEvent(ctx, userID, "2fa_disabled", "failure", map[string]interface{}{
			"reason": "invalid verification",
		}, ipAddress, userAgent)
		return common.NewBadRequestError("invalid OTP or backup code", nil)
	}

	// Disable 2FA
	if err := s.repo.Disable2FA(ctx, userID); err != nil {
		return common.NewInternalServerError("failed to disable 2FA")
	}

	// Revoke all trusted devices
	_ = s.repo.RevokeAllUserTrustedDevices(ctx, userID, "2FA disabled")

	s.logAuditEvent(ctx, userID, "2fa_disabled", "success", nil, ipAddress, userAgent)

	return nil
}

// Get2FAStatus gets the 2FA status for a user
func (s *Service) Get2FAStatus(ctx context.Context, userID uuid.UUID) (*TwoFAStatusResponse, error) {
	enabled, method, phoneVerified, totpVerified, enabledAt, err := s.repo.Get2FASettings(ctx, userID)
	if err != nil {
		return nil, common.NewInternalServerError("failed to get 2FA status")
	}

	backupCount, _ := s.repo.CountUnusedBackupCodes(ctx, userID)
	deviceCount, _ := s.repo.CountUserTrustedDevices(ctx, userID)

	return &TwoFAStatusResponse{
		Enabled:             enabled,
		Method:              method,
		PhoneVerified:       phoneVerified != nil,
		TOTPVerified:        totpVerified != nil,
		BackupCodesCount:    backupCount,
		TrustedDevicesCount: deviceCount,
		EnabledAt:           enabledAt,
	}, nil
}

// ========================================
// BACKUP CODES
// ========================================

// VerifyBackupCode verifies and uses a backup code
func (s *Service) VerifyBackupCode(ctx context.Context, userID uuid.UUID, code string) error {
	codes, err := s.repo.GetUnusedBackupCodes(ctx, userID)
	if err != nil {
		return common.NewInternalServerError("failed to get backup codes")
	}

	for _, bc := range codes {
		if err := bcrypt.CompareHashAndPassword([]byte(bc.CodeHash), []byte(code)); err == nil {
			// Code matches, mark as used
			if err := s.repo.UseBackupCode(ctx, bc.ID); err != nil {
				return common.NewInternalServerError("failed to use backup code")
			}
			return nil
		}
	}

	return common.NewBadRequestError("invalid backup code", nil)
}

// RegenerateBackupCodes generates new backup codes
func (s *Service) RegenerateBackupCodes(ctx context.Context, userID uuid.UUID, otp string, ipAddress, userAgent *string) ([]string, error) {
	// Verify OTP first
	if err := s.VerifyOTP(ctx, userID, otp, OTPTypeEnable2FA, ipAddress, userAgent); err != nil {
		return nil, err
	}

	codes, hashes, err := s.generateBackupCodes()
	if err != nil {
		return nil, common.NewInternalServerError("failed to generate backup codes")
	}

	if err := s.repo.CreateBackupCodes(ctx, userID, hashes); err != nil {
		return nil, common.NewInternalServerError("failed to store backup codes")
	}

	s.logAuditEvent(ctx, userID, "backup_codes_regenerated", "success", nil, ipAddress, userAgent)

	return codes, nil
}

// ========================================
// TOTP VERIFICATION
// ========================================

// VerifyTOTP verifies a TOTP code
func (s *Service) VerifyTOTP(ctx context.Context, userID uuid.UUID, code string) error {
	secret, err := s.repo.GetTOTPSecret(ctx, userID)
	if err != nil || secret == "" {
		return common.NewBadRequestError("TOTP not configured", nil)
	}

	valid := totp.Validate(code, secret)
	if !valid {
		return common.NewBadRequestError("invalid TOTP code", nil)
	}

	return nil
}

// ========================================
// TRUSTED DEVICES
// ========================================

// TrustDevice creates a trusted device entry
func (s *Service) TrustDevice(ctx context.Context, userID uuid.UUID, deviceName string, ipAddress, userAgent, fingerprint *string) (string, error) {
	// Generate device token
	token := s.generateSecureToken(32)

	name := "Unknown Device"
	if deviceName != "" {
		name = deviceName
	} else if userAgent != nil {
		name = parseDeviceName(*userAgent)
	}

	device := &TrustedDevice{
		ID:                uuid.New(),
		UserID:            userID,
		DeviceToken:       token,
		DeviceFingerprint: fingerprint,
		DeviceName:        &name,
		TrustedAt:         time.Now(),
		ExpiresAt:         time.Now().AddDate(0, 0, TrustedDeviceDays),
		IPAddress:         ipAddress,
		UserAgent:         userAgent,
	}

	if err := s.repo.CreateTrustedDevice(ctx, device); err != nil {
		return "", common.NewInternalServerError("failed to trust device")
	}

	s.logAuditEvent(ctx, userID, "device_trusted", "success", map[string]interface{}{
		"device_name": name,
	}, ipAddress, userAgent)

	return token, nil
}

// ValidateTrustedDevice validates a trusted device token
func (s *Service) ValidateTrustedDevice(ctx context.Context, userID uuid.UUID, token string) (bool, error) {
	device, err := s.repo.GetTrustedDevice(ctx, token)
	if err != nil {
		return false, nil // Device not found or expired
	}

	if device.UserID != userID {
		return false, nil // Device belongs to different user
	}

	// Update last used
	_ = s.repo.UpdateTrustedDeviceLastUsed(ctx, device.ID)

	return true, nil
}

// GetTrustedDevices gets all trusted devices for a user
func (s *Service) GetTrustedDevices(ctx context.Context, userID uuid.UUID, currentToken string) ([]*TrustedDeviceResponse, error) {
	devices, err := s.repo.GetUserTrustedDevices(ctx, userID)
	if err != nil {
		return nil, common.NewInternalServerError("failed to get trusted devices")
	}

	var response []*TrustedDeviceResponse
	for _, d := range devices {
		name := "Unknown Device"
		if d.DeviceName != nil {
			name = *d.DeviceName
		}

		var ip string
		if d.IPAddress != nil {
			ip = *d.IPAddress
		}

		response = append(response, &TrustedDeviceResponse{
			ID:         d.ID,
			DeviceName: name,
			TrustedAt:  d.TrustedAt,
			LastUsedAt: d.LastUsedAt,
			IPAddress:  ip,
			IsCurrent:  d.DeviceToken == currentToken,
		})
	}

	return response, nil
}

// RevokeTrustedDevice revokes a trusted device
func (s *Service) RevokeTrustedDevice(ctx context.Context, userID, deviceID uuid.UUID, ipAddress, userAgent *string) error {
	if err := s.repo.RevokeTrustedDevice(ctx, deviceID, "user revoked"); err != nil {
		return common.NewInternalServerError("failed to revoke device")
	}

	s.logAuditEvent(ctx, userID, "device_revoked", "success", map[string]interface{}{
		"device_id": deviceID,
	}, ipAddress, userAgent)

	return nil
}

// ========================================
// PENDING LOGIN
// ========================================

// CreatePendingLogin creates a pending 2FA login session
func (s *Service) CreatePendingLogin(ctx context.Context, userID uuid.UUID, ipAddress, userAgent *string) (*PendingLogin, error) {
	pending := &PendingLogin{
		ID:           uuid.New(),
		UserID:       userID,
		SessionToken: s.generateSecureToken(32),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		ExpiresAt:    time.Now().Add(time.Duration(PendingLoginExpiryMin) * time.Minute),
	}

	if err := s.repo.CreatePendingLogin(ctx, pending); err != nil {
		return nil, common.NewInternalServerError("failed to create pending login")
	}

	return pending, nil
}

// ValidatePendingLogin validates and retrieves a pending login
func (s *Service) ValidatePendingLogin(ctx context.Context, sessionToken string) (*PendingLogin, error) {
	pending, err := s.repo.GetPendingLogin(ctx, sessionToken)
	if err != nil {
		return nil, common.NewBadRequestError("invalid or expired session", nil)
	}
	return pending, nil
}

// CompletePendingLogin marks a pending login as completed
func (s *Service) CompletePendingLogin(ctx context.Context, sessionToken string) error {
	return s.repo.CompletePendingLogin(ctx, sessionToken)
}

// ========================================
// PHONE VERIFICATION
// ========================================

// VerifyPhone verifies a user's phone number
func (s *Service) VerifyPhone(ctx context.Context, userID uuid.UUID, otp string, ipAddress, userAgent *string) error {
	if err := s.VerifyOTP(ctx, userID, otp, OTPTypePhoneVerification, ipAddress, userAgent); err != nil {
		return err
	}

	if err := s.repo.SetPhoneVerified(ctx, userID); err != nil {
		return common.NewInternalServerError("failed to verify phone")
	}

	return nil
}

// ========================================
// HELPER FUNCTIONS
// ========================================

// generateOTP generates a random 6-digit OTP
func (s *Service) generateOTP() string {
	const digits = "0123456789"
	otp := make([]byte, OTPLength)
	for i := 0; i < OTPLength; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		otp[i] = digits[n.Int64()]
	}
	return string(otp)
}

// generateSecureToken generates a secure random token
func (s *Service) generateSecureToken(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes)
}

// generateBackupCodes generates backup codes and their hashes
func (s *Service) generateBackupCodes() ([]string, []string, error) {
	codes := make([]string, BackupCodesCount)
	hashes := make([]string, BackupCodesCount)

	for i := 0; i < BackupCodesCount; i++ {
		code := s.generateBackupCode()
		codes[i] = code

		hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		if err != nil {
			return nil, nil, err
		}
		hashes[i] = string(hash)
	}

	return codes, hashes, nil
}

// generateBackupCode generates a single backup code
func (s *Service) generateBackupCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Exclude confusing characters
	code := make([]byte, BackupCodeLength)
	for i := 0; i < BackupCodeLength; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		code[i] = chars[n.Int64()]
	}
	// Format as XXXX-XXXX
	return string(code[:4]) + "-" + string(code[4:])
}

// maskPhone masks a phone number for display
func maskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return phone[:len(phone)-4] + "****"
}

// parseDeviceName extracts device name from user agent
func parseDeviceName(userAgent string) string {
	// Simple parsing - in production use a proper UA parser
	ua := strings.ToLower(userAgent)
	if strings.Contains(ua, "iphone") {
		return "iPhone"
	} else if strings.Contains(ua, "android") {
		return "Android Device"
	} else if strings.Contains(ua, "chrome") {
		if strings.Contains(ua, "mac") {
			return "Chrome on Mac"
		} else if strings.Contains(ua, "windows") {
			return "Chrome on Windows"
		}
		return "Chrome Browser"
	} else if strings.Contains(ua, "safari") {
		return "Safari Browser"
	} else if strings.Contains(ua, "firefox") {
		return "Firefox Browser"
	}
	return "Unknown Device"
}

// logAuditEvent logs an authentication event
func (s *Service) logAuditEvent(ctx context.Context, userID uuid.UUID, eventType, status string, details map[string]interface{}, ipAddress, userAgent *string) {
	log := &AuthAuditLog{
		ID:           uuid.New(),
		UserID:       &userID,
		EventType:    eventType,
		EventStatus:  status,
		EventDetails: details,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	}

	if err := s.repo.CreateAuditLog(ctx, log); err != nil {
		logger.Warn("Failed to create audit log", zap.Error(err))
	}
}
