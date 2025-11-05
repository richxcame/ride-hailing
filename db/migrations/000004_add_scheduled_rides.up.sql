-- Add scheduled ride columns to rides table
ALTER TABLE rides ADD COLUMN scheduled_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE rides ADD COLUMN is_scheduled BOOLEAN DEFAULT false;
ALTER TABLE rides ADD COLUMN scheduled_notification_sent BOOLEAN DEFAULT false;

-- Create index for scheduled rides
CREATE INDEX idx_rides_scheduled_at ON rides(scheduled_at) WHERE is_scheduled = true;
CREATE INDEX idx_rides_is_scheduled ON rides(is_scheduled, scheduled_at);

-- Create scheduled_rides view for easier querying
CREATE VIEW scheduled_rides AS
SELECT
    id,
    rider_id,
    driver_id,
    status,
    pickup_address,
    dropoff_address,
    scheduled_at,
    requested_at,
    estimated_fare,
    ride_type_id,
    promo_code_id
FROM rides
WHERE is_scheduled = true
AND scheduled_at > NOW()
AND status = 'requested'
ORDER BY scheduled_at ASC;

-- Create function to get upcoming scheduled rides
CREATE OR REPLACE FUNCTION get_upcoming_scheduled_rides(minutes_ahead INTEGER DEFAULT 30)
RETURNS TABLE (
    ride_id UUID,
    rider_id UUID,
    scheduled_at TIMESTAMP WITH TIME ZONE,
    pickup_address TEXT,
    dropoff_address TEXT,
    estimated_fare DECIMAL(10,2)
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        r.id,
        r.rider_id,
        r.scheduled_at,
        r.pickup_address,
        r.dropoff_address,
        r.estimated_fare
    FROM rides r
    WHERE r.is_scheduled = true
    AND r.status = 'requested'
    AND r.scheduled_at <= NOW() + (minutes_ahead || ' minutes')::INTERVAL
    AND r.scheduled_at > NOW()
    AND r.scheduled_notification_sent = false
    ORDER BY r.scheduled_at ASC;
END;
$$ LANGUAGE plpgsql;
