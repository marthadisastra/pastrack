<script setup>
import StatusBadge from '../ui/StatusBadge.vue'
import LiveValue from '../feedback/LiveValue.vue'
defineProps({ vehicles: Array })
defineEmits(['inspect'])
</script>
<template>
  <div class="overflow-x-auto">
    <table class="w-full min-w-[1050px] text-xs">
      <thead class="border-b border-slate-200 bg-slate-50 font-mono">
        <tr>
          <th>Asset & Vehicle Model</th>
          <th>Queclink Gateway / IMEI</th>
          <th>Ignition (DI1 / ACC)</th>
          <th>Fuel CFS300B (Liter / %)</th>
          <th>Velocity & Geofence</th>
          <th>Telemetry Status</th>
          <th>Diagnostic</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-slate-100">
        <tr v-for="v in vehicles" :key="v.id" class="hover:bg-slate-50/70">
          <td>
            <div class="font-mono font-semibold">{{ v.id }}</div>
            <p class="mt-1 max-w-40 text-[10px] leading-4 text-slate-400">
              {{ v.plate }} · {{ v.model }}
            </p>
          </td>
          <td class="font-mono text-[10px]">
            <p>{{ v.imei || 'UNPAIRED' }}</p>
            <p class="mt-1 text-slate-400">
              LTE Cat 1 · CSQ <LiveValue :value="v.csq" />/31 ·
              <LiveValue :value="v.satellites" /> Sats
            </p>
          </td>
          <td>
            <StatusBadge :tone="v.ignition ? 'green' : 'slate'"
              >{{ v.ignition ? 'ACC ON' : 'ACC OFF' }} (<LiveValue
                :value="v.battery"
                suffix="V"
              />)</StatusBadge
            >
          </td>
          <td class="min-w-36">
            <div class="font-mono text-[11px]" :class="v.state === 'alert' && 'text-rose-600'">
              <LiveValue :value="v.fuel.toFixed(1)" suffix=" L" />
              <span class="text-slate-400">({{ ((v.fuel / v.capacity) * 100).toFixed(1) }}%)</span>
            </div>
            <div class="my-1 h-1.5 max-w-32 overflow-hidden rounded-full bg-slate-100">
              <div
                class="h-full rounded-full"
                :class="v.state === 'alert' ? 'bg-rose-500' : 'bg-emerald-500'"
                :style="{ width: `${(v.fuel / v.capacity) * 100}%` }"
              ></div>
            </div>
            <p class="font-mono text-[9px] text-slate-400">
              BLE <LiveValue :value="v.rssi" suffix=" dBm" /> ·
              <LiveValue :value="v.temperature" suffix="°C" />
            </p>
          </td>
          <td>
            <p
              class="font-mono text-[11px]"
              :class="v.speed ? 'text-emerald-600' : 'text-slate-500'"
            >
              <LiveValue :value="v.speed" suffix=" km/h" /> ({{ v.state }})
            </p>
            <p class="mt-1 max-w-40 text-[10px] text-slate-400">{{ v.location }}</p>
          </td>
          <td>
            <StatusBadge :tone="v.tone" dot>{{ v.status }}</StatusBadge>
          </td>
          <td>
            <button
              class="btn !py-1.5 font-mono !text-[10px] !text-blue-700"
              @click="$emit('inspect', v)"
            >
              Inspect
            </button>
          </td>
        </tr>
        <tr v-if="!vehicles.length">
          <td colspan="7" class="py-12 text-center text-slate-400">
            No vehicles match your search.
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
