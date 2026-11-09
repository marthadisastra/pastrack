'use strict';
let trackPlaybackTime=0,trackPlaybackStart=0,trackPlaybackEnd=0,trackPlaybackWall=null,trackPlaybackLayer,trackPlaybackMarkers=new Map(),trackPlaybackLastLabels=0;
const trackClockFormat=new Intl.DateTimeFormat('id-ID',{timeZone:'Asia/Jakarta',day:'2-digit',month:'short',year:'numeric',hour:'2-digit',minute:'2-digit',second:'2-digit',hour12:false});
// Upper-bound lookup keeps all vehicles on one GPS clock, regardless of sampling frequency.
function trackStateAt(route,time){
 const points=route.points;
 if(!points.length){const p=route.allowLastPosition===false?null:trackLastPoint(route);return p?{...p,visible:true,status:'outside',speed:null,heading:0,sampleTime:Date.parse(p.gps_time)}:{visible:false,status:'empty'}}
 let lo=0,hi=points.length;
 while(lo<hi){const mid=(lo+hi)>>>1;if(points[mid].t<=time)lo=mid+1;else hi=mid}
 if(lo===0)return {visible:false,status:'pending'};
 const index=lo-1,p=points[index],next=points[index+1];
 const base={lat:p.lat,lon:p.lon,visible:true,index,sampleTime:p.t,heading:Number(p.heading_deg)||0};
 if(!next)return {...base,status:'ended',speed:null};
 const elapsed=next.t-p.t;
 if(next.gap||elapsed<=0||elapsed>300000)return {...base,status:'gap',speed:null};
 const ratio=Math.max(0,Math.min(1,(time-p.t)/elapsed));
 const a=p.speed_kmh==null?NaN:Number(p.speed_kmh),b=next.speed_kmh==null?NaN:Number(next.speed_kmh);
 const speed=Number.isFinite(a)&&Number.isFinite(b)?a+(b-a)*ratio:trackDistance(p,next)/(elapsed/3600000);
 const delta=((Number(next.heading_deg||0)-base.heading+540)%360)-180;
 const lon=p.lon+(((next.lon-p.lon+540)%360)-180)*ratio;
 return {...base,lat:p.lat+(next.lat-p.lat)*ratio,lon:((lon+540)%360)-180,speed,heading:(base.heading+delta*ratio+360)%360,status:speed>3?'moving':'stopped'};
}
function trackPlaybackStatus(state){return {moving:'Bergerak',stopped:'Berhenti teramati',gap:'Jeda GPS',ended:'GPS terakhir',pending:'Belum ada sampel',outside:'Di luar periode',empty:'Tanpa posisi'}[state.status]||'Tanpa posisi'}
function trackCarIcon(color){return vehicleMapIcon(color)}
function trackPlaybackInit(start,end){
 trackStop();trackPlaybackStart=Date.parse(start);trackPlaybackEnd=Date.parse(end);trackPlaybackTime=trackPlaybackStart;trackPlaybackLastLabels=0;
 if(trackPlaybackLayer)trackPlaybackLayer.remove();trackPlaybackLayer=L.layerGroup().addTo(trackMap);trackPlaybackMarkers=new Map();
 for(const route of trackRoutes){const marker=L.marker([0,0],{icon:trackCarIcon(route.color),keyboard:true,zIndexOffset:1000,title:route.vehicle.plate_number||route.vehicle.vehicle_name,alt:route.vehicle.plate_number||route.vehicle.vehicle_name}).bindTooltip('',{permanent:true,direction:'top',offset:[0,-16],className:'track-car-label'});trackPlaybackMarkers.set(String(route.vehicle.id),{marker,visible:false,label:''})}
 const firstFix=trackRoutes.flatMap(r=>r.points.length?[r.points[0].t]:[]).filter(Number.isFinite);
 if(firstFix.length)trackPlaybackTime=Math.max(trackPlaybackStart,Math.min(...firstFix));
 const span=trackPlaybackEnd-trackPlaybackStart;
 $('#tracking-rate').value=span>7*86400000?'21600':span>86400000?'3600':'600';
 const playable=trackRoutes.some(r=>r.points.length>1)&&span>0;
 $('#tracking-play').disabled=!playable;$('#tracking-play').title='Putar seluruh kendaraan pada waktu GPS yang sama';$('#tracking-play').setAttribute('aria-pressed','false');
 $('#tracking-rate').disabled=!playable;$('#tracking-reset').disabled=!trackData.length;
 $('#tracking-slider').max='10000';$('#tracking-slider').step='1';$('#tracking-slider').disabled=!trackData.length;
 $('#tracking-playback-hint').textContent='Simulasi riwayat GPS · seluruh kendaraan mengikuti waktu yang sama. Jeda >5 menit menahan posisi terakhir; kendaraan tanpa sampel belum ditampilkan.';
 trackRenderPlayback(true);
}
function trackRenderPlayback(force=false){
 const labels=force||performance.now()-trackPlaybackLastLabels>=180;
 const counts={moving:0,stopped:0,unknown:0};
 for(const route of trackRoutes){
  const state=trackStateAt(route,trackPlaybackTime),entry=trackPlaybackMarkers.get(String(route.vehicle.id));
  if(state.status==='moving')counts.moving++;else if(state.status==='stopped')counts.stopped++;else counts.unknown++;
  if(entry){
   if(state.visible){
    entry.marker.setLatLng([state.lat,state.lon]);if(!entry.visible){entry.marker.addTo(trackPlaybackLayer);entry.visible=true}
    const uncertain=!['moving','stopped'].includes(state.status);entry.marker.setOpacity(uncertain?.5:1);
    const icon=entry.marker.getElement();if(icon){icon.dataset.playbackVehicle=String(route.vehicle.id);icon.classList.toggle('is-uncertain',uncertain);const direction=icon.querySelector('.track-car-direction');if(direction)direction.style.transform=`rotate(${state.heading||0}deg)`}
    if(labels){const label=esc(route.vehicle.plate_number||route.vehicle.vehicle_name)+' · '+trackPlaybackStatus(state)+(state.speed==null?'':` · ${num(state.speed)} km/h`);if(label!==entry.label){entry.marker.setTooltipContent(label);entry.label=label}}
   }else if(entry.visible){entry.marker.remove();entry.visible=false}
  }
  if(labels){const element=document.querySelector(`[data-playback-vehicle="${Number(route.vehicle.id)}"]`);if(element){element.textContent=trackPlaybackStatus(state)+(state.speed==null?'':` · ${num(state.speed)} km/h`);element.dataset.status=state.status}}
 }
 if(labels){
  $('#tracking-clock').textContent=trackClockFormat.format(new Date(trackPlaybackTime))+' WIB';
  $('#tracking-current').textContent=`${counts.moving} bergerak · ${counts.stopped} berhenti · ${counts.unknown} tanpa data pada waktu ini`;
  const span=trackPlaybackEnd-trackPlaybackStart;$('#tracking-slider').value=span>0?Math.round((trackPlaybackTime-trackPlaybackStart)/span*10000):0;
  $('#tracking-slider').setAttribute('aria-valuetext',trackClockFormat.format(new Date(trackPlaybackTime))+' WIB');trackPlaybackLastLabels=performance.now();
 }
}
function trackSeek(time){trackPlaybackTime=Math.max(trackPlaybackStart,Math.min(trackPlaybackEnd,time));trackRenderPlayback(true)}
function trackPlaybackFrame(now){
 if(view!=='tracking'||document.hidden){trackStop();return}
 if(trackPlaybackWall!=null){const elapsed=Math.max(0,Math.min(250,now-trackPlaybackWall));trackPlaybackTime=Math.min(trackPlaybackEnd,trackPlaybackTime+elapsed*Number($('#tracking-rate').value))}
 trackPlaybackWall=now;trackRenderPlayback();
 if(trackPlaybackTime>=trackPlaybackEnd){trackRenderPlayback(true);trackStop();return}
 trackTimer=requestAnimationFrame(trackPlaybackFrame);
}
