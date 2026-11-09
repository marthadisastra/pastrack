"""Integration checks against disposable QA only. Never mutates production."""
import hashlib,http.cookiejar,json,os,urllib.request,urllib.error,urllib.parse,zipfile,io,xml.etree.ElementTree as ET
from pathlib import Path
base='http://127.0.0.1:18090'
assert os.environ['DB_NAME']=='pastrack_web_qa'
jar=http.cookiejar.CookieJar();client=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
password=Path('/etc/pastrack/web-access.txt').read_text().split('Password: ',1)[1].strip()
client.open(urllib.request.Request(base+'/login',data=urllib.parse.urlencode(dict(username='admin',password=password)).encode()))
def call(path,method='GET',body=None,expected=200,csrf=None,origin=base):
 headers={}
 if body is not None:headers={'Content-Type':'application/json','X-CSRF-Token':token if csrf is None else csrf,'Origin':origin}
 try:
  response=client.open(urllib.request.Request(base+path,data=json.dumps(body).encode() if body is not None else None,method=method,headers=headers))
  data=response.read();code=response.status
 except urllib.error.HTTPError as e:data=e.read();code=e.code
 assert code==expected,(path,method,code,data[:800])
 return json.loads(data)
token=call('/api/master/vehicles')['csrf_token']
vehicle=dict(plate_number='QA 9001 TEST',vehicle_name='QA Enterprise Vehicle',vehicle_type='truck',fleet_code='QA-FLEET',is_active=True)
assert call('/api/master/vehicles','POST',vehicle,403,csrf='bad')
assert call('/api/master/vehicles','POST',vehicle,403,origin='https://attacker.invalid')
v=call('/api/master/vehicles','POST',vehicle)['data'];vid=int(v['id'])
call('/api/master/vehicles','POST',vehicle,409)
vehicle.update(vehicle_name='QA Edited',row_version=int(v['row_version']))
v=call('/api/master/vehicles/'+str(vid),'PUT',vehicle)['data']
call('/api/master/vehicles/'+str(vid),'PUT',vehicle,409)
vehicle['row_version']=int(v['row_version']);vehicle['vehicle_name']=''
call('/api/master/vehicles/'+str(vid),'PUT',vehicle,422)
device=dict(imei='000000000009901',model='GV30CEU',protocol_name='queclink',protocol_version='80201E0100',firmware_version='QA-FW',vehicle_id=vid,is_active=True,metadata={'source':'integration QA'})
bad=device|{'imei':'bad'};call('/api/master/devices','POST',bad,422)
call('/api/master/devices','POST',device|{'vehicle_id':2147483646},422)
d=call('/api/master/devices','POST',device)['data'];did=int(d['id'])
call('/api/master/devices','POST',device,409)
call('/api/master/vehicles/'+str(vid),'DELETE',{'row_version':int(v['row_version'])},409)
device.update(model='GV30CEU-QA',row_version=int(d['row_version']))
d=call('/api/master/devices/'+str(did),'PUT',device)['data']
originals=call('/api/master/devices')['data'];original=next((x for x in originals if x['last_seen_at']),None)
if original:
 payload={k:original.get(k) or '' for k in ['imei','model','protocol_name','protocol_version','firmware_version']}
 payload.update(imei='000000000009902',vehicle_id=original.get('vehicle_id'),metadata={},is_active=True,row_version=int(original['row_version']))
 call('/api/master/devices/'+str(original['id']),'PUT',payload,409)
d=call('/api/master/devices/'+str(did),'DELETE',{'row_version':int(d['row_version'])})['data']
v=call('/api/master/vehicles/'+str(vid),'DELETE',{'row_version':int(v['row_version'])})['data']
assert any(x['id']==vid for x in call('/api/master/vehicles?archived=1')['data'])
assert all(x['id']!=vid for x in call('/api/master/vehicles')['data'])
call('/api/master/devices/'+str(did)+'/restore','POST',{'row_version':int(d['row_version'])},422)
v=call('/api/master/vehicles/'+str(vid)+'/restore','POST',{'row_version':int(v['row_version'])})['data'];assert v['is_active']==0
restored=call('/api/master/devices/'+str(did)+'/restore','POST',{'row_version':int(d['row_version'])})['data'];assert restored['is_active']==0
print('PASS CRUD, uniqueness, validation, CSRF/origin, optimistic locking, archive relations, restore, IMEI/history guard')
output=Path('/root/pastrack-report-qa');output.mkdir(exist_ok=True,mode=0o700)
for kind in ['positions','events','messages','vehicles','devices']:
 preview=call('/api/reports?type='+kind+'&start=2026-10-01&end=2026-10-31')
 for fmt in ['xlsx','pdf']:
  url=base+'/api/reports?'+urllib.parse.urlencode(dict(type=kind,start='2026-10-01',end='2026-10-31',format=fmt))
  response=client.open(url);data=response.read();assert response.headers['X-Content-SHA256']==hashlib.sha256(data).hexdigest()
  assert response.headers['X-Report-ID']
  if fmt=='pdf':assert data.startswith(b'%PDF')
  else:
   with zipfile.ZipFile(io.BytesIO(data)) as z:
    sheet=ET.fromstring(z.read('xl/worksheets/sheet2.xml'));ns={'s':'http://schemas.openxmlformats.org/spreadsheetml/2006/main'}
    rows=sheet.findall('s:sheetData/s:row',ns);detail_rows=[r for r in rows if int(r.attrib['r'])>=5]
    assert len(detail_rows)==preview['total'],(kind,len(detail_rows),preview['total'])
  (output/(kind+'.'+fmt)).write_bytes(data)
 print('PASS report',kind,'rows',preview['total'],'xlsx/pdf and SHA256')
logs=call('/api/audit')['data'];assert sum(r['action']=='export' for r in logs)>=10
assert any(r['action']=='archive' for r in logs) and any(r['action']=='restore' for r in logs)
for bad in ['type=bad','type=positions&start=2026-11-01&end=2026-10-01','type=messages&status=bad']:
 call('/api/reports?'+bad,expected=422)
print('PASS report/date validation, audit export checksums and before/after records')
