-- Preserve the existing source GPS precision; widening only, no source alteration.
ALTER TABLE gps_positions MODIFY latitude DECIMAL(10,7) NOT NULL, MODIFY longitude DECIMAL(11,7) NOT NULL;
