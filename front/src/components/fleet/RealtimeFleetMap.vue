<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import AppIcon from '../ui/AppIcon.vue'

const props = defineProps({
  vehicles: { type: Array, default: () => [] },
  selected: String,
  fill: Boolean,
})
const emit = defineEmits(['select'])
const element = ref(null)
const tileState = ref('loading')
let map
let tiles
let observer
const markers = new Map()

const validVehicles = computed(() =>
  props.vehicles.filter((vehicle) => {
    const latitude = vehicle.gps?.latitude
    const longitude = vehicle.gps?.longitude
    return (
      Number.isFinite(latitude) &&
      Number.isFinite(longitude) &&
      latitude >= -90 &&
      latitude <= 90 &&
      longitude >= -180 &&
      longitude <= 180 &&
      !(latitude === 0 && longitude === 0) &&
      vehicle.gps?.fixStatus === 'valid'
    )
  }),
)

function markerTone(vehicle) {
  if (vehicle.activeAlerts > 0) return '#e11d48'
  if (vehicle.movementStatus === 'MOVING') return '#059669'
  if (vehicle.movementStatus === 'STOPPED') return '#2563eb'
  if (vehicle.movementStatus === 'STALE' || !vehicle.gps.fresh) return '#d97706'
  return '#64748b'
}

function markerIcon(vehicle) {
  const color = markerTone(vehicle)
  const selected = props.selected === vehicle.id
  return L.divIcon({
    className: 'pastrack-map-marker',
    html: `<span style="background:${color};box-shadow:0 0 0 ${selected ? '4px rgba(37,99,235,.25)' : '0 transparent'}"></span>`,
    iconSize: [22, 22],
    iconAnchor: [11, 11],
  })
}

function popupContent(vehicle) {
  const container = document.createElement('div')
  container.className = 'space-y-1 text-[11px]'
  const title = document.createElement('strong')
  title.textContent = vehicle.code || vehicle.plate || `Vehicle ${vehicle.id}`
  const detail = document.createElement('p')
  detail.textContent = `${vehicle.plate || 'No plate'} · ${vehicle.movementStatus} / ${vehicle.connectivity}`
  const measurement = document.createElement('p')
  measurement.textContent = `${vehicle.gps.speedKmh ?? '—'} km/h · ${vehicle.gps.latitude.toFixed(5)}, ${vehicle.gps.longitude.toFixed(5)}`
  container.append(title, detail, measurement)
  return container
}

function syncMarkers({ fit = false } = {}) {
  if (!map) return
  const active = new Set()
  for (const vehicle of validVehicles.value) {
    active.add(vehicle.id)
    const coordinates = [vehicle.gps.latitude, vehicle.gps.longitude]
    let marker = markers.get(vehicle.id)
    if (!marker) {
      marker = L.marker(coordinates, { icon: markerIcon(vehicle), keyboard: true })
        .addTo(map)
        .on('click', () => emit('select', vehicle))
      markers.set(vehicle.id, marker)
    } else {
      marker.setLatLng(coordinates)
      marker.setIcon(markerIcon(vehicle))
      marker.off('click').on('click', () => emit('select', vehicle))
    }
    marker.bindTooltip(`${vehicle.code || vehicle.plate} · ${vehicle.movementStatus}`, {
      direction: 'top',
      offset: [0, -10],
    })
    marker.bindPopup(popupContent(vehicle))
  }
  for (const [id, marker] of markers) {
    if (!active.has(id)) {
      marker.remove()
      markers.delete(id)
    }
  }
  if (fit) recenter()
}

function addTiles() {
  tiles?.remove()
  tileState.value = 'loading'
  // Esri's public World Street Map layer is a Leaflet-compatible raster layer
  // and does not require an API key. OSM's volunteer tile servers can return
  // policy/rate-limit 403s for browser dashboards with multiple live maps.
  tiles = L.tileLayer(
    'https://server.arcgisonline.com/ArcGIS/rest/services/World_Street_Map/MapServer/tile/{z}/{y}/{x}',
    {
      maxZoom: 19,
      attribution: 'Tiles © Esri — Source: Esri, OpenStreetMap contributors',
    },
  )
    .on('load', () => (tileState.value = 'ready'))
    .on('tileerror', () => (tileState.value = 'error'))
    .addTo(map)
}

