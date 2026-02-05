package subscriptions

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
// MOCK REPOSITORY (in-package)
// ========================================

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) CreatePlan(ctx context.Context, plan *SubscriptionPlan) error {
	args := m.Called(ctx, plan)
	return args.Error(0)
}

func (m *mockRepo) GetPlanByID(ctx context.Context, id uuid.UUID) (*SubscriptionPlan, error) {
	args := m.Called(ctx, id)
	plan, _ := args.Get(0).(*SubscriptionPlan)
	return plan, args.Error(1)
}

func (m *mockRepo) GetPlanBySlug(ctx context.Context, slug string) (*SubscriptionPlan, error) {
	args := m.Called(ctx, slug)
	plan, _ := args.Get(0).(*SubscriptionPlan)
	return plan, args.Error(1)
}

func (m *mockRepo) ListActivePlans(ctx context.Context) ([]*SubscriptionPlan, error) {
	args := m.Called(ctx)
	plans, _ := args.Get(0).([]*SubscriptionPlan)
	return plans, args.Error(1)
}

func (m *mockRepo) ListAllPlans(ctx context.Context) ([]*SubscriptionPlan, error) {
	args := m.Called(ctx)
	plans, _ := args.Get(0).([]*SubscriptionPlan)
	return plans, args.Error(1)
}

func (m *mockRepo) UpdatePlanStatus(ctx context.Context, id uuid.UUID, status PlanStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *mockRepo) CreateSubscription(ctx context.Context, sub *Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}

func (m *mockRepo) GetActiveSubscription(ctx context.Context, userID uuid.UUID) (*Subscription, error) {
	args := m.Called(ctx, userID)
	sub, _ := args.Get(0).(*Subscription)
	return sub, args.Error(1)
}

func (m *mockRepo) GetSubscriptionByID(ctx context.Context, id uuid.UUID) (*Subscription, error) {
	args := m.Called(ctx, id)
	sub, _ := args.Get(0).(*Subscription)
	return sub, args.Error(1)
}

func (m *mockRepo) UpdateSubscription(ctx context.Context, sub *Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}

func (m *mockRepo) IncrementRideUsage(ctx context.Context, subID uuid.UUID, savings float64) error {
	args := m.Called(ctx, subID, savings)
	return args.Error(0)
}

func (m *mockRepo) IncrementUpgradeUsage(ctx context.Context, subID uuid.UUID) error {
	args := m.Called(ctx, subID)
	return args.Error(0)
}

func (m *mockRepo) IncrementCancellationUsage(ctx context.Context, subID uuid.UUID) error {
	args := m.Called(ctx, subID)
	return args.Error(0)
}

func (m *mockRepo) GetExpiredSubscriptions(ctx context.Context) ([]*Subscription, error) {
	args := m.Called(ctx)
	subs, _ := args.Get(0).([]*Subscription)
	return subs, args.Error(1)
}

func (m *mockRepo) GetSubscriberCount(ctx context.Context, planID uuid.UUID) (int, error) {
	args := m.Called(ctx, planID)
	return args.Int(0), args.Error(1)
}

func (m *mockRepo) CreateUsageLog(ctx context.Context, log *SubscriptionUsageLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *mockRepo) GetUsageLogs(ctx context.Context, subID uuid.UUID, limit, offset int) ([]*SubscriptionUsageLog, error) {
	args := m.Called(ctx, subID, limit, offset)
	logs, _ := args.Get(0).([]*SubscriptionUsageLog)
	return logs, args.Error(1)
}

func (m *mockRepo) GetUserAverageMonthlySpend(ctx context.Context, userID uuid.UUID) (float64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(float64), args.Error(1)
}

// ========================================
// MOCK PAYMENT PROCESSOR
// ========================================

type mockPayments struct {
	mock.Mock
}

func (m *mockPayments) ChargeSubscription(ctx context.Context, userID uuid.UUID, amount float64, currency, paymentMethod string) error {
	args := m.Called(ctx, userID, amount, currency, paymentMethod)
	return args.Error(0)
}

