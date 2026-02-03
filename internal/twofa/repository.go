package twofa

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for 2FA
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new 2FA repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// OTP VERIFICATION OPERATIONS
// ========================================

// CreateOTPVerification creates a new OTP verification record
func (r *Repository) CreateOTPVerification(ctx context.Context, otp *OTPVerification) error {
	query := `
		INSERT INTO otp_verifications (
			id, user_id, otp_hash, otp_type, delivery_method, destination,
			attempts, max_attempts, expires_at, ip_address, user_agent
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at
	`

	return r.db.QueryRow(ctx, query,
		otp.ID,
		otp.UserID,
		otp.OTPHash,
		otp.OTPType,
		otp.DeliveryMethod,
		otp.Destination,
		otp.Attempts,
		otp.MaxAttempts,
		otp.ExpiresAt,
		otp.IPAddress,
		otp.UserAgent,
	).Scan(&otp.CreatedAt)
}

// GetActiveOTP gets the most recent active (not expired, not verified) OTP for a user
func (r *Repository) GetActiveOTP(ctx context.Context, userID uuid.UUID, otpType OTPType) (*OTPVerification, error) {
	query := `
		SELECT id, user_id, otp_hash, otp_type, delivery_method, destination,
			   attempts, max_attempts, expires_at, verified_at, ip_address, user_agent, created_at
		FROM otp_verifications
		WHERE user_id = $1 AND otp_type = $2 AND expires_at > NOW() AND verified_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
	`

	otp := &OTPVerification{}
	err := r.db.QueryRow(ctx, query, userID, otpType).Scan(
		&otp.ID,
		&otp.UserID,
		&otp.OTPHash,
		&otp.OTPType,
		&otp.DeliveryMethod,
		&otp.Destination,
		&otp.Attempts,
		&otp.MaxAttempts,
		&otp.ExpiresAt,
		&otp.VerifiedAt,
		&otp.IPAddress,
		&otp.UserAgent,
		&otp.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get active OTP: %w", err)
	}

	return otp, nil
}

// IncrementOTPAttempts increments the attempts count for an OTP
func (r *Repository) IncrementOTPAttempts(ctx context.Context, otpID uuid.UUID) error {
	query := `UPDATE otp_verifications SET attempts = attempts + 1 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, otpID)
	return err
}

// MarkOTPVerified marks an OTP as verified
func (r *Repository) MarkOTPVerified(ctx context.Context, otpID uuid.UUID) error {
	query := `UPDATE otp_verifications SET verified_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, otpID)
	return err
}

// InvalidateUserOTPs invalidates all active OTPs for a user of a specific type
func (r *Repository) InvalidateUserOTPs(ctx context.Context, userID uuid.UUID, otpType OTPType) error {
	query := `
		UPDATE otp_verifications
		SET expires_at = NOW()
		WHERE user_id = $1 AND otp_type = $2 AND expires_at > NOW() AND verified_at IS NULL
	`
	_, err := r.db.Exec(ctx, query, userID, otpType)
	return err
}

// ========================================
// BACKUP CODE OPERATIONS
// ========================================

// CreateBackupCodes creates backup codes for a user (deletes existing first)
func (r *Repository) CreateBackupCodes(ctx context.Context, userID uuid.UUID, codeHashes []string) error {
	// Delete existing unused codes
	deleteQuery := `DELETE FROM backup_codes WHERE user_id = $1 AND used_at IS NULL`
	_, err := r.db.Exec(ctx, deleteQuery, userID)
	if err != nil {
		return fmt.Errorf("failed to delete existing backup codes: %w", err)
	}

	// Insert new codes
	for _, hash := range codeHashes {
		insertQuery := `INSERT INTO backup_codes (id, user_id, code_hash) VALUES ($1, $2, $3)`
		_, err := r.db.Exec(ctx, insertQuery, uuid.New(), userID, hash)
		if err != nil {
			return fmt.Errorf("failed to create backup code: %w", err)
		}
	}

	return nil
}

