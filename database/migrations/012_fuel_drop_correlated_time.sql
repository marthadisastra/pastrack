-- Phase 5.1: enable conservative same-raw GPS correlation for FUEL_DROP.
UPDATE alert_rules
SET configuration=JSON_OBJECT(
 'minimum_drop_points',8,
 'window_seconds',300,
 'minimum_samples',3,
 'freshness_seconds',300,
 'stable_range_points',2,
 'recovery_samples',3,
 'maximum_gap_seconds',180,
 'maximum_correlation_lag_seconds',300,
 'readiness','READY_WITH_VALIDATED_CORRELATED_TIME',
 'correlated_confidence','MEDIUM'),
 enabled=1,
 version=2
WHERE code='FUEL_DROP';
