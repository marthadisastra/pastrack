"""Authenticated API + actual payload replay against private staging, not live TCP."""
import os,sys,json,secrets,hashlib,datetime,time,socket,urllib.request
from pathlib import Path
sys.path[:0]=['/www/server/panel','/www/server/panel/class','/www/server/panel/class_v2'];os.chdir('/www/server/panel')
import public,pymysql
root=Path('/root/pastrack-rebuild-20261103');target='pastrack_v2_stage_20261103'
c=pymysql.connect(host='localhost',user='root',password=public.M('config').where('id=?',(1,)).getField('mysql_root'),autocommit=True,cursorclass=pymysql.cursors.DictCursor);q=c.cursor();q.execute("SET time_zone='+00:00'")
q.execute('SELECT id,auth_version FROM '+target+'.users WHERE is_active=1 AND deleted_at IS NULL ORDER BY id LIMIT 1');user=q.fetchone();assert user,'no migrated login account'
token=secrets.token_hex(32);digest=hashlib.sha256(token.encode()).hexdigest();q.execute('INSERT INTO '+target+'.sessions(user_id,token_hash,auth_version,expires_at) VALUES(%s,%s,%s,%s)',(user['id'],digest,user['auth_version'],datetime.datetime.now(datetime.timezone.utc).replace(tzinfo=None)+datetime.timedelta(minutes=10)))
endpoints=json.loads((root/'api-paths.json').read_text());results=[]
try:
 for endpoint in endpoints+['fleet/overview','fleet/live','fleet/activity','trips/summary']:
  req=urllib.request.Request('http://127.0.0.1:18092/api/v1/'+endpoint,headers={'Cookie':'pastrack_v2_session='+token})
  try:
   with urllib.request.urlopen(req,timeout=10) as response:body=json.load(response);assert 'data' in body;results.append({'path':endpoint,'status':response.status})
  except Exception as e:results.append({'path':endpoint,'error_type':type(e).__name__})
 q.execute('SELECT raw_payload FROM pastrack.device_raw_messages WHERE message_type=%s ORDER BY id DESC LIMIT 1',('GTERI',));payload=q.fetchone()['raw_payload'].encode()
 q.execute('SELECT COUNT(*) n FROM '+target+'.sensor_telemetry');before=q.fetchone()['n']
 with socket.create_connection(('127.0.0.1',15002),timeout=5) as conn:
  conn.sendall(payload[:20]);conn.sendall(payload[20:]);conn.sendall(payload)
 count=before
 for _ in range(50):
  q.execute('SELECT COUNT(*) n FROM '+target+'.sensor_telemetry');count=q.fetchone()['n']
  if count>before:break
  time.sleep(.2)
 q.execute('SELECT COUNT(*) n FROM '+target+'.device_raw_messages WHERE spool_receipt_id NOT LIKE %s',('legacy:%',));raw=q.fetchone()['n']
 q.execute('SELECT validity,COUNT(*) n FROM '+target+'.fuel_readings GROUP BY validity');fuel=q.fetchall()
 result={'api':results,'replay_is_physical_device_test':False,'replayed_actual_payload':True,'raw_receipts_after_replay':raw,'new_sensor_telemetry':count-before,'fuel_states':fuel,'old_live_services_changed':False}
 (root/'verification-result.json').write_text(json.dumps(result,indent=2));print(json.dumps(result));assert all('error_type' not in x for x in results),'API acceptance failed';assert count>before,'BLE persistence failed'
finally:q.execute('DELETE FROM '+target+'.sessions WHERE token_hash=%s',(digest,));c.close()
