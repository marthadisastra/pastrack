"""User-authorized October 31 journey fixtures imported through the ordinary schema.
Origin is retained in internal provenance and audit; no existing vehicle is modified.
Run with a private DBA connection supplied by the release script, or a local test DB.
"""
import json,math,hashlib
from datetime import datetime,timedelta,timezone
from pathlib import Path

def build_samples(route):
 coords=route['coordinates'];cumulative=[0.0]
 def km(a,b):
  ax,ay,bx,by=map(math.radians,[a[0],a[1],b[0],b[1]])
  h=math.sin((by-ay)/2)**2+math.cos(ay)*math.cos(by)*math.sin((bx-ax)/2)**2
  return 6371*2*math.asin(min(1,math.sqrt(h)))
 for i in range(1,len(coords)):cumulative.append(cumulative[-1]+km(coords[i-1],coords[i]))
 weights=[0 if i==0 or 350<=i<=370 or i>=710 else .85+.15*math.sin(i/38) for i in range(721)];total_weight=sum(weights);distance=0;segment=1;samples=[];start=datetime(2026,10,30,23,0)
 for i,weight in enumerate(weights):
  step=weight/total_weight*cumulative[-1];distance+=step
  while segment<len(coords)-1 and cumulative[segment]<distance:segment+=1
  a,b=coords[segment-1],coords[segment];fraction=max(0,min(1,(distance-cumulative[segment-1])/max(1e-12,cumulative[segment]-cumulative[segment-1])))
  lon,lat=a[0]+(b[0]-a[0])*fraction,a[1]+(b[1]-a[1])*fraction
  samples.append({'at':start+timedelta(seconds=i*30),'latitude':lat,'longitude':lon,'speed':step*120,'consumed':distance/3.5,'distance_km':distance,'drop':max(0,min(40,(i-360)*10))})
 return samples

