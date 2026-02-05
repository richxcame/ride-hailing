package gamification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ========================================
// INTERNAL MOCK (implements RepositoryInterface within this package)
// ========================================

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) GetDriverGamification(ctx context.Context, driverID uuid.UUID) (*DriverGamification, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverGamification), args.Error(1)
}

func (m *mockRepo) CreateDriverGamification(ctx context.Context, profile *DriverGamification) error {
	args := m.Called(ctx, profile)
	return args.Error(0)
}

func (m *mockRepo) UpdateDriverStats(ctx context.Context, driverID uuid.UUID, rides int, earnings float64, rating float64) error {
	args := m.Called(ctx, driverID, rides, earnings, rating)
	return args.Error(0)
}

func (m *mockRepo) UpdateDriverStreak(ctx context.Context, driverID uuid.UUID, currentStreak int) error {
	args := m.Called(ctx, driverID, currentStreak)
	return args.Error(0)
}

func (m *mockRepo) UpdateDriverTier(ctx context.Context, driverID uuid.UUID, tierID uuid.UUID) error {
	args := m.Called(ctx, driverID, tierID)
	return args.Error(0)
}

func (m *mockRepo) IncrementQuestsCompleted(ctx context.Context, driverID uuid.UUID) error {
	args := m.Called(ctx, driverID)
	return args.Error(0)
}

func (m *mockRepo) AddBonusEarned(ctx context.Context, driverID uuid.UUID, bonus float64) error {
	args := m.Called(ctx, driverID, bonus)
	return args.Error(0)
}

func (m *mockRepo) AddAchievementPoints(ctx context.Context, driverID uuid.UUID, points int) error {
	args := m.Called(ctx, driverID, points)
	return args.Error(0)
}

func (m *mockRepo) ResetWeeklyStats(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockRepo) ResetMonthlyStats(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockRepo) GetTier(ctx context.Context, tierID uuid.UUID) (*DriverTier, error) {
	args := m.Called(ctx, tierID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverTier), args.Error(1)
}

func (m *mockRepo) GetTierByName(ctx context.Context, name DriverTierName) (*DriverTier, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverTier), args.Error(1)
}

func (m *mockRepo) GetAllTiers(ctx context.Context) ([]*DriverTier, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DriverTier), args.Error(1)
}

func (m *mockRepo) GetActiveQuests(ctx context.Context, tierID *uuid.UUID, city *string) ([]*DriverQuest, error) {
	args := m.Called(ctx, tierID, city)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DriverQuest), args.Error(1)
}

func (m *mockRepo) GetQuest(ctx context.Context, questID uuid.UUID) (*DriverQuest, error) {
	args := m.Called(ctx, questID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverQuest), args.Error(1)
}

func (m *mockRepo) GetQuestsByType(ctx context.Context, questType QuestType) ([]*DriverQuest, error) {
	args := m.Called(ctx, questType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DriverQuest), args.Error(1)
}

func (m *mockRepo) GetQuestProgress(ctx context.Context, driverID, questID uuid.UUID) (*DriverQuestProgress, error) {
	args := m.Called(ctx, driverID, questID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverQuestProgress), args.Error(1)
}

func (m *mockRepo) GetDriverActiveQuests(ctx context.Context, driverID uuid.UUID) ([]*DriverQuestProgress, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DriverQuestProgress), args.Error(1)
}

func (m *mockRepo) CreateQuestProgress(ctx context.Context, progress *DriverQuestProgress) error {
	args := m.Called(ctx, progress)
	return args.Error(0)
}

func (m *mockRepo) UpdateQuestProgress(ctx context.Context, progressID uuid.UUID, currentValue int, status QuestStatus) error {
	args := m.Called(ctx, progressID, currentValue, status)
	return args.Error(0)
}

func (m *mockRepo) ClaimQuestReward(ctx context.Context, progressID uuid.UUID, rewardAmount float64) error {
	args := m.Called(ctx, progressID, rewardAmount)
	return args.Error(0)
}

