-- Drop function
DROP FUNCTION IF EXISTS get_upcoming_scheduled_rides(INTEGER);

-- Drop view
DROP VIEW IF EXISTS scheduled_rides;

-- Drop indexes
DROP INDEX IF EXISTS idx_rides_is_scheduled;
DROP INDEX IF EXISTS idx_rides_scheduled_at;

-- Remove columns
ALTER TABLE rides DROP COLUMN IF EXISTS scheduled_notification_sent;
ALTER TABLE rides DROP COLUMN IF EXISTS is_scheduled;
ALTER TABLE rides DROP COLUMN IF EXISTS scheduled_at;
