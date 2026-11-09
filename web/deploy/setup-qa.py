"""Create an isolated disposable QA database. Run with existing DB admin env."""
import os,secrets,shlex
from pathlib import Path
import mysql.connector
source=os.environ['DB_NAME']
assert source.replace('_','').isalnum(), 'Unexpected schema name'
root='/www/wwwroot/pastrack'
c=mysql.connector.connect(host=os.environ['DB_HOST'],port=int(os.environ['DB_PORT']),user=os.environ['DB_USER'],password=os.environ['DB_PASSWORD'])
q=c.cursor();q.execute("SHOW DATABASES LIKE 'pastrack_web_qa'")
assert q.fetchone() is None,'QA database already exists; inspect before resetting'
q.execute('CREATE DATABASE pastrack_web_qa CHARACTER SET utf8mb4')
for t in ['vehicles','gps_devices','device_raw_messages','gps_positions','device_events']:
 q.execute(f'CREATE TABLE pastrack_web_qa.{t} LIKE `{source}`.{t}')
 q.execute(f'INSERT INTO pastrack_web_qa.{t} SELECT * FROM `{source}`.{t}')
q.execute('USE pastrack_web_qa')
for statement in Path(root+'/migrations/001_master_audit.sql').read_text().split(';'):
 if statement.strip():q.execute(statement)
password=secrets.token_urlsafe(30)
for host in ['localhost','127.0.0.1']:
 q.execute('CREATE USER IF NOT EXISTS %s@%s IDENTIFIED BY %s',('pastrack_web_qa',host,password))
 q.execute('ALTER USER %s@%s IDENTIFIED BY %s',('pastrack_web_qa',host,password))
 q.execute('GRANT SELECT ON pastrack_web_qa.* TO %s@%s',('pastrack_web_qa',host))
 for t in ['vehicles','gps_devices']:q.execute('GRANT INSERT,UPDATE ON pastrack_web_qa.'+t+' TO %s@%s',('pastrack_web_qa',host))
 q.execute('GRANT INSERT ON pastrack_web_qa.pastrack_audit_log TO %s@%s',('pastrack_web_qa',host))
c.commit()
config={k:v for k,v in os.environ.items() if k.startswith('DB_') or k in ['WEB_USER','WEB_PASSWORD_HASH','SESSION_SECRET']}
config.update(DB_NAME='pastrack_web_qa',DB_USER='pastrack_web_qa',DB_PASSWORD=password,WEB_ADDR='127.0.0.1:18090',COOKIE_SECURE='false')
path=Path('/etc/pastrack/web-qa.env')
with path.open('x') as f:
 for k,v in config.items():f.write('export '+k+'='+shlex.quote(v)+'\n')
path.chmod(0o600)
print('QA database copied, migrated and restricted QA user created. No production rows modified.')
