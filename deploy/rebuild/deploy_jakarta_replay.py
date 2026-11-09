"""Back up the production schema, then import the authorized replay."""
import gzip
import json
import os
import shutil
import subprocess
import sys
from pathlib import Path

sys.path[:0] = ['/www/server/panel', '/www/server/panel/class', '/www/server/panel/class_v2']
os.chdir('/www/server/panel')
import public
import pymysql
from import_jakarta_replay import import_replay

root = Path(sys.argv[1])
assert root.is_absolute() and str(root).startswith('/root/pastrack-jakarta-replay-')
schema = 'pastrack_v2_20261103_r2'
password = public.M('config').where('id=?', (1,)).getField('mysql_root')
backup = root/'before-replay.sql.gz'
with gzip.open(backup, 'xb') as output:
    backup.chmod(0o600)
    process = subprocess.Popen(['mysqldump', '-u', 'root', '--single-transaction',
        '--skip-lock-tables', '--hex-blob', schema], stdout=subprocess.PIPE,
        env={**os.environ, 'MYSQL_PWD': password})
    shutil.copyfileobj(process.stdout, output)
    assert process.wait() == 0
connection = pymysql.connect(host='localhost', user='root', password=password,
    database=schema, charset='utf8mb4', autocommit=True, cursorclass=pymysql.cursors.DictCursor)
try:
    result = import_replay(connection, json.loads((root/'replay.json').read_text()))
    (root/'result.json').write_text(json.dumps(result, indent=2))
    print(json.dumps(result))
finally:
    connection.close()
