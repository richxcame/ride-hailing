-- Multi-Country Foundation Migration
-- Enables hierarchical pricing by country → region → city → zone

-- Enable PostGIS extension for geospatial boundaries
CREATE EXTENSION IF NOT EXISTS postgis;

-- ========================================
-- CURRENCIES
-- ========================================

CREATE TABLE currencies (
    code VARCHAR(3) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    symbol VARCHAR(10) NOT NULL,
    decimal_places INTEGER NOT NULL DEFAULT 2,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed common currencies
INSERT INTO currencies (code, name, symbol, decimal_places) VALUES
    ('USD', 'US Dollar', '$', 2),
    ('EUR', 'Euro', '€', 2),
    ('GBP', 'British Pound', '£', 2),
    ('TMT', 'Turkmenistan Manat', 'm', 2),
    ('UZS', 'Uzbekistani Som', 'so''m', 0),
    ('KZT', 'Kazakhstani Tenge', '₸', 0),
    ('RUB', 'Russian Ruble', '₽', 2),
    ('TRY', 'Turkish Lira', '₺', 2),
    ('AED', 'UAE Dirham', 'د.إ', 2),
    ('INR', 'Indian Rupee', '₹', 2),
    ('CNY', 'Chinese Yuan', '¥', 2),
    ('JPY', 'Japanese Yen', '¥', 0),
    ('KRW', 'South Korean Won', '₩', 0),
    ('BRL', 'Brazilian Real', 'R$', 2),
    ('MXN', 'Mexican Peso', '$', 2),
    ('NGN', 'Nigerian Naira', '₦', 2),
    ('KES', 'Kenyan Shilling', 'KSh', 2),
    ('ZAR', 'South African Rand', 'R', 2),
    ('EGP', 'Egyptian Pound', 'E£', 2),
    ('PKR', 'Pakistani Rupee', '₨', 0);

-- Exchange rates (base currency: USD)
CREATE TABLE exchange_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_currency VARCHAR(3) NOT NULL REFERENCES currencies(code),
    to_currency VARCHAR(3) NOT NULL REFERENCES currencies(code),
    rate DECIMAL(18,8) NOT NULL,
    inverse_rate DECIMAL(18,8) NOT NULL,
    source VARCHAR(50) NOT NULL DEFAULT 'manual',
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_different_currencies CHECK (from_currency != to_currency)
);

CREATE INDEX idx_exchange_rates_lookup ON exchange_rates(from_currency, to_currency, valid_until DESC);
CREATE INDEX idx_exchange_rates_valid ON exchange_rates(valid_until DESC);

-- ========================================
-- GEOGRAPHIC HIERARCHY
-- ========================================

-- Countries
CREATE TABLE countries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code CHAR(2) NOT NULL UNIQUE,
    code3 CHAR(3) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    native_name VARCHAR(100),
    currency_code VARCHAR(3) NOT NULL REFERENCES currencies(code),
    default_language VARCHAR(10) NOT NULL DEFAULT 'en',
    timezone VARCHAR(50) NOT NULL,
    phone_prefix VARCHAR(10) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT false,
    launched_at TIMESTAMPTZ,
    regulations JSONB NOT NULL DEFAULT '{}',
    payment_methods JSONB NOT NULL DEFAULT '["stripe", "wallet", "cash"]',
    required_driver_documents JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_countries_active ON countries(is_active) WHERE is_active = true;
CREATE INDEX idx_countries_code ON countries(code);

-- Regions (states/provinces)
CREATE TABLE regions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    country_id UUID NOT NULL REFERENCES countries(id) ON DELETE CASCADE,
    code VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    native_name VARCHAR(100),
    timezone VARCHAR(50),
    is_active BOOLEAN NOT NULL DEFAULT false,
    launched_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(country_id, code)
);

CREATE INDEX idx_regions_country ON regions(country_id);
CREATE INDEX idx_regions_active ON regions(country_id, is_active) WHERE is_active = true;

