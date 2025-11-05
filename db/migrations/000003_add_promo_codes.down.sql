-- Drop triggers
DROP TRIGGER IF EXISTS update_ride_types_updated_at ON ride_types;
DROP TRIGGER IF EXISTS update_promo_codes_updated_at ON promo_codes;

-- Drop indexes for ride_types
DROP INDEX IF EXISTS idx_ride_types_is_active;

-- Drop indexes for referrals
DROP INDEX IF EXISTS idx_referrals_referral_code_id;
DROP INDEX IF EXISTS idx_referrals_referred_id;
DROP INDEX IF EXISTS idx_referrals_referrer_id;

-- Drop indexes for referral_codes
DROP INDEX IF EXISTS idx_referral_codes_code;
DROP INDEX IF EXISTS idx_referral_codes_user_id;

-- Drop indexes for promo_code_uses
DROP INDEX IF EXISTS idx_promo_code_uses_used_at;
DROP INDEX IF EXISTS idx_promo_code_uses_ride_id;
DROP INDEX IF EXISTS idx_promo_code_uses_user_id;
DROP INDEX IF EXISTS idx_promo_code_uses_promo_code_id;

-- Drop indexes for promo_codes
DROP INDEX IF EXISTS idx_promo_codes_valid_dates;
DROP INDEX IF EXISTS idx_promo_codes_is_active;
DROP INDEX IF EXISTS idx_promo_codes_code;

-- Remove columns from existing tables
ALTER TABLE drivers DROP COLUMN IF EXISTS ride_types;
ALTER TABLE rides DROP COLUMN IF EXISTS discount_amount;
ALTER TABLE rides DROP COLUMN IF EXISTS promo_code_id;
ALTER TABLE rides DROP COLUMN IF EXISTS ride_type_id;

-- Drop tables
DROP TABLE IF EXISTS referrals;
DROP TABLE IF EXISTS referral_codes;
DROP TABLE IF EXISTS promo_code_uses;
DROP TABLE IF EXISTS promo_codes;
DROP TABLE IF EXISTS ride_types;