function recenter() {
  if (!map || !validVehicles.value.length) return
  const bounds = L.latLngBounds(
    validVehicles.value.map((vehicle) => [vehicle.gps.latitude, vehicle.gps.longitude]),
  )
  if (validVehicles.value.length === 1) map.setView(bounds.getCenter(), 14)
  else map.fitBounds(bounds.pad(0.18), { maxZoom: 15, animate: false })
}

onMounted(async () => {
  await nextTick()
  map = L.map(element.value, { zoomControl: false, preferCanvas: true }).setView([-2.5, 118], 5)
  L.control.zoom({ position: 'bottomright' }).addTo(map)
  addTiles()
  syncMarkers({ fit: true })
  observer = new ResizeObserver(() => map?.invalidateSize({ pan: false }))
  observer.observe(element.value)
})

watch(
  () =>
    props.vehicles.map((vehicle) => [
      vehicle.id,
      vehicle.gps?.latitude,
      vehicle.gps?.longitude,
      vehicle.gps?.fresh,
      vehicle.gps?.fixStatus,
      vehicle.movementStatus,
      vehicle.connectivity,
      vehicle.activeAlerts,
    ]),
  () => syncMarkers(),
  { deep: true },
)
watch(
  () => props.selected,
  (id) => {
    syncMarkers()
    if (id && markers.has(id)) map?.panTo(markers.get(id).getLatLng(), { animate: false })
  },
)
onUnmounted(() => {
  observer?.disconnect()
  map?.remove()
  markers.clear()
})
</script>

<template>
  <div
    class="relative z-[80] overflow-hidden bg-slate-100"
    :class="fill ? 'h-full min-h-[360px]' : 'h-[240px]'"
    aria-label="Realtime fleet map"
  >
    <div ref="element" class="h-full w-full"></div>
    <div
      v-if="tileState === 'loading'"
      class="pointer-events-none absolute inset-0 z-[500] grid place-items-center bg-slate-100/70 text-[11px] text-slate-600"
      role="status"
    >
      Loading map tiles…
    </div>
    <div
      v-if="tileState === 'error'"
      class="absolute left-3 top-3 z-[500] flex items-center gap-2 rounded border border-amber-200 bg-white/95 px-2 py-1 text-[10px] text-amber-800 shadow"
      role="alert"
    >
      Map tiles unavailable.<button class="underline" @click="addTiles">Retry</button>
    </div>
    <div
      v-if="!validVehicles.length"
      class="pointer-events-none absolute inset-0 z-[450] grid place-items-center bg-slate-100/75 px-6 text-center text-[11px] text-slate-600"
    >
      No valid GPS fixes are available for the current filter.
    </div>
    <button
      v-else
      class="btn absolute right-3 top-3 z-[500] !bg-white/95 !p-1.5 shadow"
      aria-label="Fit all valid vehicles"
      @click="recenter"
    >
      <AppIcon name="my_location" />
    </button>
    <div
      class="absolute bottom-2 left-2 z-[500] rounded bg-slate-900/85 px-2 py-1 font-mono text-[8px] text-white"
    >
      {{ validVehicles.length }} VALID FIX{{ validVehicles.length === 1 ? '' : 'ES' }} · LIVE WSS
    </div>
  </div>
</template>

<style>
.pastrack-map-marker {
  background: transparent;
  border: 0;
}
.pastrack-map-marker > span {
  display: block;
  width: 18px;
  height: 18px;
  border: 3px solid white;
  border-radius: 999px;
  box-shadow: 0 2px 8px rgba(15, 23, 42, 0.35);
}
.leaflet-container {
  font-family: Inter, system-ui, sans-serif;
}
.leaflet-control-attribution {
  font-size: 8px;
}
</style>
