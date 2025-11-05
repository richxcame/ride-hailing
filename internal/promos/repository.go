package promos

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for promo codes and referrals
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new promos repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreatePromoCode creates a new promo code
func (r *Repository) CreatePromoCode(ctx context.Context, promo *PromoCode) error {
	query := `
		INSERT INTO promo_codes (id, code, description, discount_type, discount_value,
			max_discount_amount, min_ride_amount, max_uses, uses_per_user, valid_from,
			valid_until, is_active, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	promo.ID = uuid.New()
	now := time.Now()
	promo.CreatedAt = now
	promo.UpdatedAt = now

	_, err := r.db.Exec(ctx, query,
		promo.ID,
		promo.Code,
		promo.Description,
		promo.DiscountType,
		promo.DiscountValue,
		promo.MaxDiscountAmount,
		promo.MinRideAmount,
		promo.MaxUses,
		promo.UsesPerUser,
		promo.ValidFrom,
		promo.ValidUntil,
		promo.IsActive,
		promo.CreatedBy,
		promo.CreatedAt,
		promo.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create promo code: %w", err)
	}

	return nil
}

// GetPromoCodeByCode retrieves a promo code by its code
func (r *Repository) GetPromoCodeByCode(ctx context.Context, code string) (*PromoCode, error) {
	query := `
		SELECT id, code, description, discount_type, discount_value, max_discount_amount,
			min_ride_amount, max_uses, total_uses, uses_per_user, valid_from, valid_until,
			is_active, created_by, created_at, updated_at
		FROM promo_codes
		WHERE code = $1
	`

	promo := &PromoCode{}
	err := r.db.QueryRow(ctx, query, code).Scan(
		&promo.ID,
		&promo.Code,
		&promo.Description,
		&promo.DiscountType,
		&promo.DiscountValue,
		&promo.MaxDiscountAmount,
		&promo.MinRideAmount,
		&promo.MaxUses,
		&promo.TotalUses,
		&promo.UsesPerUser,
		&promo.ValidFrom,
		&promo.ValidUntil,
		&promo.IsActive,
		&promo.CreatedBy,
		&promo.CreatedAt,
		&promo.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get promo code: %w", err)
	}

	return promo, nil
}

// GetPromoCodeUsesByUser retrieves promo code uses for a specific user and promo code
func (r *Repository) GetPromoCodeUsesByUser(ctx context.Context, promoCodeID, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM promo_code_uses WHERE promo_code_id = $1 AND user_id = $2`

	var count int
	err := r.db.QueryRow(ctx, query, promoCodeID, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get promo code uses: %w", err)
	}

	return count, nil
}

// CreatePromoCodeUse records a promo code use
func (r *Repository) CreatePromoCodeUse(ctx context.Context, use *PromoCodeUse) error {
	query := `
		INSERT INTO promo_code_uses (id, promo_code_id, user_id, ride_id, discount_amount,
			original_amount, final_amount, used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	use.ID = uuid.New()
	use.UsedAt = time.Now()

	_, err := r.db.Exec(ctx, query,
		use.ID,
		use.PromoCodeID,
		use.UserID,
		use.RideID,
		use.DiscountAmount,
		use.OriginalAmount,
		use.FinalAmount,
		use.UsedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create promo code use: %w", err)
	}

	// Increment total uses
	_, err = r.db.Exec(ctx, `UPDATE promo_codes SET total_uses = total_uses + 1 WHERE id = $1`, use.PromoCodeID)
	if err != nil {
		return fmt.Errorf("failed to update promo code uses: %w", err)
	}

	return nil
}

// GetAllRideTypes retrieves all active ride types
func (r *Repository) GetAllRideTypes(ctx context.Context) ([]*RideType, error) {
	query := `
		SELECT id, name, description, base_fare, per_km_rate, per_minute_rate,
			minimum_fare, capacity, is_active, created_at, updated_at
		FROM ride_types
		WHERE is_active = true
		ORDER BY base_fare ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride types: %w", err)
	}
	defer rows.Close()

	var rideTypes []*RideType
	for rows.Next() {
		rt := &RideType{}
		err := rows.Scan(
			&rt.ID,
			&rt.Name,
			&rt.Description,
			&rt.BaseFare,
			&rt.PerKmRate,
			&rt.PerMinuteRate,
			&rt.MinimumFare,
			&rt.Capacity,
			&rt.IsActive,
			&rt.CreatedAt,
			&rt.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ride type: %w", err)
		}
		rideTypes = append(rideTypes, rt)
	}

	return rideTypes, nil
}

// GetRideTypeByID retrieves a ride type by ID
func (r *Repository) GetRideTypeByID(ctx context.Context, id uuid.UUID) (*RideType, error) {
	query := `
		SELECT id, name, description, base_fare, per_km_rate, per_minute_rate,
			minimum_fare, capacity, is_active, created_at, updated_at
		FROM ride_types
		WHERE id = $1
	`

	rt := &RideType{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&rt.ID,
		&rt.Name,
		&rt.Description,
		&rt.BaseFare,
		&rt.PerKmRate,
		&rt.PerMinuteRate,
		&rt.MinimumFare,
		&rt.Capacity,
		&rt.IsActive,
		&rt.CreatedAt,
		&rt.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get ride type: %w", err)
	}

	return rt, nil
}

