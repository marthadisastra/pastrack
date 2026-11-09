<script setup>
import { ref, watch, onBeforeUnmount } from 'vue'
const props = defineProps({ value: [String, Number], suffix: { type: String, default: '' } })
const changed = ref(false)
let timer, frame
watch(
  () => props.value,
  (value, prior) => {
    if (value === prior || prior === null || prior === undefined) return
    changed.value = false
    clearTimeout(timer)
    // The key-free retained node avoids focus changes when a value updates.
    cancelAnimationFrame(frame)
    frame = requestAnimationFrame(() => {
      changed.value = true
    })
    timer = setTimeout(() => {
      changed.value = false
    }, 700)
  },
)
onBeforeUnmount(() => {
  clearTimeout(timer)
  cancelAnimationFrame(frame)
})
</script>
<template>
  <span class="live-value tabular-nums" :class="changed && 'live-value-changed'"
    >{{ value ?? '—' }}{{ suffix }}</span
  >
</template>
