'use strict';
async function automaticFuelCalibration(row){
 const panel=el('section',undefined,'auto-calibration');
 const identity=el('div',undefined,'auto-calibration-identity'),stateLabel=el('strong','Memuat sensor…'),reading=el('div',undefined,'auto-calibration-reading'),steps=el('ol',undefined,'auto-calibration-steps');
 for(const text of ['1 · Empty','2 · Isi penuh','3 · Simpan'])steps.append(el('li',text));
 const explanation=el('p'),error=el('p',undefined,'stitch-error');error.setAttribute('role','alert');explanation.setAttribute('role','status');
 const actions=el('div',undefined,'stitch-setup-actions'),start=el('button','Start','stitch-setup-primary'),confirm=el('button','Tangki sudah penuh','stitch-setup-primary'),cancel=el('button','Hentikan');
 for(const button of [start,confirm,cancel])button.type='button';
 start.disabled=true;confirm.hidden=true;cancel.hidden=true;
 actions.append(cancel,start,confirm);panel.append(identity,steps,stateLabel,reading,explanation,el('small','Stabil ≥10 detik · minimal 3 laporan berbeda · variasi maksimal 1 raw. Mulai dengan tangki kosong.'),error,actions);
 inspect('Kalibrasi Sensor BBM',panel);
 let data=null,busy=false,timer=null,closed=false;
 const running=()=>data&&['empty','fill','full'].includes(data.state);
 const cleanup=()=>{closed=true;clearTimeout(timer);};$('#inspector').addEventListener('close',cleanup,{once:true});
 function render(){
  identity.replaceChildren(el('strong',data.sensor_name||row.sensor_name||row.identifier||'Sensor BBM'),el('span','MAC '+(data.sensor_mac||row.sensor_mac||'—')),el('span','GPS IMEI '+(data.gps_imei||row.gps_imei||'—')),el('span',(data.plate_number||'Kendaraan belum tersedia')+' · Kapasitas '+(data.capacity_l?num(data.capacity_l,'L'):'belum diisi')));
  const labels={idle:'Siap dikalibrasi',empty:'Membaca Empty',fill:'Empty tersimpan',full:'Membaca Full',complete:'Kalibrasi terakhir tersimpan',failed:'Kalibrasi dihentikan',cancelled:'Kalibrasi dihentikan'};
  stateLabel.textContent=labels[data.state]||data.state;
  const stable=data.stability||{};
  reading.replaceChildren(el('strong',stable.raw==null?'Menunggu data sensor':String(stable.raw)),el('small',(data.quantity==='volume_raw'?'Volume raw':'Fuel height raw')+' · '+Math.floor(stable.stable_seconds||0)+'/10 detik · '+(stable.sample_count||0)+' laporan'));
  if(data.empty_raw!=null)reading.append(el('span','Empty '+data.empty_raw+(data.full_raw==null?'':' · Full '+data.full_raw)));
  explanation.textContent=data.state==='empty'?'Pastikan tangki kosong. Empty direkam otomatis setelah nilai stabil.':data.state==='fill'?'Isi tangki sampai penuh, lalu tekan “Tangki sudah penuh”.':data.state==='full'?'Menunggu laporan Full yang stabil. Kalibrasi tersimpan otomatis setelah valid.':data.state==='failed'?(data.error||'Mulai ulang dengan sensor yang sesuai.'):data.source_error||'Tekan Start saat tangki benar-benar kosong.';
  error.textContent=data.source_error||data.error||'';
  steps.children[0].classList.toggle('done',data.empty_raw!=null);steps.children[1].classList.toggle('done',['full','complete'].includes(data.state));steps.children[2].classList.toggle('done',data.state==='complete');
  start.hidden=!!running();start.disabled=busy||!!data.source_error;start.textContent=['failed','cancelled','complete'].includes(data.state)?'Start ulang':'Start';
  confirm.hidden=data.state!=='fill';confirm.disabled=busy;cancel.hidden=!running();cancel.disabled=busy;
 }
 async function update(action){
  if(busy||closed)return;clearTimeout(timer);busy=true;start.disabled=confirm.disabled=cancel.disabled=true;error.textContent='';
  try{
   const result=await post('sensors/'+row.id+'/calibration-session',{action,session_id:data?.session_id||0});
   if(closed)return;data=result.data;render();
   if(action!=='start'&&data.state==='complete'){
    cleanup();$('#inspector').close();await load();toast('Kalibrasi berhasil. Konfigurasi sensor sudah diperbarui.');return;
   }
  }catch(e){if(!closed)error.textContent=e.message;}
  finally{busy=false;if(!closed){start.disabled=!!data?.source_error;confirm.disabled=cancel.disabled=false;if(running())timer=setTimeout(()=>update('progress'),2000);}}
 }
 start.onclick=()=>update('start');confirm.onclick=()=>update('confirm_full');cancel.onclick=()=>update('cancel');
 try{
  const result=await api('sensors/'+row.id+'/calibration-session');if(closed)return;data=result.data;render();if(running())timer=setTimeout(()=>update('progress'),1000);
 }catch(e){error.textContent=e.message;stateLabel.textContent='Data kalibrasi gagal dimuat';}
}
