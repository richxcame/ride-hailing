-- =============================================
-- Migration: Loyalty, Gamification & Background Checks
-- Features:
--   1. Rider Loyalty Tiers & Points
--   2. Driver Incentives (Quests, Challenges, Streaks)
--   3. Background Check Integration
--   4. Driver Selfie Verification
-- =============================================

-- =============================================
-- PART 1: RIDER LOYALTY SYSTEM
-- =============================================

-- Loyalty tiers configuration
CREATE TABLE IF NOT EXISTS loyalty_tiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,  -- Bronze, Silver, Gold, Platinum, Diamond
    display_name VARCHAR(100) NOT NULL,
    min_points INTEGER NOT NULL DEFAULT 0,
    max_points INTEGER,  -- NULL for highest tier
    multiplier DECIMAL(3,2) NOT NULL DEFAULT 1.0,  -- Points earning multiplier
    discount_percent DECIMAL(5,2) DEFAULT 0,  -- Ride discount
    priority_support BOOLEAN DEFAULT FALSE,
    free_cancellations INTEGER DEFAULT 0,  -- Per month
    free_upgrades INTEGER DEFAULT 0,  -- Per month
    airport_lounge_access BOOLEAN DEFAULT FALSE,
    dedicated_support_line BOOLEAN DEFAULT FALSE,
    icon_url VARCHAR(500),
    color_hex VARCHAR(7),
    benefits JSONB DEFAULT '[]',  -- Additional benefits list
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Rider loyalty accounts
CREATE TABLE IF NOT EXISTS rider_loyalty (
    rider_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    current_tier_id UUID REFERENCES loyalty_tiers(id),
    total_points INTEGER DEFAULT 0,
    available_points INTEGER DEFAULT 0,
    lifetime_points INTEGER DEFAULT 0,
    tier_points INTEGER DEFAULT 0,  -- Points for current tier period
    tier_period_start DATE DEFAULT CURRENT_DATE,
    tier_period_end DATE DEFAULT (CURRENT_DATE + INTERVAL '1 year'),
    free_cancellations_used INTEGER DEFAULT 0,
    free_upgrades_used INTEGER DEFAULT 0,
    streak_days INTEGER DEFAULT 0,
    last_ride_date DATE,
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    tier_upgraded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Points transactions
CREATE TABLE IF NOT EXISTS loyalty_points_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rider_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    transaction_type VARCHAR(50) NOT NULL,  -- earn, redeem, expire, bonus, adjustment
    points INTEGER NOT NULL,  -- Positive for earn, negative for redeem/expire
    balance_after INTEGER NOT NULL,
    source VARCHAR(50) NOT NULL,  -- ride, referral, promo, challenge, birthday, streak
    source_id UUID,  -- Reference to ride, promo, etc.
    description TEXT,
    expires_at TIMESTAMPTZ,  -- When points expire (typically 1 year)
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Rider challenges (weekly/monthly goals)
CREATE TABLE IF NOT EXISTS rider_challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    challenge_type VARCHAR(50) NOT NULL,  -- rides_count, spend_amount, streak, referral
    target_value INTEGER NOT NULL,
    reward_points INTEGER NOT NULL,
    reward_type VARCHAR(50) DEFAULT 'points',  -- points, discount, free_ride, upgrade
    reward_value DECIMAL(10,2),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    tier_restriction UUID REFERENCES loyalty_tiers(id),  -- NULL = all tiers
    max_participants INTEGER,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Rider challenge progress
CREATE TABLE IF NOT EXISTS rider_challenge_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rider_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    challenge_id UUID NOT NULL REFERENCES rider_challenges(id) ON DELETE CASCADE,
    current_value INTEGER DEFAULT 0,
    completed BOOLEAN DEFAULT FALSE,
    completed_at TIMESTAMPTZ,
    reward_claimed BOOLEAN DEFAULT FALSE,
    reward_claimed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(rider_id, challenge_id)
);

