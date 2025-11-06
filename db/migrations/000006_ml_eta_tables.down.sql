-- Rollback migration for ML-based ETA prediction tables

DROP MATERIALIZED VIEW IF EXISTS eta_accuracy_summary CASCADE;
DROP FUNCTION IF EXISTS refresh_eta_accuracy_summary();
DROP TABLE IF EXISTS eta_model_performance CASCADE;
DROP TABLE IF EXISTS eta_model_stats CASCADE;
DROP TABLE IF EXISTS eta_predictions CASCADE;
