-- Rollback Safety Features Migration

-- Drop triggers first
DROP TRIGGER IF EXISTS trigger_update_emergency_alert_timestamp ON emergency_alerts;
DROP TRIGGER IF EXISTS trigger_update_emergency_contact_timestamp ON emergency_contacts;
DROP TRIGGER IF EXISTS trigger_update_ride_share_link_timestamp ON ride_share_links;
DROP TRIGGER IF EXISTS trigger_update_safety_settings_timestamp ON safety_settings;

-- Drop function
DROP FUNCTION IF EXISTS update_emergency_alert_timestamp();

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS emergency_alert_notifications;
DROP TABLE IF EXISTS safety_incident_reports;
DROP TABLE IF EXISTS blocked_drivers;
DROP TABLE IF EXISTS trusted_drivers;
DROP TABLE IF EXISTS speed_alerts;
DROP TABLE IF EXISTS route_deviation_alerts;
DROP TABLE IF EXISTS safety_checks;
DROP TABLE IF EXISTS safety_settings;
DROP TABLE IF EXISTS ride_share_recipients;
DROP TABLE IF EXISTS ride_share_links;
DROP TABLE IF EXISTS emergency_contacts;
DROP TABLE IF EXISTS emergency_alerts;
