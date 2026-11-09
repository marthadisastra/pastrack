-- Tank profiles may be prepared before a vehicle is selected; claiming a profile is transactional.
ALTER TABLE fuel_tank_profiles MODIFY vehicle_id BIGINT UNSIGNED NULL;
