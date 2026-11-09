<script setup>
import { ref, watch, onBeforeUnmount } from 'vue'
import AppIcon from '../ui/AppIcon.vue'
const props = defineProps({ src: String, alt: String })
const emit = defineEmits(['error', 'loaded'])
const loading = ref(true),
  failed = ref(false),
  revision = ref(0)
let deadline
function start() {
  loading.value = true
  failed.value = false
  clearTimeout(deadline)
  deadline = setTimeout(fail, 15000)
}
function done() {
  clearTimeout(deadline)
  loading.value = false
  emit('loaded')
}
function fail() {
  clearTimeout(deadline)
  if (!loading.value) return
  failed.value = true
  loading.value = false
  emit('error')
}
function retry() {
  revision.value++
  start()
}
watch(() => props.src, start, { immediate: true })
onBeforeUnmount(() => clearTimeout(deadline))
</script>
<template>
  <div class="relative h-full w-full overflow-hidden" :aria-busy="loading">
    <div
      v-if="loading"
      class="skeleton absolute inset-0 z-10 flex items-center justify-center rounded-none"
      role="status"
    >
      <span class="rounded-md bg-white/90 px-3 py-2 text-[10px] text-slate-500"
        >Loading map image…</span
      >
    </div>
    <img
      v-if="!failed"
      :key="`${src}:${revision}`"
      :src="src"
      :alt="alt"
      class="h-full w-full object-cover transition-opacity duration-300"
      :class="loading ? 'opacity-0' : 'opacity-100'"
      decoding="async"
      @load="done"
      @error="fail"
    />
    <div
      v-else
      class="flex h-full flex-col items-center justify-center gap-2 bg-slate-100 p-3 text-center text-xs text-slate-500"
    >
      <AppIcon name="broken_image" /><span>Map image unavailable. Use vector view or retry.</span
      ><button class="btn" @click="retry">Retry image</button>
    </div>
  </div>
</template>
