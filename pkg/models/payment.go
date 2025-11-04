package models

import (
	"time"

	"github.com/google/uuid"
)

// PaymentStatus represents payment status
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

// PaymentMethod represents payment method
type PaymentMethod string

const (
	PaymentMethodCard   PaymentMethod = "card"
	PaymentMethodWallet PaymentMethod = "wallet"
	PaymentMethodCash   PaymentMethod = "cash"
)

// Payment represents a payment transaction
type Payment struct {
	ID               uuid.UUID              `json:"id" db:"id"`
	RideID           uuid.UUID              `json:"ride_id" db:"ride_id"`
	RiderID          uuid.UUID              `json:"rider_id" db:"rider_id"`
	DriverID         uuid.UUID              `json:"driver_id" db:"driver_id"`
	Amount           float64                `json:"amount" db:"amount"`
	Currency         string                 `json:"currency" db:"currency"`
	PaymentMethod    string                 `json:"payment_method" db:"payment_method"`
	Status           string                 `json:"status" db:"status"`
	StripePaymentID  *string                `json:"stripe_payment_id,omitempty" db:"stripe_payment_id"`
	StripeChargeID   *string                `json:"stripe_charge_id,omitempty" db:"stripe_charge_id"`
	Metadata         map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	Commission       float64                `json:"commission" db:"commission"`
	DriverEarnings   float64                `json:"driver_earnings" db:"driver_earnings"`
	Method           PaymentMethod          `json:"method" db:"method"`
	TransactionID    *string                `json:"transaction_id,omitempty" db:"transaction_id"`
	FailureReason    *string                `json:"failure_reason,omitempty" db:"failure_reason"`
	ProcessedAt      *time.Time             `json:"processed_at,omitempty" db:"processed_at"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
}

// Wallet represents a user's wallet
type Wallet struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Balance   float64   `json:"balance" db:"balance"`
	Currency  string    `json:"currency" db:"currency"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// WalletTransaction represents a wallet transaction
type WalletTransaction struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	WalletID      uuid.UUID  `json:"wallet_id" db:"wallet_id"`
	Type          string     `json:"type" db:"type"` // credit or debit
	Amount        float64    `json:"amount" db:"amount"`
	Description   string     `json:"description" db:"description"`
	ReferenceType string     `json:"reference_type,omitempty" db:"reference_type"` // ride, topup, withdrawal
	ReferenceID   *uuid.UUID `json:"reference_id,omitempty" db:"reference_id"`
	BalanceBefore float64    `json:"balance_before" db:"balance_before"`
	BalanceAfter  float64    `json:"balance_after" db:"balance_after"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
}

// PaymentRequest represents a payment processing request
type PaymentRequest struct {
	RideID uuid.UUID     `json:"ride_id" binding:"required"`
	Method PaymentMethod `json:"method" binding:"required,oneof=card wallet cash"`
	Token  *string       `json:"token,omitempty"` // For card payments
}
