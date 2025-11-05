package fraud

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles fraud detection data operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new fraud repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateFraudAlert creates a new fraud alert
func (r *Repository) CreateFraudAlert(ctx context.Context, alert *FraudAlert) error {
	detailsJSON, err := json.Marshal(alert.Details)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO fraud_alerts (
			id, user_id, alert_type, alert_level, status, description,
			details, risk_score, detected_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = r.db.Exec(ctx, query,
		alert.ID,
		alert.UserID,
		alert.AlertType,
		alert.AlertLevel,
		alert.Status,
		alert.Description,
		detailsJSON,
		alert.RiskScore,
		alert.DetectedAt,
	)

	return err
}

// GetFraudAlertByID retrieves a fraud alert by ID
func (r *Repository) GetFraudAlertByID(ctx context.Context, alertID uuid.UUID) (*FraudAlert, error) {
	query := `
		SELECT id, user_id, alert_type, alert_level, status, description,
		       details, risk_score, detected_at, investigated_at, investigated_by,
		       resolved_at, notes, action_taken
		FROM fraud_alerts
		WHERE id = $1
	`

	var alert FraudAlert
	var detailsJSON []byte
	var investigatedAt, resolvedAt sql.NullTime
	var investigatedBy sql.NullString
	var notes, actionTaken sql.NullString

	err := r.db.QueryRow(ctx, query, alertID).Scan(
		&alert.ID,
		&alert.UserID,
		&alert.AlertType,
		&alert.AlertLevel,
		&alert.Status,
		&alert.Description,
		&detailsJSON,
		&alert.RiskScore,
		&alert.DetectedAt,
		&investigatedAt,
		&investigatedBy,
		&resolvedAt,
		&notes,
		&actionTaken,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(detailsJSON, &alert.Details); err != nil {
		alert.Details = make(map[string]interface{})
	}

	if investigatedAt.Valid {
		alert.InvestigatedAt = &investigatedAt.Time
	}
	if investigatedBy.Valid {
		investigatedByUUID, _ := uuid.Parse(investigatedBy.String)
		alert.InvestigatedBy = &investigatedByUUID
	}
	if resolvedAt.Valid {
		alert.ResolvedAt = &resolvedAt.Time
	}
	if notes.Valid {
		alert.Notes = notes.String
	}
	if actionTaken.Valid {
		alert.ActionTaken = actionTaken.String
	}

	return &alert, nil
}

// GetAlertsByUser retrieves all fraud alerts for a user
func (r *Repository) GetAlertsByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*FraudAlert, error) {
	query := `
		SELECT id, user_id, alert_type, alert_level, status, description,
		       details, risk_score, detected_at, investigated_at, investigated_by,
		       resolved_at, notes, action_taken
		FROM fraud_alerts
		WHERE user_id = $1
		ORDER BY detected_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*FraudAlert
	for rows.Next() {
		var alert FraudAlert
		var detailsJSON []byte
		var investigatedAt, resolvedAt sql.NullTime
		var investigatedBy sql.NullString
		var notes, actionTaken sql.NullString

		err := rows.Scan(
			&alert.ID,
			&alert.UserID,
			&alert.AlertType,
			&alert.AlertLevel,
			&alert.Status,
			&alert.Description,
			&detailsJSON,
			&alert.RiskScore,
			&alert.DetectedAt,
			&investigatedAt,
			&investigatedBy,
			&resolvedAt,
			&notes,
			&actionTaken,
		)

		if err != nil {
			continue
		}

		if err := json.Unmarshal(detailsJSON, &alert.Details); err != nil {
			alert.Details = make(map[string]interface{})
		}

		if investigatedAt.Valid {
			alert.InvestigatedAt = &investigatedAt.Time
		}
		if investigatedBy.Valid {
			investigatedByUUID, _ := uuid.Parse(investigatedBy.String)
			alert.InvestigatedBy = &investigatedByUUID
		}
		if resolvedAt.Valid {
			alert.ResolvedAt = &resolvedAt.Time
		}
		if notes.Valid {
			alert.Notes = notes.String
		}
		if actionTaken.Valid {
			alert.ActionTaken = actionTaken.String
		}

		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

// GetPendingAlerts retrieves all pending fraud alerts
func (r *Repository) GetPendingAlerts(ctx context.Context, limit, offset int) ([]*FraudAlert, error) {
	query := `
		SELECT id, user_id, alert_type, alert_level, status, description,
		       details, risk_score, detected_at, investigated_at, investigated_by,
		       resolved_at, notes, action_taken
		FROM fraud_alerts
		WHERE status IN ('pending', 'investigating')
		ORDER BY alert_level DESC, detected_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*FraudAlert
	for rows.Next() {
		var alert FraudAlert
		var detailsJSON []byte
		var investigatedAt, resolvedAt sql.NullTime
		var investigatedBy sql.NullString
		var notes, actionTaken sql.NullString

		err := rows.Scan(
			&alert.ID,
			&alert.UserID,
			&alert.AlertType,
			&alert.AlertLevel,
			&alert.Status,
			&alert.Description,
			&detailsJSON,
			&alert.RiskScore,
			&alert.DetectedAt,
			&investigatedAt,
			&investigatedBy,
			&resolvedAt,
			&notes,
			&actionTaken,
		)

		if err != nil {
			continue
		}

		if err := json.Unmarshal(detailsJSON, &alert.Details); err != nil {
			alert.Details = make(map[string]interface{})
		}

		if investigatedAt.Valid {
			alert.InvestigatedAt = &investigatedAt.Time
		}
		if investigatedBy.Valid {
			investigatedByUUID, _ := uuid.Parse(investigatedBy.String)
			alert.InvestigatedBy = &investigatedByUUID
		}
		if resolvedAt.Valid {
			alert.ResolvedAt = &resolvedAt.Time
		}
		if notes.Valid {
			alert.Notes = notes.String
		}
		if actionTaken.Valid {
			alert.ActionTaken = actionTaken.String
		}

		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

// UpdateAlertStatus updates the status of a fraud alert
func (r *Repository) UpdateAlertStatus(ctx context.Context, alertID uuid.UUID, status FraudAlertStatus, investigatedBy *uuid.UUID, notes, actionTaken string) error {
	query := `
		UPDATE fraud_alerts
		SET status = $2,
		    investigated_at = CASE WHEN $3::uuid IS NOT NULL THEN NOW() ELSE investigated_at END,
		    investigated_by = COALESCE($3, investigated_by),
		    resolved_at = CASE WHEN $2 IN ('confirmed', 'false_positive', 'resolved') THEN NOW() ELSE resolved_at END,
		    notes = COALESCE(NULLIF($4, ''), notes),
		    action_taken = COALESCE(NULLIF($5, ''), action_taken),
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, alertID, status, investigatedBy, notes, actionTaken)
	return err
}

// GetUserRiskProfile retrieves or creates a risk profile for a user
func (r *Repository) GetUserRiskProfile(ctx context.Context, userID uuid.UUID) (*UserRiskProfile, error) {
	query := `
		SELECT user_id, risk_score, total_alerts, critical_alerts,
		       confirmed_fraud_cases, last_alert_at, account_suspended, last_updated
		FROM user_risk_profiles
		WHERE user_id = $1
	`

	var profile UserRiskProfile
	var lastAlertAt sql.NullTime

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&profile.UserID,
		&profile.RiskScore,
		&profile.TotalAlerts,
		&profile.CriticalAlerts,
		&profile.ConfirmedFraudCases,
		&lastAlertAt,
		&profile.AccountSuspended,
		&profile.LastUpdated,
	)

	if err != nil {
		// If profile doesn't exist, return a new one
		if err == sql.ErrNoRows || err.Error() == "no rows in result set" {
			return &UserRiskProfile{
				UserID:              userID,
				RiskScore:           0,
				TotalAlerts:         0,
				CriticalAlerts:      0,
				ConfirmedFraudCases: 0,
				AccountSuspended:    false,
				LastUpdated:         time.Now(),
			}, nil
		}
		return nil, err
	}

	if lastAlertAt.Valid {
		profile.LastAlertAt = &lastAlertAt.Time
	}

	return &profile, nil
}

// UpdateUserRiskProfile updates a user's risk profile
func (r *Repository) UpdateUserRiskProfile(ctx context.Context, profile *UserRiskProfile) error {
	query := `
		INSERT INTO user_risk_profiles (
			user_id, risk_score, total_alerts, critical_alerts,
			confirmed_fraud_cases, last_alert_at, account_suspended, last_updated
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
			risk_score = EXCLUDED.risk_score,
			total_alerts = EXCLUDED.total_alerts,
			critical_alerts = EXCLUDED.critical_alerts,
			confirmed_fraud_cases = EXCLUDED.confirmed_fraud_cases,
			last_alert_at = EXCLUDED.last_alert_at,
			account_suspended = EXCLUDED.account_suspended,
			last_updated = EXCLUDED.last_updated
	`

	_, err := r.db.Exec(ctx, query,
		profile.UserID,
		profile.RiskScore,
		profile.TotalAlerts,
		profile.CriticalAlerts,
		profile.ConfirmedFraudCases,
		profile.LastAlertAt,
		profile.AccountSuspended,
		profile.LastUpdated,
	)

	return err
}

// GetPaymentFraudIndicators analyzes payment patterns for a user
func (r *Repository) GetPaymentFraudIndicators(ctx context.Context, userID uuid.UUID) (*PaymentFraudIndicators, error) {
	query := `
		SELECT
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_attempts,
			COUNT(CASE WHEN status = 'refunded' THEN 1 END) as chargebacks,
			COUNT(DISTINCT payment_method_id) as payment_methods,
			COUNT(CASE WHEN amount > 1000 THEN 1 END) as suspicious_transactions
		FROM payments
		WHERE user_id = $1
		  AND created_at >= NOW() - INTERVAL '30 days'
	`

	indicators := &PaymentFraudIndicators{UserID: userID}

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&indicators.FailedPaymentAttempts,
		&indicators.ChargebackCount,
		&indicators.MultiplePaymentMethods,
		&indicators.SuspiciousTransactions,
	)

	if err != nil {
		return nil, err
	}

	// Check for rapid payment method changes (>3 changes in 7 days)
	changeQuery := `
		SELECT COUNT(DISTINCT payment_method_id) as changes
		FROM payments
		WHERE user_id = $1
		  AND created_at >= NOW() - INTERVAL '7 days'
	`
	var changes int
	r.db.QueryRow(ctx, changeQuery, userID).Scan(&changes)
	indicators.RapidPaymentChanges = changes > 3

	// Calculate risk score
	indicators.RiskScore = calculatePaymentRiskScore(indicators)

	return indicators, nil
}

// GetRideFraudIndicators analyzes ride patterns for a user
func (r *Repository) GetRideFraudIndicators(ctx context.Context, userID uuid.UUID) (*RideFraudIndicators, error) {
	query := `
		SELECT
			COUNT(CASE WHEN status = 'cancelled' THEN 1 END) as cancellations,
			COUNT(*) as total_rides
		FROM rides
		WHERE (rider_id = $1 OR driver_id = $1)
		  AND requested_at >= NOW() - INTERVAL '30 days'
	`

	indicators := &RideFraudIndicators{UserID: userID}
	var totalRides int

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&indicators.ExcessiveCancellations,
		&totalRides,
	)

	if err != nil {
		return nil, err
	}

	// Check for promo code abuse (>5 promo codes used in 30 days)
	promoQuery := `
		SELECT COUNT(DISTINCT promo_code_id)
		FROM rides
		WHERE rider_id = $1
		  AND promo_code_id IS NOT NULL
		  AND requested_at >= NOW() - INTERVAL '30 days'
	`
	var promoCount int
	r.db.QueryRow(ctx, promoQuery, userID).Scan(&promoCount)
	indicators.PromoAbuse = promoCount > 5

	// Detect unusual patterns
	if totalRides > 0 {
		cancellationRate := float64(indicators.ExcessiveCancellations) / float64(totalRides)
		indicators.UnusualRidePatterns = cancellationRate > 0.5 || totalRides > 100
	}

	// Calculate risk score
	indicators.RiskScore = calculateRideRiskScore(indicators)

	return indicators, nil
}

// Helper functions to calculate risk scores
func calculatePaymentRiskScore(indicators *PaymentFraudIndicators) float64 {
	score := 0.0

	// Failed attempts (0-30 points)
	score += float64(indicators.FailedPaymentAttempts) * 3
	if score > 30 {
		score = 30
	}

	// Chargebacks (0-30 points)
	score += float64(indicators.ChargebackCount) * 15
	if score > 60 {
		score = 60
	}

	// Multiple payment methods (0-20 points)
	if indicators.MultiplePaymentMethods > 5 {
		score += 20
	} else if indicators.MultiplePaymentMethods > 3 {
		score += 10
	}

	// Suspicious transactions (0-20 points)
	score += float64(indicators.SuspiciousTransactions) * 5
	if score > 100 {
		score = 100
	}

	// Rapid changes (10 points)
	if indicators.RapidPaymentChanges {
		score += 10
	}

	if score > 100 {
		score = 100
	}

	return score
}

func calculateRideRiskScore(indicators *RideFraudIndicators) float64 {
	score := 0.0

	// Excessive cancellations (0-40 points)
	score += float64(indicators.ExcessiveCancellations) * 2
	if score > 40 {
		score = 40
	}

	// Unusual patterns (20 points)
	if indicators.UnusualRidePatterns {
		score += 20
	}

	// Fake GPS (30 points)
	if indicators.FakeGPSDetected {
		score += 30
	}

	// Collision (30 points)
	if indicators.CollisionWithDriver {
		score += 30
	}

	// Promo abuse (20 points)
	if indicators.PromoAbuse {
		score += 20
	}

	if score > 100 {
		score = 100
	}

	return score
}
