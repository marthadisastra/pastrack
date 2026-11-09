# Lampiran skema live — audit Phase 1

Tanggal: 5 November 2026 (WIB). Metadata dibaca melalui transaksi `READ ONLY`; tidak ada migration atau perubahan database. Database: `pastrack_v2_20261103_r2`. Total 71 tabel, 656 kolom. Ini inventaris kolom, bukan dump data. Foreign key dan indeks rujuk migration; kesetaraan seluruh constraint live belum diverifikasi.

## alert_events

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| rule_id | bigint(20) unsigned | NO |
| device_event_id | bigint(20) unsigned | YES |
| fuel_event_id | bigint(20) unsigned | YES |
| geofence_event_id | bigint(20) unsigned | YES |
| vehicle_id | bigint(20) unsigned | YES |
| state | varchar(24) | NO |
| raised_at | datetime(3) | NO |
| acknowledged_by | bigint(20) unsigned | YES |
| acknowledged_at | datetime(3) | YES |
| resolved_at | datetime(3) | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## alert_notifications

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| alert_id | bigint(20) unsigned | NO |
| channel_id | bigint(20) unsigned | NO |
| notification_id | bigint(20) unsigned | YES |
| state | varchar(24) | NO |
| attempts | int(11) | NO |
| sent_at | datetime(3) | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## alert_rules

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| name | varchar(120) | NO |
| event_type_id | bigint(20) unsigned | YES |
| severity_id | bigint(20) unsigned | NO |
| configuration | longtext | NO |
| version | bigint(20) | NO |
| enabled | tinyint(1) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## audit_log

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| actor_id | bigint(20) unsigned | YES |
| actor_legacy | varchar(100) | YES |
| action | varchar(30) | NO |
| entity | varchar(64) | NO |
| entity_id | bigint(20) | YES |
| before_data | longtext | YES |
| after_data | longtext | YES |
| request_id | varchar(64) | NO |
| occurred_at | datetime(3) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## ble_connection_logs

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| sensor_id | bigint(20) unsigned | NO |
| device_id | bigint(20) unsigned | NO |
| raw_message_id | bigint(20) unsigned | YES |
| event_at | datetime(3) | YES |
| received_at | datetime(3) | NO |
| status | varchar(24) | NO |
| source | varchar(24) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## ble_sensors

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| sensor_identifier | varchar(64) | NO |
| mac | char(12) | YES |
| model | varchar(64) | NO |
| is_active | tinyint(1) | NO |
| deleted_at | datetime(3) | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## ble_sensor_bindings

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| sensor_id | bigint(20) unsigned | NO |
| vehicle_id | bigint(20) unsigned | YES |
| device_id | bigint(20) unsigned | YES |
| starts_at | datetime(3) | NO |
| ends_at | datetime(3) | YES |
| binding_source | varchar(24) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## ble_sensor_status

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| sensor_id | bigint(20) unsigned | NO |
| telemetry_id | bigint(20) unsigned | YES |
| last_received_at | datetime(3) | YES |
| status | varchar(24) | NO |
| stale_after_seconds | int(11) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## device_events

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| device_id | bigint(20) unsigned | YES |
| vehicle_id | bigint(20) unsigned | YES |
| raw_message_id | bigint(20) unsigned | YES |
| event_type_id | bigint(20) unsigned | YES |
| severity_id | bigint(20) unsigned | YES |
| event_at | datetime(3) | YES |
| received_at | datetime(3) | NO |
| raw_code | varchar(64) | YES |
| raw_value | text | YES |
| source | varchar(32) | NO |
| verification_state | varchar(24) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## device_raw_messages

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| device_id | bigint(20) unsigned | YES |
| legacy_id | bigint(20) | YES |
| identifier_encrypted | varbinary(255) | YES |
| message_type_id | bigint(20) unsigned | YES |
| protocol_version | varchar(32) | YES |
| payload | longblob | NO |
| payload_sha256 | binary(32) | NO |
| received_at | datetime(3) | NO |
| device_time | datetime(3) | YES |
| spool_receipt_id | varchar(64) | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## device_status_logs

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| device_id | bigint(20) unsigned | NO |
| raw_message_id | bigint(20) unsigned | NO |
| event_at | datetime(3) | YES |
| status_code_raw | varchar(32) | YES |
| decoded_status | longtext | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## event_acknowledgements

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| event_id | bigint(20) unsigned | NO |
| user_id | bigint(20) unsigned | NO |
| acknowledged_at | datetime(3) | NO |
| note | text | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## event_severities

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| code | varchar(24) | NO |
| rank | smallint(6) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## event_types

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| code | varchar(64) | NO |
| description | varchar(255) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## fuel_calibrations

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| fuel_sensor_id | bigint(20) unsigned | NO |
| tank_profile_id | bigint(20) unsigned | YES |
| revision | bigint(20) | NO |
| media | varchar(24) | NO |
| input_quantity | varchar(32) | NO |
| input_unit | varchar(24) | YES |
| output_unit | varchar(24) | NO |
| approval_state | varchar(24) | NO |
| approved_by | bigint(20) unsigned | YES |
| effective_from | datetime(3) | YES |
| effective_to | datetime(3) | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## fuel_calibration_points

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| calibration_id | bigint(20) unsigned | NO |
| input_value | decimal(16,6) | NO |
| output_value | decimal(12,3) | NO |
| legacy_percent | decimal(5,2) | YES |
| evidence | text | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## fuel_consumption_summaries

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| vehicle_id | bigint(20) unsigned | NO |
| period_start | datetime(3) | NO |
| period_end | datetime(3) | NO |
| consumed_l | decimal(14,3) | YES |
| distance_m | decimal(14,3) | YES |
| coverage_percent | decimal(5,2) | NO |
| algorithm_version | varchar(32) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## fuel_events

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| vehicle_id | bigint(20) unsigned | NO |
| fuel_sensor_id | bigint(20) unsigned | NO |
| start_reading_id | bigint(20) unsigned | NO |
| end_reading_id | bigint(20) unsigned | NO |
| event_type | varchar(24) | NO |
| delta_l | decimal(12,3) | NO |
| event_at | datetime(3) | NO |
| rule_version | varchar(32) | NO |
| confidence | varchar(24) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## fuel_event_investigations

