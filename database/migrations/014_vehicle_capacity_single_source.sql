-- Vehicle inventory is the only tank-capacity authority.
-- Preserve the relative sensor level while rescaling stored visual liters.
UPDATE fuel_consumption_summaries s
JOIN vehicle_inventory vi ON vi.vehicle_id=s.vehicle_id AND vi.tank_capacity_l>0
JOIN fuel_tank_profiles tp ON tp.id=(
 SELECT tp2.id FROM fuel_tank_profiles tp2
 WHERE tp2.vehicle_id=s.vehicle_id AND tp2.capacity_l>0
 ORDER BY tp2.effective_from DESC,tp2.id DESC LIMIT 1
)
SET s.consumed_l=s.consumed_l*vi.tank_capacity_l/tp.capacity_l
WHERE s.consumed_l IS NOT NULL;

UPDATE fuel_readings f
JOIN vehicle_inventory vi ON vi.vehicle_id=f.vehicle_id AND vi.tank_capacity_l>0
SET f.fuel_liters=LEAST(vi.tank_capacity_l,GREATEST(0,f.fuel_percent*vi.tank_capacity_l/100))
WHERE f.fuel_percent IS NOT NULL AND f.validity='valid';

UPDATE fuel_events e
JOIN fuel_readings start_reading ON start_reading.id=e.start_reading_id
JOIN fuel_readings end_reading ON end_reading.id=e.end_reading_id
SET e.delta_l=end_reading.fuel_liters-start_reading.fuel_liters
WHERE start_reading.fuel_liters IS NOT NULL AND end_reading.fuel_liters IS NOT NULL;

ALTER TABLE sensor_calibration_sessions DROP COLUMN capacity_l;
ALTER TABLE fuel_tank_profiles DROP COLUMN capacity_l;
