-- Phase 4 alert infrastructure. No business rules are seeded or enabled.

ALTER TABLE alert_events ADD COLUMN rule_code VARCHAR(64) NULL AFTER rule_id;
ALTER TABLE alert_events ADD COLUMN rule_version BIGINT NULL AFTER rule_code;
ALTER TABLE alert_events ADD COLUMN severity VARCHAR(24) NULL AFTER rule_version;
ALTER TABLE alert_events ADD COLUMN title VARCHAR(120) NULL AFTER severity;
ALTER TABLE alert_events ADD COLUMN description VARCHAR(1000) NULL AFTER title;
ALTER TABLE alert_events ADD COLUMN episode_key CHAR(64) NULL AFTER state;
ALTER TABLE alert_events ADD COLUMN scope_key VARCHAR(120) NULL AFTER episode_key;
ALTER TABLE alert_events ADD COLUMN last_detected_at DATETIME(3) NULL AFTER raised_at;
ALTER TABLE alert_events ADD COLUMN primary_device_id BIGINT UNSIGNED NULL AFTER vehicle_id;
ALTER TABLE alert_events ADD COLUMN primary_sensor_id BIGINT UNSIGNED NULL AFTER primary_device_id;
ALTER TABLE alert_events ADD COLUMN latitude DECIMAL(9,6) NULL AFTER primary_sensor_id;
ALTER TABLE alert_events ADD COLUMN longitude DECIMAL(10,6) NULL AFTER latitude;
ALTER TABLE alert_events ADD COLUMN false_positive_by BIGINT UNSIGNED NULL AFTER resolved_at;
ALTER TABLE alert_events ADD COLUMN false_positive_at DATETIME(3) NULL AFTER false_positive_by;
ALTER TABLE alert_events ADD COLUMN false_positive_reason VARCHAR(500) NULL AFTER false_positive_at;
ALTER TABLE alert_events ADD COLUMN row_version BIGINT NOT NULL DEFAULT 1 AFTER false_positive_reason;
ALTER TABLE alert_events ADD CONSTRAINT ck_alert_vehicle_required CHECK (vehicle_id IS NOT NULL);
ALTER TABLE alert_events ADD CONSTRAINT fk_alert_events_primary_device FOREIGN KEY (primary_device_id) REFERENCES gps_devices(id);
ALTER TABLE alert_events ADD CONSTRAINT fk_alert_events_primary_sensor FOREIGN KEY (primary_sensor_id) REFERENCES ble_sensors(id);
ALTER TABLE alert_events ADD CONSTRAINT fk_alert_events_false_positive_by FOREIGN KEY (false_positive_by) REFERENCES users(id);
CREATE UNIQUE INDEX uq_alert_episode_key ON alert_events(episode_key);
CREATE INDEX ix_alert_vehicle_history ON alert_events(vehicle_id,raised_at,id);
CREATE INDEX ix_alert_vehicle_state ON alert_events(vehicle_id,state,raised_at,id);

