"""Deploy authorized BLE registration flow; additive nullable-owner migration, web only."""
import os,sys,gzip,hashlib,shutil,subprocess,time,urllib.request,json
from pathlib import Path
sys.path[:0]=['/www/server/panel','/www/server/panel/class','/www/server/panel/class_v2'];os.chdir('/www/server/panel')
import public,pymysql
from BTPanel import app
from projectModelV2.goModel import main
root=Path('/root/pastrack-ble-registration-20261103');target='pastrack_v2_20261103_r2';binary=Path('/www/wwwroot/pastrack/pastrack-web')
root.mkdir(mode=0o700,exist_ok=True);root.chmod(0o700)
password=public.M('config').where('id=?',(1,)).getField('mysql_root');c=pymysql.connect(host='localhost',user='root',password=password,autocommit=True,cursorclass=pymysql.cursors.DictCursor);q=c.cursor()
# Private consistent backup, preserving the receiver's ongoing writes.
snapshot=root/'before-ble-registration.sql.gz'
if not snapshot.exists():
 env=os.environ.copy();env['MYSQL_PWD']=password;p=subprocess.Popen(['mysqldump','-u','root','--single-transaction','--skip-lock-tables','--hex-blob',target],stdout=subprocess.PIPE,stderr=subprocess.PIPE,env=env)
 with gzip.open(snapshot,'wb') as out:shutil.copyfileobj(p.stdout,out)
 p.stderr.read();assert p.wait()==0,'backup failed';snapshot.chmod(0o600)
 with gzip.open(snapshot,'rb') as f:
  while f.read(1024*1024):pass
q.execute('SET SESSION lock_wait_timeout=10')
q.execute('SELECT IS_NULLABLE FROM information_schema.columns WHERE table_schema=%s AND table_name=%s AND column_name=%s',(target,'fuel_tank_profiles','vehicle_id'));column=q.fetchone();assert column,'live target schema missing'
q.execute('USE '+target)
if column['IS_NULLABLE']!='YES':q.execute('ALTER TABLE fuel_tank_profiles MODIFY vehicle_id BIGINT UNSIGNED NULL')
statement='ALTER TABLE fuel_tank_profiles MODIFY vehicle_id BIGINT UNSIGNED NULL';checksum=hashlib.sha256(statement.encode()).hexdigest()
q.execute('INSERT INTO stage_migration_steps(version,step,checksum) VALUES(%s,0,%s) ON DUPLICATE KEY UPDATE checksum=VALUES(checksum)',('004_unassigned_tank_profiles.sql',checksum))
next_file=binary.with_name(binary.name+'.ble-next')
with gzip.open(root/'web-next.gz','rb') as source,next_file.open('wb') as out:shutil.copyfileobj(source,out)
next_file.chmod(0o750);shutil.chown(next_file,user='www',group='www');backup=root/'pastrack-web.before'
if not backup.exists():shutil.copy2(binary,backup);backup.chmod(0o600)
with app.test_request_context('/'):
 manager=main();args=public.dict_obj();args.project_name='pastrack_web';manager.stop_project(args);os.replace(next_file,binary)
 try:
  manager.start_project(args)
  for i in range(50):
   try:
    with urllib.request.urlopen('http://127.0.0.1:18089/healthz',timeout=2) as r:assert json.load(r)['schema']=='v2'
    break
   except Exception:
    if i==49:raise
    time.sleep(.2)
  with urllib.request.urlopen('https://toko.azlaundry.my.id/assets/stitch.js',timeout=10) as r:assert b'async function registerBLE' in r.read()
  with urllib.request.urlopen('http://127.0.0.1:18087/healthz',timeout=3) as r:receiver=json.load(r);assert receiver['status']=='ok'
  result={'deployed':True,'feature':'atomic BLE vehicle registration','database':target,'migration':'004_unassigned_tank_profiles.sql','receiver_restarted':False,'receiver_health':receiver,'backup':str(root)};(root/'deploy-result.json').write_text(json.dumps(result,indent=2));print(json.dumps(result))
 except Exception:
  manager.stop_project(args);shutil.copy2(backup,binary);binary.chmod(0o750);shutil.chown(binary,user='www',group='www');manager.start_project(args);raise
finally_close=True
c.close()
