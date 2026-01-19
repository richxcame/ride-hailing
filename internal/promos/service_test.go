package promos

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockPromosRepository struct {
	mock.Mock
}

func (m *mockPromosRepository) GetPromoCodeByCode(ctx context.Context, code string) (*PromoCode, error) {
	args := m.Called(ctx, code)
	promo, _ := args.Get(0).(*PromoCode)
	return promo, args.Error(1)
}

func (m *mockPromosRepository) GetPromoCodeUsesByUser(ctx context.Context, promoID uuid.UUID, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, promoID, userID)
	return args.Int(0), args.Error(1)
}

func (m *mockPromosRepository) CreatePromoCodeUse(ctx context.Context, use *PromoCodeUse) error {
	args := m.Called(ctx, use)
	return args.Error(0)
}

func (m *mockPromosRepository) CreatePromoCode(ctx context.Context, promo *PromoCode) error {
	args := m.Called(ctx, promo)
	return args.Error(0)
}

func (m *mockPromosRepository) GetReferralCodeByUserID(ctx context.Context, userID uuid.UUID) (*ReferralCode, error) {
	args := m.Called(ctx, userID)
	ref, _ := args.Get(0).(*ReferralCode)
	return ref, args.Error(1)
}

func (m *mockPromosRepository) CreateReferralCode(ctx context.Context, code *ReferralCode) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *mockPromosRepository) GetReferralCodeByCode(ctx context.Context, code string) (*ReferralCode, error) {
	args := m.Called(ctx, code)
	ref, _ := args.Get(0).(*ReferralCode)
	return ref, args.Error(1)
}

func (m *mockPromosRepository) CreateReferral(ctx context.Context, referral *Referral) error {
	args := m.Called(ctx, referral)
	return args.Error(0)
}

func (m *mockPromosRepository) GetAllRideTypes(ctx context.Context) ([]*RideType, error) {
	args := m.Called(ctx)
	types, _ := args.Get(0).([]*RideType)
	return types, args.Error(1)
}

func (m *mockPromosRepository) GetRideTypeByID(ctx context.Context, id uuid.UUID) (*RideType, error) {
	args := m.Called(ctx, id)
	rideType, _ := args.Get(0).(*RideType)
	return rideType, args.Error(1)
}

func (m *mockPromosRepository) GetReferralByReferredID(ctx context.Context, userID uuid.UUID) (*Referral, error) {
	args := m.Called(ctx, userID)
	ref, _ := args.Get(0).(*Referral)
	return ref, args.Error(1)
}

func (m *mockPromosRepository) IsFirstCompletedRide(ctx context.Context, userID uuid.UUID, rideID uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID, rideID)
	return args.Bool(0), args.Error(1)
}

func (m *mockPromosRepository) MarkReferralBonusesApplied(ctx context.Context, referralID uuid.UUID, rideID uuid.UUID) error {
	args := m.Called(ctx, referralID, rideID)
	return args.Error(0)
}

func (m *mockPromosRepository) UpdatePromoCode(ctx context.Context, promo *PromoCode) error {
	args := m.Called(ctx, promo)
	return args.Error(0)
}

func (m *mockPromosRepository) DeactivatePromoCode(ctx context.Context, promoID uuid.UUID) error {
	args := m.Called(ctx, promoID)
	return args.Error(0)
}

func (m *mockPromosRepository) GetPromoCodeByID(ctx context.Context, promoID uuid.UUID) (*PromoCode, error) {
	args := m.Called(ctx, promoID)
	promo, _ := args.Get(0).(*PromoCode)
	return promo, args.Error(1)
}

func (m *mockPromosRepository) GetPromoCodeUsageStats(ctx context.Context, promoID uuid.UUID) (map[string]interface{}, error) {
	args := m.Called(ctx, promoID)
	stats, _ := args.Get(0).(map[string]interface{})
	return stats, args.Error(1)
}

func (m *mockPromosRepository) GetAllPromoCodes(ctx context.Context, limit, offset int) ([]*PromoCode, int, error) {
	args := m.Called(ctx, limit, offset)
	promos, _ := args.Get(0).([]*PromoCode)
	return promos, args.Int(1), args.Error(2)
}

func (m *mockPromosRepository) GetAllReferralCodes(ctx context.Context, limit, offset int) ([]*ReferralCode, int, error) {
	args := m.Called(ctx, limit, offset)
	codes, _ := args.Get(0).([]*ReferralCode)
	return codes, args.Int(1), args.Error(2)
}

func (m *mockPromosRepository) GetReferralByID(ctx context.Context, referralID uuid.UUID) (*Referral, error) {
	args := m.Called(ctx, referralID)
	ref, _ := args.Get(0).(*Referral)
	return ref, args.Error(1)
}