-- Points redemption catalog
CREATE TABLE IF NOT EXISTS loyalty_rewards_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    reward_type VARCHAR(50) NOT NULL,  -- discount, free_ride, upgrade, partner_voucher
    points_required INTEGER NOT NULL,
    value DECIMAL(10,2),  -- Monetary value
    partner_name VARCHAR(100),  -- For partner rewards
    partner_logo_url VARCHAR(500),
    valid_days INTEGER DEFAULT 30,  -- Days valid after redemption
    max_redemptions_per_user INTEGER,
    total_available INTEGER,
    redeemed_count INTEGER DEFAULT 0,
    tier_restriction UUID REFERENCES loyalty_tiers(id),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Points redemptions
CREATE TABLE IF NOT EXISTS loyalty_redemptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rider_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reward_id UUID NOT NULL REFERENCES loyalty_rewards_catalog(id),
    points_spent INTEGER NOT NULL,
    redemption_code VARCHAR(50) UNIQUE,
    status VARCHAR(20) DEFAULT 'active',  -- active, used, expired, cancelled
    used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================
-- PART 2: DRIVER INCENTIVES & GAMIFICATION
-- =============================================

-- Driver tiers/levels
CREATE TABLE IF NOT EXISTS driver_tiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,  -- New, Bronze, Silver, Gold, Platinum, Diamond
    display_name VARCHAR(100) NOT NULL,
    min_rides INTEGER NOT NULL DEFAULT 0,
    min_rating DECIMAL(3,2) DEFAULT 0,
    min_acceptance_rate DECIMAL(5,2) DEFAULT 0,
    commission_rate DECIMAL(5,2) NOT NULL,  -- Lower commission for higher tiers
    priority_dispatch BOOLEAN DEFAULT FALSE,
    surge_multiplier_bonus DECIMAL(3,2) DEFAULT 0,  -- Extra surge earnings
    weekly_bonus_eligible BOOLEAN DEFAULT FALSE,
    instant_payout BOOLEAN DEFAULT FALSE,
    dedicated_support BOOLEAN DEFAULT FALSE,
    icon_url VARCHAR(500),
    color_hex VARCHAR(7),
    benefits JSONB DEFAULT '[]',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Driver gamification profile
CREATE TABLE IF NOT EXISTS driver_gamification (
    driver_id UUID PRIMARY KEY REFERENCES drivers(id) ON DELETE CASCADE,
    current_tier_id UUID REFERENCES driver_tiers(id),
    total_points INTEGER DEFAULT 0,
    weekly_points INTEGER DEFAULT 0,
    monthly_points INTEGER DEFAULT 0,
    current_streak INTEGER DEFAULT 0,  -- Consecutive days driven
    longest_streak INTEGER DEFAULT 0,
    acceptance_streak INTEGER DEFAULT 0,  -- Consecutive accepts
    five_star_streak INTEGER DEFAULT 0,  -- Consecutive 5-star rides
    total_quests_completed INTEGER DEFAULT 0,
    total_challenges_won INTEGER DEFAULT 0,
    total_bonuses_earned DECIMAL(12,2) DEFAULT 0,
    last_active_date DATE,
    tier_evaluated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Driver quests (time-limited earning opportunities)
CREATE TABLE IF NOT EXISTS driver_quests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    quest_type VARCHAR(50) NOT NULL,  -- rides_count, earnings, hours_online, acceptance_rate, rating
    target_value INTEGER NOT NULL,
    time_window_hours INTEGER,  -- e.g., 24 for daily, 168 for weekly
    reward_type VARCHAR(50) NOT NULL,  -- cash, points, multiplier
    reward_value DECIMAL(10,2) NOT NULL,
    min_tier_id UUID REFERENCES driver_tiers(id),
    region_id UUID,  -- Specific region only
    city_id UUID,
    vehicle_type VARCHAR(50),  -- Specific vehicle type
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    max_participants INTEGER,
    current_participants INTEGER DEFAULT 0,
    is_guaranteed BOOLEAN DEFAULT FALSE,  -- Guaranteed bonus if target met
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Driver quest progress
CREATE TABLE IF NOT EXISTS driver_quest_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
    quest_id UUID NOT NULL REFERENCES driver_quests(id) ON DELETE CASCADE,
    current_value DECIMAL(10,2) DEFAULT 0,
    target_value INTEGER NOT NULL,
    progress_percent DECIMAL(5,2) DEFAULT 0,
    completed BOOLEAN DEFAULT FALSE,
    completed_at TIMESTAMPTZ,
    reward_paid BOOLEAN DEFAULT FALSE,
    reward_paid_at TIMESTAMPTZ,
    reward_amount DECIMAL(10,2),
    started_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(driver_id, quest_id)
);

