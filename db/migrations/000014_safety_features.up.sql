-- Safety Features Migration
-- Emergency SOS, Ride Sharing, Safety Settings, and Incident Reporting

-- ========================================
-- EMERGENCY ALERTS
-- ========================================

CREATE TABLE IF NOT EXISTS emergency_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    ride_id UUID REFERENCES rides(id),
    type VARCHAR(50) NOT NULL, -- 'sos', 'accident', 'medical', 'threat', 'other'
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- 'active', 'responded', 'resolved', 'cancelled', 'false_alarm'

    -- Location at time of alert
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    address TEXT,

    -- Alert details
    description TEXT,
    audio_recording_url TEXT,
    video_recording_url TEXT,

    -- Response tracking
    responded_at TIMESTAMPTZ,
    responded_by UUID REFERENCES users(id), -- Admin user
    resolved_at TIMESTAMPTZ,
    resolution TEXT,

    -- Emergency services
    police_notified BOOLEAN DEFAULT FALSE,
    police_ref_number VARCHAR(100),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_emergency_alerts_user ON emergency_alerts(user_id);
CREATE INDEX idx_emergency_alerts_status ON emergency_alerts(status);
CREATE INDEX idx_emergency_alerts_ride ON emergency_alerts(ride_id) WHERE ride_id IS NOT NULL;
CREATE INDEX idx_emergency_alerts_created ON emergency_alerts(created_at DESC);
CREATE INDEX idx_emergency_alerts_active ON emergency_alerts(status, created_at DESC) WHERE status IN ('active', 'responded');

-- ========================================
-- EMERGENCY CONTACTS
-- ========================================

CREATE TABLE IF NOT EXISTS emergency_contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50) NOT NULL,
    email VARCHAR(255),
    relationship VARCHAR(100) NOT NULL, -- 'family', 'friend', 'spouse', 'other'

    is_primary BOOLEAN DEFAULT FALSE,
    notify_on_ride BOOLEAN DEFAULT FALSE, -- Auto-notify when ride starts
    notify_on_sos BOOLEAN DEFAULT TRUE, -- Notify on emergency

    verified BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT emergency_contacts_user_phone_unique UNIQUE(user_id, phone)
);

CREATE INDEX idx_emergency_contacts_user ON emergency_contacts(user_id);
CREATE INDEX idx_emergency_contacts_sos ON emergency_contacts(user_id, notify_on_sos) WHERE notify_on_sos = true;

-- ========================================
-- RIDE SHARE LINKS
-- ========================================

CREATE TABLE IF NOT EXISTS ride_share_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ride_id UUID NOT NULL REFERENCES rides(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    token VARCHAR(100) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,

    -- What to share
    share_location BOOLEAN DEFAULT TRUE,
    share_driver BOOLEAN DEFAULT TRUE,
    share_eta BOOLEAN DEFAULT TRUE,
    share_route BOOLEAN DEFAULT FALSE,

    -- Analytics
    view_count INT DEFAULT 0,
    last_viewed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ride_share_links_token ON ride_share_links(token) WHERE is_active = true;
CREATE INDEX idx_ride_share_links_ride ON ride_share_links(ride_id);
CREATE INDEX idx_ride_share_links_active ON ride_share_links(is_active, expires_at) WHERE is_active = true;

-- Share recipients tracking
CREATE TABLE IF NOT EXISTS ride_share_recipients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    share_link_id UUID NOT NULL REFERENCES ride_share_links(id) ON DELETE CASCADE,
    contact_id UUID REFERENCES emergency_contacts(id),
    name VARCHAR(255),
    phone VARCHAR(50),
    email VARCHAR(255),
    notified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    viewed_at TIMESTAMPTZ
);

CREATE INDEX idx_ride_share_recipients_link ON ride_share_recipients(share_link_id);

-- ========================================
-- SAFETY SETTINGS
-- ========================================

