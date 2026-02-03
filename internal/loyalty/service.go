package loyalty

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// Service handles loyalty business logic
type Service struct {
	repo *Repository
}

// NewService creates a new loyalty service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ========================================
// LOYALTY ACCOUNT MANAGEMENT
// ========================================

// GetOrCreateLoyaltyAccount gets or creates a loyalty account for a rider
func (s *Service) GetOrCreateLoyaltyAccount(ctx context.Context, riderID uuid.UUID) (*RiderLoyalty, error) {
	account, err := s.repo.GetRiderLoyalty(ctx, riderID)
	if err == nil {
		return account, nil
	}

	// Create new account
	bronzeTier, err := s.repo.GetTierByName(ctx, TierBronze)
	if err != nil {
		return nil, common.NewInternalServerError("failed to get default tier")
	}

	account = &RiderLoyalty{
		RiderID:         riderID,
		CurrentTierID:   &bronzeTier.ID,
		TotalPoints:     0,
		AvailablePoints: 0,
		LifetimePoints:  0,
		TierPoints:      0,
		TierPeriodStart: time.Now(),
		TierPeriodEnd:   time.Now().AddDate(1, 0, 0),
		JoinedAt:        time.Now(),
	}

	if err := s.repo.CreateRiderLoyalty(ctx, account); err != nil {
		return nil, common.NewInternalServerError("failed to create loyalty account")
	}

	account.CurrentTier = bronzeTier

	// Award signup bonus
	go func() {
		_ = s.EarnPoints(context.Background(), &EarnPointsRequest{
			RiderID:     riderID,
			Points:      100,
			Source:      SourceSignup,
			Description: "Welcome bonus!",
		})
	}()

	return account, nil
}

// GetLoyaltyStatus gets the full loyalty status for a rider
func (s *Service) GetLoyaltyStatus(ctx context.Context, riderID uuid.UUID) (*LoyaltyStatusResponse, error) {
	account, err := s.GetOrCreateLoyaltyAccount(ctx, riderID)
	if err != nil {
		return nil, err
	}

	// Get current tier details
	var currentTier *LoyaltyTier
	if account.CurrentTierID != nil {
		currentTier, _ = s.repo.GetTier(ctx, *account.CurrentTierID)
	}

	// Get next tier
	tiers, _ := s.repo.GetAllTiers(ctx)
	var nextTier *LoyaltyTier
	pointsToNext := 0
	tierProgress := 100.0

	for _, t := range tiers {
		if t.MinPoints > account.TierPoints {
			nextTier = t
			pointsToNext = t.MinPoints - account.TierPoints
			if currentTier != nil && nextTier != nil {
				tierProgress = float64(account.TierPoints-currentTier.MinPoints) / float64(nextTier.MinPoints-currentTier.MinPoints) * 100
			}
			break
		}
	}

	// Calculate remaining benefits
	freeCancellations := 0
	freeUpgrades := 0
	if currentTier != nil {
		freeCancellations = currentTier.FreeCancellations - account.FreeCancellationsUsed
		freeUpgrades = currentTier.FreeUpgrades - account.FreeUpgradesUsed
		if freeCancellations < 0 {
			freeCancellations = 0
		}
		if freeUpgrades < 0 {
			freeUpgrades = 0
		}
	}

	benefits := []string{}
	if currentTier != nil {
		benefits = currentTier.Benefits
	}

	return &LoyaltyStatusResponse{
		RiderID:           riderID,
		CurrentTier:       currentTier,
		NextTier:          nextTier,
		AvailablePoints:   account.AvailablePoints,
		LifetimePoints:    account.LifetimePoints,
		PointsToNextTier:  pointsToNext,
		TierProgress:      tierProgress,
		StreakDays:        account.StreakDays,
		FreeCancellations: freeCancellations,
		FreeUpgrades:      freeUpgrades,
		TierExpiresAt:     account.TierPeriodEnd,
		Benefits:          benefits,
	}, nil
}

// ========================================
// POINTS MANAGEMENT
// ========================================

// EarnPoints adds points to a rider's account
func (s *Service) EarnPoints(ctx context.Context, req *EarnPointsRequest) error {
	if req.Points <= 0 {
		return common.NewBadRequestError("points must be positive", nil)
	}

	account, err := s.GetOrCreateLoyaltyAccount(ctx, req.RiderID)
	if err != nil {
		return err
	}

	// Apply tier multiplier
	multiplier := 1.0
	if account.CurrentTier != nil {
		multiplier = account.CurrentTier.Multiplier
	}
	earnedPoints := int(float64(req.Points) * multiplier)

	// Update balance
	newBalance := account.AvailablePoints + earnedPoints

	// Create transaction
	tx := &PointsTransaction{
		ID:              uuid.New(),
		RiderID:         req.RiderID,
		TransactionType: TransactionEarn,
		Points:          earnedPoints,
		BalanceAfter:    newBalance,
		Source:          req.Source,
		SourceID:        req.SourceID,
		ExpiresAt:       timePtr(time.Now().AddDate(1, 0, 0)), // Points expire in 1 year
	}

	if req.Description != "" {
		tx.Description = &req.Description
	}

	if err := s.repo.CreatePointsTransaction(ctx, tx); err != nil {
		return common.NewInternalServerError("failed to record points")
	}

	// Update account
	if err := s.repo.UpdatePoints(ctx, req.RiderID, earnedPoints, earnedPoints); err != nil {
		return common.NewInternalServerError("failed to update points")
	}

	// Check for tier upgrade
	go func() {
		_ = s.checkTierUpgrade(context.Background(), req.RiderID)
	}()

	logger.Info("Points earned",
		zap.String("rider_id", req.RiderID.String()),
		zap.Int("points", earnedPoints),
		zap.String("source", string(req.Source)),
	)

	return nil
}

