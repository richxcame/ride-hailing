-- Remove trigger
DROP TRIGGER IF EXISTS update_notifications_updated_at ON notifications;

-- Remove indexes
DROP INDEX IF EXISTS idx_notifications_scheduled;
DROP INDEX IF EXISTS idx_notifications_status;

-- Remove columns
ALTER TABLE notifications DROP COLUMN IF EXISTS updated_at;
ALTER TABLE notifications DROP COLUMN IF EXISTS error_message;
ALTER TABLE notifications DROP COLUMN IF EXISTS scheduled_at;
ALTER TABLE notifications DROP COLUMN IF EXISTS status;
