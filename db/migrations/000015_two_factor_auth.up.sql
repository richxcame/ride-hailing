-- Two-Factor Authentication Migration
-- Adds support for SMS-based OTP and TOTP (authenticator app) based 2FA

-- ========================================
-- USER 2FA SETTINGS
-- ========================================

-- Add 2FA fields to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS twofa_enabled BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS twofa_method VARCHAR(20) DEFAULT 'sms'; -- 'sms', 'totp', 'both'
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone_verified_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_secret TEXT; -- Encrypted TOTP secret for authenticator apps
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_verified_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS twofa_enabled_at TIMESTAMPTZ;

-- Index for 2FA queries
CREATE INDEX IF NOT EXISTS idx_users_twofa ON users(twofa_enabled) WHERE twofa_enabled = true;

-- ========================================
-- OTP VERIFICATION RECORDS
-- ========================================

CREATE TABLE IF NOT EXISTS otp_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- OTP details
    otp_hash VARCHAR(255) NOT NULL, -- Bcrypt hashed OTP for security
    otp_type VARCHAR(50) NOT NULL, -- 'login', 'phone_verification', 'enable_2fa', 'disable_2fa', 'password_reset'

    -- Delivery info
    delivery_method VARCHAR(20) NOT NULL DEFAULT 'sms', -- 'sms', 'email'
    destination VARCHAR(255) NOT NULL, -- Phone number or email

    -- Tracking
    attempts INT DEFAULT 0,
    max_attempts INT DEFAULT 5,
    expires_at TIMESTAMPTZ NOT NULL,
    verified_at TIMESTAMPTZ,

    -- Metadata
    ip_address INET,
    user_agent TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_otp_user_active ON otp_verifications(user_id, otp_type, expires_at)
    WHERE verified_at IS NULL;
CREATE INDEX idx_otp_expires ON otp_verifications(expires_at)
    WHERE verified_at IS NULL;

-- ========================================
-- BACKUP CODES
-- ========================================

CREATE TABLE IF NOT EXISTS backup_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash VARCHAR(255) NOT NULL, -- Bcrypt hashed backup code
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_backup_codes_user ON backup_codes(user_id);
CREATE INDEX idx_backup_codes_unused ON backup_codes(user_id) WHERE used_at IS NULL;

-- ========================================
-- TRUSTED DEVICES
-- ========================================

CREATE TABLE IF NOT EXISTS trusted_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Device identification
    device_token VARCHAR(255) NOT NULL UNIQUE, -- Secure random token stored in cookie
    device_fingerprint VARCHAR(255), -- Browser/device fingerprint
    device_name VARCHAR(255), -- User-friendly name (e.g., "Chrome on MacOS")

    -- Trust details
    trusted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL, -- Usually 30 days
    last_used_at TIMESTAMPTZ,

    -- Security metadata
    ip_address INET,
    user_agent TEXT,

    -- Revocation
    revoked_at TIMESTAMPTZ,
    revoked_reason VARCHAR(255),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trusted_devices_user ON trusted_devices(user_id);
CREATE INDEX idx_trusted_devices_token ON trusted_devices(device_token)
    WHERE revoked_at IS NULL;
CREATE INDEX idx_trusted_devices_expires ON trusted_devices(expires_at)
    WHERE revoked_at IS NULL;

-- ========================================
-- 2FA LOGIN SESSIONS (Pending 2FA verification)
-- ========================================

CREATE TABLE IF NOT EXISTS twofa_pending_logins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Session token for 2FA flow
    session_token VARCHAR(255) NOT NULL UNIQUE,

    -- Security
    ip_address INET,
    user_agent TEXT,

    -- Timing
    expires_at TIMESTAMPTZ NOT NULL, -- Usually 5 minutes
    completed_at TIMESTAMPTZ, -- When 2FA was verified

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_twofa_pending_token ON twofa_pending_logins(session_token)
    WHERE completed_at IS NULL;
CREATE INDEX idx_twofa_pending_expires ON twofa_pending_logins(expires_at)
    WHERE completed_at IS NULL;

-- ========================================
-- AUTHENTICATION AUDIT LOG
-- ========================================

CREATE TABLE IF NOT EXISTS auth_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,

    -- Event details
    event_type VARCHAR(50) NOT NULL, -- 'login', 'login_failed', '2fa_enabled', '2fa_disabled', 'otp_sent', 'otp_verified', 'backup_code_used'
    event_status VARCHAR(20) NOT NULL, -- 'success', 'failure'
    event_details JSONB,

    -- Security context
    ip_address INET,
    user_agent TEXT,
    device_fingerprint VARCHAR(255),

    -- Related entities
    related_entity_type VARCHAR(50), -- 'trusted_device', 'backup_code', etc.
    related_entity_id UUID,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_auth_audit_user ON auth_audit_log(user_id, created_at DESC);
CREATE INDEX idx_auth_audit_event ON auth_audit_log(event_type, created_at DESC);
CREATE INDEX idx_auth_audit_time ON auth_audit_log(created_at DESC);

-- Partition auth audit log by month for better performance (optional)
-- This can be enabled in production for high-volume scenarios

-- ========================================
-- RATE LIMITING FOR OTP
-- ========================================