| Kolom | Tipe live | Nullable |
|---|---|---|
| event_id | bigint(20) unsigned | NO |
| state | varchar(24) | NO |
| note | text | NO |
| receipt_number | varchar(100) | NO |
| station_name | varchar(200) | NO |
| receipt_liters | decimal(12,3) | YES |
| receipt_amount | decimal(16,2) | YES |
| row_version | bigint(20) | NO |
| updated_by | bigint(20) unsigned | NO |
| updated_at | datetime(3) | NO |

## fuel_readings

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| fuel_sensor_id | bigint(20) unsigned | NO |
| telemetry_id | bigint(20) unsigned | NO |
| vehicle_id | bigint(20) unsigned | YES |
| calibration_id | bigint(20) unsigned | YES |
| reading_at | datetime(3) | YES |
| received_at | datetime(3) | NO |
| raw_measurement | decimal(16,6) | YES |
| raw_unit | varchar(24) | YES |
| fuel_liters | decimal(12,3) | YES |
| fuel_percent | decimal(5,2) | YES |
| validity | varchar(32) | NO |
| invalid_reason | varchar(120) | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## fuel_sensors

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| sensor_id | bigint(20) unsigned | NO |
| tank_profile_id | bigint(20) unsigned | YES |
| measurement_unit | varchar(24) | YES |
| decoding_state | varchar(24) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## fuel_tank_profiles

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| vehicle_id | bigint(20) unsigned | YES |
| name | varchar(100) | NO |
| capacity_l | decimal(12,3) | YES |
| revision | bigint(20) | NO |
| effective_from | datetime(3) | NO |
| effective_to | datetime(3) | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## geofences

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| name | varchar(100) | NO |
| shape | varchar(16) | NO |
| center_lat | decimal(9,6) | YES |
| center_lon | decimal(10,6) | YES |
| radius_m | decimal(12,3) | YES |
| polygon_geojson | longtext | YES |
| is_active | tinyint(1) | NO |
| row_version | bigint(20) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## geofence_events

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| geofence_id | bigint(20) unsigned | NO |
| vehicle_id | bigint(20) unsigned | NO |
| position_id | bigint(20) unsigned | NO |
| transition | varchar(16) | NO |
| event_at | datetime(3) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## geofence_vehicle_assignments

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| geofence_id | bigint(20) unsigned | NO |
| vehicle_id | bigint(20) unsigned | NO |
| starts_at | datetime(3) | NO |
| ends_at | datetime(3) | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## gps_devices

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| imei | varchar(20) | NO |
| model_id | bigint(20) unsigned | NO |
| firmware_version | varchar(64) | YES |
| protocol_version | varchar(32) | YES |
| is_active | tinyint(1) | NO |
| deleted_at | datetime(3) | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## gps_device_command_logs

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| device_id | bigint(20) unsigned | NO |
| requested_by | bigint(20) unsigned | YES |
| command_code | varchar(32) | NO |
| payload_redacted | text | NO |
| state | varchar(24) | NO |
| sent_at | datetime(3) | YES |
| acknowledged_raw_id | bigint(20) unsigned | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## gps_device_configurations

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| device_id | bigint(20) unsigned | NO |
| version | bigint(20) | NO |
| configuration | longtext | NO |
| applied_at | datetime(3) | YES |
| confirmed_at | datetime(3) | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## gps_device_models

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| manufacturer | varchar(80) | NO |
| model | varchar(80) | NO |
| protocol_family | varchar(64) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## gps_device_status

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| device_id | bigint(20) unsigned | NO |
| last_received_at | datetime(3) | YES |
| connectivity | varchar(24) | NO |
| last_raw_message_id | bigint(20) unsigned | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## gps_positions

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| device_id | bigint(20) unsigned | NO |
| vehicle_id | bigint(20) unsigned | YES |
| assignment_id | bigint(20) unsigned | YES |
| raw_message_id | bigint(20) unsigned | YES |
| sample_index | smallint(6) | NO |
| gps_time | datetime(3) | YES |
| received_at | datetime(3) | NO |
| device_send_time | datetime(3) | YES |
| latitude | decimal(10,7) | NO |
| longitude | decimal(11,7) | NO |
| speed_kmh | decimal(6,2) | YES |
| heading_deg | decimal(5,2) | YES |
| altitude_m | decimal(10,2) | YES |
| satellite_count | smallint(6) | YES |
| hdop | decimal(6,2) | YES |
| gnss_accuracy_raw | varchar(16) | YES |
| fix_status | varchar(24) | NO |
| validation_state | varchar(24) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## gps_quality_logs

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| device_id | bigint(20) unsigned | NO |
| position_id | bigint(20) unsigned | YES |
| raw_message_id | bigint(20) unsigned | NO |
| satellite_count | smallint(6) | YES |
| hdop | decimal(6,2) | YES |
| fix_status | varchar(24) | NO |
| event_at | datetime(3) | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## latest_fuel_readings

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| fuel_sensor_id | bigint(20) unsigned | NO |
| reading_id | bigint(20) unsigned | NO |
| last_valid_reading_id | bigint(20) unsigned | YES |
| updated_at | datetime(3) | NO |
| created_at | datetime(3) | NO |

