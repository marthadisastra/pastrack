<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppIcon from '../../../components/ui/AppIcon.vue'
import StatusBadge from '../../../components/ui/StatusBadge.vue'
import RealtimeFleetMap from '../../../components/fleet/RealtimeFleetMap.vue'
import { useDashboard } from '../../../composables/useDashboard'
import { useFleet } from '../../../composables/useFleet'
import { dashboardDemo } from '../../../services/dashboard'
import { exportCsv } from '../../../services/export'

const route = useRoute(),
  router = useRouter(),
  dashboard = useDashboard()
const connection = dashboard.connection,
  synchronizationError = dashboard.error
const demoFleet = useFleet()
const query = ref(''),
  state = ref(''),
  roster = ref(true),
  panel = ref(true)
const activeTab = ref('overview'),
  selectedId = ref(String(route.query.unit || ''))
const demoVehicles = computed(() =>
  demoFleet.vehicles.value.map((v) => ({
    id: String(v.id),
    code: v.id,
    plate: v.plate,
    name: v.model,
    monitored: Boolean(v.imei),
    movementStatus: String(v.state || 'unknown').toUpperCase(),
    connectivity: v.state === 'offline' ? 'OFFLINE' : 'ONLINE',
    deviceIdentifier: v.imei,
    activeAlerts: v.state === 'alert' ? 1 : 0,
    sensorStatus: 'demo',
    gps: {
      latitude: Number(v.lat),
      longitude: Number(v.lng),
      speedKmh: Number(v.speed),
      headingDeg: v.heading ?? null,
      satellites: v.satellites ?? null,
      hdop: v.hdop ?? null,
      fixStatus: 'valid',
      fresh: true,
      measuredAt: v.updatedAt,
    },
    fuel: {
      sampleId: `demo:${v.updatedAt || v.id}`,
      liters: Number(v.fuel),
      percent: Number(v.capacity) > 0 ? (Number(v.fuel) / Number(v.capacity)) * 100 : null,
      validity: 'valid',
      calibrationId: null,
      measuredAt: v.updatedAt,
    },
    ignition: { status: v.ignition ? 'ON' : 'OFF', physicalState: 'DEMO' },
  })),
)
const sourceVehicles = computed(() =>
  dashboardDemo ? demoVehicles.value : dashboard.vehicles.value,
)
const tracked = computed(() => sourceVehicles.value.filter((v) => v.monitored))
const filtered = computed(() => {
  const needle = query.value.trim().toLowerCase()
  return tracked.value.filter(
    (v) =>
      (!state.value || v.movementStatus === state.value) &&
      (!needle ||
        `${v.code} ${v.plate} ${v.name} ${v.deviceIdentifier || ''}`
          .toLowerCase()
          .includes(needle)),
  )
})
const selected = computed(
  () => tracked.value.find((v) => v.id === selectedId.value) || tracked.value[0],
)
const validFixes = computed(
  () =>
    filtered.value.filter(
      (v) =>
        v.gps.fixStatus === 'valid' &&
        Number.isFinite(v.gps.latitude) &&
        Number.isFinite(v.gps.longitude) &&
        !(v.gps.latitude === 0 && v.gps.longitude === 0),
    ).length,
)
const states = ['MOVING', 'STOPPED', 'OFFLINE', 'STALE', 'GPS_NO_FIX', 'UNKNOWN_SPEED']
const number = (value, digits = 1) =>
  value === null || value === undefined
    ? '—'
    : Number(value).toLocaleString('en-US', { maximumFractionDigits: digits })
const time = (value) =>
  value
    ? new Intl.DateTimeFormat('id-ID', {
        timeZone: 'Asia/Jakarta',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      }).format(new Date(value)) + ' WIB'
    : '—'
function select(vehicle) {
  selectedId.value = vehicle.id
  panel.value = true
  router.replace({ query: { ...route.query, unit: vehicle.id } })
}
function showTelemetry() {
  panel.value = true
  activeTab.value = 'telemetry'
}
function exportTracking() {
  exportCsv('live-tracking-snapshot.csv', filtered.value, [
    'id',
    'code',
    'plate',
    'name',
    'movementStatus',
    'connectivity',
    'activeAlerts',
  ])
}
watch(
  tracked,
  (rows) => {
    if (rows.length && !rows.some((v) => v.id === selectedId.value)) selectedId.value = rows[0].id
  },
  { immediate: true },
)
watch(
  () => route.query.unit,
  (value) => {
    if (value) selectedId.value = String(value)
  },
)
</script>

