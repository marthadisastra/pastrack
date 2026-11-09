<script setup>
import { ref } from 'vue'
import AppIcon from '../ui/AppIcon.vue'
import AsyncImage from '../feedback/AsyncImage.vue'
import LiveValue from '../feedback/LiveValue.vue'
import mapImage from '../../assets/fleet-map.jpg'
import trackingImage from '../../assets/tracking-map.jpg'
const props = defineProps({
  vehicles: { type: Array, default: () => [] },
  selected: String,
  trail: Boolean,
  geofences: { type: Boolean, default: true },
  tracking: Boolean,
  insets: { type: Object, default: () => ({ left: 0, right: 0 }) },
})
defineEmits(['select'])
const zoom = ref(1),
  satellite = ref(true)
const colors = {
  red: 'bg-rose-500',
  green: 'bg-emerald-500',
  blue: 'bg-blue-600',
  cyan: 'bg-cyan-500',
  slate: 'bg-slate-500',
}
function recenter() {
  zoom.value = 1
}
defineExpose({ recenter })
</script>
<template>
  <div
    class="relative h-full min-h-48 overflow-hidden bg-slate-200"
    aria-label="Demo fleet position map"
    :style="{ '--map-left': `${insets.left}px`, '--map-right': `${insets.right}px` }"
  >
    <div
      class="absolute inset-0 transition-transform duration-300"
      :style="{ transform: `scale(${zoom})` }"
    >
      <AsyncImage
        v-if="satellite"
        :src="tracking ? trackingImage : mapImage"
        alt="Satellite map illustration from the fleet mockup"
        class="opacity-85"
      />
      <div v-else class="absolute inset-0 bg-[#e8eee7] blueprint-grid">
        <svg
          class="h-full w-full"
          viewBox="0 0 800 500"
          preserveAspectRatio="none"
          aria-hidden="true"
        >
          <path
            d="M0 370 Q200 290 170 0 M200 500 Q250 300 800 130 M0 90 L800 430"
            fill="none"
            stroke="#fff"
            stroke-width="24"
          />
          <path
            d="M0 370 Q200 290 170 0 M200 500 Q250 300 800 130 M0 90 L800 430"
            fill="none"
            stroke="#cbd5b9"
            stroke-width="4"
          />
          <path d="M0 500 L0 410 Q300 450 440 500" fill="#add8e6" />
        </svg>
      </div>
      <div class="pointer-events-none absolute inset-0 bg-slate-900/10"></div>
      <svg
        class="pointer-events-none absolute inset-0 h-full w-full"
        viewBox="0 0 800 500"
        preserveAspectRatio="none"
        aria-hidden="true"
      >
        <g
          v-if="geofences"
          fill="#3b82f61a"
          stroke="#3b82f6"
          stroke-width="2"
          stroke-dasharray="7 5"
        >
          <path d="M70 160 L220 95 L300 240 L200 360 L65 300 Z" />
          <path d="M480 300 L680 260 L730 420 L555 450 Z" />
        </g>
        <path
          v-if="trail"
          d="M180 460 Q250 400 320 360 T450 260 L500 200"
          fill="none"
          stroke="#22d3ee"
          stroke-width="4"
          stroke-dasharray="9 4"
        /></svg
      ><span
        v-if="geofences"
        class="absolute left-[10%] top-[12%] rounded border border-blue-400/50 bg-slate-900/85 px-2 py-1 font-mono text-[9px] text-blue-200"
        >ZONE: Tg. Priok Port Terminal</span
      ><button
        v-for="v in vehicles.filter((v) => v.imei)"
        :key="v.id"
        class="map-marker absolute flex -translate-x-1/2 -translate-y-1/2 flex-col items-center gap-1 focus:outline-none"
        :style="{
          left: `clamp(calc(var(--map-safe-left, 0px) + 70px), ${v.x}%, calc(100% - var(--map-safe-right, 0px) - 70px))`,
          top: `${v.y}%`,
        }"
        :aria-label="`Select ${v.id}`"
        :aria-pressed="selected === v.id"
        @click="$emit('select', v)"
      >
        <span
          class="flex h-7 w-7 items-center justify-center rounded-full border-2 border-white text-white shadow-lg"
          :class="[colors[v.tone], selected === v.id && 'ring-4 ring-blue-300/60']"
          ><AppIcon
            :name="v.state === 'alert' ? 'warning' : v.speed ? 'navigation' : 'pause'"
            class="!text-base" /></span
        ><span
          class="whitespace-nowrap rounded border border-white/30 bg-slate-900/90 px-2 py-1 font-mono text-[8px] text-white shadow"
          >{{ v.id }}
          <span class="text-slate-300"><LiveValue :value="v.speed" suffix=" km/h" /></span
        ></span>
      </button>
    </div>
    <div
      class="map-right-control absolute right-3 top-3 flex rounded-lg border border-slate-200 bg-white/95 p-1 text-[10px] shadow"
    >
      <button
        class="rounded px-2 py-1"
        :class="!satellite && 'bg-blue-50 text-blue-700'"
        :aria-pressed="!satellite"
        @click="satellite = false"
      >
        Vector</button
      ><button
        class="rounded px-2 py-1"
        :class="satellite && 'bg-blue-50 text-blue-700'"
        :aria-pressed="satellite"
        @click="satellite = true"
      >
        Satellite
      </button>
    </div>
    <div
      class="map-right-control absolute bottom-3 right-3 flex flex-col overflow-hidden rounded-lg border border-slate-200 bg-white shadow"
    >
      <button
        class="p-1.5 text-slate-600 hover:bg-slate-50"
        aria-label="Zoom in"
        :disabled="zoom >= 1.8"
        @click="zoom = Math.min(1.8, zoom + 0.2)"
      >
        <AppIcon name="add" /></button
      ><button
        class="border-t border-slate-100 p-1.5 text-slate-600 hover:bg-slate-50"
        aria-label="Zoom out"
        :disabled="zoom <= 1"
        @click="zoom = Math.max(1, zoom - 0.2)"
      >
        <AppIcon name="remove" />
      </button>
    </div>
    <div
      class="map-left-control absolute bottom-3 left-3 max-w-[75%] truncate rounded bg-slate-900/90 px-2 py-1.5 font-mono text-[8px] text-slate-300"
    >
      DEMO GIS · {{ selected || 'FLEET OVERVIEW' }} · −6.2146°, 106.8451°
    </div>
  </div>
</template>

<style scoped>
@media (min-width: 1024px) {
  [aria-label='Demo fleet position map'] {
    --map-safe-left: var(--map-left);
    --map-safe-right: var(--map-right);
  }
  .map-right-control {
    right: calc(var(--map-right, 0px) + 12px);
  }
  .map-left-control {
    left: calc(var(--map-left, 0px) + 12px);
    max-width: calc(100% - var(--map-left, 0px) - var(--map-right, 0px) - 70px);
  }
}
</style>
