-- Ride Hailing Database Seed Script (HEAVY - Performance Testing)
-- This script creates a large dataset for performance testing and load simulation
-- Creates: 1000 users, 200 drivers, 5000 rides, comprehensive data across all tables

BEGIN;

-- Clear existing data (in reverse order of dependencies)
TRUNCATE TABLE
    eta_predictions, eta_model_stats,
    referrals, referral_codes,
    promo_code_uses, promo_codes,
    driver_locations, favorite_locations,
    notifications, wallet_transactions, payments, rides, wallets, drivers, users
CASCADE;

-- Generate users (500 riders, 200 drivers, 10 admins)
DO $$
DECLARE
    i INTEGER;
    user_uuid UUID;
    driver_uuid UUID;
    wallet_uuid UUID;
    rider_count INTEGER := 500;
    driver_count INTEGER := 200;
    admin_count INTEGER := 10;
BEGIN
    -- Create riders
    FOR i IN 1..rider_count LOOP
        user_uuid := gen_random_uuid();
        INSERT INTO users (id, email, phone_number, password_hash, first_name, last_name, role, is_active, is_verified, created_at) VALUES
        (
            user_uuid,
            'rider' || i || '@loadtest.com',
            '+1555' || LPAD(i::TEXT, 7, '0'),
            '$2a$10$rT5L3Z8xqH9yqYU5j5k5J.8KvH5L5Z5L5Z5L5Z5L5Z5L5Z5L5Z5L5',
            'Rider',
            'User' || i,
            'rider',
            true,
            i % 10 != 0, -- 90% verified
            NOW() - (random() * INTERVAL '365 days')
        );

        -- Create wallet for rider
        INSERT INTO wallets (id, user_id, balance, currency) VALUES
        (
            gen_random_uuid(),
            user_uuid,
            (random() * 500)::DECIMAL(10,2), -- $0-500 balance
            'USD'
        );

        -- Create favorite locations (1-3 per user)
        FOR j IN 1..(1 + floor(random() * 3)::INT) LOOP
            INSERT INTO favorite_locations (id, user_id, name, latitude, longitude, address) VALUES
            (
                gen_random_uuid(),
                user_uuid,
                ARRAY['Home', 'Work', 'Gym', 'Airport', 'Mall'][j],
                37.7 + (random() * 0.2),
                -122.5 + (random() * 0.2),
                j || ' Favorite St, San Francisco, CA 94102'
            );
        END LOOP;
    END LOOP;

    -- Create drivers
    FOR i IN 1..driver_count LOOP
        user_uuid := gen_random_uuid();
        driver_uuid := gen_random_uuid();

        INSERT INTO users (id, email, phone_number, password_hash, first_name, last_name, role, is_active, is_verified, created_at) VALUES
        (
            user_uuid,
            'driver' || i || '@loadtest.com',
            '+1666' || LPAD(i::TEXT, 7, '0'),
            '$2a$10$rT5L3Z8xqH9yqYU5j5k5J.8KvH5L5Z5L5Z5L5Z5L5Z5L5Z5L5Z5L5',
            'Driver',
            'Pro' || i,
            'driver',
            true,
            true,
            NOW() - (random() * INTERVAL '730 days')
        );

        -- Create driver profile
        INSERT INTO drivers (id, user_id, license_number, vehicle_model, vehicle_plate, vehicle_color, vehicle_year, is_available, is_online, rating, total_rides, current_latitude, current_longitude, last_location_update) VALUES
        (
            driver_uuid,
            user_uuid,
            'DL' || LPAD(i::TEXT, 10, '0'),
            ARRAY['Toyota Camry', 'Honda Accord', 'Tesla Model 3', 'Chevrolet Malibu', 'Ford Fusion', 'Nissan Altima'][1 + floor(random() * 6)::INT],
            ARRAY['ABC', 'XYZ', 'QRS', 'TUV', 'WXY'][1 + floor(random() * 5)::INT] || LPAD(i::TEXT, 3, '0'),
            ARRAY['Black', 'White', 'Silver', 'Blue', 'Red', 'Gray'][1 + floor(random() * 6)::INT],
            2018 + floor(random() * 6)::INT, -- 2018-2023
            random() > 0.3, -- 70% available
            random() > 0.4, -- 60% online
            4.0 + (random() * 1.0)::DECIMAL(3,2), -- 4.0-5.0 rating
            floor(random() * 1000)::INT, -- 0-1000 rides
            37.7 + (random() * 0.2), -- SF area
            -122.5 + (random() * 0.2),
            NOW() - (random() * INTERVAL '1 hour')
        );

        -- Create wallet for driver
        INSERT INTO wallets (id, user_id, balance, currency) VALUES
        (
            gen_random_uuid(),
            user_uuid,
            (500 + random() * 5000)::DECIMAL(10,2), -- $500-5500 balance
            'USD'
        );

        -- Create location history (10-50 points per driver)
        FOR j IN 1..(10 + floor(random() * 40)::INT) LOOP
            INSERT INTO driver_locations (id, driver_id, latitude, longitude, accuracy, speed, heading, recorded_at) VALUES
            (
                gen_random_uuid(),
                driver_uuid,
                37.7 + (random() * 0.2),
                -122.5 + (random() * 0.2),
                (5 + random() * 10)::DECIMAL(10,2),
                (10 + random() * 50)::DECIMAL(10,2),
                (random() * 360)::DECIMAL(5,2),
                NOW() - (random() * INTERVAL '24 hours')
            );
        END LOOP;
    END LOOP;

    -- Create admins
    FOR i IN 1..admin_count LOOP
        INSERT INTO users (id, email, phone_number, password_hash, first_name, last_name, role, is_active, is_verified, created_at) VALUES
        (
            gen_random_uuid(),
            'admin' || i || '@loadtest.com',
            '+1777' || LPAD(i::TEXT, 7, '0'),
            '$2a$10$rT5L3Z8xqH9yqYU5j5k5J.8KvH5L5Z5L5Z5L5Z5L5Z5L5Z5L5Z5L5',
            'Admin',
            'User' || i,
            'admin',
            true,
            true,
            NOW() - (random() * INTERVAL '1000 days')
        );
    END LOOP;

    RAISE NOTICE 'Created % riders, % drivers, % admins', rider_count, driver_count, admin_count;
