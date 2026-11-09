"""Deploy approved Fuel Events playback and simulations; atomic web binary replacement."""
import os,sys,gzip,hashlib,shutil,subprocess,time,urllib.request,json,secrets
from pathlib import Path
sys.path[:0]=['/www/server/panel','/www/server/panel/class','/www/server/panel/class_v2'];os.chdir('/www/server/panel')
import public,pymysql
from BTPanel import app
from projectModelV2.goModel import main
root=Path(sys.argv[1]);assert str(root).startswith('/root/pastrack-fuel-events-')
target='pastrack_v2_20261103_r2';binary=Path('/www/wwwroot/pastrack/pastrack-web')
password=public.M('config').where('id=?',(1,)).getField('mysql_root')
c=pymysql.connect(host='localhost',user='root',password=password,autocommit=True,cursorclass=pymysql.cursors.DictCursor);q=c.cursor();q.execute("SET time_zone='+00:00'")
snapshot=root/'before-fuel-events.sql.gz'
if not snapshot.exists():
 env=os.environ.copy();env['MYSQL_PWD']=password
 p=subprocess.Popen(['mysqldump','-u','root','--single-transaction','--skip-lock-tables','--hex-blob',target],stdout=subprocess.PIPE,stderr=subprocess.PIPE,env=env)
 with gzip.open(snapshot,'wb') as out:shutil.copyfileobj(p.stdout,out)
 p.stderr.read();assert p.wait()==0,'backup failed';snapshot.chmod(0o600)
q.execute('USE '+target);q.execute('SET SESSION lock_wait_timeout=10')
for statement in (root/'006_fuel_event_investigations.sql').read_text().split(';'):
 if statement.strip():q.execute(statement)
checksum=hashlib.sha256((root/'006_fuel_event_investigations.sql').read_bytes()).hexdigest()
q.execute('INSERT IGNORE INTO stage_migration_steps(version,step,checksum) VALUES(%s,0,%s)',('006_fuel_event_investigations.sql',checksum))
# Runtime grants are table-specific; grant only the operations needed by this feature.
q.execute("SELECT Host FROM mysql.user WHERE User='pastrack_v2_live_app'")
for account in q.fetchall():
 q.execute('GRANT SELECT,INSERT,UPDATE ON '+target+'.fuel_event_investigations TO %s@%s',('pastrack_v2_live_app',account['Host']))
nextfile=binary.with_name('pastrack-web.fuel-next')
with gzip.open(root/'web-next.gz','rb') as source,nextfile.open('wb') as out:shutil.copyfileobj(source,out)
nextfile.chmod(0o750);shutil.chown(nextfile,user='www',group='www');backup=root/'pastrack-web.before'
if not backup.exists():shutil.copy2(binary,backup);backup.chmod(0o600)
def get(url,cookie=None):
 req=urllib.request.Request(url,headers={'Cookie':'pastrack_v2_session='+cookie} if cookie else {})
 with urllib.request.urlopen(req,timeout=25) as r:return r.read()
with app.test_request_context('/'):
 manager=main();args=public.dict_obj();args.project_name='pastrack_web';manager.stop_project(args)
 try:
  os.replace(nextfile,binary);manager.start_project(args)
  for i in range(50):
   try:assert json.loads(get('http://127.0.0.1:18089/healthz'))['status']=='ok';break
   except Exception:
    if i==49:raise
    time.sleep(.2)
  assert b'feLoadSimulation' in get('https://toko.azlaundry.my.id/assets/fuel-events.js?v=20261104-play1')
  assert b'feStartPlayback' in get('https://toko.azlaundry.my.id/assets/fuel-player.js?v=20261104-play1')
  # Short lived session for an existing admin, used only for read-only release verification.
  q.execute("SELECT u.id,u.auth_version FROM users u JOIN user_roles ur ON ur.user_id=u.id JOIN roles r ON r.id=ur.role_id WHERE r.code='admin' AND u.is_active=1 AND u.deleted_at IS NULL LIMIT 1");user=q.fetchone();assert user
  cookie=secrets.token_hex(32);digest=hashlib.sha256(cookie.encode()).hexdigest()
  q.execute("INSERT INTO sessions(user_id,token_hash,auth_version,expires_at) VALUES(%s,%s,%s,UTC_TIMESTAMP(3)+INTERVAL 10 MINUTE)",(user['id'],digest,user['auth_version']))
  try:
   fleet=json.loads(get('http://127.0.0.1:18089/api/v1/fleet/live',cookie))['data']
   if fleet:
    from datetime import datetime,timedelta,timezone
    now=datetime.now(timezone.utc);start=now-timedelta(days=1)
    from urllib.parse import urlencode
    url='http://127.0.0.1:18089/api/v1/fuel/events/workspace?'+urlencode({'vehicle_id':fleet[0]['id'],'from':start.isoformat(),'to':now.isoformat()})
    data=json.loads(get(url,cookie))['data'];assert 'events' in data and 'readings' in data
   else:data={'events':[],'readings':[]}
   scenarios={}
   for scenario in ['normal','anomaly']:
    sample=json.loads(get('http://127.0.0.1:18089/api/v1/fuel/events/simulation?scenario='+scenario,cookie))['data']
    assert sample['simulation'] is True and len(sample['positions'])==361 and len(sample['readings'])==361
    assert len(sample['events'])==(1 if scenario=='anomaly' else 0)
    scenarios[scenario]={'gps_samples':len(sample['positions']),'fuel_samples':len(sample['readings']),'events':len(sample['events'])}
    if scenario=='anomaly':assert abs(sample['readings'][180]['fuel_liters']-sample['readings'][182]['fuel_liters']-40)<1e-8
  finally:q.execute('DELETE FROM sessions WHERE token_hash=%s',(digest,))
  receiver=json.loads(get('http://127.0.0.1:18087/healthz'));assert receiver['status']=='ok'
  result={'deployed':True,'feature':'Fuel Events synchronized playback and October 31 simulations','scenarios':scenarios,'database':target,'events':len(data['events']),'readings':len(data['readings']),'receiver_restarted':False,'receiver_health':receiver,'backup':str(root),'binary_sha256':hashlib.sha256(binary.read_bytes()).hexdigest()}
  (root/'deploy-result.json').write_text(json.dumps(result,indent=2));print(json.dumps(result))
 except Exception:
  manager.stop_project(args);rollback=binary.with_name('pastrack-web.fuel-rollback');shutil.copy2(backup,rollback);rollback.chmod(0o750);shutil.chown(rollback,user='www',group='www');os.replace(rollback,binary);manager.start_project(args);raise
c.close()
