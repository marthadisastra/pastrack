<script setup>
import { ref, watch, nextTick, onBeforeUnmount } from 'vue'
import AppIcon from './AppIcon.vue'
const props = defineProps({
  modelValue: Boolean,
  title: String,
  wide: Boolean,
  dirty: Boolean,
  busy: Boolean,
})
const emit = defineEmits(['update:modelValue'])
const dialog = ref(null),
  confirmClose = ref(false),
  layer = ref(90)
let previousFocus
function discard() {
  confirmClose.value = false
  emit('update:modelValue', false)
}
function close() {
  if (props.busy) return
  if (props.dirty) {
    confirmClose.value = true
    return
  }
  discard()
}
defineExpose({ close })
function keydown(event) {
  if (
    !props.modelValue ||
    Math.max(
      ...[...document.querySelectorAll('[data-modal-layer]')].map((el) =>
        Number(el.dataset.modalLayer),
      ),
    ) !== layer.value
  )
    return
  if (event.key === 'Escape') {
    event.preventDefault()
    if (confirmClose.value) confirmClose.value = false
    else close()
  }
  if (event.key !== 'Tab') return
  const elements = [
    ...dialog.value.querySelectorAll(
      'button:not(:disabled), a[href], input:not(:disabled), select:not(:disabled), textarea, [tabindex="0"]',
    ),
  ].filter((el) => el.getClientRects().length)
  const first = elements[0],
    last = elements.at(-1)
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last?.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first?.focus()
  }
}
watch(
  () => props.modelValue,
  async (open) => {
    if (open) {
      layer.value =
        Math.max(
          79,
          ...[...document.querySelectorAll('[data-modal-layer]')].map((el) =>
            Number(el.dataset.modalLayer),
          ),
        ) + 1
      previousFocus = document.activeElement
      document.addEventListener('keydown', keydown)
      await nextTick()
      dialog.value?.focus()
    } else {
      confirmClose.value = false
      document.removeEventListener('keydown', keydown)
      previousFocus?.focus()
    }
  },
)
onBeforeUnmount(() => {
  document.removeEventListener('keydown', keydown)
  previousFocus?.focus()
})
</script>
<template>
  <Teleport to="body"
    ><div
      v-if="modelValue"
      class="fixed inset-0 z-[80] flex items-center justify-center overflow-y-auto bg-slate-950/45 p-4 backdrop-blur-sm"
      :style="{ zIndex: layer }"
      @click.self="close"
    >
      <section
        ref="dialog"
        :data-modal-layer="layer"
        tabindex="-1"
        role="dialog"
        aria-modal="true"
        :aria-label="title"
        class="flex max-h-[88dvh] w-full flex-col overflow-hidden rounded-xl bg-white shadow-2xl outline-none"
        :class="wide ? 'max-w-3xl' : 'max-w-xl'"
      >
        <header class="flex items-center justify-between gap-4 border-b border-slate-200 px-4 py-3">
          <h2 class="font-semibold text-slate-900">{{ title }}</h2>
          <button class="btn !p-1.5" :disabled="busy" aria-label="Close dialog" @click="close">
            <AppIcon name="close" />
          </button>
        </header>
        <div class="overflow-y-auto p-4"><slot /></div>
        <div
          v-if="confirmClose"
          class="flex flex-wrap items-center justify-between gap-2 border-t border-amber-200 bg-amber-50 px-4 py-3 text-xs text-amber-900"
          role="status"
        >
          <span>Discard unsaved changes?</span>
          <div class="flex gap-2">
            <button class="btn" @click="confirmClose = false">Keep editing</button
            ><button class="btn btn-danger" @click="discard">Discard changes</button>
          </div>
        </div>
        <footer
          v-if="$slots.footer"
          class="flex flex-wrap justify-end gap-2 border-t border-slate-200 bg-slate-50 px-4 py-2.5"
        >
          <slot name="footer" :close="close" />
        </footer>
      </section></div
  ></Teleport>
</template>
