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

// GetDriverIDByUserID looks up the drivers.id for a given users.id
func (r *Repository) GetDriverIDByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var driverID uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT id FROM drivers WHERE user_id = $1`, userID).Scan(&driverID)
	return driverID, err
}

// GetUserIDByDriverID looks up the users.id for a given drivers.id
func (r *Repository) GetUserIDByDriverID(ctx context.Context, driverID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT user_id FROM drivers WHERE id = $1`, driverID).Scan(&userID)
	return userID, err
}

// GetWeeklyStats queries actual weekly ride stats for a driver.
// rides.driver_id stores users.id, so userID is required.
func (r *Repository) GetWeeklyStats(ctx context.Context, userID uuid.UUID) (*WeeklyStats, error) {
	query := `
		SELECT
			COUNT(*)::int                                      AS rides,
			COALESCE(SUM(final_fare * (1 - 0.20)), 0)::float8 AS earnings,
			COALESCE(AVG(NULLIF(rating, 0)), 0)::float8        AS average_rating
		FROM rides
		WHERE driver_id = $1
		  AND status    = 'completed'
		  AND completed_at >= NOW() - INTERVAL '7 days'
	`

	stats := &WeeklyStats{}
	err := r.db.QueryRow(ctx, query, userID).Scan(&stats.Rides, &stats.Earnings, &stats.AverageRating)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// GetDriverGamification gets a driver's gamification profile
func (r *Repository) GetDriverGamification(ctx context.Context, driverID uuid.UUID) (*DriverGamification, error) {
	query := `
		SELECT dg.driver_id, dg.current_tier_id, dg.total_points, dg.weekly_points, dg.monthly_points,
		       dg.current_streak, dg.longest_streak, dg.acceptance_streak, dg.five_star_streak,
		       dg.total_quests_completed, dg.total_challenges_won, dg.total_bonuses_earned,
		       dg.last_active_date, dg.tier_evaluated_at, dg.created_at, dg.updated_at,
		       dt.id, dt.name, dt.display_name, dt.min_rides, dt.min_rating,
		       dt.commission_rate, dt.surge_multiplier_bonus, dt.priority_dispatch, dt.benefits,
		       dt.is_active, dt.icon_url, dt.color_hex, dt.created_at
		FROM driver_gamification dg
		LEFT JOIN driver_tiers dt ON dg.current_tier_id = dt.id
		WHERE dg.driver_id = $1
	`

	profile := &DriverGamification{}
	var tier DriverTier
	var tierID *uuid.UUID

	err := r.db.QueryRow(ctx, query, driverID).Scan(
		&profile.DriverID, &profile.CurrentTierID, &profile.TotalPoints, &profile.WeeklyPoints, &profile.MonthlyPoints,
		&profile.CurrentStreak, &profile.LongestStreak, &profile.AcceptanceStreak, &profile.FiveStarStreak,
		&profile.TotalQuestsCompleted, &profile.TotalChallengesWon, &profile.TotalBonusEarned,
		&profile.LastActiveDate, &profile.TierEvaluatedAt, &profile.CreatedAt, &profile.UpdatedAt,
		&tierID, &tier.Name, &tier.DisplayName, &tier.MinRides, &tier.MinRating,
		&tier.CommissionRate, &tier.BonusMultiplier, &tier.PriorityDispatch, &tier.Benefits,
		&tier.IsActive, &tier.IconURL, &tier.ColorHex, &tier.CreatedAt,
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
		INSERT INTO driver_gamification (driver_id, current_tier_id)
		VALUES ($1, $2)
		ON CONFLICT (driver_id) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, profile.DriverID, profile.CurrentTierID)
	return err
}

// UpdateDriverStats updates driver statistics after a ride.
// The gamification DB tracks points-based metrics; ride counts/earnings are queried live from the rides table.
// This increments total_points and weekly/monthly_points as a proxy for ride activity.
func (r *Repository) UpdateDriverStats(ctx context.Context, driverID uuid.UUID, rides int, earnings float64, rating float64) error {
	query := `
		UPDATE driver_gamification
		SET total_points   = total_points   + $1,
		    weekly_points  = weekly_points  + $1,
		    monthly_points = monthly_points + $1,
		    last_active_date = CURRENT_DATE,
		    updated_at = NOW()
		WHERE driver_id = $2
	`

	_, err := r.db.Exec(ctx, query, rides, driverID)
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
		SET total_quests_completed = total_quests_completed + 1, updated_at = NOW()
		WHERE driver_id = $1
	`

	_, err := r.db.Exec(ctx, query, driverID)
	return err
}

// AddBonusEarned adds bonus to driver's total
func (r *Repository) AddBonusEarned(ctx context.Context, driverID uuid.UUID, bonus float64) error {
	query := `
		UPDATE driver_gamification
		SET total_bonuses_earned = total_bonuses_earned + $1, updated_at = NOW()
		WHERE driver_id = $2
	`

	_, err := r.db.Exec(ctx, query, bonus, driverID)
	return err
}

// AddAchievementPoints adds achievement points (stored as total_points)
func (r *Repository) AddAchievementPoints(ctx context.Context, driverID uuid.UUID, points int) error {
	query := `
		UPDATE driver_gamification
		SET total_points = total_points + $1, updated_at = NOW()
		WHERE driver_id = $2
	`

	_, err := r.db.Exec(ctx, query, points, driverID)
	return err
}

// ResetWeeklyStats resets weekly point counters
func (r *Repository) ResetWeeklyStats(ctx context.Context) error {
	query := `UPDATE driver_gamification SET weekly_points = 0, updated_at = NOW()`
	_, err := r.db.Exec(ctx, query)
	return err
}

// ResetMonthlyStats resets monthly point counters
func (r *Repository) ResetMonthlyStats(ctx context.Context) error {
	query := `UPDATE driver_gamification SET monthly_points = 0, updated_at = NOW()`
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
		       surge_multiplier_bonus, priority_dispatch, icon_url, color_hex, benefits, is_active
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
		       surge_multiplier_bonus, priority_dispatch, icon_url, color_hex, benefits, is_active
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
		       surge_multiplier_bonus, priority_dispatch, icon_url, color_hex, benefits, is_active
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
		SELECT id, name, description, quest_type, target_value, time_window_hours,
		       reward_type, reward_value, start_time, end_time, min_tier_id,
		       region_id, city_id, vehicle_type, max_participants, current_participants,
		       is_guaranteed, is_active, created_at
		FROM driver_quests
		WHERE is_active = true
		  AND start_time <= NOW()
		  AND end_time >= NOW()
		  AND (min_tier_id IS NULL OR min_tier_id = $1 OR $1 IS NULL)
		  AND (max_participants IS NULL OR current_participants < max_participants)
		ORDER BY end_time ASC
	`

	rows, err := r.db.Query(ctx, query, tierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quests []*DriverQuest
	for rows.Next() {
		q := &DriverQuest{}
		err := rows.Scan(
			&q.ID, &q.Name, &q.Description, &q.QuestType, &q.TargetValue, &q.TimeWindowHours,
			&q.RewardType, &q.RewardValue, &q.StartTime, &q.EndTime, &q.MinTierID,
			&q.RegionID, &q.CityID, &q.VehicleType, &q.MaxParticipants, &q.CurrentCount,
			&q.IsGuaranteed, &q.IsActive, &q.CreatedAt,
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
		SELECT id, name, description, quest_type, target_value, time_window_hours,
		       reward_type, reward_value, start_time, end_time, min_tier_id,
		       region_id, city_id, vehicle_type, max_participants, current_participants,
		       is_guaranteed, is_active, created_at
		FROM driver_quests
		WHERE id = $1
	`

	q := &DriverQuest{}
	err := r.db.QueryRow(ctx, query, questID).Scan(
		&q.ID, &q.Name, &q.Description, &q.QuestType, &q.TargetValue, &q.TimeWindowHours,
		&q.RewardType, &q.RewardValue, &q.StartTime, &q.EndTime, &q.MinTierID,
		&q.RegionID, &q.CityID, &q.VehicleType, &q.MaxParticipants, &q.CurrentCount,
		&q.IsGuaranteed, &q.IsActive, &q.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return q, nil
}

// GetQuestsByType gets quests by type
func (r *Repository) GetQuestsByType(ctx context.Context, questType QuestType) ([]*DriverQuest, error) {
	query := `
		SELECT id, name, description, quest_type, target_value, time_window_hours,
		       reward_type, reward_value, start_time, end_time, min_tier_id,
		       region_id, city_id, vehicle_type, max_participants, current_participants,
		       is_guaranteed, is_active, created_at
		FROM driver_quests
		WHERE quest_type = $1 AND is_active = true
		  AND start_time <= NOW() AND end_time >= NOW()
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
			&q.ID, &q.Name, &q.Description, &q.QuestType, &q.TargetValue, &q.TimeWindowHours,
			&q.RewardType, &q.RewardValue, &q.StartTime, &q.EndTime, &q.MinTierID,
			&q.RegionID, &q.CityID, &q.VehicleType, &q.MaxParticipants, &q.CurrentCount,
			&q.IsGuaranteed, &q.IsActive, &q.CreatedAt,
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
		SELECT id, driver_id, quest_id, current_value, target_value, progress_percent,
		       completed, completed_at, reward_paid, reward_paid_at, reward_amount, started_at, updated_at
		FROM driver_quest_progress
		WHERE driver_id = $1 AND quest_id = $2
	`

	progress := &DriverQuestProgress{}
	err := r.db.QueryRow(ctx, query, driverID, questID).Scan(
		&progress.ID, &progress.DriverID, &progress.QuestID, &progress.CurrentValue,
		&progress.TargetValue, &progress.ProgressPercent, &progress.Completed,
		&progress.CompletedAt, &progress.RewardPaid, &progress.RewardPaidAt,
		&progress.RewardAmount, &progress.StartedAt, &progress.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return progress, nil
}

// GetDriverActiveQuests gets all active quest progress for a driver
func (r *Repository) GetDriverActiveQuests(ctx context.Context, driverID uuid.UUID) ([]*DriverQuestProgress, error) {
	query := `
		SELECT dqp.id, dqp.driver_id, dqp.quest_id, dqp.current_value, dqp.target_value,
		       dqp.progress_percent, dqp.completed, dqp.completed_at, dqp.reward_paid,
		       dqp.reward_paid_at, dqp.reward_amount, dqp.started_at, dqp.updated_at,
		       dq.id, dq.name, dq.description, dq.quest_type, dq.target_value,
		       dq.reward_type, dq.reward_value, dq.start_time, dq.end_time
		FROM driver_quest_progress dqp
		JOIN driver_quests dq ON dqp.quest_id = dq.id
		WHERE dqp.driver_id = $1 AND dqp.completed = false
		ORDER BY dq.end_time ASC
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
			&progress.TargetValue, &progress.ProgressPercent, &progress.Completed,
			&progress.CompletedAt, &progress.RewardPaid, &progress.RewardPaidAt,
			&progress.RewardAmount, &progress.StartedAt, &progress.UpdatedAt,
			&quest.ID, &quest.Name, &quest.Description, &quest.QuestType, &quest.TargetValue,
			&quest.RewardType, &quest.RewardValue, &quest.StartTime, &quest.EndTime,
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
		INSERT INTO driver_quest_progress (id, driver_id, quest_id, current_value, target_value)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (driver_id, quest_id) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query,
		progress.ID, progress.DriverID, progress.QuestID, progress.CurrentValue, progress.TargetValue,
	)

	return err
}

// UpdateQuestProgress updates quest progress
func (r *Repository) UpdateQuestProgress(ctx context.Context, progressID uuid.UUID, currentValue int, status QuestStatus) error {
	completed := status == QuestStatusCompleted
	var completedAt *time.Time
	if completed {
		now := time.Now()
		completedAt = &now
	}

	query := `
		UPDATE driver_quest_progress
		SET current_value = $1,
		    progress_percent = LEAST(($1::numeric / NULLIF(target_value, 0)) * 100, 100),
		    completed = $2,
		    completed_at = COALESCE($3, completed_at),
		    updated_at = NOW()
		WHERE id = $4
	`

	_, err := r.db.Exec(ctx, query, currentValue, completed, completedAt, progressID)
	return err
}

// ClaimQuestReward marks a quest reward as paid
func (r *Repository) ClaimQuestReward(ctx context.Context, progressID uuid.UUID, rewardAmount float64) error {
	query := `
		UPDATE driver_quest_progress
		SET reward_paid = true, reward_paid_at = NOW(), reward_amount = $1, updated_at = NOW()
		WHERE id = $2 AND completed = true
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

// GetLeaderboard gets the leaderboard for a specific period and category.
// Uses points-based columns that exist in the DB schema.
func (r *Repository) GetLeaderboard(ctx context.Context, period string, category string, limit int) ([]LeaderboardEntry, error) {
	var orderBy string
	switch category {
	case "earnings", "rides":
		if period == "weekly" {
			orderBy = "dg.weekly_points"
		} else if period == "monthly" {
			orderBy = "dg.monthly_points"
		} else {
			orderBy = "dg.total_points"
		}
	default:
		orderBy = "dg.total_points"
	}

	query := `
		SELECT dg.driver_id, u.first_name || ' ' || LEFT(u.last_name, 1) || '.' as driver_name,
		       u.profile_image, dt.name as tier_name,
		       CASE
		           WHEN $1 = 'weekly'  THEN dg.weekly_points::float8
		           WHEN $1 = 'monthly' THEN dg.monthly_points::float8
		           ELSE dg.total_points::float8
		       END as value
		FROM driver_gamification dg
		JOIN drivers d ON dg.driver_id = d.id
		JOIN users u ON d.user_id = u.id
		LEFT JOIN driver_tiers dt ON dg.current_tier_id = dt.id
		ORDER BY ` + orderBy + ` DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, period, limit)
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

// GetDriverLeaderboardPosition gets a driver's position in the leaderboard.
// Uses points-based columns that exist in the DB schema.
func (r *Repository) GetDriverLeaderboardPosition(ctx context.Context, driverID uuid.UUID, period string, category string) (*LeaderboardEntry, error) {
	var orderBy string
	switch period {
	case "weekly":
		orderBy = "weekly_points"
	case "monthly":
		orderBy = "monthly_points"
	default:
		orderBy = "total_points"
	}

	query := `
		WITH ranked AS (
			SELECT driver_id,
			       RANK() OVER (ORDER BY ` + orderBy + ` DESC) as rank,
			       ` + orderBy + `::float8 as value
			FROM driver_gamification
		)
		SELECT r.rank, r.value, u.first_name || ' ' || LEFT(u.last_name, 1) || '.' as driver_name,
		       u.profile_image, dt.name as tier_name
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
