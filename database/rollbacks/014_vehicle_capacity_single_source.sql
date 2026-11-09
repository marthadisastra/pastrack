-- Structural compatibility rollback only. Original duplicate values are not recreated.
ALTER TABLE fuel_tank_profiles ADD COLUMN capacity_l DECIMAL(12,3) NULL AFTER name;
UPDATE fuel_tank_profiles tp JOIN vehicle_inventory vi ON vi.vehicle_id=tp.vehicle_id SET tp.capacity_l=vi.tank_capacity_l;
ALTER TABLE sensor_calibration_sessions ADD COLUMN capacity_l DECIMAL(12,3) NULL AFTER quantity;
UPDATE sensor_calibration_sessions s JOIN vehicle_inventory vi ON vi.vehicle_id=s.vehicle_id SET s.capacity_l=vi.tank_capacity_l;
