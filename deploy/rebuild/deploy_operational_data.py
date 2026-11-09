"""Deploy vehicle/date-only Fuel Events filters and scoped All overview."""
import os,sys,gzip,hashlib,shutil,subprocess,time,urllib.request,json,secrets
from pathlib import Path
sys.path[:0]=['/www/server/panel','/www/server/panel/class','/www/server/panel/class_v2'];os.chdir('/www/server/panel')
import public,pymysql
from BTPanel import app
from projectModelV2.goModel import main
root=Path(sys.argv[1]);assert str(root).startswith('/root/pastrack-operational-data-')
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
  assert b'operationalRegisters' in get('https://toko.azlaundry.my.id/assets/operational-register.js?v=20261104-data1')
  from remove_october_fixture import remove_october_fixture
  removed=remove_october_fixture(c, root/('fixture-investigations-'+secrets.token_hex(6)+'.json'))
  q.execute("SELECT COUNT(*) n FROM vehicle_assignments WHERE provenance='fixture_oct31'")
  assert q.fetchone()['n']==0
  receiver=json.loads(get('http://127.0.0.1:18087/healthz'));assert receiver['status']=='ok'
  result={'deployed':True,'feature':'operational data registers','removed_fixture_vehicles':removed,'receiver_restarted':False,'receiver_health':receiver,'backup':str(root),'binary_sha256':hashlib.sha256(binary.read_bytes()).hexdigest()}
  (root/'deploy-result.json').write_text(json.dumps(result,indent=2));print(json.dumps(result))
 except Exception:
  manager.stop_project(args);rollback=binary.with_name('pastrack-web.focus-rollback');shutil.copy2(backup,rollback);rollback.chmod(0o750);shutil.chown(rollback,user='www',group='www');os.replace(rollback,binary);manager.start_project(args);raise
c.close()
