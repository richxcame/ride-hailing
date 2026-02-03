-- Audit logs table for admin action tracking
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id UUID NOT NULL REFERENCES users(id),
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id UUID NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for efficient queries by admin
CREATE INDEX idx_audit_logs_admin ON audit_logs(admin_id);

-- Composite index for queries by target (e.g., "get all actions on user X")
CREATE INDEX idx_audit_logs_target ON audit_logs(target_type, target_id);

-- Index for chronological queries
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