-- Cities
CREATE TABLE cities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    region_id UUID NOT NULL REFERENCES regions(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    native_name VARCHAR(100),
    timezone VARCHAR(50),
    center_latitude DECIMAL(10,8) NOT NULL,
    center_longitude DECIMAL(11,8) NOT NULL,
    boundary GEOMETRY(POLYGON, 4326),
    population INTEGER,
    is_active BOOLEAN NOT NULL DEFAULT false,
    launched_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cities_region ON cities(region_id);
CREATE INDEX idx_cities_active ON cities(region_id, is_active) WHERE is_active = true;
CREATE INDEX idx_cities_boundary ON cities USING GIST(boundary);
CREATE INDEX idx_cities_center ON cities(center_latitude, center_longitude);

-- Pricing zones (airports, downtown, event venues, borders)
CREATE TABLE pricing_zones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    city_id UUID NOT NULL REFERENCES cities(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    zone_type VARCHAR(50) NOT NULL,
    boundary GEOMETRY(POLYGON, 4326) NOT NULL,
    center_latitude DECIMAL(10,8) NOT NULL,
    center_longitude DECIMAL(11,8) NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN pricing_zones.zone_type IS 'airport, downtown, transit_hub, event_venue, border_crossing, toll_zone';
COMMENT ON COLUMN pricing_zones.priority IS 'Higher priority zones take precedence when overlapping';

CREATE INDEX idx_pricing_zones_city ON pricing_zones(city_id);
CREATE INDEX idx_pricing_zones_boundary ON pricing_zones USING GIST(boundary);
CREATE INDEX idx_pricing_zones_type ON pricing_zones(zone_type);
CREATE INDEX idx_pricing_zones_active ON pricing_zones(city_id, is_active) WHERE is_active = true;

-- ========================================
-- PRICING CONFIGURATION
-- ========================================

-- Pricing config versions (for audit trail and A/B testing)
CREATE TABLE pricing_config_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version_number INTEGER NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    ab_test_percentage INTEGER,
    effective_from TIMESTAMPTZ,
    effective_until TIMESTAMPTZ,
    created_by UUID REFERENCES users(id),
    approved_by UUID REFERENCES users(id),
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_status CHECK (status IN ('draft', 'active', 'archived', 'ab_test')),
    CONSTRAINT chk_ab_percentage CHECK (ab_test_percentage IS NULL OR (ab_test_percentage >= 0 AND ab_test_percentage <= 100))
);

CREATE INDEX idx_pricing_versions_status ON pricing_config_versions(status);
CREATE INDEX idx_pricing_versions_active ON pricing_config_versions(status) WHERE status = 'active';

-- Hierarchical pricing configurations
CREATE TABLE pricing_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version_id UUID NOT NULL REFERENCES pricing_config_versions(id) ON DELETE CASCADE,

    -- Scope (only one should be set for hierarchy level, all NULL = global)
    country_id UUID REFERENCES countries(id) ON DELETE CASCADE,
    region_id UUID REFERENCES regions(id) ON DELETE CASCADE,
    city_id UUID REFERENCES cities(id) ON DELETE CASCADE,
    zone_id UUID REFERENCES pricing_zones(id) ON DELETE CASCADE,

    -- Ride type (NULL = applies to all)
    ride_type_id UUID REFERENCES ride_types(id) ON DELETE CASCADE,

    -- Core pricing (NULL = inherit from parent level)
    base_fare DECIMAL(10,2),
    per_km_rate DECIMAL(10,4),
    per_minute_rate DECIMAL(10,4),
    minimum_fare DECIMAL(10,2),
    booking_fee DECIMAL(10,2),

    -- Commission
    platform_commission_pct DECIMAL(5,2),
    driver_incentive_pct DECIMAL(5,2),

    -- Surge limits
    surge_min_multiplier DECIMAL(3,2) DEFAULT 1.00,
    surge_max_multiplier DECIMAL(3,2) DEFAULT 5.00,

    -- Tax
    tax_rate_pct DECIMAL(5,2) DEFAULT 0.00,
    tax_inclusive BOOLEAN DEFAULT false,

    -- Cancellation fees (tiered)
    cancellation_fees JSONB DEFAULT '[]',

    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure hierarchical integrity
    CONSTRAINT chk_hierarchy CHECK (
        (zone_id IS NOT NULL AND city_id IS NULL AND region_id IS NULL AND country_id IS NULL) OR
        (zone_id IS NULL AND city_id IS NOT NULL AND region_id IS NULL AND country_id IS NULL) OR
        (zone_id IS NULL AND city_id IS NULL AND region_id IS NOT NULL AND country_id IS NULL) OR
        (zone_id IS NULL AND city_id IS NULL AND region_id IS NULL AND country_id IS NOT NULL) OR
        (zone_id IS NULL AND city_id IS NULL AND region_id IS NULL AND country_id IS NULL)
    )
);

