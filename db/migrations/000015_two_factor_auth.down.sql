-- Rollback Two-Factor Authentication Migration

-- Drop functions
DROP FUNCTION IF EXISTS cleanup_expired_auth_data();
DROP FUNCTION IF EXISTS check_otp_rate_limit(VARCHAR, VARCHAR, INT, INT);

-- Drop triggers (triggers are auto-dropped with tables)

-- Drop tables
DROP TABLE IF EXISTS auth_audit_log;
DROP TABLE IF EXISTS otp_rate_limits;
DROP TABLE IF EXISTS twofa_pending_logins;
DROP TABLE IF EXISTS trusted_devices;
DROP TABLE IF EXISTS backup_codes;
DROP TABLE IF EXISTS otp_verifications;

-- Drop indexes on users table
DROP INDEX IF EXISTS idx_users_twofa;

-- Remove 2FA columns from users table
ALTER TABLE users DROP COLUMN IF EXISTS twofa_enabled_at;
ALTER TABLE users DROP COLUMN IF EXISTS totp_verified_at;
ALTER TABLE users DROP COLUMN IF EXISTS totp_secret;
ALTER TABLE users DROP COLUMN IF EXISTS phone_verified_at;
ALTER TABLE users DROP COLUMN IF EXISTS twofa_method;
ALTER TABLE users DROP COLUMN IF EXISTS twofa_enabled;
