<script setup>
import { computed, onUnmounted, ref, watch } from 'vue'
import { dashboardService } from '../../services/dashboard'
import { mergeFuelSamples } from '../../services/fuelHistory'
import StatusBadge from '../ui/StatusBadge.vue'

const props = defineProps({
  vehicles: { type: Array, default: () => [] },
  liveSamples: { type: Array, default: () => [] },
  modelValue: String,
})
const emit = defineEmits(['update:modelValue'])
const period = ref('7d')
const points = ref([])
const meta = ref(null)
const loading = ref(false)
const error = ref('')
let controller

const selected = computed(() => props.vehicles.find((vehicle) => vehicle.id === props.modelValue))
const eligible = computed(() => props.vehicles.filter((vehicle) => vehicle.monitored))

function chooseDefault() {
  if (!props.modelValue || !props.vehicles.some((vehicle) => vehicle.id === props.modelValue))
    emit('update:modelValue', eligible.value[0]?.id || props.vehicles[0]?.id || '')
}

async function load() {
  if (!props.modelValue) return
  controller?.abort()
  controller = new AbortController()
  const active = controller
  loading.value = true
  error.value = ''
  try {
    const response = await dashboardService.fuelHistory(
      props.modelValue,
      period.value,
      active.signal,
    )
    if (active !== controller) return
    points.value = response.data.points
    meta.value = response.meta
  } catch (cause) {
    if (cause.name !== 'AbortError' && active === controller) error.value = cause.message
  } finally {
    if (active === controller) loading.value = false
  }
}

watch(() => props.vehicles.map((vehicle) => vehicle.id), chooseDefault, { immediate: true })
watch([() => props.modelValue, period], load, { immediate: true })
function mergeLiveSamples() {
  const merged = mergeFuelSamples(points.value, props.liveSamples, props.modelValue, meta.value)
  points.value = merged.points
  if (meta.value && merged.to) meta.value.to = merged.to
}

watch([() => props.liveSamples, meta, () => props.modelValue], mergeLiveSamples, {
  deep: true,
  flush: 'post',
})

function bounded(values, limit = 600) {
  if (values.length <= limit) return values
  const size = Math.ceil(values.length / (limit / 2))
  const result = []
  for (let index = 0; index < values.length; index += size) {
    const bucket = values.slice(index, index + size)
    result.push(
      bucket.reduce((minimum, point) => (point.liters < minimum.liters ? point : minimum)),
      bucket.reduce((maximum, point) => (point.liters > maximum.liters ? point : maximum)),
    )
  }
  return result.sort((a, b) => new Date(a.at) - new Date(b.at)).slice(0, limit)
}

const chart = computed(() => {
  const width = 720
  const height = 180
  const left = 46
  const right = 12
  const top = 12
  const bottom = 28
  const from = new Date(meta.value?.from || Date.now() - 12 * 3600000).getTime()
  const to = new Date(meta.value?.to || Date.now()).getTime()
  const ordered = [...points.value].sort((a, b) => new Date(a.at) - new Date(b.at))
  const valid = ordered.filter(
    (point) => point.validity === 'valid' && point.liters !== null && Number.isFinite(point.liters),
  )
  const values = valid.map((point) => point.liters)
  let minimum = values.length ? Math.min(...values) : 0
  let maximum = values.length ? Math.max(...values) : 1
  if (minimum === maximum) {
    minimum = Math.max(0, minimum - 1)
    maximum += 1
  }
  const x = (at) =>
    left + ((new Date(at).getTime() - from) / Math.max(1, to - from)) * (width - left - right)
  const y = (value) => top + ((maximum - value) / (maximum - minimum)) * (height - top - bottom)
  const segments = []
  let segment = []
  for (const point of ordered) {
    if (point.validity !== 'valid' || point.liters === null || !Number.isFinite(point.liters)) {
      if (segment.length) segments.push(segment)
      segment = []
      continue
    }
    const previous = segment.at(-1)
    if (previous && new Date(point.at) - new Date(previous.at) > 30 * 60 * 1000) {
      if (segment.length) segments.push(segment)
      segment = []
    }
    segment.push(point)
  }
  if (segment.length) segments.push(segment)
  return {
    width,
    height,
    left,
    right,
    top,
    bottom,
    minimum,
    maximum,
    segments: segments.map((items) =>
      bounded(items)
        .map((point) => `${x(point.at)},${y(point.liters)}`)
        .join(' '),
    ),
    last: valid.at(-1),
  }
})