COMMENT ON COLUMN pricing_configs.cancellation_fees IS 'Array of {after_minutes: int, fee: decimal, fee_type: "fixed"|"percentage"}';

CREATE INDEX idx_pricing_configs_version ON pricing_configs(version_id);
CREATE INDEX idx_pricing_configs_country ON pricing_configs(version_id, country_id) WHERE country_id IS NOT NULL;
CREATE INDEX idx_pricing_configs_region ON pricing_configs(version_id, region_id) WHERE region_id IS NOT NULL;
CREATE INDEX idx_pricing_configs_city ON pricing_configs(version_id, city_id) WHERE city_id IS NOT NULL;
CREATE INDEX idx_pricing_configs_zone ON pricing_configs(version_id, zone_id) WHERE zone_id IS NOT NULL;
CREATE INDEX idx_pricing_configs_global ON pricing_configs(version_id)
    WHERE country_id IS NULL AND region_id IS NULL AND city_id IS NULL AND zone_id IS NULL;
CREATE INDEX idx_pricing_configs_ride_type ON pricing_configs(ride_type_id) WHERE ride_type_id IS NOT NULL;

-- Zone-specific fees (airport fees, tolls, etc.)
CREATE TABLE zone_fees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    zone_id UUID NOT NULL REFERENCES pricing_zones(id) ON DELETE CASCADE,
    version_id UUID NOT NULL REFERENCES pricing_config_versions(id) ON DELETE CASCADE,
    fee_type VARCHAR(50) NOT NULL,
    ride_type_id UUID REFERENCES ride_types(id),
    amount DECIMAL(10,2) NOT NULL,
    is_percentage BOOLEAN NOT NULL DEFAULT false,
    applies_pickup BOOLEAN NOT NULL DEFAULT true,
    applies_dropoff BOOLEAN NOT NULL DEFAULT true,
    schedule JSONB,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN zone_fees.fee_type IS 'pickup_fee, dropoff_fee, toll, border_crossing, event_surcharge';
COMMENT ON COLUMN zone_fees.schedule IS '{"days": [0-6], "start_time": "HH:MM", "end_time": "HH:MM"}';

CREATE INDEX idx_zone_fees_zone ON zone_fees(zone_id);
CREATE INDEX idx_zone_fees_version ON zone_fees(version_id);
CREATE INDEX idx_zone_fees_active ON zone_fees(zone_id, is_active) WHERE is_active = true;

-- ========================================
-- DYNAMIC PRICING RULES
-- ========================================

