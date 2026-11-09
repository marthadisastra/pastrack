<script setup>
defineProps({ tone: { type: String, default: 'slate' }, dot: Boolean })
const tones = {
  blue: 'bg-blue-50 text-blue-700 border-blue-200',
  green: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  red: 'bg-rose-50 text-rose-600 border-rose-200',
  cyan: 'bg-cyan-50 text-cyan-700 border-cyan-200',
  amber: 'bg-amber-50 text-amber-700 border-amber-200',
  slate: 'bg-slate-100 text-slate-600 border-slate-200',
}
</script>
<template>
  <span
    class="inline-flex items-center gap-1.5 rounded border px-2 py-1 text-[10px] font-semibold leading-tight"
    :class="tones[tone] || tones.slate"
    ><span v-if="dot" class="h-1.5 w-1.5 shrink-0 rounded-full bg-current"></span><slot
  /></span>
</template>
