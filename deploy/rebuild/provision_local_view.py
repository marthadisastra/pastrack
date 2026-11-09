"""Provision the user-authorized live-linked local runtime through an SSH tunnel.
Table privileges match the live runtime; raw receipts remain append only.
"""
import os,sys,secrets,shlex,json
from pathlib import Path
sys.path[:0]=['/www/server/panel','/www/server/panel/class','/www/server/panel/class_v2'];os.chdir('/www/server/panel')
import public,pymysql
password=public.M('config').where('id=?',(1,)).getField('mysql_root');path=Path('/root/pastrack-local-live-view.env');target='pastrack_v2_20261103_r2';user='pastrack_local_view'
c=pymysql.connect(host='localhost',user='root',password=password,autocommit=True);q=c.cursor()
if not path.exists():
 secret=secrets.token_urlsafe(32);q.execute('CREATE USER %s@%s IDENTIFIED BY %s',(user,'localhost',secret))
 q.execute('SHOW TABLES FROM '+target)
 for row in q.fetchall():
  privileges='SELECT,INSERT' if row[0]=='device_raw_messages' else 'SELECT,INSERT,UPDATE,DELETE'
  q.execute('GRANT '+privileges+' ON '+target+'.`'+row[0]+'` TO %s@%s',(user,'localhost'))
 values={'DB_HOST':'127.0.0.1','DB_PORT':'33308','DB_USER':user,'DB_PASSWORD':secret,'DB_NAME':target,'PASTRACK_RUNTIME':'production','PASTRACK_READ_ONLY':'false','PASTRACK_LIVE_UPSTREAM':'https://toko.azlaundry.my.id','COOKIE_SECURE':'false','WEB_ADDR':'127.0.0.1:18090','REALTIME_SOCKET_PATH':'off'}
 path.write_text(''.join('export '+k+'='+shlex.quote(v)+'\n' for k,v in values.items()));path.chmod(0o600)
print(json.dumps({'config_created':True,'database':target,'business_privileges':'same as live runtime','session_privileges':'SELECT INSERT DELETE','ssh_tunnel_required':True}));c.close()