END $$;

-- Generate promo codes (50 codes)
DO $$
DECLARE
    i INTEGER;
BEGIN
    FOR i IN 1..50 LOOP
        INSERT INTO promo_codes (id, code, discount_type, discount_value, max_uses, total_uses, expiry_date, is_active) VALUES
        (
            gen_random_uuid(),
            'PROMO' || LPAD(i::TEXT, 4, '0'),
            CASE WHEN random() > 0.5 THEN 'percentage' ELSE 'fixed' END,
            CASE WHEN random() > 0.5 THEN (5 + random() * 20)::DECIMAL(10,2) ELSE (10 + random() * 30)::DECIMAL(10,2) END,
            floor(100 + random() * 900)::INT,
            floor(random() * 50)::INT,
            NOW() + (random() * INTERVAL '180 days'),
            random() > 0.2 -- 80% active
        );
    END LOOP;
    RAISE NOTICE 'Created % promo codes', 50;
END $$;

-- Generate referral codes and referrals (for 20% of riders)
DO $$
DECLARE
    user_record RECORD;
    referral_code_uuid UUID;
    referred_user_uuid UUID;
    referrer_count INTEGER := 0;
BEGIN
    FOR user_record IN (
        SELECT id FROM users WHERE role = 'rider' AND random() < 0.2 LIMIT 100
    ) LOOP
        referral_code_uuid := gen_random_uuid();

        -- Create referral code
        INSERT INTO referral_codes (id, user_id, code, total_uses, bonus_amount) VALUES
        (
            referral_code_uuid,
            user_record.id,
            'REF' || substr(user_record.id::TEXT, 1, 6) || LPAD(referrer_count::TEXT, 2, '0'),
            floor(random() * 10)::INT,
            10.00
        );

        -- Create 1-5 referrals for this code
        FOR j IN 1..(1 + floor(random() * 5)::INT) LOOP
            SELECT id INTO referred_user_uuid FROM users WHERE role = 'rider' AND id != user_record.id ORDER BY random() LIMIT 1;

            IF referred_user_uuid IS NOT NULL THEN
                INSERT INTO referrals (id, referrer_id, referred_id, referral_code_id, bonus_paid) VALUES
                (
                    gen_random_uuid(),
                    user_record.id,
                    referred_user_uuid,
                    referral_code_uuid,
                    random() > 0.5
                );
            END IF;
        END LOOP;

        referrer_count := referrer_count + 1;
    END LOOP;
    RAISE NOTICE 'Created referral codes and referrals for % users', referrer_count;
END $$;

-- Generate rides (5000 rides with realistic distribution)
DO $$
DECLARE
    i INTEGER;
    rider_uuid UUID;
    driver_uuid UUID;
    ride_uuid UUID;
    payment_uuid UUID;
    ride_status TEXT;
    pickup_lat DECIMAL(10,8);
    pickup_lon DECIMAL(10,8);
    dropoff_lat DECIMAL(10,8);
    dropoff_lon DECIMAL(10,8);
    distance DECIMAL(10,2);
    duration INTEGER;
    fare DECIMAL(10,2);
    surge DECIMAL(3,2);
    requested_time TIMESTAMP;
    accepted_time TIMESTAMP;
    started_time TIMESTAMP;
    completed_time TIMESTAMP;
    ride_count INTEGER := 5000;
    completed_count INTEGER := 0;
    cancelled_count INTEGER := 0;
