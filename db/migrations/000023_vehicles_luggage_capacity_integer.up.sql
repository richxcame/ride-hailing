ALTER TABLE vehicles ALTER COLUMN luggage_capacity DROP DEFAULT;
ALTER TABLE vehicles ALTER COLUMN luggage_capacity TYPE integer USING luggage_capacity::integer;
ALTER TABLE vehicles ALTER COLUMN luggage_capacity SET DEFAULT 2;
