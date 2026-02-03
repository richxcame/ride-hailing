-- Drop indexes first
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_target;
DROP INDEX IF EXISTS idx_audit_logs_admin;

-- Drop table
DROP TABLE IF EXISTS audit_logs;
