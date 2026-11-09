"""Replace only tagged demonstration histories with deterministic daily speed scenarios.

Server preview: python3 deploy/simulate-october.py --start 2026-10-01
Apply: add --apply. No master vehicle names/plates/associations are rewritten.
Credentials stay server-side. Original rows are backed up before one transaction.
"""
import argparse
import datetime as dt
import gzip
import json
import math
import os
from pathlib import Path
import random
import shlex
import sqlite3
import subprocess
import socket

TAG = 'SIM-DEMO-V1'
VERSION = 2
UTC = dt.timezone.utc
WIB = dt.timezone(dt.timedelta(hours=7))
STEP = 120
PROFILES = [
    {'name':'Distribusi kota / stop-and-go','speed':32,'trips':4,'stop':14,'hour':7,
     'path':[[-6.1754,106.8272],[-6.185,106.8272],[-6.185,106.845],[-6.210,106.845],[-6.210,106.875],[-6.240,106.875]]},
    {'name':'Angkutan jauh / tol','speed':78,'trips':2,'stop':20,'hour':5,
     'path':[[-6.245,106.990],[-6.260,107.045],[-6.290,107.120],[-6.360,107.240],[-6.440,107.360],[-6.520,107.440]]},
    {'name':'Kurir / banyak kunjungan','speed':19,'trips':6,'stop':22,'hour':8,
     'path':[[-6.220,106.800],[-6.220,106.820],[-6.235,106.820],[-6.235,106.800],[-6.220,106.800]]},
    {'name':'Logistik shift malam','speed':48,'trips':4,'stop':16,'hour':23,
     'path':[[-6.150,106.880],[-6.150,106.920],[-6.170,106.920],[-6.170,106.970],[-6.190,106.970]]},
    {'name':'Depot / pemakaian rendah','speed':22,'trips':2,'stop':20,'hour':8,
     'path':[[-6.230,106.890],[-6.230,106.910],[-6.245,106.910],[-6.245,106.890],[-6.230,106.890]]},
    {'name':'Operasional / variasi rute dan jeda GPS','speed':37,'trips':3,'stop':17,'hour':6,
     'path':[[-6.300,106.700],[-6.300,106.750],[-6.330,106.750],[-6.330,106.800]]},
]


def distance(a,b):
    la,lo,lb,lob=map(math.radians,(*a,*b))
    h=math.sin((lb-la)/2)**2+math.cos(la)*math.cos(lb)*math.sin((lob-lo)/2)**2
    return 6371*2*math.asin(min(1,math.sqrt(h)))


def bearing(a,b):
    la,lo,lb,lob=map(math.radians,(*a,*b));delta=lob-lo
    return round(math.degrees(math.atan2(math.sin(delta)*math.cos(lb),math.cos(la)*math.sin(lb)-math.sin(la)*math.cos(lb)*math.cos(delta))))%360


def on_path(path,lengths,kilometers):
    for i,length in enumerate(lengths):
        if kilometers<=length or i==len(lengths)-1:
            ratio=min(1,max(0,kilometers/max(length,1e-12)))
            return tuple(path[i][j]+(path[i+1][j]-path[i][j])*ratio for j in (0,1))
        kilometers-=length
    return tuple(path[-1])


def daily_scenario(index,day):
    rng=random.Random(f'{TAG}-v{VERSION}-{index}-{day.isoformat()}')
    weekend=day.weekday()>=5
    condition=['normal','macet','sibuk','hujan simulasi','lancar'][(day.day+index*2)%5]
    factor={'normal':1.0,'macet':.62,'sibuk':.9,'hujan simulasi':.74,'lancar':1.12}[condition]
    trips=PROFILES[index]['trips']+(1 if condition=='sibuk' and index!=4 else 0)
    parked=False
    if weekend:trips=max(1,trips-2);condition='akhir pekan / operasi pendek';factor*=1.1
    if index==4:
        parked=day.day%7 not in (3,4)
        condition='parkir depot' if parked else 'uji jalan / servis ringan'
        trips=0 if parked else 2;factor=.8+rng.random()*.25
    elif index==1 and day.day in (7,14,21,28):
        parked=True;trips=0;condition='servis terjadwal'
    elif day.weekday()==6 and index in (0,2):
        parked=True;trips=0;condition='libur / parkir'
    if index==1 and day.day in (6,16,26):condition='perjalanan tol / kecepatan tinggi simulasi';factor=1.22
    # Week-by-week load changes make period comparisons meaningful.
    trips+=1 if index in (0,2,5) and day.day>=20 and not parked else 0
    start=dt.datetime.combine(day,dt.time(PROFILES[index]['hour']),WIB)+dt.timedelta(minutes=rng.choice([0,4,8,12,16]))
    if index==3:start-=dt.timedelta(days=1)
    return {'scenario_date':day.isoformat(),'profile':PROFILES[index]['name'],'condition':condition,
            'trips':trips,'speed_factor':round(factor,3),'start_wib':start.isoformat(),'parked':parked,
            'gps_outage':index==5 and day.day%5==0,'duplicate':index==5 and day.day%8==0}


