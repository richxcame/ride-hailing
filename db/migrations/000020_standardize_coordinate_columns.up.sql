-- Migration: Standardize all coordinate column names to use full words
-- This ensures consistency across the entire database schema

-- Rename abbreviated coordinate columns in cancellation_records
ALTER TABLE cancellation_records RENAME COLUMN pickup_lat TO pickup_latitude;
ALTER TABLE cancellation_records RENAME COLUMN pickup_lng TO pickup_longitude;

-- Note: All other tables already use full coordinate column names:
-- - rides table: pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude
-- - drivers table: current_latitude, current_longitude
-- - cities table: center_latitude, center_longitude
-- - pricing_zones table: uses PostGIS geometry column (no lat/lng columns)

COMMENT ON MIGRATION IS 'Standardize all coordinate columns to use full words (latitude/longitude) for consistency';
