"""Rehearse data transformation from an isolated restored snapshot to v2 staging.
Existing live schema and services are never modified. Identity values stay server-side.
"""
import sys,os,json,hashlib,datetime,decimal,gzip,subprocess
from pathlib import Path
sys.path[:0]=['/www/server/panel','/www/server/panel/class','/www/server/panel/class_v2'];os.chdir('/www/server/panel')
import public,pymysql
password=public.M('config').where('id=?',(1,)).getField('mysql_root')
root=Path(sys.argv[1]);source=os.environ.get('PASTRACK_SOURCE','pastrack_legacy_restore_20261103');target=os.environ.get('PASTRACK_TARGET','pastrack_v2_stage_20261103')
assert source=='pastrack' or source.startswith('pastrack_legacy_restore_')
assert target.startswith('pastrack_v2_') and all(ch.isalnum() or ch=='_' for ch in target)
c=pymysql.connect(host='localhost',user='root',password=password,charset='utf8mb4',autocommit=True,cursorclass=pymysql.cursors.DictCursor);q=c.cursor();q.execute("SET time_zone='+00:00'");q.execute("SET SESSION sql_mode='TRADITIONAL'")
if source!='pastrack':q.execute('CREATE DATABASE IF NOT EXISTS '+source+' CHARACTER SET utf8mb4')
q.execute('SHOW TABLES FROM '+source)
if not q.fetchall():
 env=os.environ.copy();env['MYSQL_PWD']=password
 p=subprocess.Popen(['mysql','-u','root',source],stdin=subprocess.PIPE,stderr=subprocess.PIPE,env=env)
 with gzip.open(root/'backup/pastrack-source.sql.gz','rb') as f:
  while chunk:=f.read(1024*1024):p.stdin.write(chunk)
 p.stdin.close();error=p.stderr.read();assert p.wait()==0,'restore failed'
counts=json.loads((root/'backup/source-counts.json').read_text())
for t,n in counts.items():
 q.execute('SELECT COUNT(*) n FROM '+source+'.`'+t+'`');assert q.fetchone()['n']==n,'restore count mismatch: '+t
q.execute('USE '+target)
def rows(t):q.execute('SELECT * FROM '+source+'.`'+t+'` ORDER BY '+('id' if t not in ['vehicle_assets','web_sessions','web_push_tokens','tracking_geocode_cache','tracking_geocode_usage'] else {'vehicle_assets':'vehicle_id','web_sessions':'token_hash','web_push_tokens':'token_hash','tracking_geocode_cache':'cell_key','tracking_geocode_usage':'usage_day'}[t]));return q.fetchall()
def serial(v):
 if isinstance(v,(datetime.datetime,datetime.date,decimal.Decimal)):return str(v)
 if isinstance(v,bytes):return v.hex()
 raise TypeError(type(v).__name__)
def insert(table,values):
 keys=list(values)
 if 'id' not in values:
  q.execute('SELECT id FROM `'+table+'` WHERE '+' AND '.join('`'+k+'` <=> %s' for k in keys)+' LIMIT 1',list(values.values()));existing=q.fetchone()
  if existing:return existing['id']
 q.execute('INSERT INTO `'+table+'` (`'+'`,`'.join(keys)+'`) VALUES('+','.join(['%s']*len(keys))+') ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id)',list(values.values()));return q.lastrowid or values.get('id')
def copy(t,columns,rename=None):
 for r in rows(t):
  data={k:r[k] for k in columns};insert(rename or t,data);mapped(t,r['id'] if 'id' in r else next(iter(r.values())),rename or t,data.get('id'),r)
now=datetime.datetime.now(datetime.timezone.utc).replace(tzinfo=None)
batch=insert('migration_batches',{'source_schema':source,'snapshot_reference':str(root/'backup/pastrack-source.sql.gz'),'state':'backfilling','source_watermarks':json.dumps(counts)})
def mapped(table,pk,dest,target_pk,row,state='migrated',issue=None):
 insert('migration_record_map',{'batch_id':batch,'source_table':table,'source_pk':str(pk),'target_table':dest,'target_pk':target_pk,'source_checksum':hashlib.sha256(json.dumps(row,default=serial,sort_keys=True).encode()).digest(),'state':state,'issue':issue})
