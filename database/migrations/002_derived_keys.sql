-- Derived records can be recomputed without duplicating facts.
CREATE UNIQUE INDEX uq_trips_source_version ON trips(vehicle_id,start_position_id,calculation_version);
CREATE UNIQUE INDEX uq_geofence_transition ON geofence_events(geofence_id,position_id,transition);
CREATE UNIQUE INDEX uq_fuel_event_source ON fuel_events(fuel_sensor_id,start_reading_id,end_reading_id,event_type,rule_version);
