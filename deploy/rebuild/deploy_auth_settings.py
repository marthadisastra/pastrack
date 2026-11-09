"""Web-only auth rollout. Run with aaPanel Python after staging release files.
Additive migration 017 only; preserve existing users, receiver and business tables.
SMTP remains disabled until an administrator configures it in Settings.
"""
import base64, gzip, hashlib, json, os, secrets, shlex, shutil, sys, tarfile, time, urllib.request
from pathlib import Path
sys.path[:0] = ['/www/server/panel', '/www/server/panel/class', '/www/server/panel/class_v2']
os.chdir('/www/server/panel')
import public, pymysql
from BTPanel import app
from projectModelV2.goModel import main
stage = Path(sys.argv[1]).resolve()
assert str(stage).startswith('/root/pastrack-auth-')
assert not (stage/'success.json').exists(), 'release already deployed'
envfile = Path('/etc/pastrack/v2-live.env')
original_env = envfile.read_bytes()
values = {}
for line in original_env.decode().splitlines():
    items = shlex.split(line)
    if items and items[0] == 'export': items = items[1:]
    if items and '=' in items[0]:
        key, value = items[0].split('=', 1); values[key] = value
assert values['DB_NAME'] == 'pastrack_v2_20261103_r2'
assert values.get('WEB_ADDR') == '127.0.0.1:18089'
shutil.copy2(envfile, stage/'env.before')
binary = Path('/www/wwwroot/pastrack/pastrack-web')
shutil.copy2(binary, stage/'web.before')
frontend = Path('/www/wwwroot/pastrack/front')
assert not frontend.exists(), 'initial rollout expects a new frontend directory'
nextfront = frontend.with_name('front.auth-next')
if nextfront.exists(): shutil.rmtree(nextfront)
nextfront.mkdir(exist_ok=False)
with tarfile.open(stage/'frontend.tar.gz') as archive:
    archive.extractall(nextfront, filter='data')
for path in [nextfront, *nextfront.rglob('*')]:
    shutil.chown(path, user='www', group='www'); path.chmod(0o755 if path.is_dir() else 0o644)
nextbinary = binary.with_name('pastrack-web.auth-next')
with gzip.open(stage/'web.gz','rb') as source, nextbinary.open('wb') as target:
    shutil.copyfileobj(source,target)
nextbinary.chmod(0o750);shutil.chown(nextbinary,user='www',group='www')
password = public.M('config').where('id=?',(1,)).getField('mysql_root')
connection = pymysql.connect(host='localhost',user='root',password=password,database=values['DB_NAME'],autocommit=True)
try:
    with connection.cursor() as cursor:
        cursor.execute('CREATE TABLE IF NOT EXISTS schema_migration_steps (version VARCHAR(128) NOT NULL,step INT NOT NULL,checksum CHAR(64) NOT NULL,applied_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),PRIMARY KEY(version,step))')
        clean = '\n'.join(line for line in (stage/'017_auth_password_recovery.sql').read_text().splitlines() if not line.strip().startswith('--'))
        for step, statement in enumerate(clean.split(';')):
            statement=statement.strip()
            if not statement: continue
            checksum=hashlib.sha256(statement.encode()).hexdigest()
            cursor.execute('SELECT checksum FROM schema_migration_steps WHERE version=%s AND step=%s',('migrations/017_auth_password_recovery.sql',step));prior=cursor.fetchone()
            if prior: assert prior[0]==checksum
            else:
                cursor.execute(statement)
                cursor.execute('INSERT INTO schema_migration_steps(version,step,checksum) VALUES(%s,%s,%s)',('migrations/017_auth_password_recovery.sql',step,checksum))
        for user in [values['DB_USER'], 'pastrack_local_view']:
            cursor.execute('SELECT Host FROM mysql.user WHERE User=%s',(user,))
            for (host,) in cursor.fetchall():
                for table in ['auth_password_resets','auth_smtp_settings']:
                    cursor.execute('GRANT SELECT,INSERT,UPDATE,DELETE ON '+values['DB_NAME']+'.'+table+' TO %s@%s',(user,host))
finally: connection.close()
updates={'AUTH_APP_URL':'https://toko.azlaundry.my.id/app','APP_FRONTEND_DIR':str(frontend),'AUTH_MAIL_KEY':values.get('AUTH_MAIL_KEY') or base64.b64encode(secrets.token_bytes(32)).decode()}
lines=[line for line in original_env.decode().splitlines() if not any(line.startswith(k+'=') or line.startswith('export '+k+'=') for k in updates)]
newenv='\n'.join(lines)+'\n'+''.join('export '+key+'='+shlex.quote(value)+'\n' for key,value in updates.items())
def write_env(content):
    temporary=envfile.with_suffix('.auth-next');temporary.write_bytes(content);temporary.chmod(0o640);shutil.chown(temporary,user='root',group='www');os.replace(temporary,envfile)
def probe():
    for attempt in range(60):
        try:
            with urllib.request.urlopen('http://127.0.0.1:18089/healthz',timeout=2) as response: assert response.status==200
            return
        except Exception:
            if attempt==59:raise
            time.sleep(.5)
with app.test_request_context('/'):
    manager=main();args=public.dict_obj();args.project_name='pastrack_web'
    manager.stop_project(args)
    try:
        write_env(newenv.encode());os.replace(nextfront,frontend);os.replace(nextbinary,binary);manager.start_project(args);probe()
        with urllib.request.urlopen('https://toko.azlaundry.my.id/app/',timeout=15) as response:
            html=response.read();assert b'/app/assets/' in html
        try: urllib.request.urlopen('https://toko.azlaundry.my.id/api/v1/auth/me',timeout=10)
        except urllib.error.HTTPError as error: assert error.code==401
        else: raise AssertionError('unauthenticated identity accepted')
        (stage/'success.json').write_text(json.dumps({'healthy':True,'frontend':'/app/','smtp':'awaiting administrator configuration'}))
        print('AUTH_SETTINGS_DEPLOYED_HEALTHY')
    except Exception:
        manager.stop_project(args);shutil.copy2(stage/'web.before',binary);shutil.chown(binary,user='www',group='www');write_env(original_env)
        if frontend.exists():shutil.rmtree(frontend)
        manager.start_project(args);probe();raise
