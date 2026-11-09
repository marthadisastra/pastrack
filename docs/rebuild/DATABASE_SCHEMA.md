# PasTrack — rancangan database untuk review

Status: PROPOSED; belum diterapkan. Sumber: kode lokal, migrasi lokal 001–008, migrasi sensor 20261102, audit terdahulu 2 November 2026. Tidak ada koneksi atau pemeriksaan ulang database produksi pada pekerjaan ini. DDL inti lama tidak ditemukan; tipe/indeks aktualnya memerlukan SHOW CREATE TABLE dari snapshot lokal. ERD lengkap: [ERD.mmd](ERD.mmd).

## Kamus data (65 tabel)

Semua tabel memakai id BIGINT UNSIGNED PK; tabel master dan konfigurasi memiliki created_at/updated_at UTC DATETIME(3). Tabel fakta append-only memiliki received_at atau occurred_at; tambahkan created_at bila belum tercantum. Notasi ? = NULL diperbolehkan; nama tabel pada tipe = FK BIGINT UNSIGNED; UQ = unik. DECIMAL(9_6) berarti DECIMAL(9,6). Kolom keamanan tidak dikirim oleh API. Semua FK default RESTRICT, tanpa cascade penghapusan histori.

Organisasi/multitenancy tidak diaktifkan: kebutuhan aktual belum membuktikannya. sensor_telemetry adalah tabel kanonis untuk seluruh observasi BLE; tidak membuat tabel ble_telemetry duplikat. gps_device_status adalah status saat ini, device_status_logs adalah histori.

### roles

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| code | VARCHAR(32) | UQ |
| name | VARCHAR(100) | — |

### permissions

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| code | VARCHAR(80) | UQ |
| description | VARCHAR(255) | — |

### users

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| username | VARCHAR(64) | UQ |
| full_name | VARCHAR(100) | Profil |
| email | VARCHAR(190)? | UQ |
| phone | VARCHAR(32)? | Profil |
| job_title | VARCHAR(100)? | Profil |
| department | VARCHAR(100)? | Profil |
| row_version | BIGINT UNSIGNED | Optimistic locking |
| password_changed_at | DATETIME(3)? | UTC |
| last_login_at | DATETIME(3)? | UTC |
| password_hash | VARCHAR(255) | — |
| password_algo | VARCHAR(30) | — |
| auth_version | BIGINT | — |
| is_active | BOOLEAN | — |
| deleted_at | DATETIME(3)? | — |

### user_roles

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| user_id | users | — |
| role_id | roles | — |

### role_permissions

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| role_id | roles | — |
| permission_id | permissions | — |

### sessions

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| user_id | users | — |
| token_hash | CHAR(64) | UQ |
| auth_version | BIGINT | — |
| expires_at | DATETIME(3) | — |

### vehicle_types

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| code | VARCHAR(32) | UQ |
| name | VARCHAR(100) | — |

### vehicle_groups

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| name | VARCHAR(100) | UQ |

### vehicles

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| plate_number | VARCHAR(32) | UQ |
| vehicle_name | VARCHAR(120)? | — |
| fleet_code | VARCHAR(64)? | — |
| vehicle_type_id | vehicle_types? | — |
| physical_ignition_state | VARCHAR(24)=NOT_INSTALLED | — |
| deleted_at | DATETIME(3)? | — |
| row_version | BIGINT | — |

### vehicle_group_members

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| vehicle_id | vehicles | — |
| group_id | vehicle_groups | — |

### vehicle_assets

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| vehicle_id | vehicles | UQ |
| make_model | VARCHAR(120)? | — |
| production_year | SMALLINT? | — |
| vin | VARCHAR(60)? | — |
| engine_number | VARCHAR(60)? | — |
| fuel_type | VARCHAR(40)? | — |
| ownership | VARCHAR(80)? | — |
| owner_name | VARCHAR(120)? | — |
| color | VARCHAR(60)? | — |
| transmission | VARCHAR(60)? | — |
| seats | SMALLINT? | — |
| insurance_provider | VARCHAR(120)? | — |
| insurance_policy | VARCHAR(100)? | — |
| insurance_expiry | DATE? | — |
| tax_expiry | DATE? | — |
| registration_expiry | DATE? | — |
| notes | TEXT? | — |
| photo_mime | VARCHAR(20)? | — |
| photo_updated_at | DATETIME(3)? | — |