def import_journeys(c,route):
 q=c.cursor();samples=build_samples(route);start=samples[0]['at'];end=samples[-1]['at'];result=[]
 def insert(sql,args=()):q.execute(sql,args);return q.lastrowid
 c.begin()
 try:
  for index,plate in enumerate(['B 9101 PTR','B 9102 PTR']):
   q.execute('SELECT id FROM vehicles WHERE plate_number=%s',(plate,));existing=q.fetchone()
   if existing:
    vehicle=existing['id'];q.execute("SELECT COUNT(*) n FROM vehicle_assignments WHERE vehicle_id=%s AND provenance='fixture_oct31'",(vehicle,));assert q.fetchone()['n']==1,'plate already belongs to another vehicle'
    q.execute('SELECT COUNT(*) n FROM gps_positions WHERE vehicle_id=%s',(vehicle,));assert q.fetchone()['n']==721,'incomplete existing journey; manual inspection required'
    q.execute('SELECT COUNT(*) n FROM fuel_readings WHERE vehicle_id=%s',(vehicle,));assert q.fetchone()['n']==721,'incomplete existing readings'
    q.execute('SELECT COUNT(*) n FROM trips WHERE vehicle_id=%s',(vehicle,));assert q.fetchone()['n']==2,'incomplete existing journeys'
    result.append({'vehicle_id':vehicle,'plate_number':plate,'already_imported':True});continue
   vehicle=insert("INSERT INTO vehicles(plate_number,vehicle_name,fleet_code,row_version) VALUES(%s,'Hino 500 Jakarta–Cirebon','Logistik Antar Kota',1)",(plate,))
   insert("INSERT INTO vehicle_assets(vehicle_id,make_model,fuel_type) VALUES(%s,'Hino 500','Diesel')",(vehicle,))
   q.execute("SELECT id FROM gps_device_models WHERE model='GV30CEU' ORDER BY id LIMIT 1");model=q.fetchone()
   model_id=model['id'] if model else insert("INSERT INTO gps_device_models(manufacturer,model,protocol_family) VALUES('Queclink','GV30CEU','queclink')")
   imei='09900000003100'+str(index+1)
   device=insert("INSERT INTO gps_devices(imei,model_id,firmware_version,protocol_version,is_active) VALUES(%s,%s,'2.4.11','80201E0306',1)",(imei,model_id))
   assignment=insert("INSERT INTO vehicle_assignments(vehicle_id,device_id,starts_at,provenance) VALUES(%s,%s,%s,'fixture_oct31')",(vehicle,device,start-timedelta(days=1)))
   sensor=insert("INSERT INTO ble_sensors(sensor_identifier,mac,model,is_active) VALUES(%s,%s,'CFS300B',1)",('CFS300B-PTR-'+str(index+1),'02000010310'+str(index+1)))
   binding=insert("INSERT INTO ble_sensor_bindings(sensor_id,vehicle_id,device_id,starts_at,binding_source) VALUES(%s,%s,%s,%s,'fixture_oct31')",(sensor,vehicle,device,start-timedelta(days=1)))
   tank=insert("INSERT INTO fuel_tank_profiles(vehicle_id,name,capacity_l,revision,effective_from) VALUES(%s,'Tangki utama 400 L',400,1,%s)",(vehicle,start-timedelta(days=1)))
   fs=insert("INSERT INTO fuel_sensors(sensor_id,tank_profile_id,measurement_unit,decoding_state) VALUES(%s,%s,'mm','verified')",(sensor,tank))
   calibration=insert("INSERT INTO fuel_calibrations(fuel_sensor_id,tank_profile_id,revision,media,input_quantity,input_unit,output_unit,approval_state,effective_from) VALUES(%s,%s,1,'fuel_tank','fuel_height_raw','mm','liters','approved',%s)",(fs,tank,start-timedelta(days=1)))
   for raw,liters in [(0,0),(1000,400)]:insert("INSERT INTO fuel_calibration_points(calibration_id,input_value,output_value,evidence) VALUES(%s,%s,%s,'fixture_oct31')",(calibration,raw,liters))
   position_ids=[];reading_ids=[];last_ignition=None
   for i,s in enumerate(samples):
    liters=round(320-s['consumed']-(s['drop'] if index else 0),3);height=round(liters*2.5);at=s['at'];on=s['speed']>0
    # Preserve synthetic origin internally; imported samples use exactly the ordinary relations.
    payload=json.dumps({'origin':'fixture_oct31','imei':imei,'timestamp':at.isoformat(),'latitude':s['latitude'],'longitude':s['longitude'],'speed_kmh':round(s['speed'],3),'fuel_height_raw':height},separators=(',',':')).encode()
    raw=insert("INSERT INTO device_raw_messages(device_id,protocol_version,payload,payload_sha256,received_at,device_time) VALUES(%s,'80201E0306',%s,%s,%s,%s)",(device,payload,hashlib.sha256(payload).digest(),at,at))
    insert("INSERT INTO protocol_parse_logs(raw_message_id,parser_version,parse_status,processing_status,fields,processed_at) VALUES(%s,'fixture_oct31','success','processed',%s,%s)",(raw,json.dumps({'origin':'fixture_oct31'}),at))
    position=insert("INSERT INTO gps_positions(device_id,vehicle_id,assignment_id,raw_message_id,sample_index,gps_time,received_at,latitude,longitude,speed_kmh,satellite_count,hdop,fix_status,validation_state) VALUES(%s,%s,%s,%s,0,%s,%s,%s,%s,%s,14,0.8,'valid','validated')",(device,vehicle,assignment,raw,at,at,s['latitude'],s['longitude'],s['speed']));position_ids.append(position)
    telemetry=insert("INSERT INTO sensor_telemetry(sensor_id,device_id,binding_id,raw_message_id,accessory_index,sensor_time,received_at,fuel_height_raw,temperature_c,rssi,validation_state,decoder_version) VALUES(%s,%s,%s,%s,0,%s,%s,%s,29.4,-64,'validated','fixture_oct31')",(sensor,device,binding,raw,at,at,height))
    reading=insert("INSERT INTO fuel_readings(fuel_sensor_id,telemetry_id,vehicle_id,calibration_id,reading_at,received_at,raw_measurement,raw_unit,fuel_liters,fuel_percent,validity) VALUES(%s,%s,%s,%s,%s,%s,%s,'mm',%s,%s,'valid')",(fs,telemetry,vehicle,calibration,at,at,height,liters,liters/4));reading_ids.append(reading)
    if on!=last_ignition:
     insert("INSERT INTO device_events(device_id,vehicle_id,raw_message_id,event_at,received_at,raw_code,raw_value,source,verification_state) VALUES(%s,%s,%s,%s,%s,%s,%s,'fixture_oct31','verified')",(device,vehicle,raw,at,at,'GTIGN' if on else 'GTIGF','ON' if on else 'OFF'));last_ignition=on
   insert("INSERT INTO latest_positions(vehicle_id,position_id,gps_time,received_at) VALUES(%s,%s,%s,%s)",(vehicle,position_ids[-1],end,end))
   insert("INSERT INTO gps_device_status(device_id,last_received_at,last_raw_message_id,connectivity) VALUES(%s,%s,%s,'OFFLINE')",(device,end,raw))
   insert("INSERT INTO latest_fuel_readings(fuel_sensor_id,reading_id,last_valid_reading_id,updated_at) VALUES(%s,%s,%s,%s)",(fs,reading_ids[-1],reading_ids[-1],end))
   insert("INSERT INTO ble_sensor_status(sensor_id,telemetry_id,last_received_at,status,stale_after_seconds) VALUES(%s,%s,%s,'connected',300)",(sensor,telemetry,end))
   for first,last in [(1,356),(371,716)]:
    distance=0;driving=0;stopped_seconds=0;maximum=0
    for j in range(first+1,last+1):
     a,b=samples[j-1],samples[j];rad=math.pi/180;h=math.sin((b['latitude']-a['latitude'])*rad/2)**2+math.cos(a['latitude']*rad)*math.cos(b['latitude']*rad)*math.sin((b['longitude']-a['longitude'])*rad/2)**2
     if a['speed']>=3 or b['speed']>=3:distance+=6371000*2*math.asin(min(1,math.sqrt(h)));driving+=30
     else:stopped_seconds+=30
     maximum=max(maximum,b['speed'])
    insert("INSERT INTO trips(vehicle_id,start_position_id,end_position_id,started_at,ended_at,distance_m,driving_seconds,stopped_seconds,max_speed_kmh,average_speed_kmh,calculation_version,confidence) VALUES(%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,'fixture_oct31','gps_validated')",(vehicle,position_ids[first],position_ids[last],samples[first]['at'],samples[last]['at'],distance,driving,stopped_seconds,maximum,distance/max(driving,1)*3.6))
   event=None
   if index:event=insert("INSERT INTO fuel_events(vehicle_id,fuel_sensor_id,start_reading_id,end_reading_id,event_type,delta_l,event_at,rule_version,confidence) VALUES(%s,%s,%s,%s,'fuel_drop',-40,%s,'fixture_oct31','sustained_change')",(vehicle,fs,reading_ids[360],reading_ids[364],samples[364]['at']))
   insert("INSERT INTO fuel_consumption_summaries(vehicle_id,period_start,period_end,consumed_l,distance_m,coverage_percent,algorithm_version) VALUES(%s,%s,%s,%s,%s,100,'fixture_oct31')",(vehicle,start,end,samples[-1]['consumed'],samples[-1]['distance_km']*1000))
   insert("INSERT INTO audit_log(action,entity,entity_id,request_id,occurred_at) VALUES('import_journey','vehicles',%s,%s,UTC_TIMESTAMP(3))",(vehicle,hashlib.sha256(('fixture_oct31:'+plate).encode()).hexdigest()))
   result.append({'vehicle_id':vehicle,'plate_number':plate,'device_id':device,'fuel_sensor_id':fs,'positions':len(position_ids),'readings':len(reading_ids),'event_id':event})
  c.commit();return result
 except Exception:c.rollback();raise
