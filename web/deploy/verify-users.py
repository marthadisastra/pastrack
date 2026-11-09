"""Authentication and RBAC regression checks against disposable QA only."""
import copy,hashlib,http.cookiejar,json,os,urllib.request,urllib.error,urllib.parse,zipfile,io
from pathlib import Path
assert os.environ['DB_NAME']=='pastrack_users_qa'
base='http://127.0.0.1:18090'
initial=Path('/etc/pastrack/web-access.txt').read_text().split('Password: ',1)[1].strip()
class Client:
 counter=0
 def __init__(self,name,password):
  Client.counter+=1;self.ip='198.51.100.'+str(Client.counter);self.jar=http.cookiejar.CookieJar();self.opener=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(self.jar));self.name=name
  response=self.opener.open(urllib.request.Request(base+'/login',data=urllib.parse.urlencode(dict(username=name,password=password)).encode(),headers={'X-Real-IP':self.ip}))
  self.logged_in=response.geturl().rstrip('/')==base
  self.token=''
  if self.logged_in:self.token=self.call('/api/me')['csrf_token']
 def call(self,path,method='GET',body=None,status=200,csrf=None):
  headers={'X-Real-IP':self.ip}
  if body is not None:headers.update({'Content-Type':'application/json','Origin':base,'X-CSRF-Token':self.token if csrf is None else csrf})
  try:r=self.opener.open(urllib.request.Request(base+path,method=method,data=json.dumps(body).encode() if body is not None else None,headers=headers));payload=r.read();code=r.status
  except urllib.error.HTTPError as e:payload=e.read();code=e.code
  assert code==status,(self.name,path,code,payload[:600])
  return json.loads(payload)
 def raw(self,path):return self.opener.open(base+path)
admin=Client('admin',initial);assert admin.logged_in
me=admin.call('/api/me')['data'];assert me['role']=='admin'
assert 'password_hash' not in json.dumps(me)
admin.call('/api/users/1','DELETE',{'row_version':me['row_version']},409)
profile=dict(full_name='QA Administrator',email='admin-qa@example.invalid',phone='+62 812 0000 0000',job_title='QA Lead',department='Quality',row_version=me['row_version'])
admin.call('/api/me','PUT',profile);admin.call('/api/me','PUT',profile,409)
assert admin.call('/api/me')['data']['full_name']=='QA Administrator'
print('PASS legacy admin login, profile update, stale version and last-admin protection')
def create(name,role,email):
 payload=dict(username=name,full_name='QA '+role,email=email,phone='',job_title='',department='QA',role=role,is_active=True,password='QA Initial passphrase 2026',confirm_password='QA Initial passphrase 2026')
 return admin.call('/api/users','POST',payload)['data'],payload
