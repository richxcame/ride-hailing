package gamification

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// Service handles gamification business logic
type Service struct {
	repo RepositoryInterface
}

// NewService creates a new gamification service
func NewService(repo RepositoryInterface) *Service {
	return &Service{repo: repo}
}

// ========================================
// PROFILE MANAGEMENT
// ========================================

// GetOrCreateProfile gets or creates a driver's gamification profile
func (s *Service) GetOrCreateProfile(ctx context.Context, driverID uuid.UUID) (*DriverGamification, error) {
	profile, err := s.repo.GetDriverGamification(ctx, driverID)
	if err == nil {
		return profile, nil
	}

	// Create new profile
	newTier, err := s.repo.GetTierByName(ctx, DriverTierNew)
	if err != nil {
		return nil, common.NewInternalServerError("failed to get default tier")
	}

	profile = &DriverGamification{
		DriverID:      driverID,
		CurrentTierID: &newTier.ID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.CreateDriverGamification(ctx, profile); err != nil {
		return nil, common.NewInternalServerError("failed to create gamification profile")
	}

	profile.CurrentTier = newTier
	return profile, nil
}

// GetGamificationStatus gets the full gamification status for a driver
func (s *Service) GetGamificationStatus(ctx context.Context, driverID uuid.UUID) (*GamificationStatusResponse, error) {
	profile, err := s.GetOrCreateProfile(ctx, driverID)
	if err != nil {
		return nil, err
	}

	// Get all tiers for progress calculation
	tiers, _ := s.repo.GetAllTiers(ctx)
	var nextTier *DriverTier
	ridesToNext := 0
	tierProgress := 100.0

	for _, t := range tiers {
		if t.MinRides > profile.TotalRides {
			nextTier = t
			ridesToNext = t.MinRides - profile.TotalRides
			if profile.CurrentTier != nil && nextTier != nil {
				tierProgress = float64(profile.TotalRides-profile.CurrentTier.MinRides) / float64(nextTier.MinRides-profile.CurrentTier.MinRides) * 100
			}
			break
		}
	}

	// Get active quests
	activeQuests, err := s.GetActiveQuestsWithProgress(ctx, driverID)
	if err != nil {
		activeQuests = []QuestWithProgress{}
	}

	// Get recent achievements
	achievements, err := s.repo.GetDriverAchievements(ctx, driverID)
	if err != nil {
		achievements = []*DriverAchievement{}
	}

	// Limit to 5 recent achievements
	if len(achievements) > 5 {
		achievements = achievements[:5]
	}

	// Calculate weekly stats
	weeklyStats := &WeeklyStats{
		Rides:         profile.WeeklyRides,
		Earnings:      profile.WeeklyEarnings,
		AverageRating: profile.AverageRating,
		AcceptRate:    profile.AcceptanceRate,
	}

	return &GamificationStatusResponse{
		Profile:            profile,
		CurrentTier:        profile.CurrentTier,
		NextTier:           nextTier,
		TierProgress:       tierProgress,
		RidesToNextTier:    ridesToNext,
		ActiveQuests:       activeQuests,
		RecentAchievements: toAchievementSlice(achievements),
		WeeklyStats:        weeklyStats,
	}, nil
}

// ========================================
// QUEST MANAGEMENT
// ========================================

// GetActiveQuestsWithProgress gets active quests with driver's progress
func (s *Service) GetActiveQuestsWithProgress(ctx context.Context, driverID uuid.UUID) ([]QuestWithProgress, error) {
	profile, _ := s.repo.GetDriverGamification(ctx, driverID)

	var tierID *uuid.UUID
	if profile != nil {
		tierID = profile.CurrentTierID
	}

	// Get available quests
	quests, err := s.repo.GetActiveQuests(ctx, tierID, nil)
	if err != nil {
		return nil, err
	}

	var result []QuestWithProgress
	for _, quest := range quests {
		progress, _ := s.repo.GetQuestProgress(ctx, driverID, quest.ID)

		qwp := QuestWithProgress{
			Quest:         *quest,
			CurrentValue:  0,
			Status:        QuestStatusActive,
			DaysRemaining: int(time.Until(quest.EndDate).Hours() / 24),
		}

		if progress != nil {
			qwp.CurrentValue = progress.CurrentValue
			qwp.Status = progress.Status
			qwp.ProgressPercent = float64(progress.CurrentValue) / float64(quest.TargetValue) * 100
			if qwp.ProgressPercent > 100 {
				qwp.ProgressPercent = 100
			}
		}

		result = append(result, qwp)
	}

	return result, nil
}

// JoinQuest enrolls a driver in a quest
func (s *Service) JoinQuest(ctx context.Context, driverID, questID uuid.UUID) error {
	// Check if already joined
	existing, _ := s.repo.GetQuestProgress(ctx, driverID, questID)
	if existing != nil {
		return nil // Already joined
	}

	// Create progress record
	progress := &DriverQuestProgress{
		ID:       uuid.New(),
		DriverID: driverID,
		QuestID:  questID,
		Status:   QuestStatusActive,
	}

	return s.repo.CreateQuestProgress(ctx, progress)
}

// UpdateQuestProgress updates quest progress based on events
func (s *Service) UpdateQuestProgress(ctx context.Context, driverID uuid.UUID, questType QuestType, increment int) error {
	quests, err := s.repo.GetQuestsByType(ctx, questType)
	if err != nil {
		return err
	}

	for _, quest := range quests {
		progress, _ := s.repo.GetQuestProgress(ctx, driverID, quest.ID)

		if progress == nil {
			// Auto-join the quest
			progress = &DriverQuestProgress{
				ID:       uuid.New(),
				DriverID: driverID,
				QuestID:  quest.ID,
				Status:   QuestStatusActive,
			}
			if err := s.repo.CreateQuestProgress(ctx, progress); err != nil {
				continue
			}
		}

		if progress.Status != QuestStatusActive {
			continue // Already completed or claimed
		}

		// Update progress
		newValue := progress.CurrentValue + increment
		status := progress.Status

		if newValue >= quest.TargetValue {
			status = QuestStatusCompleted
		}

		if err := s.repo.UpdateQuestProgress(ctx, progress.ID, newValue, status); err != nil {
			logger.Error("failed to update quest progress", zap.Error(err))
			continue
		}

		if status == QuestStatusCompleted {
			logger.Info("Quest completed",
				zap.String("driver_id", driverID.String()),
				zap.String("quest_id", quest.ID.String()),
			)
		}
	}

	return nil
}

// ClaimQuestReward claims a quest reward
func (s *Service) ClaimQuestReward(ctx context.Context, req *ClaimQuestRewardRequest) (*ClaimQuestRewardResponse, error) {
	progress, err := s.repo.GetQuestProgress(ctx, req.DriverID, req.QuestID)
	if err != nil {
		return nil, common.NewNotFoundError("quest progress not found", err)
	}

	if progress.Status != QuestStatusCompleted {
		return nil, common.NewBadRequestError("quest not completed yet", nil)
	}

	quest, err := s.repo.GetQuest(ctx, req.QuestID)
	if err != nil {
		return nil, common.NewNotFoundError("quest not found", err)
	}

	// Mark as claimed
	if err := s.repo.ClaimQuestReward(ctx, progress.ID, quest.RewardValue); err != nil {
		return nil, common.NewInternalServerError("failed to claim reward")
	}

	// Add bonus earned
	if quest.RewardType == "bonus" {
		if err := s.repo.AddBonusEarned(ctx, req.DriverID, quest.RewardValue); err != nil {
			logger.Error("failed to add bonus earned", zap.Error(err))
		}
	}

	// Increment quests completed
	if err := s.repo.IncrementQuestsCompleted(ctx, req.DriverID); err != nil {
		logger.Error("failed to increment quests completed", zap.Error(err))
	}

	logger.Info("Quest reward claimed",
		zap.String("driver_id", req.DriverID.String()),
		zap.String("quest_id", req.QuestID.String()),
		zap.Float64("reward", quest.RewardValue),
	)

	return &ClaimQuestRewardResponse{
		Success:      true,
		RewardType:   quest.RewardType,
		RewardAmount: quest.RewardValue,
		Message:      "Reward claimed successfully!",
	}, nil
}

// ========================================
// TIER MANAGEMENT
// ========================================

// CheckTierUpgrade checks if a driver qualifies for a tier upgrade
func (s *Service) CheckTierUpgrade(ctx context.Context, driverID uuid.UUID) error {
	profile, err := s.repo.GetDriverGamification(ctx, driverID)
	if err != nil {
		return err
	}

	tiers, err := s.repo.GetAllTiers(ctx)
	if err != nil {
		return err
	}

	// Find the highest tier the driver qualifies for
	var newTier *DriverTier
	for _, t := range tiers {
		if profile.TotalRides >= t.MinRides && profile.AverageRating >= t.MinRating {
			newTier = t
		}
	}

	if newTier == nil || (profile.CurrentTierID != nil && *profile.CurrentTierID == newTier.ID) {
		return nil // No change
	}

	// Upgrade tier
	if err := s.repo.UpdateDriverTier(ctx, driverID, newTier.ID); err != nil {
		return err
	}

	logger.Info("Driver tier upgraded",
		zap.String("driver_id", driverID.String()),
		zap.String("new_tier", string(newTier.Name)),
	)

	return nil
}

// GetAllTiers returns all driver tiers
func (s *Service) GetAllTiers(ctx context.Context) ([]*DriverTier, error) {
	return s.repo.GetAllTiers(ctx)
}

// ========================================
// ACHIEVEMENTS
// ========================================

// CheckAchievements checks and awards achievements based on current stats
func (s *Service) CheckAchievements(ctx context.Context, driverID uuid.UUID) error {
	profile, err := s.repo.GetDriverGamification(ctx, driverID)
	if err != nil {
		return err
	}

	achievements, err := s.repo.GetAllAchievements(ctx)
	if err != nil {
		return err
	}

	for _, achievement := range achievements {
		// Check if already earned
		has, _ := s.repo.HasAchievement(ctx, driverID, achievement.ID)
		if has {
			continue
		}

		// Check if criteria met
		if s.checkAchievementCriteria(profile, achievement) {
			if err := s.repo.AwardAchievement(ctx, driverID, achievement.ID); err != nil {
				logger.Error("failed to award achievement", zap.Error(err))
				continue
			}

			// Add points
			if err := s.repo.AddAchievementPoints(ctx, driverID, achievement.Points); err != nil {
				logger.Error("failed to add achievement points", zap.Error(err))
			}

			logger.Info("Achievement awarded",
				zap.String("driver_id", driverID.String()),
				zap.String("achievement", achievement.Name),
			)
		}
	}

	return nil
}

// GetDriverAchievements gets all achievements for a driver
func (s *Service) GetDriverAchievements(ctx context.Context, driverID uuid.UUID) ([]*DriverAchievement, error) {
	return s.repo.GetDriverAchievements(ctx, driverID)
}

func (s *Service) checkAchievementCriteria(profile *DriverGamification, achievement *Achievement) bool {
	// Basic criteria checking based on category
	switch achievement.Category {
	case AchievementCategoryMilestone:
		// Check ride milestones
		if profile.TotalRides >= 100 && achievement.Name == "Century Rider" {
			return true
		}
		if profile.TotalRides >= 500 && achievement.Name == "Road Warrior" {
			return true
		}
		if profile.TotalRides >= 1000 && achievement.Name == "Legendary Driver" {
			return true
		}
	case AchievementCategoryRating:
		// Check rating achievements
		if profile.AverageRating >= 4.9 && profile.TotalRides >= 50 && achievement.Name == "Five Star Elite" {
			return true
		}
	case AchievementCategoryService:
		// Check streak achievements
		if profile.LongestStreak >= 7 && achievement.Name == "Week Warrior" {
			return true
		}
		if profile.LongestStreak >= 30 && achievement.Name == "Month Master" {
			return true
		}
	}

	return false
}

// ========================================
// LEADERBOARD
// ========================================

// GetLeaderboard gets the leaderboard
func (s *Service) GetLeaderboard(ctx context.Context, driverID uuid.UUID, period, category string) (*LeaderboardResponse, error) {
	if period == "" {
		period = "weekly"
	}
	if category == "" {
		category = "rides"
	}

	entries, err := s.repo.GetLeaderboard(ctx, period, category, 100)
	if err != nil {
		return nil, common.NewInternalServerError("failed to get leaderboard")
	}

	// Get driver's position
	driverPosition, _ := s.repo.GetDriverLeaderboardPosition(ctx, driverID, period, category)

	// Mark current driver in entries
	for i := range entries {
		if entries[i].DriverID == driverID {
			entries[i].IsCurrentDriver = true
			break
		}
	}

	return &LeaderboardResponse{
		Period:         period,
		Category:       category,
		Entries:        entries,
		DriverPosition: driverPosition,
	}, nil
}

// ========================================
// EVENT PROCESSING
// ========================================

// ProcessRideCompleted processes a ride completion event
func (s *Service) ProcessRideCompleted(ctx context.Context, driverID uuid.UUID, earnings float64, rating float64) error {
	// Ensure profile exists
	if _, err := s.GetOrCreateProfile(ctx, driverID); err != nil {
		return err
	}

	// Update stats
	if err := s.repo.UpdateDriverStats(ctx, driverID, 1, earnings, rating); err != nil {
		return err
	}

	// Update quest progress for ride count
	go func() {
		if err := s.UpdateQuestProgress(context.Background(), driverID, QuestTypeRideCount, 1); err != nil {
			logger.Error("failed to update quest progress for ride count", zap.Error(err))
		}
		if err := s.UpdateQuestProgress(context.Background(), driverID, QuestTypeEarnings, int(earnings)); err != nil {
			logger.Error("failed to update quest progress for earnings", zap.Error(err))
		}
		if err := s.CheckTierUpgrade(context.Background(), driverID); err != nil {
			logger.Error("failed to check tier upgrade", zap.Error(err))
		}
		if err := s.CheckAchievements(context.Background(), driverID); err != nil {
			logger.Error("failed to check achievements", zap.Error(err))
		}
	}()

	return nil
}

// ProcessStreak updates driver's streak
func (s *Service) ProcessStreak(ctx context.Context, driverID uuid.UUID) error {
	profile, err := s.repo.GetDriverGamification(ctx, driverID)
	if err != nil {
		return err
	}

	today := time.Now().Truncate(24 * time.Hour)
	var newStreak int

	if profile.LastActiveDate != nil {
		lastActive := profile.LastActiveDate.Truncate(24 * time.Hour)
		diff := today.Sub(lastActive)

		if diff == 24*time.Hour {
			// Consecutive day
			newStreak = profile.CurrentStreak + 1
		} else if diff == 0 {
			// Same day - no change
			return nil
		} else {
			// Streak broken
			newStreak = 1
		}
	} else {
		newStreak = 1
	}

	if err := s.repo.UpdateDriverStreak(ctx, driverID, newStreak); err != nil {
		return err
	}

	// Update streak quest progress
	go func() {
		if err := s.UpdateQuestProgress(context.Background(), driverID, QuestTypeStreak, newStreak); err != nil {
			logger.Error("failed to update quest progress for streak", zap.Error(err))
		}
	}()

	return nil
}

// Helper function to convert pointer slice to value slice
func toAchievementSlice(achievements []*DriverAchievement) []DriverAchievement {
	result := make([]DriverAchievement, len(achievements))
	for i, a := range achievements {
		result[i] = *a
	}
	return result
}
