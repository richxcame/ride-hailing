-- Migration: Standardize ML ETA table coordinate column names

-- Rename abbreviated coordinate columns in eta_predictions
ALTER TABLE eta_predictions RENAME COLUMN pickup_lat TO pickup_latitude;
ALTER TABLE eta_predictions RENAME COLUMN pickup_lng TO pickup_longitude;
ALTER TABLE eta_predictions RENAME COLUMN dropoff_lat TO dropoff_latitude;
ALTER TABLE eta_predictions RENAME COLUMN dropoff_lng TO dropoff_longitude;

-- Drop old index with abbreviated names
DROP INDEX IF EXISTS idx_eta_predictions_coordinates;

-- Recreate index with full column names
CREATE INDEX idx_eta_predictions_coordinates ON eta_predictions(pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude);

COMMENT ON MIGRATION IS 'Standardize ML ETA coordinate columns to use full words';
