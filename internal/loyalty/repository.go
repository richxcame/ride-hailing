package loyalty

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for loyalty
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new loyalty repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// RIDER LOYALTY ACCOUNT
// ========================================

// GetRiderLoyalty gets a rider's loyalty account
func (r *Repository) GetRiderLoyalty(ctx context.Context, riderID uuid.UUID) (*RiderLoyalty, error) {
	query := `
		SELECT rl.rider_id, rl.current_tier_id, rl.total_points, rl.available_points,
		       rl.lifetime_points, rl.tier_points, rl.tier_period_start, rl.tier_period_end,
		       rl.streak_days, rl.last_ride_date, rl.free_cancellations_used, rl.free_upgrades_used,
		       rl.joined_at, rl.created_at, rl.updated_at,
		       lt.id, lt.name, lt.min_points, lt.multiplier, lt.benefits,
		       lt.free_cancellations, lt.free_upgrades, lt.priority_support
		FROM rider_loyalty rl
		LEFT JOIN loyalty_tiers lt ON rl.current_tier_id = lt.id
		WHERE rl.rider_id = $1
	`

	account := &RiderLoyalty{}
	var tier LoyaltyTier
	var tierID *uuid.UUID

	err := r.db.QueryRow(ctx, query, riderID).Scan(
		&account.RiderID, &account.CurrentTierID, &account.TotalPoints, &account.AvailablePoints,
		&account.LifetimePoints, &account.TierPoints, &account.TierPeriodStart, &account.TierPeriodEnd,
		&account.StreakDays, &account.LastRideDate, &account.FreeCancellationsUsed, &account.FreeUpgradesUsed,
		&account.JoinedAt, &account.CreatedAt, &account.UpdatedAt,
		&tierID, &tier.Name, &tier.MinPoints, &tier.Multiplier, &tier.Benefits,
		&tier.FreeCancellations, &tier.FreeUpgrades, &tier.PrioritySupport,
	)

	if err != nil {
		return nil, err
	}

	if tierID != nil {
		tier.ID = *tierID
		account.CurrentTier = &tier
	}

	return account, nil
}

