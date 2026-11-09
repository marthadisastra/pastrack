<script setup>
import AppIcon from './AppIcon.vue'
import LiveValue from '../feedback/LiveValue.vue'
defineProps({ label: String, value: [String, Number], icon: String, danger: Boolean })
</script>
<template>
  <article class="panel stat-card p-2.5" :class="danger && 'border-rose-200'">
    <div class="mb-1.5 flex items-center justify-between gap-2">
      <h2
        class="text-[9px] font-semibold uppercase tracking-widest"
        :class="danger ? 'text-rose-600' : 'text-slate-500'"
      >
        {{ label }}
      </h2>
      <span
        class="flex h-5 w-5 items-center justify-center rounded-lg"
        :class="danger ? 'bg-rose-50 text-rose-500' : 'bg-blue-50 text-blue-600'"
        ><AppIcon :name="icon || 'monitoring'"
      /></span>
    </div>
    <div
      class="mb-1.5 font-mono text-[19px] leading-6 font-bold tracking-tight"
      :class="danger && 'text-rose-600'"
    >
      <LiveValue :value="value" />
    </div>
    <div class="border-t border-slate-100 pt-1.5 text-[10px] leading-4 text-slate-500">
      <slot />
    </div>
  </article>
</template>