// CreateReferralCode creates a referral code for a user
func (r *Repository) CreateReferralCode(ctx context.Context, referral *ReferralCode) error {
	query := `
		INSERT INTO referral_codes (id, user_id, code, total_referrals, total_earnings, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	referral.ID = uuid.New()
	referral.CreatedAt = time.Now()

	_, err := r.db.Exec(ctx, query,
		referral.ID,
		referral.UserID,
		referral.Code,
		referral.TotalReferrals,
		referral.TotalEarnings,
		referral.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create referral code: %w", err)
	}

	return nil
}

// GetReferralCodeByCode retrieves a referral code by code
func (r *Repository) GetReferralCodeByCode(ctx context.Context, code string) (*ReferralCode, error) {
	query := `
		SELECT id, user_id, code, total_referrals, total_earnings, created_at
		FROM referral_codes
		WHERE code = $1
	`

	ref := &ReferralCode{}
	err := r.db.QueryRow(ctx, query, code).Scan(
		&ref.ID,
		&ref.UserID,
		&ref.Code,
		&ref.TotalReferrals,
		&ref.TotalEarnings,
		&ref.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get referral code: %w", err)
	}

	return ref, nil
}

// GetReferralCodeByUserID retrieves a user's referral code
func (r *Repository) GetReferralCodeByUserID(ctx context.Context, userID uuid.UUID) (*ReferralCode, error) {
	query := `
		SELECT id, user_id, code, total_referrals, total_earnings, created_at
		FROM referral_codes
		WHERE user_id = $1
	`

	ref := &ReferralCode{}
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&ref.ID,
		&ref.UserID,
		&ref.Code,
		&ref.TotalReferrals,
		&ref.TotalEarnings,
		&ref.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get referral code: %w", err)
	}

	return ref, nil
}

// CreateReferral creates a referral relationship
func (r *Repository) CreateReferral(ctx context.Context, referral *Referral) error {
	query := `
		INSERT INTO referrals (id, referrer_id, referred_id, referral_code_id, referrer_bonus,
			referred_bonus, referrer_bonus_applied, referred_bonus_applied, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	referral.ID = uuid.New()
	referral.CreatedAt = time.Now()

	_, err := r.db.Exec(ctx, query,
		referral.ID,
		referral.ReferrerID,
		referral.ReferredID,
		referral.ReferralCodeID,
		referral.ReferrerBonus,
		referral.ReferredBonus,
		referral.ReferrerBonusApplied,
		referral.ReferredBonusApplied,
		referral.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create referral: %w", err)
	}

	return nil
}

// GetReferralByReferredID retrieves a referral by the referred user ID
func (r *Repository) GetReferralByReferredID(ctx context.Context, referredID uuid.UUID) (*Referral, error) {
	query := `
		SELECT id, referrer_id, referred_id, referral_code_id, referrer_bonus, referred_bonus,
			   referrer_bonus_applied, referred_bonus_applied, referred_first_ride_id,
			   created_at, completed_at
		FROM referrals
		WHERE referred_id = $1
		LIMIT 1
	`

	referral := &Referral{}
	err := r.db.QueryRow(ctx, query, referredID).Scan(
		&referral.ID,
		&referral.ReferrerID,
		&referral.ReferredID,
		&referral.ReferralCodeID,
		&referral.ReferrerBonus,
		&referral.ReferredBonus,
		&referral.ReferrerBonusApplied,
		&referral.ReferredBonusApplied,
		&referral.ReferredFirstRideID,
		&referral.CreatedAt,
		&referral.CompletedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get referral: %w", err)
	}

	return referral, nil
}

// IsFirstCompletedRide checks if the given ride is the user's first completed ride
func (r *Repository) IsFirstCompletedRide(ctx context.Context, userID uuid.UUID, rideID uuid.UUID) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM rides
		WHERE rider_id = $1 AND status = 'completed' AND id != $2
	`

	var count int
	err := r.db.QueryRow(ctx, query, userID, rideID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to count completed rides: %w", err)
	}

	return count == 0, nil
}

// MarkReferralBonusesApplied marks the referral bonuses as applied
func (r *Repository) MarkReferralBonusesApplied(ctx context.Context, referralID uuid.UUID, rideID uuid.UUID) error {
	query := `
		UPDATE referrals
		SET referrer_bonus_applied = true,
			referred_bonus_applied = true,
			completed_at = NOW(),
			referred_first_ride_id = $2
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, referralID, rideID)
	if err != nil {
		return fmt.Errorf("failed to mark bonuses as applied: %w", err)
	}

	return nil
}
