package subscriptions

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// PaymentProcessor interface for subscription billing
type PaymentProcessor interface {
	ChargeSubscription(ctx context.Context, userID uuid.UUID, amount float64, currency, paymentMethod string) error
}

// Service handles subscription business logic
type Service struct {
	repo      RepositoryInterface
	payments  PaymentProcessor
}

// NewService creates a new subscription service
func NewService(repo RepositoryInterface, payments PaymentProcessor) *Service {
	return &Service{
		repo:     repo,
		payments: payments,
	}
}

// ========================================
// PLAN MANAGEMENT (Admin)
// ========================================

// CreatePlan creates a new subscription plan
func (s *Service) CreatePlan(ctx context.Context, req *CreatePlanRequest) (*SubscriptionPlan, error) {
	// Check for duplicate slug
	existing, _ := s.repo.GetPlanBySlug(ctx, req.Slug)
	if existing != nil {
		return nil, common.NewBadRequestError("plan slug already exists", nil)
	}

	now := time.Now()
	plan := &SubscriptionPlan{
		ID:                uuid.New(),
		Name:              req.Name,
		Slug:              req.Slug,
		Description:       req.Description,
		PlanType:          req.PlanType,
		BillingPeriod:     req.BillingPeriod,
		Price:             req.Price,
		Currency:          req.Currency,
		Status:            PlanStatusActive,
		RidesIncluded:     req.RidesIncluded,
		MaxRideValue:      req.MaxRideValue,
		DiscountPct:       req.DiscountPct,
		AllowedRideTypes:  req.AllowedRideTypes,
		AllowedCities:     req.AllowedCities,
		MaxDistanceKm:     req.MaxDistanceKm,
		PriorityMatching:  req.PriorityMatching,
		FreeUpgrades:      req.FreeUpgrades,
		FreeCancellations: req.FreeCancellations,
		SurgeProtection:   req.SurgeProtection,
		SurgeMaxCap:       req.SurgeMaxCap,
		PopularBadge:      req.PopularBadge,
		SavingsLabel:      req.SavingsLabel,
		DisplayOrder:      req.DisplayOrder,
		TrialDays:         req.TrialDays,
		TrialRides:        req.TrialRides,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.repo.CreatePlan(ctx, plan); err != nil {
		return nil, common.NewInternalServerError("failed to create plan")
	}

	logger.Info("Subscription plan created",
		zap.String("plan_id", plan.ID.String()),
		zap.String("name", plan.Name),
		zap.String("type", string(plan.PlanType)),
	)

	return plan, nil
}

// ListPlans lists available plans for users
func (s *Service) ListPlans(ctx context.Context) ([]*SubscriptionPlan, error) {
	return s.repo.ListActivePlans(ctx)
}

// ListAllPlans lists all plans (admin)
func (s *Service) ListAllPlans(ctx context.Context) ([]*SubscriptionPlan, error) {
	return s.repo.ListAllPlans(ctx)
}

// DeactivatePlan deactivates a plan (no new subscribers)
func (s *Service) DeactivatePlan(ctx context.Context, planID uuid.UUID) error {
	return s.repo.UpdatePlanStatus(ctx, planID, PlanStatusInactive)
}

// ========================================
// SUBSCRIPTION MANAGEMENT
// ========================================

// Subscribe creates a new subscription for a user
func (s *Service) Subscribe(ctx context.Context, userID uuid.UUID, req *SubscribeRequest) (*SubscriptionResponse, error) {
	// Check for existing active subscription
	existing, _ := s.repo.GetActiveSubscription(ctx, userID)
	if existing != nil {
		return nil, common.NewBadRequestError("you already have an active subscription", nil)
	}

	// Get plan
	plan, err := s.repo.GetPlanByID(ctx, req.PlanID)
	if err != nil || plan == nil {
		return nil, common.NewNotFoundError("plan not found", err)
	}

	if plan.Status != PlanStatusActive {
		return nil, common.NewBadRequestError("this plan is no longer available", nil)
	}

	now := time.Now()
	periodEnd := s.calculatePeriodEnd(now, plan.BillingPeriod)

	sub := &Subscription{
		ID:                 uuid.New(),
		UserID:             userID,
		PlanID:             plan.ID,
		Status:             SubStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   periodEnd,
		PaymentMethod:      req.PaymentMethod,
		AutoRenew:          req.AutoRenew,
		ActivatedAt:        &now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Handle trial
	if plan.TrialDays > 0 {
		sub.IsTrialActive = true
		trialEnd := now.AddDate(0, 0, plan.TrialDays)
		sub.TrialEndsAt = &trialEnd
		sub.CurrentPeriodEnd = trialEnd // Trial extends the first period
	} else {
		// Charge immediately if no trial
		if s.payments != nil {
			if err := s.payments.ChargeSubscription(ctx, userID, plan.Price, plan.Currency, req.PaymentMethod); err != nil {
				return nil, common.NewInternalServerError("payment processing failed")
			}
		}
		sub.LastPaymentDate = &now
	}

	nextBilling := sub.CurrentPeriodEnd
	sub.NextBillingDate = &nextBilling

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, common.NewInternalServerError("failed to create subscription")
	}

	logger.Info("Subscription created",
		zap.String("sub_id", sub.ID.String()),
		zap.String("user_id", userID.String()),
		zap.String("plan", plan.Name),
	)

	return s.buildResponse(ctx, sub, plan), nil
}

// GetSubscription gets the user's active subscription
func (s *Service) GetSubscription(ctx context.Context, userID uuid.UUID) (*SubscriptionResponse, error) {
	sub, err := s.repo.GetActiveSubscription(ctx, userID)
	if err != nil || sub == nil {
		return nil, common.NewNotFoundError("no active subscription found", err)
	}

	plan, _ := s.repo.GetPlanByID(ctx, sub.PlanID)

	return s.buildResponse(ctx, sub, plan), nil
}

// PauseSubscription pauses a subscription
func (s *Service) PauseSubscription(ctx context.Context, userID uuid.UUID) error {
	sub, err := s.repo.GetActiveSubscription(ctx, userID)
	if err != nil || sub == nil {
		return common.NewNotFoundError("no active subscription found", err)
	}

	if sub.Status != SubStatusActive {
		return common.NewBadRequestError("subscription is not active", nil)
	}

	now := time.Now()
	sub.Status = SubStatusPaused
	sub.PausedAt = &now
	sub.UpdatedAt = now

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return common.NewInternalServerError("failed to pause subscription")
	}

	logger.Info("Subscription paused", zap.String("sub_id", sub.ID.String()))
	return nil
}