CREATE TABLE IF NOT EXISTS otp_rate_limits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identifier VARCHAR(255) NOT NULL, -- Phone number, email, or user_id
    identifier_type VARCHAR(20) NOT NULL, -- 'phone', 'email', 'user_id'

    -- Rate limiting
    request_count INT DEFAULT 1,
    window_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    window_duration_seconds INT NOT NULL DEFAULT 3600, -- 1 hour default
    max_requests INT NOT NULL DEFAULT 5, -- Max requests per window

    -- Lockout
    locked_until TIMESTAMPTZ,
    lock_reason VARCHAR(255),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(identifier, identifier_type)
);

CREATE INDEX idx_otp_rate_limits_identifier ON otp_rate_limits(identifier, identifier_type);
CREATE INDEX idx_otp_rate_limits_window ON otp_rate_limits(window_start);

-- ========================================
-- FUNCTIONS
-- ========================================

-- Function to check and update OTP rate limits
CREATE OR REPLACE FUNCTION check_otp_rate_limit(
    p_identifier VARCHAR(255),
    p_identifier_type VARCHAR(20),
    p_max_requests INT DEFAULT 5,
    p_window_seconds INT DEFAULT 3600
) RETURNS BOOLEAN AS $$
DECLARE
    v_record otp_rate_limits%ROWTYPE;
    v_now TIMESTAMPTZ := NOW();
BEGIN
    -- Get or create rate limit record
    SELECT * INTO v_record
    FROM otp_rate_limits
    WHERE identifier = p_identifier AND identifier_type = p_identifier_type;

    -- Check if locked
    IF v_record.id IS NOT NULL AND v_record.locked_until IS NOT NULL AND v_record.locked_until > v_now THEN
        RETURN FALSE;
    END IF;

    IF v_record.id IS NULL THEN
        -- Create new record
        INSERT INTO otp_rate_limits (identifier, identifier_type, request_count, window_start, window_duration_seconds, max_requests)
        VALUES (p_identifier, p_identifier_type, 1, v_now, p_window_seconds, p_max_requests);
        RETURN TRUE;
    ELSE
        -- Check if window has expired
        IF v_now > v_record.window_start + (v_record.window_duration_seconds || ' seconds')::INTERVAL THEN
            -- Reset window
            UPDATE otp_rate_limits
            SET request_count = 1, window_start = v_now, updated_at = v_now, locked_until = NULL
            WHERE id = v_record.id;
            RETURN TRUE;
        END IF;

        -- Check if limit exceeded
        IF v_record.request_count >= v_record.max_requests THEN
            -- Lock for extended period (e.g., 24 hours)
            UPDATE otp_rate_limits
            SET locked_until = v_now + INTERVAL '24 hours', lock_reason = 'Rate limit exceeded', updated_at = v_now
            WHERE id = v_record.id;
            RETURN FALSE;
        END IF;

        -- Increment counter
        UPDATE otp_rate_limits
        SET request_count = request_count + 1, updated_at = v_now
        WHERE id = v_record.id;
        RETURN TRUE;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function to cleanup expired OTPs and sessions
CREATE OR REPLACE FUNCTION cleanup_expired_auth_data() RETURNS void AS $$
BEGIN
    -- Delete expired OTP verifications
    DELETE FROM otp_verifications WHERE expires_at < NOW() - INTERVAL '1 day';

    -- Delete expired pending logins
    DELETE FROM twofa_pending_logins WHERE expires_at < NOW() - INTERVAL '1 hour';

    -- Delete expired trusted devices
    DELETE FROM trusted_devices WHERE expires_at < NOW() AND revoked_at IS NULL;

    -- Reset expired rate limit windows
    UPDATE otp_rate_limits
    SET request_count = 0, window_start = NOW(), locked_until = NULL
    WHERE window_start < NOW() - (window_duration_seconds || ' seconds')::INTERVAL;
END;
$$ LANGUAGE plpgsql;

-- ========================================
-- TRIGGERS
-- ========================================

-- Update timestamp trigger for rate limits
CREATE TRIGGER trigger_otp_rate_limits_updated_at
    BEFORE UPDATE ON otp_rate_limits
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ========================================
-- COMMENTS
-- ========================================

COMMENT ON TABLE otp_verifications IS 'Stores OTP codes for various verification purposes (login, phone verification, etc.)';
COMMENT ON TABLE backup_codes IS 'Emergency backup codes for 2FA recovery';
COMMENT ON TABLE trusted_devices IS 'Devices that have been trusted to skip 2FA';
COMMENT ON TABLE twofa_pending_logins IS 'Temporary sessions waiting for 2FA verification';
COMMENT ON TABLE auth_audit_log IS 'Audit log for all authentication-related events';
COMMENT ON TABLE otp_rate_limits IS 'Rate limiting for OTP requests to prevent abuse';

COMMENT ON COLUMN users.twofa_enabled IS 'Whether 2FA is enabled for this user';
COMMENT ON COLUMN users.twofa_method IS 'Preferred 2FA method: sms, totp, or both';
COMMENT ON COLUMN users.phone_verified_at IS 'When the phone number was verified';
COMMENT ON COLUMN users.totp_secret IS 'Encrypted TOTP secret for authenticator apps';
