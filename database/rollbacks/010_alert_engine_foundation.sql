-- Development rollback only. Back up and verify no Phase 4 data before use.
DROP TABLE IF EXISTS alert_audit_trail;
DROP TABLE IF EXISTS alert_event_sources;
DROP TABLE IF EXISTS alert_rule_states;
DROP TABLE IF EXISTS alert_job_history;
DROP TABLE IF EXISTS alert_evaluation_jobs;
DROP TABLE IF EXISTS alert_intake_rejections;
DROP TABLE IF EXISTS alert_input_events;
DROP INDEX ix_alert_vehicle_state ON alert_events;
DROP INDEX ix_alert_vehicle_history ON alert_events;
DROP INDEX uq_alert_episode_key ON alert_events;
ALTER TABLE alert_events DROP FOREIGN KEY fk_alert_events_false_positive_by;
ALTER TABLE alert_events DROP FOREIGN KEY fk_alert_events_primary_sensor;
ALTER TABLE alert_events DROP FOREIGN KEY fk_alert_events_primary_device;
ALTER TABLE alert_events DROP CHECK ck_alert_vehicle_required;
ALTER TABLE alert_events DROP COLUMN row_version, DROP COLUMN false_positive_reason, DROP COLUMN false_positive_at, DROP COLUMN false_positive_by, DROP COLUMN longitude, DROP COLUMN latitude, DROP COLUMN primary_sensor_id, DROP COLUMN primary_device_id, DROP COLUMN last_detected_at, DROP COLUMN scope_key, DROP COLUMN episode_key, DROP COLUMN description, DROP COLUMN title, DROP COLUMN severity, DROP COLUMN rule_version, DROP COLUMN rule_code;
