package fraud

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
)

// Service handles fraud detection business logic
type Service struct {
	repo *Repository
}

// NewService creates a new fraud detection service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// AnalyzeUser performs comprehensive fraud analysis on a user
func (s *Service) AnalyzeUser(ctx context.Context, userID uuid.UUID) (*UserRiskProfile, error) {
	// Get current risk profile
	profile, err := s.repo.GetUserRiskProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Analyze payment patterns
	paymentIndicators, err := s.repo.GetPaymentFraudIndicators(ctx, userID)
	if err != nil {
		paymentIndicators = &PaymentFraudIndicators{UserID: userID, RiskScore: 0}
	}

	// Analyze ride patterns
	rideIndicators, err := s.repo.GetRideFraudIndicators(ctx, userID)
	if err != nil {
		rideIndicators = &RideFraudIndicators{UserID: userID, RiskScore: 0}
	}

	// Calculate overall risk score (weighted average)
	overallRiskScore := (paymentIndicators.RiskScore * 0.6) + (rideIndicators.RiskScore * 0.4)
	profile.RiskScore = overallRiskScore
	profile.LastUpdated = time.Now()

	// Create alerts for high-risk indicators
	if paymentIndicators.RiskScore >= 70 {
		alert := &FraudAlert{
			ID:          uuid.New(),
			UserID:      userID,
			AlertType:   AlertTypePaymentFraud,
			AlertLevel:  s.determineAlertLevel(paymentIndicators.RiskScore),
			Status:      AlertStatusPending,
			Description: "Suspicious payment activity detected",
			Details: map[string]interface{}{
				"failed_attempts":        paymentIndicators.FailedPaymentAttempts,
				"chargebacks":            paymentIndicators.ChargebackCount,
				"multiple_payment_methods": paymentIndicators.MultiplePaymentMethods,
				"rapid_changes":          paymentIndicators.RapidPaymentChanges,
			},
			RiskScore:  paymentIndicators.RiskScore,
			DetectedAt: time.Now(),
		}
		s.repo.CreateFraudAlert(ctx, alert)
		profile.TotalAlerts++
		if alert.AlertLevel == AlertLevelCritical {
			profile.CriticalAlerts++
		}
	}

	if rideIndicators.RiskScore >= 70 {
		alert := &FraudAlert{
			ID:          uuid.New(),
			UserID:      userID,
			AlertType:   AlertTypeRideFraud,
			AlertLevel:  s.determineAlertLevel(rideIndicators.RiskScore),
			Status:      AlertStatusPending,
			Description: "Suspicious ride activity detected",
			Details: map[string]interface{}{
				"excessive_cancellations": rideIndicators.ExcessiveCancellations,
				"unusual_patterns":        rideIndicators.UnusualRidePatterns,
				"promo_abuse":             rideIndicators.PromoAbuse,
			},
			RiskScore:  rideIndicators.RiskScore,
			DetectedAt: time.Now(),
		}
		s.repo.CreateFraudAlert(ctx, alert)
		profile.TotalAlerts++
		if alert.AlertLevel == AlertLevelCritical {
			profile.CriticalAlerts++
		}
	}

	// Auto-suspend account if risk score is critical
	if overallRiskScore >= 90 {
		profile.AccountSuspended = true
	}

	// Update risk profile
	if err := s.repo.UpdateUserRiskProfile(ctx, profile); err != nil {
		return profile, err
	}

	return profile, nil
}

// CreateAlert creates a new fraud alert
func (s *Service) CreateAlert(ctx context.Context, alert *FraudAlert) error {
	if alert.ID == uuid.Nil {
		alert.ID = uuid.New()
	}
	if alert.DetectedAt.IsZero() {
		alert.DetectedAt = time.Now()
	}
	if alert.Status == "" {
		alert.Status = AlertStatusPending
	}

	if err := s.repo.CreateFraudAlert(ctx, alert); err != nil {
		return common.NewInternalServerError("failed to create fraud alert")
	}

	// Update user risk profile
	profile, err := s.repo.GetUserRiskProfile(ctx, alert.UserID)
	if err != nil {
		profile = &UserRiskProfile{
			UserID:      alert.UserID,
			LastUpdated: time.Now(),
		}
	}

	profile.TotalAlerts++
	if alert.AlertLevel == AlertLevelCritical {
		profile.CriticalAlerts++
	}
	profile.RiskScore += alert.RiskScore * 0.1 // Increment risk score
	if profile.RiskScore > 100 {
		profile.RiskScore = 100
	}
	now := time.Now()
	profile.LastAlertAt = &now
	profile.LastUpdated = now

	s.repo.UpdateUserRiskProfile(ctx, profile)

	return nil
}

