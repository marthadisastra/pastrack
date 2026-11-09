<script setup>
import { computed, ref } from 'vue'
import { useFleet } from '../../composables/useFleet'
import AppIcon from '../../components/ui/AppIcon.vue'
import StatCard from '../../components/ui/StatCard.vue'
import StatusBadge from '../../components/ui/StatusBadge.vue'
import FleetMap from '../../components/fleet/FleetMap.vue'
import FuelChart from '../../components/fleet/FuelChart.vue'
import VehicleTable from '../../components/fleet/VehicleTable.vue'
import VehicleInspectModal from '../../components/fleet/VehicleInspectModal.vue'
import { exportCsv } from '../../services/export'
import LiveValue from '../../components/feedback/LiveValue.vue'
const { vehicles, load, loading, samples } = useFleet()
const chartId = ref('HAULER-04')
const query = ref(''),
  state = ref(''),
  showFilters = ref(false),
  period = ref('24H'),
  selected = ref(null),
  inspectOpen = ref(false)
const monitored = computed(() => vehicles.value.filter((v) => v.imei))
const moving = computed(() => monitored.value.filter((v) => v.state === 'moving').length)
const alerts = computed(() => monitored.value.filter((v) => v.state === 'alert').length)
const fuel = computed(() => monitored.value.reduce((sum, v) => sum + v.fuel, 0))
const capacity = computed(() => monitored.value.reduce((sum, v) => sum + v.capacity, 0))
const filtered = computed(() =>
  monitored.value.filter(
    (v) =>
      (!state.value || v.state === state.value) &&
      `${v.id} ${v.plate} ${v.imei}`.toLowerCase().includes(query.value.toLowerCase()),
  ),
)
const chartVehicle = computed(
  () => monitored.value.find((v) => v.id === chartId.value) || monitored.value[0],
)
const avgCsq = computed(() =>
  monitored.value.length
    ? (monitored.value.reduce((sum, v) => sum + (v.csq || 0), 0) / monitored.value.length).toFixed(
        1,
      )
    : '—',
)
const avgHdop = computed(() =>
  monitored.value.length
    ? (monitored.value.reduce((sum, v) => sum + (v.hdop || 0), 0) / monitored.value.length).toFixed(
        2,
      )
    : '—',
)
function inspect(v) {
  selected.value = v
  inspectOpen.value = true
}
function clearFilters() {
  state.value = ''
  query.value = ''
}
</script>
<template>
  <div class="page">
    <section class="panel flex flex-wrap items-center justify-between gap-3 p-3">
      <div>
        <div class="flex flex-wrap items-center gap-2">
          <h1 class="text-sm font-bold tracking-tight text-slate-950">
            Operations Telematics Dashboard
          </h1>
          <StatusBadge tone="blue">GV30CEU + CFS300B DUAL-LINK</StatusBadge>
        </div>
        <p class="mt-2 font-mono text-[10px] text-slate-400">
          {{ monitored.length }} Monitored:
          <span class="text-emerald-600">● {{ moving }} Moving</span> ·
          <span class="text-cyan-600"
            >{{ monitored.filter((v) => v.state === 'idle').length }} Idling</span
          >
          · <span class="text-rose-600">{{ alerts }} Alert</span>
        </p>
      </div>
      <div class="flex items-center gap-2">
        <div class="flex rounded-lg border border-slate-200 bg-slate-50 p-1">
          <button
            v-for="item in ['12H', '24H']"
            :key="item"
            class="rounded px-3 py-1.5 text-[11px]"
            :class="
              period === item ? 'bg-white font-semibold text-blue-700 shadow-sm' : 'text-slate-500'
            "
            :aria-pressed="period === item"
            @click="period = item"
          >
            {{ item }}
          </button>
        </div>
        <button class="btn" :aria-expanded="showFilters" @click="showFilters = !showFilters">
          <AppIcon name="tune" />Filters</button
        ><button class="btn" aria-label="Refresh dashboard" :disabled="loading" @click="load(true)">
          <AppIcon name="sync" :class="loading && 'animate-spin'" />
        </button>
      </div>
    </section>
    <div v-if="showFilters" class="panel flex flex-wrap items-center gap-3 p-3">
      <label class="field"
        >Operational State<select v-model="state" class="min-w-44">
          <option value="">All states</option>
          <option value="moving">Moving</option>
          <option value="idle">Idling</option>
          <option value="alert">Alert</option>
        </select></label
      ><button class="btn self-end" @click="clearFilters">Clear filters</button>
    </div>
    <div class="grid grid-cols-2 gap-3 xl:grid-cols-4">
      <StatCard
        label="Fleet Operational State"
        :value="`${monitored.length} Active Units`"
        icon="local_shipping"
        ><div class="mb-2 flex h-1.5 overflow-hidden rounded-full bg-slate-100">
          <div
            class="bg-emerald-500"
            :style="{ width: `${(moving / (monitored.length || 1)) * 100}%` }"
          ></div>
          <div class="flex-1 bg-cyan-500"></div>
          <div class="w-[10%] bg-rose-500"></div>
        </div>
        <span class="font-mono text-emerald-600">{{ moving }} Moving</span
        ><span class="float-right text-[10px]"
          >{{ monitored.length - moving }} Stationary</span
        ></StatCard
      ><StatCard
        label="CFS300B BLE Fuel Tank"
        :value="`${fuel.toLocaleString('en-US')} L`"
        icon="local_gas_station"
        ><div class="mb-2 h-1.5 rounded-full bg-slate-100">
          <div
            class="h-full rounded-full bg-cyan-600"
            :style="{ width: `${(fuel / (capacity || 1)) * 100}%` }"
          ></div>
        </div>
        <span class="font-mono">{{ ((fuel / (capacity || 1)) * 100).toFixed(1) }}% AVG</span
        ><span class="float-right text-[10px] text-emerald-600">0 Uncalibrated</span></StatCard
      ><StatCard
        label="Hardware Telemetry Alerts"
        :value="`${alerts} Active Event`"
        icon="warning"
        danger
        ><p class="font-mono text-[10px] text-rose-600">P1 HAULER-04 Siphon Drop (−48.2L)</p>
        <p class="mt-1 text-[10px]">Review vehicle inspection</p></StatCard
      ><StatCard label="Gateway Signal Quality" :value="`${avgCsq} / 31`" icon="satellite_alt"
        ><p class="font-mono text-[10px] text-emerald-600">GNSS 3D LOCKED</p>
        <p class="mt-1 text-[10px]">
          CSQ: <LiveValue :value="avgCsq" />/31 · HDOP <LiveValue :value="avgHdop" /></p
      ></StatCard>
    </div>
    <div class="grid gap-3 xl:grid-cols-[1.35fr_1fr]">
      <section class="panel">
        <header class="panel-title">
          <h2
            class="flex items-center gap-2 font-mono text-[11px] font-semibold uppercase tracking-wider"
          >
            <AppIcon name="map" class="text-blue-600" />Geospatial Fleet Telematics Radar
          </h2>
          <StatusBadge tone="green">DEMO VECTOR</StatusBadge>
        </header>
        <div class="h-[240px]"><FleetMap :vehicles="filtered" @select="inspect" /></div>
        <div
          class="truncate border-t border-slate-200 px-4 py-2 font-mono text-[9px] text-slate-400"
        >
          LAST QUECLINK PACKET:
          <span class="text-blue-600"
            >+RESP:GTFRI,{{ monitored[0]?.imei }},{{ monitored[0]?.speed }},{{
              monitored[0]?.lat
            }},{{ monitored[0]?.lng }},{{ monitored[0]?.fuel }}…</span
          >
        </div>
      </section>
      <section class="panel">
        <header class="panel-title">
          <h2
            class="flex items-center gap-2 font-mono text-[11px] font-semibold uppercase tracking-wider"
          >
            <AppIcon name="monitoring" class="text-cyan-600" />CFS300B Dynamic Fuel Level ({{
              period
            }})
          </h2>
          <StatusBadge tone="cyan">KALMAN FILTER ON</StatusBadge>
        </header>
        <div class="space-y-2 p-2.5">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div>
              <h3 class="text-xs font-semibold">{{ chartVehicle?.id || 'Fleet' }} Tank Profile</h3>
              <p class="mt-1 font-mono text-[9px] text-slate-400">
                {{ chartVehicle?.plate }} · {{ chartVehicle?.capacity }}L Primary Tank
              </p>
            </div>
            <label class="flex items-center gap-1.5"
              ><span class="sr-only">Fuel chart unit</span
              ><select v-model="chartId" class="h-7 !py-1 !text-[10px]">
                <option v-for="v in monitored" :key="v.id" :value="v.id">{{ v.id }}</option>
              </select></label
            >
          </div>
          <FuelChart
            :period="period"
            :samples="samples[chartVehicle?.id] || []"
            :capacity="chartVehicle?.capacity || 0"
          />
          <div class="grid grid-cols-3 gap-2">
            <div
              v-for="(value, key) in {
                'PROBE BATTERY': `${chartVehicle?.probeBattery ?? '—'}V`,
                'BLE 5.0 SIGNAL': `${chartVehicle?.rssi ?? '—'} dBm`,
                'ZERO-POINT DRIFT': `${chartVehicle?.drift ?? '—'} mm`,
              }"
              :key="key"
              class="rounded border border-slate-200 bg-slate-50 p-2 font-mono"
            >
              <p class="text-[8px] text-slate-400">{{ key }}</p>
              <p class="mt-1 text-[10px] font-semibold"><LiveValue :value="value" /></p>
            </div>
          </div>
        </div>
      </section>
    </div>
    <section class="panel">
      <div class="panel-title">
        <h2
          class="flex items-center gap-2 font-mono text-[11px] font-semibold uppercase tracking-wider"
        >
          <AppIcon name="table_chart" class="text-blue-600" />Active Vehicle Gateway Surveillance
          Matrix
          <span class="rounded bg-slate-200/70 px-1.5 py-1 text-[9px] normal-case text-slate-500"
            >{{ filtered.length }} Showing</span
          >
        </h2>
        <div class="flex flex-wrap gap-2">
          <label class="sr-only" for="matrix-search">Filter vehicle or IMEI</label
          ><input
            id="matrix-search"
            v-model="query"
            class="h-8 w-48 !text-xs"
            placeholder="Filter Vehicle / IMEI…"
          /><button
            class="btn !py-1.5"
            :disabled="!filtered.length"
            @click="exportCsv('fleet-telemetry.csv', filtered)"
          >
            <AppIcon name="download" class="!text-base" />Export Log
          </button>
        </div>
      </div>
      <VehicleTable :vehicles="filtered" @inspect="inspect" />
    </section>
    <VehicleInspectModal v-model="inspectOpen" :vehicle="selected" />
  </div>
</template>