// ========================================
// TEST HELPERS
// ========================================

func newActivePlan() *SubscriptionPlan {
	return &SubscriptionPlan{
		ID:                uuid.New(),
		Name:              "Monthly Ride Pass",
		Slug:              "monthly-ride-pass",
		Description:       "Unlimited rides for a month",
		PlanType:          PlanTypeUnlimited,
		BillingPeriod:     BillingMonthly,
		Price:             49.99,
		Currency:          "USD",
		Status:            PlanStatusActive,
		FreeCancellations: 3,
		SurgeProtection:   true,
		SurgeMaxCap:       ptrFloat64(1.5),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

func newDiscountPlan() *SubscriptionPlan {
	ridesIncluded := 30
	return &SubscriptionPlan{
		ID:                uuid.New(),
		Name:              "Discount Pass",
		Slug:              "discount-pass",
		Description:       "15% off all rides",
		PlanType:          PlanTypeDiscount,
		BillingPeriod:     BillingMonthly,
		Price:             19.99,
		Currency:          "USD",
		Status:            PlanStatusActive,
		DiscountPct:       15,
		RidesIncluded:     &ridesIncluded,
		FreeCancellations: 2,
		SurgeProtection:   false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

func newTrialPlan() *SubscriptionPlan {
	plan := newActivePlan()
	plan.TrialDays = 7
	plan.TrialRides = 5
	return plan
}

func newActiveSubscription(userID, planID uuid.UUID) *Subscription {
	now := time.Now()
	nextBilling := now.AddDate(0, 1, 0)
	return &Subscription{
		ID:                 uuid.New(),
		UserID:             userID,
		PlanID:             planID,
		Status:             SubStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		PaymentMethod:      "card",
		AutoRenew:          true,
		NextBillingDate:    &nextBilling,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

func newPausedSubscription(userID, planID uuid.UUID) *Subscription {
	sub := newActiveSubscription(userID, planID)
	sub.Status = SubStatusPaused
	pausedAt := time.Now()
	sub.PausedAt = &pausedAt
	return sub
}

func ptrFloat64(v float64) *float64 {
	return &v
}

// ========================================
// SUBSCRIBE TESTS
// ========================================

func TestSubscribe(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(repo *mockRepo, payments *mockPayments, userID uuid.UUID, plan *SubscriptionPlan)
		plan        func() *SubscriptionPlan
		wantErr     bool
		errContains string
		checkResult func(t *testing.T, resp *SubscriptionResponse, plan *SubscriptionPlan)
	}{
		{
			name: "success with active plan",
			plan: newActivePlan,
			setup: func(repo *mockRepo, payments *mockPayments, userID uuid.UUID, plan *SubscriptionPlan) {
				repo.On("GetActiveSubscription", mock.Anything, userID).Return((*Subscription)(nil), nil).Once()
				repo.On("GetPlanByID", mock.Anything, plan.ID).Return(plan, nil).Once()
				payments.On("ChargeSubscription", mock.Anything, userID, plan.Price, plan.Currency, "card").Return(nil).Once()
				repo.On("CreateSubscription", mock.Anything, mock.AnythingOfType("*subscriptions.Subscription")).Return(nil).Once()
			},
			wantErr: false,
			checkResult: func(t *testing.T, resp *SubscriptionResponse, plan *SubscriptionPlan) {
				require.NotNil(t, resp)
				assert.Equal(t, SubStatusActive, resp.Subscription.Status)
				assert.Equal(t, plan.ID, resp.Subscription.PlanID)
				assert.False(t, resp.Subscription.IsTrialActive)
				assert.NotNil(t, resp.Subscription.LastPaymentDate)
			},
		},
		{
			name: "plan not found",
			plan: newActivePlan,
			setup: func(repo *mockRepo, payments *mockPayments, userID uuid.UUID, plan *SubscriptionPlan) {
				repo.On("GetActiveSubscription", mock.Anything, userID).Return((*Subscription)(nil), nil).Once()
				repo.On("GetPlanByID", mock.Anything, plan.ID).Return((*SubscriptionPlan)(nil), errors.New("not found")).Once()
			},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name: "already has active subscription",
			plan: newActivePlan,
			setup: func(repo *mockRepo, payments *mockPayments, userID uuid.UUID, plan *SubscriptionPlan) {
				existing := newActiveSubscription(userID, plan.ID)
				repo.On("GetActiveSubscription", mock.Anything, userID).Return(existing, nil).Once()
			},
			wantErr:     true,
			errContains: "already have an active subscription",
		},
		{
			name: "with trial period",
			plan: newTrialPlan,
			setup: func(repo *mockRepo, payments *mockPayments, userID uuid.UUID, plan *SubscriptionPlan) {
				repo.On("GetActiveSubscription", mock.Anything, userID).Return((*Subscription)(nil), nil).Once()
				repo.On("GetPlanByID", mock.Anything, plan.ID).Return(plan, nil).Once()
				// No ChargeSubscription call expected for trial
				repo.On("CreateSubscription", mock.Anything, mock.AnythingOfType("*subscriptions.Subscription")).Return(nil).Once()
			},
			wantErr: false,
			checkResult: func(t *testing.T, resp *SubscriptionResponse, plan *SubscriptionPlan) {
				require.NotNil(t, resp)
				assert.Equal(t, SubStatusActive, resp.Subscription.Status)
				assert.True(t, resp.Subscription.IsTrialActive)
				assert.NotNil(t, resp.Subscription.TrialEndsAt)
				assert.Nil(t, resp.Subscription.LastPaymentDate)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockRepo)
			payments := new(mockPayments)
			service := NewService(repo, payments)

			ctx := context.Background()
			userID := uuid.New()
			plan := tt.plan()

			tt.setup(repo, payments, userID, plan)

			req := &SubscribeRequest{
				PlanID:        plan.ID,
				PaymentMethod: "card",
				AutoRenew:     true,
			}

			resp, err := service.Subscribe(ctx, userID, req)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, resp, plan)
				}
			}
			repo.AssertExpectations(t)
			payments.AssertExpectations(t)
		})
	}
}

// ========================================
// GET SUBSCRIPTION TESTS
// ========================================

func TestGetSubscription(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(repo *mockRepo, userID uuid.UUID, plan *SubscriptionPlan)
		wantErr     bool
		errContains string
		checkResult func(t *testing.T, resp *SubscriptionResponse)
	}{
		{
			name: "success",
			setup: func(repo *mockRepo, userID uuid.UUID, plan *SubscriptionPlan) {
				sub := newActiveSubscription(userID, plan.ID)
				repo.On("GetActiveSubscription", mock.Anything, userID).Return(sub, nil).Once()
				repo.On("GetPlanByID", mock.Anything, plan.ID).Return(plan, nil).Once()
			},
			wantErr: false,
			checkResult: func(t *testing.T, resp *SubscriptionResponse) {
				require.NotNil(t, resp)
				assert.Equal(t, SubStatusActive, resp.Subscription.Status)
				assert.NotNil(t, resp.Plan)
				assert.NotNil(t, resp.Usage)
			},
		},
		{
			name: "no active subscription",
			setup: func(repo *mockRepo, userID uuid.UUID, plan *SubscriptionPlan) {
				repo.On("GetActiveSubscription", mock.Anything, userID).Return((*Subscription)(nil), nil).Once()
			},
			wantErr:     true,
			errContains: "no active subscription found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockRepo)
			service := NewService(repo, nil)

			ctx := context.Background()
			userID := uuid.New()
			plan := newActivePlan()

			tt.setup(repo, userID, plan)

			resp, err := service.GetSubscription(ctx, userID)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, resp)
				}
			}
			repo.AssertExpectations(t)
		})
	}
}

