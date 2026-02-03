package verification

import (
	"time"

	"github.com/google/uuid"
)

// BackgroundCheckStatus represents the status of a background check
type BackgroundCheckStatus string

const (
	BGCheckStatusPending    BackgroundCheckStatus = "pending"
	BGCheckStatusInProgress BackgroundCheckStatus = "in_progress"
	BGCheckStatusCompleted  BackgroundCheckStatus = "completed"
	BGCheckStatusPassed     BackgroundCheckStatus = "passed"
	BGCheckStatusFailed     BackgroundCheckStatus = "failed"
	BGCheckStatusExpired    BackgroundCheckStatus = "expired"
)

// BackgroundCheckProvider represents the background check provider
type BackgroundCheckProvider string

const (
	ProviderCheckr   BackgroundCheckProvider = "checkr"
	ProviderSterling BackgroundCheckProvider = "sterling"
	ProviderOnfido   BackgroundCheckProvider = "onfido"
	ProviderMock     BackgroundCheckProvider = "mock"
)

// SelfieVerificationStatus represents the status of selfie verification
type SelfieVerificationStatus string

const (
	SelfieStatusPending  SelfieVerificationStatus = "pending"
	SelfieStatusVerified SelfieVerificationStatus = "verified"
	SelfieStatusFailed   SelfieVerificationStatus = "failed"
	SelfieStatusExpired  SelfieVerificationStatus = "expired"
)

// BackgroundCheck represents a driver background check
type BackgroundCheck struct {
	ID              uuid.UUID             `json:"id" db:"id"`
	DriverID        uuid.UUID             `json:"driver_id" db:"driver_id"`
	Provider        BackgroundCheckProvider `json:"provider" db:"provider"`
	ExternalID      *string               `json:"external_id,omitempty" db:"external_id"`
	Status          BackgroundCheckStatus `json:"status" db:"status"`
	CheckType       string                `json:"check_type" db:"check_type"`
	CriminalCheck   *CheckResult          `json:"criminal_check,omitempty"`
	MVRCheck        *CheckResult          `json:"mvr_check,omitempty"`
	SSNTrace        *CheckResult          `json:"ssn_trace,omitempty"`
	ReportURL       *string               `json:"report_url,omitempty" db:"report_url"`
	Notes           *string               `json:"notes,omitempty" db:"notes"`
	FailureReasons  []string              `json:"failure_reasons,omitempty" db:"failure_reasons"`
	ExpiresAt       *time.Time            `json:"expires_at,omitempty" db:"expires_at"`
	StartedAt       *time.Time            `json:"started_at,omitempty" db:"started_at"`
	CompletedAt     *time.Time            `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt       time.Time             `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at" db:"updated_at"`
}

// CheckResult represents the result of a specific background check type
type CheckResult struct {
	Status    string     `json:"status"`
	Details   string     `json:"details,omitempty"`
	CheckedAt *time.Time `json:"checked_at,omitempty"`
}

// SelfieVerification represents a driver selfie verification
type SelfieVerification struct {
	ID               uuid.UUID                `json:"id" db:"id"`
	DriverID         uuid.UUID                `json:"driver_id" db:"driver_id"`
	RideID           *uuid.UUID               `json:"ride_id,omitempty" db:"ride_id"`
	SelfieURL        string                   `json:"selfie_url" db:"selfie_url"`
	ReferencePhotoURL *string                 `json:"reference_photo_url,omitempty" db:"reference_photo_url"`
	Status           SelfieVerificationStatus `json:"status" db:"status"`
	ConfidenceScore  *float64                 `json:"confidence_score,omitempty" db:"confidence_score"`
	MatchResult      *bool                    `json:"match_result,omitempty" db:"match_result"`
	Provider         string                   `json:"provider" db:"provider"`
	ExternalID       *string                  `json:"external_id,omitempty" db:"external_id"`
	Location         *LocationData            `json:"location,omitempty"`
	FailureReason    *string                  `json:"failure_reason,omitempty" db:"failure_reason"`
	VerifiedAt       *time.Time               `json:"verified_at,omitempty" db:"verified_at"`
	ExpiresAt        *time.Time               `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt        time.Time                `json:"created_at" db:"created_at"`
}

// LocationData represents GPS location for selfie
type LocationData struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Accuracy  float64 `json:"accuracy,omitempty"`
}

// DriverVerificationStatus represents the overall verification status
type DriverVerificationStatus struct {
	DriverID                 uuid.UUID              `json:"driver_id"`
	BackgroundCheckStatus    BackgroundCheckStatus  `json:"background_check_status"`
	BackgroundCheckExpiresAt *time.Time             `json:"background_check_expires_at,omitempty"`
	LastSelfieVerification   *SelfieVerification    `json:"last_selfie_verification,omitempty"`
	SelfieVerificationRequired bool                 `json:"selfie_verification_required"`
	IsFullyVerified          bool                   `json:"is_fully_verified"`
	VerificationMessage      string                 `json:"verification_message,omitempty"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// InitiateBackgroundCheckRequest represents a request to start a background check
type InitiateBackgroundCheckRequest struct {
	DriverID  uuid.UUID               `json:"driver_id" binding:"required"`
	Provider  BackgroundCheckProvider `json:"provider,omitempty"`
	CheckType string                  `json:"check_type,omitempty"`
	// Driver info required for background check
	FirstName   string `json:"first_name" binding:"required"`
	LastName    string `json:"last_name" binding:"required"`
	DateOfBirth string `json:"date_of_birth" binding:"required"` // YYYY-MM-DD
	SSN         string `json:"ssn,omitempty"`
	Email       string `json:"email" binding:"required"`
	Phone       string `json:"phone" binding:"required"`
	// Address
	StreetAddress string `json:"street_address" binding:"required"`
	City          string `json:"city" binding:"required"`
	State         string `json:"state" binding:"required"`
	ZipCode       string `json:"zip_code" binding:"required"`
	// Driver's license
	LicenseNumber string `json:"license_number" binding:"required"`
	LicenseState  string `json:"license_state" binding:"required"`
}

// BackgroundCheckResponse represents the response after initiating a check
type BackgroundCheckResponse struct {
	CheckID       uuid.UUID             `json:"check_id"`
	Status        BackgroundCheckStatus `json:"status"`
	Provider      BackgroundCheckProvider `json:"provider"`
	ExternalID    *string               `json:"external_id,omitempty"`
	EstimatedTime string                `json:"estimated_completion,omitempty"`
}

// SubmitSelfieRequest represents a request to verify a selfie
type SubmitSelfieRequest struct {
	DriverID  uuid.UUID    `json:"driver_id" binding:"required"`
	RideID    *uuid.UUID   `json:"ride_id,omitempty"`
	SelfieURL string       `json:"selfie_url" binding:"required"`
	Location  *LocationData `json:"location,omitempty"`
}

// SelfieVerificationResponse represents the response after selfie verification
type SelfieVerificationResponse struct {
	VerificationID  uuid.UUID                `json:"verification_id"`
	Status          SelfieVerificationStatus `json:"status"`
	ConfidenceScore *float64                 `json:"confidence_score,omitempty"`
	MatchResult     *bool                    `json:"match_result,omitempty"`
	Message         string                   `json:"message,omitempty"`
}

// WebhookPayload represents a webhook from background check provider
type WebhookPayload struct {
	Provider   BackgroundCheckProvider `json:"provider"`
	EventType  string                  `json:"event_type"`
	ExternalID string                  `json:"external_id"`
	Status     string                  `json:"status"`
	Data       map[string]interface{}  `json:"data,omitempty"`
	Timestamp  time.Time               `json:"timestamp"`
}
