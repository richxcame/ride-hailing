-- Re-add legacy pricing columns to ride_types
ALTER TABLE ride_types ADD COLUMN IF NOT EXISTS base_fare DECIMAL(10,2) NOT NULL DEFAULT 3.00;
ALTER TABLE ride_types ADD COLUMN IF NOT EXISTS per_km_rate DECIMAL(10,2) NOT NULL DEFAULT 1.50;
ALTER TABLE ride_types ADD COLUMN IF NOT EXISTS per_minute_rate DECIMAL(10,2) NOT NULL DEFAULT 0.25;
ALTER TABLE ride_types ADD COLUMN IF NOT EXISTS minimum_fare DECIMAL(10,2) NOT NULL DEFAULT 5.00;

-- Drop new columns
ALTER TABLE ride_types DROP COLUMN IF EXISTS icon;
ALTER TABLE ride_types DROP COLUMN IF EXISTS sort_order;

-- Drop availability tables
DROP TABLE IF EXISTS city_ride_types;
DROP TABLE IF EXISTS country_ride_types;