// CreateRiderLoyalty creates a new rider loyalty account
func (r *Repository) CreateRiderLoyalty(ctx context.Context, account *RiderLoyalty) error {
	query := `
		INSERT INTO rider_loyalty (
			rider_id, current_tier_id, total_points, available_points,
			lifetime_points, tier_points, tier_period_start, tier_period_end, joined_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.Exec(ctx, query,
		account.RiderID, account.CurrentTierID, account.TotalPoints, account.AvailablePoints,
		account.LifetimePoints, account.TierPoints, account.TierPeriodStart, account.TierPeriodEnd,
		account.JoinedAt,
	)

	return err
}

// UpdatePoints updates a rider's points balance
func (r *Repository) UpdatePoints(ctx context.Context, riderID uuid.UUID, earnedPoints, tierPoints int) error {
	query := `
		UPDATE rider_loyalty
		SET available_points = available_points + $1,
		    total_points = total_points + $1,
		    lifetime_points = lifetime_points + $1,
		    tier_points = tier_points + $2,
		    updated_at = NOW()
		WHERE rider_id = $3
	`

	_, err := r.db.Exec(ctx, query, earnedPoints, tierPoints, riderID)
	return err
}

// DeductPoints deducts points from a rider's balance
func (r *Repository) DeductPoints(ctx context.Context, riderID uuid.UUID, points int) error {
	query := `
		UPDATE rider_loyalty
		SET available_points = available_points - $1,
		    updated_at = NOW()
		WHERE rider_id = $2 AND available_points >= $1
	`

	result, err := r.db.Exec(ctx, query, points, riderID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// UpdateTier updates a rider's tier
func (r *Repository) UpdateTier(ctx context.Context, riderID uuid.UUID, tierID uuid.UUID) error {
	query := `
		UPDATE rider_loyalty
		SET current_tier_id = $1,
		    free_cancellations_used = 0,
		    free_upgrades_used = 0,
		    updated_at = NOW()
		WHERE rider_id = $2
	`

	_, err := r.db.Exec(ctx, query, tierID, riderID)
	return err
}

// UpdateStreak updates a rider's streak
func (r *Repository) UpdateStreak(ctx context.Context, riderID uuid.UUID, streakDays int) error {
	query := `
		UPDATE rider_loyalty
		SET streak_days = $1,
		    last_ride_date = NOW(),
		    updated_at = NOW()
		WHERE rider_id = $2
	`

	_, err := r.db.Exec(ctx, query, streakDays, riderID)
	return err
}

// ========================================
// LOYALTY TIERS
// ========================================

// GetTier gets a loyalty tier by ID
func (r *Repository) GetTier(ctx context.Context, tierID uuid.UUID) (*LoyaltyTier, error) {
	query := `
		SELECT id, name, min_points, multiplier, benefits,
		       free_cancellations, free_upgrades, priority_support
		FROM loyalty_tiers
		WHERE id = $1
	`

	tier := &LoyaltyTier{}
	err := r.db.QueryRow(ctx, query, tierID).Scan(
		&tier.ID, &tier.Name, &tier.MinPoints, &tier.Multiplier, &tier.Benefits,
		&tier.FreeCancellations, &tier.FreeUpgrades, &tier.PrioritySupport,
	)

	if err != nil {
		return nil, err
	}

	return tier, nil
}

// GetTierByName gets a loyalty tier by name
func (r *Repository) GetTierByName(ctx context.Context, name TierName) (*LoyaltyTier, error) {
	query := `
		SELECT id, name, min_points, multiplier, benefits,
		       free_cancellations, free_upgrades, priority_support
		FROM loyalty_tiers
		WHERE name = $1
	`

	tier := &LoyaltyTier{}
	err := r.db.QueryRow(ctx, query, string(name)).Scan(
		&tier.ID, &tier.Name, &tier.MinPoints, &tier.Multiplier, &tier.Benefits,
		&tier.FreeCancellations, &tier.FreeUpgrades, &tier.PrioritySupport,
	)

	if err != nil {
		return nil, err
	}

	return tier, nil
}

// GetAllTiers gets all loyalty tiers ordered by min_points
func (r *Repository) GetAllTiers(ctx context.Context) ([]*LoyaltyTier, error) {
	query := `
		SELECT id, name, min_points, multiplier, benefits,
		       free_cancellations, free_upgrades, priority_support
		FROM loyalty_tiers
		ORDER BY min_points ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tiers []*LoyaltyTier
	for rows.Next() {
		tier := &LoyaltyTier{}
		err := rows.Scan(
			&tier.ID, &tier.Name, &tier.MinPoints, &tier.Multiplier, &tier.Benefits,
			&tier.FreeCancellations, &tier.FreeUpgrades, &tier.PrioritySupport,
		)
		if err != nil {
			return nil, err
		}
		tiers = append(tiers, tier)
	}

	return tiers, nil
}

// ========================================
// POINTS TRANSACTIONS
// ========================================

