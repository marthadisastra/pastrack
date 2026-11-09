'use strict';
function vehicleMapIcon(color){
 const paint=/^#[0-9a-fA-F]{6}$/.test(color)?color:'#087f72';
 return L.divIcon({className:'track-car-marker',iconSize:[30,40],iconAnchor:[15,20],html:`<span class="track-car-direction"><svg width="30" height="40" viewBox="0 0 30 40" aria-hidden="true"><path d="M8 3.5Q15 0 22 3.5L25 10V31Q25 36 20 37H10Q5 36 5 31V10Z" fill="${paint}" stroke="#fff" stroke-width="2"/><path d="M9 10Q15 8 21 10L22 16H8Z" fill="#d8f1fb"/><path d="M8 23H22V29H8Z" fill="#d8f1fb" opacity=".9"/><path d="M7 31H10M20 31H23" stroke="#fff" stroke-width="2.5" stroke-linecap="round"/><path d="M5 14H2M25 14H28M5 28H2M25 28H28" stroke="#263b50" stroke-width="3" stroke-linecap="round"/></svg></span>`});
}
