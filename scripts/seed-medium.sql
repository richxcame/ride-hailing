-- Ride Hailing Database Seed Script (MEDIUM - Realistic Testing)
-- This script creates a moderate dataset for testing common scenarios
-- Creates: 50 users, 20 drivers, 200 rides

BEGIN;

-- Clear existing data
TRUNCATE TABLE
    eta_predictions, eta_model_stats,
    referrals, referral_codes,
    promo_code_uses, promo_codes,
    driver_locations, favorite_locations,
    notifications, wallet_transactions, payments, rides, wallets, drivers, users
CASCADE;

-- Generate users (30 riders, 20 drivers, 2 admins)
DO $$
DECLARE
    i INTEGER;
    user_uuid UUID;
    driver_uuid UUID;
BEGIN
    -- Create 30 riders
    FOR i IN 1..30 LOOP
        user_uuid := gen_random_uuid();
        INSERT INTO users (id, email, phone_number, password_hash, first_name, last_name, role, is_active, is_verified, created_at) VALUES
        (
            user_uuid,
            'rider' || i || '@test.com',
            '+1555' || LPAD(i::TEXT, 7, '0'),
            '$2a$10$rT5L3Z8xqH9yqYU5j5k5J.8KvH5L5Z5L5Z5L5Z5L5Z5L5Z5L5Z5L5',
            ARRAY['Alice', 'Bob', 'Carol', 'David', 'Eve', 'Frank', 'Grace', 'Henry', 'Ivy', 'Jack'][1 + (i % 10)],
            'User' || i,
            'rider',
            true,
            i % 5 != 0, -- 80% verified
            NOW() - (random() * INTERVAL '180 days')
        );

        INSERT INTO wallets (id, user_id, balance, currency) VALUES
        (gen_random_uuid(), user_uuid, (50 + random() * 200)::DECIMAL(10,2), 'USD');

        -- Add 1-2 favorite locations
        FOR j IN 1..(1 + (i % 2)) LOOP
            INSERT INTO favorite_locations (id, user_id, name, latitude, longitude, address) VALUES
            (
                gen_random_uuid(), user_uuid,
                ARRAY['Home', 'Work', 'Gym'][j],
                37.7 + (random() * 0.1),
                -122.5 + (random() * 0.1),
                j * 100 || ' Test St, San Francisco, CA 94102'
            );
        END LOOP;
    END LOOP;

    -- Create 20 drivers
    FOR i IN 1..20 LOOP
        user_uuid := gen_random_uuid();
        driver_uuid := gen_random_uuid();

        INSERT INTO users (id, email, phone_number, password_hash, first_name, last_name, role, is_active, is_verified, created_at) VALUES
        (
            user_uuid,
            'driver' || i || '@test.com',
            '+1666' || LPAD(i::TEXT, 7, '0'),
            '$2a$10$rT5L3Z8xqH9yqYU5j5k5J.8KvH5L5Z5L5Z5L5Z5L5Z5L5Z5L5Z5L5',
            'Driver',
            'User' || i,
            'driver',
            true,
            true,
            NOW() - (random() * INTERVAL '365 days')
        );

        INSERT INTO drivers (id, user_id, license_number, vehicle_model, vehicle_plate, vehicle_color, vehicle_year, is_available, is_online, rating, total_rides, current_latitude, current_longitude, last_location_update) VALUES
        (
            driver_uuid, user_uuid,
            'DL' || LPAD(i::TEXT, 8, '0'),
            ARRAY['Toyota Camry', 'Honda Accord', 'Tesla Model 3'][1 + (i % 3)],
            'TST' || LPAD(i::TEXT, 3, '0'),
            ARRAY['Black', 'White', 'Silver'][1 + (i % 3)],
            2019 + (i % 5),
            i % 4 != 0, -- 75% available
            i % 3 != 0, -- 66% online
            4.0 + (random() * 1.0)::DECIMAL(3,2),
            floor(random() * 200)::INT,
            37.7 + (random() * 0.1),
            -122.5 + (random() * 0.1),
            NOW() - (random() * INTERVAL '30 minutes')
        );

        INSERT INTO wallets (id, user_id, balance, currency) VALUES
        (gen_random_uuid(), user_uuid, (500 + random() * 2000)::DECIMAL(10,2), 'USD');

        -- Add location history (5-10 points)
        FOR j IN 1..(5 + (i % 6)) LOOP
            INSERT INTO driver_locations (id, driver_id, latitude, longitude, accuracy, speed, heading, recorded_at) VALUES
            (
                gen_random_uuid(), driver_uuid,
                37.7 + (random() * 0.1),
                -122.5 + (random() * 0.1),
                (5 + random() * 5)::DECIMAL(10,2),
                (15 + random() * 30)::DECIMAL(10,2),
                (random() * 360)::DECIMAL(5,2),
                NOW() - (random() * INTERVAL '2 hours')
            );
        END LOOP;
    END LOOP;

    -- Create 2 admins
    FOR i IN 1..2 LOOP
        INSERT INTO users (id, email, phone_number, password_hash, first_name, last_name, role, is_active, is_verified, created_at) VALUES
        (
            gen_random_uuid(),
            'admin' || i || '@test.com',
            '+1777' || LPAD(i::TEXT, 7, '0'),
            '$2a$10$rT5L3Z8xqH9yqYU5j5k5J.8KvH5L5Z5L5Z5L5Z5L5Z5L5Z5L5Z5L5',
            'Admin',
            'User' || i,
            'admin',
            true,
            true,
            NOW() - INTERVAL '500 days'
        );
    END LOOP;

    RAISE NOTICE 'Created 30 riders, 20 drivers, 2 admins';
