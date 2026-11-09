"""Additive production migration with backups and restricted runtime grants."""
import os,secrets,shlex,shutil,json,time
from pathlib import Path
import mysql.connector
schema=os.environ['DB_NAME'];assert schema in ['pastrack','gps_tracker_rnd']
c=mysql.connector.connect(host=os.environ['DB_HOST'],port=int(os.environ['DB_PORT']),user=os.environ['DB_USER'],password=os.environ['DB_PASSWORD'],database=schema)
q=c.cursor();backup=Path('/root/pastrack-web-backups/enterprise-'+time.strftime('%Y%m%d-%H%M%S'));backup.mkdir(mode=0o700,parents=True)
shutil.copy2('/etc/pastrack/web.env',backup/'web.env');(backup/'web.env').chmod(0o600)
with (backup/'master-before.json').open('w') as f:
 data={}
 for t in ['vehicles','gps_devices']:
  q.execute('SELECT * FROM '+t);columns=[d[0] for d in q.description];data[t]=[dict(zip(columns,r)) for r in q.fetchall()]
 json.dump(data,f,default=str)
(backup/'master-before.json').chmod(0o600)
for statement in Path('/www/wwwroot/pastrack/migrations/001_master_audit.sql').read_text().split(';'):
 if statement.strip():q.execute(statement)
password=secrets.token_urlsafe(30)
for host in ['localhost','127.0.0.1']:
 q.execute('CREATE USER IF NOT EXISTS %s@%s IDENTIFIED BY %s',('pastrack_web_app',host,password))
 q.execute('ALTER USER %s@%s IDENTIFIED BY %s',('pastrack_web_app',host,password))
 q.execute('GRANT SELECT ON `'+schema+'`.* TO %s@%s',('pastrack_web_app',host))
 for t in ['vehicles','gps_devices']:q.execute('GRANT INSERT,UPDATE ON `'+schema+'`.'+t+' TO %s@%s',('pastrack_web_app',host))
 q.execute('GRANT INSERT ON `'+schema+'`.pastrack_audit_log TO %s@%s',('pastrack_web_app',host))
c.commit()
config={k:v for k,v in os.environ.items() if k.startswith('DB_') or k in ['WEB_USER','WEB_PASSWORD_HASH','SESSION_SECRET','WEB_ADDR','COOKIE_SECURE']}
config.update(DB_USER='pastrack_web_app',DB_PASSWORD=password)
path=Path('/etc/pastrack/web.env.next')
with path.open('x') as f:
 for k,v in config.items():f.write('export '+k+'='+shlex.quote(v)+'\n')
path.chmod(0o600);shutil.chown(path,user='www',group='www');os.replace(path,'/etc/pastrack/web.env')
print('Migration complete; runtime privileges restricted. Backup:',backup)
