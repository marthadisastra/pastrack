'use strict';
let vehicleView='grid',vehicleSummary={},vehicleSort='plate';
function vehicleStatus(row){return {MOVING:'Bergerak',STOPPED:'Berhenti',OFFLINE:'Offline',STALE:'GPS lama',GPS_NO_FIX:'Tanpa GPS fix',UNASSIGNED:'Belum ada GPS',UNKNOWN_SPEED:'Kecepatan belum tersedia'}[row.movement_status]||'Belum ada data';}
async function mountVehicles(){
 const screen=$('#screen');screen.className='vehicle-page';screen.dataset.designSource='Desain/vehicles/code.html';
 screen.innerHTML='<header class="vehicle-page-heading"><div><small>FLEET MANAGEMENT</small><h1>Daftar & Inventaris Kendaraan Operasional</h1><p>Daftar seluruh kendaraan, kondisi GPS dan sensor BBM, serta riwayat perjalanan.</p></div><div class="vehicle-page-actions"><button data-vehicle-export>↓ Export CSV</button><button data-vehicle-add data-create class="vehicle-primary">＋ Tambah Kendaraan Baru</button></div></header><section class="vehicle-summary" aria-label="Ringkasan seluruh armada"></section><section class="vehicle-condition-summary" aria-label="Kondisi posisi seluruh armada"></section><section class="vehicle-filters"><form><label class="vehicle-search"><span>Cari kendaraan</span><input type="search" placeholder="Cari plat nomor, nama, atau kode kendaraan" aria-label="Cari kendaraan"></label><label><span>Fleet</span><select data-vehicle-fleet aria-label="Filter fleet kendaraan"><option value="">Semua fleet</option></select></label><label><span>Kondisi</span><select data-vehicle-status aria-label="Filter status kendaraan"></select></label><label><span>Urutan halaman</span><select data-vehicle-sort><option value="plate">Plat nomor</option><option value="name">Nama kendaraan</option><option value="code">Kode kendaraan</option></select></label><button type="submit">Terapkan</button></form><div class="vehicle-view-toolbar"><div><button data-vehicle-reset>Reset filter</button><button data-vehicle-refresh>Muat ulang</button><button data-vehicle-gps>Kelola GPS</button></div><div class="vehicle-view-toggle" aria-label="Tampilan daftar"><button data-view="grid">▦ Grid Card</button><button data-view="table">☷ Tabel Data</button></div></div></section><section class="vehicle-grid" aria-label="Kartu kendaraan"></section><section class="vehicle-table" hidden><table><thead><tr><th>Vehicle Unit</th><th>Kode kendaraan</th><th>Tracker Device</th><th>Movement Status</th><th>Speed</th><th>Fuel Level</th><th>Ignition</th><th>Remote</th><th>Actions</th></tr></thead><tbody></tbody></table></section><footer class="vehicle-footer"><p>GPS, sensor BBM, dan remote memakai laporan terakhir masing-masing perangkat.</p><div class="stitch-live-pagination"><span></span><div><button data-prev data-action="prev">Sebelumnya</button><button data-next data-action="next">Berikutnya</button></div></div></footer>';
 screen.onclick=event=>{if(event.target.closest('[data-row-action],[data-action]'))handleAction(event).catch(e=>toast(e.message));};
 const permitted=identity.role!=='viewer'&&!identity.read_only;
 $('[data-vehicle-add]').hidden=!permitted;$('[data-vehicle-add]').onclick=()=>editVehicle();
 $('[data-vehicle-export]').onclick=()=>exportVehicleRegistry();
 const status=$('[data-vehicle-status]');
 for(const[value,text]of [['','Semua kondisi'],['MOVING','Bergerak'],['STOPPED','Berhenti'],['OFFLINE','Offline'],['STALE','GPS lama'],['GPS_NO_FIX','Tanpa GPS fix'],['UNASSIGNED','Belum ada GPS'],['UNKNOWN_SPEED','Kecepatan belum tersedia']])status.append(new Option(text,value));
 const search=$('.vehicle-search input'),group=$('[data-vehicle-fleet]');
 const apply=()=>{filters.search=search.value;filters.status=status.value;filters.fleetCode=group.value;cursor=null;history=[];load();};
 $('.vehicle-filters form').onsubmit=e=>{e.preventDefault();apply();};
 status.onchange=group.onchange=apply;
 let timer;search.oninput=()=>{clearTimeout(timer);timer=setTimeout(()=>{if(routeKey==='vehicles')apply();},300);};
 $('[data-vehicle-reset]').onclick=()=>{search.value=status.value=group.value='';apply();};
 $('[data-vehicle-refresh]').onclick=()=>load();$('[data-vehicle-gps]').onclick=()=>{location.hash='device-management';};
 $('[data-vehicle-sort]').value=vehicleSort;$('[data-vehicle-sort]').onchange=e=>{vehicleSort=e.target.value;renderVehicleCards();};
 for(const button of $$('[data-view]'))button.onclick=()=>{vehicleView=button.dataset.view;syncVehicleView();};
 syncVehicleView();
 api('vehicles/options').then(result=>{if(routeKey==='vehicles')for(const item of result.data||[])if(item.fleet_code)group.append(new Option(item.fleet_code,item.fleet_code));}).catch(showError);
 await loadVehicles();
}
function syncVehicleView(){if(routeKey!=='vehicles')return;$('.vehicle-grid').hidden=vehicleView!=='grid';$('.vehicle-table').hidden=vehicleView!=='table';for(const button of $$('[data-view]'))button.setAttribute('aria-pressed',String(button.dataset.view===vehicleView));}
async function loadVehicles(){
 const version=++generation;
 try{
  const [result,summary,alertResult]=await Promise.all([api('vehicles?'+queryFor('vehicles')),api('fleet/overview'),api('alerts/events?limit=200')]);
  if(routeKey!=='vehicles'||version!==generation)return;
  records=result.data||[];const activeAlerts=(alertResult.data||[]).filter(a=>!['resolved','closed','false_positive'].includes(String(a.state||'').toLowerCase()));const byVehicle=new Map();for(const alert of activeAlerts){const key=String(alert.vehicle_id);if(!byVehicle.has(key))byVehicle.set(key,new Set());byVehicle.get(key).add(String(alert.rule_code||''));}records.forEach(row=>row.active_alert_codes=[...(byVehicle.get(String(row.id))||[])].join(','));fleet=records;vehicleSummary=summary.data||{};nextCursor=result.page?.next_cursor??null;
  renderTable($('.vehicle-table table'),records,'vehicles');renderVehicleCards();renderVehicleSummary();bindPager(result.page);startValueStream();
 }catch(e){if(routeKey==='vehicles'&&version===generation)showError(e);}
}
function renderVehicleSummary(){
 const host=$('.vehicle-summary');if(!host)return;host.replaceChildren();
 for(const[label,key,tone]of [['TOTAL ARMADA','total_fleet','blue'],['GPS ONLINE','devices_online','green'],['GPS OFFLINE','vehicles_offline','red'],['BELUM ADA GPS','unassigned','blue']]){
  const card=el('article',undefined,'vehicle-summary-card '+tone);card.dataset.summaryKey=key;card.append(el('small',label),el('strong',vehicleSummary[key]==null?'—':String(vehicleSummary[key])),el('span','Kendaraan'));host.append(card);
 }
 const states=$('.vehicle-condition-summary');if(states){states.replaceChildren();states.append(el('small','KONDISI POSISI GPS'));
  for(const[label,key]of [['Bergerak','moving'],['Berhenti','stopped'],['GPS lama','gps_stale'],['Tanpa GPS fix','gps_no_fix'],['Kecepatan belum tersedia','unknown_speed']]){
   const chip=el('span',undefined,'vehicle-condition-chip');chip.dataset.summaryKey=key;chip.append(el('span',label),el('strong',vehicleSummary[key]==null?'—':String(vehicleSummary[key])));states.append(chip);
  }
 }

}
function vehicleRemoteControl(row){
 const wrap=el('div',undefined,'vehicle-engine-control');const label=el('div');label.append(el('small','Remote mesin'),el('strong',''));
 const button=el('button',undefined,'vehicle-engine-switch');button.type='button';button.dataset.rowAction='remote-output';button.setAttribute('role','switch');
 updateRemoteButton(button,row);label.lastChild.textContent=button.getAttribute('aria-checked')==='true'?'Enabled':'Disabled';
 if(typeof row.remote_output_active!=='boolean'&&!row.remote_last_command){label.lastChild.textContent='Belum ada laporan';}
 if(identity.role!=='admin'||identity.read_only||row.device_id===null){button.disabled=true;button.title=row.device_id===null?'Hubungkan GPS terlebih dahulu.':'Kontrol remote memerlukan administrator';}
 if(row.device_id===null){label.lastChild.textContent='Belum ada GPS';button.classList.remove('engine-enabled');button.setAttribute('aria-checked','false');}
 wrap.append(label,button);return wrap;
}
function renderVehicleCards(){
 const host=$('.vehicle-grid');if(!host)return;
 const focused=document.activeElement?.closest('[data-vehicle-card]');const focus=focused?{id:focused.dataset.vehicleCard,action:document.activeElement.dataset.rowAction}:null;
 host.replaceChildren();const sorted=records.slice().sort((a,b)=>String(a[{plate:'plate_number',name:'vehicle_name',code:'vehicle_code'}[vehicleSort]]||'').localeCompare(String(b[{plate:'plate_number',name:'vehicle_name',code:'vehicle_code'}[vehicleSort]]||'')));
 if(!sorted.length)host.append(el('p','Belum ada kendaraan untuk filter ini.','vehicle-empty'));
 for(const row of sorted){
  const card=el('article',undefined,'vehicle-card');card.dataset.vehicleCard=String(row.id);tableRows.set(card,{row,endpoint:'vehicles'});
  const photo=el('div',undefined,'vehicle-card-photo'),img=el('img');img.src='/api/v1/vehicles/'+row.id+'/photo?v='+encodeURIComponent(row.photo_updated_at||'default');img.alt='Foto '+(row.vehicle_name||row.plate_number);img.loading='lazy';img.onerror=()=>{img.onerror=null;img.src='/assets/vehicle-default.jpg';};photo.append(img);
  const status=el('span',vehicleStatus(row),'vehicle-status');status.dataset.state=row.movement_status||'UNKNOWN';photo.append(status);
  const codes=String(row.active_alert_codes||'').split(',').filter(Boolean);if(codes.length){const alert=el('button',codes.length+' alert aktif','vehicle-card-alert');alert.type='button';alert.title='Alert aktif pada kendaraan ini';alert.onclick=event=>{event.stopPropagation();location.hash='alert-center';};photo.append(alert);card.classList.add('vehicle-card-has-alert');}
  const plate=el('div',undefined,'vehicle-photo-caption');plate.append(el('strong',row.plate_number||'—'),el('span',currentSpeed(row)));photo.append(plate);
  const body=el('div',undefined,'vehicle-card-body');body.append(el('small',row.vehicle_code||'Kode belum diisi','vehicle-code'),el('h2',row.vehicle_name||row.plate_number));
  const metrics=el('div',undefined,'vehicle-card-metrics');
  const fuelAnomaly=codes.includes('FUEL_DROP');const fuelLow=row.fuel_percent!=null&&Number(row.fuel_percent)<=10;const fuelCondition=fuelAnomaly?'Anomali':fuelLow?'Hampir Habis':row.fuel_percent!=null?'Normal':'Belum ada data';
  for(const[label,value]of [['Volume BBM',row.fuel_liters==null?'Belum tersedia':num(row.fuel_liters,'L')],['Kapasitas tangki',row.tank_capacity_l==null?'Belum diisi':num(row.tank_capacity_l,'L')],['Kondisi fuel',fuelCondition],['GPS',row.device_identifier||'Belum terpasang']]){const metric=el('div');metric.append(el('small',label));const strong=el('strong',value);if(label==='Kondisi fuel')strong.classList.add('fuel-condition-'+(fuelAnomaly?'anomaly':fuelLow?'low':'normal'));metric.append(strong);if(label==='Volume BBM'&&(fuelAnomaly||fuelLow)){const icon=el('span','⚠','vehicle-fuel-warning-icon');icon.title=fuelAnomaly?'Fuel drop aktif':'Fuel hampir habis';icon.setAttribute('aria-label',icon.title);metric.append(icon);}metrics.append(metric);}
  const categoryLabels={CRASH:'Crash',TOW:'TOW',SOS:'SOS',OVER_SPEED:'Speeding'};for(const code of codes){if(!categoryLabels[code])continue;const badge=el('span',categoryLabels[code],'vehicle-category-alert vehicle-alert-'+code.toLowerCase());badge.title='Alert '+categoryLabels[code]+' aktif';photo.append(badge);}
  body.append(metrics);
  const location=el('p',row.latitude==null?'Belum ada posisi GPS':Number(row.latitude).toFixed(5)+', '+Number(row.longitude).toFixed(5),'vehicle-location');location.title='Koordinat GPS terakhir · '+date(row.gps_time);body.append(location);
  const level=el('div',undefined,'vehicle-fuel-level');level.append(el('span','Level tangki BBM'),el('strong',row.fuel_percent==null?'—':num(row.fuel_percent,'%')));
  const track=el('div',undefined,'vehicle-fuel-track'),fill=el('span');fill.style.width=(row.fuel_percent==null?0:Math.max(0,Math.min(100,Number(row.fuel_percent))))+'%';track.append(fill);body.append(level,track);
  if(row.fuel_received_at)body.append(el('small',date(row.fuel_received_at)+(row.fuel_fresh?'':' · data terakhir'),'vehicle-measured'));
  body.append(vehicleRemoteControl(row));
  const actions=el('div',undefined,'vehicle-card-actions');
  for(const[action,text]of [['inspect','Detail'],['history','History'],['edit','Edit'],['delete','Hapus']]){
   if(action==='edit'&&(identity.role==='viewer'||identity.read_only)||action==='delete'&&(identity.role!=='admin'||identity.read_only))continue;
   const button=el('button',text,action==='delete'?'inventory-delete':'');button.type='button';button.dataset.rowAction=action;actions.append(button);
  }
  body.append(actions);card.append(photo,body);host.append(card);
 }
 if(focus?.action)host.querySelector('[data-vehicle-card="'+focus.id+'"] [data-row-action="'+focus.action+'"]')?.focus();
}
function editVehicle(row){
 if(identity.role==='viewer'||identity.read_only)return;
 $('#editor-title').textContent=row?'Edit Kendaraan':'Tambah Kendaraan Baru';$('#form-error').textContent='';$('#save').hidden=false;$('#save').disabled=false;
 const fields=$('#fields');fields.replaceChildren();fields.classList.add('vehicle-editor-fields');
 const photo=el('div',undefined,'vehicle-photo-editor'),preview=el('img');preview.src=row?'/api/v1/vehicles/'+row.id+'/photo?v='+encodeURIComponent(row.photo_updated_at||'default'):'/assets/vehicle-default.jpg';preview.alt='Pratinjau foto kendaraan';photo.append(preview);
 const upload=el('input');upload.type='file';upload.accept='image/jpeg,image/png';upload.name='photo';const uploadLabel=el('label','Foto kendaraan');uploadLabel.append(upload,el('small','JPG atau PNG, maksimal 2 MB.'));photo.append(uploadLabel);fields.append(photo);
 let photoValue=null,reading=false;
 upload.onchange=()=>{
  const file=upload.files[0];photoValue=null;$('#form-error').textContent='';
  if(!file)return;
  if(!['image/jpeg','image/png'].includes(file.type)||file.size>2*1024*1024){$('#form-error').textContent='Pilih foto JPG atau PNG maksimal 2 MB.';upload.value='';return;}
  reading=true;$('#save').disabled=true;
  const reader=new FileReader();reader.onload=()=>{photoValue=String(reader.result).split(',')[1];preview.src=String(reader.result);reading=false;$('#save').disabled=false;};reader.onerror=()=>{reading=false;$('#save').disabled=false;$('#form-error').textContent='Foto gagal dibaca.';};reader.readAsDataURL(file);
 };
 for(const[name,title,required]of [['vehicle_code','Kode kendaraan',false],['vehicle_name','Nama kendaraan',true],['plate_number','Plat nomor',true],['tank_capacity_l','Kapasitas tangki (liter)',false],['fleet_code','Kode fleet (opsional)',false]]){
  const label=el('label',title),input=el('input');input.name=name;input.required=required;input.type=name==='tank_capacity_l'?'number':'text';
  if(input.type==='number'){input.min='.001';input.max='999999999';input.step='.001';}else input.maxLength=name==='vehicle_name'?100:name==='plate_number'?32:64;
  input.value=row?.[name]??'';label.append(input);fields.append(label);
 }
 fields.append(el('p','Kapasitas tangki adalah spesifikasi kendaraan. Konversi pembacaan BBM menggunakan kalibrasi sensor yang tersimpan.','vehicle-editor-note'));
 $('#edit-form').onsubmit=async event=>{
  event.preventDefault();if(reading)return;$('#save').disabled=true;$('#form-error').textContent='';
  const input=name=>fields.querySelector('[name='+name+']').value;
  const payload={vehicle_code:input('vehicle_code'),vehicle_name:input('vehicle_name'),plate_number:input('plate_number'),fleet_code:input('fleet_code'),tank_capacity_l:input('tank_capacity_l')===''?null:Number(input('tank_capacity_l')),row_version:row?.row_version||0};
  if(photoValue!==null)payload.photo_base64=photoValue;
  try{await api(row?'vehicles/'+row.id+'/inventory':'vehicles/inventory',{method:row?'PUT':'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});$('#editor').close();cursor=null;history=[];await load();const catalog=await api('vehicles?limit=200');vehicles=catalog.data||[];toast('Data kendaraan tersimpan.');}
  catch(e){$('#form-error').textContent=e.message;}finally{$('#save').disabled=false;}
 };
 $('#editor').showModal();
}
