package twofa

import (
	"time"

	"github.com/google/uuid"
)

// TwoFAMethod represents the 2FA method type
type TwoFAMethod string

const (
	MethodSMS  TwoFAMethod = "sms"
	MethodTOTP TwoFAMethod = "totp"
	MethodBoth TwoFAMethod = "both"
)

// OTPType represents the purpose of an OTP
type OTPType string

const (
	OTPTypeLogin             OTPType = "login"
	OTPTypePhoneVerification OTPType = "phone_verification"
	OTPTypeEnable2FA         OTPType = "enable_2fa"
	OTPTypeDisable2FA        OTPType = "disable_2fa"
	OTPTypePasswordReset     OTPType = "password_reset"
)

// DeliveryMethod represents how OTP is delivered
type DeliveryMethod string

const (
	DeliverySMS   DeliveryMethod = "sms"
	DeliveryEmail DeliveryMethod = "email"
)

// OTPVerification represents an OTP verification record
type OTPVerification struct {
	ID             uuid.UUID      `json:"id" db:"id"`
	UserID         uuid.UUID      `json:"user_id" db:"user_id"`
	OTPHash        string         `json:"-" db:"otp_hash"`
	OTPType        OTPType        `json:"otp_type" db:"otp_type"`
	DeliveryMethod DeliveryMethod `json:"delivery_method" db:"delivery_method"`
	Destination    string         `json:"destination" db:"destination"`
	Attempts       int            `json:"attempts" db:"attempts"`
	MaxAttempts    int            `json:"max_attempts" db:"max_attempts"`
	ExpiresAt      time.Time      `json:"expires_at" db:"expires_at"`
	VerifiedAt     *time.Time     `json:"verified_at" db:"verified_at"`
	IPAddress      *string        `json:"ip_address" db:"ip_address"`
	UserAgent      *string        `json:"user_agent" db:"user_agent"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
}

// BackupCode represents a backup code for 2FA recovery
type BackupCode struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	CodeHash  string     `json:"-" db:"code_hash"`
	UsedAt    *time.Time `json:"used_at" db:"used_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// TrustedDevice represents a device trusted to skip 2FA
type TrustedDevice struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	UserID            uuid.UUID  `json:"user_id" db:"user_id"`
	DeviceToken       string     `json:"-" db:"device_token"`
	DeviceFingerprint *string    `json:"device_fingerprint" db:"device_fingerprint"`
	DeviceName        *string    `json:"device_name" db:"device_name"`
	TrustedAt         time.Time  `json:"trusted_at" db:"trusted_at"`
	ExpiresAt         time.Time  `json:"expires_at" db:"expires_at"`
	LastUsedAt        *time.Time `json:"last_used_at" db:"last_used_at"`
	IPAddress         *string    `json:"ip_address" db:"ip_address"`
	UserAgent         *string    `json:"user_agent" db:"user_agent"`
	RevokedAt         *time.Time `json:"revoked_at" db:"revoked_at"`
	RevokedReason     *string    `json:"revoked_reason" db:"revoked_reason"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
}

// PendingLogin represents a login session waiting for 2FA
type PendingLogin struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	UserID       uuid.UUID  `json:"user_id" db:"user_id"`
	SessionToken string     `json:"-" db:"session_token"`
	IPAddress    *string    `json:"ip_address" db:"ip_address"`
	UserAgent    *string    `json:"user_agent" db:"user_agent"`
	ExpiresAt    time.Time  `json:"expires_at" db:"expires_at"`
	CompletedAt  *time.Time `json:"completed_at" db:"completed_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

// AuthAuditLog represents an authentication audit log entry
type AuthAuditLog struct {
	ID                uuid.UUID              `json:"id" db:"id"`
	UserID            *uuid.UUID             `json:"user_id" db:"user_id"`
	EventType         string                 `json:"event_type" db:"event_type"`
	EventStatus       string                 `json:"event_status" db:"event_status"`
	EventDetails      map[string]interface{} `json:"event_details" db:"event_details"`
	IPAddress         *string                `json:"ip_address" db:"ip_address"`
	UserAgent         *string                `json:"user_agent" db:"user_agent"`
	DeviceFingerprint *string                `json:"device_fingerprint" db:"device_fingerprint"`
	RelatedEntityType *string                `json:"related_entity_type" db:"related_entity_type"`
	RelatedEntityID   *uuid.UUID             `json:"related_entity_id" db:"related_entity_id"`
	CreatedAt         time.Time              `json:"created_at" db:"created_at"`
}

// OTPRateLimit represents rate limiting for OTP requests
type OTPRateLimit struct {
	ID                    uuid.UUID  `json:"id" db:"id"`
	Identifier            string     `json:"identifier" db:"identifier"`
	IdentifierType        string     `json:"identifier_type" db:"identifier_type"`
	RequestCount          int        `json:"request_count" db:"request_count"`
	WindowStart           time.Time  `json:"window_start" db:"window_start"`
	WindowDurationSeconds int        `json:"window_duration_seconds" db:"window_duration_seconds"`
	MaxRequests           int        `json:"max_requests" db:"max_requests"`
	LockedUntil           *time.Time `json:"locked_until" db:"locked_until"`
	LockReason            *string    `json:"lock_reason" db:"lock_reason"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// SendOTPRequest represents a request to send an OTP
type SendOTPRequest struct {
	UserID         uuid.UUID      `json:"user_id"`
	OTPType        OTPType        `json:"otp_type" binding:"required"`
	DeliveryMethod DeliveryMethod `json:"delivery_method" binding:"required"`
}

// VerifyOTPRequest represents a request to verify an OTP
type VerifyOTPRequest struct {
	UserID      uuid.UUID `json:"user_id"`
	OTP         string    `json:"otp" binding:"required,len=6"`
	OTPType     OTPType   `json:"otp_type" binding:"required"`
	TrustDevice bool      `json:"trust_device"`
	DeviceName  string    `json:"device_name"`
}

// SendOTPResponse represents the response after sending an OTP
type SendOTPResponse struct {
	Message       string    `json:"message"`
	Destination   string    `json:"destination"` // Masked destination (e.g., +1***4567)
	ExpiresAt     time.Time `json:"expires_at"`
	RetryAfterSec int       `json:"retry_after_seconds,omitempty"`
}

// VerifyOTPResponse represents the response after verifying an OTP
type VerifyOTPResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	DeviceToken  string `json:"device_token,omitempty"` // If device was trusted
	Token        string `json:"token,omitempty"`        // JWT token on successful login
	BackupCodes  []string `json:"backup_codes,omitempty"` // On 2FA enable
}

// Enable2FARequest represents a request to enable 2FA
type Enable2FARequest struct {
	Method TwoFAMethod `json:"method" binding:"required,oneof=sms totp both"`
}

// Enable2FAResponse represents the response when enabling 2FA
type Enable2FAResponse struct {
	Message       string   `json:"message"`
	Method        string   `json:"method"`
	TOTPSecret    string   `json:"totp_secret,omitempty"`    // Base32 encoded secret
	TOTPQRCode    string   `json:"totp_qr_code,omitempty"`   // QR code data URL
	BackupCodes   []string `json:"backup_codes"`             // One-time backup codes
	RequiresOTP   bool     `json:"requires_otp"`             // Whether OTP verification is needed
	OTPDestination string  `json:"otp_destination,omitempty"` // Masked phone/email
}

// Disable2FARequest represents a request to disable 2FA
type Disable2FARequest struct {
	OTP        string `json:"otp" binding:"required,len=6"`
	BackupCode string `json:"backup_code"` // Alternative to OTP
}

// TwoFALoginRequest represents a login attempt requiring 2FA
type TwoFALoginRequest struct {
	SessionToken string `json:"session_token" binding:"required"`
	OTP          string `json:"otp" binding:"required,len=6"`
	TrustDevice  bool   `json:"trust_device"`
	DeviceName   string `json:"device_name"`
}

// TwoFALoginResponse represents the response for 2FA login
type TwoFALoginResponse struct {
	RequiresTwoFA bool        `json:"requires_2fa"`
	SessionToken  string      `json:"session_token,omitempty"` // Temporary token for 2FA flow
	TwoFAMethod   TwoFAMethod `json:"two_fa_method,omitempty"`
	OTPSent       bool        `json:"otp_sent,omitempty"`
	OTPDestination string     `json:"otp_destination,omitempty"` // Masked phone number
	ExpiresAt     time.Time   `json:"expires_at,omitempty"`
}

// TwoFAStatusResponse represents the 2FA status for a user
type TwoFAStatusResponse struct {
	Enabled           bool        `json:"enabled"`
	Method            TwoFAMethod `json:"method,omitempty"`
	PhoneVerified     bool        `json:"phone_verified"`
	TOTPVerified      bool        `json:"totp_verified"`
	BackupCodesCount  int         `json:"backup_codes_count"`
	TrustedDevicesCount int       `json:"trusted_devices_count"`
	EnabledAt         *time.Time  `json:"enabled_at,omitempty"`
}

// TrustedDeviceResponse represents a trusted device for listing
type TrustedDeviceResponse struct {
	ID         uuid.UUID  `json:"id"`
	DeviceName string     `json:"device_name"`
	TrustedAt  time.Time  `json:"trusted_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	IPAddress  string     `json:"ip_address,omitempty"`
	IsCurrent  bool       `json:"is_current"`
}

// Constants for OTP configuration
const (
	OTPLength             = 6
	OTPExpiryMinutes      = 10
	OTPMaxAttempts        = 5
	PendingLoginExpiryMin = 5
	TrustedDeviceDays     = 30
	BackupCodesCount      = 10
	BackupCodeLength      = 8
	TOTPSecretLength      = 32
)