// GetUnusedBackupCodes gets all unused backup codes for a user
func (r *Repository) GetUnusedBackupCodes(ctx context.Context, userID uuid.UUID) ([]*BackupCode, error) {
	query := `
		SELECT id, user_id, code_hash, used_at, created_at
		FROM backup_codes
		WHERE user_id = $1 AND used_at IS NULL
		ORDER BY created_at
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup codes: %w", err)
	}
	defer rows.Close()

	var codes []*BackupCode
	for rows.Next() {
		code := &BackupCode{}
		if err := rows.Scan(&code.ID, &code.UserID, &code.CodeHash, &code.UsedAt, &code.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan backup code: %w", err)
		}
		codes = append(codes, code)
	}

	return codes, nil
}

// UseBackupCode marks a backup code as used
func (r *Repository) UseBackupCode(ctx context.Context, codeID uuid.UUID) error {
	query := `UPDATE backup_codes SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`
	result, err := r.db.Exec(ctx, query, codeID)
	if err != nil {
		return fmt.Errorf("failed to use backup code: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("backup code not found or already used")
	}
	return nil
}

// CountUnusedBackupCodes counts unused backup codes for a user
func (r *Repository) CountUnusedBackupCodes(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM backup_codes WHERE user_id = $1 AND used_at IS NULL`
	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

// ========================================
// TRUSTED DEVICE OPERATIONS
// ========================================

// CreateTrustedDevice creates a new trusted device
func (r *Repository) CreateTrustedDevice(ctx context.Context, device *TrustedDevice) error {
	query := `
		INSERT INTO trusted_devices (
			id, user_id, device_token, device_fingerprint, device_name,
			trusted_at, expires_at, ip_address, user_agent
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at
	`

	return r.db.QueryRow(ctx, query,
		device.ID,
		device.UserID,
		device.DeviceToken,
		device.DeviceFingerprint,
		device.DeviceName,
		device.TrustedAt,
		device.ExpiresAt,
		device.IPAddress,
		device.UserAgent,
	).Scan(&device.CreatedAt)
}

// GetTrustedDevice gets a trusted device by token
func (r *Repository) GetTrustedDevice(ctx context.Context, token string) (*TrustedDevice, error) {
	query := `
		SELECT id, user_id, device_token, device_fingerprint, device_name,
			   trusted_at, expires_at, last_used_at, ip_address, user_agent,
			   revoked_at, revoked_reason, created_at
		FROM trusted_devices
		WHERE device_token = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`

	device := &TrustedDevice{}
	err := r.db.QueryRow(ctx, query, token).Scan(
		&device.ID,
		&device.UserID,
		&device.DeviceToken,
		&device.DeviceFingerprint,
		&device.DeviceName,
		&device.TrustedAt,
		&device.ExpiresAt,
		&device.LastUsedAt,
		&device.IPAddress,
		&device.UserAgent,
		&device.RevokedAt,
		&device.RevokedReason,
		&device.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get trusted device: %w", err)
	}

	return device, nil
}

// UpdateTrustedDeviceLastUsed updates the last used timestamp
func (r *Repository) UpdateTrustedDeviceLastUsed(ctx context.Context, deviceID uuid.UUID) error {
	query := `UPDATE trusted_devices SET last_used_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, deviceID)
	return err
}

// GetUserTrustedDevices gets all trusted devices for a user
func (r *Repository) GetUserTrustedDevices(ctx context.Context, userID uuid.UUID) ([]*TrustedDevice, error) {
	query := `
		SELECT id, user_id, device_token, device_fingerprint, device_name,
			   trusted_at, expires_at, last_used_at, ip_address, user_agent,
			   revoked_at, revoked_reason, created_at
		FROM trusted_devices
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY last_used_at DESC NULLS LAST
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trusted devices: %w", err)
	}
	defer rows.Close()

	var devices []*TrustedDevice
	for rows.Next() {
		device := &TrustedDevice{}
		if err := rows.Scan(
			&device.ID, &device.UserID, &device.DeviceToken, &device.DeviceFingerprint,
			&device.DeviceName, &device.TrustedAt, &device.ExpiresAt, &device.LastUsedAt,
			&device.IPAddress, &device.UserAgent, &device.RevokedAt, &device.RevokedReason,
			&device.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan trusted device: %w", err)
		}
		devices = append(devices, device)
	}

	return devices, nil
}