def generate_day(index,day):
    config=daily_scenario(index,day);profile=PROFILES[index];rng=random.Random(f'speed-{index}-{day}')
    cursor=dt.datetime.fromisoformat(config['start_wib']);result=[];position=tuple(profile['path'][0])
    def emit(state,speed=0,heading=0):
        result.append({'at':cursor,'lat':round(position[0],7),'lon':round(position[1],7),
                       'speed':round(speed,2),'heading':heading,'state':state,
                       'scenario_date':day.isoformat(),'condition':config['condition']})
    def park(minutes,state):
        nonlocal cursor
        for _ in range(math.ceil(minutes*60/STEP)):
            cursor+=dt.timedelta(seconds=STEP);emit(state)
    emit('parked');park(10,'loading' if not config['parked'] else 'parked')
    for trip in range(config['trips']):
        path=list(reversed(profile['path'])) if trip%2 else profile['path']
        # Route starts exactly where the preceding leg ended.
        assert distance(position,path[0])<.001
        lengths=[distance(a,b) for a,b in zip(path,path[1:])];total=sum(lengths);progress=0;step=0
        pause_at=total*rng.uniform(.35,.65);paused=False
        while progress<total-1e-9:
            if not paused and progress>=pause_at:
                pause=4 if index==1 else rng.choice([6,8,10,12])
                park(pause,'delivery_stop' if index==2 else 'traffic_stop');paused=True
            phase=progress/total
            ramp=min(1,.35+step*.22,max(.22,(1-phase)*5))
            wave=.83+.2*math.sin(step*.83+day.day*.47)+rng.uniform(-.12,.12)
            speed=profile['speed']*config['speed_factor']*wave*ramp
            if index in (0,2) and cursor.weekday()<5 and (7<=cursor.hour<9 or 16<=cursor.hour<18):speed*=.62
            if index==2 and step%7 in (0,1):speed*=.6
            speed=max(4,min(112,speed))
            previous=position;progress=min(total,progress+speed*STEP/3600);position=on_path(path,lengths,progress)
            cursor+=dt.timedelta(seconds=STEP)
            # Store the speed implied by the actual rounded consecutive positions.
            actual=distance(previous,position)*3600/STEP
            emit('moving',actual,bearing(previous,position));step+=1
            if step>600:raise RuntimeError('route did not terminate')
        park(profile['stop']+rng.choice([0,4,8,12]),'delivery_stop' if index==2 else 'unloading')
    park(240 if config['parked'] else 40,'parked')
    if config['gps_outage']:
        # An absent interval, not invented movement or fake zero-speed records.
        begin=len(result)//3;result=[p for i,p in enumerate(result) if not begin<=i<begin+9]
    if config['duplicate'] and len(result)>20:result.insert(15,dict(result[15]))
    return config,result