-- Peak hour bonuses
CREATE TABLE IF NOT EXISTS driver_peak_bonuses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    bonus_type VARCHAR(50) NOT NULL,  -- multiplier, flat_amount, per_ride
    bonus_value DECIMAL(10,2) NOT NULL,
    day_of_week INTEGER[],  -- 0=Sunday, 6=Saturday, NULL=all
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    region_id UUID,
    city_id UUID,
    min_rides INTEGER DEFAULT 1,  -- Minimum rides to qualify
    is_active BOOLEAN DEFAULT TRUE,
    valid_from DATE,
    valid_until DATE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Driver achievements/badges
CREATE TABLE IF NOT EXISTS driver_achievements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL,  -- rides, rating, streak, special, milestone
    icon_url VARCHAR(500),
    requirement_type VARCHAR(50) NOT NULL,
    requirement_value INTEGER NOT NULL,
    points_reward INTEGER DEFAULT 0,
    cash_reward DECIMAL(10,2) DEFAULT 0,
    is_secret BOOLEAN DEFAULT FALSE,  -- Hidden until earned
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Driver earned achievements
CREATE TABLE IF NOT EXISTS driver_earned_achievements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
    achievement_id UUID NOT NULL REFERENCES driver_achievements(id) ON DELETE CASCADE,
    earned_at TIMESTAMPTZ DEFAULT NOW(),
    notified BOOLEAN DEFAULT FALSE,
    UNIQUE(driver_id, achievement_id)
);

-- Driver leaderboards
CREATE TABLE IF NOT EXISTS driver_leaderboards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    leaderboard_type VARCHAR(50) NOT NULL,  -- weekly_rides, weekly_earnings, monthly_rating
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    region_id UUID,
    city_id UUID,
    rankings JSONB NOT NULL DEFAULT '[]',  -- [{driver_id, rank, value, reward}]
    finalized BOOLEAN DEFAULT FALSE,
    finalized_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(leaderboard_type, period_start, region_id, city_id)
);

-- =============================================
-- PART 3: BACKGROUND CHECK INTEGRATION
-- =============================================

CREATE TABLE IF NOT EXISTS driver_background_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,  -- checkr, sterling, onfido, manual
    external_id VARCHAR(255),  -- Provider's reference ID
    check_type VARCHAR(50) NOT NULL,  -- criminal, driving, identity, employment
    status VARCHAR(30) NOT NULL DEFAULT 'pending',  -- pending, in_progress, passed, failed, review_required
    result JSONB,  -- Provider's detailed result
    report_url VARCHAR(500),  -- Link to full report
    flags JSONB DEFAULT '[]',  -- Any concerns flagged
    risk_level VARCHAR(20),  -- low, medium, high
    notes TEXT,
    reviewer_id UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    requested_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,  -- When recheck is needed
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Background check requirements by region
CREATE TABLE IF NOT EXISTS background_check_requirements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    country_code VARCHAR(3) NOT NULL,
    region_code VARCHAR(10),
    check_types VARCHAR(50)[] NOT NULL,  -- Required check types
    provider VARCHAR(50) NOT NULL,
    validity_months INTEGER DEFAULT 12,  -- How long check is valid
    is_mandatory BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================
-- PART 4: DRIVER SELFIE VERIFICATION
-- =============================================