func (m *mockRepo) GetAchievement(ctx context.Context, achievementID uuid.UUID) (*Achievement, error) {
	args := m.Called(ctx, achievementID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Achievement), args.Error(1)
}

func (m *mockRepo) GetAllAchievements(ctx context.Context) ([]*Achievement, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Achievement), args.Error(1)
}

func (m *mockRepo) GetDriverAchievements(ctx context.Context, driverID uuid.UUID) ([]*DriverAchievement, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DriverAchievement), args.Error(1)
}

func (m *mockRepo) HasAchievement(ctx context.Context, driverID, achievementID uuid.UUID) (bool, error) {
	args := m.Called(ctx, driverID, achievementID)
	return args.Bool(0), args.Error(1)
}

func (m *mockRepo) AwardAchievement(ctx context.Context, driverID, achievementID uuid.UUID) error {
	args := m.Called(ctx, driverID, achievementID)
	return args.Error(0)
}

func (m *mockRepo) GetLeaderboard(ctx context.Context, period string, category string, limit int) ([]LeaderboardEntry, error) {
	args := m.Called(ctx, period, category, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]LeaderboardEntry), args.Error(1)
}

func (m *mockRepo) GetDriverLeaderboardPosition(ctx context.Context, driverID uuid.UUID, period string, category string) (*LeaderboardEntry, error) {
	args := m.Called(ctx, driverID, period, category)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LeaderboardEntry), args.Error(1)
}

// ========================================
// TEST HELPERS
// ========================================

func newTestService(repo RepositoryInterface) *Service {
	return NewService(repo)
}

// ========================================
// TESTS
// ========================================

