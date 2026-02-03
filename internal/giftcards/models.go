package giftcards

import (
	"time"

	"github.com/google/uuid"
)

// CardStatus represents gift card lifecycle
type CardStatus string

const (
	CardStatusActive    CardStatus = "active"
	CardStatusRedeemed  CardStatus = "redeemed"   // Fully used
	CardStatusExpired   CardStatus = "expired"
	CardStatusDisabled  CardStatus = "disabled"    // Admin disabled
	CardStatusPending   CardStatus = "pending"     // Payment not yet confirmed
)

// CardType classifies how the card was created
type CardType string

const (
	CardTypePurchased   CardType = "purchased"    // User bought it
	CardTypePromotional CardType = "promotional"  // Marketing giveaway
	CardTypeCorporate   CardType = "corporate"    // Corporate bulk purchase
	CardTypeRefund      CardType = "refund"       // Refund issued as credit
)

// GiftCard represents a prepaid gift card
type GiftCard struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	Code             string     `json:"code" db:"code"`                     // Unique redemption code
	CardType         CardType   `json:"card_type" db:"card_type"`
	Status           CardStatus `json:"status" db:"status"`
	OriginalAmount   float64    `json:"original_amount" db:"original_amount"`
	RemainingAmount  float64    `json:"remaining_amount" db:"remaining_amount"`
	Currency         string     `json:"currency" db:"currency"`
	PurchaserID      *uuid.UUID `json:"purchaser_id,omitempty" db:"purchaser_id"` // Who bought it
	RecipientID      *uuid.UUID `json:"recipient_id,omitempty" db:"recipient_id"` // Who redeemed it
	RecipientEmail   *string    `json:"recipient_email,omitempty" db:"recipient_email"`
	RecipientName    *string    `json:"recipient_name,omitempty" db:"recipient_name"`
	PersonalMessage  *string    `json:"personal_message,omitempty" db:"personal_message"`
	DesignTemplate   *string    `json:"design_template,omitempty" db:"design_template"` // Card visual design
	ExpiresAt        *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	RedeemedAt       *time.Time `json:"redeemed_at,omitempty" db:"redeemed_at"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

// GiftCardTransaction records usage of gift card balance
type GiftCardTransaction struct {
	ID          uuid.UUID `json:"id" db:"id"`
	CardID      uuid.UUID `json:"card_id" db:"card_id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	RideID      *uuid.UUID `json:"ride_id,omitempty" db:"ride_id"`
	Amount      float64   `json:"amount" db:"amount"`           // Amount deducted
	BalanceBefore float64 `json:"balance_before" db:"balance_before"`
	BalanceAfter  float64 `json:"balance_after" db:"balance_after"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// PurchaseGiftCardRequest creates and purchases a gift card
type PurchaseGiftCardRequest struct {
	Amount          float64 `json:"amount" binding:"required,min=5,max=500"`
	Currency        string  `json:"currency"`
	RecipientEmail  *string `json:"recipient_email,omitempty"`
	RecipientName   *string `json:"recipient_name,omitempty"`
	PersonalMessage *string `json:"personal_message,omitempty"`
	DesignTemplate  *string `json:"design_template,omitempty"`
}

// RedeemGiftCardRequest redeems a gift card
type RedeemGiftCardRequest struct {
	Code string `json:"code" binding:"required"`
}

// CheckBalanceResponse shows gift card balance
type CheckBalanceResponse struct {
	Code            string     `json:"code"`
	Status          CardStatus `json:"status"`
	OriginalAmount  float64    `json:"original_amount"`
	RemainingAmount float64    `json:"remaining_amount"`
	Currency        string     `json:"currency"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	IsValid         bool       `json:"is_valid"`
}

// GiftCardSummary shows a user's gift card credits
type GiftCardSummary struct {
	TotalBalance     float64     `json:"total_balance"`
	ActiveCards      int         `json:"active_cards"`
	Cards            []GiftCard  `json:"cards"`
	RecentTransactions []GiftCardTransaction `json:"recent_transactions"`
}

// CreateBulkRequest creates multiple gift cards (admin/corporate)
type CreateBulkRequest struct {
	Count    int      `json:"count" binding:"required,min=1,max=1000"`
	Amount   float64  `json:"amount" binding:"required,min=5"`
	Currency string   `json:"currency"`
	CardType CardType `json:"card_type" binding:"required"`
	ExpiresInDays *int `json:"expires_in_days,omitempty"`
}

// BulkCreateResponse returns created cards
type BulkCreateResponse struct {
	Cards []GiftCard `json:"cards"`
	Count int        `json:"count"`
	Total float64    `json:"total_value"`
}
