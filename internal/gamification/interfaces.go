package gamification

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryInterface defines the interface for gamification repository operations
type RepositoryInterface interface {
	// Driver Gamification Profile
	GetDriverGamification(ctx context.Context, driverID uuid.UUID) (*DriverGamification, error)
	CreateDriverGamification(ctx context.Context, profile *DriverGamification) error
	UpdateDriverStats(ctx context.Context, driverID uuid.UUID, rides int, earnings float64, rating float64) error
	UpdateDriverStreak(ctx context.Context, driverID uuid.UUID, currentStreak int) error
	UpdateDriverTier(ctx context.Context, driverID uuid.UUID, tierID uuid.UUID) error
	IncrementQuestsCompleted(ctx context.Context, driverID uuid.UUID) error
	AddBonusEarned(ctx context.Context, driverID uuid.UUID, bonus float64) error
	AddAchievementPoints(ctx context.Context, driverID uuid.UUID, points int) error
	ResetWeeklyStats(ctx context.Context) error
	ResetMonthlyStats(ctx context.Context) error

	// Driver Tiers
	GetTier(ctx context.Context, tierID uuid.UUID) (*DriverTier, error)
	GetTierByName(ctx context.Context, name DriverTierName) (*DriverTier, error)
	GetAllTiers(ctx context.Context) ([]*DriverTier, error)

	// Quests
	GetActiveQuests(ctx context.Context, tierID *uuid.UUID, city *string) ([]*DriverQuest, error)
	GetQuest(ctx context.Context, questID uuid.UUID) (*DriverQuest, error)
	GetQuestsByType(ctx context.Context, questType QuestType) ([]*DriverQuest, error)

	// Quest Progress
	GetQuestProgress(ctx context.Context, driverID, questID uuid.UUID) (*DriverQuestProgress, error)
	GetDriverActiveQuests(ctx context.Context, driverID uuid.UUID) ([]*DriverQuestProgress, error)
	CreateQuestProgress(ctx context.Context, progress *DriverQuestProgress) error
	UpdateQuestProgress(ctx context.Context, progressID uuid.UUID, currentValue int, status QuestStatus) error
	ClaimQuestReward(ctx context.Context, progressID uuid.UUID, rewardAmount float64) error

	// Achievements
	GetAchievement(ctx context.Context, achievementID uuid.UUID) (*Achievement, error)
	GetAllAchievements(ctx context.Context) ([]*Achievement, error)
	GetDriverAchievements(ctx context.Context, driverID uuid.UUID) ([]*DriverAchievement, error)
	HasAchievement(ctx context.Context, driverID, achievementID uuid.UUID) (bool, error)
	AwardAchievement(ctx context.Context, driverID, achievementID uuid.UUID) error

	// Leaderboard
	GetLeaderboard(ctx context.Context, period string, category string, limit int) ([]LeaderboardEntry, error)
	GetDriverLeaderboardPosition(ctx context.Context, driverID uuid.UUID, period string, category string) (*LeaderboardEntry, error)
}