### gps_device_models

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| manufacturer | VARCHAR(80) | — |
| model | VARCHAR(80) | — |
| protocol_family | VARCHAR(64) | — |

### gps_devices

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| imei | VARCHAR(20) | UQ |
| model_id | gps_device_models | — |
| firmware_version | VARCHAR(64)? | — |
| protocol_version | VARCHAR(32)? | — |
| is_active | BOOLEAN | — |
| deleted_at | DATETIME(3)? | — |

### vehicle_assignments

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| vehicle_id | vehicles | — |
| device_id | gps_devices | — |
| starts_at | DATETIME(3) | — |
| ends_at | DATETIME(3)? | — |
| provenance | VARCHAR(32) | — |

### gps_device_status

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| device_id | gps_devices | UQ |
| last_received_at | DATETIME(3)? | — |
| connectivity | VARCHAR(24) | — |
| last_raw_message_id | device_raw_messages? | — |

### gps_device_configurations

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| device_id | gps_devices | — |
| version | BIGINT | — |
| configuration | JSON | — |
| applied_at | DATETIME(3)? | — |
| confirmed_at | DATETIME(3)? | — |

### gps_device_command_logs

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| device_id | gps_devices | — |
| requested_by | users? | — |
| command_code | VARCHAR(32) | — |
| payload_redacted | TEXT | — |
| state | VARCHAR(24) | — |
| sent_at | DATETIME(3)? | — |
| acknowledged_raw_id | device_raw_messages? | — |

### protocol_message_types

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| family | VARCHAR(64) | — |
| code | VARCHAR(32) | — |
| supported | BOOLEAN | — |

### device_raw_messages

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| device_id | gps_devices? | — |
| legacy_id | BIGINT? | UQ |
| identifier_encrypted | VARBINARY(255)? | — |
| message_type_id | protocol_message_types? | — |
| protocol_version | VARCHAR(32)? | — |
| payload | LONGBLOB | — |
| payload_sha256 | BINARY(32) | — |
| received_at | DATETIME(3) | — |
| device_time | DATETIME(3)? | — |
| spool_receipt_id | VARCHAR(64)? | UQ |

### protocol_parse_logs

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| raw_message_id | device_raw_messages | — |
| parser_version | VARCHAR(64) | — |
| parse_status | VARCHAR(24) | — |
| processing_status | VARCHAR(24) | — |
| fields | JSON? | — |
| processed_at | DATETIME(3)? | — |

### protocol_parser_errors

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| parse_log_id | protocol_parse_logs | — |
| code | VARCHAR(64) | — |
| field_path | VARCHAR(120)? | — |
| detail_redacted | TEXT | — |

### message_processing_keys

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| device_id | gps_devices? | — |
| message_key | BINARY(32) | UQ |
| canonical_raw_id | device_raw_messages | — |
| parser_version | VARCHAR(64) | — |

### gps_positions

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| device_id | gps_devices | — |
| vehicle_id | vehicles? | — |
| assignment_id | vehicle_assignments? | — |
| raw_message_id | device_raw_messages? | — |
| sample_index | SMALLINT | — |
| gps_time | DATETIME(3)? | — |
| received_at | DATETIME(3) | — |
| device_send_time | DATETIME(3)? | — |
| latitude | DECIMAL(9_6) | — |
| longitude | DECIMAL(10_6) | — |
| speed_kmh | DECIMAL(6_2)? | — |
| heading_deg | DECIMAL(5_2)? | — |
| altitude_m | DECIMAL(10_2)? | — |
| satellite_count | SMALLINT? | — |
| hdop | DECIMAL(6_2)? | — |
| gnss_accuracy_raw | VARCHAR(16)? | — |
| fix_status | VARCHAR(24) | — |
| validation_state | VARCHAR(24) | — |