// RedeemPoints redeems points for a reward
func (s *Service) RedeemPoints(ctx context.Context, req *RedeemPointsRequest) (*RedeemPointsResponse, error) {
	account, err := s.repo.GetRiderLoyalty(ctx, req.RiderID)
	if err != nil {
		return nil, common.NewNotFoundError("loyalty account not found", err)
	}

	reward, err := s.repo.GetReward(ctx, req.RewardID)
	if err != nil {
		return nil, common.NewNotFoundError("reward not found", err)
	}

	if !reward.IsActive {
		return nil, common.NewBadRequestError("reward is no longer available", nil)
	}

	if account.AvailablePoints < reward.PointsRequired {
		return nil, common.NewBadRequestError(
			fmt.Sprintf("insufficient points: need %d, have %d", reward.PointsRequired, account.AvailablePoints),
			nil,
		)
	}

	// Check tier restriction
	if reward.TierRestriction != nil && account.CurrentTierID != nil {
		currentTier, _ := s.repo.GetTier(ctx, *account.CurrentTierID)
		restrictedTier, _ := s.repo.GetTier(ctx, *reward.TierRestriction)
		if currentTier != nil && restrictedTier != nil && currentTier.MinPoints < restrictedTier.MinPoints {
			return nil, common.NewForbiddenError("this reward requires a higher tier")
		}
	}

	// Check max redemptions
	if reward.MaxRedemptionsPerUser != nil {
		count, _ := s.repo.GetUserRedemptionCount(ctx, req.RiderID, req.RewardID)
		if count >= *reward.MaxRedemptionsPerUser {
			return nil, common.NewBadRequestError("you have reached the maximum redemptions for this reward", nil)
		}
	}

	// Generate redemption code
	code := generateRedemptionCode()
	newBalance := account.AvailablePoints - reward.PointsRequired

	// Create redemption
	redemption := &Redemption{
		ID:             uuid.New(),
		RiderID:        req.RiderID,
		RewardID:       req.RewardID,
		PointsSpent:    reward.PointsRequired,
		RedemptionCode: code,
		Status:         "active",
		ExpiresAt:      time.Now().AddDate(0, 0, reward.ValidDays),
	}

	if err := s.repo.CreateRedemption(ctx, redemption); err != nil {
		return nil, common.NewInternalServerError("failed to create redemption")
	}

	// Create debit transaction
	tx := &PointsTransaction{
		ID:              uuid.New(),
		RiderID:         req.RiderID,
		TransactionType: TransactionRedeem,
		Points:          -reward.PointsRequired,
		BalanceAfter:    newBalance,
		Source:          PointSource("redemption"),
		SourceID:        &redemption.ID,
	}

	if err := s.repo.CreatePointsTransaction(ctx, tx); err != nil {
		return nil, common.NewInternalServerError("failed to record redemption")
	}

	// Update balance
	if err := s.repo.DeductPoints(ctx, req.RiderID, reward.PointsRequired); err != nil {
		return nil, common.NewInternalServerError("failed to deduct points")
	}

	// Increment redemption count
	_ = s.repo.IncrementRewardRedemptionCount(ctx, req.RewardID)

	logger.Info("Points redeemed",
		zap.String("rider_id", req.RiderID.String()),
		zap.String("reward_id", req.RewardID.String()),
		zap.Int("points", reward.PointsRequired),
	)

	return &RedeemPointsResponse{
		RedemptionID:   redemption.ID,
		RedemptionCode: code,
		PointsSpent:    reward.PointsRequired,
		BalanceAfter:   newBalance,
		ExpiresAt:      redemption.ExpiresAt,
		Instructions:   fmt.Sprintf("Use code %s at checkout. Valid until %s", code, redemption.ExpiresAt.Format("Jan 2, 2006")),
	}, nil
}

// GetPointsHistory gets points transaction history
func (s *Service) GetPointsHistory(ctx context.Context, riderID uuid.UUID, page, pageSize int) (*PointsHistoryResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	transactions, total, err := s.repo.GetPointsHistory(ctx, riderID, pageSize, offset)
	if err != nil {
		return nil, common.NewInternalServerError("failed to get points history")
	}

	// Convert pointers to values
	txList := make([]PointsTransaction, len(transactions))
	for i, tx := range transactions {
		txList[i] = *tx
	}

	return &PointsHistoryResponse{
		Transactions: txList,
		Total:        total,
		Page:         page,
		PageSize:     pageSize,
	}, nil
}