const format = (value) =>
  value === null || value === undefined
    ? '—'
    : Number(value).toLocaleString('en-US', { maximumFractionDigits: 1 })
onUnmounted(() => controller?.abort())
</script>

<template>
  <section class="panel overflow-hidden">
    <header class="panel-title flex-wrap gap-2">
      <div>
        <h2 class="text-xs font-semibold">Fuel History</h2>
        <p class="mt-0.5 text-[9px] text-slate-500">
          Database samples · actual event time · gaps preserved
        </p>
      </div>
      <div class="flex items-center gap-1.5">
        <label class="sr-only" for="fuel-history-vehicle">Fuel history vehicle</label>
        <select
          id="fuel-history-vehicle"
          :value="modelValue"
          class="h-7 max-w-40 !py-0 !text-[10px]"
          @change="emit('update:modelValue', $event.target.value)"
        >
          <option v-for="vehicle in eligible" :key="vehicle.id" :value="vehicle.id">
            {{ vehicle.code }} · {{ vehicle.plate }}
          </option>
        </select>
        <button
          v-for="value in ['12h', '24h', '7d']"
          :key="value"
          class="btn !px-2 !py-1 !text-[9px] uppercase"
          :class="period === value && '!border-blue-300 !bg-blue-50 !text-blue-700'"
          :aria-pressed="period === value"
          @click="period = value"
        >
          {{ value }}
        </button>
      </div>
    </header>
    <div v-if="loading" class="skeleton m-3 h-[180px]" role="status">
      <span class="sr-only">Loading fuel history…</span>
    </div>
    <div
      v-else-if="error"
      class="m-3 rounded border border-rose-200 bg-rose-50 p-3 text-[11px] text-rose-700"
      role="alert"
    >
      {{ error }}<button class="ml-2 underline" @click="load">Retry</button>
    </div>
    <div
      v-else-if="!modelValue"
      class="grid h-[180px] place-items-center p-4 text-center text-[11px] text-slate-500"
    >
      No monitored vehicle is available.
    </div>
    <div
      v-else-if="!chart.last"
      class="grid h-[180px] place-items-center p-4 text-center text-[11px] text-slate-500"
    >
      No valid fuel samples in this {{ period.toUpperCase() }} range. Invalid readings remain gaps.
    </div>
    <div v-else class="px-2 pb-2 pt-1">
      <svg
        class="h-[180px] w-full"
        :viewBox="`0 0 ${chart.width} ${chart.height}`"
        role="img"
        :aria-label="`${period} fuel history for ${selected?.code}`"
      >
        <g stroke="#e2e8f0" stroke-width="1">
          <line
            v-for="n in 4"
            :key="n"
            :x1="chart.left"
            :x2="chart.width - chart.right"
            :y1="12 + (n - 1) * 46"
            :y2="12 + (n - 1) * 46"
          />
        </g>
        <g fill="#64748b" font-size="9">
          <text x="4" y="17">{{ format(chart.maximum) }} L</text>
          <text x="4" :y="chart.height - chart.bottom">{{ format(chart.minimum) }} L</text>
          <text :x="chart.left" :y="chart.height - 7">{{ period.toUpperCase() }} AGO</text>
          <text :x="chart.width - 34" :y="chart.height - 7">NOW</text>
        </g>
        <polyline
          v-for="(segment, index) in chart.segments"
          :key="index"
          :points="segment"
          fill="none"
          stroke="#0891b2"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </svg>
      <div
        class="flex items-center justify-between border-t border-slate-100 px-2 pt-2 text-[9px] text-slate-500"
      >
        <span>{{ points.length }} samples · max 5,000 server rows</span>
        <StatusBadge tone="cyan">{{ format(chart.last.liters) }} L</StatusBadge>
      </div>
    </div>
  </section>
</template>