## latest_positions

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| vehicle_id | bigint(20) unsigned | NO |
| position_id | bigint(20) unsigned | NO |
| gps_time | datetime(3) | NO |
| received_at | datetime(3) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## message_processing_keys

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| device_id | bigint(20) unsigned | YES |
| message_key | binary(32) | NO |
| canonical_raw_id | bigint(20) unsigned | NO |
| parser_version | varchar(64) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## migration_batches

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| source_schema | varchar(64) | NO |
| snapshot_reference | varchar(255) | NO |
| state | varchar(24) | NO |
| source_watermarks | longtext | NO |
| validation_report | longtext | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## migration_record_map

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| batch_id | bigint(20) unsigned | NO |
| source_table | varchar(64) | NO |
| source_pk | varchar(128) | NO |
| target_table | varchar(64) | NO |
| target_pk | bigint(20) | YES |
| source_checksum | binary(32) | NO |
| state | varchar(24) | NO |
| issue | text | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## network_status_logs

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| device_id | bigint(20) unsigned | NO |
| raw_message_id | bigint(20) unsigned | NO |
| event_at | datetime(3) | YES |
| mcc | varchar(4) | YES |
| mnc | varchar(4) | YES |
| lac | varchar(8) | YES |
| cell_id | varchar(16) | YES |
| rssi | smallint(6) | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## notifications

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| user_id | bigint(20) unsigned | NO |
| title | varchar(120) | NO |
| body | varchar(1000) | NO |
| target | varchar(200) | NO |
| read_at | datetime(3) | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## notification_channels

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| user_id | bigint(20) unsigned | NO |
| channel | varchar(24) | NO |
| configuration_encrypted | blob | YES |
| enabled | tinyint(1) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## permissions

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| code | varchar(80) | NO |
| description | varchar(255) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## protocol_message_types

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| family | varchar(64) | NO |
| code | varchar(32) | NO |
| supported | tinyint(1) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## protocol_parser_errors

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| parse_log_id | bigint(20) unsigned | NO |
| code | varchar(64) | NO |
| field_path | varchar(120) | YES |
| detail_redacted | text | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## protocol_parse_logs

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| raw_message_id | bigint(20) unsigned | NO |
| parser_version | varchar(64) | NO |
| parse_status | varchar(24) | NO |
| processing_status | varchar(24) | NO |
| fields | longtext | YES |
| processed_at | datetime(3) | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## push_jobs

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| notification_id | bigint(20) unsigned | NO |
| push_token_id | bigint(20) unsigned | NO |
| state | varchar(24) | NO |
| attempts | int(11) | NO |
| next_attempt | datetime(3) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## push_tokens

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| user_id | bigint(20) unsigned | NO |
| session_id | bigint(20) unsigned | NO |
| token_hash | char(64) | NO |
| token_encrypted | blob | NO |
| client_id | varchar(64) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## remote_output_commands

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| device_id | bigint(20) unsigned | NO |
| vehicle_id | bigint(20) unsigned | NO |
| requested_by | bigint(20) unsigned | NO |
| serial | char(4) | NO |
| request_key | char(64) | NO |
| desired_active | tinyint(1) | NO |
| state | varchar(24) | NO |
| sent_at | datetime(3) | YES |
| acknowledged_at | datetime(3) | YES |
| acknowledged_raw_id | bigint(20) unsigned | YES |
| expires_at | datetime(3) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## report_jobs

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| requested_by | bigint(20) unsigned | NO |
| report_type | varchar(64) | NO |
| filters | longtext | NO |
| state | varchar(24) | NO |
| artifact_path | varchar(255) | YES |
| expires_at | datetime(3) | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## report_schedules

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| user_id | bigint(20) unsigned | NO |
| report_type | varchar(64) | NO |
| filters | longtext | NO |
| timezone | varchar(64) | NO |
| schedule | varchar(120) | NO |
| enabled | tinyint(1) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## roles

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| code | varchar(32) | NO |
| name | varchar(100) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## role_permissions

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| role_id | bigint(20) unsigned | NO |
| permission_id | bigint(20) unsigned | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## sensor_calibration_sessions

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| sensor_id | bigint(20) unsigned | NO |
| fuel_sensor_id | bigint(20) unsigned | NO |
| device_id | bigint(20) unsigned | NO |
| vehicle_id | bigint(20) unsigned | NO |
| quantity | varchar(32) | NO |
| capacity_l | decimal(12,3) | NO |
| state | varchar(24) | NO |
| started_by | bigint(20) unsigned | NO |
| started_at | datetime(3) | NO |
| phase_started_at | datetime(3) | NO |
| phase_telemetry_id | bigint(20) unsigned | NO |
| empty_raw | decimal(14,3) | YES |
| empty_telemetry_id | bigint(20) unsigned | YES |
| empty_evidence | longtext | YES |
| full_raw | decimal(14,3) | YES |
| full_telemetry_id | bigint(20) unsigned | YES |
| calibration_id | bigint(20) unsigned | YES |
| error_message | varchar(255) | YES |
| updated_at | datetime(3) | NO |

