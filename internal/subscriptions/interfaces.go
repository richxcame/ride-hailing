package subscriptions

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryInterface defines the interface for subscription repository operations
type RepositoryInterface interface {
	// Plans
	CreatePlan(ctx context.Context, plan *SubscriptionPlan) error
	GetPlanByID(ctx context.Context, id uuid.UUID) (*SubscriptionPlan, error)
	GetPlanBySlug(ctx context.Context, slug string) (*SubscriptionPlan, error)
	ListActivePlans(ctx context.Context) ([]*SubscriptionPlan, error)
	ListAllPlans(ctx context.Context) ([]*SubscriptionPlan, error)
	UpdatePlanStatus(ctx context.Context, id uuid.UUID, status PlanStatus) error

	// Subscriptions
	CreateSubscription(ctx context.Context, sub *Subscription) error
	GetActiveSubscription(ctx context.Context, userID uuid.UUID) (*Subscription, error)
	GetSubscriptionByID(ctx context.Context, id uuid.UUID) (*Subscription, error)
	UpdateSubscription(ctx context.Context, sub *Subscription) error
	IncrementRideUsage(ctx context.Context, subID uuid.UUID, savings float64) error
	IncrementUpgradeUsage(ctx context.Context, subID uuid.UUID) error
	IncrementCancellationUsage(ctx context.Context, subID uuid.UUID) error
	GetExpiredSubscriptions(ctx context.Context) ([]*Subscription, error)
	GetSubscriberCount(ctx context.Context, planID uuid.UUID) (int, error)

	// Usage Logs
	CreateUsageLog(ctx context.Context, log *SubscriptionUsageLog) error
	GetUsageLogs(ctx context.Context, subID uuid.UUID, limit, offset int) ([]*SubscriptionUsageLog, error)

	// Analytics
	GetUserAverageMonthlySpend(ctx context.Context, userID uuid.UUID) (float64, error)
}