func TestGetOrCreateProfile(t *testing.T) {
	driverID := uuid.New()
	tierID := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, profile *DriverGamification)
	}{
		{
			name: "existing profile found",
			setupMocks: func(m *mockRepo) {
				existingProfile := &DriverGamification{
					DriverID:      driverID,
					CurrentTierID: &tierID,
					TotalRides:    50,
					AverageRating: 4.8,
					CurrentTier: &DriverTier{
						ID:   tierID,
						Name: DriverTierBronze,
					},
				}
				m.On("GetDriverGamification", mock.Anything, driverID).Return(existingProfile, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, profile *DriverGamification) {
				assert.Equal(t, driverID, profile.DriverID)
				assert.Equal(t, 50, profile.TotalRides)
				assert.Equal(t, 4.8, profile.AverageRating)
				assert.NotNil(t, profile.CurrentTier)
				assert.Equal(t, DriverTierBronze, profile.CurrentTier.Name)
			},
		},
		{
			name: "new profile created",
			setupMocks: func(m *mockRepo) {
				newTier := &DriverTier{
					ID:          tierID,
					Name:        DriverTierNew,
					DisplayName: "New Driver",
					MinRides:    0,
					MinRating:   0.0,
				}
				m.On("GetDriverGamification", mock.Anything, driverID).Return(nil, errors.New("not found"))
				m.On("GetTierByName", mock.Anything, DriverTierNew).Return(newTier, nil)
				m.On("CreateDriverGamification", mock.Anything, mock.AnythingOfType("*gamification.DriverGamification")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, profile *DriverGamification) {
				assert.Equal(t, driverID, profile.DriverID)
				assert.NotNil(t, profile.CurrentTierID)
				assert.Equal(t, tierID, *profile.CurrentTierID)
				assert.NotNil(t, profile.CurrentTier)
				assert.Equal(t, DriverTierNew, profile.CurrentTier.Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			profile, err := svc.GetOrCreateProfile(context.Background(), driverID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, profile)
			} else {
				require.NoError(t, err)
				require.NotNil(t, profile)
				tt.validate(t, profile)
			}

			m.AssertExpectations(t)
		})
	}
}

func TestCheckTierUpgrade(t *testing.T) {
	driverID := uuid.New()
	bronzeTierID := uuid.New()
	silverTierID := uuid.New()
	goldTierID := uuid.New()

	allTiers := []*DriverTier{
		{ID: uuid.New(), Name: DriverTierNew, MinRides: 0, MinRating: 0.0},
		{ID: bronzeTierID, Name: DriverTierBronze, MinRides: 10, MinRating: 4.0},
		{ID: silverTierID, Name: DriverTierSilver, MinRides: 50, MinRating: 4.3},
		{ID: goldTierID, Name: DriverTierGold, MinRides: 200, MinRating: 4.5},
	}

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
	}{
		{
			name: "qualifies for upgrade - rides and rating met",
			setupMocks: func(m *mockRepo) {
				profile := &DriverGamification{
					DriverID:      driverID,
					CurrentTierID: &bronzeTierID,
					TotalRides:    60,
					AverageRating: 4.5,
				}
				m.On("GetDriverGamification", mock.Anything, driverID).Return(profile, nil)
				m.On("GetAllTiers", mock.Anything).Return(allTiers, nil)
				m.On("UpdateDriverTier", mock.Anything, driverID, silverTierID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "no change - already correct tier",
			setupMocks: func(m *mockRepo) {
				profile := &DriverGamification{
					DriverID:      driverID,
					CurrentTierID: &silverTierID,
					TotalRides:    60,
					AverageRating: 4.5,
				}
				m.On("GetDriverGamification", mock.Anything, driverID).Return(profile, nil)
				m.On("GetAllTiers", mock.Anything).Return(allTiers, nil)
				// UpdateDriverTier should NOT be called since the tier hasn't changed
			},
			wantErr: false,
		},
		{
			name: "rating too low - no upgrade",
			setupMocks: func(m *mockRepo) {
				profile := &DriverGamification{
					DriverID:      driverID,
					CurrentTierID: &bronzeTierID,
					TotalRides:    60,
					AverageRating: 4.1, // meets Silver ride requirement but not rating for Silver (4.3)
				}
				m.On("GetDriverGamification", mock.Anything, driverID).Return(profile, nil)
				m.On("GetAllTiers", mock.Anything).Return(allTiers, nil)
				// Only qualifies for Bronze (rides>=10, rating>=4.0), which is current tier
				// UpdateDriverTier should NOT be called since already at bronze
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.CheckTierUpgrade(context.Background(), driverID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

func TestCheckAchievements(t *testing.T) {
	driverID := uuid.New()
	centuryRiderID := uuid.New()
	roadWarriorID := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
	}{
		{
			name: "milestone reached - 100 rides awards Century Rider",
			setupMocks: func(m *mockRepo) {
				profile := &DriverGamification{
					DriverID:   driverID,
					TotalRides: 100,
				}
				achievements := []*Achievement{
					{
						ID:       centuryRiderID,
						Name:     "Century Rider",
						Category: AchievementCategoryMilestone,
						Points:   100,
					},
				}
				m.On("GetDriverGamification", mock.Anything, driverID).Return(profile, nil)
				m.On("GetAllAchievements", mock.Anything).Return(achievements, nil)
				m.On("HasAchievement", mock.Anything, driverID, centuryRiderID).Return(false, nil)
				m.On("AwardAchievement", mock.Anything, driverID, centuryRiderID).Return(nil)
				m.On("AddAchievementPoints", mock.Anything, driverID, 100).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "already earned - skipped",
			setupMocks: func(m *mockRepo) {
				profile := &DriverGamification{
					DriverID:   driverID,
					TotalRides: 500,
				}
				achievements := []*Achievement{
					{
						ID:       centuryRiderID,
						Name:     "Century Rider",
						Category: AchievementCategoryMilestone,
						Points:   100,
					},
					{
						ID:       roadWarriorID,
						Name:     "Road Warrior",
						Category: AchievementCategoryMilestone,
						Points:   250,
					},
				}
				m.On("GetDriverGamification", mock.Anything, driverID).Return(profile, nil)
				m.On("GetAllAchievements", mock.Anything).Return(achievements, nil)
				// Century Rider already earned
				m.On("HasAchievement", mock.Anything, driverID, centuryRiderID).Return(true, nil)
				// Road Warrior not yet earned
				m.On("HasAchievement", mock.Anything, driverID, roadWarriorID).Return(false, nil)
				m.On("AwardAchievement", mock.Anything, driverID, roadWarriorID).Return(nil)
				m.On("AddAchievementPoints", mock.Anything, driverID, 250).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.CheckAchievements(context.Background(), driverID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

func TestUpdateQuestProgress(t *testing.T) {
	driverID := uuid.New()
	questID := uuid.New()
	progressID := uuid.New()

	tests := []struct {
		name       string
		questType  QuestType
		increment  int
		setupMocks func(m *mockRepo)
		wantErr    bool
	}{
		{
			name:      "progress reaches target - completes quest",
			questType: QuestTypeRideCount,
			increment: 1,
			setupMocks: func(m *mockRepo) {
				quests := []*DriverQuest{
					{
						ID:          questID,
						Name:        "Complete 10 rides",
						QuestType:   QuestTypeRideCount,
						TargetValue: 10,
						RewardType:  "bonus",
						RewardValue: 50.0,
					},
				}
				progress := &DriverQuestProgress{
					ID:           progressID,
					DriverID:     driverID,
					QuestID:      questID,
					CurrentValue: 9,
					Status:       QuestStatusActive,
				}
				m.On("GetQuestsByType", mock.Anything, QuestTypeRideCount).Return(quests, nil)
				m.On("GetQuestProgress", mock.Anything, driverID, questID).Return(progress, nil)
				m.On("UpdateQuestProgress", mock.Anything, progressID, 10, QuestStatusCompleted).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.UpdateQuestProgress(context.Background(), driverID, tt.questType, tt.increment)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

func TestClaimQuestReward(t *testing.T) {
	driverID := uuid.New()
	questID := uuid.New()
	progressID := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, resp *ClaimQuestRewardResponse)
	}{
		{
			name: "success",
			setupMocks: func(m *mockRepo) {
				progress := &DriverQuestProgress{
					ID:       progressID,
					DriverID: driverID,
					QuestID:  questID,
					Status:   QuestStatusCompleted,
				}
				quest := &DriverQuest{
					ID:          questID,
					Name:        "Complete 10 rides",
					RewardType:  "bonus",
					RewardValue: 50.0,
				}
				m.On("GetQuestProgress", mock.Anything, driverID, questID).Return(progress, nil)
				m.On("GetQuest", mock.Anything, questID).Return(quest, nil)
				m.On("ClaimQuestReward", mock.Anything, progressID, 50.0).Return(nil)
				m.On("AddBonusEarned", mock.Anything, driverID, 50.0).Return(nil)
				m.On("IncrementQuestsCompleted", mock.Anything, driverID).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *ClaimQuestRewardResponse) {
				assert.True(t, resp.Success)
				assert.Equal(t, "bonus", resp.RewardType)
				assert.Equal(t, 50.0, resp.RewardAmount)
				assert.Equal(t, "Reward claimed successfully!", resp.Message)
			},
		},
		{
			name: "not completed - error",
			setupMocks: func(m *mockRepo) {
				progress := &DriverQuestProgress{
					ID:       progressID,
					DriverID: driverID,
					QuestID:  questID,
					Status:   QuestStatusActive, // Not completed
				}
				m.On("GetQuestProgress", mock.Anything, driverID, questID).Return(progress, nil)
			},
			wantErr: true,
			validate: func(t *testing.T, resp *ClaimQuestRewardResponse) {
				// resp should be nil since error is returned
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			req := &ClaimQuestRewardRequest{
				DriverID: driverID,
				QuestID:  questID,
			}
			resp, err := svc.ClaimQuestReward(context.Background(), req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				tt.validate(t, resp)
			}

			m.AssertExpectations(t)
		})
	}
}

func TestProcessStreak(t *testing.T) {
	driverID := uuid.New()

	tests := []struct {
		name           string
		setupMocks     func(m *mockRepo)
		wantErr        bool
		expectedStreak int
	}{
		{
			name: "consecutive day increments streak",
			setupMocks: func(m *mockRepo) {
				yesterday := time.Now().Add(-24 * time.Hour)
				profile := &DriverGamification{
					DriverID:       driverID,
					CurrentStreak:  5,
					LongestStreak:  10,
					LastActiveDate: &yesterday,
				}
				m.On("GetDriverGamification", mock.Anything, driverID).Return(profile, nil)
				m.On("UpdateDriverStreak", mock.Anything, driverID, 6).Return(nil)
				// ProcessStreak fires a goroutine that calls UpdateQuestProgress → GetQuestsByType
				m.On("GetQuestsByType", mock.Anything, QuestTypeStreak).Return([]*DriverQuest{}, nil).Maybe()
			},
			wantErr:        false,
			expectedStreak: 6,
		},
		{
			name: "broken streak resets to 1",
			setupMocks: func(m *mockRepo) {
				threeDaysAgo := time.Now().Add(-72 * time.Hour)
				profile := &DriverGamification{
					DriverID:       driverID,
					CurrentStreak:  5,
					LongestStreak:  10,
					LastActiveDate: &threeDaysAgo,
				}
				m.On("GetDriverGamification", mock.Anything, driverID).Return(profile, nil)
				m.On("UpdateDriverStreak", mock.Anything, driverID, 1).Return(nil)
				// ProcessStreak fires a goroutine that calls UpdateQuestProgress → GetQuestsByType
				m.On("GetQuestsByType", mock.Anything, QuestTypeStreak).Return([]*DriverQuest{}, nil).Maybe()
			},
			wantErr:        false,
			expectedStreak: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.ProcessStreak(context.Background(), driverID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Wait for the background goroutine (UpdateQuestProgress) to complete
			time.Sleep(50 * time.Millisecond)
			m.AssertExpectations(t)
		})
	}
}

func TestGetLeaderboard(t *testing.T) {
	driverID := uuid.New()
	otherDriverID := uuid.New()

	tests := []struct {
		name       string
		period     string
		category   string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, resp *LeaderboardResponse)
	}{
		{
			name:     "success",
			period:   "weekly",
			category: "rides",
			setupMocks: func(m *mockRepo) {
				entries := []LeaderboardEntry{
					{
						Rank:       1,
						DriverID:   otherDriverID,
						DriverName: "Alice A.",
						Value:      50,
					},
					{
						Rank:       2,
						DriverID:   driverID,
						DriverName: "Bob B.",
						Value:      30,
					},
				}
				driverPosition := &LeaderboardEntry{
					Rank:            2,
					DriverID:        driverID,
					DriverName:      "Bob B.",
					Value:           30,
					IsCurrentDriver: true,
				}
				m.On("GetLeaderboard", mock.Anything, "weekly", "rides", 100).Return(entries, nil)
				m.On("GetDriverLeaderboardPosition", mock.Anything, driverID, "weekly", "rides").Return(driverPosition, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *LeaderboardResponse) {
				assert.Equal(t, "weekly", resp.Period)
				assert.Equal(t, "rides", resp.Category)
				require.Len(t, resp.Entries, 2)
				assert.Equal(t, 1, resp.Entries[0].Rank)
				assert.Equal(t, otherDriverID, resp.Entries[0].DriverID)
				assert.False(t, resp.Entries[0].IsCurrentDriver)
				assert.True(t, resp.Entries[1].IsCurrentDriver)
				require.NotNil(t, resp.DriverPosition)
				assert.Equal(t, 2, resp.DriverPosition.Rank)
				assert.True(t, resp.DriverPosition.IsCurrentDriver)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			resp, err := svc.GetLeaderboard(context.Background(), driverID, tt.period, tt.category)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				tt.validate(t, resp)
			}

			m.AssertExpectations(t)
		})
	}
}
