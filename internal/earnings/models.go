package earnings

import (
	"time"

	"github.com/google/uuid"
)

// EarningType represents the type of earning
type EarningType string

const (
	EarningTypeRideFare    EarningType = "ride_fare"
	EarningTypeTip         EarningType = "tip"
	EarningTypeBonus       EarningType = "bonus"
	EarningTypeSurge       EarningType = "surge"
	EarningTypePromo       EarningType = "promo"
	EarningTypeReferral    EarningType = "referral"
	EarningTypeDelivery    EarningType = "delivery"
	EarningTypeWaitTime    EarningType = "wait_time"
	EarningTypeAdjustment  EarningType = "adjustment"
	EarningTypeCancellation EarningType = "cancellation_fee"
)

// PayoutStatus represents the payout status
type PayoutStatus string

const (
	PayoutStatusPending    PayoutStatus = "pending"
	PayoutStatusProcessing PayoutStatus = "processing"
	PayoutStatusCompleted  PayoutStatus = "completed"
	PayoutStatusFailed     PayoutStatus = "failed"
)

// PayoutMethod represents the payout method
type PayoutMethod string

const (
	PayoutMethodBankTransfer PayoutMethod = "bank_transfer"
	PayoutMethodInstantPay   PayoutMethod = "instant_pay"
	PayoutMethodWallet       PayoutMethod = "wallet"
)

// DriverEarning represents a single earning event
type DriverEarning struct {
	ID           uuid.UUID   `json:"id" db:"id"`
	DriverID     uuid.UUID   `json:"driver_id" db:"driver_id"`
	RideID       *uuid.UUID  `json:"ride_id,omitempty" db:"ride_id"`
	DeliveryID   *uuid.UUID  `json:"delivery_id,omitempty" db:"delivery_id"`
	Type         EarningType `json:"type" db:"type"`
	GrossAmount  float64     `json:"gross_amount" db:"gross_amount"`
	Commission   float64     `json:"commission" db:"commission"`
	NetAmount    float64     `json:"net_amount" db:"net_amount"`
	Currency     string      `json:"currency" db:"currency"`
	Description  string      `json:"description" db:"description"`
	IsPaidOut    bool        `json:"is_paid_out" db:"is_paid_out"`
	PayoutID     *uuid.UUID  `json:"payout_id,omitempty" db:"payout_id"`
	CreatedAt    time.Time   `json:"created_at" db:"created_at"`
}

// DriverPayout represents a payout to a driver
type DriverPayout struct {
	ID            uuid.UUID    `json:"id" db:"id"`
	DriverID      uuid.UUID    `json:"driver_id" db:"driver_id"`
	Amount        float64      `json:"amount" db:"amount"`
	Currency      string       `json:"currency" db:"currency"`
	Method        PayoutMethod `json:"method" db:"method"`
	Status        PayoutStatus `json:"status" db:"status"`
	BankAccountID *string      `json:"bank_account_id,omitempty" db:"bank_account_id"`
	Reference     string       `json:"reference" db:"reference"`
	EarningCount  int          `json:"earning_count" db:"earning_count"`
	PeriodStart   time.Time    `json:"period_start" db:"period_start"`
	PeriodEnd     time.Time    `json:"period_end" db:"period_end"`
	ProcessedAt   *time.Time   `json:"processed_at,omitempty" db:"processed_at"`
	FailureReason *string      `json:"failure_reason,omitempty" db:"failure_reason"`
	CreatedAt     time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at" db:"updated_at"`
}

