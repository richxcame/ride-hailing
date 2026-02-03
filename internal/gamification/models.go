package gamification

import (
	"time"

	"github.com/google/uuid"
)

// DriverTierName represents driver tier names
type DriverTierName string

const (
	DriverTierNew       DriverTierName = "new"
	DriverTierBronze    DriverTierName = "bronze"
	DriverTierSilver    DriverTierName = "silver"
	DriverTierGold      DriverTierName = "gold"
	DriverTierPlatinum  DriverTierName = "platinum"
	DriverTierDiamond   DriverTierName = "diamond"
)

// QuestType represents types of driver quests
type QuestType string

const (
	QuestTypeRideCount    QuestType = "ride_count"
	QuestTypeEarnings     QuestType = "earnings"
	QuestTypeRating       QuestType = "rating"
	QuestTypeStreak       QuestType = "streak"
	QuestTypeTimeOnline   QuestType = "time_online"
	QuestTypePeakHours    QuestType = "peak_hours"
	QuestTypeLocation     QuestType = "location"
	QuestTypeAcceptRate   QuestType = "accept_rate"
	QuestTypeNewRiders    QuestType = "new_riders"
)

// QuestStatus represents the status of a quest
type QuestStatus string

const (
	QuestStatusActive    QuestStatus = "active"
	QuestStatusCompleted QuestStatus = "completed"
	QuestStatusExpired   QuestStatus = "expired"
	QuestStatusClaimed   QuestStatus = "claimed"
)

// AchievementCategory represents achievement categories
type AchievementCategory string

const (
	AchievementCategoryMilestone AchievementCategory = "milestone"
	AchievementCategoryRating    AchievementCategory = "rating"
	AchievementCategoryService   AchievementCategory = "service"
	AchievementCategorySpecial   AchievementCategory = "special"
)

// DriverTier represents a driver tier configuration
type DriverTier struct {
	ID               uuid.UUID      `json:"id" db:"id"`
	Name             DriverTierName `json:"name" db:"name"`
	DisplayName      string         `json:"display_name" db:"display_name"`
	MinRides         int            `json:"min_rides" db:"min_rides"`
	MinRating        float64        `json:"min_rating" db:"min_rating"`
	CommissionRate   float64        `json:"commission_rate" db:"commission_rate"`
	BonusMultiplier  float64        `json:"bonus_multiplier" db:"bonus_multiplier"`
	PriorityDispatch bool           `json:"priority_dispatch" db:"priority_dispatch"`
	IconURL          *string        `json:"icon_url,omitempty" db:"icon_url"`
	ColorHex         *string        `json:"color_hex,omitempty" db:"color_hex"`
	Benefits         []string       `json:"benefits" db:"benefits"`
	IsActive         bool           `json:"is_active" db:"is_active"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
}

// DriverGamification represents a driver's gamification profile
type DriverGamification struct {
	DriverID               uuid.UUID    `json:"driver_id" db:"driver_id"`
	CurrentTierID          *uuid.UUID   `json:"current_tier_id,omitempty" db:"current_tier_id"`
	CurrentTier            *DriverTier  `json:"current_tier,omitempty"`
	TotalRides             int          `json:"total_rides" db:"total_rides"`
	TotalEarnings          float64      `json:"total_earnings" db:"total_earnings"`
	AverageRating          float64      `json:"average_rating" db:"average_rating"`
	CurrentStreak          int          `json:"current_streak" db:"current_streak"`
	LongestStreak          int          `json:"longest_streak" db:"longest_streak"`
	TotalBonusEarned       float64      `json:"total_bonus_earned" db:"total_bonus_earned"`
	AchievementPoints      int          `json:"achievement_points" db:"achievement_points"`
	QuestsCompleted        int          `json:"quests_completed" db:"quests_completed"`
	LastActiveDate         *time.Time   `json:"last_active_date,omitempty" db:"last_active_date"`
	WeeklyRides            int          `json:"weekly_rides" db:"weekly_rides"`
	WeeklyEarnings         float64      `json:"weekly_earnings" db:"weekly_earnings"`
	MonthlyRides           int          `json:"monthly_rides" db:"monthly_rides"`
	MonthlyEarnings        float64      `json:"monthly_earnings" db:"monthly_earnings"`
	AcceptanceRate         float64      `json:"acceptance_rate" db:"acceptance_rate"`
	CancellationRate       float64      `json:"cancellation_rate" db:"cancellation_rate"`
	TierUpgradedAt         *time.Time   `json:"tier_upgraded_at,omitempty" db:"tier_upgraded_at"`
	CreatedAt              time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time    `json:"updated_at" db:"updated_at"`
}

// DriverQuest represents a quest/challenge for drivers
type DriverQuest struct {
	ID              uuid.UUID      `json:"id" db:"id"`
	Name            string         `json:"name" db:"name"`
	Description     *string        `json:"description,omitempty" db:"description"`
	QuestType       QuestType      `json:"quest_type" db:"quest_type"`
	TargetValue     int            `json:"target_value" db:"target_value"`
	RewardType      string         `json:"reward_type" db:"reward_type"` // bonus, multiplier, badge
	RewardValue     float64        `json:"reward_value" db:"reward_value"`
	StartDate       time.Time      `json:"start_date" db:"start_date"`
	EndDate         time.Time      `json:"end_date" db:"end_date"`
	TierRestriction *uuid.UUID     `json:"tier_restriction,omitempty" db:"tier_restriction"`
	CityRestriction *string        `json:"city_restriction,omitempty" db:"city_restriction"`
	MaxParticipants *int           `json:"max_participants,omitempty" db:"max_participants"`
	CurrentCount    int            `json:"current_count" db:"current_count"`
	IsActive        bool           `json:"is_active" db:"is_active"`
	IsFeatured      bool           `json:"is_featured" db:"is_featured"`
	IconURL         *string        `json:"icon_url,omitempty" db:"icon_url"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
}

