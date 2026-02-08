-- Add indexes for ride list queries (rider/driver history sorted by time)
-- These complement the existing status-based indexes for pure time-based lookups

-- Index for rider ride history (most recent first)
-- Covers: GET /riders/:id/rides?page=X queries
CREATE INDEX IF NOT EXISTS idx_rides_rider_created
ON rides(rider_id, created_at DESC);

-- Index for driver ride history (most recent first)
-- Covers: GET /drivers/:id/rides?page=X queries
CREATE INDEX IF NOT EXISTS idx_rides_driver_created
ON rides(driver_id, created_at DESC)
WHERE driver_id IS NOT NULL;

-- Index for recent rides lookup by scheduled time
-- Covers: scheduled ride queries and admin dashboard
CREATE INDEX IF NOT EXISTS idx_rides_scheduled_at
ON rides(scheduled_at DESC)
WHERE scheduled_at IS NOT NULL;

-- Comment documentation
COMMENT ON INDEX idx_rides_rider_created IS 'Optimizes rider ride history queries sorted by time';
COMMENT ON INDEX idx_rides_driver_created IS 'Optimizes driver ride history queries sorted by time';
COMMENT ON INDEX idx_rides_scheduled_at IS 'Optimizes scheduled rides lookup';
