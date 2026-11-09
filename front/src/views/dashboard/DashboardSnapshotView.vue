<script setup>
import { computed, ref } from 'vue'
import { useDashboard } from '../../composables/useDashboard'
import AppIcon from '../../components/ui/AppIcon.vue'
import StatCard from '../../components/ui/StatCard.vue'
import StatusBadge from '../../components/ui/StatusBadge.vue'
import AppModal from '../../components/ui/AppModal.vue'
import RealtimeFleetMap from '../../components/fleet/RealtimeFleetMap.vue'
import FuelHistoryChart from '../../components/fleet/FuelHistoryChart.vue'
import { exportCsv } from '../../services/export'
const { vehicles, summary, fuelSamples, asOf, loading, error, load, connection } = useDashboard()
const query = ref(''),
  state = ref(''),
  selectedId = ref(null),
  historyVehicleId = ref(''),
  inspectOpen = ref(false)
const states = [
  'MOVING',
  'STOPPED',
  'OFFLINE',
  'UNASSIGNED',
  'STALE',
  'GPS_NO_FIX',
  'UNKNOWN_SPEED',
]
const filtered = computed(() =>
  vehicles.value.filter(
    (v) =>
      (!state.value || v.movementStatus === state.value) &&
      `${v.code} ${v.plate} ${v.deviceIdentifier || ''}`
        .toLowerCase()
        .includes(query.value.toLowerCase()),
  ),
)
const selected = computed(() => vehicles.value.find((v) => v.id === selectedId.value))
const number = (value, decimals = 1) =>
  value === null || value === undefined
    ? '—'
    : Number(value).toLocaleString('en-US', {
        minimumFractionDigits: decimals,
        maximumFractionDigits: decimals,
      })
const time = (value) =>
  value ? new Date(value).toLocaleString('en-GB', { timeZone: 'Asia/Jakarta' }) + ' WIB' : '—'
const tone = (state) =>
  state === 'MOVING' || state === 'ONLINE' ? 'green' : state === 'STOPPED' ? 'blue' : 'slate'
