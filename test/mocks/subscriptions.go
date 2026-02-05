package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/internal/subscriptions"
	"github.com/stretchr/testify/mock"
)

// MockSubscriptionsRepository is a mock implementation of the subscriptions repository
type MockSubscriptionsRepository struct {
	mock.Mock
}

// Plans

func (m *MockSubscriptionsRepository) CreatePlan(ctx context.Context, plan *subscriptions.SubscriptionPlan) error {
	args := m.Called(ctx, plan)
	return args.Error(0)
}

func (m *MockSubscriptionsRepository) GetPlanByID(ctx context.Context, id uuid.UUID) (*subscriptions.SubscriptionPlan, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*subscriptions.SubscriptionPlan), args.Error(1)
}

func (m *MockSubscriptionsRepository) GetPlanBySlug(ctx context.Context, slug string) (*subscriptions.SubscriptionPlan, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*subscriptions.SubscriptionPlan), args.Error(1)
}

func (m *MockSubscriptionsRepository) ListActivePlans(ctx context.Context) ([]*subscriptions.SubscriptionPlan, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*subscriptions.SubscriptionPlan), args.Error(1)
}

func (m *MockSubscriptionsRepository) ListAllPlans(ctx context.Context) ([]*subscriptions.SubscriptionPlan, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*subscriptions.SubscriptionPlan), args.Error(1)
}

func (m *MockSubscriptionsRepository) UpdatePlanStatus(ctx context.Context, id uuid.UUID, status subscriptions.PlanStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

// Subscriptions

func (m *MockSubscriptionsRepository) CreateSubscription(ctx context.Context, sub *subscriptions.Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}

func (m *MockSubscriptionsRepository) GetActiveSubscription(ctx context.Context, userID uuid.UUID) (*subscriptions.Subscription, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*subscriptions.Subscription), args.Error(1)
}

func (m *MockSubscriptionsRepository) GetSubscriptionByID(ctx context.Context, id uuid.UUID) (*subscriptions.Subscription, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*subscriptions.Subscription), args.Error(1)
}

func (m *MockSubscriptionsRepository) UpdateSubscription(ctx context.Context, sub *subscriptions.Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}

func (m *MockSubscriptionsRepository) IncrementRideUsage(ctx context.Context, subID uuid.UUID, savings float64) error {
	args := m.Called(ctx, subID, savings)
	return args.Error(0)
}

func (m *MockSubscriptionsRepository) IncrementUpgradeUsage(ctx context.Context, subID uuid.UUID) error {
	args := m.Called(ctx, subID)
	return args.Error(0)
}

func (m *MockSubscriptionsRepository) IncrementCancellationUsage(ctx context.Context, subID uuid.UUID) error {
	args := m.Called(ctx, subID)
	return args.Error(0)
}

func (m *MockSubscriptionsRepository) GetExpiredSubscriptions(ctx context.Context) ([]*subscriptions.Subscription, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*subscriptions.Subscription), args.Error(1)
}

func (m *MockSubscriptionsRepository) GetSubscriberCount(ctx context.Context, planID uuid.UUID) (int, error) {
	args := m.Called(ctx, planID)
	return args.Int(0), args.Error(1)
}

// Usage Logs

func (m *MockSubscriptionsRepository) CreateUsageLog(ctx context.Context, log *subscriptions.SubscriptionUsageLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockSubscriptionsRepository) GetUsageLogs(ctx context.Context, subID uuid.UUID, limit, offset int) ([]*subscriptions.SubscriptionUsageLog, error) {
	args := m.Called(ctx, subID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*subscriptions.SubscriptionUsageLog), args.Error(1)
}

// Analytics

func (m *MockSubscriptionsRepository) GetUserAverageMonthlySpend(ctx context.Context, userID uuid.UUID) (float64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(float64), args.Error(1)
}

// MockPaymentProcessor is a mock implementation of the PaymentProcessor interface
type MockPaymentProcessor struct {
	mock.Mock
}

func (m *MockPaymentProcessor) ChargeSubscription(ctx context.Context, userID uuid.UUID, amount float64, currency, paymentMethod string) error {
	args := m.Called(ctx, userID, amount, currency, paymentMethod)
	return args.Error(0)
}
