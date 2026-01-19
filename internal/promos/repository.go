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

// Ensure Repository implements PromosRepository.
var _ PromosRepository = (*Repository)(nil)

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

	rideTypes := make([]*RideType, 0)
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

// GetAllPromoCodes retrieves all promo codes with pagination
func (r *Repository) GetAllPromoCodes(ctx context.Context, limit, offset int) ([]*PromoCode, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM promo_codes`
	err := r.db.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count promo codes: %w", err)
	}

	// Get promo codes
	query := `
		SELECT id, code, description, discount_type, discount_value, max_discount_amount,
			min_ride_amount, max_uses, total_uses, uses_per_user, valid_from, valid_until,
			is_active, created_by, created_at, updated_at
		FROM promo_codes
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get promo codes: %w", err)
	}
	defer rows.Close()

	promoCodes := []*PromoCode{}
	for rows.Next() {
		promo := &PromoCode{}
		err := rows.Scan(
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
			return nil, 0, fmt.Errorf("failed to scan promo code: %w", err)
		}
		promoCodes = append(promoCodes, promo)
	}

	return promoCodes, total, nil
}

// GetAllReferralCodes retrieves all referral codes with pagination
func (r *Repository) GetAllReferralCodes(ctx context.Context, limit, offset int) ([]*ReferralCode, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM referral_codes`
	err := r.db.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count referral codes: %w", err)
	}

	// Get referral codes
	query := `
		SELECT id, user_id, code, total_referrals, total_earnings, created_at
		FROM referral_codes
		ORDER BY total_referrals DESC, created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get referral codes: %w", err)
	}
	defer rows.Close()

	codes := []*ReferralCode{}
	for rows.Next() {
		code := &ReferralCode{}
		err := rows.Scan(
			&code.ID,
			&code.UserID,
			&code.Code,
			&code.TotalReferrals,
			&code.TotalEarnings,
			&code.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan referral code: %w", err)
		}
		codes = append(codes, code)
	}

	return codes, total, nil
}

// UpdatePromoCode updates an existing promo code
func (r *Repository) UpdatePromoCode(ctx context.Context, promo *PromoCode) error {
	query := `
		UPDATE promo_codes
		SET description = $2,
		    discount_type = $3,
		    discount_value = $4,
		    max_discount_amount = $5,
		    min_ride_amount = $6,
		    max_uses = $7,
		    uses_per_user = $8,
		    valid_from = $9,
		    valid_until = $10,
		    is_active = $11,
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		promo.ID,
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
	)

	return err
}

// DeactivatePromoCode deactivates a promo code
func (r *Repository) DeactivatePromoCode(ctx context.Context, promoID uuid.UUID) error {
	query := `
		UPDATE promo_codes
		SET is_active = false,
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, promoID)
	return err
}

// GetPromoCodeByID retrieves a promo code by ID
func (r *Repository) GetPromoCodeByID(ctx context.Context, promoID uuid.UUID) (*PromoCode, error) {
	query := `
		SELECT id, code, description, discount_type, discount_value,
		       max_discount_amount, min_ride_amount, max_uses, total_uses,
		       uses_per_user, valid_from, valid_until, is_active,
		       created_by, created_at, updated_at
		FROM promo_codes
		WHERE id = $1
	`

	promo := &PromoCode{}
	err := r.db.QueryRow(ctx, query, promoID).Scan(
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

// GetPromoCodeUsageStats retrieves usage statistics for a promo code
func (r *Repository) GetPromoCodeUsageStats(ctx context.Context, promoID uuid.UUID) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(DISTINCT user_id) as unique_users,
			COUNT(*) as total_uses,
			COALESCE(SUM(discount_amount), 0) as total_discount,
			COALESCE(AVG(discount_amount), 0) as avg_discount,
			COALESCE(SUM(original_amount), 0) as total_original_amount,
			COALESCE(SUM(final_amount), 0) as total_final_amount
		FROM promo_code_uses
		WHERE promo_code_id = $1
	`

	var stats struct {
		UniqueUsers         int
		TotalUses           int
		TotalDiscount       float64
		AvgDiscount         float64
		TotalOriginalAmount float64
		TotalFinalAmount    float64
	}

	err := r.db.QueryRow(ctx, query, promoID).Scan(
		&stats.UniqueUsers,
		&stats.TotalUses,
		&stats.TotalDiscount,
		&stats.AvgDiscount,
		&stats.TotalOriginalAmount,
		&stats.TotalFinalAmount,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get promo code usage stats: %w", err)
	}

	result := map[string]interface{}{
		"unique_users":          stats.UniqueUsers,
		"total_uses":            stats.TotalUses,
		"total_discount":        stats.TotalDiscount,
		"avg_discount":          stats.AvgDiscount,
		"total_original_amount": stats.TotalOriginalAmount,
		"total_final_amount":    stats.TotalFinalAmount,
	}

	return result, nil
}

// GetReferralByID retrieves a referral by ID with detailed information
func (r *Repository) GetReferralByID(ctx context.Context, referralID uuid.UUID) (*Referral, error) {
	query := `
		SELECT id, referrer_id, referred_id, referral_code_id,
		       referrer_bonus, referred_bonus, referrer_bonus_applied,
		       referred_bonus_applied, referred_first_ride_id,
		       created_at, completed_at
		FROM referrals
		WHERE id = $1
	`

	referral := &Referral{}
	err := r.db.QueryRow(ctx, query, referralID).Scan(
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

// GetReferralEarnings retrieves referral earnings for a user
func (r *Repository) GetReferralEarnings(ctx context.Context, userID uuid.UUID) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total_referrals,
			COUNT(CASE WHEN referrer_bonus_applied THEN 1 END) as completed_referrals,
			COALESCE(SUM(CASE WHEN referrer_bonus_applied THEN referrer_bonus ELSE 0 END), 0) as total_earnings,
			COUNT(CASE WHEN NOT referrer_bonus_applied THEN 1 END) as pending_referrals,
			COALESCE(SUM(CASE WHEN NOT referrer_bonus_applied THEN referrer_bonus ELSE 0 END), 0) as pending_earnings
		FROM referrals
		WHERE referrer_id = $1
	`

	var earnings struct {
		TotalReferrals     int
		CompletedReferrals int
		TotalEarnings      float64
		PendingReferrals   int
		PendingEarnings    float64
	}

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&earnings.TotalReferrals,
		&earnings.CompletedReferrals,
		&earnings.TotalEarnings,
		&earnings.PendingReferrals,
		&earnings.PendingEarnings,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get referral earnings: %w", err)
	}

	result := map[string]interface{}{
		"total_referrals":     earnings.TotalReferrals,
		"completed_referrals": earnings.CompletedReferrals,
		"total_earnings":      earnings.TotalEarnings,
		"pending_referrals":   earnings.PendingReferrals,
		"pending_earnings":    earnings.PendingEarnings,
	}

	return result, nil
}