// GetAlert retrieves a fraud alert by ID
func (s *Service) GetAlert(ctx context.Context, alertID uuid.UUID) (*FraudAlert, error) {
	alert, err := s.repo.GetFraudAlertByID(ctx, alertID)
	if err != nil {
		return nil, common.NewNotFoundError("fraud alert not found", nil)
	}
	return alert, nil
}

// GetUserAlerts retrieves all fraud alerts for a user
func (s *Service) GetUserAlerts(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*FraudAlert, error) {
	offset := (page - 1) * perPage
	alerts, err := s.repo.GetAlertsByUser(ctx, userID, perPage, offset)
	if err != nil {
		return nil, common.NewInternalServerError("failed to get fraud alerts")
	}
	return alerts, nil
}

// GetPendingAlerts retrieves all pending fraud alerts
func (s *Service) GetPendingAlerts(ctx context.Context, page, perPage int) ([]*FraudAlert, error) {
	offset := (page - 1) * perPage
	alerts, err := s.repo.GetPendingAlerts(ctx, perPage, offset)
	if err != nil {
		return nil, common.NewInternalServerError("failed to get pending alerts")
	}
	return alerts, nil
}

// InvestigateAlert marks an alert as under investigation
func (s *Service) InvestigateAlert(ctx context.Context, alertID, investigatorID uuid.UUID, notes string) error {
	if err := s.repo.UpdateAlertStatus(ctx, alertID, AlertStatusInvestigating, &investigatorID, notes, ""); err != nil {
		return common.NewInternalServerError("failed to update alert status")
	}
	return nil
}

// ResolveAlert resolves a fraud alert
func (s *Service) ResolveAlert(ctx context.Context, alertID, investigatorID uuid.UUID, confirmed bool, notes, actionTaken string) error {
	status := AlertStatusResolved
	if confirmed {
		status = AlertStatusConfirmed
	} else {
		status = AlertStatusFalsePositive
	}

	if err := s.repo.UpdateAlertStatus(ctx, alertID, status, &investigatorID, notes, actionTaken); err != nil {
		return common.NewInternalServerError("failed to resolve alert")
	}

	// If confirmed, update user risk profile
	if confirmed {
		alert, err := s.repo.GetFraudAlertByID(ctx, alertID)
		if err == nil {
			profile, err := s.repo.GetUserRiskProfile(ctx, alert.UserID)
			if err == nil {
				profile.ConfirmedFraudCases++
				profile.RiskScore += 20 // Significant increase for confirmed fraud
				if profile.RiskScore > 100 {
					profile.RiskScore = 100
				}
				profile.LastUpdated = time.Now()

				// Auto-suspend if confirmed fraud
				if profile.ConfirmedFraudCases >= 3 {
					profile.AccountSuspended = true
				}

				s.repo.UpdateUserRiskProfile(ctx, profile)
			}
		}
	}

	return nil
}

// GetUserRiskProfile retrieves a user's risk profile
func (s *Service) GetUserRiskProfile(ctx context.Context, userID uuid.UUID) (*UserRiskProfile, error) {
	profile, err := s.repo.GetUserRiskProfile(ctx, userID)
	if err != nil {
		return nil, common.NewInternalServerError("failed to get user risk profile")
	}
	return profile, nil
}

// SuspendUser suspends a user account due to fraud
func (s *Service) SuspendUser(ctx context.Context, userID, adminID uuid.UUID, reason string) error {
	profile, err := s.repo.GetUserRiskProfile(ctx, userID)
	if err != nil {
		profile = &UserRiskProfile{
			UserID:      userID,
			LastUpdated: time.Now(),
		}
	}

	profile.AccountSuspended = true
	profile.LastUpdated = time.Now()

	if err := s.repo.UpdateUserRiskProfile(ctx, profile); err != nil {
		return common.NewInternalServerError("failed to suspend user")
	}

	// Create alert for manual suspension
	alert := &FraudAlert{
		ID:             uuid.New(),
		UserID:         userID,
		AlertType:      AlertTypeAccountFraud,
		AlertLevel:     AlertLevelCritical,
		Status:         AlertStatusConfirmed,
		Description:    "Account manually suspended for fraud",
		Details:        map[string]interface{}{"reason": reason},
		RiskScore:      100,
		DetectedAt:     time.Now(),
		InvestigatedBy: &adminID,
		ActionTaken:    "Account suspended",
		Notes:          reason,
	}
	now := time.Now()
	alert.InvestigatedAt = &now
	alert.ResolvedAt = &now

	return s.repo.CreateFraudAlert(ctx, alert)
}