BEGIN
    FOR i IN 1..ride_count LOOP
        -- Select random rider and driver
        SELECT id INTO rider_uuid FROM users WHERE role = 'rider' ORDER BY random() LIMIT 1;
        SELECT user_id INTO driver_uuid FROM drivers WHERE is_available = true ORDER BY random() LIMIT 1;

        -- Generate ride status with realistic distribution
        -- 75% completed, 15% cancelled, 5% in_progress, 3% accepted, 2% requested
        ride_status := CASE
            WHEN random() < 0.75 THEN 'completed'
            WHEN random() < 0.90 THEN 'cancelled'
            WHEN random() < 0.95 THEN 'in_progress'
            WHEN random() < 0.98 THEN 'accepted'
            ELSE 'requested'
        END;

        -- Generate realistic coordinates (SF Bay Area)
        pickup_lat := 37.7 + (random() * 0.2);
        pickup_lon := -122.5 + (random() * 0.2);
        dropoff_lat := 37.7 + (random() * 0.2);
        dropoff_lon := -122.5 + (random() * 0.2);

        -- Calculate distance and duration
        distance := (1 + random() * 15)::DECIMAL(10,2); -- 1-16 miles
        duration := (distance * 2.5 + random() * 10)::INT; -- roughly 2.5 min per mile + variability
        surge := (1 + random() * 0.5)::DECIMAL(3,2); -- 1.0-1.5x surge
        fare := ((distance * 2.5 + 5) * surge)::DECIMAL(10,2); -- base $5 + $2.5/mile

        -- Generate timestamps
        requested_time := NOW() - (random() * INTERVAL '60 days');
        accepted_time := CASE WHEN ride_status != 'requested' THEN requested_time + (random() * INTERVAL '5 minutes') ELSE NULL END;
        started_time := CASE WHEN ride_status IN ('in_progress', 'completed') THEN accepted_time + (random() * INTERVAL '10 minutes') ELSE NULL END;
        completed_time := CASE WHEN ride_status = 'completed' THEN started_time + (duration || ' minutes')::INTERVAL ELSE NULL END;

        ride_uuid := gen_random_uuid();

        -- Insert ride
        INSERT INTO rides (
            id, rider_id, driver_id, status,
            pickup_latitude, pickup_longitude, pickup_address,
            dropoff_latitude, dropoff_longitude, dropoff_address,
            estimated_distance, estimated_duration, estimated_fare,
            actual_distance, actual_duration, final_fare,
            surge_multiplier, requested_at, accepted_at, started_at, completed_at,
            rating, feedback
        ) VALUES (
            ride_uuid, rider_uuid,
            CASE WHEN ride_status = 'requested' THEN NULL ELSE driver_uuid END,
            ride_status,
            pickup_lat, pickup_lon, floor(random() * 1000) || ' Market St, San Francisco, CA',
            dropoff_lat, dropoff_lon, floor(random() * 1000) || ' Mission St, San Francisco, CA',
            distance, duration, fare,
            CASE WHEN ride_status = 'completed' THEN (distance + random() * 0.5)::DECIMAL(10,2) ELSE NULL END,
            CASE WHEN ride_status = 'completed' THEN (duration + (random() * 5)::INT) ELSE NULL END,
            CASE WHEN ride_status = 'completed' THEN (fare + (random() * 2 - 1)::DECIMAL(10,2)) ELSE NULL END,
            surge, requested_time, accepted_time, started_time, completed_time,
            CASE WHEN ride_status = 'completed' AND random() > 0.2 THEN (3 + random() * 2)::INT ELSE NULL END,
            CASE WHEN ride_status = 'completed' AND random() > 0.7 THEN ARRAY['Great!', 'Good ride', 'Nice driver', 'Excellent', 'Fast trip'][1 + floor(random() * 5)::INT] ELSE NULL END
        );

        -- Create payment for completed rides
        IF ride_status = 'completed' THEN
            completed_count := completed_count + 1;

            INSERT INTO payments (id, ride_id, rider_id, driver_id, amount, commission, driver_earnings, method, status, transaction_id, processed_at) VALUES
            (
                gen_random_uuid(),
                ride_uuid,
                rider_uuid,
                driver_uuid,
                fare,
                (fare * 0.25)::DECIMAL(10,2), -- 25% commission
                (fare * 0.75)::DECIMAL(10,2),
                ARRAY['card', 'wallet', 'cash'][1 + floor(random() * 3)::INT],
                CASE WHEN random() > 0.05 THEN 'completed' ELSE 'failed' END,
                CASE WHEN random() > 0.3 THEN 'stripe_ch_' || substr(md5(random()::TEXT), 1, 12) ELSE NULL END,
                completed_time + INTERVAL '5 minutes'
            );

            -- Create wallet transactions for wallet payments
            IF random() > 0.7 THEN
                -- Debit from rider
                INSERT INTO wallet_transactions (id, wallet_id, transaction_type, amount, description, related_ride_id, created_at) VALUES
                (
                    gen_random_uuid(),
                    (SELECT id FROM wallets WHERE user_id = rider_uuid),
                    'debit',
                    fare,
                    'Payment for ride',
                    ride_uuid,
                    completed_time
                );

                -- Credit to driver
                INSERT INTO wallet_transactions (id, wallet_id, transaction_type, amount, description, related_ride_id, created_at) VALUES
                (
                    gen_random_uuid(),
                    (SELECT id FROM wallets WHERE user_id = driver_uuid),
                    'credit',
                    (fare * 0.75)::DECIMAL(10,2),
                    'Earnings from ride',
                    ride_uuid,
                    completed_time
                );
            END IF;

            -- Create ML ETA prediction for some rides
            IF random() > 0.5 THEN
                INSERT INTO eta_predictions (id, ride_id, predicted_duration, actual_duration, features, model_version, created_at) VALUES
                (
                    gen_random_uuid(),
                    ride_uuid,
                    duration,
                    (duration + (random() * 5)::INT),
                    jsonb_build_object(
                        'distance', distance,
                        'hour', EXTRACT(HOUR FROM requested_time),
                        'day_of_week', EXTRACT(DOW FROM requested_time),
                        'surge', surge
                    ),
                    'v1.0',
                    requested_time
                );
            END IF;
        ELSIF ride_status = 'cancelled' THEN
            cancelled_count := cancelled_count + 1;
        END IF;

        -- Create notifications
        INSERT INTO notifications (id, user_id, type, channel, title, message, is_read, created_at) VALUES
        (
            gen_random_uuid(),
            rider_uuid,
            'ride_status',
            ARRAY['push', 'sms', 'email'][1 + floor(random() * 3)::INT],
            'Ride ' || ride_status,
            'Your ride is ' || ride_status,
            random() > 0.3,
            requested_time
        );

        -- Progress indicator
        IF i % 500 = 0 THEN
            RAISE NOTICE 'Created % rides (% completed, % cancelled)', i, completed_count, cancelled_count;
        END IF;
    END LOOP;

    RAISE NOTICE 'Total: % rides created (% completed, % cancelled)', ride_count, completed_count, cancelled_count;
