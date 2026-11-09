<script setup>
import { computed, ref } from 'vue'
import { useFleet } from '../../composables/useFleet'
import { useMock } from '../../services/http'
import AppIcon from '../ui/AppIcon.vue'
const { connection, paused, stale, age, lastUpdated, togglePaused, reconnect } = useFleet()
const retrying = ref(false)
const label = computed(
  () =>
    ({
      idle: 'Waiting',
      connecting: 'Connecting…',
      demo: 'Demo live',
      live: 'Live',
      reconnecting: 'Reconnecting…',
      offline: 'Offline',
      paused: 'Paused',
      background: 'Background',
      error: 'Unavailable',
    })[connection.value.status] || 'Waiting',
)
const unhealthy = computed(() =>
  ['offline', 'reconnecting', 'error'].includes(connection.value.status),
)
async function retry() {
  retrying.value = true
  try {
    await reconnect()
  } finally {
    retrying.value = false
  }
}
</script>
<template>
  <div
    class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 border-b px-3 py-1.5 text-[10px]"
    :class="
      unhealthy || stale
        ? 'border-amber-200 bg-amber-50 text-amber-800'
        : paused
          ? 'border-slate-200 bg-slate-100 text-slate-600'
          : 'border-blue-100 bg-blue-50/60 text-slate-500'
    "
  >
    <div class="flex flex-wrap items-center gap-2">
      <span class="flex items-center gap-1.5 font-semibold" role="status" aria-live="polite"
        ><span
          class="h-1.5 w-1.5 rounded-full"
          :class="unhealthy || stale ? 'bg-amber-500' : paused ? 'bg-slate-400' : 'bg-emerald-500'"
        ></span
        >{{ label }}</span
      ><span v-if="paused">Values frozen. Resume to synchronize.</span
      ><span v-else-if="unhealthy">{{ connection.detail }}</span
      ><span v-else-if="stale">No recent updates. Last received values are retained.</span
      ><span v-else-if="useMock" class="hidden sm:inline">Simulated telemetry · every 4s</span
      ><span
        v-if="lastUpdated"
        class="font-mono text-[9px]"
        :title="new Date(lastUpdated).toLocaleString()"
        >Updated {{ age === 0 ? 'just now' : `${age}s ago` }}</span
      >
    </div>
    <div class="flex items-center gap-2">
      <button
        v-if="unhealthy || stale"
        class="font-semibold underline underline-offset-2"
        :disabled="retrying"
        @click="retry"
      >
        {{ retrying ? 'Retrying…' : 'Retry connection' }}</button
      ><button
        class="flex items-center gap-1 font-medium hover:text-blue-700"
        :aria-label="paused ? 'Resume live updates' : 'Pause live updates'"
        :aria-pressed="paused"
        @click="togglePaused"
      >
        <AppIcon :name="paused ? 'play_arrow' : 'pause'" class="!text-sm" />{{
          paused ? 'Resume' : 'Pause'
        }}
      </button>
    </div>
  </div>
</template>