CREATE TABLE IF NOT EXISTS safety_settings (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,

    -- SOS settings
    sos_enabled BOOLEAN DEFAULT TRUE,
    sos_gesture VARCHAR(50) DEFAULT 'shake', -- 'shake', 'power_button', 'volume_buttons'
    sos_auto_call_911 BOOLEAN DEFAULT FALSE,

    -- Share settings
    auto_share_with_contacts BOOLEAN DEFAULT FALSE,
    share_on_ride_start BOOLEAN DEFAULT FALSE,
    share_on_emergency BOOLEAN DEFAULT TRUE,

    -- Check-in settings
    periodic_check_in BOOLEAN DEFAULT FALSE,
    check_in_interval_mins INT DEFAULT 0, -- 0 = disabled

    -- Alert preferences
    route_deviation_alert BOOLEAN DEFAULT TRUE,
    speed_alert_enabled BOOLEAN DEFAULT FALSE,
    long_stop_alert_mins INT DEFAULT 10, -- 0 = disabled

    -- Audio/Video
    allow_ride_recording BOOLEAN DEFAULT FALSE,
    record_audio_on_sos BOOLEAN DEFAULT TRUE,

    -- Night ride protection
    night_ride_protection BOOLEAN DEFAULT FALSE,
    night_protection_start VARCHAR(5) DEFAULT '22:00',
    night_protection_end VARCHAR(5) DEFAULT '06:00',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ========================================
-- SAFETY CHECKS
-- ========================================

CREATE TABLE IF NOT EXISTS safety_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ride_id UUID NOT NULL REFERENCES rides(id),
    user_id UUID NOT NULL REFERENCES users(id),
    type VARCHAR(50) NOT NULL, -- 'periodic', 'route_deviation', 'long_stop', 'speed_alert', 'manual'
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'safe', 'help', 'no_response', 'expired'

    sent_at TIMESTAMPTZ NOT NULL,
    responded_at TIMESTAMPTZ,
    response VARCHAR(255),

    trigger_reason TEXT,

    escalated BOOLEAN DEFAULT FALSE,
    escalated_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_safety_checks_ride ON safety_checks(ride_id);
CREATE INDEX idx_safety_checks_user ON safety_checks(user_id);
CREATE INDEX idx_safety_checks_pending ON safety_checks(status, sent_at) WHERE status = 'pending';

-- ========================================
-- ROUTE DEVIATION ALERTS
-- ========================================

CREATE TABLE IF NOT EXISTS route_deviation_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ride_id UUID NOT NULL REFERENCES rides(id),
    driver_id UUID NOT NULL REFERENCES users(id),

    -- Expected vs actual location
    expected_lat DOUBLE PRECISION NOT NULL,
    expected_lng DOUBLE PRECISION NOT NULL,
    actual_lat DOUBLE PRECISION NOT NULL,
    actual_lng DOUBLE PRECISION NOT NULL,
    deviation_meters INT NOT NULL,

    -- Route data
    expected_route TEXT, -- Polyline
    actual_route TEXT,

    -- Response
    rider_notified BOOLEAN DEFAULT FALSE,
    driver_response VARCHAR(255), -- 'traffic', 'road_closed', 'rider_request', etc.
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_route_deviation_ride ON route_deviation_alerts(ride_id, created_at DESC);
CREATE INDEX idx_route_deviation_driver ON route_deviation_alerts(driver_id);

-- ========================================
-- SPEED ALERTS
-- ========================================

CREATE TABLE IF NOT EXISTS speed_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ride_id UUID NOT NULL REFERENCES rides(id),
    driver_id UUID NOT NULL REFERENCES users(id),

    speed_kmh DOUBLE PRECISION NOT NULL,
    speed_limit INT, -- If known from maps data
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    road_name VARCHAR(255),

    severity VARCHAR(20) NOT NULL, -- 'minor', 'moderate', 'severe'
    duration_seconds INT DEFAULT 0, -- How long over limit

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_speed_alerts_ride ON speed_alerts(ride_id);
CREATE INDEX idx_speed_alerts_driver ON speed_alerts(driver_id, created_at DESC);

-- ========================================
-- TRUSTED/BLOCKED DRIVERS
-- ========================================

CREATE TABLE IF NOT EXISTS trusted_drivers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rider_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    driver_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT trusted_drivers_unique UNIQUE(rider_id, driver_id)
);

CREATE INDEX idx_trusted_drivers_rider ON trusted_drivers(rider_id);

CREATE TABLE IF NOT EXISTS blocked_drivers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rider_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    driver_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT blocked_drivers_unique UNIQUE(rider_id, driver_id)
);