### latest_positions

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| vehicle_id | vehicles | UQ |
| position_id | gps_positions | UQ |
| gps_time | DATETIME(3) | — |
| received_at | DATETIME(3) | — |

### gps_quality_logs

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| device_id | gps_devices | — |
| position_id | gps_positions? | — |
| raw_message_id | device_raw_messages | — |
| satellite_count | SMALLINT? | — |
| hdop | DECIMAL(6_2)? | — |
| fix_status | VARCHAR(24) | — |
| event_at | DATETIME(3)? | — |

### device_status_logs

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| device_id | gps_devices | — |
| raw_message_id | device_raw_messages | — |
| event_at | DATETIME(3)? | — |
| status_code_raw | VARCHAR(32)? | — |
| decoded_status | JSON? | — |

### network_status_logs

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| device_id | gps_devices | — |
| raw_message_id | device_raw_messages | — |
| event_at | DATETIME(3)? | — |
| mcc | VARCHAR(4)? | — |
| mnc | VARCHAR(4)? | — |
| lac | VARCHAR(8)? | — |
| cell_id | VARCHAR(16)? | — |
| rssi | SMALLINT? | — |

### trips

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| vehicle_id | vehicles | — |
| start_position_id | gps_positions | — |
| end_position_id | gps_positions? | — |
| started_at | DATETIME(3) | — |
| ended_at | DATETIME(3)? | — |
| distance_m | DECIMAL(14_3) | — |
| driving_seconds | INT | — |
| stopped_seconds | INT | — |
| max_speed_kmh | DECIMAL(6_2)? | — |
| average_speed_kmh | DECIMAL(6_2)? | — |
| calculation_version | VARCHAR(32) | — |
| confidence | VARCHAR(24) | — |

### trip_segments

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| trip_id | trips | — |
| start_position_id | gps_positions | — |
| end_position_id | gps_positions | — |
| segment_kind | VARCHAR(24) | — |
| distance_m | DECIMAL(14_3) | — |
| duration_seconds | INT | — |

### trip_events

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| trip_id | trips | — |
| position_id | gps_positions? | — |
| event_kind | VARCHAR(32) | — |
| event_at | DATETIME(3) | — |

### ble_sensors

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| sensor_identifier | VARCHAR(64) | UQ |
| mac | CHAR(12)? | UQ |
| model | VARCHAR(64) | — |
| is_active | BOOLEAN | — |
| deleted_at | DATETIME(3)? | — |

### ble_sensor_bindings

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| sensor_id | ble_sensors | — |
| vehicle_id | vehicles? | — |
| device_id | gps_devices? | — |
| starts_at | DATETIME(3) | — |
| ends_at | DATETIME(3)? | — |
| binding_source | VARCHAR(24) | — |

### sensor_telemetry

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| sensor_id | ble_sensors? | — |
| device_id | gps_devices | — |
| binding_id | ble_sensor_bindings? | — |
| raw_message_id | device_raw_messages | — |
| accessory_index | SMALLINT | — |
| sensor_time | DATETIME(3)? | — |
| received_at | DATETIME(3) | — |
| accessory_type | SMALLINT? | — |
| accessory_model | SMALLINT? | — |
| append_mask | VARCHAR(8)? | — |
| payload_hex | VARCHAR(256)? | — |
| tilt_raw | SMALLINT? | — |
| fuel_height_raw | INT? | — |
| volume_raw | INT? | — |
| battery_mv | INT? | — |
| battery_percent | DECIMAL(5_2)? | — |
| temperature_c | DECIMAL(6_2)? | — |
| rssi | SMALLINT? | — |
| validation_state | VARCHAR(24) | — |
| decoder_version | VARCHAR(64) | — |

### ble_connection_logs

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| sensor_id | ble_sensors | — |
| device_id | gps_devices | — |
| raw_message_id | device_raw_messages? | — |
| event_at | DATETIME(3)? | — |
| received_at | DATETIME(3) | — |
| status | VARCHAR(24) | — |
| source | VARCHAR(24) | — |

