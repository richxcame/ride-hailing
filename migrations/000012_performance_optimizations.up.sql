-- Performance optimization indexes

-- Users table optimizations
CREATE INDEX IF NOT EXISTS idx_users_email_lower ON users(LOWER(email));
CREATE INDEX IF NOT EXISTS idx_users_role_active ON users(role, is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);

-- Rides table optimizations
CREATE INDEX IF NOT EXISTS idx_rides_status_requested_at ON rides(status, requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_rides_driver_status ON rides(driver_id, status) WHERE driver_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_rides_rider_requested_at ON rides(rider_id, requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_rides_driver_requested_at ON rides(driver_id, requested_at DESC) WHERE driver_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_rides_completed_at ON rides(completed_at DESC) WHERE completed_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_rides_scheduled ON rides(scheduled_at) WHERE is_scheduled = true;
CREATE INDEX IF NOT EXISTS idx_rides_promo_code ON rides(promo_code_id) WHERE promo_code_id IS NOT NULL;

-- Partial index for pending rides (most commonly queried)
CREATE INDEX IF NOT EXISTS idx_rides_pending ON rides(requested_at DESC)
    WHERE status = 'requested';

-- Payments table optimizations
CREATE INDEX IF NOT EXISTS idx_payments_user_created ON payments(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payments_ride ON payments(ride_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_method ON payments(payment_method_id);
CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments(created_at DESC);

-- Notifications table optimizations
CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_sent_at ON notifications(sent_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications(notification_type);
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);

-- Promo codes table optimizations
CREATE INDEX IF NOT EXISTS idx_promo_codes_code_active ON promo_codes(code)
    WHERE is_active = true AND (valid_until IS NULL OR valid_until > NOW());
CREATE INDEX IF NOT EXISTS idx_promo_codes_valid_dates ON promo_codes(valid_from, valid_until);

-- Referrals table optimizations
CREATE INDEX IF NOT EXISTS idx_referrals_referrer ON referrals(referrer_id);
CREATE INDEX IF NOT EXISTS idx_referrals_referee ON referrals(referee_id);
CREATE INDEX IF NOT EXISTS idx_referrals_code ON referrals(referral_code);
CREATE INDEX IF NOT EXISTS idx_referrals_status ON referrals(status);

-- Create materialized view for popular demand zones (faster analytics)
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_demand_zones AS
SELECT
    ROUND(pickup_latitude::numeric, 2) as lat_zone,
    ROUND(pickup_longitude::numeric, 2) as lon_zone,
    COUNT(*) as total_rides,
    AVG(surge_multiplier) as avg_surge,
    AVG(final_fare) as avg_fare,
    COUNT(DISTINCT rider_id) as unique_riders,
    COUNT(DISTINCT driver_id) as unique_drivers
FROM rides
WHERE status = 'completed'
  AND completed_at >= NOW() - INTERVAL '30 days'
GROUP BY lat_zone, lon_zone
HAVING COUNT(*) > 10
ORDER BY total_rides DESC;

CREATE INDEX ON mv_demand_zones(lat_zone, lon_zone);
CREATE INDEX ON mv_demand_zones(total_rides DESC);

-- Create materialized view for driver performance (faster queries)
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_driver_performance AS
SELECT
    d.id as driver_id,
    u.first_name,
    u.last_name,
    u.email,
    COUNT(*) as total_rides,
    AVG(r.driver_rating) as avg_rating,
    SUM(CASE WHEN r.status = 'completed' THEN 1 ELSE 0 END) as completed_rides,
    SUM(CASE WHEN r.status = 'cancelled' THEN 1 ELSE 0 END) as cancelled_rides,
    ROUND(SUM(CASE WHEN r.status = 'completed' THEN 1 ELSE 0 END)::numeric /
          NULLIF(COUNT(*), 0) * 100, 2) as completion_rate,
    SUM(CASE WHEN r.status = 'completed' THEN COALESCE(r.final_fare, 0) ELSE 0 END) as total_earnings,
    d.vehicle_type,
    d.license_plate,
    d.is_active
FROM drivers d
INNER JOIN users u ON d.user_id = u.id
LEFT JOIN rides r ON d.id = r.driver_id
GROUP BY d.id, u.first_name, u.last_name, u.email, d.vehicle_type, d.license_plate, d.is_active;

CREATE INDEX ON mv_driver_performance(driver_id);
CREATE INDEX ON mv_driver_performance(avg_rating DESC);
CREATE INDEX ON mv_driver_performance(total_earnings DESC);

-- Create materialized view for revenue metrics (faster dashboard)
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_revenue_metrics AS
SELECT
    DATE(completed_at) as date,
    COUNT(*) as total_rides,
    SUM(COALESCE(final_fare, 0)) as total_revenue,
    AVG(COALESCE(final_fare, 0)) as avg_fare,
    SUM(COALESCE(discount_amount, 0)) as total_discounts,
    SUM(COALESCE(final_fare, 0) * 0.20) as platform_commission,
    SUM(COALESCE(final_fare, 0) * 0.80) as driver_earnings,
    AVG(surge_multiplier) as avg_surge
FROM rides
WHERE status = 'completed'
  AND completed_at IS NOT NULL
  AND completed_at >= NOW() - INTERVAL '90 days'
GROUP BY DATE(completed_at)
ORDER BY date DESC;

CREATE INDEX ON mv_revenue_metrics(date DESC);

-- Function to refresh materialized views (call this periodically via scheduler)
CREATE OR REPLACE FUNCTION refresh_analytics_views()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_demand_zones;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_driver_performance;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_revenue_metrics;
END;
$$ LANGUAGE plpgsql;

-- Partition rides table by date for better performance (for large datasets)
-- Note: This is commented out as it requires recreating the table
-- Uncomment if you want to enable partitioning for production at scale

-- ALTER TABLE rides RENAME TO rides_old;
--
-- CREATE TABLE rides (
--     LIKE rides_old INCLUDING ALL
-- ) PARTITION BY RANGE (requested_at);
--
-- -- Create partitions for current and future months
-- CREATE TABLE rides_2025_01 PARTITION OF rides
--     FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
-- CREATE TABLE rides_2025_02 PARTITION OF rides
--     FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
--
-- -- Migrate data
-- INSERT INTO rides SELECT * FROM rides_old;
-- DROP TABLE rides_old;

-- Optimize autovacuum settings for high-traffic tables
ALTER TABLE rides SET (
    autovacuum_vacuum_scale_factor = 0.05,
    autovacuum_analyze_scale_factor = 0.02
);

ALTER TABLE payments SET (
    autovacuum_vacuum_scale_factor = 0.05,
    autovacuum_analyze_scale_factor = 0.02
);

ALTER TABLE notifications SET (
    autovacuum_vacuum_scale_factor = 0.1,
    autovacuum_analyze_scale_factor = 0.05
);

-- Add table statistics for better query planning
ANALYZE users;
ANALYZE rides;
ANALYZE drivers;
ANALYZE payments;
ANALYZE notifications;
ANALYZE promo_codes;
ANALYZE referrals;

-- Comments
COMMENT ON MATERIALIZED VIEW mv_demand_zones IS 'Aggregated demand zones for faster analytics queries';
COMMENT ON MATERIALIZED VIEW mv_driver_performance IS 'Pre-calculated driver performance metrics';
COMMENT ON MATERIALIZED VIEW mv_revenue_metrics IS 'Daily revenue metrics for dashboard';
COMMENT ON FUNCTION refresh_analytics_views() IS 'Refreshes all materialized views for analytics';