END $$;

-- Create promo codes
INSERT INTO promo_codes (id, code, discount_type, discount_value, max_uses, total_uses, expiry_date, is_active) VALUES
(gen_random_uuid(), 'WELCOME10', 'fixed', 10.00, 1000, 45, NOW() + INTERVAL '90 days', true),
(gen_random_uuid(), 'SAVE20', 'percentage', 20.00, 500, 123, NOW() + INTERVAL '60 days', true),
(gen_random_uuid(), 'FIRST RIDE', 'fixed', 15.00, 100, 67, NOW() + INTERVAL '30 days', true),
(gen_random_uuid(), 'EXPIRED', 'fixed', 25.00, 50, 50, NOW() - INTERVAL '10 days', false);

-- Generate 200 rides with realistic distribution
DO $$
DECLARE
    i INTEGER;
    rider_uuid UUID;
    driver_uuid UUID;
    ride_uuid UUID;
    ride_status TEXT;
    distance DECIMAL(10,2);
    duration INTEGER;
    fare DECIMAL(10,2);
    surge DECIMAL(3,2);
    requested_time TIMESTAMP;
BEGIN
    FOR i IN 1..200 LOOP
        SELECT id INTO rider_uuid FROM users WHERE role = 'rider' ORDER BY random() LIMIT 1;
        SELECT user_id INTO driver_uuid FROM drivers WHERE is_available = true ORDER BY random() LIMIT 1;

        -- 70% completed, 20% cancelled, 5% in_progress, 3% accepted, 2% requested
        ride_status := CASE
            WHEN random() < 0.70 THEN 'completed'
            WHEN random() < 0.90 THEN 'cancelled'
            WHEN random() < 0.95 THEN 'in_progress'
            WHEN random() < 0.98 THEN 'accepted'
            ELSE 'requested'
        END;

        distance := (1 + random() * 10)::DECIMAL(10,2);
        duration := (distance * 2.5 + random() * 5)::INT;
        surge := (1 + random() * 0.3)::DECIMAL(3,2);
        fare := ((distance * 2.5 + 5) * surge)::DECIMAL(10,2);
        requested_time := NOW() - (random() * INTERVAL '30 days');

        ride_uuid := gen_random_uuid();

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
            37.7 + (random() * 0.1), -122.5 + (random() * 0.1), floor(random() * 500) || ' Market St, SF, CA',
            37.7 + (random() * 0.1), -122.5 + (random() * 0.1), floor(random() * 500) || ' Mission St, SF, CA',
            distance, duration, fare,
            CASE WHEN ride_status = 'completed' THEN (distance + random() * 0.3)::DECIMAL(10,2) ELSE NULL END,
            CASE WHEN ride_status = 'completed' THEN (duration + (random() * 3)::INT) ELSE NULL END,
            CASE WHEN ride_status = 'completed' THEN (fare + (random() - 0.5)::DECIMAL(10,2)) ELSE NULL END,
            surge, requested_time,
            CASE WHEN ride_status != 'requested' THEN requested_time + (random() * INTERVAL '3 minutes') ELSE NULL END,
            CASE WHEN ride_status IN ('in_progress', 'completed') THEN requested_time + (random() * INTERVAL '8 minutes') ELSE NULL END,
            CASE WHEN ride_status = 'completed' THEN requested_time + (duration || ' minutes')::INTERVAL ELSE NULL END,
            CASE WHEN ride_status = 'completed' AND random() > 0.2 THEN (3 + random() * 2)::INT ELSE NULL END,
            CASE WHEN ride_status = 'completed' AND random() > 0.6 THEN ARRAY['Great!', 'Good', 'Nice'][1 + floor(random() * 3)::INT] ELSE NULL END
        );

        -- Create payment for completed rides
        IF ride_status = 'completed' THEN
            INSERT INTO payments (id, ride_id, rider_id, driver_id, amount, commission, driver_earnings, method, status, transaction_id, processed_at) VALUES
            (
                gen_random_uuid(), ride_uuid, rider_uuid, driver_uuid,
                fare, (fare * 0.25)::DECIMAL(10,2), (fare * 0.75)::DECIMAL(10,2),
                ARRAY['card', 'wallet', 'cash'][1 + floor(random() * 3)::INT],
                CASE WHEN random() > 0.02 THEN 'completed' ELSE 'failed' END,
                'tx_' || substr(md5(random()::TEXT), 1, 10),
                requested_time + (duration || ' minutes')::INTERVAL + INTERVAL '2 minutes'
            );
        END IF;

        IF i % 50 = 0 THEN
            RAISE NOTICE 'Created % rides', i;
        END IF;
    END LOOP;
