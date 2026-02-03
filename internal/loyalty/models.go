package loyalty

import (
	"time"

	"github.com/google/uuid"
)

// TierName represents loyalty tier names
type TierName string

const (
	TierBronze   TierName = "bronze"
	TierSilver   TierName = "silver"
	TierGold     TierName = "gold"
	TierPlatinum TierName = "platinum"
	TierDiamond  TierName = "diamond"
)

// TransactionType represents points transaction types
type TransactionType string

const (
	TransactionEarn       TransactionType = "earn"
	TransactionRedeem     TransactionType = "redeem"
	TransactionExpire     TransactionType = "expire"
	TransactionBonus      TransactionType = "bonus"
	TransactionAdjustment TransactionType = "adjustment"
)

// PointSource represents where points came from
type PointSource string

const (
	SourceRide      PointSource = "ride"
	SourceReferral  PointSource = "referral"
	SourcePromo     PointSource = "promo"
	SourcePromotion PointSource = "promotion"
	SourceChallenge PointSource = "challenge"
	SourceBirthday  PointSource = "birthday"
	SourceStreak    PointSource = "streak"
	SourceSignup    PointSource = "signup"
)

// LoyaltyTier represents a loyalty tier configuration
type LoyaltyTier struct {
	ID                  uuid.UUID   `json:"id" db:"id"`
	Name                TierName    `json:"name" db:"name"`
	DisplayName         string      `json:"display_name" db:"display_name"`
	MinPoints           int         `json:"min_points" db:"min_points"`
	MaxPoints           *int        `json:"max_points,omitempty" db:"max_points"`
	Multiplier          float64     `json:"multiplier" db:"multiplier"`
	DiscountPercent     float64     `json:"discount_percent" db:"discount_percent"`
	PrioritySupport     bool        `json:"priority_support" db:"priority_support"`
	FreeCancellations   int         `json:"free_cancellations" db:"free_cancellations"`
	FreeUpgrades        int         `json:"free_upgrades" db:"free_upgrades"`
	AirportLoungeAccess bool        `json:"airport_lounge_access" db:"airport_lounge_access"`
	DedicatedSupport    bool        `json:"dedicated_support_line" db:"dedicated_support_line"`
	IconURL             *string     `json:"icon_url,omitempty" db:"icon_url"`
	ColorHex            *string     `json:"color_hex,omitempty" db:"color_hex"`
	Benefits            []string    `json:"benefits" db:"benefits"`
	IsActive            bool        `json:"is_active" db:"is_active"`
	CreatedAt           time.Time   `json:"created_at" db:"created_at"`
}

// RiderLoyalty represents a rider's loyalty account
type RiderLoyalty struct {
	RiderID               uuid.UUID  `json:"rider_id" db:"rider_id"`
	CurrentTierID         *uuid.UUID `json:"current_tier_id,omitempty" db:"current_tier_id"`
	CurrentTier           *LoyaltyTier `json:"current_tier,omitempty"`
	TotalPoints           int        `json:"total_points" db:"total_points"`
	AvailablePoints       int        `json:"available_points" db:"available_points"`
	LifetimePoints        int        `json:"lifetime_points" db:"lifetime_points"`
	TierPoints            int        `json:"tier_points" db:"tier_points"`
	TierPeriodStart       time.Time  `json:"tier_period_start" db:"tier_period_start"`
	TierPeriodEnd         time.Time  `json:"tier_period_end" db:"tier_period_end"`
	FreeCancellationsUsed int        `json:"free_cancellations_used" db:"free_cancellations_used"`
	FreeUpgradesUsed      int        `json:"free_upgrades_used" db:"free_upgrades_used"`
	StreakDays            int        `json:"streak_days" db:"streak_days"`
	LastRideDate          *time.Time `json:"last_ride_date,omitempty" db:"last_ride_date"`
	JoinedAt              time.Time  `json:"joined_at" db:"joined_at"`
	TierUpgradedAt        *time.Time `json:"tier_upgraded_at,omitempty" db:"tier_upgraded_at"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`
}

// PointsTransaction represents a points transaction
type PointsTransaction struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	RiderID         uuid.UUID       `json:"rider_id" db:"rider_id"`
	TransactionType TransactionType `json:"transaction_type" db:"transaction_type"`
	Points          int             `json:"points" db:"points"`
	BalanceAfter    int             `json:"balance_after" db:"balance_after"`
	Source          PointSource     `json:"source" db:"source"`
	SourceID        *uuid.UUID      `json:"source_id,omitempty" db:"source_id"`
	Description     *string         `json:"description,omitempty" db:"description"`
	ExpiresAt       *time.Time      `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
}

// RiderChallenge represents a rider challenge
type RiderChallenge struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	Name            string     `json:"name" db:"name"`
	Description     *string    `json:"description,omitempty" db:"description"`
	ChallengeType   string     `json:"challenge_type" db:"challenge_type"`
	TargetValue     int        `json:"target_value" db:"target_value"`
	RewardPoints    int        `json:"reward_points" db:"reward_points"`
	RewardType      string     `json:"reward_type" db:"reward_type"`
	RewardValue     *float64   `json:"reward_value,omitempty" db:"reward_value"`
	StartDate       time.Time  `json:"start_date" db:"start_date"`
	EndDate         time.Time  `json:"end_date" db:"end_date"`
	TierRestriction *uuid.UUID `json:"tier_restriction,omitempty" db:"tier_restriction"`
	MaxParticipants *int       `json:"max_participants,omitempty" db:"max_participants"`
	IsActive        bool       `json:"is_active" db:"is_active"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

