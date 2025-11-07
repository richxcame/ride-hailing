-- Analytics Materialized Views for Scheduler
-- These views cache expensive analytical queries for performance

-- 1. Demand Zones - High-demand geographical areas
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_demand_zones AS
SELECT
    ST_SnapToGrid(ST_MakePoint(pickup_longitude, pickup_latitude), 0.01) AS zone_point,
    COUNT(*) AS ride_count,
    AVG(estimated_fare) AS avg_fare,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) AS completed_count,
    COUNT(CASE WHEN status = 'cancelled' THEN 1 END) AS cancelled_count,
    EXTRACT(HOUR FROM requested_at) AS hour_of_day,
    EXTRACT(DOW FROM requested_at) AS day_of_week
FROM rides
WHERE requested_at >= NOW() - INTERVAL '7 days'
GROUP BY zone_point, hour_of_day, day_of_week
HAVING COUNT(*) >= 5;

-- Create index on demand zones
CREATE INDEX IF NOT EXISTS idx_mv_demand_zones_zone 
ON mv_demand_zones USING GIST(zone_point);

CREATE INDEX IF NOT EXISTS idx_mv_demand_zones_time 
ON mv_demand_zones(hour_of_day, day_of_week);

-- 2. Driver Performance Metrics
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_driver_performance AS
SELECT
    d.id AS driver_id,
    d.user_id,
    u.first_name,
    u.last_name,
    d.rating,
    d.total_rides,
    COUNT(DISTINCT r.id) AS rides_last_30_days,
    COUNT(DISTINCT CASE WHEN r.status = 'completed' THEN r.id END) AS completed_rides,
    COUNT(DISTINCT CASE WHEN r.status = 'cancelled' THEN r.id END) AS cancelled_rides,
    COALESCE(AVG(r.rating), 0) AS avg_customer_rating,
    COALESCE(SUM(p.driver_earnings), 0) AS total_earnings_30_days,
    COALESCE(AVG(p.driver_earnings), 0) AS avg_earnings_per_ride,
    COALESCE(AVG(r.actual_duration), 0) AS avg_ride_duration,
    MAX(r.completed_at) AS last_ride_at,
    -- Calculate acceptance rate
    ROUND(
        100.0 * COUNT(DISTINCT CASE WHEN r.status != 'requested' THEN r.id END) / 
        NULLIF(COUNT(DISTINCT r.id), 0), 
        2
    ) AS acceptance_rate,
    -- Calculate completion rate
    ROUND(
        100.0 * COUNT(DISTINCT CASE WHEN r.status = 'completed' THEN r.id END) / 
        NULLIF(COUNT(DISTINCT CASE WHEN r.status IN ('completed', 'cancelled') THEN r.id END), 0),
        2
    ) AS completion_rate
FROM drivers d
JOIN users u ON u.id = d.user_id
LEFT JOIN rides r ON r.driver_id = d.user_id 
    AND r.requested_at >= NOW() - INTERVAL '30 days'
LEFT JOIN payments p ON p.ride_id = r.id 
    AND p.status = 'completed'
GROUP BY d.id, d.user_id, u.first_name, u.last_name, d.rating, d.total_rides;

-- Create index on driver performance
CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_driver_performance_driver_id 
ON mv_driver_performance(driver_id);

CREATE INDEX IF NOT EXISTS idx_mv_driver_performance_rating 
ON mv_driver_performance(avg_customer_rating DESC);

-- 3. Revenue Metrics
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_revenue_metrics AS
SELECT
    DATE(p.created_at) AS date,
    COUNT(DISTINCT p.id) AS total_transactions,
    COUNT(DISTINCT p.ride_id) AS total_rides,
    COUNT(DISTINCT p.rider_id) AS unique_riders,
    COUNT(DISTINCT p.driver_id) AS active_drivers,
    COALESCE(SUM(p.amount), 0) AS gross_revenue,
    COALESCE(SUM(p.commission), 0) AS platform_commission,
    COALESCE(SUM(p.driver_earnings), 0) AS driver_earnings,
    COALESCE(AVG(p.amount), 0) AS avg_transaction_amount,
    -- Payment method breakdown
    COUNT(CASE WHEN p.method = 'card' THEN 1 END) AS card_payments,
    COUNT(CASE WHEN p.method = 'wallet' THEN 1 END) AS wallet_payments,
    COUNT(CASE WHEN p.method = 'cash' THEN 1 END) AS cash_payments,
    -- Status breakdown
    COUNT(CASE WHEN p.status = 'completed' THEN 1 END) AS successful_payments,
    COUNT(CASE WHEN p.status = 'failed' THEN 1 END) AS failed_payments,
    COUNT(CASE WHEN p.status = 'refunded' THEN 1 END) AS refunded_payments,
    -- Calculate success rate
    ROUND(
        100.0 * COUNT(CASE WHEN p.status = 'completed' THEN 1 END) / 
        NULLIF(COUNT(*), 0),
        2
    ) AS payment_success_rate
FROM payments p
WHERE p.created_at >= NOW() - INTERVAL '90 days'
GROUP BY DATE(p.created_at)
ORDER BY date DESC;

-- Create index on revenue metrics
CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_revenue_metrics_date 
ON mv_revenue_metrics(date DESC);

-- Add comments for documentation
COMMENT ON MATERIALIZED VIEW mv_demand_zones IS 
'High-demand geographical zones aggregated by location, time, and day - refreshed by scheduler';

COMMENT ON MATERIALIZED VIEW mv_driver_performance IS 
'Driver performance metrics including ratings, earnings, and completion rates - refreshed by scheduler';

COMMENT ON MATERIALIZED VIEW mv_revenue_metrics IS 
'Daily revenue and transaction metrics for business analytics - refreshed by scheduler';