// CreatePointsTransaction creates a new points transaction
func (r *Repository) CreatePointsTransaction(ctx context.Context, tx *PointsTransaction) error {
	query := `
		INSERT INTO loyalty_points_transactions (
			id, rider_id, transaction_type, points, balance_after,
			source, source_id, description, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.Exec(ctx, query,
		tx.ID, tx.RiderID, tx.TransactionType, tx.Points, tx.BalanceAfter,
		tx.Source, tx.SourceID, tx.Description, tx.ExpiresAt,
	)

	return err
}

// GetPointsHistory gets points transaction history for a rider
func (r *Repository) GetPointsHistory(ctx context.Context, riderID uuid.UUID, limit, offset int) ([]*PointsTransaction, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM loyalty_points_transactions WHERE rider_id = $1`
	var total int
	err := r.db.QueryRow(ctx, countQuery, riderID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get transactions
	query := `
		SELECT id, rider_id, transaction_type, points, balance_after,
		       source, source_id, description, expires_at, created_at
		FROM loyalty_points_transactions
		WHERE rider_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, riderID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var transactions []*PointsTransaction
	for rows.Next() {
		tx := &PointsTransaction{}
		err := rows.Scan(
			&tx.ID, &tx.RiderID, &tx.TransactionType, &tx.Points, &tx.BalanceAfter,
			&tx.Source, &tx.SourceID, &tx.Description, &tx.ExpiresAt, &tx.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		transactions = append(transactions, tx)
	}

	return transactions, total, nil
}

// ========================================
// REWARDS
// ========================================

// GetReward gets a reward by ID
func (r *Repository) GetReward(ctx context.Context, rewardID uuid.UUID) (*RewardCatalogItem, error) {
	query := `
		SELECT id, name, description, reward_type, points_required, value,
		       tier_restriction, max_redemptions_per_user, redeemed_count, total_available,
		       valid_days, partner_name, partner_logo_url, is_active, created_at
		FROM loyalty_rewards
		WHERE id = $1
	`

	reward := &RewardCatalogItem{}
	err := r.db.QueryRow(ctx, query, rewardID).Scan(
		&reward.ID, &reward.Name, &reward.Description, &reward.RewardType, &reward.PointsRequired,
		&reward.Value, &reward.TierRestriction, &reward.MaxRedemptionsPerUser,
		&reward.RedeemedCount, &reward.TotalAvailable, &reward.ValidDays, &reward.PartnerName,
		&reward.PartnerLogoURL, &reward.IsActive, &reward.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return reward, nil
}

// GetAvailableRewards gets available rewards for a tier
func (r *Repository) GetAvailableRewards(ctx context.Context, tierID *uuid.UUID) ([]*RewardCatalogItem, error) {
	query := `
		SELECT id, name, description, reward_type, points_required, value,
		       tier_restriction, max_redemptions_per_user, redeemed_count, total_available,
		       valid_days, partner_name, partner_logo_url, is_active, created_at
		FROM loyalty_rewards
		WHERE is_active = true
		  AND (total_available IS NULL OR redeemed_count < total_available)
		  AND (tier_restriction IS NULL OR tier_restriction = $1 OR $1 IS NULL)
		ORDER BY points_required ASC
	`

	rows, err := r.db.Query(ctx, query, tierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rewards []*RewardCatalogItem
	for rows.Next() {
		reward := &RewardCatalogItem{}
		err := rows.Scan(
			&reward.ID, &reward.Name, &reward.Description, &reward.RewardType, &reward.PointsRequired,
			&reward.Value, &reward.TierRestriction, &reward.MaxRedemptionsPerUser,
			&reward.RedeemedCount, &reward.TotalAvailable, &reward.ValidDays, &reward.PartnerName,
			&reward.PartnerLogoURL, &reward.IsActive, &reward.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		rewards = append(rewards, reward)
	}

	return rewards, nil
}

// GetUserRedemptionCount gets the number of times a user has redeemed a specific reward
func (r *Repository) GetUserRedemptionCount(ctx context.Context, riderID, rewardID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*) FROM loyalty_redemptions
		WHERE rider_id = $1 AND reward_id = $2
	`

	var count int
	err := r.db.QueryRow(ctx, query, riderID, rewardID).Scan(&count)
	return count, err
}

// CreateRedemption creates a new redemption record
func (r *Repository) CreateRedemption(ctx context.Context, redemption *Redemption) error {
	query := `
		INSERT INTO loyalty_redemptions (
			id, rider_id, reward_id, points_spent, redemption_code, status, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(ctx, query,
		redemption.ID, redemption.RiderID, redemption.RewardID, redemption.PointsSpent,
		redemption.RedemptionCode, redemption.Status, redemption.ExpiresAt,
	)

	return err
}

// IncrementRewardRedemptionCount increments the redemption count for a reward
func (r *Repository) IncrementRewardRedemptionCount(ctx context.Context, rewardID uuid.UUID) error {
	query := `
		UPDATE loyalty_rewards
		SET redeemed_count = redeemed_count + 1, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, rewardID)
	return err
}

// ========================================
// CHALLENGES
// ========================================

