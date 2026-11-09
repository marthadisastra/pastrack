"""Activate the authorized GTDOS transport. No output command is issued by this script."""
import os,sys,shutil,time,json,gzip,subprocess,hashlib,urllib.request
from pathlib import Path
sys.path[:0]=['/www/server/panel','/www/server/panel/class','/www/server/panel/class_v2'];os.chdir('/www/server/panel')
import public,pymysql
from BTPanel import app
from projectModelV2.goModel import main
root=Path('/root/pastrack-remote-output-20261103');root.mkdir(mode=0o700,exist_ok=True);root.chmod(0o700)
config=Path('/etc/pastrack/v2-live.env');receiver=Path('/opt/pastrack/pastrack');web=Path('/www/wwwroot/pastrack/pastrack-web')
for name,file in [('runtime.env',config),('receiver',receiver),('web',web)]:
 dest=root/('before-'+name)
 if not dest.exists():shutil.copy2(file,dest);dest.chmod(0o600 if name=='runtime.env' else 0o750)
password=public.M('config').where('id=?',(1,)).getField('mysql_root')
c=pymysql.connect(host='localhost',user='root',password=password,database='pastrack_v2_20261103_r2',autocommit=True,cursorclass=pymysql.cursors.DictCursor);q=c.cursor()
backup=root/'before.sql.gz'
if not backup.exists():
 env=os.environ.copy();env['MYSQL_PWD']=password
 process=subprocess.Popen(['mysqldump','-u','root','--single-transaction','--skip-lock-tables','--hex-blob','pastrack_v2_20261103_r2'],stdout=subprocess.PIPE,stderr=subprocess.PIPE,env=env)
 with gzip.open(backup,'wb')as out:shutil.copyfileobj(process.stdout,out)
 process.stderr.read();assert process.wait()==0,'backup failed';backup.chmod(0o600)
statement=(root/'005_remote_output.sql').read_text();q.execute(statement)
q.execute("INSERT INTO stage_migration_steps(version,step,checksum) VALUES('005_remote_output.sql',0,%s) ON DUPLICATE KEY UPDATE checksum=VALUES(checksum)",(hashlib.sha256(statement.encode()).hexdigest(),))
import shlex
values={}
for line in config.read_text().splitlines():
 line=line.removeprefix('export ').strip()
 if '=' in line and not line.startswith('#'):
  key,value=line.split('=',1);parts=shlex.split(value);values[key]=parts[0] if parts else ''
username=values['DB_USER'];q.execute('SELECT User,Host FROM mysql.user WHERE User=%s',(username,));accounts=q.fetchall()
for account in accounts:
 q.execute('GRANT SELECT,INSERT,UPDATE ON pastrack_v2_20261103_r2.remote_output_commands TO %s@%s',(account['User'],account['Host']))
# The receiver and web share a protected local Unix socket; no public command port.
lines=[line for line in config.read_text().splitlines() if not line.removeprefix('export ').startswith('REMOTE_OUTPUT_SOCKET=')]
lines.append("export REMOTE_OUTPUT_SOCKET='/opt/pastrack/v2-live/run/output.sock'")
config.write_text('\n'.join(lines)+'\n');config.chmod(0o640);shutil.chown(config,user='root',group='www')
for file,name in [(receiver,'receiver-next'),(web,'web-next')]:
 staged=file.with_name(file.name+'.remote-next');shutil.copy2(root/name,staged);staged.chmod(0o750);shutil.chown(staged,user='www',group='www')
with app.test_request_context('/'):
 manager=main()
 def control(name,action):
  args=public.dict_obj();args.project_name=name;return getattr(manager,action+'_project')(args)
 try:
  control('pastrack','stop');os.replace(receiver.with_name(receiver.name+'.remote-next'),receiver);control('pastrack','start')
  for i in range(40):
   try:
    with urllib.request.urlopen('http://127.0.0.1:18087/healthz',timeout=2)as response:assert json.load(response)['status']=='ok'
    assert Path('/opt/pastrack/v2-live/run/output.sock').is_socket();break
   except Exception:
    if i==39:raise
    time.sleep(.25)
  control('pastrack_web','stop');os.replace(web.with_name(web.name+'.remote-next'),web);control('pastrack_web','start')
  for i in range(40):
   try:
    with urllib.request.urlopen('http://127.0.0.1:18089/healthz',timeout=2)as response:assert json.load(response)['status']=='ok'
    break
   except Exception:
    if i==39:raise
    time.sleep(.25)
  print(json.dumps({'receiver':'healthy','web':'healthy','command_socket':'ready','commands_issued':0,'database_backup':str(backup)}))
 except Exception:
  control('pastrack','stop');control('pastrack_web','stop');shutil.copy2(root/'before-runtime.env',config);config.chmod(0o640);shutil.chown(config,user='root',group='www')
  for file,name in [(receiver,'receiver'),(web,'web')]:shutil.copy2(root/('before-'+name),file);file.chmod(0o750);shutil.chown(file,user='www',group='www')
  Path('/opt/pastrack/v2-live/run/output.sock').unlink(missing_ok=True);control('pastrack','start');control('pastrack_web','start');raise
c.close()