CREATE TABLE IF NOT EXISTS driver_selfie_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
    ride_id UUID REFERENCES rides(id),
    verification_type VARCHAR(30) NOT NULL,  -- ride_start, random, shift_start, daily
    selfie_url VARCHAR(500) NOT NULL,
    selfie_key VARCHAR(255) NOT NULL,
    reference_photo_url VARCHAR(500),  -- Profile/license photo to compare
    match_score DECIMAL(5,4),  -- 0.0000 to 1.0000
    match_threshold DECIMAL(5,4) DEFAULT 0.8500,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',  -- pending, passed, failed, manual_review
    failure_reason VARCHAR(100),
    processor VARCHAR(50),  -- aws_rekognition, google_vision, azure_face
    processing_time_ms INTEGER,
    metadata JSONB,
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Selfie verification settings
CREATE TABLE IF NOT EXISTS selfie_verification_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    setting_type VARCHAR(50) NOT NULL UNIQUE,  -- ride_start, random_check, shift_start
    enabled BOOLEAN DEFAULT TRUE,
    frequency VARCHAR(50),  -- always, random_10_percent, daily, weekly
    match_threshold DECIMAL(5,4) DEFAULT 0.8500,
    max_attempts INTEGER DEFAULT 3,
    lockout_minutes INTEGER DEFAULT 30,
    require_liveness BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================
-- INDEXES
-- =============================================

-- Loyalty indexes
CREATE INDEX idx_loyalty_points_transactions_rider ON loyalty_points_transactions(rider_id);
CREATE INDEX idx_loyalty_points_transactions_created ON loyalty_points_transactions(created_at DESC);
CREATE INDEX idx_rider_challenge_progress_rider ON rider_challenge_progress(rider_id);
CREATE INDEX idx_loyalty_redemptions_rider ON loyalty_redemptions(rider_id);
CREATE INDEX idx_loyalty_redemptions_status ON loyalty_redemptions(status) WHERE status = 'active';

-- Driver gamification indexes
CREATE INDEX idx_driver_quest_progress_driver ON driver_quest_progress(driver_id);
CREATE INDEX idx_driver_quest_progress_quest ON driver_quest_progress(quest_id);
CREATE INDEX idx_driver_quests_active ON driver_quests(start_time, end_time) WHERE is_active = TRUE;
CREATE INDEX idx_driver_earned_achievements_driver ON driver_earned_achievements(driver_id);
CREATE INDEX idx_driver_leaderboards_type_period ON driver_leaderboards(leaderboard_type, period_start);

-- Background check indexes
CREATE INDEX idx_background_checks_driver ON driver_background_checks(driver_id);
CREATE INDEX idx_background_checks_status ON driver_background_checks(status);
CREATE INDEX idx_background_checks_expires ON driver_background_checks(expires_at) WHERE status = 'passed';

-- Selfie verification indexes
CREATE INDEX idx_selfie_verifications_driver ON driver_selfie_verifications(driver_id);
CREATE INDEX idx_selfie_verifications_ride ON driver_selfie_verifications(ride_id);
CREATE INDEX idx_selfie_verifications_status ON driver_selfie_verifications(status) WHERE status = 'pending';

-- =============================================
-- SEED DATA
-- =============================================

-- Insert loyalty tiers
INSERT INTO loyalty_tiers (name, display_name, min_points, max_points, multiplier, discount_percent, priority_support, free_cancellations, free_upgrades, color_hex, benefits) VALUES
('bronze', 'Bronze', 0, 999, 1.0, 0, FALSE, 0, 0, '#CD7F32', '["Earn 1 point per $1 spent", "Birthday bonus points"]'),
('silver', 'Silver', 1000, 4999, 1.25, 2, FALSE, 1, 0, '#C0C0C0', '["25% bonus points", "2% ride discount", "1 free cancellation/month"]'),
('gold', 'Gold', 5000, 14999, 1.5, 5, TRUE, 2, 1, '#FFD700', '["50% bonus points", "5% ride discount", "2 free cancellations/month", "1 free upgrade/month", "Priority support"]'),
('platinum', 'Platinum', 15000, 49999, 2.0, 8, TRUE, 3, 2, '#E5E4E2', '["Double points", "8% ride discount", "3 free cancellations/month", "2 free upgrades/month", "Priority matching"]'),
('diamond', 'Diamond', 50000, NULL, 2.5, 12, TRUE, 5, 5, '#B9F2FF', '["2.5x points", "12% ride discount", "5 free cancellations/month", "5 free upgrades/month", "VIP support line", "Airport lounge access"]')
ON CONFLICT (name) DO NOTHING;