try:
 for code in ['admin','operator','viewer']:insert('roles',{'code':code,'name':code})
 q.execute('SELECT id,code FROM roles');roles={r['code']:r['id'] for r in q.fetchall()}
 for r in rows('web_users'):
  d={k:r[k] for k in ['id','username','full_name','email','phone','job_title','department','password_hash','password_algo','auth_version','is_active','row_version','password_changed_at','last_login_at','deleted_at','created_at','updated_at']};insert('users',d);insert('user_roles',{'user_id':r['id'],'role_id':roles[r['role']]});mapped('web_users',r['id'],'users',r['id'],r)
 for r in rows('vehicles'):
  type_id=insert('vehicle_types',{'code':r['vehicle_type'],'name':r['vehicle_type']})
  insert('vehicles',{k:r[k] for k in ['id','plate_number','vehicle_name','fleet_code','deleted_at','row_version','created_at','updated_at']}|{'vehicle_type_id':type_id,'physical_ignition_state':'NOT_INSTALLED'});mapped('vehicles',r['id'],'vehicles',r['id'],r)
 for r in rows('vehicle_assets'):
  d={k:v for k,v in r.items() if k!='tank_capacity_l'};d['id']=r['vehicle_id'];insert('vehicle_assets',d)
  if r['tank_capacity_l'] is not None:insert('fuel_tank_profiles',{'id':r['vehicle_id'],'vehicle_id':r['vehicle_id'],'name':'Legacy tank capacity (unverified calibration)','capacity_l':r['tank_capacity_l'],'revision':1,'effective_from':now})
  mapped('vehicle_assets',r['vehicle_id'],'vehicle_assets',r['vehicle_id'],r)
 for r in rows('gps_devices'):
  model=insert('gps_device_models',{'manufacturer':'Queclink' if r['model']=='GV30CEU' else 'unverified','model':r['model'],'protocol_family':r['protocol_name']})
  insert('gps_devices',{k:r[k] for k in ['id','imei','firmware_version','protocol_version','is_active','deleted_at','created_at','updated_at']}|{'model_id':model});mapped('gps_devices',r['id'],'gps_devices',r['id'],r)
  if r['vehicle_id']:
   # Known installed_at is admissible evidence; otherwise current-only snapshot interval.
   insert('vehicle_assignments',{'vehicle_id':r['vehicle_id'],'device_id':r['id'],'starts_at':r['installed_at'] or now,'provenance':'legacy_installed_at' if r['installed_at'] else 'snapshot_current_only'})
  insert('gps_device_status',{'device_id':r['id'],'last_received_at':r['last_seen_at'],'connectivity':'unknown'})
 raw_cache={}
 for r in rows('device_raw_messages'):
  kind=None
  if r['message_type']:kind=insert('protocol_message_types',{'family':'queclink','code':r['message_type'],'supported':False})
  payload=r['raw_payload'].encode('utf-8');insert('device_raw_messages',{'id':r['id'],'legacy_id':r['id'],'device_id':r['device_id'],'message_type_id':kind,'protocol_version':r['protocol_version'],'payload':payload,'payload_sha256':hashlib.sha256(payload).digest(),'received_at':r['received_at'],'spool_receipt_id':'legacy:'+str(r['id'])})
  log=insert('protocol_parse_logs',{'raw_message_id':r['id'],'parser_version':'legacy-import-v1','parse_status':r['parse_status'],'processing_status':'legacy_imported','fields':r['parsed_payload'],'processed_at':now})
  if r['parse_error']:insert('protocol_parser_errors',{'parse_log_id':log,'code':'legacy_parse_note','detail_redacted':'Legacy parser note retained in private source snapshot; review payload using authorized inspector.'})
  if r['device_id']:
   normalized=r['raw_payload'].strip()
   if normalized.startswith('+BUFF:'):normalized='+RESP:'+normalized[len('+BUFF:'):]
   insert('message_processing_keys',{'device_id':r['device_id'],'message_key':hashlib.sha256(normalized.encode()).digest(),'canonical_raw_id':r['id'],'parser_version':'legacy-import-v1'})
  raw_cache[r['id']]=r;mapped('device_raw_messages',r['id'],'device_raw_messages',r['id'],r)
 legacy_sample_counts={}
 for r in rows('gps_positions'):
  sample_index=legacy_sample_counts.get(r['raw_message_id'],0) if r['raw_message_id'] is not None else 0
  if r['raw_message_id'] is not None:legacy_sample_counts[r['raw_message_id']]=sample_index+1
  if r['latitude'] is None or r['longitude'] is None:
   mapped('gps_positions',r['id'],'gps_positions',None,r,'quarantined','Missing coordinates retained in source snapshot');continue
  raw=raw_cache.get(r['raw_message_id']);state='unknown';hdop=None
  if raw and raw['protocol_version']=='80201E0306' and raw['message_type']=='GTSTT':
   f=raw['raw_payload'].strip().rstrip('$').split(',')
   if len(f)>5 and f[5].isdigit():
    value=int(f[5]);state='no_fix' if value==0 else 'valid' if value<=50 else 'unknown';hdop=value if state=='valid' else None
  q.execute('SELECT id,vehicle_id FROM vehicle_assignments WHERE device_id=%s AND starts_at<=%s AND (ends_at IS NULL OR ends_at>%s)',(r['device_id'],r['gps_time'],r['gps_time']));assign=q.fetchall();a=assign[0] if len(assign)==1 else None
  insert('gps_positions',{'id':r['id'],'device_id':r['device_id'],'vehicle_id':a['vehicle_id'] if a else None,'assignment_id':a['id'] if a else None,'raw_message_id':r['raw_message_id'],'sample_index':sample_index,'gps_time':r['gps_time'],'received_at':raw['received_at'] if raw else r['created_at'],'device_send_time':r['device_send_time'],'latitude':r['latitude'],'longitude':r['longitude'],'speed_kmh':r['speed_kmh'],'heading_deg':r['heading_deg'],'altitude_m':r['altitude_m'],'satellite_count':r['satellites'],'hdop':hdop,'gnss_accuracy_raw':str(r['gps_accuracy']) if r['gps_accuracy'] is not None else None,'fix_status':state,'validation_state':'legacy_imported'})
  q.execute('UPDATE gps_positions SET latitude=%s,longitude=%s WHERE id=%s',(r['latitude'],r['longitude'],r['id']))
  mapped('gps_positions',r['id'],'gps_positions',r['id'],r)
 q.execute("SELECT vehicle_id,MAX(gps_time) at FROM gps_positions WHERE vehicle_id IS NOT NULL AND fix_status='valid' AND gps_time IS NOT NULL GROUP BY vehicle_id")
 for latest in q.fetchall():
  q.execute('SELECT id,gps_time,received_at FROM gps_positions WHERE vehicle_id=%s AND gps_time=%s AND fix_status=%s ORDER BY received_at DESC,id DESC LIMIT 1',(latest['vehicle_id'],latest['at'],'valid'));p=q.fetchone()
  insert('latest_positions',{'vehicle_id':latest['vehicle_id'],'position_id':p['id'],'gps_time':p['gps_time'],'received_at':p['received_at']})
 for r in rows('device_events'):
  raw=raw_cache.get(r['raw_message_id']);insert('device_events',{'id':r['id'],'device_id':r['device_id'],'raw_message_id':r['raw_message_id'],'event_at':r['event_time'],'received_at':raw['received_at'] if raw else r['created_at'],'raw_code':r['event_code'],'raw_value':r['event_value'],'source':'legacy','verification_state':'unverified'});mapped('device_events',r['id'],'device_events',r['id'],r)
 for r in rows('device_fuel_sensors'):
  insert('ble_sensors',{'id':r['id'],'sensor_identifier':r['sensor_id'],'model':r['model'],'is_active':r['is_active'],'deleted_at':r['deleted_at']})
  insert('fuel_sensors',{'id':r['id'],'sensor_id':r['id'],'decoding_state':'raw_units_unverified'})
  insert('ble_sensor_status',{'sensor_id':r['id'],'last_received_at':r['last_seen_at'],'status':'unknown','stale_after_seconds':300})
  if r['device_id']:
   q.execute('SELECT vehicle_id FROM '+source+'.gps_devices WHERE id=%s',(r['device_id'],));device_row=q.fetchone()
   insert('ble_sensor_bindings',{'sensor_id':r['id'],'device_id':r['device_id'],'vehicle_id':device_row['vehicle_id'] if device_row else None,'starts_at':now,'binding_source':'legacy_administrative'})
  mapped('device_fuel_sensors',r['id'],'ble_sensors',r['id'],r)
 for r in rows('fuel_calibration_points'):
  calibration=insert('fuel_calibrations',{'fuel_sensor_id':r['sensor_pk'],'revision':1,'media':'unverified','input_quantity':'legacy_observed_word','output_unit':'percent','approval_state':'unverified'})
  insert('fuel_calibration_points',{'id':r['id'],'calibration_id':calibration,'input_value':r['raw_value'],'output_value':r['level_percent'],'legacy_percent':r['level_percent'],'evidence':r['note']});mapped('fuel_calibration_points',r['id'],'fuel_calibration_points',r['id'],r)
 sessions={}
 for r in rows('web_sessions'):
  sid=insert('sessions',{k:r[k] for k in ['user_id','token_hash','auth_version','expires_at','created_at']});sessions[r['token_hash']]=sid;mapped('web_sessions',r['token_hash'],'sessions',sid,r)
 copy('web_notifications',['id','user_id','title','body','target','read_at','created_at'],'notifications')
 tokens={}
 for r in rows('web_push_tokens'):
  # Keep credentials in original private DB; do not mislabel plaintext as encrypted.
  mapped('web_push_tokens',r['token_hash'],'push_tokens',None,r,'quarantined','Credential encryption/worker port pending; source token preserved')
 for r in rows('web_push_jobs'):mapped('web_push_jobs',r['id'],'push_jobs',None,r,'quarantined','Notification worker compatibility pending; original delivery state preserved')
 for r in rows('pastrack_audit_log'):
  insert('audit_log',{'id':r['id'],'actor_legacy':r['actor'],'action':r['action'],'entity':r['entity'],'entity_id':r['entity_id'],'before_data':r['before_data'],'after_data':r['after_data'],'request_id':r['request_id'],'occurred_at':r['occurred_at']});mapped('pastrack_audit_log',r['id'],'audit_log',r['id'],r)
 copy('tracking_places',['id','name','latitude','longitude','radius_m','row_version','created_at','updated_at'])
 for t in ['tracking_geocode_cache','tracking_geocode_usage']:
  for r in rows(t):insert(t,r);mapped(t,next(iter(r.values())),t,None,r)
 q.execute('SELECT source_table,state,COUNT(*) n FROM migration_record_map WHERE batch_id=%s GROUP BY source_table,state',(batch,));report=q.fetchall()
 q.execute('SELECT COUNT(*) n FROM '+source+'.device_raw_messages s LEFT JOIN device_raw_messages t ON t.legacy_id=s.id WHERE t.id IS NULL OR t.payload<>CAST(s.raw_payload AS BINARY)');assert q.fetchone()['n']==0,'raw byte parity failed'
 q.execute('SELECT COUNT(*) n FROM '+source+'.gps_positions s LEFT JOIN gps_positions t ON t.id=s.id WHERE t.id IS NULL AND s.latitude IS NOT NULL AND s.longitude IS NOT NULL');assert q.fetchone()['n']==0,'legacy position identity/count mismatch'
 q.execute("UPDATE migration_batches SET state='rehearsed_not_cutover',validation_report=%s WHERE id=%s",(json.dumps(report),batch))
 (root/'backfill-report.json').write_text(json.dumps({'restore_counts_verified':True,'raw_byte_parity':True,'report':report,'live_services_changed':False},indent=2));print(json.dumps({'restore_verified':True,'raw_byte_parity':True,'mapping':report,'live_services_changed':False}))
finally:c.close()