END $$;

-- Add some notifications
INSERT INTO notifications (id, user_id, type, channel, title, message, is_read, created_at)
SELECT
    gen_random_uuid(),
    rider_id,
    'ride_status',
    'push',
    'Ride ' || status,
    'Your ride is ' || status,
    random() > 0.4,
    requested_at
FROM rides
LIMIT 100;

-- Refresh materialized views
REFRESH MATERIALIZED VIEW CONCURRENTLY driver_statistics;
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_demand_zones;
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_driver_performance;
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_revenue_metrics;

COMMIT;

-- Summary
SELECT 'Medium Dataset Summary' as info, '==================' as value
UNION ALL SELECT 'Total Users', COUNT(*)::TEXT FROM users
UNION ALL SELECT 'Riders', COUNT(*)::TEXT FROM users WHERE role = 'rider'
UNION ALL SELECT 'Drivers', COUNT(*)::TEXT FROM drivers
UNION ALL SELECT 'Rides', COUNT(*)::TEXT FROM rides
UNION ALL SELECT 'Completed Rides', COUNT(*)::TEXT FROM rides WHERE status = 'completed'
UNION ALL SELECT 'Payments', COUNT(*)::TEXT FROM payments
UNION ALL SELECT 'Promo Codes', COUNT(*)::TEXT FROM promo_codes
UNION ALL SELECT 'Notifications', COUNT(*)::TEXT FROM notifications;
