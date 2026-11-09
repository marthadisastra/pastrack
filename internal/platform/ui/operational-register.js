'use strict';
const relationLabels={vehicle_id:'Kendaraan',device_id:'Perangkat GPS',fuel_sensor_id:'Sensor BBM',sensor_id:'Sensor'};
function relationIdentity(row,key){
 if(key==='vehicle_id')return vehicleIdentity(row);
 if(key==='device_id')return routeKey==='sensor-management'?(row.gps_imei||row.device_identifier):row.device_identifier;
 return row.sensor_name?row.sensor_name+' ('+(row.sensor_identifier||'—')+')':row.sensor_identifier;
}
const operationalRegisters = {
 'live-tracking':'plate_number vehicle_name movement_status latitude longitude speed_kmh gps_time',
 geofences:'name shape center_lat center_lon radius_m is_active',
 'fuel-monitoring':'vehicle_id fuel_sensor_id received_at fuel_liters fuel_percent validity invalid_reason',
 'fuel-analytics':'vehicle_id period_start period_end consumed_l distance_m coverage_percent',
 'gps-quality':'device_id position_id event_at satellite_count hdop fix_status',
 'device-connectivity':'device_id event_at mcc mnc cell_id rssi',
 'data-activity':'device_id raw_message_id parser_version parse_status processing_status processed_at',
 'raw-protocol-logs':'id device_id received_at message_type protocol_version',
 'alert-center':'id vehicle_id state raised_at acknowledged_at resolved_at',
 reports:'id report_type state created_at expires_at',
 'device-management':'identifier vehicle_id connectivity gps_quality network last_received_at',
 'sensor-management':'sensor_name identifier model device_id device_protocol_version vehicle_id pairing_active capacity_l decoding_state received_at',
 'user-management':'username full_name email role is_active last_login_at',
 notifications:'title body created_at read_at',
 audit:'actor_id action entity entity_id occurred_at',
 models:'manufacturer model protocol_family',
 tanks:'vehicle_id name capacity_l revision effective_from effective_to',
 calibrations:'fuel_sensor_id media input_quantity output_unit revision approval_state',
 'event-types':'code description',
 'event-severities':'code rank',
 'geofence-assignments':'geofence_id vehicle_id starts_at ends_at',
 'sensor-telemetry':'sensor_id device_id received_at fuel_height_raw temperature_c rssi validation_state',
 events:'vehicle_id device_id event_at raw_code raw_value verification_state'
};
operationalRegisters['vehicle-fuel']=operationalRegisters['fuel-monitoring'];
operationalRegisters['vehicle-device']='identifier model manufacturer vehicle_id firmware_version is_active';
operationalRegisters['vehicle-calibration']=operationalRegisters.calibrations;
const isOperationalRegister=()=>Object.hasOwn(operationalRegisters,routeKey);

