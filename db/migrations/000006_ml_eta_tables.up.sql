-- Migration for ML-based ETA prediction tables
-- Phase 3: Enterprise Ready - Machine Learning Integration

-- Table to store ETA predictions
CREATE TABLE IF NOT EXISTS eta_predictions (
    id SERIAL PRIMARY KEY,
    pickup_lat DOUBLE PRECISION NOT NULL,
    pickup_lng DOUBLE PRECISION NOT NULL,
    dropoff_lat DOUBLE PRECISION NOT NULL,
    dropoff_lng DOUBLE PRECISION NOT NULL,
    predicted_minutes DOUBLE PRECISION NOT NULL,
    actual_minutes DOUBLE PRECISION,
    distance DOUBLE PRECISION NOT NULL,
    traffic_level VARCHAR(20),
    weather VARCHAR(50),
    time_of_day INTEGER NOT NULL CHECK (time_of_day >= 0 AND time_of_day <= 23),
    day_of_week INTEGER NOT NULL CHECK (day_of_week >= 0 AND day_of_week <= 6),
    confidence DOUBLE PRECISION NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    ride_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_eta_predictions_ride_id ON eta_predictions(ride_id);
CREATE INDEX idx_eta_predictions_created_at ON eta_predictions(created_at DESC);
CREATE INDEX idx_eta_predictions_coordinates ON eta_predictions(pickup_lat, pickup_lng, dropoff_lat, dropoff_lng);
CREATE INDEX idx_eta_predictions_actual_not_null ON eta_predictions(actual_minutes) WHERE actual_minutes IS NOT NULL;

-- Table to store model training statistics
CREATE TABLE IF NOT EXISTS eta_model_stats (
    id SERIAL PRIMARY KEY,
    trained_at TIMESTAMP NOT NULL,
    total_predictions BIGINT NOT NULL DEFAULT 0,
    accuracy_rate DOUBLE PRECISION NOT NULL,
    mean_absolute_error DOUBLE PRECISION NOT NULL,
    model_config JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_eta_model_stats_trained_at ON eta_model_stats(trained_at DESC);

-- Table to track model performance over time
CREATE TABLE IF NOT EXISTS eta_model_performance (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL,
    total_predictions INTEGER NOT NULL DEFAULT 0,
    completed_rides INTEGER NOT NULL DEFAULT 0,
    mean_absolute_error DOUBLE PRECISION,
    accuracy_percentage DOUBLE PRECISION,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(date)
);

CREATE INDEX idx_eta_model_performance_date ON eta_model_performance(date DESC);

-- Materialized view for quick access to recent accuracy metrics
CREATE MATERIALIZED VIEW IF NOT EXISTS eta_accuracy_summary AS
SELECT
    DATE(created_at) as date,
    COUNT(*) as total_predictions,
    COUNT(CASE WHEN actual_minutes IS NOT NULL THEN 1 END) as completed_rides,
    AVG(CASE
        WHEN actual_minutes IS NOT NULL
        THEN ABS(predicted_minutes - actual_minutes)
    END) as mean_absolute_error,
    AVG(CASE
        WHEN actual_minutes IS NOT NULL AND actual_minutes > 0
        THEN (1 - LEAST(ABS(predicted_minutes - actual_minutes) / actual_minutes, 1)) * 100
    END) as accuracy_percentage,
    AVG(confidence) as avg_confidence
FROM eta_predictions
WHERE created_at > CURRENT_DATE - INTERVAL '90 days'
GROUP BY DATE(created_at)
ORDER BY date DESC;

CREATE UNIQUE INDEX idx_eta_accuracy_summary_date ON eta_accuracy_summary(date);

-- Function to refresh the materialized view
CREATE OR REPLACE FUNCTION refresh_eta_accuracy_summary()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY eta_accuracy_summary;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE eta_predictions IS 'Stores all ETA predictions made by the ML model';
COMMENT ON TABLE eta_model_stats IS 'Stores model training statistics and configurations';
COMMENT ON TABLE eta_model_performance IS 'Tracks daily model performance metrics';
COMMENT ON MATERIALIZED VIEW eta_accuracy_summary IS 'Aggregated view of model accuracy over the last 90 days';
COMMENT ON COLUMN eta_predictions.confidence IS 'Prediction confidence score between 0 and 1';
COMMENT ON COLUMN eta_predictions.actual_minutes IS 'Actual ride duration in minutes (populated after ride completion)';
