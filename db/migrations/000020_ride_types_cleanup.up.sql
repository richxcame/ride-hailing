-- Remove legacy pricing fields from ride_types table
-- Pricing is now fully managed through the hierarchical pricing engine (pricing_configs)
-- ride_type_id on pricing_configs already handles ride-type-specific pricing

-- Add icon and sort_order to ride_types (product metadata)
ALTER TABLE ride_types ADD COLUMN IF NOT EXISTS icon VARCHAR(100);
ALTER TABLE ride_types ADD COLUMN IF NOT EXISTS sort_order INTEGER NOT NULL DEFAULT 0;

-- Drop legacy pricing columns
ALTER TABLE ride_types DROP COLUMN IF EXISTS base_fare;
ALTER TABLE ride_types DROP COLUMN IF EXISTS per_km_rate;
ALTER TABLE ride_types DROP COLUMN IF EXISTS per_minute_rate;
ALTER TABLE ride_types DROP COLUMN IF EXISTS minimum_fare;

-- Country-level ride type availability
-- Enables a ride type for an entire country. If no city_ride_types rows exist,
-- the ride type is available in ALL cities of that country.
CREATE TABLE IF NOT EXISTS country_ride_types (
    country_id UUID NOT NULL REFERENCES countries(id) ON DELETE CASCADE,
    ride_type_id UUID NOT NULL REFERENCES ride_types(id) ON DELETE CASCADE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (country_id, ride_type_id)
);

CREATE INDEX IF NOT EXISTS idx_country_ride_types_country_active
    ON country_ride_types(country_id) WHERE is_active = true;

-- City-level ride type availability (overrides country-level)
-- When present, overrides the country-level setting for a specific city.
-- If a ride type is enabled at country level but has NO city_ride_types row,
-- it's available in all cities. A city row with is_active=false disables it for that city.
CREATE TABLE IF NOT EXISTS city_ride_types (
    city_id UUID NOT NULL REFERENCES cities(id) ON DELETE CASCADE,
    ride_type_id UUID NOT NULL REFERENCES ride_types(id) ON DELETE CASCADE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (city_id, ride_type_id)
);

CREATE INDEX IF NOT EXISTS idx_city_ride_types_city_active
    ON city_ride_types(city_id) WHERE is_active = true;

CREATE INDEX IF NOT EXISTS idx_city_ride_types_ride_type
    ON city_ride_types(ride_type_id);
