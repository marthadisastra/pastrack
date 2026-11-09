"""Apply additive dashboard outbox migration and atomically deploy the receiver."""

import gzip
import hashlib
import json
import os
import shlex
import shutil
import subprocess
import sys
import time
import urllib.request
from pathlib import Path

sys.path[:0] = ["/www/server/panel", "/www/server/panel/class", "/www/server/panel/class_v2"]
os.chdir("/www/server/panel")
import public
import pymysql
from BTPanel import app
from projectModelV2.goModel import main

stage = Path(sys.argv[1]).resolve()
assert str(stage).startswith("/root/pastrack-dashboard-outbox-")
assert not (stage / "success.json").exists()
database = "pastrack_v2_20261103_r2"
receiver = Path("/opt/pastrack/pastrack")
backup = stage / "receiver.before"
candidate = receiver.with_name("pastrack.dashboard-outbox-next")
shutil.copy2(receiver, backup)
backup.chmod(0o600)
with gzip.open(stage / "receiver.gz", "rb") as source, candidate.open("wb") as target:
    shutil.copyfileobj(source, target)
candidate.chmod(0o750)
shutil.chown(candidate, user="root", group="root")

root_password = public.M("config").where("id=?", (1,)).getField("mysql_root")
environment = {**os.environ, "MYSQL_PWD": root_password}
with gzip.open(stage / "schema-before.sql.gz", "xb") as output:
    process = subprocess.Popen(["mysqldump", "-u", "root", "--no-data", "--routines", "--triggers", database], stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=environment)
    shutil.copyfileobj(process.stdout, output)
    error = process.stderr.read()
    assert process.wait() == 0, error.decode(errors="replace")
(stage / "schema-before.sql.gz").chmod(0o600)

values = {}
for line in Path("/etc/pastrack/v2-live.env").read_text().splitlines():
    parts = shlex.split(line)
    if parts and parts[0] == "export":
        parts = parts[1:]
    if parts and "=" in parts[0]:
        key, value = parts[0].split("=", 1)
        values[key] = value
assert values["DB_NAME"] == database

connection = pymysql.connect(host="localhost", user="root", password=root_password, database=database, autocommit=True)
cursor = connection.cursor()
cursor.execute("CREATE TABLE IF NOT EXISTS schema_migration_steps (version VARCHAR(128) NOT NULL,step INT NOT NULL,checksum CHAR(64) NOT NULL,applied_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),PRIMARY KEY(version,step))")
version = "migrations/018_dashboard_change_outbox.sql"
content = "\n".join(line for line in (stage / "018_dashboard_change_outbox.sql").read_text().splitlines() if not line.strip().startswith("--"))
for step, statement in enumerate(content.split(";")):
    statement = statement.strip()
    if not statement:
        continue
    checksum = hashlib.sha256(statement.encode()).hexdigest()
    cursor.execute("SELECT checksum FROM schema_migration_steps WHERE version=%s AND step=%s", (version, step))
    prior = cursor.fetchone()
    if prior:
        assert prior[0] == checksum
    else:
        cursor.execute(statement)
        cursor.execute("INSERT INTO schema_migration_steps(version,step,checksum) VALUES(%s,%s,%s)", (version, step, checksum))

for account, privileges in [(values["DB_USER"], "SELECT,INSERT,UPDATE,DELETE"), ("pastrack_local_view", "SELECT")]:
    cursor.execute("SELECT Host FROM mysql.user WHERE User=%s", (account,))
    for (host,) in cursor.fetchall():
        cursor.execute(f"GRANT {privileges} ON {database}.dashboard_change_outbox TO %s@%s", (account, host))


def health():
    for attempt in range(80):
        try:
            with urllib.request.urlopen("http://127.0.0.1:18087/healthz", timeout=2) as response:
                result = json.load(response)
                assert result["status"] == "ok"
                return result
        except Exception:
            if attempt == 79:
                raise
            time.sleep(0.25)


with app.test_request_context("/"):
    manager = main()
    args = public.dict_obj()
    args.project_name = "pastrack"
    manager.stop_project(args)
    try:
        os.replace(candidate, receiver)
        manager.start_project(args)
        result = health()
        cursor.execute("SELECT COUNT(*) FROM dashboard_change_outbox WHERE delivered_at IS NULL")
        pending = cursor.fetchone()[0]
        success = {"receiver_health": result, "migration": version, "pending_after_startup": pending, "receiver_sha256": hashlib.sha256(receiver.read_bytes()).hexdigest()}
        (stage / "success.json").write_text(json.dumps(success, indent=2))
        print("DASHBOARD_OUTBOX_DEPLOYED_HEALTHY")
    except Exception:
        manager.stop_project(args)
        temporary = receiver.with_name("pastrack.dashboard-outbox-rollback")
        shutil.copy2(backup, temporary)
        temporary.chmod(0o750)
        shutil.chown(temporary, user="root", group="root")
        os.replace(temporary, receiver)
        manager.start_project(args)
        health()
        raise
connection.close()