## sensor_reported_names

| Kolom | Tipe live | Nullable |
|---|---|---|
| sensor_id | bigint(20) unsigned | NO |
| reported_name | varchar(80) | NO |
| reported_at | datetime(3) | NO |
| received_at | datetime(3) | NO |
| raw_message_id | bigint(20) unsigned | NO |

## sensor_telemetry

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| sensor_id | bigint(20) unsigned | YES |
| device_id | bigint(20) unsigned | NO |
| binding_id | bigint(20) unsigned | YES |
| raw_message_id | bigint(20) unsigned | NO |
| accessory_index | smallint(6) | NO |
| sensor_time | datetime(3) | YES |
| received_at | datetime(3) | NO |
| accessory_type | smallint(6) | YES |
| accessory_model | smallint(6) | YES |
| append_mask | varchar(8) | YES |
| payload_hex | varchar(256) | YES |
| tilt_raw | smallint(6) | YES |
| fuel_height_raw | int(11) | YES |
| volume_raw | int(11) | YES |
| battery_mv | int(11) | YES |
| battery_percent | decimal(5,2) | YES |
| temperature_c | decimal(6,2) | YES |
| rssi | smallint(6) | YES |
| validation_state | varchar(24) | NO |
| decoder_version | varchar(64) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## sessions

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| user_id | bigint(20) unsigned | NO |
| token_hash | char(64) | NO |
| auth_version | bigint(20) | NO |
| expires_at | datetime(3) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## stage_migration_steps

| Kolom | Tipe live | Nullable |
|---|---|---|
| version | varchar(128) | NO |
| step | int(11) | NO |
| checksum | char(64) | YES |

