package promos

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/cache"
)

const (
	// Referral bonus amounts
	ReferrerBonusAmount = 10.00 // Bonus for the person who refers
	ReferredBonusAmount = 10.00 // Bonus for the new user
)

// PromosRepository defines the storage operations required by the service.
type PromosRepository interface {
	GetPromoCodeByCode(ctx context.Context, code string) (*PromoCode, error)
	GetPromoCodeByID(ctx context.Context, promoID uuid.UUID) (*PromoCode, error)
	GetPromoCodeUsesByUser(ctx context.Context, promoID uuid.UUID, userID uuid.UUID) (int, error)
	CreatePromoCodeUse(ctx context.Context, use *PromoCodeUse) error
	CreatePromoCode(ctx context.Context, promo *PromoCode) error
	UpdatePromoCode(ctx context.Context, promo *PromoCode) error
	DeactivatePromoCode(ctx context.Context, promoID uuid.UUID) error
	GetAllPromoCodes(ctx context.Context, limit, offset int) ([]*PromoCode, int, error)
	GetPromoCodeUsageStats(ctx context.Context, promoID uuid.UUID) (map[string]interface{}, error)
	GetReferralCodeByUserID(ctx context.Context, userID uuid.UUID) (*ReferralCode, error)
	CreateReferralCode(ctx context.Context, code *ReferralCode) error
	GetReferralCodeByCode(ctx context.Context, code string) (*ReferralCode, error)
	CreateReferral(ctx context.Context, referral *Referral) error
	GetAllReferralCodes(ctx context.Context, limit, offset int) ([]*ReferralCode, int, error)
	GetReferralByID(ctx context.Context, referralID uuid.UUID) (*Referral, error)
	GetReferralEarnings(ctx context.Context, userID uuid.UUID) (map[string]interface{}, error)
	GetAllRideTypes(ctx context.Context) ([]*RideType, error)
	GetRideTypeByID(ctx context.Context, id uuid.UUID) (*RideType, error)
	GetReferralByReferredID(ctx context.Context, userID uuid.UUID) (*Referral, error)
	IsFirstCompletedRide(ctx context.Context, userID uuid.UUID, rideID uuid.UUID) (bool, error)
	MarkReferralBonusesApplied(ctx context.Context, referralID uuid.UUID, rideID uuid.UUID) error
}

// Service handles promo code and referral business logic
type Service struct {
	repo  PromosRepository
	cache *cache.Manager
}

// NewService creates a new promos service
func NewService(repo PromosRepository) *Service {
	return &Service{repo: repo}
}

// ValidatePromoCode validates a promo code and calculates the discount
func (s *Service) ValidatePromoCode(ctx context.Context, code string, userID uuid.UUID, rideAmount float64) (*PromoCodeValidation, error) {
	// Get promo code
	promo, err := s.repo.GetPromoCodeByCode(ctx, code)
	if err != nil {
		return &PromoCodeValidation{
			Valid:   false,
			Message: "Invalid promo code",
		}, nil
	}

	// Check if active
	if !promo.IsActive {
		return &PromoCodeValidation{
			Valid:   false,
			Message: "This promo code is no longer active",
		}, nil
	}

	// Check validity dates
	now := time.Now()
	if now.Before(promo.ValidFrom) {
		return &PromoCodeValidation{
			Valid:   false,
			Message: "This promo code is not yet valid",
		}, nil
	}
	if now.After(promo.ValidUntil) {
		return &PromoCodeValidation{
			Valid:   false,
			Message: "This promo code has expired",
		}, nil
	}

	// Check max uses
	if promo.MaxUses != nil && promo.TotalUses >= *promo.MaxUses {
		return &PromoCodeValidation{
			Valid:   false,
			Message: "This promo code has reached its maximum usage limit",
		}, nil
	}

	// Check uses per user
	userUses, err := s.repo.GetPromoCodeUsesByUser(ctx, promo.ID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check user uses: %w", err)
	}

	if userUses >= promo.UsesPerUser {
		return &PromoCodeValidation{
			Valid:   false,
			Message: "You have already used this promo code the maximum number of times",
		}, nil
	}

	// Check minimum ride amount
	if promo.MinRideAmount != nil && rideAmount < *promo.MinRideAmount {
		return &PromoCodeValidation{
			Valid:   false,
			Message: fmt.Sprintf("Minimum ride amount of $%.2f required to use this promo code", *promo.MinRideAmount),
		}, nil
	}

	// Calculate discount
	var discountAmount float64
	if promo.DiscountType == "percentage" {
		discountAmount = rideAmount * (promo.DiscountValue / 100.0)
		// Apply max discount if set
		if promo.MaxDiscountAmount != nil && discountAmount > *promo.MaxDiscountAmount {
			discountAmount = *promo.MaxDiscountAmount
		}
	} else {
		// Fixed amount
		discountAmount = promo.DiscountValue
	}

	// Ensure discount doesn't exceed ride amount
	if discountAmount > rideAmount {
		discountAmount = rideAmount
	}

	finalAmount := rideAmount - discountAmount

	return &PromoCodeValidation{
		Valid:          true,
		DiscountAmount: discountAmount,
		FinalAmount:    finalAmount,
	}, nil
}