// DriverBankAccount represents a driver's bank account for payouts
type DriverBankAccount struct {
	ID            uuid.UUID `json:"id" db:"id"`
	DriverID      uuid.UUID `json:"driver_id" db:"driver_id"`
	BankName      string    `json:"bank_name" db:"bank_name"`
	AccountHolder string    `json:"account_holder" db:"account_holder"`
	AccountNumber string    `json:"account_number_masked" db:"account_number"` // Stored encrypted
	RoutingNumber *string   `json:"routing_number,omitempty" db:"routing_number"`
	IBAN          *string   `json:"iban,omitempty" db:"iban"`
	SwiftCode     *string   `json:"swift_code,omitempty" db:"swift_code"`
	Currency      string    `json:"currency" db:"currency"`
	IsPrimary     bool      `json:"is_primary" db:"is_primary"`
	IsVerified    bool      `json:"is_verified" db:"is_verified"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// EarningsSummary provides an overview of earnings for a period
type EarningsSummary struct {
	DriverID       uuid.UUID          `json:"driver_id"`
	Period         string             `json:"period"` // today, this_week, this_month, custom
	PeriodStart    time.Time          `json:"period_start"`
	PeriodEnd      time.Time          `json:"period_end"`
	GrossEarnings  float64            `json:"gross_earnings"`
	TotalCommission float64           `json:"total_commission"`
	NetEarnings    float64            `json:"net_earnings"`
	TipEarnings    float64            `json:"tip_earnings"`
	BonusEarnings  float64            `json:"bonus_earnings"`
	SurgeEarnings  float64            `json:"surge_earnings"`
	WaitTimeEarnings float64          `json:"wait_time_earnings"`
	DeliveryEarnings float64          `json:"delivery_earnings"`
	RideCount      int                `json:"ride_count"`
	DeliveryCount  int                `json:"delivery_count"`
	OnlineHours    float64            `json:"online_hours"`
	EarningsPerHour float64           `json:"earnings_per_hour"`
	Currency       string             `json:"currency"`
	Breakdown      []EarningBreakdown `json:"breakdown"`
}

// EarningBreakdown shows earnings grouped by type
type EarningBreakdown struct {
	Type   EarningType `json:"type"`
	Amount float64     `json:"amount"`
	Count  int         `json:"count"`
}

// DailyEarning represents earnings for a single day
type DailyEarning struct {
	Date          string  `json:"date"`
	GrossAmount   float64 `json:"gross_amount"`
	NetAmount     float64 `json:"net_amount"`
	Commission    float64 `json:"commission"`
	Tips          float64 `json:"tips"`
	RideCount     int     `json:"ride_count"`
	OnlineHours   float64 `json:"online_hours"`
}

// EarningsHistoryResponse returns paginated earnings
type EarningsHistoryResponse struct {
	Earnings []DriverEarning `json:"earnings"`
	Total    int             `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

// PayoutHistoryResponse returns paginated payouts
type PayoutHistoryResponse struct {
	Payouts  []DriverPayout `json:"payouts"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

// AddBankAccountRequest adds a bank account for payouts
type AddBankAccountRequest struct {
	BankName      string  `json:"bank_name" binding:"required"`
	AccountHolder string  `json:"account_holder" binding:"required"`
	AccountNumber string  `json:"account_number" binding:"required"`
	RoutingNumber *string `json:"routing_number,omitempty"`
	IBAN          *string `json:"iban,omitempty"`
	SwiftCode     *string `json:"swift_code,omitempty"`
	Currency      string  `json:"currency"`
	IsPrimary     bool    `json:"is_primary"`
}

// RequestPayoutRequest requests an instant payout
type RequestPayoutRequest struct {
	Method        PayoutMethod `json:"method" binding:"required"`
	BankAccountID *uuid.UUID   `json:"bank_account_id,omitempty"`
}

// EarningGoal represents a driver-set earning goal
type EarningGoal struct {
	ID          uuid.UUID `json:"id" db:"id"`
	DriverID    uuid.UUID `json:"driver_id" db:"driver_id"`
	TargetAmount float64  `json:"target_amount" db:"target_amount"`
	Period       string   `json:"period" db:"period"` // daily, weekly, monthly
	CurrentAmount float64 `json:"current_amount" db:"current_amount"`
	Currency     string   `json:"currency" db:"currency"`
	IsActive     bool     `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// SetEarningGoalRequest sets an earning target
type SetEarningGoalRequest struct {
	TargetAmount float64 `json:"target_amount" binding:"required"`
	Period       string  `json:"period" binding:"required"` // daily, weekly, monthly
}

// EarningGoalStatus shows progress toward a goal
type EarningGoalStatus struct {
	Goal           *EarningGoal `json:"goal"`
	CurrentAmount  float64      `json:"current_amount"`
	ProgressPct    float64      `json:"progress_percent"`
	Remaining      float64      `json:"remaining"`
	ProjectedTotal float64      `json:"projected_total"` // Based on current pace
	OnTrack        bool         `json:"on_track"`
}
