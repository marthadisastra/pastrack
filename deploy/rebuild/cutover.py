"""Authorized v2 production cutover. Preserves source DB/env/binaries and aaPanel supervision."""
import os,sys,time,json,shutil,subprocess,secrets,shlex,urllib.request,hashlib,datetime
from pathlib import Path
sys.path[:0]=['/www/server/panel','/www/server/panel/class','/www/server/panel/class_v2'];os.chdir('/www/server/panel')
import public,pymysql
from BTPanel import app
from projectModelV2.goModel import main
root=Path('/root/pastrack-live-cutover-20261103-r2');target='pastrack_v2_20261103_r2';runtime=Path('/opt/pastrack/v2-live');config=Path('/etc/pastrack/v2-live.env')
assert not (root/'cutover-success.json').exists(),'cutover already completed; inspect before changing runtime'
root.mkdir(mode=0o700,exist_ok=True);runtime.mkdir(exist_ok=True)
source_tools=Path('/root/pastrack-rebuild-20261103');(root/'migrations').mkdir(exist_ok=True)
for file in (source_tools/'migrations').glob('*.sql'):shutil.copy2(file,root/'migrations'/file.name)
for file in ['stage.py','backfill.py']:shutil.copy2(source_tools/file,root/file)
# Validated Linux binaries are staged privately before stopping any live service.
artifacts={Path('/opt/pastrack/pastrack'):Path('/opt/pastrack/v2-staging-20261103/pastrack-receiver'),Path('/www/wwwroot/pastrack/pastrack-web'):Path('/opt/pastrack/v2-staging-20261103/pastrack-web')}
wrappers=[Path('/opt/pastrack/deploy/run-aapanel.sh'),Path('/www/wwwroot/pastrack/deploy/run-aapanel.sh')]
for dest in artifacts:
 assert artifacts[dest].is_file();shutil.copy2(dest,root/(dest.name+'.original'));shutil.copy2(artifacts[dest],dest.with_name(dest.name+'.v2-next'));os.chmod(dest.with_name(dest.name+'.v2-next'),0o750);shutil.chown(dest.with_name(dest.name+'.v2-next'),user='www',group='www')
for i,path in enumerate(wrappers):shutil.copy2(path,root/('wrapper-'+str(i)+'.original'))
password=public.M('config').where('id=?',(1,)).getField('mysql_root')
c=pymysql.connect(host='localhost',user='root',password=password,autocommit=True,cursorclass=pymysql.cursors.DictCursor);q=c.cursor();q.execute("SET time_zone='+00:00'")
def health(url):
 with urllib.request.urlopen(url,timeout=4) as r:return r.read().decode()
def wait_health(url,seconds=20):
 deadline=time.monotonic()+seconds
 while time.monotonic()<deadline:
  try:return health(url)
  except Exception:time.sleep(.3)
 raise RuntimeError('health check failed: '+url)
def project(manager,name,action):
 args=public.dict_obj();args.project_name=name;getattr(manager,action+'_project')(args)
