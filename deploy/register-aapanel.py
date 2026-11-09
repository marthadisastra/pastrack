"""Run with aaPanel Python from /www/server/panel. No credentials are embedded."""
import argparse
import os
import signal
import sys
import time
import urllib.request

sys.path.insert(0, os.getcwd())
sys.path.insert(0, 'class')
sys.path.insert(0, 'class_v2')
from BTPanel import app
import public
from projectModelV2.goModel import main

args = argparse.ArgumentParser()
args.add_argument('--legacy-pid', type=int, required=True)
options = args.parse_args()

with app.test_request_context('/'):
    manager = main()
    assert not manager.get_project_find('pastrack'), 'Project exists; use aaPanel restart instead'
    # Verify the legacy PID before stopping only this process.
    pid = options.legacy_pid
    command = open('/proc/%d/cmdline' % pid, 'rb').read()
    assert b'pastrack.py' in command, 'Legacy PID changed; inspect before cutover'
    os.kill(pid, signal.SIGTERM)
    for _ in range(50):
        if not os.path.exists('/proc/%d' % pid):
            break
        time.sleep(.1)
    get = public.dict_obj()
    values = dict(project_name='pastrack', project_exe='/opt/pastrack/pastrack', project_ps='Queclink GV30CEU GPS receiver; CFS300B BLE raw capture', bind_extranet=0, domains=[], is_power_on=1, run_user='www', project_cmd='/opt/pastrack/deploy/run-aapanel.sh', port=5001, env_file='', env_list=[])
    for k, v in values.items():
        setattr(get, k, v)
    try:
        result = manager.create_project(get)
        print('REGISTER_RESULT', result)
        assert manager.get_project_find('pastrack'), 'Go project missing from aaPanel'
        assert manager.get_project_run_state(project_name='pastrack'), 'Go project not running'
        print('HEALTH', urllib.request.urlopen('http://127.0.0.1:18087/healthz', timeout=5).read().decode())
        print('AAPANEL_GO_PROJECT_OK')
    except Exception:
        # A registered process may already own the TCP port; avoid a duplicate.
        if not manager.get_project_find('pastrack') or not manager.get_project_run_state(project_name='pastrack'):
            import subprocess
            subprocess.Popen(['/bin/bash', '-c', 'set -a; source /opt/pastrack/.env; set +a; cd /opt/queclink; exec /opt/queclink/.venv/bin/python -u pastrack.py'], stdout=open('/opt/queclink/rollback.log','ab'), stderr=subprocess.STDOUT, start_new_session=True)
            print('Legacy Python rollback launched')
        raise
