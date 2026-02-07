-- Rollback: Remove ride list indexes
DROP INDEX IF EXISTS idx_rides_scheduled_at;
DROP INDEX IF EXISTS idx_rides_driver_created;
DROP INDEX IF EXISTS idx_rides_rider_created;