function mountOperationalRegister(){
 const screen=$('#screen');
 screen.replaceChildren();
 screen.onclick=null;
 screen.className='fe-screen operational-register'+(routeKey==='sensor-management'?' sensor-register':'');
 const heading=el('h1',active.title);
 screen.append(heading);
 if(routeKey.startsWith('vehicle-')){
  const tabs=el('nav',undefined,'overview-tabs');
  for(const [key,title] of Object.entries({overview:'Telematika',trips:'Perjalanan',fuel:'BBM',calibration:'Kalibrasi',device:'Perangkat'})){
   const link=el('a',title);
   link.href='#vehicles/'+selectedVehicle+'/'+key;
   if(detailRoutes[key]===routeKey)link.setAttribute('aria-current','page');
   tabs.append(link);
  }
  screen.append(tabs);
  if(routeKey==='vehicle-calibration'){const devices=el('a','Kelola perangkat GPS');devices.href='#device-management';screen.append(devices);}
 }
 const form=el('form',undefined,'fe-filters');
 const field=(title,input)=>{const name=el('label',title);name.append(input);form.append(name);};
 if(active.search){const search=el('input');search.type='search';search.name='search';field('Cari',search);}
 if(active.vehicle){
  const select=el('select');select.name='vehicle';select.append(new Option('All',''));
  for(const vehicle of vehicles)select.append(new Option(vehicleIdentity(vehicle)||'Identitas tidak tersedia',vehicle.id));
  field('Kendaraan',select);
 }
 if(active.history)for(const [name,title] of [['from','Start date (WIB)'],['to','End date (WIB)']]){
  const dateInput=el('input');dateInput.type='date';dateInput.name=name;field(title,dateInput);
 }
 const apply=el('button','Terapkan','fe-primary');apply.type='submit';form.append(apply);
 form.onsubmit=e=>{
  e.preventDefault();const data=new FormData(form),from=data.get('from'),to=data.get('to');
  try{Object.assign(filters,from||to?tripDateBounds(from,to):{from:'',to:''});}
  catch(error){$('#register-status').textContent=error.message;return;}
  filters.search=data.get('search')||'';filters.vehicle=data.get('vehicle')||'';cursor=null;history=[];load();
 };
 const refresh=el('button','Muat ulang');refresh.type='button';refresh.onclick=()=>load();form.append(refresh);
 if(routeKey!=='sensor-management'&&active.create&&!identity.read_only&&identity.role!=='viewer'&&!routeKey.startsWith('vehicle-')){
  const add=el('button',routeKey==='reports'?'Buat laporan':'Tambah');
  add.type='button';add.onclick=()=>{
   if(routeKey==='device-management')return pickDiscoveredDevice();
   if(routeKey==='reports')return edit(null,{...active,create:['report_type','format','from','to','vehicle_id']});
   edit();
  };form.append(add);
 }
 screen.append(form);
 const status=el('p');status.id='register-status';status.setAttribute('role','status');screen.append(status);
 if(active.map){const map=el('section',undefined,'stitch-map');map.dataset.map='';screen.append(map);}
 const wrapper=el('section',undefined,'th-table-wrap');
 const table=el('table');const head=el('tr');
 for(const key of operationalRegisters[routeKey].split(' '))head.append(el('th',(routeKey==='sensor-management'&&({identifier:'Sensor (MAC)',sensor_name:'Nama',device_id:'GPS (IMEI)',capacity_l:'Kapasitas kendaraan (L)',decoding_state:'Status kalibrasi',received_at:'Terakhir diterima (WIB)'}[key]))||relationLabels[key]||(routeKey==='device-management'&&deviceColumns[key])||({device_protocol_version:'Protokol GPS',pairing_active:'Status relasi'}[key])||label(key)));
 head.append(el('th','Aksi'));
 const thead=el('thead');thead.append(head);table.append(thead,el('tbody'));wrapper.append(table);screen.append(wrapper);
 const pager=el('div',undefined,'stitch-live-pagination');
 const prev=el('button','Sebelumnya'),next=el('button','Berikutnya');
 prev.dataset.prev='';next.dataset.next='';
 prev.onclick=()=>{if(history.length){cursor=history.pop();load();}};
 next.onclick=()=>{if(nextCursor){history.push(cursor);cursor=nextCursor;load();}};
 pager.append(el('span'),prev,next);screen.append(pager);
}

