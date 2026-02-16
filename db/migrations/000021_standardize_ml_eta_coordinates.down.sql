-- Rollback: Revert ML ETA coordinate column names

DROP INDEX IF EXISTS idx_eta_predictions_coordinates;

ALTER TABLE eta_predictions RENAME COLUMN pickup_latitude TO pickup_lat;
ALTER TABLE eta_predictions RENAME COLUMN pickup_longitude TO pickup_lng;
ALTER TABLE eta_predictions RENAME COLUMN dropoff_latitude TO dropoff_lat;
ALTER TABLE eta_predictions RENAME COLUMN dropoff_longitude TO dropoff_lng;

CREATE INDEX idx_eta_predictions_coordinates ON eta_predictions(pickup_lat, pickup_lng, dropoff_lat, dropoff_lng);
