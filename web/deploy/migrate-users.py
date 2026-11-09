"""Run with DBA credentials. --qa creates a disposable isolated copy before migration."""
import argparse,datetime,json,os,secrets,shlex,shutil,time
from pathlib import Path
import mysql.connector
p=argparse.ArgumentParser();p.add_argument('--qa',action='store_true');args=p.parse_args()
source=os.environ['DB_NAME'];assert source=='pastrack'
c=mysql.connector.connect(host=os.environ['DB_HOST'],port=int(os.environ['DB_PORT']),user=os.environ['DB_USER'],password=os.environ['DB_PASSWORD'])
q=c.cursor();schema='pastrack_users_qa' if args.qa else source
if args.qa:
 q.execute("SHOW DATABASES LIKE 'pastrack_users_qa'");assert q.fetchone() is None,'QA schema already exists; inspect first'
 q.execute('CREATE DATABASE pastrack_users_qa CHARACTER SET utf8mb4')
 for t in ['vehicles','gps_devices','device_raw_messages','gps_positions','device_events','pastrack_audit_log']:
  q.execute(f'CREATE TABLE {schema}.{t} LIKE {source}.{t}');q.execute(f'INSERT INTO {schema}.{t} SELECT * FROM {source}.{t}')
else:
 backup=Path('/root/pastrack-web-backups/users-'+time.strftime('%Y%m%d-%H%M%S'));backup.mkdir(parents=True,mode=0o700);shutil.copy2('/etc/pastrack/web.env',backup/'web.env');(backup/'web.env').chmod(0o600)
q.execute('USE '+schema)
for statement in Path('/www/wwwroot/pastrack/migrations/002_users_sessions.sql').read_text().split(';'):
 if statement.strip():q.execute(statement)
q.execute('SELECT COUNT(*) FROM web_users');n=q.fetchone()[0]
if n==0:
 user=os.environ.get('WEB_USER','admin').strip().lower();digest=os.environ['WEB_PASSWORD_HASH'];assert len(digest)==64
 now=datetime.datetime.now(datetime.timezone.utc).replace(tzinfo=None)
 q.execute("INSERT INTO web_users (username,full_name,role,password_hash,password_algo,is_active,created_at,updated_at) VALUES (%s,'Administrator','admin',%s,'legacy_sha256',1,%s,%s)",(user,digest,now,now))
 uid=q.lastrowid
 q.execute('INSERT INTO pastrack_audit_log (occurred_at,actor,action,entity,entity_id,after_data,request_id) VALUES (%s,%s,%s,%s,%s,%s,%s)',(now,'system:migration','bootstrap','users',uid,json.dumps({'id':uid,'username':user,'role':'admin','is_active':1}),'users-bootstrap-'+secrets.token_hex(6)))
login='pastrack_users_qa' if args.qa else 'pastrack_web_app';password=secrets.token_urlsafe(30) if args.qa else None
for host in ['localhost','127.0.0.1']:
 if args.qa:
  q.execute('CREATE USER %s@%s IDENTIFIED BY %s',(login,host,password))
  q.execute('GRANT SELECT ON '+schema+'.* TO %s@%s',(login,host))
  for t in ['vehicles','gps_devices']:q.execute('GRANT INSERT,UPDATE ON '+schema+'.'+t+' TO %s@%s',(login,host))
  q.execute('GRANT INSERT ON '+schema+'.pastrack_audit_log TO %s@%s',(login,host))
 q.execute('GRANT SELECT,INSERT,UPDATE ON '+schema+'.web_users TO %s@%s',(login,host))
 q.execute('GRANT SELECT,INSERT,UPDATE,DELETE ON '+schema+'.web_sessions TO %s@%s',(login,host))
c.commit()
if args.qa:
 config={k:v for k,v in os.environ.items() if k.startswith('DB_') or k in ['WEB_USER','WEB_PASSWORD_HASH','SESSION_SECRET']}
 config.update(DB_NAME=schema,DB_USER=login,DB_PASSWORD=password,WEB_ADDR='127.0.0.1:18090',COOKIE_SECURE='false')
 path=Path('/etc/pastrack/users-qa.env')
 with path.open('x') as f:
  for k,v in config.items():f.write('export '+k+'='+shlex.quote(v)+'\n')
 path.chmod(0o600)
print('Users/session migration complete; existing administrator password preserved. Target:',schema)
