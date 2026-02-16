-- =============================================
-- Migration 000018: New Feature Tables
-- Creates tables for: cancellation, support, disputes,
-- tips, ratings, earnings, vehicles, payment methods,
-- family, gift cards, subscriptions, preferences,
-- wait time, chat
-- =============================================

-- =============================================
-- PART 1: CANCELLATION
-- =============================================

CREATE TABLE IF NOT EXISTS cancellation_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    free_cancel_window_minutes INTEGER NOT NULL DEFAULT 2,
    max_free_cancels_per_day INTEGER NOT NULL DEFAULT 3,
    max_free_cancels_per_week INTEGER NOT NULL DEFAULT 10,
    driver_no_show_minutes INTEGER NOT NULL DEFAULT 5,
    rider_no_show_minutes INTEGER NOT NULL DEFAULT 5,
    driver_penalty_threshold INTEGER NOT NULL DEFAULT 20,
    rider_penalty_threshold INTEGER NOT NULL DEFAULT 30,
    is_default BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO cancellation_policies (name, is_default, is_active)
VALUES ('Default Policy', true, true)
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS cancellation_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ride_id UUID NOT NULL REFERENCES rides(id),
    rider_id UUID NOT NULL REFERENCES users(id),
    driver_id UUID REFERENCES users(id),
    cancelled_by VARCHAR(20) NOT NULL,
    reason_code VARCHAR(50) NOT NULL,
    reason_text TEXT,
    fee_amount DECIMAL(10,2) NOT NULL DEFAULT 0,
    fee_waived BOOLEAN NOT NULL DEFAULT false,
    waiver_reason VARCHAR(50),
    minutes_since_request DECIMAL(10,2) NOT NULL DEFAULT 0,
    minutes_since_accept DECIMAL(10,2),
    ride_status_at_cancel VARCHAR(20) NOT NULL,
    pickup_latitude DECIMAL(10,8) NOT NULL,
    pickup_longitude DECIMAL(11,8) NOT NULL,
    cancelled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cancellation_records_ride_id ON cancellation_records(ride_id);
CREATE INDEX idx_cancellation_records_rider_id ON cancellation_records(rider_id);
CREATE INDEX idx_cancellation_records_driver_id ON cancellation_records(driver_id);
CREATE INDEX idx_cancellation_records_cancelled_at ON cancellation_records(cancelled_at);
CREATE INDEX idx_cancellation_records_cancelled_by ON cancellation_records(cancelled_by);

-- =============================================
-- PART 2: SUPPORT TICKETS
-- =============================================

CREATE TABLE IF NOT EXISTS support_tickets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    ride_id UUID REFERENCES rides(id),
    assigned_to UUID REFERENCES users(id),
    ticket_number VARCHAR(20) NOT NULL UNIQUE,
    category VARCHAR(30) NOT NULL,
    priority VARCHAR(20) NOT NULL DEFAULT 'medium',
    status VARCHAR(30) NOT NULL DEFAULT 'open',
    subject VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    tags TEXT[] DEFAULT '{}',
    resolved_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_support_tickets_user_id ON support_tickets(user_id);
CREATE INDEX idx_support_tickets_status ON support_tickets(status);
CREATE INDEX idx_support_tickets_priority ON support_tickets(priority);
CREATE INDEX idx_support_tickets_category ON support_tickets(category);
CREATE INDEX idx_support_tickets_updated_at ON support_tickets(updated_at);

CREATE TABLE IF NOT EXISTS support_ticket_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ticket_id UUID NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users(id),
    sender_type VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    attachments TEXT[] DEFAULT '{}',
    is_internal BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_support_ticket_messages_ticket_id ON support_ticket_messages(ticket_id);
CREATE INDEX idx_support_ticket_messages_created_at ON support_ticket_messages(created_at);

CREATE TABLE IF NOT EXISTS faq_articles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    category VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    view_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_faq_articles_category ON faq_articles(category);
CREATE INDEX idx_faq_articles_is_active ON faq_articles(is_active);

-- =============================================
-- PART 3: FARE DISPUTES
-- =============================================

