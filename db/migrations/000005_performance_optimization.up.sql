-- Performance Optimization Migration
-- This migration adds PostGIS support, fixes payment schema, and adds performance indexes

-- 1. Enable PostGIS extension for geospatial queries
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS postgis_topology;

-- 2. Fix payments table schema to match repository/model
-- Add missing Stripe-related columns
ALTER TABLE payments ADD COLUMN IF NOT EXISTS payment_method VARCHAR(20);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS stripe_payment_id VARCHAR(255);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS stripe_charge_id VARCHAR(255);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS currency VARCHAR(3) DEFAULT 'USD';
ALTER TABLE payments ADD COLUMN IF NOT EXISTS metadata JSONB;

-- Make commission and driver_earnings nullable (calculated fields)
ALTER TABLE payments ALTER COLUMN commission DROP NOT NULL;
ALTER TABLE payments ALTER COLUMN driver_earnings DROP NOT NULL;

-- Update existing records to set payment_method from method column
UPDATE payments SET payment_method = method WHERE payment_method IS NULL;

-- 3. Add is_active column to wallets (for better queries)
ALTER TABLE wallets ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;

-- 4. Create optimized indexes for performance

-- Payments indexes
CREATE INDEX IF NOT EXISTS idx_payments_rider_id_created_at ON payments(rider_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payments_driver_id_created_at ON payments(driver_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status) WHERE status IN ('pending', 'failed');
CREATE INDEX IF NOT EXISTS idx_payments_stripe_payment_id ON payments(stripe_payment_id) WHERE stripe_payment_id IS NOT NULL;

-- Rides indexes for common queries
CREATE INDEX IF NOT EXISTS idx_rides_rider_id_status ON rides(rider_id, status);
CREATE INDEX IF NOT EXISTS idx_rides_driver_id_status ON rides(driver_id, status) WHERE driver_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_rides_status_created_at ON rides(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_rides_completed_at ON rides(completed_at DESC) WHERE completed_at IS NOT NULL;

-- Wallet transactions indexes
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_wallet_created ON wallet_transactions(wallet_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_reference ON wallet_transactions(reference_type, reference_id) WHERE reference_id IS NOT NULL;

-- Driver location indexes for geospatial queries
CREATE INDEX IF NOT EXISTS idx_driver_locations_spatial ON driver_locations USING GIST(ST_MakePoint(longitude, latitude));
CREATE INDEX IF NOT EXISTS idx_driver_locations_driver_time ON driver_locations(driver_id, recorded_at DESC);

-- Drivers geospatial index
CREATE INDEX IF NOT EXISTS idx_drivers_location ON drivers USING GIST(ST_MakePoint(current_longitude, current_latitude))
  WHERE current_longitude IS NOT NULL AND current_latitude IS NOT NULL AND is_available = true;

-- User indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_role_active ON users(role, is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_users_email_lower ON users(LOWER(email));

-- 5. Add materialized view for driver statistics (cache heavy computations)
CREATE MATERIALIZED VIEW IF NOT EXISTS driver_statistics AS
SELECT
    d.id AS driver_id,
    d.user_id,
    d.rating,
    d.total_rides,
    COUNT(DISTINCT r.id) AS completed_rides_count,
    COALESCE(SUM(p.driver_earnings), 0) AS total_earnings,
    COALESCE(AVG(r.rating), 0) AS average_rating,
    MAX(r.completed_at) AS last_ride_completed
FROM drivers d
LEFT JOIN rides r ON r.driver_id = d.user_id AND r.status = 'completed'
LEFT JOIN payments p ON p.ride_id = r.id
GROUP BY d.id, d.user_id, d.rating, d.total_rides;

-- Create index on materialized view
CREATE UNIQUE INDEX IF NOT EXISTS idx_driver_stats_driver_id ON driver_statistics(driver_id);

-- 6. Add refresh function for materialized view
CREATE OR REPLACE FUNCTION refresh_driver_statistics()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY driver_statistics;
END;
$$ LANGUAGE plpgsql;

-- 7. Create function to get nearby drivers using PostGIS
CREATE OR REPLACE FUNCTION get_nearby_drivers_postgis(
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    radius_meters DOUBLE PRECISION DEFAULT 10000
)
RETURNS TABLE (
    driver_id UUID,
    user_id UUID,
    distance_meters DOUBLE PRECISION,
    current_latitude DECIMAL(10,8),
    current_longitude DECIMAL(11,8),
    rating DECIMAL(3,2),
    total_rides INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        d.id,
        d.user_id,
        ST_Distance(
            ST_MakePoint(longitude, latitude)::geography,
            ST_MakePoint(d.current_longitude, d.current_latitude)::geography
        ) AS distance_meters,
        d.current_latitude,
        d.current_longitude,
        d.rating,
        d.total_rides
    FROM drivers d
    WHERE d.is_available = true
      AND d.is_online = true
      AND d.current_latitude IS NOT NULL
      AND d.current_longitude IS NOT NULL
      AND ST_DWithin(
          ST_MakePoint(longitude, latitude)::geography,
          ST_MakePoint(d.current_longitude, d.current_latitude)::geography,
          radius_meters
      )
    ORDER BY distance_meters ASC
    LIMIT 20;
END;
$$ LANGUAGE plpgsql;

-- 8. Add pg_stat_statements extension for query performance monitoring
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- 9. Create view for slow query monitoring
CREATE OR REPLACE VIEW slow_queries AS
SELECT
    query,
    calls,
    total_exec_time,
    mean_exec_time,
    max_exec_time,
    stddev_exec_time,
    rows
FROM pg_stat_statements
WHERE mean_exec_time > 100 -- queries taking more than 100ms on average
ORDER BY mean_exec_time DESC;

-- 10. Add comment documentation
COMMENT ON EXTENSION postgis IS 'PostGIS geometry and geography spatial types and functions';
COMMENT ON EXTENSION pg_stat_statements IS 'Track planning and execution statistics of all SQL statements';
COMMENT ON MATERIALIZED VIEW driver_statistics IS 'Cached driver performance statistics - refresh periodically';
COMMENT ON FUNCTION refresh_driver_statistics() IS 'Refresh the driver_statistics materialized view concurrently';
COMMENT ON FUNCTION get_nearby_drivers_postgis(DOUBLE PRECISION, DOUBLE PRECISION, DOUBLE PRECISION) IS 'Find nearby available drivers using PostGIS within radius (latitude, longitude, radius_meters)';
