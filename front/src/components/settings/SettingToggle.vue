<script setup>
const model = defineModel({ type: Boolean })
defineProps({ label: String, disabled: Boolean })
</script>
<template>
  <button
    type="button"
    role="switch"
    :aria-label="label"
    :aria-checked="model"
    :disabled="disabled"
    class="relative inline-flex h-[18px] w-8 shrink-0 items-center rounded-full border transition-colors focus-visible:ring-2 focus-visible:ring-blue-500"
    :class="model ? 'border-blue-600 bg-blue-600' : 'border-slate-300 bg-slate-200'"
    @click="model = !model"
  >
    <span
      class="h-3.5 w-3.5 rounded-full bg-white shadow-sm transition-transform"
      :class="model ? 'translate-x-[15px]' : 'translate-x-px'"
    ></span>
  </button>
</template>
