-- Add unique index to mv_demand_zones for concurrent refresh
-- Need to add a unique constraint for REFRESH MATERIALIZED VIEW CONCURRENTLY

-- First, add a unique identifier column by combining zone and time dimensions
CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_demand_zones_unique 
ON mv_demand_zones(zone_point, hour_of_day, day_of_week);