CREATE TABLE IF NOT EXISTS fare_disputes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ride_id UUID NOT NULL REFERENCES rides(id),
    user_id UUID NOT NULL REFERENCES users(id),
    driver_id UUID REFERENCES users(id),
    dispute_number VARCHAR(20) NOT NULL UNIQUE,
    reason VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    original_fare DECIMAL(10,2) NOT NULL,
    disputed_amount DECIMAL(10,2) NOT NULL,
    refund_amount DECIMAL(10,2),
    resolution_type VARCHAR(30),
    resolution_note TEXT,
    resolved_by UUID REFERENCES users(id),
    evidence TEXT[] DEFAULT '{}',
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fare_disputes_ride_id ON fare_disputes(ride_id);
CREATE INDEX idx_fare_disputes_user_id ON fare_disputes(user_id);
CREATE INDEX idx_fare_disputes_status ON fare_disputes(status);
CREATE INDEX idx_fare_disputes_created_at ON fare_disputes(created_at);

CREATE TABLE IF NOT EXISTS fare_dispute_comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    dispute_id UUID NOT NULL REFERENCES fare_disputes(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    user_role VARCHAR(20) NOT NULL,
    comment TEXT NOT NULL,
    is_internal BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fare_dispute_comments_dispute_id ON fare_dispute_comments(dispute_id);

-- =============================================
-- PART 4: TIPS
-- =============================================

CREATE TABLE IF NOT EXISTS tips (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ride_id UUID NOT NULL REFERENCES rides(id),
    rider_id UUID NOT NULL REFERENCES users(id),
    driver_id UUID NOT NULL REFERENCES users(id),
    amount DECIMAL(10,2) NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    message TEXT,
    is_anonymous BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tips_ride_id ON tips(ride_id);
CREATE INDEX idx_tips_rider_id ON tips(rider_id);
CREATE INDEX idx_tips_driver_id ON tips(driver_id);
CREATE INDEX idx_tips_status ON tips(status);
CREATE INDEX idx_tips_created_at ON tips(created_at);

-- =============================================
-- PART 5: RATINGS
-- =============================================

CREATE TABLE IF NOT EXISTS ratings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ride_id UUID NOT NULL REFERENCES rides(id),
    rater_id UUID NOT NULL REFERENCES users(id),
    ratee_id UUID NOT NULL REFERENCES users(id),
    rater_type VARCHAR(20) NOT NULL,
    score INTEGER NOT NULL CHECK (score >= 1 AND score <= 5),
    comment TEXT,
    tags TEXT[] DEFAULT '{}',
    is_public BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ratings_ride_id ON ratings(ride_id);
CREATE INDEX idx_ratings_rater_id ON ratings(rater_id);
CREATE INDEX idx_ratings_ratee_id ON ratings(ratee_id);
CREATE INDEX idx_ratings_created_at ON ratings(created_at);
CREATE UNIQUE INDEX idx_ratings_ride_rater_unique ON ratings(ride_id, rater_id);

CREATE TABLE IF NOT EXISTS rating_responses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rating_id UUID NOT NULL REFERENCES ratings(id) ON DELETE CASCADE UNIQUE,
    user_id UUID NOT NULL REFERENCES users(id),
    comment TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================
-- PART 6: DRIVER EARNINGS
-- =============================================

CREATE TABLE IF NOT EXISTS driver_bank_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    driver_id UUID NOT NULL REFERENCES users(id),
    bank_name VARCHAR(100) NOT NULL,
    account_holder VARCHAR(200) NOT NULL,
    account_number VARCHAR(255) NOT NULL,
    routing_number VARCHAR(50),
    iban VARCHAR(50),
    swift_code VARCHAR(20),
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    is_primary BOOLEAN NOT NULL DEFAULT false,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_driver_bank_accounts_driver_id ON driver_bank_accounts(driver_id);

CREATE TABLE IF NOT EXISTS driver_payouts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    driver_id UUID NOT NULL REFERENCES users(id),
    amount DECIMAL(12,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    method VARCHAR(30) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    bank_account_id UUID REFERENCES driver_bank_accounts(id),
    reference VARCHAR(255) NOT NULL,
    earning_count INTEGER NOT NULL DEFAULT 0,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    processed_at TIMESTAMPTZ,
    failure_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_driver_payouts_driver_id ON driver_payouts(driver_id);
CREATE INDEX idx_driver_payouts_status ON driver_payouts(status);
CREATE INDEX idx_driver_payouts_created_at ON driver_payouts(created_at);

CREATE TABLE IF NOT EXISTS driver_earnings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    driver_id UUID NOT NULL REFERENCES users(id),
    ride_id UUID REFERENCES rides(id),
    delivery_id UUID,
    type VARCHAR(30) NOT NULL,
    gross_amount DECIMAL(12,2) NOT NULL,
    commission DECIMAL(12,2) NOT NULL,
    net_amount DECIMAL(12,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    description TEXT NOT NULL DEFAULT '',
    is_paid_out BOOLEAN NOT NULL DEFAULT false,
    payout_id UUID REFERENCES driver_payouts(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_driver_earnings_driver_id ON driver_earnings(driver_id);
CREATE INDEX idx_driver_earnings_ride_id ON driver_earnings(ride_id);
CREATE INDEX idx_driver_earnings_type ON driver_earnings(type);
CREATE INDEX idx_driver_earnings_created_at ON driver_earnings(created_at);
CREATE INDEX idx_driver_earnings_is_paid_out ON driver_earnings(is_paid_out);
CREATE INDEX idx_driver_earnings_driver_date ON driver_earnings(driver_id, created_at);

CREATE TABLE IF NOT EXISTS driver_earning_goals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    driver_id UUID NOT NULL REFERENCES users(id),
    target_amount DECIMAL(12,2) NOT NULL,
    period VARCHAR(20) NOT NULL,
    current_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_driver_earning_goals_driver_id ON driver_earning_goals(driver_id);
CREATE UNIQUE INDEX idx_driver_earning_goals_active ON driver_earning_goals(driver_id, period) WHERE is_active = true;

-- =============================================
-- PART 7: VEHICLES
-- =============================================

CREATE TABLE IF NOT EXISTS vehicles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    driver_id UUID NOT NULL REFERENCES users(id),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    category VARCHAR(20) NOT NULL,
    make VARCHAR(100) NOT NULL,
    model VARCHAR(100) NOT NULL,
    year INTEGER NOT NULL,
    color VARCHAR(50) NOT NULL,
    license_plate VARCHAR(20) NOT NULL,
    vin VARCHAR(50),
    fuel_type VARCHAR(20) NOT NULL,
    max_passengers INTEGER NOT NULL DEFAULT 4,
    has_child_seat BOOLEAN NOT NULL DEFAULT false,
    has_wheelchair_access BOOLEAN NOT NULL DEFAULT false,
    has_wifi BOOLEAN NOT NULL DEFAULT false,
    has_charger BOOLEAN NOT NULL DEFAULT false,
    pet_friendly BOOLEAN NOT NULL DEFAULT false,
    luggage_capacity VARCHAR(20) NOT NULL DEFAULT '2',
    registration_photo_url TEXT,
    insurance_photo_url TEXT,
    front_photo_url TEXT,
    back_photo_url TEXT,
    side_photo_url TEXT,
    interior_photo_url TEXT,
    insurance_expiry TIMESTAMPTZ,
    registration_expiry TIMESTAMPTZ,
    inspection_expiry TIMESTAMPTZ,
    rejection_reason TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vehicles_driver_id ON vehicles(driver_id);
CREATE INDEX idx_vehicles_status ON vehicles(status);
CREATE INDEX idx_vehicles_is_active ON vehicles(is_active);
CREATE INDEX idx_vehicles_category ON vehicles(category);

CREATE TABLE IF NOT EXISTS vehicle_inspections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vehicle_id UUID NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    inspector_id UUID REFERENCES users(id),
    status VARCHAR(20) NOT NULL DEFAULT 'scheduled',
    scheduled_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vehicle_inspections_vehicle_id ON vehicle_inspections(vehicle_id);
CREATE INDEX idx_vehicle_inspections_status ON vehicle_inspections(status);

CREATE TABLE IF NOT EXISTS maintenance_reminders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vehicle_id UUID NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    due_date TIMESTAMPTZ,
    due_mileage INTEGER,
    is_completed BOOLEAN NOT NULL DEFAULT false,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_maintenance_reminders_vehicle_id ON maintenance_reminders(vehicle_id);
CREATE INDEX idx_maintenance_reminders_is_completed ON maintenance_reminders(is_completed);

-- =============================================
-- PART 8: PAYMENT METHODS
-- =============================================

CREATE TABLE IF NOT EXISTS payment_methods (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    type VARCHAR(30) NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    card_brand VARCHAR(20),
    card_last4 VARCHAR(4),
    card_exp_month INTEGER,
    card_exp_year INTEGER,
    card_holder_name VARCHAR(200),
    wallet_balance DECIMAL(12,2) DEFAULT 0,
    provider_id VARCHAR(255),
    provider_type VARCHAR(50),
    nickname VARCHAR(100),
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    country VARCHAR(3),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payment_methods_user_id ON payment_methods(user_id);
CREATE INDEX idx_payment_methods_type ON payment_methods(type);

-- Drop old wallet_transactions (uses wallet_id FK to wallets table)
-- and recreate with new schema (uses user_id + payment_method_id)
DROP TABLE IF EXISTS wallet_transactions CASCADE;

CREATE TABLE wallet_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    payment_method_id UUID NOT NULL REFERENCES payment_methods(id),
    type VARCHAR(20) NOT NULL,
    amount DECIMAL(12,2) NOT NULL,
    balance_before DECIMAL(12,2) NOT NULL DEFAULT 0,
    balance_after DECIMAL(12,2) NOT NULL DEFAULT 0,
    description TEXT NOT NULL DEFAULT '',
    ride_id UUID REFERENCES rides(id),
    reference_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wallet_transactions_user_id ON wallet_transactions(user_id);
CREATE INDEX idx_wallet_transactions_payment_method_id ON wallet_transactions(payment_method_id);
CREATE INDEX idx_wallet_transactions_created_at ON wallet_transactions(created_at);

-- =============================================
-- PART 9: FAMILY ACCOUNTS
-- =============================================

CREATE TABLE IF NOT EXISTS family_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR(100) NOT NULL,
    monthly_budget DECIMAL(12,2),
    current_spend DECIMAL(12,2) NOT NULL DEFAULT 0,
    budget_reset_day INTEGER NOT NULL DEFAULT 1 CHECK (budget_reset_day >= 1 AND budget_reset_day <= 28),
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_family_accounts_owner_id ON family_accounts(owner_id);

CREATE TABLE IF NOT EXISTS family_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    family_id UUID NOT NULL REFERENCES family_accounts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    role VARCHAR(20) NOT NULL DEFAULT 'adult',
    display_name VARCHAR(100) NOT NULL,
    use_shared_payment BOOLEAN NOT NULL DEFAULT true,
    ride_approval_required BOOLEAN NOT NULL DEFAULT false,
    max_fare_per_ride DECIMAL(10,2),
    allowed_hours_start INTEGER,
    allowed_hours_end INTEGER,
    monthly_limit DECIMAL(12,2),
    monthly_spend DECIMAL(12,2) NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_family_members_family_id ON family_members(family_id);
CREATE INDEX idx_family_members_user_id ON family_members(user_id);

CREATE TABLE IF NOT EXISTS family_invites (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    family_id UUID NOT NULL REFERENCES family_accounts(id) ON DELETE CASCADE,
    inviter_id UUID NOT NULL REFERENCES users(id),
    invitee_id UUID REFERENCES users(id),
    email VARCHAR(255),
    phone VARCHAR(20),
    role VARCHAR(20) NOT NULL DEFAULT 'adult',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    responded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_family_invites_family_id ON family_invites(family_id);
CREATE INDEX idx_family_invites_invitee_id ON family_invites(invitee_id);
CREATE INDEX idx_family_invites_status ON family_invites(status);

CREATE TABLE IF NOT EXISTS family_ride_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    family_id UUID NOT NULL REFERENCES family_accounts(id),
    member_id UUID NOT NULL REFERENCES family_members(id),
    ride_id UUID NOT NULL REFERENCES rides(id),
    fare_amount DECIMAL(10,2) NOT NULL,
    paid_by_owner BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_family_ride_logs_family_id ON family_ride_logs(family_id);
CREATE INDEX idx_family_ride_logs_member_id ON family_ride_logs(member_id);
CREATE INDEX idx_family_ride_logs_created_at ON family_ride_logs(created_at);

-- =============================================
-- PART 10: GIFT CARDS
-- =============================================

CREATE TABLE IF NOT EXISTS gift_cards (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(50) NOT NULL UNIQUE,
    card_type VARCHAR(30) NOT NULL DEFAULT 'purchased',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    original_amount DECIMAL(10,2) NOT NULL,
    remaining_amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    purchaser_id UUID REFERENCES users(id),
    recipient_id UUID REFERENCES users(id),
    recipient_email VARCHAR(255),
    recipient_name VARCHAR(200),
    personal_message TEXT,
    design_template VARCHAR(50),
    expires_at TIMESTAMPTZ,
    redeemed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_gift_cards_code ON gift_cards(code);
CREATE INDEX idx_gift_cards_purchaser_id ON gift_cards(purchaser_id);
CREATE INDEX idx_gift_cards_recipient_id ON gift_cards(recipient_id);
CREATE INDEX idx_gift_cards_status ON gift_cards(status);

CREATE TABLE IF NOT EXISTS gift_card_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    card_id UUID NOT NULL REFERENCES gift_cards(id),
    user_id UUID NOT NULL REFERENCES users(id),
    ride_id UUID REFERENCES rides(id),
    amount DECIMAL(10,2) NOT NULL,
    balance_before DECIMAL(10,2) NOT NULL,
    balance_after DECIMAL(10,2) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_gift_card_transactions_card_id ON gift_card_transactions(card_id);
CREATE INDEX idx_gift_card_transactions_user_id ON gift_card_transactions(user_id);

-- =============================================
-- PART 11: SUBSCRIPTIONS
-- =============================================

CREATE TABLE IF NOT EXISTS subscription_plans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description TEXT NOT NULL,
    plan_type VARCHAR(30) NOT NULL,
    billing_period VARCHAR(20) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    rides_included INTEGER,
    max_ride_value DECIMAL(10,2),
    discount_pct DECIMAL(5,2) NOT NULL DEFAULT 0,
    allowed_ride_types JSONB DEFAULT '[]',
    allowed_cities JSONB DEFAULT '[]',
    max_distance_km DECIMAL(10,2),
    priority_matching BOOLEAN NOT NULL DEFAULT false,
    free_upgrades INTEGER NOT NULL DEFAULT 0,
    free_cancellations INTEGER NOT NULL DEFAULT 0,
    wait_time_guarantee INTEGER,
    surge_protection BOOLEAN NOT NULL DEFAULT false,
    surge_max_cap DECIMAL(4,2),
    popular_badge BOOLEAN NOT NULL DEFAULT false,
    savings_label VARCHAR(100) NOT NULL DEFAULT '',
    display_order INTEGER NOT NULL DEFAULT 0,
    trial_days INTEGER NOT NULL DEFAULT 0,
    trial_rides INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    plan_id UUID NOT NULL REFERENCES subscription_plans(id),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end TIMESTAMPTZ NOT NULL,
    rides_used INTEGER NOT NULL DEFAULT 0,
    upgrades_used INTEGER NOT NULL DEFAULT 0,
    cancellations_used INTEGER NOT NULL DEFAULT 0,
    total_saved DECIMAL(10,2) NOT NULL DEFAULT 0,
    payment_method VARCHAR(50) NOT NULL,
    stripe_sub_id VARCHAR(255),
    next_billing_date TIMESTAMPTZ,
    last_payment_date TIMESTAMPTZ,
    failed_payments INTEGER NOT NULL DEFAULT 0,
    is_trial_active BOOLEAN NOT NULL DEFAULT false,
    trial_ends_at TIMESTAMPTZ,
    activated_at TIMESTAMPTZ,
    paused_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    cancel_reason TEXT,
    auto_renew BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_plan_id ON subscriptions(plan_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);

CREATE TABLE IF NOT EXISTS subscription_usage_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    ride_id UUID NOT NULL REFERENCES rides(id),
    usage_type VARCHAR(30) NOT NULL,
    original_fare DECIMAL(10,2) NOT NULL,
    discounted_fare DECIMAL(10,2) NOT NULL,
    savings_amount DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscription_usage_logs_subscription_id ON subscription_usage_logs(subscription_id);
CREATE INDEX idx_subscription_usage_logs_ride_id ON subscription_usage_logs(ride_id);

-- =============================================
-- PART 12: PREFERENCES
-- =============================================

CREATE TABLE IF NOT EXISTS rider_preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) UNIQUE,
    temperature VARCHAR(20) NOT NULL DEFAULT 'no_preference',
    music VARCHAR(20) NOT NULL DEFAULT 'no_preference',
    conversation VARCHAR(20) NOT NULL DEFAULT 'no_preference',
    route VARCHAR(20) NOT NULL DEFAULT 'no_preference',
    child_seat BOOLEAN NOT NULL DEFAULT false,
    pet_friendly BOOLEAN NOT NULL DEFAULT false,
    wheelchair_access BOOLEAN NOT NULL DEFAULT false,
    prefer_female_driver BOOLEAN NOT NULL DEFAULT false,
    luggage_assistance BOOLEAN NOT NULL DEFAULT false,
    max_passengers INTEGER,
    preferred_language VARCHAR(10),
    special_needs TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ride_preference_overrides (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ride_id UUID NOT NULL REFERENCES rides(id) UNIQUE,
    user_id UUID NOT NULL REFERENCES users(id),
    temperature VARCHAR(20),
    music VARCHAR(20),
    conversation VARCHAR(20),
    route VARCHAR(20),
    child_seat BOOLEAN,
    pet_friendly BOOLEAN,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS driver_capabilities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    driver_id UUID NOT NULL REFERENCES users(id) UNIQUE,
    has_child_seat BOOLEAN NOT NULL DEFAULT false,
    pet_friendly BOOLEAN NOT NULL DEFAULT false,
    wheelchair_access BOOLEAN NOT NULL DEFAULT false,
    luggage_capacity INTEGER NOT NULL DEFAULT 2,
    max_passengers INTEGER NOT NULL DEFAULT 4,
    languages TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================
-- PART 13: WAIT TIME
-- =============================================

CREATE TABLE IF NOT EXISTS wait_time_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    free_wait_minutes INTEGER NOT NULL DEFAULT 3,
    charge_per_minute DECIMAL(6,2) NOT NULL DEFAULT 0.50,
    max_wait_minutes INTEGER NOT NULL DEFAULT 15,
    max_wait_charge DECIMAL(10,2) NOT NULL DEFAULT 10.00,
    applies_to VARCHAR(20) NOT NULL DEFAULT 'pickup',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO wait_time_configs (name, free_wait_minutes, charge_per_minute, max_wait_minutes, max_wait_charge, applies_to, is_active)
VALUES ('Default Pickup Wait', 3, 0.50, 15, 10.00, 'pickup', true)
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS wait_time_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ride_id UUID NOT NULL REFERENCES rides(id),
    driver_id UUID NOT NULL REFERENCES users(id),
    config_id UUID NOT NULL REFERENCES wait_time_configs(id),
    wait_type VARCHAR(20) NOT NULL,
    arrived_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    free_minutes DECIMAL(10,2) NOT NULL DEFAULT 0,
    total_wait_minutes DECIMAL(10,2) NOT NULL DEFAULT 0,
    chargeable_minutes DECIMAL(10,2) NOT NULL DEFAULT 0,
    charge_per_minute DECIMAL(6,2) NOT NULL DEFAULT 0,
    total_charge DECIMAL(10,2) NOT NULL DEFAULT 0,
    was_capped BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(20) NOT NULL DEFAULT 'waiting',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wait_time_records_ride_id ON wait_time_records(ride_id);
CREATE INDEX idx_wait_time_records_driver_id ON wait_time_records(driver_id);

CREATE TABLE IF NOT EXISTS wait_time_notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    record_id UUID NOT NULL REFERENCES wait_time_records(id),
    ride_id UUID NOT NULL REFERENCES rides(id),
    rider_id UUID NOT NULL REFERENCES users(id),
    minutes_mark INTEGER NOT NULL,
    message TEXT NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================
-- PART 14: CHAT
-- =============================================

CREATE TABLE IF NOT EXISTS chat_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ride_id UUID NOT NULL REFERENCES rides(id),
    sender_id UUID NOT NULL REFERENCES users(id),
    sender_role VARCHAR(20) NOT NULL,
    message_type VARCHAR(20) NOT NULL DEFAULT 'text',
    content TEXT NOT NULL,
    image_url TEXT,
    latitude DECIMAL(10,8),
    longitude DECIMAL(11,8),
    status VARCHAR(20) NOT NULL DEFAULT 'sent',
    delivered_at TIMESTAMPTZ,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chat_messages_ride_id ON chat_messages(ride_id);
CREATE INDEX idx_chat_messages_sender_id ON chat_messages(sender_id);
CREATE INDEX idx_chat_messages_created_at ON chat_messages(created_at);

CREATE TABLE IF NOT EXISTS chat_quick_replies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    role VARCHAR(20) NOT NULL,
    category VARCHAR(50) NOT NULL,
    text TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true
);

-- Seed quick replies
INSERT INTO chat_quick_replies (role, category, text, sort_order, is_active) VALUES
('rider', 'arrival', 'I''m at the pickup location', 1, true),
('rider', 'arrival', 'I''ll be there in a few minutes', 2, true),
('rider', 'arrival', 'I''m running late', 3, true),
('rider', 'location', 'I''m by the main entrance', 1, true),
('rider', 'location', 'I''m on the other side of the street', 2, true),
('driver', 'arrival', 'I''m here, waiting for you', 1, true),
('driver', 'arrival', 'I''ll be there in a minute', 2, true),
('driver', 'arrival', 'I''m stuck in traffic', 3, true),
('driver', 'location', 'I''m parked near the entrance', 1, true),
('driver', 'location', 'Can you share your exact location?', 2, true)
ON CONFLICT DO NOTHING;