-- Time-based multipliers
CREATE TABLE time_multipliers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version_id UUID NOT NULL REFERENCES pricing_config_versions(id) ON DELETE CASCADE,
    country_id UUID REFERENCES countries(id) ON DELETE CASCADE,
    region_id UUID REFERENCES regions(id) ON DELETE CASCADE,
    city_id UUID REFERENCES cities(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    days_of_week INTEGER[] NOT NULL DEFAULT '{0,1,2,3,4,5,6}',
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    multiplier DECIMAL(3,2) NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN time_multipliers.days_of_week IS '0=Sunday, 6=Saturday';

CREATE INDEX idx_time_multipliers_version ON time_multipliers(version_id);
CREATE INDEX idx_time_multipliers_scope ON time_multipliers(country_id, region_id, city_id);
CREATE INDEX idx_time_multipliers_active ON time_multipliers(is_active) WHERE is_active = true;

-- Weather-based multipliers
CREATE TABLE weather_multipliers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version_id UUID NOT NULL REFERENCES pricing_config_versions(id) ON DELETE CASCADE,
    country_id UUID REFERENCES countries(id) ON DELETE CASCADE,
    region_id UUID REFERENCES regions(id) ON DELETE CASCADE,
    city_id UUID REFERENCES cities(id) ON DELETE CASCADE,
    weather_condition VARCHAR(50) NOT NULL,
    multiplier DECIMAL(3,2) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN weather_multipliers.weather_condition IS 'clear, cloudy, rain, heavy_rain, snow, storm, extreme_heat, fog';

CREATE INDEX idx_weather_multipliers_version ON weather_multipliers(version_id);
CREATE INDEX idx_weather_multipliers_condition ON weather_multipliers(weather_condition);

-- Event-based multipliers
CREATE TABLE event_multipliers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version_id UUID NOT NULL REFERENCES pricing_config_versions(id) ON DELETE CASCADE,
    zone_id UUID REFERENCES pricing_zones(id) ON DELETE CASCADE,
    city_id UUID REFERENCES cities(id) ON DELETE CASCADE,
    event_name VARCHAR(200) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    pre_event_minutes INTEGER NOT NULL DEFAULT 120,
    post_event_minutes INTEGER NOT NULL DEFAULT 120,
    multiplier DECIMAL(3,2) NOT NULL,
    expected_demand_increase INTEGER,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN event_multipliers.event_type IS 'sports, concert, conference, holiday, festival, protest, other';

CREATE INDEX idx_event_multipliers_dates ON event_multipliers(starts_at, ends_at);
CREATE INDEX idx_event_multipliers_zone ON event_multipliers(zone_id) WHERE zone_id IS NOT NULL;
CREATE INDEX idx_event_multipliers_city ON event_multipliers(city_id) WHERE city_id IS NOT NULL;
CREATE INDEX idx_event_multipliers_active ON event_multipliers(is_active, starts_at, ends_at) WHERE is_active = true;

-- Demand/supply surge thresholds
CREATE TABLE surge_thresholds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version_id UUID NOT NULL REFERENCES pricing_config_versions(id) ON DELETE CASCADE,
    country_id UUID REFERENCES countries(id) ON DELETE CASCADE,
    region_id UUID REFERENCES regions(id) ON DELETE CASCADE,
    city_id UUID REFERENCES cities(id) ON DELETE CASCADE,
    demand_supply_ratio_min DECIMAL(5,2) NOT NULL,
    demand_supply_ratio_max DECIMAL(5,2),
    multiplier DECIMAL(3,2) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN surge_thresholds.demand_supply_ratio_min IS 'demand/supply ratio minimum (e.g., 1.5 = 50% more demand than supply)';

CREATE INDEX idx_surge_thresholds_version ON surge_thresholds(version_id);
CREATE INDEX idx_surge_thresholds_scope ON surge_thresholds(country_id, region_id, city_id);

-- ========================================
-- PAYMENT METHODS PER COUNTRY
-- ========================================

CREATE TABLE country_payment_methods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    country_id UUID NOT NULL REFERENCES countries(id) ON DELETE CASCADE,
    method_type VARCHAR(50) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    display_icon VARCHAR(255),
    config JSONB NOT NULL DEFAULT '{}',
    supported_currencies VARCHAR(3)[] NOT NULL DEFAULT '{}',
    min_amount DECIMAL(10,2),
    max_amount DECIMAL(10,2),
    is_active BOOLEAN NOT NULL DEFAULT true,
    priority INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN country_payment_methods.method_type IS 'card, wallet, cash, bank_transfer, mobile_money';
COMMENT ON COLUMN country_payment_methods.provider IS 'stripe, sepa, alipay, mpesa, paytm, razorpay';

CREATE INDEX idx_country_payment_methods_country ON country_payment_methods(country_id);
CREATE INDEX idx_country_payment_methods_active ON country_payment_methods(country_id, is_active) WHERE is_active = true;

-- ========================================
-- AUDIT LOGGING
-- ========================================

CREATE TABLE pricing_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id UUID NOT NULL REFERENCES users(id),
    action VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    old_values JSONB,
    new_values JSONB,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pricing_audit_entity ON pricing_audit_logs(entity_type, entity_id);
CREATE INDEX idx_pricing_audit_admin ON pricing_audit_logs(admin_id);
CREATE INDEX idx_pricing_audit_created ON pricing_audit_logs(created_at DESC);

-- ========================================
-- PRICING HISTORY (for analytics)
-- ========================================

CREATE TABLE pricing_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ride_id UUID NOT NULL REFERENCES rides(id) ON DELETE CASCADE,
    country_id UUID REFERENCES countries(id),
    region_id UUID REFERENCES regions(id),
    city_id UUID REFERENCES cities(id),
    pickup_zone_id UUID REFERENCES pricing_zones(id),
    dropoff_zone_id UUID REFERENCES pricing_zones(id),
    version_id UUID REFERENCES pricing_config_versions(id),
    currency_code VARCHAR(3) NOT NULL,
    base_fare DECIMAL(10,2) NOT NULL,
    per_km_rate DECIMAL(10,4) NOT NULL,
    per_minute_rate DECIMAL(10,4) NOT NULL,
    distance_charge DECIMAL(10,2) NOT NULL,
    time_charge DECIMAL(10,2) NOT NULL,
    booking_fee DECIMAL(10,2) NOT NULL DEFAULT 0,
    zone_fees_total DECIMAL(10,2) NOT NULL DEFAULT 0,
    zone_fees_breakdown JSONB DEFAULT '[]',
    time_multiplier DECIMAL(3,2) NOT NULL DEFAULT 1.00,
    weather_multiplier DECIMAL(3,2) NOT NULL DEFAULT 1.00,
    event_multiplier DECIMAL(3,2) NOT NULL DEFAULT 1.00,
    surge_multiplier DECIMAL(3,2) NOT NULL DEFAULT 1.00,
    total_multiplier DECIMAL(3,2) NOT NULL DEFAULT 1.00,
    subtotal DECIMAL(10,2) NOT NULL,
    tax_rate_pct DECIMAL(5,2) NOT NULL DEFAULT 0,
    tax_amount DECIMAL(10,2) NOT NULL DEFAULT 0,
    total_fare DECIMAL(10,2) NOT NULL,
    platform_commission_pct DECIMAL(5,2) NOT NULL,
    platform_commission DECIMAL(10,2) NOT NULL,
    driver_earnings DECIMAL(10,2) NOT NULL,
    was_negotiated BOOLEAN NOT NULL DEFAULT false,
    negotiated_fare DECIMAL(10,2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pricing_history_ride ON pricing_history(ride_id);
CREATE INDEX idx_pricing_history_country ON pricing_history(country_id, created_at DESC);
CREATE INDEX idx_pricing_history_city ON pricing_history(city_id, created_at DESC);
CREATE INDEX idx_pricing_history_created ON pricing_history(created_at DESC);

-- ========================================
-- EXTEND EXISTING TABLES
-- ========================================

-- Add region context to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS country_id UUID REFERENCES countries(id);
ALTER TABLE users ADD COLUMN IF NOT EXISTS home_region_id UUID REFERENCES regions(id);
ALTER TABLE users ADD COLUMN IF NOT EXISTS preferred_language VARCHAR(10);
ALTER TABLE users ADD COLUMN IF NOT EXISTS preferred_currency VARCHAR(3) REFERENCES currencies(code);

CREATE INDEX idx_users_country ON users(country_id) WHERE country_id IS NOT NULL;

-- Add multi-currency support to payments
ALTER TABLE payments ADD COLUMN IF NOT EXISTS region_id UUID REFERENCES regions(id);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS original_currency VARCHAR(3);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS original_amount DECIMAL(10,2);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS settlement_currency VARCHAR(3);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS settlement_amount DECIMAL(10,2);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS exchange_rate DECIMAL(18,8);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS exchange_rate_id UUID REFERENCES exchange_rates(id);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS country_payment_method_id UUID REFERENCES country_payment_methods(id);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS tax_amount DECIMAL(10,2) DEFAULT 0;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS tax_rate_pct DECIMAL(5,2) DEFAULT 0;

-- Add region context to rides
ALTER TABLE rides ADD COLUMN IF NOT EXISTS country_id UUID REFERENCES countries(id);
ALTER TABLE rides ADD COLUMN IF NOT EXISTS region_id UUID REFERENCES regions(id);
ALTER TABLE rides ADD COLUMN IF NOT EXISTS city_id UUID REFERENCES cities(id);
ALTER TABLE rides ADD COLUMN IF NOT EXISTS pickup_zone_id UUID REFERENCES pricing_zones(id);
ALTER TABLE rides ADD COLUMN IF NOT EXISTS dropoff_zone_id UUID REFERENCES pricing_zones(id);
ALTER TABLE rides ADD COLUMN IF NOT EXISTS currency_code VARCHAR(3) DEFAULT 'USD';
ALTER TABLE rides ADD COLUMN IF NOT EXISTS pricing_version_id UUID REFERENCES pricing_config_versions(id);
ALTER TABLE rides ADD COLUMN IF NOT EXISTS was_negotiated BOOLEAN DEFAULT false;
ALTER TABLE rides ADD COLUMN IF NOT EXISTS negotiation_session_id UUID;

CREATE INDEX idx_rides_country ON rides(country_id) WHERE country_id IS NOT NULL;
CREATE INDEX idx_rides_city ON rides(city_id) WHERE city_id IS NOT NULL;

-- Driver operating regions
CREATE TABLE driver_regions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
    region_id UUID NOT NULL REFERENCES regions(id) ON DELETE CASCADE,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(driver_id, region_id)
);

CREATE INDEX idx_driver_regions_driver ON driver_regions(driver_id);
CREATE INDEX idx_driver_regions_region ON driver_regions(region_id);
CREATE INDEX idx_driver_regions_primary ON driver_regions(driver_id) WHERE is_primary = true;

-- ========================================
-- TRIGGERS
-- ========================================

CREATE TRIGGER update_countries_updated_at BEFORE UPDATE ON countries
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_regions_updated_at BEFORE UPDATE ON regions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_cities_updated_at BEFORE UPDATE ON cities
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_pricing_zones_updated_at BEFORE UPDATE ON pricing_zones
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_pricing_config_versions_updated_at BEFORE UPDATE ON pricing_config_versions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_pricing_configs_updated_at BEFORE UPDATE ON pricing_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_zone_fees_updated_at BEFORE UPDATE ON zone_fees
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_time_multipliers_updated_at BEFORE UPDATE ON time_multipliers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_country_payment_methods_updated_at BEFORE UPDATE ON country_payment_methods
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ========================================
-- SEED INITIAL DATA
-- ========================================

-- Create initial pricing config version
INSERT INTO pricing_config_versions (id, version_number, name, description, status, effective_from)
VALUES (
    'a0000000-0000-0000-0000-000000000001',
    1,
    'Initial Global Pricing',
    'Default pricing configuration for all regions',
    'active',
    NOW()
);

-- Create global default pricing config
INSERT INTO pricing_configs (
    version_id,
    base_fare,
    per_km_rate,
    per_minute_rate,
    minimum_fare,
    booking_fee,
    platform_commission_pct,
    driver_incentive_pct,
    surge_min_multiplier,
    surge_max_multiplier,
    tax_rate_pct,
    tax_inclusive,
    cancellation_fees
) VALUES (
    'a0000000-0000-0000-0000-000000000001',
    3.00,
    1.50,
    0.25,
    5.00,
    1.00,
    20.00,
    0.00,
    1.00,
    5.00,
    0.00,
    false,
    '[{"after_minutes": 0, "fee": 0, "fee_type": "fixed"}, {"after_minutes": 2, "fee": 5, "fee_type": "fixed"}, {"after_minutes": 5, "fee": 10, "fee_type": "fixed"}]'
);

-- Create default time multipliers
INSERT INTO time_multipliers (version_id, name, days_of_week, start_time, end_time, multiplier, priority) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'Morning Rush', '{1,2,3,4,5}', '07:00', '09:00', 1.50, 10),
    ('a0000000-0000-0000-0000-000000000001', 'Evening Rush', '{1,2,3,4,5}', '17:00', '20:00', 1.50, 10),
    ('a0000000-0000-0000-0000-000000000001', 'Late Night', '{0,1,2,3,4,5,6}', '23:00', '05:00', 1.30, 5),
    ('a0000000-0000-0000-0000-000000000001', 'Weekend Night', '{5,6}', '22:00', '03:00', 1.40, 15);