### ble_sensor_status

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| sensor_id | ble_sensors | UQ |
| telemetry_id | sensor_telemetry? | — |
| last_received_at | DATETIME(3)? | — |
| status | VARCHAR(24) | — |
| stale_after_seconds | INT | — |

### fuel_tank_profiles

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| vehicle_id | vehicles | — |
| name | VARCHAR(100) | — |
| capacity_l | DECIMAL(12_3)? | — |
| revision | BIGINT | — |
| effective_from | DATETIME(3) | — |
| effective_to | DATETIME(3)? | — |

### fuel_sensors

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| sensor_id | ble_sensors | UQ |
| tank_profile_id | fuel_tank_profiles? | — |
| measurement_unit | VARCHAR(24)? | — |
| decoding_state | VARCHAR(24) | — |

### fuel_calibrations

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| fuel_sensor_id | fuel_sensors | — |
| tank_profile_id | fuel_tank_profiles? | — |
| revision | BIGINT | — |
| media | VARCHAR(24) | — |
| input_quantity | VARCHAR(32) | — |
| input_unit | VARCHAR(24)? | — |
| output_unit | VARCHAR(24) | — |
| approval_state | VARCHAR(24) | — |
| approved_by | users? | — |
| effective_from | DATETIME(3)? | — |
| effective_to | DATETIME(3)? | — |

### fuel_calibration_points

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| calibration_id | fuel_calibrations | — |
| input_value | DECIMAL(16_6) | — |
| output_value | DECIMAL(12_3) | — |
| legacy_percent | DECIMAL(5_2)? | — |
| evidence | TEXT? | — |

### fuel_readings

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| fuel_sensor_id | fuel_sensors | — |
| telemetry_id | sensor_telemetry | UQ |
| vehicle_id | vehicles? | — |
| calibration_id | fuel_calibrations? | — |
| reading_at | DATETIME(3)? | — |
| received_at | DATETIME(3) | — |
| raw_measurement | DECIMAL(16_6)? | — |
| raw_unit | VARCHAR(24)? | — |
| fuel_liters | DECIMAL(12_3)? | — |
| fuel_percent | DECIMAL(5_2)? | — |
| validity | VARCHAR(32) | — |
| invalid_reason | VARCHAR(120)? | — |

### latest_fuel_readings

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| fuel_sensor_id | fuel_sensors | UQ |
| reading_id | fuel_readings | — |
| last_valid_reading_id | fuel_readings? | — |
| updated_at | DATETIME(3) | — |

### fuel_events

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| vehicle_id | vehicles | — |
| fuel_sensor_id | fuel_sensors | — |
| start_reading_id | fuel_readings | — |
| end_reading_id | fuel_readings | — |
| event_type | VARCHAR(24) | — |
| delta_l | DECIMAL(12_3) | — |
| event_at | DATETIME(3) | — |
| rule_version | VARCHAR(32) | — |
| confidence | VARCHAR(24) | — |

### fuel_consumption_summaries

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| vehicle_id | vehicles | — |
| period_start | DATETIME(3) | — |
| period_end | DATETIME(3) | — |
| consumed_l | DECIMAL(14_3)? | — |
| distance_m | DECIMAL(14_3)? | — |
| coverage_percent | DECIMAL(5_2) | — |
| algorithm_version | VARCHAR(32) | — |

### event_types

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| code | VARCHAR(64) | UQ |
| description | VARCHAR(255) | — |

### event_severities

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| code | VARCHAR(24) | UQ |
| rank | SMALLINT | — |

### device_events

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| device_id | gps_devices? | — |
| vehicle_id | vehicles? | — |
| raw_message_id | device_raw_messages? | — |
| event_type_id | event_types? | — |
| severity_id | event_severities? | — |
| event_at | DATETIME(3)? | — |
| received_at | DATETIME(3) | — |
| raw_code | VARCHAR(64)? | — |
| raw_value | TEXT? | — |
| source | VARCHAR(32) | — |
| verification_state | VARCHAR(24) | — |

### event_acknowledgements

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| event_id | device_events | — |
| user_id | users | — |
| acknowledged_at | DATETIME(3) | — |
| note | TEXT? | — |

