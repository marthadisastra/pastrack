"""Atomic web-only asset refresh after successful v2 cutover; receiver is unchanged."""
import os,sys,gzip,shutil,time,urllib.request
from pathlib import Path
sys.path[:0]=['/www/server/panel','/www/server/panel/class','/www/server/panel/class_v2'];os.chdir('/www/server/panel')
import public
from BTPanel import app
from projectModelV2.goModel import main
root=Path('/root/pastrack-live-cutover-20261103-r2');assert (root/'cutover-success.json').exists()
binary=Path('/www/wwwroot/pastrack/pastrack-web');next_file=binary.with_name(binary.name+'.next')
with gzip.open(root/'web-current.gz','rb') as source,next_file.open('wb') as out:shutil.copyfileobj(source,out)
next_file.chmod(0o750);shutil.chown(next_file,user='www',group='www');shutil.copy2(binary,root/'web-before-login-copy')
with app.test_request_context('/'):
 manager=main();args=public.dict_obj();args.project_name='pastrack_web';manager.stop_project(args);os.replace(next_file,binary)
 try:
  manager.start_project(args)
  for i in range(40):
   try:
    with urllib.request.urlopen('http://127.0.0.1:18089/healthz',timeout=2) as response:assert response.status==200
    break
   except Exception:
    if i==39:raise
    time.sleep(.25)
  with urllib.request.urlopen('https://toko.azlaundry.my.id/',timeout=10) as response:html=response.read();assert b'Local development' not in html and b'fuel telemetry' in html
  print('LIVE_WEB_LOGIN_COPY_UPDATED_HEALTHY')
 except Exception:
  manager.stop_project(args);shutil.copy2(root/'web-before-login-copy',binary);shutil.chown(binary,user='www',group='www');manager.start_project(args);raise