viewer,vbody=create('qa.viewer','viewer','viewer@example.invalid');operator,obody=create('qa.operator','operator','operator@example.invalid')
admin.call('/api/users','POST',vbody,409)
admin.call('/api/users','POST',vbody|{'username':'another.viewer'},409)
admin.call('/api/users','POST',vbody|{'username':'badrole','email':'bad@example.invalid','role':'root'},422)
v=Client('qa.viewer',vbody['password']);v2=Client('qa.viewer',vbody['password']);o=Client('qa.operator',obody['password']);assert v.logged_in and v2.logged_in and o.logged_in
v.call('/api/users',status=403);v.call('/api/audit',status=403);v.call('/api/devices')
vehicle=dict(plate_number='QA AUTH 01',vehicle_name='Auth QA',vehicle_type='truck',fleet_code='QA',is_active=True)
v.call('/api/master/vehicles','POST',vehicle,403)
o.call('/api/users',status=403)
created=o.call('/api/master/vehicles','POST',vehicle)['data'];assert created['plate_number']=='QA AUTH 01'
vm=v.call('/api/me')['data']
v.call('/api/me','PUT',dict(full_name='QA Viewer',email='viewer@example.invalid',phone='',job_title='',department='QA',row_version=vm['row_version'],role='admin'),422)
v.call('/api/me/password','POST',dict(current_password='wrong',new_password='QA New passphrase 2026',confirm_password='QA New passphrase 2026'),422)
v.call('/api/me/password','POST',dict(current_password=vbody['password'],new_password='short',confirm_password='short'),422)
v.call('/api/me/password','POST',dict(current_password=vbody['password'],new_password='QA New passphrase 2026',confirm_password='mismatch'),422)
v.call('/api/me/password','POST',dict(current_password=vbody['password'],new_password='QA New passphrase 2026',confirm_password='QA New passphrase 2026'))
v.call('/api/me',status=401);v2.call('/api/me',status=401)
assert not Client('qa.viewer',vbody['password']).logged_in
v=Client('qa.viewer','QA New passphrase 2026');assert v.logged_in
print('PASS role enforcement, own-password verification, password policy, old-password rejection and all-session revocation')
# Report authors must be per-request, not the old single global admin.
r=v.raw('/api/reports?type=devices&format=xlsx&start=2026-10-01&end=2026-10-31');data=r.read()
with zipfile.ZipFile(io.BytesIO(data)) as z:assert 'qa.viewer' in z.read('docProps/core.xml').decode()
# Reset a different user's password; their existing session must immediately fail.
users=admin.call('/api/users')['data'];operator=next(x for x in users if x['username']=='qa.operator')
admin.call('/api/users/'+str(operator['id'])+'/password','POST',dict(password='QA Reset passphrase 2026',confirm_password='QA Reset passphrase 2026',row_version=operator['row_version']))
o.call('/api/me',status=401);assert not Client('qa.operator',obody['password']).logged_in
assert Client('qa.operator','QA Reset passphrase 2026').logged_in
users=admin.call('/api/users')['data'];viewer=next(x for x in users if x['username']=='qa.viewer')
update={k:viewer.get(k) or '' for k in ['username','full_name','email','phone','job_title','department','role']};update.update(is_active=False,row_version=viewer['row_version'])
updated=admin.call('/api/users/'+str(viewer['id']),'PUT',update)['data'];v.call('/api/me',status=401)
assert not Client('qa.viewer','QA New passphrase 2026').logged_in
archived=admin.call('/api/users/'+str(viewer['id']),'DELETE',{'row_version':updated['row_version']})['data']
assert any(x['id']==viewer['id'] for x in admin.call('/api/users?archived=1')['data'])
restored=admin.call('/api/users/'+str(viewer['id'])+'/restore','POST',{'row_version':archived['row_version']})['data'];assert restored['is_active']==0
# A second admin may exist, but the current admin still cannot demote/archive self.
backup,_=create('qa.backup','admin','backup@example.invalid')
me=admin.call('/api/me')['data'];admin.call('/api/users/'+str(me['id']),'DELETE',{'row_version':me['row_version']},409)
logs=admin.call('/api/audit')['data'];encoded=json.dumps(logs)
for forbidden in ['password_hash','current_password','QA Initial passphrase','QA New passphrase','QA Reset passphrase','$2a$']:
 assert forbidden not in encoded,'credential leakage in audit'
assert any(x['actor']=='qa.operator' and x['entity']=='vehicles' for x in logs)
assert any(x['actor']=='qa.viewer' and x['action']=='export' for x in logs)
assert any(x['actor']=='qa.viewer' and x['action']=='password_change' for x in logs)
print('PASS reset-password revocation, disable/archive/restore, per-user audit actors and no password/hash leakage')
# Verify session bearer tokens are hashed at rest and bootstrap migrated to bcrypt.
import mysql.connector
c=mysql.connector.connect(host=os.environ['DB_HOST'],port=int(os.environ['DB_PORT']),user=os.environ['DB_USER'],password=os.environ['DB_PASSWORD'],database=os.environ['DB_NAME']);q=c.cursor()
q.execute("SELECT password_algo FROM web_users WHERE username='admin'");assert q.fetchone()[0]=='bcrypt'
session=next(x.value for x in admin.jar if x.name=='pastrack_session')
q.execute('SELECT token_hash FROM web_sessions WHERE token_hash=%s',(hashlib.sha256(session.encode()).hexdigest(),));assert q.fetchone()[0]!=session
print('PASS bcrypt migration and hashed server-side sessions; no production data modified')