// GetActiveChallenges gets active challenges for a rider
func (r *Repository) GetActiveChallenges(ctx context.Context, tierID *uuid.UUID) ([]*RiderChallenge, error) {
	query := `
		SELECT id, name, description, challenge_type, target_value, reward_points,
		       start_date, end_date, tier_restriction, is_active, created_at
		FROM rider_challenges
		WHERE is_active = true
		  AND start_date <= NOW()
		  AND end_date >= NOW()
		  AND (tier_restriction IS NULL OR tier_restriction = $1 OR $1 IS NULL)
		ORDER BY end_date ASC
	`

	rows, err := r.db.Query(ctx, query, tierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var challenges []*RiderChallenge
	for rows.Next() {
		c := &RiderChallenge{}
		err := rows.Scan(
			&c.ID, &c.Name, &c.Description, &c.ChallengeType, &c.TargetValue, &c.RewardPoints,
			&c.StartDate, &c.EndDate, &c.TierRestriction, &c.IsActive, &c.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		challenges = append(challenges, c)
	}

	return challenges, nil
}

// GetActiveChallengesByType gets active challenges by type
func (r *Repository) GetActiveChallengesByType(ctx context.Context, challengeType string, tierID *uuid.UUID) ([]*RiderChallenge, error) {
	query := `
		SELECT id, name, description, challenge_type, target_value, reward_points,
		       start_date, end_date, tier_restriction, is_active, created_at
		FROM rider_challenges
		WHERE is_active = true
		  AND challenge_type = $1
		  AND start_date <= NOW()
		  AND end_date >= NOW()
		  AND (tier_restriction IS NULL OR tier_restriction = $2 OR $2 IS NULL)
	`

	rows, err := r.db.Query(ctx, query, challengeType, tierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var challenges []*RiderChallenge
	for rows.Next() {
		c := &RiderChallenge{}
		err := rows.Scan(
			&c.ID, &c.Name, &c.Description, &c.ChallengeType, &c.TargetValue, &c.RewardPoints,
			&c.StartDate, &c.EndDate, &c.TierRestriction, &c.IsActive, &c.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		challenges = append(challenges, c)
	}

	return challenges, nil
}

// GetChallengeProgress gets a rider's progress on a challenge
func (r *Repository) GetChallengeProgress(ctx context.Context, riderID, challengeID uuid.UUID) (*ChallengeProgress, error) {
	query := `
		SELECT id, rider_id, challenge_id, current_value, completed, completed_at, created_at
		FROM rider_challenge_progress
		WHERE rider_id = $1 AND challenge_id = $2
	`

	progress := &ChallengeProgress{}
	err := r.db.QueryRow(ctx, query, riderID, challengeID).Scan(
		&progress.ID, &progress.RiderID, &progress.ChallengeID,
		&progress.CurrentValue, &progress.Completed, &progress.CompletedAt, &progress.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return progress, nil
}

// CreateChallengeProgress creates a new challenge progress record
func (r *Repository) CreateChallengeProgress(ctx context.Context, progress *ChallengeProgress) error {
	query := `
		INSERT INTO rider_challenge_progress (id, rider_id, challenge_id, current_value, completed)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.Exec(ctx, query,
		progress.ID, progress.RiderID, progress.ChallengeID, progress.CurrentValue, progress.Completed,
	)

	return err
}

// UpdateChallengeProgress updates challenge progress
func (r *Repository) UpdateChallengeProgress(ctx context.Context, progressID uuid.UUID, currentValue int, completed bool) error {
	var completedAt *time.Time
	if completed {
		now := time.Now()
		completedAt = &now
	}

	query := `
		UPDATE rider_challenge_progress
		SET current_value = $1, completed = $2, completed_at = $3, updated_at = NOW()
		WHERE id = $4
	`

	_, err := r.db.Exec(ctx, query, currentValue, completed, completedAt, progressID)
	return err
}

// ========================================
// ADMIN OPERATIONS
// ========================================

// GetLoyaltyStats gets loyalty program statistics
func (r *Repository) GetLoyaltyStats(ctx context.Context) (*LoyaltyStats, error) {
	query := `
		SELECT
			COUNT(*) as total_members,
			COUNT(*) FILTER (WHERE lt.name = 'bronze') as bronze_count,
			COUNT(*) FILTER (WHERE lt.name = 'silver') as silver_count,
			COUNT(*) FILTER (WHERE lt.name = 'gold') as gold_count,
			COUNT(*) FILTER (WHERE lt.name = 'platinum') as platinum_count,
			COUNT(*) FILTER (WHERE lt.name = 'diamond') as diamond_count,
			SUM(rl.lifetime_points) as total_points_earned,
			SUM(rl.available_points) as total_points_outstanding
		FROM rider_loyalty rl
		LEFT JOIN loyalty_tiers lt ON rl.current_tier_id = lt.id
	`

	stats := &LoyaltyStats{}
	err := r.db.QueryRow(ctx, query).Scan(
		&stats.TotalMembers, &stats.BronzeCount, &stats.SilverCount,
		&stats.GoldCount, &stats.PlatinumCount, &stats.DiamondCount,
		&stats.TotalPointsEarned, &stats.TotalPointsOutstanding,
	)

	if err != nil {
		return nil, err
	}

	return stats, nil
}

// LoyaltyStats represents loyalty program statistics
type LoyaltyStats struct {
	TotalMembers           int   `json:"total_members"`
	BronzeCount            int   `json:"bronze_count"`
	SilverCount            int   `json:"silver_count"`
	GoldCount              int   `json:"gold_count"`
	PlatinumCount          int   `json:"platinum_count"`
	DiamondCount           int   `json:"diamond_count"`
	TotalPointsEarned      int64 `json:"total_points_earned"`
	TotalPointsOutstanding int64 `json:"total_points_outstanding"`
}