-- Create default weather multipliers
INSERT INTO weather_multipliers (version_id, weather_condition, multiplier) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'rain', 1.20),
    ('a0000000-0000-0000-0000-000000000001', 'heavy_rain', 1.40),
    ('a0000000-0000-0000-0000-000000000001', 'snow', 1.50),
    ('a0000000-0000-0000-0000-000000000001', 'storm', 1.60),
    ('a0000000-0000-0000-0000-000000000001', 'extreme_heat', 1.10),
    ('a0000000-0000-0000-0000-000000000001', 'fog', 1.15);

-- Create default surge thresholds
INSERT INTO surge_thresholds (version_id, demand_supply_ratio_min, demand_supply_ratio_max, multiplier) VALUES
    ('a0000000-0000-0000-0000-000000000001', 1.00, 1.50, 1.00),
    ('a0000000-0000-0000-0000-000000000001', 1.50, 2.00, 1.25),
    ('a0000000-0000-0000-0000-000000000001', 2.00, 3.00, 1.50),
    ('a0000000-0000-0000-0000-000000000001', 3.00, 4.00, 2.00),
    ('a0000000-0000-0000-0000-000000000001', 4.00, 5.00, 2.50),
    ('a0000000-0000-0000-0000-000000000001', 5.00, NULL, 3.00);
