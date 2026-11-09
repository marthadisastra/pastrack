<script setup>
import { usePageLoading } from '../../composables/usePageLoading'
const { visible, navigationError } = usePageLoading()
const reload = () => window.location.reload()
</script>
<template>
  <div
    v-if="visible"
    class="fixed inset-x-0 top-0 z-[120] h-0.5 overflow-hidden bg-blue-100"
    role="status"
    aria-label="Loading page"
  >
    <div class="page-progress-bar h-full w-1/3 bg-blue-600"></div>
  </div>
  <div
    v-if="navigationError"
    role="alert"
    class="fixed left-1/2 top-3 z-[120] w-[calc(100%-24px)] max-w-lg -translate-x-1/2 rounded-lg border border-rose-200 bg-white p-3 text-xs text-rose-700 shadow-lg"
  >
    {{ navigationError }}<button class="btn ml-2" @click="reload">Reload</button>
  </div>
</template>
