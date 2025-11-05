package fraud

import (
	"time"

	"github.com/google/uuid"
)

// FraudAlertLevel represents the severity of a fraud alert
type FraudAlertLevel string

const (
	AlertLevelLow      FraudAlertLevel = "low"
	AlertLevelMedium   FraudAlertLevel = "medium"
	AlertLevelHigh     FraudAlertLevel = "high"
	AlertLevelCritical FraudAlertLevel = "critical"
)

// FraudAlertType represents the type of fraud detected
type FraudAlertType string

const (
	AlertTypePaymentFraud      FraudAlertType = "payment_fraud"
	AlertTypeAccountFraud      FraudAlertType = "account_fraud"
	AlertTypeLocationFraud     FraudAlertType = "location_fraud"
	AlertTypeRideFraud         FraudAlertType = "ride_fraud"
	AlertTypeRatingManipulation FraudAlertType = "rating_manipulation"
	AlertTypePromoAbuse        FraudAlertType = "promo_abuse"
)

// FraudAlertStatus represents the status of a fraud alert
type FraudAlertStatus string

const (
	AlertStatusPending    FraudAlertStatus = "pending"
	AlertStatusInvestigating FraudAlertStatus = "investigating"
	AlertStatusConfirmed  FraudAlertStatus = "confirmed"
	AlertStatusFalsePositive FraudAlertStatus = "false_positive"
	AlertStatusResolved   FraudAlertStatus = "resolved"
)

// FraudAlert represents a fraud detection alert
type FraudAlert struct {
	ID              uuid.UUID        `json:"id"`
	UserID          uuid.UUID        `json:"user_id"`
	AlertType       FraudAlertType   `json:"alert_type"`
	AlertLevel      FraudAlertLevel  `json:"alert_level"`
	Status          FraudAlertStatus `json:"status"`
	Description     string           `json:"description"`
	Details         map[string]interface{} `json:"details"`
	RiskScore       float64          `json:"risk_score"`
	DetectedAt      time.Time        `json:"detected_at"`
	InvestigatedAt  *time.Time       `json:"investigated_at,omitempty"`
	InvestigatedBy  *uuid.UUID       `json:"investigated_by,omitempty"`
	ResolvedAt      *time.Time       `json:"resolved_at,omitempty"`
	Notes           string           `json:"notes,omitempty"`
	ActionTaken     string           `json:"action_taken,omitempty"`
}

// UserRiskProfile represents a user's risk assessment
type UserRiskProfile struct {
	UserID               uuid.UUID `json:"user_id"`
	RiskScore            float64   `json:"risk_score"` // 0-100
	TotalAlerts          int       `json:"total_alerts"`
	CriticalAlerts       int       `json:"critical_alerts"`
	ConfirmedFraudCases  int       `json:"confirmed_fraud_cases"`
	LastAlertAt          *time.Time `json:"last_alert_at,omitempty"`
	AccountSuspended     bool      `json:"account_suspended"`
	LastUpdated          time.Time `json:"last_updated"`
}

// FraudPattern represents detected fraud patterns
type FraudPattern struct {
	PatternType     string                 `json:"pattern_type"`
	Description     string                 `json:"description"`
	Occurrences     int                    `json:"occurrences"`
	AffectedUsers   []uuid.UUID            `json:"affected_users"`
	FirstDetected   time.Time              `json:"first_detected"`
	LastDetected    time.Time              `json:"last_detected"`
	Details         map[string]interface{} `json:"details"`
	Severity        FraudAlertLevel        `json:"severity"`
}

// PaymentFraudIndicators represents suspicious payment patterns
type PaymentFraudIndicators struct {
	UserID                  uuid.UUID `json:"user_id"`
	FailedPaymentAttempts   int       `json:"failed_payment_attempts"`
	ChargebackCount         int       `json:"chargeback_count"`
	MultiplePaymentMethods  int       `json:"multiple_payment_methods"`
	SuspiciousTransactions  int       `json:"suspicious_transactions"`
	RapidPaymentChanges     bool      `json:"rapid_payment_changes"`
	RiskScore               float64   `json:"risk_score"`
}

// RideFraudIndicators represents suspicious ride patterns
type RideFraudIndicators struct {
	UserID                uuid.UUID `json:"user_id"`
	ExcessiveCancellations int       `json:"excessive_cancellations"`
	UnusualRidePatterns   bool      `json:"unusual_ride_patterns"`
	FakeGPSDetected       bool      `json:"fake_gps_detected"`
	CollisionWithDriver   bool      `json:"collision_with_driver"` // Driver-rider collusion
	PromoAbuse            bool      `json:"promo_abuse"`
	RiskScore             float64   `json:"risk_score"`
}

// AccountFraudIndicators represents suspicious account activity
type AccountFraudIndicators struct {
	UserID                 uuid.UUID `json:"user_id"`
	MultipleAccountsSameDevice bool   `json:"multiple_accounts_same_device"`
	RapidAccountCreation   bool      `json:"rapid_account_creation"`
	SuspiciousEmailPattern bool      `json:"suspicious_email_pattern"`
	FakePhoneNumber        bool      `json:"fake_phone_number"`
	VPNUsage               bool      `json:"vpn_usage"`
	RiskScore              float64   `json:"risk_score"`
}

// FraudStatistics represents fraud detection metrics
type FraudStatistics struct {
	Period                 string  `json:"period"`
	TotalAlerts            int     `json:"total_alerts"`
	CriticalAlerts         int     `json:"critical_alerts"`
	HighAlerts             int     `json:"high_alerts"`
	MediumAlerts           int     `json:"medium_alerts"`
	LowAlerts              int     `json:"low_alerts"`
	ConfirmedFraudCases    int     `json:"confirmed_fraud_cases"`
	FalsePositives         int     `json:"false_positives"`
	PendingInvestigation   int     `json:"pending_investigation"`
	EstimatedLossPrevented float64 `json:"estimated_loss_prevented"`
	AverageResponseTime    int     `json:"average_response_time_minutes"`
}
