<script setup>
import { useToast } from '../../composables/useToast'
import AppIcon from './AppIcon.vue'
const { messages, dismiss, pause, resume } = useToast()
</script>
<template>
  <div
    class="pointer-events-none fixed bottom-10 right-3 z-[100] flex w-[340px] max-w-[calc(100vw-24px)] flex-col gap-2"
    aria-relevant="additions"
  >
    <TransitionGroup name="toast"
      ><div
        v-for="item in messages"
        :key="item.id"
        class="pointer-events-auto flex items-start gap-2 rounded-lg border bg-white p-3 shadow-lg"
        :class="
          item.tone === 'error'
            ? 'border-red-200 text-red-700'
            : item.tone === 'info'
              ? 'border-blue-200 text-blue-700'
              : 'border-emerald-200 text-emerald-700'
        "
        :role="item.tone === 'error' ? 'alert' : 'status'"
        @mouseenter="pause(item.id)"
        @mouseleave="resume(item.id)"
        @focusin="pause(item.id)"
        @focusout="resume(item.id)"
      >
        <AppIcon
          :name="item.tone === 'error' ? 'error' : item.tone === 'info' ? 'info' : 'check_circle'"
          class="mt-0.5 shrink-0 !text-lg"
        /><span class="flex-1 text-[11px] leading-5">{{ item.message }}</span
        ><button
          class="shrink-0 rounded text-current opacity-60 hover:opacity-100"
          aria-label="Dismiss notification"
          @click="dismiss(item.id)"
        >
          <AppIcon name="close" class="!text-base" />
        </button></div
    ></TransitionGroup>
  </div>
</template>
