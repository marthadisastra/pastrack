-- Phase 5 rule definitions and centralized default configuration.
-- No worker/runtime is enabled by this migration alone.
ALTER TABLE alert_rules ADD COLUMN code VARCHAR(64) NULL AFTER name;
CREATE UNIQUE INDEX uq_alert_rules_code ON alert_rules(code);

INSERT INTO event_severities(code,`rank`) VALUES
 ('CRITICAL',40),('WARNING',30)
ON DUPLICATE KEY UPDATE `rank`=VALUES(`rank`);

INSERT INTO alert_rules(name,code,severity_id,configuration,version,enabled)
SELECT 'BBM Kendaraan Hampir Habis','FUEL_LOW',id,
 JSON_OBJECT('threshold_percent',10,'recovery_percent',15,'minimum_samples',2,'recovery_samples',2,'freshness_seconds',300),1,1
FROM event_severities WHERE code='CRITICAL'
ON DUPLICATE KEY UPDATE name=VALUES(name),severity_id=VALUES(severity_id),configuration=VALUES(configuration),version=VALUES(version);

INSERT INTO alert_rules(name,code,severity_id,configuration,version,enabled)
SELECT 'Anomali Penurunan BBM','FUEL_DROP',id,
 JSON_OBJECT('minimum_drop_points',8,'window_seconds',300,'minimum_samples',3,'freshness_seconds',300,'stable_range_points',2,'recovery_samples',3,'readiness','NOT_READY_REQUIRES_ORIGINAL_SENSOR_TIME'),1,0
FROM event_severities WHERE code='WARNING'
ON DUPLICATE KEY UPDATE name=VALUES(name),severity_id=VALUES(severity_id),configuration=VALUES(configuration),version=VALUES(version),enabled=VALUES(enabled);

INSERT INTO alert_rules(name,code,severity_id,configuration,version,enabled)
SELECT 'Peringatan Kendaraan Melebihi Batas Kecepatan','OVER_SPEED',id,
 JSON_OBJECT('threshold_kmh',80,'recovery_kmh',75,'minimum_duration_seconds',10,'maximum_gap_seconds',15,'freshness_seconds',300,'maximum_speed_kmh',250),1,1
FROM event_severities WHERE code='WARNING'
ON DUPLICATE KEY UPDATE name=VALUES(name),severity_id=VALUES(severity_id),configuration=VALUES(configuration),version=VALUES(version);