function inspect(vehicle) {
  selectedId.value = vehicle.id
  historyVehicleId.value = vehicle.id
  inspectOpen.value = true
}
function download() {
  exportCsv(
    'dashboard-snapshot.csv',
    filtered.value.map((v) => ({
      vehicle_id: v.id,
      vehicle_code: v.code,
      plate: v.plate,
      gateway: v.deviceIdentifier,
      movement: v.movementStatus,
      connectivity: v.connectivity,
      latitude: v.gps.latitude,
      longitude: v.gps.longitude,
      speed_kmh: v.gps.speedKmh,
      fuel_liters: v.fuel.liters,
      fuel_validity: v.fuel.validity,
      fuel_fresh: v.fuel.fresh,
      ignition: v.ignition.status,
      active_alerts: v.activeAlerts,
      snapshot_at: asOf.value,
    })),
    [
      'vehicle_id',
      'vehicle_code',
      'plate',
      'gateway',
      'movement',
      'connectivity',
      'latitude',
      'longitude',
      'speed_kmh',
      'fuel_liters',
      'fuel_validity',
      'fuel_fresh',
      'ignition',
      'active_alerts',
      'snapshot_at',
    ],
  )
}
</script>
<template>
  <div class="page">
    <section class="panel flex flex-wrap items-center justify-between gap-3 p-3">
      <div>
        <h1 class="text-sm font-bold">Operations Telematics Dashboard</h1>
        <p class="mt-1 text-[10px] text-slate-500">Realtime projection · {{ time(asOf) }}</p>
      </div>
      <div class="flex items-center gap-2">
        <StatusBadge :tone="connection.status === 'live' ? 'green' : 'blue'">{{
          connection.status === 'live' ? 'WEBSOCKET LIVE' : connection.status.toUpperCase()
        }}</StatusBadge
        ><button class="btn" aria-label="Refresh dashboard" :disabled="loading" @click="load">
          <AppIcon name="sync" :class="loading && 'animate-spin'" />{{
            loading ? 'Loading…' : 'Resync'
          }}
        </button>
      </div>
    </section>
    <div
      v-if="error"
      role="alert"
      class="rounded-lg border border-rose-200 bg-rose-50 p-3 text-xs text-rose-700"
    >
      {{ error }}
      <span v-if="summary">The previous snapshot is retained; its timestamp has not changed.</span
      ><button class="ml-3 underline" :disabled="loading" @click="load">Retry</button>
    </div>
    <div v-if="loading && !summary" role="status" class="grid grid-cols-2 gap-3 xl:grid-cols-4">
      <div v-for="n in 4" :key="n" class="panel skeleton h-28"></div>
      <span class="sr-only">Loading dashboard snapshot…</span>
    </div>
    <template v-if="summary">
      <div class="grid grid-cols-2 gap-3 xl:grid-cols-4">
        <StatCard
          label="Fleet Operational State"
          :value="`${summary.monitoredVehicles} Monitored`"
          icon="local_shipping"
          ><p class="text-[10px]">
            {{ summary.movementCounts.MOVING }} Moving ·
            {{ summary.movementCounts.STOPPED }} Stopped
          </p>
          <p class="mt-1 text-[10px] text-slate-500">
            {{ summary.totalVehicles }} Total · {{ summary.movementCounts.UNASSIGNED }} Unassigned
          </p></StatCard
        >
        <StatCard
          label="Valid Fresh Fuel"
          :value="`${number(summary.fuelLiters)} L`"
          icon="local_gas_station"
          ><p class="text-[10px]">
            {{ number(summary.fuelFillPercent) }}% weighted fill ·
            {{ summary.fuelCapacityEligibleVehicles }} with known capacity
          </p>
          <p class="mt-1 text-[10px] text-slate-500">
            {{ summary.fuelValidFreshVehicles }}/{{ summary.monitoredVehicles }} valid and fresh ·
            {{ summary.calibrationUnknownVehicles }} calibration unknown
          </p></StatCard
        >
        <StatCard
          label="Hardware Telemetry Alerts"
          :value="`${summary.activeAlerts} Active Events`"
          icon="warning"
          :danger="summary.activeAlerts > 0"
          ><p class="text-[10px]">Excludes resolved, closed and false-positive events.</p></StatCard
        >
        <StatCard
          label="Gateway & GPS Quality"
          :value="`${summary.devicesOnline} Online`"
          icon="satellite_alt"
          ><p class="text-[10px]">
            {{ summary.gpsFreshVehicles }} Valid fresh fixes · HDOP
            {{ number(summary.averageHdop, 2) }}
          </p>
          <p class="mt-1 text-[10px] text-slate-500">
            {{ summary.hdopSamples }} HDOP samples · CSQ not available
          </p></StatCard
        >
      </div>
      <div class="grid gap-3 md:grid-cols-2">
        <section class="panel overflow-hidden">
          <header class="panel-title">
            <div>
              <h2 class="text-xs font-semibold">Geographic Fleet Map</h2>
              <p class="mt-0.5 text-[9px] text-slate-500">
                Valid coordinates only · realtime marker updates
              </p>
            </div>
            <StatusBadge :tone="connection.status === 'live' ? 'green' : 'slate'"
              >LIVE GIS</StatusBadge
            >
          </header>
          <RealtimeFleetMap :vehicles="filtered" :selected="selectedId" @select="inspect" />
        </section>
        <FuelHistoryChart
          v-model="historyVehicleId"
          :vehicles="vehicles"
          :live-samples="fuelSamples"
        />
      </div>
      <section class="panel">
        <header class="panel-title flex-wrap gap-2">
          <h2 class="text-xs font-semibold">
            Vehicle Gateway Matrix · {{ filtered.length }} Showing
          </h2>
          <div class="flex flex-wrap gap-2">
            <label class="sr-only" for="snapshot-state">Operational state</label
            ><select id="snapshot-state" v-model="state" class="h-8 !py-1 !text-[10px]">
              <option value="">All states</option>
              <option v-for="item in states" :key="item">{{ item }}</option></select
            ><label class="sr-only" for="snapshot-search">Filter vehicle or gateway</label
            ><input
              id="snapshot-search"
              v-model="query"
              placeholder="Code / Plate / Gateway…"
              class="h-8 w-44 !text-xs"
            /><button class="btn" :disabled="!filtered.length" @click="download">
              Export Snapshot
            </button>
          </div>
        </header>
        <div class="overflow-x-auto">
          <table class="w-full min-w-[870px] text-[11px]">
            <thead class="bg-slate-50">
              <tr>
                <th>Asset / Plate</th>
                <th>Gateway</th>
                <th>State / Connectivity</th>
                <th>Ignition</th>
                <th>Fuel / Capacity</th>
                <th>Speed / GPS</th>
                <th>Alerts</th>
                <th>Inspection</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-100">
              <tr v-for="v in filtered" :key="v.id">
                <td>
                  <p class="font-semibold">{{ v.code }}</p>
                  <p class="mt-1 text-[10px] text-slate-500">{{ v.plate }} · {{ v.name }}</p>
                </td>
                <td>{{ v.deviceIdentifier || 'UNASSIGNED' }}</td>
                <td>
                  <StatusBadge :tone="tone(v.movementStatus)">{{ v.movementStatus }}</StatusBadge>
                  <p class="mt-1 text-[10px]">{{ v.connectivity }}</p>
                </td>
                <td>
                  {{ v.ignition.status }}
                  <p class="mt-1 text-[9px] text-slate-500">
                    Physical: {{ v.ignition.physicalState }}
                  </p>
                </td>
                <td>
                  {{ number(v.fuel.liters) }} / {{ number(v.fuel.capacityLiters, 0) }} L
                  <p class="mt-1 text-[9px] text-slate-500">
                    {{ v.fuel.validity }} · {{ v.fuel.fresh ? 'FRESH' : 'NOT FRESH' }}
                  </p>
                </td>
                <td>
                  {{ number(v.gps.speedKmh) }} km/h
                  <p class="mt-1 text-[9px] text-slate-500">
                    {{ v.gps.fixStatus }} · {{ v.gps.fresh ? 'FRESH' : 'NOT FRESH' }}
                  </p>
                </td>
                <td>{{ v.activeAlerts }}</td>
                <td><button class="btn !py-1" @click="inspect(v)">Inspect</button></td>
              </tr>
              <tr v-if="!filtered.length">
                <td colspan="8" class="py-10 text-center text-slate-500">
                  {{
                    vehicles.length
                      ? 'No vehicles match your filters.'
                      : 'No vehicles are registered in this database scope.'
                  }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>
    <AppModal
      v-model="inspectOpen"
      :title="`Vehicle Inspection · ${selected?.code || 'Unavailable'}`"
      wide
      ><div v-if="selected" class="space-y-3 text-xs">
        <p class="text-slate-500">Database ID {{ selected.id }} · Snapshot {{ time(asOf) }}</p>
        <div class="grid gap-4 sm:grid-cols-2">
          <dl class="space-y-2">
            <div>
              <dt class="text-slate-500">Coordinates</dt>
              <dd>
                {{ number(selected.gps.latitude, 6) }}, {{ number(selected.gps.longitude, 6) }}
              </dd>
            </div>
            <div>
              <dt class="text-slate-500">GPS measured / received</dt>
              <dd>{{ time(selected.gps.measuredAt) }} / {{ time(selected.gps.receivedAt) }}</dd>
            </div>
            <div>
              <dt class="text-slate-500">Ignition source / reported</dt>
              <dd>
                {{ selected.ignition.source || 'Unknown' }} ·
                {{ time(selected.ignition.reportedAt) }}
              </dd>
            </div>
            <div>
              <dt class="text-slate-500">Gateway last received</dt>
              <dd>{{ time(selected.gatewayReceivedAt) }}</dd>
            </div>
          </dl>
          <dl class="space-y-2">
            <div>
              <dt class="text-slate-500">Fuel measured / received</dt>
              <dd>{{ time(selected.fuel.measuredAt) }} / {{ time(selected.fuel.receivedAt) }}</dd>
            </div>
            <div>
              <dt class="text-slate-500">Calibration reference</dt>
              <dd>{{ selected.fuel.calibrationId || 'Unknown' }}</dd>
            </div>
            <div>
              <dt class="text-slate-500">Sensor status</dt>
              <dd>
                {{ selected.sensorStatus || 'Unknown' }} · {{ time(selected.sensorReceivedAt) }}
              </dd>
            </div>
            <div>
              <dt class="text-slate-500">Active alert codes</dt>
              <dd>{{ selected.alertCodes.join(', ') || 'None' }}</dd>
            </div>
          </dl>
        </div>
      </div>
      <p v-else class="text-xs">This vehicle is no longer in the snapshot.</p>
      <template #footer
        ><button class="btn" @click="inspectOpen = false">Close Inspection</button></template
      ></AppModal
    >
  </div>
</template>
