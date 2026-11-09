"""Atomically deploy the dashboard snapshot endpoint and Vue bundle."""

import gzip
import json
import os
import shutil
import sys
import tarfile
import time
import urllib.error
import urllib.request
from pathlib import Path

sys.path[:0] = ["/www/server/panel", "/www/server/panel/class", "/www/server/panel/class_v2"]
os.chdir("/www/server/panel")
import public
from BTPanel import app
from projectModelV2.goModel import main

stage = Path(sys.argv[1]).resolve()
assert str(stage).startswith("/root/pastrack-dashboard-")
assert not (stage / "success.json").exists()
binary = Path("/www/wwwroot/pastrack/pastrack-web")
frontend = Path("/www/wwwroot/pastrack/front")
backup_binary = stage / "web.before"
backup_frontend = stage / "front.before"
next_binary = binary.with_name("pastrack-web.dashboard-next")
next_frontend = frontend.with_name("front.dashboard-next")

shutil.copy2(binary, backup_binary)
if backup_frontend.exists():
    shutil.rmtree(backup_frontend)
shutil.copytree(frontend, backup_frontend)
if next_frontend.exists():
    shutil.rmtree(next_frontend)
next_frontend.mkdir()
with tarfile.open(stage / "frontend.tar.gz") as archive:
    archive.extractall(next_frontend, filter="data")
with gzip.open(stage / "web.gz", "rb") as source, next_binary.open("wb") as target:
    shutil.copyfileobj(source, target)
next_binary.chmod(0o750)
shutil.chown(next_binary, user="www", group="www")
for path in [next_frontend, *next_frontend.rglob("*")]:
    shutil.chown(path, user="www", group="www")
    path.chmod(0o755 if path.is_dir() else 0o644)


def probe():
    for attempt in range(60):
        try:
            with urllib.request.urlopen("http://127.0.0.1:18089/healthz", timeout=2) as response:
                assert response.status == 200
            return
        except Exception:
            if attempt == 59:
                raise
            time.sleep(0.5)


with app.test_request_context("/"):
    manager = main()
    args = public.dict_obj()
    args.project_name = "pastrack_web"
    manager.stop_project(args)
    try:
        os.replace(next_binary, binary)
        old_frontend = frontend.with_name("front.dashboard-old")
        if old_frontend.exists():
            shutil.rmtree(old_frontend)
        os.replace(frontend, old_frontend)
        os.replace(next_frontend, frontend)
        shutil.rmtree(old_frontend)
        manager.start_project(args)
        probe()
        with urllib.request.urlopen("https://toko.azlaundry.my.id/app/", timeout=15) as response:
            html = response.read()
            assert b"/app/assets/" in html
        request = urllib.request.Request("https://toko.azlaundry.my.id/api/v1/dashboard/snapshot")
        try:
            urllib.request.urlopen(request, timeout=10)
        except urllib.error.HTTPError as error:
            assert error.code == 401
        else:
            raise AssertionError("anonymous dashboard snapshot accepted")
        (stage / "success.json").write_text(
            json.dumps({"healthy": True, "endpoint": "/api/v1/dashboard/snapshot", "mode": "snapshot"})
        )
        print("DASHBOARD_SNAPSHOT_DEPLOYED_HEALTHY")
    except Exception:
        manager.stop_project(args)
        shutil.copy2(backup_binary, binary)
        shutil.chown(binary, user="www", group="www")
        if frontend.exists():
            shutil.rmtree(frontend)
        shutil.copytree(backup_frontend, frontend)
        for path in [frontend, *frontend.rglob("*")]:
            shutil.chown(path, user="www", group="www")
        manager.start_project(args)
        probe()
        raise
