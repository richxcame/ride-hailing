-- Drop function
DROP FUNCTION IF EXISTS refresh_analytics_views();

-- Drop materialized views
DROP MATERIALIZED VIEW IF EXISTS mv_revenue_metrics;
DROP MATERIALIZED VIEW IF EXISTS mv_driver_performance;
DROP MATERIALIZED VIEW IF EXISTS mv_demand_zones;

-- Drop indexes on referrals
DROP INDEX IF EXISTS idx_referrals_status;
DROP INDEX IF EXISTS idx_referrals_code;
DROP INDEX IF EXISTS idx_referrals_referee;
DROP INDEX IF EXISTS idx_referrals_referrer;

-- Drop indexes on promo_codes
DROP INDEX IF EXISTS idx_promo_codes_valid_dates;
DROP INDEX IF EXISTS idx_promo_codes_code_active;

-- Drop indexes on notifications
DROP INDEX IF EXISTS idx_notifications_status;
DROP INDEX IF EXISTS idx_notifications_type;
DROP INDEX IF EXISTS idx_notifications_sent_at;
DROP INDEX IF EXISTS idx_notifications_user_created;

-- Drop indexes on payments
DROP INDEX IF EXISTS idx_payments_created_at;
DROP INDEX IF EXISTS idx_payments_method;
DROP INDEX IF EXISTS idx_payments_status;
DROP INDEX IF EXISTS idx_payments_ride;
DROP INDEX IF EXISTS idx_payments_user_created;

-- Drop indexes on rides
DROP INDEX IF EXISTS idx_rides_pending;
DROP INDEX IF EXISTS idx_rides_promo_code;
DROP INDEX IF EXISTS idx_rides_scheduled;
DROP INDEX IF EXISTS idx_rides_completed_at;
DROP INDEX IF EXISTS idx_rides_driver_requested_at;
DROP INDEX IF EXISTS idx_rides_rider_requested_at;
DROP INDEX IF EXISTS idx_rides_driver_status;
DROP INDEX IF EXISTS idx_rides_status_requested_at;

-- Drop indexes on users
DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_users_role_active;
DROP INDEX IF EXISTS idx_users_email_lower;

-- Reset autovacuum settings to defaults
ALTER TABLE rides RESET (autovacuum_vacuum_scale_factor, autovacuum_analyze_scale_factor);
ALTER TABLE payments RESET (autovacuum_vacuum_scale_factor, autovacuum_analyze_scale_factor);
ALTER TABLE notifications RESET (autovacuum_vacuum_scale_factor, autovacuum_analyze_scale_factor);
