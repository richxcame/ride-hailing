-- Drop triggers
DROP TRIGGER IF EXISTS fraud_alerts_updated_at ON fraud_alerts;
DROP FUNCTION IF EXISTS update_fraud_alerts_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_fraud_alerts_user_id;
DROP INDEX IF EXISTS idx_fraud_alerts_status;
DROP INDEX IF EXISTS idx_fraud_alerts_alert_level;
DROP INDEX IF EXISTS idx_fraud_alerts_alert_type;
DROP INDEX IF EXISTS idx_fraud_alerts_detected_at;
DROP INDEX IF EXISTS idx_fraud_alerts_investigated_by;
DROP INDEX IF EXISTS idx_user_risk_profiles_risk_score;
DROP INDEX IF EXISTS idx_user_risk_profiles_account_suspended;
DROP INDEX IF EXISTS idx_user_risk_profiles_last_alert_at;

-- Drop tables
DROP TABLE IF EXISTS user_risk_profiles;
DROP TABLE IF EXISTS fraud_alerts;
