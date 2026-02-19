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
	BonusMultiplier  float64        `json:"bonus_multiplier" db:"surge_multiplier_bonus"`
	PriorityDispatch bool           `json:"priority_dispatch" db:"priority_dispatch"`
	IconURL          *string        `json:"icon_url,omitempty" db:"icon_url"`
	ColorHex         *string        `json:"color_hex,omitempty" db:"color_hex"`
	Benefits         []string       `json:"benefits" db:"benefits"`
	IsActive         bool           `json:"is_active" db:"is_active"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
}

// DriverGamification represents a driver's gamification profile
type DriverGamification struct {
	DriverID             uuid.UUID   `json:"driver_id" db:"driver_id"`
	CurrentTierID        *uuid.UUID  `json:"current_tier_id,omitempty" db:"current_tier_id"`
	CurrentTier          *DriverTier `json:"current_tier,omitempty"`
	TotalPoints          int         `json:"total_points" db:"total_points"`
	WeeklyPoints         int         `json:"weekly_points" db:"weekly_points"`
	MonthlyPoints        int         `json:"monthly_points" db:"monthly_points"`
	CurrentStreak        int         `json:"current_streak" db:"current_streak"`
	LongestStreak        int         `json:"longest_streak" db:"longest_streak"`
	AcceptanceStreak     int         `json:"acceptance_streak" db:"acceptance_streak"`
	FiveStarStreak       int         `json:"five_star_streak" db:"five_star_streak"`
	TotalQuestsCompleted int         `json:"total_quests_completed" db:"total_quests_completed"`
	TotalChallengesWon   int         `json:"total_challenges_won" db:"total_challenges_won"`
	TotalBonusEarned     float64     `json:"total_bonuses_earned" db:"total_bonuses_earned"`
	LastActiveDate       *time.Time  `json:"last_active_date,omitempty" db:"last_active_date"`
	TierEvaluatedAt      *time.Time  `json:"tier_evaluated_at,omitempty" db:"tier_evaluated_at"`
	CreatedAt            time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time   `json:"updated_at" db:"updated_at"`
}

// DriverQuest represents a quest/challenge for drivers
type DriverQuest struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	Name            string     `json:"name" db:"name"`
	Description     *string    `json:"description,omitempty" db:"description"`
	QuestType       QuestType  `json:"quest_type" db:"quest_type"`
	TargetValue     int        `json:"target_value" db:"target_value"`
	TimeWindowHours *int       `json:"time_window_hours,omitempty" db:"time_window_hours"`
	RewardType      string     `json:"reward_type" db:"reward_type"` // bonus, multiplier, badge
	RewardValue     float64    `json:"reward_value" db:"reward_value"`
	StartTime       time.Time  `json:"start_time" db:"start_time"`
	EndTime         time.Time  `json:"end_time" db:"end_time"`
	MinTierID       *uuid.UUID `json:"min_tier_id,omitempty" db:"min_tier_id"`
	RegionID        *uuid.UUID `json:"region_id,omitempty" db:"region_id"`
	CityID          *uuid.UUID `json:"city_id,omitempty" db:"city_id"`
	VehicleType     *string    `json:"vehicle_type,omitempty" db:"vehicle_type"`
	MaxParticipants *int       `json:"max_participants,omitempty" db:"max_participants"`
	CurrentCount    int        `json:"current_participants" db:"current_participants"`
	IsGuaranteed    bool       `json:"is_guaranteed" db:"is_guaranteed"`
	IsActive        bool       `json:"is_active" db:"is_active"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

// DriverQuestProgress represents a driver's progress on a quest
type DriverQuestProgress struct {
	ID              uuid.UUID    `json:"id" db:"id"`
	DriverID        uuid.UUID    `json:"driver_id" db:"driver_id"`
	QuestID         uuid.UUID    `json:"quest_id" db:"quest_id"`
	Quest           *DriverQuest `json:"quest,omitempty"`
	CurrentValue    float64      `json:"current_value" db:"current_value"`
	TargetValue     int          `json:"target_value" db:"target_value"`
	ProgressPercent float64      `json:"progress_percent" db:"progress_percent"`
	Completed       bool         `json:"completed" db:"completed"`
	CompletedAt     *time.Time   `json:"completed_at,omitempty" db:"completed_at"`
	RewardPaid      bool         `json:"reward_paid" db:"reward_paid"`
	RewardPaidAt    *time.Time   `json:"reward_paid_at,omitempty" db:"reward_paid_at"`
	RewardAmount    *float64     `json:"reward_amount,omitempty" db:"reward_amount"`
	StartedAt       time.Time    `json:"started_at" db:"started_at"`
	UpdatedAt       time.Time    `json:"updated_at" db:"updated_at"`
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
	TierName        *string   `json:"tier_name,omitempty"`
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
	CurrentValue    float64     `json:"current_value"`
	ProgressPercent float64     `json:"progress_percent"`
	Completed       bool        `json:"completed"`
	RewardPaid      bool        `json:"reward_paid"`
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