def build_dataset(start,until):
    end=until.astimezone(WIB).date();begin=dt.datetime.combine(start,dt.time(),WIB)
    output=[]
    for index,profile in enumerate(PROFILES):
        points=[];plans=[];day=start
        while day<=end:
            config,generated=generate_day(index,day)
            kept=[p for p in generated if begin<=p['at']<=until-dt.timedelta(minutes=1)]
            config['positions']=len(kept);config['clipped_start']=bool(generated and generated[0]['at']<begin);config['complete']=bool(generated and not config['clipped_start'] and generated[-1]['at']<=until-dt.timedelta(minutes=1))
            moving=[p['speed'] for p in kept if p['speed']>3]
            config['max_speed']=max((p['speed'] for p in kept),default=0)
            config['average_moving_sample_speed']=round(sum(moving)/max(1,len(moving)),2)
            km_total=0;moving_seconds=0;stopped_seconds=0
            for a,b in zip(kept,kept[1:]):
                seconds=(b['at']-a['at']).total_seconds();km=distance((a['lat'],a['lon']),(b['lat'],b['lon']))
                if 0<seconds<=1800 and km/seconds*3600<=200:km_total+=km
                if 0<seconds<=300:
                    if a['speed']>3 and b['speed']>3:moving_seconds+=seconds
                    elif a['speed']<=3 and b['speed']<=3 and km/seconds*3600<=3:stopped_seconds+=seconds
            config.update(distance_km=round(km_total,2),moving_minutes=round(moving_seconds/60,2),stopped_minutes=round(stopped_seconds/60,2))
            points+=kept;plans.append(config);day+=dt.timedelta(days=1)
        points.sort(key=lambda p:p['at']);assert len(points)<=20000
        output.append({'index':index,'profile':profile['name'],'points':points,'days':plans})
    assert sum(len(x['points']) for x in output)<49000,'Dataset exceeds complete XLSX export budget'
    return output


def summarize(dataset):
    result=[]
    for car in dataset:
        points=car['points'];moving=[p['speed'] for p in points if p['speed']>3];kilometers=0
        for a,b in zip(points,points[1:]):
            seconds=(b['at']-a['at']).total_seconds();km=distance((a['lat'],a['lon']),(b['lat'],b['lon']))
            if 0<seconds<=1800 and km/seconds*3600<=200:kilometers+=km
        result.append({'index':car['index'],'profile':car['profile'],'positions':len(points),
            'first_gps':points[0]['at'].isoformat() if points else None,'latest_gps':points[-1]['at'].isoformat() if points else None,
            'distinct_speeds':len({p['speed'] for p in points}),'average_moving_sample_speed':round(sum(moving)/max(1,len(moving)),2),
            'max_speed':max((p['speed'] for p in points),default=0),'distance_km':round(kilometers,2),'days':car['days']})
    return result


def env_file(path):
    result={}
    for line in Path(path).read_text().splitlines():
        parts=shlex.split(line.removeprefix('export '))
        if len(parts)==1 and '=' in parts[0]:k,v=parts[0].split('=',1);result[k]=v
    return result


def text(value):
    return 'CONVERT(0x'+str(value).encode().hex()+' USING utf8mb4)' if str(value) else "''"


def stamp(at):return "'"+at.astimezone(UTC).strftime('%Y-%m-%d %H:%M:%S')+"'"