CREATE INDEX idx_blocked_drivers_rider ON blocked_drivers(rider_id);

-- ========================================
-- SAFETY INCIDENT REPORTS
-- ========================================

CREATE TABLE IF NOT EXISTS safety_incident_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id UUID NOT NULL REFERENCES users(id),
    reported_user_id UUID REFERENCES users(id),
    ride_id UUID REFERENCES rides(id),

    category VARCHAR(100) NOT NULL, -- 'harassment', 'unsafe_driving', 'assault', 'theft', 'other'
    severity VARCHAR(20) NOT NULL, -- 'low', 'medium', 'high', 'critical'
    description TEXT NOT NULL,

    -- Evidence
    photos TEXT[], -- Array of URLs
    audio_url TEXT,
    video_url TEXT,

    -- Admin handling
    status VARCHAR(50) DEFAULT 'pending', -- 'pending', 'investigating', 'resolved', 'dismissed'
    assigned_to UUID REFERENCES users(id),
    resolution TEXT,
    action_taken TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    investigated_at TIMESTAMPTZ,
    resolved_at TIMESTAMPTZ
);

CREATE INDEX idx_safety_incidents_reporter ON safety_incident_reports(reporter_id);
CREATE INDEX idx_safety_incidents_reported ON safety_incident_reports(reported_user_id) WHERE reported_user_id IS NOT NULL;
CREATE INDEX idx_safety_incidents_status ON safety_incident_reports(status);
CREATE INDEX idx_safety_incidents_severity ON safety_incident_reports(severity, status) WHERE status IN ('pending', 'investigating');
CREATE INDEX idx_safety_incidents_created ON safety_incident_reports(created_at DESC);

-- ========================================
-- EMERGENCY ALERT CONTACTS NOTIFIED
-- ========================================

CREATE TABLE IF NOT EXISTS emergency_alert_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_id UUID NOT NULL REFERENCES emergency_alerts(id) ON DELETE CASCADE,
    contact_id UUID REFERENCES emergency_contacts(id),
    phone VARCHAR(50),
    notification_type VARCHAR(20) NOT NULL, -- 'sms', 'email', 'push'
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at TIMESTAMPTZ,
    read_at TIMESTAMPTZ,
    failed BOOLEAN DEFAULT FALSE,
    failure_reason TEXT
);

CREATE INDEX idx_emergency_notifications_alert ON emergency_alert_notifications(alert_id);

-- ========================================
-- TRIGGERS
-- ========================================

-- Update updated_at for emergency_alerts
CREATE OR REPLACE FUNCTION update_emergency_alert_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_emergency_alert_timestamp
    BEFORE UPDATE ON emergency_alerts
    FOR EACH ROW
    EXECUTE FUNCTION update_emergency_alert_timestamp();

-- Update updated_at for emergency_contacts
CREATE TRIGGER trigger_update_emergency_contact_timestamp
    BEFORE UPDATE ON emergency_contacts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Update updated_at for ride_share_links
CREATE TRIGGER trigger_update_ride_share_link_timestamp
    BEFORE UPDATE ON ride_share_links
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Update updated_at for safety_settings
CREATE TRIGGER trigger_update_safety_settings_timestamp
    BEFORE UPDATE ON safety_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ========================================
-- COMMENTS FOR DOCUMENTATION
-- ========================================

COMMENT ON TABLE emergency_alerts IS 'SOS emergency alerts triggered by users';
COMMENT ON TABLE emergency_contacts IS 'User''s emergency contacts to notify in emergencies';
COMMENT ON TABLE ride_share_links IS 'Shareable links for live ride tracking';
COMMENT ON TABLE safety_settings IS 'User safety preferences and settings';
COMMENT ON TABLE safety_checks IS 'Periodic safety check-ins during rides';
COMMENT ON TABLE route_deviation_alerts IS 'Alerts when driver deviates from expected route';
COMMENT ON TABLE speed_alerts IS 'Alerts when driver exceeds speed limits';
COMMENT ON TABLE trusted_drivers IS 'Drivers marked as trusted by riders';
COMMENT ON TABLE blocked_drivers IS 'Drivers blocked by riders';
COMMENT ON TABLE safety_incident_reports IS 'Safety incident reports from users';