// ApplyPromoCode applies a promo code to a ride
func (s *Service) ApplyPromoCode(ctx context.Context, code string, userID, rideID uuid.UUID, originalAmount float64) (*PromoCodeUse, error) {
	// Validate first
	validation, err := s.ValidatePromoCode(ctx, code, userID, originalAmount)
	if err != nil {
		return nil, err
	}

	if !validation.Valid {
		return nil, fmt.Errorf("%s", validation.Message)
	}

	// Get promo code again
	promo, err := s.repo.GetPromoCodeByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	// Create promo code use record
	use := &PromoCodeUse{
		PromoCodeID:    promo.ID,
		UserID:         userID,
		RideID:         rideID,
		DiscountAmount: validation.DiscountAmount,
		OriginalAmount: originalAmount,
		FinalAmount:    validation.FinalAmount,
	}

	err = s.repo.CreatePromoCodeUse(ctx, use)
	if err != nil {
		return nil, fmt.Errorf("failed to apply promo code: %w", err)
	}

	return use, nil
}

// CreatePromoCode creates a new promo code (admin only)
func (s *Service) CreatePromoCode(ctx context.Context, promo *PromoCode) error {
	// Validate promo code
	if promo.Code == "" {
		return fmt.Errorf("promo code cannot be empty")
	}

	// Normalize code (uppercase, no spaces)
	promo.Code = strings.ToUpper(strings.TrimSpace(promo.Code))

	if promo.DiscountType != "percentage" && promo.DiscountType != "fixed_amount" {
		return fmt.Errorf("discount type must be 'percentage' or 'fixed_amount'")
	}

	if promo.DiscountValue <= 0 {
		return fmt.Errorf("discount value must be greater than 0")
	}

	if promo.DiscountType == "percentage" && promo.DiscountValue > 100 {
		return fmt.Errorf("percentage discount cannot exceed 100%%")
	}

	if promo.ValidFrom.After(promo.ValidUntil) {
		return fmt.Errorf("valid_from must be before valid_until")
	}

	return s.repo.CreatePromoCode(ctx, promo)
}

// GenerateReferralCode generates a unique referral code for a user
func (s *Service) GenerateReferralCode(ctx context.Context, userID uuid.UUID, baseCode string) (*ReferralCode, error) {
	// Try to get existing referral code
	existing, err := s.repo.GetReferralCodeByUserID(ctx, userID)
	if err == nil {
		return existing, nil
	}

	// Generate unique code
	code := s.generateUniqueCode(baseCode)

	referral := &ReferralCode{
		UserID:         userID,
		Code:           code,
		TotalReferrals: 0,
		TotalEarnings:  0,
	}

	err = s.repo.CreateReferralCode(ctx, referral)
	if err != nil {
		return nil, fmt.Errorf("failed to create referral code: %w", err)
	}

	return referral, nil
}