### geofences

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| name | VARCHAR(100) | — |
| shape | VARCHAR(16) | — |
| center_lat | DECIMAL(9_6)? | — |
| center_lon | DECIMAL(10_6)? | — |
| radius_m | DECIMAL(12_3)? | — |
| polygon_geojson | JSON? | — |
| is_active | BOOLEAN | — |
| row_version | BIGINT | — |

### geofence_vehicle_assignments

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| geofence_id | geofences | — |
| vehicle_id | vehicles | — |
| starts_at | DATETIME(3) | — |
| ends_at | DATETIME(3)? | — |

### geofence_events

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| geofence_id | geofences | — |
| vehicle_id | vehicles | — |
| position_id | gps_positions | — |
| transition | VARCHAR(16) | — |
| event_at | DATETIME(3) | — |

### alert_rules

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| name | VARCHAR(120) | — |
| event_type_id | event_types? | — |
| severity_id | event_severities | — |
| configuration | JSON | — |
| version | BIGINT | — |
| enabled | BOOLEAN | — |

### alert_events

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| rule_id | alert_rules | — |
| device_event_id | device_events? | — |
| fuel_event_id | fuel_events? | — |
| geofence_event_id | geofence_events? | — |
| vehicle_id | vehicles? | — |
| state | VARCHAR(24) | — |
| raised_at | DATETIME(3) | — |
| acknowledged_by | users? | — |
| acknowledged_at | DATETIME(3)? | — |
| resolved_at | DATETIME(3)? | — |

### notification_channels

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| user_id | users | — |
| channel | VARCHAR(24) | — |
| configuration_encrypted | BLOB? | — |
| enabled | BOOLEAN | — |

### alert_notifications

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| alert_id | alert_events | — |
| channel_id | notification_channels | — |
| notification_id | notifications? | — |
| state | VARCHAR(24) | — |
| attempts | INT | — |
| sent_at | DATETIME(3)? | — |

### notifications

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| user_id | users | — |
| title | VARCHAR(120) | — |
| body | VARCHAR(1000) | — |
| target | VARCHAR(200) | — |
| read_at | DATETIME(3)? | — |

### push_tokens

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| user_id | users | — |
| session_id | sessions | — |
| token_hash | CHAR(64) | UQ |
| token_encrypted | BLOB | — |
| client_id | VARCHAR(64) | — |

### push_jobs

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| notification_id | notifications | — |
| push_token_id | push_tokens | — |
| state | VARCHAR(24) | — |
| attempts | INT | — |
| next_attempt | DATETIME(3) | — |

### audit_log

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| actor_id | users? | — |
| actor_legacy | VARCHAR(100)? | — |
| action | VARCHAR(30) | — |
| entity | VARCHAR(64) | — |
| entity_id | BIGINT? | — |
| before_data | JSON? | — |
| after_data | JSON? | — |
| request_id | VARCHAR(64) | — |
| occurred_at | DATETIME(3) | — |

### tracking_places

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| name | VARCHAR(100) | — |
| latitude | DECIMAL(9_6) | — |
| longitude | DECIMAL(10_6) | — |
| radius_m | INT | — |
| row_version | BIGINT | — |

### tracking_geocode_cache

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| cell_key | VARCHAR(64) | UQ |
| name | VARCHAR(200) | — |
| address | VARCHAR(1000) | — |
| result_type | VARCHAR(40) | — |
| distance_m | DOUBLE? | — |
| expires_at | DATETIME(3) | — |

### tracking_geocode_usage

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| usage_day | DATE | UQ |
| requests | INT | — |

### report_jobs

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| requested_by | users | — |
| report_type | VARCHAR(64) | — |
| filters | JSON | — |
| state | VARCHAR(24) | — |
| artifact_path | VARCHAR(255)? | — |
| expires_at | DATETIME(3)? | — |

### report_schedules

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| user_id | users | — |
| report_type | VARCHAR(64) | — |
| filters | JSON | — |
| timezone | VARCHAR(64) | — |
| schedule | VARCHAR(120) | — |
| enabled | BOOLEAN | — |

