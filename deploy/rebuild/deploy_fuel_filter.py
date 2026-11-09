"""Deploy vehicle/date-only Fuel Events filters and scoped All overview."""
import os,sys,gzip,hashlib,shutil,subprocess,time,urllib.request,json,secrets
from pathlib import Path
sys.path[:0]=['/www/server/panel','/www/server/panel/class','/www/server/panel/class_v2'];os.chdir('/www/server/panel')
import public,pymysql
from BTPanel import app
from projectModelV2.goModel import main
root=Path(sys.argv[1]);assert str(root).startswith('/root/pastrack-fuel-filter-')
target='pastrack_v2_20261103_r2';binary=Path('/www/wwwroot/pastrack/pastrack-web')
password=public.M('config').where('id=?',(1,)).getField('mysql_root')
c=pymysql.connect(host='localhost',user='root',password=password,charset='utf8mb4',autocommit=True,cursorclass=pymysql.cursors.DictCursor);q=c.cursor();q.execute("SET time_zone='+00:00'")
snapshot=root/'before-release.sql.gz'
if not snapshot.exists():
 env=os.environ.copy();env['MYSQL_PWD']=password
 p=subprocess.Popen(['mysqldump','-u','root','--single-transaction','--skip-lock-tables','--hex-blob',target],stdout=subprocess.PIPE,stderr=subprocess.PIPE,env=env)
 with gzip.open(snapshot,'wb') as out:shutil.copyfileobj(p.stdout,out)
 p.stderr.read();assert p.wait()==0,'backup failed';snapshot.chmod(0o600)
q.execute('USE '+target)
q.execute("SELECT id vehicle_id,plate_number FROM vehicles WHERE plate_number IN ('B 9101 PTR','B 9102 PTR') ORDER BY plate_number")
imported=q.fetchall();assert len(imported)==2
nextfile=binary.with_name('pastrack-web.focus-next')
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
  fuel_js=get('https://toko.azlaundry.my.id/assets/fuel-events.js?v=20261104-vehicle3')
  trip_js=get('https://toko.azlaundry.my.id/assets/trip-history.js?v=20261104-vehicle3')
  assert b'fe-source' not in fuel_js and b'th-source' not in trip_js
  assert b'fe-fleet' not in fuel_js and b'fe-sensor' not in fuel_js
  assert b'Start date (WIB)' in fuel_js and b'End date (WIB)' in fuel_js
  assert b'data-series' in fuel_js and b'draw(data.positions' not in fuel_js
  # Short lived session for an existing admin, used only for read-only release verification.
  q.execute("SELECT u.id,u.auth_version FROM users u JOIN user_roles ur ON ur.user_id=u.id JOIN roles r ON r.id=ur.role_id WHERE r.code='admin' AND u.is_active=1 AND u.deleted_at IS NULL LIMIT 1");user=q.fetchone();assert user
  cookie=secrets.token_hex(32);digest=hashlib.sha256(cookie.encode()).hexdigest()
  q.execute("INSERT INTO sessions(user_id,token_hash,auth_version,expires_at) VALUES(%s,%s,%s,UTC_TIMESTAMP(3)+INTERVAL 10 MINUTE)",(user['id'],digest,user['auth_version']))
  try:
   from urllib.parse import urlencode
   verified=[]
   for record in imported:
    query=urlencode({'vehicle_id':record['vehicle_id'],'from':'2026-10-30T23:00:00Z','to':'2026-10-31T05:01:00Z'})
    data=json.loads(get('http://127.0.0.1:18089/api/v1/fuel/events/workspace?'+query,cookie))['data']
    history=json.loads(get('http://127.0.0.1:18089/api/v1/trips/history?'+query,cookie))['data']
    assert len(data['positions'])==721 and len(data['readings'])==721 and len(history['trips'])==2
    if record['plate_number']=='B 9102 PTR':assert len(data['events'])==1 and float(data['events'][0]['delta_l'])==-40
    else:assert len(data['events'])==0
    verified.append({'plate_number':record['plate_number'],'vehicle_id':record['vehicle_id'],'positions':len(data['positions']),'readings':len(data['readings']),'trips':len(history['trips']),'events':len(data['events'])})
   all_data=json.loads(get('http://127.0.0.1:18089/api/v1/fuel/events/workspace?'+urlencode({'vehicle_id':0,'from':'2026-10-30T17:00:00Z','to':'2026-10-31T17:00:00Z'}),cookie))['data']
   assert all_data['all_vehicles'] is True and len(all_data['events'])>=1 and not all_data['positions'] and not all_data['readings']
   from urllib.error import HTTPError
   try:get('http://127.0.0.1:18089/api/v1/fuel/events/simulation?scenario=normal',cookie);raise AssertionError('retired endpoint is still served')
   except HTTPError as e:assert e.code==404
  finally:q.execute('DELETE FROM sessions WHERE token_hash=%s',(digest,))
  receiver=json.loads(get('http://127.0.0.1:18087/healthz'));assert receiver['status']=='ok'
  result={'deployed':True,'feature':'vehicle/date-only Fuel Events filters and All event overview','vehicles':verified,'receiver_restarted':False,'receiver_health':receiver,'backup':str(root),'binary_sha256':hashlib.sha256(binary.read_bytes()).hexdigest()}
  (root/'deploy-result.json').write_text(json.dumps(result,indent=2));print(json.dumps(result))
 except Exception:
  manager.stop_project(args);rollback=binary.with_name('pastrack-web.focus-rollback');shutil.copy2(backup,rollback);rollback.chmod(0o750);shutil.chown(rollback,user='www',group='www');os.replace(rollback,binary);manager.start_project(args);raise
c.close()
