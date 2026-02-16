-- Rollback: Remove approval status columns

DROP INDEX IF EXISTS idx_drivers_approval_status;

ALTER TABLE drivers
DROP COLUMN IF EXISTS approval_status,
DROP COLUMN IF EXISTS approved_by,
DROP COLUMN IF EXISTS approved_at,
DROP COLUMN IF EXISTS rejection_reason,
DROP COLUMN IF EXISTS rejected_at;

-- Restore is_available logic (approved = available)
UPDATE drivers SET is_available = true WHERE approval_status = 'approved';
