-- Prevent duplicate ride_fare earnings for the same ride
CREATE UNIQUE INDEX IF NOT EXISTS idx_driver_earnings_ride_type
    ON driver_earnings (ride_id, type)
    WHERE ride_id IS NOT NULL;
