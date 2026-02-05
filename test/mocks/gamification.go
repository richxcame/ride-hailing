package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/internal/gamification"
	"github.com/stretchr/testify/mock"
)

// MockGamificationRepository is a mock implementation of the gamification repository
type MockGamificationRepository struct {
	mock.Mock
}

// Driver Gamification Profile

func (m *MockGamificationRepository) GetDriverGamification(ctx context.Context, driverID uuid.UUID) (*gamification.DriverGamification, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gamification.DriverGamification), args.Error(1)
}

func (m *MockGamificationRepository) CreateDriverGamification(ctx context.Context, profile *gamification.DriverGamification) error {
	args := m.Called(ctx, profile)
	return args.Error(0)
}

func (m *MockGamificationRepository) UpdateDriverStats(ctx context.Context, driverID uuid.UUID, rides int, earnings float64, rating float64) error {
	args := m.Called(ctx, driverID, rides, earnings, rating)
	return args.Error(0)
}

func (m *MockGamificationRepository) UpdateDriverStreak(ctx context.Context, driverID uuid.UUID, currentStreak int) error {
	args := m.Called(ctx, driverID, currentStreak)
	return args.Error(0)
}

func (m *MockGamificationRepository) UpdateDriverTier(ctx context.Context, driverID uuid.UUID, tierID uuid.UUID) error {
	args := m.Called(ctx, driverID, tierID)
	return args.Error(0)
}

func (m *MockGamificationRepository) IncrementQuestsCompleted(ctx context.Context, driverID uuid.UUID) error {
	args := m.Called(ctx, driverID)
	return args.Error(0)
}

func (m *MockGamificationRepository) AddBonusEarned(ctx context.Context, driverID uuid.UUID, bonus float64) error {
	args := m.Called(ctx, driverID, bonus)
	return args.Error(0)
}

func (m *MockGamificationRepository) AddAchievementPoints(ctx context.Context, driverID uuid.UUID, points int) error {
	args := m.Called(ctx, driverID, points)
	return args.Error(0)
}

func (m *MockGamificationRepository) ResetWeeklyStats(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockGamificationRepository) ResetMonthlyStats(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Driver Tiers

func (m *MockGamificationRepository) GetTier(ctx context.Context, tierID uuid.UUID) (*gamification.DriverTier, error) {
	args := m.Called(ctx, tierID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gamification.DriverTier), args.Error(1)
}

func (m *MockGamificationRepository) GetTierByName(ctx context.Context, name gamification.DriverTierName) (*gamification.DriverTier, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gamification.DriverTier), args.Error(1)
}

func (m *MockGamificationRepository) GetAllTiers(ctx context.Context) ([]*gamification.DriverTier, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*gamification.DriverTier), args.Error(1)
}

// Quests

func (m *MockGamificationRepository) GetActiveQuests(ctx context.Context, tierID *uuid.UUID, city *string) ([]*gamification.DriverQuest, error) {
	args := m.Called(ctx, tierID, city)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*gamification.DriverQuest), args.Error(1)
}

func (m *MockGamificationRepository) GetQuest(ctx context.Context, questID uuid.UUID) (*gamification.DriverQuest, error) {
	args := m.Called(ctx, questID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gamification.DriverQuest), args.Error(1)
}

func (m *MockGamificationRepository) GetQuestsByType(ctx context.Context, questType gamification.QuestType) ([]*gamification.DriverQuest, error) {
	args := m.Called(ctx, questType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*gamification.DriverQuest), args.Error(1)
}

// Quest Progress

func (m *MockGamificationRepository) GetQuestProgress(ctx context.Context, driverID, questID uuid.UUID) (*gamification.DriverQuestProgress, error) {
	args := m.Called(ctx, driverID, questID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gamification.DriverQuestProgress), args.Error(1)
}

func (m *MockGamificationRepository) GetDriverActiveQuests(ctx context.Context, driverID uuid.UUID) ([]*gamification.DriverQuestProgress, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*gamification.DriverQuestProgress), args.Error(1)
}

func (m *MockGamificationRepository) CreateQuestProgress(ctx context.Context, progress *gamification.DriverQuestProgress) error {
	args := m.Called(ctx, progress)
	return args.Error(0)
}

func (m *MockGamificationRepository) UpdateQuestProgress(ctx context.Context, progressID uuid.UUID, currentValue int, status gamification.QuestStatus) error {
	args := m.Called(ctx, progressID, currentValue, status)
	return args.Error(0)
}

func (m *MockGamificationRepository) ClaimQuestReward(ctx context.Context, progressID uuid.UUID, rewardAmount float64) error {
	args := m.Called(ctx, progressID, rewardAmount)
	return args.Error(0)
}

// Achievements

func (m *MockGamificationRepository) GetAchievement(ctx context.Context, achievementID uuid.UUID) (*gamification.Achievement, error) {
	args := m.Called(ctx, achievementID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gamification.Achievement), args.Error(1)
}

func (m *MockGamificationRepository) GetAllAchievements(ctx context.Context) ([]*gamification.Achievement, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*gamification.Achievement), args.Error(1)
}

func (m *MockGamificationRepository) GetDriverAchievements(ctx context.Context, driverID uuid.UUID) ([]*gamification.DriverAchievement, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*gamification.DriverAchievement), args.Error(1)
}

func (m *MockGamificationRepository) HasAchievement(ctx context.Context, driverID, achievementID uuid.UUID) (bool, error) {
	args := m.Called(ctx, driverID, achievementID)
	return args.Bool(0), args.Error(1)
}

func (m *MockGamificationRepository) AwardAchievement(ctx context.Context, driverID, achievementID uuid.UUID) error {
	args := m.Called(ctx, driverID, achievementID)
	return args.Error(0)
}

// Leaderboard

func (m *MockGamificationRepository) GetLeaderboard(ctx context.Context, period string, category string, limit int) ([]gamification.LeaderboardEntry, error) {
	args := m.Called(ctx, period, category, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]gamification.LeaderboardEntry), args.Error(1)
}

func (m *MockGamificationRepository) GetDriverLeaderboardPosition(ctx context.Context, driverID uuid.UUID, period string, category string) (*gamification.LeaderboardEntry, error) {
	args := m.Called(ctx, driverID, period, category)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gamification.LeaderboardEntry), args.Error(1)
}