### migration_batches

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| source_schema | VARCHAR(64) | — |
| snapshot_reference | VARCHAR(255) | — |
| state | VARCHAR(24) | — |
| source_watermarks | JSON | — |
| validation_report | JSON? | — |

### migration_record_map

| Kolom | Tipe / hubungan | Aturan |
|---|---|---|
| id | BIGINT UNSIGNED | PK |
| batch_id | migration_batches | — |
| source_table | VARCHAR(64) | — |
| source_pk | VARCHAR(128) | — |
| target_table | VARCHAR(64) | — |
| target_pk | BIGINT? | — |
| source_checksum | BINARY(32) | — |
| state | VARCHAR(24) | — |
| issue | TEXT? | — |

## Constraint dan indeks wajib

- FK seluruh hubungan ERD; seluruh FK berindeks. FK nullable tidak diisi 0. Data yatim dipertahankan di sumber dan ditandai migration_record_map, bukan dihapus.
- Composite UNIQUE: user_roles(user_id,role_id), role_permissions(role_id,permission_id), vehicle_group_members(vehicle_id,group_id), gps_device_models(manufacturer,model), protocol_message_types(family,code), gps_device_configurations(device_id,version), protocol_parse_logs(raw_message_id,parser_version), gps_positions(raw_message_id,sample_index), sensor_telemetry(raw_message_id,accessory_index), fuel_calibrations(fuel_sensor_id,revision), fuel_calibration_points(calibration_id,input_value), fuel_consumption_summaries(vehicle_id,period_start,period_end,algorithm_version), migration_record_map(batch_id,source_table,source_pk,target_table), push_jobs(notification_id,push_token_id). NULL raw_message_id pada histori legacy diperbolehkan; tidak deduplikasi data tanpa bukti.
- Indeks histori: gps_positions(vehicle_id,gps_time,id), gps_positions(device_id,gps_time,id), device_raw_messages(device_id,received_at,id), device_raw_messages(message_type_id,received_at,id), protocol_parse_logs(parse_status,processed_at,id), sensor_telemetry(sensor_id,received_at,id), fuel_readings(vehicle_id,reading_at,id), device_events(vehicle_id,event_at,id), alert_events(state,raised_at,id), push_jobs(state,next_attempt,id). Gunakan cursor keyset, bukan scan semua histori.
- CHECK latitude -90..90, longitude -180..180, speed >=0, heading 0..<360, percent 0..100, capacity >0 jika tersedia, readings >=0 jika terverifikasi. Koordinat 0,0 tidak otomatis invalid secara global; gunakan GNSS validity/context.
- latest_positions diperbarui hanya oleh posisi tervalidasi dengan gps_time; urutkan (gps_time,received_at,id). Telemetri terlambat tidak menimpa state lebih baru. Latest fuel menyimpan referensi latest observed dan last valid terpisah.
- Assignment bersifat interval [starts_at,ends_at); beberapa GPS per kendaraan diperbolehkan, satu GPS tidak boleh aktif pada dua kendaraan bersamaan. Validasi overlap dalam transaksi dengan lock baris device. Sensor binding eksplisit terpisah dari observed association; binding tidak ditebak dari nama/MAC yang berdekatan.
- Physical ignition untuk kendaraan saat ini: NOT_INSTALLED. Movement dan status raw tetap independen. Tidak membentuk engine-on, idle berbasis mesin, ataupun durasi mesin dari motion code.
- Raw payload tidak diubah; parse state disimpan di tabel terpisah. Akun runtime hanya INSERT/SELECT raw. Percobaan spool berulang memakai spool_receipt_id stabil; pesan jaringan identik tetap disimpan sebagai receipt terpisah, hanya derived processing yang idempoten. Jika identitas pesan tidak dapat dibuktikan, jangan gabungkan hanya berdasarkan timestamp/IMEI.
- Tidak memakai partisi sampai volume/EXPLAIN membuktikan kebutuhan. Tidak menambahkan Redis/Kafka. Semua datetime UTC, tampilan Asia/Jakarta.
