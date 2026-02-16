-- Rollback: Revert coordinate column names to abbreviated form

ALTER TABLE cancellation_records RENAME COLUMN pickup_latitude TO pickup_lat;
ALTER TABLE cancellation_records RENAME COLUMN pickup_longitude TO pickup_lng;