// RevokeTrustedDevice revokes a trusted device
func (r *Repository) RevokeTrustedDevice(ctx context.Context, deviceID uuid.UUID, reason string) error {
	query := `UPDATE trusted_devices SET revoked_at = NOW(), revoked_reason = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, reason, deviceID)
	return err
}

// RevokeAllUserTrustedDevices revokes all trusted devices for a user
func (r *Repository) RevokeAllUserTrustedDevices(ctx context.Context, userID uuid.UUID, reason string) error {
	query := `
		UPDATE trusted_devices
		SET revoked_at = NOW(), revoked_reason = $1
		WHERE user_id = $2 AND revoked_at IS NULL
	`
	_, err := r.db.Exec(ctx, query, reason, userID)
	return err
}

// CountUserTrustedDevices counts active trusted devices for a user
func (r *Repository) CountUserTrustedDevices(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM trusted_devices
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`
	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

// ========================================
// PENDING LOGIN OPERATIONS
// ========================================

// CreatePendingLogin creates a pending 2FA login session
func (r *Repository) CreatePendingLogin(ctx context.Context, pending *PendingLogin) error {
	query := `
		INSERT INTO twofa_pending_logins (id, user_id, session_token, ip_address, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at
	`

	return r.db.QueryRow(ctx, query,
		pending.ID,
		pending.UserID,
		pending.SessionToken,
		pending.IPAddress,
		pending.UserAgent,
		pending.ExpiresAt,
	).Scan(&pending.CreatedAt)
}

// GetPendingLogin gets a pending login by session token
func (r *Repository) GetPendingLogin(ctx context.Context, sessionToken string) (*PendingLogin, error) {
	query := `
		SELECT id, user_id, session_token, ip_address, user_agent, expires_at, completed_at, created_at
		FROM twofa_pending_logins
		WHERE session_token = $1 AND expires_at > NOW() AND completed_at IS NULL
	`

	pending := &PendingLogin{}
	err := r.db.QueryRow(ctx, query, sessionToken).Scan(
		&pending.ID,
		&pending.UserID,
		&pending.SessionToken,
		&pending.IPAddress,
		&pending.UserAgent,
		&pending.ExpiresAt,
		&pending.CompletedAt,
		&pending.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get pending login: %w", err)
	}

	return pending, nil
}

// CompletePendingLogin marks a pending login as completed
func (r *Repository) CompletePendingLogin(ctx context.Context, sessionToken string) error {
	query := `UPDATE twofa_pending_logins SET completed_at = NOW() WHERE session_token = $1`
	_, err := r.db.Exec(ctx, query, sessionToken)
	return err
}

// ========================================
// USER 2FA SETTINGS
// ========================================

// Get2FASettings gets 2FA settings for a user
func (r *Repository) Get2FASettings(ctx context.Context, userID uuid.UUID) (enabled bool, method TwoFAMethod, phoneVerifiedAt, totpVerifiedAt, enabledAt *time.Time, err error) {
	query := `
		SELECT COALESCE(twofa_enabled, false), COALESCE(twofa_method, 'sms'),
			   phone_verified_at, totp_verified_at, twofa_enabled_at
		FROM users
		WHERE id = $1
	`

	var methodStr string
	err = r.db.QueryRow(ctx, query, userID).Scan(&enabled, &methodStr, &phoneVerifiedAt, &totpVerifiedAt, &enabledAt)
	method = TwoFAMethod(methodStr)
	return
}

// Enable2FA enables 2FA for a user
func (r *Repository) Enable2FA(ctx context.Context, userID uuid.UUID, method TwoFAMethod, totpSecret string) error {
	query := `
		UPDATE users
		SET twofa_enabled = true, twofa_method = $1, totp_secret = $2, twofa_enabled_at = NOW(), updated_at = NOW()
		WHERE id = $3
	`
	_, err := r.db.Exec(ctx, query, method, totpSecret, userID)
	return err
}

// Disable2FA disables 2FA for a user
func (r *Repository) Disable2FA(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET twofa_enabled = false, totp_secret = NULL, totp_verified_at = NULL, twofa_enabled_at = NULL, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

// SetPhoneVerified marks phone as verified for a user
func (r *Repository) SetPhoneVerified(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE users SET phone_verified_at = NOW(), is_verified = true, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

// SetTOTPVerified marks TOTP as verified for a user
func (r *Repository) SetTOTPVerified(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE users SET totp_verified_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

// GetTOTPSecret gets the TOTP secret for a user
func (r *Repository) GetTOTPSecret(ctx context.Context, userID uuid.UUID) (string, error) {
	query := `SELECT COALESCE(totp_secret, '') FROM users WHERE id = $1`
	var secret string
	err := r.db.QueryRow(ctx, query, userID).Scan(&secret)
	return secret, err
}

// ========================================
// AUDIT LOG OPERATIONS
// ========================================

// CreateAuditLog creates an audit log entry
func (r *Repository) CreateAuditLog(ctx context.Context, log *AuthAuditLog) error {
	detailsJSON, _ := json.Marshal(log.EventDetails)

	query := `
		INSERT INTO auth_audit_log (
			id, user_id, event_type, event_status, event_details,
			ip_address, user_agent, device_fingerprint, related_entity_type, related_entity_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at
	`

	return r.db.QueryRow(ctx, query,
		log.ID,
		log.UserID,
		log.EventType,
		log.EventStatus,
		detailsJSON,
		log.IPAddress,
		log.UserAgent,
		log.DeviceFingerprint,
		log.RelatedEntityType,
		log.RelatedEntityID,
	).Scan(&log.CreatedAt)
}

// GetUserAuditLogs gets recent audit logs for a user
func (r *Repository) GetUserAuditLogs(ctx context.Context, userID uuid.UUID, limit int) ([]*AuthAuditLog, error) {
	query := `
		SELECT id, user_id, event_type, event_status, event_details,
			   ip_address, user_agent, device_fingerprint, related_entity_type, related_entity_id, created_at
		FROM auth_audit_log
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*AuthAuditLog
	for rows.Next() {
		log := &AuthAuditLog{}
		var detailsJSON []byte
		if err := rows.Scan(
			&log.ID, &log.UserID, &log.EventType, &log.EventStatus, &detailsJSON,
			&log.IPAddress, &log.UserAgent, &log.DeviceFingerprint,
			&log.RelatedEntityType, &log.RelatedEntityID, &log.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &log.EventDetails)
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// ========================================
// RATE LIMITING
// ========================================

// CheckRateLimit checks if an identifier is rate limited (uses database function)
func (r *Repository) CheckRateLimit(ctx context.Context, identifier, identifierType string, maxRequests, windowSeconds int) (bool, error) {
	query := `SELECT check_otp_rate_limit($1, $2, $3, $4)`
	var allowed bool
	err := r.db.QueryRow(ctx, query, identifier, identifierType, maxRequests, windowSeconds).Scan(&allowed)
	return allowed, err
}

// GetRateLimitInfo gets rate limit info for an identifier
func (r *Repository) GetRateLimitInfo(ctx context.Context, identifier, identifierType string) (*OTPRateLimit, error) {
	query := `
		SELECT id, identifier, identifier_type, request_count, window_start,
			   window_duration_seconds, max_requests, locked_until, lock_reason, created_at, updated_at
		FROM otp_rate_limits
		WHERE identifier = $1 AND identifier_type = $2
	`

	limit := &OTPRateLimit{}
	err := r.db.QueryRow(ctx, query, identifier, identifierType).Scan(
		&limit.ID, &limit.Identifier, &limit.IdentifierType, &limit.RequestCount,
		&limit.WindowStart, &limit.WindowDurationSeconds, &limit.MaxRequests,
		&limit.LockedUntil, &limit.LockReason, &limit.CreatedAt, &limit.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return limit, nil
}