// DriverQuestProgress represents a driver's progress on a quest
type DriverQuestProgress struct {
	ID            uuid.UUID    `json:"id" db:"id"`
	DriverID      uuid.UUID    `json:"driver_id" db:"driver_id"`
	QuestID       uuid.UUID    `json:"quest_id" db:"quest_id"`
	Quest         *DriverQuest `json:"quest,omitempty"`
	CurrentValue  int          `json:"current_value" db:"current_value"`
	Status        QuestStatus  `json:"status" db:"status"`
	CompletedAt   *time.Time   `json:"completed_at,omitempty" db:"completed_at"`
	ClaimedAt     *time.Time   `json:"claimed_at,omitempty" db:"claimed_at"`
	RewardAmount  *float64     `json:"reward_amount,omitempty" db:"reward_amount"`
	CreatedAt     time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at" db:"updated_at"`
}

// Achievement represents a driver achievement/badge
type Achievement struct {
	ID          uuid.UUID           `json:"id" db:"id"`
	Name        string              `json:"name" db:"name"`
	Description *string             `json:"description,omitempty" db:"description"`
	Category    AchievementCategory `json:"category" db:"category"`
	IconURL     *string             `json:"icon_url,omitempty" db:"icon_url"`
	Points      int                 `json:"points" db:"points"`
	Criteria    string              `json:"criteria" db:"criteria"` // JSON criteria
	IsSecret    bool                `json:"is_secret" db:"is_secret"`
	Rarity      string              `json:"rarity" db:"rarity"` // common, rare, epic, legendary
	IsActive    bool                `json:"is_active" db:"is_active"`
	CreatedAt   time.Time           `json:"created_at" db:"created_at"`
}

// DriverAchievement represents an achievement earned by a driver
type DriverAchievement struct {
	ID            uuid.UUID    `json:"id" db:"id"`
	DriverID      uuid.UUID    `json:"driver_id" db:"driver_id"`
	AchievementID uuid.UUID    `json:"achievement_id" db:"achievement_id"`
	Achievement   *Achievement `json:"achievement,omitempty"`
	EarnedAt      time.Time    `json:"earned_at" db:"earned_at"`
	NotifiedAt    *time.Time   `json:"notified_at,omitempty" db:"notified_at"`
}

// LeaderboardEntry represents an entry in the leaderboard
type LeaderboardEntry struct {
	Rank            int       `json:"rank"`
	DriverID        uuid.UUID `json:"driver_id"`
	DriverName      string    `json:"driver_name"`
	ProfilePhotoURL *string   `json:"profile_photo_url,omitempty"`
	TierName        string    `json:"tier_name"`
	Value           float64   `json:"value"` // Rides, earnings, or rating
	IsCurrentDriver bool      `json:"is_current_driver,omitempty"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// GamificationStatusResponse represents a driver's gamification overview
type GamificationStatusResponse struct {
	Profile           *DriverGamification    `json:"profile"`
	CurrentTier       *DriverTier            `json:"current_tier"`
	NextTier          *DriverTier            `json:"next_tier,omitempty"`
	TierProgress      float64                `json:"tier_progress_percent"`
	RidesToNextTier   int                    `json:"rides_to_next_tier"`
	ActiveQuests      []QuestWithProgress    `json:"active_quests"`
	RecentAchievements []DriverAchievement   `json:"recent_achievements"`
	WeeklyStats       *WeeklyStats           `json:"weekly_stats"`
}

// QuestWithProgress combines quest with driver's progress
type QuestWithProgress struct {
	Quest           DriverQuest `json:"quest"`
	CurrentValue    int         `json:"current_value"`
	ProgressPercent float64     `json:"progress_percent"`
	Status          QuestStatus `json:"status"`
	DaysRemaining   int         `json:"days_remaining"`
}

// WeeklyStats represents weekly driver statistics
type WeeklyStats struct {
	Rides         int     `json:"rides"`
	Earnings      float64 `json:"earnings"`
	BonusEarned   float64 `json:"bonus_earned"`
	AverageRating float64 `json:"average_rating"`
	OnlineHours   float64 `json:"online_hours"`
	AcceptRate    float64 `json:"accept_rate"`
}

// LeaderboardResponse represents the leaderboard
type LeaderboardResponse struct {
	Period         string             `json:"period"` // daily, weekly, monthly
	Category       string             `json:"category"` // rides, earnings, rating
	Entries        []LeaderboardEntry `json:"entries"`
	DriverPosition *LeaderboardEntry  `json:"driver_position,omitempty"`
}

// ClaimQuestRewardRequest represents a request to claim a quest reward
type ClaimQuestRewardRequest struct {
	DriverID uuid.UUID `json:"driver_id"`
	QuestID  uuid.UUID `json:"quest_id"`
}

// ClaimQuestRewardResponse represents the response after claiming a reward
type ClaimQuestRewardResponse struct {
	Success      bool     `json:"success"`
	RewardType   string   `json:"reward_type"`
	RewardAmount float64  `json:"reward_amount"`
	Message      string   `json:"message"`
}

// UpdateProgressRequest represents an event that updates quest/achievement progress
type UpdateProgressRequest struct {
	DriverID   uuid.UUID         `json:"driver_id"`
	EventType  string            `json:"event_type"` // ride_completed, rating_received, etc.
	EventData  map[string]interface{} `json:"event_data"`
}
