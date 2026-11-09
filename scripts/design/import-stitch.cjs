// Import trusted local exports. No CDN runtime and no demo telemetry in app templates.
const fs=require('fs'),path=require('path'),vm=require('vm');
const {JSDOM}=require('../../web/node_modules/jsdom');
const root=path.resolve(__dirname,'../..'),out=path.join(root,'internal/platform/ui');
const sources=fs.readdirSync(path.join(root,'Desain')).filter(n=>fs.existsSync(path.join(root,'Desain',n,'code.html')));
const manifest={};let shell,config;const styles=new Set();
const mapPages=new Set(['fleet_overview','vehicles','vehicles_telematics_overview_b_9421_uei','vehicles_trip_history_b_9421_uei','live_tracking_maps_telemetri_real_time','geofences_management','rute_playback_riwayat_telemetri_b_9421_uei']);
const keepNumbers=/^(?:\d+(?:h|x|%| Days?| Hari| Bulan)|(?:Last |Hari Ini|Today|7 Days|24h|1h|6h|28 Hari|Rentang|\+|−|1x|2x|5x|10x|20x)|(?:GPS|BLE|Queclink|CFS300B|GV30CEU|2FA|4G|LTE|HDOP|RSRP|DO1|SIM|CSV|PDF|GPX|KML|OTA|FOTA|GSM|IoT|N\/A|NOT_INSTALLED)$)/i;
function nearestMetricLabel(el){for(let p=el.parentElement,level=0;p&&level<4;p=p.parentElement,level++){const leaves=[...p.querySelectorAll('span,p,div')].filter(n=>!n.children.length&&n!==el&&!n.classList.contains('material-symbols-outlined'));const candidate=leaves.find(n=>/total fleet|moving|stopped|offline|active alerts|total.*kendaraan|jumlah.*perangkat|online|total.*sensor/i.test(n.textContent));if(candidate)return candidate.textContent.trim();}return '';}
function sanitize(main,name){
 // Transport labels reflect the mechanism actually implemented, not the demo.
 for(const e of main.querySelectorAll('*')){if(!e.children.length&&e.textContent.includes('Live MQTT'))e.textContent=e.textContent.replace('Live MQTT','Live SSE').replace(' ultrasonic sensors',' sensors');}

 if(name==='geofences_management'){const heading=main.querySelector('h3');let card=heading?.parentElement;while(card&&card!==main&&!card.className.includes('rounded'))card=card.parentElement;if(card?.parentElement){card.parentElement.dataset.geofenceList='';card.parentElement.dataset.cardClass=card.className;}}
 for(const e of main.querySelectorAll('*')){for(const attribute of [...e.attributes]){if(attribute.name.startsWith('on')||attribute.value.startsWith('javascript:'))e.removeAttribute(attribute.name);}if(e.className?.baseVal===undefined&&typeof e.className==='string'&&e.className.includes('url('))e.className=e.className.split(/\s+/).filter(c=>!c.includes('url(')).join(' ');}
 for(const e of main.querySelectorAll('script,iframe,img'))e.remove();
 // Offline charts are replaced by API-rendered plots in their exact original slots.
 const svgs=[...main.querySelectorAll('svg')];
 let first=true;
 for(const svg of svgs){if(!svg.isConnected)continue;let mapParent=null;
  if(first&&mapPages.has(name)){for(let p=svg.parentElement;p&&p!==main;p=p.parentElement){const c=p.getAttribute('class')||'';if(c.includes('relative')&&/h-\[|h-\d|h-screen/.test(c)){mapParent=p;break;}}}first=false;
  if(mapParent){mapParent.replaceChildren();mapParent.removeAttribute('style');mapParent.dataset.map='';mapParent.classList.add('stitch-map');continue;}
  const parent=svg.parentElement;const slot=main.ownerDocument.createElement('div');slot.className='stitch-chart-slot';slot.dataset.chart='';slot.setAttribute('aria-label','Grafik berdasarkan data API');slot.style.minHeight=(svg.classList.contains('h-24')||svg.classList.contains('h-32'))?'96px':'180px';svg.replaceWith(slot);
 }
 for(const table of main.querySelectorAll('table')){table.dataset.liveTable='';const body=table.querySelector('tbody')||table.appendChild(main.ownerDocument.createElement('tbody'));body.replaceChildren();}
 // Runtime owns all checkboxes/field state. Exported values are examples.
 for(const e of main.querySelectorAll('input,textarea')){e.value='';e.removeAttribute('value');if(e.type==='checkbox')e.checked=false;}
 for(const e of main.querySelectorAll('[style]')){if(/background-image|url\(/i.test(e.getAttribute('style')))e.style.removeProperty('background-image');}
 const walker=main.ownerDocument.createTreeWalker(main,4);const nodes=[];while(walker.nextNode())nodes.push(walker.currentNode);
 for(const node of nodes){const el=node.parentElement;if(!el||!el.isConnected||el.closest('style,script,th,thead')||el.classList.contains('material-symbols-outlined'))continue;
  const value=node.textContent.trim();if(!value)continue;
  if(/\[Demo Mode/.test(value)){node.textContent='Live Workspace';el.dataset.copy='workspace';continue;}
  if(/Real-time operational visibility across/.test(value)){node.textContent='Real-time operational visibility across your registered fleet vehicles';el.dataset.copy='fleet-description';continue;}
  if(/Real-time breakdown of/.test(value)){node.textContent='Real-time breakdown of registered units';el.dataset.copy='fleet-breakdown';continue;}
  if(/100% telemetry enrolled/.test(value)){node.textContent='Telemetry coverage: N/A';continue;}
  if(/Avg speed/.test(value)){node.textContent='Avg speed: N/A';el.dataset.copy='average-speed';continue;}
  if(/^Offline \(/.test(value)){node.textContent='Offline';continue;}
  if(/^Showing /.test(value)){node.textContent='Showing ';continue;}
  if(/Showing|vehicles.*Telemetry refresh/.test(value)&&el.closest('table')===null){continue;}
  const enumValue=/^(?:LIVE )?(?:MOVING|STOPPED|CONNECTED|DISCONNECTED|ONLINE|OFFLINE|READY|PAIRED|VALID|STABLE|ACTIVE|NORMAL|EXCELLENT)$/i;
  if(enumValue.test(value)&&!el.closest('button,select,th,thead')&&el.tagName==='SPAN'&&/px-|bg-|rounded/.test(el.className+' '+el.parentElement.className)){el.dataset.liveValue='status';node.textContent='N/A';continue;}

  if(/Ignition|ignition|Kontak|Harness/.test(value)&&/^N\/A|Unwired|Not Installed|NOT_INSTALLED/.test(value)){node.textContent='NOT_INSTALLED';continue;}
  const numeric=/\d/.test(value.replace(/CFS300B|GV30CEU|2FA|4G|DO1|BLE\s*5\.[0-9]/g,''));
  const person=/Budi Santoso|Santoso|Pratama|Saputra|Wijaya|Hidayat|Agus Setiawan|Ahmad|Sutrisno|Joko|Slamet|Asep|Dedi|Rizky|Wahyu|Hendra|Siti|Dimas|Fajar|Rudi|Yanto|Eko|Bambang|Adi Nugroho/i.test(value);
  const email=/[\w.+-]+@[\w.-]+\.[a-z]{2,}/i.test(value);
  const demoLocation=/Pool Pusat Cakung|Rest Area Tol Cipali|Warehouse Unilever|Zona Terlarang \/ Pantura|Terminal Petikemas|Tol Cipali KM|Java-Sumatra Trans|Subang Rest Area|Hub Logistik Cikarang|Java Core|Trans-Java Tollway/i.test(value);
  if(person||email||demoLocation){node.textContent='N/A';el.dataset.liveValue='text';continue;}
  if(numeric&&!keepNumbers.test(value)){
   if(/\b(?:UTC|GT\w{3}|mm|cm|Hz|pF|km\/h|L\/100|mV|VDC|CFS300B|GV30CEU)\b/.test(value)&&value.length<34&&!/[=:]|\d{3}|\d\.\d/.test(value))continue;
   if(el.closest('button')){
    if(/^[\d.,]+$/.test(value)){const full=el.closest('button').textContent;el.dataset.liveValue=/all vehicles/i.test(full)?'total fleet':/moving/i.test(full)?'moving':/stopped/i.test(full)?'stopped':/offline/i.test(full)?'offline':/active alerts/i.test(full)?'active alerts':'value';node.textContent='N/A';continue;}
    node.textContent=value.replace(/\([^)]*\d[^)]*\)/g,'').replace(/\d[\d.,]*(?:\s*%|\s*Logs?|\s*Units?|\s*Events?|\s*Stops?)?/g,'').trim()||'N/A';continue;}
   const metric=/^[\d\s.,%+−-]+(?:L|km\/h|s|m|min|Menit|Kasus|Unit)?$/i.test(value);
   const plate=/^[A-Z]{1,2}\s*\d{3,4}\s*[A-Z]{1,3}/.test(value);
   const label=metric?nearestMetricLabel(el):'';el.dataset.liveValue=plate?'vehicle_plate':(label||'value');el.dataset.originalValue=value;node.textContent='N/A';
  }
  if(/Connected \(Valid\)|Sensor Connected|Normal \(Valid\)|Authenticated|Authenticated Secure|Excellent|Signal Stable|Live Feed OK/i.test(value)){node.textContent='N/A';el.dataset.liveValue='status';}
  if(/IGNITION ON|IGNITION OFF|Engine ON|Engine OFF|Idling|Idle \//i.test(value)&&!el.closest('th')){node.textContent='Ignition: NOT_INSTALLED';}
 }
 // Numeric CSS graphs/bars are examples too. Keep tracks and dimensions, clear fills.
 for(const e of main.querySelectorAll('[style]')){if(/width:\s*[\d.]+%/.test(e.getAttribute('style'))){e.dataset.level='';e.style.width='0%';}}
 for(const e of main.querySelectorAll('[data-original-value]'))e.removeAttribute('data-original-value');
 // Remove mock incident/vehicle cards with a rendered plate, keeping their original list layout.
 // Remaining dynamic slots are replaced with API values at runtime.
}
for(const name of sources){const dom=new JSDOM(fs.readFileSync(path.join(root,'Desain',name,'code.html'),'utf8'));const doc=dom.window.document;
 for(const style of doc.querySelectorAll('head style'))styles.add(style.textContent.replace(/@import\s+url\([^)]*\);?/g,''));
 if(name==='fleet_overview'){
  const context={tailwind:{}};vm.runInNewContext(doc.querySelector('#tailwind-config').textContent,context);config=context.tailwind.config;
  const aside=doc.querySelector('aside').cloneNode(true),header=doc.querySelector('header').cloneNode(true);
  for(const a of aside.querySelectorAll('a[data-path]')){a.href='#'+a.dataset.path;a.classList.add('stitch-nav-link','text-inverse-on-surface/80','hover:text-inverse-on-surface','hover:bg-surface-container-high/20','font-body-sm','text-body-sm');}
  header.id='stitch-header';header.querySelector('input').id='global-query';
  for(const span of header.querySelectorAll('span')){if(span.textContent.trim()==='Budi Santoso'){span.textContent='';span.id='account';}if(span.textContent.includes('Fleet Ops Head')){span.textContent='';span.id='account-role';}if(span.textContent.includes('Sync:')){span.textContent='Sync: SSE';span.id='sync';}if(span.textContent.includes('3s ago')){span.textContent='';span.id='freshness';}if(span.textContent.trim()==='3'){span.textContent='';span.id='notification-count';span.hidden=true;}}
  const account=header.querySelector('#account')?.closest('.pl-space-xs');if(account){account.dataset.action='account';account.setAttribute('role','button');account.tabIndex=0;}
  const notification=header.querySelector('button');if(notification)notification.dataset.action='notifications';
  const mobile=doc.createElement('button');mobile.id='menu';mobile.className='stitch-mobile-menu';mobile.setAttribute('aria-label','Buka navigasi');mobile.innerHTML='<span class="material-symbols-outlined">menu</span>';header.prepend(mobile);
  shell={aside:aside.outerHTML,header:header.outerHTML};
 }
 const main=doc.querySelector('main');sanitize(main,name);manifest[name]={html:main.innerHTML,classes:main.className,source:'Desain/'+name+'/code.html',tables:[...main.querySelectorAll('table')].map(t=>[...t.querySelectorAll('th')].map(e=>e.textContent.trim().replace(/\s+/g,' ')))};
 dom.window.close();
}
fs.writeFileSync(path.join(out,'stitch-pages.json'),JSON.stringify(manifest));
fs.writeFileSync(path.join(out,'stitch-export.css'),[...styles].join('\n'));
fs.writeFileSync(path.join(__dirname,'tailwind.config.cjs'),'module.exports='+JSON.stringify({...config,content:['../../Desain/**/code.html','../../internal/platform/ui/*.html','../../internal/platform/ui/stitch*.js'],safelist:['text-primary','bg-primary-container','text-on-primary-container','font-semibold','rounded-lg','hidden','bg-surface-container-low','text-error','text-outline','text-secondary','text-tertiary','bg-error-container']},null,2)+';\n');
const old=fs.readFileSync(path.join(out,'platform.js'),'utf8');if(!fs.existsSync(path.join(out,'stitch-resources.js'))){const start=old.indexOf('const pages='),end=old.indexOf('\nlet active',start);fs.writeFileSync(path.join(out,'stitch-resources.js'),old.slice(start,end)+'\n');}
const index=`<!doctype html><html lang="id"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>PasTrack · Fleet Intelligence</title><link rel="stylesheet" href="/assets/fonts.css"><link rel="stylesheet" href="/assets/stitch-utilities.css"><link rel="stylesheet" href="/assets/stitch-export.css"><link rel="stylesheet" href="/assets/leaflet.css"><link rel="stylesheet" href="/assets/stitch.css"><script src="/assets/leaflet.js" defer></script><script src="/assets/stitch-resources.js" defer></script><script src="/assets/stitch.js" defer></script></head><body class="bg-surface font-body-md text-on-surface antialiased selection:bg-primary selection:text-on-primary">${shell.aside}<div class="pl-64 stitch-workspace">${shell.header}<main id="screen" class="w-full pt-16 bg-surface min-h-screen" aria-live="polite"></main></div><div id="toast" role="status" hidden></div><dialog id="editor"><form id="edit-form"><h2 id="editor-title"></h2><div id="fields"></div><p id="form-error" role="alert"></p><div class="stitch-dialog-actions"><button type="button" id="cancel">Batal</button><button id="save">Simpan</button></div></form></dialog><dialog id="inspector"><div class="stitch-dialog-heading"><h2 id="inspector-title"></h2><button id="inspector-close" aria-label="Tutup">×</button></div><div id="inspector-content"></div></dialog><div id="sidebar-scrim" hidden></div></body></html>`;
fs.writeFileSync(path.join(out,'index.html'),index);
console.log('Imported '+sources.length+' exact local page structures and '+(shell.aside.match(/data-path=/g)||[]).length+' navigation links.');
