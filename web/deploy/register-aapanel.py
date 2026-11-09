"""Run with aaPanel Python from /www/server/panel; registers only the web app."""
import os
import sys
sys.path[:0] = [os.getcwd(), 'class', 'class_v2']
from BTPanel import app
import public
from projectModelV2.goModel import main
with app.test_request_context('/'):
    manager = main()
    existing = manager.get_project_find('pastrack_web')
    args = public.dict_obj()
    values = dict(project_name='pastrack_web',project_exe='/www/wwwroot/pastrack/pastrack-web',project_ps='Pastrack Beego fleet monitoring dashboard',bind_extranet=0,domains=[],is_power_on=1,run_user='www',project_cmd='/www/wwwroot/pastrack/deploy/run-aapanel.sh',port=18089,env_file='',env_list=[])
    for k,v in values.items():setattr(args,k,v)
    if existing:
        print('EXISTING_PROJECT',existing['id'])
    else:
        print('REGISTER',manager.create_project(args))
    assert manager.get_project_run_state(project_name='pastrack_web'), 'Web project did not start'
    p=manager.get_project_find('pastrack_web')
    print('WEB_PROJECT_OK',p['id'],p['name'],p['project_config']['is_power_on'])
