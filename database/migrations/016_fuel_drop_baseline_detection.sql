-- Evaluate fuel drop against the immediately previous valid reading.
UPDATE alert_rules
SET configuration=JSON_OBJECT(
 'minimum_drop_points',8,
 'window_seconds',900,
 'minimum_samples',2,
 'freshness_seconds',300,
 'stable_range_points',2,
 'recovery_samples',3,
 'maximum_gap_seconds',600,
 'maximum_correlation_lag_seconds',300,
 'readiness','READY_BASELINE_COMPARISON',
 'correlated_confidence','MEDIUM'),
 enabled=1,
 version=3
WHERE code='FUEL_DROP';