CREATE TABLE IF NOT EXISTS alert_input_events (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 event_key BINARY(32) NOT NULL,
 event_type VARCHAR(64) NOT NULL,
 vehicle_id BIGINT UNSIGNED NOT NULL,
 source_type VARCHAR(32) NOT NULL,
 source_identifier VARCHAR(64) NOT NULL,
 device_id BIGINT UNSIGNED NULL,
 sensor_id BIGINT UNSIGNED NULL,
 gps_position_id BIGINT UNSIGNED NULL,
 fuel_reading_id BIGINT UNSIGNED NULL,
 device_event_id BIGINT UNSIGNED NULL,
 sensor_telemetry_id BIGINT UNSIGNED NULL,
 raw_message_id BIGINT UNSIGNED NULL,
 event_time DATETIME(3) NULL,
 received_at DATETIME(3) NOT NULL,
 timestamp_quality VARCHAR(32) NOT NULL,
 resolution_status VARCHAR(24) NOT NULL,
 payload JSON NOT NULL,
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 UNIQUE KEY uq_alert_input_event_key(event_key),
 KEY ix_alert_input_vehicle_time(vehicle_id,event_time,id),
 KEY ix_alert_input_source(source_type,created_at,id),
 CONSTRAINT fk_alert_input_vehicle FOREIGN KEY(vehicle_id) REFERENCES vehicles(id),
 CONSTRAINT fk_alert_input_device FOREIGN KEY(device_id) REFERENCES gps_devices(id),
 CONSTRAINT fk_alert_input_sensor FOREIGN KEY(sensor_id) REFERENCES ble_sensors(id),
 CONSTRAINT fk_alert_input_position FOREIGN KEY(gps_position_id) REFERENCES gps_positions(id),
 CONSTRAINT fk_alert_input_fuel FOREIGN KEY(fuel_reading_id) REFERENCES fuel_readings(id),
 CONSTRAINT fk_alert_input_device_event FOREIGN KEY(device_event_id) REFERENCES device_events(id),
 CONSTRAINT fk_alert_input_telemetry FOREIGN KEY(sensor_telemetry_id) REFERENCES sensor_telemetry(id),
 CONSTRAINT fk_alert_input_raw FOREIGN KEY(raw_message_id) REFERENCES device_raw_messages(id),
 CONSTRAINT ck_alert_input_resolution CHECK(resolution_status='RESOLVED'),
 CONSTRAINT ck_alert_input_time CHECK((timestamp_quality IN ('ORIGINAL_DEVICE_TIME','CORRELATED_TIME') AND event_time IS NOT NULL) OR (timestamp_quality IN ('RECEIVED_TIME_ONLY','UNKNOWN') AND event_time IS NULL))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS alert_intake_rejections (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 event_key BINARY(32) NULL,
 event_type VARCHAR(64) NULL,
 vehicle_id BIGINT UNSIGNED NULL,
 source_type VARCHAR(32) NULL,
 source_id BIGINT UNSIGNED NULL,
 reason_code VARCHAR(64) NOT NULL,
 timestamp_quality VARCHAR(32) NULL,
 resolution_status VARCHAR(24) NULL,
 detail_redacted VARCHAR(500) NOT NULL,
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 KEY ix_alert_rejections_reason(reason_code,created_at,id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS alert_evaluation_jobs (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 event_id BIGINT UNSIGNED NOT NULL,
 vehicle_id BIGINT UNSIGNED NOT NULL,
 job_type VARCHAR(32) NOT NULL,
 state VARCHAR(24) NOT NULL,
 attempts INT NOT NULL DEFAULT 0,
 max_attempts INT NOT NULL DEFAULT 5,
 available_at DATETIME(3) NOT NULL,
 locked_at DATETIME(3) NULL,
 locked_by VARCHAR(80) NULL,
 processed_at DATETIME(3) NULL,
 last_error_code VARCHAR(64) NULL,
 last_error_redacted VARCHAR(500) NULL,
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
 UNIQUE KEY uq_alert_job_event_type(event_id,job_type),
 KEY ix_alert_job_claim(state,available_at,locked_at,id),
 CONSTRAINT fk_alert_job_event FOREIGN KEY(event_id) REFERENCES alert_input_events(id),
 CONSTRAINT fk_alert_job_vehicle FOREIGN KEY(vehicle_id) REFERENCES vehicles(id),
 CONSTRAINT ck_alert_job_state CHECK(state IN ('PENDING','PROCESSING','SUCCESS','RETRY','FAILED')),
 CONSTRAINT ck_alert_job_attempts CHECK(attempts>=0 AND max_attempts>0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS alert_job_history (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 job_id BIGINT UNSIGNED NOT NULL,
 previous_state VARCHAR(24) NULL,
 new_state VARCHAR(24) NOT NULL,
 worker_id VARCHAR(80) NULL,
 reason_code VARCHAR(64) NULL,
 detail_redacted VARCHAR(500) NULL,
 occurred_at DATETIME(3) NOT NULL,
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 KEY ix_alert_job_history(job_id,occurred_at,id),
 CONSTRAINT fk_alert_job_history_job FOREIGN KEY(job_id) REFERENCES alert_evaluation_jobs(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS alert_rule_states (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 rule_id BIGINT UNSIGNED NOT NULL,
 vehicle_id BIGINT UNSIGNED NOT NULL,
 scope_key VARCHAR(120) NOT NULL,
 current_alert_id BIGINT UNSIGNED NULL,
 evaluator_state JSON NULL,
 event_watermark DATETIME(3) NULL,
 last_event_id BIGINT UNSIGNED NULL,
 row_version BIGINT NOT NULL DEFAULT 1,
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
 UNIQUE KEY uq_alert_rule_state(rule_id,vehicle_id,scope_key),
 KEY fk_alert_rule_state_current(current_alert_id),
 CONSTRAINT fk_alert_rule_state_rule FOREIGN KEY(rule_id) REFERENCES alert_rules(id),
 CONSTRAINT fk_alert_rule_state_vehicle FOREIGN KEY(vehicle_id) REFERENCES vehicles(id),
 CONSTRAINT fk_alert_rule_state_current FOREIGN KEY(current_alert_id) REFERENCES alert_events(id),
 CONSTRAINT fk_alert_rule_state_last_event FOREIGN KEY(last_event_id) REFERENCES alert_input_events(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS alert_event_sources (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 alert_id BIGINT UNSIGNED NOT NULL,
 input_event_id BIGINT UNSIGNED NOT NULL,
 source_role VARCHAR(24) NOT NULL,
 source_type VARCHAR(32) NOT NULL,
 device_id BIGINT UNSIGNED NULL,
 sensor_id BIGINT UNSIGNED NULL,
 event_time DATETIME(3) NULL,
 received_at DATETIME(3) NOT NULL,
 timestamp_quality VARCHAR(32) NOT NULL,
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 UNIQUE KEY uq_alert_source_event(alert_id,input_event_id),
 KEY ix_alert_source_input(input_event_id,alert_id),
 CONSTRAINT fk_alert_source_alert FOREIGN KEY(alert_id) REFERENCES alert_events(id),
 CONSTRAINT fk_alert_source_input FOREIGN KEY(input_event_id) REFERENCES alert_input_events(id),
 CONSTRAINT fk_alert_source_device FOREIGN KEY(device_id) REFERENCES gps_devices(id),
 CONSTRAINT fk_alert_source_sensor FOREIGN KEY(sensor_id) REFERENCES ble_sensors(id),
 CONSTRAINT ck_alert_source_role CHECK(source_role IN ('TRIGGER','OBSERVATION','RECOVERY'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS alert_audit_trail (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 alert_id BIGINT UNSIGNED NOT NULL,
 action VARCHAR(40) NOT NULL,
 previous_state VARCHAR(24) NULL,
 new_state VARCHAR(24) NULL,
 actor_id BIGINT UNSIGNED NULL,
 actor_type VARCHAR(16) NOT NULL,
 reason_code VARCHAR(64) NULL,
 reason_text VARCHAR(500) NULL,
 input_event_id BIGINT UNSIGNED NULL,
 request_id VARCHAR(64) NOT NULL,
 occurred_at DATETIME(3) NOT NULL,
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 KEY ix_alert_audit_history(alert_id,occurred_at,id),
 CONSTRAINT fk_alert_audit_alert FOREIGN KEY(alert_id) REFERENCES alert_events(id),
 CONSTRAINT fk_alert_audit_actor FOREIGN KEY(actor_id) REFERENCES users(id),
 CONSTRAINT fk_alert_audit_input FOREIGN KEY(input_event_id) REFERENCES alert_input_events(id),
 CONSTRAINT ck_alert_audit_actor CHECK((actor_type='SYSTEM' AND actor_id IS NULL) OR (actor_type='USER' AND actor_id IS NOT NULL))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