// ResumeSubscription resumes a paused subscription
func (s *Service) ResumeSubscription(ctx context.Context, userID uuid.UUID) error {
	sub, err := s.repo.GetActiveSubscription(ctx, userID)
	if err != nil || sub == nil {
		return common.NewNotFoundError("no subscription found", err)
	}

	if sub.Status != SubStatusPaused {
		return common.NewBadRequestError("subscription is not paused", nil)
	}

	sub.Status = SubStatusActive
	sub.PausedAt = nil
	sub.UpdatedAt = time.Now()

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return common.NewInternalServerError("failed to resume subscription")
	}

	logger.Info("Subscription resumed", zap.String("sub_id", sub.ID.String()))
	return nil
}

// CancelSubscription cancels a subscription
func (s *Service) CancelSubscription(ctx context.Context, userID uuid.UUID, reason string) error {
	sub, err := s.repo.GetActiveSubscription(ctx, userID)
	if err != nil || sub == nil {
		return common.NewNotFoundError("no active subscription found", err)
	}

	now := time.Now()
	sub.Status = SubStatusCancelled
	sub.CancelledAt = &now
	sub.CancelReason = &reason
	sub.AutoRenew = false
	sub.UpdatedAt = now

	if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
		return common.NewInternalServerError("failed to cancel subscription")
	}

	logger.Info("Subscription cancelled",
		zap.String("sub_id", sub.ID.String()),
		zap.String("reason", reason),
	)
	return nil
}

// ========================================
// SUBSCRIPTION USAGE
// ========================================