func (m *mockPromosRepository) GetReferralEarnings(ctx context.Context, userID uuid.UUID) (map[string]interface{}, error) {
	args := m.Called(ctx, userID)
	earnings, _ := args.Get(0).(map[string]interface{})
	return earnings, args.Error(1)
}

func TestValidatePromoCodePercentageDiscount(t *testing.T) {
	ctx := context.Background()
	repo := new(mockPromosRepository)
	service := NewService(repo)
	userID := uuid.New()
	promo := validPercentagePromo()

	repo.On("GetPromoCodeByCode", ctx, "SAVE20").Return(promo, nil).Once()
	repo.On("GetPromoCodeUsesByUser", ctx, promo.ID, userID).Return(0, nil).Once()

	result, err := service.ValidatePromoCode(ctx, "SAVE20", userID, 50)
	assert.NoError(t, err)
	assert.True(t, result.Valid)
	assert.InDelta(t, 10.0, result.DiscountAmount, 0.0001)
	assert.InDelta(t, 40.0, result.FinalAmount, 0.0001)
	repo.AssertExpectations(t)
}

func TestValidatePromoCodeMinRideAmountFailure(t *testing.T) {
	ctx := context.Background()
	repo := new(mockPromosRepository)
	service := NewService(repo)
	userID := uuid.New()
	minAmount := 100.0
	promo := validPercentagePromo()
	promo.MinRideAmount = &minAmount

	repo.On("GetPromoCodeByCode", ctx, "SAVE20").Return(promo, nil).Once()
	repo.On("GetPromoCodeUsesByUser", ctx, promo.ID, userID).Return(0, nil).Once()

	result, err := service.ValidatePromoCode(ctx, "SAVE20", userID, 50)
	assert.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "Minimum ride amount")
	repo.AssertExpectations(t)
}

func TestApplyPromoCodeCreatesUseRecord(t *testing.T) {
	ctx := context.Background()
	repo := new(mockPromosRepository)
	service := NewService(repo)
	userID := uuid.New()
	rideID := uuid.New()
	promo := validFixedPromo()

	repo.On("GetPromoCodeByCode", ctx, "WELCOME5").Return(promo, nil).Twice()
	repo.On("GetPromoCodeUsesByUser", ctx, promo.ID, userID).Return(0, nil).Once()
	repo.On("CreatePromoCodeUse", ctx, mock.MatchedBy(func(use *PromoCodeUse) bool {
		return use.PromoCodeID == promo.ID &&
			use.UserID == userID &&
			use.RideID == rideID &&
			use.DiscountAmount == 5 &&
			use.FinalAmount == 45
	})).Return(nil).Once()

	use, err := service.ApplyPromoCode(ctx, "WELCOME5", userID, rideID, 50)
	assert.NoError(t, err)
	assert.Equal(t, promo.ID, use.PromoCodeID)
	assert.InDelta(t, 5.0, use.DiscountAmount, 0.0001)
	repo.AssertExpectations(t)
}

func TestApplyPromoCodeInvalidWhenNotActive(t *testing.T) {
	ctx := context.Background()
	repo := new(mockPromosRepository)
	service := NewService(repo)
	userID := uuid.New()
	rideID := uuid.New()
	promo := validFixedPromo()
	promo.IsActive = false

	repo.On("GetPromoCodeByCode", ctx, "WELCOME5").Return(promo, nil).Once()

	use, err := service.ApplyPromoCode(ctx, "WELCOME5", userID, rideID, 50)
	assert.Error(t, err)
	assert.Nil(t, use)
	repo.AssertExpectations(t)
}

func TestProcessReferralBonusFirstRide(t *testing.T) {
	ctx := context.Background()
	repo := new(mockPromosRepository)
	service := NewService(repo)
	userID := uuid.New()
	rideID := uuid.New()
	referral := &Referral{
		ID:                   uuid.New(),
		ReferrerID:           uuid.New(),
		ReferredID:           userID,
		ReferrerBonus:        ReferrerBonusAmount,
		ReferredBonus:        ReferredBonusAmount,
		ReferrerBonusApplied: false,
		ReferredBonusApplied: false,
	}

	repo.On("GetReferralByReferredID", ctx, userID).Return(referral, nil).Once()
	repo.On("IsFirstCompletedRide", ctx, userID, rideID).Return(true, nil).Once()
	repo.On("MarkReferralBonusesApplied", ctx, referral.ID, rideID).Return(nil).Once()

	result, err := service.ProcessReferralBonus(ctx, userID, rideID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result["has_bonus"].(bool))
	assert.Equal(t, referral.ReferrerID, result["referrer_id"])
	assert.Equal(t, referral.ReferrerBonus, result["referrer_bonus"])
	assert.Equal(t, referral.ReferredID, result["referred_id"])
	assert.Equal(t, referral.ReferredBonus, result["referred_bonus"])
	repo.AssertExpectations(t)
}

