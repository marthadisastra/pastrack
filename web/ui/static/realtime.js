'use strict';
let liveSource,liveRetry,liveLastInstance='',liveLastResources={},liveQueued=false,liveRefreshing=false;
const liveViews=['overview','positions','devices','events','messages','dash-vehicles','dash-gps','dash-activity','dash-fuel'];
function realtimeState(label,healthy){$('#realtime-status').textContent=label;$('#realtime-status').classList.toggle('realtime-offline',!healthy)}
async function liveFlush(){
 if(!liveQueued||liveRefreshing||document.hidden)return;
 if(view==='tracking'){liveQueued=false;if(typeof trackRealtimeChanged==='function')trackRealtimeChanged();return}
 if(!liveViews.includes(view))return;
 if(busy||(typeof analyticsBusy!=='undefined'&&analyticsBusy)||$('#detail').open)return;
 liveQueued=false;liveRefreshing=true;
 try{if(view.startsWith('dash-'))await refreshAnalytics(true);else await refresh()}finally{liveRefreshing=false;if(liveQueued)queueMicrotask(liveFlush)}
}
function liveRefresh(){liveQueued=true;queueMicrotask(liveFlush)}
function realtimeEvent(data){
 const reset=data.instance!==liveLastInstance,resources=data.resources||{};
 const changed=name=>reset||resources[name]!==liveLastResources[name];
 if(changed('master')||changed('places')||(reset&&liveLastInstance)||(changed('positions')&&(view==='tracking'||typeof applyLiveEntity!=='function')))liveRefresh();
 if(changed('notifications')&&typeof refreshNotifications==='function')refreshNotifications();
 liveLastInstance=data.instance;liveLastResources={...resources};realtimeState('Live · push',true);
}
function realtimeConnect(){
 if(document.hidden||liveSource)return;const source=new EventSource('/api/realtime');liveSource=source;realtimeState('Menghubungkan…',false);
 source.addEventListener('session',()=>{source.close();liveSource=null;location.href='/login'});
 source.addEventListener('positions',e=>{try{realtimeEvent(JSON.parse(e.data))}catch{realtimeState('Pembaruan belum dapat dibaca',false)}});
 source.addEventListener('entity',e=>{try{const change=JSON.parse(e.data);if(change.kind==='gps'&&view==='tracking'&&typeof trackRealtimeChanged==='function')trackRealtimeChanged();if(typeof applyLiveEntity==='function')applyLiveEntity(change).catch(()=>realtimeState('Pembaruan tertunda',false))}catch{realtimeState('Pembaruan belum dapat dibaca',false)}});
 source.onopen=()=>realtimeState('Live · push',true);
 source.onerror=()=>{realtimeState('Menghubungkan ulang…',false);if(source.readyState===EventSource.CLOSED){liveSource=null;clearTimeout(liveRetry);liveRetry=setTimeout(realtimeConnect,3000)}};
}
window.addEventListener('pastrack-refresh-finished',liveFlush);window.addEventListener('pastrack-data-changed',liveRefresh);$('#detail').addEventListener('close',liveFlush);
document.addEventListener('visibilitychange',()=>{if(document.hidden){liveSource?.close();liveSource=null;clearTimeout(liveRetry)}else{realtimeConnect();liveRefresh();if(typeof refreshNotifications==='function')refreshNotifications()}});
window.addEventListener('online',realtimeConnect);window.addEventListener('offline',()=>{liveSource?.close();liveSource=null;realtimeState('Offline',false)});realtimeConnect();
function renderRecentPositions(data){const list=data.slice(0,8);$('#overview-positions').innerHTML=list.length?table(['Waktu GPS (WIB)','Kendaraan / perangkat','Koordinat','Kecepatan','Diterima (WIB)'],list.map(p=>`<tr><td>${date(p.gps_time)}</td><td>${esc(p.plate_number||p.imei)}</td><td>${esc(p.latitude)}, ${esc(p.longitude)}</td><td>${num(p.speed_kmh)} km/h</td><td>${date(p.created_at)}</td></tr>`)):empty('Belum ada posisi pada periode ini')}

if('serviceWorker' in navigator)navigator.serviceWorker.addEventListener('message',e=>{if(e.data?.type==='pastrack-data-changed')liveRefresh()});
