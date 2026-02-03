package gamification

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for gamification
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new gamification repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// DRIVER GAMIFICATION PROFILE
// ========================================

// GetDriverGamification gets a driver's gamification profile
func (r *Repository) GetDriverGamification(ctx context.Context, driverID uuid.UUID) (*DriverGamification, error) {
	query := `
		SELECT dg.driver_id, dg.current_tier_id, dg.total_rides, dg.total_earnings,
		       dg.average_rating, dg.current_streak, dg.longest_streak, dg.total_bonus_earned,
		       dg.achievement_points, dg.quests_completed, dg.last_active_date,
		       dg.weekly_rides, dg.weekly_earnings, dg.monthly_rides, dg.monthly_earnings,
		       dg.acceptance_rate, dg.cancellation_rate, dg.tier_upgraded_at, dg.created_at, dg.updated_at,
		       dt.id, dt.name, dt.display_name, dt.min_rides, dt.min_rating,
		       dt.commission_rate, dt.bonus_multiplier, dt.priority_dispatch, dt.benefits
		FROM driver_gamification dg
		LEFT JOIN driver_tiers dt ON dg.current_tier_id = dt.id
		WHERE dg.driver_id = $1
	`

	profile := &DriverGamification{}
	var tier DriverTier
	var tierID *uuid.UUID

	err := r.db.QueryRow(ctx, query, driverID).Scan(
		&profile.DriverID, &profile.CurrentTierID, &profile.TotalRides, &profile.TotalEarnings,
		&profile.AverageRating, &profile.CurrentStreak, &profile.LongestStreak, &profile.TotalBonusEarned,
		&profile.AchievementPoints, &profile.QuestsCompleted, &profile.LastActiveDate,
		&profile.WeeklyRides, &profile.WeeklyEarnings, &profile.MonthlyRides, &profile.MonthlyEarnings,
		&profile.AcceptanceRate, &profile.CancellationRate, &profile.TierUpgradedAt, &profile.CreatedAt, &profile.UpdatedAt,
		&tierID, &tier.Name, &tier.DisplayName, &tier.MinRides, &tier.MinRating,
		&tier.CommissionRate, &tier.BonusMultiplier, &tier.PriorityDispatch, &tier.Benefits,
	)

	if err != nil {
		return nil, err
	}

	if tierID != nil {
		tier.ID = *tierID
		profile.CurrentTier = &tier
	}

	return profile, nil
}