// ApplyReferralCode applies a referral code when a new user signs up
func (s *Service) ApplyReferralCode(ctx context.Context, referralCode string, newUserID uuid.UUID) error {
	// Get referral code
	refCode, err := s.repo.GetReferralCodeByCode(ctx, referralCode)
	if err != nil {
		return fmt.Errorf("invalid referral code")
	}

	// Can't refer yourself
	if refCode.UserID == newUserID {
		return fmt.Errorf("you cannot use your own referral code")
	}

	// Create referral relationship
	referral := &Referral{
		ReferrerID:           refCode.UserID,
		ReferredID:           newUserID,
		ReferralCodeID:       refCode.ID,
		ReferrerBonus:        ReferrerBonusAmount,
		ReferredBonus:        ReferredBonusAmount,
		ReferrerBonusApplied: false,
		ReferredBonusApplied: false,
	}

	err = s.repo.CreateReferral(ctx, referral)
	if err != nil {
		return fmt.Errorf("failed to apply referral code: %w", err)
	}

	return nil
}

// SetCache sets an optional cache manager for read-heavy operations.
func (s *Service) SetCache(cm *cache.Manager) {
	s.cache = cm
}

// GetAllRideTypes retrieves all available ride types (cached for 15 minutes).
func (s *Service) GetAllRideTypes(ctx context.Context) ([]*RideType, error) {
	if s.cache != nil {
		var cached []*RideType
		err := s.cache.GetOrSet(ctx, "ride_types:all", cache.TTL.Medium(), &cached, func() (interface{}, error) {
			return s.repo.GetAllRideTypes(ctx)
		})
		if err == nil {
			if cached == nil {
				cached = []*RideType{}
			}
			return cached, nil
		}
		// Fall through to DB on cache error
	}

	rideTypes, err := s.repo.GetAllRideTypes(ctx)
	if err != nil {
		return nil, err
	}
	if rideTypes == nil {
		rideTypes = []*RideType{}
	}
	return rideTypes, nil
}

// GetRideTypeByID retrieves a specific ride type (cached for 15 minutes).
func (s *Service) GetRideTypeByID(ctx context.Context, id uuid.UUID) (*RideType, error) {
	if s.cache != nil {
		var cached RideType
		err := s.cache.GetOrSet(ctx, cache.Keys.RideType(id.String()), cache.TTL.Medium(), &cached, func() (interface{}, error) {
			return s.repo.GetRideTypeByID(ctx, id)
		})
		if err == nil {
			return &cached, nil
		}
	}
	return s.repo.GetRideTypeByID(ctx, id)
}

// CalculateFareForRideType calculates fare based on ride type
func (s *Service) CalculateFareForRideType(ctx context.Context, rideTypeID uuid.UUID, distance float64, duration int, surgeMultiplier float64) (float64, error) {
	rideType, err := s.GetRideTypeByID(ctx, rideTypeID)
	if err != nil {
		return 0, err
	}

	// Calculate base fare
	distanceCost := distance * rideType.PerKmRate
	timeCost := float64(duration) * rideType.PerMinuteRate
	baseFare := rideType.BaseFare + distanceCost + timeCost

	// Apply surge multiplier
	fare := baseFare * surgeMultiplier

	// Ensure minimum fare
	if fare < rideType.MinimumFare {
		fare = rideType.MinimumFare
	}

	return fare, nil
}

// generateUniqueCode generates a unique referral code
func (s *Service) generateUniqueCode(base string) string {
	// Clean base string
	base = strings.ToUpper(strings.ReplaceAll(base, " ", ""))
	if len(base) > 6 {
		base = base[:6]
	}

	// Add random suffix
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123916789"
	suffix := make([]byte, 4)
	for i := range suffix {
		suffix[i] = charset[rand.Intn(len(charset))]
	}

	return base + string(suffix)
}

