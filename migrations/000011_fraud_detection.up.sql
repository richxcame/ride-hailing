-- Fraud alerts table
CREATE TABLE IF NOT EXISTS fraud_alerts (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    alert_type VARCHAR(50) NOT NULL,
    alert_level VARCHAR(20) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    description TEXT NOT NULL,
    details JSONB,
    risk_score DECIMAL(5,2) NOT NULL DEFAULT 0,
    detected_at TIMESTAMP NOT NULL DEFAULT NOW(),
    investigated_at TIMESTAMP,
    investigated_by UUID REFERENCES users(id),
    resolved_at TIMESTAMP,
    notes TEXT,
    action_taken TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- User risk profiles table
CREATE TABLE IF NOT EXISTS user_risk_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id),
    risk_score DECIMAL(5,2) NOT NULL DEFAULT 0,
    total_alerts INTEGER NOT NULL DEFAULT 0,
    critical_alerts INTEGER NOT NULL DEFAULT 0,
    confirmed_fraud_cases INTEGER NOT NULL DEFAULT 0,
    last_alert_at TIMESTAMP,
    account_suspended BOOLEAN NOT NULL DEFAULT FALSE,
    last_updated TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for fraud_alerts
CREATE INDEX idx_fraud_alerts_user_id ON fraud_alerts(user_id);
CREATE INDEX idx_fraud_alerts_status ON fraud_alerts(status);
CREATE INDEX idx_fraud_alerts_alert_level ON fraud_alerts(alert_level);
CREATE INDEX idx_fraud_alerts_alert_type ON fraud_alerts(alert_type);
CREATE INDEX idx_fraud_alerts_detected_at ON fraud_alerts(detected_at DESC);
CREATE INDEX idx_fraud_alerts_investigated_by ON fraud_alerts(investigated_by);

-- Indexes for user_risk_profiles
CREATE INDEX idx_user_risk_profiles_risk_score ON user_risk_profiles(risk_score DESC);
CREATE INDEX idx_user_risk_profiles_account_suspended ON user_risk_profiles(account_suspended);
CREATE INDEX idx_user_risk_profiles_last_alert_at ON user_risk_profiles(last_alert_at DESC);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_fraud_alerts_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER fraud_alerts_updated_at
    BEFORE UPDATE ON fraud_alerts
    FOR EACH ROW
    EXECUTE FUNCTION update_fraud_alerts_updated_at();

-- Comments
COMMENT ON TABLE fraud_alerts IS 'Stores fraud detection alerts';
COMMENT ON TABLE user_risk_profiles IS 'Stores user risk assessment profiles';
COMMENT ON COLUMN fraud_alerts.alert_type IS 'Type of fraud: payment_fraud, account_fraud, location_fraud, ride_fraud, rating_manipulation, promo_abuse';
COMMENT ON COLUMN fraud_alerts.alert_level IS 'Severity: low, medium, high, critical';
COMMENT ON COLUMN fraud_alerts.status IS 'Status: pending, investigating, confirmed, false_positive, resolved';
COMMENT ON COLUMN user_risk_profiles.risk_score IS 'Risk score from 0-100';