def main():
    parser=argparse.ArgumentParser();parser.add_argument('--start',default='2026-10-01');parser.add_argument('--apply',action='store_true');args=parser.parse_args()
    start=dt.date.fromisoformat(args.start);now=dt.datetime.now(UTC).replace(second=0,microsecond=0)
    assert 0<=(now.astimezone(WIB).date()-start).days<=30,'Range must fit Tracking maximum of 31 days'
    dataset=build_dataset(start,now);summary=summarize(dataset)
    config=env_file('/etc/pastrack/web.env');assert config['DB_NAME']=='pastrack'
    secret=sqlite3.connect('/www/server/panel/data/default.db').execute('SELECT mysql_root FROM config LIMIT 1').fetchone()[0]
    process_env=os.environ.copy();process_env['MYSQL_PWD']=secret
    mysql=['mysql','--batch','--raw','--skip-column-names','--default-character-set=utf8mb4','-u','root',config['DB_NAME']]
    def query(sql):
        p=subprocess.run(mysql,input="SET time_zone='+00:00'; SET SESSION group_concat_max_len=16777216;\n"+sql,env=process_env,text=True,capture_output=True)
        if p.returncode:raise RuntimeError(p.stderr[:1500])
        return p.stdout.strip()
    existing=json.loads(query("SELECT JSON_ARRAYAGG(JSON_OBJECT('id',d.id,'imei',d.imei,'vehicle_id',d.vehicle_id,'plate',v.plate_number,'vehicle_name',v.vehicle_name,'metadata',d.metadata,'protocol',d.protocol_name)) FROM gps_devices d JOIN vehicles v ON v.id=d.vehicle_id WHERE JSON_UNQUOTE(JSON_EXTRACT(d.metadata,'$.batch'))="+text(TAG)))
    assert isinstance(existing,list) and len(existing)==6,'Expected six original tagged simulator devices'
    by_imei={r['imei']:r for r in existing}
    for i,car in enumerate(summary):
        imei='00000999000'+str(i+1).zfill(4);device=by_imei[imei];metadata=device['metadata'] if isinstance(device['metadata'],dict) else json.loads(device['metadata'])
        assert metadata['simulation'] is True and metadata['batch']==TAG and device['protocol']=='simulation'
        car.update(device_id=device['id'],vehicle_id=device['vehicle_id'],plate=device['plate'],vehicle_name=device['vehicle_name'])
    ids=','.join(str(int(c['device_id'])) for c in summary)
    predicate="r.device_id IN ("+ids+") AND r.message_type='SIM_POSITION' AND JSON_EXTRACT(r.parsed_payload,'$.simulation')=true AND JSON_UNQUOTE(JSON_EXTRACT(r.parsed_payload,'$.batch'))="+text(TAG)
    unknown=int(query("SELECT COUNT(*) FROM gps_positions p LEFT JOIN device_raw_messages r ON r.id=p.raw_message_id WHERE p.device_id IN ("+ids+") AND NOT COALESCE(("+predicate+"),0)"))
    assert unknown==0,'Untagged positions exist on simulator devices; refusing to replace them'
    manifest={'batch':TAG,'simulation_version':VERSION,'start_wib':start.isoformat(),'generated_at':now.isoformat(),
              'interval_seconds':STEP,'coordinate_source':'synthetic waypoints; no road routing service','total_positions':sum(x['positions'] for x in summary),'vehicles':summary}
    print(json.dumps({**manifest,'vehicles':[{k:v for k,v in c.items() if k!='days'} for c in summary]},indent=2))
    if not args.apply:print('PREVIEW ONLY: no database rows changed.');return
    private=Path('/root/pastrack-simulations')/TAG/('v2-'+now.strftime('%Y%m%d-%H%M%S'));private.mkdir(mode=0o700,parents=True,exist_ok=False)
    with (private/'before.sql.gz').open('xb') as target:
        os.chmod(target.name,0o600)
        dump=subprocess.Popen(['mysqldump','-u','root','--single-transaction',config['DB_NAME'],'vehicles','gps_devices','gps_positions','device_raw_messages','device_events','pastrack_audit_log'],env=process_env,stdout=subprocess.PIPE)
        with gzip.GzipFile(fileobj=target,mode='wb') as compressed:
            while chunk:=dump.stdout.read(1024*1024):compressed.write(chunk)
        assert dump.wait()==0,'Backup failed'
    # Fingerprint every original non-simulator position, and preserve master vehicle rows.
    original_max=int(query("SELECT COALESCE(MAX(id),0) FROM gps_positions WHERE device_id NOT IN ("+ids+")"))
    fingerprint="SELECT SHA2(GROUP_CONCAT(CONCAT_WS('|',id,device_id,gps_time,latitude,longitude,speed_kmh) ORDER BY id SEPARATOR ';'),256) FROM gps_positions WHERE device_id NOT IN ("+ids+") AND id<="+str(original_max)
    before=query(fingerprint);masters_before=query('SELECT JSON_ARRAYAGG(JSON_OBJECT(\'id\',id,\'plate\',plate_number,\'name\',vehicle_name,\'fleet\',fleet_code)) FROM vehicles')
    sql_path=private/'fixture.sql'
    with sql_path.open('x') as f:
        os.chmod(sql_path,0o600)
        f.write("SET time_zone='+00:00'; START TRANSACTION;\n")
        # The source tag is the boundary: never remove unrelated device messages/history.
        f.write('DELETE e FROM device_events e JOIN device_raw_messages r ON r.id=e.raw_message_id WHERE '+predicate+';\n')
        f.write('DELETE p FROM gps_positions p JOIN device_raw_messages r ON r.id=p.raw_message_id WHERE '+predicate+';\n')
        f.write('DELETE r FROM device_raw_messages r WHERE '+predicate+';\n')
        for car,info in zip(dataset,summary):
            device=info['device_id'];imei='00000999000'+str(car['index']+1).zfill(4);last_state=None
            for n,p in enumerate(car['points']):
                payload={'simulation':True,'batch':TAG,'simulation_version':VERSION,'sequence':n,'scenario_date':p['scenario_date'],'profile':car['profile'],'condition':p['condition'],'state':p['state'],'gps_time':p['at'].astimezone(UTC).isoformat(),'latitude':p['lat'],'longitude':p['lon'],'speed_kmh':p['speed'],'heading_deg':p['heading']}
                raw=json.dumps(payload,separators=(',',':'));received=p['at']+dt.timedelta(seconds=2)
                f.write('INSERT INTO device_raw_messages(device_id,imei,message_type,protocol_version,raw_payload,parsed_payload,received_at,parse_status,created_at) VALUES('+str(device)+','+text(imei)+",'SIM_POSITION','SIM-V2',"+text('SIMULASI '+raw)+','+text(raw)+','+stamp(received)+",'success',"+stamp(received)+'); SET @raw=LAST_INSERT_ID();\n')
                f.write('INSERT INTO gps_positions(device_id,raw_message_id,gps_time,device_send_time,latitude,longitude,speed_kmh,heading_deg,altitude_m,gps_accuracy,satellites,created_at) VALUES('+str(device)+',@raw,'+stamp(p['at'])+','+stamp(received)+f",{p['lat']:.7f},{p['lon']:.7f},{p['speed']:.2f},{p['heading']},25,5,12,"+stamp(received)+');\n')
                state=(p['scenario_date'],p['state'])
                if state!=last_state:
                    f.write('INSERT INTO device_events(device_id,raw_message_id,message_type,event_code,event_value,event_time,created_at) VALUES('+str(device)+",@raw,'SIM_POSITION',"+text('SIM_'+p['state'].upper())+','+text('SIMULASI '+p['condition']+' / '+p['state'])+','+stamp(p['at'])+','+stamp(received)+');\n')
                last_state=state
            f.write('UPDATE gps_devices SET last_seen_at='+stamp(car['points'][-1]['at'])+",protocol_version='SIM-V2',metadata=JSON_SET(metadata,'$.simulation_version',2,'$.scenario',"+text(car['profile'])+",'$.start_wib',"+text(start.isoformat())+",'$.seeded_at',"+text(now.isoformat())+'),row_version=row_version+1 WHERE id='+str(device)+" AND protocol_name='simulation';\n")
            audit={k:v for k,v in info.items() if k!='days'}
            f.write('INSERT INTO pastrack_audit_log(occurred_at,actor,action,entity,entity_id,after_data,request_id) VALUES(UTC_TIMESTAMP(3),\'simulation-seeder\',\'refresh_simulation\',\'devices\','+str(device)+','+text(json.dumps(audit))+','+text('SIM-V2-'+now.strftime('%Y%m%d%H%M')+'-'+str(device))+');\n')
        f.write('COMMIT;\n')
    with sql_path.open() as source:
        result=subprocess.run(mysql,stdin=source,env=process_env,capture_output=True,text=True)
    if result.returncode:raise RuntimeError(result.stderr[:1500])
    assert query(fingerprint)==before,'Original device positions changed unexpectedly'
    manifest['vehicle_master_before']=json.loads(masters_before)
    manifest['vehicle_master_after']=json.loads(query('SELECT JSON_ARRAYAGG(JSON_OBJECT(\'id\',id,\'plate\',plate_number,\'name\',vehicle_name,\'fleet\',fleet_code)) FROM vehicles'))
    manifest['original_positions_sha256']=before
    manifest['backup']=str(private/'before.sql.gz')
    (private/'manifest.json').write_text(json.dumps(manifest,indent=2));(private/'manifest.json').chmod(0o600)
    try:
        with socket.socket(socket.AF_UNIX,socket.SOCK_DGRAM) as signal:
            signal.sendto(b'positions',config.get('REALTIME_SOCKET_PATH','/tmp/pastrack/live.sock'))
    except OSError:
        print('History stored; realtime service unavailable for this invalidation.')
    print('APPLIED',manifest['total_positions'],'positions;',len(summary),'vehicles. Manifest:',private/'manifest.json')

if __name__=='__main__':main()