## tracking_geocode_cache

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| cell_key | varchar(64) | NO |
| name | varchar(200) | NO |
| address | varchar(1000) | NO |
| result_type | varchar(40) | NO |
| distance_m | double | YES |
| expires_at | datetime(3) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## tracking_geocode_usage

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| usage_day | date | NO |
| requests | int(11) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## tracking_places

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| name | varchar(100) | NO |
| latitude | decimal(9,6) | NO |
| longitude | decimal(10,6) | NO |
| radius_m | int(11) | NO |
| row_version | bigint(20) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## trips

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| vehicle_id | bigint(20) unsigned | NO |
| start_position_id | bigint(20) unsigned | NO |
| end_position_id | bigint(20) unsigned | YES |
| started_at | datetime(3) | NO |
| ended_at | datetime(3) | YES |
| distance_m | decimal(14,3) | NO |
| driving_seconds | int(11) | NO |
| stopped_seconds | int(11) | NO |
| max_speed_kmh | decimal(6,2) | YES |
| average_speed_kmh | decimal(6,2) | YES |
| calculation_version | varchar(32) | NO |
| confidence | varchar(24) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## trip_events

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| trip_id | bigint(20) unsigned | NO |
| position_id | bigint(20) unsigned | YES |
| event_kind | varchar(32) | NO |
| event_at | datetime(3) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## trip_segments

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| trip_id | bigint(20) unsigned | NO |
| start_position_id | bigint(20) unsigned | NO |
| end_position_id | bigint(20) unsigned | NO |
| segment_kind | varchar(24) | NO |
| distance_m | decimal(14,3) | NO |
| duration_seconds | int(11) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## users

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| username | varchar(64) | NO |
| password_hash | varchar(255) | NO |
| password_algo | varchar(30) | NO |
| auth_version | bigint(20) | NO |
| is_active | tinyint(1) | NO |
| deleted_at | datetime(3) | YES |
| full_name | varchar(100) | NO |
| email | varchar(190) | YES |
| phone | varchar(32) | YES |
| job_title | varchar(100) | YES |
| department | varchar(100) | YES |
| row_version | bigint(20) | NO |
| password_changed_at | datetime(3) | YES |
| last_login_at | datetime(3) | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## user_roles

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| user_id | bigint(20) unsigned | NO |
| role_id | bigint(20) unsigned | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## vehicles

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| plate_number | varchar(32) | NO |
| vehicle_name | varchar(120) | YES |
| fleet_code | varchar(64) | YES |
| vehicle_type_id | bigint(20) unsigned | YES |
| physical_ignition_state | varchar(24) | NO |
| deleted_at | datetime(3) | YES |
| row_version | bigint(20) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## vehicle_assets

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| vehicle_id | bigint(20) unsigned | NO |
| make_model | varchar(120) | YES |
| production_year | smallint(6) | YES |
| vin | varchar(60) | YES |
| engine_number | varchar(60) | YES |
| fuel_type | varchar(40) | YES |
| ownership | varchar(80) | YES |
| owner_name | varchar(120) | YES |
| color | varchar(60) | YES |
| transmission | varchar(60) | YES |
| seats | smallint(6) | YES |
| insurance_provider | varchar(120) | YES |
| insurance_policy | varchar(100) | YES |
| insurance_expiry | date | YES |
| tax_expiry | date | YES |
| registration_expiry | date | YES |
| notes | text | YES |
| photo_mime | varchar(20) | YES |
| photo_updated_at | datetime(3) | YES |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## vehicle_assignments

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| vehicle_id | bigint(20) unsigned | NO |
| device_id | bigint(20) unsigned | NO |
| starts_at | datetime(3) | NO |
| ends_at | datetime(3) | YES |
| provenance | varchar(32) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## vehicle_groups

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| name | varchar(100) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## vehicle_group_members

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| vehicle_id | bigint(20) unsigned | NO |
| group_id | bigint(20) unsigned | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

## vehicle_inventory

| Kolom | Tipe live | Nullable |
|---|---|---|
| vehicle_id | bigint(20) unsigned | NO |
| vehicle_code | varchar(64) | YES |
| tank_capacity_l | decimal(12,3) | YES |
| photo | mediumblob | YES |
| photo_mime | varchar(32) | YES |
| photo_updated_at | datetime(3) | YES |
| updated_at | datetime(3) | NO |

## vehicle_types

| Kolom | Tipe live | Nullable |
|---|---|---|
| id | bigint(20) unsigned | NO |
| code | varchar(32) | NO |
| name | varchar(100) | NO |
| created_at | datetime(3) | NO |
| updated_at | datetime(3) | NO |