// ApplySubscriptionDiscount applies subscription benefits to a ride fare
func (s *Service) ApplySubscriptionDiscount(ctx context.Context, userID, rideID uuid.UUID, originalFare float64, rideType string) (float64, error) {
	sub, err := s.repo.GetActiveSubscription(ctx, userID)
	if err != nil || sub == nil {
		return originalFare, nil // No subscription, return original fare
	}

	if sub.Status != SubStatusActive {
		return originalFare, nil
	}

	plan, err := s.repo.GetPlanByID(ctx, sub.PlanID)
	if err != nil || plan == nil {
		return originalFare, nil
	}

	// Check ride type eligibility
	if len(plan.AllowedRideTypes) > 0 {
		eligible := false
		for _, t := range plan.AllowedRideTypes {
			if t == rideType {
				eligible = true
				break
			}
		}
		if !eligible {
			return originalFare, nil
		}
	}

	// Check if rides limit reached
	if plan.RidesIncluded != nil && sub.RidesUsed >= *plan.RidesIncluded {
		return originalFare, nil
	}

	// Check per-ride cap
	fareToDiscount := originalFare
	if plan.MaxRideValue != nil && fareToDiscount > *plan.MaxRideValue {
		fareToDiscount = *plan.MaxRideValue
	}

	// Calculate discounted fare
	var discountedFare float64
	switch plan.PlanType {
	case PlanTypeUnlimited:
		discountedFare = 0 // Ride is fully covered
	case PlanTypePackage:
		discountedFare = 0 // Ride is covered by package
	case PlanTypeDiscount:
		discount := fareToDiscount * (plan.DiscountPct / 100)
		discountedFare = originalFare - discount
	case PlanTypePriority:
		discount := fareToDiscount * (plan.DiscountPct / 100)
		discountedFare = originalFare - discount
	default:
		discountedFare = originalFare
	}

	if discountedFare < 0 {
		discountedFare = 0
	}

	savings := originalFare - discountedFare

	// Record usage
	usageLog := &SubscriptionUsageLog{
		ID:             uuid.New(),
		SubscriptionID: sub.ID,
		RideID:         rideID,
		UsageType:      "ride",
		OriginalFare:   originalFare,
		DiscountedFare: discountedFare,
		SavingsAmount:  savings,
		CreatedAt:      time.Now(),
	}
	_ = s.repo.CreateUsageLog(ctx, usageLog)
	_ = s.repo.IncrementRideUsage(ctx, sub.ID, savings)

	return discountedFare, nil
}

// HasFreeCancellation checks if the user has free cancellations remaining
func (s *Service) HasFreeCancellation(ctx context.Context, userID uuid.UUID) bool {
	sub, err := s.repo.GetActiveSubscription(ctx, userID)
	if err != nil || sub == nil {
		return false
	}

	plan, err := s.repo.GetPlanByID(ctx, sub.PlanID)
	if err != nil || plan == nil {
		return false
	}

	return sub.CancellationsUsed < plan.FreeCancellations
}

// UseFreeCancellation records a free cancellation usage
func (s *Service) UseFreeCancellation(ctx context.Context, userID uuid.UUID) error {
	sub, err := s.repo.GetActiveSubscription(ctx, userID)
	if err != nil || sub == nil {
		return common.NewBadRequestError("no active subscription", nil)
	}

	return s.repo.IncrementCancellationUsage(ctx, sub.ID)
}

// HasSurgeProtection checks if the user has surge pricing protection
func (s *Service) HasSurgeProtection(ctx context.Context, userID uuid.UUID) (bool, *float64) {
	sub, err := s.repo.GetActiveSubscription(ctx, userID)
	if err != nil || sub == nil {
		return false, nil
	}

	plan, err := s.repo.GetPlanByID(ctx, sub.PlanID)
	if err != nil || plan == nil {
		return false, nil
	}

	return plan.SurgeProtection, plan.SurgeMaxCap
}

// HasPriorityMatching checks if the user gets priority driver matching
func (s *Service) HasPriorityMatching(ctx context.Context, userID uuid.UUID) bool {
	sub, err := s.repo.GetActiveSubscription(ctx, userID)
	if err != nil || sub == nil {
		return false
	}

	plan, err := s.repo.GetPlanByID(ctx, sub.PlanID)
	if err != nil || plan == nil {
		return false
	}

	return plan.PriorityMatching
}

// ========================================
// PLAN COMPARISON
// ========================================

// ComparePlans returns plans with personalized savings estimates
func (s *Service) ComparePlans(ctx context.Context, userID uuid.UUID) (*PlanComparisonResponse, error) {
	plans, err := s.repo.ListActivePlans(ctx)
	if err != nil {
		return nil, common.NewInternalServerError("failed to list plans")
	}

	avgSpend, _ := s.repo.GetUserAverageMonthlySpend(ctx, userID)

	savings := make(map[uuid.UUID]float64)
	var bestPlanID *uuid.UUID
	var bestSavings float64

	for _, plan := range plans {
		var estimated float64
		switch plan.PlanType {
		case PlanTypeDiscount:
			estimated = avgSpend * (plan.DiscountPct / 100) - plan.Price
		case PlanTypePackage:
			if plan.RidesIncluded != nil {
				avgPerRide := avgSpend / 20 // Assume ~20 rides/month
				estimated = avgPerRide*float64(*plan.RidesIncluded) - plan.Price
			}
		case PlanTypeUnlimited:
			estimated = avgSpend - plan.Price
		case PlanTypePriority:
			estimated = avgSpend*(plan.DiscountPct/100) - plan.Price
		}

		if estimated < 0 {
			estimated = 0
		}
		savings[plan.ID] = estimated

		if estimated > bestSavings {
			bestSavings = estimated
			id := plan.ID
			bestPlanID = &id
		}
	}

	return &PlanComparisonResponse{
		Plans:            plans,
		UserAvgSpend:     avgSpend,
		BestValuePlan:    bestPlanID,
		EstimatedSavings: savings,
	}, nil
}