// CreateDriverGamification creates a new gamification profile
func (r *Repository) CreateDriverGamification(ctx context.Context, profile *DriverGamification) error {
	query := `
		INSERT INTO driver_gamification (
			driver_id, current_tier_id, total_rides, total_earnings,
			average_rating, current_streak, longest_streak
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (driver_id) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query,
		profile.DriverID, profile.CurrentTierID, profile.TotalRides,
		profile.TotalEarnings, profile.AverageRating, profile.CurrentStreak, profile.LongestStreak,
	)

	return err
}

// UpdateDriverStats updates driver statistics after a ride
func (r *Repository) UpdateDriverStats(ctx context.Context, driverID uuid.UUID, rides int, earnings float64, rating float64) error {
	query := `
		UPDATE driver_gamification
		SET total_rides = total_rides + $1,
		    total_earnings = total_earnings + $2,
		    average_rating = CASE
		        WHEN total_rides = 0 THEN $3
		        ELSE (average_rating * total_rides + $3) / (total_rides + 1)
		    END,
		    weekly_rides = weekly_rides + $1,
		    weekly_earnings = weekly_earnings + $2,
		    monthly_rides = monthly_rides + $1,
		    monthly_earnings = monthly_earnings + $2,
		    last_active_date = NOW(),
		    updated_at = NOW()
		WHERE driver_id = $4
	`

	_, err := r.db.Exec(ctx, query, rides, earnings, rating, driverID)
	return err
}

// UpdateDriverStreak updates driver's streak
func (r *Repository) UpdateDriverStreak(ctx context.Context, driverID uuid.UUID, currentStreak int) error {
	query := `
		UPDATE driver_gamification
		SET current_streak = $1,
		    longest_streak = GREATEST(longest_streak, $1),
		    last_active_date = NOW(),
		    updated_at = NOW()
		WHERE driver_id = $2
	`

	_, err := r.db.Exec(ctx, query, currentStreak, driverID)
	return err
}

// UpdateDriverTier updates driver's tier
func (r *Repository) UpdateDriverTier(ctx context.Context, driverID uuid.UUID, tierID uuid.UUID) error {
	query := `
		UPDATE driver_gamification
		SET current_tier_id = $1, tier_upgraded_at = NOW(), updated_at = NOW()
		WHERE driver_id = $2
	`

	_, err := r.db.Exec(ctx, query, tierID, driverID)
	return err
}

// IncrementQuestsCompleted increments quests completed count
func (r *Repository) IncrementQuestsCompleted(ctx context.Context, driverID uuid.UUID) error {
	query := `
		UPDATE driver_gamification
		SET quests_completed = quests_completed + 1, updated_at = NOW()
		WHERE driver_id = $1
	`

	_, err := r.db.Exec(ctx, query, driverID)
	return err
}

// AddBonusEarned adds bonus to driver's total
func (r *Repository) AddBonusEarned(ctx context.Context, driverID uuid.UUID, bonus float64) error {
	query := `
		UPDATE driver_gamification
		SET total_bonus_earned = total_bonus_earned + $1, updated_at = NOW()
		WHERE driver_id = $2
	`

	_, err := r.db.Exec(ctx, query, bonus, driverID)
	return err
}

// AddAchievementPoints adds achievement points
func (r *Repository) AddAchievementPoints(ctx context.Context, driverID uuid.UUID, points int) error {
	query := `
		UPDATE driver_gamification
		SET achievement_points = achievement_points + $1, updated_at = NOW()
		WHERE driver_id = $2
	`

	_, err := r.db.Exec(ctx, query, points, driverID)
	return err
}

// ResetWeeklyStats resets weekly statistics
func (r *Repository) ResetWeeklyStats(ctx context.Context) error {
	query := `UPDATE driver_gamification SET weekly_rides = 0, weekly_earnings = 0`
	_, err := r.db.Exec(ctx, query)
	return err
}

// ResetMonthlyStats resets monthly statistics
func (r *Repository) ResetMonthlyStats(ctx context.Context) error {
	query := `UPDATE driver_gamification SET monthly_rides = 0, monthly_earnings = 0`
	_, err := r.db.Exec(ctx, query)
	return err
}

// ========================================
// DRIVER TIERS
// ========================================

// GetTier gets a tier by ID
func (r *Repository) GetTier(ctx context.Context, tierID uuid.UUID) (*DriverTier, error) {
	query := `
		SELECT id, name, display_name, min_rides, min_rating, commission_rate,
		       bonus_multiplier, priority_dispatch, icon_url, color_hex, benefits, is_active
		FROM driver_tiers
		WHERE id = $1
	`

	tier := &DriverTier{}
	err := r.db.QueryRow(ctx, query, tierID).Scan(
		&tier.ID, &tier.Name, &tier.DisplayName, &tier.MinRides, &tier.MinRating,
		&tier.CommissionRate, &tier.BonusMultiplier, &tier.PriorityDispatch,
		&tier.IconURL, &tier.ColorHex, &tier.Benefits, &tier.IsActive,
	)

	if err != nil {
		return nil, err
	}

	return tier, nil
}

// GetTierByName gets a tier by name
func (r *Repository) GetTierByName(ctx context.Context, name DriverTierName) (*DriverTier, error) {
	query := `
		SELECT id, name, display_name, min_rides, min_rating, commission_rate,
		       bonus_multiplier, priority_dispatch, icon_url, color_hex, benefits, is_active
		FROM driver_tiers
		WHERE name = $1
	`

	tier := &DriverTier{}
	err := r.db.QueryRow(ctx, query, string(name)).Scan(
		&tier.ID, &tier.Name, &tier.DisplayName, &tier.MinRides, &tier.MinRating,
		&tier.CommissionRate, &tier.BonusMultiplier, &tier.PriorityDispatch,
		&tier.IconURL, &tier.ColorHex, &tier.Benefits, &tier.IsActive,
	)

	if err != nil {
		return nil, err
	}

	return tier, nil
}

// GetAllTiers gets all tiers ordered by min_rides
func (r *Repository) GetAllTiers(ctx context.Context) ([]*DriverTier, error) {
	query := `
		SELECT id, name, display_name, min_rides, min_rating, commission_rate,
		       bonus_multiplier, priority_dispatch, icon_url, color_hex, benefits, is_active
		FROM driver_tiers
		WHERE is_active = true
		ORDER BY min_rides ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tiers []*DriverTier
	for rows.Next() {
		tier := &DriverTier{}
		err := rows.Scan(
			&tier.ID, &tier.Name, &tier.DisplayName, &tier.MinRides, &tier.MinRating,
			&tier.CommissionRate, &tier.BonusMultiplier, &tier.PriorityDispatch,
			&tier.IconURL, &tier.ColorHex, &tier.Benefits, &tier.IsActive,
		)
		if err != nil {
			return nil, err
		}
		tiers = append(tiers, tier)
	}

	return tiers, nil
}

