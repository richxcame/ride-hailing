package verification

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for verification
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new verification repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// BACKGROUND CHECK OPERATIONS
// ========================================

// CreateBackgroundCheck creates a new background check record
func (r *Repository) CreateBackgroundCheck(ctx context.Context, check *BackgroundCheck) error {
	query := `
		INSERT INTO driver_background_checks (
			id, driver_id, provider, external_id, status, check_type,
			report_url, notes, failure_reasons, expires_at, started_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
	`

	_, err := r.db.Exec(ctx, query,
		check.ID, check.DriverID, check.Provider, check.ExternalID,
		check.Status, check.CheckType, check.ReportURL, check.Notes,
		check.FailureReasons, check.ExpiresAt, check.StartedAt,
	)

	return err
}

// GetBackgroundCheck gets a background check by ID
func (r *Repository) GetBackgroundCheck(ctx context.Context, checkID uuid.UUID) (*BackgroundCheck, error) {
	query := `
		SELECT id, driver_id, provider, external_id, status, check_type,
		       report_url, notes, failure_reasons, expires_at, started_at, completed_at, created_at, updated_at
		FROM driver_background_checks
		WHERE id = $1
	`

	check := &BackgroundCheck{}
	err := r.db.QueryRow(ctx, query, checkID).Scan(
		&check.ID, &check.DriverID, &check.Provider, &check.ExternalID,
		&check.Status, &check.CheckType, &check.ReportURL, &check.Notes,
		&check.FailureReasons, &check.ExpiresAt, &check.StartedAt,
		&check.CompletedAt, &check.CreatedAt, &check.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return check, nil
}

// GetBackgroundCheckByExternalID gets a background check by external provider ID
func (r *Repository) GetBackgroundCheckByExternalID(ctx context.Context, provider BackgroundCheckProvider, externalID string) (*BackgroundCheck, error) {
	query := `
		SELECT id, driver_id, provider, external_id, status, check_type,
		       report_url, notes, failure_reasons, expires_at, started_at, completed_at, created_at, updated_at
		FROM driver_background_checks
		WHERE provider = $1 AND external_id = $2
	`

	check := &BackgroundCheck{}
	err := r.db.QueryRow(ctx, query, provider, externalID).Scan(
		&check.ID, &check.DriverID, &check.Provider, &check.ExternalID,
		&check.Status, &check.CheckType, &check.ReportURL, &check.Notes,
		&check.FailureReasons, &check.ExpiresAt, &check.StartedAt,
		&check.CompletedAt, &check.CreatedAt, &check.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return check, nil
}

// GetLatestBackgroundCheck gets the most recent background check for a driver
func (r *Repository) GetLatestBackgroundCheck(ctx context.Context, driverID uuid.UUID) (*BackgroundCheck, error) {
	query := `
		SELECT id, driver_id, provider, external_id, status, check_type,
		       report_url, notes, failure_reasons, expires_at, started_at, completed_at, created_at, updated_at
		FROM driver_background_checks
		WHERE driver_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	check := &BackgroundCheck{}
	err := r.db.QueryRow(ctx, query, driverID).Scan(
		&check.ID, &check.DriverID, &check.Provider, &check.ExternalID,
		&check.Status, &check.CheckType, &check.ReportURL, &check.Notes,
		&check.FailureReasons, &check.ExpiresAt, &check.StartedAt,
		&check.CompletedAt, &check.CreatedAt, &check.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return check, nil
}

// UpdateBackgroundCheckStatus updates the status of a background check
func (r *Repository) UpdateBackgroundCheckStatus(ctx context.Context, checkID uuid.UUID, status BackgroundCheckStatus, notes *string, failureReasons []string) error {
	var completedAt *time.Time
	if status == BGCheckStatusPassed || status == BGCheckStatusFailed || status == BGCheckStatusCompleted {
		now := time.Now()
		completedAt = &now
	}

	query := `
		UPDATE driver_background_checks
		SET status = $1, notes = COALESCE($2, notes), failure_reasons = COALESCE($3, failure_reasons),
		    completed_at = COALESCE($4, completed_at), updated_at = NOW()
		WHERE id = $5
	`

	_, err := r.db.Exec(ctx, query, status, notes, failureReasons, completedAt, checkID)
	return err
}

// UpdateBackgroundCheckStarted marks a check as started
func (r *Repository) UpdateBackgroundCheckStarted(ctx context.Context, checkID uuid.UUID, externalID string) error {
	query := `
		UPDATE driver_background_checks
		SET status = 'in_progress', external_id = $1, started_at = NOW(), updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, externalID, checkID)
	return err
}

// UpdateBackgroundCheckCompleted marks a check as completed
func (r *Repository) UpdateBackgroundCheckCompleted(ctx context.Context, checkID uuid.UUID, status BackgroundCheckStatus, reportURL *string, expiresAt *time.Time) error {
	query := `
		UPDATE driver_background_checks
		SET status = $1, report_url = $2, expires_at = $3, completed_at = NOW(), updated_at = NOW()
		WHERE id = $4
	`

	_, err := r.db.Exec(ctx, query, status, reportURL, expiresAt, checkID)
	return err
}

// GetPendingBackgroundChecks gets all pending background checks
func (r *Repository) GetPendingBackgroundChecks(ctx context.Context, limit int) ([]*BackgroundCheck, error) {
	query := `
		SELECT id, driver_id, provider, external_id, status, check_type,
		       report_url, notes, failure_reasons, expires_at, started_at, completed_at, created_at, updated_at
		FROM driver_background_checks
		WHERE status IN ('pending', 'in_progress')
		ORDER BY created_at ASC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []*BackgroundCheck
	for rows.Next() {
		check := &BackgroundCheck{}
		err := rows.Scan(
			&check.ID, &check.DriverID, &check.Provider, &check.ExternalID,
			&check.Status, &check.CheckType, &check.ReportURL, &check.Notes,
			&check.FailureReasons, &check.ExpiresAt, &check.StartedAt,
			&check.CompletedAt, &check.CreatedAt, &check.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		checks = append(checks, check)
	}

	return checks, nil
}

// ========================================
// SELFIE VERIFICATION OPERATIONS
// ========================================

// CreateSelfieVerification creates a new selfie verification record
func (r *Repository) CreateSelfieVerification(ctx context.Context, verification *SelfieVerification) error {
	query := `
		INSERT INTO driver_selfie_verifications (
			id, driver_id, ride_id, selfie_url, reference_photo_url, status,
			confidence_score, match_result, provider, external_id, failure_reason, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
	`

	_, err := r.db.Exec(ctx, query,
		verification.ID, verification.DriverID, verification.RideID,
		verification.SelfieURL, verification.ReferencePhotoURL, verification.Status,
		verification.ConfidenceScore, verification.MatchResult, verification.Provider,
		verification.ExternalID, verification.FailureReason,
	)

	return err
}

// GetSelfieVerification gets a selfie verification by ID
func (r *Repository) GetSelfieVerification(ctx context.Context, verificationID uuid.UUID) (*SelfieVerification, error) {
	query := `
		SELECT id, driver_id, ride_id, selfie_url, reference_photo_url, status,
		       confidence_score, match_result, provider, external_id, failure_reason,
		       verified_at, expires_at, created_at
		FROM driver_selfie_verifications
		WHERE id = $1
	`

	v := &SelfieVerification{}
	err := r.db.QueryRow(ctx, query, verificationID).Scan(
		&v.ID, &v.DriverID, &v.RideID, &v.SelfieURL, &v.ReferencePhotoURL,
		&v.Status, &v.ConfidenceScore, &v.MatchResult, &v.Provider,
		&v.ExternalID, &v.FailureReason, &v.VerifiedAt, &v.ExpiresAt, &v.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return v, nil
}

// GetLatestSelfieVerification gets the most recent selfie verification for a driver
func (r *Repository) GetLatestSelfieVerification(ctx context.Context, driverID uuid.UUID) (*SelfieVerification, error) {
	query := `
		SELECT id, driver_id, ride_id, selfie_url, reference_photo_url, status,
		       confidence_score, match_result, provider, external_id, failure_reason,
		       verified_at, expires_at, created_at
		FROM driver_selfie_verifications
		WHERE driver_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	v := &SelfieVerification{}
	err := r.db.QueryRow(ctx, query, driverID).Scan(
		&v.ID, &v.DriverID, &v.RideID, &v.SelfieURL, &v.ReferencePhotoURL,
		&v.Status, &v.ConfidenceScore, &v.MatchResult, &v.Provider,
		&v.ExternalID, &v.FailureReason, &v.VerifiedAt, &v.ExpiresAt, &v.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return v, nil
}

// GetTodaysSelfieVerification gets today's selfie verification for a driver
func (r *Repository) GetTodaysSelfieVerification(ctx context.Context, driverID uuid.UUID) (*SelfieVerification, error) {
	query := `
		SELECT id, driver_id, ride_id, selfie_url, reference_photo_url, status,
		       confidence_score, match_result, provider, external_id, failure_reason,
		       verified_at, expires_at, created_at
		FROM driver_selfie_verifications
		WHERE driver_id = $1
		  AND created_at >= CURRENT_DATE
		  AND status = 'verified'
		ORDER BY created_at DESC
		LIMIT 1
	`

	v := &SelfieVerification{}
	err := r.db.QueryRow(ctx, query, driverID).Scan(
		&v.ID, &v.DriverID, &v.RideID, &v.SelfieURL, &v.ReferencePhotoURL,
		&v.Status, &v.ConfidenceScore, &v.MatchResult, &v.Provider,
		&v.ExternalID, &v.FailureReason, &v.VerifiedAt, &v.ExpiresAt, &v.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return v, nil
}

// UpdateSelfieVerificationResult updates the verification result
func (r *Repository) UpdateSelfieVerificationResult(ctx context.Context, verificationID uuid.UUID, status SelfieVerificationStatus, confidenceScore *float64, matchResult *bool, failureReason *string) error {
	var verifiedAt *time.Time
	if status == SelfieStatusVerified {
		now := time.Now()
		verifiedAt = &now
	}

	query := `
		UPDATE driver_selfie_verifications
		SET status = $1, confidence_score = $2, match_result = $3, failure_reason = $4, verified_at = $5
		WHERE id = $6
	`

	_, err := r.db.Exec(ctx, query, status, confidenceScore, matchResult, failureReason, verifiedAt, verificationID)
	return err
}

// GetDriverReferencePhoto gets the driver's reference photo URL
func (r *Repository) GetDriverReferencePhoto(ctx context.Context, driverID uuid.UUID) (*string, error) {
	query := `
		SELECT photo_url FROM driver_documents
		WHERE driver_id = $1 AND document_type = 'selfie' AND status = 'verified'
		ORDER BY created_at DESC
		LIMIT 1
	`

	var photoURL *string
	err := r.db.QueryRow(ctx, query, driverID).Scan(&photoURL)
	if err != nil {
		return nil, err
	}

	return photoURL, nil
}

// ========================================
// DRIVER VERIFICATION STATUS
// ========================================

// GetDriverVerificationStatus gets the overall verification status for a driver
func (r *Repository) GetDriverVerificationStatus(ctx context.Context, driverID uuid.UUID) (*DriverVerificationStatus, error) {
	status := &DriverVerificationStatus{
		DriverID:                   driverID,
		BackgroundCheckStatus:      BGCheckStatusPending,
		SelfieVerificationRequired: true,
		IsFullyVerified:            false,
	}

	// Get latest background check
	bgCheck, err := r.GetLatestBackgroundCheck(ctx, driverID)
	if err == nil && bgCheck != nil {
		status.BackgroundCheckStatus = bgCheck.Status
		status.BackgroundCheckExpiresAt = bgCheck.ExpiresAt
	}

	// Get today's selfie verification
	selfie, err := r.GetTodaysSelfieVerification(ctx, driverID)
	if err == nil && selfie != nil {
		status.LastSelfieVerification = selfie
		status.SelfieVerificationRequired = false
	}

	// Determine if fully verified
	if status.BackgroundCheckStatus == BGCheckStatusPassed && !status.SelfieVerificationRequired {
		status.IsFullyVerified = true
		status.VerificationMessage = "Fully verified and ready to accept rides"
	} else if status.BackgroundCheckStatus != BGCheckStatusPassed {
		status.VerificationMessage = "Background check not passed"
	} else if status.SelfieVerificationRequired {
		status.VerificationMessage = "Daily selfie verification required"
	}

	return status, nil
}

// UpdateDriverApproval updates the driver approval status based on verification
func (r *Repository) UpdateDriverApproval(ctx context.Context, driverID uuid.UUID, approved bool, approvedBy *uuid.UUID, reason *string) error {
	if approved {
		query := `
			UPDATE drivers
			SET is_approved = true, approved_at = NOW(), approved_by = $1, updated_at = NOW()
			WHERE id = $2
		`
		_, err := r.db.Exec(ctx, query, approvedBy, driverID)
		return err
	}

	query := `
		UPDATE drivers
		SET is_approved = false, rejection_reason = $1, rejected_at = NOW(), rejected_by = $2, updated_at = NOW()
		WHERE id = $3
	`
	_, err := r.db.Exec(ctx, query, reason, approvedBy, driverID)
	return err
}
