import os,sys,shutil,time,json,urllib.request
from pathlib import Path
sys.path[:0]=['/www/server/panel','/www/server/panel/class','/www/server/panel/class_v2'];os.chdir('/www/server/panel')
import public
from BTPanel import app
from projectModelV2.goModel import main
binary=Path('/opt/pastrack/pastrack');backup=Path('/root/pastrack-live-cutover-20261103-r2/receiver-before-direct-remote');incoming=Path('/root/pastrack-live-cutover-20261103-r2/receiver-direct-next');nextfile=binary.with_name('pastrack.next')
shutil.copy2(binary,backup);shutil.copy2(incoming,nextfile);nextfile.chmod(0o750);shutil.chown(nextfile,user='www',group='www')
with app.test_request_context('/'):
 manager=main();args=public.dict_obj();args.project_name='pastrack';manager.stop_project(args)
 try:
  os.replace(nextfile,binary);manager.start_project(args)
  for attempt in range(40):
   try:
    with urllib.request.urlopen('http://127.0.0.1:18087/healthz',timeout=2) as response:health=json.load(response);assert health['status']=='ok'
    break
   except Exception:
    if attempt==39:raise
    time.sleep(.25)
  print(json.dumps({'receiver_updated':True,'health':health}))
 except Exception:
  manager.stop_project(args);shutil.copy2(backup,binary);manager.start_project(args);raise