const deviceColumns={identifier:'Perangkat GPS',connectivity:'Koneksi',gps_quality:'Kualitas GPS',network:'Jaringan',last_received_at:'Terakhir Diterima'};
function deviceChip(text,tone='neutral'){return el('span',text,'device-chip device-chip-'+tone);}
function deviceConnectivity(row){
 if(row.is_active===0||row.is_active===false)return deviceChip('Nonaktif');
 if(row.connectivity==='ONLINE')return deviceChip('Online','good');
 if(row.connectivity==='DEVICE_OFFLINE')return deviceChip('Offline','bad');
 return deviceChip('Belum ada data');
}
function deviceGPS(row){
 if(row.fix_status==='valid')return deviceChip('Fix valid','good');
 if(row.fix_status==='no_fix')return deviceChip('Tanpa fix','warn');
 return deviceChip(row.fix_status||'Belum ada data',row.fix_status?'warn':'neutral');
}
function deviceNetwork(row){
 const code=[row.mcc,row.mnc].filter(value=>value!=null&&value!=='').join(' / ');
 return code?'MCC / MNC '+code:null;
}
function deviceCell(key,row){
 const cell=el('td');
 if(key==='identifier'){cell.append(el('strong',row.identifier||'—'));if(row.model)cell.append(el('small',row.model));}
 else if(key==='connectivity')cell.append(deviceConnectivity(row));
 else if(key==='gps_quality'){
  cell.append(deviceGPS(row));
  const measures=[row.satellite_count==null?'':row.satellite_count+' satelit',row.hdop==null?'':'HDOP '+row.hdop].filter(Boolean);
  if(measures.length)cell.append(el('small',measures.join(' · ')));
 }else if(key==='network'){cell.append(deviceNetwork(row)||deviceChip('Belum ada data'));}
 else cell.textContent=row.last_received_at?date(row.last_received_at):'—';
 cell.classList.add('device-summary-cell');
 return cell;
}
function deviceDetail(row){
 const panel=el('div',undefined,'device-detail');
 const section=(title,values,chip)=>{
  const block=el('section');block.append(el('h3',title));if(chip)block.append(chip);
  block.append(detailFields(values));panel.append(block);
 };
 section('Identitas perangkat',{model:row.model,manufacturer:row.manufacturer,Kendaraan:vehicleIdentity(row),firmware_version:row.firmware_version,protocol_version:row.protocol_version});
 section('Koneksi server',{'Terakhir diterima (WIB)':row.last_received_at?date(row.last_received_at):null},deviceConnectivity(row));
 section('Kualitas GPS',{'Jumlah satelit':row.satellite_count,HDOP:row.hdop,'Waktu pengukuran (WIB)':row.gps_quality_at?date(row.gps_quality_at):null},deviceGPS(row));
 section('Jaringan seluler dan SIM',{MCC:row.mcc,MNC:row.mnc,LAC:row.lac,'Cell ID':row.cell_id,RSSI:row.rssi,'Waktu pengukuran (WIB)':row.network_at?date(row.network_at):null,'Identitas SIM':'Belum ada data'});
 const batteryEvent={GTBPL:'Alarm baterai cadangan rendah',GTBTC:'Mulai mengisi daya'}[row.battery_event];
 section('Baterai GPS',{
  'Tegangan cadangan terakhir':row.battery_voltage==null?'Belum ada data':num(row.battery_voltage,'V'),
  'Waktu laporan tegangan (WIB)':row.battery_voltage_at?date(row.battery_voltage_at):null,
  'Event terakhir':batteryEvent||'Belum ada data',
  'Waktu laporan event (WIB)':row.battery_event_at?date(row.battery_event_at):null
 },batteryEvent?deviceChip(batteryEvent,row.battery_event==='GTBPL'?'warn':'neutral'):null);
 inspect('Detail GPS '+(row.identifier||''),panel);
}

