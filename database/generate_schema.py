"""Generate reviewable MySQL DDL from the versioned schema dictionary. No DB access."""
import json
from pathlib import Path
base=Path(__file__).parent
schema=json.loads((base/'schema.json').read_text())
lines=['-- PasTrack v2: isolated development schema only. No source tables modified.']
for table, columns in schema.items():
 definitions=['id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY']
 for col in columns:
  name, raw, *flags=col; nullable='?' in raw
  typ=raw.replace('?','').split('=')[0]
  typ='BIGINT UNSIGNED' if typ in schema else typ.replace('_',',')
  default=" DEFAULT 'NOT_INSTALLED'" if '=NOT_INSTALLED' in raw else ''
  definitions.append(f'`{name}` {typ} '+('NULL' if nullable else 'NOT NULL')+default+(' UNIQUE' if 'UQ' in flags else ''))
 
 existing={col[0] for col in columns}
 if 'created_at' not in existing: definitions.append('created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)')
 if 'updated_at' not in existing: definitions.append('updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)')
 lines.append(f'CREATE TABLE IF NOT EXISTS `{table}` (\n  '+',\n  '.join(definitions)+'\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;')
# FKs emitted after all CREATEs; permits cyclic optional references.
for table,columns in schema.items():
 for col in columns:
  target=col[1].replace('?','').split('=')[0]
  if target in schema:
   lines.append(f'ALTER TABLE `{table}` ADD CONSTRAINT `fk_{table}_{col[0]}` FOREIGN KEY (`{col[0]}`) REFERENCES `{target}`(id);')
indexes={
'user_roles':[('user_id','role_id')], 'role_permissions':[('role_id','permission_id')],
'vehicle_group_members':[('vehicle_id','group_id')], 'gps_device_models':[('manufacturer','model')],
'protocol_message_types':[('family','code')], 'gps_device_configurations':[('device_id','version')],
'protocol_parse_logs':[('raw_message_id','parser_version')], 'gps_positions':[('raw_message_id','sample_index')],
'sensor_telemetry':[('raw_message_id','accessory_index')], 'fuel_calibrations':[('fuel_sensor_id','revision')],
'fuel_calibration_points':[('calibration_id','input_value')], 'push_jobs':[('notification_id','push_token_id')],
'fuel_consumption_summaries':[('vehicle_id','period_start','period_end','algorithm_version')],
'migration_record_map':[('batch_id','source_table','source_pk','target_table')]}
for table,groups in indexes.items():
 for n,cols in enumerate(groups): lines.append(f'CREATE UNIQUE INDEX uq_{table}_{n} ON {table} ({",".join(cols)});')
for table,cols in {
'gps_positions':'vehicle_id,gps_time,id', 'device_raw_messages':'device_id,received_at,id',
'protocol_parse_logs':'parse_status,processed_at,id', 'sensor_telemetry':'sensor_id,received_at,id',
'fuel_readings':'vehicle_id,reading_at,id', 'device_events':'vehicle_id,event_at,id',
'alert_events':'state,raised_at,id', 'push_jobs':'state,next_attempt,id',
'vehicle_assignments':'device_id,starts_at,ends_at', 'ble_sensor_bindings':'sensor_id,starts_at,ends_at'}.items():
 lines.append(f'CREATE INDEX ix_{table}_history ON {table} ({cols});')
lines+=['ALTER TABLE gps_positions ADD CONSTRAINT ck_position_bounds CHECK (latitude BETWEEN -90 AND 90 AND longitude BETWEEN -180 AND 180 AND (speed_kmh IS NULL OR speed_kmh>=0) AND (heading_deg IS NULL OR (heading_deg>=0 AND heading_deg<360)));',
'ALTER TABLE fuel_readings ADD CONSTRAINT ck_fuel_range CHECK ((fuel_liters IS NULL OR fuel_liters>=0) AND (fuel_percent IS NULL OR fuel_percent BETWEEN 0 AND 100));',
"ALTER TABLE vehicles ADD CONSTRAINT ck_physical_ignition CHECK (physical_ignition_state IN ('NOT_INSTALLED','NOT_AVAILABLE','INSTALLED'));",
'ALTER TABLE vehicle_assignments ADD CONSTRAINT ck_assignment_interval CHECK (ends_at IS NULL OR ends_at>starts_at);']
(base/'migrations'/'001_platform.sql').write_text('\n\n'.join(lines)+'\n')