// ========================================
// CHALLENGES
// ========================================

// GetActiveChallenges gets active challenges for a rider
func (s *Service) GetActiveChallenges(ctx context.Context, riderID uuid.UUID) (*ActiveChallengesResponse, error) {
	account, err := s.GetOrCreateLoyaltyAccount(ctx, riderID)
	if err != nil {
		return nil, err
	}

	challenges, err := s.repo.GetActiveChallenges(ctx, account.CurrentTierID)
	if err != nil {
		return nil, common.NewInternalServerError("failed to get challenges")
	}

	var result []ChallengeWithProgress
	for _, c := range challenges {
		progress, _ := s.repo.GetChallengeProgress(ctx, riderID, c.ID)

		cwp := ChallengeWithProgress{
			Challenge:     *c,
			CurrentValue:  0,
			Completed:     false,
			DaysRemaining: int(time.Until(c.EndDate).Hours() / 24),
		}

		if progress != nil {
			cwp.CurrentValue = progress.CurrentValue
			cwp.Completed = progress.Completed
			cwp.ProgressPercent = float64(progress.CurrentValue) / float64(c.TargetValue) * 100
			if cwp.ProgressPercent > 100 {
				cwp.ProgressPercent = 100
			}
		}

		result = append(result, cwp)
	}

	return &ActiveChallengesResponse{Challenges: result}, nil
}

// UpdateChallengeProgress updates a rider's progress on challenges
func (s *Service) UpdateChallengeProgress(ctx context.Context, riderID uuid.UUID, challengeType string, increment int) error {
	account, err := s.repo.GetRiderLoyalty(ctx, riderID)
	if err != nil {
		return nil // No account, skip
	}

	challenges, err := s.repo.GetActiveChallengesByType(ctx, challengeType, account.CurrentTierID)
	if err != nil {
		return err
	}

	for _, challenge := range challenges {
		progress, _ := s.repo.GetChallengeProgress(ctx, riderID, challenge.ID)

		if progress == nil {
			// Create new progress record
			progress = &ChallengeProgress{
				ID:          uuid.New(),
				RiderID:     riderID,
				ChallengeID: challenge.ID,
			}
			if err := s.repo.CreateChallengeProgress(ctx, progress); err != nil {
				continue
			}
		}

		if progress.Completed {
			continue // Already completed
		}

		// Update progress
		newValue := progress.CurrentValue + increment
		completed := newValue >= challenge.TargetValue

		if err := s.repo.UpdateChallengeProgress(ctx, progress.ID, newValue, completed); err != nil {
			continue
		}

		// Award points if completed
		if completed && !progress.Completed {
			_ = s.EarnPoints(ctx, &EarnPointsRequest{
				RiderID:     riderID,
				Points:      challenge.RewardPoints,
				Source:      SourceChallenge,
				SourceID:    &challenge.ID,
				Description: fmt.Sprintf("Completed challenge: %s", challenge.Name),
			})
		}
	}

	return nil
}

// ========================================
// TIER MANAGEMENT
// ========================================

// checkTierUpgrade checks if a rider qualifies for a tier upgrade
func (s *Service) checkTierUpgrade(ctx context.Context, riderID uuid.UUID) error {
	account, err := s.repo.GetRiderLoyalty(ctx, riderID)
	if err != nil {
		return err
	}

	tiers, err := s.repo.GetAllTiers(ctx)
	if err != nil {
		return err
	}

	// Find the highest tier the rider qualifies for
	var newTier *LoyaltyTier
	for _, t := range tiers {
		if account.TierPoints >= t.MinPoints {
			newTier = t
		}
	}

	if newTier == nil || (account.CurrentTierID != nil && *account.CurrentTierID == newTier.ID) {
		return nil // No change
	}

	// Upgrade tier
	if err := s.repo.UpdateTier(ctx, riderID, newTier.ID); err != nil {
		return err
	}

	logger.Info("Tier upgraded",
		zap.String("rider_id", riderID.String()),
		zap.String("new_tier", string(newTier.Name)),
	)

	return nil
}

// ========================================
// REWARDS CATALOG
// ========================================

// GetRewardsCatalog gets available rewards
func (s *Service) GetRewardsCatalog(ctx context.Context, riderID uuid.UUID) ([]*RewardCatalogItem, error) {
	account, _ := s.repo.GetRiderLoyalty(ctx, riderID)

	var tierID *uuid.UUID
	if account != nil {
		tierID = account.CurrentTierID
	}

	return s.repo.GetAvailableRewards(ctx, tierID)
}

// GetAllTiers returns all loyalty tiers
func (s *Service) GetAllTiers(ctx context.Context) ([]*LoyaltyTier, error) {
	return s.repo.GetAllTiers(ctx)
}

// ========================================
// HELPER FUNCTIONS
// ========================================

func generateRedemptionCode() string {
	bytes := make([]byte, 6)
	rand.Read(bytes)
	return "RDM-" + hex.EncodeToString(bytes)[:8]
}

func timePtr(t time.Time) *time.Time {
	return &t
}
