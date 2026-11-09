<script setup>
import { computed } from 'vue'
const props = defineProps({
  period: { type: String, default: '24H' },
  samples: { type: Array, default: () => [] },
  capacity: { type: Number, default: 400 },
})
const points = computed(() => {
  const rows = props.samples.slice(-60)
  return rows.map((row, index) => [
    rows.length === 1 ? 0 : (index / (rows.length - 1)) * 560,
    175 - Math.min(1, Math.max(0, row.value / (props.capacity || 1))) * 165,
  ])
})
const line = computed(() => {
  if (!points.value.length)
    return props.period === '12H'
      ? 'M0 35 L150 40 L300 43 L301 125 L560 148'
      : 'M0 25 L120 33 L265 38 L266 118 L560 142'
  const path = points.value
    .map(([x, y], index) => `${index ? 'L' : 'M'}${x.toFixed(1)} ${y.toFixed(1)}`)
    .join(' ')
  return points.value.length === 1 ? `${path} L560 ${points.value[0][1]}` : path
})
const last = computed(() => points.value.at(-1))
</script>
<template>
  <div class="relative rounded-lg border border-slate-200 bg-slate-50 p-2">
    <div class="flex justify-between gap-2 font-mono text-[9px] text-slate-400">
      <span>{{ capacity }}L (100%)</span
      ><span
        class="rounded border px-1"
        :class="
          samples.length
            ? 'border-blue-100 bg-blue-50 text-blue-600'
            : 'border-rose-200 bg-white text-rose-500'
        "
        >{{
          samples.length
            ? `${samples.length} LIVE SESSION SAMPLES`
            : 'Reference · ACC OFF Fuel Drainage'
        }}</span
      >
    </div>
    <svg
      viewBox="0 0 560 180"
      preserveAspectRatio="none"
      class="mt-1 h-[88px] w-full"
      role="img"
      :aria-label="
        samples.length
          ? `Session fuel readings. Latest ${samples.at(-1).value.toFixed(1)} liters.`
          : `Reference fuel profile over ${period}`
      "
    >
      <path
        d="M0 35 H560 M0 80 H560 M0 125 H560"
        stroke="#dbe5ef"
        stroke-dasharray="5 5"
        fill="none"
      />
      <path :d="`${line} L560 180 L0 180 Z`" fill="#0891b2" fill-opacity=".13" />
      <path
        :d="line"
        fill="none"
        stroke="#0891b2"
        stroke-width="4"
        vector-effect="non-scaling-stroke"
      />
      <circle
        v-if="last"
        :cx="last[0]"
        :cy="last[1]"
        r="5"
        fill="#2563eb"
        stroke="white"
        stroke-width="2"
      />
    </svg>
    <div class="mt-1 flex justify-between font-mono text-[8px] text-slate-400">
      <span>0L (Empty)</span
      ><span>{{
        samples.length
          ? 'Session feed · historical API not connected'
          : `${period} REFERENCE PROFILE`
      }}</span>
    </div>
  </div>
</template>