// ProcessReferralBonus processes referral bonuses when a user completes their first ride
// Returns the referrer and referred bonuses to be credited
func (s *Service) ProcessReferralBonus(ctx context.Context, userID uuid.UUID, rideID uuid.UUID) (map[string]interface{}, error) {
	// Check if this user was referred
	referral, err := s.repo.GetReferralByReferredID(ctx, userID)
	if err != nil {
		// User wasn't referred, no bonus to process
		return nil, nil
	}

	// Check if bonuses have already been applied
	if referral.ReferrerBonusApplied && referral.ReferredBonusApplied {
		return nil, nil
	}

	// Check if this is the user's first completed ride
	isFirstRide, err := s.repo.IsFirstCompletedRide(ctx, userID, rideID)
	if err != nil {
		return nil, fmt.Errorf("failed to check if first ride: %w", err)
	}

	if !isFirstRide {
		return nil, nil
	}

	// Mark bonuses as ready to be applied
	err = s.repo.MarkReferralBonusesApplied(ctx, referral.ID, rideID)
	if err != nil {
		return nil, fmt.Errorf("failed to mark bonuses as applied: %w", err)
	}

	// Return bonus information for both users
	result := map[string]interface{}{
		"referrer_id":    referral.ReferrerID,
		"referrer_bonus": referral.ReferrerBonus,
		"referred_id":    referral.ReferredID,
		"referred_bonus": referral.ReferredBonus,
		"has_bonus":      true,
	}

	return result, nil
}

// GetAllPromoCodes retrieves all promo codes with pagination
func (s *Service) GetAllPromoCodes(ctx context.Context, limit, offset int) ([]*PromoCode, int, error) {
	return s.repo.GetAllPromoCodes(ctx, limit, offset)
}

// GetAllReferralCodes retrieves all referral codes with pagination
func (s *Service) GetAllReferralCodes(ctx context.Context, limit, offset int) ([]*ReferralCode, int, error) {
	return s.repo.GetAllReferralCodes(ctx, limit, offset)
}

// UpdatePromoCode updates an existing promo code
func (s *Service) UpdatePromoCode(ctx context.Context, promo *PromoCode) error {
	// Validate promo code
	if promo.DiscountType != "percentage" && promo.DiscountType != "fixed_amount" {
		return fmt.Errorf("discount type must be 'percentage' or 'fixed_amount'")
	}

	if promo.DiscountValue <= 0 {
		return fmt.Errorf("discount value must be greater than 0")
	}

	if promo.DiscountType == "percentage" && promo.DiscountValue > 100 {
		return fmt.Errorf("percentage discount cannot exceed 100%%")
	}

	if promo.ValidFrom.After(promo.ValidUntil) {
		return fmt.Errorf("valid_from must be before valid_until")
	}

	return s.repo.UpdatePromoCode(ctx, promo)
}

// DeactivatePromoCode deactivates a promo code
func (s *Service) DeactivatePromoCode(ctx context.Context, promoID uuid.UUID) error {
	return s.repo.DeactivatePromoCode(ctx, promoID)
}

// GetPromoCodeByID retrieves a promo code by ID
func (s *Service) GetPromoCodeByID(ctx context.Context, promoID uuid.UUID) (*PromoCode, error) {
	return s.repo.GetPromoCodeByID(ctx, promoID)
}

// GetPromoCodeUsageStats retrieves usage statistics for a promo code
func (s *Service) GetPromoCodeUsageStats(ctx context.Context, promoID uuid.UUID) (map[string]interface{}, error) {
	return s.repo.GetPromoCodeUsageStats(ctx, promoID)
}

// GetReferralByID retrieves a referral by ID
func (s *Service) GetReferralByID(ctx context.Context, referralID uuid.UUID) (*Referral, error) {
	return s.repo.GetReferralByID(ctx, referralID)
}

// GetReferralEarnings retrieves referral earnings for a user
func (s *Service) GetReferralEarnings(ctx context.Context, userID uuid.UUID) (map[string]interface{}, error) {
	return s.repo.GetReferralEarnings(ctx, userID)
}