async function editGPS(row){
 const form=el('form',undefined,'stitch-setup-form device-edit');
 const summary=el('div',undefined,'device-edit-identity');
 summary.append(el('strong','IMEI '+(row.identifier||'—')),el('small',vehicleIdentity(row)||'Belum terhubung ke kendaraan'));
 const status=el('p','Memuat katalog model…');status.setAttribute('role','status');
 form.append(summary,status);
 inspect('Edit perangkat GPS',form);
 try{
  const models=[];
  let cursor=null;
  do{
   const params=new URLSearchParams({limit:'200'});
   if(cursor)params.set('cursor',cursor);
   const page=await api('device-models?'+params);
   models.push(...page.data);
   cursor=page.page?.has_more?page.page.next_cursor:null;
  }while(cursor);
  const model=el('select');model.name='model_id';model.required=true;
  model.append(new Option('Pilih model perangkat',''));
  for(const entry of models)model.append(new Option([entry.manufacturer,entry.model].filter(Boolean).join(' · '),String(entry.id)));
  const initialModel=String(row.model_id??'');
  model.value=initialModel;
  const modelLabel=el('label','Model perangkat');modelLabel.append(model);
  const active=el('select');active.name='is_active';active.append(new Option('Aktif','true'),new Option('Nonaktif','false'));
  const initialActive=String(row.is_active===true||Number(row.is_active)===1);
  active.value=initialActive;
  const activeLabel=el('label','Status perangkat');activeLabel.append(active);
  const note=el('small','Perangkat nonaktif berhenti diproses sebagai GPS aktif. Hubungan kendaraan dikelola melalui Hubungkan kendaraan.');
  const error=el('p',undefined,'stitch-error');error.setAttribute('role','alert');
  const actions=el('div',undefined,'stitch-setup-actions');
  const cancel=el('button','Batal'),save=el('button','Simpan perubahan','stitch-setup-primary');
  cancel.type='button';cancel.onclick=()=>$('#inspector').close();
  save.type='submit';
  actions.append(cancel,save);
  const update=()=>{save.disabled=!model.value||(model.value===initialModel&&active.value===initialActive);};
  model.onchange=active.onchange=update;update();
  status.remove();form.append(modelLabel,activeLabel,note,error,actions);
  form.onsubmit=async event=>{
   event.preventDefault();
   if(save.disabled)return;
   save.disabled=true;model.disabled=active.disabled=true;error.textContent='';
   try{
    await api('devices/'+row.id,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({model_id:Number(model.value),is_active:active.value==='true'})});
    $('#inspector').close();await load();toast('Pengaturan GPS tersimpan.');
   }catch(e){error.textContent=e.message;}finally{model.disabled=active.disabled=false;update();}
  };
 }catch(e){status.textContent='Katalog model gagal dimuat. Buka kembali Edit untuk mencoba lagi. '+e.message;}
}

function renderOperationalRegister(table,list,endpoint){
 const body=table.querySelector('tbody');body.replaceChildren();
 if(!list.length){setEmpty(body,'Belum ada data untuk filter ini.');return;}
 for(const row of list){
  const tr=el('tr');
  for(const key of operationalRegisters[routeKey].split(' ')){
   if(routeKey==='device-management'&&deviceColumns[key]){tr.append(deviceCell(key,row));continue;}
   if(routeKey==='sensor-management'&&key==='identifier'&&row.sensor_mac){tr.append(el('td',row.sensor_mac));continue;}
   if(key==='pairing_active'){const td=el('td');td.append(Number(row.pairing_active)?deviceChip('Terpasang','good'):deviceChip('Nonaktif · relasi belum valid'));tr.append(td);continue;}
   if(routeKey==='sensor-management'&&key==='decoding_state'){tr.append(el('td',({raw_units_unverified:'Belum dikalibrasi',uncalibrated:'Belum dikalibrasi',calibrated_raw_signal:'Terkalibrasi'}[row[key]])||row[key]||'Belum tersedia'));continue;}
   const value=relationLabels[key]?relationIdentity(row,key):row[key];
   tr.append(el('td',value==null?'—':(/_at$|_time$|^period_(start|end)$|^effective_(from|to)$/.test(key)?date(value):String(value))));
  }
  const actions=el('td');
  const detail=el('button','Detail');
  detail.onclick=()=>{
   if(routeKey==='device-management'||routeKey==='vehicle-device')return deviceDetail(row);
   if(routeKey==='sensor-management')return inspect('Detail sensor '+(row.sensor_mac||row.identifier),{'Nama sensor':row.sensor_name||'Belum dikirim','MAC sensor':row.sensor_mac||row.identifier,'IMEI GPS penerima':row.gps_imei||row.device_identifier,'Protokol kiriman GPS':row.device_protocol_version,'Kendaraan':vehicleIdentity(row),'Raw sumber':row.source_raw_id,'Terakhir diterima (WIB)':date(row.received_at),'Validitas relasi':Number(row.pairing_active)?'GPS dan kendaraan teridentifikasi dari kiriman':'GPS atau kendaraan aktif belum tersedia','Status konversi BBM':row.decoding_state});
   if(routeKey==='raw-protocol-logs')return inspectRaw(row.id);
   if(routeKey==='alert-center')return alertDetail(row);
   if(['calibrations','vehicle-calibration'].includes(routeKey))return calibrationDetail(row);
   inspect(active.title+' #'+row.id,row);
  };
  if(routeKey==='device-management')detail.classList.add('device-detail-button');
  actions.append(detail);
  if(endpoint==='reports'&&row.state==='completed'){
   const download=el('a','Unduh');download.href='/api/v1/reports/'+row.id+'/download';actions.append(download);
  }
  if(!identity.read_only&&identity.role!=='viewer'){
   if(routeKey==='device-management'){
    const link=el('button',row.vehicle_id?'Pindahkan / Tukar GPS':'Hubungkan kendaraan');link.dataset.rowAction='connect-vehicle';link.onclick=()=>connectGPS(row);actions.append(link);
    if(identity.role==='admin'){const remove=el('button','Hapus','inventory-delete');remove.onclick=()=>confirmInventoryDelete('devices',row);actions.append(remove);}
   }
   if(routeKey==='sensor-management'&&identity.role==='admin'){
    const calibration=el('button','Kalibrasi');calibration.dataset.rowAction='quick-calibration';calibration.disabled=!Number(row.pairing_active);if(calibration.disabled)calibration.title='Sensor memerlukan relasi GPS dan kendaraan yang valid.';calibration.onclick=()=>quickFuel(row);actions.append(calibration);
   }
   if(active.create&&endpoint!=='reports'&&!routeKey.startsWith('vehicle-')){
    const editButton=el('button','Edit');editButton.onclick=()=>{if(routeKey==='device-management')return editGPS(row);if(routeKey==='sensor-management')return edit({...row,is_active:!!Number(row.is_active)},{...active,create:['is_active']});edit(row);};if(routeKey==='sensor-management'&&!Number(row.pairing_active)){editButton.disabled=true;editButton.title='Sensor memerlukan relasi GPS dan kendaraan yang valid.';}actions.append(editButton);
   }
  }
  tr.append(actions);body.append(tr);
 }
}