// ========================================
// QUESTS
// ========================================

// GetActiveQuests gets active quests for a driver
func (r *Repository) GetActiveQuests(ctx context.Context, tierID *uuid.UUID, city *string) ([]*DriverQuest, error) {
	query := `
		SELECT id, name, description, quest_type, target_value, reward_type, reward_value,
		       start_date, end_date, tier_restriction, city_restriction, max_participants,
		       current_count, is_active, is_featured, icon_url, created_at
		FROM driver_quests
		WHERE is_active = true
		  AND start_date <= NOW()
		  AND end_date >= NOW()
		  AND (tier_restriction IS NULL OR tier_restriction = $1 OR $1 IS NULL)
		  AND (city_restriction IS NULL OR city_restriction = $2 OR $2 IS NULL)
		  AND (max_participants IS NULL OR current_count < max_participants)
		ORDER BY is_featured DESC, end_date ASC
	`

	rows, err := r.db.Query(ctx, query, tierID, city)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quests []*DriverQuest
	for rows.Next() {
		q := &DriverQuest{}
		err := rows.Scan(
			&q.ID, &q.Name, &q.Description, &q.QuestType, &q.TargetValue, &q.RewardType, &q.RewardValue,
			&q.StartDate, &q.EndDate, &q.TierRestriction, &q.CityRestriction, &q.MaxParticipants,
			&q.CurrentCount, &q.IsActive, &q.IsFeatured, &q.IconURL, &q.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		quests = append(quests, q)
	}

	return quests, nil
}

// GetQuest gets a quest by ID
func (r *Repository) GetQuest(ctx context.Context, questID uuid.UUID) (*DriverQuest, error) {
	query := `
		SELECT id, name, description, quest_type, target_value, reward_type, reward_value,
		       start_date, end_date, tier_restriction, city_restriction, max_participants,
		       current_count, is_active, is_featured, icon_url, created_at
		FROM driver_quests
		WHERE id = $1
	`

	q := &DriverQuest{}
	err := r.db.QueryRow(ctx, query, questID).Scan(
		&q.ID, &q.Name, &q.Description, &q.QuestType, &q.TargetValue, &q.RewardType, &q.RewardValue,
		&q.StartDate, &q.EndDate, &q.TierRestriction, &q.CityRestriction, &q.MaxParticipants,
		&q.CurrentCount, &q.IsActive, &q.IsFeatured, &q.IconURL, &q.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return q, nil
}

// GetQuestsByType gets quests by type
func (r *Repository) GetQuestsByType(ctx context.Context, questType QuestType) ([]*DriverQuest, error) {
	query := `
		SELECT id, name, description, quest_type, target_value, reward_type, reward_value,
		       start_date, end_date, tier_restriction, city_restriction, max_participants,
		       current_count, is_active, is_featured, icon_url, created_at
		FROM driver_quests
		WHERE quest_type = $1 AND is_active = true
		  AND start_date <= NOW() AND end_date >= NOW()
	`

	rows, err := r.db.Query(ctx, query, questType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quests []*DriverQuest
	for rows.Next() {
		q := &DriverQuest{}
		err := rows.Scan(
			&q.ID, &q.Name, &q.Description, &q.QuestType, &q.TargetValue, &q.RewardType, &q.RewardValue,
			&q.StartDate, &q.EndDate, &q.TierRestriction, &q.CityRestriction, &q.MaxParticipants,
			&q.CurrentCount, &q.IsActive, &q.IsFeatured, &q.IconURL, &q.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		quests = append(quests, q)
	}

	return quests, nil
}

// ========================================
// QUEST PROGRESS
// ========================================

// GetQuestProgress gets a driver's progress on a quest
func (r *Repository) GetQuestProgress(ctx context.Context, driverID, questID uuid.UUID) (*DriverQuestProgress, error) {
	query := `
		SELECT id, driver_id, quest_id, current_value, status, completed_at, claimed_at, reward_amount, created_at, updated_at
		FROM driver_quest_progress
		WHERE driver_id = $1 AND quest_id = $2
	`

	progress := &DriverQuestProgress{}
	err := r.db.QueryRow(ctx, query, driverID, questID).Scan(
		&progress.ID, &progress.DriverID, &progress.QuestID, &progress.CurrentValue,
		&progress.Status, &progress.CompletedAt, &progress.ClaimedAt, &progress.RewardAmount,
		&progress.CreatedAt, &progress.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return progress, nil
}

// GetDriverActiveQuests gets all active quest progress for a driver
func (r *Repository) GetDriverActiveQuests(ctx context.Context, driverID uuid.UUID) ([]*DriverQuestProgress, error) {
	query := `
		SELECT dqp.id, dqp.driver_id, dqp.quest_id, dqp.current_value, dqp.status,
		       dqp.completed_at, dqp.claimed_at, dqp.reward_amount, dqp.created_at, dqp.updated_at,
		       dq.id, dq.name, dq.description, dq.quest_type, dq.target_value, dq.reward_type, dq.reward_value,
		       dq.start_date, dq.end_date, dq.is_featured, dq.icon_url
		FROM driver_quest_progress dqp
		JOIN driver_quests dq ON dqp.quest_id = dq.id
		WHERE dqp.driver_id = $1 AND dqp.status IN ('active', 'completed')
		ORDER BY dq.end_date ASC
	`

	rows, err := r.db.Query(ctx, query, driverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var progressList []*DriverQuestProgress
	for rows.Next() {
		progress := &DriverQuestProgress{}
		quest := &DriverQuest{}

		err := rows.Scan(
			&progress.ID, &progress.DriverID, &progress.QuestID, &progress.CurrentValue,
			&progress.Status, &progress.CompletedAt, &progress.ClaimedAt, &progress.RewardAmount,
			&progress.CreatedAt, &progress.UpdatedAt,
			&quest.ID, &quest.Name, &quest.Description, &quest.QuestType, &quest.TargetValue,
			&quest.RewardType, &quest.RewardValue, &quest.StartDate, &quest.EndDate,
			&quest.IsFeatured, &quest.IconURL,
		)
		if err != nil {
			return nil, err
		}

		progress.Quest = quest
		progressList = append(progressList, progress)
	}

	return progressList, nil
}

// CreateQuestProgress creates a new quest progress record
func (r *Repository) CreateQuestProgress(ctx context.Context, progress *DriverQuestProgress) error {
	query := `
		INSERT INTO driver_quest_progress (id, driver_id, quest_id, current_value, status)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.Exec(ctx, query,
		progress.ID, progress.DriverID, progress.QuestID, progress.CurrentValue, progress.Status,
	)

	return err
}

// UpdateQuestProgress updates quest progress
func (r *Repository) UpdateQuestProgress(ctx context.Context, progressID uuid.UUID, currentValue int, status QuestStatus) error {
	var completedAt *time.Time
	if status == QuestStatusCompleted {
		now := time.Now()
		completedAt = &now
	}

	query := `
		UPDATE driver_quest_progress
		SET current_value = $1, status = $2, completed_at = COALESCE($3, completed_at), updated_at = NOW()
		WHERE id = $4
	`

	_, err := r.db.Exec(ctx, query, currentValue, status, completedAt, progressID)
	return err
}

// ClaimQuestReward marks a quest as claimed
func (r *Repository) ClaimQuestReward(ctx context.Context, progressID uuid.UUID, rewardAmount float64) error {
	query := `
		UPDATE driver_quest_progress
		SET status = 'claimed', claimed_at = NOW(), reward_amount = $1, updated_at = NOW()
		WHERE id = $2 AND status = 'completed'
	`

	_, err := r.db.Exec(ctx, query, rewardAmount, progressID)
	return err
}

// ========================================
// ACHIEVEMENTS
// ========================================

// GetAchievement gets an achievement by ID
func (r *Repository) GetAchievement(ctx context.Context, achievementID uuid.UUID) (*Achievement, error) {
	query := `
		SELECT id, name, description, category, icon_url, points, criteria, is_secret, rarity, is_active
		FROM driver_achievements
		WHERE id = $1
	`

	a := &Achievement{}
	err := r.db.QueryRow(ctx, query, achievementID).Scan(
		&a.ID, &a.Name, &a.Description, &a.Category, &a.IconURL,
		&a.Points, &a.Criteria, &a.IsSecret, &a.Rarity, &a.IsActive,
	)

	if err != nil {
		return nil, err
	}

	return a, nil
}

// GetAllAchievements gets all achievements
func (r *Repository) GetAllAchievements(ctx context.Context) ([]*Achievement, error) {
	query := `
		SELECT id, name, description, category, icon_url, points, criteria, is_secret, rarity, is_active
		FROM driver_achievements
		WHERE is_active = true
		ORDER BY category, points DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var achievements []*Achievement
	for rows.Next() {
		a := &Achievement{}
		err := rows.Scan(
			&a.ID, &a.Name, &a.Description, &a.Category, &a.IconURL,
			&a.Points, &a.Criteria, &a.IsSecret, &a.Rarity, &a.IsActive,
		)
		if err != nil {
			return nil, err
		}
		achievements = append(achievements, a)
	}

	return achievements, nil
}

// GetDriverAchievements gets achievements earned by a driver
func (r *Repository) GetDriverAchievements(ctx context.Context, driverID uuid.UUID) ([]*DriverAchievement, error) {
	query := `
		SELECT da.id, da.driver_id, da.achievement_id, da.earned_at, da.notified_at,
		       a.id, a.name, a.description, a.category, a.icon_url, a.points, a.rarity
		FROM driver_achievement_progress da
		JOIN driver_achievements a ON da.achievement_id = a.id
		WHERE da.driver_id = $1
		ORDER BY da.earned_at DESC
	`

	rows, err := r.db.Query(ctx, query, driverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var achievements []*DriverAchievement
	for rows.Next() {
		da := &DriverAchievement{}
		a := &Achievement{}

		err := rows.Scan(
			&da.ID, &da.DriverID, &da.AchievementID, &da.EarnedAt, &da.NotifiedAt,
			&a.ID, &a.Name, &a.Description, &a.Category, &a.IconURL, &a.Points, &a.Rarity,
		)
		if err != nil {
			return nil, err
		}

		da.Achievement = a
		achievements = append(achievements, da)
	}

	return achievements, nil
}

// HasAchievement checks if a driver has an achievement
func (r *Repository) HasAchievement(ctx context.Context, driverID, achievementID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM driver_achievement_progress WHERE driver_id = $1 AND achievement_id = $2)`

	var exists bool
	err := r.db.QueryRow(ctx, query, driverID, achievementID).Scan(&exists)
	return exists, err
}

// AwardAchievement awards an achievement to a driver
func (r *Repository) AwardAchievement(ctx context.Context, driverID, achievementID uuid.UUID) error {
	query := `
		INSERT INTO driver_achievement_progress (id, driver_id, achievement_id, earned_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (driver_id, achievement_id) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, uuid.New(), driverID, achievementID)
	return err
}

// ========================================
// LEADERBOARD
// ========================================

// GetLeaderboard gets the leaderboard for a specific period and category
func (r *Repository) GetLeaderboard(ctx context.Context, period string, category string, limit int) ([]LeaderboardEntry, error) {
	var orderBy string
	switch category {
	case "rides":
		if period == "weekly" {
			orderBy = "dg.weekly_rides"
		} else if period == "monthly" {
			orderBy = "dg.monthly_rides"
		} else {
			orderBy = "dg.total_rides"
		}
	case "earnings":
		if period == "weekly" {
			orderBy = "dg.weekly_earnings"
		} else if period == "monthly" {
			orderBy = "dg.monthly_earnings"
		} else {
			orderBy = "dg.total_earnings"
		}
	case "rating":
		orderBy = "dg.average_rating"
	default:
		orderBy = "dg.total_rides"
	}

	query := `
		SELECT dg.driver_id, u.first_name || ' ' || LEFT(u.last_name, 1) || '.' as driver_name,
		       u.profile_photo_url, dt.name as tier_name,
		       CASE
		           WHEN $2 = 'rides' AND $1 = 'weekly' THEN dg.weekly_rides::float8
		           WHEN $2 = 'rides' AND $1 = 'monthly' THEN dg.monthly_rides::float8
		           WHEN $2 = 'rides' THEN dg.total_rides::float8
		           WHEN $2 = 'earnings' AND $1 = 'weekly' THEN dg.weekly_earnings
		           WHEN $2 = 'earnings' AND $1 = 'monthly' THEN dg.monthly_earnings
		           WHEN $2 = 'earnings' THEN dg.total_earnings
		           WHEN $2 = 'rating' THEN dg.average_rating
		           ELSE dg.total_rides::float8
		       END as value
		FROM driver_gamification dg
		JOIN drivers d ON dg.driver_id = d.id
		JOIN users u ON d.user_id = u.id
		LEFT JOIN driver_tiers dt ON dg.current_tier_id = dt.id
		ORDER BY ` + orderBy + ` DESC
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, query, period, category, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	rank := 1
	for rows.Next() {
		entry := LeaderboardEntry{Rank: rank}
		err := rows.Scan(
			&entry.DriverID, &entry.DriverName, &entry.ProfilePhotoURL,
			&entry.TierName, &entry.Value,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
		rank++
	}

	return entries, nil
}

// GetDriverLeaderboardPosition gets a driver's position in the leaderboard
func (r *Repository) GetDriverLeaderboardPosition(ctx context.Context, driverID uuid.UUID, period string, category string) (*LeaderboardEntry, error) {
	var orderBy string
	switch category {
	case "rides":
		if period == "weekly" {
			orderBy = "weekly_rides"
		} else if period == "monthly" {
			orderBy = "monthly_rides"
		} else {
			orderBy = "total_rides"
		}
	case "earnings":
		if period == "weekly" {
			orderBy = "weekly_earnings"
		} else if period == "monthly" {
			orderBy = "monthly_earnings"
		} else {
			orderBy = "total_earnings"
		}
	case "rating":
		orderBy = "average_rating"
	default:
		orderBy = "total_rides"
	}

	query := `
		WITH ranked AS (
			SELECT driver_id,
			       RANK() OVER (ORDER BY ` + orderBy + ` DESC) as rank,
			       ` + orderBy + ` as value
			FROM driver_gamification
		)
		SELECT r.rank, r.value, u.first_name || ' ' || LEFT(u.last_name, 1) || '.' as driver_name,
		       u.profile_photo_url, dt.name as tier_name
		FROM ranked r
		JOIN drivers d ON r.driver_id = d.id
		JOIN users u ON d.user_id = u.id
		JOIN driver_gamification dg ON r.driver_id = dg.driver_id
		LEFT JOIN driver_tiers dt ON dg.current_tier_id = dt.id
		WHERE r.driver_id = $1
	`

	entry := &LeaderboardEntry{DriverID: driverID, IsCurrentDriver: true}
	err := r.db.QueryRow(ctx, query, driverID).Scan(
		&entry.Rank, &entry.Value, &entry.DriverName,
		&entry.ProfilePhotoURL, &entry.TierName,
	)

	if err != nil {
		return nil, err
	}

	return entry, nil
}