func TestProcessReferralBonusSkipsWhenNotFirstRide(t *testing.T) {
	ctx := context.Background()
	repo := new(mockPromosRepository)
	service := NewService(repo)
	userID := uuid.New()
	rideID := uuid.New()
	referral := &Referral{
		ID:                   uuid.New(),
		ReferrerID:           uuid.New(),
		ReferredID:           userID,
		ReferrerBonus:        ReferrerBonusAmount,
		ReferredBonus:        ReferredBonusAmount,
		ReferrerBonusApplied: false,
		ReferredBonusApplied: false,
	}

	repo.On("GetReferralByReferredID", ctx, userID).Return(referral, nil).Once()
	repo.On("IsFirstCompletedRide", ctx, userID, rideID).Return(false, nil).Once()

	result, err := service.ProcessReferralBonus(ctx, userID, rideID)
	assert.NoError(t, err)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestApplyReferralCodePreventsSelfReferral(t *testing.T) {
	ctx := context.Background()
	repo := new(mockPromosRepository)
	service := NewService(repo)
	userID := uuid.New()
	code := &ReferralCode{
		ID:     uuid.New(),
		UserID: userID,
		Code:   "SELF1234",
	}

	repo.On("GetReferralCodeByCode", ctx, "SELF1234").Return(code, nil).Once()

	err := service.ApplyReferralCode(ctx, "SELF1234", userID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot use your own referral code")
	repo.AssertExpectations(t)
}

func TestApplyReferralCodeCreatesReferral(t *testing.T) {
	ctx := context.Background()
	repo := new(mockPromosRepository)
	service := NewService(repo)
	referrerID := uuid.New()
	newUserID := uuid.New()
	code := &ReferralCode{
		ID:     uuid.New(),
		UserID: referrerID,
		Code:   "REFER123",
	}

	repo.On("GetReferralCodeByCode", ctx, "REFER123").Return(code, nil).Once()
	repo.On("CreateReferral", ctx, mock.MatchedBy(func(r *Referral) bool {
		return r.ReferrerID == referrerID &&
			r.ReferredID == newUserID &&
			r.ReferrerBonus == ReferrerBonusAmount &&
			r.ReferredBonus == ReferredBonusAmount
	})).Return(nil).Once()

	err := service.ApplyReferralCode(ctx, "REFER123", newUserID)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestGenerateReferralCodeReturnsExisting(t *testing.T) {
	ctx := context.Background()
	repo := new(mockPromosRepository)
	service := NewService(repo)
	userID := uuid.New()
	existing := &ReferralCode{
		ID:     uuid.New(),
		UserID: userID,
		Code:   "EXIST1",
	}

	repo.On("GetReferralCodeByUserID", ctx, userID).Return(existing, nil).Once()

	result, err := service.GenerateReferralCode(ctx, userID, "base")
	assert.NoError(t, err)
	assert.Equal(t, existing, result)
	repo.AssertExpectations(t)
}

func TestGenerateReferralCodeCreatesNew(t *testing.T) {
	ctx := context.Background()
	repo := new(mockPromosRepository)
	service := NewService(repo)
	userID := uuid.New()

	repo.On("GetReferralCodeByUserID", ctx, userID).Return((*ReferralCode)(nil), errors.New("not found")).Once()
	repo.On("CreateReferralCode", ctx, mock.MatchedBy(func(code *ReferralCode) bool {
		return code.UserID == userID && code.Code != ""
	})).Return(nil).Once()

	result, err := service.GenerateReferralCode(ctx, userID, "friend")
	assert.NoError(t, err)
	assert.Equal(t, userID, result.UserID)
	assert.NotEmpty(t, result.Code)
	repo.AssertExpectations(t)
}

func validPercentagePromo() *PromoCode {
	return &PromoCode{
		ID:                uuid.New(),
		Code:              "SAVE20",
		DiscountType:      "percentage",
		DiscountValue:     20,
		UsesPerUser:       2,
		IsActive:          true,
		ValidFrom:         time.Now().Add(-time.Hour),
		ValidUntil:        time.Now().Add(time.Hour),
		TotalUses:         0,
		MaxDiscountAmount: nil,
	}
}

func validFixedPromo() *PromoCode {
	return &PromoCode{
		ID:            uuid.New(),
		Code:          "WELCOME5",
		DiscountType:  "fixed_amount",
		DiscountValue: 5,
		UsesPerUser:   1,
		IsActive:      true,
		ValidFrom:     time.Now().Add(-time.Hour),
		ValidUntil:    time.Now().Add(time.Hour),
	}
}
