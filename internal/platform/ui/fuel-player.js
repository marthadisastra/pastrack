'use strict';
const fePlayback={time:0,start:0,end:0,playing:false,engaged:false,rate:600,frame:null,last:null,positions:[],fuel:new Map(),marker:null,trail:null};
function feStopPlayback(){if(fePlayback.frame!=null)cancelAnimationFrame(fePlayback.frame);fePlayback.frame=null;fePlayback.playing=false;fePlayback.last=null;const b=document.querySelector('#fe-play');if(b)b.textContent='▶ Play';}
function fePlayerMarkup(){return `<section class="fe-playback" aria-label="Playback rute dan fuel"><div class="fe-play-controls"><button type="button" id="fe-play" class="fe-primary" disabled>▶ Play</button><button type="button" id="fe-restart" disabled>↺ Ulang</button><button type="button" id="fe-back" aria-label="Mundur satu menit" disabled>−1m</button><button type="button" id="fe-forward" aria-label="Maju satu menit" disabled>+1m</button><label>Kecepatan <select id="fe-rate"><option value="60">60×</option><option value="120">120×</option><option value="300">300×</option><option value="600" selected>600×</option><option value="1800">1800×</option></select></label><output id="fe-play-time">Belum ada data playback</output></div><input id="fe-timeline" type="range" min="0" max="10000" value="0" disabled aria-label="Waktu playback rute dan BBM"><div class="fe-play-readout" id="fe-play-readout" aria-live="off"></div></section>`;}
function feBindPlayer(){
 document.querySelector('#fe-play').onclick=()=>{if(fePlayback.playing){feStopPlayback();return;}feStartPlayback();};
 document.querySelector('#fe-restart').onclick=()=>{feStopPlayback();fePlayback.time=fePlayback.start;fePlayback.engaged=false;feUpdatePlayback();};
 document.querySelector('#fe-timeline').oninput=e=>feSeekPlayback(feAxisTime(fePlayback.axis.duration*Number(e.target.value)/10000));
 document.querySelector('#fe-back').onclick=()=>feSeekPlayback(fePlayback.time-60000);
 document.querySelector('#fe-forward').onclick=()=>feSeekPlayback(fePlayback.time+60000);
 document.querySelector('#fe-rate').onchange=e=>{fePlayback.rate=Number(e.target.value);};
}
// Compress unobserved gaps on the shared axis while retaining real timestamps.
function feBuildAxis(times){
 const segments=[];
 for(const time of [...new Set(times)].sort((a,b)=>a-b)){
  const last=segments.at(-1);
  if(last&&time-last.end<=300000){last.end=time;continue;}
  segments.push({start:time,end:time,offset:last?last.offset+last.end-last.start+60000:0});
 }
 const last=segments.at(-1);
 const duration=last?last.offset+last.end-last.start:0;
 return {segments,duration};
}
function feAxisOffset(time){
 const segments=fePlayback.axis.segments;
 const i=segments.findIndex(segment=>time<=segment.end);
 if(i===-1)return fePlayback.axis.duration;
 const segment=segments[i],previous=segments[i-1];
 if(time>=segment.start)return segment.offset+time-segment.start;
 if(!previous)return 0;
 return segment.offset-60000*(segment.start-time)/(segment.start-previous.end);
}
function feAxisTime(offset){
 const segments=fePlayback.axis.segments;
 const i=segments.findIndex(segment=>offset<=segment.offset+segment.end-segment.start);
 if(i===-1)return fePlayback.end;
 const segment=segments[i],previous=segments[i-1];
 if(offset>=segment.offset)return segment.start+offset-segment.offset;
 if(!previous)return segment.start;
 return segment.start-(segment.offset-offset)/60000*(segment.start-previous.end);
}
function fePreparePlayback(preserve=false){
 const wasPlaying=fePlayback.playing,oldTime=fePlayback.time,engaged=fePlayback.engaged;feStopPlayback();
 fePlayback.positions=(feData?.positions||[]).filter(p=>Number.isFinite(Date.parse(p.gps_time))&&p.latitude!=null&&p.longitude!=null&&Math.abs(Number(p.latitude))<=90&&Math.abs(Number(p.longitude))<=180&&!(Number(p.latitude)===0&&Number(p.longitude)===0)).slice().sort((a,b)=>Date.parse(a.gps_time)-Date.parse(b.gps_time));
 fePlayback.fuel=new Map();for(const reading of feData?.readings||[]){if(!Number.isFinite(Date.parse(reading.received_at)))continue;const key=String(reading.fuel_sensor_id);if(!fePlayback.fuel.has(key))fePlayback.fuel.set(key,[]);fePlayback.fuel.get(key).push(reading);}for(const list of fePlayback.fuel.values())list.sort((a,b)=>Date.parse(a.received_at)-Date.parse(b.received_at));
 const times=[...fePlayback.positions.map(p=>Date.parse(p.gps_time)),...[...fePlayback.fuel.values()].flatMap(list=>list.map(r=>Date.parse(r.received_at)))];
 for(const event of feData?.events||[]){const time=Date.parse(event.event_at);if(Number.isFinite(time))times.push(time);}
 fePlayback.axis=feBuildAxis(times);fePlayback.start=fePlayback.axis.segments[0]?.start??0;fePlayback.end=fePlayback.axis.segments.at(-1)?.end??0;fePlayback.time=preserve?Math.max(fePlayback.start,Math.min(fePlayback.end,oldTime)):fePlayback.start;fePlayback.engaged=preserve&&engaged;
 const ready=fePlayback.end>fePlayback.start;for(const id of ['fe-play','fe-restart','fe-back','fe-forward','fe-timeline']){const n=document.querySelector('#'+id);if(n)n.disabled=!ready;}
 feRenderChart();
 if(!times.length){mapTankOverlay(feMap,feData,0);document.querySelector('#fe-play-time').textContent='Belum ada data playback';document.querySelector('#fe-play-readout').textContent='Pilih periode yang memiliki riwayat GPS atau pembacaan BBM.';return;}
 feUpdatePlayback();if(preserve&&wasPlaying&&ready)feStartPlayback();
}
// Binary lookup: interpolate only across continuous source samples; never bridge a data gap.
function feSampleAt(list,time,key,fields,sourceKey,valid=()=>true){
 let low=0,high=list.length;while(low<high){const mid=(low+high)>>1;if(Date.parse(list[mid][key])<=time)low=mid+1;else high=mid;}
 if(low===0)return null;const a=list[low-1],at=Date.parse(a[key]);if(time-at>300000||!valid(a))return null;
 const b=list[low],result={...a};if(!b||!valid(b)||String(a[sourceKey])!==String(b[sourceKey]))return result;
 const bt=Date.parse(b[key]);if(bt<=at||bt-at>300000)return result;const f=(time-at)/(bt-at);
 for(const field of fields){if(a[field]!=null&&b[field]!=null)result[field]=Number(a[field])+(Number(b[field])-Number(a[field]))*f;}
 return result;
}
function feSeekPlayback(time){feStopPlayback();fePlayback.engaged=true;fePlayback.time=Math.max(fePlayback.start,Math.min(fePlayback.end,time));feUpdatePlayback();}
function feStartPlayback(){
 if(fePlayback.end<=fePlayback.start)return;if(fePlayback.playing)return;if(fePlayback.time>=fePlayback.end)fePlayback.time=fePlayback.start;
 fePlayback.engaged=true;fePlayback.playing=true;fePlayback.last=null;document.querySelector('#fe-play').textContent='❚❚ Pause';
 const tick=now=>{if(routeKey!=='fuel-events'||!document.querySelector('#fe-play')){feStopPlayback();return;}if(fePlayback.last!=null)fePlayback.time=feAxisTime(Math.min(fePlayback.axis.duration,feAxisOffset(fePlayback.time)+Math.min(250,now-fePlayback.last)*fePlayback.rate));fePlayback.last=now;feUpdatePlayback();if(fePlayback.time<fePlayback.end&&fePlayback.playing)fePlayback.frame=requestAnimationFrame(tick);else feStopPlayback();};
 fePlayback.frame=requestAnimationFrame(tick);
}
function feUpdatePlayback(){
 if(!feData||!document.querySelector('#fe-play-time'))return;
 const time=fePlayback.time,position=feSampleAt(fePlayback.positions,time,'gps_time',['latitude','longitude','speed_kmh'],'device_id');
 document.querySelector('#fe-play-time').textContent=date(new Date(time).toISOString())+(time>=fePlayback.end?' · Selesai':'');
 document.querySelector('#fe-timeline').value=fePlayback.end>fePlayback.start?String(Math.round(feAxisOffset(time)/fePlayback.axis.duration*10000)):'0';
 const fuel=[];for(const [id,list]of fePlayback.fuel){const sample=feSampleAt(list,time,'received_at',['fuel_liters','fuel_percent'],'calibration_id',feValid);fuel.push('Sensor '+id+': '+(sample?num(sample.fuel_liters,'L')+' ('+num(sample.fuel_percent,'%')+')':'jeda / belum ada data valid'));}
 const activeEvent=feData.events.find(e=>time>=Date.parse(e.start_at||e.event_at)&&time<=Date.parse(e.end_at||e.event_at)+60000);
 document.querySelector('#fe-play-readout').textContent=(position?'↗ '+num(position.speed_kmh,'km/h')+' · '+Number(position.latitude).toFixed(5)+', '+Number(position.longitude).toFixed(5):'GPS: jeda / belum ada posisi')+' | '+(fuel.join(' · ')||'BBM: belum ada pembacaan')+(activeEvent?' | ⚠ '+'EVT-'+activeEvent.id+' · '+num(activeEvent.delta_l,'L'):'');
 document.querySelector('#fe-play-readout').classList.toggle('fe-play-alert',!!activeEvent);
 // The base route stays visible; the blue trail and marker share the graph's exact clock.
 if(fePlayback.positions.length&&feMap&&feLayers&&typeof L.circleMarker==='function'){
  if(!fePlayback.marker){fePlayback.marker=mapVehicleMarker([0,0]).addTo(feLayers);fePlayback.marker.bindTooltip(el('span','Kendaraan playback'));}
  if(position){fePlayback.marker.setLatLng([Number(position.latitude),Number(position.longitude)]);mapUpdateVehicle(fePlayback.marker,mapHistoryIgnition(feData,time,position.device_id),mapAnomalyAt(feData,time),true);}else mapUpdateVehicle(fePlayback.marker,{},false,false);
  const segments=[];let segment=[];for(const p of fePlayback.positions){if(Date.parse(p.gps_time)>time)break;const previous=segment.at(-1);if(previous&&(Date.parse(p.gps_time)-Date.parse(previous.gps_time)>300000||p.device_id!==previous.device_id)){if(segment.length>1)segments.push(segment.map(s=>[Number(s.latitude),Number(s.longitude)]));segment=[];}segment.push(p);}if(position)segment.push(position);if(segment.length>1)segments.push(segment.map(s=>[Number(s.latitude),Number(s.longitude)]));
  if(!fePlayback.trail&&typeof L.polyline==='function')fePlayback.trail=L.polyline([],{color:'#2563eb',weight:5,opacity:.9}).addTo(feLayers);fePlayback.trail?.setLatLngs(segments);
 }
 mapTankOverlay(feMap,feData,time);
 const x=60+875*feAxisOffset(time)/(fePlayback.axis.duration||1);
 const cursor=document.querySelector('#fe-chart-cursor');if(cursor){cursor.setAttribute('x1',x);cursor.setAttribute('x2',x);}
 const reveal=document.querySelector('#fe-chart-reveal');if(reveal)reveal.setAttribute('width',fePlayback.engaged?Math.max(0,x-60):875);
 document.querySelectorAll('.fe-event').forEach(card=>{const event=feData.events.find(e=>String(e.id)===card.dataset.eventId);card.classList.toggle('fe-event-playing',!!event&&time>=Date.parse(event.start_at||event.event_at)&&time<=Date.parse(event.end_at||event.event_at)+60000);});
}
