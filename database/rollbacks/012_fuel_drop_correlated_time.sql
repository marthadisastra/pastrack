-- Development rollback: return FUEL_DROP to the Phase 5 safety state.
UPDATE alert_rules SET enabled=0,version=1,
 configuration=JSON_OBJECT('minimum_drop_points',8,'window_seconds',300,'minimum_samples',3,'freshness_seconds',300,'stable_range_points',2,'recovery_samples',3,'readiness','NOT_READY_REQUIRES_ORIGINAL_SENSOR_TIME')
WHERE code='FUEL_DROP';