END $$;

-- Create some ML model statistics
INSERT INTO eta_model_stats (id, model_version, training_date, num_samples, mae, rmse, r_squared) VALUES
(gen_random_uuid(), 'v1.0', NOW() - INTERVAL '30 days', 10000, 2.5, 3.2, 0.85),
(gen_random_uuid(), 'v1.1', NOW() - INTERVAL '15 days', 15000, 2.2, 2.9, 0.88),
(gen_random_uuid(), 'v1.2', NOW() - INTERVAL '7 days', 20000, 2.0, 2.6, 0.90);

-- Refresh materialized views
REFRESH MATERIALIZED VIEW CONCURRENTLY driver_statistics;
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_demand_zones;
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_driver_performance;
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_revenue_metrics;

COMMIT;

-- Verify data was inserted
SELECT 'Summary' as info, '==========' as value
UNION ALL
SELECT 'Users', COUNT(*)::TEXT FROM users
UNION ALL
SELECT 'Riders', COUNT(*)::TEXT FROM users WHERE role = 'rider'
UNION ALL
SELECT 'Drivers', COUNT(*)::TEXT FROM users WHERE role = 'driver'
UNION ALL
SELECT 'Admins', COUNT(*)::TEXT FROM users WHERE role = 'admin'
UNION ALL
SELECT 'Driver Profiles', COUNT(*)::TEXT FROM drivers
UNION ALL
SELECT 'Wallets', COUNT(*)::TEXT FROM wallets
UNION ALL
SELECT 'Rides', COUNT(*)::TEXT FROM rides
UNION ALL
SELECT 'Completed Rides', COUNT(*)::TEXT FROM rides WHERE status = 'completed'
UNION ALL
SELECT 'Payments', COUNT(*)::TEXT FROM payments
UNION ALL
SELECT 'Promo Codes', COUNT(*)::TEXT FROM promo_codes
UNION ALL
SELECT 'Referral Codes', COUNT(*)::TEXT FROM referral_codes
UNION ALL
SELECT 'Notifications', COUNT(*)::TEXT FROM notifications
UNION ALL
SELECT 'Driver Locations', COUNT(*)::TEXT FROM driver_locations
UNION ALL
SELECT 'ETA Predictions', COUNT(*)::TEXT FROM eta_predictions;
