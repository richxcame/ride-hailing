package paymentmethods

import (
	"time"

	"github.com/google/uuid"
)

// PaymentMethodType represents the type of payment method
type PaymentMethodType string

const (
	PaymentMethodCard       PaymentMethodType = "card"
	PaymentMethodWallet     PaymentMethodType = "wallet"
	PaymentMethodApplePay   PaymentMethodType = "apple_pay"
	PaymentMethodGooglePay  PaymentMethodType = "google_pay"
	PaymentMethodPayPal     PaymentMethodType = "paypal"
	PaymentMethodCash       PaymentMethodType = "cash"
	PaymentMethodBankTransfer PaymentMethodType = "bank_transfer"
	PaymentMethodGiftCard   PaymentMethodType = "gift_card"
)

// CardBrand represents the card brand
type CardBrand string

const (
	CardBrandVisa       CardBrand = "visa"
	CardBrandMastercard CardBrand = "mastercard"
	CardBrandAmex       CardBrand = "amex"
	CardBrandDiscover   CardBrand = "discover"
	CardBrandUnionPay   CardBrand = "unionpay"
	CardBrandOther      CardBrand = "other"
)

// PaymentMethod represents a user's stored payment method
type PaymentMethod struct {
	ID                uuid.UUID         `json:"id" db:"id"`
	UserID            uuid.UUID         `json:"user_id" db:"user_id"`
	Type              PaymentMethodType `json:"type" db:"type"`
	IsDefault         bool              `json:"is_default" db:"is_default"`
	IsActive          bool              `json:"is_active" db:"is_active"`

	// Card details (masked)
	CardBrand         *CardBrand `json:"card_brand,omitempty" db:"card_brand"`
	CardLast4         *string    `json:"card_last4,omitempty" db:"card_last4"`
	CardExpMonth      *int       `json:"card_exp_month,omitempty" db:"card_exp_month"`
	CardExpYear       *int       `json:"card_exp_year,omitempty" db:"card_exp_year"`
	CardHolderName    *string    `json:"card_holder_name,omitempty" db:"card_holder_name"`

	// Wallet details
	WalletBalance     *float64   `json:"wallet_balance,omitempty" db:"wallet_balance"`

	// External provider
	ProviderID        *string    `json:"provider_id,omitempty" db:"provider_id"`         // Stripe payment method ID, PayPal email, etc.
	ProviderType      *string    `json:"provider_type,omitempty" db:"provider_type"`       // stripe, paypal, etc.

	// Metadata
	Nickname          *string    `json:"nickname,omitempty" db:"nickname"`     // "My Visa", "Business Card"
	Currency          string     `json:"currency" db:"currency"`
	Country           *string    `json:"country,omitempty" db:"country"`

	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

// WalletTransaction represents a wallet credit/debit
type WalletTransaction struct {
	ID              uuid.UUID `json:"id" db:"id"`
	UserID          uuid.UUID `json:"user_id" db:"user_id"`
	PaymentMethodID uuid.UUID `json:"payment_method_id" db:"payment_method_id"`
	Type            string    `json:"type" db:"type"` // credit, debit, refund, topup
	Amount          float64   `json:"amount" db:"amount"`
	BalanceBefore   float64   `json:"balance_before" db:"balance_before"`
	BalanceAfter    float64   `json:"balance_after" db:"balance_after"`
	Description     string    `json:"description" db:"description"`
	RideID          *uuid.UUID `json:"ride_id,omitempty" db:"ride_id"`
	ReferenceID     *string   `json:"reference_id,omitempty" db:"reference_id"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// AddCardRequest adds a card payment method
type AddCardRequest struct {
	Token          string  `json:"token" binding:"required"` // Stripe token or equivalent
	Nickname       *string `json:"nickname,omitempty"`
	SetAsDefault   bool    `json:"set_as_default"`
}

// AddWalletPaymentRequest adds a digital wallet (Apple Pay, Google Pay)
type AddWalletPaymentRequest struct {
	Type     PaymentMethodType `json:"type" binding:"required"` // apple_pay, google_pay
	Token    string            `json:"token" binding:"required"`
	Nickname *string           `json:"nickname,omitempty"`
}

// TopUpWalletRequest adds funds to the wallet
type TopUpWalletRequest struct {
	Amount          float64    `json:"amount" binding:"required"`
	SourceMethodID  uuid.UUID  `json:"source_method_id" binding:"required"` // Card to charge
}

// SetDefaultRequest sets the default payment method
type SetDefaultRequest struct {
	PaymentMethodID uuid.UUID `json:"payment_method_id" binding:"required"`
}

// PaymentMethodsResponse returns all user payment methods
type PaymentMethodsResponse struct {
	Methods       []PaymentMethod `json:"methods"`
	DefaultMethod *uuid.UUID      `json:"default_method_id,omitempty"`
	WalletBalance float64         `json:"wallet_balance"`
	Currency      string          `json:"currency"`
}

// WalletSummary shows wallet info
type WalletSummary struct {
	Balance            float64             `json:"balance"`
	Currency           string              `json:"currency"`
	RecentTransactions []WalletTransaction `json:"recent_transactions"`
}
