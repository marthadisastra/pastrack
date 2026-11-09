"""Import the three authorized replay days without editing existing vehicles."""
import json
import hashlib
from datetime import datetime, timedelta

def import_replay(c, scenarios):
 q=c.cursor()
 plate='UJI JKT BKS'
 start=datetime.fromisoformat(scenarios[0]['samples'][0]['at'])
 def insert(sql,args=()):
  q.execute(sql,args)
  return q.lastrowid
 c.begin()
 try:
   q.execute("SELECT id FROM vehicles WHERE plate_number=%s FOR UPDATE",(plate,))
   existing=q.fetchone()
   if existing:
    q.execute("SELECT COUNT(*) n FROM vehicle_assignments WHERE vehicle_id=%s AND provenance='fixture_jkt_bks'",(existing['id'],))
    assert q.fetchone()['n']==1, 'vehicle identifier already in use'
    raise ValueError('Replay already imported')
   vehicle=insert("INSERT INTO vehicles(plate_number,vehicle_name,fleet_code,row_version) VALUES(%s,'Kendaraan Uji Jakarta–Bekasi',NULL,1)",(plate,))
   model_id=insert("INSERT INTO gps_device_models(manufacturer,model,protocol_family) VALUES('PasTrack','Virtual Replay','fixture')")
   imei='REPLAY-JKT-BKS-01'
   device=insert("INSERT INTO gps_devices(imei,model_id,is_active) VALUES(%s,%s,1)",(imei,model_id))
   assignment=insert("INSERT INTO vehicle_assignments(vehicle_id,device_id,starts_at,provenance) VALUES(%s,%s,%s,'fixture_jkt_bks')",(vehicle,device,start-timedelta(days=1)))
   sensor=insert("INSERT INTO ble_sensors(sensor_identifier,mac,model,is_active) VALUES(%s,%s,'Virtual Fuel',1)",('REPLAY-JKT-BKS-FUEL','020000103101'))
   binding=insert("INSERT INTO ble_sensor_bindings(sensor_id,vehicle_id,device_id,starts_at,binding_source) VALUES(%s,%s,%s,%s,'fixture_jkt_bks')",(sensor,vehicle,device,start-timedelta(days=1)))
   tank=insert("INSERT INTO fuel_tank_profiles(vehicle_id,name,capacity_l,revision,effective_from) VALUES(%s,'Tangki utama 400 L',400,1,%s)",(vehicle,start-timedelta(days=1)))
   fs=insert("INSERT INTO fuel_sensors(sensor_id,tank_profile_id,measurement_unit,decoding_state) VALUES(%s,%s,'mm','verified')",(sensor,tank))
   calibration=insert("INSERT INTO fuel_calibrations(fuel_sensor_id,tank_profile_id,revision,media,input_quantity,input_unit,output_unit,approval_state,effective_from) VALUES(%s,%s,1,'fuel_tank','fuel_height_raw','mm','liters','approved',%s)",(fs,tank,start-timedelta(days=1)))
   for raw,liters in [(0,0),(1000,400)]:insert("INSERT INTO fuel_calibration_points(calibration_id,input_value,output_value,evidence) VALUES(%s,%s,%s,'fixture_jkt_bks')",(calibration,raw,liters))
   results=[]
   for scenario in scenarios:
    samples=scenario['samples']
    position_ids=[];reading_ids=[]
    for i,s in enumerate(samples):
     liters=s['liters'];height=round(liters*2.5,3);at=datetime.fromisoformat(s['at'])
     # Preserve synthetic origin internally; imported samples use exactly the ordinary relations.
     payload=json.dumps({'origin':'fixture_jkt_bks','imei':imei,'timestamp':at.isoformat(),'latitude':s['latitude'],'longitude':s['longitude'],'speed_kmh':round(s['speed'],3),'fuel_height_raw':height},separators=(',',':')).encode()
     raw=insert("INSERT INTO device_raw_messages(device_id,payload,payload_sha256,received_at,device_time) VALUES(%s,%s,%s,%s,%s)",(device,payload,hashlib.sha256(payload).digest(),at,at))
     insert("INSERT INTO protocol_parse_logs(raw_message_id,parser_version,parse_status,processing_status,fields,processed_at) VALUES(%s,'fixture_jkt_bks','success','processed',%s,%s)",(raw,json.dumps({'origin':'fixture_jkt_bks'}),at))
     position=insert("INSERT INTO gps_positions(device_id,vehicle_id,assignment_id,raw_message_id,sample_index,gps_time,received_at,latitude,longitude,speed_kmh,satellite_count,hdop,fix_status,validation_state) VALUES(%s,%s,%s,%s,0,%s,%s,%s,%s,%s,NULL,NULL,'valid','validated')",(device,vehicle,assignment,raw,at,at,s['latitude'],s['longitude'],s['speed']));position_ids.append(position)
     telemetry=insert("INSERT INTO sensor_telemetry(sensor_id,device_id,binding_id,raw_message_id,accessory_index,sensor_time,received_at,fuel_height_raw,temperature_c,rssi,validation_state,decoder_version) VALUES(%s,%s,%s,%s,0,%s,%s,%s,NULL,NULL,'validated','fixture_jkt_bks')",(sensor,device,binding,raw,at,at,height))
     reading=insert("INSERT INTO fuel_readings(fuel_sensor_id,telemetry_id,vehicle_id,calibration_id,reading_at,received_at,raw_measurement,raw_unit,fuel_liters,fuel_percent,validity) VALUES(%s,%s,%s,%s,%s,%s,%s,'mm',%s,%s,'valid')",(fs,telemetry,vehicle,calibration,at,at,height,liters,liters/4));reading_ids.append(reading)
    event=scenario['event']
    if event:
     first,last=event['first'],event['last']
     insert("INSERT INTO fuel_events(vehicle_id,fuel_sensor_id,start_reading_id,end_reading_id,event_type,delta_l,event_at,rule_version,confidence) VALUES(%s,%s,%s,%s,%s,%s,%s,'fixture_jkt_bks','sustained_change')",(vehicle,fs,reading_ids[first],reading_ids[last],event['kind'],event['delta'],datetime.fromisoformat(samples[last]['at'])))
    for leg in scenario['trips']:
     first,last=leg['first'],leg['last']
     speeds=[sample['speed'] for sample in samples[first:last+1]]
     moving=15*sum(speed>=3 for speed in speeds)
     insert("INSERT INTO trips(vehicle_id,start_position_id,end_position_id,started_at,ended_at,distance_m,driving_seconds,stopped_seconds,max_speed_kmh,average_speed_kmh,calculation_version,confidence) VALUES(%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,'fixture_jkt_bks','gps_validated')",(vehicle,position_ids[first],position_ids[last],datetime.fromisoformat(samples[first]['at']),datetime.fromisoformat(samples[last]['at']),round(leg['distance_km']*1000,3),moving,leg['seconds']-moving,max(speeds),round(leg['distance_km']*3600/max(1,moving),2)))
    end=datetime.fromisoformat(samples[-1]['at'])
    insert("INSERT INTO fuel_consumption_summaries(vehicle_id,period_start,period_end,consumed_l,distance_m,coverage_percent,algorithm_version) VALUES(%s,%s,%s,%s,%s,100,'fixture_jkt_bks')",(vehicle,datetime.fromisoformat(samples[0]['at']),end,scenario['consumed_l'],scenario['distance_km']*1000))
    results.append(dict(day=scenario['day'],traffic=scenario['traffic'],samples=len(samples),minutes=(len(samples)-1)/4,distance_km=scenario['distance_km'],fuel_event=event))
   insert("INSERT INTO latest_positions(vehicle_id,position_id,gps_time,received_at) VALUES(%s,%s,%s,%s)",(vehicle,position_ids[-1],end,end))
   insert("INSERT INTO latest_fuel_readings(fuel_sensor_id,reading_id,last_valid_reading_id,updated_at) VALUES(%s,%s,%s,%s)",(fs,reading_ids[-1],reading_ids[-1],end))
   insert("INSERT INTO audit_log(action,entity,entity_id,request_id,occurred_at) VALUES('import_journey','vehicles',%s,%s,UTC_TIMESTAMP(3))",(vehicle,hashlib.sha256(b'fixture_jkt_bks').hexdigest()))
   c.commit()
   return dict(vehicle_id=vehicle,plate=plate,scenarios=results)
 except BaseException:
   c.rollback()
   raise
