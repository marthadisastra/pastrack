<script setup>
import AppModal from '../ui/AppModal.vue'
import LiveValue from '../feedback/LiveValue.vue'
import { useFleet } from '../../composables/useFleet'
const { samples } = useFleet()
import StatusBadge from '../ui/StatusBadge.vue'
import AppIcon from '../ui/AppIcon.vue'
import FuelChart from './FuelChart.vue'
import { exportCsv } from '../../services/export'
import { useToast } from '../../composables/useToast'
defineProps({ modelValue: Boolean, vehicle: Object, live: Boolean })
defineEmits(['update:modelValue'])
const { notify } = useToast()
const number = (value, digits = 1) =>
  value === null || value === undefined || value === '' ? '—' : Number(value).toFixed(digits)
</script>
<template>
  <AppModal
    :model-value="modelValue"
    :title="`Asset Inspection · ${vehicle?.id || ''}`"
    wide
    @update:model-value="$emit('update:modelValue', $event)"
    ><div v-if="vehicle && live" class="space-y-5">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 class="font-mono text-xl font-bold">{{ vehicle.id }}</h3>
          <p class="mt-1 text-xs text-slate-500">{{ vehicle.plate }} · {{ vehicle.model }}</p>
        </div>
        <StatusBadge :tone="vehicle.tone" dot>{{ vehicle.status }}</StatusBadge>
      </div>
      <dl class="grid gap-3 rounded-lg border border-slate-200 p-4 text-xs sm:grid-cols-2">
        <div><dt class="text-slate-400">Fleet</dt><dd class="mt-1 font-medium">{{ vehicle.fleet_code || 'Not assigned' }}</dd></div>
        <div><dt class="text-slate-400">Tank capacity</dt><dd class="mt-1 font-mono">{{ vehicle.capacity || '—' }} L</dd></div>
        <div><dt class="text-slate-400">Gateway</dt><dd class="mt-1 font-mono">{{ vehicle.imei || 'Not paired' }}</dd></div>
        <div><dt class="text-slate-400">Gateway model</dt><dd class="mt-1">{{ vehicle.device_model || '—' }}</dd></div>
        <div><dt class="text-slate-400">Last device message</dt><dd class="mt-1">{{ vehicle.updatedAt ? new Date(vehicle.updatedAt).toLocaleString() : 'No message received' }}</dd></div>
      </dl>
      <p class="text-[11px] leading-5 text-slate-500">Live telemetry history, BLE sensor status, and calibration are available from their respective integrated pages.</p>
    </div>
    <div v-else-if="vehicle" class="space-y-5">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 class="font-mono text-xl font-bold">{{ vehicle.id }}</h3>
          <p class="mt-1 text-xs text-slate-500">{{ vehicle.plate }} · {{ vehicle.model }}</p>
        </div>
        <StatusBadge :tone="vehicle.tone" dot>{{ vehicle.status }}</StatusBadge>
      </div>
      <div
        v-if="vehicle.state === 'alert'"
        class="rounded-lg border border-rose-200 bg-rose-50 p-4 text-xs leading-5 text-rose-700"
      >
        <strong>P1 · Fuel siphon detected</strong>
        <p>
          Demo event: −48.2 liters while ignition is off. Review telemetry before taking action.
        </p>
      </div>
      <div class="grid gap-4 sm:grid-cols-2">
        <div class="rounded-lg border border-slate-200 p-4">
          <h4 class="mb-4 flex items-center gap-2 text-xs font-semibold">
            <AppIcon name="router" />Queclink GV30CEU Gateway
          </h4>
          <dl class="space-y-3 text-xs">
            <div
              v-for="(value, label) in {
                IMEI: vehicle.imei || 'Not paired',
                'GNSS Lock': vehicle.imei ? '3D FIX · 14 Sats' : 'Unavailable',
                Coordinates: `${vehicle.lat}, ${vehicle.lng}`,
                'Main Battery': `${vehicle.battery} V`,
                Ignition: vehicle.ignition ? 'ACC ON' : 'ACC OFF',
                Velocity: `${vehicle.speed} km/h`,
              }"
              :key="label"
              class="flex flex-wrap justify-between gap-1"
            >
              <dt class="text-slate-400">{{ label }}</dt>
              <dd class="font-mono text-[11px]"><LiveValue :value="value" /></dd>
            </div>
          </dl>
        </div>
        <div class="rounded-lg border border-slate-200 p-4">
          <h4 class="mb-4 flex items-center gap-2 text-xs font-semibold text-cyan-700">
            <AppIcon name="local_gas_station" />CFS300B BLE Fuel Probe
          </h4>
          <p class="mb-4 font-mono text-2xl font-bold">
            <LiveValue :value="number(vehicle.fuel)" />
            <span class="text-xs font-normal text-slate-400">/ {{ vehicle.capacity }} L</span>
          </p>
          <dl class="space-y-3 text-xs">
            <div
              v-for="(value, label) in {
                'Fuel Level': `${vehicle.capacity > 0 ? number((vehicle.fuel / vehicle.capacity) * 100) : '—'}%`,
                'Probe Battery': `${vehicle.probeBattery} V`,
                'BLE RSSI': `${vehicle.rssi} dBm`,
                Calibration: 'Zero-point OK',
              }"
              :key="label"
              class="flex justify-between"
            >
              <dt class="text-slate-400">{{ label }}</dt>
              <dd class="font-mono text-[11px]"><LiveValue :value="value" /></dd>
            </div>
          </dl>
        </div>
      </div>
      <FuelChart :samples="samples[vehicle.id] || []" :capacity="vehicle.capacity" />
      <div class="rounded-lg bg-slate-950 p-4 font-mono text-[10px] leading-6 text-emerald-400">
        <p class="mb-1 text-slate-400">LATEST RAW TELEMETRY · GTRFI</p>
        <p class="break-all">
          +RESP:GTFRI,1A0103,{{ vehicle.imei || 'UNPAIRED' }},{{ vehicle.id }},{{
            vehicle.speed
          }},{{ vehicle.lng }},{{ vehicle.lat }},{{ vehicle.fuel }}$
        </p>
      </div>
    </div>
    <template #footer
      ><button
        v-if="!live"
        class="btn"
        @click="notify('Latest telemetry already loaded from the backend.', 'info')"
      >
        <AppIcon name="sync" />Poll Demo Packet</button
      ><button
        v-if="vehicle"
        class="btn"
        @click="exportCsv(`${vehicle.id}-${live ? 'asset' : 'telemetry'}.csv`, [vehicle])"
      >
        <AppIcon name="download" />{{ live ? 'Export Asset' : 'Export CSV' }}</button
      ><button class="btn btn-primary" @click="$emit('update:modelValue', false)">
        Close Inspection
      </button></template
    ></AppModal
  >
</template>
