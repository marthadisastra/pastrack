"""Start v2 privately for server integration tests; never replace live services."""
import os,sys,secrets,shlex,subprocess,json,hashlib,datetime,urllib.request
from pathlib import Path
sys.path[:0]=['/www/server/panel','/www/server/panel/class','/www/server/panel/class_v2'];os.chdir('/www/server/panel')
import public,pymysql
root=Path('/root/pastrack-rebuild-20261103');runtime=Path('/opt/pastrack/v2-staging-20261103');runtime.mkdir(exist_ok=True)
password=public.M('config').where('id=?',(1,)).getField('mysql_root');target='pastrack_v2_stage_20261103'
c=pymysql.connect(host='localhost',user='root',password=password,autocommit=True,cursorclass=pymysql.cursors.DictCursor);q=c.cursor();q.execute("SET time_zone='+00:00'")
config=Path('/etc/pastrack/v2-stage.env')
if not config.exists():
 appPassword=secrets.token_urlsafe(32);username='pastrack_v2_stage_app'
 q.execute('CREATE USER %s@%s IDENTIFIED BY %s',(username,'localhost',appPassword))
 q.execute('SHOW TABLES FROM '+target);tables=[next(iter(r.values())) for r in q.fetchall()]
 for table in tables:
  privilege='SELECT,INSERT' if table=='device_raw_messages' else 'SELECT,INSERT,UPDATE,DELETE'
  q.execute('GRANT '+privilege+' ON '+target+'.`'+table+'` TO %s@%s',(username,'localhost'))
 values={'DB_HOST':'127.0.0.1','DB_PORT':'3306','DB_NAME':target,'DB_USER':username,'DB_PASSWORD':appPassword,'PASTRACK_RUNTIME':'production','PASTRACK_SCHEMA':'v2','WEB_ADDR':'127.0.0.1:18092','GPS_HOST':'127.0.0.1','GPS_PORT':'15002','HEALTH_ADDR':'127.0.0.1:18093','REALTIME_SOCKET_PATH':str(runtime/'run/live.sock'),'SPOOL_DIR':str(runtime/'spool'),'LOCAL_REPORT_DIR':str(runtime/'reports'),'COOKIE_SECURE':'true'}
 config.write_text(''.join('export '+k+'='+shlex.quote(v)+'\n' for k,v in values.items()));config.chmod(0o640);subprocess.run(['chown','root:www',str(config)],check=True)
for name in ['spool','run','reports']:(runtime/name).mkdir(exist_ok=True)
subprocess.run(['chown','-R','www:www',str(runtime)],check=True);runtime.chmod(0o750)
for name,unit in [('web','pastrack-v2-stage-web'),('receiver','pastrack-v2-stage-receiver')]:
 binary=runtime/('pastrack-'+name);binary.chmod(0o750)
 script=runtime/('run-'+name+'.sh');script.write_text('#!/bin/bash\nset -euo pipefail\ncd '+str(runtime)+'\nsource '+str(config)+'\nexec '+str(binary)+'\n');script.chmod(0o750);subprocess.run(['chown','www:www',str(script),str(binary)],check=True)
 state=subprocess.run(['systemctl','is-active','--quiet',unit]).returncode
 if state==0:continue
 subprocess.run(['systemd-run','--unit='+unit,'--uid=www','--property=Restart=on-failure','--property=UMask=0077','/bin/bash',str(script)],check=True,stdout=subprocess.DEVNULL)
print(json.dumps({'staging_web':'127.0.0.1:18092','staging_receiver':'127.0.0.1:15002','target':target,'old_live_services_changed':False}))
