package promos

import (
	"time"

	"github.com/google/uuid"
)

// PromoCode represents a promotional discount code
type PromoCode struct {
	ID                uuid.UUID  `json:"id"`
	Code              string     `json:"code"`
	Description       string     `json:"description"`
	DiscountType      string     `json:"discount_type"` // "percentage" or "fixed_amount"
	DiscountValue     float64    `json:"discount_value"`
	MaxDiscountAmount *float64   `json:"max_discount_amount,omitempty"`
	MinRideAmount     *float64   `json:"min_ride_amount,omitempty"`
	MaxUses           *int       `json:"max_uses,omitempty"`
	TotalUses         int        `json:"total_uses"`
	UsesPerUser       int        `json:"uses_per_user"`
	ValidFrom         time.Time  `json:"valid_from"`
	ValidUntil        time.Time  `json:"valid_until"`
	IsActive          bool       `json:"is_active"`
	CreatedBy         *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// PromoCodeUse represents a single use of a promo code
type PromoCodeUse struct {
	ID             uuid.UUID `json:"id"`
	PromoCodeID    uuid.UUID `json:"promo_code_id"`
	UserID         uuid.UUID `json:"user_id"`
	RideID         uuid.UUID `json:"ride_id"`
	DiscountAmount float64   `json:"discount_amount"`
	OriginalAmount float64   `json:"original_amount"`
	FinalAmount    float64   `json:"final_amount"`
	UsedAt         time.Time `json:"used_at"`
}

// ReferralCode represents a user's referral code
type ReferralCode struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	Code           string    `json:"code"`
	TotalReferrals int       `json:"total_referrals"`
	TotalEarnings  float64   `json:"total_earnings"`
	CreatedAt      time.Time `json:"created_at"`
}

// Referral represents a referral relationship
type Referral struct {
	ID                     uuid.UUID  `json:"id"`
	ReferrerID             uuid.UUID  `json:"referrer_id"`
	ReferredID             uuid.UUID  `json:"referred_id"`
	ReferralCodeID         uuid.UUID  `json:"referral_code_id"`
	ReferrerBonus          float64    `json:"referrer_bonus"`
	ReferredBonus          float64    `json:"referred_bonus"`
	ReferrerBonusApplied   bool       `json:"referrer_bonus_applied"`
	ReferredBonusApplied   bool       `json:"referred_bonus_applied"`
	ReferredFirstRideID    *uuid.UUID `json:"referred_first_ride_id,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	CompletedAt            *time.Time `json:"completed_at,omitempty"`
}

// RideType represents different types of rides with different pricing
type RideType struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	BaseFare       float64   `json:"base_fare"`
	PerKmRate      float64   `json:"per_km_rate"`
	PerMinuteRate  float64   `json:"per_minute_rate"`
	MinimumFare    float64   `json:"minimum_fare"`
	Capacity       int       `json:"capacity"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// PromoCodeValidation contains validation result
type PromoCodeValidation struct {
	Valid          bool    `json:"valid"`
	Message        string  `json:"message,omitempty"`
	DiscountAmount float64 `json:"discount_amount"`
	FinalAmount    float64 `json:"final_amount"`
}