<template>
  <div class="tracking-view flex min-h-full flex-col bg-slate-100 lg:h-full lg:min-h-0">
    <header
      class="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-slate-200 bg-white px-3 py-2"
    >
      <div class="flex items-center gap-2">
        <h1 class="flex items-center gap-2 text-sm font-bold">
          <AppIcon name="near_me" class="text-blue-700" />Live Tracking
        </h1>
        <StatusBadge :tone="dashboardDemo || connection.status === 'live' ? 'green' : 'amber'">{{
          dashboardDemo ? 'DEMO GPS' : connection.status === 'live' ? 'WSS LIVE' : connection.status
        }}</StatusBadge>
        <span class="hidden text-[10px] text-slate-500 sm:inline"
          >{{ validFixes }} valid fixes · {{ tracked.length }} monitored</span
        >
      </div>
      <div class="flex flex-wrap gap-2">
        <button class="btn !py-1.5" :aria-pressed="roster" @click="roster = !roster">
          <AppIcon name="view_sidebar" />Roster
        </button>
        <button class="btn !py-1.5" :aria-pressed="panel" @click="showTelemetry">
          <AppIcon name="sensors" />Telemetry
        </button>
        <button
          class="btn !py-1.5"
          @click="
            dashboardDemo
              ? demoFleet.reconnect()
              : dashboard.load('Live Tracking resync requested.')
          "
        >
          <AppIcon name="sync" />Resync
        </button>
        <button class="btn !py-1.5" @click="exportTracking">
          <AppIcon name="download" />Export
        </button>
      </div>
    </header>
    <p
      v-if="synchronizationError"
      class="m-3 rounded border border-rose-200 bg-rose-50 p-2 text-xs text-rose-700"
      role="alert"
    >
      {{ synchronizationError }}
    </p>
    <div class="tracking-stage relative grid min-h-0 flex-1 grid-cols-1 gap-3 p-3 lg:block lg:p-0">
      <section
        v-if="roster"
        class="tracking-roster panel z-[600] order-2 flex max-h-[300px] min-h-0 flex-col lg:absolute lg:bottom-2 lg:left-2 lg:top-2 lg:max-h-none lg:w-[224px]"
      >
        <div class="panel-title">
          <h2 class="text-xs font-semibold">FLEET ROSTER</h2>
          <button aria-label="Minimize fleet roster" class="text-slate-400" @click="roster = false">
            <AppIcon name="left_panel_close" class="!text-base" />
          </button>
        </div>
        <div class="space-y-2 border-b border-slate-200 p-2">
          <label class="sr-only" for="tracking-search">Search tracking units</label
          ><input
            id="tracking-search"
            v-model="query"
            class="w-full !text-xs"
            placeholder="Code / Plate / Gateway…"
          />
          <select v-model="state" class="h-7 w-full !py-0 !text-[10px]" aria-label="Tracking state">
            <option value="">All operational states</option>
            <option v-for="item in states" :key="item">{{ item }}</option>
          </select>
        </div>
        <div class="space-y-1.5 overflow-y-auto p-2">
          <button
            v-for="vehicle in filtered"
            :key="vehicle.id"
            class="w-full rounded-md border p-2 text-left transition hover:border-blue-300"
            :class="selected?.id === vehicle.id ? 'border-blue-300 bg-blue-50' : 'border-slate-200'"
            :aria-pressed="selected?.id === vehicle.id"
            @click="select(vehicle)"
          >
            <div class="flex items-center justify-between gap-2">
              <span class="truncate font-mono text-[10px] font-semibold">{{ vehicle.code }}</span
              ><StatusBadge
                :tone="vehicle.activeAlerts ? 'red' : vehicle.gps.fresh ? 'green' : 'slate'"
                >{{ vehicle.movementStatus }}</StatusBadge
              >
            </div>
            <p class="mt-1 truncate text-[9px] text-slate-500">
              {{ vehicle.plate }} · {{ vehicle.name }}
            </p>
            <div class="mt-1 flex justify-between font-mono text-[9px] text-slate-600">
              <span>{{ number(vehicle.gps.speedKmh) }} km/h</span
              ><span>{{ number(vehicle.fuel.liters) }} L</span>
            </div>
          </button>
          <p v-if="!filtered.length" class="p-5 text-center text-xs text-slate-400">
            No units found.
          </p>
        </div>
      </section>
      <div
        class="tracking-map panel relative order-1 h-[380px] lg:absolute lg:inset-0 lg:h-full lg:rounded-none lg:border-0"
      >
        <RealtimeFleetMap fill :vehicles="filtered" :selected="selected?.id" @select="select" />
      </div>
      <section
        v-if="panel && selected"
        class="tracking-hud panel z-[600] order-3 min-h-0 overflow-y-auto lg:absolute lg:bottom-2 lg:right-2 lg:top-2 lg:w-[280px]"
      >
        <div class="panel-title">
          <div class="min-w-0">
            <h2 class="truncate font-mono text-xs font-semibold">{{ selected.code }}</h2>
            <p class="truncate text-[9px] text-slate-500">{{ selected.plate }}</p>
          </div>
          <button
            aria-label="Minimize telemetry panel"
            class="text-slate-400"
            @click="panel = false"
          >
            <AppIcon name="right_panel_close" class="!text-base" />
          </button>
        </div>
        <div
          class="grid grid-cols-2 border-b border-slate-200 p-1"
          role="tablist"
          aria-label="Tracking detail"
        >
          <button
            v-for="tab in ['overview', 'telemetry']"
            :key="tab"
            role="tab"
            class="rounded px-2 py-1.5 text-[10px] font-semibold uppercase"
            :class="activeTab === tab ? 'bg-blue-50 text-blue-700' : 'text-slate-500'"
            :aria-selected="activeTab === tab"
            @click="activeTab = tab"
          >
            {{ tab }}
          </button>
        </div>
        <div v-if="activeTab === 'overview'" class="space-y-2 p-2">
          <div class="grid grid-cols-2 gap-2">
            <div class="rounded border border-slate-200 p-2">
              <p class="text-[9px] text-slate-400">SPEED</p>
              <p class="mt-1 font-mono text-sm font-semibold">
                {{ number(selected.gps.speedKmh) }} km/h
              </p>
            </div>
            <div class="rounded border border-slate-200 p-2">
              <p class="text-[9px] text-slate-400">FUEL</p>
              <p class="mt-1 font-mono text-sm font-semibold">
                {{ number(selected.fuel.liters) }} L
              </p>
            </div>
          </div>
          <dl class="space-y-2 rounded border border-slate-200 bg-slate-50 p-2 text-[10px]">
            <div class="flex justify-between">
              <dt>Movement</dt>
              <dd>{{ selected.movementStatus }}</dd>
            </div>
            <div class="flex justify-between">
              <dt>Connectivity</dt>
              <dd>{{ selected.connectivity }}</dd>
            </div>
            <div class="flex justify-between">
              <dt>Ignition</dt>
              <dd>{{ selected.ignition.status }}</dd>
            </div>
            <div class="flex justify-between">
              <dt>Physical input</dt>
              <dd>{{ selected.ignition.physicalState }}</dd>
            </div>
            <div class="flex justify-between">
              <dt>Active alerts</dt>
              <dd>{{ selected.activeAlerts }}</dd>
            </div>
          </dl>
        </div>
        <div v-else class="space-y-2 p-2" role="tabpanel" aria-label="Telemetry">
          <div class="rounded border border-slate-200 bg-slate-50 p-2">
            <h3 class="font-mono text-[10px] font-semibold text-slate-600">GPS TELEMETRY</h3>
            <dl class="mt-2 space-y-1.5 font-mono text-[9px]">
              <div class="flex justify-between">
                <dt>Latitude</dt>
                <dd>{{ number(selected.gps.latitude, 6) }}</dd>
              </div>
              <div class="flex justify-between">
                <dt>Longitude</dt>
                <dd>{{ number(selected.gps.longitude, 6) }}</dd>
              </div>
              <div class="flex justify-between">
                <dt>Heading</dt>
                <dd>{{ number(selected.gps.headingDeg) }}°</dd>
              </div>
              <div class="flex justify-between">
                <dt>Satellites</dt>
                <dd>{{ number(selected.gps.satellites, 0) }}</dd>
              </div>
              <div class="flex justify-between">
                <dt>HDOP</dt>
                <dd>{{ number(selected.gps.hdop, 2) }}</dd>
              </div>
              <div class="flex justify-between">
                <dt>Fix</dt>
                <dd>{{ selected.gps.fixStatus }}</dd>
              </div>
              <div class="flex justify-between">
                <dt>Measured</dt>
                <dd>{{ time(selected.gps.measuredAt) }}</dd>
              </div>
            </dl>
          </div>
          <div class="rounded border border-slate-200 bg-slate-50 p-2">
            <h3 class="font-mono text-[10px] font-semibold text-slate-600">
              FUEL / SENSOR TELEMETRY
            </h3>
            <dl class="mt-2 space-y-1.5 font-mono text-[9px]">
              <div class="flex justify-between">
                <dt>Sample ID</dt>
                <dd>{{ selected.fuel.sampleId || '—' }}</dd>
              </div>
              <div class="flex justify-between">
                <dt>Liters</dt>
                <dd>{{ number(selected.fuel.liters) }} L</dd>
              </div>
              <div class="flex justify-between">
                <dt>Percent</dt>
                <dd>{{ number(selected.fuel.percent) }}%</dd>
              </div>
              <div class="flex justify-between">
                <dt>Validity</dt>
                <dd>{{ selected.fuel.validity }}</dd>
              </div>
              <div class="flex justify-between">
                <dt>Calibration</dt>
                <dd>{{ selected.fuel.calibrationId || '—' }}</dd>
              </div>
              <div class="flex justify-between">
                <dt>Sensor</dt>
                <dd>{{ selected.sensorStatus || '—' }}</dd>
              </div>
              <div class="flex justify-between">
                <dt>Measured</dt>
                <dd>{{ time(selected.fuel.measuredAt) }}</dd>
              </div>
            </dl>
          </div>
          <p class="rounded bg-blue-50 p-2 text-[9px] leading-4 text-blue-800">
            Values update from committed WebSocket events. Missing hardware values remain
            unavailable.
          </p>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
@media (min-width: 1024px) {
  .tracking-stage {
    overflow: hidden;
  }
  .tracking-hud {
    height: calc(100% - 16px);
  }
}
</style>