q.execute('SELECT COUNT(*) n FROM information_schema.schemata WHERE schema_name=%s',(target,));assert q.fetchone()['n']==0,'target already exists; inspect/reconcile rather than overwrite'
# Leave nginx upstream/address unchanged. Both managed projects retain their old paths/ports.
with app.test_request_context('/'):
 manager=main();stopped=False;replaced=False
 try:
  assert 'queued":0' in health('http://127.0.0.1:18087/healthz').replace(' ',''),'legacy spool must be drained first'
  project(manager,'pastrack_web','stop');project(manager,'pastrack','stop');stopped=True
  # Confirm no writer survives the manager operation before the final source snapshot.
  for _ in range(50):
   if not manager.get_project_run_state(project_name='pastrack') and not manager.get_project_run_state(project_name='pastrack_web'):break
   time.sleep(.1)
  assert not manager.get_project_run_state(project_name='pastrack'),'receiver did not stop'
  assert not manager.get_project_run_state(project_name='pastrack_web'),'web did not stop'
  env=os.environ.copy();env['PASTRACK_TARGET']=target;env['PASTRACK_SOURCE']='pastrack'
  subprocess.run([sys.executable,str(root/'stage.py'),str(root)],env=env,check=True,stdout=(root/'schema-result.log').open('w'))
  subprocess.run([sys.executable,str(root/'backfill.py'),str(root)],env=env,check=True,stdout=(root/'backfill-result.log').open('w'))
  # Validate raw byte and coordinate/time preservation before activation.
  q.execute('SELECT COUNT(*) n FROM pastrack.gps_positions s JOIN '+target+'.gps_positions t ON t.id=s.id WHERE NOT(t.latitude<=>s.latitude) OR NOT(t.longitude<=>s.longitude) OR NOT(t.gps_time<=>s.gps_time) OR NOT(t.device_send_time<=>s.device_send_time)');assert q.fetchone()['n']==0,'position parity failed'
  for table in ['device_raw_messages','gps_positions','vehicles','gps_devices','device_events']:
   q.execute('SELECT COUNT(*) n FROM pastrack.'+table);source_count=q.fetchone()['n'];q.execute('SELECT COUNT(*) n FROM '+target+'.'+table);assert q.fetchone()['n']==source_count,'count parity failed: '+table
  appPassword=secrets.token_urlsafe(32);username='pastrack_v2_live_app';q.execute('CREATE USER %s@%s IDENTIFIED BY %s',(username,'localhost',appPassword))
  q.execute('SHOW TABLES FROM '+target)
  for row in q.fetchall():
   table=next(iter(row.values()));privilege='SELECT,INSERT' if table=='device_raw_messages' else 'SELECT,INSERT,UPDATE,DELETE';q.execute('GRANT '+privilege+' ON '+target+'.`'+table+'` TO %s@%s',(username,'localhost'))
  for name in ['run','reports']:(runtime/name).mkdir(exist_ok=True)
  subprocess.run(['chown','-R','www:www',str(runtime)],check=True);runtime.chmod(0o750)
  values={'DB_HOST':'127.0.0.1','DB_PORT':'3306','DB_NAME':target,'DB_USER':username,'DB_PASSWORD':appPassword,'PASTRACK_RUNTIME':'production','PASTRACK_SCHEMA':'v2','WEB_ADDR':'127.0.0.1:18089','GPS_HOST':'0.0.0.0','GPS_PORT':'5001','HEALTH_ADDR':'127.0.0.1:18087','REALTIME_SOCKET_PATH':str(runtime/'run/live.sock'),'SPOOL_DIR':'/opt/pastrack/spool','LOCAL_REPORT_DIR':str(runtime/'reports'),'COOKIE_SECURE':'true','RAW_TEXT_LOG':'off'}
  assert not config.exists(),'live v2 config already exists; reconcile before retry'
  config.write_text(''.join('export '+k+'='+shlex.quote(v)+'\n' for k,v in values.items()));config.chmod(0o640);shutil.chown(config,user='root',group='www')
  for dest in artifacts:os.replace(dest.with_name(dest.name+'.v2-next'),dest)
  for path,binary in zip(wrappers,[Path('/opt/pastrack/pastrack'),Path('/www/wwwroot/pastrack/pastrack-web')]):
   temp=path.with_name(path.name+'.next');temp.write_text('#!/bin/bash\nset -euo pipefail\ncd '+str(binary.parent)+'\nsource '+str(config)+'\nexec '+str(binary)+'\n');temp.chmod(0o750);shutil.chown(temp,user='www',group='www');os.replace(temp,path)
  replaced=True
  project(manager,'pastrack_web','start');web=wait_health('http://127.0.0.1:18089/healthz')
  project(manager,'pastrack','start');receiver=wait_health('http://127.0.0.1:18087/healthz')
  assert json.loads(web)['schema']=='v2' and json.loads(receiver)['schema']=='v2','wrong running release'
  with urllib.request.urlopen('https://toko.azlaundry.my.id/',timeout=15) as response:assert response.status==200,'public web unavailable'
  result={'state':'deployed_for_user_acceptance','url':'https://toko.azlaundry.my.id/','database':target,'web_health':json.loads(web),'receiver_health':json.loads(receiver),'backup':str(root/'backup'),'old_database_preserved':True,'web_and_receiver_managed_by_aapanel':True,'limitations':['push delivery not ported','unknown historical assignment remains unresolved','unsupported hardware commands disabled']}
  (root/'cutover-success.json').write_text(json.dumps(result,indent=2));print(json.dumps(result))
 except Exception:
  # Keep evidence privately. Never discard newly accepted raw if an activation check fails.
  if replaced:
   project(manager,'pastrack','stop');project(manager,'pastrack_web','stop')
   q.execute('SELECT id,payload,received_at FROM '+target+'.device_raw_messages WHERE legacy_id IS NULL ORDER BY id')
   for row in q.fetchall():
    filename=Path('/opt/pastrack/spool')/('rollback-v2-%020d.json'%row['id']);record={'payload':row['payload'].decode('utf-8'),'received_at':row['received_at'].isoformat()+'Z','peer':'v2-cutover-recovery'}
    with filename.open('x') as f:json.dump(record,f);f.flush();os.fsync(f.fileno())
    filename.chmod(0o600);shutil.chown(filename,user='www',group='www')
   for dest in artifacts:shutil.copy2(root/(dest.name+'.original'),dest);shutil.chown(dest,user='www',group='www')
   for i,path in enumerate(wrappers):shutil.copy2(root/('wrapper-'+str(i)+'.original'),path);shutil.chown(path,user='www',group='www')
  if stopped:
   project(manager,'pastrack_web','start');project(manager,'pastrack','start')
  raise
 finally:c.close()