-- Insert driver tiers
INSERT INTO driver_tiers (name, display_name, min_rides, min_rating, min_acceptance_rate, commission_rate, priority_dispatch, surge_multiplier_bonus, weekly_bonus_eligible, instant_payout, color_hex, benefits) VALUES
('new', 'New Driver', 0, 0, 0, 25.00, FALSE, 0, FALSE, FALSE, '#808080', '["Welcome bonus eligible", "Training support"]'),
('bronze', 'Bronze', 50, 4.5, 70, 23.00, FALSE, 0, TRUE, FALSE, '#CD7F32', '["Weekly bonus eligible", "2% lower commission"]'),
('silver', 'Silver', 200, 4.6, 75, 21.00, FALSE, 0.05, TRUE, FALSE, '#C0C0C0', '["5% surge bonus", "4% lower commission", "Priority support"]'),
('gold', 'Gold', 500, 4.7, 80, 19.00, TRUE, 0.10, TRUE, TRUE, '#FFD700', '["Priority dispatch", "10% surge bonus", "Instant payout", "6% lower commission"]'),
('platinum', 'Platinum', 1000, 4.8, 85, 17.00, TRUE, 0.15, TRUE, TRUE, '#E5E4E2', '["Top priority dispatch", "15% surge bonus", "8% lower commission", "Exclusive quests"]'),
('diamond', 'Diamond', 2500, 4.9, 90, 15.00, TRUE, 0.20, TRUE, TRUE, '#B9F2FF', '["Highest priority", "20% surge bonus", "10% lower commission", "VIP support", "Exclusive events"]')
ON CONFLICT (name) DO NOTHING;

-- Insert default achievements
INSERT INTO driver_achievements (code, name, description, category, requirement_type, requirement_value, points_reward) VALUES
('first_ride', 'First Ride', 'Complete your first ride', 'milestone', 'rides_completed', 1, 50),
('century', 'Century Club', 'Complete 100 rides', 'milestone', 'rides_completed', 100, 500),
('thousand', 'Road Warrior', 'Complete 1000 rides', 'milestone', 'rides_completed', 1000, 2000),
('five_thousand', 'Legend', 'Complete 5000 rides', 'milestone', 'rides_completed', 5000, 10000),
('perfect_week', 'Perfect Week', 'Maintain 5.0 rating for a week', 'rating', 'perfect_rating_days', 7, 200),
('streak_7', 'Week Warrior', 'Drive 7 days in a row', 'streak', 'consecutive_days', 7, 100),
('streak_30', 'Monthly Master', 'Drive 30 days in a row', 'streak', 'consecutive_days', 30, 500),
('early_bird', 'Early Bird', 'Complete 50 rides before 8 AM', 'special', 'early_rides', 50, 300),
('night_owl', 'Night Owl', 'Complete 50 rides after 10 PM', 'special', 'night_rides', 50, 300),
('five_star_50', 'Five Star Streak', 'Get 50 consecutive 5-star ratings', 'rating', 'five_star_streak', 50, 1000)
ON CONFLICT (code) DO NOTHING;

-- Insert selfie verification settings
INSERT INTO selfie_verification_settings (setting_type, enabled, frequency, match_threshold, max_attempts, require_liveness) VALUES
('ride_start', TRUE, 'random_10_percent', 0.85, 3, TRUE),
('shift_start', TRUE, 'always', 0.85, 3, TRUE),
('random_check', TRUE, 'random_5_percent', 0.85, 2, FALSE)
ON CONFLICT (setting_type) DO NOTHING;
