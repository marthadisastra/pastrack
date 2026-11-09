function mapVehicleIcon(row={},anomaly=false){
 const state=row.ignition_status||'UNKNOWN';
 const body='<div class="map-vehicle '+(state==='ON'?'map-vehicle-on':state==='OFF'?'map-vehicle-off':'map-vehicle-unknown')+' '+(anomaly?'map-anomaly-pulse':'')+'" title="'+(state==='ON'?'Ignition ON':state==='OFF'?'Ignition OFF':'Ignition belum tersedia')+'"><svg viewBox="0 0 32 32" aria-hidden="true"><path d="M8 8h16l4 9v10h-4v-4H8v4H4V17zm2 3-3 7h18l-3-7z" fill="currentColor"/><circle cx="9" cy="21" r="2" fill="white"/><circle cx="23" cy="21" r="2" fill="white"/></svg></div>';
 return L.divIcon({html:body,className:'map-vehicle-container',iconSize:[36,36],iconAnchor:[18,18]});
}
function mapVehicleMarker(coords,row={},anomaly=false){
 if(typeof L.marker!=='function'||typeof L.divIcon!=='function')return L.circleMarker(coords,{radius:9,color:'#fff',fillColor:'#64748b',fillOpacity:1});
 return L.marker(coords,{icon:mapVehicleIcon(row,anomaly)});
}
function mapUpdateVehicle(marker,row={},anomaly=false,visible=true){
 if(marker?.setIcon)marker.setIcon(mapVehicleIcon(row,anomaly));
 marker?.setOpacity?.(visible?1:0);
}
function mapHistoryIgnition(data,time,device){
 return (data?.ignitions||[]).filter(r=>String(r.device_id)===String(device)&&Date.parse(r.event_at)<=time).sort((a,b)=>Date.parse(b.event_at)-Date.parse(a.event_at))[0]||{};
}
function mapAnomalyAt(data,time){
 return (data?.events||[]).some(e=>e.event_type==='fuel_drop'&&time>=Date.parse(e.start_at||e.event_at)&&time<=Date.parse(e.end_at||e.event_at)+60000);
}
function mapEventMarker(event){
 if(typeof L.marker!=='function'||typeof L.divIcon!=='function')return L.circleMarker([+event.latitude,+event.longitude],{radius:9,color:'#dc2626'});
 return L.marker([+event.latitude,+event.longitude],{icon:L.divIcon({html:'<span class="map-event-dot '+(event.event_type==='fuel_drop'?'map-anomaly-pulse':'map-refuel-dot')+'"></span>',className:'map-event-container',iconSize:[20,20],iconAnchor:[10,10]})});
}
const mapTankSources=new WeakMap();
function mapTankOverlay(map,data,time){
 if(!map?.getContainer)return;
 const host=map.getContainer();
 let card=host.querySelector('.map-tank-overlay');
 if(!card){
  card=document.createElement('section');card.className='map-tank-overlay';card.setAttribute('aria-label','Kondisi tangki pada waktu playback');
  card.innerHTML='<strong>Tangki BBM</strong><div class="map-tank-visual"><div class="map-tank-fill"></div><div class="map-tank-value"></div></div><small class="map-tank-meta"></small>';
  host.append(card);L.DomEvent?.disableClickPropagation(card);L.DomEvent?.disableScrollPropagation(card);
 }
 const rows=data?.readings||[];
 let low=0,high=rows.length;
 while(low<high){const mid=(low+high)>>1;if(Date.parse(rows[mid].received_at)<=time)low=mid+1;else high=mid;}
 const latest=rows[low-1];
 if(data&&!mapTankSources.has(data))mapTankSources.set(data,new Set(rows.map(r=>String(r.fuel_sensor_id))).size);
 const ready=mapTankSources.get(data)<=1&&latest&&time-Date.parse(latest.received_at)<=300000&&feValid(latest);
 const percent=ready&&latest.fuel_percent!=null?Math.max(0,Math.min(100,Number(latest.fuel_percent))):null;
 card.classList.toggle('map-tank-empty',!ready);
 card.querySelector('.map-tank-fill').style.height=(percent??0)+'%';
 card.querySelector('.map-tank-value').textContent=ready?num(latest.fuel_liters,'L')+(percent!=null?' · '+num(percent,'%'):''):'BBM belum tersedia';
 const capacity=latest?.capacity_l;
 card.querySelector('.map-tank-meta').textContent=ready?'Kapasitas '+(capacity==null?'belum tersedia':num(capacity,'L'))+' · '+date(latest.received_at):'Belum ada pembacaan liter valid pada waktu ini';
}
