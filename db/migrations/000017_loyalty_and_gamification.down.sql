-- Down migration: Loyalty, Gamification & Background Checks

-- Drop selfie verification
DROP TABLE IF EXISTS selfie_verification_settings CASCADE;
DROP TABLE IF EXISTS driver_selfie_verifications CASCADE;

-- Drop background checks
DROP TABLE IF EXISTS background_check_requirements CASCADE;
DROP TABLE IF EXISTS driver_background_checks CASCADE;

-- Drop driver gamification
DROP TABLE IF EXISTS driver_leaderboards CASCADE;
DROP TABLE IF EXISTS driver_earned_achievements CASCADE;
DROP TABLE IF EXISTS driver_achievements CASCADE;
DROP TABLE IF EXISTS driver_peak_bonuses CASCADE;
DROP TABLE IF EXISTS driver_quest_progress CASCADE;
DROP TABLE IF EXISTS driver_quests CASCADE;
DROP TABLE IF EXISTS driver_gamification CASCADE;
DROP TABLE IF EXISTS driver_tiers CASCADE;

-- Drop rider loyalty
DROP TABLE IF EXISTS loyalty_redemptions CASCADE;
DROP TABLE IF EXISTS loyalty_rewards_catalog CASCADE;
DROP TABLE IF EXISTS rider_challenge_progress CASCADE;
DROP TABLE IF EXISTS rider_challenges CASCADE;
DROP TABLE IF EXISTS loyalty_points_transactions CASCADE;
DROP TABLE IF EXISTS rider_loyalty CASCADE;
DROP TABLE IF EXISTS loyalty_tiers CASCADE;
