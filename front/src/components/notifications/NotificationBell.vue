<script setup>
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useNotifications } from '../../composables/useNotifications'
import { useFleet } from '../../composables/useFleet'
import AppIcon from '../ui/AppIcon.vue'
const router = useRouter(),
  route = useRoute()
const { items, unread, busy, error, markRead, markAllRead } = useNotifications()
const { now, loading } = useFleet()
const open = ref(false),
  filter = ref('all'),
  root = ref(null),
  panel = ref(null),
  trigger = ref(null)
const filtered = computed(() =>
  items.value.filter((item) => filter.value === 'all' || !item.read).slice(0, 30),
)
const icons = { critical: 'warning', warning: 'error', success: 'check_circle', info: 'info' }
function time(item) {
  const minutes = Math.max(0, Math.floor((now.value - new Date(item.createdAt).getTime()) / 60000))
  return minutes < 1
    ? 'Just now'
    : minutes < 60
      ? `${minutes}m ago`
      : `${Math.floor(minutes / 60)}h ago`
}
function close(restore = true) {
  open.value = false
  if (restore) trigger.value?.focus()
}
async function toggle() {
  open.value = !open.value
  if (open.value) {
    await nextTick()
    panel.value?.focus()
  }
}
function outside(event) {
  if (open.value && !root.value?.contains(event.target)) close(false)
}
function key(event) {
  if (event.key === 'Escape' && open.value) {
    event.preventDefault()
    close()
  }
}
async function markOne(id) {
  try {
    await markRead(id)
  } catch {
    /* Error is rendered in the panel. */
  }
}
async function markEvery() {
  try {
    await markAllRead()
  } catch {
    /* Error is rendered in the panel. */
  }
}
async function view(item) {
  try {
    await markRead(item.id)
  } catch {
    return
  }
  close(false)
  router.push(item.target || '/alerts')
}
watch(open, (value) => {
  if (value) {
    document.addEventListener('pointerdown', outside)
    document.addEventListener('keydown', key)
  } else {
    document.removeEventListener('pointerdown', outside)
    document.removeEventListener('keydown', key)
  }
})
watch(
  () => route.fullPath,
  () => close(false),
)
onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', outside)
  document.removeEventListener('keydown', key)
})
</script>
<template>
  <div ref="root" class="relative ml-auto md:ml-0">
    <button
      ref="trigger"
      class="relative flex h-7 w-7 items-center justify-center rounded text-slate-500 hover:bg-slate-100"
      aria-label="Notifications"
      aria-describedby="notification-summary"
      aria-haspopup="dialog"
      :aria-expanded="open"
      aria-controls="notification-panel"
      @click="toggle"
    >
      <AppIcon name="notifications" class="!text-lg" /><span
        v-if="unread"
        class="absolute -right-0.5 -top-0.5 min-w-3 rounded-full bg-rose-600 px-1 text-[8px] text-white"
        aria-hidden="true"
        >{{ unread > 99 ? '99+' : unread }}</span
      ><span id="notification-summary" class="sr-only" role="status" aria-live="polite"
        >{{ unread }} unread notifications</span
      ></button
    ><Transition name="popover"
      ><section
        v-if="open"
        id="notification-panel"
        ref="panel"
        tabindex="-1"
        role="dialog"
        aria-label="Notifications panel"
        class="fixed right-3 top-[52px] z-[70] flex max-h-[calc(100dvh-100px)] w-[370px] max-w-[calc(100vw-24px)] flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xl outline-none"
      >
        <header
          class="flex shrink-0 items-center justify-between gap-2 border-b border-slate-100 px-3 py-3"
        >
          <div>
            <h2 class="text-xs font-semibold text-slate-900">
              Notifications
              <span class="ml-1 text-[10px] font-normal text-slate-400">{{
                loading && !items.length ? 'Loading…' : `${unread} unread`
              }}</span>
            </h2>
            <p class="mt-1 text-[9px] text-slate-400">
              Reading a message does not resolve the incident.
            </p>
          </div>
          <button
            class="text-slate-400 hover:text-slate-700"
            aria-label="Close notifications"
            @click="close()"
          >
            <AppIcon name="close" class="!text-base" />
          </button>
        </header>
        <div class="flex shrink-0 items-center gap-1 border-b border-slate-100 px-3 py-2">
          <button
            v-for="tab in ['all', 'unread']"
            :key="tab"
            class="rounded px-2 py-1 text-[10px] font-medium capitalize"
            :class="
              filter === tab ? 'bg-blue-50 text-blue-700' : 'text-slate-500 hover:bg-slate-50'
            "
            :aria-pressed="filter === tab"
            @click="filter = tab"
          >
            {{ tab }}</button
          ><button
            class="ml-auto text-[10px] font-medium text-blue-700 disabled:text-slate-300"
            :disabled="!unread || busy"
            @click="markEvery"
          >
            Mark all read
          </button>
        </div>
        <p
          v-if="error"
          class="border-b border-rose-100 bg-rose-50 px-3 py-2 text-[10px] text-rose-700"
          role="alert"
        >
          {{ error }}
        </p>
        <div class="min-h-0 overflow-y-auto">
          <div
            v-if="loading && !items.length"
            class="flex items-center justify-center gap-2 p-8 text-[11px] text-slate-500"
            role="status"
          >
            <AppIcon name="progress_activity" class="animate-spin !text-base" />Loading
            notifications…
          </div>
          <ul v-else-if="filtered.length" class="divide-y divide-slate-100">
            <li
              v-for="item in filtered"
              :key="item.id"
              class="flex gap-2.5 p-3"
              :class="!item.read && 'bg-blue-50/40'"
            >
              <AppIcon
                :name="icons[item.severity] || 'info'"
                class="mt-0.5 shrink-0 !text-lg"
                :class="
                  item.severity === 'critical'
                    ? 'text-rose-600'
                    : item.severity === 'warning'
                      ? 'text-amber-600'
                      : 'text-blue-600'
                "
              />
              <div class="min-w-0 flex-1">
                <button
                  class="block text-left text-[11px] font-semibold leading-4 text-slate-800 hover:text-blue-700"
                  @click="view(item)"
                >
                  {{ item.title
                  }}<span
                    v-if="!item.read"
                    class="ml-1 inline-block h-1.5 w-1.5 rounded-full bg-blue-600"
                    aria-label="Unread"
                  ></span>
                </button>
                <p class="mt-1 text-[10px] leading-4 text-slate-500">{{ item.message }}</p>
                <div class="mt-2 flex items-center justify-between text-[9px] text-slate-400">
                  <time :datetime="item.createdAt">{{ time(item) }}</time
                  ><button
                    v-if="!item.read"
                    class="font-medium text-blue-700"
                    :aria-label="`Mark read: ${item.title}`"
                    :disabled="busy"
                    @click="markOne(item.id)"
                  >
                    Mark read</button
                  ><span v-else>Read</span>
                </div>
              </div>
            </li>
          </ul>
          <div v-else class="flex flex-col items-center gap-2 px-4 py-10 text-center">
            <AppIcon name="notifications_none" class="!text-3xl text-slate-300" />
            <p class="text-xs font-medium text-slate-600">
              {{ filter === 'unread' ? 'You are all caught up' : 'No notifications yet' }}
            </p>
            <p class="text-[10px] text-slate-400">Relevant system events will appear here.</p>
          </div>
        </div>
        <footer class="shrink-0 border-t border-slate-100 bg-slate-50 p-2 text-center">
          <RouterLink
            to="/alerts"
            class="text-[10px] font-semibold text-blue-700"
            @click="close(false)"
            >View notification history →</RouterLink
          >
        </footer>
      </section></Transition
    >
  </div>
</template>