// ChallengeProgress represents a rider's progress on a challenge
type ChallengeProgress struct {
	ID              uuid.UUID        `json:"id" db:"id"`
	RiderID         uuid.UUID        `json:"rider_id" db:"rider_id"`
	ChallengeID     uuid.UUID        `json:"challenge_id" db:"challenge_id"`
	Challenge       *RiderChallenge  `json:"challenge,omitempty"`
	CurrentValue    int              `json:"current_value" db:"current_value"`
	TargetValue     int              `json:"target_value"`
	ProgressPercent float64          `json:"progress_percent"`
	Completed       bool             `json:"completed" db:"completed"`
	CompletedAt     *time.Time       `json:"completed_at,omitempty" db:"completed_at"`
	RewardClaimed   bool             `json:"reward_claimed" db:"reward_claimed"`
	RewardClaimedAt *time.Time       `json:"reward_claimed_at,omitempty" db:"reward_claimed_at"`
	CreatedAt       time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at" db:"updated_at"`
}

// RewardCatalogItem represents an item in the rewards catalog
type RewardCatalogItem struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	Name                 string     `json:"name" db:"name"`
	Description          *string    `json:"description,omitempty" db:"description"`
	RewardType           string     `json:"reward_type" db:"reward_type"`
	PointsRequired       int        `json:"points_required" db:"points_required"`
	Value                *float64   `json:"value,omitempty" db:"value"`
	PartnerName          *string    `json:"partner_name,omitempty" db:"partner_name"`
	PartnerLogoURL       *string    `json:"partner_logo_url,omitempty" db:"partner_logo_url"`
	ValidDays            int        `json:"valid_days" db:"valid_days"`
	MaxRedemptionsPerUser *int      `json:"max_redemptions_per_user,omitempty" db:"max_redemptions_per_user"`
	TotalAvailable       *int       `json:"total_available,omitempty" db:"total_available"`
	RedeemedCount        int        `json:"redeemed_count" db:"redeemed_count"`
	TierRestriction      *uuid.UUID `json:"tier_restriction,omitempty" db:"tier_restriction"`
	IsActive             bool       `json:"is_active" db:"is_active"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
}

// Redemption represents a points redemption
type Redemption struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	RiderID        uuid.UUID  `json:"rider_id" db:"rider_id"`
	RewardID       uuid.UUID  `json:"reward_id" db:"reward_id"`
	Reward         *RewardCatalogItem `json:"reward,omitempty"`
	PointsSpent    int        `json:"points_spent" db:"points_spent"`
	RedemptionCode string     `json:"redemption_code" db:"redemption_code"`
	Status         string     `json:"status" db:"status"`
	UsedAt         *time.Time `json:"used_at,omitempty" db:"used_at"`
	ExpiresAt      time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// LoyaltyStatusResponse represents the loyalty status for a rider
type LoyaltyStatusResponse struct {
	RiderID           uuid.UUID     `json:"rider_id"`
	CurrentTier       *LoyaltyTier  `json:"current_tier"`
	NextTier          *LoyaltyTier  `json:"next_tier,omitempty"`
	AvailablePoints   int           `json:"available_points"`
	LifetimePoints    int           `json:"lifetime_points"`
	PointsToNextTier  int           `json:"points_to_next_tier"`
	TierProgress      float64       `json:"tier_progress_percent"`
	StreakDays        int           `json:"streak_days"`
	FreeCancellations int           `json:"free_cancellations_remaining"`
	FreeUpgrades      int           `json:"free_upgrades_remaining"`
	TierExpiresAt     time.Time     `json:"tier_expires_at"`
	Benefits          []string      `json:"benefits"`
}

// EarnPointsRequest represents a request to earn points
type EarnPointsRequest struct {
	RiderID     uuid.UUID   `json:"rider_id"`
	Points      int         `json:"points"`
	Source      PointSource `json:"source"`
	SourceID    *uuid.UUID  `json:"source_id,omitempty"`
	Description string      `json:"description,omitempty"`
}

// RedeemPointsRequest represents a request to redeem points
type RedeemPointsRequest struct {
	RiderID  uuid.UUID `json:"rider_id"`
	RewardID uuid.UUID `json:"reward_id"`
}

// RedeemPointsResponse represents the response after redeeming points
type RedeemPointsResponse struct {
	RedemptionID   uuid.UUID  `json:"redemption_id"`
	RedemptionCode string     `json:"redemption_code"`
	PointsSpent    int        `json:"points_spent"`
	BalanceAfter   int        `json:"balance_after"`
	ExpiresAt      time.Time  `json:"expires_at"`
	Instructions   string     `json:"instructions,omitempty"`
}

// PointsHistoryResponse represents points history
type PointsHistoryResponse struct {
	Transactions []PointsTransaction `json:"transactions"`
	Total        int                 `json:"total"`
	Page         int                 `json:"page"`
	PageSize     int                 `json:"page_size"`
}

// ActiveChallengesResponse represents active challenges for a rider
type ActiveChallengesResponse struct {
	Challenges []ChallengeWithProgress `json:"challenges"`
}

// ChallengeWithProgress combines challenge with rider's progress
type ChallengeWithProgress struct {
	Challenge       RiderChallenge `json:"challenge"`
	CurrentValue    int            `json:"current_value"`
	ProgressPercent float64        `json:"progress_percent"`
	Completed       bool           `json:"completed"`
	DaysRemaining   int            `json:"days_remaining"`
}