// ========================================
// RENEWAL PROCESSING
// ========================================

// ProcessRenewals handles subscription renewals
func (s *Service) ProcessRenewals(ctx context.Context) error {
	expired, err := s.repo.GetExpiredSubscriptions(ctx)
	if err != nil {
		return err
	}

	for _, sub := range expired {
		plan, err := s.repo.GetPlanByID(ctx, sub.PlanID)
		if err != nil || plan == nil {
			continue
		}

		// End trial if active
		if sub.IsTrialActive {
			sub.IsTrialActive = false
		}

		// Charge for renewal
		if s.payments != nil {
			if err := s.payments.ChargeSubscription(ctx, sub.UserID, plan.Price, plan.Currency, sub.PaymentMethod); err != nil {
				sub.FailedPayments++
				if sub.FailedPayments >= 3 {
					sub.Status = SubStatusPastDue
				}
				_ = s.repo.UpdateSubscription(ctx, sub)

				logger.Warn("Subscription renewal failed",
					zap.String("sub_id", sub.ID.String()),
					zap.Int("failed_count", sub.FailedPayments),
				)
				continue
			}
		}

		// Reset period
		now := time.Now()
		sub.CurrentPeriodStart = now
		sub.CurrentPeriodEnd = s.calculatePeriodEnd(now, plan.BillingPeriod)
		sub.RidesUsed = 0
		sub.UpgradesUsed = 0
		sub.CancellationsUsed = 0
		sub.LastPaymentDate = &now
		sub.FailedPayments = 0
		nextBilling := sub.CurrentPeriodEnd
		sub.NextBillingDate = &nextBilling
		sub.UpdatedAt = now

		if err := s.repo.UpdateSubscription(ctx, sub); err != nil {
			logger.Error("Failed to update renewed subscription",
				zap.String("sub_id", sub.ID.String()),
				zap.Error(err),
			)
		}

		logger.Info("Subscription renewed",
			zap.String("sub_id", sub.ID.String()),
			zap.String("user_id", sub.UserID.String()),
		)
	}

	return nil
}

// ========================================
// HELPERS
// ========================================

func (s *Service) calculatePeriodEnd(start time.Time, period BillingPeriod) time.Time {
	switch period {
	case BillingWeekly:
		return start.AddDate(0, 0, 7)
	case BillingMonthly:
		return start.AddDate(0, 1, 0)
	case BillingYearly:
		return start.AddDate(1, 0, 0)
	default:
		return start.AddDate(0, 1, 0)
	}
}

func (s *Service) buildResponse(ctx context.Context, sub *Subscription, plan *SubscriptionPlan) *SubscriptionResponse {
	usage := &UsageSummary{
		RidesUsed:         sub.RidesUsed,
		UpgradesUsed:      sub.UpgradesUsed,
		CancellationsUsed: sub.CancellationsUsed,
		TotalSaved:        sub.TotalSaved,
	}

	if plan != nil {
		if plan.RidesIncluded != nil {
			remaining := *plan.RidesIncluded - sub.RidesUsed
			if remaining < 0 {
				remaining = 0
			}
			usage.RidesRemaining = &remaining
		}
		usage.UpgradesLeft = plan.FreeUpgrades - sub.UpgradesUsed
		if usage.UpgradesLeft < 0 {
			usage.UpgradesLeft = 0
		}
		usage.CancellationsLeft = plan.FreeCancellations - sub.CancellationsUsed
		if usage.CancellationsLeft < 0 {
			usage.CancellationsLeft = 0
		}

		// Calculate utilization
		if plan.RidesIncluded != nil && *plan.RidesIncluded > 0 {
			usage.UtilizationPct = float64(sub.RidesUsed) / float64(*plan.RidesIncluded) * 100
		}
	}

	// Days remaining
	daysRemaining := int(sub.CurrentPeriodEnd.Sub(time.Now()).Hours() / 24)
	if daysRemaining < 0 {
		daysRemaining = 0
	}
	usage.DaysRemaining = daysRemaining

	return &SubscriptionResponse{
		Subscription: sub,
		Plan:         plan,
		Usage:        usage,
	}
}
