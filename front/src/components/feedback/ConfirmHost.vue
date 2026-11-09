<script setup>
import AppModal from '../ui/AppModal.vue'
import { useConfirm } from '../../composables/useConfirm'
const { confirmation, answer } = useConfirm()
</script>
<template>
  <AppModal
    :model-value="Boolean(confirmation)"
    :title="confirmation?.title || 'Confirm action'"
    @update:model-value="answer(false)"
    ><p class="text-xs leading-6 text-slate-600">{{ confirmation?.message }}</p>
    <template #footer
      ><button class="btn" @click="answer(false)">
        {{ confirmation?.cancelLabel || 'Keep editing' }}</button
      ><button class="btn btn-danger" @click="answer(true)">
        {{ confirmation?.confirmLabel || 'Discard changes' }}
      </button></template
    ></AppModal
  >
</template>
