"""Atomically deploy the dashboard WebSocket endpoint, proxy, and Vue bundle."""

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
assert str(stage).startswith("/root/pastrack-dashboard-ws-")
assert not (stage / "success.json").exists()
binary = Path("/www/wwwroot/pastrack/pastrack-web")
frontend = Path("/www/wwwroot/pastrack/front")
nginx_config = Path("/www/server/panel/vhost/nginx/toko.azlaundry.my.id.conf")
backup_binary = stage / "web.before"
backup_frontend = stage / "front.before"
backup_nginx = stage / "nginx.before"
next_binary = binary.with_name("pastrack-web.dashboard-ws-next")
next_frontend = frontend.with_name("front.dashboard-ws-next")

shutil.copy2(binary, backup_binary)
shutil.copy2(nginx_config, backup_nginx)
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

config = nginx_config.read_text()
marker = "    location = /healthz { return 404; }\n"
websocket_location = """    location = /api/v1/ws {
        proxy_pass http://127.0.0.1:18089;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection \"upgrade\";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_connect_timeout 5s;
        proxy_read_timeout 70s;
        proxy_send_timeout 70s;
        proxy_buffering off;
    }
"""
if "location = /api/v1/ws" not in config:
    assert config.count(marker) == 1
    config = config.replace(marker, marker + websocket_location)


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
    # aaPanel can return before the wrapper's child has released the realtime
    # socket. Stop only the web binary process and remove its stale socket
    # before starting the replacement; the GPS receiver has a separate process.
    os.system("pkill -TERM -f '^/www/wwwroot/pastrack/pastrack-web$' >/dev/null 2>&1 || true")
    time.sleep(1)
    realtime_socket = Path("/opt/pastrack/v2-live/run/live.sock")
    if realtime_socket.exists():
        try:
            realtime_socket.unlink()
        except OSError:
            pass
    try:
        os.replace(next_binary, binary)
        old_frontend = frontend.with_name("front.dashboard-ws-old")
        if old_frontend.exists():
            shutil.rmtree(old_frontend)
        os.replace(frontend, old_frontend)
        os.replace(next_frontend, frontend)
        shutil.rmtree(old_frontend)
        nginx_config.write_text(config)
        assert os.system("/www/server/nginx/sbin/nginx -t -c /www/server/nginx/conf/nginx.conf") == 0
        assert os.system("/www/server/nginx/sbin/nginx -s reload -c /www/server/nginx/conf/nginx.conf") == 0
        manager.start_project(args)
        probe()
        with urllib.request.urlopen("https://toko.azlaundry.my.id/app/", timeout=15) as response:
            assert b"/assets/" in response.read()
        request = urllib.request.Request("https://toko.azlaundry.my.id/api/v1/ws", headers={"Origin": "https://toko.azlaundry.my.id"})
        try:
            urllib.request.urlopen(request, timeout=10)
        except urllib.error.HTTPError as error:
            assert error.code == 401
        else:
            raise AssertionError("anonymous WebSocket endpoint accepted")
        result = {"healthy": True, "endpoint": "/api/v1/ws", "mode": "event-driven-websocket"}
        (stage / "success.json").write_text(json.dumps(result))
        print("DASHBOARD_WEBSOCKET_DEPLOYED_HEALTHY")
    except Exception:
        manager.stop_project(args)
        shutil.copy2(backup_binary, binary)
        shutil.chown(binary, user="www", group="www")
        if frontend.exists():
            shutil.rmtree(frontend)
        shutil.copytree(backup_frontend, frontend)
        shutil.copy2(backup_nginx, nginx_config)
        assert os.system("/www/server/nginx/sbin/nginx -t -c /www/server/nginx/conf/nginx.conf") == 0
        os.system("/www/server/nginx/sbin/nginx -s reload -c /www/server/nginx/conf/nginx.conf")
        manager.start_project(args)
        probe()
        raise
