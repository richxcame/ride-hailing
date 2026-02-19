ALTER TABLE vehicles ALTER COLUMN luggage_capacity DROP DEFAULT;
ALTER TABLE vehicles ALTER COLUMN luggage_capacity TYPE character varying(20) USING luggage_capacity::text;
ALTER TABLE vehicles ALTER COLUMN luggage_capacity SET DEFAULT '2';