// ========================================
// PAUSE SUBSCRIPTION TESTS
// ========================================

func TestPauseSubscription(t *testing.T) {
	t.Run("success active to paused", func(t *testing.T) {
		repo := new(mockRepo)
		service := NewService(repo, nil)

		ctx := context.Background()
		userID := uuid.New()
		plan := newActivePlan()
		sub := newActiveSubscription(userID, plan.ID)

		repo.On("GetActiveSubscription", mock.Anything, userID).Return(sub, nil).Once()
		repo.On("UpdateSubscription", mock.Anything, mock.MatchedBy(func(s *Subscription) bool {
			return s.Status == SubStatusPaused && s.PausedAt != nil
		})).Return(nil).Once()

		err := service.PauseSubscription(ctx, userID)
		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

// ========================================
// RESUME SUBSCRIPTION TESTS
// ========================================

func TestResumeSubscription(t *testing.T) {
	t.Run("success paused to active", func(t *testing.T) {
		repo := new(mockRepo)
		service := NewService(repo, nil)

		ctx := context.Background()
		userID := uuid.New()
		plan := newActivePlan()
		sub := newPausedSubscription(userID, plan.ID)

		repo.On("GetActiveSubscription", mock.Anything, userID).Return(sub, nil).Once()
		repo.On("UpdateSubscription", mock.Anything, mock.MatchedBy(func(s *Subscription) bool {
			return s.Status == SubStatusActive && s.PausedAt == nil
		})).Return(nil).Once()

		err := service.ResumeSubscription(ctx, userID)
		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

// ========================================
// CANCEL SUBSCRIPTION TESTS
// ========================================

func TestCancelSubscription(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := new(mockRepo)
		service := NewService(repo, nil)

		ctx := context.Background()
		userID := uuid.New()
		plan := newActivePlan()
		sub := newActiveSubscription(userID, plan.ID)

		repo.On("GetActiveSubscription", mock.Anything, userID).Return(sub, nil).Once()
		repo.On("UpdateSubscription", mock.Anything, mock.MatchedBy(func(s *Subscription) bool {
			return s.Status == SubStatusCancelled &&
				s.CancelledAt != nil &&
				s.CancelReason != nil &&
				*s.CancelReason == "too expensive" &&
				s.AutoRenew == false
		})).Return(nil).Once()

		err := service.CancelSubscription(ctx, userID, "too expensive")
		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

// ========================================
// APPLY SUBSCRIPTION DISCOUNT TESTS
// ========================================

func TestApplySubscriptionDiscount(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(repo *mockRepo, userID uuid.UUID) (planID uuid.UUID)
		originalFare float64
		rideType     string
		expectedFare float64
	}{
		{
			name: "unlimited plan fare becomes 0",
			setup: func(repo *mockRepo, userID uuid.UUID) uuid.UUID {
				plan := newActivePlan()
				plan.PlanType = PlanTypeUnlimited
				sub := newActiveSubscription(userID, plan.ID)

				repo.On("GetActiveSubscription", mock.Anything, userID).Return(sub, nil).Once()
				repo.On("GetPlanByID", mock.Anything, plan.ID).Return(plan, nil).Once()
				repo.On("CreateUsageLog", mock.Anything, mock.AnythingOfType("*subscriptions.SubscriptionUsageLog")).Return(nil).Once()
				repo.On("IncrementRideUsage", mock.Anything, sub.ID, 25.0).Return(nil).Once()
				return plan.ID
			},
			originalFare: 25.0,
			rideType:     "standard",
			expectedFare: 0,
		},
		{
			name: "percentage discount",
			setup: func(repo *mockRepo, userID uuid.UUID) uuid.UUID {
				plan := newDiscountPlan()
				sub := newActiveSubscription(userID, plan.ID)

				repo.On("GetActiveSubscription", mock.Anything, userID).Return(sub, nil).Once()
				repo.On("GetPlanByID", mock.Anything, plan.ID).Return(plan, nil).Once()
				// 15% of 100 = 15, so discounted fare = 85, savings = 15
				repo.On("CreateUsageLog", mock.Anything, mock.AnythingOfType("*subscriptions.SubscriptionUsageLog")).Return(nil).Once()
				repo.On("IncrementRideUsage", mock.Anything, sub.ID, 15.0).Return(nil).Once()
				return plan.ID
			},
			originalFare: 100.0,
			rideType:     "standard",
			expectedFare: 85.0,
		},
		{
			name: "rides exhausted no discount",
			setup: func(repo *mockRepo, userID uuid.UUID) uuid.UUID {
				plan := newDiscountPlan()
				ridesIncluded := 10
				plan.RidesIncluded = &ridesIncluded
				sub := newActiveSubscription(userID, plan.ID)
				sub.RidesUsed = 10 // Exhausted

				repo.On("GetActiveSubscription", mock.Anything, userID).Return(sub, nil).Once()
				repo.On("GetPlanByID", mock.Anything, plan.ID).Return(plan, nil).Once()
				return plan.ID
			},
			originalFare: 50.0,
			rideType:     "standard",
			expectedFare: 50.0, // No discount applied
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockRepo)
			service := NewService(repo, nil)

			ctx := context.Background()
			userID := uuid.New()
			rideID := uuid.New()

			tt.setup(repo, userID)

			fare, err := service.ApplySubscriptionDiscount(ctx, userID, rideID, tt.originalFare, tt.rideType)
			require.NoError(t, err)
			assert.InDelta(t, tt.expectedFare, fare, 0.01)
			repo.AssertExpectations(t)
		})
	}
}

// ========================================
// HAS FREE CANCELLATION TESTS
// ========================================

func TestHasFreeCancellation(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(repo *mockRepo, userID uuid.UUID)
		expected bool
	}{
		{
			name: "true when remaining greater than 0",
			setup: func(repo *mockRepo, userID uuid.UUID) {
				plan := newActivePlan()
				plan.FreeCancellations = 3
				sub := newActiveSubscription(userID, plan.ID)
				sub.CancellationsUsed = 1

				repo.On("GetActiveSubscription", mock.Anything, userID).Return(sub, nil).Once()
				repo.On("GetPlanByID", mock.Anything, plan.ID).Return(plan, nil).Once()
			},
			expected: true,
		},
		{
			name: "false when 0 remaining",
			setup: func(repo *mockRepo, userID uuid.UUID) {
				plan := newActivePlan()
				plan.FreeCancellations = 3
				sub := newActiveSubscription(userID, plan.ID)
				sub.CancellationsUsed = 3

				repo.On("GetActiveSubscription", mock.Anything, userID).Return(sub, nil).Once()
				repo.On("GetPlanByID", mock.Anything, plan.ID).Return(plan, nil).Once()
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockRepo)
			service := NewService(repo, nil)

			ctx := context.Background()
			userID := uuid.New()

			tt.setup(repo, userID)

			result := service.HasFreeCancellation(ctx, userID)
			assert.Equal(t, tt.expected, result)
			repo.AssertExpectations(t)
		})
	}
}

// ========================================
// HAS SURGE PROTECTION TESTS
// ========================================

func TestHasSurgeProtection(t *testing.T) {
	t.Run("returns cap", func(t *testing.T) {
		repo := new(mockRepo)
		service := NewService(repo, nil)

		ctx := context.Background()
		userID := uuid.New()
		plan := newActivePlan()
		plan.SurgeProtection = true
		expectedCap := 1.5
		plan.SurgeMaxCap = &expectedCap
		sub := newActiveSubscription(userID, plan.ID)

		repo.On("GetActiveSubscription", mock.Anything, userID).Return(sub, nil).Once()
		repo.On("GetPlanByID", mock.Anything, plan.ID).Return(plan, nil).Once()

		hasSurge, cap := service.HasSurgeProtection(ctx, userID)
		assert.True(t, hasSurge)
		require.NotNil(t, cap)
		assert.InDelta(t, expectedCap, *cap, 0.01)
		repo.AssertExpectations(t)
	})
}
