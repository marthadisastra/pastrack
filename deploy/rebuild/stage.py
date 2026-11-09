"""Server-side staging only. Uses existing aaPanel DBA access without printing secrets.
Does not stop services, modify source tables, replace runtime config, or deploy binaries.
"""
import sys,os,json,hashlib,subprocess,gzip,shutil,time
from pathlib import Path
sys.path[:0]=['/www/server/panel','/www/server/panel/class','/www/server/panel/class_v2']
os.chdir('/www/server/panel')
import public,pymysql
password=public.M('config').where('id=?',(1,)).getField('mysql_root')
root=Path(sys.argv[1]);source='pastrack';target=os.environ.get('PASTRACK_TARGET','pastrack_v2_stage_20261103')
assert target.startswith('pastrack_v2_') and all(ch.isalnum() or ch=='_' for ch in target)
assert root.is_absolute() and root.is_dir()
backup=root/'backup';backup.mkdir(mode=0o700,exist_ok=True)
cn=pymysql.connect(host='localhost',user='root',password=password,charset='utf8mb4',autocommit=True,cursorclass=pymysql.cursors.DictCursor)
q=cn.cursor();q.execute("SET time_zone='+00:00'")
# Credentials only in subprocess environment; never command arguments/logs.
env=os.environ.copy();env['MYSQL_PWD']=password
snapshot=backup/'pastrack-source.sql.gz'
if not snapshot.exists():
 process=subprocess.Popen(['mysqldump','-u','root','--single-transaction','--skip-lock-tables','--routines','--triggers','--events','--hex-blob',source],stdout=subprocess.PIPE,stderr=subprocess.PIPE,env=env)
 with gzip.open(snapshot,'wb') as out:shutil.copyfileobj(process.stdout,out)
 error=process.stderr.read();code=process.wait();assert code==0,'source backup failed'
 snapshot.chmod(0o600)
 with gzip.open(snapshot,'rb') as f:
  while f.read(1024*1024):pass
 for path in ['/etc/pastrack/web.env','/opt/pastrack/.env','/opt/pastrack/pastrack','/www/wwwroot/pastrack/pastrack-web']:
  dest=backup/(path.replace('/','_').lstrip('_'));shutil.copy2(path,dest);dest.chmod(0o600)
 q.execute('SHOW TABLES FROM '+source);tables=[next(iter(r.values())) for r in q.fetchall()]
 manifest={}
 for table in tables:
  q.execute('SELECT COUNT(*) AS n FROM '+source+'.`'+table+'`');manifest[table]=q.fetchone()['n']
 (backup/'source-counts.json').write_text(json.dumps(manifest,indent=2));(backup/'source-counts.json').chmod(0o600)
q.execute('CREATE DATABASE IF NOT EXISTS '+target+' CHARACTER SET utf8mb4')
q.execute('USE '+target)
q.execute('CREATE TABLE IF NOT EXISTS stage_migration_steps(version VARCHAR(128),step INT,checksum CHAR(64),PRIMARY KEY(version,step))')
for file in sorted((root/'migrations').glob('*.sql')):
 text='\n'.join(line for line in file.read_text().splitlines() if not line.strip().startswith('--'))
 for i,statement in enumerate(text.split(';')):
  statement=statement.strip()
  if not statement:continue
  checksum=hashlib.sha256(statement.encode()).hexdigest();q.execute('SELECT checksum FROM stage_migration_steps WHERE version=%s AND step=%s',(file.name,i));old=q.fetchone()
  if old:assert old['checksum']==checksum,'changed applied migration';continue
  q.execute(statement);q.execute('INSERT INTO stage_migration_steps VALUES(%s,%s,%s)',(file.name,i,checksum))
q.execute('SELECT COUNT(*) AS n FROM information_schema.referential_constraints WHERE constraint_schema=%s',(target,));fks=q.fetchone()['n']
result={'source':source,'target':target,'source_backup':str(snapshot),'backup_sha256':hashlib.sha256(snapshot.read_bytes()).hexdigest(),'foreign_keys':fks,'source_services_changed':False}
(root/'stage-result.json').write_text(json.dumps(result,indent=2));print(json.dumps(result));cn.close()
