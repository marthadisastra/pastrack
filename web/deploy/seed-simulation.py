"""One-shot, tagged simulation fixture for Pastrack. Run on the server as DBA.

No tracker commands, external routing calls, or real-device updates. Existing
SIM-DEMO-V1 fixtures cause a safe no-op. Writes run in one SQL transaction;
a private SQL backup and a fixture manifest are saved outside the web root.
"""
import argparse, datetime as dt, json, math, os, shlex, sqlite3, subprocess
import socket
from pathlib import Path

TAG = 'SIM-DEMO-V1'
UTC = dt.timezone.utc
WIB = dt.timezone(dt.timedelta(hours=7))

def env_file(path):
    result = {}
    for line in Path(path).read_text().splitlines():
        parts = shlex.split(line.removeprefix('export '))
        if len(parts) == 1 and '=' in parts[0]:
            k, v = parts[0].split('=', 1); result[k] = v
    return result

def text(value):
    return "CONVERT(0x" + str(value).encode().hex() + " USING utf8mb4)" if str(value) else "''"

def stamp(at):
    return "'" + at.astimezone(UTC).strftime('%Y-%m-%d %H:%M:%S') + "'"

def distance(a, b):
    lat1, lon1, lat2, lon2 = map(math.radians, (*a, *b))
    h=math.sin((lat2-lat1)/2)**2+math.cos(lat1)*math.cos(lat2)*math.sin((lon2-lon1)/2)**2
    return 6371*2*math.asin(min(1,math.sqrt(h)))

def route(points, minutes):
    lengths=[distance(a,b) for a,b in zip(points,points[1:])]; total=sum(lengths)
    result=[]
    for i in range(minutes+1):
        target=total*i/minutes; part=0
        while part<len(lengths)-1 and target>lengths[part]: target-=lengths[part];part+=1
        ratio=target/max(lengths[part],1e-9);a,b=points[part:part+2]
        result.append((a[0]+(b[0]-a[0])*ratio,a[1]+(b[1]-a[1])*ratio))
    return result

def scenarios():
    return [
      ('SIM 1001','SIMULASI Distribusi Kota','van',[-6.1754,106.8272],[[-6.1754,106.8272],[-6.185,106.8272],[-6.185,106.845],[-6.210,106.845],[-6.210,106.875],[-6.240,106.875]],70,15,3),
      ('SIM 1002','SIMULASI Angkutan Jarak Jauh','truck',[-6.245,106.990],[[-6.245,106.990],[-6.260,107.045],[-6.290,107.120],[-6.360,107.240],[-6.440,107.360],[-6.520,107.440]],55,10,2),
      ('SIM 1003','SIMULASI Kurir Banyak Berhenti','van',[-6.220,106.800],[[-6.220,106.800],[-6.220,106.820],[-6.235,106.820],[-6.235,106.800],[-6.220,106.800]],85,30,4),
      ('SIM 1004','SIMULASI Shift Malam','truck',[-6.150,106.880],[[-6.150,106.880],[-6.150,106.920],[-6.170,106.920],[-6.170,106.970],[-6.190,106.970]],60,20,2),
      ('SIM 1005','SIMULASI Parkir Depot','truck',[-6.230,106.890],None,0,0,0),
      ('SIM 1006','SIMULASI GPS Putus dan Duplikat','van',[-6.300,106.700],[[-6.300,106.700],[-6.300,106.750],[-6.330,106.750],[-6.330,106.800]],45,15,2),
    ]