// ReinstateUser reinstates a suspended user account
func (s *Service) ReinstateUser(ctx context.Context, userID, adminID uuid.UUID, reason string) error {
	profile, err := s.repo.GetUserRiskProfile(ctx, userID)
	if err != nil {
		return common.NewNotFoundError("user risk profile not found", nil)
	}

	profile.AccountSuspended = false
	profile.RiskScore = profile.RiskScore * 0.5 // Reduce risk score by half
	profile.LastUpdated = time.Now()

	if err := s.repo.UpdateUserRiskProfile(ctx, profile); err != nil {
		return common.NewInternalServerError("failed to reinstate user")
	}

	// Create alert for reinstatement
	alert := &FraudAlert{
		ID:             uuid.New(),
		UserID:         userID,
		AlertType:      AlertTypeAccountFraud,
		AlertLevel:     AlertLevelLow,
		Status:         AlertStatusResolved,
		Description:    "Account reinstated",
		Details:        map[string]interface{}{"reason": reason},
		RiskScore:      0,
		DetectedAt:     time.Now(),
		InvestigatedBy: &adminID,
		ActionTaken:    "Account reinstated",
		Notes:          reason,
	}
	now := time.Now()
	alert.InvestigatedAt = &now
	alert.ResolvedAt = &now

	return s.repo.CreateFraudAlert(ctx, alert)
}

// DetectPaymentFraud analyzes payment patterns and creates alerts if needed
func (s *Service) DetectPaymentFraud(ctx context.Context, userID uuid.UUID) error {
	indicators, err := s.repo.GetPaymentFraudIndicators(ctx, userID)
	if err != nil {
		return err
	}

	if indicators.RiskScore >= 70 {
		alert := &FraudAlert{
			ID:          uuid.New(),
			UserID:      userID,
			AlertType:   AlertTypePaymentFraud,
			AlertLevel:  s.determineAlertLevel(indicators.RiskScore),
			Status:      AlertStatusPending,
			Description: fmt.Sprintf("Payment fraud risk score: %.1f", indicators.RiskScore),
			Details: map[string]interface{}{
				"failed_attempts":          indicators.FailedPaymentAttempts,
				"chargebacks":              indicators.ChargebackCount,
				"multiple_payment_methods": indicators.MultiplePaymentMethods,
				"suspicious_transactions":  indicators.SuspiciousTransactions,
				"rapid_changes":            indicators.RapidPaymentChanges,
			},
			RiskScore:  indicators.RiskScore,
			DetectedAt: time.Now(),
		}
		return s.CreateAlert(ctx, alert)
	}

	return nil
}

// DetectRideFraud analyzes ride patterns and creates alerts if needed
func (s *Service) DetectRideFraud(ctx context.Context, userID uuid.UUID) error {
	indicators, err := s.repo.GetRideFraudIndicators(ctx, userID)
	if err != nil {
		return err
	}

	if indicators.RiskScore >= 70 {
		alert := &FraudAlert{
			ID:          uuid.New(),
			UserID:      userID,
			AlertType:   AlertTypeRideFraud,
			AlertLevel:  s.determineAlertLevel(indicators.RiskScore),
			Status:      AlertStatusPending,
			Description: fmt.Sprintf("Ride fraud risk score: %.1f", indicators.RiskScore),
			Details: map[string]interface{}{
				"excessive_cancellations": indicators.ExcessiveCancellations,
				"unusual_patterns":        indicators.UnusualRidePatterns,
				"fake_gps":                indicators.FakeGPSDetected,
				"collision":               indicators.CollisionWithDriver,
				"promo_abuse":             indicators.PromoAbuse,
			},
			RiskScore:  indicators.RiskScore,
			DetectedAt: time.Now(),
		}
		return s.CreateAlert(ctx, alert)
	}

	return nil
}

// Helper function to determine alert level based on risk score
func (s *Service) determineAlertLevel(riskScore float64) FraudAlertLevel {
	if riskScore >= 90 {
		return AlertLevelCritical
	} else if riskScore >= 70 {
		return AlertLevelHigh
	} else if riskScore >= 50 {
		return AlertLevelMedium
	}
	return AlertLevelLow
}
