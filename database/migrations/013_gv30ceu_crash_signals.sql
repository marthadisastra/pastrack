CREATE TABLE queclink_signal_catalog (
 message_type VARCHAR(16) NOT NULL PRIMARY KEY,
 display_name VARCHAR(100) NOT NULL, category VARCHAR(40) NOT NULL,
 description VARCHAR(500) NOT NULL, event_class VARCHAR(40) NOT NULL,
 supported_device VARCHAR(40) NOT NULL, firmware_requirement VARCHAR(120) NULL,
 parser_status VARCHAR(40) NOT NULL, evaluator_status VARCHAR(40) NOT NULL,
 alert_eligibility VARCHAR(40) NOT NULL, documentation_reference VARCHAR(255) NOT NULL,
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE crash_evidence_sets (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 sequence_key CHAR(64) NOT NULL,
 device_id BIGINT UNSIGNED NOT NULL, vehicle_id BIGINT UNSIGNED NOT NULL,
 assignment_id BIGINT UNSIGNED NOT NULL, crash_counter TINYINT UNSIGNED NOT NULL,
 anchor_at DATETIME(3) NOT NULL, primary_message_type VARCHAR(16) NULL,
 evidence_status VARCHAR(24) NOT NULL DEFAULT 'PENDING', expected_frames INT NULL,
 received_frames INT NOT NULL DEFAULT 0, row_version BIGINT NOT NULL DEFAULT 1,
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
 UNIQUE KEY uq_crash_sequence_key(sequence_key),
 KEY ix_crash_correlate(device_id,crash_counter,anchor_at),
 CONSTRAINT fk_crash_set_device FOREIGN KEY(device_id) REFERENCES gps_devices(id),
 CONSTRAINT fk_crash_set_vehicle FOREIGN KEY(vehicle_id) REFERENCES vehicles(id),
 CONSTRAINT fk_crash_set_assignment FOREIGN KEY(assignment_id) REFERENCES vehicle_assignments(id),
 CONSTRAINT ck_crash_evidence_status CHECK(evidence_status IN ('PENDING','PARTIAL','COMPLETE','UNAVAILABLE'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE crash_signal_events (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 evidence_set_id BIGINT UNSIGNED NULL, raw_message_id BIGINT UNSIGNED NOT NULL,
 device_id BIGINT UNSIGNED NOT NULL, vehicle_id BIGINT UNSIGNED NOT NULL,
 assignment_id BIGINT UNSIGNED NOT NULL, message_type VARCHAR(16) NOT NULL,
 crash_counter TINYINT UNSIGNED NULL, data_type TINYINT UNSIGNED NULL,
 total_frames INT NULL, frame_number INT NULL, acceleration_hex MEDIUMTEXT NULL,
 trigger_degrees INT NULL, engine_status BOOLEAN NULL,
 event_time DATETIME(3) NOT NULL, received_at DATETIME(3) NOT NULL,
 timestamp_quality VARCHAR(32) NOT NULL, latitude DECIMAL(9,6) NULL,
 longitude DECIMAL(10,6) NULL, speed_kmh DECIMAL(6,2) NULL,
 firmware_version VARCHAR(64) NULL, protocol_version VARCHAR(32) NOT NULL,
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 UNIQUE KEY uq_crash_signal_raw(raw_message_id),
 UNIQUE KEY uq_crash_frame(evidence_set_id,message_type,data_type,frame_number),
 KEY ix_crash_signal_vehicle(vehicle_id,event_time,id),
 CONSTRAINT fk_crash_signal_set FOREIGN KEY(evidence_set_id) REFERENCES crash_evidence_sets(id),
 CONSTRAINT fk_crash_signal_raw FOREIGN KEY(raw_message_id) REFERENCES device_raw_messages(id),
 CONSTRAINT fk_crash_signal_device FOREIGN KEY(device_id) REFERENCES gps_devices(id),
 CONSTRAINT fk_crash_signal_vehicle FOREIGN KEY(vehicle_id) REFERENCES vehicles(id),
 CONSTRAINT fk_crash_signal_assignment FOREIGN KEY(assignment_id) REFERENCES vehicle_assignments(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO queclink_signal_catalog(message_type,display_name,category,description,event_class,supported_device,firmware_requirement,parser_status,evaluator_status,alert_eligibility,documentation_reference) VALUES
('GTCRA','Crash Alarm','CRASH','Native crash alarm','CRITICAL','GV30CEU','Protocol v0306, live 80201E0306 verified','VERIFIED_LIVE','TRIGGER','CRASH_PRIMARY','GV30CEU v0306 sec. 4.5.24'),
('GTECR','Extended Crash Alarm','CRASH','Extended native crash alarm','CRITICAL','GV30CEU','Protocol v0306, no live sample','VERIFIED_DOCUMENT_ONLY','CONDITIONAL_TRIGGER','CRASH_PRIMARY_CONDITIONAL','GV30CEU v0306 sec. 4.5.33'),
('GTCRD','Crash Data','CRASH','Framed tri-axial crash reconstruction data','SUPPORTING_EVIDENCE','GV30CEU','Protocol v0306, live verified','VERIFIED_LIVE','SUPPORTING','NO_NEW_ALERT','GV30CEU v0306 sec. 4.6.1'),
('GTACC','Acceleration Data','TELEMETRY','Tri-axial acceleration packet','TELEMETRY','GV30CEU','Protocol v0306, live verified','VERIFIED_LIVE','INFORMATIONAL','NO_ALERT','GV30CEU v0306 sec. 4.6.2'),
('GTHBM','Harsh Behavior','DRIVING_BEHAVIOR','Harsh driving report','WARNING','GV30CEU','Protocol v0306','CATALOG_ONLY','NOT_IMPLEMENTED','FUTURE_RULE','GV30CEU v0306 sec. 4.2.1'),
('GTSPD','Speed Alarm','DRIVING_BEHAVIOR','Device-native speed alarm','WARNING','GV30CEU','Protocol v0306','CATALOG_ONLY','NOT_IMPLEMENTED','FUTURE_RULE','GV30CEU v0306 sec. 4.2.1'),
('GTTOW','Tow Alarm','VEHICLE_SECURITY','Motion-sensor tow alarm','WARNING','GV30CEU','Protocol v0306','CATALOG_ONLY','NOT_IMPLEMENTED','FUTURE_RULE','GV30CEU v0306 sec. 4.2.1'),
('GTSOS','SOS Alarm','VEHICLE_SECURITY','Digital input SOS report','CRITICAL','GV30CEU','Protocol v0306, input config required','CATALOG_ONLY','NOT_IMPLEMENTED','FUTURE_RULE','GV30CEU v0306 sec. 4.2.1'),
('GTDIS','Digital Input','DEVICE_INPUT','Digital input state report','INFORMATIONAL','GV30CEU','Protocol v0306, wiring/config required','CATALOG_ONLY','NOT_IMPLEMENTED','NO_ALERT','GV30CEU v0306 sec. 4.2.1'),
('GTVGL','Virtual Ignition','DEVICE_INPUT','Virtual ignition location report','INFORMATIONAL','GV30CEU','Protocol v0306','VERIFIED_PARSER','NOT_ELIGIBLE','NO_IGNITION_ALERT','GV30CEU v0306 sec. 4.2.1'),
('GTIGL','Ignition Location','DEVICE_INPUT','Physical ignition location report','INFORMATIONAL','GV30CEU','Protocol v0306, wiring required','VERIFIED_LIVE','NOT_ELIGIBLE','NO_IGNITION_ALERT','GV30CEU v0306 sec. 4.2.1');

INSERT INTO alert_rules(name,code,severity_id,configuration,version,enabled)
SELECT 'Indikasi Kecelakaan Kendaraan','CRASH',id,
 JSON_OBJECT('correlation_window_seconds',600,'evidence_wait_seconds',30,'readiness','READY_NATIVE_GTCRA','gtecr_enabled',false),1,1
FROM event_severities WHERE code='CRITICAL'
ON DUPLICATE KEY UPDATE name=VALUES(name),severity_id=VALUES(severity_id),configuration=VALUES(configuration),version=VALUES(version),enabled=VALUES(enabled);