def vehicle_points(index, day, scenario):
    plate,name,kind,home,path,minutes,stop,legs=scenario
    start=dt.datetime.combine(day,dt.time(6,0),WIB)
    if index==3: start=dt.datetime.combine(day-dt.timedelta(days=1),dt.time(23,0),WIB)
    result=[]
    if path is None:
        return [(start+dt.timedelta(minutes=i),*home,0.0,'parked') for i in range(241)]
    cursor=start; reverse=False
    for leg in range(legs):
        points=route(list(reversed(path)) if reverse else path,minutes)
        for i,(lat,lon) in enumerate(points):
            previous=points[max(0,i-1)];speed=distance(previous,(lat,lon))*60 if i else distance(points[0],points[1])*60
            result.append((cursor+dt.timedelta(minutes=i),lat,lon,speed,'moving'))
        cursor+=dt.timedelta(minutes=minutes)
        for i in range(1,stop+1):result.append((cursor+dt.timedelta(minutes=i),*points[-1],0.0,'stop'))
        cursor+=dt.timedelta(minutes=stop+1);reverse=not reverse
    # GPS outage, one duplicate, and an outlier make quality widgets testable.
    if index==5:
        result=[p for i,p in enumerate(result) if not 20<=i<=60]
        result.insert(10,result[10])
        p=result[15];result[15]=(p[0],-7.8,110.4,25.0,'gps_outlier')
    return result

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--apply',action='store_true');args=parser.parse_args()
    config=env_file('/etc/pastrack/web.env');schema=config['DB_NAME'];assert schema=='pastrack'
    private=Path('/root/pastrack-simulations')/TAG;private.mkdir(parents=True,exist_ok=True,mode=0o700)
    secret=sqlite3.connect('/www/server/panel/data/default.db').execute('SELECT mysql_root FROM config LIMIT 1').fetchone()[0]
    process_env=os.environ.copy();process_env['MYSQL_PWD']=secret
    mysql=['mysql','--batch','--raw','--skip-column-names','--default-character-set=utf8mb4','-u','root',schema]
    def query(sql):
        run=subprocess.run(mysql,input=sql,env=process_env,text=True,capture_output=True)
        if run.returncode:raise RuntimeError(run.stderr[:1000])
        return run.stdout.strip()
    existing=query("SELECT COUNT(*) FROM vehicles WHERE fleet_code="+text(TAG))
    if existing!='0':
        print('SAFE NO-OP: simulation batch already exists;',existing,'vehicles. Manifest:',private/'manifest.json');return
    now=dt.datetime.now(UTC).replace(second=0,microsecond=0);today=now.astimezone(WIB).date()
    sql=["SET time_zone='+00:00'; START TRANSACTION;"];manifest={'batch':TAG,'created_at':now.isoformat(),'source':'synthetic minute-resolution waypoints; not surveyed road geometry','vehicles':[],'days':7}
    for index,s in enumerate(scenarios()):
        plate,name,kind,home,path,minutes,stop,legs=s;imei='00000999000'+str(index+1).zfill(4)
        all_points=[]
        for offset in range(6,-1,-1):all_points+=vehicle_points(index,today-dt.timedelta(days=offset),s)
        all_points=[p for p in all_points if p[0].astimezone(UTC)<=now-dt.timedelta(minutes=10)]
        all_points.sort(key=lambda p:p[0]);assert all_points
        metadata=json.dumps({'simulation':True,'batch':TAG,'scenario':name,'coordinate_source':'synthetic waypoints','seeded_at':now.isoformat()})
        sql.append('INSERT INTO vehicles(plate_number,vehicle_name,vehicle_type,fleet_code,is_active) VALUES('+','.join(map(text,[plate,name,kind,TAG]))+',1); SET @vehicle=LAST_INSERT_ID();')
        sql.append('INSERT INTO gps_devices(imei,model,protocol_name,protocol_version,firmware_version,vehicle_id,last_seen_at,is_active,metadata) VALUES('+','.join(map(text,[imei,'SIMULATOR','simulation','SIM-V1','fixture-v1']))+',@vehicle,'+stamp(all_points[-1][0])+',1,'+text(metadata)+'); SET @device=LAST_INSERT_ID();')
        for i,(at,lat,lon,speed,state) in enumerate(all_points):
            payload=json.dumps({'simulation':True,'batch':TAG,'sequence':i,'scenario':name,'state':state,'latitude':round(lat,7),'longitude':round(lon,7),'speed_kmh':round(speed,2),'gps_time':at.astimezone(UTC).isoformat()},separators=(',',':'))
            received=at+dt.timedelta(seconds=2)
            sql.append('INSERT INTO device_raw_messages(device_id,imei,message_type,protocol_version,raw_payload,parsed_payload,received_at,parse_status,created_at) VALUES(@device,'+text(imei)+','+text('SIM_POSITION')+','+text('SIM-V1')+','+text('SIMULASI '+payload)+','+text(payload)+','+stamp(received)+",'success',"+stamp(received)+'); SET @raw=LAST_INSERT_ID();')
            sql.append('INSERT INTO gps_positions(device_id,raw_message_id,gps_time,device_send_time,latitude,longitude,speed_kmh,heading_deg,altitude_m,gps_accuracy,satellites,created_at) VALUES(@device,@raw,'+stamp(at)+','+stamp(received)+f',{lat:.7f},{lon:.7f},{speed:.2f},0,25,5,12,'+stamp(received)+');')
            if i==0 or state!=all_points[i-1][-1]:
                sql.append('INSERT INTO device_events(device_id,raw_message_id,message_type,event_code,event_value,event_time,created_at) VALUES(@device,@raw,'+text('SIM_POSITION')+','+text('SIM_'+state.upper())+','+text('SIMULASI: '+state)+','+stamp(at)+','+stamp(received)+');')
        summary={'plate':plate,'imei':imei,'scenario':name,'positions':len(all_points),'first_gps':all_points[0][0].isoformat(),'latest_gps':all_points[-1][0].isoformat()};manifest['vehicles'].append(summary)
        sql.append('INSERT INTO pastrack_audit_log(occurred_at,actor,action,entity,entity_id,after_data,request_id) VALUES(UTC_TIMESTAMP(3),'+text('simulation-seeder')+','+text('seed_simulation')+','+text('vehicles')+',@vehicle,'+text(json.dumps(summary))+','+text(TAG+'-'+str(index+1))+');')
    sql.append('COMMIT;')
    content='\n'.join(sql);manifest['total_positions']=sum(v['positions'] for v in manifest['vehicles'])
    print(json.dumps(manifest,indent=2))
    if not args.apply: print('DRY RUN: use --apply to insert.');return
    backup=private/'before.sql'
    if backup.exists():raise RuntimeError('Existing backup found; inspect before retrying')
    with backup.open('x') as f:
        backup.chmod(0o600)
        subprocess.run(['mysqldump','-u','root','--single-transaction',schema,'vehicles','gps_devices','gps_positions','device_raw_messages','device_events','pastrack_audit_log'],env=process_env,stdout=f,check=True)
    (private/'fixture.sql').write_text(content);(private/'fixture.sql').chmod(0o600)
    query(content)
    ids=json.loads(query("SELECT JSON_ARRAYAGG(JSON_OBJECT('vehicle_id',v.id,'device_id',d.id,'plate',v.plate_number,'imei',d.imei)) FROM vehicles v JOIN gps_devices d ON d.vehicle_id=v.id WHERE v.fleet_code="+text(TAG)))
    manifest['ids']=ids;(private/'manifest.json').write_text(json.dumps(manifest,indent=2));(private/'manifest.json').chmod(0o600)
    try:
        with socket.socket(socket.AF_UNIX,socket.SOCK_DGRAM) as signal:
            signal.sendto(b'positions',config.get('REALTIME_SOCKET_PATH','/tmp/pastrack/live.sock'))
    except OSError:
        print('History stored; realtime service unavailable for this invalidation.')
    print('APPLIED',manifest['total_positions'],'positions. Backup and manifest:',private)
if __name__=='__main__':main()