async function pickDiscoveredDevice(){
 const panel=el('div'), status=el('p','Memuat laporan perangkat…');
 status.setAttribute('role','status');
 panel.append(status);
 inspect('Tambah perangkat dari laporan server',panel);
 try{
  const {data}=await api('devices/discovered');
  if(!data.length){
   status.textContent='Belum ada perangkat yang memenuhi syarat pada laporan 7 hari terakhir. Pastikan perangkat mengirim data, lalu buka kembali daftar ini.';
   return;
  }
  status.textContent='Pilih perangkat berdasarkan akhiran IMEI dan waktu laporan terakhir.';
  const form=el('form',undefined,'fe-investigation-form');
  const field=el('label','Perangkat belum terdaftar');
  const picker=el('select');
  picker.required=true;
  picker.append(new Option('Pilih perangkat',''));
  for(const row of data){
   const option=new Option(row.identifier+' · '+(row.model||'Model belum terverifikasi')+' · '+row.protocol_version+' · '+date(row.received_at),row.receipt_id);
   option.disabled=row.registerable===false;
   picker.append(option);
  }
  field.append(picker);
  const submit=el('button','Daftarkan perangkat','fe-primary');
  submit.type='submit';
  form.append(field,submit);
  panel.append(form);
  form.onsubmit=async event=>{
   event.preventDefault();
   if(!picker.value)return;
   submit.disabled=true;
   try{
    await post('devices/discovered',{receipt_id:Number(picker.value)});
    $('#inspector').close();
    await load();
    toast('Perangkat terdaftar. Hubungkan ke kendaraan bila diperlukan.');
   }catch(error){
    const messages={'device already registered':'Perangkat sudah terdaftar. Muat ulang daftar.','device model not supported':'Model dari laporan belum tersedia di katalog perangkat.','device receipt not found':'Laporan perangkat sudah tidak tersedia. Buka ulang daftar.'};
    status.textContent=messages[error.message]||error.message;
   }finally{submit.disabled=false;}
  };
 }catch(error){status.textContent=error.message;}
}
