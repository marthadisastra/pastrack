<script setup>
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useNotifications } from '../../composables/useNotifications'
import AppIcon from '../../components/ui/AppIcon.vue'
import StatusBadge from '../../components/ui/StatusBadge.vue'
const router = useRouter(),
  { items, unread, markRead, markAllRead } = useNotifications()
const query = ref(''),
  filter = ref('all')
const filtered = computed(() =>
  items.value.filter(
    (item) =>
      (filter.value === 'all' ||
        (filter.value === 'unread' ? !item.read : item.severity === filter.value)) &&
      `${item.title} ${item.message}`.toLowerCase().includes(query.value.toLowerCase()),
  ),
)
function view(item) {
  markRead(item.id)
  router.push(item.target && item.target !== '/alerts' ? item.target : '/dashboard')
}
</script>
<template>
  <div class="page">
    <header class="panel flex flex-wrap items-center justify-between gap-3 p-3">
      <div>
        <h1 class="flex items-center gap-2 text-sm font-bold">
          <AppIcon name="notifications_active" class="text-blue-700" />Alert Center
        </h1>
        <p class="mt-1 text-[10px] text-slate-400">
          Notification history · {{ unread }} unread · Read status does not resolve an incident.
        </p>
      </div>
      <button class="btn" :disabled="!unread" @click="markAllRead">Mark all read</button>
    </header>
    <div class="panel flex flex-wrap items-center gap-2 p-3">
      <label class="min-w-40 flex-1"
        ><span class="sr-only">Search notifications</span
        ><input
          v-model="query"
          type="search"
          placeholder="Search notifications…"
          class="h-8 w-full" /></label
      ><label
        ><span class="sr-only">Notification filter</span
        ><select v-model="filter" class="h-8">
          <option value="all">All notifications</option>
          <option value="unread">Unread only</option>
          <option value="critical">Critical</option>
          <option value="warning">Warnings</option>
          <option value="success">Completed actions</option>
        </select></label
      >
    </div>
    <section class="panel">
      <ul v-if="filtered.length" class="divide-y divide-slate-100">
        <li
          v-for="item in filtered"
          :key="item.id"
          class="flex items-start gap-3 p-3"
          :class="!item.read && 'bg-blue-50/30'"
        >
          <AppIcon
            :name="
              item.severity === 'critical'
                ? 'warning'
                : item.severity === 'success'
                  ? 'check_circle'
                  : 'info'
            "
            class="mt-0.5 shrink-0 !text-lg"
            :class="item.severity === 'critical' ? 'text-rose-500' : 'text-blue-600'"
          />
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <h2 class="text-xs font-semibold">{{ item.title }}</h2>
              <StatusBadge
                :tone="
                  item.severity === 'critical'
                    ? 'red'
                    : item.severity === 'warning'
                      ? 'amber'
                      : 'blue'
                "
                >{{ item.severity }}</StatusBadge
              ><span v-if="!item.read" class="text-[9px] text-blue-600">Unread</span>
            </div>
            <p class="mt-1 text-[11px] leading-5 text-slate-500">{{ item.message }}</p>
            <time class="mt-1 block text-[9px] text-slate-400" :datetime="item.createdAt"
              >{{
                new Date(item.createdAt).toLocaleString('en-GB', { timeZone: 'Asia/Jakarta' })
              }}
              WIB</time
            >
            <div class="mt-2 flex gap-3">
              <button class="text-[10px] font-semibold text-blue-700" @click="view(item)">
                {{ item.vehicleId ? 'Inspect unit' : 'Open related page' }}</button
              ><button
                v-if="!item.read"
                class="text-[10px] text-slate-500"
                @click="markRead(item.id)"
              >
                Mark read
              </button>
            </div>
          </div>
        </li>
      </ul>
      <div v-else class="py-14 text-center">
        <AppIcon name="notifications_none" class="!text-3xl text-slate-300" />
        <h2 class="mt-2 text-xs font-semibold">No matching notifications</h2>
        <p class="mt-1 text-[10px] text-slate-400">
          Try another filter or wait for a relevant system event.
        </p>
      </div>
    </section>
  </div>
</template>
