-- Promo Codes table
CREATE TABLE promo_codes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    discount_type VARCHAR(20) NOT NULL CHECK (discount_type IN ('percentage', 'fixed_amount')),
    discount_value DECIMAL(10,2) NOT NULL CHECK (discount_value > 0),
    max_discount_amount DECIMAL(10,2), -- Maximum discount for percentage type
    min_ride_amount DECIMAL(10,2), -- Minimum ride amount to apply promo
    max_uses INTEGER, -- NULL means unlimited
    total_uses INTEGER DEFAULT 0,
    uses_per_user INTEGER DEFAULT 1, -- How many times a user can use this code
    valid_from TIMESTAMP WITH TIME ZONE NOT NULL,
    valid_until TIMESTAMP WITH TIME ZONE NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Promo Code Uses tracking table
CREATE TABLE promo_code_uses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    promo_code_id UUID NOT NULL REFERENCES promo_codes(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ride_id UUID NOT NULL REFERENCES rides(id) ON DELETE CASCADE,
    discount_amount DECIMAL(10,2) NOT NULL,
    original_amount DECIMAL(10,2) NOT NULL,
    final_amount DECIMAL(10,2) NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(promo_code_id, ride_id)
);

-- Referral Codes table
CREATE TABLE referral_codes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    code VARCHAR(20) UNIQUE NOT NULL,
    total_referrals INTEGER DEFAULT 0,
    total_earnings DECIMAL(10,2) DEFAULT 0.00,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Referrals tracking table
CREATE TABLE referrals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    referrer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    referred_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    referral_code_id UUID NOT NULL REFERENCES referral_codes(id) ON DELETE CASCADE,
    referrer_bonus DECIMAL(10,2) DEFAULT 0.00,
    referred_bonus DECIMAL(10,2) DEFAULT 0.00,
    referrer_bonus_applied BOOLEAN DEFAULT false,
    referred_bonus_applied BOOLEAN DEFAULT false,
    referred_first_ride_id UUID REFERENCES rides(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Ride Types table
CREATE TABLE ride_types (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    base_fare DECIMAL(10,2) NOT NULL,
    per_km_rate DECIMAL(10,2) NOT NULL,
    per_minute_rate DECIMAL(10,2) NOT NULL,
    minimum_fare DECIMAL(10,2) NOT NULL,
    capacity INTEGER NOT NULL DEFAULT 4,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Add ride_type_id to rides table
ALTER TABLE rides ADD COLUMN ride_type_id UUID REFERENCES ride_types(id);
ALTER TABLE rides ADD COLUMN promo_code_id UUID REFERENCES promo_codes(id);
ALTER TABLE rides ADD COLUMN discount_amount DECIMAL(10,2) DEFAULT 0.00;

-- Add ride_type_id to drivers table (what types can they accept)
ALTER TABLE drivers ADD COLUMN ride_types UUID[] DEFAULT ARRAY[]::UUID[];

-- Indexes for promo_codes
CREATE INDEX idx_promo_codes_code ON promo_codes(code);
CREATE INDEX idx_promo_codes_is_active ON promo_codes(is_active);
CREATE INDEX idx_promo_codes_valid_dates ON promo_codes(valid_from, valid_until);

-- Indexes for promo_code_uses
CREATE INDEX idx_promo_code_uses_promo_code_id ON promo_code_uses(promo_code_id);
CREATE INDEX idx_promo_code_uses_user_id ON promo_code_uses(user_id);
CREATE INDEX idx_promo_code_uses_ride_id ON promo_code_uses(ride_id);
CREATE INDEX idx_promo_code_uses_used_at ON promo_code_uses(used_at);

-- Indexes for referral_codes
CREATE INDEX idx_referral_codes_user_id ON referral_codes(user_id);
CREATE INDEX idx_referral_codes_code ON referral_codes(code);

-- Indexes for referrals
CREATE INDEX idx_referrals_referrer_id ON referrals(referrer_id);
CREATE INDEX idx_referrals_referred_id ON referrals(referred_id);
CREATE INDEX idx_referrals_referral_code_id ON referrals(referral_code_id);

-- Indexes for ride_types
CREATE INDEX idx_ride_types_is_active ON ride_types(is_active);

-- Add triggers for updated_at
CREATE TRIGGER update_promo_codes_updated_at BEFORE UPDATE ON promo_codes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_ride_types_updated_at BEFORE UPDATE ON ride_types
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert default ride types
INSERT INTO ride_types (name, description, base_fare, per_km_rate, per_minute_rate, minimum_fare, capacity, is_active) VALUES
('Economy', 'Standard rides for everyday travel', 3.00, 1.50, 0.25, 5.00, 4, true),
('Premium', 'Luxury rides with high-end vehicles', 5.00, 2.50, 0.40, 10.00, 4, true),
('XL', 'Extra-large vehicles for groups or luggage', 4.00, 2.00, 0.30, 8.00, 6, true);
